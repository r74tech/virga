package session

import (
	"fmt"
	"sync"

	"github.com/r74tech/virga/internal/server/protocol"
)

// Manager is the manager for session management
type Manager struct {
	sessions      map[string]*Session
	mu            sync.RWMutex
	totalSessions int
	disconnected  int
}

// NewManager creates a new session manager
func NewManager() *Manager {
	return &Manager{
		sessions: make(map[string]*Session),
	}
}

// AddSession adds a session
func (m *Manager) AddSession(session *Session) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.sessions[session.ID] = session
	m.totalSessions++
}

// GetSession gets a session by ID
func (m *Manager) GetSession(id string) *Session {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.sessions[id]
}

// GetActiveSessions gets all active sessions
func (m *Manager) GetActiveSessions() []*Session {
	m.mu.RLock()
	defer m.mu.RUnlock()

	sessions := make([]*Session, 0, len(m.sessions))
	for _, session := range m.sessions {
		if session.Status == "active" {
			sessions = append(sessions, session)
		}
	}
	return sessions
}

// RemoveSession removes a session
func (m *Manager) RemoveSession(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if session, exists := m.sessions[id]; exists {
		if session.Status == "active" {
			m.disconnected++
		}
		delete(m.sessions, id)
	}
}

// GetTotalSessionCount gets the total number of sessions
func (m *Manager) GetTotalSessionCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.totalSessions
}

// GetDisconnectedSessionCount gets the number of disconnected sessions
func (m *Manager) GetDisconnectedSessionCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.disconnected
}

// AddTask adds a task to a session
func (m *Manager) AddTask(sessionID string, task *protocol.Task) error {
	m.mu.RLock()
	session := m.sessions[sessionID]
	m.mu.RUnlock()

	if session == nil {
		return ErrSessionNotFound
	}

	// session.AddTask adds the task to pendingTasks
	session.AddTask(protocol.Task{
		ID:        task.ID, // Include the ID field
		TaskID:    task.ID,
		Type:      task.Type,
		Payload:   task.Arguments,
		Arguments: task.Arguments,
		Command:   task.Command,
		SentTime:  task.CreatedAt,
		CreatedAt: task.CreatedAt,
		SessionID: task.SessionID,
		Status:    task.Status,
	})

	// Adding to the Tasks field should be moved to the session.AddTask method,
	// but for now, we manage it here (recommended refactoring in the future)
	session.mutex.Lock()
	session.Tasks = append(session.Tasks, task)
	session.mutex.Unlock()

	return nil
}

// GetSessionsByBeacon returns all sessions for a specific beacon
func (m *Manager) GetSessionsByBeacon(beaconID string) []*Session {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var sessions []*Session
	for _, session := range m.sessions {
		if session.BeaconID == beaconID {
			sessions = append(sessions, session)
		}
	}
	return sessions
}

// Session errors
var (
	ErrSessionNotFound = fmt.Errorf("session not found")
)
