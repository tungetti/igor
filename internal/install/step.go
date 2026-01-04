package install

import (
	"time"
)

// Step represents a single installation step in a workflow.
// Steps are executed in order and can be rolled back if they support it.
type Step interface {
	// Name returns the unique name of the step.
	Name() string

	// Description returns a human-readable description of what the step does.
	Description() string

	// Execute runs the step with the given context.
	// Returns a StepResult containing the outcome.
	Execute(ctx *Context) StepResult

	// Rollback reverses the step if possible.
	// This is called when a later step fails and rollback is needed.
	Rollback(ctx *Context) error

	// CanRollback returns whether this step supports rollback.
	CanRollback() bool

	// Validate checks if the step can be executed with the given context.
	// Returns an error if the step cannot be executed.
	Validate(ctx *Context) error
}

// BaseStep provides common functionality for steps.
// It should be embedded in concrete step implementations.
type BaseStep struct {
	name        string
	description string
	canRollback bool
}

// NewBaseStep creates a new base step with the given name, description, and rollback capability.
func NewBaseStep(name, description string, canRollback bool) BaseStep {
	return BaseStep{
		name:        name,
		description: description,
		canRollback: canRollback,
	}
}

// Name returns the step name.
func (s BaseStep) Name() string {
	return s.name
}

// Description returns the step description.
func (s BaseStep) Description() string {
	return s.description
}

// CanRollback returns whether this step supports rollback.
func (s BaseStep) CanRollback() bool {
	return s.canRollback
}

// FuncStep is a step implementation that uses functions for execution and rollback.
// This is useful for simple steps that don't need a full struct implementation.
type FuncStep struct {
	BaseStep
	executeFunc  func(ctx *Context) StepResult
	rollbackFunc func(ctx *Context) error
	validateFunc func(ctx *Context) error
}

// FuncStepOption is a functional option for FuncStep.
type FuncStepOption func(*FuncStep)

// WithRollbackFunc sets the rollback function for a FuncStep.
func WithRollbackFunc(fn func(ctx *Context) error) FuncStepOption {
	return func(s *FuncStep) {
		s.rollbackFunc = fn
		s.canRollback = fn != nil
	}
}

// WithValidateFunc sets the validate function for a FuncStep.
func WithValidateFunc(fn func(ctx *Context) error) FuncStepOption {
	return func(s *FuncStep) {
		s.validateFunc = fn
	}
}

// NewFuncStep creates a new function-based step.
func NewFuncStep(name, description string, executeFunc func(ctx *Context) StepResult, opts ...FuncStepOption) *FuncStep {
	s := &FuncStep{
		BaseStep:    NewBaseStep(name, description, false),
		executeFunc: executeFunc,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Execute runs the step's execute function.
func (s *FuncStep) Execute(ctx *Context) StepResult {
	if s.executeFunc == nil {
		return NewStepResult(StepStatusFailed, "no execute function defined")
	}

	start := time.Now()
	result := s.executeFunc(ctx)
	result.Duration = time.Since(start)
	result.CanRollback = s.canRollback

	return result
}

// Rollback runs the step's rollback function if defined.
func (s *FuncStep) Rollback(ctx *Context) error {
	if s.rollbackFunc == nil {
		return nil
	}
	return s.rollbackFunc(ctx)
}

// Validate runs the step's validate function if defined.
func (s *FuncStep) Validate(ctx *Context) error {
	if s.validateFunc == nil {
		return nil
	}
	return s.validateFunc(ctx)
}

// SkipStep creates a step result indicating the step was skipped.
func SkipStep(reason string) StepResult {
	return NewStepResult(StepStatusSkipped, reason)
}

// CompleteStep creates a step result indicating successful completion.
func CompleteStep(message string) StepResult {
	return NewStepResult(StepStatusCompleted, message)
}

// FailStep creates a step result indicating failure.
func FailStep(message string, err error) StepResult {
	return NewStepResult(StepStatusFailed, message).WithError(err)
}

// Ensure FuncStep implements Step.
var _ Step = (*FuncStep)(nil)
