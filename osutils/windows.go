//go:build windows

package osutils

/*
#include <windows.h>

int IsElevated() {
    HANDLE token = NULL;
    TOKEN_ELEVATION elevation;
    DWORD size;

    if (!OpenProcessToken(GetCurrentProcess(), TOKEN_QUERY, &token)) {
        return -1;  // Error opening the process token
    }

    if (!GetTokenInformation(token, TokenElevation, &elevation, sizeof(elevation), &size)) {
        CloseHandle(token);
        return -1;  // Error getting the token information
    }

    CloseHandle(token);

    return elevation.TokenIsElevated ? 1 : 0;
}
*/
import "C"
import (
	"errors"
	"fmt"
	"github.com/benyaa/virtual-printer-process-engine/consts"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sys/windows/registry"
	"os"
	"strings"
	"syscall"
)

func IsAdmin() (bool, error) {

	b := C.IsElevated()
	if b == -1 {
		return false, syscall.GetLastError()
	}

	return b == 1, nil
}

func IsRunningAtStartup() (bool, error) {
	log.Tracef("checking if running at startup")
	key, err := registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Run`, registry.QUERY_VALUE)
	if err != nil {
		return false, err
	}
	log.Debugf("opened registry key")
	defer key.Close()

	log.Debugf("getting string value from registry key")
	_, _, err = key.GetStringValue(consts.AppName)
	if err != nil {
		if errors.Is(err, registry.ErrNotExist) {
			return false, nil
		}
		return false, err
	}

	log.Debugf("got string value from registry key")

	return true, nil
}

func RunAtStartup(args ...string) error {
	log.Tracef("make sure running at startup")
	executablePath, err := os.Executable()
	if err != nil {
		return err
	}
	log.Debugf("executable path: %s, creating registry key", executablePath)
	key, _, err := registry.CreateKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Run`, registry.SET_VALUE)
	if err != nil {
		return err
	}
	log.Debugf("created registry key")
	defer key.Close()

	log.Infof("setting value in registry key")
	err = key.SetStringValue(consts.AppName, executablePath+" "+wrapArgsWithQuotes(args))
	if err != nil {
		return err
	}
	log.Debugf("set value in registry key")

	return nil
}

func wrapArgsWithQuotes(args []string) string {
	for i, arg := range args {
		args[i] = fmt.Sprintf("\"%s\"", arg)
	}
	return strings.Join(args, " ")
}

func RemoveFromStartup() error {
	log.Tracef("removing from startup")
	isRunningAtStartup, err := IsRunningAtStartup()
	if err != nil {
		return err
	}
	if !isRunningAtStartup {
		log.Debugf("not running at startup")
		return nil
	}
	log.Debugf("opening key")
	key, err := registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Run`, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer key.Close()
	log.Debugf("deleting value from registry key")
	err = key.DeleteValue(consts.AppName)
	if err != nil {
		return err
	}

	log.Debugf("deleted value from registry key")

	return nil
}
