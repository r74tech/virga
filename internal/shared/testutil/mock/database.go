package mock

import (
	"database/sql"
	"fmt"
	"sync"
	"time"
)

// MockDatabase implements a mock version of the database.Database interface for testing
type MockDatabase struct {
	mu sync.Mutex

	// Function fields for customizing behavior
	SaveAgentFunc         func(agentID, ipAddress string) error
	UpdateAgentInfoFunc   func(agentID, hostname, username, os string) error
	GetAgentsFunc         func() ([]map[string]interface{}, error)
	CreateSessionFunc     func(sessionID, agentID string) error
	CloseSessionFunc      func(sessionID string) error
	LogCommandFunc        func(sessionID, command, arguments string) (int64, error)
	LogCommandResultFunc  func(commandID int64, output string, exitCode int, errorMsg string) error
	GetCommandHistoryFunc func(sessionID string) ([]map[string]interface{}, error)
	GetAgentHistoryFunc   func(agentID string) ([]map[string]interface{}, error)
	LogFileFunc           func(sessionID, filename, filePath, fileHash string, fileSize int64) error
	GetFileOperationsFunc func(sessionID string) ([]map[string]interface{}, error)
	GetStatsFunc          func() map[string]int
	QueryRowFunc          func(query string, args ...interface{}) *sql.Row
	ExecFunc              func(query string, args ...interface{}) (sql.Result, error)
	CloseFunc             func() error

	// Storage for tracking calls and data
	agents    map[string]map[string]interface{}
	sessions  map[string]map[string]interface{}
	commands  []map[string]interface{}
	commandID int64
	files     []map[string]interface{}
	closed    bool
}

// NewMockDatabase creates a new mock database with default behavior
func NewMockDatabase() *MockDatabase {
	return &MockDatabase{
		agents:   make(map[string]map[string]interface{}),
		sessions: make(map[string]map[string]interface{}),
		commands: make([]map[string]interface{}, 0),
		files:    make([]map[string]interface{}, 0),
	}
}

// SaveAgent saves or updates an agent
func (m *MockDatabase) SaveAgent(agentID, ipAddress string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.SaveAgentFunc != nil {
		return m.SaveAgentFunc(agentID, ipAddress)
	}

	// Default behavior
	if _, exists := m.agents[agentID]; !exists {
		m.agents[agentID] = make(map[string]interface{})
	}
	m.agents[agentID]["id"] = agentID
	m.agents[agentID]["ip_address"] = ipAddress
	m.agents[agentID]["last_seen"] = time.Now()

	return nil
}

// UpdateAgentInfo updates agent information
func (m *MockDatabase) UpdateAgentInfo(agentID, hostname, username, os string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.UpdateAgentInfoFunc != nil {
		return m.UpdateAgentInfoFunc(agentID, hostname, username, os)
	}

	// Default behavior
	if agent, exists := m.agents[agentID]; exists {
		agent["hostname"] = hostname
		agent["username"] = username
		agent["os"] = os
		return nil
	}

	return fmt.Errorf("agent not found: %s", agentID)
}

// GetAgents returns all agents
func (m *MockDatabase) GetAgents() ([]map[string]interface{}, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.GetAgentsFunc != nil {
		return m.GetAgentsFunc()
	}

	// Default behavior
	agents := make([]map[string]interface{}, 0, len(m.agents))
	for _, agent := range m.agents {
		// Create a copy to prevent modification
		agentCopy := make(map[string]interface{})
		for k, v := range agent {
			agentCopy[k] = v
		}
		agents = append(agents, agentCopy)
	}

	return agents, nil
}

// CreateSession creates a new session
func (m *MockDatabase) CreateSession(sessionID, agentID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.CreateSessionFunc != nil {
		return m.CreateSessionFunc(sessionID, agentID)
	}

	// Default behavior
	m.sessions[sessionID] = map[string]interface{}{
		"id":         sessionID,
		"agent_id":   agentID,
		"start_time": time.Now(),
		"end_time":   nil,
	}

	return nil
}

// CloseSession closes a session
func (m *MockDatabase) CloseSession(sessionID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.CloseSessionFunc != nil {
		return m.CloseSessionFunc(sessionID)
	}

	// Default behavior
	if session, exists := m.sessions[sessionID]; exists {
		session["end_time"] = time.Now()
		return nil
	}

	return fmt.Errorf("session not found: %s", sessionID)
}

// LogCommand logs a command execution
func (m *MockDatabase) LogCommand(sessionID, command, arguments string) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.LogCommandFunc != nil {
		return m.LogCommandFunc(sessionID, command, arguments)
	}

	// Default behavior
	m.commandID++
	m.commands = append(m.commands, map[string]interface{}{
		"id":         m.commandID,
		"session_id": sessionID,
		"command":    command,
		"arguments":  arguments,
		"timestamp":  time.Now(),
		"status":     "pending",
	})

	return m.commandID, nil
}

// LogCommandResult logs command execution result
func (m *MockDatabase) LogCommandResult(commandID int64, output string, exitCode int, errorMsg string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.LogCommandResultFunc != nil {
		return m.LogCommandResultFunc(commandID, output, exitCode, errorMsg)
	}

	// Default behavior
	for _, cmd := range m.commands {
		if id, ok := cmd["id"].(int64); ok && id == commandID {
			cmd["output"] = output
			cmd["exit_code"] = exitCode
			cmd["error"] = errorMsg
			if exitCode == 0 && errorMsg == "" {
				cmd["status"] = "success"
			} else {
				cmd["status"] = "failed"
			}
			return nil
		}
	}

	return fmt.Errorf("command not found: %d", commandID)
}

// GetCommandHistory gets command history for a session
func (m *MockDatabase) GetCommandHistory(sessionID string) ([]map[string]interface{}, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.GetCommandHistoryFunc != nil {
		return m.GetCommandHistoryFunc(sessionID)
	}

	// Default behavior
	history := make([]map[string]interface{}, 0)
	for _, cmd := range m.commands {
		if sid, ok := cmd["session_id"].(string); ok && sid == sessionID {
			// Create a copy
			cmdCopy := make(map[string]interface{})
			for k, v := range cmd {
				cmdCopy[k] = v
			}
			history = append(history, cmdCopy)
		}
	}

	return history, nil
}

// GetAgentHistory gets all command history for an agent
func (m *MockDatabase) GetAgentHistory(agentID string) ([]map[string]interface{}, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.GetAgentHistoryFunc != nil {
		return m.GetAgentHistoryFunc(agentID)
	}

	// Default behavior - find all sessions for agent, then all commands
	history := make([]map[string]interface{}, 0)
	for sessionID, session := range m.sessions {
		if aid, ok := session["agent_id"].(string); ok && aid == agentID {
			// Get commands for this session
			for _, cmd := range m.commands {
				if sid, ok := cmd["session_id"].(string); ok && sid == sessionID {
					// Create a copy
					cmdCopy := make(map[string]interface{})
					for k, v := range cmd {
						cmdCopy[k] = v
					}
					history = append(history, cmdCopy)
				}
			}
		}
	}

	return history, nil
}

// LogFile logs a file operation
func (m *MockDatabase) LogFile(sessionID, filename, filePath, fileHash string, fileSize int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.LogFileFunc != nil {
		return m.LogFileFunc(sessionID, filename, filePath, fileHash, fileSize)
	}

	// Default behavior
	m.files = append(m.files, map[string]interface{}{
		"session_id":  sessionID,
		"filename":    filename,
		"file_path":   filePath,
		"file_hash":   fileHash,
		"file_size":   fileSize,
		"upload_time": time.Now(),
	})

	return nil
}

// GetFileOperations gets file operations for a session
func (m *MockDatabase) GetFileOperations(sessionID string) ([]map[string]interface{}, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.GetFileOperationsFunc != nil {
		return m.GetFileOperationsFunc(sessionID)
	}

	// Default behavior
	operations := make([]map[string]interface{}, 0)
	for _, file := range m.files {
		if sid, ok := file["session_id"].(string); ok && sid == sessionID {
			// Create a copy
			fileCopy := make(map[string]interface{})
			for k, v := range file {
				fileCopy[k] = v
			}
			operations = append(operations, fileCopy)
		}
	}

	return operations, nil
}

// GetStats returns database statistics
func (m *MockDatabase) GetStats() map[string]int {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.GetStatsFunc != nil {
		return m.GetStatsFunc()
	}

	// Default behavior
	return map[string]int{
		"total_agents":   len(m.agents),
		"total_sessions": len(m.sessions),
		"total_commands": len(m.commands),
		"total_files":    len(m.files),
	}
}

// QueryRow executes a custom query (minimal implementation)
func (m *MockDatabase) QueryRow(query string, args ...interface{}) *sql.Row {
	if m.QueryRowFunc != nil {
		return m.QueryRowFunc(query, args...)
	}

	// This is a simplified mock - real implementation would need more
	return nil
}

// Exec executes a query without returning any rows.
func (m *MockDatabase) Exec(query string, args ...interface{}) (sql.Result, error) {
	if m.ExecFunc != nil {
		return m.ExecFunc(query, args...)
	}
	return nil, nil
}

// Close closes the database
func (m *MockDatabase) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.CloseFunc != nil {
		return m.CloseFunc()
	}

	// Default behavior
	m.closed = true
	return nil
}

// Test helper methods

// IsClosed returns whether the database has been closed
func (m *MockDatabase) IsClosed() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.closed
}

// GetAgent returns a specific agent by ID
func (m *MockDatabase) GetAgent(agentID string) (map[string]interface{}, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	agent, exists := m.agents[agentID]
	if !exists {
		return nil, false
	}

	// Return a copy
	agentCopy := make(map[string]interface{})
	for k, v := range agent {
		agentCopy[k] = v
	}
	return agentCopy, true
}

// GetSession returns a specific session by ID
func (m *MockDatabase) GetSession(sessionID string) (map[string]interface{}, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	session, exists := m.sessions[sessionID]
	if !exists {
		return nil, false
	}

	// Return a copy
	sessionCopy := make(map[string]interface{})
	for k, v := range session {
		sessionCopy[k] = v
	}
	return sessionCopy, true
}

// GetCommandCount returns the number of logged commands
func (m *MockDatabase) GetCommandCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.commands)
}

// GetFileCount returns the number of logged files
func (m *MockDatabase) GetFileCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.files)
}

// Reset clears all data
func (m *MockDatabase) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.agents = make(map[string]map[string]interface{})
	m.sessions = make(map[string]map[string]interface{})
	m.commands = make([]map[string]interface{}, 0)
	m.files = make([]map[string]interface{}, 0)
	m.commandID = 0
	m.closed = false
}
