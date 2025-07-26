package logger

import "sync"

// Global logger instance
var (
	globalLogger Logger = NewNullLogger()
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

// Convenience functions using the global logger
func Debug(format string, args ...any) {
	GetGlobalLogger().Debug(format, args...)
}

func Info(format string, args ...any) {
	GetGlobalLogger().Info(format, args...)
}

func Warn(format string, args ...any) {
	GetGlobalLogger().Warn(format, args...)
}

func Error(format string, args ...any) {
	GetGlobalLogger().Error(format, args...)
}
