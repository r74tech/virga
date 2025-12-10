package command

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/r74tech/virga/internal/cli/client"
	"github.com/r74tech/virga/internal/cli/config"
	"github.com/r74tech/virga/internal/cli/generator"
	"github.com/r74tech/virga/internal/cli/session"
	"github.com/r74tech/virga/internal/server/protocol"
	"github.com/r74tech/virga/internal/shared/logger"
)

// Command is the CLI command interface.
type Command interface {
	Execute(sessionManager *session.Manager, args []string) error
	Help() string
}

// Manager is the CLI command manager.
type Manager struct {
	mu       sync.RWMutex
	commands map[string]Command
	config   *config.Config
	client   *client.APIClient
}

// NewManager creates a new command manager.
func NewManager(cfg *config.Config) *Manager {
	// Create API client
	apiClient := client.NewAPIClient(cfg)

	m := &Manager{
		commands: make(map[string]Command),
		config:   cfg,
		client:   apiClient,
	}

	// Register commands
	m.registerCommands()

	return m
}

// registerCommands registers available commands.
func (m *Manager) registerCommands() {
	// Basic commands
	m.RegisterCommand("help", &HelpCommand{manager: m})
	m.RegisterCommand("sessions", &SessionsCommand{})
	m.RegisterCommand("interact", &InteractCommand{})
	m.RegisterCommand("beacons", &BeaconsCommand{})
	m.RegisterCommand("listeners", &ListenersCommand{})
	m.RegisterCommand("generate", NewGenerateCommand(m.client))
	m.RegisterCommand("use", &UseCommand{client: m.client})
	m.RegisterCommand("workflow", &WorkflowCommand{})

	// Session interaction commands
	m.RegisterCommand("shell", &ShellCommand{})
	m.RegisterCommand("exec", &ExecCommand{})

	// File operation commands
	m.RegisterCommand("upload", &UploadCommand{})
	m.RegisterCommand("download", &DownloadCommand{})
	m.RegisterCommand("ls", &LsCommand{})
	m.RegisterCommand("cd", &CdCommand{})
	m.RegisterCommand("pwd", &PwdCommand{})

	// Information gathering commands
	m.RegisterCommand("info", &InfoCommand{})
	m.RegisterCommand("sysinfo", &SysInfoCommand{})
	m.RegisterCommand("netinfo", &NetworkInfoCommand{})
	m.RegisterCommand("ps", &ProcessListCommand{})

	// Process management commands
	m.RegisterCommand("kill", &KillCommand{})

	// Network commands
	m.RegisterCommand("netstat", &NetstatCommand{})
	m.RegisterCommand("portfwd", &PortfwdCommand{})

	// Configuration commands
	m.RegisterCommand("log", &LogCommand{})

	// Security commands
	m.RegisterCommand("killswitch", &KillswitchCommand{})

	// History command
	m.RegisterCommand("history", &HistoryCommand{})

	// Llama AI commands
	m.RegisterCommand("llama", &LlamaCommand{})
	m.RegisterCommand("memdb", &MemDBCommand{})
}

// RegisterCommand registers a command.
func (m *Manager) RegisterCommand(name string, cmd Command) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.commands[name] = cmd
}

// GetCommand retrieves a command by its name.
func (m *Manager) GetCommand(name string) (Command, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	cmd, exists := m.commands[name]
	return cmd, exists
}

// GetAllCommands returns all commands.
func (m *Manager) GetAllCommands() map[string]Command {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Return a copy to avoid external modifications
	cmdCopy := make(map[string]Command, len(m.commands))
	for k, v := range m.commands {
		cmdCopy[k] = v
	}
	return cmdCopy
}

// HelpCommand is the implementation of the 'help' command.
type HelpCommand struct {
	manager *Manager
}

// Execute executes the 'help' command.
func (c *HelpCommand) Execute(sessionManager *session.Manager, args []string) error {
	if len(args) > 0 {
		// Show help for a specific command
		cmdName := args[0]
		cmd, exists := c.manager.GetCommand(cmdName)
		if !exists {
			return fmt.Errorf("unknown command: %s", cmdName)
		}
		logger.Info(cmd.Help())
		return nil
	}

	// Show a list of all commands
	logger.Info("Available commands:")
	commands := c.manager.GetAllCommands()
	for name, cmd := range commands {
		helpText := cmd.Help()
		summary := strings.Split(helpText, "\n")[0]
		logger.Info("  %-15s %s", name, summary)
	}
	logger.Info("\nUse 'help <command>' for more information about a specific command.")
	return nil
}

// Help returns the help for the 'help' command.
func (c *HelpCommand) Help() string {
	return "Display help information about available commands.\n" +
		"Usage: help [command]\n" +
		"  command    The command to get help for. If omitted, lists all commands."
}

// SessionsCommand is the implementation of the 'sessions' command.
type SessionsCommand struct{}

// Execute executes the 'sessions' command.
func (c *SessionsCommand) Execute(sessionManager *session.Manager, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("missing subcommand. Use 'sessions list', 'sessions kill', etc.")
	}

	switch args[0] {
	case "list":
		// Display a list of active sessions
		sessions := sessionManager.GetAllSessions()
		if len(sessions) == 0 {
			logger.Info("No active sessions.")
			return nil
		}

		logger.Info("Active sessions:")
		logger.Info("%-36s %-15s %-20s %s", "ID", "IP", "Hostname", "Last Seen")
		logger.Info(strings.Repeat("-", 80))
		for _, s := range sessions {
			info := s.GetInformation()
			hostname := "unknown"
			if h, ok := info["hostname"].(string); ok {
				hostname = h
			}
			logger.Info("%-36s %-15s %-20s %s", s.ID, s.RemoteAddr(), hostname, s.LastSeen().Format("2006-01-02 15:04:05"))
		}
		return nil

	case "kill":
		if len(args) < 2 {
			return fmt.Errorf("session ID required. Usage: sessions kill <id>")
		}
		sessionID := args[1]
		if err := sessionManager.KillSession(sessionID); err != nil {
			return err
		}
		logger.Info("Session %s terminated.", sessionID)
		return nil

	default:
		return fmt.Errorf("unknown subcommand: %s", args[0])
	}
}

// Help returns the help for the 'sessions' command.
func (c *SessionsCommand) Help() string {
	return "Manage agent sessions.\n" +
		"Usage: sessions <subcommand> [args]\n" +
		"Subcommands:\n" +
		"  list       List all active sessions\n" +
		"  kill <id>  Terminate the specified session"
}

// InteractCommand is the implementation of the 'interact' command.
type InteractCommand struct{}

// Execute executes the 'interact' command.
func (c *InteractCommand) Execute(sessionManager *session.Manager, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("session ID required. Usage: interact <id>")
	}

	sessionID := args[0]
	if err := sessionManager.SetCurrentSession(sessionID); err != nil {
		return err
	}

	sess := sessionManager.GetCurrentSession()
	info := sess.GetInformation()
	hostname := "unknown"
	if h, ok := info["hostname"].(string); ok {
		hostname = h
	}

	logger.Info("Now interacting with session %s (%s)", sessionID, hostname)
	return nil
}

// Help returns the help for the 'interact' command.
func (c *InteractCommand) Help() string {
	return "Interact with a specific agent session.\n" +
		"Usage: interact <session_id>\n" +
		"  session_id    The ID of the session to interact with."
}

// BeaconsCommand is the implementation of the 'beacons' command.
type BeaconsCommand struct{}

// Execute executes the 'beacons' command.
func (c *BeaconsCommand) Execute(sessionManager *session.Manager, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("missing subcommand. Use 'beacons list', 'beacons set-sleep', etc.")
	}

	// Check the current session
	if sessionManager.GetCurrentSession() == nil {
		return fmt.Errorf("no session selected. Use 'interact <session_id>' first")
	}

	sessionID := sessionManager.GetCurrentSession().ID

	switch args[0] {
	case "list":
		logger.Info("Beacon configuration for session %s:", sessionID)
		// Get and display the beacon configuration here
		// In a real implementation, this would be fetched from the server
		logger.Info("Sleep time: 60 seconds")
		logger.Info("Jitter: 20%")
		return nil

	case "set-sleep":
		if len(args) < 2 {
			return fmt.Errorf("sleep time required. Usage: beacons set-sleep <seconds>")
		}
		// Send the command to the server here
		logger.Info("Sleep time for session %s updated to %s seconds.", sessionID, args[1])
		return nil

	case "set-jitter":
		if len(args) < 2 {
			return fmt.Errorf("jitter percentage required. Usage: beacons set-jitter <percent>")
		}
		// Send the command to the server here
		logger.Info("Jitter for session %s updated to %s%%.", sessionID, args[1])
		return nil

	default:
		return fmt.Errorf("unknown subcommand: %s", args[0])
	}
}

// Help returns the help for the 'beacons' command.
func (c *BeaconsCommand) Help() string {
	return "Manage beacon configurations.\n" +
		"Usage: beacons <subcommand> [args]\n" +
		"Subcommands:\n" +
		"  list            Show beacon configuration for current session\n" +
		"  set-sleep <sec> Set sleep time in seconds\n" +
		"  set-jitter <p>  Set jitter percentage (0-50)"
}

// ListenersCommand is the implementation of the 'listeners' command.
type ListenersCommand struct{}

// Execute executes the 'listeners' command.
func (c *ListenersCommand) Execute(sessionManager *session.Manager, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("missing subcommand. Use 'listeners list', 'listeners add', etc.")
	}

	switch args[0] {
	case "list":
		logger.Info("Active listeners:")
		logger.Info("%-15s %-10s %-15s %-5s %s", "Name", "Type", "BindAddress", "Port", "Status")
		logger.Info(strings.Repeat("-", 80))
		// Display the list of listeners here (in a production implementation, get the actual listener information)
		logger.Info("%-15s %-10s %-15s %-5d %s", "default-http", "http", "0.0.0.0", 8080, "Running")
		return nil

	case "add":
		// Validate arguments
		if len(args) < 4 {
			return fmt.Errorf("insufficient arguments. Usage: listeners add <n> <type> <bind_address> <port>")
		}

		name := args[1]
		listenerType := args[2]
		bindAddress := args[3]
		port := args[4]

		// Add the listener here (in a production implementation, send a command to the server)
		logger.Info("Listener %s added (%s://%s:%s).", name, listenerType, bindAddress, port)
		return nil

	case "remove":
		if len(args) < 2 {
			return fmt.Errorf("listener name required. Usage: listeners remove <n>")
		}
		name := args[1]
		// Remove the listener here
		logger.Info("Listener %s removed.", name)
		return nil

	default:
		return fmt.Errorf("unknown subcommand: %s", args[0])
	}
}

// Help returns the help for the 'listeners' command.
func (c *ListenersCommand) Help() string {
	return "Manage C2 listeners.\n" +
		"Usage: listeners <subcommand> [args]\n" +
		"Subcommands:\n" +
		"  list                                   List all active listeners\n" +
		"  add <n> <type> <bind_address> <port> Add a new listener\n" +
		"  remove <n>                          Remove a listener"
}

// GenerateCommand is the implementation of the 'generate' command.
type GenerateCommand struct {
	Client *client.APIClient
}

// NewGenerateCommand creates a new GenerateCommand.
func NewGenerateCommand(client *client.APIClient) *GenerateCommand {
	return &GenerateCommand{
		Client: client,
	}
}

// extractFlagValue extracts the value of a flag from the arguments.
func extractFlagValue(args []string, flag string) (string, bool) {
	for i := 0; i < len(args)-1; i++ {
		if args[i] == flag {
			return args[i+1], true
		}
	}
	return "", false
}

// hasFlagOption checks if a flag is present in the arguments.
func hasFlagOption(args []string, flag string) bool {
	for _, arg := range args {
		if arg == flag {
			return true
		}
	}
	return false
}

// getOptionValue returns the value for a given option flag
// For example: getOptionValue(args, "--llama-mode") returns "selfextract" from ["--llama-mode", "selfextract"]
func getOptionValue(args []string, flag string) string {
	for i, arg := range args {
		if arg == flag && i+1 < len(args) {
			return args[i+1]
		}
	}
	return ""
}

// Execute executes the 'generate' command.
func (c *GenerateCommand) Execute(sessionManager *session.Manager, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("payload type required. Use 'generate beacon|stager|shellcode [options]'")
	}

	payloadType := args[0]
	switch payloadType {
	case "beacon", "agent":
		// Options for binary payload generation
		binGenerator, err := generator.NewBinaryGenerator()
		if err != nil {
			return fmt.Errorf("failed to create binary generator: %v", err)
		}
		opts := binGenerator.GetDefaultOptions()

		// Initialize ImplantOpts with defaults
		opts.ImplantOpts = &generator.ImplantOpts{
			LogEnabled:  false,
			LogFilePath: "",
			LogLevel:    "info",
			Debug:       false,
		}

		// Check for YAML config file
		if configPath, ok := extractFlagValue(args, "--config"); ok {
			beaconConfig, err := config.LoadBeaconConfig(configPath)
			if err != nil {
				return fmt.Errorf("failed to load config file: %v", err)
			}
			// Apply options from YAML
			opts.C2Host = beaconConfig.Beacon.C2.Host
			opts.C2Port = beaconConfig.Beacon.C2.Port
			opts.Protocol = beaconConfig.Beacon.C2.Protocol
			opts.URIPath = beaconConfig.Beacon.C2.URIPath
			opts.UserAgent = beaconConfig.Beacon.Behavior.UserAgent
			opts.SleepTime = beaconConfig.Beacon.Behavior.SleepTime
			opts.Jitter = beaconConfig.Beacon.Behavior.Jitter
			if beaconConfig.Beacon.Target.OS != "" {
				opts.TargetOS = beaconConfig.Beacon.Target.OS
			}
			if beaconConfig.Beacon.Target.Arch != "" {
				opts.TargetArch = beaconConfig.Beacon.Target.Arch
			}
			if beaconConfig.Beacon.Target.Format != "" {
				opts.Format = beaconConfig.Beacon.Target.Format
			}
			if beaconConfig.Output.Path != "" {
				opts.OutputPath = beaconConfig.Output.Path
			}
			opts.EnableLlama = beaconConfig.Llama.Enabled

			// Apply Llama settings
			if beaconConfig.Llama.Enabled {
				opts.LlamaConfig = &generator.LlamaOpts{
					Context:       beaconConfig.Llama.Model.Context,
					MaxTokens:     beaconConfig.Llama.Model.MaxTokens,
					MaxIterations: beaconConfig.Llama.Autonomous.MaxIterations,
					Temperature:   beaconConfig.Llama.Model.Temperature,
					GPULayers:     beaconConfig.Llama.Model.GPULayers,
					SystemPrompt:  beaconConfig.Llama.Prompt.Preset,
					AutoMode:      beaconConfig.Llama.Autonomous.Enabled,
					LogEnabled:    beaconConfig.Llama.LogEnabled,
				}

				// Convert initial tasks
				if len(beaconConfig.Llama.Autonomous.InitialTasks) > 0 {
					opts.LlamaConfig.InitialTasks = make([]generator.InitialTask, len(beaconConfig.Llama.Autonomous.InitialTasks))
					for i, task := range beaconConfig.Llama.Autonomous.InitialTasks {
						opts.LlamaConfig.InitialTasks[i] = generator.InitialTask{
							Type:        task.Type,
							Description: task.Description,
						}
					}
				}

				// Apply task prompts
				if len(beaconConfig.Llama.TaskPrompts) > 0 {
					opts.LlamaConfig.TaskPrompts = beaconConfig.Llama.TaskPrompts
				}
			}

			// Apply Implant settings
			if beaconConfig.Implant.LogEnabled || beaconConfig.Implant.LogFilePath != "" || beaconConfig.Implant.LogLevel != "" {
				opts.ImplantOpts = &generator.ImplantOpts{
					LogEnabled:  beaconConfig.Implant.LogEnabled,
					LogFilePath: beaconConfig.Implant.LogFilePath,
					LogLevel:    beaconConfig.Implant.LogLevel,
				}
			}

			// Apply embedded payloads (if any)
			if len(beaconConfig.Beacon.Payloads) > 0 {
				baseDir := filepath.Dir(configPath)
				for _, payload := range beaconConfig.Beacon.Payloads {
					source := payload.Source
					if source == "" {
						continue
					}
					if !strings.HasPrefix(source, "http://") && !strings.HasPrefix(source, "https://") && !filepath.IsAbs(source) {
						source = filepath.Join(baseDir, source)
					}

					data, err := generator.LoadPayloadSource(source)
					if err != nil {
						return fmt.Errorf("failed to load payload %s: %w", payload.Name, err)
					}

					typeName := payload.Type
					if typeName == "" {
						typeName = "powershell"
					}

					// Resolve CommandsFile path if provided
					commandsFile := payload.CommandsFile
					if commandsFile != "" && !filepath.IsAbs(commandsFile) {
						commandsFile = filepath.Join(baseDir, commandsFile)
					}

					opts.Payloads = append(opts.Payloads, generator.PayloadAsset{
						Name:         payload.Name,
						Type:         typeName,
						Content:      data,
						CommandsFile: commandsFile,
					})
				}
			}

			if beaconConfig.Beacon.Shell.PowerShell.AMSIBypass != "" {
				opts.AMSIBypass = beaconConfig.Beacon.Shell.PowerShell.AMSIBypass
			}
		}

		// Parse command-line options
		if c2Host, ok := extractFlagValue(args, "--c2"); ok {
			opts.C2Host = c2Host
		} else if c2Host, ok := extractFlagValue(args, "--host"); ok {
			opts.C2Host = c2Host
		} else if mtlsHost, ok := extractFlagValue(args, "--mtls"); ok {
			opts.C2Host = mtlsHost
			opts.Protocol = "mtls"
		} else if httpsHost, ok := extractFlagValue(args, "--https"); ok {
			opts.C2Host = httpsHost
			opts.Protocol = "https"
		} else if httpHost, ok := extractFlagValue(args, "--http"); ok {
			opts.C2Host = httpHost
			opts.Protocol = "http"
		} else if dnsHost, ok := extractFlagValue(args, "--dns"); ok {
			opts.C2Host = dnsHost
			opts.Protocol = "dns"
		}

		if c2PortStr, ok := extractFlagValue(args, "--port"); ok {
			if port, err := strconv.Atoi(c2PortStr); err == nil {
				opts.C2Port = port
			}
		}

		if uriPath, ok := extractFlagValue(args, "--uri"); ok {
			opts.URIPath = uriPath
		}

		if targetOS, ok := extractFlagValue(args, "--os"); ok {
			opts.TargetOS = targetOS
		}

		if targetArch, ok := extractFlagValue(args, "--arch"); ok {
			opts.TargetArch = targetArch
		}

		if format, ok := extractFlagValue(args, "--format"); ok {
			opts.Format = format
		}

		if sleepStr, ok := extractFlagValue(args, "--sleep"); ok {
			if sleep, err := strconv.Atoi(sleepStr); err == nil {
				opts.SleepTime = sleep
			}
		}

		if jitterStr, ok := extractFlagValue(args, "--jitter"); ok {
			if jitter, err := strconv.Atoi(jitterStr); err == nil {
				opts.Jitter = jitter
			}
		}

		if outputPath, ok := extractFlagValue(args, "--output"); ok {
			opts.OutputPath = outputPath
		} else if outputPath, ok := extractFlagValue(args, "--save"); ok {
			opts.OutputPath = outputPath
		}

		// Enable Llama integration (command line takes precedence)
		if hasFlagOption(args, "--enable-llama") {
			opts.EnableLlama = true
			// Apply default Llama settings (if not overridden by config)
			if opts.LlamaConfig == nil {
				opts.LlamaConfig = &generator.LlamaOpts{
					Context:       8192,
					MaxTokens:     2048,
					MaxIterations: 50,
					Temperature:   0.3,
					GPULayers:     0,
					SystemPrompt:  "default",
					AutoMode:      true,
					LogEnabled:    false,
				}
			}
		} else if hasFlagOption(args, "--no-llama") {
			opts.EnableLlama = false
		}

		// Enable Llama logging
		if hasFlagOption(args, "--llama-log") && opts.LlamaConfig != nil {
			opts.LlamaConfig.LogEnabled = true
		}

		// Force self-extracting mode (mmap) for Llama
		// Support both --llama-model-self-extract and --llama-mode selfextract
		if opts.LlamaConfig != nil {
			if hasFlagOption(args, "--llama-model-self-extract") {
				opts.LlamaConfig.ForceSelfExtract = true
			} else if llamaMode := getOptionValue(args, "--llama-mode"); llamaMode == "selfextract" {
				opts.LlamaConfig.ForceSelfExtract = true
			}
		}

		// Set Implant logging
		if hasFlagOption(args, "--implant-log") {
			opts.ImplantOpts.LogEnabled = true
		}

		if logPath, ok := extractFlagValue(args, "--implant-log-path"); ok {
			opts.ImplantOpts.LogFilePath = logPath
		}

		if logLevel, ok := extractFlagValue(args, "--implant-log-level"); ok {
			opts.ImplantOpts.LogLevel = logLevel
		}

		// Check for debug flag
		if hasFlagOption(args, "--debug") {
			opts.ImplantOpts.Debug = true
		}

		// Generate payload
		if opts.EnableLlama {
			logger.Info("Generating %s beacon for %s/%s targeting %s with Llama AI integration...",
				opts.Format, opts.Protocol, opts.C2Host, opts.TargetOS)
		} else {
			logger.Info("Generating %s beacon for %s/%s targeting %s...",
				opts.Format, opts.Protocol, opts.C2Host, opts.TargetOS)
		}

		binary, err := binGenerator.Generate(opts)
		if err != nil {
			return fmt.Errorf("failed to generate binary payload: %v", err)
		}

		// Save to file
		if err := binGenerator.SaveToFile(binary, opts.OutputPath); err != nil {
			return fmt.Errorf("failed to save payload to file: %v", err)
		}

		// Clean up temporary files
		binGenerator.Cleanup()

		// Success message
		absPath, _ := filepath.Abs(opts.OutputPath)
		logger.Info("Binary payload saved to: %s", absPath)
		logger.Info("Size: %d bytes", len(binary))

		return nil

	case "shellcode", "raw":
		logger.Info("Raw shellcode payload generation not implemented yet.")
		return nil

	default:
		return fmt.Errorf("unknown payload type: %s", payloadType)
	}
}

// Help returns the help for the 'generate' command.
func (c *GenerateCommand) Help() string {
	return "Generate payloads.\n" +
		"Usage: generate <type> [options]\n" +
		"Types:\n" +
		"  powershell   Generate PowerShell script\n" +
		"  beacon       Generate binary beacon\n" +
		"  shellcode    Generate raw shellcode (not implemented yet)\n" +
		"\n" +
		"Common Options:\n" +
		"  --c2, --host <host>  C2 server address\n" +
		"  --port <port>        C2 server port\n" +
		"  --uri <path>         URI path for callbacks\n" +
		"  --http <host>        Use HTTP protocol\n" +
		"  --https <host>       Use HTTPS protocol\n" +
		"  --mtls <host>        Use Mutual TLS protocol\n" +
		"  --dns <host>         Use DNS protocol\n" +
		"  --sleep <sec>        Sleep time in seconds\n" +
		"  --jitter <percent>   Jitter percentage\n" +
		"  --save, --output <path> Output file path\n" +
		"\n" +
		"Binary Options:\n" +
		"  --os <os>            Target OS: windows, linux, darwin\n" +
		"  --arch <arch>        Target architecture: amd64, 386, arm64\n" +
		"  --format <format>    Output format: exe, dll, elf, dylib, etc.\n" +
		"  --enable-llama       Enable Llama AI integration (requires deps)\n" +
		"  --no-llama           Disable Llama AI integration\n" +
		"  --llama-log          Enable Llama logging\n" +
		"  --llama-mode <mode>  Llama integration mode: embed (default) or selfextract\n" +
		"  --llama-model-self-extract  [DEPRECATED] Use --llama-mode selfextract\n" +
		"  --implant-log        Enable implant logging\n" +
		"  --implant-log-path   Set custom log file path (default: implant.log)\n" +
		"  --implant-log-level  Set log level: debug, info, warn, error (default: info)\n" +
		"  --debug              Enable debug output during generation\n" +
		"  --config <path>      Use YAML configuration file\n" +
		"\n" +
		"PowerShell Options:\n" +
		"  --ua <agent>         User agent string\n" +
		"  --no-amsi            Disable AMSI bypass\n" +
		"  --no-etw             Disable ETW bypass\n" +
		"  --self-delete        Add self-deletion code\n" +
		"  --obfuscate          Obfuscate the script\n" +
		"\n" +
		"Examples:\n" +
		"  generate powershell --https example.com --port 443 --save /tmp/agent.ps1\n" +
		"  generate beacon --mtls 10.10.14.3 --save /home/user/agent --os linux --arch amd64\n" +
		"  generate beacon --config beacon-config.yaml\n" +
		"  generate beacon --config beacon-config.yaml --no-llama  # Override config\n" +
		"  generate beacon --enable-llama --llama-mode selfextract  # Force mmap mode"
}

// UseCommand is the implementation of the 'use' command for extensions.
type UseCommand struct {
	client *client.APIClient
}

// Execute executes the 'use' command.
func (c *UseCommand) Execute(sessionManager *session.Manager, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("extension path or subcommand required. Usage: use <path> or use <subcommand>")
	}

	subcommand := args[0]

	// Handle help command specially (no session required)
	if subcommand == "help" {
		logger.Info(c.Help())
		return nil
	}

	// Check if we have a current session for other commands
	currentSession := sessionManager.GetCurrentSession()
	if currentSession == nil && subcommand != "help" {
		return fmt.Errorf("no session selected. Use 'interact <session_id>' first")
	}

	switch subcommand {
	case "list":
		// List loaded extensions
		extensions, err := c.client.ListExtensions(currentSession.ID)
		if err != nil {
			return fmt.Errorf("failed to list extensions: %v", err)
		}

		if len(extensions) == 0 {
			logger.Info("No extensions loaded.")
			return nil
		}

		logger.Info("Loaded extensions:")
		logger.Info("%-20s %-10s %-10s %s", "Name", "Version", "Type", "Description")
		logger.Info(strings.Repeat("-", 60))
		for _, ext := range extensions {
			name := "unknown"
			if n, ok := ext["name"].(string); ok {
				name = n
			}
			version := "unknown"
			if v, ok := ext["version"].(string); ok {
				version = v
			}
			extType := "unknown"
			if t, ok := ext["type"].(string); ok {
				extType = t
			}
			description := ""
			if d, ok := ext["description"].(string); ok {
				description = d
			}
			logger.Info("%-20s %-10s %-10s %s", name, version, extType, description)
		}
		return nil

	case "unload":
		if len(args) < 2 {
			return fmt.Errorf("extension name required. Usage: use unload <name>")
		}
		name := args[1]
		err := c.client.DeleteExtension(currentSession.ID, name)
		if err != nil {
			return fmt.Errorf("failed to unload extension: %v", err)
		}
		logger.Info("Extension '%s' unloaded successfully.", name)
		return nil

	case "execute", "exec":
		if len(args) < 2 {
			return fmt.Errorf("extension name required. Usage: use execute <name> [command] [args...]")
		}
		name := args[1]

		// Default command if not specified
		command := "default"
		if len(args) > 2 {
			command = args[2]
		}

		// Parse additional arguments
		extArgs := make(map[string]interface{})
		for i := 3; i < len(args); i++ {
			// Simple key=value parsing
			parts := strings.SplitN(args[i], "=", 2)
			if len(parts) == 2 {
				extArgs[parts[0]] = parts[1]
			}
		}

		// Execute extension and get task ID
		taskID, err := c.client.SendSessionCommand(currentSession.ID, "extension_execute", map[string]interface{}{
			"extension": name,
			"command":   command,
			"arguments": extArgs,
		})
		if err != nil {
			return fmt.Errorf("failed to execute extension: %v", err)
		}

		logger.Info("Extension execution task queued (Task ID: %s)", taskID)
		logger.Info("Use 'use status <task-id>' to check the result")
		return nil

	case "status":
		if len(args) < 2 {
			return fmt.Errorf("task ID required. Usage: use status <task-id>")
		}
		taskID := args[1]

		// Get task result
		result, err := currentSession.GetTaskResult(taskID)
		if err != nil {
			return fmt.Errorf("failed to get task result: %v", err)
		}

		if result == nil {
			logger.Info("Task %s is still pending...", taskID)
			return nil
		}

		// Display result
		logger.Info("Task %s completed", taskID)
		logger.Info("Exit Code: %d", result.ExitCode)
		if result.Error != "" {
			logger.Error("Error: %s", result.Error)
		}
		if result.Output != "" {
			logger.Info("Output:\n%s", result.Output)
		}
		return nil

	case "help":
		// Already handled above, but kept for switch completeness
		logger.Info(c.Help())
		return nil

	default:
		// Assume it's a path to load an extension
		path := args[0]

		// Read extension manifest and data
		manifest, data, err := c.readExtensionPackage(path)
		if err != nil {
			return fmt.Errorf("failed to read extension package: %v", err)
		}

		// Upload extension to server
		err = c.client.UploadExtension(currentSession.ID, manifest, data)
		if err != nil {
			return fmt.Errorf("failed to upload extension: %v", err)
		}

		logger.Info("Extension loaded successfully from '%s'.", path)
		return nil
	}
}

// readExtensionPackage reads an extension package from disk
func (c *UseCommand) readExtensionPackage(path string) (string, []byte, error) {
	// Check if path is a directory or file
	info, err := os.Stat(path)
	if err != nil {
		return "", nil, fmt.Errorf("failed to stat path: %v", err)
	}

	var manifestPath string
	var dataPath string

	// Check if it's a tar.gz file
	if !info.IsDir() && (strings.HasSuffix(path, ".tar.gz") || strings.HasSuffix(path, ".tgz")) {
		// Extract tar.gz package
		manifestJSON, binaryData, err := extractTarGzPackage(path)
		if err != nil {
			return "", nil, fmt.Errorf("failed to extract tar.gz package: %v", err)
		}
		return manifestJSON, binaryData, nil
	}

	if info.IsDir() {
		// Directory - look for manifest.json
		manifestPath = filepath.Join(path, "manifest.json")

		// Read manifest to determine the correct binary file
		manifestData, err := os.ReadFile(manifestPath)
		if err != nil {
			return "", nil, fmt.Errorf("failed to read manifest.json: %v", err)
		}

		var manifest protocol.ExtensionManifest
		if err := json.Unmarshal(manifestData, &manifest); err != nil {
			return "", nil, fmt.Errorf("failed to parse manifest.json: %v", err)
		}

		// Find the appropriate platform file
		// TODO: Implement platform detection and selection
		// For now, just use the first file
		if len(manifest.Platforms) == 0 {
			return "", nil, fmt.Errorf("no platform files specified in manifest")
		}

		dataPath = filepath.Join(path, manifest.Platforms[0].Path)
	} else {
		// Single file - assume it's the extension binary
		// Look for manifest.json in the same directory
		dir := filepath.Dir(path)
		manifestPath = filepath.Join(dir, "manifest.json")
		dataPath = path
	}

	// Read manifest
	manifestData, err := os.ReadFile(manifestPath)
	if err != nil {
		return "", nil, fmt.Errorf("failed to read manifest: %v", err)
	}

	// Read binary data
	binaryData, err := os.ReadFile(dataPath)
	if err != nil {
		return "", nil, fmt.Errorf("failed to read extension binary: %v", err)
	}

	return string(manifestData), binaryData, nil
}

// extractTarGzPackage extracts manifest.json and binary data from a tar.gz package
func extractTarGzPackage(path string) (string, []byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", nil, fmt.Errorf("failed to open tar.gz file: %v", err)
	}
	defer file.Close()

	// Create gzip reader
	gzReader, err := gzip.NewReader(file)
	if err != nil {
		return "", nil, fmt.Errorf("failed to create gzip reader: %v", err)
	}
	defer gzReader.Close()

	// Create tar reader
	tarReader := tar.NewReader(gzReader)

	var manifestJSON string
	var binaryData []byte
	var foundManifest bool
	var foundBinary bool

	// Read files from tar
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", nil, fmt.Errorf("failed to read tar header: %v", err)
		}

		// Skip directories
		if header.Typeflag == tar.TypeDir {
			continue
		}

		// Read file contents
		data, err := io.ReadAll(tarReader)
		if err != nil {
			return "", nil, fmt.Errorf("failed to read file %s: %v", header.Name, err)
		}

		// Check for manifest.json
		if filepath.Base(header.Name) == "manifest.json" {
			manifestJSON = string(data)
			foundManifest = true
		} else if !foundBinary {
			// For now, assume the first non-manifest file is the binary
			// TODO: Parse manifest to determine correct binary file
			binaryData = data
			foundBinary = true
		}

		// If we have both files, we can stop
		if foundManifest && foundBinary {
			break
		}
	}

	if !foundManifest {
		return "", nil, fmt.Errorf("manifest.json not found in tar.gz package")
	}
	if !foundBinary {
		return "", nil, fmt.Errorf("no binary file found in tar.gz package")
	}

	return manifestJSON, binaryData, nil
}

// Help returns the help for the 'use' command.
func (c *UseCommand) Help() string {
	return "Manage and use extensions.\n" +
		"Usage: use <path|subcommand> [args]\n" +
		"Subcommands:\n" +
		"  <path>              Load an extension from the specified path\n" +
		"  list                List all loaded extensions\n" +
		"  unload <name>       Unload a specific extension\n" +
		"  execute <name> [command] [args...] Execute an extension with arguments\n" +
		"  status <task-id>    Check the result of an extension execution task\n" +
		"  help                Display this help message\n" +
		"\n" +
		"Examples:\n" +
		"  use /tmp/extensions/rubeus     Load the Rubeus extension\n" +
		"  use list                       List loaded extensions\n" +
		"  use execute rubeus triage      Execute Rubeus with 'triage' command\n" +
		"  use execute ps_killer kill pid=1234  Execute with arguments\n" +
		"  use status task-123            Check the result of a task\n" +
		"  use unload rubeus              Unload the Rubeus extension"
}

// WorkflowCommand is the implementation of the 'workflow' command.
type WorkflowCommand struct{}

// Execute executes the 'workflow' command.
func (c *WorkflowCommand) Execute(sessionManager *session.Manager, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("missing subcommand. Use 'workflow list', 'workflow run', etc.")
	}

	switch args[0] {
	case "list":
		logger.Info("Available workflows:")
		logger.Info("  recon           Basic reconnaissance workflow")
		logger.Info("  privesc         Privilege escalation workflow")
		logger.Info("  persistence     Establish persistence workflow")
		return nil

	case "run":
		if len(args) < 2 {
			return fmt.Errorf("workflow name required. Usage: workflow run <n>")
		}

		workflowName := args[1]

		// Check if a session is selected
		if sessionManager.GetCurrentSession() == nil {
			return fmt.Errorf("no session selected. Use 'interact <session_id>' first")
		}

		logger.Info("Running workflow '%s'...", workflowName)
		// Workflow execution process here

		return nil

	default:
		return fmt.Errorf("unknown subcommand: %s", args[0])
	}
}

// Help returns the help for the 'workflow' command.
func (c *WorkflowCommand) Help() string {
	return "Manage automated workflows.\n" +
		"Usage: workflow <subcommand> [args]\n" +
		"Subcommands:\n" +
		"  list       List available workflows\n" +
		"  run <n> Execute a workflow on the current session"
}
