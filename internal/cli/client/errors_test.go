package client

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"
)

func TestAPIError(t *testing.T) {
	tests := []struct {
		name    string
		err     *APIError
		wantMsg string
	}{
		{
			name: "with all fields",
			err: &APIError{
				StatusCode: 404,
				Message:    "Session not found",
				Endpoint:   "/api/sessions/123",
			},
			wantMsg: "API error (404) at /api/sessions/123: Session not found",
		},
		{
			name: "without endpoint",
			err: &APIError{
				StatusCode: 500,
				Message:    "Internal server error",
			},
			wantMsg: "API error (500): Internal server error",
		},
		{
			name: "without message",
			err: &APIError{
				StatusCode: 403,
				Endpoint:   "/api/admin",
			},
			wantMsg: "API error (403) at /api/admin",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.err.Error()
			if got != tt.wantMsg {
				t.Errorf("APIError.Error() = %v, want %v", got, tt.wantMsg)
			}
		})
	}
}

func TestAPIError_IsNotFound(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		want       bool
	}{
		{"404", http.StatusNotFound, true},
		{"200", http.StatusOK, false},
		{"500", http.StatusInternalServerError, false},
		{"403", http.StatusForbidden, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := &APIError{StatusCode: tt.statusCode}
			if got := err.IsNotFound(); got != tt.want {
				t.Errorf("APIError.IsNotFound() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAPIError_IsUnauthorized(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		want       bool
	}{
		{"401", http.StatusUnauthorized, true},
		{"200", http.StatusOK, false},
		{"403", http.StatusForbidden, false},
		{"404", http.StatusNotFound, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := &APIError{StatusCode: tt.statusCode}
			if got := err.IsUnauthorized(); got != tt.want {
				t.Errorf("APIError.IsUnauthorized() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAPIError_IsServerError(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		want       bool
	}{
		{"500", http.StatusInternalServerError, true},
		{"502", http.StatusBadGateway, true},
		{"503", http.StatusServiceUnavailable, true},
		{"504", http.StatusGatewayTimeout, true},
		{"400", http.StatusBadRequest, false},
		{"404", http.StatusNotFound, false},
		{"200", http.StatusOK, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := &APIError{StatusCode: tt.statusCode}
			if got := err.IsServerError(); got != tt.want {
				t.Errorf("APIError.IsServerError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConnectionError(t *testing.T) {
	tests := []struct {
		name    string
		err     *ConnectionError
		wantMsg string
	}{
		{
			name: "with wrapped error",
			err: &ConnectionError{
				Host: "localhost:8080",
				Err:  errors.New("connection refused"),
			},
			wantMsg: "connection to localhost:8080 failed: connection refused",
		},
		{
			name: "without wrapped error",
			err: &ConnectionError{
				Host: "example.com",
			},
			wantMsg: "connection to example.com failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.err.Error()
			if got != tt.wantMsg {
				t.Errorf("ConnectionError.Error() = %v, want %v", got, tt.wantMsg)
			}
		})
	}
}

func TestConnectionError_Unwrap(t *testing.T) {
	baseErr := errors.New("base error")
	connErr := &ConnectionError{
		Host: "test",
		Err:  baseErr,
	}

	unwrapped := connErr.Unwrap()
	if unwrapped != baseErr {
		t.Errorf("ConnectionError.Unwrap() = %v, want %v", unwrapped, baseErr)
	}
}

func TestRequestError(t *testing.T) {
	tests := []struct {
		name    string
		err     *RequestError
		wantMsg string
		wantStr []string // strings that should be in the error message
	}{
		{
			name: "GET request with all fields",
			err: &RequestError{
				Method:   "GET",
				Endpoint: "/api/sessions",
				Err:      errors.New("timeout"),
			},
			wantStr: []string{"request", "GET /api/sessions", "timeout"},
		},
		{
			name: "POST request without error",
			err: &RequestError{
				Method:   "POST",
				Endpoint: "/api/tasks",
			},
			wantStr: []string{"request", "POST /api/tasks"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.err.Error()
			for _, substr := range tt.wantStr {
				if !strings.Contains(got, substr) {
					t.Errorf("RequestError.Error() = %v, want to contain %v", got, substr)
				}
			}
		})
	}
}

func TestRequestError_Unwrap(t *testing.T) {
	baseErr := errors.New("base error")
	reqErr := &RequestError{
		Method: "GET",
		Err:    baseErr,
	}

	unwrapped := reqErr.Unwrap()
	if unwrapped != baseErr {
		t.Errorf("RequestError.Unwrap() = %v, want %v", unwrapped, baseErr)
	}
}

func TestIsAPIError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "APIError",
			err:  &APIError{StatusCode: 404},
			want: true,
		},
		{
			name: "wrapped APIError",
			err:  fmt.Errorf("wrapped: %w", &APIError{StatusCode: 404}),
			want: true,
		},
		{
			name: "other error",
			err:  errors.New("some other error"),
			want: false,
		},
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsAPIError(tt.err)
			if got != tt.want {
				t.Errorf("IsAPIError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsConnectionError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "ConnectionError",
			err:  &ConnectionError{Host: "localhost"},
			want: true,
		},
		{
			name: "wrapped ConnectionError",
			err:  fmt.Errorf("wrapped: %w", &ConnectionError{Host: "localhost"}),
			want: true,
		},
		{
			name: "other error",
			err:  errors.New("some other error"),
			want: false,
		},
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsConnectionError(tt.err)
			if got != tt.want {
				t.Errorf("IsConnectionError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewAPIError(t *testing.T) {
	err := NewAPIError(404, "Not found", "/api/test")

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("NewAPIError() returned %T, want *APIError", err)
	}

	if apiErr.StatusCode != 404 {
		t.Errorf("APIError.StatusCode = %v, want %v", apiErr.StatusCode, 404)
	}

	if apiErr.Message != "Not found" {
		t.Errorf("APIError.Message = %v, want %v", apiErr.Message, "Not found")
	}

	if apiErr.Endpoint != "/api/test" {
		t.Errorf("APIError.Endpoint = %v, want %v", apiErr.Endpoint, "/api/test")
	}
}

func TestNewConnectionError(t *testing.T) {
	baseErr := errors.New("network error")
	err := NewConnectionError("example.com:443", baseErr)

	connErr, ok := err.(*ConnectionError)
	if !ok {
		t.Fatalf("NewConnectionError() returned %T, want *ConnectionError", err)
	}

	if connErr.Host != "example.com:443" {
		t.Errorf("ConnectionError.Host = %v, want %v", connErr.Host, "example.com:443")
	}

	if connErr.Err != baseErr {
		t.Errorf("ConnectionError.Err = %v, want %v", connErr.Err, baseErr)
	}
}

func TestNewRequestError(t *testing.T) {
	baseErr := errors.New("request failed")
	err := NewRequestError("POST", "/api/sessions", baseErr)

	reqErr, ok := err.(*RequestError)
	if !ok {
		t.Fatalf("NewRequestError() returned %T, want *RequestError", err)
	}

	if reqErr.Method != "POST" {
		t.Errorf("RequestError.Method = %v, want %v", reqErr.Method, "POST")
	}

	if reqErr.Endpoint != "/api/sessions" {
		t.Errorf("RequestError.Endpoint = %v, want %v", reqErr.Endpoint, "/api/sessions")
	}

	if reqErr.Err != baseErr {
		t.Errorf("RequestError.Err = %v, want %v", reqErr.Err, baseErr)
	}
}

func TestGetStatusCode(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want int
	}{
		{
			name: "APIError",
			err:  &APIError{StatusCode: 404},
			want: 404,
		},
		{
			name: "wrapped APIError",
			err:  fmt.Errorf("wrapped: %w", &APIError{StatusCode: 500}),
			want: 500,
		},
		{
			name: "non-API error",
			err:  errors.New("not an API error"),
			want: 0,
		},
		{
			name: "nil error",
			err:  nil,
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetStatusCode(tt.err)
			if got != tt.want {
				t.Errorf("GetStatusCode() = %v, want %v", got, tt.want)
			}
		})
	}
}
