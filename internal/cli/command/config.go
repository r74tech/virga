package command

import (
	"fmt"
	"strings"

	"github.com/r74tech/virga/internal/cli/session"
)

// LogCommand is the implementation of the 'log' command.
type LogCommand struct{}

// Execute executes the 'log' command.
func (c *LogCommand) Execute(sessionManager *session.Manager, args []string) error {
	// Check the current session
	currentSession := sessionManager.GetCurrentSession()
	if currentSession == nil {
		return fmt.Errorf("no session selected. Use 'interact <session_id>' first")
	}

	// Parse subcommand
	if len(args) < 1 {
		return fmt.Errorf("usage: log <status|level|enable|disable> [options]")
	}

	subcommand := strings.ToLower(args[0])

	// Build the args array for the module
	moduleArgs := []string{subcommand}

	switch subcommand {
	case "status":
		// No additional arguments needed

	case "level":
		if len(args) >= 2 {
			level := strings.ToLower(args[1])
			validLevels := []string{"debug", "info", "warn", "error", "off"}
			valid := false
			for _, v := range validLevels {
				if level == v {
					valid = true
					break
				}
			}
			if !valid {
				return fmt.Errorf("invalid log level: %s. Use debug, info, warn, error, or off", level)
			}
			moduleArgs = append(moduleArgs, level)
		}
		// If no level specified, just get current level

	case "enable":
		// No additional arguments needed

	case "disable":
		// No additional arguments needed

	default:
		return fmt.Errorf("unknown subcommand: %s. Use 'status', 'level', 'enable', or 'disable'", subcommand)
	}

	payload := map[string]interface{}{
		"args": moduleArgs,
	}

	// Send the log request
	taskID, err := currentSession.ExecuteCommand("log", payload)
	if err != nil {
		return fmt.Errorf("error sending log request: %v", err)
	}

	// Wait for the result
	result, err := waitForTaskResult(currentSession, taskID, 10)
	if err != nil {
		return fmt.Errorf("error waiting for log result: %v", err)
	}

	// Display the result
	if result.ExitCode == 0 {
		if subcommand == "status" {
			fmt.Println("Log Configuration:")
			fmt.Println("--------------------------------------------------------------------------------")
		}
		fmt.Println(result.Output)
	} else {
		if result.Error != "" {
			return fmt.Errorf("error: %s", result.Error)
		}
		return fmt.Errorf("command failed: %s", result.Output)
	}

	return nil
}

// Help returns the help message for the 'log' command.
func (c *LogCommand) Help() string {
	return "Control implant logging configuration.\n" +
		"Usage:\n" +
		"  log status              - Show current log configuration\n" +
		"  log level <level>       - Set log level (debug, info, warn, error)\n" +
		"  log enable              - Enable logging\n" +
		"  log disable             - Disable logging\n" +
		"\n" +
		"Examples:\n" +
		"  log status              - Display current logging status and level\n" +
		"  log level debug         - Set log level to debug (most verbose)\n" +
		"  log disable             - Turn off all logging\n" +
		"\n" +
		"Note: This command requires an active session. Use 'interact <session_id>' first."
}
