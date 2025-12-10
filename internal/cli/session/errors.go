package session

import (
	"errors"
	"fmt"
	"strings"
)

// Common errors
var (
	ErrSessionNotFound  = errors.New("session not found")
	ErrNoAPIClient      = errors.New("API client not set")
	ErrNoCurrentSession = errors.New("no current session")
	ErrSyncFailed       = errors.New("failed to sync with server")
)

// SessionError represents a session-related error
type SessionError struct {
	Op        string // Operation that failed
	SessionID string // Session ID (if applicable)
	Err       error  // Underlying error
}

// Error implements the error interface
func (e *SessionError) Error() string {
	if e.SessionID != "" {
		return fmt.Sprintf("%s: session %s: %v", e.Op, e.SessionID, e.Err)
	}
	return fmt.Sprintf("%s: %v", e.Op, e.Err)
}

// Unwrap returns the underlying error
func (e *SessionError) Unwrap() error {
	return e.Err
}

// Is checks if the error matches the target error
func (e *SessionError) Is(target error) bool {
	return errors.Is(e.Err, target)
}

// ValidationError represents a validation error
type ValidationError struct {
	Field   string
	Message string
}

// Error implements the error interface
func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation error: %s %s", e.Field, e.Message)
}

// NotFoundError represents a not found error
type NotFoundError struct {
	Resource string
	ID       string
}

// Error implements the error interface
func (e *NotFoundError) Error() string {
	return fmt.Sprintf("%s %s not found", e.Resource, e.ID)
}

// TimeoutError represents a timeout error
type TimeoutError struct {
	Operation string
	Duration  int // in seconds
}

// Error implements the error interface
func (e *TimeoutError) Error() string {
	if e.Duration > 0 {
		return fmt.Sprintf("operation %s timed out after %ds", e.Operation, e.Duration)
	}
	return fmt.Sprintf("operation %s timed out", e.Operation)
}

// IsNotFound checks if an error is a NotFoundError
func IsNotFound(err error) bool {
	var notFound *NotFoundError
	return errors.As(err, &notFound)
}

// IsTimeout checks if an error is a timeout
func IsTimeout(err error) bool {
	var timeout *TimeoutError
	if errors.As(err, &timeout) {
		return true
	}

	// Also check for standard timeout errors
	if err != nil {
		errStr := err.Error()
		return strings.Contains(errStr, "timeout") ||
			strings.Contains(errStr, "deadline exceeded")
	}

	return false
}

// IsValidationError checks if an error is a ValidationError
func IsValidationError(err error) bool {
	var validationErr *ValidationError
	return errors.As(err, &validationErr)
}

// WrapError wraps an error with session context
func WrapError(op, sessionID string, err error) error {
	return &SessionError{
		Op:        op,
		SessionID: sessionID,
		Err:       err,
	}
}

// NewValidationError creates a new validation error
func NewValidationError(field, message string) error {
	return &ValidationError{
		Field:   field,
		Message: message,
	}
}

// NewNotFoundError creates a new not found error
func NewNotFoundError(resource, id string) error {
	return &NotFoundError{
		Resource: resource,
		ID:       id,
	}
}

// NewTimeoutError creates a new timeout error
func NewTimeoutError(operation string, duration int) error {
	return &TimeoutError{
		Operation: operation,
		Duration:  duration,
	}
}
