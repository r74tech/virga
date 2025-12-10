//go:build windows
// +build windows

package netstat

import (
	"fmt"
	"os/exec"
	"strings"
)

// getNetworkConnections returns network connections on Windows
func getNetworkConnections() (string, error) {
	// Use netstat with options to show all connections and PIDs
	cmd := exec.Command("netstat", "-ano")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("netstat command failed: %w", err)
	}

	// Parse output
	lines := strings.Split(string(output), "\n")
	var result strings.Builder

	result.WriteString("PROTO\tLOCAL ADDRESS\tFOREIGN ADDRESS\tSTATE\tPID\n")
	result.WriteString(strings.Repeat("-", 80) + "\n")

	// Skip header lines
	inData := false
	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Skip empty lines and headers
		if line == "" {
			continue
		}

		// Look for the start of actual data
		if strings.HasPrefix(line, "Proto") {
			inData = true
			continue
		}

		if !inData {
			continue
		}

		// Parse netstat output
		fields := strings.Fields(line)
		if len(fields) >= 4 {
			proto := fields[0]
			local := fields[1]
			foreign := fields[2]

			// TCP has state and PID
			if strings.HasPrefix(proto, "TCP") && len(fields) >= 5 {
				state := fields[3]
				pid := fields[4]
				result.WriteString(fmt.Sprintf("%s\t%s\t%s\t%s\t%s\n",
					proto, local, foreign, state, pid))
			} else if strings.HasPrefix(proto, "UDP") && len(fields) >= 4 {
				// UDP has no state
				pid := fields[3]
				result.WriteString(fmt.Sprintf("%s\t%s\t%s\t-\t%s\n",
					proto, local, foreign, pid))
			}
		}
	}

	// Try to get process names for PIDs
	enhancedOutput := enhanceWithProcessNames(result.String())
	if enhancedOutput != "" {
		return enhancedOutput, nil
	}

	return result.String(), nil
}

// enhanceWithProcessNames tries to add process names to PIDs
func enhanceWithProcessNames(netstatOutput string) string {
	// Try using netstat -anob (requires admin)
	cmd := exec.Command("netstat", "-anob")
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Not admin or command failed
		return ""
	}

	// Parse the enhanced output
	lines := strings.Split(string(output), "\n")
	var result strings.Builder

	result.WriteString("PROTO\tLOCAL ADDRESS\tFOREIGN ADDRESS\tSTATE\tPID\tPROCESS\n")
	result.WriteString(strings.Repeat("-", 100) + "\n")

	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" || strings.HasPrefix(line, "Active") || strings.HasPrefix(line, "Proto") {
			continue
		}

		// Parse connection info
		fields := strings.Fields(line)
		if len(fields) >= 4 {
			proto := fields[0]
			if strings.HasPrefix(proto, "TCP") || strings.HasPrefix(proto, "UDP") {
				local := fields[1]
				foreign := fields[2]

				state := "-"
				pid := "-"
				process := "-"

				if strings.HasPrefix(proto, "TCP") && len(fields) >= 5 {
					state = fields[3]
					pid = fields[4]
				} else if strings.HasPrefix(proto, "UDP") && len(fields) >= 4 {
					pid = fields[3]
				}

				// Next line might contain process name in brackets
				if i+1 < len(lines) {
					nextLine := strings.TrimSpace(lines[i+1])
					if strings.HasPrefix(nextLine, "[") && strings.HasSuffix(nextLine, "]") {
						process = strings.Trim(nextLine, "[]")
						i++ // Skip the process line
					}
				}

				result.WriteString(fmt.Sprintf("%s\t%s\t%s\t%s\t%s\t%s\n",
					proto, local, foreign, state, pid, process))
			}
		}
	}

	return result.String()
}
