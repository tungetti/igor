package exec

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tungetti/igor/internal/errors"
	"github.com/tungetti/igor/internal/privilege"
)

// =============================================================================
// Result Tests
// =============================================================================

func TestResult_Success(t *testing.T) {
	tests := []struct {
		name     string
		result   *Result
		expected bool
	}{
		{
			name:     "successful execution",
			result:   &Result{ExitCode: 0, Error: nil},
			expected: true,
		},
		{
			name:     "non-zero exit code",
			result:   &Result{ExitCode: 1, Error: nil},
			expected: false,
		},
		{
			name:     "error present",
			result:   &Result{ExitCode: 0, Error: errors.New(errors.Execution, "test error")},
			expected: false,
		},
		{
			name:     "both error and non-zero exit",
			result:   &Result{ExitCode: 1, Error: errors.New(errors.Execution, "test error")},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.result.Success())
		})
	}
}

func TestResult_Failed(t *testing.T) {
	tests := []struct {
		name     string
		result   *Result
		expected bool
	}{
		{
			name:     "successful execution",
			result:   &Result{ExitCode: 0, Error: nil},
			expected: false,
		},
		{
			name:     "non-zero exit code",
			result:   &Result{ExitCode: 1, Error: nil},
			expected: true,
		},
		{
			name:     "error present",
			result:   &Result{ExitCode: 0, Error: errors.New(errors.Execution, "test error")},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.result.Failed())
		})
	}
}

func TestResult_StdoutString(t *testing.T) {
	result := &Result{Stdout: []byte("hello world")}
	assert.Equal(t, "hello world", result.StdoutString())
}

func TestResult_StderrString(t *testing.T) {
	result := &Result{Stderr: []byte("error message")}
	assert.Equal(t, "error message", result.StderrString())
}

func TestResult_StdoutLines(t *testing.T) {
	tests := []struct {
		name     string
		stdout   string
		expected []string
	}{
		{
			name:     "single line",
			stdout:   "hello",
			expected: []string{"hello"},
		},
		{
			name:     "multiple lines",
			stdout:   "line1\nline2\nline3",
			expected: []string{"line1", "line2", "line3"},
		},
		{
			name:     "with trailing newline",
			stdout:   "line1\nline2\n",
			expected: []string{"line1", "line2"},
		},
		{
			name:     "empty output",
			stdout:   "",
			expected: []string{},
		},
		{
			name:     "whitespace only",
			stdout:   "  \n  ",
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &Result{Stdout: []byte(tt.stdout)}
			assert.Equal(t, tt.expected, result.StdoutLines())
		})
	}
}

func TestResult_StderrLines(t *testing.T) {
	tests := []struct {
		name     string
		stderr   string
		expected []string
	}{
		{
			name:     "single line",
			stderr:   "error",
			expected: []string{"error"},
		},
		{
			name:     "multiple lines",
			stderr:   "error1\nerror2",
			expected: []string{"error1", "error2"},
		},
		{
			name:     "empty output",
			stderr:   "",
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &Result{Stderr: []byte(tt.stderr)}
			assert.Equal(t, tt.expected, result.StderrLines())
		})
	}
}

func TestResult_CombinedOutput(t *testing.T) {
	result := &Result{
		Stdout: []byte("stdout content"),
		Stderr: []byte("stderr content"),
	}

	combined := result.CombinedOutput()
	assert.Equal(t, "stdout contentstderr content", string(combined))
}

func TestResult_CombinedString(t *testing.T) {
	result := &Result{
		Stdout: []byte("stdout "),
		Stderr: []byte("stderr"),
	}

	assert.Equal(t, "stdout stderr", result.CombinedString())
}

func TestResult_HasOutput(t *testing.T) {
	assert.True(t, (&Result{Stdout: []byte("data")}).HasOutput())
	assert.False(t, (&Result{Stdout: []byte{}}).HasOutput())
	assert.False(t, (&Result{}).HasOutput())
}

func TestResult_HasError(t *testing.T) {
	assert.True(t, (&Result{Stderr: []byte("error")}).HasError())
	assert.False(t, (&Result{Stderr: []byte{}}).HasError())
	assert.False(t, (&Result{}).HasError())
}

// =============================================================================
// RealExecutor Tests
// =============================================================================

func TestNewExecutor(t *testing.T) {
	opts := DefaultOptions()
	priv := privilege.NewManager()

	exec := NewExecutor(opts, priv)

	require.NotNil(t, exec)
	assert.Equal(t, opts.Timeout, exec.opts.Timeout)
	assert.Equal(t, opts.SanitizeEnv, exec.opts.SanitizeEnv)
}

func TestNewExecutor_NilPrivilege(t *testing.T) {
	opts := DefaultOptions()

	exec := NewExecutor(opts, nil)

	require.NotNil(t, exec)
	assert.Nil(t, exec.privilege)
}

func TestDefaultOptions(t *testing.T) {
	opts := DefaultOptions()

	assert.Equal(t, 2*time.Minute, opts.Timeout)
	assert.True(t, opts.SanitizeEnv)
	assert.Empty(t, opts.WorkDir)
	assert.Empty(t, opts.Env)
}

func TestRealExecutor_Execute_Success(t *testing.T) {
	exec := NewExecutor(DefaultOptions(), nil)
	ctx := context.Background()

	result := exec.Execute(ctx, "echo", "hello", "world")

	require.NotNil(t, result)
	assert.True(t, result.Success())
	assert.Equal(t, 0, result.ExitCode)
	assert.Contains(t, result.StdoutString(), "hello world")
	assert.Empty(t, result.Stderr)
	assert.Equal(t, "echo", result.Command)
	assert.Equal(t, []string{"hello", "world"}, result.Args)
	assert.True(t, result.Duration > 0)
}

func TestRealExecutor_Execute_Failure(t *testing.T) {
	exec := NewExecutor(DefaultOptions(), nil)
	ctx := context.Background()

	// Using false which always returns exit code 1
	result := exec.Execute(ctx, "false")

	require.NotNil(t, result)
	assert.False(t, result.Success())
	assert.Equal(t, 1, result.ExitCode)
	assert.Nil(t, result.Error) // Exit code errors don't set Error field
}

func TestRealExecutor_Execute_NonexistentCommand(t *testing.T) {
	exec := NewExecutor(DefaultOptions(), nil)
	ctx := context.Background()

	result := exec.Execute(ctx, "nonexistent_command_12345")

	require.NotNil(t, result)
	assert.False(t, result.Success())
	assert.Equal(t, -1, result.ExitCode)
	assert.NotNil(t, result.Error)
	assert.True(t, errors.IsCode(result.Error, errors.Execution))
}

func TestRealExecutor_Execute_CapturesStderr(t *testing.T) {
	exec := NewExecutor(DefaultOptions(), nil)
	ctx := context.Background()

	// ls with invalid option writes to stderr
	result := exec.Execute(ctx, "ls", "--invalid-option-12345")

	require.NotNil(t, result)
	assert.True(t, result.HasError())
	assert.Contains(t, result.StderrString(), "invalid")
}

func TestRealExecutor_Execute_Timeout(t *testing.T) {
	opts := Options{Timeout: 100 * time.Millisecond}
	exec := NewExecutor(opts, nil)
	ctx := context.Background()

	result := exec.Execute(ctx, "sleep", "10")

	require.NotNil(t, result)
	assert.False(t, result.Success())
	assert.Equal(t, -1, result.ExitCode)
	require.NotNil(t, result.Error)
	assert.True(t, errors.IsCode(result.Error, errors.Timeout))
}

func TestRealExecutor_Execute_ContextCancellation(t *testing.T) {
	exec := NewExecutor(Options{Timeout: 0}, nil) // No default timeout
	ctx, cancel := context.WithCancel(context.Background())

	// Cancel immediately
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	result := exec.Execute(ctx, "sleep", "10")

	require.NotNil(t, result)
	assert.False(t, result.Success())
}

func TestRealExecutor_ExecuteWithInput(t *testing.T) {
	exec := NewExecutor(DefaultOptions(), nil)
	ctx := context.Background()

	input := []byte("hello from stdin")
	result := exec.ExecuteWithInput(ctx, input, "cat")

	require.NotNil(t, result)
	assert.True(t, result.Success())
	assert.Equal(t, "hello from stdin", result.StdoutString())
}

func TestRealExecutor_ExecuteWithInput_EmptyInput(t *testing.T) {
	exec := NewExecutor(DefaultOptions(), nil)
	ctx := context.Background()

	result := exec.ExecuteWithInput(ctx, []byte{}, "cat")

	require.NotNil(t, result)
	assert.True(t, result.Success())
	assert.Empty(t, result.Stdout)
}

func TestRealExecutor_Stream(t *testing.T) {
	exec := NewExecutor(DefaultOptions(), nil)
	ctx := context.Background()

	var stdout, stderr bytes.Buffer
	result := exec.Stream(ctx, &stdout, &stderr, "echo", "streaming")

	require.NotNil(t, result)
	assert.True(t, result.Success())
	assert.Contains(t, stdout.String(), "streaming")
	// Result should also have captured output
	assert.Contains(t, result.StdoutString(), "streaming")
}

func TestRealExecutor_Stream_WithStderr(t *testing.T) {
	exec := NewExecutor(DefaultOptions(), nil)
	ctx := context.Background()

	var stdout, stderr bytes.Buffer
	result := exec.Stream(ctx, &stdout, &stderr, "ls", "--invalid-option-12345")

	require.NotNil(t, result)
	assert.True(t, stderr.Len() > 0)
	// Result should also have captured stderr
	assert.True(t, result.HasError())
}

func TestRealExecutor_Stream_NilWriters(t *testing.T) {
	exec := NewExecutor(DefaultOptions(), nil)
	ctx := context.Background()

	// Should not panic with nil writers
	result := exec.Stream(ctx, nil, nil, "echo", "test")

	require.NotNil(t, result)
	assert.True(t, result.Success())
	// Output should still be captured in result
	assert.Contains(t, result.StdoutString(), "test")
}

func TestRealExecutor_StreamElevated(t *testing.T) {
	priv := privilege.NewManager()
	priv.SetRoot(true) // Simulate running as root

	exec := NewExecutor(DefaultOptions(), priv)
	ctx := context.Background()

	var stdout, stderr bytes.Buffer
	result := exec.StreamElevated(ctx, &stdout, &stderr, "echo", "elevated stream")

	require.NotNil(t, result)
	assert.True(t, result.Success())
	assert.Contains(t, stdout.String(), "elevated stream")
	assert.Contains(t, result.StdoutString(), "elevated stream")
}

func TestRealExecutor_Stream_Timeout(t *testing.T) {
	opts := Options{Timeout: 100 * time.Millisecond}
	exec := NewExecutor(opts, nil)
	ctx := context.Background()

	var stdout, stderr bytes.Buffer
	result := exec.Stream(ctx, &stdout, &stderr, "sleep", "10")

	require.NotNil(t, result)
	assert.False(t, result.Success())
	assert.Equal(t, -1, result.ExitCode)
	require.NotNil(t, result.Error)
	assert.True(t, errors.IsCode(result.Error, errors.Timeout))
}

func TestRealExecutor_Stream_NonexistentCommand(t *testing.T) {
	exec := NewExecutor(DefaultOptions(), nil)
	ctx := context.Background()

	var stdout, stderr bytes.Buffer
	result := exec.Stream(ctx, &stdout, &stderr, "nonexistent_command_12345")

	require.NotNil(t, result)
	assert.False(t, result.Success())
	assert.Equal(t, -1, result.ExitCode)
	assert.NotNil(t, result.Error)
	assert.True(t, errors.IsCode(result.Error, errors.Execution))
}

func TestRealExecutor_Stream_ContextCancellation(t *testing.T) {
	exec := NewExecutor(Options{Timeout: 0}, nil)
	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	var stdout, stderr bytes.Buffer
	result := exec.Stream(ctx, &stdout, &stderr, "sleep", "10")

	require.NotNil(t, result)
	assert.False(t, result.Success())
}

func TestRealExecutor_Stream_WithEnv(t *testing.T) {
	opts := Options{
		Env:     []string{"TEST_VAR=hello_world"},
		Timeout: 30 * time.Second,
	}
	exec := NewExecutor(opts, nil)
	ctx := context.Background()

	var stdout, stderr bytes.Buffer
	result := exec.Stream(ctx, &stdout, &stderr, "sh", "-c", "echo $TEST_VAR")

	require.NotNil(t, result)
	assert.True(t, result.Success())
	assert.Contains(t, stdout.String(), "hello_world")
}

func TestRealExecutor_ExecuteElevated_AsRoot(t *testing.T) {
	priv := privilege.NewManager()
	priv.SetRoot(true) // Simulate running as root

	exec := NewExecutor(DefaultOptions(), priv)
	ctx := context.Background()

	result := exec.ExecuteElevated(ctx, "echo", "elevated")

	require.NotNil(t, result)
	assert.True(t, result.Success())
	assert.Contains(t, result.StdoutString(), "elevated")
}

func TestRealExecutor_ExecuteElevated_WithPrivilegeManager(t *testing.T) {
	priv := privilege.NewManager()
	exec := NewExecutor(DefaultOptions(), priv)
	ctx := context.Background()

	// Just verify it doesn't panic and returns a result
	// The actual elevation depends on sudo being available
	result := exec.ExecuteElevated(ctx, "echo", "test")

	require.NotNil(t, result)
	assert.Equal(t, "echo", result.Command)
}

func TestRealExecutor_ExecuteElevated_NilPrivilegeManager(t *testing.T) {
	exec := NewExecutor(DefaultOptions(), nil)
	ctx := context.Background()

	// Should run without elevation
	result := exec.ExecuteElevated(ctx, "echo", "test")

	require.NotNil(t, result)
	assert.True(t, result.Success())
}

func TestRealExecutor_WorkDir(t *testing.T) {
	opts := Options{
		WorkDir: "/tmp",
		Timeout: 30 * time.Second,
	}
	exec := NewExecutor(opts, nil)
	ctx := context.Background()

	result := exec.Execute(ctx, "pwd")

	require.NotNil(t, result)
	assert.True(t, result.Success())
	assert.Contains(t, result.StdoutString(), "/tmp")
}

func TestRealExecutor_SetOptions(t *testing.T) {
	exec := NewExecutor(DefaultOptions(), nil)

	newOpts := Options{
		Timeout: 5 * time.Minute,
		WorkDir: "/tmp",
	}
	exec.SetOptions(newOpts)

	assert.Equal(t, newOpts, exec.Options())
}

func TestRealExecutor_SetTimeout(t *testing.T) {
	exec := NewExecutor(DefaultOptions(), nil)

	exec.SetTimeout(5 * time.Minute)

	assert.Equal(t, 5*time.Minute, exec.opts.Timeout)
}

func TestRealExecutor_SetWorkDir(t *testing.T) {
	exec := NewExecutor(DefaultOptions(), nil)

	exec.SetWorkDir("/tmp")

	assert.Equal(t, "/tmp", exec.opts.WorkDir)
}

func TestRealExecutor_Duration(t *testing.T) {
	exec := NewExecutor(DefaultOptions(), nil)
	ctx := context.Background()

	result := exec.Execute(ctx, "sleep", "0.1")

	require.NotNil(t, result)
	assert.True(t, result.Duration >= 100*time.Millisecond)
	assert.False(t, result.StartTime.IsZero())
	assert.False(t, result.EndTime.IsZero())
	assert.True(t, result.EndTime.After(result.StartTime))
}

// =============================================================================
// MockExecutor Tests
// =============================================================================

func TestNewMockExecutor(t *testing.T) {
	mock := NewMockExecutor()

	require.NotNil(t, mock)
	assert.Empty(t, mock.Calls())
	assert.NotNil(t, mock.responses)
}

func TestMockExecutor_Execute(t *testing.T) {
	mock := NewMockExecutor()
	ctx := context.Background()

	result := mock.Execute(ctx, "test", "arg1", "arg2")

	require.NotNil(t, result)
	assert.Equal(t, 0, result.ExitCode)
	assert.Equal(t, "test", result.Command)
	assert.Equal(t, []string{"arg1", "arg2"}, result.Args)

	// Verify call was recorded
	calls := mock.Calls()
	require.Len(t, calls, 1)
	assert.Equal(t, "test", calls[0].Command)
	assert.Equal(t, []string{"arg1", "arg2"}, calls[0].Args)
	assert.False(t, calls[0].Elevated)
	assert.Nil(t, calls[0].Input)
}

func TestMockExecutor_ExecuteElevated(t *testing.T) {
	mock := NewMockExecutor()
	ctx := context.Background()

	mock.ExecuteElevated(ctx, "sudo_cmd", "arg")

	calls := mock.Calls()
	require.Len(t, calls, 1)
	assert.True(t, calls[0].Elevated)
}

func TestMockExecutor_ExecuteWithInput(t *testing.T) {
	mock := NewMockExecutor()
	ctx := context.Background()
	input := []byte("test input")

	mock.ExecuteWithInput(ctx, input, "cmd")

	calls := mock.Calls()
	require.Len(t, calls, 1)
	assert.Equal(t, input, calls[0].Input)
}

func TestMockExecutor_SetResponse(t *testing.T) {
	mock := NewMockExecutor()
	ctx := context.Background()

	expected := &Result{
		Stdout:   []byte("mocked output"),
		ExitCode: 0,
	}
	mock.SetResponse("mycommand", expected)

	result := mock.Execute(ctx, "mycommand", "arg")

	assert.Equal(t, "mocked output", result.StdoutString())
	assert.Equal(t, 0, result.ExitCode)
	assert.Equal(t, "mycommand", result.Command)
}

func TestMockExecutor_SetDefaultResponse(t *testing.T) {
	mock := NewMockExecutor()
	ctx := context.Background()

	defaultResult := &Result{
		Stdout:   []byte("default output"),
		ExitCode: 42,
	}
	mock.SetDefaultResponse(defaultResult)

	result := mock.Execute(ctx, "unknown_cmd")

	assert.Equal(t, "default output", result.StdoutString())
	assert.Equal(t, 42, result.ExitCode)
}

func TestMockExecutor_SetResponse_OverridesDefault(t *testing.T) {
	mock := NewMockExecutor()
	ctx := context.Background()

	mock.SetDefaultResponse(&Result{Stdout: []byte("default")})
	mock.SetResponse("specific", &Result{Stdout: []byte("specific")})

	result := mock.Execute(ctx, "specific")

	assert.Equal(t, "specific", result.StdoutString())
}

func TestMockExecutor_Stream(t *testing.T) {
	mock := NewMockExecutor()
	ctx := context.Background()

	mock.SetResponse("cmd", &Result{
		Stdout: []byte("stdout content"),
		Stderr: []byte("stderr content"),
	})

	var stdout, stderr bytes.Buffer
	result := mock.Stream(ctx, &stdout, &stderr, "cmd")

	assert.Equal(t, "stdout content", stdout.String())
	assert.Equal(t, "stderr content", stderr.String())
	assert.Equal(t, "stdout content", result.StdoutString())
}

func TestMockExecutor_Stream_NilWriters(t *testing.T) {
	mock := NewMockExecutor()
	ctx := context.Background()

	mock.SetResponse("cmd", &Result{Stdout: []byte("output")})

	// Should not panic
	result := mock.Stream(ctx, nil, nil, "cmd")

	assert.Equal(t, "output", result.StdoutString())
}

func TestMockExecutor_Reset(t *testing.T) {
	mock := NewMockExecutor()
	ctx := context.Background()

	mock.Execute(ctx, "cmd1")
	mock.Execute(ctx, "cmd2")
	mock.SetResponse("cmd", SuccessResult("test"))

	mock.Reset()

	assert.Empty(t, mock.Calls())
	// Responses should still be set
	result := mock.Execute(ctx, "cmd")
	assert.Equal(t, "test", result.StdoutString())
}

func TestMockExecutor_ResetAll(t *testing.T) {
	mock := NewMockExecutor()
	ctx := context.Background()

	mock.Execute(ctx, "cmd")
	mock.SetResponse("cmd", SuccessResult("test"))
	mock.SetDefaultResponse(SuccessResult("default"))

	mock.ResetAll()

	assert.Empty(t, mock.Calls())
	// Responses should be cleared
	result := mock.Execute(ctx, "cmd")
	assert.Empty(t, result.StdoutString())
}

func TestMockExecutor_CallCount(t *testing.T) {
	mock := NewMockExecutor()
	ctx := context.Background()

	assert.Equal(t, 0, mock.CallCount())

	mock.Execute(ctx, "cmd1")
	assert.Equal(t, 1, mock.CallCount())

	mock.Execute(ctx, "cmd2")
	mock.ExecuteElevated(ctx, "cmd3")
	assert.Equal(t, 3, mock.CallCount())
}

func TestMockExecutor_LastCall(t *testing.T) {
	mock := NewMockExecutor()
	ctx := context.Background()

	// Empty calls
	assert.Equal(t, MockCall{}, mock.LastCall())

	mock.Execute(ctx, "cmd1", "arg1")
	mock.Execute(ctx, "cmd2", "arg2")

	last := mock.LastCall()
	assert.Equal(t, "cmd2", last.Command)
	assert.Equal(t, []string{"arg2"}, last.Args)
}

func TestMockExecutor_WasCalled(t *testing.T) {
	mock := NewMockExecutor()
	ctx := context.Background()

	assert.False(t, mock.WasCalled("cmd"))

	mock.Execute(ctx, "cmd", "arg")

	assert.True(t, mock.WasCalled("cmd"))
	assert.False(t, mock.WasCalled("other"))
}

func TestMockExecutor_WasCalledWith(t *testing.T) {
	mock := NewMockExecutor()
	ctx := context.Background()

	mock.Execute(ctx, "cmd", "arg1", "arg2")

	assert.True(t, mock.WasCalledWith("cmd", "arg1", "arg2"))
	assert.False(t, mock.WasCalledWith("cmd", "arg1"))
	assert.False(t, mock.WasCalledWith("cmd", "arg1", "arg3"))
	assert.False(t, mock.WasCalledWith("other", "arg1", "arg2"))
}

// =============================================================================
// Helper Function Tests
// =============================================================================

func TestSuccessResult(t *testing.T) {
	result := SuccessResult("output text")

	assert.Equal(t, 0, result.ExitCode)
	assert.Equal(t, "output text", result.StdoutString())
	assert.True(t, result.Success())
	assert.False(t, result.StartTime.IsZero())
}

func TestSuccessResultWithStderr(t *testing.T) {
	result := SuccessResultWithStderr("stdout", "stderr")

	assert.Equal(t, 0, result.ExitCode)
	assert.Equal(t, "stdout", result.StdoutString())
	assert.Equal(t, "stderr", result.StderrString())
	assert.True(t, result.Success())
}

func TestFailureResult(t *testing.T) {
	result := FailureResult(42, "error message")

	assert.Equal(t, 42, result.ExitCode)
	assert.Equal(t, "error message", result.StderrString())
	assert.False(t, result.Success())
}

func TestErrorResult(t *testing.T) {
	err := errors.New(errors.Execution, "test error")
	result := ErrorResult(err)

	assert.Equal(t, -1, result.ExitCode)
	assert.Equal(t, err, result.Error)
	assert.False(t, result.Success())
}

func TestSlicesEqual(t *testing.T) {
	tests := []struct {
		name     string
		a        []string
		b        []string
		expected bool
	}{
		{"both empty", []string{}, []string{}, true},
		{"both nil", nil, nil, true},
		{"equal", []string{"a", "b"}, []string{"a", "b"}, true},
		{"different length", []string{"a"}, []string{"a", "b"}, false},
		{"different content", []string{"a", "b"}, []string{"a", "c"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, slicesEqual(tt.a, tt.b))
		})
	}
}

// =============================================================================
// Interface Compliance Tests
// =============================================================================

func TestExecutorInterface(t *testing.T) {
	// Ensure both implementations satisfy the Executor interface
	var _ Executor = (*RealExecutor)(nil)
	var _ Executor = (*MockExecutor)(nil)
}

// =============================================================================
// Concurrency Tests
// =============================================================================

func TestMockExecutor_Concurrent(t *testing.T) {
	mock := NewMockExecutor()
	ctx := context.Background()
	mock.SetDefaultResponse(SuccessResult("output"))

	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(n int) {
			for j := 0; j < 100; j++ {
				mock.Execute(ctx, "cmd", "arg")
			}
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	assert.Equal(t, 1000, mock.CallCount())
}

func TestRealExecutor_Concurrent(t *testing.T) {
	exec := NewExecutor(DefaultOptions(), nil)
	ctx := context.Background()

	done := make(chan bool)
	for i := 0; i < 5; i++ {
		go func() {
			for j := 0; j < 10; j++ {
				result := exec.Execute(ctx, "echo", "test")
				assert.True(t, result.Success())
			}
			done <- true
		}()
	}

	for i := 0; i < 5; i++ {
		<-done
	}
}

// =============================================================================
// Edge Cases
// =============================================================================

func TestRealExecutor_EmptyCommand(t *testing.T) {
	exec := NewExecutor(DefaultOptions(), nil)
	ctx := context.Background()

	result := exec.Execute(ctx, "")

	assert.False(t, result.Success())
	assert.NotNil(t, result.Error)
}

func TestRealExecutor_SpecialCharactersInArgs(t *testing.T) {
	exec := NewExecutor(DefaultOptions(), nil)
	ctx := context.Background()

	result := exec.Execute(ctx, "echo", "hello world", "with\nnewline", "$variable")

	require.NotNil(t, result)
	assert.True(t, result.Success())
	assert.Contains(t, result.StdoutString(), "hello world")
}

func TestMockExecutor_MultipleSameCommand(t *testing.T) {
	mock := NewMockExecutor()
	ctx := context.Background()

	mock.Execute(ctx, "cmd", "first")
	mock.Execute(ctx, "cmd", "second")
	mock.Execute(ctx, "cmd", "third")

	calls := mock.Calls()
	assert.Len(t, calls, 3)
	assert.Equal(t, []string{"first"}, calls[0].Args)
	assert.Equal(t, []string{"second"}, calls[1].Args)
	assert.Equal(t, []string{"third"}, calls[2].Args)
}
