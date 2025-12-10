package config

// LlamaConfiguration is embedded into the implant at build time
var (
	// BuildLlamaEnabled indicates if Llama is enabled
	BuildLlamaEnabled = "false"

	// BuildLlamaContext is the context window size
	BuildLlamaContext = "8192"

	// BuildLlamaMaxTokens is the max tokens per response
	BuildLlamaMaxTokens = "2048"

	// BuildLlamaMaxIterations is the max iterations per task
	BuildLlamaMaxIterations = "50"

	// BuildLlamaTemperature controls randomness
	BuildLlamaTemperature = "0.3"

	// BuildLlamaGPULayers is the number of layers to offload to GPU
	BuildLlamaGPULayers = "0"

	// BuildLlamaSystemPrompt is the system prompt preset
	BuildLlamaSystemPrompt = "default"

	// BuildLlamaAutoMode enables autonomous mode
	BuildLlamaAutoMode = "true"

	// BuildLlamaInitialTasks contains base64 encoded JSON of initial tasks
	BuildLlamaInitialTasks = ""

	// BuildLlamaTaskPrompts contains base64 encoded JSON of task prompts
	BuildLlamaTaskPrompts = ""

	// BuildLlamaPayloadInfos contains base64 encoded JSON array of payload command information
	BuildLlamaPayloadInfos = ""

	// BuildLlamaLogEnabled controls whether Llama logs are written to file
	BuildLlamaLogEnabled = "false"
)

// GetLlamaEnabled returns whether Llama is enabled
func GetLlamaEnabled() bool {
	return BuildLlamaEnabled == "true"
}

// GetLlamaAutoMode returns whether autonomous mode is enabled
func GetLlamaAutoMode() bool {
	return BuildLlamaAutoMode == "true"
}

// GetLlamaLogEnabled returns whether Llama logging is enabled
func GetLlamaLogEnabled() bool {
	return BuildLlamaLogEnabled == "true"
}
