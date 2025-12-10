//go:build llama_embed || llama_external || llama_selfextract
// +build llama_embed llama_external llama_selfextract

package llama

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// PayloadCommandInfo represents the command information for a payload
type PayloadCommandInfo struct {
	Name        string              `json:"name"`
	Type        string              `json:"type"`
	Description string              `json:"description"`
	Commands    []CommandDefinition `json:"commands"`
}

// CommandDefinition defines a single command within a payload
type CommandDefinition struct {
	Name        string                `json:"name"`
	Description string                `json:"description"`
	Parameters  []ParameterDefinition `json:"parameters,omitempty"`
	Examples    []CommandExample      `json:"examples"`
}

// ParameterDefinition defines a parameter for a command
type ParameterDefinition struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Required    bool   `json:"required"`
	Description string `json:"description"`
}

// CommandExample provides usage examples for a command
type CommandExample struct {
	Command     string `json:"command"`
	Description string `json:"description"`
}

// LoadPayloadCommandInfo loads payload command information from a JSON file
func LoadPayloadCommandInfo(path string) (*PayloadCommandInfo, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read payload commands file: %w", err)
	}

	var info PayloadCommandInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, fmt.Errorf("failed to parse payload commands JSON: %w", err)
	}

	return &info, nil
}

// GeneratePayloadPrompt generates a minimal prompt section for the payload commands
// Only includes payload name, description, and command names list
func (p *PayloadCommandInfo) GeneratePayloadPrompt() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("\n=== %s Commands ===\n", p.Name))
	sb.WriteString(fmt.Sprintf("%s\n\n", p.Description))

	// Important note about direct usage
	sb.WriteString(fmt.Sprintf("IMPORTANT: %s is already loaded. DO NOT use 'Import-Module %s'. Use commands directly.\n\n", p.Name, p.Name))

	// Generate comma-separated list of command names only
	sb.WriteString("Available commands: ")
	commandNames := make([]string, len(p.Commands))
	for i, cmd := range p.Commands {
		commandNames[i] = cmd.Name
	}
	sb.WriteString(strings.Join(commandNames, ", "))
	sb.WriteString("\n")

	return sb.String()
}

// GenerateMultiplePayloadPrompt generates a minimal combined prompt for multiple payloads
func GenerateMultiplePayloadPrompt(payloadInfos []*PayloadCommandInfo) string {
	if len(payloadInfos) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("\n=== Available Payloads ===\n")

	for _, info := range payloadInfos {
		sb.WriteString(info.GeneratePayloadPrompt())
	}

	sb.WriteString("\nNote: Use these payload commands when appropriate for the task.\n\n")

	return sb.String()
}
