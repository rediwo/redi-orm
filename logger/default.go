package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"sync"
	"time"
)

// DefaultLogger is the default logger implementation
type DefaultLogger struct {
	mu     sync.RWMutex
	level  LogLevel
	logger *log.Logger
	prefix string
}

// NewDefaultLogger creates a new default logger
func NewDefaultLogger(prefix string) *DefaultLogger {
	return &DefaultLogger{
		level:  LogLevelInfo,
		logger: log.New(os.Stdout, "", 0),
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

// log logs a message at the specified level
func (l *DefaultLogger) log(level LogLevel, format string, args ...any) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	if l.level >= level {
		timestamp := time.Now().Format("15:04:05.000")
		message := fmt.Sprintf(format, args...)
		levelStr := level.String()
		colorCode := GetLevelColor(level)

		if l.prefix != "" {
			l.logger.Printf("%s [%s] %s%s%s: %s", timestamp, l.prefix, colorCode, levelStr, ColorReset, message)
		} else {
			l.logger.Printf("%s %s%s%s: %s", timestamp, colorCode, levelStr, ColorReset, message)
		}
	}
}

// Debug logs a debug message
func (l *DefaultLogger) Debug(format string, args ...any) {
	l.log(LogLevelDebug, format, args...)
}

// Info logs an info message
func (l *DefaultLogger) Info(format string, args ...any) {
	l.log(LogLevelInfo, format, args...)
}

// Warn logs a warning message
func (l *DefaultLogger) Warn(format string, args ...any) {
	l.log(LogLevelWarn, format, args...)
}

// Error logs an error message
func (l *DefaultLogger) Error(format string, args ...any) {
	l.log(LogLevelError, format, args...)
}
