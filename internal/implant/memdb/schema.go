package memdb

import (
	"time"

	"github.com/hashicorp/go-memdb"
)

// CommandResult stores executed command results
type CommandResult struct {
	ID        string
	Command   string
	Args      []string
	Output    string
	Error     string
	ExitCode  int
	StartTime time.Time
	EndTime   time.Time
	Module    string // shell, file, info, etc.
}

// ServerTask stores tasks received from C2 server
type ServerTask struct {
	ID          string
	Type        string // command, file_download, file_upload, etc.
	Payload     string
	Status      string // pending, executing, completed, failed
	ReceivedAt  time.Time
	CompletedAt *time.Time
	Result      string
	Error       string
}

// LlamaInteraction stores Llama prompts and responses
type LlamaInteraction struct {
	ID             string
	TaskID         string // Server task ID for tracking
	Prompt         string
	Response       string
	TaskType       string // llama_task, llama_autonomous, etc.
	StartTime      time.Time
	EndTime        time.Time
	TokensUsed     int
	Temperature    float32
	CommandsIssued []string // Commands extracted from response
	Status         string   // success, failed, timeout
}

// SystemInfo stores periodic system information snapshots
type SystemInfo struct {
	ID          string
	Timestamp   time.Time
	CPUUsage    float64
	MemoryUsage uint64
	DiskUsage   map[string]uint64
	NetworkIO   map[string]uint64
	Processes   int
}

// LlamaObservation stores each observation during ReAct pattern execution
// This is separate from CommandResult as it tracks Llama-specific context
type LlamaObservation struct {
	ID           string    // UUID
	TaskID       string    // Parent Llama task ID
	Iteration    int       // Iteration number in ReAct loop
	Command      string    // Executed command
	Output       string    // Full command output
	ExitCode     int       // Command exit code
	Timestamp    time.Time // Execution timestamp
	ExtractedKey string    // LLM-extracted key information
}

// Schema returns the memdb schema definition
func Schema() *memdb.DBSchema {
	return &memdb.DBSchema{
		Tables: map[string]*memdb.TableSchema{
			"command_results": {
				Name: "command_results",
				Indexes: map[string]*memdb.IndexSchema{
					"id": {
						Name:    "id",
						Unique:  true,
						Indexer: &memdb.StringFieldIndex{Field: "ID"},
					},
					"command": {
						Name:    "command",
						Unique:  false,
						Indexer: &memdb.StringFieldIndex{Field: "Command"},
					},
					"module": {
						Name:    "module",
						Unique:  false,
						Indexer: &memdb.StringFieldIndex{Field: "Module"},
					},
					"start_time": {
						Name:    "start_time",
						Unique:  false,
						Indexer: &TimeFieldIndex{Field: "StartTime"},
					},
				},
			},
			"server_tasks": {
				Name: "server_tasks",
				Indexes: map[string]*memdb.IndexSchema{
					"id": {
						Name:    "id",
						Unique:  true,
						Indexer: &memdb.StringFieldIndex{Field: "ID"},
					},
					"type": {
						Name:    "type",
						Unique:  false,
						Indexer: &memdb.StringFieldIndex{Field: "Type"},
					},
					"status": {
						Name:    "status",
						Unique:  false,
						Indexer: &memdb.StringFieldIndex{Field: "Status"},
					},
					"received_at": {
						Name:    "received_at",
						Unique:  false,
						Indexer: &TimeFieldIndex{Field: "ReceivedAt"},
					},
				},
			},
			"llama_interactions": {
				Name: "llama_interactions",
				Indexes: map[string]*memdb.IndexSchema{
					"id": {
						Name:    "id",
						Unique:  true,
						Indexer: &memdb.StringFieldIndex{Field: "ID"},
					},
					"task_id": {
						Name:    "task_id",
						Unique:  false,
						Indexer: &memdb.StringFieldIndex{Field: "TaskID"},
					},
					"task_type": {
						Name:    "task_type",
						Unique:  false,
						Indexer: &memdb.StringFieldIndex{Field: "TaskType"},
					},
					"status": {
						Name:    "status",
						Unique:  false,
						Indexer: &memdb.StringFieldIndex{Field: "Status"},
					},
					"start_time": {
						Name:    "start_time",
						Unique:  false,
						Indexer: &TimeFieldIndex{Field: "StartTime"},
					},
				},
			},
			"system_info": {
				Name: "system_info",
				Indexes: map[string]*memdb.IndexSchema{
					"id": {
						Name:    "id",
						Unique:  true,
						Indexer: &memdb.StringFieldIndex{Field: "ID"},
					},
					"timestamp": {
						Name:    "timestamp",
						Unique:  false,
						Indexer: &TimeFieldIndex{Field: "Timestamp"},
					},
				},
			},
			"llama_observations": {
				Name: "llama_observations",
				Indexes: map[string]*memdb.IndexSchema{
					"id": {
						Name:    "id",
						Unique:  true,
						Indexer: &memdb.StringFieldIndex{Field: "ID"},
					},
					"task_id": {
						Name:    "task_id",
						Unique:  false,
						Indexer: &memdb.StringFieldIndex{Field: "TaskID"},
					},
					"iteration": {
						Name:    "iteration",
						Unique:  false,
						Indexer: &memdb.IntFieldIndex{Field: "Iteration"},
					},
					"timestamp": {
						Name:    "timestamp",
						Unique:  false,
						Indexer: &TimeFieldIndex{Field: "Timestamp"},
					},
				},
			},
		},
	}
}
