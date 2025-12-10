package extensions

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestScriptExtension_FromTestdata tests loading and executing script extensions from testdata
func TestScriptExtension_FromTestdata(t *testing.T) {
	// Check which interpreters are available
	interpreters := map[string]string{
		"python": "hello.py",
		"node":   "hello.js",
		"ruby":   "hello.rb",
	}

	for interpreter, filename := range interpreters {
		t.Run(interpreter, func(t *testing.T) {
			// Check if interpreter is available
			if _, err := exec.LookPath(interpreter); err != nil {
				t.Skipf("%s not available: %v", interpreter, err)
			}

			// Create manifest
			manifest := &Manifest{
				Name:            "test-" + interpreter,
				Version:         "1.0.0",
				Type:            "script",
				ExtensionAuthor: "Test",
				Help:            "Test " + interpreter + " extension",
				Arguments: []ExtensionArgument{
					{
						Name:        "name",
						Type:        "string",
						Description: "Name to greet",
						Optional:    true,
						Default:     "World",
					},
				},
			}

			// Create loader
			loader := NewScriptLoader()

			// Load extension
			scriptPath := filepath.Join("testdata", "script", filename)
			ext, err := loader.Load(manifest, scriptPath)
			if err != nil {
				t.Fatalf("Failed to load %s extension: %v", interpreter, err)
			}

			// Test with default arguments
			t.Run("default_args", func(t *testing.T) {
				result, err := ext.Execute(map[string]interface{}{})
				if err != nil {
					t.Fatalf("Failed to execute: %v", err)
				}

				if !result.Success {
					t.Errorf("Extension failed: %s", result.Error)
				}

				if !strings.Contains(result.Output, "Hello, World!") {
					t.Errorf("Unexpected output: %s", result.Output)
				}

				// Check that version info is included
				if result.Data != nil {
					t.Logf("Extension data: %v", result.Data)
				}
			})

			// Test with custom arguments
			t.Run("custom_args", func(t *testing.T) {
				result, err := ext.Execute(map[string]interface{}{
					"name": "Virga",
				})
				if err != nil {
					t.Fatalf("Failed to execute: %v", err)
				}

				if !result.Success {
					t.Errorf("Extension failed: %s", result.Error)
				}

				if !strings.Contains(result.Output, "Hello, Virga") {
					t.Errorf("Unexpected output: %s", result.Output)
				}
			})

			// Cleanup
			if err := ext.Cleanup(); err != nil {
				t.Errorf("Cleanup failed: %v", err)
			}
		})
	}
}

// createTestCOFFData creates a minimal valid COFF file for testing
func createTestCOFFData() []byte {
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

// TestBOFExtension_FromTestdata tests BOF extension with generated COFF data
func TestBOFExtension_FromTestdata(t *testing.T) {
	// Create test COFF data
	coffData := createTestCOFFData()

	// Write to temporary file
	tmpDir := t.TempDir()
	bofPath := filepath.Join(tmpDir, "test.o")
	if err := os.WriteFile(bofPath, coffData, 0o644); err != nil {
		t.Fatalf("Failed to write BOF file: %v", err)
	}

	// Create manifest
	manifest := &Manifest{
		Name:            "test-bof",
		Version:         "1.0.0",
		Type:            "bof",
		ExtensionAuthor: "Test",
		Help:            "Test BOF extension",
		Commands: []ExtensionCommand{
			{
				Name: "go",
				Help: "Execute BOF",
			},
		},
	}

	// Create loader
	loader := NewBOFLoader()

	// Load extension
	ext, err := loader.Load(manifest, bofPath)
	if err != nil {
		t.Fatalf("Failed to load BOF: %v", err)
	}

	// Execute (note: this won't actually run the BOF code, just test the loading)
	result, err := ext.Execute(map[string]interface{}{})
	if err != nil {
		// This is expected since we're not actually executing real BOF code
		t.Logf("Expected execution error: %v", err)
	} else {
		t.Logf("BOF execution result: %+v", result)
	}

	// Cleanup
	if err := ext.Cleanup(); err != nil {
		t.Errorf("Cleanup failed: %v", err)
	}
}

// TestAliasExtension_FromTestdata tests alias extension
func TestAliasExtension_FromTestdata(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping shell script test on Windows")
	}

	// Make echo wrapper executable
	echoPath := filepath.Join("testdata", "alias", "echo_wrapper.sh")
	if err := os.Chmod(echoPath, 0o755); err != nil {
		t.Skipf("Cannot make echo wrapper executable: %v", err)
	}

	// Get absolute path
	absPath, err := filepath.Abs(echoPath)
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}

	// Create manifest
	manifest := &Manifest{
		Name:            "test-echo",
		Version:         "1.0.0",
		Type:            "alias",
		ExtensionAuthor: "Test",
		Help:            "Test echo alias",
		Entrypoint:      absPath,
		Arguments: []ExtensionArgument{
			{
				Name:        "message",
				Type:        "string",
				Description: "Message to echo",
				Optional:    false,
			},
		},
	}

	// Create loader
	loader := NewAliasLoader()

	// Load extension
	ext, err := loader.Load(manifest, "")
	if err != nil {
		t.Fatalf("Failed to load alias: %v", err)
	}

	// Execute
	result, err := ext.Execute(map[string]interface{}{
		"message": "Hello from alias!",
	})
	if err != nil {
		t.Fatalf("Failed to execute alias: %v", err)
	}

	if !result.Success {
		t.Errorf("Alias failed: %s", result.Error)
	}

	if !strings.Contains(result.Output, "Hello from alias!") {
		t.Errorf("Unexpected output: %s", result.Output)
	}

	// Cleanup
	if err := ext.Cleanup(); err != nil {
		t.Errorf("Cleanup failed: %v", err)
	}
}

// TestExtensionManager_LoadFromTestdata tests the extension manager with testdata
func TestExtensionManager_LoadFromTestdata(t *testing.T) {
	// Set up callbacks
	callbacks := &ExtensionCallbacks{
		Log: func(level, message string, fields map[string]interface{}) {
			t.Logf("[%s] %s %v", level, message, fields)
		},
		ReadFile: func(path string) ([]byte, error) {
			return os.ReadFile(path)
		},
	}

	// Create manager
	manager := NewManager(callbacks)

	// Test loading Python extension if available
	if _, err := exec.LookPath("python"); err == nil {
		t.Run("LoadPythonScript", func(t *testing.T) {
			// Create a test directory with manifest and script
			tmpDir := t.TempDir()

			// Copy script
			scriptData, err := os.ReadFile(filepath.Join("testdata", "script", "hello.py"))
			if err != nil {
				t.Fatalf("Failed to read script: %v", err)
			}
			if err := os.WriteFile(filepath.Join(tmpDir, "hello.py"), scriptData, 0o755); err != nil {
				t.Fatalf("Failed to write script: %v", err)
			}

			// Create manifest
			manifest := &Manifest{
				Name:            "test-python-manager",
				Version:         "1.0.0",
				Type:            "script",
				ExtensionAuthor: "Test",
				Help:            "Test Python via manager",
				Files: []ExtensionFile{
					{
						OS:   "any",
						Arch: "any",
						Path: "hello.py",
					},
				},
			}

			manifestData, _ := json.Marshal(manifest)
			if err := os.WriteFile(filepath.Join(tmpDir, "manifest.json"), manifestData, 0o644); err != nil {
				t.Fatalf("Failed to write manifest: %v", err)
			}

			// Load extension directory
			if err := manager.LoadExtension(tmpDir); err != nil {
				t.Fatalf("Failed to load Python extension: %v", err)
			}

			// Execute
			result, err := manager.ExecuteExtension("test-python-manager", map[string]interface{}{
				"name": "Manager Test",
			})
			if err != nil {
				t.Fatalf("Failed to execute: %v", err)
			}

			if !result.Success {
				t.Errorf("Extension failed: %s", result.Error)
			}

			if !strings.Contains(result.Output, "Manager Test") {
				t.Errorf("Unexpected output: %s", result.Output)
			}

			// List extensions
			loaded := manager.GetLoadedExtensions()
			found := false
			for _, extName := range loaded {
				if extName == "test-python-manager" {
					found = true
					break
				}
			}
			if !found {
				t.Error("Extension not found in loaded list")
			}

			// Unload
			if err := manager.UnloadExtension("test-python-manager"); err != nil {
				t.Errorf("Failed to unload: %v", err)
			}
		})
	}
}

// TestExtensionPayload_WithTestdata tests loading extensions from payload format
func TestExtensionPayload_WithTestdata(t *testing.T) {
	// Only test if Python is available
	if _, err := exec.LookPath("python"); err != nil {
		t.Skip("Python not available")
	}

	// Read the test script
	scriptData, err := os.ReadFile(filepath.Join("testdata", "script", "hello.py"))
	if err != nil {
		t.Fatalf("Failed to read test script: %v", err)
	}

	// Create manifest
	manifest := &Manifest{
		Name:            "payload-test",
		Version:         "1.0.0",
		Type:            "script",
		ExtensionAuthor: "Test",
		Help:            "Test from payload",
		Files: []ExtensionFile{
			{
				OS:   "any",
				Arch: "any",
				Path: "hello.py",
			},
		},
	}

	// Create payload
	manifestJSON, _ := json.Marshal(manifest)
	payload := map[string]interface{}{
		"manifest": string(manifestJSON),
		"data":     scriptData,
	}

	// Set up manager
	callbacks := &ExtensionCallbacks{
		Log: func(level, message string, fields map[string]interface{}) {
			t.Logf("[%s] %s %v", level, message, fields)
		},
		ReadFile: func(path string) ([]byte, error) {
			return os.ReadFile(path)
		},
	}

	manager := NewManager(callbacks)

	// Load from payload
	if err := manager.LoadExtensionFromPayload(payload); err != nil {
		t.Fatalf("Failed to load from payload: %v", err)
	}

	// Execute
	result, err := manager.ExecuteExtension("payload-test", map[string]interface{}{
		"name": "Payload",
	})
	if err != nil {
		t.Fatalf("Failed to execute: %v", err)
	}

	if !result.Success {
		t.Errorf("Extension failed: %s", result.Error)
	}

	if !strings.Contains(result.Output, "Hello, Payload!") {
		t.Errorf("Unexpected output: %s", result.Output)
	}

	// Cleanup
	if err := manager.UnloadExtension("payload-test"); err != nil {
		t.Errorf("Failed to unload: %v", err)
	}
}

// Benchmark tests
func BenchmarkScriptExtension_Execute(b *testing.B) {
	// Only benchmark if Python is available
	if _, err := exec.LookPath("python"); err != nil {
		b.Skip("Python not available")
	}

	// Set up extension
	manifest := &Manifest{
		Name:            "bench-python",
		Version:         "1.0.0",
		Type:            "script",
		ExtensionAuthor: "Benchmark",
		Help:            "Benchmark Python extension",
	}

	loader := NewScriptLoader()
	scriptPath := filepath.Join("testdata", "script", "hello.py")
	ext, err := loader.Load(manifest, scriptPath)
	if err != nil {
		b.Fatalf("Failed to load: %v", err)
	}
	defer ext.Cleanup()

	// Reset timer after setup
	b.ResetTimer()

	// Run benchmark
	for i := 0; i < b.N; i++ {
		result, err := ext.Execute(map[string]interface{}{
			"name": "Benchmark",
		})
		if err != nil {
			b.Fatalf("Execute failed: %v", err)
		}
		if !result.Success {
			b.Fatalf("Extension failed: %s", result.Error)
		}
	}
}

// Example test showing usage
func ExampleManager_LoadExtension() {
	// Create callbacks
	callbacks := &ExtensionCallbacks{
		Log: func(level, message string, fields map[string]interface{}) {
			// Log to your logging system
		},
		ReadFile: func(path string) ([]byte, error) {
			return os.ReadFile(path)
		},
	}

	// Create manager
	manager := NewManager(callbacks)

	// Load extension from directory
	_ = manager.LoadExtension("path/to/extension/dir")

	// Execute extension
	result, _ := manager.ExecuteExtension("example-ext", map[string]interface{}{
		"arg1": "value1",
	})

	// Check result
	if result.Success {
		// Handle success
		_ = result.Output
	}
}
