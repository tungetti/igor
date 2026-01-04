package install

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBaseStep(t *testing.T) {
	t.Run("creates with values", func(t *testing.T) {
		step := NewBaseStep("test-step", "Test step description", true)

		assert.Equal(t, "test-step", step.Name())
		assert.Equal(t, "Test step description", step.Description())
		assert.True(t, step.CanRollback())
	})

	t.Run("creates without rollback", func(t *testing.T) {
		step := NewBaseStep("no-rollback", "No rollback step", false)

		assert.Equal(t, "no-rollback", step.Name())
		assert.False(t, step.CanRollback())
	})
}

func TestFuncStep_Execute(t *testing.T) {
	t.Run("executes function successfully", func(t *testing.T) {
		executeCalled := false
		step := NewFuncStep("test", "Test step", func(ctx *Context) StepResult {
			executeCalled = true
			return CompleteStep("done")
		})

		ctx := NewContext()
		result := step.Execute(ctx)

		assert.True(t, executeCalled)
		assert.Equal(t, StepStatusCompleted, result.Status)
		assert.Equal(t, "done", result.Message)
		assert.Greater(t, result.Duration.Nanoseconds(), int64(0))
	})

	t.Run("handles nil execute function", func(t *testing.T) {
		step := &FuncStep{
			BaseStep:    NewBaseStep("nil-exec", "Nil execute", false),
			executeFunc: nil,
		}

		ctx := NewContext()
		result := step.Execute(ctx)

		assert.Equal(t, StepStatusFailed, result.Status)
		assert.Contains(t, result.Message, "no execute function")
	})

	t.Run("returns failure on error", func(t *testing.T) {
		testErr := errors.New("execution failed")
		step := NewFuncStep("fail", "Failing step", func(ctx *Context) StepResult {
			return FailStep("step failed", testErr)
		})

		ctx := NewContext()
		result := step.Execute(ctx)

		assert.Equal(t, StepStatusFailed, result.Status)
		assert.Equal(t, testErr, result.Error)
	})

	t.Run("tracks duration", func(t *testing.T) {
		step := NewFuncStep("timed", "Timed step", func(ctx *Context) StepResult {
			return CompleteStep("done")
		})

		ctx := NewContext()
		result := step.Execute(ctx)

		// Duration should be set (non-zero)
		assert.True(t, result.Duration >= 0)
	})
}

func TestFuncStep_Rollback(t *testing.T) {
	t.Run("calls rollback function", func(t *testing.T) {
		rollbackCalled := false
		step := NewFuncStep("rollback-test", "Test", func(ctx *Context) StepResult {
			return CompleteStep("done")
		}, WithRollbackFunc(func(ctx *Context) error {
			rollbackCalled = true
			return nil
		}))

		ctx := NewContext()
		err := step.Rollback(ctx)

		assert.NoError(t, err)
		assert.True(t, rollbackCalled)
		assert.True(t, step.CanRollback())
	})

	t.Run("returns error from rollback", func(t *testing.T) {
		testErr := errors.New("rollback failed")
		step := NewFuncStep("rollback-fail", "Test", func(ctx *Context) StepResult {
			return CompleteStep("done")
		}, WithRollbackFunc(func(ctx *Context) error {
			return testErr
		}))

		ctx := NewContext()
		err := step.Rollback(ctx)

		assert.Equal(t, testErr, err)
	})

	t.Run("handles nil rollback function", func(t *testing.T) {
		step := NewFuncStep("no-rollback", "Test", func(ctx *Context) StepResult {
			return CompleteStep("done")
		})

		ctx := NewContext()
		err := step.Rollback(ctx)

		assert.NoError(t, err)
		assert.False(t, step.CanRollback())
	})
}

func TestFuncStep_Validate(t *testing.T) {
	t.Run("calls validate function", func(t *testing.T) {
		validateCalled := false
		step := NewFuncStep("validate-test", "Test", func(ctx *Context) StepResult {
			return CompleteStep("done")
		}, WithValidateFunc(func(ctx *Context) error {
			validateCalled = true
			return nil
		}))

		ctx := NewContext()
		err := step.Validate(ctx)

		assert.NoError(t, err)
		assert.True(t, validateCalled)
	})

	t.Run("returns error from validate", func(t *testing.T) {
		testErr := errors.New("validation failed")
		step := NewFuncStep("validate-fail", "Test", func(ctx *Context) StepResult {
			return CompleteStep("done")
		}, WithValidateFunc(func(ctx *Context) error {
			return testErr
		}))

		ctx := NewContext()
		err := step.Validate(ctx)

		assert.Equal(t, testErr, err)
	})

	t.Run("handles nil validate function", func(t *testing.T) {
		step := NewFuncStep("no-validate", "Test", func(ctx *Context) StepResult {
			return CompleteStep("done")
		})

		ctx := NewContext()
		err := step.Validate(ctx)

		assert.NoError(t, err)
	})
}

func TestFuncStep_WithMultipleOptions(t *testing.T) {
	rollbackCalled := false
	validateCalled := false

	step := NewFuncStep("multi-option", "Test", func(ctx *Context) StepResult {
		return CompleteStep("done")
	},
		WithRollbackFunc(func(ctx *Context) error {
			rollbackCalled = true
			return nil
		}),
		WithValidateFunc(func(ctx *Context) error {
			validateCalled = true
			return nil
		}),
	)

	ctx := NewContext()

	require.NoError(t, step.Validate(ctx))
	assert.True(t, validateCalled)

	require.NoError(t, step.Rollback(ctx))
	assert.True(t, rollbackCalled)

	assert.True(t, step.CanRollback())
}

func TestSkipStep(t *testing.T) {
	result := SkipStep("already installed")

	assert.Equal(t, StepStatusSkipped, result.Status)
	assert.Equal(t, "already installed", result.Message)
	assert.Nil(t, result.Error)
}

func TestCompleteStep(t *testing.T) {
	result := CompleteStep("installation successful")

	assert.Equal(t, StepStatusCompleted, result.Status)
	assert.Equal(t, "installation successful", result.Message)
	assert.Nil(t, result.Error)
}

func TestFailStep(t *testing.T) {
	err := errors.New("something went wrong")
	result := FailStep("installation failed", err)

	assert.Equal(t, StepStatusFailed, result.Status)
	assert.Equal(t, "installation failed", result.Message)
	assert.Equal(t, err, result.Error)
}

func TestFuncStep_ContextAccess(t *testing.T) {
	t.Run("accesses context state", func(t *testing.T) {
		var capturedValue string
		step := NewFuncStep("state-access", "Test", func(ctx *Context) StepResult {
			capturedValue = ctx.GetStateString("test-key")
			return CompleteStep("done")
		})

		ctx := NewContext()
		ctx.SetState("test-key", "test-value")

		step.Execute(ctx)

		assert.Equal(t, "test-value", capturedValue)
	})

	t.Run("modifies context state", func(t *testing.T) {
		step := NewFuncStep("state-modify", "Test", func(ctx *Context) StepResult {
			ctx.SetState("result-key", "result-value")
			return CompleteStep("done")
		})

		ctx := NewContext()
		step.Execute(ctx)

		assert.Equal(t, "result-value", ctx.GetStateString("result-key"))
	})

	t.Run("respects dry run", func(t *testing.T) {
		var wasDryRun bool
		step := NewFuncStep("dry-run", "Test", func(ctx *Context) StepResult {
			wasDryRun = ctx.DryRun
			if ctx.DryRun {
				return SkipStep("dry run mode")
			}
			return CompleteStep("done")
		})

		ctx := NewContext(WithDryRun(true))
		result := step.Execute(ctx)

		assert.True(t, wasDryRun)
		assert.Equal(t, StepStatusSkipped, result.Status)
	})
}

// MockStep implements Step interface for testing
type MockStep struct {
	BaseStep
	executeResult  StepResult
	rollbackError  error
	validateError  error
	executeCalled  bool
	rollbackCalled bool
	validateCalled bool
}

func NewMockStep(name string, canRollback bool) *MockStep {
	return &MockStep{
		BaseStep:      NewBaseStep(name, "Mock step: "+name, canRollback),
		executeResult: CompleteStep("mock executed"),
	}
}

func (m *MockStep) Execute(ctx *Context) StepResult {
	m.executeCalled = true
	return m.executeResult
}

func (m *MockStep) Rollback(ctx *Context) error {
	m.rollbackCalled = true
	return m.rollbackError
}

func (m *MockStep) Validate(ctx *Context) error {
	m.validateCalled = true
	return m.validateError
}

func (m *MockStep) SetExecuteResult(result StepResult) {
	m.executeResult = result
}

func (m *MockStep) SetRollbackError(err error) {
	m.rollbackError = err
}

func (m *MockStep) SetValidateError(err error) {
	m.validateError = err
}

func TestMockStep(t *testing.T) {
	t.Run("tracks execute calls", func(t *testing.T) {
		step := NewMockStep("mock", false)

		ctx := NewContext()
		step.Execute(ctx)

		assert.True(t, step.executeCalled)
	})

	t.Run("returns configured result", func(t *testing.T) {
		step := NewMockStep("mock", false)
		step.SetExecuteResult(FailStep("configured failure", errors.New("test")))

		ctx := NewContext()
		result := step.Execute(ctx)

		assert.Equal(t, StepStatusFailed, result.Status)
	})
}
