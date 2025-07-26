package logger

// ANSI color codes for terminal output
const (
	ColorReset  = "\033[0m"
	ColorGray   = "\033[90m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorRed    = "\033[31m"
	ColorBlue   = "\033[34m"
	ColorCyan   = "\033[36m"
)

// GetLevelColor returns the color code for a given log level
func GetLevelColor(level LogLevel) string {
	switch level {
	case LogLevelError:
		return ColorRed
	case LogLevelWarn:
		return ColorYellow
	case LogLevelInfo:
		return ColorGreen
	case LogLevelDebug:
		return ColorGray
	default:
		return ColorReset
	}
}
