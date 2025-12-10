package protocol

// TaskType represents the type of task
type TaskType string

const (
	TaskTypeCommand          TaskType = "command"
	TaskTypeFileUpload       TaskType = "file_upload"
	TaskTypeFileDownload     TaskType = "file_download"
	TaskTypeFileList         TaskType = "file_list"
	TaskTypeFileDelete       TaskType = "file_delete"
	TaskTypeLlamaTask        TaskType = "llama_task"
	TaskTypeLlamaInteractive TaskType = "llama_interactive"
	TaskTypeLlamaAutonomous  TaskType = "llama_autonomous"
	TaskTypeLlamaStatus      TaskType = "llama_status"
	TaskTypeLlamaCancel      TaskType = "llama_cancel"
	TaskTypeMemDBQuery       TaskType = "memdb_query"

	// Extension related tasks
	TaskTypeExtensionUpload  TaskType = "extension_upload"  // Server->Implant: Upload extension
	TaskTypeExtensionList    TaskType = "extension_list"    // List loaded extensions
	TaskTypeExtensionExecute TaskType = "extension_execute" // Execute extension
	TaskTypeExtensionUnload  TaskType = "extension_unload"  // Unload extension
)

// String returns the string representation of TaskType
func (t TaskType) String() string {
	return string(t)
}

// TaskStatus represents the status of a task
type TaskStatus string

const (
	TaskStatusPending   TaskStatus = "pending"
	TaskStatusRunning   TaskStatus = "running"
	TaskStatusCompleted TaskStatus = "completed"
	TaskStatusFailed    TaskStatus = "failed"
	TaskStatusCancelled TaskStatus = "cancelled"
)
