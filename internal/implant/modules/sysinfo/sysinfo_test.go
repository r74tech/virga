package sysinfo

import (
	"os"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

func TestSysinfoModule_Name(t *testing.T) {
	module := &SysinfoModule{}
	if name := module.Name(); name != "sysinfo" {
		t.Errorf("Expected name 'sysinfo', got '%s'", name)
	}
}

func TestSysinfoModule_Execute(t *testing.T) {
	module := &SysinfoModule{}

	tests := []struct {
		name        string
		args        []string
		expectError bool
		expectCode  int
		verifyFunc  func(string) error
	}{
		{
			name:        "Get system information",
			args:        []string{},
			expectError: false,
			expectCode:  0,
			verifyFunc: func(output string) error {
				// Check for required fields
				requiredFields := []string{
					"Hostname:",
					"OS:",
					"Architecture:",
					"CPU Cores:",
					"Total Memory:",
					"Available Memory:",
					"Boot Time:",
					"Process Count:",
				}

				for _, field := range requiredFields {
					if !strings.Contains(output, field) {
						t.Errorf("Output missing required field: %s", field)
					}
				}

				// Check hostname matches
				hostname, _ := os.Hostname()
				if hostname != "" && !strings.Contains(output, hostname) {
					t.Error("Output should contain actual hostname")
				}

				// Check OS matches runtime
				if !strings.Contains(strings.ToLower(output), runtime.GOOS) {
					t.Errorf("Output should contain OS: %s", runtime.GOOS)
				}

				// Check architecture matches
				if !strings.Contains(output, runtime.GOARCH) {
					t.Errorf("Output should contain architecture: %s", runtime.GOARCH)
				}

				// Platform-specific checks
				switch runtime.GOOS {
				case "linux":
					// Linux-specific fields
					if !strings.Contains(output, "Kernel Version:") {
						t.Error("Linux output should contain kernel version")
					}
					if !strings.Contains(output, "Distribution:") {
						t.Log("Warning: Linux output missing distribution info")
					}
				case "darwin":
					// macOS-specific fields
					if !strings.Contains(output, "macOS Version:") {
						t.Error("macOS output should contain OS version")
					}
				case "windows":
					// Windows-specific fields
					if !strings.Contains(output, "Windows Version:") {
						t.Error("Windows output should contain OS version")
					}
				}

				return nil
			},
		},
		{
			name:        "Extra arguments ignored",
			args:        []string{"ignored", "arguments"},
			expectError: false,
			expectCode:  0,
			verifyFunc: func(output string) error {
				// Should still return system info
				if !strings.Contains(output, "Hostname:") {
					t.Error("Output should contain system info even with extra args")
				}
				return nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, exitCode, err := module.Execute(tt.args)

			// Check error expectation
			if tt.expectError {
				if err == nil && exitCode == 0 {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}

			// Check exit code
			if exitCode != tt.expectCode {
				t.Errorf("Expected exit code %d, got %d", tt.expectCode, exitCode)
			}

			// Run verification function if provided
			if tt.verifyFunc != nil && !tt.expectError {
				if err := tt.verifyFunc(output); err != nil {
					t.Errorf("Verification failed: %v", err)
				}
			}
		})
	}
}

func TestGetDetailedSystemInfo_Unix(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping Unix-specific test on Windows")
	}

	output, err := getDetailedSystemInfo()
	if err != nil {
		t.Fatalf("Failed to get system info: %v", err)
	}

	// Check basic system info formatting
	lines := strings.Split(output, "\n")
	if len(lines) < 5 {
		t.Error("System info should have multiple lines")
	}

	// Check for specific Unix information
	hasMemInfo := false
	hasCPUInfo := false
	hasKernelInfo := false

	for _, line := range lines {
		if strings.Contains(line, "Memory:") || strings.Contains(line, "Total Memory:") {
			hasMemInfo = true
		}
		if strings.Contains(line, "CPU") || strings.Contains(line, "Cores:") {
			hasCPUInfo = true
		}
		if strings.Contains(line, "Kernel") || strings.Contains(line, "Darwin") || strings.Contains(line, "Linux") {
			hasKernelInfo = true
		}
	}

	if !hasMemInfo {
		t.Error("Unix system info should contain memory information")
	}
	if !hasCPUInfo {
		t.Error("Unix system info should contain CPU information")
	}
	if !hasKernelInfo {
		t.Error("Unix system info should contain kernel information")
	}
}

func TestGetDetailedSystemInfo_Windows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Skipping Windows-specific test on non-Windows")
	}

	output, err := getDetailedSystemInfo()
	if err != nil {
		t.Fatalf("Failed to get system info: %v", err)
	}

	// Check Windows-specific information
	windowsFields := []string{
		"Windows Version:",
		"Computer Name:",
		"Total Physical Memory:",
		"Available Physical Memory:",
		"Number Of Processors:",
	}

	for _, field := range windowsFields {
		if !strings.Contains(output, field) {
			t.Errorf("Windows system info missing field: %s", field)
		}
	}
}

func TestSysinfoModule_NumericValues(t *testing.T) {
	module := &SysinfoModule{}
	output, _, err := module.Execute([]string{})
	if err != nil {
		t.Fatalf("Failed to execute sysinfo: %v", err)
	}

	// Test that numeric values are valid
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Check CPU cores
		if strings.HasPrefix(line, "CPU Cores:") {
			parts := strings.Split(line, ":")
			if len(parts) >= 2 {
				coresStr := strings.TrimSpace(parts[1])
				cores, err := strconv.Atoi(coresStr)
				if err != nil {
					t.Errorf("Invalid CPU cores value: %s", coresStr)
				}
				if cores < 1 || cores > 1024 {
					t.Errorf("Unrealistic CPU core count: %d", cores)
				}
			}
		}

		// Check process count
		if strings.HasPrefix(line, "Process Count:") {
			parts := strings.Split(line, ":")
			if len(parts) >= 2 {
				countStr := strings.TrimSpace(parts[1])
				count, err := strconv.Atoi(countStr)
				if err != nil {
					t.Errorf("Invalid process count value: %s", countStr)
				}
				if count < 1 {
					t.Errorf("Process count should be at least 1: %d", count)
				}
			}
		}

		// Check memory values
		if strings.Contains(line, "Memory:") && strings.Contains(line, "MB") {
			// Extract numeric value from "X MB" format
			parts := strings.Fields(line)
			for i, part := range parts {
				if part == "MB" && i > 0 {
					memStr := parts[i-1]
					mem, err := strconv.ParseFloat(memStr, 64)
					if err == nil {
						if mem < 0 {
							t.Errorf("Memory value should not be negative: %f", mem)
						}
						if mem > 1000000 { // 1 million MB = 1 PB
							t.Errorf("Unrealistic memory value: %f MB", mem)
						}
					}
				}
			}
		}
	}
}

func TestSysinfoModule_CurrentUser(t *testing.T) {
	module := &SysinfoModule{}
	output, _, err := module.Execute([]string{})
	if err != nil {
		t.Fatalf("Failed to execute sysinfo: %v", err)
	}

	// Check that current user information is present
	currentUser := os.Getenv("USER")
	if currentUser == "" {
		currentUser = os.Getenv("USERNAME") // Windows
	}

	if currentUser != "" {
		if !strings.Contains(output, "Current User:") {
			t.Error("Output should contain current user information")
		}
		// The actual username might be in a different format, so we don't strictly check for exact match
	}
}

func TestSysinfoModule_NetworkInterfaces(t *testing.T) {
	module := &SysinfoModule{}
	output, _, err := module.Execute([]string{})
	if err != nil {
		t.Fatalf("Failed to execute sysinfo: %v", err)
	}

	// Check for network interface information
	if !strings.Contains(output, "Network Interfaces:") {
		t.Error("Output should contain network interface information")
	}

	// Should have at least loopback interface
	if !strings.Contains(output, "127.0.0.1") && !strings.Contains(output, "::1") {
		t.Log("Warning: No loopback interface found in network info")
	}
}

func TestSysinfoModule_Consistency(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping consistency test in short mode")
	}

	module := &SysinfoModule{}

	// Run multiple times to ensure consistent output format
	const iterations = 5
	outputs := make([]string, iterations)

	for i := 0; i < iterations; i++ {
		output, exitCode, err := module.Execute([]string{})
		if err != nil {
			t.Errorf("Iteration %d failed: %v", i, err)
		}
		if exitCode != 0 {
			t.Errorf("Iteration %d: expected exit code 0, got %d", i, exitCode)
		}
		outputs[i] = output
	}

	// Check that all outputs have similar structure
	firstLines := strings.Split(outputs[0], "\n")
	firstFieldCount := 0
	for _, line := range firstLines {
		if strings.Contains(line, ":") {
			firstFieldCount++
		}
	}

	for i := 1; i < iterations; i++ {
		lines := strings.Split(outputs[i], "\n")
		fieldCount := 0
		for _, line := range lines {
			if strings.Contains(line, ":") {
				fieldCount++
			}
		}

		// Allow small variations (±2 fields) due to dynamic information
		if fieldCount < firstFieldCount-2 || fieldCount > firstFieldCount+2 {
			t.Errorf("Iteration %d: field count %d differs significantly from first iteration %d",
				i, fieldCount, firstFieldCount)
		}
	}
}

func BenchmarkSysinfoModule_Execute(b *testing.B) {
	module := &SysinfoModule{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = module.Execute([]string{})
	}
}

func BenchmarkGetDetailedSystemInfo(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = getDetailedSystemInfo()
	}
}

// TestSysinfoModule_LargeOutput ensures the module handles large output gracefully
func TestSysinfoModule_LargeOutput(t *testing.T) {
	module := &SysinfoModule{}
	output, exitCode, err := module.Execute([]string{})

	if err != nil {
		t.Fatalf("Failed to execute sysinfo: %v", err)
	}
	if exitCode != 0 {
		t.Fatalf("Expected exit code 0, got %d", exitCode)
	}

	// Check output size is reasonable (not truncated but not excessive)
	outputSize := len(output)
	if outputSize < 100 {
		t.Error("System info output seems too small")
	}
	if outputSize > 1024*1024 { // 1MB
		t.Error("System info output seems excessively large")
	}

	// Ensure output is properly formatted (not corrupted)
	if !strings.HasSuffix(strings.TrimSpace(output), ":") &&
		!strings.HasSuffix(strings.TrimSpace(output), ")") &&
		!strings.HasSuffix(strings.TrimSpace(output), "processes") {
		// Output should end with a complete line
		lastLine := ""
		lines := strings.Split(output, "\n")
		for i := len(lines) - 1; i >= 0; i-- {
			if strings.TrimSpace(lines[i]) != "" {
				lastLine = lines[i]
				break
			}
		}
		if lastLine != "" && len(lastLine) > 50 {
			t.Logf("Warning: Last line might be truncated: %s", lastLine)
		}
	}
}
