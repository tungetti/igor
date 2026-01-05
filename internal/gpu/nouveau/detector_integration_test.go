package nouveau

import (
	"context"
	"io/fs"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tungetti/igor/internal/gpu/pci"
)

// =============================================================================
// Integration Test: Nouveau States
// =============================================================================

func TestDetector_NouveauStates(t *testing.T) {
	t.Run("nouveau loaded and in use by GPU", func(t *testing.T) {
		const modulePath = "/sys/module/nouveau"
		const modprobePath = "/etc/modprobe.d"

		mockFS := NewMockFileSystem()
		mockFS.Stats[modulePath] = true
		mockFS.Dirs[modprobePath] = []fs.DirEntry{}

		mockScanner := &MockPCIScanner{
			Devices: []pci.PCIDevice{
				{
					Address:  "0000:01:00.0",
					VendorID: "10de",
					DeviceID: "2684",
					Class:    "030000",
					Driver:   "nouveau",
				},
			},
		}

		detector := NewDetector(
			WithFileSystem(mockFS),
			WithPCIScanner(mockScanner),
			WithModulePath(modulePath),
			WithModprobePath(modprobePath),
		)

		ctx := context.Background()
		status, err := detector.Detect(ctx)

		require.NoError(t, err)
		assert.True(t, status.Loaded, "nouveau should be loaded")
		assert.True(t, status.InUse, "nouveau should be in use")
		assert.Len(t, status.BoundDevices, 1)
		assert.Equal(t, "0000:01:00.0", status.BoundDevices[0])
		assert.False(t, status.BlacklistExists, "nouveau should not be blacklisted")
	})

	t.Run("nouveau blacklisted but still loaded", func(t *testing.T) {
		const modulePath = "/sys/module/nouveau"
		const modprobePath = "/etc/modprobe.d"

		mockFS := NewMockFileSystem()
		mockFS.Stats[modulePath] = true
		mockFS.Dirs[modprobePath] = []fs.DirEntry{
			mockDirEntry{name: "blacklist-nouveau.conf", isDir: false},
		}
		mockFS.Files[filepath.Join(modprobePath, "blacklist-nouveau.conf")] = `# Blacklist nouveau for NVIDIA proprietary driver
blacklist nouveau
options nouveau modeset=0
`

		mockScanner := &MockPCIScanner{
			Devices: []pci.PCIDevice{
				{
					Address:  "0000:01:00.0",
					VendorID: "10de",
					DeviceID: "2684",
					Class:    "030000",
					Driver:   "nvidia",
				},
			},
		}

		detector := NewDetector(
			WithFileSystem(mockFS),
			WithPCIScanner(mockScanner),
			WithModulePath(modulePath),
			WithModprobePath(modprobePath),
		)

		ctx := context.Background()
		status, err := detector.Detect(ctx)

		require.NoError(t, err)
		assert.True(t, status.Loaded, "nouveau module is still loaded")
		assert.False(t, status.InUse, "nouveau should not be in use")
		assert.Empty(t, status.BoundDevices)
		assert.True(t, status.BlacklistExists, "nouveau should be blacklisted")
		assert.Len(t, status.BlacklistFiles, 1)
	})

	t.Run("nouveau completely disabled - initramfs scenario", func(t *testing.T) {
		const modulePath = "/sys/module/nouveau"
		const modprobePath = "/etc/modprobe.d"

		mockFS := NewMockFileSystem()
		// Module not in stats = not loaded
		mockFS.Dirs[modprobePath] = []fs.DirEntry{
			mockDirEntry{name: "nvidia-installer.conf", isDir: false},
			mockDirEntry{name: "blacklist-nouveau.conf", isDir: false},
		}
		mockFS.Files[filepath.Join(modprobePath, "nvidia-installer.conf")] = `# NVIDIA installer generated blacklist
blacklist nouveau
install nouveau /bin/false
`
		mockFS.Files[filepath.Join(modprobePath, "blacklist-nouveau.conf")] = `options nouveau modeset=0
`

		mockScanner := &MockPCIScanner{
			Devices: []pci.PCIDevice{
				{
					Address:  "0000:01:00.0",
					VendorID: "10de",
					DeviceID: "2684",
					Class:    "030000",
					Driver:   "nvidia",
				},
			},
		}

		detector := NewDetector(
			WithFileSystem(mockFS),
			WithPCIScanner(mockScanner),
			WithModulePath(modulePath),
			WithModprobePath(modprobePath),
		)

		ctx := context.Background()
		status, err := detector.Detect(ctx)

		require.NoError(t, err)
		assert.False(t, status.Loaded, "nouveau should not be loaded")
		assert.False(t, status.InUse, "nouveau should not be in use")
		assert.Empty(t, status.BoundDevices)
		assert.True(t, status.BlacklistExists)
		assert.Len(t, status.BlacklistFiles, 2)
	})

	t.Run("multi-GPU with nouveau on one card", func(t *testing.T) {
		const modulePath = "/sys/module/nouveau"
		const modprobePath = "/etc/modprobe.d"

		mockFS := NewMockFileSystem()
		mockFS.Stats[modulePath] = true
		mockFS.Dirs[modprobePath] = []fs.DirEntry{}

		mockScanner := &MockPCIScanner{
			Devices: []pci.PCIDevice{
				{
					Address:  "0000:01:00.0",
					VendorID: "10de",
					DeviceID: "2684",
					Class:    "030000",
					Driver:   "nouveau",
				},
				{
					Address:  "0000:02:00.0",
					VendorID: "10de",
					DeviceID: "2684",
					Class:    "030000",
					Driver:   "nvidia",
				},
				{
					Address:  "0000:00:02.0",
					VendorID: "8086",
					DeviceID: "9a49",
					Class:    "030000",
					Driver:   "i915",
				},
			},
		}

		detector := NewDetector(
			WithFileSystem(mockFS),
			WithPCIScanner(mockScanner),
			WithModulePath(modulePath),
			WithModprobePath(modprobePath),
		)

		ctx := context.Background()
		status, err := detector.Detect(ctx)

		require.NoError(t, err)
		assert.True(t, status.Loaded)
		assert.True(t, status.InUse)
		assert.Len(t, status.BoundDevices, 1)
		assert.Contains(t, status.BoundDevices, "0000:01:00.0")
	})

	t.Run("fresh system without any GPU driver", func(t *testing.T) {
		const modulePath = "/sys/module/nouveau"
		const modprobePath = "/etc/modprobe.d"

		mockFS := NewMockFileSystem()
		// No nouveau module loaded
		mockFS.Dirs[modprobePath] = []fs.DirEntry{}

		mockScanner := &MockPCIScanner{
			Devices: []pci.PCIDevice{
				{
					Address:  "0000:01:00.0",
					VendorID: "10de",
					DeviceID: "2684",
					Class:    "030000",
					Driver:   "", // No driver bound
				},
			},
		}

		detector := NewDetector(
			WithFileSystem(mockFS),
			WithPCIScanner(mockScanner),
			WithModulePath(modulePath),
			WithModprobePath(modprobePath),
		)

		ctx := context.Background()
		status, err := detector.Detect(ctx)

		require.NoError(t, err)
		assert.False(t, status.Loaded)
		assert.False(t, status.InUse)
		assert.Empty(t, status.BoundDevices)
		assert.False(t, status.BlacklistExists)
	})
}

// =============================================================================
// Integration Test: Blacklist Detection
// =============================================================================

func TestDetector_BlacklistDetection(t *testing.T) {
	t.Run("standard blacklist format", func(t *testing.T) {
		const modprobePath = "/etc/modprobe.d"

		mockFS := NewMockFileSystem()
		mockFS.Dirs[modprobePath] = []fs.DirEntry{
			mockDirEntry{name: "blacklist-nouveau.conf", isDir: false},
		}
		mockFS.Files[filepath.Join(modprobePath, "blacklist-nouveau.conf")] = "blacklist nouveau\n"

		detector := NewDetector(WithFileSystem(mockFS), WithModprobePath(modprobePath))

		ctx := context.Background()
		blacklisted, err := detector.IsBlacklisted(ctx)

		require.NoError(t, err)
		assert.True(t, blacklisted)
	})

	t.Run("modeset=0 as blacklist", func(t *testing.T) {
		const modprobePath = "/etc/modprobe.d"

		mockFS := NewMockFileSystem()
		mockFS.Dirs[modprobePath] = []fs.DirEntry{
			mockDirEntry{name: "nvidia.conf", isDir: false},
		}
		mockFS.Files[filepath.Join(modprobePath, "nvidia.conf")] = "options nouveau modeset=0\n"

		detector := NewDetector(WithFileSystem(mockFS), WithModprobePath(modprobePath))

		ctx := context.Background()
		blacklisted, err := detector.IsBlacklisted(ctx)

		require.NoError(t, err)
		assert.True(t, blacklisted)
	})

	t.Run("distro-specific blacklist files", func(t *testing.T) {
		const modprobePath = "/etc/modprobe.d"

		mockFS := NewMockFileSystem()
		mockFS.Dirs[modprobePath] = []fs.DirEntry{
			mockDirEntry{name: "nvidia-graphics-drivers.conf", isDir: false},
			mockDirEntry{name: "nvidia-kernel-common.conf", isDir: false},
		}
		mockFS.Files[filepath.Join(modprobePath, "nvidia-graphics-drivers.conf")] = `# Ubuntu NVIDIA driver blacklist
blacklist nouveau
blacklist lbm-nouveau
options nouveau modeset=0
alias nouveau off
alias lbm-nouveau off
`
		mockFS.Files[filepath.Join(modprobePath, "nvidia-kernel-common.conf")] = `# Fedora NVIDIA kmod blacklist
blacklist nouveau
`

		detector := NewDetector(WithFileSystem(mockFS), WithModprobePath(modprobePath))

		ctx := context.Background()
		blacklisted, err := detector.IsBlacklisted(ctx)

		require.NoError(t, err)
		assert.True(t, blacklisted)
	})

	t.Run("commented blacklist should not match", func(t *testing.T) {
		const modprobePath = "/etc/modprobe.d"

		mockFS := NewMockFileSystem()
		mockFS.Dirs[modprobePath] = []fs.DirEntry{
			mockDirEntry{name: "example.conf", isDir: false},
		}
		mockFS.Files[filepath.Join(modprobePath, "example.conf")] = `# Example configuration
# blacklist nouveau
# options nouveau modeset=0
`

		detector := NewDetector(WithFileSystem(mockFS), WithModprobePath(modprobePath))

		ctx := context.Background()
		blacklisted, err := detector.IsBlacklisted(ctx)

		require.NoError(t, err)
		assert.False(t, blacklisted)
	})

	t.Run("mixed valid and commented entries", func(t *testing.T) {
		const modprobePath = "/etc/modprobe.d"

		mockFS := NewMockFileSystem()
		mockFS.Dirs[modprobePath] = []fs.DirEntry{
			mockDirEntry{name: "blacklist.conf", isDir: false},
		}
		mockFS.Files[filepath.Join(modprobePath, "blacklist.conf")] = `# This is commented:
# blacklist nouveau
# This is active:
blacklist nouveau
`

		detector := NewDetector(WithFileSystem(mockFS), WithModprobePath(modprobePath))

		ctx := context.Background()
		blacklisted, err := detector.IsBlacklisted(ctx)

		require.NoError(t, err)
		assert.True(t, blacklisted)
	})

	t.Run("extra whitespace handling", func(t *testing.T) {
		const modprobePath = "/etc/modprobe.d"

		mockFS := NewMockFileSystem()
		mockFS.Dirs[modprobePath] = []fs.DirEntry{
			mockDirEntry{name: "blacklist.conf", isDir: false},
		}
		mockFS.Files[filepath.Join(modprobePath, "blacklist.conf")] = "   blacklist    nouveau   \n"

		detector := NewDetector(WithFileSystem(mockFS), WithModprobePath(modprobePath))

		ctx := context.Background()
		blacklisted, err := detector.IsBlacklisted(ctx)

		require.NoError(t, err)
		assert.True(t, blacklisted)
	})

	t.Run("tabs instead of spaces", func(t *testing.T) {
		const modprobePath = "/etc/modprobe.d"

		mockFS := NewMockFileSystem()
		mockFS.Dirs[modprobePath] = []fs.DirEntry{
			mockDirEntry{name: "blacklist.conf", isDir: false},
		}
		mockFS.Files[filepath.Join(modprobePath, "blacklist.conf")] = "blacklist\tnouveau\n"

		detector := NewDetector(WithFileSystem(mockFS), WithModprobePath(modprobePath))

		ctx := context.Background()
		blacklisted, err := detector.IsBlacklisted(ctx)

		require.NoError(t, err)
		assert.True(t, blacklisted)
	})

	t.Run("no modprobe.d directory", func(t *testing.T) {
		const modprobePath = "/etc/modprobe.d"

		mockFS := NewMockFileSystem()
		// Don't add modprobePath to Dirs - simulates non-existent directory

		detector := NewDetector(WithFileSystem(mockFS), WithModprobePath(modprobePath))

		ctx := context.Background()
		blacklisted, err := detector.IsBlacklisted(ctx)

		require.NoError(t, err)
		assert.False(t, blacklisted)
	})

	t.Run("empty modprobe.d directory", func(t *testing.T) {
		const modprobePath = "/etc/modprobe.d"

		mockFS := NewMockFileSystem()
		mockFS.Dirs[modprobePath] = []fs.DirEntry{}

		detector := NewDetector(WithFileSystem(mockFS), WithModprobePath(modprobePath))

		ctx := context.Background()
		blacklisted, err := detector.IsBlacklisted(ctx)

		require.NoError(t, err)
		assert.False(t, blacklisted)
	})
}

// =============================================================================
// Integration Test: Real-World Scenarios
// =============================================================================

func TestDetector_RealWorldScenarios(t *testing.T) {
	t.Run("NVIDIA driver installation in progress", func(t *testing.T) {
		const modulePath = "/sys/module/nouveau"
		const modprobePath = "/etc/modprobe.d"

		mockFS := NewMockFileSystem()
		// Nouveau still loaded but blacklisted
		mockFS.Stats[modulePath] = true
		mockFS.Dirs[modprobePath] = []fs.DirEntry{
			mockDirEntry{name: "nvidia-installer-disable-nouveau.conf", isDir: false},
		}
		mockFS.Files[filepath.Join(modprobePath, "nvidia-installer-disable-nouveau.conf")] = `# Generated by nvidia-installer
blacklist nouveau
options nouveau modeset=0
`

		mockScanner := &MockPCIScanner{
			Devices: []pci.PCIDevice{
				{
					Address:  "0000:01:00.0",
					VendorID: "10de",
					DeviceID: "2684",
					Class:    "030000",
					Driver:   "nouveau", // Still bound before reboot
				},
			},
		}

		detector := NewDetector(
			WithFileSystem(mockFS),
			WithPCIScanner(mockScanner),
			WithModulePath(modulePath),
			WithModprobePath(modprobePath),
		)

		ctx := context.Background()
		status, err := detector.Detect(ctx)

		require.NoError(t, err)
		assert.True(t, status.Loaded)
		assert.True(t, status.InUse)
		assert.True(t, status.BlacklistExists)
		// System needs reboot to complete transition
	})

	t.Run("laptop with Optimus - integrated GPU only mode", func(t *testing.T) {
		const modulePath = "/sys/module/nouveau"
		const modprobePath = "/etc/modprobe.d"

		mockFS := NewMockFileSystem()
		mockFS.Dirs[modprobePath] = []fs.DirEntry{
			mockDirEntry{name: "optimus.conf", isDir: false},
		}
		mockFS.Files[filepath.Join(modprobePath, "optimus.conf")] = `# Disable discrete GPU
blacklist nouveau
options nouveau modeset=0
`

		mockScanner := &MockPCIScanner{
			Devices: []pci.PCIDevice{
				{
					Address:  "0000:00:02.0",
					VendorID: "8086",
					DeviceID: "9a49",
					Class:    "030000",
					Driver:   "i915",
				},
				{
					Address:  "0000:01:00.0",
					VendorID: "10de",
					DeviceID: "2560",
					Class:    "030200",
					Driver:   "", // Disabled
				},
			},
		}

		detector := NewDetector(
			WithFileSystem(mockFS),
			WithPCIScanner(mockScanner),
			WithModulePath(modulePath),
			WithModprobePath(modprobePath),
		)

		ctx := context.Background()
		status, err := detector.Detect(ctx)

		require.NoError(t, err)
		assert.False(t, status.Loaded)
		assert.False(t, status.InUse)
		assert.True(t, status.BlacklistExists)
	})

	t.Run("VFIO passthrough configuration", func(t *testing.T) {
		const modulePath = "/sys/module/nouveau"
		const modprobePath = "/etc/modprobe.d"

		mockFS := NewMockFileSystem()
		mockFS.Dirs[modprobePath] = []fs.DirEntry{
			mockDirEntry{name: "vfio.conf", isDir: false},
		}
		mockFS.Files[filepath.Join(modprobePath, "vfio.conf")] = `# VFIO passthrough
blacklist nouveau
blacklist nvidia
options vfio-pci ids=10de:2684
`

		mockScanner := &MockPCIScanner{
			Devices: []pci.PCIDevice{
				{
					Address:  "0000:01:00.0",
					VendorID: "10de",
					DeviceID: "2684",
					Class:    "030000",
					Driver:   "vfio-pci",
				},
			},
		}

		detector := NewDetector(
			WithFileSystem(mockFS),
			WithPCIScanner(mockScanner),
			WithModulePath(modulePath),
			WithModprobePath(modprobePath),
		)

		ctx := context.Background()
		status, err := detector.Detect(ctx)

		require.NoError(t, err)
		assert.False(t, status.Loaded)
		assert.False(t, status.InUse)
		assert.True(t, status.BlacklistExists)
	})

	t.Run("legacy nouveau with multiple conf files", func(t *testing.T) {
		const modulePath = "/sys/module/nouveau"
		const modprobePath = "/etc/modprobe.d"

		mockFS := NewMockFileSystem()
		mockFS.Stats[modulePath] = true
		mockFS.Dirs[modprobePath] = []fs.DirEntry{
			mockDirEntry{name: "nvidia-blacklists-nouveau.conf", isDir: false},
			mockDirEntry{name: "nouveau-blacklist.conf", isDir: false},
			mockDirEntry{name: "nvidia-modprobe.conf", isDir: false},
		}
		mockFS.Files[filepath.Join(modprobePath, "nvidia-blacklists-nouveau.conf")] = "blacklist nouveau\n"
		mockFS.Files[filepath.Join(modprobePath, "nouveau-blacklist.conf")] = "options nouveau modeset=0\n"
		mockFS.Files[filepath.Join(modprobePath, "nvidia-modprobe.conf")] = "alias char-major-195 nvidia\n"

		mockScanner := &MockPCIScanner{
			Devices: []pci.PCIDevice{},
		}

		detector := NewDetector(
			WithFileSystem(mockFS),
			WithPCIScanner(mockScanner),
			WithModulePath(modulePath),
			WithModprobePath(modprobePath),
		)

		ctx := context.Background()
		status, err := detector.Detect(ctx)

		require.NoError(t, err)
		assert.True(t, status.BlacklistExists)
		assert.Len(t, status.BlacklistFiles, 2) // Two files contain blacklist patterns
	})
}

// =============================================================================
// Benchmarks
// =============================================================================

func BenchmarkDetector_Detect(b *testing.B) {
	const modulePath = "/sys/module/nouveau"
	const modprobePath = "/etc/modprobe.d"

	mockFS := NewMockFileSystem()
	mockFS.Stats[modulePath] = true
	mockFS.Dirs[modprobePath] = []fs.DirEntry{
		mockDirEntry{name: "blacklist-nouveau.conf", isDir: false},
	}
	mockFS.Files[filepath.Join(modprobePath, "blacklist-nouveau.conf")] = "blacklist nouveau\n"

	mockScanner := &MockPCIScanner{
		Devices: []pci.PCIDevice{
			{Address: "0000:01:00.0", VendorID: "10de", DeviceID: "2684", Class: "030000", Driver: "nvidia"},
		},
	}

	detector := NewDetector(
		WithFileSystem(mockFS),
		WithPCIScanner(mockScanner),
		WithModulePath(modulePath),
		WithModprobePath(modprobePath),
	)

	ctx := context.Background()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = detector.Detect(ctx)
	}
}

func BenchmarkDetector_IsLoaded(b *testing.B) {
	const modulePath = "/sys/module/nouveau"

	mockFS := NewMockFileSystem()
	mockFS.Stats[modulePath] = true

	detector := NewDetector(
		WithFileSystem(mockFS),
		WithModulePath(modulePath),
	)

	ctx := context.Background()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = detector.IsLoaded(ctx)
	}
}

func BenchmarkDetector_IsBlacklisted(b *testing.B) {
	const modprobePath = "/etc/modprobe.d"

	mockFS := NewMockFileSystem()
	mockFS.Dirs[modprobePath] = []fs.DirEntry{
		mockDirEntry{name: "blacklist-nouveau.conf", isDir: false},
		mockDirEntry{name: "nvidia.conf", isDir: false},
		mockDirEntry{name: "other.conf", isDir: false},
	}
	mockFS.Files[filepath.Join(modprobePath, "blacklist-nouveau.conf")] = "blacklist nouveau\n"
	mockFS.Files[filepath.Join(modprobePath, "nvidia.conf")] = "options nouveau modeset=0\n"
	mockFS.Files[filepath.Join(modprobePath, "other.conf")] = "# some other config\n"

	detector := NewDetector(
		WithFileSystem(mockFS),
		WithModprobePath(modprobePath),
	)

	ctx := context.Background()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = detector.IsBlacklisted(ctx)
	}
}

func BenchmarkDetector_GetBoundDevices(b *testing.B) {
	mockScanner := &MockPCIScanner{
		Devices: []pci.PCIDevice{
			{Address: "0000:01:00.0", VendorID: "10de", DeviceID: "2684", Class: "030000", Driver: "nouveau"},
			{Address: "0000:02:00.0", VendorID: "10de", DeviceID: "1234", Class: "030200", Driver: "nvidia"},
			{Address: "0000:03:00.0", VendorID: "8086", DeviceID: "5678", Class: "030000", Driver: "i915"},
		},
	}

	detector := NewDetector(WithPCIScanner(mockScanner))

	ctx := context.Background()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = detector.GetBoundDevices(ctx)
	}
}

func BenchmarkDetector_ManyConfFiles(b *testing.B) {
	const modprobePath = "/etc/modprobe.d"

	mockFS := NewMockFileSystem()

	// Create 50 conf files to simulate a realistic system
	entries := make([]fs.DirEntry, 50)
	for i := 0; i < 50; i++ {
		name := filepath.Join(modprobePath, "config-"+string(rune('a'+i%26))+".conf")
		entries[i] = mockDirEntry{name: "config-" + string(rune('a'+i%26)) + ".conf", isDir: false}
		mockFS.Files[name] = "# Some other module config\n"
	}

	// Add the actual blacklist file
	entries = append(entries, mockDirEntry{name: "blacklist-nouveau.conf", isDir: false})
	mockFS.Files[filepath.Join(modprobePath, "blacklist-nouveau.conf")] = "blacklist nouveau\n"
	mockFS.Dirs[modprobePath] = entries

	detector := NewDetector(
		WithFileSystem(mockFS),
		WithModprobePath(modprobePath),
	)

	ctx := context.Background()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = detector.IsBlacklisted(ctx)
	}
}

func BenchmarkContainsBlacklistPattern(b *testing.B) {
	contents := [][]byte{
		[]byte("blacklist nouveau\n"),
		[]byte("options nouveau modeset=0\n"),
		[]byte("# Comment\nblacklist some_other_module\nalias some_module off\n"),
		[]byte(""),
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		for _, content := range contents {
			_ = containsBlacklistPattern(content)
		}
	}
}
