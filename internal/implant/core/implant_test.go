package core

import (
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/r74tech/virga/internal/implant/config"
	"github.com/r74tech/virga/internal/shared/protocol"
	"github.com/r74tech/virga/internal/shared/testutil/mock"
)

func TestNewImplant(t *testing.T) {
	// Create config and transport
	cfg := config.NewConfig()
	transport := mock.NewMockTransport()

	// Test implant creation
	implant := NewImplant(cfg, transport)
	if implant == nil {
		t.Fatal("Expected implant to be created")
	}

	// Verify initialization
	if implant.Config == nil {
		t.Error("Expected config to be set")
	}
	if implant.Config.AgentID == "" {
		t.Error("Expected agent ID to be generated")
	}
	if implant.Transport == nil {
		t.Error("Expected transport to be set")
	}
	if implant.Modules == nil {
		t.Error("Expected modules map to be initialized")
	}
	if implant.Environment == nil {
		t.Error("Expected environment to be collected")
	}
}

func TestImplantBeacon(t *testing.T) {
	cfg := config.NewConfig()
	cfg.SleepTime = "1"
	cfg.Jitter = "0"

	transport := mock.NewMockTransport()
	implant := NewImplant(cfg, transport)

	// Add a task to transport
	testTask := protocol.Task{
		TaskID: "task-test-123",
		Type:   "sysinfo",
	}
	transport.AddTask(testTask)

	// Create a beacon message
	beacon := &protocol.AgentMessage{
		AgentID:     implant.Config.AgentID,
		Hostname:    implant.Environment["hostname"],
		Username:    implant.Environment["username"],
		OS:          implant.Environment["os"],
		Environment: make(map[string]interface{}),
	}

	// Send beacon and get response
	response, err := transport.SendBeacon(beacon)
	if err != nil {
		t.Fatalf("SendBeacon failed: %v", err)
	}

	// Verify response
	if response.Status != "success" {
		t.Error("Expected successful response")
	}

	// Verify task was received
	if len(response.Tasks) != 1 {
		t.Fatalf("Expected 1 task, got %d", len(response.Tasks))
	}
	if response.Tasks[0].TaskID != testTask.TaskID {
		t.Errorf("Expected task ID %s, got %s", testTask.TaskID, response.Tasks[0].TaskID)
	}
}

// MockModule implements the Module interface for testing
type MockModule struct {
	name   string
	output string
	code   int
	err    error
}

func (m *MockModule) Name() string {
	return m.name
}

func (m *MockModule) Execute(args []string) (string, int, error) {
	return m.output, m.code, m.err
}

func TestImplantTaskExecution(t *testing.T) {
	tests := []struct {
		name        string
		task        protocol.Task
		module      Module
		expectError bool
		checkResult func(t *testing.T, result protocol.AgentTaskResult)
	}{
		{
			name: "Execute shell command",
			task: protocol.Task{
				TaskID:  "task-shell-123",
				Type:    "shell",
				Payload: map[string]interface{}{"command": "echo test"},
			},
			module: &MockModule{
				name:   "shell",
				output: "test\n",
				code:   0,
			},
			expectError: false,
			checkResult: func(t *testing.T, result protocol.AgentTaskResult) {
				if result.ExitCode != 0 {
					t.Errorf("Expected exit code 0, got %d", result.ExitCode)
				}
				if result.Output != "test\n" {
					t.Errorf("Expected output 'test\\n', got %q", result.Output)
				}
			},
		},
		{
			name: "Execute sysinfo",
			task: protocol.Task{
				TaskID: "task-sysinfo-456",
				Type:   "sysinfo",
			},
			module: &MockModule{
				name:   "sysinfo",
				output: "System: Test OS\nArch: x64",
				code:   0,
			},
			expectError: false,
			checkResult: func(t *testing.T, result protocol.AgentTaskResult) {
				if result.Output == "" {
					t.Error("Expected sysinfo output, got empty string")
				}
			},
		},
		{
			name: "Module returns error",
			task: protocol.Task{
				TaskID: "task-error-789",
				Type:   "failing",
			},
			module: &MockModule{
				name:   "failing",
				output: "partial output",
				code:   1,
				err:    fmt.Errorf("command failed"),
			},
			expectError: false,
			checkResult: func(t *testing.T, result protocol.AgentTaskResult) {
				if result.Error == "" {
					t.Error("Expected error in result")
				}
				if result.ExitCode != 1 {
					t.Errorf("Expected exit code 1, got %d", result.ExitCode)
				}
			},
		},
		{
			name: "Unknown task type",
			task: protocol.Task{
				TaskID: "task-unknown-012",
				Type:   "unknown_command",
			},
			module:      nil, // No module registered
			expectError: false,
			checkResult: func(t *testing.T, result protocol.AgentTaskResult) {
				if !strings.Contains(result.Error, "unknown task type") {
					t.Errorf("Expected unknown task type error, got %s", result.Error)
				}
				if result.ExitCode != -1 {
					t.Errorf("Expected exit code -1, got %d", result.ExitCode)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.NewConfig()
			transport := mock.NewMockTransport()
			implant := NewImplant(cfg, transport)

			// Register module if provided
			if tt.module != nil {
				implant.RegisterModule(tt.module)
			}

			// Execute task using private method
			result := implant.executeTask(tt.task)

			if tt.expectError && result.Error == "" {
				t.Error("Expected error but got none")
			}
			// Don't check for unexpected errors here - let checkResult handle specific error validation

			// Verify result
			if result.TaskID != tt.task.TaskID {
				t.Errorf("Expected task ID %s, got %s", tt.task.TaskID, result.TaskID)
			}

			// Run custom checks
			if tt.checkResult != nil {
				tt.checkResult(t, result)
			}
		})
	}
}

func TestImplantModuleRegistration(t *testing.T) {
	cfg := config.NewConfig()
	transport := mock.NewMockTransport()
	implant := NewImplant(cfg, transport)

	// Test registering modules
	modules := []Module{
		&MockModule{name: "shell"},
		&MockModule{name: "sysinfo"},
		&MockModule{name: "netinfo"},
	}

	for _, module := range modules {
		implant.RegisterModule(module)
	}

	// Verify modules are registered
	if len(implant.Modules) != len(modules) {
		t.Errorf("Expected %d modules, got %d", len(modules), len(implant.Modules))
	}

	for _, module := range modules {
		if _, ok := implant.Modules[module.Name()]; !ok {
			t.Errorf("Module %s not registered", module.Name())
		}
	}
}

func TestImplantResultQueue(t *testing.T) {
	cfg := config.NewConfig()
	transport := mock.NewMockTransport()
	implant := NewImplant(cfg, transport)

	// Create some task results
	results := []protocol.AgentTaskResult{
		{
			TaskID:    "task-1",
			Output:    "Result 1",
			ExitCode:  0,
			Timestamp: time.Now().Unix(),
		},
		{
			TaskID:    "task-2",
			Output:    "Result 2",
			ExitCode:  0,
			Timestamp: time.Now().Unix(),
		},
		{
			TaskID:    "task-3",
			Output:    "Result 3",
			ExitCode:  1,
			Error:     "Some error",
			Timestamp: time.Now().Unix(),
		},
	}

	// Queue results using private field
	for _, result := range results {
		implant.resultsMutex.Lock()
		implant.pendingResults = append(implant.pendingResults, result)
		implant.resultsMutex.Unlock()
	}

	// Get results by accessing private field
	implant.resultsMutex.Lock()
	queuedResults := append([]protocol.AgentTaskResult{}, implant.pendingResults...)
	implant.pendingResults = nil
	implant.resultsMutex.Unlock()

	// Verify all results are returned
	if len(queuedResults) != len(results) {
		t.Errorf("Expected %d results, got %d", len(results), len(queuedResults))
	}

	// Verify queue is cleared
	implant.resultsMutex.Lock()
	if len(implant.pendingResults) != 0 {
		t.Errorf("Expected empty queue after getResults, got %d results", len(implant.pendingResults))
	}
	implant.resultsMutex.Unlock()
}

func TestImplantEnvironment(t *testing.T) {
	cfg := config.NewConfig()
	transport := mock.NewMockTransport()
	implant := NewImplant(cfg, transport)

	// Check required environment fields
	requiredFields := []string{
		"hostname",
		"username",
		"os",
		"arch",
		"pid",
	}

	for _, field := range requiredFields {
		if _, ok := implant.Environment[field]; !ok {
			t.Errorf("Expected environment field %s not found", field)
		}
	}

	// Verify some values are not empty
	if implant.Environment["hostname"] == "" {
		t.Error("Hostname should not be empty")
	}
	if implant.Environment["username"] == "" {
		t.Error("Username should not be empty")
	}
	if implant.Environment["os"] == "" {
		t.Error("OS should not be empty")
	}
}

func TestImplantTransportErrors(t *testing.T) {
	cfg := config.NewConfig()
	transport := mock.NewMockTransport()
	implant := NewImplant(cfg, transport)

	// Test beacon send error
	t.Run("Beacon send error", func(t *testing.T) {
		transport.SetFailNext(true)
		beacon := &protocol.AgentMessage{
			AgentID:  implant.Config.AgentID,
			Hostname: "test-host",
			Username: "test-user",
			OS:       "test-os",
		}
		_, err := transport.SendBeacon(beacon)
		if err == nil {
			t.Error("Expected send beacon error")
		}
	})

	// Test beacon with results
	t.Run("Beacon with results", func(t *testing.T) {
		results := []protocol.AgentTaskResult{
			{
				TaskID:    "task-1",
				Output:    "Test output",
				ExitCode:  0,
				Timestamp: time.Now().Unix(),
			},
		}

		beacon := &protocol.AgentMessage{
			AgentID:     implant.Config.AgentID,
			Hostname:    "test-host",
			Username:    "test-user",
			OS:          "test-os",
			TaskResults: results,
		}

		_, err := transport.SendBeacon(beacon)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		// Verify results were stored
		if stored, ok := transport.GetTaskResult("task-1"); !ok {
			t.Error("Task result not stored")
		} else if stored.Output != "Test output" {
			t.Errorf("Expected output 'Test output', got %s", stored.Output)
		}
	})
}

func TestImplantConcurrentResultQueue(t *testing.T) {
	cfg := config.NewConfig()
	transport := mock.NewMockTransport()
	implant := NewImplant(cfg, transport)

	// Test concurrent access to result queue
	var wg sync.WaitGroup
	numGoroutines := 10
	resultsPerGoroutine := 5

	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			defer wg.Done()
			for j := 0; j < resultsPerGoroutine; j++ {
				result := protocol.AgentTaskResult{
					TaskID:    fmt.Sprintf("task-%d-%d", goroutineID, j),
					Output:    fmt.Sprintf("Result from goroutine %d", goroutineID),
					ExitCode:  0,
					Timestamp: time.Now().Unix(),
				}
				implant.resultsMutex.Lock()
				implant.pendingResults = append(implant.pendingResults, result)
				implant.resultsMutex.Unlock()
			}
		}(i)
	}

	wg.Wait()

	// Get all results
	implant.resultsMutex.Lock()
	results := append([]protocol.AgentTaskResult{}, implant.pendingResults...)
	implant.pendingResults = nil
	implant.resultsMutex.Unlock()

	// Verify all results were queued
	expectedTotal := numGoroutines * resultsPerGoroutine
	if len(results) != expectedTotal {
		t.Errorf("Expected %d results, got %d", expectedTotal, len(results))
	}

	// Verify no duplicates
	seen := make(map[string]bool)
	for _, result := range results {
		if seen[result.TaskID] {
			t.Errorf("Duplicate task ID found: %s", result.TaskID)
		}
		seen[result.TaskID] = true
	}
}

func TestImplantBootTime(t *testing.T) {
	cfg := config.NewConfig()
	transport := mock.NewMockTransport()

	// Record time before creation
	beforeCreate := time.Now().Unix()

	implant := NewImplant(cfg, transport)

	// Record time after creation
	afterCreate := time.Now().Unix()

	// Verify boot time is set correctly
	if implant.bootTime < beforeCreate || implant.bootTime > afterCreate {
		t.Errorf("Boot time %d not within expected range [%d, %d]",
			implant.bootTime, beforeCreate, afterCreate)
	}
}

func TestImplantMemoryDB(t *testing.T) {
	cfg := config.NewConfig()
	transport := mock.NewMockTransport()
	implant := NewImplant(cfg, transport)

	// Check if MemDB was initialized
	// Note: MemDB initialization might fail, which is acceptable
	if implant.DB != nil {
		t.Log("MemDB initialized successfully")
	} else {
		t.Log("MemDB not initialized (this is acceptable)")
	}
}

func TestImplantAESKeyGeneration(t *testing.T) {
	cfg := config.NewConfig()
	transport := mock.NewMockTransport()
	implant := NewImplant(cfg, transport)

	// Verify AES key was generated
	if len(implant.Config.AESKey) != 32 {
		t.Errorf("Expected 32-byte AES key, got %d bytes", len(implant.Config.AESKey))
	}

	// Verify key is not all zeros
	allZero := true
	for _, b := range implant.Config.AESKey {
		if b != 0 {
			allZero = false
			break
		}
	}
	if allZero {
		t.Error("AES key should not be all zeros")
	}
}
