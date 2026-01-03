// Package exec provides a testable command execution abstraction that captures
// output, handles timeouts, and supports mocking for testing.
package exec

import (
	"bytes"
	"strings"
	"time"
)

// Result represents the result of command execution.
type Result struct {
	Command   string        // The command that was executed
	Args      []string      // The arguments passed to the command
	Stdout    []byte        // Captured standard output
	Stderr    []byte        // Captured standard error
	ExitCode  int           // Exit code of the process (0 = success)
	Duration  time.Duration // How long the command took to run
	Error     error         // Error if the command failed to execute
	StartTime time.Time     // When the command started
	EndTime   time.Time     // When the command finished
}

// Success returns true if the command exited successfully (exit code 0 and no error).
func (r *Result) Success() bool {
	return r.ExitCode == 0 && r.Error == nil
}

// StdoutString returns stdout as a string.
func (r *Result) StdoutString() string {
	return string(r.Stdout)
}

// StderrString returns stderr as a string.
func (r *Result) StderrString() string {
	return string(r.Stderr)
}

// StdoutLines returns stdout split by newlines.
// Empty output returns a slice with a single empty string.
func (r *Result) StdoutLines() []string {
	trimmed := strings.TrimSpace(r.StdoutString())
	if trimmed == "" {
		return []string{}
	}
	return strings.Split(trimmed, "\n")
}

// StderrLines returns stderr split by newlines.
// Empty output returns a slice with a single empty string.
func (r *Result) StderrLines() []string {
	trimmed := strings.TrimSpace(r.StderrString())
	if trimmed == "" {
		return []string{}
	}
	return strings.Split(trimmed, "\n")
}

// CombinedOutput returns stdout and stderr combined.
func (r *Result) CombinedOutput() []byte {
	var buf bytes.Buffer
	buf.Write(r.Stdout)
	buf.Write(r.Stderr)
	return buf.Bytes()
}

// CombinedString returns combined output as string.
func (r *Result) CombinedString() string {
	return string(r.CombinedOutput())
}

// Failed returns true if the command failed (non-zero exit code or error).
func (r *Result) Failed() bool {
	return !r.Success()
}

// HasOutput returns true if there is any stdout output.
func (r *Result) HasOutput() bool {
	return len(r.Stdout) > 0
}

// HasError returns true if there is any stderr output.
func (r *Result) HasError() bool {
	return len(r.Stderr) > 0
}
