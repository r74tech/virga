package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config represents the application-wide configuration
type Config struct {
	Server    ServerConfig     `yaml:"server"`
	Database  DatabaseConfig   `yaml:"database"`
	Listeners []ListenerConfig `yaml:"listeners"`
	Generator GeneratorConfig  `yaml:"generator"`
	MCP       MCPConfig        `yaml:"mcp"`
}

// ServerConfig represents the server-related configuration
type ServerConfig struct {
	Host           string        `yaml:"host"`
	AdminPort      int           `yaml:"admin_port"`
	SessionTimeout time.Duration `yaml:"session_timeout"`
	LogLevel       string        `yaml:"log_level"`
	LogPath        string        `yaml:"log_path"`
}

// DatabaseConfig represents the database-related configuration
type DatabaseConfig struct {
	Path string `yaml:"path"`
	Type string `yaml:"type"`
}

// ListenerConfig represents the listener-related configuration
type ListenerConfig struct {
	Name        string           `yaml:"name"`
	Type        string           `yaml:"type"`
	BindAddress string           `yaml:"bind_address"`
	Port        int              `yaml:"port"`
	UseSSL      bool             `yaml:"use_ssl"`
	URIPath     string           `yaml:"uri_path"`
	SSL         SSLConfig        `yaml:"ssl,omitempty"`
	Encryption  EncryptionConfig `yaml:"encryption"`
}

// SSLConfig represents the SSL-related configuration
type SSLConfig struct {
	Cert string `yaml:"cert"`
	Key  string `yaml:"key"`
}

// EncryptionConfig represents the encryption-related configuration
type EncryptionConfig struct {
	Type string `yaml:"type"`
	Key  string `yaml:"key"`
}

// GeneratorConfig represents the payload generation-related configuration
type GeneratorConfig struct {
	UserAgent    string `yaml:"user_agent"`
	InitialSleep int    `yaml:"initial_sleep"`
	Jitter       int    `yaml:"jitter"`
	Obfuscation  bool   `yaml:"obfuscation"`
	AntiAV       bool   `yaml:"anti_av"`
	AntiETW      bool   `yaml:"anti_etw"`
	SelfDelete   bool   `yaml:"self_delete"`
}

// MCPConfig represents the MCP server-related configuration
type MCPConfig struct {
	Enabled           bool   `yaml:"enabled"`
	SSEEnabled        bool   `yaml:"sse_enabled"`
	SSEPort           string `yaml:"sse_port"`
	SSEBasePath       string `yaml:"sse_base_path"`
	StdioEnabled      bool   `yaml:"stdio_enabled"`
	RemoteEnabled     bool   `yaml:"remote_enabled"`
	RemoteBaseURL     string `yaml:"remote_base_url"`
	StreamableEnabled bool   `yaml:"streamable_enabled"`
	StreamablePort    string `yaml:"streamable_port"`
}

// LoadConfig loads the configuration from a file
func LoadConfig(filePath string) (*Config, error) {
	// Read the configuration file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("could not read config file: %w", err)
	}

	// Parse the YAML file
	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("could not parse config file: %w", err)
	}

	// Validate the configuration
	if err := validateConfig(&config); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return &config, nil
}

// validateConfig validates the configuration
func validateConfig(config *Config) error {
	// Minimum validation
	if config.Database.Path == "" {
		return fmt.Errorf("database path is required")
	}

	if len(config.Listeners) == 0 {
		return fmt.Errorf("at least one listener is required")
	}

	for i, listener := range config.Listeners {
		if listener.Name == "" {
			return fmt.Errorf("listener[%d]: name is required", i)
		}
		if listener.Port <= 0 || listener.Port > 65535 {
			return fmt.Errorf("listener[%d]: invalid port number: %d", i, listener.Port)
		}
		if listener.UseSSL && (listener.SSL.Cert == "" || listener.SSL.Key == "") {
			return fmt.Errorf("listener[%d]: SSL enabled but missing cert/key", i)
		}
	}

	return nil
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Host:           "0.0.0.0",
			AdminPort:      8443,
			SessionTimeout: 30 * time.Minute,
			LogLevel:       "info",
			LogPath:        "logs/server.log",
		},
		Database: DatabaseConfig{
			Path: "data/virga.db",
			Type: "sqlite3",
		},
		Listeners: []ListenerConfig{
			{
				Name:        "default-http",
				Type:        "http",
				BindAddress: "0.0.0.0",
				Port:        8080,
				UseSSL:      false,
				URIPath:     "api/updates",
				Encryption: EncryptionConfig{
					Type: "aes-256",
					Key:  "default-placeholder-key-change-in-production",
				},
			},
		},
		Generator: GeneratorConfig{
			UserAgent:    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/96.0.4664.110 Safari/537.36",
			InitialSleep: 60,
			Jitter:       20,
			Obfuscation:  true,
			AntiAV:       true,
			AntiETW:      true,
			SelfDelete:   false,
		},
		MCP: MCPConfig{
			Enabled:           true,
			SSEEnabled:        true,
			SSEPort:           ":8444",
			SSEBasePath:       "/mcp",
			StdioEnabled:      false,
			RemoteEnabled:     true,
			RemoteBaseURL:     "http://localhost:8444",
			StreamableEnabled: true,
			StreamablePort:    ":50012",
		},
	}
}
