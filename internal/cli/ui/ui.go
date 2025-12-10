package ui

import (
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/peterh/liner"
	"github.com/r74tech/virga/internal/shared/completion"
	"github.com/r74tech/virga/internal/shared/logger"
)

// Completer interface for command completion
type Completer interface {
	Complete(line string, pos int) (completions []string, startPos int)
}

// Build variables set at compile time
var (
	// Version is the application version
	Version = "1.0.0"
	// GitCommit is the Git commit hash at build time
	GitCommit = "unknown"
	// BuildDate is the build timestamp
	BuildDate = "unknown"
)

// UI is the CLI user interface helper.
type UI struct {
	mu                    sync.RWMutex
	liner                 *liner.State
	config                *Config
	historyWriter         *historyWriter
	extendedHistoryWriter *ExtendedHistoryWriter
	quietMode             bool
	colorMode             bool
	commandCompleter      Completer
}

// NewUI creates a new UI helper with default configuration.
func NewUI() (*UI, error) {
	return NewUIWithConfig(DefaultConfig())
}

// NewUIWithConfig creates a new UI helper with custom configuration.
func NewUIWithConfig(config *Config) (*UI, error) {
	ui := &UI{
		config:    config,
		colorMode: config.EnableColors,
	}

	// Initialize the liner
	ui.liner = liner.NewLiner()
	ui.liner.SetCtrlCAborts(true)

	// Setup history if enabled
	if config.EnableHistory {
		if err := ui.setupHistory(); err != nil {
			return nil, newUIError("NewUI", "failed to setup history", err)
		}

		// Setup extended history writer
		configDir, _ := getConfigDir()
		extHistoryFile := filepath.Join(configDir, "history.json")
		ehw, err := NewExtendedHistoryWriter(extHistoryFile)
		if err != nil {
			// Log error but don't fail
			logger.Warn("Failed to create extended history writer: %v", err)
		} else {
			ui.extendedHistoryWriter = ehw
		}
	}

	// Set the basic completion function initially
	ui.liner.SetCompleter(ui.completer)

	return ui, nil
}

// setupHistory sets up the history management
func (ui *UI) setupHistory() error {
	// Ensure config directory exists with proper permissions
	configDir, err := getConfigDir()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(configDir, 0o700); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}

	// Create history writer
	hw, err := newHistoryWriter(
		ui.config.HistoryFile,
		ui.config.HistoryBatchSize,
		ui.config.HistoryFlushTime,
		ui.config.MaxHistorySize,
	)
	if err != nil {
		return err
	}
	ui.historyWriter = hw

	// Load existing history
	history, err := hw.LoadHistory()
	if err != nil {
		// Log error but don't fail
		ui.PrintWarning(fmt.Sprintf("Failed to load history: %v", err))
	} else {
		for _, line := range history {
			ui.liner.AppendHistory(line)
		}
	}

	return nil
}

// completer provides command completion
func (ui *UI) completer(line string) []string {
	// Use the advanced completer if available
	if ui.commandCompleter != nil {
		completions, startPos := ui.commandCompleter.Complete(line, len(line))

		// liner expects full replacements, so we need to prepend the prefix
		if startPos > 0 && startPos < len(line) {
			prefix := line[:startPos]
			var results []string
			for _, comp := range completions {
				results = append(results, prefix+comp)
			}
			return results
		}

		return completions
	}

	// Fallback to basic completion
	cmds := []string{
		"help", "exit", "quit", "sessions", "interact", "beacons",
		"listeners", "generate", "use", "workflow", "shell", "exec",
		"upload", "download", "ls", "cd", "pwd", "info", "sysinfo",
		"netinfo", "ps", "kill", "netstat", "portfwd", "log",
		"llama", "memdb", "clear", "cls", "sleep",
	}

	var completions []string

	// Command completion
	if len(line) == 0 {
		return cmds
	}

	// Simple prefix matching
	for _, cmd := range cmds {
		if strings.HasPrefix(cmd, line) {
			completions = append(completions, cmd)
		}
	}

	return completions
}

// ReadLine displays a prompt and reads user input.
func (ui *UI) ReadLine(prompt string) (string, error) {
	ui.mu.Lock()
	defer ui.mu.Unlock()

	line, err := ui.liner.Prompt(prompt)
	if err != nil {
		return "", err
	}

	// Add to history
	if line != "" && ui.config.EnableHistory {
		ui.liner.AppendHistory(line)
		if ui.historyWriter != nil {
			ui.historyWriter.Add(line)
		}
	}

	return line, nil
}

// Close releases the UI resources.
func (ui *UI) Close() error {
	ui.mu.Lock()
	defer ui.mu.Unlock()

	var errs []error

	if ui.historyWriter != nil {
		if err := ui.historyWriter.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	if ui.liner != nil {
		ui.liner.Close()
	}

	if len(errs) > 0 {
		return errs[0]
	}
	return nil
}

// PrintBanner displays the application banner.
func (ui *UI) PrintBanner() {
	rand.Seed(time.Now().UnixNano())

	banners := []string{
		`
 ██▒   █▓ ██▓ ██▀███    ▄████  ▄▄▄      
▓██░   █▒▓██▒▓██ ▒ ██▒ ██▒ ▀█▒▒████▄    
 ▓██  █▒░▒██▒▓██ ░▄█ ▒▒██░▄▄▄░▒██  ▀█▄  
  ▒██ █░░░██░▒██▀▀█▄  ░▓█  ██▓░██▄▄▄▄██ 
   ▒▀█░  ░██░░██▓ ▒██▒░▒▓███▀▒ ▓█   ▓██▒
   ░ ▐░  ░▓  ░ ▒▓ ░▒▓░ ░▒   ▒  ▒▒   ▓▒█░
   ░ ░░   ▒ ░  ░▒ ░ ▒░  ░   ░   ▒   ▒▒ ░
     ░░   ▒ ░  ░░   ░ ░ ░   ░   ░   ▒   
      ░   ░     ░           ░       ░  ░
     ░                                  
`,
		`
██╗   ██╗██╗██████╗  ██████╗  █████╗ 
██║   ██║██║██╔══██╗██╔════╝ ██╔══██╗
██║   ██║██║██████╔╝██║  ███╗███████║
╚██╗ ██╔╝██║██╔══██╗██║   ██║██╔══██║
 ╚████╔╝ ██║██║  ██║╚██████╔╝██║  ██║
  ╚═══╝  ╚═╝╚═╝  ╚═╝ ╚═════╝ ╚═╝  ╚═╝
`,
		`
              ,,                            
` + "`" + `7MMF'   ` + "`" + `7MF'db                            
  ` + "`" + `MA     ,V                                
   VM:   ,V ` + "`" + `7MM  ` + "`" + `7Mb,od8 .P"Ybmmm ,6"Yb.  
    MM.  M'   MM    MM' "':MI  I8  8)   MM  
    ` + "`" + `MM A'    MM    MM     WmmmP"   ,pm9MM  
     :MM;     MM    MM    8M       8M   MM  
      VF    .JMML..JMML.   YMMMMMb ` + "`" + `Moo9^Yo.
                          6'     dP         
                          Ybmmmd'           
`,
		`
 __     ___                 
 \ \   / (_)_ __ __ _  __ _ 
  \ \ / /| | '__/ _` + "`" + ` |/ _` + "`" + ` |
   \ V / | | | | (_| | (_| |
    \_/  |_|_|  \__, |\__,_|
                |___/       
`,
		`
 ,ggg,         ,gg                                     
dP""Y8a       ,8P                                      
Yb, ` + "`" + `88       d8'                                      
 ` + "`" + `"  88       88gg                                     
     88       88""                                     
     I8       8Igg    ,gggggg,    ,gggg,gg    ,gggg,gg 
     ` + "`" + `8,     ,8'88    dP""""8I   dP"  "Y8I   dP"  "Y8I 
      Y8,   ,8P 88   ,8'    8I  i8'    ,8I  i8'    ,8I 
       Yb,_,dP_,88,_,dP     Y8,,d8,   ,d8I ,d8,   ,d8b,
        "Y8P" 8P""Y88P      ` + "`" + `Y8P"Y8888P"888P"Y8888P"` + "`" + `Y8
                                      ,d8I'            
                                    ,dP'8I             
                                   ,8"  8I             
                                   I8   8I             
                                   ` + "`" + `8, ,8I             
                                    ` + "`" + `Y8P"              
`,
		`
           d8,                             
          ` + "`" + `8P                              
                                           
?88   d8P  88b  88bd88b d888b8b   d888b8b  
d88  d8P'  88P  88P'  ` + "`" + `d8P' ?88  d8P' ?88  
?8b ,88'  d88  d88     88b  ,88b 88b  ,88b 
` + "`" + `?888P'  d88' d88'     ` + "`" + `?88P'` + "`" + `88b` + "`" + `?88P'` + "`" + `88b
                              )88          
                             ,88P          
                         ` + "`" + `?8888P           
`,
	}

	selectedBanner := banners[rand.Intn(len(banners))]

	// Build version info
	// Shorten Git commit hash
	shortCommit := GitCommit
	if len(GitCommit) > 7 {
		shortCommit = GitCommit[:7]
	}

	versionInfo := fmt.Sprintf(`
       Virga Development Build v%s
         Git: %s | Built: %s
`, Version, shortCommit, BuildDate)

	fullBanner := selectedBanner + versionInfo

	// Apply colorization based on color mode
	fullBanner = colorize(ui.config.ColorScheme.Info, fullBanner, !ui.colorMode)
	fmt.Println(fullBanner)
}

// SetQuietMode enables or disables quiet mode
func (ui *UI) SetQuietMode(quiet bool) {
	ui.mu.Lock()
	defer ui.mu.Unlock()
	ui.quietMode = quiet
}

// SetColorMode enables or disables color output
func (ui *UI) SetColorMode(enableColor bool) {
	ui.mu.Lock()
	defer ui.mu.Unlock()
	ui.colorMode = enableColor
}

// SetCompleter sets the command completer
func (ui *UI) SetCompleter(completer Completer) {
	ui.mu.Lock()
	defer ui.mu.Unlock()
	ui.commandCompleter = completer

	// Update liner's completer
	if completer != nil {
		adapter := completion.NewLinerAdapter(completer)
		ui.liner.SetCompleter(adapter.Complete)
	} else {
		ui.liner.SetCompleter(ui.completer)
	}
}

// PrintInfo displays an information message.
func (ui *UI) PrintInfo(message string) {
	ui.mu.RLock()
	defer ui.mu.RUnlock()

	if ui.quietMode {
		return
	}

	prefix := colorize(ui.config.ColorScheme.Info, "[*]", !ui.colorMode)
	logger.Info(fmt.Sprintf("%s %s", prefix, message))
}

// PrintSuccess displays a success message.
func (ui *UI) PrintSuccess(message string) {
	ui.mu.RLock()
	defer ui.mu.RUnlock()

	if ui.quietMode {
		return
	}

	prefix := colorize(ui.config.ColorScheme.Success, "[+]", !ui.colorMode)
	logger.Info(fmt.Sprintf("%s %s", prefix, message))
}

// PrintWarning displays a warning message.
func (ui *UI) PrintWarning(message string) {
	ui.mu.RLock()
	defer ui.mu.RUnlock()

	if ui.quietMode {
		return
	}

	prefix := colorize(ui.config.ColorScheme.Warning, "[!]", !ui.colorMode)
	logger.Warn(fmt.Sprintf("%s %s", prefix, message))
}

// PrintError displays an error message.
func (ui *UI) PrintError(message string) {
	ui.mu.RLock()
	defer ui.mu.RUnlock()

	// Error messages are shown even in quiet mode
	prefix := colorize(ui.config.ColorScheme.Error, "[-]", !ui.colorMode)
	logger.Error(fmt.Sprintf("%s Error: %s", prefix, message))
}

// PrintDebug displays a debug message.
func (ui *UI) PrintDebug(message string) {
	ui.mu.RLock()
	defer ui.mu.RUnlock()

	if ui.quietMode {
		return
	}

	prefix := colorize(ui.config.ColorScheme.Debug, "[D]", !ui.colorMode)
	logger.Debug(fmt.Sprintf("%s %s", prefix, message))
}

// PrintInfof displays a formatted information message.
func (ui *UI) PrintInfof(format string, args ...interface{}) {
	ui.PrintInfo(fmt.Sprintf(format, args...))
}

// PrintSuccessf displays a formatted success message.
func (ui *UI) PrintSuccessf(format string, args ...interface{}) {
	ui.PrintSuccess(fmt.Sprintf(format, args...))
}

// PrintWarningf displays a formatted warning message.
func (ui *UI) PrintWarningf(format string, args ...interface{}) {
	ui.PrintWarning(fmt.Sprintf(format, args...))
}

// PrintErrorf displays a formatted error message.
func (ui *UI) PrintErrorf(format string, args ...interface{}) {
	ui.PrintError(fmt.Sprintf(format, args...))
}

// PrintSessionInfo displays the session information.
func (ui *UI) PrintSessionInfo(sessionID string, info map[string]interface{}) {
	ui.mu.RLock()
	defer ui.mu.RUnlock()

	header := colorize(ui.config.ColorScheme.Info, fmt.Sprintf("\nSession Information (%s):", sessionID), !ui.colorMode)
	logger.Info(header)
	logger.Info(string(repeatChar('-', 60)))

	// Display basic information
	if hostname, ok := info["hostname"].(string); ok {
		logger.Info(fmt.Sprintf("Hostname:      %s", hostname))
	}

	if username, ok := info["username"].(string); ok {
		logger.Info(fmt.Sprintf("Username:      %s", username))
	}

	if osInfo, ok := info["os"].(string); ok {
		logger.Info(fmt.Sprintf("OS:            %s", osInfo))
	}

	if ip, ok := info["ip"].(string); ok {
		logger.Info(fmt.Sprintf("IP Address:    %s", ip))
	}

	// Display admin status
	if isAdmin, ok := info["is_admin"].(bool); ok {
		adminStatus := "No"
		statusColor := ui.config.ColorScheme.Warning
		if isAdmin {
			adminStatus = "Yes"
			statusColor = ui.config.ColorScheme.Success
		}
		logger.Info(fmt.Sprintf("Admin:         %s", colorize(statusColor, adminStatus, !ui.colorMode)))
	}

	// Display security products
	if secProducts, ok := info["security_products"].([]interface{}); ok && len(secProducts) > 0 {
		logger.Info("\nSecurity Products:")
		for _, product := range secProducts {
			if productStr, ok := product.(string); ok {
				logger.Info(fmt.Sprintf("  - %s", colorize(ui.config.ColorScheme.Warning, productStr, !ui.colorMode)))
			}
		}
	}

	logger.Info("")
}

// PrintModuleHelp displays the help for a module.
func (ui *UI) PrintModuleHelp(moduleName string, description string, options map[string]string) {
	ui.mu.RLock()
	defer ui.mu.RUnlock()

	moduleHeader := colorize(ui.config.ColorScheme.Info, fmt.Sprintf("\nModule: %s", moduleName), !ui.colorMode)
	logger.Info(moduleHeader)
	logger.Info(description)
	logger.Info("\nOptions:")

	maxKeyLen := 0
	for key := range options {
		if len(key) > maxKeyLen {
			maxKeyLen = len(key)
		}
	}

	format := fmt.Sprintf("  %%-%ds  %%s\n", maxKeyLen+2)

	for key, desc := range options {
		keyColored := colorize(ui.config.ColorScheme.Success, key, !ui.colorMode)
		logger.Info(fmt.Sprintf(format, keyColored, desc))
	}

	logger.Info("")
}

// PrintTable displays data in a table format.
func (ui *UI) PrintTable(headers []string, rows [][]string) {
	ui.mu.RLock()
	defer ui.mu.RUnlock()

	// Calculate the maximum width for each column
	colWidths := make([]int, len(headers))
	for i, header := range headers {
		colWidths[i] = len(header)
	}

	for _, row := range rows {
		for i, cell := range row {
			if i < len(colWidths) && len(cell) > colWidths[i] {
				colWidths[i] = len(cell)
			}
		}
	}

	// Display the header row
	var headerLine strings.Builder
	for i, header := range headers {
		format := fmt.Sprintf("%%-%ds", colWidths[i]+2)
		headerColored := colorize(ui.config.ColorScheme.Info+StyleBold, header, !ui.colorMode)
		headerLine.WriteString(fmt.Sprintf(format, headerColored))
	}
	logger.Info(headerLine.String())

	// Display the separator line
	var separatorLine strings.Builder
	for _, width := range colWidths {
		separatorLine.WriteString(string(repeatChar('-', width+2)))
	}
	logger.Info(separatorLine.String())

	// Display the data rows
	for _, row := range rows {
		var rowLine strings.Builder
		for i, cell := range row {
			if i < len(colWidths) {
				format := fmt.Sprintf("%%-%ds", colWidths[i]+2)
				rowLine.WriteString(fmt.Sprintf(format, cell))
			}
		}
		logger.Info(rowLine.String())
	}
}

// PromptYesNo displays a Yes/No confirmation prompt.
func (ui *UI) PromptYesNo(prompt string) bool {
	ui.mu.RLock()
	defer ui.mu.RUnlock()

	promptColored := colorize(ui.config.ColorScheme.Warning, prompt, !ui.colorMode)
	// For interactive prompts, we still need to use fmt.Printf
	fmt.Printf("%s [y/N]: ", promptColored)

	var response string
	fmt.Scanln(&response)

	return response == "y" || response == "Y" || response == "yes" || response == "Yes"
}

// Confirm is an alias for PromptYesNo with a default parameter
func (ui *UI) Confirm(message string, defaultYes bool) bool {
	ui.mu.RLock()
	defer ui.mu.RUnlock()

	defaultStr := "y/N"
	if defaultYes {
		defaultStr = "Y/n"
	}

	promptColored := colorize(ui.config.ColorScheme.Warning, message, !ui.colorMode)
	// For interactive prompts, we still need to use fmt.Printf
	fmt.Printf("%s [%s]: ", promptColored, defaultStr)

	var response string
	fmt.Scanln(&response)

	response = strings.ToLower(strings.TrimSpace(response))
	if response == "" {
		return defaultYes
	}

	return response == "y" || response == "yes"
}

// ClearScreen clears the terminal screen
func (ui *UI) ClearScreen() {
	if runtime.GOOS == "windows" {
		cmd := exec.Command("cmd", "/c", "cls")
		cmd.Stdout = os.Stdout
		cmd.Run()
	} else {
		// For terminal control sequences, we need to use fmt.Print
		fmt.Print("\033[2J\033[H")
	}
}

// StartCommand starts tracking a new command for extended history
func (ui *UI) StartCommand(command string, sessionID string, sessionInfo map[string]interface{}) {
	ui.mu.Lock()
	defer ui.mu.Unlock()

	if ui.extendedHistoryWriter != nil {
		ui.extendedHistoryWriter.StartCommand(command, sessionID, sessionInfo)
	}
}

// SetCommandResult sets the result of the current command
func (ui *UI) SetCommandResult(result string, success bool) {
	ui.mu.Lock()
	defer ui.mu.Unlock()

	if ui.extendedHistoryWriter != nil {
		ui.extendedHistoryWriter.SetResult(result, success)
	}
}

// EndCommand completes the current command tracking
func (ui *UI) EndCommand() error {
	ui.mu.Lock()
	defer ui.mu.Unlock()

	if ui.extendedHistoryWriter != nil {
		return ui.extendedHistoryWriter.EndCommand()
	}
	return nil
}

// repeatChar repeats the specified character the specified number of times.
func repeatChar(char byte, count int) []byte {
	result := make([]byte, count)
	for i := 0; i < count; i++ {
		result[i] = char
	}
	return result
}
