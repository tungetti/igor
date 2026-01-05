package install

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newMockStep creates a mock step with the given name and status.
func newMockStep(name string, status StepStatus) Step {
	return NewFuncStep(name, "Mock step: "+name, func(ctx *Context) StepResult {
		return NewStepResult(status, name+" completed")
	})
}

// newMockStepWithRollback creates a mock step with rollback support.
func newMockStepWithRollback(name string, status StepStatus, rollbackFn func(ctx *Context) error) Step {
	return NewFuncStep(name, "Mock step: "+name, func(ctx *Context) StepResult {
		return NewStepResult(status, name+" completed")
	}, WithRollbackFunc(rollbackFn))
}

// newMockWorkflow creates a mock workflow with the given steps.
func newMockWorkflow(steps ...Step) Workflow {
	w := NewWorkflow("test-workflow")
	for _, s := range steps {
		w.AddStep(s)
	}
	return w
}

func TestNewOrchestrator(t *testing.T) {
	w := newMockWorkflow()
	o := NewOrchestrator(w)

	assert.NotNil(t, o)
	assert.Equal(t, w, o.Workflow())
	assert.False(t, o.autoRollback)
	assert.True(t, o.stopOnFirstError)
	assert.False(t, o.dryRun)
	assert.Nil(t, o.preExecuteHook)
	assert.Nil(t, o.postExecuteHook)
	assert.Nil(t, o.preStepHook)
	assert.Nil(t, o.postStepHook)
	assert.Nil(t, o.progressCallback)
	assert.Empty(t, o.GetExecutionLog())
}

func TestOrchestratorOptions(t *testing.T) {
	t.Run("WithAutoRollback", func(t *testing.T) {
		o := NewOrchestrator(newMockWorkflow(), WithAutoRollback(true))
		assert.True(t, o.autoRollback)
	})

	t.Run("WithStopOnFirstError", func(t *testing.T) {
		o := NewOrchestrator(newMockWorkflow(), WithStopOnFirstError(false))
		assert.False(t, o.stopOnFirstError)
	})

	t.Run("WithOrchestratorDryRun", func(t *testing.T) {
		o := NewOrchestrator(newMockWorkflow(), WithOrchestratorDryRun(true))
		assert.True(t, o.dryRun)
	})

	t.Run("WithPreExecuteHook", func(t *testing.T) {
		hook := func(ctx *Context, w Workflow) error { return nil }
		o := NewOrchestrator(newMockWorkflow(), WithPreExecuteHook(hook))
		assert.NotNil(t, o.preExecuteHook)
	})

	t.Run("WithPostExecuteHook", func(t *testing.T) {
		hook := func(ctx *Context, w Workflow) error { return nil }
		o := NewOrchestrator(newMockWorkflow(), WithPostExecuteHook(hook))
		assert.NotNil(t, o.postExecuteHook)
	})

	t.Run("WithPreStepHook", func(t *testing.T) {
		hook := func(ctx *Context, s Step, r *StepResult) error { return nil }
		o := NewOrchestrator(newMockWorkflow(), WithPreStepHook(hook))
		assert.NotNil(t, o.preStepHook)
	})

	t.Run("WithPostStepHook", func(t *testing.T) {
		hook := func(ctx *Context, s Step, r *StepResult) error { return nil }
		o := NewOrchestrator(newMockWorkflow(), WithPostStepHook(hook))
		assert.NotNil(t, o.postStepHook)
	})

	t.Run("WithOrchestratorProgress", func(t *testing.T) {
		callback := func(p StepProgress) {}
		o := NewOrchestrator(newMockWorkflow(), WithOrchestratorProgress(callback))
		assert.NotNil(t, o.progressCallback)
	})

	t.Run("multiple options", func(t *testing.T) {
		o := NewOrchestrator(
			newMockWorkflow(),
			WithAutoRollback(true),
			WithStopOnFirstError(false),
			WithOrchestratorDryRun(true),
		)
		assert.True(t, o.autoRollback)
		assert.False(t, o.stopOnFirstError)
		assert.True(t, o.dryRun)
	})
}

func TestOrchestrator_Execute_Success(t *testing.T) {
	step1 := newMockStep("step1", StepStatusCompleted)
	step2 := newMockStep("step2", StepStatusCompleted)
	step3 := newMockStep("step3", StepStatusCompleted)

	w := newMockWorkflow(step1, step2, step3)
	o := NewOrchestrator(w)

	ctx := NewContext()
	report := o.Execute(ctx)

	assert.Equal(t, WorkflowStatusCompleted, report.Status)
	assert.Equal(t, "test-workflow", report.WorkflowName)
	assert.True(t, report.TotalDuration > 0)
	assert.Nil(t, report.Error)
	assert.False(t, report.RollbackPerformed)
	assert.True(t, report.RollbackSuccess)

	// Check execution log
	log := o.GetExecutionLog()
	assert.True(t, len(log) >= 2) // At least workflow started and completed

	// Find start and end events
	hasStart := false
	hasEnd := false
	for _, entry := range log {
		if entry.EventType == EventWorkflowStarted {
			hasStart = true
		}
		if entry.EventType == EventWorkflowCompleted {
			hasEnd = true
		}
	}
	assert.True(t, hasStart)
	assert.True(t, hasEnd)
}

func TestOrchestrator_Execute_StepFailure(t *testing.T) {
	step1 := newMockStep("step1", StepStatusCompleted)
	step2 := NewFuncStep("step2", "Failing step", func(ctx *Context) StepResult {
		return FailStep("step failed", errors.New("test error"))
	})
	step3 := newMockStep("step3", StepStatusCompleted)

	w := newMockWorkflow(step1, step2, step3)
	o := NewOrchestrator(w)

	ctx := NewContext()
	report := o.Execute(ctx)

	assert.Equal(t, WorkflowStatusFailed, report.Status)
	assert.Error(t, report.Error)
	assert.False(t, report.RollbackPerformed)

	// step3 should not have executed due to stopOnFirstError
	log := o.GetExecutionLog()
	step3Started := false
	for _, entry := range log {
		if entry.StepName == "step3" && entry.EventType == EventStepStarted {
			step3Started = true
		}
	}
	assert.False(t, step3Started)
}

func TestOrchestrator_Execute_AutoRollback(t *testing.T) {
	rollbackCalled := false
	step1 := newMockStepWithRollback("step1", StepStatusCompleted, func(ctx *Context) error {
		rollbackCalled = true
		return nil
	})
	step2 := NewFuncStep("step2", "Failing step", func(ctx *Context) StepResult {
		return FailStep("step failed", errors.New("test error"))
	})

	w := newMockWorkflow(step1, step2)
	o := NewOrchestrator(w, WithAutoRollback(true))

	ctx := NewContext()
	report := o.Execute(ctx)

	assert.Equal(t, WorkflowStatusFailed, report.Status)
	assert.True(t, report.RollbackPerformed)
	assert.True(t, rollbackCalled)
}

func TestOrchestrator_Execute_DryRun(t *testing.T) {
	var wasDryRun bool
	step := NewFuncStep("check-dry-run", "Check dry run", func(ctx *Context) StepResult {
		wasDryRun = ctx.DryRun
		return CompleteStep("checked")
	})

	w := newMockWorkflow(step)
	o := NewOrchestrator(w, WithOrchestratorDryRun(true))

	ctx := NewContext()
	report := o.Execute(ctx)

	assert.Equal(t, WorkflowStatusCompleted, report.Status)
	assert.True(t, wasDryRun)
}

func TestOrchestrator_Execute_Cancelled(t *testing.T) {
	step1 := newMockStep("step1", StepStatusCompleted)
	step2 := NewFuncStep("step2", "Slow step", func(ctx *Context) StepResult {
		time.Sleep(100 * time.Millisecond)
		return CompleteStep("done")
	})

	w := newMockWorkflow(step1, step2)
	o := NewOrchestrator(w)

	ctx := NewContext()

	// Cancel before execution
	ctx.Cancel()

	report := o.Execute(ctx)

	assert.Equal(t, WorkflowStatusCancelled, report.Status)
}

func TestOrchestrator_Execute_PreExecuteHook(t *testing.T) {
	hookCalled := false
	var hookWorkflow Workflow

	hook := func(ctx *Context, w Workflow) error {
		hookCalled = true
		hookWorkflow = w
		return nil
	}

	w := newMockWorkflow(newMockStep("step1", StepStatusCompleted))
	o := NewOrchestrator(w, WithPreExecuteHook(hook))

	ctx := NewContext()
	report := o.Execute(ctx)

	assert.Equal(t, WorkflowStatusCompleted, report.Status)
	assert.True(t, hookCalled)
	assert.Equal(t, w, hookWorkflow)
}

func TestOrchestrator_Execute_PostExecuteHook(t *testing.T) {
	hookCalled := false

	hook := func(ctx *Context, w Workflow) error {
		hookCalled = true
		return nil
	}

	w := newMockWorkflow(newMockStep("step1", StepStatusCompleted))
	o := NewOrchestrator(w, WithPostExecuteHook(hook))

	ctx := NewContext()
	report := o.Execute(ctx)

	assert.Equal(t, WorkflowStatusCompleted, report.Status)
	assert.True(t, hookCalled)
}

func TestOrchestrator_Execute_PreStepHook(t *testing.T) {
	stepNames := make([]string, 0)
	var mu sync.Mutex

	hook := func(ctx *Context, s Step, r *StepResult) error {
		mu.Lock()
		stepNames = append(stepNames, s.Name())
		mu.Unlock()
		return nil
	}

	step1 := newMockStep("step1", StepStatusCompleted)
	step2 := newMockStep("step2", StepStatusCompleted)
	step3 := newMockStep("step3", StepStatusCompleted)

	w := newMockWorkflow(step1, step2, step3)
	o := NewOrchestrator(w, WithPreStepHook(hook))

	ctx := NewContext()
	report := o.Execute(ctx)

	assert.Equal(t, WorkflowStatusCompleted, report.Status)

	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, []string{"step1", "step2", "step3"}, stepNames)
}

func TestOrchestrator_Execute_PostStepHook(t *testing.T) {
	stepResults := make(map[string]StepStatus)
	var mu sync.Mutex

	hook := func(ctx *Context, s Step, r *StepResult) error {
		mu.Lock()
		if r != nil {
			stepResults[s.Name()] = r.Status
		}
		mu.Unlock()
		return nil
	}

	step1 := newMockStep("step1", StepStatusCompleted)
	step2 := NewFuncStep("step2", "Skip step", func(ctx *Context) StepResult {
		return SkipStep("already done")
	})

	w := newMockWorkflow(step1, step2)
	o := NewOrchestrator(w, WithPostStepHook(hook))

	ctx := NewContext()
	report := o.Execute(ctx)

	assert.Equal(t, WorkflowStatusCompleted, report.Status)

	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, StepStatusCompleted, stepResults["step1"])
	assert.Equal(t, StepStatusSkipped, stepResults["step2"])
}

func TestOrchestrator_Execute_HookError(t *testing.T) {
	t.Run("pre-execute hook error", func(t *testing.T) {
		hookErr := errors.New("pre-execute hook failed")
		hook := func(ctx *Context, w Workflow) error {
			return hookErr
		}

		w := newMockWorkflow(newMockStep("step1", StepStatusCompleted))
		o := NewOrchestrator(w, WithPreExecuteHook(hook))

		ctx := NewContext()
		report := o.Execute(ctx)

		assert.Equal(t, WorkflowStatusFailed, report.Status)
		assert.Equal(t, hookErr, report.Error)
	})

	t.Run("post-execute hook error", func(t *testing.T) {
		hookErr := errors.New("post-execute hook failed")
		hook := func(ctx *Context, w Workflow) error {
			return hookErr
		}

		w := newMockWorkflow(newMockStep("step1", StepStatusCompleted))
		o := NewOrchestrator(w, WithPostExecuteHook(hook))

		ctx := NewContext()
		report := o.Execute(ctx)

		assert.Equal(t, WorkflowStatusFailed, report.Status)
		assert.Equal(t, hookErr, report.Error)
	})

	t.Run("pre-step hook error", func(t *testing.T) {
		hookErr := errors.New("pre-step hook failed")
		hook := func(ctx *Context, s Step, r *StepResult) error {
			if s.Name() == "step2" {
				return hookErr
			}
			return nil
		}

		step1 := newMockStep("step1", StepStatusCompleted)
		step2 := newMockStep("step2", StepStatusCompleted)

		w := newMockWorkflow(step1, step2)
		o := NewOrchestrator(w, WithPreStepHook(hook))

		ctx := NewContext()
		report := o.Execute(ctx)

		assert.Equal(t, WorkflowStatusFailed, report.Status)
		assert.Equal(t, hookErr, report.Error)
	})

	t.Run("post-step hook error", func(t *testing.T) {
		hookErr := errors.New("post-step hook failed")
		hook := func(ctx *Context, s Step, r *StepResult) error {
			if s.Name() == "step1" {
				return hookErr
			}
			return nil
		}

		step1 := newMockStep("step1", StepStatusCompleted)
		step2 := newMockStep("step2", StepStatusCompleted)

		w := newMockWorkflow(step1, step2)
		o := NewOrchestrator(w, WithPostStepHook(hook))

		ctx := NewContext()
		report := o.Execute(ctx)

		assert.Equal(t, WorkflowStatusFailed, report.Status)
		assert.Equal(t, hookErr, report.Error)
	})
}

func TestOrchestrator_Execute_ProgressCallback(t *testing.T) {
	progressUpdates := make([]StepProgress, 0)
	var mu sync.Mutex

	callback := func(p StepProgress) {
		mu.Lock()
		progressUpdates = append(progressUpdates, p)
		mu.Unlock()
	}

	step1 := newMockStep("step1", StepStatusCompleted)
	step2 := newMockStep("step2", StepStatusCompleted)

	w := newMockWorkflow(step1, step2)
	o := NewOrchestrator(w, WithOrchestratorProgress(callback))

	ctx := NewContext()
	report := o.Execute(ctx)

	assert.Equal(t, WorkflowStatusCompleted, report.Status)

	mu.Lock()
	defer mu.Unlock()
	assert.True(t, len(progressUpdates) >= 2)
}

func TestOrchestrator_ExecuteWithRollback(t *testing.T) {
	rollbackCalled := false
	step1 := newMockStepWithRollback("step1", StepStatusCompleted, func(ctx *Context) error {
		rollbackCalled = true
		return nil
	})
	step2 := NewFuncStep("step2", "Failing step", func(ctx *Context) StepResult {
		return FailStep("step failed", errors.New("test error"))
	})

	w := newMockWorkflow(step1, step2)
	o := NewOrchestrator(w)

	ctx := NewContext()
	report := o.ExecuteWithRollback(ctx)

	assert.Equal(t, WorkflowStatusFailed, report.Status)
	assert.True(t, report.RollbackPerformed)
	assert.True(t, rollbackCalled)
}

func TestOrchestrator_GetExecutionLog(t *testing.T) {
	step := newMockStep("step1", StepStatusCompleted)
	w := newMockWorkflow(step)
	o := NewOrchestrator(w)

	// Before execution, log should be empty
	assert.Empty(t, o.GetExecutionLog())

	ctx := NewContext()
	o.Execute(ctx)

	// After execution, log should have entries
	log := o.GetExecutionLog()
	assert.NotEmpty(t, log)

	// Verify log is a copy (modifying it doesn't affect internal state)
	log[0].Message = "modified"
	originalLog := o.GetExecutionLog()
	assert.NotEqual(t, "modified", originalLog[0].Message)
}

func TestOrchestrator_Reset(t *testing.T) {
	step := newMockStep("step1", StepStatusCompleted)
	w := newMockWorkflow(step)
	o := NewOrchestrator(w)

	ctx := NewContext()
	o.Execute(ctx)

	assert.NotEmpty(t, o.GetExecutionLog())

	o.Reset()

	assert.Empty(t, o.GetExecutionLog())
}

func TestOrchestrator_SetWorkflow(t *testing.T) {
	w1 := newMockWorkflow(newMockStep("step1", StepStatusCompleted))
	w2 := newMockWorkflow(newMockStep("step2", StepStatusCompleted))

	o := NewOrchestrator(w1)
	assert.Equal(t, w1, o.Workflow())

	o.SetWorkflow(w2)
	assert.Equal(t, w2, o.Workflow())
}

func TestExecutionReport_Generation(t *testing.T) {
	t.Run("successful workflow", func(t *testing.T) {
		step1 := newMockStep("step1", StepStatusCompleted)
		step2 := NewFuncStep("step2", "Skip step", func(ctx *Context) StepResult {
			return SkipStep("already done")
		})

		w := newMockWorkflow(step1, step2)
		o := NewOrchestrator(w, WithPreStepHook(func(ctx *Context, s Step, r *StepResult) error {
			return nil
		}))

		ctx := NewContext()
		report := o.Execute(ctx)

		assert.Equal(t, "test-workflow", report.WorkflowName)
		assert.Equal(t, WorkflowStatusCompleted, report.Status)
		assert.True(t, report.TotalDuration > 0)
		assert.True(t, report.StepsExecuted >= 2)
		assert.True(t, report.StepsCompleted >= 1)
		assert.True(t, report.StepsSkipped >= 1)
		assert.Equal(t, 0, report.StepsFailed)
		assert.False(t, report.RollbackPerformed)
		assert.True(t, report.RollbackSuccess)
		assert.Nil(t, report.Error)
		assert.NotEmpty(t, report.ExecutionLog)
	})

	t.Run("failed workflow with rollback", func(t *testing.T) {
		rollbackCalled := false
		step1 := newMockStepWithRollback("step1", StepStatusCompleted, func(ctx *Context) error {
			rollbackCalled = true
			return nil
		})
		step2 := NewFuncStep("step2", "Failing step", func(ctx *Context) StepResult {
			return FailStep("failed", errors.New("error"))
		})

		w := newMockWorkflow(step1, step2)
		// Use workflow directly without step hooks to ensure completed steps are tracked
		o := NewOrchestrator(w, WithAutoRollback(true))

		ctx := NewContext()
		report := o.Execute(ctx)

		assert.Equal(t, WorkflowStatusFailed, report.Status)
		assert.True(t, report.RollbackPerformed)
		assert.True(t, report.RollbackSuccess)
		assert.True(t, rollbackCalled)
		assert.Error(t, report.Error)
	})

	t.Run("failed rollback", func(t *testing.T) {
		rollbackErr := errors.New("rollback error")
		step1 := newMockStepWithRollback("step1", StepStatusCompleted, func(ctx *Context) error {
			return rollbackErr
		})
		step2 := NewFuncStep("step2", "Failing step", func(ctx *Context) StepResult {
			return FailStep("failed", errors.New("error"))
		})

		w := newMockWorkflow(step1, step2)
		// Use workflow directly without step hooks to ensure completed steps are tracked
		o := NewOrchestrator(w, WithAutoRollback(true))

		ctx := NewContext()
		report := o.Execute(ctx)

		assert.Equal(t, WorkflowStatusFailed, report.Status)
		assert.True(t, report.RollbackPerformed)
		assert.False(t, report.RollbackSuccess)
	})
}

func TestExecutionEventType_String(t *testing.T) {
	testCases := []struct {
		eventType ExecutionEventType
		expected  string
	}{
		{EventWorkflowStarted, "workflow_started"},
		{EventWorkflowCompleted, "workflow_completed"},
		{EventWorkflowFailed, "workflow_failed"},
		{EventWorkflowCancelled, "workflow_cancelled"},
		{EventWorkflowRollbackStarted, "workflow_rollback_started"},
		{EventWorkflowRollbackCompleted, "workflow_rollback_completed"},
		{EventStepStarted, "step_started"},
		{EventStepCompleted, "step_completed"},
		{EventStepSkipped, "step_skipped"},
		{EventStepFailed, "step_failed"},
		{EventStepRollbackStarted, "step_rollback_started"},
		{EventStepRollbackCompleted, "step_rollback_completed"},
		{ExecutionEventType(100), "unknown"},
	}

	for _, tc := range testCases {
		t.Run(tc.expected, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.eventType.String())
		})
	}
}

func TestOrchestrator_ConcurrentAccess(t *testing.T) {
	step := newMockStep("step1", StepStatusCompleted)
	w := newMockWorkflow(step)
	o := NewOrchestrator(w)

	var wg sync.WaitGroup

	// Concurrent reads
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = o.Workflow()
			_ = o.GetExecutionLog()
		}()
	}

	// Concurrent reset
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			o.Reset()
		}()
	}

	// Concurrent SetWorkflow
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			newW := newMockWorkflow(newMockStep("new-step", StepStatusCompleted))
			o.SetWorkflow(newW)
		}()
	}

	wg.Wait()
}

func TestOrchestrator_Execute_EmptyWorkflow(t *testing.T) {
	w := newMockWorkflow()
	o := NewOrchestrator(w)

	ctx := NewContext()
	report := o.Execute(ctx)

	assert.Equal(t, WorkflowStatusCompleted, report.Status)
	assert.Equal(t, 0, report.StepsExecuted)
}

func TestOrchestrator_Execute_NilContext(t *testing.T) {
	step := newMockStep("step1", StepStatusCompleted)
	w := newMockWorkflow(step)
	o := NewOrchestrator(w)

	// Should not panic with nil context
	report := o.Execute(nil)

	assert.Equal(t, WorkflowStatusCompleted, report.Status)
}

func TestOrchestrator_Execute_StepWithHooks_SkippedStep(t *testing.T) {
	postStepCalled := false
	hook := func(ctx *Context, s Step, r *StepResult) error {
		postStepCalled = true
		return nil
	}

	step := NewFuncStep("skip-step", "Skipped step", func(ctx *Context) StepResult {
		return SkipStep("not needed")
	})

	w := newMockWorkflow(step)
	o := NewOrchestrator(w, WithPostStepHook(hook))

	ctx := NewContext()
	report := o.Execute(ctx)

	assert.Equal(t, WorkflowStatusCompleted, report.Status)
	assert.True(t, postStepCalled)
	assert.True(t, report.StepsSkipped >= 1)
}

func TestOrchestrator_Execute_StopOnFirstError_Disabled(t *testing.T) {
	step1 := newMockStep("step1", StepStatusCompleted)
	step2 := NewFuncStep("step2", "Failing step", func(ctx *Context) StepResult {
		return FailStep("failed", errors.New("error"))
	})
	step3 := newMockStep("step3", StepStatusCompleted)

	w := newMockWorkflow(step1, step2, step3)
	o := NewOrchestrator(w,
		WithStopOnFirstError(false),
		WithPreStepHook(func(ctx *Context, s Step, r *StepResult) error {
			return nil
		}),
	)

	ctx := NewContext()
	report := o.Execute(ctx)

	// Should still fail, but step3 should have been executed
	assert.Equal(t, WorkflowStatusFailed, report.Status)

	// Verify step3 was executed
	step3Executed := false
	for _, entry := range o.GetExecutionLog() {
		if entry.StepName == "step3" && entry.EventType == EventStepStarted {
			step3Executed = true
		}
	}
	assert.True(t, step3Executed)
}

func TestOrchestrator_Execute_WithProgressCallback_IntegrationWithStepHooks(t *testing.T) {
	progressUpdates := make([]StepProgress, 0)
	stepHookCalls := make([]string, 0)
	var mu sync.Mutex

	progressCallback := func(p StepProgress) {
		mu.Lock()
		progressUpdates = append(progressUpdates, p)
		mu.Unlock()
	}

	preStepHook := func(ctx *Context, s Step, r *StepResult) error {
		mu.Lock()
		stepHookCalls = append(stepHookCalls, "pre-"+s.Name())
		mu.Unlock()
		return nil
	}

	postStepHook := func(ctx *Context, s Step, r *StepResult) error {
		mu.Lock()
		stepHookCalls = append(stepHookCalls, "post-"+s.Name())
		mu.Unlock()
		return nil
	}

	step1 := newMockStep("step1", StepStatusCompleted)
	step2 := newMockStep("step2", StepStatusCompleted)

	w := newMockWorkflow(step1, step2)
	o := NewOrchestrator(w,
		WithOrchestratorProgress(progressCallback),
		WithPreStepHook(preStepHook),
		WithPostStepHook(postStepHook),
	)

	ctx := NewContext()
	report := o.Execute(ctx)

	assert.Equal(t, WorkflowStatusCompleted, report.Status)

	mu.Lock()
	defer mu.Unlock()

	// Verify step hooks were called in order
	require.Len(t, stepHookCalls, 4)
	assert.Equal(t, "pre-step1", stepHookCalls[0])
	assert.Equal(t, "post-step1", stepHookCalls[1])
	assert.Equal(t, "pre-step2", stepHookCalls[2])
	assert.Equal(t, "post-step2", stepHookCalls[3])

	// Verify progress was reported
	assert.True(t, len(progressUpdates) >= 2)
}

func TestOrchestrator_Execute_PostExecuteHook_NotOverrideFailure(t *testing.T) {
	// When workflow fails and post-execute hook also fails,
	// the original error should be preserved
	workflowErr := errors.New("workflow error")
	hookErr := errors.New("hook error")

	step := NewFuncStep("fail", "Failing step", func(ctx *Context) StepResult {
		return FailStep("failed", workflowErr)
	})

	hook := func(ctx *Context, w Workflow) error {
		return hookErr
	}

	w := newMockWorkflow(step)
	o := NewOrchestrator(w, WithPostExecuteHook(hook))

	ctx := NewContext()
	report := o.Execute(ctx)

	assert.Equal(t, WorkflowStatusFailed, report.Status)
	// Original workflow error should be preserved (not the hook error)
	assert.Equal(t, workflowErr, report.Error)
}

func TestOrchestrator_Execute_WithValidation(t *testing.T) {
	validationErr := errors.New("validation failed")
	step := NewFuncStep("validate", "Step with validation", func(ctx *Context) StepResult {
		return CompleteStep("done")
	}, WithValidateFunc(func(ctx *Context) error {
		return validationErr
	}))

	w := newMockWorkflow(step)
	o := NewOrchestrator(w, WithPreStepHook(func(ctx *Context, s Step, r *StepResult) error {
		return nil
	}))

	ctx := NewContext()
	report := o.Execute(ctx)

	assert.Equal(t, WorkflowStatusFailed, report.Status)
	assert.Equal(t, validationErr, report.Error)
}

func TestOrchestrator_Execute_Timestamps(t *testing.T) {
	step := newMockStep("step1", StepStatusCompleted)
	w := newMockWorkflow(step)
	o := NewOrchestrator(w)

	beforeExecution := time.Now()
	ctx := NewContext()
	report := o.Execute(ctx)
	afterExecution := time.Now()

	assert.True(t, report.StartTime.After(beforeExecution) || report.StartTime.Equal(beforeExecution))
	assert.True(t, report.EndTime.Before(afterExecution) || report.EndTime.Equal(afterExecution))
	assert.True(t, report.EndTime.After(report.StartTime) || report.EndTime.Equal(report.StartTime))
}

func TestOrchestrator_Execute_ClearsLogOnEachExecution(t *testing.T) {
	step := newMockStep("step1", StepStatusCompleted)
	w := newMockWorkflow(step)
	o := NewOrchestrator(w)

	ctx := NewContext()

	// First execution
	o.Execute(ctx)
	firstLogLen := len(o.GetExecutionLog())

	// Second execution
	o.Execute(ctx)
	secondLogLen := len(o.GetExecutionLog())

	// Log should be cleared and rebuilt on each execution
	assert.Equal(t, firstLogLen, secondLogLen)
}

func TestOrchestrator_Execute_NilWorkflow(t *testing.T) {
	o := NewOrchestrator(nil, WithPreStepHook(func(ctx *Context, s Step, r *StepResult) error {
		return nil
	}))

	ctx := NewContext()
	report := o.Execute(ctx)

	assert.Equal(t, WorkflowStatusFailed, report.Status)
	assert.Empty(t, report.WorkflowName)
}

func TestOrchestrator_Execute_UnexpectedStepStatus(t *testing.T) {
	// Create a step that returns an unexpected status
	unexpectedStep := NewFuncStep("unexpected", "Unexpected status", func(ctx *Context) StepResult {
		return NewStepResult(StepStatusRunning, "still running") // Running is not a terminal status
	})

	w := newMockWorkflow(unexpectedStep)
	o := NewOrchestrator(w, WithPreStepHook(func(ctx *Context, s Step, r *StepResult) error {
		return nil
	}))

	ctx := NewContext()
	report := o.Execute(ctx)

	assert.Equal(t, WorkflowStatusFailed, report.Status)
}

func TestOrchestrator_Execute_CancelledDuringExecution(t *testing.T) {
	step1 := NewFuncStep("step1", "Step 1", func(ctx *Context) StepResult {
		return CompleteStep("done")
	})
	step2 := NewFuncStep("step2", "Step 2 (cancels)", func(ctx *Context) StepResult {
		ctx.Cancel() // Cancel during step execution
		return CompleteStep("done")
	})
	step3 := newMockStep("step3", StepStatusCompleted)

	w := newMockWorkflow(step1, step2, step3)
	o := NewOrchestrator(w, WithPreStepHook(func(ctx *Context, s Step, r *StepResult) error {
		return nil
	}))

	ctx := NewContext()
	report := o.Execute(ctx)

	// Should be cancelled after step2
	assert.Equal(t, WorkflowStatusCancelled, report.Status)
}
