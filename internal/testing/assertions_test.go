package testing

import (
	"context"
	"path/filepath"
	stdtesting "testing"

	"github.com/tungetti/igor/internal/errors"
	"github.com/tungetti/igor/internal/install"
	"github.com/tungetti/igor/internal/logging"
	"github.com/tungetti/igor/internal/pkg"
)

// ============================================================================
// Error Assertion Tests
// ============================================================================

func TestAssertErrorCode_Passes(t *stdtesting.T) {
	err := errors.New(errors.GPUDetection, "GPU not found")

	// This should pass without error
	AssertErrorCode(t, err, errors.GPUDetection)
}

func TestAssertErrorContains_Passes(t *stdtesting.T) {
	err := errors.New(errors.GPUDetection, "NVIDIA GPU not found on system")

	// This should pass without error
	AssertErrorContains(t, err, "NVIDIA")
	AssertErrorContains(t, err, "not found")
}

func TestAssertNoError_PassesWithNil(t *stdtesting.T) {
	// This should pass without error
	AssertNoError(t, nil)
}

func TestAssertError_PassesWithError(t *stdtesting.T) {
	// This should pass without error
	AssertError(t, errors.New(errors.Unknown, "some error"))
}

// ============================================================================
// Step Result Assertion Tests
// ============================================================================

func TestAssertStepSuccess_Passes(t *stdtesting.T) {
	result := install.NewStepResult(install.StepStatusCompleted, "Step completed")

	// This should pass without error
	AssertStepSuccess(t, result)
}

func TestAssertStepFailed_Passes(t *stdtesting.T) {
	result := install.NewStepResult(install.StepStatusFailed, "Step failed")

	// This should pass without error
	AssertStepFailed(t, result)
}

func TestAssertStepSkipped_Passes(t *stdtesting.T) {
	result := install.NewStepResult(install.StepStatusSkipped, "Step skipped")

	// This should pass without error
	AssertStepSkipped(t, result)
}

func TestAssertStepStatus_Passes(t *stdtesting.T) {
	testCases := []struct {
		status   install.StepStatus
		expected install.StepStatus
	}{
		{install.StepStatusCompleted, install.StepStatusCompleted},
		{install.StepStatusFailed, install.StepStatusFailed},
		{install.StepStatusSkipped, install.StepStatusSkipped},
		{install.StepStatusPending, install.StepStatusPending},
		{install.StepStatusRunning, install.StepStatusRunning},
	}

	for _, tc := range testCases {
		t.Run(tc.status.String(), func(t *stdtesting.T) {
			result := install.NewStepResult(tc.status, "test")
			AssertStepStatus(t, result, tc.expected)
		})
	}
}

func TestAssertStepMessage_Passes(t *stdtesting.T) {
	result := install.NewStepResult(install.StepStatusCompleted, "Installation completed successfully")

	AssertStepMessage(t, result, "completed")
	AssertStepMessage(t, result, "successfully")
}

// ============================================================================
// Package Manager Assertion Tests
// ============================================================================

func TestAssertPackageInstalled_Passes(t *stdtesting.T) {
	pm := NewMockPackageManager()
	ctx := context.Background()

	_ = pm.Install(ctx, pkg.DefaultInstallOptions(), "nvidia-driver-535")

	// This should pass without error
	AssertPackageInstalled(t, pm, "nvidia-driver-535")
}

func TestAssertPackageRemoved_Passes(t *stdtesting.T) {
	pm := NewMockPackageManager()
	ctx := context.Background()

	_ = pm.Remove(ctx, pkg.DefaultRemoveOptions(), "old-package")

	// This should pass without error
	AssertPackageRemoved(t, pm, "old-package")
}

func TestAssertPackageNotInstalled_Passes(t *stdtesting.T) {
	pm := NewMockPackageManager()
	ctx := context.Background()

	_ = pm.Install(ctx, pkg.DefaultInstallOptions(), "nvidia-driver-535")

	// This should pass without error (different package)
	AssertPackageNotInstalled(t, pm, "nvidia-driver-545")
}

func TestAssertPackageNotRemoved_Passes(t *stdtesting.T) {
	pm := NewMockPackageManager()
	ctx := context.Background()

	_ = pm.Remove(ctx, pkg.DefaultRemoveOptions(), "old-package")

	// This should pass without error (different package)
	AssertPackageNotRemoved(t, pm, "other-package")
}

func TestAssertInstallCallCount_Passes(t *stdtesting.T) {
	pm := NewMockPackageManager()
	ctx := context.Background()

	_ = pm.Install(ctx, pkg.DefaultInstallOptions(), "pkg1")
	_ = pm.Install(ctx, pkg.DefaultInstallOptions(), "pkg2")

	// This should pass without error
	AssertInstallCallCount(t, pm, 2)
}

func TestAssertRemoveCallCount_Passes(t *stdtesting.T) {
	pm := NewMockPackageManager()
	ctx := context.Background()

	_ = pm.Remove(ctx, pkg.DefaultRemoveOptions(), "pkg1")

	// This should pass without error
	AssertRemoveCallCount(t, pm, 1)
}

// ============================================================================
// Executor Assertion Tests
// ============================================================================

func TestAssertCommandCalled_Passes(t *stdtesting.T) {
	executor := NewExecutorBuilder().
		WithDefaultSuccess().
		Build()

	executor.Execute(context.Background(), "nvidia-smi")

	// This should pass without error
	AssertCommandCalled(t, executor, "nvidia-smi")
}

func TestAssertCommandNotCalled_Passes(t *stdtesting.T) {
	executor := NewExecutorBuilder().
		WithDefaultSuccess().
		Build()

	executor.Execute(context.Background(), "nvidia-smi")

	// This should pass without error (different command)
	AssertCommandNotCalled(t, executor, "lspci")
}

func TestAssertCommandCalledWith_Passes(t *stdtesting.T) {
	executor := NewExecutorBuilder().
		WithDefaultSuccess().
		Build()

	executor.Execute(context.Background(), "modprobe", "nvidia")

	// This should pass without error
	AssertCommandCalledWith(t, executor, "modprobe", "nvidia")
}

func TestAssertCallCount_Passes(t *stdtesting.T) {
	executor := NewExecutorBuilder().
		WithDefaultSuccess().
		Build()

	executor.Execute(context.Background(), "cmd1")
	executor.Execute(context.Background(), "cmd2")
	executor.Execute(context.Background(), "cmd3")

	// This should pass without error
	AssertCallCount(t, executor, 3)
}

// ============================================================================
// Logger Assertion Tests
// ============================================================================

func TestAssertLogContains_Passes(t *stdtesting.T) {
	logger := NewMockLogger()

	logger.Info("Installing NVIDIA driver version 535")

	// This should pass without error
	AssertLogContains(t, logger, "NVIDIA")
	AssertLogContains(t, logger, "535")
}

func TestAssertLogNotContains_Passes(t *stdtesting.T) {
	logger := NewMockLogger()

	logger.Info("Installing driver")

	// This should pass without error
	AssertLogNotContains(t, logger, "error")
}

func TestAssertLogLevel_Passes(t *stdtesting.T) {
	logger := NewMockLogger()

	logger.Debug("debug message")
	logger.Info("info message")
	logger.Warn("warn message")
	logger.Error("error message")

	// This should pass without error
	AssertLogLevel(t, logger, logging.LevelDebug, "debug")
	AssertLogLevel(t, logger, logging.LevelInfo, "info")
	AssertLogLevel(t, logger, logging.LevelWarn, "warn")
	AssertLogLevel(t, logger, logging.LevelError, "error")
}

func TestAssertLogEmpty_Passes(t *stdtesting.T) {
	logger := NewMockLogger()

	// This should pass without error (no messages logged)
	AssertLogEmpty(t, logger)
}

func TestAssertLogCount_Passes(t *stdtesting.T) {
	logger := NewMockLogger()

	logger.Info("message 1")
	logger.Info("message 2")
	logger.Info("message 3")

	// This should pass without error
	AssertLogCount(t, logger, 3)
}

// ============================================================================
// File System Assertion Tests
// ============================================================================

func TestAssertFileExists_Passes(t *stdtesting.T) {
	path, cleanup := TempFile(t, "test content")
	defer cleanup()

	// This should pass without error
	AssertFileExists(t, path)
}

func TestAssertFileNotExists_Passes(t *stdtesting.T) {
	// This should pass without error
	AssertFileNotExists(t, "/nonexistent/path/to/file.txt")
}

func TestAssertFileContains_Passes(t *stdtesting.T) {
	path, cleanup := TempFile(t, "hello world content")
	defer cleanup()

	// This should pass without error
	AssertFileContains(t, path, "hello")
	AssertFileContains(t, path, "world")
}

func TestAssertFileNotContains_Passes(t *stdtesting.T) {
	path, cleanup := TempFile(t, "hello world")
	defer cleanup()

	// This should pass without error
	AssertFileNotContains(t, path, "goodbye")
}

func TestAssertFileEquals_Passes(t *stdtesting.T) {
	content := "exact content"
	path, cleanup := TempFile(t, content)
	defer cleanup()

	// This should pass without error
	AssertFileEquals(t, path, content)
}

func TestAssertDirExists_Passes(t *stdtesting.T) {
	dir, cleanup := TempDir(t)
	defer cleanup()

	// This should pass without error
	AssertDirExists(t, dir)
}

func TestAssertDirNotExists_Passes(t *stdtesting.T) {
	// This should pass without error
	AssertDirNotExists(t, "/nonexistent/directory/path")
}

// ============================================================================
// Value Assertion Tests
// ============================================================================

func TestAssertEqual_Passes(t *stdtesting.T) {
	AssertEqual(t, 42, 42)
	AssertEqual(t, "hello", "hello")
	AssertEqual(t, true, true)
}

func TestAssertNotEqual_Passes(t *stdtesting.T) {
	AssertNotEqual(t, 42, 43)
	AssertNotEqual(t, "hello", "world")
}

func TestAssertTrue_Passes(t *stdtesting.T) {
	AssertTrue(t, true)
	AssertTrue(t, 1 == 1)
}

func TestAssertFalse_Passes(t *stdtesting.T) {
	AssertFalse(t, false)
	AssertFalse(t, 1 == 2)
}

func TestAssertNil_Passes(t *stdtesting.T) {
	// Test with actual nil - not a typed nil pointer
	AssertNil(t, nil)
}

func TestAssertNotNil_Passes(t *stdtesting.T) {
	value := 42
	AssertNotNil(t, &value)
}

func TestAssertLen_Passes(t *stdtesting.T) {
	AssertLen(t, "hello", 5)
	AssertLen(t, []string{"a", "b", "c"}, 3)
	AssertLen(t, []int{1, 2}, 2)
}

func TestAssertContains_Passes(t *stdtesting.T) {
	AssertContains(t, "hello world", "hello")
	AssertContains(t, "hello world", "world")
}

func TestAssertNotContains_Passes(t *stdtesting.T) {
	AssertNotContains(t, "hello world", "goodbye")
}

func TestAssertEmpty_Passes(t *stdtesting.T) {
	AssertEmpty(t, "")
	AssertEmpty(t, []string{})
}

func TestAssertNotEmpty_Passes(t *stdtesting.T) {
	AssertNotEmpty(t, "hello")
	AssertNotEmpty(t, []string{"item"})
}

// ============================================================================
// Integration Tests - Assertions in realistic scenarios
// ============================================================================

func TestAssertions_GPUDetectionScenario(t *stdtesting.T) {
	logger := NewMockLogger()
	executor := NewExecutorBuilder().
		WithCommandSuccess("lspci", LspciOutputWithNvidiaGPU()).
		WithCommandSuccess("nvidia-smi", NvidiaSMIOutput()).
		Build()

	// Simulate GPU detection
	result := executor.Execute(context.Background(), "lspci", "-v")
	AssertNoError(t, result.Error)
	AssertCommandCalled(t, executor, "lspci")

	logger.Info("detected NVIDIA GPU", "card", "RTX 3080")
	AssertLogContains(t, logger, "NVIDIA")
}

func TestAssertions_PackageInstallScenario(t *stdtesting.T) {
	pm := NewMockPackageManager()
	logger := NewMockLogger()
	ctx := context.Background()

	// Simulate package installation
	packages := []string{"nvidia-driver-535", "nvidia-utils-535", "libnvidia-gl-535"}

	for _, pkgName := range packages {
		err := pm.Install(ctx, pkg.DefaultInstallOptions(), pkgName)
		AssertNoError(t, err)
		logger.Info("installed package", "name", pkgName)
	}

	// Verify all packages were installed
	for _, pkgName := range packages {
		AssertPackageInstalled(t, pm, pkgName)
	}

	AssertInstallCallCount(t, pm, 3)
	AssertLogCount(t, logger, 3)
}

func TestAssertions_StepExecutionScenario(t *stdtesting.T) {
	logger := NewMockLogger()

	// Simulate step execution
	steps := []struct {
		name   string
		status install.StepStatus
	}{
		{"validate", install.StepStatusCompleted},
		{"backup", install.StepStatusSkipped},
		{"install", install.StepStatusCompleted},
		{"configure", install.StepStatusCompleted},
	}

	for _, step := range steps {
		result := install.NewStepResult(step.status, step.name+" done")
		logger.Info("step completed", "name", step.name)

		if step.status == install.StepStatusCompleted {
			AssertStepSuccess(t, result)
		} else {
			AssertStepSkipped(t, result)
		}
	}

	AssertLogCount(t, logger, len(steps))
}

func TestAssertions_FileOperationsScenario(t *stdtesting.T) {
	// Create a temp directory with config files
	builder := NewTempDirBuilder().
		WithFile("etc/modprobe.d/blacklist-nouveau.conf", NouveauBlacklistContent()).
		WithFile("etc/X11/xorg.conf.d/10-nvidia.conf", XorgConfNvidia())

	dir, cleanup := builder.Build(t)
	defer cleanup()

	// Verify files were created correctly
	blacklistPath := filepath.Join(dir, "etc/modprobe.d/blacklist-nouveau.conf")
	AssertFileExists(t, blacklistPath)
	AssertFileContains(t, blacklistPath, "blacklist nouveau")

	xorgPath := filepath.Join(dir, "etc/X11/xorg.conf.d/10-nvidia.conf")
	AssertFileExists(t, xorgPath)
	AssertFileContains(t, xorgPath, "nvidia")

	// Verify directory structure
	AssertDirExists(t, filepath.Join(dir, "etc"))
	AssertDirExists(t, filepath.Join(dir, "etc/modprobe.d"))
	AssertDirExists(t, filepath.Join(dir, "etc/X11/xorg.conf.d"))
}

func TestAssertions_ErrorHandlingScenario(t *stdtesting.T) {
	// Test various error scenarios
	err := errors.New(errors.GPUDetection, "no NVIDIA GPU found")
	AssertError(t, err)
	AssertErrorCode(t, err, errors.GPUDetection)
	AssertErrorContains(t, err, "NVIDIA")

	// Test wrapped error
	wrappedErr := errors.Wrap(errors.Installation, "installation failed", err)
	AssertErrorCode(t, wrappedErr, errors.Installation)
}

// ============================================================================
// Additional Assertion Tests
// ============================================================================

func TestAssertErrorIs_Passes(t *stdtesting.T) {
	err := errors.New(errors.GPUDetection, "GPU not found")
	target := errors.New(errors.GPUDetection, "other GPU error")

	// Both have the same error code
	AssertErrorIs(t, err, target)
}

func TestAssertEmpty_PassesWithMap(t *stdtesting.T) {
	AssertEmpty(t, map[string]interface{}{})
}

func TestAssertNotEmpty_PassesWithMap(t *stdtesting.T) {
	AssertNotEmpty(t, map[string]interface{}{"key": "value"})
}

func TestAssertLen_PassesWithIntSlice(t *stdtesting.T) {
	AssertLen(t, []int{1, 2, 3, 4, 5}, 5)
}

func TestAssertLen_PassesWithMap(t *stdtesting.T) {
	AssertLen(t, map[string]string{"a": "1", "b": "2"}, 2)
}

func TestAssertLen_PassesWithInterfaceMap(t *stdtesting.T) {
	AssertLen(t, map[string]interface{}{"a": 1}, 1)
}

func TestAssertLen_PassesWithInterfaceSlice(t *stdtesting.T) {
	AssertLen(t, []interface{}{1, "two", 3.0}, 3)
}

func TestAssertEmpty_PassesWithInterfaceSlice(t *stdtesting.T) {
	AssertEmpty(t, []interface{}{})
}

func TestAssertEmpty_PassesWithNil(t *stdtesting.T) {
	AssertEmpty(t, nil)
}

func TestAssertNotEmpty_PassesWithInterfaceSlice(t *stdtesting.T) {
	AssertNotEmpty(t, []interface{}{1})
}

// ============================================================================
// Additional Value Assertion Tests
// ============================================================================

func TestAssertEqual_WithMessage(t *stdtesting.T) {
	AssertEqual(t, 42, 42, "values should be equal")
}

func TestAssertNotEqual_WithMessage(t *stdtesting.T) {
	AssertNotEqual(t, 42, 43, "values should be different")
}

func TestAssertTrue_WithMessage(t *stdtesting.T) {
	AssertTrue(t, true, "should be true")
}

func TestAssertFalse_WithMessage(t *stdtesting.T) {
	AssertFalse(t, false, "should be false")
}

func TestAssertNil_WithMessage(t *stdtesting.T) {
	AssertNil(t, nil, "should be nil")
}

func TestAssertNotNil_WithMessage(t *stdtesting.T) {
	value := "test"
	AssertNotNil(t, value, "should not be nil")
}

func TestAssertContains_WithMessage(t *stdtesting.T) {
	AssertContains(t, "hello world", "hello", "should contain hello")
}

func TestAssertNotContains_WithMessage(t *stdtesting.T) {
	AssertNotContains(t, "hello world", "goodbye", "should not contain goodbye")
}

func TestAssertNoError_WithMessage(t *stdtesting.T) {
	AssertNoError(t, nil, "operation should succeed")
}

func TestAssertError_WithMessage(t *stdtesting.T) {
	err := errors.New(errors.Unknown, "test error")
	AssertError(t, err, "should have error")
}

func TestAssertEmpty_WithMessage(t *stdtesting.T) {
	AssertEmpty(t, "", "string should be empty")
}

func TestAssertNotEmpty_WithMessage(t *stdtesting.T) {
	AssertNotEmpty(t, "content", "string should not be empty")
}

func TestAssertLen_WithMessage(t *stdtesting.T) {
	AssertLen(t, "hello", 5, "should have length 5")
}

// Helper functions are defined using pkg package types directly in the tests
