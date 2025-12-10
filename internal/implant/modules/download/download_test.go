package download

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// parseDownloadOutput parses the download module output format
// which is: size:X\nname:Y\ncontent:Z
func parseDownloadOutput(output string) (content string, err error) {
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "content:") {
			return strings.TrimPrefix(line, "content:"), nil
		}
	}
	return "", fmt.Errorf("content not found in output")
}

func TestDownloadModule_Name(t *testing.T) {
	module := &DownloadModule{}
	if name := module.Name(); name != "download" {
		t.Errorf("Expected name 'download', got '%s'", name)
	}
}

func TestDownloadModule_Execute(t *testing.T) {
	module := &DownloadModule{}

	// Create a temporary directory for tests
	tempDir, err := os.MkdirTemp("", "download_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tests := []struct {
		name         string
		args         []string
		expectError  bool
		expectCode   int
		setupFunc    func() string      // returns file path
		verifyFunc   func(string) error // verify output
		errorMessage string
	}{
		{
			name:         "No arguments",
			args:         []string{},
			expectError:  true,
			expectCode:   1,
			errorMessage: "download requires remote_path",
		},
		{
			name: "Download text file",
			setupFunc: func() string {
				filePath := filepath.Join(tempDir, "test.txt")
				content := "Hello, Download Test!"
				if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
					t.Fatalf("Failed to create test file: %v", err)
				}
				return filePath
			},
			expectError: false,
			expectCode:  0,
			verifyFunc: func(output string) error {
				// Debug: print the actual output
				t.Logf("Actual output: %q", output)
				// Parse the output format
				content, err := parseDownloadOutput(output)
				if err != nil {
					return err
				}
				decoded, err := base64.StdEncoding.DecodeString(content)
				if err != nil {
					return err
				}
				expected := "Hello, Download Test!"
				if string(decoded) != expected {
					t.Errorf("Expected content '%s', got '%s'", expected, string(decoded))
				}
				return nil
			},
		},
		{
			name: "Download binary file",
			setupFunc: func() string {
				filePath := filepath.Join(tempDir, "binary.bin")
				content := []byte{0x00, 0x01, 0x02, 0x03, 0xFF, 0xFE, 0xFD}
				if err := os.WriteFile(filePath, content, 0644); err != nil {
					t.Fatalf("Failed to create binary file: %v", err)
				}
				return filePath
			},
			expectError: false,
			expectCode:  0,
			verifyFunc: func(output string) error {
				// Parse the output format
				content, err := parseDownloadOutput(output)
				if err != nil {
					return err
				}
				decoded, err := base64.StdEncoding.DecodeString(content)
				if err != nil {
					return err
				}
				expected := []byte{0x00, 0x01, 0x02, 0x03, 0xFF, 0xFE, 0xFD}
				if len(decoded) != len(expected) {
					t.Errorf("Expected %d bytes, got %d", len(expected), len(decoded))
				}
				for i, b := range decoded {
					if b != expected[i] {
						t.Errorf("Byte %d: expected %02x, got %02x", i, expected[i], b)
					}
				}
				return nil
			},
		},
		{
			name: "Download empty file",
			setupFunc: func() string {
				filePath := filepath.Join(tempDir, "empty.txt")
				if err := os.WriteFile(filePath, []byte{}, 0644); err != nil {
					t.Fatalf("Failed to create empty file: %v", err)
				}
				return filePath
			},
			expectError: false,
			expectCode:  0,
			verifyFunc: func(output string) error {
				// Parse the output format
				content, err := parseDownloadOutput(output)
				if err != nil {
					return err
				}
				decoded, err := base64.StdEncoding.DecodeString(content)
				if err != nil {
					return err
				}
				if len(decoded) != 0 {
					t.Errorf("Expected empty content, got %d bytes", len(decoded))
				}
				return nil
			},
		},
		{
			name: "Download non-existent file",
			setupFunc: func() string {
				return filepath.Join(tempDir, "non_existent_file.txt")
			},
			expectError:  true,
			expectCode:   1,
			errorMessage: "no such file",
		},
		{
			name: "Download from protected directory",
			setupFunc: func() string {
				// Try to read from a protected file
				if os.Getuid() == 0 {
					t.Skip("Running as root, skipping permission test")
				}
				return "/etc/shadow" // Usually not readable by non-root users
			},
			expectError:  true,
			expectCode:   1,
			errorMessage: "", // Can be either "permission denied" or "no such file",
		},
		{
			name: "Download file with spaces in name",
			setupFunc: func() string {
				filePath := filepath.Join(tempDir, "file with spaces.txt")
				content := "Content with spaces in filename"
				if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
					t.Fatalf("Failed to create file with spaces: %v", err)
				}
				return filePath
			},
			expectError: false,
			expectCode:  0,
			verifyFunc: func(output string) error {
				// Parse the output format
				content, err := parseDownloadOutput(output)
				if err != nil {
					return err
				}
				decoded, err := base64.StdEncoding.DecodeString(content)
				if err != nil {
					return err
				}
				expected := "Content with spaces in filename"
				if string(decoded) != expected {
					t.Errorf("Expected '%s', got '%s'", expected, string(decoded))
				}
				return nil
			},
		},
		{
			name: "Download large file",
			setupFunc: func() string {
				filePath := filepath.Join(tempDir, "large.bin")
				// Create 1MB file
				content := make([]byte, 1024*1024)
				for i := range content {
					content[i] = byte(i % 256)
				}
				if err := os.WriteFile(filePath, content, 0644); err != nil {
					t.Fatalf("Failed to create large file: %v", err)
				}
				return filePath
			},
			expectError: false,
			expectCode:  0,
			verifyFunc: func(output string) error {
				// Parse the output format
				content, err := parseDownloadOutput(output)
				if err != nil {
					return err
				}
				decoded, err := base64.StdEncoding.DecodeString(content)
				if err != nil {
					return err
				}
				if len(decoded) != 1024*1024 {
					t.Errorf("Expected 1MB, got %d bytes", len(decoded))
				}
				// Verify pattern
				for i := 0; i < len(decoded) && i < 1000; i++ {
					if decoded[i] != byte(i%256) {
						t.Errorf("Byte %d: expected %02x, got %02x", i, byte(i%256), decoded[i])
						break
					}
				}
				return nil
			},
		},
		{
			name: "Download directory (should fail)",
			setupFunc: func() string {
				dirPath := filepath.Join(tempDir, "testdir")
				if err := os.Mkdir(dirPath, 0755); err != nil {
					t.Fatalf("Failed to create directory: %v", err)
				}
				return dirPath
			},
			expectError:  true,
			expectCode:   1,
			errorMessage: "is a directory",
		},
		{
			name: "Download with relative path",
			setupFunc: func() string {
				// Create a file in current directory
				filePath := filepath.Join(tempDir, "relative_test.txt")
				content := "Relative path test"
				if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
					t.Fatalf("Failed to create file: %v", err)
				}
				// Change to temp directory
				originalDir, _ := os.Getwd()
				os.Chdir(tempDir)
				t.Cleanup(func() { os.Chdir(originalDir) })
				return "relative_test.txt"
			},
			expectError: false,
			expectCode:  0,
			verifyFunc: func(output string) error {
				// Parse the output format
				content, err := parseDownloadOutput(output)
				if err != nil {
					return err
				}
				decoded, err := base64.StdEncoding.DecodeString(content)
				if err != nil {
					return err
				}
				expected := "Relative path test"
				if string(decoded) != expected {
					t.Errorf("Expected '%s', got '%s'", expected, string(decoded))
				}
				return nil
			},
		},
		{
			name: "Download with UTF-8 content",
			setupFunc: func() string {
				filePath := filepath.Join(tempDir, "utf8.txt")
				content := "Hello UTF-8! 🌍 Здравствуй мир!"
				if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
					t.Fatalf("Failed to create UTF-8 file: %v", err)
				}
				return filePath
			},
			expectError: false,
			expectCode:  0,
			verifyFunc: func(output string) error {
				// Parse the output format
				content, err := parseDownloadOutput(output)
				if err != nil {
					return err
				}
				decoded, err := base64.StdEncoding.DecodeString(content)
				if err != nil {
					return err
				}
				expected := "Hello UTF-8! 🌍 Здравствуй мир!"
				if string(decoded) != expected {
					t.Errorf("Expected '%s', got '%s'", expected, string(decoded))
				}
				return nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var args []string

			if tt.setupFunc != nil {
				filePath := tt.setupFunc()
				args = []string{filePath}
			} else {
				args = tt.args
			}

			output, exitCode, err := module.Execute(args)

			// Check error expectation
			if tt.expectError {
				if err == nil && exitCode == 0 {
					t.Error("Expected error but got none")
				}
				if tt.errorMessage != "" && !strings.Contains(strings.ToLower(output), strings.ToLower(tt.errorMessage)) &&
					(err == nil || !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(tt.errorMessage))) {
					t.Errorf("Expected error message containing '%s', got output: '%s', error: %v", tt.errorMessage, output, err)
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
			if tt.verifyFunc != nil && !tt.expectError {
				if err := tt.verifyFunc(output); err != nil {
					t.Errorf("Verification failed: %v", err)
				}
			}
		})
	}
}

func TestDownloadModule_SymbolicLink(t *testing.T) {
	// Skip on Windows as it handles symlinks differently
	if os.Getenv("GOOS") == "windows" {
		t.Skip("Skipping symlink test on Windows")
	}

	module := &DownloadModule{}
	tempDir, err := os.MkdirTemp("", "download_symlink_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create target file
	targetPath := filepath.Join(tempDir, "target.txt")
	targetContent := "Target file content"
	if err := os.WriteFile(targetPath, []byte(targetContent), 0644); err != nil {
		t.Fatalf("Failed to create target file: %v", err)
	}

	// Create symlink
	linkPath := filepath.Join(tempDir, "link.txt")
	if err := os.Symlink(targetPath, linkPath); err != nil {
		t.Fatalf("Failed to create symlink: %v", err)
	}

	// Download through symlink
	output, exitCode, err := module.Execute([]string{linkPath})
	if err != nil {
		t.Errorf("Failed to download through symlink: %v", err)
	}
	if exitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", exitCode)
	}

	// Verify content
	content, err := parseDownloadOutput(output)
	if err != nil {
		t.Fatalf("Failed to parse output: %v", err)
	}
	decoded, err := base64.StdEncoding.DecodeString(content)
	if err != nil {
		t.Fatalf("Failed to decode output: %v", err)
	}
	if string(decoded) != targetContent {
		t.Errorf("Expected '%s', got '%s'", targetContent, string(decoded))
	}
}

func BenchmarkDownloadModule_SmallFile(b *testing.B) {
	module := &DownloadModule{}
	tempDir, _ := os.MkdirTemp("", "download_bench")
	defer os.RemoveAll(tempDir)

	// Create small test file
	filePath := filepath.Join(tempDir, "small.txt")
	content := "Small file content for benchmarking"
	os.WriteFile(filePath, []byte(content), 0644)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		module.Execute([]string{filePath})
	}
}

func BenchmarkDownloadModule_LargeFile(b *testing.B) {
	module := &DownloadModule{}
	tempDir, _ := os.MkdirTemp("", "download_bench")
	defer os.RemoveAll(tempDir)

	// Create 100KB file
	filePath := filepath.Join(tempDir, "large.bin")
	content := make([]byte, 100*1024)
	for i := range content {
		content[i] = byte(i % 256)
	}
	os.WriteFile(filePath, content, 0644)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		module.Execute([]string{filePath})
	}
}

func BenchmarkBase64Encoding(b *testing.B) {
	// Benchmark just the base64 encoding part
	data := make([]byte, 10*1024) // 10KB
	for i := range data {
		data[i] = byte(i % 256)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = base64.StdEncoding.EncodeToString(data)
	}
}
