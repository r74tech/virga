//go:build ignore
// +build ignore

package main

import (
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/r74tech/virga/internal/shared/logger"
)

const (
	modelURL  = "https://huggingface.co/ggml-org/Qwen3-4B-GGUF/resolve/main/Qwen3-4B-Q4_K_M.gguf"
	modelPath = "internal/implant/llama/models/model.gguf"
)

func main() {
	// Create models directory
	modelDir := filepath.Dir(modelPath)
	if err := os.MkdirAll(modelDir, 0755); err != nil {
		logger.Error("Failed to create models directory: %v", err)
		os.Exit(1)
	}

	// Check if model already exists
	if stat, err := os.Stat(modelPath); err == nil {
		logger.Info("Model already exists (%.2f MB)", float64(stat.Size())/(1024*1024))
		return
	}

	logger.Info("Downloading TinyLlama model from Hugging Face...")
	logger.Info("URL: %s", modelURL)

	// Download the model
	resp, err := http.Get(modelURL)
	if err != nil {
		logger.Error("Failed to download model: %v", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logger.Error("Failed to download model: HTTP %d", resp.StatusCode)
		os.Exit(1)
	}

	// Create the output file
	out, err := os.Create(modelPath)
	if err != nil {
		logger.Error("Failed to create model file: %v", err)
		os.Exit(1)
	}
	defer out.Close()

	// Copy with progress
	written, err := io.Copy(out, resp.Body)
	if err != nil {
		logger.Error("Failed to save model: %v", err)
		os.Exit(1)
	}

	logger.Info("Successfully downloaded model (%.2f MB)", float64(written)/(1024*1024))
	logger.Info("Model saved to: %s", modelPath)
}
