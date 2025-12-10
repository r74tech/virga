package extensions

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestScriptLoader_Load(t *testing.T) {
	loader := NewScriptLoader()

	tests := []struct {
		name      string
		manifest  *Manifest
		script    string
		extension string
		wantErr   bool
	}{
		{
			name: "Python script",
			manifest: &Manifest{
				Name:    "test-python",
				Version: "1.0.0",
			},
			script:    "#!/usr/bin/env python3\nprint('Hello from Python')",
			extension: ".py",
			wantErr:   false,
		},
		{
			name: "JavaScript script",
			manifest: &Manifest{
				Name:    "test-js",
				Version: "1.0.0",
			},
			script:    "#!/usr/bin/env node\nconsole.log('Hello from Node.js')",
			extension: ".js",
			wantErr:   false,
		},
		{
			name: "Unsupported script type",
			manifest: &Manifest{
				Name:    "test-unknown",
				Version: "1.0.0",
			},
			script:    "echo 'Hello'",
			extension: ".xyz",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary script file
			tempFile := filepath.Join(t.TempDir(), "script"+tt.extension)
			err := os.WriteFile(tempFile, []byte(tt.script), 0o755)
			if err != nil {
				t.Fatalf("Failed to write temp script: %v", err)
			}

			// Test loading
			ext, err := loader.Load(tt.manifest, tempFile)
			if (err != nil) != tt.wantErr {
				t.Errorf("Load() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr && ext == nil {
				t.Error("Load() returned nil extension without error")
			}
		})
	}
}

func TestScriptExtension_Execute(t *testing.T) {
	// Skip if Python is not available
	if _, err := exec.LookPath("python3"); err != nil && runtime.GOOS != "windows" {
		t.Skip("Python3 not available")
	}
	if _, err := exec.LookPath("python"); err != nil && runtime.GOOS == "windows" {
		t.Skip("Python not available")
	}

	// Create a simple Python script that echoes arguments
	script := `#!/usr/bin/env python3
import os
import json
import sys

args_json = os.environ.get('EXTENSION_ARGS', '{}')
args = json.loads(args_json)

print(f"Message: {args.get('message', 'No message')}")
print(f"Count: {args.get('count', 0)}")

sys.exit(0)
`

	// Create script extension
	manifest := &Manifest{
		Name:    "test-echo",
		Version: "1.0.0",
	}

	interpreter := "python3"
	if runtime.GOOS == "windows" {
		interpreter = "python"
	}

	ext := &ScriptExtension{
		manifest:    manifest,
		interpreter: interpreter,
		script:      []byte(script),
		extension:   ".py",
	}

	// Initialize with mock callbacks
	callbacks := &ExtensionCallbacks{
		Log: func(level, message string, fields map[string]interface{}) {
			t.Logf("[%s] %s %v", level, message, fields)
		},
	}

	err := ext.Initialize(callbacks)
	if err != nil {
		t.Fatalf("Initialize() failed: %v", err)
	}

	// Test execution
	tests := []struct {
		name    string
		args    map[string]interface{}
		wantOut string
	}{
		{
			name: "With arguments",
			args: map[string]interface{}{
				"message": "Hello World",
				"count":   42,
			},
			wantOut: "Message: Hello World\nCount: 42\n",
		},
		{
			name:    "Without arguments",
			args:    map[string]interface{}{},
			wantOut: "Message: No message\nCount: 0\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ext.Execute(tt.args)
			if err != nil {
				t.Fatalf("Execute() failed: %v", err)
			}

			if !result.Success {
				t.Errorf("Execute() Success = false, want true")
			}

			if result.Output != tt.wantOut {
				t.Errorf("Execute() Output = %q, want %q", result.Output, tt.wantOut)
			}

			if result.ExitCode != 0 {
				t.Errorf("Execute() ExitCode = %d, want 0", result.ExitCode)
			}
		})
	}
}

func TestScriptExtension_Timeout(t *testing.T) {
	// Skip if Python is not available
	if _, err := exec.LookPath("python3"); err != nil && runtime.GOOS != "windows" {
		t.Skip("Python3 not available")
	}
	if _, err := exec.LookPath("python"); err != nil && runtime.GOOS == "windows" {
		t.Skip("Python not available")
	}

	// Create a script that sleeps longer than timeout
	script := `#!/usr/bin/env python3
import time
time.sleep(35)  # Sleep longer than 30s timeout
print("This should not be printed")
`

	manifest := &Manifest{
		Name:    "test-timeout",
		Version: "1.0.0",
	}

	interpreter := "python3"
	if runtime.GOOS == "windows" {
		interpreter = "python"
	}

	ext := &ScriptExtension{
		manifest:    manifest,
		interpreter: interpreter,
		script:      []byte(script),
		extension:   ".py",
	}

	// Mock the execution with a shorter timeout for testing
	start := time.Now()
	result, err := ext.Execute(map[string]interface{}{})
	duration := time.Since(start)

	// Should timeout after ~30 seconds
	if duration < 29*time.Second || duration > 35*time.Second {
		t.Errorf("Execute() took %v, expected ~30s timeout", duration)
	}

	if err == nil {
		t.Error("Execute() should return timeout error")
	}

	if result.Success {
		t.Error("Execute() Success = true, want false for timeout")
	}

	if result.Output != "Script execution timed out" {
		t.Errorf("Execute() Output = %q, want timeout message", result.Output)
	}
}

func TestScriptExtension_ErrorHandling(t *testing.T) {
	// Skip if Python is not available
	if _, err := exec.LookPath("python3"); err != nil && runtime.GOOS != "windows" {
		t.Skip("Python3 not available")
	}
	if _, err := exec.LookPath("python"); err != nil && runtime.GOOS == "windows" {
		t.Skip("Python not available")
	}

	// Create a script that exits with error
	script := `#!/usr/bin/env python3
import sys
print("Error occurred", file=sys.stderr)
sys.exit(1)
`

	manifest := &Manifest{
		Name:    "test-error",
		Version: "1.0.0",
	}

	interpreter := "python3"
	if runtime.GOOS == "windows" {
		interpreter = "python"
	}

	ext := &ScriptExtension{
		manifest:    manifest,
		interpreter: interpreter,
		script:      []byte(script),
		extension:   ".py",
	}

	result, err := ext.Execute(map[string]interface{}{})
	if err != nil {
		t.Logf("Execute() returned error: %v", err)
	}

	if result.Success {
		t.Error("Execute() Success = true, want false for error exit")
	}

	if result.ExitCode != 1 {
		t.Errorf("Execute() ExitCode = %d, want 1", result.ExitCode)
	}

	if result.Output == "" {
		t.Error("Execute() Output is empty, should contain stderr")
	}
}

func TestDetectInterpreter(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			name:    "Python shebang",
			content: "#!/usr/bin/env python3\nprint('hello')",
			want:    "python3",
		},
		{
			name:    "Node.js shebang",
			content: "#!/usr/bin/env node\nconsole.log('hello')",
			want:    "node",
		},
		{
			name:    "Direct interpreter path",
			content: "#!/usr/bin/python\nprint('hello')",
			want:    "python",
		},
		{
			name:    "No shebang",
			content: "print('hello')",
			want:    "",
		},
		{
			name:    "Empty content",
			content: "",
			want:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := detectInterpreter([]byte(tt.content))
			if got != tt.want {
				t.Errorf("detectInterpreter() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestScriptLoader_GetType(t *testing.T) {
	loader := NewScriptLoader()
	if loader.GetType() != ExtensionTypeScript {
		t.Errorf("GetType() = %v, want %v", loader.GetType(), ExtensionTypeScript)
	}
}

// Benchmark script execution
func BenchmarkScriptExecution(b *testing.B) {
	// Skip if Python is not available
	if _, err := exec.LookPath("python3"); err != nil && runtime.GOOS != "windows" {
		b.Skip("Python3 not available")
	}
	if _, err := exec.LookPath("python"); err != nil && runtime.GOOS == "windows" {
		b.Skip("Python not available")
	}

	script := `#!/usr/bin/env python3
import os
import json

args = json.loads(os.environ.get('EXTENSION_ARGS', '{}'))
print(f"Processed: {args.get('data', 'none')}")
`

	manifest := &Manifest{
		Name:    "bench-script",
		Version: "1.0.0",
	}

	interpreter := "python3"
	if runtime.GOOS == "windows" {
		interpreter = "python"
	}

	ext := &ScriptExtension{
		manifest:    manifest,
		interpreter: interpreter,
		script:      []byte(script),
		extension:   ".py",
	}

	args := map[string]interface{}{
		"data": "benchmark test data",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := ext.Execute(args)
		if err != nil {
			b.Fatalf("Execute() failed: %v", err)
		}
	}
}
