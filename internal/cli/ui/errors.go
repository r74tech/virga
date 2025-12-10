package ui

import (
	"fmt"
)

// UIError represents a UI-specific error
type UIError struct {
	Op      string // Operation that failed
	Message string // Error message
	Err     error  // Underlying error
}

// Error implements the error interface
func (e *UIError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("ui: %s: %s: %v", e.Op, e.Message, e.Err)
	}
	return fmt.Sprintf("ui: %s: %s", e.Op, e.Message)
}

// Unwrap returns the underlying error
func (e *UIError) Unwrap() error {
	return e.Err
}

// newUIError creates a new UI error
func newUIError(op, message string, err error) error {
	return &UIError{
		Op:      op,
		Message: message,
		Err:     err,
	}
}

// Common UI errors
var (
	ErrNoTerminal    = &UIError{Op: "terminal", Message: "no terminal available"}
	ErrHistoryFile   = &UIError{Op: "history", Message: "failed to access history file"}
	ErrInvalidConfig = &UIError{Op: "config", Message: "invalid configuration"}
)
