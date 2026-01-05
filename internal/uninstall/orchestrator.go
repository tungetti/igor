// Package uninstall provides the uninstallation workflow framework for Igor.
// This file implements the UninstallOrchestrator which coordinates all uninstall steps.
package uninstall

import (
	"fmt"
	"sync"
	"time"

	"github.com/tungetti/igor/internal/install"
)

// UninstallExecutionHook is called before/after workflow execution.
type UninstallExecutionHook func(ctx *Context, workflow UninstallWorkflow) error

// UninstallStepHook is called before/after each step execution.
type UninstallStepHook func(ctx *Context, step UninstallStep, result *install.StepResult) error

// UninstallExecutionEventType represents the type of execution event.
type UninstallExecutionEventType int

const (
	// UninstallEventWorkflowStarted indicates the workflow has started.
	UninstallEventWorkflowStarted UninstallExecutionEventType = iota
	// UninstallEventWorkflowCompleted indicates the workflow completed successfully.
	UninstallEventWorkflowCompleted
	// UninstallEventWorkflowFailed indicates the workflow failed.
	UninstallEventWorkflowFailed
	// UninstallEventWorkflowCancelled indicates the workflow was cancelled.
	UninstallEventWorkflowCancelled
	// UninstallEventStepStarted indicates a step has started.
	UninstallEventStepStarted
	// UninstallEventStepCompleted indicates a step completed successfully.
	UninstallEventStepCompleted
	// UninstallEventStepSkipped indicates a step was skipped.
	UninstallEventStepSkipped
	// UninstallEventStepFailed indicates a step failed.
	UninstallEventStepFailed
)

// String returns the string representation of the execution event type.
func (e UninstallExecutionEventType) String() string {
	switch e {
	case UninstallEventWorkflowStarted:
		return "workflow_started"
	case UninstallEventWorkflowCompleted:
		return "workflow_completed"
	case UninstallEventWorkflowFailed:
		return "workflow_failed"
	case UninstallEventWorkflowCancelled:
		return "workflow_cancelled"
	case UninstallEventStepStarted:
		return "step_started"
	case UninstallEventStepCompleted:
		return "step_completed"
	case UninstallEventStepSkipped:
		return "step_skipped"
	case UninstallEventStepFailed:
		return "step_failed"
	default:
		return "unknown"
	}
}

// UninstallExecutionEntry represents a single execution event in the log.
type UninstallExecutionEntry struct {
	// Timestamp is when this event occurred.
	Timestamp time.Time
	// StepName is the name of the step (if applicable).
	StepName string
	// EventType is the type of event.
	EventType UninstallExecutionEventType
	// Message is a human-readable message about the event.
	Message string
	// Duration is the duration of the step (if applicable).
	Duration time.Duration
	// Error is the error that occurred (if any).
	Error error
}

// UninstallExecutionReport provides detailed information about workflow execution.
type UninstallExecutionReport struct {
	// WorkflowName is the name of the workflow that was executed.
	WorkflowName string
	// Status is the final status of the workflow.
	Status UninstallStatus
	// StartTime is when the workflow started.
	StartTime time.Time
	// EndTime is when the workflow ended.
	EndTime time.Time
	// TotalDuration is the total time taken.
	TotalDuration time.Duration
	// StepsExecuted is the number of steps that were started.
	StepsExecuted int
	// StepsCompleted is the number of steps that completed successfully.
	StepsCompleted int
	// StepsSkipped is the number of steps that were skipped.
	StepsSkipped int
	// StepsFailed is the number of steps that failed.
	StepsFailed int
	// RemovedPackages lists packages that were successfully removed.
	RemovedPackages []string
	// CleanedConfigs lists configuration files that were removed.
	CleanedConfigs []string
	// NouveauRestored indicates if nouveau driver was restored.
	NouveauRestored bool
	// NeedsReboot indicates whether a system reboot is required.
	NeedsReboot bool
	// ExecutionLog contains all execution events.
	ExecutionLog []UninstallExecutionEntry
	// Error is the error that caused failure, if any.
	Error error
}

// UninstallOrchestrator manages the execution of uninstallation workflows.
// It provides features like hooks, execution policies, and detailed execution reporting.
type UninstallOrchestrator struct {
	workflow         UninstallWorkflow
	discovery        Discovery
	autoRollback     bool // Not really applicable for uninstall but kept for interface consistency
	stopOnFirstError bool
	dryRun           bool
	preExecuteHook   UninstallExecutionHook
	postExecuteHook  UninstallExecutionHook
	preStepHook      UninstallStepHook
	postStepHook     UninstallStepHook
	progressCallback func(install.StepProgress)
	executionLog     []UninstallExecutionEntry
	completedSteps   []UninstallStep
	mu               sync.RWMutex
}

// UninstallOrchestratorOption is a functional option for UninstallOrchestrator.
type UninstallOrchestratorOption func(*UninstallOrchestrator)

// WithUninstallWorkflow sets the workflow to execute.
func WithUninstallWorkflow(workflow UninstallWorkflow) UninstallOrchestratorOption {
	return func(o *UninstallOrchestrator) {
		o.workflow = workflow
	}
}

// WithUninstallOrchestratorDiscovery sets the discovery instance for the orchestrator.
func WithUninstallOrchestratorDiscovery(discovery Discovery) UninstallOrchestratorOption {
	return func(o *UninstallOrchestrator) {
		o.discovery = discovery
	}
}

// WithUninstallAutoRollback sets whether to automatically rollback on failure.
// Note: This is kept for interface consistency but is not typically used for uninstallation
// since "rolling back" an uninstall would mean re-installing, which is complex.
func WithUninstallAutoRollback(enabled bool) UninstallOrchestratorOption {
	return func(o *UninstallOrchestrator) {
		o.autoRollback = enabled
	}
}

// WithUninstallStopOnFirstError sets whether to stop immediately on first error.
func WithUninstallStopOnFirstError(enabled bool) UninstallOrchestratorOption {
	return func(o *UninstallOrchestrator) {
		o.stopOnFirstError = enabled
	}
}

// WithUninstallOrchestratorDryRun sets whether to execute in dry-run mode.
func WithUninstallOrchestratorDryRun(enabled bool) UninstallOrchestratorOption {
	return func(o *UninstallOrchestrator) {
		o.dryRun = enabled
	}
}

// WithUninstallPreExecuteHook sets the hook to call before workflow execution.
func WithUninstallPreExecuteHook(hook UninstallExecutionHook) UninstallOrchestratorOption {
	return func(o *UninstallOrchestrator) {
		o.preExecuteHook = hook
	}
}

// WithUninstallPostExecuteHook sets the hook to call after workflow execution.
func WithUninstallPostExecuteHook(hook UninstallExecutionHook) UninstallOrchestratorOption {
	return func(o *UninstallOrchestrator) {
		o.postExecuteHook = hook
	}
}

// WithUninstallPreStepHook sets the hook to call before each step execution.
func WithUninstallPreStepHook(hook UninstallStepHook) UninstallOrchestratorOption {
	return func(o *UninstallOrchestrator) {
		o.preStepHook = hook
	}
}

// WithUninstallPostStepHook sets the hook to call after each step execution.
func WithUninstallPostStepHook(hook UninstallStepHook) UninstallOrchestratorOption {
	return func(o *UninstallOrchestrator) {
		o.postStepHook = hook
	}
}

// WithUninstallOrchestratorProgress sets the progress callback.
func WithUninstallOrchestratorProgress(callback func(install.StepProgress)) UninstallOrchestratorOption {
	return func(o *UninstallOrchestrator) {
		o.progressCallback = callback
	}
}

// NewUninstallOrchestrator creates a new orchestrator with the given options.
func NewUninstallOrchestrator(opts ...UninstallOrchestratorOption) *UninstallOrchestrator {
	o := &UninstallOrchestrator{
		autoRollback:     false,
		stopOnFirstError: true,
		dryRun:           false,
		executionLog:     make([]UninstallExecutionEntry, 0),
		completedSteps:   make([]UninstallStep, 0),
	}
	for _, opt := range opts {
		opt(o)
	}
	return o
}

// Execute runs the workflow with orchestration features.
// Returns an UninstallExecutionReport with detailed execution information.
func (o *UninstallOrchestrator) Execute(ctx *Context) UninstallExecutionReport {
	startTime := time.Now()

	// Clear previous execution state
	o.mu.Lock()
	o.executionLog = make([]UninstallExecutionEntry, 0)
	o.completedSteps = make([]UninstallStep, 0)
	o.mu.Unlock()

	// Set dry run mode on context if enabled
	if o.dryRun && ctx != nil {
		ctx.DryRun = true
	}

	// Record workflow start event
	o.logEvent(UninstallExecutionEntry{
		Timestamp: time.Now(),
		EventType: UninstallEventWorkflowStarted,
		Message:   "Uninstall workflow execution started",
	})

	// Call preExecuteHook if set
	if o.preExecuteHook != nil {
		if err := o.preExecuteHook(ctx, o.workflow); err != nil {
			o.logEvent(UninstallExecutionEntry{
				Timestamp: time.Now(),
				EventType: UninstallEventWorkflowFailed,
				Message:   "Pre-execute hook failed",
				Error:     err,
			})
			return o.generateReport(startTime, NewUninstallResult(UninstallStatusFailed).WithError("pre-execute-hook", err))
		}
	}

	// Set up progress callback on workflow
	if o.progressCallback != nil && o.workflow != nil {
		o.workflow.OnProgress(o.progressCallback)
	}

	// Execute the workflow with step hooks
	result := o.executeWithHooks(ctx)

	// Call postExecuteHook if set
	if o.postExecuteHook != nil {
		if err := o.postExecuteHook(ctx, o.workflow); err != nil {
			o.logEvent(UninstallExecutionEntry{
				Timestamp: time.Now(),
				EventType: UninstallEventWorkflowFailed,
				Message:   "Post-execute hook failed",
				Error:     err,
			})
			// Only update result if workflow was successful
			if result.Status == UninstallStatusCompleted {
				result.Status = UninstallStatusFailed
				result.Error = err
			}
		}
	}

	// Record workflow end event
	eventType := UninstallEventWorkflowCompleted
	message := "Uninstall workflow execution completed successfully"
	switch result.Status {
	case UninstallStatusFailed:
		eventType = UninstallEventWorkflowFailed
		message = "Uninstall workflow execution failed"
	case UninstallStatusCancelled:
		eventType = UninstallEventWorkflowCancelled
		message = "Uninstall workflow execution cancelled"
	case UninstallStatusPartial:
		eventType = UninstallEventWorkflowCompleted
		message = "Uninstall workflow completed with partial results"
	}

	o.logEvent(UninstallExecutionEntry{
		Timestamp: time.Now(),
		EventType: eventType,
		Message:   message,
		Duration:  time.Since(startTime),
		Error:     result.Error,
	})

	return o.generateReport(startTime, result)
}

// executeWithHooks executes the workflow with step hooks.
func (o *UninstallOrchestrator) executeWithHooks(ctx *Context) UninstallResult {
	if o.workflow == nil {
		return NewUninstallResult(UninstallStatusFailed).WithError("workflow", fmt.Errorf("workflow is nil"))
	}

	// If no step hooks are set, just execute the workflow directly
	if o.preStepHook == nil && o.postStepHook == nil {
		return o.workflow.Execute(ctx)
	}

	// Get steps from workflow
	steps := o.workflow.Steps()
	startTime := time.Now()
	result := NewUninstallResult(UninstallStatusRunning)

	// Create an install.Context wrapper for step execution
	installCtx := o.createInstallContext(ctx)

	for i, step := range steps {
		// Check for cancellation
		if ctx != nil && ctx.IsCancelled() {
			result.Status = UninstallStatusCancelled
			result.TotalDuration = time.Since(startTime)
			return result
		}

		// Log step started
		stepStartTime := time.Now()
		o.logEvent(UninstallExecutionEntry{
			Timestamp: stepStartTime,
			StepName:  step.Name(),
			EventType: UninstallEventStepStarted,
			Message:   step.Description(),
		})

		// Call preStepHook if set
		if o.preStepHook != nil {
			if err := o.preStepHook(ctx, step, nil); err != nil {
				result.Status = UninstallStatusFailed
				result.FailedStep = step.Name()
				result.Error = err
				result.TotalDuration = time.Since(startTime)
				o.logEvent(UninstallExecutionEntry{
					Timestamp: time.Now(),
					StepName:  step.Name(),
					EventType: UninstallEventStepFailed,
					Message:   "Pre-step hook failed",
					Duration:  time.Since(stepStartTime),
					Error:     err,
				})
				return result
			}
		}

		// Validate step
		if err := step.Validate(installCtx); err != nil {
			result.Status = UninstallStatusFailed
			result.FailedStep = step.Name()
			result.Error = err
			result.TotalDuration = time.Since(startTime)
			o.logEvent(UninstallExecutionEntry{
				Timestamp: time.Now(),
				StepName:  step.Name(),
				EventType: UninstallEventStepFailed,
				Message:   "Validation failed",
				Duration:  time.Since(stepStartTime),
				Error:     err,
			})
			return result
		}

		// Execute step
		stepResult := step.Execute(installCtx)
		stepDuration := time.Since(stepStartTime)

		// Log step result
		switch stepResult.Status {
		case install.StepStatusCompleted:
			result.AddCompletedStep(step.Name())
			// Track completed step
			o.mu.Lock()
			o.completedSteps = append(o.completedSteps, step)
			o.mu.Unlock()
			o.logEvent(UninstallExecutionEntry{
				Timestamp: time.Now(),
				StepName:  step.Name(),
				EventType: UninstallEventStepCompleted,
				Message:   stepResult.Message,
				Duration:  stepDuration,
			})

			// Sync state from install context back to uninstall context and result
			o.syncStateFromInstallContext(installCtx, ctx, &result)

		case install.StepStatusSkipped:
			o.logEvent(UninstallExecutionEntry{
				Timestamp: time.Now(),
				StepName:  step.Name(),
				EventType: UninstallEventStepSkipped,
				Message:   stepResult.Message,
				Duration:  stepDuration,
			})

		case install.StepStatusFailed:
			result.Status = UninstallStatusFailed
			result.FailedStep = step.Name()
			result.Error = stepResult.Error
			o.logEvent(UninstallExecutionEntry{
				Timestamp: time.Now(),
				StepName:  step.Name(),
				EventType: UninstallEventStepFailed,
				Message:   stepResult.Message,
				Duration:  stepDuration,
				Error:     stepResult.Error,
			})

			// Call postStepHook before handling failure
			if o.postStepHook != nil {
				_ = o.postStepHook(ctx, step, &stepResult)
			}

			if o.stopOnFirstError {
				result.TotalDuration = time.Since(startTime)
				return result
			}
			// Continue to next step, skip the postStepHook call at end of loop
			continue

		default:
			result.Status = UninstallStatusFailed
			result.FailedStep = step.Name()
			result.Error = stepResult.Error
			o.logEvent(UninstallExecutionEntry{
				Timestamp: time.Now(),
				StepName:  step.Name(),
				EventType: UninstallEventStepFailed,
				Message:   "Unexpected step status",
				Duration:  stepDuration,
			})

			// Call postStepHook before handling failure
			if o.postStepHook != nil {
				_ = o.postStepHook(ctx, step, &stepResult)
			}

			if o.stopOnFirstError {
				result.TotalDuration = time.Since(startTime)
				return result
			}
			// Continue to next step, skip the postStepHook call at end of loop
			continue
		}

		// Call postStepHook if set (for completed/skipped steps)
		if o.postStepHook != nil {
			if err := o.postStepHook(ctx, step, &stepResult); err != nil {
				result.Status = UninstallStatusFailed
				result.FailedStep = step.Name()
				result.Error = err
				result.TotalDuration = time.Since(startTime)
				o.logEvent(UninstallExecutionEntry{
					Timestamp: time.Now(),
					StepName:  step.Name(),
					EventType: UninstallEventStepFailed,
					Message:   "Post-step hook failed",
					Duration:  time.Since(stepStartTime),
					Error:     err,
				})
				return result
			}
		}

		// Report progress
		if o.progressCallback != nil {
			o.progressCallback(install.NewStepProgress(step.Name(), i+1, len(steps), stepResult.Message))
		}
	}

	// All steps completed (or none failed fatally with stopOnFirstError=false)
	if result.Status == UninstallStatusRunning {
		result.Status = UninstallStatusCompleted
	}
	result.TotalDuration = time.Since(startTime)

	// Check if it's a partial result (some packages failed)
	if len(result.FailedPackages) > 0 && len(result.RemovedPackages) > 0 {
		result.Status = UninstallStatusPartial
	}

	return result
}

// GetExecutionLog returns a copy of the execution log entries.
func (o *UninstallOrchestrator) GetExecutionLog() []UninstallExecutionEntry {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return append([]UninstallExecutionEntry{}, o.executionLog...)
}

// Reset clears the execution state for re-use.
func (o *UninstallOrchestrator) Reset() {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.executionLog = make([]UninstallExecutionEntry, 0)
	o.completedSteps = make([]UninstallStep, 0)
}

// SetWorkflow replaces the workflow to execute.
func (o *UninstallOrchestrator) SetWorkflow(workflow UninstallWorkflow) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.workflow = workflow
}

// Workflow returns the current workflow.
func (o *UninstallOrchestrator) Workflow() UninstallWorkflow {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.workflow
}

// SetDiscovery sets the discovery instance.
func (o *UninstallOrchestrator) SetDiscovery(discovery Discovery) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.discovery = discovery
}

// Discovery returns the current discovery instance.
func (o *UninstallOrchestrator) Discovery() Discovery {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.discovery
}

// logEvent adds an execution event to the log.
func (o *UninstallOrchestrator) logEvent(event UninstallExecutionEntry) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.executionLog = append(o.executionLog, event)
}

// generateReport creates an UninstallExecutionReport from the workflow result.
func (o *UninstallOrchestrator) generateReport(startTime time.Time, result UninstallResult) UninstallExecutionReport {
	o.mu.RLock()
	executionLog := append([]UninstallExecutionEntry{}, o.executionLog...)
	o.mu.RUnlock()

	endTime := time.Now()

	// Count step statistics from execution log
	stepsExecuted := 0
	stepsCompleted := 0
	stepsSkipped := 0
	stepsFailed := 0

	for _, entry := range executionLog {
		switch entry.EventType {
		case UninstallEventStepStarted:
			stepsExecuted++
		case UninstallEventStepCompleted:
			stepsCompleted++
		case UninstallEventStepSkipped:
			stepsSkipped++
		case UninstallEventStepFailed:
			stepsFailed++
		}
	}

	workflowName := ""
	if o.workflow != nil {
		workflowName = o.workflow.Name()
	}

	return UninstallExecutionReport{
		WorkflowName:    workflowName,
		Status:          result.Status,
		StartTime:       startTime,
		EndTime:         endTime,
		TotalDuration:   endTime.Sub(startTime),
		StepsExecuted:   stepsExecuted,
		StepsCompleted:  stepsCompleted,
		StepsSkipped:    stepsSkipped,
		StepsFailed:     stepsFailed,
		RemovedPackages: append([]string{}, result.RemovedPackages...),
		CleanedConfigs:  append([]string{}, result.CleanedConfigs...),
		NouveauRestored: result.NouveauRestored,
		NeedsReboot:     result.NeedsReboot,
		ExecutionLog:    executionLog,
		Error:           result.Error,
	}
}

// createInstallContext creates an install.Context from an uninstall.Context.
// This allows us to reuse the install.Step interface.
func (o *UninstallOrchestrator) createInstallContext(ctx *Context) *install.Context {
	if ctx == nil {
		return install.NewContext()
	}
	return install.NewContext(
		install.WithDistroInfo(ctx.DistroInfo),
		install.WithPackageManager(ctx.PackageManager),
		install.WithExecutor(ctx.Executor),
		install.WithPrivilege(ctx.Privilege),
		install.WithLogger(ctx.Logger),
		install.WithDryRun(ctx.DryRun),
	)
}

// syncStateFromInstallContext syncs relevant state from the install context
// back to the uninstall context and result.
func (o *UninstallOrchestrator) syncStateFromInstallContext(installCtx *install.Context, ctx *Context, result *UninstallResult) {
	if installCtx == nil || ctx == nil {
		return
	}

	// Sync removed packages
	if packages, ok := installCtx.GetState(StateRemovedPackages); ok {
		if pkgList, ok := packages.([]string); ok {
			for _, pkg := range pkgList {
				result.AddRemovedPackage(pkg)
			}
			ctx.SetState(StateRemovedPackages, pkgList)
		}
	}

	// Sync cleaned configs
	if configs, ok := installCtx.GetState(StateCleanedConfigs); ok {
		if configList, ok := configs.([]string); ok {
			for _, cfg := range configList {
				result.AddCleanedConfig(cfg)
			}
			ctx.SetState(StateCleanedConfigs, configList)
		}
	}

	// Sync boolean states
	if v, ok := installCtx.GetState(StatePackagesRemoved); ok {
		ctx.SetState(StatePackagesRemoved, v)
	}
	if v, ok := installCtx.GetState(StateConfigsCleaned); ok {
		ctx.SetState(StateConfigsCleaned, v)
	}
	if v, ok := installCtx.GetState(StateModulesUnloaded); ok {
		ctx.SetState(StateModulesUnloaded, v)
	}
	if v, ok := installCtx.GetState(StateNouveauUnblocked); ok {
		ctx.SetState(StateNouveauUnblocked, v)
	}
	if restored, ok := installCtx.GetState(StateNouveauRestored); ok {
		ctx.SetState(StateNouveauRestored, restored)
		if b, ok := restored.(bool); ok {
			result.NouveauRestored = b
		}
	}
}

// Ensure UninstallOrchestrator is properly constructed with compile-time checks.
var _ = NewUninstallOrchestrator
