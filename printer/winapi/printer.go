//go:build windows

package winapi

/*
#include <windows.h>
#include <winspool.h>
#include <stdio.h>
#include <stdlib.h>

int printerExists(char *printerName) {
    DWORD needed, returned;
    PRINTER_INFO_2 *printerInfo;
    EnumPrinters(PRINTER_ENUM_LOCAL | PRINTER_ENUM_CONNECTIONS, NULL, 2, NULL, 0, &needed, &returned);
    printerInfo = (PRINTER_INFO_2*)malloc(needed);
    if (!EnumPrinters(PRINTER_ENUM_LOCAL | PRINTER_ENUM_CONNECTIONS, NULL, 2, (LPBYTE)printerInfo, needed, &needed, &returned)) {
        free(printerInfo);
        return 0;
    }

    for (DWORD i = 0; i < returned; i++) {
        if (_stricmp(printerInfo[i].pPrinterName, printerName) == 0) { // Case-insensitive comparison
            free(printerInfo);
            return 1;
        }
    }
    free(printerInfo);
    return 0;
}
*/
import "C"
import "unsafe"

func PrinterExists(printerName string) (bool, error) {
	name := C.CString(printerName)
	defer C.free(unsafe.Pointer(name))

	exists := C.printerExists(name)
	return exists == 1, nil
}
