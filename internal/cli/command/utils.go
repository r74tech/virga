package command

import (
	"fmt"
	"time"

	"github.com/r74tech/virga/internal/cli/session"
	"github.com/r74tech/virga/internal/shared/logger"
	"github.com/r74tech/virga/internal/shared/protocol"
)

// waitForTaskResult waits until a task result is returned (common function).
func waitForTaskResult(session *session.Session, taskID string, timeoutSecs int) (*protocol.TaskResult, error) {
	start := time.Now()
	timeout := time.Duration(timeoutSecs) * time.Second
	attempts := 0
	maxAttempts := 120 // Increased maximum number of attempts

	// Timer for progress display
	progressTimer := time.NewTimer(2 * time.Second)
	defer progressTimer.Stop()

	progressShown := false

	logger.Debug("Waiting for task %s result...", taskID)
	logger.Debug("Timeout set to %d seconds", timeoutSecs)

	for {
		attempts++

		// Timeout check
		elapsed := time.Since(start)
		if elapsed > timeout {
			return nil, fmt.Errorf("timeout waiting for task result after %.1fs", elapsed.Seconds())
		}

		// Improved progress display
		select {
		case <-progressTimer.C:
			if !progressShown {
				logger.Info("Waiting for task result... (elapsed: %.1fs)", elapsed.Seconds())
				progressShown = true
			} else {
				// Update progress periodically
				logger.Debug("Still waiting... (elapsed: %.1fs)", elapsed.Seconds())
			}
			progressTimer.Reset(3 * time.Second) // Update every 3 seconds
		default:
			// Do nothing if the timer has not fired yet
		}

		// Get task result using the API
		result, err := session.GetTaskResult(taskID)
		if err == nil && result != nil {
			// If progress was shown, display a completion message
			if progressShown {
				logger.Debug("Task result received (after %.1fs)", elapsed.Seconds())
			}
			logger.Debug("Task %s completed successfully", taskID)
			return result, nil
		}

		// Add debug information
		if attempts <= 3 {
			logger.Debug("Attempt %d: result=%v, err=%v", attempts, result != nil, err)
		}

		// Improved error logging (for debugging)
		if err != nil {
			if attempts%10 == 0 { // Display error every 10 attempts
				logger.Debug("API error after %d attempts (%.1fs): %v", attempts, elapsed.Seconds(), err)
			}
		} else if attempts%20 == 0 {
			// If there is no error but the result is not yet available
			logger.Debug("No result yet after %d attempts (%.1fs)", attempts, elapsed.Seconds())
		}

		// Check if the maximum number of attempts has been reached
		if attempts >= maxAttempts {
			return nil, fmt.Errorf("maximum retry attempts (%d) reached waiting for task result after %.1fs", maxAttempts, elapsed.Seconds())
		}

		// Adjust wait time
		// Check at shorter intervals in interactive mode
		waitTime := 500 * time.Millisecond
		if attempts > 5 {
			waitTime = 1 * time.Second
		}
		if attempts > 20 {
			waitTime = 2 * time.Second
		}

		time.Sleep(waitTime)
	}
}

// formatDuration formats a duration into a human-readable format (common function).
func formatDuration(d time.Duration) string {
	days := int(d.Hours() / 24)
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60

	if days > 0 {
		return fmt.Sprintf("%dd %dh %dm %ds", days, hours, minutes, seconds)
	} else if hours > 0 {
		return fmt.Sprintf("%dh %dm %ds", hours, minutes, seconds)
	} else if minutes > 0 {
		return fmt.Sprintf("%dm %ds", minutes, seconds)
	}
	return fmt.Sprintf("%ds", seconds)
}
