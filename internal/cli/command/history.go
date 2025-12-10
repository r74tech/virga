package command

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/r74tech/virga/internal/cli/session"
	"github.com/r74tech/virga/internal/shared/logger"
)

// HistoryEntry represents a single command history entry
type HistoryEntry struct {
	Timestamp   time.Time              `json:"timestamp"`
	Command     string                 `json:"command"`
	SessionID   string                 `json:"session_id,omitempty"`
	SessionInfo map[string]interface{} `json:"session_info,omitempty"`
	Result      string                 `json:"result,omitempty"`
	Success     bool                   `json:"success"`
}

// HistoryCommand is the history command
type HistoryCommand struct{}

// Execute executes the history command
func (c *HistoryCommand) Execute(sessionManager *session.Manager, args []string) error {
	// Parse flags
	var showDetails, showVerbose bool
	var limit int = 50 // Default to last 50 commands

	// Simple flag parsing
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-d", "--details":
			showDetails = true
		case "-v", "--verbose":
			showVerbose = true
		case "-n", "--number":
			if i+1 < len(args) {
				fmt.Sscanf(args[i+1], "%d", &limit)
				i++ // Skip next arg
			}
		case "-h", "--help":
			fmt.Println(c.Help())
			return nil
		}
	}

	// Get history file path
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	historyFile := filepath.Join(homeDir, ".config", "virga", "history.json")

	// Read history file
	data, err := os.ReadFile(historyFile)
	if err != nil {
		if os.IsNotExist(err) {
			logger.Info("No command history found.")
			return nil
		}
		return fmt.Errorf("failed to read history file: %w", err)
	}

	// Parse history entries
	var entries []HistoryEntry
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var entry HistoryEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			// Skip invalid entries
			continue
		}
		entries = append(entries, entry)
	}

	// Limit the number of entries
	if len(entries) > limit {
		entries = entries[len(entries)-limit:]
	}

	// Display history
	if showVerbose {
		// Verbose output with results
		for i, entry := range entries {
			fmt.Printf("\n[%d] %s\n", i+1, entry.Timestamp.Format("2006-01-02 15:04:05"))

			if showDetails && entry.SessionID != "" {
				fmt.Printf("  Session: %s\n", entry.SessionID)
				if entry.SessionInfo != nil {
					if hostname, ok := entry.SessionInfo["hostname"].(string); ok {
						fmt.Printf("  Host: %s", hostname)
					}
					if username, ok := entry.SessionInfo["username"].(string); ok {
						fmt.Printf(" | User: %s", username)
					}
					fmt.Println()
				}
			}

			fmt.Printf("  Command: %s\n", entry.Command)

			if entry.Result != "" {
				fmt.Printf("  Result: %s\n", entry.Result)
			}

			if entry.Success {
				fmt.Printf("  Status: Success\n")
			} else {
				fmt.Printf("  Status: Failed\n")
			}
		}
	} else if showDetails {
		// Detailed output with session info
		fmt.Printf("%-20s %-40s %-20s %s\n", "Timestamp", "Command", "Session", "Host/User")
		fmt.Println(strings.Repeat("-", 100))

		for _, entry := range entries {
			timestamp := entry.Timestamp.Format("2006-01-02 15:04:05")
			command := truncateString(entry.Command, 40)

			sessionInfo := ""
			if entry.SessionID != "" {
				sessionInfo = truncateString(entry.SessionID, 20)
			} else {
				sessionInfo = "local"
			}

			hostUser := ""
			if entry.SessionInfo != nil {
				if hostname, ok := entry.SessionInfo["hostname"].(string); ok {
					hostUser = hostname
				}
				if username, ok := entry.SessionInfo["username"].(string); ok {
					if hostUser != "" {
						hostUser += "/" + username
					} else {
						hostUser = username
					}
				}
			}

			fmt.Printf("%-20s %-40s %-20s %s\n", timestamp, command, sessionInfo, hostUser)
		}
	} else {
		// Simple output
		for i, entry := range entries {
			fmt.Printf("%5d  %s\n", i+1, entry.Command)
		}
	}

	return nil
}

// Help returns the help message for the history command
func (c *HistoryCommand) Help() string {
	return `Show command history.

Usage: history [options]

Options:
  -n, --number <count>  Show last N commands (default: 50)
  -d, --details         Show session details (timestamp, session ID, host/user)
  -v, --verbose         Show command results and detailed information
  -h, --help            Show this help message

Examples:
  history                    Show last 50 commands
  history -n 100            Show last 100 commands
  history -d                Show commands with session details
  history -v                Show commands with results
  history -d -v             Show full command history details`
}

// truncateString truncates a string to the specified length
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}
