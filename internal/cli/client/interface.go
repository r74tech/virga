package client

import (
	"github.com/r74tech/virga/internal/shared/protocol"
)

// APIClientInterface defines the interface for API client operations
type APIClientInterface interface {
	// Authentication
	Authenticate() error

	// Session management
	GetSessions() ([]SessionInfo, error)
	SetSessionInteractive(sessionID string, interactive bool) error

	// Task management
	SendSessionCommand(sessionID, cmdType string, payload map[string]interface{}) (string, error)
	GetTaskResult(sessionID, taskID string) (*protocol.TaskResult, error)
	GetTaskResults(sessionID string) ([]protocol.TaskResult, error)

	// Payload generation
	GeneratePayload(payloadType, listenerName string) ([]byte, string, error)

	// Listener management
	GetListeners() ([]map[string]interface{}, error)

	// Configuration
	SetDebugMode(debug bool)
	GetServerURL() string
}
