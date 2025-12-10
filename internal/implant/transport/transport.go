package transport

import "github.com/r74tech/virga/internal/shared/protocol"

// Transport is the interface for C2 communication
type Transport interface {
	// SendBeacon sends a beacon and returns a response
	SendBeacon(msg *protocol.AgentMessage) (*protocol.AgentResponse, error)

	// Close closes the transport
	Close() error
}
