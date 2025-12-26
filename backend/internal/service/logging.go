package service

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
	"time"
)

// LogLevel represents different logging levels
type LogLevel int

const (
	DebugLevel LogLevel = iota
	InfoLevel
	WarnLevel
	ErrorLevel
	FatalLevel
)

// LogEntry represents a structured log entry
type LogEntry struct {
	Timestamp time.Time              `json:"timestamp"`
	Level     LogLevel               `json:"level"`
	Message   string                 `json:"message"`
	Component string                 `json:"component"`
	Operation string                 `json:"operation"`
	Error     string                 `json:"error,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// LoggerService provides structured logging capabilities
type LoggerService struct {
	mu         sync.RWMutex
	logLevel   LogLevel
	components map[string]LogLevel
	file       *os.File
	stdout     bool
}

// NewLoggerService creates a new logger service
func NewLoggerService(level string, logFile string, stdout bool) (*LoggerService, error) {
	logger := &LoggerService{
		logLevel:   LogLevelFromString(level),
		components: make(map[string]LogLevel),
		stdout:     stdout,
	}

	if logFile != "" {
		file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			return nil, fmt.Errorf("failed to open log file: %w", err)
		}
		logger.file = file
	}

	return logger, nil
}

// SetLogLevel sets the global log level
func (ls *LoggerService) SetLogLevel(level LogLevel) {
	ls.mu.Lock()
	defer ls.mu.Unlock()
	ls.logLevel = level
}

// SetComponentLogLevel sets the log level for a specific component
func (ls *LoggerService) SetComponentLogLevel(component string, level LogLevel) {
	ls.mu.Lock()
	defer ls.mu.Unlock()
	ls.components[component] = level
}

// ShouldLog checks if a log entry should be logged based on level and component
func (ls *LoggerService) ShouldLog(level LogLevel, component string) bool {
	ls.mu.RLock()
	defer ls.mu.RUnlock()

	// Check component-specific level first
	if componentLevel, exists := ls.components[component]; exists {
		return level >= componentLevel
	}

	// Fall back to global level
	return level >= ls.logLevel
}

// Debug logs a debug message
func (ls *LoggerService) Debug(component, operation, message string, metadata map[string]interface{}) {
	ls.log(DebugLevel, component, operation, message, "", metadata)
}

// Info logs an info message
func (ls *LoggerService) Info(component, operation, message string, metadata map[string]interface{}) {
	ls.log(InfoLevel, component, operation, message, "", metadata)
}

// Warn logs a warning message
func (ls *LoggerService) Warn(component, operation, message string, metadata map[string]interface{}) {
	ls.log(WarnLevel, component, operation, message, "", metadata)
}

// Error logs an error message
func (ls *LoggerService) Error(component, operation, message string, err error, metadata map[string]interface{}) {
	errorMsg := ""
	if err != nil {
		errorMsg = err.Error()
	}
	ls.log(ErrorLevel, component, operation, message, errorMsg, metadata)
}

// Fatal logs a fatal message and exits
func (ls *LoggerService) Fatal(component, operation, message string, err error, metadata map[string]interface{}) {
	errorMsg := ""
	if err != nil {
		errorMsg = err.Error()
	}
	ls.log(FatalLevel, component, operation, message, errorMsg, metadata)
	os.Exit(1)
}

// log is the internal logging method
func (ls *LoggerService) log(level LogLevel, component, operation, message, errorMsg string, metadata map[string]interface{}) {
	if !ls.ShouldLog(level, component) {
		return
	}

	entry := LogEntry{
		Timestamp: time.Now(),
		Level:     level,
		Message:   message,
		Component: component,
		Operation: operation,
		Error:     errorMsg,
		Metadata:  metadata,
	}

	// Create log output
	logOutput, err := json.Marshal(entry)
	if err != nil {
		// Fallback to simple logging if JSON marshaling fails
		log.Printf("[%s] %s - %s: %s", level.String(), component, operation, message)
		return
	}

	// Write to file if configured
	if ls.file != nil {
		ls.file.WriteString(string(logOutput) + "\n")
	}

	// Write to stdout if configured
	if ls.stdout {
		fmt.Println(string(logOutput))
	}
}

// LogSystemEvent logs system-wide events
func (ls *LoggerService) LogSystemEvent(eventType, message string, metadata map[string]interface{}) {
	ls.Info("system", eventType, message, metadata)
}

// LogPerformance logs performance metrics
func (ls *LoggerService) LogPerformance(component, operation string, duration time.Duration, metadata map[string]interface{}) {
	if metadata == nil {
		metadata = make(map[string]interface{})
	}
	metadata["duration_ms"] = duration.Milliseconds()
	ls.Debug(component, operation, "Performance metric", metadata)
}

// LogError logs an error with context
func (ls *LoggerService) LogError(component, operation string, err error, context map[string]interface{}) {
	ls.Error(component, operation, err.Error(), err, context)
}

// LogSecurity logs security-related events
func (ls *LoggerService) LogSecurity(eventType, message string, metadata map[string]interface{}) {
	ls.Warn("security", eventType, message, metadata)
}

// LogAudit logs audit trail events
func (ls *LoggerService) LogAudit(user, action, resource string, success bool, metadata map[string]interface{}) {
	if metadata == nil {
		metadata = make(map[string]interface{})
	}
	metadata["user"] = user
	metadata["action"] = action
	metadata["resource"] = resource
	metadata["success"] = success

	status := "success"
	if !success {
		status = "failed"
	}

	ls.Info("audit", "access", fmt.Sprintf("User %s %s %s on %s", user, status, action, resource), metadata)
}

// Close closes the logger
func (ls *LoggerService) Close() error {
	if ls.file != nil {
		return ls.file.Close()
	}
	return nil
}

// LogLevelString converts LogLevel to string
func (level LogLevel) String() string {
	switch level {
	case DebugLevel:
		return "DEBUG"
	case InfoLevel:
		return "INFO"
	case WarnLevel:
		return "WARN"
	case ErrorLevel:
		return "ERROR"
	case FatalLevel:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

// LogLevelFromString converts string to LogLevel
func LogLevelFromString(level string) LogLevel {
	switch level {
	case "debug":
		return DebugLevel
	case "info":
		return InfoLevel
	case "warn":
		return WarnLevel
	case "error":
		return ErrorLevel
	case "fatal":
		return FatalLevel
	default:
		return InfoLevel
	}
}

// ErrorHandler provides centralized error handling
type ErrorHandler struct {
	logger *LoggerService
}

// NewErrorHandler creates a new error handler
func NewErrorHandler(logger *LoggerService) *ErrorHandler {
	return &ErrorHandler{
		logger: logger,
	}
}

// HandleError handles errors with appropriate logging
func (eh *ErrorHandler) HandleError(component, operation string, err error, metadata map[string]interface{}) error {
	if err == nil {
		return nil
	}

	eh.logger.LogError(component, operation, err, metadata)
	return err
}

// HandlePanic recovers from panics and logs them
func (eh *ErrorHandler) HandlePanic(component, operation string) {
	if r := recover(); r != nil {
		err, ok := r.(error)
		if !ok {
			err = fmt.Errorf("%v", r)
		}
		eh.logger.Fatal(component, operation, "Panic recovered", err, nil)
	}
}

// ValidateInput validates input parameters
func (eh *ErrorHandler) ValidateInput(component, operation string, params map[string]interface{}) error {
	if params == nil {
		return eh.HandleError(component, operation, fmt.Errorf("nil parameters"), nil)
	}
	return nil
}

// WrapError wraps an error with additional context
func (eh *ErrorHandler) WrapError(component, operation, context string, err error) error {
	if err == nil {
		return nil
	}

	wrappedErr := fmt.Errorf("%s: %w", context, err)
	eh.logger.LogError(component, operation, wrappedErr, map[string]interface{}{
		"original_error": err.Error(),
		"context":        context,
	})
	return wrappedErr
}
