package printer

import "github.com/benyaa/virtual-printer-process-engine/definitions"

type Creator interface {
	CreateVirtualPrinter() error
	RemoveVirtualPrinter() error
	GetChannel() chan definitions.PrintInfo
}
