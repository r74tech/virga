package server

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/r74tech/virga/internal/server/config"
	"github.com/r74tech/virga/internal/server/database"
	"github.com/r74tech/virga/internal/server/listener"
	"github.com/r74tech/virga/internal/server/protocol"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockAgentConnection implements protocol.AgentConnection for testing
type MockAgentConnection struct {
	remoteAddr   string
	responses    chan []byte
	closed       bool
	lastActivity time.Time
}

func (c *MockAgentConnection) RemoteAddr() string {
	return c.remoteAddr
}

func (c *MockAgentConnection) Send(data []byte) error {
	if c.closed {
		return fmt.Errorf("connection closed")
	}
	c.lastActivity = time.Now()
	select {
	case c.responses <- data:
		return nil
	default:
		return fmt.Errorf("response channel full")
	}
}

func (c *MockAgentConnection) Receive() ([]byte, error) {
	if c.closed {
		return nil, fmt.Errorf("connection closed")
	}
	select {
	case data := <-c.responses:
		c.lastActivity = time.Now()
		return data, nil
	case <-time.After(100 * time.Millisecond):
		return nil, fmt.Errorf("receive timeout")
	}
}

func (c *MockAgentConnection) LastActivity() time.Time {
	return c.lastActivity
}

func (c *MockAgentConnection) Close() error {
	c.closed = true
	close(c.responses)
	return nil
}

func setupIntegrationTest(t *testing.T) (*Server, *database.Database, string) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "virga-integration-test-*")
	require.NoError(t, err)

	// Create database
	dbPath := filepath.Join(tmpDir, "test.db")
	db, err := database.Initialize(dbPath)
	require.NoError(t, err)

	// Create config
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host:           "127.0.0.1",
			AdminPort:      0, // Use random port
			SessionTimeout: 30 * time.Second,
		},
		Database: config.DatabaseConfig{
			Path: dbPath,
		},
		Listeners: []config.ListenerConfig{},
	}

	// Create server
	server := NewServer(cfg, db)

	// Clean up on test completion
	t.Cleanup(func() {
		// server.Stop() // Stop method doesn't exist, just cleanup
		db.Close()
		os.RemoveAll(tmpDir)
	})

	return server, db, tmpDir
}

func TestIntegrationDatabaseWrites(t *testing.T) {
	server, db, _ := setupIntegrationTest(t)

	// Start server
	go server.Start()
	time.Sleep(100 * time.Millisecond) // Give server time to start

	// Test 1: New agent connection
	agentID := "test-agent-001"
	conn := &MockAgentConnection{
		remoteAddr:   "192.168.1.100:12345",
		responses:    make(chan []byte, 10),
		lastActivity: time.Now(),
	}

	// Simulate new agent connection
	server.handleNewAgent(agentID, conn)

	// Verify agent was saved to database
	agents, err := db.GetAgents()
	require.NoError(t, err)
	assert.Len(t, agents, 1)
	assert.Equal(t, agentID, agents[0]["id"])
	assert.Equal(t, "192.168.1.100:12345", agents[0]["ip_address"])

	// Verify session was created in database
	var sessionCount int
	err = db.QueryRow("SELECT COUNT(*) FROM sessions WHERE agent_id = ?", agentID).Scan(&sessionCount)
	require.NoError(t, err)
	assert.Equal(t, 1, sessionCount)

	// Test 2: Add task through API
	sess, exists := server.GetSession(agentID)
	require.True(t, exists)

	// Create task request
	taskRequest := struct {
		Type    string                 `json:"type"`
		Payload map[string]interface{} `json:"payload"`
	}{
		Type: "shell",
		Payload: map[string]interface{}{
			"command": "whoami",
		},
	}

	// Add task directly (simulating API call)
	taskID := generateUUID()
	task := protocol.Task{
		ID:        taskID,
		TaskID:    taskID,
		Type:      protocol.TaskType(taskRequest.Type),
		SessionID: agentID,
		Payload:   taskRequest.Payload,
		Arguments: taskRequest.Payload,
		CreatedAt: time.Now(),
		SentTime:  time.Now(),
		Status:    protocol.TaskStatusPending,
	}

	sess.AddTask(task)

	// Log task to database (simulating what handleAddTask does)
	payloadJSON, _ := json.Marshal(taskRequest.Payload)
	commandID, err := db.LogCommand(agentID, string(task.Type), string(payloadJSON))
	require.NoError(t, err)
	sess.SetTaskCommandMapping(task.TaskID, commandID)

	// Save task metadata
	metadata := map[string]interface{}{
		"created_at": task.CreatedAt,
		"status":     task.Status,
	}
	err = db.SaveTaskMetadata(commandID, task.TaskID, string(task.Type), 5, metadata)
	require.NoError(t, err)

	// Verify command was logged
	var cmdCount int
	err = db.QueryRow("SELECT COUNT(*) FROM commands WHERE session_id = ?", agentID).Scan(&cmdCount)
	require.NoError(t, err)
	assert.Equal(t, 1, cmdCount)

	// Verify task metadata was saved
	taskMeta, err := db.GetTaskMetadata(taskID)
	require.NoError(t, err)
	assert.Equal(t, taskID, taskMeta["task_id"])
	assert.Equal(t, "shell", taskMeta["task_type"])

	// Test 3: Simulate task result
	taskResult := protocol.TaskResult{
		TaskID:   taskID,
		Output:   "testuser",
		ExitCode: 0,
		Error:    "",
		Time:     time.Now(),
	}

	sess.AddTaskResult(taskResult)

	// Log command result (simulating what HTTP listener does)
	if cmdID, exists := sess.GetTaskCommandMapping(taskID); exists {
		err = db.LogCommandResult(cmdID, taskResult.Output, taskResult.ExitCode, taskResult.Error)
		require.NoError(t, err)
		sess.ClearTaskCommandMapping(taskID)
	}

	// Verify command result was logged
	history, err := db.GetCommandHistory(agentID)
	require.NoError(t, err)
	assert.Len(t, history, 1)
	assert.Equal(t, "shell", history[0]["command"])
	assert.Equal(t, "testuser", history[0]["output"])
	assert.Equal(t, int64(0), history[0]["exit_code"])

	// Test 4: Update agent information
	err = db.UpdateAgentInfo(agentID, "workstation-1", "testuser", "linux")
	require.NoError(t, err)

	// Verify agent info was updated
	agents, err = db.GetAgents()
	require.NoError(t, err)
	assert.Len(t, agents, 1)
	assert.Equal(t, "workstation-1", agents[0]["hostname"])
	assert.Equal(t, "testuser", agents[0]["username"])
	assert.Equal(t, "linux", agents[0]["os"])

	// Test 5: Extended functionality - Llama interaction
	llamaTaskID := "llama-task-001"
	llamaPrompt := "Find all large files in /var/log"
	llamaID, err := db.LogLlamaInteraction(agentID, llamaTaskID, llamaPrompt, "llama", 5, 0.3)
	require.NoError(t, err)

	// Simulate Llama response
	llamaResponse := "Found 3 large files:\n1. /var/log/system.log (1.2GB)\n2. /var/log/messages (800MB)"
	err = db.UpdateLlamaInteraction(llamaID, llamaResponse, "completed")
	require.NoError(t, err)

	// Verify Llama interaction was saved
	llamaInteractions, err := db.GetLlamaInteractions(agentID)
	require.NoError(t, err)
	assert.Len(t, llamaInteractions, 1)
	assert.Equal(t, llamaPrompt, llamaInteractions[0]["prompt"])
	assert.Equal(t, llamaResponse, llamaInteractions[0]["response"])

	// Test 6: MemDB query
	memdbQuery := "SELECT * FROM agents WHERE status='active'"
	err = db.LogMemDBQuery(agentID, memdbQuery, "agent-001|active\nagent-002|active", 2, 25)
	require.NoError(t, err)

	// Verify MemDB query was saved
	memdbQueries, err := db.GetMemDBQueries(agentID)
	require.NoError(t, err)
	assert.Len(t, memdbQueries, 1)
	assert.Equal(t, memdbQuery, memdbQueries[0]["query"])

	// Test 7: File operation
	err = db.LogFile(agentID, "config.txt", "/etc/app/config.txt", "abc123", 1024)
	require.NoError(t, err)

	// Verify file operation was logged
	files, err := db.GetFileOperations(agentID)
	require.NoError(t, err)
	assert.Len(t, files, 1)
	assert.Equal(t, "config.txt", files[0]["filename"])

	// Test 8: Get extended stats
	stats := db.GetExtendedStats()
	assert.GreaterOrEqual(t, stats["total_agents"], 1)
	assert.GreaterOrEqual(t, stats["total_sessions"], 1)
	assert.GreaterOrEqual(t, stats["total_commands"], 1)
	assert.GreaterOrEqual(t, stats["total_llama_interactions"], 1)
	assert.GreaterOrEqual(t, stats["total_memdb_queries"], 1)
	assert.GreaterOrEqual(t, stats["total_files"], 1)
}

func TestIntegrationSessionTimeout(t *testing.T) {
	server, db, _ := setupIntegrationTest(t)

	// Set short timeout for testing
	server.config.Server.SessionTimeout = 100 * time.Millisecond

	// Start server but with a modified cleanup interval for testing
	// We need to manually trigger the cleanup since the default is 30s
	go server.Start()
	time.Sleep(50 * time.Millisecond)

	// Create agent connection
	agentID := "timeout-agent-001"
	conn := &MockAgentConnection{
		remoteAddr:   "192.168.1.200:12345",
		responses:    make(chan []byte, 10),
		lastActivity: time.Now(),
	}

	// Add agent
	server.handleNewAgent(agentID, conn)

	// Verify session exists
	sess, exists := server.GetSession(agentID)
	require.True(t, exists)
	require.NotNil(t, sess)

	// Wait for timeout
	time.Sleep(200 * time.Millisecond)

	// Manually trigger session cleanup instead of waiting 30s
	server.sessionsMu.Lock()
	now := time.Now()
	for id, sess := range server.sessions {
		if now.Sub(sess.LastSeen()) > server.config.Server.SessionTimeout {
			// Handle timed out sessions
			listener.SetSessionHandler(id, nil)
			delete(server.sessions, id)

			// Close session in database
			if err := server.db.CloseSession(id); err != nil {
				t.Logf("Failed to close session in database: %v", err)
			}
		}
	}
	server.sessionsMu.Unlock()

	// Verify session was removed
	_, exists = server.GetSession(agentID)
	assert.False(t, exists)

	// Verify session was closed in database
	var endTime *time.Time
	err := db.QueryRow("SELECT end_time FROM sessions WHERE id = ?", agentID).Scan(&endTime)
	require.NoError(t, err)
	assert.NotNil(t, endTime, "Session should have end_time set")
}

func TestIntegrationMultipleAgents(t *testing.T) {
	server, db, _ := setupIntegrationTest(t)

	// Start server
	go server.Start()
	time.Sleep(100 * time.Millisecond)

	// Create multiple agents
	agentCount := 5
	for i := 0; i < agentCount; i++ {
		agentID := fmt.Sprintf("multi-agent-%03d", i)
		conn := &MockAgentConnection{
			remoteAddr:   fmt.Sprintf("192.168.1.%d:12345", 100+i),
			responses:    make(chan []byte, 10),
			lastActivity: time.Now(),
		}

		server.handleNewAgent(agentID, conn)

		// Update agent info
		err := db.UpdateAgentInfo(agentID,
			fmt.Sprintf("host-%d", i),
			fmt.Sprintf("user-%d", i),
			"linux")
		require.NoError(t, err)

		// Add some commands
		for j := 0; j < 3; j++ {
			cmdID, err := db.LogCommand(agentID, "shell", fmt.Sprintf("command-%d", j))
			require.NoError(t, err)

			err = db.LogCommandResult(cmdID, fmt.Sprintf("output-%d", j), 0, "")
			require.NoError(t, err)
		}
	}

	// Verify all agents were saved
	agents, err := db.GetAgents()
	require.NoError(t, err)
	assert.Len(t, agents, agentCount)

	// Verify sessions were created
	var sessionCount int
	err = db.QueryRow("SELECT COUNT(*) FROM sessions").Scan(&sessionCount)
	require.NoError(t, err)
	assert.Equal(t, agentCount, sessionCount)

	// Verify commands were logged
	var commandCount int
	err = db.QueryRow("SELECT COUNT(*) FROM commands").Scan(&commandCount)
	require.NoError(t, err)
	assert.Equal(t, agentCount*3, commandCount)

	// Test stats
	stats := db.GetStats()
	assert.Equal(t, agentCount, stats["total_agents"])
	assert.Equal(t, agentCount, stats["total_sessions"])
	assert.Equal(t, agentCount*3, stats["total_commands"])
}

func TestIntegrationMCPHandlers(t *testing.T) {
	server, db, _ := setupIntegrationTest(t)

	// Start server
	go server.Start()
	time.Sleep(100 * time.Millisecond)

	// Create agent
	agentID := "mcp-agent-001"
	conn := &MockAgentConnection{
		remoteAddr:   "192.168.1.150:12345",
		responses:    make(chan []byte, 10),
		lastActivity: time.Now(),
	}

	server.handleNewAgent(agentID, conn)

	// Test MCP interaction logging
	toolName := "session_command"
	action := "execute"
	parameters := `{"session_id": "` + agentID + `", "command": "ls -la"}`

	mcpID, err := db.LogMCPInteraction(agentID, toolName, action, parameters)
	require.NoError(t, err)

	// Simulate MCP result
	result := "file1.txt\nfile2.txt\nfile3.txt"
	err = db.UpdateMCPInteraction(mcpID, result, "success", "")
	require.NoError(t, err)

	// Verify MCP interaction was saved
	interactions, err := db.GetMCPInteractions(agentID)
	require.NoError(t, err)
	assert.Len(t, interactions, 1)
	assert.Equal(t, toolName, interactions[0]["tool_name"])
	assert.Equal(t, "success", interactions[0]["status"])
}

// BenchmarkDatabaseWrites benchmarks database write operations
func BenchmarkDatabaseWrites(b *testing.B) {
	// Setup
	tmpDir, err := os.MkdirTemp("", "virga-bench-*")
	require.NoError(b, err)
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "bench.db")
	db, err := database.Initialize(dbPath)
	require.NoError(b, err)
	defer db.Close()

	b.ResetTimer()

	b.Run("LogCommand", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			sessionID := fmt.Sprintf("bench-session-%d", i%100)
			db.LogCommand(sessionID, "shell", "test command")
		}
	})

	b.Run("LogLlamaInteraction", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			sessionID := fmt.Sprintf("bench-session-%d", i%100)
			db.LogLlamaInteraction(sessionID, fmt.Sprintf("task-%d", i), "prompt", "llama", 5, 0.3)
		}
	})

	b.Run("LogMemDBQuery", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			sessionID := fmt.Sprintf("bench-session-%d", i%100)
			db.LogMemDBQuery(sessionID, "SELECT *", "result", 10, 50)
		}
	})
}
