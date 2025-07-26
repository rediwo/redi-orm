package mcp

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/rediwo/redi-orm/logger"
)

// LoggerWriter implements io.Writer to bridge MCP's logging to our logger
type LoggerWriter struct {
	logger logger.Logger
	prefix string
}

// NewLoggerWriter creates a new logger writer
func NewLoggerWriter(l logger.Logger, prefix string) *LoggerWriter {
	return &LoggerWriter{
		logger: l,
		prefix: prefix,
	}
}

// Write implements io.Writer interface
func (w *LoggerWriter) Write(p []byte) (n int, err error) {
	if len(p) == 0 {
		return 0, nil
	}

	// Convert bytes to string and trim whitespace
	msg := strings.TrimSpace(string(p))
	if msg == "" {
		return len(p), nil
	}

	// Try to parse as JSON-RPC message
	var jsonMsg map[string]any
	if err := json.Unmarshal([]byte(msg), &jsonMsg); err == nil {
		w.logJSONRPC(jsonMsg)
	} else {
		// Not JSON, log as plain text
		w.logPlainText(msg)
	}

	return len(p), nil
}

// logJSONRPC logs a JSON-RPC message with appropriate formatting
func (w *LoggerWriter) logJSONRPC(msg map[string]any) {
	// Determine message type
	if method, ok := msg["method"].(string); ok {
		// This is a request
		id := w.formatID(msg["id"])
		params := w.formatParams(msg["params"])

		if w.prefix != "" {
			w.logger.Debug("[%s] → Request #%s: %s %s", w.prefix, id, method, params)
		} else {
			w.logger.Debug("→ Request #%s: %s %s", id, method, params)
		}
	} else if _, hasResult := msg["result"]; hasResult {
		// This is a response
		id := w.formatID(msg["id"])
		result := w.formatResult(msg["result"])

		if w.prefix != "" {
			w.logger.Debug("[%s] ← Response #%s: %s", w.prefix, id, result)
		} else {
			w.logger.Debug("← Response #%s: %s", id, result)
		}
	} else if _, hasError := msg["error"]; hasError {
		// This is an error response
		id := w.formatID(msg["id"])
		errMsg := w.formatError(msg["error"])

		if w.prefix != "" {
			w.logger.Error("[%s] ← Error #%s: %s", w.prefix, id, errMsg)
		} else {
			w.logger.Error("← Error #%s: %s", id, errMsg)
		}
	} else {
		// Unknown JSON-RPC message type
		if w.prefix != "" {
			w.logger.Debug("[%s] %s", w.prefix, w.truncate(fmt.Sprintf("%v", msg), 200))
		} else {
			w.logger.Debug("%s", w.truncate(fmt.Sprintf("%v", msg), 200))
		}
	}
}

// logPlainText logs a plain text message
func (w *LoggerWriter) logPlainText(msg string) {
	// Determine log level based on content
	level := w.detectLogLevel(msg)

	if w.prefix != "" {
		switch level {
		case logger.LogLevelError:
			w.logger.Error("[%s] %s", w.prefix, msg)
		case logger.LogLevelWarn:
			w.logger.Warn("[%s] %s", w.prefix, msg)
		case logger.LogLevelInfo:
			w.logger.Info("[%s] %s", w.prefix, msg)
		default:
			w.logger.Debug("[%s] %s", w.prefix, msg)
		}
	} else {
		switch level {
		case logger.LogLevelError:
			w.logger.Error("%s", msg)
		case logger.LogLevelWarn:
			w.logger.Warn("%s", msg)
		case logger.LogLevelInfo:
			w.logger.Info("%s", msg)
		default:
			w.logger.Debug("%s", msg)
		}
	}
}

// detectLogLevel tries to determine the appropriate log level from message content
func (w *LoggerWriter) detectLogLevel(msg string) logger.LogLevel {
	msgLower := strings.ToLower(msg)

	if strings.Contains(msgLower, "error") || strings.Contains(msgLower, "fail") {
		return logger.LogLevelError
	}
	if strings.Contains(msgLower, "warn") {
		return logger.LogLevelWarn
	}
	if strings.Contains(msgLower, "info") || strings.Contains(msgLower, "start") || strings.Contains(msgLower, "connect") {
		return logger.LogLevelInfo
	}

	return logger.LogLevelDebug
}

// formatID formats a JSON-RPC ID value
func (w *LoggerWriter) formatID(id any) string {
	if id == nil {
		return "null"
	}
	return fmt.Sprintf("%v", id)
}

// formatParams formats JSON-RPC params for logging
func (w *LoggerWriter) formatParams(params any) string {
	if params == nil {
		return ""
	}

	// Convert to JSON for pretty printing
	data, err := json.Marshal(params)
	if err != nil {
		return fmt.Sprintf("(%v)", params)
	}

	// Truncate if too long
	result := string(data)
	return w.truncate(result, 200)
}

// formatResult formats JSON-RPC result for logging
func (w *LoggerWriter) formatResult(result any) string {
	if result == nil {
		return "null"
	}

	// Convert to JSON for pretty printing
	data, err := json.Marshal(result)
	if err != nil {
		return fmt.Sprintf("%v", result)
	}

	// Truncate if too long
	resultStr := string(data)
	return w.truncate(resultStr, 200)
}

// formatError formats JSON-RPC error for logging
func (w *LoggerWriter) formatError(err any) string {
	if err == nil {
		return "unknown error"
	}

	if errMap, ok := err.(map[string]any); ok {
		code := errMap["code"]
		message := errMap["message"]
		data := errMap["data"]

		var parts []string
		if code != nil {
			parts = append(parts, fmt.Sprintf("code=%v", code))
		}
		if message != nil {
			parts = append(parts, fmt.Sprintf("message=%v", message))
		}
		if data != nil {
			parts = append(parts, fmt.Sprintf("data=%v", data))
		}

		return strings.Join(parts, ", ")
	}

	return fmt.Sprintf("%v", err)
}

// truncate truncates a string to the specified length
func (w *LoggerWriter) truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return "..."
	}
	return s[:maxLen-3] + "..."
}
