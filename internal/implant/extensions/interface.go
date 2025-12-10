package extensions

import (
	"fmt"
	"time"
)

// Extension is the interface that all extensions must implement
type Extension interface {
	// GetManifest returns the extension manifest
	GetManifest() *Manifest

	// Initialize is called when the extension is loaded
	Initialize(callbacks *ExtensionCallbacks) error

	// Execute runs the extension with the given arguments
	Execute(args map[string]interface{}) (*ExtensionResult, error)

	// Cleanup is called when the extension is unloaded
	Cleanup() error
}

// Manifest describes an extension's metadata and requirements
type Manifest struct {
	Name            string              `json:"name"`
	Version         string              `json:"version"`
	Type            string              `json:"type,omitempty"` // Extension type: native, bof, script, alias
	ExtensionAuthor string              `json:"extension_author"`
	OriginalAuthor  string              `json:"original_author,omitempty"`
	RepoURL         string              `json:"repo_url,omitempty"`
	Help            string              `json:"help"`
	LongHelp        string              `json:"long_help,omitempty"`
	Entrypoint      string              `json:"entrypoint,omitempty"`
	Init            string              `json:"init,omitempty"`
	DependsOn       []string            `json:"depends_on,omitempty"`
	Files           []ExtensionFile     `json:"files"`
	Arguments       []ExtensionArgument `json:"arguments,omitempty"`
	Commands        []ExtensionCommand  `json:"commands,omitempty"`
}

// ExtensionFile describes a platform-specific extension file
type ExtensionFile struct {
	OS   string `json:"os"`
	Arch string `json:"arch"`
	Path string `json:"path"`
}

// ExtensionArgument describes an argument for the extension
type ExtensionArgument struct {
	Name        string `json:"name"`
	Type        string `json:"type"` // string, int, bool, file
	Description string `json:"desc"`
	Optional    bool   `json:"optional"`
	Default     string `json:"default,omitempty"`
}

// ExtensionCommand describes a command provided by the extension
type ExtensionCommand struct {
	Name     string              `json:"name"`
	Help     string              `json:"help"`
	LongHelp string              `json:"long_help,omitempty"`
	Args     []ExtensionArgument `json:"args,omitempty"`
}

// ExtensionCallbacks provides callbacks for extensions to interact with the implant
type ExtensionCallbacks struct {
	// SendOutput sends output back to the C2 server
	SendOutput func(output string) error

	// SendError sends an error message back to the C2 server
	SendError func(err error) error

	// GetEnv gets an environment variable
	GetEnv func(key string) string

	// SetEnv sets an environment variable
	SetEnv func(key, value string) error

	// ReadFile reads a file from the filesystem
	ReadFile func(path string) ([]byte, error)

	// WriteFile writes a file to the filesystem
	WriteFile func(path string, data []byte) error

	// ExecuteCommand executes a system command
	ExecuteCommand func(command string, args []string) (string, error)

	// Log logs a message
	Log func(level, message string, fields map[string]interface{})

	// GetSystemInfo gets system information
	GetSystemInfo func() map[string]interface{}
}

// ExtensionResult represents the result of an extension execution
type ExtensionResult struct {
	Success   bool                   `json:"success"`
	Output    string                 `json:"output,omitempty"`
	Error     string                 `json:"error,omitempty"`
	ExitCode  int                    `json:"exit_code"`
	Data      map[string]interface{} `json:"data,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

// ExtensionType represents the type of extension
type ExtensionType string

const (
	ExtensionTypeNative ExtensionType = "native" // .dll, .so, .dylib
	ExtensionTypeBOF    ExtensionType = "bof"    // .o files
	ExtensionTypeScript ExtensionType = "script" // .py, .js, etc.
	ExtensionTypeAlias  ExtensionType = "alias"  // Command aliases
)

// ExtensionError represents an extension-specific error
type ExtensionError struct {
	Extension string
	Stage     string // load, init, execute, cleanup
	Err       error
}

func (e *ExtensionError) Error() string {
	return fmt.Sprintf("extension error [%s] during %s: %v", e.Extension, e.Stage, e.Err)
}
