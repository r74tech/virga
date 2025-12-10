package command

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/r74tech/virga/internal/cli/session"
	"github.com/r74tech/virga/internal/shared/logger"
)

// LlamaCommand is the implementation of the 'llama' command.
type LlamaCommand struct{}

// Execute executes the 'llama' command.
func (c *LlamaCommand) Execute(sessionManager *session.Manager, args []string) error {
	// Check the current session
	currentSession := sessionManager.GetCurrentSession()
	if currentSession == nil {
		return fmt.Errorf("no session selected. Use 'interact <session_id>' first")
	}

	if len(args) == 0 {
		return fmt.Errorf("subcommand required. Usage: llama <prompt|status> [options]")
	}

	switch args[0] {
	case "prompt":
		if len(args) < 2 {
			return fmt.Errorf("prompt required. Usage: llama prompt <prompt> [--iterations N] [--temperature F]")
		}

		// Default values
		maxIterations := 5
		temperature := 0.3

		// Parse prompt and options
		promptParts := []string{}
		i := 1
		for i < len(args) {
			if args[i] == "--iterations" && i+1 < len(args) {
				if val, err := strconv.Atoi(args[i+1]); err == nil {
					maxIterations = val
				}
				i += 2
			} else if args[i] == "--temperature" && i+1 < len(args) {
				if val, err := strconv.ParseFloat(args[i+1], 64); err == nil {
					temperature = val
				}
				i += 2
			} else {
				promptParts = append(promptParts, args[i])
				i++
			}
		}

		prompt := strings.Join(promptParts, " ")
		if prompt == "" {
			return fmt.Errorf("prompt cannot be empty")
		}

		logger.Info("Sending Llama prompt: %s", prompt)
		logger.Info("Max iterations: %d, Temperature: %.2f", maxIterations, temperature)

		// Execute Llama task
		payload := map[string]interface{}{
			"prompt":         prompt,
			"max_iterations": maxIterations,
			"temperature":    temperature,
		}

		taskID, err := currentSession.ExecuteCommand("llama_interactive", payload)
		if err != nil {
			return fmt.Errorf("error executing Llama prompt: %v", err)
		}

		logger.Info("\n=== Llama Task Started ===")
		logger.Info("Task ID: %s", taskID)
		logger.Info("\nProcessing Llama request in background...")
		logger.Info("Use 'llama result %s' to get the result when complete.", taskID)
		logger.Info("Use 'llama result %s --poll' to wait for the result.", taskID)
		logger.Info("Use 'llama status' to check progress or 'llama cancel' to stop execution.")

		// Start background monitoring for auto-notification
		ctx := context.Background()
		go monitorLlamaTaskResult(ctx, currentSession, taskID, prompt)

		return nil

	case "cancel":
		// Cancel active Llama task
		taskID, err := currentSession.ExecuteCommand("llama_cancel", map[string]interface{}{})
		if err != nil {
			return fmt.Errorf("error cancelling Llama task: %v", err)
		}

		result, err := waitForTaskResult(currentSession, taskID, 10)
		if err != nil {
			return fmt.Errorf("error waiting for cancel result: %v", err)
		}

		logger.Info(result.Output)
		return nil

	case "status":
		// Get Llama status
		taskID, err := currentSession.ExecuteCommand("llama_status", map[string]interface{}{})
		if err != nil {
			return fmt.Errorf("error getting Llama status: %v", err)
		}

		result, err := waitForTaskResult(currentSession, taskID, 30)
		if err != nil {
			return fmt.Errorf("error waiting for status result: %v", err)
		}

		logger.Info("=== Llama Status ===")
		logger.Info(result.Output)
		return nil

	case "auto":
		// Start autonomous mode
		logger.Info("Starting Llama autonomous mode...")

		tasks := []map[string]string{
			{"type": "system_reconnaissance", "description": "Perform system reconnaissance"},
			{"type": "user_activity", "description": "Analyze user activities"},
			{"type": "network_discovery", "description": "Discover network topology"},
		}

		payload := map[string]interface{}{
			"tasks":           tasks,
			"report_interval": 300, // 5 minutes
		}

		taskID, err := currentSession.ExecuteCommand("llama_autonomous", payload)
		if err != nil {
			return fmt.Errorf("error starting autonomous mode: %v", err)
		}

		logger.Info("Autonomous mode started (Task ID: %s)", taskID)
		logger.Info("Reports will be sent to C2 server every 5 minutes.")
		return nil

	case "tasks":
		// List recent Llama tasks
		limit := 20 // default
		if len(args) >= 2 {
			if args[1] == "--limit" && len(args) >= 3 {
				if val, err := strconv.Atoi(args[2]); err == nil && val > 0 {
					limit = val
				}
			}
		}

		payload := map[string]interface{}{
			"limit": limit,
		}

		taskID, err := currentSession.ExecuteCommand("llama_list_tasks", payload)
		if err != nil {
			return fmt.Errorf("error executing llama tasks: %v", err)
		}

		// Wait for result (synchronous operation)
		result, err := waitForTaskResult(currentSession, taskID, 10)
		if err != nil {
			return fmt.Errorf("error waiting for tasks result: %v", err)
		}

		if result == nil {
			return fmt.Errorf("no result received for tasks command")
		}

		logger.Info("\n=== Recent Llama Tasks ===")
		logger.Info(result.Output)

		return nil

	case "result":
		// Get Llama task result
		if len(args) < 2 {
			return fmt.Errorf("task ID required. Usage: llama result <task_id> [--poll] [--poll-interval N]")
		}

		taskID := args[1]
		pollMode := false
		pollInterval := 5 // default 5 seconds

		// Parse options
		for i := 2; i < len(args); i++ {
			if args[i] == "--poll" {
				pollMode = true
			} else if args[i] == "--poll-interval" && i+1 < len(args) {
				if val, err := strconv.Atoi(args[i+1]); err == nil && val > 0 {
					pollInterval = val
				}
				i++
			}
		}

		if pollMode {
			logger.Info("Polling for task %s result every %d seconds...", taskID, pollInterval)
			logger.Info("Press Ctrl+C to stop polling.")

			for {
				result, err := currentSession.GetTaskResult(taskID)
				if err != nil {
					logger.Error("Error getting task result: %v", err)
					time.Sleep(time.Duration(pollInterval) * time.Second)
					continue
				}

				// Check if this is just the initial "started in background" message
				if strings.Contains(result.Output, "started in background") {
					logger.Debug("Task %s is still running...", taskID)
					time.Sleep(time.Duration(pollInterval) * time.Second)
					continue
				}

				// Real result received
				logger.Info("\n=== Llama Task Result ===")
				logger.Info("Task ID: %s", taskID)
				logger.Info("Exit Code: %d", result.ExitCode)
				if result.Error != "" {
					logger.Error("Error: %s", result.Error)
				}
				logger.Info("\n--- Output ---")
				logger.Info(result.Output)
				break
			}
		} else {
			// Single check
			result, err := currentSession.GetTaskResult(taskID)
			if err != nil {
				return fmt.Errorf("error getting task result: %v", err)
			}

			if result == nil {
				return fmt.Errorf("task result not found for ID: %s (task may still be running or completed without storing result)", taskID)
			}

			logger.Info("\n=== Llama Task Result ===")
			logger.Info("Task ID: %s", taskID)
			logger.Info("Exit Code: %d", result.ExitCode)
			if result.Error != "" {
				logger.Error("Error: %s", result.Error)
			}
			logger.Info("\n--- Output ---")
			logger.Info(result.Output)
		}

		return nil

	default:
		return fmt.Errorf("unknown subcommand: %s", args[0])
	}
}

// Help returns the help for the 'llama' command.
func (c *LlamaCommand) Help() string {
	return `Interact with Llama AI on the target system.
Usage: llama <subcommand> [options]

Subcommands:
  prompt <text>         Send a prompt to Llama AI (runs in background)
    --iterations N      Maximum iterations (default: 5)
    --temperature F     Temperature 0.0-1.0 (default: 0.3)

  tasks                List recent Llama tasks from implant MemDB
    --limit N          Number of tasks to show (default: 20)

  result <task_id>      Get result of a Llama task
    --poll              Poll for result until task completes
    --poll-interval N   Polling interval in seconds (default: 5)

  status               Check Llama integration status
  cancel               Cancel active Llama task
  auto                 Start autonomous reconnaissance mode

Examples:
  llama prompt "Find all services running as SYSTEM"
  llama prompt "Check for privilege escalation paths" --iterations 10
  llama tasks --limit 10
  llama result auto_system_reconnaissance_1762349811703389000
  llama result a1b2c3d4-e5f6-7890-abcd-ef1234567890 --poll --poll-interval 3
  llama status
  llama cancel`
}

// monitorLlamaTaskResult monitors a Llama task in the background and notifies when complete
func monitorLlamaTaskResult(ctx context.Context, session *session.Session, taskID string, prompt string) {
	// Wait a bit before starting to poll
	time.Sleep(2 * time.Second)

	initialPollInterval := 5 * time.Second
	maxAttempts := 120 // 10 minutes max
	attempts := 0
	pollInterval := initialPollInterval

	for attempts < maxAttempts {
		select {
		case <-ctx.Done():
			logger.Info("Polling canceled by user.")
			return
		default:
			result, err := session.GetTaskResult(taskID)
			if err != nil || result == nil {
				// Task result not found yet
				attempts++
				time.Sleep(pollInterval)
				pollInterval = time.Duration(float64(pollInterval) * 1.5) // Exponential backoff
				if pollInterval > 30*time.Second {
					pollInterval = 30 * time.Second // Cap at 30 seconds
				}
				continue
			}

			// Check if this is just the initial "started in background" message
			if strings.Contains(result.Output, "started in background") {
				attempts++
				time.Sleep(pollInterval)
				pollInterval = time.Duration(float64(pollInterval) * 1.5) // Exponential backoff
				if pollInterval > 30*time.Second {
					pollInterval = 30 * time.Second // Cap at 30 seconds
				}
				continue
			}

			// Real result received - notify user
			logger.Info("\n\n")
			logger.Info("================================================================================")
			logger.Info("🔔 LLAMA TASK COMPLETED! (Task ID: %s)", taskID)
			logger.Info("================================================================================")
			logger.Info("Prompt: %s", prompt)
			logger.Info("Exit Code: %d", result.ExitCode)
			if result.Error != "" {
				logger.Error("Error: %s", result.Error)
			}
			logger.Info("\n--- Result Preview ---")

			// Show preview of output (first 500 chars)
			preview := result.Output
			if len(preview) > 500 {
				preview = preview[:500] + "...\n[Output truncated]"
			}
			logger.Info(preview)

			logger.Info("\n--------------------------------------------------------------------------------")
			logger.Info("To see full result, run: llama result %s", taskID)
			logger.Info("================================================================================")
			logger.Info("\n")

			// Play a bell sound if terminal supports it
			fmt.Print("\a")

			return
		}
	}

	// Timeout reached
	logger.Info("\n\n")
	logger.Info("================================================================================")
	logger.Info("⏱️  LLAMA TASK TIMEOUT (Task ID: %s)", taskID)
	logger.Info("================================================================================")
	logger.Info("Task has been running for over 10 minutes.")
	logger.Info("Use 'llama result %s' to manually check the result.", taskID)
	logger.Info("Use 'llama status' to check if it's still running.")
	logger.Info("================================================================================")
	logger.Info("\n")
}

// MemDBCommand is the implementation of the 'memdb' command.
type MemDBCommand struct{}

// Execute executes the 'memdb' command.
func (c *MemDBCommand) Execute(sessionManager *session.Manager, args []string) error {
	// Check the current session
	currentSession := sessionManager.GetCurrentSession()
	if currentSession == nil {
		return fmt.Errorf("no session selected. Use 'interact <session_id>' first")
	}

	if len(args) == 0 {
		args = []string{"help"} // Default to help
	}

	// Build memdb query
	query := strings.Join(args, " ")

	// Execute memdb query
	payload := map[string]interface{}{
		"query": query,
	}

	taskID, err := currentSession.ExecuteCommand("memdb_query", payload)
	if err != nil {
		return fmt.Errorf("error executing MemDB query: %v", err)
	}

	// Wait for result
	result, err := waitForTaskResult(currentSession, taskID, 30)
	if err != nil {
		return fmt.Errorf("error waiting for MemDB result: %v", err)
	}

	// Display result
	logger.Info(result.Output)
	if result.ExitCode != 0 {
		logger.Error("\nError: Query failed with exit code %d", result.ExitCode)
		if result.Error != "" {
			logger.Error("Error details: %s", result.Error)
		}
	}

	return nil
}

// Help returns the help for the 'memdb' command.
func (c *MemDBCommand) Help() string {
	return `Query the implant's in-memory database.
Usage: memdb <query> [options]

Queries:
  stats                 Show database statistics
  commands [limit]      Show recent commands (default: 10)
  tasks                 Show pending tasks
  llama [limit]         Show recent Llama interactions (default: 5)
  sysinfo [hours]       Show system info history (default: 1 hour)
  clear [hours]         Clear data older than X hours (default: 24)

Examples:
  memdb stats
  memdb commands 20
  memdb llama 10
  memdb sysinfo 24`
}
