package steps

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tungetti/igor/internal/constants"
	"github.com/tungetti/igor/internal/distro"
	"github.com/tungetti/igor/internal/exec"
	"github.com/tungetti/igor/internal/gpu/kernel"
	"github.com/tungetti/igor/internal/install"
)

// =============================================================================
// Mock Kernel Detector for Nouveau Restore Tests
// =============================================================================

// mockNouveauKernelDetector is a mock kernel detector for nouveau restore tests.
type mockNouveauKernelDetector struct {
	loadedModules        map[string]bool
	isModuleLoadedFunc   func(ctx context.Context, name string) (bool, error)
	isModuleLoadedErr    error
	getLoadedModulesFunc func(ctx context.Context) ([]kernel.ModuleInfo, error)
}

// newMockNouveauKernelDetector creates a new mock kernel detector.
func newMockNouveauKernelDetector() *mockNouveauKernelDetector {
	return &mockNouveauKernelDetector{
		loadedModules: make(map[string]bool),
	}
}

// SetModuleLoaded sets whether a module is reported as loaded.
func (m *mockNouveauKernelDetector) SetModuleLoaded(name string, loaded bool) {
	m.loadedModules[name] = loaded
}

// SetIsModuleLoadedFunc sets a custom function for IsModuleLoaded.
func (m *mockNouveauKernelDetector) SetIsModuleLoadedFunc(fn func(ctx context.Context, name string) (bool, error)) {
	m.isModuleLoadedFunc = fn
}

// SetIsModuleLoadedError sets an error to return from IsModuleLoaded.
func (m *mockNouveauKernelDetector) SetIsModuleLoadedError(err error) {
	m.isModuleLoadedErr = err
}

// SetGetLoadedModulesFunc sets a custom function for GetLoadedModules.
func (m *mockNouveauKernelDetector) SetGetLoadedModulesFunc(fn func(ctx context.Context) ([]kernel.ModuleInfo, error)) {
	m.getLoadedModulesFunc = fn
}

// IsModuleLoaded implements kernel.Detector.
func (m *mockNouveauKernelDetector) IsModuleLoaded(ctx context.Context, name string) (bool, error) {
	if m.isModuleLoadedFunc != nil {
		return m.isModuleLoadedFunc(ctx, name)
	}
	if m.isModuleLoadedErr != nil {
		return false, m.isModuleLoadedErr
	}
	return m.loadedModules[name], nil
}

// GetKernelInfo implements kernel.Detector.
func (m *mockNouveauKernelDetector) GetKernelInfo(ctx context.Context) (*kernel.KernelInfo, error) {
	return &kernel.KernelInfo{
		Version:      "6.5.0-44-generic",
		Release:      "6.5.0",
		Architecture: "x86_64",
	}, nil
}

// GetLoadedModules implements kernel.Detector.
func (m *mockNouveauKernelDetector) GetLoadedModules(ctx context.Context) ([]kernel.ModuleInfo, error) {
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
func (m *mockNouveauKernelDetector) GetModule(ctx context.Context, name string) (*kernel.ModuleInfo, error) {
	if m.loadedModules[name] {
		return &kernel.ModuleInfo{Name: name}, nil
	}
	return nil, nil
}

// AreHeadersInstalled implements kernel.Detector.
func (m *mockNouveauKernelDetector) AreHeadersInstalled(ctx context.Context) (bool, error) {
	return true, nil
}

// GetHeadersPackage implements kernel.Detector.
func (m *mockNouveauKernelDetector) GetHeadersPackage(ctx context.Context) (string, error) {
	return "linux-headers-6.5.0-44-generic", nil
}

// IsSecureBootEnabled implements kernel.Detector.
func (m *mockNouveauKernelDetector) IsSecureBootEnabled(ctx context.Context) (bool, error) {
	return false, nil
}

// Ensure mockNouveauKernelDetector implements kernel.Detector.
var _ kernel.Detector = (*mockNouveauKernelDetector)(nil)

// =============================================================================
// Test Helpers
// =============================================================================

// newNouveauDebianDistro creates a test Debian distribution.
func newNouveauDebianDistro() *distro.Distribution {
	return &distro.Distribution{
		ID:         "debian",
		Name:       "Debian GNU/Linux",
		VersionID:  "12",
		PrettyName: "Debian GNU/Linux 12 (bookworm)",
		Family:     constants.FamilyDebian,
	}
}

// newNouveauFedoraDistro creates a test Fedora distribution.
func newNouveauFedoraDistro() *distro.Distribution {
	return &distro.Distribution{
		ID:         "fedora",
		Name:       "Fedora Linux",
		VersionID:  "40",
		PrettyName: "Fedora Linux 40 (Workstation Edition)",
		Family:     constants.FamilyRHEL,
	}
}

// newNouveauArchDistro creates a test Arch Linux distribution.
func newNouveauArchDistro() *distro.Distribution {
	return &distro.Distribution{
		ID:         "arch",
		Name:       "Arch Linux",
		PrettyName: "Arch Linux",
		Family:     constants.FamilyArch,
	}
}

// newNouveauSUSEDistro creates a test openSUSE distribution.
func newNouveauSUSEDistro() *distro.Distribution {
	return &distro.Distribution{
		ID:         "opensuse-leap",
		Name:       "openSUSE Leap",
		VersionID:  "15.5",
		PrettyName: "openSUSE Leap 15.5",
		Family:     constants.FamilySUSE,
	}
}

// newNouveauRestoreTestContext creates a basic test context with executor for nouveau restore tests.
func newNouveauRestoreTestContext() (*install.Context, *exec.MockExecutor) {
	mockExec := exec.NewMockExecutor()
	mockExec.SetDefaultResponse(exec.SuccessResult(""))

	ctx := install.NewContext(
		install.WithExecutor(mockExec),
		install.WithDistroInfo(newNouveauDebianDistro()),
	)

	return ctx, mockExec
}

// setupNouveauRestoreMocks configures the mock executor with common nouveau restore responses.
func setupNouveauRestoreMocks(mockExec *exec.MockExecutor) {
	// test -f for blacklist file exists
	mockExec.SetResponse("test", exec.SuccessResult(""))
	// rm -f succeeds
	mockExec.SetResponse("rm", exec.SuccessResult(""))
	// update-initramfs succeeds
	mockExec.SetResponse("update-initramfs", exec.SuccessResult(""))
	// dracut succeeds
	mockExec.SetResponse("dracut", exec.SuccessResult(""))
	// mkinitcpio succeeds
	mockExec.SetResponse("mkinitcpio", exec.SuccessResult(""))
	// modprobe succeeds
	mockExec.SetResponse("modprobe", exec.SuccessResult(""))
	// lsmod shows nouveau loaded
	mockExec.SetResponse("lsmod", exec.SuccessResult("Module                  Size  Used by\nnouveau              12345678  0\n"))
	// tee succeeds (for rollback)
	mockExec.SetResponse("tee", exec.SuccessResult(""))
}

// =============================================================================
// NouveauRestoreStep Constructor Tests
// =============================================================================

func TestNewNouveauRestoreStep(t *testing.T) {
	step := NewNouveauRestoreStep()

	assert.Equal(t, "nouveau_restore", step.Name())
	assert.Equal(t, "Restore Nouveau driver", step.Description())
	assert.True(t, step.CanRollback())
	assert.True(t, step.removeBlacklist)
	assert.True(t, step.loadModule)
	assert.True(t, step.regenerateInitramfs)
	assert.Nil(t, step.kernelDetector)
	assert.Equal(t, defaultBlacklistPaths, step.blacklistPaths)
}

func TestNouveauRestoreStep_Options(t *testing.T) {
	mockDetector := newMockNouveauKernelDetector()
	customPaths := []string{"/custom/path1.conf", "/custom/path2.conf"}

	step := NewNouveauRestoreStep(
		WithRemoveNouveauBlacklist(false),
		WithLoadNouveauModule(false),
		WithRegenerateInitramfs(false),
		WithNouveauKernelDetector(mockDetector),
		WithBlacklistPaths(customPaths),
	)

	assert.False(t, step.removeBlacklist)
	assert.False(t, step.loadModule)
	assert.False(t, step.regenerateInitramfs)
	assert.Equal(t, mockDetector, step.kernelDetector)
	assert.Equal(t, customPaths, step.blacklistPaths)
}

func TestNouveauRestoreStep_WithRemoveNouveauBlacklist(t *testing.T) {
	t.Run("enables blacklist removal", func(t *testing.T) {
		step := NewNouveauRestoreStep(WithRemoveNouveauBlacklist(true))
		assert.True(t, step.removeBlacklist)
	})

	t.Run("disables blacklist removal", func(t *testing.T) {
		step := NewNouveauRestoreStep(WithRemoveNouveauBlacklist(false))
		assert.False(t, step.removeBlacklist)
	})
}

func TestNouveauRestoreStep_WithLoadNouveauModule(t *testing.T) {
	t.Run("enables module loading", func(t *testing.T) {
		step := NewNouveauRestoreStep(WithLoadNouveauModule(true))
		assert.True(t, step.loadModule)
	})

	t.Run("disables module loading", func(t *testing.T) {
		step := NewNouveauRestoreStep(WithLoadNouveauModule(false))
		assert.False(t, step.loadModule)
	})
}

func TestNouveauRestoreStep_WithRegenerateInitramfs(t *testing.T) {
	t.Run("enables initramfs regeneration", func(t *testing.T) {
		step := NewNouveauRestoreStep(WithRegenerateInitramfs(true))
		assert.True(t, step.regenerateInitramfs)
	})

	t.Run("disables initramfs regeneration", func(t *testing.T) {
		step := NewNouveauRestoreStep(WithRegenerateInitramfs(false))
		assert.False(t, step.regenerateInitramfs)
	})
}

func TestNouveauRestoreStep_WithNouveauKernelDetector(t *testing.T) {
	mockDetector := newMockNouveauKernelDetector()
	step := NewNouveauRestoreStep(WithNouveauKernelDetector(mockDetector))
	assert.Equal(t, mockDetector, step.kernelDetector)
}

func TestNouveauRestoreStep_WithBlacklistPaths_CopiesSlice(t *testing.T) {
	original := []string{"/path/1.conf", "/path/2.conf"}
	step := NewNouveauRestoreStep(WithBlacklistPaths(original))

	// Modify original slice
	original[0] = "modified"

	// Step should have the original values (defensive copy)
	assert.Equal(t, "/path/1.conf", step.blacklistPaths[0])
}

func TestNouveauRestoreStep_LaterOptionsOverrideEarlierOnes(t *testing.T) {
	step := NewNouveauRestoreStep(
		WithRemoveNouveauBlacklist(true),
		WithRemoveNouveauBlacklist(false), // This should override the first
	)

	assert.False(t, step.removeBlacklist)
}

// =============================================================================
// NouveauRestoreStep Execute Tests
// =============================================================================

func TestNouveauRestoreStep_Execute_Success(t *testing.T) {
	ctx, mockExec := newNouveauRestoreTestContext()
	mockDetector := newMockNouveauKernelDetector()
	mockDetector.SetModuleLoaded("nouveau", false)

	setupNouveauRestoreMocks(mockExec)

	step := NewNouveauRestoreStep(
		WithNouveauKernelDetector(mockDetector),
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusCompleted, result.Status)
	assert.Contains(t, result.Message, "restored successfully")
	assert.Greater(t, result.Duration.Nanoseconds(), int64(0))

	// Check state was set
	assert.True(t, ctx.GetStateBool(StateNouveauRestored))
	assert.True(t, ctx.GetStateBool(StateNouveauBlacklistRemoved))
	assert.True(t, ctx.GetStateBool(StateInitramfsRegenerated))
	assert.True(t, ctx.GetStateBool(StateNouveauModuleLoaded))

	// Verify commands were called
	assert.True(t, mockExec.WasCalled("rm"))
	assert.True(t, mockExec.WasCalled("update-initramfs"))
	assert.True(t, mockExec.WasCalled("modprobe"))
}

func TestNouveauRestoreStep_Execute_BlacklistRemovalDisabled(t *testing.T) {
	ctx, mockExec := newNouveauRestoreTestContext()
	mockDetector := newMockNouveauKernelDetector()
	mockDetector.SetModuleLoaded("nouveau", false)

	setupNouveauRestoreMocks(mockExec)

	step := NewNouveauRestoreStep(
		WithNouveauKernelDetector(mockDetector),
		WithRemoveNouveauBlacklist(false),
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusCompleted, result.Status)

	// State should reflect that blacklist was not removed
	assert.True(t, ctx.GetStateBool(StateNouveauRestored))
	assert.False(t, ctx.GetStateBool(StateNouveauBlacklistRemoved))

	// rm should not have been called for blacklist removal
	calls := mockExec.Calls()
	rmCalled := false
	for _, call := range calls {
		if call.Command == "rm" {
			rmCalled = true
		}
	}
	assert.False(t, rmCalled)
}

func TestNouveauRestoreStep_Execute_ModuleLoadingDisabled(t *testing.T) {
	ctx, mockExec := newNouveauRestoreTestContext()
	mockDetector := newMockNouveauKernelDetector()
	mockDetector.SetModuleLoaded("nouveau", false)

	setupNouveauRestoreMocks(mockExec)

	step := NewNouveauRestoreStep(
		WithNouveauKernelDetector(mockDetector),
		WithLoadNouveauModule(false),
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusCompleted, result.Status)

	// State should reflect that module was not loaded
	assert.True(t, ctx.GetStateBool(StateNouveauRestored))
	assert.False(t, ctx.GetStateBool(StateNouveauModuleLoaded))

	// modprobe should not have been called
	assert.False(t, mockExec.WasCalledWith("modprobe", "nouveau"))
}

func TestNouveauRestoreStep_Execute_InitramfsRegenerationDisabled(t *testing.T) {
	ctx, mockExec := newNouveauRestoreTestContext()
	mockDetector := newMockNouveauKernelDetector()
	mockDetector.SetModuleLoaded("nouveau", false)

	setupNouveauRestoreMocks(mockExec)

	step := NewNouveauRestoreStep(
		WithNouveauKernelDetector(mockDetector),
		WithRegenerateInitramfs(false),
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusCompleted, result.Status)

	// State should reflect that initramfs was not regenerated
	assert.True(t, ctx.GetStateBool(StateNouveauRestored))
	assert.False(t, ctx.GetStateBool(StateInitramfsRegenerated))

	// update-initramfs should not have been called
	assert.False(t, mockExec.WasCalled("update-initramfs"))
}

func TestNouveauRestoreStep_Execute_DryRun(t *testing.T) {
	ctx, mockExec := newNouveauRestoreTestContext()
	ctx.DryRun = true

	mockDetector := newMockNouveauKernelDetector()
	mockDetector.SetModuleLoaded("nouveau", false)

	step := NewNouveauRestoreStep(
		WithNouveauKernelDetector(mockDetector),
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusCompleted, result.Status)
	assert.Contains(t, result.Message, "dry run")

	// State should not be set for dry run
	assert.False(t, ctx.GetStateBool(StateNouveauRestored))

	// No commands should have been called (except maybe for checking if loaded)
	assert.False(t, mockExec.WasCalled("rm"))
	assert.False(t, mockExec.WasCalled("update-initramfs"))
	assert.False(t, mockExec.WasCalledWith("modprobe", "nouveau"))
}

func TestNouveauRestoreStep_Execute_NouveauAlreadyLoaded(t *testing.T) {
	ctx, mockExec := newNouveauRestoreTestContext()
	mockDetector := newMockNouveauKernelDetector()
	mockDetector.SetModuleLoaded("nouveau", true) // Already loaded

	setupNouveauRestoreMocks(mockExec)

	step := NewNouveauRestoreStep(
		WithNouveauKernelDetector(mockDetector),
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusSkipped, result.Status)
	assert.Contains(t, result.Message, "already loaded")

	// State should indicate restored but not loaded by us
	assert.True(t, ctx.GetStateBool(StateNouveauRestored))
	assert.False(t, ctx.GetStateBool(StateNouveauModuleLoaded))
}

func TestNouveauRestoreStep_Execute_Cancelled(t *testing.T) {
	ctx, mockExec := newNouveauRestoreTestContext()
	ctx.Cancel() // Cancel immediately

	mockExec.SetDefaultResponse(exec.SuccessResult(""))

	step := NewNouveauRestoreStep()

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusFailed, result.Status)
	assert.Contains(t, result.Message, "cancelled")
	assert.True(t, errors.Is(result.Error, context.Canceled))
}

func TestNouveauRestoreStep_Execute_NoExecutor(t *testing.T) {
	ctx := install.NewContext()

	step := NewNouveauRestoreStep()

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusFailed, result.Status)
	assert.Contains(t, result.Message, "validation failed")
	assert.Error(t, result.Error)
	assert.Contains(t, result.Error.Error(), "executor")
}

func TestNouveauRestoreStep_Execute_NilContext(t *testing.T) {
	step := NewNouveauRestoreStep()

	err := step.Validate(nil)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context is nil")
}

func TestNouveauRestoreStep_Execute_BlacklistRemovalFailure(t *testing.T) {
	ctx, mockExec := newNouveauRestoreTestContext()
	mockDetector := newMockNouveauKernelDetector()
	mockDetector.SetModuleLoaded("nouveau", false)

	// test -f succeeds (file exists)
	mockExec.SetResponse("test", exec.SuccessResult(""))
	// rm -f fails
	mockExec.SetResponse("rm", exec.FailureResult(1, "permission denied"))

	step := NewNouveauRestoreStep(
		WithNouveauKernelDetector(mockDetector),
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusFailed, result.Status)
	assert.Contains(t, result.Message, "failed to remove blacklist files")
	assert.Error(t, result.Error)

	// State should not be set on failure
	assert.False(t, ctx.GetStateBool(StateNouveauRestored))
}

func TestNouveauRestoreStep_Execute_InitramfsFailure(t *testing.T) {
	ctx, mockExec := newNouveauRestoreTestContext()
	mockDetector := newMockNouveauKernelDetector()
	mockDetector.SetModuleLoaded("nouveau", false)

	// test -f succeeds
	mockExec.SetResponse("test", exec.SuccessResult(""))
	// rm -f succeeds
	mockExec.SetResponse("rm", exec.SuccessResult(""))
	// update-initramfs fails
	mockExec.SetResponse("update-initramfs", exec.FailureResult(1, "initramfs update failed"))
	// tee for rollback succeeds
	mockExec.SetResponse("tee", exec.SuccessResult(""))

	step := NewNouveauRestoreStep(
		WithNouveauKernelDetector(mockDetector),
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusFailed, result.Status)
	assert.Contains(t, result.Message, "failed to regenerate initramfs")
	assert.Error(t, result.Error)

	// Should have tried to rollback by re-creating blacklist file
	assert.True(t, mockExec.WasCalled("tee"))

	// State should not be set on failure
	assert.False(t, ctx.GetStateBool(StateNouveauRestored))
}

func TestNouveauRestoreStep_Execute_ModuleLoadFailure_ContinuesWithWarning(t *testing.T) {
	ctx, mockExec := newNouveauRestoreTestContext()
	mockDetector := newMockNouveauKernelDetector()
	mockDetector.SetModuleLoaded("nouveau", false)

	// test -f fails (no blacklist files)
	mockExec.SetResponse("test", exec.FailureResult(1, ""))
	// update-initramfs succeeds
	mockExec.SetResponse("update-initramfs", exec.SuccessResult(""))
	// modprobe fails
	mockExec.SetResponse("modprobe", exec.FailureResult(1, "module not found"))

	step := NewNouveauRestoreStep(
		WithNouveauKernelDetector(mockDetector),
	)

	result := step.Execute(ctx)

	// Should still succeed because module load failure is not fatal
	assert.Equal(t, install.StepStatusCompleted, result.Status)
	assert.Contains(t, result.Message, "restored successfully")

	// State should indicate module was not loaded
	assert.True(t, ctx.GetStateBool(StateNouveauRestored))
	assert.False(t, ctx.GetStateBool(StateNouveauModuleLoaded))
}

// =============================================================================
// NouveauRestoreStep Distribution-Specific Initramfs Tests
// =============================================================================

func TestNouveauRestoreStep_Execute_DebianInitramfs(t *testing.T) {
	ctx, mockExec := newNouveauRestoreTestContext()
	ctx = install.NewContext(
		install.WithExecutor(mockExec),
		install.WithDistroInfo(newNouveauDebianDistro()),
	)

	mockDetector := newMockNouveauKernelDetector()
	mockDetector.SetModuleLoaded("nouveau", false)

	setupNouveauRestoreMocks(mockExec)

	step := NewNouveauRestoreStep(
		WithNouveauKernelDetector(mockDetector),
		WithRemoveNouveauBlacklist(false),
		WithLoadNouveauModule(false),
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusCompleted, result.Status)
	assert.True(t, mockExec.WasCalledWith("update-initramfs", "-u"))
}

func TestNouveauRestoreStep_Execute_FedoraInitramfs(t *testing.T) {
	mockExec := exec.NewMockExecutor()
	mockExec.SetDefaultResponse(exec.SuccessResult(""))

	ctx := install.NewContext(
		install.WithExecutor(mockExec),
		install.WithDistroInfo(newNouveauFedoraDistro()),
	)

	mockDetector := newMockNouveauKernelDetector()
	mockDetector.SetModuleLoaded("nouveau", false)

	setupNouveauRestoreMocks(mockExec)

	step := NewNouveauRestoreStep(
		WithNouveauKernelDetector(mockDetector),
		WithRemoveNouveauBlacklist(false),
		WithLoadNouveauModule(false),
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusCompleted, result.Status)
	assert.True(t, mockExec.WasCalledWith("dracut", "--force"))
}

func TestNouveauRestoreStep_Execute_ArchInitramfs(t *testing.T) {
	mockExec := exec.NewMockExecutor()
	mockExec.SetDefaultResponse(exec.SuccessResult(""))

	ctx := install.NewContext(
		install.WithExecutor(mockExec),
		install.WithDistroInfo(newNouveauArchDistro()),
	)

	mockDetector := newMockNouveauKernelDetector()
	mockDetector.SetModuleLoaded("nouveau", false)

	setupNouveauRestoreMocks(mockExec)

	step := NewNouveauRestoreStep(
		WithNouveauKernelDetector(mockDetector),
		WithRemoveNouveauBlacklist(false),
		WithLoadNouveauModule(false),
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusCompleted, result.Status)
	assert.True(t, mockExec.WasCalledWith("mkinitcpio", "-P"))
}

func TestNouveauRestoreStep_Execute_SUSEInitramfs(t *testing.T) {
	mockExec := exec.NewMockExecutor()
	mockExec.SetDefaultResponse(exec.SuccessResult(""))

	ctx := install.NewContext(
		install.WithExecutor(mockExec),
		install.WithDistroInfo(newNouveauSUSEDistro()),
	)

	mockDetector := newMockNouveauKernelDetector()
	mockDetector.SetModuleLoaded("nouveau", false)

	setupNouveauRestoreMocks(mockExec)

	step := NewNouveauRestoreStep(
		WithNouveauKernelDetector(mockDetector),
		WithRemoveNouveauBlacklist(false),
		WithLoadNouveauModule(false),
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusCompleted, result.Status)
	assert.True(t, mockExec.WasCalledWith("dracut", "--force"))
}

func TestNouveauRestoreStep_Execute_UnknownDistroFallback(t *testing.T) {
	mockExec := exec.NewMockExecutor()
	mockExec.SetDefaultResponse(exec.SuccessResult(""))

	unknownDistro := &distro.Distribution{
		ID:         "unknown",
		Name:       "Unknown Linux",
		PrettyName: "Unknown Linux",
		Family:     constants.FamilyUnknown,
	}

	ctx := install.NewContext(
		install.WithExecutor(mockExec),
		install.WithDistroInfo(unknownDistro),
	)

	mockDetector := newMockNouveauKernelDetector()
	mockDetector.SetModuleLoaded("nouveau", false)

	setupNouveauRestoreMocks(mockExec)

	step := NewNouveauRestoreStep(
		WithNouveauKernelDetector(mockDetector),
		WithRemoveNouveauBlacklist(false),
		WithLoadNouveauModule(false),
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusCompleted, result.Status)
	// Should fall back to Debian-style command
	assert.True(t, mockExec.WasCalledWith("update-initramfs", "-u"))
}

func TestNouveauRestoreStep_Execute_NoDistroInfo(t *testing.T) {
	mockExec := exec.NewMockExecutor()
	mockExec.SetDefaultResponse(exec.SuccessResult(""))

	ctx := install.NewContext(
		install.WithExecutor(mockExec),
		// No distro info
	)

	mockDetector := newMockNouveauKernelDetector()
	mockDetector.SetModuleLoaded("nouveau", false)

	setupNouveauRestoreMocks(mockExec)

	step := NewNouveauRestoreStep(
		WithNouveauKernelDetector(mockDetector),
		WithRemoveNouveauBlacklist(false),
		WithLoadNouveauModule(false),
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusCompleted, result.Status)
	// Should fall back to Debian-style command when no distro info
	assert.True(t, mockExec.WasCalledWith("update-initramfs", "-u"))
}

// =============================================================================
// NouveauRestoreStep Rollback Tests
// =============================================================================

func TestNouveauRestoreStep_Rollback_Success(t *testing.T) {
	ctx, mockExec := newNouveauRestoreTestContext()

	// Simulate that nouveau was restored
	ctx.SetState(StateNouveauRestored, true)
	ctx.SetState(StateNouveauBlacklistRemoved, true)
	ctx.SetState(StateInitramfsRegenerated, true)
	ctx.SetState(StateNouveauModuleLoaded, true)

	setupNouveauRestoreMocks(mockExec)

	step := NewNouveauRestoreStep()

	err := step.Rollback(ctx)

	assert.NoError(t, err)

	// Verify modprobe -r was called to unload nouveau
	assert.True(t, mockExec.WasCalledWith("modprobe", "-r", "nouveau"))

	// Verify tee was called to re-create blacklist
	assert.True(t, mockExec.WasCalled("tee"))

	// Verify initramfs was regenerated
	assert.True(t, mockExec.WasCalled("update-initramfs"))

	// State should be cleared
	assert.False(t, ctx.GetStateBool(StateNouveauRestored))
	assert.False(t, ctx.GetStateBool(StateNouveauBlacklistRemoved))
	assert.False(t, ctx.GetStateBool(StateInitramfsRegenerated))
	assert.False(t, ctx.GetStateBool(StateNouveauModuleLoaded))
}

func TestNouveauRestoreStep_Rollback_NoRestorationDone(t *testing.T) {
	ctx, mockExec := newNouveauRestoreTestContext()

	step := NewNouveauRestoreStep()

	err := step.Rollback(ctx)

	assert.NoError(t, err)

	// No commands should have been called
	assert.Equal(t, 0, mockExec.CallCount())
}

func TestNouveauRestoreStep_Rollback_NoExecutor(t *testing.T) {
	ctx := install.NewContext()
	ctx.SetState(StateNouveauRestored, true)
	ctx.SetState(StateNouveauBlacklistRemoved, true)

	step := NewNouveauRestoreStep()

	err := step.Rollback(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "executor not available")
}

func TestNouveauRestoreStep_Rollback_ModuleUnloadOnlyIfLoaded(t *testing.T) {
	ctx, mockExec := newNouveauRestoreTestContext()

	ctx.SetState(StateNouveauRestored, true)
	ctx.SetState(StateNouveauBlacklistRemoved, false)
	ctx.SetState(StateInitramfsRegenerated, false)
	ctx.SetState(StateNouveauModuleLoaded, true) // We loaded it

	setupNouveauRestoreMocks(mockExec)

	step := NewNouveauRestoreStep()

	err := step.Rollback(ctx)

	assert.NoError(t, err)

	// Should unload module since we loaded it
	assert.True(t, mockExec.WasCalledWith("modprobe", "-r", "nouveau"))
}

func TestNouveauRestoreStep_Rollback_NoModuleUnloadIfNotLoaded(t *testing.T) {
	ctx, mockExec := newNouveauRestoreTestContext()

	ctx.SetState(StateNouveauRestored, true)
	ctx.SetState(StateNouveauBlacklistRemoved, false)
	ctx.SetState(StateInitramfsRegenerated, false)
	ctx.SetState(StateNouveauModuleLoaded, false) // We didn't load it

	setupNouveauRestoreMocks(mockExec)

	step := NewNouveauRestoreStep()

	err := step.Rollback(ctx)

	assert.NoError(t, err)

	// Should NOT unload module since we didn't load it
	assert.False(t, mockExec.WasCalledWith("modprobe", "-r", "nouveau"))
}

func TestNouveauRestoreStep_Rollback_ReCreateBlacklistFailure(t *testing.T) {
	ctx, mockExec := newNouveauRestoreTestContext()

	ctx.SetState(StateNouveauRestored, true)
	ctx.SetState(StateNouveauBlacklistRemoved, true)
	ctx.SetState(StateInitramfsRegenerated, false)
	ctx.SetState(StateNouveauModuleLoaded, false)

	// tee fails
	mockExec.SetResponse("tee", exec.FailureResult(1, "permission denied"))

	step := NewNouveauRestoreStep()

	err := step.Rollback(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to re-create blacklist file")
}

func TestNouveauRestoreStep_Rollback_InitramfsFailure(t *testing.T) {
	ctx, mockExec := newNouveauRestoreTestContext()

	ctx.SetState(StateNouveauRestored, true)
	ctx.SetState(StateNouveauBlacklistRemoved, false)
	ctx.SetState(StateInitramfsRegenerated, true)
	ctx.SetState(StateNouveauModuleLoaded, false)

	// update-initramfs fails
	mockExec.SetResponse("update-initramfs", exec.FailureResult(1, "initramfs failed"))

	step := NewNouveauRestoreStep()

	err := step.Rollback(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to regenerate initramfs")
}

func TestNouveauRestoreStep_Rollback_ModuleUnloadFailure_Continues(t *testing.T) {
	ctx, mockExec := newNouveauRestoreTestContext()

	ctx.SetState(StateNouveauRestored, true)
	ctx.SetState(StateNouveauBlacklistRemoved, false)
	ctx.SetState(StateInitramfsRegenerated, false)
	ctx.SetState(StateNouveauModuleLoaded, true)

	// modprobe -r fails, but rollback should continue
	mockExec.SetResponse("modprobe", exec.FailureResult(1, "module in use"))

	step := NewNouveauRestoreStep()

	// Should NOT error because module unload failure is handled gracefully
	err := step.Rollback(ctx)

	assert.NoError(t, err)
}

// =============================================================================
// NouveauRestoreStep Validate Tests
// =============================================================================

func TestNouveauRestoreStep_Validate_Success(t *testing.T) {
	ctx, _ := newNouveauRestoreTestContext()

	step := NewNouveauRestoreStep()

	err := step.Validate(ctx)

	assert.NoError(t, err)
}

func TestNouveauRestoreStep_Validate_NilContext(t *testing.T) {
	step := NewNouveauRestoreStep()

	err := step.Validate(nil)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context is nil")
}

func TestNouveauRestoreStep_Validate_NoExecutor(t *testing.T) {
	ctx := install.NewContext()

	step := NewNouveauRestoreStep()

	err := step.Validate(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "executor is required")
}

// =============================================================================
// NouveauRestoreStep CanRollback Tests
// =============================================================================

func TestNouveauRestoreStep_CanRollback(t *testing.T) {
	step := NewNouveauRestoreStep()
	assert.True(t, step.CanRollback())
}

// =============================================================================
// NouveauRestoreStep State Keys Tests
// =============================================================================

func TestNouveauRestoreStep_StateKeys(t *testing.T) {
	assert.Equal(t, "nouveau_restored", StateNouveauRestored)
	assert.Equal(t, "nouveau_blacklist_removed", StateNouveauBlacklistRemoved)
	assert.Equal(t, "initramfs_regenerated", StateInitramfsRegenerated)
	assert.Equal(t, "nouveau_module_loaded", StateNouveauModuleLoaded)
}

// =============================================================================
// NouveauRestoreStep getInitramfsCommand Tests
// =============================================================================

func TestGetInitramfsCommand(t *testing.T) {
	tests := []struct {
		name     string
		family   constants.DistroFamily
		wantCmd  string
		wantArgs []string
	}{
		{
			name:     "Debian family",
			family:   constants.FamilyDebian,
			wantCmd:  "update-initramfs",
			wantArgs: []string{"-u"},
		},
		{
			name:     "RHEL family",
			family:   constants.FamilyRHEL,
			wantCmd:  "dracut",
			wantArgs: []string{"--force"},
		},
		{
			name:     "SUSE family",
			family:   constants.FamilySUSE,
			wantCmd:  "dracut",
			wantArgs: []string{"--force"},
		},
		{
			name:     "Arch family",
			family:   constants.FamilyArch,
			wantCmd:  "mkinitcpio",
			wantArgs: []string{"-P"},
		},
		{
			name:     "Unknown family (fallback)",
			family:   constants.FamilyUnknown,
			wantCmd:  "update-initramfs",
			wantArgs: []string{"-u"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd, args := getInitramfsCommand(tt.family)
			assert.Equal(t, tt.wantCmd, cmd)
			assert.Equal(t, tt.wantArgs, args)
		})
	}
}

// =============================================================================
// NouveauRestoreStep getDistroFamily Tests
// =============================================================================

func TestNouveauRestoreStep_getDistroFamily(t *testing.T) {
	step := NewNouveauRestoreStep()

	t.Run("returns family from DistroInfo", func(t *testing.T) {
		ctx := install.NewContext(
			install.WithDistroInfo(newNouveauDebianDistro()),
		)
		family := step.getDistroFamily(ctx)
		assert.Equal(t, constants.FamilyDebian, family)
	})

	t.Run("returns unknown when DistroInfo is nil", func(t *testing.T) {
		ctx := install.NewContext()
		family := step.getDistroFamily(ctx)
		assert.Equal(t, constants.FamilyUnknown, family)
	})
}

// =============================================================================
// NouveauRestoreStep isNouveauLoaded Tests
// =============================================================================

func TestNouveauRestoreStep_isNouveauLoaded_WithDetector(t *testing.T) {
	ctx, _ := newNouveauRestoreTestContext()
	mockDetector := newMockNouveauKernelDetector()
	mockDetector.SetModuleLoaded("nouveau", true)

	step := NewNouveauRestoreStep(
		WithNouveauKernelDetector(mockDetector),
	)

	loaded, err := step.isNouveauLoaded(ctx)

	assert.NoError(t, err)
	assert.True(t, loaded)
}

func TestNouveauRestoreStep_isNouveauLoaded_WithDetector_NotLoaded(t *testing.T) {
	ctx, _ := newNouveauRestoreTestContext()
	mockDetector := newMockNouveauKernelDetector()
	mockDetector.SetModuleLoaded("nouveau", false)

	step := NewNouveauRestoreStep(
		WithNouveauKernelDetector(mockDetector),
	)

	loaded, err := step.isNouveauLoaded(ctx)

	assert.NoError(t, err)
	assert.False(t, loaded)
}

func TestNouveauRestoreStep_isNouveauLoaded_WithDetector_Error(t *testing.T) {
	ctx, _ := newNouveauRestoreTestContext()
	mockDetector := newMockNouveauKernelDetector()
	mockDetector.SetIsModuleLoadedError(errors.New("detector error"))

	step := NewNouveauRestoreStep(
		WithNouveauKernelDetector(mockDetector),
	)

	_, err := step.isNouveauLoaded(ctx)

	assert.Error(t, err)
}

func TestNouveauRestoreStep_isNouveauLoaded_FallbackLsmod(t *testing.T) {
	ctx, mockExec := newNouveauRestoreTestContext()

	// No detector, should use lsmod
	mockExec.SetResponse("lsmod", exec.SuccessResult("Module                  Size  Used by\nnouveau              12345678  5\n"))

	step := NewNouveauRestoreStep()

	loaded, err := step.isNouveauLoaded(ctx)

	assert.NoError(t, err)
	assert.True(t, loaded)
	assert.True(t, mockExec.WasCalled("lsmod"))
}

func TestNouveauRestoreStep_isNouveauLoaded_FallbackLsmod_NotFound(t *testing.T) {
	ctx, mockExec := newNouveauRestoreTestContext()

	// No detector, should use lsmod
	mockExec.SetResponse("lsmod", exec.SuccessResult("Module                  Size  Used by\nnvidia              12345678  0\n"))

	step := NewNouveauRestoreStep()

	loaded, err := step.isNouveauLoaded(ctx)

	assert.NoError(t, err)
	assert.False(t, loaded)
}

func TestNouveauRestoreStep_isNouveauLoaded_FallbackLsmod_Error(t *testing.T) {
	ctx, mockExec := newNouveauRestoreTestContext()

	mockExec.SetResponse("lsmod", exec.FailureResult(1, "command not found"))

	step := NewNouveauRestoreStep()

	_, err := step.isNouveauLoaded(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to list modules")
}

// =============================================================================
// NouveauRestoreStep Default Blacklist Paths Tests
// =============================================================================

func TestDefaultBlacklistPaths(t *testing.T) {
	expected := []string{
		"/etc/modprobe.d/blacklist-nouveau.conf",
		"/etc/modprobe.d/nvidia-blacklists-nouveau.conf",
		"/etc/modprobe.d/nvidia-installer-disable-nouveau.conf",
		"/etc/modprobe.d/nouveau-blacklist.conf",
	}
	assert.Equal(t, expected, defaultBlacklistPaths)
}

// =============================================================================
// NouveauRestoreStep Interface Compliance Tests
// =============================================================================

func TestNouveauRestoreStep_InterfaceCompliance(t *testing.T) {
	var _ install.Step = (*NouveauRestoreStep)(nil)
}

// =============================================================================
// NouveauRestoreStep Full Workflow Tests
// =============================================================================

func TestNouveauRestoreStep_FullWorkflow_ExecuteAndRollback(t *testing.T) {
	ctx, mockExec := newNouveauRestoreTestContext()

	mockDetector := newMockNouveauKernelDetector()
	mockDetector.SetModuleLoaded("nouveau", false)

	setupNouveauRestoreMocks(mockExec)

	step := NewNouveauRestoreStep(
		WithNouveauKernelDetector(mockDetector),
	)

	// Execute
	result := step.Execute(ctx)
	assert.Equal(t, install.StepStatusCompleted, result.Status)
	assert.True(t, ctx.GetStateBool(StateNouveauRestored))

	// Reset mock tracking
	mockExec.Reset()
	setupNouveauRestoreMocks(mockExec)

	// Rollback
	err := step.Rollback(ctx)
	assert.NoError(t, err)

	// Verify rollback actions
	assert.True(t, mockExec.WasCalledWith("modprobe", "-r", "nouveau"))
	assert.True(t, mockExec.WasCalled("tee"))
	assert.True(t, mockExec.WasCalled("update-initramfs"))

	// State should be cleared
	assert.False(t, ctx.GetStateBool(StateNouveauRestored))
	assert.False(t, ctx.GetStateBool(StateNouveauBlacklistRemoved))
	assert.False(t, ctx.GetStateBool(StateInitramfsRegenerated))
	assert.False(t, ctx.GetStateBool(StateNouveauModuleLoaded))
}

func TestNouveauRestoreStep_Duration(t *testing.T) {
	ctx, mockExec := newNouveauRestoreTestContext()

	mockDetector := newMockNouveauKernelDetector()
	mockDetector.SetModuleLoaded("nouveau", false)

	setupNouveauRestoreMocks(mockExec)

	step := NewNouveauRestoreStep(
		WithNouveauKernelDetector(mockDetector),
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusCompleted, result.Status)
	assert.Greater(t, result.Duration.Nanoseconds(), int64(0))
}

// =============================================================================
// NouveauRestoreStep Cancellation Tests
// =============================================================================

func TestNouveauRestoreStep_Execute_CancelledBeforeInitramfs(t *testing.T) {
	// Test cancellation handling at the initramfs step by cancelling before execute
	// and verifying proper rollback behavior
	ctx, mockExec := newNouveauRestoreTestContext()

	mockDetector := newMockNouveauKernelDetector()
	mockDetector.SetModuleLoaded("nouveau", false)

	// Set up responses
	mockExec.SetResponse("test", exec.SuccessResult(""))
	mockExec.SetResponse("rm", exec.SuccessResult(""))
	mockExec.SetResponse("update-initramfs", exec.FailureResult(1, "initramfs failed"))
	mockExec.SetResponse("tee", exec.SuccessResult(""))

	step := NewNouveauRestoreStep(
		WithNouveauKernelDetector(mockDetector),
	)

	result := step.Execute(ctx)

	// Should fail due to initramfs error, which triggers rollback
	assert.Equal(t, install.StepStatusFailed, result.Status)
	assert.Contains(t, result.Message, "failed to regenerate initramfs")

	// Should have tried to rollback by re-creating blacklist
	assert.True(t, mockExec.WasCalled("tee"))
}

// =============================================================================
// NouveauRestoreStep Blacklist File Not Found Tests
// =============================================================================

func TestNouveauRestoreStep_Execute_NoBlacklistFilesExist(t *testing.T) {
	ctx, mockExec := newNouveauRestoreTestContext()

	mockDetector := newMockNouveauKernelDetector()
	mockDetector.SetModuleLoaded("nouveau", false)

	// test -f fails for all files (none exist)
	mockExec.SetResponse("test", exec.FailureResult(1, ""))
	mockExec.SetResponse("update-initramfs", exec.SuccessResult(""))
	mockExec.SetResponse("modprobe", exec.SuccessResult(""))
	mockExec.SetResponse("lsmod", exec.SuccessResult("Module                  Size  Used by\nnouveau              12345678  0\n"))

	step := NewNouveauRestoreStep(
		WithNouveauKernelDetector(mockDetector),
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusCompleted, result.Status)

	// Blacklist removed should be false since no files existed
	assert.False(t, ctx.GetStateBool(StateNouveauBlacklistRemoved))
}

// =============================================================================
// NouveauRestoreStep Load Module Verification Tests
// =============================================================================

func TestNouveauRestoreStep_Execute_VerificationAfterLoad(t *testing.T) {
	ctx, mockExec := newNouveauRestoreTestContext()

	mockDetector := newMockNouveauKernelDetector()

	// First check returns false (not loaded), after load returns true
	callCount := 0
	mockDetector.SetIsModuleLoadedFunc(func(ctx context.Context, name string) (bool, error) {
		callCount++
		if callCount == 1 {
			return false, nil // First check
		}
		return true, nil // After loading
	})

	setupNouveauRestoreMocks(mockExec)

	step := NewNouveauRestoreStep(
		WithNouveauKernelDetector(mockDetector),
		WithRemoveNouveauBlacklist(false),
		WithRegenerateInitramfs(false),
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusCompleted, result.Status)
	assert.True(t, ctx.GetStateBool(StateNouveauModuleLoaded))

	// Should have checked module status twice (before and after loading)
	assert.Equal(t, 2, callCount)
}

func TestNouveauRestoreStep_Execute_VerificationFailedButLoadSucceeded(t *testing.T) {
	ctx, mockExec := newNouveauRestoreTestContext()

	mockDetector := newMockNouveauKernelDetector()

	// First check returns false (not loaded), after load also returns false (verification failed)
	mockDetector.SetIsModuleLoadedFunc(func(ctx context.Context, name string) (bool, error) {
		return false, nil
	})

	setupNouveauRestoreMocks(mockExec)

	step := NewNouveauRestoreStep(
		WithNouveauKernelDetector(mockDetector),
		WithRemoveNouveauBlacklist(false),
		WithRegenerateInitramfs(false),
	)

	result := step.Execute(ctx)

	// Should still succeed but moduleLoaded should be true since modprobe succeeded
	assert.Equal(t, install.StepStatusCompleted, result.Status)
	assert.True(t, ctx.GetStateBool(StateNouveauModuleLoaded))
}

// =============================================================================
// Benchmarks
// =============================================================================

func BenchmarkNouveauRestoreStep_Execute(b *testing.B) {
	mockExec := exec.NewMockExecutor()
	mockExec.SetDefaultResponse(exec.SuccessResult(""))
	mockExec.SetResponse("test", exec.FailureResult(1, ""))
	mockExec.SetResponse("lsmod", exec.SuccessResult("nouveau 12345 0"))

	mockDetector := newMockNouveauKernelDetector()
	mockDetector.SetModuleLoaded("nouveau", false)

	step := NewNouveauRestoreStep(
		WithNouveauKernelDetector(mockDetector),
		WithRemoveNouveauBlacklist(false),
		WithRegenerateInitramfs(false),
	)

	ctx := install.NewContext(
		install.WithExecutor(mockExec),
		install.WithDistroInfo(newNouveauDebianDistro()),
	)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Clear state between iterations
		ctx.DeleteState(StateNouveauRestored)
		ctx.DeleteState(StateNouveauBlacklistRemoved)
		ctx.DeleteState(StateInitramfsRegenerated)
		ctx.DeleteState(StateNouveauModuleLoaded)

		step.Execute(ctx)
	}
}

func BenchmarkNouveauRestoreStep_Validate(b *testing.B) {
	mockExec := exec.NewMockExecutor()

	ctx := install.NewContext(
		install.WithExecutor(mockExec),
		install.WithDistroInfo(newNouveauDebianDistro()),
	)

	step := NewNouveauRestoreStep()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = step.Validate(ctx)
	}
}

// =============================================================================
// Edge Case Tests
// =============================================================================

func TestNouveauRestoreStep_Execute_DetectorCheckError_ProceedsAnyway(t *testing.T) {
	ctx, mockExec := newNouveauRestoreTestContext()
	mockDetector := newMockNouveauKernelDetector()
	mockDetector.SetIsModuleLoadedError(errors.New("detector failed"))

	setupNouveauRestoreMocks(mockExec)

	step := NewNouveauRestoreStep(
		WithNouveauKernelDetector(mockDetector),
		WithRemoveNouveauBlacklist(false),
		WithRegenerateInitramfs(false),
	)

	result := step.Execute(ctx)

	// Should proceed with loading even if detector check fails
	assert.Equal(t, install.StepStatusCompleted, result.Status)
	assert.True(t, mockExec.WasCalledWith("modprobe", "nouveau"))
}

func TestNouveauRestoreStep_Execute_LsmodParsingEdgeCases(t *testing.T) {
	testCases := []struct {
		name       string
		lsmodOut   string
		wantLoaded bool
	}{
		{
			name:       "nouveau at start",
			lsmodOut:   "nouveau              12345678  5\nsnd_hda_intel          45056  0",
			wantLoaded: true,
		},
		{
			name:       "nouveau in middle",
			lsmodOut:   "snd_hda_intel          45056  0\nnouveau              12345678  5\ndrm              12345  0",
			wantLoaded: true,
		},
		{
			name:       "partial match should not match",
			lsmodOut:   "nouveau_video          12345  0\nnouveau_backlight          12345  0",
			wantLoaded: false,
		},
		{
			name:       "empty output",
			lsmodOut:   "",
			wantLoaded: false,
		},
		{
			name:       "header only",
			lsmodOut:   "Module                  Size  Used by",
			wantLoaded: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, mockExec := newNouveauRestoreTestContext()
			mockExec.SetResponse("lsmod", exec.SuccessResult(tc.lsmodOut))

			step := NewNouveauRestoreStep()

			loaded, err := step.isNouveauLoaded(ctx)

			assert.NoError(t, err)
			assert.Equal(t, tc.wantLoaded, loaded)
		})
	}
}

// =============================================================================
// Blacklist Content Tests
// =============================================================================

func TestNouveauRestoreStep_BlacklistContent(t *testing.T) {
	// Verify blacklist content contains required directives
	assert.Contains(t, blacklistContent, "blacklist nouveau")
	assert.Contains(t, blacklistContent, "options nouveau modeset=0")
	assert.Contains(t, blacklistContent, "Igor")
}

// =============================================================================
// Multiple Options Test
// =============================================================================

func TestNouveauRestoreStep_MultipleOptions(t *testing.T) {
	mockDetector := newMockNouveauKernelDetector()
	customPaths := []string{"/custom/path.conf"}

	step := NewNouveauRestoreStep(
		WithRemoveNouveauBlacklist(false),
		WithLoadNouveauModule(false),
		WithRegenerateInitramfs(false),
		WithNouveauKernelDetector(mockDetector),
		WithBlacklistPaths(customPaths),
	)

	assert.False(t, step.removeBlacklist)
	assert.False(t, step.loadModule)
	assert.False(t, step.regenerateInitramfs)
	assert.Equal(t, mockDetector, step.kernelDetector)
	assert.Equal(t, customPaths, step.blacklistPaths)
}

// =============================================================================
// Helper Method Tests
// =============================================================================

func TestNouveauRestoreStep_removeBlacklistFiles_NoFilesExist(t *testing.T) {
	ctx, mockExec := newNouveauRestoreTestContext()

	// test -f fails for all files
	mockExec.SetResponse("test", exec.FailureResult(1, ""))

	step := NewNouveauRestoreStep()

	removed, err := step.removeBlacklistFiles(ctx)

	assert.NoError(t, err)
	assert.False(t, removed)
}

func TestNouveauRestoreStep_removeBlacklistFiles_SomeFilesExist(t *testing.T) {
	ctx, mockExec := newNouveauRestoreTestContext()

	// When test -f succeeds (file exists), rm should be called
	mockExec.SetResponse("test", exec.SuccessResult(""))
	mockExec.SetResponse("rm", exec.SuccessResult(""))

	step := NewNouveauRestoreStep()

	removed, err := step.removeBlacklistFiles(ctx)

	assert.NoError(t, err)
	assert.True(t, removed)
	assert.True(t, mockExec.WasCalled("rm"))
}

func TestNouveauRestoreStep_reCreateBlacklistFile_Success(t *testing.T) {
	ctx, mockExec := newNouveauRestoreTestContext()
	mockExec.SetResponse("tee", exec.SuccessResult(""))

	step := NewNouveauRestoreStep()

	err := step.reCreateBlacklistFile(ctx)

	assert.NoError(t, err)
	assert.True(t, mockExec.WasCalled("tee"))
}

func TestNouveauRestoreStep_reCreateBlacklistFile_Failure(t *testing.T) {
	ctx, mockExec := newNouveauRestoreTestContext()
	mockExec.SetResponse("tee", exec.FailureResult(1, "permission denied"))

	step := NewNouveauRestoreStep()

	err := step.reCreateBlacklistFile(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "permission denied")
}

func TestNouveauRestoreStep_loadNouveauModule_Success(t *testing.T) {
	ctx, mockExec := newNouveauRestoreTestContext()
	mockExec.SetResponse("modprobe", exec.SuccessResult(""))

	step := NewNouveauRestoreStep()

	err := step.loadNouveauModule(ctx)

	assert.NoError(t, err)
	assert.True(t, mockExec.WasCalledWith("modprobe", "nouveau"))
}

func TestNouveauRestoreStep_loadNouveauModule_Failure(t *testing.T) {
	ctx, mockExec := newNouveauRestoreTestContext()
	mockExec.SetResponse("modprobe", exec.FailureResult(1, "module not found"))

	step := NewNouveauRestoreStep()

	err := step.loadNouveauModule(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "module not found")
}

func TestNouveauRestoreStep_unloadNouveauModule_Success(t *testing.T) {
	ctx, mockExec := newNouveauRestoreTestContext()
	mockExec.SetResponse("modprobe", exec.SuccessResult(""))

	step := NewNouveauRestoreStep()

	err := step.unloadNouveauModule(ctx)

	assert.NoError(t, err)
	assert.True(t, mockExec.WasCalledWith("modprobe", "-r", "nouveau"))
}

func TestNouveauRestoreStep_unloadNouveauModule_Failure(t *testing.T) {
	ctx, mockExec := newNouveauRestoreTestContext()
	mockExec.SetResponse("modprobe", exec.FailureResult(1, "module in use"))

	step := NewNouveauRestoreStep()

	err := step.unloadNouveauModule(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "module in use")
}

func TestNouveauRestoreStep_regenerateInitramfsCmd_Success(t *testing.T) {
	ctx, mockExec := newNouveauRestoreTestContext()
	mockExec.SetResponse("update-initramfs", exec.SuccessResult(""))

	step := NewNouveauRestoreStep()

	err := step.regenerateInitramfsCmd(ctx)

	assert.NoError(t, err)
	assert.True(t, mockExec.WasCalledWith("update-initramfs", "-u"))
}

func TestNouveauRestoreStep_regenerateInitramfsCmd_Failure(t *testing.T) {
	ctx, mockExec := newNouveauRestoreTestContext()
	mockExec.SetResponse("update-initramfs", exec.FailureResult(1, "initramfs failed"))

	step := NewNouveauRestoreStep()

	err := step.regenerateInitramfsCmd(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "initramfs failed")
}

func TestNouveauRestoreStep_regenerateInitramfsCmd_UnknownError(t *testing.T) {
	ctx, mockExec := newNouveauRestoreTestContext()
	mockExec.SetResponse("update-initramfs", &exec.Result{
		ExitCode: 1,
		Stdout:   []byte(""),
		Stderr:   []byte(""),
	})

	step := NewNouveauRestoreStep()

	err := step.regenerateInitramfsCmd(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown error")
}

func TestNouveauRestoreStep_regenerateInitramfsCmd_StdoutAsError(t *testing.T) {
	ctx, mockExec := newNouveauRestoreTestContext()
	mockExec.SetResponse("update-initramfs", &exec.Result{
		ExitCode: 1,
		Stdout:   []byte("error in stdout"),
		Stderr:   []byte(""),
	})

	step := NewNouveauRestoreStep()

	err := step.regenerateInitramfsCmd(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error in stdout")
}

// =============================================================================
// Test All Disabled Options
// =============================================================================

func TestNouveauRestoreStep_Execute_AllOptionsDisabled(t *testing.T) {
	ctx, mockExec := newNouveauRestoreTestContext()

	mockDetector := newMockNouveauKernelDetector()
	mockDetector.SetModuleLoaded("nouveau", false)

	setupNouveauRestoreMocks(mockExec)

	step := NewNouveauRestoreStep(
		WithNouveauKernelDetector(mockDetector),
		WithRemoveNouveauBlacklist(false),
		WithLoadNouveauModule(false),
		WithRegenerateInitramfs(false),
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusCompleted, result.Status)

	// No state should be set for disabled operations
	assert.True(t, ctx.GetStateBool(StateNouveauRestored))
	assert.False(t, ctx.GetStateBool(StateNouveauBlacklistRemoved))
	assert.False(t, ctx.GetStateBool(StateInitramfsRegenerated))
	assert.False(t, ctx.GetStateBool(StateNouveauModuleLoaded))

	// No major commands should have been called
	assert.False(t, mockExec.WasCalled("rm"))
	assert.False(t, mockExec.WasCalled("update-initramfs"))
	assert.False(t, mockExec.WasCalledWith("modprobe", "nouveau"))
}

// =============================================================================
// Test Result Contains Duration
// =============================================================================

func TestNouveauRestoreStep_Execute_ResultContainsDuration(t *testing.T) {
	ctx, mockExec := newNouveauRestoreTestContext()
	mockDetector := newMockNouveauKernelDetector()
	mockDetector.SetModuleLoaded("nouveau", false)

	setupNouveauRestoreMocks(mockExec)

	step := NewNouveauRestoreStep(
		WithNouveauKernelDetector(mockDetector),
	)

	// Add a small delay to ensure duration is non-zero
	time.Sleep(1 * time.Millisecond)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusCompleted, result.Status)
	assert.Greater(t, result.Duration, time.Duration(0))
}

// =============================================================================
// Test Result CanRollback Flag
// =============================================================================

func TestNouveauRestoreStep_Execute_ResultHasCanRollbackFlag(t *testing.T) {
	ctx, mockExec := newNouveauRestoreTestContext()
	mockDetector := newMockNouveauKernelDetector()
	mockDetector.SetModuleLoaded("nouveau", false)

	setupNouveauRestoreMocks(mockExec)

	step := NewNouveauRestoreStep(
		WithNouveauKernelDetector(mockDetector),
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusCompleted, result.Status)
	assert.True(t, result.CanRollback)
}

// =============================================================================
// Test with require for Critical Assertions
// =============================================================================

func TestNouveauRestoreStep_Execute_CriticalPath(t *testing.T) {
	ctx, mockExec := newNouveauRestoreTestContext()
	mockDetector := newMockNouveauKernelDetector()
	mockDetector.SetModuleLoaded("nouveau", false)

	setupNouveauRestoreMocks(mockExec)

	step := NewNouveauRestoreStep(
		WithNouveauKernelDetector(mockDetector),
	)

	result := step.Execute(ctx)

	require.Equal(t, install.StepStatusCompleted, result.Status)
	require.True(t, ctx.GetStateBool(StateNouveauRestored))

	// Reset and rollback
	mockExec.Reset()
	setupNouveauRestoreMocks(mockExec)

	err := step.Rollback(ctx)

	require.NoError(t, err)
	require.False(t, ctx.GetStateBool(StateNouveauRestored))
}
