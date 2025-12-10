package session

import (
	"fmt"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/r74tech/virga/internal/server/protocol"
)

// MockAgentConnection simulates an agent connection for testing
type MockAgentConnection struct {
	sentData     [][]byte
	closed       bool
	remoteAddr   string
	lastActivity time.Time
	mu           sync.Mutex
}

func NewMockAgentConnection() *MockAgentConnection {
	return &MockAgentConnection{
		remoteAddr:   "127.0.0.1:12345",
		lastActivity: time.Now(),
	}
}

func (m *MockAgentConnection) Send(data []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sentData = append(m.sentData, data)
	m.lastActivity = time.Now()
	return nil
}

func (m *MockAgentConnection) RemoteAddr() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.remoteAddr
}

func (m *MockAgentConnection) LastActivity() time.Time {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.lastActivity
}

func (m *MockAgentConnection) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
	return nil
}

func (m *MockAgentConnection) GetSentData() [][]byte {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([][]byte{}, m.sentData...)
}

func TestNewSession(t *testing.T) {
	mockConn := NewMockAgentConnection()
	sessionID := "test-session-123"

	session := NewSession(sessionID, mockConn)

	// Verify session creation
	if session.ID != sessionID {
		t.Errorf("Session ID = %s, want %s", session.ID, sessionID)
	}

	if session.agentConnection == nil {
		t.Error("Agent connection is nil")
	}

	if session.information == nil {
		t.Error("Information map is nil")
	}

	if session.pendingTasks == nil {
		t.Error("Pending tasks slice is nil")
	}

	if session.completedTasks == nil {
		t.Error("Completed tasks map is nil")
	}

	if session.taskResults == nil {
		t.Error("Task results slice is nil")
	}

	if session.Tasks == nil {
		t.Error("Tasks slice is nil")
	}

	if session.lastSeen.IsZero() {
		t.Error("Last seen time is zero")
	}

	if session.reconnectTime != 30*time.Second {
		t.Errorf("Reconnect time = %v, want 30s", session.reconnectTime)
	}

	if session.isInteractive {
		t.Error("Session should not be interactive by default")
	}

	if session.debugMode {
		t.Error("Debug mode should be false by default")
	}

	// Verify default values
	if session.Status != "active" {
		t.Errorf("Status = %s, want active", session.Status)
	}

	if session.CreatedAt.IsZero() {
		t.Error("CreatedAt should not be zero")
	}
}

func TestSession_UpdateInformation(t *testing.T) {
	session := NewSession("test", NewMockAgentConnection())

	info := map[string]interface{}{
		"hostname": "test-host",
		"username": "test-user",
		"os":       "linux",
		"arch":     "amd64",
		"ip":       "192.168.1.100",
		"pid":      1234,
		"cwd":      "/home/test",
	}

	session.UpdateInformation(info)

	// Verify information was stored
	for key, expectedValue := range info {
		if value, exists := session.information[key]; !exists {
			t.Errorf("Information[%s] does not exist", key)
		} else if value != expectedValue {
			t.Errorf("Information[%s] = %v, want %v", key, value, expectedValue)
		}
	}

	// Test partial update
	newInfo := map[string]interface{}{
		"hostname": "new-host",
		"new_key":  "new_value",
	}

	session.UpdateInformation(newInfo)

	// Verify update
	if hostname := session.information["hostname"]; hostname != "new-host" {
		t.Errorf("Hostname = %v, want new-host", hostname)
	}

	// Verify old data is preserved
	if username := session.information["username"]; username != "test-user" {
		t.Errorf("Username = %v, want test-user", username)
	}

	// Verify new key was added
	if newValue := session.information["new_key"]; newValue != "new_value" {
		t.Errorf("new_key = %v, want new_value", newValue)
	}
}

func TestSession_Information(t *testing.T) {
	session := NewSession("test", NewMockAgentConnection())

	info := map[string]interface{}{
		"hostname": "test-host",
		"username": "test-user",
	}

	session.UpdateInformation(info)

	// Test Information() method
	retrievedInfo := session.Information()

	if !reflect.DeepEqual(retrievedInfo, info) {
		t.Errorf("Retrieved info = %v, want %v", retrievedInfo, info)
	}

	// Test GetInformation() method
	retrievedInfo2 := session.GetInformation()

	if !reflect.DeepEqual(retrievedInfo2, info) {
		t.Errorf("GetInformation() = %v, want %v", retrievedInfo2, info)
	}
}

func TestSession_GetInformation(t *testing.T) {
	session := NewSession("test", NewMockAgentConnection())

	info := map[string]interface{}{
		"hostname": "test-host",
		"username": "test-user",
		"os":       "linux",
	}

	session.UpdateInformation(info)

	// Test existing key through Information map
	retrievedInfo := session.Information()
	if value := retrievedInfo["hostname"]; value != "test-host" {
		t.Errorf("Information[hostname] = %v, want test-host", value)
	}

	// Test non-existent key
	if value := retrievedInfo["non-existent"]; value != nil {
		t.Errorf("Information[non-existent] = %v, want nil", value)
	}
}

func TestSession_UpdateConnection(t *testing.T) {
	originalConn := NewMockAgentConnection()
	session := NewSession("test", originalConn)
	originalLastSeen := session.lastSeen

	// Wait to ensure time difference
	time.Sleep(10 * time.Millisecond)

	newConn := NewMockAgentConnection()
	session.UpdateConnection(newConn)

	// Since agentConnection is private, we can't directly check it
	// But we can verify lastSeen was updated
	if !session.lastSeen.After(originalLastSeen) {
		t.Error("Last seen time was not updated")
	}
}

func TestSession_SetInteractive(t *testing.T) {
	session := NewSession("test", NewMockAgentConnection())

	// Initially should be false
	if session.isInteractive {
		t.Error("Session should not be interactive by default")
	}

	// Set to true
	session.SetInteractive(true)
	if !session.isInteractive {
		t.Error("Session should be interactive after setting to true")
	}

	// Set to false
	session.SetInteractive(false)
	if session.isInteractive {
		t.Error("Session should not be interactive after setting to false")
	}
}

func TestSession_IsInteractive(t *testing.T) {
	session := NewSession("test", NewMockAgentConnection())

	if session.IsInteractive() {
		t.Error("IsInteractive() should return false by default")
	}

	session.SetInteractive(true)
	if !session.IsInteractive() {
		t.Error("IsInteractive() should return true after setting")
	}
}

func TestSession_SetDebugMode(t *testing.T) {
	session := NewSession("test", NewMockAgentConnection())

	// Initially should be false
	if session.debugMode {
		t.Error("Debug mode should be false by default")
	}

	// Set to true
	session.SetDebugMode(true)
	if !session.debugMode {
		t.Error("Debug mode should be true after setting")
	}

	// Set to false
	session.SetDebugMode(false)
	if session.debugMode {
		t.Error("Debug mode should be false after setting")
	}
}

func TestSession_SetReconnectTime(t *testing.T) {
	session := NewSession("test", NewMockAgentConnection())

	// Initially should be 30 seconds
	if session.reconnectTime != 30*time.Second {
		t.Errorf("Reconnect time = %v, want 30s", session.reconnectTime)
	}

	// Set reconnect time
	newTime := 45 * time.Second
	session.SetReconnectTime(newTime)
	if session.reconnectTime != newTime {
		t.Errorf("Reconnect time = %v, want %v", session.reconnectTime, newTime)
	}
}

func TestSession_GetReconnectTime(t *testing.T) {
	session := NewSession("test", NewMockAgentConnection())

	if duration := session.GetReconnectTime(); duration != 30*time.Second {
		t.Errorf("GetReconnectTime() = %v, want 30s", duration)
	}

	newTime := 45 * time.Second
	session.SetReconnectTime(newTime)
	if duration := session.GetReconnectTime(); duration != newTime {
		t.Errorf("GetReconnectTime() = %v, want %v", duration, newTime)
	}
}

func TestSession_AddTask(t *testing.T) {
	session := NewSession("test", NewMockAgentConnection())

	task1 := protocol.Task{
		ID:        "task-1",
		Type:      "command",
		Command:   "ls",
		Status:    "pending",
		CreatedAt: time.Now(),
	}

	task2 := protocol.Task{
		ID:        "task-2",
		Type:      "file_list",
		Arguments: map[string]interface{}{"path": "/tmp"},
		Status:    "pending",
		CreatedAt: time.Now(),
	}

	// Add tasks
	session.AddTask(task1)
	session.AddTask(task2)

	// Verify tasks were added
	if len(session.pendingTasks) != 2 {
		t.Errorf("Pending tasks count = %d, want 2", len(session.pendingTasks))
	}

	// Verify task order
	if session.pendingTasks[0].ID != "task-1" {
		t.Errorf("First task ID = %s, want task-1", session.pendingTasks[0].ID)
	}
	if session.pendingTasks[1].ID != "task-2" {
		t.Errorf("Second task ID = %s, want task-2", session.pendingTasks[1].ID)
	}
}

func TestSession_GetPendingTasks(t *testing.T) {
	session := NewSession("test", NewMockAgentConnection())

	// Test with no tasks
	tasks := session.GetPendingTasks()
	if len(tasks) != 0 {
		t.Errorf("Empty session returned %d tasks", len(tasks))
	}

	// Add tasks
	task1 := protocol.Task{ID: "task-1", TaskID: "task-1", Type: "command", Status: "pending", CreatedAt: time.Now()}
	task2 := protocol.Task{ID: "task-2", TaskID: "task-2", Type: "file_list", Status: "pending", CreatedAt: time.Now()}
	task3 := protocol.Task{ID: "task-3", TaskID: "task-3", Type: "command", Status: "pending", CreatedAt: time.Now()}

	session.AddTask(task1)
	session.AddTask(task2)
	session.AddTask(task3)

	// Test non-interactive mode (should clear all tasks)
	session.SetInteractive(false)
	tasks = session.GetPendingTasks()

	if len(tasks) != 3 {
		t.Errorf("Got %d tasks, want 3", len(tasks))
	}

	// Verify pending tasks were cleared
	if len(session.pendingTasks) != 0 {
		t.Errorf("Pending tasks not cleared, count = %d", len(session.pendingTasks))
	}

	// Test interactive mode (should filter completed tasks)
	session.SetInteractive(true)
	session.AddTask(task1)
	session.AddTask(task2)
	session.AddTask(task3)

	// Mark task2 as completed
	session.completedTasks["task-2"] = task2

	tasks = session.GetPendingTasks()

	// Should return only uncompleted tasks
	if len(tasks) != 2 {
		t.Errorf("Got %d tasks in interactive mode, want 2", len(tasks))
	}

	// Verify correct tasks were returned
	for _, task := range tasks {
		if task.ID == "task-2" {
			t.Error("Completed task was returned in interactive mode")
		}
	}

	// Verify pending tasks were updated in interactive mode (only uncompleted tasks remain)
	if len(session.pendingTasks) != 2 {
		t.Errorf("Pending tasks count in interactive mode = %d, want 2", len(session.pendingTasks))
	}
}

func TestSession_AddTaskResult(t *testing.T) {
	session := NewSession("test", NewMockAgentConnection())

	// First add tasks to pending
	task1 := protocol.Task{
		TaskID: "task-1",
		ID:     "task-1",
		Type:   "command",
		Status: "pending",
	}
	task2 := protocol.Task{
		TaskID: "task-2",
		ID:     "task-2",
		Type:   "command",
		Status: "pending",
	}
	session.AddTask(task1)
	session.AddTask(task2)

	result1 := protocol.TaskResult{
		TaskID:   "task-1",
		Output:   "command output",
		ExitCode: 0,
		Time:     time.Now(),
	}

	result2 := protocol.TaskResult{
		TaskID:   "task-2",
		Output:   "error output",
		ExitCode: 1,
		Error:    "permission denied",
		Time:     time.Now(),
	}

	// Add results
	session.AddTaskResult(result1)
	session.AddTaskResult(result2)

	// Verify results were added
	if len(session.taskResults) != 2 {
		t.Errorf("Task results count = %d, want 2", len(session.taskResults))
	}

	// Verify completed tasks were marked
	if _, exists := session.completedTasks["task-1"]; !exists {
		t.Error("task-1 not marked as completed")
	}
	if _, exists := session.completedTasks["task-2"]; !exists {
		t.Error("task-2 not marked as completed")
	}

	// Verify result order
	if session.taskResults[0].TaskID != "task-1" {
		t.Errorf("First result TaskID = %s, want task-1", session.taskResults[0].TaskID)
	}
	if session.taskResults[1].TaskID != "task-2" {
		t.Errorf("Second result TaskID = %s, want task-2", session.taskResults[1].TaskID)
	}

	// Verify pending tasks were cleared
	if len(session.pendingTasks) != 0 {
		t.Errorf("Pending tasks not cleared, count = %d", len(session.pendingTasks))
	}
}

func TestSession_GetTaskResults(t *testing.T) {
	session := NewSession("test", NewMockAgentConnection())

	// Test with no results
	results := session.GetTaskResults()
	if len(results) != 0 {
		t.Errorf("Empty session returned %d results", len(results))
	}

	// Add results
	result1 := protocol.TaskResult{TaskID: "task-1", Output: "output1", Time: time.Now()}
	result2 := protocol.TaskResult{TaskID: "task-2", Output: "output2", Time: time.Now()}

	session.AddTaskResult(result1)
	session.AddTaskResult(result2)

	// Get results
	results = session.GetTaskResults()

	if len(results) != 2 {
		t.Errorf("Got %d results, want 2", len(results))
	}

	// Verify results are the same
	if results[0].TaskID != "task-1" || results[1].TaskID != "task-2" {
		t.Error("Results order mismatch")
	}
}

func TestSession_ExecuteCommand(t *testing.T) {
	mockConn := NewMockAgentConnection()
	session := NewSession("test", mockConn)

	// Execute command - cmdType is the type, payload is the data
	payload := map[string]interface{}{
		"command": "ls -la /tmp",
	}
	id, err := session.ExecuteCommand("command", payload)
	if err != nil {
		t.Errorf("ExecuteCommand failed: %v", err)
	}

	if id == "" {
		t.Error("ExecuteCommand returned empty task ID")
	}

	// Get pending tasks to verify the command was added
	// Note: GetPendingTasks clears the pending tasks in non-interactive mode
	tasks := session.GetPendingTasks()
	if len(tasks) != 1 {
		t.Errorf("Pending tasks count = %d, want 1", len(tasks))
	}

	// Verify task structure
	task := tasks[0]

	// GetPendingTasks converts "command" to "shell"
	if task.Type != "shell" {
		t.Errorf("Task type = %s, want shell", task.Type)
	}

	// Check payload
	if task.Payload == nil {
		t.Error("Task payload is nil")
	} else if cmd, ok := task.Payload.(map[string]interface{})["command"]; !ok || cmd != "ls -la /tmp" {
		t.Errorf("Task payload command = %v, want 'ls -la /tmp'", cmd)
	}

	// Verify task ID was generated
	if task.ID == "" {
		t.Error("Task ID is empty")
	}

	// Verify timestamp
	if task.CreatedAt.IsZero() {
		t.Error("Task CreatedAt is zero")
	}
}

func TestSession_ConcurrentOperations(t *testing.T) {
	session := NewSession("test", NewMockAgentConnection())

	// Run concurrent operations
	var wg sync.WaitGroup
	numGoroutines := 10
	numOperations := 100

	wg.Add(numGoroutines * 4)

	// Concurrent task additions
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				task := protocol.Task{
					ID:        fmt.Sprintf("task-%d-%d", id, j),
					Type:      "command",
					Status:    "pending",
					CreatedAt: time.Now(),
				}
				session.AddTask(task)
			}
		}(i)
	}

	// Concurrent info updates
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				info := map[string]interface{}{
					fmt.Sprintf("key-%d", id): fmt.Sprintf("value-%d", j),
				}
				session.UpdateInformation(info)
			}
		}(i)
	}

	// Concurrent result additions
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				result := protocol.TaskResult{
					TaskID: fmt.Sprintf("result-%d-%d", id, j),
					Output: "test output",
					Time:   time.Now(),
				}
				session.AddTaskResult(result)
			}
		}(i)
	}

	// Concurrent reads
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				session.GetPendingTasks()
				session.GetTaskResults()
				session.Information()
			}
		}()
	}

	wg.Wait()

	// Verify session is still in valid state
	if session.ID != "test" {
		t.Error("Session ID corrupted after concurrent access")
	}

	// Verify we have some data
	if len(session.information) == 0 {
		t.Error("No information stored after concurrent updates")
	}

	if len(session.taskResults) == 0 {
		t.Error("No task results stored after concurrent additions")
	}
}

func TestSession_LastSeenUpdate(t *testing.T) {
	session := NewSession("test", NewMockAgentConnection())
	originalLastSeen := session.lastSeen

	// Wait a bit
	time.Sleep(10 * time.Millisecond)

	// Operations that should update lastSeen
	session.UpdateConnection(NewMockAgentConnection())
	if !session.lastSeen.After(originalLastSeen) {
		t.Error("UpdateConnection did not update lastSeen")
	}

	// Get current lastSeen
	currentLastSeen := session.lastSeen
	time.Sleep(10 * time.Millisecond)

	// UpdateInformation should also update lastSeen
	info := map[string]interface{}{"test": "value"}
	session.UpdateInformation(info)

	// UpdateInformation updates lastSeen
	if !session.lastSeen.After(currentLastSeen) {
		t.Error("UpdateInformation did not update lastSeen")
	}
}

// Benchmarks
func BenchmarkAddTask(b *testing.B) {
	session := NewSession("bench", NewMockAgentConnection())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		task := protocol.Task{
			ID:        fmt.Sprintf("task-%d", i),
			Type:      "command",
			Status:    "pending",
			CreatedAt: time.Now(),
		}
		session.AddTask(task)
	}
}

func BenchmarkGetPendingTasks(b *testing.B) {
	session := NewSession("bench", NewMockAgentConnection())

	// Pre-populate with tasks
	for i := 0; i < 1000; i++ {
		task := protocol.Task{
			ID:        fmt.Sprintf("task-%d", i),
			Type:      "command",
			Status:    "pending",
			CreatedAt: time.Now(),
		}
		session.AddTask(task)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		session.GetPendingTasks()
	}
}

func BenchmarkConcurrentAccess(b *testing.B) {
	session := NewSession("bench", NewMockAgentConnection())

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			switch i % 4 {
			case 0:
				task := protocol.Task{
					ID:        fmt.Sprintf("task-%d", i),
					Type:      "command",
					Status:    "pending",
					CreatedAt: time.Now(),
				}
				session.AddTask(task)
			case 1:
				session.GetPendingTasks()
			case 2:
				info := map[string]interface{}{
					"key": fmt.Sprintf("value-%d", i),
				}
				session.UpdateInformation(info)
			case 3:
				session.Information()
			}
			i++
		}
	})
}
