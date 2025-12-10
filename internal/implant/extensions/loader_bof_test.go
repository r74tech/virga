package extensions

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
)

func TestBOFLoader_Load(t *testing.T) {
	loader := NewBOFLoader()

	manifest := &Manifest{
		Name:    "test-bof",
		Version: "1.0.0",
		Type:    "bof",
		Arguments: []ExtensionArgument{
			{
				Name:     "target",
				Type:     "string",
				Optional: false,
			},
		},
		Files: []ExtensionFile{
			{OS: "windows", Arch: "amd64", Path: "test.o"},
		},
	}

	// Create a minimal COFF file (header only for testing)
	// This is a very simplified COFF that won't actually execute
	var coffData bytes.Buffer
	header := COFFHeader{
		Machine:              IMAGE_FILE_MACHINE_AMD64,
		NumberOfSections:     0,
		TimeDateStamp:        0,
		PointerToSymbolTable: 0,
		NumberOfSymbols:      0,
		SizeOfOptionalHeader: 0,
		Characteristics:      0,
	}
	binary.Write(&coffData, binary.LittleEndian, header)

	bofPath := filepath.Join(t.TempDir(), "test.o")
	if err := os.WriteFile(bofPath, coffData.Bytes(), 0o644); err != nil {
		t.Fatalf("Failed to write test BOF file: %v", err)
	}

	ext, err := loader.Load(manifest, bofPath)
	if err != nil {
		t.Errorf("Load() failed: %v", err)
	}

	if ext == nil {
		t.Error("Load() returned nil extension")
	}

	bofExt, ok := ext.(*BOFExtension)
	if !ok {
		t.Error("Load() returned wrong type")
	}

	if bofExt.manifest.Name != manifest.Name {
		t.Errorf("Extension name = %q, want %q", bofExt.manifest.Name, manifest.Name)
	}
}

func TestBOFExtension_PackArguments(t *testing.T) {
	manifest := &Manifest{
		Name: "test-pack",
		Arguments: []ExtensionArgument{
			{
				Name: "str_arg",
				Type: "string",
			},
			{
				Name: "int_arg",
				Type: "int",
			},
			{
				Name: "short_arg",
				Type: "short",
			},
		},
	}

	ext := &BOFExtension{
		manifest: manifest,
	}

	args := map[string]interface{}{
		"str_arg":   "hello",
		"int_arg":   42,
		"short_arg": int16(100),
	}

	data, err := ext.packArguments(args)
	if err != nil {
		t.Fatalf("packArguments() failed: %v", err)
	}

	// Verify packed data structure
	buf := bytes.NewReader(data)

	// String argument: size(4) + "hello" + null
	var strSize uint32
	if err := binary.Read(buf, binary.LittleEndian, &strSize); err != nil {
		t.Fatalf("Failed to read string size: %v", err)
	}
	if strSize != 6 { // "hello" + null terminator
		t.Errorf("String size = %d, want 6", strSize)
	}

	strData := make([]byte, strSize)
	if _, err := buf.Read(strData); err != nil {
		t.Fatalf("Failed to read string data: %v", err)
	}
	if string(strData[:5]) != "hello" {
		t.Errorf("String data = %q, want 'hello'", strData[:5])
	}
	if strData[5] != 0 {
		t.Error("String not null terminated")
	}

	// Int argument: size(4) + value(4)
	var intSize uint32
	if err := binary.Read(buf, binary.LittleEndian, &intSize); err != nil {
		t.Fatalf("Failed to read int size: %v", err)
	}
	if intSize != 4 {
		t.Errorf("Int size = %d, want 4", intSize)
	}

	var intValue int32
	if err := binary.Read(buf, binary.LittleEndian, &intValue); err != nil {
		t.Fatalf("Failed to read int value: %v", err)
	}
	if intValue != 42 {
		t.Errorf("Int value = %d, want 42", intValue)
	}

	// Short argument: size(4) + value(2)
	var shortSize uint32
	if err := binary.Read(buf, binary.LittleEndian, &shortSize); err != nil {
		t.Fatalf("Failed to read short size: %v", err)
	}
	if shortSize != 2 {
		t.Errorf("Short size = %d, want 2", shortSize)
	}

	var shortValue int16
	if err := binary.Read(buf, binary.LittleEndian, &shortValue); err != nil {
		t.Fatalf("Failed to read short value: %v", err)
	}
	if shortValue != 100 {
		t.Errorf("Short value = %d, want 100", shortValue)
	}
}

func TestBOFExtension_PackArguments_Errors(t *testing.T) {
	manifest := &Manifest{
		Name: "test-errors",
		Arguments: []ExtensionArgument{
			{
				Name:     "required_arg",
				Type:     "string",
				Optional: false,
			},
			{
				Name: "unsupported_type",
				Type: "unsupported",
			},
		},
	}

	ext := &BOFExtension{
		manifest: manifest,
	}

	// Test missing required argument
	_, err := ext.packArguments(map[string]interface{}{})
	if err == nil {
		t.Error("packArguments() should fail with missing required argument")
	}

	// Test unsupported type
	args := map[string]interface{}{
		"required_arg":     "value",
		"unsupported_type": "test",
	}
	_, err = ext.packArguments(args)
	if err == nil {
		t.Error("packArguments() should fail with unsupported type")
	}
}

func TestBOFExtension_TypeConversion(t *testing.T) {
	manifest := &Manifest{
		Name: "test-conversion",
		Arguments: []ExtensionArgument{
			{
				Name: "int_arg",
				Type: "int",
			},
		},
	}

	ext := &BOFExtension{
		manifest: manifest,
	}

	// Test float64 to int conversion (JSON numbers come as float64)
	args := map[string]interface{}{
		"int_arg": float64(123),
	}

	data, err := ext.packArguments(args)
	if err != nil {
		t.Fatalf("packArguments() failed: %v", err)
	}

	// Verify conversion
	buf := bytes.NewReader(data)
	var size uint32
	if err := binary.Read(buf, binary.LittleEndian, &size); err != nil {
		t.Fatalf("Failed to read size: %v", err)
	}

	var value int32
	if err := binary.Read(buf, binary.LittleEndian, &value); err != nil {
		t.Fatalf("Failed to read value: %v", err)
	}
	if value != 123 {
		t.Errorf("Converted int value = %d, want 123", value)
	}
}

func TestBOFLoader_GetType(t *testing.T) {
	loader := NewBOFLoader()
	if loader.GetType() != ExtensionTypeBOF {
		t.Errorf("GetType() = %v, want %v", loader.GetType(), ExtensionTypeBOF)
	}
}

func TestBOFLoader_Unload(t *testing.T) {
	loader := NewBOFLoader()

	manifest := &Manifest{
		Name: "test-unload",
	}

	// Create a BOF extension
	ext := &BOFExtension{
		manifest: manifest,
	}

	// Store in loader
	loader.loadedBOFs[manifest.Name] = &loadedBOF{
		manifest: manifest,
	}

	// Unload
	err := loader.Unload(ext)
	if err != nil {
		t.Errorf("Unload() failed: %v", err)
	}

	// Verify removed
	if _, exists := loader.loadedBOFs[manifest.Name]; exists {
		t.Error("BOF still in loadedBOFs after unload")
	}
}

func TestCOFFStructures(t *testing.T) {
	// Test COFF header size
	var header COFFHeader
	size := binary.Size(header)
	if size != 20 { // COFF header is 20 bytes
		t.Errorf("COFF header size = %d, want 20", size)
	}

	// Test section header size
	var section COFFSectionHeader
	size = binary.Size(section)
	if size != 40 { // Section header is 40 bytes
		t.Errorf("Section header size = %d, want 40", size)
	}
}
