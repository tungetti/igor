package uninstall

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tungetti/igor/internal/constants"
	"github.com/tungetti/igor/internal/distro"
	"github.com/tungetti/igor/internal/exec"
	"github.com/tungetti/igor/internal/pkg"
)

// =============================================================================
// Mock Package Manager
// =============================================================================

// MockPackageManager is a test implementation of pkg.Manager
type MockPackageManager struct {
	name              string
	family            constants.DistroFamily
	installedPackages []pkg.Package
	listInstalledErr  error
}

func NewMockPackageManager() *MockPackageManager {
	return &MockPackageManager{
		name:              "mock",
		family:            constants.FamilyDebian,
		installedPackages: []pkg.Package{},
	}
}

func (m *MockPackageManager) SetInstalledPackages(packages []pkg.Package) {
	m.installedPackages = packages
}

func (m *MockPackageManager) SetListInstalledError(err error) {
	m.listInstalledErr = err
}

func (m *MockPackageManager) SetFamily(family constants.DistroFamily) {
	m.family = family
}

func (m *MockPackageManager) Install(ctx context.Context, opts pkg.InstallOptions, packages ...string) error {
	return nil
}

func (m *MockPackageManager) Remove(ctx context.Context, opts pkg.RemoveOptions, packages ...string) error {
	return nil
}

func (m *MockPackageManager) Update(ctx context.Context, opts pkg.UpdateOptions) error {
	return nil
}

func (m *MockPackageManager) Upgrade(ctx context.Context, opts pkg.InstallOptions, packages ...string) error {
	return nil
}

func (m *MockPackageManager) IsInstalled(ctx context.Context, pkgName string) (bool, error) {
	for _, p := range m.installedPackages {
		if p.Name == pkgName {
			return true, nil
		}
	}
	return false, nil
}

func (m *MockPackageManager) Search(ctx context.Context, query string, opts pkg.SearchOptions) ([]pkg.Package, error) {
	return nil, nil
}

func (m *MockPackageManager) Info(ctx context.Context, pkgName string) (*pkg.Package, error) {
	return nil, nil
}

func (m *MockPackageManager) ListInstalled(ctx context.Context) ([]pkg.Package, error) {
	if m.listInstalledErr != nil {
		return nil, m.listInstalledErr
	}
	return m.installedPackages, nil
}

func (m *MockPackageManager) ListUpgradable(ctx context.Context) ([]pkg.Package, error) {
	return nil, nil
}

func (m *MockPackageManager) AddRepository(ctx context.Context, repo pkg.Repository) error {
	return nil
}

func (m *MockPackageManager) RemoveRepository(ctx context.Context, name string) error {
	return nil
}

func (m *MockPackageManager) ListRepositories(ctx context.Context) ([]pkg.Repository, error) {
	return nil, nil
}

func (m *MockPackageManager) EnableRepository(ctx context.Context, name string) error {
	return nil
}

func (m *MockPackageManager) DisableRepository(ctx context.Context, name string) error {
	return nil
}

func (m *MockPackageManager) RefreshRepositories(ctx context.Context) error {
	return nil
}

func (m *MockPackageManager) Clean(ctx context.Context) error {
	return nil
}

func (m *MockPackageManager) AutoRemove(ctx context.Context) error {
	return nil
}

func (m *MockPackageManager) Verify(ctx context.Context, pkgName string) (bool, error) {
	return false, nil
}

func (m *MockPackageManager) Name() string {
	return m.name
}

func (m *MockPackageManager) Family() constants.DistroFamily {
	return m.family
}

func (m *MockPackageManager) IsAvailable() bool {
	return true
}

// Compile-time check that MockPackageManager implements pkg.Manager
var _ pkg.Manager = (*MockPackageManager)(nil)

// =============================================================================
// NewPackageDiscovery Tests
// =============================================================================

func TestNewPackageDiscovery(t *testing.T) {
	t.Run("with package manager only", func(t *testing.T) {
		pm := NewMockPackageManager()
		pd := NewPackageDiscovery(pm)

		assert.NotNil(t, pd)
		assert.Equal(t, pm, pd.pm)
		assert.Nil(t, pd.distro)
		assert.Nil(t, pd.exec)
	})

	t.Run("with distro option", func(t *testing.T) {
		pm := NewMockPackageManager()
		dist := &distro.Distribution{
			ID:     "ubuntu",
			Name:   "Ubuntu",
			Family: constants.FamilyDebian,
		}

		pd := NewPackageDiscovery(pm, WithDiscoveryDistro(dist))

		assert.NotNil(t, pd)
		assert.Equal(t, dist, pd.distro)
	})

	t.Run("with executor option", func(t *testing.T) {
		pm := NewMockPackageManager()
		executor := exec.NewMockExecutor()

		pd := NewPackageDiscovery(pm, WithDiscoveryExecutor(executor))

		assert.NotNil(t, pd)
		assert.Equal(t, executor, pd.exec)
	})

	t.Run("with all options", func(t *testing.T) {
		pm := NewMockPackageManager()
		dist := &distro.Distribution{
			ID:     "fedora",
			Name:   "Fedora",
			Family: constants.FamilyRHEL,
		}
		executor := exec.NewMockExecutor()

		pd := NewPackageDiscovery(pm,
			WithDiscoveryDistro(dist),
			WithDiscoveryExecutor(executor),
		)

		assert.NotNil(t, pd)
		assert.Equal(t, pm, pd.pm)
		assert.Equal(t, dist, pd.distro)
		assert.Equal(t, executor, pd.exec)
	})
}

// =============================================================================
// Discover Tests
// =============================================================================

func TestPackageDiscovery_Discover(t *testing.T) {
	ctx := context.Background()

	t.Run("nil package manager returns error", func(t *testing.T) {
		pd := NewPackageDiscovery(nil)

		result, err := pd.Discover(ctx)

		assert.Nil(t, result)
		assert.Error(t, err)
	})

	t.Run("empty package list", func(t *testing.T) {
		pm := NewMockPackageManager()
		pd := NewPackageDiscovery(pm)

		result, err := pd.Discover(ctx)

		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.True(t, result.IsEmpty())
		assert.Equal(t, 0, result.TotalCount)
		assert.Empty(t, result.AllPackages)
	})

	t.Run("no nvidia packages", func(t *testing.T) {
		pm := NewMockPackageManager()
		pm.SetInstalledPackages([]pkg.Package{
			{Name: "vim", Installed: true},
			{Name: "git", Installed: true},
			{Name: "python3", Installed: true},
		})
		pd := NewPackageDiscovery(pm)

		result, err := pd.Discover(ctx)

		require.NoError(t, err)
		assert.True(t, result.IsEmpty())
		assert.Equal(t, 0, result.TotalCount)
	})

	t.Run("with nvidia driver packages - debian", func(t *testing.T) {
		pm := NewMockPackageManager()
		pm.SetFamily(constants.FamilyDebian)
		pm.SetInstalledPackages([]pkg.Package{
			{Name: "nvidia-driver-550", Installed: true},
			{Name: "nvidia-settings", Installed: true},
			{Name: "libnvidia-gl-550", Installed: true},
			{Name: "nvidia-dkms", Installed: true},
			{Name: "vim", Installed: true}, // non-nvidia package
		})
		pd := NewPackageDiscovery(pm)

		result, err := pd.Discover(ctx)

		require.NoError(t, err)
		assert.False(t, result.IsEmpty())
		assert.Equal(t, 4, result.TotalCount)
		assert.True(t, result.HasDriver())
		assert.Contains(t, result.DriverPackages, "nvidia-driver-550")
		assert.Contains(t, result.DriverPackages, "libnvidia-gl-550")
		assert.Contains(t, result.UtilityPackages, "nvidia-settings")
		assert.Contains(t, result.KernelModulePackages, "nvidia-dkms")
		assert.Equal(t, "550", result.DriverVersion)
	})

	t.Run("with cuda packages - debian", func(t *testing.T) {
		pm := NewMockPackageManager()
		pm.SetFamily(constants.FamilyDebian)
		pm.SetInstalledPackages([]pkg.Package{
			{Name: "cuda-toolkit-12-4", Installed: true},
			{Name: "cuda-runtime-12-4", Installed: true},
			{Name: "libcudnn8", Installed: true},
		})
		pd := NewPackageDiscovery(pm)

		result, err := pd.Discover(ctx)

		require.NoError(t, err)
		assert.True(t, result.HasCUDA())
		assert.Contains(t, result.CUDAPackages, "cuda-toolkit-12-4")
		assert.Contains(t, result.CUDAPackages, "cuda-runtime-12-4")
		assert.Contains(t, result.LibraryPackages, "libcudnn8")
		assert.Equal(t, "12.4", result.CUDAVersion)
	})

	t.Run("rhel family packages", func(t *testing.T) {
		pm := NewMockPackageManager()
		pm.SetFamily(constants.FamilyRHEL)
		pm.SetInstalledPackages([]pkg.Package{
			{Name: "nvidia-driver", Installed: true},
			{Name: "kmod-nvidia", Installed: true},
			{Name: "akmod-nvidia", Installed: true},
			{Name: "xorg-x11-drv-nvidia", Installed: true},
			{Name: "nvidia-settings", Installed: true},
		})
		pd := NewPackageDiscovery(pm)

		result, err := pd.Discover(ctx)

		require.NoError(t, err)
		assert.True(t, result.HasDriver())
		// kmod-nvidia and akmod-nvidia are kernel packages in RHEL
		assert.Contains(t, result.KernelModulePackages, "kmod-nvidia")
		assert.Contains(t, result.KernelModulePackages, "akmod-nvidia")
	})

	t.Run("arch family packages", func(t *testing.T) {
		pm := NewMockPackageManager()
		pm.SetFamily(constants.FamilyArch)
		pm.SetInstalledPackages([]pkg.Package{
			{Name: "nvidia", Installed: true},
			{Name: "nvidia-utils", Installed: true},
			{Name: "nvidia-settings", Installed: true},
			{Name: "nvidia-dkms", Installed: true},
			{Name: "cuda", Installed: true},
			{Name: "cudnn", Installed: true},
		})
		pd := NewPackageDiscovery(pm)

		result, err := pd.Discover(ctx)

		require.NoError(t, err)
		assert.True(t, result.HasDriver())
		assert.True(t, result.HasCUDA())
		assert.Contains(t, result.DriverPackages, "nvidia")
		assert.Contains(t, result.UtilityPackages, "nvidia-utils")
		assert.Contains(t, result.UtilityPackages, "nvidia-settings")
		assert.Contains(t, result.KernelModulePackages, "nvidia-dkms")
		assert.Contains(t, result.CUDAPackages, "cuda")
		assert.Contains(t, result.LibraryPackages, "cudnn")
	})

	t.Run("suse family packages", func(t *testing.T) {
		pm := NewMockPackageManager()
		pm.SetFamily(constants.FamilySUSE)
		pm.SetInstalledPackages([]pkg.Package{
			{Name: "nvidia-driver-G06-kmp-default", Installed: true},
			{Name: "nvidia-video-G06", Installed: true},
			{Name: "nvidia-settings", Installed: true},
			{Name: "nvidia-gfxG06-kmp-default", Installed: true},
		})
		pd := NewPackageDiscovery(pm)

		result, err := pd.Discover(ctx)

		require.NoError(t, err)
		assert.True(t, result.HasDriver())
		assert.Contains(t, result.DriverPackages, "nvidia-driver-G06-kmp-default")
		assert.Contains(t, result.DriverPackages, "nvidia-video-G06")
		assert.Contains(t, result.UtilityPackages, "nvidia-settings")
		assert.Contains(t, result.KernelModulePackages, "nvidia-gfxG06-kmp-default")
	})

	t.Run("list installed error", func(t *testing.T) {
		pm := NewMockPackageManager()
		pm.SetListInstalledError(pkg.ErrRemoveFailed)
		pd := NewPackageDiscovery(pm)

		result, err := pd.Discover(ctx)

		assert.Nil(t, result)
		assert.Error(t, err)
	})

	t.Run("uses distro family when set", func(t *testing.T) {
		pm := NewMockPackageManager()
		pm.SetFamily(constants.FamilyDebian) // PM says Debian
		dist := &distro.Distribution{
			ID:     "arch",
			Family: constants.FamilyArch, // But distro says Arch
		}
		pm.SetInstalledPackages([]pkg.Package{
			{Name: "nvidia", Installed: true}, // Arch-style package
		})

		pd := NewPackageDiscovery(pm, WithDiscoveryDistro(dist))

		result, err := pd.Discover(ctx)

		require.NoError(t, err)
		// Should use distro family (Arch), not PM family (Debian)
		assert.Contains(t, result.DriverPackages, "nvidia")
	})

	t.Run("discovery time is set", func(t *testing.T) {
		pm := NewMockPackageManager()
		pm.SetInstalledPackages([]pkg.Package{
			{Name: "nvidia-driver-550", Installed: true},
		})
		pd := NewPackageDiscovery(pm)

		before := time.Now()
		result, err := pd.Discover(ctx)
		after := time.Now()

		require.NoError(t, err)
		assert.False(t, result.DiscoveryTime.IsZero())
		assert.True(t, result.DiscoveryTime.After(before) || result.DiscoveryTime.Equal(before))
		assert.True(t, result.DiscoveryTime.Before(after) || result.DiscoveryTime.Equal(after))
	})
}

// =============================================================================
// DiscoverDriver Tests
// =============================================================================

func TestPackageDiscovery_DiscoverDriver(t *testing.T) {
	ctx := context.Background()

	t.Run("returns driver packages and version", func(t *testing.T) {
		pm := NewMockPackageManager()
		pm.SetFamily(constants.FamilyDebian)
		pm.SetInstalledPackages([]pkg.Package{
			{Name: "nvidia-driver-550", Installed: true},
			{Name: "nvidia-settings", Installed: true},
			{Name: "cuda-toolkit-12-4", Installed: true},
		})
		pd := NewPackageDiscovery(pm)

		packages, version, err := pd.DiscoverDriver(ctx)

		require.NoError(t, err)
		assert.Contains(t, packages, "nvidia-driver-550")
		assert.NotContains(t, packages, "cuda-toolkit-12-4")
		assert.Equal(t, "550", version)
	})

	t.Run("empty when no drivers", func(t *testing.T) {
		pm := NewMockPackageManager()
		pm.SetInstalledPackages([]pkg.Package{
			{Name: "vim", Installed: true},
		})
		pd := NewPackageDiscovery(pm)

		packages, version, err := pd.DiscoverDriver(ctx)

		require.NoError(t, err)
		assert.Empty(t, packages)
		assert.Empty(t, version)
	})
}

// =============================================================================
// DiscoverCUDA Tests
// =============================================================================

func TestPackageDiscovery_DiscoverCUDA(t *testing.T) {
	ctx := context.Background()

	t.Run("returns cuda packages and version", func(t *testing.T) {
		pm := NewMockPackageManager()
		pm.SetFamily(constants.FamilyDebian)
		pm.SetInstalledPackages([]pkg.Package{
			{Name: "nvidia-driver-550", Installed: true},
			{Name: "cuda-toolkit-12-4", Installed: true},
			{Name: "cuda-runtime-12-4", Installed: true},
		})
		pd := NewPackageDiscovery(pm)

		packages, version, err := pd.DiscoverCUDA(ctx)

		require.NoError(t, err)
		assert.Contains(t, packages, "cuda-toolkit-12-4")
		assert.Contains(t, packages, "cuda-runtime-12-4")
		assert.NotContains(t, packages, "nvidia-driver-550")
		assert.Equal(t, "12.4", version)
	})

	t.Run("empty when no cuda", func(t *testing.T) {
		pm := NewMockPackageManager()
		pm.SetInstalledPackages([]pkg.Package{
			{Name: "nvidia-driver-550", Installed: true},
		})
		pd := NewPackageDiscovery(pm)

		packages, version, err := pd.DiscoverCUDA(ctx)

		require.NoError(t, err)
		assert.Empty(t, packages)
		assert.Empty(t, version)
	})
}

// =============================================================================
// IsNVIDIAInstalled Tests
// =============================================================================

func TestPackageDiscovery_IsNVIDIAInstalled(t *testing.T) {
	ctx := context.Background()

	t.Run("returns true when nvidia packages installed", func(t *testing.T) {
		pm := NewMockPackageManager()
		pm.SetInstalledPackages([]pkg.Package{
			{Name: "nvidia-driver-550", Installed: true},
		})
		pd := NewPackageDiscovery(pm)

		installed, err := pd.IsNVIDIAInstalled(ctx)

		require.NoError(t, err)
		assert.True(t, installed)
	})

	t.Run("returns false when no nvidia packages", func(t *testing.T) {
		pm := NewMockPackageManager()
		pm.SetInstalledPackages([]pkg.Package{
			{Name: "vim", Installed: true},
			{Name: "git", Installed: true},
		})
		pd := NewPackageDiscovery(pm)

		installed, err := pd.IsNVIDIAInstalled(ctx)

		require.NoError(t, err)
		assert.False(t, installed)
	})

	t.Run("returns false when empty", func(t *testing.T) {
		pm := NewMockPackageManager()
		pd := NewPackageDiscovery(pm)

		installed, err := pd.IsNVIDIAInstalled(ctx)

		require.NoError(t, err)
		assert.False(t, installed)
	})
}

// =============================================================================
// GetDriverVersion Tests
// =============================================================================

func TestPackageDiscovery_GetDriverVersion(t *testing.T) {
	ctx := context.Background()

	t.Run("returns version when driver installed", func(t *testing.T) {
		pm := NewMockPackageManager()
		pm.SetFamily(constants.FamilyDebian)
		pm.SetInstalledPackages([]pkg.Package{
			{Name: "nvidia-driver-550", Installed: true},
		})
		pd := NewPackageDiscovery(pm)

		version, err := pd.GetDriverVersion(ctx)

		require.NoError(t, err)
		assert.Equal(t, "550", version)
	})

	t.Run("returns empty when no driver", func(t *testing.T) {
		pm := NewMockPackageManager()
		pd := NewPackageDiscovery(pm)

		version, err := pd.GetDriverVersion(ctx)

		require.NoError(t, err)
		assert.Empty(t, version)
	})
}

// =============================================================================
// GetDriverVersionFromPackages Tests
// =============================================================================

func TestGetDriverVersionFromPackages(t *testing.T) {
	tests := []struct {
		name     string
		packages []string
		expected string
	}{
		{
			name:     "nvidia-driver-550",
			packages: []string{"nvidia-driver-550"},
			expected: "550",
		},
		{
			name:     "nvidia-driver-550-server",
			packages: []string{"nvidia-driver-550-server"},
			expected: "550",
		},
		{
			name:     "nvidia-utils-550",
			packages: []string{"nvidia-utils-550"},
			expected: "550",
		},
		{
			name:     "libnvidia-gl-550",
			packages: []string{"libnvidia-gl-550"},
			expected: "550",
		},
		{
			name:     "multiple packages - first match wins",
			packages: []string{"libnvidia-gl-550", "nvidia-driver-560"},
			expected: "550",
		},
		{
			name:     "no version in package name",
			packages: []string{"nvidia-settings", "nvidia-utils"},
			expected: "",
		},
		{
			name:     "empty list",
			packages: []string{},
			expected: "",
		},
		{
			name:     "nil list",
			packages: nil,
			expected: "",
		},
		{
			name:     "xorg-x11-drv-nvidia-550",
			packages: []string{"xorg-x11-drv-nvidia-550"},
			expected: "550",
		},
		{
			name:     "nvidia-470-driver",
			packages: []string{"nvidia-470-driver"},
			expected: "470",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetDriverVersionFromPackages(tt.packages)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// =============================================================================
// GetCUDAVersionFromPackages Tests
// =============================================================================

func TestGetCUDAVersionFromPackages(t *testing.T) {
	tests := []struct {
		name     string
		packages []string
		expected string
	}{
		{
			name:     "cuda-toolkit-12-4",
			packages: []string{"cuda-toolkit-12-4"},
			expected: "12.4",
		},
		{
			name:     "cuda-12-4",
			packages: []string{"cuda-12-4"},
			expected: "12.4",
		},
		{
			name:     "cuda-runtime-12-4",
			packages: []string{"cuda-runtime-12-4"},
			expected: "12.4",
		},
		{
			name:     "cuda-libraries-12-4",
			packages: []string{"cuda-libraries-12-4"},
			expected: "12.4",
		},
		{
			name:     "cuda-toolkit-11-8",
			packages: []string{"cuda-toolkit-11-8"},
			expected: "11.8",
		},
		{
			name:     "multiple packages - first match wins",
			packages: []string{"cuda-12-4", "cuda-toolkit-11-8"},
			expected: "12.4",
		},
		{
			name:     "no version in package name",
			packages: []string{"cuda", "cuda-tools"},
			expected: "",
		},
		{
			name:     "empty list",
			packages: []string{},
			expected: "",
		},
		{
			name:     "nil list",
			packages: nil,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetCUDAVersionFromPackages(tt.packages)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// =============================================================================
// FilterNVIDIAPackages Tests
// =============================================================================

func TestFilterNVIDIAPackages(t *testing.T) {
	t.Run("filters nvidia packages correctly", func(t *testing.T) {
		packages := []string{
			"nvidia-driver-550",
			"vim",
			"cuda-toolkit-12-4",
			"git",
			"libnvidia-gl-550",
			"python3",
			"libcudnn8",
		}

		result := FilterNVIDIAPackages(packages)

		assert.Len(t, result, 4)
		assert.Contains(t, result, "nvidia-driver-550")
		assert.Contains(t, result, "cuda-toolkit-12-4")
		assert.Contains(t, result, "libnvidia-gl-550")
		assert.Contains(t, result, "libcudnn8")
		assert.NotContains(t, result, "vim")
		assert.NotContains(t, result, "git")
		assert.NotContains(t, result, "python3")
	})

	t.Run("returns sorted list", func(t *testing.T) {
		packages := []string{
			"nvidia-settings",
			"cuda-toolkit-12-4",
			"nvidia-driver-550",
		}

		result := FilterNVIDIAPackages(packages)

		assert.Equal(t, []string{
			"cuda-toolkit-12-4",
			"nvidia-driver-550",
			"nvidia-settings",
		}, result)
	})

	t.Run("handles empty list", func(t *testing.T) {
		result := FilterNVIDIAPackages([]string{})

		assert.Empty(t, result)
	})

	t.Run("handles nil list", func(t *testing.T) {
		result := FilterNVIDIAPackages(nil)

		assert.Empty(t, result)
	})

	t.Run("handles no matching packages", func(t *testing.T) {
		packages := []string{"vim", "git", "emacs"}

		result := FilterNVIDIAPackages(packages)

		assert.Empty(t, result)
	})

	t.Run("case insensitive matching", func(t *testing.T) {
		packages := []string{
			"NVIDIA-driver-550",
			"Cuda-toolkit-12-4",
			"LibNVIDIA-gl-550",
		}

		result := FilterNVIDIAPackages(packages)

		assert.Len(t, result, 3)
	})

	t.Run("filters various nvidia package types", func(t *testing.T) {
		packages := []string{
			"kmod-nvidia",
			"akmod-nvidia",
			"dkms-nvidia",
			"xorg-x11-drv-nvidia",
			"x11-video-nvidia",
			"libnccl2",
			"cudnn-dev",
			"nccl",
		}

		result := FilterNVIDIAPackages(packages)

		assert.Len(t, result, 8)
	})
}

// =============================================================================
// CategorizePackages Tests
// =============================================================================

func TestCategorizePackages(t *testing.T) {
	t.Run("categorizes debian packages", func(t *testing.T) {
		packages := []string{
			"nvidia-driver-550",
			"nvidia-settings",
			"nvidia-dkms",
			"cuda-toolkit-12-4",
			"libcudnn8",
			"libnvidia-gl-550",
		}

		result := CategorizePackages(packages, constants.FamilyDebian)

		assert.Contains(t, result.DriverPackages, "nvidia-driver-550")
		assert.Contains(t, result.DriverPackages, "libnvidia-gl-550")
		assert.Contains(t, result.UtilityPackages, "nvidia-settings")
		assert.Contains(t, result.KernelModulePackages, "nvidia-dkms")
		assert.Contains(t, result.CUDAPackages, "cuda-toolkit-12-4")
		assert.Contains(t, result.LibraryPackages, "libcudnn8")
		assert.Equal(t, 6, result.TotalCount)
		assert.Equal(t, "550", result.DriverVersion)
		assert.Equal(t, "12.4", result.CUDAVersion)
	})

	t.Run("categorizes rhel packages", func(t *testing.T) {
		packages := []string{
			"nvidia-driver",
			"kmod-nvidia",
			"akmod-nvidia",
			"nvidia-settings",
			"cuda-12-4",
		}

		result := CategorizePackages(packages, constants.FamilyRHEL)

		assert.Contains(t, result.DriverPackages, "nvidia-driver")
		assert.Contains(t, result.KernelModulePackages, "kmod-nvidia")
		assert.Contains(t, result.KernelModulePackages, "akmod-nvidia")
		assert.Contains(t, result.UtilityPackages, "nvidia-settings")
		assert.Contains(t, result.CUDAPackages, "cuda-12-4")
	})

	t.Run("categorizes arch packages", func(t *testing.T) {
		packages := []string{
			"nvidia",
			"nvidia-utils",
			"nvidia-dkms",
			"nvidia-settings",
			"cuda",
			"cudnn",
		}

		result := CategorizePackages(packages, constants.FamilyArch)

		assert.Contains(t, result.DriverPackages, "nvidia")
		assert.Contains(t, result.UtilityPackages, "nvidia-utils")
		assert.Contains(t, result.UtilityPackages, "nvidia-settings")
		assert.Contains(t, result.KernelModulePackages, "nvidia-dkms")
		assert.Contains(t, result.CUDAPackages, "cuda")
		assert.Contains(t, result.LibraryPackages, "cudnn")
	})

	t.Run("categorizes suse packages", func(t *testing.T) {
		packages := []string{
			"nvidia-driver-G06-kmp-default",
			"nvidia-video-G06",
			"nvidia-settings",
			"nvidia-gfxG06-kmp-default",
		}

		result := CategorizePackages(packages, constants.FamilySUSE)

		assert.Contains(t, result.DriverPackages, "nvidia-driver-G06-kmp-default")
		assert.Contains(t, result.DriverPackages, "nvidia-video-G06")
		assert.Contains(t, result.UtilityPackages, "nvidia-settings")
		assert.Contains(t, result.KernelModulePackages, "nvidia-gfxG06-kmp-default")
	})

	t.Run("empty package list", func(t *testing.T) {
		result := CategorizePackages([]string{}, constants.FamilyDebian)

		assert.Empty(t, result.AllPackages)
		assert.Equal(t, 0, result.TotalCount)
		assert.Empty(t, result.DriverVersion)
		assert.Empty(t, result.CUDAVersion)
	})

	t.Run("unknown family uses default patterns", func(t *testing.T) {
		packages := []string{
			"nvidia-driver-550",
			"nvidia-settings",
		}

		result := CategorizePackages(packages, constants.FamilyUnknown)

		// Should still categorize based on default patterns
		assert.NotEmpty(t, result.AllPackages)
	})

	t.Run("all packages included in AllPackages", func(t *testing.T) {
		packages := []string{
			"nvidia-driver-550",
			"nvidia-settings",
			"cuda-toolkit-12-4",
		}

		result := CategorizePackages(packages, constants.FamilyDebian)

		assert.ElementsMatch(t, packages, result.AllPackages)
	})
}

// =============================================================================
// DiscoveredPackages Helper Method Tests
// =============================================================================

func TestDiscoveredPackages_IsEmpty(t *testing.T) {
	t.Run("empty when total count is zero", func(t *testing.T) {
		dp := &DiscoveredPackages{TotalCount: 0}
		assert.True(t, dp.IsEmpty())
	})

	t.Run("not empty when total count is positive", func(t *testing.T) {
		dp := &DiscoveredPackages{TotalCount: 5}
		assert.False(t, dp.IsEmpty())
	})
}

func TestDiscoveredPackages_HasDriver(t *testing.T) {
	t.Run("true when driver packages exist", func(t *testing.T) {
		dp := &DiscoveredPackages{
			DriverPackages: []string{"nvidia-driver-550"},
		}
		assert.True(t, dp.HasDriver())
	})

	t.Run("false when no driver packages", func(t *testing.T) {
		dp := &DiscoveredPackages{
			DriverPackages: []string{},
		}
		assert.False(t, dp.HasDriver())
	})

	t.Run("false when nil driver packages", func(t *testing.T) {
		dp := &DiscoveredPackages{}
		assert.False(t, dp.HasDriver())
	})
}

func TestDiscoveredPackages_HasCUDA(t *testing.T) {
	t.Run("true when cuda packages exist", func(t *testing.T) {
		dp := &DiscoveredPackages{
			CUDAPackages: []string{"cuda-toolkit-12-4"},
		}
		assert.True(t, dp.HasCUDA())
	})

	t.Run("false when no cuda packages", func(t *testing.T) {
		dp := &DiscoveredPackages{
			CUDAPackages: []string{},
		}
		assert.False(t, dp.HasCUDA())
	})

	t.Run("false when nil cuda packages", func(t *testing.T) {
		dp := &DiscoveredPackages{}
		assert.False(t, dp.HasCUDA())
	})
}

// =============================================================================
// Interface Compliance Tests
// =============================================================================

func TestPackageDiscovery_ImplementsDiscovery(t *testing.T) {
	// Compile-time check
	var _ Discovery = (*PackageDiscovery)(nil)
}

// =============================================================================
// Edge Cases and Regression Tests
// =============================================================================

func TestPackageDiscovery_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	pm := NewMockPackageManager()
	pm.SetInstalledPackages([]pkg.Package{
		{Name: "nvidia-driver-550", Installed: true},
	})
	pd := NewPackageDiscovery(pm)

	// Our mock doesn't check context, but real implementations should
	// This test verifies the interface accepts context correctly
	result, err := pd.Discover(ctx)

	// Mock doesn't honor context cancellation, so this should still work
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCategorizePackages_ConfigPackages(t *testing.T) {
	packages := []string{
		"nvidia-prime-config",
		"nvidia-xconfig",
	}

	result := CategorizePackages(packages, constants.FamilyDebian)

	// nvidia-xconfig is a utility on Debian (via pattern matching)
	// But the config pattern might match nvidia-prime-config
	assert.NotEmpty(t, result.AllPackages)
}

func TestFilterNVIDIAPackages_LargeList(t *testing.T) {
	// Test with a large list to ensure no performance issues
	packages := make([]string, 1000)
	for i := 0; i < 500; i++ {
		packages[i*2] = "nvidia-package-" + string(rune('a'+i%26))
		packages[i*2+1] = "other-package-" + string(rune('a'+i%26))
	}

	result := FilterNVIDIAPackages(packages)

	assert.Len(t, result, 500) // Only nvidia packages
}
