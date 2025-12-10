package sse

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/r74tech/virga/internal/server/beacons"
	"github.com/r74tech/virga/internal/server/database"
	"github.com/r74tech/virga/internal/server/mcp/config"
	"github.com/r74tech/virga/internal/server/session"
)

// TestNewServer tests server creation
func TestNewServer(t *testing.T) {
	// Setup test dependencies
	cfg := &config.Config{
		Name:         "Test SSE Server",
		Version:      "1.0.0",
		SSEPort:      ":8445",
		SSEBasePath:  "/test/mcp",
		SSERemoteURL: "http://localhost:8445",
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

// TestServerLifecycle tests server start and shutdown
func TestServerLifecycle(t *testing.T) {
	// Setup test dependencies
	cfg := &config.Config{
		Name:         "Test SSE Server",
		Version:      "1.0.0",
		SSEPort:      ":8446",
		SSEBasePath:  "/test/mcp",
		SSERemoteURL: "http://localhost:8446",
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
	if server.Transport() != config.TransportSSE {
		t.Errorf("Expected transport %v, got %v", config.TransportSSE, server.Transport())
	}

	// Start server in goroutine
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- server.Start(ctx)
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Test IsRunning after start
	if !server.IsRunning() {
		t.Error("Server should be running after Start")
	}

	// Test shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	err = server.Shutdown(shutdownCtx)
	if err != nil {
		t.Errorf("Failed to shutdown server: %v", err)
	}
	if server.IsRunning() {
		t.Error("Server should not be running after Shutdown")
	}
}

// TestServerStartAlreadyRunning tests starting an already running server
func TestServerStartAlreadyRunning(t *testing.T) {
	cfg := &config.Config{
		Name:         "Test SSE Server",
		Version:      "1.0.0",
		SSEPort:      ":8447",
		SSEBasePath:  "/test/mcp",
		SSERemoteURL: "http://localhost:8447",
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
	if err != nil && !contains(err.Error(), "already running") {
		t.Errorf("Expected 'already running' error, got: %v", err)
	}
}

// TestShutdownNotRunning tests shutting down a server that's not running
func TestShutdownNotRunning(t *testing.T) {
	cfg := &config.Config{
		Name:         "Test SSE Server",
		Version:      "1.0.0",
		SSEPort:      ":8448",
		SSEBasePath:  "/test/mcp",
		SSERemoteURL: "http://localhost:8448",
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

	// Shutdown without starting
	ctx := context.Background()
	err = server.Shutdown(ctx)
	if err != nil {
		t.Errorf("Shutdown on non-running server should not error: %v", err)
	}
}

// Helper function
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}
