package logging

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLevelString tests the Level.String() method.
func TestLevelString(t *testing.T) {
	tests := []struct {
		level    Level
		expected string
	}{
		{LevelDebug, "debug"},
		{LevelInfo, "info"},
		{LevelWarn, "warn"},
		{LevelError, "error"},
		{Level(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.level.String())
		})
	}
}

// TestParseLevel tests the ParseLevel function.
func TestParseLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected Level
	}{
		{"debug", LevelDebug},
		{"DEBUG", LevelDebug},
		{"info", LevelInfo},
		{"INFO", LevelInfo},
		{"warn", LevelWarn},
		{"WARN", LevelWarn},
		{"warning", LevelWarn},
		{"WARNING", LevelWarn},
		{"error", LevelError},
		{"ERROR", LevelError},
		{"unknown", LevelInfo},  // default
		{"", LevelInfo},         // default
		{"CRITICAL", LevelInfo}, // default for unrecognized
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, ParseLevel(tt.input))
		})
	}
}

// TestDefaultOptions tests the DefaultOptions function.
func TestDefaultOptions(t *testing.T) {
	opts := DefaultOptions()

	assert.Equal(t, LevelInfo, opts.Level)
	assert.Equal(t, os.Stderr, opts.Output)
	assert.Equal(t, "15:04:05", opts.TimeFormat)
	assert.False(t, opts.NoColor)
	assert.True(t, opts.ReportTimestamp)
}

// TestFileOptions tests the FileOptions function.
func TestFileOptions(t *testing.T) {
	var buf bytes.Buffer
	opts := FileOptions(&buf)

	assert.Equal(t, LevelDebug, opts.Level)
	assert.Equal(t, &buf, opts.Output)
	assert.Equal(t, "2006-01-02 15:04:05", opts.TimeFormat)
	assert.True(t, opts.NoColor)
	assert.True(t, opts.ReportTimestamp)
}

// TestNewLogger tests creating a new logger.
func TestNewLogger(t *testing.T) {
	var buf bytes.Buffer
	opts := Options{
		Level:           LevelInfo,
		Output:          &buf,
		TimeFormat:      "15:04:05",
		NoColor:         true,
		ReportTimestamp: true,
	}

	logger := New(opts)
	require.NotNil(t, logger)

	assert.Equal(t, LevelInfo, logger.GetLevel())
}

// TestLoggerLevels tests all log levels.
func TestLoggerLevels(t *testing.T) {
	var buf bytes.Buffer
	opts := Options{
		Level:           LevelDebug,
		Output:          &buf,
		TimeFormat:      "15:04:05",
		NoColor:         true,
		ReportTimestamp: false,
	}

	logger := New(opts)

	logger.Debug("debug message")
	assert.Contains(t, buf.String(), "debug message")
	buf.Reset()

	logger.Info("info message")
	assert.Contains(t, buf.String(), "info message")
	buf.Reset()

	logger.Warn("warn message")
	assert.Contains(t, buf.String(), "warn message")
	buf.Reset()

	logger.Error("error message")
	assert.Contains(t, buf.String(), "error message")
}

// TestLoggerLevelFiltering tests that log messages below the set level are filtered.
func TestLoggerLevelFiltering(t *testing.T) {
	tests := []struct {
		name        string
		level       Level
		expectDebug bool
		expectInfo  bool
		expectWarn  bool
		expectError bool
	}{
		{"LevelDebug", LevelDebug, true, true, true, true},
		{"LevelInfo", LevelInfo, false, true, true, true},
		{"LevelWarn", LevelWarn, false, false, true, true},
		{"LevelError", LevelError, false, false, false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			opts := Options{
				Level:           tt.level,
				Output:          &buf,
				NoColor:         true,
				ReportTimestamp: false,
			}

			logger := New(opts)

			buf.Reset()
			logger.Debug("debug")
			if tt.expectDebug {
				assert.Contains(t, buf.String(), "debug")
			} else {
				assert.NotContains(t, buf.String(), "debug")
			}

			buf.Reset()
			logger.Info("info")
			if tt.expectInfo {
				assert.Contains(t, buf.String(), "info")
			} else {
				assert.NotContains(t, buf.String(), "info")
			}

			buf.Reset()
			logger.Warn("warn")
			if tt.expectWarn {
				assert.Contains(t, buf.String(), "warn")
			} else {
				assert.NotContains(t, buf.String(), "warn")
			}

			buf.Reset()
			logger.Error("error")
			// Error is always logged
			assert.Contains(t, buf.String(), "error")
		})
	}
}

// TestLoggerKeyValues tests structured logging with key-value pairs.
func TestLoggerKeyValues(t *testing.T) {
	var buf bytes.Buffer
	opts := Options{
		Level:           LevelDebug,
		Output:          &buf,
		NoColor:         true,
		ReportTimestamp: false,
	}

	logger := New(opts)

	logger.Info("test message", "key1", "value1", "key2", 42)
	output := buf.String()

	assert.Contains(t, output, "test message")
	assert.Contains(t, output, "key1")
	assert.Contains(t, output, "value1")
	assert.Contains(t, output, "key2")
	assert.Contains(t, output, "42")
}

// TestLoggerWithPrefix tests the WithPrefix method.
func TestLoggerWithPrefix(t *testing.T) {
	var buf bytes.Buffer
	opts := Options{
		Level:           LevelInfo,
		Output:          &buf,
		NoColor:         true,
		ReportTimestamp: false,
	}

	logger := New(opts)
	prefixedLogger := logger.WithPrefix("TEST")

	prefixedLogger.Info("prefixed message")
	assert.Contains(t, buf.String(), "TEST")
	assert.Contains(t, buf.String(), "prefixed message")
}

// TestLoggerWithFields tests the WithFields method.
func TestLoggerWithFields(t *testing.T) {
	var buf bytes.Buffer
	opts := Options{
		Level:           LevelInfo,
		Output:          &buf,
		NoColor:         true,
		ReportTimestamp: false,
	}

	logger := New(opts)
	fieldLogger := logger.WithFields("component", "test", "version", "1.0")

	fieldLogger.Info("message with fields")
	output := buf.String()

	assert.Contains(t, output, "message with fields")
	assert.Contains(t, output, "component")
	assert.Contains(t, output, "test")
	assert.Contains(t, output, "version")
	assert.Contains(t, output, "1.0")

	// Test that additional key-values are merged
	buf.Reset()
	fieldLogger.Info("another message", "extra", "data")
	output = buf.String()

	assert.Contains(t, output, "component")
	assert.Contains(t, output, "extra")
	assert.Contains(t, output, "data")
}

// TestLoggerWithFieldsChained tests chaining WithFields calls.
func TestLoggerWithFieldsChained(t *testing.T) {
	var buf bytes.Buffer
	opts := Options{
		Level:           LevelInfo,
		Output:          &buf,
		NoColor:         true,
		ReportTimestamp: false,
	}

	logger := New(opts)
	chainedLogger := logger.WithFields("field1", "v1").WithFields("field2", "v2")

	chainedLogger.Info("chained message")
	output := buf.String()

	assert.Contains(t, output, "field1")
	assert.Contains(t, output, "v1")
	assert.Contains(t, output, "field2")
	assert.Contains(t, output, "v2")
}

// TestLoggerSetLevel tests dynamic level changes.
func TestLoggerSetLevel(t *testing.T) {
	var buf bytes.Buffer
	opts := Options{
		Level:           LevelInfo,
		Output:          &buf,
		NoColor:         true,
		ReportTimestamp: false,
	}

	logger := New(opts)
	assert.Equal(t, LevelInfo, logger.GetLevel())

	// Debug should not appear at Info level
	logger.Debug("should not appear")
	assert.NotContains(t, buf.String(), "should not appear")

	// Change to Debug level
	logger.SetLevel(LevelDebug)
	assert.Equal(t, LevelDebug, logger.GetLevel())

	// Now debug should appear
	logger.Debug("should appear")
	assert.Contains(t, buf.String(), "should appear")
}

// TestNopLogger tests the no-op logger.
func TestNopLogger(t *testing.T) {
	logger := NewNop()
	require.NotNil(t, logger)

	// All methods should be callable without panic
	logger.Debug("debug")
	logger.Info("info")
	logger.Warn("warn")
	logger.Error("error")

	// WithPrefix returns same nop logger
	prefixed := logger.WithPrefix("TEST")
	assert.NotNil(t, prefixed)

	// WithFields returns same nop logger
	withFields := logger.WithFields("key", "value")
	assert.NotNil(t, withFields)

	// SetLevel should not panic
	logger.SetLevel(LevelDebug)

	// GetLevel returns default
	assert.Equal(t, LevelInfo, logger.GetLevel())
}

// TestMultiLogger tests the multi-logger.
func TestMultiLogger(t *testing.T) {
	var buf1, buf2 bytes.Buffer

	logger1 := New(Options{
		Level:           LevelInfo,
		Output:          &buf1,
		NoColor:         true,
		ReportTimestamp: false,
	})

	logger2 := New(Options{
		Level:           LevelDebug,
		Output:          &buf2,
		NoColor:         true,
		ReportTimestamp: false,
	})

	multi := NewMultiLogger(logger1, logger2)

	multi.Info("test message")

	assert.Contains(t, buf1.String(), "test message")
	assert.Contains(t, buf2.String(), "test message")
}

// TestMultiLoggerAllLevels tests all log levels on multi-logger.
func TestMultiLoggerAllLevels(t *testing.T) {
	var buf bytes.Buffer
	logger := New(Options{
		Level:           LevelDebug,
		Output:          &buf,
		NoColor:         true,
		ReportTimestamp: false,
	})

	multi := NewMultiLogger(logger)

	buf.Reset()
	multi.Debug("debug")
	assert.Contains(t, buf.String(), "debug")

	buf.Reset()
	multi.Info("info")
	assert.Contains(t, buf.String(), "info")

	buf.Reset()
	multi.Warn("warn")
	assert.Contains(t, buf.String(), "warn")

	buf.Reset()
	multi.Error("error")
	assert.Contains(t, buf.String(), "error")
}

// TestMultiLoggerWithPrefix tests WithPrefix on multi-logger.
func TestMultiLoggerWithPrefix(t *testing.T) {
	var buf1, buf2 bytes.Buffer

	logger1 := New(Options{
		Level:           LevelInfo,
		Output:          &buf1,
		NoColor:         true,
		ReportTimestamp: false,
	})

	logger2 := New(Options{
		Level:           LevelInfo,
		Output:          &buf2,
		NoColor:         true,
		ReportTimestamp: false,
	})

	multi := NewMultiLogger(logger1, logger2)
	prefixed := multi.WithPrefix("PREFIX")

	prefixed.Info("prefixed message")

	assert.Contains(t, buf1.String(), "PREFIX")
	assert.Contains(t, buf2.String(), "PREFIX")
}

// TestMultiLoggerWithFields tests WithFields on multi-logger.
func TestMultiLoggerWithFields(t *testing.T) {
	var buf1, buf2 bytes.Buffer

	logger1 := New(Options{
		Level:           LevelInfo,
		Output:          &buf1,
		NoColor:         true,
		ReportTimestamp: false,
	})

	logger2 := New(Options{
		Level:           LevelInfo,
		Output:          &buf2,
		NoColor:         true,
		ReportTimestamp: false,
	})

	multi := NewMultiLogger(logger1, logger2)
	withFields := multi.WithFields("key", "value")

	withFields.Info("message")

	assert.Contains(t, buf1.String(), "key")
	assert.Contains(t, buf1.String(), "value")
	assert.Contains(t, buf2.String(), "key")
	assert.Contains(t, buf2.String(), "value")
}

// TestMultiLoggerSetLevel tests SetLevel on multi-logger.
func TestMultiLoggerSetLevel(t *testing.T) {
	var buf1, buf2 bytes.Buffer

	logger1 := New(Options{
		Level:           LevelInfo,
		Output:          &buf1,
		NoColor:         true,
		ReportTimestamp: false,
	})

	logger2 := New(Options{
		Level:           LevelInfo,
		Output:          &buf2,
		NoColor:         true,
		ReportTimestamp: false,
	})

	multi := NewMultiLogger(logger1, logger2)

	// Initially at Info, debug shouldn't appear
	multi.Debug("should not appear")
	assert.NotContains(t, buf1.String(), "should not appear")
	assert.NotContains(t, buf2.String(), "should not appear")

	// Set to Debug level
	multi.SetLevel(LevelDebug)

	multi.Debug("should appear")
	assert.Contains(t, buf1.String(), "should appear")
	assert.Contains(t, buf2.String(), "should appear")
}

// TestMultiLoggerGetLevel tests GetLevel on multi-logger.
func TestMultiLoggerGetLevel(t *testing.T) {
	logger := New(Options{
		Level:           LevelWarn,
		Output:          &bytes.Buffer{},
		NoColor:         true,
		ReportTimestamp: false,
	})

	multi := NewMultiLogger(logger)
	assert.Equal(t, LevelWarn, multi.GetLevel())

	// Empty multi-logger
	emptyMulti := NewMultiLogger()
	assert.Equal(t, LevelInfo, emptyMulti.GetLevel())
}

// TestFileLogger tests file-based logging.
func TestFileLogger(t *testing.T) {
	// Create a temporary file
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.log")

	logger, err := NewFileLogger(logPath, LevelDebug)
	require.NoError(t, err)
	require.NotNil(t, logger)

	logger.Info("file log message", "key", "value")

	// Read the file
	content, err := os.ReadFile(logPath)
	require.NoError(t, err)

	assert.Contains(t, string(content), "file log message")
	assert.Contains(t, string(content), "key")
	assert.Contains(t, string(content), "value")
}

// TestFileLoggerError tests file logger creation with invalid path.
func TestFileLoggerError(t *testing.T) {
	// Try to create a logger with an invalid path
	logger, err := NewFileLogger("/nonexistent/directory/test.log", LevelInfo)
	assert.Error(t, err)
	assert.Nil(t, logger)
}

// TestFileLoggerAppendsToExistingFile tests that file logger appends to existing files.
func TestFileLoggerAppendsToExistingFile(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "append.log")

	// Create first logger and write
	logger1, err := NewFileLogger(logPath, LevelInfo)
	require.NoError(t, err)
	logger1.Info("first message")

	// Create second logger and write
	logger2, err := NewFileLogger(logPath, LevelInfo)
	require.NoError(t, err)
	logger2.Info("second message")

	// Read the file
	content, err := os.ReadFile(logPath)
	require.NoError(t, err)

	assert.Contains(t, string(content), "first message")
	assert.Contains(t, string(content), "second message")
}

// TestThreadSafety tests concurrent logging.
func TestThreadSafety(t *testing.T) {
	var buf bytes.Buffer
	opts := Options{
		Level:           LevelDebug,
		Output:          &buf,
		NoColor:         true,
		ReportTimestamp: false,
	}

	logger := New(opts)

	var wg sync.WaitGroup
	numGoroutines := 100
	numMessages := 10

	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numMessages; j++ {
				logger.Debug("debug message", "goroutine", id, "iter", j)
				logger.Info("info message", "goroutine", id, "iter", j)
				logger.Warn("warn message", "goroutine", id, "iter", j)
				logger.Error("error message", "goroutine", id, "iter", j)
			}
		}(i)
	}

	wg.Wait()

	// Just verify no panics and something was written
	output := buf.String()
	assert.NotEmpty(t, output)
	assert.Contains(t, output, "debug message")
	assert.Contains(t, output, "info message")
	assert.Contains(t, output, "warn message")
	assert.Contains(t, output, "error message")
}

// TestThreadSafetyWithLevelChange tests concurrent logging with level changes.
func TestThreadSafetyWithLevelChange(t *testing.T) {
	var buf bytes.Buffer
	opts := Options{
		Level:           LevelInfo,
		Output:          &buf,
		NoColor:         true,
		ReportTimestamp: false,
	}

	logger := New(opts)

	var wg sync.WaitGroup
	wg.Add(2)

	// Writer goroutine
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			logger.Info("message", "iter", i)
		}
	}()

	// Level changer goroutine
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			if i%2 == 0 {
				logger.SetLevel(LevelDebug)
			} else {
				logger.SetLevel(LevelInfo)
			}
			_ = logger.GetLevel()
		}
	}()

	wg.Wait()
	// Just verify no panics occurred
	assert.NotEmpty(t, buf.String())
}

// TestLoggerWithPrefixDoesNotAffectOriginal tests that WithPrefix returns a new logger.
func TestLoggerWithPrefixDoesNotAffectOriginal(t *testing.T) {
	var buf bytes.Buffer
	opts := Options{
		Level:           LevelInfo,
		Output:          &buf,
		NoColor:         true,
		ReportTimestamp: false,
	}

	original := New(opts)
	prefixed := original.WithPrefix("PREFIXED")

	buf.Reset()
	original.Info("original message")
	originalOutput := buf.String()

	buf.Reset()
	prefixed.Info("prefixed message")
	prefixedOutput := buf.String()

	// Original should not have the prefix
	assert.NotContains(t, originalOutput, "PREFIXED")

	// Prefixed should have the prefix
	assert.Contains(t, prefixedOutput, "PREFIXED")
}

// TestLoggerWithFieldsDoesNotAffectOriginal tests that WithFields returns a new logger.
func TestLoggerWithFieldsDoesNotAffectOriginal(t *testing.T) {
	var buf bytes.Buffer
	opts := Options{
		Level:           LevelInfo,
		Output:          &buf,
		NoColor:         true,
		ReportTimestamp: false,
	}

	original := New(opts)
	withFields := original.WithFields("extra", "field")

	buf.Reset()
	original.Info("original message")
	originalOutput := buf.String()

	buf.Reset()
	withFields.Info("fields message")
	fieldsOutput := buf.String()

	// Original should not have the extra field
	assert.NotContains(t, originalOutput, "extra")
	assert.NotContains(t, originalOutput, "field")

	// WithFields logger should have the field
	assert.Contains(t, fieldsOutput, "extra")
	assert.Contains(t, fieldsOutput, "field")
}

// TestLoggerNoColor tests that NoColor option works correctly.
func TestLoggerNoColor(t *testing.T) {
	var buf bytes.Buffer
	opts := Options{
		Level:           LevelInfo,
		Output:          &buf,
		NoColor:         true,
		ReportTimestamp: false,
	}

	logger := New(opts)
	logger.Info("no color message")

	output := buf.String()
	// ANSI escape sequences start with \x1b[ or \033[
	assert.False(t, strings.Contains(output, "\x1b["), "output should not contain ANSI escape codes")
}

// TestLoggerTimestamp tests timestamp output.
func TestLoggerTimestamp(t *testing.T) {
	var buf bytes.Buffer
	opts := Options{
		Level:           LevelInfo,
		Output:          &buf,
		TimeFormat:      "15:04:05",
		NoColor:         true,
		ReportTimestamp: true,
	}

	logger := New(opts)
	logger.Info("timestamped message")

	output := buf.String()
	// Check for timestamp pattern (HH:MM:SS)
	assert.Regexp(t, `\d{2}:\d{2}:\d{2}`, output)
}

// TestLoggerInterface tests that logger implements the Logger interface.
func TestLoggerInterface(t *testing.T) {
	var _ Logger = New(DefaultOptions())
	var _ Logger = NewNop()
	var _ Logger = NewMultiLogger()
}

// TestToCharmLevel tests the level conversion function.
func TestToCharmLevel(t *testing.T) {
	// This is an internal function, but we can test it indirectly
	// by setting levels and verifying behavior
	var buf bytes.Buffer
	opts := Options{
		Level:           LevelDebug,
		Output:          &buf,
		NoColor:         true,
		ReportTimestamp: false,
	}

	logger := New(opts)

	// Test each level
	levels := []Level{LevelDebug, LevelInfo, LevelWarn, LevelError}
	for _, level := range levels {
		logger.SetLevel(level)
		assert.Equal(t, level, logger.GetLevel())
	}
}
