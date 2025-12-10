package mock

import (
	"fmt"
	"sync"

	"github.com/r74tech/virga/internal/server/protocol"
	"github.com/r74tech/virga/internal/server/session"
)

// MockSessionManager implements a mock version of the session.Manager for testing
type MockSessionManager struct {
	mu                sync.RWMutex
	sessions          map[string]*session.Session
	totalSessionCount int

	// Function fields for customizing behavior
	AddSessionFunc           func(session *session.Session)
	GetSessionFunc           func(sessionID string) *session.Session
	RemoveSessionFunc        func(sessionID string)
	GetActiveSessionsFunc    func() []*session.Session
	AddTaskFunc              func(sessionID string, task *protocol.Task) error
	GetSessionsByBeaconFunc  func(beaconID string) []*session.Session
	GetTotalSessionCountFunc func() int
}

// NewMockSessionManager creates a new mock session manager with default behavior
func NewMockSessionManager() *MockSessionManager {
	return &MockSessionManager{
		sessions: make(map[string]*session.Session),
	}
}

// AddSession adds a session to the manager
func (m *MockSessionManager) AddSession(s *session.Session) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.AddSessionFunc != nil {
		m.AddSessionFunc(s)
		return
	}

	// Default behavior
	m.sessions[s.ID] = s
	m.totalSessionCount++
}

// GetSession retrieves a session by ID
func (m *MockSessionManager) GetSession(sessionID string) *session.Session {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.GetSessionFunc != nil {
		return m.GetSessionFunc(sessionID)
	}

	// Default behavior
	return m.sessions[sessionID]
}

// RemoveSession removes a session from the manager
func (m *MockSessionManager) RemoveSession(sessionID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.RemoveSessionFunc != nil {
		m.RemoveSessionFunc(sessionID)
		return
	}

	// Default behavior
	delete(m.sessions, sessionID)
}

// GetActiveSessions returns all active sessions
func (m *MockSessionManager) GetActiveSessions() []*session.Session {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.GetActiveSessionsFunc != nil {
		return m.GetActiveSessionsFunc()
	}

	// Default behavior
	sessions := make([]*session.Session, 0, len(m.sessions))
	for _, s := range m.sessions {
		sessions = append(sessions, s)
	}
	return sessions
}

// AddTask adds a task to a specific session
func (m *MockSessionManager) AddTask(sessionID string, task *protocol.Task) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.AddTaskFunc != nil {
		return m.AddTaskFunc(sessionID, task)
	}

	// Default behavior
	if s, exists := m.sessions[sessionID]; exists {
		s.AddTask(*task)
		return nil
	}

	return fmt.Errorf("session not found: %s", sessionID)
}

// GetSessionsByBeacon returns all sessions for a given beacon
func (m *MockSessionManager) GetSessionsByBeacon(beaconID string) []*session.Session {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.GetSessionsByBeaconFunc != nil {
		return m.GetSessionsByBeaconFunc(beaconID)
	}

	// Default behavior
	sessions := make([]*session.Session, 0)
	for _, s := range m.sessions {
		if s.BeaconID == beaconID {
			sessions = append(sessions, s)
		}
	}
	return sessions
}

// GetTotalSessionCount returns the total number of sessions ever created
func (m *MockSessionManager) GetTotalSessionCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.GetTotalSessionCountFunc != nil {
		return m.GetTotalSessionCountFunc()
	}

	// Default behavior
	return m.totalSessionCount
}

// Test helper methods

// CreateTestSession creates a test session and adds it to the manager
func (m *MockSessionManager) CreateTestSession(id, agentID string) *session.Session {
	conn := NewMockAgentConnection()
	s := session.NewSession(id, conn)
	s.UpdateInformation(map[string]interface{}{
		"agent_id": agentID,
		"hostname": "test-host",
		"username": "test-user",
		"os":       "test-os",
	})

	m.AddSession(s)
	return s
}

// GetSessionCount returns the current number of sessions
func (m *MockSessionManager) GetSessionCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.sessions)
}

// HasSession checks if a session exists
func (m *MockSessionManager) HasSession(sessionID string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, exists := m.sessions[sessionID]
	return exists
}

// GetAllSessions returns all sessions (for testing)
func (m *MockSessionManager) GetAllSessions() map[string]*session.Session {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Return a copy to prevent modification
	sessionsCopy := make(map[string]*session.Session)
	for k, v := range m.sessions {
		sessionsCopy[k] = v
	}
	return sessionsCopy
}

// Clear removes all sessions (for testing)
func (m *MockSessionManager) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sessions = make(map[string]*session.Session)
	// Note: totalSessionCount is not reset to maintain accurate count
}

// Reset clears everything including counters (for testing)
func (m *MockSessionManager) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sessions = make(map[string]*session.Session)
	m.totalSessionCount = 0
}
