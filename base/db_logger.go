package base

import (
	"fmt"
	"strings"
	"time"

	"github.com/rediwo/redi-orm/logger"
)

// DBLogger wraps a logger.Logger and adds database-specific logging methods
type DBLogger struct {
	logger.Logger
}

// NewDBLogger creates a new database logger
func NewDBLogger(l logger.Logger) *DBLogger {
	if l == nil {
		l = logger.NewNullLogger()
	}
	return &DBLogger{Logger: l}
}

// LogSQL logs SQL query with parameters and duration
func (l *DBLogger) LogSQL(sql string, args []any, duration time.Duration) {
	if l.GetLevel() >= logger.LogLevelDebug {
		// Format SQL for readability
		formattedSQL := strings.TrimSpace(sql)

		// Log the SQL with duration
		l.Debug("SQL (%v):\n%s", duration, formattedSQL)

		// Log parameters if any
		if len(args) > 0 {
			argsStr := make([]string, len(args))
			for i, arg := range args {
				argsStr[i] = fmt.Sprintf("%v", arg)
			}
			l.Debug("Args: [%s]", strings.Join(argsStr, ", "))
		}
	}
}

// LogCommand logs a command (like MongoDB commands) with duration
func (l *DBLogger) LogCommand(command string, duration time.Duration) {
	if l.GetLevel() >= logger.LogLevelDebug {
		l.Debug("Command (%v):\n%s", duration, command)
	}
}
