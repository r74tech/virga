package command

import (
	"context"
	"fmt"
	"time"

	"github.com/r74tech/virga/internal/cli/session"
	"github.com/r74tech/virga/internal/shared/logger"
	"github.com/r74tech/virga/internal/shared/protocol"
)

// BaseCommand provides common functionality for commands
type BaseCommand struct{}

// RequireSession checks if a session is selected and returns it
func (b *BaseCommand) RequireSession(sm *session.Manager) (*session.Session, error) {
	currentSession := sm.GetCurrentSession()
	if currentSession == nil {
		return nil, fmt.Errorf("no session selected. Use 'interact <session_id>' first")
	}
	return currentSession, nil
}

// ExecuteAndWait executes a command and waits for the result
func (b *BaseCommand) ExecuteAndWait(session *session.Session, cmdType string, payload map[string]interface{}, timeout time.Duration) (*protocol.TaskResult, error) {
	// Send the command
	taskID, err := session.ExecuteCommand(cmdType, payload)
	if err != nil {
		return nil, fmt.Errorf("error sending %s request: %w", cmdType, err)
	}

	// Wait for the result
	result, err := waitForTaskResultWithTimeout(session, taskID, timeout)
	if err != nil {
		return nil, fmt.Errorf("error waiting for %s result: %w", cmdType, err)
	}

	return result, nil
}

// ExecuteAndWaitContext executes a command with context support
func (b *BaseCommand) ExecuteAndWaitContext(ctx context.Context, session *session.Session, cmdType string, payload map[string]interface{}, timeout time.Duration) (*protocol.TaskResult, error) {
	// Send the command
	taskID, err := session.ExecuteCommand(cmdType, payload)
	if err != nil {
		return nil, fmt.Errorf("error sending %s request: %w", cmdType, err)
	}

	// Create a timeout context
	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Wait for the result with context
	result, err := waitForTaskResultWithContext(timeoutCtx, session, taskID)
	if err != nil {
		return nil, fmt.Errorf("error waiting for %s result: %w", cmdType, err)
	}

	return result, nil
}

// waitForTaskResultWithTimeout waits for a task result with a specified timeout
func waitForTaskResultWithTimeout(session *session.Session, taskID string, timeout time.Duration) (*protocol.TaskResult, error) {
	// Convert timeout to seconds for compatibility
	timeoutSec := int(timeout.Seconds())
	if timeoutSec < 1 {
		timeoutSec = 1
	}

	return waitForTaskResult(session, taskID, timeoutSec)
}

// waitForTaskResultWithContext waits for a task result with context support
func waitForTaskResultWithContext(ctx context.Context, session *session.Session, taskID string) (*protocol.TaskResult, error) {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			result, err := session.GetTaskResult(taskID)
			if err == nil && result != nil {
				return result, nil
			}
		}
	}
}

// FormatTaskError formats a task error for display
func (b *BaseCommand) FormatTaskError(result *protocol.TaskResult) string {
	if result.Error != "" {
		return result.Error
	}
	if result.ExitCode != 0 {
		return fmt.Sprintf("command failed with exit code %d", result.ExitCode)
	}
	return "unknown error"
}

// PrintTaskResult prints a task result in a consistent format
func (b *BaseCommand) PrintTaskResult(result *protocol.TaskResult, showExitCode bool) {
	if result.Output != "" {
		logger.Info(result.Output)
	}

	if showExitCode {
		logger.Debug("Exit code: %d", result.ExitCode)
	}

	if result.Error != "" {
		logger.Error("Task error: %s", result.Error)
	}
}
