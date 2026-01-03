package pci

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

// AddDevice adds a complete mock PCI device to the filesystem.
func (m *MockFileSystem) AddDevice(sysfsPath, address, vendorID, deviceID, class string) {
	devicePath := filepath.Join(sysfsPath, address)

	// Add to directory listing
	if entries, ok := m.Dirs[sysfsPath]; ok {
		m.Dirs[sysfsPath] = append(entries, mockDirEntry{name: address, isDir: true})
	} else {
		m.Dirs[sysfsPath] = []fs.DirEntry{mockDirEntry{name: address, isDir: true}}
	}

	// Add device files
	m.Files[filepath.Join(devicePath, "vendor")] = "0x" + vendorID
	m.Files[filepath.Join(devicePath, "device")] = "0x" + deviceID
	m.Files[filepath.Join(devicePath, "class")] = "0x" + class
}

// AddDeviceWithSubsystem adds a mock PCI device with subsystem information.
func (m *MockFileSystem) AddDeviceWithSubsystem(sysfsPath, address, vendorID, deviceID, class, subVendor, subDevice, revision string) {
	m.AddDevice(sysfsPath, address, vendorID, deviceID, class)
	devicePath := filepath.Join(sysfsPath, address)
	m.Files[filepath.Join(devicePath, "subsystem_vendor")] = "0x" + subVendor
	m.Files[filepath.Join(devicePath, "subsystem_device")] = "0x" + subDevice
	m.Files[filepath.Join(devicePath, "revision")] = "0x" + revision
}

// AddDriver adds a driver symlink for a device.
func (m *MockFileSystem) AddDriver(sysfsPath, address, driverName string) {
	driverPath := filepath.Join(sysfsPath, address, "driver")
	m.Stats[driverPath] = true
	m.Links[driverPath] = "../../../bus/pci/drivers/" + driverName
}

// --- Tests ---

func TestNewScanner(t *testing.T) {
	t.Run("default configuration", func(t *testing.T) {
		scanner := NewScanner()
		assert.NotNil(t, scanner)
		assert.Equal(t, DefaultSysfsPath, scanner.sysfsPath)
		assert.IsType(t, RealFileSystem{}, scanner.fs)
	})

	t.Run("with custom sysfs path", func(t *testing.T) {
		customPath := "/custom/sysfs/path"
		scanner := NewScanner(WithSysfsPath(customPath))
		assert.Equal(t, customPath, scanner.sysfsPath)
	})

	t.Run("with custom filesystem", func(t *testing.T) {
		mockFS := NewMockFileSystem()
		scanner := NewScanner(WithFileSystem(mockFS))
		assert.Equal(t, mockFS, scanner.fs)
	})
}

func TestScanAll(t *testing.T) {
	const sysfsPath = "/test/sys/bus/pci/devices"

	t.Run("scan multiple devices", func(t *testing.T) {
		mockFS := NewMockFileSystem()
		mockFS.AddDevice(sysfsPath, "0000:01:00.0", "10de", "2684", "030000") // NVIDIA RTX 4090
		mockFS.AddDevice(sysfsPath, "0000:02:00.0", "8086", "1234", "020000") // Intel network
		mockFS.AddDevice(sysfsPath, "0000:03:00.0", "1002", "5678", "030200") // AMD GPU

		scanner := NewScanner(WithFileSystem(mockFS), WithSysfsPath(sysfsPath))
		devices, err := scanner.ScanAll(context.Background())

		require.NoError(t, err)
		assert.Len(t, devices, 3)
	})

	t.Run("empty sysfs directory", func(t *testing.T) {
		mockFS := NewMockFileSystem()
		mockFS.Dirs[sysfsPath] = []fs.DirEntry{}

		scanner := NewScanner(WithFileSystem(mockFS), WithSysfsPath(sysfsPath))
		devices, err := scanner.ScanAll(context.Background())

		require.NoError(t, err)
		assert.Empty(t, devices)
	})

	t.Run("sysfs directory not found", func(t *testing.T) {
		mockFS := NewMockFileSystem()
		// Don't add anything to the mock - directory doesn't exist

		scanner := NewScanner(WithFileSystem(mockFS), WithSysfsPath(sysfsPath))
		devices, err := scanner.ScanAll(context.Background())

		require.Error(t, err)
		assert.Nil(t, devices)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("permission denied on sysfs directory", func(t *testing.T) {
		mockFS := NewMockFileSystem()
		mockFS.Errors[sysfsPath] = os.ErrPermission

		scanner := NewScanner(WithFileSystem(mockFS), WithSysfsPath(sysfsPath))
		devices, err := scanner.ScanAll(context.Background())

		require.Error(t, err)
		assert.Nil(t, devices)
		assert.Contains(t, err.Error(), "permission denied")
	})

	t.Run("context cancellation before scan", func(t *testing.T) {
		mockFS := NewMockFileSystem()
		mockFS.AddDevice(sysfsPath, "0000:01:00.0", "10de", "2684", "030000")

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		scanner := NewScanner(WithFileSystem(mockFS), WithSysfsPath(sysfsPath))
		devices, err := scanner.ScanAll(ctx)

		require.Error(t, err)
		assert.Nil(t, devices)
		assert.Contains(t, err.Error(), "cancelled")
	})

	t.Run("skip files that are not directories", func(t *testing.T) {
		mockFS := NewMockFileSystem()
		mockFS.Dirs[sysfsPath] = []fs.DirEntry{
			mockDirEntry{name: "0000:01:00.0", isDir: true},
			mockDirEntry{name: "some_file", isDir: false}, // Should be skipped
		}
		mockFS.Files[filepath.Join(sysfsPath, "0000:01:00.0", "vendor")] = "0x10de"
		mockFS.Files[filepath.Join(sysfsPath, "0000:01:00.0", "device")] = "0x2684"
		mockFS.Files[filepath.Join(sysfsPath, "0000:01:00.0", "class")] = "0x030000"

		scanner := NewScanner(WithFileSystem(mockFS), WithSysfsPath(sysfsPath))
		devices, err := scanner.ScanAll(context.Background())

		require.NoError(t, err)
		assert.Len(t, devices, 1)
	})

	t.Run("skip devices with missing vendor file", func(t *testing.T) {
		mockFS := NewMockFileSystem()
		mockFS.Dirs[sysfsPath] = []fs.DirEntry{
			mockDirEntry{name: "0000:01:00.0", isDir: true},
			mockDirEntry{name: "0000:02:00.0", isDir: true},
		}
		// Only add files for first device
		mockFS.Files[filepath.Join(sysfsPath, "0000:01:00.0", "vendor")] = "0x10de"
		mockFS.Files[filepath.Join(sysfsPath, "0000:01:00.0", "device")] = "0x2684"
		mockFS.Files[filepath.Join(sysfsPath, "0000:01:00.0", "class")] = "0x030000"
		// Second device has no files

		scanner := NewScanner(WithFileSystem(mockFS), WithSysfsPath(sysfsPath))
		devices, err := scanner.ScanAll(context.Background())

		require.NoError(t, err)
		assert.Len(t, devices, 1)
		assert.Equal(t, "0000:01:00.0", devices[0].Address)
	})
}

func TestScanByVendor(t *testing.T) {
	const sysfsPath = "/test/sys/bus/pci/devices"

	t.Run("filter by NVIDIA vendor", func(t *testing.T) {
		mockFS := NewMockFileSystem()
		mockFS.AddDevice(sysfsPath, "0000:01:00.0", "10de", "2684", "030000") // NVIDIA
		mockFS.AddDevice(sysfsPath, "0000:02:00.0", "8086", "1234", "020000") // Intel
		mockFS.AddDevice(sysfsPath, "0000:03:00.0", "10de", "1234", "030200") // NVIDIA

		scanner := NewScanner(WithFileSystem(mockFS), WithSysfsPath(sysfsPath))
		devices, err := scanner.ScanByVendor(context.Background(), "10de")

		require.NoError(t, err)
		assert.Len(t, devices, 2)
		for _, d := range devices {
			assert.Equal(t, VendorNVIDIA, d.VendorID)
		}
	})

	t.Run("vendor with 0x prefix", func(t *testing.T) {
		mockFS := NewMockFileSystem()
		mockFS.AddDevice(sysfsPath, "0000:01:00.0", "10de", "2684", "030000")
		mockFS.AddDevice(sysfsPath, "0000:02:00.0", "8086", "1234", "020000")

		scanner := NewScanner(WithFileSystem(mockFS), WithSysfsPath(sysfsPath))
		devices, err := scanner.ScanByVendor(context.Background(), "0x10de")

		require.NoError(t, err)
		assert.Len(t, devices, 1)
		assert.Equal(t, "10de", devices[0].VendorID)
	})

	t.Run("case insensitive vendor match", func(t *testing.T) {
		mockFS := NewMockFileSystem()
		mockFS.AddDevice(sysfsPath, "0000:01:00.0", "10de", "2684", "030000")

		scanner := NewScanner(WithFileSystem(mockFS), WithSysfsPath(sysfsPath))
		devices, err := scanner.ScanByVendor(context.Background(), "10DE")

		require.NoError(t, err)
		assert.Len(t, devices, 1)
	})

	t.Run("no matching vendor", func(t *testing.T) {
		mockFS := NewMockFileSystem()
		mockFS.AddDevice(sysfsPath, "0000:01:00.0", "10de", "2684", "030000")

		scanner := NewScanner(WithFileSystem(mockFS), WithSysfsPath(sysfsPath))
		devices, err := scanner.ScanByVendor(context.Background(), "1234")

		require.NoError(t, err)
		assert.Empty(t, devices)
	})
}

func TestScanByClass(t *testing.T) {
	const sysfsPath = "/test/sys/bus/pci/devices"

	t.Run("filter by VGA class", func(t *testing.T) {
		mockFS := NewMockFileSystem()
		mockFS.AddDevice(sysfsPath, "0000:01:00.0", "10de", "2684", "030000") // VGA
		mockFS.AddDevice(sysfsPath, "0000:02:00.0", "8086", "1234", "020000") // Network
		mockFS.AddDevice(sysfsPath, "0000:03:00.0", "1002", "5678", "030200") // 3D controller

		scanner := NewScanner(WithFileSystem(mockFS), WithSysfsPath(sysfsPath))
		devices, err := scanner.ScanByClass(context.Background(), "0300")

		require.NoError(t, err)
		assert.Len(t, devices, 1)
		assert.Equal(t, "030000", devices[0].Class)
	})

	t.Run("filter by 3D controller class", func(t *testing.T) {
		mockFS := NewMockFileSystem()
		mockFS.AddDevice(sysfsPath, "0000:01:00.0", "10de", "2684", "030000") // VGA
		mockFS.AddDevice(sysfsPath, "0000:02:00.0", "10de", "5678", "030200") // 3D controller

		scanner := NewScanner(WithFileSystem(mockFS), WithSysfsPath(sysfsPath))
		devices, err := scanner.ScanByClass(context.Background(), "0302")

		require.NoError(t, err)
		assert.Len(t, devices, 1)
		assert.Equal(t, "030200", devices[0].Class)
	})

	t.Run("class with 0x prefix", func(t *testing.T) {
		mockFS := NewMockFileSystem()
		mockFS.AddDevice(sysfsPath, "0000:01:00.0", "10de", "2684", "030000")

		scanner := NewScanner(WithFileSystem(mockFS), WithSysfsPath(sysfsPath))
		devices, err := scanner.ScanByClass(context.Background(), "0x0300")

		require.NoError(t, err)
		assert.Len(t, devices, 1)
	})

	t.Run("filter by display class prefix 03", func(t *testing.T) {
		mockFS := NewMockFileSystem()
		mockFS.AddDevice(sysfsPath, "0000:01:00.0", "10de", "2684", "030000") // VGA
		mockFS.AddDevice(sysfsPath, "0000:02:00.0", "8086", "1234", "020000") // Network
		mockFS.AddDevice(sysfsPath, "0000:03:00.0", "1002", "5678", "030200") // 3D controller
		mockFS.AddDevice(sysfsPath, "0000:04:00.0", "10de", "abcd", "038000") // Display controller

		scanner := NewScanner(WithFileSystem(mockFS), WithSysfsPath(sysfsPath))
		devices, err := scanner.ScanByClass(context.Background(), "03")

		require.NoError(t, err)
		assert.Len(t, devices, 3) // All display devices
	})
}

func TestScanNVIDIA(t *testing.T) {
	const sysfsPath = "/test/sys/bus/pci/devices"

	t.Run("find NVIDIA GPUs only", func(t *testing.T) {
		mockFS := NewMockFileSystem()
		mockFS.AddDevice(sysfsPath, "0000:01:00.0", "10de", "2684", "030000") // NVIDIA VGA
		mockFS.AddDevice(sysfsPath, "0000:02:00.0", "8086", "1234", "030000") // Intel VGA
		mockFS.AddDevice(sysfsPath, "0000:03:00.0", "1002", "5678", "030000") // AMD VGA
		mockFS.AddDevice(sysfsPath, "0000:04:00.0", "10de", "abcd", "030200") // NVIDIA 3D controller
		mockFS.AddDevice(sysfsPath, "0000:05:00.0", "10de", "efgh", "020000") // NVIDIA network (should not match)

		scanner := NewScanner(WithFileSystem(mockFS), WithSysfsPath(sysfsPath))
		devices, err := scanner.ScanNVIDIA(context.Background())

		require.NoError(t, err)
		assert.Len(t, devices, 2)
		for _, d := range devices {
			assert.Equal(t, VendorNVIDIA, d.VendorID)
			assert.True(t, d.IsGPU())
		}
	})

	t.Run("no NVIDIA GPUs", func(t *testing.T) {
		mockFS := NewMockFileSystem()
		mockFS.AddDevice(sysfsPath, "0000:01:00.0", "8086", "1234", "030000") // Intel
		mockFS.AddDevice(sysfsPath, "0000:02:00.0", "1002", "5678", "030000") // AMD

		scanner := NewScanner(WithFileSystem(mockFS), WithSysfsPath(sysfsPath))
		devices, err := scanner.ScanNVIDIA(context.Background())

		require.NoError(t, err)
		assert.Empty(t, devices)
	})

	t.Run("NVIDIA display controller (class 0380)", func(t *testing.T) {
		mockFS := NewMockFileSystem()
		mockFS.AddDevice(sysfsPath, "0000:01:00.0", "10de", "2684", "038000") // NVIDIA display controller

		scanner := NewScanner(WithFileSystem(mockFS), WithSysfsPath(sysfsPath))
		devices, err := scanner.ScanNVIDIA(context.Background())

		require.NoError(t, err)
		assert.Len(t, devices, 1)
	})
}

func TestDeviceWithDriver(t *testing.T) {
	const sysfsPath = "/test/sys/bus/pci/devices"

	t.Run("device with nvidia driver", func(t *testing.T) {
		mockFS := NewMockFileSystem()
		mockFS.AddDevice(sysfsPath, "0000:01:00.0", "10de", "2684", "030000")
		mockFS.AddDriver(sysfsPath, "0000:01:00.0", "nvidia")

		scanner := NewScanner(WithFileSystem(mockFS), WithSysfsPath(sysfsPath))
		devices, err := scanner.ScanAll(context.Background())

		require.NoError(t, err)
		require.Len(t, devices, 1)
		assert.Equal(t, DriverNVIDIA, devices[0].Driver)
		assert.True(t, devices[0].IsUsingProprietaryDriver())
	})

	t.Run("device with nouveau driver", func(t *testing.T) {
		mockFS := NewMockFileSystem()
		mockFS.AddDevice(sysfsPath, "0000:01:00.0", "10de", "2684", "030000")
		mockFS.AddDriver(sysfsPath, "0000:01:00.0", "nouveau")

		scanner := NewScanner(WithFileSystem(mockFS), WithSysfsPath(sysfsPath))
		devices, err := scanner.ScanAll(context.Background())

		require.NoError(t, err)
		require.Len(t, devices, 1)
		assert.Equal(t, DriverNouveau, devices[0].Driver)
		assert.True(t, devices[0].IsUsingNouveau())
	})

	t.Run("device with vfio-pci driver", func(t *testing.T) {
		mockFS := NewMockFileSystem()
		mockFS.AddDevice(sysfsPath, "0000:01:00.0", "10de", "2684", "030000")
		mockFS.AddDriver(sysfsPath, "0000:01:00.0", "vfio-pci")

		scanner := NewScanner(WithFileSystem(mockFS), WithSysfsPath(sysfsPath))
		devices, err := scanner.ScanAll(context.Background())

		require.NoError(t, err)
		require.Len(t, devices, 1)
		assert.Equal(t, DriverVFIOPCI, devices[0].Driver)
		assert.True(t, devices[0].IsUsingVFIO())
	})

	t.Run("device with no driver", func(t *testing.T) {
		mockFS := NewMockFileSystem()
		mockFS.AddDevice(sysfsPath, "0000:01:00.0", "10de", "2684", "030000")
		// No driver added

		scanner := NewScanner(WithFileSystem(mockFS), WithSysfsPath(sysfsPath))
		devices, err := scanner.ScanAll(context.Background())

		require.NoError(t, err)
		require.Len(t, devices, 1)
		assert.Equal(t, "", devices[0].Driver)
		assert.False(t, devices[0].HasDriver())
	})
}

func TestDeviceWithSubsystemInfo(t *testing.T) {
	const sysfsPath = "/test/sys/bus/pci/devices"

	t.Run("device with full subsystem info", func(t *testing.T) {
		mockFS := NewMockFileSystem()
		mockFS.AddDeviceWithSubsystem(sysfsPath, "0000:01:00.0", "10de", "2684", "030000", "1458", "4038", "a1")

		scanner := NewScanner(WithFileSystem(mockFS), WithSysfsPath(sysfsPath))
		devices, err := scanner.ScanAll(context.Background())

		require.NoError(t, err)
		require.Len(t, devices, 1)
		assert.Equal(t, "1458", devices[0].SubVendorID)
		assert.Equal(t, "4038", devices[0].SubDeviceID)
		assert.Equal(t, "a1", devices[0].Revision)
	})
}

func TestPCIDevice(t *testing.T) {
	t.Run("IsNVIDIA", func(t *testing.T) {
		d := PCIDevice{VendorID: "10de"}
		assert.True(t, d.IsNVIDIA())

		d.VendorID = "10DE" // uppercase
		assert.True(t, d.IsNVIDIA())

		d.VendorID = "8086"
		assert.False(t, d.IsNVIDIA())
	})

	t.Run("IsGPU", func(t *testing.T) {
		// VGA controller
		d := PCIDevice{Class: "030000"}
		assert.True(t, d.IsGPU())

		// 3D controller
		d.Class = "030200"
		assert.True(t, d.IsGPU())

		// Display controller
		d.Class = "038000"
		assert.True(t, d.IsGPU())

		// Network controller
		d.Class = "020000"
		assert.False(t, d.IsGPU())

		// Storage controller
		d.Class = "010600"
		assert.False(t, d.IsGPU())
	})

	t.Run("IsNVIDIAGPU", func(t *testing.T) {
		d := PCIDevice{VendorID: "10de", Class: "030000"}
		assert.True(t, d.IsNVIDIAGPU())

		d.VendorID = "8086" // Intel
		assert.False(t, d.IsNVIDIAGPU())

		d.VendorID = "10de"
		d.Class = "020000" // Network
		assert.False(t, d.IsNVIDIAGPU())
	})

	t.Run("String representation", func(t *testing.T) {
		d := PCIDevice{
			Address:  "0000:01:00.0",
			VendorID: "10de",
			DeviceID: "2684",
			Class:    "030000",
		}
		s := d.String()
		assert.Contains(t, s, "0000:01:00.0")
		assert.Contains(t, s, "10de")
		assert.Contains(t, s, "2684")
		assert.Contains(t, s, "no driver")

		d.Driver = "nvidia"
		s = d.String()
		assert.Contains(t, s, "driver: nvidia")
	})

	t.Run("PCIID", func(t *testing.T) {
		d := PCIDevice{
			VendorID:    "10de",
			DeviceID:    "2684",
			SubVendorID: "1458",
			SubDeviceID: "4038",
		}
		assert.Equal(t, "10de:2684:1458:4038", d.PCIID())
	})

	t.Run("ShortID", func(t *testing.T) {
		d := PCIDevice{
			VendorID: "10de",
			DeviceID: "2684",
		}
		assert.Equal(t, "10de:2684", d.ShortID())
	})

	t.Run("MatchesVendor", func(t *testing.T) {
		d := PCIDevice{VendorID: "10de"}
		assert.True(t, d.MatchesVendor("10de"))
		assert.True(t, d.MatchesVendor("0x10de"))
		assert.True(t, d.MatchesVendor("10DE"))
		assert.False(t, d.MatchesVendor("8086"))
	})

	t.Run("MatchesClass", func(t *testing.T) {
		d := PCIDevice{Class: "030000"}
		assert.True(t, d.MatchesClass("0300"))
		assert.True(t, d.MatchesClass("03"))
		assert.True(t, d.MatchesClass("0x0300"))
		assert.False(t, d.MatchesClass("0302"))
		assert.False(t, d.MatchesClass("02"))
	})
}

func TestParseHexID(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"10de", "10de"},
		{"10DE", "10de"},
		{"0x10de", "10de"},
		{"0X10DE", "10de"},
		{"  10de  ", "10de"},
		{" 0x10de ", "10de"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := ParseHexID(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
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

func TestScannerImplementsInterface(t *testing.T) {
	var _ Scanner = (*ScannerImpl)(nil)
}

func TestConstants(t *testing.T) {
	assert.Equal(t, "10de", VendorNVIDIA)
	assert.Equal(t, "0300", ClassVGA)
	assert.Equal(t, "0302", Class3DController)
	assert.Equal(t, "0380", ClassDisplayController)
	assert.Equal(t, "/sys/bus/pci/devices", DefaultSysfsPath)
}

func TestEdgeCases(t *testing.T) {
	const sysfsPath = "/test/sys/bus/pci/devices"

	t.Run("device with short class code", func(t *testing.T) {
		// A device with only "03" class code cannot be reliably identified as GPU
		// since it doesn't have the full subclass
		d := PCIDevice{Class: "03"}
		assert.False(t, d.IsGPU()) // Not enough info to determine GPU

		// But "0300" (VGA) is a GPU
		d.Class = "0300"
		assert.True(t, d.IsGPU())
	})

	t.Run("device with empty class code", func(t *testing.T) {
		d := PCIDevice{Class: ""}
		assert.False(t, d.IsGPU())
	})

	t.Run("generic filesystem error", func(t *testing.T) {
		mockFS := NewMockFileSystem()
		mockFS.Errors[sysfsPath] = os.ErrInvalid // Generic error

		scanner := NewScanner(WithFileSystem(mockFS), WithSysfsPath(sysfsPath))
		devices, err := scanner.ScanAll(context.Background())

		require.Error(t, err)
		assert.Nil(t, devices)
		assert.Contains(t, err.Error(), "failed to read PCI devices")
	})
}
