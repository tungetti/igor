// Package nvidia provides NVIDIA-specific package mappings and repository definitions
// for different Linux distributions. It maps NVIDIA software components (drivers, CUDA,
// cuDNN, etc.) to the actual package names used by each distribution's package manager.
package nvidia

import (
	"fmt"
	"strings"

	"github.com/tungetti/igor/internal/constants"
	"github.com/tungetti/igor/internal/distro"
)

// Component represents an NVIDIA software component that can be installed.
type Component string

const (
	// ComponentDriver is the main NVIDIA GPU driver.
	ComponentDriver Component = "driver"
	// ComponentDriverDKMS is the DKMS driver that builds for any kernel version.
	ComponentDriverDKMS Component = "driver-dkms"
	// ComponentCUDA is the CUDA Toolkit.
	ComponentCUDA Component = "cuda"
	// ComponentCUDNN is the cuDNN library for deep learning.
	ComponentCUDNN Component = "cudnn"
	// ComponentNVCC is the CUDA compiler.
	ComponentNVCC Component = "nvcc"
	// ComponentUtils contains nvidia-utils, nvidia-smi, and other utilities.
	ComponentUtils Component = "utils"
	// ComponentSettings is the nvidia-settings GUI application.
	ComponentSettings Component = "settings"
	// ComponentOpenCL provides OpenCL support.
	ComponentOpenCL Component = "opencl"
	// ComponentVulkan provides the Vulkan ICD.
	ComponentVulkan Component = "vulkan"
)

// AllComponents returns a slice of all available NVIDIA components.
func AllComponents() []Component {
	return []Component{
		ComponentDriver,
		ComponentDriverDKMS,
		ComponentCUDA,
		ComponentCUDNN,
		ComponentNVCC,
		ComponentUtils,
		ComponentSettings,
		ComponentOpenCL,
		ComponentVulkan,
	}
}

// String returns the string representation of the component.
func (c Component) String() string {
	return string(c)
}

// IsValid checks if the component is a valid NVIDIA component.
func (c Component) IsValid() bool {
	switch c {
	case ComponentDriver, ComponentDriverDKMS, ComponentCUDA, ComponentCUDNN,
		ComponentNVCC, ComponentUtils, ComponentSettings, ComponentOpenCL, ComponentVulkan:
		return true
	default:
		return false
	}
}

// PackageSet contains the package names for all NVIDIA components on a distribution.
// Each field contains a slice of package names because some components may require
// multiple packages, or different versions may be available.
type PackageSet struct {
	// Family is the distribution family (Debian, RHEL, Arch, SUSE).
	Family constants.DistroFamily

	// Distribution is the specific distro ID (e.g., "ubuntu", "fedora", "arch").
	Distribution string

	// Driver contains the main driver packages.
	Driver []string

	// DriverDKMS contains the DKMS driver packages.
	DriverDKMS []string

	// Utils contains utility packages (nvidia-smi, etc.).
	Utils []string

	// Settings contains the nvidia-settings GUI packages.
	Settings []string

	// CUDA contains CUDA toolkit packages.
	CUDA []string

	// CUDACompiler contains NVCC compiler packages.
	CUDACompiler []string

	// CUDALibs contains CUDA library packages.
	CUDALibs []string

	// CUDnn contains cuDNN packages.
	CUDnn []string

	// OpenCL contains OpenCL packages.
	OpenCL []string

	// Vulkan contains Vulkan ICD packages.
	Vulkan []string

	// DriverVersionPattern is a format string for version-specific driver packages.
	// For example, "nvidia-driver-%s" where %s is replaced with the version.
	DriverVersionPattern string

	// DKMSVersionPattern is a format string for version-specific DKMS packages.
	DKMSVersionPattern string

	// Notes contains additional information about the package set.
	Notes string
}

// SupportedDriverVersions contains the currently supported NVIDIA driver versions.
// These are the versions that Igor will offer for installation.
var SupportedDriverVersions = []string{
	"550", // Latest production branch
	"545", // Previous production branch
	"535", // Long-term support branch
	"525", // Older LTS
	"470", // Legacy (for older GPUs)
}

// GetPackages returns the package names for a specific component.
func (ps *PackageSet) GetPackages(component Component) []string {
	switch component {
	case ComponentDriver:
		return ps.Driver
	case ComponentDriverDKMS:
		return ps.DriverDKMS
	case ComponentCUDA:
		return ps.CUDA
	case ComponentCUDNN:
		return ps.CUDnn
	case ComponentNVCC:
		return ps.CUDACompiler
	case ComponentUtils:
		return ps.Utils
	case ComponentSettings:
		return ps.Settings
	case ComponentOpenCL:
		return ps.OpenCL
	case ComponentVulkan:
		return ps.Vulkan
	default:
		return nil
	}
}

// GetPackagesForVersion returns driver packages for a specific version.
// If the PackageSet has a DriverVersionPattern, it uses that to generate
// version-specific package names. Otherwise, it returns the default Driver packages.
func (ps *PackageSet) GetPackagesForVersion(version string) []string {
	if ps.DriverVersionPattern == "" || version == "" {
		return ps.Driver
	}

	// Clean the version string
	version = strings.TrimSpace(version)

	return []string{fmt.Sprintf(ps.DriverVersionPattern, version)}
}

// GetDKMSPackagesForVersion returns DKMS driver packages for a specific version.
func (ps *PackageSet) GetDKMSPackagesForVersion(version string) []string {
	if ps.DKMSVersionPattern == "" || version == "" {
		return ps.DriverDKMS
	}

	version = strings.TrimSpace(version)
	return []string{fmt.Sprintf(ps.DKMSVersionPattern, version)}
}

// GetAllPackages returns all packages needed for a full NVIDIA installation.
// This includes the driver, utilities, settings, CUDA, and all supporting packages.
func (ps *PackageSet) GetAllPackages() []string {
	var packages []string
	seen := make(map[string]bool)

	// Helper to add unique packages
	addPackages := func(pkgs []string) {
		for _, p := range pkgs {
			if p != "" && !seen[p] {
				seen[p] = true
				packages = append(packages, p)
			}
		}
	}

	// Add packages in order of importance
	addPackages(ps.Driver)
	addPackages(ps.Utils)
	addPackages(ps.Settings)
	addPackages(ps.CUDA)
	addPackages(ps.CUDACompiler)
	addPackages(ps.CUDALibs)
	addPackages(ps.CUDnn)
	addPackages(ps.OpenCL)
	addPackages(ps.Vulkan)

	return packages
}

// GetMinimalPackages returns packages for a minimal driver-only installation.
// This includes just the driver and essential utilities.
func (ps *PackageSet) GetMinimalPackages() []string {
	var packages []string
	seen := make(map[string]bool)

	addPackages := func(pkgs []string) {
		for _, p := range pkgs {
			if p != "" && !seen[p] {
				seen[p] = true
				packages = append(packages, p)
			}
		}
	}

	// For minimal installation: driver + utils only
	addPackages(ps.Driver)
	addPackages(ps.Utils)

	return packages
}

// GetDevelopmentPackages returns packages for CUDA development.
// This includes CUDA toolkit, compiler, and development libraries.
func (ps *PackageSet) GetDevelopmentPackages() []string {
	var packages []string
	seen := make(map[string]bool)

	addPackages := func(pkgs []string) {
		for _, p := range pkgs {
			if p != "" && !seen[p] {
				seen[p] = true
				packages = append(packages, p)
			}
		}
	}

	// CUDA development packages
	addPackages(ps.CUDA)
	addPackages(ps.CUDACompiler)
	addPackages(ps.CUDALibs)
	addPackages(ps.CUDnn)

	return packages
}

// GetGraphicsPackages returns packages for graphics/gaming use.
// This includes driver, Vulkan, and OpenCL support.
func (ps *PackageSet) GetGraphicsPackages() []string {
	var packages []string
	seen := make(map[string]bool)

	addPackages := func(pkgs []string) {
		for _, p := range pkgs {
			if p != "" && !seen[p] {
				seen[p] = true
				packages = append(packages, p)
			}
		}
	}

	addPackages(ps.Driver)
	addPackages(ps.Utils)
	addPackages(ps.Settings)
	addPackages(ps.Vulkan)
	addPackages(ps.OpenCL)

	return packages
}

// packageSets holds the package mappings for each distribution family.
var packageSets = map[constants.DistroFamily]*PackageSet{
	constants.FamilyDebian: {
		Family:       constants.FamilyDebian,
		Distribution: "debian/ubuntu",
		Driver: []string{
			"nvidia-driver-550",
			"nvidia-driver-545",
			"nvidia-driver-535",
		},
		DriverDKMS: []string{
			"nvidia-dkms-550",
			"nvidia-dkms-545",
			"nvidia-dkms-535",
		},
		Utils:    []string{"nvidia-utils-550"},
		Settings: []string{"nvidia-settings"},
		CUDA: []string{
			"nvidia-cuda-toolkit",
		},
		CUDACompiler: []string{
			"nvidia-cuda-toolkit",
		},
		CUDALibs: []string{
			"nvidia-cuda-dev",
		},
		CUDnn: []string{
			"libcudnn8",
			"libcudnn8-dev",
		},
		OpenCL: []string{
			"nvidia-opencl-icd",
		},
		Vulkan: []string{
			"nvidia-vulkan-icd",
		},
		DriverVersionPattern: "nvidia-driver-%s",
		DKMSVersionPattern:   "nvidia-dkms-%s",
		Notes:                "Ubuntu/Debian use the graphics-drivers PPA or official CUDA repository",
	},

	constants.FamilyRHEL: {
		Family:       constants.FamilyRHEL,
		Distribution: "fedora/rhel",
		Driver: []string{
			"akmod-nvidia",
			"xorg-x11-drv-nvidia",
		},
		DriverDKMS: []string{
			"akmod-nvidia",
		},
		Utils: []string{
			"nvidia-settings",
			"xorg-x11-drv-nvidia-libs",
			"xorg-x11-drv-nvidia-libs.i686", // 32-bit support
		},
		Settings: []string{
			"nvidia-settings",
		},
		CUDA: []string{
			"cuda",
		},
		CUDACompiler: []string{
			"cuda-compiler",
		},
		CUDALibs: []string{
			"cuda-libs",
			"cuda-devel",
		},
		CUDnn: []string{
			"cudnn",
		},
		OpenCL: []string{
			"nvidia-driver-cuda",
			"xorg-x11-drv-nvidia-cuda-libs",
		},
		Vulkan: []string{
			"vulkan-loader",
			"xorg-x11-drv-nvidia-vulkan",
		},
		Notes: "Requires RPM Fusion nonfree repository",
	},

	constants.FamilyArch: {
		Family:       constants.FamilyArch,
		Distribution: "arch",
		Driver: []string{
			"nvidia",
		},
		DriverDKMS: []string{
			"nvidia-dkms",
		},
		Utils: []string{
			"nvidia-utils",
			"lib32-nvidia-utils", // 32-bit support
		},
		Settings: []string{
			"nvidia-settings",
		},
		CUDA: []string{
			"cuda",
		},
		CUDACompiler: []string{
			"cuda",
		},
		CUDALibs: []string{
			"cuda",
		},
		CUDnn: []string{
			"cudnn",
		},
		OpenCL: []string{
			"opencl-nvidia",
		},
		Vulkan: []string{
			"nvidia-utils", // Vulkan ICD is included in nvidia-utils
		},
		Notes: "Uses official Arch repositories, no extra repository needed",
	},

	constants.FamilySUSE: {
		Family:       constants.FamilySUSE,
		Distribution: "opensuse",
		Driver: []string{
			"nvidia-driver-G06-kmp-default",
		},
		DriverDKMS: []string{
			"nvidia-driver-G06-kmp-default",
		},
		Utils: []string{
			"nvidia-driver-G06",
		},
		Settings: []string{
			"nvidia-settings",
		},
		CUDA: []string{
			"cuda",
		},
		CUDACompiler: []string{
			"cuda",
		},
		CUDALibs: []string{
			"cuda-devel",
		},
		CUDnn: []string{
			"libcudnn8",
			"libcudnn8-devel",
		},
		OpenCL: []string{
			"nvidia-driver-G06",
		},
		Vulkan: []string{
			"nvidia-driver-G06",
		},
		Notes: "Uses official NVIDIA openSUSE repository",
	},
}

// distroSpecificPackageSets contains overrides for specific distributions.
var distroSpecificPackageSets = map[string]*PackageSet{
	"ubuntu": {
		Family:       constants.FamilyDebian,
		Distribution: "ubuntu",
		Driver: []string{
			"nvidia-driver-550",
			"nvidia-driver-545",
			"nvidia-driver-535",
		},
		DriverDKMS: []string{
			"nvidia-dkms-550",
			"nvidia-dkms-545",
			"nvidia-dkms-535",
		},
		Utils: []string{
			"nvidia-utils-550",
		},
		Settings: []string{
			"nvidia-settings",
		},
		CUDA: []string{
			"nvidia-cuda-toolkit",
		},
		CUDACompiler: []string{
			"nvidia-cuda-toolkit",
		},
		CUDALibs: []string{
			"nvidia-cuda-dev",
			"libcublas-dev",
		},
		CUDnn: []string{
			"libcudnn8",
			"libcudnn8-dev",
		},
		OpenCL: []string{
			"nvidia-opencl-icd",
		},
		Vulkan: []string{
			"nvidia-vulkan-icd",
		},
		DriverVersionPattern: "nvidia-driver-%s",
		DKMSVersionPattern:   "nvidia-dkms-%s",
		Notes:                "Ubuntu uses the graphics-drivers PPA or official NVIDIA CUDA repository",
	},

	"pop": {
		Family:       constants.FamilyDebian,
		Distribution: "pop",
		Driver: []string{
			"system76-driver-nvidia",
		},
		DriverDKMS: []string{
			"system76-driver-nvidia",
		},
		Utils: []string{
			"nvidia-utils-550",
		},
		Settings: []string{
			"nvidia-settings",
		},
		CUDA: []string{
			"nvidia-cuda-toolkit",
		},
		CUDACompiler: []string{
			"nvidia-cuda-toolkit",
		},
		CUDALibs: []string{
			"nvidia-cuda-dev",
		},
		CUDnn: []string{
			"libcudnn8",
			"libcudnn8-dev",
		},
		OpenCL: []string{
			"nvidia-opencl-icd",
		},
		Vulkan: []string{
			"nvidia-vulkan-icd",
		},
		Notes: "Pop!_OS uses System76's driver package",
	},

	"fedora": {
		Family:       constants.FamilyRHEL,
		Distribution: "fedora",
		Driver: []string{
			"akmod-nvidia",
			"xorg-x11-drv-nvidia",
		},
		DriverDKMS: []string{
			"akmod-nvidia",
		},
		Utils: []string{
			"nvidia-settings",
			"xorg-x11-drv-nvidia-libs",
			"xorg-x11-drv-nvidia-libs.i686",
		},
		Settings: []string{
			"nvidia-settings",
		},
		CUDA: []string{
			"cuda",
			"xorg-x11-drv-nvidia-cuda",
		},
		CUDACompiler: []string{
			"cuda-compiler",
		},
		CUDALibs: []string{
			"cuda-libs",
			"cuda-devel",
			"xorg-x11-drv-nvidia-cuda-libs",
		},
		CUDnn: []string{
			"cudnn",
		},
		OpenCL: []string{
			"xorg-x11-drv-nvidia-cuda-libs",
		},
		Vulkan: []string{
			"vulkan-loader",
			"xorg-x11-drv-nvidia-vulkan",
		},
		Notes: "Fedora requires RPM Fusion nonfree repository",
	},

	"opensuse-tumbleweed": {
		Family:       constants.FamilySUSE,
		Distribution: "opensuse-tumbleweed",
		Driver: []string{
			"nvidia-driver-G06-kmp-default",
		},
		DriverDKMS: []string{
			"nvidia-driver-G06-kmp-default",
		},
		Utils: []string{
			"nvidia-driver-G06",
			"nvidia-compute-utils-G06",
		},
		Settings: []string{
			"nvidia-settings",
		},
		CUDA: []string{
			"cuda",
		},
		CUDACompiler: []string{
			"cuda",
		},
		CUDALibs: []string{
			"cuda-devel",
		},
		CUDnn: []string{
			"libcudnn8",
		},
		OpenCL: []string{
			"nvidia-gl-G06",
		},
		Vulkan: []string{
			"nvidia-gl-G06",
		},
		Notes: "openSUSE Tumbleweed uses the official NVIDIA repository",
	},

	"opensuse-leap": {
		Family:       constants.FamilySUSE,
		Distribution: "opensuse-leap",
		Driver: []string{
			"nvidia-driver-G06-kmp-default",
		},
		DriverDKMS: []string{
			"nvidia-driver-G06-kmp-default",
		},
		Utils: []string{
			"nvidia-driver-G06",
		},
		Settings: []string{
			"nvidia-settings",
		},
		CUDA: []string{
			"cuda",
		},
		CUDACompiler: []string{
			"cuda",
		},
		CUDALibs: []string{
			"cuda-devel",
		},
		CUDnn: []string{
			"libcudnn8",
		},
		OpenCL: []string{
			"nvidia-gl-G06",
		},
		Vulkan: []string{
			"nvidia-gl-G06",
		},
		Notes: "openSUSE Leap uses the official NVIDIA repository for the specific version",
	},

	"manjaro": {
		Family:       constants.FamilyArch,
		Distribution: "manjaro",
		Driver: []string{
			"linux-nvidia", // Manjaro's version-matched driver
			"nvidia",
		},
		DriverDKMS: []string{
			"nvidia-dkms",
		},
		Utils: []string{
			"nvidia-utils",
			"lib32-nvidia-utils",
		},
		Settings: []string{
			"nvidia-settings",
		},
		CUDA: []string{
			"cuda",
		},
		CUDACompiler: []string{
			"cuda",
		},
		CUDALibs: []string{
			"cuda",
		},
		CUDnn: []string{
			"cudnn",
		},
		OpenCL: []string{
			"opencl-nvidia",
		},
		Vulkan: []string{
			"nvidia-utils",
		},
		Notes: "Manjaro provides version-matched nvidia packages via mhwd",
	},
}

// GetPackageSet returns the PackageSet for a specific distribution.
// It first checks for distribution-specific overrides, then falls back
// to the family-level package set.
func GetPackageSet(dist *distro.Distribution) *PackageSet {
	if dist == nil {
		return nil
	}

	// Check for distribution-specific package set first
	if ps, ok := distroSpecificPackageSets[dist.ID]; ok {
		return ps
	}

	// Fall back to family-level package set
	return GetPackageSetForFamily(dist.Family)
}

// GetPackageSetForFamily returns the default PackageSet for a distribution family.
func GetPackageSetForFamily(family constants.DistroFamily) *PackageSet {
	if ps, ok := packageSets[family]; ok {
		return ps
	}
	return nil
}

// GetPackageSetByID returns the PackageSet for a specific distribution ID.
func GetPackageSetByID(distroID string) *PackageSet {
	distroID = strings.ToLower(distroID)
	if ps, ok := distroSpecificPackageSets[distroID]; ok {
		return ps
	}
	return nil
}

// IsSupported returns true if the given distribution family is supported.
func IsSupported(family constants.DistroFamily) bool {
	_, ok := packageSets[family]
	return ok
}

// SupportedFamilies returns a list of all supported distribution families.
func SupportedFamilies() []constants.DistroFamily {
	return []constants.DistroFamily{
		constants.FamilyDebian,
		constants.FamilyRHEL,
		constants.FamilyArch,
		constants.FamilySUSE,
	}
}

// GetRecommendedDriverVersion returns the recommended driver version for new installations.
func GetRecommendedDriverVersion() string {
	if len(SupportedDriverVersions) > 0 {
		return SupportedDriverVersions[0]
	}
	return "550"
}

// GetLTSDriverVersion returns the Long-Term Support driver version.
func GetLTSDriverVersion() string {
	return "535"
}

// GetLegacyDriverVersion returns the legacy driver version for older GPUs.
func GetLegacyDriverVersion() string {
	return "470"
}
