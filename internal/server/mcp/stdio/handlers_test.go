package stdio

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/ThinkInAIXYZ/go-mcp/protocol"
	"github.com/r74tech/virga/internal/server/beacons"
	"github.com/r74tech/virga/internal/server/database"
	"github.com/r74tech/virga/internal/server/mcp/config"
	serverprotocol "github.com/r74tech/virga/internal/server/protocol"
	"github.com/r74tech/virga/internal/server/session"
)

func TestHandleSessionList(t *testing.T) {
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
	req := &protocol.CallToolRequest{
		Arguments: map[string]interface{}{},
	}

	result, err := server.handleSessionList(ctx, req)
	if err != nil {
		t.Fatalf("handleSessionList failed: %v", err)
	}

	if result == nil {
		t.Fatal("Expected result, got nil")
	}

	if len(result.Content) == 0 {
		t.Fatal("Expected content in result")
	}

	// Check content
	textContent, ok := result.Content[0].(*protocol.TextContent)
	if !ok {
		t.Fatal("Expected TextContent")
	}

	// Verify sessions are listed
	if !strings.Contains(textContent.Text, "Active sessions:") {
		t.Error("Expected 'Active sessions:' in output")
	}
	if !strings.Contains(textContent.Text, "session-1") {
		t.Error("Expected 'session-1' in output")
	}
	if !strings.Contains(textContent.Text, "session-2") {
		t.Error("Expected 'session-2' in output")
	}
}

func setupTestServer(t *testing.T) (*Server, *session.Manager) {
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
	t.Cleanup(func() { db.Close() })

	server, err := NewServer(cfg, beaconMgr, sessionMgr, db)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}
	return server, sessionMgr
}

func executeAndCheckTask(t *testing.T, s *Server, sess *session.Session, handlerFunc func(context.Context, *protocol.CallToolRequest) (*protocol.CallToolResult, error), req *protocol.CallToolRequest, expectedOutput string) {
	t.Helper()

	// Execute in a goroutine to simulate async task completion
	done := make(chan bool, 1)
	go func() {
		// Wait a bit for the handler to create the task
		time.Sleep(100 * time.Millisecond)

		// Simulate task completion
		tasks := sess.GetPendingTasks()
		if len(tasks) > 0 {
			task := tasks[0]
			// Add result to session
			sess.AddTaskResult(serverprotocol.TaskResult{
				TaskID:   task.ID,
				Output:   expectedOutput,
				ExitCode: 0,
				Time:     time.Now(),
			})
		}
		done <- true
	}()

	// Call handler with short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	result, err := handlerFunc(ctx, req)

	// Wait for task completion simulation
	select {
	case <-done:
		// Task completion simulated
	case <-time.After(3 * time.Second):
		t.Fatal("Timeout waiting for task completion simulation")
	}

	if err != nil {
		t.Fatalf("handler function failed: %v", err)
	}
	if result == nil {
		t.Fatal("Expected result, got nil")
	}

	// Verify the result returned by the handler
	// This part of the test might need to be adjusted based on what the handler actually returns.
}

func TestHandleExecuteCommand(t *testing.T) {
	server, sessionMgr := setupTestServer(t)
	sess := session.NewSession("test-session", nil)
	sessionMgr.AddSession(sess)

	t.Run("Valid shell command", func(t *testing.T) {
		req := &protocol.CallToolRequest{
			Arguments: map[string]interface{}{
				"session_id": "test-session",
				"command":    "echo hello",
			},
		}
		executeAndCheckTask(t, server, sess, server.handleExecuteCommand, req, "hello")
	})

	// ... (other test cases for handleExecuteCommand)
}

func TestHandleSystemInfo(t *testing.T) {
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

	// Add test session
	sess := session.NewSession("test-session", nil)
	sess.UpdateInformation(map[string]interface{}{
		"hostname": "testhost",
		"username": "testuser",
		"os":       "linux",
		"arch":     "amd64",
		"ip":       "192.168.1.100",
	})
	sessionMgr.AddSession(sess)

	// Test valid request
	ctx := context.Background()
	req := &protocol.CallToolRequest{
		Arguments: map[string]interface{}{
			"session_id": "test-session",
		},
	}

	result, err := server.handleSystemInfo(ctx, req)
	if err != nil {
		t.Fatalf("handleSystemInfo failed: %v", err)
	}

	if result == nil {
		t.Fatal("Expected result, got nil")
	}

	// Check content
	textContent, ok := result.Content[0].(*protocol.TextContent)
	if !ok {
		t.Fatal("Expected TextContent")
	}

	// Verify it's asking for system info task
	if !strings.Contains(textContent.Text, "task") {
		t.Error("Expected task creation message")
	}

	// Test missing session_id
	req2 := &protocol.CallToolRequest{
		Arguments: map[string]interface{}{},
	}

	_, err = server.handleSystemInfo(ctx, req2)
	if err == nil {
		t.Error("Expected error for missing session_id")
	}

	// Test invalid session
	req3 := &protocol.CallToolRequest{
		Arguments: map[string]interface{}{
			"session_id": "invalid-session",
		},
	}

	_, err = server.handleSystemInfo(ctx, req3)
	if err == nil {
		t.Error("Expected error for invalid session")
	}
}

func TestHandleUploadFile(t *testing.T) {
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

	// Add test session
	sess := session.NewSession("test-session", nil)
	sessionMgr.AddSession(sess)

	tests := []struct {
		name      string
		args      map[string]interface{}
		wantError bool
	}{
		{
			name: "Valid upload",
			args: map[string]interface{}{
				"session_id":  "test-session",
				"remote_path": "/tmp/test.txt",
				"content":     "SGVsbG8gV29ybGQ=", // "Hello World" in base64
			},
			wantError: false,
		},
		{
			name: "Missing session_id",
			args: map[string]interface{}{
				"remote_path": "/tmp/test.txt",
				"content":     "SGVsbG8gV29ybGQ=",
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			req := &protocol.CallToolRequest{
				Arguments: tt.args,
			}

			_, err := server.handleFileUpload(ctx, req)

			if tt.wantError {
				if err == nil {
					t.Error("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
			}
		})
	}
}

func TestHandleProcessList(t *testing.T) {
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

	// Add test session
	sess := session.NewSession("test-session", nil)
	sessionMgr.AddSession(sess)

	// Test request
	ctx := context.Background()
	req := &protocol.CallToolRequest{
		Arguments: map[string]interface{}{
			"session_id": "test-session",
		},
	}

	result, err := server.handleProcessList(ctx, req)
	if err != nil {
		t.Fatalf("handleProcessList failed: %v", err)
	}

	if result == nil {
		t.Fatal("Expected result, got nil")
	}

	// Check content
	textContent, ok := result.Content[0].(*protocol.TextContent)
	if !ok {
		t.Fatal("Expected TextContent")
	}

	// Verify it created a task
	if !strings.Contains(textContent.Text, "task") {
		t.Error("Expected task creation message")
	}

	// Test missing session_id
	req2 := &protocol.CallToolRequest{
		Arguments: map[string]interface{}{},
	}

	_, err = server.handleProcessList(ctx, req2)
	if err == nil {
		t.Error("Expected error for missing session_id")
	}
}

func TestHandleNetworkConnections(t *testing.T) {
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

	// Add test session
	sess := session.NewSession("test-session", nil)
	sessionMgr.AddSession(sess)

	tests := []struct {
		name      string
		args      map[string]interface{}
		wantError bool
	}{
		{
			name: "Valid request",
			args: map[string]interface{}{
				"session_id": "test-session",
			},
			wantError: false,
		},
		{
			name:      "Missing session_id",
			args:      map[string]interface{}{},
			wantError: true,
		},
		{
			name: "Invalid session",
			args: map[string]interface{}{
				"session_id": "invalid-session",
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			req := &protocol.CallToolRequest{
				Arguments: tt.args,
			}

			_, err := server.handleNetworkConnections(ctx, req)

			if tt.wantError {
				if err == nil {
					t.Error("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
			}
		})
	}
}

func TestHandlePortForward(t *testing.T) {
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

	// Add test session
	sess := session.NewSession("test-session", nil)
	sessionMgr.AddSession(sess)

	tests := []struct {
		name      string
		args      map[string]interface{}
		wantError bool
	}{
		{
			name: "Valid port forward",
			args: map[string]interface{}{
				"session_id":  "test-session",
				"local_port":  8080.0,
				"remote_host": "localhost",
				"remote_port": 80.0,
			},
			wantError: false,
		},
		{
			name: "Missing local_port",
			args: map[string]interface{}{
				"session_id":  "test-session",
				"remote_host": "localhost",
				"remote_port": 80.0,
			},
			wantError: true,
		},
		{
			name: "Missing session_id",
			args: map[string]interface{}{
				"local_port":  8080.0,
				"remote_host": "localhost",
				"remote_port": 80.0,
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			req := &protocol.CallToolRequest{
				Arguments: tt.args,
			}

			_, err := server.handlePortForward(ctx, req)

			if tt.wantError {
				if err == nil {
					t.Error("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
			}
		})
	}
}

func TestHandleDownloadFile(t *testing.T) {
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

	// Add test session
	sess := session.NewSession("test-session", nil)
	sessionMgr.AddSession(sess)

	tests := []struct {
		name      string
		args      map[string]interface{}
		wantError bool
	}{
		{
			name: "Valid download",
			args: map[string]interface{}{
				"session_id":  "test-session",
				"remote_path": "/etc/passwd",
			},
			wantError: false,
		},
		{
			name: "Missing session_id",
			args: map[string]interface{}{
				"remote_path": "/etc/passwd",
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			req := &protocol.CallToolRequest{
				Arguments: tt.args,
			}

			_, err := server.handleFileDownload(ctx, req)

			if tt.wantError {
				if err == nil {
					t.Error("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
			}
		})
	}
}
