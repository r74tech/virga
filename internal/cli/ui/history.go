package ui

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// historyWriter manages batched history writes
type historyWriter struct {
	mu        sync.Mutex
	file      string
	pending   []string
	ticker    *time.Ticker
	done      chan struct{}
	batchSize int
	maxSize   int
}

// newHistoryWriter creates a new history writer
func newHistoryWriter(file string, batchSize int, flushTime time.Duration, maxSize int) (*historyWriter, error) {
	// Ensure directory exists with proper permissions
	dir := filepath.Dir(file)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("create history directory: %w", err)
	}

	hw := &historyWriter{
		file:      file,
		pending:   make([]string, 0, batchSize),
		ticker:    time.NewTicker(flushTime),
		done:      make(chan struct{}),
		batchSize: batchSize,
		maxSize:   maxSize,
	}

	// Start the background flusher
	go hw.backgroundFlusher()

	return hw, nil
}

// Add adds a line to the history
func (hw *historyWriter) Add(line string) {
	if line == "" {
		return
	}

	hw.mu.Lock()
	defer hw.mu.Unlock()

	hw.pending = append(hw.pending, line)

	// Flush if batch size reached
	if len(hw.pending) >= hw.batchSize {
		hw.flushLocked()
	}
}

// Flush writes pending history to disk
func (hw *historyWriter) Flush() error {
	hw.mu.Lock()
	defer hw.mu.Unlock()

	return hw.flushLocked()
}

// flushLocked writes pending history to disk (must be called with lock held)
func (hw *historyWriter) flushLocked() error {
	if len(hw.pending) == 0 {
		return nil
	}

	// Open file with secure permissions
	file, err := os.OpenFile(hw.file, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("open history file: %w", err)
	}
	defer file.Close()

	// Write pending lines
	writer := bufio.NewWriter(file)
	for _, line := range hw.pending {
		if _, err := writer.WriteString(line + "\n"); err != nil {
			return fmt.Errorf("write history: %w", err)
		}
	}

	if err := writer.Flush(); err != nil {
		return fmt.Errorf("flush history: %w", err)
	}

	// Clear pending
	hw.pending = hw.pending[:0]

	// Trim history file if needed
	return hw.trimHistoryLocked()
}

// trimHistoryLocked trims the history file to maxSize (must be called with lock held)
func (hw *historyWriter) trimHistoryLocked() error {
	// Read all lines
	lines, err := hw.readAllLines()
	if err != nil {
		return err
	}

	// Check if trimming is needed
	if len(lines) <= hw.maxSize {
		return nil
	}

	// Keep only the most recent lines
	lines = lines[len(lines)-hw.maxSize:]

	// Write back to file
	file, err := os.OpenFile(hw.file, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("open history file for trimming: %w", err)
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	for _, line := range lines {
		if _, err := writer.WriteString(line + "\n"); err != nil {
			return fmt.Errorf("write trimmed history: %w", err)
		}
	}

	return writer.Flush()
}

// readAllLines reads all lines from the history file
func (hw *historyWriter) readAllLines() ([]string, error) {
	file, err := os.Open(hw.file)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("open history file: %w", err)
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read history: %w", err)
	}

	return lines, nil
}

// backgroundFlusher periodically flushes history to disk
func (hw *historyWriter) backgroundFlusher() {
	for {
		select {
		case <-hw.ticker.C:
			hw.Flush()
		case <-hw.done:
			hw.ticker.Stop()
			hw.Flush()
			return
		}
	}
}

// Close stops the history writer and flushes pending data
func (hw *historyWriter) Close() error {
	close(hw.done)
	return hw.Flush()
}

// LoadHistory loads history from file
func (hw *historyWriter) LoadHistory() ([]string, error) {
	hw.mu.Lock()
	defer hw.mu.Unlock()

	return hw.readAllLines()
}
