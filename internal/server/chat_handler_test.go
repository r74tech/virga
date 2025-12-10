package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/r74tech/virga/internal/server/protocol"
	"github.com/r74tech/virga/internal/server/session"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandleChat(t *testing.T) {
	tests := []struct {
		name           string
		sessionID      string
		requestBody    interface{}
		setupSession   bool
		expectedStatus int
		expectedFields map[string]interface{}
	}{
		{
			name:      "normal request",
			sessionID: "test-session-1",
			requestBody: map[string]interface{}{
				"message": "get process list",
				"options": map[string]interface{}{
					"maxIterations": 10,
					"temperature":   0.7,
				},
			},
			setupSession:   true,
			expectedStatus: http.StatusOK,
			expectedFields: map[string]interface{}{
				"success": true,
				"status":  "queued",
			},
		},
		{
			name:      "default values applied",
			sessionID: "test-session-2",
			requestBody: map[string]interface{}{
				"message": "tell me about the system",
			},
			setupSession:   true,
			expectedStatus: http.StatusOK,
			expectedFields: map[string]interface{}{
				"success": true,
				"status":  "queued",
			},
		},
		{
			name:      "empty message",
			sessionID: "test-session-3",
			requestBody: map[string]interface{}{
				"message": "",
			},
			setupSession:   true,
			expectedStatus: http.StatusBadRequest,
			expectedFields: map[string]interface{}{
				"success": false,
				"message": "Message required",
			},
		},
		{
			name:      "non-existent session",
			sessionID: "non-existent-session",
			requestBody: map[string]interface{}{
				"message": "test message",
			},
			setupSession:   false,
			expectedStatus: http.StatusNotFound,
			expectedFields: map[string]interface{}{
				"success": false,
				"message": "Session not found",
			},
		},
		{
			name:           "invalid request format",
			sessionID:      "test-session-4",
			requestBody:    "invalid json",
			setupSession:   true,
			expectedStatus: http.StatusBadRequest,
			expectedFields: map[string]interface{}{
				"success": false,
				"message": "Invalid request format",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// setup server and database
			server, db, _ := setupIntegrationTest(t)
			defer db.Close()

			// setup session
			if tt.setupSession {
				mockConn := &MockAgentConnection{
					remoteAddr:   "127.0.0.1:12345",
					responses:    make(chan []byte, 10),
					lastActivity: time.Now(),
				}
				sess := session.NewSession(tt.sessionID, mockConn)
				server.sessionsMu.Lock()
				server.sessions[tt.sessionID] = sess
				server.sessionsMu.Unlock()
			}

			// create request body
			var reqBody []byte
			var err error
			if str, ok := tt.requestBody.(string); ok {
				reqBody = []byte(str)
			} else {
				reqBody, err = json.Marshal(tt.requestBody)
				require.NoError(t, err)
			}

			// create HTTP request
			req := httptest.NewRequest(http.MethodPost, "/api/sessions/"+tt.sessionID+"/chat", bytes.NewReader(reqBody))
			req.Header.Set("Content-Type", "application/json")
			req = mux.SetURLVars(req, map[string]string{"id": tt.sessionID})

			// create response recorder
			rr := httptest.NewRecorder()

			// execute handler
			server.handleChat(rr, req)

			// check status code
			assert.Equal(t, tt.expectedStatus, rr.Code, "status code does not match")

			// check response body
			var response map[string]interface{}
			err = json.Unmarshal(rr.Body.Bytes(), &response)
			require.NoError(t, err, "failed to parse response JSON")

			// check expected fields
			for key, expectedValue := range tt.expectedFields {
				actualValue, ok := response[key]
				assert.True(t, ok, "field '%s' does not exist", key)
				assert.Equal(t, expectedValue, actualValue, "field '%s' value does not match", key)
			}

			// additional check for success
			if tt.expectedStatus == http.StatusOK {
				// check if taskId exists
				taskID, ok := response["taskId"].(string)
				assert.True(t, ok, "taskId is not a string")
				assert.NotEmpty(t, taskID, "taskId is empty")

				// check if task is added to session
				if tt.setupSession {
					server.sessionsMu.RLock()
					sess := server.sessions[tt.sessionID]
					server.sessionsMu.RUnlock()

					pendingTasks := sess.GetPendingTasks()
					assert.Greater(t, len(pendingTasks), 0, "task is not added to session")

					// check task content
					task := pendingTasks[len(pendingTasks)-1]
					assert.Equal(t, protocol.TaskType("llama_interactive"), task.Type, "task type does not match")
					assert.Equal(t, tt.sessionID, task.SessionID, "session ID does not match")

					// check payload
					payload, ok := task.Payload.(map[string]interface{})
					assert.True(t, ok, "payload is not a map")

					if reqMap, ok := tt.requestBody.(map[string]interface{}); ok {
						expectedMessage := reqMap["message"].(string)
						assert.Equal(t, expectedMessage, payload["prompt"], "prompt does not match")

						// check options
						if options, hasOptions := reqMap["options"].(map[string]interface{}); hasOptions {
							if maxIter, ok := options["maxIterations"].(int); ok {
								assert.Equal(t, maxIter, int(payload["max_iterations"].(int)), "max_iterations does not match")
							}
							if temp, ok := options["temperature"].(float64); ok {
								assert.Equal(t, temp, payload["temperature"].(float64), "temperature does not match")
							}
						} else {
							// check default values
							assert.Equal(t, 5, int(payload["max_iterations"].(int)), "default max_iterations does not match")
							assert.Equal(t, 0.3, payload["temperature"].(float64), "default temperature does not match")
						}
					}
				}
			}
		})
	}
}

func TestHandleChatCORS(t *testing.T) {
	// setup server
	server, db, _ := setupIntegrationTest(t)
	defer db.Close()

	// setup session
	sessionID := "test-session-cors"
	mockConn := &MockAgentConnection{
		remoteAddr:   "127.0.0.1:12345",
		responses:    make(chan []byte, 10),
		lastActivity: time.Now(),
	}
	sess := session.NewSession(sessionID, mockConn)
	server.sessionsMu.Lock()
	server.sessions[sessionID] = sess
	server.sessionsMu.Unlock()

	// create OPTIONS request
	req := httptest.NewRequest(http.MethodOptions, "/api/sessions/"+sessionID+"/chat", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	req = mux.SetURLVars(req, map[string]string{"id": sessionID})

	// create response recorder
	rr := httptest.NewRecorder()

	// execute handler with CORS middleware
	handler := server.corsMiddleware(http.HandlerFunc(server.handleChat))
	handler.ServeHTTP(rr, req)

	// check CORS headers
	assert.Equal(t, "http://localhost:5173", rr.Header().Get("Access-Control-Allow-Origin"))
	assert.Contains(t, rr.Header().Get("Access-Control-Allow-Methods"), "POST")
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestHandleChatWithDatabase(t *testing.T) {
	// setup server and database
	server, db, _ := setupIntegrationTest(t)
	defer db.Close()

	// setup session
	sessionID := "test-session-db"
	mockConn := &MockAgentConnection{
		remoteAddr:   "127.0.0.1:12345",
		responses:    make(chan []byte, 10),
		lastActivity: time.Now(),
	}
	sess := session.NewSession(sessionID, mockConn)
	server.sessionsMu.Lock()
	server.sessions[sessionID] = sess
	server.sessionsMu.Unlock()

	// save agent information to database
	err := db.SaveAgent(sessionID, "127.0.0.1:12345")
	require.NoError(t, err)

	// create session in database
	err = db.CreateSession(sessionID, sessionID)
	require.NoError(t, err)

	// create request
	requestBody := map[string]interface{}{
		"message": "database test message",
		"options": map[string]interface{}{
			"maxIterations": 3,
			"temperature":   0.5,
		},
	}
	reqBody, err := json.Marshal(requestBody)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/sessions/"+sessionID+"/chat", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"id": sessionID})

	// create response recorder
	rr := httptest.NewRecorder()

	// execute handler
	server.handleChat(rr, req)

	// check status code
	assert.Equal(t, http.StatusOK, rr.Code)

	// parse response
	var response map[string]interface{}
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	require.NoError(t, err)

	// get task ID
	taskID, ok := response["taskId"].(string)
	require.True(t, ok)
	require.NotEmpty(t, taskID)

	// check if command is recorded in database
	// Note: database.Database has appropriate methods to check this
	// here we check if the task mapping is set in the session
	commandID, exists := sess.GetTaskCommandMapping(taskID)
	assert.True(t, exists, "task and command mapping does not exist")
	assert.Greater(t, commandID, int64(0), "command ID is invalid")
}
