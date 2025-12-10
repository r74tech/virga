package streamable

import (
	"context"
	"testing"
	"time"

	"github.com/r74tech/virga/internal/server/beacons"
	"github.com/r74tech/virga/internal/server/database"
	"github.com/r74tech/virga/internal/server/mcp/config"
	"github.com/r74tech/virga/internal/server/session"
)

func TestNewStreamableServer(t *testing.T) {
	// Setup
	beaconMgr := beacons.NewManager()
	sessionMgr := session.NewManager()

	db, err := database.Initialize(":memory:")
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	tests := []struct {
		name    string
		cfg     *Config
		wantErr bool
	}{
		{
			name: "Valid configuration",
			cfg: &Config{
				Name:    "Test Streamable Server",
				Version: "1.0.0",
				Port:    ":8081",
			},
			wantErr: false,
		},
		{
			name: "Empty configuration",
			cfg: &Config{
				Name:    "",
				Version: "",
				Port:    "",
			},
			wantErr: false, // Should still create server with defaults
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, err := NewStreamableServer(tt.cfg, beaconMgr, sessionMgr, db)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
				if server == nil {
					t.Error("Expected server, got nil")
				}
			}
		})
	}
}

func TestStreamableServerLifecycle(t *testing.T) {
	done := make(chan bool)
	go func() {
		defer func() { done <- true }()
		// Setup
		beaconMgr := beacons.NewManager()
		sessionMgr := session.NewManager()

		db, err := database.Initialize(":memory:")
		if err != nil {
			t.Fatalf("Failed to create database: %v", err)
		}
		defer db.Close()

		cfg := &Config{
			Name:    "Test Streamable Server",
			Version: "1.0.0",
			Port:    ":0", // Use random port
		}

		server, err := NewStreamableServer(cfg, beaconMgr, sessionMgr, db)
		if err != nil {
			t.Fatalf("Failed to create server: %v", err)
		}

		// Test initial state
		if server.IsRunning() {
			t.Error("Server should not be running initially")
		}

		// Test Transport method
		if server.Transport() != config.TransportStreamable {
			t.Errorf("Expected transport %v, got %v", config.TransportStreamable, server.Transport())
		}

		// Start server in goroutine
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		go func() {
			_ = server.Start(ctx)
		}()

		// Wait a bit for server to start
		time.Sleep(100 * time.Millisecond)

		// Check if running
		if !server.IsRunning() {
			t.Error("Server should be running after Start")
		}

		// Shutdown
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			t.Errorf("Failed to shutdown server: %v", err)
		}

		// Check if stopped
		time.Sleep(100 * time.Millisecond) // Give shutdown a moment
		if server.IsRunning() {
			t.Error("Server should not be running after Shutdown")
		}
	}()

	select {
	case <-done:
		// Test completed
	case <-time.After(30 * time.Second):
		t.Fatal("Test timed out after 30 seconds")
	}
}

func TestStreamableServerConcurrency(t *testing.T) {
	// Setup
	beaconMgr := beacons.NewManager()
	sessionMgr := session.NewManager()

	db, err := database.Initialize(":memory:")
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	cfg := &Config{
		Name:    "Test Streamable Server",
		Version: "1.0.0",
		Port:    ":0",
	}

	server, err := NewStreamableServer(cfg, beaconMgr, sessionMgr, db)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// Test concurrent IsRunning calls
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				_ = server.IsRunning()
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}
