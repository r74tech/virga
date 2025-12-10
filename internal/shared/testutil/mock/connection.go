package mock

import (
	"sync"
	"time"
)

// MockAgentConnection simulates an agent connection for testing
// Implements the protocol.AgentConnection interface:
// - Send(data []byte) error
// - RemoteAddr() string
// - LastActivity() time.Time
// - Close() error
type MockAgentConnection struct {
	sentData     [][]byte
	closed       bool
	remoteAddr   string
	lastActivity time.Time
	mu           sync.Mutex

	// Function fields for customizing behavior
	SendFunc         func(data []byte) error
	RemoteAddrFunc   func() string
	LastActivityFunc func() time.Time
	CloseFunc        func() error
}

// NewMockAgentConnection creates a new mock agent connection with default behavior
func NewMockAgentConnection() *MockAgentConnection {
	return &MockAgentConnection{
		remoteAddr:   "127.0.0.1:12345",
		lastActivity: time.Now(),
		sentData:     make([][]byte, 0),
	}
}

// Send simulates sending data to the agent
func (m *MockAgentConnection) Send(data []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Use custom function if provided
	if m.SendFunc != nil {
		return m.SendFunc(data)
	}

	// Default behavior
	m.sentData = append(m.sentData, data)
	m.lastActivity = time.Now()
	return nil
}

// RemoteAddr returns the remote address of the connection
func (m *MockAgentConnection) RemoteAddr() string {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Use custom function if provided
	if m.RemoteAddrFunc != nil {
		return m.RemoteAddrFunc()
	}

	// Default behavior
	return m.remoteAddr
}

// LastActivity returns the last activity time of the connection
func (m *MockAgentConnection) LastActivity() time.Time {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Use custom function if provided
	if m.LastActivityFunc != nil {
		return m.LastActivityFunc()
	}

	// Default behavior
	return m.lastActivity
}

// Close closes the connection
func (m *MockAgentConnection) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Use custom function if provided
	if m.CloseFunc != nil {
		return m.CloseFunc()
	}

	// Default behavior
	m.closed = true
	return nil
}

// Test helper methods

// GetSentData returns all data sent through this connection
func (m *MockAgentConnection) GetSentData() [][]byte {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Return a copy to prevent concurrent modification
	dataCopy := make([][]byte, len(m.sentData))
	copy(dataCopy, m.sentData)
	return dataCopy
}

// SetRemoteAddr sets the remote address
func (m *MockAgentConnection) SetRemoteAddr(addr string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.remoteAddr = addr
}

// UpdateLastActivity updates the last activity time to now
func (m *MockAgentConnection) UpdateLastActivity() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.lastActivity = time.Now()
}

// IsClosed returns whether the connection has been closed
func (m *MockAgentConnection) IsClosed() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.closed
}

// ClearSentData clears all sent data
func (m *MockAgentConnection) ClearSentData() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sentData = make([][]byte, 0)
}

// GetSentDataCount returns the number of data packets sent
func (m *MockAgentConnection) GetSentDataCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.sentData)
}

// GetLastSentData returns the last data sent, or nil if none
func (m *MockAgentConnection) GetLastSentData() []byte {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.sentData) == 0 {
		return nil
	}

	// Return a copy of the last sent data
	lastData := m.sentData[len(m.sentData)-1]
	dataCopy := make([]byte, len(lastData))
	copy(dataCopy, lastData)
	return dataCopy
}
