package config

import (
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"runtime"
	"strings"
	"syscall"

	"github.com/r74tech/virga/internal/shared/logger"
	"golang.org/x/term"
)

// CredentialStore interface for secure credential storage
type CredentialStore interface {
	GetPassword(service, account string) (string, error)
	SetPassword(service, account, password string) error
	DeletePassword(service, account string) error
}

// MemoryCredentialStore stores credentials in memory only (not persistent)
type MemoryCredentialStore struct {
	creds map[string]string
}

// NewMemoryCredentialStore creates a new in-memory credential store
func NewMemoryCredentialStore() *MemoryCredentialStore {
	return &MemoryCredentialStore{
		creds: make(map[string]string),
	}
}

// GetPassword retrieves a password from memory
func (m *MemoryCredentialStore) GetPassword(service, account string) (string, error) {
	key := fmt.Sprintf("%s:%s", service, account)
	password, ok := m.creds[key]
	if !ok {
		return "", errors.New("credential not found")
	}

	// Decode from base64
	decoded, err := base64.StdEncoding.DecodeString(password)
	if err != nil {
		return "", fmt.Errorf("decode password: %w", err)
	}

	return string(decoded), nil
}

// SetPassword stores a password in memory
func (m *MemoryCredentialStore) SetPassword(service, account, password string) error {
	key := fmt.Sprintf("%s:%s", service, account)
	// Encode to base64 for minimal obfuscation in memory
	encoded := base64.StdEncoding.EncodeToString([]byte(password))
	m.creds[key] = encoded
	return nil
}

// DeletePassword removes a password from memory
func (m *MemoryCredentialStore) DeletePassword(service, account string) error {
	key := fmt.Sprintf("%s:%s", service, account)
	delete(m.creds, key)
	return nil
}

// EnvCredentialStore reads credentials from environment variables
type EnvCredentialStore struct {
	prefix string
}

// NewEnvCredentialStore creates a new environment-based credential store
func NewEnvCredentialStore(prefix string) *EnvCredentialStore {
	return &EnvCredentialStore{
		prefix: prefix,
	}
}

// GetPassword retrieves a password from environment variables
func (e *EnvCredentialStore) GetPassword(service, account string) (string, error) {
	// Convert to environment variable name
	envName := e.makeEnvName(service, account)

	password := os.Getenv(envName)
	if password == "" {
		return "", errors.New("credential not found in environment")
	}

	return password, nil
}

// SetPassword is not supported for environment variables
func (e *EnvCredentialStore) SetPassword(service, account, password string) error {
	return errors.New("cannot set environment variables")
}

// DeletePassword is not supported for environment variables
func (e *EnvCredentialStore) DeletePassword(service, account string) error {
	return errors.New("cannot delete environment variables")
}

// makeEnvName converts service and account to environment variable name
func (e *EnvCredentialStore) makeEnvName(service, account string) string {
	// Example: VIRGA_SERVER_ADMIN_PASSWORD
	name := fmt.Sprintf("%s_%s_%s", e.prefix, service, account)
	name = strings.ToUpper(name)
	name = strings.ReplaceAll(name, "-", "_")
	name = strings.ReplaceAll(name, ".", "_")
	return name
}

// FileCredentialStore stores credentials in an encrypted file
type FileCredentialStore struct {
	filePath string
	key      []byte
}

// PromptForPassword prompts the user for a password securely
func PromptForPassword(prompt string) (string, error) {
	// For password prompts, we need to use fmt.Print directly
	fmt.Print(prompt)

	// Read password without echoing
	var password []byte
	var err error

	if runtime.GOOS == "windows" {
		// Windows-specific implementation would go here
		// For now, fall back to regular input
		logger.Warn("WARNING: Password will be visible on Windows")
		var input string
		if _, err = fmt.Scanln(&input); err != nil {
			return "", fmt.Errorf("read password fallback: %w", err)
		}
		password = []byte(input)
	} else {
		// Unix-like systems
		fd := int(syscall.Stdin)
		password, err = term.ReadPassword(fd)
		fmt.Println() // New line after password input - keep fmt.Println for terminal control
	}

	if err != nil {
		return "", fmt.Errorf("read password: %w", err)
	}

	return string(password), nil
}

// GetCredentials retrieves credentials using multiple strategies
func GetCredentials(config *Config) error {
	// Check environment variables first
	envStore := NewEnvCredentialStore("Virga")

	// Try to get password from environment
	if config.Server.Password == "" {
		if pass, err := envStore.GetPassword("SERVER", config.Server.Username); err == nil {
			config.Server.Password = pass
		}
	}

	// Try to get API key from environment
	if config.Server.APIKey == "" {
		if key, err := envStore.GetPassword("SERVER", "APIKEY"); err == nil {
			config.Server.APIKey = key
		}
	}

	// If still missing, prompt user
	if config.Server.Password == "" && config.Server.Username != "" {
		prompt := fmt.Sprintf("Password for %s@%s: ", config.Server.Username, config.Server.Host)
		password, err := PromptForPassword(prompt)
		if err != nil {
			return fmt.Errorf("prompt for password: %w", err)
		}
		config.Server.Password = password
	}

	return nil
}

// ClearSensitiveData removes sensitive data from config
func ClearSensitiveData(config *Config) {
	config.Server.Password = ""
	config.Server.APIKey = ""
}

// ValidateNoSensitiveData ensures no sensitive data is in the config
func ValidateNoSensitiveData(config *Config) error {
	var errs []error

	if config.Server.Password != "" {
		errs = append(errs, ErrPasswordInConfig)
	}

	if config.Server.APIKey != "" {
		errs = append(errs, ErrAPIKeyInConfig)
	}

	if len(errs) > 0 {
		return fmt.Errorf("sensitive data found in config: %v", errs)
	}

	return nil
}

// SecureConfigLoader loads config and handles credentials securely
type SecureConfigLoader struct {
	configPath string
	credStore  CredentialStore
}

// NewSecureConfigLoader creates a new secure config loader
func NewSecureConfigLoader(configPath string) *SecureConfigLoader {
	return &SecureConfigLoader{
		configPath: configPath,
		credStore:  NewMemoryCredentialStore(),
	}
}

// Load loads config and credentials securely
func (s *SecureConfigLoader) Load() (*Config, error) {
	// Load config file
	config, err := LoadConfig(s.configPath)
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	// Warn if sensitive data is in config
	if err := ValidateNoSensitiveData(config); err != nil {
		logger.Warn(fmt.Sprintf("WARNING: %v", err))
		logger.Warn("Consider using environment variables or prompting instead")
	}

	// Get credentials from environment or prompt
	if err := GetCredentials(config); err != nil {
		return nil, fmt.Errorf("get credentials: %w", err)
	}

	return config, nil
}

// Save saves config without sensitive data
func (s *SecureConfigLoader) Save(config *Config) error {
	// Create a copy to avoid modifying the original
	configCopy := *config

	// Clear sensitive data
	ClearSensitiveData(&configCopy)

	// Save config
	return SaveConfig(&configCopy, s.configPath)
}
