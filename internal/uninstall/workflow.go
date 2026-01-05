package uninstall

import (
	"fmt"
	"sync"
	"time"

	"github.com/tungetti/igor/internal/install"
)

// UninstallWorkflow represents an uninstallation workflow.
type UninstallWorkflow interface {
	// Name returns the workflow name.
	Name() string

	// Steps returns the ordered list of steps in the workflow.
	Steps() []UninstallStep

	// AddStep adds a step to the workflow.
	AddStep(step UninstallStep)

	// Execute runs all steps in order.
	Execute(ctx *Context) UninstallResult

	// OnProgress sets a callback for progress updates.
	OnProgress(callback func(install.StepProgress))

	// Cancel requests cancellation of the workflow.
	Cancel()
}

// BaseUninstallWorkflow provides a default uninstall workflow implementation.
type BaseUninstallWorkflow struct {
	name           string
	steps          []UninstallStep
	completedSteps []UninstallStep
	progressCb     func(install.StepProgress)
	cancelled      bool
	mu             sync.RWMutex
}

// NewUninstallWorkflow creates a new uninstall workflow with the given name.
func NewUninstallWorkflow(name string) *BaseUninstallWorkflow {
	return &BaseUninstallWorkflow{
		name:           name,
		steps:          make([]UninstallStep, 0),
		completedSteps: make([]UninstallStep, 0),
	}
}

// Name returns the workflow name.
func (w *BaseUninstallWorkflow) Name() string {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.name
}

// Steps returns a copy of the steps slice.
func (w *BaseUninstallWorkflow) Steps() []UninstallStep {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return append([]UninstallStep{}, w.steps...)
}

// AddStep adds a step to the workflow.
func (w *BaseUninstallWorkflow) AddStep(step UninstallStep) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.steps = append(w.steps, step)
}

// Execute runs all steps in order.
func (w *BaseUninstallWorkflow) Execute(ctx *Context) UninstallResult {
	startTime := time.Now()

	w.mu.Lock()
	w.cancelled = false
	w.completedSteps = make([]UninstallStep, 0)
	steps := append([]UninstallStep{}, w.steps...)
	w.mu.Unlock()

	result := NewUninstallResult(UninstallStatusRunning)

	// Create an install.Context wrapper for step execution
	installCtx := w.createInstallContext(ctx)

	for i, step := range steps {
		// Check for cancellation
		if w.isCancelled() {
			result.Status = UninstallStatusCancelled
			result.TotalDuration = time.Since(startTime)
			return result
		}

		// Check context cancellation
		if ctx != nil && ctx.IsCancelled() {
			result.Status = UninstallStatusCancelled
			result.TotalDuration = time.Since(startTime)
			return result
		}

		// Report progress
		w.reportProgress(step.Name(), i, len(steps), fmt.Sprintf("Starting: %s", step.Description()))

		// Validate step
		if err := step.Validate(installCtx); err != nil {
			result.Status = UninstallStatusFailed
			result.FailedStep = step.Name()
			result.Error = fmt.Errorf("validation failed: %w", err)
			result.TotalDuration = time.Since(startTime)
			return result
		}

		// Execute step
		stepResult := step.Execute(installCtx)

		// Handle step result
		switch stepResult.Status {
		case install.StepStatusCompleted:
			w.mu.Lock()
			w.completedSteps = append(w.completedSteps, step)
			w.mu.Unlock()
			result.AddCompletedStep(step.Name())
			w.reportProgress(step.Name(), i, len(steps), fmt.Sprintf("Completed: %s", stepResult.Message))

			// Sync state from install context back to uninstall context
			w.syncStateFromInstallContext(installCtx, ctx, &result)

		case install.StepStatusSkipped:
			w.reportProgress(step.Name(), i, len(steps), fmt.Sprintf("Skipped: %s", stepResult.Message))

		case install.StepStatusFailed:
			result.Status = UninstallStatusFailed
			result.FailedStep = step.Name()
			result.Error = stepResult.Error
			result.TotalDuration = time.Since(startTime)
			w.reportProgress(step.Name(), i, len(steps), fmt.Sprintf("Failed: %s", stepResult.Message))
			return result

		default:
			// Unexpected status, treat as failure
			result.Status = UninstallStatusFailed
			result.FailedStep = step.Name()
			result.Error = fmt.Errorf("unexpected step status: %s", stepResult.Status)
			result.TotalDuration = time.Since(startTime)
			return result
		}
	}

	// All steps completed successfully
	result.Status = UninstallStatusCompleted
	result.TotalDuration = time.Since(startTime)

	// Check if it's a partial result (some packages failed)
	if len(result.FailedPackages) > 0 && len(result.RemovedPackages) > 0 {
		result.Status = UninstallStatusPartial
	}

	// Report final progress
	w.reportProgress("", len(steps), len(steps), "Uninstallation completed successfully")

	return result
}

// OnProgress sets a callback for progress updates.
func (w *BaseUninstallWorkflow) OnProgress(callback func(install.StepProgress)) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.progressCb = callback
}

// Cancel requests cancellation of the workflow.
func (w *BaseUninstallWorkflow) Cancel() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.cancelled = true
}

// isCancelled returns whether cancellation has been requested.
func (w *BaseUninstallWorkflow) isCancelled() bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.cancelled
}

// reportProgress sends a progress update to the callback if set.
func (w *BaseUninstallWorkflow) reportProgress(stepName string, index, total int, message string) {
	w.mu.RLock()
	cb := w.progressCb
	w.mu.RUnlock()

	if cb != nil {
		cb(install.NewStepProgress(stepName, index, total, message))
	}
}

// CompletedSteps returns the list of completed steps.
func (w *BaseUninstallWorkflow) CompletedSteps() []UninstallStep {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return append([]UninstallStep{}, w.completedSteps...)
}

// IsCancelled returns whether the workflow was cancelled.
func (w *BaseUninstallWorkflow) IsCancelled() bool {
	return w.isCancelled()
}

// Reset resets the workflow state for re-execution.
func (w *BaseUninstallWorkflow) Reset() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.completedSteps = make([]UninstallStep, 0)
	w.cancelled = false
}

// createInstallContext creates an install.Context from an uninstall.Context.
// This allows us to reuse the install.Step interface.
func (w *BaseUninstallWorkflow) createInstallContext(ctx *Context) *install.Context {
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
func (w *BaseUninstallWorkflow) syncStateFromInstallContext(installCtx *install.Context, ctx *Context, result *UninstallResult) {
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

// Ensure BaseUninstallWorkflow implements UninstallWorkflow.
var _ UninstallWorkflow = (*BaseUninstallWorkflow)(nil)
