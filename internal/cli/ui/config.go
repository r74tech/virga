package ui

import (
	"os"
	"path/filepath"
	"time"
)

// Config holds the UI configuration
type Config struct {
	EnableColors     bool
	EnableHistory    bool
	HistoryFile      string
	MaxHistorySize   int
	ColorScheme      ColorScheme
	HistoryBatchSize int
	HistoryFlushTime time.Duration
}

// DefaultConfig returns the default UI configuration
func DefaultConfig() *Config {
	configDir, err := getConfigDir()
	if err != nil {
		configDir = filepath.Join(os.TempDir(), "virga")
	}

	return &Config{
		EnableColors:     !shouldDisableColors(),
		EnableHistory:    true,
		HistoryFile:      filepath.Join(configDir, "history"),
		MaxHistorySize:   10000,
		ColorScheme:      DefaultColorScheme,
		HistoryBatchSize: 100,
		HistoryFlushTime: 30 * time.Second,
	}
}

// getConfigDir returns the configuration directory path
func getConfigDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	// Use XDG_CONFIG_HOME if set
	if xdgConfig := os.Getenv("XDG_CONFIG_HOME"); xdgConfig != "" {
		return filepath.Join(xdgConfig, "virga"), nil
	}

	return filepath.Join(homeDir, ".config", "virga"), nil
}
