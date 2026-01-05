package uninstall

import (
	"time"

	"github.com/tungetti/igor/internal/install"
)

// UninstallStep is an alias for install.Step - uninstall steps follow the same interface.
// Steps execute removal operations and can be "rolled back" by re-installing if needed.
type UninstallStep = install.Step

// BaseUninstallStep provides common functionality for uninstall steps.
// It wraps the install.BaseStep to provide consistent behavior.
type BaseUninstallStep = install.BaseStep

// NewBaseUninstallStep creates a new base uninstall step with the given name, description, and rollback capability.
func NewBaseUninstallStep(name, description string, canRollback bool) BaseUninstallStep {
	return install.NewBaseStep(name, description, canRollback)
}

// UninstallFuncStep is a step implementation that uses functions for execution and rollback.
// This is useful for simple uninstall steps that don't need a full struct implementation.
type UninstallFuncStep struct {
	install.BaseStep
	executeFunc  func(ctx *Context) install.StepResult
	rollbackFunc func(ctx *Context) error
	validateFunc func(ctx *Context) error
}

// UninstallFuncStepOption is a functional option for UninstallFuncStep.
type UninstallFuncStepOption func(*UninstallFuncStep)

// WithUninstallRollbackFunc sets the rollback function for an UninstallFuncStep.
func WithUninstallRollbackFunc(fn func(ctx *Context) error) UninstallFuncStepOption {
	return func(s *UninstallFuncStep) {
		s.rollbackFunc = fn
	}
}

// WithUninstallValidateFunc sets the validate function for an UninstallFuncStep.
func WithUninstallValidateFunc(fn func(ctx *Context) error) UninstallFuncStepOption {
	return func(s *UninstallFuncStep) {
		s.validateFunc = fn
	}
}

// NewUninstallFuncStep creates a new function-based uninstall step.
func NewUninstallFuncStep(name, description string, executeFunc func(ctx *Context) install.StepResult, opts ...UninstallFuncStepOption) *UninstallFuncStep {
	s := &UninstallFuncStep{
		BaseStep:    install.NewBaseStep(name, description, false),
		executeFunc: executeFunc,
	}
	for _, opt := range opts {
		opt(s)
	}
	// Update CanRollback based on whether a rollback function was provided
	if s.rollbackFunc != nil {
		s.BaseStep = install.NewBaseStep(name, description, true)
	}
	return s
}

// Execute runs the step's execute function.
func (s *UninstallFuncStep) Execute(ctx *install.Context) install.StepResult {
	if s.executeFunc == nil {
		return install.NewStepResult(install.StepStatusFailed, "no execute function defined")
	}

	// Convert install.Context to our Context for the execute function
	uninstallCtx := contextFromInstallContext(ctx)

	start := time.Now()
	result := s.executeFunc(uninstallCtx)
	result.Duration = time.Since(start)
	result.CanRollback = s.CanRollback()

	return result
}

// Rollback runs the step's rollback function if defined.
func (s *UninstallFuncStep) Rollback(ctx *install.Context) error {
	if s.rollbackFunc == nil {
		return nil
	}
	uninstallCtx := contextFromInstallContext(ctx)
	return s.rollbackFunc(uninstallCtx)
}

// Validate runs the step's validate function if defined.
func (s *UninstallFuncStep) Validate(ctx *install.Context) error {
	if s.validateFunc == nil {
		return nil
	}
	uninstallCtx := contextFromInstallContext(ctx)
	return s.validateFunc(uninstallCtx)
}

// contextFromInstallContext creates a minimal Context wrapper from an install.Context.
// This is a helper for bridging the two context types.
//
// Note: Uninstall-specific fields (Force, KeepConfig, InstalledDriver, InstalledPackages)
// are not transferred as they don't exist on install.Context. For full context access,
// use the workflow's Execute method which properly manages both contexts and syncs state.
func contextFromInstallContext(installCtx *install.Context) *Context {
	if installCtx == nil {
		return NewUninstallContext()
	}
	// Create a new uninstall context with matching fields
	return NewUninstallContext(
		WithUninstallDistroInfo(installCtx.DistroInfo),
		WithUninstallPackageManager(installCtx.PackageManager),
		WithUninstallExecutor(installCtx.Executor),
		WithUninstallPrivilege(installCtx.Privilege),
		WithUninstallLogger(installCtx.Logger),
		WithUninstallDryRun(installCtx.DryRun),
	)
}

// SkipUninstallStep creates a step result indicating the step was skipped.
func SkipUninstallStep(reason string) install.StepResult {
	return install.SkipStep(reason)
}

// CompleteUninstallStep creates a step result indicating successful completion.
func CompleteUninstallStep(message string) install.StepResult {
	return install.CompleteStep(message)
}

// FailUninstallStep creates a step result indicating failure.
func FailUninstallStep(message string, err error) install.StepResult {
	return install.FailStep(message, err)
}

// Ensure UninstallFuncStep implements install.Step.
var _ install.Step = (*UninstallFuncStep)(nil)
