//go:build windows

package printer

import (
	"context"
	"fmt"
	"github.com/benyaa/virtual-printer-process-engine/config"
	"github.com/benyaa/virtual-printer-process-engine/definitions"
	"github.com/benyaa/virtual-printer-process-engine/printer/winapi"
	"github.com/benyaa/virtual-printer-process-engine/utils"
	log "github.com/sirupsen/logrus"
	"os"
	"strings"
	"time"
)

type windowsPrinter struct {
	printerName      string
	ctx              context.Context
	ch               chan string
	printerProcessor winapi.PrinterProcessor
	conf             config.Config
}

func Create(ctx context.Context, conf config.Config, dir string) Creator {
	printerName := conf.Printer.Name
	tmpFolder := strings.ReplaceAll(dir, "/", `\`)
	log.Infof("creating Windows printer %s for path %s", printerName, conf.Workdir)
	err := os.MkdirAll(tmpFolder, os.ModePerm)
	if err != nil {
		log.WithError(err).Fatalf("failed to create temp folder %s", tmpFolder)
		panic(err)
	}
	return &windowsPrinter{
		printerName:      printerName,
		ctx:              ctx,
		printerProcessor: winapi.NewPrinterProcessor(ctx, tmpFolder, printerName),
		conf:             conf,
	}
}

func (p *windowsPrinter) CreateVirtualPrinter() error {
	log.Tracef("checking if printer %s exists", p.printerName)
	exists, err := winapi.PrinterExists(p.printerName)
	if err != nil {
		return fmt.Errorf("failed to check if printer exists: %v", err)
	}
	if exists {
		log.Warnf("printer %s already exists, skipping creation", p.printerName)
	} else {
		log.Infof("printer %s does not exist, creating...", p.printerName)
		err = p.createVirtualPrinter(err)
		if err != nil {
			return err
		}
	}

	log.Debugf("starting printer processor")

	go p.printerProcessor.RunService(time.Duration(p.conf.Printer.MonitorInterval) * time.Millisecond)

	return nil
}

func (p *windowsPrinter) createVirtualPrinter(err error) error {
	powershellCmd := fmt.Sprintf(`
		$printerName = "%s"
		# Output folder to save files
		$outputFolder = "$env:TMP"
		
		$printerDriver = "Microsoft Print To PDF"
		
		try {
			Add-PrinterPort -Name $printerName -PrinterHostAddress "127.0.0.1" -ErrorAction SilentlyContinue
		} catch {
		}

		try {
			Add-Printer -Name $printerName -DriverName $printerDriver -PortName $printerName -ErrorAction SilentlyContinue
		} catch {
		}

		exit 0
	`, p.printerName)

	output, err := utils.ExecuteCommand("powershell", "-NoProfile", "-NonInteractive", "-Command", powershellCmd)
	if err != nil {
		log.WithError(err).Errorf("failed to create printer: %s, output: %s", err, output)
		return fmt.Errorf("failed to create printer: %s, output: %s", err, output)
	}

	log.Info("virtual printer created successfully, waiting for 1 seconds to ensure the printer is ready")

	time.Sleep(1 * time.Second)
	return nil
}

func (p *windowsPrinter) GetChannel() chan definitions.PrintInfo {
	return p.printerProcessor.GetChannel()
}

func (p *windowsPrinter) RemoveVirtualPrinter() error {
	output, err := utils.ExecuteCommand("powershell", "-Command", fmt.Sprintf(`Remove-Printer -Name "%s"`, p.printerName))
	if err != nil {
		return fmt.Errorf("failed to remove printer: %s, output: %s", err, string(output))
	}

	return nil
}
