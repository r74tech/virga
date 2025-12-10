package stdio

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

// Server represents the STDIO MCP server
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

// NewServer creates a new STDIO MCP server
func NewServer(cfg *config.Config, beaconMgr *beacons.Manager, sessionMgr *session.Manager, db *database.Database) (*Server, error) {
	// Create STDIO transport
	stdioTransport := transport.NewStdioServerTransport()

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
		stdioTransport,
		server.WithServerInfo(serverInfo),
		server.WithCapabilities(capabilities),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create MCP server: %w", err)
	}

	srv := &Server{
		mcpServer:      mcpServer,
		transport:      stdioTransport,
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

// Start starts the STDIO MCP server
func (s *Server) Start(ctx context.Context) error {
	s.mu.Lock()

	if s.isRunning {
		s.mu.Unlock()
		return fmt.Errorf("STDIO server is already running")
	}

	s.isRunning = true
	s.startTime = time.Now()
	s.mu.Unlock()

	log.Printf("Starting STDIO MCP server (version %s)", s.config.Version)

	go func() {
		if err := s.mcpServer.Run(); err != nil {
			log.Printf("STDIO server error: %v", err)
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
		return fmt.Errorf("failed to shutdown STDIO server: %w", err)
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
	return config.TransportStdio
}
