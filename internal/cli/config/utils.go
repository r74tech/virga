package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	// File permissions for config files (owner read/write only)
	configFilePerms = 0o600
	// Directory permissions (owner read/write/execute only)
	configDirPerms = 0o700
)

// Common errors
var (
	ErrEmptyHost        = errors.New("server host cannot be empty")
	ErrInvalidPort      = errors.New("port must be between 1 and 65535")
	ErrInvalidLogLevel  = errors.New("invalid log level")
	ErrInvalidURL       = errors.New("invalid URL")
	ErrInvalidProtocol  = errors.New("invalid protocol")
	ErrInvalidJitter    = errors.New("jitter must be between 0 and 100")
	ErrInvalidSleepTime = errors.New("sleep time must be positive")
	ErrPasswordInConfig = errors.New("password should not be stored in config file")
	ErrAPIKeyInConfig   = errors.New("API key should not be stored in config file")
)

// YAMLConfig interface for common config operations
type YAMLConfig interface {
	Validate() error
	SetDefaults()
}

// LoadYAML loads a YAML config file with proper error handling
func LoadYAML(path string, cfg YAMLConfig) error {
	// Expand path
	expandedPath, err := expandPath(path)
	if err != nil {
		return fmt.Errorf("expand path: %w", err)
	}

	// Check if file exists
	info, err := os.Stat(expandedPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Set defaults and create file
			cfg.SetDefaults()
			return SaveYAML(expandedPath, cfg, configFilePerms)
		}
		return fmt.Errorf("stat file: %w", err)
	}

	// Check file permissions (warn if too permissive)
	if info.Mode().Perm()&0o077 != 0 {
		// File is readable by group or others
		fmt.Fprintf(os.Stderr, "WARNING: Config file %s has overly permissive permissions %v\n", path, info.Mode().Perm())
		fmt.Fprintf(os.Stderr, "Consider running: chmod 600 %s\n", path)
	}

	// Read file
	data, err := os.ReadFile(expandedPath)
	if err != nil {
		return fmt.Errorf("read file: %w", err)
	}

	// Unmarshal YAML
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return fmt.Errorf("parse YAML: %w", err)
	}

	// Validate
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("validate config: %w", err)
	}

	return nil
}

// SaveYAML saves a YAML config file with secure permissions
func SaveYAML(path string, cfg YAMLConfig, perm os.FileMode) error {
	// Validate first
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("validate config: %w", err)
	}

	// Expand path
	expandedPath, err := expandPath(path)
	if err != nil {
		return fmt.Errorf("expand path: %w", err)
	}

	// Ensure directory exists with secure permissions
	dir := filepath.Dir(expandedPath)
	if err := ensureDir(dir); err != nil {
		return fmt.Errorf("ensure directory: %w", err)
	}

	// Marshal to YAML
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal YAML: %w", err)
	}

	// Write atomically with secure permissions
	return writeFileAtomic(expandedPath, data, perm)
}

// expandPath expands ~ in paths
func expandPath(path string) (string, error) {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("get home dir: %w", err)
		}
		path = filepath.Join(home, path[2:])
	}
	return filepath.Clean(path), nil
}

// ensureDir creates a directory with secure permissions if it doesn't exist
func ensureDir(dir string) error {
	if dir == "." || dir == "" {
		return nil
	}

	info, err := os.Stat(dir)
	if err == nil {
		if !info.IsDir() {
			return fmt.Errorf("%s exists but is not a directory", dir)
		}
		return nil
	}

	if !os.IsNotExist(err) {
		return fmt.Errorf("stat directory: %w", err)
	}

	// Create directory with secure permissions
	if err := os.MkdirAll(dir, configDirPerms); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}

	return nil
}

// writeFileAtomic writes a file atomically with specified permissions
func writeFileAtomic(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)

	// Create temp file in same directory
	tmp, err := os.CreateTemp(dir, ".tmp-")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpName := tmp.Name()

	// Clean up on error
	defer func() {
		if tmp != nil {
			tmp.Close()
			os.Remove(tmpName)
		}
	}()

	// Write data
	if _, err := tmp.Write(data); err != nil {
		return fmt.Errorf("write data: %w", err)
	}

	// Set permissions
	if err := tmp.Chmod(perm); err != nil {
		return fmt.Errorf("set permissions: %w", err)
	}

	// Sync to disk
	if err := tmp.Sync(); err != nil {
		return fmt.Errorf("sync to disk: %w", err)
	}

	// Close file
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp file: %w", err)
	}
	tmp = nil

	// Atomic rename
	if err := os.Rename(tmpName, path); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("atomic rename: %w", err)
	}

	return nil
}

// ValidatePort checks if a port number is valid
func ValidatePort(port int) error {
	if port < 1 || port > 65535 {
		return ErrInvalidPort
	}
	return nil
}

// ValidateLogLevel checks if a log level is valid
func ValidateLogLevel(level string) error {
	validLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}
	if !validLevels[strings.ToLower(level)] {
		return ErrInvalidLogLevel
	}
	return nil
}

// ValidateURL checks if a URL is valid
func ValidateURL(rawURL string) error {
	if rawURL == "" {
		return ErrInvalidURL
	}

	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidURL, err)
	}

	if u.Scheme == "" || u.Host == "" {
		return ErrInvalidURL
	}

	return nil
}

// ValidateProtocol checks if a protocol is valid
func ValidateProtocol(protocol string) error {
	validProtocols := map[string]bool{
		"http":  true,
		"https": true,
		"mtls":  true,
		"dns":   true,
	}
	if !validProtocols[strings.ToLower(protocol)] {
		return ErrInvalidProtocol
	}
	return nil
}

// SanitizeConfig removes sensitive information from config for display
func SanitizeConfig(cfg interface{}) map[string]interface{} {
	// Convert to map for manipulation
	data, _ := yaml.Marshal(cfg)
	var result map[string]interface{}
	yaml.Unmarshal(data, &result)

	// Recursively sanitize
	sanitizeMap(result)

	return result
}

// sanitizeMap recursively sanitizes sensitive fields in a map
func sanitizeMap(m map[string]interface{}) {
	sensitiveFields := map[string]bool{
		"password": true,
		"api_key":  true,
		"apikey":   true,
		"secret":   true,
		"token":    true,
		"key":      true,
	}

	for k, v := range m {
		// Check if field name suggests sensitive data
		if sensitiveFields[strings.ToLower(k)] {
			if str, ok := v.(string); ok && str != "" {
				m[k] = "<redacted>"
			}
		}

		// Recursively process nested maps
		if nested, ok := v.(map[string]interface{}); ok {
			sanitizeMap(nested)
		}

		// Process slices of maps
		if slice, ok := v.([]interface{}); ok {
			for _, item := range slice {
				if nested, ok := item.(map[string]interface{}); ok {
					sanitizeMap(nested)
				}
			}
		}
	}
}
