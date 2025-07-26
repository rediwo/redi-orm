package logger

import "io"

// NullLogger is a logger that does nothing
type NullLogger struct {
	level LogLevel
}

// NewNullLogger creates a new null logger
func NewNullLogger() *NullLogger {
	return &NullLogger{level: LogLevelNone}
}

func (n *NullLogger) Debug(format string, args ...any) {}
func (n *NullLogger) Info(format string, args ...any)  {}
func (n *NullLogger) Warn(format string, args ...any)  {}
func (n *NullLogger) Error(format string, args ...any) {}

func (n *NullLogger) SetLevel(level LogLevel) {
	n.level = level
}

func (n *NullLogger) GetLevel() LogLevel {
	return n.level
}

func (n *NullLogger) SetOutput(w io.Writer) {}
