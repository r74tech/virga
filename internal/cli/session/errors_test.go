package session

import (
	"errors"
	"strings"
	"testing"
)

func TestSessionError(t *testing.T) {
	tests := []struct {
		name    string
		err     *SessionError
		wantMsg string
	}{
		{
			name: "with all fields",
			err: &SessionError{
				Op:        "GetSession",
				SessionID: "sess-123",
				Err:       errors.New("connection refused"),
			},
			wantMsg: "GetSession: session sess-123: connection refused",
		},
		{
			name: "without session ID",
			err: &SessionError{
				Op:  "ListSessions",
				Err: errors.New("timeout"),
			},
			wantMsg: "ListSessions: timeout",
		},
		{
			name: "without error",
			err: &SessionError{
				Op:        "UpdateSession",
				SessionID: "sess-456",
			},
			wantMsg: "UpdateSession: session sess-456: <nil>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.err.Error()
			if got != tt.wantMsg {
				t.Errorf("SessionError.Error() = %v, want %v", got, tt.wantMsg)
			}
		})
	}
}

func TestSessionError_Unwrap(t *testing.T) {
	baseErr := errors.New("base error")
	sessionErr := &SessionError{
		Op:  "test",
		Err: baseErr,
	}

	unwrapped := sessionErr.Unwrap()
	if unwrapped != baseErr {
		t.Errorf("SessionError.Unwrap() = %v, want %v", unwrapped, baseErr)
	}
}

func TestValidationError(t *testing.T) {
	err := &ValidationError{
		Field:   "SessionID",
		Message: "cannot be empty",
	}

	want := "validation error: SessionID cannot be empty"
	got := err.Error()

	if got != want {
		t.Errorf("ValidationError.Error() = %v, want %v", got, want)
	}
}

func TestNotFoundError(t *testing.T) {
	err := &NotFoundError{
		Resource: "session",
		ID:       "sess-999",
	}

	want := "session sess-999 not found"
	got := err.Error()

	if got != want {
		t.Errorf("NotFoundError.Error() = %v, want %v", got, want)
	}
}

func TestTimeoutError(t *testing.T) {
	tests := []struct {
		name    string
		err     *TimeoutError
		wantMsg string
	}{
		{
			name: "with duration",
			err: &TimeoutError{
				Operation: "WaitForResult",
				Duration:  30,
			},
			wantMsg: "operation WaitForResult timed out after 30s",
		},
		{
			name: "without duration",
			err: &TimeoutError{
				Operation: "Connect",
			},
			wantMsg: "operation Connect timed out",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.err.Error()
			if got != tt.wantMsg {
				t.Errorf("TimeoutError.Error() = %v, want %v", got, tt.wantMsg)
			}
		})
	}
}

func TestIsNotFound(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "NotFoundError",
			err:  &NotFoundError{Resource: "test", ID: "123"},
			want: true,
		},
		{
			name: "wrapped NotFoundError",
			err: &SessionError{
				Op:  "test",
				Err: &NotFoundError{Resource: "test", ID: "123"},
			},
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
			got := IsNotFound(tt.err)
			if got != tt.want {
				t.Errorf("IsNotFound() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsTimeout(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "TimeoutError",
			err:  &TimeoutError{Operation: "test"},
			want: true,
		},
		{
			name: "wrapped TimeoutError",
			err: &SessionError{
				Op:  "test",
				Err: &TimeoutError{Operation: "test"},
			},
			want: true,
		},
		{
			name: "context deadline exceeded",
			err:  errors.New("context deadline exceeded"),
			want: true,
		},
		{
			name: "i/o timeout",
			err:  errors.New("i/o timeout"),
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
			got := IsTimeout(tt.err)
			if got != tt.want {
				t.Errorf("IsTimeout() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsValidationError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "ValidationError",
			err:  &ValidationError{Field: "test", Message: "invalid"},
			want: true,
		},
		{
			name: "wrapped ValidationError",
			err: &SessionError{
				Op:  "test",
				Err: &ValidationError{Field: "test", Message: "invalid"},
			},
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
			got := IsValidationError(tt.err)
			if got != tt.want {
				t.Errorf("IsValidationError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWrapError(t *testing.T) {
	baseErr := errors.New("base error")

	wrapped := WrapError("TestOp", "sess-123", baseErr)

	sessionErr, ok := wrapped.(*SessionError)
	if !ok {
		t.Fatalf("WrapError() returned %T, want *SessionError", wrapped)
	}

	if sessionErr.Op != "TestOp" {
		t.Errorf("SessionError.Op = %v, want %v", sessionErr.Op, "TestOp")
	}

	if sessionErr.SessionID != "sess-123" {
		t.Errorf("SessionError.SessionID = %v, want %v", sessionErr.SessionID, "sess-123")
	}

	if sessionErr.Err != baseErr {
		t.Errorf("SessionError.Err = %v, want %v", sessionErr.Err, baseErr)
	}
}

func TestNewValidationError(t *testing.T) {
	err := NewValidationError("Username", "cannot contain spaces")

	validationErr, ok := err.(*ValidationError)
	if !ok {
		t.Fatalf("NewValidationError() returned %T, want *ValidationError", err)
	}

	if validationErr.Field != "Username" {
		t.Errorf("ValidationError.Field = %v, want %v", validationErr.Field, "Username")
	}

	if validationErr.Message != "cannot contain spaces" {
		t.Errorf("ValidationError.Message = %v, want %v", validationErr.Message, "cannot contain spaces")
	}
}

func TestNewNotFoundError(t *testing.T) {
	err := NewNotFoundError("beacon", "beacon-123")

	notFoundErr, ok := err.(*NotFoundError)
	if !ok {
		t.Fatalf("NewNotFoundError() returned %T, want *NotFoundError", err)
	}

	if notFoundErr.Resource != "beacon" {
		t.Errorf("NotFoundError.Resource = %v, want %v", notFoundErr.Resource, "beacon")
	}

	if notFoundErr.ID != "beacon-123" {
		t.Errorf("NotFoundError.ID = %v, want %v", notFoundErr.ID, "beacon-123")
	}
}

func TestNewTimeoutError(t *testing.T) {
	err := NewTimeoutError("Connect", 30)

	timeoutErr, ok := err.(*TimeoutError)
	if !ok {
		t.Fatalf("NewTimeoutError() returned %T, want *TimeoutError", err)
	}

	if timeoutErr.Operation != "Connect" {
		t.Errorf("TimeoutError.Operation = %v, want %v", timeoutErr.Operation, "Connect")
	}

	if timeoutErr.Duration != 30 {
		t.Errorf("TimeoutError.Duration = %v, want %v", timeoutErr.Duration, 30)
	}
}

func TestErrorMessages(t *testing.T) {
	// Test that error messages contain expected substrings
	tests := []struct {
		name     string
		err      error
		contains []string
	}{
		{
			name: "SessionError contains all parts",
			err: &SessionError{
				Op:        "UpdateSession",
				SessionID: "sess-789",
				Err:       errors.New("network error"),
			},
			contains: []string{"session", "UpdateSession", "sess-789", "network error"},
		},
		{
			name: "ValidationError contains field and message",
			err: &ValidationError{
				Field:   "Port",
				Message: "must be between 1 and 65535",
			},
			contains: []string{"validation error", "Port", "must be between 1 and 65535"},
		},
		{
			name: "NotFoundError contains resource and ID",
			err: &NotFoundError{
				Resource: "listener",
				ID:       "listen-456",
			},
			contains: []string{"listener", "listen-456", "not found"},
		},
		{
			name: "TimeoutError with duration",
			err: &TimeoutError{
				Operation: "WaitForTask",
				Duration:  60,
			},
			contains: []string{"WaitForTask", "timed out", "60s"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := tt.err.Error()
			for _, substr := range tt.contains {
				if !strings.Contains(msg, substr) {
					t.Errorf("Error message %q does not contain %q", msg, substr)
				}
			}
		})
	}
}
