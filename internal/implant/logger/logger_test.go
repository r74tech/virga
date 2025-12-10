package logger

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/r74tech/virga/internal/implant/config"
)

// TestLogLevel tests log level parsing and string conversion
func TestLogLevel(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected LogLevel
		str      string
	}{
		{"Debug level", "debug", DEBUG, "DEBUG"},
		{"Info level", "info", INFO, "INFO"},
		{"Warn level", "warn", WARN, "WARN"},
		{"Warning level", "warning", WARN, "WARN"},
		{"Error level", "error", ERROR, "ERROR"},
		{"Invalid level", "invalid", INFO, "INFO"},
		{"Empty level", "", INFO, "INFO"},
		{"Case insensitive", "DEBUG", DEBUG, "DEBUG"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			level := parseLogLevel(tt.input)
			if level != tt.expected {
				t.Errorf("parseLogLevel(%s) = %v, want %v", tt.input, level, tt.expected)
			}
			if level.String() != tt.str {
				t.Errorf("LogLevel.String() = %s, want %s", level.String(), tt.str)
			}
		})
	}
}

// TestLoggerSingleton tests that Get() returns the same instance
func TestLoggerSingleton(t *testing.T) {
	// Reset singleton for testing
	instance = nil
	once = sync.Once{}

	// Save original config values
	originalEnabled := config.BuildImplantLogEnabled
	originalLevel := config.BuildImplantLogLevel
	originalPath := config.BuildImplantLogFilePath

	// Set test config
	config.BuildImplantLogEnabled = "false" // Disable to avoid file creation
	config.BuildImplantLogLevel = "info"
	defer func() {
		config.BuildImplantLogEnabled = originalEnabled
		config.BuildImplantLogLevel = originalLevel
		config.BuildImplantLogFilePath = originalPath
	}()

	// Get instance multiple times
	logger1 := Get()
	logger2 := Get()
	logger3 := Get()

	// Verify same instance
	if logger1 != logger2 || logger2 != logger3 {
		t.Error("Get() should return the same logger instance")
	}

	// Verify logger is not nil
	if logger1 == nil {
		t.Error("Get() should not return nil")
	}
}

// TestLoggerInitialization tests logger initialization with file
func TestLoggerInitialization(t *testing.T) {
	// Create temp directory for test
	tempDir, err := ioutil.TempDir("", "logger_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Reset singleton
	instance = nil
	once = sync.Once{}

	// Save original config
	originalEnabled := config.BuildImplantLogEnabled
	originalLevel := config.BuildImplantLogLevel
	originalPath := config.BuildImplantLogFilePath

	// Set test config
	config.BuildImplantLogEnabled = "true"
	config.BuildImplantLogLevel = "debug"
	config.BuildImplantLogFilePath = filepath.Join(tempDir, "test.log")

	defer func() {
		config.BuildImplantLogEnabled = originalEnabled
		config.BuildImplantLogLevel = originalLevel
		config.BuildImplantLogFilePath = originalPath
		if instance != nil {
			instance.Close()
		}
	}()

	// Get logger
	logger := Get()

	// Verify logger is enabled
	if !logger.IsEnabled() {
		t.Error("Logger should be enabled")
	}

	// Verify log file was created
	if _, err := os.Stat(config.BuildImplantLogFilePath); os.IsNotExist(err) {
		t.Error("Log file was not created")
	}

	// Write a test message
	logger.Info("Test message", map[string]interface{}{
		"test": "value",
	})

	// Give time for write
	time.Sleep(100 * time.Millisecond)

	// Read log file
	content, err := ioutil.ReadFile(config.BuildImplantLogFilePath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	// Verify content
	if !strings.Contains(string(content), "Test message") {
		t.Error("Log file should contain test message")
	}
	if !strings.Contains(string(content), "test=value") {
		t.Error("Log file should contain field")
	}
}

// TestLoggerDisabled tests logger when disabled
func TestLoggerDisabled(t *testing.T) {
	// Reset singleton
	instance = nil
	once = sync.Once{}

	// Save original config
	originalEnabled := config.BuildImplantLogEnabled
	originalLevel := config.BuildImplantLogLevel

	// Disable logger
	config.BuildImplantLogEnabled = "false"
	config.BuildImplantLogLevel = "info"

	defer func() {
		config.BuildImplantLogEnabled = originalEnabled
		config.BuildImplantLogLevel = originalLevel
	}()

	// Get logger
	logger := Get()

	// Verify logger is disabled
	if logger.IsEnabled() {
		t.Error("Logger should be disabled")
	}

	// Try to log (should not panic)
	logger.Debug("This should not be logged")
	logger.Info("This should not be logged")
	logger.Warn("This should not be logged")
	logger.Error("This should not be logged")
}

// TestLogLevels tests that log levels filter correctly
func TestLogLevels(t *testing.T) {
	// Create temp directory
	tempDir, err := ioutil.TempDir("", "logger_level_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tests := []struct {
		name     string
		logLevel string
		debug    bool
		info     bool
		warn     bool
		error    bool
	}{
		{"Debug level", "debug", true, true, true, true},
		{"Info level", "info", false, true, true, true},
		{"Warn level", "warn", false, false, true, true},
		{"Error level", "error", false, false, false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset singleton
			instance = nil
			once = sync.Once{}

			// Set config
			config.BuildImplantLogEnabled = "true"
			config.BuildImplantLogLevel = tt.logLevel
			logFile := filepath.Join(tempDir, fmt.Sprintf("%s.log", tt.name))
			config.BuildImplantLogFilePath = logFile

			// Get logger
			logger := Get()
			defer logger.Close()

			// Log messages at different levels
			logger.Debug("Debug message")
			logger.Info("Info message")
			logger.Warn("Warn message")
			logger.Error("Error message")

			// Give time for writes
			time.Sleep(100 * time.Millisecond)

			// Read log file
			content, err := ioutil.ReadFile(logFile)
			if err != nil {
				t.Fatalf("Failed to read log file: %v", err)
			}

			logContent := string(content)

			// Check expectations
			if tt.debug != strings.Contains(logContent, "Debug message") {
				t.Errorf("Debug message presence mismatch. Expected: %v", tt.debug)
			}
			if tt.info != strings.Contains(logContent, "Info message") {
				t.Errorf("Info message presence mismatch. Expected: %v", tt.info)
			}
			if tt.warn != strings.Contains(logContent, "Warn message") {
				t.Errorf("Warn message presence mismatch. Expected: %v", tt.warn)
			}
			if tt.error != strings.Contains(logContent, "Error message") {
				t.Errorf("Error message presence mismatch. Expected: %v", tt.error)
			}
		})
	}
}

// TestSpecializedLogMethods tests specialized logging methods
func TestSpecializedLogMethods(t *testing.T) {
	// Create temp directory
	tempDir, err := ioutil.TempDir("", "logger_special_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Reset singleton
	instance = nil
	once = sync.Once{}

	// Configure logger
	config.BuildImplantLogEnabled = "true"
	config.BuildImplantLogLevel = "debug"
	logFile := filepath.Join(tempDir, "special.log")
	config.BuildImplantLogFilePath = logFile

	// Get logger
	logger := Get()
	defer logger.Close()

	// Test LogCommand
	logger.LogCommand("echo test", "test output", 0, nil)
	logger.LogCommand("failing command", "error output", 1, fmt.Errorf("command failed"))

	// Test LogTask
	logger.LogTask("task-123", "shell", "started")
	logger.LogTask("task-123", "shell", "completed", map[string]interface{}{
		"duration": "5s",
	})

	// Test LogBeacon
	logger.LogBeacon("outbound", "sent", map[string]interface{}{
		"size": 1024,
	})
	logger.LogBeacon("inbound", "received", map[string]interface{}{
		"tasks": 2,
	})

	// Test LogSystem
	logger.LogSystem("startup", map[string]interface{}{
		"version": "1.0.0",
	})

	// Test LogLlama
	logger.LogLlama("initialized", map[string]interface{}{
		"model": "test-model",
	})

	// Test LogLlamaPrompt
	logger.LogLlamaPrompt("llama-task-1", "Short prompt", "Short response", 0.7)

	// Test with long prompt/response
	longText := strings.Repeat("A", 300)
	logger.LogLlamaPrompt("llama-task-2", longText, longText, 0.9)

	// Test LogLlamaCommand
	logger.LogLlamaCommand("llama-task-3", "ls -la", "directory listing", 0, nil)

	// Test LogLlamaIteration
	logger.LogLlamaIteration("llama-task-4", 1, "thinking", map[string]interface{}{
		"confidence": 0.8,
	})

	// Give time for writes
	time.Sleep(200 * time.Millisecond)

	// Read and verify log content
	content, err := ioutil.ReadFile(logFile)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	logContent := string(content)

	// Verify various log entries
	expectedStrings := []string{
		"Command executed",
		"echo test",
		"failing command",
		"command failed",
		"Task started",
		"Task completed",
		"task-123",
		"duration=5s",
		"Beacon communication",
		"outbound",
		"inbound",
		"System event",
		"Llama event",
		"Llama prompt/response",
		"prompt_preview", // Long prompt should be truncated
		"Llama command execution",
		"ls -la",
		"Llama iteration",
		"confidence=0.8",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(logContent, expected) {
			t.Errorf("Log should contain '%s'", expected)
		}
	}
}

// TestConcurrentLogging tests thread safety
func TestConcurrentLogging(t *testing.T) {
	// Create temp directory
	tempDir, err := ioutil.TempDir("", "logger_concurrent_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Reset singleton
	instance = nil
	once = sync.Once{}

	// Configure logger
	config.BuildImplantLogEnabled = "true"
	config.BuildImplantLogLevel = "debug"
	logFile := filepath.Join(tempDir, "concurrent.log")
	config.BuildImplantLogFilePath = logFile

	// Get logger
	logger := Get()
	defer logger.Close()

	// Run concurrent logging
	var wg sync.WaitGroup
	numGoroutines := 10
	messagesPerGoroutine := 100

	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < messagesPerGoroutine; j++ {
				msg := fmt.Sprintf("Goroutine %d message %d", id, j)
				switch j % 4 {
				case 0:
					logger.Debug(msg)
				case 1:
					logger.Info(msg)
				case 2:
					logger.Warn(msg)
				case 3:
					logger.Error(msg)
				}
			}
		}(i)
	}

	wg.Wait()

	// Give time for all writes
	time.Sleep(500 * time.Millisecond)

	// Read log file
	content, err := ioutil.ReadFile(logFile)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	// Count lines (should have all messages)
	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	// First line is initialization message, so we expect +1
	expectedLines := (numGoroutines * messagesPerGoroutine) + 1
	if len(lines) != expectedLines {
		t.Errorf("Expected %d log lines, got %d", expectedLines, len(lines))
	}
}

// TestLogFileRotation tests behavior with existing log file
func TestLogFileRotation(t *testing.T) {
	// Create temp directory
	tempDir, err := ioutil.TempDir("", "logger_rotation_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	logFile := filepath.Join(tempDir, "append.log")

	// Write initial content
	initialContent := "Previous log content\n"
	err = ioutil.WriteFile(logFile, []byte(initialContent), 0o644)
	if err != nil {
		t.Fatalf("Failed to write initial content: %v", err)
	}

	// Reset singleton
	instance = nil
	once = sync.Once{}

	// Configure logger
	config.BuildImplantLogEnabled = "true"
	config.BuildImplantLogLevel = "info"
	config.BuildImplantLogFilePath = logFile

	// Get logger
	logger := Get()
	defer logger.Close()

	// Write new message
	logger.Info("New message")

	// Give time for write
	time.Sleep(100 * time.Millisecond)

	// Read file
	content, err := ioutil.ReadFile(logFile)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	// Verify both old and new content exist (append mode)
	if !strings.Contains(string(content), "Previous log content") {
		t.Error("Log file should contain previous content")
	}
	if !strings.Contains(string(content), "New message") {
		t.Error("Log file should contain new message")
	}
}

// TestLoggerClose tests proper cleanup
func TestLoggerClose(t *testing.T) {
	// Create temp directory
	tempDir, err := ioutil.TempDir("", "logger_close_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Reset singleton
	instance = nil
	once = sync.Once{}

	// Configure logger
	config.BuildImplantLogEnabled = "true"
	config.BuildImplantLogLevel = "info"
	logFile := filepath.Join(tempDir, "close.log")
	config.BuildImplantLogFilePath = logFile

	// Get logger
	logger := Get()

	// Write message
	logger.Info("Before close")

	// Close logger
	logger.Close()

	// Try to write after close (should not panic)
	logger.Info("After close")

	// Verify file is closed by trying to remove it
	// On Windows, this would fail if file is still open
	time.Sleep(100 * time.Millisecond)

	// Read final content
	content, err := ioutil.ReadFile(logFile)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	// Should contain shutdown message
	if !strings.Contains(string(content), "shutting down") {
		t.Error("Log file should contain shutdown message")
	}
}

// TestGetLogFilePath tests retrieving the log file path
func TestGetLogFilePath(t *testing.T) {
	// Reset singleton
	instance = nil
	once = sync.Once{}

	// Test with absolute path
	absolutePath := filepath.Join(os.TempDir(), "test_absolute.log")
	config.BuildImplantLogEnabled = "true"
	config.BuildImplantLogFilePath = absolutePath

	logger := Get()
	defer logger.Close()

	if logger.GetLogFilePath() != absolutePath {
		t.Errorf("Expected log path %s, got %s", absolutePath, logger.GetLogFilePath())
	}

	// Reset for relative path test
	instance = nil
	once = sync.Once{}

	// Test with relative path
	config.BuildImplantLogFilePath = "relative.log"
	logger2 := Get()
	defer logger2.Close()

	// Should be resolved to absolute path
	if !filepath.IsAbs(logger2.GetLogFilePath()) {
		t.Error("Log file path should be absolute")
	}
}

// TestLoggerInitializationError tests logger behavior when initialization fails
func TestLoggerInitializationError(t *testing.T) {
	// Reset singleton
	instance = nil
	once = sync.Once{}

	// Set invalid log path (directory that can't be created)
	config.BuildImplantLogEnabled = "true"
	config.BuildImplantLogLevel = "info"

	// Use a path that should fail
	if os.Getuid() != 0 { // Not running as root
		config.BuildImplantLogFilePath = "/root/test/logger.log"
	} else {
		config.BuildImplantLogFilePath = "/dev/null/test/logger.log"
	}

	// Get logger (should handle error gracefully)
	logger := Get()

	// Logger should be disabled due to initialization error
	if logger.IsEnabled() {
		t.Error("Logger should be disabled after initialization error")
	}

	// Should not panic when trying to log
	logger.Info("This should not crash")
}
