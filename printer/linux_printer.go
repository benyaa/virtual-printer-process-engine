//go:build linux

package printer

import "C"
import (
	"context"
	"fmt"
	"github.com/benyaa/virtual-printer-process-engine/config"
	"github.com/benyaa/virtual-printer-process-engine/definitions"
	"github.com/benyaa/virtual-printer-process-engine/utils"
	"github.com/google/uuid"
	"github.com/pdfcpu/pdfcpu/pkg/api"
	log "github.com/sirupsen/logrus"
	"io/fs"
	"os"
	"path/filepath"
	"time"
)

type printerCreator struct {
	conf    config.Config
	ctx     context.Context
	channel chan definitions.PrintInfo
	dir     string
}

func Create(ctx context.Context, conf config.Config, dir string) Creator {
	return &printerCreator{
		conf:    conf,
		channel: make(chan definitions.PrintInfo),
		ctx:     ctx,
		dir:     dir,
	}
}

func (pc *printerCreator) GetChannel() chan definitions.PrintInfo {
	return pc.channel
}

func (pc *printerCreator) CreateVirtualPrinter() error {
	output, err := utils.ExecuteCommand("lpadmin",
		"-p", pc.conf.Printer.Name,
		"-E",
		"-v", "cups-pdf:/",
		"-m", "raw")
	if err != nil {
		log.WithError(err).Errorf("failed to create virtual printer: %s", output)
		return fmt.Errorf("failed to create virtual printer: %v", err)
	}

	go pc.monitorOutputDirectory()

	return nil
}

func (pc *printerCreator) RemoveVirtualPrinter() error {
	log.Infof("removing virtual printer: %s", pc.conf.Printer.Name)

	_, err := utils.ExecuteCommand(
		"lpadmin",
		"-x", pc.conf.Printer.Name,
	)

	if err != nil {
		return fmt.Errorf("failed to remove virtual printer: %v", err)
	}

	return nil
}

func (pc *printerCreator) monitorOutputDirectory() {
	outputDir := filepath.Join(os.Getenv("HOME"), "PDF") // Default output directory for cups-pdf
	ticker := time.NewTicker(5 * time.Second)            // Adjust the interval as needed
	defer ticker.Stop()

	for {
		select {
		case <-pc.ctx.Done():
			log.Infof("context canceled, stopping monitorOutputDirectory.")
			return
		case <-ticker.C:
			err := filepath.WalkDir(outputDir, func(path string, d fs.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if !d.IsDir() {
					pc.processPDF(path)
				}
				return nil
			})
			if err != nil {
				log.WithError(err).Errorf("failed to walk output directory: %s", outputDir)
			}
		}
	}
}

func (pc *printerCreator) processPDF(path string) {
	log.Infof("processing PDF file: %s", path)
	outputPath := filepath.Join(pc.dir, fmt.Sprintf("job_%s.pdf", uuid.New().String()[0:8]))
	pages, err := pc.getNumberOfPages(path)
	if err != nil {
		log.WithError(err).Errorf("failed to get number of pages for PDF file: %s", path)
		return
	}

	log.Debugf("moving PDF file to output directory: %s", outputPath)
	err = utils.CopyFile(path, outputPath)
	if err != nil {
		log.WithError(err).Errorf("failed to move PDF file: %s", path)
		return
	}

	log.Debugf("removing original PDF file: %s", path)

	err = os.Remove(path)
	if err != nil {
		log.Printf("Failed to remove original PDF file %s: %v", path, err)
		return
	}

	pc.channel <- definitions.PrintInfo{
		Filepath: outputPath,
		Pages:    pages,
	}
	log.Debugf("added PDF file to channel: %s", outputPath)
}

func (pc *printerCreator) getNumberOfPages(filePath string) (int, error) {
	pdf, err := api.ReadContextFile(filePath)
	if err != nil {
		return 0, fmt.Errorf("failed to read PDF file: %v", err)
	}
	return pdf.PageCount, nil
}
