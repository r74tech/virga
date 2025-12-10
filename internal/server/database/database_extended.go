package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// ExtendedSchema contains additional schema for new features
const ExtendedSchema = `
CREATE TABLE IF NOT EXISTS llama_interactions (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	session_id TEXT,
	task_id TEXT,
	prompt TEXT,
	response TEXT,
	model TEXT DEFAULT 'llama',
	max_iterations INTEGER,
	temperature REAL,
	status TEXT,
	created_at TIMESTAMP,
	completed_at TIMESTAMP,
	FOREIGN KEY (session_id) REFERENCES sessions(id)
);

CREATE TABLE IF NOT EXISTS memdb_queries (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	session_id TEXT,
	query TEXT,
	result TEXT,
	result_count INTEGER,
	execution_time_ms INTEGER,
	created_at TIMESTAMP,
	FOREIGN KEY (session_id) REFERENCES sessions(id)
);

CREATE TABLE IF NOT EXISTS mcp_interactions (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	session_id TEXT,
	tool_name TEXT,
	action TEXT,
	parameters TEXT,
	result TEXT,
	status TEXT,
	error TEXT,
	created_at TIMESTAMP,
	completed_at TIMESTAMP,
	FOREIGN KEY (session_id) REFERENCES sessions(id)
);

CREATE TABLE IF NOT EXISTS autonomous_tasks (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	session_id TEXT,
	task_name TEXT,
	task_type TEXT,
	configuration TEXT,
	status TEXT,
	start_time TIMESTAMP,
	end_time TIMESTAMP,
	result TEXT,
	error TEXT,
	FOREIGN KEY (session_id) REFERENCES sessions(id)
);

CREATE TABLE IF NOT EXISTS beacon_configs (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	beacon_id TEXT UNIQUE,
	agent_id TEXT,
	config_json TEXT,
	sleep_time INTEGER,
	jitter INTEGER,
	kill_date TIMESTAMP,
	created_at TIMESTAMP,
	updated_at TIMESTAMP,
	FOREIGN KEY (agent_id) REFERENCES agents(id)
);

CREATE TABLE IF NOT EXISTS task_metadata (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	command_id INTEGER,
	task_id TEXT UNIQUE,
	task_type TEXT,
	priority INTEGER DEFAULT 5,
	metadata_json TEXT,
	created_at TIMESTAMP,
	FOREIGN KEY (command_id) REFERENCES commands(id)
);

CREATE INDEX IF NOT EXISTS idx_llama_session ON llama_interactions(session_id);
CREATE INDEX IF NOT EXISTS idx_llama_task ON llama_interactions(task_id);
CREATE INDEX IF NOT EXISTS idx_memdb_session ON memdb_queries(session_id);
CREATE INDEX IF NOT EXISTS idx_mcp_session ON mcp_interactions(session_id);
CREATE INDEX IF NOT EXISTS idx_mcp_tool ON mcp_interactions(tool_name);
CREATE INDEX IF NOT EXISTS idx_autonomous_session ON autonomous_tasks(session_id);
CREATE INDEX IF NOT EXISTS idx_task_metadata_task ON task_metadata(task_id);
`

// InitializeExtendedSchema initializes the extended database schema
func (d *Database) InitializeExtendedSchema() error {
	_, err := d.db.Exec(ExtendedSchema)
	return err
}

// LogLlamaInteraction logs a Llama AI interaction
func (d *Database) LogLlamaInteraction(sessionID, taskID, prompt, model string, maxIterations int, temperature float64) (int64, error) {
	result, err := d.db.Exec(
		`INSERT INTO llama_interactions (session_id, task_id, prompt, model, max_iterations, temperature, status, created_at) 
		VALUES (?, ?, ?, ?, ?, ?, 'pending', ?)`,
		sessionID, taskID, prompt, model, maxIterations, temperature, time.Now(),
	)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

// UpdateLlamaInteraction updates a Llama interaction with the response
func (d *Database) UpdateLlamaInteraction(id int64, response, status string) error {
	_, err := d.db.Exec(
		`UPDATE llama_interactions SET response = ?, status = ?, completed_at = ? WHERE id = ?`,
		response, status, time.Now(), id,
	)
	return err
}

// LogMemDBQuery logs a MemDB query
func (d *Database) LogMemDBQuery(sessionID, query, result string, resultCount int, executionTimeMs int) error {
	_, err := d.db.Exec(
		`INSERT INTO memdb_queries (session_id, query, result, result_count, execution_time_ms, created_at) 
		VALUES (?, ?, ?, ?, ?, ?)`,
		sessionID, query, result, resultCount, executionTimeMs, time.Now(),
	)
	return err
}

// LogMCPInteraction logs an MCP tool interaction
func (d *Database) LogMCPInteraction(sessionID, toolName, action, parameters string) (int64, error) {
	result, err := d.db.Exec(
		`INSERT INTO mcp_interactions (session_id, tool_name, action, parameters, status, created_at) 
		VALUES (?, ?, ?, ?, 'pending', ?)`,
		sessionID, toolName, action, parameters, time.Now(),
	)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

// UpdateMCPInteraction updates an MCP interaction with the result
func (d *Database) UpdateMCPInteraction(id int64, result, status, errorMsg string) error {
	_, err := d.db.Exec(
		`UPDATE mcp_interactions SET result = ?, status = ?, error = ?, completed_at = ? WHERE id = ?`,
		result, status, errorMsg, time.Now(), id,
	)
	return err
}

// LogAutonomousTask logs an autonomous task
func (d *Database) LogAutonomousTask(sessionID, taskName, taskType, configuration string) (int64, error) {
	result, err := d.db.Exec(
		`INSERT INTO autonomous_tasks (session_id, task_name, task_type, configuration, status, start_time) 
		VALUES (?, ?, ?, ?, 'running', ?)`,
		sessionID, taskName, taskType, configuration, time.Now(),
	)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

// UpdateAutonomousTask updates an autonomous task with the result
func (d *Database) UpdateAutonomousTask(id int64, status, result, errorMsg string) error {
	_, err := d.db.Exec(
		`UPDATE autonomous_tasks SET status = ?, result = ?, error = ?, end_time = ? WHERE id = ?`,
		status, result, errorMsg, time.Now(), id,
	)
	return err
}

// SaveBeaconConfig saves or updates beacon configuration
func (d *Database) SaveBeaconConfig(beaconID, agentID, configJSON string, sleepTime, jitter int, killDate *time.Time) error {
	// Check if beacon config exists
	var count int
	err := d.db.QueryRow("SELECT COUNT(*) FROM beacon_configs WHERE beacon_id = ?", beaconID).Scan(&count)
	if err != nil {
		return err
	}

	if count > 0 {
		// Update existing config
		_, err = d.db.Exec(
			`UPDATE beacon_configs SET agent_id = ?, config_json = ?, sleep_time = ?, jitter = ?, kill_date = ?, updated_at = ? 
			WHERE beacon_id = ?`,
			agentID, configJSON, sleepTime, jitter, killDate, time.Now(), beaconID,
		)
	} else {
		// Insert new config
		_, err = d.db.Exec(
			`INSERT INTO beacon_configs (beacon_id, agent_id, config_json, sleep_time, jitter, kill_date, created_at, updated_at) 
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			beaconID, agentID, configJSON, sleepTime, jitter, killDate, time.Now(), time.Now(),
		)
	}
	return err
}

// SaveTaskMetadata saves task metadata
func (d *Database) SaveTaskMetadata(commandID int64, taskID, taskType string, priority int, metadata map[string]interface{}) error {
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return err
	}

	_, err = d.db.Exec(
		`INSERT INTO task_metadata (command_id, task_id, task_type, priority, metadata_json, created_at) 
		VALUES (?, ?, ?, ?, ?, ?)`,
		commandID, taskID, taskType, priority, string(metadataJSON), time.Now(),
	)
	return err
}

// GetLlamaInteractions retrieves Llama interactions for a session
func (d *Database) GetLlamaInteractions(sessionID string) ([]map[string]interface{}, error) {
	rows, err := d.db.Query(
		`SELECT id, task_id, prompt, response, model, max_iterations, temperature, status, created_at, completed_at 
		FROM llama_interactions 
		WHERE session_id = ? 
		ORDER BY created_at DESC`,
		sessionID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	interactions := []map[string]interface{}{}
	for rows.Next() {
		var id int64
		var taskID, prompt, response, model, status string
		var maxIterations int
		var temperature float64
		var createdAt time.Time
		var completedAt sql.NullTime

		if err := rows.Scan(&id, &taskID, &prompt, &response, &model, &maxIterations, &temperature, &status, &createdAt, &completedAt); err != nil {
			return nil, err
		}

		interaction := map[string]interface{}{
			"id":             id,
			"task_id":        taskID,
			"prompt":         prompt,
			"response":       response,
			"model":          model,
			"max_iterations": maxIterations,
			"temperature":    temperature,
			"status":         status,
			"created_at":     createdAt,
		}
		if completedAt.Valid {
			interaction["completed_at"] = completedAt.Time
		}

		interactions = append(interactions, interaction)
	}

	return interactions, nil
}

// GetMemDBQueries retrieves MemDB queries for a session
func (d *Database) GetMemDBQueries(sessionID string) ([]map[string]interface{}, error) {
	rows, err := d.db.Query(
		`SELECT id, query, result, result_count, execution_time_ms, created_at 
		FROM memdb_queries 
		WHERE session_id = ? 
		ORDER BY created_at DESC`,
		sessionID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	queries := []map[string]interface{}{}
	for rows.Next() {
		var id int64
		var query, result string
		var resultCount, executionTimeMs int
		var createdAt time.Time

		if err := rows.Scan(&id, &query, &result, &resultCount, &executionTimeMs, &createdAt); err != nil {
			return nil, err
		}

		q := map[string]interface{}{
			"id":                id,
			"query":             query,
			"result":            result,
			"result_count":      resultCount,
			"execution_time_ms": executionTimeMs,
			"created_at":        createdAt,
		}

		queries = append(queries, q)
	}

	return queries, nil
}

// GetMCPInteractions retrieves MCP interactions for a session
func (d *Database) GetMCPInteractions(sessionID string) ([]map[string]interface{}, error) {
	rows, err := d.db.Query(
		`SELECT id, tool_name, action, parameters, result, status, error, created_at, completed_at 
		FROM mcp_interactions 
		WHERE session_id = ? 
		ORDER BY created_at DESC`,
		sessionID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	interactions := []map[string]interface{}{}
	for rows.Next() {
		var id int64
		var toolName, action, parameters, result, status string
		var errorMsg sql.NullString
		var createdAt time.Time
		var completedAt sql.NullTime

		if err := rows.Scan(&id, &toolName, &action, &parameters, &result, &status, &errorMsg, &createdAt, &completedAt); err != nil {
			return nil, err
		}

		interaction := map[string]interface{}{
			"id":         id,
			"tool_name":  toolName,
			"action":     action,
			"parameters": parameters,
			"result":     result,
			"status":     status,
			"created_at": createdAt,
		}
		if errorMsg.Valid {
			interaction["error"] = errorMsg.String
		}
		if completedAt.Valid {
			interaction["completed_at"] = completedAt.Time
		}

		interactions = append(interactions, interaction)
	}

	return interactions, nil
}

// GetExtendedStats returns extended statistics including new tables
func (d *Database) GetExtendedStats() map[string]int {
	stats := d.GetStats() // Get basic stats first

	// Count Llama interactions
	var llamaCount int
	d.db.QueryRow("SELECT COUNT(*) FROM llama_interactions").Scan(&llamaCount)
	stats["total_llama_interactions"] = llamaCount

	// Count MemDB queries
	var memdbCount int
	d.db.QueryRow("SELECT COUNT(*) FROM memdb_queries").Scan(&memdbCount)
	stats["total_memdb_queries"] = memdbCount

	// Count MCP interactions
	var mcpCount int
	d.db.QueryRow("SELECT COUNT(*) FROM mcp_interactions").Scan(&mcpCount)
	stats["total_mcp_interactions"] = mcpCount

	// Count autonomous tasks
	var autonomousCount int
	d.db.QueryRow("SELECT COUNT(*) FROM autonomous_tasks").Scan(&autonomousCount)
	stats["total_autonomous_tasks"] = autonomousCount

	// Count beacon configs
	var beaconCount int
	d.db.QueryRow("SELECT COUNT(*) FROM beacon_configs").Scan(&beaconCount)
	stats["total_beacon_configs"] = beaconCount

	return stats
}

// GetAutonomousTasks retrieves autonomous tasks for a session
func (d *Database) GetAutonomousTasks(sessionID string) ([]map[string]interface{}, error) {
	rows, err := d.db.Query(
		`SELECT id, task_name, task_type, configuration, status, start_time, end_time, result, error 
		FROM autonomous_tasks 
		WHERE session_id = ? 
		ORDER BY start_time DESC`,
		sessionID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tasks := []map[string]interface{}{}
	for rows.Next() {
		var id int64
		var taskName, taskType, configuration, status, result string
		var errorMsg sql.NullString
		var startTime time.Time
		var endTime sql.NullTime

		if err := rows.Scan(&id, &taskName, &taskType, &configuration, &status, &startTime, &endTime, &result, &errorMsg); err != nil {
			return nil, err
		}

		task := map[string]interface{}{
			"id":            id,
			"task_name":     taskName,
			"task_type":     taskType,
			"configuration": configuration,
			"status":        status,
			"start_time":    startTime,
			"result":        result,
		}
		if errorMsg.Valid {
			task["error"] = errorMsg.String
		}
		if endTime.Valid {
			task["end_time"] = endTime.Time
		}

		tasks = append(tasks, task)
	}

	return tasks, nil
}

// GetTaskMetadata retrieves metadata for a specific task
func (d *Database) GetTaskMetadata(taskID string) (map[string]interface{}, error) {
	var id, commandID int64
	var taskType, metadataJSON string
	var priority int
	var createdAt time.Time

	err := d.db.QueryRow(
		`SELECT id, command_id, task_type, priority, metadata_json, created_at 
		FROM task_metadata 
		WHERE task_id = ?`,
		taskID,
	).Scan(&id, &commandID, &taskType, &priority, &metadataJSON, &createdAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("task metadata not found for task_id: %s", taskID)
		}
		return nil, err
	}

	var metadata map[string]interface{}
	if err := json.Unmarshal([]byte(metadataJSON), &metadata); err != nil {
		metadata = map[string]interface{}{"raw": metadataJSON}
	}

	return map[string]interface{}{
		"id":         id,
		"command_id": commandID,
		"task_id":    taskID,
		"task_type":  taskType,
		"priority":   priority,
		"metadata":   metadata,
		"created_at": createdAt,
	}, nil
}
