package errors

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCode_String(t *testing.T) {
	tests := []struct {
		code     Code
		expected string
	}{
		{Unknown, "Unknown"},
		{DistroDetection, "DistroDetection"},
		{GPUDetection, "GPUDetection"},
		{PackageManager, "PackageManager"},
		{Installation, "Installation"},
		{Permission, "Permission"},
		{Network, "Network"},
		{Configuration, "Configuration"},
		{Validation, "Validation"},
		{Execution, "Execution"},
		{Timeout, "Timeout"},
		{NotFound, "NotFound"},
		{AlreadyExists, "AlreadyExists"},
		{Unsupported, "Unsupported"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.code.String())
		})
	}
}

func TestCode_String_Unknown(t *testing.T) {
	// Test an undefined code value
	code := Code(999)
	assert.Equal(t, "Code(999)", code.String())
}

func TestNew(t *testing.T) {
	err := New(GPUDetection, "no GPU found")

	require.NotNil(t, err)
	assert.Equal(t, GPUDetection, err.Code)
	assert.Equal(t, "no GPU found", err.Message)
	assert.Empty(t, err.Op)
	assert.Nil(t, err.Cause)
}

func TestNewf(t *testing.T) {
	err := Newf(Installation, "failed to install package %s", "nvidia-driver")

	require.NotNil(t, err)
	assert.Equal(t, Installation, err.Code)
	assert.Equal(t, "failed to install package nvidia-driver", err.Message)
	assert.Empty(t, err.Op)
	assert.Nil(t, err.Cause)
}

func TestWrap(t *testing.T) {
	cause := fmt.Errorf("underlying error")
	err := Wrap(Network, "connection failed", cause)

	require.NotNil(t, err)
	assert.Equal(t, Network, err.Code)
	assert.Equal(t, "connection failed", err.Message)
	assert.Equal(t, cause, err.Cause)
}

func TestWrapf(t *testing.T) {
	cause := fmt.Errorf("timeout")
	err := Wrapf(Execution, cause, "command %s failed", "apt-get")

	require.NotNil(t, err)
	assert.Equal(t, Execution, err.Code)
	assert.Equal(t, "command apt-get failed", err.Message)
	assert.Equal(t, cause, err.Cause)
}

func TestError_WithOp(t *testing.T) {
	err := New(GPUDetection, "detection failed").WithOp("gpu.Detect")

	assert.Equal(t, "gpu.Detect", err.Op)
	assert.Equal(t, GPUDetection, err.Code)
}

func TestError_WithOp_Chaining(t *testing.T) {
	// Test that WithOp returns the same error for chaining
	err := New(Permission, "access denied")
	result := err.WithOp("file.Open")

	assert.Same(t, err, result)
}

func TestError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *Error
		expected string
	}{
		{
			name:     "message only",
			err:      New(Unknown, "simple error"),
			expected: "simple error",
		},
		{
			name:     "with operation",
			err:      New(GPUDetection, "detection failed").WithOp("gpu.Detect"),
			expected: "gpu.Detect: detection failed",
		},
		{
			name:     "with cause",
			err:      Wrap(Network, "request failed", fmt.Errorf("timeout")),
			expected: "request failed: timeout",
		},
		{
			name:     "with operation and cause",
			err:      Wrap(Execution, "command failed", fmt.Errorf("exit code 1")).WithOp("exec.Run"),
			expected: "exec.Run: command failed: exit code 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.err.Error())
		})
	}
}

func TestError_Unwrap(t *testing.T) {
	cause := fmt.Errorf("original error")
	err := Wrap(Unknown, "wrapped", cause)

	unwrapped := err.Unwrap()

	assert.Equal(t, cause, unwrapped)
}

func TestError_Unwrap_NoCause(t *testing.T) {
	err := New(Unknown, "no cause")

	unwrapped := err.Unwrap()

	assert.Nil(t, unwrapped)
}

func TestError_Is(t *testing.T) {
	err1 := New(GPUDetection, "first error")
	err2 := New(GPUDetection, "second error")
	err3 := New(Network, "network error")

	// Same code should match
	assert.True(t, err1.Is(err2))
	assert.True(t, err2.Is(err1))

	// Different code should not match
	assert.False(t, err1.Is(err3))
	assert.False(t, err3.Is(err1))
}

func TestError_Is_NonError(t *testing.T) {
	err := New(GPUDetection, "error")
	stdErr := fmt.Errorf("standard error")

	// Should not match non-Error types
	assert.False(t, err.Is(stdErr))
}

func TestErrorsIs_Integration(t *testing.T) {
	// Test that errors.Is works with our custom Is method
	err1 := New(Permission, "permission denied")
	err2 := New(Permission, "another permission error")
	err3 := New(Network, "network error")

	// errors.Is should use our Is method
	assert.True(t, errors.Is(err1, err2))
	assert.False(t, errors.Is(err1, err3))
}

func TestErrorsIs_WithWrapping(t *testing.T) {
	// Test error chain with standard library wrapping
	innerErr := New(GPUDetection, "no GPU")
	wrappedErr := fmt.Errorf("wrapped: %w", innerErr)

	// Should find our error in the chain
	var e *Error
	assert.True(t, errors.As(wrappedErr, &e))
	assert.Equal(t, GPUDetection, e.Code)
}

func TestErrorsAs_Integration(t *testing.T) {
	// Test that errors.As works with our Error type
	err := New(Installation, "install failed")

	var e *Error
	ok := errors.As(err, &e)

	assert.True(t, ok)
	assert.Equal(t, Installation, e.Code)
	assert.Equal(t, "install failed", e.Message)
}

func TestErrorsAs_WithChain(t *testing.T) {
	// Test errors.As through our Wrap
	innerErr := New(Network, "connection refused")
	outerErr := Wrap(Execution, "operation failed", innerErr)

	var e *Error
	ok := errors.As(outerErr, &e)

	// Should match the outer error first
	assert.True(t, ok)
	assert.Equal(t, Execution, e.Code)

	// Can also unwrap to find inner error
	var inner *Error
	ok = errors.As(outerErr.Cause, &inner)
	assert.True(t, ok)
	assert.Equal(t, Network, inner.Code)
}

func TestGetCode(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected Code
	}{
		{
			name:     "Error type",
			err:      New(GPUDetection, "test"),
			expected: GPUDetection,
		},
		{
			name:     "wrapped Error",
			err:      Wrap(Permission, "wrapped", New(Network, "inner")),
			expected: Permission,
		},
		{
			name:     "standard error",
			err:      fmt.Errorf("standard"),
			expected: Unknown,
		},
		{
			name:     "nil error",
			err:      nil,
			expected: Unknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, GetCode(tt.err))
		})
	}
}

func TestGetCode_ChainedError(t *testing.T) {
	// Test GetCode with standard library wrapping
	innerErr := New(Validation, "invalid input")
	wrappedErr := fmt.Errorf("wrapper: %w", innerErr)

	// Should find the Error in the chain
	code := GetCode(wrappedErr)
	assert.Equal(t, Validation, code)
}

func TestIsCode(t *testing.T) {
	err := New(Permission, "not root")

	assert.True(t, IsCode(err, Permission))
	assert.False(t, IsCode(err, Network))
	assert.False(t, IsCode(nil, Permission))
}

func TestIsCode_StandardError(t *testing.T) {
	err := fmt.Errorf("standard error")

	// Standard errors should only match Unknown
	assert.True(t, IsCode(err, Unknown))
	assert.False(t, IsCode(err, Permission))
}

func TestSentinelErrors(t *testing.T) {
	// Test that sentinel errors are properly defined
	tests := []struct {
		name     string
		err      *Error
		code     Code
		contains string
	}{
		{"ErrNotRoot", ErrNotRoot, Permission, "root"},
		{"ErrNoGPU", ErrNoGPU, GPUDetection, "GPU"},
		{"ErrUnsupportedOS", ErrUnsupportedOS, Unsupported, "unsupported"},
		{"ErrTimeout", ErrTimeout, Timeout, "timed out"},
		{"ErrCancelled", ErrCancelled, Unknown, "cancelled"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.code, tt.err.Code)
			assert.Contains(t, tt.err.Error(), tt.contains)
		})
	}
}

func TestSentinelErrors_UsableWithErrorsIs(t *testing.T) {
	// Test that sentinel errors can be used with errors.Is
	err := Wrap(Permission, "failed", ErrNotRoot)

	// Can find the wrapped sentinel
	var e *Error
	require.True(t, errors.As(err.Cause, &e))
	assert.True(t, errors.Is(e, ErrNotRoot))
}

func TestError_ImplementsErrorInterface(t *testing.T) {
	// Compile-time check that Error implements error
	var _ error = &Error{}
	var _ error = New(Unknown, "test")
}

func TestNestedWrapping(t *testing.T) {
	// Test deeply nested error chains
	level1 := fmt.Errorf("root cause")
	level2 := Wrap(Network, "network layer", level1)
	level3 := Wrap(Execution, "execution layer", level2)
	level4 := fmt.Errorf("outer wrapper: %w", level3)

	// Should be able to extract the Error type
	var e *Error
	require.True(t, errors.As(level4, &e))
	assert.Equal(t, Execution, e.Code)

	// Can traverse the chain
	assert.True(t, errors.Is(level4, level3))

	// Can find the root cause
	var inner *Error
	require.True(t, errors.As(level3.Cause, &inner))
	assert.Equal(t, Network, inner.Code)
}

func TestError_ErrorWithEmptyMessage(t *testing.T) {
	err := New(Unknown, "")

	// Should handle empty message gracefully
	assert.Equal(t, "", err.Error())
}

func TestWrap_NilCause(t *testing.T) {
	// Wrapping with nil cause should work
	err := Wrap(Unknown, "no cause", nil)

	require.NotNil(t, err)
	assert.Nil(t, err.Cause)
	assert.Equal(t, "no cause", err.Error())
}

func TestCode_AllCodesHaveStrings(t *testing.T) {
	// Verify all defined codes have proper string representations
	// and are not the default fallback format
	codes := []Code{
		Unknown, DistroDetection, GPUDetection, PackageManager,
		Installation, Permission, Network, Configuration,
		Validation, Execution, Timeout, NotFound,
		AlreadyExists, Unsupported,
	}

	for _, code := range codes {
		str := code.String()
		assert.NotEmpty(t, str)
		assert.NotContains(t, str, "Code(", "code %d should have a defined string", code)
	}
}
