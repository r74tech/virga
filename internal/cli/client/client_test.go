package client

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"
	"time"

	"github.com/r74tech/virga/internal/cli/config"
	"github.com/r74tech/virga/internal/shared/protocol"
)

// RoundTripFunc allows us to implement http.RoundTripper for testing
type RoundTripFunc func(req *http.Request) *http.Response

func (f RoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req), nil
}

// NewTestClient creates a test client with a custom HTTP transport
func NewTestClient(fn RoundTripFunc) *http.Client {
	return &http.Client{
		Transport: fn,
	}
}

func TestNewAPIClient(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host: "localhost",
			Port: 8080,
		},
	}

	client := NewAPIClient(cfg)
	if client == nil {
		t.Fatal("Expected client, got nil")
	}

	if client.Config != cfg {
		t.Error("Config mismatch")
	}

	if client.HttpClient == nil {
		t.Error("HttpClient is nil")
	}

	if client.IsDebugEnabled() {
		t.Error("Debug should be false by default")
	}
}

func TestSetDebugMode(t *testing.T) {
	cfg := &config.Config{}
	client := NewAPIClient(cfg)

	// Enable debug mode
	client.SetDebugMode(true)
	if !client.IsDebugEnabled() {
		t.Error("Expected debug to be true")
	}

	// Disable debug mode
	client.SetDebugMode(false)
	if client.IsDebugEnabled() {
		t.Error("Expected debug to be false")
	}
}

func TestGetServerURL(t *testing.T) {
	tests := []struct {
		name     string
		config   *config.Config
		expected string
	}{
		{
			name: "Default HTTP",
			config: &config.Config{
				Server: config.ServerConfig{
					Host: "localhost",
					Port: 8080,
				},
			},
			expected: "http://localhost:8080",
		},
		{
			name: "Custom host and port",
			config: &config.Config{
				Server: config.ServerConfig{
					Host: "192.168.1.100",
					Port: 9090,
				},
			},
			expected: "http://192.168.1.100:9090",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewAPIClient(tt.config)
			url := client.GetServerURL()
			if url != tt.expected {
				t.Errorf("Expected URL %s, got %s", tt.expected, url)
			}
		})
	}
}

func TestAuthenticate(t *testing.T) {
	cfg := &config.Config{}
	client := NewAPIClient(cfg)

	// First authentication
	err := client.Authenticate()
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if client.GetAuthToken() == "" {
		t.Error("Expected auth token to be set")
	}

	// Second authentication (should skip)
	firstToken := client.GetAuthToken()
	err = client.Authenticate()
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if client.GetAuthToken() != firstToken {
		t.Error("Token should not change on second authentication")
	}
}

func TestGetSessions(t *testing.T) {
	tests := []struct {
		name         string
		mockResponse string
		statusCode   int
		wantSessions int
		wantErr      bool
	}{
		{
			name: "Successful response",
			mockResponse: `{
				"success": true,
				"sessions": [
					{
						"id": "session-1",
						"remote_addr": "192.168.1.100:12345",
						"hostname": "host1",
						"username": "user1",
						"os": "linux"
					},
					{
						"id": "session-2",
						"remote_addr": "192.168.1.101:12346",
						"hostname": "host2",
						"username": "user2",
						"os": "windows"
					}
				]
			}`,
			statusCode:   200,
			wantSessions: 2,
			wantErr:      false,
		},
		{
			name:         "Empty sessions",
			mockResponse: `{"success": true, "sessions": []}`,
			statusCode:   200,
			wantSessions: 0,
			wantErr:      false,
		},
		{
			name:         "Server error",
			mockResponse: `{"error": "Internal server error"}`,
			statusCode:   500,
			wantSessions: 0,
			wantErr:      false, // Client returns empty array on error
		},
		{
			name:         "Invalid JSON",
			mockResponse: `invalid json`,
			statusCode:   200,
			wantSessions: 0,
			wantErr:      false, // Client returns empty array on parse error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				Server: config.ServerConfig{
					Host: "localhost",
					Port: 8080,
				},
			}
			client := NewAPIClient(cfg)

			// Set up mock HTTP transport
			client.HttpClient = NewTestClient(func(req *http.Request) *http.Response {
				// Verify request
				if req.URL.Path != "/api/sessions" {
					t.Errorf("Expected path /api/sessions, got %s", req.URL.Path)
				}
				if req.Method != "GET" {
					t.Errorf("Expected method GET, got %s", req.Method)
				}

				return &http.Response{
					StatusCode: tt.statusCode,
					Body:       io.NopCloser(bytes.NewBufferString(tt.mockResponse)),
					Header:     make(http.Header),
				}
			})

			sessions, err := client.GetSessions()
			if tt.wantErr {
				if err == nil {
					t.Error("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
				if len(sessions) != tt.wantSessions {
					t.Errorf("Expected %d sessions, got %d", tt.wantSessions, len(sessions))
				}
			}
		})
	}
}

func TestSetSessionInteractive(t *testing.T) {
	tests := []struct {
		name         string
		sessionID    string
		interactive  bool
		mockResponse string
		statusCode   int
		wantErr      bool
	}{
		{
			name:         "Successfully set interactive",
			sessionID:    "session-123",
			interactive:  true,
			mockResponse: `{"success": true}`,
			statusCode:   200,
			wantErr:      false,
		},
		{
			name:         "Session not found",
			sessionID:    "invalid-session",
			interactive:  true,
			mockResponse: `{"error": "Session not found"}`,
			statusCode:   404,
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				Server: config.ServerConfig{
					Host: "localhost",
					Port: 8080,
				},
			}
			client := NewAPIClient(cfg)

			client.HttpClient = NewTestClient(func(req *http.Request) *http.Response {
				expectedPath := "/api/sessions/" + tt.sessionID + "/interactive"
				if req.URL.Path != expectedPath {
					t.Errorf("Expected path %s, got %s", expectedPath, req.URL.Path)
				}
				if req.Method != "POST" {
					t.Errorf("Expected method POST, got %s", req.Method)
				}

				// Verify request body
				var body map[string]interface{}
				if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
					t.Fatalf("Failed to decode request body: %v", err)
				}
				if body["interactive"] != tt.interactive {
					t.Errorf("Expected interactive %v, got %v", tt.interactive, body["interactive"])
				}

				return &http.Response{
					StatusCode: tt.statusCode,
					Body:       io.NopCloser(bytes.NewBufferString(tt.mockResponse)),
					Header:     make(http.Header),
				}
			})

			err := client.SetSessionInteractive(tt.sessionID, tt.interactive)
			if tt.wantErr {
				if err == nil {
					t.Error("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
			}
		})
	}
}

func TestSendSessionCommand(t *testing.T) {
	tests := []struct {
		name         string
		sessionID    string
		cmdType      string
		payload      map[string]interface{}
		mockResponse string
		statusCode   int
		wantTaskID   string
		wantErr      bool
	}{
		{
			name:      "Successful command",
			sessionID: "session-123",
			cmdType:   "shell",
			payload: map[string]interface{}{
				"command": "whoami",
			},
			mockResponse: `{
				"success": true,
				"task_id": "task-456",
				"message": "Command sent"
			}`,
			statusCode: 200,
			wantTaskID: "task-456",
			wantErr:    false,
		},
		{
			name:      "Llama command",
			sessionID: "session-123",
			cmdType:   "llama",
			payload: map[string]interface{}{
				"prompt":      "Find all processes",
				"iterations":  5,
				"temperature": 0.7,
			},
			mockResponse: `{
				"success": true,
				"task_id": "task-789"
			}`,
			statusCode: 200,
			wantTaskID: "task-789",
			wantErr:    false,
		},
		{
			name:      "Session not found",
			sessionID: "invalid-session",
			cmdType:   "shell",
			payload: map[string]interface{}{
				"command": "ls",
			},
			mockResponse: `{
				"success": false,
				"message": "Session not found"
			}`,
			statusCode: 404,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				Server: config.ServerConfig{
					Host: "localhost",
					Port: 8080,
				},
			}
			client := NewAPIClient(cfg)

			client.HttpClient = NewTestClient(func(req *http.Request) *http.Response {
				expectedPath := "/api/sessions/" + tt.sessionID + "/tasks"
				if req.URL.Path != expectedPath {
					t.Errorf("Expected path %s, got %s", expectedPath, req.URL.Path)
				}
				if req.Method != "POST" {
					t.Errorf("Expected method POST, got %s", req.Method)
				}

				// Verify request body
				var body map[string]interface{}
				if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
					t.Fatalf("Failed to decode request body: %v", err)
				}
				if body["type"] != tt.cmdType {
					t.Errorf("Expected type %s, got %v", tt.cmdType, body["type"])
				}

				return &http.Response{
					StatusCode: tt.statusCode,
					Body:       io.NopCloser(bytes.NewBufferString(tt.mockResponse)),
					Header:     make(http.Header),
				}
			})

			taskID, err := client.SendSessionCommand(tt.sessionID, tt.cmdType, tt.payload)
			if tt.wantErr {
				if err == nil {
					t.Error("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
				if taskID != tt.wantTaskID {
					t.Errorf("Expected task ID %s, got %s", tt.wantTaskID, taskID)
				}
			}
		})
	}
}

func TestGetTaskResult(t *testing.T) {
	tests := []struct {
		name         string
		sessionID    string
		taskID       string
		mockResponse string
		statusCode   int
		wantResult   bool
		wantErr      bool
	}{
		{
			name:      "Task completed",
			sessionID: "session-123",
			taskID:    "task-456",
			mockResponse: `{
				"success": true,
				"task_id": "task-456",
				"output": "user1",
				"exit_code": 0,
				"error": "",
				"time": "2025-01-20T10:30:00Z",
				"is_complete": true
			}`,
			statusCode: 200,
			wantResult: true,
			wantErr:    false,
		},
		{
			name:       "Task not found (404)",
			sessionID:  "session-123",
			taskID:     "invalid-task",
			statusCode: 404,
			wantResult: false,
			wantErr:    false, // 404 returns nil, nil
		},
		{
			name:      "API error",
			sessionID: "session-123",
			taskID:    "task-456",
			mockResponse: `{
				"success": false
			}`,
			statusCode: 200,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				Server: config.ServerConfig{
					Host: "localhost",
					Port: 8080,
				},
			}
			client := NewAPIClient(cfg)

			client.HttpClient = NewTestClient(func(req *http.Request) *http.Response {
				expectedPath := "/api/sessions/" + tt.sessionID + "/tasks/" + tt.taskID
				if req.URL.Path != expectedPath {
					t.Errorf("Expected path %s, got %s", expectedPath, req.URL.Path)
				}
				if req.Method != "GET" {
					t.Errorf("Expected method GET, got %s", req.Method)
				}

				return &http.Response{
					StatusCode: tt.statusCode,
					Body:       io.NopCloser(bytes.NewBufferString(tt.mockResponse)),
					Header:     make(http.Header),
				}
			})

			result, err := client.GetTaskResult(tt.sessionID, tt.taskID)
			if tt.wantErr {
				if err == nil {
					t.Error("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
				if tt.wantResult && result == nil {
					t.Error("Expected result, got nil")
				}
				if !tt.wantResult && result != nil {
					t.Error("Expected nil result, got value")
				}
			}
		})
	}
}

func TestGetTaskResults(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host: "localhost",
			Port: 8080,
		},
	}
	client := NewAPIClient(cfg)

	mockResponse := `{
		"success": true,
		"results": [
			{
				"task_id": "task-1",
				"output": "result1",
				"exit_code": 0
			},
			{
				"task_id": "task-2",
				"output": "result2",
				"exit_code": 0
			}
		]
	}`

	client.HttpClient = NewTestClient(func(req *http.Request) *http.Response {
		if req.URL.Path != "/api/sessions/session-123/tasks" {
			t.Errorf("Expected path /api/sessions/session-123/tasks, got %s", req.URL.Path)
		}
		if req.Method != "GET" {
			t.Errorf("Expected method GET, got %s", req.Method)
		}

		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(bytes.NewBufferString(mockResponse)),
			Header:     make(http.Header),
		}
	})

	results, err := client.GetTaskResults("session-123")
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}
}

func TestGeneratePayload(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		response := `{"success": true, "payload": "dGVzdC1wYXlsb2Fk", "filename": "test.exe"}`
		w.Write([]byte(response))
	}))
	defer mockServer.Close()

	serverURL, err := url.Parse(mockServer.URL)
	if err != nil {
		t.Fatalf("Failed to parse mock server URL: %v", err)
	}
	port, _ := strconv.Atoi(serverURL.Port())

	tests := []struct {
		name         string
		payloadType  string
		listenerName string
		wantErr      bool
	}{
		{
			name:         "PowerShell payload",
			payloadType:  "powershell",
			listenerName: "default-http",
			wantErr:      false,
		},
		{
			name:         "EXE payload",
			payloadType:  "exe",
			listenerName: "default-http",
			wantErr:      false,
		},
		{
			name:         "DLL payload",
			payloadType:  "dll",
			listenerName: "default-http",
			wantErr:      false,
		},
		{
			name:         "Unsupported payload type",
			payloadType:  "unknown",
			listenerName: "default-http",
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				Server: config.ServerConfig{
					Host: serverURL.Hostname(),
					Port: port,
				},
			}
			client := NewAPIClient(cfg)

			payload, filename, err := client.GeneratePayload(tt.payloadType, tt.listenerName)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
				if len(payload) == 0 {
					t.Error("Expected payload data, got empty")
				}
				if filename == "" {
					t.Error("Expected filename, got empty")
				}
			}
		})
	}
}

func TestGetListeners(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host: "localhost",
			Port: 8080,
		},
	}
	client := NewAPIClient(cfg)

	listeners, err := client.GetListeners()
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if len(listeners) == 0 {
		t.Error("Expected at least one listener")
	}

	// Check the default listener
	if len(listeners) > 0 {
		defaultListener := listeners[0]
		if defaultListener["name"] != "default-http" {
			t.Errorf("Expected name 'default-http', got %v", defaultListener["name"])
		}
		if defaultListener["type"] != "http" {
			t.Errorf("Expected type 'http', got %v", defaultListener["type"])
		}
	}
}

func TestClientWithAuth(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host: "localhost",
			Port: 8080,
		},
	}
	client := NewAPIClient(cfg)
	client.SetAuthToken("test-auth-token")

	client.HttpClient = NewTestClient(func(req *http.Request) *http.Response {
		// Verify auth header
		authHeader := req.Header.Get("Authorization")
		expectedHeader := "Bearer test-auth-token"
		if authHeader != expectedHeader {
			t.Errorf("Expected auth header %s, got %s", expectedHeader, authHeader)
		}

		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(bytes.NewBufferString(`{"success": true, "sessions": []}`)),
			Header:     make(http.Header),
		}
	})

	_, err := client.GetSessions()
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
}

// TestTaskResultStructure verifies the protocol.TaskResult structure
func TestTaskResultStructure(t *testing.T) {
	// This test verifies that protocol.TaskResult has the expected fields
	result := &protocol.TaskResult{
		TaskID:   "task-123",
		Output:   "test output",
		ExitCode: 0,
		Error:    "",
		Time:     time.Now(),
	}

	if result.TaskID != "task-123" {
		t.Errorf("Expected TaskID 'task-123', got %s", result.TaskID)
	}
	if result.Output != "test output" {
		t.Errorf("Expected Output 'test output', got %s", result.Output)
	}
	if result.ExitCode != 0 {
		t.Errorf("Expected ExitCode 0, got %d", result.ExitCode)
	}
}

// BenchmarkGetSessions tests performance of session retrieval
func BenchmarkGetSessions(b *testing.B) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host: "localhost",
			Port: 8080,
		},
	}
	client := NewAPIClient(cfg)

	client.HttpClient = NewTestClient(func(req *http.Request) *http.Response {
		return &http.Response{
			StatusCode: 200,
			Body: io.NopCloser(bytes.NewBufferString(`{
				"success": true,
				"sessions": [
					{"id": "session-1", "hostname": "host1"},
					{"id": "session-2", "hostname": "host2"},
					{"id": "session-3", "hostname": "host3"}
				]
			}`)),
			Header: make(http.Header),
		}
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = client.GetSessions()
	}
}
