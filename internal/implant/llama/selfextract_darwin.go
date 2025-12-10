//go:build llama_selfextract && darwin
// +build llama_selfextract,darwin

package llama

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// LoadModelFromSelf loads embedded model using memory loading on macOS
// This avoids llama.cpp mmap issues on macOS by reading directly into memory
func LoadModelFromSelf() ([]byte, error) {
	execPath, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("failed to get executable path: %w", err)
	}

	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve executable path: %w", err)
	}

	file, err := os.Open(execPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open executable: %w", err)
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to stat executable: %w", err)
	}
	fileSize := stat.Size()

	if fileSize < metadataSize {
		return nil, fmt.Errorf("executable too small to contain embedded model")
	}

	// Read metadata (8 bytes for model size)
	metadataBytes := make([]byte, metadataSize)
	_, err = file.ReadAt(metadataBytes, fileSize-metadataSize)
	if err != nil {
		return nil, fmt.Errorf("failed to read metadata: %w", err)
	}

	modelSize := binary.LittleEndian.Uint64(metadataBytes)
	if modelSize == 0 {
		return nil, fmt.Errorf("no embedded model found (size is 0)")
	}

	modelOffset := fileSize - int64(modelSize) - metadataSize
	if modelOffset < 0 {
		return nil, fmt.Errorf("invalid model size in metadata: %d (larger than file)", modelSize)
	}

	// macOS: Use memory loading instead of mmap to avoid llama.cpp issues
	fmt.Printf("[Llama] macOS detected, using memory loading instead of mmap\n")
	fmt.Printf("[Llama] Loading model: %.2f MB from offset %d\n",
		float64(modelSize)/(1024*1024), modelOffset)

	// Seek to model start
	_, err = file.Seek(modelOffset, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to seek to model start: %w", err)
	}

	// Read model data directly into memory
	modelData := make([]byte, modelSize)
	n, err := io.ReadFull(file, modelData)
	if err != nil {
		return nil, fmt.Errorf("failed to read model data: %w", err)
	}
	if n != int(modelSize) {
		return nil, fmt.Errorf("incomplete read: expected %d bytes, got %d", modelSize, n)
	}

	// Verify GGUF header
	if len(modelData) >= 4 {
		if modelData[0] == 'G' && modelData[1] == 'G' && modelData[2] == 'U' && modelData[3] == 'F' {
			fmt.Printf("[Llama] ✓ Valid GGUF header found\n")
		} else {
			return nil, fmt.Errorf("invalid GGUF magic: got %02x %02x %02x %02x",
				modelData[0], modelData[1], modelData[2], modelData[3])
		}
	}

	fmt.Printf("[Llama] ✓ Successfully loaded self-contained model into memory (%.2f MB)\n",
		float64(modelSize)/(1024*1024))

	return modelData, nil
}
