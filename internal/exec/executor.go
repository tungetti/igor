package exec

import (
	"bytes"
	"context"
	"io"
	"os/exec"
	"sync"
	"time"

	"github.com/tungetti/igor/internal/errors"
	"github.com/tungetti/igor/internal/privilege"
)

// Executor defines the interface for command execution.
// All implementations must be safe for concurrent use.
type Executor interface {
	// Execute runs a command and returns the result.
	Execute(ctx context.Context, cmd string, args ...string) *Result

	// ExecuteElevated runs a command with root privileges.
	ExecuteElevated(ctx context.Context, cmd string, args ...string) *Result

	// ExecuteWithInput runs a command with stdin input.
	ExecuteWithInput(ctx context.Context, input []byte, cmd string, args ...string) *Result

	// Stream runs a command and streams output to writers.
	Stream(ctx context.Context, stdout, stderr io.Writer, cmd string, args ...string) *Result
}

// Options configures the executor behavior.
type Options struct {
	Timeout     time.Duration // Default timeout for commands (0 = no timeout)
	WorkDir     string        // Working directory for command execution
	Env         []string      // Environment variables to set
	SanitizeEnv bool          // Whether to sanitize environment variables for security
}

// DefaultOptions returns sensible defaults for command execution.
func DefaultOptions() Options {
	return Options{
		Timeout:     2 * time.Minute,
		SanitizeEnv: true,
	}
}

// RealExecutor is the production implementation of Executor.
// It executes actual system commands and integrates with privilege.Manager
// for elevated command execution.
type RealExecutor struct {
	opts      Options
	privilege *privilege.Manager
	mu        sync.Mutex
}

// NewExecutor creates a new real executor with the given options and privilege manager.
// If priv is nil, elevated commands will run without privilege escalation.
func NewExecutor(opts Options, priv *privilege.Manager) *RealExecutor {
	return &RealExecutor{
		opts:      opts,
		privilege: priv,
	}
}

// Execute runs a command and returns the result.
func (e *RealExecutor) Execute(ctx context.Context, cmd string, args ...string) *Result {
	return e.execute(ctx, nil, cmd, args, false)
}

// ExecuteElevated runs a command with root privileges.
func (e *RealExecutor) ExecuteElevated(ctx context.Context, cmd string, args ...string) *Result {
	return e.execute(ctx, nil, cmd, args, true)
}

// ExecuteWithInput runs a command with stdin input.
func (e *RealExecutor) ExecuteWithInput(ctx context.Context, input []byte, cmd string, args ...string) *Result {
	return e.execute(ctx, input, cmd, args, false)
}

// Stream runs a command and streams output to writers.
func (e *RealExecutor) Stream(ctx context.Context, stdout, stderr io.Writer, cmd string, args ...string) *Result {
	return e.stream(ctx, stdout, stderr, cmd, args, false)
}

// StreamElevated runs a command with root privileges and streams output to writers.
func (e *RealExecutor) StreamElevated(ctx context.Context, stdout, stderr io.Writer, cmd string, args ...string) *Result {
	return e.stream(ctx, stdout, stderr, cmd, args, true)
}

func (e *RealExecutor) execute(ctx context.Context, input []byte, cmd string, args []string, elevated bool) *Result {
	result := &Result{
		Command:   cmd,
		Args:      args,
		StartTime: time.Now(),
	}

	// Apply timeout if not already in context
	if e.opts.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, e.opts.Timeout)
		defer cancel()
	}

	// Build command with elevation if needed
	execCmd, execArgs := cmd, args
	if elevated && e.privilege != nil && !e.privilege.IsRoot() {
		execCmd, execArgs = e.privilege.ElevatedCommand(cmd, args...)
	}

	c := exec.CommandContext(ctx, execCmd, execArgs...)

	// Set working directory
	if e.opts.WorkDir != "" {
		c.Dir = e.opts.WorkDir
	}

	// Set environment
	if e.opts.SanitizeEnv && e.privilege != nil {
		c.Env = e.privilege.SanitizedEnv()
	} else if len(e.opts.Env) > 0 {
		c.Env = e.opts.Env
	}

	// Set up input if provided
	if input != nil {
		c.Stdin = bytes.NewReader(input)
	}

	// Capture output
	var stdout, stderr bytes.Buffer
	c.Stdout = &stdout
	c.Stderr = &stderr

	// Run command
	err := c.Run()

	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)
	result.Stdout = stdout.Bytes()
	result.Stderr = stderr.Bytes()

	// Handle exit code and errors
	if err != nil {
		// Check context errors first - these take priority over exit errors
		// because the process may have been killed due to timeout/cancellation
		if ctx.Err() == context.DeadlineExceeded {
			result.Error = errors.Wrap(errors.Timeout, "command timed out", err)
			result.ExitCode = -1
		} else if ctx.Err() == context.Canceled {
			result.Error = errors.Wrap(errors.Unknown, "command cancelled", err)
			result.ExitCode = -1
		} else if exitErr, ok := err.(*exec.ExitError); ok {
			// Process exited with non-zero exit code
			result.ExitCode = exitErr.ExitCode()
		} else {
			// Other errors (e.g., command not found)
			result.Error = errors.Wrap(errors.Execution, "command execution failed", err)
			result.ExitCode = -1
		}
	}

	return result
}

func (e *RealExecutor) stream(ctx context.Context, stdout, stderr io.Writer, cmd string, args []string, elevated bool) *Result {
	result := &Result{
		Command:   cmd,
		Args:      args,
		StartTime: time.Now(),
	}

	// Apply timeout
	if e.opts.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, e.opts.Timeout)
		defer cancel()
	}

	// Build command with elevation if needed
	execCmd, execArgs := cmd, args
	if elevated && e.privilege != nil && !e.privilege.IsRoot() {
		execCmd, execArgs = e.privilege.ElevatedCommand(cmd, args...)
	}

	c := exec.CommandContext(ctx, execCmd, execArgs...)

	if e.opts.WorkDir != "" {
		c.Dir = e.opts.WorkDir
	}

	if e.opts.SanitizeEnv && e.privilege != nil {
		c.Env = e.privilege.SanitizedEnv()
	} else if len(e.opts.Env) > 0 {
		c.Env = e.opts.Env
	}

	// Stream output directly to provided writers
	// Also capture output in buffers for the result
	var stdoutBuf, stderrBuf bytes.Buffer
	if stdout != nil {
		c.Stdout = io.MultiWriter(stdout, &stdoutBuf)
	} else {
		c.Stdout = &stdoutBuf
	}
	if stderr != nil {
		c.Stderr = io.MultiWriter(stderr, &stderrBuf)
	} else {
		c.Stderr = &stderrBuf
	}

	err := c.Run()

	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)
	result.Stdout = stdoutBuf.Bytes()
	result.Stderr = stderrBuf.Bytes()

	if err != nil {
		// Check context errors first - these take priority over exit errors
		// because the process may have been killed due to timeout/cancellation
		if ctx.Err() == context.DeadlineExceeded {
			result.Error = errors.Wrap(errors.Timeout, "command timed out", err)
			result.ExitCode = -1
		} else if ctx.Err() == context.Canceled {
			result.Error = errors.Wrap(errors.Unknown, "command cancelled", err)
			result.ExitCode = -1
		} else if exitErr, ok := err.(*exec.ExitError); ok {
			// Process exited with non-zero exit code
			result.ExitCode = exitErr.ExitCode()
		} else {
			// Other errors (e.g., command not found)
			result.Error = errors.Wrap(errors.Execution, "command execution failed", err)
			result.ExitCode = -1
		}
	}

	return result
}

// Options returns the current executor options.
func (e *RealExecutor) Options() Options {
	return e.opts
}

// SetOptions updates the executor options.
func (e *RealExecutor) SetOptions(opts Options) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.opts = opts
}

// SetTimeout updates the default timeout.
func (e *RealExecutor) SetTimeout(timeout time.Duration) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.opts.Timeout = timeout
}

// SetWorkDir updates the working directory.
func (e *RealExecutor) SetWorkDir(workDir string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.opts.WorkDir = workDir
}
