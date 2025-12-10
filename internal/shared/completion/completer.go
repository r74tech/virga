package completion

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Completer is the interface for command completion
type Completer interface {
	Complete(line string, pos int) (completions []string, startPos int)
}

// CompletionItem represents a completion item with priority
type CompletionItem struct {
	Text     string
	Priority int
}

// SortOrder represents the sort order type
type SortOrder string

const (
	SortByRank         SortOrder = "rank"
	SortByAlphabetical SortOrder = "alphabetical"
)

// CommandCompleter handles command completion
type CommandCompleter struct {
	// Available commands and their subcommands
	commands map[string][]string
	// Command priorities
	priorities map[string]int
	// Session IDs for interact command
	sessionIDProvider func() []string
	// File path completer
	fileCompleter *FileCompleter
	// Sort order for completions
	sortOrder SortOrder
}

// NewCommandCompleter creates a new command completer
// This now loads commands from the YAML configuration file
func NewCommandCompleter() *CommandCompleter {
	cc := &CommandCompleter{
		commands:      make(map[string][]string),
		priorities:    make(map[string]int),
		fileCompleter: NewFileCompleter(),
		sortOrder:     SortByRank, // Default to priority-based sorting
	}

	// Try to load commands from YAML config
	if err := cc.loadCommandsFromConfig(); err != nil {
		// Fallback to hardcoded commands if config fails
		cc.initializeCommands()
	}

	return cc
}

// SetSortOrder sets the sort order for completions
func (cc *CommandCompleter) SetSortOrder(order SortOrder) {
	cc.sortOrder = order
}

// SetSessionIDProvider sets the function to get available session IDs
func (cc *CommandCompleter) SetSessionIDProvider(provider func() []string) {
	cc.sessionIDProvider = provider
}

// loadCommandsFromConfig loads commands from the embedded YAML configuration
func (cc *CommandCompleter) loadCommandsFromConfig() error {
	// Load embedded config
	config, err := LoadCommandConfig()
	if err != nil {
		return err
	}

	// Convert config to simple command map
	for cmdName, cmdDef := range config.Commands {
		// Get subcommand names
		var subcommands []string
		for subName := range cmdDef.Subcommands {
			subcommands = append(subcommands, subName)
		}
		cc.commands[cmdName] = subcommands

		// Set priority
		if cmdDef.Priority > 0 {
			cc.priorities[cmdName] = cmdDef.Priority
		} else {
			cc.priorities[cmdName] = 50 // Default priority
		}

		// Also add aliases as top-level commands
		for _, alias := range cmdDef.Aliases {
			cc.commands[alias] = subcommands
			cc.priorities[alias] = cmdDef.Priority // Aliases get same priority
		}
	}

	return nil
}

// initializeCommands sets up hardcoded commands as a fallback
// This is only used if the YAML configuration file cannot be loaded
func (cc *CommandCompleter) initializeCommands() {
	// Basic commands - High priority
	cc.commands["help"] = []string{}
	cc.priorities["help"] = 100
	cc.commands["exit"] = []string{}
	cc.priorities["exit"] = 90
	cc.commands["quit"] = []string{}
	cc.priorities["quit"] = 90
	cc.commands["clear"] = []string{}
	cc.priorities["clear"] = 80
	cc.commands["cls"] = []string{}
	cc.priorities["cls"] = 80
	cc.commands["history"] = []string{}
	cc.priorities["history"] = 85

	// Session management - Very high priority
	cc.commands["sessions"] = []string{"list", "kill"}
	cc.priorities["sessions"] = 95
	cc.commands["interact"] = []string{} // Will be populated with session IDs
	cc.priorities["interact"] = 95

	// Beacon and listener management
	cc.commands["beacons"] = []string{"list", "set-sleep", "set-jitter"}
	cc.priorities["beacons"] = 70
	cc.commands["listeners"] = []string{"list", "add", "remove"}
	cc.priorities["listeners"] = 70

	// Generation
	cc.commands["generate"] = []string{"exe", "dll", "shellcode", "powershell"}
	cc.priorities["generate"] = 75

	// File operations (session-specific) - High priority for common ops
	cc.commands["upload"] = []string{} // Will use file path completion
	cc.priorities["upload"] = 85
	cc.commands["download"] = []string{}
	cc.priorities["download"] = 85
	cc.commands["ls"] = []string{}
	cc.priorities["ls"] = 90
	cc.commands["cd"] = []string{} // Will use directory completion
	cc.priorities["cd"] = 90
	cc.commands["pwd"] = []string{}
	cc.priorities["pwd"] = 85

	// Process management
	cc.commands["ps"] = []string{}
	cc.priorities["ps"] = 80
	cc.commands["kill"] = []string{}
	cc.priorities["kill"] = 75

	// Network
	cc.commands["netstat"] = []string{}
	cc.priorities["netstat"] = 70
	cc.commands["portfwd"] = []string{"add", "list", "remove"}
	cc.priorities["portfwd"] = 65

	// System info
	cc.commands["info"] = []string{}
	cc.priorities["info"] = 75
	cc.commands["sysinfo"] = []string{}
	cc.priorities["sysinfo"] = 75
	cc.commands["netinfo"] = []string{}
	cc.priorities["netinfo"] = 70

	// Execution - Very high priority
	cc.commands["shell"] = []string{}
	cc.priorities["shell"] = 92
	cc.commands["exec"] = []string{}
	cc.priorities["exec"] = 92

	// AI features
	cc.commands["llama"] = []string{"status", "cancel"}
	cc.priorities["llama"] = 60
	cc.commands["memdb"] = []string{"SELECT", "FROM", "WHERE", "LIMIT", "ORDER BY"}
	cc.priorities["memdb"] = 60

	// Configuration
	cc.commands["log"] = []string{"status", "level", "enable", "disable"}
	cc.priorities["log"] = 65

	// Extension management
	cc.commands["use"] = []string{"list", "execute", "unload"}
	cc.priorities["use"] = 70

	// Workflow
	cc.commands["workflow"] = []string{"list", "run"}
	cc.priorities["workflow"] = 65
}

// Complete provides command completions based on the current line
func (cc *CommandCompleter) Complete(line string, pos int) (completions []string, startPos int) {
	// Get the line up to the cursor position
	if pos > len(line) {
		pos = len(line)
	}
	currentLine := line[:pos]

	// Split the line into parts
	parts := splitCommandLine(currentLine)

	// If empty line, return all top-level commands
	if len(parts) == 0 {
		for cmd := range cc.commands {
			completions = append(completions, cmd)
		}
		// Sort completions before returning
		completions = cc.sortCompletions(completions)
		return completions, 0
	}

	// Check if line ends with space (completing new argument)
	endsWithSpace := len(currentLine) > 0 && currentLine[len(currentLine)-1] == ' '

	// Get the current part being typed
	currentPart := ""
	if !endsWithSpace && len(parts) > 0 {
		currentPart = parts[len(parts)-1]
	}

	// Calculate start position for replacement
	startPos = len(currentLine) - len(currentPart)

	// First part - complete command names (only if not ending with space)
	if len(parts) == 1 && !endsWithSpace {
		prefix := strings.ToLower(currentPart)
		for cmd := range cc.commands {
			if strings.HasPrefix(cmd, prefix) {
				completions = append(completions, cmd)
			}
		}
		// Sort completions before returning
		completions = cc.sortCompletions(completions)
		return completions, startPos
	}

	// Get the main command
	mainCmd := parts[0]

	// Handle special cases for commands that need dynamic completion
	switch mainCmd {
	case "interact":
		// Complete with session IDs
		if cc.sessionIDProvider != nil && len(parts) == 2 {
			sessionIDs := cc.sessionIDProvider()
			prefix := strings.ToLower(currentPart)
			for _, id := range sessionIDs {
				if strings.HasPrefix(strings.ToLower(id), prefix) {
					completions = append(completions, id)
				}
			}
		}

	case "upload", "download", "cd", "use":
		// Use file path completion
		// Check if we should complete file paths (either we have 2+ parts, or command ends with space)
		if len(parts) >= 2 || (len(parts) == 1 && endsWithSpace) {
			// Get the last part as the path to complete
			pathToComplete := currentPart

			// Handle quoted paths
			if strings.HasPrefix(pathToComplete, "\"") {
				pathToComplete = strings.TrimPrefix(pathToComplete, "\"")
			}

			fileCompletions, fileStartPos := cc.fileCompleter.Complete(pathToComplete, len(pathToComplete))

			// Adjust start position
			startPos = len(currentLine) - len(currentPart) + fileStartPos

			// Add quotes if path contains spaces
			for _, fc := range fileCompletions {
				if strings.Contains(fc, " ") {
					completions = append(completions, "\""+fc+"\"")
				} else {
					completions = append(completions, fc)
				}
			}
		}

	case "kill":
		// Could complete with PIDs if we have a PID provider
		// For now, no completion

	case "portfwd":
		// Complete subcommands
		if len(parts) == 2 {
			subcommands := cc.commands[mainCmd]
			prefix := strings.ToLower(currentPart)
			for _, sub := range subcommands {
				if strings.HasPrefix(sub, prefix) {
					completions = append(completions, sub)
				}
			}
		}

	case "log":
		// Complete subcommands and levels
		if len(parts) == 2 {
			subcommands := cc.commands[mainCmd]
			prefix := strings.ToLower(currentPart)
			for _, sub := range subcommands {
				if strings.HasPrefix(sub, prefix) {
					completions = append(completions, sub)
				}
			}
		} else if len(parts) == 3 && parts[1] == "level" {
			// Complete log levels
			levels := []string{"debug", "info", "warn", "error", "off"}
			prefix := strings.ToLower(currentPart)
			for _, level := range levels {
				if strings.HasPrefix(level, prefix) {
					completions = append(completions, level)
				}
			}
		}

	default:
		// For other commands, complete with subcommands if available
		if subcommands, ok := cc.commands[mainCmd]; ok && len(subcommands) > 0 && len(parts) == 2 {
			prefix := strings.ToLower(currentPart)
			for _, sub := range subcommands {
				if strings.HasPrefix(strings.ToLower(sub), prefix) {
					completions = append(completions, sub)
				}
			}
		}
	}

	// Sort completions before returning
	completions = cc.sortCompletions(completions)
	return completions, startPos
}

// FileCompleter handles file path completion
type FileCompleter struct{}

// NewFileCompleter creates a new file completer
func NewFileCompleter() *FileCompleter {
	return &FileCompleter{}
}

// Complete provides file path completions
func (fc *FileCompleter) Complete(path string, pos int) (completions []string, startPos int) {
	// Handle home directory expansion
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err == nil {
			path = filepath.Join(home, path[2:])
		}
	}

	// Split into directory and file parts
	dir := filepath.Dir(path)
	prefix := filepath.Base(path)

	// If path ends with separator, we're completing in that directory
	if strings.HasSuffix(path, string(filepath.Separator)) {
		dir = path
		prefix = ""
	}

	// Read directory contents
	entries, err := os.ReadDir(dir)
	if err != nil {
		// If directory doesn't exist, try parent directory
		if os.IsNotExist(err) && dir != "." {
			parentDir := filepath.Dir(dir)
			parentPrefix := filepath.Base(dir)
			entries, err = os.ReadDir(parentDir)
			if err == nil {
				dir = parentDir
				prefix = parentPrefix
			}
		}

		if err != nil {
			return nil, 0
		}
	}

	// Find matching entries
	for _, entry := range entries {
		name := entry.Name()

		// Skip hidden files unless prefix starts with .
		if strings.HasPrefix(name, ".") && !strings.HasPrefix(prefix, ".") {
			continue
		}

		// Check if name matches prefix
		if strings.HasPrefix(strings.ToLower(name), strings.ToLower(prefix)) {
			fullPath := filepath.Join(dir, name)

			// Add separator for directories
			if entry.IsDir() {
				fullPath += string(filepath.Separator)
			}

			// Make path relative if original was relative
			if !filepath.IsAbs(path) {
				relPath, err := filepath.Rel(".", fullPath)
				if err == nil {
					fullPath = relPath
				}
			}

			completions = append(completions, fullPath)
		}
	}

	// Calculate start position
	startPos = len(path) - len(prefix)

	return completions, startPos
}

// splitCommandLine splits a command line into parts, respecting quotes
func splitCommandLine(line string) []string {
	var parts []string
	var current strings.Builder
	inQuote := false
	quoteChar := rune(0)

	for _, r := range line {
		switch {
		case r == '"' || r == '\'':
			if inQuote && r == quoteChar {
				inQuote = false
				quoteChar = 0
			} else if !inQuote {
				inQuote = true
				quoteChar = r
			} else {
				current.WriteRune(r)
			}
		case r == ' ' && !inQuote:
			if current.Len() > 0 {
				parts = append(parts, current.String())
				current.Reset()
			}
		default:
			current.WriteRune(r)
		}
	}

	if current.Len() > 0 {
		parts = append(parts, current.String())
	}

	return parts
}

// sortCompletions sorts the completions based on the current sort order
func (cc *CommandCompleter) sortCompletions(completions []string) []string {
	// Create completion items with priorities
	items := make([]CompletionItem, len(completions))
	for i, completion := range completions {
		priority := cc.priorities[completion]
		if priority == 0 {
			// Default priority for items not in the priority map
			priority = 50
		}
		items[i] = CompletionItem{
			Text:     completion,
			Priority: priority,
		}
	}

	// Sort based on the selected order
	switch cc.sortOrder {
	case SortByRank:
		// Sort by priority (descending) then alphabetically
		sort.Slice(items, func(i, j int) bool {
			if items[i].Priority != items[j].Priority {
				return items[i].Priority > items[j].Priority
			}
			return items[i].Text < items[j].Text
		})
	case SortByAlphabetical:
		// Sort alphabetically only
		sort.Slice(items, func(i, j int) bool {
			return items[i].Text < items[j].Text
		})
	}

	// Extract the sorted text
	sorted := make([]string, len(items))
	for i, item := range items {
		sorted[i] = item.Text
	}

	return sorted
}
