//go:build darwin || linux
// +build darwin linux

package sysinfo

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

// getDetailedSystemInfo returns platform-specific system information for Unix systems
func getDetailedSystemInfo() (string, error) {
	var result strings.Builder

	// Get OS specific info
	if runtime.GOOS == "darwin" {
		// macOS specific
		if output, err := exec.Command("sw_vers").CombinedOutput(); err == nil {
			result.WriteString("\nmacOS Version:\n")
			result.WriteString(string(output))
		}

		// Add kernel info
		if output, err := exec.Command("uname", "-a").CombinedOutput(); err == nil {
			result.WriteString(fmt.Sprintf("\nKernel: %s", strings.TrimSpace(string(output))))
		}

		if output, err := exec.Command("sysctl", "-n", "hw.model").CombinedOutput(); err == nil {
			result.WriteString(fmt.Sprintf("\nHardware Model: %s", strings.TrimSpace(string(output))))
		}

		if output, err := exec.Command("sysctl", "-n", "hw.memsize").CombinedOutput(); err == nil {
			result.WriteString(fmt.Sprintf("\nMemory: %s", strings.TrimSpace(string(output))))
		}

		// CPU info
		if output, err := exec.Command("sysctl", "-n", "machdep.cpu.brand_string").CombinedOutput(); err == nil {
			result.WriteString(fmt.Sprintf("\nCPU: %s", strings.TrimSpace(string(output))))
		}
	} else {
		// Linux specific
		if output, err := exec.Command("uname", "-a").CombinedOutput(); err == nil {
			result.WriteString(fmt.Sprintf("\nKernel: %s", strings.TrimSpace(string(output))))
		}

		// Try to read /etc/os-release
		if output, err := exec.Command("cat", "/etc/os-release").CombinedOutput(); err == nil {
			lines := strings.Split(string(output), "\n")
			for _, line := range lines {
				if strings.HasPrefix(line, "PRETTY_NAME=") {
					name := strings.Trim(strings.TrimPrefix(line, "PRETTY_NAME="), "\"")
					result.WriteString(fmt.Sprintf("\nDistribution: %s", name))
					break
				}
			}
		}

		// CPU info
		if output, err := exec.Command("lscpu").CombinedOutput(); err == nil {
			result.WriteString("\n\nCPU Information:\n")
			result.WriteString(string(output))
		}

		// Memory info
		if output, err := exec.Command("free", "-h").CombinedOutput(); err == nil {
			result.WriteString("\n\nMemory:\n")
			result.WriteString(string(output))
		}

		// Disk info
		if output, err := exec.Command("df", "-h").CombinedOutput(); err == nil {
			result.WriteString("\n\nDisk Usage:\n")
			result.WriteString(string(output))
		}
	}

	// Common Unix commands
	if output, err := exec.Command("uptime").CombinedOutput(); err == nil {
		result.WriteString(fmt.Sprintf("\n\nUptime: %s", strings.TrimSpace(string(output))))
	}

	// Current users
	if output, err := exec.Command("who").CombinedOutput(); err == nil {
		result.WriteString("\n\nLogged in users:\n")
		result.WriteString(string(output))
	}

	return result.String(), nil
}
