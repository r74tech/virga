//go:build darwin || linux
// +build darwin linux

package ps

import (
	"fmt"
	"os/exec"
	"strings"
)

// getProcessList returns a list of running processes on Unix systems
func getProcessList() (string, error) {
	// Use ps command with standard format
	cmd := exec.Command("ps", "aux")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("ps command failed: %w", err)
	}

	// Parse output to create structured result
	lines := strings.Split(string(output), "\n")
	if len(lines) == 0 {
		return "", fmt.Errorf("no process information available")
	}

	var result strings.Builder
	result.WriteString("PID  PPID  USER  CPU%  MEM%  COMMAND\n")
	result.WriteString(strings.Repeat("-", 80) + "\n")

	// Skip header line
	for i := 1; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}

		// Parse ps output (USER PID %CPU %MEM VSZ RSS TTY STAT START TIME COMMAND)
		fields := strings.Fields(line)
		if len(fields) >= 11 {
			user := fields[0]
			pid := fields[1]
			cpu := fields[2]
			mem := fields[3]
			// Command is everything from field 10 onwards
			command := strings.Join(fields[10:], " ")

			// Note: ps aux doesn't show PPID, so we use "-" as placeholder
			result.WriteString(fmt.Sprintf("%s  -  %s  %s  %s  %s\n",
				pid, user, cpu, mem, command))
		}
	}

	return result.String(), nil
}
