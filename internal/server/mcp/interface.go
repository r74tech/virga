package mcp

import (
	"context"

	"github.com/r74tech/virga/internal/server/mcp/config"
)

// Server defines the common interface for all MCP servers
type Server interface {
	// Start starts the MCP server
	Start(ctx context.Context) error

	// Shutdown gracefully shuts down the server
	Shutdown(ctx context.Context) error

	// IsRunning returns whether the server is running
	IsRunning() bool

	// Transport returns the transport type
	Transport() config.Transport
}
