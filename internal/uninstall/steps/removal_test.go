package steps

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tungetti/igor/internal/constants"
	"github.com/tungetti/igor/internal/install"
	"github.com/tungetti/igor/internal/pkg"
	"github.com/tungetti/igor/internal/uninstall"
)

// =============================================================================
// Mock Discovery for Testing
// =============================================================================

// MockDiscovery is a mock implementation of Discovery for testing.
type MockDiscovery struct {
	DiscoverFunc          func(ctx context.Context) (*uninstall.DiscoveredPackages, error)
	DiscoverDriverFunc    func(ctx context.Context) ([]string, string, error)
	DiscoverCUDAFunc      func(ctx context.Context) ([]string, string, error)
	IsNVIDIAInstalledFunc func(ctx context.Context) (bool, error)
	GetDriverVersionFunc  func(ctx context.Context) (string, error)
}

// Discover implements Discovery.
func (m *MockDiscovery) Discover(ctx context.Context) (*uninstall.DiscoveredPackages, error) {
	if m.DiscoverFunc != nil {
		return m.DiscoverFunc(ctx)
	}
	return &uninstall.DiscoveredPackages{
		AllPackages: []string{},
		TotalCount:  0,
	}, nil
}

// DiscoverDriver implements Discovery.
func (m *MockDiscovery) DiscoverDriver(ctx context.Context) ([]string, string, error) {
	if m.DiscoverDriverFunc != nil {
		return m.DiscoverDriverFunc(ctx)
	}
	return nil, "", nil
}

// DiscoverCUDA implements Discovery.
func (m *MockDiscovery) DiscoverCUDA(ctx context.Context) ([]string, string, error) {
	if m.DiscoverCUDAFunc != nil {
		return m.DiscoverCUDAFunc(ctx)
	}
	return nil, "", nil
}

// IsNVIDIAInstalled implements Discovery.
func (m *MockDiscovery) IsNVIDIAInstalled(ctx context.Context) (bool, error) {
	if m.IsNVIDIAInstalledFunc != nil {
		return m.IsNVIDIAInstalledFunc(ctx)
	}
	return false, nil
}

// GetDriverVersion implements Discovery.
func (m *MockDiscovery) GetDriverVersion(ctx context.Context) (string, error) {
	if m.GetDriverVersionFunc != nil {
		return m.GetDriverVersionFunc(ctx)
	}
	return "", nil
}

// Ensure MockDiscovery implements Discovery.
var _ Discovery = (*MockDiscovery)(nil)

// =============================================================================
// Mock Package Manager for Testing
// =============================================================================

// RemovalMockManager implements pkg.Manager for testing package removal.
type RemovalMockManager struct {
	name   string
	family constants.DistroFamily

	// Error injection
	removeErr error

	// Tracking calls
	removeCalled   bool
	removePackages []string
	removeCount    int

	// Options tracking
	lastRemoveOpts pkg.RemoveOptions

	// Callback for custom behavior
	removeCallback func(ctx context.Context, opts pkg.RemoveOptions, packages ...string) error

	// Installed packages for ListInstalled
	installedPackages []pkg.Package
}

// NewRemovalMockManager creates a new mock package manager for removal testing.
func NewRemovalMockManager() *RemovalMockManager {
	return &RemovalMockManager{
		name:           "apt",
		family:         constants.FamilyDebian,
		removePackages: make([]string, 0),
	}
}

// SetRemoveError sets an error to return from Remove.
func (m *RemovalMockManager) SetRemoveError(err error) {
	m.removeErr = err
}

// SetRemoveCallback sets a callback for Remove operations.
func (m *RemovalMockManager) SetRemoveCallback(fn func(ctx context.Context, opts pkg.RemoveOptions, packages ...string) error) {
	m.removeCallback = fn
}

// SetInstalledPackages sets the packages to return from ListInstalled.
func (m *RemovalMockManager) SetInstalledPackages(packages []pkg.Package) {
	m.installedPackages = packages
}

// Reset clears tracking data.
func (m *RemovalMockManager) Reset() {
	m.removeCalled = false
	m.removePackages = make([]string, 0)
	m.removeCount = 0
}

// Remove implements pkg.Manager.
func (m *RemovalMockManager) Remove(ctx context.Context, opts pkg.RemoveOptions, packages ...string) error {
	m.removeCalled = true
	m.removeCount++
	m.removePackages = append(m.removePackages, packages...)
	m.lastRemoveOpts = opts
	if m.removeCallback != nil {
		return m.removeCallback(ctx, opts, packages...)
	}
	return m.removeErr
}

// Install implements pkg.Manager.
func (m *RemovalMockManager) Install(ctx context.Context, opts pkg.InstallOptions, packages ...string) error {
	return nil
}

// AddRepository implements pkg.Manager.
func (m *RemovalMockManager) AddRepository(ctx context.Context, repo pkg.Repository) error {
	return nil
}

// RemoveRepository implements pkg.Manager.
func (m *RemovalMockManager) RemoveRepository(ctx context.Context, name string) error {
	return nil
}

// Update implements pkg.Manager.
func (m *RemovalMockManager) Update(ctx context.Context, opts pkg.UpdateOptions) error {
	return nil
}

// Upgrade implements pkg.Manager.
func (m *RemovalMockManager) Upgrade(ctx context.Context, opts pkg.InstallOptions, packages ...string) error {
	return nil
}

// IsInstalled implements pkg.Manager.
func (m *RemovalMockManager) IsInstalled(ctx context.Context, pkgName string) (bool, error) {
	return false, nil
}

// Search implements pkg.Manager.
func (m *RemovalMockManager) Search(ctx context.Context, query string, opts pkg.SearchOptions) ([]pkg.Package, error) {
	return nil, nil
}

// Info implements pkg.Manager.
func (m *RemovalMockManager) Info(ctx context.Context, pkgName string) (*pkg.Package, error) {
	return nil, nil
}

// ListInstalled implements pkg.Manager.
func (m *RemovalMockManager) ListInstalled(ctx context.Context) ([]pkg.Package, error) {
	return m.installedPackages, nil
}

// ListUpgradable implements pkg.Manager.
func (m *RemovalMockManager) ListUpgradable(ctx context.Context) ([]pkg.Package, error) {
	return nil, nil
}

// ListRepositories implements pkg.Manager.
func (m *RemovalMockManager) ListRepositories(ctx context.Context) ([]pkg.Repository, error) {
	return nil, nil
}

// EnableRepository implements pkg.Manager.
func (m *RemovalMockManager) EnableRepository(ctx context.Context, name string) error {
	return nil
}

// DisableRepository implements pkg.Manager.
func (m *RemovalMockManager) DisableRepository(ctx context.Context, name string) error {
	return nil
}

// RefreshRepositories implements pkg.Manager.
func (m *RemovalMockManager) RefreshRepositories(ctx context.Context) error {
	return nil
}

// Clean implements pkg.Manager.
func (m *RemovalMockManager) Clean(ctx context.Context) error {
	return nil
}

// AutoRemove implements pkg.Manager.
func (m *RemovalMockManager) AutoRemove(ctx context.Context) error {
	return nil
}

// Verify implements pkg.Manager.
func (m *RemovalMockManager) Verify(ctx context.Context, pkgName string) (bool, error) {
	return false, nil
}

// Name implements pkg.Manager.
func (m *RemovalMockManager) Name() string {
	return m.name
}

// Family implements pkg.Manager.
func (m *RemovalMockManager) Family() constants.DistroFamily {
	return m.family
}

// IsAvailable implements pkg.Manager.
func (m *RemovalMockManager) IsAvailable() bool {
	return true
}

// Ensure RemovalMockManager implements pkg.Manager.
var _ pkg.Manager = (*RemovalMockManager)(nil)

// =============================================================================
// Test Helpers
// =============================================================================

// newTestDiscoveredPackages creates a standard set of discovered packages for testing.
func newTestDiscoveredPackages() *uninstall.DiscoveredPackages {
	return &uninstall.DiscoveredPackages{
		DriverPackages:       []string{"nvidia-driver-550", "libnvidia-gl-550"},
		DriverVersion:        "550",
		CUDAPackages:         []string{"cuda-toolkit-12-4", "cuda-libraries-12-4"},
		CUDAVersion:          "12.4",
		LibraryPackages:      []string{"libcudnn8"},
		UtilityPackages:      []string{"nvidia-settings"},
		KernelModulePackages: []string{"nvidia-dkms-550"},
		ConfigPackages:       []string{},
		AllPackages: []string{
			"nvidia-driver-550",
			"libnvidia-gl-550",
			"cuda-toolkit-12-4",
			"cuda-libraries-12-4",
			"libcudnn8",
			"nvidia-settings",
			"nvidia-dkms-550",
		},
		TotalCount:    7,
		DiscoveryTime: time.Now(),
	}
}

// newMockDiscoveryWithPackages creates a mock discovery that returns specified packages.
func newMockDiscoveryWithPackages(packages []string) *MockDiscovery {
	return &MockDiscovery{
		DiscoverFunc: func(ctx context.Context) (*uninstall.DiscoveredPackages, error) {
			return &uninstall.DiscoveredPackages{
				AllPackages: packages,
				TotalCount:  len(packages),
			}, nil
		},
	}
}

// =============================================================================
// PackageRemovalStep Constructor Tests
// =============================================================================

func TestNewPackageRemovalStep_DefaultOptions(t *testing.T) {
	step := NewPackageRemovalStep()

	assert.Equal(t, "removal", step.Name())
	assert.Equal(t, "Remove NVIDIA packages", step.Description())
	assert.False(t, step.CanRollback())
	assert.Empty(t, step.packagesToRemove)
	assert.False(t, step.removeAll)
	assert.False(t, step.purge)
	assert.True(t, step.autoRemove) // Default is true
	assert.Equal(t, 0, step.batchSize)
	assert.Nil(t, step.discovery)
}

func TestNewPackageRemovalStep_WithAllOptions(t *testing.T) {
	mockDiscovery := &MockDiscovery{}
	step := NewPackageRemovalStep(
		WithPackagesToRemove([]string{"pkg1", "pkg2"}),
		WithRemoveAll(true),
		WithPurge(true),
		WithAutoRemove(false),
		WithRemovalBatchSize(5),
		WithRemovalDiscovery(mockDiscovery),
	)

	assert.Equal(t, []string{"pkg1", "pkg2"}, step.packagesToRemove)
	assert.True(t, step.removeAll)
	assert.True(t, step.purge)
	assert.False(t, step.autoRemove)
	assert.Equal(t, 5, step.batchSize)
	assert.NotNil(t, step.discovery)
}

func TestNewPackageRemovalStep_WithPackagesToRemove_AppendsMultipleCalls(t *testing.T) {
	step := NewPackageRemovalStep(
		WithPackagesToRemove([]string{"pkg1"}),
		WithPackagesToRemove([]string{"pkg2", "pkg3"}),
	)

	assert.Equal(t, []string{"pkg1", "pkg2", "pkg3"}, step.packagesToRemove)
}

func TestPackageRemovalStep_Name(t *testing.T) {
	step := NewPackageRemovalStep()
	assert.Equal(t, "removal", step.Name())
}

func TestPackageRemovalStep_Description(t *testing.T) {
	step := NewPackageRemovalStep()
	assert.Equal(t, "Remove NVIDIA packages", step.Description())
}

func TestPackageRemovalStep_CanRollback(t *testing.T) {
	step := NewPackageRemovalStep()
	assert.False(t, step.CanRollback())
}

// =============================================================================
// PackageRemovalStep Execute Tests - Successful Removal
// =============================================================================

func TestPackageRemovalStep_Execute_Success_WithSpecificPackages(t *testing.T) {
	mockPM := NewRemovalMockManager()
	packages := []string{"nvidia-driver-550", "nvidia-settings"}

	step := NewPackageRemovalStep(
		WithPackagesToRemove(packages),
	)

	ctx := install.NewContext(
		install.WithPackageManager(mockPM),
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusCompleted, result.Status)
	assert.Contains(t, result.Message, "packages removed successfully")
	assert.True(t, mockPM.removeCalled)
	assert.ElementsMatch(t, packages, mockPM.removePackages)

	// Check state was set
	assert.True(t, ctx.GetStateBool(StatePackagesRemoved))

	// Check removed packages were stored
	removedRaw, ok := ctx.GetState(StateRemovedPackages)
	assert.True(t, ok)
	removed, ok := removedRaw.([]string)
	assert.True(t, ok)
	assert.ElementsMatch(t, packages, removed)

	// Check failed packages is empty
	failedRaw, ok := ctx.GetState(StateFailedPackages)
	assert.True(t, ok)
	failed, ok := failedRaw.([]string)
	assert.True(t, ok)
	assert.Empty(t, failed)
}

func TestPackageRemovalStep_Execute_Success_WithRemoveAll(t *testing.T) {
	mockPM := NewRemovalMockManager()
	discoveredPackages := newTestDiscoveredPackages()
	mockDiscovery := &MockDiscovery{
		DiscoverFunc: func(ctx context.Context) (*uninstall.DiscoveredPackages, error) {
			return discoveredPackages, nil
		},
	}

	step := NewPackageRemovalStep(
		WithRemoveAll(true),
		WithRemovalDiscovery(mockDiscovery),
	)

	ctx := install.NewContext(
		install.WithPackageManager(mockPM),
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusCompleted, result.Status)
	assert.True(t, mockPM.removeCalled)
	assert.ElementsMatch(t, discoveredPackages.AllPackages, mockPM.removePackages)
}

func TestPackageRemovalStep_Execute_Success_WithPurge(t *testing.T) {
	mockPM := NewRemovalMockManager()

	step := NewPackageRemovalStep(
		WithPackagesToRemove([]string{"nvidia-driver-550"}),
		WithPurge(true),
	)

	ctx := install.NewContext(
		install.WithPackageManager(mockPM),
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusCompleted, result.Status)
	assert.True(t, mockPM.lastRemoveOpts.Purge)
	assert.True(t, ctx.GetStateBool(StateRemovalPurged))
}

func TestPackageRemovalStep_Execute_Success_WithAutoRemove(t *testing.T) {
	mockPM := NewRemovalMockManager()

	step := NewPackageRemovalStep(
		WithPackagesToRemove([]string{"nvidia-driver-550"}),
		WithAutoRemove(true),
	)

	ctx := install.NewContext(
		install.WithPackageManager(mockPM),
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusCompleted, result.Status)
	assert.True(t, mockPM.lastRemoveOpts.AutoRemove)
}

// =============================================================================
// PackageRemovalStep Execute Tests - Dry Run
// =============================================================================

func TestPackageRemovalStep_Execute_DryRun(t *testing.T) {
	mockPM := NewRemovalMockManager()
	packages := []string{"nvidia-driver-550", "nvidia-settings"}

	step := NewPackageRemovalStep(
		WithPackagesToRemove(packages),
	)

	ctx := install.NewContext(
		install.WithPackageManager(mockPM),
		install.WithDryRun(true),
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusCompleted, result.Status)
	assert.Contains(t, result.Message, "dry run")
	// Should NOT have actually removed anything
	assert.False(t, mockPM.removeCalled)

	// State should not be set in dry run
	assert.False(t, ctx.GetStateBool(StatePackagesRemoved))
}

// =============================================================================
// PackageRemovalStep Execute Tests - Empty Package List
// =============================================================================

func TestPackageRemovalStep_Execute_EmptyPackageList_Skips(t *testing.T) {
	mockPM := NewRemovalMockManager()
	mockDiscovery := newMockDiscoveryWithPackages([]string{}) // Empty

	step := NewPackageRemovalStep(
		WithRemoveAll(true),
		WithRemovalDiscovery(mockDiscovery),
	)

	ctx := install.NewContext(
		install.WithPackageManager(mockPM),
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusSkipped, result.Status)
	assert.Contains(t, result.Message, "no packages to remove")
	assert.False(t, mockPM.removeCalled)
}

// =============================================================================
// PackageRemovalStep Execute Tests - Cancellation
// =============================================================================

func TestPackageRemovalStep_Execute_Cancelled_BeforeStart(t *testing.T) {
	mockPM := NewRemovalMockManager()

	step := NewPackageRemovalStep(
		WithPackagesToRemove([]string{"nvidia-driver-550"}),
	)

	ctx := install.NewContext(
		install.WithPackageManager(mockPM),
	)
	ctx.Cancel() // Cancel immediately

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusFailed, result.Status)
	assert.Contains(t, result.Message, "cancelled")
	assert.True(t, errors.Is(result.Error, context.Canceled))
	assert.False(t, mockPM.removeCalled)
}

func TestPackageRemovalStep_Execute_Cancelled_DuringBatch(t *testing.T) {
	mockPM := NewRemovalMockManager()

	ctx := install.NewContext(
		install.WithPackageManager(mockPM),
	)

	// Cancel on second batch - when removeCount reaches 2, return Canceled
	mockPM.SetRemoveCallback(func(c context.Context, opts pkg.RemoveOptions, packages ...string) error {
		if mockPM.removeCount >= 2 {
			ctx.Cancel()
			return context.Canceled
		}
		return nil
	})

	step := NewPackageRemovalStep(
		WithPackagesToRemove([]string{"pkg1", "pkg2", "pkg3"}),
		WithRemovalBatchSize(1),
	)

	_ = step.Execute(ctx)

	// Should have called remove
	assert.True(t, mockPM.removeCalled)

	// Check that some packages failed
	failedRaw, ok := ctx.GetState(StateFailedPackages)
	assert.True(t, ok)
	failed, ok := failedRaw.([]string)
	assert.True(t, ok)
	// pkg2 fails (Canceled returned), pkg3 is added to failed due to context cancellation check
	assert.GreaterOrEqual(t, len(failed), 1)

	// Check that some packages succeeded
	removedRaw, ok := ctx.GetState(StateRemovedPackages)
	assert.True(t, ok)
	removed, ok := removedRaw.([]string)
	assert.True(t, ok)
	// pkg1 should have succeeded
	assert.Contains(t, removed, "pkg1")
}

// =============================================================================
// PackageRemovalStep Execute Tests - Error Handling
// =============================================================================

func TestPackageRemovalStep_Execute_NilContext(t *testing.T) {
	step := NewPackageRemovalStep(
		WithPackagesToRemove([]string{"nvidia-driver-550"}),
	)

	// This should panic or fail gracefully
	// Using a nil Context is an edge case; the step validates it
	defer func() {
		if r := recover(); r != nil {
			// Panic is acceptable for nil context
			t.Log("Recovered from panic on nil context")
		}
	}()

	result := step.Execute(nil)

	assert.Equal(t, install.StepStatusFailed, result.Status)
}

func TestPackageRemovalStep_Execute_NilPackageManager(t *testing.T) {
	step := NewPackageRemovalStep(
		WithPackagesToRemove([]string{"nvidia-driver-550"}),
	)

	ctx := install.NewContext()

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusFailed, result.Status)
	assert.Contains(t, result.Message, "validation failed")
	assert.Error(t, result.Error)
	assert.Contains(t, result.Error.Error(), "package manager")
}

func TestPackageRemovalStep_Execute_CompleteFailed(t *testing.T) {
	mockPM := NewRemovalMockManager()
	mockPM.SetRemoveError(errors.New("removal failed: package in use"))

	step := NewPackageRemovalStep(
		WithPackagesToRemove([]string{"nvidia-driver-550"}),
	)

	ctx := install.NewContext(
		install.WithPackageManager(mockPM),
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusFailed, result.Status)
	assert.Contains(t, result.Message, "failed to remove any packages")
	assert.Error(t, result.Error)
	assert.False(t, ctx.GetStateBool(StatePackagesRemoved))

	// Check failed packages were stored
	failedRaw, ok := ctx.GetState(StateFailedPackages)
	assert.True(t, ok)
	failed, ok := failedRaw.([]string)
	assert.True(t, ok)
	assert.Contains(t, failed, "nvidia-driver-550")
}

func TestPackageRemovalStep_Execute_PartialFailure(t *testing.T) {
	mockPM := NewRemovalMockManager()

	// Fail on second batch
	mockPM.SetRemoveCallback(func(ctx context.Context, opts pkg.RemoveOptions, packages ...string) error {
		if mockPM.removeCount == 2 {
			return errors.New("failed to remove pkg2")
		}
		return nil
	})

	step := NewPackageRemovalStep(
		WithPackagesToRemove([]string{"pkg1", "pkg2", "pkg3"}),
		WithRemovalBatchSize(1),
	)

	ctx := install.NewContext(
		install.WithPackageManager(mockPM),
	)

	result := step.Execute(ctx)

	// Should complete with partial success
	assert.Equal(t, install.StepStatusCompleted, result.Status)
	assert.Contains(t, result.Message, "partially removed")
	assert.True(t, ctx.GetStateBool(StatePackagesRemoved))

	// Check removed packages
	removedRaw, ok := ctx.GetState(StateRemovedPackages)
	assert.True(t, ok)
	removed, ok := removedRaw.([]string)
	assert.True(t, ok)
	assert.Equal(t, 2, len(removed)) // pkg1 and pkg3

	// Check failed packages
	failedRaw, ok := ctx.GetState(StateFailedPackages)
	assert.True(t, ok)
	failed, ok := failedRaw.([]string)
	assert.True(t, ok)
	assert.Equal(t, 1, len(failed)) // pkg2
}

func TestPackageRemovalStep_Execute_DiscoveryFails(t *testing.T) {
	mockPM := NewRemovalMockManager()
	mockDiscovery := &MockDiscovery{
		DiscoverFunc: func(ctx context.Context) (*uninstall.DiscoveredPackages, error) {
			return nil, errors.New("discovery failed")
		},
	}

	step := NewPackageRemovalStep(
		WithRemoveAll(true),
		WithRemovalDiscovery(mockDiscovery),
	)

	ctx := install.NewContext(
		install.WithPackageManager(mockPM),
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusFailed, result.Status)
	assert.Contains(t, result.Message, "failed to determine packages")
	assert.Error(t, result.Error)
	assert.False(t, mockPM.removeCalled)
}

// =============================================================================
// PackageRemovalStep Execute Tests - Batch Removal
// =============================================================================

func TestPackageRemovalStep_Execute_BatchRemoval(t *testing.T) {
	mockPM := NewRemovalMockManager()
	packages := []string{"pkg1", "pkg2", "pkg3", "pkg4", "pkg5"}

	step := NewPackageRemovalStep(
		WithPackagesToRemove(packages),
		WithRemovalBatchSize(2),
	)

	ctx := install.NewContext(
		install.WithPackageManager(mockPM),
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusCompleted, result.Status)
	// With batch size 2 and 5 packages, should have 3 remove calls
	assert.Equal(t, 3, mockPM.removeCount)
	assert.ElementsMatch(t, packages, mockPM.removePackages)
}

func TestPackageRemovalStep_Execute_BatchRemoval_ZeroBatchSize(t *testing.T) {
	mockPM := NewRemovalMockManager()
	packages := []string{"pkg1", "pkg2", "pkg3"}

	step := NewPackageRemovalStep(
		WithPackagesToRemove(packages),
		WithRemovalBatchSize(0), // Should remove all at once
	)

	ctx := install.NewContext(
		install.WithPackageManager(mockPM),
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusCompleted, result.Status)
	assert.Equal(t, 1, mockPM.removeCount) // Single call
}

// =============================================================================
// PackageRemovalStep Validate Tests
// =============================================================================

func TestPackageRemovalStep_Validate_Success_WithPackages(t *testing.T) {
	mockPM := NewRemovalMockManager()

	step := NewPackageRemovalStep(
		WithPackagesToRemove([]string{"nvidia-driver-550"}),
	)

	ctx := install.NewContext(
		install.WithPackageManager(mockPM),
	)

	err := step.Validate(ctx)

	assert.NoError(t, err)
}

func TestPackageRemovalStep_Validate_Success_WithRemoveAll(t *testing.T) {
	mockPM := NewRemovalMockManager()
	mockDiscovery := &MockDiscovery{}

	step := NewPackageRemovalStep(
		WithRemoveAll(true),
		WithRemovalDiscovery(mockDiscovery),
	)

	ctx := install.NewContext(
		install.WithPackageManager(mockPM),
	)

	err := step.Validate(ctx)

	assert.NoError(t, err)
}

func TestPackageRemovalStep_Validate_NilContext(t *testing.T) {
	step := NewPackageRemovalStep(
		WithPackagesToRemove([]string{"nvidia-driver-550"}),
	)

	err := step.Validate(nil)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context is nil")
}

func TestPackageRemovalStep_Validate_NoPackageManager(t *testing.T) {
	step := NewPackageRemovalStep(
		WithPackagesToRemove([]string{"nvidia-driver-550"}),
	)

	ctx := install.NewContext()

	err := step.Validate(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "package manager is required")
}

func TestPackageRemovalStep_Validate_NoPackagesSpecified(t *testing.T) {
	mockPM := NewRemovalMockManager()

	step := NewPackageRemovalStep(
	// No packages and removeAll is false
	)

	ctx := install.NewContext(
		install.WithPackageManager(mockPM),
	)

	err := step.Validate(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "removeAll must be true or packagesToRemove must be specified")
}

func TestPackageRemovalStep_Validate_RemoveAllWithoutDiscovery(t *testing.T) {
	mockPM := NewRemovalMockManager()

	step := NewPackageRemovalStep(
		WithRemoveAll(true),
		// No discovery set
	)

	ctx := install.NewContext(
		install.WithPackageManager(mockPM),
	)

	err := step.Validate(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "discovery is required when removeAll is true")
}

// =============================================================================
// PackageRemovalStep Rollback Tests
// =============================================================================

func TestPackageRemovalStep_Rollback_IsNoOp(t *testing.T) {
	mockPM := NewRemovalMockManager()

	step := NewPackageRemovalStep(
		WithPackagesToRemove([]string{"nvidia-driver-550"}),
	)

	ctx := install.NewContext(
		install.WithPackageManager(mockPM),
	)

	// Execute first
	result := step.Execute(ctx)
	require.Equal(t, install.StepStatusCompleted, result.Status)

	mockPM.Reset()

	// Rollback should be a no-op
	err := step.Rollback(ctx)

	assert.NoError(t, err)
	assert.False(t, mockPM.removeCalled)
}

func TestPackageRemovalStep_CanRollback_ReturnsFalse(t *testing.T) {
	step := NewPackageRemovalStep()
	assert.False(t, step.CanRollback())
}

// =============================================================================
// PackageRemovalStep State Tests
// =============================================================================

func TestPackageRemovalStep_StateKeys(t *testing.T) {
	assert.Equal(t, "packages_removed", StatePackagesRemoved)
	assert.Equal(t, "removed_packages", StateRemovedPackages)
	assert.Equal(t, "failed_packages", StateFailedPackages)
	assert.Equal(t, "removal_purged", StateRemovalPurged)
}

func TestPackageRemovalStep_Execute_StoresCorrectState(t *testing.T) {
	mockPM := NewRemovalMockManager()
	packages := []string{"nvidia-driver-550", "nvidia-settings"}

	step := NewPackageRemovalStep(
		WithPackagesToRemove(packages),
		WithPurge(true),
	)

	ctx := install.NewContext(
		install.WithPackageManager(mockPM),
	)

	result := step.Execute(ctx)
	require.Equal(t, install.StepStatusCompleted, result.Status)

	// Verify all state keys are set correctly
	assert.True(t, ctx.GetStateBool(StatePackagesRemoved))
	assert.True(t, ctx.GetStateBool(StateRemovalPurged))

	removedRaw, ok := ctx.GetState(StateRemovedPackages)
	require.True(t, ok)
	removed, ok := removedRaw.([]string)
	require.True(t, ok)
	assert.ElementsMatch(t, packages, removed)

	failedRaw, ok := ctx.GetState(StateFailedPackages)
	require.True(t, ok)
	failed, ok := failedRaw.([]string)
	require.True(t, ok)
	assert.Empty(t, failed)
}

// =============================================================================
// PackageRemovalStep Options Tests
// =============================================================================

func TestPackageRemovalStep_Options(t *testing.T) {
	t.Run("WithPackagesToRemove", func(t *testing.T) {
		step := NewPackageRemovalStep(
			WithPackagesToRemove([]string{"pkg1", "pkg2"}),
		)
		assert.Equal(t, []string{"pkg1", "pkg2"}, step.packagesToRemove)
	})

	t.Run("WithRemoveAll true", func(t *testing.T) {
		step := NewPackageRemovalStep(WithRemoveAll(true))
		assert.True(t, step.removeAll)
	})

	t.Run("WithRemoveAll false", func(t *testing.T) {
		step := NewPackageRemovalStep(WithRemoveAll(false))
		assert.False(t, step.removeAll)
	})

	t.Run("WithPurge true", func(t *testing.T) {
		step := NewPackageRemovalStep(WithPurge(true))
		assert.True(t, step.purge)
	})

	t.Run("WithPurge false", func(t *testing.T) {
		step := NewPackageRemovalStep(WithPurge(false))
		assert.False(t, step.purge)
	})

	t.Run("WithAutoRemove true", func(t *testing.T) {
		step := NewPackageRemovalStep(WithAutoRemove(true))
		assert.True(t, step.autoRemove)
	})

	t.Run("WithAutoRemove false", func(t *testing.T) {
		step := NewPackageRemovalStep(WithAutoRemove(false))
		assert.False(t, step.autoRemove)
	})

	t.Run("WithRemovalBatchSize", func(t *testing.T) {
		step := NewPackageRemovalStep(WithRemovalBatchSize(10))
		assert.Equal(t, 10, step.batchSize)
	})

	t.Run("WithRemovalDiscovery", func(t *testing.T) {
		discovery := &MockDiscovery{}
		step := NewPackageRemovalStep(WithRemovalDiscovery(discovery))
		assert.NotNil(t, step.discovery)
	})
}

// =============================================================================
// PackageRemovalStep Interface Compliance Tests
// =============================================================================

func TestPackageRemovalStep_InterfaceCompliance(t *testing.T) {
	var _ install.Step = (*PackageRemovalStep)(nil)
}

// =============================================================================
// PackageRemovalStep Duration Tests
// =============================================================================

func TestPackageRemovalStep_Execute_Duration(t *testing.T) {
	mockPM := NewRemovalMockManager()

	step := NewPackageRemovalStep(
		WithPackagesToRemove([]string{"nvidia-driver-550"}),
	)

	ctx := install.NewContext(
		install.WithPackageManager(mockPM),
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusCompleted, result.Status)
	assert.Greater(t, result.Duration.Nanoseconds(), int64(0))
}

// =============================================================================
// PackageRemovalStep Combined Scenarios Tests
// =============================================================================

func TestPackageRemovalStep_Execute_CombinedRemoveAllAndSpecificPackages(t *testing.T) {
	mockPM := NewRemovalMockManager()
	discoveredPackages := []string{"discovered-pkg1", "discovered-pkg2"}
	specificPackages := []string{"specific-pkg1"}

	mockDiscovery := newMockDiscoveryWithPackages(discoveredPackages)

	step := NewPackageRemovalStep(
		WithRemoveAll(true),
		WithRemovalDiscovery(mockDiscovery),
		WithPackagesToRemove(specificPackages),
	)

	ctx := install.NewContext(
		install.WithPackageManager(mockPM),
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusCompleted, result.Status)
	// Should include both discovered and specific packages
	assert.Contains(t, mockPM.removePackages, "discovered-pkg1")
	assert.Contains(t, mockPM.removePackages, "discovered-pkg2")
	assert.Contains(t, mockPM.removePackages, "specific-pkg1")
}

func TestPackageRemovalStep_Execute_DeduplicatesPackages(t *testing.T) {
	mockPM := NewRemovalMockManager()
	discoveredPackages := []string{"pkg1", "pkg2"}
	specificPackages := []string{"pkg1", "pkg3"} // pkg1 is duplicate

	mockDiscovery := newMockDiscoveryWithPackages(discoveredPackages)

	step := NewPackageRemovalStep(
		WithRemoveAll(true),
		WithRemovalDiscovery(mockDiscovery),
		WithPackagesToRemove(specificPackages),
	)

	ctx := install.NewContext(
		install.WithPackageManager(mockPM),
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusCompleted, result.Status)

	// Count occurrences of pkg1
	count := 0
	for _, p := range mockPM.removePackages {
		if p == "pkg1" {
			count++
		}
	}
	assert.Equal(t, 1, count, "pkg1 should appear only once")
}

func TestPackageRemovalStep_Execute_RemoveOptions(t *testing.T) {
	mockPM := NewRemovalMockManager()

	step := NewPackageRemovalStep(
		WithPackagesToRemove([]string{"nvidia-driver-550"}),
		WithPurge(true),
		WithAutoRemove(true),
	)

	ctx := install.NewContext(
		install.WithPackageManager(mockPM),
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusCompleted, result.Status)
	assert.True(t, mockPM.lastRemoveOpts.Purge)
	assert.True(t, mockPM.lastRemoveOpts.AutoRemove)
	assert.True(t, mockPM.lastRemoveOpts.NoConfirm)
}

// =============================================================================
// PackageRemovalStep Edge Case Tests
// =============================================================================

func TestPackageRemovalStep_Execute_EmptySpecificPackages(t *testing.T) {
	mockPM := NewRemovalMockManager()
	mockDiscovery := newMockDiscoveryWithPackages([]string{"pkg1"})

	step := NewPackageRemovalStep(
		WithRemoveAll(true),
		WithRemovalDiscovery(mockDiscovery),
		WithPackagesToRemove([]string{}), // Empty array
	)

	ctx := install.NewContext(
		install.WithPackageManager(mockPM),
	)

	result := step.Execute(ctx)

	// Should still work with discovered packages
	assert.Equal(t, install.StepStatusCompleted, result.Status)
	assert.Contains(t, mockPM.removePackages, "pkg1")
}

func TestPackageRemovalStep_Execute_DiscoveryReturnsNil(t *testing.T) {
	mockPM := NewRemovalMockManager()
	mockDiscovery := &MockDiscovery{
		DiscoverFunc: func(ctx context.Context) (*uninstall.DiscoveredPackages, error) {
			return nil, nil // nil result, no error
		},
	}

	step := NewPackageRemovalStep(
		WithRemoveAll(true),
		WithRemovalDiscovery(mockDiscovery),
	)

	ctx := install.NewContext(
		install.WithPackageManager(mockPM),
	)

	result := step.Execute(ctx)

	// Should skip since no packages found
	assert.Equal(t, install.StepStatusSkipped, result.Status)
}

func TestPackageRemovalStep_Execute_NegativeBatchSize(t *testing.T) {
	mockPM := NewRemovalMockManager()
	packages := []string{"pkg1", "pkg2", "pkg3"}

	step := NewPackageRemovalStep(
		WithPackagesToRemove(packages),
		WithRemovalBatchSize(-1), // Negative should behave like 0
	)

	ctx := install.NewContext(
		install.WithPackageManager(mockPM),
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusCompleted, result.Status)
	assert.Equal(t, 1, mockPM.removeCount) // Single call
}

func TestPackageRemovalStep_Execute_LargeBatchSize(t *testing.T) {
	mockPM := NewRemovalMockManager()
	packages := []string{"pkg1", "pkg2", "pkg3"}

	step := NewPackageRemovalStep(
		WithPackagesToRemove(packages),
		WithRemovalBatchSize(100), // Larger than package count
	)

	ctx := install.NewContext(
		install.WithPackageManager(mockPM),
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusCompleted, result.Status)
	assert.Equal(t, 1, mockPM.removeCount) // Single call since batch > packages
}

// =============================================================================
// Benchmarks
// =============================================================================

func BenchmarkPackageRemovalStep_Execute(b *testing.B) {
	mockPM := NewRemovalMockManager()
	packages := []string{"nvidia-driver-550", "nvidia-settings", "nvidia-dkms-550"}

	step := NewPackageRemovalStep(
		WithPackagesToRemove(packages),
	)

	ctx := install.NewContext(
		install.WithPackageManager(mockPM),
	)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Clear state between iterations
		ctx.DeleteState(StatePackagesRemoved)
		ctx.DeleteState(StateRemovedPackages)
		ctx.DeleteState(StateFailedPackages)
		ctx.DeleteState(StateRemovalPurged)

		mockPM.Reset()
		step.Execute(ctx)
	}
}

func BenchmarkPackageRemovalStep_Validate(b *testing.B) {
	mockPM := NewRemovalMockManager()

	step := NewPackageRemovalStep(
		WithPackagesToRemove([]string{"nvidia-driver-550"}),
	)

	ctx := install.NewContext(
		install.WithPackageManager(mockPM),
	)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = step.Validate(ctx)
	}
}

func BenchmarkPackageRemovalStep_DeterminePackages(b *testing.B) {
	mockPM := NewRemovalMockManager()
	mockDiscovery := newMockDiscoveryWithPackages([]string{
		"nvidia-driver-550",
		"libnvidia-gl-550",
		"nvidia-settings",
		"nvidia-dkms-550",
		"cuda-toolkit-12-4",
	})

	step := NewPackageRemovalStep(
		WithRemoveAll(true),
		WithRemovalDiscovery(mockDiscovery),
		WithPackagesToRemove([]string{"extra-pkg"}),
	)

	ctx := install.NewContext(
		install.WithPackageManager(mockPM),
	)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = step.determinePackages(ctx)
	}
}
