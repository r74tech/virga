package utils

import (
	"errors"
	"fmt"
)

// Common file operation errors
var (
	ErrPathEscapesRoot = errors.New("path escapes root directory")
	ErrPathTooLong     = errors.New("path exceeds maximum length")
	ErrInvalidPath     = errors.New("invalid path")
	ErrCrossDevice     = errors.New("cross-device operation")
	ErrNotAFile        = errors.New("not a file")
	ErrNotADirectory   = errors.New("not a directory")
	ErrIsDirectory     = errors.New("is a directory")
	ErrEmptyPath       = errors.New("empty path")
	ErrNullCharacter   = errors.New("path contains null character")
)

// FileError represents a file operation error with context
type FileError struct {
	Op   string // Operation name
	Path string // File path
	Err  error  // Underlying error
}

// Error implements the error interface
func (e *FileError) Error() string {
	if e.Path != "" {
		return fmt.Sprintf("%s %s: %v", e.Op, e.Path, e.Err)
	}
	return fmt.Sprintf("%s: %v", e.Op, e.Err)
}

// Unwrap returns the underlying error
func (e *FileError) Unwrap() error {
	return e.Err
}

// newFileError creates a new file error
func newFileError(op, path string, err error) error {
	return &FileError{
		Op:   op,
		Path: path,
		Err:  err,
	}
}
