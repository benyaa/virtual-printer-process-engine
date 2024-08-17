//go:build linux

package osutils

import (
	"fmt"
	"github.com/benyaa/virtual-printer-process-engine/consts"
	log "github.com/sirupsen/logrus"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
)

var autoStartPath = filepath.Join(os.Getenv("HOME"), ".config/autostart", consts.AppName+".desktop")

func IsAdmin() (bool, error) {
	log.Tracef("checking if running as root")
	currentUser, err := user.Current()
	if err != nil {
		return false, err
	}

	log.Debugf("current user: %s", currentUser.Username)

	uid, err := strconv.Atoi(currentUser.Uid)
	if err != nil {
		return false, err
	}

	log.Debugf("current user UID: %d", uid)
	// Check if UID is 0 (root)
	return uid == 0, nil
}

func RunAtStartup(args ...string) error {
	log.Tracef("checking if running at startup")
	isRunningAtStartup, err := IsRunningAtStartup()
	if err != nil {
		return err
	}
	if isRunningAtStartup {
		log.Infof("app is already running at startup, skipping...")
		return nil
	}
	log.Debugf("app is not running at startup, getting executable path")
	executablePath, err := os.Executable()
	if err != nil {
		return err
	}
	log.Debugf("executable path: %s, creating autostart file", executablePath)
	desktopFileContent := fmt.Sprintf(`[Desktop Entry]
Type=Application
Exec=%s %s
Hidden=false
NoDisplay=true
X-GNOME-Autostart-enabled=true
Name=%s
Comment=Virtual Printer to whatever
`, executablePath, strings.Join(args, " "), consts.AppName)

	log.Debugf("writing autostart file")
	return os.WriteFile(autoStartPath, []byte(desktopFileContent), 0644)
}

func IsRunningAtStartup() (bool, error) {
	log.Debug("checking if running at startup")
	if _, err := os.Stat(autoStartPath); os.IsNotExist(err) {
		log.Debugf("app is not running at startup")
		return false, nil
	} else if err != nil {
		return false, err
	}

	log.Debugf("app is running at startup", autoStartPath)
	return true, nil
}

func RemoveFromStartup() error {
	isRunningAtStartup, err := IsRunningAtStartup()
	if err != nil {
		return err
	}
	if !isRunningAtStartup {
		log.Debugf("app is not running at startup, skipping...")
		return nil
	}

	log.Debugf("removing autostart file")
	return os.Remove(autoStartPath)
}
