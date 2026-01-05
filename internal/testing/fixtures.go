package testing

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/tungetti/igor/internal/constants"
	"github.com/tungetti/igor/internal/distro"
	"github.com/tungetti/igor/internal/gpu/pci"
	"github.com/tungetti/igor/internal/pkg"
)

// ============================================================================
// OS Release File Contents - Test fixtures for distro detection
// ============================================================================

// UbuntuOSRelease contains sample /etc/os-release content for Ubuntu 22.04.
const UbuntuOSRelease = `NAME="Ubuntu"
VERSION="22.04.3 LTS (Jammy Jellyfish)"
ID=ubuntu
ID_LIKE=debian
PRETTY_NAME="Ubuntu 22.04.3 LTS"
VERSION_ID="22.04"
VERSION_CODENAME=jammy
HOME_URL="https://www.ubuntu.com/"
SUPPORT_URL="https://help.ubuntu.com/"`

// Ubuntu2404OSRelease contains sample /etc/os-release content for Ubuntu 24.04.
const Ubuntu2404OSRelease = `NAME="Ubuntu"
VERSION="24.04 LTS (Noble Numbat)"
ID=ubuntu
ID_LIKE=debian
PRETTY_NAME="Ubuntu 24.04 LTS"
VERSION_ID="24.04"
VERSION_CODENAME=noble
HOME_URL="https://www.ubuntu.com/"
SUPPORT_URL="https://help.ubuntu.com/"`

// FedoraOSRelease contains sample /etc/os-release content for Fedora 39.
const FedoraOSRelease = `NAME="Fedora Linux"
VERSION="39 (Workstation Edition)"
ID=fedora
VERSION_ID=39
PRETTY_NAME="Fedora Linux 39 (Workstation Edition)"
HOME_URL="https://fedoraproject.org/"
SUPPORT_URL="https://ask.fedoraproject.org/"`

// Fedora40OSRelease contains sample /etc/os-release content for Fedora 40.
const Fedora40OSRelease = `NAME="Fedora Linux"
VERSION="40 (Workstation Edition)"
ID=fedora
VERSION_ID=40
PRETTY_NAME="Fedora Linux 40 (Workstation Edition)"
HOME_URL="https://fedoraproject.org/"
SUPPORT_URL="https://ask.fedoraproject.org/"`

// ArchOSRelease contains sample /etc/os-release content for Arch Linux.
const ArchOSRelease = `NAME="Arch Linux"
PRETTY_NAME="Arch Linux"
ID=arch
BUILD_ID=rolling
HOME_URL="https://archlinux.org/"
SUPPORT_URL="https://wiki.archlinux.org/"`

// OpenSUSEOSRelease contains sample /etc/os-release content for openSUSE Leap 15.5.
const OpenSUSEOSRelease = `NAME="openSUSE Leap"
VERSION="15.5"
ID="opensuse-leap"
ID_LIKE="suse opensuse"
VERSION_ID="15.5"
PRETTY_NAME="openSUSE Leap 15.5"
HOME_URL="https://www.opensuse.org/"
SUPPORT_URL="https://en.opensuse.org/Portal:Support"`

// OpenSUSETumbleweedOSRelease contains sample /etc/os-release content for openSUSE Tumbleweed.
const OpenSUSETumbleweedOSRelease = `NAME="openSUSE Tumbleweed"
ID="opensuse-tumbleweed"
ID_LIKE="opensuse suse"
PRETTY_NAME="openSUSE Tumbleweed"
BUILD_ID=20231215
HOME_URL="https://www.opensuse.org/"`

// DebianOSRelease contains sample /etc/os-release content for Debian 12.
const DebianOSRelease = `NAME="Debian GNU/Linux"
VERSION="12 (bookworm)"
ID=debian
VERSION_ID="12"
VERSION_CODENAME=bookworm
PRETTY_NAME="Debian GNU/Linux 12 (bookworm)"
HOME_URL="https://www.debian.org/"
SUPPORT_URL="https://www.debian.org/support"`

// RHELOSRelease contains sample /etc/os-release content for RHEL 9.
const RHELOSRelease = `NAME="Red Hat Enterprise Linux"
VERSION="9.3 (Plow)"
ID=rhel
ID_LIKE="fedora"
VERSION_ID="9.3"
PRETTY_NAME="Red Hat Enterprise Linux 9.3 (Plow)"
HOME_URL="https://www.redhat.com/"
SUPPORT_URL="https://access.redhat.com/support"`

// CentOSOSRelease contains sample /etc/os-release content for CentOS Stream 9.
const CentOSOSRelease = `NAME="CentOS Stream"
VERSION="9"
ID=centos
ID_LIKE="rhel fedora"
VERSION_ID="9"
PRETTY_NAME="CentOS Stream 9"
HOME_URL="https://centos.org/"
SUPPORT_URL="https://centos.org/help/"`

// RockyOSRelease contains sample /etc/os-release content for Rocky Linux 9.
const RockyOSRelease = `NAME="Rocky Linux"
VERSION="9.3 (Blue Onyx)"
ID=rocky
ID_LIKE="rhel centos fedora"
VERSION_ID="9.3"
PRETTY_NAME="Rocky Linux 9.3 (Blue Onyx)"
HOME_URL="https://rockylinux.org/"
SUPPORT_URL="https://wiki.rockylinux.org/"`

// ManjaroOSRelease contains sample /etc/os-release content for Manjaro.
const ManjaroOSRelease = `NAME="Manjaro Linux"
PRETTY_NAME="Manjaro Linux"
ID=manjaro
ID_LIKE=arch
BUILD_ID=rolling
HOME_URL="https://manjaro.org/"
SUPPORT_URL="https://wiki.manjaro.org/"`

// PopOSOSRelease contains sample /etc/os-release content for Pop!_OS.
const PopOSOSRelease = `NAME="Pop!_OS"
VERSION="22.04 LTS"
ID=pop
ID_LIKE="ubuntu debian"
VERSION_ID="22.04"
PRETTY_NAME="Pop!_OS 22.04 LTS"
HOME_URL="https://pop.system76.com/"
SUPPORT_URL="https://support.system76.com/"`

// ============================================================================
// Distribution Fixtures
// ============================================================================

// UbuntuDistribution returns a sample Ubuntu 22.04 distribution.
func UbuntuDistribution() *distro.Distribution {
	return &distro.Distribution{
		ID:              "ubuntu",
		Name:            "Ubuntu",
		Version:         "22.04.3 LTS (Jammy Jellyfish)",
		VersionID:       "22.04",
		VersionCodename: "jammy",
		PrettyName:      "Ubuntu 22.04.3 LTS",
		Family:          constants.FamilyDebian,
		IDLike:          []string{"debian"},
		HomeURL:         "https://www.ubuntu.com/",
		SupportURL:      "https://help.ubuntu.com/",
	}
}

// FedoraDistribution returns a sample Fedora 39 distribution.
func FedoraDistribution() *distro.Distribution {
	return &distro.Distribution{
		ID:         "fedora",
		Name:       "Fedora Linux",
		Version:    "39 (Workstation Edition)",
		VersionID:  "39",
		PrettyName: "Fedora Linux 39 (Workstation Edition)",
		Family:     constants.FamilyRHEL,
		HomeURL:    "https://fedoraproject.org/",
		SupportURL: "https://ask.fedoraproject.org/",
	}
}

// ArchDistribution returns a sample Arch Linux distribution.
func ArchDistribution() *distro.Distribution {
	return &distro.Distribution{
		ID:         "arch",
		Name:       "Arch Linux",
		PrettyName: "Arch Linux",
		Family:     constants.FamilyArch,
		BuildID:    "rolling",
		HomeURL:    "https://archlinux.org/",
		SupportURL: "https://wiki.archlinux.org/",
	}
}

// OpenSUSEDistribution returns a sample openSUSE Leap 15.5 distribution.
func OpenSUSEDistribution() *distro.Distribution {
	return &distro.Distribution{
		ID:         "opensuse-leap",
		Name:       "openSUSE Leap",
		Version:    "15.5",
		VersionID:  "15.5",
		PrettyName: "openSUSE Leap 15.5",
		Family:     constants.FamilySUSE,
		IDLike:     []string{"suse", "opensuse"},
		HomeURL:    "https://www.opensuse.org/",
		SupportURL: "https://en.opensuse.org/Portal:Support",
	}
}

// DebianDistribution returns a sample Debian 12 distribution.
func DebianDistribution() *distro.Distribution {
	return &distro.Distribution{
		ID:              "debian",
		Name:            "Debian GNU/Linux",
		Version:         "12 (bookworm)",
		VersionID:       "12",
		VersionCodename: "bookworm",
		PrettyName:      "Debian GNU/Linux 12 (bookworm)",
		Family:          constants.FamilyDebian,
		HomeURL:         "https://www.debian.org/",
		SupportURL:      "https://www.debian.org/support",
	}
}

// RHELDistribution returns a sample RHEL 9 distribution.
func RHELDistribution() *distro.Distribution {
	return &distro.Distribution{
		ID:         "rhel",
		Name:       "Red Hat Enterprise Linux",
		Version:    "9.3 (Plow)",
		VersionID:  "9.3",
		PrettyName: "Red Hat Enterprise Linux 9.3 (Plow)",
		Family:     constants.FamilyRHEL,
		IDLike:     []string{"fedora"},
		HomeURL:    "https://www.redhat.com/",
		SupportURL: "https://access.redhat.com/support",
	}
}

// DistributionForFamily returns a sample distribution for the given family.
func DistributionForFamily(family constants.DistroFamily) *distro.Distribution {
	switch family {
	case constants.FamilyDebian:
		return UbuntuDistribution()
	case constants.FamilyRHEL:
		return FedoraDistribution()
	case constants.FamilyArch:
		return ArchDistribution()
	case constants.FamilySUSE:
		return OpenSUSEDistribution()
	default:
		return &distro.Distribution{
			ID:         "unknown",
			Name:       "Unknown Linux",
			PrettyName: "Unknown Linux Distribution",
			Family:     constants.FamilyUnknown,
		}
	}
}

// ============================================================================
// GPU Fixtures
// ============================================================================

// GPUFixture contains sample GPU device data for testing.
type GPUFixture struct {
	Device       *pci.PCIDevice
	VendorID     string
	DeviceID     string
	Name         string
	Architecture string
}

// SampleNvidiaGPU returns a sample RTX 3080 GPU fixture.
func SampleNvidiaGPU() GPUFixture {
	return GPUFixture{
		Device: &pci.PCIDevice{
			Address:     "0000:01:00.0",
			VendorID:    "10de",
			DeviceID:    "2206",
			Class:       "030000",
			SubVendorID: "10de",
			SubDeviceID: "1467",
			Driver:      "nvidia",
		},
		VendorID:     "10de",
		DeviceID:     "2206",
		Name:         "NVIDIA GeForce RTX 3080",
		Architecture: "Ampere",
	}
}

// SampleNvidiaDatacenterGPU returns a sample A100 GPU fixture.
func SampleNvidiaDatacenterGPU() GPUFixture {
	return GPUFixture{
		Device: &pci.PCIDevice{
			Address:     "0000:01:00.0",
			VendorID:    "10de",
			DeviceID:    "20b0",
			Class:       "030200",
			SubVendorID: "10de",
			SubDeviceID: "1450",
			Driver:      "nvidia",
		},
		VendorID:     "10de",
		DeviceID:     "20b0",
		Name:         "NVIDIA A100 PCIe 40GB",
		Architecture: "Ampere",
	}
}

// SampleOldNvidiaGPU returns a sample GTX 750 Ti GPU fixture.
func SampleOldNvidiaGPU() GPUFixture {
	return GPUFixture{
		Device: &pci.PCIDevice{
			Address:     "0000:01:00.0",
			VendorID:    "10de",
			DeviceID:    "1380",
			Class:       "030000",
			SubVendorID: "10de",
			SubDeviceID: "1033",
			Driver:      "nouveau",
		},
		VendorID:     "10de",
		DeviceID:     "1380",
		Name:         "NVIDIA GeForce GTX 750 Ti",
		Architecture: "Maxwell",
	}
}

// SampleAMDGPU returns a sample AMD RX 6800 GPU fixture (for testing non-NVIDIA).
func SampleAMDGPU() GPUFixture {
	return GPUFixture{
		Device: &pci.PCIDevice{
			Address:     "0000:01:00.0",
			VendorID:    "1002",
			DeviceID:    "73bf",
			Class:       "030000",
			SubVendorID: "1002",
			SubDeviceID: "0e36",
			Driver:      "amdgpu",
		},
		VendorID:     "1002",
		DeviceID:     "73bf",
		Name:         "AMD Radeon RX 6800",
		Architecture: "RDNA 2",
	}
}

// SampleNvidiaGPUNoDriver returns a sample NVIDIA GPU with no driver bound.
func SampleNvidiaGPUNoDriver() GPUFixture {
	return GPUFixture{
		Device: &pci.PCIDevice{
			Address:     "0000:01:00.0",
			VendorID:    "10de",
			DeviceID:    "2684",
			Class:       "030000",
			SubVendorID: "10de",
			SubDeviceID: "16a1",
			Driver:      "",
		},
		VendorID:     "10de",
		DeviceID:     "2684",
		Name:         "NVIDIA GeForce RTX 4090",
		Architecture: "Ada Lovelace",
	}
}

// ============================================================================
// lspci Output Fixtures
// ============================================================================

// LspciOutputWithNvidiaGPU returns sample lspci output with an NVIDIA GPU.
func LspciOutputWithNvidiaGPU() string {
	return `00:00.0 Host bridge: Intel Corporation 12th Gen Core Processor Host Bridge/DRAM Registers (rev 02)
00:01.0 PCI bridge: Intel Corporation 12th Gen Core Processor PCI Express x16 Controller #1 (rev 02)
00:02.0 VGA compatible controller: Intel Corporation Alder Lake-P GT2 [Iris Xe Graphics] (rev 0c)
01:00.0 VGA compatible controller: NVIDIA Corporation GA104 [GeForce RTX 3080] (rev a1)
01:00.1 Audio device: NVIDIA Corporation GA104 High Definition Audio Controller (rev a1)`
}

// LspciOutputWithMultipleGPUs returns sample lspci output with multiple GPUs.
func LspciOutputWithMultipleGPUs() string {
	return `00:00.0 Host bridge: Intel Corporation Device 9a14 (rev 01)
00:02.0 VGA compatible controller: Intel Corporation Device 9a49 (rev 03)
01:00.0 VGA compatible controller: NVIDIA Corporation GA104 [GeForce RTX 3080] (rev a1)
02:00.0 VGA compatible controller: NVIDIA Corporation GA102 [GeForce RTX 3090] (rev a1)
03:00.0 3D controller: NVIDIA Corporation GA100 [A100 PCIe 40GB] (rev a1)`
}

// LspciOutputWithNoGPU returns sample lspci output with no discrete GPU.
func LspciOutputWithNoGPU() string {
	return `00:00.0 Host bridge: Intel Corporation 12th Gen Core Processor Host Bridge/DRAM Registers (rev 02)
00:02.0 VGA compatible controller: Intel Corporation Alder Lake-P GT2 [Iris Xe Graphics] (rev 0c)
00:14.0 USB controller: Intel Corporation Alder Lake PCH USB 3.2 xHCI Host Controller (rev 01)
00:1f.0 ISA bridge: Intel Corporation Alder Lake PCH eSPI Controller (rev 01)`
}

// LspciOutputWithNouveauDriver returns sample lspci output with nouveau driver.
func LspciOutputWithNouveauDriver() string {
	return `01:00.0 VGA compatible controller: NVIDIA Corporation GA104 [GeForce RTX 3080] (rev a1) (prog-if 00 [VGA controller])
	Subsystem: NVIDIA Corporation GeForce RTX 3080
	Flags: bus master, fast devsel, latency 0, IRQ 16
	Kernel driver in use: nouveau
	Kernel modules: nouveau, nvidia_drm, nvidia`
}

// ============================================================================
// nvidia-smi Output Fixtures
// ============================================================================

// NvidiaSMIOutput returns sample nvidia-smi output.
func NvidiaSMIOutput() string {
	return `Wed Dec 20 10:30:45 2023
+-----------------------------------------------------------------------------+
| NVIDIA-SMI 535.129.03   Driver Version: 535.129.03   CUDA Version: 12.2     |
|-------------------------------+----------------------+----------------------+
| GPU  Name        Persistence-M| Bus-Id        Disp.A | Volatile Uncorr. ECC |
| Fan  Temp  Perf  Pwr:Usage/Cap|         Memory-Usage | GPU-Util  Compute M. |
|                               |                      |               MIG M. |
|===============================+======================+======================|
|   0  NVIDIA GeForce ...  Off  | 00000000:01:00.0  On |                  N/A |
|  0%   38C    P8    19W / 320W |    654MiB / 10240MiB |      0%      Default |
|                               |                      |                  N/A |
+-------------------------------+----------------------+----------------------+

+-----------------------------------------------------------------------------+
| Processes:                                                                  |
|  GPU   GI   CI        PID   Type   Process name                  GPU Memory |
|        ID   ID                                                   Usage      |
|=============================================================================|
|    0   N/A  N/A      1234      G   /usr/lib/xorg/Xorg                 123MiB |
|    0   N/A  N/A      5678      G   /usr/bin/gnome-shell               234MiB |
+-----------------------------------------------------------------------------+`
}

// NvidiaSMIOutputMultiGPU returns sample nvidia-smi output with multiple GPUs.
func NvidiaSMIOutputMultiGPU() string {
	return `Wed Dec 20 10:30:45 2023
+-----------------------------------------------------------------------------+
| NVIDIA-SMI 535.129.03   Driver Version: 535.129.03   CUDA Version: 12.2     |
|-------------------------------+----------------------+----------------------+
| GPU  Name        Persistence-M| Bus-Id        Disp.A | Volatile Uncorr. ECC |
|===============================+======================+======================|
|   0  NVIDIA GeForce ...  Off  | 00000000:01:00.0  On |                  N/A |
|   1  NVIDIA GeForce ...  Off  | 00000000:02:00.0 Off |                  N/A |
+-------------------------------+----------------------+----------------------+`
}

// NvidiaSMIError returns sample nvidia-smi error output.
func NvidiaSMIError() string {
	return `NVIDIA-SMI has failed because it couldn't communicate with the NVIDIA driver. Make sure that the latest NVIDIA driver is installed and running.`
}

// NvidiaSMIQueryGPU returns sample nvidia-smi query output in CSV format.
func NvidiaSMIQueryGPU() string {
	return `name, pci.bus_id, driver_version, memory.total [MiB], compute_cap
NVIDIA GeForce RTX 3080, 00000000:01:00.0, 535.129.03, 10240 MiB, 8.6`
}

// ============================================================================
// Package Fixtures
// ============================================================================

// SampleNvidiaDriverPackage returns a sample NVIDIA driver package.
func SampleNvidiaDriverPackage(version string) pkg.Package {
	return pkg.Package{
		Name:         "nvidia-driver-" + version,
		Version:      version + "-1",
		Installed:    false,
		Repository:   "ubuntu-drivers",
		Description:  "NVIDIA driver metapackage",
		Architecture: "amd64",
	}
}

// SampleCUDAPackage returns a sample CUDA package.
func SampleCUDAPackage(version string) pkg.Package {
	return pkg.Package{
		Name:         "cuda-toolkit-" + version,
		Version:      version + "-1",
		Installed:    false,
		Repository:   "nvidia-cuda",
		Description:  "NVIDIA CUDA Toolkit",
		Architecture: "amd64",
	}
}

// InstalledNvidiaPackages returns a list of installed NVIDIA packages.
func InstalledNvidiaPackages() []pkg.Package {
	return []pkg.Package{
		{Name: "nvidia-driver-535", Version: "535.154.05-0ubuntu0.22.04.1", Installed: true},
		{Name: "libnvidia-gl-535", Version: "535.154.05-0ubuntu0.22.04.1", Installed: true},
		{Name: "nvidia-utils-535", Version: "535.154.05-0ubuntu0.22.04.1", Installed: true},
		{Name: "libnvidia-compute-535", Version: "535.154.05-0ubuntu0.22.04.1", Installed: true},
		{Name: "nvidia-kernel-common-535", Version: "535.154.05-0ubuntu0.22.04.1", Installed: true},
		{Name: "nvidia-dkms-535", Version: "535.154.05-0ubuntu0.22.04.1", Installed: true},
	}
}

// AvailableNvidiaDrivers returns a list of available NVIDIA driver packages.
func AvailableNvidiaDrivers() []pkg.Package {
	return []pkg.Package{
		{Name: "nvidia-driver-470", Version: "470.223.02-0ubuntu0.22.04.1", Installed: false},
		{Name: "nvidia-driver-535", Version: "535.154.05-0ubuntu0.22.04.1", Installed: false},
		{Name: "nvidia-driver-545", Version: "545.29.06-0ubuntu0.22.04.1", Installed: false},
		{Name: "nvidia-driver-550", Version: "550.54.14-0ubuntu0.22.04.1", Installed: false},
	}
}

// ============================================================================
// Proc/Modules Content Fixtures
// ============================================================================

// ProcModulesWithNvidia returns sample /proc/modules content with NVIDIA loaded.
func ProcModulesWithNvidia() string {
	return `nvidia_drm 77824 2 - Live 0xffffffffc1000000
nvidia_modeset 1323008 3 nvidia_drm, Live 0xffffffffc0e00000
nvidia 56762368 51 nvidia_modeset, Live 0xffffffffc0800000
drm_kms_helper 299008 1 nvidia_drm, Live 0xffffffffc0700000
drm 610304 5 nvidia_drm,drm_kms_helper, Live 0xffffffffc0600000
i2c_nvidia_gpu 16384 0 - Live 0xffffffffc0500000`
}

// ProcModulesWithNouveau returns sample /proc/modules content with nouveau loaded.
func ProcModulesWithNouveau() string {
	return `nouveau 2355200 4 - Live 0xffffffffc0800000
mxm_wmi 16384 1 nouveau, Live 0xffffffffc0700000
video 53248 2 nouveau, Live 0xffffffffc0600000
drm_kms_helper 299008 1 nouveau, Live 0xffffffffc0500000
drm 610304 5 nouveau,drm_kms_helper, Live 0xffffffffc0400000`
}

// ProcModulesWithBoth returns sample /proc/modules content with both nvidia and nouveau.
func ProcModulesWithBoth() string {
	return `nvidia_drm 77824 0 - Live 0xffffffffc1000000
nvidia_modeset 1323008 1 nvidia_drm, Live 0xffffffffc0e00000
nvidia 56762368 1 nvidia_modeset, Live 0xffffffffc0800000
nouveau 2355200 0 - Live 0xffffffffc0700000
drm_kms_helper 299008 2 nvidia_drm,nouveau, Live 0xffffffffc0600000
drm 610304 6 nvidia_drm,drm_kms_helper,nouveau, Live 0xffffffffc0500000`
}

// ProcModulesClean returns sample /proc/modules content with neither driver.
func ProcModulesClean() string {
	return `drm_kms_helper 299008 1 - Live 0xffffffffc0500000
drm 610304 2 drm_kms_helper, Live 0xffffffffc0400000
i2c_core 77824 2 drm_kms_helper,drm, Live 0xffffffffc0300000`
}

// ============================================================================
// Modprobe Configuration Fixtures
// ============================================================================

// NouveauBlacklistContent returns sample nouveau blacklist configuration.
func NouveauBlacklistContent() string {
	return `# Blacklist nouveau driver to use NVIDIA proprietary driver
# Generated by igor

blacklist nouveau
blacklist lbm-nouveau
alias nouveau off
alias lbm-nouveau off
options nouveau modeset=0`
}

// ============================================================================
// Xorg Configuration Fixtures
// ============================================================================

// XorgConfNvidia returns sample xorg.conf for NVIDIA driver.
func XorgConfNvidia() string {
	return `# Generated by igor
Section "OutputClass"
    Identifier "nvidia"
    MatchDriver "nvidia-drm"
    Driver "nvidia"
    Option "AllowEmptyInitialConfiguration"
    Option "PrimaryGPU" "yes"
    ModulePath "/usr/lib/nvidia/xorg"
    ModulePath "/usr/lib/xorg/modules"
EndSection`
}

// XorgConfNouveau returns sample xorg.conf for nouveau driver.
func XorgConfNouveau() string {
	return `Section "Device"
    Identifier "nouveau"
    Driver "nouveau"
EndSection`
}

// XorgConfPrimeSync returns sample xorg.conf for PRIME synchronization.
func XorgConfPrimeSync() string {
	return `# PRIME Render Offload configuration
Section "ServerLayout"
    Identifier "layout"
    Option "AllowNVIDIAGPUScreens"
EndSection

Section "Device"
    Identifier "nvidia"
    Driver "nvidia"
    BusID "PCI:1:0:0"
    Option "AllowEmptyInitialConfiguration"
EndSection`
}

// ============================================================================
// TempDirBuilder - Create temporary directories with files for testing
// ============================================================================

// TempDirBuilder helps create temporary directories with files for testing.
type TempDirBuilder struct {
	files map[string]string
}

// NewTempDirBuilder creates a new TempDirBuilder.
func NewTempDirBuilder() *TempDirBuilder {
	return &TempDirBuilder{
		files: make(map[string]string),
	}
}

// WithFile adds a file with the given path and content.
// Path is relative to the temp directory root.
func (b *TempDirBuilder) WithFile(path, content string) *TempDirBuilder {
	b.files[path] = content
	return b
}

// WithOSRelease adds an os-release file with the given content.
func (b *TempDirBuilder) WithOSRelease(content string) *TempDirBuilder {
	return b.WithFile("etc/os-release", content)
}

// WithProcModules adds a proc/modules file with the given content.
func (b *TempDirBuilder) WithProcModules(content string) *TempDirBuilder {
	return b.WithFile("proc/modules", content)
}

// WithNouveauBlacklist adds the nouveau blacklist file.
func (b *TempDirBuilder) WithNouveauBlacklist() *TempDirBuilder {
	return b.WithFile("etc/modprobe.d/blacklist-nouveau.conf", NouveauBlacklistContent())
}

// Build creates the temporary directory with all configured files.
// Returns the temp directory path and a cleanup function.
func (b *TempDirBuilder) Build(t testing.TB) (string, func()) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "igor-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	for path, content := range b.files {
		fullPath := filepath.Join(tmpDir, path)

		// Create parent directories
		dir := filepath.Dir(fullPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			os.RemoveAll(tmpDir)
			t.Fatalf("failed to create directory %s: %v", dir, err)
		}

		// Write file
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			os.RemoveAll(tmpDir)
			t.Fatalf("failed to write file %s: %v", fullPath, err)
		}
	}

	cleanup := func() {
		os.RemoveAll(tmpDir)
	}

	return tmpDir, cleanup
}

// ============================================================================
// MockFileReader - Implements distro.FileReader for testing
// ============================================================================

// MockFileReader implements distro.FileReader for testing.
type MockFileReader struct {
	files map[string]string
}

// NewMockFileReader creates a new MockFileReader with the given files.
func NewMockFileReader(files map[string]string) *MockFileReader {
	if files == nil {
		files = make(map[string]string)
	}
	return &MockFileReader{files: files}
}

// ReadFile reads the content of a file.
func (r *MockFileReader) ReadFile(path string) ([]byte, error) {
	content, ok := r.files[path]
	if !ok {
		return nil, os.ErrNotExist
	}
	return []byte(content), nil
}

// FileExists checks if a file exists.
func (r *MockFileReader) FileExists(path string) bool {
	_, ok := r.files[path]
	return ok
}

// AddFile adds a file to the mock filesystem.
func (r *MockFileReader) AddFile(path, content string) {
	r.files[path] = content
}

// RemoveFile removes a file from the mock filesystem.
func (r *MockFileReader) RemoveFile(path string) {
	delete(r.files, path)
}
