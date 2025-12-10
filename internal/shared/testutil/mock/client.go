package mock

import (
	"github.com/r74tech/virga/internal/cli/client"
	"github.com/r74tech/virga/internal/cli/config"
	"github.com/r74tech/virga/internal/shared/protocol"
)

// Compile-time check to ensure MockAPIClient implements client.APIClientInterface
var _ client.APIClientInterface = (*MockAPIClient)(nil)

// MockAPIClient implements a mock version of the APIClientInterface for testing
type MockAPIClient struct {
	// Function fields for customizing behavior
	GetSessionsFunc         func() ([]client.SessionInfo, error)
	SendSessionCommandFunc  func(sessionID, cmdType string, payload map[string]interface{}) (string, error)
	GetTaskResultFunc       func(sessionID, taskID string) (*protocol.TaskResult, error)
	GetTaskResultsFunc      func(sessionID string) ([]protocol.TaskResult, error)
	SetSessionInteractiveFn func(sessionID string, interactive bool) error
	AuthenticateFunc        func() error
	SetDebugModeFunc        func(debug bool)
	GeneratePayloadFunc     func(payloadType, listenerName string) ([]byte, string, error)
	GetListenersFunc        func() ([]map[string]interface{}, error)
	GetServerURLFunc        func() string

	// Fields to match the real APIClient
	Config    *config.Config
	AuthToken string
	Debug     bool
}

// GetSessions returns mock session information
func (m *MockAPIClient) GetSessions() ([]client.SessionInfo, error) {
	if m.GetSessionsFunc != nil {
		return m.GetSessionsFunc()
	}
	return []client.SessionInfo{}, nil
}

// SendSessionCommand sends a mock command to a session
func (m *MockAPIClient) SendSessionCommand(sessionID, cmdType string, payload map[string]interface{}) (string, error) {
	if m.SendSessionCommandFunc != nil {
		return m.SendSessionCommandFunc(sessionID, cmdType, payload)
	}
	return "mock-task-id", nil
}

// GetTaskResult retrieves a mock task result
func (m *MockAPIClient) GetTaskResult(sessionID, taskID string) (*protocol.TaskResult, error) {
	if m.GetTaskResultFunc != nil {
		return m.GetTaskResultFunc(sessionID, taskID)
	}
	return &protocol.TaskResult{
		TaskID:   taskID,
		Output:   "mock output",
		ExitCode: 0,
	}, nil
}

// GetTaskResults retrieves all task results for a session
func (m *MockAPIClient) GetTaskResults(sessionID string) ([]protocol.TaskResult, error) {
	if m.GetTaskResultsFunc != nil {
		return m.GetTaskResultsFunc(sessionID)
	}
	return []protocol.TaskResult{}, nil
}

// SetSessionInteractive sets the interactive mode for a session
func (m *MockAPIClient) SetSessionInteractive(sessionID string, interactive bool) error {
	if m.SetSessionInteractiveFn != nil {
		return m.SetSessionInteractiveFn(sessionID, interactive)
	}
	return nil
}

// Authenticate performs mock authentication
func (m *MockAPIClient) Authenticate() error {
	if m.AuthenticateFunc != nil {
		return m.AuthenticateFunc()
	}
	m.AuthToken = "mock-token"
	return nil
}

// SetDebugMode sets the debug mode
func (m *MockAPIClient) SetDebugMode(debug bool) {
	m.Debug = debug
	if m.SetDebugModeFunc != nil {
		m.SetDebugModeFunc(debug)
	}
}

// GeneratePayload generates a mock payload
func (m *MockAPIClient) GeneratePayload(payloadType, listenerName string) ([]byte, string, error) {
	if m.GeneratePayloadFunc != nil {
		return m.GeneratePayloadFunc(payloadType, listenerName)
	}
	return []byte("mock payload data"), "payload." + payloadType, nil
}

// GetListeners returns mock listener information
func (m *MockAPIClient) GetListeners() ([]map[string]interface{}, error) {
	if m.GetListenersFunc != nil {
		return m.GetListenersFunc()
	}
	return []map[string]interface{}{
		{
			"name":   "default-listener",
			"type":   "http",
			"port":   8080,
			"status": "running",
		},
	}, nil
}

// GetServerURL returns the mock server URL
func (m *MockAPIClient) GetServerURL() string {
	if m.GetServerURLFunc != nil {
		return m.GetServerURLFunc()
	}
	return "http://localhost:8080"
}

// NewMockAPIClient creates a new mock API client with default behavior
func NewMockAPIClient() *MockAPIClient {
	return &MockAPIClient{
		Config: &config.Config{
			Server: config.ServerConfig{
				Host: "localhost",
				Port: 8080,
			},
		},
		Debug: false,
	}
}
