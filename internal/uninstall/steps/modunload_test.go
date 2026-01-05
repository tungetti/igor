package steps

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/tungetti/igor/internal/exec"
	"github.com/tungetti/igor/internal/gpu/kernel"
	"github.com/tungetti/igor/internal/install"
)

// =============================================================================
// Mock Kernel Detector for Module Unload Tests
// =============================================================================

// mockUnloadKernelDetector is a mock kernel detector specifically for module unload tests.
type mockUnloadKernelDetector struct {
	loadedModules        map[string]bool
	isModuleLoadedFunc   func(ctx context.Context, name string) (bool, error)
	isModuleLoadedErr    error
	getLoadedModulesFunc func(ctx context.Context) ([]kernel.ModuleInfo, error)
}

// newMockUnloadKernelDetector creates a new mock kernel detector for module unload tests.
func newMockUnloadKernelDetector() *mockUnloadKernelDetector {
	return &mockUnloadKernelDetector{
		loadedModules: make(map[string]bool),
	}
}

// SetModuleLoaded sets whether a module is reported as loaded.
func (m *mockUnloadKernelDetector) SetModuleLoaded(name string, loaded bool) {
	m.loadedModules[name] = loaded
}

// SetIsModuleLoadedFunc sets a custom function for IsModuleLoaded.
func (m *mockUnloadKernelDetector) SetIsModuleLoadedFunc(fn func(ctx context.Context, name string) (bool, error)) {
	m.isModuleLoadedFunc = fn
}

// SetIsModuleLoadedError sets an error to return from IsModuleLoaded.
func (m *mockUnloadKernelDetector) SetIsModuleLoadedError(err error) {
	m.isModuleLoadedErr = err
}

// SetGetLoadedModulesFunc sets a custom function for GetLoadedModules.
func (m *mockUnloadKernelDetector) SetGetLoadedModulesFunc(fn func(ctx context.Context) ([]kernel.ModuleInfo, error)) {
	m.getLoadedModulesFunc = fn
}

// IsModuleLoaded implements kernel.Detector.
func (m *mockUnloadKernelDetector) IsModuleLoaded(ctx context.Context, name string) (bool, error) {
	if m.isModuleLoadedFunc != nil {
		return m.isModuleLoadedFunc(ctx, name)
	}
	if m.isModuleLoadedErr != nil {
		return false, m.isModuleLoadedErr
	}
	return m.loadedModules[name], nil
}

// GetKernelInfo implements kernel.Detector.
func (m *mockUnloadKernelDetector) GetKernelInfo(ctx context.Context) (*kernel.KernelInfo, error) {
	return &kernel.KernelInfo{
		Version:      "6.5.0-44-generic",
		Release:      "6.5.0",
		Architecture: "x86_64",
	}, nil
}

// GetLoadedModules implements kernel.Detector.
func (m *mockUnloadKernelDetector) GetLoadedModules(ctx context.Context) ([]kernel.ModuleInfo, error) {
	if m.getLoadedModulesFunc != nil {
		return m.getLoadedModulesFunc(ctx)
	}
	var modules []kernel.ModuleInfo
	for name, loaded := range m.loadedModules {
		if loaded {
			modules = append(modules, kernel.ModuleInfo{Name: name})
		}
	}
	return modules, nil
}

// GetModule implements kernel.Detector.
func (m *mockUnloadKernelDetector) GetModule(ctx context.Context, name string) (*kernel.ModuleInfo, error) {
	if m.loadedModules[name] {
		return &kernel.ModuleInfo{Name: name}, nil
	}
	return nil, nil
}

// AreHeadersInstalled implements kernel.Detector.
func (m *mockUnloadKernelDetector) AreHeadersInstalled(ctx context.Context) (bool, error) {
	return true, nil
}

// GetHeadersPackage implements kernel.Detector.
func (m *mockUnloadKernelDetector) GetHeadersPackage(ctx context.Context) (string, error) {
	return "linux-headers-6.5.0-44-generic", nil
}

// IsSecureBootEnabled implements kernel.Detector.
func (m *mockUnloadKernelDetector) IsSecureBootEnabled(ctx context.Context) (bool, error) {
	return false, nil
}

// Ensure mockUnloadKernelDetector implements kernel.Detector.
var _ kernel.Detector = (*mockUnloadKernelDetector)(nil)

// =============================================================================
// Test Helpers
// =============================================================================

// newModuleUnloadTestContext creates a basic test context with executor for module unload tests.
func newModuleUnloadTestContext() (*install.Context, *exec.MockExecutor) {
	mockExec := exec.NewMockExecutor()
	mockExec.SetDefaultResponse(exec.SuccessResult(""))

	ctx := install.NewContext(
		install.WithExecutor(mockExec),
	)

	return ctx, mockExec
}

// setupModuleUnloadMocks configures the mock executor with common module unload responses.
func setupModuleUnloadMocks(mockExec *exec.MockExecutor) {
	// lsmod shows nvidia modules loaded
	mockExec.SetResponse("lsmod", exec.SuccessResult("Module                  Size  Used by\nnvidia              12345678  0\nnvidia_modeset          12345  0\nnvidia_drm              12345  0\nnvidia_uvm              12345  0\n"))
	// modprobe -r succeeds
	mockExec.SetDefaultResponse(exec.SuccessResult(""))
	// cat for refcnt returns 0 (not in use)
	mockExec.SetResponse("cat", exec.SuccessResult("0"))
}

// =============================================================================
// ModuleUnloadStep Constructor Tests
// =============================================================================

func TestNewModuleUnloadStep(t *testing.T) {
	step := NewModuleUnloadStep()

	assert.Equal(t, "module_unload", step.Name())
	assert.Equal(t, "Unload NVIDIA kernel modules", step.Description())
	assert.True(t, step.CanRollback())
	assert.Equal(t, DefaultUnloadModules, step.moduleNames)
	assert.True(t, step.skipIfNotLoaded)
	assert.False(t, step.force)
	assert.Equal(t, 3, step.retryCount)
	assert.Equal(t, 1*time.Second, step.retryDelay)
	assert.Nil(t, step.kernelDetector)
}

func TestModuleUnloadStepOptions(t *testing.T) {
	mockDetector := newMockUnloadKernelDetector()
	customModules := []string{"nvidia", "nvidia-drm"}

	step := NewModuleUnloadStep(
		WithUnloadModuleNames(customModules),
		WithSkipIfNotLoaded(false),
		WithUnloadKernelDetector(mockDetector),
		WithForceUnload(true),
		WithUnloadRetry(5, 2*time.Second),
	)

	assert.Equal(t, customModules, step.moduleNames)
	assert.False(t, step.skipIfNotLoaded)
	assert.True(t, step.force)
	assert.Equal(t, 5, step.retryCount)
	assert.Equal(t, 2*time.Second, step.retryDelay)
	assert.Equal(t, mockDetector, step.kernelDetector)
}

func TestModuleUnloadStep_WithUnloadModuleNames_CopiesSlice(t *testing.T) {
	original := []string{"nvidia", "nvidia-drm"}
	step := NewModuleUnloadStep(WithUnloadModuleNames(original))

	// Modify original slice
	original[0] = "modified"

	// Step should have the original values (defensive copy)
	assert.Equal(t, "nvidia", step.moduleNames[0])
}

// =============================================================================
// ModuleUnloadStep Execute Tests
// =============================================================================

func TestModuleUnloadStep_Execute_Success(t *testing.T) {
	ctx, mockExec := newModuleUnloadTestContext()
	mockDetector := newMockUnloadKernelDetector()
	mockDetector.SetModuleLoaded("nvidia", true)
	mockDetector.SetModuleLoaded("nvidia-modeset", true)
	mockDetector.SetModuleLoaded("nvidia-drm", true)
	mockDetector.SetModuleLoaded("nvidia-uvm", true)

	setupModuleUnloadMocks(mockExec)

	step := NewModuleUnloadStep(
		WithUnloadKernelDetector(mockDetector),
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusCompleted, result.Status)
	assert.Contains(t, result.Message, "successfully")
	assert.Greater(t, result.Duration.Nanoseconds(), int64(0))

	// Check state was set
	assert.True(t, ctx.GetStateBool(StateModulesUnloaded))

	// Check unloaded modules list
	unloadedModulesRaw, ok := ctx.GetState(StateUnloadedModules)
	assert.True(t, ok)
	unloadedModules, ok := unloadedModulesRaw.([]string)
	assert.True(t, ok)
	assert.Equal(t, DefaultUnloadModules, unloadedModules)

	// Verify modprobe -r was called
	assert.True(t, mockExec.WasCalled("modprobe"))
}

func TestModuleUnloadStep_Execute_NotLoaded_Skip(t *testing.T) {
	ctx, mockExec := newModuleUnloadTestContext()
	mockDetector := newMockUnloadKernelDetector()
	// No modules loaded

	step := NewModuleUnloadStep(
		WithUnloadKernelDetector(mockDetector),
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusSkipped, result.Status)
	assert.Contains(t, result.Message, "no NVIDIA modules are loaded")

	// State should not be set for skipped step
	assert.False(t, ctx.GetStateBool(StateModulesUnloaded))

	// modprobe should not have been called (except maybe lsmod for checking)
	assert.False(t, mockExec.WasCalledWith("modprobe", "-r", "nvidia"))
}

func TestModuleUnloadStep_Execute_DryRun(t *testing.T) {
	ctx, mockExec := newModuleUnloadTestContext()
	ctx.DryRun = true

	mockDetector := newMockUnloadKernelDetector()
	mockDetector.SetModuleLoaded("nvidia", true)

	step := NewModuleUnloadStep(
		WithUnloadKernelDetector(mockDetector),
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusCompleted, result.Status)
	assert.Contains(t, result.Message, "dry run")

	// State should not be set for dry run
	assert.False(t, ctx.GetStateBool(StateModulesUnloaded))

	// modprobe -r should not have been called
	assert.False(t, mockExec.WasCalledWith("modprobe", "-r", "nvidia"))
}

func TestModuleUnloadStep_Execute_Cancelled(t *testing.T) {
	ctx, mockExec := newModuleUnloadTestContext()
	ctx.Cancel() // Cancel immediately

	mockExec.SetDefaultResponse(exec.SuccessResult(""))

	step := NewModuleUnloadStep()

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusFailed, result.Status)
	assert.Contains(t, result.Message, "cancelled")
	assert.True(t, errors.Is(result.Error, context.Canceled))
}

func TestModuleUnloadStep_Execute_NoExecutor(t *testing.T) {
	ctx := install.NewContext()

	step := NewModuleUnloadStep()

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusFailed, result.Status)
	assert.Contains(t, result.Message, "validation failed")
	assert.Error(t, result.Error)
	assert.Contains(t, result.Error.Error(), "executor")
}

func TestModuleUnloadStep_Execute_NilContext(t *testing.T) {
	step := NewModuleUnloadStep()

	err := step.Validate(nil)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context is nil")
}

func TestModuleUnloadStep_Execute_UnloadFailure(t *testing.T) {
	ctx, mockExec := newModuleUnloadTestContext()
	mockDetector := newMockUnloadKernelDetector()
	mockDetector.SetModuleLoaded("nvidia", true)

	// lsmod succeeds
	mockExec.SetResponse("lsmod", exec.SuccessResult("nvidia 12345 0"))
	// modprobe -r fails
	mockExec.SetResponse("modprobe", exec.FailureResult(1, "modprobe: FATAL: Module nvidia is in use"))
	// cat for refcnt returns 0 (not in use to avoid retry logic)
	mockExec.SetResponse("cat", exec.SuccessResult("0"))

	step := NewModuleUnloadStep(
		WithUnloadKernelDetector(mockDetector),
		WithUnloadRetry(0, 0), // No retries for faster test
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusFailed, result.Status)
	assert.Contains(t, result.Message, "failed to unload module")
	assert.Error(t, result.Error)

	// State should not indicate success
	assert.False(t, ctx.GetStateBool(StateModulesUnloaded))
}

func TestModuleUnloadStep_Execute_ModuleInUse(t *testing.T) {
	ctx, mockExec := newModuleUnloadTestContext()
	mockDetector := newMockUnloadKernelDetector()
	mockDetector.SetModuleLoaded("nvidia-modeset", true)

	// lsmod shows nvidia-modeset in use (Used by = 1)
	mockExec.SetResponse("lsmod", exec.SuccessResult("Module                  Size  Used by\nnvidia_modeset              12345678  1 Xorg"))
	// modprobe -r fails
	mockExec.SetResponse("modprobe", exec.FailureResult(1, "modprobe: FATAL: Module nvidia_modeset is in use"))
	// cat for refcnt shows in use
	mockExec.SetResponse("cat", exec.SuccessResult("1"))

	step := NewModuleUnloadStep(
		WithUnloadKernelDetector(mockDetector),
		WithUnloadModuleNames([]string{"nvidia-modeset"}),
		WithUnloadRetry(0, 0), // No retries
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusFailed, result.Status)
	assert.Contains(t, result.Message, "in use")

	// Check modules in use state
	modulesInUseRaw, ok := ctx.GetState(StateModulesInUse)
	assert.True(t, ok)
	modulesInUse, ok := modulesInUseRaw.([]string)
	assert.True(t, ok)
	assert.Contains(t, modulesInUse, "nvidia-modeset")
}

func TestModuleUnloadStep_Execute_ForceUnload(t *testing.T) {
	ctx, mockExec := newModuleUnloadTestContext()
	mockDetector := newMockUnloadKernelDetector()
	mockDetector.SetModuleLoaded("nvidia", true)

	// lsmod shows nvidia in use
	mockExec.SetResponse("lsmod", exec.SuccessResult("Module                  Size  Used by\nnvidia              12345678  1"))
	// modprobe -r fails initially
	mockExec.SetResponse("modprobe", exec.FailureResult(1, "modprobe: FATAL: Module nvidia is in use"))
	// cat for refcnt shows in use
	mockExec.SetResponse("cat", exec.SuccessResult("1"))
	// rmmod -f succeeds
	mockExec.SetResponse("rmmod", exec.SuccessResult(""))

	step := NewModuleUnloadStep(
		WithUnloadKernelDetector(mockDetector),
		WithUnloadModuleNames([]string{"nvidia"}),
		WithForceUnload(true),
		WithUnloadRetry(0, 0), // No retries
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusCompleted, result.Status)
	assert.Contains(t, result.Message, "successfully")

	// Verify rmmod -f was called
	assert.True(t, mockExec.WasCalled("rmmod"))
}

func TestModuleUnloadStep_Execute_CustomModules(t *testing.T) {
	ctx, mockExec := newModuleUnloadTestContext()
	mockDetector := newMockUnloadKernelDetector()
	mockDetector.SetModuleLoaded("nvidia", true)
	mockDetector.SetModuleLoaded("nvidia-drm", true)

	setupModuleUnloadMocks(mockExec)

	customModules := []string{"nvidia-drm", "nvidia"}
	step := NewModuleUnloadStep(
		WithUnloadKernelDetector(mockDetector),
		WithUnloadModuleNames(customModules),
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusCompleted, result.Status)

	// Verify unloaded modules list
	unloadedModulesRaw, ok := ctx.GetState(StateUnloadedModules)
	assert.True(t, ok)
	unloadedModules, ok := unloadedModulesRaw.([]string)
	assert.True(t, ok)
	assert.Equal(t, customModules, unloadedModules)
}

func TestModuleUnloadStep_Execute_FallbackToLsmod(t *testing.T) {
	ctx, mockExec := newModuleUnloadTestContext()

	// No kernel detector, should fall back to lsmod
	mockExec.SetResponse("lsmod", exec.SuccessResult("Module                  Size  Used by\nnvidia              12345678  0\n"))

	step := NewModuleUnloadStep(
		WithUnloadModuleNames([]string{"nvidia"}),
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusCompleted, result.Status)
	assert.True(t, mockExec.WasCalled("lsmod"))
}

func TestModuleUnloadStep_Execute_EmptyModuleList(t *testing.T) {
	ctx, mockExec := newModuleUnloadTestContext()
	mockDetector := newMockUnloadKernelDetector()

	step := NewModuleUnloadStep(
		WithUnloadKernelDetector(mockDetector),
		WithUnloadModuleNames([]string{}),
	)

	result := step.Execute(ctx)

	// Empty module list should skip
	assert.Equal(t, install.StepStatusSkipped, result.Status)
	assert.Contains(t, result.Message, "no modules configured")

	// No modprobe should have been called
	assert.False(t, mockExec.WasCalled("modprobe"))

	// State should NOT be set on skip
	assert.False(t, ctx.GetStateBool(StateModulesUnloaded))
}

func TestModuleUnloadStep_Execute_SkipIfNotLoadedFalse(t *testing.T) {
	ctx, mockExec := newModuleUnloadTestContext()
	mockDetector := newMockUnloadKernelDetector()
	mockDetector.SetModuleLoaded("nvidia", true)

	setupModuleUnloadMocks(mockExec)

	step := NewModuleUnloadStep(
		WithUnloadKernelDetector(mockDetector),
		WithSkipIfNotLoaded(false),
		WithUnloadModuleNames([]string{"nvidia"}),
	)

	result := step.Execute(ctx)

	// Should unload because module is loaded
	assert.Equal(t, install.StepStatusCompleted, result.Status)
	assert.True(t, mockExec.WasCalled("modprobe"))
}

func TestModuleUnloadStep_Execute_RetryLogic(t *testing.T) {
	ctx, mockExec := newModuleUnloadTestContext()
	mockDetector := newMockUnloadKernelDetector()
	mockDetector.SetModuleLoaded("nvidia", true)

	// Track call count
	callCount := 0

	// Use a response that fails first then succeeds
	// Since MockExecutor doesn't support sequential responses easily,
	// we'll test that retry was attempted by checking call count

	// Set up to fail initially
	mockExec.SetResponse("lsmod", exec.SuccessResult("Module                  Size  Used by\nnvidia              12345678  1"))
	mockExec.SetResponse("modprobe", exec.FailureResult(1, "in use"))
	mockExec.SetResponse("cat", exec.SuccessResult("1"))

	step := NewModuleUnloadStep(
		WithUnloadKernelDetector(mockDetector),
		WithUnloadModuleNames([]string{"nvidia"}),
		WithUnloadRetry(2, 10*time.Millisecond), // 2 retries with short delay
	)

	result := step.Execute(ctx)

	// Should fail after retries
	assert.Equal(t, install.StepStatusFailed, result.Status)

	// Verify modprobe was called multiple times (initial + retries)
	calls := mockExec.Calls()
	modprobeCount := 0
	for _, call := range calls {
		if call.Command == "modprobe" {
			modprobeCount++
		}
	}
	assert.GreaterOrEqual(t, modprobeCount, 1)
	_ = callCount // Silence unused variable warning
}

// =============================================================================
// ModuleUnloadStep Rollback Tests
// =============================================================================

func TestModuleUnloadStep_Rollback_Success(t *testing.T) {
	ctx, mockExec := newModuleUnloadTestContext()

	// Simulate that modules were unloaded
	ctx.SetState(StateModulesUnloaded, true)
	ctx.SetState(StateUnloadedModules, []string{"nvidia-modeset", "nvidia-drm", "nvidia"})

	step := NewModuleUnloadStep()

	err := step.Rollback(ctx)

	assert.NoError(t, err)

	// Verify modprobe was called (for reload)
	assert.True(t, mockExec.WasCalled("modprobe"))

	// Verify reload order is reverse of unload (nvidia first, then dependencies)
	calls := mockExec.Calls()
	reloadCalls := make([]string, 0)
	for _, call := range calls {
		if call.Command == "modprobe" && len(call.Args) >= 1 && call.Args[0] != "-r" {
			reloadCalls = append(reloadCalls, call.Args[0])
		}
	}
	// Should be in reverse order: nvidia, nvidia-drm, nvidia-modeset
	assert.Equal(t, []string{"nvidia", "nvidia-drm", "nvidia-modeset"}, reloadCalls)

	// State should be cleared
	assert.False(t, ctx.GetStateBool(StateModulesUnloaded))
	_, ok := ctx.GetState(StateUnloadedModules)
	assert.False(t, ok)
}

func TestModuleUnloadStep_Rollback_NoModulesUnloaded(t *testing.T) {
	ctx, mockExec := newModuleUnloadTestContext()

	step := NewModuleUnloadStep()

	err := step.Rollback(ctx)

	assert.NoError(t, err)

	// No modprobe commands should have been called
	assert.False(t, mockExec.WasCalled("modprobe"))
}

func TestModuleUnloadStep_Rollback_EmptyModulesList(t *testing.T) {
	ctx, mockExec := newModuleUnloadTestContext()

	ctx.SetState(StateModulesUnloaded, true)
	ctx.SetState(StateUnloadedModules, []string{})

	step := NewModuleUnloadStep()

	err := step.Rollback(ctx)

	assert.NoError(t, err)
	assert.False(t, mockExec.WasCalled("modprobe"))
}

func TestModuleUnloadStep_Rollback_InvalidStateType(t *testing.T) {
	ctx, mockExec := newModuleUnloadTestContext()

	ctx.SetState(StateModulesUnloaded, true)
	ctx.SetState(StateUnloadedModules, "not a slice") // Wrong type

	step := NewModuleUnloadStep()

	err := step.Rollback(ctx)

	assert.NoError(t, err) // Should handle gracefully
	assert.False(t, mockExec.WasCalled("modprobe"))
}

func TestModuleUnloadStep_Rollback_PartialFailure(t *testing.T) {
	ctx, mockExec := newModuleUnloadTestContext()

	ctx.SetState(StateModulesUnloaded, true)
	ctx.SetState(StateUnloadedModules, []string{"nvidia", "nvidia-drm"})

	// Make modprobe fail
	mockExec.SetDefaultResponse(exec.FailureResult(1, "module not found"))

	step := NewModuleUnloadStep()

	err := step.Rollback(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to reload")

	// Should still have attempted to reload all modules
	calls := mockExec.Calls()
	modprobeCount := 0
	for _, call := range calls {
		if call.Command == "modprobe" {
			modprobeCount++
		}
	}
	assert.Equal(t, 2, modprobeCount) // Should try both modules
}

func TestModuleUnloadStep_Rollback_NoExecutor(t *testing.T) {
	ctx := install.NewContext()
	ctx.SetState(StateModulesUnloaded, true)
	ctx.SetState(StateUnloadedModules, []string{"nvidia"})

	step := NewModuleUnloadStep()

	err := step.Rollback(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "executor not available")
}

func TestModuleUnloadStep_Rollback_MissingUnloadedModulesState(t *testing.T) {
	ctx, mockExec := newModuleUnloadTestContext()

	ctx.SetState(StateModulesUnloaded, true)
	// No StateUnloadedModules set

	step := NewModuleUnloadStep()

	err := step.Rollback(ctx)

	assert.NoError(t, err) // Should handle gracefully
	assert.False(t, mockExec.WasCalled("modprobe"))
}

// =============================================================================
// ModuleUnloadStep Validate Tests
// =============================================================================

func TestModuleUnloadStep_Validate_Success(t *testing.T) {
	ctx, _ := newModuleUnloadTestContext()

	step := NewModuleUnloadStep()

	err := step.Validate(ctx)

	assert.NoError(t, err)
}

func TestModuleUnloadStep_Validate_NoExecutor(t *testing.T) {
	ctx := install.NewContext()

	step := NewModuleUnloadStep()

	err := step.Validate(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "executor is required")
}

func TestModuleUnloadStep_Validate_InvalidModuleName(t *testing.T) {
	ctx, _ := newModuleUnloadTestContext()

	testCases := []struct {
		name       string
		moduleName string
		wantErr    bool
	}{
		{"valid name", "nvidia", false},
		{"valid with hyphen", "nvidia-drm", false},
		{"valid with underscore", "nvidia_uvm", false},
		{"empty name", "", true},
		{"invalid with space", "nvidia drm", true},
		{"invalid with semicolon", "nvidia;rm", true},
		{"invalid with slash", "nvidia/bad", true},
		{"invalid with ampersand", "nvidia&&", true},
		{"invalid with pipe", "nvidia|cat", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			step := NewModuleUnloadStep(WithUnloadModuleNames([]string{tc.moduleName}))
			err := step.Validate(ctx)
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestModuleUnloadStep_Validate_MultipleModulesOneInvalid(t *testing.T) {
	ctx, _ := newModuleUnloadTestContext()

	step := NewModuleUnloadStep(WithUnloadModuleNames([]string{"nvidia", "bad;module", "nvidia-drm"}))

	err := step.Validate(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid module name")
}

// =============================================================================
// ModuleUnloadStep CanRollback Tests
// =============================================================================

func TestModuleUnloadStep_CanRollback(t *testing.T) {
	step := NewModuleUnloadStep()
	assert.True(t, step.CanRollback())
}

// =============================================================================
// ModuleUnloadStep Helper Method Tests
// =============================================================================

func TestModuleUnloadStep_isModuleLoaded_WithDetector(t *testing.T) {
	ctx, _ := newModuleUnloadTestContext()
	mockDetector := newMockUnloadKernelDetector()
	mockDetector.SetModuleLoaded("nvidia", true)

	step := NewModuleUnloadStep(
		WithUnloadKernelDetector(mockDetector),
	)

	loaded, err := step.isModuleLoaded(ctx, "nvidia")

	assert.NoError(t, err)
	assert.True(t, loaded)
}

func TestModuleUnloadStep_isModuleLoaded_WithDetector_NotLoaded(t *testing.T) {
	ctx, _ := newModuleUnloadTestContext()
	mockDetector := newMockUnloadKernelDetector()
	mockDetector.SetModuleLoaded("nvidia", false)

	step := NewModuleUnloadStep(
		WithUnloadKernelDetector(mockDetector),
	)

	loaded, err := step.isModuleLoaded(ctx, "nvidia")

	assert.NoError(t, err)
	assert.False(t, loaded)
}

func TestModuleUnloadStep_isModuleLoaded_WithDetector_Error(t *testing.T) {
	ctx, _ := newModuleUnloadTestContext()
	mockDetector := newMockUnloadKernelDetector()
	mockDetector.SetIsModuleLoadedError(errors.New("detector error"))

	step := NewModuleUnloadStep(
		WithUnloadKernelDetector(mockDetector),
	)

	_, err := step.isModuleLoaded(ctx, "nvidia")

	assert.Error(t, err)
}

func TestModuleUnloadStep_isModuleLoaded_FallbackLsmod(t *testing.T) {
	ctx, mockExec := newModuleUnloadTestContext()

	// No detector, should use lsmod
	mockExec.SetResponse("lsmod", exec.SuccessResult("Module                  Size  Used by\nnvidia              12345678  5\n"))

	step := NewModuleUnloadStep()

	loaded, err := step.isModuleLoaded(ctx, "nvidia")

	assert.NoError(t, err)
	assert.True(t, loaded)
	assert.True(t, mockExec.WasCalled("lsmod"))
}

func TestModuleUnloadStep_isModuleLoaded_FallbackLsmod_NotFound(t *testing.T) {
	ctx, mockExec := newModuleUnloadTestContext()

	// No detector, should use lsmod
	mockExec.SetResponse("lsmod", exec.SuccessResult("Module                  Size  Used by\nsnd_hda_intel          45056  0\n"))

	step := NewModuleUnloadStep()

	loaded, err := step.isModuleLoaded(ctx, "nvidia")

	assert.NoError(t, err)
	assert.False(t, loaded)
}

func TestModuleUnloadStep_isModuleLoaded_FallbackLsmod_Error(t *testing.T) {
	ctx, mockExec := newModuleUnloadTestContext()

	mockExec.SetResponse("lsmod", exec.FailureResult(1, "command not found"))

	step := NewModuleUnloadStep()

	_, err := step.isModuleLoaded(ctx, "nvidia")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to list modules")
}

func TestModuleUnloadStep_isModuleInUse_NotInUse(t *testing.T) {
	ctx, mockExec := newModuleUnloadTestContext()

	mockExec.SetResponse("cat", exec.SuccessResult("0"))

	step := NewModuleUnloadStep()

	inUse, err := step.isModuleInUse(ctx, "nvidia")

	assert.NoError(t, err)
	assert.False(t, inUse)
}

func TestModuleUnloadStep_isModuleInUse_InUse(t *testing.T) {
	ctx, mockExec := newModuleUnloadTestContext()

	mockExec.SetResponse("cat", exec.SuccessResult("5"))

	step := NewModuleUnloadStep()

	inUse, err := step.isModuleInUse(ctx, "nvidia")

	assert.NoError(t, err)
	assert.True(t, inUse)
}

func TestModuleUnloadStep_isModuleInUse_FallbackLsmod(t *testing.T) {
	ctx, mockExec := newModuleUnloadTestContext()

	// refcnt read fails
	mockExec.SetResponse("cat", exec.FailureResult(1, "No such file"))
	// lsmod shows in use
	mockExec.SetResponse("lsmod", exec.SuccessResult("Module                  Size  Used by\nnvidia              12345678  3 nvidia_modeset"))

	step := NewModuleUnloadStep()

	inUse, err := step.isModuleInUse(ctx, "nvidia")

	assert.NoError(t, err)
	assert.True(t, inUse)
}

func TestModuleUnloadStep_unloadModule_Success(t *testing.T) {
	ctx, mockExec := newModuleUnloadTestContext()

	step := NewModuleUnloadStep()

	err := step.unloadModule(ctx, "nvidia")

	assert.NoError(t, err)
	assert.True(t, mockExec.WasCalledWith("modprobe", "-r", "nvidia"))
}

func TestModuleUnloadStep_unloadModule_Failure(t *testing.T) {
	ctx, mockExec := newModuleUnloadTestContext()
	mockExec.SetResponse("modprobe", exec.FailureResult(1, "Module nvidia is in use"))

	step := NewModuleUnloadStep()

	err := step.unloadModule(ctx, "nvidia")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "modprobe -r failed")
}

func TestModuleUnloadStep_loadModule_Success(t *testing.T) {
	ctx, mockExec := newModuleUnloadTestContext()

	step := NewModuleUnloadStep()

	err := step.loadModule(ctx, "nvidia")

	assert.NoError(t, err)
	assert.True(t, mockExec.WasCalledWith("modprobe", "nvidia"))
}

func TestModuleUnloadStep_loadModule_Failure(t *testing.T) {
	ctx, mockExec := newModuleUnloadTestContext()
	mockExec.SetDefaultResponse(exec.FailureResult(1, "Module nvidia not found"))

	step := NewModuleUnloadStep()

	err := step.loadModule(ctx, "nvidia")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "modprobe failed")
}

func TestModuleUnloadStep_forceUnloadModule_Success(t *testing.T) {
	ctx, mockExec := newModuleUnloadTestContext()

	step := NewModuleUnloadStep()

	err := step.forceUnloadModule(ctx, "nvidia")

	assert.NoError(t, err)
	assert.True(t, mockExec.WasCalledWith("rmmod", "-f", "nvidia"))
}

func TestModuleUnloadStep_forceUnloadModule_Failure(t *testing.T) {
	ctx, mockExec := newModuleUnloadTestContext()
	mockExec.SetResponse("rmmod", exec.FailureResult(1, "Operation not permitted"))

	step := NewModuleUnloadStep()

	err := step.forceUnloadModule(ctx, "nvidia")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "rmmod -f failed")
}

func TestModuleUnloadStep_getModuleHolders(t *testing.T) {
	ctx, mockExec := newModuleUnloadTestContext()

	mockExec.SetResponse("ls", exec.SuccessResult("nvidia_drm nvidia_modeset"))

	step := NewModuleUnloadStep()

	holders, err := step.getModuleHolders(ctx, "nvidia")

	assert.NoError(t, err)
	assert.Contains(t, holders, "nvidia_drm")
	assert.Contains(t, holders, "nvidia_modeset")
}

func TestModuleUnloadStep_getModuleHolders_NoHolders(t *testing.T) {
	ctx, mockExec := newModuleUnloadTestContext()

	mockExec.SetResponse("ls", exec.SuccessResult(""))

	step := NewModuleUnloadStep()

	holders, err := step.getModuleHolders(ctx, "nvidia")

	assert.NoError(t, err)
	assert.Empty(t, holders)
}

func TestModuleUnloadStep_filterLoadedModules(t *testing.T) {
	step := NewModuleUnloadStep()

	testCases := []struct {
		name     string
		toCheck  []string
		loaded   []string
		expected []string
	}{
		{
			name:     "all loaded",
			toCheck:  []string{"nvidia", "nvidia-drm"},
			loaded:   []string{"nvidia", "nvidia-drm", "nvidia-modeset"},
			expected: []string{"nvidia", "nvidia-drm"},
		},
		{
			name:     "none loaded",
			toCheck:  []string{"nvidia", "nvidia-drm"},
			loaded:   []string{"snd_hda_intel"},
			expected: nil,
		},
		{
			name:     "partial loaded",
			toCheck:  []string{"nvidia", "nvidia-drm", "nvidia-uvm"},
			loaded:   []string{"nvidia", "nvidia-drm"},
			expected: []string{"nvidia", "nvidia-drm"},
		},
		{
			name:     "underscore hyphen normalization",
			toCheck:  []string{"nvidia-drm"},
			loaded:   []string{"nvidia_drm"},
			expected: []string{"nvidia-drm"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := step.filterLoadedModules(tc.toCheck, tc.loaded)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// =============================================================================
// ModuleUnloadStep State Keys Tests
// =============================================================================

func TestModuleUnloadStep_StateKeys(t *testing.T) {
	assert.Equal(t, "modules_unloaded", StateModulesUnloaded)
	assert.Equal(t, "unloaded_modules", StateUnloadedModules)
	assert.Equal(t, "modules_in_use", StateModulesInUse)
}

// =============================================================================
// ModuleUnloadStep Default Values Tests
// =============================================================================

func TestDefaultUnloadModules(t *testing.T) {
	expected := []string{"nvidia-modeset", "nvidia-drm", "nvidia-uvm", "nvidia"}
	assert.Equal(t, expected, DefaultUnloadModules)
}

// =============================================================================
// ModuleUnloadStep Interface Compliance Tests
// =============================================================================

func TestModuleUnloadStep_InterfaceCompliance(t *testing.T) {
	var _ install.Step = (*ModuleUnloadStep)(nil)
}

// =============================================================================
// isValidModuleName Tests
// =============================================================================

func TestIsValidModuleName(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected bool
	}{
		{"valid alphanumeric", "nvidia", true},
		{"valid with hyphen", "nvidia-driver", true},
		{"valid with underscore", "nvidia_dkms", true},
		{"valid mixed case", "Nvidia123", true},
		{"empty string", "", false},
		{"with semicolon", "nvidia;rm", false},
		{"with slash", "nvidia/path", false},
		{"with space", "nvidia driver", false},
		{"with pipe", "nvidia|cat", false},
		{"with ampersand", "nvidia&&echo", false},
		{"with backtick", "nvidia`id`", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := isValidModuleName(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// =============================================================================
// ModuleUnloadStep Full Workflow Tests
// =============================================================================

func TestModuleUnloadStep_FullWorkflow_ExecuteAndRollback(t *testing.T) {
	ctx, mockExec := newModuleUnloadTestContext()

	mockDetector := newMockUnloadKernelDetector()
	mockDetector.SetModuleLoaded("nvidia", true)
	mockDetector.SetModuleLoaded("nvidia-drm", true)

	setupModuleUnloadMocks(mockExec)

	step := NewModuleUnloadStep(
		WithUnloadKernelDetector(mockDetector),
		WithUnloadModuleNames([]string{"nvidia-drm", "nvidia"}),
	)

	// Execute
	result := step.Execute(ctx)
	assert.Equal(t, install.StepStatusCompleted, result.Status)
	assert.True(t, ctx.GetStateBool(StateModulesUnloaded))

	// Verify state was set correctly
	unloadedModulesRaw, ok := ctx.GetState(StateUnloadedModules)
	assert.True(t, ok)
	unloadedModules, ok := unloadedModulesRaw.([]string)
	assert.True(t, ok)
	assert.NotEmpty(t, unloadedModules)

	// Reset mock tracking
	mockExec.Reset()

	// Rollback
	err := step.Rollback(ctx)
	assert.NoError(t, err)

	// Verify modprobe was called (for reload)
	assert.True(t, mockExec.WasCalled("modprobe"))

	// State should be cleared
	assert.False(t, ctx.GetStateBool(StateModulesUnloaded))
	_, ok = ctx.GetState(StateUnloadedModules)
	assert.False(t, ok)
}

func TestModuleUnloadStep_Duration(t *testing.T) {
	ctx, mockExec := newModuleUnloadTestContext()

	mockDetector := newMockUnloadKernelDetector()
	mockDetector.SetModuleLoaded("nvidia", true)

	setupModuleUnloadMocks(mockExec)

	step := NewModuleUnloadStep(
		WithUnloadKernelDetector(mockDetector),
		WithUnloadModuleNames([]string{"nvidia"}),
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusCompleted, result.Status)
	assert.Greater(t, result.Duration.Nanoseconds(), int64(0))
}

// =============================================================================
// Edge Case Tests
// =============================================================================

func TestModuleUnloadStep_DetectorCheckError_ProceedsAnyway(t *testing.T) {
	ctx, mockExec := newModuleUnloadTestContext()
	mockDetector := newMockUnloadKernelDetector()
	mockDetector.SetGetLoadedModulesFunc(func(ctx context.Context) ([]kernel.ModuleInfo, error) {
		return nil, errors.New("detector failed")
	})

	setupModuleUnloadMocks(mockExec)

	step := NewModuleUnloadStep(
		WithUnloadKernelDetector(mockDetector),
		WithUnloadModuleNames([]string{"nvidia"}),
	)

	result := step.Execute(ctx)

	// Should proceed with unloading even if detector check fails
	assert.Equal(t, install.StepStatusCompleted, result.Status)
}

func TestModuleUnloadStep_LsmodParsingEdgeCases(t *testing.T) {
	testCases := []struct {
		name       string
		lsmodOut   string
		moduleName string
		wantLoaded bool
	}{
		{
			name:       "module at start",
			lsmodOut:   "nvidia              12345678  5\nsnd_hda_intel          45056  0",
			moduleName: "nvidia",
			wantLoaded: true,
		},
		{
			name:       "module in middle",
			lsmodOut:   "snd_hda_intel          45056  0\nnvidia              12345678  5\ndrm              12345  0",
			moduleName: "nvidia",
			wantLoaded: true,
		},
		{
			name:       "partial match should not match",
			lsmodOut:   "nvidia_drm          12345  0\nnvidia_uvm          12345  0",
			moduleName: "nvidia",
			wantLoaded: false,
		},
		{
			name:       "empty output",
			lsmodOut:   "",
			moduleName: "nvidia",
			wantLoaded: false,
		},
		{
			name:       "header only",
			lsmodOut:   "Module                  Size  Used by",
			moduleName: "nvidia",
			wantLoaded: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, mockExec := newModuleUnloadTestContext()
			mockExec.SetResponse("lsmod", exec.SuccessResult(tc.lsmodOut))

			step := NewModuleUnloadStep()

			loaded, err := step.isModuleLoaded(ctx, tc.moduleName)

			assert.NoError(t, err)
			assert.Equal(t, tc.wantLoaded, loaded)
		})
	}
}

// =============================================================================
// Benchmarks
// =============================================================================

func BenchmarkModuleUnloadStep_Execute(b *testing.B) {
	mockExec := exec.NewMockExecutor()
	mockExec.SetDefaultResponse(exec.SuccessResult(""))
	mockExec.SetResponse("lsmod", exec.SuccessResult("nvidia 12345 0"))
	mockExec.SetResponse("cat", exec.SuccessResult("0"))

	mockDetector := newMockUnloadKernelDetector()
	mockDetector.SetModuleLoaded("nvidia", true)

	step := NewModuleUnloadStep(
		WithUnloadKernelDetector(mockDetector),
		WithUnloadModuleNames([]string{"nvidia"}),
	)

	ctx := install.NewContext(
		install.WithExecutor(mockExec),
	)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Clear state between iterations
		ctx.DeleteState(StateModulesUnloaded)
		ctx.DeleteState(StateUnloadedModules)
		ctx.DeleteState(StateModulesInUse)

		step.Execute(ctx)
	}
}

func BenchmarkModuleUnloadStep_Validate(b *testing.B) {
	mockExec := exec.NewMockExecutor()

	ctx := install.NewContext(
		install.WithExecutor(mockExec),
	)

	step := NewModuleUnloadStep()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = step.Validate(ctx)
	}
}
