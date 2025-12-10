package command

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/r74tech/virga/internal/cli/session"
	"github.com/r74tech/virga/internal/shared/logger"
)

// validateLocalPath validates a local file path to prevent directory traversal attacks
func validateLocalPath(path string) error {
	if path == "" {
		return fmt.Errorf("path cannot be empty")
	}

	// Clean the path
	cleaned := filepath.Clean(path)

	// Check for path traversal attempts in original path first
	if strings.Contains(path, "..") {
		absPath, err := filepath.Abs(path)
		if err != nil {
			return fmt.Errorf("invalid path: %w", err)
		}
		cwd, err := filepath.Abs(".")
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
		rel, err := filepath.Rel(cwd, absPath)
		if err != nil || strings.HasPrefix(rel, "..") {
			return fmt.Errorf("directory traversal detected: %s", path)
		}
	}

	// Also check cleaned path
	if strings.Contains(cleaned, "..") {
		// Check if it's a legitimate relative path by converting to absolute
		abs, err := filepath.Abs(cleaned)
		if err != nil {
			return fmt.Errorf("invalid path: %w", err)
		}

		// Ensure the absolute path doesn't escape the current working directory
		// if we want to restrict to current directory
		cwd, err := filepath.Abs(".")
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}

		// Allow paths that don't escape the current directory tree
		if !strings.HasPrefix(abs, cwd) && !isAllowedSystemPath(abs) {
			return fmt.Errorf("directory traversal detected: %s", path)
		}
	}

	// Reject null bytes
	if strings.Contains(path, "\x00") {
		return fmt.Errorf("path contains null character")
	}

	// On Windows, check for device names
	if isWindowsReservedName(filepath.Base(cleaned)) {
		return fmt.Errorf("reserved device name")
	}

	// On Windows, check for invalid characters
	if runtime.GOOS == "windows" {
		invalidChars := []string{"<", ">", ":", "\"", "|", "?", "*"}
		for _, char := range invalidChars {
			if strings.Contains(path, char) && !strings.HasPrefix(path, "C:") && !strings.HasPrefix(path, "D:") {
				// Allow drive letters with colons
				if char == ":" && len(path) > 2 && path[1] == ':' && strings.Contains(path[2:], char) {
					return fmt.Errorf("invalid character in path")
				} else if char != ":" {
					return fmt.Errorf("invalid character in path")
				}
			}
		}
	}

	return nil
}

// isAllowedSystemPath checks if a path is in an allowed system directory
func isAllowedSystemPath(path string) bool {
	// Define allowed directories outside CWD (e.g., home directory, temp)
	home, err := os.UserHomeDir()
	if err == nil && strings.HasPrefix(path, home) {
		return true
	}

	// Allow temp directory
	if strings.HasPrefix(path, filepath.Join("/tmp")) ||
		strings.HasPrefix(path, filepath.Join("/var/tmp")) {
		return true
	}

	return false
}

// isWindowsReservedName checks if a filename is a reserved Windows device name
func isWindowsReservedName(name string) bool {
	reserved := []string{
		"CON", "PRN", "AUX", "NUL",
		"COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9",
		"LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9",
	}

	nameUpper := strings.ToUpper(strings.TrimSuffix(name, filepath.Ext(name)))
	for _, r := range reserved {
		if nameUpper == r {
			return true
		}
	}

	return false
}

// sanitizeShellCommand performs basic sanitization of shell commands
// Note: This is basic sanitization. The actual command execution should be done
// securely on the server side using proper exec functions, not shell interpretation
func sanitizeShellCommand(cmd string) string {
	// Remove null bytes
	cmd = strings.ReplaceAll(cmd, "\x00", "")

	// Trim whitespace
	cmd = strings.TrimSpace(cmd)

	return cmd
}

// validateShellCommand validates a shell command for obvious malicious patterns
// This is for client-side validation only - server must do proper validation
func validateShellCommand(cmd string) error {
	if cmd == "" {
		return fmt.Errorf("command cannot be empty")
	}

	// Check for null bytes
	if strings.Contains(cmd, "\x00") {
		return fmt.Errorf("command contains null byte")
	}

	// Warn about potentially dangerous patterns (but don't block - this is a C2 tool)
	dangerousPatterns := []struct {
		pattern string
		warning string
	}{
		{`rm\s+-rf\s+/`, "WARNING: Command appears to delete system root"},
		{`:(){ :|:& };:`, "WARNING: Fork bomb detected"},
		{`>\s*/dev/sda`, "WARNING: Direct disk write detected"},
		{`dd\s+if=/dev/zero\s+of=/dev/`, "WARNING: Disk wipe pattern detected"},
	}

	for _, dp := range dangerousPatterns {
		if matched, _ := regexp.MatchString(dp.pattern, cmd); matched {
			logger.Info("\n%s", dp.warning)
			logger.Info("Are you sure you want to execute this command? (y/N): ")
			var response string
			fmt.Scanln(&response)
			if strings.ToLower(response) != "y" && strings.ToLower(response) != "yes" {
				return fmt.Errorf("command execution cancelled by user")
			}
		}
	}

	return nil
}

// validateFilename validates a filename (not full path)
func validateFilename(name string) error {
	if name == "" {
		return fmt.Errorf("filename cannot be empty")
	}

	// Check for directory separators in filename
	if strings.ContainsAny(name, "/\\") {
		return fmt.Errorf("filename cannot contain path separators")
	}

	// Check for null bytes
	if strings.Contains(name, "\x00") {
		return fmt.Errorf("filename contains null byte")
	}

	// Check length
	if len(name) > 255 {
		return fmt.Errorf("filename too long (max 255 characters)")
	}

	// Check for Windows reserved names
	if isWindowsReservedName(name) {
		return fmt.Errorf("filename is a reserved Windows device name")
	}

	return nil
}

// validateRemotePath validates a remote file path
func validateRemotePath(path string) error {
	if path == "" {
		return fmt.Errorf("path cannot be empty")
	}

	// Check for null bytes
	if strings.Contains(path, "\x00") {
		return fmt.Errorf("path contains null character")
	}

	// Check path length
	if len(path) > 4096 {
		return fmt.Errorf("path too long (max 4096 characters)")
	}

	return nil
}

// validatePort validates a port number
func validatePort(port int) error {
	if port < 1 || port > 65535 {
		return fmt.Errorf("invalid port number: %d (must be 1-65535)", port)
	}
	return nil
}

// validateCommand validates a command for dangerous patterns
func validateCommand(cmd string) error {
	if cmd == "" {
		return fmt.Errorf("command cannot be empty")
	}

	// Check for dangerous characters that could lead to command injection
	dangerousChars := []string{";", "&", "|", "`", "$", "(", ")", "{", "}", "<", ">", "\n", "\r"}
	for _, char := range dangerousChars {
		if strings.Contains(cmd, char) {
			return fmt.Errorf("command contains dangerous characters")
		}
	}

	return nil
}

// sanitizeCommand removes dangerous characters from command and arguments
func sanitizeCommand(command string, args []string) (string, []string) {
	// Characters to remove
	dangerous := []string{";", "&", "|", "`", "$", "(", ")", "{", "}", "[", "]", "<", ">", "\n", "\r", "\x00"}

	// Sanitize command
	for _, char := range dangerous {
		command = strings.ReplaceAll(command, char, "")
	}

	// Sanitize arguments
	sanitizedArgs := make([]string, len(args))
	for i, arg := range args {
		sanitized := arg
		for _, char := range dangerous {
			sanitized = strings.ReplaceAll(sanitized, char, "")
		}
		sanitizedArgs[i] = sanitized
	}

	return command, sanitizedArgs
}

// SecurePathJoin safely joins path elements
func SecurePathJoin(elem ...string) (string, error) {
	// Join the path elements
	joined := filepath.Join(elem...)

	// Clean the path
	cleaned := filepath.Clean(joined)

	// Validate the result
	if err := validateLocalPath(cleaned); err != nil {
		return "", err
	}

	return cleaned, nil
}

// KillswitchCommand is the implementation of the 'killswitch' command.
type KillswitchCommand struct{}

// Execute executes the 'killswitch' command.
func (c *KillswitchCommand) Execute(sessionManager *session.Manager, args []string) error {
	// Check the current session
	currentSession := sessionManager.GetCurrentSession()
	if currentSession == nil {
		return fmt.Errorf("no session selected. Use 'interact <session_id>' first")
	}

	// Parse arguments
	mode := "shutdown" // Default mode
	message := ""

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--mode", "-m":
			if i+1 >= len(args) {
				return fmt.Errorf("--mode requires an argument (shutdown or remove)")
			}
			mode = args[i+1]
			i++
		case "--message", "-msg":
			if i+1 >= len(args) {
				return fmt.Errorf("--message requires an argument")
			}
			message = args[i+1]
			i++
		default:
			// If no flag, treat as message
			if mode == "shutdown" && message == "" {
				message = strings.Join(args[i:], " ")
				break
			}
		}
	}

	// Validate mode
	if mode != "shutdown" && mode != "remove" {
		return fmt.Errorf("invalid mode: %s (must be 'shutdown' or 'remove')", mode)
	}

	// Display warning
	logger.Info("\n" + strings.Repeat("=", 70))
	logger.Info("                       KILLSWITCH WARNING")
	logger.Info(strings.Repeat("=", 70))
	logger.Info("You are about to activate the killswitch for session: %s", currentSession.ID)
	logger.Info("")
	logger.Info("Mode: %s", mode)
	if mode == "remove" {
		logger.Info("  - The implant will PERMANENTLY DELETE itself from the target system")
		logger.Info("  - This action CANNOT BE UNDONE")
	} else {
		logger.Info("  - The implant will shut down cleanly")
		logger.Info("  - The binary file will remain on the target system")
	}
	logger.Info("")
	if message != "" {
		logger.Info("Message: %s", message)
		logger.Info("")
	}
	logger.Info("Session Info:")
	if info := currentSession.Information; info != nil {
		if hostname, ok := info["hostname"].(string); ok {
			logger.Info("  Hostname: %s", hostname)
		}
		if username, ok := info["username"].(string); ok {
			logger.Info("  Username: %s", username)
		}
		if osInfo, ok := info["os"].(string); ok {
			logger.Info("  OS: %s", osInfo)
		}
	}
	logger.Info(strings.Repeat("=", 70))
	logger.Info("")

	// Confirmation prompt
	logger.Info("Type 'YES' (in uppercase) to confirm killswitch activation: ")
	reader := bufio.NewReader(os.Stdin)
	confirmation, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read confirmation: %v", err)
	}
	confirmation = strings.TrimSpace(confirmation)

	if confirmation != "YES" {
		logger.Info("Killswitch activation cancelled.")
		return nil
	}

	// Send the killswitch task via API
	apiClient := sessionManager.GetAPIClient()
	if apiClient == nil {
		return fmt.Errorf("API client not available")
	}

	// Build the task payload
	payload := map[string]interface{}{
		"mode":    mode,
		"message": message,
		"confirm": true,
	}

	// Send killswitch task
	taskID, err := apiClient.SendSessionCommand(currentSession.ID, "killswitch", payload)
	if err != nil {
		return fmt.Errorf("failed to send killswitch task: %v", err)
	}

	logger.Info("\nKillswitch task sent successfully! (Task ID: %s)", taskID)
	logger.Info("The implant will terminate shortly.")
	if mode == "remove" {
		logger.Info("The binary file will be removed from the target system.")
	}
	logger.Info("\nNote: The session will be disconnected within a few seconds.")

	return nil
}

// Help returns the help message for the 'killswitch' command.
func (c *KillswitchCommand) Help() string {
	return `Activate the killswitch to terminate the implant.

Usage: killswitch [options]

Options:
  --mode, -m <mode>       Killswitch mode (default: shutdown)
                          - shutdown: Stop the implant cleanly
                          - remove: Delete the implant binary and stop
  --message, -msg <text>  Optional message for audit log

Modes:
  shutdown  - Terminates the implant process cleanly. The binary file
              remains on the target system.

  remove    - Terminates the implant process AND deletes the binary file
              from the target system. This action is PERMANENT and cannot
              be undone.

Examples:
  killswitch
  killswitch --mode shutdown
  killswitch --mode remove --message "Security incident response"
  killswitch -m remove -msg "End of operation"

IMPORTANT:
  - This command requires explicit confirmation
  - Once activated, the implant will terminate within seconds
  - In 'remove' mode, the binary will be permanently deleted
  - The session will be disconnected and cannot be recovered

Note: This command requires an active session. Use 'interact <session_id>' first.`
}
