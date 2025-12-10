package streamable

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/ThinkInAIXYZ/go-mcp/protocol"
	"github.com/ThinkInAIXYZ/go-mcp/server"
	"github.com/r74tech/virga/internal/server/beacons"
	"github.com/r74tech/virga/internal/server/database"
	"github.com/r74tech/virga/internal/server/session"
)

// RegisterResources registers all resources to the streamable MCP server
func RegisterResources(mcpServer *server.Server, sessionMgr *session.Manager, beaconMgr *beacons.Manager, db *database.Database) error {
	// sessions_overview resource
	resource := &protocol.Resource{
		Name:        "sessions_overview",
		URI:         "virga://sessions/overview",
		Description: "Current status of all C2 sessions",
		MimeType:    "application/json",
	}
	mcpServer.RegisterResource(resource, func(ctx context.Context, req *protocol.ReadResourceRequest) (*protocol.ReadResourceResult, error) {
		sessions := sessionMgr.GetActiveSessions()

		overview := map[string]interface{}{
			"timestamp": time.Now().Format(time.RFC3339),
			"count":     len(sessions),
			"sessions":  []map[string]interface{}{},
		}

		for _, sess := range sessions {
			info := sess.Information()
			sessionData := map[string]interface{}{
				"id":        sess.ID,
				"agent_id":  sess.ID, // Use session ID as agent_id
				"username":  info["username"],
				"hostname":  info["hostname"],
				"os":        info["os"],
				"arch":      info["arch"],
				"ip":        info["ip"],
				"last_seen": sess.LastSeen().Format(time.RFC3339),
				"active":    sess.Status == "active",
			}
			overview["sessions"] = append(overview["sessions"].([]map[string]interface{}), sessionData)
		}

		data, err := json.MarshalIndent(overview, "", "  ")
		if err != nil {
			return nil, err
		}

		return &protocol.ReadResourceResult{
			Contents: []protocol.ResourceContents{
				&protocol.TextResourceContents{
					URI:      "virga://sessions/overview",
					MimeType: "application/json",
					Text:     string(data),
				},
			},
		}, nil
	})

	// session/{id}/info resource (dynamic)
	resource = &protocol.Resource{
		Name:        "session_info",
		URI:         "virga://session/{id}/info",
		Description: "Detailed information about a specific session",
		MimeType:    "application/json",
	}
	mcpServer.RegisterResource(resource, func(ctx context.Context, req *protocol.ReadResourceRequest) (*protocol.ReadResourceResult, error) {
		// Extract session ID from URI
		sessionID := ""
		if req.URI != "" {
			// Extract ID from "virga://session/{id}/info"
			parts := parseURI(req.URI)
			if len(parts) > 2 && parts[0] == "session" && parts[2] == "info" {
				sessionID = parts[1]
			}
		}

		if sessionID == "" {
			return nil, fmt.Errorf("invalid URI: %s", req.URI)
		}

		sess := sessionMgr.GetSession(sessionID)
		if sess == nil {
			return nil, fmt.Errorf("session not found: %s", sessionID)
		}

		info := sess.Information()
		details := map[string]interface{}{
			"id":              sessionID,
			"agent_id":        sess.ID, // Use session ID as agent_id
			"username":        info["username"],
			"hostname":        info["hostname"],
			"os":              info["os"],
			"arch":            info["arch"],
			"ip":              info["ip"],
			"last_seen":       sess.LastSeen().Format(time.RFC3339),
			"active":          sess.Status == "active",
			"interactive":     sess.IsInteractive(),
			"reconnect_time":  sess.GetReconnectTime().String(),
			"pending_tasks":   len(sess.GetPendingTasks()),
			"completed_tasks": len(sess.GetTaskResults()),
		}

		data, err := json.MarshalIndent(details, "", "  ")
		if err != nil {
			return nil, err
		}

		return &protocol.ReadResourceResult{
			Contents: []protocol.ResourceContents{
				&protocol.TextResourceContents{
					URI:      req.URI,
					MimeType: "application/json",
					Text:     string(data),
				},
			},
		}, nil
	})

	// beacons_overview resource
	resource = &protocol.Resource{
		Name:        "beacons_overview",
		URI:         "virga://beacons/overview",
		Description: "Overview of all registered beacons",
		MimeType:    "application/json",
	}
	mcpServer.RegisterResource(resource, func(ctx context.Context, req *protocol.ReadResourceRequest) (*protocol.ReadResourceResult, error) {
		beacons := beaconMgr.ListBeacons()

		overview := map[string]interface{}{
			"timestamp": time.Now().Format(time.RFC3339),
			"count":     len(beacons),
			"beacons":   []map[string]interface{}{},
		}

		for _, beacon := range beacons {
			beaconData := map[string]interface{}{
				"id":         beacon.ID,
				"name":       beacon.Name,
				"type":       beacon.Type,
				"created_at": beacon.CreatedAt.Format(time.RFC3339),
			}
			overview["beacons"] = append(overview["beacons"].([]map[string]interface{}), beaconData)
		}

		data, err := json.MarshalIndent(overview, "", "  ")
		if err != nil {
			return nil, err
		}

		return &protocol.ReadResourceResult{
			Contents: []protocol.ResourceContents{
				&protocol.TextResourceContents{
					URI:      "virga://beacons/overview",
					MimeType: "application/json",
					Text:     string(data),
				},
			},
		}, nil
	})

	// system_stats resource
	resource = &protocol.Resource{
		Name:        "system_stats",
		URI:         "virga://system/stats",
		Description: "Virga system statistics",
		MimeType:    "application/json",
	}
	mcpServer.RegisterResource(resource, func(ctx context.Context, req *protocol.ReadResourceRequest) (*protocol.ReadResourceResult, error) {
		var totalSessions, totalTasks int
		if err := db.QueryRow("SELECT COUNT(*) FROM sessions").Scan(&totalSessions); err != nil {
			return nil, fmt.Errorf("failed to count sessions: %w", err)
		}
		if err := db.QueryRow("SELECT COUNT(*) FROM tasks").Scan(&totalTasks); err != nil {
			return nil, fmt.Errorf("failed to count tasks: %w", err)
		}

		stats := map[string]interface{}{
			"timestamp": time.Now().Format(time.RFC3339),
			"server": map[string]interface{}{
				"name":    "Virga",
				"version": "2.0.0",
				"uptime":  time.Since(startTime).String(),
			},
			"sessions": map[string]interface{}{
				"active": len(sessionMgr.GetActiveSessions()),
				"total":  totalSessions,
			},
			"beacons": map[string]interface{}{
				"total": beaconMgr.Count(),
			},
			"tasks": map[string]interface{}{
				"total": totalTasks,
			},
		}

		data, err := json.MarshalIndent(stats, "", "  ")
		if err != nil {
			return nil, err
		}

		return &protocol.ReadResourceResult{
			Contents: []protocol.ResourceContents{
				&protocol.TextResourceContents{
					URI:      "virga://system/stats",
					MimeType: "application/json",
					Text:     string(data),
				},
			},
		}, nil
	})

	return nil
}

// parseURI parses a URI like "virga://session/123/info" into parts
func parseURI(uri string) []string {
	// Remove scheme
	const scheme = "virga://"
	if len(uri) < len(scheme) {
		return []string{}
	}

	path := uri[len(scheme):]
	parts := []string{}

	// Split by '/' and filter empty strings
	for _, part := range strings.Split(path, "/") {
		if part != "" {
			parts = append(parts, part)
		}
	}

	return parts
}
