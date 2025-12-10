package extensions

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

// BOFLoader loads and executes Beacon Object Files (BOFs)
type BOFLoader struct {
	// Store loaded BOFs for potential reuse
	loadedBOFs map[string]*loadedBOF
}

// loadedBOF represents a loaded BOF in memory
type loadedBOF struct {
	manifest *Manifest
	code     []byte
	data     []byte
	symbols  map[string]uint64
	loaded   *LoadedBOF
	context  *BOFContext
}

// NewBOFLoader creates a new BOF loader
func NewBOFLoader() *BOFLoader {
	return &BOFLoader{
		loadedBOFs: make(map[string]*loadedBOF),
	}
}

// Load loads a BOF extension
func (l *BOFLoader) Load(manifest *Manifest, path string) (Extension, error) {
	// This is a simplified implementation
	// A full implementation would need to:
	// 1. Parse COFF file format
	// 2. Handle relocations
	// 3. Resolve symbols
	// 4. Allocate executable memory
	// 5. Load sections into memory

	return &BOFExtension{
		manifest: manifest,
		loader:   l,
		path:     path,
	}, nil
}

// Unload unloads a BOF extension
func (l *BOFLoader) Unload(extension Extension) error {
	if bofExt, ok := extension.(*BOFExtension); ok {
		delete(l.loadedBOFs, bofExt.manifest.Name)
	}
	return nil
}

// GetType returns the type of extensions this loader handles
func (l *BOFLoader) GetType() ExtensionType {
	return ExtensionTypeBOF
}

// BOFExtension represents a BOF extension
type BOFExtension struct {
	manifest  *Manifest
	loader    *BOFLoader
	path      string
	callbacks *ExtensionCallbacks
	bof       *loadedBOF
}

// GetManifest returns the extension manifest
func (e *BOFExtension) GetManifest() *Manifest {
	return e.manifest
}

// Initialize initializes the BOF extension
func (e *BOFExtension) Initialize(callbacks *ExtensionCallbacks) error {
	e.callbacks = callbacks

	// If path is empty (test case), skip loading
	if e.path == "" {
		if callbacks != nil && callbacks.Log != nil {
			callbacks.Log("info", fmt.Sprintf("BOF extension '%s' initialized (no file)", e.manifest.Name), nil)
		}
		return nil
	}

	// Load the BOF file
	data, err := callbacks.ReadFile(e.path)
	if err != nil {
		return fmt.Errorf("failed to read BOF file: %w", err)
	}

	// Parse COFF file
	coffFile, err := ParseCOFF(data)
	if err != nil {
		return fmt.Errorf("failed to parse COFF: %w", err)
	}

	// Create BOF context
	ctx := &BOFContext{
		extension:   e,
		callbacks:   callbacks,
		output:      bytes.NewBuffer(nil),
		dataParsers: make(map[uintptr]*BOFDataParser),
	}

	// Create BOF API
	bofAPI := CreateBOFAPI(ctx)

	// Load BOF into memory
	loaded, err := LoadBOF(coffFile, bofAPI)
	if err != nil {
		return fmt.Errorf("failed to load BOF: %w", err)
	}

	e.bof = &loadedBOF{
		manifest: e.manifest,
		code:     data,
		data:     nil,
		symbols:  make(map[string]uint64),
		loaded:   loaded,
		context:  ctx,
	}

	// Convert loaded addresses to uint64 for compatibility
	for name, addr := range loaded.Symbols {
		e.bof.symbols[name] = uint64(addr)
	}

	if callbacks.Log != nil {
		callbacks.Log("info", fmt.Sprintf("BOF extension '%s' initialized", e.manifest.Name), nil)
	}

	return nil
}

// Execute executes the BOF
func (e *BOFExtension) Execute(args map[string]interface{}) (*ExtensionResult, error) {
	if e.bof == nil || e.bof.loaded == nil {
		return nil, fmt.Errorf("BOF not loaded")
	}

	// Convert arguments to BOF format
	argData, err := e.packArguments(args)
	if err != nil {
		return nil, fmt.Errorf("failed to pack arguments: %w", err)
	}

	// Clear previous output
	e.bof.context.output.Reset()

	// Execute the BOF
	err = ExecuteBOF(e.bof.loaded, argData, e.bof.context)
	if err != nil {
		return &ExtensionResult{
			Success:  false,
			Output:   fmt.Sprintf("BOF execution failed: %v", err),
			Error:    err.Error(),
			ExitCode: -1,
		}, nil
	}

	// Get output
	output := e.bof.context.output.String()

	return &ExtensionResult{
		Success:  true,
		Output:   output,
		ExitCode: 0,
	}, nil
}

// Cleanup cleans up the BOF extension
func (e *BOFExtension) Cleanup() error {
	// Free any allocated memory
	return nil
}

// packArguments packs arguments for BOF execution
func (e *BOFExtension) packArguments(args map[string]interface{}) ([]byte, error) {
	// BOF argument format:
	// [size][data][size][data]...

	var buf bytes.Buffer

	// Pack each argument according to its type
	for _, arg := range e.manifest.Arguments {
		value, exists := args[arg.Name]
		if !exists && !arg.Optional {
			return nil, fmt.Errorf("missing required argument: %s", arg.Name)
		}

		if !exists {
			// Use default value if available
			if arg.Default != "" {
				value = arg.Default
			} else {
				continue
			}
		}

		// Pack based on type
		switch arg.Type {
		case "string":
			str, ok := value.(string)
			if !ok {
				return nil, fmt.Errorf("argument %s must be string", arg.Name)
			}
			// Write size
			binary.Write(&buf, binary.LittleEndian, uint32(len(str)+1))
			// Write string with null terminator
			buf.WriteString(str)
			buf.WriteByte(0)

		case "int":
			var intVal int32
			switch v := value.(type) {
			case int:
				intVal = int32(v)
			case int32:
				intVal = v
			case float64:
				intVal = int32(v)
			default:
				return nil, fmt.Errorf("argument %s must be int", arg.Name)
			}
			binary.Write(&buf, binary.LittleEndian, uint32(4))
			binary.Write(&buf, binary.LittleEndian, intVal)

		case "short":
			var shortVal int16
			switch v := value.(type) {
			case int:
				shortVal = int16(v)
			case int16:
				shortVal = v
			case float64:
				shortVal = int16(v)
			default:
				return nil, fmt.Errorf("argument %s must be short", arg.Name)
			}
			binary.Write(&buf, binary.LittleEndian, uint32(2))
			binary.Write(&buf, binary.LittleEndian, shortVal)

		default:
			return nil, fmt.Errorf("unsupported argument type: %s", arg.Type)
		}
	}

	return buf.Bytes(), nil
}
