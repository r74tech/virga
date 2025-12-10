package upload

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUploadModule_Name(t *testing.T) {
	module := &UploadModule{}
	if name := module.Name(); name != "upload" {
		t.Errorf("Expected name 'upload', got '%s'", name)
	}
}

func TestUploadModule_Execute(t *testing.T) {
	module := &UploadModule{}

	// Create a temporary directory for tests
	tempDir, err := os.MkdirTemp("", "upload_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tests := []struct {
		name         string
		args         []string
		expectError  bool
		expectCode   int
		setupFunc    func() (string, string) // returns (remotePath, encodedContent)
		verifyFunc   func(string) error
		errorMessage string
	}{
		{
			name:         "No arguments",
			args:         []string{},
			expectError:  true,
			expectCode:   1,
			errorMessage: "upload requires remote_path and content",
		},
		{
			name:         "Only one argument",
			args:         []string{"/tmp/test.txt"},
			expectError:  true,
			expectCode:   1,
			errorMessage: "upload requires remote_path and content",
		},
		{
			name: "Valid upload",
			setupFunc: func() (string, string) {
				remotePath := filepath.Join(tempDir, "test.txt")
				content := "Hello, World!"
				encoded := base64.StdEncoding.EncodeToString([]byte(content))
				return remotePath, encoded
			},
			expectError: false,
			expectCode:  0,
			verifyFunc: func(path string) error {
				data, err := os.ReadFile(path)
				if err != nil {
					return err
				}
				if string(data) != "Hello, World!" {
					t.Errorf("Expected content 'Hello, World!', got '%s'", string(data))
				}
				return nil
			},
		},
		{
			name: "Upload with subdirectory creation",
			setupFunc: func() (string, string) {
				remotePath := filepath.Join(tempDir, "subdir", "test.txt")
				content := "Subdirectory test"
				encoded := base64.StdEncoding.EncodeToString([]byte(content))
				return remotePath, encoded
			},
			expectError: false,
			expectCode:  0,
			verifyFunc: func(path string) error {
				data, err := os.ReadFile(path)
				if err != nil {
					return err
				}
				if string(data) != "Subdirectory test" {
					t.Errorf("Expected content 'Subdirectory test', got '%s'", string(data))
				}
				return nil
			},
		},
		{
			name: "Binary content upload",
			setupFunc: func() (string, string) {
				remotePath := filepath.Join(tempDir, "binary.bin")
				content := []byte{0x00, 0x01, 0x02, 0x03, 0xFF, 0xFE, 0xFD}
				encoded := base64.StdEncoding.EncodeToString(content)
				return remotePath, encoded
			},
			expectError: false,
			expectCode:  0,
			verifyFunc: func(path string) error {
				data, err := os.ReadFile(path)
				if err != nil {
					return err
				}
				expected := []byte{0x00, 0x01, 0x02, 0x03, 0xFF, 0xFE, 0xFD}
				if len(data) != len(expected) {
					t.Errorf("Expected %d bytes, got %d", len(expected), len(data))
				}
				for i, b := range data {
					if b != expected[i] {
						t.Errorf("Byte %d: expected %02x, got %02x", i, expected[i], b)
					}
				}
				return nil
			},
		},
		{
			name: "Invalid base64 content",
			setupFunc: func() (string, string) {
				remotePath := filepath.Join(tempDir, "invalid.txt")
				invalidBase64 := "This is not valid base64!!!"
				return remotePath, invalidBase64
			},
			expectError:  true,
			expectCode:   1,
			errorMessage: "failed to decode content",
		},
		{
			name: "Empty content",
			setupFunc: func() (string, string) {
				remotePath := filepath.Join(tempDir, "empty.txt")
				encoded := base64.StdEncoding.EncodeToString([]byte(""))
				return remotePath, encoded
			},
			expectError: false,
			expectCode:  0,
			verifyFunc: func(path string) error {
				info, err := os.Stat(path)
				if err != nil {
					return err
				}
				if info.Size() != 0 {
					t.Errorf("Expected empty file, got size %d", info.Size())
				}
				return nil
			},
		},
		{
			name: "Overwrite existing file",
			setupFunc: func() (string, string) {
				remotePath := filepath.Join(tempDir, "existing.txt")
				// Create existing file
				os.WriteFile(remotePath, []byte("Old content"), 0o644)

				newContent := "New content"
				encoded := base64.StdEncoding.EncodeToString([]byte(newContent))
				return remotePath, encoded
			},
			expectError: false,
			expectCode:  0,
			verifyFunc: func(path string) error {
				data, err := os.ReadFile(path)
				if err != nil {
					return err
				}
				if string(data) != "New content" {
					t.Errorf("Expected 'New content', got '%s'", string(data))
				}
				return nil
			},
		},
		{
			name: "Upload to protected directory",
			setupFunc: func() (string, string) {
				// Try to write to a system directory (will fail on most systems)
				remotePath := "/root/test.txt"
				if os.Getuid() == 0 {
					// If running as root, skip this test
					t.Skip("Running as root, skipping permission test")
				}
				content := "Should fail"
				encoded := base64.StdEncoding.EncodeToString([]byte(content))
				return remotePath, encoded
			},
			expectError:  true,
			expectCode:   1,
			errorMessage: "", // Can be "permission denied" or "read-only file system",
		},
		{
			name: "Large file upload",
			setupFunc: func() (string, string) {
				remotePath := filepath.Join(tempDir, "large.txt")
				// Create 1MB of content
				content := make([]byte, 1024*1024)
				for i := range content {
					content[i] = byte(i % 256)
				}
				encoded := base64.StdEncoding.EncodeToString(content)
				return remotePath, encoded
			},
			expectError: false,
			expectCode:  0,
			verifyFunc: func(path string) error {
				info, err := os.Stat(path)
				if err != nil {
					return err
				}
				if info.Size() != 1024*1024 {
					t.Errorf("Expected 1MB file, got %d bytes", info.Size())
				}
				return nil
			},
		},
		{
			name: "Path with spaces",
			setupFunc: func() (string, string) {
				remotePath := filepath.Join(tempDir, "file with spaces.txt")
				content := "Content with spaces in filename"
				encoded := base64.StdEncoding.EncodeToString([]byte(content))
				return remotePath, encoded
			},
			expectError: false,
			expectCode:  0,
			verifyFunc: func(path string) error {
				data, err := os.ReadFile(path)
				if err != nil {
					return err
				}
				if string(data) != "Content with spaces in filename" {
					t.Errorf("Unexpected content: %s", string(data))
				}
				return nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var args []string
			var remotePath string

			if tt.setupFunc != nil {
				remotePath, encodedContent := tt.setupFunc()
				args = []string{remotePath, encodedContent}
			} else {
				args = tt.args
			}

			output, exitCode, err := module.Execute(args)

			// Check error expectation
			if tt.expectError {
				if err == nil && exitCode == 0 {
					t.Error("Expected error but got none")
				}
				if tt.errorMessage != "" && !strings.Contains(strings.ToLower(output), strings.ToLower(tt.errorMessage)) {
					t.Errorf("Expected error message containing '%s', got '%s'", tt.errorMessage, output)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}

			// Check exit code
			if exitCode != tt.expectCode {
				t.Errorf("Expected exit code %d, got %d", tt.expectCode, exitCode)
			}

			// Run verification function if provided
			if tt.verifyFunc != nil && remotePath != "" {
				if err := tt.verifyFunc(remotePath); err != nil {
					t.Errorf("Verification failed: %v", err)
				}
			}

			// Check success message
			if !tt.expectError && !strings.Contains(output, "uploaded successfully") {
				t.Errorf("Expected success message in output, got: %s", output)
			}
		})
	}
}

func TestUploadModule_FilePermissions(t *testing.T) {
	// Skip on Windows as it handles permissions differently
	if os.Getenv("GOOS") == "windows" {
		t.Skip("Skipping permission test on Windows")
	}

	module := &UploadModule{}
	tempDir, err := os.MkdirTemp("", "upload_perm_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	remotePath := filepath.Join(tempDir, "perm_test.txt")
	content := "Permission test"
	encoded := base64.StdEncoding.EncodeToString([]byte(content))

	_, _, err = module.Execute([]string{remotePath, encoded})
	if err != nil {
		t.Fatalf("Failed to upload file: %v", err)
	}

	// Check file permissions
	info, err := os.Stat(remotePath)
	if err != nil {
		t.Fatalf("Failed to stat file: %v", err)
	}

	// Should be readable and writable by owner (0644)
	perm := info.Mode().Perm()
	if perm != 0o644 {
		t.Errorf("Expected permissions 0644, got %o", perm)
	}
}

func BenchmarkUploadModule_SmallFile(b *testing.B) {
	module := &UploadModule{}
	tempDir, _ := os.MkdirTemp("", "upload_bench")
	defer os.RemoveAll(tempDir)

	content := "Small file content"
	encoded := base64.StdEncoding.EncodeToString([]byte(content))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		remotePath := filepath.Join(tempDir, fmt.Sprintf("file_%d.txt", i))
		module.Execute([]string{remotePath, encoded})
	}
}

func BenchmarkUploadModule_LargeFile(b *testing.B) {
	module := &UploadModule{}
	tempDir, _ := os.MkdirTemp("", "upload_bench")
	defer os.RemoveAll(tempDir)

	// 100KB file
	content := make([]byte, 100*1024)
	for i := range content {
		content[i] = byte(i % 256)
	}
	encoded := base64.StdEncoding.EncodeToString(content)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		remotePath := filepath.Join(tempDir, fmt.Sprintf("file_%d.bin", i))
		module.Execute([]string{remotePath, encoded})
	}
}
