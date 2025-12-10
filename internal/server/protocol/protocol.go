package protocol

import "time"

// Task is the task sent to the agent
type Task struct {
	ID          string                 `json:"id"`
	TaskID      string                 `json:"task_id"` // For compatibility, keep this
	Type        TaskType               `json:"type"`
	SessionID   string                 `json:"session_id"`
	Command     string                 `json:"command,omitempty"`
	Arguments   map[string]interface{} `json:"arguments,omitempty"`
	Payload     interface{}            `json:"payload,omitempty"` // For compatibility, keep this
	CreatedAt   time.Time              `json:"created_at"`
	CompletedAt *time.Time             `json:"completed_at,omitempty"`
	SentTime    time.Time              `json:"sent_time,omitempty"`
	Status      TaskStatus             `json:"status"`
	Result      interface{}            `json:"result,omitempty"`
	Error       string                 `json:"error,omitempty"`
}

// TaskResult is the task execution result
type TaskResult struct {
	TaskID   string    `json:"task_id"`
	Output   string    `json:"output"`
	ExitCode int       `json:"exit_code"`
	Error    string    `json:"error,omitempty"`
	Time     time.Time `json:"time"`
	Partial  bool      `json:"partial"` // True if this is a partial/intermediate result
}

// SessionHandler is the session management interface
type SessionHandler interface {
	GetPendingTasks() []Task
	AddTaskResult(result TaskResult)
}

// AgentConnection is the agent connection interface
type AgentConnection interface {
	Send(data []byte) error
	RemoteAddr() string
	LastActivity() time.Time
	Close() error
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
