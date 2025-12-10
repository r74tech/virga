package mcp

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/r74tech/virga/internal/server/beacons"
	"github.com/r74tech/virga/internal/server/database"
	"github.com/r74tech/virga/internal/server/mcp/config"
	"github.com/r74tech/virga/internal/server/mcp/sse"
	"github.com/r74tech/virga/internal/server/mcp/stdio"
	"github.com/r74tech/virga/internal/server/mcp/streamable"
	"github.com/r74tech/virga/internal/server/session"
)

// Manager manages multiple MCP server instances
type Manager struct {
	servers        map[config.Transport]Server
	config         *config.Config
	beaconManager  *beacons.Manager
	sessionManager *session.Manager
	db             *database.Database
	mu             sync.RWMutex
}

// NewManager creates a new MCP manager
func NewManager(cfg *config.Config, beaconMgr *beacons.Manager, sessionMgr *session.Manager, db *database.Database) (*Manager, error) {
	m := &Manager{
		servers:        make(map[config.Transport]Server),
		config:         cfg,
		beaconManager:  beaconMgr,
		sessionManager: sessionMgr,
		db:             db,
	}

	// Initialize enabled servers
	if cfg.StdioEnabled {
		server, err := stdio.NewServer(cfg, beaconMgr, sessionMgr, db)
		if err != nil {
			return nil, fmt.Errorf("failed to create stdio server: %w", err)
		}
		m.servers[config.TransportStdio] = server
	}

	if cfg.SSEEnabled {
		server, err := sse.NewServer(cfg, beaconMgr, sessionMgr, db)
		if err != nil {
			return nil, fmt.Errorf("failed to create SSE server: %w", err)
		}
		m.servers[config.TransportSSE] = server
	}

	if cfg.StreamableEnabled {
		streamableCfg := &streamable.Config{
			Name:    cfg.Name,
			Version: cfg.Version,
			Port:    cfg.StreamablePort,
		}
		server, err := streamable.NewStreamableServer(streamableCfg, beaconMgr, sessionMgr, db)
		if err != nil {
			return nil, fmt.Errorf("failed to create streamable server: %w", err)
		}
		m.servers[config.TransportStreamable] = server
	}

	return m, nil
}

// StartAll starts all enabled MCP servers
func (m *Manager) StartAll(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if context is already cancelled
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	for transport, server := range m.servers {
		log.Printf("Starting MCP %s server...", transport)
		go func(t config.Transport, s Server) {
			if err := s.Start(ctx); err != nil {
				log.Printf("MCP %s server error: %v", t, err)
			}
		}(transport, server)
	}

	return nil
}

// Shutdown gracefully shuts down all MCP servers
func (m *Manager) Shutdown(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var wg sync.WaitGroup
	errc := make(chan error, len(m.servers))

	for transport, server := range m.servers {
		if server.IsRunning() {
			wg.Add(1)
			go func(t config.Transport, s Server) {
				defer wg.Done()
				log.Printf("Shutting down MCP %s server...", t)
				if err := s.Shutdown(ctx); err != nil {
					errc <- fmt.Errorf("%s: %w", t, err)
				}
			}(transport, server)
		} else {
			wg.Add(1)
			go func() {
				defer wg.Done()
			}()
		}
	}

	go func() {
		wg.Wait()
		close(errc)
	}()

	var errs []error
	for err := range errc {
		if err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("shutdown errors: %v", errs)
	}

	return nil
}

// GetServer returns a specific server by transport type
func (m *Manager) GetServer(transport config.Transport) (Server, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	server, ok := m.servers[transport]
	return server, ok
}

// IsAnyRunning returns true if any server is running
func (m *Manager) IsAnyRunning() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, server := range m.servers {
		if server.IsRunning() {
			return true
		}
	}
	return false
}

// EnabledTransports returns a list of enabled transports
func (m *Manager) EnabledTransports() []config.Transport {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var transports []config.Transport
	for t := range m.servers {
		transports = append(transports, t)
	}
	return transports
}
