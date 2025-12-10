package extensions

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

// TestIntegration_FullExtensionFlow tests the complete extension flow
func TestIntegration_FullExtensionFlow(t *testing.T) {
	// Skip if Python is not available (needed for script extension test)
	if _, err := exec.LookPath("python3"); err != nil && runtime.GOOS != "windows" {
		t.Skip("Python3 not available")
	}
	if _, err := exec.LookPath("python"); err != nil && runtime.GOOS == "windows" {
		t.Skip("Python not available")
	}

	// Create extension manager
	callbacks := &ExtensionCallbacks{
		Log: func(level, message string, fields map[string]interface{}) {
			t.Logf("[%s] %s %v", level, message, fields)
		},
		SendOutput: func(output string) error {
			t.Logf("Output: %s", output)
			return nil
		},
		SendError: func(err error) error {
			t.Logf("Error: %v", err)
			return nil
		},
	}

	manager := NewManager(callbacks)

	// Create a Python extension
	pythonScript := `#!/usr/bin/env python3
import os
import json
import sys

# Get arguments
args_json = os.environ.get('EXTENSION_ARGS', '{}')
args = json.loads(args_json)

# Process
target = args.get('target', 'unknown')
port = args.get('port', 80)

print(f"Scanning {target}:{port}")
print(f"Extension: {os.environ.get('EXTENSION_NAME')} v{os.environ.get('EXTENSION_VERSION')}")

# Simulate some work
import time
time.sleep(0.1)

print("Scan complete!")
sys.exit(0)
`

	// Create manifest
	manifest := Manifest{
		Name:            "scanner",
		Version:         "1.0.0",
		Type:            "script",
		ExtensionAuthor: "Test Author",
		Help:            "Test scanner extension",
		Commands: []ExtensionCommand{
			{
				Name: "scan",
				Help: "Scan a target",
				Args: []ExtensionArgument{
					{
						Name:        "target",
						Type:        "string",
						Description: "Target to scan",
						Optional:    true,
					},
					{
						Name:        "port",
						Type:        "int",
						Description: "Port to scan",
						Optional:    true,
						Default:     "80",
					},
				},
			},
		},
		Files: []ExtensionFile{
			{
				OS:   "any",
				Arch: "any",
				Path: "scanner.py",
			},
		},
	}

	// Test 1: Load extension from payload
	t.Run("LoadFromPayload", func(t *testing.T) {
		manifestJSON, _ := json.Marshal(manifest)
		payload := map[string]interface{}{
			"manifest": string(manifestJSON),
			"data":     []byte(pythonScript),
		}

		err := manager.LoadExtensionFromPayload(payload)
		if err != nil {
			t.Fatalf("Failed to load extension: %v", err)
		}

		if !manager.IsLoaded("scanner") {
			t.Fatal("Extension not loaded")
		}
	})

	// Test 2: Execute extension
	t.Run("Execute", func(t *testing.T) {
		args := map[string]interface{}{
			"target": "192.168.1.1",
			"port":   443,
		}

		result, err := manager.ExecuteExtension("scanner", args)
		if err != nil {
			t.Fatalf("Failed to execute extension: %v", err)
		}

		if !result.Success {
			t.Errorf("Execution failed: %s", result.Error)
		}

		// Check output contains expected strings
		expectedStrings := []string{
			"Scanning 192.168.1.1:443",
			"Extension: scanner v1.0.0",
			"Scan complete!",
		}

		for _, expected := range expectedStrings {
			if !contains(result.Output, expected) {
				t.Errorf("Output missing expected string: %q\nFull output: %s", expected, result.Output)
			}
		}
	})

	// Test 3: List extensions
	t.Run("List", func(t *testing.T) {
		extensions := manager.GetLoadedExtensions()
		if len(extensions) != 1 {
			t.Errorf("Expected 1 extension, got %d", len(extensions))
		}

		found := false
		for _, name := range extensions {
			if name == "scanner" {
				found = true
				break
			}
		}

		if !found {
			t.Error("Scanner extension not in loaded list")
		}
	})

	// Test 4: Unload extension
	t.Run("Unload", func(t *testing.T) {
		err := manager.UnloadExtension("scanner")
		if err != nil {
			t.Fatalf("Failed to unload extension: %v", err)
		}

		if manager.IsLoaded("scanner") {
			t.Error("Extension still loaded after unload")
		}

		// Try to execute after unload
		_, err = manager.ExecuteExtension("scanner", nil)
		if err == nil {
			t.Error("Expected error executing unloaded extension")
		}
	})
}

// createTestCOFF creates a minimal valid COFF file for testing
func createTestCOFF() []byte {
	var buf bytes.Buffer

	// Write COFF header
	header := COFFHeader{
		Machine:              IMAGE_FILE_MACHINE_AMD64,
		NumberOfSections:     1,
		TimeDateStamp:        0,
		PointerToSymbolTable: 100, // After header and section
		NumberOfSymbols:      1,
		SizeOfOptionalHeader: 0,
		Characteristics:      0,
	}
	binary.Write(&buf, binary.LittleEndian, header)

	// Write section header
	sectionHeader := COFFSectionHeader{
		Name:                 [8]byte{'.', 't', 'e', 'x', 't', 0, 0, 0},
		VirtualSize:          10,
		VirtualAddress:       0,
		SizeOfRawData:        10,
		PointerToRawData:     60, // After headers
		PointerToRelocations: 0,
		PointerToLinenumbers: 0,
		NumberOfRelocations:  0,
		NumberOfLinenumbers:  0,
		Characteristics:      IMAGE_SCN_CNT_CODE | IMAGE_SCN_MEM_EXECUTE | IMAGE_SCN_MEM_READ,
	}
	binary.Write(&buf, binary.LittleEndian, sectionHeader)

	// Write section data
	sectionData := []byte{0x90, 0x90, 0x90, 0x90, 0x90, 0xC3, 0x00, 0x00, 0x00, 0x00} // NOPs and RET
	buf.Write(sectionData)

	// Align to symbol table
	for buf.Len() < 100 {
		buf.WriteByte(0)
	}

	// Write symbol
	symbol := COFFSymbol{
		Name:               [8]byte{'g', 'o', 0, 0, 0, 0, 0, 0},
		Value:              0,
		SectionNumber:      1,
		Type:               IMAGE_SYM_TYPE_FUNC,
		StorageClass:       IMAGE_SYM_CLASS_EXTERNAL,
		NumberOfAuxSymbols: 0,
	}
	binary.Write(&buf, binary.LittleEndian, symbol)

	// Write string table (just size for empty table)
	binary.Write(&buf, binary.LittleEndian, uint32(4))

	return buf.Bytes()
}

// TestIntegration_MultipleExtensions tests loading and executing multiple extensions
func TestIntegration_MultipleExtensions(t *testing.T) {
	callbacks := &ExtensionCallbacks{
		Log: func(level, message string, fields map[string]interface{}) {
			t.Logf("[%s] %s %v", level, message, fields)
		},
		ReadFile: func(path string) ([]byte, error) {
			// Return test COFF data for BOF extensions
			if filepath.Ext(path) == ".o" {
				return createTestCOFF(), nil
			}
			return os.ReadFile(path)
		},
	}

	manager := NewManager(callbacks)

	// Load multiple extensions of different types
	extensions := []struct {
		name     string
		extType  string
		manifest Manifest
	}{
		{
			name:    "alias-ext",
			extType: "alias",
			manifest: Manifest{
				Name:            "echo-alias",
				Version:         "1.0.0",
				Type:            "alias",
				Entrypoint:      "echo",
				Help:            "Echo command alias",
				ExtensionAuthor: "Test",
				Files: []ExtensionFile{
					{
						OS:   "any",
						Arch: "any",
						Path: "echo",
					},
				},
			},
		},
		{
			name:    "bof-ext",
			extType: "bof",
			manifest: Manifest{
				Name:            "test-bof",
				Version:         "1.0.0",
				Type:            "bof",
				Help:            "Test BOF extension",
				ExtensionAuthor: "Test",
				Files: []ExtensionFile{
					{
						OS:   "any",
						Arch: "any",
						Path: "test.o",
					},
				},
			},
		},
	}

	// Load all extensions
	for _, ext := range extensions {
		manifestJSON, _ := json.Marshal(ext.manifest)

		// Use proper data based on extension type
		var data []byte
		if ext.extType == "bof" {
			data = createTestCOFF()
		} else {
			data = []byte("test data")
		}

		payload := map[string]interface{}{
			"manifest": string(manifestJSON),
			"data":     data,
		}

		err := manager.LoadExtensionFromPayload(payload)
		if err != nil {
			t.Errorf("Failed to load %s: %v", ext.name, err)
		}
	}

	// Check all are loaded
	loaded := manager.GetLoadedExtensions()
	if len(loaded) != len(extensions) {
		t.Errorf("Expected %d extensions loaded, got %d", len(extensions), len(loaded))
	}

	// Execute each
	for _, ext := range extensions {
		result, err := manager.ExecuteExtension(ext.manifest.Name, nil)
		if err != nil {
			t.Errorf("Failed to execute %s: %v", ext.manifest.Name, err)
		}

		// All should succeed (even if just placeholder execution)
		if !result.Success {
			t.Errorf("Extension %s execution failed", ext.manifest.Name)
		}
	}
}

// TestIntegration_ExtensionLifecycle tests the full lifecycle of an extension
func TestIntegration_ExtensionLifecycle(t *testing.T) {
	tempDir := t.TempDir()

	// Create extension files
	manifest := Manifest{
		Name:            "lifecycle-test",
		Version:         "1.0.0",
		Type:            "script",
		ExtensionAuthor: "Test",
		Help:            "Lifecycle test extension",
		Files: []ExtensionFile{
			{
				OS:   runtime.GOOS,
				Arch: runtime.GOARCH,
				Path: "lifecycle.py",
			},
		},
	}

	// Write manifest
	manifestData, _ := json.MarshalIndent(manifest, "", "  ")
	manifestPath := filepath.Join(tempDir, "extension.json")
	os.WriteFile(manifestPath, manifestData, 0o644)

	// Write script
	script := `#!/usr/bin/env python3
print("Lifecycle test executed")
`
	scriptPath := filepath.Join(tempDir, "lifecycle.py")
	os.WriteFile(scriptPath, []byte(script), 0o755)

	// Create manager
	callbacks := &ExtensionCallbacks{
		Log: func(level, message string, fields map[string]interface{}) {
			t.Logf("[%s] %s %v", level, message, fields)
		},
	}

	manager := NewManager(callbacks)

	// Load from directory
	err := manager.LoadExtension(tempDir)
	if err != nil {
		t.Fatalf("Failed to load extension from directory: %v", err)
	}

	// Get extension
	ext, exists := manager.GetExtension("lifecycle-test")
	if !exists {
		t.Fatal("Extension not found after loading")
	}

	// Check manifest
	if ext.GetManifest().Name != "lifecycle-test" {
		t.Error("Extension manifest mismatch")
	}

	// Execute
	result, err := manager.ExecuteExtension("lifecycle-test", nil)
	if err != nil {
		t.Fatalf("Failed to execute: %v", err)
	}

	if !result.Success {
		t.Errorf("Execution failed: %s", result.Error)
	}

	// Unload
	err = manager.UnloadExtension("lifecycle-test")
	if err != nil {
		t.Fatalf("Failed to unload: %v", err)
	}
}

// Helper function
func contains(str, substr string) bool {
	return len(str) >= len(substr) &&
		(str == substr || len(str) > 0 && containsHelper(str, substr))
}

func containsHelper(str, substr string) bool {
	for i := 0; i <= len(str)-len(substr); i++ {
		if str[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
