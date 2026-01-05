package uninstall

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tungetti/igor/internal/install"
)

func TestNewUninstallWorkflow(t *testing.T) {
	w := NewUninstallWorkflow("test-uninstall")

	assert.Equal(t, "test-uninstall", w.Name())
	assert.Empty(t, w.Steps())
	assert.Empty(t, w.CompletedSteps())
	assert.False(t, w.IsCancelled())
}

func TestBaseUninstallWorkflow_AddStep(t *testing.T) {
	w := NewUninstallWorkflow("test")

	step1 := NewMockUninstallStep("step1", false)
	step2 := NewMockUninstallStep("step2", true)

	w.AddStep(step1)
	w.AddStep(step2)

	steps := w.Steps()
	assert.Len(t, steps, 2)
	assert.Equal(t, "step1", steps[0].Name())
	assert.Equal(t, "step2", steps[1].Name())
}

func TestBaseUninstallWorkflow_Execute_Success(t *testing.T) {
	w := NewUninstallWorkflow("test")

	step1 := NewMockUninstallStep("step1", false)
	step2 := NewMockUninstallStep("step2", false)
	step3 := NewMockUninstallStep("step3", false)

	w.AddStep(step1)
	w.AddStep(step2)
	w.AddStep(step3)

	ctx := NewUninstallContext()
	result := w.Execute(ctx)

	assert.Equal(t, UninstallStatusCompleted, result.Status)
	assert.Nil(t, result.Error)
	assert.Empty(t, result.FailedStep)
	assert.Equal(t, []string{"step1", "step2", "step3"}, result.CompletedSteps)
	assert.True(t, result.TotalDuration > 0)

	assert.True(t, step1.executeCalled)
	assert.True(t, step2.executeCalled)
	assert.True(t, step3.executeCalled)
}

func TestBaseUninstallWorkflow_Execute_StepFailure(t *testing.T) {
	w := NewUninstallWorkflow("test")

	step1 := NewMockUninstallStep("step1", false)
	step2 := NewMockUninstallStep("step2", false)
	step2.SetExecuteResult(install.FailStep("step failed", errors.New("test error")))
	step3 := NewMockUninstallStep("step3", false)

	w.AddStep(step1)
	w.AddStep(step2)
	w.AddStep(step3)

	ctx := NewUninstallContext()
	result := w.Execute(ctx)

	assert.Equal(t, UninstallStatusFailed, result.Status)
	assert.Equal(t, "step2", result.FailedStep)
	assert.Error(t, result.Error)
	assert.Equal(t, []string{"step1"}, result.CompletedSteps)

	assert.True(t, step1.executeCalled)
	assert.True(t, step2.executeCalled)
	assert.False(t, step3.executeCalled) // Should not be called after failure
}

func TestBaseUninstallWorkflow_Execute_ValidationFailure(t *testing.T) {
	w := NewUninstallWorkflow("test")

	step1 := NewMockUninstallStep("step1", false)
	step2 := NewMockUninstallStep("step2", false)
	step2.SetValidateError(errors.New("validation failed"))

	w.AddStep(step1)
	w.AddStep(step2)

	ctx := NewUninstallContext()
	result := w.Execute(ctx)

	assert.Equal(t, UninstallStatusFailed, result.Status)
	assert.Equal(t, "step2", result.FailedStep)
	assert.Contains(t, result.Error.Error(), "validation failed")
	assert.Equal(t, []string{"step1"}, result.CompletedSteps)

	assert.True(t, step1.executeCalled)
	assert.True(t, step2.validateCalled)
	assert.False(t, step2.executeCalled) // Should not execute after validation failure
}

func TestBaseUninstallWorkflow_Execute_SkippedStep(t *testing.T) {
	w := NewUninstallWorkflow("test")

	step1 := NewMockUninstallStep("step1", false)
	step2 := NewMockUninstallStep("step2", false)
	step2.SetExecuteResult(install.SkipStep("already removed"))
	step3 := NewMockUninstallStep("step3", false)

	w.AddStep(step1)
	w.AddStep(step2)
	w.AddStep(step3)

	ctx := NewUninstallContext()
	result := w.Execute(ctx)

	assert.Equal(t, UninstallStatusCompleted, result.Status)
	// Skipped steps are not added to completed
	assert.Equal(t, []string{"step1", "step3"}, result.CompletedSteps)

	assert.True(t, step1.executeCalled)
	assert.True(t, step2.executeCalled)
	assert.True(t, step3.executeCalled)
}

func TestBaseUninstallWorkflow_Execute_Cancel(t *testing.T) {
	w := NewUninstallWorkflow("test")

	step1 := NewMockUninstallStep("step1", false)
	step2 := install.NewFuncStep("step2", "Slow step", func(ctx *install.Context) install.StepResult {
		time.Sleep(100 * time.Millisecond)
		return install.CompleteStep("done")
	})
	step3 := NewMockUninstallStep("step3", false)

	w.AddStep(step1)
	w.AddStep(step2)
	w.AddStep(step3)

	ctx := NewUninstallContext()

	// Cancel immediately after starting
	go func() {
		time.Sleep(10 * time.Millisecond)
		w.Cancel()
	}()

	result := w.Execute(ctx)

	// Should be cancelled before completing all steps
	assert.Equal(t, UninstallStatusCancelled, result.Status)
	assert.True(t, w.IsCancelled())
}

func TestBaseUninstallWorkflow_Execute_ContextCancelled(t *testing.T) {
	w := NewUninstallWorkflow("test")

	step1 := NewMockUninstallStep("step1", false)
	step2 := NewMockUninstallStep("step2", false)

	w.AddStep(step1)
	w.AddStep(step2)

	ctx := NewUninstallContext()
	ctx.Cancel() // Cancel before execution

	result := w.Execute(ctx)

	assert.Equal(t, UninstallStatusCancelled, result.Status)
}

func TestBaseUninstallWorkflow_Execute_EmptyWorkflow(t *testing.T) {
	w := NewUninstallWorkflow("empty")

	ctx := NewUninstallContext()
	result := w.Execute(ctx)

	assert.Equal(t, UninstallStatusCompleted, result.Status)
	assert.Empty(t, result.CompletedSteps)
}

func TestBaseUninstallWorkflow_OnProgress(t *testing.T) {
	w := NewUninstallWorkflow("test")

	step1 := NewMockUninstallStep("step1", false)
	step2 := NewMockUninstallStep("step2", false)

	w.AddStep(step1)
	w.AddStep(step2)

	progressUpdates := make([]install.StepProgress, 0)
	var mu sync.Mutex

	w.OnProgress(func(p install.StepProgress) {
		mu.Lock()
		progressUpdates = append(progressUpdates, p)
		mu.Unlock()
	})

	ctx := NewUninstallContext()
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

func TestBaseUninstallWorkflow_Reset(t *testing.T) {
	w := NewUninstallWorkflow("test")

	step1 := NewMockUninstallStep("step1", false)
	w.AddStep(step1)

	ctx := NewUninstallContext()
	w.Execute(ctx)

	assert.NotEmpty(t, w.CompletedSteps())

	w.Cancel()
	assert.True(t, w.IsCancelled())

	w.Reset()

	assert.Empty(t, w.CompletedSteps())
	assert.False(t, w.IsCancelled())
}

func TestBaseUninstallWorkflow_ThreadSafety(t *testing.T) {
	w := NewUninstallWorkflow("concurrent")

	// Add steps concurrently
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			w.AddStep(NewMockUninstallStep("step-"+string(rune('a'+idx)), false))
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

func TestBaseUninstallWorkflow_CompletedSteps(t *testing.T) {
	w := NewUninstallWorkflow("test")

	step1 := NewMockUninstallStep("step1", false)
	step2 := NewMockUninstallStep("step2", false)
	step3 := NewMockUninstallStep("step3", false)
	step3.SetExecuteResult(install.FailStep("failed", errors.New("error")))

	w.AddStep(step1)
	w.AddStep(step2)
	w.AddStep(step3)

	ctx := NewUninstallContext()
	w.Execute(ctx)

	completed := w.CompletedSteps()
	require.Len(t, completed, 2)
	assert.Equal(t, "step1", completed[0].Name())
	assert.Equal(t, "step2", completed[1].Name())
}

func TestBaseUninstallWorkflow_Execute_WithRealSteps(t *testing.T) {
	w := NewUninstallWorkflow("real-workflow")

	stateKey := "counter"

	step1 := install.NewFuncStep("increment", "Increment counter", func(ctx *install.Context) install.StepResult {
		ctx.SetState(stateKey, ctx.GetStateInt(stateKey)+1)
		return install.CompleteStep("incremented")
	})

	step2 := install.NewFuncStep("increment2", "Increment counter again", func(ctx *install.Context) install.StepResult {
		ctx.SetState(stateKey, ctx.GetStateInt(stateKey)+1)
		return install.CompleteStep("incremented again")
	})

	step3 := install.NewFuncStep("check", "Check counter", func(ctx *install.Context) install.StepResult {
		if ctx.GetStateInt(stateKey) != 2 {
			return install.FailStep("wrong counter value", errors.New("expected 2"))
		}
		return install.CompleteStep("counter is correct")
	})

	w.AddStep(step1)
	w.AddStep(step2)
	w.AddStep(step3)

	ctx := NewUninstallContext()
	result := w.Execute(ctx)

	assert.Equal(t, UninstallStatusCompleted, result.Status)
}

func TestBaseUninstallWorkflow_PartialResult(t *testing.T) {
	w := NewUninstallWorkflow("partial-test")

	step1 := install.NewFuncStep("remove-some", "Remove some packages", func(ctx *install.Context) install.StepResult {
		// Simulate partial removal - set state for removed and failed packages
		ctx.SetState(StateRemovedPackages, []string{"nvidia-driver"})
		return install.CompleteStep("partial removal")
	})

	w.AddStep(step1)

	ctx := NewUninstallContext()
	result := w.Execute(ctx)

	// Verify the result captures removed packages
	assert.Equal(t, UninstallStatusCompleted, result.Status)
	assert.Contains(t, result.RemovedPackages, "nvidia-driver")
}

func TestBaseUninstallWorkflow_NouveauRestored(t *testing.T) {
	w := NewUninstallWorkflow("nouveau-test")

	step1 := install.NewFuncStep("restore-nouveau", "Restore nouveau driver", func(ctx *install.Context) install.StepResult {
		ctx.SetState(StateNouveauRestored, true)
		return install.CompleteStep("nouveau restored")
	})

	w.AddStep(step1)

	ctx := NewUninstallContext()
	result := w.Execute(ctx)

	assert.True(t, result.NouveauRestored)
}

func TestBaseUninstallWorkflow_Execute_NilContext(t *testing.T) {
	w := NewUninstallWorkflow("test")

	step1 := NewMockUninstallStep("step1", false)
	w.AddStep(step1)

	// Execute with nil context should not panic
	result := w.Execute(nil)

	// With nil context, we still try to execute but may fail or complete
	// The important thing is it doesn't panic
	assert.True(t, result.Status == UninstallStatusCompleted || result.Status == UninstallStatusFailed)
}

func TestUninstallWorkflowInterface(t *testing.T) {
	// Verify that BaseUninstallWorkflow implements UninstallWorkflow
	var _ UninstallWorkflow = (*BaseUninstallWorkflow)(nil)
}

// MockUninstallStep implements install.Step interface for testing
type MockUninstallStep struct {
	install.BaseStep
	executeResult  install.StepResult
	rollbackError  error
	validateError  error
	executeCalled  bool
	rollbackCalled bool
	validateCalled bool
}

func NewMockUninstallStep(name string, canRollback bool) *MockUninstallStep {
	return &MockUninstallStep{
		BaseStep:      install.NewBaseStep(name, "Mock step: "+name, canRollback),
		executeResult: install.CompleteStep("mock executed"),
	}
}

func (m *MockUninstallStep) Execute(ctx *install.Context) install.StepResult {
	m.executeCalled = true
	return m.executeResult
}

func (m *MockUninstallStep) Rollback(ctx *install.Context) error {
	m.rollbackCalled = true
	return m.rollbackError
}

func (m *MockUninstallStep) Validate(ctx *install.Context) error {
	m.validateCalled = true
	return m.validateError
}

func (m *MockUninstallStep) SetExecuteResult(result install.StepResult) {
	m.executeResult = result
}

func (m *MockUninstallStep) SetRollbackError(err error) {
	m.rollbackError = err
}

func (m *MockUninstallStep) SetValidateError(err error) {
	m.validateError = err
}
