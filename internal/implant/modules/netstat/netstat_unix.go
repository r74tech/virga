//go:build darwin || linux
// +build darwin linux

package netstat

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

// getNetworkConnections returns network connections on Unix systems
func getNetworkConnections() (string, error) {
	var cmd *exec.Cmd

	// Use different commands based on OS
	if runtime.GOOS == "darwin" {
		// macOS netstat
		cmd = exec.Command("netstat", "-anp", "tcp")
	} else {
		// Linux netstat or ss
		// Try ss first (newer, faster)
		if _, err := exec.LookPath("ss"); err == nil {
			cmd = exec.Command("ss", "-tupan")
		} else {
			// Fallback to netstat
			cmd = exec.Command("netstat", "-tupan")
		}
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		// If the command failed, try a simpler version
		if runtime.GOOS == "darwin" {
			cmd = exec.Command("netstat", "-an")
		} else {
			cmd = exec.Command("netstat", "-tun")
		}

		output, err = cmd.CombinedOutput()
		if err != nil {
			return "", fmt.Errorf("netstat command failed: %w", err)
		}
	}

	// Parse and format output
	lines := strings.Split(string(output), "\n")
	var result strings.Builder

	result.WriteString("PROTO\tLOCAL ADDRESS\tFOREIGN ADDRESS\tSTATE\tPID/PROGRAM\n")
	result.WriteString(strings.Repeat("-", 80) + "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "Active") || strings.HasPrefix(line, "Proto") {
			continue
		}

		// Parse netstat output
		fields := strings.Fields(line)
		if len(fields) >= 4 {
			proto := fields[0]
			local := fields[3]
			foreign := fields[4]

			state := "-"
			pid := "-"

			// TCP connections have state
			if strings.HasPrefix(proto, "tcp") && len(fields) >= 6 {
				state = fields[5]
				if len(fields) >= 7 {
					pid = fields[6]
				}
			}

			// For UDP, no state
			if strings.HasPrefix(proto, "udp") && len(fields) >= 5 {
				pid = fields[5]
			}

			result.WriteString(fmt.Sprintf("%s\t%s\t%s\t%s\t%s\n",
				proto, local, foreign, state, pid))
		}
	}

	return result.String(), nil
}
