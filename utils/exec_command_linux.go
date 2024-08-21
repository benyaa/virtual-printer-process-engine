//go:build linux

package utils

import (
	"os/exec"
)

var ExecuteCommand = executeCommand

func executeCommand(command string, args ...string) (string, error) {
	cmd := exec.Command(command, args...)
	output, err := cmd.CombinedOutput()

	return string(output), err
}
