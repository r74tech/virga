package command

import (
	"fmt"
	"strings"
	"time"

	"github.com/r74tech/virga/internal/cli/session"
	"github.com/r74tech/virga/internal/shared/logger"
)

// InfoCommand is the implementation of the 'info' command.
type InfoCommand struct{}

// Execute executes the 'info' command.
func (c *InfoCommand) Execute(sessionManager *session.Manager, args []string) error {
	// Check the current session
	currentSession := sessionManager.GetCurrentSession()
	if currentSession == nil {
		return fmt.Errorf("no session selected. Use 'interact <session_id>' first")
	}

	// Get session information
	info := currentSession.GetInformation()

	// Output format
	logger.Info("Session Information:")
	logger.Info(strings.Repeat("-", 80))
	logger.Info("Session ID: %s", currentSession.ID)
	logger.Info("Connection: %s", currentSession.RemoteAddr())
	logger.Info("First Seen: %s", currentSession.FirstSeen().Format("2006-01-02 15:04:05"))
	logger.Info("Last Seen: %s", currentSession.LastSeen().Format("2006-01-02 15:04:05"))
	logger.Info("Uptime: %s", formatDuration(time.Since(currentSession.FirstSeen())))
	logger.Info("")

	logger.Info("System Information:")
	logger.Info(strings.Repeat("-", 80))

	// Display system information
	if hostname, ok := info["hostname"].(string); ok {
		logger.Info("Hostname: %s", hostname)
	}

	if username, ok := info["username"].(string); ok {
		logger.Info("Username: %s", username)
	}

	if osInfo, ok := info["os"].(string); ok {
		logger.Info("OS: %s", osInfo)
	}

	if arch, ok := info["arch"].(string); ok {
		logger.Info("Architecture: %s", arch)
	}

	if pid, ok := info["pid"].(float64); ok {
		logger.Info("PID: %d", int(pid))
	}

	if elevated, ok := info["is_elevated"].(bool); ok {
		elevatedStr := "No"
		if elevated {
			elevatedStr = "Yes"
		}
		logger.Info("Elevated: %s", elevatedStr)
	}

	// Network information
	logger.Info("")
	logger.Info("Network Information:")
	logger.Info(strings.Repeat("-", 80))

	if ipAddress, ok := info["ip_address"].(string); ok {
		logger.Info("IP Address: %s", ipAddress)
	}

	if macAddress, ok := info["mac_address"].(string); ok {
		logger.Info("MAC Address: %s", macAddress)
	}

	// Beacon configuration
	logger.Info("")
	logger.Info("Beacon Configuration:")
	logger.Info(strings.Repeat("-", 80))

	if sleepTime, ok := info["sleep_time"].(float64); ok {
		logger.Info("Sleep: %d seconds", int(sleepTime))
	}

	if jitter, ok := info["jitter"].(float64); ok {
		logger.Info("Jitter: %d%%", int(jitter))
	}

	if c2, ok := info["c2_server"].(string); ok {
		logger.Info("C2 Server: %s", c2)
	}

	return nil
}

// Help returns the help for the 'info' command.
func (c *InfoCommand) Help() string {
	return "Display detailed information about the current session.\n" +
		"Usage: info\n" +
		"Note: This command requires an active session. Use 'interact <session_id>' first."
}

// SysInfoCommand is the implementation of the 'sysinfo' command.
type SysInfoCommand struct{}

// Execute executes the 'sysinfo' command.
func (c *SysInfoCommand) Execute(sessionManager *session.Manager, args []string) error {
	// Check the current session
	currentSession := sessionManager.GetCurrentSession()
	if currentSession == nil {
		return fmt.Errorf("no session selected. Use 'interact <session_id>' first")
	}

	// Send the system information collection request
	taskID, err := currentSession.ExecuteCommand("sysinfo", map[string]interface{}{})
	if err != nil {
		return fmt.Errorf("error sending sysinfo request: %v", err)
	}

	logger.Info("Gathering system information...")

	// Wait for the result
	result, err := waitForTaskResult(currentSession, taskID, 30)
	if err != nil {
		return fmt.Errorf("error waiting for sysinfo result: %v", err)
	}

	// Display the result
	logger.Info("System Information:")
	logger.Info(strings.Repeat("-", 80))
	logger.Info(result.Output)

	return nil
}

// Help returns the help for the 'sysinfo' command.
func (c *SysInfoCommand) Help() string {
	return "Collect detailed system information from the target.\n" +
		"Usage: sysinfo\n" +
		"Note: This command requires an active session. Use 'interact <session_id>' first."
}

// NetworkInfoCommand is the implementation of the 'netinfo' command.
type NetworkInfoCommand struct{}

// Execute executes the 'netinfo' command.
func (c *NetworkInfoCommand) Execute(sessionManager *session.Manager, args []string) error {
	// Check the current session
	currentSession := sessionManager.GetCurrentSession()
	if currentSession == nil {
		return fmt.Errorf("no session selected. Use 'interact <session_id>' first")
	}

	// Send the network information collection request
	taskID, err := currentSession.ExecuteCommand("netinfo", map[string]interface{}{})
	if err != nil {
		return fmt.Errorf("error sending netinfo request: %v", err)
	}

	logger.Info("Gathering network information...")

	// Wait for the result
	result, err := waitForTaskResult(currentSession, taskID, 30)
	if err != nil {
		return fmt.Errorf("error waiting for netinfo result: %v", err)
	}

	// Display the result
	logger.Info("Network Information:")
	logger.Info(strings.Repeat("-", 80))
	logger.Info(result.Output)

	return nil
}

// Help returns the help for the 'netinfo' command.
func (c *NetworkInfoCommand) Help() string {
	return "Collect network information from the target system.\n" +
		"Usage: netinfo\n" +
		"Note: This command requires an active session. Use 'interact <session_id>' first."
}

// ProcessListCommand is the implementation of the 'ps' command.
type ProcessListCommand struct{}

// Execute executes the 'ps' command.
func (c *ProcessListCommand) Execute(sessionManager *session.Manager, args []string) error {
	// Check the current session
	currentSession := sessionManager.GetCurrentSession()
	if currentSession == nil {
		return fmt.Errorf("no session selected. Use 'interact <session_id>' first")
	}

	// Send the process list acquisition request
	taskID, err := currentSession.ExecuteCommand("ps", map[string]interface{}{})
	if err != nil {
		return fmt.Errorf("error sending process list request: %v", err)
	}

	logger.Info("Gathering process information...")

	// Wait for the result
	result, err := waitForTaskResult(currentSession, taskID, 30)
	if err != nil {
		return fmt.Errorf("error waiting for process list result: %v", err)
	}

	// Display the result
	logger.Info("Process List:")
	logger.Info(strings.Repeat("-", 80))
	logger.Info(result.Output)

	return nil
}

// Help returns the help for the 'ps' command.
func (c *ProcessListCommand) Help() string {
	return "Display the process list on the target system.\n" +
		"Usage: ps\n" +
		"Note: This command requires an active session. Use 'interact <session_id>' first."
}
