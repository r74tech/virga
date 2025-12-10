package config

import (
	"testing"
)

func TestTransportTypeConstants(t *testing.T) {
	// Verify transport type constants are defined correctly
	tests := []struct {
		transport Transport
		expected  string
	}{
		{TransportSSE, "sse"},
		{TransportStdio, "stdio"},
		{TransportStreamable, "streamable"},
	}

	for _, tt := range tests {
		t.Run(string(tt.transport), func(t *testing.T) {
			if string(tt.transport) != tt.expected {
				t.Errorf("Transport %s != expected %s", tt.transport, tt.expected)
			}
		})
	}
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "Valid config with all transports",
			config: &Config{
				Name:              "Test Server",
				Version:           "1.0.0",
				SSEEnabled:        true,
				SSEPort:           ":8444",
				SSEBasePath:       "/mcp",
				StdioEnabled:      true,
				StreamableEnabled: true,
				StreamablePort:    ":50012",
			},
			wantErr: false,
		},
		{
			name: "SSE enabled without port",
			config: &Config{
				Name:       "Test Server",
				Version:    "1.0.0",
				SSEEnabled: true,
				SSEPort:    "",
			},
			wantErr: true,
			errMsg:  "SSE port",
		},
		{
			name: "Streamable enabled without port",
			config: &Config{
				Name:              "Test Server",
				Version:           "1.0.0",
				StreamableEnabled: true,
				StreamablePort:    "",
			},
			wantErr: true,
			errMsg:  "Streamable port",
		},
		{
			name: "Empty name",
			config: &Config{
				Name:    "",
				Version: "1.0.0",
			},
			wantErr: true,
			errMsg:  "name",
		},
		{
			name: "Empty version",
			config: &Config{
				Name:    "Test Server",
				Version: "",
			},
			wantErr: true,
			errMsg:  "version",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && tt.errMsg != "" {
				if !contains(err.Error(), tt.errMsg) {
					t.Errorf("Validate() error = %v, want error containing %v", err, tt.errMsg)
				}
			}
		})
	}
}

func TestConfigDefaults(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Name == "" {
		t.Error("Default config should have a name")
	}
	if cfg.Version == "" {
		t.Error("Default config should have a version")
	}
	if cfg.SSEBasePath == "" {
		t.Error("Default config should have SSE base path")
	}
}

func TestConfigCopy(t *testing.T) {
	original := &Config{
		Name:              "Original",
		Version:           "1.0.0",
		SSEEnabled:        true,
		SSEPort:           ":8444",
		SSEBasePath:       "/mcp",
		SSERemoteURL:      "http://localhost:8444",
		StdioEnabled:      true,
		StreamableEnabled: true,
		StreamablePort:    ":50012",
	}

	// Test that modifying copy doesn't affect original
	copy := original.Copy()
	copy.Name = "Modified"
	copy.SSEEnabled = false

	if original.Name != "Original" {
		t.Error("Modifying copy affected original name")
	}
	if !original.SSEEnabled {
		t.Error("Modifying copy affected original SSEEnabled")
	}
}

// Helper functions
func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[:len(substr)] == substr ||
		len(s) > len(substr) && contains(s[1:], substr)
}

// Add these methods to config.go to make tests pass:
// Validate() error - validates the configuration
// NewDefaultConfig() *Config - returns default configuration
// Copy() *Config - returns a deep copy of the configuration
