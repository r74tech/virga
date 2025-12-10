package command

import (
	"fmt"
	"io"
	"strings"

	"github.com/peterh/liner"
	"github.com/r74tech/virga/internal/cli/session"
	"github.com/r74tech/virga/internal/shared/logger"
)

// ShellCommand is the implementation of the 'shell' command.
type ShellCommand struct{}

// Execute executes the 'shell' command.
func (c *ShellCommand) Execute(sessionManager *session.Manager, args []string) error {
	// Check the current session
	currentSession := sessionManager.GetCurrentSession()
	if currentSession == nil {
		return fmt.Errorf("no session selected. Use 'interact <session_id>' first")
	}

	if len(args) > 0 {
		return fmt.Errorf("shell command does not take any arguments")
	}

	// Display debug information
	logger.Debug("Starting shell session with session: %s", currentSession.ID)
	if apiClient := sessionManager.GetAPIClient(); apiClient != nil {
		globalLogger := logger.GetGlobalLogger()
		if globalLogger.GetLevel() <= logger.DEBUG {
			apiClient.SetDebugMode(true)
			logger.Debug("API Client debug mode enabled")
		}
	} else {
		logger.Warn("No API client available")
	}

	logger.Info("Starting interactive shell. Type 'exit' to return to virga.")
	logger.Info("------------------------------------------------------------")

	// Read input using the liner library
	liner := liner.NewLiner()
	defer liner.Close()

	liner.SetCtrlCAborts(true)

	for {
		cmd, err := liner.Prompt("shell> ")
		if err != nil {
			// Check the error type
			if err == io.EOF {
				// EOF (Ctrl+D)
				break
			} else if strings.Contains(err.Error(), "Interrupt") {
				// Ctrl+C
				logger.Info("^C")
				continue
			} else {
				// Other errors
				logger.Error("Error reading input: %v", err)
				break
			}
		}

		if cmd == "exit" || cmd == "quit" {
			break
		}

		if cmd == "" {
			continue
		}

		// Add to command history
		liner.AppendHistory(cmd)

		// Validate and sanitize command
		if err := validateShellCommand(cmd); err != nil {
			logger.Error("Command validation failed: %v", err)
			continue
		}

		// Sanitize the command
		sanitizedCmd := sanitizeShellCommand(cmd)

		// Send the command execution request
		logger.Debug("Executing command: %s", sanitizedCmd)
		taskID, err := currentSession.ExecuteCommand("shell", map[string]interface{}{
			"command": sanitizedCmd,
		})
		if err != nil {
			logger.Error("Error executing command: %v", err)
			continue
		}

		logger.Debug("Task ID generated: %s", taskID)

		// Wait for the result
		result, err := waitForTaskResult(currentSession, taskID, 60)
		if err != nil {
			logger.Error("Error waiting for result: %v", err)
			continue
		}

		// Display the result
		logger.Debug("Task result received: exit_code=%d, output_len=%d", result.ExitCode, len(result.Output))
		logger.Info(result.Output)
	}

	logger.Info("------------------------------------------------------------")
	logger.Info("Shell session terminated.")
	return nil
}

// Help returns the help for the 'shell' command.
func (c *ShellCommand) Help() string {
	return "Start an interactive shell on the target system.\n" +
		"Usage: shell\n" +
		"  shell             Launch interactive shell\n" +
		"  shell @payload    Execute an embedded payload (defined via beacon.payloads)\n" +
		"Note: This command requires an active session. Use 'interact <session_id>' first."
}

// ExecCommand is the implementation of the 'exec' command.
type ExecCommand struct{}

// Execute executes the 'exec' command.
func (c *ExecCommand) Execute(sessionManager *session.Manager, args []string) error {
	// Check the current session
	currentSession := sessionManager.GetCurrentSession()
	if currentSession == nil {
		return fmt.Errorf("no session selected. Use 'interact <session_id>' first")
	}

	if len(args) == 0 {
		return fmt.Errorf("command required. Usage: exec <command>")
	}

	// Build the command
	command := strings.Join(args, " ")

	// Validate and sanitize command
	if err := validateShellCommand(command); err != nil {
		return fmt.Errorf("command validation failed: %w", err)
	}

	// Sanitize the command
	sanitizedCmd := sanitizeShellCommand(command)

	// Send the command execution request
	taskID, err := currentSession.ExecuteCommand("shell", map[string]interface{}{
		"command": sanitizedCmd,
	})
	if err != nil {
		return fmt.Errorf("error executing command: %v", err)
	}

	logger.Debug("Task ID: %s - Executing: %s", taskID, command)

	// Wait for the result
	result, err := waitForTaskResult(currentSession, taskID, 60)
	if err != nil {
		return fmt.Errorf("error waiting for result: %v", err)
	}

	// Display the result
	logger.Info("Command output:")
	logger.Info("------------------------------------------------------------")
	logger.Info(result.Output)
	logger.Info("------------------------------------------------------------")
	logger.Debug("Exit code: %d", result.ExitCode)

	return nil
}

// Help returns the help for the 'exec' command.
func (c *ExecCommand) Help() string {
	return "Execute a command on the target system.\n" +
		"Usage: exec <command>\n" +
		"  command    The command to execute on the target system.\n" +
		"Note: This command requires an active session. Use 'interact <session_id>' first."
}
