package stdio

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ThinkInAIXYZ/go-mcp/protocol"
	"github.com/google/uuid"
	"github.com/r74tech/virga/internal/server/mcp/tools"
	localprotocol "github.com/r74tech/virga/internal/server/protocol"
)

// registerTools registers all tools for the STDIO server
func (s *Server) registerTools() error {
	// Use centralized tool definitions
	s.mcpServer.RegisterTool(tools.SessionListTool, s.handleSessionList)

	s.mcpServer.RegisterTool(tools.SessionCommandTool, s.handleExecuteCommand)

	s.mcpServer.RegisterTool(tools.FileUploadTool, s.handleFileUpload)

	s.mcpServer.RegisterTool(tools.FileDownloadTool, s.handleFileDownload)

	s.mcpServer.RegisterTool(tools.ProcessListTool, s.handleProcessList)

	s.mcpServer.RegisterTool(tools.KillProcessTool, s.handleKillProcess)

	s.mcpServer.RegisterTool(tools.SystemInfoTool, s.handleSystemInfo)

	s.mcpServer.RegisterTool(tools.NetworkConnectionsTool, s.handleNetworkConnections)

	s.mcpServer.RegisterTool(tools.PortForwardTool, s.handlePortForward)

	return nil
}

// Tool handlers
func (s *Server) handleSessionList(ctx context.Context, req *protocol.CallToolRequest) (*protocol.CallToolResult, error) {
	sessions := s.sessionManager.GetActiveSessions()

	content := &protocol.TextContent{
		Type: "text",
		Text: fmt.Sprintf("Active sessions: %d", len(sessions)),
	}

	if len(sessions) > 0 {
		for _, session := range sessions {
			info := session.Information()
			content.Text += fmt.Sprintf("\n\nID: %s", session.ID)
			content.Text += fmt.Sprintf("\n  User: %s@%s", info["username"], info["hostname"])
			content.Text += fmt.Sprintf("\n  OS: %s (%s)", info["os"], info["arch"])
			content.Text += fmt.Sprintf("\n  IP: %s", info["ip"])
			content.Text += fmt.Sprintf("\n  Last Seen: %s", session.LastSeen().Format("2006-01-02 15:04:05"))
		}
	}

	return &protocol.CallToolResult{
		Content: []protocol.Content{content},
	}, nil
}

func (s *Server) handleExecuteCommand(ctx context.Context, req *protocol.CallToolRequest) (*protocol.CallToolResult, error) {
	sessionID, ok := req.Arguments["session_id"].(string)
	if !ok {
		return nil, fmt.Errorf("session_id is required")
	}

	command, ok := req.Arguments["command"].(string)
	if !ok {
		return nil, fmt.Errorf("command is required")
	}

	session := s.sessionManager.GetSession(sessionID)
	if session == nil {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}

	// Check if this is a Llama command
	useLlama := false
	if val, ok := req.Arguments["use_llama"].(bool); ok {
		useLlama = val
	}

	// Check if this is a MemDB command
	isMemDB := false
	memdbQuery := ""
	if strings.HasPrefix(strings.ToLower(command), "memdb ") {
		isMemDB = true
		memdbQuery = strings.TrimPrefix(command, "memdb ")
		memdbQuery = strings.TrimPrefix(memdbQuery, "MemDB ")
	}

	var taskID string
	var err error

	if useLlama {
		// Get Llama parameters
		maxIterations := 5
		if val, ok := req.Arguments["max_iterations"].(float64); ok {
			maxIterations = int(val)
		}

		temperature := 0.3
		if val, ok := req.Arguments["temperature"].(float64); ok {
			temperature = val
		}

		// Execute as Llama interactive task
		taskID, err = session.ExecuteCommand(string(localprotocol.TaskTypeLlamaInteractive), map[string]interface{}{
			"prompt":         command,
			"max_iterations": maxIterations,
			"temperature":    temperature,
		})
	} else if isMemDB {
		// Execute as MemDB query
		taskID, err = session.ExecuteCommand(string(localprotocol.TaskTypeMemDBQuery), map[string]interface{}{
			"query": memdbQuery,
		})
	} else {
		// Execute as regular shell command
		taskID, err = session.ExecuteCommand("shell", map[string]interface{}{
			"command": command,
		})
	}

	if err != nil {
		return nil, fmt.Errorf("failed to execute command: %w", err)
	}

	// Return immediately with task info
	var commandType string
	if useLlama {
		commandType = "Llama interactive"
	} else if isMemDB {
		commandType = "MemDB query"
	} else {
		commandType = "Shell command"
	}

	return &protocol.CallToolResult{
		Content: []protocol.Content{
			&protocol.TextContent{
				Type: "text",
				Text: fmt.Sprintf("%s task created with ID: %s\nCommand: %s", commandType, taskID, command),
			},
		},
	}, nil
}

func (s *Server) handleFileUpload(ctx context.Context, req *protocol.CallToolRequest) (*protocol.CallToolResult, error) {
	sessionID, ok := req.Arguments["session_id"].(string)
	if !ok {
		return nil, fmt.Errorf("session_id is required")
	}

	remotePath, ok := req.Arguments["remote_path"].(string)
	if !ok {
		return nil, fmt.Errorf("remote_path is required")
	}

	contentB64, ok := req.Arguments["content"].(string)
	if !ok {
		return nil, fmt.Errorf("content is required")
	}

	content, err := base64.StdEncoding.DecodeString(contentB64)
	if err != nil {
		return nil, fmt.Errorf("failed to decode content: %w", err)
	}

	session := s.sessionManager.GetSession(sessionID)
	if session == nil {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}

	// Execute upload command
	taskID, err := session.ExecuteCommand("upload", map[string]interface{}{
		"remote_path": remotePath,
		"content":     base64.StdEncoding.EncodeToString(content),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to upload file: %w", err)
	}

	// Return immediately with task info
	return &protocol.CallToolResult{
		Content: []protocol.Content{
			&protocol.TextContent{
				Type: "text",
				Text: fmt.Sprintf("File upload task created with ID: %s for %s", taskID, remotePath),
			},
		},
	}, nil
}

func (s *Server) handleFileDownload(ctx context.Context, req *protocol.CallToolRequest) (*protocol.CallToolResult, error) {
	sessionID, ok := req.Arguments["session_id"].(string)
	if !ok {
		return nil, fmt.Errorf("session_id is required")
	}

	remotePath, ok := req.Arguments["remote_path"].(string)
	if !ok {
		return nil, fmt.Errorf("remote_path is required")
	}

	session := s.sessionManager.GetSession(sessionID)
	if session == nil {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}

	// Execute download command
	taskID, err := session.ExecuteCommand("download", map[string]interface{}{
		"remote_path": remotePath,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to download file: %w", err)
	}

	// Return immediately with task info
	return &protocol.CallToolResult{
		Content: []protocol.Content{
			&protocol.TextContent{
				Type: "text",
				Text: fmt.Sprintf("File download task created with ID: %s for %s", taskID, remotePath),
			},
		},
	}, nil
}

func (s *Server) handleProcessList(ctx context.Context, req *protocol.CallToolRequest) (*protocol.CallToolResult, error) {
	sessionID, ok := req.Arguments["session_id"].(string)
	if !ok {
		return nil, fmt.Errorf("session_id is required")
	}

	session := s.sessionManager.GetSession(sessionID)
	if session == nil {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}

	// Create process list task
	taskID := uuid.New().String()
	task := localprotocol.Task{
		TaskID: taskID,
		Type:   "ps",
	}

	session.AddTask(task)

	// Return immediately with task info
	return &protocol.CallToolResult{
		Content: []protocol.Content{
			&protocol.TextContent{
				Type: "text",
				Text: fmt.Sprintf("Process list task created with ID: %s", taskID),
			},
		},
	}, nil
}

func (s *Server) handleKillProcess(ctx context.Context, req *protocol.CallToolRequest) (*protocol.CallToolResult, error) {
	sessionID, ok := req.Arguments["session_id"].(string)
	if !ok {
		return nil, fmt.Errorf("session_id is required")
	}

	pidFloat, ok := req.Arguments["pid"].(float64)
	if !ok {
		return nil, fmt.Errorf("pid is required")
	}
	pid := int(pidFloat)

	session := s.sessionManager.GetSession(sessionID)
	if session == nil {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}

	// Create kill process task
	taskID := uuid.New().String()
	task := localprotocol.Task{
		TaskID:  taskID,
		Type:    "kill",
		Command: fmt.Sprintf("%d", pid),
	}

	session.AddTask(task)

	// Return immediately with task info
	return &protocol.CallToolResult{
		Content: []protocol.Content{
			&protocol.TextContent{
				Type: "text",
				Text: fmt.Sprintf("Kill process task created with ID: %s for PID %d", taskID, pid),
			},
		},
	}, nil
}

func (s *Server) handleSystemInfo(ctx context.Context, req *protocol.CallToolRequest) (*protocol.CallToolResult, error) {
	sessionID, ok := req.Arguments["session_id"].(string)
	if !ok {
		return nil, fmt.Errorf("session_id is required")
	}

	session := s.sessionManager.GetSession(sessionID)
	if session == nil {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}

	// Get session info
	info := session.Information()

	// Create system info task for more details
	taskID := uuid.New().String()
	task := localprotocol.Task{
		TaskID: taskID,
		Type:   "sysinfo",
	}

	session.AddTask(task)

	// Return immediately with task info and basic session info
	infoJSON, _ := json.MarshalIndent(info, "", "  ")
	return &protocol.CallToolResult{
		Content: []protocol.Content{
			&protocol.TextContent{
				Type: "text",
				Text: fmt.Sprintf("System info task created with ID: %s\nBasic system information:\n%s", taskID, infoJSON),
			},
		},
	}, nil
}

func (s *Server) handleNetworkConnections(ctx context.Context, req *protocol.CallToolRequest) (*protocol.CallToolResult, error) {
	sessionID, ok := req.Arguments["session_id"].(string)
	if !ok {
		return nil, fmt.Errorf("session_id is required")
	}

	session := s.sessionManager.GetSession(sessionID)
	if session == nil {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}

	// Create network connections task
	taskID := uuid.New().String()
	task := localprotocol.Task{
		TaskID: taskID,
		Type:   "netstat",
	}

	session.AddTask(task)

	// Return immediately with task info
	return &protocol.CallToolResult{
		Content: []protocol.Content{
			&protocol.TextContent{
				Type: "text",
				Text: fmt.Sprintf("Network connections task created with ID: %s", taskID),
			},
		},
	}, nil
}

func (s *Server) handlePortForward(ctx context.Context, req *protocol.CallToolRequest) (*protocol.CallToolResult, error) {
	sessionID, ok := req.Arguments["session_id"].(string)
	if !ok {
		return nil, fmt.Errorf("session_id is required")
	}

	localPortFloat, ok := req.Arguments["local_port"].(float64)
	if !ok {
		return nil, fmt.Errorf("local_port is required")
	}
	localPort := int(localPortFloat)

	remoteHost, ok := req.Arguments["remote_host"].(string)
	if !ok {
		return nil, fmt.Errorf("remote_host is required")
	}

	remotePortFloat, ok := req.Arguments["remote_port"].(float64)
	if !ok {
		return nil, fmt.Errorf("remote_port is required")
	}
	remotePort := int(remotePortFloat)

	session := s.sessionManager.GetSession(sessionID)
	if session == nil {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}

	// Create port forward task
	taskID := uuid.New().String()
	task := localprotocol.Task{
		TaskID:  taskID,
		Type:    "portfwd",
		Command: fmt.Sprintf("%d:%s:%d", localPort, remoteHost, remotePort),
	}

	session.AddTask(task)

	// Return immediately with task info
	return &protocol.CallToolResult{
		Content: []protocol.Content{
			&protocol.TextContent{
				Type: "text",
				Text: fmt.Sprintf("Port forward task created with ID: %s\nForwarding localhost:%d -> %s:%d", taskID, localPort, remoteHost, remotePort),
			},
		},
	}, nil
}
