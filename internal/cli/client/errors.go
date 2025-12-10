package client

import (
	"errors"
	"fmt"
	"net/http"
)

// APIError represents an error from the API
type APIError struct {
	StatusCode int
	Message    string
	Endpoint   string
}

// Error implements the error interface
func (e *APIError) Error() string {
	if e.Endpoint != "" && e.Message != "" {
		return fmt.Sprintf("API error (%d) at %s: %s", e.StatusCode, e.Endpoint, e.Message)
	}
	if e.Endpoint != "" {
		return fmt.Sprintf("API error (%d) at %s", e.StatusCode, e.Endpoint)
	}
	if e.Message != "" {
		return fmt.Sprintf("API error (%d): %s", e.StatusCode, e.Message)
	}
	return fmt.Sprintf("API error (%d)", e.StatusCode)
}

// IsNotFound returns true if the error is a 404
func (e *APIError) IsNotFound() bool {
	return e.StatusCode == http.StatusNotFound
}

// IsUnauthorized returns true if the error is a 401
func (e *APIError) IsUnauthorized() bool {
	return e.StatusCode == http.StatusUnauthorized
}

// IsServerError returns true if the error is a 5xx error
func (e *APIError) IsServerError() bool {
	return e.StatusCode >= 500 && e.StatusCode < 600
}

// ConnectionError represents a connection error
type ConnectionError struct {
	Host string
	Err  error
}

// Error implements the error interface
func (e *ConnectionError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("connection to %s failed: %v", e.Host, e.Err)
	}
	return fmt.Sprintf("connection to %s failed", e.Host)
}

// Unwrap returns the underlying error
func (e *ConnectionError) Unwrap() error {
	return e.Err
}

// RequestError represents a request error
type RequestError struct {
	Method   string
	Endpoint string
	Err      error
}

// Error implements the error interface
func (e *RequestError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("request %s %s failed: %v", e.Method, e.Endpoint, e.Err)
	}
	return fmt.Sprintf("request %s %s failed", e.Method, e.Endpoint)
}

// Unwrap returns the underlying error
func (e *RequestError) Unwrap() error {
	return e.Err
}

// IsAPIError checks if an error is an APIError
func IsAPIError(err error) bool {
	var apiErr *APIError
	return errors.As(err, &apiErr)
}

// IsConnectionError checks if an error is a ConnectionError
func IsConnectionError(err error) bool {
	var connErr *ConnectionError
	return errors.As(err, &connErr)
}

// NewAPIError creates a new APIError
func NewAPIError(statusCode int, message, endpoint string) error {
	return &APIError{
		StatusCode: statusCode,
		Message:    message,
		Endpoint:   endpoint,
	}
}

// NewConnectionError creates a new ConnectionError
func NewConnectionError(host string, err error) error {
	return &ConnectionError{
		Host: host,
		Err:  err,
	}
}

// NewRequestError creates a new RequestError
func NewRequestError(method, endpoint string, err error) error {
	return &RequestError{
		Method:   method,
		Endpoint: endpoint,
		Err:      err,
	}
}

// GetStatusCode extracts the status code from an error if it's an APIError
func GetStatusCode(err error) int {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode
	}
	return 0
}
