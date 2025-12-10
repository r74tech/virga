package database

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupExtendedTestDB(t *testing.T) *Database {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "virga-extended-test-*")
	require.NoError(t, err)

	// Create database
	dbPath := filepath.Join(tmpDir, "test.db")
	db, err := Initialize(dbPath)
	require.NoError(t, err)

	// Clean up on test completion
	t.Cleanup(func() {
		db.Close()
		os.RemoveAll(tmpDir)
	})

	return db
}

func TestInitializeExtendedSchema(t *testing.T) {
	db := setupExtendedTestDB(t)

	// Check if all new tables exist
	tables := []string{
		"llama_interactions",
		"memdb_queries",
		"mcp_interactions",
		"autonomous_tasks",
		"beacon_configs",
		"task_metadata",
	}

	for _, table := range tables {
		var name string
		err := db.QueryRow(
			"SELECT name FROM sqlite_master WHERE type='table' AND name=?",
			table,
		).Scan(&name)
		assert.NoError(t, err, "Table %s should exist", table)
		assert.Equal(t, table, name)
	}
}

func TestLlamaInteractions(t *testing.T) {
	db := setupExtendedTestDB(t)

	// Test logging a Llama interaction
	sessionID := "test-session-123"
	taskID := "task-456"
	prompt := "Find all large files in /var/log"
	model := "llama"
	maxIterations := 5
	temperature := 0.3

	id, err := db.LogLlamaInteraction(sessionID, taskID, prompt, model, maxIterations, temperature)
	require.NoError(t, err)
	assert.Greater(t, id, int64(0))

	// Test updating the interaction
	response := "Found 3 large files:\n1. /var/log/system.log (1.2GB)\n2. /var/log/messages (800MB)\n3. /var/log/kern.log (500MB)"
	err = db.UpdateLlamaInteraction(id, response, "completed")
	require.NoError(t, err)

	// Test retrieving interactions
	interactions, err := db.GetLlamaInteractions(sessionID)
	require.NoError(t, err)
	assert.Len(t, interactions, 1)

	interaction := interactions[0]
	assert.Equal(t, taskID, interaction["task_id"])
	assert.Equal(t, prompt, interaction["prompt"])
	assert.Equal(t, response, interaction["response"])
	assert.Equal(t, model, interaction["model"])
	assert.Equal(t, maxIterations, interaction["max_iterations"])
	assert.Equal(t, temperature, interaction["temperature"])
	assert.Equal(t, "completed", interaction["status"])
	assert.NotNil(t, interaction["created_at"])
	assert.NotNil(t, interaction["completed_at"])
}

func TestMemDBQueries(t *testing.T) {
	db := setupExtendedTestDB(t)

	// Test logging a MemDB query
	sessionID := "test-session-789"
	query := "SELECT * FROM agents WHERE os='linux'"
	result := `agent-001|linux|server1
agent-002|linux|server2
agent-003|linux|server3`
	resultCount := 3
	executionTimeMs := 45

	err := db.LogMemDBQuery(sessionID, query, result, resultCount, executionTimeMs)
	require.NoError(t, err)

	// Test retrieving queries
	queries, err := db.GetMemDBQueries(sessionID)
	require.NoError(t, err)
	assert.Len(t, queries, 1)

	q := queries[0]
	assert.Equal(t, query, q["query"])
	assert.Equal(t, result, q["result"])
	assert.Equal(t, resultCount, q["result_count"])
	assert.Equal(t, executionTimeMs, q["execution_time_ms"])
	assert.NotNil(t, q["created_at"])
}

func TestMCPInteractions(t *testing.T) {
	db := setupExtendedTestDB(t)

	// Test logging an MCP interaction
	sessionID := "test-session-mcp"
	toolName := "session_command"
	action := "execute"
	parameters := `{"command": "ls -la", "session_id": "sess-123"}`

	id, err := db.LogMCPInteraction(sessionID, toolName, action, parameters)
	require.NoError(t, err)
	assert.Greater(t, id, int64(0))

	// Test updating the interaction
	result := `total 24
drwxr-xr-x  5 user  staff   160 Dec 26 10:00 .
drwxr-xr-x  3 user  staff    96 Dec 26 09:00 ..
-rw-r--r--  1 user  staff  1024 Dec 26 10:00 file1.txt`
	err = db.UpdateMCPInteraction(id, result, "success", "")
	require.NoError(t, err)

	// Test retrieving interactions
	interactions, err := db.GetMCPInteractions(sessionID)
	require.NoError(t, err)
	assert.Len(t, interactions, 1)

	interaction := interactions[0]
	assert.Equal(t, toolName, interaction["tool_name"])
	assert.Equal(t, action, interaction["action"])
	assert.Equal(t, parameters, interaction["parameters"])
	assert.Equal(t, result, interaction["result"])
	assert.Equal(t, "success", interaction["status"])
	assert.NotNil(t, interaction["created_at"])
	assert.NotNil(t, interaction["completed_at"])
}

func TestAutonomousTasks(t *testing.T) {
	db := setupExtendedTestDB(t)

	// Test logging an autonomous task
	sessionID := "test-session-auto"
	taskName := "system_reconnaissance"
	taskType := "initial_recon"
	configuration := `{"scan_ports": true, "enumerate_users": true, "check_processes": true}`

	id, err := db.LogAutonomousTask(sessionID, taskName, taskType, configuration)
	require.NoError(t, err)
	assert.Greater(t, id, int64(0))

	// Test updating the task
	result := `Reconnaissance completed:
- Open ports: 22, 80, 443
- Users: root, admin, user1
- Running processes: 47`
	err = db.UpdateAutonomousTask(id, "completed", result, "")
	require.NoError(t, err)

	// Test retrieving tasks
	tasks, err := db.GetAutonomousTasks(sessionID)
	require.NoError(t, err)
	assert.Len(t, tasks, 1)

	task := tasks[0]
	assert.Equal(t, taskName, task["task_name"])
	assert.Equal(t, taskType, task["task_type"])
	assert.Equal(t, configuration, task["configuration"])
	assert.Equal(t, "completed", task["status"])
	assert.Equal(t, result, task["result"])
	assert.NotNil(t, task["start_time"])
	assert.NotNil(t, task["end_time"])
}

func TestBeaconConfigs(t *testing.T) {
	db := setupExtendedTestDB(t)

	// Test saving a beacon config
	beaconID := "beacon-test-123"
	agentID := "agent-test-456"
	configJSON := `{"protocol": "https", "callback_interval": 30, "jitter": 20}`
	sleepTime := 30
	jitter := 20
	killDate := time.Now().AddDate(0, 1, 0) // 1 month from now

	err := db.SaveBeaconConfig(beaconID, agentID, configJSON, sleepTime, jitter, &killDate)
	require.NoError(t, err)

	// Test updating the beacon config
	newConfigJSON := `{"protocol": "https", "callback_interval": 60, "jitter": 30}`
	newSleepTime := 60
	newJitter := 30
	err = db.SaveBeaconConfig(beaconID, agentID, newConfigJSON, newSleepTime, newJitter, &killDate)
	require.NoError(t, err)

	// Verify the update
	var storedConfig string
	var storedSleepTime, storedJitter int
	err = db.QueryRow(
		"SELECT config_json, sleep_time, jitter FROM beacon_configs WHERE beacon_id = ?",
		beaconID,
	).Scan(&storedConfig, &storedSleepTime, &storedJitter)
	require.NoError(t, err)
	assert.Equal(t, newConfigJSON, storedConfig)
	assert.Equal(t, newSleepTime, storedSleepTime)
	assert.Equal(t, newJitter, storedJitter)
}

func TestTaskMetadata(t *testing.T) {
	db := setupExtendedTestDB(t)

	// First create a command to get a valid command ID
	sessionID := "test-session-meta"
	commandID, err := db.LogCommand(sessionID, "test", "test command")
	require.NoError(t, err)

	// Test saving task metadata
	taskID := "task-meta-789"
	taskType := "llama_interactive"
	priority := 8
	metadata := map[string]interface{}{
		"prompt":         "Find security vulnerabilities",
		"max_iterations": 10,
		"temperature":    0.2,
		"model":          "llama-security",
		"tags":           []string{"security", "audit", "vulnerability"},
	}

	err = db.SaveTaskMetadata(commandID, taskID, taskType, priority, metadata)
	require.NoError(t, err)

	// Test retrieving task metadata
	result, err := db.GetTaskMetadata(taskID)
	require.NoError(t, err)

	assert.Equal(t, commandID, result["command_id"])
	assert.Equal(t, taskID, result["task_id"])
	assert.Equal(t, taskType, result["task_type"])
	assert.Equal(t, priority, result["priority"])
	assert.NotNil(t, result["metadata"])
	assert.NotNil(t, result["created_at"])

	// Verify metadata content
	meta, ok := result["metadata"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "Find security vulnerabilities", meta["prompt"])
	assert.Equal(t, float64(10), meta["max_iterations"]) // JSON unmarshals numbers as float64
	assert.Equal(t, 0.2, meta["temperature"])
}

func TestGetExtendedStats(t *testing.T) {
	db := setupExtendedTestDB(t)

	// Add some test data
	sessionID := "test-session-stats"

	// Add Llama interaction
	llamaID, err := db.LogLlamaInteraction(sessionID, "task-1", "test prompt", "llama", 5, 0.3)
	require.NoError(t, err)
	err = db.UpdateLlamaInteraction(llamaID, "test response", "completed")
	require.NoError(t, err)

	// Add MemDB query
	err = db.LogMemDBQuery(sessionID, "SELECT * FROM test", "result", 10, 50)
	require.NoError(t, err)

	// Add MCP interaction
	mcpID, err := db.LogMCPInteraction(sessionID, "test_tool", "execute", "{}")
	require.NoError(t, err)
	err = db.UpdateMCPInteraction(mcpID, "success", "completed", "")
	require.NoError(t, err)

	// Add autonomous task
	autoID, err := db.LogAutonomousTask(sessionID, "test_task", "test", "{}")
	require.NoError(t, err)
	err = db.UpdateAutonomousTask(autoID, "completed", "done", "")
	require.NoError(t, err)

	// Get extended stats
	stats := db.GetExtendedStats()

	// Verify counts
	assert.GreaterOrEqual(t, stats["total_llama_interactions"], 1)
	assert.GreaterOrEqual(t, stats["total_memdb_queries"], 1)
	assert.GreaterOrEqual(t, stats["total_mcp_interactions"], 1)
	assert.GreaterOrEqual(t, stats["total_autonomous_tasks"], 1)
}

func TestComplexScenario(t *testing.T) {
	db := setupExtendedTestDB(t)

	// Simulate a complex scenario with multiple operations
	agentID := "agent-complex-001"
	sessionID := "session-complex-001"

	// 1. Save agent
	err := db.SaveAgent(agentID, "192.168.1.100")
	require.NoError(t, err)

	// 2. Create session
	err = db.CreateSession(sessionID, agentID)
	require.NoError(t, err)

	// 3. Update agent info
	err = db.UpdateAgentInfo(agentID, "workstation-1", "user1", "windows")
	require.NoError(t, err)

	// 4. Execute regular command
	cmdID1, err := db.LogCommand(sessionID, "shell", "whoami")
	require.NoError(t, err)
	err = db.LogCommandResult(cmdID1, "user1\\workstation-1", 0, "")
	require.NoError(t, err)

	// 5. Execute Llama command
	taskID := "llama-task-001"
	llamaID, err := db.LogLlamaInteraction(sessionID, taskID, "Find all running services", "llama", 5, 0.3)
	require.NoError(t, err)

	cmdID2, err := db.LogCommand(sessionID, "llama_interactive", `{"prompt":"Find all running services"}`)
	require.NoError(t, err)

	// Simulate Llama response
	llamaResponse := "Found 47 running services including: sshd, httpd, mysql..."
	err = db.UpdateLlamaInteraction(llamaID, llamaResponse, "completed")
	require.NoError(t, err)
	err = db.LogCommandResult(cmdID2, llamaResponse, 0, "")
	require.NoError(t, err)

	// 6. Execute MemDB query
	memdbQuery := "SELECT * FROM services WHERE status='running'"
	err = db.LogMemDBQuery(sessionID, memdbQuery, "sshd|running\nhttpd|running", 2, 15)
	require.NoError(t, err)

	// 7. Log file operation
	err = db.LogFile(sessionID, "config.json", "/etc/app/config.json", "abc123def456", 2048)
	require.NoError(t, err)

	// 8. Close session
	err = db.CloseSession(sessionID)
	require.NoError(t, err)

	// Verify all data was saved correctly
	agents, err := db.GetAgents()
	require.NoError(t, err)
	assert.Len(t, agents, 1)
	assert.Equal(t, "workstation-1", agents[0]["hostname"])

	history, err := db.GetAgentHistory(agentID)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(history), 2)

	llamaInteractions, err := db.GetLlamaInteractions(sessionID)
	require.NoError(t, err)
	assert.Len(t, llamaInteractions, 1)

	memdbQueries, err := db.GetMemDBQueries(sessionID)
	require.NoError(t, err)
	assert.Len(t, memdbQueries, 1)

	files, err := db.GetFileOperations(sessionID)
	require.NoError(t, err)
	assert.Len(t, files, 1)

	// Check extended stats
	stats := db.GetExtendedStats()
	assert.GreaterOrEqual(t, stats["total_agents"], 1)
	assert.GreaterOrEqual(t, stats["total_sessions"], 1)
	assert.GreaterOrEqual(t, stats["total_commands"], 2)
	assert.GreaterOrEqual(t, stats["total_files"], 1)
	assert.GreaterOrEqual(t, stats["total_llama_interactions"], 1)
	assert.GreaterOrEqual(t, stats["total_memdb_queries"], 1)
}

// TestConcurrentWrites tests that the database handles concurrent writes correctly
func TestConcurrentWrites(t *testing.T) {
	db := setupExtendedTestDB(t)

	// Number of concurrent operations
	numGoroutines := 10
	done := make(chan bool, numGoroutines)

	// Launch concurrent operations
	for i := 0; i < numGoroutines; i++ {
		go func(index int) {
			sessionID := fmt.Sprintf("concurrent-session-%d", index)

			// Log Llama interaction
			_, err := db.LogLlamaInteraction(sessionID, fmt.Sprintf("task-%d", index),
				"test prompt", "llama", 5, 0.3)
			assert.NoError(t, err)

			// Log MemDB query
			err = db.LogMemDBQuery(sessionID, "SELECT *", "result", 1, 10)
			assert.NoError(t, err)

			// Log MCP interaction
			_, err = db.LogMCPInteraction(sessionID, "tool", "action", "{}")
			assert.NoError(t, err)

			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// Verify all writes succeeded
	stats := db.GetExtendedStats()
	assert.Equal(t, numGoroutines, stats["total_llama_interactions"])
	assert.Equal(t, numGoroutines, stats["total_memdb_queries"])
	assert.Equal(t, numGoroutines, stats["total_mcp_interactions"])
}

func TestJSONMarshaling(t *testing.T) {
	db := setupExtendedTestDB(t)

	// Test complex JSON metadata
	sessionID := "test-json-session"
	cmdID, err := db.LogCommand(sessionID, "test", "test")
	require.NoError(t, err)

	complexMetadata := map[string]interface{}{
		"nested": map[string]interface{}{
			"level1": map[string]interface{}{
				"level2": "deep value",
				"array":  []int{1, 2, 3},
			},
		},
		"boolean": true,
		"number":  42.5,
		"null":    nil,
	}

	err = db.SaveTaskMetadata(cmdID, "json-task", "test", 5, complexMetadata)
	require.NoError(t, err)

	// Retrieve and verify
	result, err := db.GetTaskMetadata("json-task")
	require.NoError(t, err)

	metadata, ok := result["metadata"].(map[string]interface{})
	require.True(t, ok)

	// Verify nested structure
	nested, ok := metadata["nested"].(map[string]interface{})
	require.True(t, ok)

	level1, ok := nested["level1"].(map[string]interface{})
	require.True(t, ok)

	assert.Equal(t, "deep value", level1["level2"])
	assert.Equal(t, true, metadata["boolean"])
	assert.Equal(t, 42.5, metadata["number"])
}

func TestErrorCases(t *testing.T) {
	db := setupExtendedTestDB(t)

	// Test updating non-existent Llama interaction
	err := db.UpdateLlamaInteraction(99999, "response", "completed")
	assert.NoError(t, err) // SQLite doesn't error on UPDATE with no matches

	// Test getting task metadata for non-existent task
	_, err = db.GetTaskMetadata("non-existent-task")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "task metadata not found")

	// Test saving task metadata with invalid command ID
	err = db.SaveTaskMetadata(99999, "task-invalid", "test", 5, map[string]interface{}{})
	// This should succeed as SQLite doesn't enforce foreign key by default
	// but in production, this would fail with foreign key constraint
	assert.NoError(t, err)
}
