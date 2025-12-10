package llama

import (
	"fmt"
	"strings"

	"github.com/r74tech/virga/internal/implant/memdb"
)

// MultiContextManager manages multiple parallel contexts for LLM-based ReAct execution
// It combines LLM-based extraction with hierarchical context management
type MultiContextManager struct {
	// Context 1: Full history in MemDB (reference only)
	memdb  *memdb.DB
	taskID string

	// Context 2: Extracted key facts (LLM-generated)
	extractedFacts []string

	// Context 3: Recent detailed observations (last N)
	recentObservations []DetailedObservation
	maxRecent          int // Maximum number of recent observations to keep

	// Context 4: Compressed summary (for older observations)
	compressedSummary string

	// Context 5: Command history (quick reference)
	commandHistory []string

	// Current iteration counter
	currentIteration int

	// LLM predictor function
	llmPredict func(string, PredictOptions) (string, error)

	// Base prompt (task description)
	basePrompt string
}

// DetailedObservation represents a recent observation with full details
type DetailedObservation struct {
	Command  string
	Output   string // Summarized if too long
	ExitCode int
	KeyInfo  string // LLM-extracted key information
}

// PredictOptions contains options for LLM prediction
type PredictOptions struct {
	Temperature float32
	MaxTokens   int
}

// NewMultiContextManager creates a new multi-context manager
func NewMultiContextManager(
	db *memdb.DB,
	taskID string,
	basePrompt string,
	maxRecent int,
	llmPredict func(string, PredictOptions) (string, error),
) *MultiContextManager {
	if maxRecent == 0 {
		maxRecent = 2 // Default: keep 2 most recent observations
	}

	return &MultiContextManager{
		memdb:              db,
		taskID:             taskID,
		basePrompt:         basePrompt,
		extractedFacts:     make([]string, 0),
		recentObservations: make([]DetailedObservation, 0, maxRecent),
		commandHistory:     make([]string, 0),
		maxRecent:          maxRecent,
		currentIteration:   0,
		llmPredict:         llmPredict,
	}
}

// AddObservation adds a new observation and manages context automatically
func (mcm *MultiContextManager) AddObservation(cmd string, output string, exitCode int) error {
	mcm.currentIteration++

	// 1. Extract key information using LLM
	keyInfo := mcm.extractKeyInformation(cmd, output, exitCode)

	// 2. Store full observation in MemDB for complete history
	obs := &memdb.LlamaObservation{
		TaskID:       mcm.taskID,
		Iteration:    mcm.currentIteration,
		Command:      cmd,
		Output:       output,
		ExitCode:     exitCode,
		ExtractedKey: keyInfo,
	}

	if err := mcm.memdb.StoreLlamaObservation(obs); err != nil {
		// Log error but continue - MemDB storage is not critical for execution
		fmt.Printf("Warning: failed to store observation in MemDB: %v\n", err)
	}

	// 3. Add to extracted facts (if meaningful)
	if keyInfo != "" && keyInfo != "No key information." {
		mcm.extractedFacts = append(mcm.extractedFacts, keyInfo)
	}

	// 4. Manage sliding window for recent observations
	summarized := summarizeIfLong(output, 20)

	detailedObs := DetailedObservation{
		Command:  cmd,
		Output:   summarized,
		ExitCode: exitCode,
		KeyInfo:  keyInfo,
	}

	if len(mcm.recentObservations) >= mcm.maxRecent {
		// Compress oldest observation before removing
		oldest := mcm.recentObservations[0]
		mcm.compressOldestObservation(oldest)

		// Shift window
		mcm.recentObservations = mcm.recentObservations[1:]
	}

	mcm.recentObservations = append(mcm.recentObservations, detailedObs)

	// 5. Add to command history
	mcm.commandHistory = append(mcm.commandHistory, cmd)

	return nil
}

// BuildPrompt constructs the full prompt from managed contexts
func (mcm *MultiContextManager) BuildPrompt() string {
	var builder strings.Builder

	// 1. Base prompt (task description)
	builder.WriteString(mcm.basePrompt)
	builder.WriteString("\n\n")

	// 2. Extracted facts (most important - always visible)
	if len(mcm.extractedFacts) > 0 {
		builder.WriteString("=== KEY FACTS DISCOVERED ===\n")
		for _, fact := range mcm.extractedFacts {
			builder.WriteString("- " + fact + "\n")
		}
		builder.WriteString("\n")
	}

	// 3. Command history (what was executed)
	if len(mcm.commandHistory) > 0 {
		builder.WriteString("=== COMMANDS EXECUTED ===\n")
		for i, cmd := range mcm.commandHistory {
			builder.WriteString(fmt.Sprintf("%d. %s\n", i+1, cmd))
		}
		builder.WriteString("\n")
	}

	// 4. Compressed summary (if exists)
	if mcm.compressedSummary != "" {
		builder.WriteString("=== EARLIER PROGRESS ===\n")
		builder.WriteString(mcm.compressedSummary)
		builder.WriteString("\n\n")
	}

	// 5. Recent observations (detailed)
	if len(mcm.recentObservations) > 0 {
		builder.WriteString("=== RECENT OBSERVATIONS ===\n")
		for i, obs := range mcm.recentObservations {
			builder.WriteString(fmt.Sprintf("Observation %d:\n", i+1))
			builder.WriteString(fmt.Sprintf("Command: %s\n", obs.Command))
			if obs.KeyInfo != "" {
				builder.WriteString(fmt.Sprintf("Key Info: %s\n", obs.KeyInfo))
			}
			builder.WriteString(fmt.Sprintf("Output:\n%s\n", obs.Output))
			builder.WriteString(fmt.Sprintf("Exit Code: %d\n\n", obs.ExitCode))
		}
	}

	builder.WriteString("Thought:")

	return builder.String()
}

// extractKeyInformation uses LLM to extract key information from command output
func (mcm *MultiContextManager) extractKeyInformation(cmd string, output string, exitCode int) string {
	// Skip extraction for failed commands or empty output
	if exitCode != 0 || output == "" {
		return ""
	}

	// Build extraction prompt
	extractPrompt := fmt.Sprintf(`Task: %s

Latest command execution:
Command: %s
Output (first/last 10 lines):
%s
Exit Code: %d

From this output, extract ONLY the key information relevant to the task.
Provide 1-3 lines maximum, focusing on what's important for completing the task.
If nothing important, say "No key information."

Key information:`,
		mcm.basePrompt,
		cmd,
		summarizeOutput(output, 10, 10),
		exitCode)

	// Call LLM with low temperature and limited tokens
	keyInfo, err := mcm.llmPredict(extractPrompt, PredictOptions{
		Temperature: 0.1, // Low temperature for factual extraction
		MaxTokens:   100, // Short response only
	})

	if err != nil || keyInfo == "" || keyInfo == "No key information." {
		return ""
	}

	return strings.TrimSpace(keyInfo)
}

// compressOldestObservation compresses an observation using LLM
func (mcm *MultiContextManager) compressOldestObservation(obs DetailedObservation) {
	compressPrompt := fmt.Sprintf(`Summarize this observation in 1 sentence:
Command: %s
Key Info: %s
Output: %s

One sentence summary:`, obs.Command, obs.KeyInfo, truncate(obs.Output, 100))

	summary, err := mcm.llmPredict(compressPrompt, PredictOptions{
		Temperature: 0.1,
		MaxTokens:   50,
	})

	if err == nil && summary != "" {
		if mcm.compressedSummary == "" {
			mcm.compressedSummary = summary
		} else {
			mcm.compressedSummary += "\n" + summary
		}
	}
}

// GetFullHistory retrieves complete observation history from MemDB
func (mcm *MultiContextManager) GetFullHistory() ([]*memdb.LlamaObservation, error) {
	return mcm.memdb.GetLlamaObservationsByTask(mcm.taskID)
}

// GetExtractedFacts returns the extracted facts
func (mcm *MultiContextManager) GetExtractedFacts() []string {
	return mcm.extractedFacts
}

// GetStats returns statistics about the current context
func (mcm *MultiContextManager) GetStats() map[string]interface{} {
	prompt := mcm.BuildPrompt()

	return map[string]interface{}{
		"prompt_length":    len(prompt),
		"prompt_lines":     len(strings.Split(prompt, "\n")),
		"facts_count":      len(mcm.extractedFacts),
		"recent_obs_count": len(mcm.recentObservations),
		"commands_total":   len(mcm.commandHistory),
		"iteration":        mcm.currentIteration,
	}
}

// summarizeOutput extracts first N and last M lines from output
func summarizeOutput(output string, firstN, lastM int) string {
	lines := strings.Split(output, "\n")

	if len(lines) <= firstN+lastM {
		return output
	}

	var result strings.Builder

	// First N lines
	for i := 0; i < firstN && i < len(lines); i++ {
		result.WriteString(lines[i])
		result.WriteString("\n")
	}

	result.WriteString(fmt.Sprintf("\n... [%d lines omitted] ...\n\n", len(lines)-(firstN+lastM)))

	// Last M lines
	for i := len(lines) - lastM; i < len(lines); i++ {
		if i >= 0 {
			result.WriteString(lines[i])
			result.WriteString("\n")
		}
	}

	return result.String()
}

// summarizeIfLong truncates long output while preserving structure
func summarizeIfLong(output string, maxLines int) string {
	lines := strings.Split(output, "\n")

	if len(lines) <= maxLines {
		return output
	}

	// For long output, preserve head, middle sample, and tail
	head := maxLines / 3
	middle := maxLines / 3
	tail := maxLines - head - middle

	var result strings.Builder

	// Head (beginning)
	for i := 0; i < head && i < len(lines); i++ {
		result.WriteString(lines[i])
		result.WriteString("\n")
	}

	result.WriteString(fmt.Sprintf("\n... [%d lines omitted] ...\n\n", len(lines)-maxLines))

	// Middle sample (around center)
	midStart := len(lines)/2 - middle/2
	for i := midStart; i < midStart+middle && i < len(lines); i++ {
		result.WriteString(lines[i])
		result.WriteString("\n")
	}

	result.WriteString("\n... [continuing] ...\n\n")

	// Tail (end)
	for i := len(lines) - tail; i < len(lines); i++ {
		if i >= 0 {
			result.WriteString(lines[i])
			result.WriteString("\n")
		}
	}

	return result.String()
}

// truncate truncates a string to maxLen characters
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
