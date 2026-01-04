package install

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewWorkflow(t *testing.T) {
	w := NewWorkflow("test-workflow")

	assert.Equal(t, "test-workflow", w.Name())
	assert.Empty(t, w.Steps())
	assert.Empty(t, w.CompletedSteps())
	assert.False(t, w.IsCancelled())
}

func TestBaseWorkflow_AddStep(t *testing.T) {
	w := NewWorkflow("test")

	step1 := NewMockStep("step1", false)
	step2 := NewMockStep("step2", true)

	w.AddStep(step1)
	w.AddStep(step2)

	steps := w.Steps()
	assert.Len(t, steps, 2)
	assert.Equal(t, "step1", steps[0].Name())
	assert.Equal(t, "step2", steps[1].Name())
}

func TestBaseWorkflow_Execute_Success(t *testing.T) {
	w := NewWorkflow("test")

	step1 := NewMockStep("step1", false)
	step2 := NewMockStep("step2", false)
	step3 := NewMockStep("step3", false)

	w.AddStep(step1)
	w.AddStep(step2)
	w.AddStep(step3)

	ctx := NewContext()
	result := w.Execute(ctx)

	assert.Equal(t, WorkflowStatusCompleted, result.Status)
	assert.Nil(t, result.Error)
	assert.Empty(t, result.FailedStep)
	assert.Equal(t, []string{"step1", "step2", "step3"}, result.CompletedSteps)
	assert.True(t, result.TotalDuration > 0)

	assert.True(t, step1.executeCalled)
	assert.True(t, step2.executeCalled)
	assert.True(t, step3.executeCalled)
}

func TestBaseWorkflow_Execute_StepFailure(t *testing.T) {
	w := NewWorkflow("test")

	step1 := NewMockStep("step1", false)
	step2 := NewMockStep("step2", false)
	step2.SetExecuteResult(FailStep("step failed", errors.New("test error")))
	step3 := NewMockStep("step3", false)

	w.AddStep(step1)
	w.AddStep(step2)
	w.AddStep(step3)

	ctx := NewContext()
	result := w.Execute(ctx)

	assert.Equal(t, WorkflowStatusFailed, result.Status)
	assert.Equal(t, "step2", result.FailedStep)
	assert.Error(t, result.Error)
	assert.Equal(t, []string{"step1"}, result.CompletedSteps)

	assert.True(t, step1.executeCalled)
	assert.True(t, step2.executeCalled)
	assert.False(t, step3.executeCalled) // Should not be called after failure
}

func TestBaseWorkflow_Execute_ValidationFailure(t *testing.T) {
	w := NewWorkflow("test")

	step1 := NewMockStep("step1", false)
	step2 := NewMockStep("step2", false)
	step2.SetValidateError(errors.New("validation failed"))

	w.AddStep(step1)
	w.AddStep(step2)

	ctx := NewContext()
	result := w.Execute(ctx)

	assert.Equal(t, WorkflowStatusFailed, result.Status)
	assert.Equal(t, "step2", result.FailedStep)
	assert.Contains(t, result.Error.Error(), "validation failed")
	assert.Equal(t, []string{"step1"}, result.CompletedSteps)

	assert.True(t, step1.executeCalled)
	assert.True(t, step2.validateCalled)
	assert.False(t, step2.executeCalled) // Should not execute after validation failure
}

func TestBaseWorkflow_Execute_SkippedStep(t *testing.T) {
	w := NewWorkflow("test")

	step1 := NewMockStep("step1", false)
	step2 := NewMockStep("step2", false)
	step2.SetExecuteResult(SkipStep("already done"))
	step3 := NewMockStep("step3", false)

	w.AddStep(step1)
	w.AddStep(step2)
	w.AddStep(step3)

	ctx := NewContext()
	result := w.Execute(ctx)

	assert.Equal(t, WorkflowStatusCompleted, result.Status)
	// Skipped steps are not added to completed
	assert.Equal(t, []string{"step1", "step3"}, result.CompletedSteps)

	assert.True(t, step1.executeCalled)
	assert.True(t, step2.executeCalled)
	assert.True(t, step3.executeCalled)
}

func TestBaseWorkflow_Execute_Cancel(t *testing.T) {
	w := NewWorkflow("test")

	step1 := NewMockStep("step1", false)
	step2 := NewFuncStep("step2", "Slow step", func(ctx *Context) StepResult {
		time.Sleep(100 * time.Millisecond)
		return CompleteStep("done")
	})
	step3 := NewMockStep("step3", false)

	w.AddStep(step1)
	w.AddStep(step2)
	w.AddStep(step3)

	ctx := NewContext()

	// Cancel immediately after starting
	go func() {
		time.Sleep(10 * time.Millisecond)
		w.Cancel()
	}()

	result := w.Execute(ctx)

	// Should be cancelled before completing all steps
	assert.Equal(t, WorkflowStatusCancelled, result.Status)
	assert.True(t, w.IsCancelled())
}

func TestBaseWorkflow_Execute_ContextCancelled(t *testing.T) {
	w := NewWorkflow("test")

	step1 := NewMockStep("step1", false)
	step2 := NewMockStep("step2", false)

	w.AddStep(step1)
	w.AddStep(step2)

	ctx := NewContext()
	ctx.Cancel() // Cancel before execution

	result := w.Execute(ctx)

	assert.Equal(t, WorkflowStatusCancelled, result.Status)
}

func TestBaseWorkflow_Execute_EmptyWorkflow(t *testing.T) {
	w := NewWorkflow("empty")

	ctx := NewContext()
	result := w.Execute(ctx)

	assert.Equal(t, WorkflowStatusCompleted, result.Status)
	assert.Empty(t, result.CompletedSteps)
}

func TestBaseWorkflow_Rollback(t *testing.T) {
	w := NewWorkflow("test")

	rollbackOrder := make([]string, 0)
	var mu sync.Mutex

	step1 := NewFuncStep("step1", "Step 1", func(ctx *Context) StepResult {
		return CompleteStep("done")
	}, WithRollbackFunc(func(ctx *Context) error {
		mu.Lock()
		rollbackOrder = append(rollbackOrder, "step1")
		mu.Unlock()
		return nil
	}))

	step2 := NewFuncStep("step2", "Step 2", func(ctx *Context) StepResult {
		return CompleteStep("done")
	}, WithRollbackFunc(func(ctx *Context) error {
		mu.Lock()
		rollbackOrder = append(rollbackOrder, "step2")
		mu.Unlock()
		return nil
	}))

	step3 := NewFuncStep("step3", "Step 3", func(ctx *Context) StepResult {
		return CompleteStep("done")
	}, WithRollbackFunc(func(ctx *Context) error {
		mu.Lock()
		rollbackOrder = append(rollbackOrder, "step3")
		mu.Unlock()
		return nil
	}))

	w.AddStep(step1)
	w.AddStep(step2)
	w.AddStep(step3)

	ctx := NewContext()
	w.Execute(ctx)

	err := w.Rollback(ctx)

	assert.NoError(t, err)
	// Rollback should happen in reverse order
	assert.Equal(t, []string{"step3", "step2", "step1"}, rollbackOrder)
}

func TestBaseWorkflow_Rollback_WithErrors(t *testing.T) {
	w := NewWorkflow("test")

	step1 := NewFuncStep("step1", "Step 1", func(ctx *Context) StepResult {
		return CompleteStep("done")
	}, WithRollbackFunc(func(ctx *Context) error {
		return nil
	}))

	step2 := NewFuncStep("step2", "Step 2", func(ctx *Context) StepResult {
		return CompleteStep("done")
	}, WithRollbackFunc(func(ctx *Context) error {
		return errors.New("rollback failed")
	}))

	w.AddStep(step1)
	w.AddStep(step2)

	ctx := NewContext()
	w.Execute(ctx)

	err := w.Rollback(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "rollback completed with 1 errors")
}

func TestBaseWorkflow_Rollback_SkipsNonRollbackable(t *testing.T) {
	w := NewWorkflow("test")

	step1RolledBack := false
	step2RolledBack := false

	step1 := NewFuncStep("step1", "Step 1", func(ctx *Context) StepResult {
		return CompleteStep("done")
	}, WithRollbackFunc(func(ctx *Context) error {
		step1RolledBack = true
		return nil
	}))

	// Step 2 has no rollback function
	step2 := NewFuncStep("step2", "Step 2", func(ctx *Context) StepResult {
		return CompleteStep("done")
	})

	step3 := NewFuncStep("step3", "Step 3", func(ctx *Context) StepResult {
		return CompleteStep("done")
	}, WithRollbackFunc(func(ctx *Context) error {
		step2RolledBack = true
		return nil
	}))

	w.AddStep(step1)
	w.AddStep(step2)
	w.AddStep(step3)

	ctx := NewContext()
	w.Execute(ctx)

	err := w.Rollback(ctx)

	assert.NoError(t, err)
	assert.True(t, step1RolledBack)
	assert.True(t, step2RolledBack)
	assert.False(t, step2.CanRollback())
}

func TestBaseWorkflow_OnProgress(t *testing.T) {
	w := NewWorkflow("test")

	step1 := NewMockStep("step1", false)
	step2 := NewMockStep("step2", false)

	w.AddStep(step1)
	w.AddStep(step2)

	progressUpdates := make([]StepProgress, 0)
	var mu sync.Mutex

	w.OnProgress(func(p StepProgress) {
		mu.Lock()
		progressUpdates = append(progressUpdates, p)
		mu.Unlock()
	})

	ctx := NewContext()
	w.Execute(ctx)

	mu.Lock()
	defer mu.Unlock()

	// Should have progress updates for starting and completing each step
	assert.True(t, len(progressUpdates) >= 4)

	// Check that step names are in progress
	names := make([]string, 0)
	for _, p := range progressUpdates {
		if p.StepName != "" {
			names = append(names, p.StepName)
		}
	}
	assert.Contains(t, names, "step1")
	assert.Contains(t, names, "step2")
}

func TestBaseWorkflow_Reset(t *testing.T) {
	w := NewWorkflow("test")

	step1 := NewMockStep("step1", false)
	w.AddStep(step1)

	ctx := NewContext()
	w.Execute(ctx)

	assert.NotEmpty(t, w.CompletedSteps())

	w.Cancel()
	assert.True(t, w.IsCancelled())

	w.Reset()

	assert.Empty(t, w.CompletedSteps())
	assert.False(t, w.IsCancelled())
}

func TestBaseWorkflow_ThreadSafety(t *testing.T) {
	w := NewWorkflow("concurrent")

	// Add steps concurrently
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			w.AddStep(NewMockStep("step-"+string(rune('a'+idx)), false))
		}(i)
	}
	wg.Wait()

	assert.Len(t, w.Steps(), 10)

	// Read operations concurrently
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = w.Name()
			_ = w.Steps()
			_ = w.IsCancelled()
		}()
	}
	wg.Wait()
}

func TestBaseWorkflow_CompletedSteps(t *testing.T) {
	w := NewWorkflow("test")

	step1 := NewMockStep("step1", false)
	step2 := NewMockStep("step2", false)
	step3 := NewMockStep("step3", false)
	step3.SetExecuteResult(FailStep("failed", errors.New("error")))

	w.AddStep(step1)
	w.AddStep(step2)
	w.AddStep(step3)

	ctx := NewContext()
	w.Execute(ctx)

	completed := w.CompletedSteps()
	require.Len(t, completed, 2)
	assert.Equal(t, "step1", completed[0].Name())
	assert.Equal(t, "step2", completed[1].Name())
}

func TestBaseWorkflow_Execute_WithRealSteps(t *testing.T) {
	w := NewWorkflow("real-workflow")

	stateKey := "counter"

	step1 := NewFuncStep("increment", "Increment counter", func(ctx *Context) StepResult {
		ctx.SetState(stateKey, ctx.GetStateInt(stateKey)+1)
		return CompleteStep("incremented")
	})

	step2 := NewFuncStep("increment2", "Increment counter again", func(ctx *Context) StepResult {
		ctx.SetState(stateKey, ctx.GetStateInt(stateKey)+1)
		return CompleteStep("incremented again")
	})

	step3 := NewFuncStep("check", "Check counter", func(ctx *Context) StepResult {
		if ctx.GetStateInt(stateKey) != 2 {
			return FailStep("wrong counter value", errors.New("expected 2"))
		}
		return CompleteStep("counter is correct")
	})

	w.AddStep(step1)
	w.AddStep(step2)
	w.AddStep(step3)

	ctx := NewContext()
	result := w.Execute(ctx)

	assert.Equal(t, WorkflowStatusCompleted, result.Status)
	assert.Equal(t, 2, ctx.GetStateInt(stateKey))
}

func TestBaseWorkflow_Execute_RollbackOnFailure(t *testing.T) {
	w := NewWorkflow("rollback-test")

	rollbackOrder := make([]string, 0)
	var mu sync.Mutex

	step1 := NewFuncStep("setup", "Setup", func(ctx *Context) StepResult {
		return CompleteStep("setup done")
	}, WithRollbackFunc(func(ctx *Context) error {
		mu.Lock()
		rollbackOrder = append(rollbackOrder, "setup")
		mu.Unlock()
		return nil
	}))

	step2 := NewFuncStep("install", "Install", func(ctx *Context) StepResult {
		return CompleteStep("install done")
	}, WithRollbackFunc(func(ctx *Context) error {
		mu.Lock()
		rollbackOrder = append(rollbackOrder, "install")
		mu.Unlock()
		return nil
	}))

	step3 := NewFuncStep("configure", "Configure", func(ctx *Context) StepResult {
		return FailStep("configuration failed", errors.New("config error"))
	})

	w.AddStep(step1)
	w.AddStep(step2)
	w.AddStep(step3)

	ctx := NewContext()
	result := w.Execute(ctx)

	assert.Equal(t, WorkflowStatusFailed, result.Status)
	assert.Equal(t, "configure", result.FailedStep)

	// Perform rollback after failure
	err := w.Rollback(ctx)
	assert.NoError(t, err)

	// Verify rollback order (reverse of completion order)
	assert.Equal(t, []string{"install", "setup"}, rollbackOrder)
}
