package llama

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

const (
	// metadataSize is the size of the metadata footer (uint64 for model size)
	metadataSize = 8
	// bufferSize for reading operations (64KB)
	bufferSize = 64 * 1024
)

// HasEmbeddedModel checks if the current executable has an embedded model
// without loading the entire model into memory.
func HasEmbeddedModel() (bool, uint64, error) {
	execPath, err := os.Executable()
	if err != nil {
		return false, 0, fmt.Errorf("failed to get executable path: %w", err)
	}

	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		return false, 0, fmt.Errorf("failed to resolve executable path: %w", err)
	}

	file, err := os.Open(execPath)
	if err != nil {
		return false, 0, fmt.Errorf("failed to open executable: %w", err)
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return false, 0, fmt.Errorf("failed to stat executable: %w", err)
	}

	fileSize := stat.Size()
	if fileSize < metadataSize {
		return false, 0, nil
	}

	// Read metadata
	metadataBytes := make([]byte, metadataSize)
	_, err = file.ReadAt(metadataBytes, fileSize-metadataSize)
	if err != nil {
		return false, 0, fmt.Errorf("failed to read metadata: %w", err)
	}

	modelSize := binary.LittleEndian.Uint64(metadataBytes)
	if modelSize == 0 {
		return false, 0, nil
	}

	// Validate that the model size makes sense
	modelOffset := fileSize - int64(modelSize) - metadataSize
	if modelOffset < 0 {
		return false, 0, nil
	}

	return true, modelSize, nil
}

// ModelInfo provides information about the embedded model
// without loading it into memory.
type ModelInfo struct {
	HasModel    bool
	ModelSize   uint64
	FileSize    int64
	ModelOffset int64
}

// GetEmbeddedModelInfo returns information about the embedded model.
func GetEmbeddedModelInfo() (*ModelInfo, error) {
	info := &ModelInfo{}

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

	info.FileSize = stat.Size()

	if info.FileSize < metadataSize {
		return info, nil
	}

	// Read metadata
	metadataBytes := make([]byte, metadataSize)
	_, err = file.ReadAt(metadataBytes, info.FileSize-metadataSize)
	if err != nil {
		return nil, fmt.Errorf("failed to read metadata: %w", err)
	}

	info.ModelSize = binary.LittleEndian.Uint64(metadataBytes)
	if info.ModelSize > 0 {
		info.ModelOffset = info.FileSize - int64(info.ModelSize) - metadataSize
		if info.ModelOffset >= 0 {
			info.HasModel = true
		}
	}

	return info, nil
}

// LoadModelFromSelfCopy reads the embedded model by copying it into memory.
// This is a fallback method that uses traditional memory allocation.
// Use this only when mmap is not available or suitable.
func LoadModelFromSelfCopy() ([]byte, error) {
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

	// Allocate memory for the entire model
	modelData := make([]byte, modelSize)

	// Read the model data directly into memory
	// Using ReadAt for thread-safe reading at specific offset
	bytesRead, err := file.ReadAt(modelData, modelOffset)
	if err != nil && err != io.EOF {
		return nil, fmt.Errorf("failed to read model data: %w", err)
	}

	if uint64(bytesRead) != modelSize {
		return nil, fmt.Errorf("incomplete model read: expected %d bytes, got %d", modelSize, bytesRead)
	}

	return modelData, nil
}

// LoadModelFromSelfStreaming reads the embedded model with a streaming approach
// for better memory efficiency during the loading phase.
// It still returns the full model in memory, but uses buffered reading.
func LoadModelFromSelfStreaming() ([]byte, error) {
	// Get the path to the current executable
	execPath, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("failed to get executable path: %w", err)
	}

	// Resolve any symlinks
	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve executable path: %w", err)
	}

	// Open the executable
	file, err := os.Open(execPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open executable: %w", err)
	}
	defer file.Close()

	// Get file info
	stat, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to stat executable: %w", err)
	}
	fileSize := stat.Size()

	// Read metadata
	if fileSize < metadataSize {
		return nil, fmt.Errorf("executable too small to contain embedded model")
	}

	// Seek to metadata position
	_, err = file.Seek(fileSize-metadataSize, io.SeekStart)
	if err != nil {
		return nil, fmt.Errorf("failed to seek to metadata: %w", err)
	}

	// Read model size
	var modelSize uint64
	err = binary.Read(file, binary.LittleEndian, &modelSize)
	if err != nil {
		return nil, fmt.Errorf("failed to read model size: %w", err)
	}

	// Validate
	if modelSize == 0 {
		return nil, fmt.Errorf("no embedded model found")
	}

	modelOffset := fileSize - int64(modelSize) - metadataSize
	if modelOffset < 0 {
		return nil, fmt.Errorf("invalid model size: %d", modelSize)
	}

	// Seek to model start
	_, err = file.Seek(modelOffset, io.SeekStart)
	if err != nil {
		return nil, fmt.Errorf("failed to seek to model data: %w", err)
	}

	// Allocate memory for the model
	modelData := make([]byte, modelSize)

	// Read model data with buffered reading for efficiency
	buffer := make([]byte, bufferSize)
	var totalRead uint64 = 0

	for totalRead < modelSize {
		// Calculate how much to read in this iteration
		remaining := modelSize - totalRead
		readSize := bufferSize
		if remaining < uint64(bufferSize) {
			readSize = int(remaining)
		}

		// Read into temporary buffer
		n, err := file.Read(buffer[:readSize])
		if err != nil && err != io.EOF {
			return nil, fmt.Errorf("failed to read model data at offset %d: %w", totalRead, err)
		}

		if n == 0 {
			break
		}

		// Copy from buffer to final destination
		copy(modelData[totalRead:totalRead+uint64(n)], buffer[:n])
		totalRead += uint64(n)
	}

	if totalRead != modelSize {
		return nil, fmt.Errorf("incomplete model read: expected %d bytes, got %d", modelSize, totalRead)
	}

	return modelData, nil
}
