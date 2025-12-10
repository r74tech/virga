package memdb

import (
	"fmt"
	"strings"
	"time"

	"github.com/r74tech/virga/internal/implant/memdb"
)

// MemDBModule provides access to the in-memory database
type MemDBModule struct {
	db *memdb.DB
}

// NewMemDBModule creates a new memdb module
func NewMemDBModule(db *memdb.DB) *MemDBModule {
	return &MemDBModule{db: db}
}

func (m *MemDBModule) Name() string {
	return "memdb"
}

func (m *MemDBModule) Execute(args []string) (string, int, error) {
	if m.db == nil {
		return "MemDB not initialized", 1, fmt.Errorf("memdb not available")
	}

	if len(args) == 0 || args[0] == "help" {
		return m.showHelp(), 0, nil
	}

	// SQL-like query interface
	if strings.ToUpper(args[0]) == "SELECT" {
		return m.executeQuery(args)
	}

	// Legacy command interface
	switch args[0] {
	case "stats":
		return m.getStats()
	case "commands":
		limit := 10
		if len(args) > 1 {
			fmt.Sscanf(args[1], "%d", &limit)
		}
		return m.getRecentCommands(limit)
	case "tasks":
		return m.getPendingTasks()
	case "llama":
		limit := 5
		if len(args) > 1 {
			fmt.Sscanf(args[1], "%d", &limit)
		}
		return m.getLlamaInteractions(limit)
	case "get":
		if len(args) < 3 {
			return "Usage: memdb get <type> <id>", 1, nil
		}
		return m.getByID(args[1], args[2])
	case "sysinfo":
		hours := 1
		if len(args) > 1 {
			fmt.Sscanf(args[1], "%d", &hours)
		}
		return m.getSystemInfo(hours)
	case "clear":
		hours := 24
		if len(args) > 1 {
			fmt.Sscanf(args[1], "%d", &hours)
		}
		return m.clearOldData(hours)
	default:
		return m.showHelp(), 0, nil
	}
}

func (m *MemDBModule) showHelp() string {
	return `MemDB Query Interface:

=== TABLES ===
  commands  - Shell command execution history
  llama     - Llama AI interactions and autonomous tasks
  tasks     - Pending tasks in queue
  sysinfo   - System information snapshots

=== SQL-STYLE QUERIES ===
  SELECT * FROM commands LIMIT 10
  SELECT * FROM commands WHERE module='llama' LIMIT 5
  SELECT * FROM commands WHERE module='shell' LIMIT 5
  SELECT * FROM commands WHERE exit_code != 0 LIMIT 5
  SELECT * FROM llama ORDER BY timestamp DESC LIMIT 10
  SELECT * FROM llama WHERE status='success' AND temperature > 0.7
  SELECT * FROM llama WHERE task_type='llama_autonomous'
  SELECT * FROM tasks WHERE status='pending'
  SELECT * FROM sysinfo WHERE timestamp > '1h'
  SELECT COUNT(*) FROM commands

=== LEGACY COMMANDS ===
  stats              - Show database statistics
  commands [limit]   - Show recent commands (default: 10)
  tasks              - Show pending tasks
  llama [limit]      - Show recent Llama interactions (default: 5)
                      NOTE: Response is truncated to 5 lines in list view
  sysinfo [hours]    - Show system info history (default: 1 hour)
  clear [hours]      - Clear data older than X hours (default: 24)
  get <type> <id>    - Get specific record by ID with FULL CONTENT
  help               - Show this help message

=== VIEWING FULL LLAMA RESPONSES ===
The 'llama' command shows truncated responses for readability.
To view the COMPLETE response and command details:
  1. First list interactions: memdb llama 10
  2. Note the Task ID from the output (e.g., "auto_system_reconnaissance_123456")
  3. Get full details: memdb get llama <task_id>

Examples:
  memdb llama 5                    # List last 5 Llama interactions (truncated)
  memdb get llama auto_system_reconnaissance_123456  # View FULL response using Task ID

=== MORE EXAMPLES ===
  # Find failed commands
  memdb SELECT * FROM commands WHERE exit_code != 0 LIMIT 5
  
  # Search commands by content
  memdb SELECT command FROM commands WHERE source='user' LIMIT 20
  
  # Get specific command details
  memdb get command d81b98ed-b8d8-489f-ac16-2c2ea05b9d05
  
  # Filter Llama by temperature
  memdb SELECT * FROM llama WHERE temperature > 0.5
  
  # Find autonomous Llama tasks
  memdb SELECT * FROM llama WHERE task_type='llama_autonomous'
  
  # Find successful Llama interactions
  memdb SELECT * FROM llama WHERE status='success' LIMIT 10
  
  # Find Llama interactions with commands
  memdb SELECT * FROM llama WHERE commands_issued > 0
  
  # Find recent Llama interactions (last 2 hours)
  memdb SELECT * FROM llama WHERE timestamp > '2h'
  
  # Count total Llama interactions
  memdb SELECT COUNT(*) FROM llama`
}

func (m *MemDBModule) getStats() (string, int, error) {
	stats, err := m.db.Stats()
	if err != nil {
		return fmt.Sprintf("Failed to get stats: %v", err), 1, err
	}

	var output strings.Builder
	output.WriteString("MemDB Statistics:\n")
	output.WriteString(fmt.Sprintf("  Command Results: %d\n", stats["command_results"]))
	output.WriteString(fmt.Sprintf("  Server Tasks: %d\n", stats["server_tasks"]))
	output.WriteString(fmt.Sprintf("  Llama Interactions: %d\n", stats["llama_interactions"]))
	output.WriteString(fmt.Sprintf("  System Info Records: %d\n", stats["system_info"]))

	return output.String(), 0, nil
}

func (m *MemDBModule) getRecentCommands(limit int) (string, int, error) {
	commands, err := m.db.GetRecentCommands(limit)
	if err != nil {
		return fmt.Sprintf("Failed to get commands: %v", err), 1, err
	}

	if len(commands) == 0 {
		return "No recent commands found", 0, nil
	}

	var output strings.Builder
	output.WriteString(fmt.Sprintf("Recent Commands (last %d):\n", limit))
	for i, cmd := range commands {
		output.WriteString(fmt.Sprintf("\n[%d] %s\n", i+1, cmd.StartTime.Format(time.RFC3339)))
		output.WriteString(fmt.Sprintf("  Module: %s\n", cmd.Module))
		output.WriteString(fmt.Sprintf("  Command: %s\n", cmd.Command))
		if len(cmd.Args) > 0 {
			output.WriteString(fmt.Sprintf("  Args: %v\n", cmd.Args))
		}
		output.WriteString(fmt.Sprintf("  Exit Code: %d\n", cmd.ExitCode))
		if cmd.Error != "" {
			output.WriteString(fmt.Sprintf("  Error: %s\n", cmd.Error))
		}
		if len(cmd.Output) > 200 {
			output.WriteString(fmt.Sprintf("  Output: %s... (truncated)\n", cmd.Output[:200]))
		} else {
			output.WriteString(fmt.Sprintf("  Output: %s\n", cmd.Output))
		}
	}

	return output.String(), 0, nil
}

func (m *MemDBModule) getPendingTasks() (string, int, error) {
	tasks, err := m.db.GetPendingTasks()
	if err != nil {
		return fmt.Sprintf("Failed to get tasks: %v", err), 1, err
	}

	if len(tasks) == 0 {
		return "No pending tasks", 0, nil
	}

	var output strings.Builder
	output.WriteString("Pending Tasks:\n")
	for i, task := range tasks {
		output.WriteString(fmt.Sprintf("\n[%d] %s\n", i+1, task.ReceivedAt.Format(time.RFC3339)))
		output.WriteString(fmt.Sprintf("  ID: %s\n", task.ID))
		output.WriteString(fmt.Sprintf("  Type: %s\n", task.Type))
		output.WriteString(fmt.Sprintf("  Status: %s\n", task.Status))
		if task.Payload != "" {
			output.WriteString(fmt.Sprintf("  Payload: %s\n", task.Payload))
		}
	}

	return output.String(), 0, nil
}

func (m *MemDBModule) getLlamaInteractions(limit int) (string, int, error) {
	interactions, err := m.db.GetRecentLlamaInteractions(limit)
	if err != nil {
		return fmt.Sprintf("Failed to get llama interactions: %v", err), 1, err
	}

	if len(interactions) == 0 {
		return "No recent Llama interactions found", 0, nil
	}

	var output strings.Builder
	output.WriteString(fmt.Sprintf("Recent Llama Interactions (last %d):\n", limit))
	for i, interaction := range interactions {
		output.WriteString(fmt.Sprintf("\n[%d] %s\n", i+1, interaction.StartTime.Format(time.RFC3339)))
		if interaction.TaskID != "" {
			output.WriteString(fmt.Sprintf("  Task ID: %s\n", interaction.TaskID))
		}
		output.WriteString(fmt.Sprintf("  Task Type: %s\n", interaction.TaskType))
		output.WriteString(fmt.Sprintf("  Status: %s\n", interaction.Status))
		output.WriteString(fmt.Sprintf("  Temperature: %.2f\n", interaction.Temperature))

		if interaction.Prompt != "" {
			if len(interaction.Prompt) > 100 {
				output.WriteString(fmt.Sprintf("  Prompt: %s... (truncated)\n", interaction.Prompt[:100]))
			} else {
				output.WriteString(fmt.Sprintf("  Prompt: %s\n", interaction.Prompt))
			}
		}

		// Show full response
		if interaction.Response != "" {
			lines := strings.Split(interaction.Response, "\n")
			output.WriteString("  Response:\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if line == "" {
					continue
				}
				output.WriteString(fmt.Sprintf("    %s\n", line))
			}
		}

		if len(interaction.CommandsIssued) > 0 {
			output.WriteString(fmt.Sprintf("  Commands Issued: %d\n", len(interaction.CommandsIssued)))

			// Try to find command execution results
			commandResults := m.getCommandResultsForLlama(interaction.TaskID, interaction.StartTime, len(interaction.CommandsIssued))

			for j, cmd := range interaction.CommandsIssued {
				if j >= 3 { // Show first 3 commands
					output.WriteString(fmt.Sprintf("    ... and %d more commands\n", len(interaction.CommandsIssued)-3))
					break
				}

				output.WriteString(fmt.Sprintf("    [%d] %s\n", j+1, cmd))

				// Show execution result if available
				if j < len(commandResults) && commandResults[j] != nil {
					result := commandResults[j]
					output.WriteString(fmt.Sprintf("        Exit Code: %d", result.ExitCode))
					if result.Error != "" {
						output.WriteString(fmt.Sprintf(" (Error: %s)", result.Error))
					}
					output.WriteString("\n")

					// Show first line of output if available
					if len(result.Output) > 0 {
						lines := strings.Split(strings.TrimSpace(result.Output), "\n")
						if len(lines) > 0 && lines[0] != "" {
							firstLine := lines[0]
							if len(firstLine) > 60 {
								firstLine = firstLine[:57] + "..."
							}
							output.WriteString(fmt.Sprintf("        Output: %s\n", firstLine))
						}
					}
				}
			}
		} else {
			output.WriteString("  Commands Issued: 0 (No commands generated)\n")
		}
	}

	return output.String(), 0, nil
}

func (m *MemDBModule) getSystemInfo(hours int) (string, int, error) {
	since := time.Now().Add(-time.Duration(hours) * time.Hour)
	infos, err := m.db.GetSystemInfoHistory(since)
	if err != nil {
		return fmt.Sprintf("Failed to get system info: %v", err), 1, err
	}

	if len(infos) == 0 {
		return fmt.Sprintf("No system info records in the last %d hour(s)", hours), 0, nil
	}

	var output strings.Builder
	output.WriteString(fmt.Sprintf("System Info (last %d hour(s)):\n", hours))

	// Show summary
	if len(infos) > 0 {
		latest := infos[len(infos)-1]
		output.WriteString(fmt.Sprintf("\nLatest (%s):\n", latest.Timestamp.Format(time.RFC3339)))
		output.WriteString(fmt.Sprintf("  CPU Usage: %.2f%%\n", latest.CPUUsage))
		output.WriteString(fmt.Sprintf("  Memory Usage: %d MB\n", latest.MemoryUsage/1024/1024))
		output.WriteString(fmt.Sprintf("  Processes: %d\n", latest.Processes))

		if len(latest.DiskUsage) > 0 {
			output.WriteString("  Disk Usage:\n")
			for disk, usage := range latest.DiskUsage {
				output.WriteString(fmt.Sprintf("    %s: %d GB\n", disk, usage/1024/1024/1024))
			}
		}
	}

	output.WriteString(fmt.Sprintf("\nTotal records: %d\n", len(infos)))

	return output.String(), 0, nil
}

func (m *MemDBModule) clearOldData(hours int) (string, int, error) {
	err := m.db.ClearOldData(time.Duration(hours) * time.Hour)
	if err != nil {
		return fmt.Sprintf("Failed to clear old data: %v", err), 1, err
	}

	return fmt.Sprintf("Cleared data older than %d hours", hours), 0, nil
}

// executeQuery executes SQL-like queries
func (m *MemDBModule) executeQuery(args []string) (string, int, error) {
	query := strings.Join(args, " ")
	query = strings.ToUpper(query)

	// Simple SQL parser
	if strings.Contains(query, "FROM COMMANDS") {
		return m.executeCommandQuery(query)
	} else if strings.Contains(query, "FROM LLAMA") {
		return m.executeLlamaQuery(query)
	} else if strings.Contains(query, "FROM TASKS") {
		return m.executeTaskQuery(query)
	} else if strings.Contains(query, "FROM SYSINFO") {
		return m.executeSysinfoQuery(query)
	}

	return "Invalid query. Supported tables: commands, llama, tasks, sysinfo", 1, nil
}

// getByID retrieves a specific record by ID
func (m *MemDBModule) getByID(recordType, id string) (string, int, error) {
	switch recordType {
	case "command":
		// Get command result by task ID
		results, err := m.db.GetRecentCommands(1000) // Search in recent 1000 commands
		if err != nil {
			return fmt.Sprintf("Error: %v", err), 1, err
		}

		for _, result := range results {
			if result.ID == id {
				var output strings.Builder
				output.WriteString(fmt.Sprintf("Command Result (ID: %s)\n", id))
				output.WriteString(fmt.Sprintf("  Time: %s\n", result.StartTime.Format(time.RFC3339)))
				output.WriteString(fmt.Sprintf("  Command: %s\n", result.Command))
				output.WriteString(fmt.Sprintf("  Exit Code: %d\n", result.ExitCode))
				output.WriteString(fmt.Sprintf("  Module: %s\n", result.Module))
				if result.Error != "" {
					output.WriteString(fmt.Sprintf("  Error: %s\n", result.Error))
				}
				output.WriteString("  Output:\n")
				output.WriteString(result.Output)
				return output.String(), 0, nil
			}
		}
		return fmt.Sprintf("Command with ID %s not found", id), 1, nil

	case "llama":
		// Get Llama interaction by TaskID (what's shown to the user)
		interactions, err := m.db.GetRecentLlamaInteractions(100)
		if err != nil {
			return fmt.Sprintf("Error: %v", err), 1, err
		}

		for _, interaction := range interactions {
			// Check both TaskID (what's shown to user) and ID (internal UUID) for backward compatibility
			if interaction.TaskID == id || interaction.ID == id {
				var output strings.Builder
				output.WriteString(fmt.Sprintf("Llama Interaction (ID: %s)\n", id))
				if interaction.TaskID != "" {
					output.WriteString(fmt.Sprintf("  Task ID: %s\n", interaction.TaskID))
				}
				output.WriteString(fmt.Sprintf("  Start: %s\n", interaction.StartTime.Format(time.RFC3339)))
				output.WriteString(fmt.Sprintf("  End: %s\n", interaction.EndTime.Format(time.RFC3339)))
				output.WriteString(fmt.Sprintf("  Task Type: %s\n", interaction.TaskType))
				output.WriteString(fmt.Sprintf("  Status: %s\n", interaction.Status))
				output.WriteString(fmt.Sprintf("  Temperature: %.2f\n", interaction.Temperature))
				output.WriteString(fmt.Sprintf("  Prompt: %s\n", interaction.Prompt))
				output.WriteString("  Response:\n")
				output.WriteString(interaction.Response)
				if len(interaction.CommandsIssued) > 0 {
					output.WriteString(fmt.Sprintf("\n  Commands Issued (%d):\n", len(interaction.CommandsIssued)))
					for _, cmd := range interaction.CommandsIssued {
						output.WriteString(fmt.Sprintf("    - %s\n", cmd))
					}
				}
				return output.String(), 0, nil
			}
		}
		return fmt.Sprintf("Llama interaction with Task ID '%s' not found. Use 'memdb llama' to list available Task IDs.", id), 1, nil

	default:
		return fmt.Sprintf("Unknown record type: %s. Supported: command, llama", recordType), 1, nil
	}
}

// SQL query executors
func (m *MemDBModule) executeCommandQuery(query string) (string, int, error) {
	limit := 10
	if strings.Contains(query, "LIMIT") {
		fmt.Sscanf(query[strings.Index(query, "LIMIT")+6:], "%d", &limit)
	}

	// Check for WHERE clauses
	if strings.Contains(query, "WHERE") {
		if strings.Contains(query, "MODULE='LLAMA'") {
			return m.getCommandsBySource("llama", limit)
		} else if strings.Contains(query, "MODULE='SHELL'") {
			return m.getCommandsBySource("shell", limit)
		} else if strings.Contains(query, "EXIT_CODE != 0") || strings.Contains(query, "EXIT_CODE!=0") {
			return m.getFailedCommands(limit)
		}
	}

	// Check for COUNT
	if strings.Contains(query, "COUNT(*)") {
		stats, _ := m.db.Stats()
		return fmt.Sprintf("Total commands: %d", stats["command_results"]), 0, nil
	}

	return m.getRecentCommands(limit)
}

func (m *MemDBModule) executeLlamaQuery(query string) (string, int, error) {
	limit := 5
	if strings.Contains(query, "LIMIT") {
		fmt.Sscanf(query[strings.Index(query, "LIMIT")+6:], "%d", &limit)
	}

	// Check for COUNT
	if strings.Contains(query, "COUNT(*)") {
		stats, _ := m.db.Stats()
		return fmt.Sprintf("Total Llama interactions: %d", stats["llama_interactions"]), 0, nil
	}

	// Check for WHERE clauses
	upperQuery := strings.ToUpper(query)
	if strings.Contains(upperQuery, "WHERE") {
		// Filter by status
		if strings.Contains(upperQuery, "STATUS='SUCCESS'") {
			return m.getLlamaInteractionsByStatus("success", limit)
		} else if strings.Contains(upperQuery, "STATUS='FAILURE'") || strings.Contains(upperQuery, "STATUS='FAILED'") {
			return m.getLlamaInteractionsByStatus("failure", limit)
		}

		// Filter by task type
		if strings.Contains(upperQuery, "TASK_TYPE='LLAMA_AUTONOMOUS'") {
			return m.getLlamaInteractionsByTaskType("llama_autonomous", limit)
		} else if strings.Contains(upperQuery, "TASK_TYPE='LLAMA'") {
			return m.getLlamaInteractionsByTaskType("llama", limit)
		}

		// Filter by temperature
		if strings.Contains(upperQuery, "TEMPERATURE") {
			var tempOp string
			var tempVal float64
			if idx := strings.Index(upperQuery, "TEMPERATURE"); idx >= 0 {
				tempPart := query[idx+11:] // Skip "TEMPERATURE"
				tempPart = strings.TrimSpace(tempPart)
				if strings.HasPrefix(tempPart, ">") {
					tempOp = ">"
					fmt.Sscanf(tempPart[1:], "%f", &tempVal)
				} else if strings.HasPrefix(tempPart, "<") {
					tempOp = "<"
					fmt.Sscanf(tempPart[1:], "%f", &tempVal)
				} else if strings.HasPrefix(tempPart, "=") {
					tempOp = "="
					fmt.Sscanf(tempPart[1:], "%f", &tempVal)
				}
			}
			if tempOp != "" {
				return m.getLlamaInteractionsByTemperature(tempOp, tempVal, limit)
			}
		}

		// Filter by commands issued
		if strings.Contains(upperQuery, "COMMANDS_ISSUED > 0") {
			return m.getLlamaInteractionsWithCommands(limit)
		}

		// Filter by date range
		if strings.Contains(upperQuery, "TIMESTAMP") {
			// Parse time-based queries like "timestamp > '2h'" or "timestamp < '24h'"
			if idx := strings.Index(upperQuery, "TIMESTAMP"); idx >= 0 {
				timePart := query[idx+9:] // Skip "TIMESTAMP"
				timePart = strings.TrimSpace(timePart)
				var hours int
				var op string
				if strings.HasPrefix(timePart, ">") {
					op = ">"
					if timeIdx := strings.Index(timePart, "'"); timeIdx > 0 {
						timeStr := timePart[timeIdx+1:]
						if hIdx := strings.Index(timeStr, "H'"); hIdx > 0 {
							fmt.Sscanf(timeStr[:hIdx], "%d", &hours)
						}
					}
				} else if strings.HasPrefix(timePart, "<") {
					op = "<"
					if timeIdx := strings.Index(timePart, "'"); timeIdx > 0 {
						timeStr := timePart[timeIdx+1:]
						if hIdx := strings.Index(timeStr, "H'"); hIdx > 0 {
							fmt.Sscanf(timeStr[:hIdx], "%d", &hours)
						}
					}
				}
				if hours > 0 {
					return m.getLlamaInteractionsByTime(op, hours, limit)
				}
			}
		}
	}

	return m.getLlamaInteractions(limit)
}

func (m *MemDBModule) executeTaskQuery(query string) (string, int, error) {
	return m.getPendingTasks()
}

func (m *MemDBModule) executeSysinfoQuery(query string) (string, int, error) {
	hours := 1
	if strings.Contains(query, "TIMESTAMP > '") {
		timeStr := query[strings.Index(query, "TIMESTAMP > '")+13:]
		if idx := strings.Index(timeStr, "H'"); idx > 0 {
			fmt.Sscanf(timeStr[:idx], "%d", &hours)
		}
	}

	return m.getSystemInfo(hours)
}

// Helper functions for filtered queries
func (m *MemDBModule) getCommandsBySource(source string, limit int) (string, int, error) {
	results, err := m.db.GetRecentCommands(limit * 5) // Get more to filter
	if err != nil {
		return fmt.Sprintf("Failed to get commands: %v", err), 1, err
	}

	var filtered []memdb.CommandResult
	for _, result := range results {
		if strings.ToLower(result.Module) == source && len(filtered) < limit {
			filtered = append(filtered, *result)
		}
	}

	if len(filtered) == 0 {
		return fmt.Sprintf("No commands from source '%s'", source), 0, nil
	}

	var output strings.Builder
	output.WriteString(fmt.Sprintf("Commands from source '%s' (last %d):\n", source, len(filtered)))

	for i, result := range filtered {
		output.WriteString(fmt.Sprintf("\n[%d] %s (ID: %s)\n", i+1, result.StartTime.Format(time.RFC3339), result.ID))
		output.WriteString(fmt.Sprintf("  Command: %s\n", result.Command))
		output.WriteString(fmt.Sprintf("  Exit Code: %d\n", result.ExitCode))
		if result.Error != "" {
			output.WriteString(fmt.Sprintf("  Error: %s\n", result.Error))
		}
	}

	return output.String(), 0, nil
}

func (m *MemDBModule) getFailedCommands(limit int) (string, int, error) {
	results, err := m.db.GetRecentCommands(limit * 5) // Get more to filter
	if err != nil {
		return fmt.Sprintf("Failed to get commands: %v", err), 1, err
	}

	var failed []memdb.CommandResult
	for _, result := range results {
		if result.ExitCode != 0 && len(failed) < limit {
			failed = append(failed, *result)
		}
	}

	if len(failed) == 0 {
		return "No failed commands found", 0, nil
	}

	var output strings.Builder
	output.WriteString(fmt.Sprintf("Failed commands (last %d):\n", len(failed)))

	for i, result := range failed {
		output.WriteString(fmt.Sprintf("\n[%d] %s (ID: %s)\n", i+1, result.StartTime.Format(time.RFC3339), result.ID))
		output.WriteString(fmt.Sprintf("  Command: %s\n", result.Command))
		output.WriteString(fmt.Sprintf("  Exit Code: %d\n", result.ExitCode))
		if result.Error != "" {
			output.WriteString(fmt.Sprintf("  Error: %s\n", result.Error))
		}
		if len(result.Output) > 200 {
			output.WriteString(fmt.Sprintf("  Output: %s...\n", result.Output[:200]))
		} else {
			output.WriteString(fmt.Sprintf("  Output: %s\n", result.Output))
		}
	}

	return output.String(), 0, nil
}

// Helper functions for Llama query filtering
func (m *MemDBModule) getLlamaInteractionsByStatus(status string, limit int) (string, int, error) {
	interactions, err := m.db.GetRecentLlamaInteractions(limit * 3) // Get more to filter
	if err != nil {
		return fmt.Sprintf("Failed to get Llama interactions: %v", err), 1, err
	}

	var filtered []memdb.LlamaInteraction
	for _, interaction := range interactions {
		if strings.ToLower(interaction.Status) == status && len(filtered) < limit {
			filtered = append(filtered, *interaction)
		}
	}

	if len(filtered) == 0 {
		return fmt.Sprintf("No Llama interactions with status '%s'", status), 0, nil
	}

	return m.formatLlamaInteractions(filtered, fmt.Sprintf("Llama interactions with status '%s'", status))
}

func (m *MemDBModule) getLlamaInteractionsByTaskType(taskType string, limit int) (string, int, error) {
	interactions, err := m.db.GetRecentLlamaInteractions(limit * 3)
	if err != nil {
		return fmt.Sprintf("Failed to get Llama interactions: %v", err), 1, err
	}

	var filtered []memdb.LlamaInteraction
	for _, interaction := range interactions {
		if strings.ToLower(interaction.TaskType) == taskType && len(filtered) < limit {
			filtered = append(filtered, *interaction)
		}
	}

	if len(filtered) == 0 {
		return fmt.Sprintf("No Llama interactions with task type '%s'", taskType), 0, nil
	}

	return m.formatLlamaInteractions(filtered, fmt.Sprintf("Llama interactions with task type '%s'", taskType))
}

func (m *MemDBModule) getLlamaInteractionsByTemperature(op string, temp float64, limit int) (string, int, error) {
	interactions, err := m.db.GetRecentLlamaInteractions(limit * 3)
	if err != nil {
		return fmt.Sprintf("Failed to get Llama interactions: %v", err), 1, err
	}

	var filtered []memdb.LlamaInteraction
	for _, interaction := range interactions {
		match := false
		switch op {
		case ">":
			match = float64(interaction.Temperature) > temp
		case "<":
			match = float64(interaction.Temperature) < temp
		case "=":
			match = float64(interaction.Temperature) == temp
		}
		if match && len(filtered) < limit {
			filtered = append(filtered, *interaction)
		}
	}

	if len(filtered) == 0 {
		return fmt.Sprintf("No Llama interactions with temperature %s %.2f", op, temp), 0, nil
	}

	return m.formatLlamaInteractions(filtered, fmt.Sprintf("Llama interactions with temperature %s %.2f", op, temp))
}

func (m *MemDBModule) getLlamaInteractionsWithCommands(limit int) (string, int, error) {
	interactions, err := m.db.GetRecentLlamaInteractions(limit * 2)
	if err != nil {
		return fmt.Sprintf("Failed to get Llama interactions: %v", err), 1, err
	}

	var filtered []memdb.LlamaInteraction
	for _, interaction := range interactions {
		if len(interaction.CommandsIssued) > 0 && len(filtered) < limit {
			filtered = append(filtered, *interaction)
		}
	}

	if len(filtered) == 0 {
		return "No Llama interactions with commands issued", 0, nil
	}

	return m.formatLlamaInteractions(filtered, "Llama interactions with commands issued")
}

func (m *MemDBModule) getLlamaInteractionsByTime(op string, hours int, limit int) (string, int, error) {
	interactions, err := m.db.GetRecentLlamaInteractions(limit * 3)
	if err != nil {
		return fmt.Sprintf("Failed to get Llama interactions: %v", err), 1, err
	}

	cutoffTime := time.Now().Add(-time.Duration(hours) * time.Hour)
	var filtered []memdb.LlamaInteraction

	for _, interaction := range interactions {
		match := false
		switch op {
		case ">":
			match = interaction.StartTime.After(cutoffTime)
		case "<":
			match = interaction.StartTime.Before(cutoffTime)
		}
		if match && len(filtered) < limit {
			filtered = append(filtered, *interaction)
		}
	}

	if len(filtered) == 0 {
		return fmt.Sprintf("No Llama interactions %s %d hours ago", op, hours), 0, nil
	}

	timeDesc := "newer than"
	if op == "<" {
		timeDesc = "older than"
	}
	return m.formatLlamaInteractions(filtered, fmt.Sprintf("Llama interactions %s %d hours", timeDesc, hours))
}

// Helper function to format Llama interactions consistently
func (m *MemDBModule) formatLlamaInteractions(interactions []memdb.LlamaInteraction, title string) (string, int, error) {
	var output strings.Builder
	output.WriteString(fmt.Sprintf("%s (last %d):\n", title, len(interactions)))

	for i, interaction := range interactions {
		output.WriteString(fmt.Sprintf("\n[%d] %s\n", i+1, interaction.StartTime.Format("2006-01-02T15:04:05-07:00")))
		output.WriteString(fmt.Sprintf("  Task ID: %s\n", interaction.TaskID))
		output.WriteString(fmt.Sprintf("  Task Type: %s\n", interaction.TaskType))
		output.WriteString(fmt.Sprintf("  Status: %s\n", interaction.Status))
		output.WriteString(fmt.Sprintf("  Temperature: %.2f\n", interaction.Temperature))

		// Truncate prompt if too long
		prompt := interaction.Prompt
		if len(prompt) > 100 {
			prompt = prompt[:97] + "..."
		}
		output.WriteString(fmt.Sprintf("  Prompt: %s\n", prompt))

		// Format response
		output.WriteString("  Response:\n")
		lines := strings.Split(strings.TrimSpace(interaction.Response), "\n")
		displayedLines := 0
		for _, line := range lines {
			if line = strings.TrimSpace(line); line != "" {
				displayedLines++
				if displayedLines > 5 {
					output.WriteString(fmt.Sprintf("    ... and %d more lines\n", len(lines)-5))
					break
				}
				if len(line) > 80 {
					line = line[:77] + "..."
				}
				output.WriteString(fmt.Sprintf("    %s\n", line))
			}
		}

		// Show commands if any
		if len(interaction.CommandsIssued) > 0 {
			output.WriteString(fmt.Sprintf("  Commands Issued: %d\n", len(interaction.CommandsIssued)))

			// Try to find command execution results
			commandResults := m.getCommandResultsForLlama(interaction.TaskID, interaction.StartTime, len(interaction.CommandsIssued))

			for j, cmd := range interaction.CommandsIssued {
				if j >= 3 {
					output.WriteString(fmt.Sprintf("    ... and %d more commands\n", len(interaction.CommandsIssued)-3))
					break
				}

				output.WriteString(fmt.Sprintf("    [%d] %s\n", j+1, cmd))

				// Show execution result if available
				if j < len(commandResults) && commandResults[j] != nil {
					result := commandResults[j]
					output.WriteString(fmt.Sprintf("        Exit Code: %d", result.ExitCode))
					if result.Error != "" {
						output.WriteString(fmt.Sprintf(" (Error: %s)", result.Error))
					}
					output.WriteString("\n")

					// Show first line of output if available
					if len(result.Output) > 0 {
						lines := strings.Split(strings.TrimSpace(result.Output), "\n")
						if len(lines) > 0 && lines[0] != "" {
							firstLine := lines[0]
							if len(firstLine) > 60 {
								firstLine = firstLine[:57] + "..."
							}
							output.WriteString(fmt.Sprintf("        Output: %s\n", firstLine))
						}
					}
				}
			}
		} else {
			output.WriteString("  Commands Issued: 0 (No commands generated)\n")
		}
	}

	output.WriteString(fmt.Sprintf("\nNote: Use 'memdb get llama <task_id>' with the Task ID shown above to view full response and all commands"))

	return output.String(), 0, nil
}

// Helper function to get command results associated with a Llama interaction
func (m *MemDBModule) getCommandResultsForLlama(taskID string, startTime time.Time, expectedCount int) []*memdb.CommandResult {
	// Get recent commands that were executed after the Llama interaction started
	// and are from the "llama" module
	results, err := m.db.GetRecentCommands(expectedCount + 10) // Get a few extra to ensure we find them
	if err != nil {
		return nil
	}

	var commandResults []*memdb.CommandResult
	for _, result := range results {
		// Check if this command was issued by Llama and after the interaction started
		if result.Module == "llama" && result.StartTime.After(startTime) {
			commandResults = append(commandResults, result)
			if len(commandResults) >= expectedCount {
				break
			}
		}
	}

	return commandResults
}
