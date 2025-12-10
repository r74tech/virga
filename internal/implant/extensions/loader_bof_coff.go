package extensions

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"unsafe"
)

// COFF constants
const (
	IMAGE_FILE_MACHINE_AMD64 = 0x8664
	IMAGE_FILE_MACHINE_I386  = 0x014c

	// Section characteristics
	IMAGE_SCN_CNT_CODE               = 0x00000020
	IMAGE_SCN_CNT_INITIALIZED_DATA   = 0x00000040
	IMAGE_SCN_CNT_UNINITIALIZED_DATA = 0x00000080
	IMAGE_SCN_MEM_EXECUTE            = 0x20000000
	IMAGE_SCN_MEM_READ               = 0x40000000
	IMAGE_SCN_MEM_WRITE              = 0x80000000

	// Symbol types
	IMAGE_SYM_TYPE_NULL  = 0
	IMAGE_SYM_TYPE_FUNC  = 0x20
	IMAGE_SYM_DTYPE_NULL = 0

	// Symbol storage classes
	IMAGE_SYM_CLASS_EXTERNAL = 2
	IMAGE_SYM_CLASS_STATIC   = 3
	IMAGE_SYM_CLASS_LABEL    = 6

	// Relocation types for AMD64
	IMAGE_REL_AMD64_ADDR64   = 0x0001
	IMAGE_REL_AMD64_ADDR32   = 0x0002
	IMAGE_REL_AMD64_ADDR32NB = 0x0003
	IMAGE_REL_AMD64_REL32    = 0x0004
	IMAGE_REL_AMD64_REL32_1  = 0x0005
	IMAGE_REL_AMD64_REL32_2  = 0x0006
	IMAGE_REL_AMD64_REL32_3  = 0x0007
	IMAGE_REL_AMD64_REL32_4  = 0x0008
	IMAGE_REL_AMD64_REL32_5  = 0x0009

	// Relocation types for i386
	IMAGE_REL_I386_ABSOLUTE = 0x0000
	IMAGE_REL_I386_DIR32    = 0x0006
	IMAGE_REL_I386_DIR32NB  = 0x0007
	IMAGE_REL_I386_REL32    = 0x0014
)

// COFFFile represents a parsed COFF file
type COFFFile struct {
	Header      COFFHeader
	Sections    []COFFSection
	Symbols     []COFFSymbol
	StringTable []byte
}

// COFFHeader represents the COFF file header
type COFFHeader struct {
	Machine              uint16
	NumberOfSections     uint16
	TimeDateStamp        uint32
	PointerToSymbolTable uint32
	NumberOfSymbols      uint32
	SizeOfOptionalHeader uint16
	Characteristics      uint16
}

// COFFSection represents a COFF section
type COFFSection struct {
	Header      COFFSectionHeader
	Data        []byte
	Relocations []COFFRelocation
}

// COFFSectionHeader represents a COFF section header
type COFFSectionHeader struct {
	Name                 [8]byte
	VirtualSize          uint32
	VirtualAddress       uint32
	SizeOfRawData        uint32
	PointerToRawData     uint32
	PointerToRelocations uint32
	PointerToLinenumbers uint32
	NumberOfRelocations  uint16
	NumberOfLinenumbers  uint16
	Characteristics      uint32
}

// COFFSymbol represents a COFF symbol
type COFFSymbol struct {
	Name               [8]byte
	Value              uint32
	SectionNumber      int16
	Type               uint16
	StorageClass       uint8
	NumberOfAuxSymbols uint8
}

// COFFRelocation represents a COFF relocation
type COFFRelocation struct {
	VirtualAddress   uint32
	SymbolTableIndex uint32
	Type             uint16
}

// ParseCOFF parses a COFF file
func ParseCOFF(data []byte) (*COFFFile, error) {
	reader := bytes.NewReader(data)
	coff := &COFFFile{}

	// Read COFF header
	if err := binary.Read(reader, binary.LittleEndian, &coff.Header); err != nil {
		return nil, fmt.Errorf("failed to read COFF header: %w", err)
	}

	// Validate machine type
	if coff.Header.Machine != IMAGE_FILE_MACHINE_AMD64 && coff.Header.Machine != IMAGE_FILE_MACHINE_I386 {
		return nil, fmt.Errorf("unsupported machine type: 0x%x", coff.Header.Machine)
	}

	// Skip optional header if present
	if coff.Header.SizeOfOptionalHeader > 0 {
		if _, err := reader.Seek(int64(coff.Header.SizeOfOptionalHeader), io.SeekCurrent); err != nil {
			return nil, fmt.Errorf("failed to skip optional header: %w", err)
		}
	}

	// Read sections
	coff.Sections = make([]COFFSection, coff.Header.NumberOfSections)
	for i := range coff.Sections {
		if err := binary.Read(reader, binary.LittleEndian, &coff.Sections[i].Header); err != nil {
			return nil, fmt.Errorf("failed to read section header %d: %w", i, err)
		}
	}

	// Read section data and relocations
	for i := range coff.Sections {
		section := &coff.Sections[i]

		// Read section data
		if section.Header.SizeOfRawData > 0 {
			if _, err := reader.Seek(int64(section.Header.PointerToRawData), io.SeekStart); err != nil {
				return nil, fmt.Errorf("failed to seek to section %d data: %w", i, err)
			}
			section.Data = make([]byte, section.Header.SizeOfRawData)
			if _, err := reader.Read(section.Data); err != nil {
				return nil, fmt.Errorf("failed to read section %d data: %w", i, err)
			}
		}

		// Read relocations
		if section.Header.NumberOfRelocations > 0 {
			if _, err := reader.Seek(int64(section.Header.PointerToRelocations), io.SeekStart); err != nil {
				return nil, fmt.Errorf("failed to seek to section %d relocations: %w", i, err)
			}
			section.Relocations = make([]COFFRelocation, section.Header.NumberOfRelocations)
			for j := range section.Relocations {
				if err := binary.Read(reader, binary.LittleEndian, &section.Relocations[j]); err != nil {
					return nil, fmt.Errorf("failed to read relocation %d in section %d: %w", j, i, err)
				}
			}
		}
	}

	// Read symbol table
	if coff.Header.NumberOfSymbols > 0 {
		if _, err := reader.Seek(int64(coff.Header.PointerToSymbolTable), io.SeekStart); err != nil {
			return nil, fmt.Errorf("failed to seek to symbol table: %w", err)
		}
		coff.Symbols = make([]COFFSymbol, coff.Header.NumberOfSymbols)
		for i := range coff.Symbols {
			if err := binary.Read(reader, binary.LittleEndian, &coff.Symbols[i]); err != nil {
				return nil, fmt.Errorf("failed to read symbol %d: %w", i, err)
			}

			// Skip auxiliary symbols
			if coff.Symbols[i].NumberOfAuxSymbols > 0 {
				auxSize := int64(coff.Symbols[i].NumberOfAuxSymbols) * 18 // Size of symbol entry
				if _, err := reader.Seek(auxSize, io.SeekCurrent); err != nil {
					return nil, fmt.Errorf("failed to skip auxiliary symbols: %w", err)
				}
			}
		}

		// Read string table (immediately after symbol table)
		var stringTableSize uint32
		if err := binary.Read(reader, binary.LittleEndian, &stringTableSize); err != nil {
			return nil, fmt.Errorf("failed to read string table size: %w", err)
		}

		if stringTableSize > 4 {
			coff.StringTable = make([]byte, stringTableSize-4)
			if _, err := reader.Read(coff.StringTable); err != nil {
				return nil, fmt.Errorf("failed to read string table: %w", err)
			}
		}
	}

	return coff, nil
}

// GetSymbolName returns the name of a symbol
func (c *COFFFile) GetSymbolName(sym *COFFSymbol) string {
	// Check if name is in string table
	if sym.Name[0] == 0 && sym.Name[1] == 0 && sym.Name[2] == 0 && sym.Name[3] == 0 {
		// Name is in string table
		offset := binary.LittleEndian.Uint32(sym.Name[4:8])
		if offset >= 4 && int(offset-4) <= len(c.StringTable) {
			// Find null terminator
			end := bytes.IndexByte(c.StringTable[offset-4:], 0)
			if end == -1 {
				end = len(c.StringTable) - int(offset-4)
			}
			return string(c.StringTable[offset-4 : offset-4+uint32(end)])
		}
	}

	// Name is in the symbol entry
	nameBytes := sym.Name[:]
	if idx := bytes.IndexByte(nameBytes, 0); idx != -1 {
		return string(nameBytes[:idx])
	}
	return string(nameBytes)
}

// GetSectionName returns the name of a section
func (c *COFFFile) GetSectionName(section *COFFSection) string {
	nameBytes := section.Header.Name[:]

	// Check if name is in string table (starts with /)
	if nameBytes[0] == '/' {
		// Parse offset
		offsetStr := string(bytes.TrimRight(nameBytes[1:], "\x00"))
		var offset uint32
		if _, err := fmt.Sscanf(offsetStr, "%d", &offset); err != nil {
			return "" // Invalid offset format
		}

		if offset >= 4 && int(offset-4) <= len(c.StringTable) {
			// Find null terminator
			end := bytes.IndexByte(c.StringTable[offset-4:], 0)
			if end == -1 {
				end = len(c.StringTable) - int(offset-4)
			}
			return string(c.StringTable[offset-4 : offset-4+uint32(end)])
		}
	}

	// Name is in the section header
	if idx := bytes.IndexByte(nameBytes, 0); idx != -1 {
		return string(nameBytes[:idx])
	}
	return string(nameBytes)
}

// LoadedBOF represents a BOF loaded into memory
type LoadedBOF struct {
	BaseAddress uintptr
	Size        uint32
	Sections    []LoadedSection
	Symbols     map[string]uintptr
	EntryPoint  uintptr
}

// LoadedSection represents a loaded section in memory
type LoadedSection struct {
	Name        string
	Address     uintptr
	Size        uint32
	Permissions uint32
}

// LoadBOF loads a BOF into memory
func LoadBOF(coffFile *COFFFile, bofAPI map[string]uintptr) (*LoadedBOF, error) {
	loaded := &LoadedBOF{
		Symbols: make(map[string]uintptr),
	}

	// Calculate total size needed
	var totalSize uint32
	sectionOffsets := make(map[int]uint32)
	var currentOffset uint32

	for i, section := range coffFile.Sections {
		// Align to 16-byte boundary
		if currentOffset%16 != 0 {
			currentOffset = (currentOffset + 15) &^ 15
		}

		sectionOffsets[i] = currentOffset
		size := section.Header.VirtualSize
		if size == 0 {
			size = section.Header.SizeOfRawData
		}
		currentOffset += size
	}
	totalSize = currentOffset

	// Allocate memory for BOF
	// In a real implementation, this would use VirtualAlloc on Windows
	// For now, we'll use a simple byte slice
	memory := make([]byte, totalSize)
	loaded.BaseAddress = uintptr(unsafe.Pointer(&memory[0]))
	loaded.Size = totalSize

	// Load sections
	for i, section := range coffFile.Sections {
		offset := sectionOffsets[i]
		sectionAddr := loaded.BaseAddress + uintptr(offset)

		// Copy section data
		if len(section.Data) > 0 {
			copy(memory[offset:], section.Data)
		}

		// Record loaded section
		loaded.Sections = append(loaded.Sections, LoadedSection{
			Name:    coffFile.GetSectionName(&section),
			Address: sectionAddr,
			Size:    section.Header.VirtualSize,
		})

		// Apply relocations
		for _, reloc := range section.Relocations {
			if err := applyRelocation(coffFile, &section, &reloc, memory[offset:],
				sectionAddr, sectionOffsets, loaded.Symbols, bofAPI); err != nil {
				return nil, fmt.Errorf("failed to apply relocation: %w", err)
			}
		}
	}

	// Build symbol table
	for i := 0; i < len(coffFile.Symbols); i++ {
		sym := &coffFile.Symbols[i]
		if sym.StorageClass == IMAGE_SYM_CLASS_EXTERNAL || sym.StorageClass == IMAGE_SYM_CLASS_STATIC {
			name := coffFile.GetSymbolName(sym)

			if sym.SectionNumber > 0 && int(sym.SectionNumber) <= len(sectionOffsets) {
				// Symbol points to a section
				sectionIdx := int(sym.SectionNumber) - 1
				symbolAddr := loaded.BaseAddress + uintptr(sectionOffsets[sectionIdx]) + uintptr(sym.Value)
				loaded.Symbols[name] = symbolAddr

				// Check if this is the entry point
				if name == "go" || name == "_go" || name == "main" || name == "_main" {
					loaded.EntryPoint = symbolAddr
				}
			}
		}

		// Skip auxiliary symbols
		i += int(sym.NumberOfAuxSymbols)
	}

	return loaded, nil
}

// applyRelocation applies a single relocation
func applyRelocation(coffFile *COFFFile, _ *COFFSection, reloc *COFFRelocation,
	sectionData []byte, sectionAddr uintptr, sectionOffsets map[int]uint32,
	_ map[string]uintptr, bofAPI map[string]uintptr,
) error {
	// Get target symbol
	if reloc.SymbolTableIndex >= uint32(len(coffFile.Symbols)) {
		return fmt.Errorf("invalid symbol index: %d", reloc.SymbolTableIndex)
	}

	symbol := &coffFile.Symbols[reloc.SymbolTableIndex]
	symbolName := coffFile.GetSymbolName(symbol)

	// Resolve symbol address
	var targetAddr uintptr
	if symbol.SectionNumber > 0 {
		// Internal symbol
		sectionIdx := int(symbol.SectionNumber) - 1
		if sectionIdx >= len(sectionOffsets) {
			return fmt.Errorf("invalid section number: %d", symbol.SectionNumber)
		}
		targetAddr = uintptr(sectionOffsets[sectionIdx]) + uintptr(symbol.Value)
	} else {
		// External symbol - check BOF API
		if addr, ok := bofAPI[symbolName]; ok {
			targetAddr = addr
		} else {
			return fmt.Errorf("unresolved external symbol: %s", symbolName)
		}
	}

	// Apply relocation based on type
	relocAddr := reloc.VirtualAddress
	if relocAddr >= uint32(len(sectionData)) {
		return fmt.Errorf("relocation address out of bounds: %d", relocAddr)
	}

	switch coffFile.Header.Machine {
	case IMAGE_FILE_MACHINE_AMD64:
		return applyAMD64Relocation(reloc.Type, sectionData, relocAddr, sectionAddr, targetAddr)
	case IMAGE_FILE_MACHINE_I386:
		return applyI386Relocation(reloc.Type, sectionData, relocAddr, sectionAddr, targetAddr)
	default:
		return fmt.Errorf("unsupported machine type: 0x%x", coffFile.Header.Machine)
	}
}

// applyAMD64Relocation applies an AMD64 relocation
func applyAMD64Relocation(relocType uint16, sectionData []byte, relocAddr uint32,
	sectionAddr, targetAddr uintptr,
) error {
	switch relocType {
	case IMAGE_REL_AMD64_ADDR64:
		// 64-bit absolute address
		if relocAddr+8 > uint32(len(sectionData)) {
			return fmt.Errorf("relocation out of bounds")
		}
		binary.LittleEndian.PutUint64(sectionData[relocAddr:], uint64(targetAddr))

	case IMAGE_REL_AMD64_ADDR32:
		// 32-bit absolute address
		if relocAddr+4 > uint32(len(sectionData)) {
			return fmt.Errorf("relocation out of bounds")
		}
		binary.LittleEndian.PutUint32(sectionData[relocAddr:], uint32(targetAddr))

	case IMAGE_REL_AMD64_REL32:
		// 32-bit relative address
		if relocAddr+4 > uint32(len(sectionData)) {
			return fmt.Errorf("relocation out of bounds")
		}
		pcAddr := sectionAddr + uintptr(relocAddr) + 4 // PC points to next instruction
		offset := int32(targetAddr - pcAddr)
		binary.LittleEndian.PutUint32(sectionData[relocAddr:], uint32(offset))

	default:
		return fmt.Errorf("unsupported AMD64 relocation type: 0x%x", relocType)
	}

	return nil
}

// applyI386Relocation applies an i386 relocation
func applyI386Relocation(relocType uint16, sectionData []byte, relocAddr uint32,
	sectionAddr, targetAddr uintptr,
) error {
	switch relocType {
	case IMAGE_REL_I386_DIR32:
		// 32-bit absolute address
		if relocAddr+4 > uint32(len(sectionData)) {
			return fmt.Errorf("relocation out of bounds")
		}
		binary.LittleEndian.PutUint32(sectionData[relocAddr:], uint32(targetAddr))

	case IMAGE_REL_I386_REL32:
		// 32-bit relative address
		if relocAddr+4 > uint32(len(sectionData)) {
			return fmt.Errorf("relocation out of bounds")
		}
		pcAddr := sectionAddr + uintptr(relocAddr) + 4
		offset := int32(targetAddr - pcAddr)
		binary.LittleEndian.PutUint32(sectionData[relocAddr:], uint32(offset))

	default:
		return fmt.Errorf("unsupported i386 relocation type: 0x%x", relocType)
	}

	return nil
}
