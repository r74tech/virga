package sse

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/ThinkInAIXYZ/go-mcp/protocol"
)

// registerResources registers all resources for the SSE server
func (s *Server) registerResources() error {
	// Sessions list resource
	sessionsResource := &protocol.Resource{
		URI:         "sessions://list",
		Name:        "Sessions List",
		Description: "List of all active sessions",
		MimeType:    "application/json",
	}
	s.mcpServer.RegisterResource(sessionsResource, s.getSessionsList)

	// Session detail resource template
	sessionDetailTemplate := &protocol.ResourceTemplate{
		URITemplate: "sessions://detail/{session_id}",
		Name:        "Session Details",
		Description: "Detailed information about a specific session",
		MimeType:    "application/json",
	}
	if err := s.mcpServer.RegisterResourceTemplate(sessionDetailTemplate, s.getSessionDetail); err != nil {
		return fmt.Errorf("failed to register session detail template: %w", err)
	}

	// Beacons list resource
	beaconsResource := &protocol.Resource{
		URI:         "beacons://list",
		Name:        "Beacons List",
		Description: "List of all configured beacons",
		MimeType:    "application/json",
	}
	s.mcpServer.RegisterResource(beaconsResource, s.getBeaconsList)

	// Beacon detail resource template
	beaconDetailTemplate := &protocol.ResourceTemplate{
		URITemplate: "beacons://detail/{beacon_id}",
		Name:        "Beacon Details",
		Description: "Detailed information about a specific beacon",
		MimeType:    "application/json",
	}
	if err := s.mcpServer.RegisterResourceTemplate(beaconDetailTemplate, s.getBeaconDetail); err != nil {
		return fmt.Errorf("failed to register beacon detail template: %w", err)
	}

	// System status resource
	systemStatusResource := &protocol.Resource{
		URI:         "system://status",
		Name:        "System Status",
		Description: "Current system status and statistics",
		MimeType:    "application/json",
	}
	s.mcpServer.RegisterResource(systemStatusResource, s.getSystemStatus)

	// Command history resource template
	commandHistoryTemplate := &protocol.ResourceTemplate{
		URITemplate: "history://commands/{session_id}",
		Name:        "Command History",
		Description: "Command history for a specific session",
		MimeType:    "application/json",
	}
	if err := s.mcpServer.RegisterResourceTemplate(commandHistoryTemplate, s.getCommandHistory); err != nil {
		return fmt.Errorf("failed to register command history template: %w", err)
	}

	// Session files resource template
	sessionFilesTemplate := &protocol.ResourceTemplate{
		URITemplate: "files://session/{session_id}",
		Name:        "Session Files",
		Description: "List of files uploaded/downloaded for a session",
		MimeType:    "application/json",
	}
	if err := s.mcpServer.RegisterResourceTemplate(sessionFilesTemplate, s.getSessionFiles); err != nil {
		return fmt.Errorf("failed to register session files template: %w", err)
	}

	return nil
}

// Resource handler implementations

func (s *Server) getSessionsList(ctx context.Context, request *protocol.ReadResourceRequest) (*protocol.ReadResourceResult, error) {
	sessions := s.sessionManager.GetActiveSessions()

	var sessionList []map[string]interface{}
	for _, session := range sessions {
		sessionInfo := map[string]interface{}{
			"id":         session.ID,
			"hostname":   session.Hostname,
			"username":   session.Username,
			"os":         session.OS,
			"arch":       session.Arch,
			"ip":         session.IP,
			"beacon_id":  session.BeaconID,
			"status":     session.Status,
			"last_seen":  session.LastSeen().Format(time.RFC3339),
			"created_at": session.CreatedAt.Format(time.RFC3339),
		}
		sessionList = append(sessionList, sessionInfo)
	}

	content, err := json.MarshalIndent(sessionList, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal sessions list: %w", err)
	}

	return &protocol.ReadResourceResult{
		Contents: []protocol.ResourceContents{
			&protocol.TextResourceContents{
				URI:      "sessions://list",
				MimeType: "application/json",
				Text:     string(content),
			},
		},
	}, nil
}

func (s *Server) getSessionDetail(ctx context.Context, request *protocol.ReadResourceRequest) (*protocol.ReadResourceResult, error) {
	// Extract session_id from URI
	parts := strings.Split(request.URI, "/")
	if len(parts) < 4 || parts[2] != "detail" {
		return nil, fmt.Errorf("invalid URI format for session detail: %s", request.URI)
	}
	sessionID := parts[3]

	session := s.sessionManager.GetSession(sessionID)
	if session == nil {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}

	sessionDetail := map[string]interface{}{
		"id":            session.ID,
		"hostname":      session.Hostname,
		"username":      session.Username,
		"os":            session.OS,
		"arch":          session.Arch,
		"ip":            session.IP,
		"beacon_id":     session.BeaconID,
		"status":        session.Status,
		"last_seen":     session.LastSeen().Format(time.RFC3339),
		"created_at":    session.CreatedAt.Format(time.RFC3339),
		"updated_at":    session.UpdatedAt.Format(time.RFC3339),
		"metadata":      session.Metadata,
		"command_count": getCommandCount(s.db, sessionID),
	}

	content, err := json.MarshalIndent(sessionDetail, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal session detail: %w", err)
	}

	return &protocol.ReadResourceResult{
		Contents: []protocol.ResourceContents{
			&protocol.TextResourceContents{
				URI:      request.URI,
				MimeType: "application/json",
				Text:     string(content),
			},
		},
	}, nil
}

func (s *Server) getBeaconsList(ctx context.Context, request *protocol.ReadResourceRequest) (*protocol.ReadResourceResult, error) {
	beacons := s.beaconManager.ListBeacons()
	beaconList := []map[string]interface{}{}

	for _, beacon := range beacons {
		beaconInfo := map[string]interface{}{
			"id":         beacon.ID,
			"name":       beacon.Name,
			"type":       beacon.Type,
			"created_at": beacon.CreatedAt.Format(time.RFC3339),
		}
		beaconList = append(beaconList, beaconInfo)
	}

	content, err := json.MarshalIndent(beaconList, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal beacons list: %w", err)
	}

	return &protocol.ReadResourceResult{
		Contents: []protocol.ResourceContents{
			&protocol.TextResourceContents{
				URI:      "beacons://list",
				MimeType: "application/json",
				Text:     string(content),
			},
		},
	}, nil
}

func (s *Server) getBeaconDetail(ctx context.Context, request *protocol.ReadResourceRequest) (*protocol.ReadResourceResult, error) {
	// Extract beacon_id from URI
	parts := strings.Split(request.URI, "/")
	if len(parts) < 3 || parts[2] != "detail" {
		return nil, fmt.Errorf("invalid URI format: %s", request.URI)
	}
	beaconID := parts[3]

	beacon := s.beaconManager.GetBeacon(beaconID)
	if beacon == nil {
		return nil, fmt.Errorf("beacon not found: %s", beaconID)
	}

	beaconDetail := map[string]interface{}{
		"id":         beacon.ID,
		"name":       beacon.Name,
		"type":       beacon.Type,
		"created_at": beacon.CreatedAt.Format(time.RFC3339),
	}

	content, err := json.MarshalIndent(beaconDetail, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal beacon detail: %w", err)
	}

	return &protocol.ReadResourceResult{
		Contents: []protocol.ResourceContents{
			&protocol.TextResourceContents{
				URI:      request.URI,
				MimeType: "application/json",
				Text:     string(content),
			},
		},
	}, nil
}

func (s *Server) getSystemStatus(ctx context.Context, request *protocol.ReadResourceRequest) (*protocol.ReadResourceResult, error) {
	uptime := time.Since(s.startTime)

	status := map[string]interface{}{
		"server_type":     "SSE",
		"version":         s.config.Version,
		"uptime_seconds":  int(uptime.Seconds()),
		"uptime":          uptime.String(),
		"active_sessions": len(s.sessionManager.GetActiveSessions()),
		"total_beacons":   len(s.beaconManager.ListBeacons()),
		"database_stats":  getDatabaseStats(s.db),
		"server_time":     time.Now().Format(time.RFC3339),
	}

	content, err := json.MarshalIndent(status, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal system status: %w", err)
	}

	return &protocol.ReadResourceResult{
		Contents: []protocol.ResourceContents{
			&protocol.TextResourceContents{
				URI:      "system://status",
				MimeType: "application/json",
				Text:     string(content),
			},
		},
	}, nil
}

func (s *Server) getCommandHistory(ctx context.Context, request *protocol.ReadResourceRequest) (*protocol.ReadResourceResult, error) {
	// Extract session_id from URI
	parts := strings.Split(request.URI, "/")
	if len(parts) < 4 || parts[2] != "commands" {
		return nil, fmt.Errorf("invalid URI format: %s", request.URI)
	}
	sessionID := parts[3]
	if sessionID == "" {
		return nil, fmt.Errorf("missing session ID in URI: %s", request.URI)
	}

	// Retrieve command history from database
	history, err := s.db.GetCommandHistory(sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get command history: %w", err)
	}

	content, err := json.MarshalIndent(history, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal command history: %w", err)
	}

	return &protocol.ReadResourceResult{
		Contents: []protocol.ResourceContents{
			&protocol.TextResourceContents{
				URI:      request.URI,
				MimeType: "application/json",
				Text:     string(content),
			},
		},
	}, nil
}

func (s *Server) getSessionFiles(ctx context.Context, request *protocol.ReadResourceRequest) (*protocol.ReadResourceResult, error) {
	// Extract session_id from URI
	parts := strings.Split(request.URI, "/")
	if len(parts) < 4 || parts[2] != "session" {
		return nil, fmt.Errorf("invalid URI format: %s", request.URI)
	}
	sessionID := parts[3]
	if sessionID == "" {
		return nil, fmt.Errorf("missing session ID in URI: %s", request.URI)
	}

	// Retrieve file operations from database
	files, err := s.db.GetFileOperations(sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get file operations: %w", err)
	}

	content, err := json.MarshalIndent(files, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal session files: %w", err)
	}

	return &protocol.ReadResourceResult{
		Contents: []protocol.ResourceContents{
			&protocol.TextResourceContents{
				URI:      request.URI,
				MimeType: "application/json",
				Text:     string(content),
			},
		},
	}, nil
}

// Helper functions

func getCommandCount(_ interface{}, _ string) int {
	// TODO: Implement command count retrieval from database
	return 0
}

func getDatabaseStats(_ interface{}) map[string]interface{} {
	// TODO: Implement database statistics retrieval
	return map[string]interface{}{
		"total_commands": 0,
		"total_files":    0,
		"total_events":   0,
	}
}
