package lib

import (
	"fmt"
	"log"
)

type LeibnizLogger struct {
	module string
}

// NewLogger creates a new LeibnizLogger.
func NewLogger(module string) *LeibnizLogger {
	return &LeibnizLogger{module: module}
}

// Debug logs a debug message.
func (l *LeibnizLogger) Debug(msg string, args ...interface{}) {
	log.Printf("{%s} [DEBUG]: %s\n", l.module, fmt.Sprintf(msg, args...))
}

// Info logs an info message.
func (l *LeibnizLogger) Info(msg string, args ...interface{}) {
	log.Printf("{%s} [INFO]: %s\n", l.module, fmt.Sprintf(msg, args...))
}

// Warn logs a warning message.
func (l *LeibnizLogger) Warn(msg string, args ...interface{}) {
	log.Printf("{%s} [WARN]: %s\n", l.module, fmt.Sprintf(msg, args...))
}

// Error logs an error message.
func (l *LeibnizLogger) Error(msg string, args ...interface{}) {
	log.Printf("{%s} [ERROR]: %s\n", l.module, fmt.Sprintf(msg, args...))
}

// Fatal logs a fatal message.
func (l *LeibnizLogger) Fatal(msg string, args ...interface{}) {
	log.Fatalf("{%s} [FATAL]: %s\n", l.module, fmt.Sprintf(msg, args...))
}

func (l *LeibnizLogger) Log(level, msg string, args ...interface{}) {
	log.Printf("{%s} [%s]: %s\n", l.module, level, fmt.Sprintf(msg, args...))
}
