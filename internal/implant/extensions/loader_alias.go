package extensions

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"text/template"
	"time"
)

// AliasLoader loads alias/wrapper extensions (simple command wrappers)
type AliasLoader struct {
	// Store loaded aliases
	loadedAliases map[string]*AliasConfig
}

// AliasConfig represents the configuration for an alias
type AliasConfig struct {
	Command    string            `json:"command"`
	Args       []string          `json:"args"`
	WorkingDir string            `json:"working_dir"`
	Env        map[string]string `json:"env"`
	Shell      bool              `json:"shell"` // Whether to run through shell
}

// NewAliasLoader creates a new alias extension loader
func NewAliasLoader() *AliasLoader {
	return &AliasLoader{
		loadedAliases: make(map[string]*AliasConfig),
	}
}

// Load loads an alias extension
func (l *AliasLoader) Load(manifest *Manifest, path string) (Extension, error) {
	// For alias extensions, the manifest should contain the command info
	// The path might point to a config file or executable

	config := &AliasConfig{
		Shell: true, // Default to shell execution
	}

	// Extract command from manifest
	if manifest.Entrypoint != "" {
		config.Command = manifest.Entrypoint
	} else {
		// Use the path as the command
		config.Command = path
	}

	// Store the alias config
	l.loadedAliases[manifest.Name] = config

	return &AliasExtension{
		manifest: manifest,
		config:   config,
		loader:   l,
	}, nil
}

// Unload unloads an alias extension
func (l *AliasLoader) Unload(extension Extension) error {
	if aliasExt, ok := extension.(*AliasExtension); ok {
		delete(l.loadedAliases, aliasExt.manifest.Name)
	}
	return nil
}

// GetType returns the type of extensions this loader handles
func (l *AliasLoader) GetType() ExtensionType {
	return ExtensionTypeAlias
}

// AliasExtension represents an alias extension
type AliasExtension struct {
	manifest  *Manifest
	config    *AliasConfig
	loader    *AliasLoader
	callbacks *ExtensionCallbacks
}

// GetManifest returns the extension manifest
func (e *AliasExtension) GetManifest() *Manifest {
	return e.manifest
}

// Initialize initializes the alias extension
func (e *AliasExtension) Initialize(callbacks *ExtensionCallbacks) error {
	e.callbacks = callbacks

	// Verify the command exists
	if !e.config.Shell {
		if _, err := exec.LookPath(e.config.Command); err != nil {
			return fmt.Errorf("command not found: %s", e.config.Command)
		}
	}

	if callbacks != nil && callbacks.Log != nil {
		callbacks.Log("info", fmt.Sprintf("Alias extension '%s' initialized", e.manifest.Name), nil)
	}

	return nil
}

// Execute executes the alias extension
func (e *AliasExtension) Execute(args map[string]interface{}) (*ExtensionResult, error) {
	// Build the command
	cmdStr, cmdArgs := e.buildCommand(args)

	var cmd *exec.Cmd
	if e.config.Shell {
		// Execute through shell
		if runtime.GOOS == "windows" {
			cmd = exec.Command("cmd", "/c", cmdStr)
		} else {
			cmd = exec.Command("sh", "-c", cmdStr)
		}
	} else {
		// Direct execution
		cmd = exec.Command(cmdStr, cmdArgs...)
	}

	// Set working directory if specified
	if e.config.WorkingDir != "" {
		cmd.Dir = e.config.WorkingDir
	}

	// Set environment variables
	if len(e.config.Env) > 0 {
		cmd.Env = os.Environ()
		for k, v := range e.config.Env {
			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
		}
	}

	// Capture output
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Execute with timeout
	done := make(chan error, 1)
	go func() {
		done <- cmd.Run()
	}()

	// Default timeout of 5 minutes for alias commands
	timeout := 5 * time.Minute

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
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
		return &ExtensionResult{
			Success:   false,
			Output:    "Command execution timed out",
			Error:     "timeout",
			ExitCode:  -1,
			Timestamp: time.Now(),
		}, fmt.Errorf("command execution timed out after %v", timeout)
	}
}

// Cleanup cleans up the alias extension
func (e *AliasExtension) Cleanup() error {
	if e.callbacks != nil && e.callbacks.Log != nil {
		e.callbacks.Log("info", fmt.Sprintf("Alias extension '%s' cleaned up", e.manifest.Name), nil)
	}
	return nil
}

// buildCommand builds the command string and arguments from the extension arguments
func (e *AliasExtension) buildCommand(args map[string]interface{}) (string, []string) {
	// Start with the base command
	cmdParts := []string{e.config.Command}

	// Add configured arguments
	cmdParts = append(cmdParts, e.config.Args...)

	// Process extension arguments based on manifest
	for _, arg := range e.manifest.Arguments {
		if value, exists := args[arg.Name]; exists {
			// Convert value to string
			strValue := fmt.Sprintf("%v", value)

			// Apply any templating if needed
			if strings.Contains(strValue, "{{") {
				tmpl, err := template.New("arg").Parse(strValue)
				if err == nil {
					var buf bytes.Buffer
					tmpl.Execute(&buf, args)
					strValue = buf.String()
				}
			}

			cmdParts = append(cmdParts, strValue)
		} else if arg.Default != "" {
			cmdParts = append(cmdParts, arg.Default)
		}
	}

	if e.config.Shell {
		// Return as a single command string for shell execution
		return strings.Join(cmdParts, " "), nil
	}

	// Return command and args separately
	if len(cmdParts) > 1 {
		return cmdParts[0], cmdParts[1:]
	}
	return cmdParts[0], nil
}

// Common alias configurations - commented out as unused
// var commonAliases = map[string]*AliasConfig{
// 	"nmap": {
// 		Command: "nmap",
// 		Args:    []string{"-sn"},
// 		Shell:   false,
// 	},
// 	"curl": {
// 		Command: "curl",
// 		Args:    []string{"-s"},
// 		Shell:   false,
// 	},
// 	"wget": {
// 		Command: "wget",
// 		Args:    []string{"-q", "-O", "-"},
// 		Shell:   false,
// 	},
// 	"ping": {
// 		Command: "ping",
// 		Args:    []string{"-c", "4"},
// 		Shell:   false,
// 	},
// }
