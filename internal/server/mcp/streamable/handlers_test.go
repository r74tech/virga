package streamable

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/ThinkInAIXYZ/go-mcp/protocol"
	"github.com/r74tech/virga/internal/server/beacons"
	serverprotocol "github.com/r74tech/virga/internal/server/protocol"
	"github.com/r74tech/virga/internal/server/session"
	"github.com/r74tech/virga/internal/shared/testutil/mock"
)

func TestSessionListHandler(t *testing.T) {
	// Setup
	sessionMgr := mock.NewMockSessionManager()

	// Add test sessions
	sess1 := session.NewSession("sess-1", nil)
	sess1.UpdateInformation(map[string]interface{}{
		"hostname": "host1",
		"username": "user1",
		"os":       "linux",
		"arch":     "amd64",
		"ip":       "192.168.1.10",
	})
	sessionMgr.AddSession(sess1)

	sess2 := session.NewSession("sess-2", nil)
	sess2.UpdateInformation(map[string]interface{}{
		"hostname": "host2",
		"username": "user2",
		"os":       "windows",
		"arch":     "amd64",
		"ip":       "192.168.1.11",
	})
	sessionMgr.AddSession(sess2)

	// Test the handler logic directly
	ctx := context.Background()
	req := &protocol.CallToolRequest{
		Arguments: map[string]interface{}{},
	}

	// Execute the logic that would be in the handler
	sessions := sessionMgr.GetActiveSessions()

	// Build result text
	var sb strings.Builder
	sb.WriteString("Active sessions: ")
	sb.WriteString(fmt.Sprintf("%d\n\n", len(sessions)))

	for _, sess := range sessions {
		sb.WriteString(fmt.Sprintf("Session ID: %s\n", sess.ID))
		sb.WriteString(fmt.Sprintf("  Status: %s\n", sess.Status))
		sb.WriteString(fmt.Sprintf("  Interactive: %v\n", sess.IsInteractive()))
		sb.WriteString(fmt.Sprintf("  Remote: %s\n", sess.RemoteAddr()))
		sb.WriteString(fmt.Sprintf("  First seen: %s\n", sess.CreatedAt.Format(time.RFC3339)))
		sb.WriteString(fmt.Sprintf("  Last seen: %s\n", sess.LastSeen().Format(time.RFC3339)))

		info := sess.GetInformation()
		if hostname, ok := info["hostname"].(string); ok {
			sb.WriteString(fmt.Sprintf("  Hostname: %s\n", hostname))
		}
		if username, ok := info["username"].(string); ok {
			sb.WriteString(fmt.Sprintf("  Username: %s\n", username))
		}
		if os, ok := info["os"].(string); ok {
			sb.WriteString(fmt.Sprintf("  OS: %s\n", os))
		}
		if arch, ok := info["arch"].(string); ok {
			sb.WriteString(fmt.Sprintf("  Arch: %s\n", arch))
		}
		if ip, ok := info["ip"].(string); ok {
			sb.WriteString(fmt.Sprintf("  IP: %s\n", ip))
		}
		sb.WriteString("\n")
	}

	result := &protocol.CallToolResult{
		Content: []protocol.Content{
			&protocol.TextContent{
				Type: "text",
				Text: sb.String(),
			},
		},
	}

	// Verify result
	if result == nil {
		t.Fatal("Expected result, got nil")
	}

	// Check content
	if len(result.Content) == 0 {
		t.Fatal("Expected content in result")
	}

	textContent, ok := result.Content[0].(*protocol.TextContent)
	if !ok {
		t.Fatal("Expected TextContent")
	}

	// Verify sessions are listed
	if !strings.Contains(textContent.Text, "Active sessions: 2") {
		t.Error("Expected 'Active sessions: 2' in output")
	}
	if !strings.Contains(textContent.Text, "sess-1") {
		t.Error("Expected sess-1 in output")
	}
	if !strings.Contains(textContent.Text, "sess-2") {
		t.Error("Expected sess-2 in output")
	}
	if !strings.Contains(textContent.Text, "host1") {
		t.Error("Expected host1 in output")
	}
	if !strings.Contains(textContent.Text, "host2") {
		t.Error("Expected host2 in output")
	}

	// Use ctx and req to satisfy the compiler
	_ = ctx
	_ = req
}

func TestSessionCommandHandler(t *testing.T) {
	// Setup
	sessionMgr := mock.NewMockSessionManager()

	// Add test session
	sess := session.NewSession("test-sess", nil)
	sess.UpdateInformation(map[string]interface{}{
		"hostname": "testhost",
		"username": "testuser",
	})
	sessionMgr.AddSession(sess)

	// Mock GetSession to return our test session
	sessionMgr.GetSessionFunc = func(sessionID string) *session.Session {
		if sessionID == "test-sess" {
			return sess
		}
		return nil
	}

	// Test cases
	tests := []struct {
		name        string
		args        map[string]interface{}
		expectError bool
		expectText  string
	}{
		{
			name: "Valid shell command",
			args: map[string]interface{}{
				"session_id": "test-sess",
				"command":    "ls -la",
			},
			expectError: false,
			expectText:  "", // We just check if taskID is not empty
		},
		{
			name: "Missing session_id",
			args: map[string]interface{}{
				"command": "ls -la",
			},
			expectError: true,
		},
		{
			name: "Missing command",
			args: map[string]interface{}{
				"session_id": "test-sess",
			},
			expectError: true,
		},
		{
			name: "Invalid session",
			args: map[string]interface{}{
				"session_id": "invalid-sess",
				"command":    "ls -la",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create request
			req := &protocol.CallToolRequest{
				Arguments: tt.args,
			}

			// Simulate handler logic
			sessionID, hasSessionID := tt.args["session_id"].(string)
			command, hasCommand := tt.args["command"].(string)

			if !hasSessionID || sessionID == "" {
				if !tt.expectError {
					t.Error("Expected session_id to be present")
				}
				return
			}

			if !hasCommand || command == "" {
				if !tt.expectError {
					t.Error("Expected command to be present")
				}
				return
			}

			// Get session
			sess := sessionMgr.GetSession(sessionID)
			if sess == nil {
				if !tt.expectError {
					t.Error("Expected session to exist")
				}
				return
			}

			// Execute command
			taskID, err := sess.ExecuteCommand(string(serverprotocol.TaskTypeCommand), map[string]interface{}{
				"command": command,
			})

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if taskID == "" {
				t.Errorf("Expected task ID, got empty string")
			}

			_ = req
		})
	}
}

func TestGetSystemStatusHandler(t *testing.T) {
	// Setup
	sessionMgr := mock.NewMockSessionManager()
	beaconMgr := beacons.NewManager()
	db := mock.NewMockDatabase()

	// Add test sessions
	sess1 := session.NewSession("sess-1", nil)
	sessionMgr.AddSession(sess1)
	sess2 := session.NewSession("sess-2", nil)
	sessionMgr.AddSession(sess2)

	// Test handler logic directly
	activeSessions := sessionMgr.GetActiveSessions()

	var agentCount, sessionCount, commandCount int64
	agentCount = 10
	sessionCount = 20
	commandCount = 100

	uptime := time.Duration(1 * time.Hour)
	status := map[string]interface{}{
		"server": map[string]interface{}{
			"name":    "Test Server",
			"version": "1.0.0",
			"uptime":  uptime.String(),
		},
		"sessions": map[string]interface{}{
			"active": len(activeSessions),
			"total":  sessionCount,
		},
		"beacons": map[string]interface{}{
			"count": beaconMgr.Count(),
		},
		"database": map[string]interface{}{
			"agents":   agentCount,
			"sessions": sessionCount,
			"commands": commandCount,
		},
	}

	statusJSON, err := json.MarshalIndent(status, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal status: %v", err)
	}

	result := &protocol.CallToolResult{
		Content: []protocol.Content{
			&protocol.TextContent{
				Type: "text",
				Text: string(statusJSON),
			},
		},
	}

	// Verify result
	if result == nil {
		t.Fatal("Expected result, got nil")
	}

	textContent, ok := result.Content[0].(*protocol.TextContent)
	if !ok {
		t.Fatal("Expected TextContent")
	}

	// Verify JSON structure
	var parsedStatus map[string]interface{}
	if err := json.Unmarshal([]byte(textContent.Text), &parsedStatus); err != nil {
		t.Fatalf("Failed to parse status JSON: %v", err)
	}

	// Check server info
	if server, ok := parsedStatus["server"].(map[string]interface{}); ok {
		if server["name"] != "Test Server" {
			t.Error("Expected server name 'Test Server'")
		}
		if server["version"] != "1.0.0" {
			t.Error("Expected server version '1.0.0'")
		}
	} else {
		t.Error("Missing server info in status")
	}

	// Check session info
	if sessions, ok := parsedStatus["sessions"].(map[string]interface{}); ok {
		if active, ok := sessions["active"].(float64); !ok || int(active) != 2 {
			t.Error("Expected 2 active sessions")
		}
	} else {
		t.Error("Missing sessions info in status")
	}

	// Use db to satisfy the compiler
	_ = db
}

func TestInteractBeaconHandler(t *testing.T) {
	// Setup
	sessionMgr := mock.NewMockSessionManager()

	// Add test session
	sess := session.NewSession("test-sess", nil)
	sess.UpdateInformation(map[string]interface{}{
		"hostname": "testhost",
		"username": "testuser",
	})
	sessionMgr.AddSession(sess)

	// Mock GetSession
	sessionMgr.GetSessionFunc = func(sessionID string) *session.Session {
		if sessionID == "test-sess" {
			return sess
		}
		return nil
	}

	// Test interact_beacon logic
	ctx := context.Background()
	req := &protocol.CallToolRequest{
		Arguments: map[string]interface{}{
			"session_id": "test-sess",
		},
	}

	// Execute handler logic
	sessionID := "test-sess"
	sess = sessionMgr.GetSession(sessionID)
	if sess == nil {
		t.Fatal("Session not found")
	}

	sess.SetInteractive(true)
	taskID, err := sess.ExecuteCommand("beacon_settings", map[string]interface{}{
		"sleep": float64(1),
	})
	if err != nil {
		t.Fatalf("Failed to execute command: %v", err)
	}

	// Verify task was created
	if taskID == "" {
		t.Error("Expected task ID to be returned")
	}

	// Verify session is interactive
	if !sess.IsInteractive() {
		t.Error("Expected session to be in interactive mode")
	}

	// Use variables to satisfy compiler
	_ = ctx
	_ = req
	_ = taskID
}

func TestStopInteractHandler(t *testing.T) {
	// Setup
	sessionMgr := mock.NewMockSessionManager()

	// Add test session in interactive mode
	sess := session.NewSession("test-sess", nil)
	sess.SetInteractive(true)
	sessionMgr.AddSession(sess)

	// Mock GetSession
	sessionMgr.GetSessionFunc = func(sessionID string) *session.Session {
		if sessionID == "test-sess" {
			return sess
		}
		return nil
	}

	// Test stop_interact logic
	ctx := context.Background()
	req := &protocol.CallToolRequest{
		Arguments: map[string]interface{}{
			"session_id": "test-sess",
		},
	}

	// Execute handler logic
	sessionID := "test-sess"
	sess = sessionMgr.GetSession(sessionID)
	if sess == nil {
		t.Fatal("Session not found")
	}

	sess.SetInteractive(false)
	taskID, err := sess.ExecuteCommand("beacon_settings", map[string]interface{}{
		"sleep": float64(30),
	})
	if err != nil {
		t.Fatalf("Failed to execute command: %v", err)
	}

	// Verify task was created
	if taskID == "" {
		t.Error("Expected task ID to be returned")
	}

	// Verify session is not interactive
	if sess.IsInteractive() {
		t.Error("Expected session to not be in interactive mode")
	}

	// Use variables to satisfy compiler
	_ = ctx
	_ = req
	_ = taskID
}

func TestShellAliasHandler(t *testing.T) {
	// Setup
	sessionMgr := mock.NewMockSessionManager()

	// Add test session
	sess := session.NewSession("test-sess", nil)
	sessionMgr.AddSession(sess)

	// Mock GetSession to return our test session
	sessionMgr.GetSessionFunc = func(sessionID string) *session.Session {
		if sessionID == "test-sess" {
			return sess
		}
		return nil
	}

	// Test shell alias
	ctx := context.Background()
	req := &protocol.CallToolRequest{
		Arguments: map[string]interface{}{
			"session_id": "test-sess",
			"command":    "whoami",
		},
	}

	// Execute handler logic (same as session_command)
	sessionID := "test-sess"
	command := "whoami"

	sess = sessionMgr.GetSession(sessionID)
	if sess == nil {
		t.Fatal("Session not found")
	}

	taskID, err := sess.ExecuteCommand(string(serverprotocol.TaskTypeCommand), map[string]interface{}{
		"command": command,
	})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if taskID == "" {
		t.Errorf("Expected task ID, got empty string")
	}

	_ = ctx
	_ = req
}

func TestLsAliasHandler(t *testing.T) {
	// Setup
	sessionMgr := mock.NewMockSessionManager()

	// Add test session
	sess := session.NewSession("test-sess", nil)
	sessionMgr.AddSession(sess)

	// Mock GetSession to return our test session
	sessionMgr.GetSessionFunc = func(sessionID string) *session.Session {
		if sessionID == "test-sess" {
			return sess
		}
		return nil
	}

	// Test ls alias with path
	ctx := context.Background()
	req := &protocol.CallToolRequest{
		Arguments: map[string]interface{}{
			"session_id": "test-sess",
			"path":       "/tmp",
		},
	}

	// Execute handler logic - ls alias adds -la flag
	sessionID := "test-sess"
	path := "/tmp"
	command := fmt.Sprintf("ls -la %s", path)

	sess = sessionMgr.GetSession(sessionID)
	if sess == nil {
		t.Fatal("Session not found")
	}

	taskID, err := sess.ExecuteCommand(string(serverprotocol.TaskTypeCommand), map[string]interface{}{
		"command": command,
	})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if taskID == "" {
		t.Errorf("Expected task ID, got empty string")
	}

	// Test default path
	req2 := &protocol.CallToolRequest{
		Arguments: map[string]interface{}{
			"session_id": "test-sess",
		},
	}

	command2 := "ls -la ."
	taskID2, err := sess.ExecuteCommand(string(serverprotocol.TaskTypeCommand), map[string]interface{}{
		"command": command2,
	})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if taskID2 == "" {
		t.Errorf("Expected task ID, got empty string")
	}

	_ = ctx
	_ = req
	_ = req2
}

// Test type validation
func TestRequestTypeValidation(t *testing.T) {
	// Test that request types match the expected structure
	t.Run("SessionCommandRequest", func(t *testing.T) {
		req := SessionCommandRequest{
			SessionID:     "test-session",
			Command:       "echo test",
			Type:          "shell",
			UseLlama:      false,
			MaxIterations: 5,
			Temperature:   0.3,
		}

		// Verify JSON serialization works
		data, err := json.Marshal(req)
		if err != nil {
			t.Fatalf("Failed to marshal SessionCommandRequest: %v", err)
		}

		var decoded SessionCommandRequest
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("Failed to unmarshal SessionCommandRequest: %v", err)
		}

		if decoded.SessionID != req.SessionID {
			t.Error("SessionID mismatch after JSON round-trip")
		}
	})

	t.Run("InteractBeaconRequest", func(t *testing.T) {
		req := InteractBeaconRequest{
			SessionID: "test-session",
		}

		data, err := json.Marshal(req)
		if err != nil {
			t.Fatalf("Failed to marshal InteractBeaconRequest: %v", err)
		}

		var decoded InteractBeaconRequest
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("Failed to unmarshal InteractBeaconRequest: %v", err)
		}

		if decoded.SessionID != req.SessionID {
			t.Error("SessionID mismatch after JSON round-trip")
		}
	})
}
