//go:build windows
// +build windows

package ps

import (
	"fmt"
	"os/exec"
	"strings"
)

// getProcessList returns a list of running processes on Windows
func getProcessList() (string, error) {
	// Use wmic to get process information
	cmd := exec.Command("wmic", "process", "get", "ProcessId,ParentProcessId,Name,CommandLine,WorkingSetSize,PageFileUsage", "/format:csv")
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Fallback to tasklist if wmic fails
		return getProcessListTasklist()
	}

	// Parse CSV output
	lines := strings.Split(string(output), "\n")
	if len(lines) < 3 { // Should have header + data
		return getProcessListTasklist()
	}

	var result strings.Builder
	result.WriteString("PID\tPPID\tNAME\tMEMORY\tCOMMAND\n")
	result.WriteString(strings.Repeat("-", 80) + "\n")

	// Skip empty first line and header
	for i := 2; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}

		// Parse CSV: Node,CommandLine,Name,PageFileUsage,ParentProcessId,ProcessId,WorkingSetSize
		fields := strings.Split(line, ",")
		if len(fields) >= 7 {
			pid := fields[5]
			ppid := fields[4]
			name := fields[2]
			workingSet := fields[6]
			cmdLine := fields[1]

			// Convert memory from bytes to KB
			var memKB string
			if workingSet != "" {
				if bytes, err := fmt.Sscanf(workingSet, "%d", &memKB); err == nil && bytes > 0 {
					memKB = fmt.Sprintf("%d KB", bytes/1024)
				} else {
					memKB = "N/A"
				}
			} else {
				memKB = "N/A"
			}

			result.WriteString(fmt.Sprintf("%s\t%s\t%s\t%s\t%s\n",
				pid, ppid, name, memKB, cmdLine))
		}
	}

	return result.String(), nil
}

// getProcessListTasklist uses tasklist as fallback
func getProcessListTasklist() (string, error) {
	cmd := exec.Command("tasklist", "/V", "/FO", "CSV")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("tasklist command failed: %w", err)
	}

	// Parse CSV output
	lines := strings.Split(string(output), "\n")
	if len(lines) < 2 {
		return "", fmt.Errorf("no process information available")
	}

	var result strings.Builder
	result.WriteString("PID\tPPID\tNAME\tMEMORY\tUSER\n")
	result.WriteString(strings.Repeat("-", 80) + "\n")

	// Skip header
	for i := 1; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}

		// Parse CSV fields
		fields := parseCSVLine(line)
		if len(fields) >= 7 {
			name := strings.Trim(fields[0], "\"")
			pid := strings.Trim(fields[1], "\"")
			// session := strings.Trim(fields[2], "\"")
			memory := strings.Trim(fields[4], "\"")
			user := strings.Trim(fields[6], "\"")

			result.WriteString(fmt.Sprintf("%s\t-\t%s\t%s\t%s\n",
				pid, name, memory, user))
		}
	}

	return result.String(), nil
}

// parseCSVLine parses a CSV line handling quoted fields
func parseCSVLine(line string) []string {
	var fields []string
	var current strings.Builder
	inQuotes := false

	for _, r := range line {
		if r == '"' {
			inQuotes = !inQuotes
		} else if r == ',' && !inQuotes {
			fields = append(fields, current.String())
			current.Reset()
		} else {
			current.WriteRune(r)
		}
	}

	if current.Len() > 0 {
		fields = append(fields, current.String())
	}

	return fields
}
