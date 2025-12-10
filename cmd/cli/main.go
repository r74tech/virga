package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/r74tech/virga/internal/cli/client"
	"github.com/r74tech/virga/internal/cli/command"
	"github.com/r74tech/virga/internal/cli/config"
	"github.com/r74tech/virga/internal/cli/session"
	"github.com/r74tech/virga/internal/cli/ui"
	"github.com/r74tech/virga/internal/shared/completion"
	"github.com/r74tech/virga/internal/shared/logger"
)

// Version information (set during build)
var (
	version = "1.0.0"
	commit  = "dev"
	date    = "unknown"
)

func main() {
	// Parse command-line flags
	configFile := flag.String("config", "", "Path to configuration file")
	serverHost := flag.String("host", "", "C2 server hostname or IP address")
	serverPort := flag.Int("port", 0, "C2 server port number")
	debug := flag.Bool("debug", false, "Enable debug mode")
	showVersion := flag.Bool("version", false, "Show version information")
	execCmd := flag.String("exec", "", "Execute command and exit")
	execCmdShort := flag.String("e", "", "Execute command and exit (short form)")
	sessionID := flag.String("session", "", "Session ID to use for command execution")
	quiet := flag.Bool("quiet", false, "Minimal output")
	noColor := flag.Bool("no-color", false, "Disable color output")
	logLevel := flag.String("log-level", "", "Set log level (debug, info, warn, error, off)")
	jsonlLog := flag.String("jsonl-log", "", "Enable JSONL logging to specified file")
	jsonlLogLevel := flag.String("jsonl-log-level", "debug", "JSONL log level (debug, info, warn, error)")
	flag.Parse()

	// Show version information
	if *showVersion {
		logger.Info(fmt.Sprintf("Virga CLI %s (%s) built on %s", version, commit, date))
		os.Exit(0)
	}

	// Load configuration
	cfg, err := loadConfig(*configFile)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Override server settings if specified in command-line
	if *serverHost != "" {
		cfg.Server.Host = *serverHost
	}
	if *serverPort > 0 {
		cfg.Server.Port = *serverPort
	}

	// Determine debug mode (CLI flag takes precedence, otherwise from config.yaml)
	debugMode := *debug
	if !debugMode && cfg.General.Debug {
		debugMode = true
	}

	// If log level is set to debug, enable debug mode
	if *logLevel == "debug" {
		debugMode = true
	}

	// Generate API client and set debug mode
	apiClient := client.NewAPIClient(cfg)
	if debugMode {
		apiClient.SetDebugMode(true)
		logger.Debug("Debug mode enabled from %s", func() string {
			if *debug {
				return "CLI flag"
			}
			return "config.yaml"
		}())
	}

	// Generate session manager and set API client
	sessionManager := session.NewManager()
	sessionManager.SetAPIClient(apiClient)
	if debugMode {
		sessionManager.SetDebugMode(true)
	}

	// Set log level early from environment or flag
	logger.SetGlobalLogLevelFromEnv("VIRGA_CLI_LOG_LEVEL")
	if *logLevel != "" {
		setLogLevel(*logLevel)
	}

	// If quiet mode is enabled, set log level to ERROR
	if *quiet {
		logger.SetGlobalLogLevel(logger.ERROR)
	}

	// Setup JSONL logging if enabled (command line flag takes precedence)
	if *jsonlLog != "" {
		// Command line flag overrides config
		jsonlPath := expandJSONLPath(*jsonlLog)
		jsonlLevel := logger.ParseLogLevel(*jsonlLogLevel)
		if err := logger.SetGlobalJSONLLogger(jsonlPath, jsonlLevel); err != nil {
			logger.Warn("Failed to setup JSONL logging: %v", err)
		} else {
			logger.Debug("JSONL logging enabled: %s (level: %s)", jsonlPath, *jsonlLogLevel)
		}
	} else if cfg.Logging.JSONLEnabled {
		// Use config file settings
		jsonlPath := expandJSONLPath(cfg.Logging.JSONLFile)
		jsonlLevel := logger.ParseLogLevel(cfg.Logging.JSONLLevel)
		if err := logger.SetGlobalJSONLLogger(jsonlPath, jsonlLevel); err != nil {
			logger.Warn("Failed to setup JSONL logging: %v", err)
		} else {
			logger.Debug("JSONL logging enabled: %s (level: %s)", jsonlPath, cfg.Logging.JSONLLevel)
		}
	}

	// Create command manager
	cmdManager := command.NewManager(cfg)

	// Initialize CLI UI
	console, err := ui.NewUI()
	if err != nil {
		log.Fatalf("Failed to initialize UI: %v", err)
	}

	// Set UI modes before printing anything
	if *quiet {
		console.SetQuietMode(true)
	}

	// Set color mode
	if *noColor {
		console.SetColorMode(false)
	}

	// Set up advanced command completion
	var completer completion.Completer
	configCompleter, err := completion.NewConfigCompleter()
	if err != nil {
		// If config-based completer fails, try the basic one (which also tries to load YAML)
		logger.Warn("Failed to create config completer: %v", err)
		completer = completion.NewCommandCompleter()
	} else {
		completer = configCompleter
	}

	// Set session ID provider for interact command completion
	if provider, ok := completer.(interface{ SetSessionIDProvider(func() []string) }); ok {
		provider.SetSessionIDProvider(func() []string {
			sessions, err := sessionManager.ListSessions()
			if err != nil {
				return nil
			}

			var ids []string
			for _, sess := range sessions {
				ids = append(ids, sess.ID)
			}
			return ids
		})
	}

	// Set the completer on the UI
	console.SetCompleter(completer)

	// Set completer sort order based on config
	if cmdCompleter, ok := completer.(*completion.CommandCompleter); ok {
		// Get sort order from config (default to "rank" if not specified)
		sortOrder := cfg.General.CompletionSortOrder
		if sortOrder == "" {
			sortOrder = "rank"
		}

		// Set the sort order
		if sortOrder == "alphabetical" {
			cmdCompleter.SetSortOrder(completion.SortByAlphabetical)
		} else {
			cmdCompleter.SetSortOrder(completion.SortByRank)
		}
	}

	// Now print banner and debug info
	console.PrintBanner()

	// Only print debug info if debug mode is enabled
	if debugMode {
		logger.Debug("Running on %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	// Determine if we should execute a command and exit
	executeCommand := ""
	if *execCmd != "" {
		executeCommand = *execCmd
	} else if *execCmdShort != "" {
		executeCommand = *execCmdShort
	}

	// If execute command is specified, run it and exit
	if executeCommand != "" {
		// If session ID is specified, switch to that session first
		if *sessionID != "" {
			if err := switchToSession(sessionManager, *sessionID); err != nil {
				log.Fatalf("Failed to switch to session %s: %v", *sessionID, err)
			}
		}

		// Execute the command
		exitCode := executeAndExit(executeCommand, sessionManager, cmdManager, console)
		os.Exit(exitCode)
	}

	// Main interactive loop
	runCLI(console, sessionManager, cmdManager)
}

// Load configuration
func loadConfig(configFile string) (*config.Config, error) {
	if configFile == "" {
		// Search for default configuration file paths in order
		searchPaths := []string{
			"configs/client.yaml",
			"config.yaml",
		}

		// Add home directory paths
		if homeDir, err := os.UserHomeDir(); err == nil {
			searchPaths = append(searchPaths,
				filepath.Join(homeDir, ".config", "virga", "client.yaml"),
				filepath.Join(homeDir, ".config", "virga", "config.yaml"),
			)
		}

		// Find the first existing config file
		for _, path := range searchPaths {
			if _, err := os.Stat(path); err == nil {
				configFile = path
				break
			}
		}

		// If no config file found, use default path
		if configFile == "" {
			configFile = "configs/client.yaml"
		}
	}

	// Load configuration file
	cfg, err := config.LoadConfig(configFile)
	if err != nil {
		if strings.Contains(err.Error(), "no such file") {
			// If configuration file is not found, use default configuration
			return config.DefaultConfig(), nil
		}
		return nil, err
	}

	return cfg, nil
}

// CLI main loop
func runCLI(console *ui.UI, sessionManager *session.Manager, cmdManager *command.Manager) {
	// Ensure cleanup of reader
	defer console.Close()

	for {
		// Display prompt and read input
		currentSession := sessionManager.GetCurrentSession()
		var prompt string
		if currentSession != nil {
			prompt = fmt.Sprintf("virga (%s)> ", currentSession.ID)
		} else {
			prompt = "virga> "
		}

		input, err := console.ReadLine(prompt)
		if err != nil {
			if err.Error() == "EOF" {
				logger.Info("\nGoodbye!")
				break
			}
			logger.Error("Error reading input: %v", err)
			continue
		}

		// Skip if input is empty
		input = strings.TrimSpace(input)
		if input == "" {
			continue
		}

		// Exit command
		if input == "exit" || input == "quit" {
			logger.Info("Goodbye!")
			break
		}

		// Clear screen command
		if input == "clear" || input == "cls" {
			console.ClearScreen()
			continue
		}

		// Parse command
		cmdParts := splitCommand(input)
		if len(cmdParts) == 0 {
			continue
		}

		cmdName := cmdParts[0]
		var cmdArgs []string
		if len(cmdParts) > 1 {
			cmdArgs = cmdParts[1:]
		}

		// Track command for extended history
		currentSession = sessionManager.GetCurrentSession()
		var sessionID string
		var sessionInfo map[string]interface{}
		if currentSession != nil {
			sessionID = currentSession.ID
			// Extract session info from Information map
			if currentSession.Information != nil {
				sessionInfo = currentSession.Information
			}
		}
		console.StartCommand(input, sessionID, sessionInfo)

		// Execute
		cmd, exists := cmdManager.GetCommand(cmdName)
		if !exists {
			logger.Error("Unknown command: %s", cmdName)
			console.SetCommandResult("Unknown command", false)
			console.EndCommand()
			continue
		}

		// Log command execution with fields
		logFields := map[string]interface{}{
			"command_name": cmdName,
			"args":         cmdArgs,
		}
		if currentSession != nil {
			logFields["session_id"] = currentSession.ID
		}
		logger.LogWithFields(logger.INFO, fmt.Sprintf("Executing command: %s", cmdName), logFields)

		// Execute the command
		err = cmd.Execute(sessionManager, cmdArgs)

		if err != nil {
			logger.Error("%v", err)
			logFields["error"] = err.Error()
			logger.LogWithFields(logger.ERROR, fmt.Sprintf("Command failed: %s", cmdName), logFields)
			console.SetCommandResult(err.Error(), false)
		} else {
			// For commands that interact with sessions and produce output,
			// try to get the last task result
			resultOutput := ""
			if needsResultCapture(cmdName) && currentSession != nil {
				// Get the most recent task results
				if taskResults, err := currentSession.GetTaskResults(); err == nil && len(taskResults) > 0 {
					// Get the last result
					lastResult := taskResults[len(taskResults)-1]
					if lastResult.Output != "" {
						resultOutput = lastResult.Output
					}
				}
			}

			if resultOutput != "" {
				console.SetCommandResult(resultOutput, true)
				logFields["result_output_len"] = len(resultOutput)
			} else {
				console.SetCommandResult("Command executed successfully", true)
			}
			logger.LogWithFields(logger.INFO, fmt.Sprintf("Command succeeded: %s", cmdName), logFields)
		}
		console.EndCommand()
	}
}

// Split command (consider quotes)
func splitCommand(cmd string) []string {
	var parts []string
	var current string
	inQuote := false
	quoteChar := rune(0)

	for _, c := range cmd {
		switch {
		case c == '"' || c == '\'':
			if inQuote && c == quoteChar {
				inQuote = false
				quoteChar = rune(0)
			} else if !inQuote {
				inQuote = true
				quoteChar = c
			} else {
				current += string(c)
			}
		case c == ' ' && !inQuote:
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		default:
			current += string(c)
		}
	}

	if current != "" {
		parts = append(parts, current)
	}

	return parts
}

// executeAndExit executes a command and returns the exit code
func executeAndExit(cmdString string, sessionManager *session.Manager, cmdManager *command.Manager, _ *ui.UI) int {
	// Support multiple commands separated by semicolon
	commands := strings.Split(cmdString, ";")

	for _, cmd := range commands {
		cmd = strings.TrimSpace(cmd)
		if cmd == "" {
			continue
		}

		// Parse command
		cmdParts := splitCommand(cmd)
		if len(cmdParts) == 0 {
			continue
		}

		cmdName := cmdParts[0]
		var cmdArgs []string
		if len(cmdParts) > 1 {
			cmdArgs = cmdParts[1:]
		}

		// Execute command
		cmdObj, exists := cmdManager.GetCommand(cmdName)
		if !exists {
			logger.Error("Unknown command: %s", cmdName)
			return 1
		}

		if err := cmdObj.Execute(sessionManager, cmdArgs); err != nil {
			logger.Error("%v", err)
			return 1
		}
	}

	return 0
}

// switchToSession switches to the specified session
func switchToSession(sessionManager *session.Manager, sessionID string) error {
	sessions, err := sessionManager.ListSessions()
	if err != nil {
		return fmt.Errorf("failed to list sessions: %w", err)
	}

	for _, sess := range sessions {
		if sess.ID == sessionID || strings.HasPrefix(sess.ID, sessionID) {
			return sessionManager.SetCurrentSession(sess.ID)
		}
	}

	return fmt.Errorf("session not found: %s", sessionID)
}

// setLogLevel sets the logging level
func setLogLevel(level string) {
	// Set the global log level
	logger.SetGlobalLogLevelFromString(level)

	// Also set environment variable for child processes
	os.Setenv("VIRGA_CLI_LOG_LEVEL", level)
}

// needsResultCapture determines if a command's output should be captured for history
func needsResultCapture(cmdName string) bool {
	// Commands that typically produce output that should be captured
	outputCommands := map[string]bool{
		"shell":     true,
		"exec":      true,
		"ls":        true,
		"pwd":       true,
		"ps":        true,
		"netstat":   true,
		"sysinfo":   true,
		"netinfo":   true,
		"info":      true,
		"sessions":  true,
		"beacons":   true,
		"listeners": true,
		"workflow":  true,
		"llama":     true,
		"memdb":     true,
		"help":      true,
		"history":   true,
	}

	return outputCommands[cmdName]
}

// expandJSONLPath expands placeholders in JSONL file path
func expandJSONLPath(path string) string {
	now := time.Now()

	// Replace placeholders
	replacements := map[string]string{
		"{date}":      now.Format("20060102"),
		"{datetime}":  now.Format("20060102-150405"),
		"{time}":      now.Format("150405"),
		"{year}":      now.Format("2006"),
		"{month}":     now.Format("01"),
		"{day}":       now.Format("02"),
		"{hour}":      now.Format("15"),
		"{timestamp}": fmt.Sprintf("%d", now.Unix()),
	}

	result := path
	for placeholder, value := range replacements {
		result = strings.ReplaceAll(result, placeholder, value)
	}

	return result
}
