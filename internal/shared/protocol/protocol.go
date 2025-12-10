package protocol

import "time"

// Task is the task sent to the agent
type Task struct {
	TaskID   string      `json:"task_id"`
	Type     string      `json:"type"`
	Payload  interface{} `json:"payload"`
	SentTime time.Time   `json:"sent_time,omitempty"`
}

// TaskResult is the task execution result
type TaskResult struct {
	TaskID   string    `json:"task_id"`
	Output   string    `json:"output"`
	ExitCode int       `json:"exit_code"`
	Error    string    `json:"error,omitempty"`
	Time     time.Time `json:"time"`
	Partial  bool      `json:"partial,omitempty"` // True if this is a partial/intermediate result
}

// TaskProgress represents the progress of a long-running task (for Web UI)
type TaskProgress struct {
	CurrentIteration int      `json:"current_iteration"`
	MaxIterations    int      `json:"max_iterations"`
	LastUpdate       int64    `json:"last_update"`
	IterationOutputs []string `json:"iteration_outputs,omitempty"`
}

// AgentMessage is the message from the agent
type AgentMessage struct {
	AgentID     string                 `json:"agent_id"`
	Hostname    string                 `json:"hostname"`
	Username    string                 `json:"username"`
	OS          string                 `json:"os"`
	Version     string                 `json:"version"`
	IP          string                 `json:"ip"`
	BootTime    int64                  `json:"boot_time"`
	Environment map[string]interface{} `json:"environment"`
	TaskResults []AgentTaskResult      `json:"task_results,omitempty"`
}

// AgentTaskResult is the task execution result returned from the agent
type AgentTaskResult struct {
	TaskID    string `json:"task_id"`
	Output    string `json:"output"`
	ExitCode  int    `json:"exit_code"`
	Error     string `json:"error,omitempty"`
	Timestamp int64  `json:"timestamp"`
	Partial   bool   `json:"partial"` // True if this is a partial/intermediate result
}

// AgentResponse is the response to the agent
type AgentResponse struct {
	Status        string `json:"status"`
	SessionID     string `json:"session_id,omitempty"` // Session ID for this beacon
	Tasks         []Task `json:"tasks,omitempty"`
	Message       string `json:"message,omitempty"`
	Interactive   bool   `json:"interactive,omitempty"`
	ReconnectTime int    `json:"reconnect_time,omitempty"` // Reconnect interval (seconds)
}

// SystemInfo holds system information
type SystemInfo struct {
	Hostname  string    `json:"hostname"`
	Username  string    `json:"username"`
	OS        string    `json:"os"`
	PID       int       `json:"pid"`
	IsAdmin   bool      `json:"is_admin"`
	Timestamp time.Time `json:"timestamp"`
}
