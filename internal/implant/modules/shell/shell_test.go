package shell

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestShellModule_Name(t *testing.T) {
	module := &ShellModule{}
	if name := module.Name(); name != "shell" {
		t.Errorf("Expected name 'shell', got '%s'", name)
	}
}

func TestShellModule_Execute(t *testing.T) {
	module := &ShellModule{}

	tests := []struct {
		name          string
		args          []string
		expectError   bool
		checkOutput   func(string) bool
		skipOnWindows bool
	}{
		{
			name:        "No command",
			args:        []string{},
			expectError: true,
		},
		{
			name: "Echo command",
			args: []string{"echo hello"},
			checkOutput: func(output string) bool {
				return strings.TrimSpace(output) == "hello"
			},
		},
		{
			name: "Echo with quotes",
			args: []string{"echo \"hello world\""},
			checkOutput: func(output string) bool {
				return strings.TrimSpace(output) == "hello world"
			},
		},
		{
			name:          "Echo with single quotes",
			args:          []string{"echo 'hello world'"},
			skipOnWindows: true, // Windows doesn't handle single quotes the same way
			checkOutput: func(output string) bool {
				return strings.TrimSpace(output) == "hello world"
			},
		},
		{
			name:        "Invalid command",
			args:        []string{"thisisnotavalidcommand12345"},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skipOnWindows && runtime.GOOS == "windows" {
				t.Skip("Skipping test on Windows")
			}

			output, exitCode, err := module.Execute(tt.args)

			if tt.expectError {
				if exitCode == 0 {
					t.Error("Expected non-zero exit code for error case")
				}
				if err == nil && output == "" {
					t.Error("Expected error or error output")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if exitCode != 0 {
					t.Errorf("Expected exit code 0, got %d", exitCode)
				}
				if tt.checkOutput != nil && !tt.checkOutput(output) {
					t.Errorf("Output check failed. Got: %s", output)
				}
			}
		})
	}
}

func TestParseCommandLine(t *testing.T) {
	tests := []struct {
		name     string
		command  string
		expected []string
	}{
		{
			name:     "Simple command",
			command:  "echo hello",
			expected: []string{"echo", "hello"},
		},
		{
			name:     "Double quoted string",
			command:  `echo "hello world"`,
			expected: []string{"echo", "hello world"},
		},
		{
			name:     "Single quoted string",
			command:  `echo 'hello world'`,
			expected: []string{"echo", "hello world"},
		},
		{
			name:     "Mixed quotes",
			command:  `echo "hello 'world'"`,
			expected: []string{"echo", "hello 'world'"},
		},
		{
			name:     "Escaped characters",
			command:  `echo hello\ world`,
			expected: []string{"echo", "hello world"},
		},
		{
			name:     "Multiple spaces",
			command:  "echo    hello    world",
			expected: []string{"echo", "hello", "world"},
		},
		{
			name:     "Empty quotes",
			command:  `echo "" hello`,
			expected: []string{"echo", "", "hello"},
		},
		{
			name:     "Command with path",
			command:  "/usr/bin/echo hello",
			expected: []string{"/usr/bin/echo", "hello"},
		},
		{
			name:     "Complex arguments",
			command:  `grep -E "pattern with spaces" file.txt`,
			expected: []string{"grep", "-E", "pattern with spaces", "file.txt"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseCommandLine(tt.command)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if len(result) != len(tt.expected) {
				t.Fatalf("Expected %d args, got %d: %v", len(tt.expected), len(result), result)
			}

			for i, arg := range result {
				if arg != tt.expected[i] {
					t.Errorf("Arg %d: expected '%s', got '%s'", i, tt.expected[i], arg)
				}
			}
		})
	}
}

func TestResolveExecutablePath(t *testing.T) {
	tests := []struct {
		name        string
		executable  string
		expectError bool
		checkPath   func(string) bool
	}{
		{
			name:       "Common command",
			executable: "echo",
			checkPath: func(path string) bool {
				return strings.Contains(path, "echo")
			},
		},
		{
			name:       "Absolute path",
			executable: "/usr/bin/echo",
			checkPath: func(path string) bool {
				return path == "/usr/bin/echo"
			},
		},
		{
			name:        "Non-existent command",
			executable:  "thiscommanddoesnotexist12345",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Skip absolute path test on Windows
			if strings.HasPrefix(tt.executable, "/") && runtime.GOOS == "windows" {
				t.Skip("Skipping Unix path test on Windows")
			}

			path, err := resolveExecutablePath(tt.executable)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if tt.checkPath != nil && !tt.checkPath(path) {
					t.Errorf("Path check failed. Got: %s", path)
				}
			}
		})
	}
}

func TestPwdModule(t *testing.T) {
	module := &PwdModule{}

	if name := module.Name(); name != "pwd" {
		t.Errorf("Expected name 'pwd', got '%s'", name)
	}

	output, exitCode, err := module.Execute([]string{})
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if exitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", exitCode)
	}

	// Verify it returns the current directory
	expectedPwd, _ := os.Getwd()
	if output != expectedPwd {
		t.Errorf("Expected pwd '%s', got '%s'", expectedPwd, output)
	}
}

func TestCdModule(t *testing.T) {
	module := &CdModule{}

	if name := module.Name(); name != "cd" {
		t.Errorf("Expected name 'cd', got '%s'", name)
	}

	// Save current directory
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(originalDir)

	// Create temp directory
	tempDir, err := os.MkdirTemp("", "cdtest")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tests := []struct {
		name        string
		args        []string
		expectError bool
	}{
		{
			name:        "No directory specified",
			args:        []string{},
			expectError: true,
		},
		{
			name: "Valid directory",
			args: []string{tempDir},
		},
		{
			name:        "Invalid directory",
			args:        []string{"/this/directory/should/not/exist/12345"},
			expectError: true,
		},
		{
			name: "Parent directory",
			args: []string{".."},
		},
		{
			name: "Home directory symbol",
			args: []string{"."},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset to original directory before each test
			os.Chdir(originalDir)

			output, exitCode, err := module.Execute(tt.args)

			if tt.expectError {
				if exitCode == 0 {
					t.Error("Expected non-zero exit code for error case")
				}
				if err == nil && !strings.Contains(output, "Error") {
					t.Error("Expected error")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if exitCode != 0 {
					t.Errorf("Expected exit code 0, got %d", exitCode)
				}
				// Output should be the new directory
				if output == "" {
					t.Error("Expected directory path in output")
				}
			}
		})
	}
}

func TestLsModule(t *testing.T) {
	module := &LsModule{}

	if name := module.Name(); name != "ls" {
		t.Errorf("Expected name 'ls', got '%s'", name)
	}

	// Create temp directory with files
	tempDir, err := os.MkdirTemp("", "lstest")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test files
	testFiles := []string{"file1.txt", "file2.txt", "test.log"}
	for _, fname := range testFiles {
		f, err := os.Create(filepath.Join(tempDir, fname))
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
		f.WriteString("test content")
		f.Close()
	}

	// Create subdirectory
	subDir := filepath.Join(tempDir, "subdir")
	if err := os.Mkdir(subDir, 0o755); err != nil {
		t.Fatalf("Failed to create subdirectory: %v", err)
	}

	tests := []struct {
		name        string
		args        []string
		expectError bool
		checkOutput func(string) bool
	}{
		{
			name: "Current directory",
			args: []string{},
			checkOutput: func(output string) bool {
				// Should list files in current directory
				return output != ""
			},
		},
		{
			name: "Specific directory",
			args: []string{tempDir},
			checkOutput: func(output string) bool {
				// Should contain all test files
				for _, fname := range testFiles {
					if !strings.Contains(output, fname) {
						return false
					}
				}
				return strings.Contains(output, "subdir")
			},
		},
		{
			name:        "Non-existent directory",
			args:        []string{"/this/directory/should/not/exist/12345"},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, exitCode, err := module.Execute(tt.args)

			if tt.expectError {
				if exitCode == 0 {
					t.Error("Expected non-zero exit code for error case")
				}
				if err == nil && !strings.Contains(output, "Error") {
					t.Error("Expected error")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if exitCode != 0 {
					t.Errorf("Expected exit code 0, got %d", exitCode)
				}
				if tt.checkOutput != nil && !tt.checkOutput(output) {
					t.Errorf("Output check failed. Got:\n%s", output)
				}
			}
		})
	}
}

func TestExecuteShellCommand(t *testing.T) {
	tests := []struct {
		name          string
		command       string
		expectError   bool
		checkOutput   func(string) bool
		checkExitCode func(int) bool
		skipOnWindows bool
	}{
		{
			name:    "Simple echo",
			command: "echo test",
			checkOutput: func(output string) bool {
				return strings.TrimSpace(output) == "test"
			},
			checkExitCode: func(code int) bool {
				return code == 0
			},
		},
		{
			name:        "Invalid command",
			command:     "invalidcommand12345",
			expectError: true,
			checkExitCode: func(code int) bool {
				return code != 0
			},
		},
		{
			name:          "Command with exit code",
			command:       "false", // Use 'false' which returns exit code 1
			skipOnWindows: true,    // 'false' might not exist on Windows
			checkExitCode: func(code int) bool {
				return code == 1
			},
		},
		{
			name:    "Multiple arguments",
			command: "echo one two three",
			checkOutput: func(output string) bool {
				return strings.TrimSpace(output) == "one two three"
			},
		},
		{
			name:        "Empty command",
			command:     "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skipOnWindows && runtime.GOOS == "windows" {
				t.Skip("Skipping test on Windows")
			}

			output, exitCode, err := executeShellCommand(tt.command)

			if tt.expectError {
				if err == nil && exitCode == 0 {
					t.Error("Expected error or non-zero exit code")
				}
			} else {
				// Note: err might be non-nil even for successful commands with non-zero exit codes
				if tt.checkOutput != nil && !tt.checkOutput(output) {
					t.Errorf("Output check failed. Got: '%s'", output)
				}
			}

			if tt.checkExitCode != nil && !tt.checkExitCode(exitCode) {
				t.Errorf("Exit code check failed. Got: %d", exitCode)
			}
		})
	}
}

func TestTrySystemPaths(t *testing.T) {
	// This test is platform-specific
	if runtime.GOOS == "windows" {
		// Test Windows-specific paths
		tests := []struct {
			name       string
			executable string
			shouldFind bool
		}{
			{
				name:       "cmd.exe",
				executable: "cmd",
				shouldFind: true,
			},
			{
				name:       "notepad.exe",
				executable: "notepad",
				shouldFind: true,
			},
			{
				name:       "Non-existent",
				executable: "thisfiledoesnotexist",
				shouldFind: false,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				path, err := trySystemPaths(tt.executable)

				if tt.shouldFind {
					if err != nil {
						t.Errorf("Expected to find %s, but got error: %v", tt.executable, err)
					}
					if !strings.HasSuffix(path, ".exe") {
						t.Errorf("Expected .exe suffix in path: %s", path)
					}
				} else {
					if err == nil {
						t.Errorf("Expected error for %s, but found at: %s", tt.executable, path)
					}
				}
			})
		}
	} else {
		// Test Unix-specific paths
		tests := []struct {
			name       string
			executable string
			shouldFind bool
		}{
			{
				name:       "ls",
				executable: "ls",
				shouldFind: true,
			},
			{
				name:       "echo",
				executable: "echo",
				shouldFind: true,
			},
			{
				name:       "Non-existent",
				executable: "thisfiledoesnotexist",
				shouldFind: false,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				path, err := trySystemPaths(tt.executable)

				if tt.shouldFind {
					if err != nil {
						t.Errorf("Expected to find %s, but got error: %v", tt.executable, err)
					}
					if !strings.Contains(path, tt.executable) {
						t.Errorf("Expected executable name in path: %s", path)
					}
				} else {
					if err == nil {
						t.Errorf("Expected error for %s, but found at: %s", tt.executable, path)
					}
				}
			})
		}
	}
}

// Integration tests
func TestShellModule_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	pwd := &PwdModule{}
	cd := &CdModule{}
	ls := &LsModule{}

	// Get current directory
	output1, exitCode1, err1 := pwd.Execute([]string{})
	if err1 != nil || exitCode1 != 0 {
		t.Fatalf("PWD failed: %v", err1)
	}
	originalDir := output1

	// Create temp directory
	tempDir, err := os.MkdirTemp("", "shelltest")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	defer os.Chdir(originalDir) // Ensure we return to original dir

	// Change to temp directory
	output2, exitCode2, err2 := cd.Execute([]string{tempDir})
	if err2 != nil || exitCode2 != 0 {
		t.Fatalf("CD failed: %v", err2)
	}
	// On macOS, paths might be resolved (e.g., /var -> /private/var)
	resolvedTempDir, _ := filepath.EvalSymlinks(tempDir)
	resolvedOutput2, _ := filepath.EvalSymlinks(output2)
	if resolvedOutput2 != resolvedTempDir {
		t.Errorf("CD output mismatch: expected %s, got %s", resolvedTempDir, resolvedOutput2)
	}

	// Create a file directly (shell redirection doesn't work without shell)
	testFile := filepath.Join(tempDir, "test.txt")
	err = os.WriteFile(testFile, []byte("test\n"), 0o644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// List directory
	output4, exitCode4, err4 := ls.Execute([]string{tempDir})
	if err4 != nil || exitCode4 != 0 {
		t.Fatalf("LS failed: %v", err4)
	}
	if !strings.Contains(output4, "test.txt") {
		t.Errorf("LS output doesn't contain created file: %s", output4)
	}
}

// Benchmark tests
func BenchmarkParseCommandLine(b *testing.B) {
	command := `grep -E "pattern with spaces" --include="*.txt" /path/to/files`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = parseCommandLine(command)
	}
}

func BenchmarkExecuteShellCommand(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = executeShellCommand("echo benchmark test")
	}
}

func BenchmarkResolveExecutablePath(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = resolveExecutablePath("echo")
	}
}
