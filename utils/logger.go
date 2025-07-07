package utils

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"sync"
	"time"
)

// LogLevel represents logging verbosity
type LogLevel int

const (
	LogLevelNone LogLevel = iota
	LogLevelError
	LogLevelWarn
	LogLevelInfo
	LogLevelDebug
)

// Logger interface defines logging methods
type Logger interface {
	Debug(format string, args ...any)
	Info(format string, args ...any)
	Warn(format string, args ...any)
	Error(format string, args ...any)

	// SQL/Command specific logging
	LogSQL(sql string, args []any, duration time.Duration)
	LogCommand(command string, duration time.Duration)

	// Configuration
	SetLevel(level LogLevel)
	SetOutput(w io.Writer)
}

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

// SetOutput sets the output writer
func (l *DefaultLogger) SetOutput(w io.Writer) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.logger.SetOutput(w)
}

// ANSI color codes
const (
	colorReset  = "\033[0m"
	colorGray   = "\033[90m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorRed    = "\033[31m"
)

// log logs a message at the specified level
func (l *DefaultLogger) log(level LogLevel, format string, args ...any) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	if l.level >= level {
		levelStr := ""
		colorCode := ""
		switch level {
		case LogLevelError:
			levelStr = "ERROR"
			colorCode = colorRed
		case LogLevelWarn:
			levelStr = "WARN"
			colorCode = colorYellow
		case LogLevelInfo:
			levelStr = "INFO"
			colorCode = colorGreen
		case LogLevelDebug:
			levelStr = "DEBUG"
			colorCode = colorGray
		}

		timestamp := time.Now().Format("15:04:05.000")
		message := fmt.Sprintf(format, args...)

		if l.prefix != "" {
			l.logger.Printf("%s [%s] %s%s%s: %s", timestamp, l.prefix, colorCode, levelStr, colorReset, message)
		} else {
			l.logger.Printf("%s %s%s%s: %s", timestamp, colorCode, levelStr, colorReset, message)
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

// LogSQL logs SQL query with parameters and duration
func (l *DefaultLogger) LogSQL(sql string, args []any, duration time.Duration) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	if l.level >= LogLevelDebug {
		// Format SQL for readability
		formattedSQL := strings.TrimSpace(sql)

		// Log the SQL
		timestamp := time.Now().Format("15:04:05.000")
		if l.prefix != "" {
			l.logger.Printf("%s [%s] %sDEBUG%s: SQL (%v):\n%s", timestamp, l.prefix, colorGray, colorReset, duration, formattedSQL)
		} else {
			l.logger.Printf("%s %sDEBUG%s: SQL (%v):\n%s", timestamp, colorGray, colorReset, duration, formattedSQL)
		}

		// Log parameters if any
		if len(args) > 0 {
			argsStr := make([]string, len(args))
			for i, arg := range args {
				argsStr[i] = fmt.Sprintf("%v", arg)
			}
			if l.prefix != "" {
				l.logger.Printf("%s [%s] %sDEBUG%s: Args: [%s]", timestamp, l.prefix, colorGray, colorReset, strings.Join(argsStr, ", "))
			} else {
				l.logger.Printf("%s %sDEBUG%s: Args: [%s]", timestamp, colorGray, colorReset, strings.Join(argsStr, ", "))
			}
		}
	}
}

// LogCommand logs a command (like MongoDB commands) with duration
func (l *DefaultLogger) LogCommand(command string, duration time.Duration) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	if l.level >= LogLevelDebug {
		timestamp := time.Now().Format("15:04:05.000")
		if l.prefix != "" {
			l.logger.Printf("%s [%s] %sDEBUG%s: Command (%v):\n%s", timestamp, l.prefix, colorGray, colorReset, duration, command)
		} else {
			l.logger.Printf("%s %sDEBUG%s: Command (%v):\n%s", timestamp, colorGray, colorReset, duration, command)
		}
	}
}

// NullLogger is a logger that does nothing
type NullLogger struct{}

func (n *NullLogger) Debug(format string, args ...any)                      {}
func (n *NullLogger) Info(format string, args ...any)                       {}
func (n *NullLogger) Warn(format string, args ...any)                       {}
func (n *NullLogger) Error(format string, args ...any)                      {}
func (n *NullLogger) LogSQL(sql string, args []any, duration time.Duration) {}
func (n *NullLogger) LogCommand(command string, duration time.Duration)     {}
func (n *NullLogger) SetLevel(level LogLevel)                               {}
func (n *NullLogger) SetOutput(w io.Writer)                                 {}

// Global logger instance
var globalLogger Logger = &NullLogger{}
var globalMu sync.RWMutex

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

// Convenience functions using the global logger
func LogDebug(format string, args ...any) {
	GetGlobalLogger().Debug(format, args...)
}

func LogInfo(format string, args ...any) {
	GetGlobalLogger().Info(format, args...)
}

func LogWarn(format string, args ...any) {
	GetGlobalLogger().Warn(format, args...)
}

func LogError(format string, args ...any) {
	GetGlobalLogger().Error(format, args...)
}

func LogSQL(sql string, args []any, duration time.Duration) {
	GetGlobalLogger().LogSQL(sql, args, duration)
}

func LogCommand(command string, duration time.Duration) {
	GetGlobalLogger().LogCommand(command, duration)
}
