package main

import (
	"encoding/binary"
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run verify-embedded-model.go <beacon-binary>")
		os.Exit(1)
	}

	binaryPath := os.Args[1]

	file, err := os.Open(binaryPath)
	if err != nil {
		fmt.Printf("Error opening file: %v\n", err)
		os.Exit(1)
	}
	defer file.Close()

	// Get file size
	stat, err := file.Stat()
	if err != nil {
		fmt.Printf("Error getting file info: %v\n", err)
		os.Exit(1)
	}
	fileSize := stat.Size()
	fmt.Printf("Total file size: %d bytes (%.2f GB)\n", fileSize, float64(fileSize)/(1024*1024*1024))

	// Read the last 8 bytes (metadata)
	metadataBytes := make([]byte, 8)
	_, err = file.ReadAt(metadataBytes, fileSize-8)
	if err != nil {
		fmt.Printf("Error reading metadata: %v\n", err)
		os.Exit(1)
	}

	modelSize := binary.LittleEndian.Uint64(metadataBytes)
	fmt.Printf("Model size from metadata: %d bytes (%.2f GB)\n", modelSize, float64(modelSize)/(1024*1024*1024))

	// Calculate model offset
	modelOffset := fileSize - int64(modelSize) - 8
	fmt.Printf("Model starts at offset: %d (0x%x)\n", modelOffset, modelOffset)
	fmt.Printf("Model ends at offset: %d (0x%x)\n", fileSize-8, fileSize-8)

	// Read first 16 bytes at model offset to check for GGUF header
	headerBytes := make([]byte, 16)
	_, err = file.ReadAt(headerBytes, modelOffset)
	if err != nil {
		fmt.Printf("Error reading model header: %v\n", err)
		os.Exit(1)
	}

	// Check for GGUF magic (0x46554747 = "GGUF" in little-endian)
	magic := binary.LittleEndian.Uint32(headerBytes[0:4])
	version := binary.LittleEndian.Uint32(headerBytes[4:8])

	fmt.Printf("\nAt model offset:\n")
	fmt.Printf("  First 16 bytes (hex): % 02x\n", headerBytes)
	fmt.Printf("  Magic: 0x%08x (expected 0x46554747 for GGUF)\n", magic)

	if magic == 0x46554747 {
		fmt.Printf("  ✅ Valid GGUF header found!\n")
		fmt.Printf("  GGUF Version: %d\n", version)
	} else {
		fmt.Printf("  ❌ Invalid GGUF header!\n")

		// Show what's actually there as ASCII
		fmt.Printf("  As ASCII: ")
		for _, b := range headerBytes[:16] {
			if b >= 32 && b <= 126 {
				fmt.Printf("%c", b)
			} else {
				fmt.Printf(".")
			}
		}
		fmt.Printf("\n")

		// Try to find GGUF header by scanning
		fmt.Println("\nScanning for GGUF header...")
		scanBuf := make([]byte, 4)
		found := false
		for offset := int64(0); offset < fileSize-4; offset += 1024 * 1024 { // Scan every 1MB
			file.ReadAt(scanBuf, offset)
			if binary.LittleEndian.Uint32(scanBuf) == 0x46554747 {
				fmt.Printf("  Found GGUF header at offset %d (0x%x)\n", offset, offset)
				found = true
				break
			}
		}
		if !found {
			fmt.Println("  GGUF header not found in file")
		}
	}

	// Check binary size without model
	originalBinarySize := modelOffset
	fmt.Printf("\nOriginal binary size (without model): %d bytes (%.2f MB)\n", originalBinarySize, float64(originalBinarySize)/(1024*1024))
}
