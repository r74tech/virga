package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// ValidationOptions configures path validation behavior
type ValidationOptions struct {
	RootDir         string // Restrict paths to this root directory
	ResolveSymlinks bool   // Whether to resolve symbolic links
	MaxPathLength   int    // Maximum allowed path length (0 = no limit)
	AllowRelative   bool   // Whether to allow relative paths
}

// DefaultValidationOptions returns default validation options
func DefaultValidationOptions() ValidationOptions {
	maxLen := 4096 // Default for most systems
	if runtime.GOOS == "windows" {
		maxLen = 260 // MAX_PATH on Windows
	}

	return ValidationOptions{
		RootDir:         "",
		ResolveSymlinks: true,
		MaxPathLength:   maxLen,
		AllowRelative:   false,
	}
}

// validatePlatformSpecific performs platform-specific path validation
func validatePlatformSpecific(path string) error {
	if runtime.GOOS == "windows" {
		return validateWindowsPath(path)
	}
	return nil
}

// validateWindowsPath validates Windows-specific path requirements
func validateWindowsPath(path string) error {
	// Check for reserved device names
	deviceNames := []string{
		"CON", "PRN", "AUX", "NUL",
		"COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9",
		"LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9",
	}

	// Get the base name without extension
	base := filepath.Base(path)
	if ext := filepath.Ext(base); ext != "" {
		base = base[:len(base)-len(ext)]
	}

	upperBase := strings.ToUpper(base)
	for _, device := range deviceNames {
		if upperBase == device {
			return fmt.Errorf("reserved device name: %s", base)
		}
	}

	// Check for invalid characters
	invalidChars := []rune{'<', '>', ':', '"', '|', '?', '*'}
	for _, char := range invalidChars {
		if strings.ContainsRune(path, char) {
			return fmt.Errorf("invalid character in path: %c", char)
		}
	}

	// Check for trailing spaces or periods
	parts := strings.Split(path, string(filepath.Separator))
	for _, part := range parts {
		if part != "" && (strings.HasSuffix(part, " ") || strings.HasSuffix(part, ".")) {
			return fmt.Errorf("invalid trailing character in path component: %s", part)
		}
	}

	return nil
}

// IsExecutable checks if a file is executable
func IsExecutable(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}

	mode := info.Mode()

	// Unix systems
	if runtime.GOOS != "windows" {
		return mode&0111 != 0
	}

	// Windows - check by extension
	ext := strings.ToLower(filepath.Ext(path))
	executableExts := []string{".exe", ".bat", ".cmd", ".com", ".ps1", ".vbs", ".js"}

	for _, e := range executableExts {
		if ext == e {
			return true
		}
	}

	return false
}
