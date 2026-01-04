package install

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestStepStatus_String(t *testing.T) {
	tests := []struct {
		status   StepStatus
		expected string
	}{
		{StepStatusPending, "pending"},
		{StepStatusRunning, "running"},
		{StepStatusCompleted, "completed"},
		{StepStatusFailed, "failed"},
		{StepStatusSkipped, "skipped"},
		{StepStatusRolledBack, "rolled_back"},
		{StepStatus(99), "unknown(99)"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.status.String())
		})
	}
}

func TestStepStatus_IsTerminal(t *testing.T) {
	tests := []struct {
		status   StepStatus
		terminal bool
	}{
		{StepStatusPending, false},
		{StepStatusRunning, false},
		{StepStatusCompleted, true},
		{StepStatusFailed, true},
		{StepStatusSkipped, true},
		{StepStatusRolledBack, true},
	}

	for _, tt := range tests {
		t.Run(tt.status.String(), func(t *testing.T) {
			assert.Equal(t, tt.terminal, tt.status.IsTerminal())
		})
	}
}

func TestStepStatus_IsSuccess(t *testing.T) {
	tests := []struct {
		status  StepStatus
		success bool
	}{
		{StepStatusPending, false},
		{StepStatusRunning, false},
		{StepStatusCompleted, true},
		{StepStatusFailed, false},
		{StepStatusSkipped, true},
		{StepStatusRolledBack, false},
	}

	for _, tt := range tests {
		t.Run(tt.status.String(), func(t *testing.T) {
			assert.Equal(t, tt.success, tt.status.IsSuccess())
		})
	}
}

func TestStepResult_New(t *testing.T) {
	result := NewStepResult(StepStatusCompleted, "step completed")

	assert.Equal(t, StepStatusCompleted, result.Status)
	assert.Equal(t, "step completed", result.Message)
	assert.Nil(t, result.Error)
	assert.Zero(t, result.Duration)
	assert.False(t, result.CanRollback)
}

func TestStepResult_WithError(t *testing.T) {
	err := errors.New("test error")
	result := NewStepResult(StepStatusFailed, "step failed").WithError(err)

	assert.Equal(t, StepStatusFailed, result.Status)
	assert.Equal(t, err, result.Error)
}

func TestStepResult_WithDuration(t *testing.T) {
	duration := 5 * time.Second
	result := NewStepResult(StepStatusCompleted, "done").WithDuration(duration)

	assert.Equal(t, duration, result.Duration)
}

func TestStepResult_WithCanRollback(t *testing.T) {
	result := NewStepResult(StepStatusCompleted, "done").WithCanRollback(true)

	assert.True(t, result.CanRollback)
}

func TestStepResult_ChainedMethods(t *testing.T) {
	err := errors.New("test error")
	duration := 3 * time.Second

	result := NewStepResult(StepStatusFailed, "failed").
		WithError(err).
		WithDuration(duration).
		WithCanRollback(true)

	assert.Equal(t, StepStatusFailed, result.Status)
	assert.Equal(t, "failed", result.Message)
	assert.Equal(t, err, result.Error)
	assert.Equal(t, duration, result.Duration)
	assert.True(t, result.CanRollback)
}

func TestStepResult_IsSuccess(t *testing.T) {
	assert.True(t, NewStepResult(StepStatusCompleted, "").IsSuccess())
	assert.True(t, NewStepResult(StepStatusSkipped, "").IsSuccess())
	assert.False(t, NewStepResult(StepStatusFailed, "").IsSuccess())
	assert.False(t, NewStepResult(StepStatusPending, "").IsSuccess())
}

func TestStepResult_IsFailure(t *testing.T) {
	assert.True(t, NewStepResult(StepStatusFailed, "").IsFailure())
	assert.False(t, NewStepResult(StepStatusCompleted, "").IsFailure())
	assert.False(t, NewStepResult(StepStatusSkipped, "").IsFailure())
}

func TestStepResult_String(t *testing.T) {
	t.Run("without error", func(t *testing.T) {
		result := NewStepResult(StepStatusCompleted, "step done")
		assert.Equal(t, "completed: step done", result.String())
	})

	t.Run("with error", func(t *testing.T) {
		err := errors.New("something went wrong")
		result := NewStepResult(StepStatusFailed, "step failed").WithError(err)
		assert.Contains(t, result.String(), "failed")
		assert.Contains(t, result.String(), "step failed")
		assert.Contains(t, result.String(), "something went wrong")
	})
}

func TestStepProgress_New(t *testing.T) {
	progress := NewStepProgress("install", 2, 5, "Installing packages")

	assert.Equal(t, "install", progress.StepName)
	assert.Equal(t, 2, progress.StepIndex)
	assert.Equal(t, 5, progress.TotalSteps)
	assert.Equal(t, 40.0, progress.Percent) // 2/5 * 100
	assert.Equal(t, "Installing packages", progress.Message)
}

func TestStepProgress_New_ZeroTotal(t *testing.T) {
	progress := NewStepProgress("install", 0, 0, "message")

	assert.Equal(t, 0.0, progress.Percent)
}

func TestStepProgress_String(t *testing.T) {
	progress := NewStepProgress("install", 1, 3, "Installing")

	str := progress.String()
	assert.Contains(t, str, "[2/3]") // 1-based display
	assert.Contains(t, str, "install")
	assert.Contains(t, str, "Installing")
}

func TestWorkflowStatus_String(t *testing.T) {
	tests := []struct {
		status   WorkflowStatus
		expected string
	}{
		{WorkflowStatusPending, "pending"},
		{WorkflowStatusRunning, "running"},
		{WorkflowStatusCompleted, "completed"},
		{WorkflowStatusFailed, "failed"},
		{WorkflowStatusCancelled, "cancelled"},
		{WorkflowStatusRollingBack, "rolling_back"},
		{WorkflowStatusRolledBack, "rolled_back"},
		{WorkflowStatus(99), "unknown(99)"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.status.String())
		})
	}
}

func TestWorkflowStatus_IsTerminal(t *testing.T) {
	tests := []struct {
		status   WorkflowStatus
		terminal bool
	}{
		{WorkflowStatusPending, false},
		{WorkflowStatusRunning, false},
		{WorkflowStatusCompleted, true},
		{WorkflowStatusFailed, true},
		{WorkflowStatusCancelled, true},
		{WorkflowStatusRollingBack, false},
		{WorkflowStatusRolledBack, true},
	}

	for _, tt := range tests {
		t.Run(tt.status.String(), func(t *testing.T) {
			assert.Equal(t, tt.terminal, tt.status.IsTerminal())
		})
	}
}

func TestWorkflowStatus_IsSuccess(t *testing.T) {
	tests := []struct {
		status  WorkflowStatus
		success bool
	}{
		{WorkflowStatusPending, false},
		{WorkflowStatusRunning, false},
		{WorkflowStatusCompleted, true},
		{WorkflowStatusFailed, false},
		{WorkflowStatusCancelled, false},
		{WorkflowStatusRollingBack, false},
		{WorkflowStatusRolledBack, false},
	}

	for _, tt := range tests {
		t.Run(tt.status.String(), func(t *testing.T) {
			assert.Equal(t, tt.success, tt.status.IsSuccess())
		})
	}
}

func TestWorkflowResult_New(t *testing.T) {
	result := NewWorkflowResult(WorkflowStatusPending)

	assert.Equal(t, WorkflowStatusPending, result.Status)
	assert.NotNil(t, result.CompletedSteps)
	assert.Empty(t, result.CompletedSteps)
	assert.Empty(t, result.FailedStep)
	assert.Nil(t, result.Error)
	assert.Zero(t, result.TotalDuration)
	assert.False(t, result.NeedsReboot)
}

func TestWorkflowResult_WithError(t *testing.T) {
	err := errors.New("test error")
	result := NewWorkflowResult(WorkflowStatusFailed).WithError("install-packages", err)

	assert.Equal(t, "install-packages", result.FailedStep)
	assert.Equal(t, err, result.Error)
}

func TestWorkflowResult_WithDuration(t *testing.T) {
	duration := 10 * time.Minute
	result := NewWorkflowResult(WorkflowStatusCompleted).WithDuration(duration)

	assert.Equal(t, duration, result.TotalDuration)
}

func TestWorkflowResult_WithNeedsReboot(t *testing.T) {
	result := NewWorkflowResult(WorkflowStatusCompleted).WithNeedsReboot(true)

	assert.True(t, result.NeedsReboot)
}

func TestWorkflowResult_AddCompletedStep(t *testing.T) {
	result := NewWorkflowResult(WorkflowStatusRunning)

	result.AddCompletedStep("step1")
	result.AddCompletedStep("step2")
	result.AddCompletedStep("step3")

	assert.Equal(t, []string{"step1", "step2", "step3"}, result.CompletedSteps)
}

func TestWorkflowResult_IsSuccess(t *testing.T) {
	assert.True(t, NewWorkflowResult(WorkflowStatusCompleted).IsSuccess())
	assert.False(t, NewWorkflowResult(WorkflowStatusFailed).IsSuccess())
	assert.False(t, NewWorkflowResult(WorkflowStatusCancelled).IsSuccess())
}

func TestWorkflowResult_IsFailure(t *testing.T) {
	assert.True(t, NewWorkflowResult(WorkflowStatusFailed).IsFailure())
	assert.False(t, NewWorkflowResult(WorkflowStatusCompleted).IsFailure())
	assert.False(t, NewWorkflowResult(WorkflowStatusCancelled).IsFailure())
}

func TestWorkflowResult_String(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		result := NewWorkflowResult(WorkflowStatusCompleted).WithDuration(5 * time.Second)
		result.AddCompletedStep("step1")
		result.AddCompletedStep("step2")

		str := result.String()
		assert.Contains(t, str, "completed")
		assert.Contains(t, str, "2 steps")
	})

	t.Run("failure", func(t *testing.T) {
		err := errors.New("installation failed")
		result := NewWorkflowResult(WorkflowStatusFailed).WithError("install", err)

		str := result.String()
		assert.Contains(t, str, "failed")
		assert.Contains(t, str, "install")
		assert.Contains(t, str, "installation failed")
	})
}

func TestWorkflowResult_ChainedMethods(t *testing.T) {
	err := errors.New("test error")
	result := NewWorkflowResult(WorkflowStatusFailed).
		WithError("step1", err).
		WithDuration(30 * time.Second).
		WithNeedsReboot(true)

	assert.Equal(t, WorkflowStatusFailed, result.Status)
	assert.Equal(t, "step1", result.FailedStep)
	assert.Equal(t, err, result.Error)
	assert.Equal(t, 30*time.Second, result.TotalDuration)
	assert.True(t, result.NeedsReboot)
}
