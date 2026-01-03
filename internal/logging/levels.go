// Package logging provides centralized logging with configurable levels and file output.
// The logging is designed to work seamlessly with TUI applications by supporting
// file-based output that doesn't interfere with terminal displays.
package logging

// Level represents logging severity levels.
// Levels are ordered from most verbose (Debug) to least verbose (Error).
type Level int

const (
	// LevelDebug is for detailed debugging information.
	LevelDebug Level = iota
	// LevelInfo is for general informational messages.
	LevelInfo
	// LevelWarn is for warning messages about potential issues.
	LevelWarn
	// LevelError is for error messages about failures.
	LevelError
)

// String returns the string representation of the level.
func (l Level) String() string {
	switch l {
	case LevelDebug:
		return "debug"
	case LevelInfo:
		return "info"
	case LevelWarn:
		return "warn"
	case LevelError:
		return "error"
	default:
		return "unknown"
	}
}

// ParseLevel converts a string to a Level.
// Unrecognized strings default to LevelInfo.
func ParseLevel(s string) Level {
	switch s {
	case "debug", "DEBUG":
		return LevelDebug
	case "info", "INFO":
		return LevelInfo
	case "warn", "WARN", "warning", "WARNING":
		return LevelWarn
	case "error", "ERROR":
		return LevelError
	default:
		return LevelInfo
	}
}
