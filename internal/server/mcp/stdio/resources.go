package stdio

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ThinkInAIXYZ/go-mcp/protocol"
)

// registerResources registers all resources for the SSE server
func (s *Server) registerResources() error {
	// Sessions list resource
	sessionsListResource := &protocol.Resource{
		URI:      "sessions-list",
		Name:     "Active Sessions",
		MimeType: "application/json",
	}
	s.mcpServer.RegisterResource(sessionsListResource, s.getSessionsListHandler)

	// Beacons list resource
	beaconsListResource := &protocol.Resource{
		URI:      "beacons-list",
		Name:     "Configured Beacons",
		MimeType: "application/json",
	}
	s.mcpServer.RegisterResource(beaconsListResource, s.getBeaconsListHandler)

	// System status resource
	systemStatusResource := &protocol.Resource{
		URI:      "system-status",
		Name:     "System Status",
		MimeType: "application/json",
	}
	s.mcpServer.RegisterResource(systemStatusResource, s.getSystemStatusHandler)

	// Note: Dynamic resources with parameters (session/{id}, beacon/{id}, etc.)
	// need to be handled differently in the new API
	// For now, we'll register these as static resources and handle the routing manually

	return nil
}

// Resource handler implementations

func (s *Server) getSessionsListHandler(ctx context.Context, req *protocol.ReadResourceRequest) (*protocol.ReadResourceResult, error) {
	sessions := s.sessionManager.GetActiveSessions()

	var sessionList []map[string]interface{}
	for _, session := range sessions {
		sessionInfo := map[string]interface{}{
			"id":        session.ID,
			"hostname":  session.Hostname,
			"username":  session.Username,
			"os":        session.OS,
			"arch":      session.Arch,
			"ip":        session.IP,
			"last_seen": session.LastSeen().Format(time.RFC3339),
			"status":    session.Status,
			"beacon_id": session.BeaconID,
		}
		sessionList = append(sessionList, sessionInfo)
	}

	data, err := json.MarshalIndent(sessionList, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal sessions list: %w", err)
	}

	return &protocol.ReadResourceResult{
		Contents: []protocol.ResourceContents{
			&protocol.TextResourceContents{
				URI:      "sessions-list",
				MimeType: "application/json",
				Text:     string(data),
			},
		},
	}, nil
}

func (s *Server) getBeaconsListHandler(ctx context.Context, req *protocol.ReadResourceRequest) (*protocol.ReadResourceResult, error) {
	beacons := s.beaconManager.ListBeacons()

	var beaconList []map[string]interface{}
	for _, beacon := range beacons {
		beaconInfo := map[string]interface{}{
			"id":          beacon.ID,
			"name":        beacon.Name,
			"type":        beacon.Type,
			"active":      beacon.Active,
			"created_at":  beacon.CreatedAt.Format(time.RFC3339),
			"last_active": beacon.LastActive.Format(time.RFC3339),
			"sessions":    len(s.sessionManager.GetSessionsByBeacon(beacon.ID)),
		}
		beaconList = append(beaconList, beaconInfo)
	}

	data, err := json.MarshalIndent(beaconList, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal beacons list: %w", err)
	}

	return &protocol.ReadResourceResult{
		Contents: []protocol.ResourceContents{
			&protocol.TextResourceContents{
				URI:      "beacons-list",
				MimeType: "application/json",
				Text:     string(data),
			},
		},
	}, nil
}

func (s *Server) getSystemStatusHandler(ctx context.Context, req *protocol.ReadResourceRequest) (*protocol.ReadResourceResult, error) {
	activeSessions := s.sessionManager.GetActiveSessions()
	activeBeacons := s.beaconManager.ListBeacons()

	var activeBeaconCount int
	for _, beacon := range activeBeacons {
		if beacon.Active {
			activeBeaconCount++
		}
	}

	status := map[string]interface{}{
		"timestamp":      time.Now().Format(time.RFC3339),
		"server_type":    "Stdio",
		"server_version": s.config.Version,
		"sessions": map[string]interface{}{
			"total":  len(activeSessions),
			"active": len(activeSessions),
		},
		"beacons": map[string]interface{}{
			"total":  len(activeBeacons),
			"active": activeBeaconCount,
		},
		"uptime": time.Since(s.startTime).String(),
	}

	// Add database stats if available
	dbStats := s.db.GetStats()
	if dbStats != nil {
		status["database"] = dbStats
	}

	data, err := json.MarshalIndent(status, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal system status: %w", err)
	}

	return &protocol.ReadResourceResult{
		Contents: []protocol.ResourceContents{
			&protocol.TextResourceContents{
				URI:      "system-status",
				MimeType: "application/json",
				Text:     string(data),
			},
		},
	}, nil
}

// Legacy methods for dynamic resources - these would need to be handled differently
// in the new API, possibly through tools or a different mechanism

func (s *Server) getSessionDetail(sessionID string) (map[string]interface{}, error) {
	session := s.sessionManager.GetSession(sessionID)
	if session == nil {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}

	// Get additional session details
	sessionInfo := map[string]interface{}{
		"id":         session.ID,
		"hostname":   session.Hostname,
		"username":   session.Username,
		"os":         session.OS,
		"arch":       session.Arch,
		"ip":         session.IP,
		"last_seen":  session.LastSeen().Format(time.RFC3339),
		"status":     session.Status,
		"beacon_id":  session.BeaconID,
		"created_at": session.CreatedAt.Format(time.RFC3339),
		"updated_at": session.UpdatedAt.Format(time.RFC3339),
		"metadata":   session.Metadata,
	}

	// Add command statistics if available
	if history, err := s.db.GetCommandHistory(sessionID); err == nil {
		sessionInfo["command_count"] = len(history)
		if len(history) > 0 {
			sessionInfo["last_command"] = history[0]
		}
	}

	return sessionInfo, nil
}

func (s *Server) getBeaconDetail(beaconID string) (map[string]interface{}, error) {
	beacon := s.beaconManager.GetBeacon(beaconID)
	if beacon == nil {
		return nil, fmt.Errorf("beacon not found: %s", beaconID)
	}

	// Get sessions for this beacon
	sessions := s.sessionManager.GetSessionsByBeacon(beaconID)
	sessionIDs := make([]string, len(sessions))
	for i, session := range sessions {
		sessionIDs[i] = session.ID
	}

	beaconInfo := map[string]interface{}{
		"id":            beacon.ID,
		"name":          beacon.Name,
		"type":          beacon.Type,
		"active":        beacon.Active,
		"config":        beacon.Config,
		"created_at":    beacon.CreatedAt.Format(time.RFC3339),
		"updated_at":    beacon.UpdatedAt.Format(time.RFC3339),
		"last_active":   beacon.LastActive.Format(time.RFC3339),
		"session_ids":   sessionIDs,
		"session_count": len(sessionIDs),
	}

	return beaconInfo, nil
}

func (s *Server) getCommandHistory(sessionID string) ([]map[string]interface{}, error) {
	session := s.sessionManager.GetSession(sessionID)
	if session == nil {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}

	// Get command history from database
	history, err := s.db.GetCommandHistory(sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get command history: %w", err)
	}

	return history, nil
}

func (s *Server) getSessionFiles(sessionID string) (map[string]interface{}, error) {
	session := s.sessionManager.GetSession(sessionID)
	if session == nil {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}

	// Get file operations history from database
	fileOps, err := s.db.GetFileOperations(sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get file operations: %w", err)
	}

	// Group by operation type
	grouped := map[string][]interface{}{
		"uploads":   {},
		"downloads": {},
	}

	for _, op := range fileOps {
		if opType, ok := op["type"].(string); ok {
			switch opType {
			case "upload":
				grouped["uploads"] = append(grouped["uploads"], op)
			case "download":
				grouped["downloads"] = append(grouped["downloads"], op)
			}
		}
	}

	result := map[string]interface{}{
		"session_id": sessionID,
		"operations": grouped,
		"summary": map[string]int{
			"total_uploads":   len(grouped["uploads"]),
			"total_downloads": len(grouped["downloads"]),
		},
	}

	return result, nil
}
