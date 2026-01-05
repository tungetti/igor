package gpu

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/tungetti/igor/internal/gpu/kernel"
	"github.com/tungetti/igor/internal/gpu/nouveau"
	"github.com/tungetti/igor/internal/gpu/nvidia"
	"github.com/tungetti/igor/internal/gpu/pci"
	"github.com/tungetti/igor/internal/gpu/smi"
	"github.com/tungetti/igor/internal/gpu/validator"
)

// =============================================================================
// Integration Test: Complete Detection Workflow
// =============================================================================

func TestOrchestrator_CompleteDetectionWorkflow(t *testing.T) {
	t.Run("full workflow: scan PCI, identify GPUs, get info, check drivers, validate", func(t *testing.T) {
		// Setup comprehensive mocks
		scanner := &MockPCIScanner{}
		database := &MockGPUDatabase{}
		parser := &MockSMIParser{}
		nouveauDet := &MockNouveauDetector{}
		kernelDet := &MockKernelDetector{}
		val := &MockValidator{}

		// Step 1: PCI scan returns NVIDIA GPU
		device := pci.PCIDevice{
			Address:  "0000:01:00.0",
			VendorID: "10de",
			DeviceID: "2684",
			Class:    "030000",
			Driver:   "nvidia",
			Revision: "a1",
		}
		scanner.On("ScanNVIDIA", mock.Anything).Return([]pci.PCIDevice{device}, nil)

		// Step 2: Database lookup for GPU info
		model := &nvidia.GPUModel{
			DeviceID:          "2684",
			Name:              "GeForce RTX 4090",
			Architecture:      nvidia.ArchAdaLovelace,
			MinDriverVersion:  "525.60",
			ComputeCapability: "8.9",
			MemorySize:        "24GB",
		}
		database.On("Lookup", "2684").Return(model, true)

		// Step 3: nvidia-smi returns driver info
		smiInfo := &smi.SMIInfo{
			DriverVersion: "550.54.14",
			CUDAVersion:   "12.4",
			Available:     true,
			GPUs: []smi.SMIGPUInfo{
				{
					Index:          0,
					Name:           "NVIDIA GeForce RTX 4090",
					UUID:           "GPU-12345678-1234-1234-1234-123456789abc",
					MemoryTotal:    "24564 MiB",
					MemoryUsed:     "1234 MiB",
					MemoryFree:     "23330 MiB",
					MemoryTotalMiB: 24564,
					MemoryUsedMiB:  1234,
					MemoryFreeMiB:  23330,
					Temperature:    45,
					UtilizationGPU: 10,
					UtilizationMem: 5,
				},
			},
		}
		parser.On("Parse", mock.Anything).Return(smiInfo, nil)
		parser.On("IsAvailable", mock.Anything).Return(true)
		parser.On("GetDriverVersion", mock.Anything).Return("550.54.14", nil)
		parser.On("GetCUDAVersion", mock.Anything).Return("12.4", nil)

		// Step 4: Nouveau not loaded (good for NVIDIA driver)
		nouveauDet.On("Detect", mock.Anything).Return(&nouveau.Status{
			Loaded:          false,
			InUse:           false,
			BlacklistExists: true,
			BlacklistFiles:  []string{"/etc/modprobe.d/blacklist-nouveau.conf"},
		}, nil)

		// Step 5: Kernel info
		kernelInfo := &kernel.KernelInfo{
			Version:           "6.5.0-44-generic",
			Release:           "6.5.0",
			Architecture:      "x86_64",
			HeadersPath:       "/usr/src/linux-headers-6.5.0-44-generic",
			HeadersInstalled:  true,
			SecureBootEnabled: false,
		}
		kernelDet.On("GetKernelInfo", mock.Anything).Return(kernelInfo, nil)

		// Step 6: Validation passes
		report := validator.NewValidationReport()
		report.AddCheck(validator.NewCheckResult(
			validator.CheckKernelVersion,
			true,
			"kernel version 6.5.0 is compatible",
			validator.SeverityInfo,
		))
		report.AddCheck(validator.NewCheckResult(
			validator.CheckDiskSpace,
			true,
			"sufficient disk space available (10GB+)",
			validator.SeverityInfo,
		))
		report.AddCheck(validator.NewCheckResult(
			validator.CheckKernelHeaders,
			true,
			"kernel headers installed",
			validator.SeverityInfo,
		))
		val.On("Validate", mock.Anything).Return(report, nil)

		// Create orchestrator with all components
		o := NewOrchestrator(
			WithPCIScanner(scanner),
			WithGPUDatabase(database),
			WithSMIParser(parser),
			WithNouveauDetector(nouveauDet),
			WithKernelDetector(kernelDet),
			WithSystemValidator(val),
			WithTimeout(60*time.Second),
		)

		// Execute full detection
		ctx := context.Background()
		info, err := o.DetectAll(ctx)

		// Verify complete workflow
		require.NoError(t, err)
		require.NotNil(t, info)

		// GPU detection
		assert.True(t, info.HasNVIDIAGPUs())
		assert.Equal(t, 1, info.GPUCount())
		assert.Equal(t, "GeForce RTX 4090", info.NVIDIAGPUs[0].Name())
		assert.Equal(t, "ada", info.NVIDIAGPUs[0].Architecture())

		// Driver status
		assert.True(t, info.IsDriverInstalled())
		assert.NotNil(t, info.InstalledDriver)
		assert.Equal(t, DriverTypeNVIDIA, info.InstalledDriver.Type)
		assert.Equal(t, "550.54.14", info.InstalledDriver.Version)
		assert.Equal(t, "12.4", info.InstalledDriver.CUDAVersion)

		// Nouveau status
		assert.False(t, info.IsNouveauLoaded())
		assert.NotNil(t, info.NouveauStatus)
		assert.True(t, info.NouveauStatus.BlacklistExists)

		// Kernel info
		assert.NotNil(t, info.KernelInfo)
		assert.Equal(t, "6.5.0-44-generic", info.KernelInfo.Version)
		assert.True(t, info.KernelInfo.HeadersInstalled)
		assert.False(t, info.KernelInfo.SecureBootEnabled)

		// Validation
		assert.NotNil(t, info.ValidationReport)
		assert.True(t, info.ValidationReport.Passed)
		assert.False(t, info.HasValidationErrors())

		// Metadata
		assert.False(t, info.HasErrors())
		assert.Greater(t, info.Duration, time.Duration(0))
	})

	t.Run("workflow with nvidia-smi enrichment of GPU data", func(t *testing.T) {
		scanner := &MockPCIScanner{}
		database := &MockGPUDatabase{}
		parser := &MockSMIParser{}

		device := pci.PCIDevice{
			Address:  "0000:01:00.0",
			VendorID: "10de",
			DeviceID: "2684",
			Class:    "030000",
			Driver:   "nvidia",
		}
		scanner.On("ScanNVIDIA", mock.Anything).Return([]pci.PCIDevice{device}, nil)

		model := &nvidia.GPUModel{
			DeviceID:     "2684",
			Name:         "GeForce RTX 4090",
			Architecture: nvidia.ArchAdaLovelace,
		}
		database.On("Lookup", "2684").Return(model, true)

		smiInfo := &smi.SMIInfo{
			DriverVersion: "550.54.14",
			CUDAVersion:   "12.4",
			Available:     true,
			GPUs: []smi.SMIGPUInfo{
				{
					Index:          0,
					Name:           "NVIDIA GeForce RTX 4090",
					MemoryTotalMiB: 24564,
					MemoryUsedMiB:  1234,
					Temperature:    45,
					UtilizationGPU: 10,
				},
			},
		}
		parser.On("Parse", mock.Anything).Return(smiInfo, nil)
		parser.On("IsAvailable", mock.Anything).Return(true)
		parser.On("GetDriverVersion", mock.Anything).Return("550.54.14", nil)
		parser.On("GetCUDAVersion", mock.Anything).Return("12.4", nil)

		o := NewOrchestrator(
			WithPCIScanner(scanner),
			WithGPUDatabase(database),
			WithSMIParser(parser),
		)

		ctx := context.Background()
		info, err := o.DetectAll(ctx)

		require.NoError(t, err)
		require.Len(t, info.NVIDIAGPUs, 1)

		// Verify SMI data enrichment
		assert.NotNil(t, info.NVIDIAGPUs[0].SMIInfo)
		assert.Equal(t, int64(24564), info.NVIDIAGPUs[0].SMIInfo.MemoryTotalMiB)
		assert.Equal(t, 45, info.NVIDIAGPUs[0].SMIInfo.Temperature)
	})
}

// =============================================================================
// Integration Test: Multi-GPU Scenarios
// =============================================================================

func TestOrchestrator_MultiGPUScenarios(t *testing.T) {
	t.Run("two NVIDIA GPUs of same generation", func(t *testing.T) {
		scanner := &MockPCIScanner{}
		database := &MockGPUDatabase{}
		parser := &MockSMIParser{}

		devices := []pci.PCIDevice{
			{Address: "0000:01:00.0", VendorID: "10de", DeviceID: "2684", Class: "030000", Driver: "nvidia"},
			{Address: "0000:02:00.0", VendorID: "10de", DeviceID: "2684", Class: "030000", Driver: "nvidia"},
		}
		scanner.On("ScanNVIDIA", mock.Anything).Return(devices, nil)

		model := &nvidia.GPUModel{
			DeviceID:     "2684",
			Name:         "GeForce RTX 4090",
			Architecture: nvidia.ArchAdaLovelace,
		}
		database.On("Lookup", "2684").Return(model, true)

		smiInfo := &smi.SMIInfo{
			DriverVersion: "550.54.14",
			CUDAVersion:   "12.4",
			Available:     true,
			GPUs: []smi.SMIGPUInfo{
				{Index: 0, Name: "NVIDIA GeForce RTX 4090", MemoryTotalMiB: 24564},
				{Index: 1, Name: "NVIDIA GeForce RTX 4090", MemoryTotalMiB: 24564},
			},
		}
		parser.On("Parse", mock.Anything).Return(smiInfo, nil)
		parser.On("IsAvailable", mock.Anything).Return(true)
		parser.On("GetDriverVersion", mock.Anything).Return("550.54.14", nil)
		parser.On("GetCUDAVersion", mock.Anything).Return("12.4", nil)

		o := NewOrchestrator(
			WithPCIScanner(scanner),
			WithGPUDatabase(database),
			WithSMIParser(parser),
		)

		ctx := context.Background()
		info, err := o.DetectAll(ctx)

		require.NoError(t, err)
		assert.Equal(t, 2, info.GPUCount())
		assert.Equal(t, "GeForce RTX 4090", info.NVIDIAGPUs[0].Name())
		assert.Equal(t, "GeForce RTX 4090", info.NVIDIAGPUs[1].Name())
	})

	t.Run("mixed NVIDIA GPUs different generations", func(t *testing.T) {
		scanner := &MockPCIScanner{}
		database := &MockGPUDatabase{}
		parser := &MockSMIParser{}

		// RTX 4090 (Ada) + RTX 3080 (Ampere)
		devices := []pci.PCIDevice{
			{Address: "0000:01:00.0", VendorID: "10de", DeviceID: "2684", Class: "030000", Driver: "nvidia"},
			{Address: "0000:02:00.0", VendorID: "10de", DeviceID: "2206", Class: "030000", Driver: "nvidia"},
		}
		scanner.On("ScanNVIDIA", mock.Anything).Return(devices, nil)

		rtx4090 := &nvidia.GPUModel{DeviceID: "2684", Name: "GeForce RTX 4090", Architecture: nvidia.ArchAdaLovelace}
		rtx3080 := &nvidia.GPUModel{DeviceID: "2206", Name: "GeForce RTX 3080", Architecture: nvidia.ArchAmpere}
		database.On("Lookup", "2684").Return(rtx4090, true)
		database.On("Lookup", "2206").Return(rtx3080, true)

		smiInfo := &smi.SMIInfo{
			DriverVersion: "550.54.14",
			CUDAVersion:   "12.4",
			Available:     true,
			GPUs: []smi.SMIGPUInfo{
				{Index: 0, Name: "NVIDIA GeForce RTX 4090", MemoryTotalMiB: 24564},
				{Index: 1, Name: "NVIDIA GeForce RTX 3080", MemoryTotalMiB: 10240},
			},
		}
		parser.On("Parse", mock.Anything).Return(smiInfo, nil)
		parser.On("IsAvailable", mock.Anything).Return(true)
		parser.On("GetDriverVersion", mock.Anything).Return("550.54.14", nil)
		parser.On("GetCUDAVersion", mock.Anything).Return("12.4", nil)

		o := NewOrchestrator(
			WithPCIScanner(scanner),
			WithGPUDatabase(database),
			WithSMIParser(parser),
		)

		ctx := context.Background()
		info, err := o.DetectAll(ctx)

		require.NoError(t, err)
		assert.Equal(t, 2, info.GPUCount())
		assert.Equal(t, "ada", info.NVIDIAGPUs[0].Architecture())
		assert.Equal(t, "ampere", info.NVIDIAGPUs[1].Architecture())
	})

	t.Run("datacenter vs consumer GPUs", func(t *testing.T) {
		scanner := &MockPCIScanner{}
		database := &MockGPUDatabase{}
		parser := &MockSMIParser{}

		// H100 (datacenter) + RTX 4090 (consumer)
		devices := []pci.PCIDevice{
			{Address: "0000:01:00.0", VendorID: "10de", DeviceID: "2322", Class: "030200", Driver: "nvidia"},
			{Address: "0000:02:00.0", VendorID: "10de", DeviceID: "2684", Class: "030000", Driver: "nvidia"},
		}
		scanner.On("ScanNVIDIA", mock.Anything).Return(devices, nil)

		h100 := &nvidia.GPUModel{
			DeviceID:     "2322",
			Name:         "NVIDIA H100 PCIe",
			Architecture: nvidia.ArchHopper,
			IsDataCenter: true,
			MemorySize:   "80GB",
		}
		rtx4090 := &nvidia.GPUModel{
			DeviceID:     "2684",
			Name:         "GeForce RTX 4090",
			Architecture: nvidia.ArchAdaLovelace,
			IsDataCenter: false,
			MemorySize:   "24GB",
		}
		database.On("Lookup", "2322").Return(h100, true)
		database.On("Lookup", "2684").Return(rtx4090, true)

		smiInfo := &smi.SMIInfo{
			DriverVersion: "550.54.14",
			CUDAVersion:   "12.4",
			Available:     true,
			GPUs: []smi.SMIGPUInfo{
				{Index: 0, Name: "NVIDIA H100 PCIe", MemoryTotalMiB: 81920},
				{Index: 1, Name: "NVIDIA GeForce RTX 4090", MemoryTotalMiB: 24564},
			},
		}
		parser.On("Parse", mock.Anything).Return(smiInfo, nil)
		parser.On("IsAvailable", mock.Anything).Return(true)
		parser.On("GetDriverVersion", mock.Anything).Return("550.54.14", nil)
		parser.On("GetCUDAVersion", mock.Anything).Return("12.4", nil)

		o := NewOrchestrator(
			WithPCIScanner(scanner),
			WithGPUDatabase(database),
			WithSMIParser(parser),
		)

		ctx := context.Background()
		info, err := o.DetectAll(ctx)

		require.NoError(t, err)
		assert.Equal(t, 2, info.GPUCount())

		// Check datacenter GPU
		assert.True(t, info.NVIDIAGPUs[0].Model.IsDataCenter)
		assert.Equal(t, "hopper", info.NVIDIAGPUs[0].Architecture())

		// Check consumer GPU
		assert.False(t, info.NVIDIAGPUs[1].Model.IsDataCenter)
		assert.Equal(t, "ada", info.NVIDIAGPUs[1].Architecture())
	})

	t.Run("four GPU compute node", func(t *testing.T) {
		scanner := &MockPCIScanner{}
		database := &MockGPUDatabase{}
		parser := &MockSMIParser{}

		// 4x A100 GPUs
		devices := []pci.PCIDevice{
			{Address: "0000:01:00.0", VendorID: "10de", DeviceID: "20b2", Class: "030200", Driver: "nvidia"},
			{Address: "0000:02:00.0", VendorID: "10de", DeviceID: "20b2", Class: "030200", Driver: "nvidia"},
			{Address: "0000:03:00.0", VendorID: "10de", DeviceID: "20b2", Class: "030200", Driver: "nvidia"},
			{Address: "0000:04:00.0", VendorID: "10de", DeviceID: "20b2", Class: "030200", Driver: "nvidia"},
		}
		scanner.On("ScanNVIDIA", mock.Anything).Return(devices, nil)

		a100 := &nvidia.GPUModel{
			DeviceID:     "20b2",
			Name:         "NVIDIA A100 PCIe 80GB",
			Architecture: nvidia.ArchAmpere,
			IsDataCenter: true,
			MemorySize:   "80GB",
		}
		database.On("Lookup", "20b2").Return(a100, true)

		smiGPUs := make([]smi.SMIGPUInfo, 4)
		for i := 0; i < 4; i++ {
			smiGPUs[i] = smi.SMIGPUInfo{
				Index:          i,
				Name:           "NVIDIA A100-PCIE-80GB",
				MemoryTotalMiB: 81920,
			}
		}
		smiInfo := &smi.SMIInfo{
			DriverVersion: "535.154.05",
			CUDAVersion:   "12.2",
			Available:     true,
			GPUs:          smiGPUs,
		}
		parser.On("Parse", mock.Anything).Return(smiInfo, nil)
		parser.On("IsAvailable", mock.Anything).Return(true)
		parser.On("GetDriverVersion", mock.Anything).Return("535.154.05", nil)
		parser.On("GetCUDAVersion", mock.Anything).Return("12.2", nil)

		o := NewOrchestrator(
			WithPCIScanner(scanner),
			WithGPUDatabase(database),
			WithSMIParser(parser),
		)

		ctx := context.Background()
		info, err := o.DetectAll(ctx)

		require.NoError(t, err)
		assert.Equal(t, 4, info.GPUCount())
		for i := 0; i < 4; i++ {
			assert.True(t, info.NVIDIAGPUs[i].Model.IsDataCenter)
			assert.Equal(t, "ampere", info.NVIDIAGPUs[i].Architecture())
		}
	})
}

// =============================================================================
// Integration Test: Error Recovery
// =============================================================================

func TestOrchestrator_ErrorRecovery(t *testing.T) {
	t.Run("continues when PCI scan fails but other components succeed", func(t *testing.T) {
		scanner := &MockPCIScanner{}
		kernelDet := &MockKernelDetector{}
		nouveauDet := &MockNouveauDetector{}

		// PCI scan fails
		scanner.On("ScanNVIDIA", mock.Anything).Return(nil, assert.AnError)

		// But kernel detection succeeds
		kernelInfo := &kernel.KernelInfo{
			Version:      "6.5.0-44-generic",
			Architecture: "x86_64",
		}
		kernelDet.On("GetKernelInfo", mock.Anything).Return(kernelInfo, nil)

		// Nouveau detection succeeds
		nouveauDet.On("Detect", mock.Anything).Return(&nouveau.Status{Loaded: false}, nil)

		o := NewOrchestrator(
			WithPCIScanner(scanner),
			WithKernelDetector(kernelDet),
			WithNouveauDetector(nouveauDet),
		)

		ctx := context.Background()
		info, err := o.DetectAll(ctx)

		// Should not return error - collects errors internally
		require.NoError(t, err)
		require.NotNil(t, info)

		// GPU detection failed but other info available
		assert.False(t, info.HasNVIDIAGPUs())
		assert.NotNil(t, info.KernelInfo)
		assert.NotNil(t, info.NouveauStatus)
		assert.True(t, info.HasErrors())
	})

	t.Run("nvidia-smi not installed - graceful degradation", func(t *testing.T) {
		scanner := &MockPCIScanner{}
		parser := &MockSMIParser{}
		database := &MockGPUDatabase{}

		device := pci.PCIDevice{
			Address:  "0000:01:00.0",
			VendorID: "10de",
			DeviceID: "2684",
			Class:    "030000",
			Driver:   "", // No driver loaded
		}
		scanner.On("ScanNVIDIA", mock.Anything).Return([]pci.PCIDevice{device}, nil)

		model := &nvidia.GPUModel{
			DeviceID:     "2684",
			Name:         "GeForce RTX 4090",
			Architecture: nvidia.ArchAdaLovelace,
		}
		database.On("Lookup", "2684").Return(model, true)

		// nvidia-smi not available
		parser.On("Parse", mock.Anything).Return(nil, assert.AnError)
		parser.On("IsAvailable", mock.Anything).Return(false)

		o := NewOrchestrator(
			WithPCIScanner(scanner),
			WithGPUDatabase(database),
			WithSMIParser(parser),
		)

		ctx := context.Background()
		info, err := o.DetectAll(ctx)

		require.NoError(t, err)
		require.NotNil(t, info)

		// GPU detected via PCI, just no SMI enrichment
		assert.True(t, info.HasNVIDIAGPUs())
		assert.Equal(t, "GeForce RTX 4090", info.NVIDIAGPUs[0].Name())
		assert.Nil(t, info.NVIDIAGPUs[0].SMIInfo)
	})

	t.Run("unknown GPU in database - still detected", func(t *testing.T) {
		scanner := &MockPCIScanner{}
		database := &MockGPUDatabase{}

		// Future GPU not in database
		device := pci.PCIDevice{
			Address:  "0000:01:00.0",
			VendorID: "10de",
			DeviceID: "9999",
			Class:    "030000",
			Driver:   "nvidia",
		}
		scanner.On("ScanNVIDIA", mock.Anything).Return([]pci.PCIDevice{device}, nil)
		database.On("Lookup", "9999").Return(nil, false)

		o := NewOrchestrator(
			WithPCIScanner(scanner),
			WithGPUDatabase(database),
		)

		ctx := context.Background()
		gpus, err := o.DetectGPUs(ctx)

		require.NoError(t, err)
		require.Len(t, gpus, 1)

		// GPU detected but no model info
		assert.Nil(t, gpus[0].Model)
		assert.Contains(t, gpus[0].Name(), "9999")
	})

	t.Run("validation fails but detection succeeds", func(t *testing.T) {
		scanner := &MockPCIScanner{}
		database := &MockGPUDatabase{}
		val := &MockValidator{}

		device := pci.PCIDevice{
			Address:  "0000:01:00.0",
			VendorID: "10de",
			DeviceID: "2684",
			Class:    "030000",
			Driver:   "nvidia",
		}
		scanner.On("ScanNVIDIA", mock.Anything).Return([]pci.PCIDevice{device}, nil)

		model := &nvidia.GPUModel{DeviceID: "2684", Name: "GeForce RTX 4090"}
		database.On("Lookup", "2684").Return(model, true)

		// Validation has errors
		report := validator.NewValidationReport()
		report.AddCheck(validator.NewCheckResult(
			validator.CheckKernelHeaders,
			false,
			"kernel headers not installed",
			validator.SeverityError,
		).WithRemediation("Install linux-headers-$(uname -r)"))
		val.On("Validate", mock.Anything).Return(report, nil)

		o := NewOrchestrator(
			WithPCIScanner(scanner),
			WithGPUDatabase(database),
			WithSystemValidator(val),
		)

		ctx := context.Background()
		info, err := o.DetectAll(ctx)

		require.NoError(t, err)
		require.NotNil(t, info)

		// GPU detected
		assert.True(t, info.HasNVIDIAGPUs())

		// But validation failed
		assert.True(t, info.HasValidationErrors())
		assert.False(t, info.ValidationReport.Passed)
	})

	t.Run("driver detection with multiple fallbacks", func(t *testing.T) {
		scanner := &MockPCIScanner{}
		parser := &MockSMIParser{}
		nouveauDet := &MockNouveauDetector{}

		// PCI shows no driver
		device := pci.PCIDevice{
			Address:  "0000:01:00.0",
			VendorID: "10de",
			DeviceID: "2684",
			Class:    "030000",
			Driver:   "",
		}
		scanner.On("ScanNVIDIA", mock.Anything).Return([]pci.PCIDevice{device}, nil)

		// nvidia-smi not available
		parser.On("IsAvailable", mock.Anything).Return(false)

		// Fallback to nouveau detector - nouveau is loaded
		nouveauDet.On("Detect", mock.Anything).Return(&nouveau.Status{
			Loaded: true,
			InUse:  true,
		}, nil)

		o := NewOrchestrator(
			WithPCIScanner(scanner),
			WithSMIParser(parser),
			WithNouveauDetector(nouveauDet),
		)

		ctx := context.Background()
		driverInfo, err := o.GetDriverStatus(ctx)

		require.NoError(t, err)
		assert.True(t, driverInfo.Installed)
		assert.Equal(t, DriverTypeNouveau, driverInfo.Type)
	})
}

// =============================================================================
// Integration Test: Concurrent Detection
// =============================================================================

func TestOrchestrator_ConcurrentDetection(t *testing.T) {
	t.Run("multiple concurrent detection requests", func(t *testing.T) {
		scanner := &MockPCIScanner{}
		database := &MockGPUDatabase{}
		parser := &MockSMIParser{}
		nouveauDet := &MockNouveauDetector{}
		kernelDet := &MockKernelDetector{}
		val := &MockValidator{}

		device := pci.PCIDevice{
			Address:  "0000:01:00.0",
			VendorID: "10de",
			DeviceID: "2684",
			Class:    "030000",
			Driver:   "nvidia",
		}
		scanner.On("ScanNVIDIA", mock.Anything).Return([]pci.PCIDevice{device}, nil)

		model := &nvidia.GPUModel{DeviceID: "2684", Name: "GeForce RTX 4090", Architecture: nvidia.ArchAdaLovelace}
		database.On("Lookup", "2684").Return(model, true)

		smiInfo := &smi.SMIInfo{
			DriverVersion: "550.54.14",
			CUDAVersion:   "12.4",
			Available:     true,
			GPUs:          []smi.SMIGPUInfo{{Index: 0, Name: "NVIDIA GeForce RTX 4090"}},
		}
		parser.On("Parse", mock.Anything).Return(smiInfo, nil)
		parser.On("IsAvailable", mock.Anything).Return(true)
		parser.On("GetDriverVersion", mock.Anything).Return("550.54.14", nil)
		parser.On("GetCUDAVersion", mock.Anything).Return("12.4", nil)

		nouveauDet.On("Detect", mock.Anything).Return(&nouveau.Status{Loaded: false}, nil)

		kernelInfo := &kernel.KernelInfo{Version: "6.5.0-44-generic", Architecture: "x86_64"}
		kernelDet.On("GetKernelInfo", mock.Anything).Return(kernelInfo, nil)

		report := createPassingValidationReport()
		val.On("Validate", mock.Anything).Return(report, nil)

		o := NewOrchestrator(
			WithPCIScanner(scanner),
			WithGPUDatabase(database),
			WithSMIParser(parser),
			WithNouveauDetector(nouveauDet),
			WithKernelDetector(kernelDet),
			WithSystemValidator(val),
		)

		const numConcurrent = 10
		var wg sync.WaitGroup
		results := make([]*GPUInfo, numConcurrent)
		errors := make([]error, numConcurrent)

		for i := 0; i < numConcurrent; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				ctx := context.Background()
				results[idx], errors[idx] = o.DetectAll(ctx)
			}(i)
		}

		wg.Wait()

		// All detections should succeed
		for i := 0; i < numConcurrent; i++ {
			require.NoError(t, errors[i], "Detection %d failed", i)
			require.NotNil(t, results[i], "Result %d is nil", i)
			assert.True(t, results[i].HasNVIDIAGPUs())
			assert.Equal(t, 1, results[i].GPUCount())
		}
	})

	t.Run("concurrent DetectGPUs and GetDriverStatus", func(t *testing.T) {
		scanner := &MockPCIScanner{}
		database := &MockGPUDatabase{}
		parser := &MockSMIParser{}

		device := pci.PCIDevice{
			Address:  "0000:01:00.0",
			VendorID: "10de",
			DeviceID: "2684",
			Class:    "030000",
			Driver:   "nvidia",
		}
		scanner.On("ScanNVIDIA", mock.Anything).Return([]pci.PCIDevice{device}, nil)

		model := &nvidia.GPUModel{DeviceID: "2684", Name: "GeForce RTX 4090"}
		database.On("Lookup", "2684").Return(model, true)

		smiInfo := &smi.SMIInfo{
			DriverVersion: "550.54.14",
			Available:     true,
			GPUs:          []smi.SMIGPUInfo{{Index: 0, Name: "NVIDIA GeForce RTX 4090"}},
		}
		parser.On("Parse", mock.Anything).Return(smiInfo, nil)
		parser.On("IsAvailable", mock.Anything).Return(true)
		parser.On("GetDriverVersion", mock.Anything).Return("550.54.14", nil)
		parser.On("GetCUDAVersion", mock.Anything).Return("12.4", nil)

		o := NewOrchestrator(
			WithPCIScanner(scanner),
			WithGPUDatabase(database),
			WithSMIParser(parser),
		)

		var wg sync.WaitGroup
		var gpuErr, driverErr error
		var gpus []NVIDIAGPUInfo
		var driverInfo *DriverInfo

		wg.Add(2)
		go func() {
			defer wg.Done()
			gpus, gpuErr = o.DetectGPUs(context.Background())
		}()
		go func() {
			defer wg.Done()
			driverInfo, driverErr = o.GetDriverStatus(context.Background())
		}()

		wg.Wait()

		require.NoError(t, gpuErr)
		require.NoError(t, driverErr)
		require.Len(t, gpus, 1)
		require.NotNil(t, driverInfo)
		assert.True(t, driverInfo.Installed)
	})
}

// =============================================================================
// Benchmarks
// =============================================================================

func BenchmarkOrchestrator_Detect(b *testing.B) {
	scanner := &MockPCIScanner{}
	database := &MockGPUDatabase{}
	parser := &MockSMIParser{}
	nouveauDet := &MockNouveauDetector{}
	kernelDet := &MockKernelDetector{}
	val := &MockValidator{}

	device := pci.PCIDevice{Address: "0000:01:00.0", VendorID: "10de", DeviceID: "2684", Class: "030000", Driver: "nvidia"}
	scanner.On("ScanNVIDIA", mock.Anything).Return([]pci.PCIDevice{device}, nil)

	model := &nvidia.GPUModel{DeviceID: "2684", Name: "GeForce RTX 4090"}
	database.On("Lookup", "2684").Return(model, true)

	smiInfo := &smi.SMIInfo{DriverVersion: "550.54.14", CUDAVersion: "12.4", Available: true, GPUs: []smi.SMIGPUInfo{{Index: 0}}}
	parser.On("Parse", mock.Anything).Return(smiInfo, nil)
	parser.On("IsAvailable", mock.Anything).Return(true)
	parser.On("GetDriverVersion", mock.Anything).Return("550.54.14", nil)
	parser.On("GetCUDAVersion", mock.Anything).Return("12.4", nil)

	nouveauDet.On("Detect", mock.Anything).Return(&nouveau.Status{Loaded: false}, nil)
	kernelDet.On("GetKernelInfo", mock.Anything).Return(&kernel.KernelInfo{Version: "6.5.0"}, nil)
	val.On("Validate", mock.Anything).Return(createPassingValidationReport(), nil)

	o := NewOrchestrator(
		WithPCIScanner(scanner),
		WithGPUDatabase(database),
		WithSMIParser(parser),
		WithNouveauDetector(nouveauDet),
		WithKernelDetector(kernelDet),
		WithSystemValidator(val),
	)

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = o.DetectAll(ctx)
	}
}

func BenchmarkOrchestrator_DetectGPUs(b *testing.B) {
	scanner := &MockPCIScanner{}
	database := &MockGPUDatabase{}

	device := pci.PCIDevice{Address: "0000:01:00.0", VendorID: "10de", DeviceID: "2684", Class: "030000"}
	scanner.On("ScanNVIDIA", mock.Anything).Return([]pci.PCIDevice{device}, nil)

	model := &nvidia.GPUModel{DeviceID: "2684", Name: "GeForce RTX 4090"}
	database.On("Lookup", "2684").Return(model, true)

	o := NewOrchestrator(
		WithPCIScanner(scanner),
		WithGPUDatabase(database),
	)

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = o.DetectGPUs(ctx)
	}
}

func BenchmarkOrchestrator_GetDriverStatus(b *testing.B) {
	scanner := &MockPCIScanner{}
	parser := &MockSMIParser{}

	device := pci.PCIDevice{Address: "0000:01:00.0", VendorID: "10de", DeviceID: "2684", Driver: "nvidia"}
	scanner.On("ScanNVIDIA", mock.Anything).Return([]pci.PCIDevice{device}, nil)
	parser.On("IsAvailable", mock.Anything).Return(true)
	parser.On("GetDriverVersion", mock.Anything).Return("550.54.14", nil)
	parser.On("GetCUDAVersion", mock.Anything).Return("12.4", nil)

	o := NewOrchestrator(
		WithPCIScanner(scanner),
		WithSMIParser(parser),
	)

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = o.GetDriverStatus(ctx)
	}
}

func BenchmarkOrchestrator_IsReadyForInstall(b *testing.B) {
	scanner := &MockPCIScanner{}
	database := &MockGPUDatabase{}
	val := &MockValidator{}

	device := pci.PCIDevice{Address: "0000:01:00.0", VendorID: "10de", DeviceID: "2684", Class: "030000"}
	scanner.On("ScanNVIDIA", mock.Anything).Return([]pci.PCIDevice{device}, nil)

	model := &nvidia.GPUModel{DeviceID: "2684", Name: "GeForce RTX 4090"}
	database.On("Lookup", "2684").Return(model, true)

	report := createPassingValidationReport()
	val.On("Validate", mock.Anything).Return(report, nil)

	o := NewOrchestrator(
		WithPCIScanner(scanner),
		WithGPUDatabase(database),
		WithSystemValidator(val),
	)

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = o.IsReadyForInstall(ctx)
	}
}

func BenchmarkOrchestrator_Concurrent(b *testing.B) {
	scanner := &MockPCIScanner{}
	database := &MockGPUDatabase{}

	device := pci.PCIDevice{Address: "0000:01:00.0", VendorID: "10de", DeviceID: "2684", Class: "030000"}
	scanner.On("ScanNVIDIA", mock.Anything).Return([]pci.PCIDevice{device}, nil)

	model := &nvidia.GPUModel{DeviceID: "2684", Name: "GeForce RTX 4090"}
	database.On("Lookup", "2684").Return(model, true)

	o := NewOrchestrator(
		WithPCIScanner(scanner),
		WithGPUDatabase(database),
	)

	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = o.DetectGPUs(ctx)
		}
	})
}
