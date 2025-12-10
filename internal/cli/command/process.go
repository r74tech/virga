package command

import (
	"fmt"
	"strconv"

	"github.com/r74tech/virga/internal/cli/session"
	"github.com/r74tech/virga/internal/shared/logger"
)

// KillCommand is the implementation of the 'kill' command.
type KillCommand struct{}

// Execute executes the 'kill' command.
func (c *KillCommand) Execute(sessionManager *session.Manager, args []string) error {
	// Check the current session
	currentSession := sessionManager.GetCurrentSession()
	if currentSession == nil {
		return fmt.Errorf("no session selected. Use 'interact <session_id>' first")
	}

	// Validate arguments
	if len(args) < 1 {
		return fmt.Errorf("usage: kill <pid>")
	}

	// Validate PID
	pid, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("invalid PID: %s", args[0])
	}

	// Send the kill request
	taskID, err := currentSession.ExecuteCommand("kill", map[string]interface{}{
		"args": []string{strconv.Itoa(pid)},
	})
	if err != nil {
		return fmt.Errorf("error sending kill request: %v", err)
	}

	logger.Info("Sending kill signal to process %d...", pid)

	// Wait for the result
	result, err := waitForTaskResult(currentSession, taskID, 10)
	if err != nil {
		return fmt.Errorf("error waiting for kill result: %v", err)
	}

	// Display the result
	if result.ExitCode == 0 {
		fmt.Println(result.Output)
	} else {
		if result.Error != "" {
			return fmt.Errorf("%s", result.Error)
		}
		return fmt.Errorf("%s", result.Output)
	}

	return nil
}

// Help returns the help message for the 'kill' command.
func (c *KillCommand) Help() string {
	return "Kill a process by PID.\n" +
		"Usage: kill <pid>\n" +
		"Example: kill 1234\n" +
		"Note: This command requires an active session. Use 'interact <session_id>' first."
}
