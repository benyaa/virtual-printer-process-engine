//go:build windows

package winapi

import (
	"context"
	"fmt"
	"github.com/benyaa/virtual-printer-process-engine/definitions"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"os"
	"path/filepath"
	"strings"
	"time"
)

/*
#cgo LDFLAGS: -lgdi32 -lwinspool
#include <windows.h>
#include <stdio.h>
#include <winspool.h>
#include <stdlib.h>

DWORD getPrintJobPages(HANDLE hPrinter, DWORD jobId) {
    DWORD needed, returned;
    JOB_INFO_2 *pJobInfo = NULL;

    // First, get the size of the job info structure
    if (!GetJob(hPrinter, jobId, 2, NULL, 0, &needed)) {
        if (GetLastError() != ERROR_INSUFFICIENT_BUFFER) {
            return 0; // Failed to get job info
        }
    }

    // Allocate memory for the job info structure
    pJobInfo = (JOB_INFO_2*)malloc(needed);
    if (pJobInfo == NULL) {
        return 0; // Memory allocation failure
    }

    // Now retrieve the job information
    if (!GetJob(hPrinter, jobId, 2, (LPBYTE)pJobInfo, needed, &returned)) {
        free(pJobInfo);
        return 0; // Failed to get job info
    }

    // Get the total number of pages
    DWORD totalPages = pJobInfo->TotalPages;

    // Clean up
    free(pJobInfo);

    return totalPages;
}

int copy_file(const char *src, const char *dst) {
    FILE *source = fopen(src, "rb");
    if (source == NULL) {
        return -1; // Error opening source file
    }

    FILE *destination = fopen(dst, "wb");
    if (destination == NULL) {
        fclose(source);
        return -2; // Error opening destination file
    }

    char buffer[8192];
    size_t bytes;
    while ((bytes = fread(buffer, 1, sizeof(buffer), source)) > 0) {
        if (fwrite(buffer, 1, bytes, destination) != bytes) {
            fclose(source);
            fclose(destination);
            return -3; // Error writing to destination file
        }
    }

    fclose(source);
    fclose(destination);
    return 0; // Success
}

void DeletePrintJob(const char* printerName, DWORD jobId) {
    HANDLE hPrinter;
    PRINTER_DEFAULTS pd;
    pd.pDatatype = NULL;
    pd.pDevMode = NULL;
    pd.DesiredAccess = PRINTER_ACCESS_ADMINISTER | PRINTER_ACCESS_USE;

    if (OpenPrinter((LPSTR)printerName, &hPrinter, &pd)) {
        if (SetJob(hPrinter, jobId, 0, NULL, JOB_CONTROL_DELETE)) {
            printf("Print job %d deleted successfully.\n", jobId);
        } else {
            printf("Failed to delete print job %d.\n", jobId);
        }
        ClosePrinter(hPrinter);
    } else {
        printf("Failed to open printer %s.\n", printerName);
    }
}
*/
import "C"
import (
	"unsafe"
)

const (
	JOB_STATUS_PAUSED   = 0x0001
	JOB_STATUS_ERROR    = 0x0008
	JOB_STATUS_DELETING = 0x0004
	JOB_STATUS_PRINTING = 0x0010
	JOB_STATUS_SPOOLING = 0x0002
	JOB_STATUS_OFFLINE  = 0x0020
	JOB_STATUS_PAPEROUT = 0x0040
	JOB_STATUS_RESTART  = 0x2000
)

type PrinterProcessor interface {
	GetChannel() chan definitions.PrintInfo
	RunService(monitorInterval time.Duration)
}

type processor struct {
	ctx               context.Context
	ch                chan definitions.PrintInfo
	destinationFolder string
	printerName       string
}

func (p *processor) GetChannel() chan definitions.PrintInfo {
	return p.ch
}

func NewPrinterProcessor(ctx context.Context, destinationFolder, printerName string) PrinterProcessor {
	return &processor{
		ctx:               ctx,
		destinationFolder: destinationFolder,
		printerName:       printerName,
		ch:                make(chan definitions.PrintInfo),
	}
}

func openPrinter(printerName string) C.HANDLE {
	var hPrinter C.HANDLE
	pname := C.CString(printerName)
	defer C.free(unsafe.Pointer(pname))

	if C.OpenPrinter(pname, &hPrinter, nil) == 0 {
		log.Fatalf("Failed to open printer: %s", printerName)
		panic("Failed to open printer")
	}
	return hPrinter
}

func enumJobs(hPrinter C.HANDLE) []C.JOB_INFO_1 {
	var needed, returned C.DWORD

	// Query the number of jobs and the required buffer size
	C.EnumJobs(hPrinter, 0, 255, 1, nil, 0, &needed, &returned)
	if needed == 0 {
		time.Sleep(1 * time.Second)
		return nil
	}

	// Allocate the buffer with the required size
	buffer := make([]byte, needed)

	// Call EnumJobs to retrieve the job information
	if C.EnumJobs(hPrinter, 0, 255, 1, (*C.BYTE)(unsafe.Pointer(&buffer[0])), needed, &needed, &returned) == 0 {
		log.Println("Failed to enumerate jobs")
		return nil
	}

	// Convert the buffer into a slice of JOB_INFO_1 structures
	var jobInfoSlice []C.JOB_INFO_1
	for i := 0; i < int(returned); i++ {
		jobInfo := (*C.JOB_INFO_1)(unsafe.Pointer(&buffer[i*int(unsafe.Sizeof(C.JOB_INFO_1{}))]))
		jobInfoSlice = append(jobInfoSlice, *jobInfo)
	}

	return jobInfoSlice
}

func closePrinter(hPrinter C.HANDLE) {
	if C.ClosePrinter(hPrinter) == 0 {
		log.Fatalf("Failed to close printer")
	}
}

func (p *processor) copySpoolFileAsXps(jobId uint32) (string, error) {
	spoolDir := `C:\Windows\System32\spool\PRINTERS`
	var spoolFilePath string

	// Locate the .spl file associated with the job
	err := filepath.Walk(spoolDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.ToLower(filepath.Ext(info.Name())) == ".spl" {
			// Naively assume the .spl file is associated with our job
			spoolFilePath = path
			return nil
		}
		return nil
	})
	if err != nil {
		return "", err
	}

	if spoolFilePath == "" {
		return "", fmt.Errorf("spool file for job %d not found", jobId)
	}

	// Copy the file to a new location with a new name
	destPath := fmt.Sprintf(p.destinationFolder+`\Job_%d_%s.xps`, jobId, uuid.New().String()[0:8])
	cSrc := C.CString(spoolFilePath)
	cDst := C.CString(destPath)
	defer C.free(unsafe.Pointer(cSrc))
	defer C.free(unsafe.Pointer(cDst))
	result := C.copy_file(cSrc, cDst)
	if result != 0 {
		return "", fmt.Errorf("failed to copy spool file: %d", result)
	}

	return destPath, nil
}

func jobStatusToString(status uint32) string {
	var statuses []string

	if status&JOB_STATUS_PAUSED != 0 {
		statuses = append(statuses, "PAUSED")
	}
	if status&JOB_STATUS_ERROR != 0 {
		statuses = append(statuses, "ERROR")
	}
	if status&JOB_STATUS_DELETING != 0 {
		statuses = append(statuses, "DELETING")
	}
	if status&JOB_STATUS_PRINTING != 0 {
		statuses = append(statuses, "PRINTING")
	}
	if status&JOB_STATUS_SPOOLING != 0 {
		statuses = append(statuses, "SPOOLING")
	}
	if status&JOB_STATUS_OFFLINE != 0 {
		statuses = append(statuses, "OFFLINE")
	}
	if status&JOB_STATUS_PAPEROUT != 0 {
		statuses = append(statuses, "PAPER OUT")
	}
	if status&JOB_STATUS_RESTART != 0 {
		statuses = append(statuses, "RESTART")
	}
	// Add other status checks as needed

	if len(statuses) == 0 {
		return "UNKNOWN"
	}

	return strings.Join(statuses, " | ")
}

func (p *processor) RunService(monitorInterval time.Duration) {
	hPrinter := openPrinter(p.printerName)
	defer closePrinter(hPrinter)
	lastHandled := false

	for {
		select {
		case <-p.ctx.Done():
			log.Infof("shutdown signal received")
			return
		default:
			log.Tracef("checking for new print jobs") // this log can really spam
			jobs := enumJobs(hPrinter)
			if jobs != nil {
				for _, job := range jobs {

					if job.Status&JOB_STATUS_DELETING != 0 ||
						(job.Status&JOB_STATUS_PRINTING == 0 &&
							job.Status&JOB_STATUS_SPOOLING == 0 &&
							job.Status&JOB_STATUS_RESTART == 0) {
						log.Tracef("skipping job %d with status %s", job.JobId, jobStatusToString(uint32(job.Status))) // this log can really spam
						continue
					}
					lastHandled = true
					log.Infof("processing Job ID: %d, Document: %s, Status: %s\n",
						job.JobId,
						C.GoString(job.pDocument),
						jobStatusToString(uint32(job.Status)))

					xpsFile, err := p.copySpoolFileAsXps(uint32(job.JobId))
					if err != nil {
						log.Errorf("failed to get XPS file for job %d: %v", job.JobId, err)
						continue
					}
					cPrinterName := C.CString(p.printerName)
					defer C.free(unsafe.Pointer(cPrinterName))
					cJobId := C.DWORD(job.JobId)
					printerInfo := definitions.PrintInfo{
						Filepath: xpsFile,
						Pages:    int(C.getPrintJobPages(hPrinter, cJobId)),
					}
					C.DeletePrintJob(cPrinterName, cJobId)
					p.ch <- printerInfo
				}
			}

			if !lastHandled { // if no jobs were handled, sleep for a bit to avoid killing the cpu
				time.Sleep(monitorInterval)
			}

			lastHandled = false
		}
	}
}
