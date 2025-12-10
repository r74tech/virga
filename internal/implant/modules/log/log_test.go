package log

import (
	"strings"
	"testing"

	"github.com/r74tech/virga/internal/implant/logger"
)

func TestLogModule_Name(t *testing.T) {
	module := &LogModule{}
	if name := module.Name(); name != "log" {
		t.Errorf("Expected name 'log', got '%s'", name)
	}
}

func TestLogModule_Execute_Help(t *testing.T) {
	module := &LogModule{}

	tests := []struct {
		name string
		args []string
	}{
		{
			name: "No arguments shows help",
			args: []string{},
		},
		{
			name: "Explicit help command",
			args: []string{"help"},
		},
		{
			name: "Help with extra arguments",
			args: []string{"help", "ignored"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, exitCode, err := module.Execute(tt.args)

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if exitCode != 0 {
				t.Errorf("Expected exit code 0, got %d", exitCode)
			}

			// Check help content
			expectedCommands := []string{
				"log level",
				"log disable",
				"log enable",
				"log status",
				"log help",
			}

			for _, cmd := range expectedCommands {
				if !strings.Contains(output, cmd) {
					t.Errorf("Help output missing command: %s", cmd)
				}
			}
		})
	}
}

func TestLogModule_Execute_LoggerDisabled(t *testing.T) {
	module := &LogModule{}

	// Note: We can't easily disable the singleton logger for testing
	// This test demonstrates what should happen if logger is disabled
	// In real scenario, this would require build flags

	log := logger.Get()
	if !log.IsEnabled() {
		// If logger is actually disabled (rare in tests)
		args := []string{"level"}
		output, exitCode, err := module.Execute(args)

		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if exitCode != 1 {
			t.Errorf("Expected exit code 1 for disabled logger, got %d", exitCode)
		}
		if output != "Logging is disabled" {
			t.Errorf("Expected 'Logging is disabled', got '%s'", output)
		}
	} else {
		t.Skip("Logger is enabled, skipping disabled logger test")
	}
}

func TestLogModule_Execute_Level(t *testing.T) {
	module := &LogModule{}
	log := logger.Get()

	if !log.IsEnabled() {
		t.Skip("Logger is disabled, skipping level tests")
	}

	// Save original level to restore later
	originalLevel := log.GetLevel()
	defer log.SetLevel(originalLevel)

	tests := []struct {
		name        string
		args        []string
		expectError bool
		expectCode  int
		verifyFunc  func(string) error
	}{
		{
			name:        "Get current level",
			args:        []string{"level"},
			expectError: false,
			expectCode:  0,
			verifyFunc: func(output string) error {
				if !strings.Contains(output, "Current log level:") {
					t.Error("Output should contain current level")
				}
				return nil
			},
		},
		{
			name:        "Set level to debug",
			args:        []string{"level", "debug"},
			expectError: false,
			expectCode:  0,
			verifyFunc: func(output string) error {
				if !strings.Contains(output, "Log level set to: debug") {
					t.Error("Output should confirm level change")
				}
				if log.GetLevel() != logger.DEBUG {
					t.Error("Level should be DEBUG")
				}
				return nil
			},
		},
		{
			name:        "Set level to info",
			args:        []string{"level", "info"},
			expectError: false,
			expectCode:  0,
			verifyFunc: func(output string) error {
				if log.GetLevel() != logger.INFO {
					t.Error("Level should be INFO")
				}
				return nil
			},
		},
		{
			name:        "Set level to warn",
			args:        []string{"level", "warn"},
			expectError: false,
			expectCode:  0,
			verifyFunc: func(output string) error {
				if log.GetLevel() != logger.WARN {
					t.Error("Level should be WARN")
				}
				return nil
			},
		},
		{
			name:        "Set level to error",
			args:        []string{"level", "error"},
			expectError: false,
			expectCode:  0,
			verifyFunc: func(output string) error {
				if log.GetLevel() != logger.ERROR {
					t.Error("Level should be ERROR")
				}
				return nil
			},
		},
		{
			name:        "Set level to off",
			args:        []string{"level", "off"},
			expectError: false,
			expectCode:  0,
			verifyFunc: func(output string) error {
				// OFF level is ERROR + 1
				if log.GetLevel() <= logger.ERROR {
					t.Error("Level should be higher than ERROR (OFF)")
				}
				return nil
			},
		},
		{
			name:        "Set level with uppercase",
			args:        []string{"level", "DEBUG"},
			expectError: false,
			expectCode:  0,
			verifyFunc: func(output string) error {
				if !strings.Contains(output, "Log level set to: DEBUG") {
					t.Error("Should accept uppercase level")
				}
				return nil
			},
		},
		{
			name:        "Set level with mixed case",
			args:        []string{"level", "InFo"},
			expectError: false,
			expectCode:  0,
			verifyFunc: func(output string) error {
				if log.GetLevel() != logger.INFO {
					t.Error("Should accept mixed case level")
				}
				return nil
			},
		},
		{
			name:        "Set invalid level defaults to INFO",
			args:        []string{"level", "invalid"},
			expectError: false,
			expectCode:  0,
			verifyFunc: func(output string) error {
				// The logger defaults invalid levels to INFO
				if log.GetLevel() != logger.INFO {
					t.Error("Invalid level should default to INFO")
				}
				return nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, exitCode, err := module.Execute(tt.args)

			if tt.expectError {
				if err == nil && exitCode == 0 {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}

			if exitCode != tt.expectCode {
				t.Errorf("Expected exit code %d, got %d", tt.expectCode, exitCode)
			}

			if tt.verifyFunc != nil && !tt.expectError {
				if err := tt.verifyFunc(output); err != nil {
					t.Errorf("Verification failed: %v", err)
				}
			}
		})
	}
}

func TestLogModule_Execute_DisableEnable(t *testing.T) {
	module := &LogModule{}
	log := logger.Get()

	if !log.IsEnabled() {
		t.Skip("Logger is disabled, skipping disable/enable tests")
	}

	// Save original level
	originalLevel := log.GetLevel()
	defer log.SetLevel(originalLevel)

	tests := []struct {
		name       string
		args       []string
		expectCode int
		verifyFunc func(string) error
	}{
		{
			name:       "Disable logging",
			args:       []string{"disable"},
			expectCode: 0,
			verifyFunc: func(output string) error {
				if !strings.Contains(output, "Logging disabled") {
					t.Error("Output should confirm logging disabled")
				}
				// Check that level is set to OFF (higher than ERROR)
				if log.GetLevel() <= logger.ERROR {
					t.Error("Level should be OFF (higher than ERROR)")
				}
				return nil
			},
		},
		{
			name:       "Enable logging",
			args:       []string{"enable"},
			expectCode: 0,
			verifyFunc: func(output string) error {
				if !strings.Contains(output, "Logging enabled") {
					t.Error("Output should confirm logging enabled")
				}
				// Check that level is set to INFO
				if log.GetLevel() != logger.INFO {
					t.Error("Level should be INFO after enabling")
				}
				return nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, exitCode, err := module.Execute(tt.args)

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if exitCode != tt.expectCode {
				t.Errorf("Expected exit code %d, got %d", tt.expectCode, exitCode)
			}

			if tt.verifyFunc != nil {
				if err := tt.verifyFunc(output); err != nil {
					t.Errorf("Verification failed: %v", err)
				}
			}
		})
	}
}

func TestLogModule_Execute_Status(t *testing.T) {
	module := &LogModule{}
	log := logger.Get()

	if !log.IsEnabled() {
		t.Skip("Logger is disabled, skipping status test")
	}

	output, exitCode, err := module.Execute([]string{"status"})

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if exitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", exitCode)
	}

	// Check status output contains expected information
	expectedFields := []string{
		"Logging Status:",
		"Enabled:",
		"Level:",
		"Path:",
	}

	for _, field := range expectedFields {
		if !strings.Contains(output, field) {
			t.Errorf("Status output missing field: %s", field)
		}
	}

	// Verify actual values
	if !strings.Contains(output, "Enabled: true") {
		t.Error("Status should show logger is enabled")
	}

	// Check that path is not empty
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "Path:") {
			pathPart := strings.TrimPrefix(strings.TrimSpace(line), "Path:")
			if strings.TrimSpace(pathPart) == "" {
				t.Error("Path should not be empty")
			}
		}
	}
}

func TestLogModule_Execute_UnknownCommand(t *testing.T) {
	module := &LogModule{}
	log := logger.Get()

	// Skip test if logger is disabled
	if !log.IsEnabled() {
		t.Skip("Logger is disabled, skipping unknown command test")
	}

	tests := []struct {
		name string
		args []string
	}{
		{
			name: "Unknown command",
			args: []string{"unknown"},
		},
		{
			name: "Unknown command with arguments",
			args: []string{"invalid", "arg1", "arg2"},
		},
		{
			name: "Misspelled command",
			args: []string{"levl"}, // typo of "level"
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, exitCode, err := module.Execute(tt.args)

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if exitCode != 1 {
				t.Errorf("Expected exit code 1 for unknown command, got %d", exitCode)
			}

			// Should show unknown command message and help
			if !strings.Contains(output, "Unknown log command:") {
				t.Error("Output should indicate unknown command")
			}
			if !strings.Contains(output, tt.args[0]) {
				t.Error("Output should include the unknown command")
			}
			// Should include help text
			if !strings.Contains(output, "Log Module Commands:") {
				t.Error("Output should include help text")
			}
		})
	}
}

func TestLogModule_Execute_ExtraArguments(t *testing.T) {
	module := &LogModule{}
	log := logger.Get()

	if !log.IsEnabled() {
		t.Skip("Logger is disabled, skipping extra arguments test")
	}

	tests := []struct {
		name string
		args []string
	}{
		{
			name: "Status with extra arguments",
			args: []string{"status", "ignored", "arguments"},
		},
		{
			name: "Disable with extra arguments",
			args: []string{"disable", "extra"},
		},
		{
			name: "Enable with extra arguments",
			args: []string{"enable", "arg1", "arg2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, exitCode, err := module.Execute(tt.args)

			// These commands should still work, ignoring extra arguments
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if exitCode != 0 {
				t.Errorf("Expected exit code 0, got %d", exitCode)
			}

			// Verify command executed correctly
			switch tt.args[0] {
			case "status":
				if !strings.Contains(output, "Logging Status:") {
					t.Error("Status command should execute despite extra args")
				}
			case "disable":
				if !strings.Contains(output, "Logging disabled") {
					t.Error("Disable command should execute despite extra args")
				}
			case "enable":
				if !strings.Contains(output, "Logging enabled") {
					t.Error("Enable command should execute despite extra args")
				}
			}
		})
	}
}

func TestLogModule_ShowHelp(t *testing.T) {
	module := &LogModule{}
	help := module.showHelp()

	// Check help structure
	if !strings.HasPrefix(help, "Log Module Commands:") {
		t.Error("Help should start with 'Log Module Commands:'")
	}

	// Check all commands are documented
	commands := []struct {
		cmd  string
		desc string
	}{
		{"log level", "Get or set log level"},
		{"log disable", "Disable logging"},
		{"log enable", "Enable logging"},
		{"log status", "Show logging status"},
		{"log help", "Show this help"},
	}

	for _, cmd := range commands {
		if !strings.Contains(help, cmd.cmd) {
			t.Errorf("Help missing command: %s", cmd.cmd)
		}
		if !strings.Contains(help, cmd.desc) {
			t.Errorf("Help missing description for: %s", cmd.cmd)
		}
	}

	// Check formatting
	lines := strings.Split(help, "\n")
	for i, line := range lines {
		if i == 0 {
			continue // Skip header
		}
		if strings.TrimSpace(line) != "" && !strings.HasPrefix(line, "  ") {
			t.Errorf("Help line %d should be indented: %s", i, line)
		}
	}
}

func TestLogModule_ConcurrentAccess(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrent access test in short mode")
	}

	module := &LogModule{}
	log := logger.Get()

	if !log.IsEnabled() {
		t.Skip("Logger is disabled, skipping concurrent test")
	}

	// Save original level
	originalLevel := log.GetLevel()
	defer log.SetLevel(originalLevel)

	// Run multiple operations concurrently
	done := make(chan bool, 4)

	go func() {
		for i := 0; i < 10; i++ {
			module.Execute([]string{"status"})
		}
		done <- true
	}()

	go func() {
		levels := []string{"debug", "info", "warn", "error"}
		for i := 0; i < 10; i++ {
			module.Execute([]string{"level", levels[i%len(levels)]})
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 5; i++ {
			module.Execute([]string{"disable"})
			module.Execute([]string{"enable"})
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 10; i++ {
			module.Execute([]string{"level"}) // Get current level
		}
		done <- true
	}()

	// Wait for all goroutines
	for i := 0; i < 4; i++ {
		<-done
	}

	// Verify logger is still functional
	output, exitCode, err := module.Execute([]string{"status"})
	if err != nil {
		t.Errorf("Logger failed after concurrent access: %v", err)
	}
	if exitCode != 0 {
		t.Error("Logger returned error after concurrent access")
	}
	if !strings.Contains(output, "Logging Status:") {
		t.Error("Logger not returning proper status after concurrent access")
	}
}

func BenchmarkLogModule_Execute(b *testing.B) {
	module := &LogModule{}

	b.Run("Help", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			module.Execute([]string{})
		}
	})

	b.Run("Status", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			module.Execute([]string{"status"})
		}
	})

	b.Run("GetLevel", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			module.Execute([]string{"level"})
		}
	})

	b.Run("SetLevel", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			module.Execute([]string{"level", "info"})
		}
	})
}

func BenchmarkLogModule_ShowHelp(b *testing.B) {
	module := &LogModule{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = module.showHelp()
	}
}

// TestLogModule_LoggerInteraction verifies correct interaction with logger
func TestLogModule_LoggerInteraction(t *testing.T) {
	module := &LogModule{}
	log := logger.Get()

	if !log.IsEnabled() {
		// Test that all commands fail gracefully when logger is disabled
		commands := [][]string{
			{"level"},
			{"level", "debug"},
			{"disable"},
			{"enable"},
			{"status"},
		}

		for _, cmd := range commands {
			output, exitCode, _ := module.Execute(cmd)
			if exitCode != 1 {
				t.Errorf("Command %v should fail with exit code 1 when logger disabled", cmd)
			}
			if output != "Logging is disabled" {
				t.Errorf("Command %v should return 'Logging is disabled'", cmd)
			}
		}
		return
	}

	// Test level changes are reflected
	testCases := []struct {
		setLevel      string
		expectedLevel logger.LogLevel
	}{
		{"debug", logger.DEBUG},
		{"info", logger.INFO},
		{"warn", logger.WARN},
		{"error", logger.ERROR},
	}

	for _, tc := range testCases {
		module.Execute([]string{"level", tc.setLevel})
		if log.GetLevel() != tc.expectedLevel {
			t.Errorf("Logger level should be %v after setting to %s", tc.expectedLevel, tc.setLevel)
		}
	}
}
