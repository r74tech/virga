package config

import (
	"testing"
)

func TestNewConfig(t *testing.T) {
	// Save original values
	originalHost := BuildC2Host
	originalPort := BuildC2Port
	originalProtocol := BuildC2Protocol
	originalPath := BuildC2Path
	originalUserAgent := BuildUserAgent
	originalSleepTime := BuildSleepTime
	originalJitter := BuildJitter

	// Set test values
	BuildC2Host = "test.example.com"
	BuildC2Port = "8443"
	BuildC2Protocol = "https"
	BuildC2Path = "test/api"
	BuildUserAgent = "TestAgent/1.0"
	BuildSleepTime = "120"
	BuildJitter = "30"

	// Create config
	cfg := NewConfig()

	// Verify values
	if cfg.C2Host != BuildC2Host {
		t.Errorf("Expected C2Host %s, got %s", BuildC2Host, cfg.C2Host)
	}
	if cfg.C2Port != BuildC2Port {
		t.Errorf("Expected C2Port %s, got %s", BuildC2Port, cfg.C2Port)
	}
	if cfg.C2Protocol != BuildC2Protocol {
		t.Errorf("Expected C2Protocol %s, got %s", BuildC2Protocol, cfg.C2Protocol)
	}
	if cfg.C2Path != BuildC2Path {
		t.Errorf("Expected C2Path %s, got %s", BuildC2Path, cfg.C2Path)
	}
	if cfg.UserAgent != BuildUserAgent {
		t.Errorf("Expected UserAgent %s, got %s", BuildUserAgent, cfg.UserAgent)
	}
	if cfg.SleepTime != BuildSleepTime {
		t.Errorf("Expected SleepTime %s, got %s", BuildSleepTime, cfg.SleepTime)
	}
	if cfg.Jitter != BuildJitter {
		t.Errorf("Expected Jitter %s, got %s", BuildJitter, cfg.Jitter)
	}

	// Restore original values
	BuildC2Host = originalHost
	BuildC2Port = originalPort
	BuildC2Protocol = originalProtocol
	BuildC2Path = originalPath
	BuildUserAgent = originalUserAgent
	BuildSleepTime = originalSleepTime
	BuildJitter = originalJitter
}

func TestGetImplantLogEnabled(t *testing.T) {
	tests := []struct {
		name       string
		buildValue string
		expected   bool
	}{
		{
			name:       "Enabled - true",
			buildValue: "true",
			expected:   true,
		},
		{
			name:       "Enabled - TRUE",
			buildValue: "TRUE",
			expected:   false, // Case sensitive
		},
		{
			name:       "Disabled - false",
			buildValue: "false",
			expected:   false,
		},
		{
			name:       "Empty value",
			buildValue: "",
			expected:   false,
		},
		{
			name:       "Invalid value",
			buildValue: "yes",
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalBuild := BuildImplantLogEnabled
			BuildImplantLogEnabled = tt.buildValue

			result := GetImplantLogEnabled()
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}

			BuildImplantLogEnabled = originalBuild
		})
	}
}

func TestGetImplantLogFilePath(t *testing.T) {
	tests := []struct {
		name       string
		buildValue string
		expected   string
	}{
		{
			name:       "Custom log path",
			buildValue: "/var/log/implant.log",
			expected:   "/var/log/implant.log",
		},
		{
			name:       "Empty value",
			buildValue: "",
			expected:   "implant.log",
		},
		{
			name:       "Windows path",
			buildValue: "C:\\Logs\\implant.log",
			expected:   "C:\\Logs\\implant.log",
		},
		{
			name:       "Relative path",
			buildValue: "../logs/implant.log",
			expected:   "../logs/implant.log",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalBuild := BuildImplantLogFilePath
			BuildImplantLogFilePath = tt.buildValue

			result := GetImplantLogFilePath()
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}

			BuildImplantLogFilePath = originalBuild
		})
	}
}

func TestGetImplantLogLevel(t *testing.T) {
	tests := []struct {
		name       string
		buildValue string
		expected   string
	}{
		{
			name:       "Debug level",
			buildValue: "debug",
			expected:   "debug",
		},
		{
			name:       "Info level",
			buildValue: "info",
			expected:   "info",
		},
		{
			name:       "Warning level",
			buildValue: "warning",
			expected:   "warning",
		},
		{
			name:       "Error level",
			buildValue: "error",
			expected:   "error",
		},
		{
			name:       "Empty value",
			buildValue: "",
			expected:   "info",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalBuild := BuildImplantLogLevel
			BuildImplantLogLevel = tt.buildValue

			result := GetImplantLogLevel()
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}

			BuildImplantLogLevel = originalBuild
		})
	}
}

func TestBuildDefaults(t *testing.T) {
	// Test default build values
	tests := []struct {
		name     string
		variable *string
		expected string
	}{
		{
			name:     "Default C2Host",
			variable: &BuildC2Host,
			expected: "localhost",
		},
		{
			name:     "Default C2Port",
			variable: &BuildC2Port,
			expected: "443",
		},
		{
			name:     "Default C2Protocol",
			variable: &BuildC2Protocol,
			expected: "https",
		},
		{
			name:     "Default C2Path",
			variable: &BuildC2Path,
			expected: "api/updates",
		},
		{
			name:     "Default SleepTime",
			variable: &BuildSleepTime,
			expected: "60",
		},
		{
			name:     "Default Jitter",
			variable: &BuildJitter,
			expected: "20",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if *tt.variable != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, *tt.variable)
			}
		})
	}
}

func TestConfigCreation(t *testing.T) {
	// Test that Config struct is properly created
	cfg := NewConfig()

	if cfg == nil {
		t.Fatal("Expected config to be created")
	}

	// Verify all fields are set from build variables
	if cfg.C2Host == "" {
		t.Error("C2Host should not be empty")
	}
	if cfg.C2Port == "" {
		t.Error("C2Port should not be empty")
	}
	if cfg.C2Protocol == "" {
		t.Error("C2Protocol should not be empty")
	}
	if cfg.C2Path == "" {
		t.Error("C2Path should not be empty")
	}
	if cfg.UserAgent == "" {
		t.Error("UserAgent should not be empty")
	}
	if cfg.SleepTime == "" {
		t.Error("SleepTime should not be empty")
	}
	if cfg.Jitter == "" {
		t.Error("Jitter should not be empty")
	}
}

func TestBuildVariableModification(t *testing.T) {
	// Test that build variables can be modified
	original := BuildC2Host
	defer func() { BuildC2Host = original }()

	// Modify and verify
	BuildC2Host = "modified.example.com"
	cfg := NewConfig()

	if cfg.C2Host != "modified.example.com" {
		t.Errorf("Expected modified C2Host, got %s", cfg.C2Host)
	}
}
