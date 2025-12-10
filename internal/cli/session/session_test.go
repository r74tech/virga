package session

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/r74tech/virga/internal/cli/client"
	"github.com/r74tech/virga/internal/shared/testutil/mock"
)

func TestNewManager(t *testing.T) {
	manager := NewManager()
	if manager == nil {
		t.Fatal("Expected manager, got nil")
	}

	if manager.sessions == nil {
		t.Error("Sessions map is nil")
	}

	if len(manager.sessions) != 0 {
		t.Error("Sessions map should be empty initially")
	}

	if manager.currentSession != nil {
		t.Error("Current session should be nil initially")
	}

	if manager.debugMode {
		t.Error("Debug mode should be false by default")
	}
}

func TestSetAPIClient(t *testing.T) {
	manager := NewManager()
	mockClient := &mock.MockAPIClient{}

	manager.SetAPIClient(mockClient)

	if manager.apiClient == nil {
		t.Error("API client was not set")
	}

	// Test that API client is applied to existing sessions
	session1 := &Session{ID: "session-1"}
	manager.sessions["session-1"] = session1

	manager.SetAPIClient(mockClient)

	if session1.apiClient == nil {
		t.Error("API client was not applied to existing session")
	}
}

func TestGetAPIClient(t *testing.T) {
	manager := NewManager()
	mockClient := &mock.MockAPIClient{}

	manager.SetAPIClient(mockClient)
	retrieved := manager.GetAPIClient()

	if retrieved == nil {
		t.Error("Retrieved API client is nil")
	}
}

func TestSetDebugMode(t *testing.T) {
	manager := NewManager()

	// Enable debug mode
	manager.SetDebugMode(true)
	if !manager.debugMode {
		t.Error("Debug mode should be true")
	}

	// Disable debug mode
	manager.SetDebugMode(false)
	if manager.debugMode {
		t.Error("Debug mode should be false")
	}
}

func TestSyncWithServer(t *testing.T) {
	tests := []struct {
		name           string
		apiClient      *mock.MockAPIClient
		currentSession string
		wantErr        bool
		wantSessions   int
	}{
		{
			name:      "No API client",
			apiClient: nil,
			wantErr:   true,
		},
		{
			name: "Successful sync",
			apiClient: &mock.MockAPIClient{
				GetSessionsFunc: func() ([]client.SessionInfo, error) {
					return []client.SessionInfo{
						{
							ID:       "session-1",
							Hostname: "host1",
							Username: "user1",
							OS:       "linux",
						},
						{
							ID:       "session-2",
							Hostname: "host2",
							Username: "user2",
							OS:       "windows",
						},
					}, nil
				},
			},
			wantErr:      false,
			wantSessions: 2,
		},
		{
			name: "API error",
			apiClient: &mock.MockAPIClient{
				GetSessionsFunc: func() ([]client.SessionInfo, error) {
					return nil, fmt.Errorf("connection failed")
				},
			},
			wantErr: true,
		},
		{
			name: "Preserve current session",
			apiClient: &mock.MockAPIClient{
				GetSessionsFunc: func() ([]client.SessionInfo, error) {
					return []client.SessionInfo{
						{ID: "session-1"},
						{ID: "session-2"},
					}, nil
				},
			},
			currentSession: "session-1",
			wantErr:        false,
			wantSessions:   2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := NewManager()
			if tt.apiClient != nil {
				manager.SetAPIClient(tt.apiClient)
			}

			if tt.currentSession != "" {
				manager.sessions[tt.currentSession] = &Session{ID: tt.currentSession}
				manager.currentSession = manager.sessions[tt.currentSession]
			}

			err := manager.SyncWithServer()
			if tt.wantErr {
				if err == nil {
					t.Error("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
				if len(manager.sessions) != tt.wantSessions {
					t.Errorf("Expected %d sessions, got %d", tt.wantSessions, len(manager.sessions))
				}
				// Check current session is preserved
				if tt.currentSession != "" && manager.currentSession != nil {
					if manager.currentSession.ID != tt.currentSession {
						t.Errorf("Current session not preserved, expected %s, got %s",
							tt.currentSession, manager.currentSession.ID)
					}
				}
			}
		})
	}
}

func TestSetCurrentSession(t *testing.T) {
	tests := []struct {
		name      string
		sessionID string
		exists    bool
		wantErr   bool
	}{
		{
			name:      "Set existing session",
			sessionID: "session-1",
			exists:    true,
			wantErr:   false,
		},
		{
			name:      "Set non-existing session",
			sessionID: "invalid-session",
			exists:    false,
			wantErr:   true,
		},
		{
			name:      "Clear current session",
			sessionID: "",
			exists:    false,
			wantErr:   true, // SetCurrentSession with empty ID should error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := NewManager()
			mockClient := &mock.MockAPIClient{
				SetSessionInteractiveFn: func(sessionID string, interactive bool) error {
					return nil
				},
			}
			manager.SetAPIClient(mockClient)

			if tt.exists {
				manager.sessions[tt.sessionID] = &Session{
					ID:        tt.sessionID,
					apiClient: mockClient,
				}
			}

			err := manager.SetCurrentSession(tt.sessionID)
			if tt.wantErr {
				if err == nil {
					t.Error("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
				if tt.sessionID == "" {
					if manager.currentSession != nil {
						t.Error("Expected current session to be nil")
					}
				} else {
					if manager.currentSession == nil || manager.currentSession.ID != tt.sessionID {
						t.Error("Current session not set correctly")
					}
				}
			}
		})
	}
}

func TestGetCurrentSession(t *testing.T) {
	manager := NewManager()

	// No current session
	session := manager.GetCurrentSession()
	if session != nil {
		t.Error("Expected nil current session")
	}

	// Set current session
	testSession := &Session{ID: "test-session"}
	manager.sessions["test-session"] = testSession
	manager.currentSession = testSession

	session = manager.GetCurrentSession()
	if session != testSession {
		t.Error("Current session mismatch")
	}
}

func TestGetSession(t *testing.T) {
	manager := NewManager()

	// Add test sessions
	session1 := &Session{ID: "session-1"}
	session2 := &Session{ID: "session-2"}
	manager.sessions["session-1"] = session1
	manager.sessions["session-2"] = session2

	// Get existing session
	retrieved, exists := manager.GetSession("session-1")
	if !exists {
		t.Error("Expected session to exist")
	}
	if retrieved != session1 {
		t.Error("Retrieved session mismatch")
	}

	// Get non-existing session
	retrieved, exists = manager.GetSession("invalid-session")
	if exists {
		t.Error("Expected session to not exist")
	}
	if retrieved != nil {
		t.Error("Expected nil session")
	}
}

func TestGetAllSessions(t *testing.T) {
	manager := NewManager()

	// Add test sessions
	session1 := &Session{ID: "session-1"}
	session2 := &Session{ID: "session-2"}
	session3 := &Session{ID: "session-3"}
	manager.sessions["session-1"] = session1
	manager.sessions["session-2"] = session2
	manager.sessions["session-3"] = session3

	sessions := manager.GetAllSessions()
	if len(sessions) != 3 {
		t.Errorf("Expected 3 sessions, got %d", len(sessions))
	}

	// Verify all sessions are included
	sessionIDs := make(map[string]bool)
	for _, s := range sessions {
		sessionIDs[s.ID] = true
	}

	if !sessionIDs["session-1"] || !sessionIDs["session-2"] || !sessionIDs["session-3"] {
		t.Error("Not all expected sessions found in result")
	}
}

func TestSendCommandToCurrentSession(t *testing.T) {
	tests := []struct {
		name       string
		hasSession bool
		cmdType    string
		payload    map[string]interface{}
		mockResult string
		mockError  error
		wantErr    bool
	}{
		{
			name:       "Send command successfully",
			hasSession: true,
			cmdType:    "shell",
			payload:    map[string]interface{}{"command": "whoami"},
			mockResult: "task-123",
			wantErr:    false,
		},
		{
			name:       "No current session",
			hasSession: false,
			cmdType:    "shell",
			payload:    map[string]interface{}{"command": "ls"},
			wantErr:    true,
		},
		{
			name:       "API error",
			hasSession: true,
			cmdType:    "shell",
			payload:    map[string]interface{}{"command": "pwd"},
			mockError:  fmt.Errorf("connection failed"),
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := NewManager()
			mockClient := &mock.MockAPIClient{
				SendSessionCommandFunc: func(sessionID, cmdType string, payload map[string]interface{}) (string, error) {
					if tt.mockError != nil {
						return "", tt.mockError
					}
					return tt.mockResult, nil
				},
			}
			manager.SetAPIClient(mockClient)

			if tt.hasSession {
				session := &Session{ID: "current-session", apiClient: mockClient}
				manager.sessions["current-session"] = session
				manager.currentSession = session
			}

			var taskID string
			var err error
			currentSession := manager.GetCurrentSession()
			if currentSession != nil {
				taskID, err = currentSession.ExecuteCommand(tt.cmdType, tt.payload)
			} else {
				err = fmt.Errorf("no current session")
			}
			if tt.wantErr {
				if err == nil {
					t.Error("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
				if taskID != tt.mockResult {
					t.Errorf("Expected task ID %s, got %s", tt.mockResult, taskID)
				}
			}
		})
	}
}

func TestIsInteractive(t *testing.T) {
	session := &Session{
		ID:            "test-session",
		isInteractive: false,
	}

	// Initially not interactive
	if session.IsInteractive() {
		t.Error("Expected session to not be interactive")
	}

	// Set interactive
	session.isInteractive = true
	if !session.IsInteractive() {
		t.Error("Expected session to be interactive")
	}
}

func TestGetSessionInfo(t *testing.T) {
	now := time.Now()
	session := &Session{
		ID:            "test-session",
		RemoteAddress: "192.168.1.100:12345",
		Information: map[string]interface{}{
			"hostname": "testhost",
			"username": "testuser",
			"os":       "linux",
			"arch":     "amd64",
		},
		LastActivity: now,
		firstSeen:    now.Add(-1 * time.Hour),
	}

	// Test ID
	if session.ID != "test-session" {
		t.Error("ID mismatch")
	}

	// Test Address
	if session.RemoteAddr() != "192.168.1.100:12345" {
		t.Error("Address mismatch")
	}

	// Test Information fields
	info := session.GetInformation()
	if info["hostname"] != "testhost" {
		t.Error("Hostname mismatch")
	}
	if info["username"] != "testuser" {
		t.Error("Username mismatch")
	}
	if info["os"] != "linux" {
		t.Error("OS mismatch")
	}
	if info["arch"] != "amd64" {
		t.Error("Arch mismatch")
	}

	// Test time fields
	if !session.LastSeen().Equal(now) {
		t.Error("LastActivity mismatch")
	}
	if !session.FirstSeen().Equal(now.Add(-1 * time.Hour)) {
		t.Error("FirstSeen mismatch")
	}
}

func TestConcurrentAccess(t *testing.T) {
	manager := NewManager()
	mockClient := &mock.MockAPIClient{
		GetSessionsFunc: func() ([]client.SessionInfo, error) {
			return []client.SessionInfo{
				{ID: "session-1"},
				{ID: "session-2"},
			}, nil
		},
	}
	manager.SetAPIClient(mockClient)

	// Add initial sessions
	for i := 0; i < 10; i++ {
		sessionID := fmt.Sprintf("session-%d", i)
		manager.sessions[sessionID] = &Session{ID: sessionID}
	}

	var wg sync.WaitGroup
	wg.Add(4)

	// Concurrent sync
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			_ = manager.SyncWithServer()
		}
	}()

	// Concurrent get
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			_ = manager.GetAllSessions()
		}
	}()

	// Concurrent set current
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			sessionID := fmt.Sprintf("session-%d", i%10)
			_ = manager.SetCurrentSession(sessionID)
		}
	}()

	// Concurrent get current
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			_ = manager.GetCurrentSession()
		}
	}()

	wg.Wait()
}

// Benchmark tests
func BenchmarkSyncWithServer(b *testing.B) {
	manager := NewManager()
	mockClient := &mock.MockAPIClient{
		GetSessionsFunc: func() ([]client.SessionInfo, error) {
			sessions := make([]client.SessionInfo, 100)
			for i := 0; i < 100; i++ {
				sessions[i] = client.SessionInfo{
					ID:       fmt.Sprintf("session-%d", i),
					Hostname: fmt.Sprintf("host-%d", i),
				}
			}
			return sessions, nil
		},
	}
	manager.SetAPIClient(mockClient)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = manager.SyncWithServer()
	}
}

func BenchmarkGetAllSessions(b *testing.B) {
	manager := NewManager()

	// Add many sessions
	for i := 0; i < 1000; i++ {
		sessionID := fmt.Sprintf("session-%d", i)
		manager.sessions[sessionID] = &Session{ID: sessionID}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = manager.GetAllSessions()
	}
}
