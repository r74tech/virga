//go:build llama_embed || llama_external || llama_selfextract
// +build llama_embed llama_external llama_selfextract

package llama

/*
#include <stdlib.h>
*/
import "C"

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Qitmeer/llama.go/wrapper"
	"github.com/r74tech/virga/internal/implant/config"
	"github.com/r74tech/virga/internal/implant/logger"
	"github.com/r74tech/virga/internal/implant/memdb"
)

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// LlamaEngine uses the llama.go wrapper
type LlamaEngine struct {
	loaded                  bool // Whether the model is loaded
	config                  Config
	systemPrompt            string
	mutex                   sync.Mutex
	executeCommand          func(string) CommandExecution
	memdb                   *memdb.DB // In-memory database for observation storage
	iterationResultCallback func(taskID, iterationOutput string)
}

// Config is the configuration for the LlamaEngine
type Config struct {
	ModelPath        string
	ModelData        []byte
	UseMemory        bool
	UseMmap          bool // Whether the data is mmaped (true when using selfextract)
	UseSelfContained bool // Whether to use self-contained binary
	Context          int
	GPULayers        int
	Threads          int
	Temperature      float64
	TopK             int
	TopP             float64
	SystemPrompt     string
	MaxTokens        int                   // Maximum tokens per response
	MaxIterations    int                   // Maximum iterations per task
	TaskPrompts      map[string]string     // Task-specific prompts
	EnableAutoMode   bool                  // Enable autonomous mode
	Jinja            bool                  // Enable Jinja template processing (required for some models like gpt-oss-20b)
	MaxWorkers       int                   // Maximum number of concurrent task workers (default: 2)
	PayloadInfos     []*PayloadCommandInfo // Payload command information for prompt injection
}

// Task represents a Llama task
type Task struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	Description string                 `json:"description"`
	Prompt      string                 `json:"prompt"`
	Status      string                 `json:"status"`
	Result      interface{}            `json:"result,omitempty"`
	Error       string                 `json:"error,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	StartTime   time.Time              `json:"start_time"`
	EndTime     time.Time              `json:"end_time,omitempty"`
}

// TaskResult represents the result of a task execution
type TaskResult struct {
	Output    string                 `json:"output"`
	Commands  []CommandExecution     `json:"commands,omitempty"`
	Findings  map[string]interface{} `json:"findings,omitempty"`
	NextSteps []string               `json:"next_steps,omitempty"`
}

// CommandExecution represents the result of a command execution
type CommandExecution struct {
	Command  string `json:"command"`
	Output   string `json:"output"`
	Error    string `json:"error,omitempty"`
	ExitCode int    `json:"exit_code"`
}

// Default configuration
var defaultConfig = Config{
	Context:       8192,
	Threads:       runtime.NumCPU(),
	Temperature:   0.3,
	TopK:          40,
	TopP:          0.95,
	GPULayers:     0,
	MaxTokens:     2048,
	MaxIterations: 50,
	Jinja:         false, // Jinja template processing (required for some models like gpt-oss-20b)
}

// NewLlamaEngine creates a new LlamaEngine
func NewLlamaEngine(engineConfig Config) (*LlamaEngine, error) {
	// Llama log control
	if !config.GetLlamaLogEnabled() {
		os.Setenv("LLAMA_LOG_DISABLE", "1")
	}

	// Apply default values
	if engineConfig.Context == 0 {
		engineConfig.Context = defaultConfig.Context
	}
	if engineConfig.Threads == 0 {
		engineConfig.Threads = defaultConfig.Threads
	}
	if engineConfig.Temperature == 0 {
		engineConfig.Temperature = defaultConfig.Temperature
	}
	if engineConfig.TopK == 0 {
		engineConfig.TopK = defaultConfig.TopK
	}
	if engineConfig.TopP == 0 {
		engineConfig.TopP = defaultConfig.TopP
	}
	if engineConfig.MaxTokens == 0 {
		engineConfig.MaxTokens = defaultConfig.MaxTokens
	}
	if engineConfig.MaxIterations == 0 {
		engineConfig.MaxIterations = defaultConfig.MaxIterations
	}

	// Validate thread count
	if engineConfig.Threads < 1 {
		engineConfig.Threads = 1
	} else if engineConfig.Threads > runtime.NumCPU()*2 {
		engineConfig.Threads = runtime.NumCPU()
	}

	// Build arguments for llama.go wrapper
	args := buildArgs(engineConfig)

	var err error

	// Recovery mechanism
	func() {
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("model initialization panic: %v", r)
			}
		}()

		if engineConfig.UseSelfContained {
			// Load self-contained model
			// TODO: Implement in selfextract.go
			err = fmt.Errorf("self-contained model loading not yet implemented")
		} else if engineConfig.UseMemory && len(engineConfig.ModelData) > 0 {
			// Validate GGUF magic before passing to llama.go
			if len(engineConfig.ModelData) >= 16 {
				magic := engineConfig.ModelData[0:4]
				if log := logger.Get(); log != nil {
					log.Debug("Model data validation before llama.go", map[string]interface{}{
						"model_size_mb":  float64(len(engineConfig.ModelData)) / (1024 * 1024),
						"magic_hex":      fmt.Sprintf("%x", magic),
						"magic_string":   string(magic),
						"first_16_bytes": fmt.Sprintf("%x", engineConfig.ModelData[0:16]),
					})
				}
				if string(magic) != "GGUF" {
					if log := logger.Get(); log != nil {
						log.Error("Invalid GGUF magic in model data", map[string]interface{}{
							"got_hex":      fmt.Sprintf("%x", magic),
							"got_string":   string(magic),
							"expected_hex": "47475546",
						})
					}
				}
			}

			// Load from memory
			// Use LoadFromMmap for mmaped data (important for large models)
			if engineConfig.UseMmap {
				if log := logger.Get(); log != nil {
					log.Debug("Loading model from mmap", map[string]interface{}{
						"model_size_mb": float64(len(engineConfig.ModelData)) / (1024 * 1024),
					})
				}
				err = wrapper.LoadFromMmap(0, engineConfig.ModelData, args)
			} else {
				if log := logger.Get(); log != nil {
					log.Debug("Loading model from memory", map[string]interface{}{
						"model_size_mb": float64(len(engineConfig.ModelData)) / (1024 * 1024),
					})
				}
				err = wrapper.LoadFromMemory(engineConfig.ModelData, args)
			}
		} else if engineConfig.ModelPath != "" {
			// Load from file
			// wrapper.LlamaStart takes config.Config, so we need a different method
			// Instead, use LoadFromMemory or load the model
			_ = fmt.Sprintf("%s --model %s", args, engineConfig.ModelPath) // Suppress unused warning
			err = fmt.Errorf("file-based loading not yet implemented, use LoadFromMemory")
		} else {
			err = fmt.Errorf("no model provided")
		}
	}()

	if err != nil {
		return nil, fmt.Errorf("failed to load model: %w", err)
	}

	systemPrompt := engineConfig.SystemPrompt
	if systemPrompt == "" {
		systemPrompt = GetPromptByPreset("default")
	}

	// Inject payload command information into system prompt
	if len(engineConfig.PayloadInfos) > 0 {
		payloadPrompt := GenerateMultiplePayloadPrompt(engineConfig.PayloadInfos)
		systemPrompt = systemPrompt + payloadPrompt
	}

	// Initialize command execution functionality with payload information
	var executor *SystemCommandExecutor
	if len(engineConfig.PayloadInfos) > 0 {
		executor = NewCommandExecutorWithPayloads(engineConfig.PayloadInfos)
	} else {
		executor = NewCommandExecutor()
	}

	engine := &LlamaEngine{
		loaded:         true,
		config:         engineConfig,
		systemPrompt:   systemPrompt,
		executeCommand: executor.Execute,
	}

	// Log model information
	engine.LogModelInfo()

	return engine, nil
}

// buildArgs builds the arguments string for the llama.go wrapper
func buildArgs(cfg Config) string {
	args := fmt.Sprintf("llama --ctx-size %d --n-gpu-layers %d --batch-size 2048 --ubatch-size 512 --threads %d",
		cfg.Context,
		cfg.GPULayers,
		cfg.Threads,
	)

	if cfg.Jinja {
		args += " --jinja"
	}

	args += " --log-disable"
	return args
}

// Close closes the engine
func (le *LlamaEngine) Close() {
	le.mutex.Lock()
	defer le.mutex.Unlock()

	if le.loaded {
		func() {
			defer func() {
				if r := recover(); r != nil {
					if log := logger.Get(); log != nil {
						log.Warn("Panic during model cleanup", map[string]interface{}{
							"panic": fmt.Sprintf("%v", r),
						})
					}
				}
			}()
			wrapper.LlamaStop()
		}()
		le.loaded = false
	}
}

// ExecuteTask executes a task
// ProgressCallback is called on each iteration with current progress
type ProgressCallback func(currentIteration, maxIterations int)

func (le *LlamaEngine) ExecuteTask(ctx context.Context, task *Task) (*TaskResult, error) {
	return le.ExecuteTaskWithProgress(ctx, task, nil)
}

func (le *LlamaEngine) ExecuteTaskWithProgress(ctx context.Context, task *Task, progressCallback ProgressCallback) (*TaskResult, error) {
	le.mutex.Lock()
	defer le.mutex.Unlock()

	log := logger.Get()

	// Check if the model is loaded
	if !le.loaded {
		return nil, fmt.Errorf("llama model is not initialized")
	}

	task.StartTime = time.Now()
	task.Status = "running"

	log.LogLlama("execute_task_start", map[string]interface{}{
		"task_id":   task.ID,
		"task_type": task.Type,
	})

	log.Info("=== LLAMA TASK START ===", map[string]interface{}{
		"task_id":           task.ID,
		"task_type":         task.Type,
		"task_description":  task.Description,
		"model_temperature": le.config.Temperature,
		"max_iterations":    le.config.MaxIterations,
		"max_tokens":        le.config.MaxTokens,
		"context_size":      le.config.Context,
		"threads":           le.config.Threads,
		"gpu_layers":        le.config.GPULayers,
		"jinja_enabled":     le.config.Jinja,
		"use_memory":        le.config.UseMemory,
		"use_mmap":          le.config.UseMmap,
	})

	basePrompt := le.buildPrompt(task)

	// Initialize MultiContextManager for LLM-based context management
	contextMgr := NewMultiContextManager(
		le.memdb,
		task.ID,
		basePrompt,
		2, // Keep 2 most recent observations
		le.predictWithOptions,
	)

	log.Info("Task prompt built with MultiContextManager", map[string]interface{}{
		"task_id":       task.ID,
		"task_type":     task.Type,
		"prompt_length": len(basePrompt),
		"prompt_preview": func() string {
			if len(basePrompt) > 500 {
				return basePrompt[:500] + "..."
			}
			return basePrompt
		}(),
	})

	// debug mode: log full initial prompt
	log.Debug("Full initial prompt", map[string]interface{}{
		"task_id":      task.ID,
		"prompt_full":  basePrompt,
		"prompt_lines": len(strings.Split(basePrompt, "\n")),
	})

	iterations := 0
	maxIterations := le.config.MaxIterations
	if maxIterations == 0 || maxIterations > 20 {
		maxIterations = 10
	}

	var result TaskResult
	result.Commands = make([]CommandExecution, 0)
	result.Findings = make(map[string]interface{})
	result.NextSteps = make([]string, 0)

	var fullOutput strings.Builder

	// Iterative task execution
	for iterations < maxIterations {
		select {
		case <-ctx.Done():
			task.Status = "cancelled"
			task.Error = "task cancelled"
			return &result, ctx.Err()
		default:
		}

		iterations++

		// Report progress via callback
		if progressCallback != nil {
			progressCallback(iterations, maxIterations)
		}

		// Build prompt from managed context
		prompt := contextMgr.BuildPrompt()
		contextStats := contextMgr.GetStats()

		log.Info(fmt.Sprintf("==================== ITERATION %d/%d START ====================", iterations, maxIterations), map[string]interface{}{
			"task_id":       task.ID,
			"iteration":     iterations,
			"max":           maxIterations,
			"prompt_length": contextStats["prompt_length"],
			"facts_count":   contextStats["facts_count"],
			"recent_obs":    contextStats["recent_obs_count"],
			"old_summaries": contextStats["old_summaries"],
			"commands_exec": contextStats["commands_total"],
		})

		// debug mode: log full prompt for this iteration
		log.Debug("Full prompt for this iteration", map[string]interface{}{
			"task_id":      task.ID,
			"iteration":    iterations,
			"prompt_full":  prompt,
			"prompt_lines": contextStats["prompt_lines"],
		})

		// Execute Llama prediction
		response, err := le.predict(prompt)
		if err != nil {
			task.Status = "failed"
			task.Error = err.Error()
			log.Error("Llama prediction failed", map[string]interface{}{
				"task_id":   task.ID,
				"iteration": iterations,
				"error":     err.Error(),
			})
			return nil, fmt.Errorf("prediction failed at iteration %d: %w", iterations, err)
		}

		fullOutput.WriteString(response)
		fullOutput.WriteString("\n")

		// debug mode: log full response for this iteration
		log.Info("Llama Response", map[string]interface{}{
			"task_id":   task.ID,
			"iteration": iterations,
			"response":  response,
		})
		log.Debug("Response statistics", map[string]interface{}{
			"task_id":         task.ID,
			"iteration":       iterations,
			"response_length": len(response),
			"response_lines":  len(strings.Split(response, "\n")),
			"tokens_approx":   len(strings.Fields(response)),
		})

		// Extract and execute commands
		commands := le.extractCommands(response)
		if len(commands) > 0 {
			log.Info(fmt.Sprintf("Executing %d command(s)", len(commands)), map[string]interface{}{
				"task_id":   task.ID,
				"iteration": iterations,
				"commands":  commands,
			})

			for i, cmd := range commands {
				log.Info(fmt.Sprintf("Command %d/%d: %s", i+1, len(commands), cmd), map[string]interface{}{
					"task_id":   task.ID,
					"iteration": iterations,
				})

				cmdResult := le.executeCommand(cmd)
				result.Commands = append(result.Commands, cmdResult)

				// Add observation to MultiContextManager
				contextMgr.AddObservation(cmd, cmdResult.Output, cmdResult.ExitCode)

				// Log command result with truncated output for readability
				outputPreview := cmdResult.Output
				if len(outputPreview) > 200 {
					outputPreview = outputPreview[:200] + "... (truncated)"
				}
				log.Info("Command result", map[string]interface{}{
					"task_id":   task.ID,
					"iteration": iterations,
					"command":   cmd,
					"exit_code": cmdResult.ExitCode,
					"output":    outputPreview,
				})
			}
		}

		// Extract findings
		findings := le.extractFindings(response)
		for k, v := range findings {
			result.Findings[k] = v
		}

		// Send partial result after each iteration
		if le.iterationResultCallback != nil {
			iterationOutput := fmt.Sprintf("=== Iteration %d/%d ===\n%s", iterations, maxIterations, response)
			log.Debug("Sending partial result via callback", map[string]interface{}{
				"task_id":    task.ID,
				"iteration":  iterations,
				"output_len": len(iterationOutput),
			})
			le.iterationResultCallback(task.ID, iterationOutput)
		} else {
			log.Warn("Iteration result callback is nil", map[string]interface{}{
				"task_id":   task.ID,
				"iteration": iterations,
			})
		}

		// Check if task is complete
		if strings.Contains(response, "[TASK_COMPLETE]") {
			log.Info("Task marked as complete", map[string]interface{}{
				"task_id":    task.ID,
				"iterations": iterations,
			})
			break
		}

		// If there are no commands, prompt for reconsideration (ReAct improvement)
		if len(commands) == 0 {
			log.Debug("No commands extracted, prompting reconsideration", map[string]interface{}{
				"task_id":    task.ID,
				"iterations": iterations,
			})

			// prompt for reconsideration (using current context)
			currentPrompt := contextMgr.BuildPrompt()
			reconsiderPrompt := currentPrompt + `

[SYSTEM NOTICE]
No action was detected in your previous response.

Please reconsider:
- Have you completed the task? If YES, output [TASK_COMPLETE]
- If NO, what else needs to be done? Provide your next action.

Thought:`

			// get response for reconsideration
			reconsiderResponse, err := le.predict(reconsiderPrompt)
			if err != nil {
				log.Error("Reconsideration prediction failed", map[string]interface{}{
					"task_id":   task.ID,
					"iteration": iterations,
					"error":     err.Error(),
				})
				break
			}

			fullOutput.WriteString(reconsiderResponse)
			fullOutput.WriteString("\n")

			log.Debug("Reconsideration response received", map[string]interface{}{
				"task_id":         task.ID,
				"iteration":       iterations,
				"response_length": len(reconsiderResponse),
			})

			// extract commands again
			retriedCommands := le.extractCommands(reconsiderResponse)

			// exit if task is explicitly marked complete
			if len(retriedCommands) == 0 && strings.Contains(reconsiderResponse, "[TASK_COMPLETE]") {
				log.Info("Task explicitly marked complete after reconsideration", map[string]interface{}{
					"task_id":    task.ID,
					"iterations": iterations,
				})
				break
			}

			// continue to next iteration if no commands are extracted
			if len(retriedCommands) == 0 {
				log.Warn("Still no commands after reconsideration, continuing to next iteration", map[string]interface{}{
					"task_id":    task.ID,
					"iterations": iterations,
				})
				continue
			}

			// use re-extracted commands
			log.Info("Commands extracted after reconsideration", map[string]interface{}{
				"task_id":       task.ID,
				"iteration":     iterations,
				"command_count": len(retriedCommands),
			})

			// execute re-extracted commands
			for _, cmd := range retriedCommands {
				cmdResult := le.executeCommand(cmd)
				result.Commands = append(result.Commands, cmdResult)

				// Add observation to MultiContextManager
				contextMgr.AddObservation(cmd, cmdResult.Output, cmdResult.ExitCode)

				log.Debug("Observation added to context manager (after reconsideration)", map[string]interface{}{
					"task_id":    task.ID,
					"iteration":  iterations,
					"command":    cmd,
					"exit_code":  cmdResult.ExitCode,
					"output_len": len(cmdResult.Output),
				})
			}
		}

		// log end of iteration
		endStats := contextMgr.GetStats()
		log.Info(fmt.Sprintf("==================== ITERATION %d/%d END ====================", iterations, maxIterations), map[string]interface{}{
			"task_id":            task.ID,
			"iteration":          iterations,
			"commands_this_iter": len(commands),
			"total_commands":     len(result.Commands),
			"findings_count":     len(result.Findings),
			"context_size":       endStats["prompt_length"],
			"facts_discovered":   endStats["facts_count"],
		})
	}

	// Generate final report from collected information
	finalReport, err := le.generateFinalReport(task, &result, fullOutput.String(), contextMgr)
	if err != nil {
		log.Warn("Failed to generate final report, using fallback summary", map[string]interface{}{
			"task_id": task.ID,
			"error":   err.Error(),
		})

		// Create a simple fallback summary instead of using full output
		var fallbackSummary strings.Builder
		fallbackSummary.WriteString(fmt.Sprintf("## Task: %s\n\n", task.Prompt))
		fallbackSummary.WriteString(fmt.Sprintf("**Status**: Completed with %d iterations\n\n", iterations))
		fallbackSummary.WriteString(fmt.Sprintf("**Commands Executed**: %d\n\n", len(result.Commands)))

		if len(result.Commands) > 0 {
			fallbackSummary.WriteString("### Commands:\n")
			for i, cmd := range result.Commands {
				fallbackSummary.WriteString(fmt.Sprintf("%d. `%s`\n", i+1, cmd.Command))
				if cmd.ExitCode != 0 {
					fallbackSummary.WriteString(fmt.Sprintf("   - Exit code: %d\n", cmd.ExitCode))
				}
			}
		}

		result.Output = fallbackSummary.String()
	} else {
		result.Output = finalReport
	}

	task.Status = "completed"
	task.EndTime = time.Now()

	log.Info("=== LLAMA TASK COMPLETE ===", map[string]interface{}{
		"task_id":             task.ID,
		"task_type":           task.Type,
		"iterations":          iterations,
		"max_iterations":      maxIterations,
		"commands_executed":   len(result.Commands),
		"findings_count":      len(result.Findings),
		"duration_seconds":    task.EndTime.Sub(task.StartTime).Seconds(),
		"output_length":       len(result.Output),
		"avg_tokens_per_iter": len(strings.Fields(result.Output)) / max(iterations, 1),
	})

	return &result, nil
}

// predict executes Llama prediction (using llama.go wrapper)
func (le *LlamaEngine) predict(prompt string) (string, error) {
	log := logger.Get()

	// Check if the model is loaded
	if !le.loaded {
		return "", fmt.Errorf("llama model is not initialized")
	}

	// Validate input
	if len(prompt) == 0 {
		return "", fmt.Errorf("empty prompt provided")
	}

	// Limit prompt size
	maxPromptSize := le.config.Context * 4
	if len(prompt) > maxPromptSize {
		prompt = prompt[:maxPromptSize]
	}

	maxTokens := le.config.MaxTokens
	if maxTokens == 0 {
		maxTokens = 1024
	}

	log.Debug("Llama prediction starting", map[string]interface{}{
		"prompt_len":  len(prompt),
		"temperature": le.config.Temperature,
		"threads":     le.config.Threads,
		"max_tokens":  maxTokens,
	})

	predictionStartTime := time.Now()

	// Execute prediction using llama.go wrapper
	id, ch := wrapper.NewChan()
	if id == 0 {
		return "", fmt.Errorf("failed to create channel")
	}
	// Note: wrapper.CloseChan requires wrapper's C.int type which is incompatible with our C.int
	// The channel will be garbage collected when ch goes out of scope
	// TODO: Add a wrapper function in llama.go/wrapper package that accepts Go int
	_ = id // Keep id for potential future use

	// Build JSON request
	req := map[string]interface{}{
		"prompt":         prompt,
		"stream":         true,
		"temperature":    le.config.Temperature,
		"top_k":          le.config.TopK,
		"top_p":          le.config.TopP,
		"n_predict":      maxTokens,
		"repeat_penalty": 1.2,   // Penalize repetitions (1.0 = no penalty, higher = more penalty)
		"repeat_last_n":  128,   // Look back 128 tokens for repetitions
		"penalize_nl":    false, // Don't penalize newlines
		"stop": []string{ // Stop sequences
			"Observation:", // After action, wait for observation
			"<|end|>",      // Special token
			"<|start|>",    // Special token
		},
	}

	jsonBytes, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// Execute prediction asynchronously
	var predictErr error
	go func() {
		predictErr = wrapper.LlamaGenerate(id, string(jsonBytes))
	}()

	// Receive tokens from channel
	var result strings.Builder
	tokenCount := 0
	lastLogTime := time.Now()
	tokenLogInterval := 200
	timeLogInterval := 30 * time.Second

	// JSON response parsing structure
	// llama.go wrapper returns OpenAI-compatible JSON streaming format
	type StreamResponse struct {
		Choices []struct {
			Text         string `json:"text"`
			Index        int    `json:"index"`
			FinishReason string `json:"finish_reason"`
		} `json:"choices"`
		Created int64  `json:"created"`
		Model   string `json:"model"`
	}

	for token := range ch {
		if jsonStr, ok := token.(string); ok {
			// Debug: Log received raw tokens (only the first 10 tokens)
			if tokenCount < 10 {
				log.Debug("Llama token received", map[string]interface{}{
					"token_index": tokenCount,
					"raw_length":  len(jsonStr),
					"raw_preview": jsonStr[:min(100, len(jsonStr))],
				})
			}

			// data: Remove prefix (Server-Sent Events format)
			jsonStr = strings.TrimPrefix(jsonStr, "data: ")
			jsonStr = strings.TrimSpace(jsonStr)

			// [DONE] message is skipped
			if jsonStr == "[DONE]" {
				log.Debug("Llama received [DONE] message", nil)
				break
			}

			// Empty response is skipped
			if jsonStr == "" {
				continue
			}

			// Parse as JSON
			var response StreamResponse
			if err := json.Unmarshal([]byte(jsonStr), &response); err != nil {
				// JSON parse failed - treat as raw token text
				if tokenCount < 10 {
					log.Debug("JSON parse failed, treating as raw text", map[string]interface{}{
						"error":        err.Error(),
						"text_preview": jsonStr[:min(50, len(jsonStr))],
					})
				}
				result.WriteString(jsonStr)
				tokenCount++

				// Progress log
				if tokenCount%tokenLogInterval == 0 || time.Since(lastLogTime) > timeLogInterval {
					log.Debug("Llama token generation progress", map[string]interface{}{
						"tokens_generated":  tokenCount,
						"elapsed_seconds":   int(time.Since(predictionStartTime).Seconds()),
						"tokens_per_second": float64(tokenCount) / time.Since(predictionStartTime).Seconds(),
					})
					lastLogTime = time.Now()
				}
				continue
			}

			// Get text
			if len(response.Choices) > 0 {
				text := response.Choices[0].Text
				if text != "" {
					result.WriteString(text)
					tokenCount++

					// Progress log
					if tokenCount%tokenLogInterval == 0 || time.Since(lastLogTime) > timeLogInterval {
						log.Debug("Llama token generation progress", map[string]interface{}{
							"tokens_generated":  tokenCount,
							"elapsed_seconds":   int(time.Since(predictionStartTime).Seconds()),
							"tokens_per_second": float64(tokenCount) / time.Since(predictionStartTime).Seconds(),
						})
						lastLogTime = time.Now()
					}
				}

				// If finish_reason is present, exit
				finishReason := response.Choices[0].FinishReason
				if finishReason != "" && finishReason != "null" {
					log.Debug("Llama prediction finished", map[string]interface{}{
						"finish_reason": finishReason,
						"total_tokens":  tokenCount,
					})
					break
				}
			}
		}
	}

	if predictErr != nil {
		log.Error("Llama prediction failed", map[string]interface{}{
			"error":           predictErr.Error(),
			"elapsed_seconds": int(time.Since(predictionStartTime).Seconds()),
		})
		return "", fmt.Errorf("prediction failed: %w", predictErr)
	}

	log.Debug("Llama prediction completed", map[string]interface{}{
		"total_tokens":      tokenCount,
		"elapsed_seconds":   int(time.Since(predictionStartTime).Seconds()),
		"tokens_per_second": float64(tokenCount) / time.Since(predictionStartTime).Seconds(),
		"output_length":     result.Len(),
	})

	return result.String(), nil
}

// predictWithOptions executes prediction with custom options (for MultiContextManager)
func (le *LlamaEngine) predictWithOptions(prompt string, opts PredictOptions) (string, error) {
	log := logger.Get()

	// Check if the model is loaded
	if !le.loaded {
		return "", fmt.Errorf("llama model is not initialized")
	}

	// Validate input
	if len(prompt) == 0 {
		return "", fmt.Errorf("empty prompt provided")
	}

	// Limit prompt size
	maxPromptSize := le.config.Context * 4
	if len(prompt) > maxPromptSize {
		prompt = prompt[:maxPromptSize]
	}

	maxTokens := int(opts.MaxTokens)
	if maxTokens == 0 {
		maxTokens = 1024
	}

	temperature := opts.Temperature
	if temperature == 0 {
		temperature = float32(le.config.Temperature) // Use default if not specified
	}

	log.Debug("Llama prediction with options starting", map[string]interface{}{
		"prompt_len":  len(prompt),
		"temperature": temperature,
		"max_tokens":  maxTokens,
	})

	predictionStartTime := time.Now()

	// Execute prediction using llama.go wrapper
	id, ch := wrapper.NewChan()
	if id == 0 {
		return "", fmt.Errorf("failed to create channel")
	}

	// Build JSON request
	req := map[string]interface{}{
		"prompt":         prompt,
		"stream":         true,
		"temperature":    temperature,
		"top_k":          le.config.TopK,
		"top_p":          le.config.TopP,
		"n_predict":      maxTokens,
		"repeat_penalty": 1.2,
		"repeat_last_n":  128,
		"penalize_nl":    false,
		"stop": []string{
			"\n\n\n",
			"Observation:",
			"<|end|>",
			"<|start|>",
		},
	}

	jsonBytes, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// Execute prediction asynchronously
	var predictErr error
	go func() {
		predictErr = wrapper.LlamaGenerate(id, string(jsonBytes))
	}()

	// Read response stream
	var result strings.Builder
	tokenCount := 0

	for {
		select {
		case str, ok := <-ch:
			if !ok {
				goto done
			}

			// Parse JSON response
			var response struct {
				Choices []struct {
					Text         string `json:"text"`
					FinishReason string `json:"finish_reason"`
				} `json:"choices"`
				Error struct {
					Message string `json:"message"`
					Code    int    `json:"code"`
				} `json:"error"`
			}

			// Type assert str to string
			strVal, ok := str.(string)
			if !ok {
				continue
			}

			if err := json.Unmarshal([]byte(strVal), &response); err != nil {
				continue
			}

			// Check for errors
			if response.Error.Message != "" {
				predictErr = fmt.Errorf("llama error: %s (code: %d)", response.Error.Message, response.Error.Code)
				goto done
			}

			// Check for stop signal
			if len(response.Choices) > 0 && response.Choices[0].FinishReason != "" {
				break
			}

			// Get text
			if len(response.Choices) > 0 {
				text := response.Choices[0].Text
				if text != "" {
					result.WriteString(text)
					tokenCount++
				}
			}
		case <-time.After(120 * time.Second):
			predictErr = fmt.Errorf("prediction timeout")
			goto done
		}
	}

done:
	if predictErr != nil {
		log.Error("Llama prediction with options failed", map[string]interface{}{
			"error":           predictErr.Error(),
			"elapsed_seconds": int(time.Since(predictionStartTime).Seconds()),
		})
		return "", fmt.Errorf("prediction failed: %w", predictErr)
	}

	log.Debug("Llama prediction with options completed", map[string]interface{}{
		"total_tokens":      tokenCount,
		"elapsed_seconds":   int(time.Since(predictionStartTime).Seconds()),
		"tokens_per_second": float64(tokenCount) / time.Since(predictionStartTime).Seconds(),
		"output_length":     result.Len(),
	})

	return result.String(), nil
}

// buildPrompt builds the prompt for a task
func (le *LlamaEngine) buildPrompt(task *Task) string {
	var promptBuilder strings.Builder

	// === SECTION 1: SYSTEM INSTRUCTIONS ===
	promptBuilder.WriteString("=== SYSTEM INSTRUCTIONS ===\n\n")
	promptBuilder.WriteString(le.systemPrompt)
	promptBuilder.WriteString("\n\n")

	// === SECTION 1.5: SYSTEM CONTEXT ===
	osInfo := runtime.GOOS
	archInfo := runtime.GOARCH
	promptBuilder.WriteString("=== SYSTEM CONTEXT ===\n\n")
	promptBuilder.WriteString(fmt.Sprintf("Target System: %s/%s\n", osInfo, archInfo))
	promptBuilder.WriteString("\nIMPORTANT: Use commands appropriate for this operating system:\n")
	switch osInfo {
	case "windows":
		promptBuilder.WriteString("- This is a Windows system\n")
		promptBuilder.WriteString("- Use PowerShell/cmd commands: systeminfo, whoami, hostname, ipconfig, tasklist, Get-ComputerInfo, etc.\n")
		promptBuilder.WriteString("- PowerShell-specific: Get-ComputerInfo, Get-Process, Get-Service\n")
		promptBuilder.WriteString("- AVOID cmd-only commands like 'ver' (not available in PowerShell)\n")
		promptBuilder.WriteString("- Do NOT use Unix commands like: uname, sw_vers, ps, ls, etc.\n")
	case "darwin":
		promptBuilder.WriteString("- This is a macOS (Darwin) system\n")
		promptBuilder.WriteString("- Use macOS commands: sw_vers, whoami, hostname, ifconfig, ps, etc.\n")
		promptBuilder.WriteString("- Do NOT use Windows commands like: systeminfo, ipconfig /all, etc.\n")
	case "linux":
		promptBuilder.WriteString("- This is a Linux system\n")
		promptBuilder.WriteString("- Use Linux commands: uname, whoami, hostname, ip addr, ps, etc.\n")
		promptBuilder.WriteString("- Do NOT use Windows commands like: systeminfo, ipconfig /all, etc.\n")
	default:
		promptBuilder.WriteString(fmt.Sprintf("- Unknown OS: %s\n", osInfo))
		promptBuilder.WriteString("- Try cross-platform commands first: hostname, whoami, etc.\n")
	}
	promptBuilder.WriteString("\n")

	// === SECTION 2: TASK DEFINITION ===
	promptBuilder.WriteString("=== TASK DEFINITION ===\n\n")

	if task.Prompt != "" {
		// Interactive mode with user-provided prompt
		promptBuilder.WriteString(fmt.Sprintf("User Request: %s\n", task.Prompt))
	} else if taskPrompt, ok := le.config.TaskPrompts[task.Type]; ok {
		// Task-specific prompt from configuration
		promptBuilder.WriteString(taskPrompt)
		promptBuilder.WriteString("\n\n")
		if task.Description != "" {
			promptBuilder.WriteString(fmt.Sprintf("Additional Context: %s\n", task.Description))
		}
	} else {
		// Fallback: generic task description
		promptBuilder.WriteString(fmt.Sprintf("Task Type: %s\n", task.Type))
		if task.Description != "" {
			promptBuilder.WriteString(fmt.Sprintf("Description: %s\n", task.Description))
		}
	}

	// === SECTION 3: EXECUTION REMINDER ===
	promptBuilder.WriteString("\n=== EXECUTION REMINDER ===\n\n")
	promptBuilder.WriteString("Follow the ReAct pattern:\n")
	promptBuilder.WriteString("1. THOUGHT: Plan your next action\n")
	promptBuilder.WriteString("2. ACTION: [EXECUTE: actual_command_here]\n")
	promptBuilder.WriteString("3. Wait for OBSERVATION\n")
	promptBuilder.WriteString("4. THOUGHT: Analyze results\n")
	promptBuilder.WriteString("5. Repeat until [TASK_COMPLETE]\n\n")

	promptBuilder.WriteString("EXAMPLES:\n")
	promptBuilder.WriteString("Correct: [EXECUTE: uname -s]\n")
	promptBuilder.WriteString("Correct: [EXECUTE: whoami]\n")
	promptBuilder.WriteString("Wrong: [EXECUTE: command] (too generic)\n")
	promptBuilder.WriteString("Wrong: [EXECUTE: next_command] (placeholder)\n\n")

	// === SECTION 4: BEGIN EXECUTION ===
	promptBuilder.WriteString("=== BEGIN EXECUTION ===\n\n")
	promptBuilder.WriteString("Start now with your first thought and action.\n")

	return promptBuilder.String()
}

// extractCommands extracts commands from text using multiple patterns
func (le *LlamaEngine) extractCommands(text string) []string {
	log := logger.Get()
	commands := make([]string, 0)
	seen := make(map[string]bool) // prevent duplicates

	// pattern 1: [EXECUTE: command] (recommended format)
	pattern1 := regexp.MustCompile(`\[EXECUTE:\s*([^\]]+)\]`)
	matches1 := pattern1.FindAllStringSubmatch(text, -1)
	for _, match := range matches1 {
		if len(match) > 1 {
			cmd := strings.TrimSpace(match[1])

			// clean special tokens before the command (known tokens only)
			cmd = le.cleanSpecialTokens(cmd)

			// filter invalid commands
			if le.isInvalidCommand(cmd) {
				log.Debug("Command filtered (invalid)", map[string]interface{}{
					"command": cmd,
					"reason":  "generic placeholder or invalid",
				})
				continue
			}

			if !seen[cmd] {
				commands = append(commands, cmd)
				seen[cmd] = true
				log.Debug("Command extracted (pattern1)", map[string]interface{}{
					"command": cmd,
					"pattern": "[EXECUTE: ...]",
				})
			}
		}
	}

	// pattern 2: Action: command (alternative format)
	pattern2 := regexp.MustCompile(`(?i)Action:\s*(.+?)(?:\n|$)`)
	matches2 := pattern2.FindAllStringSubmatch(text, -1)
	for _, match := range matches2 {
		if len(match) > 1 {
			actionText := strings.TrimSpace(match[1])

			// skip if [EXECUTE:] format is included (pattern 1 already processed)
			if strings.Contains(actionText, "[EXECUTE:") {
				continue
			}

			// clean special tokens before the command
			actionText = le.cleanSpecialTokens(actionText)

			// split by dot and get only the first sentence (if multiple sentences are included)
			if idx := strings.Index(actionText, ". "); idx != -1 {
				actionText = strings.TrimSpace(actionText[:idx])
			}

			// exclude text that is clearly not a command
			if len(actionText) > 0 && !strings.HasPrefix(actionText, "I ") &&
				!strings.HasPrefix(actionText, "The ") && !strings.HasPrefix(actionText, "We ") &&
				!strings.HasPrefix(actionText, "This ") {

				// filter invalid commands
				if le.isInvalidCommand(actionText) {
					log.Debug("Command filtered (invalid)", map[string]interface{}{
						"command": actionText,
						"reason":  "generic placeholder or invalid",
					})
					continue
				}

				if !seen[actionText] {
					commands = append(commands, actionText)
					seen[actionText] = true
					log.Debug("Command extracted (pattern2)", map[string]interface{}{
						"command": actionText,
						"pattern": "Action: ...",
					})
				}
			}
		}
	}

	// pattern 3: ```bash ... ``` (code block)
	pattern3 := regexp.MustCompile("```(?:bash|sh|shell)?\\s*\n?([^`]+)```")
	matches3 := pattern3.FindAllStringSubmatch(text, -1)
	for _, match := range matches3 {
		if len(match) > 1 {
			cmd := strings.TrimSpace(match[1])
			if !seen[cmd] {
				commands = append(commands, cmd)
				seen[cmd] = true
				log.Debug("Command extracted (pattern3)", map[string]interface{}{
					"command": cmd,
					"pattern": "```bash ... ```",
				})
			}
		}
	}

	// pattern 4: $ command (shell prompt format)
	pattern4 := regexp.MustCompile(`(?m)^\$\s+(.+)$`)
	matches4 := pattern4.FindAllStringSubmatch(text, -1)
	for _, match := range matches4 {
		if len(match) > 1 {
			cmd := strings.TrimSpace(match[1])
			if !seen[cmd] {
				commands = append(commands, cmd)
				seen[cmd] = true
				log.Debug("Command extracted (pattern4)", map[string]interface{}{
					"command": cmd,
					"pattern": "$ ...",
				})
			}
		}
	}

	if len(commands) == 0 {
		log.Debug("No commands extracted from response", map[string]interface{}{
			"text_length": len(text),
			"text_preview": func() string {
				preview := text
				if len(preview) > 200 {
					preview = preview[:200] + "..."
				}
				return preview
			}(),
		})
	} else {
		log.Debug("Commands extraction complete", map[string]interface{}{
			"total_commands": len(commands),
			"commands":       commands,
		})
	}

	return commands
}

// cleanSpecialTokens removes special tokens from the command string
// Returns the cleaned string, truncated before the first special token
func (le *LlamaEngine) cleanSpecialTokens(text string) string {
	// known special tokens list
	specialTokens := []string{
		"<|end|>",
		"<|start|>",
		"<|channel|>",
		"<|message|>",
		"<|assistant|>",
		"<|user|>",
		"<|system|>",
	}

	// find the position of the first special token
	minIdx := -1
	for _, token := range specialTokens {
		if idx := strings.Index(text, token); idx != -1 {
			if minIdx == -1 || idx < minIdx {
				minIdx = idx
			}
		}
	}

	// clean special tokens before the command
	if minIdx != -1 {
		text = strings.TrimSpace(text[:minIdx])
	}

	// remove trailing dot (not needed as a command)
	text = strings.TrimSuffix(text, ".")

	return text
}

// isInvalidCommand checks if a command is a generic placeholder or invalid
func (le *LlamaEngine) isInvalidCommand(cmd string) bool {
	// empty or very short
	if len(cmd) < 2 {
		return true
	}

	// very long (500 characters or more is suspicious)
	if len(cmd) > 500 {
		return true
	}

	// [TASK_COMPLETE] is not a command (completion signal)
	if strings.Contains(strings.ToUpper(cmd), "TASK_COMPLETE") {
		return true
	}

	// generic placeholders
	genericPlaceholders := []string{
		"command",
		"next_command",
		"your_command",
		"actual_command",
		"...",
		"<command>",
		"[command]",
		"${command}",
	}

	cmdLower := strings.ToLower(cmd)
	for _, placeholder := range genericPlaceholders {
		if cmdLower == placeholder || strings.HasPrefix(cmdLower, placeholder+" ") {
			return true
		}
	}

	// suspicious patterns (part of a sentence or explanation)
	suspiciousPatterns := []string{
		"<after receiving observation>",
		"the observation is inserted",
		"we respond with",
		"let's produce",
		"we need to",
		"according to instructions",
		"then sysctl",
		"let's proceed",
	}

	for _, pattern := range suspiciousPatterns {
		if strings.Contains(cmdLower, pattern) {
			return true
		}
	}

	// check redundant prefixes ("run sw_vers" -> "sw_vers" should be)
	redundantPrefixes := []string{
		"run ",
		"execute ",
		"please ",
	}

	for _, prefix := range redundantPrefixes {
		if strings.HasPrefix(cmdLower, prefix) {
			return true
		}
	}

	return false
}

// generateFinalReport generates a concise final report from execution logs
func (le *LlamaEngine) generateFinalReport(task *Task, result *TaskResult, rawOutput string, contextMgr *MultiContextManager) (string, error) {
	log := logger.Get()

	// Build summary of commands executed (deduplicate: keep only latest execution of each unique command)
	var commandSummary strings.Builder
	uniqueCommands := make(map[string]int) // command -> last index
	for i, cmd := range result.Commands {
		uniqueCommands[cmd.Command] = i
	}

	// Collect unique commands with their last execution index
	type cmdWithIndex struct {
		cmd   CommandExecution
		index int
	}
	var uniqueCmdList []cmdWithIndex
	for _, i := range uniqueCommands {
		uniqueCmdList = append(uniqueCmdList, cmdWithIndex{result.Commands[i], i})
	}

	// Sort by original index to maintain execution order
	sort.Slice(uniqueCmdList, func(i, j int) bool {
		return uniqueCmdList[i].index < uniqueCmdList[j].index
	})

	// Build summary with deduplicated commands
	for i, item := range uniqueCmdList {
		commandSummary.WriteString(fmt.Sprintf("%d. %s\n", i+1, item.cmd.Command))
	}

	log.Debug("Command deduplication", map[string]interface{}{
		"total_commands":  len(result.Commands),
		"unique_commands": len(uniqueCmdList),
	})

	// Get extracted facts from context manager
	var factsSummary strings.Builder
	extractedFacts := contextMgr.GetExtractedFacts()
	if len(extractedFacts) > 0 {
		for _, fact := range extractedFacts {
			factsSummary.WriteString("- " + fact + "\n")
		}
	}

	// Clean and limit raw output for report generation
	cleanedOutput := le.cleanSpecialTokens(rawOutput)

	// Further clean: remove excessive repetitions
	cleanedOutput = strings.ReplaceAll(cleanedOutput, "We.", "")
	cleanedOutput = strings.ReplaceAll(cleanedOutput, "Ok.", "")

	// Further limit to last 4000 characters to avoid token limits
	if len(cleanedOutput) > 4000 {
		cleanedOutput = "...\n" + cleanedOutput[len(cleanedOutput)-4000:]
	}

	// Build report generation prompt with extracted facts
	reportPrompt := fmt.Sprintf(`Based on the following information, generate a CONCISE final report answering the user's question.

USER QUESTION: %s

KEY FACTS DISCOVERED:
%s

COMMANDS EXECUTED:
%s

EXECUTION LOG (for additional context):
%s

Generate a clear, concise report that:
1. Directly answers the user's question using the KEY FACTS
2. Summarizes key findings
3. Is well-formatted and easy to read
4. Does NOT include execution details or process logs
5. Is 5-15 lines maximum

FINAL REPORT:`, task.Prompt, factsSummary.String(), commandSummary.String(), cleanedOutput)

	log.Debug("Generating final report", map[string]interface{}{
		"task_id":      task.ID,
		"prompt_len":   len(reportPrompt),
		"commands_run": len(result.Commands),
	})

	// Use lower temperature for more focused report
	originalTemp := le.config.Temperature
	le.config.Temperature = 0.3
	defer func() { le.config.Temperature = originalTemp }()

	// Generate report with timeout
	report, err := le.predict(reportPrompt)
	if err != nil {
		return "", fmt.Errorf("failed to generate report: %w", err)
	}

	// Clean up report - remove special tokens and trim
	report = le.cleanSpecialTokens(report)
	report = strings.TrimSpace(report)

	// If report generation failed or produced garbage, fallback
	if len(report) < 10 || strings.Count(report, "…") > 20 {
		log.Warn("Report generation produced invalid output, using fallback", map[string]interface{}{
			"task_id":    task.ID,
			"report_len": len(report),
		})

		// Enhanced fallback: include extracted facts and commands
		var fallback strings.Builder
		fallback.WriteString(fmt.Sprintf("Task: %s\n\n", task.Prompt))

		// Include extracted facts if available
		if len(extractedFacts) > 0 {
			fallback.WriteString("Key Findings:\n")
			for _, fact := range extractedFacts {
				fallback.WriteString(fmt.Sprintf("- %s\n", fact))
			}
			fallback.WriteString("\n")
		}

		fallback.WriteString(fmt.Sprintf("Commands executed: %d\n", len(result.Commands)))
		for i, cmd := range result.Commands {
			fallback.WriteString(fmt.Sprintf("%d. %s\n", i+1, cmd))
		}
		return fallback.String(), nil
	}

	log.Debug("Final report generated", map[string]interface{}{
		"task_id":      task.ID,
		"report_len":   len(report),
		"report_lines": len(strings.Split(report, "\n")),
	})

	return report, nil
}

// extractFindings extracts findings from text
func (le *LlamaEngine) extractFindings(text string) map[string]interface{} {
	findings := make(map[string]interface{})
	lines := strings.Split(text, "\n")

	for _, line := range lines {
		if strings.Contains(line, "[FINDING:") {
			start := strings.Index(line, "[FINDING:") + 9
			end := strings.Index(line[start:], "]")
			if end > 0 {
				finding := strings.TrimSpace(line[start : start+end])
				parts := strings.SplitN(finding, ":", 2)
				if len(parts) == 2 {
					key := strings.TrimSpace(parts[0])
					value := strings.TrimSpace(parts[1])
					findings[key] = value
				}
			}
		}
	}

	return findings
}
