package uninstall

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestUninstallStatus_String(t *testing.T) {
	tests := []struct {
		status   UninstallStatus
		expected string
	}{
		{UninstallStatusPending, "pending"},
		{UninstallStatusRunning, "running"},
		{UninstallStatusCompleted, "completed"},
		{UninstallStatusPartial, "partial"},
		{UninstallStatusFailed, "failed"},
		{UninstallStatusCancelled, "cancelled"},
		{UninstallStatus(99), "unknown(99)"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.status.String())
		})
	}
}

func TestUninstallStatus_IsTerminal(t *testing.T) {
	tests := []struct {
		status   UninstallStatus
		terminal bool
	}{
		{UninstallStatusPending, false},
		{UninstallStatusRunning, false},
		{UninstallStatusCompleted, true},
		{UninstallStatusPartial, true},
		{UninstallStatusFailed, true},
		{UninstallStatusCancelled, true},
	}

	for _, tt := range tests {
		t.Run(tt.status.String(), func(t *testing.T) {
			assert.Equal(t, tt.terminal, tt.status.IsTerminal())
		})
	}
}

func TestUninstallStatus_IsSuccess(t *testing.T) {
	tests := []struct {
		status  UninstallStatus
		success bool
	}{
		{UninstallStatusPending, false},
		{UninstallStatusRunning, false},
		{UninstallStatusCompleted, true},
		{UninstallStatusPartial, false},
		{UninstallStatusFailed, false},
		{UninstallStatusCancelled, false},
	}

	for _, tt := range tests {
		t.Run(tt.status.String(), func(t *testing.T) {
			assert.Equal(t, tt.success, tt.status.IsSuccess())
		})
	}
}

func TestUninstallStatus_IsPartial(t *testing.T) {
	tests := []struct {
		status  UninstallStatus
		partial bool
	}{
		{UninstallStatusPending, false},
		{UninstallStatusRunning, false},
		{UninstallStatusCompleted, false},
		{UninstallStatusPartial, true},
		{UninstallStatusFailed, false},
		{UninstallStatusCancelled, false},
	}

	for _, tt := range tests {
		t.Run(tt.status.String(), func(t *testing.T) {
			assert.Equal(t, tt.partial, tt.status.IsPartial())
		})
	}
}

func TestUninstallResult_New(t *testing.T) {
	result := NewUninstallResult(UninstallStatusPending)

	assert.Equal(t, UninstallStatusPending, result.Status)
	assert.NotNil(t, result.RemovedPackages)
	assert.Empty(t, result.RemovedPackages)
	assert.NotNil(t, result.FailedPackages)
	assert.Empty(t, result.FailedPackages)
	assert.NotNil(t, result.CleanedConfigs)
	assert.Empty(t, result.CleanedConfigs)
	assert.NotNil(t, result.CompletedSteps)
	assert.Empty(t, result.CompletedSteps)
	assert.Empty(t, result.FailedStep)
	assert.Nil(t, result.Error)
	assert.Zero(t, result.TotalDuration)
	assert.False(t, result.NeedsReboot)
	assert.False(t, result.NouveauRestored)
}

func TestUninstallResult_WithError(t *testing.T) {
	err := errors.New("test error")
	result := NewUninstallResult(UninstallStatusFailed).WithError("remove-packages", err)

	assert.Equal(t, "remove-packages", result.FailedStep)
	assert.Equal(t, err, result.Error)
}

func TestUninstallResult_WithDuration(t *testing.T) {
	duration := 10 * time.Minute
	result := NewUninstallResult(UninstallStatusCompleted).WithDuration(duration)

	assert.Equal(t, duration, result.TotalDuration)
}

func TestUninstallResult_WithNeedsReboot(t *testing.T) {
	result := NewUninstallResult(UninstallStatusCompleted).WithNeedsReboot(true)

	assert.True(t, result.NeedsReboot)
}

func TestUninstallResult_WithNouveauRestored(t *testing.T) {
	result := NewUninstallResult(UninstallStatusCompleted).WithNouveauRestored(true)

	assert.True(t, result.NouveauRestored)
}

func TestUninstallResult_AddCompletedStep(t *testing.T) {
	result := NewUninstallResult(UninstallStatusRunning)

	result.AddCompletedStep("step1")
	result.AddCompletedStep("step2")
	result.AddCompletedStep("step3")

	assert.Equal(t, []string{"step1", "step2", "step3"}, result.CompletedSteps)
}

func TestUninstallResult_AddRemovedPackage(t *testing.T) {
	result := NewUninstallResult(UninstallStatusRunning)

	result.AddRemovedPackage("nvidia-driver-550")
	result.AddRemovedPackage("nvidia-settings")

	assert.Equal(t, []string{"nvidia-driver-550", "nvidia-settings"}, result.RemovedPackages)
}

func TestUninstallResult_AddFailedPackage(t *testing.T) {
	result := NewUninstallResult(UninstallStatusPartial)

	result.AddFailedPackage("nvidia-cuda-toolkit")
	result.AddFailedPackage("nvidia-cudnn")

	assert.Equal(t, []string{"nvidia-cuda-toolkit", "nvidia-cudnn"}, result.FailedPackages)
}

func TestUninstallResult_AddCleanedConfig(t *testing.T) {
	result := NewUninstallResult(UninstallStatusCompleted)

	result.AddCleanedConfig("/etc/X11/xorg.conf.d/10-nvidia.conf")
	result.AddCleanedConfig("/etc/modprobe.d/nvidia.conf")

	assert.Equal(t, []string{
		"/etc/X11/xorg.conf.d/10-nvidia.conf",
		"/etc/modprobe.d/nvidia.conf",
	}, result.CleanedConfigs)
}

func TestUninstallResult_IsSuccess(t *testing.T) {
	assert.True(t, NewUninstallResult(UninstallStatusCompleted).IsSuccess())
	assert.False(t, NewUninstallResult(UninstallStatusFailed).IsSuccess())
	assert.False(t, NewUninstallResult(UninstallStatusPartial).IsSuccess())
	assert.False(t, NewUninstallResult(UninstallStatusCancelled).IsSuccess())
}

func TestUninstallResult_IsFailure(t *testing.T) {
	assert.True(t, NewUninstallResult(UninstallStatusFailed).IsFailure())
	assert.False(t, NewUninstallResult(UninstallStatusCompleted).IsFailure())
	assert.False(t, NewUninstallResult(UninstallStatusPartial).IsFailure())
	assert.False(t, NewUninstallResult(UninstallStatusCancelled).IsFailure())
}

func TestUninstallResult_IsPartial(t *testing.T) {
	assert.True(t, NewUninstallResult(UninstallStatusPartial).IsPartial())
	assert.False(t, NewUninstallResult(UninstallStatusCompleted).IsPartial())
	assert.False(t, NewUninstallResult(UninstallStatusFailed).IsPartial())
}

func TestUninstallResult_String(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		result := NewUninstallResult(UninstallStatusCompleted).WithDuration(5 * time.Second)
		result.AddRemovedPackage("nvidia-driver")
		result.AddCleanedConfig("/etc/nvidia.conf")

		str := result.String()
		assert.Contains(t, str, "completed")
		assert.Contains(t, str, "1 packages")
		assert.Contains(t, str, "1 configs")
	})

	t.Run("failure", func(t *testing.T) {
		err := errors.New("removal failed")
		result := NewUninstallResult(UninstallStatusFailed).WithError("remove-packages", err)

		str := result.String()
		assert.Contains(t, str, "failed")
		assert.Contains(t, str, "remove-packages")
		assert.Contains(t, str, "removal failed")
	})

	t.Run("partial", func(t *testing.T) {
		result := NewUninstallResult(UninstallStatusPartial).WithDuration(10 * time.Second)
		result.AddRemovedPackage("nvidia-driver")
		result.AddFailedPackage("nvidia-cuda")

		str := result.String()
		assert.Contains(t, str, "partial")
		assert.Contains(t, str, "1 failed")
	})
}

func TestUninstallResult_ChainedMethods(t *testing.T) {
	err := errors.New("test error")
	result := NewUninstallResult(UninstallStatusFailed).
		WithError("step1", err).
		WithDuration(30 * time.Second).
		WithNeedsReboot(true).
		WithNouveauRestored(true)

	assert.Equal(t, UninstallStatusFailed, result.Status)
	assert.Equal(t, "step1", result.FailedStep)
	assert.Equal(t, err, result.Error)
	assert.Equal(t, 30*time.Second, result.TotalDuration)
	assert.True(t, result.NeedsReboot)
	assert.True(t, result.NouveauRestored)
}

func TestStateKeys(t *testing.T) {
	// Verify state key constants are defined
	assert.Equal(t, "packages_removed", StatePackagesRemoved)
	assert.Equal(t, "removed_packages", StateRemovedPackages)
	assert.Equal(t, "configs_cleaned", StateConfigsCleaned)
	assert.Equal(t, "cleaned_configs", StateCleanedConfigs)
	assert.Equal(t, "modules_unloaded", StateModulesUnloaded)
	assert.Equal(t, "nouveau_unblocked", StateNouveauUnblocked)
	assert.Equal(t, "nouveau_restored", StateNouveauRestored)
}
