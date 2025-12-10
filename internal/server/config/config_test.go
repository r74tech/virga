package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config == nil {
		t.Fatal("Expected config, got nil")
	}

	// Test Server config
	if config.Server.Host != "0.0.0.0" {
		t.Errorf("Expected host 0.0.0.0, got %s", config.Server.Host)
	}

	if config.Server.AdminPort != 8443 {
		t.Errorf("Expected admin port 8443, got %d", config.Server.AdminPort)
	}

	if config.Server.SessionTimeout != 30*time.Minute {
		t.Errorf("Expected session timeout 30m, got %v", config.Server.SessionTimeout)
	}

	// Test Database config
	if config.Database.Path != "data/virga.db" {
		t.Errorf("Expected database path data/virga.db, got %s", config.Database.Path)
	}

	// Test Listeners
	if len(config.Listeners) != 1 {
		t.Fatalf("Expected 1 listener, got %d", len(config.Listeners))
	}

	listener := config.Listeners[0]
	if listener.Name != "default-http" {
		t.Errorf("Expected listener name default-http, got %s", listener.Name)
	}

	if listener.Port != 8080 {
		t.Errorf("Expected listener port 8080, got %d", listener.Port)
	}

	// Test MCP config
	if !config.MCP.Enabled {
		t.Error("Expected MCP to be enabled by default")
	}
}

func TestLoadConfig(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test_config.yaml")

	configContent := `
server:
  host: 127.0.0.1
  admin_port: 9443
  session_timeout: 45m
  log_level: debug
  log_path: /tmp/test.log

database:
  path: /tmp/test.db
  type: sqlite3

listeners:
  - name: test-listener
    type: http
    bind_address: 127.0.0.1
    port: 9090
    use_ssl: false
    uri_path: /test
    encryption:
      type: aes-256
      key: test-key

generator:
  user_agent: TestAgent/1.0
  initial_sleep: 30
  jitter: 10
  obfuscation: false
  anti_av: false
  anti_etw: false
  self_delete: false

mcp:
  enabled: true
  sse_enabled: false
  stdio_enabled: true
`

	if err := os.WriteFile(configPath, []byte(configContent), 0o644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	// Load the config
	config, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify loaded values
	if config.Server.Host != "127.0.0.1" {
		t.Errorf("Expected host 127.0.0.1, got %s", config.Server.Host)
	}

	if config.Server.AdminPort != 9443 {
		t.Errorf("Expected admin port 9443, got %d", config.Server.AdminPort)
	}

	if config.Server.SessionTimeout != 45*time.Minute {
		t.Errorf("Expected session timeout 45m, got %v", config.Server.SessionTimeout)
	}

	if len(config.Listeners) != 1 {
		t.Fatalf("Expected 1 listener, got %d", len(config.Listeners))
	}

	if config.Listeners[0].Port != 9090 {
		t.Errorf("Expected listener port 9090, got %d", config.Listeners[0].Port)
	}

	if config.MCP.StdioEnabled != true {
		t.Error("Expected MCP stdio to be enabled")
	}
}

func TestLoadConfigErrors(t *testing.T) {
	tests := []struct {
		name          string
		configYAML    string
		wantErr       bool
		errorContains string
	}{
		{
			name: "Missing database path",
			configYAML: `
server:
  host: 0.0.0.0
database:
  type: sqlite3
listeners:
  - name: test
    port: 8080
`,
			wantErr:       true,
			errorContains: "database path is required",
		},
		{
			name: "No listeners",
			configYAML: `
database:
  path: /tmp/test.db
listeners: []
`,
			wantErr:       true,
			errorContains: "at least one listener is required",
		},
		{
			name: "Invalid port",
			configYAML: `
database:
  path: /tmp/test.db
listeners:
  - name: test
    port: 99999
`,
			wantErr:       true,
			errorContains: "invalid port number",
		},
		{
			name: "SSL enabled without cert/key",
			configYAML: `
database:
  path: /tmp/test.db
listeners:
  - name: test
    port: 8443
    use_ssl: true
`,
			wantErr:       true,
			errorContains: "SSL enabled but missing cert/key",
		},
		{
			name: "Missing listener name",
			configYAML: `
database:
  path: /tmp/test.db
listeners:
  - port: 8080
`,
			wantErr:       true,
			errorContains: "name is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "test.yaml")

			if err := os.WriteFile(configPath, []byte(tt.configYAML), 0o644); err != nil {
				t.Fatalf("Failed to write test config: %v", err)
			}

			_, err := LoadConfig(configPath)
			if tt.wantErr {
				if err == nil {
					t.Error("Expected error, got nil")
				} else if tt.errorContains != "" && !contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error containing %q, got %v", tt.errorContains, err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
			}
		})
	}
}

func TestLoadConfigFileNotFound(t *testing.T) {
	_, err := LoadConfig("/non/existent/file.yaml")
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
}

func TestLoadConfigInvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "invalid.yaml")

	invalidYAML := `
server:
  host: [this is not valid yaml
`

	if err := os.WriteFile(configPath, []byte(invalidYAML), 0o644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	_, err := LoadConfig(configPath)
	if err == nil {
		t.Error("Expected error for invalid YAML")
	}

	if !contains(err.Error(), "could not parse config file") {
		t.Errorf("Expected parse error, got: %v", err)
	}
}

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "Valid config",
			config: &Config{
				Database: DatabaseConfig{
					Path: "/tmp/test.db",
				},
				Listeners: []ListenerConfig{
					{
						Name: "test",
						Port: 8080,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Zero port",
			config: &Config{
				Database: DatabaseConfig{
					Path: "/tmp/test.db",
				},
				Listeners: []ListenerConfig{
					{
						Name: "test",
						Port: 0,
					},
				},
			},
			wantErr: true,
		},
		{
			name: "Negative port",
			config: &Config{
				Database: DatabaseConfig{
					Path: "/tmp/test.db",
				},
				Listeners: []ListenerConfig{
					{
						Name: "test",
						Port: -1,
					},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateConfig(tt.config)
			if tt.wantErr {
				if err == nil {
					t.Error("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
			}
		})
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && len(substr) > 0 &&
		(s == substr || (len(s) > len(substr) && contains(s[1:], substr)) ||
			(len(s) >= len(substr) && s[:len(substr)] == substr))
}
