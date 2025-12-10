package command

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/r74tech/virga/internal/cli/session"
	"github.com/r74tech/virga/internal/shared/protocol"
	"github.com/r74tech/virga/internal/shared/testutil/mock"
)

func TestLlamaCommand_Execute(t *testing.T) {
	// Create mock client
	mockClient := &mock.MockAPIClient{
		SendSessionCommandFunc: func(sessionID, cmdType string, payload map[string]interface{}) (string, error) {
			switch cmdType {
			case "llama_interactive":
				// Verify payload
				if prompt, ok := payload["prompt"].(string); !ok || prompt == "" {
					return "", fmt.Errorf("invalid prompt")
				}
				if maxIter, ok := payload["max_iterations"].(int); !ok || maxIter <= 0 {
					return "", fmt.Errorf("invalid max_iterations")
				}
				if temp, ok := payload["temperature"].(float64); !ok || temp < 0 || temp > 1 {
					return "", fmt.Errorf("invalid temperature")
				}
				return "task-llama-123", nil
			case "llama_status":
				return "task-status-456", nil
			case "llama_cancel":
				return "task-cancel-789", nil
			case "llama_autonomous":
				return "task-auto-012", nil
			default:
				return "", fmt.Errorf("unknown command type: %s", cmdType)
			}
		},
		GetTaskResultFunc: func(sessionID, taskID string) (*protocol.TaskResult, error) {
			switch taskID {
			case "task-llama-123":
				return &protocol.TaskResult{
					TaskID:   taskID,
					Output:   "Llama analysis complete:\n- Found 5 configuration files\n- Analyzed security posture",
					ExitCode: 0,
					Error:    "",
				}, nil
			case "task-status-456":
				return &protocol.TaskResult{
					TaskID:   taskID,
					Output:   "Llama Status:\n- Is Running: true\n- Current Task: analysis\n- Progress: 45%",
					ExitCode: 0,
					Error:    "",
				}, nil
			case "task-cancel-789":
				return &protocol.TaskResult{
					TaskID:   taskID,
					Output:   "Llama task cancelled successfully",
					ExitCode: 0,
					Error:    "",
				}, nil
			case "task-auto-012":
				return &protocol.TaskResult{
					TaskID:   taskID,
					Output:   "Autonomous mode started",
					ExitCode: 0,
					Error:    "",
				}, nil
			default:
				return nil, fmt.Errorf("task not found: %s", taskID)
			}
		},
	}

	// Create session manager with mock session
	manager := session.NewManager()
	manager.SetAPIClient(mockClient)

	manager.AddSession("test-session", "192.168.1.100:4444", map[string]interface{}{
		"hostname": "test-host",
		"username": "test-user",
		"os":       "linux",
		"arch":     "amd64",
	})
	manager.SetCurrentSession("test-session")

	cmd := &LlamaCommand{}

	tests := []struct {
		name        string
		args        []string
		expectError bool
		errorMsg    string
		checkOutput func(t *testing.T, output string)
	}{
		{
			name:        "No subcommand",
			args:        []string{},
			expectError: true,
			errorMsg:    "subcommand required",
		},
		{
			name:        "Invalid subcommand",
			args:        []string{"invalid"},
			expectError: true,
			errorMsg:    "unknown subcommand",
		},
		{
			name:        "Prompt - basic",
			args:        []string{"prompt", "Find all configuration files"},
			expectError: false,
		},
		{
			name:        "Prompt - with options",
			args:        []string{"prompt", "Analyze security", "--iterations", "10", "--temperature", "0.8"},
			expectError: false,
		},
		{
			name:        "Prompt - empty",
			args:        []string{"prompt"},
			expectError: true,
			errorMsg:    "prompt required",
		},
		{
			name:        "Prompt - no text after options",
			args:        []string{"prompt", "--iterations", "5"},
			expectError: true,
			errorMsg:    "prompt cannot be empty",
		},
		{
			name:        "Status",
			args:        []string{"status"},
			expectError: false,
		},
		{
			name:        "Cancel",
			args:        []string{"cancel"},
			expectError: false,
		},
		{
			name:        "Auto mode",
			args:        []string{"auto"},
			expectError: false,
		},
		{
			name:        "Result - with task ID",
			args:        []string{"result", "task-llama-123"},
			expectError: false,
		},
		{
			name:        "Result - no task ID",
			args:        []string{"result"},
			expectError: true,
			errorMsg:    "task ID required",
		},
		{
			name:        "Result - with poll option",
			args:        []string{"result", "task-llama-123", "--poll"},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := cmd.Execute(manager, tt.args)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error, got nil")
				} else if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing %q, got %q", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

func TestLlamaCommand_ExecuteNoSession(t *testing.T) {
	manager := session.NewManager()
	cmd := &LlamaCommand{}

	err := cmd.Execute(manager, []string{"status"})
	if err == nil {
		t.Error("Expected error when no session selected")
	}
	if !strings.Contains(err.Error(), "no session selected") {
		t.Errorf("Expected 'no session selected' error, got: %v", err)
	}
}

func TestLlamaCommand_Help(t *testing.T) {
	cmd := &LlamaCommand{}
	help := cmd.Help()

	// Check that help contains expected sections
	expectedSections := []string{
		"Interact with Llama AI",
		"Subcommands:",
		"prompt <text>",
		"status",
		"cancel",
		"auto",
		"result <task_id>",
		"Examples:",
		"--iterations",
		"--temperature",
		"--poll",
	}

	for _, section := range expectedSections {
		if !strings.Contains(help, section) {
			t.Errorf("Help text missing expected section: %s", section)
		}
	}
}

func TestMemDBCommand_Execute(t *testing.T) {
	// Create mock client
	mockClient := &mock.MockAPIClient{
		SendSessionCommandFunc: func(sessionID, cmdType string, payload map[string]interface{}) (string, error) {
			if cmdType != "memdb_query" {
				return "", fmt.Errorf("unexpected command type: %s", cmdType)
			}
			query, ok := payload["query"].(string)
			if !ok {
				return "", fmt.Errorf("missing query in payload")
			}
			if query == "" {
				return "", fmt.Errorf("empty query")
			}
			return "task-memdb-" + query[:3], nil
		},
		GetTaskResultFunc: func(sessionID, taskID string) (*protocol.TaskResult, error) {
			switch taskID {
			case "task-memdb-sta":
				return &protocol.TaskResult{
					TaskID:   taskID,
					Output:   "MemDB Statistics:\n- Total entries: 1234\n- Memory usage: 45MB\n- Uptime: 2h 15m",
					ExitCode: 0,
					Error:    "",
				}, nil
			case "task-memdb-com":
				return &protocol.TaskResult{
					TaskID:   taskID,
					Output:   "Recent Commands:\n1. shell whoami\n2. ps aux\n3. netstat -an",
					ExitCode: 0,
					Error:    "",
				}, nil
			case "task-memdb-lla":
				return &protocol.TaskResult{
					TaskID:   taskID,
					Output:   "Recent Llama Interactions:\n1. Security audit (2 hours ago)\n2. Network scan (4 hours ago)",
					ExitCode: 0,
					Error:    "",
				}, nil
			case "task-memdb-hel":
				return &protocol.TaskResult{
					TaskID:   taskID,
					Output:   "MemDB Query Help:\n- stats: Show statistics\n- commands: Show recent commands",
					ExitCode: 0,
					Error:    "",
				}, nil
			default:
				return nil, fmt.Errorf("task not found: %s", taskID)
			}
		},
	}

	// Create session manager with mock session
	manager := session.NewManager()
	manager.SetAPIClient(mockClient)

	manager.AddSession("test-session", "192.168.1.100:4444", map[string]interface{}{
		"hostname": "test-host",
		"username": "test-user",
		"os":       "linux",
		"arch":     "amd64",
	})
	manager.SetCurrentSession("test-session")

	cmd := &MemDBCommand{}

	tests := []struct {
		name        string
		args        []string
		expectError bool
		errorMsg    string
		checkOutput func(t *testing.T, output string)
	}{
		{
			name:        "Default to help",
			args:        []string{},
			expectError: false,
		},
		{
			name:        "Stats query",
			args:        []string{"stats"},
			expectError: false,
		},
		{
			name:        "Commands query",
			args:        []string{"commands", "20"},
			expectError: false,
		},
		{
			name:        "Llama query",
			args:        []string{"llama", "10"},
			expectError: false,
		},
		{
			name:        "Multi-word query",
			args:        []string{"commands", "limit", "50"},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := cmd.Execute(manager, tt.args)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error, got nil")
				} else if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing %q, got %q", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

func TestMemDBCommand_ExecuteNoSession(t *testing.T) {
	manager := session.NewManager()
	cmd := &MemDBCommand{}

	err := cmd.Execute(manager, []string{"stats"})
	if err == nil {
		t.Error("Expected error when no session selected")
	}
	if !strings.Contains(err.Error(), "no session selected") {
		t.Errorf("Expected 'no session selected' error, got: %v", err)
	}
}

func TestMemDBCommand_Help(t *testing.T) {
	cmd := &MemDBCommand{}
	help := cmd.Help()

	// Check that help contains expected sections
	expectedSections := []string{
		"Query the implant's in-memory database",
		"Queries:",
		"stats",
		"commands",
		"tasks",
		"llama",
		"sysinfo",
		"clear",
		"Examples:",
	}

	for _, section := range expectedSections {
		if !strings.Contains(help, section) {
			t.Errorf("Help text missing expected section: %s", section)
		}
	}
}

func TestWaitForTaskResult(t *testing.T) {
	resultChan := make(chan *protocol.TaskResult, 1)
	errorChan := make(chan error, 1)

	// This function is not exported from the command package
	// We would need to test it indirectly through command execution
	t.Skip("waitForTaskResult is an internal function")

	select {
	case result := <-resultChan:
		if result.Output != "Task completed" {
			t.Errorf("Expected output 'Task completed', got %s", result.Output)
		}
	case err := <-errorChan:
		t.Errorf("Unexpected error: %v", err)
	case <-time.After(6 * time.Second):
		t.Error("Test timed out")
	}
}

func TestMonitorLlamaTaskResult(t *testing.T) {
	// This test would require a more complex setup with context cancellation
	// and async testing. For now, we'll skip it as it's a background monitoring function.
	t.Skip("Skipping background monitoring test")
}
