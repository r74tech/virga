package protocol

import (
	"testing"
)

func TestTaskTypeString(t *testing.T) {
	tests := []struct {
		taskType TaskType
		expected string
	}{
		{TaskTypeCommand, "command"},
		{TaskTypeFileUpload, "file_upload"},
		{TaskTypeFileDownload, "file_download"},
		{TaskTypeFileList, "file_list"},
		{TaskTypeFileDelete, "file_delete"},
		{TaskTypeLlamaTask, "llama_task"},
		{TaskTypeLlamaInteractive, "llama_interactive"},
		{TaskTypeLlamaAutonomous, "llama_autonomous"},
		{TaskTypeLlamaStatus, "llama_status"},
		{TaskTypeLlamaCancel, "llama_cancel"},
		{TaskTypeMemDBQuery, "memdb_query"},
	}

	for _, tt := range tests {
		t.Run(string(tt.taskType), func(t *testing.T) {
			result := tt.taskType.String()
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestTaskTypeConstants(t *testing.T) {
	// Ensure all task types have unique values
	taskTypes := map[TaskType]bool{
		TaskTypeCommand:          true,
		TaskTypeFileUpload:       true,
		TaskTypeFileDownload:     true,
		TaskTypeFileList:         true,
		TaskTypeFileDelete:       true,
		TaskTypeLlamaTask:        true,
		TaskTypeLlamaInteractive: true,
		TaskTypeLlamaAutonomous:  true,
		TaskTypeLlamaStatus:      true,
		TaskTypeLlamaCancel:      true,
		TaskTypeMemDBQuery:       true,
	}

	if len(taskTypes) != 11 {
		t.Errorf("Expected 11 unique task types, got %d", len(taskTypes))
	}
}

func TestTaskStatusConstants(t *testing.T) {
	// Ensure all task statuses have unique values
	statuses := map[TaskStatus]bool{
		TaskStatusPending:   true,
		TaskStatusRunning:   true,
		TaskStatusCompleted: true,
		TaskStatusFailed:    true,
		TaskStatusCancelled: true,
	}

	if len(statuses) != 5 {
		t.Errorf("Expected 5 unique task statuses, got %d", len(statuses))
	}
}

func TestTaskStatusValues(t *testing.T) {
	tests := []struct {
		status   TaskStatus
		expected string
	}{
		{TaskStatusPending, "pending"},
		{TaskStatusRunning, "running"},
		{TaskStatusCompleted, "completed"},
		{TaskStatusFailed, "failed"},
		{TaskStatusCancelled, "cancelled"},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			if string(tt.status) != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, string(tt.status))
			}
		})
	}
}

func TestCustomTaskType(t *testing.T) {
	// Test that custom task types can be created
	customType := TaskType("custom_task")
	if customType.String() != "custom_task" {
		t.Errorf("Expected custom_task, got %s", customType.String())
	}
}

func TestTaskTypeComparison(t *testing.T) {
	// Test task type comparisons
	if TaskTypeCommand == TaskTypeFileUpload {
		t.Error("Different task types should not be equal")
	}

	if TaskTypeCommand != TaskTypeCommand {
		t.Error("Same task types should be equal")
	}

	// Test with variable
	taskType := TaskTypeCommand
	if taskType != TaskTypeCommand {
		t.Error("Task type variable should equal constant")
	}
}

func TestTaskStatusComparison(t *testing.T) {
	// Test status comparisons
	if TaskStatusPending == TaskStatusRunning {
		t.Error("Different statuses should not be equal")
	}

	if TaskStatusCompleted != TaskStatusCompleted {
		t.Error("Same statuses should be equal")
	}

	// Test with variable
	status := TaskStatusPending
	if status != TaskStatusPending {
		t.Error("Status variable should equal constant")
	}
}
