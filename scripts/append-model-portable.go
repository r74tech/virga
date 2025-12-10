// append-model-portable.go
// Portable Go script to append a model to a binary with metadata footer.
// This works on all platforms (Windows, Linux, macOS).
//
// Usage: go run append-model-portable.go <binary> <model.gguf> [output]

//go:build ignore
// +build ignore

package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: go run append-model-portable.go <binary> <model.gguf> [output]")
		fmt.Println("  binary     - The Go executable to append the model to")
		fmt.Println("  model.gguf - The model file to embed")
		fmt.Println("  output     - Optional output filename (default: binary_with_model)")
		os.Exit(1)
	}

	binaryPath := os.Args[1]
	modelPath := os.Args[2]
	outputPath := binaryPath + "_with_model"
	if len(os.Args) > 3 {
		outputPath = os.Args[3]
	}

	// Check if files exist
	binaryStat, err := os.Stat(binaryPath)
	if err != nil {
		fmt.Printf("Error: Cannot access binary file '%s': %v\n", binaryPath, err)
		os.Exit(1)
	}

	modelStat, err := os.Stat(modelPath)
	if err != nil {
		fmt.Printf("Error: Cannot access model file '%s': %v\n", modelPath, err)
		os.Exit(1)
	}

	modelSize := modelStat.Size()
	if modelSize == 0 {
		fmt.Println("Error: Model file is empty")
		os.Exit(1)
	}

	fmt.Println("Embedding model into binary...")
	fmt.Printf("  Binary: %s (%.2f MB)\n", binaryPath, float64(binaryStat.Size())/1024/1024)
	fmt.Printf("  Model:  %s (%.2f GB)\n", modelPath, float64(modelSize)/1024/1024/1024)
	fmt.Printf("  Output: %s\n", outputPath)

	// Open input files
	binaryFile, err := os.Open(binaryPath)
	if err != nil {
		fmt.Printf("Error opening binary: %v\n", err)
		os.Exit(1)
	}
	defer binaryFile.Close()

	modelFile, err := os.Open(modelPath)
	if err != nil {
		fmt.Printf("Error opening model: %v\n", err)
		os.Exit(1)
	}
	defer modelFile.Close()

	// Create output file
	outputFile, err := os.Create(outputPath)
	if err != nil {
		fmt.Printf("Error creating output file: %v\n", err)
		os.Exit(1)
	}
	defer outputFile.Close()

	// Copy binary
	fmt.Print("Copying binary...")
	bytesCopied, err := io.Copy(outputFile, binaryFile)
	if err != nil {
		fmt.Printf("\nError copying binary: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf(" %d bytes\n", bytesCopied)

	// Copy model
	fmt.Print("Appending model...")
	modelBytesCopied, err := io.Copy(outputFile, modelFile)
	if err != nil {
		fmt.Printf("\nError copying model: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf(" %d bytes\n", modelBytesCopied)

	if modelBytesCopied != modelSize {
		fmt.Printf("Warning: Model size mismatch. Expected %d, copied %d\n", modelSize, modelBytesCopied)
	}

	// Write metadata (model size as uint64, little-endian)
	fmt.Print("Writing metadata...")
	err = binary.Write(outputFile, binary.LittleEndian, uint64(modelSize))
	if err != nil {
		fmt.Printf("\nError writing metadata: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(" 8 bytes")

	// Make executable on Unix-like systems
	outputFile.Close()
	if err := os.Chmod(outputPath, 0755); err != nil {
		// Ignore error on Windows
		if !os.IsNotExist(err) && !os.IsPermission(err) {
			fmt.Printf("Warning: Could not set executable permission: %v\n", err)
		}
	}

	// Verify result
	finalStat, err := os.Stat(outputPath)
	if err != nil {
		fmt.Printf("Error verifying output: %v\n", err)
		os.Exit(1)
	}

	expectedSize := binaryStat.Size() + modelSize + 8
	if finalStat.Size() != expectedSize {
		fmt.Printf("Warning: Size mismatch\n")
		fmt.Printf("  Expected: %d bytes\n", expectedSize)
		fmt.Printf("  Actual:   %d bytes\n", finalStat.Size())
	} else {
		fmt.Println("Success! Model embedded successfully.")
		fmt.Printf("Final binary size: %.2f GB\n", float64(finalStat.Size())/1024/1024/1024)
	}
}

// Helper function to format bytes in human-readable format
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.2f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}