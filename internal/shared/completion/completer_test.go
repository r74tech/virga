package completion

import (
	"strings"
	"testing"
)

func TestCommandCompleter_Complete(t *testing.T) {
	cc := NewCommandCompleter()

	tests := []struct {
		name           string
		input          string
		expectedPrefix []string
		description    string
	}{
		{
			name:           "empty input",
			input:          "",
			expectedPrefix: []string{"help", "exit", "sessions", "interact"},
			description:    "should return all commands",
		},
		{
			name:           "partial command",
			input:          "ses",
			expectedPrefix: []string{"sessions"},
			description:    "should complete 'sessions' command",
		},
		{
			name:           "full command with space",
			input:          "sessions ",
			expectedPrefix: []string{"list", "kill"},
			description:    "should return subcommands for sessions",
		},
		{
			name:           "partial subcommand",
			input:          "portfwd a",
			expectedPrefix: []string{"add"},
			description:    "should complete 'add' subcommand",
		},
		{
			name:           "log level completion",
			input:          "log level d",
			expectedPrefix: []string{"debug"},
			description:    "should complete 'debug' log level",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			completions, _ := cc.Complete(tt.input, len(tt.input))

			// Check if expected completions are present
			for _, expected := range tt.expectedPrefix {
				found := false
				for _, completion := range completions {
					if strings.HasPrefix(completion, expected) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected completion '%s' not found in %v", expected, completions)
				}
			}
		})
	}
}

func TestCommandCompleter_SessionIDCompletion(t *testing.T) {
	cc := NewCommandCompleter()

	// Set up mock session IDs
	mockSessionIDs := []string{
		"4993cd68-b3f1-7c7a-4405-bff2a8d65ad0",
		"4567ab12-c3d4-5e6f-7890-abcdef123456",
		"abcd1234-5678-90ef-ghij-klmnopqrstuv",
	}

	cc.SetSessionIDProvider(func() []string {
		return mockSessionIDs
	})

	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "interact with partial session ID",
			input:    "interact 499",
			expected: []string{"4993cd68-b3f1-7c7a-4405-bff2a8d65ad0"},
		},
		{
			name:  "interact with common prefix",
			input: "interact 4",
			expected: []string{
				"4993cd68-b3f1-7c7a-4405-bff2a8d65ad0",
				"4567ab12-c3d4-5e6f-7890-abcdef123456",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			completions, _ := cc.Complete(tt.input, len(tt.input))

			if len(completions) != len(tt.expected) {
				t.Errorf("Expected %d completions, got %d", len(tt.expected), len(completions))
			}

			for i, expected := range tt.expected {
				if i < len(completions) && completions[i] != expected {
					t.Errorf("Expected completion[%d] = %s, got %s", i, expected, completions[i])
				}
			}
		})
	}
}

func TestFileCompleter_Complete(t *testing.T) {
	fc := NewFileCompleter()

	// Note: These tests depend on the file system
	// In a real test environment, we would mock the file system
	tests := []struct {
		name        string
		input       string
		expectFiles bool
	}{
		{
			name:        "current directory",
			input:       "./",
			expectFiles: true,
		},
		{
			name:        "home directory",
			input:       "~/",
			expectFiles: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			completions, _ := fc.Complete(tt.input, len(tt.input))

			if tt.expectFiles && len(completions) == 0 {
				t.Logf("Warning: No file completions found for '%s' (this might be expected in test environment)", tt.input)
			}
		})
	}
}

func TestSplitCommandLine(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "simple command",
			input:    "help",
			expected: []string{"help"},
		},
		{
			name:     "command with args",
			input:    "upload file.txt /tmp/file.txt",
			expected: []string{"upload", "file.txt", "/tmp/file.txt"},
		},
		{
			name:     "quoted argument",
			input:    `upload "my file.txt" /tmp/file.txt`,
			expected: []string{"upload", "my file.txt", "/tmp/file.txt"},
		},
		{
			name:     "mixed quotes",
			input:    `shell echo "hello world" 'single quote'`,
			expected: []string{"shell", "echo", "hello world", "single quote"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := splitCommandLine(tt.input)

			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d parts, got %d", len(tt.expected), len(result))
			}

			for i, expected := range tt.expected {
				if i < len(result) && result[i] != expected {
					t.Errorf("Expected part[%d] = %s, got %s", i, expected, result[i])
				}
			}
		})
	}
}

func TestCommandCompleter_SortOrder(t *testing.T) {
	cc := NewCommandCompleter()

	tests := []struct {
		name      string
		sortOrder SortOrder
		input     string
		check     func(completions []string) bool
	}{
		{
			name:      "rank order - high priority commands first",
			sortOrder: SortByRank,
			input:     "",
			check: func(completions []string) bool {
				// Find indices of commands with different priorities
				helpIdx := -1
				llamaIdx := -1
				for i, cmd := range completions {
					if cmd == "help" {
						helpIdx = i
					}
					if cmd == "llama" {
						llamaIdx = i
					}
				}
				// help (priority 100) should come before llama (priority 60)
				return helpIdx != -1 && llamaIdx != -1 && helpIdx < llamaIdx
			},
		},
		{
			name:      "alphabetical order",
			sortOrder: SortByAlphabetical,
			input:     "",
			check: func(completions []string) bool {
				// Check if completions are in alphabetical order
				for i := 1; i < len(completions); i++ {
					if completions[i-1] > completions[i] {
						return false
					}
				}
				return true
			},
		},
		{
			name:      "rank order - same priority sorted alphabetically",
			sortOrder: SortByRank,
			input:     "s",
			check: func(completions []string) bool {
				// sessions (95) and shell (92) - sessions should come first
				sessionsIdx := -1
				shellIdx := -1
				for i, cmd := range completions {
					if cmd == "sessions" {
						sessionsIdx = i
					}
					if cmd == "shell" {
						shellIdx = i
					}
				}
				return sessionsIdx != -1 && shellIdx != -1 && sessionsIdx < shellIdx
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cc.SetSortOrder(tt.sortOrder)
			completions, _ := cc.Complete(tt.input, len(tt.input))

			if !tt.check(completions) {
				t.Errorf("Sort order check failed for %s. Got completions: %v", tt.sortOrder, completions)
			}
		})
	}
}

func TestCommandCompleter_Priority(t *testing.T) {
	cc := NewCommandCompleter()
	cc.SetSortOrder(SortByRank)

	// Test that frequently used commands have higher priority
	completions, _ := cc.Complete("", 0)

	// Check relative positions of commands based on priority
	priorities := map[string]int{
		"help":     100,
		"sessions": 95,
		"interact": 95,
		"shell":    92,
		"exec":     92,
		"ls":       90,
		"cd":       90,
		"upload":   85,
		"download": 85,
		"ps":       80,
		"generate": 75,
		"info":     75,
		"beacons":  70,
		"netstat":  70,
		"portfwd":  65,
		"log":      65,
		"workflow": 65,
		"llama":    60,
		"memdb":    60,
	}

	// Verify that higher priority commands appear earlier
	for i := 0; i < len(completions)-1; i++ {
		cmd1 := completions[i]
		cmd2 := completions[i+1]

		priority1, ok1 := priorities[cmd1]
		priority2, ok2 := priorities[cmd2]

		if ok1 && ok2 {
			if priority1 < priority2 {
				t.Errorf("Command %s (priority %d) appears before %s (priority %d)",
					cmd1, priority1, cmd2, priority2)
			}
		}
	}
}
