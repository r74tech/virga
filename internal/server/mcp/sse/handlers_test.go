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
	serverprotocol "github.com/r74tech/virga/internal/server/protocol"
	"github.com/r74tech/virga/internal/server/session"
)

func TestHandleSessionList(t *testing.T) {
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
	sess1 := session.NewSession("agent-1", nil)
	sess1.UpdateInformation(map[string]interface{}{
		"hostname": "host1",
		"username": "user1",
		"os":       "linux",
	})
	sessionMgr.AddSession(sess1)

	sess2 := session.NewSession("agent-2", nil)
	sess2.UpdateInformation(map[string]interface{}{
		"hostname": "host2",
		"username": "user2",
		"os":       "windows",
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
}

func setupTestServer(t *testing.T) (*Server, *session.Manager) {
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

	t.Run("Valid command", func(t *testing.T) {
		req := &protocol.CallToolRequest{
			Arguments: map[string]interface{}{
				"session_id": "test-session",
				"command":    "ls -la",
			},
		}
		executeAndCheckTask(t, server, sess, server.handleExecuteCommand, req, "total 0")
	})

	// ... (other test cases for handleExecuteCommand)
}

func TestHandleUploadFile(t *testing.T) {
	server, sessionMgr := setupTestServer(t)
	sess := session.NewSession("test-session", nil)
	sessionMgr.AddSession(sess)

	t.Run("Valid upload", func(t *testing.T) {
		req := &protocol.CallToolRequest{
			Arguments: map[string]interface{}{
				"session_id":  "test-session",
				"remote_path": "/tmp/test.txt",
				"content":     "SGVsbG8gV29ybGQ=", // "Hello World" in base64
			},
		}
		executeAndCheckTask(t, server, sess, server.handleFileUpload, req, "File uploaded successfully")
	})

	// ... (other test cases for handleFileUpload)
}

func TestHandleDownloadFile(t *testing.T) {
	server, sessionMgr := setupTestServer(t)
	sess := session.NewSession("test-session", nil)
	sessionMgr.AddSession(sess)

	t.Run("Valid download", func(t *testing.T) {
		req := &protocol.CallToolRequest{
			Arguments: map[string]interface{}{
				"session_id":  "test-session",
				"remote_path": "/etc/passwd",
			},
		}
		executeAndCheckTask(t, server, sess, server.handleFileDownload, req, "root:x:0:0:root:/root:/bin/bash")
	})

	// ... (other test cases for handleFileDownload)
}

func TestHandleSystemInfo(t *testing.T) {
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
	sess := session.NewSession("test-session", nil)
	sessionMgr.AddSession(sess)

	ctx := context.Background()
	req := &protocol.CallToolRequest{
		Arguments: map[string]interface{}{
			"session_id": "test-session",
		},
	}

	result, err := server.handleSystemInfo(ctx, req)
	if err != nil {
		t.Fatalf("handleGetSystemInfo failed: %v", err)
	}

	if result == nil {
		t.Fatal("Expected result, got nil")
	}

	// Test missing session_id
	req2 := &protocol.CallToolRequest{
		Arguments: map[string]interface{}{},
	}

	_, err = server.handleSystemInfo(ctx, req2)
	if err == nil {
		t.Error("Expected error for missing session_id")
	}
}

func TestHandleNetworkConnections(t *testing.T) {
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
	sess := session.NewSession("test-session", nil)
	sessionMgr.AddSession(sess)

	ctx := context.Background()
	req := &protocol.CallToolRequest{
		Arguments: map[string]interface{}{
			"session_id": "test-session",
		},
	}

	result, err := server.handleNetworkConnections(ctx, req)
	if err != nil {
		t.Fatalf("handleListNetworkConnections failed: %v", err)
	}

	if result == nil {
		t.Fatal("Expected result, got nil")
	}
}

func TestHandlePortForward(t *testing.T) {
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
			name: "Missing remote_host",
			args: map[string]interface{}{
				"session_id":  "test-session",
				"local_port":  8080.0,
				"remote_port": 80.0,
			},
			wantError: true,
		},
		{
			name: "Missing remote_port",
			args: map[string]interface{}{
				"session_id":  "test-session",
				"local_port":  8080.0,
				"remote_host": "localhost",
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

			result, err := server.handlePortForward(ctx, req)

			if tt.wantError {
				if err == nil {
					t.Error("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
				if result == nil {
					t.Error("Expected result, got nil")
				}
			}
		})
	}
}
