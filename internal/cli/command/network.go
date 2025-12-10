package command

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/r74tech/virga/internal/cli/session"
	"github.com/r74tech/virga/internal/shared/logger"
)

// NetstatCommand is the implementation of the 'netstat' command.
type NetstatCommand struct{}

// Execute executes the 'netstat' command.
func (c *NetstatCommand) Execute(sessionManager *session.Manager, args []string) error {
	// Check the current session
	currentSession := sessionManager.GetCurrentSession()
	if currentSession == nil {
		return fmt.Errorf("no session selected. Use 'interact <session_id>' first")
	}

	// Send the netstat request
	taskID, err := currentSession.ExecuteCommand("netstat", map[string]interface{}{})
	if err != nil {
		return fmt.Errorf("error sending netstat request: %v", err)
	}

	logger.Info("Gathering network connection information...")

	// Wait for the result
	result, err := waitForTaskResult(currentSession, taskID, 30)
	if err != nil {
		return fmt.Errorf("error waiting for netstat result: %v", err)
	}

	// Display the result
	if result.ExitCode == 0 {
		fmt.Println("Network Connections:")
		fmt.Println("--------------------------------------------------------------------------------")
		fmt.Println(result.Output)
	} else {
		if result.Error != "" {
			return fmt.Errorf("error: %s", result.Error)
		}
		return fmt.Errorf("command failed: %s", result.Output)
	}

	return nil
}

// Help returns the help message for the 'netstat' command.
func (c *NetstatCommand) Help() string {
	return "Display network connections.\n" +
		"Usage: netstat\n" +
		"Shows active network connections including protocol, local/remote addresses, and state.\n" +
		"Note: This command requires an active session. Use 'interact <session_id>' first."
}

// PortfwdCommand is the implementation of the 'portfwd' command.
type PortfwdCommand struct{}

// Execute executes the 'portfwd' command.
func (c *PortfwdCommand) Execute(sessionManager *session.Manager, args []string) error {
	// Check the current session
	currentSession := sessionManager.GetCurrentSession()
	if currentSession == nil {
		return fmt.Errorf("no session selected. Use 'interact <session_id>' first")
	}

	// Parse subcommand
	if len(args) < 1 {
		return fmt.Errorf("usage: portfwd <add|list|remove> [options]")
	}

	subcommand := strings.ToLower(args[0])

	// Build the args array for the module
	moduleArgs := []string{subcommand}

	switch subcommand {
	case "add":
		if len(args) < 3 {
			return fmt.Errorf("usage: portfwd add <local_port> <remote_host:remote_port>")
		}

		localPort, err := strconv.Atoi(args[1])
		if err != nil {
			return fmt.Errorf("invalid local port: %s", args[1])
		}

		remoteAddr := args[2]
		if !strings.Contains(remoteAddr, ":") {
			return fmt.Errorf("invalid remote address format. Use host:port")
		}

		// Split remote address into host and port
		parts := strings.Split(remoteAddr, ":")
		remoteHost := strings.Join(parts[:len(parts)-1], ":")
		remotePort, err := strconv.Atoi(parts[len(parts)-1])
		if err != nil {
			return fmt.Errorf("invalid remote port in address: %s", remoteAddr)
		}

		// Add arguments in the format expected by the module
		moduleArgs = append(moduleArgs, strconv.Itoa(localPort), remoteHost, strconv.Itoa(remotePort))

	case "remove":
		if len(args) < 2 {
			return fmt.Errorf("usage: portfwd remove <local_port>")
		}

		localPort, err := strconv.Atoi(args[1])
		if err != nil {
			return fmt.Errorf("invalid local port: %s", args[1])
		}

		moduleArgs = append(moduleArgs, strconv.Itoa(localPort))

	case "list":
		// No additional arguments needed

	default:
		return fmt.Errorf("unknown subcommand: %s. Use 'add', 'list', or 'remove'", subcommand)
	}

	payload := map[string]interface{}{
		"args": moduleArgs,
	}

	// Send the portfwd request
	taskID, err := currentSession.ExecuteCommand("portfwd", payload)
	if err != nil {
		return fmt.Errorf("error sending portfwd request: %v", err)
	}

	// Wait for the result
	result, err := waitForTaskResult(currentSession, taskID, 10)
	if err != nil {
		return fmt.Errorf("error waiting for portfwd result: %v", err)
	}

	// Display the result
	if result.ExitCode == 0 {
		fmt.Println(result.Output)
	} else {
		if result.Error != "" {
			return fmt.Errorf("error: %s", result.Error)
		}
		return fmt.Errorf("command failed: %s", result.Output)
	}

	return nil
}

// Help returns the help message for the 'portfwd' command.
func (c *PortfwdCommand) Help() string {
	return "Manage port forwarding.\n" +
		"Usage:\n" +
		"  portfwd add <local_port> <remote_host:remote_port>  - Add a port forward\n" +
		"  portfwd list                                        - List active port forwards\n" +
		"  portfwd remove <local_port>                         - Remove a port forward\n" +
		"\n" +
		"Examples:\n" +
		"  portfwd add 8080 192.168.1.10:80    - Forward local port 8080 to 192.168.1.10:80\n" +
		"  portfwd list                         - Show all active port forwards\n" +
		"  portfwd remove 8080                  - Remove forward on local port 8080\n" +
		"\n" +
		"Note: This command requires an active session. Use 'interact <session_id>' first."
}
