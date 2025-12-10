package mock

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
)

// HTTPTestServer provides a mock HTTP server for testing
type HTTPTestServer struct {
	*httptest.Server
	mu             sync.Mutex
	requests       []*http.Request
	requestBodies  [][]byte
	handlers       map[string]http.HandlerFunc
	defaultHandler http.HandlerFunc
}

// NewHTTPTestServer creates a new mock HTTP test server
func NewHTTPTestServer() *HTTPTestServer {
	mock := &HTTPTestServer{
		requests:      make([]*http.Request, 0),
		requestBodies: make([][]byte, 0),
		handlers:      make(map[string]http.HandlerFunc),
	}

	// Set up the main handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mock.mu.Lock()
		defer mock.mu.Unlock()

		// Store request
		mock.requests = append(mock.requests, r)

		// Read and store body
		if r.Body != nil {
			body := make([]byte, 0)
			buf := make([]byte, 1024)
			for {
				n, err := r.Body.Read(buf)
				if n > 0 {
					body = append(body, buf[:n]...)
				}
				if err != nil {
					break
				}
			}
			mock.requestBodies = append(mock.requestBodies, body)
		} else {
			mock.requestBodies = append(mock.requestBodies, nil)
		}

		// Find handler
		key := fmt.Sprintf("%s %s", r.Method, r.URL.Path)
		if handler, ok := mock.handlers[key]; ok {
			handler(w, r)
			return
		}

		// Use default handler if set
		if mock.defaultHandler != nil {
			mock.defaultHandler(w, r)
			return
		}

		// Default response
		w.WriteHeader(http.StatusNotFound)
	})

	mock.Server = httptest.NewServer(handler)
	return mock
}

// SetHandler sets a handler for a specific method and path
func (m *HTTPTestServer) SetHandler(method, path string, handler http.HandlerFunc) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := fmt.Sprintf("%s %s", method, path)
	m.handlers[key] = handler
}

// SetDefaultHandler sets the default handler for unmatched requests
func (m *HTTPTestServer) SetDefaultHandler(handler http.HandlerFunc) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.defaultHandler = handler
}

// GetRequests returns all received requests
func (m *HTTPTestServer) GetRequests() []*http.Request {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Return a copy to prevent modification
	requests := make([]*http.Request, len(m.requests))
	copy(requests, m.requests)
	return requests
}

// GetRequestBodies returns all received request bodies
func (m *HTTPTestServer) GetRequestBodies() [][]byte {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Return a copy to prevent modification
	bodies := make([][]byte, len(m.requestBodies))
	for i, body := range m.requestBodies {
		if body != nil {
			bodyCopy := make([]byte, len(body))
			copy(bodyCopy, body)
			bodies[i] = bodyCopy
		}
	}
	return bodies
}

// GetLastRequest returns the last received request
func (m *HTTPTestServer) GetLastRequest() *http.Request {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.requests) == 0 {
		return nil
	}
	return m.requests[len(m.requests)-1]
}

// GetLastRequestBody returns the last received request body
func (m *HTTPTestServer) GetLastRequestBody() []byte {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.requestBodies) == 0 {
		return nil
	}

	body := m.requestBodies[len(m.requestBodies)-1]
	if body == nil {
		return nil
	}

	// Return a copy
	bodyCopy := make([]byte, len(body))
	copy(bodyCopy, body)
	return bodyCopy
}

// Reset clears all stored requests
func (m *HTTPTestServer) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.requests = make([]*http.Request, 0)
	m.requestBodies = make([][]byte, 0)
}

// GetRequestCount returns the number of requests received
func (m *HTTPTestServer) GetRequestCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.requests)
}

// Common response helpers

// RespondWithJSON responds with a JSON payload
func RespondWithJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

// RespondWithError responds with an error message
func RespondWithError(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]string{
		"error": message,
	})
}

// RespondWithSuccess responds with a success message
func RespondWithSuccess(w http.ResponseWriter, data interface{}) {
	RespondWithJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    data,
	})
}

// Predefined handlers for common scenarios

// NewAuthHandler creates a handler for authentication endpoints
func NewAuthHandler(token string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		RespondWithJSON(w, http.StatusOK, map[string]string{
			"token": token,
		})
	}
}

// NewSessionsHandler creates a handler for session list endpoints
func NewSessionsHandler(sessions []interface{}) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		RespondWithJSON(w, http.StatusOK, map[string]interface{}{
			"sessions": sessions,
		})
	}
}

// NewTaskHandler creates a handler for task submission endpoints
func NewTaskHandler(taskID string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		RespondWithJSON(w, http.StatusOK, map[string]string{
			"task_id": taskID,
		})
	}
}

// NewTaskResultHandler creates a handler for task result endpoints
func NewTaskResultHandler(output string, exitCode int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		RespondWithJSON(w, http.StatusOK, map[string]interface{}{
			"output":    output,
			"exit_code": exitCode,
		})
	}
}

// NewPayloadHandler creates a handler for payload generation endpoints
func NewPayloadHandler(payload, filename string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		RespondWithJSON(w, http.StatusOK, map[string]interface{}{
			"success":  true,
			"payload":  payload,
			"filename": filename,
		})
	}
}
