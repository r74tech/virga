package logger

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// JSONLWriter writes log entries in JSONL format
type JSONLWriter struct {
	mu     sync.Mutex
	file   *os.File
	level  LogLevel
	fields map[string]interface{} // Additional fields to include in every log
}

// LogEntry represents a single log entry in JSONL format
type LogEntry struct {
	Timestamp   time.Time              `json:"timestamp"`
	Level       string                 `json:"level"`
	Message     string                 `json:"message"`
	Component   string                 `json:"component,omitempty"`
	SessionID   string                 `json:"session_id,omitempty"`
	CommandName string                 `json:"command_name,omitempty"`
	Fields      map[string]interface{} `json:"fields,omitempty"`
}

// NewJSONLWriter creates a new JSONL log writer
func NewJSONLWriter(logPath string, level LogLevel) (*JSONLWriter, error) {
	// Ensure directory exists
	dir := filepath.Dir(logPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	// Open file in append mode
	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	return &JSONLWriter{
		file:   file,
		level:  level,
		fields: make(map[string]interface{}),
	}, nil
}

// SetLevel sets the minimum log level
func (w *JSONLWriter) SetLevel(level LogLevel) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.level = level
}

// SetField sets a field that will be included in all log entries
func (w *JSONLWriter) SetField(key string, value interface{}) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.fields[key] = value
}

// SetFields sets multiple fields at once
func (w *JSONLWriter) SetFields(fields map[string]interface{}) {
	w.mu.Lock()
	defer w.mu.Unlock()
	for k, v := range fields {
		w.fields[k] = v
	}
}

// Write writes a log entry if it meets the level threshold
func (w *JSONLWriter) Write(level LogLevel, component, message string, fields map[string]interface{}) error {
	if level < w.level {
		return nil
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	entry := LogEntry{
		Timestamp: time.Now(),
		Level:     level.String(),
		Message:   message,
		Component: component,
		Fields:    make(map[string]interface{}),
	}

	// Add global fields
	for k, v := range w.fields {
		entry.Fields[k] = v
	}

	// Add specific fields
	for k, v := range fields {
		entry.Fields[k] = v
	}

	// Extract special fields
	if sessionID, ok := fields["session_id"].(string); ok {
		entry.SessionID = sessionID
		delete(entry.Fields, "session_id")
	}
	if cmdName, ok := fields["command_name"].(string); ok {
		entry.CommandName = cmdName
		delete(entry.Fields, "command_name")
	}

	// Write to file
	encoder := json.NewEncoder(w.file)
	if err := encoder.Encode(entry); err != nil {
		return fmt.Errorf("failed to write log entry: %w", err)
	}

	// Flush to ensure data is written
	return w.file.Sync()
}

// Close closes the log file
func (w *JSONLWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.file != nil {
		return w.file.Close()
	}
	return nil
}

// Rotate rotates the log file (creates a new file with timestamp)
func (w *JSONLWriter) Rotate() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.file == nil {
		return fmt.Errorf("log file not open")
	}

	// Close current file
	if err := w.file.Close(); err != nil {
		return fmt.Errorf("failed to close current log file: %w", err)
	}

	// Get current file path
	currentPath := w.file.Name()

	// Create new file name with timestamp
	dir := filepath.Dir(currentPath)
	base := filepath.Base(currentPath)
	ext := filepath.Ext(base)
	name := base[:len(base)-len(ext)]

	timestamp := time.Now().Format("20060102-150405")
	newPath := filepath.Join(dir, fmt.Sprintf("%s-%s%s", name, timestamp, ext))

	// Rename current file
	if err := os.Rename(currentPath, newPath); err != nil {
		return fmt.Errorf("failed to rename log file: %w", err)
	}

	// Open new file
	file, err := os.OpenFile(currentPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("failed to open new log file: %w", err)
	}

	w.file = file
	return nil
}
