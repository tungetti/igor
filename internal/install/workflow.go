package install

import (
	"fmt"
	"sync"
	"time"
)

// Workflow represents an installation workflow composed of multiple steps.
type Workflow interface {
	// Name returns the workflow name.
	Name() string

	// Steps returns the ordered list of steps in the workflow.
	Steps() []Step

	// AddStep adds a step to the workflow.
	AddStep(step Step)

	// Execute runs all steps in order.
	// Returns a WorkflowResult containing the outcome.
	Execute(ctx *Context) WorkflowResult

	// Rollback reverses completed steps in reverse order.
	Rollback(ctx *Context) error

	// OnProgress sets a callback for progress updates.
	OnProgress(callback func(StepProgress))

	// Cancel requests cancellation of the workflow.
	Cancel()
}

// BaseWorkflow provides a default workflow implementation.
type BaseWorkflow struct {
	name           string
	steps          []Step
	completedSteps []Step
	progressCb     func(StepProgress)
	cancelled      bool
	mu             sync.RWMutex
}

// NewWorkflow creates a new workflow with the given name.
func NewWorkflow(name string) *BaseWorkflow {
	return &BaseWorkflow{
		name:           name,
		steps:          make([]Step, 0),
		completedSteps: make([]Step, 0),
	}
}

// Name returns the workflow name.
func (w *BaseWorkflow) Name() string {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.name
}

// Steps returns a copy of the steps slice.
func (w *BaseWorkflow) Steps() []Step {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return append([]Step{}, w.steps...)
}

// AddStep adds a step to the workflow.
func (w *BaseWorkflow) AddStep(step Step) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.steps = append(w.steps, step)
}

// Execute runs all steps in order.
func (w *BaseWorkflow) Execute(ctx *Context) WorkflowResult {
	startTime := time.Now()

	w.mu.Lock()
	w.cancelled = false
	w.completedSteps = make([]Step, 0)
	steps := append([]Step{}, w.steps...)
	w.mu.Unlock()

	result := NewWorkflowResult(WorkflowStatusRunning)

	for i, step := range steps {
		// Check for cancellation
		if w.isCancelled() {
			result.Status = WorkflowStatusCancelled
			result.TotalDuration = time.Since(startTime)
			return result
		}

		// Check context cancellation
		if ctx != nil && ctx.IsCancelled() {
			result.Status = WorkflowStatusCancelled
			result.TotalDuration = time.Since(startTime)
			return result
		}

		// Report progress
		w.reportProgress(step.Name(), i, len(steps), fmt.Sprintf("Starting: %s", step.Description()))

		// Validate step
		if err := step.Validate(ctx); err != nil {
			result.Status = WorkflowStatusFailed
			result.FailedStep = step.Name()
			result.Error = fmt.Errorf("validation failed: %w", err)
			result.TotalDuration = time.Since(startTime)
			return result
		}

		// Execute step
		stepResult := step.Execute(ctx)

		// Handle step result
		switch stepResult.Status {
		case StepStatusCompleted:
			w.mu.Lock()
			w.completedSteps = append(w.completedSteps, step)
			w.mu.Unlock()
			result.AddCompletedStep(step.Name())
			w.reportProgress(step.Name(), i, len(steps), fmt.Sprintf("Completed: %s", stepResult.Message))

		case StepStatusSkipped:
			w.reportProgress(step.Name(), i, len(steps), fmt.Sprintf("Skipped: %s", stepResult.Message))

		case StepStatusFailed:
			result.Status = WorkflowStatusFailed
			result.FailedStep = step.Name()
			result.Error = stepResult.Error
			result.TotalDuration = time.Since(startTime)
			w.reportProgress(step.Name(), i, len(steps), fmt.Sprintf("Failed: %s", stepResult.Message))
			return result

		default:
			// Unexpected status, treat as failure
			result.Status = WorkflowStatusFailed
			result.FailedStep = step.Name()
			result.Error = fmt.Errorf("unexpected step status: %s", stepResult.Status)
			result.TotalDuration = time.Since(startTime)
			return result
		}
	}

	// All steps completed successfully
	result.Status = WorkflowStatusCompleted
	result.TotalDuration = time.Since(startTime)

	// Report final progress
	w.reportProgress("", len(steps), len(steps), "Workflow completed successfully")

	return result
}

// Rollback reverses completed steps in reverse order.
func (w *BaseWorkflow) Rollback(ctx *Context) error {
	w.mu.Lock()
	completed := append([]Step{}, w.completedSteps...)
	w.mu.Unlock()

	// Rollback in reverse order
	var rollbackErrors []error
	for i := len(completed) - 1; i >= 0; i-- {
		step := completed[i]
		if !step.CanRollback() {
			continue
		}

		w.reportProgress(step.Name(), i, len(completed), fmt.Sprintf("Rolling back: %s", step.Description()))

		if err := step.Rollback(ctx); err != nil {
			rollbackErrors = append(rollbackErrors, fmt.Errorf("rollback of '%s' failed: %w", step.Name(), err))
		}
	}

	if len(rollbackErrors) > 0 {
		return fmt.Errorf("rollback completed with %d errors: %v", len(rollbackErrors), rollbackErrors)
	}

	return nil
}

// OnProgress sets a callback for progress updates.
func (w *BaseWorkflow) OnProgress(callback func(StepProgress)) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.progressCb = callback
}

// Cancel requests cancellation of the workflow.
func (w *BaseWorkflow) Cancel() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.cancelled = true
}

// isCancelled returns whether cancellation has been requested.
func (w *BaseWorkflow) isCancelled() bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.cancelled
}

// reportProgress sends a progress update to the callback if set.
func (w *BaseWorkflow) reportProgress(stepName string, index, total int, message string) {
	w.mu.RLock()
	cb := w.progressCb
	w.mu.RUnlock()

	if cb != nil {
		cb(NewStepProgress(stepName, index, total, message))
	}
}

// CompletedSteps returns the list of completed steps.
func (w *BaseWorkflow) CompletedSteps() []Step {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return append([]Step{}, w.completedSteps...)
}

// IsCancelled returns whether the workflow was cancelled.
func (w *BaseWorkflow) IsCancelled() bool {
	return w.isCancelled()
}

// Reset resets the workflow state for re-execution.
func (w *BaseWorkflow) Reset() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.completedSteps = make([]Step, 0)
	w.cancelled = false
}

// Ensure BaseWorkflow implements Workflow.
var _ Workflow = (*BaseWorkflow)(nil)
