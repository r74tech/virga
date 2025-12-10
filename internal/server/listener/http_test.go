package listener

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/r74tech/virga/internal/server/config"
	"github.com/r74tech/virga/internal/server/protocol"
	"github.com/r74tech/virga/internal/shared/crypto"
)

func createTestListener() *HTTPListener {
	cfg := config.ListenerConfig{
		Name:        "test-listener",
		Type:        "http",
		BindAddress: "127.0.0.1",
		Port:        8080,
		URIPath:     "test",
		UseSSL:      false,
		Encryption: config.EncryptionConfig{
			Key: "0123456789abcdef0123456789abcdef", // 32 bytes for AES-256
		},
	}

	return NewHTTPListener(cfg).(*HTTPListener)
}

func TestNewHTTPListener(t *testing.T) {
	listener := createTestListener()

	if listener.config.Name != "test-listener" {
		t.Errorf("Expected name 'test-listener', got '%s'", listener.config.Name)
	}

	if listener.server.Addr != "127.0.0.1:8080" {
		t.Errorf("Expected address '127.0.0.1:8080', got '%s'", listener.server.Addr)
	}

	if len(listener.aesKey) != 32 {
		t.Errorf("Expected AES key length 32, got %d", len(listener.aesKey))
	}
}

func TestHTTPListener_Name(t *testing.T) {
	listener := createTestListener()
	if listener.Name() != "test-listener" {
		t.Errorf("Expected name 'test-listener', got '%s'", listener.Name())
	}
}

func TestHTTPListener_SetDebugMode(t *testing.T) {
	listener := createTestListener()

	// Initially should be false
	if listener.debugMode {
		t.Error("Debug mode should be false by default")
	}

	// Enable debug mode
	listener.SetDebugMode(true)
	if !listener.debugMode {
		t.Error("Debug mode should be true after setting")
	}

	// Disable debug mode
	listener.SetDebugMode(false)
	if listener.debugMode {
		t.Error("Debug mode should be false after unsetting")
	}
}

func TestHTTPListener_HandleAgent(t *testing.T) {
	listener := createTestListener()
	listener.SetDebugMode(true)

	// Track agent handler calls
	var handlerCalled bool
	var capturedAgentID string
	var capturedConn protocol.AgentConnection

	listener.agentHandler = func(agentID string, conn protocol.AgentConnection) {
		handlerCalled = true
		capturedAgentID = agentID
		capturedConn = conn
	}

	// Create test agent message
	agentMsg := protocol.AgentMessage{
		AgentID:  "test-agent-123",
		Hostname: "test-host",
		Username: "test-user",
		OS:       "linux",
		Version:  "1.0",
		IP:       "192.168.1.100",
		BootTime: time.Now().Unix(),
		Environment: map[string]interface{}{
			"test": "value",
		},
		TaskResults: []protocol.AgentTaskResult{
			{
				TaskID:    "task-1",
				Output:    "test output",
				ExitCode:  0,
				Timestamp: time.Now().Unix(),
			},
		},
	}

	// Marshal and encrypt the message
	jsonData, err := json.Marshal(agentMsg)
	if err != nil {
		t.Fatalf("Failed to marshal agent message: %v", err)
	}

	encryptedData, err := crypto.Encrypt(jsonData, listener.aesKey)
	if err != nil {
		t.Fatalf("Failed to encrypt data: %v", err)
	}

	encodedData := base64.StdEncoding.EncodeToString(encryptedData)

	// Create test request
	req := httptest.NewRequest("POST", "/test", bytes.NewBufferString(encodedData))
	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("User-Agent", "Test-Agent/1.0")

	// Create response recorder
	w := httptest.NewRecorder()

	// Handle the request
	listener.handleAgent(w, req)

	// Check response
	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Errorf("Expected status 200, got %d. Body: %s", resp.StatusCode, string(body))
	}

	// Verify agent handler was called
	if !handlerCalled {
		t.Error("Agent handler was not called")
	}

	if capturedAgentID != "test-agent-123" {
		t.Errorf("Expected agent ID 'test-agent-123', got '%s'", capturedAgentID)
	}

	if capturedConn == nil {
		t.Error("Captured connection is nil")
	}
}

func TestHTTPListener_HandleAgent_Errors(t *testing.T) {
	listener := createTestListener()

	tests := []struct {
		name           string
		requestBody    string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "Empty body",
			requestBody:    "",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Empty request body",
		},
		{
			name:           "Invalid base64",
			requestBody:    "not-valid-base64!@#$",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid base64 data",
		},
		{
			name:           "Invalid encryption",
			requestBody:    base64.StdEncoding.EncodeToString([]byte("invalid-encrypted-data")),
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Decryption failed",
		},
		{
			name: "Invalid JSON",
			requestBody: func() string {
				encrypted, _ := crypto.Encrypt([]byte("invalid-json"), listener.aesKey)
				return base64.StdEncoding.EncodeToString(encrypted)
			}(),
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid JSON format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/test", bytes.NewBufferString(tt.requestBody))
			w := httptest.NewRecorder()

			listener.handleAgent(w, req)

			resp := w.Result()
			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, resp.StatusCode)
			}

			body, _ := io.ReadAll(resp.Body)
			bodyStr := string(body)
			if bodyStr != tt.expectedError+"\n" {
				t.Errorf("Expected error '%s', got '%s'", tt.expectedError, bodyStr)
			}
		})
	}
}

func TestHTTPListener_HandleStage2(t *testing.T) {
	listener := createTestListener()

	req := httptest.NewRequest("GET", "/test/stage2", nil)
	w := httptest.NewRecorder()

	listener.handleStage2(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	if len(body) == 0 {
		t.Error("Expected non-empty stage2 payload")
	}
}

func TestHTTPListener_RootHandler(t *testing.T) {
	router := mux.NewRouter()

	// Set up root handler
	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("<html><body><h1>404 - Page Not Found</h1></body></html>"))
	})

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	expected := "<html><body><h1>404 - Page Not Found</h1></body></html>"
	if string(body) != expected {
		t.Errorf("Expected body '%s', got '%s'", expected, string(body))
	}
}

func TestHTTPListener_StartStop(t *testing.T) {
	cfg := config.ListenerConfig{
		Name:        "test-start-stop",
		Type:        "http",
		BindAddress: "127.0.0.1",
		Port:        0, // Use random port
		URIPath:     "test",
		UseSSL:      false,
		Encryption: config.EncryptionConfig{
			Key: "0123456789abcdef0123456789abcdef",
		},
	}

	listener := NewHTTPListener(cfg).(*HTTPListener)

	// Start listener in background
	errChan := make(chan error, 1)
	go func() {
		err := listener.Start(func(agentID string, conn protocol.AgentConnection) {
			// Mock handler
		})
		if err != nil && err != http.ErrServerClosed {
			errChan <- err
		}
		close(errChan)
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Stop the listener
	err := listener.Stop()
	if err != nil {
		t.Errorf("Failed to stop listener: %v", err)
	}

	// Check for start errors
	select {
	case err := <-errChan:
		if err != nil {
			t.Errorf("Listener start error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Error("Listener did not stop within timeout")
	}
}

func TestHTTPListener_TLS(t *testing.T) {
	cfg := config.ListenerConfig{
		Name:        "test-tls",
		Type:        "https",
		BindAddress: "127.0.0.1",
		Port:        8443,
		URIPath:     "secure",
		UseSSL:      true,
		SSL: config.SSLConfig{
			Cert: "test-cert.pem",
			Key:  "test-key.pem",
		},
		Encryption: config.EncryptionConfig{
			Key: "0123456789abcdef0123456789abcdef",
		},
	}

	listener := NewHTTPListener(cfg).(*HTTPListener)

	// Verify TLS config was set
	if listener.server.TLSConfig == nil {
		t.Error("TLS config should be set when UseSSL is true")
	}

	if listener.server.TLSConfig.MinVersion != tls.VersionTLS12 {
		t.Errorf("Expected MinVersion TLS 1.2, got %v", listener.server.TLSConfig.MinVersion)
	}
}

func TestHTTPListener_ConcurrentRequests(t *testing.T) {
	listener := createTestListener()

	// Track handler calls
	var mu sync.Mutex
	handlerCalls := 0

	listener.agentHandler = func(agentID string, conn protocol.AgentConnection) {
		mu.Lock()
		handlerCalls++
		mu.Unlock()
	}

	// Create valid agent message
	agentMsg := protocol.AgentMessage{
		AgentID:  "concurrent-agent",
		Hostname: "test-host",
		Username: "test-user",
		OS:       "linux",
	}

	jsonData, _ := json.Marshal(agentMsg)
	encryptedData, _ := crypto.Encrypt(jsonData, listener.aesKey)
	encodedData := base64.StdEncoding.EncodeToString(encryptedData)

	// Send concurrent requests
	numRequests := 10
	var wg sync.WaitGroup
	wg.Add(numRequests)

	for i := 0; i < numRequests; i++ {
		go func(id int) {
			defer wg.Done()

			req := httptest.NewRequest("POST", "/test", bytes.NewBufferString(encodedData))
			w := httptest.NewRecorder()

			listener.handleAgent(w, req)

			resp := w.Result()
			if resp.StatusCode != http.StatusOK {
				t.Errorf("Request %d failed with status %d", id, resp.StatusCode)
			}
		}(i)
	}

	wg.Wait()

	// Verify all handlers were called
	if handlerCalls != numRequests {
		t.Errorf("Expected %d handler calls, got %d", numRequests, handlerCalls)
	}
}

func TestHTTPListener_FullIntegration(t *testing.T) {
	// Create listener with test server
	listener := createTestListener()

	// Set up test routes
	router := mux.NewRouter()
	listener.router = router

	// Track connections
	connections := make(map[string]protocol.AgentConnection)
	var connMu sync.Mutex

	// Set up agent handler
	agentHandler := func(agentID string, conn protocol.AgentConnection) {
		connMu.Lock()
		connections[agentID] = conn
		connMu.Unlock()
	}

	// Register routes
	router.HandleFunc("/"+listener.config.URIPath, func(w http.ResponseWriter, r *http.Request) {
		listener.agentHandler = agentHandler
		listener.handleAgent(w, r)
	}).Methods("POST")

	router.HandleFunc("/"+listener.config.URIPath+"/stage2", listener.handleStage2).Methods("GET")

	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("<html><body><h1>404 - Page Not Found</h1></body></html>"))
	})

	// Create test server
	server := httptest.NewServer(router)
	defer server.Close()

	// Test 1: Send agent beacon
	agentMsg := protocol.AgentMessage{
		AgentID:  "integration-agent",
		Hostname: "integration-host",
		Username: "integration-user",
		OS:       "linux",
		IP:       "10.0.0.1",
		TaskResults: []protocol.AgentTaskResult{
			{
				TaskID:   "task-123",
				Output:   "Integration test output",
				ExitCode: 0,
			},
		},
	}

	jsonData, _ := json.Marshal(agentMsg)
	encryptedData, _ := crypto.Encrypt(jsonData, listener.aesKey)
	encodedData := base64.StdEncoding.EncodeToString(encryptedData)

	resp, err := http.Post(server.URL+"/test", "application/octet-stream", bytes.NewBufferString(encodedData))
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Verify connection was established
	connMu.Lock()
	conn, exists := connections["integration-agent"]
	connMu.Unlock()

	if !exists {
		t.Error("Agent connection was not established")
	}
	if conn == nil {
		t.Error("Agent connection is nil")
	}

	// Test 2: Get stage2 payload
	resp2, err := http.Get(server.URL + "/test/stage2")
	if err != nil {
		t.Fatalf("Failed to get stage2: %v", err)
	}
	defer resp2.Body.Close()

	if resp2.StatusCode != http.StatusOK {
		t.Errorf("Expected stage2 status 200, got %d", resp2.StatusCode)
	}

	// Test 3: 404 handler
	resp3, err := http.Get(server.URL + "/nonexistent")
	if err != nil {
		t.Fatalf("Failed to get 404: %v", err)
	}
	defer resp3.Body.Close()

	if resp3.StatusCode != http.StatusNotFound {
		t.Errorf("Expected 404 status, got %d", resp3.StatusCode)
	}
}

// Benchmark tests
func BenchmarkHandleAgent(b *testing.B) {
	listener := createTestListener()

	listener.agentHandler = func(agentID string, conn protocol.AgentConnection) {
		// No-op handler for benchmark
	}

	// Prepare test data
	agentMsg := protocol.AgentMessage{
		AgentID:  "bench-agent",
		Hostname: "bench-host",
		Username: "bench-user",
		OS:       "linux",
	}

	jsonData, _ := json.Marshal(agentMsg)
	encryptedData, _ := crypto.Encrypt(jsonData, listener.aesKey)
	encodedData := base64.StdEncoding.EncodeToString(encryptedData)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/test", bytes.NewBufferString(encodedData))
		w := httptest.NewRecorder()
		listener.handleAgent(w, req)
	}
}

func BenchmarkEncryptDecrypt(b *testing.B) {
	listener := createTestListener()

	// Prepare test data
	agentMsg := protocol.AgentMessage{
		AgentID:  "bench-agent",
		Hostname: "bench-host",
		Username: "bench-user",
		OS:       "linux",
	}

	jsonData, _ := json.Marshal(agentMsg)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Encrypt
		encryptedData, err := crypto.Encrypt(jsonData, listener.aesKey)
		if err != nil {
			b.Fatal(err)
		}

		// Decrypt
		_, err = crypto.Decrypt(encryptedData, listener.aesKey)
		if err != nil {
			b.Fatal(err)
		}
	}
}
