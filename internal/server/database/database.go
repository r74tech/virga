package database

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/r74tech/virga/internal/shared/protocol"
)

// Database manages the database operations
type Database struct {
	db *sql.DB
}

// Initialize initializes the database
func Initialize(dbPath string) (*Database, error) {
	// Ensure the database directory exists
	dbDir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dbDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	// Open the database connection
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test the connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Initialize the database schema
	if err := initializeSchema(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize database schema: %w", err)
	}

	// Initialize extended schema for new features
	d := &Database{db: db}
	if err := d.InitializeExtendedSchema(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize extended database schema: %w", err)
	}

	return d, nil
}

// initializeSchema initializes the database schema
func initializeSchema(db *sql.DB) error {
	// Create the tables
	schemaSql := `
	CREATE TABLE IF NOT EXISTS agents (
		id TEXT PRIMARY KEY,
		ip_address TEXT,
		hostname TEXT,
		username TEXT,
		os TEXT,
		first_seen TIMESTAMP,
		last_seen TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS sessions (
		id TEXT PRIMARY KEY,
		agent_id TEXT,
		start_time TIMESTAMP,
		end_time TIMESTAMP,
		FOREIGN KEY (agent_id) REFERENCES agents(id)
	);

	CREATE TABLE IF NOT EXISTS commands (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		session_id TEXT,
		command TEXT,
		arguments TEXT,
		timestamp TIMESTAMP,
		status TEXT,
		FOREIGN KEY (session_id) REFERENCES sessions(id)
	);

	CREATE TABLE IF NOT EXISTS command_results (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		command_id INTEGER,
		output TEXT,
		exit_code INTEGER,
		error TEXT,
		timestamp TIMESTAMP,
		FOREIGN KEY (command_id) REFERENCES commands(id)
	);

	CREATE TABLE IF NOT EXISTS files (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		session_id TEXT,
		filename TEXT,
		file_path TEXT,
		file_size INTEGER,
		file_hash TEXT,
		upload_time TIMESTAMP,
		FOREIGN KEY (session_id) REFERENCES sessions(id)
	);
	`

	_, err := db.Exec(schemaSql)
	return err
}

// Close closes the database connection
func (d *Database) Close() error {
	return d.db.Close()
}

// SaveAgent saves the agent information to the database
func (d *Database) SaveAgent(agentID, ipAddress string) error {
	now := time.Now()

	// Check if the agent already exists
	var count int
	err := d.db.QueryRow("SELECT COUNT(*) FROM agents WHERE id = ?", agentID).Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check agent existence: %w", err)
	}

	if count > 0 {
		// Update the existing agent
		_, err := d.db.Exec(
			"UPDATE agents SET ip_address = ?, last_seen = ? WHERE id = ?",
			ipAddress, now, agentID,
		)
		return err
	} else {
		// Add a new agent
		_, err := d.db.Exec(
			"INSERT INTO agents (id, ip_address, first_seen, last_seen) VALUES (?, ?, ?, ?)",
			agentID, ipAddress, now, now,
		)
		return err
	}
}

// UpdateAgentInfo updates the agent information
func (d *Database) UpdateAgentInfo(agentID, hostname, username, os string) error {
	now := time.Now()

	_, err := d.db.Exec(
		"UPDATE agents SET hostname = ?, username = ?, os = ?, last_seen = ? WHERE id = ?",
		hostname, username, os, now, agentID,
	)
	return err
}

// CreateSession creates a session
func (d *Database) CreateSession(sessionID, agentID string) error {
	now := time.Now()

	_, err := d.db.Exec(
		"INSERT INTO sessions (id, agent_id, start_time) VALUES (?, ?, ?)",
		sessionID, agentID, now,
	)
	return err
}

// CloseSession records the session end
func (d *Database) CloseSession(sessionID string) error {
	now := time.Now()

	_, err := d.db.Exec(
		"UPDATE sessions SET end_time = ? WHERE id = ?",
		now, sessionID,
	)
	return err
}

// LogCommand records the command execution
func (d *Database) LogCommand(sessionID, command, arguments string) (int64, error) {
	now := time.Now()

	result, err := d.db.Exec(
		"INSERT INTO commands (session_id, command, arguments, timestamp, status) VALUES (?, ?, ?, ?, ?)",
		sessionID, command, arguments, now, "pending",
	)
	if err != nil {
		return 0, err
	}

	return result.LastInsertId()
}

// LogCommandResult records the command execution result
func (d *Database) LogCommandResult(commandID int64, output string, exitCode int, errorMsg string) error {
	now := time.Now()

	// Record the result
	_, err := d.db.Exec(
		"INSERT INTO command_results (command_id, output, exit_code, error, timestamp) VALUES (?, ?, ?, ?, ?)",
		commandID, output, exitCode, errorMsg, now,
	)
	if err != nil {
		return err
	}

	// Update the command status
	status := "success"
	if exitCode != 0 || errorMsg != "" {
		status = "failed"
	}

	_, err = d.db.Exec(
		"UPDATE commands SET status = ? WHERE id = ?",
		status, commandID,
	)
	return err
}

// LogFile records the file information
func (d *Database) LogFile(sessionID, filename, filePath, fileHash string, fileSize int64) error {
	now := time.Now()

	_, err := d.db.Exec(
		"INSERT INTO files (session_id, filename, file_path, file_size, file_hash, upload_time) VALUES (?, ?, ?, ?, ?, ?)",
		sessionID, filename, filePath, fileSize, fileHash, now,
	)
	return err
}

// GetAgents gets all the agent information
func (d *Database) GetAgents() ([]map[string]interface{}, error) {
	rows, err := d.db.Query(
		"SELECT id, ip_address, hostname, username, os, first_seen, last_seen FROM agents ORDER BY last_seen DESC",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	agents := []map[string]interface{}{}
	for rows.Next() {
		var id, ipAddress string
		var hostname, username, os sql.NullString
		var firstSeen, lastSeen time.Time

		if err := rows.Scan(&id, &ipAddress, &hostname, &username, &os, &firstSeen, &lastSeen); err != nil {
			return nil, err
		}

		agent := map[string]interface{}{
			"id":         id,
			"ip_address": ipAddress,
			"hostname":   hostname.String,
			"username":   username.String,
			"os":         os.String,
			"first_seen": firstSeen,
			"last_seen":  lastSeen,
		}

		agents = append(agents, agent)
	}

	return agents, nil
}

// GetAgentHistory gets the command execution history for an agent
func (d *Database) GetAgentHistory(agentID string) ([]map[string]interface{}, error) {
	query := `
	SELECT c.id, c.command, c.arguments, c.timestamp, c.status, cr.output, cr.exit_code, cr.error
	FROM commands c
	LEFT JOIN command_results cr ON c.id = cr.command_id
	LEFT JOIN sessions s ON c.session_id = s.id
	WHERE s.agent_id = ?
	ORDER BY c.timestamp DESC
	`

	rows, err := d.db.Query(query, agentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	history := []map[string]interface{}{}
	for rows.Next() {
		var id int64
		var command, arguments, status, output, errorMsg string
		var timestamp time.Time
		var exitCode int

		if err := rows.Scan(&id, &command, &arguments, &timestamp, &status, &output, &exitCode, &errorMsg); err != nil {
			return nil, err
		}

		entry := map[string]interface{}{
			"id":        id,
			"command":   command,
			"arguments": arguments,
			"timestamp": timestamp,
			"status":    status,
			"output":    output,
			"exit_code": exitCode,
			"error":     errorMsg,
		}

		history = append(history, entry)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return history, nil
}

// GetCommandHistory returns command history for a specific session
func (d *Database) GetCommandHistory(sessionID string) ([]map[string]interface{}, error) {
	query := `
	SELECT c.id, c.command, c.arguments, c.timestamp, c.status, cr.output, cr.exit_code, cr.error
	FROM commands c
	LEFT JOIN command_results cr ON c.id = cr.command_id
	WHERE c.session_id = ?
	ORDER BY c.timestamp DESC
	`

	rows, err := d.db.Query(query, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	history := []map[string]interface{}{}
	for rows.Next() {
		var id int64
		var command, arguments, status string
		var output, errorMsg sql.NullString
		var timestamp time.Time
		var exitCode sql.NullInt64

		if err := rows.Scan(&id, &command, &arguments, &timestamp, &status, &output, &exitCode, &errorMsg); err != nil {
			return nil, err
		}

		entry := map[string]interface{}{
			"id":        id,
			"command":   command,
			"arguments": arguments,
			"timestamp": timestamp,
			"status":    status,
			"output":    output.String,
			"exit_code": exitCode.Int64,
			"error":     errorMsg.String,
		}

		history = append(history, entry)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return history, nil
}

// GetStats returns database statistics
func (d *Database) GetStats() map[string]int {
	stats := make(map[string]int)

	// Count total agents
	var agentCount int
	d.db.QueryRow("SELECT COUNT(*) FROM agents").Scan(&agentCount)
	stats["total_agents"] = agentCount

	// Count total sessions
	var sessionCount int
	d.db.QueryRow("SELECT COUNT(*) FROM sessions").Scan(&sessionCount)
	stats["total_sessions"] = sessionCount

	// Count total commands
	var commandCount int
	d.db.QueryRow("SELECT COUNT(*) FROM commands").Scan(&commandCount)
	stats["total_commands"] = commandCount

	// Count total files
	var fileCount int
	d.db.QueryRow("SELECT COUNT(*) FROM files").Scan(&fileCount)
	stats["total_files"] = fileCount

	return stats
}

// GetFileOperations returns file operations for a specific session
func (d *Database) GetFileOperations(sessionID string) ([]map[string]interface{}, error) {
	query := `
	SELECT id, filename, file_path, file_size, file_hash, upload_time
	FROM files
	WHERE session_id = ?
	ORDER BY upload_time DESC
	`

	rows, err := d.db.Query(query, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	files := []map[string]interface{}{}
	for rows.Next() {
		var id int64
		var filename, filePath, fileHash string
		var fileSize int64
		var uploadTime time.Time

		if err := rows.Scan(&id, &filename, &filePath, &fileSize, &fileHash, &uploadTime); err != nil {
			return nil, err
		}

		file := map[string]interface{}{
			"id":          id,
			"filename":    filename,
			"file_path":   filePath,
			"file_size":   fileSize,
			"file_hash":   fileHash,
			"upload_time": uploadTime,
		}

		files = append(files, file)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return files, nil
}

// CreateTaskResult records the task execution result
func (d *Database) CreateTaskResult(result *protocol.TaskResult) error {
	// Get the commandID from the TaskID
	// In this example, we assume TaskID is the same as commandID
	commandID, err := strconv.ParseInt(result.TaskID, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid task id: %w", err)
	}
	return d.LogCommandResult(commandID, result.Output, result.ExitCode, result.Error)
}

// QueryRow executes a query and returns a single row
func (d *Database) QueryRow(query string, args ...interface{}) *sql.Row {
	return d.db.QueryRow(query, args...)
}
