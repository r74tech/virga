package extensions

import (
	"runtime"
	"strings"
	"testing"
)

func TestAliasLoader_Load(t *testing.T) {
	loader := NewAliasLoader()

	tests := []struct {
		name     string
		manifest *Manifest
		path     string
		wantErr  bool
	}{
		{
			name: "Load with entrypoint",
			manifest: &Manifest{
				Name:       "test-alias",
				Version:    "1.0.0",
				Entrypoint: "echo",
			},
			path:    "/usr/bin/echo",
			wantErr: false,
		},
		{
			name: "Load with path as command",
			manifest: &Manifest{
				Name:    "test-path",
				Version: "1.0.0",
			},
			path:    "ls",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ext, err := loader.Load(tt.manifest, tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("Load() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr && ext == nil {
				t.Error("Load() returned nil extension without error")
			}

			if !tt.wantErr {
				aliasExt, ok := ext.(*AliasExtension)
				if !ok {
					t.Error("Load() returned wrong type")
				}

				if aliasExt.manifest.Name != tt.manifest.Name {
					t.Errorf("Extension name = %q, want %q", aliasExt.manifest.Name, tt.manifest.Name)
				}
			}
		})
	}
}

func TestAliasExtension_Execute(t *testing.T) {
	// Use echo command which is available on all platforms
	echoCmd := "echo"
	if runtime.GOOS == "windows" {
		echoCmd = "cmd"
	}

	manifest := &Manifest{
		Name:    "test-echo",
		Version: "1.0.0",
		Arguments: []ExtensionArgument{
			{
				Name:        "message",
				Type:        "string",
				Description: "Message to echo",
				Optional:    false,
			},
		},
	}

	config := &AliasConfig{
		Command: echoCmd,
		Shell:   true,
	}

	if runtime.GOOS == "windows" {
		config.Args = []string{"/c", "echo"}
	}

	ext := &AliasExtension{
		manifest: manifest,
		config:   config,
	}

	// Initialize
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
		name       string
		args       map[string]interface{}
		wantOutput string
		wantErr    bool
	}{
		{
			name: "Simple echo",
			args: map[string]interface{}{
				"message": "Hello World",
			},
			wantOutput: "Hello World",
			wantErr:    false,
		},
		{
			name: "Empty message",
			args: map[string]interface{}{
				"message": "",
			},
			wantOutput: "",
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ext.Execute(tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				if !result.Success {
					t.Error("Execute() Success = false, want true")
				}

				// Normalize output (remove trailing newlines/spaces)
				gotOutput := strings.TrimSpace(result.Output)
				wantOutput := strings.TrimSpace(tt.wantOutput)

				if gotOutput != wantOutput {
					t.Errorf("Execute() Output = %q, want %q", gotOutput, wantOutput)
				}

				if result.ExitCode != 0 {
					t.Errorf("Execute() ExitCode = %d, want 0", result.ExitCode)
				}
			}
		})
	}
}

func TestAliasExtension_BuildCommand(t *testing.T) {
	manifest := &Manifest{
		Name:    "test-build",
		Version: "1.0.0",
		Arguments: []ExtensionArgument{
			{
				Name:     "target",
				Type:     "string",
				Optional: false,
			},
			{
				Name:     "port",
				Type:     "int",
				Optional: true,
				Default:  "80",
			},
		},
	}

	config := &AliasConfig{
		Command: "nmap",
		Args:    []string{"-sn"},
		Shell:   false,
	}

	ext := &AliasExtension{
		manifest: manifest,
		config:   config,
	}

	tests := []struct {
		name     string
		args     map[string]interface{}
		wantCmd  string
		wantArgs []string
	}{
		{
			name: "All arguments",
			args: map[string]interface{}{
				"target": "192.168.1.1",
				"port":   443,
			},
			wantCmd:  "nmap",
			wantArgs: []string{"-sn", "192.168.1.1", "443"},
		},
		{
			name: "Default port",
			args: map[string]interface{}{
				"target": "192.168.1.1",
			},
			wantCmd:  "nmap",
			wantArgs: []string{"-sn", "192.168.1.1", "80"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd, args := ext.buildCommand(tt.args)

			if cmd != tt.wantCmd {
				t.Errorf("buildCommand() cmd = %q, want %q", cmd, tt.wantCmd)
			}

			if len(args) != len(tt.wantArgs) {
				t.Errorf("buildCommand() args length = %d, want %d", len(args), len(tt.wantArgs))
			}

			for i, arg := range args {
				if i < len(tt.wantArgs) && arg != tt.wantArgs[i] {
					t.Errorf("buildCommand() args[%d] = %q, want %q", i, arg, tt.wantArgs[i])
				}
			}
		})
	}
}

func TestAliasExtension_ShellMode(t *testing.T) {
	manifest := &Manifest{
		Name:    "test-shell",
		Version: "1.0.0",
	}

	config := &AliasConfig{
		Command: "echo 'test command'",
		Shell:   true,
	}

	ext := &AliasExtension{
		manifest: manifest,
		config:   config,
	}

	cmd, args := ext.buildCommand(map[string]interface{}{})

	if config.Shell {
		// In shell mode, should return full command as string
		if cmd != "echo 'test command'" {
			t.Errorf("buildCommand() in shell mode cmd = %q, want full command", cmd)
		}

		if args != nil {
			t.Error("buildCommand() in shell mode should return nil args")
		}
	}
}

func TestAliasLoader_GetType(t *testing.T) {
	loader := NewAliasLoader()
	if loader.GetType() != ExtensionTypeAlias {
		t.Errorf("GetType() = %v, want %v", loader.GetType(), ExtensionTypeAlias)
	}
}

func TestAliasLoader_Unload(t *testing.T) {
	loader := NewAliasLoader()

	manifest := &Manifest{
		Name:    "test-unload",
		Version: "1.0.0",
	}

	// Load an extension
	ext, err := loader.Load(manifest, "echo")
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	// Verify it was stored
	if _, exists := loader.loadedAliases[manifest.Name]; !exists {
		t.Error("Extension not stored in loadedAliases")
	}

	// Unload
	err = loader.Unload(ext)
	if err != nil {
		t.Errorf("Unload() failed: %v", err)
	}

	// Verify it was removed
	if _, exists := loader.loadedAliases[manifest.Name]; exists {
		t.Error("Extension still in loadedAliases after unload")
	}
}
