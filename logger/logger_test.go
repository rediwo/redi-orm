package logger

import (
	"bytes"
	"strings"
	"testing"
)

func TestDefaultLogger(t *testing.T) {
	// Create a buffer to capture output
	var buf bytes.Buffer

	// Create logger with custom output
	logger := NewDefaultLogger("TestApp")
	logger.SetOutput(&buf)

	// Test different log levels
	tests := []struct {
		level    LogLevel
		logFunc  func(string, ...any)
		message  string
		expected string
	}{
		{LogLevelDebug, logger.Debug, "Debug message", "DEBUG"},
		{LogLevelInfo, logger.Info, "Info message", "INFO"},
		{LogLevelWarn, logger.Warn, "Warn message", "WARN"},
		{LogLevelError, logger.Error, "Error message", "ERROR"},
	}

	for _, tt := range tests {
		t.Run(tt.level.String(), func(t *testing.T) {
			buf.Reset()
			logger.SetLevel(LogLevelDebug) // Enable all levels

			// Log the message
			tt.logFunc(tt.message)

			// Check output
			output := buf.String()
			if !strings.Contains(output, tt.expected) {
				t.Errorf("Expected output to contain %q, got %q", tt.expected, output)
			}
			if !strings.Contains(output, tt.message) {
				t.Errorf("Expected output to contain message %q, got %q", tt.message, output)
			}
		})
	}
}

func TestLogLevel(t *testing.T) {
	var buf bytes.Buffer
	logger := NewDefaultLogger("TestApp")
	logger.SetOutput(&buf)

	// Set level to WARN
	logger.SetLevel(LogLevelWarn)

	// Debug and Info should not be logged
	buf.Reset()
	logger.Debug("This should not appear")
	if buf.Len() > 0 {
		t.Error("Debug message was logged when level is WARN")
	}

	buf.Reset()
	logger.Info("This should not appear")
	if buf.Len() > 0 {
		t.Error("Info message was logged when level is WARN")
	}

	// Warn and Error should be logged
	buf.Reset()
	logger.Warn("This should appear")
	if buf.Len() == 0 {
		t.Error("Warn message was not logged when level is WARN")
	}

	buf.Reset()
	logger.Error("This should appear")
	if buf.Len() == 0 {
		t.Error("Error message was not logged when level is WARN")
	}
}

func TestParseLogLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected LogLevel
	}{
		{"debug", LogLevelDebug},
		{"DEBUG", LogLevelDebug},
		{"info", LogLevelInfo},
		{"INFO", LogLevelInfo},
		{"warn", LogLevelWarn},
		{"warning", LogLevelWarn},
		{"WARN", LogLevelWarn},
		{"error", LogLevelError},
		{"ERROR", LogLevelError},
		{"none", LogLevelNone},
		{"off", LogLevelNone},
		{"invalid", LogLevelInfo}, // default
		{"", LogLevelInfo},        // default
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := ParseLogLevel(tt.input)
			if result != tt.expected {
				t.Errorf("ParseLogLevel(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestLogLevelString(t *testing.T) {
	tests := []struct {
		level    LogLevel
		expected string
	}{
		{LogLevelNone, "NONE"},
		{LogLevelError, "ERROR"},
		{LogLevelWarn, "WARN"},
		{LogLevelInfo, "INFO"},
		{LogLevelDebug, "DEBUG"},
		{LogLevel(99), "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := tt.level.String()
			if result != tt.expected {
				t.Errorf("LogLevel(%d).String() = %q, want %q", tt.level, result, tt.expected)
			}
		})
	}
}
