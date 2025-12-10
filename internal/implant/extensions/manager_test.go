package extensions

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestManager_RegisterLoader(t *testing.T) {
	callbacks := &ExtensionCallbacks{}
	manager := NewManager(callbacks)

	// Check default loaders are registered
	expectedTypes := []ExtensionType{
		ExtensionTypeNative,
		ExtensionTypeBOF,
		ExtensionTypeScript,
		ExtensionTypeAlias,
	}

	for _, extType := range expectedTypes {
		if _, exists := manager.loaders[extType]; !exists {
			t.Errorf("Default loader for %s not registered", extType)
		}
	}

	// Register a custom loader
	customLoader := &mockLoader{extType: ExtensionType("custom")}
	manager.RegisterLoader(customLoader)

	if _, exists := manager.loaders[ExtensionType("custom")]; !exists {
		t.Error("Custom loader not registered")
	}
}

func TestManager_LoadExtension(t *testing.T) {
	callbacks := &ExtensionCallbacks{
		Log: func(level, message string, fields map[string]interface{}) {
			t.Logf("[%s] %s %v", level, message, fields)
		},
	}
	manager := NewManager(callbacks)

	// Create test extension directory
	tempDir := t.TempDir()

	// Create manifest
	manifest := Manifest{
		Name:    "test-extension",
		Version: "1.0.0",
		Type:    "script",
		Files: []ExtensionFile{
			{
				OS:   runtime.GOOS,
				Arch: runtime.GOARCH,
				Path: "test.py",
			},
		},
	}

	manifestData, _ := json.MarshalIndent(manifest, "", "  ")
	manifestPath := filepath.Join(tempDir, "extension.json")
	os.WriteFile(manifestPath, manifestData, 0o644)

	// Create script file
	scriptPath := filepath.Join(tempDir, "test.py")
	os.WriteFile(scriptPath, []byte("print('test')"), 0o644)

	// Test loading
	err := manager.LoadExtension(tempDir)
	if err != nil {
		t.Errorf("LoadExtension() failed: %v", err)
	}

	// Check if loaded
	if !manager.IsLoaded(manifest.Name) {
		t.Error("Extension not loaded")
	}

	// Test loading again (should fail)
	err = manager.LoadExtension(tempDir)
	if err == nil {
		t.Error("LoadExtension() should fail when loading duplicate")
	}
}

func TestManager_LoadExtensionFromPayload(t *testing.T) {
	callbacks := &ExtensionCallbacks{
		Log: func(level, message string, fields map[string]interface{}) {
			t.Logf("[%s] %s %v", level, message, fields)
		},
	}
	manager := NewManager(callbacks)

	// Create test manifest
	manifest := Manifest{
		Name:    "payload-extension",
		Version: "1.0.0",
		Type:    "script",
	}

	manifestJSON, _ := json.Marshal(manifest)

	// Test with valid payload
	// For the test, we'll make the data field a base64 encoded string
	// since that's what the implementation expects
	testData := []byte("test data")
	payload := map[string]interface{}{
		"manifest": string(manifestJSON),
		"data":     testData,
	}

	err := manager.LoadExtensionFromPayload(payload)
	if err != nil {
		t.Logf("LoadExtensionFromPayload() with byte data failed: %v", err)

		// Try with base64 string
		payload["data"] = base64.StdEncoding.EncodeToString(testData)
		err = manager.LoadExtensionFromPayload(payload)
	}

	if err != nil {
		t.Errorf("LoadExtensionFromPayload() failed: %v", err)
	}

	// Check if loaded
	if !manager.IsLoaded(manifest.Name) {
		t.Error("Extension not loaded from payload")
	}

	// Test with invalid payload
	invalidPayload := "not a map"
	err = manager.LoadExtensionFromPayload(invalidPayload)
	if err == nil {
		t.Error("LoadExtensionFromPayload() should fail with invalid payload")
	}

	// Test with missing manifest
	missingManifest := map[string]interface{}{
		"data": []byte("test data"),
	}
	err = manager.LoadExtensionFromPayload(missingManifest)
	if err == nil {
		t.Error("LoadExtensionFromPayload() should fail without manifest")
	}
}

func TestManager_ExecuteExtension(t *testing.T) {
	callbacks := &ExtensionCallbacks{
		Log: func(level, message string, fields map[string]interface{}) {
			t.Logf("[%s] %s %v", level, message, fields)
		},
	}
	manager := NewManager(callbacks)

	// Register mock extension
	mockExt := &mockExtension{
		manifest: &Manifest{
			Name:    "mock-extension",
			Version: "1.0.0",
		},
		executeFunc: func(args map[string]interface{}) (*ExtensionResult, error) {
			return &ExtensionResult{
				Success:   true,
				Output:    "Mock output",
				ExitCode:  0,
				Timestamp: time.Now(),
			}, nil
		},
	}

	manager.mu.Lock()
	manager.extensions["mock-extension"] = mockExt
	manager.mu.Unlock()

	// Test execution
	args := map[string]interface{}{
		"test": "value",
	}

	result, err := manager.ExecuteExtension("mock-extension", args)
	if err != nil {
		t.Errorf("ExecuteExtension() failed: %v", err)
	}

	if !result.Success {
		t.Error("ExecuteExtension() result Success = false, want true")
	}

	if result.Output != "Mock output" {
		t.Errorf("ExecuteExtension() result Output = %q, want 'Mock output'", result.Output)
	}

	// Test non-existent extension
	_, err = manager.ExecuteExtension("non-existent", args)
	if err == nil {
		t.Error("ExecuteExtension() should fail for non-existent extension")
	}
}

func TestManager_UnloadExtension(t *testing.T) {
	callbacks := &ExtensionCallbacks{
		Log: func(level, message string, fields map[string]interface{}) {
			t.Logf("[%s] %s %v", level, message, fields)
		},
	}
	manager := NewManager(callbacks)

	// Register mock extension
	mockExt := &mockExtension{
		manifest: &Manifest{
			Name:    "unload-test",
			Version: "1.0.0",
		},
	}

	manager.mu.Lock()
	manager.extensions["unload-test"] = mockExt
	manager.mu.Unlock()

	// Test unload
	err := manager.UnloadExtension("unload-test")
	if err != nil {
		t.Errorf("UnloadExtension() failed: %v", err)
	}

	// Check if unloaded
	if manager.IsLoaded("unload-test") {
		t.Error("Extension still loaded after unload")
	}

	// Check cleanup was called
	if !mockExt.cleanupCalled {
		t.Error("Extension Cleanup() not called")
	}

	// Test unloading non-existent
	err = manager.UnloadExtension("non-existent")
	if err == nil {
		t.Error("UnloadExtension() should fail for non-existent extension")
	}
}

func TestManager_GetLoadedExtensions(t *testing.T) {
	callbacks := &ExtensionCallbacks{}
	manager := NewManager(callbacks)

	// Initially empty
	extensions := manager.GetLoadedExtensions()
	if len(extensions) != 0 {
		t.Errorf("GetLoadedExtensions() returned %d extensions, want 0", len(extensions))
	}

	// Add some extensions
	manager.mu.Lock()
	manager.extensions["ext1"] = &mockExtension{}
	manager.extensions["ext2"] = &mockExtension{}
	manager.mu.Unlock()

	extensions = manager.GetLoadedExtensions()
	if len(extensions) != 2 {
		t.Errorf("GetLoadedExtensions() returned %d extensions, want 2", len(extensions))
	}

	// Check names are present
	nameMap := make(map[string]bool)
	for _, name := range extensions {
		nameMap[name] = true
	}

	if !nameMap["ext1"] || !nameMap["ext2"] {
		t.Error("GetLoadedExtensions() missing expected extension names")
	}
}

func TestManager_GetExtension(t *testing.T) {
	callbacks := &ExtensionCallbacks{}
	manager := NewManager(callbacks)

	// Add test extension
	testExt := &mockExtension{
		manifest: &Manifest{
			Name: "get-test",
		},
	}

	manager.mu.Lock()
	manager.extensions["get-test"] = testExt
	manager.mu.Unlock()

	// Test getting existing
	ext, exists := manager.GetExtension("get-test")
	if !exists {
		t.Error("GetExtension() exists = false for existing extension")
	}

	if ext != testExt {
		t.Error("GetExtension() returned different extension")
	}

	// Test getting non-existent
	_, exists = manager.GetExtension("non-existent")
	if exists {
		t.Error("GetExtension() exists = true for non-existent extension")
	}
}

func TestManager_GetExtensionTypeFromManifest(t *testing.T) {
	manager := &Manager{}

	tests := []struct {
		name     string
		manifest *Manifest
		want     ExtensionType
	}{
		{
			name: "Explicit type",
			manifest: &Manifest{
				Type: "bof",
			},
			want: ExtensionTypeBOF,
		},
		{
			name: "From file extension",
			manifest: &Manifest{
				Files: []ExtensionFile{
					{
						OS:   runtime.GOOS,
						Arch: runtime.GOARCH,
						Path: "test.py",
					},
				},
			},
			want: ExtensionTypeScript,
		},
		{
			name: "Default to alias",
			manifest: &Manifest{
				Name: "no-type",
			},
			want: ExtensionTypeAlias,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := manager.getExtensionTypeFromManifest(tt.manifest)
			if got != tt.want {
				t.Errorf("getExtensionTypeFromManifest() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Mock types for testing

type mockLoader struct {
	extType ExtensionType
}

func (l *mockLoader) Load(manifest *Manifest, path string) (Extension, error) {
	return &mockExtension{manifest: manifest}, nil
}

func (l *mockLoader) Unload(extension Extension) error {
	return nil
}

func (l *mockLoader) GetType() ExtensionType {
	return l.extType
}

type mockExtension struct {
	manifest      *Manifest
	executeFunc   func(args map[string]interface{}) (*ExtensionResult, error)
	cleanupCalled bool
}

func (e *mockExtension) GetManifest() *Manifest {
	return e.manifest
}

func (e *mockExtension) Initialize(callbacks *ExtensionCallbacks) error {
	return nil
}

func (e *mockExtension) Execute(args map[string]interface{}) (*ExtensionResult, error) {
	if e.executeFunc != nil {
		return e.executeFunc(args)
	}
	return &ExtensionResult{
		Success:  true,
		Output:   "mock output",
		ExitCode: 0,
	}, nil
}

func (e *mockExtension) Cleanup() error {
	e.cleanupCalled = true
	return nil
}
