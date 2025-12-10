package session

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/r74tech/virga/internal/server/protocol"
)

func TestNewManager(t *testing.T) {
	manager := NewManager()

	if manager == nil {
		t.Fatal("NewManager returned nil")
	}

	if manager.sessions == nil {
		t.Error("Sessions map is nil")
	}

	// Verify empty manager
	if count := manager.GetTotalSessionCount(); count != 0 {
		t.Errorf("Initial session count = %d, want 0", count)
	}
}

func TestManager_AddSession(t *testing.T) {
	manager := NewManager()
	mockConn := &MockAgentConnection{}

	// Add session
	session := NewSession("session-1", mockConn)
	manager.AddSession(session)

	if session == nil {
		t.Fatal("Session is nil")
	}

	if session.ID != "session-1" {
		t.Errorf("Session ID = %s, want session-1", session.ID)
	}

	// Verify session was added
	if count := manager.GetTotalSessionCount(); count != 1 {
		t.Errorf("Session count = %d, want 1", count)
	}

	// Add another session
	session2 := NewSession("session-2", mockConn)
	manager.AddSession(session2)

	if session2.ID != "session-2" {
		t.Errorf("Session2 ID = %s, want session-2", session2.ID)
	}

	if count := manager.GetTotalSessionCount(); count != 2 {
		t.Errorf("Session count = %d, want 2", count)
	}

	// Try to add duplicate session ID
	session3 := NewSession("session-1", mockConn)
	manager.AddSession(session3)

	// totalSessions increments even for duplicates (by design)
	if count := manager.GetTotalSessionCount(); count != 3 {
		t.Errorf("Session count after duplicate = %d, want 3", count)
	}

	// But actual session map should still have 2 unique sessions
	if len(manager.sessions) != 2 {
		t.Errorf("Actual session map size = %d, want 2", len(manager.sessions))
	}
}

func TestManager_GetSession(t *testing.T) {
	manager := NewManager()
	mockConn := &MockAgentConnection{}

	// Test getting non-existent session
	session := manager.GetSession("non-existent")
	if session != nil {
		t.Error("GetSession returned non-nil for non-existent session")
	}

	// Add session
	addedSession := NewSession("test-session", mockConn)
	manager.AddSession(addedSession)

	// Get session
	retrievedSession := manager.GetSession("test-session")

	if retrievedSession == nil {
		t.Fatal("GetSession returned nil for existing session")
	}

	if retrievedSession != addedSession {
		t.Error("Retrieved session is not the same as added session")
	}

	if retrievedSession.ID != "test-session" {
		t.Errorf("Retrieved session ID = %s, want test-session", retrievedSession.ID)
	}
}

func TestManager_RemoveSession(t *testing.T) {
	manager := NewManager()
	mockConn := &MockAgentConnection{}

	// Add sessions
	session1 := NewSession("session-1", mockConn)
	manager.AddSession(session1)
	session2 := NewSession("session-2", mockConn)
	manager.AddSession(session2)
	session3 := NewSession("session-3", mockConn)
	manager.AddSession(session3)

	// Verify initial count
	if count := manager.GetTotalSessionCount(); count != 3 {
		t.Errorf("Initial session count = %d, want 3", count)
	}

	// Remove session
	manager.RemoveSession("session-2")

	// Verify count should stay the same (total count doesn't decrease)
	if count := manager.GetTotalSessionCount(); count != 3 {
		t.Errorf("Session count after removal = %d, want 3", count)
	}

	// Verify session was removed
	if session := manager.GetSession("session-2"); session != nil {
		t.Error("Removed session still exists")
	}

	// Verify other sessions still exist
	if session := manager.GetSession("session-1"); session == nil {
		t.Error("session-1 was removed incorrectly")
	}
	if session := manager.GetSession("session-3"); session == nil {
		t.Error("session-3 was removed incorrectly")
	}

	// Remove non-existent session (should not panic)
	manager.RemoveSession("non-existent")

	if count := manager.GetTotalSessionCount(); count != 3 {
		t.Errorf("Session count after removing non-existent = %d, want 3", count)
	}
}

func TestManager_GetActiveSessions(t *testing.T) {
	manager := NewManager()
	mockConn := &MockAgentConnection{}

	// Test with no sessions
	sessions := manager.GetActiveSessions()
	if len(sessions) != 0 {
		t.Errorf("Empty manager returned %d sessions", len(sessions))
	}

	// Add sessions
	session1 := NewSession("session-1", mockConn)
	manager.AddSession(session1)
	session2 := NewSession("session-2", mockConn)
	manager.AddSession(session2)
	session3 := NewSession("session-3", mockConn)
	manager.AddSession(session3)

	// Set different statuses
	session1.Status = "active"
	session2.Status = "disconnected"
	session3.Status = "active"

	// Get active sessions (returns only active sessions in current implementation)
	sessions = manager.GetActiveSessions()

	if len(sessions) != 2 {
		t.Errorf("GetActiveSessions returned %d sessions, want 2", len(sessions))
	}

	// Verify correct sessions are returned
	foundSessions := make(map[string]bool)
	for _, session := range sessions {
		foundSessions[session.ID] = true
	}

	if !foundSessions["session-1"] {
		t.Error("session-1 not found in active sessions")
	}
	if foundSessions["session-2"] {
		t.Error("session-2 (disconnected) found in active sessions")
	}
	if !foundSessions["session-3"] {
		t.Error("session-3 not found in active sessions")
	}
}

func TestManager_AddTask(t *testing.T) {
	manager := NewManager()
	mockConn := &MockAgentConnection{}

	// Test adding task to non-existent session
	task := &protocol.Task{
		ID:        "task-1",
		Type:      protocol.TaskTypeCommand,
		Command:   "whoami",
		CreatedAt: time.Now(),
		Status:    protocol.TaskStatusPending,
	}

	err := manager.AddTask("non-existent", task)
	if err == nil {
		t.Error("Expected error when adding task to non-existent session")
	}

	// Add session
	session := NewSession("test-session", mockConn)
	manager.AddSession(session)

	// Add task
	err = manager.AddTask("test-session", task)
	if err != nil {
		t.Errorf("Failed to add task: %v", err)
	}

	// Verify task was added to session
	pendingTasks := session.GetPendingTasks()
	if len(pendingTasks) != 1 {
		t.Errorf("Pending tasks count = %d, want 1", len(pendingTasks))
	}

	if pendingTasks[0].ID != "task-1" {
		t.Errorf("Task ID = %s, want task-1", pendingTasks[0].ID)
	}
}

func TestManager_GetSessionsByBeacon(t *testing.T) {
	manager := NewManager()
	mockConn := &MockAgentConnection{}

	// Add sessions with different beacon IDs
	session1 := NewSession("session-1", mockConn)
	session1.BeaconID = "beacon-A"
	manager.AddSession(session1)

	session2 := NewSession("session-2", mockConn)
	session2.BeaconID = "beacon-B"
	manager.AddSession(session2)

	session3 := NewSession("session-3", mockConn)
	session3.BeaconID = "beacon-A"
	manager.AddSession(session3)

	session4 := NewSession("session-4", mockConn)
	session4.BeaconID = "beacon-C"
	manager.AddSession(session4)

	// Get sessions for beacon-A
	beaconASessions := manager.GetSessionsByBeacon("beacon-A")
	if len(beaconASessions) != 2 {
		t.Errorf("beacon-A sessions count = %d, want 2", len(beaconASessions))
	}

	// Verify correct sessions
	foundSession1 := false
	foundSession3 := false
	for _, session := range beaconASessions {
		if session.ID == "session-1" {
			foundSession1 = true
		}
		if session.ID == "session-3" {
			foundSession3 = true
		}
	}

	if !foundSession1 {
		t.Error("session-1 not found in beacon-A sessions")
	}
	if !foundSession3 {
		t.Error("session-3 not found in beacon-A sessions")
	}

	// Get sessions for beacon-B
	beaconBSessions := manager.GetSessionsByBeacon("beacon-B")
	if len(beaconBSessions) != 1 {
		t.Errorf("beacon-B sessions count = %d, want 1", len(beaconBSessions))
	}

	if beaconBSessions[0].ID != "session-2" {
		t.Errorf("beacon-B session ID = %s, want session-2", beaconBSessions[0].ID)
	}

	// Get sessions for non-existent beacon
	nonExistentSessions := manager.GetSessionsByBeacon("non-existent")
	if len(nonExistentSessions) != 0 {
		t.Errorf("Non-existent beacon returned %d sessions", len(nonExistentSessions))
	}
}

func TestManager_ConcurrentOperations(t *testing.T) {
	manager := NewManager()
	mockConn := &MockAgentConnection{}

	var wg sync.WaitGroup
	numGoroutines := 10
	numOperations := 100

	wg.Add(numGoroutines * 5)

	// Concurrent session additions
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				sessionID := fmt.Sprintf("session-%d-%d", id, j)
				session := NewSession(sessionID, mockConn)
				manager.AddSession(session)
			}
		}(i)
	}

	// Concurrent session retrievals
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				sessionID := fmt.Sprintf("session-%d-%d", id, j)
				manager.GetSession(sessionID)
			}
		}(i)
	}

	// Concurrent active session listings
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				manager.GetActiveSessions()
			}
		}()
	}

	// Concurrent task additions
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				sessionID := fmt.Sprintf("session-%d-%d", id, j)
				task := &protocol.Task{
					ID:        fmt.Sprintf("task-%d-%d", id, j),
					Type:      protocol.TaskTypeCommand,
					CreatedAt: time.Now(),
					Status:    protocol.TaskStatusPending,
				}
				manager.AddTask(sessionID, task)
			}
		}(i)
	}

	// Concurrent session removals
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations/2; j++ { // Remove half
				sessionID := fmt.Sprintf("session-%d-%d", id, j)
				manager.RemoveSession(sessionID)
			}
		}(i)
	}

	wg.Wait()

	// Verify manager is still in valid state
	count := manager.GetTotalSessionCount()
	t.Logf("Final session count: %d", count)

	// The total session count tracks all sessions ever created, not current active
	expectedMin := numGoroutines * numOperations
	if count < expectedMin {
		t.Errorf("Final session count %d less than expected minimum %d", count, expectedMin)
	}
}

func TestManager_SessionUpdateOperations(t *testing.T) {
	manager := NewManager()
	mockConn := &MockAgentConnection{}

	// Add session
	sessionID := "update-test"
	session := NewSession(sessionID, mockConn)
	manager.AddSession(session)

	// Update session information through manager
	info := map[string]interface{}{
		"hostname": "test-host",
		"username": "test-user",
		"os":       "linux",
	}

	// Get session and update
	retrievedSession := manager.GetSession(sessionID)
	if retrievedSession == nil {
		t.Fatal("Failed to retrieve session")
	}

	retrievedSession.UpdateInformation(info)

	// Verify updates persist
	session2 := manager.GetSession(sessionID)
	sessionInfo := session2.Information()
	if hostname := sessionInfo["hostname"]; hostname != "test-host" {
		t.Errorf("Hostname = %v, want test-host", hostname)
	}

	// Add tasks through manager
	task1 := &protocol.Task{
		ID:        "task-1",
		Type:      protocol.TaskTypeCommand,
		Command:   "whoami",
		CreatedAt: time.Now(),
		Status:    protocol.TaskStatusPending,
	}
	task2 := &protocol.Task{
		ID:        "task-2",
		Type:      protocol.TaskTypeFileList,
		CreatedAt: time.Now(),
		Status:    protocol.TaskStatusPending,
	}

	err := manager.AddTask(sessionID, task1)
	if err != nil {
		t.Errorf("Failed to add task1: %v", err)
	}
	err = manager.AddTask(sessionID, task2)
	if err != nil {
		t.Errorf("Failed to add task2: %v", err)
	}

	// Verify tasks were added
	tasks := session2.GetPendingTasks()
	if len(tasks) != 2 {
		t.Errorf("Pending tasks = %d, want 2", len(tasks))
	}
}

func TestManager_MultipleBeaconSessions(t *testing.T) {
	manager := NewManager()
	mockConn := &MockAgentConnection{}

	// Simulate multiple sessions per beacon
	beaconSessions := map[string][]string{
		"beacon-1": {"s1-1", "s1-2", "s1-3"},
		"beacon-2": {"s2-1", "s2-2"},
		"beacon-3": {"s3-1"},
	}

	// Add all sessions
	for beaconID, sessionIDs := range beaconSessions {
		for _, sessionID := range sessionIDs {
			session := NewSession(sessionID, mockConn)
			session.BeaconID = beaconID
			manager.AddSession(session)
		}
	}

	// Verify total count
	if count := manager.GetTotalSessionCount(); count != 6 {
		t.Errorf("Total session count = %d, want 6", count)
	}

	// Verify beacon grouping
	for beaconID, expectedSessions := range beaconSessions {
		sessions := manager.GetSessionsByBeacon(beaconID)
		if len(sessions) != len(expectedSessions) {
			t.Errorf("Beacon %s has %d sessions, want %d", beaconID, len(sessions), len(expectedSessions))
		}

		// Verify session IDs match
		sessionIDMap := make(map[string]bool)
		for _, session := range sessions {
			sessionIDMap[session.ID] = true
		}

		for _, expectedID := range expectedSessions {
			if !sessionIDMap[expectedID] {
				t.Errorf("Session %s not found for beacon %s", expectedID, beaconID)
			}
		}
	}
}

// Integration test simulating real-world usage
func TestManager_IntegrationScenario(t *testing.T) {
	manager := NewManager()

	// Simulate agent connections
	agent1Conn := &MockAgentConnection{}
	agent2Conn := &MockAgentConnection{}

	// Agent 1 connects
	session1 := NewSession("agent-1", agent1Conn)
	session1.BeaconID = "http-beacon"
	manager.AddSession(session1)
	session1.UpdateInformation(map[string]interface{}{
		"hostname": "workstation-1",
		"username": "user1",
		"os":       "windows",
		"arch":     "amd64",
	})

	// Agent 2 connects
	session2 := NewSession("agent-2", agent2Conn)
	session2.BeaconID = "https-beacon"
	manager.AddSession(session2)
	session2.UpdateInformation(map[string]interface{}{
		"hostname": "server-1",
		"username": "admin",
		"os":       "linux",
		"arch":     "amd64",
	})

	// Send tasks to agents
	cmdTask1 := &protocol.Task{
		ID:        "cmd-1",
		Type:      protocol.TaskTypeCommand,
		Command:   "whoami",
		Arguments: map[string]interface{}{"command": "whoami"},
		Status:    protocol.TaskStatusPending,
		CreatedAt: time.Now(),
	}

	fileTask := &protocol.Task{
		ID:        "file-1",
		Type:      protocol.TaskTypeFileList,
		Arguments: map[string]interface{}{"path": "/tmp"},
		Status:    protocol.TaskStatusPending,
		CreatedAt: time.Now(),
	}

	err := manager.AddTask("agent-1", cmdTask1)
	if err != nil {
		t.Errorf("Failed to add cmdTask1: %v", err)
	}
	err = manager.AddTask("agent-2", fileTask)
	if err != nil {
		t.Errorf("Failed to add fileTask: %v", err)
	}

	// Simulate task execution and results
	time.Sleep(10 * time.Millisecond)

	result1 := protocol.TaskResult{
		TaskID:   "cmd-1",
		Output:   "user1\\workstation-1",
		ExitCode: 0,
		Time:     time.Now(),
	}

	result2 := protocol.TaskResult{
		TaskID:   "file-1",
		Output:   "file1.txt\nfile2.log\ndir1/",
		ExitCode: 0,
		Time:     time.Now(),
	}

	session1.AddTaskResult(result1)
	session2.AddTaskResult(result2)

	// Verify final state
	activeSessions := manager.GetActiveSessions()
	if len(activeSessions) != 2 {
		t.Errorf("Active sessions = %d, want 2", len(activeSessions))
	}

	// Verify task results
	results1 := session1.GetTaskResults()
	if len(results1) != 1 {
		t.Errorf("Agent-1 results = %d, want 1", len(results1))
	}

	results2 := session2.GetTaskResults()
	if len(results2) != 1 {
		t.Errorf("Agent-2 results = %d, want 1", len(results2))
	}

	// Simulate agent disconnect
	manager.RemoveSession("agent-1")

	// Check remaining active sessions
	remainingSessions := manager.GetActiveSessions()
	if len(remainingSessions) != 1 {
		t.Errorf("Remaining active sessions = %d, want 1", len(remainingSessions))
	}
}

// Benchmarks
func BenchmarkManager_AddSession(b *testing.B) {
	manager := NewManager()
	mockConn := &MockAgentConnection{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sessionID := fmt.Sprintf("session-%d", i)
		session := NewSession(sessionID, mockConn)
		manager.AddSession(session)
	}
}

func BenchmarkManager_GetSession(b *testing.B) {
	manager := NewManager()
	mockConn := &MockAgentConnection{}

	// Pre-populate sessions
	for i := 0; i < 1000; i++ {
		sessionID := fmt.Sprintf("session-%d", i)
		session := NewSession(sessionID, mockConn)
		manager.AddSession(session)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sessionID := fmt.Sprintf("session-%d", i%1000)
		manager.GetSession(sessionID)
	}
}

func BenchmarkManager_GetActiveSessions(b *testing.B) {
	manager := NewManager()
	mockConn := &MockAgentConnection{}

	// Pre-populate sessions
	for i := 0; i < 100; i++ {
		sessionID := fmt.Sprintf("session-%d", i)
		session := NewSession(sessionID, mockConn)
		manager.AddSession(session)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		manager.GetActiveSessions()
	}
}

func BenchmarkManager_ConcurrentAccess(b *testing.B) {
	manager := NewManager()
	mockConn := &MockAgentConnection{}

	// Pre-populate some sessions
	for i := 0; i < 100; i++ {
		sessionID := fmt.Sprintf("session-%d", i)
		session := NewSession(sessionID, mockConn)
		manager.AddSession(session)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			sessionID := fmt.Sprintf("session-%d", i%100)

			switch i % 4 {
			case 0:
				manager.GetSession(sessionID)
			case 1:
				manager.GetActiveSessions()
			case 2:
				task := &protocol.Task{
					ID:        fmt.Sprintf("task-%d", i),
					Type:      protocol.TaskTypeCommand,
					CreatedAt: time.Now(),
					Status:    protocol.TaskStatusPending,
				}
				manager.AddTask(sessionID, task)
			case 3:
				manager.GetSessionsByBeacon(fmt.Sprintf("beacon-%d", i%10))
			}
			i++
		}
	})
}
