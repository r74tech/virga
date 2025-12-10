package session

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/r74tech/virga/internal/server/protocol"
	sharedprotocol "github.com/r74tech/virga/internal/shared/protocol"
)

// generateUUID generates a UUID
func generateUUID() string {
	uuid := make([]byte, 16)
	_, err := rand.Read(uuid)
	if err != nil {
		// If an error occurs, use the current time as a fallback
		return fmt.Sprintf("time-%d", time.Now().UnixNano())
	}

	// Set bits according to the UUID v4 format
	uuid[8] = uuid[8]&^0xc0 | 0x80
	uuid[6] = uuid[6]&^0xf0 | 0x40

	return hex.EncodeToString(uuid[0:4]) + "-" +
		hex.EncodeToString(uuid[4:6]) + "-" +
		hex.EncodeToString(uuid[6:8]) + "-" +
		hex.EncodeToString(uuid[8:10]) + "-" +
		hex.EncodeToString(uuid[10:])
}

// Session represents the session with the agent
type Session struct {
	ID              string
	agentConnection protocol.AgentConnection
	information     map[string]interface{}
	lastSeen        time.Time
	pendingTasks    []protocol.Task
	completedTasks  map[string]protocol.Task // Record completed tasks
	taskResults     []protocol.TaskResult
	mutex           sync.RWMutex
	isInteractive   bool          // Interactive mode flag
	reconnectTime   time.Duration // Reconnect interval
	debugMode       bool          // Debug mode flag

	// Task tracking for Web UI (prevent 404 errors)
	sentTasks map[string]time.Time // Maps TaskID to sent time
	sentMutex sync.RWMutex         // Mutex for sentTasks

	// Task progress tracking for Web UI
	taskProgress  map[string]*sharedprotocol.TaskProgress
	progressMutex sync.RWMutex

	// MCP integration fields
	Hostname   string
	Username   string
	OS         string
	Arch       string
	IP         string
	CreatedAt  time.Time
	UpdatedAt  time.Time
	Status     string
	Integrity  string
	Privileges string
	ProcessID  int
	WorkingDir string
	Tasks      []*protocol.Task
	Metadata   map[string]interface{}
	BeaconID   string

	// Database integration
	taskToCommandMap map[string]int64 // Maps TaskID to database CommandID
}

// NewSession creates a new session
func NewSession(agentID string, conn protocol.AgentConnection) *Session {
	now := time.Now()
	return &Session{
		ID:              agentID,
		agentConnection: conn,
		information:     make(map[string]interface{}),
		lastSeen:        now,
		pendingTasks:    []protocol.Task{},
		completedTasks:  make(map[string]protocol.Task), // Initialize completed tasks map
		taskResults:     []protocol.TaskResult{},
		sentTasks:       make(map[string]time.Time),                    // Initialize sent tasks tracking
		taskProgress:    make(map[string]*sharedprotocol.TaskProgress), // Initialize task progress tracking
		mutex:           sync.RWMutex{},
		isInteractive:   false,            // Default is non-interactive
		reconnectTime:   30 * time.Second, // Default reconnect interval is 30 seconds
		debugMode:       false,            // Default is debug mode disabled

		// MCP integration fields
		CreatedAt:        now,
		UpdatedAt:        now,
		Status:           "active",
		Tasks:            []*protocol.Task{},
		Metadata:         make(map[string]interface{}),
		BeaconID:         agentID,
		taskToCommandMap: make(map[string]int64), // Initialize task-to-command mapping
	}
}

// UpdateConnection updates the agent connection
func (s *Session) UpdateConnection(conn protocol.AgentConnection) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.agentConnection = conn
	s.lastSeen = time.Now()
}

// UpdateInformation updates the agent information
func (s *Session) UpdateInformation(info map[string]interface{}) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	for k, v := range info {
		s.information[k] = v

		// Update MCP fields
		switch k {
		case "hostname":
			if h, ok := v.(string); ok {
				s.Hostname = h
			}
		case "username":
			if u, ok := v.(string); ok {
				s.Username = u
			}
		case "os":
			if o, ok := v.(string); ok {
				s.OS = o
			}
		case "arch":
			if a, ok := v.(string); ok {
				s.Arch = a
			}
		case "ip":
			if i, ok := v.(string); ok {
				s.IP = i
			}
		case "process_id":
			// JSON-loaded numbers are float64
			switch p := v.(type) {
			case float64:
				s.ProcessID = int(p)
			case int:
				s.ProcessID = p
			}
		case "working_dir":
			if w, ok := v.(string); ok {
				s.WorkingDir = w
			}
		case "integrity":
			if i, ok := v.(string); ok {
				s.Integrity = i
			}
		case "privileges":
			if p, ok := v.(string); ok {
				s.Privileges = p
			}
		}
	}
	s.lastSeen = time.Now()
}

// LastSeen returns the last connection time
func (s *Session) LastSeen() time.Time {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.lastSeen
}

// Information returns the agent information
func (s *Session) Information() map[string]interface{} {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	// Create a copy of the information
	info := make(map[string]interface{}, len(s.information))
	for k, v := range s.information {
		info[k] = v
	}

	return info
}

// GetInformation returns the agent information (for API integration)
func (s *Session) GetInformation() map[string]interface{} {
	return s.Information()
}

// RemoteAddr returns the agent's remote address
func (s *Session) RemoteAddr() string {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	if s.agentConnection != nil {
		return s.agentConnection.RemoteAddr()
	}
	return "unknown"
}

// AddTask adds a task to the agent
func (s *Session) AddTask(task protocol.Task) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	task.SentTime = time.Now()
	s.pendingTasks = append(s.pendingTasks, task)

	if s.debugMode {
		log.Printf("[DEBUG] Session %s: Task %s added (type: %s, pending count: %d)",
			s.ID, task.TaskID, task.Type, len(s.pendingTasks))
	}
}

// AddTaskResult adds a task execution result
func (s *Session) AddTaskResult(result protocol.TaskResult) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	result.Time = time.Now()
	s.taskResults = append(s.taskResults, result)

	// Remove from sentTasks (task is now completed)
	s.sentMutex.Lock()
	delete(s.sentTasks, result.TaskID)
	s.sentMutex.Unlock()

	// Remove completed tasks from pending tasks and record them as completed
	newPendingTasks := []protocol.Task{}
	taskFound := false
	for _, task := range s.pendingTasks {
		if task.TaskID != result.TaskID {
			newPendingTasks = append(newPendingTasks, task)
		} else {
			// Record completed tasks
			s.completedTasks[result.TaskID] = task
			taskFound = true
		}
	}
	s.pendingTasks = newPendingTasks

	// Set flag for debug logging
	if s.debugMode {
		if taskFound {
			log.Printf("[DEBUG] Session %s: Task %s completed successfully (exit code: %d, output: %d bytes)",
				s.ID, result.TaskID, result.ExitCode, len(result.Output))
			log.Printf("[DEBUG] Session %s: Pending tasks count: %d -> %d",
				s.ID, len(s.pendingTasks)+1, len(s.pendingTasks))
		} else {
			log.Printf("[DEBUG] Session %s: WARNING - Task result received for unknown task %s",
				s.ID, result.TaskID)
			log.Printf("[DEBUG] Session %s: Current pending tasks: %d", s.ID, len(s.pendingTasks))
			for i, task := range s.pendingTasks {
				log.Printf("[DEBUG] Session %s:   Task #%d: %s (%s)",
					s.ID, i+1, task.TaskID, task.Type)
			}
		}
	}
}

// IsTaskCompleted checks if a task is completed
func (s *Session) IsTaskCompleted(taskID string) bool {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	_, completed := s.completedTasks[taskID]
	return completed
}

// GetPendingTasks returns pending tasks (for interactive mode)
func (s *Session) GetPendingTasks() []protocol.Task {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// For interactive mode, only send tasks that are not completed
	if s.isInteractive {
		activeTasks := []protocol.Task{}
		remainingTasks := []protocol.Task{}

		for _, task := range s.pendingTasks {
			// Only process tasks that are not completed
			if _, completed := s.completedTasks[task.TaskID]; !completed {
				// Create a copy of the task and set the Payload
				taskCopy := task

				// Convert task type to "shell"
				if task.Type == protocol.TaskTypeCommand || task.Type == "command" {
					taskCopy.Type = "shell"
				}

				// Command type tasks set Command field to Payload
				if task.Command != "" {
					taskCopy.Payload = map[string]interface{}{
						"command": task.Command,
					}
				}
				activeTasks = append(activeTasks, taskCopy)

				// Track this task as sent (for Web UI 404 prevention)
				s.sentMutex.Lock()
				s.sentTasks[task.TaskID] = time.Now()
				s.sentMutex.Unlock()
			} else {
				// Completed tasks are not included in the remaining tasks
				continue
			}
			// Tasks that are not yet completed are kept as remaining tasks
			remainingTasks = append(remainingTasks, task)
		}

		// Update pendingTasks (keep only tasks that are not completed)
		s.pendingTasks = remainingTasks
		s.lastSeen = time.Now()
		return activeTasks
	}

	// For non-interactive mode, proceed as usual
	tasks := make([]protocol.Task, len(s.pendingTasks))
	for i, task := range s.pendingTasks {
		// Create a copy of the task and set the Payload
		taskCopy := task

		// Convert task type to "shell"
		if task.Type == protocol.TaskTypeCommand || task.Type == "command" {
			taskCopy.Type = "shell"
		}

		// Command type tasks set Command field to Payload
		if task.Command != "" {
			taskCopy.Payload = map[string]interface{}{
				"command": task.Command,
			}
		}
		tasks[i] = taskCopy

		// Track this task as sent (for Web UI 404 prevention)
		s.sentMutex.Lock()
		s.sentTasks[task.TaskID] = time.Now()
		s.sentMutex.Unlock()
	}
	s.pendingTasks = []protocol.Task{}

	s.lastSeen = time.Now()
	return tasks
}

// GetTaskResults returns task execution results
func (s *Session) GetTaskResults() []protocol.TaskResult {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	results := make([]protocol.TaskResult, len(s.taskResults))
	copy(results, s.taskResults)

	return results
}

// GetTaskResult returns a specific task result
func (s *Session) GetTaskResult(taskID string) *protocol.TaskResult {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	for _, result := range s.taskResults {
		if result.TaskID == taskID {
			return &result
		}
	}
	return nil
}

// ClearTaskResults clears task execution results
func (s *Session) ClearTaskResults() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.taskResults = []protocol.TaskResult{}
}

// ClearCompletedTasks clears completed tasks
func (s *Session) ClearCompletedTasks() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.completedTasks = make(map[string]protocol.Task)
}

// ExecuteCommand executes a command on the agent
func (s *Session) ExecuteCommand(cmdType string, payload interface{}) (string, error) {
	s.mutex.Lock()

	// Generate a UUID-based task ID
	taskID := generateUUID()
	now := time.Now()

	// Create a task
	task := protocol.Task{
		ID:        taskID,
		TaskID:    taskID, // For compatibility
		Type:      protocol.TaskType(cmdType),
		SessionID: s.ID,
		Payload:   payload,
		Arguments: make(map[string]interface{}),
		CreatedAt: now,
		SentTime:  now,
		Status:    protocol.TaskStatusPending,
	}

	// If payload is map[string]interface{}, copy it to Arguments
	if args, ok := payload.(map[string]interface{}); ok {
		task.Arguments = args
	}

	// Add the task
	s.pendingTasks = append(s.pendingTasks, task)
	s.mutex.Unlock()

	// Send the task to the agent
	if s.agentConnection != nil {
		taskJSON, err := json.Marshal(task)
		if err != nil {
			return "", fmt.Errorf("failed to marshal task: %w", err)
		}

		if err := s.agentConnection.Send(taskJSON); err != nil {
			return "", fmt.Errorf("failed to send task: %w", err)
		}
	}

	// Return the task ID
	return taskID, nil
}

// SendResponse sends a response to the agent
func (s *Session) SendResponse(response []byte) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.agentConnection == nil {
		return fmt.Errorf("no active agent connection")
	}

	s.lastSeen = time.Now()
	return s.agentConnection.Send(response)
}

// Close closes the session
func (s *Session) Close() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.agentConnection != nil {
		return s.agentConnection.Close()
	}

	return nil
}

// SetInteractive sets the interactive mode
func (s *Session) SetInteractive(interactive bool) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.isInteractive = interactive

	// If interactive mode is disabled, clear completed tasks
	if !interactive {
		s.completedTasks = make(map[string]protocol.Task)
	}
}

// IsInteractive returns whether the session is in interactive mode
func (s *Session) IsInteractive() bool {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	return s.isInteractive
}

// GetReconnectTime returns the reconnect interval
func (s *Session) GetReconnectTime() time.Duration {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	return s.reconnectTime
}

// SetReconnectTime sets the reconnect interval
func (s *Session) SetReconnectTime(duration time.Duration) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.reconnectTime = duration
}

// SetDebugMode sets the debug mode
func (s *Session) SetDebugMode(debug bool) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.debugMode = debug
	if debug {
		log.Printf("[DEBUG] Session %s: Debug mode enabled", s.ID)
	}
}

// IsDebugMode returns whether the session is in debug mode
func (s *Session) IsDebugMode() bool {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	return s.debugMode
}

// SetTaskCommandMapping stores the mapping between a task ID and database command ID
func (s *Session) SetTaskCommandMapping(taskID string, commandID int64) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.taskToCommandMap == nil {
		s.taskToCommandMap = make(map[string]int64)
	}
	s.taskToCommandMap[taskID] = commandID
}

// GetTaskCommandMapping retrieves the database command ID for a given task ID
func (s *Session) GetTaskCommandMapping(taskID string) (int64, bool) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	if s.taskToCommandMap == nil {
		return 0, false
	}
	commandID, exists := s.taskToCommandMap[taskID]
	return commandID, exists
}

// ClearTaskCommandMapping removes the mapping for a given task ID
func (s *Session) ClearTaskCommandMapping(taskID string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.taskToCommandMap != nil {
		delete(s.taskToCommandMap, taskID)
	}
}

// IsTaskPending checks if a task is pending (not yet sent) or sent (but not completed)
func (s *Session) IsTaskPending(taskID string) bool {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	// Check if task is in pendingTasks
	for _, task := range s.pendingTasks {
		if task.TaskID == taskID {
			return true
		}
	}

	// Check if task has been sent to implant
	s.sentMutex.RLock()
	_, sent := s.sentTasks[taskID]
	s.sentMutex.RUnlock()

	return sent
}

// UpdateTaskProgress updates the progress information for a task
func (s *Session) UpdateTaskProgress(taskID string, progress *sharedprotocol.TaskProgress) {
	s.progressMutex.Lock()
	defer s.progressMutex.Unlock()
	s.taskProgress[taskID] = progress
}

// GetTaskProgress retrieves the progress information for a task
func (s *Session) GetTaskProgress(taskID string) *sharedprotocol.TaskProgress {
	s.progressMutex.RLock()
	defer s.progressMutex.RUnlock()
	return s.taskProgress[taskID]
}

// ClearTaskProgress removes the progress information for a task
func (s *Session) ClearTaskProgress(taskID string) {
	s.progressMutex.Lock()
	defer s.progressMutex.Unlock()
	delete(s.taskProgress, taskID)
}
