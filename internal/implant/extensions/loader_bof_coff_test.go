package extensions

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func TestParseCOFF(t *testing.T) {
	// Create a simple COFF file with one section
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

	// Parse the COFF
	coff, err := ParseCOFF(buf.Bytes())
	if err != nil {
		t.Fatalf("ParseCOFF failed: %v", err)
	}

	// Verify header
	if coff.Header.Machine != IMAGE_FILE_MACHINE_AMD64 {
		t.Errorf("Machine = 0x%x, want 0x%x", coff.Header.Machine, IMAGE_FILE_MACHINE_AMD64)
	}
	if coff.Header.NumberOfSections != 1 {
		t.Errorf("NumberOfSections = %d, want 1", coff.Header.NumberOfSections)
	}

	// Verify section
	if len(coff.Sections) != 1 {
		t.Fatalf("Got %d sections, want 1", len(coff.Sections))
	}

	section := &coff.Sections[0]
	if coff.GetSectionName(section) != ".text" {
		t.Errorf("Section name = %q, want '.text'", coff.GetSectionName(section))
	}
	if len(section.Data) != 10 {
		t.Errorf("Section data length = %d, want 10", len(section.Data))
	}

	// Verify symbol
	if len(coff.Symbols) != 1 {
		t.Fatalf("Got %d symbols, want 1", len(coff.Symbols))
	}

	if coff.GetSymbolName(&coff.Symbols[0]) != "go" {
		t.Errorf("Symbol name = %q, want 'go'", coff.GetSymbolName(&coff.Symbols[0]))
	}
}

func TestParseCOFF_InvalidMachine(t *testing.T) {
	var buf bytes.Buffer

	// Write COFF header with invalid machine
	header := COFFHeader{
		Machine: 0xFFFF, // Invalid
	}
	binary.Write(&buf, binary.LittleEndian, header)

	_, err := ParseCOFF(buf.Bytes())
	if err == nil {
		t.Error("ParseCOFF should fail with invalid machine type")
	}
}

func TestGetSymbolName(t *testing.T) {
	coff := &COFFFile{
		StringTable: []byte("long_symbol_name\x00other_name\x00"),
	}

	tests := []struct {
		name     string
		symbol   COFFSymbol
		expected string
	}{
		{
			name: "Short name",
			symbol: COFFSymbol{
				Name: [8]byte{'s', 'h', 'o', 'r', 't', 0, 0, 0},
			},
			expected: "short",
		},
		{
			name: "String table reference",
			symbol: COFFSymbol{
				Name: [8]byte{0, 0, 0, 0, 4, 0, 0, 0}, // Offset 4 into string table (absolute)
			},
			expected: "long_symbol_name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			name := coff.GetSymbolName(&tt.symbol)
			if name != tt.expected {
				t.Errorf("GetSymbolName() = %q, want %q", name, tt.expected)
			}
		})
	}
}

func TestGetSectionName(t *testing.T) {
	coff := &COFFFile{
		StringTable: []byte(".long_section_name\x00.other\x00"),
	}

	tests := []struct {
		name     string
		section  COFFSection
		expected string
	}{
		{
			name: "Short name",
			section: COFFSection{
				Header: COFFSectionHeader{
					Name: [8]byte{'.', 't', 'e', 'x', 't', 0, 0, 0},
				},
			},
			expected: ".text",
		},
		{
			name: "String table reference",
			section: COFFSection{
				Header: COFFSectionHeader{
					Name: [8]byte{'/', '4', 0, 0, 0, 0, 0, 0}, // /4 means offset 4
				},
			},
			expected: ".long_section_name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			name := coff.GetSectionName(&tt.section)
			if name != tt.expected {
				t.Errorf("GetSectionName() = %q, want %q", name, tt.expected)
			}
		})
	}
}

func TestApplyRelocations(t *testing.T) {
	// Test AMD64 REL32 relocation
	sectionData := make([]byte, 8)
	sectionAddr := uintptr(0x1000)
	targetAddr := uintptr(0x2000)
	relocAddr := uint32(0)

	err := applyAMD64Relocation(IMAGE_REL_AMD64_REL32, sectionData, relocAddr, sectionAddr, targetAddr)
	if err != nil {
		t.Fatalf("applyAMD64Relocation failed: %v", err)
	}

	// REL32 calculation: target - (section + reloc + 4)
	// 0x2000 - (0x1000 + 0 + 4) = 0xFFC
	expected := int32(0xFFC)
	actual := int32(binary.LittleEndian.Uint32(sectionData[0:4]))

	if actual != expected {
		t.Errorf("REL32 relocation = 0x%x, want 0x%x", actual, expected)
	}
}

func TestLoadBOF_Simple(t *testing.T) {
	// Create a minimal COFF
	coff := &COFFFile{
		Header: COFFHeader{
			Machine:          IMAGE_FILE_MACHINE_AMD64,
			NumberOfSections: 1,
		},
		Sections: []COFFSection{
			{
				Header: COFFSectionHeader{
					Name:          [8]byte{'.', 't', 'e', 'x', 't'},
					VirtualSize:   4,
					SizeOfRawData: 4,
				},
				Data: []byte{0x90, 0x90, 0x90, 0xC3}, // NOP NOP NOP RET
			},
		},
		Symbols: []COFFSymbol{
			{
				Name:          [8]byte{'g', 'o'},
				Value:         0,
				SectionNumber: 1,
				StorageClass:  IMAGE_SYM_CLASS_EXTERNAL,
			},
		},
	}

	// Empty BOF API for testing
	bofAPI := make(map[string]uintptr)

	loaded, err := LoadBOF(coff, bofAPI)
	if err != nil {
		t.Fatalf("LoadBOF failed: %v", err)
	}

	if loaded.EntryPoint == 0 {
		t.Error("No entry point found")
	}

	if loaded.Size < 4 {
		t.Errorf("Loaded size = %d, want at least 4", loaded.Size)
	}

	if len(loaded.Sections) != 1 {
		t.Errorf("Got %d sections, want 1", len(loaded.Sections))
	}

	if _, ok := loaded.Symbols["go"]; !ok {
		t.Error("Symbol 'go' not found")
	}
}
