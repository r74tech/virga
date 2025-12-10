package stdio

import (
	"context"
	"testing"

	"github.com/r74tech/virga/internal/server/beacons"
	"github.com/r74tech/virga/internal/server/database"
	"github.com/r74tech/virga/internal/server/mcp/config"
	"github.com/r74tech/virga/internal/server/session"
)

func TestNewServer(t *testing.T) {
	// Setup
	cfg := &config.Config{
		Name:         "Test STDIO Server",
		Version:      "1.0.0",
		StdioEnabled: true,
	}

	beaconMgr := beacons.NewManager()
	sessionMgr := session.NewManager()

	db, err := database.Initialize(":memory:")
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	// Test server creation
	server, err := NewServer(cfg, beaconMgr, sessionMgr, db)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	if server == nil {
		t.Fatal("Expected server, got nil")
	}

	// Verify fields
	if server.config != cfg {
		t.Error("Config mismatch")
	}
	if server.beaconManager != beaconMgr {
		t.Error("BeaconManager mismatch")
	}
	if server.sessionManager != sessionMgr {
		t.Error("SessionManager mismatch")
	}
	if server.db != db {
		t.Error("Database mismatch")
	}
	if server.isRunning {
		t.Error("Server should not be running initially")
	}
}

func TestServerLifecycle(t *testing.T) {
	// Setup
	cfg := &config.Config{
		Name:         "Test STDIO Server",
		Version:      "1.0.0",
		StdioEnabled: true,
	}

	beaconMgr := beacons.NewManager()
	sessionMgr := session.NewManager()

	db, err := database.Initialize(":memory:")
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	server, err := NewServer(cfg, beaconMgr, sessionMgr, db)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// Test IsRunning before start
	if server.IsRunning() {
		t.Error("Server should not be running initially")
	}

	// Test Transport method
	if server.Transport() != config.TransportStdio {
		t.Errorf("Expected transport %v, got %v", config.TransportStdio, server.Transport())
	}

	// Note: We can't easily test Start() for STDIO as it blocks on stdin/stdout
	// In a real test environment, we would need to set up pipes or mock the transport

	// Test shutdown on non-running server
	ctx := context.Background()
	err = server.Shutdown(ctx)
	if err != nil {
		t.Errorf("Shutdown on non-running server should not error: %v", err)
	}
}

func TestServerStartAlreadyRunning(t *testing.T) {
	// Setup
	cfg := &config.Config{
		Name:         "Test STDIO Server",
		Version:      "1.0.0",
		StdioEnabled: true,
	}

	beaconMgr := beacons.NewManager()
	sessionMgr := session.NewManager()

	db, err := database.Initialize(":memory:")
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	server, err := NewServer(cfg, beaconMgr, sessionMgr, db)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// Manually set isRunning to true
	server.mu.Lock()
	server.isRunning = true
	server.mu.Unlock()

	// Try to start again
	ctx := context.Background()
	err = server.Start(ctx)
	if err == nil {
		t.Error("Expected error when starting already running server")
	}
}

func TestServerConcurrency(t *testing.T) {
	// Setup
	cfg := &config.Config{
		Name:         "Test STDIO Server",
		Version:      "1.0.0",
		StdioEnabled: true,
	}

	beaconMgr := beacons.NewManager()
	sessionMgr := session.NewManager()

	db, err := database.Initialize(":memory:")
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	server, err := NewServer(cfg, beaconMgr, sessionMgr, db)
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
