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
	"github.com/tungetti/igor/internal/install"
	"github.com/tungetti/igor/internal/pkg"
	"github.com/tungetti/igor/internal/pkg/nvidia"
)

// =============================================================================
// Mock Package Manager for Package Installation Tests
// =============================================================================

// PackageMockManager implements pkg.Manager for testing package installation.
type PackageMockManager struct {
	name   string
	family constants.DistroFamily

	// Error injection
	installErr error
	removeErr  error

	// Tracking calls
	installCalled   bool
	removeCalled    bool
	installPackages []string
	removePackages  []string
	installCount    int
	removeCount     int

	// Options tracking
	lastInstallOpts pkg.InstallOptions
	lastRemoveOpts  pkg.RemoveOptions

	// Callback for custom behavior
	installCallback func(ctx context.Context, opts pkg.InstallOptions, packages ...string) error
}

// NewPackageMockManager creates a new mock package manager for testing.
func NewPackageMockManager() *PackageMockManager {
	return &PackageMockManager{
		name:            "apt",
		family:          constants.FamilyDebian,
		installPackages: make([]string, 0),
		removePackages:  make([]string, 0),
	}
}

// SetInstallError sets an error to return from Install.
func (m *PackageMockManager) SetInstallError(err error) {
	m.installErr = err
}

// SetRemoveError sets an error to return from Remove.
func (m *PackageMockManager) SetRemoveError(err error) {
	m.removeErr = err
}

// SetFamily sets the distribution family for the mock.
func (m *PackageMockManager) SetFamily(family constants.DistroFamily) {
	m.family = family
}

// Reset clears tracking data.
func (m *PackageMockManager) Reset() {
	m.installCalled = false
	m.removeCalled = false
	m.installPackages = make([]string, 0)
	m.removePackages = make([]string, 0)
	m.installCount = 0
	m.removeCount = 0
}

// SetInstallCallback sets a callback to be called during Install.
func (m *PackageMockManager) SetInstallCallback(fn func(ctx context.Context, opts pkg.InstallOptions, packages ...string) error) {
	m.installCallback = fn
}

// Install implements pkg.Manager.
func (m *PackageMockManager) Install(ctx context.Context, opts pkg.InstallOptions, packages ...string) error {
	m.installCalled = true
	m.installCount++
	m.installPackages = append(m.installPackages, packages...)
	m.lastInstallOpts = opts
	if m.installCallback != nil {
		return m.installCallback(ctx, opts, packages...)
	}
	return m.installErr
}

// Remove implements pkg.Manager.
func (m *PackageMockManager) Remove(ctx context.Context, opts pkg.RemoveOptions, packages ...string) error {
	m.removeCalled = true
	m.removeCount++
	m.removePackages = append(m.removePackages, packages...)
	m.lastRemoveOpts = opts
	return m.removeErr
}

// AddRepository implements pkg.Manager.
func (m *PackageMockManager) AddRepository(ctx context.Context, repo pkg.Repository) error {
	return nil
}

// RemoveRepository implements pkg.Manager.
func (m *PackageMockManager) RemoveRepository(ctx context.Context, name string) error {
	return nil
}

// Update implements pkg.Manager.
func (m *PackageMockManager) Update(ctx context.Context, opts pkg.UpdateOptions) error {
	return nil
}

// Upgrade implements pkg.Manager.
func (m *PackageMockManager) Upgrade(ctx context.Context, opts pkg.InstallOptions, packages ...string) error {
	return nil
}

// IsInstalled implements pkg.Manager.
func (m *PackageMockManager) IsInstalled(ctx context.Context, pkgName string) (bool, error) {
	return false, nil
}

// Search implements pkg.Manager.
func (m *PackageMockManager) Search(ctx context.Context, query string, opts pkg.SearchOptions) ([]pkg.Package, error) {
	return nil, nil
}

// Info implements pkg.Manager.
func (m *PackageMockManager) Info(ctx context.Context, pkgName string) (*pkg.Package, error) {
	return nil, nil
}

// ListInstalled implements pkg.Manager.
func (m *PackageMockManager) ListInstalled(ctx context.Context) ([]pkg.Package, error) {
	return nil, nil
}

// ListUpgradable implements pkg.Manager.
func (m *PackageMockManager) ListUpgradable(ctx context.Context) ([]pkg.Package, error) {
	return nil, nil
}

// ListRepositories implements pkg.Manager.
func (m *PackageMockManager) ListRepositories(ctx context.Context) ([]pkg.Repository, error) {
	return nil, nil
}

// EnableRepository implements pkg.Manager.
func (m *PackageMockManager) EnableRepository(ctx context.Context, name string) error {
	return nil
}

// DisableRepository implements pkg.Manager.
func (m *PackageMockManager) DisableRepository(ctx context.Context, name string) error {
	return nil
}

// RefreshRepositories implements pkg.Manager.
func (m *PackageMockManager) RefreshRepositories(ctx context.Context) error {
	return nil
}

// Clean implements pkg.Manager.
func (m *PackageMockManager) Clean(ctx context.Context) error {
	return nil
}

// AutoRemove implements pkg.Manager.
func (m *PackageMockManager) AutoRemove(ctx context.Context) error {
	return nil
}

// Verify implements pkg.Manager.
func (m *PackageMockManager) Verify(ctx context.Context, pkgName string) (bool, error) {
	return false, nil
}

// Name implements pkg.Manager.
func (m *PackageMockManager) Name() string {
	return m.name
}

// Family implements pkg.Manager.
func (m *PackageMockManager) Family() constants.DistroFamily {
	return m.family
}

// IsAvailable implements pkg.Manager.
func (m *PackageMockManager) IsAvailable() bool {
	return true
}

// Ensure PackageMockManager implements pkg.Manager.
var _ pkg.Manager = (*PackageMockManager)(nil)

// =============================================================================
// Test Distribution Helpers
// =============================================================================

// newTestUbuntuDistro creates a test Ubuntu distribution.
func newTestUbuntuDistro() *distro.Distribution {
	return &distro.Distribution{
		ID:              "ubuntu",
		Name:            "Ubuntu",
		Version:         "22.04 LTS (Jammy Jellyfish)",
		VersionID:       "22.04",
		VersionCodename: "jammy",
		PrettyName:      "Ubuntu 22.04 LTS",
		Family:          constants.FamilyDebian,
	}
}

// newTestFedoraDistro creates a test Fedora distribution.
func newTestFedoraDistro() *distro.Distribution {
	return &distro.Distribution{
		ID:              "fedora",
		Name:            "Fedora Linux",
		Version:         "40 (Workstation Edition)",
		VersionID:       "40",
		VersionCodename: "",
		PrettyName:      "Fedora Linux 40 (Workstation Edition)",
		Family:          constants.FamilyRHEL,
	}
}

// newTestArchDistro creates a test Arch Linux distribution.
func newTestArchDistro() *distro.Distribution {
	return &distro.Distribution{
		ID:         "arch",
		Name:       "Arch Linux",
		PrettyName: "Arch Linux",
		Family:     constants.FamilyArch,
	}
}

// newTestSUSEDistro creates a test openSUSE distribution.
func newTestSUSEDistro() *distro.Distribution {
	return &distro.Distribution{
		ID:         "opensuse-leap",
		Name:       "openSUSE Leap",
		VersionID:  "15.5",
		PrettyName: "openSUSE Leap 15.5",
		Family:     constants.FamilySUSE,
	}
}

// =============================================================================
// PackageInstallationStep Constructor Tests
// =============================================================================

func TestNewPackageInstallationStep_DefaultOptions(t *testing.T) {
	step := NewPackageInstallationStep()

	assert.Equal(t, "packages", step.Name())
	assert.Equal(t, "Install NVIDIA packages", step.Description())
	assert.True(t, step.CanRollback())
	assert.Empty(t, step.additionalPackages)
	assert.False(t, step.skipDependencies)
	assert.Equal(t, 0, step.batchSize)
	assert.Nil(t, step.preInstallHook)
	assert.Nil(t, step.postInstallHook)
}

func TestNewPackageInstallationStep_WithOptions(t *testing.T) {
	hookCalled := false
	preHook := func(ctx *install.Context) error {
		hookCalled = true
		return nil
	}
	postHook := func(ctx *install.Context) error {
		return nil
	}

	step := NewPackageInstallationStep(
		WithAdditionalPackages("extra-pkg1", "extra-pkg2"),
		WithSkipDependencies(true),
		WithBatchSize(5),
		WithPreInstallHook(preHook),
		WithPostInstallHook(postHook),
	)

	assert.Equal(t, []string{"extra-pkg1", "extra-pkg2"}, step.additionalPackages)
	assert.True(t, step.skipDependencies)
	assert.Equal(t, 5, step.batchSize)
	assert.NotNil(t, step.preInstallHook)
	assert.NotNil(t, step.postInstallHook)

	// Verify pre-hook is callable
	_ = step.preInstallHook(nil)
	assert.True(t, hookCalled)
}

func TestPackageInstallationStep_Name(t *testing.T) {
	step := NewPackageInstallationStep()
	assert.Equal(t, "packages", step.Name())
}

func TestPackageInstallationStep_Description(t *testing.T) {
	step := NewPackageInstallationStep()
	assert.Equal(t, "Install NVIDIA packages", step.Description())
}

func TestPackageInstallationStep_CanRollback(t *testing.T) {
	step := NewPackageInstallationStep()
	assert.True(t, step.CanRollback())
}

// =============================================================================
// PackageInstallationStep Execute Tests
// =============================================================================

func TestPackageInstallationStep_Execute_Success(t *testing.T) {
	mockPM := NewPackageMockManager()
	step := NewPackageInstallationStep()

	ctx := install.NewContext(
		install.WithPackageManager(mockPM),
		install.WithDistroInfo(newTestUbuntuDistro()),
		install.WithDriverVersion("550"),
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusCompleted, result.Status)
	assert.Contains(t, result.Message, "packages installed successfully")
	assert.True(t, mockPM.installCalled)
	assert.Greater(t, len(mockPM.installPackages), 0)

	// Check state was set
	assert.True(t, ctx.GetStateBool(StatePackagesInstalled))

	// Check installed packages were stored
	packagesRaw, ok := ctx.GetState(StateInstalledPackages)
	assert.True(t, ok)
	packages, ok := packagesRaw.([]string)
	assert.True(t, ok)
	assert.Greater(t, len(packages), 0)

	// Check install time was stored
	_, ok = ctx.GetState(StatePackageInstallTime)
	assert.True(t, ok)
}

func TestPackageInstallationStep_Execute_DryRun(t *testing.T) {
	mockPM := NewPackageMockManager()
	step := NewPackageInstallationStep()

	ctx := install.NewContext(
		install.WithPackageManager(mockPM),
		install.WithDistroInfo(newTestUbuntuDistro()),
		install.WithDriverVersion("550"),
		install.WithDryRun(true),
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusCompleted, result.Status)
	assert.Contains(t, result.Message, "dry run")
	// Should NOT have actually installed anything
	assert.False(t, mockPM.installCalled)

	// State should not be set in dry run
	assert.False(t, ctx.GetStateBool(StatePackagesInstalled))
}

func TestPackageInstallationStep_Execute_Cancelled(t *testing.T) {
	mockPM := NewPackageMockManager()
	step := NewPackageInstallationStep()

	ctx := install.NewContext(
		install.WithPackageManager(mockPM),
		install.WithDistroInfo(newTestUbuntuDistro()),
		install.WithDriverVersion("550"),
	)
	ctx.Cancel() // Cancel immediately

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusFailed, result.Status)
	assert.Contains(t, result.Message, "cancelled")
	assert.True(t, errors.Is(result.Error, context.Canceled))
	assert.False(t, mockPM.installCalled)
}

func TestPackageInstallationStep_Execute_NoPackageManager(t *testing.T) {
	step := NewPackageInstallationStep()

	ctx := install.NewContext(
		install.WithDistroInfo(newTestUbuntuDistro()),
		install.WithDriverVersion("550"),
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusFailed, result.Status)
	assert.Contains(t, result.Message, "validation failed")
	assert.Error(t, result.Error)
	assert.Contains(t, result.Error.Error(), "package manager")
}

func TestPackageInstallationStep_Execute_NoDistroInfo(t *testing.T) {
	mockPM := NewPackageMockManager()
	step := NewPackageInstallationStep()

	ctx := install.NewContext(
		install.WithPackageManager(mockPM),
		install.WithDriverVersion("550"),
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusFailed, result.Status)
	assert.Contains(t, result.Message, "validation failed")
	assert.Error(t, result.Error)
	assert.Contains(t, result.Error.Error(), "distribution info")
}

func TestPackageInstallationStep_Execute_InstallFails(t *testing.T) {
	mockPM := NewPackageMockManager()
	mockPM.SetInstallError(errors.New("installation failed: disk full"))
	step := NewPackageInstallationStep()

	ctx := install.NewContext(
		install.WithPackageManager(mockPM),
		install.WithDistroInfo(newTestUbuntuDistro()),
		install.WithDriverVersion("550"),
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusFailed, result.Status)
	assert.Contains(t, result.Message, "failed to install packages")
	assert.Error(t, result.Error)
	assert.True(t, mockPM.installCalled)

	// State should not be set on failure
	assert.False(t, ctx.GetStateBool(StatePackagesInstalled))
}

func TestPackageInstallationStep_Execute_WithDriverVersion(t *testing.T) {
	mockPM := NewPackageMockManager()
	step := NewPackageInstallationStep()

	ctx := install.NewContext(
		install.WithPackageManager(mockPM),
		install.WithDistroInfo(newTestUbuntuDistro()),
		install.WithDriverVersion("535"),
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusCompleted, result.Status)
	assert.True(t, mockPM.installCalled)

	// Should contain the version-specific package
	assert.Contains(t, mockPM.installPackages, "nvidia-driver-535")
}

func TestPackageInstallationStep_Execute_WithComponents(t *testing.T) {
	mockPM := NewPackageMockManager()
	step := NewPackageInstallationStep()

	ctx := install.NewContext(
		install.WithPackageManager(mockPM),
		install.WithDistroInfo(newTestUbuntuDistro()),
		install.WithComponents([]string{string(nvidia.ComponentUtils), string(nvidia.ComponentSettings)}),
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusCompleted, result.Status)
	assert.True(t, mockPM.installCalled)
	assert.Greater(t, len(mockPM.installPackages), 0)
}

func TestPackageInstallationStep_Execute_WithAdditionalPackages(t *testing.T) {
	mockPM := NewPackageMockManager()
	step := NewPackageInstallationStep(
		WithAdditionalPackages("custom-pkg", "another-pkg"),
	)

	ctx := install.NewContext(
		install.WithPackageManager(mockPM),
		install.WithDistroInfo(newTestUbuntuDistro()),
		install.WithDriverVersion("550"),
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusCompleted, result.Status)
	assert.True(t, mockPM.installCalled)
	assert.Contains(t, mockPM.installPackages, "custom-pkg")
	assert.Contains(t, mockPM.installPackages, "another-pkg")
}

func TestPackageInstallationStep_Execute_WithPreInstallHook(t *testing.T) {
	mockPM := NewPackageMockManager()
	hookCalled := false
	step := NewPackageInstallationStep(
		WithPreInstallHook(func(ctx *install.Context) error {
			hookCalled = true
			return nil
		}),
	)

	ctx := install.NewContext(
		install.WithPackageManager(mockPM),
		install.WithDistroInfo(newTestUbuntuDistro()),
		install.WithDriverVersion("550"),
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusCompleted, result.Status)
	assert.True(t, hookCalled)
	assert.True(t, mockPM.installCalled)
}

func TestPackageInstallationStep_Execute_WithPostInstallHook(t *testing.T) {
	mockPM := NewPackageMockManager()
	hookCalled := false
	step := NewPackageInstallationStep(
		WithPostInstallHook(func(ctx *install.Context) error {
			hookCalled = true
			return nil
		}),
	)

	ctx := install.NewContext(
		install.WithPackageManager(mockPM),
		install.WithDistroInfo(newTestUbuntuDistro()),
		install.WithDriverVersion("550"),
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusCompleted, result.Status)
	assert.True(t, hookCalled)
	assert.True(t, mockPM.installCalled)
}

func TestPackageInstallationStep_Execute_PreInstallHookFails(t *testing.T) {
	mockPM := NewPackageMockManager()
	step := NewPackageInstallationStep(
		WithPreInstallHook(func(ctx *install.Context) error {
			return errors.New("pre-install hook error")
		}),
	)

	ctx := install.NewContext(
		install.WithPackageManager(mockPM),
		install.WithDistroInfo(newTestUbuntuDistro()),
		install.WithDriverVersion("550"),
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusFailed, result.Status)
	assert.Contains(t, result.Message, "pre-install hook failed")
	assert.Error(t, result.Error)
	// Should NOT have installed packages
	assert.False(t, mockPM.installCalled)
}

func TestPackageInstallationStep_Execute_PostInstallHookFails(t *testing.T) {
	mockPM := NewPackageMockManager()
	step := NewPackageInstallationStep(
		WithPostInstallHook(func(ctx *install.Context) error {
			return errors.New("post-install hook error")
		}),
	)

	ctx := install.NewContext(
		install.WithPackageManager(mockPM),
		install.WithDistroInfo(newTestUbuntuDistro()),
		install.WithDriverVersion("550"),
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusFailed, result.Status)
	assert.Contains(t, result.Message, "post-install hook failed")
	assert.Error(t, result.Error)
	// Should have installed packages
	assert.True(t, mockPM.installCalled)
	// Should have tried to rollback
	assert.True(t, mockPM.removeCalled)
}

func TestPackageInstallationStep_Execute_WithBatchSize(t *testing.T) {
	mockPM := NewPackageMockManager()
	step := NewPackageInstallationStep(
		WithBatchSize(1),
		WithAdditionalPackages("pkg1", "pkg2", "pkg3"),
	)

	ctx := install.NewContext(
		install.WithPackageManager(mockPM),
		install.WithDistroInfo(newTestUbuntuDistro()),
		install.WithDriverVersion("550"),
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusCompleted, result.Status)
	// With batch size of 1, should have multiple Install calls
	assert.Greater(t, mockPM.installCount, 1)
}

func TestPackageInstallationStep_Execute_AllDistroFamilies(t *testing.T) {
	tests := []struct {
		name   string
		distro *distro.Distribution
		family constants.DistroFamily
	}{
		{
			name:   "Debian (Ubuntu)",
			distro: newTestUbuntuDistro(),
			family: constants.FamilyDebian,
		},
		{
			name:   "RHEL (Fedora)",
			distro: newTestFedoraDistro(),
			family: constants.FamilyRHEL,
		},
		{
			name:   "Arch Linux",
			distro: newTestArchDistro(),
			family: constants.FamilyArch,
		},
		{
			name:   "SUSE (openSUSE)",
			distro: newTestSUSEDistro(),
			family: constants.FamilySUSE,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockPM := NewPackageMockManager()
			mockPM.SetFamily(tt.family)
			step := NewPackageInstallationStep()

			ctx := install.NewContext(
				install.WithPackageManager(mockPM),
				install.WithDistroInfo(tt.distro),
				install.WithDriverVersion("550"),
			)

			result := step.Execute(ctx)

			assert.Equal(t, install.StepStatusCompleted, result.Status)
			assert.True(t, mockPM.installCalled)
			assert.Greater(t, len(mockPM.installPackages), 0)
		})
	}
}

func TestPackageInstallationStep_Execute_NoPackagesToInstall(t *testing.T) {
	mockPM := NewPackageMockManager()
	step := NewPackageInstallationStep()

	// Create a distro with unknown family that won't have a package set
	unknownDistro := &distro.Distribution{
		ID:         "unknown",
		Name:       "Unknown Linux",
		PrettyName: "Unknown Linux",
		Family:     constants.FamilyUnknown,
	}

	ctx := install.NewContext(
		install.WithPackageManager(mockPM),
		install.WithDistroInfo(unknownDistro),
		install.WithDriverVersion("550"),
	)

	result := step.Execute(ctx)

	// Should fail because no package set is available for unknown distro
	assert.Equal(t, install.StepStatusFailed, result.Status)
	assert.Contains(t, result.Message, "failed to compute packages")
}

func TestPackageInstallationStep_Execute_UnknownComponent(t *testing.T) {
	mockPM := NewPackageMockManager()
	step := NewPackageInstallationStep(
		WithAdditionalPackages("fallback-pkg"), // So we have something to install
	)

	ctx := install.NewContext(
		install.WithPackageManager(mockPM),
		install.WithDistroInfo(newTestUbuntuDistro()),
		install.WithComponents([]string{"unknown-component"}),
	)

	result := step.Execute(ctx)

	// Should still succeed, just skip the unknown component
	assert.Equal(t, install.StepStatusCompleted, result.Status)
	assert.True(t, mockPM.installCalled)
	assert.Contains(t, mockPM.installPackages, "fallback-pkg")
}

func TestPackageInstallationStep_Execute_Duration(t *testing.T) {
	mockPM := NewPackageMockManager()
	step := NewPackageInstallationStep()

	ctx := install.NewContext(
		install.WithPackageManager(mockPM),
		install.WithDistroInfo(newTestUbuntuDistro()),
		install.WithDriverVersion("550"),
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusCompleted, result.Status)
	assert.Greater(t, result.Duration.Nanoseconds(), int64(0))
}

func TestPackageInstallationStep_Execute_CancelledBeforeInstall(t *testing.T) {
	mockPM := NewPackageMockManager()
	step := NewPackageInstallationStep(
		WithPreInstallHook(func(ctx *install.Context) error {
			ctx.Cancel() // Cancel after validation but before install
			return nil
		}),
	)

	ctx := install.NewContext(
		install.WithPackageManager(mockPM),
		install.WithDistroInfo(newTestUbuntuDistro()),
		install.WithDriverVersion("550"),
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusFailed, result.Status)
	assert.Contains(t, result.Message, "cancelled")
	// The hook ran but then cancellation was detected before install
	assert.False(t, mockPM.installCalled)
}

func TestPackageInstallationStep_Execute_CancelledDuringBatch(t *testing.T) {
	mockPM := NewPackageMockManager()

	ctx := install.NewContext(
		install.WithPackageManager(mockPM),
		install.WithDistroInfo(newTestUbuntuDistro()),
		install.WithDriverVersion("550"),
	)

	// Use callback to cancel on second install call
	mockPM.SetInstallCallback(func(c context.Context, opts pkg.InstallOptions, packages ...string) error {
		if mockPM.installCount >= 2 {
			ctx.Cancel()
			return context.Canceled
		}
		return nil
	})

	step := NewPackageInstallationStep(
		WithBatchSize(1),
		WithAdditionalPackages("pkg1", "pkg2", "pkg3"),
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusFailed, result.Status)
	// First batch should have been installed, then cancelled
	assert.True(t, mockPM.installCalled)
}

// =============================================================================
// PackageInstallationStep Rollback Tests
// =============================================================================

func TestPackageInstallationStep_Rollback_Success(t *testing.T) {
	mockPM := NewPackageMockManager()
	step := NewPackageInstallationStep()

	ctx := install.NewContext(
		install.WithPackageManager(mockPM),
		install.WithDistroInfo(newTestUbuntuDistro()),
		install.WithDriverVersion("550"),
	)

	// First execute the step
	result := step.Execute(ctx)
	require.Equal(t, install.StepStatusCompleted, result.Status)

	// Reset tracking
	mockPM.Reset()

	// Now rollback
	err := step.Rollback(ctx)

	assert.NoError(t, err)
	assert.True(t, mockPM.removeCalled)
	assert.Greater(t, len(mockPM.removePackages), 0)

	// State should be cleared
	assert.False(t, ctx.GetStateBool(StatePackagesInstalled))
	_, ok := ctx.GetState(StateInstalledPackages)
	assert.False(t, ok)
	_, ok = ctx.GetState(StatePackageInstallTime)
	assert.False(t, ok)
}

func TestPackageInstallationStep_Rollback_NoPackagesInstalled(t *testing.T) {
	mockPM := NewPackageMockManager()
	step := NewPackageInstallationStep()

	ctx := install.NewContext(
		install.WithPackageManager(mockPM),
		install.WithDistroInfo(newTestUbuntuDistro()),
	)

	// Don't execute, just rollback directly
	err := step.Rollback(ctx)

	assert.NoError(t, err)
	assert.False(t, mockPM.removeCalled)
}

func TestPackageInstallationStep_Rollback_RemoveFails(t *testing.T) {
	mockPM := NewPackageMockManager()
	step := NewPackageInstallationStep()

	ctx := install.NewContext(
		install.WithPackageManager(mockPM),
		install.WithDistroInfo(newTestUbuntuDistro()),
		install.WithDriverVersion("550"),
	)

	// First execute the step
	result := step.Execute(ctx)
	require.Equal(t, install.StepStatusCompleted, result.Status)

	// Set error for rollback
	mockPM.SetRemoveError(errors.New("remove failed: package in use"))

	err := step.Rollback(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to remove packages")
}

func TestPackageInstallationStep_Rollback_DryRun(t *testing.T) {
	mockPM := NewPackageMockManager()
	step := NewPackageInstallationStep()

	ctx := install.NewContext(
		install.WithPackageManager(mockPM),
		install.WithDistroInfo(newTestUbuntuDistro()),
		install.WithDriverVersion("550"),
		install.WithDryRun(true),
	)

	// Execute in dry run mode
	result := step.Execute(ctx)
	require.Equal(t, install.StepStatusCompleted, result.Status)

	// Rollback should have nothing to do
	err := step.Rollback(ctx)

	assert.NoError(t, err)
	assert.False(t, mockPM.removeCalled)
}

func TestPackageInstallationStep_Rollback_NilPackageManager(t *testing.T) {
	step := NewPackageInstallationStep()

	ctx := install.NewContext()
	ctx.SetState(StatePackagesInstalled, true)
	ctx.SetState(StateInstalledPackages, []string{"pkg1", "pkg2"})

	err := step.Rollback(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "package manager not available")
}

func TestPackageInstallationStep_Rollback_InvalidPackagesState(t *testing.T) {
	mockPM := NewPackageMockManager()
	step := NewPackageInstallationStep()

	ctx := install.NewContext(
		install.WithPackageManager(mockPM),
	)

	// Set invalid state
	ctx.SetState(StatePackagesInstalled, true)
	ctx.SetState(StateInstalledPackages, "not-a-slice") // Wrong type

	err := step.Rollback(ctx)

	assert.NoError(t, err) // Should handle gracefully
	assert.False(t, mockPM.removeCalled)
}

// =============================================================================
// PackageInstallationStep Validate Tests
// =============================================================================

func TestPackageInstallationStep_Validate_Success(t *testing.T) {
	mockPM := NewPackageMockManager()
	step := NewPackageInstallationStep()

	ctx := install.NewContext(
		install.WithPackageManager(mockPM),
		install.WithDistroInfo(newTestUbuntuDistro()),
		install.WithDriverVersion("550"),
	)

	err := step.Validate(ctx)

	assert.NoError(t, err)
}

func TestPackageInstallationStep_Validate_NoPackageManager(t *testing.T) {
	step := NewPackageInstallationStep()

	ctx := install.NewContext(
		install.WithDistroInfo(newTestUbuntuDistro()),
		install.WithDriverVersion("550"),
	)

	err := step.Validate(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "package manager is required")
}

func TestPackageInstallationStep_Validate_NoDistroInfo(t *testing.T) {
	mockPM := NewPackageMockManager()
	step := NewPackageInstallationStep()

	ctx := install.NewContext(
		install.WithPackageManager(mockPM),
		install.WithDriverVersion("550"),
	)

	err := step.Validate(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "distribution info is required")
}

func TestPackageInstallationStep_Validate_NoPackagesToInstall(t *testing.T) {
	mockPM := NewPackageMockManager()
	step := NewPackageInstallationStep()

	ctx := install.NewContext(
		install.WithPackageManager(mockPM),
		install.WithDistroInfo(newTestUbuntuDistro()),
		// No driver version, no components, no additional packages
	)

	err := step.Validate(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "at least one component")
}

func TestPackageInstallationStep_Validate_WithOnlyAdditionalPackages(t *testing.T) {
	mockPM := NewPackageMockManager()
	step := NewPackageInstallationStep(
		WithAdditionalPackages("custom-pkg"),
	)

	ctx := install.NewContext(
		install.WithPackageManager(mockPM),
		install.WithDistroInfo(newTestUbuntuDistro()),
		// No driver version, no components - but we have additional packages
	)

	err := step.Validate(ctx)

	assert.NoError(t, err)
}

func TestPackageInstallationStep_Validate_WithOnlyComponents(t *testing.T) {
	mockPM := NewPackageMockManager()
	step := NewPackageInstallationStep()

	ctx := install.NewContext(
		install.WithPackageManager(mockPM),
		install.WithDistroInfo(newTestUbuntuDistro()),
		install.WithComponents([]string{string(nvidia.ComponentUtils)}),
	)

	err := step.Validate(ctx)

	assert.NoError(t, err)
}

// =============================================================================
// PackageInstallationStep computePackages Tests
// =============================================================================

func TestPackageInstallationStep_ComputePackages(t *testing.T) {
	mockPM := NewPackageMockManager()
	step := NewPackageInstallationStep()

	ctx := install.NewContext(
		install.WithPackageManager(mockPM),
		install.WithDistroInfo(newTestUbuntuDistro()),
		install.WithDriverVersion("550"),
	)

	packages, err := step.computePackages(ctx)

	assert.NoError(t, err)
	assert.Greater(t, len(packages), 0)
	assert.Contains(t, packages, "nvidia-driver-550")
}

func TestPackageInstallationStep_ComputePackages_WithVersion(t *testing.T) {
	mockPM := NewPackageMockManager()
	step := NewPackageInstallationStep()

	ctx := install.NewContext(
		install.WithPackageManager(mockPM),
		install.WithDistroInfo(newTestUbuntuDistro()),
		install.WithDriverVersion("535"),
	)

	packages, err := step.computePackages(ctx)

	assert.NoError(t, err)
	assert.Contains(t, packages, "nvidia-driver-535")
}

func TestPackageInstallationStep_ComputePackages_WithComponents(t *testing.T) {
	mockPM := NewPackageMockManager()
	step := NewPackageInstallationStep()

	ctx := install.NewContext(
		install.WithPackageManager(mockPM),
		install.WithDistroInfo(newTestUbuntuDistro()),
		install.WithComponents([]string{string(nvidia.ComponentCUDA), string(nvidia.ComponentSettings)}),
	)

	packages, err := step.computePackages(ctx)

	assert.NoError(t, err)
	assert.Greater(t, len(packages), 0)
	// Should include CUDA and settings packages
	assert.Contains(t, packages, "nvidia-cuda-toolkit")
	assert.Contains(t, packages, "nvidia-settings")
}

func TestPackageInstallationStep_ComputePackages_Deduplication(t *testing.T) {
	mockPM := NewPackageMockManager()
	step := NewPackageInstallationStep(
		WithAdditionalPackages("nvidia-driver-550"), // Duplicate of what driver version gives
	)

	ctx := install.NewContext(
		install.WithPackageManager(mockPM),
		install.WithDistroInfo(newTestUbuntuDistro()),
		install.WithDriverVersion("550"),
	)

	packages, err := step.computePackages(ctx)

	assert.NoError(t, err)

	// Count occurrences of nvidia-driver-550
	count := 0
	for _, p := range packages {
		if p == "nvidia-driver-550" {
			count++
		}
	}
	assert.Equal(t, 1, count, "nvidia-driver-550 should appear only once")
}

func TestPackageInstallationStep_ComputePackages_NilDistro(t *testing.T) {
	mockPM := NewPackageMockManager()
	step := NewPackageInstallationStep()

	ctx := install.NewContext(
		install.WithPackageManager(mockPM),
		install.WithDriverVersion("550"),
	)

	packages, err := step.computePackages(ctx)

	assert.Error(t, err)
	assert.Nil(t, packages)
	assert.Contains(t, err.Error(), "no package set available")
}

func TestPackageInstallationStep_ComputePackages_UnknownFamily(t *testing.T) {
	mockPM := NewPackageMockManager()
	step := NewPackageInstallationStep()

	unknownDistro := &distro.Distribution{
		ID:         "unknown",
		Name:       "Unknown Linux",
		PrettyName: "Unknown Linux",
		Family:     constants.FamilyUnknown,
	}

	ctx := install.NewContext(
		install.WithPackageManager(mockPM),
		install.WithDistroInfo(unknownDistro),
		install.WithDriverVersion("550"),
	)

	packages, err := step.computePackages(ctx)

	assert.Error(t, err)
	assert.Nil(t, packages)
}

func TestPackageInstallationStep_ComputePackages_OnlyAdditionalPackages(t *testing.T) {
	mockPM := NewPackageMockManager()
	step := NewPackageInstallationStep(
		WithAdditionalPackages("custom-pkg-1", "custom-pkg-2"),
	)

	ctx := install.NewContext(
		install.WithPackageManager(mockPM),
		install.WithDistroInfo(newTestUbuntuDistro()),
		// No driver version, no components
	)

	packages, err := step.computePackages(ctx)

	assert.NoError(t, err)
	assert.Equal(t, []string{"custom-pkg-1", "custom-pkg-2"}, packages)
}

// =============================================================================
// PackageInstallationStep Interface Compliance Tests
// =============================================================================

func TestPackageInstallationStep_InterfaceCompliance(t *testing.T) {
	var _ install.Step = (*PackageInstallationStep)(nil)
}

// =============================================================================
// PackageInstallationStep State Keys Tests
// =============================================================================

func TestPackageInstallationStep_StateKeys(t *testing.T) {
	assert.Equal(t, "packages_installed", StatePackagesInstalled)
	assert.Equal(t, "installed_packages", StateInstalledPackages)
	assert.Equal(t, "package_install_time", StatePackageInstallTime)
}

// =============================================================================
// PackageInstallationStep Options Tests
// =============================================================================

func TestPackageInstallationStep_Options(t *testing.T) {
	t.Run("WithAdditionalPackages adds packages", func(t *testing.T) {
		step := NewPackageInstallationStep(
			WithAdditionalPackages("pkg1"),
			WithAdditionalPackages("pkg2", "pkg3"), // Multiple calls should append
		)
		assert.Equal(t, []string{"pkg1", "pkg2", "pkg3"}, step.additionalPackages)
	})

	t.Run("WithSkipDependencies sets skipDependencies to true", func(t *testing.T) {
		step := NewPackageInstallationStep(WithSkipDependencies(true))
		assert.True(t, step.skipDependencies)
	})

	t.Run("WithSkipDependencies sets skipDependencies to false", func(t *testing.T) {
		step := NewPackageInstallationStep(WithSkipDependencies(false))
		assert.False(t, step.skipDependencies)
	})

	t.Run("WithBatchSize sets batch size", func(t *testing.T) {
		step := NewPackageInstallationStep(WithBatchSize(10))
		assert.Equal(t, 10, step.batchSize)
	})

	t.Run("WithPreInstallHook sets hook", func(t *testing.T) {
		hook := func(ctx *install.Context) error { return nil }
		step := NewPackageInstallationStep(WithPreInstallHook(hook))
		assert.NotNil(t, step.preInstallHook)
	})

	t.Run("WithPostInstallHook sets hook", func(t *testing.T) {
		hook := func(ctx *install.Context) error { return nil }
		step := NewPackageInstallationStep(WithPostInstallHook(hook))
		assert.NotNil(t, step.postInstallHook)
	})

	t.Run("default values", func(t *testing.T) {
		step := NewPackageInstallationStep()
		assert.Empty(t, step.additionalPackages)
		assert.False(t, step.skipDependencies)
		assert.Equal(t, 0, step.batchSize)
		assert.Nil(t, step.preInstallHook)
		assert.Nil(t, step.postInstallHook)
	})
}

// =============================================================================
// PackageInstallationStep Full Workflow Tests
// =============================================================================

func TestPackageInstallationStep_FullWorkflow_ExecuteAndRollback(t *testing.T) {
	mockPM := NewPackageMockManager()
	step := NewPackageInstallationStep()

	ctx := install.NewContext(
		install.WithPackageManager(mockPM),
		install.WithDistroInfo(newTestUbuntuDistro()),
		install.WithDriverVersion("550"),
	)

	// Execute
	result := step.Execute(ctx)
	assert.Equal(t, install.StepStatusCompleted, result.Status)
	assert.True(t, ctx.GetStateBool(StatePackagesInstalled))

	// Get the packages that were stored
	packagesRaw, ok := ctx.GetState(StateInstalledPackages)
	require.True(t, ok)
	packages, ok := packagesRaw.([]string)
	require.True(t, ok)
	assert.Greater(t, len(packages), 0)

	// Reset mock tracking
	mockPM.Reset()

	// Rollback
	err := step.Rollback(ctx)
	assert.NoError(t, err)
	assert.True(t, mockPM.removeCalled)

	// Verify the same packages were removed
	assert.ElementsMatch(t, packages, mockPM.removePackages)

	// State should be cleared
	assert.False(t, ctx.GetStateBool(StatePackagesInstalled))
}

func TestPackageInstallationStep_FullWorkflow_WithBatches(t *testing.T) {
	mockPM := NewPackageMockManager()
	step := NewPackageInstallationStep(
		WithBatchSize(2),
		WithAdditionalPackages("pkg1", "pkg2", "pkg3", "pkg4", "pkg5"),
	)

	ctx := install.NewContext(
		install.WithPackageManager(mockPM),
		install.WithDistroInfo(newTestUbuntuDistro()),
	)

	// Execute
	result := step.Execute(ctx)
	assert.Equal(t, install.StepStatusCompleted, result.Status)

	// With batch size 2 and 5 extra packages + potentially driver packages,
	// we should have multiple install calls
	assert.GreaterOrEqual(t, mockPM.installCount, 3)
}

func TestPackageInstallationStep_FullWorkflow_WithHooks(t *testing.T) {
	mockPM := NewPackageMockManager()

	preHookCalled := false
	postHookCalled := false
	callOrder := make([]string, 0)

	// Use callback to track install order
	mockPM.SetInstallCallback(func(ctx context.Context, opts pkg.InstallOptions, packages ...string) error {
		callOrder = append(callOrder, "install")
		return nil
	})

	step := NewPackageInstallationStep(
		WithPreInstallHook(func(ctx *install.Context) error {
			preHookCalled = true
			callOrder = append(callOrder, "pre")
			return nil
		}),
		WithPostInstallHook(func(ctx *install.Context) error {
			postHookCalled = true
			callOrder = append(callOrder, "post")
			return nil
		}),
	)

	ctx := install.NewContext(
		install.WithPackageManager(mockPM),
		install.WithDistroInfo(newTestUbuntuDistro()),
		install.WithDriverVersion("550"),
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusCompleted, result.Status)
	assert.True(t, preHookCalled)
	assert.True(t, postHookCalled)
	assert.Equal(t, []string{"pre", "install", "post"}, callOrder)
}

// =============================================================================
// PackageInstallationStep Install Options Tests
// =============================================================================

func TestPackageInstallationStep_UsesNonInteractiveOptions(t *testing.T) {
	mockPM := NewPackageMockManager()
	step := NewPackageInstallationStep()

	ctx := install.NewContext(
		install.WithPackageManager(mockPM),
		install.WithDistroInfo(newTestUbuntuDistro()),
		install.WithDriverVersion("550"),
	)

	_ = step.Execute(ctx)

	// Verify non-interactive options were used
	assert.True(t, mockPM.lastInstallOpts.NoConfirm)
	assert.False(t, mockPM.lastInstallOpts.Force)
}

func TestPackageInstallationStep_Rollback_UsesAutoRemove(t *testing.T) {
	mockPM := NewPackageMockManager()
	step := NewPackageInstallationStep()

	ctx := install.NewContext(
		install.WithPackageManager(mockPM),
		install.WithDistroInfo(newTestUbuntuDistro()),
		install.WithDriverVersion("550"),
	)

	// Execute first
	result := step.Execute(ctx)
	require.Equal(t, install.StepStatusCompleted, result.Status)

	// Rollback
	_ = step.Rollback(ctx)

	// Verify auto-remove option was used
	assert.True(t, mockPM.lastRemoveOpts.NoConfirm)
	assert.True(t, mockPM.lastRemoveOpts.AutoRemove)
}

// =============================================================================
// PackageInstallationStep Error Recovery Tests
// =============================================================================

func TestPackageInstallationStep_PartialInstallRollback(t *testing.T) {
	// Create a mock that fails on the second batch
	mockPM := NewPackageMockManager()

	// Use callback to fail on second install call
	mockPM.SetInstallCallback(func(ctx context.Context, opts pkg.InstallOptions, packages ...string) error {
		if mockPM.installCount >= 2 {
			return errors.New("second batch failed")
		}
		return nil
	})

	step := NewPackageInstallationStep(
		WithBatchSize(1),
		WithAdditionalPackages("pkg1", "pkg2", "pkg3"),
	)

	ctx := install.NewContext(
		install.WithPackageManager(mockPM),
		install.WithDistroInfo(newTestUbuntuDistro()),
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusFailed, result.Status)
	// Should have tried to remove the successfully installed packages
	assert.True(t, mockPM.removeCalled)
}

// =============================================================================
// Benchmarks
// =============================================================================

func BenchmarkPackageInstallationStep_Execute(b *testing.B) {
	mockPM := NewPackageMockManager()
	step := NewPackageInstallationStep()

	ctx := install.NewContext(
		install.WithPackageManager(mockPM),
		install.WithDistroInfo(newTestUbuntuDistro()),
		install.WithDriverVersion("550"),
	)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Clear state between iterations
		ctx.DeleteState(StatePackagesInstalled)
		ctx.DeleteState(StateInstalledPackages)
		ctx.DeleteState(StatePackageInstallTime)

		step.Execute(ctx)
	}
}

func BenchmarkPackageInstallationStep_Validate(b *testing.B) {
	mockPM := NewPackageMockManager()
	step := NewPackageInstallationStep()

	ctx := install.NewContext(
		install.WithPackageManager(mockPM),
		install.WithDistroInfo(newTestUbuntuDistro()),
		install.WithDriverVersion("550"),
	)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = step.Validate(ctx)
	}
}

func BenchmarkPackageInstallationStep_ComputePackages(b *testing.B) {
	mockPM := NewPackageMockManager()
	step := NewPackageInstallationStep()

	ctx := install.NewContext(
		install.WithPackageManager(mockPM),
		install.WithDistroInfo(newTestUbuntuDistro()),
		install.WithDriverVersion("550"),
		install.WithComponents([]string{
			string(nvidia.ComponentDriver),
			string(nvidia.ComponentCUDA),
			string(nvidia.ComponentSettings),
		}),
	)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = step.computePackages(ctx)
	}
}

// =============================================================================
// Edge Case Tests
// =============================================================================

func TestPackageInstallationStep_EmptyAdditionalPackages(t *testing.T) {
	mockPM := NewPackageMockManager()
	step := NewPackageInstallationStep(
		WithAdditionalPackages(), // Empty
	)

	ctx := install.NewContext(
		install.WithPackageManager(mockPM),
		install.WithDistroInfo(newTestUbuntuDistro()),
		install.WithDriverVersion("550"),
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusCompleted, result.Status)
}

func TestPackageInstallationStep_ZeroBatchSize(t *testing.T) {
	mockPM := NewPackageMockManager()
	step := NewPackageInstallationStep(
		WithBatchSize(0), // Explicitly 0
		WithAdditionalPackages("pkg1", "pkg2", "pkg3"),
	)

	ctx := install.NewContext(
		install.WithPackageManager(mockPM),
		install.WithDistroInfo(newTestUbuntuDistro()),
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusCompleted, result.Status)
	// Should install all at once with batch size 0
	assert.Equal(t, 1, mockPM.installCount)
}

func TestPackageInstallationStep_NilHooks(t *testing.T) {
	mockPM := NewPackageMockManager()
	step := NewPackageInstallationStep(
		WithPreInstallHook(nil),
		WithPostInstallHook(nil),
	)

	ctx := install.NewContext(
		install.WithPackageManager(mockPM),
		install.WithDistroInfo(newTestUbuntuDistro()),
		install.WithDriverVersion("550"),
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusCompleted, result.Status)
}

func TestPackageInstallationStep_VersionWithWhitespace(t *testing.T) {
	mockPM := NewPackageMockManager()
	step := NewPackageInstallationStep()

	ctx := install.NewContext(
		install.WithPackageManager(mockPM),
		install.WithDistroInfo(newTestUbuntuDistro()),
		install.WithDriverVersion("  550  "), // Whitespace should be trimmed by GetPackagesForVersion
	)

	packages, err := step.computePackages(ctx)

	assert.NoError(t, err)
	assert.Contains(t, packages, "nvidia-driver-550")
}

func TestPackageInstallationStep_MultipleComponents(t *testing.T) {
	mockPM := NewPackageMockManager()
	step := NewPackageInstallationStep()

	ctx := install.NewContext(
		install.WithPackageManager(mockPM),
		install.WithDistroInfo(newTestUbuntuDistro()),
		install.WithComponents([]string{
			string(nvidia.ComponentDriver),
			string(nvidia.ComponentCUDA),
			string(nvidia.ComponentCUDNN),
			string(nvidia.ComponentSettings),
			string(nvidia.ComponentOpenCL),
			string(nvidia.ComponentVulkan),
		}),
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusCompleted, result.Status)
	assert.True(t, mockPM.installCalled)
	assert.Greater(t, len(mockPM.installPackages), 5)
}

func TestPackageInstallationStep_StateInstallTime(t *testing.T) {
	mockPM := NewPackageMockManager()
	step := NewPackageInstallationStep()

	ctx := install.NewContext(
		install.WithPackageManager(mockPM),
		install.WithDistroInfo(newTestUbuntuDistro()),
		install.WithDriverVersion("550"),
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusCompleted, result.Status)

	// Verify install time was stored
	installTimeRaw, ok := ctx.GetState(StatePackageInstallTime)
	assert.True(t, ok)

	installTime, ok := installTimeRaw.(time.Duration)
	assert.True(t, ok)
	assert.Greater(t, installTime.Nanoseconds(), int64(0))
}
