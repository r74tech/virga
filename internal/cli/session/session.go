package session

import (
	"fmt"
	"sync"
	"time"

	"github.com/r74tech/virga/internal/cli/client"
	"github.com/r74tech/virga/internal/shared/logger"
	"github.com/r74tech/virga/internal/shared/protocol"
)

// Session is the session information for the CLI.
type Session struct {
	ID            string
	RemoteAddress string
	Information   map[string]interface{}
	LastActivity  time.Time
	firstSeen     time.Time
	apiClient     client.APIClientInterface
	isInteractive bool
}

// Manager is the CLI session manager.
type Manager struct {
	sessions       map[string]*Session
	currentSession *Session
	mutex          sync.RWMutex
	apiClient      client.APIClientInterface
	debugMode      bool
}

// NewManager creates a new session manager.
func NewManager() *Manager {
	return &Manager{
		sessions:  make(map[string]*Session),
		debugMode: false,
	}
}

// SetAPIClient sets the API client.
func (m *Manager) SetAPIClient(client client.APIClientInterface) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.apiClient = client
	// Apply to existing sessions
	for _, s := range m.sessions {
		s.apiClient = client
	}
}

// GetAPIClient gets the API client.
func (m *Manager) GetAPIClient() client.APIClientInterface {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.apiClient
}

// SetDebugMode enables/disables debug mode.
func (m *Manager) SetDebugMode(debug bool) {
	m.debugMode = debug
	if debug {
		logger.Debug("Session Manager debug mode enabled")
	}
}

// SyncWithServer synchronizes session information from the server.
func (m *Manager) SyncWithServer() error {
	if m.apiClient == nil {
		if m.debugMode {
			logger.Debug("API client not set, skipping sync")
		}
		return fmt.Errorf("API client not set")
	}

	// Get session information from the server
	apiSessions, err := m.apiClient.GetSessions()
	if err != nil {
		if m.debugMode {
			logger.Debug("Failed to get sessions from server: %v", err)
		}
		return err
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.debugMode {
		logger.Debug("Syncing %d sessions from server", len(apiSessions))
	}

	// Record the current session ID
	currentSessionID := ""
	if m.currentSession != nil {
		currentSessionID = m.currentSession.ID
	}

	// Create a new session map
	newSessions := make(map[string]*Session)

	// Add sessions from the server
	for _, apiSession := range apiSessions {
		// Set basic information
		info := make(map[string]interface{})
		if apiSession.Environment != nil {
			info = apiSession.Environment
		}

		// Add basic fields
		info["hostname"] = apiSession.Hostname
		info["username"] = apiSession.Username
		info["os"] = apiSession.OS

		// Inherit first seen time and interactive flag from existing sessions, or create new ones
		var firstSeen time.Time
		var isInteractive bool
		if existingSession, exists := m.sessions[apiSession.ID]; exists {
			firstSeen = existingSession.firstSeen
			isInteractive = existingSession.isInteractive
		} else {
			firstSeen = apiSession.LastActivity // For new sessions, use last activity as first seen time
		}

		// Create session
		session := &Session{
			ID:            apiSession.ID,
			RemoteAddress: apiSession.RemoteAddr,
			Information:   info,
			LastActivity:  apiSession.LastActivity,
			firstSeen:     firstSeen,
			apiClient:     m.apiClient, // Set API client
			isInteractive: isInteractive,
		}

		newSessions[apiSession.ID] = session

		if m.debugMode {
			logger.Debug("Synced session: %s (%s)", session.ID, session.RemoteAddress)
		}
	}

	// Update sessions
	m.sessions = newSessions

	// Restore current session (if it exists)
	m.currentSession = nil
	if currentSessionID != "" {
		if session, exists := m.sessions[currentSessionID]; exists {
			m.currentSession = session
			if m.debugMode {
				logger.Debug("Restored current session: %s", currentSessionID)
			}
		} else if m.debugMode {
			logger.Debug("Previous current session %s no longer exists", currentSessionID)
		}
	}

	return nil
}

// AddSession adds a session.
func (m *Manager) AddSession(id, remoteAddr string, info map[string]interface{}) *Session {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	now := time.Now()
	session := &Session{
		ID:            id,
		RemoteAddress: remoteAddr,
		Information:   info,
		LastActivity:  now,
		firstSeen:     now,         // Set first seen time
		apiClient:     m.apiClient, // Set API client
		isInteractive: false,
	}

	m.sessions[id] = session
	return session
}

// GetSession gets a session by ID.
func (m *Manager) GetSession(id string) (*Session, bool) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	session, found := m.sessions[id]
	return session, found
}

// RemoveSession removes a session.
func (m *Manager) RemoveSession(id string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.currentSession != nil && m.currentSession.ID == id {
		m.currentSession = nil
	}

	delete(m.sessions, id)
}

// GetAllSessions gets all sessions.
func (m *Manager) GetAllSessions() []*Session {
	// Sync with server if API client is available
	// Note: SyncWithServer already handles its own locking
	if m.apiClient != nil {
		if err := m.SyncWithServer(); err != nil {
			// Log error but continue with cached data
			if m.debugMode {
				logger.Debug("Sync failed, using cached data: %v", err)
			}
		}
	}

	m.mutex.RLock()
	defer m.mutex.RUnlock()

	// Create a copy of sessions to avoid data races
	sessions := make([]*Session, 0, len(m.sessions))
	for _, session := range m.sessions {
		// Create a shallow copy of the session to prevent external modifications
		sessionCopy := *session
		sessions = append(sessions, &sessionCopy)
	}

	return sessions
}

// SetCurrentSession sets the current session.
func (m *Manager) SetCurrentSession(id string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	session, found := m.sessions[id]
	if !found {
		return fmt.Errorf("session %s not found", id)
	}

	// If there is a previous session, clear the interactive mode
	if m.currentSession != nil && m.currentSession.ID != id {
		m.currentSession.isInteractive = false
		// Notify the server side to clear the interactive mode
		if m.apiClient != nil {
			m.apiClient.SetSessionInteractive(m.currentSession.ID, false)
		}
	}

	// Set the new session to interactive mode
	session.isInteractive = true
	m.currentSession = session

	// Notify the server side to set the interactive mode
	if m.apiClient != nil {
		m.apiClient.SetSessionInteractive(id, true)
	}

	return nil
}

// GetCurrentSession gets the current session.
func (m *Manager) GetCurrentSession() *Session {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	return m.currentSession
}

// ClearCurrentSession clears the current session selection.
func (m *Manager) ClearCurrentSession() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.currentSession != nil {
		// Clear the interactive mode
		m.currentSession.isInteractive = false
		// Notify the server side
		if m.apiClient != nil {
			m.apiClient.SetSessionInteractive(m.currentSession.ID, false)
		}
	}

	m.currentSession = nil
}

// UpdateSessionInfo updates the session information.
func (m *Manager) UpdateSessionInfo(id string, info map[string]interface{}) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	session, found := m.sessions[id]
	if !found {
		return fmt.Errorf("session %s not found", id)
	}

	// Update the information
	for k, v := range info {
		session.Information[k] = v
	}

	session.LastActivity = time.Now()
	return nil
}

// KillSession kills a session.
func (m *Manager) KillSession(id string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	_, found := m.sessions[id]
	if !found {
		return fmt.Errorf("session %s not found", id)
	}

	// If the current session is killed, clear it
	if m.currentSession != nil && m.currentSession.ID == id {
		m.currentSession = nil
	}

	// Session termination processing
	// In actual implementation, send a request to the server
	if m.apiClient != nil {
		// Clear the interactive mode
		m.apiClient.SetSessionInteractive(id, false)
	}

	delete(m.sessions, id)
	return nil
}

// GetInformation returns the session information.
func (s *Session) GetInformation() map[string]interface{} {
	return s.Information
}

// LastSeen returns the last activity time.
func (s *Session) LastSeen() time.Time {
	return s.LastActivity
}

// RemoteAddr returns the remote address.
func (s *Session) RemoteAddr() string {
	return s.RemoteAddress
}

// FirstSeen returns the first seen time.
func (s *Session) FirstSeen() time.Time {
	return s.firstSeen
}

// IsInteractive returns whether the session is interactive.
func (s *Session) IsInteractive() bool {
	return s.isInteractive
}

// SetInteractive sets the interactive mode.
func (s *Session) SetInteractive(interactive bool) {
	s.isInteractive = interactive
}

// ExecuteCommand executes a command on the agent.
func (s *Session) ExecuteCommand(cmdType string, payload map[string]interface{}) (string, error) {
	if s.apiClient == nil {
		return "", fmt.Errorf("API client not initialized")
	}

	// Send the command using the API client
	taskID, err := s.apiClient.SendSessionCommand(s.ID, cmdType, payload)
	if err != nil {
		return "", fmt.Errorf("failed to send command: %w", err)
	}

	return taskID, nil
}

// GetTaskResult gets the result of a specific task.
func (s *Session) GetTaskResult(taskID string) (*protocol.TaskResult, error) {
	if s.apiClient == nil {
		return nil, fmt.Errorf("API client not initialized")
	}

	// Get the task result using the API client
	result, err := s.apiClient.GetTaskResult(s.ID, taskID)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// GetTaskResults gets all task results for the session.
func (s *Session) GetTaskResults() ([]protocol.TaskResult, error) {
	if s.apiClient == nil {
		return nil, fmt.Errorf("API client not initialized")
	}

	// Get all task results using the API client
	results, err := s.apiClient.GetTaskResults(s.ID)
	if err != nil {
		return nil, err
	}

	return results, nil
}

// ListSessions is an alias for GetAllSessions for compatibility
func (m *Manager) ListSessions() ([]*Session, error) {
	sessions := m.GetAllSessions()
	return sessions, nil
}
