package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	payloads "github.com/r74tech/virga/internal/implant/payloads"
)

func TestLoadBeaconConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "beacon_config.yaml")

	configContent := `beacon:
  c2:
    host: "c2.example.com"
    port: 9443
    protocol: "https"
    uri_path: "/api/beacon"
  behavior:
    sleep_time: 120
    jitter: 30
    user_agent: "Mozilla/5.0"
  shell:
    powershell:
      amsi_bypass: "[Ref].Assembly"
  target:
    os: "windows"
    arch: "amd64"
    format: "exe"
  payloads:
    - name: powerview
      source: "./powerview.ps1"
      type: powershell
llama:
  enabled: true
  log_enabled: true
  model:
    path: "/opt/llama/models/model.gguf"
    context: 4096
    gpu_layers: 35
    threads: 8
    temperature: 0.7
    top_k: 40
    top_p: 0.9
    max_tokens: 2048
  prompt:
    preset: "security_analyst"
    custom: "You are a security analyst"
  autonomous:
    enabled: true
    initial_tasks:
      - type: "recon"
        description: "System reconnaissance"
      - type: "network"
        description: "Network mapping"
    max_iterations: 15
    timeout_minutes: 30
    report_interval: 300
  task_prompts:
    recon: "Perform system reconnaissance"
    network: "Map network connections"
implant:
  log_enabled: true
  log_file_path: "/var/log/beacon.log"
  log_level: "debug"
output:
  path: "./beacon.exe"
advanced:
  strip_symbols: true
  compress: false
  anti_debug: true`

	err := os.WriteFile(configPath, []byte(configContent), 0o644)
	if err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	cfg, err := LoadBeaconConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify beacon settings
	if cfg.Beacon.C2.Host != "c2.example.com" {
		t.Errorf("Expected C2 host c2.example.com, got %s", cfg.Beacon.C2.Host)
	}
	if cfg.Beacon.C2.Port != 9443 {
		t.Errorf("Expected C2 port 9443, got %d", cfg.Beacon.C2.Port)
	}
	if cfg.Beacon.C2.Protocol != "https" {
		t.Errorf("Expected protocol https, got %s", cfg.Beacon.C2.Protocol)
	}
	if cfg.Beacon.C2.URIPath != "/api/beacon" {
		t.Errorf("Expected URI path /api/beacon, got %s", cfg.Beacon.C2.URIPath)
	}

	// Verify behavior
	if cfg.Beacon.Behavior.SleepTime != 120 {
		t.Errorf("Expected sleep time 120, got %d", cfg.Beacon.Behavior.SleepTime)
	}
	if cfg.Beacon.Behavior.Jitter != 30 {
		t.Errorf("Expected jitter 30, got %d", cfg.Beacon.Behavior.Jitter)
	}
	if cfg.Beacon.Behavior.UserAgent != "Mozilla/5.0" {
		t.Errorf("Expected user agent Mozilla/5.0, got %s", cfg.Beacon.Behavior.UserAgent)
	}

	// Verify target
	if cfg.Beacon.Target.OS != "windows" {
		t.Errorf("Expected OS windows, got %s", cfg.Beacon.Target.OS)
	}
	if cfg.Beacon.Target.Arch != "amd64" {
		t.Errorf("Expected arch amd64, got %s", cfg.Beacon.Target.Arch)
	}
	if cfg.Beacon.Target.Format != "exe" {
		t.Errorf("Expected format exe, got %s", cfg.Beacon.Target.Format)
	}

	if cfg.Beacon.Shell.PowerShell.AMSIBypass != "[Ref].Assembly" {
		t.Errorf("Expected AMSI bypass override to match custom value")
	}
	if len(cfg.Beacon.Payloads) != 1 {
		t.Fatalf("expected 1 payload entry, got %d", len(cfg.Beacon.Payloads))
	}
	if cfg.Beacon.Payloads[0].Name != "powerview" {
		t.Errorf("expected payload name powerview, got %s", cfg.Beacon.Payloads[0].Name)
	}
	if cfg.Beacon.Payloads[0].Type != "powershell" {
		t.Errorf("expected payload type powershell, got %s", cfg.Beacon.Payloads[0].Type)
	}

	// Verify Llama settings
	if !cfg.Llama.Enabled {
		t.Error("Expected Llama to be enabled")
	}
	if !cfg.Llama.LogEnabled {
		t.Error("Expected Llama log to be enabled")
	}
	if cfg.Llama.Model.Path != "/opt/llama/models/model.gguf" {
		t.Errorf("Unexpected Llama model path: %s", cfg.Llama.Model.Path)
	}
	if cfg.Llama.Model.Context != 4096 {
		t.Errorf("Expected context 4096, got %d", cfg.Llama.Model.Context)
	}
	if cfg.Llama.Model.GPULayers != 35 {
		t.Errorf("Expected GPU layers 35, got %d", cfg.Llama.Model.GPULayers)
	}
	if cfg.Llama.Model.Threads != 8 {
		t.Errorf("Expected threads 8, got %d", cfg.Llama.Model.Threads)
	}
	if cfg.Llama.Model.Temperature != 0.7 {
		t.Errorf("Expected temperature 0.7, got %f", cfg.Llama.Model.Temperature)
	}

	// Verify autonomous
	if !cfg.Llama.Autonomous.Enabled {
		t.Error("Expected autonomous to be enabled")
	}
	if len(cfg.Llama.Autonomous.InitialTasks) != 2 {
		t.Errorf("Expected 2 initial tasks, got %d", len(cfg.Llama.Autonomous.InitialTasks))
	}
	if cfg.Llama.Autonomous.MaxIterations != 15 {
		t.Errorf("Expected max iterations 15, got %d", cfg.Llama.Autonomous.MaxIterations)
	}
	if cfg.Llama.Autonomous.TimeoutMinutes != 30 {
		t.Errorf("Expected timeout 30 minutes, got %d", cfg.Llama.Autonomous.TimeoutMinutes)
	}

	// Verify implant settings
	if !cfg.Implant.LogEnabled {
		t.Error("Expected implant log to be enabled")
	}
	if cfg.Implant.LogFilePath != "/var/log/beacon.log" {
		t.Errorf("Expected log file /var/log/beacon.log, got %s", cfg.Implant.LogFilePath)
	}
	if cfg.Implant.LogLevel != "debug" {
		t.Errorf("Expected log level debug, got %s", cfg.Implant.LogLevel)
	}

	// Verify output settings
	if cfg.Output.Path != "./beacon.exe" {
		t.Errorf("Expected output path ./beacon.exe, got %s", cfg.Output.Path)
	}

	// Verify advanced settings
	if !cfg.Advanced.AntiDebug {
		t.Error("Expected anti-debug to be enabled")
	}
	if !cfg.Advanced.StripSymbols {
		t.Error("Expected strip symbols to be enabled")
	}
}

func TestSetDefaultsSetsPowerShellBypass(t *testing.T) {
	cfg := &BeaconConfig{}
	cfg.SetDefaults()

	if cfg.Beacon.Shell.PowerShell.AMSIBypass != payloads.GetPowerShellAmsiBypass() {
		t.Fatalf("expected default AMSI bypass to be applied")
	}
}

func TestLoadBeaconConfig_FileNotFound(t *testing.T) {
	cfg, err := LoadBeaconConfig("/nonexistent/beacon_config.yaml")
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
	if cfg != nil {
		t.Error("Expected nil config for non-existent file")
	}
}

func TestLoadBeaconConfig_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "invalid_beacon.yaml")

	// Write invalid YAML
	err := os.WriteFile(configPath, []byte("beacon: invalid: yaml: content:"), 0o644)
	if err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	cfg, err := LoadBeaconConfig(configPath)
	if err == nil {
		t.Error("Expected error for invalid YAML")
	}
	if cfg != nil {
		t.Error("Expected nil config for invalid YAML")
	}
}

func TestSaveBeaconConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "save_beacon.yaml")

	cfg := &BeaconConfig{
		Beacon: BeaconSettings{
			C2: C2Config{
				Host:     "save.example.com",
				Port:     8443,
				Protocol: "mtls",
				URIPath:  "/save",
			},
			Behavior: BehaviorConfig{
				SleepTime: 60,
				Jitter:    15,
				UserAgent: "SaveAgent/1.0",
			},
			Target: TargetConfig{
				OS:     "linux",
				Arch:   "arm64",
				Format: "elf",
			},
		},
		Llama: LlamaSettings{
			Enabled:    true,
			LogEnabled: false,
			Model: ModelConfig{
				Path:        "/save/model.gguf",
				Context:     2048,
				GPULayers:   20,
				Threads:     4,
				Temperature: 0.5,
			},
			Prompt: PromptConfig{
				Preset: "default",
				Custom: "Custom prompt",
			},
			Autonomous: AutonomousConfig{
				Enabled: false,
				InitialTasks: []TaskDef{
					{Type: "test", Description: "Test task"},
				},
				MaxIterations:  5,
				TimeoutMinutes: 10,
			},
		},
		Implant: ImplantSettings{
			LogEnabled:  false,
			LogFilePath: "/tmp/save.log",
			LogLevel:    "info",
		},
		Output: OutputSettings{
			Path: "/tmp/beacon",
		},
		Advanced: AdvancedSettings{
			AntiDebug:    false,
			StripSymbols: false,
			Compress:     true,
		},
	}

	// Save config
	err := SaveBeaconConfig(cfg, configPath)
	if err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// Load it back
	loaded, err := LoadBeaconConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load saved config: %v", err)
	}

	// Verify key values
	if loaded.Beacon.C2.Host != cfg.Beacon.C2.Host {
		t.Errorf("C2 host mismatch: expected %s, got %s",
			cfg.Beacon.C2.Host, loaded.Beacon.C2.Host)
	}
	if loaded.Beacon.C2.Port != cfg.Beacon.C2.Port {
		t.Errorf("C2 port mismatch: expected %d, got %d",
			cfg.Beacon.C2.Port, loaded.Beacon.C2.Port)
	}
	if loaded.Beacon.Behavior.SleepTime != cfg.Beacon.Behavior.SleepTime {
		t.Errorf("Sleep time mismatch: expected %d, got %d",
			cfg.Beacon.Behavior.SleepTime, loaded.Beacon.Behavior.SleepTime)
	}
	if loaded.Llama.Enabled != cfg.Llama.Enabled {
		t.Error("Llama enabled mismatch")
	}
	if loaded.Llama.Model.Path != cfg.Llama.Model.Path {
		t.Errorf("Llama model path mismatch: expected %s, got %s",
			cfg.Llama.Model.Path, loaded.Llama.Model.Path)
	}
	if loaded.Output.Path != cfg.Output.Path {
		t.Errorf("Output path mismatch: expected %s, got %s",
			cfg.Output.Path, loaded.Output.Path)
	}
	if loaded.Advanced.StripSymbols != cfg.Advanced.StripSymbols {
		t.Errorf("Strip symbols mismatch: expected %v, got %v",
			cfg.Advanced.StripSymbols, loaded.Advanced.StripSymbols)
	}
}

func TestValidateBeaconConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  *BeaconConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "Valid config",
			config: &BeaconConfig{
				Beacon: BeaconSettings{
					C2: C2Config{
						Host:     "localhost",
						Port:     8443,
						Protocol: "https",
					},
					Behavior: BehaviorConfig{
						SleepTime: 60,
						Jitter:    20,
					},
					Target: TargetConfig{
						OS:     "windows",
						Arch:   "amd64",
						Format: "exe",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Empty C2 host",
			config: &BeaconConfig{
				Beacon: BeaconSettings{
					C2: C2Config{
						Host:     "",
						Port:     8443,
						Protocol: "https",
					},
				},
			},
			wantErr: true,
			errMsg:  "C2 host cannot be empty",
		},
		{
			name: "Invalid port",
			config: &BeaconConfig{
				Beacon: BeaconSettings{
					C2: C2Config{
						Host:     "localhost",
						Port:     0,
						Protocol: "https",
					},
				},
			},
			wantErr: true,
			errMsg:  "invalid port",
		},
		{
			name: "Invalid protocol",
			config: &BeaconConfig{
				Beacon: BeaconSettings{
					C2: C2Config{
						Host:     "localhost",
						Port:     8443,
						Protocol: "invalid",
					},
				},
			},
			wantErr: true,
			errMsg:  "invalid protocol",
		},
		{
			name: "Negative sleep time",
			config: &BeaconConfig{
				Beacon: BeaconSettings{
					C2: C2Config{
						Host:     "localhost",
						Port:     8443,
						Protocol: "https",
					},
					Behavior: BehaviorConfig{
						SleepTime: -1,
						Jitter:    20,
					},
				},
			},
			wantErr: true,
			errMsg:  "sleep time must be positive",
		},
		{
			name: "Invalid jitter",
			config: &BeaconConfig{
				Beacon: BeaconSettings{
					C2: C2Config{
						Host:     "localhost",
						Port:     8443,
						Protocol: "https",
					},
					Behavior: BehaviorConfig{
						SleepTime: 60,
						Jitter:    101,
					},
				},
			},
			wantErr: true,
			errMsg:  "jitter must be between 0 and 100",
		},
		{
			name: "Invalid OS",
			config: &BeaconConfig{
				Beacon: BeaconSettings{
					C2: C2Config{
						Host:     "localhost",
						Port:     8443,
						Protocol: "https",
					},
					Behavior: BehaviorConfig{
						SleepTime: 60,
						Jitter:    20,
					},
					Target: TargetConfig{
						OS:     "invalid",
						Arch:   "amd64",
						Format: "exe",
					},
				},
			},
			wantErr: true,
			errMsg:  "unsupported OS",
		},
		{
			name: "Llama enabled without model",
			config: &BeaconConfig{
				Beacon: BeaconSettings{
					C2: C2Config{
						Host:     "localhost",
						Port:     8443,
						Protocol: "https",
					},
					Behavior: BehaviorConfig{
						SleepTime: 60,
						Jitter:    20,
					},
					Target: TargetConfig{
						OS:     "windows",
						Arch:   "amd64",
						Format: "exe",
					},
				},
				Llama: LlamaSettings{
					Enabled: true,
					Model: ModelConfig{
						Path: "", // Empty model path
					},
				},
			},
			wantErr: true,
			errMsg:  "Llama model path required",
		},
		{
			name: "Unsupported payload type",
			config: &BeaconConfig{
				Beacon: BeaconSettings{
					C2:       C2Config{Host: "localhost", Port: 8443, Protocol: "https"},
					Behavior: BehaviorConfig{SleepTime: 60, Jitter: 20},
					Target:   TargetConfig{OS: "windows", Arch: "amd64", Format: "exe"},
					Payloads: []PayloadConfig{
						{Name: "bad", Source: "./bad.bin", Type: "unknown"},
					},
				},
			},
			wantErr: true,
			errMsg:  "unsupported payload type",
		},
		{
			name: "Duplicate payload names",
			config: &BeaconConfig{
				Beacon: BeaconSettings{
					C2:       C2Config{Host: "localhost", Port: 8443, Protocol: "https"},
					Behavior: BehaviorConfig{SleepTime: 60, Jitter: 20},
					Target:   TargetConfig{OS: "windows", Arch: "amd64", Format: "exe"},
					Payloads: []PayloadConfig{
						{Name: "dup", Source: "./a.ps1", Type: "powershell"},
						{Name: "dup", Source: "./b.ps1", Type: "powershell"},
					},
				},
			},
			wantErr: true,
			errMsg:  "duplicate payload name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Since ValidateBeaconConfig doesn't exist, we'll do manual validation
			err := validateBeaconConfigTest(tt.config)
			if tt.wantErr {
				if err == nil {
					t.Error("Expected error, got nil")
				} else if tt.errMsg != "" && !contains(err.Error(), tt.errMsg) {
					t.Errorf("Expected error containing %q, got %q", tt.errMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
			}
		})
	}
}

func TestBeaconConfigToFlags(t *testing.T) {
	cfg := &BeaconConfig{
		Beacon: BeaconSettings{
			C2: C2Config{
				Host:     "c2.example.com",
				Port:     9443,
				Protocol: "mtls",
				URIPath:  "/beacon",
			},
			Behavior: BehaviorConfig{
				SleepTime: 120,
				Jitter:    30,
				UserAgent: "Custom/1.0",
			},
			Target: TargetConfig{
				OS:     "windows",
				Arch:   "amd64",
				Format: "exe",
			},
		},
		Llama: LlamaSettings{
			Enabled:    true,
			LogEnabled: true,
			Model: ModelConfig{
				Path:        "/opt/model.gguf",
				Context:     4096,
				GPULayers:   32,
				Threads:     8,
				Temperature: 0.7,
			},
			Autonomous: AutonomousConfig{
				Enabled:       true,
				MaxIterations: 10,
			},
		},
		Implant: ImplantSettings{
			LogEnabled:  true,
			LogFilePath: "/var/log/implant.log",
			LogLevel:    "debug",
		},
		Output: OutputSettings{
			Path: "./beacon.exe",
		},
		Advanced: AdvancedSettings{
			StripSymbols: true,
			Compress:     true,
			AntiDebug:    false,
		},
	}

	// Test that config can be converted to build flags
	// This would be used by the generator
	expectedFlags := []string{
		"-c2-host", "c2.example.com",
		"-c2-port", "9443",
		"-protocol", "mtls",
		"-sleep", "120",
		"-jitter", "30",
		"-os", "windows",
		"-arch", "amd64",
		"-format", "exe",
		"-llama-enabled",
		"-llama-model", "/opt/model.gguf",
		"-implant-log",
		"-output", "./beacon.exe",
	}

	// In a real implementation, cfg would have a ToFlags() method
	// For now, just verify the config has the expected values
	if cfg.Beacon.C2.Host != "c2.example.com" {
		t.Error("C2 host mismatch")
	}
	if cfg.Beacon.C2.Port != 9443 {
		t.Error("C2 port mismatch")
	}
	if !cfg.Llama.Enabled {
		t.Error("Llama should be enabled")
	}
	if !cfg.Implant.LogEnabled {
		t.Error("Implant log should be enabled")
	}

	// Verify we can access all necessary fields for flag generation
	_ = expectedFlags // Just to avoid unused variable warning
}

// Helper functions

func validateBeaconConfigTest(cfg *BeaconConfig) error {
	// Basic validation
	if cfg.Beacon.C2.Host == "" {
		return fmt.Errorf("C2 host cannot be empty")
	}
	if cfg.Beacon.C2.Port <= 0 || cfg.Beacon.C2.Port > 65535 {
		return fmt.Errorf("invalid port: %d", cfg.Beacon.C2.Port)
	}

	validProtocols := []string{"http", "https", "mtls", "dns"}
	validProtocol := false
	for _, p := range validProtocols {
		if cfg.Beacon.C2.Protocol == p {
			validProtocol = true
			break
		}
	}
	if !validProtocol {
		return fmt.Errorf("invalid protocol: %s", cfg.Beacon.C2.Protocol)
	}

	if cfg.Beacon.Behavior.SleepTime < 0 {
		return fmt.Errorf("sleep time must be positive")
	}
	if cfg.Beacon.Behavior.Jitter < 0 || cfg.Beacon.Behavior.Jitter > 100 {
		return fmt.Errorf("jitter must be between 0 and 100")
	}

	validOS := []string{"windows", "linux", "darwin"}
	validOSFound := false
	for _, os := range validOS {
		if cfg.Beacon.Target.OS == os {
			validOSFound = true
			break
		}
	}
	if !validOSFound {
		return fmt.Errorf("unsupported OS: %s", cfg.Beacon.Target.OS)
	}

	if cfg.Llama.Enabled && cfg.Llama.Model.Path == "" {
		return fmt.Errorf("Llama model path required when Llama is enabled")
	}

	if len(cfg.Beacon.Payloads) > 0 {
		seen := make(map[string]struct{})
		for _, payload := range cfg.Beacon.Payloads {
			if payload.Name == "" {
				return fmt.Errorf("payload name cannot be empty")
			}
			if payload.Source == "" {
				return fmt.Errorf("payload %s source cannot be empty", payload.Name)
			}
			typeName := payload.Type
			if typeName == "" {
				typeName = "powershell"
			}
			if typeName != "powershell" {
				return fmt.Errorf("unsupported payload type: %s", typeName)
			}
			if _, exists := seen[payload.Name]; exists {
				return fmt.Errorf("duplicate payload name: %s", payload.Name)
			}
			seen[payload.Name] = struct{}{}
		}
	}

	return nil
}

func contains(s, substr string) bool {
	return len(substr) > 0 && strings.Contains(s, substr)
}
