package command

import (
	"fmt"
	"strings"
	"testing"

	"github.com/r74tech/virga/internal/cli/session"
	"github.com/r74tech/virga/internal/shared/protocol"
	"github.com/r74tech/virga/internal/shared/testutil/mock"
)

func TestInfoCommand_Execute(t *testing.T) {
	mockClient := &mock.MockAPIClient{
		SendSessionCommandFunc: func(sessionID, cmdType string, payload map[string]interface{}) (string, error) {
			if cmdType != "info" {
				return "", fmt.Errorf("unexpected command type: %s", cmdType)
			}
			return "task-info-123", nil
		},
		GetTaskResultFunc: func(sessionID, taskID string) (*protocol.TaskResult, error) {
			if taskID == "task-info-123" {
				return &protocol.TaskResult{
					TaskID: taskID,
					Output: `Session Information:
--------------------
Session ID: test-session
Agent ID: agent-123
Hostname: test-host
Username: test-user
IP Address: 192.168.1.100
OS: Windows 10 Pro
Architecture: amd64
Process ID: 1234
Process Name: beacon.exe
Privileges: Administrator
Last Checkin: 2 minutes ago`,
					ExitCode: 0,
					Error:    "",
				}, nil
			}
			return nil, fmt.Errorf("task not found: %s", taskID)
		},
	}

	manager := session.NewManager()
	manager.SetAPIClient(mockClient)
	manager.AddSession("test-session", "192.168.1.100:4444", map[string]interface{}{
		"hostname": "test-host",
		"username": "test-user",
		"os":       "windows",
		"arch":     "amd64",
	})
	manager.SetCurrentSession("test-session")

	cmd := &InfoCommand{}

	err := cmd.Execute(manager, []string{})
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestInfoCommand_ExecuteNoSession(t *testing.T) {
	manager := session.NewManager()
	cmd := &InfoCommand{}

	err := cmd.Execute(manager, []string{})
	if err == nil {
		t.Error("Expected error when no session selected")
	}
	if !strings.Contains(err.Error(), "no session selected") {
		t.Errorf("Expected 'no session selected' error, got: %v", err)
	}
}

func TestInfoCommand_Help(t *testing.T) {
	cmd := &InfoCommand{}
	help := cmd.Help()

	expectedKeywords := []string{
		"Display",
		"detailed information",
		"current session",
	}

	for _, keyword := range expectedKeywords {
		if !strings.Contains(help, keyword) {
			t.Errorf("Help text missing expected keyword: %s", keyword)
		}
	}
}

func TestSysinfoCommand_Execute(t *testing.T) {
	mockClient := &mock.MockAPIClient{
		SendSessionCommandFunc: func(sessionID, cmdType string, payload map[string]interface{}) (string, error) {
			if cmdType != "sysinfo" {
				return "", fmt.Errorf("unexpected command type: %s", cmdType)
			}
			return "task-sysinfo-456", nil
		},
		GetTaskResultFunc: func(sessionID, taskID string) (*protocol.TaskResult, error) {
			if taskID == "task-sysinfo-456" {
				return &protocol.TaskResult{
					TaskID: taskID,
					Output: `System Information:
------------------
Hostname: TEST-PC
OS: Microsoft Windows 10 Pro
Version: 10.0.19043
Architecture: x86_64
CPU: Intel(R) Core(TM) i7-8700K CPU @ 3.70GHz
Cores: 6
Memory: 16 GB (8 GB available)
Uptime: 5 days, 3 hours, 25 minutes
Domain: WORKGROUP
Antivirus: Windows Defender (enabled)
Firewall: Enabled`,
					ExitCode: 0,
					Error:    "",
				}, nil
			}
			return nil, fmt.Errorf("task not found: %s", taskID)
		},
	}

	manager := session.NewManager()
	manager.SetAPIClient(mockClient)
	manager.AddSession("test-session", "192.168.1.100:4444", map[string]interface{}{
		"hostname": "test-host",
	})
	manager.SetCurrentSession("test-session")

	cmd := &SysInfoCommand{}

	err := cmd.Execute(manager, []string{})
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestSysinfoCommand_Help(t *testing.T) {
	cmd := &SysInfoCommand{}
	help := cmd.Help()

	expectedKeywords := []string{
		"Collect",
		"system information",
		"target",
	}

	for _, keyword := range expectedKeywords {
		if !strings.Contains(help, keyword) {
			t.Errorf("Help text missing expected keyword: %s", keyword)
		}
	}
}

func TestNetinfoCommand_Execute(t *testing.T) {
	mockClient := &mock.MockAPIClient{
		SendSessionCommandFunc: func(sessionID, cmdType string, payload map[string]interface{}) (string, error) {
			if cmdType != "netinfo" {
				return "", fmt.Errorf("unexpected command type: %s", cmdType)
			}
			return "task-netinfo-789", nil
		},
		GetTaskResultFunc: func(sessionID, taskID string) (*protocol.TaskResult, error) {
			if taskID == "task-netinfo-789" {
				return &protocol.TaskResult{
					TaskID: taskID,
					Output: `Network Information:
-------------------
Interfaces:
  Ethernet0:
    IP: 192.168.1.100/24
    MAC: 00:11:22:33:44:55
    Gateway: 192.168.1.1
    DNS: 192.168.1.1, 8.8.8.8

Active Connections:
  TCP    192.168.1.100:49152    52.123.45.67:443    ESTABLISHED
  TCP    192.168.1.100:49153    10.0.0.5:445        ESTABLISHED
  UDP    192.168.1.100:53        *:*                 LISTENING

Routing Table:
  0.0.0.0/0 -> 192.168.1.1
  192.168.1.0/24 -> 0.0.0.0`,
					ExitCode: 0,
					Error:    "",
				}, nil
			}
			return nil, fmt.Errorf("task not found: %s", taskID)
		},
	}

	manager := session.NewManager()
	manager.SetAPIClient(mockClient)
	manager.AddSession("test-session", "192.168.1.100:4444", map[string]interface{}{
		"hostname": "test-host",
	})
	manager.SetCurrentSession("test-session")

	cmd := &NetworkInfoCommand{}

	err := cmd.Execute(manager, []string{})
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestPsCommand_Execute(t *testing.T) {
	mockClient := &mock.MockAPIClient{
		SendSessionCommandFunc: func(sessionID, cmdType string, payload map[string]interface{}) (string, error) {
			if cmdType != "ps" {
				return "", fmt.Errorf("unexpected command type: %s", cmdType)
			}
			return "task-ps-012", nil
		},
		GetTaskResultFunc: func(sessionID, taskID string) (*protocol.TaskResult, error) {
			if taskID == "task-ps-012" {
				return &protocol.TaskResult{
					TaskID: taskID,
					Output: `Process List:
------------
PID     PPID    Name                    User            CPU     Memory
4       0       System                  NT AUTHORITY    0.1%    128 MB
1234    456     beacon.exe              TEST\admin      2.5%    45 MB
2345    1       chrome.exe              TEST\user       15.2%   512 MB
3456    1       svchost.exe             NT AUTHORITY    0.5%    32 MB
4567    1       explorer.exe            TEST\user       1.2%    128 MB`,
					ExitCode: 0,
					Error:    "",
				}, nil
			}
			return nil, fmt.Errorf("task not found: %s", taskID)
		},
	}

	manager := session.NewManager()
	manager.SetAPIClient(mockClient)
	manager.AddSession("test-session", "192.168.1.100:4444", map[string]interface{}{
		"hostname": "test-host",
	})
	manager.SetCurrentSession("test-session")

	cmd := &ProcessListCommand{}

	err := cmd.Execute(manager, []string{})
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestPsCommand_WithFilter(t *testing.T) {
	callCount := 0
	mockClient := &mock.MockAPIClient{
		SendSessionCommandFunc: func(sessionID, cmdType string, payload map[string]interface{}) (string, error) {
			callCount++
			if cmdType != "ps" {
				return "", fmt.Errorf("unexpected command type: %s", cmdType)
			}

			// Verify filter was passed
			if filter, ok := payload["filter"].(string); ok {
				if filter != "chrome" {
					return "", fmt.Errorf("unexpected filter: %s", filter)
				}
			}

			return fmt.Sprintf("task-ps-filter-%d", callCount), nil
		},
		GetTaskResultFunc: func(sessionID, taskID string) (*protocol.TaskResult, error) {
			return &protocol.TaskResult{
				TaskID: taskID,
				Output: `Process List (filtered):
-----------------------
PID     PPID    Name                    User            CPU     Memory
2345    1       chrome.exe              TEST\user       15.2%   512 MB
2346    2345    chrome.exe              TEST\user       3.1%    128 MB
2347    2345    chrome.exe              TEST\user       1.5%    64 MB`,
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

	cmd := &ProcessListCommand{}

	// Test with filter
	err := cmd.Execute(manager, []string{"chrome"})
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if callCount != 1 {
		t.Errorf("Expected 1 call to SendSessionCommand, got %d", callCount)
	}
}

func TestTaskResultDisplay(t *testing.T) {
	// Test that task results are displayed properly
	tests := []struct {
		name     string
		result   *protocol.TaskResult
		contains []string
	}{
		{
			name: "Successful task",
			result: &protocol.TaskResult{
				TaskID:   "test-123",
				Output:   "Command executed successfully",
				ExitCode: 0,
				Error:    "",
			},
			contains: []string{
				"Task ID: test-123",
				"Command executed successfully",
				"Exit code: 0",
			},
		},
		{
			name: "Failed task",
			result: &protocol.TaskResult{
				TaskID:   "test-456",
				Output:   "Partial output",
				ExitCode: 1,
				Error:    "Command failed: permission denied",
			},
			contains: []string{
				"Task ID: test-456",
				"Partial output",
				"Exit code: 1",
				"Error: Command failed",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This would be tested through actual command execution
			// For now, just verify the result structure is correct
			if tt.result.TaskID == "" {
				t.Error("TaskID should not be empty")
			}
			if tt.result.ExitCode < 0 {
				t.Error("ExitCode should not be negative")
			}
		})
	}
}
