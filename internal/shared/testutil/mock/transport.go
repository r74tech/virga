package mock

import (
	"fmt"
	"sync"

	"github.com/r74tech/virga/internal/shared/protocol"
)

// Compile-time check to ensure MockTransport implements the interface
// Note: We can't import the transport package here due to import cycles,
// but the interface is simple enough to document
// MockTransport implements: SendBeacon(*protocol.AgentMessage) (*protocol.AgentResponse, error) and Close() error

// MockTransport implements a mock version of the transport.Transport interface for testing
type MockTransport struct {
	mu           sync.Mutex
	taskQueue    []protocol.Task
	taskResults  map[string]protocol.AgentTaskResult
	connected    bool
	registeredID string
	failNext     bool
	lastBeacon   *protocol.AgentMessage

	// Function fields for customizing behavior
	SendBeaconFunc func(msg *protocol.AgentMessage) (*protocol.AgentResponse, error)
	CloseFunc      func() error
}

// NewMockTransport creates a new mock transport with default behavior
func NewMockTransport() *MockTransport {
	return &MockTransport{
		taskResults: make(map[string]protocol.AgentTaskResult),
		connected:   true,
	}
}

// SendBeacon sends a beacon message and returns a response
func (m *MockTransport) SendBeacon(msg *protocol.AgentMessage) (*protocol.AgentResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Use custom function if provided
	if m.SendBeaconFunc != nil {
		return m.SendBeaconFunc(msg)
	}

	// Default behavior
	m.lastBeacon = msg

	if m.failNext {
		m.failNext = false
		return nil, fmt.Errorf("mock transport: send beacon failed")
	}

	// Store any task results from the beacon
	if msg.TaskResults != nil {
		for _, result := range msg.TaskResults {
			m.taskResults[result.TaskID] = result
		}
	}

	// Create response with any queued tasks
	response := &protocol.AgentResponse{
		Status: "success",
	}

	if len(m.taskQueue) > 0 {
		response.Tasks = m.taskQueue
		m.taskQueue = nil // Clear queue after sending
	}

	return response, nil
}

// Close closes the transport connection
func (m *MockTransport) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Use custom function if provided
	if m.CloseFunc != nil {
		return m.CloseFunc()
	}

	// Default behavior
	m.connected = false
	return nil
}

// Test helper methods

// AddTask adds a task to the queue for the next response
func (m *MockTransport) AddTask(task protocol.Task) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.taskQueue = append(m.taskQueue, task)
}

// GetTaskResult retrieves a task result by ID
func (m *MockTransport) GetTaskResult(taskID string) (protocol.AgentTaskResult, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	result, ok := m.taskResults[taskID]
	return result, ok
}

// SetFailNext causes the next SendBeacon call to fail
func (m *MockTransport) SetFailNext(fail bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.failNext = fail
}

// GetLastBeacon returns the last beacon message sent
func (m *MockTransport) GetLastBeacon() *protocol.AgentMessage {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.lastBeacon
}

// IsConnected returns the connection status
func (m *MockTransport) IsConnected() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.connected
}

// GetAllTaskResults returns all stored task results
func (m *MockTransport) GetAllTaskResults() map[string]protocol.AgentTaskResult {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Return a copy to prevent concurrent modification
	results := make(map[string]protocol.AgentTaskResult)
	for k, v := range m.taskResults {
		results[k] = v
	}
	return results
}

// ClearTaskResults clears all stored task results
func (m *MockTransport) ClearTaskResults() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.taskResults = make(map[string]protocol.AgentTaskResult)
}

// GetQueuedTasks returns the current task queue
func (m *MockTransport) GetQueuedTasks() []protocol.Task {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Return a copy to prevent concurrent modification
	tasks := make([]protocol.Task, len(m.taskQueue))
	copy(tasks, m.taskQueue)
	return tasks
}
