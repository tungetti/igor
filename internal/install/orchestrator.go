package install

import (
	"fmt"
	"sync"
	"time"
)

// ExecutionHook is called before/after workflow execution.
type ExecutionHook func(ctx *Context, workflow Workflow) error

// StepHook is called before/after each step execution.
type StepHook func(ctx *Context, step Step, result *StepResult) error

// ExecutionEventType represents the type of execution event.
type ExecutionEventType int

const (
	EventWorkflowStarted ExecutionEventType = iota
	EventWorkflowCompleted
	EventWorkflowFailed
	EventWorkflowCancelled
	EventWorkflowRollbackStarted
	EventWorkflowRollbackCompleted
	EventStepStarted
	EventStepCompleted
	EventStepSkipped
	EventStepFailed
	EventStepRollbackStarted
	EventStepRollbackCompleted
)

// String returns the string representation of the execution event type.
func (e ExecutionEventType) String() string {
	switch e {
	case EventWorkflowStarted:
		return "workflow_started"
	case EventWorkflowCompleted:
		return "workflow_completed"
	case EventWorkflowFailed:
		return "workflow_failed"
	case EventWorkflowCancelled:
		return "workflow_cancelled"
	case EventWorkflowRollbackStarted:
		return "workflow_rollback_started"
	case EventWorkflowRollbackCompleted:
		return "workflow_rollback_completed"
	case EventStepStarted:
		return "step_started"
	case EventStepCompleted:
		return "step_completed"
	case EventStepSkipped:
		return "step_skipped"
	case EventStepFailed:
		return "step_failed"
	case EventStepRollbackStarted:
		return "step_rollback_started"
	case EventStepRollbackCompleted:
		return "step_rollback_completed"
	default:
		return "unknown"
	}
}

// ExecutionEntry represents a single execution event.
type ExecutionEntry struct {
	Timestamp time.Time
	StepName  string
	EventType ExecutionEventType
	Message   string
	Duration  time.Duration
	Error     error
}

// ExecutionReport provides detailed information about workflow execution.
type ExecutionReport struct {
	WorkflowName      string
	Status            WorkflowStatus
	StartTime         time.Time
	EndTime           time.Time
	TotalDuration     time.Duration
	StepsExecuted     int
	StepsCompleted    int
	StepsSkipped      int
	StepsFailed       int
	RollbackPerformed bool
	RollbackSuccess   bool
	ExecutionLog      []ExecutionEntry
	Error             error
}

// Orchestrator manages the execution of installation workflows.
// It provides features like automatic rollback, hooks, execution policies,
// and detailed execution reporting.
type Orchestrator struct {
	workflow         Workflow
	autoRollback     bool
	stopOnFirstError bool
	dryRun           bool
	preExecuteHook   ExecutionHook
	postExecuteHook  ExecutionHook
	preStepHook      StepHook
	postStepHook     StepHook
	progressCallback func(StepProgress)
	executionLog     []ExecutionEntry
	completedSteps   []Step // Steps completed during execution (for rollback when using hooks)
	mu               sync.RWMutex
}

// OrchestratorOption is a functional option for Orchestrator.
type OrchestratorOption func(*Orchestrator)

// WithAutoRollback sets whether to automatically rollback on failure.
func WithAutoRollback(enabled bool) OrchestratorOption {
	return func(o *Orchestrator) {
		o.autoRollback = enabled
	}
}

// WithStopOnFirstError sets whether to stop immediately on first error.
func WithStopOnFirstError(enabled bool) OrchestratorOption {
	return func(o *Orchestrator) {
		o.stopOnFirstError = enabled
	}
}

// WithOrchestratorDryRun sets whether to execute in dry-run mode.
func WithOrchestratorDryRun(enabled bool) OrchestratorOption {
	return func(o *Orchestrator) {
		o.dryRun = enabled
	}
}

// WithPreExecuteHook sets the hook to call before workflow execution.
func WithPreExecuteHook(hook ExecutionHook) OrchestratorOption {
	return func(o *Orchestrator) {
		o.preExecuteHook = hook
	}
}

// WithPostExecuteHook sets the hook to call after workflow execution.
func WithPostExecuteHook(hook ExecutionHook) OrchestratorOption {
	return func(o *Orchestrator) {
		o.postExecuteHook = hook
	}
}

// WithPreStepHook sets the hook to call before each step execution.
func WithPreStepHook(hook StepHook) OrchestratorOption {
	return func(o *Orchestrator) {
		o.preStepHook = hook
	}
}

// WithPostStepHook sets the hook to call after each step execution.
func WithPostStepHook(hook StepHook) OrchestratorOption {
	return func(o *Orchestrator) {
		o.postStepHook = hook
	}
}

// WithOrchestratorProgress sets the progress callback.
func WithOrchestratorProgress(callback func(StepProgress)) OrchestratorOption {
	return func(o *Orchestrator) {
		o.progressCallback = callback
	}
}

// NewOrchestrator creates a new orchestrator for the given workflow.
func NewOrchestrator(workflow Workflow, opts ...OrchestratorOption) *Orchestrator {
	o := &Orchestrator{
		workflow:         workflow,
		autoRollback:     false,
		stopOnFirstError: true,
		dryRun:           false,
		executionLog:     make([]ExecutionEntry, 0),
	}
	for _, opt := range opts {
		opt(o)
	}
	return o
}

// Execute runs the workflow with orchestration features.
// Returns an ExecutionReport with detailed execution information.
func (o *Orchestrator) Execute(ctx *Context) ExecutionReport {
	startTime := time.Now()

	// Clear previous execution state
	o.mu.Lock()
	o.executionLog = make([]ExecutionEntry, 0)
	o.completedSteps = make([]Step, 0)
	o.mu.Unlock()

	// Set dry run mode on context if enabled
	if o.dryRun && ctx != nil {
		ctx.DryRun = true
	}

	// Record workflow start event
	o.logEvent(ExecutionEntry{
		Timestamp: time.Now(),
		EventType: EventWorkflowStarted,
		Message:   "Workflow execution started",
	})

	// Call preExecuteHook if set
	if o.preExecuteHook != nil {
		if err := o.preExecuteHook(ctx, o.workflow); err != nil {
			o.logEvent(ExecutionEntry{
				Timestamp: time.Now(),
				EventType: EventWorkflowFailed,
				Message:   "Pre-execute hook failed",
				Error:     err,
			})
			return o.generateReport(startTime, NewWorkflowResult(WorkflowStatusFailed).WithError("pre-execute-hook", err))
		}
	}

	// Set up progress callback on workflow
	if o.progressCallback != nil {
		o.workflow.OnProgress(o.progressCallback)
	}

	// Execute the workflow with step hooks
	result := o.executeWithHooks(ctx)

	// Handle auto rollback on failure
	if result.Status == WorkflowStatusFailed && o.autoRollback {
		rollbackErr := o.performRollback(ctx)
		if rollbackErr != nil {
			result.Error = rollbackErr
		}
	}

	// Call postExecuteHook if set
	if o.postExecuteHook != nil {
		if err := o.postExecuteHook(ctx, o.workflow); err != nil {
			o.logEvent(ExecutionEntry{
				Timestamp: time.Now(),
				EventType: EventWorkflowFailed,
				Message:   "Post-execute hook failed",
				Error:     err,
			})
			// Only update result if workflow was successful
			if result.Status == WorkflowStatusCompleted {
				result.Status = WorkflowStatusFailed
				result.Error = err
			}
		}
	}

	// Record workflow end event
	eventType := EventWorkflowCompleted
	message := "Workflow execution completed successfully"
	switch result.Status {
	case WorkflowStatusFailed:
		eventType = EventWorkflowFailed
		message = "Workflow execution failed"
	case WorkflowStatusCancelled:
		eventType = EventWorkflowCancelled
		message = "Workflow execution cancelled"
	}

	o.logEvent(ExecutionEntry{
		Timestamp: time.Now(),
		EventType: eventType,
		Message:   message,
		Duration:  time.Since(startTime),
		Error:     result.Error,
	})

	return o.generateReport(startTime, result)
}

// executeWithHooks executes the workflow with step hooks.
func (o *Orchestrator) executeWithHooks(ctx *Context) WorkflowResult {
	if o.workflow == nil {
		return NewWorkflowResult(WorkflowStatusFailed).WithError("workflow", fmt.Errorf("workflow is nil"))
	}

	// If no step hooks are set, just execute the workflow directly
	if o.preStepHook == nil && o.postStepHook == nil {
		return o.workflow.Execute(ctx)
	}

	// Get steps from workflow
	steps := o.workflow.Steps()
	startTime := time.Now()
	result := NewWorkflowResult(WorkflowStatusRunning)

	for i, step := range steps {
		// Check for cancellation
		if ctx != nil && ctx.IsCancelled() {
			result.Status = WorkflowStatusCancelled
			result.TotalDuration = time.Since(startTime)
			return result
		}

		// Log step started
		stepStartTime := time.Now()
		o.logEvent(ExecutionEntry{
			Timestamp: stepStartTime,
			StepName:  step.Name(),
			EventType: EventStepStarted,
			Message:   step.Description(),
		})

		// Call preStepHook if set
		if o.preStepHook != nil {
			if err := o.preStepHook(ctx, step, nil); err != nil {
				result.Status = WorkflowStatusFailed
				result.FailedStep = step.Name()
				result.Error = err
				result.TotalDuration = time.Since(startTime)
				o.logEvent(ExecutionEntry{
					Timestamp: time.Now(),
					StepName:  step.Name(),
					EventType: EventStepFailed,
					Message:   "Pre-step hook failed",
					Duration:  time.Since(stepStartTime),
					Error:     err,
				})
				return result
			}
		}

		// Validate step
		if err := step.Validate(ctx); err != nil {
			result.Status = WorkflowStatusFailed
			result.FailedStep = step.Name()
			result.Error = err
			result.TotalDuration = time.Since(startTime)
			o.logEvent(ExecutionEntry{
				Timestamp: time.Now(),
				StepName:  step.Name(),
				EventType: EventStepFailed,
				Message:   "Validation failed",
				Duration:  time.Since(stepStartTime),
				Error:     err,
			})
			return result
		}

		// Execute step
		stepResult := step.Execute(ctx)
		stepDuration := time.Since(stepStartTime)

		// Log step result
		switch stepResult.Status {
		case StepStatusCompleted:
			result.AddCompletedStep(step.Name())
			// Track completed step for rollback
			o.mu.Lock()
			o.completedSteps = append(o.completedSteps, step)
			o.mu.Unlock()
			o.logEvent(ExecutionEntry{
				Timestamp: time.Now(),
				StepName:  step.Name(),
				EventType: EventStepCompleted,
				Message:   stepResult.Message,
				Duration:  stepDuration,
			})

		case StepStatusSkipped:
			o.logEvent(ExecutionEntry{
				Timestamp: time.Now(),
				StepName:  step.Name(),
				EventType: EventStepSkipped,
				Message:   stepResult.Message,
				Duration:  stepDuration,
			})

		case StepStatusFailed:
			result.Status = WorkflowStatusFailed
			result.FailedStep = step.Name()
			result.Error = stepResult.Error
			o.logEvent(ExecutionEntry{
				Timestamp: time.Now(),
				StepName:  step.Name(),
				EventType: EventStepFailed,
				Message:   stepResult.Message,
				Duration:  stepDuration,
				Error:     stepResult.Error,
			})

			// Call postStepHook before returning on failure
			if o.postStepHook != nil {
				_ = o.postStepHook(ctx, step, &stepResult)
			}

			if o.stopOnFirstError {
				result.TotalDuration = time.Since(startTime)
				return result
			}

		default:
			result.Status = WorkflowStatusFailed
			result.FailedStep = step.Name()
			result.Error = stepResult.Error
			o.logEvent(ExecutionEntry{
				Timestamp: time.Now(),
				StepName:  step.Name(),
				EventType: EventStepFailed,
				Message:   "Unexpected step status",
				Duration:  stepDuration,
			})

			if o.stopOnFirstError {
				result.TotalDuration = time.Since(startTime)
				return result
			}
		}

		// Call postStepHook if set
		if o.postStepHook != nil {
			if err := o.postStepHook(ctx, step, &stepResult); err != nil {
				result.Status = WorkflowStatusFailed
				result.FailedStep = step.Name()
				result.Error = err
				result.TotalDuration = time.Since(startTime)
				o.logEvent(ExecutionEntry{
					Timestamp: time.Now(),
					StepName:  step.Name(),
					EventType: EventStepFailed,
					Message:   "Post-step hook failed",
					Duration:  time.Since(stepStartTime),
					Error:     err,
				})
				return result
			}
		}

		// Report progress
		if o.progressCallback != nil {
			o.progressCallback(NewStepProgress(step.Name(), i+1, len(steps), stepResult.Message))
		}
	}

	// All steps completed (or none failed fatally with stopOnFirstError=false)
	if result.Status == WorkflowStatusRunning {
		result.Status = WorkflowStatusCompleted
	}
	result.TotalDuration = time.Since(startTime)

	return result
}

// ExecuteWithRollback runs the workflow and automatically rolls back on failure.
func (o *Orchestrator) ExecuteWithRollback(ctx *Context) ExecutionReport {
	o.mu.Lock()
	o.autoRollback = true
	o.mu.Unlock()

	return o.Execute(ctx)
}

// performRollback performs rollback on the workflow.
// If step hooks were used, rolls back using orchestrator's tracked steps.
// Otherwise, delegates to workflow's Rollback method.
func (o *Orchestrator) performRollback(ctx *Context) error {
	o.logEvent(ExecutionEntry{
		Timestamp: time.Now(),
		EventType: EventWorkflowRollbackStarted,
		Message:   "Starting workflow rollback",
	})

	var err error

	// If we tracked completed steps (step hooks were used), rollback manually
	o.mu.RLock()
	completedSteps := append([]Step{}, o.completedSteps...)
	o.mu.RUnlock()

	if len(completedSteps) > 0 {
		// Rollback in reverse order
		var rollbackErrors []error
		for i := len(completedSteps) - 1; i >= 0; i-- {
			step := completedSteps[i]
			if !step.CanRollback() {
				continue
			}

			o.logEvent(ExecutionEntry{
				Timestamp: time.Now(),
				StepName:  step.Name(),
				EventType: EventStepRollbackStarted,
				Message:   "Rolling back step",
			})

			if stepErr := step.Rollback(ctx); stepErr != nil {
				rollbackErrors = append(rollbackErrors, stepErr)
				o.logEvent(ExecutionEntry{
					Timestamp: time.Now(),
					StepName:  step.Name(),
					EventType: EventStepRollbackCompleted,
					Message:   "Step rollback failed",
					Error:     stepErr,
				})
			} else {
				o.logEvent(ExecutionEntry{
					Timestamp: time.Now(),
					StepName:  step.Name(),
					EventType: EventStepRollbackCompleted,
					Message:   "Step rollback completed",
				})
			}
		}

		if len(rollbackErrors) > 0 {
			err = rollbackErrors[0] // Return first error
		}
	} else {
		// Delegate to workflow's rollback (steps were executed by workflow directly)
		err = o.workflow.Rollback(ctx)
	}

	eventType := EventWorkflowRollbackCompleted
	message := "Workflow rollback completed successfully"
	if err != nil {
		message = "Workflow rollback completed with errors"
	}

	o.logEvent(ExecutionEntry{
		Timestamp: time.Now(),
		EventType: eventType,
		Message:   message,
		Error:     err,
	})

	return err
}

// GetExecutionLog returns a copy of the execution log entries.
func (o *Orchestrator) GetExecutionLog() []ExecutionEntry {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return append([]ExecutionEntry{}, o.executionLog...)
}

// Reset clears the execution state for re-use.
func (o *Orchestrator) Reset() {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.executionLog = make([]ExecutionEntry, 0)
	o.completedSteps = make([]Step, 0)
}

// SetWorkflow replaces the workflow to execute.
func (o *Orchestrator) SetWorkflow(workflow Workflow) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.workflow = workflow
}

// Workflow returns the current workflow.
func (o *Orchestrator) Workflow() Workflow {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.workflow
}

// logEvent adds an execution event to the log.
func (o *Orchestrator) logEvent(event ExecutionEntry) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.executionLog = append(o.executionLog, event)
}

// generateReport creates an ExecutionReport from the workflow result.
func (o *Orchestrator) generateReport(startTime time.Time, result WorkflowResult) ExecutionReport {
	o.mu.RLock()
	executionLog := append([]ExecutionEntry{}, o.executionLog...)
	o.mu.RUnlock()

	endTime := time.Now()

	// Count step statistics from execution log
	stepsExecuted := 0
	stepsCompleted := 0
	stepsSkipped := 0
	stepsFailed := 0
	rollbackPerformed := false
	rollbackSuccess := true

	for _, entry := range executionLog {
		switch entry.EventType {
		case EventStepStarted:
			stepsExecuted++
		case EventStepCompleted:
			stepsCompleted++
		case EventStepSkipped:
			stepsSkipped++
		case EventStepFailed:
			stepsFailed++
		case EventWorkflowRollbackStarted:
			rollbackPerformed = true
		case EventWorkflowRollbackCompleted:
			if entry.Error != nil {
				rollbackSuccess = false
			}
		}
	}

	workflowName := ""
	if o.workflow != nil {
		workflowName = o.workflow.Name()
	}

	return ExecutionReport{
		WorkflowName:      workflowName,
		Status:            result.Status,
		StartTime:         startTime,
		EndTime:           endTime,
		TotalDuration:     endTime.Sub(startTime),
		StepsExecuted:     stepsExecuted,
		StepsCompleted:    stepsCompleted,
		StepsSkipped:      stepsSkipped,
		StepsFailed:       stepsFailed,
		RollbackPerformed: rollbackPerformed,
		RollbackSuccess:   rollbackSuccess,
		ExecutionLog:      executionLog,
		Error:             result.Error,
	}
}
