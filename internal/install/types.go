// Package install provides the installation workflow framework for Igor.
// It defines interfaces and types for creating, executing, and managing
// multi-step installation workflows with rollback support.
package install

import (
	"fmt"
	"time"
)

// StepStatus represents the status of an installation step.
type StepStatus int

const (
	// StepStatusPending indicates the step has not yet been executed.
	StepStatusPending StepStatus = iota
	// StepStatusRunning indicates the step is currently executing.
	StepStatusRunning
	// StepStatusCompleted indicates the step completed successfully.
	StepStatusCompleted
	// StepStatusFailed indicates the step failed during execution.
	StepStatusFailed
	// StepStatusSkipped indicates the step was skipped.
	StepStatusSkipped
	// StepStatusRolledBack indicates the step was rolled back after failure.
	StepStatusRolledBack
)

// String returns the string representation of the step status.
func (s StepStatus) String() string {
	switch s {
	case StepStatusPending:
		return "pending"
	case StepStatusRunning:
		return "running"
	case StepStatusCompleted:
		return "completed"
	case StepStatusFailed:
		return "failed"
	case StepStatusSkipped:
		return "skipped"
	case StepStatusRolledBack:
		return "rolled_back"
	default:
		return fmt.Sprintf("unknown(%d)", int(s))
	}
}

// IsTerminal returns true if this status represents a terminal state.
func (s StepStatus) IsTerminal() bool {
	switch s {
	case StepStatusCompleted, StepStatusFailed, StepStatusSkipped, StepStatusRolledBack:
		return true
	default:
		return false
	}
}

// IsSuccess returns true if this status represents a successful outcome.
func (s StepStatus) IsSuccess() bool {
	return s == StepStatusCompleted || s == StepStatusSkipped
}

// StepResult contains the result of a step execution.
type StepResult struct {
	// Status is the final status of the step.
	Status StepStatus

	// Message is a human-readable message describing the result.
	Message string

	// Error is the error that caused failure, if any.
	Error error

	// Duration is how long the step took to execute.
	Duration time.Duration

	// CanRollback indicates whether this step can be rolled back.
	CanRollback bool
}

// NewStepResult creates a new step result with the given status and message.
func NewStepResult(status StepStatus, message string) StepResult {
	return StepResult{
		Status:  status,
		Message: message,
	}
}

// WithError adds an error to the step result.
func (r StepResult) WithError(err error) StepResult {
	r.Error = err
	return r
}

// WithDuration adds a duration to the step result.
func (r StepResult) WithDuration(d time.Duration) StepResult {
	r.Duration = d
	return r
}

// WithCanRollback sets whether the step can be rolled back.
func (r StepResult) WithCanRollback(canRollback bool) StepResult {
	r.CanRollback = canRollback
	return r
}

// IsSuccess returns true if the step completed successfully.
func (r StepResult) IsSuccess() bool {
	return r.Status.IsSuccess()
}

// IsFailure returns true if the step failed.
func (r StepResult) IsFailure() bool {
	return r.Status == StepStatusFailed
}

// String returns a human-readable representation of the step result.
func (r StepResult) String() string {
	if r.Error != nil {
		return fmt.Sprintf("%s: %s (error: %v)", r.Status, r.Message, r.Error)
	}
	return fmt.Sprintf("%s: %s", r.Status, r.Message)
}

// StepProgress contains progress information for a step.
type StepProgress struct {
	// StepName is the name of the current step.
	StepName string

	// StepIndex is the 0-based index of the current step.
	StepIndex int

	// TotalSteps is the total number of steps in the workflow.
	TotalSteps int

	// Percent is the overall progress percentage (0-100).
	Percent float64

	// Message is a human-readable progress message.
	Message string
}

// NewStepProgress creates a new step progress instance.
func NewStepProgress(name string, index, total int, message string) StepProgress {
	percent := 0.0
	if total > 0 {
		percent = float64(index) / float64(total) * 100
	}
	return StepProgress{
		StepName:   name,
		StepIndex:  index,
		TotalSteps: total,
		Percent:    percent,
		Message:    message,
	}
}

// String returns a human-readable representation of the progress.
func (p StepProgress) String() string {
	return fmt.Sprintf("[%d/%d] %s: %s (%.1f%%)",
		p.StepIndex+1, p.TotalSteps, p.StepName, p.Message, p.Percent)
}

// WorkflowStatus represents the overall workflow status.
type WorkflowStatus int

const (
	// WorkflowStatusPending indicates the workflow has not yet started.
	WorkflowStatusPending WorkflowStatus = iota
	// WorkflowStatusRunning indicates the workflow is currently executing.
	WorkflowStatusRunning
	// WorkflowStatusCompleted indicates the workflow completed successfully.
	WorkflowStatusCompleted
	// WorkflowStatusFailed indicates the workflow failed.
	WorkflowStatusFailed
	// WorkflowStatusCancelled indicates the workflow was cancelled.
	WorkflowStatusCancelled
	// WorkflowStatusRollingBack indicates the workflow is being rolled back.
	WorkflowStatusRollingBack
	// WorkflowStatusRolledBack indicates the workflow was rolled back.
	WorkflowStatusRolledBack
)

// String returns the string representation of the workflow status.
func (s WorkflowStatus) String() string {
	switch s {
	case WorkflowStatusPending:
		return "pending"
	case WorkflowStatusRunning:
		return "running"
	case WorkflowStatusCompleted:
		return "completed"
	case WorkflowStatusFailed:
		return "failed"
	case WorkflowStatusCancelled:
		return "cancelled"
	case WorkflowStatusRollingBack:
		return "rolling_back"
	case WorkflowStatusRolledBack:
		return "rolled_back"
	default:
		return fmt.Sprintf("unknown(%d)", int(s))
	}
}

// IsTerminal returns true if this status represents a terminal state.
func (s WorkflowStatus) IsTerminal() bool {
	switch s {
	case WorkflowStatusCompleted, WorkflowStatusFailed, WorkflowStatusCancelled, WorkflowStatusRolledBack:
		return true
	default:
		return false
	}
}

// IsSuccess returns true if this status represents a successful outcome.
func (s WorkflowStatus) IsSuccess() bool {
	return s == WorkflowStatusCompleted
}

// WorkflowResult contains the result of a workflow execution.
type WorkflowResult struct {
	// Status is the final status of the workflow.
	Status WorkflowStatus

	// CompletedSteps contains the names of successfully completed steps.
	CompletedSteps []string

	// FailedStep is the name of the step that failed, if any.
	FailedStep string

	// Error is the error that caused workflow failure, if any.
	Error error

	// TotalDuration is the total time taken by the workflow.
	TotalDuration time.Duration

	// NeedsReboot indicates whether a system reboot is required.
	NeedsReboot bool
}

// NewWorkflowResult creates a new workflow result with the given status.
func NewWorkflowResult(status WorkflowStatus) WorkflowResult {
	return WorkflowResult{
		Status:         status,
		CompletedSteps: make([]string, 0),
	}
}

// WithError adds an error and failed step to the workflow result.
func (r WorkflowResult) WithError(stepName string, err error) WorkflowResult {
	r.FailedStep = stepName
	r.Error = err
	return r
}

// WithDuration adds a total duration to the workflow result.
func (r WorkflowResult) WithDuration(d time.Duration) WorkflowResult {
	r.TotalDuration = d
	return r
}

// WithNeedsReboot sets whether a reboot is required.
func (r WorkflowResult) WithNeedsReboot(needsReboot bool) WorkflowResult {
	r.NeedsReboot = needsReboot
	return r
}

// AddCompletedStep adds a step name to the list of completed steps.
func (r *WorkflowResult) AddCompletedStep(stepName string) {
	r.CompletedSteps = append(r.CompletedSteps, stepName)
}

// IsSuccess returns true if the workflow completed successfully.
func (r WorkflowResult) IsSuccess() bool {
	return r.Status.IsSuccess()
}

// IsFailure returns true if the workflow failed.
func (r WorkflowResult) IsFailure() bool {
	return r.Status == WorkflowStatusFailed
}

// String returns a human-readable representation of the workflow result.
func (r WorkflowResult) String() string {
	if r.Error != nil {
		return fmt.Sprintf("%s: failed at step '%s' (error: %v)", r.Status, r.FailedStep, r.Error)
	}
	return fmt.Sprintf("%s: completed %d steps in %v", r.Status, len(r.CompletedSteps), r.TotalDuration)
}
