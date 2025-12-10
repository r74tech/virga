package mock

import (
	"fmt"
	"strings"
	"sync"
)

// MockCommandExecutor implements a mock command executor for testing
// This is commonly used for testing Llama integration and other command execution scenarios
type MockCommandExecutor struct {
	mu               sync.Mutex
	executedCommands []ExecutedCommand
	defaultOutput    string
	defaultExitCode  int
	failNext         bool

	// Function field for customizing behavior
	ExecuteFunc func(command string) (string, int, error)
}

// ExecutedCommand represents a command that was executed
type ExecutedCommand struct {
	Command  string
	Output   string
	ExitCode int
	Error    error
}

// NewMockCommandExecutor creates a new mock command executor with default behavior
func NewMockCommandExecutor() *MockCommandExecutor {
	return &MockCommandExecutor{
		defaultOutput:   "mock output",
		defaultExitCode: 0,
	}
}

// Execute executes a command and returns output
func (m *MockCommandExecutor) Execute(command string) (string, int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Use custom function if provided
	if m.ExecuteFunc != nil {
		output, exitCode, err := m.ExecuteFunc(command)
		m.executedCommands = append(m.executedCommands, ExecutedCommand{
			Command:  command,
			Output:   output,
			ExitCode: exitCode,
			Error:    err,
		})
		return output, exitCode, err
	}

	// Default behavior
	if m.failNext {
		m.failNext = false
		err := fmt.Errorf("mock executor: command failed")
		m.executedCommands = append(m.executedCommands, ExecutedCommand{
			Command:  command,
			Output:   "",
			ExitCode: 1,
			Error:    err,
		})
		return "", 1, err
	}

	// Simulate some common commands
	output := m.defaultOutput
	exitCode := m.defaultExitCode

	// Simple command simulation
	if strings.HasPrefix(command, "echo ") {
		output = strings.TrimPrefix(command, "echo ")
		output = strings.TrimSpace(output)
	} else if command == "pwd" {
		output = "/home/user"
	} else if command == "whoami" {
		output = "testuser"
	} else if strings.HasPrefix(command, "ls") {
		output = "file1.txt\nfile2.log\ndir1/"
	}

	m.executedCommands = append(m.executedCommands, ExecutedCommand{
		Command:  command,
		Output:   output,
		ExitCode: exitCode,
		Error:    nil,
	})

	return output, exitCode, nil
}

// Test helper methods

// SetDefaultOutput sets the default output for commands
func (m *MockCommandExecutor) SetDefaultOutput(output string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.defaultOutput = output
}

// SetDefaultExitCode sets the default exit code for commands
func (m *MockCommandExecutor) SetDefaultExitCode(code int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.defaultExitCode = code
}

// SetFailNext causes the next command execution to fail
func (m *MockCommandExecutor) SetFailNext(fail bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.failNext = fail
}

// GetExecutedCommands returns all executed commands
func (m *MockCommandExecutor) GetExecutedCommands() []ExecutedCommand {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Return a copy to prevent concurrent modification
	commands := make([]ExecutedCommand, len(m.executedCommands))
	copy(commands, m.executedCommands)
	return commands
}

// GetLastExecutedCommand returns the last executed command, or nil if none
func (m *MockCommandExecutor) GetLastExecutedCommand() *ExecutedCommand {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.executedCommands) == 0 {
		return nil
	}

	// Return a copy of the last command
	last := m.executedCommands[len(m.executedCommands)-1]
	return &last
}

// ClearExecutedCommands clears the history of executed commands
func (m *MockCommandExecutor) ClearExecutedCommands() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.executedCommands = make([]ExecutedCommand, 0)
}

// GetExecutedCommandStrings returns just the command strings that were executed
func (m *MockCommandExecutor) GetExecutedCommandStrings() []string {
	m.mu.Lock()
	defer m.mu.Unlock()

	commands := make([]string, len(m.executedCommands))
	for i, cmd := range m.executedCommands {
		commands[i] = cmd.Command
	}
	return commands
}

// WasCommandExecuted checks if a specific command was executed
func (m *MockCommandExecutor) WasCommandExecuted(command string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, cmd := range m.executedCommands {
		if cmd.Command == command {
			return true
		}
	}
	return false
}

// GetCommandExecutionCount returns how many commands were executed
func (m *MockCommandExecutor) GetCommandExecutionCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.executedCommands)
}
