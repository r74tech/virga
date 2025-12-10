package database

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func setupTestDB(t *testing.T) (*Database, string) {
	// Create temporary directory for test database
	tempDir, err := os.MkdirTemp("", "VIRGA_test_")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	dbPath := filepath.Join(tempDir, "test.db")

	// Initialize database
	db, err := Initialize(dbPath)
	if err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("Failed to initialize database: %v", err)
	}

	return db, tempDir
}

func cleanupTestDB(db *Database, tempDir string) {
	if db != nil && db.db != nil {
		db.Close()
	}
	os.RemoveAll(tempDir)
}

func TestInitialize(t *testing.T) {
	db, tempDir := setupTestDB(t)
	defer cleanupTestDB(db, tempDir)

	// Verify database was created
	if db == nil {
		t.Fatal("Database instance is nil")
	}

	if db.db == nil {
		t.Fatal("Database connection is nil")
	}

	// Verify tables were created
	tables := []string{"agents", "sessions", "commands", "command_results", "files"}
	for _, table := range tables {
		var name string
		err := db.db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&name)
		if err != nil {
			t.Errorf("Table %s was not created: %v", table, err)
		}
	}
}

func TestSaveAgent(t *testing.T) {
	db, tempDir := setupTestDB(t)
	defer cleanupTestDB(db, tempDir)

	agentID := "test-agent-123"
	ipAddress := "192.168.1.100"

	// Test creating new agent
	err := db.SaveAgent(agentID, ipAddress)
	if err != nil {
		t.Fatalf("Failed to save agent: %v", err)
	}

	// Verify agent was saved
	var savedID, savedIP string
	err = db.db.QueryRow("SELECT id, ip_address FROM agents WHERE id = ?", agentID).Scan(&savedID, &savedIP)
	if err != nil {
		t.Fatalf("Failed to retrieve saved agent: %v", err)
	}

	if savedID != agentID {
		t.Errorf("Agent ID = %s, want %s", savedID, agentID)
	}
	if savedIP != ipAddress {
		t.Errorf("Agent IP = %s, want %s", savedIP, ipAddress)
	}

	// Test updating existing agent
	time.Sleep(10 * time.Millisecond) // Ensure time difference

	newIP := "192.168.1.101"
	err = db.SaveAgent(agentID, newIP)
	if err != nil {
		t.Fatalf("Failed to update agent: %v", err)
	}

	// Verify update
	err = db.db.QueryRow("SELECT ip_address FROM agents WHERE id = ?", agentID).Scan(&savedIP)
	if err != nil {
		t.Fatalf("Failed to retrieve updated agent: %v", err)
	}

	if savedIP != newIP {
		t.Errorf("Updated IP = %s, want %s", savedIP, newIP)
	}
}

func TestUpdateAgentInfo(t *testing.T) {
	db, tempDir := setupTestDB(t)
	defer cleanupTestDB(db, tempDir)

	// Create initial agent
	agentID := "test-agent-456"
	err := db.SaveAgent(agentID, "10.0.0.1")
	if err != nil {
		t.Fatalf("Failed to save initial agent: %v", err)
	}

	// Update agent info
	hostname := "new-host"
	username := "new-user"
	os := "linux"

	err = db.UpdateAgentInfo(agentID, hostname, username, os)
	if err != nil {
		t.Fatalf("Failed to update agent info: %v", err)
	}

	// Verify update
	var savedHostname, savedUsername, savedOS string
	err = db.db.QueryRow("SELECT hostname, username, os FROM agents WHERE id = ?", agentID).Scan(
		&savedHostname, &savedUsername, &savedOS,
	)
	if err != nil {
		t.Fatalf("Failed to retrieve updated agent: %v", err)
	}

	if savedHostname != hostname {
		t.Errorf("Hostname = %s, want %s", savedHostname, hostname)
	}
	if savedUsername != username {
		t.Errorf("Username = %s, want %s", savedUsername, username)
	}
	if savedOS != os {
		t.Errorf("OS = %s, want %s", savedOS, os)
	}
}

func TestGetAgents(t *testing.T) {
	db, tempDir := setupTestDB(t)
	defer cleanupTestDB(db, tempDir)

	// Test with no agents
	agents, err := db.GetAgents()
	if err != nil {
		t.Fatalf("Failed to get agents: %v", err)
	}
	if len(agents) != 0 {
		t.Errorf("Expected 0 agents, got %d", len(agents))
	}

	// Add multiple agents
	testAgents := []struct {
		id string
		ip string
	}{
		{"agent-1", "192.168.1.1"},
		{"agent-2", "192.168.1.2"},
		{"agent-3", "192.168.1.3"},
	}

	for _, agent := range testAgents {
		if err := db.SaveAgent(agent.id, agent.ip); err != nil {
			t.Fatalf("Failed to save agent %s: %v", agent.id, err)
		}
		time.Sleep(5 * time.Millisecond) // Ensure different timestamps
	}

	// Update agent info for testing
	db.UpdateAgentInfo("agent-1", "host-1", "user-1", "linux")
	db.UpdateAgentInfo("agent-2", "host-2", "user-2", "windows")
	db.UpdateAgentInfo("agent-3", "host-3", "user-3", "darwin")

	// Get all agents
	agents, err = db.GetAgents()
	if err != nil {
		t.Fatalf("Failed to get agents: %v", err)
	}

	if len(agents) != len(testAgents) {
		t.Errorf("Expected %d agents, got %d", len(testAgents), len(agents))
	}

	// Verify each agent
	agentMap := make(map[string]map[string]interface{})
	for _, agent := range agents {
		agentMap[agent["id"].(string)] = agent
	}

	for i, expected := range testAgents {
		agent, exists := agentMap[expected.id]
		if !exists {
			t.Errorf("Agent %s not found in results", expected.id)
			continue
		}

		if agent["ip_address"] != expected.ip {
			t.Errorf("Agent %s IP = %s, want %s", expected.id, agent["ip_address"], expected.ip)
		}

		// Verify timestamps are set
		if firstSeen, ok := agent["first_seen"].(time.Time); !ok || firstSeen.IsZero() {
			t.Errorf("Agent %s FirstSeen is invalid", expected.id)
		}
		if lastSeen, ok := agent["last_seen"].(time.Time); !ok || lastSeen.IsZero() {
			t.Errorf("Agent %s LastSeen is invalid", expected.id)
		}

		// Verify hostname was set
		expectedHostname := fmt.Sprintf("host-%d", i+1)
		if agent["hostname"] != expectedHostname {
			t.Errorf("Agent %s hostname = %v, want %s", expected.id, agent["hostname"], expectedHostname)
		}
	}
}

func TestSessionOperations(t *testing.T) {
	db, tempDir := setupTestDB(t)
	defer cleanupTestDB(db, tempDir)

	// Create agent first
	agentID := "test-agent"
	if err := db.SaveAgent(agentID, "192.168.1.100"); err != nil {
		t.Fatalf("Failed to save agent: %v", err)
	}

	// Create session
	sessionID := "test-session-123"
	err := db.CreateSession(sessionID, agentID)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Verify session was created
	var count int
	err = db.db.QueryRow("SELECT COUNT(*) FROM sessions WHERE id = ? AND agent_id = ?", sessionID, agentID).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query session: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected 1 session, got %d", count)
	}

	// Verify session is open (end_time is NULL)
	var endTime sql.NullTime
	err = db.db.QueryRow("SELECT end_time FROM sessions WHERE id = ?", sessionID).Scan(&endTime)
	if err != nil {
		t.Fatalf("Failed to query session end_time: %v", err)
	}
	if endTime.Valid {
		t.Error("New session should have NULL end_time")
	}

	// Close session
	err = db.CloseSession(sessionID)
	if err != nil {
		t.Fatalf("Failed to close session: %v", err)
	}

	// Verify session was closed
	err = db.db.QueryRow("SELECT end_time FROM sessions WHERE id = ?", sessionID).Scan(&endTime)
	if err != nil {
		t.Fatalf("Failed to query closed session: %v", err)
	}
	if !endTime.Valid {
		t.Error("Closed session should have non-NULL end_time")
	}
}

func TestCommandOperations(t *testing.T) {
	db, tempDir := setupTestDB(t)
	defer cleanupTestDB(db, tempDir)

	// Setup agent and session
	agentID := "test-agent"
	sessionID := "test-session"

	if err := db.SaveAgent(agentID, "192.168.1.100"); err != nil {
		t.Fatalf("Failed to save agent: %v", err)
	}

	if err := db.CreateSession(sessionID, agentID); err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Log command
	command := "ls -la /tmp"
	arguments := "-la /tmp"
	commandID, err := db.LogCommand(sessionID, command, arguments)
	if err != nil {
		t.Fatalf("Failed to log command: %v", err)
	}

	if commandID == 0 {
		t.Error("Command ID should not be 0")
	}

	// Verify command was logged
	var savedCmd struct {
		SessionID string
		Command   string
		Arguments string
		Status    string
	}

	err = db.db.QueryRow("SELECT session_id, command, arguments, status FROM commands WHERE id = ?", commandID).Scan(
		&savedCmd.SessionID,
		&savedCmd.Command,
		&savedCmd.Arguments,
		&savedCmd.Status,
	)
	if err != nil {
		t.Fatalf("Failed to query command: %v", err)
	}

	if savedCmd.SessionID != sessionID {
		t.Errorf("Session ID = %s, want %s", savedCmd.SessionID, sessionID)
	}
	if savedCmd.Command != command {
		t.Errorf("Command = %s, want %s", savedCmd.Command, command)
	}
	if savedCmd.Arguments != arguments {
		t.Errorf("Arguments = %s, want %s", savedCmd.Arguments, arguments)
	}
	if savedCmd.Status != "pending" {
		t.Errorf("Status = %s, want pending", savedCmd.Status)
	}

	// Log command result
	output := "file1.txt\nfile2.log\ndir1/"
	exitCode := 0
	err = db.LogCommandResult(commandID, output, exitCode, "")
	if err != nil {
		t.Fatalf("Failed to log command result: %v", err)
	}

	// Verify result was logged
	var result struct {
		Output   string
		ExitCode int
		Error    sql.NullString
	}

	err = db.db.QueryRow("SELECT output, exit_code, error FROM command_results WHERE command_id = ?", commandID).Scan(
		&result.Output,
		&result.ExitCode,
		&result.Error,
	)
	if err != nil {
		t.Fatalf("Failed to query command result: %v", err)
	}

	if result.Output != output {
		t.Errorf("Output = %s, want %s", result.Output, output)
	}
	if result.ExitCode != exitCode {
		t.Errorf("Exit code = %d, want %d", result.ExitCode, exitCode)
	}
	// Empty string error message should result in empty string, not NULL
	if result.Error.Valid && result.Error.String != "" {
		t.Errorf("Error = %v, want empty string or NULL", result.Error.String)
	}

	// Verify command status was updated
	err = db.db.QueryRow("SELECT status FROM commands WHERE id = ?", commandID).Scan(&savedCmd.Status)
	if err != nil {
		t.Fatalf("Failed to query command status: %v", err)
	}

	if savedCmd.Status != "success" {
		t.Errorf("Command status = %s, want success", savedCmd.Status)
	}
}

func TestCommandResultWithError(t *testing.T) {
	db, tempDir := setupTestDB(t)
	defer cleanupTestDB(db, tempDir)

	// Setup
	agentID := "test-agent"
	sessionID := "test-session"

	db.SaveAgent(agentID, "192.168.1.100")
	db.CreateSession(sessionID, agentID)

	// Log command
	commandID, _ := db.LogCommand(sessionID, "cat /non/existent/file", "/non/existent/file")

	// Log result with error
	errorMsg := "No such file or directory"
	err := db.LogCommandResult(commandID, "", 1, errorMsg)
	if err != nil {
		t.Fatalf("Failed to log command result with error: %v", err)
	}

	// Verify error was saved
	var savedError string
	err = db.db.QueryRow("SELECT error FROM command_results WHERE command_id = ?", commandID).Scan(&savedError)
	if err != nil {
		t.Fatalf("Failed to query error: %v", err)
	}

	if savedError != errorMsg {
		t.Errorf("Error = %s, want %s", savedError, errorMsg)
	}

	// Verify command status
	var status string
	db.db.QueryRow("SELECT status FROM commands WHERE id = ?", commandID).Scan(&status)

	if status != "failed" {
		t.Errorf("Command status = %s, want failed", status)
	}
}

func TestGetCommandHistory(t *testing.T) {
	db, tempDir := setupTestDB(t)
	defer cleanupTestDB(db, tempDir)

	// Setup
	agentID := "test-agent"
	sessionID := "test-session"

	db.SaveAgent(agentID, "192.168.1.100")
	db.CreateSession(sessionID, agentID)

	// Log multiple commands
	commands := []struct {
		cmd    string
		args   string
		output string
		code   int
		err    string
	}{
		{"ls", "-la", "file1\nfile2", 0, ""},
		{"pwd", "", "/home/user", 0, ""},
		{"cat", "nonexistent", "", 1, "file not found"},
	}

	for _, cmd := range commands {
		id, err := db.LogCommand(sessionID, cmd.cmd, cmd.args)
		if err != nil {
			t.Fatalf("Failed to log command %s: %v", cmd.cmd, err)
		}

		err = db.LogCommandResult(id, cmd.output, cmd.code, cmd.err)
		if err != nil {
			t.Fatalf("Failed to log result for command %s: %v", cmd.cmd, err)
		}

		time.Sleep(5 * time.Millisecond) // Ensure different timestamps
	}

	// Get command history
	history, err := db.GetCommandHistory(sessionID)
	if err != nil {
		t.Fatalf("Failed to get command history: %v", err)
	}

	if len(history) != len(commands) {
		t.Errorf("Expected %d commands, got %d", len(commands), len(history))
	}

	// Verify commands are in order (newest first due to ORDER BY timestamp DESC)
	for i, cmd := range history {
		expectedIdx := len(commands) - 1 - i
		expected := commands[expectedIdx]

		if cmd["command"] != expected.cmd {
			t.Errorf("Command[%d] = %s, want %s", i, cmd["command"], expected.cmd)
		}
		if cmd["arguments"] != expected.args {
			t.Errorf("Arguments[%d] = %s, want %s", i, cmd["arguments"], expected.args)
		}
		if cmd["output"] != expected.output {
			t.Errorf("Output[%d] = %s, want %s", i, cmd["output"], expected.output)
		}
		if exitCode, ok := cmd["exit_code"].(int64); !ok || int(exitCode) != expected.code {
			t.Errorf("ExitCode[%d] = %v, want %d", i, cmd["exit_code"], expected.code)
		}

		// Check error field
		if cmd["error"] != expected.err {
			t.Errorf("Error[%d] = %v, want %s", i, cmd["error"], expected.err)
		}

		// Verify timestamp is set
		if timestamp, ok := cmd["timestamp"].(time.Time); !ok || timestamp.IsZero() {
			t.Errorf("Command[%d] timestamp is invalid", i)
		}
	}
}

func TestGetAgentHistory(t *testing.T) {
	db, tempDir := setupTestDB(t)
	defer cleanupTestDB(db, tempDir)

	// Create agent
	agentID := "test-agent"
	db.SaveAgent(agentID, "192.168.1.100")

	// Create multiple sessions
	sessions := []string{"session-1", "session-2", "session-3"}
	for _, sessionID := range sessions {
		db.CreateSession(sessionID, agentID)

		// Log commands for each session
		for i := 0; i < 3; i++ {
			cmd := fmt.Sprintf("cmd-%s-%d", sessionID, i)
			id, _ := db.LogCommand(sessionID, cmd, "")
			db.LogCommandResult(id, fmt.Sprintf("output-%s-%d", sessionID, i), 0, "")
			time.Sleep(5 * time.Millisecond)
		}
	}

	// Get agent history
	history, err := db.GetAgentHistory(agentID)
	if err != nil {
		t.Fatalf("Failed to get agent history: %v", err)
	}

	// Should have commands from all sessions
	expectedTotal := len(sessions) * 3
	if len(history) != expectedTotal {
		t.Errorf("Expected %d commands, got %d", expectedTotal, len(history))
	}

	// Verify all sessions are represented
	commandCount := 0
	for _, cmd := range history {
		if _, ok := cmd["command"].(string); ok {
			commandCount++
		}
	}

	if commandCount != expectedTotal {
		t.Errorf("Expected %d valid commands, got %d", expectedTotal, commandCount)
	}
}

func TestFileOperations(t *testing.T) {
	db, tempDir := setupTestDB(t)
	defer cleanupTestDB(db, tempDir)

	// Setup
	agentID := "test-agent"
	sessionID := "test-session"

	db.SaveAgent(agentID, "192.168.1.100")
	db.CreateSession(sessionID, agentID)

	// Log file
	filename := "test-file.txt"
	filePath := "/tmp/test-file.txt"
	fileSize := int64(1024)
	fileHash := "d41d8cd98f00b204e9800998ecf8427e"

	err := db.LogFile(sessionID, filename, filePath, fileHash, fileSize)
	if err != nil {
		t.Fatalf("Failed to log file: %v", err)
	}

	// Verify file was logged
	var count int
	err = db.db.QueryRow("SELECT COUNT(*) FROM files WHERE session_id = ? AND filename = ?", sessionID, filename).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query file: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected 1 file, got %d", count)
	}

	// Get file operations
	files, err := db.GetFileOperations(sessionID)
	if err != nil {
		t.Fatalf("Failed to get file operations: %v", err)
	}

	if len(files) != 1 {
		t.Fatalf("Expected 1 file operation, got %d", len(files))
	}

	file := files[0]
	if file["filename"] != filename {
		t.Errorf("Filename = %v, want %s", file["filename"], filename)
	}
	if file["file_path"] != filePath {
		t.Errorf("FilePath = %v, want %s", file["file_path"], filePath)
	}
	if fileSize, ok := file["file_size"].(int64); !ok || fileSize != 1024 {
		t.Errorf("FileSize = %v, want %d", file["file_size"], 1024)
	}
	if file["file_hash"] != fileHash {
		t.Errorf("FileHash = %v, want %s", file["file_hash"], fileHash)
	}
	if uploadTime, ok := file["upload_time"].(time.Time); !ok || uploadTime.IsZero() {
		t.Error("UploadTime is invalid")
	}
}

func TestGetStats(t *testing.T) {
	db, tempDir := setupTestDB(t)
	defer cleanupTestDB(db, tempDir)

	// Get initial stats (empty database)
	stats := db.GetStats()

	if stats["total_agents"] != 0 {
		t.Errorf("Initial total_agents = %d, want 0", stats["total_agents"])
	}
	if stats["total_sessions"] != 0 {
		t.Errorf("Initial total_sessions = %d, want 0", stats["total_sessions"])
	}
	if stats["total_commands"] != 0 {
		t.Errorf("Initial total_commands = %d, want 0", stats["total_commands"])
	}
	if stats["total_files"] != 0 {
		t.Errorf("Initial total_files = %d, want 0", stats["total_files"])
	}

	// Add data
	for i := 0; i < 3; i++ {
		agentID := fmt.Sprintf("agent-%d", i)
		db.SaveAgent(agentID, "192.168.1.100")

		// Create 2 sessions per agent
		for j := 0; j < 2; j++ {
			sessionID := fmt.Sprintf("session-%d-%d", i, j)
			db.CreateSession(sessionID, agentID)

			// Add commands
			for k := 0; k < 5; k++ {
				id, _ := db.LogCommand(sessionID, fmt.Sprintf("cmd-%d", k), "")
				db.LogCommandResult(id, "output", 0, "")
			}

			// Add files
			for k := 0; k < 2; k++ {
				db.LogFile(sessionID, fmt.Sprintf("file-%d.txt", k), "/tmp", "hash", 1024)
			}
		}
	}

	// Get updated stats
	stats = db.GetStats()

	if stats["total_agents"] != 3 {
		t.Errorf("total_agents = %d, want 3", stats["total_agents"])
	}
	if stats["total_sessions"] != 6 { // 3 agents * 2 sessions
		t.Errorf("total_sessions = %d, want 6", stats["total_sessions"])
	}
	if stats["total_commands"] != 30 { // 3 agents * 2 sessions * 5 commands
		t.Errorf("total_commands = %d, want 30", stats["total_commands"])
	}
	if stats["total_files"] != 12 { // 3 agents * 2 sessions * 2 files
		t.Errorf("total_files = %d, want 12", stats["total_files"])
	}
}

func TestConcurrentOperations(t *testing.T) {
	db, tempDir := setupTestDB(t)
	defer cleanupTestDB(db, tempDir)

	// Create multiple agents concurrently
	numAgents := 10
	done := make(chan bool, numAgents)

	for i := 0; i < numAgents; i++ {
		go func(id int) {
			agentID := fmt.Sprintf("agent-%d", id)

			if err := db.SaveAgent(agentID, fmt.Sprintf("192.168.1.%d", id)); err != nil {
				t.Errorf("Failed to save agent %s: %v", agentID, err)
			}

			// Update agent info to avoid NULL values
			if err := db.UpdateAgentInfo(agentID, fmt.Sprintf("host-%d", id), fmt.Sprintf("user-%d", id), "linux"); err != nil {
				t.Errorf("Failed to update agent info %s: %v", agentID, err)
			}

			// Create session
			sessionID := fmt.Sprintf("session-%d", id)
			if err := db.CreateSession(sessionID, agentID); err != nil {
				t.Errorf("Failed to create session %s: %v", sessionID, err)
			}

			// Log commands
			for j := 0; j < 5; j++ {
				cmdID, err := db.LogCommand(sessionID, fmt.Sprintf("cmd-%d-%d", id, j), "")
				if err != nil {
					t.Errorf("Failed to log command: %v", err)
					continue
				}

				if err := db.LogCommandResult(cmdID, "output", 0, ""); err != nil {
					t.Errorf("Failed to log result: %v", err)
				}
			}

			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < numAgents; i++ {
		<-done
	}

	// Verify all agents were created
	agents, err := db.GetAgents()
	if err != nil {
		t.Fatalf("Failed to get agents: %v", err)
	}

	if len(agents) != numAgents {
		t.Errorf("Expected %d agents, got %d", numAgents, len(agents))
	}

	// Verify stats
	stats := db.GetStats()

	if stats["total_agents"] != numAgents {
		t.Errorf("total_agents = %d, want %d", stats["total_agents"], numAgents)
	}
	if stats["total_commands"] != numAgents*5 {
		t.Errorf("total_commands = %d, want %d", stats["total_commands"], numAgents*5)
	}
}

func TestQueryRow(t *testing.T) {
	db, tempDir := setupTestDB(t)
	defer cleanupTestDB(db, tempDir)

	// Add test data
	db.SaveAgent("test-agent", "192.168.1.100")
	db.UpdateAgentInfo("test-agent", "test-host", "test-user", "linux")

	// Test custom query
	var hostname string
	err := db.QueryRow("SELECT hostname FROM agents WHERE id = ?", "test-agent").Scan(&hostname)
	if err != nil {
		t.Fatalf("Failed to execute custom query: %v", err)
	}

	if hostname != "test-host" {
		t.Errorf("Hostname = %s, want test-host", hostname)
	}

	// Test query with no results
	err = db.QueryRow("SELECT hostname FROM agents WHERE id = ?", "non-existent").Scan(&hostname)
	if err == nil {
		t.Error("Expected error for non-existent record")
	}
}

func TestDatabaseClose(t *testing.T) {
	db, tempDir := setupTestDB(t)
	defer os.RemoveAll(tempDir)

	// Close database
	err := db.Close()
	if err != nil {
		t.Fatalf("Failed to close database: %v", err)
	}

	// Verify operations fail after close
	err = db.SaveAgent("test", "192.168.1.100")
	if err == nil {
		t.Error("Expected error when using closed database")
	}
}

// Benchmark tests
func BenchmarkSaveAgent(b *testing.B) {
	db, tempDir := setupTestDB(&testing.T{})
	defer cleanupTestDB(db, tempDir)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		db.SaveAgent(fmt.Sprintf("agent-%d", i), fmt.Sprintf("192.168.1.%d", i%255))
	}
}

func BenchmarkLogCommand(b *testing.B) {
	db, tempDir := setupTestDB(&testing.T{})
	defer cleanupTestDB(db, tempDir)

	// Setup
	db.SaveAgent("bench-agent", "192.168.1.100")
	db.CreateSession("bench-session", "bench-agent")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		db.LogCommand("bench-session", fmt.Sprintf("command-%d", i), "args")
	}
}

func BenchmarkGetCommandHistory(b *testing.B) {
	db, tempDir := setupTestDB(&testing.T{})
	defer cleanupTestDB(db, tempDir)

	// Setup with many commands
	db.SaveAgent("bench-agent", "192.168.1.100")
	db.CreateSession("bench-session", "bench-agent")

	for i := 0; i < 1000; i++ {
		id, _ := db.LogCommand("bench-session", fmt.Sprintf("cmd-%d", i), "")
		db.LogCommandResult(id, "output", 0, "")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		db.GetCommandHistory("bench-session")
	}
}

func BenchmarkGetStats(b *testing.B) {
	db, tempDir := setupTestDB(&testing.T{})
	defer cleanupTestDB(db, tempDir)

	// Add some data
	for i := 0; i < 100; i++ {
		agentID := fmt.Sprintf("agent-%d", i)
		db.SaveAgent(agentID, "192.168.1.100")
		sessionID := fmt.Sprintf("session-%d", i)
		db.CreateSession(sessionID, agentID)

		for j := 0; j < 10; j++ {
			id, _ := db.LogCommand(sessionID, "cmd", "")
			db.LogCommandResult(id, "output", 0, "")
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		db.GetStats()
	}
}
