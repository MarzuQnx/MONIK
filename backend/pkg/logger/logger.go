package logger

import (
	"io"
	"log"
	"os"
)

// Logger represents the application logger
type Logger struct {
	*log.Logger
}

// Global logger instance
var defaultLogger *Logger

// Init initializes the global logger
func Init() {
	defaultLogger = &Logger{
		Logger: log.New(os.Stdout, "[MONIK] ", log.LstdFlags),
	}
}

// Info logs an info message
func Info(format string, v ...interface{}) {
	if defaultLogger != nil {
		defaultLogger.Printf("[INFO] "+format, v...)
	}
}

// Error logs an error message
func Error(format string, v ...interface{}) {
	if defaultLogger != nil {
		defaultLogger.Printf("[ERROR] "+format, v...)
	}
}

// Warn logs a warning message
func Warn(format string, v ...interface{}) {
	if defaultLogger != nil {
		defaultLogger.Printf("[WARN] "+format, v...)
	}
}

// Debug logs a debug message
func Debug(format string, v ...interface{}) {
	if defaultLogger != nil {
		defaultLogger.Printf("[DEBUG] "+format, v...)
	}
}

// SetOutput sets the output destination for the logger
func SetOutput(w io.Writer) {
	if defaultLogger != nil {
		defaultLogger.SetOutput(w)
	}
}
