//go:build windows

package utils

import (
	"os/exec"
	"syscall"
)

var ExecuteCommand = executeCommand

func executeCommand(command string, args ...string) (string, error) {
	cmd := exec.Command(command, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow: true,
	}
	output, err := cmd.CombinedOutput()

	return string(output), err
}
