package completion

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ConfigCompleter uses YAML configuration for command completion
type ConfigCompleter struct {
	config            *CommandConfig
	sessionIDProvider func() []string
	providers         map[string]CompletionProvider
}

// CompletionProvider provides completions for a specific type
type CompletionProvider interface {
	GetCompletions(prefix string) []string
}

// NewConfigCompleter creates a new configuration-based completer
func NewConfigCompleter() (*ConfigCompleter, error) {
	// Load embedded config
	config, err := LoadCommandConfig()
	if err != nil {
		return nil, fmt.Errorf("load command config: %w", err)
	}

	cc := &ConfigCompleter{
		config:    config,
		providers: make(map[string]CompletionProvider),
	}

	// Initialize built-in providers
	cc.initializeProviders()

	return cc, nil
}

// SetSessionIDProvider sets the function to get available session IDs
func (cc *ConfigCompleter) SetSessionIDProvider(provider func() []string) {
	cc.sessionIDProvider = provider
	// Update the session_list provider
	if sp, ok := cc.providers["session_list"].(*sessionProvider); ok {
		sp.getSessionIDs = provider
	}
}

// Complete provides command completions based on the current line
func (cc *ConfigCompleter) Complete(line string, pos int) (completions []string, startPos int) {
	if pos > len(line) {
		pos = len(line)
	}
	currentLine := line[:pos]

	// Parse the command line
	parts := splitCommandLine(currentLine)
	if len(parts) == 0 {
		// Return all top-level commands
		for cmd := range cc.config.Commands {
			completions = append(completions, cmd)
		}
		// Add aliases
		for _, cmdDef := range cc.config.Commands {
			completions = append(completions, cmdDef.Aliases...)
		}
		return completions, 0
	}

	// Check if line ends with space
	endsWithSpace := len(currentLine) > 0 && currentLine[len(currentLine)-1] == ' '

	// Get the current part being typed
	currentPart := ""
	if !endsWithSpace && len(parts) > 0 {
		currentPart = parts[len(parts)-1]
	}

	// First, try to complete command names
	if len(parts) == 1 && !endsWithSpace {
		return cc.completeCommand(currentPart), len(currentLine) - len(currentPart)
	}

	// Find the command definition
	cmdDef, cmdParts := cc.findCommand(parts)
	if cmdDef == nil {
		return nil, 0
	}

	// Calculate position in command (after main command and subcommands)
	argIndex := len(parts) - len(cmdParts) - 1
	if endsWithSpace {
		argIndex++
	}

	// Complete subcommands
	if len(cmdDef.Subcommands) > 0 && argIndex == 0 {
		prefix := strings.ToLower(currentPart)
		for sub := range cmdDef.Subcommands {
			if strings.HasPrefix(strings.ToLower(sub), prefix) {
				completions = append(completions, sub)
			}
		}
		return completions, len(currentLine) - len(currentPart)
	}

	// Complete arguments
	if argIndex >= 0 && argIndex < len(cmdDef.Args) {
		arg := cmdDef.Args[argIndex]
		return cc.completeArgument(arg, currentPart), len(currentLine) - len(currentPart)
	}

	// Complete from static completions if defined
	if len(cmdDef.Completions) > 0 {
		prefix := strings.ToLower(currentPart)
		for _, comp := range cmdDef.Completions {
			if strings.HasPrefix(strings.ToLower(comp), prefix) {
				completions = append(completions, comp)
			}
		}
		return completions, len(currentLine) - len(currentPart)
	}

	return nil, 0
}

// completeCommand completes command names
func (cc *ConfigCompleter) completeCommand(prefix string) []string {
	var completions []string
	lowerPrefix := strings.ToLower(prefix)

	// Check main commands
	for cmd := range cc.config.Commands {
		if strings.HasPrefix(strings.ToLower(cmd), lowerPrefix) {
			completions = append(completions, cmd)
		}
	}

	// Check aliases
	for cmd, cmdDef := range cc.config.Commands {
		for _, alias := range cmdDef.Aliases {
			if strings.HasPrefix(strings.ToLower(alias), lowerPrefix) {
				// Return the main command, not the alias
				completions = append(completions, cmd)
			}
		}
	}

	return completions
}

// findCommand finds the command definition for the given parts
func (cc *ConfigCompleter) findCommand(parts []string) (*CommandDef, []string) {
	if len(parts) == 0 {
		return nil, nil
	}

	// Look up main command (or alias)
	mainCmd := parts[0]
	cmdDef, ok := cc.config.Commands[mainCmd]
	if !ok {
		// Check aliases
		for cmd, def := range cc.config.Commands {
			for _, alias := range def.Aliases {
				if alias == mainCmd {
					cmdDef = def
					mainCmd = cmd
					break
				}
			}
			if ok {
				break
			}
		}
		if !ok {
			return nil, nil
		}
	}

	// Follow subcommands
	cmdParts := []string{mainCmd}
	for i := 1; i < len(parts); i++ {
		if subDef, ok := cmdDef.Subcommands[parts[i]]; ok {
			cmdDef = subDef
			cmdParts = append(cmdParts, parts[i])
		} else {
			// No more subcommands
			break
		}
	}

	return &cmdDef, cmdParts
}

// completeArgument completes an argument based on its type
func (cc *ConfigCompleter) completeArgument(arg ArgDef, prefix string) []string {
	// Handle enum types
	if arg.Type == "enum" && len(arg.Values) > 0 {
		var completions []string
		lowerPrefix := strings.ToLower(prefix)
		for _, val := range arg.Values {
			if strings.HasPrefix(strings.ToLower(val), lowerPrefix) {
				completions = append(completions, val)
			}
		}
		return completions
	}

	// Use type-specific provider
	if typeDef, ok := cc.config.CompletionTypes[arg.Type]; ok {
		if provider, ok := cc.providers[typeDef.Provider]; ok {
			return provider.GetCompletions(prefix)
		}
	}

	return nil
}

// initializeProviders sets up built-in completion providers
func (cc *ConfigCompleter) initializeProviders() {
	// File system provider
	cc.providers["file_system"] = &fileSystemProvider{}

	// Session list provider
	cc.providers["session_list"] = &sessionProvider{
		getSessionIDs: cc.sessionIDProvider,
	}

	// Static provider (returns predefined values)
	cc.providers["static"] = &staticProvider{}
}

// fileSystemProvider provides file path completions
type fileSystemProvider struct{}

func (p *fileSystemProvider) GetCompletions(prefix string) []string {
	// Handle home directory expansion
	if strings.HasPrefix(prefix, "~/") {
		home, err := os.UserHomeDir()
		if err == nil {
			prefix = filepath.Join(home, prefix[2:])
		}
	}

	// Split into directory and file parts
	dir := filepath.Dir(prefix)
	filePrefix := filepath.Base(prefix)

	// If prefix ends with separator, we're completing in that directory
	if strings.HasSuffix(prefix, string(filepath.Separator)) {
		dir = prefix
		filePrefix = ""
	}

	// Read directory contents
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}

	var completions []string
	for _, entry := range entries {
		name := entry.Name()

		// Skip hidden files unless prefix starts with .
		if strings.HasPrefix(name, ".") && !strings.HasPrefix(filePrefix, ".") {
			continue
		}

		// Check if name matches prefix
		if strings.HasPrefix(strings.ToLower(name), strings.ToLower(filePrefix)) {
			fullPath := filepath.Join(dir, name)

			// Add separator for directories
			if entry.IsDir() {
				fullPath += string(filepath.Separator)
			}

			// Make path relative if original was relative
			if !filepath.IsAbs(prefix) {
				relPath, err := filepath.Rel(".", fullPath)
				if err == nil {
					fullPath = relPath
				}
			}

			completions = append(completions, fullPath)
		}
	}

	return completions
}

// sessionProvider provides session ID completions
type sessionProvider struct {
	getSessionIDs func() []string
}

func (p *sessionProvider) GetCompletions(prefix string) []string {
	if p.getSessionIDs == nil {
		return nil
	}

	var completions []string
	lowerPrefix := strings.ToLower(prefix)
	for _, id := range p.getSessionIDs() {
		if strings.HasPrefix(strings.ToLower(id), lowerPrefix) {
			completions = append(completions, id)
		}
	}
	return completions
}

// staticProvider returns predefined values
type staticProvider struct{}

func (p *staticProvider) GetCompletions(prefix string) []string {
	// This would be used for enum types with predefined values
	return nil
}
