package streamable

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/ThinkInAIXYZ/go-mcp/protocol"
	"github.com/ThinkInAIXYZ/go-mcp/server"
	"github.com/ThinkInAIXYZ/go-mcp/transport"
	"github.com/r74tech/virga/internal/server/beacons"
	"github.com/r74tech/virga/internal/server/database"
	"github.com/r74tech/virga/internal/server/mcp/config"
	"github.com/r74tech/virga/internal/server/session"
)

// StreamableServer represents the streamable HTTP MCP server
type StreamableServer struct {
	server         *server.Server
	transport      transport.ServerTransport
	beaconManager  *beacons.Manager
	sessionManager *session.Manager
	db             *database.Database
	mu             sync.RWMutex
	isRunning      bool
}

// Config for streamable server
type Config struct {
	Name    string
	Version string
	Port    string
}

// NewStreamableServer creates a new streamable HTTP MCP server
func NewStreamableServer(cfg *Config, beaconMgr *beacons.Manager, sessionMgr *session.Manager, db *database.Database) (*StreamableServer, error) {
	// Create Streamable HTTP transport (same as dice-server)
	transportServer := transport.NewStreamableHTTPServerTransport(
		cfg.Port,
		transport.WithStreamableHTTPServerTransportOptionEndpoint("/mcp"),
		transport.WithStreamableHTTPServerTransportOptionStateMode(transport.Stateful),
	)

	// Create MCP server (same as dice-server)
	mcpServer, err := server.NewServer(
		transportServer,
		server.WithServerInfo(protocol.Implementation{
			Name:    cfg.Name,
			Version: cfg.Version,
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create MCP server: %w", err)
	}

	srv := &StreamableServer{
		server:         mcpServer,
		transport:      transportServer,
		beaconManager:  beaconMgr,
		sessionManager: sessionMgr,
		db:             db,
	}

	// Register tools
	if err := RegisterTools(mcpServer, sessionMgr, beaconMgr, db); err != nil {
		return nil, fmt.Errorf("failed to register tools: %w", err)
	}

	// Register resources
	if err := RegisterResources(mcpServer, sessionMgr, beaconMgr, db); err != nil {
		return nil, fmt.Errorf("failed to register resources: %w", err)
	}

	return srv, nil
}

// Start starts the streamable server
func (s *StreamableServer) Start(ctx context.Context) error {
	s.mu.Lock()

	if s.isRunning {
		s.mu.Unlock()
		return fmt.Errorf("streamable server is already running")
	}

	s.isRunning = true
	s.mu.Unlock()

	// Output debug information
	fmt.Fprintf(os.Stderr, "Starting Streamable HTTP MCP server...\n")

	go func() {
		if err := s.server.Run(); err != nil {
			log.Printf("streamable server error: %v", err)
		}
		s.mu.Lock()
		s.isRunning = false
		s.mu.Unlock()
	}()

	return nil
}

// Shutdown stops the streamable server
func (s *StreamableServer) Shutdown(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.isRunning {
		return nil
	}

	log.Printf("Shutting down streamable HTTP server...")
	if err := s.server.Shutdown(ctx); err != nil {
		return fmt.Errorf("failed to shutdown streamable server: %w", err)
	}

	s.isRunning = false
	return nil
}

// IsRunning returns whether the streamable server is running
func (s *StreamableServer) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.isRunning
}

// Transport returns the transport type
func (s *StreamableServer) Transport() config.Transport {
	return config.TransportStreamable
}
