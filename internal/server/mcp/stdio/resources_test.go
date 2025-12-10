package stdio

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/ThinkInAIXYZ/go-mcp/protocol"
	"github.com/r74tech/virga/internal/server/beacons"
	"github.com/r74tech/virga/internal/server/database"
	"github.com/r74tech/virga/internal/server/mcp/config"
	"github.com/r74tech/virga/internal/server/session"
)

func TestResourceSessionsList(t *testing.T) {
	// Setup
	cfg := &config.Config{
		Name:    "Test STDIO Server",
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

	// Call the handler directly
	result, err := server.getSessionsListHandler(ctx, req)
	if err != nil {
		t.Fatalf("resourceSessionsList failed: %v", err)
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

	// Parse JSON
	var sessions []map[string]interface{}
	if err := json.Unmarshal([]byte(textContents.Text), &sessions); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	if len(sessions) != 2 {
		t.Errorf("Expected 2 sessions, got %d", len(sessions))
	}

	// Verify session data
	foundSession1 := false
	foundSession2 := false
	for _, sess := range sessions {
		if sess["id"] == "session-1" {
			foundSession1 = true
			if sess["hostname"] != "host1" {
				t.Error("Session 1 hostname mismatch")
			}
		}
		if sess["id"] == "session-2" {
			foundSession2 = true
			if sess["hostname"] != "host2" {
				t.Error("Session 2 hostname mismatch")
			}
		}
	}

	if !foundSession1 || !foundSession2 {
		t.Error("Not all sessions found in result")
	}
}

func TestResourceSessionInfo(t *testing.T) {
	// Setup
	cfg := &config.Config{
		Name:    "Test STDIO Server",
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
	_ = server // Mark as used

	// Add test session
	sess := session.NewSession("test-session-123", nil)
	sess.UpdateInformation(map[string]interface{}{
		"hostname":   "testhost",
		"username":   "testuser",
		"os":         "linux",
		"arch":       "amd64",
		"ip":         "192.168.1.100",
		"domain":     "test.local",
		"privileges": "user",
	})
	sessionMgr.AddSession(sess)

	// Test valid session
	ctx := context.Background()
	req := &protocol.ReadResourceRequest{
		URI: "sessions://test-session-123",
	}
	_ = ctx // Mark as used
	_ = req // Mark as used

	// STDIO doesn't have individual session info resource
	// Skip this test for STDIO
	t.Skip("STDIO doesn't implement individual session info resource")

	// Test cases for invalid URIs are not applicable for STDIO
	// as it doesn't support individual session resources
}

func TestResourceBeaconsList(t *testing.T) {
	// Setup
	cfg := &config.Config{
		Name:    "Test STDIO Server",
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

	// Note: Since beaconMgr.RegisterBeacon only registers configurations,
	// not actual beacon objects, we'll test with empty beacon list
	// In real usage, beacons would be added via different methods

	// Test request
	ctx := context.Background()
	req := &protocol.ReadResourceRequest{
		URI: "beacons://list",
	}

	result, err := server.getBeaconsListHandler(ctx, req)
	if err != nil {
		t.Fatalf("resourceBeaconsList failed: %v", err)
	}

	if result == nil {
		t.Fatal("Expected result, got nil")
	}

	// Check content
	textContents, ok := result.Contents[0].(*protocol.TextResourceContents)
	if !ok {
		t.Fatal("Expected TextResourceContents")
	}

	// Parse JSON
	var beacons []map[string]interface{}
	if err := json.Unmarshal([]byte(textContents.Text), &beacons); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	// Since we don't have actual beacon objects in the test,
	// we expect count to be 0
	if len(beacons) != 0 {
		t.Errorf("Expected 0 beacons, got %d", len(beacons))
	}
}

func TestResourceSystemStatus(t *testing.T) {
	// Setup
	cfg := &config.Config{
		Name:    "Test STDIO Server",
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

	// Add test data
	sess := session.NewSession("session-1", nil)
	sessionMgr.AddSession(sess)
	beaconMgr.RegisterBeacon("beacon-1", &beacons.BeaconConfig{})

	// Test request
	ctx := context.Background()
	req := &protocol.ReadResourceRequest{
		URI: "system://status",
	}

	result, err := server.getSystemStatusHandler(ctx, req)
	if err != nil {
		t.Fatalf("resourceSystemStatus failed: %v", err)
	}

	if result == nil {
		t.Fatal("Expected result, got nil")
	}

	// Check content
	textContents, ok := result.Contents[0].(*protocol.TextResourceContents)
	if !ok {
		t.Fatal("Expected TextResourceContents")
	}

	// Parse JSON
	var status map[string]interface{}
	if err := json.Unmarshal([]byte(textContents.Text), &status); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	// Verify structure
	if _, ok := status["server_type"]; !ok {
		t.Error("Expected 'server_type' in status")
	}
	if _, ok := status["server_version"]; !ok {
		t.Error("Expected 'server_version' in status")
	}
	if _, ok := status["uptime"]; !ok {
		t.Error("Expected 'uptime' in status")
	}
	if _, ok := status["sessions"]; !ok {
		t.Error("Expected 'sessions' in status")
	}
	if _, ok := status["beacons"]; !ok {
		t.Error("Expected 'beacons' in status")
	}

	// Check values
	sessions := status["sessions"].(map[string]interface{})
	if sessions["active"].(float64) != 1 {
		t.Errorf("Expected 1 active session, got %v", sessions["active"])
	}

	beacons := status["beacons"].(map[string]interface{})
	// Since RegisterBeacon only registers configs, not actual beacons, expect 0
	if beacons["total"].(float64) != 0 {
		t.Errorf("Expected 0 beacons, got %v", beacons["total"])
	}
}

func TestResourceCommandHistory(t *testing.T) {
	// Setup
	cfg := &config.Config{
		Name:    "Test STDIO Server",
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
	_ = server // Mark as used

	// Test request
	ctx := context.Background()
	req := &protocol.ReadResourceRequest{
		URI: "history://commands/session-123",
	}
	_ = ctx // Mark as used
	_ = req // Mark as used

	// STDIO doesn't have command history resource
	t.Skip("STDIO doesn't implement command history resource")
}

func TestResourceLlamaStatus(t *testing.T) {
	// Setup
	cfg := &config.Config{
		Name:    "Test STDIO Server",
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
	_ = server // Mark as used

	// Test request
	ctx := context.Background()
	req := &protocol.ReadResourceRequest{
		URI: "llama://status/session-123",
	}
	_ = ctx // Mark as used
	_ = req // Mark as used

	// STDIO doesn't have Llama status resource
	t.Skip("STDIO doesn't implement Llama status resource")
}

func TestResourceMemDBHistory(t *testing.T) {
	// Setup
	cfg := &config.Config{
		Name:    "Test STDIO Server",
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
	_ = server // Mark as used

	// Test request
	ctx := context.Background()
	req := &protocol.ReadResourceRequest{
		URI: "memdb://history/session-123",
	}
	_ = ctx // Mark as used
	_ = req // Mark as used

	// STDIO doesn't have MemDB history resource
	t.Skip("STDIO doesn't implement MemDB history resource")
}
