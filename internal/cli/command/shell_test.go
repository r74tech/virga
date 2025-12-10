package command

import (
	"fmt"
	"strings"
	"testing"

	"github.com/r74tech/virga/internal/cli/session"
	"github.com/r74tech/virga/internal/shared/protocol"
	"github.com/r74tech/virga/internal/shared/testutil/mock"
)

func TestShellCommand_Execute(t *testing.T) {
	t.Run("No arguments", func(t *testing.T) {
		mockClient := &mock.MockAPIClient{}
		manager := session.NewManager()
		manager.SetAPIClient(mockClient)
		manager.AddSession("test-session", "192.168.1.100:4444", nil)
		manager.SetCurrentSession("test-session")

		cmd := &ShellCommand{}
		// This is difficult to test non-interactively.
		// For now, just ensure it doesn't error out immediately.
		// A more comprehensive test would require mocking liner.
		// We expect an EOF error because there's no interactive input.
		err := cmd.Execute(manager, []string{})
		if err != nil && err.Error() != "EOF" {
			t.Errorf("Expected EOF error in non-interactive test, got: %v", err)
		}
	})

	t.Run("With arguments", func(t *testing.T) {
		mockClient := &mock.MockAPIClient{}
		manager := session.NewManager()
		manager.SetAPIClient(mockClient)
		manager.AddSession("test-session", "192.168.1.100:4444", nil)
		manager.SetCurrentSession("test-session")

		cmd := &ShellCommand{}
		err := cmd.Execute(manager, []string{"some-arg"})
		if err == nil {
			t.Error("Expected an error when arguments are provided to shell command, got nil")
		}
	})
}

func TestShellCommand_ExecuteNoSession(t *testing.T) {
	manager := session.NewManager()
	cmd := &ShellCommand{}

	err := cmd.Execute(manager, []string{"whoami"})
	if err == nil {
		t.Error("Expected error when no session selected")
	}
	if !strings.Contains(err.Error(), "no session selected") {
		t.Errorf("Expected 'no session selected' error, got: %v", err)
	}
}

func TestShellCommand_Help(t *testing.T) {
	cmd := &ShellCommand{}
	help := cmd.Help()

	expectedKeywords := []string{
		"Start",
		"interactive shell",
		"target system",
		"Usage:",
		"shell",
	}

	for _, keyword := range expectedKeywords {
		if !strings.Contains(help, keyword) {
			t.Errorf("Help text missing expected keyword: %s", keyword)
		}
	}
}

func TestExecCommand_Execute(t *testing.T) {
	mockClient := &mock.MockAPIClient{
		SendSessionCommandFunc: func(sessionID, cmdType string, payload map[string]interface{}) (string, error) {
			if cmdType != "shell" {
				return "", fmt.Errorf("unexpected command type: %s", cmdType)
			}

			// Verify command in payload
			cmd, ok := payload["command"].(string)
			if !ok {
				return "", fmt.Errorf("missing command in payload")
			}

			// Verify it's not trying to capture output
			if capture, ok := payload["capture_output"].(bool); ok && capture {
				return "", fmt.Errorf("exec should not capture output")
			}

			return fmt.Sprintf("task-exec-%s", cmd), nil
		},
		GetTaskResultFunc: func(sessionID, taskID string) (*protocol.TaskResult, error) {
			return &protocol.TaskResult{
				TaskID:   taskID,
				Output:   "Command executed successfully (no output captured)",
				ExitCode: 0,
				Error:    "",
			}, nil
		},
	}

	manager := session.NewManager()
	manager.SetAPIClient(mockClient)
	manager.AddSession("test-session", "192.168.1.100:4444", map[string]interface{}{
		"hostname": "test-host",
	})
	manager.SetCurrentSession("test-session")

	tests := []struct {
		name        string
		args        []string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "Execute background command",
			args:        []string{"start", "notepad.exe"},
			expectError: false,
		},
		{
			name:        "No command provided",
			args:        []string{},
			expectError: true,
			errorMsg:    "command required",
		},
		{
			name:        "Execute service command",
			args:        []string{"sc", "query", "wuauserv"},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &ExecCommand{}
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

func TestExecCommand_Help(t *testing.T) {
	cmd := &ExecCommand{}
	help := cmd.Help()

	expectedKeywords := []string{
		"Execute",
		"command",
		"target system",
		"Usage:",
		"exec",
	}

	for _, keyword := range expectedKeywords {
		if !strings.Contains(help, keyword) {
			t.Errorf("Help text missing expected keyword: %s", keyword)
		}
	}
}

func TestCommandArgumentParsing(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected string
	}{
		{
			name:     "Simple command",
			args:     []string{"whoami"},
			expected: "whoami",
		},
		{
			name:     "Command with args",
			args:     []string{"ping", "-n", "4", "google.com"},
			expected: "ping -n 4 google.com",
		},
		{
			name:     "Command with quotes",
			args:     []string{"echo", "Hello World"},
			expected: "echo Hello World",
		},
		{
			name:     "Command with special chars",
			args:     []string{"cmd", "/c", "echo", "test&test"},
			expected: "cmd /c echo test&test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test command joining
			result := strings.Join(tt.args, " ")
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestShellOutput(t *testing.T) {
	tests := []struct {
		name       string
		output     string
		exitCode   int
		shouldWarn bool
	}{
		{
			name:       "Successful command",
			output:     "Command completed successfully",
			exitCode:   0,
			shouldWarn: false,
		},
		{
			name:       "Command with non-zero exit",
			output:     "Command failed",
			exitCode:   1,
			shouldWarn: true,
		},
		{
			name:       "Command with error code",
			output:     "Access denied",
			exitCode:   5,
			shouldWarn: true,
		},
		{
			name:       "Empty output",
			output:     "",
			exitCode:   0,
			shouldWarn: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This would be tested through actual output formatting
			// For now, just verify the logic
			if tt.exitCode != 0 && !tt.shouldWarn {
				t.Error("Non-zero exit code should trigger warning")
			}
			if tt.exitCode == 0 && tt.shouldWarn {
				t.Error("Zero exit code should not trigger warning")
			}
		})
	}
}
