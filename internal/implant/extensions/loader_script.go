package extensions

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// ScriptLoader loads and executes script-based extensions (.py, .js, .rb, etc.)
// WARNING: Script extensions are NOT recommended for production use!
// They require interpreters on the target system and are easily detected.
// Use BOF or Native extensions for real operations.
type ScriptLoader struct {
	interpreters map[string]string
	tempDir      string
}

// NewScriptLoader creates a new script extension loader
func NewScriptLoader() *ScriptLoader {
	// Map file extensions to interpreter commands
	interpreters := map[string]string{
		".py":  "python3",
		".js":  "node",
		".rb":  "ruby",
		".pl":  "perl",
		".lua": "lua",
		".php": "php",
	}

	// On Windows, Python might be "python" instead of "python3"
	if runtime.GOOS == "windows" {
		interpreters[".py"] = "python"
	}

	return &ScriptLoader{
		interpreters: interpreters,
		tempDir:      os.TempDir(),
	}
}

// Load loads a script extension
func (l *ScriptLoader) Load(manifest *Manifest, path string) (Extension, error) {
	ext := filepath.Ext(path)
	interpreter, ok := l.interpreters[ext]
	if !ok {
		return nil, fmt.Errorf("unsupported script type: %s", ext)
	}

	// Read the script content
	scriptContent, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read script: %w", err)
	}

	// Verify interpreter is available
	if _, err := exec.LookPath(interpreter); err != nil {
		return nil, fmt.Errorf("interpreter %s not found in PATH", interpreter)
	}

	// Log warning about script extensions
	fmt.Fprintf(os.Stderr, "[WARNING] Loading script extension '%s'. Script extensions are NOT recommended for production use!\n", manifest.Name)

	return &ScriptExtension{
		manifest:    manifest,
		interpreter: interpreter,
		scriptPath:  path,
		script:      scriptContent,
		extension:   ext,
	}, nil
}

// Unload unloads a script extension
func (l *ScriptLoader) Unload(extension Extension) error {
	// Scripts don't need special unloading
	return nil
}

// GetType returns the type of extensions this loader handles
func (l *ScriptLoader) GetType() ExtensionType {
	return ExtensionTypeScript
}

// ScriptExtension represents a script-based extension
type ScriptExtension struct {
	manifest    *Manifest
	interpreter string
	scriptPath  string
	script      []byte
	extension   string
	callbacks   *ExtensionCallbacks
}

// GetManifest returns the extension manifest
func (e *ScriptExtension) GetManifest() *Manifest {
	return e.manifest
}

// Initialize initializes the script extension
func (e *ScriptExtension) Initialize(callbacks *ExtensionCallbacks) error {
	e.callbacks = callbacks

	// Log initialization
	if callbacks != nil && callbacks.Log != nil {
		callbacks.Log("info", fmt.Sprintf("Script extension '%s' initialized", e.manifest.Name), nil)
	}

	return nil
}

// Execute executes the script extension
func (e *ScriptExtension) Execute(args map[string]interface{}) (*ExtensionResult, error) {
	// Create a temporary script file
	tempFile := filepath.Join(os.TempDir(), fmt.Sprintf("ext_%s_%d%s",
		e.manifest.Name, time.Now().UnixNano(), e.extension))

	// Write script to temp file
	if err := os.WriteFile(tempFile, e.script, 0o600); err != nil {
		return nil, fmt.Errorf("failed to write temp script: %w", err)
	}
	defer os.Remove(tempFile)

	// Prepare command and arguments
	cmdArgs := []string{tempFile}

	// Convert arguments to JSON and pass as environment variable or stdin
	if len(args) > 0 {
		argsJSON, err := json.Marshal(args)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal arguments: %w", err)
		}

		// Set as environment variable
		os.Setenv("EXTENSION_ARGS", string(argsJSON))
		defer os.Unsetenv("EXTENSION_ARGS")
	}

	// Prepare the command
	cmd := exec.Command(e.interpreter, cmdArgs...)

	// Set up pipes
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Set working directory to temp directory
	cmd.Dir = os.TempDir()

	// Set environment variables for the script
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("EXTENSION_NAME=%s", e.manifest.Name),
		fmt.Sprintf("EXTENSION_VERSION=%s", e.manifest.Version),
	)

	// Execute the script with timeout
	done := make(chan error, 1)
	go func() {
		done <- cmd.Run()
	}()

	// Wait for completion or timeout (30 seconds default)
	timeout := 30 * time.Second
	select {
	case err := <-done:
		exitCode := 0
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
			} else {
				exitCode = -1
			}
		}

		output := stdout.String()
		errorOutput := stderr.String()

		// Combine outputs
		if errorOutput != "" {
			if output != "" {
				output = fmt.Sprintf("%s\nError:\n%s", output, errorOutput)
			} else {
				output = errorOutput
			}
		}

		return &ExtensionResult{
			Success:   err == nil,
			Output:    output,
			ExitCode:  exitCode,
			Timestamp: time.Now(),
		}, nil

	case <-time.After(timeout):
		// Kill the process
		cmd.Process.Kill()
		return &ExtensionResult{
			Success:   false,
			Output:    "Script execution timed out",
			Error:     "timeout",
			ExitCode:  -1,
			Timestamp: time.Now(),
		}, fmt.Errorf("script execution timed out after %v", timeout)
	}
}

// Cleanup cleans up the script extension
func (e *ScriptExtension) Cleanup() error {
	if e.callbacks != nil && e.callbacks.Log != nil {
		e.callbacks.Log("info", fmt.Sprintf("Script extension '%s' cleaned up", e.manifest.Name), nil)
	}
	return nil
}

// Script templates commented out - kept for future reference
// const pythonTemplate = `#!/usr/bin/env python3
// import os
// import sys
// import json
//
// def main():
//     # Get arguments from environment
//     args_json = os.environ.get('EXTENSION_ARGS', '{}')
//     args = json.loads(args_json)
//
//     # Extension logic here
//     print(f"Extension executed with args: {args}")
//
//     # Return success
//     sys.exit(0)
//
// if __name__ == "__main__":
//     main()
// `
//
// // JavaScript/Node.js script template for extensions
// const nodeTemplate = `#!/usr/bin/env node
// const args = JSON.parse(process.env.EXTENSION_ARGS || '{}');
//
// // Extension logic here
// console.log('Extension executed with args:', args);
//
// // Return success
// process.exit(0);
// `

// detectInterpreter tries to detect the appropriate interpreter from shebang
func detectInterpreter(content []byte) string {
	lines := strings.Split(string(content), "\n")
	if len(lines) == 0 {
		return ""
	}

	firstLine := strings.TrimSpace(lines[0])
	if strings.HasPrefix(firstLine, "#!") {
		shebang := strings.TrimPrefix(firstLine, "#!")
		parts := strings.Fields(shebang)
		if len(parts) > 0 {
			// Extract just the interpreter name
			interpreter := filepath.Base(parts[0])
			// Handle env cases like "#!/usr/bin/env python3"
			if interpreter == "env" && len(parts) > 1 {
				interpreter = parts[1]
			}
			return interpreter
		}
	}

	return ""
}
