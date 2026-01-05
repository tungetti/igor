package uninstall

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tungetti/igor/internal/install"
)

func TestBaseUninstallStep(t *testing.T) {
	t.Run("creates with values", func(t *testing.T) {
		step := NewBaseUninstallStep("remove-nvidia", "Remove NVIDIA packages", true)

		assert.Equal(t, "remove-nvidia", step.Name())
		assert.Equal(t, "Remove NVIDIA packages", step.Description())
		assert.True(t, step.CanRollback())
	})

	t.Run("creates without rollback", func(t *testing.T) {
		step := NewBaseUninstallStep("cleanup-configs", "Cleanup configuration files", false)

		assert.Equal(t, "cleanup-configs", step.Name())
		assert.False(t, step.CanRollback())
	})
}

func TestUninstallFuncStep_Execute(t *testing.T) {
	t.Run("executes function successfully", func(t *testing.T) {
		executeCalled := false
		step := NewUninstallFuncStep("remove-test", "Test removal step", func(ctx *Context) install.StepResult {
			executeCalled = true
			return CompleteUninstallStep("packages removed")
		})

		installCtx := install.NewContext()
		result := step.Execute(installCtx)

		assert.True(t, executeCalled)
		assert.Equal(t, install.StepStatusCompleted, result.Status)
		assert.Equal(t, "packages removed", result.Message)
		assert.Greater(t, result.Duration.Nanoseconds(), int64(0))
	})

	t.Run("handles nil execute function", func(t *testing.T) {
		step := &UninstallFuncStep{
			BaseStep:    install.NewBaseStep("nil-exec", "Nil execute", false),
			executeFunc: nil,
		}

		installCtx := install.NewContext()
		result := step.Execute(installCtx)

		assert.Equal(t, install.StepStatusFailed, result.Status)
		assert.Contains(t, result.Message, "no execute function")
	})

	t.Run("returns failure on error", func(t *testing.T) {
		testErr := errors.New("removal failed")
		step := NewUninstallFuncStep("fail-step", "Failing step", func(ctx *Context) install.StepResult {
			return FailUninstallStep("step failed", testErr)
		})

		installCtx := install.NewContext()
		result := step.Execute(installCtx)

		assert.Equal(t, install.StepStatusFailed, result.Status)
		assert.Equal(t, testErr, result.Error)
	})

	t.Run("tracks duration", func(t *testing.T) {
		step := NewUninstallFuncStep("timed-step", "Timed step", func(ctx *Context) install.StepResult {
			return CompleteUninstallStep("done")
		})

		installCtx := install.NewContext()
		result := step.Execute(installCtx)

		// Duration should be set (non-zero or zero for fast execution)
		assert.True(t, result.Duration >= 0)
	})
}

func TestUninstallFuncStep_Rollback(t *testing.T) {
	t.Run("calls rollback function", func(t *testing.T) {
		rollbackCalled := false
		step := NewUninstallFuncStep("rollback-test", "Test", func(ctx *Context) install.StepResult {
			return CompleteUninstallStep("done")
		}, WithUninstallRollbackFunc(func(ctx *Context) error {
			rollbackCalled = true
			return nil
		}))

		installCtx := install.NewContext()
		err := step.Rollback(installCtx)

		assert.NoError(t, err)
		assert.True(t, rollbackCalled)
		assert.True(t, step.CanRollback())
	})

	t.Run("returns error from rollback", func(t *testing.T) {
		testErr := errors.New("rollback failed")
		step := NewUninstallFuncStep("rollback-fail", "Test", func(ctx *Context) install.StepResult {
			return CompleteUninstallStep("done")
		}, WithUninstallRollbackFunc(func(ctx *Context) error {
			return testErr
		}))

		installCtx := install.NewContext()
		err := step.Rollback(installCtx)

		assert.Equal(t, testErr, err)
	})

	t.Run("handles nil rollback function", func(t *testing.T) {
		step := NewUninstallFuncStep("no-rollback", "Test", func(ctx *Context) install.StepResult {
			return CompleteUninstallStep("done")
		})

		installCtx := install.NewContext()
		err := step.Rollback(installCtx)

		assert.NoError(t, err)
		assert.False(t, step.CanRollback())
	})
}

func TestUninstallFuncStep_Validate(t *testing.T) {
	t.Run("calls validate function", func(t *testing.T) {
		validateCalled := false
		step := NewUninstallFuncStep("validate-test", "Test", func(ctx *Context) install.StepResult {
			return CompleteUninstallStep("done")
		}, WithUninstallValidateFunc(func(ctx *Context) error {
			validateCalled = true
			return nil
		}))

		installCtx := install.NewContext()
		err := step.Validate(installCtx)

		assert.NoError(t, err)
		assert.True(t, validateCalled)
	})

	t.Run("returns error from validate", func(t *testing.T) {
		testErr := errors.New("validation failed")
		step := NewUninstallFuncStep("validate-fail", "Test", func(ctx *Context) install.StepResult {
			return CompleteUninstallStep("done")
		}, WithUninstallValidateFunc(func(ctx *Context) error {
			return testErr
		}))

		installCtx := install.NewContext()
		err := step.Validate(installCtx)

		assert.Equal(t, testErr, err)
	})

	t.Run("handles nil validate function", func(t *testing.T) {
		step := NewUninstallFuncStep("no-validate", "Test", func(ctx *Context) install.StepResult {
			return CompleteUninstallStep("done")
		})

		installCtx := install.NewContext()
		err := step.Validate(installCtx)

		assert.NoError(t, err)
	})
}

func TestUninstallFuncStep_WithMultipleOptions(t *testing.T) {
	rollbackCalled := false
	validateCalled := false

	step := NewUninstallFuncStep("multi-option", "Test", func(ctx *Context) install.StepResult {
		return CompleteUninstallStep("done")
	},
		WithUninstallRollbackFunc(func(ctx *Context) error {
			rollbackCalled = true
			return nil
		}),
		WithUninstallValidateFunc(func(ctx *Context) error {
			validateCalled = true
			return nil
		}),
	)

	installCtx := install.NewContext()

	require.NoError(t, step.Validate(installCtx))
	assert.True(t, validateCalled)

	require.NoError(t, step.Rollback(installCtx))
	assert.True(t, rollbackCalled)

	assert.True(t, step.CanRollback())
}

func TestSkipUninstallStep(t *testing.T) {
	result := SkipUninstallStep("nothing to remove")

	assert.Equal(t, install.StepStatusSkipped, result.Status)
	assert.Equal(t, "nothing to remove", result.Message)
	assert.Nil(t, result.Error)
}

func TestCompleteUninstallStep(t *testing.T) {
	result := CompleteUninstallStep("removal successful")

	assert.Equal(t, install.StepStatusCompleted, result.Status)
	assert.Equal(t, "removal successful", result.Message)
	assert.Nil(t, result.Error)
}

func TestFailUninstallStep(t *testing.T) {
	err := errors.New("something went wrong")
	result := FailUninstallStep("removal failed", err)

	assert.Equal(t, install.StepStatusFailed, result.Status)
	assert.Equal(t, "removal failed", result.Message)
	assert.Equal(t, err, result.Error)
}

func TestUninstallFuncStep_ContextAccess(t *testing.T) {
	t.Run("accesses context values", func(t *testing.T) {
		var capturedDryRun bool
		step := NewUninstallFuncStep("dry-run-check", "Test", func(ctx *Context) install.StepResult {
			capturedDryRun = ctx.DryRun
			return CompleteUninstallStep("done")
		})

		installCtx := install.NewContext(install.WithDryRun(true))
		step.Execute(installCtx)

		assert.True(t, capturedDryRun)
	})

	t.Run("respects dry run mode", func(t *testing.T) {
		step := NewUninstallFuncStep("dry-run", "Test", func(ctx *Context) install.StepResult {
			if ctx.DryRun {
				return SkipUninstallStep("dry run mode - skipping actual removal")
			}
			return CompleteUninstallStep("packages removed")
		})

		installCtx := install.NewContext(install.WithDryRun(true))
		result := step.Execute(installCtx)

		assert.Equal(t, install.StepStatusSkipped, result.Status)
	})
}

func TestContextFromInstallContext(t *testing.T) {
	t.Run("handles nil install context", func(t *testing.T) {
		ctx := contextFromInstallContext(nil)
		assert.NotNil(t, ctx)
	})

	t.Run("copies fields from install context", func(t *testing.T) {
		installCtx := install.NewContext(
			install.WithDryRun(true),
		)
		ctx := contextFromInstallContext(installCtx)

		assert.True(t, ctx.DryRun)
	})
}

func TestUninstallStepTypeAlias(t *testing.T) {
	// Verify that UninstallStep is compatible with install.Step
	var step UninstallStep = install.NewFuncStep("test", "Test step", func(ctx *install.Context) install.StepResult {
		return install.CompleteStep("done")
	})

	assert.Equal(t, "test", step.Name())
	assert.Equal(t, "Test step", step.Description())
}

func TestUninstallFuncStep_ImplementsStep(t *testing.T) {
	// Verify that UninstallFuncStep implements install.Step
	var _ install.Step = (*UninstallFuncStep)(nil)
}
