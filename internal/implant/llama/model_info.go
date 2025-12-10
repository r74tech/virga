//go:build llama_embed || llama_external || llama_selfextract
// +build llama_embed llama_external llama_selfextract

package llama

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"

	"github.com/r74tech/virga/internal/implant/logger"
)

// GGUFModelInfo contains information about the loaded GGUF model file
type GGUFModelInfo struct {
	Magic       string
	Version     uint32
	TensorCount uint64
	MetadataKV  map[string]interface{}
}

// ReadGGUFModelInfo reads basic information from a GGUF file
func ReadGGUFModelInfo(path string) (*GGUFModelInfo, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	info := &GGUFModelInfo{
		MetadataKV: make(map[string]interface{}),
	}

	// Read magic (4 bytes)
	magic := make([]byte, 4)
	if _, err := io.ReadFull(f, magic); err != nil {
		return nil, err
	}
	info.Magic = string(magic)

	if info.Magic != "GGUF" {
		return nil, fmt.Errorf("invalid GGUF magic: %s", info.Magic)
	}

	// Read version (4 bytes, little endian)
	if err := binary.Read(f, binary.LittleEndian, &info.Version); err != nil {
		return nil, err
	}

	// Read tensor count (8 bytes, little endian)
	if err := binary.Read(f, binary.LittleEndian, &info.TensorCount); err != nil {
		return nil, err
	}

	// Note: Full metadata parsing is complex and depends on version
	// For now, we just read the basic header information

	return info, nil
}

// LogModelInfo logs information about the loaded model
func (le *LlamaEngine) LogModelInfo() {
	log := logger.Get()

	log.Info("=== LLAMA MODEL INFORMATION ===", nil)

	if le.config.ModelPath != "" {
		log.Info("Model source", map[string]interface{}{
			"source":     "file",
			"model_path": le.config.ModelPath,
		})

		// Try to read GGUF model info
		info, err := ReadGGUFModelInfo(le.config.ModelPath)
		if err != nil {
			log.Warn("Failed to read GGUF model info", map[string]interface{}{
				"error": err.Error(),
			})
		} else {
			log.Info("GGUF model file information", map[string]interface{}{
				"magic":        info.Magic,
				"version":      info.Version,
				"tensor_count": info.TensorCount,
			})

			// Get file size
			stat, err := os.Stat(le.config.ModelPath)
			if err == nil {
				log.Info("Model file size", map[string]interface{}{
					"size_bytes": stat.Size(),
					"size_mb":    float64(stat.Size()) / 1024.0 / 1024.0,
					"size_gb":    float64(stat.Size()) / 1024.0 / 1024.0 / 1024.0,
				})
			}
		}
	} else if len(le.config.ModelData) > 0 {
		log.Info("Model source", map[string]interface{}{
			"source":     "memory",
			"size_mb":    float64(len(le.config.ModelData)) / 1024.0 / 1024.0,
			"size_gb":    float64(len(le.config.ModelData)) / 1024.0 / 1024.0 / 1024.0,
			"use_mmap":   le.config.UseMmap,
			"use_memory": le.config.UseMemory,
		})

		// Check magic number in memory
		if len(le.config.ModelData) >= 4 {
			magic := string(le.config.ModelData[0:4])
			log.Debug("Model data magic number", map[string]interface{}{
				"magic": magic,
			})
		}
	} else {
		log.Warn("No model data found", nil)
	}

	// Log configuration
	log.Info("Llama engine configuration", map[string]interface{}{
		"context":            le.config.Context,
		"gpu_layers":         le.config.GPULayers,
		"threads":            le.config.Threads,
		"temperature":        le.config.Temperature,
		"top_k":              le.config.TopK,
		"top_p":              le.config.TopP,
		"max_tokens":         le.config.MaxTokens,
		"max_iterations":     le.config.MaxIterations,
		"jinja":              le.config.Jinja,
		"use_self_contained": le.config.UseSelfContained,
		"enable_auto_mode":   le.config.EnableAutoMode,
	})

	// Log system prompt info
	if le.systemPrompt != "" {
		log.Debug("System prompt information", map[string]interface{}{
			"prompt_length": len(le.systemPrompt),
			"prompt_lines":  len([]byte(le.systemPrompt)),
			"prompt_preview": func() string {
				if len(le.systemPrompt) > 200 {
					return le.systemPrompt[:200] + "..."
				}
				return le.systemPrompt
			}(),
		})
	}

	// Log task prompts
	if len(le.config.TaskPrompts) > 0 {
		log.Info("Task-specific prompts", map[string]interface{}{
			"task_count": len(le.config.TaskPrompts),
			"tasks": func() []string {
				tasks := make([]string, 0, len(le.config.TaskPrompts))
				for k := range le.config.TaskPrompts {
					tasks = append(tasks, k)
				}
				return tasks
			}(),
		})
	}

	log.Info("=== END MODEL INFORMATION ===", nil)
}
