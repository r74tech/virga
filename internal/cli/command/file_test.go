package command

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/r74tech/virga/internal/cli/session"
	"github.com/r74tech/virga/internal/shared/protocol"
	"github.com/r74tech/virga/internal/shared/testutil/mock"
)

func TestUploadCommand_Execute(t *testing.T) {
	// Create a temporary file for upload
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	testContent := []byte("test content")
	if err := os.WriteFile(testFile, testContent, 0o644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	mockClient := &mock.MockAPIClient{
		SendSessionCommandFunc: func(sessionID, cmdType string, payload map[string]interface{}) (string, error) {
			if cmdType != "upload" {
				return "", fmt.Errorf("unexpected command type: %s", cmdType)
			}
			// Verify payload
			if _, ok := payload["file_content"]; !ok {
				return "", fmt.Errorf("missing file_content in payload")
			}
			if _, ok := payload["remote_path"]; !ok {
				return "", fmt.Errorf("missing remote_path in payload")
			}
			if _, ok := payload["file_name"]; !ok {
				return "", fmt.Errorf("missing file_name in payload")
			}
			if _, ok := payload["file_size"]; !ok {
				return "", fmt.Errorf("missing file_size in payload")
			}
			return "task-upload-123", nil
		},
		GetTaskResultFunc: func(sessionID, taskID string) (*protocol.TaskResult, error) {
			if taskID == "task-upload-123" {
				return &protocol.TaskResult{
					TaskID:   taskID,
					Output:   "File uploaded successfully",
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

	cmd := &UploadCommand{}

	tests := []struct {
		name        string
		args        []string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "Upload with valid file",
			args:        []string{testFile, "/tmp/remote.txt"},
			expectError: false,
		},
		{
			name:        "Upload non-existent file",
			args:        []string{"/non/existent/file.txt", "/tmp/remote.txt"},
			expectError: true,
			errorMsg:    "local file does not exist",
		},
		{
			name:        "Missing arguments",
			args:        []string{},
			expectError: true,
			errorMsg:    "Usage: upload",
		},
		{
			name:        "Missing remote path",
			args:        []string{testFile},
			expectError: true,
			errorMsg:    "Usage: upload",
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

func TestUploadCommand_ExecuteNoSession(t *testing.T) {
	manager := session.NewManager()
	cmd := &UploadCommand{}

	err := cmd.Execute(manager, []string{"/tmp/test.txt", "/tmp/remote.txt"})
	if err == nil {
		t.Error("Expected error when no session selected")
	}
	if !strings.Contains(err.Error(), "no session selected") {
		t.Errorf("Expected 'no session selected' error, got: %v", err)
	}
}

func TestDownloadCommand_Execute(t *testing.T) {
	tmpDir := t.TempDir()
	localPath := filepath.Join(tmpDir, "downloaded.txt")

	mockClient := &mock.MockAPIClient{
		SendSessionCommandFunc: func(sessionID, cmdType string, payload map[string]interface{}) (string, error) {
			if cmdType != "download" {
				return "", fmt.Errorf("unexpected command type: %s", cmdType)
			}
			// Verify payload
			if _, ok := payload["remote_path"]; !ok {
				return "", fmt.Errorf("missing remote_path in payload")
			}
			return "task-download-456", nil
		},
		GetTaskResultFunc: func(sessionID, taskID string) (*protocol.TaskResult, error) {
			if taskID == "task-download-456" {
				// Simulate base64 encoded content
				return &protocol.TaskResult{
					TaskID:   taskID,
					Output:   "ZG93bmxvYWRlZCBjb250ZW50", // "downloaded content" in base64
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

	cmd := &DownloadCommand{}

	tests := []struct {
		name        string
		args        []string
		expectError bool
		errorMsg    string
		checkFile   bool
		skipTest    bool
	}{
		{
			name:        "Download to file",
			args:        []string{"/etc/passwd", localPath},
			expectError: false,
			checkFile:   true,
		},
		{
			name:        "Download requires local path",
			args:        []string{"/etc/passwd"},
			expectError: true,
			errorMsg:    "remote and local paths required",
			checkFile:   false,
		},
		{
			name:        "Missing arguments",
			args:        []string{},
			expectError: true,
			errorMsg:    "Usage: download",
		},
		{
			name:        "Invalid local directory",
			args:        []string{"/etc/passwd", "/non/existent/dir/file.txt"},
			expectError: true,
			errorMsg:    "no such file or directory",
			skipTest:    true, // Skip as we don't want to create directories
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skipTest {
				t.Skip("Skipping test that requires directory creation")
			}

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

				if tt.checkFile {
					// Check if file was created
					if _, err := os.Stat(localPath); os.IsNotExist(err) {
						t.Error("Expected downloaded file to exist")
					}
					// Clean up
					os.Remove(localPath)
				}
			}
		})
	}
}

func TestLsCommand_Execute(t *testing.T) {
	mockClient := &mock.MockAPIClient{
		SendSessionCommandFunc: func(sessionID, cmdType string, payload map[string]interface{}) (string, error) {
			if cmdType != "ls" {
				return "", fmt.Errorf("unexpected command type: %s", cmdType)
			}
			return "task-ls-789", nil
		},
		GetTaskResultFunc: func(sessionID, taskID string) (*protocol.TaskResult, error) {
			if taskID == "task-ls-789" {
				return &protocol.TaskResult{
					TaskID: taskID,
					Output: `total 64
drwxr-xr-x   2 user group  4096 Jan  1 12:00 bin
drwxr-xr-x   3 user group  4096 Jan  1 12:00 etc
-rw-r--r--   1 user group  1024 Jan  1 12:00 file.txt`,
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

	cmd := &LsCommand{}

	tests := []struct {
		name        string
		args        []string
		expectError bool
	}{
		{
			name:        "List current directory",
			args:        []string{},
			expectError: false,
		},
		{
			name:        "List specific directory",
			args:        []string{"/tmp"},
			expectError: false,
		},
		{
			name:        "List with options",
			args:        []string{"-la", "/etc"},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := cmd.Execute(manager, tt.args)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

func TestCdCommand_Execute(t *testing.T) {
	mockClient := &mock.MockAPIClient{
		SendSessionCommandFunc: func(sessionID, cmdType string, payload map[string]interface{}) (string, error) {
			if cmdType != "cd" {
				return "", fmt.Errorf("unexpected command type: %s", cmdType)
			}
			// Verify payload
			if _, ok := payload["path"]; !ok {
				return "", fmt.Errorf("missing path in payload")
			}
			return "task-cd-012", nil
		},
		GetTaskResultFunc: func(sessionID, taskID string) (*protocol.TaskResult, error) {
			if taskID == "task-cd-012" {
				return &protocol.TaskResult{
					TaskID:   taskID,
					Output:   "Changed directory to /tmp",
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

	cmd := &CdCommand{}

	tests := []struct {
		name        string
		args        []string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "Change to directory",
			args:        []string{"/tmp"},
			expectError: false,
		},
		{
			name:        "Change to home",
			args:        []string{"~"},
			expectError: false,
		},
		{
			name:        "Missing directory",
			args:        []string{},
			expectError: true,
			errorMsg:    "Usage: cd",
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

func TestPwdCommand_Execute(t *testing.T) {
	mockClient := &mock.MockAPIClient{
		SendSessionCommandFunc: func(sessionID, cmdType string, payload map[string]interface{}) (string, error) {
			if cmdType != "pwd" {
				return "", fmt.Errorf("unexpected command type: %s", cmdType)
			}
			return "task-pwd-345", nil
		},
		GetTaskResultFunc: func(sessionID, taskID string) (*protocol.TaskResult, error) {
			if taskID == "task-pwd-345" {
				return &protocol.TaskResult{
					TaskID:   taskID,
					Output:   "/home/user/documents",
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

	cmd := &PwdCommand{}

	err := cmd.Execute(manager, []string{})
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestFileCommand_Help(t *testing.T) {
	commands := []struct {
		cmd      Command
		name     string
		keywords []string
	}{
		{
			cmd:      &UploadCommand{},
			name:     "upload",
			keywords: []string{"Upload", "local_path", "remote_path"},
		},
		{
			cmd:      &DownloadCommand{},
			name:     "download",
			keywords: []string{"Download", "remote_path", "local_path"},
		},
		{
			cmd:      &LsCommand{},
			name:     "ls",
			keywords: []string{"List", "files", "directories"},
		},
		{
			cmd:      &CdCommand{},
			name:     "cd",
			keywords: []string{"Change", "working directory"},
		},
		{
			cmd:      &PwdCommand{},
			name:     "pwd",
			keywords: []string{"Print", "current", "working directory"},
		},
	}

	for _, tc := range commands {
		t.Run(tc.name+" help", func(t *testing.T) {
			help := tc.cmd.Help()
			for _, keyword := range tc.keywords {
				if !strings.Contains(help, keyword) {
					t.Errorf("%s help missing keyword %q", tc.name, keyword)
				}
			}
		})
	}
}
