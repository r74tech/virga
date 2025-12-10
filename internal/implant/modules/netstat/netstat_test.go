package netstat

import (
	"fmt"
	"net"
	"os"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestNetstatModule_Name(t *testing.T) {
	module := &NetstatModule{}
	if name := module.Name(); name != "netstat" {
		t.Errorf("Expected name 'netstat', got '%s'", name)
	}
}

func TestNetstatModule_Execute(t *testing.T) {
	module := &NetstatModule{}

	tests := []struct {
		name        string
		args        []string
		expectError bool
		expectCode  int
		setupFunc   func() func() // returns cleanup function
		verifyFunc  func(string) error
	}{
		{
			name:        "List network connections",
			args:        []string{},
			expectError: false,
			expectCode:  0,
			verifyFunc: func(output string) error {
				// Should contain some output
				if len(output) < 10 {
					t.Error("Output seems too short for network connections list")
				}

				// Platform-specific checks
				switch runtime.GOOS {
				case "windows":
					// Windows netstat output
					if !strings.Contains(output, "TCP") && !strings.Contains(output, "UDP") &&
						!strings.Contains(output, "Local Address") {
						t.Error("Windows output should contain TCP/UDP or connection information")
					}
				case "darwin", "linux":
					// Unix netstat output
					if !strings.Contains(output, "tcp") && !strings.Contains(output, "udp") &&
						!strings.Contains(output, "LISTEN") && !strings.Contains(output, "ESTABLISHED") {
						t.Error("Unix output should contain protocol or state information")
					}
				}

				// Should contain at least localhost connections
				if !strings.Contains(output, "127.0.0.1") && !strings.Contains(output, "::1") &&
					!strings.Contains(output, "localhost") {
					t.Log("Warning: No localhost connections found")
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
				// Should still return network connections
				if len(output) < 10 {
					t.Error("Output should contain network connections even with extra args")
				}
				return nil
			},
		},
		{
			name: "With active connection",
			setupFunc: func() func() {
				// Skip in CI/CD environments
				if os.Getenv("CI") != "" || os.Getenv("GITHUB_ACTIONS") != "" {
					t.Skip("Skipping network test in CI environment")
				}

				// Create a test listener
				listener, err := net.Listen("tcp", "127.0.0.1:0")
				if err != nil {
					t.Fatalf("Failed to create test listener: %v", err)
				}

				// Get the actual port
				addr := listener.Addr().(*net.TCPAddr)
				port := addr.Port

				// Create connections with timeout
				done := make(chan struct{})
				go func() {
					conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), 1*time.Second)
					if err == nil {
						defer conn.Close()
						select {
						case <-done:
						case <-time.After(500 * time.Millisecond):
						}
					}
				}()

				// Accept the connection with timeout
				go func() {
					listener.(*net.TCPListener).SetDeadline(time.Now().Add(1 * time.Second))
					conn, err := listener.Accept()
					if err == nil {
						defer conn.Close()
						select {
						case <-done:
						case <-time.After(500 * time.Millisecond):
						}
					}
				}()

				// Give connections time to establish
				time.Sleep(100 * time.Millisecond)

				return func() {
					close(done)
					listener.Close()
				}
			},
			expectError: false,
			expectCode:  0,
			verifyFunc: func(output string) error {
				// Should show our test connection
				if !strings.Contains(output, "127.0.0.1") {
					t.Error("Output should contain localhost connections")
				}

				// Should show established connections
				if runtime.GOOS != "windows" {
					if !strings.Contains(output, "ESTABLISHED") && !strings.Contains(output, "ESTAB") {
						t.Log("Warning: No established connections found")
					}
				}

				return nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var cleanup func()
			if tt.setupFunc != nil {
				cleanup = tt.setupFunc()
				if cleanup != nil {
					defer cleanup()
				}
			}

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

func TestGetNetworkConnections_Unix(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping Unix-specific test on Windows")
	}

	output, err := getNetworkConnections()
	if err != nil {
		// Some environments might not have netstat installed
		if strings.Contains(err.Error(), "executable file not found") {
			t.Skip("netstat not available in this environment")
		}
		t.Fatalf("Failed to get network connections: %v", err)
	}

	// Check output format
	lines := strings.Split(output, "\n")
	if len(lines) < 2 {
		t.Error("Network connections output should have multiple lines")
	}

	// Look for common patterns in netstat output
	hasProtocol := false
	hasAddress := false
	hasState := false

	for _, line := range lines {
		lineLower := strings.ToLower(line)
		if strings.Contains(lineLower, "tcp") || strings.Contains(lineLower, "udp") {
			hasProtocol = true
		}
		if strings.Contains(line, ":") || strings.Contains(line, ".") {
			hasAddress = true
		}
		if strings.Contains(lineLower, "listen") || strings.Contains(lineLower, "established") ||
			strings.Contains(lineLower, "time_wait") {
			hasState = true
		}
	}

	if !hasProtocol {
		t.Error("Unix netstat output should contain protocol information")
	}
	if !hasAddress {
		t.Error("Unix netstat output should contain address information")
	}
	if !hasState {
		t.Log("Warning: No connection state information found (hasState=false)")
	}
}

func TestGetNetworkConnections_Windows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Skipping Windows-specific test on non-Windows")
	}

	output, err := getNetworkConnections()
	if err != nil {
		t.Fatalf("Failed to get network connections: %v", err)
	}

	// Check Windows netstat output
	if !strings.Contains(output, "TCP") && !strings.Contains(output, "UDP") {
		t.Error("Windows netstat output should contain TCP or UDP")
	}

	// Should have header or connection information
	if !strings.Contains(output, "Local Address") && !strings.Contains(output, "Foreign Address") &&
		!strings.Contains(output, ":") {
		t.Error("Windows netstat output should contain address information")
	}
}

func TestNetstatModule_CommonPorts(t *testing.T) {
	module := &NetstatModule{}
	output, _, err := module.Execute([]string{})
	if err != nil {
		t.Fatalf("Failed to execute netstat: %v", err)
	}

	// Check for common ports that might be in use
	commonPorts := []string{
		"22",    // SSH
		"80",    // HTTP
		"443",   // HTTPS
		"445",   // SMB
		"3306",  // MySQL
		"5432",  // PostgreSQL
		"8080",  // HTTP alternate
		"27017", // MongoDB
	}

	foundPorts := 0
	for _, port := range commonPorts {
		if strings.Contains(output, ":"+port) || strings.Contains(output, "."+port) {
			foundPorts++
			t.Logf("Found common port %s in use", port)
		}
	}

	// We don't require any specific ports, just log what we find
	t.Logf("Found %d common ports in use", foundPorts)
}

func TestNetstatModule_MultipleListeners(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping multiple listeners test in short mode")
	}

	// Skip in CI/CD environments
	if os.Getenv("CI") != "" || os.Getenv("GITHUB_ACTIONS") != "" {
		t.Skip("Skipping multiple network test in CI environment")
	}

	module := &NetstatModule{}

	// Create fewer test listeners to reduce resource usage
	var listeners []net.Listener
	var ports []int

	for i := 0; i < 2; i++ {
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatalf("Failed to create test listener %d: %v", i, err)
		}

		addr := listener.Addr().(*net.TCPAddr)
		port := addr.Port

		listeners = append(listeners, listener)
		ports = append(ports, port)
	}

	// Cleanup with timeout
	defer func() {
		done := make(chan struct{})
		go func() {
			for _, listener := range listeners {
				listener.Close()
			}
			close(done)
		}()

		select {
		case <-done:
		case <-time.After(1 * time.Second):
			t.Log("Warning: Listener cleanup timed out")
		}
	}()

	// Give OS time to update network tables
	time.Sleep(100 * time.Millisecond)

	// Get network connections
	output, exitCode, err := module.Execute([]string{})
	if err != nil {
		t.Errorf("Failed to execute netstat: %v", err)
	}
	if exitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", exitCode)
	}

	// Check if our listeners appear in the output
	foundCount := 0
	for _, port := range ports {
		portStr := strconv.Itoa(port)
		if strings.Contains(output, portStr) {
			foundCount++
		}
	}

	// We should find at least some of our listeners
	if foundCount == 0 {
		t.Error("None of the test listeners appeared in netstat output")
	} else {
		t.Logf("Found %d/%d test listeners in output", foundCount, len(ports))
	}
}

func TestNetstatModule_IPv6(t *testing.T) {
	// Skip in CI/CD environments
	if os.Getenv("CI") != "" || os.Getenv("GITHUB_ACTIONS") != "" {
		t.Skip("Skipping IPv6 test in CI environment")
	}

	// Try to create an IPv6 listener
	listener, err := net.Listen("tcp6", "[::1]:0")
	if err != nil {
		t.Skip("IPv6 not available on this system")
	}
	defer listener.Close()

	module := &NetstatModule{}
	output, _, err := module.Execute([]string{})
	if err != nil {
		t.Fatalf("Failed to execute netstat: %v", err)
	}

	// Check for IPv6 addresses in output
	if strings.Contains(output, "::1") || strings.Contains(output, "::") ||
		strings.Contains(output, "tcp6") || strings.Contains(output, "udp6") {
		t.Log("IPv6 connections found in output")
	} else {
		t.Log("No IPv6 connections found in output (might be filtered by netstat)")
	}
}

func BenchmarkNetstatModule_Execute(b *testing.B) {
	module := &NetstatModule{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = module.Execute([]string{})
	}
}

func BenchmarkGetNetworkConnections(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = getNetworkConnections()
	}
}

// TestNetstatModule_OutputParsing verifies the output is parseable
func TestNetstatModule_OutputParsing(t *testing.T) {
	module := &NetstatModule{}
	output, _, err := module.Execute([]string{})
	if err != nil {
		t.Fatalf("Failed to execute netstat: %v", err)
	}

	lines := strings.Split(output, "\n")
	validLines := 0

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Skip header lines
		if strings.Contains(line, "Active") || strings.Contains(line, "Proto") ||
			strings.Contains(line, "Local Address") || strings.Contains(line, "Foreign Address") {
			continue
		}

		// Check if line has expected format (contains colons or dots for addresses)
		if strings.Contains(line, ":") || strings.Contains(line, ".") {
			validLines++
		}
	}

	if validLines == 0 {
		t.Error("No valid connection lines found in netstat output")
	} else {
		t.Logf("Found %d valid connection lines", validLines)
	}
}
