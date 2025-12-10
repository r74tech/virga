package ui

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// ExtendedHistoryEntry represents a command history entry with additional metadata
type ExtendedHistoryEntry struct {
	Timestamp   time.Time              `json:"timestamp"`
	Command     string                 `json:"command"`
	SessionID   string                 `json:"session_id,omitempty"`
	SessionInfo map[string]interface{} `json:"session_info,omitempty"`
	Result      string                 `json:"result,omitempty"`
	Success     bool                   `json:"success"`
}

// ExtendedHistoryWriter manages extended history with metadata
type ExtendedHistoryWriter struct {
	mu           sync.Mutex
	file         string
	currentEntry *ExtendedHistoryEntry
}

// NewExtendedHistoryWriter creates a new extended history writer
func NewExtendedHistoryWriter(file string) (*ExtendedHistoryWriter, error) {
	// Ensure directory exists with proper permissions
	dir := filepath.Dir(file)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("create history directory: %w", err)
	}

	return &ExtendedHistoryWriter{
		file: file,
	}, nil
}

// StartCommand starts tracking a new command
func (ehw *ExtendedHistoryWriter) StartCommand(command string, sessionID string, sessionInfo map[string]interface{}) {
	ehw.mu.Lock()
	defer ehw.mu.Unlock()

	ehw.currentEntry = &ExtendedHistoryEntry{
		Timestamp:   time.Now(),
		Command:     command,
		SessionID:   sessionID,
		SessionInfo: sessionInfo,
		Success:     true, // Default to success
	}
}

// SetResult sets the result of the current command
func (ehw *ExtendedHistoryWriter) SetResult(result string, success bool) {
	ehw.mu.Lock()
	defer ehw.mu.Unlock()

	if ehw.currentEntry != nil {
		ehw.currentEntry.Result = result
		ehw.currentEntry.Success = success
	}
}

// EndCommand completes the current command and writes it to history
func (ehw *ExtendedHistoryWriter) EndCommand() error {
	ehw.mu.Lock()
	defer ehw.mu.Unlock()

	if ehw.currentEntry == nil {
		return nil
	}

	// Marshal entry to JSON
	data, err := json.Marshal(ehw.currentEntry)
	if err != nil {
		return fmt.Errorf("marshal history entry: %w", err)
	}

	// Open file with secure permissions
	file, err := os.OpenFile(ehw.file, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("open history file: %w", err)
	}
	defer file.Close()

	// Write entry
	if _, err := file.Write(data); err != nil {
		return fmt.Errorf("write history: %w", err)
	}
	if _, err := file.WriteString("\n"); err != nil {
		return fmt.Errorf("write history newline: %w", err)
	}

	// Clear current entry
	ehw.currentEntry = nil

	return nil
}

// LoadHistory loads all history entries
func (ehw *ExtendedHistoryWriter) LoadHistory() ([]ExtendedHistoryEntry, error) {
	ehw.mu.Lock()
	defer ehw.mu.Unlock()

	data, err := os.ReadFile(ehw.file)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read history file: %w", err)
	}

	var entries []ExtendedHistoryEntry
	lines := string(data)
	for _, line := range splitLines(lines) {
		line = trimSpace(line)
		if line == "" {
			continue
		}

		var entry ExtendedHistoryEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			// Skip invalid entries
			continue
		}
		entries = append(entries, entry)
	}

	return entries, nil
}

// Helper functions to avoid importing strings package
func splitLines(s string) []string {
	var lines []string
	var current []byte
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, string(current))
			current = current[:0]
		} else {
			current = append(current, s[i])
		}
	}
	if len(current) > 0 {
		lines = append(lines, string(current))
	}
	return lines
}

func trimSpace(s string) string {
	start := 0
	end := len(s)

	// Trim leading spaces
	for start < end && (s[start] == ' ' || s[start] == '\t' || s[start] == '\n' || s[start] == '\r') {
		start++
	}

	// Trim trailing spaces
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\n' || s[end-1] == '\r') {
		end--
	}

	return s[start:end]
}
