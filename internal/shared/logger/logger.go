package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"sync"
)

// LogLevel represents the logging level
type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
	OFF
)

// String returns the string representation of the log level
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
	case OFF:
		return "OFF"
	default:
		return "UNKNOWN"
	}
}

// ParseLogLevel parses a string into a LogLevel
func ParseLogLevel(level string) LogLevel {
	switch strings.ToUpper(level) {
	case "DEBUG":
		return DEBUG
	case "INFO":
		return INFO
	case "WARN", "WARNING":
		return WARN
	case "ERROR":
		return ERROR
	case "OFF", "NONE":
		return OFF
	default:
		return INFO
	}
}

// Logger interface defines the common logging methods
type Logger interface {
	Debug(format string, args ...interface{})
	Info(format string, args ...interface{})
	Warn(format string, args ...interface{})
	Error(format string, args ...interface{})
	SetLevel(level LogLevel)
	GetLevel() LogLevel
	SetOutput(w io.Writer)
}

// DefaultLogger implements the Logger interface
type DefaultLogger struct {
	mu          sync.RWMutex
	level       LogLevel
	logger      *log.Logger
	prefix      string
	jsonlWriter *JSONLWriter
}

// NewLogger creates a new logger with the given prefix
func NewLogger(prefix string) *DefaultLogger {
	return &DefaultLogger{
		level:  INFO,
		logger: log.New(os.Stdout, "", log.LstdFlags),
		prefix: prefix,
	}
}

// SetLevel sets the logging level
func (l *DefaultLogger) SetLevel(level LogLevel) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
}

// GetLevel returns the current logging level
func (l *DefaultLogger) GetLevel() LogLevel {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.level
}

// SetOutput sets the output writer
func (l *DefaultLogger) SetOutput(w io.Writer) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.logger.SetOutput(w)
}

// log is the internal logging method
func (l *DefaultLogger) log(level LogLevel, format string, args ...interface{}) {
	l.mu.RLock()
	currentLevel := l.level
	jsonlWriter := l.jsonlWriter
	l.mu.RUnlock()

	if level < currentLevel || currentLevel == OFF {
		return
	}

	msg := fmt.Sprintf(format, args...)

	// Write to JSONL if writer is set
	if jsonlWriter != nil {
		jsonlWriter.Write(level, l.prefix, msg, nil)
	}

	// For INFO level, output without timestamp and prefix
	if currentLevel >= INFO && level == INFO {
		fmt.Print(msg)
		if len(msg) == 0 || msg[len(msg)-1] != '\n' {
			fmt.Println()
		}
	} else if level == WARN {
		// For WARN, use yellow color
		fmt.Printf("\033[33m[%s] %s\033[0m", level.String(), msg)
		if len(msg) == 0 || msg[len(msg)-1] != '\n' {
			fmt.Println()
		}
	} else if level == ERROR {
		// For ERROR, use red color
		fmt.Printf("\033[31m[%s] %s\033[0m", level.String(), msg)
		if len(msg) == 0 || msg[len(msg)-1] != '\n' {
			fmt.Println()
		}
	} else {
		// For DEBUG level, include timestamp and prefix
		prefix := fmt.Sprintf("[%s] %s: ", level.String(), l.prefix)
		l.logger.Printf("%s%s", prefix, msg)
	}
}

// Debug logs a debug message
func (l *DefaultLogger) Debug(format string, args ...interface{}) {
	l.log(DEBUG, format, args...)
}

// Info logs an info message
func (l *DefaultLogger) Info(format string, args ...interface{}) {
	l.log(INFO, format, args...)
}

// Warn logs a warning message
func (l *DefaultLogger) Warn(format string, args ...interface{}) {
	l.log(WARN, format, args...)
}

// Error logs an error message
func (l *DefaultLogger) Error(format string, args ...interface{}) {
	l.log(ERROR, format, args...)
}

// SetJSONLWriter sets the JSONL writer for this logger
func (l *DefaultLogger) SetJSONLWriter(writer *JSONLWriter) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.jsonlWriter = writer
}

// LogWithFields logs a message with additional fields (for JSONL)
func (l *DefaultLogger) LogWithFields(level LogLevel, message string, fields map[string]interface{}) {
	l.mu.RLock()
	currentLevel := l.level
	jsonlWriter := l.jsonlWriter
	l.mu.RUnlock()

	if level < currentLevel || currentLevel == OFF {
		return
	}

	// Write to JSONL with fields
	if jsonlWriter != nil {
		jsonlWriter.Write(level, l.prefix, message, fields)
	}

	// Also write to regular log
	l.log(level, message)
}

// GlobalLogger is the default global logger
var (
	globalLogger Logger = NewLogger("virga")
	globalMu     sync.RWMutex
)

// SetGlobalLogger sets the global logger
func SetGlobalLogger(logger Logger) {
	globalMu.Lock()
	defer globalMu.Unlock()
	globalLogger = logger
}

// GetGlobalLogger returns the global logger
func GetGlobalLogger() Logger {
	globalMu.RLock()
	defer globalMu.RUnlock()
	return globalLogger
}

// SetGlobalLogLevel sets the global log level
func SetGlobalLogLevel(level LogLevel) {
	globalMu.RLock()
	logger := globalLogger
	globalMu.RUnlock()
	logger.SetLevel(level)
}

// SetGlobalLogLevelFromString sets the global log level from a string
func SetGlobalLogLevelFromString(level string) {
	SetGlobalLogLevel(ParseLogLevel(level))
}

// SetGlobalLogLevelFromEnv sets the global log level from environment variable
func SetGlobalLogLevelFromEnv(envVar string) {
	if level := os.Getenv(envVar); level != "" {
		SetGlobalLogLevelFromString(level)
	}
}

// Helper functions for global logger
func Debug(format string, args ...interface{}) {
	GetGlobalLogger().Debug(format, args...)
}

func Info(format string, args ...interface{}) {
	GetGlobalLogger().Info(format, args...)
}

func Warn(format string, args ...interface{}) {
	GetGlobalLogger().Warn(format, args...)
}

func Error(format string, args ...interface{}) {
	GetGlobalLogger().Error(format, args...)
}

// SetGlobalJSONLLogger sets up JSONL logging for the global logger
func SetGlobalJSONLLogger(logPath string, level LogLevel) error {
	writer, err := NewJSONLWriter(logPath, level)
	if err != nil {
		return err
	}

	// Try to set JSONL writer on the global logger if it's a DefaultLogger
	if dl, ok := GetGlobalLogger().(*DefaultLogger); ok {
		dl.SetJSONLWriter(writer)
	}

	return nil
}

// LogWithFields logs a message with additional fields using the global logger
func LogWithFields(level LogLevel, message string, fields map[string]interface{}) {
	if dl, ok := GetGlobalLogger().(*DefaultLogger); ok {
		dl.LogWithFields(level, message, fields)
	} else {
		// Fallback to regular logging
		switch level {
		case DEBUG:
			Debug(message)
		case INFO:
			Info(message)
		case WARN:
			Warn(message)
		case ERROR:
			Error(message)
		}
	}
}
