package client

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"sync"
	"time"

	"github.com/r74tech/virga/internal/cli/config"
	"github.com/r74tech/virga/internal/shared/logger"
	"github.com/r74tech/virga/internal/shared/protocol"
)

// APIClient handles communication with the server.
type APIClient struct {
	Config     *config.Config
	HttpClient *http.Client
	authToken  string
	debug      bool         // Debug mode flag
	mu         sync.RWMutex // protects debug flag
	authMu     sync.RWMutex // protects authToken
}

// SessionInfo holds session information obtained via the API.
type SessionInfo struct {
	ID           string                 `json:"id"`
	RemoteAddr   string                 `json:"remote_addr"`
	Hostname     string                 `json:"hostname"`
	Username     string                 `json:"username"`
	OS           string                 `json:"os"`
	LastActivity time.Time              `json:"last_activity"`
	Environment  map[string]interface{} `json:"environment"`
}

// NewAPIClient creates a new API client.
func NewAPIClient(cfg *config.Config) *APIClient {
	return &APIClient{
		Config: cfg,
		HttpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		debug: false, // Debug mode is disabled by default.
	}
}

// SetDebugMode enables/disables debug mode.
func (c *APIClient) SetDebugMode(debug bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.debug = debug
	if debug {
		logger.Debug("API Client debug mode enabled")
	}
}

// IsDebugEnabled returns whether debug mode is enabled
func (c *APIClient) IsDebugEnabled() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.debug
}

// isDebugEnabled returns whether debug mode is enabled (internal use)
func (c *APIClient) isDebugEnabled() bool {
	return c.IsDebugEnabled()
}

// GetAuthToken returns the current auth token
func (c *APIClient) GetAuthToken() string {
	c.authMu.RLock()
	defer c.authMu.RUnlock()
	return c.authToken
}

// getAuthToken returns the current auth token (internal use)
func (c *APIClient) getAuthToken() string {
	return c.GetAuthToken()
}

// SetAuthToken sets the auth token
func (c *APIClient) SetAuthToken(token string) {
	c.authMu.Lock()
	defer c.authMu.Unlock()
	c.authToken = token
}

// setAuthToken sets the auth token (internal use)
func (c *APIClient) setAuthToken(token string) {
	c.SetAuthToken(token)
}

// GetServerURL returns the server URL.
func (c *APIClient) GetServerURL() string {
	// Note: HTTP protocol is forced here.
	// In a real implementation, this should be configured appropriately based on the environment.
	scheme := "http"
	return fmt.Sprintf("%s://%s:%d", scheme, c.Config.Server.Host, c.Config.Server.Port)
}

// doRequest performs a common HTTP request with authentication and error handling
func (c *APIClient) doRequest(method, path string, body interface{}) ([]byte, error) {
	// Ensure authentication
	if err := c.Authenticate(); err != nil {
		return nil, fmt.Errorf("authentication failed: %w", err)
	}

	// Build URL
	url := fmt.Sprintf("%s%s", c.GetServerURL(), path)
	if c.isDebugEnabled() {
		logger.Debug("API %s %s", method, url)
	}

	// Prepare request body
	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonBody)
		if c.isDebugEnabled() {
			logger.Debug("Request body: %s", string(jsonBody))
		}
	}

	// Create request
	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set common headers
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.getAuthToken()))
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	// Send request
	resp, err := c.HttpClient.Do(req)
	if err != nil {
		if c.isDebugEnabled() {
			logger.Debug("Request failed: %v", err)
		}
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if c.isDebugEnabled() {
		logger.Debug("Response status: %d", resp.StatusCode)
		logger.Debug("Response body: %s", string(respBody))
	}

	// Check status code
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// doRequestWithContext performs a request with context support
func (c *APIClient) doRequestWithContext(ctx context.Context, method, path string, body interface{}) ([]byte, error) {
	// Ensure authentication
	if err := c.Authenticate(); err != nil {
		return nil, fmt.Errorf("authentication failed: %w", err)
	}

	// Build URL
	url := fmt.Sprintf("%s%s", c.GetServerURL(), path)

	// Prepare request body
	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonBody)
	}

	// Create request with context
	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set common headers
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.getAuthToken()))
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	// Send request
	resp, err := c.HttpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check status code
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// Authenticate performs authentication with the server.
func (c *APIClient) Authenticate() error {
	// Skip if authentication is not required or already authenticated.
	if c.getAuthToken() != "" {
		return nil
	}

	// For now, authentication is not implemented, so return a dummy token.
	c.setAuthToken("dummy-token")
	return nil
}

// GetSessions retrieves a list of active sessions.
func (c *APIClient) GetSessions() ([]SessionInfo, error) {
	if c.isDebugEnabled() {
		logger.Debug("Fetching sessions from server...")
	}

	// Make request
	respBody, err := c.doRequest("GET", "/api/sessions", nil)
	if err != nil {
		if c.isDebugEnabled() {
			logger.Debug("Failed to get sessions: %v", err)
		}
		// Return empty data to maintain minimal functionality even in case of server connection errors.
		return []SessionInfo{}, nil
	}

	// Parse response
	var result struct {
		Success  bool          `json:"success"`
		Sessions []SessionInfo `json:"sessions"`
		Message  string        `json:"message"`
	}

	if err := json.Unmarshal(respBody, &result); err != nil {
		if c.isDebugEnabled() {
			logger.Debug("Failed to parse API response: %v", err)
			logger.Debug("Response: %s", string(respBody))
		}
		return []SessionInfo{}, nil
	}

	if !result.Success {
		if c.isDebugEnabled() {
			logger.Debug("API returned error: %s", result.Message)
		}
		return []SessionInfo{}, nil
	}

	if c.isDebugEnabled() {
		logger.Debug("Retrieved %d sessions from server", len(result.Sessions))
	}

	return result.Sessions, nil
}

// SetSessionInteractive sets the interactive mode for a session.
func (c *APIClient) SetSessionInteractive(sessionID string, interactive bool) error {
	if c.isDebugEnabled() {
		logger.Debug("Setting session %s interactive mode to %v", sessionID, interactive)
	}

	// Request data
	reqData := map[string]interface{}{
		"interactive": interactive,
	}

	// Make request
	_, err := c.doRequest("POST", fmt.Sprintf("/api/sessions/%s/interactive", sessionID), reqData)
	if err != nil {
		return err
	}

	if c.isDebugEnabled() {
		logger.Debug("Successfully set session %s interactive mode to %v", sessionID, interactive)
	}

	return nil
}

// SendSessionCommand sends a command to a session.
func (c *APIClient) SendSessionCommand(sessionID, cmdType string, payload map[string]interface{}) (string, error) {
	if c.isDebugEnabled() {
		logger.Debug("Sending command to session %s (type: %s)", sessionID, cmdType)
	}

	// Request data
	reqData := map[string]interface{}{
		"type":    cmdType,
		"payload": payload,
	}

	// Make request
	respBody, err := c.doRequest("POST", fmt.Sprintf("/api/sessions/%s/tasks", sessionID), reqData)
	if err != nil {
		return "", err
	}

	// Parse response
	var result struct {
		Success bool   `json:"success"`
		TaskID  string `json:"task_id"`
		Message string `json:"message"`
	}

	if err := json.Unmarshal(respBody, &result); err != nil {
		if c.isDebugEnabled() {
			logger.Debug("Failed to parse API response: %v", err)
			logger.Debug("Response: %s", string(respBody))
		}
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if !result.Success {
		if c.isDebugEnabled() {
			logger.Debug("API returned error: %s", result.Message)
		}
		return "", fmt.Errorf("API returned error: %s", result.Message)
	}

	if c.isDebugEnabled() {
		logger.Debug("Command sent, task ID: %s", result.TaskID)
	}

	return result.TaskID, nil
}

// GetTaskResult retrieves the result of a task execution.
func (c *APIClient) GetTaskResult(sessionID, taskID string) (*protocol.TaskResult, error) {
	if c.isDebugEnabled() {
		logger.Debug("Fetching task result for session %s, task %s", sessionID, taskID)
	}

	// Special handling for task results - we need to check status codes
	// Ensure authentication
	if err := c.Authenticate(); err != nil {
		return nil, fmt.Errorf("authentication failed: %w", err)
	}

	// Build URL
	url := fmt.Sprintf("%s/api/sessions/%s/tasks/%s", c.GetServerURL(), sessionID, taskID)

	// Create request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.getAuthToken()))

	// Send request
	resp, err := c.HttpClient.Do(req)
	if err != nil {
		if c.isDebugEnabled() {
			logger.Debug("API request failed: %v", err)
		}
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode == http.StatusNotFound {
		// If the task result is not yet available
		if c.isDebugEnabled() {
			logger.Debug("Task result not found yet for session %s, task %s", sessionID, taskID)
		}
		return nil, nil
	}

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		if c.isDebugEnabled() {
			logger.Debug("API error (status %d): %s", resp.StatusCode, string(respBody))
		}
		return nil, fmt.Errorf("API returned error (status %d): %s", resp.StatusCode, string(respBody))
	}

	// Parse response
	var result struct {
		Success    bool      `json:"success"`
		TaskID     string    `json:"task_id"`
		Output     string    `json:"output"`
		ExitCode   int       `json:"exit_code"`
		Error      string    `json:"error"`
		Time       time.Time `json:"time"`
		IsComplete bool      `json:"is_complete"`
	}

	if err := json.Unmarshal(respBody, &result); err != nil {
		if c.isDebugEnabled() {
			logger.Debug("Failed to parse API response: %v", err)
			logger.Debug("Response: %s", string(respBody))
		}
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if !result.Success {
		if c.isDebugEnabled() {
			logger.Debug("API returned error for task result")
		}
		return nil, fmt.Errorf("failed to get task result: API returned unsuccessful status")
	}

	// If task is not yet complete, return nil to continue polling
	if !result.IsComplete {
		if c.isDebugEnabled() {
			logger.Debug("Task %s is still running (is_complete=false)", taskID)
		}
		return nil, nil
	}

	taskResult := &protocol.TaskResult{
		TaskID:   result.TaskID,
		Output:   result.Output,
		ExitCode: result.ExitCode,
		Error:    result.Error,
		Time:     result.Time,
	}

	if c.isDebugEnabled() {
		logger.Debug("Retrieved task result for task %s", taskID)
	}

	return taskResult, nil
}

// GetTaskResults retrieves all task results for a session.
func (c *APIClient) GetTaskResults(sessionID string) ([]protocol.TaskResult, error) {
	if c.isDebugEnabled() {
		logger.Debug("Fetching all task results for session %s", sessionID)
	}

	// Make request
	respBody, err := c.doRequest("GET", fmt.Sprintf("/api/sessions/%s/tasks", sessionID), nil)
	if err != nil {
		return nil, err
	}

	// Parse response
	var result struct {
		Success bool                  `json:"success"`
		Results []protocol.TaskResult `json:"results"`
	}

	if err := json.Unmarshal(respBody, &result); err != nil {
		if c.isDebugEnabled() {
			logger.Debug("Failed to parse API response: %v", err)
			logger.Debug("Response: %s", string(respBody))
		}
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if !result.Success {
		if c.isDebugEnabled() {
			logger.Debug("API returned error for task results")
		}
		return nil, fmt.Errorf("API returned error for task results")
	}

	if c.isDebugEnabled() {
		logger.Debug("Retrieved %d task results for session %s", len(result.Results), sessionID)
	}

	return result.Results, nil
}

// GeneratePayload generates a payload.
func (c *APIClient) GeneratePayload(payloadType, listenerName string) ([]byte, string, error) {
	switch payloadType {
	case "powershell", "exe", "dll":
		// Supported types
	default:
		return nil, "", fmt.Errorf("unsupported payload type: %s", payloadType)
	}

	// Request data
	reqData := map[string]string{
		"listener": listenerName,
	}

	// Make request
	respBody, err := c.doRequest("POST", fmt.Sprintf("/api/generate/%s", payloadType), reqData)
	if err != nil {
		return nil, "", err
	}

	// Parse response
	var result struct {
		Success  bool   `json:"success"`
		Payload  string `json:"payload"`
		Filename string `json:"filename"`
		Message  string `json:"message"`
	}

	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, "", fmt.Errorf("failed to parse response: %w", err)
	}

	if !result.Success {
		return nil, "", fmt.Errorf("failed to generate payload: %s", result.Message)
	}

	payload, err := base64.StdEncoding.DecodeString(result.Payload)
	if err != nil {
		return nil, "", fmt.Errorf("failed to decode payload: %w", err)
	}

	return payload, result.Filename, nil
}

// GetListeners retrieves a list of listeners.
func (c *APIClient) GetListeners() ([]map[string]interface{}, error) {
	// Return dummy listener information (in a real implementation, this data would be fetched from the server).
	listeners := []map[string]interface{}{
		{
			"name":         "default-http",
			"type":         "http",
			"bind_address": "0.0.0.0",
			"port":         8080,
			"status":       "running",
		},
	}

	return listeners, nil
}

// UploadExtension uploads an extension to the server for a specific session
func (c *APIClient) UploadExtension(sessionID string, manifestJSON string, extensionData []byte) error {
	if c.isDebugEnabled() {
		logger.Debug("Uploading extension to session %s", sessionID)
	}

	// Create multipart form data
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add manifest field
	manifestField, err := writer.CreateFormField("manifest")
	if err != nil {
		return fmt.Errorf("error creating manifest field: %v", err)
	}
	if _, err := manifestField.Write([]byte(manifestJSON)); err != nil {
		return fmt.Errorf("error writing manifest data: %v", err)
	}

	// Add data field
	dataField, err := writer.CreateFormFile("data", "extension.bin")
	if err != nil {
		return fmt.Errorf("error creating data field: %v", err)
	}
	if _, err := dataField.Write(extensionData); err != nil {
		return fmt.Errorf("error writing extension data: %v", err)
	}

	// Close the writer to finalize the form data
	if err := writer.Close(); err != nil {
		return fmt.Errorf("error closing multipart writer: %v", err)
	}

	// Create request
	url := fmt.Sprintf("%s/api/sessions/%s/extensions", c.GetServerURL(), sessionID)
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return fmt.Errorf("error creating request: %v", err)
	}

	// Set headers
	req.Header.Set("Content-Type", writer.FormDataContentType())
	if token := c.getAuthToken(); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	// Send request
	resp, err := c.HttpClient.Do(req)
	if err != nil {
		return fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()

	// Check response
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("server returned error: %s", string(bodyBytes))
	}

	if c.isDebugEnabled() {
		logger.Debug("Extension uploaded successfully")
	}

	return nil
}

// ListExtensions gets the list of extensions for a session
func (c *APIClient) ListExtensions(sessionID string) ([]map[string]interface{}, error) {
	if c.isDebugEnabled() {
		logger.Debug("Getting extensions for session %s", sessionID)
	}

	url := fmt.Sprintf("%s/api/sessions/%s/extensions", c.GetServerURL(), sessionID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}

	// Set headers
	if token := c.getAuthToken(); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	// Send request
	resp, err := c.HttpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()

	// Parse response
	var response struct {
		Success    bool                     `json:"success"`
		Extensions []map[string]interface{} `json:"extensions"`
		Message    string                   `json:"message"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	if !response.Success {
		return nil, fmt.Errorf("server error: %s", response.Message)
	}

	return response.Extensions, nil
}

// ExecuteExtension executes an extension
func (c *APIClient) ExecuteExtension(sessionID, extensionName, command string, arguments map[string]interface{}) (string, error) {
	if c.isDebugEnabled() {
		logger.Debug("Executing extension %s on session %s", extensionName, sessionID)
	}

	// Create request body
	reqBody := map[string]interface{}{
		"command":   command,
		"arguments": arguments,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("error encoding request: %v", err)
	}

	// Create request
	url := fmt.Sprintf("%s/api/sessions/%s/extensions/%s/execute", c.GetServerURL(), sessionID, extensionName)
	req, err := http.NewRequest("POST", url, bytes.NewReader(jsonBody))
	if err != nil {
		return "", fmt.Errorf("error creating request: %v", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	if token := c.getAuthToken(); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	// Send request
	resp, err := c.HttpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()

	// Parse response
	var response struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
		TaskID  string `json:"task_id"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", fmt.Errorf("error parsing response: %v", err)
	}

	if !response.Success {
		return "", fmt.Errorf("server error: %s", response.Message)
	}

	// For now, we return a success message. In the future, we might want to return the task ID
	// and let the caller poll for results
	return "Extension execution task queued", nil
}

// DeleteExtension deletes an extension from a session
func (c *APIClient) DeleteExtension(sessionID, extensionName string) error {
	if c.isDebugEnabled() {
		logger.Debug("Deleting extension %s from session %s", extensionName, sessionID)
	}

	// Create request
	url := fmt.Sprintf("%s/api/sessions/%s/extensions/%s", c.GetServerURL(), sessionID, extensionName)
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("error creating request: %v", err)
	}

	// Set headers
	if token := c.getAuthToken(); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	// Send request
	resp, err := c.HttpClient.Do(req)
	if err != nil {
		return fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()

	// Check response
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("server returned error: %s", string(bodyBytes))
	}

	if c.isDebugEnabled() {
		logger.Debug("Extension deleted successfully")
	}

	return nil
}
