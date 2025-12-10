package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config is the CLI configuration.
type Config struct {
	Server     ServerConfig     `yaml:"server"`
	General    GeneralConfig    `yaml:"general"`
	Logging    LoggingConfig    `yaml:"logging"`
	Appearance AppearanceConfig `yaml:"appearance"`
}

// ServerConfig is the server connection related configuration.
type ServerConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	UseSSL   bool   `yaml:"use_ssl"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	APIKey   string `yaml:"api_key"`
}

// GeneralConfig is the general configuration.
type GeneralConfig struct {
	DefaultListener     string `yaml:"default_listener"`
	HistoryFile         string `yaml:"history_file"`
	OutputDir           string `yaml:"output_dir"`
	AutoCompletion      bool   `yaml:"auto_completion"`
	Debug               bool   `yaml:"debug"`
	CompletionSortOrder string `yaml:"completion_sort_order"` // "rank" or "alphabetical"
}

// LoggingConfig is the logging related configuration.
type LoggingConfig struct {
	Enabled      bool   `yaml:"enabled"`
	LogFile      string `yaml:"log_file"`
	LogLevel     string `yaml:"log_level"`
	MaxSize      int    `yaml:"max_size"`
	MaxBackup    int    `yaml:"max_backup"`
	JSONLEnabled bool   `yaml:"jsonl_enabled"`
	JSONLFile    string `yaml:"jsonl_file"`
	JSONLLevel   string `yaml:"jsonl_level"`
}

// AppearanceConfig is the appearance related configuration.
type AppearanceConfig struct {
	ColorEnabled     bool   `yaml:"color_enabled"`
	DefaultPrompt    string `yaml:"default_prompt"`
	SessionPrompt    string `yaml:"session_prompt"`
	TimestampFormat  string `yaml:"timestamp_format"`
	BannerEnabled    bool   `yaml:"banner_enabled"`
	SessionInfoStyle string `yaml:"session_info_style"`
}

// LoadConfig reads the configuration file.
func LoadConfig(filePath string) (*Config, error) {
	// Read the configuration file
	data, err := os.ReadFile(filePath)
	if err != nil {
		// If the configuration file does not exist, create a default configuration
		if os.IsNotExist(err) {
			cfg := DefaultConfig()

			// Create the configuration directory
			configDir := filepath.Dir(filePath)
			if err := os.MkdirAll(configDir, 0o755); err != nil {
				return nil, fmt.Errorf("could not create config directory: %w", err)
			}

			// Save the default configuration
			if err := SaveConfig(cfg, filePath); err != nil {
				return nil, fmt.Errorf("could not save default config: %w", err)
			}

			return cfg, nil
		}

		return nil, fmt.Errorf("could not read config file: %w", err)
	}

	// Parse YAML
	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("could not parse config file: %w", err)
	}

	return &config, nil
}

// SaveConfig saves the configuration to a file.
func SaveConfig(config *Config, filePath string) error {
	// Convert to YAML format
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("could not marshal config: %w", err)
	}

	// Write to file with secure permissions (owner read/write only)
	if err := os.WriteFile(filePath, data, 0o600); err != nil {
		return fmt.Errorf("could not write config file: %w", err)
	}

	return nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	// Server validation
	if c.Server.Host == "" {
		return ErrEmptyHost
	}

	if err := ValidatePort(c.Server.Port); err != nil {
		return err
	}

	// Logging validation
	if c.Logging.Enabled {
		if err := ValidateLogLevel(c.Logging.LogLevel); err != nil {
			return err
		}

		if c.Logging.MaxSize <= 0 {
			return fmt.Errorf("log max size must be positive")
		}

		if c.Logging.MaxBackup < 0 {
			return fmt.Errorf("log max backup must be non-negative")
		}
	}

	return nil
}

// SetDefaults sets default values for the configuration
func (c *Config) SetDefaults() {
	*c = *DefaultConfig()
}

// DefaultConfig returns the default configuration.
func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Host:     "localhost",
			Port:     8443,
			UseSSL:   false, // Disable HTTPS
			Username: "",
			Password: "",
			APIKey:   "",
		},
		General: GeneralConfig{
			DefaultListener:     "default-http",
			HistoryFile:         ".virga_history",
			OutputDir:           "dist",
			AutoCompletion:      true,
			Debug:               false,
			CompletionSortOrder: "rank", // Default to priority-based sorting
		},
		Logging: LoggingConfig{
			Enabled:      true,
			LogFile:      "logs/cli.log",
			LogLevel:     "info",
			MaxSize:      10, // MB
			MaxBackup:    5,
			JSONLEnabled: false,
			JSONLFile:    "logs/virga.jsonl",
			JSONLLevel:   "debug", // Record all logs in JSONL
		},
		Appearance: AppearanceConfig{
			ColorEnabled:     true,
			DefaultPrompt:    "virga> ",
			SessionPrompt:    "virga (%s)> ",
			TimestampFormat:  "2006-01-02 15:04:05",
			BannerEnabled:    true,
			SessionInfoStyle: "compact",
		},
	}
}
