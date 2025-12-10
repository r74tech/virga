package sse

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/ThinkInAIXYZ/go-mcp/protocol"
	"github.com/r74tech/virga/internal/server/beacons"
	"github.com/r74tech/virga/internal/server/database"
	"github.com/r74tech/virga/internal/server/mcp/config"
	"github.com/r74tech/virga/internal/server/session"
)

func TestGetSessionsList(t *testing.T) {
	// Setup
	cfg := &config.Config{
		Name:    "Test SSE Server",
		Version: "1.0.0",
	}

	beaconMgr := beacons.NewManager()
	sessionMgr := session.NewManager()

	// Create test database
	db, err := database.Initialize(":memory:")
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	server, err := NewServer(cfg, beaconMgr, sessionMgr, db)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// Add test sessions
	sess1 := session.NewSession("session-1", nil)
	sess1.UpdateInformation(map[string]interface{}{
		"hostname": "host1",
		"username": "user1",
		"os":       "linux",
		"arch":     "amd64",
		"ip":       "192.168.1.100",
	})
	sessionMgr.AddSession(sess1)

	sess2 := session.NewSession("session-2", nil)
	sess2.UpdateInformation(map[string]interface{}{
		"hostname": "host2",
		"username": "user2",
		"os":       "windows",
		"arch":     "amd64",
		"ip":       "192.168.1.101",
	})
	sessionMgr.AddSession(sess2)

	// Test request
	ctx := context.Background()
	req := &protocol.ReadResourceRequest{
		URI: "sessions://list",
	}

	result, err := server.getSessionsList(ctx, req)
	if err != nil {
		t.Fatalf("getSessionsList failed: %v", err)
	}

	if result == nil {
		t.Fatal("Expected result, got nil")
	}

	if len(result.Contents) == 0 {
		t.Fatal("Expected contents in result")
	}

	// Check content
	textContents, ok := result.Contents[0].(*protocol.TextResourceContents)
	if !ok {
		t.Fatal("Expected TextResourceContents")
	}

	// Verify it's valid JSON containing sessions
	if !strings.Contains(textContents.Text, "session-1") {
		t.Error("Expected session-1 in output")
	}
	if !strings.Contains(textContents.Text, "session-2") {
		t.Error("Expected session-2 in output")
	}
}

func TestGetBeaconsList(t *testing.T) {
	// Setup
	cfg := &config.Config{
		Name:    "Test SSE Server",
		Version: "1.0.0",
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

	// Register test beacons
	beaconConfig := &beacons.BeaconConfig{
		SleepTime:         30,
		Jitter:            20,
		MaxMissedCheckins: 3,
		KillDate:          time.Now().Add(24 * time.Hour),
	}
	beaconMgr.RegisterBeacon("beacon-1", beaconConfig)
	beaconMgr.RegisterBeacon("beacon-2", beaconConfig)

	// Test request
	ctx := context.Background()
	req := &protocol.ReadResourceRequest{
		URI: "beacons://list",
	}

	result, err := server.getBeaconsList(ctx, req)
	if err != nil {
		t.Fatalf("getBeaconsList failed: %v", err)
	}

	if result == nil {
		t.Fatal("Expected result, got nil")
	}

	// Check content
	textContents, ok := result.Contents[0].(*protocol.TextResourceContents)
	if !ok {
		t.Fatal("Expected TextResourceContents")
	}

	// Should have JSON array with beacons
	if !strings.Contains(textContents.Text, "[") {
		t.Error("Expected JSON array in beacon list")
	}
}

func TestGetSystemStatus(t *testing.T) {
	// Setup
	cfg := &config.Config{
		Name:    "Test SSE Server",
		Version: "1.0.0",
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

	// Test request
	ctx := context.Background()
	req := &protocol.ReadResourceRequest{
		URI: "system://status",
	}

	result, err := server.getSystemStatus(ctx, req)
	if err != nil {
		t.Fatalf("getSystemStatus failed: %v", err)
	}

	if result == nil {
		t.Fatal("Expected result, got nil")
	}

	// Check content
	textContents, ok := result.Contents[0].(*protocol.TextResourceContents)
	if !ok {
		t.Fatal("Expected TextResourceContents")
	}

	// Verify expected fields
	if !strings.Contains(textContents.Text, "uptime") {
		t.Error("Expected 'uptime' in system status")
	}
	if !strings.Contains(textContents.Text, "active_sessions") {
		t.Error("Expected 'active_sessions' in system status")
	}
}

func TestGetSessionDetail(t *testing.T) {
	// Setup
	cfg := &config.Config{
		Name:    "Test SSE Server",
		Version: "1.0.0",
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

	// Add test session
	sess := session.NewSession("test-session-123", nil)
	sess.UpdateInformation(map[string]interface{}{
		"hostname": "testhost",
		"username": "testuser",
		"os":       "linux",
		"arch":     "amd64",
		"ip":       "192.168.1.100",
	})
	sessionMgr.AddSession(sess)

	// Test valid session detail
	ctx := context.Background()
	req := &protocol.ReadResourceRequest{
		URI: "sessions://detail/test-session-123",
	}

	result, err := server.getSessionDetail(ctx, req)
	if err != nil {
		t.Fatalf("getSessionDetail failed: %v", err)
	}

	if result == nil {
		t.Fatal("Expected result, got nil")
	}

	// Test invalid session
	req2 := &protocol.ReadResourceRequest{
		URI: "sessions://detail/invalid-session",
	}

	_, err = server.getSessionDetail(ctx, req2)
	if err == nil {
		t.Error("Expected error for invalid session")
	}

	// Test malformed URI
	req3 := &protocol.ReadResourceRequest{
		URI: "sessions://detail",
	}

	_, err = server.getSessionDetail(ctx, req3)
	if err == nil {
		t.Error("Expected error for malformed URI")
	}
}

func TestGetCommandHistory(t *testing.T) {
	// Setup
	cfg := &config.Config{
		Name:    "Test SSE Server",
		Version: "1.0.0",
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

	// Test request
	ctx := context.Background()
	req := &protocol.ReadResourceRequest{
		URI: "history://commands/test-session-123",
	}

	result, err := server.getCommandHistory(ctx, req)
	if err != nil {
		t.Fatalf("getCommandHistory failed: %v", err)
	}

	if result == nil {
		t.Fatal("Expected result, got nil")
	}

	// Check content
	textContents, ok := result.Contents[0].(*protocol.TextResourceContents)
	if !ok {
		t.Fatal("Expected TextResourceContents")
	}

	// Should return empty array for no history
	if !strings.Contains(textContents.Text, "[]") {
		t.Error("Expected empty JSON array for no history")
	}
}

func TestGetSessionFiles(t *testing.T) {
	// Setup
	cfg := &config.Config{
		Name:    "Test SSE Server",
		Version: "1.0.0",
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

	// Test request
	ctx := context.Background()
	req := &protocol.ReadResourceRequest{
		URI: "files://session/test-session-123",
	}

	result, err := server.getSessionFiles(ctx, req)
	if err != nil {
		t.Fatalf("getSessionFiles failed: %v", err)
	}

	if result == nil {
		t.Fatal("Expected result, got nil")
	}

	// Check content
	textContents, ok := result.Contents[0].(*protocol.TextResourceContents)
	if !ok {
		t.Fatal("Expected TextResourceContents")
	}

	// Should return empty array for no operations
	if !strings.Contains(textContents.Text, "[]") {
		t.Error("Expected empty JSON array for no file operations")
	}
}

func TestInvalidResourceURI(t *testing.T) {
	// Setup
	cfg := &config.Config{
		Name:    "Test SSE Server",
		Version: "1.0.0",
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

	tests := []struct {
		name string
		uri  string
	}{
		{"Invalid scheme", "invalid://resource"},
		{"Missing ID in detail", "sessions://detail/"},
		{"Missing ID in history", "history://commands/"},
		{"Missing ID in files", "files://session/"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Each handler should handle its own invalid URIs
			// This test ensures we don't panic on malformed URIs
			ctx := context.Background()
			req := &protocol.ReadResourceRequest{
				URI: tt.uri,
			}

			// Try each handler - they should handle errors gracefully
			switch {
			case strings.HasPrefix(tt.uri, "sessions://detail"):
				_, err := server.getSessionDetail(ctx, req)
				if err == nil {
					t.Error("Expected error for invalid URI")
				}
			case strings.HasPrefix(tt.uri, "history://commands"):
				_, err := server.getCommandHistory(ctx, req)
				if err == nil {
					t.Error("Expected error for invalid URI")
				}
			case strings.HasPrefix(tt.uri, "files://session"):
				_, err := server.getSessionFiles(ctx, req)
				if err == nil {
					t.Error("Expected error for invalid URI")
				}
			}
		})
	}
}
