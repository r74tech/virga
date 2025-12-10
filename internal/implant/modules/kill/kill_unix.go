//go:build darwin || linux
// +build darwin linux

package kill

import (
	"fmt"
	"os"
	"syscall"
	"time"
)

// killProcess terminates a process on Unix systems
func killProcess(pid int) error {
	// First check with syscall.Kill
	err := syscall.Kill(pid, 0)
	if err == syscall.ESRCH {
		return fmt.Errorf("process does not exist")
	}

	// Also get the Process handle for additional control
	proc, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("failed to find process: %w", err)
	}

	// Send SIGKILL via both methods for maximum effectiveness
	// Sometimes one method works better than the other on macOS
	syscall.Kill(pid, syscall.SIGKILL)
	proc.Kill()

	// Also try to kill the process group
	syscall.Kill(-pid, syscall.SIGKILL)

	// Wait for process to actually die
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		// Check if process exists
		err := syscall.Kill(pid, 0)
		if err == syscall.ESRCH {
			// Process is gone
			return nil
		}

		// Process still exists, keep trying
		time.Sleep(100 * time.Millisecond)

		// Send kill signal again
		syscall.Kill(pid, syscall.SIGKILL)
		proc.Kill()
	}

	// Final check
	err = syscall.Kill(pid, 0)
	if err == syscall.ESRCH {
		return nil
	}

	return fmt.Errorf("process still alive after SIGKILL")
}
