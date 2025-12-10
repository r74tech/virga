//go:build llama_selfextract && !darwin && unix
// +build llama_selfextract,!darwin,unix

// selfextract.go
// Self-extracting binary implementation for embedding large models (>2GB) in Go executables.
// The model is appended to the binary and loaded directly into memory at runtime.

package llama

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/sys/unix"
)

// LoadModelFromSelf reads the embedded model from the current executable using mmap.
// This approach avoids copying the entire model into memory, significantly reducing
// memory usage for large models.
func LoadModelFromSelf() ([]byte, error) {
	// Try mmap for Unix-like systems (Linux, BSD)
	// If mmap fails, fall back to streaming approach
	data, err := LoadModelFromSelfMmap()
	if err != nil {
		// Log the mmap failure and try streaming fallback
		fmt.Printf("mmap failed (%v), falling back to streaming mode\n", err)
		return LoadModelFromSelfStreaming()
	}
	return data, nil
}

// LoadModelFromSelfMmap uses memory mapping to access the embedded model without copying.
// This is the most memory-efficient approach for large models.
func LoadModelFromSelfMmap() ([]byte, error) {
	// Get the path to the current executable
	execPath, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("failed to get executable path: %w", err)
	}

	// Resolve any symlinks to get the actual file
	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve executable path: %w", err)
	}

	// Open the executable file for reading
	file, err := os.Open(execPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open executable: %w", err)
	}
	defer file.Close()

	// Get file size
	stat, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to stat executable: %w", err)
	}
	fileSize := stat.Size()

	// Check if file is large enough to contain metadata
	if fileSize < metadataSize {
		return nil, fmt.Errorf("executable too small to contain embedded model")
	}

	// Read the model size from the last 8 bytes (uint64, little-endian)
	metadataOffset := fileSize - metadataSize
	metadataBytes := make([]byte, metadataSize)

	_, err = file.ReadAt(metadataBytes, metadataOffset)
	if err != nil {
		return nil, fmt.Errorf("failed to read metadata: %w", err)
	}

	// Parse the model size
	modelSize := binary.LittleEndian.Uint64(metadataBytes)

	// Validate model size
	if modelSize == 0 {
		return nil, fmt.Errorf("no embedded model found (size is 0)")
	}

	// Calculate the total expected size (original binary + model + metadata)
	// The model starts at: fileSize - modelSize - metadataSize
	modelOffset := fileSize - int64(modelSize) - metadataSize

	if modelOffset < 0 {
		return nil, fmt.Errorf("invalid model size in metadata: %d (larger than file)", modelSize)
	}

	// Direct syscall mmap for zero-copy access
	fd := int(file.Fd())

	// Calculate page-aligned offset and adjustment
	pageSize := int64(os.Getpagesize())
	pageAlignedOffset := (modelOffset / pageSize) * pageSize
	adjustment := modelOffset - pageAlignedOffset
	mapSize := int(modelSize + uint64(adjustment))

	// Memory map the model portion of the file
	// Using PROT_READ for read-only access
	// Using MAP_PRIVATE for copy-on-write semantics
	prot := unix.PROT_READ
	flags := unix.MAP_PRIVATE

	mappedData, err := unix.Mmap(fd, pageAlignedOffset, mapSize, prot, flags)
	if err != nil {
		return nil, fmt.Errorf("failed to mmap model data: %w", err)
	}

	// Validate GGUF magic number before returning
	modelData := mappedData[int(adjustment) : int(adjustment)+int(modelSize)]
	if len(modelData) >= 4 {
		magic := modelData[0:4]
		fmt.Printf("[DEBUG] Selfextract mmap info:\n")
		fmt.Printf("  - File size: %d\n", fileSize)
		fmt.Printf("  - Model size: %d\n", modelSize)
		fmt.Printf("  - Model offset: %d\n", modelOffset)
		fmt.Printf("  - Page size: %d\n", pageSize)
		fmt.Printf("  - Page aligned offset: %d\n", pageAlignedOffset)
		fmt.Printf("  - Adjustment: %d\n", adjustment)
		fmt.Printf("  - Map size: %d\n", mapSize)
		fmt.Printf("  - Model magic bytes: %x (expected: 47475546 for 'GGUF')\n", magic)

		if string(magic) != "GGUF" {
			return nil, fmt.Errorf("invalid GGUF magic: got %x, expected 47475546 (GGUF)", magic)
		}
		fmt.Printf("  - ✓ GGUF magic validated successfully\n")
	}

	// Return a slice that points to the model data within the mapped region
	// The slice starts at the adjustment offset to skip the page alignment padding
	// and has exactly modelSize bytes
	return modelData, nil
}
