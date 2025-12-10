//go:build windows
// +build windows

package kill

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
)

// killProcess terminates a process on Windows
func killProcess(pid int) error {
	// First try using os.Process
	proc, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("process not found: %w", err)
	}

	// Try to kill the process
	err = proc.Kill()
	if err != nil {
		// Fallback to taskkill command
		cmd := exec.Command("taskkill", "/F", "/PID", strconv.Itoa(pid))
		output, cmdErr := cmd.CombinedOutput()
		if cmdErr != nil {
			return fmt.Errorf("failed to kill process: %s", string(output))
		}
	}

	return nil
}
