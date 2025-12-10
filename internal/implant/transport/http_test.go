package transport

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/r74tech/virga/internal/implant/config"
	"github.com/r74tech/virga/internal/shared/protocol"
)

// Helper function to encrypt data for testing
func encryptTestData(data []byte, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	encrypted := gcm.Seal(nonce, nonce, data, nil)
	return encrypted, nil
}

func TestHTTPTransportJSONUnmarshal(t *testing.T) {
	// Create a test server that returns various responses
	tests := []struct {
		name        string
		response    string
		encrypted   bool
		expectError bool
		description string
	}{
		{
			name:        "ValidJSON",
			response:    `{"status":"ok","message":"test"}`,
			encrypted:   true,
			expectError: false,
			description: "Valid JSON should parse correctly",
		},
		{
			name:        "EmptyResponse",
			response:    "",
			encrypted:   false,
			expectError: false,
			description: "Empty response should return ok status",
		},
		{
			name:        "InvalidJSON",
			response:    `{"status":"ok", invalid json`,
			encrypted:   true,
			expectError: true,
			description: "Invalid JSON should return error",
		},
		{
			name:        "NonJSON",
			response:    "This is not JSON",
			encrypted:   true,
			expectError: true,
			description: "Non-JSON response should return error",
		},
	}

	// Create a fixed AES key for testing
	testKey := make([]byte, 32)
	for i := range testKey {
		testKey[i] = byte(i)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.response != "" {
					// Encrypt the response if needed
					if tt.encrypted {
						encrypted, err := encryptTestData([]byte(tt.response), testKey)
						if err != nil {
							t.Fatalf("Failed to encrypt test data: %v", err)
						}
						encoded := base64.StdEncoding.EncodeToString(encrypted)
						w.Write([]byte(encoded))
					} else {
						// For empty response test
						encoded := base64.StdEncoding.EncodeToString([]byte(tt.response))
						w.Write([]byte(encoded))
					}
				}
			}))
			defer server.Close()

			// Create transport with test config
			transport := &HTTPTransport{
				url: server.URL,
				config: &config.Config{
					C2Host:     "localhost",
					C2Port:     "443",
					C2Protocol: "https",
					C2Path:     "api/updates",
					AESKey:     testKey,
					UserAgent:  "test-agent",
				},
				client: &http.Client{},
			}

			// Create a test beacon
			beacon := &protocol.AgentMessage{
				AgentID:  "test-agent",
				Hostname: "test-host",
			}

			// Send beacon and check result
			resp, err := transport.SendBeacon(beacon)

			if tt.expectError && err == nil {
				t.Errorf("%s: expected error but got none", tt.description)
			}
			if !tt.expectError && err != nil {
				t.Errorf("%s: unexpected error: %v", tt.description, err)
			}
			if !tt.expectError && resp == nil {
				t.Errorf("%s: expected response but got nil", tt.description)
			}
		})
	}
}

// TestJSONValidation tests the JSON validation logic
func TestJSONValidation(t *testing.T) {
	tests := []struct {
		data     []byte
		expected bool
	}{
		{[]byte(`{"valid": "json"}`), true},
		{[]byte(`{"invalid": json}`), false},
		{[]byte(`not json at all`), false},
		{[]byte(``), false},
		{[]byte(`null`), true},
		{[]byte(`{}`), true},
		{[]byte(`[]`), true},
	}

	for _, tt := range tests {
		result := json.Valid(tt.data)
		if result != tt.expected {
			t.Errorf("json.Valid(%s) = %v, expected %v", string(tt.data), result, tt.expected)
		}
	}
}
