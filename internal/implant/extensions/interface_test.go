package extensions

import (
	"testing"
	"time"
)

func TestExtensionError_Error(t *testing.T) {
	tests := []struct {
		name string
		err  *ExtensionError
		want string
	}{
		{
			name: "Simple error",
			err: &ExtensionError{
				Extension: "test-ext",
				Stage:     "load",
				Err:       &testError{"failed to load"},
			},
			want: "extension error [test-ext] during load: failed to load",
		},
		{
			name: "Empty extension name",
			err: &ExtensionError{
				Extension: "",
				Stage:     "execute",
				Err:       &testError{"execution failed"},
			},
			want: "extension error [] during execute: execution failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.err.Error()
			if got != tt.want {
				t.Errorf("ExtensionError.Error() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExtensionResult_Validation(t *testing.T) {
	// Test creating various extension results
	tests := []struct {
		name   string
		result ExtensionResult
		valid  bool
	}{
		{
			name: "Success result",
			result: ExtensionResult{
				Success:   true,
				Output:    "Command executed successfully",
				ExitCode:  0,
				Timestamp: time.Now(),
			},
			valid: true,
		},
		{
			name: "Failure result",
			result: ExtensionResult{
				Success:   false,
				Output:    "",
				Error:     "Command failed",
				ExitCode:  1,
				Timestamp: time.Now(),
			},
			valid: true,
		},
		{
			name: "Result with data",
			result: ExtensionResult{
				Success:  true,
				Output:   "Processed",
				ExitCode: 0,
				Data: map[string]interface{}{
					"count":  10,
					"status": "completed",
				},
				Timestamp: time.Now(),
			},
			valid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Basic validation
			if tt.result.Success && tt.result.ExitCode != 0 {
				t.Error("Success = true but ExitCode != 0")
			}

			if !tt.result.Success && tt.result.Error == "" {
				t.Error("Success = false but no Error message")
			}

			if tt.result.Timestamp.IsZero() {
				t.Error("Timestamp is zero")
			}
		})
	}
}

func TestManifest_Validation(t *testing.T) {
	tests := []struct {
		name     string
		manifest Manifest
		wantErr  bool
	}{
		{
			name: "Valid manifest",
			manifest: Manifest{
				Name:            "valid-extension",
				Version:         "1.0.0",
				ExtensionAuthor: "Test Author",
				Help:            "A test extension",
				Files: []ExtensionFile{
					{
						OS:   "linux",
						Arch: "amd64",
						Path: "extension.so",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Manifest with commands",
			manifest: Manifest{
				Name:            "command-extension",
				Version:         "1.0.0",
				ExtensionAuthor: "Test Author",
				Help:            "Extension with commands",
				Commands: []ExtensionCommand{
					{
						Name: "scan",
						Help: "Scan a target",
						Args: []ExtensionArgument{
							{
								Name:        "target",
								Type:        "string",
								Description: "Target to scan",
								Optional:    false,
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name:     "Empty manifest",
			manifest: Manifest{},
			wantErr:  true, // Should have at least name and version
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simple validation
			err := validateManifest(&tt.manifest)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateManifest() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestExtensionType_String(t *testing.T) {
	tests := []struct {
		extType ExtensionType
		want    string
	}{
		{ExtensionTypeNative, "native"},
		{ExtensionTypeBOF, "bof"},
		{ExtensionTypeScript, "script"},
		{ExtensionTypeAlias, "alias"},
		{ExtensionType("custom"), "custom"},
	}

	for _, tt := range tests {
		t.Run(string(tt.extType), func(t *testing.T) {
			if string(tt.extType) != tt.want {
				t.Errorf("ExtensionType string = %q, want %q", tt.extType, tt.want)
			}
		})
	}
}

// Helper types and functions

type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}

func validateManifest(manifest *Manifest) error {
	if manifest.Name == "" {
		return &ExtensionError{
			Extension: "unknown",
			Stage:     "validate",
			Err:       &testError{"name is required"},
		}
	}

	if manifest.Version == "" {
		return &ExtensionError{
			Extension: manifest.Name,
			Stage:     "validate",
			Err:       &testError{"version is required"},
		}
	}

	return nil
}
