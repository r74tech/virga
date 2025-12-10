package sse

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/ThinkInAIXYZ/go-mcp/protocol"
	"github.com/ThinkInAIXYZ/go-mcp/server"
	"github.com/ThinkInAIXYZ/go-mcp/transport"
	"github.com/r74tech/virga/internal/server/beacons"
	"github.com/r74tech/virga/internal/server/database"
	"github.com/r74tech/virga/internal/server/mcp/config"
	"github.com/r74tech/virga/internal/server/session"
)

// Server represents the SSE MCP server
type Server struct {
	mcpServer      *server.Server
	transport      transport.ServerTransport
	beaconManager  *beacons.Manager
	sessionManager *session.Manager
	db             *database.Database
	config         *config.Config
	mu             sync.RWMutex
	isRunning      bool
	startTime      time.Time
}

// NewServer creates a new SSE MCP server
func NewServer(cfg *config.Config, beaconMgr *beacons.Manager, sessionMgr *session.Manager, db *database.Database) (*Server, error) {
	// Create SSE transport
	sseTransport, err := transport.NewSSEServerTransport(
		cfg.SSEPort,
		transport.WithSSEServerTransportOptionSSEPath(cfg.SSEBasePath+"/events"),
		transport.WithSSEServerTransportOptionMessagePath(cfg.SSEBasePath+"/message"),
		transport.WithSSEServerTransportOptionURLPrefix(cfg.SSERemoteURL),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create SSE transport: %w", err)
	}

	// Create server info
	serverInfo := protocol.Implementation{
		Name:    cfg.Name,
		Version: cfg.Version,
	}

	// Create capabilities
	capabilities := protocol.ServerCapabilities{
		Tools: &protocol.ToolsCapability{
			ListChanged: true,
		},
		Resources: &protocol.ResourcesCapability{
			Subscribe:   true,
			ListChanged: true,
		},
		Prompts: &protocol.PromptsCapability{
			ListChanged: true,
		},
	}

	// Create MCP server
	mcpServer, err := server.NewServer(
		sseTransport,
		server.WithServerInfo(serverInfo),
		server.WithCapabilities(capabilities),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create MCP server: %w", err)
	}

	srv := &Server{
		mcpServer:      mcpServer,
		transport:      sseTransport,
		beaconManager:  beaconMgr,
		sessionManager: sessionMgr,
		db:             db,
		config:         cfg,
	}

	// Register tools
	if err := srv.registerTools(); err != nil {
		return nil, fmt.Errorf("failed to register tools: %w", err)
	}

	// Register resources
	if err := srv.registerResources(); err != nil {
		return nil, fmt.Errorf("failed to register resources: %w", err)
	}

	return srv, nil
}

// Start starts the SSE MCP server
func (s *Server) Start(ctx context.Context) error {
	s.mu.Lock()

	if s.isRunning {
		s.mu.Unlock()
		return fmt.Errorf("SSE server is already running")
	}

	s.isRunning = true
	s.startTime = time.Now()
	s.mu.Unlock()

	log.Printf("Starting SSE server on %s", s.config.SSEPort)
	log.Printf("SSE endpoints: %s/events, %s/message", s.config.SSEBasePath, s.config.SSEBasePath)

	go func() {
		if err := s.mcpServer.Run(); err != nil {
			log.Printf("SSE server error: %v", err)
		}
		s.mu.Lock()
		s.isRunning = false
		s.mu.Unlock()
	}()

	return nil
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.isRunning {
		return nil
	}

	if err := s.mcpServer.Shutdown(ctx); err != nil {
		return fmt.Errorf("failed to shutdown SSE server: %w", err)
	}

	s.isRunning = false
	return nil
}

// IsRunning returns whether the server is running
func (s *Server) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.isRunning
}

// Transport returns the transport type
func (s *Server) Transport() config.Transport {
	return config.TransportSSE
}
