package kernel

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tungetti/igor/internal/constants"
	"github.com/tungetti/igor/internal/exec"
)

// =============================================================================
// Integration Test: Module States
// =============================================================================

func TestDetector_ModuleStates(t *testing.T) {
	t.Run("nvidia module loaded and in use", func(t *testing.T) {
		mockFS := NewMockFileSystem()
		procModulesContent := `nvidia 55123456 10 nvidia_modeset,nvidia_drm, Live 0xffffffffc0a00000
nvidia_modeset 1234567 1 nvidia_drm, Live 0xffffffffc0600000
nvidia_drm 65536 5 - Live 0xffffffffc0700000
`
		mockFS.AddFile("/proc/modules", []byte(procModulesContent))

		detector := NewDetector(
			WithFileSystem(mockFS),
			WithProcModulesPath("/proc/modules"),
		)

		ctx := context.Background()
		modules, err := detector.GetLoadedModules(ctx)

		require.NoError(t, err)
		assert.Len(t, modules, 3)

		// Check nvidia module
		nvidia := FindModule(modules, "nvidia")
		require.NotNil(t, nvidia)
		assert.Equal(t, int64(55123456), nvidia.Size)
		assert.Equal(t, 10, nvidia.UsedCount)
		assert.Equal(t, "Live", nvidia.State)
		assert.Contains(t, nvidia.UsedBy, "nvidia_modeset")
		assert.Contains(t, nvidia.UsedBy, "nvidia_drm")
	})

	t.Run("nvidia_drm loaded for display", func(t *testing.T) {
		mockFS := NewMockFileSystem()
		procModulesContent := `nvidia_drm 65536 5 - Live 0xffffffffc0700000
nvidia_modeset 1234567 1 nvidia_drm, Live 0xffffffffc0600000
nvidia 55123456 2 nvidia_modeset, Live 0xffffffffc0a00000
`
		mockFS.AddFile("/proc/modules", []byte(procModulesContent))

		detector := NewDetector(
			WithFileSystem(mockFS),
			WithProcModulesPath("/proc/modules"),
		)

		ctx := context.Background()
		loaded, err := detector.IsModuleLoaded(ctx, "nvidia_drm")

		require.NoError(t, err)
		assert.True(t, loaded)

		// Get NVIDIA modules
		modules, _ := detector.GetLoadedModules(ctx)
		nvidiaModules := GetNVIDIAModules(modules)
		assert.Len(t, nvidiaModules, 3)
	})

	t.Run("nouveau loaded instead of nvidia", func(t *testing.T) {
		mockFS := NewMockFileSystem()
		procModulesContent := `nouveau 1234567 2 - Live 0xffffffffc0800000
drm_kms_helper 200000 1 nouveau, Live 0xffffffffc0500000
drm 500000 3 nouveau,drm_kms_helper, Live 0xffffffffc0400000
`
		mockFS.AddFile("/proc/modules", []byte(procModulesContent))

		detector := NewDetector(
			WithFileSystem(mockFS),
			WithProcModulesPath("/proc/modules"),
		)

		ctx := context.Background()

		// Nouveau is loaded
		loaded, err := detector.IsModuleLoaded(ctx, "nouveau")
		require.NoError(t, err)
		assert.True(t, loaded)

		// NVIDIA is not loaded
		loaded, err = detector.IsModuleLoaded(ctx, "nvidia")
		require.NoError(t, err)
		assert.False(t, loaded)

		// No NVIDIA modules
		modules, _ := detector.GetLoadedModules(ctx)
		nvidiaModules := GetNVIDIAModules(modules)
		assert.Empty(t, nvidiaModules)
	})

	t.Run("both nvidia and nouveau loaded - conflict scenario", func(t *testing.T) {
		mockFS := NewMockFileSystem()
		// This shouldn't normally happen, but test handling
		procModulesContent := `nvidia 55123456 0 - Live 0xffffffffc0a00000
nouveau 1234567 0 - Live 0xffffffffc0800000
`
		mockFS.AddFile("/proc/modules", []byte(procModulesContent))

		detector := NewDetector(
			WithFileSystem(mockFS),
			WithProcModulesPath("/proc/modules"),
		)

		ctx := context.Background()

		// Both loaded
		nvidiaLoaded, _ := detector.IsModuleLoaded(ctx, "nvidia")
		nouveauLoaded, _ := detector.IsModuleLoaded(ctx, "nouveau")

		assert.True(t, nvidiaLoaded)
		assert.True(t, nouveauLoaded)
	})

	t.Run("neither nvidia nor nouveau loaded", func(t *testing.T) {
		mockFS := NewMockFileSystem()
		procModulesContent := `i915 2000000 10 - Live 0xffffffffc0900000
drm_kms_helper 200000 1 i915, Live 0xffffffffc0500000
drm 500000 2 i915,drm_kms_helper, Live 0xffffffffc0400000
`
		mockFS.AddFile("/proc/modules", []byte(procModulesContent))

		detector := NewDetector(
			WithFileSystem(mockFS),
			WithProcModulesPath("/proc/modules"),
		)

		ctx := context.Background()

		nvidiaLoaded, _ := detector.IsModuleLoaded(ctx, "nvidia")
		nouveauLoaded, _ := detector.IsModuleLoaded(ctx, "nouveau")

		assert.False(t, nvidiaLoaded)
		assert.False(t, nouveauLoaded)
	})

	t.Run("nvidia module in loading state", func(t *testing.T) {
		mockFS := NewMockFileSystem()
		procModulesContent := `nvidia 55123456 0 - Loading 0xffffffffc0a00000
`
		mockFS.AddFile("/proc/modules", []byte(procModulesContent))

		detector := NewDetector(
			WithFileSystem(mockFS),
			WithProcModulesPath("/proc/modules"),
		)

		ctx := context.Background()
		modules, err := detector.GetLoadedModules(ctx)

		require.NoError(t, err)
		require.Len(t, modules, 1)

		assert.Equal(t, "Loading", modules[0].State)
		liveModules := FilterModulesByState(modules, "Live")
		assert.Empty(t, liveModules)
	})

	t.Run("nvidia module in unloading state", func(t *testing.T) {
		mockFS := NewMockFileSystem()
		procModulesContent := `nvidia 55123456 0 - Unloading 0xffffffffc0a00000
`
		mockFS.AddFile("/proc/modules", []byte(procModulesContent))

		detector := NewDetector(
			WithFileSystem(mockFS),
			WithProcModulesPath("/proc/modules"),
		)

		ctx := context.Background()
		modules, err := detector.GetLoadedModules(ctx)

		require.NoError(t, err)
		require.Len(t, modules, 1)

		assert.Equal(t, "Unloading", modules[0].State)
	})
}

// =============================================================================
// Integration Test: /proc/modules Parsing
// =============================================================================

func TestDetector_ProcModules(t *testing.T) {
	t.Run("typical Ubuntu desktop /proc/modules format", func(t *testing.T) {
		procModulesContent := `nvidia_drm 73728 14 - Live 0xffffffffc10fe000 (POE)
nvidia_modeset 1556480 24 nvidia_drm, Live 0xffffffffc0f7d000 (POE)
nvidia 56745984 1052 nvidia_modeset, Live 0xffffffffc0910000 (POE)
drm_kms_helper 299008 1 nvidia_drm, Live 0xffffffffc0878000
drm 651264 27 nvidia_drm,nvidia_modeset,drm_kms_helper, Live 0xffffffffc07d0000
`
		modules, err := ParseModulesContent([]byte(procModulesContent))

		require.NoError(t, err)
		assert.Len(t, modules, 5)

		// Check nvidia module
		nvidia := FindModule(modules, "nvidia")
		require.NotNil(t, nvidia)
		assert.Equal(t, int64(56745984), nvidia.Size)
		assert.Equal(t, 1052, nvidia.UsedCount)
	})

	t.Run("Fedora /proc/modules format", func(t *testing.T) {
		procModulesContent := `nvidia_drm 77824 12 - Live 0xffffffffc1234000
nvidia_modeset 1495040 17 nvidia_drm, Live 0xffffffffc0f00000
nvidia_uvm 3145728 0 - Live 0xffffffffc0c00000
nvidia 54198272 1068 nvidia_modeset,nvidia_uvm, Live 0xffffffffc0900000
`
		modules, err := ParseModulesContent([]byte(procModulesContent))

		require.NoError(t, err)
		assert.Len(t, modules, 4)

		// nvidia_uvm is used for CUDA
		uvm := FindModule(modules, "nvidia_uvm")
		require.NotNil(t, uvm)
		assert.Equal(t, int64(3145728), uvm.Size)
	})

	t.Run("module with multiple dependencies", func(t *testing.T) {
		procModulesContent := `nvidia 56745984 1052 nvidia_modeset,nvidia_drm,nvidia_uvm, Live 0xffffffffc0910000
`
		modules, err := ParseModulesContent([]byte(procModulesContent))

		require.NoError(t, err)
		require.Len(t, modules, 1)

		nvidia := modules[0]
		assert.Len(t, nvidia.UsedBy, 3)
		assert.Contains(t, nvidia.UsedBy, "nvidia_modeset")
		assert.Contains(t, nvidia.UsedBy, "nvidia_drm")
		assert.Contains(t, nvidia.UsedBy, "nvidia_uvm")
	})

	t.Run("module with no dependencies", func(t *testing.T) {
		procModulesContent := `nvidia_peermem 16384 0 - Live 0xffffffffc1500000
`
		modules, err := ParseModulesContent([]byte(procModulesContent))

		require.NoError(t, err)
		require.Len(t, modules, 1)
		assert.Empty(t, modules[0].UsedBy)
	})

	t.Run("many modules with mixed vendors", func(t *testing.T) {
		procModulesContent := `nvidia_drm 73728 14 - Live 0xffffffffc10fe000
nvidia_modeset 1556480 24 nvidia_drm, Live 0xffffffffc0f7d000
nvidia 56745984 1052 nvidia_modeset, Live 0xffffffffc0910000
i915 3276800 57 - Live 0xffffffffc0600000
snd_hda_intel 57344 5 - Live 0xffffffffc0500000
iwlwifi 413696 1 iwlmvm, Live 0xffffffffc0400000
e1000e 303104 0 - Live 0xffffffffc0300000
xhci_hcd 315392 1 xhci_pci, Live 0xffffffffc0200000
`
		modules, err := ParseModulesContent([]byte(procModulesContent))

		require.NoError(t, err)
		assert.Len(t, modules, 8)

		nvidiaModules := GetNVIDIAModules(modules)
		assert.Len(t, nvidiaModules, 3)
	})
}

// =============================================================================
// Integration Test: Kernel Headers Detection
// =============================================================================

func TestDetector_KernelHeaders(t *testing.T) {
	t.Run("headers in /usr/src path", func(t *testing.T) {
		mockExec := exec.NewMockExecutor()
		mockFS := NewMockFileSystem()

		mockExec.SetResponse("uname", exec.SuccessResult("6.5.0-44-generic\n"))
		mockFS.AddDir("/usr/src/linux-headers-6.5.0-44-generic")

		detector := NewDetector(
			WithExecutor(mockExec),
			WithFileSystem(mockFS),
			WithDistroFamily(constants.FamilyDebian),
		)

		ctx := context.Background()
		installed, err := detector.AreHeadersInstalled(ctx)

		require.NoError(t, err)
		assert.True(t, installed)

		// Get full kernel info
		info, err := detector.GetKernelInfo(ctx)
		require.NoError(t, err)
		assert.True(t, info.HeadersInstalled)
		assert.Equal(t, "/usr/src/linux-headers-6.5.0-44-generic", info.HeadersPath)
	})

	t.Run("headers in /lib/modules/build path", func(t *testing.T) {
		mockExec := exec.NewMockExecutor()
		mockFS := NewMockFileSystem()

		mockExec.SetResponse("uname", exec.SuccessResult("6.6.8-arch1-1\n"))
		// Not in /usr/src
		// But in /lib/modules
		mockFS.AddDir("/lib/modules/6.6.8-arch1-1/build")

		detector := NewDetector(
			WithExecutor(mockExec),
			WithFileSystem(mockFS),
			WithDistroFamily(constants.FamilyArch),
		)

		ctx := context.Background()
		installed, err := detector.AreHeadersInstalled(ctx)

		require.NoError(t, err)
		assert.True(t, installed)
	})

	t.Run("no headers installed", func(t *testing.T) {
		mockExec := exec.NewMockExecutor()
		mockFS := NewMockFileSystem()

		mockExec.SetResponse("uname", exec.SuccessResult("6.5.0-44-generic\n"))
		// Neither path exists

		detector := NewDetector(
			WithExecutor(mockExec),
			WithFileSystem(mockFS),
		)

		ctx := context.Background()
		installed, err := detector.AreHeadersInstalled(ctx)

		require.NoError(t, err)
		assert.False(t, installed)
	})

	t.Run("Debian headers package name", func(t *testing.T) {
		mockExec := exec.NewMockExecutor()
		mockExec.SetResponse("uname", exec.SuccessResult("6.5.0-44-generic\n"))

		detector := NewDetector(
			WithExecutor(mockExec),
			WithDistroFamily(constants.FamilyDebian),
		)

		ctx := context.Background()
		pkg, err := detector.GetHeadersPackage(ctx)

		require.NoError(t, err)
		assert.Equal(t, "linux-headers-6.5.0-44-generic", pkg)
	})

	t.Run("RHEL headers package name", func(t *testing.T) {
		mockExec := exec.NewMockExecutor()
		mockExec.SetResponse("uname", exec.SuccessResult("5.14.0-284.el9.x86_64\n"))

		detector := NewDetector(
			WithExecutor(mockExec),
			WithDistroFamily(constants.FamilyRHEL),
		)

		ctx := context.Background()
		pkg, err := detector.GetHeadersPackage(ctx)

		require.NoError(t, err)
		assert.Equal(t, "kernel-devel-5.14.0-284.el9.x86_64", pkg)
	})

	t.Run("Arch Linux headers package name", func(t *testing.T) {
		mockExec := exec.NewMockExecutor()
		mockExec.SetResponse("uname", exec.SuccessResult("6.6.8-arch1-1\n"))

		detector := NewDetector(
			WithExecutor(mockExec),
			WithDistroFamily(constants.FamilyArch),
		)

		ctx := context.Background()
		pkg, err := detector.GetHeadersPackage(ctx)

		require.NoError(t, err)
		assert.Equal(t, "linux-headers", pkg)
	})

	t.Run("Arch Linux LTS headers package name", func(t *testing.T) {
		mockExec := exec.NewMockExecutor()
		mockExec.SetResponse("uname", exec.SuccessResult("6.1.69-lts-1\n"))

		detector := NewDetector(
			WithExecutor(mockExec),
			WithDistroFamily(constants.FamilyArch),
		)

		ctx := context.Background()
		pkg, err := detector.GetHeadersPackage(ctx)

		require.NoError(t, err)
		assert.Equal(t, "linux-lts-headers", pkg)
	})
}

// =============================================================================
// Integration Test: Secure Boot Detection
// =============================================================================

func TestDetector_SecureBootDetection(t *testing.T) {
	t.Run("secure boot enabled via mokutil", func(t *testing.T) {
		mockExec := exec.NewMockExecutor()
		mockExec.SetResponse("uname", exec.SuccessResult("6.5.0-44-generic\n"))
		mockExec.SetResponse("mokutil", exec.SuccessResult("SecureBoot enabled\n"))

		detector := NewDetector(WithExecutor(mockExec))

		ctx := context.Background()
		enabled, err := detector.IsSecureBootEnabled(ctx)

		require.NoError(t, err)
		assert.True(t, enabled)
	})

	t.Run("secure boot disabled via mokutil", func(t *testing.T) {
		mockExec := exec.NewMockExecutor()
		mockExec.SetResponse("uname", exec.SuccessResult("6.5.0-44-generic\n"))
		mockExec.SetResponse("mokutil", exec.SuccessResult("SecureBoot disabled\n"))

		detector := NewDetector(WithExecutor(mockExec))

		ctx := context.Background()
		enabled, err := detector.IsSecureBootEnabled(ctx)

		require.NoError(t, err)
		assert.False(t, enabled)
	})

	t.Run("secure boot enabled via EFI variable", func(t *testing.T) {
		mockFS := NewMockFileSystem()

		// Add EFI variable
		mockFS.AddDirEntry("/sys/firmware/efi/efivars", &mockDirEntry{
			name: "SecureBoot-8be4df61-93ca-11d2-aa0d-00e098032b8c",
		})
		// EFI variable: 4 bytes attribute + 1 byte value (0x01 = enabled)
		mockFS.AddFile(
			"/sys/firmware/efi/efivars/SecureBoot-8be4df61-93ca-11d2-aa0d-00e098032b8c",
			[]byte{0x06, 0x00, 0x00, 0x00, 0x01},
		)

		detector := NewDetector(
			WithFileSystem(mockFS),
			WithEFIVarsPath("/sys/firmware/efi/efivars"),
		)

		ctx := context.Background()
		enabled, err := detector.IsSecureBootEnabled(ctx)

		require.NoError(t, err)
		assert.True(t, enabled)
	})

	t.Run("legacy BIOS system - no EFI", func(t *testing.T) {
		mockFS := NewMockFileSystem()
		// No EFI vars directory

		detector := NewDetector(
			WithFileSystem(mockFS),
			WithEFIVarsPath("/sys/firmware/efi/efivars"),
		)

		ctx := context.Background()
		enabled, err := detector.IsSecureBootEnabled(ctx)

		require.NoError(t, err)
		assert.False(t, enabled)
	})
}

// =============================================================================
// Integration Test: Complete Kernel Info
// =============================================================================

func TestDetector_CompleteKernelInfo(t *testing.T) {
	t.Run("Ubuntu 22.04 desktop", func(t *testing.T) {
		mockExec := exec.NewMockExecutor()
		mockFS := NewMockFileSystem()

		mockExec.SetResponse("uname", exec.SuccessResult("6.5.0-44-generic\n"))
		mockExec.SetResponse("mokutil", exec.SuccessResult("SecureBoot disabled\n"))
		mockFS.AddDir("/usr/src/linux-headers-6.5.0-44-generic")

		detector := NewDetector(
			WithExecutor(mockExec),
			WithFileSystem(mockFS),
			WithDistroFamily(constants.FamilyDebian),
		)

		ctx := context.Background()
		info, err := detector.GetKernelInfo(ctx)

		require.NoError(t, err)
		assert.Equal(t, "6.5.0-44-generic", info.Version)
		assert.Equal(t, "6.5.0", info.Release)
		assert.True(t, info.HeadersInstalled)
		assert.False(t, info.SecureBootEnabled)
	})

	t.Run("Fedora 39 workstation", func(t *testing.T) {
		mockExec := exec.NewMockExecutor()
		mockFS := NewMockFileSystem()

		mockExec.SetResponse("uname", exec.SuccessResult("6.6.8-200.fc39.x86_64\n"))
		mockExec.SetResponse("mokutil", exec.SuccessResult("SecureBoot enabled\n"))
		mockFS.AddDir("/lib/modules/6.6.8-200.fc39.x86_64/build")

		detector := NewDetector(
			WithExecutor(mockExec),
			WithFileSystem(mockFS),
			WithDistroFamily(constants.FamilyRHEL),
		)

		ctx := context.Background()
		info, err := detector.GetKernelInfo(ctx)

		require.NoError(t, err)
		assert.Equal(t, "6.6.8-200.fc39.x86_64", info.Version)
		assert.Equal(t, "6.6.8", info.Release)
		assert.True(t, info.HeadersInstalled)
		assert.True(t, info.SecureBootEnabled)
	})

	t.Run("Arch Linux latest", func(t *testing.T) {
		mockExec := exec.NewMockExecutor()
		mockFS := NewMockFileSystem()

		mockExec.SetResponse("uname", exec.SuccessResult("6.7.1-arch1-1\n"))
		mockExec.SetResponse("mokutil", exec.SuccessResult("SecureBoot disabled\n"))
		mockFS.AddDir("/lib/modules/6.7.1-arch1-1/build")

		detector := NewDetector(
			WithExecutor(mockExec),
			WithFileSystem(mockFS),
			WithDistroFamily(constants.FamilyArch),
		)

		ctx := context.Background()
		info, err := detector.GetKernelInfo(ctx)

		require.NoError(t, err)
		assert.Equal(t, "6.7.1-arch1-1", info.Version)
		assert.Equal(t, "6.7.1", info.Release)
		assert.True(t, info.HeadersInstalled)
	})
}

// =============================================================================
// Benchmarks
// =============================================================================

func BenchmarkDetector_IsLoaded(b *testing.B) {
	mockFS := NewMockFileSystem()
	procModulesContent := `nvidia 55123456 10 - Live 0xffffffffc0a00000
nvidia_drm 65536 5 nvidia - Live 0xffffffffc0700000
nouveau 1234567 0 - Live 0xffffffffc0800000
`
	mockFS.AddFile("/proc/modules", []byte(procModulesContent))

	detector := NewDetector(
		WithFileSystem(mockFS),
		WithProcModulesPath("/proc/modules"),
	)

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = detector.IsModuleLoaded(ctx, "nvidia")
	}
}

func BenchmarkDetector_GetLoadedModules(b *testing.B) {
	mockFS := NewMockFileSystem()
	procModulesContent := `nvidia 55123456 10 - Live 0xffffffffc0a00000
nvidia_drm 65536 5 nvidia - Live 0xffffffffc0700000
nvidia_modeset 1234567 1 nvidia_drm - Live 0xffffffffc0600000
i915 2000000 10 - Live 0xffffffffc0500000
drm 500000 3 nvidia_drm,i915 - Live 0xffffffffc0400000
`
	mockFS.AddFile("/proc/modules", []byte(procModulesContent))

	detector := NewDetector(
		WithFileSystem(mockFS),
		WithProcModulesPath("/proc/modules"),
	)

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = detector.GetLoadedModules(ctx)
	}
}

func BenchmarkParseModulesContent(b *testing.B) {
	content := []byte(`nvidia 55123456 10 - Live 0xffffffffc0a00000
nvidia_drm 65536 5 nvidia - Live 0xffffffffc0700000
nvidia_modeset 1234567 1 nvidia_drm - Live 0xffffffffc0600000
i915 2000000 10 - Live 0xffffffffc0500000
drm 500000 3 nvidia_drm,i915 - Live 0xffffffffc0400000
drm_kms_helper 200000 2 nvidia_drm,i915 - Live 0xffffffffc0300000
snd_hda_intel 57344 5 - Live 0xffffffffc0200000
xhci_hcd 315392 1 xhci_pci - Live 0xffffffffc0100000
`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ParseModulesContent(content)
	}
}

func BenchmarkFindModule(b *testing.B) {
	modules := []ModuleInfo{
		{Name: "nvidia", Size: 55123456},
		{Name: "nvidia_drm", Size: 65536},
		{Name: "nvidia_modeset", Size: 1234567},
		{Name: "i915", Size: 2000000},
		{Name: "drm", Size: 500000},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = FindModule(modules, "nvidia")
	}
}

func BenchmarkGetNVIDIAModules(b *testing.B) {
	modules := []ModuleInfo{
		{Name: "nvidia", Size: 55123456},
		{Name: "nvidia_drm", Size: 65536},
		{Name: "nvidia_modeset", Size: 1234567},
		{Name: "i915", Size: 2000000},
		{Name: "drm", Size: 500000},
		{Name: "snd_hda_intel", Size: 57344},
		{Name: "iwlwifi", Size: 413696},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = GetNVIDIAModules(modules)
	}
}

func BenchmarkExtractRelease(b *testing.B) {
	versions := []string{
		"6.5.0-44-generic",
		"5.15.0-91-generic",
		"6.6.8-arch1-1",
		"5.14.0-284.el9.x86_64",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, v := range versions {
			_ = extractRelease(v)
		}
	}
}
