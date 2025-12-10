package command

import (
	"runtime"
	"strings"
	"testing"

	"github.com/r74tech/virga/internal/cli/session"
)

func TestValidateLocalPath(t *testing.T) {
	tests := []struct {
		name      string
		path      string
		wantError bool
		errorMsg  string
	}{
		{
			name: "valid path",
			path: "/tmp/file.txt",
		},
		{
			name:      "empty path",
			path:      "",
			wantError: true,
			errorMsg:  "path cannot be empty",
		},
		{
			name:      "path with null character",
			path:      "/tmp/file\x00.txt",
			wantError: true,
			errorMsg:  "null character",
		},
		{
			name:      "path traversal attempt",
			path:      "../../../etc/passwd",
			wantError: true,
			errorMsg:  "directory traversal",
		},
		{
			name:      "path with double dots in middle",
			path:      "/tmp/../etc/passwd",
			wantError: true,
			errorMsg:  "directory traversal",
		},
	}

	// Add Windows-specific tests if on Windows
	if runtime.GOOS == "windows" {
		tests = append(tests, []struct {
			name      string
			path      string
			wantError bool
			errorMsg  string
		}{
			{
				name:      "Windows reserved device CON",
				path:      "CON",
				wantError: true,
				errorMsg:  "reserved device name",
			},
			{
				name:      "Windows reserved device PRN.txt",
				path:      "PRN.txt",
				wantError: true,
				errorMsg:  "reserved device name",
			},
			{
				name:      "Windows invalid character",
				path:      "file<name>.txt",
				wantError: true,
				errorMsg:  "invalid character",
			},
		}...)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateLocalPath(tt.path)
			if (err != nil) != tt.wantError {
				t.Errorf("validateLocalPath() error = %v, wantError %v", err, tt.wantError)
				return
			}
			if err != nil && tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
				t.Errorf("validateLocalPath() error = %v, want error containing %v", err, tt.errorMsg)
			}
		})
	}
}

func TestSanitizeCommand(t *testing.T) {
	tests := []struct {
		name     string
		command  string
		args     []string
		wantCmd  string
		wantArgs []string
	}{
		{
			name:     "simple command",
			command:  "ls",
			args:     []string{"-la", "/tmp"},
			wantCmd:  "ls",
			wantArgs: []string{"-la", "/tmp"},
		},
		{
			name:     "command with semicolon",
			command:  "ls;rm",
			args:     []string{"-rf", "/"},
			wantCmd:  "lsrm",
			wantArgs: []string{"-rf", "/"},
		},
		{
			name:     "command with pipe",
			command:  "cat|grep",
			args:     []string{"password"},
			wantCmd:  "catgrep",
			wantArgs: []string{"password"},
		},
		{
			name:     "args with dangerous characters",
			command:  "echo",
			args:     []string{"test;rm", "-rf", "$HOME"},
			wantCmd:  "echo",
			wantArgs: []string{"testrm", "-rf", "HOME"},
		},
		{
			name:     "command with backticks",
			command:  "echo `id`",
			args:     []string{},
			wantCmd:  "echo id",
			wantArgs: []string{},
		},
		{
			name:     "all dangerous characters",
			command:  "test;&|`$(){}[]<>",
			args:     []string{";&|`$(){}[]<>"},
			wantCmd:  "test",
			wantArgs: []string{""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotCmd, gotArgs := sanitizeCommand(tt.command, tt.args)
			if gotCmd != tt.wantCmd {
				t.Errorf("sanitizeCommand() command = %v, want %v", gotCmd, tt.wantCmd)
			}
			if len(gotArgs) != len(tt.wantArgs) {
				t.Errorf("sanitizeCommand() args length = %v, want %v", len(gotArgs), len(tt.wantArgs))
				return
			}
			for i := range gotArgs {
				if gotArgs[i] != tt.wantArgs[i] {
					t.Errorf("sanitizeCommand() args[%d] = %v, want %v", i, gotArgs[i], tt.wantArgs[i])
				}
			}
		})
	}
}

func TestValidateCommand(t *testing.T) {
	tests := []struct {
		name      string
		command   string
		wantError bool
		errorMsg  string
	}{
		{
			name:    "valid command",
			command: "ls",
		},
		{
			name:    "valid command with path",
			command: "/usr/bin/ls",
		},
		{
			name:      "empty command",
			command:   "",
			wantError: true,
			errorMsg:  "command cannot be empty",
		},
		{
			name:      "command with dangerous characters",
			command:   "ls;rm -rf /",
			wantError: true,
			errorMsg:  "dangerous characters",
		},
		{
			name:      "command with pipe",
			command:   "cat | grep",
			wantError: true,
			errorMsg:  "dangerous characters",
		},
		{
			name:      "command with redirection",
			command:   "echo test > /etc/passwd",
			wantError: true,
			errorMsg:  "dangerous characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateCommand(tt.command)
			if (err != nil) != tt.wantError {
				t.Errorf("validateCommand() error = %v, wantError %v", err, tt.wantError)
				return
			}
			if err != nil && tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
				t.Errorf("validateCommand() error = %v, want error containing %v", err, tt.errorMsg)
			}
		})
	}
}

func TestValidateRemotePath(t *testing.T) {
	tests := []struct {
		name      string
		path      string
		wantError bool
		errorMsg  string
	}{
		{
			name: "valid absolute path",
			path: "/home/user/file.txt",
		},
		{
			name: "valid relative path",
			path: "documents/file.txt",
		},
		{
			name: "valid Windows path",
			path: "C:\\Users\\file.txt",
		},
		{
			name:      "empty path",
			path:      "",
			wantError: true,
			errorMsg:  "path cannot be empty",
		},
		{
			name:      "path with null character",
			path:      "/tmp/file\x00.txt",
			wantError: true,
			errorMsg:  "null character",
		},
		{
			name:      "excessive path length",
			path:      strings.Repeat("a", 4097),
			wantError: true,
			errorMsg:  "path too long",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateRemotePath(tt.path)
			if (err != nil) != tt.wantError {
				t.Errorf("validateRemotePath() error = %v, wantError %v", err, tt.wantError)
				return
			}
			if err != nil && tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
				t.Errorf("validateRemotePath() error = %v, want error containing %v", err, tt.errorMsg)
			}
		})
	}
}

func TestValidatePort(t *testing.T) {
	tests := []struct {
		name      string
		port      int
		wantError bool
	}{
		{
			name: "valid port 80",
			port: 80,
		},
		{
			name: "valid port 443",
			port: 443,
		},
		{
			name: "valid port 8080",
			port: 8080,
		},
		{
			name: "minimum valid port",
			port: 1,
		},
		{
			name: "maximum valid port",
			port: 65535,
		},
		{
			name:      "port 0",
			port:      0,
			wantError: true,
		},
		{
			name:      "negative port",
			port:      -1,
			wantError: true,
		},
		{
			name:      "port too high",
			port:      65536,
			wantError: true,
		},
		{
			name:      "very high port",
			port:      100000,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePort(tt.port)
			if (err != nil) != tt.wantError {
				t.Errorf("validatePort() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestKillswitchCommand_Help(t *testing.T) {
	cmd := &KillswitchCommand{}
	help := cmd.Help()

	// Help text should contain key information
	requiredStrings := []string{
		"killswitch",
		"shutdown",
		"remove",
		"--mode",
		"--message",
		"confirmation",
	}

	for _, required := range requiredStrings {
		if !strings.Contains(help, required) {
			t.Errorf("Help text missing required string: %s", required)
		}
	}
}

func TestKillswitchCommand_Execute_NoSession(t *testing.T) {
	cmd := &KillswitchCommand{}
	manager := session.NewManager()

	err := cmd.Execute(manager, []string{})
	if err == nil {
		t.Error("Expected error when no session is selected")
	}
	if !strings.Contains(err.Error(), "no session selected") {
		t.Errorf("Expected 'no session selected' error, got: %v", err)
	}
}
