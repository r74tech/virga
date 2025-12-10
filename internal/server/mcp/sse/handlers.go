package sse

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/ThinkInAIXYZ/go-mcp/protocol"
	"github.com/r74tech/virga/internal/server/mcp/tools"
	localprotocol "github.com/r74tech/virga/internal/server/protocol"
)

// registerTools registers all tools for the SSE server
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
	if lower := strings.ToLower(command); strings.HasPrefix(lower, "memdb ") {
		isMemDB = true
		memdbQuery = strings.TrimSpace(command[len("memdb "):])
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

		// Log Llama interaction to database
		if err == nil && s.db != nil {
			llamaID, dbErr := s.db.LogLlamaInteraction(sessionID, taskID, command, "llama", maxIterations, temperature)
			if dbErr != nil {
				// Log error but don't fail the command
				log.Printf("Failed to log Llama interaction to database: %v\n", dbErr)
			} else {
				// Store the Llama interaction ID for later update
				session.SetTaskCommandMapping(taskID, llamaID)
			}
		}
	} else if isMemDB {
		// Execute as MemDB query
		taskID, err = session.ExecuteCommand(string(localprotocol.TaskTypeMemDBQuery), map[string]interface{}{
			"query": memdbQuery,
		})

		// Log MemDB query to database - we'll log result when it completes
		if err == nil && s.db != nil {
			// Store task info for later when we get the result
			session.SetTaskCommandMapping(taskID, -1) // Special marker for MemDB queries
		}
	} else {
		// Execute as regular shell command
		taskID, err = session.ExecuteCommand("shell", map[string]interface{}{
			"command": command,
		})
	}

	if err != nil {
		return nil, fmt.Errorf("failed to execute command: %w", err)
	}

	// Wait for result - longer timeout for Llama commands
	timeout := 30 * time.Second
	if useLlama {
		timeout = 5 * time.Minute // Llama commands can take longer
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("command execution timeout")
		case <-time.After(100 * time.Millisecond):
			results := session.GetTaskResults()
			for _, result := range results {
				if result.TaskID == taskID {
					// Update database based on task type
					if s.db != nil {
						if commandID, exists := session.GetTaskCommandMapping(taskID); exists {
							if commandID == -1 && isMemDB {
								// This is a MemDB query result
								startTime := time.Now()
								executionTimeMs := int(time.Since(startTime).Milliseconds())

								// Count results (simple line count for now)
								resultCount := len(strings.Split(strings.TrimSpace(result.Output), "\n"))
								if result.Output == "" {
									resultCount = 0
								}

								if err := s.db.LogMemDBQuery(sessionID, memdbQuery, result.Output, resultCount, executionTimeMs); err != nil {
									log.Printf("Failed to log MemDB query to database: %v\n", err)
								}
							} else if commandID > 0 && useLlama {
								// This is a Llama interaction result
								status := "completed"
								if result.ExitCode != 0 || result.Error != "" {
									status = "failed"
								}
								if err := s.db.UpdateLlamaInteraction(commandID, result.Output, status); err != nil {
									log.Printf("Failed to update Llama interaction in database: %v\n", err)
								}
							}
							// Clear the mapping
							session.ClearTaskCommandMapping(taskID)
						}
					}

					content := &protocol.TextContent{
						Type: "text",
						Text: fmt.Sprintf("Command output:\n%s", result.Output),
					}
					if result.ExitCode != 0 {
						content.Text = fmt.Sprintf("Command failed (exit code %d):\n%s", result.ExitCode, result.Output)
					}
					return &protocol.CallToolResult{
						Content: []protocol.Content{content},
					}, nil
				}
			}
		}
	}
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

	// Create upload task
	taskID, err := session.ExecuteCommand("upload", map[string]interface{}{
		"remote_path": remotePath,
		"content":     content,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to upload file: %w", err)
	}

	// Wait for result
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("upload timeout")
		case <-time.After(100 * time.Millisecond):
			results := session.GetTaskResults()
			for _, result := range results {
				if result.TaskID == taskID {
					text := fmt.Sprintf("File uploaded successfully to %s", remotePath)
					if result.ExitCode != 0 {
						text = fmt.Sprintf("Upload failed: %s", result.Output)
					} else {
						// Log successful file upload to database
						if s.db != nil {
							// Calculate file hash (simple implementation - in production use crypto/sha256)
							fileSize := int64(len(content))
							fileHash := fmt.Sprintf("%x", len(content)) // Simple hash for now
							filename := remotePath
							if idx := strings.LastIndex(remotePath, "/"); idx >= 0 {
								filename = remotePath[idx+1:]
							}

							if err := s.db.LogFile(sessionID, filename, remotePath, fileHash, fileSize); err != nil {
								log.Printf("Failed to log file upload to database: %v\n", err)
							}
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

	// Create download task
	taskID, err := session.ExecuteCommand("download", map[string]interface{}{
		"remote_path": remotePath,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to download file: %w", err)
	}

	// Wait for result
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("download timeout")
		case <-time.After(100 * time.Millisecond):
			results := session.GetTaskResults()
			for _, result := range results {
				if result.TaskID == taskID {
					if result.ExitCode != 0 {
						return nil, fmt.Errorf("download failed: %s", result.Output)
					}

					// Log successful file download to database
					if s.db != nil {
						// Extract filename from path
						filename := remotePath
						if idx := strings.LastIndex(remotePath, "/"); idx >= 0 {
							filename = remotePath[idx+1:]
						}

						// For downloads, we don't have the actual file size/hash from the output
						// In a real implementation, this would be included in the task result
						fileSize := int64(len(result.Output))             // Approximate size from output
						fileHash := fmt.Sprintf("%x", len(result.Output)) // Simple hash for now

						if err := s.db.LogFile(sessionID, filename, remotePath, fileHash, fileSize); err != nil {
							log.Printf("Failed to log file download to database: %v\n", err)
						}
					}

					// Encode data as base64
					// Note: TaskResult doesn't have Data field, so we'll return the output as is
					return &protocol.CallToolResult{
						Content: []protocol.Content{
							&protocol.TextContent{
								Type: "text",
								Text: fmt.Sprintf("File downloaded from %s\nOutput:\n%s", remotePath, result.Output),
							},
						},
					}, nil
				}
			}
		}
	}
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
	taskID, err := session.ExecuteCommand("ps", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list processes: %w", err)
	}

	// Wait for result
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("process list timeout")
		case <-time.After(100 * time.Millisecond):
			results := session.GetTaskResults()
			for _, result := range results {
				if result.TaskID == taskID {
					return &protocol.CallToolResult{
						Content: []protocol.Content{
							&protocol.TextContent{
								Type: "text",
								Text: fmt.Sprintf("Process list:\n%s", result.Output),
							},
						},
					}, nil
				}
			}
		}
	}
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
	taskID, err := session.ExecuteCommand("kill", map[string]interface{}{
		"pid": pid,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to kill process: %w", err)
	}

	// Wait for result
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("kill process timeout")
		case <-time.After(100 * time.Millisecond):
			results := session.GetTaskResults()
			for _, result := range results {
				if result.TaskID == taskID {
					text := fmt.Sprintf("Process %d killed successfully", pid)
					if result.ExitCode != 0 {
						text = fmt.Sprintf("Failed to kill process %d: %s", pid, result.Output)
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
	taskID, err := session.ExecuteCommand("sysinfo", nil)
	if err != nil {
		// Return basic info if task fails
		infoJSON, _ := json.MarshalIndent(info, "", "  ")
		return &protocol.CallToolResult{
			Content: []protocol.Content{
				&protocol.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Basic system information:\n%s", infoJSON),
				},
			},
		}, nil
	}

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
	taskID, err := session.ExecuteCommand("netstat", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list network connections: %w", err)
	}

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
	taskID, err := session.ExecuteCommand("portfwd", map[string]interface{}{
		"local_port":  localPort,
		"remote_host": remoteHost,
		"remote_port": remotePort,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to setup port forward: %w", err)
	}

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
