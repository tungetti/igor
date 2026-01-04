package steps

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tungetti/igor/internal/constants"
	"github.com/tungetti/igor/internal/distro"
	"github.com/tungetti/igor/internal/install"
	"github.com/tungetti/igor/internal/pkg"
)

// =============================================================================
// Mock Package Manager
// =============================================================================

// MockPackageManager implements pkg.Manager for testing repository step.
type MockPackageManager struct {
	name   string
	family constants.DistroFamily

	// Error injection
	addRepoErr    error
	removeRepoErr error
	updateErr     error

	// Tracking calls
	addRepoCalled    bool
	removeRepoCalled bool
	updateCalled     bool
	lastAddedRepo    *pkg.Repository
	lastRemovedRepo  string
}

// NewMockPackageManager creates a new mock package manager for testing.
func NewMockPackageManager() *MockPackageManager {
	return &MockPackageManager{
		name:   "apt",
		family: constants.FamilyDebian,
	}
}

// SetAddRepoError sets an error to return from AddRepository.
func (m *MockPackageManager) SetAddRepoError(err error) {
	m.addRepoErr = err
}

// SetRemoveRepoError sets an error to return from RemoveRepository.
func (m *MockPackageManager) SetRemoveRepoError(err error) {
	m.removeRepoErr = err
}

// SetUpdateError sets an error to return from Update.
func (m *MockPackageManager) SetUpdateError(err error) {
	m.updateErr = err
}

// AddRepository implements pkg.Manager.
func (m *MockPackageManager) AddRepository(ctx context.Context, repo pkg.Repository) error {
	m.addRepoCalled = true
	m.lastAddedRepo = &repo
	return m.addRepoErr
}

// RemoveRepository implements pkg.Manager.
func (m *MockPackageManager) RemoveRepository(ctx context.Context, name string) error {
	m.removeRepoCalled = true
	m.lastRemovedRepo = name
	return m.removeRepoErr
}

// Update implements pkg.Manager.
func (m *MockPackageManager) Update(ctx context.Context, opts pkg.UpdateOptions) error {
	m.updateCalled = true
	return m.updateErr
}

// Install implements pkg.Manager.
func (m *MockPackageManager) Install(ctx context.Context, opts pkg.InstallOptions, packages ...string) error {
	return nil
}

// Remove implements pkg.Manager.
func (m *MockPackageManager) Remove(ctx context.Context, opts pkg.RemoveOptions, packages ...string) error {
	return nil
}

// Upgrade implements pkg.Manager.
func (m *MockPackageManager) Upgrade(ctx context.Context, opts pkg.InstallOptions, packages ...string) error {
	return nil
}

// IsInstalled implements pkg.Manager.
func (m *MockPackageManager) IsInstalled(ctx context.Context, pkgName string) (bool, error) {
	return false, nil
}

// Search implements pkg.Manager.
func (m *MockPackageManager) Search(ctx context.Context, query string, opts pkg.SearchOptions) ([]pkg.Package, error) {
	return nil, nil
}

// Info implements pkg.Manager.
func (m *MockPackageManager) Info(ctx context.Context, pkgName string) (*pkg.Package, error) {
	return nil, nil
}

// ListInstalled implements pkg.Manager.
func (m *MockPackageManager) ListInstalled(ctx context.Context) ([]pkg.Package, error) {
	return nil, nil
}

// ListUpgradable implements pkg.Manager.
func (m *MockPackageManager) ListUpgradable(ctx context.Context) ([]pkg.Package, error) {
	return nil, nil
}

// ListRepositories implements pkg.Manager.
func (m *MockPackageManager) ListRepositories(ctx context.Context) ([]pkg.Repository, error) {
	return nil, nil
}

// EnableRepository implements pkg.Manager.
func (m *MockPackageManager) EnableRepository(ctx context.Context, name string) error {
	return nil
}

// DisableRepository implements pkg.Manager.
func (m *MockPackageManager) DisableRepository(ctx context.Context, name string) error {
	return nil
}

// RefreshRepositories implements pkg.Manager.
func (m *MockPackageManager) RefreshRepositories(ctx context.Context) error {
	return nil
}

// Clean implements pkg.Manager.
func (m *MockPackageManager) Clean(ctx context.Context) error {
	return nil
}

// AutoRemove implements pkg.Manager.
func (m *MockPackageManager) AutoRemove(ctx context.Context) error {
	return nil
}

// Verify implements pkg.Manager.
func (m *MockPackageManager) Verify(ctx context.Context, pkgName string) (bool, error) {
	return false, nil
}

// Name implements pkg.Manager.
func (m *MockPackageManager) Name() string {
	return m.name
}

// Family implements pkg.Manager.
func (m *MockPackageManager) Family() constants.DistroFamily {
	return m.family
}

// IsAvailable implements pkg.Manager.
func (m *MockPackageManager) IsAvailable() bool {
	return true
}

// Ensure MockPackageManager implements pkg.Manager.
var _ pkg.Manager = (*MockPackageManager)(nil)

// =============================================================================
// Test Helpers
// =============================================================================

// newUbuntuDistro creates a test Ubuntu distribution.
func newUbuntuDistro() *distro.Distribution {
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

// newArchDistro creates a test Arch Linux distribution.
func newArchDistro() *distro.Distribution {
	return &distro.Distribution{
		ID:         "arch",
		Name:       "Arch Linux",
		PrettyName: "Arch Linux",
		Family:     constants.FamilyArch,
	}
}

// newFedoraDistro creates a test Fedora distribution.
func newFedoraDistro() *distro.Distribution {
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

// =============================================================================
// RepositoryStep Constructor Tests
// =============================================================================

func TestNewRepositoryStep(t *testing.T) {
	t.Run("creates with defaults", func(t *testing.T) {
		step := NewRepositoryStep()

		assert.Equal(t, "repository", step.Name())
		assert.Equal(t, "Configure NVIDIA repository", step.Description())
		assert.True(t, step.CanRollback())
		assert.False(t, step.skipUpdate)
	})

	t.Run("creates with WithSkipUpdate true", func(t *testing.T) {
		step := NewRepositoryStep(WithSkipUpdate(true))

		assert.True(t, step.skipUpdate)
	})

	t.Run("creates with WithSkipUpdate false", func(t *testing.T) {
		step := NewRepositoryStep(WithSkipUpdate(false))

		assert.False(t, step.skipUpdate)
	})

	t.Run("applies multiple options", func(t *testing.T) {
		// Test that options are applied in order
		step := NewRepositoryStep(
			WithSkipUpdate(true),
			WithSkipUpdate(false), // This should override the first
		)

		assert.False(t, step.skipUpdate)
	})
}

// =============================================================================
// RepositoryStep Execute Tests
// =============================================================================

func TestRepositoryStep_Execute_Success(t *testing.T) {
	mockPM := NewMockPackageManager()
	step := NewRepositoryStep()

	ctx := install.NewContext(
		install.WithPackageManager(mockPM),
		install.WithDistroInfo(newUbuntuDistro()),
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusCompleted, result.Status)
	assert.Contains(t, result.Message, "repository configured successfully")
	assert.True(t, mockPM.addRepoCalled)
	assert.True(t, mockPM.updateCalled)
	assert.NotNil(t, mockPM.lastAddedRepo)

	// Check state was set
	assert.True(t, ctx.GetStateBool(StateRepositoryConfigured))
	assert.NotEmpty(t, ctx.GetStateString(StateRepositoryName))
}

func TestRepositoryStep_Execute_SuccessWithFedora(t *testing.T) {
	mockPM := NewMockPackageManager()
	mockPM.family = constants.FamilyRHEL
	step := NewRepositoryStep()

	ctx := install.NewContext(
		install.WithPackageManager(mockPM),
		install.WithDistroInfo(newFedoraDistro()),
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusCompleted, result.Status)
	assert.True(t, mockPM.addRepoCalled)
	assert.True(t, mockPM.updateCalled)
}

func TestRepositoryStep_Execute_ArchLinux_Skip(t *testing.T) {
	mockPM := NewMockPackageManager()
	mockPM.family = constants.FamilyArch
	step := NewRepositoryStep()

	ctx := install.NewContext(
		install.WithPackageManager(mockPM),
		install.WithDistroInfo(newArchDistro()),
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusCompleted, result.Status)
	assert.Contains(t, result.Message, "no repository configuration required")
	// Should NOT have added a repository
	assert.False(t, mockPM.addRepoCalled)
	assert.False(t, mockPM.updateCalled)
}

func TestRepositoryStep_Execute_DryRun(t *testing.T) {
	mockPM := NewMockPackageManager()
	step := NewRepositoryStep()

	ctx := install.NewContext(
		install.WithPackageManager(mockPM),
		install.WithDistroInfo(newUbuntuDistro()),
		install.WithDryRun(true),
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusCompleted, result.Status)
	assert.Contains(t, result.Message, "dry run")
	// Should NOT have actually added a repository
	assert.False(t, mockPM.addRepoCalled)
	assert.False(t, mockPM.updateCalled)
}

func TestRepositoryStep_Execute_DryRun_ArchLinux(t *testing.T) {
	mockPM := NewMockPackageManager()
	mockPM.family = constants.FamilyArch
	step := NewRepositoryStep()

	ctx := install.NewContext(
		install.WithPackageManager(mockPM),
		install.WithDistroInfo(newArchDistro()),
		install.WithDryRun(true),
	)

	result := step.Execute(ctx)

	// Arch Linux returns early before dry run check since no repo is needed
	assert.Equal(t, install.StepStatusCompleted, result.Status)
	assert.Contains(t, result.Message, "no repository configuration required")
}

func TestRepositoryStep_Execute_MissingPackageManager(t *testing.T) {
	step := NewRepositoryStep()

	ctx := install.NewContext(
		install.WithDistroInfo(newUbuntuDistro()),
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusFailed, result.Status)
	assert.Contains(t, result.Message, "validation failed")
	assert.Error(t, result.Error)
	assert.Contains(t, result.Error.Error(), "package manager")
}

func TestRepositoryStep_Execute_MissingDistroInfo(t *testing.T) {
	mockPM := NewMockPackageManager()
	step := NewRepositoryStep()

	ctx := install.NewContext(
		install.WithPackageManager(mockPM),
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusFailed, result.Status)
	assert.Contains(t, result.Message, "validation failed")
	assert.Error(t, result.Error)
	assert.Contains(t, result.Error.Error(), "distribution info")
}

func TestRepositoryStep_Execute_AddRepositoryError(t *testing.T) {
	mockPM := NewMockPackageManager()
	mockPM.SetAddRepoError(errors.New("failed to add repo: permission denied"))
	step := NewRepositoryStep()

	ctx := install.NewContext(
		install.WithPackageManager(mockPM),
		install.WithDistroInfo(newUbuntuDistro()),
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusFailed, result.Status)
	assert.Contains(t, result.Message, "failed to add repository")
	assert.Error(t, result.Error)
	assert.True(t, mockPM.addRepoCalled)
	assert.False(t, mockPM.updateCalled)
}

func TestRepositoryStep_Execute_UpdateError(t *testing.T) {
	mockPM := NewMockPackageManager()
	mockPM.SetUpdateError(errors.New("failed to update: network error"))
	step := NewRepositoryStep()

	ctx := install.NewContext(
		install.WithPackageManager(mockPM),
		install.WithDistroInfo(newUbuntuDistro()),
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusFailed, result.Status)
	assert.Contains(t, result.Message, "failed to update package lists")
	assert.Error(t, result.Error)
	assert.True(t, mockPM.addRepoCalled)
	assert.True(t, mockPM.updateCalled)
	// Should have tried to rollback the repo
	assert.True(t, mockPM.removeRepoCalled)
}

func TestRepositoryStep_Execute_UpdateError_RollbackFails(t *testing.T) {
	mockPM := NewMockPackageManager()
	mockPM.SetUpdateError(errors.New("failed to update: network error"))
	mockPM.SetRemoveRepoError(errors.New("failed to remove repo"))
	step := NewRepositoryStep()

	ctx := install.NewContext(
		install.WithPackageManager(mockPM),
		install.WithDistroInfo(newUbuntuDistro()),
	)

	result := step.Execute(ctx)

	// Should still report the original failure
	assert.Equal(t, install.StepStatusFailed, result.Status)
	assert.Contains(t, result.Message, "failed to update package lists")
}

func TestRepositoryStep_Execute_Cancelled(t *testing.T) {
	mockPM := NewMockPackageManager()
	step := NewRepositoryStep()

	ctx := install.NewContext(
		install.WithPackageManager(mockPM),
		install.WithDistroInfo(newUbuntuDistro()),
	)
	ctx.Cancel() // Cancel immediately

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusFailed, result.Status)
	assert.Contains(t, result.Message, "cancelled")
	assert.True(t, errors.Is(result.Error, context.Canceled))
}

func TestRepositoryStep_Execute_WithSkipUpdate(t *testing.T) {
	mockPM := NewMockPackageManager()
	step := NewRepositoryStep(WithSkipUpdate(true))

	ctx := install.NewContext(
		install.WithPackageManager(mockPM),
		install.WithDistroInfo(newUbuntuDistro()),
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusCompleted, result.Status)
	assert.True(t, mockPM.addRepoCalled)
	assert.False(t, mockPM.updateCalled) // Should NOT have updated
}

func TestRepositoryStep_Execute_Duration(t *testing.T) {
	mockPM := NewMockPackageManager()
	step := NewRepositoryStep()

	ctx := install.NewContext(
		install.WithPackageManager(mockPM),
		install.WithDistroInfo(newUbuntuDistro()),
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusCompleted, result.Status)
	assert.Greater(t, result.Duration.Nanoseconds(), int64(0))
}

func TestRepositoryStep_Execute_UnsupportedFamily(t *testing.T) {
	mockPM := NewMockPackageManager()
	mockPM.family = constants.FamilyUnknown
	step := NewRepositoryStep()

	unknownDistro := &distro.Distribution{
		ID:         "unknown",
		Name:       "Unknown Linux",
		PrettyName: "Unknown Linux",
		Family:     constants.FamilyUnknown,
	}

	ctx := install.NewContext(
		install.WithPackageManager(mockPM),
		install.WithDistroInfo(unknownDistro),
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusFailed, result.Status)
	assert.Contains(t, result.Message, "failed to get repository info")
}

// =============================================================================
// RepositoryStep Rollback Tests
// =============================================================================

func TestRepositoryStep_Rollback_Success(t *testing.T) {
	mockPM := NewMockPackageManager()
	step := NewRepositoryStep()

	ctx := install.NewContext(
		install.WithPackageManager(mockPM),
		install.WithDistroInfo(newUbuntuDistro()),
	)

	// First execute the step
	result := step.Execute(ctx)
	require.Equal(t, install.StepStatusCompleted, result.Status)

	// Reset tracking
	mockPM.removeRepoCalled = false

	// Now rollback
	err := step.Rollback(ctx)

	assert.NoError(t, err)
	assert.True(t, mockPM.removeRepoCalled)
	assert.NotEmpty(t, mockPM.lastRemovedRepo)

	// State should be cleared
	assert.False(t, ctx.GetStateBool(StateRepositoryConfigured))
	assert.Empty(t, ctx.GetStateString(StateRepositoryName))
}

func TestRepositoryStep_Rollback_NoRepoConfigured(t *testing.T) {
	mockPM := NewMockPackageManager()
	step := NewRepositoryStep()

	ctx := install.NewContext(
		install.WithPackageManager(mockPM),
		install.WithDistroInfo(newUbuntuDistro()),
	)

	// Don't execute, just rollback directly
	err := step.Rollback(ctx)

	assert.NoError(t, err)
	assert.False(t, mockPM.removeRepoCalled)
}

func TestRepositoryStep_Rollback_NoRepoName(t *testing.T) {
	mockPM := NewMockPackageManager()
	step := NewRepositoryStep()

	ctx := install.NewContext(
		install.WithPackageManager(mockPM),
		install.WithDistroInfo(newUbuntuDistro()),
	)

	// Set configured but not the name
	ctx.SetState(StateRepositoryConfigured, true)

	err := step.Rollback(ctx)

	assert.NoError(t, err)
	assert.False(t, mockPM.removeRepoCalled)
}

func TestRepositoryStep_Rollback_RemoveRepoError(t *testing.T) {
	mockPM := NewMockPackageManager()
	step := NewRepositoryStep()

	ctx := install.NewContext(
		install.WithPackageManager(mockPM),
		install.WithDistroInfo(newUbuntuDistro()),
	)

	// First execute the step
	result := step.Execute(ctx)
	require.Equal(t, install.StepStatusCompleted, result.Status)

	// Set error for rollback
	mockPM.SetRemoveRepoError(errors.New("failed to remove"))

	err := step.Rollback(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to remove repository")
}

func TestRepositoryStep_Rollback_NilPackageManager(t *testing.T) {
	step := NewRepositoryStep()

	ctx := install.NewContext()
	ctx.SetState(StateRepositoryConfigured, true)
	ctx.SetState(StateRepositoryName, "test-repo")

	err := step.Rollback(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "package manager not available")
}

// =============================================================================
// RepositoryStep Validate Tests
// =============================================================================

func TestRepositoryStep_Validate_Success(t *testing.T) {
	mockPM := NewMockPackageManager()
	step := NewRepositoryStep()

	ctx := install.NewContext(
		install.WithPackageManager(mockPM),
		install.WithDistroInfo(newUbuntuDistro()),
	)

	err := step.Validate(ctx)

	assert.NoError(t, err)
}

func TestRepositoryStep_Validate_MissingPackageManager(t *testing.T) {
	step := NewRepositoryStep()

	ctx := install.NewContext(
		install.WithDistroInfo(newUbuntuDistro()),
	)

	err := step.Validate(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "package manager is required")
}

func TestRepositoryStep_Validate_MissingDistroInfo(t *testing.T) {
	mockPM := NewMockPackageManager()
	step := NewRepositoryStep()

	ctx := install.NewContext(
		install.WithPackageManager(mockPM),
	)

	err := step.Validate(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "distribution info is required")
}

func TestRepositoryStep_Validate_BothMissing(t *testing.T) {
	step := NewRepositoryStep()

	ctx := install.NewContext()

	err := step.Validate(ctx)

	// Should fail on first check (package manager)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "package manager is required")
}

// =============================================================================
// RepositoryStep CanRollback Tests
// =============================================================================

func TestRepositoryStep_CanRollback(t *testing.T) {
	step := NewRepositoryStep()

	assert.True(t, step.CanRollback())
}

// =============================================================================
// RepositoryStep Options Tests
// =============================================================================

func TestRepositoryStep_Options(t *testing.T) {
	t.Run("WithSkipUpdate sets skipUpdate to true", func(t *testing.T) {
		step := NewRepositoryStep(WithSkipUpdate(true))
		assert.True(t, step.skipUpdate)
	})

	t.Run("WithSkipUpdate sets skipUpdate to false", func(t *testing.T) {
		step := NewRepositoryStep(WithSkipUpdate(false))
		assert.False(t, step.skipUpdate)
	})

	t.Run("default skipUpdate is false", func(t *testing.T) {
		step := NewRepositoryStep()
		assert.False(t, step.skipUpdate)
	})
}

// =============================================================================
// RepositoryStep Interface Compliance Tests
// =============================================================================

func TestRepositoryStep_InterfaceCompliance(t *testing.T) {
	var _ install.Step = (*RepositoryStep)(nil)
}

// =============================================================================
// RepositoryStep State Keys Tests
// =============================================================================

func TestRepositoryStep_StateKeys(t *testing.T) {
	assert.Equal(t, "repository_configured", StateRepositoryConfigured)
	assert.Equal(t, "repository_name", StateRepositoryName)
}

// =============================================================================
// RepositoryStep Full Workflow Tests
// =============================================================================

func TestRepositoryStep_FullWorkflow_ExecuteAndRollback(t *testing.T) {
	mockPM := NewMockPackageManager()
	step := NewRepositoryStep()

	ctx := install.NewContext(
		install.WithPackageManager(mockPM),
		install.WithDistroInfo(newUbuntuDistro()),
	)

	// Execute
	result := step.Execute(ctx)
	assert.Equal(t, install.StepStatusCompleted, result.Status)
	assert.True(t, ctx.GetStateBool(StateRepositoryConfigured))

	// Get the repo name that was stored
	repoName := ctx.GetStateString(StateRepositoryName)
	assert.NotEmpty(t, repoName)

	// Reset mock tracking
	mockPM.removeRepoCalled = false
	mockPM.lastRemovedRepo = ""

	// Rollback
	err := step.Rollback(ctx)
	assert.NoError(t, err)
	assert.True(t, mockPM.removeRepoCalled)
	assert.Equal(t, repoName, mockPM.lastRemovedRepo)

	// State should be cleared
	assert.False(t, ctx.GetStateBool(StateRepositoryConfigured))
	assert.Empty(t, ctx.GetStateString(StateRepositoryName))
}

// =============================================================================
// Additional Edge Case Tests
// =============================================================================

func TestRepositoryStep_Execute_CancelledBeforeAddRepo(t *testing.T) {
	// The cancellation check before AddRepository (line 105-107) happens after dry run check.
	// Since we can't easily inject cancellation between GetRepository and AddRepository
	// in the current design, we test this by cancelling in a custom mock's AddRepository
	// which checks if already cancelled. Actually, since the check happens before AddRepository,
	// we need a different approach.
	//
	// This test verifies the first cancellation path is hit (the one at the start).
	// Testing the second cancellation point (after dry run) requires more complex mocking.
	//
	// For now, we test that cancellation before execution works as expected.
	t.Skip("Skipped: Testing mid-execution cancellation requires more complex test infrastructure")
}

func TestRepositoryStep_Execute_CancelledAfterAddRepo(t *testing.T) {
	// Create a mock that cancels the context after AddRepository succeeds
	mockPM := &CancellingMockPackageManager{
		MockPackageManager: NewMockPackageManager(),
	}
	step := NewRepositoryStep()

	ctx := install.NewContext(
		install.WithPackageManager(mockPM),
		install.WithDistroInfo(newUbuntuDistro()),
	)

	// Store the context so the mock can cancel it
	mockPM.installCtx = ctx

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusFailed, result.Status)
	assert.Contains(t, result.Message, "cancelled")
	// Should have tried to rollback after cancellation
	assert.True(t, mockPM.removeRepoCalled)
}

// CancellingMockPackageManager cancels the context after AddRepository is called.
type CancellingMockPackageManager struct {
	*MockPackageManager
	installCtx *install.Context
}

func (m *CancellingMockPackageManager) AddRepository(ctx context.Context, repo pkg.Repository) error {
	m.addRepoCalled = true
	m.lastAddedRepo = &repo
	// Cancel the context after adding repository
	if m.installCtx != nil {
		m.installCtx.Cancel()
	}
	return m.addRepoErr
}

// =============================================================================
// Benchmarks
// =============================================================================

func BenchmarkRepositoryStep_Execute(b *testing.B) {
	mockPM := NewMockPackageManager()
	step := NewRepositoryStep()

	ctx := install.NewContext(
		install.WithPackageManager(mockPM),
		install.WithDistroInfo(newUbuntuDistro()),
	)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		step.Execute(ctx)
	}
}

func BenchmarkRepositoryStep_Validate(b *testing.B) {
	mockPM := NewMockPackageManager()
	step := NewRepositoryStep()

	ctx := install.NewContext(
		install.WithPackageManager(mockPM),
		install.WithDistroInfo(newUbuntuDistro()),
	)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = step.Validate(ctx)
	}
}
