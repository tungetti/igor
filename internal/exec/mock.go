package exec

import (
	"context"
	"io"
	"sync"
	"time"
)

// MockExecutor is a test implementation of Executor that records calls
// and returns pre-configured responses. It is safe for concurrent use.
type MockExecutor struct {
	mu            sync.Mutex
	responses     map[string]*Result
	calls         []MockCall
	defaultResult *Result
}

// MockCall records a call to the mock executor.
type MockCall struct {
	Command  string   // The command that was called
	Args     []string // The arguments passed
	Elevated bool     // Whether ExecuteElevated was used
	Input    []byte   // The input provided (for ExecuteWithInput)
}

// NewMockExecutor creates a new mock executor.
func NewMockExecutor() *MockExecutor {
	return &MockExecutor{
		responses: make(map[string]*Result),
	}
}

// SetResponse sets a canned response for a specific command.
// The command is used as the key for lookup.
func (m *MockExecutor) SetResponse(cmd string, result *Result) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.responses[cmd] = result
}

// SetDefaultResponse sets the default response for commands without a specific response.
func (m *MockExecutor) SetDefaultResponse(result *Result) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.defaultResult = result
}

// Calls returns all recorded calls made to the mock.
// Returns a copy of the slice to prevent external modification.
func (m *MockExecutor) Calls() []MockCall {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]MockCall{}, m.calls...)
}

// CallCount returns the number of calls made to the mock.
func (m *MockExecutor) CallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.calls)
}

// LastCall returns the most recent call, or an empty MockCall if no calls were made.
func (m *MockExecutor) LastCall() MockCall {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.calls) == 0 {
		return MockCall{}
	}
	return m.calls[len(m.calls)-1]
}

// Reset clears all recorded calls but keeps responses.
func (m *MockExecutor) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = nil
}

// ResetAll clears all recorded calls and responses.
func (m *MockExecutor) ResetAll() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = nil
	m.responses = make(map[string]*Result)
	m.defaultResult = nil
}

// WasCalled returns true if the given command was called.
func (m *MockExecutor) WasCalled(cmd string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, call := range m.calls {
		if call.Command == cmd {
			return true
		}
	}
	return false
}

// WasCalledWith returns true if the given command was called with the specified args.
func (m *MockExecutor) WasCalledWith(cmd string, args ...string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, call := range m.calls {
		if call.Command == cmd && slicesEqual(call.Args, args) {
			return true
		}
	}
	return false
}

// slicesEqual compares two string slices for equality.
func slicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// Execute implements Executor.
func (m *MockExecutor) Execute(ctx context.Context, cmd string, args ...string) *Result {
	return m.record(cmd, args, false, nil)
}

// ExecuteElevated implements Executor.
func (m *MockExecutor) ExecuteElevated(ctx context.Context, cmd string, args ...string) *Result {
	return m.record(cmd, args, true, nil)
}

// ExecuteWithInput implements Executor.
func (m *MockExecutor) ExecuteWithInput(ctx context.Context, input []byte, cmd string, args ...string) *Result {
	return m.record(cmd, args, false, input)
}

// Stream implements Executor.
func (m *MockExecutor) Stream(ctx context.Context, stdout, stderr io.Writer, cmd string, args ...string) *Result {
	result := m.record(cmd, args, false, nil)
	if stdout != nil && result.Stdout != nil {
		stdout.Write(result.Stdout)
	}
	if stderr != nil && result.Stderr != nil {
		stderr.Write(result.Stderr)
	}
	return result
}

func (m *MockExecutor) record(cmd string, args []string, elevated bool, input []byte) *Result {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.calls = append(m.calls, MockCall{
		Command:  cmd,
		Args:     args,
		Elevated: elevated,
		Input:    input,
	})

	if result, ok := m.responses[cmd]; ok {
		// Return a copy with command info filled in
		return &Result{
			Command:   cmd,
			Args:      args,
			Stdout:    result.Stdout,
			Stderr:    result.Stderr,
			ExitCode:  result.ExitCode,
			Duration:  result.Duration,
			Error:     result.Error,
			StartTime: result.StartTime,
			EndTime:   result.EndTime,
		}
	}

	if m.defaultResult != nil {
		return &Result{
			Command:   cmd,
			Args:      args,
			Stdout:    m.defaultResult.Stdout,
			Stderr:    m.defaultResult.Stderr,
			ExitCode:  m.defaultResult.ExitCode,
			Duration:  m.defaultResult.Duration,
			Error:     m.defaultResult.Error,
			StartTime: m.defaultResult.StartTime,
			EndTime:   m.defaultResult.EndTime,
		}
	}

	// Default: return a successful empty result
	now := time.Now()
	return &Result{
		Command:   cmd,
		Args:      args,
		ExitCode:  0,
		StartTime: now,
		EndTime:   now,
	}
}

// SuccessResult creates a successful result with the given stdout.
func SuccessResult(stdout string) *Result {
	now := time.Now()
	return &Result{
		ExitCode:  0,
		Stdout:    []byte(stdout),
		StartTime: now,
		EndTime:   now,
	}
}

// SuccessResultWithStderr creates a successful result with stdout and stderr.
func SuccessResultWithStderr(stdout, stderr string) *Result {
	now := time.Now()
	return &Result{
		ExitCode:  0,
		Stdout:    []byte(stdout),
		Stderr:    []byte(stderr),
		StartTime: now,
		EndTime:   now,
	}
}

// FailureResult creates a failed result with the given exit code and stderr.
func FailureResult(exitCode int, stderr string) *Result {
	now := time.Now()
	return &Result{
		ExitCode:  exitCode,
		Stderr:    []byte(stderr),
		StartTime: now,
		EndTime:   now,
	}
}

// ErrorResult creates a result with an execution error.
func ErrorResult(err error) *Result {
	now := time.Now()
	return &Result{
		ExitCode:  -1,
		Error:     err,
		StartTime: now,
		EndTime:   now,
	}
}

// Ensure MockExecutor implements Executor.
var _ Executor = (*MockExecutor)(nil)
