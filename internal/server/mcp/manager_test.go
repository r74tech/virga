package mcp

import (
	"context"
	"testing"
	"time"

	"github.com/r74tech/virga/internal/server/beacons"
	"github.com/r74tech/virga/internal/server/database"
	"github.com/r74tech/virga/internal/server/mcp/config"
	"github.com/r74tech/virga/internal/server/session"
)

// Mock dependencies
type mockBeaconManager struct {
	beacons.Manager
}

func newMockBeaconManager() *beacons.Manager {
	return beacons.NewManager()
}

func newMockSessionManager() *session.Manager {
	return session.NewManager()
}

func TestNewManager(t *testing.T) {
	tests := []struct {
		name      string
		config    *config.Config
		wantError bool
	}{
		{
			name: "All transports disabled",
			config: &config.Config{
				Name:              "Test MCP Server",
				Version:           "1.0.0",
				SSEEnabled:        false,
				StdioEnabled:      false,
				StreamableEnabled: false,
			},
			wantError: false,
		},
		{
			name: "SSE enabled",
			config: &config.Config{
				Name:         "Test MCP Server",
				Version:      "1.0.0",
				SSEEnabled:   true,
				SSEPort:      ":8444",
				SSEBasePath:  "/mcp",
				SSERemoteURL: "http://localhost:8444",
			},
			wantError: false,
		},
		{
			name: "STDIO enabled",
			config: &config.Config{
				Name:         "Test MCP Server",
				Version:      "1.0.0",
				StdioEnabled: true,
			},
			wantError: false,
		},
		{
			name: "Streamable enabled",
			config: &config.Config{
				Name:              "Test MCP Server",
				Version:           "1.0.0",
				StreamableEnabled: true,
				StreamablePort:    ":50012",
			},
			wantError: false,
		},
		{
			name: "All transports enabled",
			config: &config.Config{
				Name:              "Test MCP Server",
				Version:           "1.0.0",
				SSEEnabled:        true,
				SSEPort:           ":8444",
				SSEBasePath:       "/mcp",
				StdioEnabled:      true,
				StreamableEnabled: true,
				StreamablePort:    ":50012",
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test database
			db, err := database.Initialize(":memory:")
			if err != nil {
				t.Fatalf("Failed to create test database: %v", err)
			}
			defer db.Close()

			beaconMgr := newMockBeaconManager()
			sessionMgr := newMockSessionManager()

			manager, err := NewManager(tt.config, beaconMgr, sessionMgr, db)
			if (err != nil) != tt.wantError {
				t.Errorf("NewManager() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if !tt.wantError && manager == nil {
				t.Error("NewManager() returned nil manager without error")
				return
			}

			if manager != nil {
				// Verify servers were created correctly
				if tt.config.SSEEnabled {
					if _, ok := manager.GetServer(config.TransportSSE); !ok {
						t.Error("SSE server not created when enabled")
					}
				}
				if tt.config.StdioEnabled {
					if _, ok := manager.GetServer(config.TransportStdio); !ok {
						t.Error("STDIO server not created when enabled")
					}
				}
				if tt.config.StreamableEnabled {
					if _, ok := manager.GetServer(config.TransportStreamable); !ok {
						t.Error("Streamable server not created when enabled")
					}
				}
			}
		})
	}
}

func TestManagerStartAll(t *testing.T) {
	done := make(chan bool)
	go func() {
		defer func() { done <- true }()
		// Create test configuration
		cfg := &config.Config{
			Name:              "Test MCP Server",
			Version:           "1.0.0",
			SSEEnabled:        true,
			SSEPort:           ":0",  // Use random port
			StdioEnabled:      false, // STDIO blocks, so disable for this test
			StreamableEnabled: false, // Streamable also blocks
		}

		// Create dependencies
		db, err := database.Initialize(":memory:")
		if err != nil {
			t.Fatalf("Failed to create test database: %v", err)
		}
		defer db.Close()

		beaconMgr := newMockBeaconManager()
		sessionMgr := newMockSessionManager()

		// Create manager
		manager, err := NewManager(cfg, beaconMgr, sessionMgr, db)
		if err != nil {
			t.Fatalf("Failed to create manager: %v", err)
		}

		// Start all servers
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel() // Ensure context is cancelled to stop servers

		err = manager.StartAll(ctx)
		if err != nil {
			t.Errorf("StartAll() error = %v", err)
		}

		// Give servers time to start
		time.Sleep(200 * time.Millisecond)

		// Verify SSE server is running
		if !manager.IsAnyRunning() {
			t.Error("Expected at least one server to be running")
		}

		// Shutdown
		err = manager.Shutdown(ctx)
		if err != nil {
			t.Errorf("Shutdown() error = %v", err)
		}

		// Verify all servers stopped
		time.Sleep(100 * time.Millisecond) // Give shutdown a moment
		if manager.IsAnyRunning() {
			t.Error("Expected all servers to be stopped after shutdown")
		}
	}()

	select {
	case <-done:
		// Test completed
	case <-time.After(30 * time.Second):
		t.Fatal("Test timed out after 30 seconds")
	}
}

func TestManagerGetServer(t *testing.T) {
	cfg := &config.Config{
		Name:              "Test MCP Server",
		Version:           "1.0.0",
		SSEEnabled:        true,
		SSEPort:           ":8444",
		StdioEnabled:      true,
		StreamableEnabled: true,
		StreamablePort:    ":50012",
	}

	db, _ := database.Initialize(":memory:")
	defer db.Close()

	manager, err := NewManager(cfg, newMockBeaconManager(), newMockSessionManager(), db)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	tests := []struct {
		transport config.Transport
		wantNil   bool
	}{
		{config.TransportSSE, false},
		{config.TransportStdio, false},
		{config.TransportStreamable, false},
		{config.Transport("invalid"), true},
	}

	for _, tt := range tests {
		t.Run(string(tt.transport), func(t *testing.T) {
			server, ok := manager.GetServer(tt.transport)
			if ok == tt.wantNil {
				t.Errorf("GetServer(%s) ok = %v, want ok = %v", tt.transport, ok, !tt.wantNil)
			}
			if ok && server == nil {
				t.Errorf("GetServer(%s) returned nil server with ok=true", tt.transport)
			}
		})
	}
}

func TestManagerEnabledTransports(t *testing.T) {
	tests := []struct {
		name      string
		config    *config.Config
		wantCount int
	}{
		{
			name: "No transports enabled",
			config: &config.Config{
				SSEEnabled:        false,
				StdioEnabled:      false,
				StreamableEnabled: false,
			},
			wantCount: 0,
		},
		{
			name: "Only SSE enabled",
			config: &config.Config{
				SSEEnabled:        true,
				SSEPort:           ":8444",
				StdioEnabled:      false,
				StreamableEnabled: false,
			},
			wantCount: 1,
		},
		{
			name: "Two transports enabled",
			config: &config.Config{
				SSEEnabled:        true,
				SSEPort:           ":8444",
				StdioEnabled:      true,
				StreamableEnabled: false,
			},
			wantCount: 2,
		},
		{
			name: "All transports enabled",
			config: &config.Config{
				SSEEnabled:        true,
				SSEPort:           ":8444",
				StdioEnabled:      true,
				StreamableEnabled: true,
				StreamablePort:    ":50012",
			},
			wantCount: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, _ := database.Initialize(":memory:")
			defer db.Close()

			manager, err := NewManager(tt.config, newMockBeaconManager(), newMockSessionManager(), db)
			if err != nil {
				t.Fatalf("Failed to create manager: %v", err)
			}

			transports := manager.EnabledTransports()
			if len(transports) != tt.wantCount {
				t.Errorf("EnabledTransports() returned %d transports, want %d", len(transports), tt.wantCount)
			}
		})
	}
}

func TestManagerConcurrentOperations(t *testing.T) {
	cfg := &config.Config{
		Name:       "Test MCP Server",
		Version:    "1.0.0",
		SSEEnabled: true,
		SSEPort:    ":0",
	}

	db, _ := database.Initialize(":memory:")
	defer db.Close()

	manager, err := NewManager(cfg, newMockBeaconManager(), newMockSessionManager(), db)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// Run concurrent operations
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			// Concurrent GetServer calls
			_, _ = manager.GetServer(config.TransportSSE)
			_, _ = manager.GetServer(config.TransportStdio)

			// Concurrent IsAnyRunning calls
			_ = manager.IsAnyRunning()

			// Concurrent EnabledTransports calls
			_ = manager.EnabledTransports()

			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestManagerLifecycleWithContext(t *testing.T) {
	cfg := &config.Config{
		Name:       "Test MCP Server",
		Version:    "1.0.0",
		SSEEnabled: true,
		SSEPort:    ":0",
	}

	db, _ := database.Initialize(":memory:")
	defer db.Close()

	manager, err := NewManager(cfg, newMockBeaconManager(), newMockSessionManager(), db)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// Test with cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err = manager.StartAll(ctx)
	if err == nil {
		t.Error("Expected error when starting with cancelled context")
	}

	// Test shutdown with timeout
	ctx2, cancel2 := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel2()

	// Start normally
	err = manager.StartAll(context.Background())
	if err != nil {
		t.Fatalf("Failed to start: %v", err)
	}

	// Shutdown with timeout context
	err = manager.Shutdown(ctx2)
	if err != nil {
		t.Errorf("Shutdown with timeout failed: %v", err)
	}
}
