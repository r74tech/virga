package memdb

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

// TestNew tests creating a new database instance
func TestNew(t *testing.T) {
	db, err := New()
	if err != nil {
		t.Fatalf("Failed to create new DB: %v", err)
	}

	if db == nil {
		t.Fatal("Expected non-nil DB")
	}

	if db.db == nil {
		t.Fatal("Expected non-nil internal memdb")
	}
}

// TestStoreCommandResult tests storing command results
func TestStoreCommandResult(t *testing.T) {
	db, err := New()
	if err != nil {
		t.Fatalf("Failed to create DB: %v", err)
	}

	tests := []struct {
		name     string
		cmd      string
		args     []string
		output   string
		errMsg   string
		exitCode int
		module   string
	}{
		{
			name:     "Successful command",
			cmd:      "echo",
			args:     []string{"hello", "world"},
			output:   "hello world\n",
			errMsg:   "",
			exitCode: 0,
			module:   "shell",
		},
		{
			name:     "Failed command",
			cmd:      "invalid_cmd",
			args:     []string{},
			output:   "",
			errMsg:   "command not found",
			exitCode: 127,
			module:   "shell",
		},
		{
			name:     "Command with no args",
			cmd:      "pwd",
			args:     nil,
			output:   "/home/user\n",
			errMsg:   "",
			exitCode: 0,
			module:   "shell",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := db.StoreCommandResult(tt.cmd, tt.args, tt.output, tt.errMsg, tt.exitCode, tt.module)
			if err != nil {
				t.Errorf("Failed to store command result: %v", err)
			}
		})
	}

	// Verify stored results
	results, err := db.GetRecentCommands(10)
	if err != nil {
		t.Fatalf("Failed to get recent commands: %v", err)
	}

	if len(results) != len(tests) {
		t.Errorf("Expected %d results, got %d", len(tests), len(results))
	}
}

// TestStoreServerTask tests storing and updating server tasks
func TestStoreServerTask(t *testing.T) {
	db, err := New()
	if err != nil {
		t.Fatalf("Failed to create DB: %v", err)
	}

	// Store a task
	task, err := db.StoreServerTask("shell", "echo test")
	if err != nil {
		t.Fatalf("Failed to store server task: %v", err)
	}

	if task.ID == "" {
		t.Error("Expected non-empty task ID")
	}
	if task.Status != "pending" {
		t.Errorf("Expected status 'pending', got '%s'", task.Status)
	}

	// Update task status
	err = db.UpdateServerTask(task.ID, "completed", "test\n", "")
	if err != nil {
		t.Errorf("Failed to update server task: %v", err)
	}

	// Verify update
	pending, err := db.GetPendingTasks()
	if err != nil {
		t.Fatalf("Failed to get pending tasks: %v", err)
	}

	if len(pending) != 0 {
		t.Errorf("Expected 0 pending tasks after update, got %d", len(pending))
	}
}

// TestUpdateServerTaskNotFound tests updating non-existent task
func TestUpdateServerTaskNotFound(t *testing.T) {
	db, err := New()
	if err != nil {
		t.Fatalf("Failed to create DB: %v", err)
	}

	err = db.UpdateServerTask("non-existent-id", "completed", "", "")
	if err == nil {
		t.Error("Expected error for non-existent task ID")
	}
}

// TestStoreLlamaInteraction tests storing Llama interactions
func TestStoreLlamaInteraction(t *testing.T) {
	db, err := New()
	if err != nil {
		t.Fatalf("Failed to create DB: %v", err)
	}

	tests := []struct {
		name        string
		taskID      string
		prompt      string
		response    string
		taskType    string
		temperature float32
		commands    []string
	}{
		{
			name:        "Basic interaction",
			taskID:      "llama-1",
			prompt:      "List files",
			response:    "I'll list the files. [EXECUTE: ls -la]",
			taskType:    "llama_task",
			temperature: 0.7,
			commands:    []string{"ls -la"},
		},
		{
			name:        "Multiple commands",
			taskID:      "llama-2",
			prompt:      "System info",
			response:    "Getting system info...",
			taskType:    "llama_autonomous",
			temperature: 0.3,
			commands:    []string{"uname -a", "whoami", "pwd"},
		},
		{
			name:        "Update existing",
			taskID:      "llama-1", // Same as first
			prompt:      "List files",
			response:    "Updated response with more info",
			taskType:    "llama_task",
			temperature: 0.7,
			commands:    []string{"ls -la", "pwd"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := db.StoreLlamaInteraction(tt.taskID, tt.prompt, tt.response, tt.taskType, tt.temperature, tt.commands)
			if err != nil {
				t.Errorf("Failed to store llama interaction: %v", err)
			}
		})
	}

	// Verify stored interactions
	interactions, err := db.GetRecentLlamaInteractions(10)
	if err != nil {
		t.Fatalf("Failed to get recent llama interactions: %v", err)
	}

	// Should have 2 unique interactions (one was updated)
	if len(interactions) != 2 {
		t.Errorf("Expected 2 interactions, got %d", len(interactions))
	}

	// Check the updated interaction
	for _, interaction := range interactions {
		if interaction.TaskID == "llama-1" {
			if interaction.Response != "Updated response with more info" {
				t.Error("Expected interaction to be updated")
			}
			if len(interaction.CommandsIssued) != 2 {
				t.Errorf("Expected 2 commands in updated interaction, got %d", len(interaction.CommandsIssued))
			}
		}
	}
}

// TestStoreLlamaTaskStart tests storing the start of a Llama task
func TestStoreLlamaTaskStart(t *testing.T) {
	db, err := New()
	if err != nil {
		t.Fatalf("Failed to create DB: %v", err)
	}

	taskID := "background-task-1"
	err = db.StoreLlamaTaskStart(taskID, "Background analysis", "llama_autonomous", 0.5)
	if err != nil {
		t.Fatalf("Failed to store llama task start: %v", err)
	}

	// Get the interaction
	interactions, err := db.GetRecentLlamaInteractions(1)
	if err != nil {
		t.Fatalf("Failed to get interactions: %v", err)
	}

	if len(interactions) != 1 {
		t.Fatalf("Expected 1 interaction, got %d", len(interactions))
	}

	interaction := interactions[0]
	if interaction.TaskID != taskID {
		t.Errorf("Expected task ID %s, got %s", taskID, interaction.TaskID)
	}
	if interaction.Status != "running" {
		t.Errorf("Expected status 'running', got '%s'", interaction.Status)
	}
	if interaction.Response != "" {
		t.Error("Expected empty response for task start")
	}
}

// TestStoreSystemInfo tests storing system information
func TestStoreSystemInfo(t *testing.T) {
	db, err := New()
	if err != nil {
		t.Fatalf("Failed to create DB: %v", err)
	}

	diskUsage := map[string]uint64{
		"/":     1000000,
		"/home": 500000,
	}

	networkIO := map[string]uint64{
		"eth0_rx": 1000,
		"eth0_tx": 2000,
	}

	err = db.StoreSystemInfo(45.5, 8192, diskUsage, networkIO, 150)
	if err != nil {
		t.Fatalf("Failed to store system info: %v", err)
	}

	// Get system info history
	history, err := db.GetSystemInfoHistory(time.Now().Add(-1 * time.Hour))
	if err != nil {
		t.Fatalf("Failed to get system info history: %v", err)
	}

	if len(history) != 1 {
		t.Errorf("Expected 1 system info entry, got %d", len(history))
	}

	info := history[0]
	if info.CPUUsage != 45.5 {
		t.Errorf("Expected CPU usage 45.5, got %f", info.CPUUsage)
	}
	if info.MemoryUsage != 8192 {
		t.Errorf("Expected memory usage 8192, got %d", info.MemoryUsage)
	}
	if info.Processes != 150 {
		t.Errorf("Expected 150 processes, got %d", info.Processes)
	}
}

// TestGetRecentCommands tests retrieving recent commands with limit
func TestGetRecentCommands(t *testing.T) {
	db, err := New()
	if err != nil {
		t.Fatalf("Failed to create DB: %v", err)
	}

	// Store multiple commands
	for i := 0; i < 10; i++ {
		err := db.StoreCommandResult("echo", []string{string(rune('A' + i))}, "output", "", 0, "shell")
		if err != nil {
			t.Fatalf("Failed to store command %d: %v", i, err)
		}
		time.Sleep(10 * time.Millisecond) // Ensure different timestamps
	}

	// Test with limit
	results, err := db.GetRecentCommands(5)
	if err != nil {
		t.Fatalf("Failed to get recent commands: %v", err)
	}

	if len(results) != 5 {
		t.Errorf("Expected 5 results with limit, got %d", len(results))
	}

	// Verify they are in reverse chronological order
	for i := 1; i < len(results); i++ {
		if results[i-1].StartTime.Before(results[i].StartTime) {
			t.Error("Results not in reverse chronological order")
		}
	}
}

// TestGetPendingTasks tests retrieving pending tasks
func TestGetPendingTasks(t *testing.T) {
	db, err := New()
	if err != nil {
		t.Fatalf("Failed to create DB: %v", err)
	}

	// Store tasks with different statuses
	statuses := []string{"pending", "pending", "completed", "failed", "pending"}
	var taskIDs []string

	for i, status := range statuses {
		task, err := db.StoreServerTask("test", "payload")
		if err != nil {
			t.Fatalf("Failed to store task %d: %v", i, err)
		}
		taskIDs = append(taskIDs, task.ID)

		if status != "pending" {
			err = db.UpdateServerTask(task.ID, status, "", "")
			if err != nil {
				t.Fatalf("Failed to update task %d: %v", i, err)
			}
		}
	}

	// Get pending tasks
	pending, err := db.GetPendingTasks()
	if err != nil {
		t.Fatalf("Failed to get pending tasks: %v", err)
	}

	if len(pending) != 3 {
		t.Errorf("Expected 3 pending tasks, got %d", len(pending))
	}
}

// TestClearOldData tests removing old data
func TestClearOldData(t *testing.T) {
	db, err := New()
	if err != nil {
		t.Fatalf("Failed to create DB: %v", err)
	}

	// Store some data
	for i := 0; i < 5; i++ {
		db.StoreCommandResult("echo", []string{"test"}, "output", "", 0, "shell")
		db.StoreSystemInfo(float64(i), uint64(i*1024), nil, nil, i*10)
		db.StoreLlamaInteraction("task-"+string(rune('A'+i)), "prompt", "response", "test", 0.5, nil)
	}

	// Clear data older than 1 second
	time.Sleep(1100 * time.Millisecond)

	// Store some new data
	db.StoreCommandResult("echo", []string{"new"}, "new output", "", 0, "shell")
	db.StoreSystemInfo(99.9, 9999, nil, nil, 999)
	db.StoreLlamaInteraction("new-task", "new prompt", "new response", "test", 0.7, nil)

	// Clear old data
	err = db.ClearOldData(1 * time.Second)
	if err != nil {
		t.Fatalf("Failed to clear old data: %v", err)
	}

	// Verify only new data remains
	commands, _ := db.GetRecentCommands(10)
	if len(commands) != 1 {
		t.Errorf("Expected 1 command after clear, got %d", len(commands))
	}

	sysInfo, _ := db.GetSystemInfoHistory(time.Now().Add(-2 * time.Hour))
	if len(sysInfo) != 1 {
		t.Errorf("Expected 1 system info after clear, got %d", len(sysInfo))
	}

	llama, _ := db.GetRecentLlamaInteractions(10)
	if len(llama) != 1 {
		t.Errorf("Expected 1 llama interaction after clear, got %d", len(llama))
	}
}

// TestStats tests database statistics
func TestStats(t *testing.T) {
	db, err := New()
	if err != nil {
		t.Fatalf("Failed to create DB: %v", err)
	}

	// Store various data
	db.StoreCommandResult("echo", []string{"test"}, "output", "", 0, "shell")
	db.StoreCommandResult("ls", []string{"-la"}, "files", "", 0, "shell")

	db.StoreServerTask("task1", "payload1")
	db.StoreServerTask("task2", "payload2")
	db.StoreServerTask("task3", "payload3")

	db.StoreLlamaInteraction("llama1", "prompt1", "response1", "test", 0.5, nil)

	db.StoreSystemInfo(50.0, 4096, nil, nil, 100)
	db.StoreSystemInfo(60.0, 5120, nil, nil, 110)

	// Get stats
	stats, err := db.Stats()
	if err != nil {
		t.Fatalf("Failed to get stats: %v", err)
	}

	expected := map[string]int{
		"command_results":    2,
		"server_tasks":       3,
		"llama_interactions": 1,
		"system_info":        2,
	}

	for key, expectedCount := range expected {
		if stats[key] != expectedCount {
			t.Errorf("Expected %s count %d, got %d", key, expectedCount, stats[key])
		}
	}
}

// TestConcurrentAccess tests concurrent database operations
func TestConcurrentAccess(t *testing.T) {
	db, err := New()
	if err != nil {
		t.Fatalf("Failed to create DB: %v", err)
	}

	var wg sync.WaitGroup
	numGoroutines := 10
	opsPerGoroutine := 20

	// Concurrent writes
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			for j := 0; j < opsPerGoroutine; j++ {
				// Mix different operations
				switch j % 4 {
				case 0:
					db.StoreCommandResult("cmd", []string{}, "output", "", 0, "test")
				case 1:
					db.StoreServerTask("type", "payload")
				case 2:
					// Use unique task ID to avoid updates
					taskID := fmt.Sprintf("task-%d-%d", id, j)
					db.StoreLlamaInteraction(taskID, "prompt", "response", "test", 0.5, nil)
				case 3:
					db.StoreSystemInfo(50.0, 1024, nil, nil, 10)
				}
			}
		}(i)
	}

	// Concurrent reads
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			for j := 0; j < opsPerGoroutine; j++ {
				switch j % 4 {
				case 0:
					db.GetRecentCommands(5)
				case 1:
					db.GetPendingTasks()
				case 2:
					db.GetRecentLlamaInteractions(5)
				case 3:
					db.GetSystemInfoHistory(time.Now().Add(-1 * time.Hour))
				}
			}
		}(i)
	}

	wg.Wait()

	// Verify data integrity
	stats, err := db.Stats()
	if err != nil {
		t.Fatalf("Failed to get stats after concurrent access: %v", err)
	}

	// Each type should have numGoroutines * (opsPerGoroutine / 4) entries
	expectedPerType := numGoroutines * (opsPerGoroutine / 4)

	for key, count := range stats {
		if count < expectedPerType {
			t.Errorf("Expected at least %d entries for %s, got %d", expectedPerType, key, count)
		}
	}
}

// TestLargeDataSet tests handling of large amounts of data
func TestLargeDataSet(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping large dataset test in short mode")
	}

	db, err := New()
	if err != nil {
		t.Fatalf("Failed to create DB: %v", err)
	}

	// Store large amount of data
	numEntries := 1000

	start := time.Now()
	for i := 0; i < numEntries; i++ {
		db.StoreCommandResult("cmd", []string{}, "output", "", i%256, "test")
	}
	insertDuration := time.Since(start)

	t.Logf("Inserted %d entries in %v", numEntries, insertDuration)

	// Query performance
	start = time.Now()
	results, err := db.GetRecentCommands(100)
	if err != nil {
		t.Fatalf("Failed to query commands: %v", err)
	}
	queryDuration := time.Since(start)

	if len(results) != 100 {
		t.Errorf("Expected 100 results, got %d", len(results))
	}

	t.Logf("Queried 100 entries in %v", queryDuration)

	// Clear old data performance
	start = time.Now()
	err = db.ClearOldData(0) // Clear all
	if err != nil {
		t.Fatalf("Failed to clear data: %v", err)
	}
	clearDuration := time.Since(start)

	t.Logf("Cleared %d entries in %v", numEntries, clearDuration)

	// Verify all cleared
	stats, _ := db.Stats()
	if stats["command_results"] != 0 {
		t.Errorf("Expected 0 commands after clear, got %d", stats["command_results"])
	}
}

// TestEdgeCases tests various edge cases
func TestEdgeCases(t *testing.T) {
	db, err := New()
	if err != nil {
		t.Fatalf("Failed to create DB: %v", err)
	}

	// Test empty strings
	err = db.StoreCommandResult("", []string{}, "", "", 0, "")
	if err != nil {
		t.Errorf("Failed to store command with empty strings: %v", err)
	}

	// Test nil slices
	err = db.StoreCommandResult("cmd", nil, "output", "", 0, "test")
	if err != nil {
		t.Errorf("Failed to store command with nil args: %v", err)
	}

	// Test very long strings
	longString := string(make([]byte, 10000))
	err = db.StoreCommandResult("cmd", []string{longString}, longString, longString, 0, "test")
	if err != nil {
		t.Errorf("Failed to store command with long strings: %v", err)
	}

	// Test special characters
	specialChars := "!@#$%^&*()_+-=[]{}|;':\",./<>?\n\t\r"
	err = db.StoreCommandResult(specialChars, []string{specialChars}, specialChars, "", 0, "test")
	if err != nil {
		t.Errorf("Failed to store command with special characters: %v", err)
	}
}

// Benchmark tests
func BenchmarkStoreCommandResult(b *testing.B) {
	db, _ := New()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		db.StoreCommandResult("echo", []string{"test"}, "output", "", 0, "shell")
	}
}

func BenchmarkGetRecentCommands(b *testing.B) {
	db, _ := New()

	// Populate with data
	for i := 0; i < 1000; i++ {
		db.StoreCommandResult("echo", []string{"test"}, "output", "", 0, "shell")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		db.GetRecentCommands(10)
	}
}

func BenchmarkConcurrentWrites(b *testing.B) {
	db, _ := New()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			db.StoreCommandResult("echo", []string{"test"}, "output", "", 0, "shell")
		}
	})
}
