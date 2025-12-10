package protocol

import (
	"testing"
	"time"
)

func TestTask(t *testing.T) {
	now := time.Now()

	task := &Task{
		ID:        "task-123",
		TaskID:    "task-123",
		Type:      TaskTypeCommand,
		SessionID: "session-456",
		Command:   "whoami",
		Arguments: map[string]interface{}{
			"timeout": 30,
		},
		CreatedAt: now,
		Status:    TaskStatusPending,
	}

	// Test initial state
	if task.ID != "task-123" {
		t.Errorf("Expected ID task-123, got %s", task.ID)
	}

	if task.Type != TaskTypeCommand {
		t.Errorf("Expected type %s, got %s", TaskTypeCommand, task.Type)
	}

	if task.Status != TaskStatusPending {
		t.Errorf("Expected status %s, got %s", TaskStatusPending, task.Status)
	}

	if task.CompletedAt != nil {
		t.Error("Expected CompletedAt to be nil for new task")
	}

	// Test updating task
	completedTime := now.Add(5 * time.Second)
	task.CompletedAt = &completedTime
	task.Status = TaskStatusCompleted
	task.Result = "user123"

	if task.CompletedAt == nil || !task.CompletedAt.Equal(completedTime) {
		t.Error("CompletedAt not set correctly")
	}

	if task.Status != TaskStatusCompleted {
		t.Errorf("Expected status %s, got %s", TaskStatusCompleted, task.Status)
	}
}

func TestTaskResult(t *testing.T) {
	now := time.Now()

	result := TaskResult{
		TaskID:   "task-123",
		Output:   "Command executed successfully",
		ExitCode: 0,
		Time:     now,
	}

	if result.TaskID != "task-123" {
		t.Errorf("Expected TaskID task-123, got %s", result.TaskID)
	}

	if result.ExitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", result.ExitCode)
	}

	// Test with error
	errorResult := TaskResult{
		TaskID:   "task-456",
		Output:   "",
		ExitCode: 1,
		Error:    "Command not found",
		Time:     now,
	}

	if errorResult.Error != "Command not found" {
		t.Errorf("Expected error 'Command not found', got %s", errorResult.Error)
	}

	if errorResult.ExitCode != 1 {
		t.Errorf("Expected exit code 1, got %d", errorResult.ExitCode)
	}
}

func TestAgentMessage(t *testing.T) {
	msg := AgentMessage{
		AgentID:  "agent-789",
		Hostname: "test-host",
		Username: "test-user",
		OS:       "linux",
		Version:  "1.0.0",
		IP:       "192.168.1.100",
		BootTime: time.Now().Unix(),
		Environment: map[string]interface{}{
			"arch": "amd64",
			"pid":  12345,
		},
		TaskResults: []AgentTaskResult{
			{
				TaskID:    "task-001",
				Output:    "result1",
				ExitCode:  0,
				Timestamp: time.Now().Unix(),
			},
		},
	}

	// Test fields
	if msg.AgentID != "agent-789" {
		t.Errorf("Expected AgentID agent-789, got %s", msg.AgentID)
	}

	if msg.Hostname != "test-host" {
		t.Errorf("Expected Hostname test-host, got %s", msg.Hostname)
	}

	if len(msg.TaskResults) != 1 {
		t.Fatalf("Expected 1 task result, got %d", len(msg.TaskResults))
	}

	if msg.TaskResults[0].TaskID != "task-001" {
		t.Errorf("Expected task result ID task-001, got %s", msg.TaskResults[0].TaskID)
	}

	// Test environment
	if arch, ok := msg.Environment["arch"].(string); !ok || arch != "amd64" {
		t.Error("Environment arch not set correctly")
	}
}

func TestAgentTaskResult(t *testing.T) {
	result := AgentTaskResult{
		TaskID:    "task-123",
		Output:    "Success",
		ExitCode:  0,
		Timestamp: time.Now().Unix(),
	}

	if result.TaskID != "task-123" {
		t.Errorf("Expected TaskID task-123, got %s", result.TaskID)
	}

	if result.ExitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", result.ExitCode)
	}

	if result.Error != "" {
		t.Errorf("Expected no error, got %s", result.Error)
	}
}

func TestAgentResponse(t *testing.T) {
	tasks := []Task{
		{
			ID:     "task-1",
			Type:   TaskTypeCommand,
			Status: TaskStatusPending,
		},
		{
			ID:     "task-2",
			Type:   TaskTypeFileList,
			Status: TaskStatusPending,
		},
	}

	response := AgentResponse{
		Status:        "ok",
		Tasks:         tasks,
		Interactive:   true,
		ReconnectTime: 60,
	}

	if response.Status != "ok" {
		t.Errorf("Expected status ok, got %s", response.Status)
	}

	if len(response.Tasks) != 2 {
		t.Fatalf("Expected 2 tasks, got %d", len(response.Tasks))
	}

	if !response.Interactive {
		t.Error("Expected interactive mode to be true")
	}

	if response.ReconnectTime != 60 {
		t.Errorf("Expected reconnect time 60, got %d", response.ReconnectTime)
	}
}

// Mock implementations for interface testing
type mockSessionHandler struct {
	pendingTasks []Task
	results      []TaskResult
}

func (m *mockSessionHandler) GetPendingTasks() []Task {
	return m.pendingTasks
}

func (m *mockSessionHandler) AddTaskResult(result TaskResult) {
	m.results = append(m.results, result)
}

type mockAgentConnection struct {
	sendErr      error
	remoteAddr   string
	lastActivity time.Time
	closed       bool
}

func (m *mockAgentConnection) Send(data []byte) error {
	return m.sendErr
}

func (m *mockAgentConnection) RemoteAddr() string {
	return m.remoteAddr
}

func (m *mockAgentConnection) LastActivity() time.Time {
	return m.lastActivity
}

func (m *mockAgentConnection) Close() error {
	m.closed = true
	return nil
}

func TestSessionHandlerInterface(t *testing.T) {
	handler := &mockSessionHandler{
		pendingTasks: []Task{
			{ID: "task-1", Status: TaskStatusPending},
		},
	}

	// Test GetPendingTasks
	tasks := handler.GetPendingTasks()
	if len(tasks) != 1 {
		t.Fatalf("Expected 1 pending task, got %d", len(tasks))
	}

	// Test AddTaskResult
	result := TaskResult{
		TaskID:   "task-1",
		Output:   "Done",
		ExitCode: 0,
		Time:     time.Now(),
	}

	handler.AddTaskResult(result)
	if len(handler.results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(handler.results))
	}

	if handler.results[0].TaskID != "task-1" {
		t.Errorf("Expected result TaskID task-1, got %s", handler.results[0].TaskID)
	}
}

func TestAgentConnectionInterface(t *testing.T) {
	now := time.Now()
	conn := &mockAgentConnection{
		remoteAddr:   "192.168.1.100:12345",
		lastActivity: now,
	}

	// Test RemoteAddr
	if conn.RemoteAddr() != "192.168.1.100:12345" {
		t.Errorf("Expected remote addr 192.168.1.100:12345, got %s", conn.RemoteAddr())
	}

	// Test LastActivity
	if !conn.LastActivity().Equal(now) {
		t.Error("LastActivity not correct")
	}

	// Test Send
	err := conn.Send([]byte("test data"))
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Test Close
	err = conn.Close()
	if err != nil {
		t.Errorf("Expected no error on close, got %v", err)
	}

	if !conn.closed {
		t.Error("Connection should be marked as closed")
	}
}
