package logger

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/r74tech/virga/internal/implant/config"
)

// LogLevel represents the severity of log messages
type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
)

var (
	instance *Logger
	once     sync.Once
)

// Logger provides thread-safe logging with file output
type Logger struct {
	file        *os.File
	logger      *log.Logger
	level       LogLevel
	enabled     bool
	mu          sync.Mutex
	logFilePath string
}

// Get returns the singleton logger instance
func Get() *Logger {
	once.Do(func() {
		// Check environment variable first
		logLevelStr := os.Getenv("VIRGA_IMPLANT_LOG_LEVEL")
		if logLevelStr == "" {
			logLevelStr = config.GetImplantLogLevel()
		}

		instance = &Logger{
			enabled: config.GetImplantLogEnabled(),
			level:   parseLogLevel(logLevelStr),
		}

		if instance.enabled {
			if err := instance.initialize(); err != nil {
				// If we can't create the log file, disable logging but print error
				fmt.Fprintf(os.Stderr, "[ERROR] Logger initialization failed: %v\n", err)
				instance.enabled = false
			}
		} else {
			fmt.Fprintf(os.Stderr, "[INFO] Logger is disabled (BuildImplantLogEnabled=%s)\n", config.BuildImplantLogEnabled)
		}
	})
	return instance
}

// initialize sets up the log file and logger
func (l *Logger) initialize() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Get log file path
	logPath := config.GetImplantLogFilePath()

	// Debug: Print initial log path to stderr
	if os.Getenv("VIRGA_DEBUG") != "" {
		fmt.Fprintf(os.Stderr, "[DEBUG] Initial log path: %s\n", logPath)
	}

	// If path is relative, make it relative to appropriate directory
	if !filepath.IsAbs(logPath) {
		var baseDir string

		// On Windows, prefer current directory for relative paths when in Downloads or similar
		if runtime.GOOS == "windows" {
			// Get current directory
			currentDir, err := os.Getwd()
			if err == nil {
				// Check if we're in a user-writable directory (Downloads, Desktop, etc.)
				userProfile := os.Getenv("USERPROFILE")
				if userProfile != "" && strings.HasPrefix(currentDir, userProfile) {
					// Use current directory for logs
					baseDir = currentDir
					if os.Getenv("VIRGA_DEBUG") != "" {
						fmt.Fprintf(os.Stderr, "[DEBUG] Using current directory for logs: %s\n", baseDir)
					}
				} else {
					// Fall back to AppData\Local\Temp
					baseDir = filepath.Join(userProfile, "AppData", "Local", "Temp")
				}
			} else {
				// Fall back to temp directory
				baseDir = os.TempDir()
			}
		} else {
			baseDir = os.TempDir()
		}

		logPath = filepath.Join(baseDir, logPath)
		if os.Getenv("VIRGA_DEBUG") != "" {
			fmt.Fprintf(os.Stderr, "[DEBUG] Resolved relative path to: %s (base dir: %s)\n", logPath, baseDir)
		}
	}

	// Convert to clean path (handles path separators correctly for the OS)
	logPath = filepath.Clean(logPath)
	if os.Getenv("VIRGA_DEBUG") != "" {
		fmt.Fprintf(os.Stderr, "[DEBUG] Clean log path: %s\n", logPath)
	}

	// Create directory if needed
	logDir := filepath.Dir(logPath)
	if os.Getenv("VIRGA_DEBUG") != "" {
		fmt.Fprintf(os.Stderr, "[DEBUG] Creating log directory: %s\n", logDir)
	}

	// Use os.ModePerm for cross-platform compatibility
	if err := os.MkdirAll(logDir, os.ModePerm); err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] Failed to create log directory %s: %v\n", logDir, err)
		return fmt.Errorf("failed to create log directory: %v", err)
	}

	// Open log file (append mode)
	// Use os.ModePerm for cross-platform compatibility
	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o666)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] Failed to open log file %s: %v\n", logPath, err)
		// Try to provide more detailed error information
		if os.IsPermission(err) {
			fmt.Fprintf(os.Stderr, "[ERROR] Permission denied. Check write permissions for: %s\n", logDir)
		}
		if os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "[ERROR] Path does not exist: %s\n", logPath)
		}
		return fmt.Errorf("failed to open log file: %v", err)
	}

	l.file = file
	l.logFilePath = logPath
	l.logger = log.New(file, "", 0) // We'll format timestamps ourselves

	// Write initialization message
	l.writeLog(INFO, "Implant logger initialized", map[string]interface{}{
		"log_path":  logPath,
		"log_level": config.GetImplantLogLevel(),
		"pid":       os.Getpid(),
		"os":        runtime.GOOS,
		"arch":      runtime.GOARCH,
	})

	if os.Getenv("VIRGA_DEBUG") != "" {
		fmt.Fprintf(os.Stderr, "[DEBUG] Logger successfully initialized. Log file: %s\n", logPath)
	}
	return nil
}

// parseLogLevel converts string log level to LogLevel
func parseLogLevel(level string) LogLevel {
	switch strings.ToLower(level) {
	case "debug":
		return DEBUG
	case "warn", "warning":
		return WARN
	case "error":
		return ERROR
	case "off":
		return ERROR + 1 // Higher than ERROR to disable all logs
	default:
		return INFO
	}
}

// formatLogLevel converts LogLevel to string
func (l LogLevel) String() string {
	switch l {
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case WARN:
		return "WARN"
	case ERROR:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// writeLog writes a log entry to file
func (l *Logger) writeLog(level LogLevel, message string, fields map[string]interface{}) {
	if !l.enabled || level < l.level || l.logger == nil {
		return
	}

	// Format timestamp
	timestamp := time.Now().Format("2006-01-02 15:04:05.000")

	// Build log entry
	logEntry := fmt.Sprintf("[%s] [%s] %s", timestamp, level.String(), message)

	// Add fields if any
	if len(fields) > 0 {
		fieldParts := make([]string, 0, len(fields))
		for k, v := range fields {
			fieldParts = append(fieldParts, fmt.Sprintf("%s=%v", k, v))
		}
		logEntry += " | " + strings.Join(fieldParts, " ")
	}

	// Write to file
	if l.logger != nil {
		l.logger.Println(logEntry)
		// Force flush on Windows
		if l.file != nil {
			l.file.Sync()
		}
	} else {
		fmt.Fprintf(os.Stderr, "[WARNING] Logger not initialized, cannot write: %s\n", logEntry)
	}
}

// Debug logs a debug message
func (l *Logger) Debug(message string, fields ...map[string]interface{}) {
	var f map[string]interface{}
	if len(fields) > 0 {
		f = fields[0]
	}
	l.writeLog(DEBUG, message, f)
}

// Info logs an info message
func (l *Logger) Info(message string, fields ...map[string]interface{}) {
	var f map[string]interface{}
	if len(fields) > 0 {
		f = fields[0]
	}
	l.writeLog(INFO, message, f)
}

// Warn logs a warning message
func (l *Logger) Warn(message string, fields ...map[string]interface{}) {
	var f map[string]interface{}
	if len(fields) > 0 {
		f = fields[0]
	}
	l.writeLog(WARN, message, f)
}

// Error logs an error message
func (l *Logger) Error(message string, fields ...map[string]interface{}) {
	var f map[string]interface{}
	if len(fields) > 0 {
		f = fields[0]
	}
	l.writeLog(ERROR, message, f)
}

// LogCommand logs a command execution
func (l *Logger) LogCommand(command string, output string, exitCode int, err error) {
	fields := map[string]interface{}{
		"command":    command,
		"exit_code":  exitCode,
		"output_len": len(output),
	}

	if err != nil {
		fields["error"] = err.Error()
	}

	// Truncate output if too long
	if len(output) > 500 {
		fields["output_preview"] = output[:500] + "..."
	} else if output != "" {
		fields["output"] = output
	}

	l.Info("Command executed", fields)
}

// LogTask logs a task execution
func (l *Logger) LogTask(taskID string, taskType string, status string, fields ...map[string]interface{}) {
	logFields := map[string]interface{}{
		"task_id":   taskID,
		"task_type": taskType,
		"status":    status,
	}

	// Merge additional fields
	if len(fields) > 0 && fields[0] != nil {
		for k, v := range fields[0] {
			logFields[k] = v
		}
	}

	l.Info("Task "+status, logFields)
}

// LogBeacon logs a beacon communication
func (l *Logger) LogBeacon(direction string, status string, fields ...map[string]interface{}) {
	logFields := map[string]interface{}{
		"direction": direction, // "outbound" or "inbound"
		"status":    status,
	}

	// Merge additional fields
	if len(fields) > 0 && fields[0] != nil {
		for k, v := range fields[0] {
			logFields[k] = v
		}
	}

	l.Debug("Beacon communication", logFields)
}

// LogSystem logs system-related events
func (l *Logger) LogSystem(event string, fields ...map[string]interface{}) {
	var f map[string]interface{}
	if len(fields) > 0 {
		f = fields[0]
	}
	l.Info("System event: "+event, f)
}

// LogLlama logs Llama-related events
func (l *Logger) LogLlama(event string, fields ...map[string]interface{}) {
	var f map[string]interface{}
	if len(fields) > 0 {
		f = fields[0]
	}
	l.Info("Llama event: "+event, f)
}

// LogLlamaPrompt logs a Llama prompt and response
func (l *Logger) LogLlamaPrompt(taskID string, prompt string, response string, temperature float32, fields ...map[string]interface{}) {
	logFields := map[string]interface{}{
		"task_id":      taskID,
		"prompt_len":   len(prompt),
		"response_len": len(response),
		"temperature":  temperature,
	}

	// Truncate prompt and response if too long
	const maxLen = 200
	if len(prompt) > maxLen {
		logFields["prompt_preview"] = prompt[:maxLen] + "..."
	} else {
		logFields["prompt"] = prompt
	}

	if len(response) > maxLen {
		logFields["response_preview"] = response[:maxLen] + "..."
	} else {
		logFields["response"] = response
	}

	// Merge additional fields
	if len(fields) > 0 && fields[0] != nil {
		for k, v := range fields[0] {
			logFields[k] = v
		}
	}

	l.Info("Llama prompt/response", logFields)
}

// LogLlamaCommand logs a command executed by Llama
func (l *Logger) LogLlamaCommand(taskID string, command string, output string, exitCode int, err error) {
	fields := map[string]interface{}{
		"task_id":    taskID,
		"command":    command,
		"exit_code":  exitCode,
		"output_len": len(output),
	}

	if err != nil {
		fields["error"] = err.Error()
	}

	// Truncate output if too long
	if len(output) > 300 {
		fields["output_preview"] = output[:300] + "..."
	} else if output != "" {
		fields["output"] = output
	}

	l.Info("Llama command execution", fields)
}

// LogLlamaIteration logs an iteration in Llama's execution
func (l *Logger) LogLlamaIteration(taskID string, iteration int, action string, fields ...map[string]interface{}) {
	logFields := map[string]interface{}{
		"task_id":   taskID,
		"iteration": iteration,
		"action":    action,
	}

	// Merge additional fields
	if len(fields) > 0 && fields[0] != nil {
		for k, v := range fields[0] {
			logFields[k] = v
		}
	}

	l.Debug("Llama iteration", logFields)
}

// SetLevel dynamically changes the log level
func (l *Logger) SetLevel(level LogLevel) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
}

// GetLevel returns the current log level
func (l *Logger) GetLevel() LogLevel {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.level
}

// SetLevelFromString sets the log level from a string
func (l *Logger) SetLevelFromString(levelStr string) {
	level := parseLogLevel(levelStr)
	l.SetLevel(level)
}

// Close closes the log file
func (l *Logger) Close() {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.file != nil {
		l.Info("Implant logger shutting down")
		l.file.Close()
		l.file = nil
		l.logger = nil
	}
}

// GetLogFilePath returns the current log file path
func (l *Logger) GetLogFilePath() string {
	return l.logFilePath
}

// IsEnabled returns whether logging is enabled
func (l *Logger) IsEnabled() bool {
	return l.enabled
}
