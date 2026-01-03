package pkg

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tungetti/igor/internal/constants"
	igorerrors "github.com/tungetti/igor/internal/errors"
)

// =============================================================================
// Package Type Tests
// =============================================================================

func TestPackage_String(t *testing.T) {
	tests := []struct {
		name     string
		pkg      Package
		expected string
	}{
		{
			name: "installed with version",
			pkg: Package{
				Name:      "nvidia-driver-535",
				Version:   "535.154.05-1",
				Installed: true,
			},
			expected: "nvidia-driver-535 (535.154.05-1) [installed]",
		},
		{
			name: "not installed with version",
			pkg: Package{
				Name:      "nvidia-driver-535",
				Version:   "535.154.05-1",
				Installed: false,
			},
			expected: "nvidia-driver-535 (535.154.05-1) [not installed]",
		},
		{
			name: "installed without version",
			pkg: Package{
				Name:      "nvidia-driver",
				Installed: true,
			},
			expected: "nvidia-driver [installed]",
		},
		{
			name: "not installed without version",
			pkg: Package{
				Name:      "nvidia-driver",
				Installed: false,
			},
			expected: "nvidia-driver [not installed]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.pkg.String())
		})
	}
}

func TestPackage_FullName(t *testing.T) {
	tests := []struct {
		name     string
		pkg      Package
		expected string
	}{
		{
			name: "with version",
			pkg: Package{
				Name:    "nvidia-driver",
				Version: "535.154.05",
			},
			expected: "nvidia-driver-535.154.05",
		},
		{
			name: "without version",
			pkg: Package{
				Name: "nvidia-driver",
			},
			expected: "nvidia-driver",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.pkg.FullName())
		})
	}
}

func TestPackage_IsEmpty(t *testing.T) {
	tests := []struct {
		name     string
		pkg      Package
		expected bool
	}{
		{
			name:     "empty package",
			pkg:      Package{},
			expected: true,
		},
		{
			name:     "package with name",
			pkg:      Package{Name: "nvidia-driver"},
			expected: false,
		},
		{
			name:     "package with only version",
			pkg:      Package{Version: "1.0.0"},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.pkg.IsEmpty())
		})
	}
}

func TestPackage_AllFields(t *testing.T) {
	pkg := Package{
		Name:         "nvidia-driver-535",
		Version:      "535.154.05-1",
		Installed:    true,
		Repository:   "nvidia-driver",
		Description:  "NVIDIA driver metapackage",
		Size:         1024000,
		Architecture: "amd64",
		Dependencies: []string{"libc6", "libx11-6"},
	}

	assert.Equal(t, "nvidia-driver-535", pkg.Name)
	assert.Equal(t, "535.154.05-1", pkg.Version)
	assert.True(t, pkg.Installed)
	assert.Equal(t, "nvidia-driver", pkg.Repository)
	assert.Equal(t, "NVIDIA driver metapackage", pkg.Description)
	assert.Equal(t, int64(1024000), pkg.Size)
	assert.Equal(t, "amd64", pkg.Architecture)
	assert.ElementsMatch(t, []string{"libc6", "libx11-6"}, pkg.Dependencies)
}

// =============================================================================
// Repository Type Tests
// =============================================================================

func TestRepository_String(t *testing.T) {
	tests := []struct {
		name     string
		repo     Repository
		expected string
	}{
		{
			name: "enabled repository",
			repo: Repository{
				Name:    "nvidia-driver",
				URL:     "https://developer.download.nvidia.com/compute/cuda/repos",
				Enabled: true,
			},
			expected: "nvidia-driver (https://developer.download.nvidia.com/compute/cuda/repos) [enabled]",
		},
		{
			name: "disabled repository",
			repo: Repository{
				Name:    "nvidia-driver",
				URL:     "https://developer.download.nvidia.com/compute/cuda/repos",
				Enabled: false,
			},
			expected: "nvidia-driver (https://developer.download.nvidia.com/compute/cuda/repos) [disabled]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.repo.String())
		})
	}
}

func TestRepository_IsEmpty(t *testing.T) {
	tests := []struct {
		name     string
		repo     Repository
		expected bool
	}{
		{
			name:     "empty repository",
			repo:     Repository{},
			expected: true,
		},
		{
			name:     "repository with name",
			repo:     Repository{Name: "nvidia-driver"},
			expected: false,
		},
		{
			name:     "repository with only URL",
			repo:     Repository{URL: "https://example.com"},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.repo.IsEmpty())
		})
	}
}

func TestRepository_HasGPGKey(t *testing.T) {
	tests := []struct {
		name     string
		repo     Repository
		expected bool
	}{
		{
			name:     "no GPG key",
			repo:     Repository{Name: "test"},
			expected: false,
		},
		{
			name:     "with GPG key",
			repo:     Repository{Name: "test", GPGKey: "https://example.com/key.gpg"},
			expected: true,
		},
		{
			name:     "empty GPG key",
			repo:     Repository{Name: "test", GPGKey: ""},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.repo.HasGPGKey())
		})
	}
}

func TestRepository_AllFields(t *testing.T) {
	repo := Repository{
		Name:         "nvidia-driver",
		URL:          "https://developer.download.nvidia.com/compute/cuda/repos",
		Enabled:      true,
		GPGKey:       "https://developer.download.nvidia.com/compute/cuda/repos/gpg.key",
		Type:         "deb",
		Components:   []string{"main", "contrib"},
		Distribution: "ubuntu2204",
		Priority:     100,
	}

	assert.Equal(t, "nvidia-driver", repo.Name)
	assert.Equal(t, "https://developer.download.nvidia.com/compute/cuda/repos", repo.URL)
	assert.True(t, repo.Enabled)
	assert.Equal(t, "https://developer.download.nvidia.com/compute/cuda/repos/gpg.key", repo.GPGKey)
	assert.Equal(t, "deb", repo.Type)
	assert.ElementsMatch(t, []string{"main", "contrib"}, repo.Components)
	assert.Equal(t, "ubuntu2204", repo.Distribution)
	assert.Equal(t, 100, repo.Priority)
}

// =============================================================================
// Options Type Tests
// =============================================================================

func TestDefaultInstallOptions(t *testing.T) {
	opts := DefaultInstallOptions()

	assert.False(t, opts.Force)
	assert.False(t, opts.NoConfirm)
	assert.False(t, opts.SkipVerify)
	assert.False(t, opts.DownloadOnly)
	assert.False(t, opts.Reinstall)
	assert.False(t, opts.AllowDowngrade)
}

func TestNonInteractiveInstallOptions(t *testing.T) {
	opts := NonInteractiveInstallOptions()

	assert.False(t, opts.Force)
	assert.True(t, opts.NoConfirm)
	assert.False(t, opts.SkipVerify)
	assert.False(t, opts.DownloadOnly)
	assert.False(t, opts.Reinstall)
	assert.False(t, opts.AllowDowngrade)
}

func TestDefaultUpdateOptions(t *testing.T) {
	opts := DefaultUpdateOptions()

	assert.False(t, opts.Quiet)
	assert.False(t, opts.ForceRefresh)
}

func TestDefaultRemoveOptions(t *testing.T) {
	opts := DefaultRemoveOptions()

	assert.False(t, opts.Purge)
	assert.False(t, opts.AutoRemove)
	assert.False(t, opts.NoConfirm)
}

func TestDefaultSearchOptions(t *testing.T) {
	opts := DefaultSearchOptions()

	assert.True(t, opts.IncludeInstalled)
	assert.False(t, opts.ExactMatch)
	assert.Equal(t, 0, opts.Limit)
}

// =============================================================================
// Error Type Tests
// =============================================================================

func TestPackageError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *PackageError
		expected string
	}{
		{
			name:     "simple message",
			err:      ErrPackageNotFound,
			expected: "package not found",
		},
		{
			name:     "with package name",
			err:      ErrPackageNotFound.WithPackage("nvidia-driver"),
			expected: "package not found [nvidia-driver]",
		},
		{
			name:     "with operation",
			err:      ErrPackageNotFound.WithOp("pkg.Install"),
			expected: "pkg.Install: package not found",
		},
		{
			name:     "with cause",
			err:      ErrPackageNotFound.WithCause(fmt.Errorf("underlying error")),
			expected: "package not found: underlying error",
		},
		{
			name:     "with all fields",
			err:      ErrPackageNotFound.WithPackage("nvidia-driver").WithOp("pkg.Install").WithCause(fmt.Errorf("apt error")),
			expected: "pkg.Install: package not found [nvidia-driver]: apt error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.err.Error())
		})
	}
}

func TestPackageError_Unwrap(t *testing.T) {
	cause := fmt.Errorf("original error")
	err := ErrInstallFailed.WithCause(cause)

	unwrapped := err.Unwrap()

	assert.Equal(t, cause, unwrapped)
}

func TestPackageError_Unwrap_NoCause(t *testing.T) {
	err := ErrPackageNotFound

	unwrapped := err.Unwrap()

	assert.Nil(t, unwrapped)
}

func TestPackageError_Is(t *testing.T) {
	// Same sentinel should match
	err1 := ErrPackageNotFound.WithPackage("pkg1")
	err2 := ErrPackageNotFound.WithPackage("pkg2")

	assert.True(t, errors.Is(err1, err2))
	assert.True(t, errors.Is(err2, err1))
	assert.True(t, errors.Is(err1, ErrPackageNotFound))

	// Different sentinels should not match
	assert.False(t, errors.Is(err1, ErrPackageInstalled))
	assert.False(t, errors.Is(ErrInstallFailed, ErrRemoveFailed))
}

func TestPackageError_Is_NonPackageError(t *testing.T) {
	err := ErrPackageNotFound
	stdErr := fmt.Errorf("standard error")

	assert.False(t, errors.Is(err, stdErr))
}

func TestPackageError_Code(t *testing.T) {
	tests := []struct {
		name     string
		err      *PackageError
		expected igorerrors.Code
	}{
		{"ErrPackageNotFound", ErrPackageNotFound, igorerrors.NotFound},
		{"ErrPackageInstalled", ErrPackageInstalled, igorerrors.AlreadyExists},
		{"ErrPackageNotInstalled", ErrPackageNotInstalled, igorerrors.NotFound},
		{"ErrRepositoryExists", ErrRepositoryExists, igorerrors.AlreadyExists},
		{"ErrRepositoryNotFound", ErrRepositoryNotFound, igorerrors.NotFound},
		{"ErrUpdateFailed", ErrUpdateFailed, igorerrors.PackageManager},
		{"ErrInstallFailed", ErrInstallFailed, igorerrors.Installation},
		{"ErrRemoveFailed", ErrRemoveFailed, igorerrors.PackageManager},
		{"ErrLockAcquireFailed", ErrLockAcquireFailed, igorerrors.PackageManager},
		{"ErrDependencyConflict", ErrDependencyConflict, igorerrors.PackageManager},
		{"ErrGPGVerificationFailed", ErrGPGVerificationFailed, igorerrors.Validation},
		{"ErrUnsupportedOperation", ErrUnsupportedOperation, igorerrors.Unsupported},
		{"ErrNetworkUnavailable", ErrNetworkUnavailable, igorerrors.Network},
		{"ErrInsufficientSpace", ErrInsufficientSpace, igorerrors.PackageManager},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.err.Code())
		})
	}
}

func TestPackageError_PackageName(t *testing.T) {
	err := ErrPackageNotFound.WithPackage("nvidia-driver")

	assert.Equal(t, "nvidia-driver", err.PackageName())
}

func TestPackageError_Immutability(t *testing.T) {
	// Ensure WithX methods create new errors, not modify existing
	original := ErrPackageNotFound
	withPkg := original.WithPackage("test")
	withOp := original.WithOp("test.Op")

	assert.Empty(t, original.PackageName())
	assert.Equal(t, "test", withPkg.PackageName())
	assert.NotEqual(t, original, withPkg)
	assert.NotEqual(t, original, withOp)
}

func TestWrap(t *testing.T) {
	cause := fmt.Errorf("underlying")
	err := Wrap(ErrInstallFailed, cause)

	assert.True(t, errors.Is(err, ErrInstallFailed))
	assert.Equal(t, cause, err.Unwrap())
}

func TestWrapWithPackage(t *testing.T) {
	cause := fmt.Errorf("underlying")
	err := WrapWithPackage(ErrInstallFailed, "nvidia-driver", cause)

	assert.True(t, errors.Is(err, ErrInstallFailed))
	assert.Equal(t, "nvidia-driver", err.PackageName())
	assert.Equal(t, cause, err.Unwrap())
}

func TestNewPackageError(t *testing.T) {
	err := NewPackageError(igorerrors.PackageManager, "custom error")

	assert.Equal(t, igorerrors.PackageManager, err.Code())
	assert.Equal(t, "custom error", err.Error())
}

func TestNewPackageErrorf(t *testing.T) {
	err := NewPackageErrorf(igorerrors.PackageManager, "failed to install %s", "nvidia-driver")

	assert.Equal(t, igorerrors.PackageManager, err.Code())
	assert.Equal(t, "failed to install nvidia-driver", err.Error())
}

func TestIsPackageError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"PackageError", ErrPackageNotFound, true},
		{"wrapped PackageError", fmt.Errorf("wrapped: %w", ErrPackageNotFound), true},
		{"standard error", fmt.Errorf("standard"), false},
		{"nil error", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, IsPackageError(tt.err))
		})
	}
}

func TestGetPackageError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		wantNil  bool
		wantCode igorerrors.Code
	}{
		{"PackageError", ErrPackageNotFound, false, igorerrors.NotFound},
		{"wrapped PackageError", fmt.Errorf("wrapped: %w", ErrInstallFailed), false, igorerrors.Installation},
		{"standard error", fmt.Errorf("standard"), true, igorerrors.Unknown},
		{"nil error", nil, true, igorerrors.Unknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pe := GetPackageError(tt.err)
			if tt.wantNil {
				assert.Nil(t, pe)
			} else {
				require.NotNil(t, pe)
				assert.Equal(t, tt.wantCode, pe.Code())
			}
		})
	}
}

func TestPackageError_ToIgorError(t *testing.T) {
	cause := fmt.Errorf("underlying")
	pkgErr := ErrInstallFailed.WithPackage("nvidia-driver").WithOp("pkg.Install").WithCause(cause)

	igorErr := pkgErr.ToIgorError()

	require.NotNil(t, igorErr)
	assert.Equal(t, igorerrors.Installation, igorErr.Code)
	assert.Equal(t, "pkg.Install", igorErr.Op)
	assert.Equal(t, cause, igorErr.Cause)
}

func TestSentinelErrors_Existence(t *testing.T) {
	// Verify all sentinel errors are properly defined
	sentinels := []*PackageError{
		ErrPackageNotFound,
		ErrPackageInstalled,
		ErrPackageNotInstalled,
		ErrRepositoryExists,
		ErrRepositoryNotFound,
		ErrUpdateFailed,
		ErrInstallFailed,
		ErrRemoveFailed,
		ErrLockAcquireFailed,
		ErrDependencyConflict,
		ErrGPGVerificationFailed,
		ErrUnsupportedOperation,
		ErrNetworkUnavailable,
		ErrInsufficientSpace,
	}

	for _, s := range sentinels {
		require.NotNil(t, s)
		assert.NotEmpty(t, s.Error())
	}
}

func TestSentinelErrors_AreDistinct(t *testing.T) {
	sentinels := []*PackageError{
		ErrPackageNotFound,
		ErrPackageInstalled,
		ErrPackageNotInstalled,
		ErrRepositoryExists,
		ErrRepositoryNotFound,
		ErrUpdateFailed,
		ErrInstallFailed,
		ErrRemoveFailed,
		ErrLockAcquireFailed,
		ErrDependencyConflict,
		ErrGPGVerificationFailed,
		ErrUnsupportedOperation,
		ErrNetworkUnavailable,
		ErrInsufficientSpace,
	}

	// Each sentinel should only match itself
	for i, s1 := range sentinels {
		for j, s2 := range sentinels {
			if i == j {
				assert.True(t, errors.Is(s1, s2), "sentinel %d should match itself", i)
			} else {
				assert.False(t, errors.Is(s1, s2), "sentinel %d should not match sentinel %d", i, j)
			}
		}
	}
}

// =============================================================================
// HistoryEntry Type Tests
// =============================================================================

func TestHistoryEntry_String(t *testing.T) {
	tests := []struct {
		name     string
		entry    HistoryEntry
		expected string
	}{
		{
			name: "successful operation",
			entry: HistoryEntry{
				Operation: "install",
				Success:   true,
			},
			expected: "install [success]",
		},
		{
			name: "failed operation",
			entry: HistoryEntry{
				Operation: "remove",
				Success:   false,
			},
			expected: "remove [failed]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.entry.String())
		})
	}
}

func TestHistoryEntry_AllFields(t *testing.T) {
	entry := HistoryEntry{
		ID:        "tx-123",
		Timestamp: 1704067200,
		Operation: "install",
		Packages:  []string{"nvidia-driver", "cuda-toolkit"},
		Success:   true,
		Details:   "Installed 2 packages",
	}

	assert.Equal(t, "tx-123", entry.ID)
	assert.Equal(t, int64(1704067200), entry.Timestamp)
	assert.Equal(t, "install", entry.Operation)
	assert.ElementsMatch(t, []string{"nvidia-driver", "cuda-toolkit"}, entry.Packages)
	assert.True(t, entry.Success)
	assert.Equal(t, "Installed 2 packages", entry.Details)
}

// =============================================================================
// Interface Compliance Tests
// =============================================================================

// MockManager is a test implementation of the Manager interface
type MockManager struct {
	name   string
	family constants.DistroFamily
}

func (m *MockManager) Install(ctx context.Context, opts InstallOptions, packages ...string) error {
	return nil
}

func (m *MockManager) Remove(ctx context.Context, opts RemoveOptions, packages ...string) error {
	return nil
}

func (m *MockManager) Update(ctx context.Context, opts UpdateOptions) error {
	return nil
}

func (m *MockManager) Upgrade(ctx context.Context, opts InstallOptions, packages ...string) error {
	return nil
}

func (m *MockManager) IsInstalled(ctx context.Context, pkg string) (bool, error) {
	return false, nil
}

func (m *MockManager) Search(ctx context.Context, query string, opts SearchOptions) ([]Package, error) {
	return nil, nil
}

func (m *MockManager) Info(ctx context.Context, pkg string) (*Package, error) {
	return nil, nil
}

func (m *MockManager) ListInstalled(ctx context.Context) ([]Package, error) {
	return nil, nil
}

func (m *MockManager) ListUpgradable(ctx context.Context) ([]Package, error) {
	return nil, nil
}

func (m *MockManager) AddRepository(ctx context.Context, repo Repository) error {
	return nil
}

func (m *MockManager) RemoveRepository(ctx context.Context, name string) error {
	return nil
}

func (m *MockManager) ListRepositories(ctx context.Context) ([]Repository, error) {
	return nil, nil
}

func (m *MockManager) EnableRepository(ctx context.Context, name string) error {
	return nil
}

func (m *MockManager) DisableRepository(ctx context.Context, name string) error {
	return nil
}

func (m *MockManager) RefreshRepositories(ctx context.Context) error {
	return nil
}

func (m *MockManager) Clean(ctx context.Context) error {
	return nil
}

func (m *MockManager) AutoRemove(ctx context.Context) error {
	return nil
}

func (m *MockManager) Verify(ctx context.Context, pkg string) (bool, error) {
	return false, nil
}

func (m *MockManager) Name() string {
	return m.name
}

func (m *MockManager) Family() constants.DistroFamily {
	return m.family
}

func (m *MockManager) IsAvailable() bool {
	return true
}

func TestManager_InterfaceCompliance(t *testing.T) {
	// Compile-time check that MockManager implements Manager
	var _ Manager = &MockManager{}
}

func TestMockManager_Methods(t *testing.T) {
	ctx := context.Background()
	m := &MockManager{
		name:   "apt",
		family: constants.FamilyDebian,
	}

	// Test all methods can be called without panic
	assert.NoError(t, m.Install(ctx, DefaultInstallOptions(), "pkg"))
	assert.NoError(t, m.Remove(ctx, DefaultRemoveOptions(), "pkg"))
	assert.NoError(t, m.Update(ctx, DefaultUpdateOptions()))
	assert.NoError(t, m.Upgrade(ctx, DefaultInstallOptions(), "pkg"))

	installed, err := m.IsInstalled(ctx, "pkg")
	assert.NoError(t, err)
	assert.False(t, installed)

	pkgs, err := m.Search(ctx, "query", DefaultSearchOptions())
	assert.NoError(t, err)
	assert.Nil(t, pkgs)

	info, err := m.Info(ctx, "pkg")
	assert.NoError(t, err)
	assert.Nil(t, info)

	installed_pkgs, err := m.ListInstalled(ctx)
	assert.NoError(t, err)
	assert.Nil(t, installed_pkgs)

	upgradable, err := m.ListUpgradable(ctx)
	assert.NoError(t, err)
	assert.Nil(t, upgradable)

	assert.NoError(t, m.AddRepository(ctx, Repository{}))
	assert.NoError(t, m.RemoveRepository(ctx, "repo"))

	repos, err := m.ListRepositories(ctx)
	assert.NoError(t, err)
	assert.Nil(t, repos)

	assert.NoError(t, m.EnableRepository(ctx, "repo"))
	assert.NoError(t, m.DisableRepository(ctx, "repo"))
	assert.NoError(t, m.RefreshRepositories(ctx))
	assert.NoError(t, m.Clean(ctx))
	assert.NoError(t, m.AutoRemove(ctx))

	verified, err := m.Verify(ctx, "pkg")
	assert.NoError(t, err)
	assert.False(t, verified)

	assert.Equal(t, "apt", m.Name())
	assert.Equal(t, constants.FamilyDebian, m.Family())
	assert.True(t, m.IsAvailable())
}

// =============================================================================
// Context Cancellation Tests
// =============================================================================

func TestManager_ContextUsage(t *testing.T) {
	// Test that context can be cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	m := &MockManager{name: "apt", family: constants.FamilyDebian}

	// Our mock doesn't check context, but real implementations should
	// This test verifies the interface accepts context correctly
	_ = m.Install(ctx, DefaultInstallOptions(), "pkg")
	_ = m.Remove(ctx, DefaultRemoveOptions(), "pkg")
	_ = m.Update(ctx, DefaultUpdateOptions())

	// Verify context is already cancelled
	assert.Error(t, ctx.Err())
}

// =============================================================================
// Error Chaining Tests
// =============================================================================

func TestPackageError_ErrorChain(t *testing.T) {
	// Create a chain of errors
	rootCause := fmt.Errorf("disk full")
	level1 := ErrInstallFailed.WithCause(rootCause)
	level2 := fmt.Errorf("installation aborted: %w", level1)

	// Should be able to find ErrInstallFailed in the chain
	assert.True(t, errors.Is(level2, ErrInstallFailed))

	// Should be able to extract PackageError
	var pe *PackageError
	require.True(t, errors.As(level2, &pe))
	assert.Equal(t, igorerrors.Installation, pe.Code())

	// Root cause should be accessible
	assert.Contains(t, level2.Error(), "disk full")
}

func TestPackageError_MultipleWrapping(t *testing.T) {
	// Wrap multiple times with different contexts
	err := ErrPackageNotFound.
		WithPackage("nvidia-driver").
		WithOp("apt.Install").
		WithCause(fmt.Errorf("404 not found"))

	assert.Contains(t, err.Error(), "nvidia-driver")
	assert.Contains(t, err.Error(), "apt.Install")
	assert.Contains(t, err.Error(), "404 not found")
	assert.True(t, errors.Is(err, ErrPackageNotFound))
}

// =============================================================================
// Edge Cases
// =============================================================================

func TestPackage_ZeroValue(t *testing.T) {
	var pkg Package

	assert.Empty(t, pkg.Name)
	assert.Empty(t, pkg.Version)
	assert.False(t, pkg.Installed)
	assert.True(t, pkg.IsEmpty())
	assert.Equal(t, " [not installed]", pkg.String())
}

func TestRepository_ZeroValue(t *testing.T) {
	var repo Repository

	assert.Empty(t, repo.Name)
	assert.Empty(t, repo.URL)
	assert.False(t, repo.Enabled)
	assert.True(t, repo.IsEmpty())
	assert.False(t, repo.HasGPGKey())
	assert.Equal(t, " () [disabled]", repo.String())
}

func TestHistoryEntry_ZeroValue(t *testing.T) {
	var entry HistoryEntry

	assert.Empty(t, entry.ID)
	assert.Equal(t, int64(0), entry.Timestamp)
	assert.Empty(t, entry.Operation)
	assert.False(t, entry.Success)
	assert.Equal(t, " [failed]", entry.String())
}
