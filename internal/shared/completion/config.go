package completion

import (
	_ "embed"
	"fmt"

	"gopkg.in/yaml.v3"
)

//go:embed commands.yaml
var commandsYAML []byte

// CommandConfig represents the YAML command configuration
type CommandConfig struct {
	Commands        map[string]CommandDef        `yaml:"commands"`
	CompletionTypes map[string]CompletionTypeDef `yaml:"completion_types"`
}

// CommandDef defines a command
type CommandDef struct {
	Description string                `yaml:"description"`
	Priority    int                   `yaml:"priority,omitempty"`
	Aliases     []string              `yaml:"aliases,omitempty"`
	Subcommands map[string]CommandDef `yaml:"subcommands,omitempty"`
	Args        []ArgDef              `yaml:"args,omitempty"`
	Flags       []FlagDef             `yaml:"flags,omitempty"`
	Completions []string              `yaml:"completions,omitempty"`
}

// ArgDef defines a command argument
type ArgDef struct {
	Name        string   `yaml:"name"`
	Type        string   `yaml:"type"`
	Description string   `yaml:"description"`
	Optional    bool     `yaml:"optional,omitempty"`
	Rest        bool     `yaml:"rest,omitempty"`   // Consumes rest of line
	Values      []string `yaml:"values,omitempty"` // For enum types
}

// FlagDef defines a command flag
type FlagDef struct {
	Name        string `yaml:"name"`
	Type        string `yaml:"type"`
	Description string `yaml:"description"`
}

// CompletionTypeDef defines how to complete a type
type CompletionTypeDef struct {
	Provider    string `yaml:"provider"`
	Filter      string `yaml:"filter,omitempty"`
	Description string `yaml:"description"`
}

// LoadCommandConfig loads command configuration from embedded YAML
func LoadCommandConfig() (*CommandConfig, error) {
	var config CommandConfig
	if err := yaml.Unmarshal(commandsYAML, &config); err != nil {
		return nil, fmt.Errorf("parse YAML: %w", err)
	}

	return &config, nil
}
