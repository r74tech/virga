//go:build windows

package shell

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

// executeShellCommand executes a command without a shell (Windows)
func executeShellCommand(command string) (string, int, error) {
	// Parse the command line string to separate the executable and arguments
	args, err := parseCommandLine(command)
	if err != nil {
		return "", 1, err
	}

	if len(args) == 0 {
		return "", 1, fmt.Errorf("empty command")
	}

	// Resolve the executable path
	exePath, err := resolveExecutablePath(args[0])
	if err != nil {
		return "", 1, fmt.Errorf("executable not found: %s", args[0])
	}

	// Prepare arguments
	var cmdArgs []string
	if len(args) > 1 {
		cmdArgs = args[1:]
	}

	// Execute directly without a shell
	cmd := exec.Command(exePath, cmdArgs...)

	// For Windows, hide the console window
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow: true,
	}

	// Capture output and error
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Run the command
	err = cmd.Run()

	// Combine output
	output := stdout.String()
	if stderr.Len() > 0 {
		if output != "" {
			output += "\n"
		}
		output += stderr.String()
	}

	// Get the exit code
	exitCode := 0
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			exitCode = exitError.ExitCode()
		} else {
			// If the command execution itself fails
			return fmt.Sprintf("command execution error: %v", err), 1, err
		}
	}

	return strings.TrimSpace(output), exitCode, nil
}

// trySystemPaths searches for an executable in standard system locations
func trySystemPaths(executable string) (string, error) {
	systemPaths := []string{
		"C:\\Windows\\System32\\",
		"C:\\Windows\\",
		"C:\\Windows\\System32\\WindowsPowerShell\\v1.0\\",
	}
	if !strings.HasSuffix(executable, ".exe") {
		executable += ".exe"
	}

	for _, path := range systemPaths {
		fullPath := filepath.Join(path, executable)
		if _, err := os.Stat(fullPath); err == nil {
			return fullPath, nil
		}
	}

	return "", fmt.Errorf("executable not found: %s", executable)
}
