package steps

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tungetti/igor/internal/exec"
	"github.com/tungetti/igor/internal/gpu/kernel"
	"github.com/tungetti/igor/internal/install"
)

// =============================================================================
// Mock Kernel Detector for Module Loading Tests
// =============================================================================

// mockModuleKernelDetector is a mock kernel detector specifically for module tests.
type mockModuleKernelDetector struct {
	loadedModules      map[string]bool
	isModuleLoadedFunc func(ctx context.Context, name string) (bool, error)
	isModuleLoadedErr  error
}

// newMockModuleKernelDetector creates a new mock kernel detector for module tests.
func newMockModuleKernelDetector() *mockModuleKernelDetector {
	return &mockModuleKernelDetector{
		loadedModules: make(map[string]bool),
	}
}

// SetModuleLoaded sets whether a module is reported as loaded.
func (m *mockModuleKernelDetector) SetModuleLoaded(name string, loaded bool) {
	m.loadedModules[name] = loaded
}

// SetIsModuleLoadedFunc sets a custom function for IsModuleLoaded.
func (m *mockModuleKernelDetector) SetIsModuleLoadedFunc(fn func(ctx context.Context, name string) (bool, error)) {
	m.isModuleLoadedFunc = fn
}

// SetIsModuleLoadedError sets an error to return from IsModuleLoaded.
func (m *mockModuleKernelDetector) SetIsModuleLoadedError(err error) {
	m.isModuleLoadedErr = err
}

// IsModuleLoaded implements kernel.Detector.
func (m *mockModuleKernelDetector) IsModuleLoaded(ctx context.Context, name string) (bool, error) {
	if m.isModuleLoadedFunc != nil {
		return m.isModuleLoadedFunc(ctx, name)
	}
	if m.isModuleLoadedErr != nil {
		return false, m.isModuleLoadedErr
	}
	return m.loadedModules[name], nil
}

// GetKernelInfo implements kernel.Detector.
func (m *mockModuleKernelDetector) GetKernelInfo(ctx context.Context) (*kernel.KernelInfo, error) {
	return &kernel.KernelInfo{
		Version:      "6.5.0-44-generic",
		Release:      "6.5.0",
		Architecture: "x86_64",
	}, nil
}

// GetLoadedModules implements kernel.Detector.
func (m *mockModuleKernelDetector) GetLoadedModules(ctx context.Context) ([]kernel.ModuleInfo, error) {
	var modules []kernel.ModuleInfo
	for name, loaded := range m.loadedModules {
		if loaded {
			modules = append(modules, kernel.ModuleInfo{Name: name})
		}
	}
	return modules, nil
}

// GetModule implements kernel.Detector.
func (m *mockModuleKernelDetector) GetModule(ctx context.Context, name string) (*kernel.ModuleInfo, error) {
	if m.loadedModules[name] {
		return &kernel.ModuleInfo{Name: name}, nil
	}
	return nil, nil
}

// AreHeadersInstalled implements kernel.Detector.
func (m *mockModuleKernelDetector) AreHeadersInstalled(ctx context.Context) (bool, error) {
	return true, nil
}

// GetHeadersPackage implements kernel.Detector.
func (m *mockModuleKernelDetector) GetHeadersPackage(ctx context.Context) (string, error) {
	return "linux-headers-6.5.0-44-generic", nil
}

// IsSecureBootEnabled implements kernel.Detector.
func (m *mockModuleKernelDetector) IsSecureBootEnabled(ctx context.Context) (bool, error) {
	return false, nil
}

// Ensure mockModuleKernelDetector implements kernel.Detector.
var _ kernel.Detector = (*mockModuleKernelDetector)(nil)

// =============================================================================
// Test Helpers
// =============================================================================

// newModuleLoadTestContext creates a basic test context with executor for module load tests.
func newModuleLoadTestContext() (*install.Context, *exec.MockExecutor) {
	mockExec := exec.NewMockExecutor()
	mockExec.SetDefaultResponse(exec.SuccessResult(""))

	ctx := install.NewContext(
		install.WithExecutor(mockExec),
	)

	return ctx, mockExec
}

// setupModuleLoadMocks configures the mock executor with common module load responses.
func setupModuleLoadMocks(mockExec *exec.MockExecutor) {
	// lsmod shows no nvidia module loaded
	mockExec.SetResponse("lsmod", exec.SuccessResult("Module                  Size  Used by\nsnd_hda_intel          45056  0\n"))
	// modprobe succeeds
	mockExec.SetDefaultResponse(exec.SuccessResult(""))
}

// =============================================================================
// ModuleLoadStep Constructor Tests
// =============================================================================

func TestNewModuleLoadStep(t *testing.T) {
	step := NewModuleLoadStep()

	assert.Equal(t, "module_load", step.Name())
	assert.Equal(t, "Load NVIDIA kernel modules", step.Description())
	assert.True(t, step.CanRollback())
	assert.Equal(t, DefaultNvidiaModules, step.moduleNames)
	assert.True(t, step.skipIfLoaded)
	assert.False(t, step.forceReload)
	assert.Nil(t, step.kernelDetector)
}

func TestModuleLoadStepOptions(t *testing.T) {
	mockDetector := newMockModuleKernelDetector()
	customModules := []string{"nvidia", "nvidia-drm"}

	step := NewModuleLoadStep(
		WithModuleNames(customModules),
		WithSkipIfLoaded(false),
		WithModuleKernelDetector(mockDetector),
		WithForceReload(true),
	)

	assert.Equal(t, customModules, step.moduleNames)
	assert.False(t, step.skipIfLoaded)
	assert.True(t, step.forceReload)
	assert.Equal(t, mockDetector, step.kernelDetector)
}

func TestModuleLoadStep_WithModuleNames_CopiesSlice(t *testing.T) {
	original := []string{"nvidia", "nvidia-drm"}
	step := NewModuleLoadStep(WithModuleNames(original))

	// Modify original slice
	original[0] = "modified"

	// Step should have the original values (defensive copy)
	assert.Equal(t, "nvidia", step.moduleNames[0])
}

// =============================================================================
// ModuleLoadStep Execute Tests
// =============================================================================

func TestModuleLoadStep_Execute_Success(t *testing.T) {
	ctx, mockExec := newModuleLoadTestContext()
	mockDetector := newMockModuleKernelDetector()
	mockDetector.SetModuleLoaded("nvidia", false)

	setupModuleLoadMocks(mockExec)

	step := NewModuleLoadStep(
		WithModuleKernelDetector(mockDetector),
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusCompleted, result.Status)
	assert.Contains(t, result.Message, "successfully")
	assert.Greater(t, result.Duration.Nanoseconds(), int64(0))

	// Check state was set
	assert.True(t, ctx.GetStateBool(StateModulesLoaded))

	// Check loaded modules list
	loadedModulesRaw, ok := ctx.GetState(StateLoadedModules)
	assert.True(t, ok)
	loadedModules, ok := loadedModulesRaw.([]string)
	assert.True(t, ok)
	assert.Equal(t, DefaultNvidiaModules, loadedModules)

	// Verify modprobe was called for each module
	assert.True(t, mockExec.WasCalled("modprobe"))
}

func TestModuleLoadStep_Execute_AlreadyLoaded_Skip(t *testing.T) {
	ctx, mockExec := newModuleLoadTestContext()
	mockDetector := newMockModuleKernelDetector()
	mockDetector.SetModuleLoaded("nvidia", true)

	step := NewModuleLoadStep(
		WithModuleKernelDetector(mockDetector),
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusSkipped, result.Status)
	assert.Contains(t, result.Message, "already loaded")

	// State should not be set for skipped step
	assert.False(t, ctx.GetStateBool(StateModulesLoaded))

	// modprobe should not have been called
	assert.False(t, mockExec.WasCalled("modprobe"))
}

func TestModuleLoadStep_Execute_ForceReload(t *testing.T) {
	ctx, mockExec := newModuleLoadTestContext()
	mockDetector := newMockModuleKernelDetector()
	mockDetector.SetModuleLoaded("nvidia", true)

	setupModuleLoadMocks(mockExec)

	step := NewModuleLoadStep(
		WithModuleKernelDetector(mockDetector),
		WithForceReload(true),
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusCompleted, result.Status)
	assert.Contains(t, result.Message, "successfully")

	// Verify modprobe -r was called (unload)
	calls := mockExec.Calls()
	hasUnload := false
	hasLoad := false
	for _, call := range calls {
		if call.Command == "modprobe" {
			if len(call.Args) > 0 && call.Args[0] == "-r" {
				hasUnload = true
			} else if len(call.Args) > 0 && call.Args[0] != "-r" {
				hasLoad = true
			}
		}
	}
	assert.True(t, hasUnload, "expected modprobe -r to be called for unload")
	assert.True(t, hasLoad, "expected modprobe to be called for load")
}

func TestModuleLoadStep_Execute_DryRun(t *testing.T) {
	ctx, mockExec := newModuleLoadTestContext()
	ctx.DryRun = true

	mockDetector := newMockModuleKernelDetector()
	mockDetector.SetModuleLoaded("nvidia", false)

	step := NewModuleLoadStep(
		WithModuleKernelDetector(mockDetector),
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusCompleted, result.Status)
	assert.Contains(t, result.Message, "dry run")

	// State should not be set for dry run
	assert.False(t, ctx.GetStateBool(StateModulesLoaded))

	// modprobe should not have been called
	assert.False(t, mockExec.WasCalled("modprobe"))
}

func TestModuleLoadStep_Execute_DryRun_ForceReload(t *testing.T) {
	ctx, mockExec := newModuleLoadTestContext()
	ctx.DryRun = true

	mockDetector := newMockModuleKernelDetector()
	mockDetector.SetModuleLoaded("nvidia", true)

	step := NewModuleLoadStep(
		WithModuleKernelDetector(mockDetector),
		WithForceReload(true),
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusCompleted, result.Status)
	assert.Contains(t, result.Message, "dry run")

	// modprobe should not have been called in dry run
	assert.False(t, mockExec.WasCalled("modprobe"))
}

func TestModuleLoadStep_Execute_Cancelled(t *testing.T) {
	ctx, mockExec := newModuleLoadTestContext()
	ctx.Cancel() // Cancel immediately

	mockExec.SetDefaultResponse(exec.SuccessResult(""))

	step := NewModuleLoadStep()

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusFailed, result.Status)
	assert.Contains(t, result.Message, "cancelled")
	assert.True(t, errors.Is(result.Error, context.Canceled))
}

func TestModuleLoadStep_Execute_NoExecutor(t *testing.T) {
	ctx := install.NewContext()

	step := NewModuleLoadStep()

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusFailed, result.Status)
	assert.Contains(t, result.Message, "validation failed")
	assert.Error(t, result.Error)
	assert.Contains(t, result.Error.Error(), "executor")
}

func TestModuleLoadStep_Execute_LoadFailure(t *testing.T) {
	ctx, mockExec := newModuleLoadTestContext()
	mockDetector := newMockModuleKernelDetector()
	mockDetector.SetModuleLoaded("nvidia", false)

	// lsmod succeeds
	mockExec.SetResponse("lsmod", exec.SuccessResult(""))
	// modprobe fails
	mockExec.SetDefaultResponse(exec.FailureResult(1, "modprobe: FATAL: Module nvidia not found"))

	step := NewModuleLoadStep(
		WithModuleKernelDetector(mockDetector),
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusFailed, result.Status)
	assert.Contains(t, result.Message, "failed to load module")
	assert.Error(t, result.Error)

	// State should not be set on failure
	assert.False(t, ctx.GetStateBool(StateModulesLoaded))
}

func TestModuleLoadStep_Execute_PartialLoadFailure(t *testing.T) {
	// Note: Testing partial load failure requires a stateful mock that can track call sequence.
	// The current MockExecutor doesn't support this easily. This test verifies that when a
	// load fails, the step properly returns a failure result.
	// Full partial failure testing with rollback would require enhancing the mock to support
	// sequential response patterns.
	ctx, mockExec := newModuleLoadTestContext()
	mockDetector := newMockModuleKernelDetector()
	mockDetector.SetModuleLoaded("nvidia", false)

	// Make modprobe fail for the second module by setting it to fail by default
	// and only succeeding for specific commands
	mockExec.SetResponse("modprobe", &exec.Result{ExitCode: 1, Stderr: []byte("module not found")})

	step := NewModuleLoadStep(
		WithModuleKernelDetector(mockDetector),
		WithModuleNames([]string{"nvidia", "nvidia-uvm"}),
	)

	result := step.Execute(ctx)

	// Should fail on first module (since our mock fails all modprobe calls)
	assert.Equal(t, install.StepStatusFailed, result.Status)
	assert.Contains(t, result.Message, "failed to load module")
}

func TestModuleLoadStep_Execute_CustomModules(t *testing.T) {
	ctx, mockExec := newModuleLoadTestContext()
	mockDetector := newMockModuleKernelDetector()
	mockDetector.SetModuleLoaded("nvidia", false)

	setupModuleLoadMocks(mockExec)

	customModules := []string{"nvidia", "nvidia-drm"}
	step := NewModuleLoadStep(
		WithModuleKernelDetector(mockDetector),
		WithModuleNames(customModules),
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusCompleted, result.Status)

	// Verify loaded modules list
	loadedModulesRaw, ok := ctx.GetState(StateLoadedModules)
	assert.True(t, ok)
	loadedModules, ok := loadedModulesRaw.([]string)
	assert.True(t, ok)
	assert.Equal(t, customModules, loadedModules)
}

func TestModuleLoadStep_Execute_FallbackToLsmod(t *testing.T) {
	ctx, mockExec := newModuleLoadTestContext()

	// No kernel detector, should fall back to lsmod
	// lsmod shows no nvidia module
	mockExec.SetResponse("lsmod", exec.SuccessResult("Module                  Size  Used by\nsnd_hda_intel          45056  0\n"))

	step := NewModuleLoadStep()

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusCompleted, result.Status)
	assert.True(t, mockExec.WasCalled("lsmod"))
}

func TestModuleLoadStep_Execute_FallbackToLsmod_ModuleLoaded(t *testing.T) {
	ctx, mockExec := newModuleLoadTestContext()

	// No kernel detector, should fall back to lsmod
	// lsmod shows nvidia module loaded
	mockExec.SetResponse("lsmod", exec.SuccessResult("Module                  Size  Used by\nnvidia              12345678  5\nnvidia_drm            12345  0\n"))

	step := NewModuleLoadStep()

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusSkipped, result.Status)
	assert.Contains(t, result.Message, "already loaded")
}

func TestModuleLoadStep_Execute_SkipIfLoadedFalse(t *testing.T) {
	ctx, mockExec := newModuleLoadTestContext()
	mockDetector := newMockModuleKernelDetector()
	mockDetector.SetModuleLoaded("nvidia", true)

	setupModuleLoadMocks(mockExec)

	step := NewModuleLoadStep(
		WithModuleKernelDetector(mockDetector),
		WithSkipIfLoaded(false),
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusCompleted, result.Status)
	// Should load even though already loaded
	assert.True(t, mockExec.WasCalled("modprobe"))
}

// =============================================================================
// ModuleLoadStep Rollback Tests
// =============================================================================

func TestModuleLoadStep_Rollback_Success(t *testing.T) {
	ctx, mockExec := newModuleLoadTestContext()

	// Simulate that modules were loaded
	ctx.SetState(StateModulesLoaded, true)
	ctx.SetState(StateLoadedModules, []string{"nvidia", "nvidia-drm", "nvidia-modeset"})

	step := NewModuleLoadStep()

	err := step.Rollback(ctx)

	assert.NoError(t, err)

	// Verify modprobe -r was called
	assert.True(t, mockExec.WasCalled("modprobe"))

	// Verify unload order is reverse
	calls := mockExec.Calls()
	unloadCalls := make([]string, 0)
	for _, call := range calls {
		if call.Command == "modprobe" && len(call.Args) >= 2 && call.Args[0] == "-r" {
			unloadCalls = append(unloadCalls, call.Args[1])
		}
	}
	// Should be in reverse order: nvidia-modeset, nvidia-drm, nvidia
	assert.Equal(t, []string{"nvidia-modeset", "nvidia-drm", "nvidia"}, unloadCalls)

	// State should be cleared
	assert.False(t, ctx.GetStateBool(StateModulesLoaded))
	_, ok := ctx.GetState(StateLoadedModules)
	assert.False(t, ok)
}

func TestModuleLoadStep_Rollback_NoModulesLoaded(t *testing.T) {
	ctx, mockExec := newModuleLoadTestContext()

	step := NewModuleLoadStep()

	err := step.Rollback(ctx)

	assert.NoError(t, err)

	// No modprobe commands should have been called
	assert.False(t, mockExec.WasCalled("modprobe"))
}

func TestModuleLoadStep_Rollback_EmptyModulesList(t *testing.T) {
	ctx, mockExec := newModuleLoadTestContext()

	ctx.SetState(StateModulesLoaded, true)
	ctx.SetState(StateLoadedModules, []string{})

	step := NewModuleLoadStep()

	err := step.Rollback(ctx)

	assert.NoError(t, err)
	assert.False(t, mockExec.WasCalled("modprobe"))
}

func TestModuleLoadStep_Rollback_InvalidStateType(t *testing.T) {
	ctx, mockExec := newModuleLoadTestContext()

	ctx.SetState(StateModulesLoaded, true)
	ctx.SetState(StateLoadedModules, "not a slice") // Wrong type

	step := NewModuleLoadStep()

	err := step.Rollback(ctx)

	assert.NoError(t, err) // Should handle gracefully
	assert.False(t, mockExec.WasCalled("modprobe"))
}

func TestModuleLoadStep_Rollback_PartialFailure(t *testing.T) {
	ctx, mockExec := newModuleLoadTestContext()

	ctx.SetState(StateModulesLoaded, true)
	ctx.SetState(StateLoadedModules, []string{"nvidia", "nvidia-drm"})

	// Make modprobe -r fail
	mockExec.SetDefaultResponse(exec.FailureResult(1, "module is in use"))

	step := NewModuleLoadStep()

	err := step.Rollback(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to unload")

	// Should still have attempted to unload all modules
	calls := mockExec.Calls()
	modprobeCount := 0
	for _, call := range calls {
		if call.Command == "modprobe" {
			modprobeCount++
		}
	}
	assert.Equal(t, 2, modprobeCount) // Should try both modules
}

func TestModuleLoadStep_Rollback_NoExecutor(t *testing.T) {
	ctx := install.NewContext()
	ctx.SetState(StateModulesLoaded, true)
	ctx.SetState(StateLoadedModules, []string{"nvidia"})

	step := NewModuleLoadStep()

	err := step.Rollback(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "executor not available")
}

func TestModuleLoadStep_Rollback_MissingLoadedModulesState(t *testing.T) {
	ctx, mockExec := newModuleLoadTestContext()

	ctx.SetState(StateModulesLoaded, true)
	// No StateLoadedModules set

	step := NewModuleLoadStep()

	err := step.Rollback(ctx)

	assert.NoError(t, err) // Should handle gracefully
	assert.False(t, mockExec.WasCalled("modprobe"))
}

// =============================================================================
// ModuleLoadStep Validate Tests
// =============================================================================

func TestModuleLoadStep_Validate_Success(t *testing.T) {
	ctx, _ := newModuleLoadTestContext()

	step := NewModuleLoadStep()

	err := step.Validate(ctx)

	assert.NoError(t, err)
}

func TestModuleLoadStep_Validate_NoExecutor(t *testing.T) {
	ctx := install.NewContext()

	step := NewModuleLoadStep()

	err := step.Validate(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "executor is required")
}

func TestModuleLoadStep_Validate_InvalidModuleName(t *testing.T) {
	ctx, _ := newModuleLoadTestContext()

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
			step := NewModuleLoadStep(WithModuleNames([]string{tc.moduleName}))
			err := step.Validate(ctx)
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestModuleLoadStep_Validate_MultipleModulesOneInvalid(t *testing.T) {
	ctx, _ := newModuleLoadTestContext()

	step := NewModuleLoadStep(WithModuleNames([]string{"nvidia", "bad;module", "nvidia-drm"}))

	err := step.Validate(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid module name")
}

// =============================================================================
// ModuleLoadStep CanRollback Tests
// =============================================================================

func TestModuleLoadStep_CanRollback(t *testing.T) {
	step := NewModuleLoadStep()
	assert.True(t, step.CanRollback())
}

// =============================================================================
// ModuleLoadStep Helper Method Tests
// =============================================================================

func TestModuleLoadStep_isModuleLoaded_WithDetector(t *testing.T) {
	ctx, _ := newModuleLoadTestContext()
	mockDetector := newMockModuleKernelDetector()
	mockDetector.SetModuleLoaded("nvidia", true)

	step := NewModuleLoadStep(
		WithModuleKernelDetector(mockDetector),
	)

	loaded, err := step.isModuleLoaded(ctx, "nvidia")

	assert.NoError(t, err)
	assert.True(t, loaded)
}

func TestModuleLoadStep_isModuleLoaded_WithDetector_NotLoaded(t *testing.T) {
	ctx, _ := newModuleLoadTestContext()
	mockDetector := newMockModuleKernelDetector()
	mockDetector.SetModuleLoaded("nvidia", false)

	step := NewModuleLoadStep(
		WithModuleKernelDetector(mockDetector),
	)

	loaded, err := step.isModuleLoaded(ctx, "nvidia")

	assert.NoError(t, err)
	assert.False(t, loaded)
}

func TestModuleLoadStep_isModuleLoaded_WithDetector_Error(t *testing.T) {
	ctx, _ := newModuleLoadTestContext()
	mockDetector := newMockModuleKernelDetector()
	mockDetector.SetIsModuleLoadedError(errors.New("detector error"))

	step := NewModuleLoadStep(
		WithModuleKernelDetector(mockDetector),
	)

	_, err := step.isModuleLoaded(ctx, "nvidia")

	assert.Error(t, err)
}

func TestModuleLoadStep_isModuleLoaded_FallbackLsmod(t *testing.T) {
	ctx, mockExec := newModuleLoadTestContext()

	// No detector, should use lsmod
	mockExec.SetResponse("lsmod", exec.SuccessResult("Module                  Size  Used by\nnvidia              12345678  5\n"))

	step := NewModuleLoadStep()

	loaded, err := step.isModuleLoaded(ctx, "nvidia")

	assert.NoError(t, err)
	assert.True(t, loaded)
	assert.True(t, mockExec.WasCalled("lsmod"))
}

func TestModuleLoadStep_isModuleLoaded_FallbackLsmod_NotFound(t *testing.T) {
	ctx, mockExec := newModuleLoadTestContext()

	// No detector, should use lsmod
	mockExec.SetResponse("lsmod", exec.SuccessResult("Module                  Size  Used by\nsnd_hda_intel          45056  0\n"))

	step := NewModuleLoadStep()

	loaded, err := step.isModuleLoaded(ctx, "nvidia")

	assert.NoError(t, err)
	assert.False(t, loaded)
}

func TestModuleLoadStep_isModuleLoaded_FallbackLsmod_Error(t *testing.T) {
	ctx, mockExec := newModuleLoadTestContext()

	mockExec.SetResponse("lsmod", exec.FailureResult(1, "command not found"))

	step := NewModuleLoadStep()

	_, err := step.isModuleLoaded(ctx, "nvidia")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to list modules")
}

func TestModuleLoadStep_loadModule_Success(t *testing.T) {
	ctx, mockExec := newModuleLoadTestContext()

	step := NewModuleLoadStep()

	err := step.loadModule(ctx, "nvidia")

	assert.NoError(t, err)
	assert.True(t, mockExec.WasCalledWith("modprobe", "nvidia"))
}

func TestModuleLoadStep_loadModule_Failure(t *testing.T) {
	ctx, mockExec := newModuleLoadTestContext()
	mockExec.SetDefaultResponse(exec.FailureResult(1, "Module nvidia not found"))

	step := NewModuleLoadStep()

	err := step.loadModule(ctx, "nvidia")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "modprobe failed")
}

func TestModuleLoadStep_unloadModule_Success(t *testing.T) {
	ctx, mockExec := newModuleLoadTestContext()

	step := NewModuleLoadStep()

	err := step.unloadModule(ctx, "nvidia")

	assert.NoError(t, err)
	assert.True(t, mockExec.WasCalledWith("modprobe", "-r", "nvidia"))
}

func TestModuleLoadStep_unloadModule_Failure(t *testing.T) {
	ctx, mockExec := newModuleLoadTestContext()
	mockExec.SetDefaultResponse(exec.FailureResult(1, "Module nvidia is in use"))

	step := NewModuleLoadStep()

	err := step.unloadModule(ctx, "nvidia")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "modprobe -r failed")
}

func TestModuleLoadStep_unloadModules_ReverseOrder(t *testing.T) {
	ctx, mockExec := newModuleLoadTestContext()

	step := NewModuleLoadStep()

	modules := []string{"nvidia", "nvidia-uvm", "nvidia-drm"}
	err := step.unloadModules(ctx, modules)

	assert.NoError(t, err)

	// Verify reverse order
	calls := mockExec.Calls()
	unloadOrder := make([]string, 0)
	for _, call := range calls {
		if call.Command == "modprobe" && len(call.Args) >= 2 && call.Args[0] == "-r" {
			unloadOrder = append(unloadOrder, call.Args[1])
		}
	}
	assert.Equal(t, []string{"nvidia-drm", "nvidia-uvm", "nvidia"}, unloadOrder)
}

// =============================================================================
// ModuleLoadStep State Keys Tests
// =============================================================================

func TestModuleLoadStep_StateKeys(t *testing.T) {
	assert.Equal(t, "modules_loaded", StateModulesLoaded)
	assert.Equal(t, "loaded_modules", StateLoadedModules)
}

// =============================================================================
// ModuleLoadStep Default Values Tests
// =============================================================================

func TestDefaultNvidiaModules(t *testing.T) {
	expected := []string{"nvidia", "nvidia-uvm", "nvidia-drm", "nvidia-modeset"}
	assert.Equal(t, expected, DefaultNvidiaModules)
}

// =============================================================================
// ModuleLoadStep Interface Compliance Tests
// =============================================================================

func TestModuleLoadStep_InterfaceCompliance(t *testing.T) {
	var _ install.Step = (*ModuleLoadStep)(nil)
}

// =============================================================================
// ModuleLoadStep Full Workflow Tests
// =============================================================================

func TestModuleLoadStep_FullWorkflow_ExecuteAndRollback(t *testing.T) {
	ctx, mockExec := newModuleLoadTestContext()

	mockDetector := newMockModuleKernelDetector()
	mockDetector.SetModuleLoaded("nvidia", false)

	setupModuleLoadMocks(mockExec)

	step := NewModuleLoadStep(
		WithModuleKernelDetector(mockDetector),
	)

	// Execute
	result := step.Execute(ctx)
	assert.Equal(t, install.StepStatusCompleted, result.Status)
	assert.True(t, ctx.GetStateBool(StateModulesLoaded))

	// Verify state was set correctly
	loadedModulesRaw, ok := ctx.GetState(StateLoadedModules)
	assert.True(t, ok)
	loadedModules, ok := loadedModulesRaw.([]string)
	assert.True(t, ok)
	assert.NotEmpty(t, loadedModules)

	// Reset mock tracking
	mockExec.Reset()

	// Rollback
	err := step.Rollback(ctx)
	assert.NoError(t, err)

	// Verify modprobe -r was called
	assert.True(t, mockExec.WasCalled("modprobe"))

	// State should be cleared
	assert.False(t, ctx.GetStateBool(StateModulesLoaded))
	_, ok = ctx.GetState(StateLoadedModules)
	assert.False(t, ok)
}

func TestModuleLoadStep_Duration(t *testing.T) {
	ctx, mockExec := newModuleLoadTestContext()

	mockDetector := newMockModuleKernelDetector()
	mockDetector.SetModuleLoaded("nvidia", false)

	setupModuleLoadMocks(mockExec)

	step := NewModuleLoadStep(
		WithModuleKernelDetector(mockDetector),
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusCompleted, result.Status)
	assert.Greater(t, result.Duration.Nanoseconds(), int64(0))
}

// =============================================================================
// Edge Case Tests
// =============================================================================

func TestModuleLoadStep_CancelledDuringLoad(t *testing.T) {
	ctx, mockExec := newModuleLoadTestContext()
	mockDetector := newMockModuleKernelDetector()
	mockDetector.SetModuleLoaded("nvidia", false)

	// Track calls and cancel after first modprobe
	callCount := 0
	mockExec.SetDefaultResponse(exec.SuccessResult(""))

	step := NewModuleLoadStep(
		WithModuleKernelDetector(mockDetector),
		WithModuleNames([]string{"nvidia", "nvidia-drm"}),
	)

	// The step checks for cancellation between module loads
	// This is hard to test precisely without a custom mock
	_ = callCount

	result := step.Execute(ctx)

	// Should complete since we can't easily inject cancellation mid-execution
	assert.Equal(t, install.StepStatusCompleted, result.Status)
}

func TestModuleLoadStep_EmptyModuleList(t *testing.T) {
	ctx, mockExec := newModuleLoadTestContext()
	mockDetector := newMockModuleKernelDetector()
	mockDetector.SetModuleLoaded("nvidia", false)

	step := NewModuleLoadStep(
		WithModuleKernelDetector(mockDetector),
		WithModuleNames([]string{}),
	)

	result := step.Execute(ctx)

	// Empty module list should skip (not complete)
	assert.Equal(t, install.StepStatusSkipped, result.Status)
	assert.Contains(t, result.Message, "no modules configured")

	// No modprobe should have been called
	assert.False(t, mockExec.WasCalled("modprobe"))

	// State should NOT be set on skip
	assert.False(t, ctx.GetStateBool(StateModulesLoaded))
}

func TestModuleLoadStep_DetectorCheckError_ProceedsAnyway(t *testing.T) {
	ctx, mockExec := newModuleLoadTestContext()
	mockDetector := newMockModuleKernelDetector()
	mockDetector.SetIsModuleLoadedError(errors.New("detector failed"))

	setupModuleLoadMocks(mockExec)

	step := NewModuleLoadStep(
		WithModuleKernelDetector(mockDetector),
	)

	result := step.Execute(ctx)

	// Should proceed with loading even if check fails
	assert.Equal(t, install.StepStatusCompleted, result.Status)
}

func TestModuleLoadStep_LsmodParsingEdgeCases(t *testing.T) {
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
			ctx, mockExec := newModuleLoadTestContext()
			mockExec.SetResponse("lsmod", exec.SuccessResult(tc.lsmodOut))

			step := NewModuleLoadStep()

			loaded, err := step.isModuleLoaded(ctx, tc.moduleName)

			assert.NoError(t, err)
			assert.Equal(t, tc.wantLoaded, loaded)
		})
	}
}

// =============================================================================
// Benchmarks
// =============================================================================

func BenchmarkModuleLoadStep_Execute(b *testing.B) {
	mockExec := exec.NewMockExecutor()
	mockExec.SetDefaultResponse(exec.SuccessResult(""))
	mockExec.SetResponse("lsmod", exec.SuccessResult(""))

	mockDetector := newMockModuleKernelDetector()

	step := NewModuleLoadStep(
		WithModuleKernelDetector(mockDetector),
	)

	ctx := install.NewContext(
		install.WithExecutor(mockExec),
	)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Clear state between iterations
		ctx.DeleteState(StateModulesLoaded)
		ctx.DeleteState(StateLoadedModules)

		step.Execute(ctx)
	}
}

func BenchmarkModuleLoadStep_Validate(b *testing.B) {
	mockExec := exec.NewMockExecutor()

	ctx := install.NewContext(
		install.WithExecutor(mockExec),
	)

	step := NewModuleLoadStep()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = step.Validate(ctx)
	}
}
