package logging

import (
	"io"
	"os"
	"sync"

	"github.com/charmbracelet/log"
	"github.com/muesli/termenv"
)

// Logger defines the interface for logging operations.
// This interface is designed for easy mocking in tests.
type Logger interface {
	// Debug logs a debug message with optional key-value pairs.
	Debug(msg string, keyvals ...interface{})
	// Info logs an info message with optional key-value pairs.
	Info(msg string, keyvals ...interface{})
	// Warn logs a warning message with optional key-value pairs.
	Warn(msg string, keyvals ...interface{})
	// Error logs an error message with optional key-value pairs.
	Error(msg string, keyvals ...interface{})
	// WithPrefix returns a new Logger with the given prefix.
	WithPrefix(prefix string) Logger
	// WithFields returns a new Logger with the given fields added to all messages.
	WithFields(keyvals ...interface{}) Logger
	// SetLevel sets the minimum log level.
	SetLevel(level Level)
	// GetLevel returns the current log level.
	GetLevel() Level
}

// Options configures the logger.
type Options struct {
	// Level is the minimum log level to output.
	Level Level
	// Output is the destination for log messages.
	Output io.Writer
	// TimeFormat is the format string for timestamps.
	TimeFormat string
	// Prefix is an optional prefix for all log messages.
	Prefix string
	// NoColor disables colorized output.
	NoColor bool
	// ReportTimestamp enables timestamp output.
	ReportTimestamp bool
}

// DefaultOptions returns sensible defaults for console logging.
func DefaultOptions() Options {
	return Options{
		Level:           LevelInfo,
		Output:          os.Stderr,
		TimeFormat:      "15:04:05",
		NoColor:         false,
		ReportTimestamp: true,
	}
}

// FileOptions returns options optimized for file logging (no color, full timestamp).
func FileOptions(w io.Writer) Options {
	return Options{
		Level:           LevelDebug,
		Output:          w,
		TimeFormat:      "2006-01-02 15:04:05",
		NoColor:         true,
		ReportTimestamp: true,
	}
}

// logger is the concrete implementation of Logger.
type logger struct {
	mu     sync.RWMutex
	impl   *log.Logger
	level  Level
	fields []interface{}
	prefix string
	output io.Writer
}

// New creates a new logger with the given options.
func New(opts Options) Logger {
	l := log.NewWithOptions(opts.Output, log.Options{
		TimeFormat:      opts.TimeFormat,
		Level:           toCharmLevel(opts.Level),
		Prefix:          opts.Prefix,
		ReportTimestamp: opts.ReportTimestamp,
	})

	if opts.NoColor {
		l.SetColorProfile(termenv.Ascii)
	}

	return &logger{
		impl:   l,
		level:  opts.Level,
		prefix: opts.Prefix,
		output: opts.Output,
	}
}

// NewNop returns a no-op logger that discards all output.
// Useful for testing or when logging should be completely disabled.
func NewNop() Logger {
	return &nopLogger{}
}

// NewFileLogger creates a logger that writes to a file.
// The file is created if it doesn't exist, or appended to if it does.
func NewFileLogger(path string, level Level) (Logger, error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}

	opts := FileOptions(file)
	opts.Level = level
	return New(opts), nil
}

// NewMultiLogger creates a logger that writes to multiple loggers.
// All loggers receive all log messages at their respective levels.
func NewMultiLogger(loggers ...Logger) Logger {
	return &multiLogger{loggers: loggers}
}

func (l *logger) Debug(msg string, keyvals ...interface{}) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	if l.level <= LevelDebug {
		l.impl.Debug(msg, append(l.fields, keyvals...)...)
	}
}

func (l *logger) Info(msg string, keyvals ...interface{}) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	if l.level <= LevelInfo {
		l.impl.Info(msg, append(l.fields, keyvals...)...)
	}
}

func (l *logger) Warn(msg string, keyvals ...interface{}) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	if l.level <= LevelWarn {
		l.impl.Warn(msg, append(l.fields, keyvals...)...)
	}
}

func (l *logger) Error(msg string, keyvals ...interface{}) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	// Error is always logged regardless of level
	l.impl.Error(msg, append(l.fields, keyvals...)...)
}

func (l *logger) WithPrefix(prefix string) Logger {
	l.mu.RLock()
	defer l.mu.RUnlock()

	newImpl := l.impl.WithPrefix(prefix)

	return &logger{
		impl:   newImpl,
		level:  l.level,
		fields: l.fields,
		prefix: prefix,
		output: l.output,
	}
}

func (l *logger) WithFields(keyvals ...interface{}) Logger {
	l.mu.RLock()
	defer l.mu.RUnlock()

	newFields := make([]interface{}, len(l.fields)+len(keyvals))
	copy(newFields, l.fields)
	copy(newFields[len(l.fields):], keyvals)

	return &logger{
		impl:   l.impl,
		level:  l.level,
		fields: newFields,
		prefix: l.prefix,
		output: l.output,
	}
}

func (l *logger) SetLevel(level Level) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
	l.impl.SetLevel(toCharmLevel(level))
}

func (l *logger) GetLevel() Level {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.level
}

// toCharmLevel converts our Level to charmbracelet/log Level.
func toCharmLevel(l Level) log.Level {
	switch l {
	case LevelDebug:
		return log.DebugLevel
	case LevelInfo:
		return log.InfoLevel
	case LevelWarn:
		return log.WarnLevel
	case LevelError:
		return log.ErrorLevel
	default:
		return log.InfoLevel
	}
}

// nopLogger discards all log output.
type nopLogger struct{}

func (n *nopLogger) Debug(msg string, keyvals ...interface{}) {}
func (n *nopLogger) Info(msg string, keyvals ...interface{})  {}
func (n *nopLogger) Warn(msg string, keyvals ...interface{})  {}
func (n *nopLogger) Error(msg string, keyvals ...interface{}) {}
func (n *nopLogger) WithPrefix(prefix string) Logger          { return n }
func (n *nopLogger) WithFields(keyvals ...interface{}) Logger { return n }
func (n *nopLogger) SetLevel(level Level)                     {}
func (n *nopLogger) GetLevel() Level                          { return LevelInfo }

// multiLogger writes to multiple loggers.
type multiLogger struct {
	loggers []Logger
}

func (m *multiLogger) Debug(msg string, keyvals ...interface{}) {
	for _, l := range m.loggers {
		l.Debug(msg, keyvals...)
	}
}

func (m *multiLogger) Info(msg string, keyvals ...interface{}) {
	for _, l := range m.loggers {
		l.Info(msg, keyvals...)
	}
}

func (m *multiLogger) Warn(msg string, keyvals ...interface{}) {
	for _, l := range m.loggers {
		l.Warn(msg, keyvals...)
	}
}

func (m *multiLogger) Error(msg string, keyvals ...interface{}) {
	for _, l := range m.loggers {
		l.Error(msg, keyvals...)
	}
}

func (m *multiLogger) WithPrefix(prefix string) Logger {
	newLoggers := make([]Logger, len(m.loggers))
	for i, l := range m.loggers {
		newLoggers[i] = l.WithPrefix(prefix)
	}
	return &multiLogger{loggers: newLoggers}
}

func (m *multiLogger) WithFields(keyvals ...interface{}) Logger {
	newLoggers := make([]Logger, len(m.loggers))
	for i, l := range m.loggers {
		newLoggers[i] = l.WithFields(keyvals...)
	}
	return &multiLogger{loggers: newLoggers}
}

func (m *multiLogger) SetLevel(level Level) {
	for _, l := range m.loggers {
		l.SetLevel(level)
	}
}

func (m *multiLogger) GetLevel() Level {
	if len(m.loggers) > 0 {
		return m.loggers[0].GetLevel()
	}
	return LevelInfo
}
