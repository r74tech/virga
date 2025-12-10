package ps

import (
	"os"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestPsModule_Name(t *testing.T) {
	module := &PsModule{}
	if name := module.Name(); name != "ps" {
		t.Errorf("Expected name 'ps', got '%s'", name)
	}
}

func TestPsModule_Execute(t *testing.T) {
	module := &PsModule{}

	tests := []struct {
		name        string
		args        []string
		expectError bool
		expectCode  int
		verifyFunc  func(string) error
	}{
		{
			name:        "List all processes",
			args:        []string{},
			expectError: false,
			expectCode:  0,
			verifyFunc: func(output string) error {
				// Should contain at least the current process
				currentPID := strconv.Itoa(os.Getpid())
				if !strings.Contains(output, currentPID) {
					t.Errorf("Output should contain current process PID %s", currentPID)
				}

				// Should have header or process information
				if len(output) < 50 {
					t.Error("Output seems too short for process list")
				}

				// Platform-specific checks
				switch runtime.GOOS {
				case "windows":
					// Windows output might contain "System Idle Process" or current executable
					if !strings.Contains(output, ".exe") && !strings.Contains(output, "System") {
						t.Error("Windows output should contain .exe processes or System processes")
					}
				case "darwin", "linux":
					// Unix output should contain PID, USER, or process names
					if !strings.Contains(output, "PID") && !strings.Contains(output, currentPID) {
						t.Error("Unix output should contain PID information")
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
				// Should still return process list
				if len(output) < 50 {
					t.Error("Output should contain process list even with extra args")
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

func TestGetProcessList_Unix(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping Unix-specific test on Windows")
	}

	output, err := getProcessList()
	if err != nil {
		t.Fatalf("Failed to get process list: %v", err)
	}

	// Unix ps output checks
	if !strings.Contains(output, "PID") {
		t.Error("Unix ps output should contain PID header")
	}

	// Should contain common processes
	commonProcesses := []string{"init", "systemd", "kernel", "launchd"}
	foundAny := false
	for _, proc := range commonProcesses {
		if strings.Contains(strings.ToLower(output), proc) {
			foundAny = true
			break
		}
	}
	if !foundAny && runtime.GOOS != "darwin" { // macOS might not show these
		t.Log("Warning: No common system processes found")
	}

	// Check format (should have multiple lines)
	lines := strings.Split(output, "\n")
	if len(lines) < 5 {
		t.Error("Process list should have multiple lines")
	}
}

func TestGetProcessList_Windows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Skipping Windows-specific test on non-Windows")
	}

	output, err := getProcessList()
	if err != nil {
		t.Fatalf("Failed to get process list: %v", err)
	}

	// Windows output checks
	// Should contain common Windows processes
	commonProcesses := []string{"System", "svchost.exe", "csrss.exe", "services.exe"}
	foundCount := 0
	for _, proc := range commonProcesses {
		if strings.Contains(output, proc) {
			foundCount++
		}
	}
	if foundCount == 0 {
		t.Error("Windows output should contain at least one common system process")
	}

	// Check that it's not an error message
	if strings.Contains(strings.ToLower(output), "error") {
		t.Error("Output contains error message")
	}
}

func TestPsModule_OutputFormat(t *testing.T) {
	module := &PsModule{}
	output, exitCode, err := module.Execute([]string{})
	if err != nil {
		t.Fatalf("Failed to execute ps module: %v", err)
	}
	if exitCode != 0 {
		t.Fatalf("Expected exit code 0, got %d", exitCode)
	}

	// Check output structure
	lines := strings.Split(output, "\n")
	nonEmptyLines := 0
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			nonEmptyLines++
		}
	}

	if nonEmptyLines < 2 {
		t.Error("Process list should have at least 2 non-empty lines")
	}

	// Platform-specific format checks
	switch runtime.GOOS {
	case "windows":
		// Windows might use wmic or tasklist format
		// Both should have some structure
		if !strings.Contains(output, " ") {
			t.Error("Windows output should have space-separated values")
		}
	case "darwin", "linux":
		// Unix output should be formatted with columns
		if len(lines) > 2 {
			// Check if output looks like it has columns
			// Skip header and separator lines
			firstDataLine := ""
			for _, line := range lines[2:] {
				trimmed := strings.TrimSpace(line)
				if trimmed != "" && !strings.HasPrefix(trimmed, "---") {
					firstDataLine = line
					break
				}
			}
			if firstDataLine != "" && !strings.Contains(firstDataLine, " ") {
				t.Logf("First data line: %q", firstDataLine)
				t.Error("Unix output should have space-separated columns")
			}
		}
	}
}

func TestPsModule_CurrentProcess(t *testing.T) {
	module := &PsModule{}
	output, _, err := module.Execute([]string{})
	if err != nil {
		t.Fatalf("Failed to execute ps module: %v", err)
	}

	// Get current process info
	currentPID := os.Getpid()
	currentPIDStr := strconv.Itoa(currentPID)

	// Check if current process is in the list
	if !strings.Contains(output, currentPIDStr) {
		t.Errorf("Current process PID %s not found in process list", currentPIDStr)
	}

	// On Unix, we can check for the test binary name
	if runtime.GOOS != "windows" {
		lines := strings.Split(output, "\n")
		foundCurrentProcess := false
		for _, line := range lines {
			if strings.Contains(line, currentPIDStr) {
				foundCurrentProcess = true
				// Should contain test binary or go test
				if !strings.Contains(line, "test") && !strings.Contains(line, "go") {
					t.Log("Warning: Current process line doesn't contain expected test binary name")
				}
				break
			}
		}
		if !foundCurrentProcess {
			t.Error("Could not find current process line in output")
		}
	}
}

func TestPsModule_ErrorHandling(t *testing.T) {
	// This test is platform-specific and tests the underlying implementation
	// On Unix, we test the command execution
	// On Windows, we test different approaches

	if runtime.GOOS == "windows" {
		// Test Windows-specific error paths
		// Note: These are internal implementation details
		// In real scenario, these should rarely fail
		output, err := getProcessList()
		if err != nil {
			// If it fails, it should still return some output
			if output == "" {
				t.Error("Windows implementation should return output even on partial failure")
			}
		}
	} else {
		// Test Unix-specific implementation
		output, err := getProcessList()
		if err != nil {
			t.Logf("Unix ps command failed (might be expected in some environments): %v", err)
			// Even on error, should have some output
			if output == "" && err != nil {
				t.Error("Unix implementation should capture output even on error")
			}
		}
	}
}

func BenchmarkPsModule_Execute(b *testing.B) {
	module := &PsModule{}

	// Limit benchmark runs to avoid overwhelming the system
	if b.N > 100 {
		b.N = 100
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Add small delay to avoid overwhelming the system
		if i > 0 && i%10 == 0 {
			time.Sleep(10 * time.Millisecond)
		}
		_, _, _ = module.Execute([]string{})
	}
}

func BenchmarkGetProcessList(b *testing.B) {
	// Limit benchmark runs to avoid overwhelming the system
	if b.N > 100 {
		b.N = 100
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Add small delay to avoid overwhelming the system
		if i > 0 && i%10 == 0 {
			time.Sleep(10 * time.Millisecond)
		}
		_, _ = getProcessList()
	}
}

// TestPsModule_Stability runs the ps command multiple times to ensure stability
func TestPsModule_Stability(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stability test in short mode")
	}

	// Skip in CI/CD environments to avoid resource exhaustion
	if os.Getenv("CI") != "" || os.Getenv("GITHUB_ACTIONS") != "" {
		t.Skip("Skipping stability test in CI environment")
	}

	module := &PsModule{}
	const iterations = 5 // Reduced iterations

	// Add timeout for the entire test
	testTimeout := time.After(30 * time.Second)
	outputs := make([]string, iterations)

	for i := 0; i < iterations; i++ {
		select {
		case <-testTimeout:
			t.Fatalf("Test timed out after %d iterations", i)
		default:
			// Add a small delay between iterations to avoid overwhelming the system
			if i > 0 {
				time.Sleep(100 * time.Millisecond)
			}

			output, exitCode, err := module.Execute([]string{})
			if err != nil {
				t.Errorf("Iteration %d failed: %v", i, err)
			}
			if exitCode != 0 {
				t.Errorf("Iteration %d: expected exit code 0, got %d", i, exitCode)
			}
			outputs[i] = output
		}
	}

	// Check that we get consistent output structure
	firstLineCount := len(strings.Split(outputs[0], "\n"))
	for i := 1; i < iterations; i++ {
		lineCount := len(strings.Split(outputs[i], "\n"))
		// Allow some variation in process count (±50%)
		minExpected := firstLineCount / 2
		maxExpected := firstLineCount * 2
		if lineCount < minExpected || lineCount > maxExpected {
			t.Errorf("Iteration %d: line count %d is too different from first iteration %d",
				i, lineCount, firstLineCount)
		}
	}
}
