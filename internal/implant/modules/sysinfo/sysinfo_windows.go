//go:build windows
// +build windows

package sysinfo

import (
	"fmt"
	"os/exec"
	"strings"
)

// getDetailedSystemInfo returns platform-specific system information for Windows
func getDetailedSystemInfo() (string, error) {
	var result strings.Builder

	// Get Windows version
	if output, err := exec.Command("wmic", "os", "get", "Caption,Version,BuildNumber", "/value").CombinedOutput(); err == nil {
		lines := strings.Split(string(output), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "Caption=") {
				result.WriteString(fmt.Sprintf("\nOS: %s", strings.TrimPrefix(line, "Caption=")))
			} else if strings.HasPrefix(line, "Version=") {
				result.WriteString(fmt.Sprintf("\nVersion: %s", strings.TrimPrefix(line, "Version=")))
			} else if strings.HasPrefix(line, "BuildNumber=") {
				result.WriteString(fmt.Sprintf("\nBuild: %s", strings.TrimPrefix(line, "BuildNumber=")))
			}
		}
	}

	// Get hardware info
	if output, err := exec.Command("wmic", "computersystem", "get", "Model,Manufacturer", "/value").CombinedOutput(); err == nil {
		lines := strings.Split(string(output), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "Manufacturer=") {
				result.WriteString(fmt.Sprintf("\n\nManufacturer: %s", strings.TrimPrefix(line, "Manufacturer=")))
			} else if strings.HasPrefix(line, "Model=") {
				result.WriteString(fmt.Sprintf("\nModel: %s", strings.TrimPrefix(line, "Model=")))
			}
		}
	}

	// Get memory info
	if output, err := exec.Command("wmic", "computersystem", "get", "TotalPhysicalMemory", "/value").CombinedOutput(); err == nil {
		lines := strings.Split(string(output), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "TotalPhysicalMemory=") {
				memStr := strings.TrimPrefix(line, "TotalPhysicalMemory=")
				if memBytes := parseMemorySize(memStr); memBytes > 0 {
					result.WriteString(fmt.Sprintf("\n\nTotal Memory: %.2f GB", float64(memBytes)/(1024*1024*1024)))
				}
			}
		}
	}

	// Get CPU info
	if output, err := exec.Command("wmic", "cpu", "get", "Name,NumberOfCores,MaxClockSpeed", "/value").CombinedOutput(); err == nil {
		lines := strings.Split(string(output), "\n")
		cpuInfo := make(map[string]string)
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.Contains(line, "=") {
				parts := strings.SplitN(line, "=", 2)
				if len(parts) == 2 && parts[1] != "" {
					cpuInfo[parts[0]] = parts[1]
				}
			}
		}
		if name, ok := cpuInfo["Name"]; ok {
			result.WriteString(fmt.Sprintf("\n\nCPU: %s", name))
		}
		if cores, ok := cpuInfo["NumberOfCores"]; ok {
			result.WriteString(fmt.Sprintf("\nCores: %s", cores))
		}
		if speed, ok := cpuInfo["MaxClockSpeed"]; ok {
			result.WriteString(fmt.Sprintf("\nMax Speed: %s MHz", speed))
		}
	}

	// Get disk info
	if output, err := exec.Command("wmic", "logicaldisk", "get", "size,freespace,caption", "/value").CombinedOutput(); err == nil {
		result.WriteString("\n\nDisk Usage:")
		lines := strings.Split(string(output), "\n")
		var caption, size, free string
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "Caption=") {
				caption = strings.TrimPrefix(line, "Caption=")
			} else if strings.HasPrefix(line, "Size=") {
				size = strings.TrimPrefix(line, "Size=")
			} else if strings.HasPrefix(line, "FreeSpace=") {
				free = strings.TrimPrefix(line, "FreeSpace=")
			}

			// When we have all three values, print and reset
			if caption != "" && size != "" && free != "" {
				sizeBytes := parseMemorySize(size)
				freeBytes := parseMemorySize(free)
				if sizeBytes > 0 {
					usedBytes := sizeBytes - freeBytes
					result.WriteString(fmt.Sprintf("\n%s %.1f GB total, %.1f GB used, %.1f GB free",
						caption,
						float64(sizeBytes)/(1024*1024*1024),
						float64(usedBytes)/(1024*1024*1024),
						float64(freeBytes)/(1024*1024*1024)))
				}
				caption, size, free = "", "", ""
			}
		}
	}

	// Get logged in users
	if output, err := exec.Command("query", "user").CombinedOutput(); err == nil {
		result.WriteString("\n\nLogged in users:\n")
		result.WriteString(string(output))
	}

	return result.String(), nil
}

// parseMemorySize converts string memory size to int64 bytes
func parseMemorySize(s string) int64 {
	var size int64
	fmt.Sscanf(s, "%d", &size)
	return size
}
