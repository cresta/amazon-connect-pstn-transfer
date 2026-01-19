package main

import (
	"log"
	"os"
)

// Logger provides structured logging functionality
type Logger struct {
	debugEnabled bool
}

// NewLogger creates a new logger instance
func NewLogger() *Logger {
	return &Logger{
		debugEnabled: os.Getenv("DEBUG_LOGGING") == "true",
	}
}

// Debugf logs a debug message
func (l *Logger) Debugf(format string, v ...any) {
	if l.debugEnabled {
		log.Printf("[DEBUG] "+format, v...)
	}
}

// Infof logs an info message
func (l *Logger) Infof(format string, v ...any) {
	log.Printf("[INFO] "+format, v...)
}

// Errorf logs an error message
func (l *Logger) Errorf(format string, v ...any) {
	log.Printf("[ERROR] "+format, v...)
}

// Default logger instance
var defaultLogger = NewLogger()
