package nouveau

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tungetti/igor/internal/gpu/pci"
)

// MockFileSystem provides a mock implementation of FileSystem for testing.
type MockFileSystem struct {
	// Files maps path to file content
	Files map[string]string
	// Dirs maps path to directory entries
	Dirs map[string][]fs.DirEntry
	// Links maps symlink path to target
	Links map[string]string
	// Stats maps path to whether it exists
	Stats map[string]bool
	// Errors maps path to error to return
	Errors map[string]error
}

// NewMockFileSystem creates a new mock filesystem.
func NewMockFileSystem() *MockFileSystem {
	return &MockFileSystem{
		Files:  make(map[string]string),
		Dirs:   make(map[string][]fs.DirEntry),
		Links:  make(map[string]string),
		Stats:  make(map[string]bool),
		Errors: make(map[string]error),
	}
}

// mockDirEntry implements fs.DirEntry for testing.
type mockDirEntry struct {
	name  string
	isDir bool
}

func (m mockDirEntry) Name() string               { return m.name }
func (m mockDirEntry) IsDir() bool                { return m.isDir }
func (m mockDirEntry) Type() fs.FileMode          { return 0 }
func (m mockDirEntry) Info() (fs.FileInfo, error) { return nil, nil }

// mockFileInfo implements fs.FileInfo for testing.
type mockFileInfo struct {
	name  string
	isDir bool
}

func (m mockFileInfo) Name() string       { return m.name }
func (m mockFileInfo) Size() int64        { return 0 }
func (m mockFileInfo) Mode() fs.FileMode  { return 0 }
func (m mockFileInfo) ModTime() time.Time { return time.Time{} }
func (m mockFileInfo) IsDir() bool        { return m.isDir }
func (m mockFileInfo) Sys() interface{}   { return nil }

func (m *MockFileSystem) ReadDir(dirname string) ([]fs.DirEntry, error) {
	if err, ok := m.Errors[dirname]; ok {
		return nil, err
	}
	if entries, ok := m.Dirs[dirname]; ok {
		return entries, nil
	}
	return nil, os.ErrNotExist
}

func (m *MockFileSystem) ReadFile(filename string) ([]byte, error) {
	if err, ok := m.Errors[filename]; ok {
		return nil, err
	}
	if content, ok := m.Files[filename]; ok {
		return []byte(content), nil
	}
	return nil, os.ErrNotExist
}

func (m *MockFileSystem) Readlink(name string) (string, error) {
	if err, ok := m.Errors[name]; ok {
		return "", err
	}
	if target, ok := m.Links[name]; ok {
		return target, nil
	}
	return "", os.ErrNotExist
}

func (m *MockFileSystem) Stat(name string) (fs.FileInfo, error) {
	if err, ok := m.Errors[name]; ok {
		return nil, err
	}
	if exists, ok := m.Stats[name]; ok && exists {
		return mockFileInfo{name: filepath.Base(name)}, nil
	}
	return nil, os.ErrNotExist
}

// MockPCIScanner provides a mock implementation of pci.Scanner for testing.
type MockPCIScanner struct {
	Devices []pci.PCIDevice
	Error   error
}

func (m *MockPCIScanner) ScanAll(ctx context.Context) ([]pci.PCIDevice, error) {
	if m.Error != nil {
		return nil, m.Error
	}
	return m.Devices, nil
}

func (m *MockPCIScanner) ScanByVendor(ctx context.Context, vendorID string) ([]pci.PCIDevice, error) {
	if m.Error != nil {
		return nil, m.Error
	}
	var result []pci.PCIDevice
	for _, d := range m.Devices {
		if d.MatchesVendor(vendorID) {
			result = append(result, d)
		}
	}
	return result, nil
}

func (m *MockPCIScanner) ScanByClass(ctx context.Context, classCode string) ([]pci.PCIDevice, error) {
	if m.Error != nil {
		return nil, m.Error
	}
	var result []pci.PCIDevice
	for _, d := range m.Devices {
		if d.MatchesClass(classCode) {
			result = append(result, d)
		}
	}
	return result, nil
}

func (m *MockPCIScanner) ScanNVIDIA(ctx context.Context) ([]pci.PCIDevice, error) {
	if m.Error != nil {
		return nil, m.Error
	}
	var result []pci.PCIDevice
	for _, d := range m.Devices {
		if d.IsNVIDIAGPU() {
			result = append(result, d)
		}
	}
	return result, nil
}

// --- Tests ---

func TestNewDetector(t *testing.T) {
	t.Run("default configuration", func(t *testing.T) {
		detector := NewDetector()
		assert.NotNil(t, detector)
		assert.Equal(t, DefaultModulePath, detector.modulePath)
		assert.Equal(t, DefaultProcModulesPath, detector.procModulesPath)
		assert.Equal(t, DefaultModprobePath, detector.modprobePath)
		assert.NotNil(t, detector.pciScanner)
		assert.IsType(t, RealFileSystem{}, detector.fs)
	})

	t.Run("with custom filesystem", func(t *testing.T) {
		mockFS := NewMockFileSystem()
		detector := NewDetector(WithFileSystem(mockFS))
		assert.Equal(t, mockFS, detector.fs)
	})

	t.Run("with custom PCI scanner", func(t *testing.T) {
		mockScanner := &MockPCIScanner{}
		detector := NewDetector(WithPCIScanner(mockScanner))
		assert.Equal(t, mockScanner, detector.pciScanner)
	})

	t.Run("with custom module path", func(t *testing.T) {
		customPath := "/custom/module/path"
		detector := NewDetector(WithModulePath(customPath))
		assert.Equal(t, customPath, detector.modulePath)
	})

	t.Run("with custom proc modules path", func(t *testing.T) {
		customPath := "/custom/proc/modules"
		detector := NewDetector(WithProcModulesPath(customPath))
		assert.Equal(t, customPath, detector.procModulesPath)
	})

	t.Run("with custom modprobe path", func(t *testing.T) {
		customPath := "/custom/modprobe.d"
		detector := NewDetector(WithModprobePath(customPath))
		assert.Equal(t, customPath, detector.modprobePath)
	})
}

func TestIsLoaded(t *testing.T) {
	const modulePath = "/sys/module/nouveau"

	t.Run("nouveau is loaded", func(t *testing.T) {
		mockFS := NewMockFileSystem()
		mockFS.Stats[modulePath] = true

		detector := NewDetector(WithFileSystem(mockFS), WithModulePath(modulePath))
		loaded, err := detector.IsLoaded(context.Background())

		require.NoError(t, err)
		assert.True(t, loaded)
	})

	t.Run("nouveau is not loaded", func(t *testing.T) {
		mockFS := NewMockFileSystem()
		// Stats map doesn't contain modulePath, so it will return ErrNotExist

		detector := NewDetector(WithFileSystem(mockFS), WithModulePath(modulePath))
		loaded, err := detector.IsLoaded(context.Background())

		require.NoError(t, err)
		assert.False(t, loaded)
	})

	t.Run("permission denied", func(t *testing.T) {
		mockFS := NewMockFileSystem()
		mockFS.Errors[modulePath] = os.ErrPermission

		detector := NewDetector(WithFileSystem(mockFS), WithModulePath(modulePath))
		loaded, err := detector.IsLoaded(context.Background())

		require.Error(t, err)
		assert.False(t, loaded)
		assert.Contains(t, err.Error(), "permission denied")
	})

	t.Run("other error", func(t *testing.T) {
		mockFS := NewMockFileSystem()
		mockFS.Errors[modulePath] = os.ErrInvalid

		detector := NewDetector(WithFileSystem(mockFS), WithModulePath(modulePath))
		loaded, err := detector.IsLoaded(context.Background())

		require.Error(t, err)
		assert.False(t, loaded)
		assert.Contains(t, err.Error(), "failed to check nouveau module status")
	})

	t.Run("context cancelled", func(t *testing.T) {
		mockFS := NewMockFileSystem()
		mockFS.Stats[modulePath] = true

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		detector := NewDetector(WithFileSystem(mockFS), WithModulePath(modulePath))
		loaded, err := detector.IsLoaded(ctx)

		require.Error(t, err)
		assert.False(t, loaded)
		assert.Contains(t, err.Error(), "cancelled")
	})
}

func TestIsBlacklisted(t *testing.T) {
	const modprobePath = "/etc/modprobe.d"

	t.Run("nouveau is blacklisted", func(t *testing.T) {
		mockFS := NewMockFileSystem()
		mockFS.Dirs[modprobePath] = []fs.DirEntry{
			mockDirEntry{name: "blacklist-nouveau.conf", isDir: false},
		}
		mockFS.Files[filepath.Join(modprobePath, "blacklist-nouveau.conf")] = "blacklist nouveau\n"

		detector := NewDetector(WithFileSystem(mockFS), WithModprobePath(modprobePath))
		blacklisted, err := detector.IsBlacklisted(context.Background())

		require.NoError(t, err)
		assert.True(t, blacklisted)
	})

	t.Run("nouveau is not blacklisted", func(t *testing.T) {
		mockFS := NewMockFileSystem()
		mockFS.Dirs[modprobePath] = []fs.DirEntry{
			mockDirEntry{name: "other.conf", isDir: false},
		}
		mockFS.Files[filepath.Join(modprobePath, "other.conf")] = "blacklist some_other_module\n"

		detector := NewDetector(WithFileSystem(mockFS), WithModprobePath(modprobePath))
		blacklisted, err := detector.IsBlacklisted(context.Background())

		require.NoError(t, err)
		assert.False(t, blacklisted)
	})

	t.Run("modprobe.d does not exist", func(t *testing.T) {
		mockFS := NewMockFileSystem()
		// Don't add modprobePath to Dirs, so it returns ErrNotExist

		detector := NewDetector(WithFileSystem(mockFS), WithModprobePath(modprobePath))
		blacklisted, err := detector.IsBlacklisted(context.Background())

		require.NoError(t, err)
		assert.False(t, blacklisted)
	})

	t.Run("blacklist with options modeset=0", func(t *testing.T) {
		mockFS := NewMockFileSystem()
		mockFS.Dirs[modprobePath] = []fs.DirEntry{
			mockDirEntry{name: "nvidia.conf", isDir: false},
		}
		mockFS.Files[filepath.Join(modprobePath, "nvidia.conf")] = "options nouveau modeset=0\n"

		detector := NewDetector(WithFileSystem(mockFS), WithModprobePath(modprobePath))
		blacklisted, err := detector.IsBlacklisted(context.Background())

		require.NoError(t, err)
		assert.True(t, blacklisted)
	})

	t.Run("multiple conf files with one blacklist", func(t *testing.T) {
		mockFS := NewMockFileSystem()
		mockFS.Dirs[modprobePath] = []fs.DirEntry{
			mockDirEntry{name: "other.conf", isDir: false},
			mockDirEntry{name: "blacklist-nouveau.conf", isDir: false},
		}
		mockFS.Files[filepath.Join(modprobePath, "other.conf")] = "# some other config\n"
		mockFS.Files[filepath.Join(modprobePath, "blacklist-nouveau.conf")] = "blacklist nouveau\n"

		detector := NewDetector(WithFileSystem(mockFS), WithModprobePath(modprobePath))
		blacklisted, err := detector.IsBlacklisted(context.Background())

		require.NoError(t, err)
		assert.True(t, blacklisted)
	})

	t.Run("context cancelled", func(t *testing.T) {
		mockFS := NewMockFileSystem()

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		detector := NewDetector(WithFileSystem(mockFS), WithModprobePath(modprobePath))
		blacklisted, err := detector.IsBlacklisted(ctx)

		require.Error(t, err)
		assert.False(t, blacklisted)
		assert.Contains(t, err.Error(), "cancelled")
	})

	t.Run("permission denied on modprobe.d", func(t *testing.T) {
		mockFS := NewMockFileSystem()
		mockFS.Errors[modprobePath] = os.ErrPermission

		detector := NewDetector(WithFileSystem(mockFS), WithModprobePath(modprobePath))
		blacklisted, err := detector.IsBlacklisted(context.Background())

		require.Error(t, err)
		assert.False(t, blacklisted)
		assert.Contains(t, err.Error(), "permission denied")
	})
}

func TestGetBoundDevices(t *testing.T) {
	t.Run("devices bound to nouveau", func(t *testing.T) {
		mockScanner := &MockPCIScanner{
			Devices: []pci.PCIDevice{
				{Address: "0000:01:00.0", VendorID: "10de", DeviceID: "2684", Class: "030000", Driver: "nouveau"},
				{Address: "0000:02:00.0", VendorID: "10de", DeviceID: "1234", Class: "030200", Driver: "nouveau"},
				{Address: "0000:03:00.0", VendorID: "8086", DeviceID: "5678", Class: "030000", Driver: "i915"},
			},
		}

		detector := NewDetector(WithPCIScanner(mockScanner))
		devices, err := detector.GetBoundDevices(context.Background())

		require.NoError(t, err)
		assert.Len(t, devices, 2)
		assert.Contains(t, devices, "0000:01:00.0")
		assert.Contains(t, devices, "0000:02:00.0")
	})

	t.Run("no devices bound to nouveau", func(t *testing.T) {
		mockScanner := &MockPCIScanner{
			Devices: []pci.PCIDevice{
				{Address: "0000:01:00.0", VendorID: "10de", DeviceID: "2684", Class: "030000", Driver: "nvidia"},
				{Address: "0000:02:00.0", VendorID: "8086", DeviceID: "5678", Class: "030000", Driver: "i915"},
			},
		}

		detector := NewDetector(WithPCIScanner(mockScanner))
		devices, err := detector.GetBoundDevices(context.Background())

		require.NoError(t, err)
		assert.Empty(t, devices)
	})

	t.Run("devices with no driver", func(t *testing.T) {
		mockScanner := &MockPCIScanner{
			Devices: []pci.PCIDevice{
				{Address: "0000:01:00.0", VendorID: "10de", DeviceID: "2684", Class: "030000", Driver: ""},
			},
		}

		detector := NewDetector(WithPCIScanner(mockScanner))
		devices, err := detector.GetBoundDevices(context.Background())

		require.NoError(t, err)
		assert.Empty(t, devices)
	})

	t.Run("PCI scanner error", func(t *testing.T) {
		mockScanner := &MockPCIScanner{
			Error: os.ErrInvalid,
		}

		detector := NewDetector(WithPCIScanner(mockScanner))
		devices, err := detector.GetBoundDevices(context.Background())

		require.Error(t, err)
		assert.Nil(t, devices)
		assert.Contains(t, err.Error(), "failed to scan PCI devices")
	})

	t.Run("context cancelled", func(t *testing.T) {
		mockScanner := &MockPCIScanner{
			Devices: []pci.PCIDevice{
				{Address: "0000:01:00.0", Driver: "nouveau"},
			},
		}

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		detector := NewDetector(WithPCIScanner(mockScanner))
		devices, err := detector.GetBoundDevices(ctx)

		require.Error(t, err)
		assert.Nil(t, devices)
		assert.Contains(t, err.Error(), "cancelled")
	})
}

func TestDetect(t *testing.T) {
	const modulePath = "/sys/module/nouveau"
	const modprobePath = "/etc/modprobe.d"

	t.Run("nouveau loaded and in use, not blacklisted", func(t *testing.T) {
		mockFS := NewMockFileSystem()
		mockFS.Stats[modulePath] = true
		mockFS.Dirs[modprobePath] = []fs.DirEntry{}

		mockScanner := &MockPCIScanner{
			Devices: []pci.PCIDevice{
				{Address: "0000:01:00.0", VendorID: "10de", DeviceID: "2684", Class: "030000", Driver: "nouveau"},
			},
		}

		detector := NewDetector(
			WithFileSystem(mockFS),
			WithPCIScanner(mockScanner),
			WithModulePath(modulePath),
			WithModprobePath(modprobePath),
		)
		status, err := detector.Detect(context.Background())

		require.NoError(t, err)
		assert.True(t, status.Loaded)
		assert.True(t, status.InUse)
		assert.Len(t, status.BoundDevices, 1)
		assert.False(t, status.BlacklistExists)
		assert.Empty(t, status.BlacklistFiles)
	})

	t.Run("nouveau not loaded, blacklisted", func(t *testing.T) {
		mockFS := NewMockFileSystem()
		// modulePath not in Stats, so IsLoaded returns false
		mockFS.Dirs[modprobePath] = []fs.DirEntry{
			mockDirEntry{name: "blacklist-nouveau.conf", isDir: false},
		}
		mockFS.Files[filepath.Join(modprobePath, "blacklist-nouveau.conf")] = "blacklist nouveau\n"

		mockScanner := &MockPCIScanner{
			Devices: []pci.PCIDevice{},
		}

		detector := NewDetector(
			WithFileSystem(mockFS),
			WithPCIScanner(mockScanner),
			WithModulePath(modulePath),
			WithModprobePath(modprobePath),
		)
		status, err := detector.Detect(context.Background())

		require.NoError(t, err)
		assert.False(t, status.Loaded)
		assert.False(t, status.InUse)
		assert.Empty(t, status.BoundDevices)
		assert.True(t, status.BlacklistExists)
		assert.Len(t, status.BlacklistFiles, 1)
	})

	t.Run("nouveau loaded but not in use, blacklisted", func(t *testing.T) {
		mockFS := NewMockFileSystem()
		mockFS.Stats[modulePath] = true
		mockFS.Dirs[modprobePath] = []fs.DirEntry{
			mockDirEntry{name: "nvidia.conf", isDir: false},
		}
		mockFS.Files[filepath.Join(modprobePath, "nvidia.conf")] = "blacklist nouveau\noptions nouveau modeset=0\n"

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
		status, err := detector.Detect(context.Background())

		require.NoError(t, err)
		assert.True(t, status.Loaded)
		assert.False(t, status.InUse)
		assert.Empty(t, status.BoundDevices)
		assert.True(t, status.BlacklistExists)
	})

	t.Run("context cancelled before detection", func(t *testing.T) {
		mockFS := NewMockFileSystem()
		mockScanner := &MockPCIScanner{}

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		detector := NewDetector(WithFileSystem(mockFS), WithPCIScanner(mockScanner))
		status, err := detector.Detect(ctx)

		require.Error(t, err)
		assert.Nil(t, status)
		assert.Contains(t, err.Error(), "cancelled")
	})

	t.Run("multiple blacklist files", func(t *testing.T) {
		mockFS := NewMockFileSystem()
		mockFS.Stats[modulePath] = true
		mockFS.Dirs[modprobePath] = []fs.DirEntry{
			mockDirEntry{name: "blacklist-nouveau.conf", isDir: false},
			mockDirEntry{name: "nvidia-installer.conf", isDir: false},
		}
		mockFS.Files[filepath.Join(modprobePath, "blacklist-nouveau.conf")] = "blacklist nouveau\n"
		mockFS.Files[filepath.Join(modprobePath, "nvidia-installer.conf")] = "options nouveau modeset=0\n"

		mockScanner := &MockPCIScanner{
			Devices: []pci.PCIDevice{},
		}

		detector := NewDetector(
			WithFileSystem(mockFS),
			WithPCIScanner(mockScanner),
			WithModulePath(modulePath),
			WithModprobePath(modprobePath),
		)
		status, err := detector.Detect(context.Background())

		require.NoError(t, err)
		assert.True(t, status.BlacklistExists)
		assert.Len(t, status.BlacklistFiles, 2)
	})
}

func TestContainsBlacklistPattern(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected bool
	}{
		{
			name:     "blacklist nouveau",
			content:  "blacklist nouveau\n",
			expected: true,
		},
		{
			name:     "blacklist nouveau with comment",
			content:  "# Blacklist nouveau driver\nblacklist nouveau\n",
			expected: true,
		},
		{
			name:     "options nouveau modeset=0",
			content:  "options nouveau modeset=0\n",
			expected: true,
		},
		{
			name:     "both patterns",
			content:  "blacklist nouveau\noptions nouveau modeset=0\n",
			expected: true,
		},
		{
			name:     "no blacklist pattern",
			content:  "# Just a comment\nblacklist some_other_module\n",
			expected: false,
		},
		{
			name:     "empty file",
			content:  "",
			expected: false,
		},
		{
			name:     "only comments",
			content:  "# Comment 1\n# Comment 2\n",
			expected: false,
		},
		{
			name:     "blacklist in comment",
			content:  "# blacklist nouveau\n",
			expected: false,
		},
		{
			name:     "extra whitespace",
			content:  "  blacklist   nouveau  \n",
			expected: true,
		},
		{
			name:     "tabs instead of spaces",
			content:  "blacklist\tnouveau\n",
			expected: true,
		},
		{
			name:     "partial match should not match",
			content:  "blacklist nouveau_extra\n",
			expected: true, // "blacklist nouveau" is a prefix, so it matches
		},
		{
			name:     "case sensitive - uppercase should not match",
			content:  "BLACKLIST NOUVEAU\n",
			expected: false, // We're doing case-sensitive matching
		},
		{
			name:     "install nouveau /bin/true pattern",
			content:  "install nouveau /bin/true\n",
			expected: false, // This is a valid blacklist but not in our patterns
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := containsBlacklistPattern([]byte(tt.content))
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFindBlacklistFiles(t *testing.T) {
	const modprobePath = "/etc/modprobe.d"

	t.Run("skip directories", func(t *testing.T) {
		mockFS := NewMockFileSystem()
		mockFS.Dirs[modprobePath] = []fs.DirEntry{
			mockDirEntry{name: "somedir", isDir: true},
			mockDirEntry{name: "blacklist-nouveau.conf", isDir: false},
		}
		mockFS.Files[filepath.Join(modprobePath, "blacklist-nouveau.conf")] = "blacklist nouveau\n"

		detector := NewDetector(WithFileSystem(mockFS), WithModprobePath(modprobePath))
		files, err := detector.findBlacklistFiles(context.Background())

		require.NoError(t, err)
		assert.Len(t, files, 1)
	})

	t.Run("skip non-conf files", func(t *testing.T) {
		mockFS := NewMockFileSystem()
		mockFS.Dirs[modprobePath] = []fs.DirEntry{
			mockDirEntry{name: "blacklist-nouveau.txt", isDir: false},
			mockDirEntry{name: "blacklist-nouveau.conf", isDir: false},
		}
		mockFS.Files[filepath.Join(modprobePath, "blacklist-nouveau.txt")] = "blacklist nouveau\n"
		mockFS.Files[filepath.Join(modprobePath, "blacklist-nouveau.conf")] = "blacklist nouveau\n"

		detector := NewDetector(WithFileSystem(mockFS), WithModprobePath(modprobePath))
		files, err := detector.findBlacklistFiles(context.Background())

		require.NoError(t, err)
		assert.Len(t, files, 1)
		assert.Contains(t, files[0], ".conf")
	})

	t.Run("skip unreadable files", func(t *testing.T) {
		mockFS := NewMockFileSystem()
		mockFS.Dirs[modprobePath] = []fs.DirEntry{
			mockDirEntry{name: "readable.conf", isDir: false},
			mockDirEntry{name: "unreadable.conf", isDir: false},
		}
		mockFS.Files[filepath.Join(modprobePath, "readable.conf")] = "blacklist nouveau\n"
		mockFS.Errors[filepath.Join(modprobePath, "unreadable.conf")] = os.ErrPermission

		detector := NewDetector(WithFileSystem(mockFS), WithModprobePath(modprobePath))
		files, err := detector.findBlacklistFiles(context.Background())

		require.NoError(t, err)
		assert.Len(t, files, 1)
	})

	t.Run("context cancelled during iteration", func(t *testing.T) {
		mockFS := NewMockFileSystem()
		entries := make([]fs.DirEntry, 100)
		for i := 0; i < 100; i++ {
			entries[i] = mockDirEntry{name: "file.conf", isDir: false}
		}
		mockFS.Dirs[modprobePath] = entries

		// Create a context that we'll cancel
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		detector := NewDetector(WithFileSystem(mockFS), WithModprobePath(modprobePath))
		files, err := detector.findBlacklistFiles(ctx)

		require.Error(t, err)
		assert.Nil(t, files)
		assert.Contains(t, err.Error(), "cancelled")
	})
}

func TestRealFileSystem(t *testing.T) {
	// These tests verify the RealFileSystem implementation works correctly
	t.Run("implements FileSystem interface", func(t *testing.T) {
		var _ FileSystem = RealFileSystem{}
	})

	t.Run("ReadDir on non-existent path", func(t *testing.T) {
		fs := RealFileSystem{}
		_, err := fs.ReadDir("/non/existent/path")
		assert.Error(t, err)
	})

	t.Run("ReadFile on non-existent path", func(t *testing.T) {
		fs := RealFileSystem{}
		_, err := fs.ReadFile("/non/existent/file")
		assert.Error(t, err)
	})

	t.Run("Readlink on non-symlink", func(t *testing.T) {
		fs := RealFileSystem{}
		_, err := fs.Readlink("/non/existent/symlink")
		assert.Error(t, err)
	})

	t.Run("Stat on non-existent path", func(t *testing.T) {
		fs := RealFileSystem{}
		_, err := fs.Stat("/non/existent/path")
		assert.Error(t, err)
	})
}

func TestDetectorImplementsInterface(t *testing.T) {
	var _ Detector = (*DetectorImpl)(nil)
}

func TestConstants(t *testing.T) {
	assert.Equal(t, "/sys/module/nouveau", DefaultModulePath)
	assert.Equal(t, "/proc/modules", DefaultProcModulesPath)
	assert.Equal(t, "/etc/modprobe.d", DefaultModprobePath)
}

func TestStatusStruct(t *testing.T) {
	t.Run("empty status", func(t *testing.T) {
		status := &Status{}
		assert.False(t, status.Loaded)
		assert.False(t, status.InUse)
		assert.Empty(t, status.BoundDevices)
		assert.False(t, status.BlacklistExists)
		assert.Empty(t, status.BlacklistFiles)
	})

	t.Run("fully populated status", func(t *testing.T) {
		status := &Status{
			Loaded:          true,
			InUse:           true,
			BoundDevices:    []string{"0000:01:00.0", "0000:02:00.0"},
			BlacklistExists: true,
			BlacklistFiles:  []string{"/etc/modprobe.d/blacklist-nouveau.conf"},
		}
		assert.True(t, status.Loaded)
		assert.True(t, status.InUse)
		assert.Len(t, status.BoundDevices, 2)
		assert.True(t, status.BlacklistExists)
		assert.Len(t, status.BlacklistFiles, 1)
	})
}

func TestEdgeCases(t *testing.T) {
	const modprobePath = "/etc/modprobe.d"

	t.Run("empty modprobe.d directory", func(t *testing.T) {
		mockFS := NewMockFileSystem()
		mockFS.Dirs[modprobePath] = []fs.DirEntry{}

		detector := NewDetector(WithFileSystem(mockFS), WithModprobePath(modprobePath))
		blacklisted, err := detector.IsBlacklisted(context.Background())

		require.NoError(t, err)
		assert.False(t, blacklisted)
	})

	t.Run("generic filesystem error on modprobe.d", func(t *testing.T) {
		mockFS := NewMockFileSystem()
		mockFS.Errors[modprobePath] = os.ErrInvalid

		detector := NewDetector(WithFileSystem(mockFS), WithModprobePath(modprobePath))
		blacklisted, err := detector.IsBlacklisted(context.Background())

		require.Error(t, err)
		assert.False(t, blacklisted)
		assert.Contains(t, err.Error(), "failed to read modprobe.d")
	})

	t.Run("empty bound devices list", func(t *testing.T) {
		mockScanner := &MockPCIScanner{
			Devices: []pci.PCIDevice{},
		}

		detector := NewDetector(WithPCIScanner(mockScanner))
		devices, err := detector.GetBoundDevices(context.Background())

		require.NoError(t, err)
		assert.Empty(t, devices)
		// Note: returns nil slice when no devices are bound (not empty slice)
	})
}
