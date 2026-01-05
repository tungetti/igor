package pci

import (
	"context"
	"io/fs"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Integration Test: Real-World lspci Outputs
// =============================================================================

func TestScanner_RealWorldLspciOutputs(t *testing.T) {
	t.Run("typical desktop with RTX 3080", func(t *testing.T) {
		const sysfsPath = "/test/sys/bus/pci/devices"
		mockFS := NewMockFileSystem()

		// Simulate RTX 3080: 01:00.0 VGA compatible controller [0300]: NVIDIA Corporation GA102 [GeForce RTX 3080] [10de:2206] (rev a1)
		mockFS.AddDeviceWithSubsystem(sysfsPath, "0000:01:00.0", "10de", "2206", "030000", "1458", "4038", "a1")
		mockFS.AddDriver(sysfsPath, "0000:01:00.0", "nvidia")

		// Audio controller for the GPU
		mockFS.AddDevice(sysfsPath, "0000:01:00.1", "10de", "1aef", "040300")
		mockFS.AddDriver(sysfsPath, "0000:01:00.1", "snd_hda_intel")

		scanner := NewScanner(WithFileSystem(mockFS), WithSysfsPath(sysfsPath))
		devices, err := scanner.ScanNVIDIA(context.Background())

		require.NoError(t, err)
		require.Len(t, devices, 1)

		gpu := devices[0]
		assert.Equal(t, "0000:01:00.0", gpu.Address)
		assert.Equal(t, "10de", gpu.VendorID)
		assert.Equal(t, "2206", gpu.DeviceID)
		assert.Equal(t, "030000", gpu.Class)
		assert.Equal(t, "nvidia", gpu.Driver)
		assert.Equal(t, "1458", gpu.SubVendorID)
		assert.Equal(t, "4038", gpu.SubDeviceID)
		assert.Equal(t, "a1", gpu.Revision)
		assert.True(t, gpu.IsNVIDIAGPU())
		assert.True(t, gpu.IsUsingProprietaryDriver())
	})

	t.Run("workstation with Quadro RTX 6000", func(t *testing.T) {
		const sysfsPath = "/test/sys/bus/pci/devices"
		mockFS := NewMockFileSystem()

		// Quadro RTX 6000: VGA controller
		mockFS.AddDeviceWithSubsystem(sysfsPath, "0000:65:00.0", "10de", "1e30", "030000", "10de", "12ba", "a1")
		mockFS.AddDriver(sysfsPath, "0000:65:00.0", "nvidia")

		scanner := NewScanner(WithFileSystem(mockFS), WithSysfsPath(sysfsPath))
		devices, err := scanner.ScanNVIDIA(context.Background())

		require.NoError(t, err)
		require.Len(t, devices, 1)

		assert.Equal(t, "1e30", devices[0].DeviceID)
		assert.True(t, devices[0].IsUsingProprietaryDriver())
	})

	t.Run("server with Tesla V100 (3D controller class)", func(t *testing.T) {
		const sysfsPath = "/test/sys/bus/pci/devices"
		mockFS := NewMockFileSystem()

		// Tesla V100 - uses 3D controller class (0302)
		mockFS.AddDeviceWithSubsystem(sysfsPath, "0000:3b:00.0", "10de", "1db4", "030200", "10de", "1214", "a1")
		mockFS.AddDriver(sysfsPath, "0000:3b:00.0", "nvidia")

		scanner := NewScanner(WithFileSystem(mockFS), WithSysfsPath(sysfsPath))
		devices, err := scanner.ScanNVIDIA(context.Background())

		require.NoError(t, err)
		require.Len(t, devices, 1)

		gpu := devices[0]
		assert.Equal(t, "030200", gpu.Class)
		assert.True(t, gpu.IsGPU())
		assert.True(t, gpu.IsNVIDIAGPU())
	})

	t.Run("laptop with GTX 1650 Mobile (Optimus)", func(t *testing.T) {
		const sysfsPath = "/test/sys/bus/pci/devices"
		mockFS := NewMockFileSystem()

		// Intel integrated GPU
		mockFS.AddDevice(sysfsPath, "0000:00:02.0", "8086", "9a49", "030000")
		mockFS.AddDriver(sysfsPath, "0000:00:02.0", "i915")

		// NVIDIA discrete GPU (may not have driver if powered off)
		mockFS.AddDevice(sysfsPath, "0000:01:00.0", "10de", "1f99", "030200")
		// No driver - GPU might be powered off

		scanner := NewScanner(WithFileSystem(mockFS), WithSysfsPath(sysfsPath))

		// All GPUs
		allDevices, err := scanner.ScanByClass(context.Background(), "03")
		require.NoError(t, err)
		assert.Len(t, allDevices, 2)

		// Just NVIDIA
		nvidiaDevices, err := scanner.ScanNVIDIA(context.Background())
		require.NoError(t, err)
		require.Len(t, nvidiaDevices, 1)

		assert.Equal(t, "1f99", nvidiaDevices[0].DeviceID)
		assert.False(t, nvidiaDevices[0].HasDriver())
	})

	t.Run("system with nouveau driver loaded", func(t *testing.T) {
		const sysfsPath = "/test/sys/bus/pci/devices"
		mockFS := NewMockFileSystem()

		// GPU with nouveau driver
		mockFS.AddDevice(sysfsPath, "0000:01:00.0", "10de", "1c03", "030000")
		mockFS.AddDriver(sysfsPath, "0000:01:00.0", "nouveau")

		scanner := NewScanner(WithFileSystem(mockFS), WithSysfsPath(sysfsPath))
		devices, err := scanner.ScanNVIDIA(context.Background())

		require.NoError(t, err)
		require.Len(t, devices, 1)

		assert.True(t, devices[0].IsUsingNouveau())
		assert.False(t, devices[0].IsUsingProprietaryDriver())
	})

	t.Run("GPU passed through to VM with vfio-pci", func(t *testing.T) {
		const sysfsPath = "/test/sys/bus/pci/devices"
		mockFS := NewMockFileSystem()

		// GPU bound to vfio-pci for passthrough
		mockFS.AddDevice(sysfsPath, "0000:41:00.0", "10de", "2684", "030000")
		mockFS.AddDriver(sysfsPath, "0000:41:00.0", "vfio-pci")

		scanner := NewScanner(WithFileSystem(mockFS), WithSysfsPath(sysfsPath))
		devices, err := scanner.ScanNVIDIA(context.Background())

		require.NoError(t, err)
		require.Len(t, devices, 1)

		assert.True(t, devices[0].IsUsingVFIO())
		assert.False(t, devices[0].IsUsingProprietaryDriver())
		assert.False(t, devices[0].IsUsingNouveau())
	})
}

// =============================================================================
// Integration Test: Edge Cases
// =============================================================================

func TestScanner_EdgeCases(t *testing.T) {
	t.Run("empty sysfs output", func(t *testing.T) {
		const sysfsPath = "/test/sys/bus/pci/devices"
		mockFS := NewMockFileSystem()
		mockFS.Dirs[sysfsPath] = []fs.DirEntry{}

		scanner := NewScanner(WithFileSystem(mockFS), WithSysfsPath(sysfsPath))
		devices, err := scanner.ScanAll(context.Background())

		require.NoError(t, err)
		assert.Empty(t, devices)
	})

	t.Run("no VGA devices in system", func(t *testing.T) {
		const sysfsPath = "/test/sys/bus/pci/devices"
		mockFS := NewMockFileSystem()

		// Only network and storage devices
		mockFS.AddDevice(sysfsPath, "0000:00:1f.6", "8086", "15bc", "020000") // Network
		mockFS.AddDevice(sysfsPath, "0000:00:17.0", "8086", "a352", "010600") // SATA

		scanner := NewScanner(WithFileSystem(mockFS), WithSysfsPath(sysfsPath))
		devices, err := scanner.ScanByClass(context.Background(), "03")

		require.NoError(t, err)
		assert.Empty(t, devices)
	})

	t.Run("multiple VGA + 3D controllers", func(t *testing.T) {
		const sysfsPath = "/test/sys/bus/pci/devices"
		mockFS := NewMockFileSystem()

		// Intel iGPU (VGA)
		mockFS.AddDevice(sysfsPath, "0000:00:02.0", "8086", "9a49", "030000")
		mockFS.AddDriver(sysfsPath, "0000:00:02.0", "i915")

		// NVIDIA VGA (gaming GPU)
		mockFS.AddDevice(sysfsPath, "0000:01:00.0", "10de", "2684", "030000")
		mockFS.AddDriver(sysfsPath, "0000:01:00.0", "nvidia")

		// NVIDIA 3D controller (compute GPU)
		mockFS.AddDevice(sysfsPath, "0000:02:00.0", "10de", "2322", "030200")
		mockFS.AddDriver(sysfsPath, "0000:02:00.0", "nvidia")

		// AMD VGA
		mockFS.AddDevice(sysfsPath, "0000:03:00.0", "1002", "73a5", "030000")
		mockFS.AddDriver(sysfsPath, "0000:03:00.0", "amdgpu")

		scanner := NewScanner(WithFileSystem(mockFS), WithSysfsPath(sysfsPath))

		// All display devices
		allDisplay, err := scanner.ScanByClass(context.Background(), "03")
		require.NoError(t, err)
		assert.Len(t, allDisplay, 4)

		// Just NVIDIA
		nvidia, err := scanner.ScanNVIDIA(context.Background())
		require.NoError(t, err)
		assert.Len(t, nvidia, 2)
	})

	t.Run("unknown vendor/device IDs", func(t *testing.T) {
		const sysfsPath = "/test/sys/bus/pci/devices"
		mockFS := NewMockFileSystem()

		// Unknown/prototype device
		mockFS.AddDevice(sysfsPath, "0000:01:00.0", "10de", "ffff", "030000")

		scanner := NewScanner(WithFileSystem(mockFS), WithSysfsPath(sysfsPath))
		devices, err := scanner.ScanNVIDIA(context.Background())

		require.NoError(t, err)
		require.Len(t, devices, 1)

		// Still detected as NVIDIA GPU
		assert.True(t, devices[0].IsNVIDIA())
		assert.True(t, devices[0].IsGPU())
		assert.Equal(t, "ffff", devices[0].DeviceID)
	})

	t.Run("malformed device entries - missing vendor", func(t *testing.T) {
		const sysfsPath = "/test/sys/bus/pci/devices"
		mockFS := NewMockFileSystem()

		// Valid device
		mockFS.AddDevice(sysfsPath, "0000:01:00.0", "10de", "2684", "030000")

		// Malformed device - no vendor file
		mockFS.Dirs[sysfsPath] = append(mockFS.Dirs[sysfsPath], mockDirEntry{name: "0000:02:00.0", isDir: true})
		// Only add class file, not vendor
		mockFS.Files[filepath.Join(sysfsPath, "0000:02:00.0", "class")] = "0x030000"

		scanner := NewScanner(WithFileSystem(mockFS), WithSysfsPath(sysfsPath))
		devices, err := scanner.ScanAll(context.Background())

		require.NoError(t, err)
		// Only valid device should be returned
		assert.Len(t, devices, 1)
		assert.Equal(t, "0000:01:00.0", devices[0].Address)
	})

	t.Run("device with very long PCI address", func(t *testing.T) {
		const sysfsPath = "/test/sys/bus/pci/devices"
		mockFS := NewMockFileSystem()

		// Extended PCI address (common in large systems)
		mockFS.AddDevice(sysfsPath, "0000:b3:00.0", "10de", "2684", "030000")
		mockFS.AddDriver(sysfsPath, "0000:b3:00.0", "nvidia")

		scanner := NewScanner(WithFileSystem(mockFS), WithSysfsPath(sysfsPath))
		devices, err := scanner.ScanNVIDIA(context.Background())

		require.NoError(t, err)
		require.Len(t, devices, 1)
		assert.Equal(t, "0000:b3:00.0", devices[0].Address)
	})

	t.Run("device with domain other than 0000", func(t *testing.T) {
		const sysfsPath = "/test/sys/bus/pci/devices"
		mockFS := NewMockFileSystem()

		// Multi-domain system
		mockFS.AddDevice(sysfsPath, "0001:01:00.0", "10de", "2684", "030000")
		mockFS.AddDevice(sysfsPath, "0002:01:00.0", "10de", "2684", "030000")

		scanner := NewScanner(WithFileSystem(mockFS), WithSysfsPath(sysfsPath))
		devices, err := scanner.ScanNVIDIA(context.Background())

		require.NoError(t, err)
		assert.Len(t, devices, 2)
	})
}

// =============================================================================
// Integration Test: GPU Architecture Identification
// =============================================================================

func TestScanner_GPUArchitectureIdentification(t *testing.T) {
	architectureTests := []struct {
		name       string
		deviceID   string
		deviceName string
		class      string
	}{
		// Ada Lovelace
		{"RTX 4090 (Ada)", "2684", "GeForce RTX 4090", "030000"},
		{"RTX 4080 (Ada)", "2704", "GeForce RTX 4080", "030000"},
		{"RTX 4070 Ti (Ada)", "2782", "GeForce RTX 4070 Ti", "030000"},
		{"RTX 4060 (Ada)", "2860", "GeForce RTX 4060", "030000"},

		// Ampere
		{"RTX 3090 Ti (Ampere)", "2204", "GeForce RTX 3090 Ti", "030000"},
		{"RTX 3090 (Ampere)", "2205", "GeForce RTX 3090", "030000"},
		{"RTX 3080 (Ampere)", "2206", "GeForce RTX 3080", "030000"},
		{"RTX 3070 (Ampere)", "2488", "GeForce RTX 3070", "030000"},
		{"RTX 3060 (Ampere)", "2503", "GeForce RTX 3060", "030000"},

		// Turing
		{"RTX 2080 Ti (Turing)", "1e04", "GeForce RTX 2080 Ti", "030000"},
		{"RTX 2080 (Turing)", "1e82", "GeForce RTX 2080", "030000"},
		{"RTX 2070 (Turing)", "1f02", "GeForce RTX 2070", "030000"},
		{"GTX 1660 Ti (Turing)", "2182", "GeForce GTX 1660 Ti", "030000"},

		// Pascal
		{"GTX 1080 Ti (Pascal)", "1b06", "GeForce GTX 1080 Ti", "030000"},
		{"GTX 1080 (Pascal)", "1b80", "GeForce GTX 1080", "030000"},
		{"GTX 1070 (Pascal)", "1b81", "GeForce GTX 1070", "030000"},
		{"GTX 1060 (Pascal)", "1c03", "GeForce GTX 1060 6GB", "030000"},

		// Hopper (Data Center)
		{"H100 PCIe (Hopper)", "2322", "NVIDIA H100 PCIe", "030200"},
		{"H100 SXM (Hopper)", "2324", "NVIDIA H100 SXM", "030200"},

		// Ampere Data Center
		{"A100 PCIe 80GB (Ampere)", "20b2", "NVIDIA A100 PCIe 80GB", "030200"},
		{"A40 (Ampere)", "2235", "NVIDIA A40", "030200"},

		// Volta Data Center
		{"V100 PCIe (Volta)", "1db1", "NVIDIA V100 PCIe 16GB", "030200"},
	}

	for _, tc := range architectureTests {
		t.Run(tc.name, func(t *testing.T) {
			const sysfsPath = "/test/sys/bus/pci/devices"
			mockFS := NewMockFileSystem()

			mockFS.AddDevice(sysfsPath, "0000:01:00.0", "10de", tc.deviceID, tc.class)
			mockFS.AddDriver(sysfsPath, "0000:01:00.0", "nvidia")

			scanner := NewScanner(WithFileSystem(mockFS), WithSysfsPath(sysfsPath))
			devices, err := scanner.ScanNVIDIA(context.Background())

			require.NoError(t, err)
			require.Len(t, devices, 1)

			device := devices[0]
			assert.Equal(t, tc.deviceID, device.DeviceID)
			assert.True(t, device.IsNVIDIAGPU())
			assert.Equal(t, VendorNVIDIA, device.VendorID)
		})
	}
}

// =============================================================================
// Integration Test: Vendor Filtering
// =============================================================================

func TestScanner_VendorFiltering(t *testing.T) {
	const sysfsPath = "/test/sys/bus/pci/devices"

	t.Run("mixed GPU vendors", func(t *testing.T) {
		mockFS := NewMockFileSystem()

		// NVIDIA
		mockFS.AddDevice(sysfsPath, "0000:01:00.0", "10de", "2684", "030000")
		mockFS.AddDriver(sysfsPath, "0000:01:00.0", "nvidia")

		// AMD
		mockFS.AddDevice(sysfsPath, "0000:02:00.0", "1002", "73bf", "030000")
		mockFS.AddDriver(sysfsPath, "0000:02:00.0", "amdgpu")

		// Intel
		mockFS.AddDevice(sysfsPath, "0000:00:02.0", "8086", "9a49", "030000")
		mockFS.AddDriver(sysfsPath, "0000:00:02.0", "i915")

		scanner := NewScanner(WithFileSystem(mockFS), WithSysfsPath(sysfsPath))

		// Test vendor filtering
		nvidia, err := scanner.ScanByVendor(context.Background(), VendorNVIDIA)
		require.NoError(t, err)
		assert.Len(t, nvidia, 1)
		assert.Equal(t, "10de", nvidia[0].VendorID)

		amd, err := scanner.ScanByVendor(context.Background(), "1002")
		require.NoError(t, err)
		assert.Len(t, amd, 1)
		assert.Equal(t, "1002", amd[0].VendorID)

		intel, err := scanner.ScanByVendor(context.Background(), "8086")
		require.NoError(t, err)
		assert.Len(t, intel, 1)
		assert.Equal(t, "8086", intel[0].VendorID)
	})
}

// =============================================================================
// Benchmarks
// =============================================================================

func BenchmarkScanner_ParseLspci(b *testing.B) {
	const sysfsPath = "/test/sys/bus/pci/devices"
	mockFS := NewMockFileSystem()

	// Create a realistic number of devices (typical desktop has 20-40 PCI devices)
	devices := []struct {
		address  string
		vendorID string
		deviceID string
		class    string
		driver   string
	}{
		{"0000:00:00.0", "8086", "9a36", "060000", "skl_uncore"},
		{"0000:00:02.0", "8086", "9a49", "030000", "i915"},
		{"0000:00:08.0", "8086", "9a11", "088000", ""},
		{"0000:00:14.0", "8086", "a0ed", "0c0330", "xhci_hcd"},
		{"0000:00:17.0", "8086", "a0d3", "010601", "ahci"},
		{"0000:00:1f.0", "8086", "a082", "060100", ""},
		{"0000:00:1f.3", "8086", "a0c8", "040380", "snd_hda_intel"},
		{"0000:00:1f.4", "8086", "a0a3", "0c0500", "i2c_i801"},
		{"0000:00:1f.5", "8086", "a0a4", "0c8000", "intel_spi"},
		{"0000:01:00.0", "10de", "2684", "030000", "nvidia"},
		{"0000:01:00.1", "10de", "22be", "040300", "snd_hda_intel"},
		{"0000:02:00.0", "144d", "a808", "010802", "nvme"},
		{"0000:03:00.0", "8086", "2725", "028000", "iwlwifi"},
		{"0000:04:00.0", "8086", "15bc", "020000", "e1000e"},
	}

	for _, d := range devices {
		mockFS.AddDevice(sysfsPath, d.address, d.vendorID, d.deviceID, d.class)
		if d.driver != "" {
			mockFS.AddDriver(sysfsPath, d.address, d.driver)
		}
	}

	scanner := NewScanner(WithFileSystem(mockFS), WithSysfsPath(sysfsPath))
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = scanner.ScanAll(ctx)
	}
}

func BenchmarkScanner_ScanNVIDIA(b *testing.B) {
	const sysfsPath = "/test/sys/bus/pci/devices"
	mockFS := NewMockFileSystem()

	// Simulate a system with many devices but only 2 NVIDIA GPUs
	for i := 0; i < 30; i++ {
		address := "0000:0" + string(rune('0'+i/10)) + ":0" + string(rune('0'+i%10)) + ".0"
		if i%15 == 0 {
			// NVIDIA GPU
			mockFS.AddDevice(sysfsPath, address, "10de", "2684", "030000")
			mockFS.AddDriver(sysfsPath, address, "nvidia")
		} else if i%5 == 0 {
			// Intel device
			mockFS.AddDevice(sysfsPath, address, "8086", "1234", "060000")
		} else {
			// Other devices
			mockFS.AddDevice(sysfsPath, address, "1022", "5678", "010600")
		}
	}

	scanner := NewScanner(WithFileSystem(mockFS), WithSysfsPath(sysfsPath))
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = scanner.ScanNVIDIA(ctx)
	}
}

func BenchmarkScanner_ScanByVendor(b *testing.B) {
	const sysfsPath = "/test/sys/bus/pci/devices"
	mockFS := NewMockFileSystem()

	for i := 0; i < 40; i++ {
		address := "0000:" + string(rune('0'+i/16)) + string(rune('0'+i%16)) + ":00.0"
		mockFS.AddDevice(sysfsPath, address, "10de", "2684", "030000")
	}

	scanner := NewScanner(WithFileSystem(mockFS), WithSysfsPath(sysfsPath))
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = scanner.ScanByVendor(ctx, "10de")
	}
}

func BenchmarkParseHexID(b *testing.B) {
	inputs := []string{
		"10de",
		"0x10de",
		"0X10DE",
		"  10de  ",
		" 0x10de ",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, input := range inputs {
			_ = ParseHexID(input)
		}
	}
}

func BenchmarkPCIDevice_IsNVIDIAGPU(b *testing.B) {
	device := PCIDevice{
		Address:  "0000:01:00.0",
		VendorID: "10de",
		DeviceID: "2684",
		Class:    "030000",
		Driver:   "nvidia",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = device.IsNVIDIAGPU()
	}
}
