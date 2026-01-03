package nvidia

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tungetti/igor/internal/constants"
	"github.com/tungetti/igor/internal/distro"
)

func TestComponent_String(t *testing.T) {
	tests := []struct {
		component Component
		expected  string
	}{
		{ComponentDriver, "driver"},
		{ComponentDriverDKMS, "driver-dkms"},
		{ComponentCUDA, "cuda"},
		{ComponentCUDNN, "cudnn"},
		{ComponentNVCC, "nvcc"},
		{ComponentUtils, "utils"},
		{ComponentSettings, "settings"},
		{ComponentOpenCL, "opencl"},
		{ComponentVulkan, "vulkan"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.component.String())
		})
	}
}

func TestComponent_IsValid(t *testing.T) {
	tests := []struct {
		component Component
		valid     bool
	}{
		{ComponentDriver, true},
		{ComponentDriverDKMS, true},
		{ComponentCUDA, true},
		{ComponentCUDNN, true},
		{ComponentNVCC, true},
		{ComponentUtils, true},
		{ComponentSettings, true},
		{ComponentOpenCL, true},
		{ComponentVulkan, true},
		{Component("invalid"), false},
		{Component(""), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.component), func(t *testing.T) {
			assert.Equal(t, tt.valid, tt.component.IsValid())
		})
	}
}

func TestAllComponents(t *testing.T) {
	components := AllComponents()
	assert.Len(t, components, 9)

	// Verify all components are valid
	for _, c := range components {
		assert.True(t, c.IsValid(), "component %s should be valid", c)
	}
}

func TestPackageSet_GetPackages(t *testing.T) {
	ps := &PackageSet{
		Family:       constants.FamilyDebian,
		Distribution: "ubuntu",
		Driver:       []string{"nvidia-driver-550"},
		DriverDKMS:   []string{"nvidia-dkms-550"},
		Utils:        []string{"nvidia-utils-550"},
		Settings:     []string{"nvidia-settings"},
		CUDA:         []string{"nvidia-cuda-toolkit"},
		CUDACompiler: []string{"nvidia-cuda-toolkit"},
		CUDnn:        []string{"libcudnn8"},
		OpenCL:       []string{"nvidia-opencl-icd"},
		Vulkan:       []string{"nvidia-vulkan-icd"},
	}

	tests := []struct {
		component Component
		expected  []string
	}{
		{ComponentDriver, []string{"nvidia-driver-550"}},
		{ComponentDriverDKMS, []string{"nvidia-dkms-550"}},
		{ComponentUtils, []string{"nvidia-utils-550"}},
		{ComponentSettings, []string{"nvidia-settings"}},
		{ComponentCUDA, []string{"nvidia-cuda-toolkit"}},
		{ComponentNVCC, []string{"nvidia-cuda-toolkit"}},
		{ComponentCUDNN, []string{"libcudnn8"}},
		{ComponentOpenCL, []string{"nvidia-opencl-icd"}},
		{ComponentVulkan, []string{"nvidia-vulkan-icd"}},
		{Component("invalid"), nil},
	}

	for _, tt := range tests {
		t.Run(string(tt.component), func(t *testing.T) {
			packages := ps.GetPackages(tt.component)
			assert.Equal(t, tt.expected, packages)
		})
	}
}

func TestPackageSet_GetPackagesForVersion(t *testing.T) {
	ps := &PackageSet{
		Family:               constants.FamilyDebian,
		Driver:               []string{"nvidia-driver-550"},
		DriverVersionPattern: "nvidia-driver-%s",
	}

	tests := []struct {
		name     string
		version  string
		expected []string
	}{
		{"specific version", "535", []string{"nvidia-driver-535"}},
		{"another version", "550", []string{"nvidia-driver-550"}},
		{"empty version", "", []string{"nvidia-driver-550"}},
		{"version with spaces", "  545  ", []string{"nvidia-driver-545"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			packages := ps.GetPackagesForVersion(tt.version)
			assert.Equal(t, tt.expected, packages)
		})
	}
}

func TestPackageSet_GetPackagesForVersion_NoPattern(t *testing.T) {
	ps := &PackageSet{
		Family: constants.FamilyArch,
		Driver: []string{"nvidia"},
	}

	// Without a pattern, should return default driver packages
	packages := ps.GetPackagesForVersion("550")
	assert.Equal(t, []string{"nvidia"}, packages)
}

func TestPackageSet_GetAllPackages(t *testing.T) {
	ps := &PackageSet{
		Family:       constants.FamilyDebian,
		Driver:       []string{"nvidia-driver-550"},
		DriverDKMS:   []string{"nvidia-dkms-550"},
		Utils:        []string{"nvidia-utils-550"},
		Settings:     []string{"nvidia-settings"},
		CUDA:         []string{"nvidia-cuda-toolkit"},
		CUDACompiler: []string{"nvidia-cuda-toolkit"}, // Duplicate, should be deduplicated
		CUDALibs:     []string{"nvidia-cuda-dev"},
		CUDnn:        []string{"libcudnn8", "libcudnn8-dev"},
		OpenCL:       []string{"nvidia-opencl-icd"},
		Vulkan:       []string{"nvidia-vulkan-icd"},
	}

	packages := ps.GetAllPackages()

	// Should contain all unique packages
	assert.Contains(t, packages, "nvidia-driver-550")
	assert.Contains(t, packages, "nvidia-utils-550")
	assert.Contains(t, packages, "nvidia-settings")
	assert.Contains(t, packages, "nvidia-cuda-toolkit")
	assert.Contains(t, packages, "nvidia-cuda-dev")
	assert.Contains(t, packages, "libcudnn8")
	assert.Contains(t, packages, "libcudnn8-dev")
	assert.Contains(t, packages, "nvidia-opencl-icd")
	assert.Contains(t, packages, "nvidia-vulkan-icd")

	// Verify no duplicates
	seen := make(map[string]bool)
	for _, p := range packages {
		assert.False(t, seen[p], "package %s is duplicated", p)
		seen[p] = true
	}
}

func TestPackageSet_GetMinimalPackages(t *testing.T) {
	ps := &PackageSet{
		Family:   constants.FamilyDebian,
		Driver:   []string{"nvidia-driver-550"},
		Utils:    []string{"nvidia-utils-550"},
		Settings: []string{"nvidia-settings"},
		CUDA:     []string{"nvidia-cuda-toolkit"},
	}

	packages := ps.GetMinimalPackages()

	// Should only contain driver and utils
	assert.Contains(t, packages, "nvidia-driver-550")
	assert.Contains(t, packages, "nvidia-utils-550")
	assert.NotContains(t, packages, "nvidia-settings")
	assert.NotContains(t, packages, "nvidia-cuda-toolkit")
}

func TestPackageSet_GetDevelopmentPackages(t *testing.T) {
	ps := &PackageSet{
		Family:       constants.FamilyDebian,
		Driver:       []string{"nvidia-driver-550"},
		Utils:        []string{"nvidia-utils-550"},
		CUDA:         []string{"nvidia-cuda-toolkit"},
		CUDACompiler: []string{"nvcc"},
		CUDALibs:     []string{"cuda-libs"},
		CUDnn:        []string{"libcudnn8"},
	}

	packages := ps.GetDevelopmentPackages()

	// Should contain CUDA-related packages only
	assert.Contains(t, packages, "nvidia-cuda-toolkit")
	assert.Contains(t, packages, "nvcc")
	assert.Contains(t, packages, "cuda-libs")
	assert.Contains(t, packages, "libcudnn8")
	assert.NotContains(t, packages, "nvidia-driver-550")
	assert.NotContains(t, packages, "nvidia-utils-550")
}

func TestPackageSet_GetGraphicsPackages(t *testing.T) {
	ps := &PackageSet{
		Family:   constants.FamilyDebian,
		Driver:   []string{"nvidia-driver-550"},
		Utils:    []string{"nvidia-utils-550"},
		Settings: []string{"nvidia-settings"},
		CUDA:     []string{"nvidia-cuda-toolkit"},
		OpenCL:   []string{"nvidia-opencl-icd"},
		Vulkan:   []string{"nvidia-vulkan-icd"},
	}

	packages := ps.GetGraphicsPackages()

	// Should contain graphics-related packages
	assert.Contains(t, packages, "nvidia-driver-550")
	assert.Contains(t, packages, "nvidia-utils-550")
	assert.Contains(t, packages, "nvidia-settings")
	assert.Contains(t, packages, "nvidia-vulkan-icd")
	assert.Contains(t, packages, "nvidia-opencl-icd")
	assert.NotContains(t, packages, "nvidia-cuda-toolkit")
}

func TestGetPackageSetForFamily(t *testing.T) {
	tests := []struct {
		family   constants.DistroFamily
		notNil   bool
		contains string
	}{
		{constants.FamilyDebian, true, "nvidia-driver"},
		{constants.FamilyRHEL, true, "akmod-nvidia"},
		{constants.FamilyArch, true, "nvidia"},
		{constants.FamilySUSE, true, "nvidia-driver-G06"},
		{constants.FamilyUnknown, false, ""},
	}

	for _, tt := range tests {
		t.Run(string(tt.family), func(t *testing.T) {
			ps := GetPackageSetForFamily(tt.family)

			if tt.notNil {
				require.NotNil(t, ps)
				assert.NotEmpty(t, ps.Driver)

				// Check that at least one driver package contains the expected string
				found := false
				for _, pkg := range ps.Driver {
					if strings.Contains(pkg, tt.contains) {
						found = true
						break
					}
				}
				assert.True(t, found, "expected driver package containing %s", tt.contains)
			} else {
				assert.Nil(t, ps)
			}
		})
	}
}

func TestGetPackageSet(t *testing.T) {
	tests := []struct {
		name     string
		dist     *distro.Distribution
		expected string // Expected package name substring
	}{
		{
			name: "ubuntu",
			dist: &distro.Distribution{
				ID:     "ubuntu",
				Family: constants.FamilyDebian,
			},
			expected: "nvidia-driver",
		},
		{
			name: "pop_os",
			dist: &distro.Distribution{
				ID:     "pop",
				Family: constants.FamilyDebian,
			},
			expected: "system76-driver",
		},
		{
			name: "fedora",
			dist: &distro.Distribution{
				ID:     "fedora",
				Family: constants.FamilyRHEL,
			},
			expected: "akmod-nvidia",
		},
		{
			name: "arch",
			dist: &distro.Distribution{
				ID:     "arch",
				Family: constants.FamilyArch,
			},
			expected: "nvidia",
		},
		{
			name: "manjaro",
			dist: &distro.Distribution{
				ID:     "manjaro",
				Family: constants.FamilyArch,
			},
			expected: "nvidia",
		},
		{
			name: "opensuse_tumbleweed",
			dist: &distro.Distribution{
				ID:     "opensuse-tumbleweed",
				Family: constants.FamilySUSE,
			},
			expected: "nvidia-driver-G06",
		},
		{
			name: "opensuse_leap",
			dist: &distro.Distribution{
				ID:     "opensuse-leap",
				Family: constants.FamilySUSE,
			},
			expected: "nvidia-driver-G06",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ps := GetPackageSet(tt.dist)
			require.NotNil(t, ps)
			assert.NotEmpty(t, ps.Driver)

			// Check that at least one driver package contains the expected string
			found := false
			for _, pkg := range ps.Driver {
				if strings.Contains(pkg, tt.expected) {
					found = true
					break
				}
			}
			assert.True(t, found, "expected driver package containing %s, got %v", tt.expected, ps.Driver)
		})
	}
}

func TestGetPackageSet_NilDistribution(t *testing.T) {
	ps := GetPackageSet(nil)
	assert.Nil(t, ps)
}

func TestGetPackageSetByID(t *testing.T) {
	tests := []struct {
		distroID string
		notNil   bool
	}{
		{"ubuntu", true},
		{"pop", true},
		{"fedora", true},
		{"manjaro", true},
		{"opensuse-tumbleweed", true},
		{"opensuse-leap", true},
		{"unknown", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.distroID, func(t *testing.T) {
			ps := GetPackageSetByID(tt.distroID)
			if tt.notNil {
				assert.NotNil(t, ps)
			} else {
				assert.Nil(t, ps)
			}
		})
	}
}

func TestIsSupported(t *testing.T) {
	tests := []struct {
		family    constants.DistroFamily
		supported bool
	}{
		{constants.FamilyDebian, true},
		{constants.FamilyRHEL, true},
		{constants.FamilyArch, true},
		{constants.FamilySUSE, true},
		{constants.FamilyUnknown, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.family), func(t *testing.T) {
			assert.Equal(t, tt.supported, IsSupported(tt.family))
		})
	}
}

func TestSupportedFamilies(t *testing.T) {
	families := SupportedFamilies()
	assert.Len(t, families, 4)
	assert.Contains(t, families, constants.FamilyDebian)
	assert.Contains(t, families, constants.FamilyRHEL)
	assert.Contains(t, families, constants.FamilyArch)
	assert.Contains(t, families, constants.FamilySUSE)
}

func TestSupportedDriverVersions(t *testing.T) {
	assert.NotEmpty(t, SupportedDriverVersions)
	assert.Contains(t, SupportedDriverVersions, "550")
	assert.Contains(t, SupportedDriverVersions, "535")
}

func TestGetRecommendedDriverVersion(t *testing.T) {
	version := GetRecommendedDriverVersion()
	assert.NotEmpty(t, version)
	assert.Equal(t, SupportedDriverVersions[0], version)
}

func TestGetLTSDriverVersion(t *testing.T) {
	version := GetLTSDriverVersion()
	assert.Equal(t, "535", version)
}

func TestGetLegacyDriverVersion(t *testing.T) {
	version := GetLegacyDriverVersion()
	assert.Equal(t, "470", version)
}

func TestDebian_PackageSet_VersionPattern(t *testing.T) {
	ps := GetPackageSetForFamily(constants.FamilyDebian)
	require.NotNil(t, ps)
	assert.NotEmpty(t, ps.DriverVersionPattern)

	// Test version pattern
	packages := ps.GetPackagesForVersion("535")
	assert.Equal(t, []string{"nvidia-driver-535"}, packages)
}

func TestRHEL_PackageSet_NoVersionPattern(t *testing.T) {
	ps := GetPackageSetForFamily(constants.FamilyRHEL)
	require.NotNil(t, ps)

	// RHEL uses akmod-nvidia which doesn't have version pattern
	packages := ps.GetPackagesForVersion("535")
	assert.Equal(t, ps.Driver, packages)
}

func TestArch_PackageSet_NoVersionPattern(t *testing.T) {
	ps := GetPackageSetForFamily(constants.FamilyArch)
	require.NotNil(t, ps)

	// Arch uses simple "nvidia" package
	packages := ps.GetPackagesForVersion("535")
	assert.Equal(t, ps.Driver, packages)
}

func TestPackageSet_HasAllComponents(t *testing.T) {
	families := []constants.DistroFamily{
		constants.FamilyDebian,
		constants.FamilyRHEL,
		constants.FamilyArch,
		constants.FamilySUSE,
	}

	for _, family := range families {
		t.Run(string(family), func(t *testing.T) {
			ps := GetPackageSetForFamily(family)
			require.NotNil(t, ps)

			// Every family should have at least these components
			assert.NotEmpty(t, ps.Driver, "Driver packages should not be empty")
			assert.NotEmpty(t, ps.Utils, "Utils packages should not be empty")
			assert.NotEmpty(t, ps.CUDA, "CUDA packages should not be empty")
		})
	}
}

func TestPackageSet_NoDuplicatesInGetAllPackages(t *testing.T) {
	families := []constants.DistroFamily{
		constants.FamilyDebian,
		constants.FamilyRHEL,
		constants.FamilyArch,
		constants.FamilySUSE,
	}

	for _, family := range families {
		t.Run(string(family), func(t *testing.T) {
			ps := GetPackageSetForFamily(family)
			require.NotNil(t, ps)

			packages := ps.GetAllPackages()
			seen := make(map[string]bool)

			for _, pkg := range packages {
				assert.False(t, seen[pkg], "duplicate package found: %s", pkg)
				seen[pkg] = true
			}
		})
	}
}

func TestGetDKMSPackagesForVersion(t *testing.T) {
	ps := &PackageSet{
		Family:             constants.FamilyDebian,
		DriverDKMS:         []string{"nvidia-dkms-550"},
		DKMSVersionPattern: "nvidia-dkms-%s",
	}

	tests := []struct {
		name     string
		version  string
		expected []string
	}{
		{"specific version", "535", []string{"nvidia-dkms-535"}},
		{"empty version", "", []string{"nvidia-dkms-550"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			packages := ps.GetDKMSPackagesForVersion(tt.version)
			assert.Equal(t, tt.expected, packages)
		})
	}
}

func TestGetDKMSPackagesForVersion_NoPattern(t *testing.T) {
	ps := &PackageSet{
		Family:     constants.FamilyArch,
		DriverDKMS: []string{"nvidia-dkms"},
	}

	packages := ps.GetDKMSPackagesForVersion("550")
	assert.Equal(t, []string{"nvidia-dkms"}, packages)
}

func TestGetPackageSet_FallbackToFamily(t *testing.T) {
	// Test that an unknown distro ID falls back to family-level package set
	dist := &distro.Distribution{
		ID:     "unknown-debian-derivative",
		Family: constants.FamilyDebian,
	}

	ps := GetPackageSet(dist)
	require.NotNil(t, ps)
	assert.Equal(t, constants.FamilyDebian, ps.Family)
}

func TestGetPackageSetByID_CaseInsensitive(t *testing.T) {
	// Test case insensitivity
	ps := GetPackageSetByID("UBUNTU")
	require.NotNil(t, ps)
	assert.Equal(t, "ubuntu", ps.Distribution)

	ps = GetPackageSetByID("Fedora")
	require.NotNil(t, ps)
	assert.Equal(t, "fedora", ps.Distribution)
}

func TestPackageSet_EmptyPackages(t *testing.T) {
	ps := &PackageSet{
		Family: constants.FamilyDebian,
	}

	// All methods should handle empty slices gracefully
	assert.Empty(t, ps.GetPackages(ComponentDriver))
	assert.Empty(t, ps.GetAllPackages())
	assert.Empty(t, ps.GetMinimalPackages())
	assert.Empty(t, ps.GetDevelopmentPackages())
	assert.Empty(t, ps.GetGraphicsPackages())
}

func TestPackageSet_EmptyStringsFiltered(t *testing.T) {
	ps := &PackageSet{
		Family:   constants.FamilyDebian,
		Driver:   []string{"nvidia-driver", "", "nvidia-driver-550"},
		Utils:    []string{"", "nvidia-utils"},
		Settings: []string{""},
	}

	packages := ps.GetAllPackages()

	// Empty strings should not be included
	for _, p := range packages {
		assert.NotEmpty(t, p, "empty string should not be in packages")
	}
}

func TestSupportedDriverVersions_Order(t *testing.T) {
	// Verify versions are ordered from newest to oldest
	versions := SupportedDriverVersions
	require.GreaterOrEqual(t, len(versions), 2)

	// First version should be the recommended one
	assert.Equal(t, GetRecommendedDriverVersion(), versions[0])
}
