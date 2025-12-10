package memdb

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/r74tech/virga/internal/implant/memdb"
)

func TestMemDBModule_Name(t *testing.T) {
	db, err := memdb.New()
	if err != nil {
		t.Fatalf("Failed to create memdb: %v", err)
	}

	module := NewMemDBModule(db)
	if name := module.Name(); name != "memdb" {
		t.Errorf("Expected name 'memdb', got '%s'", name)
	}
}

func TestNewMemDBModule(t *testing.T) {
	db, err := memdb.New()
	if err != nil {
		t.Fatalf("Failed to create memdb: %v", err)
	}

	module := NewMemDBModule(db)
	if module == nil {
		t.Fatal("NewMemDBModule returned nil")
	}
	if module.db != db {
		t.Error("Module db reference doesn't match")
	}
}

func TestMemDBModule_Execute_NilDB(t *testing.T) {
	module := &MemDBModule{db: nil}

	output, exitCode, err := module.Execute([]string{"stats"})

	if err == nil {
		t.Error("Expected error for nil db")
	}
	if exitCode != 1 {
		t.Errorf("Expected exit code 1, got %d", exitCode)
	}
	if output != "MemDB not initialized" {
		t.Errorf("Expected 'MemDB not initialized', got '%s'", output)
	}
}

func TestMemDBModule_Execute_Help(t *testing.T) {
	db, err := memdb.New()
	if err != nil {
		t.Fatalf("Failed to create memdb: %v", err)
	}
	module := NewMemDBModule(db)

	tests := []struct {
		name string
		args []string
	}{
		{
			name: "No arguments shows help",
			args: []string{},
		},
		{
			name: "Explicit help command",
			args: []string{"help"},
		},
		{
			name: "Unknown command shows help",
			args: []string{"unknown-command"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, exitCode, err := module.Execute(tt.args)

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if exitCode != 0 {
				t.Errorf("Expected exit code 0, got %d", exitCode)
			}

			// Check help contains key sections
			expectedSections := []string{
				"=== TABLES ===",
				"=== SQL-STYLE QUERIES ===",
				"=== LEGACY COMMANDS ===",
				"=== VIEWING FULL LLAMA RESPONSES ===",
				"=== MORE EXAMPLES ===",
			}

			for _, section := range expectedSections {
				if !strings.Contains(output, section) {
					t.Errorf("Help output missing section: %s", section)
				}
			}

			// Check tables are documented
			tables := []string{"commands", "llama", "tasks", "sysinfo"}
			for _, table := range tables {
				if !strings.Contains(output, table) {
					t.Errorf("Help output missing table: %s", table)
				}
			}
		})
	}
}

func TestMemDBModule_Execute_Stats(t *testing.T) {
	db, err := memdb.New()
	if err != nil {
		t.Fatalf("Failed to create memdb: %v", err)
	}
	module := NewMemDBModule(db)

	// Initial stats should be zero
	output, exitCode, err := module.Execute([]string{"stats"})
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if exitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", exitCode)
	}

	// Check output format
	if !strings.Contains(output, "MemDB Statistics:") {
		t.Error("Output should contain statistics header")
	}
	if !strings.Contains(output, "Command Results: 0") {
		t.Error("Output should show 0 command results initially")
	}

	// Add some data and check stats again
	db.StoreCommandResult("test", []string{}, "output", "", 0, "shell")
	db.StoreServerTask("test-task", "payload")
	db.StoreLlamaInteraction("task1", "prompt", "response", "llama", 0.7, []string{})

	output, exitCode, err = module.Execute([]string{"stats"})
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if !strings.Contains(output, "Command Results: 1") {
		t.Error("Stats should show 1 command result")
	}
	if !strings.Contains(output, "Server Tasks: 1") {
		t.Error("Stats should show 1 server task")
	}
	if !strings.Contains(output, "Llama Interactions: 1") {
		t.Error("Stats should show 1 llama interaction")
	}
}

func TestMemDBModule_Execute_Commands(t *testing.T) {
	db, err := memdb.New()
	if err != nil {
		t.Fatalf("Failed to create memdb: %v", err)
	}
	module := NewMemDBModule(db)

	// Test with no commands
	output, exitCode, err := module.Execute([]string{"commands"})
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if exitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", exitCode)
	}
	if !strings.Contains(output, "No recent commands found") {
		t.Error("Should indicate no commands found")
	}

	// Add some commands
	for i := 0; i < 15; i++ {
		db.StoreCommandResult(
			fmt.Sprintf("cmd%d", i),
			[]string{"arg1", "arg2"},
			fmt.Sprintf("output %d", i),
			"",
			i%2, // Alternate exit codes
			"shell",
		)
		time.Sleep(1 * time.Millisecond) // Ensure different timestamps
	}

	// Test default limit (10)
	output, exitCode, err = module.Execute([]string{"commands"})
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if !strings.Contains(output, "Recent Commands (last 10)") {
		t.Error("Should show last 10 commands")
	}

	// Test custom limit
	output, exitCode, err = module.Execute([]string{"commands", "5"})
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if !strings.Contains(output, "Recent Commands (last 5)") {
		t.Error("Should show last 5 commands")
	}

	// Check output format
	if !strings.Contains(output, "Module:") {
		t.Error("Output should contain module information")
	}
	if !strings.Contains(output, "Command:") {
		t.Error("Output should contain command information")
	}
	if !strings.Contains(output, "Exit Code:") {
		t.Error("Output should contain exit code")
	}
}

func TestMemDBModule_Execute_Tasks(t *testing.T) {
	db, err := memdb.New()
	if err != nil {
		t.Fatalf("Failed to create memdb: %v", err)
	}
	module := NewMemDBModule(db)

	// Test with no tasks
	output, exitCode, err := module.Execute([]string{"tasks"})
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if !strings.Contains(output, "No pending tasks") {
		t.Error("Should indicate no pending tasks")
	}

	// Add some tasks
	task1, _ := db.StoreServerTask("shell", "echo test")
	task2, _ := db.StoreServerTask("llama", "analyze system")

	// Mark one as completed
	db.UpdateServerTask(task1.ID, "completed", "test", "")

	// Should only show pending task
	output, exitCode, err = module.Execute([]string{"tasks"})
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if exitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", exitCode)
	}

	// Should show the pending task
	if !strings.Contains(output, "Pending Tasks:") {
		t.Error("Should show pending tasks header")
	}
	if !strings.Contains(output, task2.ID) {
		t.Error("Should show pending task ID")
	}
	if strings.Contains(output, task1.ID) {
		t.Error("Should not show completed task")
	}
}

func TestMemDBModule_Execute_Llama(t *testing.T) {
	db, err := memdb.New()
	if err != nil {
		t.Fatalf("Failed to create memdb: %v", err)
	}
	module := NewMemDBModule(db)

	// Test with no interactions
	output, exitCode, err := module.Execute([]string{"llama"})
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if !strings.Contains(output, "No recent Llama interactions found") {
		t.Error("Should indicate no interactions found")
	}

	// Add some interactions
	for i := 0; i < 7; i++ {
		db.StoreLlamaInteraction(
			fmt.Sprintf("task-%d", i),
			fmt.Sprintf("Prompt %d", i),
			strings.Repeat(fmt.Sprintf("Response line %d\n", i), 10), // Multi-line response
			"llama_autonomous",
			0.5+float32(i)*0.1,
			[]string{fmt.Sprintf("cmd%d", i), fmt.Sprintf("cmd%d-2", i)},
		)
		time.Sleep(1 * time.Millisecond)
	}

	// Test default limit (5)
	output, exitCode, err = module.Execute([]string{"llama"})
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if exitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", exitCode)
	}
	if !strings.Contains(output, "Recent Llama Interactions (last 5)") {
		t.Error("Should show last 5 interactions")
	}

	// Check truncation
	if !strings.Contains(output, "... and") && !strings.Contains(output, "more lines") {
		t.Log("Warning: Expected response truncation indicator")
	}

	// Check commands issued
	if !strings.Contains(output, "Commands Issued:") {
		t.Error("Should show commands issued")
	}

	// Test custom limit
	output, exitCode, err = module.Execute([]string{"llama", "3"})
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if !strings.Contains(output, "Recent Llama Interactions (last 3)") {
		t.Error("Should show last 3 interactions")
	}
}

func TestMemDBModule_Execute_Get(t *testing.T) {
	db, err := memdb.New()
	if err != nil {
		t.Fatalf("Failed to create memdb: %v", err)
	}
	module := NewMemDBModule(db)

	// Test with insufficient arguments
	output, exitCode, err := module.Execute([]string{"get"})
	if exitCode != 1 {
		t.Errorf("Expected exit code 1 for insufficient args, got %d", exitCode)
	}
	if !strings.Contains(output, "Usage: memdb get <type> <id>") {
		t.Error("Should show usage for get command")
	}

	output, exitCode, err = module.Execute([]string{"get", "command"})
	if exitCode != 1 {
		t.Errorf("Expected exit code 1 for insufficient args, got %d", exitCode)
	}

	// Store test data
	db.StoreCommandResult("test-cmd", []string{"arg"}, "test output", "", 0, "shell")

	// Get recent commands to find ID
	commands, _ := db.GetRecentCommands(1)
	if len(commands) == 0 {
		t.Fatal("No commands stored")
	}
	cmdID := commands[0].ID

	// Test get command by ID
	output, exitCode, err = module.Execute([]string{"get", "command", cmdID})
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if exitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", exitCode)
	}
	if !strings.Contains(output, "test-cmd") {
		t.Error("Should contain the command")
	}
	if !strings.Contains(output, "test output") {
		t.Error("Should contain full output")
	}

	// Test get non-existent command
	output, exitCode, err = module.Execute([]string{"get", "command", "non-existent-id"})
	if exitCode != 1 {
		t.Errorf("Expected exit code 1 for non-existent ID, got %d", exitCode)
	}
	if !strings.Contains(output, "not found") {
		t.Error("Should indicate command not found")
	}

	// Test get llama interaction
	db.StoreLlamaInteraction("test-task-123", "Test prompt", "Test response\nWith multiple lines", "llama", 0.7, []string{"ls", "pwd"})

	// Test get by task ID
	output, exitCode, err = module.Execute([]string{"get", "llama", "test-task-123"})
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if exitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", exitCode)
	}
	if !strings.Contains(output, "Test prompt") {
		t.Error("Should contain the prompt")
	}
	if !strings.Contains(output, "Test response\nWith multiple lines") {
		t.Error("Should contain full response without truncation")
	}

	// Test invalid record type
	output, exitCode, err = module.Execute([]string{"get", "invalid", "id"})
	if exitCode != 1 {
		t.Errorf("Expected exit code 1 for invalid type, got %d", exitCode)
	}
	if !strings.Contains(output, "Unknown record type") {
		t.Error("Should indicate unknown record type")
	}
}

func TestMemDBModule_Execute_Sysinfo(t *testing.T) {
	db, err := memdb.New()
	if err != nil {
		t.Fatalf("Failed to create memdb: %v", err)
	}
	module := NewMemDBModule(db)

	// Test with no system info
	output, exitCode, err := module.Execute([]string{"sysinfo"})
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if !strings.Contains(output, "No system info records") {
		t.Error("Should indicate no system info")
	}

	// Add system info
	db.StoreSystemInfo(50.5, 1024*1024*1024, map[string]uint64{"/": 50 * 1024 * 1024 * 1024}, map[string]uint64{"eth0": 1024 * 1024}, 100)

	// Test default (1 hour)
	output, exitCode, err = module.Execute([]string{"sysinfo"})
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if exitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", exitCode)
	}
	if !strings.Contains(output, "System Info (last 1 hour") {
		t.Error("Should show system info for last hour")
	}
	if !strings.Contains(output, "CPU Usage:") {
		t.Error("Should show CPU usage")
	}
	if !strings.Contains(output, "Memory Usage:") {
		t.Error("Should show memory usage")
	}

	// Test custom hours
	output, exitCode, err = module.Execute([]string{"sysinfo", "24"})
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if !strings.Contains(output, "System Info (last 24 hour") {
		t.Error("Should show system info for last 24 hours")
	}
}

func TestMemDBModule_Execute_Clear(t *testing.T) {
	db, err := memdb.New()
	if err != nil {
		t.Fatalf("Failed to create memdb: %v", err)
	}
	module := NewMemDBModule(db)

	// Add some data
	db.StoreCommandResult("old-cmd", []string{}, "output", "", 0, "shell")
	db.StoreServerTask("old-task", "payload")

	// Clear with default (24 hours)
	output, exitCode, err := module.Execute([]string{"clear"})
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if exitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", exitCode)
	}
	if !strings.Contains(output, "Cleared data older than 24 hours") {
		t.Error("Should confirm data cleared")
	}

	// Clear with custom hours
	output, exitCode, err = module.Execute([]string{"clear", "1"})
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if !strings.Contains(output, "Cleared data older than 1 hours") {
		t.Error("Should confirm data cleared with custom hours")
	}
}

func TestMemDBModule_Execute_SQLQueries(t *testing.T) {
	db, err := memdb.New()
	if err != nil {
		t.Fatalf("Failed to create memdb: %v", err)
	}
	module := NewMemDBModule(db)

	// Add test data
	for i := 0; i < 10; i++ {
		module := "shell"
		if i%2 == 0 {
			module = "llama"
		}
		exitCode := 0
		if i%3 == 0 {
			exitCode = 1
		}
		db.StoreCommandResult(fmt.Sprintf("cmd%d", i), []string{}, fmt.Sprintf("output%d", i), "", exitCode, module)
	}

	tests := []struct {
		name        string
		query       []string
		expectError bool
		verifyFunc  func(string) error
	}{
		{
			name:  "Simple SELECT from commands",
			query: []string{"SELECT", "*", "FROM", "commands", "LIMIT", "5"},
			verifyFunc: func(output string) error {
				if !strings.Contains(output, "Recent Commands (last 5)") {
					return fmt.Errorf("Should show 5 commands")
				}
				return nil
			},
		},
		{
			name:  "SELECT commands by module",
			query: []string{"SELECT", "*", "FROM", "commands", "WHERE", "module='llama'", "LIMIT", "3"},
			verifyFunc: func(output string) error {
				if !strings.Contains(output, "Commands from source 'llama'") {
					return fmt.Errorf("Should filter by llama module")
				}
				return nil
			},
		},
		{
			name:  "SELECT failed commands",
			query: []string{"SELECT", "*", "FROM", "commands", "WHERE", "exit_code", "!=", "0"},
			verifyFunc: func(output string) error {
				if !strings.Contains(output, "Failed commands") {
					return fmt.Errorf("Should show failed commands")
				}
				return nil
			},
		},
		{
			name:  "COUNT commands",
			query: []string{"SELECT", "COUNT(*)", "FROM", "commands"},
			verifyFunc: func(output string) error {
				if !strings.Contains(output, "Total commands:") {
					return fmt.Errorf("Should show command count")
				}
				return nil
			},
		},
		{
			name:        "Invalid table",
			query:       []string{"SELECT", "*", "FROM", "invalid_table"},
			expectError: true,
			verifyFunc: func(output string) error {
				if !strings.Contains(output, "Invalid query") {
					return fmt.Errorf("Should indicate invalid query")
				}
				return nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, exitCode, err := module.Execute(tt.query)

			if tt.expectError {
				if exitCode == 0 {
					t.Error("Expected non-zero exit code for error")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if exitCode != 0 {
					t.Errorf("Expected exit code 0, got %d", exitCode)
				}
			}

			if tt.verifyFunc != nil {
				if err := tt.verifyFunc(output); err != nil {
					t.Errorf("Verification failed: %v", err)
				}
			}
		})
	}
}

func TestMemDBModule_Execute_LlamaQueries(t *testing.T) {
	db, err := memdb.New()
	if err != nil {
		t.Fatalf("Failed to create memdb: %v", err)
	}
	module := NewMemDBModule(db)

	// Add Llama test data with various attributes
	for i := 0; i < 10; i++ {
		taskType := "llama"
		if i%2 == 0 {
			taskType = "llama_autonomous"
		}
		temp := 0.3 + float32(i)*0.1
		commands := []string{}
		if i > 5 {
			commands = []string{fmt.Sprintf("cmd%d", i)}
		}

		db.StoreLlamaInteraction(
			fmt.Sprintf("task-%d", i),
			fmt.Sprintf("Prompt %d", i),
			fmt.Sprintf("Response %d", i),
			taskType,
			temp,
			commands,
		)

		// Update status for some
		if i%3 == 0 {
			// Simulate failure by storing again with empty response
			db.StoreLlamaInteraction(
				fmt.Sprintf("task-%d", i),
				fmt.Sprintf("Prompt %d", i),
				"", // empty response for failure
				taskType,
				temp,
				[]string{},
			)
		}

		time.Sleep(1 * time.Millisecond)
	}

	tests := []struct {
		name       string
		query      []string
		verifyFunc func(string) error
	}{
		{
			name:  "SELECT from llama with ORDER BY",
			query: []string{"SELECT", "*", "FROM", "llama", "ORDER", "BY", "timestamp", "DESC", "LIMIT", "5"},
			verifyFunc: func(output string) error {
				if !strings.Contains(output, "Llama Interactions") {
					return fmt.Errorf("Should show llama interactions")
				}
				return nil
			},
		},
		{
			name:  "SELECT llama by task type",
			query: []string{"SELECT", "*", "FROM", "llama", "WHERE", "task_type='llama_autonomous'"},
			verifyFunc: func(output string) error {
				if !strings.Contains(output, "task type 'llama_autonomous'") {
					return fmt.Errorf("Should filter by task type")
				}
				return nil
			},
		},
		{
			name:  "SELECT llama with commands",
			query: []string{"SELECT", "*", "FROM", "llama", "WHERE", "commands_issued", ">", "0"},
			verifyFunc: func(output string) error {
				if !strings.Contains(output, "commands issued") {
					return fmt.Errorf("Should show interactions with commands")
				}
				return nil
			},
		},
		{
			name:  "SELECT llama by temperature",
			query: []string{"SELECT", "*", "FROM", "llama", "WHERE", "temperature", ">", "0.7"},
			verifyFunc: func(output string) error {
				if !strings.Contains(output, "temperature > 0.70") {
					return fmt.Errorf("Should filter by temperature")
				}
				return nil
			},
		},
		{
			name:  "SELECT recent llama (time-based)",
			query: []string{"SELECT", "*", "FROM", "llama", "WHERE", "timestamp", ">", "'2h'"},
			verifyFunc: func(output string) error {
				if !strings.Contains(output, "newer than 2 hours") {
					return fmt.Errorf("Should filter by time")
				}
				return nil
			},
		},
		{
			name:  "COUNT llama interactions",
			query: []string{"SELECT", "COUNT(*)", "FROM", "llama"},
			verifyFunc: func(output string) error {
				if !strings.Contains(output, "Total Llama interactions:") {
					return fmt.Errorf("Should show llama count")
				}
				return nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, exitCode, err := module.Execute(tt.query)

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if exitCode != 0 {
				t.Errorf("Expected exit code 0, got %d. Output: %s", exitCode, output)
			}

			if tt.verifyFunc != nil {
				if err := tt.verifyFunc(output); err != nil {
					t.Errorf("Verification failed: %v. Output: %s", err, output)
				}
			}
		})
	}
}

func TestMemDBModule_Execute_TasksQuery(t *testing.T) {
	db, err := memdb.New()
	if err != nil {
		t.Fatalf("Failed to create memdb: %v", err)
	}
	module := NewMemDBModule(db)

	// Add tasks
	db.StoreServerTask("shell", "echo test")
	task2, _ := db.StoreServerTask("llama", "analyze")
	db.UpdateServerTask(task2.ID, "completed", "done", "")

	output, exitCode, err := module.Execute([]string{"SELECT", "*", "FROM", "tasks", "WHERE", "status='pending'"})
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if exitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", exitCode)
	}
	if !strings.Contains(output, "Pending Tasks:") {
		t.Error("Should show pending tasks")
	}
}

func TestMemDBModule_Execute_SysinfoQuery(t *testing.T) {
	db, err := memdb.New()
	if err != nil {
		t.Fatalf("Failed to create memdb: %v", err)
	}
	module := NewMemDBModule(db)

	// Add system info
	db.StoreSystemInfo(75.5, 2*1024*1024*1024, map[string]uint64{"/": 100 * 1024 * 1024 * 1024}, map[string]uint64{"eth0": 2 * 1024 * 1024}, 150)

	output, exitCode, err := module.Execute([]string{"SELECT", "*", "FROM", "sysinfo", "WHERE", "timestamp", ">", "'1h'"})
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if exitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", exitCode)
	}
	if !strings.Contains(output, "System Info") {
		t.Error("Should show system info")
	}
}

func TestMemDBModule_OutputTruncation(t *testing.T) {
	db, err := memdb.New()
	if err != nil {
		t.Fatalf("Failed to create memdb: %v", err)
	}
	module := NewMemDBModule(db)

	// Add command with long output
	longOutput := strings.Repeat("A", 300)
	db.StoreCommandResult("long-cmd", []string{}, longOutput, "", 0, "shell")

	output, _, _ := module.Execute([]string{"commands", "1"})

	// Should truncate at 200 chars
	if !strings.Contains(output, "... (truncated)") {
		t.Error("Long output should be truncated")
	}
	if strings.Count(output, "A") > 210 { // Some buffer for formatting
		t.Error("Output not properly truncated")
	}

	// Get by ID should show full output
	commands, _ := db.GetRecentCommands(1)
	if len(commands) > 0 {
		output, _, _ = module.Execute([]string{"get", "command", commands[0].ID})
		if !strings.Contains(output, strings.Repeat("A", 300)) {
			t.Error("Get by ID should show full output")
		}
	}
}

func TestMemDBModule_LlamaResponseTruncation(t *testing.T) {
	db, err := memdb.New()
	if err != nil {
		t.Fatalf("Failed to create memdb: %v", err)
	}
	module := NewMemDBModule(db)

	// Create long multi-line response
	var longResponse strings.Builder
	for i := 0; i < 20; i++ {
		longResponse.WriteString(fmt.Sprintf("Line %d: This is a long line of text that should be truncated if too long\n", i))
	}

	db.StoreLlamaInteraction("task-long", "Test prompt", longResponse.String(), "llama", 0.7, []string{"cmd1", "cmd2", "cmd3", "cmd4", "cmd5"})

	// List view should truncate
	output, _, _ := module.Execute([]string{"llama", "1"})

	// Should show only first 5 lines
	if !strings.Contains(output, "... and") || !strings.Contains(output, "more lines") {
		t.Error("Should indicate more lines are truncated")
	}

	// Should truncate commands too
	if strings.Contains(output, "cmd5") {
		t.Error("Should only show first 3 commands")
	}
	if !strings.Contains(output, "... and 2 more commands") {
		t.Error("Should indicate more commands")
	}

	// Get by ID should show full content
	output, _, _ = module.Execute([]string{"get", "llama", "task-long"})
	if !strings.Contains(output, "Line 19:") {
		t.Error("Get by ID should show full response")
	}
	if !strings.Contains(output, "cmd5") {
		t.Error("Get by ID should show all commands")
	}
}

func TestMemDBModule_EdgeCases(t *testing.T) {
	db, err := memdb.New()
	if err != nil {
		t.Fatalf("Failed to create memdb: %v", err)
	}
	module := NewMemDBModule(db)

	tests := []struct {
		name  string
		args  []string
		setup func()
	}{
		{
			name: "Invalid limit for commands",
			args: []string{"commands", "invalid"},
			setup: func() {
				db.StoreCommandResult("test", []string{}, "output", "", 0, "shell")
			},
		},
		{
			name: "Negative limit for llama",
			args: []string{"llama", "-5"},
		},
		{
			name: "Very large limit",
			args: []string{"commands", "999999"},
		},
		{
			name: "SQL query with mixed case",
			args: []string{"SeLeCt", "*", "FrOm", "CoMmAnDs", "LiMiT", "5"},
		},
		{
			name: "Empty command stored",
			args: []string{"commands"},
			setup: func() {
				db.StoreCommandResult("", []string{}, "", "", 0, "")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}

			output, exitCode, err := module.Execute(tt.args)

			// Should handle gracefully without panic
			if output == "" && err == nil && exitCode == 0 {
				t.Error("Should produce some output or error")
			}
		})
	}
}

func TestMemDBModule_ConcurrentAccess(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrent test in short mode")
	}

	db, err := memdb.New()
	if err != nil {
		t.Fatalf("Failed to create memdb: %v", err)
	}
	module := NewMemDBModule(db)

	// Run multiple operations concurrently
	done := make(chan bool, 5)

	// Writer 1: Add commands
	go func() {
		for i := 0; i < 50; i++ {
			db.StoreCommandResult(fmt.Sprintf("cmd%d", i), []string{}, "output", "", 0, "shell")
		}
		done <- true
	}()

	// Writer 2: Add llama interactions
	go func() {
		for i := 0; i < 30; i++ {
			db.StoreLlamaInteraction(fmt.Sprintf("task%d", i), "prompt", "response", "llama", 0.7, []string{})
		}
		done <- true
	}()

	// Reader 1: Query commands
	go func() {
		for i := 0; i < 20; i++ {
			module.Execute([]string{"commands", "5"})
		}
		done <- true
	}()

	// Reader 2: Query stats
	go func() {
		for i := 0; i < 30; i++ {
			module.Execute([]string{"stats"})
		}
		done <- true
	}()

	// Reader 3: SQL queries
	go func() {
		for i := 0; i < 20; i++ {
			module.Execute([]string{"SELECT", "*", "FROM", "commands", "LIMIT", "10"})
		}
		done <- true
	}()

	// Wait for all
	for i := 0; i < 5; i++ {
		<-done
	}

	// Verify final state
	output, exitCode, err := module.Execute([]string{"stats"})
	if err != nil {
		t.Errorf("Stats failed after concurrent access: %v", err)
	}
	if exitCode != 0 {
		t.Error("Stats returned error after concurrent access")
	}
	if !strings.Contains(output, "Command Results:") {
		t.Error("Stats output corrupted after concurrent access")
	}
}

func BenchmarkMemDBModule_Execute(b *testing.B) {
	db, _ := memdb.New()
	module := NewMemDBModule(db)

	// Add test data
	for i := 0; i < 100; i++ {
		db.StoreCommandResult(fmt.Sprintf("cmd%d", i), []string{}, "output", "", 0, "shell")
	}

	b.Run("Stats", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			module.Execute([]string{"stats"})
		}
	})

	b.Run("Commands", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			module.Execute([]string{"commands", "10"})
		}
	})

	b.Run("SQLQuery", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			module.Execute([]string{"SELECT", "*", "FROM", "commands", "WHERE", "exit_code=0", "LIMIT", "5"})
		}
	})
}

func BenchmarkMemDBModule_ShowHelp(b *testing.B) {
	db, _ := memdb.New()
	module := NewMemDBModule(db)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		module.showHelp()
	}
}

// Test helper functions
func TestMemDBModule_HelperFunctions(t *testing.T) {
	db, err := memdb.New()
	if err != nil {
		t.Fatalf("Failed to create memdb: %v", err)
	}
	module := NewMemDBModule(db)

	// Test getCommandResultsForLlama
	t.Run("getCommandResultsForLlama", func(t *testing.T) {
		// Store llama interaction
		taskID := "test-llama-task"
		startTime := time.Now()

		// Store some commands after the llama task
		time.Sleep(10 * time.Millisecond)
		for i := 0; i < 3; i++ {
			db.StoreCommandResult(fmt.Sprintf("llama-cmd-%d", i), []string{}, fmt.Sprintf("output-%d", i), "", 0, "llama")
		}

		// Also store non-llama commands (should be filtered out)
		db.StoreCommandResult("other-cmd", []string{}, "other-output", "", 0, "shell")

		results := module.getCommandResultsForLlama(taskID, startTime, 3)

		if len(results) != 3 {
			t.Errorf("Expected 3 command results, got %d", len(results))
		}

		// Verify they are llama commands
		for _, result := range results {
			if result.Module != "llama" {
				t.Error("Should only return llama module commands")
			}
		}
	})
}
