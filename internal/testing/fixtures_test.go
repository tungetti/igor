package testing

import (
	"os"
	"path/filepath"
	"strings"
	stdtesting "testing"

	"github.com/tungetti/igor/internal/constants"
	"github.com/tungetti/igor/internal/distro"
)

// ============================================================================
// OS Release Content Tests
// ============================================================================

func TestUbuntuOSRelease_ParsesCorrectly(t *stdtesting.T) {
	dist, err := distro.ParseOSReleaseContent(UbuntuOSRelease)
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}

	if dist.ID != "ubuntu" {
		t.Errorf("expected ID 'ubuntu', got %s", dist.ID)
	}
	if dist.VersionID != "22.04" {
		t.Errorf("expected VersionID '22.04', got %s", dist.VersionID)
	}
	if dist.VersionCodename != "jammy" {
		t.Errorf("expected VersionCodename 'jammy', got %s", dist.VersionCodename)
	}
	if dist.Family != constants.FamilyDebian {
		t.Errorf("expected family %s, got %s", constants.FamilyDebian, dist.Family)
	}
}

func TestFedoraOSRelease_ParsesCorrectly(t *stdtesting.T) {
	dist, err := distro.ParseOSReleaseContent(FedoraOSRelease)
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}

	if dist.ID != "fedora" {
		t.Errorf("expected ID 'fedora', got %s", dist.ID)
	}
	if dist.VersionID != "39" {
		t.Errorf("expected VersionID '39', got %s", dist.VersionID)
	}
	if dist.Family != constants.FamilyRHEL {
		t.Errorf("expected family %s, got %s", constants.FamilyRHEL, dist.Family)
	}
}

func TestArchOSRelease_ParsesCorrectly(t *stdtesting.T) {
	dist, err := distro.ParseOSReleaseContent(ArchOSRelease)
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}

	if dist.ID != "arch" {
		t.Errorf("expected ID 'arch', got %s", dist.ID)
	}
	if dist.BuildID != "rolling" {
		t.Errorf("expected BuildID 'rolling', got %s", dist.BuildID)
	}
	if dist.Family != constants.FamilyArch {
		t.Errorf("expected family %s, got %s", constants.FamilyArch, dist.Family)
	}
}

func TestOpenSUSEOSRelease_ParsesCorrectly(t *stdtesting.T) {
	dist, err := distro.ParseOSReleaseContent(OpenSUSEOSRelease)
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}

	if dist.ID != "opensuse-leap" {
		t.Errorf("expected ID 'opensuse-leap', got %s", dist.ID)
	}
	if dist.VersionID != "15.5" {
		t.Errorf("expected VersionID '15.5', got %s", dist.VersionID)
	}
	if dist.Family != constants.FamilySUSE {
		t.Errorf("expected family %s, got %s", constants.FamilySUSE, dist.Family)
	}
}

func TestDebianOSRelease_ParsesCorrectly(t *stdtesting.T) {
	dist, err := distro.ParseOSReleaseContent(DebianOSRelease)
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}

	if dist.ID != "debian" {
		t.Errorf("expected ID 'debian', got %s", dist.ID)
	}
	if dist.VersionID != "12" {
		t.Errorf("expected VersionID '12', got %s", dist.VersionID)
	}
	if dist.VersionCodename != "bookworm" {
		t.Errorf("expected VersionCodename 'bookworm', got %s", dist.VersionCodename)
	}
}

func TestRHELOSRelease_ParsesCorrectly(t *stdtesting.T) {
	dist, err := distro.ParseOSReleaseContent(RHELOSRelease)
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}

	if dist.ID != "rhel" {
		t.Errorf("expected ID 'rhel', got %s", dist.ID)
	}
	if dist.VersionID != "9.3" {
		t.Errorf("expected VersionID '9.3', got %s", dist.VersionID)
	}
}

func TestCentOSOSRelease_ParsesCorrectly(t *stdtesting.T) {
	dist, err := distro.ParseOSReleaseContent(CentOSOSRelease)
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}

	if dist.ID != "centos" {
		t.Errorf("expected ID 'centos', got %s", dist.ID)
	}
}

func TestRockyOSRelease_ParsesCorrectly(t *stdtesting.T) {
	dist, err := distro.ParseOSReleaseContent(RockyOSRelease)
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}

	if dist.ID != "rocky" {
		t.Errorf("expected ID 'rocky', got %s", dist.ID)
	}
	if dist.Family != constants.FamilyRHEL {
		t.Errorf("expected family %s, got %s", constants.FamilyRHEL, dist.Family)
	}
}

func TestManjaroOSRelease_ParsesCorrectly(t *stdtesting.T) {
	dist, err := distro.ParseOSReleaseContent(ManjaroOSRelease)
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}

	if dist.ID != "manjaro" {
		t.Errorf("expected ID 'manjaro', got %s", dist.ID)
	}
	if dist.Family != constants.FamilyArch {
		t.Errorf("expected family %s, got %s", constants.FamilyArch, dist.Family)
	}
}

func TestPopOSOSRelease_ParsesCorrectly(t *stdtesting.T) {
	dist, err := distro.ParseOSReleaseContent(PopOSOSRelease)
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}

	if dist.ID != "pop" {
		t.Errorf("expected ID 'pop', got %s", dist.ID)
	}
	if dist.Family != constants.FamilyDebian {
		t.Errorf("expected family %s, got %s", constants.FamilyDebian, dist.Family)
	}
}

// ============================================================================
// Distribution Fixture Tests
// ============================================================================

func TestUbuntuDistribution_ReturnsCorrectValues(t *stdtesting.T) {
	dist := UbuntuDistribution()

	if dist.ID != "ubuntu" {
		t.Errorf("expected ID 'ubuntu', got %s", dist.ID)
	}
	if dist.Family != constants.FamilyDebian {
		t.Errorf("expected family %s, got %s", constants.FamilyDebian, dist.Family)
	}
	if dist.VersionID != "22.04" {
		t.Errorf("expected VersionID '22.04', got %s", dist.VersionID)
	}
	if dist.VersionCodename != "jammy" {
		t.Errorf("expected VersionCodename 'jammy', got %s", dist.VersionCodename)
	}
	if len(dist.IDLike) == 0 || dist.IDLike[0] != "debian" {
		t.Errorf("expected IDLike to include 'debian'")
	}
}

func TestFedoraDistribution_ReturnsCorrectValues(t *stdtesting.T) {
	dist := FedoraDistribution()

	if dist.ID != "fedora" {
		t.Errorf("expected ID 'fedora', got %s", dist.ID)
	}
	if dist.Family != constants.FamilyRHEL {
		t.Errorf("expected family %s, got %s", constants.FamilyRHEL, dist.Family)
	}
}

func TestArchDistribution_ReturnsCorrectValues(t *stdtesting.T) {
	dist := ArchDistribution()

	if dist.ID != "arch" {
		t.Errorf("expected ID 'arch', got %s", dist.ID)
	}
	if dist.Family != constants.FamilyArch {
		t.Errorf("expected family %s, got %s", constants.FamilyArch, dist.Family)
	}
	if dist.BuildID != "rolling" {
		t.Errorf("expected BuildID 'rolling', got %s", dist.BuildID)
	}
}

func TestOpenSUSEDistribution_ReturnsCorrectValues(t *stdtesting.T) {
	dist := OpenSUSEDistribution()

	if dist.ID != "opensuse-leap" {
		t.Errorf("expected ID 'opensuse-leap', got %s", dist.ID)
	}
	if dist.Family != constants.FamilySUSE {
		t.Errorf("expected family %s, got %s", constants.FamilySUSE, dist.Family)
	}
}

func TestDistributionForFamily(t *stdtesting.T) {
	testCases := []struct {
		family     constants.DistroFamily
		expectedID string
	}{
		{constants.FamilyDebian, "ubuntu"},
		{constants.FamilyRHEL, "fedora"},
		{constants.FamilyArch, "arch"},
		{constants.FamilySUSE, "opensuse-leap"},
		{constants.FamilyUnknown, "unknown"},
	}

	for _, tc := range testCases {
		t.Run(string(tc.family), func(t *stdtesting.T) {
			dist := DistributionForFamily(tc.family)
			if dist.ID != tc.expectedID {
				t.Errorf("expected ID %s, got %s", tc.expectedID, dist.ID)
			}
		})
	}
}

// ============================================================================
// GPU Fixture Tests
// ============================================================================

func TestSampleNvidiaGPU_HasValidData(t *stdtesting.T) {
	gpu := SampleNvidiaGPU()

	if gpu.VendorID != "10de" {
		t.Errorf("expected vendor ID '10de', got %s", gpu.VendorID)
	}
	if gpu.Device == nil {
		t.Fatal("expected device to be non-nil")
	}
	if !gpu.Device.IsNVIDIA() {
		t.Error("expected device to be NVIDIA")
	}
	if !gpu.Device.IsGPU() {
		t.Error("expected device to be a GPU")
	}
	if gpu.Device.Driver != "nvidia" {
		t.Errorf("expected driver 'nvidia', got %s", gpu.Device.Driver)
	}
	if gpu.Architecture != "Ampere" {
		t.Errorf("expected architecture 'Ampere', got %s", gpu.Architecture)
	}
}

func TestSampleNvidiaDatacenterGPU_HasValidData(t *stdtesting.T) {
	gpu := SampleNvidiaDatacenterGPU()

	if gpu.VendorID != "10de" {
		t.Errorf("expected vendor ID '10de', got %s", gpu.VendorID)
	}
	if !gpu.Device.IsNVIDIAGPU() {
		t.Error("expected device to be NVIDIA GPU")
	}
	// A100 has class 0302 (3D controller)
	if !strings.HasPrefix(gpu.Device.Class, "0302") {
		t.Errorf("expected class to start with '0302', got %s", gpu.Device.Class)
	}
}

func TestSampleOldNvidiaGPU_HasValidData(t *stdtesting.T) {
	gpu := SampleOldNvidiaGPU()

	if gpu.VendorID != "10de" {
		t.Errorf("expected vendor ID '10de', got %s", gpu.VendorID)
	}
	if gpu.Device.Driver != "nouveau" {
		t.Errorf("expected driver 'nouveau', got %s", gpu.Device.Driver)
	}
	if gpu.Architecture != "Maxwell" {
		t.Errorf("expected architecture 'Maxwell', got %s", gpu.Architecture)
	}
}

func TestSampleAMDGPU_HasValidData(t *stdtesting.T) {
	gpu := SampleAMDGPU()

	if gpu.VendorID != "1002" {
		t.Errorf("expected vendor ID '1002' (AMD), got %s", gpu.VendorID)
	}
	if gpu.Device.IsNVIDIA() {
		t.Error("expected device NOT to be NVIDIA")
	}
	if gpu.Device.Driver != "amdgpu" {
		t.Errorf("expected driver 'amdgpu', got %s", gpu.Device.Driver)
	}
}

func TestSampleNvidiaGPUNoDriver_HasValidData(t *stdtesting.T) {
	gpu := SampleNvidiaGPUNoDriver()

	if gpu.VendorID != "10de" {
		t.Errorf("expected vendor ID '10de', got %s", gpu.VendorID)
	}
	if gpu.Device.Driver != "" {
		t.Errorf("expected no driver, got %s", gpu.Device.Driver)
	}
	if gpu.Device.HasDriver() {
		t.Error("expected HasDriver to be false")
	}
}

// ============================================================================
// lspci Output Tests
// ============================================================================

func TestLspciOutputWithNvidiaGPU_ContainsNVIDIA(t *stdtesting.T) {
	output := LspciOutputWithNvidiaGPU()

	if !strings.Contains(output, "NVIDIA") {
		t.Error("expected output to contain 'NVIDIA'")
	}
	if !strings.Contains(output, "VGA compatible controller") {
		t.Error("expected output to contain 'VGA compatible controller'")
	}
}

func TestLspciOutputWithMultipleGPUs_ContainsMultiple(t *stdtesting.T) {
	output := LspciOutputWithMultipleGPUs()

	// Count NVIDIA occurrences
	count := strings.Count(output, "NVIDIA")
	if count < 2 {
		t.Errorf("expected at least 2 NVIDIA entries, got %d", count)
	}

	// Should contain 3D controller
	if !strings.Contains(output, "3D controller") {
		t.Error("expected output to contain '3D controller'")
	}
}

func TestLspciOutputWithNoGPU_NoDiscreteGPU(t *stdtesting.T) {
	output := LspciOutputWithNoGPU()

	if strings.Contains(output, "NVIDIA") {
		t.Error("expected output NOT to contain 'NVIDIA'")
	}
	// Should have integrated graphics
	if !strings.Contains(output, "Intel") {
		t.Error("expected output to contain Intel integrated graphics")
	}
}

// ============================================================================
// nvidia-smi Output Tests
// ============================================================================

func TestNvidiaSMIOutput_ContainsDriverInfo(t *stdtesting.T) {
	output := NvidiaSMIOutput()

	if !strings.Contains(output, "NVIDIA-SMI") {
		t.Error("expected output to contain 'NVIDIA-SMI'")
	}
	if !strings.Contains(output, "Driver Version") {
		t.Error("expected output to contain 'Driver Version'")
	}
	if !strings.Contains(output, "CUDA Version") {
		t.Error("expected output to contain 'CUDA Version'")
	}
}

func TestNvidiaSMIOutputMultiGPU_HasMultipleGPUs(t *stdtesting.T) {
	output := NvidiaSMIOutputMultiGPU()

	// Should have GPU 0 and GPU 1
	if !strings.Contains(output, "0  NVIDIA") {
		t.Error("expected output to contain GPU 0")
	}
	if !strings.Contains(output, "1  NVIDIA") {
		t.Error("expected output to contain GPU 1")
	}
}

func TestNvidiaSMIError_ContainsErrorMessage(t *stdtesting.T) {
	output := NvidiaSMIError()

	if !strings.Contains(output, "failed") {
		t.Error("expected output to contain 'failed'")
	}
}

// ============================================================================
// Package Fixture Tests
// ============================================================================

func TestSampleNvidiaDriverPackage_HasCorrectFormat(t *stdtesting.T) {
	pkg := SampleNvidiaDriverPackage("535")

	if pkg.Name != "nvidia-driver-535" {
		t.Errorf("expected name 'nvidia-driver-535', got %s", pkg.Name)
	}
	if !strings.HasPrefix(pkg.Version, "535") {
		t.Errorf("expected version to start with '535', got %s", pkg.Version)
	}
}

func TestSampleCUDAPackage_HasCorrectFormat(t *stdtesting.T) {
	pkg := SampleCUDAPackage("12.2")

	if pkg.Name != "cuda-toolkit-12.2" {
		t.Errorf("expected name 'cuda-toolkit-12.2', got %s", pkg.Name)
	}
}

func TestInstalledNvidiaPackages_AllInstalled(t *stdtesting.T) {
	packages := InstalledNvidiaPackages()

	if len(packages) == 0 {
		t.Fatal("expected non-empty package list")
	}

	for _, pkg := range packages {
		if !pkg.Installed {
			t.Errorf("expected package %s to be marked as installed", pkg.Name)
		}
	}
}

func TestAvailableNvidiaDrivers_AllNotInstalled(t *stdtesting.T) {
	packages := AvailableNvidiaDrivers()

	if len(packages) == 0 {
		t.Fatal("expected non-empty package list")
	}

	for _, pkg := range packages {
		if pkg.Installed {
			t.Errorf("expected package %s to NOT be marked as installed", pkg.Name)
		}
	}
}

// ============================================================================
// Proc/Modules Content Tests
// ============================================================================

func TestProcModulesWithNvidia_ContainsNvidiaModules(t *stdtesting.T) {
	content := ProcModulesWithNvidia()

	if !strings.Contains(content, "nvidia_drm") {
		t.Error("expected content to contain 'nvidia_drm'")
	}
	if !strings.Contains(content, "nvidia_modeset") {
		t.Error("expected content to contain 'nvidia_modeset'")
	}
	if !strings.Contains(content, "nvidia ") {
		t.Error("expected content to contain 'nvidia' module")
	}
}

func TestProcModulesWithNouveau_ContainsNouveau(t *stdtesting.T) {
	content := ProcModulesWithNouveau()

	if !strings.Contains(content, "nouveau") {
		t.Error("expected content to contain 'nouveau'")
	}
	if strings.Contains(content, "nvidia") {
		t.Error("expected content NOT to contain 'nvidia'")
	}
}

func TestProcModulesWithBoth_ContainsBoth(t *stdtesting.T) {
	content := ProcModulesWithBoth()

	if !strings.Contains(content, "nvidia") {
		t.Error("expected content to contain 'nvidia'")
	}
	if !strings.Contains(content, "nouveau") {
		t.Error("expected content to contain 'nouveau'")
	}
}

func TestProcModulesClean_ContainsNeither(t *stdtesting.T) {
	content := ProcModulesClean()

	if strings.Contains(content, "nvidia") {
		t.Error("expected content NOT to contain 'nvidia'")
	}
	if strings.Contains(content, "nouveau") {
		t.Error("expected content NOT to contain 'nouveau'")
	}
}

// ============================================================================
// Configuration Content Tests
// ============================================================================

func TestNouveauBlacklistContent_ContainsBlacklist(t *stdtesting.T) {
	content := NouveauBlacklistContent()

	if !strings.Contains(content, "blacklist nouveau") {
		t.Error("expected content to contain 'blacklist nouveau'")
	}
	if !strings.Contains(content, "modeset=0") {
		t.Error("expected content to contain 'modeset=0'")
	}
}

func TestXorgConfNvidia_ContainsNvidiaConfig(t *stdtesting.T) {
	content := XorgConfNvidia()

	if !strings.Contains(content, "nvidia") {
		t.Error("expected content to contain 'nvidia'")
	}
	if !strings.Contains(content, "OutputClass") {
		t.Error("expected content to contain 'OutputClass'")
	}
}

func TestXorgConfNouveau_ContainsNouveauConfig(t *stdtesting.T) {
	content := XorgConfNouveau()

	if !strings.Contains(content, "nouveau") {
		t.Error("expected content to contain 'nouveau'")
	}
	if !strings.Contains(content, "Device") {
		t.Error("expected content to contain 'Device' section")
	}
}

// ============================================================================
// TempDirBuilder Tests
// ============================================================================

func TestTempDirBuilder_BuildCreatesDirectory(t *stdtesting.T) {
	builder := NewTempDirBuilder()
	dir, cleanup := builder.Build(t)
	defer cleanup()

	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("failed to stat temp dir: %v", err)
	}
	if !info.IsDir() {
		t.Error("expected a directory")
	}
}

func TestTempDirBuilder_WithFile(t *stdtesting.T) {
	builder := NewTempDirBuilder().
		WithFile("test.txt", "hello world")

	dir, cleanup := builder.Build(t)
	defer cleanup()

	filePath := filepath.Join(dir, "test.txt")
	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	if string(content) != "hello world" {
		t.Errorf("expected 'hello world', got %s", string(content))
	}
}

func TestTempDirBuilder_WithNestedFile(t *stdtesting.T) {
	builder := NewTempDirBuilder().
		WithFile("etc/os-release", UbuntuOSRelease)

	dir, cleanup := builder.Build(t)
	defer cleanup()

	filePath := filepath.Join(dir, "etc", "os-release")
	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	if !strings.Contains(string(content), "ubuntu") {
		t.Error("expected file to contain 'ubuntu'")
	}
}

func TestTempDirBuilder_WithOSRelease(t *stdtesting.T) {
	builder := NewTempDirBuilder().
		WithOSRelease(FedoraOSRelease)

	dir, cleanup := builder.Build(t)
	defer cleanup()

	filePath := filepath.Join(dir, "etc", "os-release")
	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	if !strings.Contains(string(content), "fedora") {
		t.Error("expected file to contain 'fedora'")
	}
}

func TestTempDirBuilder_WithProcModules(t *stdtesting.T) {
	builder := NewTempDirBuilder().
		WithProcModules(ProcModulesWithNvidia())

	dir, cleanup := builder.Build(t)
	defer cleanup()

	filePath := filepath.Join(dir, "proc", "modules")
	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	if !strings.Contains(string(content), "nvidia") {
		t.Error("expected file to contain 'nvidia'")
	}
}

func TestTempDirBuilder_WithNouveauBlacklist(t *stdtesting.T) {
	builder := NewTempDirBuilder().
		WithNouveauBlacklist()

	dir, cleanup := builder.Build(t)
	defer cleanup()

	filePath := filepath.Join(dir, "etc", "modprobe.d", "blacklist-nouveau.conf")
	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	if !strings.Contains(string(content), "blacklist nouveau") {
		t.Error("expected file to contain 'blacklist nouveau'")
	}
}

func TestTempDirBuilder_MultipleFiles(t *stdtesting.T) {
	builder := NewTempDirBuilder().
		WithFile("file1.txt", "content1").
		WithFile("dir/file2.txt", "content2").
		WithFile("dir/subdir/file3.txt", "content3")

	dir, cleanup := builder.Build(t)
	defer cleanup()

	files := []struct {
		path    string
		content string
	}{
		{"file1.txt", "content1"},
		{"dir/file2.txt", "content2"},
		{"dir/subdir/file3.txt", "content3"},
	}

	for _, f := range files {
		filePath := filepath.Join(dir, f.path)
		content, err := os.ReadFile(filePath)
		if err != nil {
			t.Errorf("failed to read %s: %v", f.path, err)
			continue
		}
		if string(content) != f.content {
			t.Errorf("expected %s content %q, got %q", f.path, f.content, string(content))
		}
	}
}

func TestTempDirBuilder_Cleanup(t *stdtesting.T) {
	builder := NewTempDirBuilder().
		WithFile("test.txt", "content")

	dir, cleanup := builder.Build(t)

	// Verify directory exists
	if _, err := os.Stat(dir); err != nil {
		t.Fatalf("temp dir should exist: %v", err)
	}

	// Call cleanup
	cleanup()

	// Verify directory is removed
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Error("expected temp dir to be removed after cleanup")
	}
}

// ============================================================================
// MockFileReader Tests
// ============================================================================

func TestMockFileReader_ReadFile(t *stdtesting.T) {
	reader := NewMockFileReader(map[string]string{
		"/etc/os-release": UbuntuOSRelease,
	})

	content, err := reader.ReadFile("/etc/os-release")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if string(content) != UbuntuOSRelease {
		t.Error("content mismatch")
	}
}

func TestMockFileReader_ReadFile_NotFound(t *stdtesting.T) {
	reader := NewMockFileReader(nil)

	_, err := reader.ReadFile("/nonexistent")
	if !os.IsNotExist(err) {
		t.Errorf("expected os.ErrNotExist, got %v", err)
	}
}

func TestMockFileReader_FileExists(t *stdtesting.T) {
	reader := NewMockFileReader(map[string]string{
		"/etc/os-release": "content",
	})

	if !reader.FileExists("/etc/os-release") {
		t.Error("expected file to exist")
	}

	if reader.FileExists("/nonexistent") {
		t.Error("expected file NOT to exist")
	}
}

func TestMockFileReader_AddFile(t *stdtesting.T) {
	reader := NewMockFileReader(nil)

	if reader.FileExists("/test") {
		t.Error("expected file NOT to exist before add")
	}

	reader.AddFile("/test", "content")

	if !reader.FileExists("/test") {
		t.Error("expected file to exist after add")
	}

	content, err := reader.ReadFile("/test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(content) != "content" {
		t.Errorf("expected 'content', got %s", string(content))
	}
}

func TestMockFileReader_RemoveFile(t *stdtesting.T) {
	reader := NewMockFileReader(map[string]string{
		"/test": "content",
	})

	if !reader.FileExists("/test") {
		t.Error("expected file to exist before remove")
	}

	reader.RemoveFile("/test")

	if reader.FileExists("/test") {
		t.Error("expected file NOT to exist after remove")
	}
}

// ============================================================================
// Additional Distribution Fixture Tests
// ============================================================================

func TestDebianDistribution_ReturnsCorrectValues(t *stdtesting.T) {
	dist := DebianDistribution()

	if dist.ID != "debian" {
		t.Errorf("expected ID 'debian', got %s", dist.ID)
	}
	if dist.Family != constants.FamilyDebian {
		t.Errorf("expected family %s, got %s", constants.FamilyDebian, dist.Family)
	}
}

func TestRHELDistribution_ReturnsCorrectValues(t *stdtesting.T) {
	dist := RHELDistribution()

	if dist.ID != "rhel" {
		t.Errorf("expected ID 'rhel', got %s", dist.ID)
	}
	if dist.Family != constants.FamilyRHEL {
		t.Errorf("expected family %s, got %s", constants.FamilyRHEL, dist.Family)
	}
}

// ============================================================================
// Additional Output Fixture Tests
// ============================================================================

func TestNvidiaSMIQueryGPU_HasValidFormat(t *stdtesting.T) {
	output := NvidiaSMIQueryGPU()

	if !strings.Contains(output, "name") {
		t.Error("expected output to contain 'name'")
	}
	if !strings.Contains(output, "driver_version") {
		t.Error("expected output to contain 'driver_version'")
	}
}

func TestLspciOutputWithNouveauDriver_ContainsNouveau(t *stdtesting.T) {
	output := LspciOutputWithNouveauDriver()

	if !strings.Contains(output, "nouveau") {
		t.Error("expected output to contain 'nouveau'")
	}
}

func TestXorgConfPrimeSync_ContainsPrimeConfig(t *stdtesting.T) {
	content := XorgConfPrimeSync()

	if !strings.Contains(content, "PRIME") || !strings.Contains(content, "AllowNVIDIAGPUScreens") {
		t.Error("expected content to contain PRIME configuration")
	}
}
