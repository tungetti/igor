package steps

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tungetti/igor/internal/exec"
	"github.com/tungetti/igor/internal/gpu/kernel"
	"github.com/tungetti/igor/internal/install"
)

// =============================================================================
// Mock Kernel Detector for Verification Tests
// =============================================================================

// mockVerificationKernelDetector is a mock kernel detector for verification tests.
type mockVerificationKernelDetector struct {
	moduleLoaded      map[string]bool
	moduleLoadedError error
}

// newMockVerificationKernelDetector creates a new mock kernel detector for verification tests.
func newMockVerificationKernelDetector() *mockVerificationKernelDetector {
	return &mockVerificationKernelDetector{
		moduleLoaded: make(map[string]bool),
	}
}

// SetModuleLoaded sets whether a module is reported as loaded.
func (m *mockVerificationKernelDetector) SetModuleLoaded(name string, loaded bool) {
	m.moduleLoaded[name] = loaded
}

// SetModuleLoadedError sets an error to return from IsModuleLoaded.
func (m *mockVerificationKernelDetector) SetModuleLoadedError(err error) {
	m.moduleLoadedError = err
}

// IsModuleLoaded implements kernel.Detector.
func (m *mockVerificationKernelDetector) IsModuleLoaded(ctx context.Context, name string) (bool, error) {
	if m.moduleLoadedError != nil {
		return false, m.moduleLoadedError
	}
	return m.moduleLoaded[name], nil
}

// GetKernelInfo implements kernel.Detector.
func (m *mockVerificationKernelDetector) GetKernelInfo(ctx context.Context) (*kernel.KernelInfo, error) {
	return &kernel.KernelInfo{
		Version:      "6.5.0-44-generic",
		Release:      "6.5.0",
		Architecture: "x86_64",
	}, nil
}

// GetLoadedModules implements kernel.Detector.
func (m *mockVerificationKernelDetector) GetLoadedModules(ctx context.Context) ([]kernel.ModuleInfo, error) {
	var modules []kernel.ModuleInfo
	for name, loaded := range m.moduleLoaded {
		if loaded {
			modules = append(modules, kernel.ModuleInfo{Name: name})
		}
	}
	return modules, nil
}

// GetModule implements kernel.Detector.
func (m *mockVerificationKernelDetector) GetModule(ctx context.Context, name string) (*kernel.ModuleInfo, error) {
	if m.moduleLoaded[name] {
		return &kernel.ModuleInfo{Name: name}, nil
	}
	return nil, nil
}

// AreHeadersInstalled implements kernel.Detector.
func (m *mockVerificationKernelDetector) AreHeadersInstalled(ctx context.Context) (bool, error) {
	return true, nil
}

// GetHeadersPackage implements kernel.Detector.
func (m *mockVerificationKernelDetector) GetHeadersPackage(ctx context.Context) (string, error) {
	return "linux-headers-6.5.0-44-generic", nil
}

// IsSecureBootEnabled implements kernel.Detector.
func (m *mockVerificationKernelDetector) IsSecureBootEnabled(ctx context.Context) (bool, error) {
	return false, nil
}

// Ensure mockVerificationKernelDetector implements kernel.Detector.
var _ kernel.Detector = (*mockVerificationKernelDetector)(nil)

// =============================================================================
// Test Helpers
// =============================================================================

// newVerificationTestContext creates a test context with mock executor.
func newVerificationTestContext() (*install.Context, *exec.MockExecutor) {
	mockExec := exec.NewMockExecutor()
	mockExec.SetDefaultResponse(exec.SuccessResult(""))

	ctx := install.NewContext(
		install.WithExecutor(mockExec),
	)

	return ctx, mockExec
}

// setupAllSuccessfulChecks configures mock for all successful checks.
func setupAllSuccessfulChecks(mockExec *exec.MockExecutor) {
	// nvidia-smi for driver version and GPU detection
	mockExec.SetResponse("nvidia-smi", exec.SuccessResult("550.54.14"))
	// lsmod for module check
	mockExec.SetResponse("lsmod", exec.SuccessResult("Module                  Size  Used by\nnvidia              12345678  5\n"))
	// test for file existence
	mockExec.SetResponse("test", exec.SuccessResult(""))
}

// =============================================================================
// Constructor Tests
// =============================================================================

func TestNewVerificationStep(t *testing.T) {
	step := NewVerificationStep()

	assert.Equal(t, "verification", step.Name())
	assert.Equal(t, "Verify NVIDIA driver installation", step.Description())
	assert.False(t, step.CanRollback())
	assert.True(t, step.checkNvidiaSmi)
	assert.True(t, step.checkModuleLoaded)
	assert.True(t, step.checkGPUDetected)
	assert.False(t, step.checkXorgConfig)
	assert.False(t, step.failOnWarning)
	assert.Nil(t, step.kernelDetector)
	assert.Empty(t, step.customChecks)
}

func TestVerificationStepOptions(t *testing.T) {
	mockDetector := newMockVerificationKernelDetector()
	customCheck := func(ctx *install.Context) VerificationCheck {
		return VerificationCheck{Name: "custom", Passed: true}
	}

	step := NewVerificationStep(
		WithCheckNvidiaSmi(false),
		WithCheckModuleLoaded(false),
		WithCheckGPUDetected(false),
		WithCheckXorgConfig(true),
		WithFailOnWarning(true),
		WithVerificationKernelDetector(mockDetector),
		WithCustomCheck(customCheck),
	)

	assert.False(t, step.checkNvidiaSmi)
	assert.False(t, step.checkModuleLoaded)
	assert.False(t, step.checkGPUDetected)
	assert.True(t, step.checkXorgConfig)
	assert.True(t, step.failOnWarning)
	assert.Equal(t, mockDetector, step.kernelDetector)
	assert.Len(t, step.customChecks, 1)
}

func TestVerificationStepOptions_MultipleCustomChecks(t *testing.T) {
	check1 := func(ctx *install.Context) VerificationCheck {
		return VerificationCheck{Name: "check1", Passed: true}
	}
	check2 := func(ctx *install.Context) VerificationCheck {
		return VerificationCheck{Name: "check2", Passed: true}
	}

	step := NewVerificationStep(
		WithCustomCheck(check1),
		WithCustomCheck(check2),
	)

	assert.Len(t, step.customChecks, 2)
}

// =============================================================================
// Execute Tests - All Checks Pass
// =============================================================================

func TestVerificationStep_Execute_AllChecksPassed(t *testing.T) {
	ctx, mockExec := newVerificationTestContext()
	mockDetector := newMockVerificationKernelDetector()
	mockDetector.SetModuleLoaded("nvidia", true)

	setupAllSuccessfulChecks(mockExec)

	step := NewVerificationStep(
		WithVerificationKernelDetector(mockDetector),
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusCompleted, result.Status)
	assert.Contains(t, result.Message, "passed")
	assert.True(t, ctx.GetStateBool(StateVerificationPassed))
	assert.True(t, ctx.GetStateBool(StateNvidiaSmiAvailable))
	assert.True(t, ctx.GetStateBool(StateModuleLoaded))
	assert.True(t, ctx.GetStateBool(StateGPUDetected))
}

// =============================================================================
// Execute Tests - nvidia-smi Not Found
// =============================================================================

func TestVerificationStep_Execute_NvidiaSmiNotFound(t *testing.T) {
	ctx, mockExec := newVerificationTestContext()

	mockExec.SetResponse("nvidia-smi", exec.FailureResult(127, "nvidia-smi: command not found"))

	step := NewVerificationStep(
		WithCheckModuleLoaded(false),
		WithCheckGPUDetected(false),
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusFailed, result.Status)
	assert.Contains(t, result.Message, "verification failed")
	assert.False(t, ctx.GetStateBool(StateVerificationPassed))
	assert.False(t, ctx.GetStateBool(StateNvidiaSmiAvailable))
}

func TestVerificationStep_Execute_NvidiaSmiEmpty(t *testing.T) {
	ctx, mockExec := newVerificationTestContext()

	mockExec.SetResponse("nvidia-smi", exec.SuccessResult(""))

	step := NewVerificationStep(
		WithCheckModuleLoaded(false),
		WithCheckGPUDetected(false),
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusFailed, result.Status)
	assert.False(t, ctx.GetStateBool(StateNvidiaSmiAvailable))
}

// =============================================================================
// Execute Tests - Module Not Loaded
// =============================================================================

func TestVerificationStep_Execute_ModuleNotLoaded(t *testing.T) {
	ctx, mockExec := newVerificationTestContext()
	mockDetector := newMockVerificationKernelDetector()
	mockDetector.SetModuleLoaded("nvidia", false)

	mockExec.SetResponse("nvidia-smi", exec.SuccessResult("550.54.14"))

	step := NewVerificationStep(
		WithVerificationKernelDetector(mockDetector),
		WithCheckNvidiaSmi(false),
		WithCheckGPUDetected(false),
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusFailed, result.Status)
	assert.Contains(t, result.Message, "verification failed")
	assert.False(t, ctx.GetStateBool(StateVerificationPassed))
	assert.False(t, ctx.GetStateBool(StateModuleLoaded))
}

func TestVerificationStep_Execute_ModuleNotLoaded_ViaLsmod(t *testing.T) {
	ctx, mockExec := newVerificationTestContext()

	// No kernel detector - falls back to lsmod
	mockExec.SetResponse("lsmod", exec.SuccessResult("Module                  Size  Used by\nsnd_hda_intel          45056  0\n"))

	step := NewVerificationStep(
		WithCheckNvidiaSmi(false),
		WithCheckGPUDetected(false),
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusFailed, result.Status)
	assert.False(t, ctx.GetStateBool(StateModuleLoaded))
}

func TestVerificationStep_Execute_ModuleLoaded_ViaLsmod(t *testing.T) {
	ctx, mockExec := newVerificationTestContext()

	// No kernel detector - falls back to lsmod
	mockExec.SetResponse("lsmod", exec.SuccessResult("Module                  Size  Used by\nnvidia              12345678  5\n"))

	step := NewVerificationStep(
		WithCheckNvidiaSmi(false),
		WithCheckGPUDetected(false),
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusCompleted, result.Status)
	assert.True(t, ctx.GetStateBool(StateModuleLoaded))
}

// =============================================================================
// Execute Tests - GPU Not Detected
// =============================================================================

func TestVerificationStep_Execute_GPUNotDetected(t *testing.T) {
	ctx, mockExec := newVerificationTestContext()

	// Empty output = no GPU
	mockExec.SetResponse("nvidia-smi", exec.SuccessResult(""))

	step := NewVerificationStep(
		WithCheckNvidiaSmi(false),
		WithCheckModuleLoaded(false),
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusFailed, result.Status)
	assert.Contains(t, result.Message, "verification failed")
	assert.False(t, ctx.GetStateBool(StateVerificationPassed))
	assert.False(t, ctx.GetStateBool(StateGPUDetected))
}

func TestVerificationStep_Execute_GPUDetectionFailed(t *testing.T) {
	ctx, mockExec := newVerificationTestContext()

	mockExec.SetResponse("nvidia-smi", exec.FailureResult(1, "Unable to determine the device handle"))

	step := NewVerificationStep(
		WithCheckNvidiaSmi(false),
		WithCheckModuleLoaded(false),
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusFailed, result.Status)
	assert.False(t, ctx.GetStateBool(StateGPUDetected))
}

func TestVerificationStep_Execute_GPUDetected(t *testing.T) {
	ctx, mockExec := newVerificationTestContext()

	mockExec.SetResponse("nvidia-smi", exec.SuccessResult("NVIDIA GeForce RTX 3080, 10240 MiB"))

	step := NewVerificationStep(
		WithCheckNvidiaSmi(false),
		WithCheckModuleLoaded(false),
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusCompleted, result.Status)
	assert.True(t, ctx.GetStateBool(StateGPUDetected))
}

func TestVerificationStep_Execute_MultipleGPUsDetected(t *testing.T) {
	ctx, mockExec := newVerificationTestContext()

	mockExec.SetResponse("nvidia-smi", exec.SuccessResult("NVIDIA GeForce RTX 3080, 10240 MiB\nNVIDIA GeForce RTX 3070, 8192 MiB"))

	step := NewVerificationStep(
		WithCheckNvidiaSmi(false),
		WithCheckModuleLoaded(false),
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusCompleted, result.Status)
	assert.True(t, ctx.GetStateBool(StateGPUDetected))
}

// =============================================================================
// Execute Tests - X.org Config
// =============================================================================

func TestVerificationStep_Execute_XorgConfigMissing(t *testing.T) {
	ctx, mockExec := newVerificationTestContext()
	mockDetector := newMockVerificationKernelDetector()
	mockDetector.SetModuleLoaded("nvidia", true)

	mockExec.SetResponse("nvidia-smi", exec.SuccessResult("NVIDIA GeForce RTX 3080, 10240 MiB"))
	mockExec.SetResponse("test", exec.FailureResult(1, "")) // File not found

	step := NewVerificationStep(
		WithVerificationKernelDetector(mockDetector),
		WithCheckXorgConfig(true),
	)

	result := step.Execute(ctx)

	// Should still pass since X.org config is not critical
	assert.Equal(t, install.StepStatusCompleted, result.Status)
	assert.True(t, ctx.GetStateBool(StateVerificationPassed))
}

func TestVerificationStep_Execute_XorgConfigExists(t *testing.T) {
	ctx, mockExec := newVerificationTestContext()
	mockDetector := newMockVerificationKernelDetector()
	mockDetector.SetModuleLoaded("nvidia", true)

	mockExec.SetResponse("nvidia-smi", exec.SuccessResult("NVIDIA GeForce RTX 3080, 10240 MiB"))
	mockExec.SetResponse("test", exec.SuccessResult("")) // File exists

	step := NewVerificationStep(
		WithVerificationKernelDetector(mockDetector),
		WithCheckXorgConfig(true),
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusCompleted, result.Status)
}

// =============================================================================
// Execute Tests - Partial Failure
// =============================================================================

func TestVerificationStep_Execute_PartialFailure_NonCritical(t *testing.T) {
	ctx, mockExec := newVerificationTestContext()
	mockDetector := newMockVerificationKernelDetector()
	mockDetector.SetModuleLoaded("nvidia", true)

	mockExec.SetResponse("nvidia-smi", exec.SuccessResult("NVIDIA GeForce RTX 3080, 10240 MiB"))
	mockExec.SetResponse("test", exec.FailureResult(1, "")) // X.org config not found

	step := NewVerificationStep(
		WithVerificationKernelDetector(mockDetector),
		WithCheckXorgConfig(true), // Non-critical check
	)

	result := step.Execute(ctx)

	// Should pass with warnings since X.org is non-critical
	assert.Equal(t, install.StepStatusCompleted, result.Status)
	assert.Contains(t, result.Message, "warning")
	assert.True(t, ctx.GetStateBool(StateVerificationPassed))
}

// =============================================================================
// Execute Tests - Critical Failure
// =============================================================================

func TestVerificationStep_Execute_CriticalFailure(t *testing.T) {
	ctx, mockExec := newVerificationTestContext()

	mockExec.SetResponse("nvidia-smi", exec.FailureResult(1, "NVIDIA-SMI has failed"))

	step := NewVerificationStep(
		WithCheckModuleLoaded(false),
		WithCheckGPUDetected(false),
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusFailed, result.Status)
	assert.Contains(t, result.Message, "verification failed")
	assert.False(t, ctx.GetStateBool(StateVerificationPassed))
}

// =============================================================================
// Execute Tests - Dry Run
// =============================================================================

func TestVerificationStep_Execute_DryRun(t *testing.T) {
	ctx, mockExec := newVerificationTestContext()
	ctx.DryRun = true

	step := NewVerificationStep()

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusCompleted, result.Status)
	assert.Contains(t, result.Message, "dry run")

	// No actual commands should be executed
	assert.False(t, mockExec.WasCalled("nvidia-smi"))
	assert.False(t, mockExec.WasCalled("lsmod"))
}

func TestVerificationStep_Execute_DryRun_WithAllOptions(t *testing.T) {
	ctx, _ := newVerificationTestContext()
	ctx.DryRun = true

	customCheck := func(ctx *install.Context) VerificationCheck {
		return VerificationCheck{Name: "custom", Passed: true}
	}

	step := NewVerificationStep(
		WithCheckXorgConfig(true),
		WithCustomCheck(customCheck),
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusCompleted, result.Status)
	assert.Contains(t, result.Message, "dry run")
}

// =============================================================================
// Execute Tests - Cancelled
// =============================================================================

func TestVerificationStep_Execute_Cancelled(t *testing.T) {
	ctx, _ := newVerificationTestContext()
	ctx.Cancel()

	step := NewVerificationStep()

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusFailed, result.Status)
	assert.Contains(t, result.Message, "cancelled")
	assert.True(t, errors.Is(result.Error, context.Canceled))
}

// =============================================================================
// Execute Tests - No Executor
// =============================================================================

func TestVerificationStep_Execute_NoExecutor(t *testing.T) {
	ctx := install.NewContext()

	step := NewVerificationStep()

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusFailed, result.Status)
	assert.Contains(t, result.Message, "validation failed")
	assert.Error(t, result.Error)
	assert.Contains(t, result.Error.Error(), "executor")
}

// =============================================================================
// Execute Tests - Custom Check
// =============================================================================

func TestVerificationStep_Execute_CustomCheck_Passes(t *testing.T) {
	ctx, mockExec := newVerificationTestContext()
	mockDetector := newMockVerificationKernelDetector()
	mockDetector.SetModuleLoaded("nvidia", true)

	setupAllSuccessfulChecks(mockExec)

	customCheck := func(ctx *install.Context) VerificationCheck {
		return VerificationCheck{
			Name:     "custom-check",
			Passed:   true,
			Message:  "custom check passed",
			Critical: false,
		}
	}

	step := NewVerificationStep(
		WithVerificationKernelDetector(mockDetector),
		WithCustomCheck(customCheck),
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusCompleted, result.Status)
	assert.True(t, ctx.GetStateBool(StateVerificationPassed))
}

func TestVerificationStep_Execute_CustomCheck_FailsCritical(t *testing.T) {
	ctx, mockExec := newVerificationTestContext()
	mockDetector := newMockVerificationKernelDetector()
	mockDetector.SetModuleLoaded("nvidia", true)

	setupAllSuccessfulChecks(mockExec)

	customCheck := func(ctx *install.Context) VerificationCheck {
		return VerificationCheck{
			Name:     "custom-critical",
			Passed:   false,
			Message:  "custom critical check failed",
			Critical: true,
		}
	}

	step := NewVerificationStep(
		WithVerificationKernelDetector(mockDetector),
		WithCustomCheck(customCheck),
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusFailed, result.Status)
	assert.False(t, ctx.GetStateBool(StateVerificationPassed))
}

func TestVerificationStep_Execute_CustomCheck_FailsNonCritical(t *testing.T) {
	ctx, mockExec := newVerificationTestContext()
	mockDetector := newMockVerificationKernelDetector()
	mockDetector.SetModuleLoaded("nvidia", true)

	setupAllSuccessfulChecks(mockExec)

	customCheck := func(ctx *install.Context) VerificationCheck {
		return VerificationCheck{
			Name:     "custom-warning",
			Passed:   false,
			Message:  "custom warning",
			Critical: false,
		}
	}

	step := NewVerificationStep(
		WithVerificationKernelDetector(mockDetector),
		WithCustomCheck(customCheck),
	)

	result := step.Execute(ctx)

	// Should pass with warning
	assert.Equal(t, install.StepStatusCompleted, result.Status)
	assert.True(t, ctx.GetStateBool(StateVerificationPassed))
}

func TestVerificationStep_Execute_MultipleCustomChecks(t *testing.T) {
	ctx, mockExec := newVerificationTestContext()
	mockDetector := newMockVerificationKernelDetector()
	mockDetector.SetModuleLoaded("nvidia", true)

	setupAllSuccessfulChecks(mockExec)

	check1 := func(ctx *install.Context) VerificationCheck {
		return VerificationCheck{Name: "check1", Passed: true, Message: "check1 passed", Critical: false}
	}
	check2 := func(ctx *install.Context) VerificationCheck {
		return VerificationCheck{Name: "check2", Passed: true, Message: "check2 passed", Critical: false}
	}

	step := NewVerificationStep(
		WithVerificationKernelDetector(mockDetector),
		WithCustomCheck(check1),
		WithCustomCheck(check2),
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusCompleted, result.Status)
	assert.True(t, ctx.GetStateBool(StateVerificationPassed))
}

// =============================================================================
// Execute Tests - Fail On Warning
// =============================================================================

func TestVerificationStep_Execute_FailOnWarning(t *testing.T) {
	ctx, mockExec := newVerificationTestContext()
	mockDetector := newMockVerificationKernelDetector()
	mockDetector.SetModuleLoaded("nvidia", true)

	mockExec.SetResponse("nvidia-smi", exec.SuccessResult("NVIDIA GeForce RTX 3080, 10240 MiB"))
	mockExec.SetResponse("test", exec.FailureResult(1, "")) // X.org config not found

	step := NewVerificationStep(
		WithVerificationKernelDetector(mockDetector),
		WithCheckXorgConfig(true),
		WithFailOnWarning(true),
	)

	result := step.Execute(ctx)

	// Should fail because failOnWarning is true
	assert.Equal(t, install.StepStatusFailed, result.Status)
	assert.False(t, ctx.GetStateBool(StateVerificationPassed))
}

// =============================================================================
// Rollback Tests
// =============================================================================

func TestVerificationStep_Rollback(t *testing.T) {
	ctx, _ := newVerificationTestContext()

	step := NewVerificationStep()

	err := step.Rollback(ctx)

	assert.NoError(t, err) // Rollback is a no-op
}

func TestVerificationStep_Rollback_WithState(t *testing.T) {
	ctx, _ := newVerificationTestContext()

	// Set some verification state
	ctx.SetState(StateVerificationPassed, true)
	ctx.SetState(StateDriverVersion, "550.54.14")

	step := NewVerificationStep()

	err := step.Rollback(ctx)

	assert.NoError(t, err)

	// State should remain (rollback is a no-op)
	assert.True(t, ctx.GetStateBool(StateVerificationPassed))
}

// =============================================================================
// CanRollback Tests
// =============================================================================

func TestVerificationStep_CanRollback(t *testing.T) {
	step := NewVerificationStep()

	assert.False(t, step.CanRollback())
}

// =============================================================================
// Validate Tests
// =============================================================================

func TestVerificationStep_Validate_Success(t *testing.T) {
	ctx, _ := newVerificationTestContext()

	step := NewVerificationStep()

	err := step.Validate(ctx)

	assert.NoError(t, err)
}

func TestVerificationStep_Validate_NoExecutor(t *testing.T) {
	ctx := install.NewContext()

	step := NewVerificationStep()

	err := step.Validate(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "executor is required")
}

// =============================================================================
// ParseDriverVersion Tests
// =============================================================================

func TestVerificationStep_ParseDriverVersion(t *testing.T) {
	step := NewVerificationStep()

	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple version",
			input:    "550.54.14",
			expected: "550.54.14",
		},
		{
			name:     "version with whitespace",
			input:    "  550.54.14  ",
			expected: "550.54.14",
		},
		{
			name:     "version with newline",
			input:    "550.54.14\n",
			expected: "550.54.14",
		},
		{
			name:     "csv format",
			input:    "550.54.14, NVIDIA GeForce RTX 3080, 10240 MiB",
			expected: "550.54.14",
		},
		{
			name:     "multi-line output",
			input:    "550.54.14\n550.54.14",
			expected: "550.54.14",
		},
		{
			name:     "empty input",
			input:    "",
			expected: "",
		},
		{
			name:     "whitespace only",
			input:    "   ",
			expected: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := step.parseDriverVersion(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// =============================================================================
// VerificationCheck Struct Tests
// =============================================================================

func TestVerificationCheck_Struct(t *testing.T) {
	check := VerificationCheck{
		Name:        "test-check",
		Description: "Test description",
		Passed:      true,
		Message:     "Test message",
		Critical:    true,
	}

	assert.Equal(t, "test-check", check.Name)
	assert.Equal(t, "Test description", check.Description)
	assert.True(t, check.Passed)
	assert.Equal(t, "Test message", check.Message)
	assert.True(t, check.Critical)
}

// =============================================================================
// State Keys Tests
// =============================================================================

func TestVerificationStep_StateKeys(t *testing.T) {
	assert.Equal(t, "verification_passed", StateVerificationPassed)
	assert.Equal(t, "driver_version", StateDriverVersion)
	assert.Equal(t, "nvidia_smi_available", StateNvidiaSmiAvailable)
	assert.Equal(t, "module_loaded", StateModuleLoaded)
	assert.Equal(t, "gpu_detected", StateGPUDetected)
	assert.Equal(t, "verification_errors", StateVerificationErrors)
}

// =============================================================================
// Interface Compliance Tests
// =============================================================================

func TestVerificationStep_InterfaceCompliance(t *testing.T) {
	var _ install.Step = (*VerificationStep)(nil)
}

// =============================================================================
// Helper Method Tests
// =============================================================================

func TestVerificationStep_checkModuleViaLsmod(t *testing.T) {
	testCases := []struct {
		name       string
		lsmodOut   string
		moduleName string
		wantLoaded bool
		wantErr    bool
	}{
		{
			name:       "module loaded",
			lsmodOut:   "Module                  Size  Used by\nnvidia              12345678  5\n",
			moduleName: "nvidia",
			wantLoaded: true,
			wantErr:    false,
		},
		{
			name:       "module not loaded",
			lsmodOut:   "Module                  Size  Used by\nsnd_hda_intel          45056  0\n",
			moduleName: "nvidia",
			wantLoaded: false,
			wantErr:    false,
		},
		{
			name:       "partial match should not match",
			lsmodOut:   "Module                  Size  Used by\nnvidia_drm          12345  0\n",
			moduleName: "nvidia",
			wantLoaded: false,
			wantErr:    false,
		},
		{
			name:       "empty output",
			lsmodOut:   "",
			moduleName: "nvidia",
			wantLoaded: false,
			wantErr:    false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, mockExec := newVerificationTestContext()
			mockExec.SetResponse("lsmod", exec.SuccessResult(tc.lsmodOut))

			step := NewVerificationStep()
			loaded, err := step.checkModuleViaLsmod(ctx, tc.moduleName)

			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.wantLoaded, loaded)
			}
		})
	}
}

func TestVerificationStep_checkModuleViaLsmod_Error(t *testing.T) {
	ctx, mockExec := newVerificationTestContext()
	mockExec.SetResponse("lsmod", exec.FailureResult(1, "command failed"))

	step := NewVerificationStep()
	_, err := step.checkModuleViaLsmod(ctx, "nvidia")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to list modules")
}

// =============================================================================
// Kernel Detector Fallback Tests
// =============================================================================

func TestVerificationStep_Execute_KernelDetectorError_FallsBackToLsmod(t *testing.T) {
	ctx, mockExec := newVerificationTestContext()
	mockDetector := newMockVerificationKernelDetector()
	mockDetector.SetModuleLoadedError(errors.New("detector error"))

	mockExec.SetResponse("lsmod", exec.SuccessResult("Module                  Size  Used by\nnvidia              12345678  5\n"))
	mockExec.SetResponse("nvidia-smi", exec.SuccessResult("NVIDIA GeForce RTX 3080, 10240 MiB"))

	step := NewVerificationStep(
		WithVerificationKernelDetector(mockDetector),
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusCompleted, result.Status)
	assert.True(t, ctx.GetStateBool(StateModuleLoaded))
}

// =============================================================================
// Store Results Tests
// =============================================================================

func TestVerificationStep_StoresVerificationErrors(t *testing.T) {
	ctx, mockExec := newVerificationTestContext()

	mockExec.SetResponse("nvidia-smi", exec.FailureResult(1, "command failed"))

	step := NewVerificationStep(
		WithCheckModuleLoaded(false),
		WithCheckGPUDetected(false),
	)

	step.Execute(ctx)

	errorsRaw, ok := ctx.GetState(StateVerificationErrors)
	require.True(t, ok)
	errs, ok := errorsRaw.([]string)
	require.True(t, ok)
	assert.NotEmpty(t, errs)
}

func TestVerificationStep_StoresDriverVersion(t *testing.T) {
	ctx, mockExec := newVerificationTestContext()
	mockDetector := newMockVerificationKernelDetector()
	mockDetector.SetModuleLoaded("nvidia", true)

	mockExec.SetResponse("nvidia-smi", exec.SuccessResult("550.54.14"))

	step := NewVerificationStep(
		WithVerificationKernelDetector(mockDetector),
		WithCheckGPUDetected(false),
	)

	step.Execute(ctx)

	driverVersion := ctx.GetStateString(StateDriverVersion)
	assert.Equal(t, "550.54.14", driverVersion)
}

// =============================================================================
// Duration Tests
// =============================================================================

func TestVerificationStep_Duration(t *testing.T) {
	ctx, mockExec := newVerificationTestContext()
	mockDetector := newMockVerificationKernelDetector()
	mockDetector.SetModuleLoaded("nvidia", true)

	setupAllSuccessfulChecks(mockExec)

	step := NewVerificationStep(
		WithVerificationKernelDetector(mockDetector),
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusCompleted, result.Status)
	assert.Greater(t, result.Duration.Nanoseconds(), int64(0))
}

// =============================================================================
// Disabled Checks Tests
// =============================================================================

func TestVerificationStep_Execute_AllChecksDisabled(t *testing.T) {
	ctx, mockExec := newVerificationTestContext()

	step := NewVerificationStep(
		WithCheckNvidiaSmi(false),
		WithCheckModuleLoaded(false),
		WithCheckGPUDetected(false),
		WithCheckXorgConfig(false),
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusCompleted, result.Status)
	assert.True(t, ctx.GetStateBool(StateVerificationPassed))

	// No commands should be called
	assert.False(t, mockExec.WasCalled("nvidia-smi"))
	assert.False(t, mockExec.WasCalled("lsmod"))
}

func TestVerificationStep_Execute_OnlyNvidiaSmi(t *testing.T) {
	ctx, mockExec := newVerificationTestContext()

	mockExec.SetResponse("nvidia-smi", exec.SuccessResult("550.54.14"))

	step := NewVerificationStep(
		WithCheckModuleLoaded(false),
		WithCheckGPUDetected(false),
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusCompleted, result.Status)
	assert.True(t, mockExec.WasCalled("nvidia-smi"))
	assert.False(t, mockExec.WasCalled("lsmod"))
}

func TestVerificationStep_Execute_OnlyModuleLoaded(t *testing.T) {
	ctx, mockExec := newVerificationTestContext()

	mockExec.SetResponse("lsmod", exec.SuccessResult("Module                  Size  Used by\nnvidia              12345678  5\n"))

	step := NewVerificationStep(
		WithCheckNvidiaSmi(false),
		WithCheckGPUDetected(false),
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusCompleted, result.Status)
	assert.True(t, mockExec.WasCalled("lsmod"))
	assert.False(t, mockExec.WasCalled("nvidia-smi"))
}

// =============================================================================
// Benchmarks
// =============================================================================

func BenchmarkVerificationStep_Execute(b *testing.B) {
	mockExec := exec.NewMockExecutor()
	setupAllSuccessfulChecks(mockExec)

	mockDetector := newMockVerificationKernelDetector()
	mockDetector.SetModuleLoaded("nvidia", true)

	step := NewVerificationStep(
		WithVerificationKernelDetector(mockDetector),
	)

	ctx := install.NewContext(
		install.WithExecutor(mockExec),
	)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		step.Execute(ctx)
	}
}

func BenchmarkVerificationStep_Validate(b *testing.B) {
	mockExec := exec.NewMockExecutor()

	ctx := install.NewContext(
		install.WithExecutor(mockExec),
	)

	step := NewVerificationStep()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = step.Validate(ctx)
	}
}
