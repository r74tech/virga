package portfwd

import (
	"fmt"
	"net"
	"strings"
	"testing"
	"time"
)

func TestPortfwdModule_Name(t *testing.T) {
	module := NewPortfwdModule()
	if name := module.Name(); name != "portfwd" {
		t.Errorf("Expected name 'portfwd', got '%s'", name)
	}
}

func TestNewPortfwdModule(t *testing.T) {
	module := NewPortfwdModule()

	if module == nil {
		t.Fatal("NewPortfwdModule returned nil")
	}

	if module.forwards == nil {
		t.Error("forwards map not initialized")
	}
}

func TestPortfwdModule_Execute_List(t *testing.T) {
	module := NewPortfwdModule()

	tests := []struct {
		name        string
		args        []string
		expectError bool
		expectCode  int
		verifyFunc  func(string) error
	}{
		{
			name:        "No arguments defaults to list",
			args:        []string{},
			expectError: false,
			expectCode:  0,
			verifyFunc: func(output string) error {
				if !strings.Contains(output, "No active port forwards") &&
					!strings.Contains(output, "Active port forwards:") {
					t.Error("Output should indicate port forward status")
				}
				return nil
			},
		},
		{
			name:        "Explicit list command",
			args:        []string{"list"},
			expectError: false,
			expectCode:  0,
			verifyFunc: func(output string) error {
				if !strings.Contains(output, "No active port forwards") &&
					!strings.Contains(output, "Active port forwards:") {
					t.Error("Output should indicate port forward status")
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

func TestPortfwdModule_Execute_Add(t *testing.T) {
	module := NewPortfwdModule()

	tests := []struct {
		name         string
		args         []string
		expectError  bool
		expectCode   int
		errorMessage string
		verifyFunc   func() error
		cleanupFunc  func()
	}{
		{
			name:         "Add with insufficient arguments",
			args:         []string{"add"},
			expectError:  true,
			expectCode:   1,
			errorMessage: "requires local_port remote_host remote_port",
		},
		{
			name:         "Add with invalid local port",
			args:         []string{"add", "not-a-port", "example.com", "80"},
			expectError:  true,
			expectCode:   1,
			errorMessage: "invalid local port",
		},
		{
			name:         "Add with invalid remote port",
			args:         []string{"add", "8080", "example.com", "not-a-port"},
			expectError:  true,
			expectCode:   1,
			errorMessage: "invalid remote port",
		},
		{
			name:         "Add with out of range local port",
			args:         []string{"add", "99999", "example.com", "80"},
			expectError:  true,
			expectCode:   1,
			errorMessage: "invalid port",
		},
		{
			name:         "Add with out of range remote port",
			args:         []string{"add", "8080", "example.com", "99999"},
			expectError:  true,
			expectCode:   1,
			errorMessage: "invalid port",
		},
		{
			name:        "Add valid port forward",
			args:        []string{"add", "0", "127.0.0.1", "12345"}, // Use port 0 for auto-assign
			expectError: false,
			expectCode:  0,
			verifyFunc: func() error {
				// Check that a forward was added
				module.mu.Lock()
				defer module.mu.Unlock()

				if len(module.forwards) != 1 {
					return fmt.Errorf("Expected 1 forward, got %d", len(module.forwards))
				}

				// Check forward is active
				for _, fwd := range module.forwards {
					if !fwd.active {
						return fmt.Errorf("Forward should be active")
					}
					if fwd.listener == nil {
						return fmt.Errorf("Forward should have a listener")
					}
				}
				return nil
			},
			cleanupFunc: func() {
				// Clean up the forward with timeout
				done := make(chan struct{})
				go func() {
					module.mu.Lock()
					var ports []string
					for key := range module.forwards {
						parts := strings.Split(key, ":")
						if len(parts) > 1 {
							ports = append(ports, parts[1])
						}
					}
					module.mu.Unlock()

					// Remove each forward
					for _, port := range ports {
						module.Execute([]string{"remove", port})
					}
					close(done)
				}()

				select {
				case <-done:
				case <-time.After(2 * time.Second):
					t.Log("Warning: Cleanup timed out")
				}
			},
		},
		{
			name:        "Add duplicate port forward",
			args:        []string{"add", "0", "127.0.0.1", "12345"},
			expectError: false,
			expectCode:  0,
			verifyFunc: func() error {
				// Add first forward
				_, _, err := module.Execute([]string{"add", "0", "127.0.0.1", "12345"})
				if err != nil {
					return err
				}

				// Try to add duplicate (should succeed with different local port)
				output, exitCode, err := module.Execute([]string{"add", "8081", "127.0.0.1", "12345"})
				if err != nil || exitCode != 0 {
					return fmt.Errorf("Should allow multiple forwards to same target")
				}

				if !strings.Contains(output, "Port forward added") {
					return fmt.Errorf("Expected success message")
				}

				return nil
			},
			cleanupFunc: func() {
				// Clean up all forwards
				module.mu.Lock()
				for key := range module.forwards {
					module.mu.Unlock()
					module.Execute([]string{"remove", key})
					module.mu.Lock()
				}
				module.mu.Unlock()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.cleanupFunc != nil {
				defer tt.cleanupFunc()
			}

			output, exitCode, err := module.Execute(tt.args)

			// Check error expectation
			if tt.expectError {
				if err == nil && exitCode == 0 {
					t.Error("Expected error but got none")
				}
				if tt.errorMessage != "" && !strings.Contains(strings.ToLower(output), strings.ToLower(tt.errorMessage)) &&
					(err == nil || !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(tt.errorMessage))) {
					t.Errorf("Expected error message containing '%s', got output: '%s', error: %v", tt.errorMessage, output, err)
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
				if err := tt.verifyFunc(); err != nil {
					t.Errorf("Verification failed: %v", err)
				}
			}
		})
	}
}

func TestPortfwdModule_Execute_Remove(t *testing.T) {
	module := NewPortfwdModule()

	tests := []struct {
		name         string
		setupFunc    func() string // returns local port
		args         []string
		expectError  bool
		expectCode   int
		errorMessage string
	}{
		{
			name:         "Remove with no arguments",
			args:         []string{"remove"},
			expectError:  true,
			expectCode:   1,
			errorMessage: "requires local_port",
		},
		{
			name:         "Remove non-existent forward",
			args:         []string{"remove", "9999"},
			expectError:  true,
			expectCode:   1,
			errorMessage: "not found",
		},
		{
			name: "Remove existing forward",
			setupFunc: func() string {
				// Add a forward first
				output, _, _ := module.Execute([]string{"add", "0", "127.0.0.1", "12345"})
				// Extract the actual port from output
				if strings.Contains(output, "localhost:") {
					parts := strings.Split(output, "localhost:")
					if len(parts) > 1 {
						portParts := strings.Fields(parts[1])
						if len(portParts) > 0 {
							return portParts[0]
						}
					}
				}
				return "0"
			},
			expectError: false,
			expectCode:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var actualPort string
			if tt.setupFunc != nil {
				actualPort = tt.setupFunc()
				tt.args = []string{"remove", actualPort}
			}

			output, exitCode, err := module.Execute(tt.args)

			// Check error expectation
			if tt.expectError {
				if err == nil && exitCode == 0 {
					t.Error("Expected error but got none")
				}
				if tt.errorMessage != "" && !strings.Contains(strings.ToLower(output), strings.ToLower(tt.errorMessage)) {
					t.Errorf("Expected error message containing '%s', got '%s'", tt.errorMessage, output)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if !strings.Contains(output, "removed") {
					t.Error("Expected success message about removal")
				}
			}

			// Check exit code
			if exitCode != tt.expectCode {
				t.Errorf("Expected exit code %d, got %d", tt.expectCode, exitCode)
			}
		})
	}
}

func TestPortfwdModule_Execute_Unknown(t *testing.T) {
	module := NewPortfwdModule()

	output, exitCode, err := module.Execute([]string{"unknown-command"})

	if err == nil {
		t.Error("Expected error for unknown command")
	}
	if exitCode != 1 {
		t.Errorf("Expected exit code 1, got %d", exitCode)
	}
	if !strings.Contains(output, "unknown command") {
		t.Errorf("Expected 'unknown command' in output, got: %s", output)
	}
}

func TestPortfwdModule_PortForwarding(t *testing.T) {
	module := NewPortfwdModule()

	// Create a test server to forward to
	testServer, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to create test server: %v", err)
	}
	defer testServer.Close()

	testServerAddr := testServer.Addr().(*net.TCPAddr)
	testServerPort := testServerAddr.Port

	// Handle connections to test server
	go func() {
		for {
			conn, err := testServer.Accept()
			if err != nil {
				return
			}
			// Echo server - write back what we receive
			go func(c net.Conn) {
				defer c.Close()
				buf := make([]byte, 1024)
				n, _ := c.Read(buf)
				c.Write(buf[:n])
			}(conn)
		}
	}()

	// Add port forward
	output, exitCode, err := module.Execute([]string{"add", "0", "127.0.0.1", fmt.Sprintf("%d", testServerPort)})
	if err != nil {
		t.Fatalf("Failed to add port forward: %v", err)
	}
	if exitCode != 0 {
		t.Fatalf("Expected exit code 0, got %d", exitCode)
	}

	// Extract the local port from output
	var localPort string
	if strings.Contains(output, "localhost:") {
		parts := strings.Split(output, "localhost:")
		if len(parts) > 1 {
			portParts := strings.Fields(parts[1])
			if len(portParts) > 0 {
				localPort = portParts[0]
			}
		}
	}
	if localPort == "" {
		t.Fatal("Could not extract local port from output")
	}

	// Clean up when done
	defer module.Execute([]string{"remove", localPort})

	// Give time for port forward to start
	time.Sleep(100 * time.Millisecond)

	// Test the port forward
	conn, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%s", localPort))
	if err != nil {
		t.Fatalf("Failed to connect to forwarded port: %v", err)
	}
	defer conn.Close()

	// Send test data
	testData := []byte("Hello, portforward!")
	_, err = conn.Write(testData)
	if err != nil {
		t.Fatalf("Failed to write to forwarded connection: %v", err)
	}

	// Read response
	buf := make([]byte, len(testData))
	_, err = conn.Read(buf)
	if err != nil {
		t.Fatalf("Failed to read from forwarded connection: %v", err)
	}

	// Verify data
	if string(buf) != string(testData) {
		t.Errorf("Expected '%s', got '%s'", testData, buf)
	}
}

func TestPortfwdModule_ConcurrentForwards(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrent forwards test in short mode")
	}

	module := NewPortfwdModule()

	// Create multiple test servers
	const numForwards = 3
	var servers []net.Listener
	var localPorts []string

	for i := 0; i < numForwards; i++ {
		server, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatalf("Failed to create test server %d: %v", i, err)
		}
		servers = append(servers, server)

		// Echo server
		go func(s net.Listener, id int) {
			for {
				conn, err := s.Accept()
				if err != nil {
					return
				}
				go func(c net.Conn) {
					defer c.Close()
					c.Write([]byte(fmt.Sprintf("Server %d", id)))
				}(conn)
			}
		}(server, i)
	}

	// Clean up servers
	defer func() {
		for _, s := range servers {
			s.Close()
		}
	}()

	// Add port forwards
	for i, server := range servers {
		addr := server.Addr().(*net.TCPAddr)
		output, exitCode, err := module.Execute([]string{"add", "0", "127.0.0.1", fmt.Sprintf("%d", addr.Port)})
		if err != nil {
			t.Errorf("Failed to add forward %d: %v", i, err)
			continue
		}
		if exitCode != 0 {
			t.Errorf("Forward %d: expected exit code 0, got %d", i, exitCode)
			continue
		}

		// Extract local port
		if strings.Contains(output, "localhost:") {
			parts := strings.Split(output, "localhost:")
			if len(parts) > 1 {
				portParts := strings.Fields(parts[1])
				if len(portParts) > 0 {
					localPorts = append(localPorts, portParts[0])
				}
			}
		}
	}

	// Clean up forwards
	defer func() {
		for _, port := range localPorts {
			module.Execute([]string{"remove", port})
		}
	}()

	// Verify all forwards are active
	output, _, _ := module.Execute([]string{"list"})
	if strings.Count(output, "active") != numForwards {
		t.Errorf("Expected %d active forwards, got: %s", numForwards, output)
	}

	// Test each forward
	for i, port := range localPorts {
		conn, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%s", port))
		if err != nil {
			t.Errorf("Failed to connect to forward %d: %v", i, err)
			continue
		}

		buf := make([]byte, 100)
		n, _ := conn.Read(buf)
		conn.Close()

		expected := fmt.Sprintf("Server %d", i)
		if string(buf[:n]) != expected {
			t.Errorf("Forward %d: expected '%s', got '%s'", i, expected, string(buf[:n]))
		}
	}
}

func BenchmarkPortfwdModule_Execute(b *testing.B) {
	module := NewPortfwdModule()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = module.Execute([]string{"list"})
	}
}

func BenchmarkPortfwdModule_AddRemove(b *testing.B) {
	module := NewPortfwdModule()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Add forward
		output, _, _ := module.Execute([]string{"add", "0", "127.0.0.1", "12345"})

		// Extract port and remove
		if strings.Contains(output, "localhost:") {
			parts := strings.Split(output, "localhost:")
			if len(parts) > 1 {
				portParts := strings.Fields(parts[1])
				if len(portParts) > 0 {
					module.Execute([]string{"remove", portParts[0]})
				}
			}
		}
	}
}
