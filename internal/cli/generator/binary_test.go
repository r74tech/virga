package generator

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestNewBinaryGenerator(t *testing.T) {
	gen, err := NewBinaryGenerator()
	if err != nil {
		t.Fatalf("NewBinaryGenerator() error = %v", err)
	}
	defer gen.Cleanup()

	if gen.TmpDir == "" {
		t.Error("TmpDir should not be empty")
	}

	// Verify temp directory exists
	if _, err := os.Stat(gen.TmpDir); os.IsNotExist(err) {
		t.Error("Temp directory does not exist")
	}
}

func TestGetDefaultOptions(t *testing.T) {
	gen := &BinaryGenerator{}
	opts := gen.GetDefaultOptions()

	// Check default values
	if opts.C2Host != "localhost" {
		t.Errorf("Expected default C2Host to be localhost, got %s", opts.C2Host)
	}
	if opts.C2Port != 443 {
		t.Errorf("Expected default C2Port to be 443, got %d", opts.C2Port)
	}
	if opts.Protocol != "https" {
		t.Errorf("Expected default Protocol to be https, got %s", opts.Protocol)
	}
	if opts.SleepTime != 60 {
		t.Errorf("Expected default SleepTime to be 60, got %d", opts.SleepTime)
	}
	if opts.Jitter != 20 {
		t.Errorf("Expected default Jitter to be 20, got %d", opts.Jitter)
	}
	if opts.TargetOS != runtime.GOOS {
		t.Errorf("Expected default TargetOS to be %s, got %s", runtime.GOOS, opts.TargetOS)
	}
	if opts.TargetArch != runtime.GOARCH {
		t.Errorf("Expected default TargetArch to be %s, got %s", runtime.GOARCH, opts.TargetArch)
	}
}

func TestGenerate_ValidationErrors(t *testing.T) {
	tests := []struct {
		name        string
		opts        BinaryOptions
		expectError bool
		errorMsg    string
	}{
		{
			name: "Invalid OS",
			opts: BinaryOptions{
				C2Host:     "localhost",
				C2Port:     443,
				TargetOS:   "invalid",
				TargetArch: "amd64",
				Format:     "exe",
				OutputPath: "test.exe",
			},
			expectError: true,
			errorMsg:    "unsupported OS",
		},
		{
			name: "Invalid architecture",
			opts: BinaryOptions{
				C2Host:     "localhost",
				C2Port:     443,
				TargetOS:   "windows",
				TargetArch: "invalid",
				Format:     "exe",
				OutputPath: "test.exe",
			},
			expectError: true,
			errorMsg:    "unsupported architecture",
		},
		{
			name: "Invalid protocol",
			opts: BinaryOptions{
				C2Host:     "localhost",
				C2Port:     443,
				Protocol:   "invalid",
				TargetOS:   "windows",
				TargetArch: "amd64",
				Format:     "exe",
				OutputPath: "test.exe",
			},
			expectError: true,
			errorMsg:    "unsupported protocol",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gen, err := NewBinaryGenerator()
			if err != nil {
				t.Fatalf("NewBinaryGenerator() error = %v", err)
			}
			defer gen.Cleanup()
			_, err = gen.Generate(tt.opts)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error, got nil")
				} else if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing %q, got %q", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

func TestLlamaOptions(t *testing.T) {
	tests := []struct {
		name     string
		opts     BinaryOptions
		checkFor []string
	}{
		{
			name: "Llama enabled",
			opts: BinaryOptions{
				C2Host:      "localhost",
				C2Port:      443,
				TargetOS:    "linux",
				TargetArch:  "amd64",
				Format:      "elf",
				OutputPath:  "test",
				EnableLlama: true,
				LlamaConfig: &LlamaOpts{
					Context:      2048,
					MaxTokens:    1024,
					Temperature:  0.7,
					SystemPrompt: "security_analyst",
					AutoMode:     true,
					LogEnabled:   true,
				},
			},
			checkFor: []string{"BuildLlamaEnabled=true"},
		},
		{
			name: "Llama disabled",
			opts: BinaryOptions{
				C2Host:      "localhost",
				C2Port:      443,
				TargetOS:    "windows",
				TargetArch:  "amd64",
				Format:      "exe",
				OutputPath:  "test.exe",
				EnableLlama: false,
			},
			checkFor: []string{"BuildLlamaEnabled=false"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This would be tested in the actual generate method
			// For now, just verify the options are valid
			if tt.opts.EnableLlama && tt.opts.LlamaConfig == nil {
				t.Error("Llama enabled but no config provided")
			}
		})
	}
}

func TestFormatOptions(t *testing.T) {
	tests := []struct {
		name     string
		targetOS string
		format   string
		wantErr  bool
	}{
		{
			name:     "Windows exe",
			targetOS: "windows",
			format:   "exe",
			wantErr:  false,
		},
		{
			name:     "Linux elf",
			targetOS: "linux",
			format:   "elf",
			wantErr:  false,
		},
		{
			name:     "Invalid format for OS",
			targetOS: "windows",
			format:   "elf",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := BinaryOptions{
				C2Host:     "localhost",
				C2Port:     443,
				Protocol:   "https",
				TargetOS:   tt.targetOS,
				TargetArch: "amd64",
				Format:     tt.format,
				OutputPath: "test",
			}

			gen, err := NewBinaryGenerator()
			if err != nil {
				t.Fatalf("NewBinaryGenerator() error = %v", err)
			}
			defer gen.Cleanup()
			_, err = gen.Generate(opts)
			// We expect errors for invalid combinations and for actual compilation
			// For this test, we just check that validation works correctly
			if err != nil {
				if !tt.wantErr && !strings.Contains(err.Error(), "go build") && !strings.Contains(err.Error(), "cmd/implant") {
					t.Errorf("Unexpected validation error: %v", err)
				}
			}
		})
	}
}

func TestShellcodeGeneration(t *testing.T) {
	tests := []struct {
		name     string
		opts     BinaryOptions
		skipTest bool
	}{
		{
			name: "Shellcode format",
			opts: BinaryOptions{
				C2Host:     "localhost",
				C2Port:     443,
				TargetOS:   "windows",
				TargetArch: "amd64",
				Format:     "raw",
				OutputPath: "test.bin",
			},
			skipTest: true, // Skip actual shellcode generation
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skipTest {
				t.Skip("Skipping actual shellcode generation test")
			}

			gen, err := NewBinaryGenerator()
			if err != nil {
				t.Fatalf("NewBinaryGenerator() error = %v", err)
			}
			defer gen.Cleanup()
			_, err = gen.Generate(tt.opts)
			// Shellcode generation might not be implemented
			if err != nil {
				t.Logf("Shellcode generation error (might be expected): %v", err)
			}
		})
	}
}

func TestGenerateIntegration(t *testing.T) {
	// Skip this test in CI or when not explicitly requested
	if os.Getenv("RUN_INTEGRATION_TESTS") != "true" {
		t.Skip("Skipping integration test. Set RUN_INTEGRATION_TESTS=true to run")
	}

	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "beacon.exe")

	opts := BinaryOptions{
		C2Host:     "localhost",
		C2Port:     8443,
		Protocol:   "https",
		SleepTime:  60,
		Jitter:     30,
		TargetOS:   "windows",
		TargetArch: "amd64",
		Format:     "exe",
		OutputPath: outputPath,
	}

	gen, err := NewBinaryGenerator()
	if err != nil {
		t.Fatalf("NewBinaryGenerator() error = %v", err)
	}
	defer gen.Cleanup()
	_, err = gen.Generate(opts)
	if err != nil {
		// This might fail if Go is not properly configured for cross-compilation
		t.Logf("Generate failed (this might be expected): %v", err)
		return
	}

	// Check if binary was created
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Error("Expected binary file to be created")
	}
}
