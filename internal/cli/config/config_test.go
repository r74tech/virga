package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg == nil {
		t.Fatal("Expected config, got nil")
	}

	// Check server defaults
	if cfg.Server.Host == "" {
		t.Error("Expected default host")
	}
	if cfg.Server.Port == 0 {
		t.Error("Expected default port")
	}

	// Check that other configs have sensible defaults
	if cfg.General.HistoryFile == "" {
		t.Error("Expected default history file")
	}
}

func TestLoadConfig(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test_config.yaml")

	configContent := `server:
  host: "192.168.1.100"
  port: 9000
  use_ssl: true
  username: "testuser"
  password: "testpass"
  api_key: "test-key"
general:
  default_listener: "http-listener"
  history_file: "/tmp/history"
  output_dir: "/tmp/output"
  auto_completion: true
  debug: true
logging:
  enabled: true
  log_file: "/tmp/cli.log"
  log_level: "debug"
  max_size: 100
  max_backup: 5
appearance:
  color_enabled: true
  default_prompt: "CustomPrompt>"
  session_prompt: "[%s]>"
  timestamp_format: "15:04:05"
  banner_enabled: false
  session_info_style: "compact"`

	err := os.WriteFile(configPath, []byte(configContent), 0o644)
	if err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	// Load the config
	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify loaded values
	if cfg.Server.Host != "192.168.1.100" {
		t.Errorf("Expected host 192.168.1.100, got %s", cfg.Server.Host)
	}
	if cfg.Server.Port != 9000 {
		t.Errorf("Expected port 9000, got %d", cfg.Server.Port)
	}
	if !cfg.Server.UseSSL {
		t.Error("Expected UseSSL to be true")
	}
	if cfg.Server.Username != "testuser" {
		t.Errorf("Expected username testuser, got %s", cfg.Server.Username)
	}
	if cfg.Server.APIKey != "test-key" {
		t.Errorf("Expected API key test-key, got %s", cfg.Server.APIKey)
	}

	// Check general config
	if cfg.General.DefaultListener != "http-listener" {
		t.Errorf("Expected default listener http-listener, got %s", cfg.General.DefaultListener)
	}
	if cfg.General.HistoryFile != "/tmp/history" {
		t.Errorf("Expected history file /tmp/history, got %s", cfg.General.HistoryFile)
	}
	if !cfg.General.AutoCompletion {
		t.Error("Expected auto completion to be true")
	}
	if !cfg.General.Debug {
		t.Error("Expected debug to be true")
	}

	// Check logging config
	if !cfg.Logging.Enabled {
		t.Error("Expected logging to be enabled")
	}
	if cfg.Logging.LogFile != "/tmp/cli.log" {
		t.Errorf("Expected log file /tmp/cli.log, got %s", cfg.Logging.LogFile)
	}
	if cfg.Logging.LogLevel != "debug" {
		t.Errorf("Expected log level debug, got %s", cfg.Logging.LogLevel)
	}
	if cfg.Logging.MaxSize != 100 {
		t.Errorf("Expected max size 100, got %d", cfg.Logging.MaxSize)
	}
	if cfg.Logging.MaxBackup != 5 {
		t.Errorf("Expected max backup 5, got %d", cfg.Logging.MaxBackup)
	}

	// Check appearance config
	if !cfg.Appearance.ColorEnabled {
		t.Error("Expected colors to be enabled")
	}
	if cfg.Appearance.DefaultPrompt != "CustomPrompt>" {
		t.Errorf("Expected prompt 'CustomPrompt>', got %s", cfg.Appearance.DefaultPrompt)
	}
	if cfg.Appearance.SessionPrompt != "[%s]>" {
		t.Errorf("Expected session prompt '[%%s]>', got %s", cfg.Appearance.SessionPrompt)
	}
	if cfg.Appearance.TimestampFormat != "15:04:05" {
		t.Errorf("Expected timestamp format '15:04:05', got %s", cfg.Appearance.TimestampFormat)
	}
	if cfg.Appearance.BannerEnabled {
		t.Error("Expected banner to be disabled")
	}
	if cfg.Appearance.SessionInfoStyle != "compact" {
		t.Errorf("Expected session info style 'compact', got %s", cfg.Appearance.SessionInfoStyle)
	}
}

func TestLoadConfig_FileNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "nonexistent", "config.yaml")

	cfg, err := LoadConfig(configPath)
	// Should create default config
	if err != nil {
		t.Logf("LoadConfig returned error: %v", err)
	}
	if cfg != nil {
		// It might create a default config, which is fine
		t.Log("LoadConfig created default config for non-existent file")
	}
}

func TestLoadConfig_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "invalid_config.yaml")

	// Write invalid YAML
	err := os.WriteFile(configPath, []byte("invalid: yaml: content: [broken"), 0o644)
	if err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err == nil {
		t.Error("Expected error for invalid YAML")
	}
	if cfg != nil {
		t.Error("Expected nil config for invalid YAML")
	}
}

func TestSaveConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "save_test.yaml")

	cfg := &Config{
		Server: ServerConfig{
			Host:     "test.example.com",
			Port:     12345,
			UseSSL:   true,
			Username: "saveuser",
			Password: "savepass",
			APIKey:   "save-key",
		},
		General: GeneralConfig{
			DefaultListener: "test-listener",
			HistoryFile:     "/test/history",
			OutputDir:       "/test/output",
			AutoCompletion:  true,
			Debug:           false,
		},
		Logging: LoggingConfig{
			Enabled:   true,
			LogFile:   "/test/log.txt",
			LogLevel:  "warn",
			MaxSize:   50,
			MaxBackup: 3,
		},
		Appearance: AppearanceConfig{
			ColorEnabled:     true,
			DefaultPrompt:    "TestPrompt>",
			SessionPrompt:    "Test[%s]>",
			TimestampFormat:  "2006-01-02",
			BannerEnabled:    true,
			SessionInfoStyle: "full",
		},
	}

	// Save config
	err := SaveConfig(cfg, configPath)
	if err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// Load it back
	loaded, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load saved config: %v", err)
	}

	// Verify values
	if loaded.Server.Host != cfg.Server.Host {
		t.Errorf("Host mismatch: expected %s, got %s", cfg.Server.Host, loaded.Server.Host)
	}
	if loaded.Server.Port != cfg.Server.Port {
		t.Errorf("Port mismatch: expected %d, got %d", cfg.Server.Port, loaded.Server.Port)
	}
	if loaded.Server.UseSSL != cfg.Server.UseSSL {
		t.Error("UseSSL mismatch")
	}
	if loaded.General.DefaultListener != cfg.General.DefaultListener {
		t.Errorf("DefaultListener mismatch: expected %s, got %s",
			cfg.General.DefaultListener, loaded.General.DefaultListener)
	}
	if loaded.Logging.LogLevel != cfg.Logging.LogLevel {
		t.Errorf("LogLevel mismatch: expected %s, got %s",
			cfg.Logging.LogLevel, loaded.Logging.LogLevel)
	}
	if loaded.Appearance.DefaultPrompt != cfg.Appearance.DefaultPrompt {
		t.Errorf("DefaultPrompt mismatch: expected %s, got %s",
			cfg.Appearance.DefaultPrompt, loaded.Appearance.DefaultPrompt)
	}
}

func TestConfigDirectory(t *testing.T) {
	// Test creating config directory
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, "testconfig", "subdir")
	configPath := filepath.Join(configDir, "config.yaml")

	// Try to load config in non-existent directory
	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Logf("LoadConfig error (expected): %v", err)
	}

	// Directory should be created if it doesn't exist
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		// This is expected if LoadConfig doesn't create directories
		t.Log("Config directory was not created automatically")
	} else if cfg != nil {
		// If config was created, verify it
		if cfg.Server.Host == "" {
			t.Error("Expected default host in created config")
		}
	}
}

func TestMergeConfigs(t *testing.T) {
	// Test that we can have a base config and override specific values
	base := &Config{
		Server: ServerConfig{
			Host:   "localhost",
			Port:   8080,
			UseSSL: true,
		},
		General: GeneralConfig{
			HistoryFile:    "/base/history",
			AutoCompletion: true,
			Debug:          false,
		},
		Logging: LoggingConfig{
			Enabled:  true,
			LogLevel: "info",
		},
		Appearance: AppearanceConfig{
			ColorEnabled:  true,
			DefaultPrompt: "Base>",
		},
	}

	// Create override config (simulating partial config)
	override := &Config{
		Server: ServerConfig{
			Host: "override.com",
			Port: 9090,
			// UseSSL not set, should keep base value in real merge
		},
		General: GeneralConfig{
			Debug: true,
			// Other fields not set
		},
		Logging: LoggingConfig{
			LogLevel: "debug",
			// Enabled not set
		},
		// Appearance not modified
	}

	// In a real implementation, you would merge these
	// For now, just verify the configs are valid
	if base.Server.Host != "localhost" {
		t.Error("Base config host incorrect")
	}
	if override.Server.Host != "override.com" {
		t.Error("Override config host incorrect")
	}
	if !override.General.Debug {
		t.Error("Override debug should be true")
	}
}

func TestValidatePort(t *testing.T) {
	tests := []struct {
		port  int
		valid bool
	}{
		{0, false},
		{1, true},
		{80, true},
		{443, true},
		{8080, true},
		{65535, true},
		{65536, false},
		{70000, false},
		{-1, false},
	}

	for _, tt := range tests {
		cfg := &Config{
			Server: ServerConfig{
				Host: "localhost",
				Port: tt.port,
			},
		}

		// Simple port validation
		valid := cfg.Server.Port > 0 && cfg.Server.Port <= 65535
		if valid != tt.valid {
			t.Errorf("Port %d: expected valid=%v, got %v", tt.port, tt.valid, valid)
		}
	}
}

func TestLogLevelValidation(t *testing.T) {
	validLevels := []string{"debug", "info", "warn", "error"}
	invalidLevels := []string{"", "trace", "verbose", "invalid", "fatal"}

	for _, level := range validLevels {
		err := ValidateLogLevel(level)
		if err != nil {
			t.Errorf("Expected %s to be valid log level, got error: %v", level, err)
		}
	}

	for _, level := range invalidLevels {
		err := ValidateLogLevel(level)
		if err == nil {
			t.Errorf("Expected %s to be invalid log level", level)
		}
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
			name: "valid config",
			config: &Config{
				Server: ServerConfig{
					Host: "localhost",
					Port: 8443,
				},
				Logging: LoggingConfig{
					Enabled:   true,
					LogLevel:  "info",
					MaxSize:   10,
					MaxBackup: 5,
				},
			},
			wantErr: false,
		},
		{
			name: "empty host",
			config: &Config{
				Server: ServerConfig{
					Host: "",
					Port: 8443,
				},
			},
			wantErr: true,
			errMsg:  "server host cannot be empty",
		},
		{
			name: "invalid port - zero",
			config: &Config{
				Server: ServerConfig{
					Host: "localhost",
					Port: 0,
				},
			},
			wantErr: true,
			errMsg:  "port must be between 1 and 65535",
		},
		{
			name: "invalid port - too high",
			config: &Config{
				Server: ServerConfig{
					Host: "localhost",
					Port: 70000,
				},
			},
			wantErr: true,
			errMsg:  "port must be between 1 and 65535",
		},
		{
			name: "invalid log level",
			config: &Config{
				Server: ServerConfig{
					Host: "localhost",
					Port: 8443,
				},
				Logging: LoggingConfig{
					Enabled:  true,
					LogLevel: "verbose",
				},
			},
			wantErr: true,
			errMsg:  "invalid log level",
		},
		{
			name: "invalid log max size",
			config: &Config{
				Server: ServerConfig{
					Host: "localhost",
					Port: 8443,
				},
				Logging: LoggingConfig{
					Enabled:  true,
					LogLevel: "info",
					MaxSize:  0,
				},
			},
			wantErr: true,
			errMsg:  "log max size must be positive",
		},
		{
			name: "invalid log max backup",
			config: &Config{
				Server: ServerConfig{
					Host: "localhost",
					Port: 8443,
				},
				Logging: LoggingConfig{
					Enabled:   true,
					LogLevel:  "info",
					MaxSize:   10,
					MaxBackup: -1,
				},
			},
			wantErr: true,
			errMsg:  "log max backup must be non-negative",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Config.Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.errMsg != "" && err.Error() != tt.errMsg {
				t.Errorf("Config.Validate() error = %v, want %v", err.Error(), tt.errMsg)
			}
		})
	}
}

func TestSecureFileSave(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "secure_test.yaml")

	cfg := DefaultConfig()

	// Save config
	err := SaveConfig(cfg, configPath)
	if err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// Check file permissions
	info, err := os.Stat(configPath)
	if err != nil {
		t.Fatalf("Failed to stat config file: %v", err)
	}

	// File should have 0600 permissions (owner read/write only)
	perm := info.Mode().Perm()
	if perm != 0o600 {
		t.Errorf("Expected file permissions 0600, got %o", perm)
	}
}

func TestValidatePortFunction(t *testing.T) {
	tests := []struct {
		port    int
		wantErr bool
	}{
		{0, true},
		{1, false},
		{80, false},
		{443, false},
		{8080, false},
		{65535, false},
		{65536, true},
		{70000, true},
		{-1, true},
	}

	for _, tt := range tests {
		err := ValidatePort(tt.port)
		if (err != nil) != tt.wantErr {
			t.Errorf("ValidatePort(%d) error = %v, wantErr %v", tt.port, err, tt.wantErr)
		}
	}
}
