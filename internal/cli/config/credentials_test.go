package config

import (
	"os"
	"testing"
)

func TestMemoryCredentialStore(t *testing.T) {
	store := NewMemoryCredentialStore()

	t.Run("save and retrieve password", func(t *testing.T) {
		// Test SetPassword
		err := store.SetPassword("test-service", "test-account", "test-password")
		if err != nil {
			t.Fatalf("SetPassword() error = %v", err)
		}

		// Test GetPassword
		password, err := store.GetPassword("test-service", "test-account")
		if err != nil {
			t.Fatalf("GetPassword() error = %v", err)
		}

		if password != "test-password" {
			t.Errorf("GetPassword() = %v, want %v", password, "test-password")
		}

		// Test DeletePassword
		err = store.DeletePassword("test-service", "test-account")
		if err != nil {
			t.Fatalf("DeletePassword() error = %v", err)
		}

		// Verify deletion
		_, err = store.GetPassword("test-service", "test-account")
		if err == nil {
			t.Error("GetPassword() after DeletePassword() should return error")
		}
	})

	t.Run("get non-existent password", func(t *testing.T) {
		_, err := store.GetPassword("non-existent", "account")
		if err == nil {
			t.Error("GetPassword() should return error for non-existent credential")
		}
	})
}

func TestEnvCredentialStore(t *testing.T) {
	store := NewEnvCredentialStore("Virga")

	t.Run("get password from environment", func(t *testing.T) {
		// Set up test environment
		os.Setenv("VIRGA_SERVER_ADMIN_PASSWORD", "env-password")
		defer os.Unsetenv("VIRGA_SERVER_ADMIN_PASSWORD")

		password, err := store.GetPassword("SERVER", "ADMIN_PASSWORD")
		if err != nil {
			t.Fatalf("GetPassword() error = %v", err)
		}

		if password != "env-password" {
			t.Errorf("GetPassword() = %v, want %v", password, "env-password")
		}
	})

	t.Run("get non-existent password", func(t *testing.T) {
		_, err := store.GetPassword("UNKNOWN", "SERVICE")
		if err == nil {
			t.Error("GetPassword() should return error for non-existent environment variable")
		}
	})

	t.Run("set password not supported", func(t *testing.T) {
		err := store.SetPassword("SERVICE", "ACCOUNT", "password")
		if err == nil {
			t.Error("SetPassword() should return error for environment store")
		}
	})

	t.Run("delete password not supported", func(t *testing.T) {
		err := store.DeletePassword("SERVICE", "ACCOUNT")
		if err == nil {
			t.Error("DeletePassword() should return error for environment store")
		}
	})
}

func TestGetCredentials(t *testing.T) {
	t.Run("get password from environment", func(t *testing.T) {
		// Set up test environment
		os.Setenv("VIRGA_SERVER_TESTUSER", "env-password")
		defer os.Unsetenv("VIRGA_SERVER_TESTUSER")

		config := &Config{
			Server: ServerConfig{
				Host:     "test.example.com",
				Username: "testuser",
			},
		}

		err := GetCredentials(config)
		if err != nil {
			t.Fatalf("GetCredentials() error = %v", err)
		}

		if config.Server.Password != "env-password" {
			t.Errorf("GetCredentials() password = %v, want %v", config.Server.Password, "env-password")
		}
	})

	t.Run("get API key from environment", func(t *testing.T) {
		// Set up test environment
		os.Setenv("VIRGA_SERVER_APIKEY", "test-api-key")
		defer os.Unsetenv("VIRGA_SERVER_APIKEY")

		config := &Config{
			Server: ServerConfig{
				Host: "test.example.com",
			},
		}

		err := GetCredentials(config)
		if err != nil {
			t.Fatalf("GetCredentials() error = %v", err)
		}

		if config.Server.APIKey != "test-api-key" {
			t.Errorf("GetCredentials() APIKey = %v, want %v", config.Server.APIKey, "test-api-key")
		}
	})
}

func TestClearSensitiveData(t *testing.T) {
	config := &Config{
		Server: ServerConfig{
			Host:     "test.example.com",
			Username: "testuser",
			Password: "secret-password",
			APIKey:   "secret-api-key",
		},
	}

	ClearSensitiveData(config)

	if config.Server.Password != "" {
		t.Errorf("ClearSensitiveData() password = %v, want empty", config.Server.Password)
	}

	if config.Server.APIKey != "" {
		t.Errorf("ClearSensitiveData() APIKey = %v, want empty", config.Server.APIKey)
	}

	// Ensure other fields are not cleared
	if config.Server.Host != "test.example.com" {
		t.Errorf("ClearSensitiveData() should not clear Host")
	}
}

func TestValidateNoSensitiveData(t *testing.T) {
	tests := []struct {
		name      string
		config    *Config
		wantError bool
	}{
		{
			name: "no sensitive data",
			config: &Config{
				Server: ServerConfig{
					Host:     "test.example.com",
					Username: "testuser",
				},
			},
			wantError: false,
		},
		{
			name: "password in config",
			config: &Config{
				Server: ServerConfig{
					Host:     "test.example.com",
					Username: "testuser",
					Password: "secret",
				},
			},
			wantError: true,
		},
		{
			name: "API key in config",
			config: &Config{
				Server: ServerConfig{
					Host:   "test.example.com",
					APIKey: "secret-key",
				},
			},
			wantError: true,
		},
		{
			name: "both password and API key",
			config: &Config{
				Server: ServerConfig{
					Host:     "test.example.com",
					Username: "testuser",
					Password: "secret",
					APIKey:   "secret-key",
				},
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateNoSensitiveData(tt.config)
			if (err != nil) != tt.wantError {
				t.Errorf("ValidateNoSensitiveData() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestSecureConfigLoader(t *testing.T) {
	t.Run("save without sensitive data", func(t *testing.T) {
		tmpFile := t.TempDir() + "/config.yaml"
		loader := NewSecureConfigLoader(tmpFile)

		config := &Config{
			Server: ServerConfig{
				Host:     "test.example.com",
				Username: "testuser",
				Password: "secret-password", // Should be cleared
				APIKey:   "secret-key",      // Should be cleared
			},
		}

		// Save config
		err := loader.Save(config)
		if err != nil {
			t.Fatalf("Save() error = %v", err)
		}

		// Verify original config is not modified
		if config.Server.Password != "secret-password" {
			t.Error("Save() should not modify the original config")
		}

		// Load saved config and verify sensitive data is cleared
		loadedConfig, err := LoadConfig(tmpFile)
		if err != nil {
			t.Fatalf("LoadConfig() error = %v", err)
		}

		if loadedConfig.Server.Password != "" {
			t.Errorf("Saved config should not contain password")
		}

		if loadedConfig.Server.APIKey != "" {
			t.Errorf("Saved config should not contain API key")
		}
	})
}
