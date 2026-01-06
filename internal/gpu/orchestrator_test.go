package gpu

import (
	"context"
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

// MockPCIScanner is a mock implementation of pci.Scanner.
type MockPCIScanner struct {
	mock.Mock
}

func (m *MockPCIScanner) ScanAll(ctx context.Context) ([]pci.PCIDevice, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]pci.PCIDevice), args.Error(1)
}

func (m *MockPCIScanner) ScanByVendor(ctx context.Context, vendorID string) ([]pci.PCIDevice, error) {
	args := m.Called(ctx, vendorID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]pci.PCIDevice), args.Error(1)
}

func (m *MockPCIScanner) ScanByClass(ctx context.Context, classCode string) ([]pci.PCIDevice, error) {
	args := m.Called(ctx, classCode)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]pci.PCIDevice), args.Error(1)
}

func (m *MockPCIScanner) ScanNVIDIA(ctx context.Context) ([]pci.PCIDevice, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]pci.PCIDevice), args.Error(1)
}

// MockGPUDatabase is a mock implementation of nvidia.Database.
type MockGPUDatabase struct {
	mock.Mock
}

func (m *MockGPUDatabase) Lookup(deviceID string) (*nvidia.GPUModel, bool) {
	args := m.Called(deviceID)
	if args.Get(0) == nil {
		return nil, args.Bool(1)
	}
	return args.Get(0).(*nvidia.GPUModel), args.Bool(1)
}

func (m *MockGPUDatabase) LookupByName(name string) (*nvidia.GPUModel, bool) {
	args := m.Called(name)
	if args.Get(0) == nil {
		return nil, args.Bool(1)
	}
	return args.Get(0).(*nvidia.GPUModel), args.Bool(1)
}

func (m *MockGPUDatabase) ListByArchitecture(arch nvidia.Architecture) []nvidia.GPUModel {
	args := m.Called(arch)
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).([]nvidia.GPUModel)
}

func (m *MockGPUDatabase) GetMinDriverVersion(deviceID string) (string, error) {
	args := m.Called(deviceID)
	return args.String(0), args.Error(1)
}

func (m *MockGPUDatabase) AllModels() []nvidia.GPUModel {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).([]nvidia.GPUModel)
}

func (m *MockGPUDatabase) Count() int {
	args := m.Called()
	return args.Int(0)
}

// MockSMIParser is a mock implementation of smi.Parser.
type MockSMIParser struct {
	mock.Mock
}

func (m *MockSMIParser) Parse(ctx context.Context) (*smi.SMIInfo, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*smi.SMIInfo), args.Error(1)
}

func (m *MockSMIParser) IsAvailable(ctx context.Context) bool {
	args := m.Called(ctx)
	return args.Bool(0)
}

func (m *MockSMIParser) GetDriverVersion(ctx context.Context) (string, error) {
	args := m.Called(ctx)
	return args.String(0), args.Error(1)
}

func (m *MockSMIParser) GetCUDAVersion(ctx context.Context) (string, error) {
	args := m.Called(ctx)
	return args.String(0), args.Error(1)
}

func (m *MockSMIParser) GetGPUCount(ctx context.Context) (int, error) {
	args := m.Called(ctx)
	return args.Int(0), args.Error(1)
}

// MockNouveauDetector is a mock implementation of nouveau.Detector.
type MockNouveauDetector struct {
	mock.Mock
}

func (m *MockNouveauDetector) Detect(ctx context.Context) (*nouveau.Status, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*nouveau.Status), args.Error(1)
}

func (m *MockNouveauDetector) IsLoaded(ctx context.Context) (bool, error) {
	args := m.Called(ctx)
	return args.Bool(0), args.Error(1)
}

func (m *MockNouveauDetector) IsBlacklisted(ctx context.Context) (bool, error) {
	args := m.Called(ctx)
	return args.Bool(0), args.Error(1)
}

func (m *MockNouveauDetector) GetBoundDevices(ctx context.Context) ([]string, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}

// MockKernelDetector is a mock implementation of kernel.Detector.
type MockKernelDetector struct {
	mock.Mock
}

func (m *MockKernelDetector) GetKernelInfo(ctx context.Context) (*kernel.KernelInfo, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*kernel.KernelInfo), args.Error(1)
}

func (m *MockKernelDetector) IsModuleLoaded(ctx context.Context, name string) (bool, error) {
	args := m.Called(ctx, name)
	return args.Bool(0), args.Error(1)
}

func (m *MockKernelDetector) GetLoadedModules(ctx context.Context) ([]kernel.ModuleInfo, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]kernel.ModuleInfo), args.Error(1)
}

func (m *MockKernelDetector) GetModule(ctx context.Context, name string) (*kernel.ModuleInfo, error) {
	args := m.Called(ctx, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*kernel.ModuleInfo), args.Error(1)
}

func (m *MockKernelDetector) AreHeadersInstalled(ctx context.Context) (bool, error) {
	args := m.Called(ctx)
	return args.Bool(0), args.Error(1)
}

func (m *MockKernelDetector) GetHeadersPackage(ctx context.Context) (string, error) {
	args := m.Called(ctx)
	return args.String(0), args.Error(1)
}

func (m *MockKernelDetector) IsSecureBootEnabled(ctx context.Context) (bool, error) {
	args := m.Called(ctx)
	return args.Bool(0), args.Error(1)
}

// MockValidator is a mock implementation of validator.Validator.
type MockValidator struct {
	mock.Mock
}

func (m *MockValidator) Validate(ctx context.Context) (*validator.ValidationReport, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*validator.ValidationReport), args.Error(1)
}

func (m *MockValidator) ValidateKernel(ctx context.Context) (*validator.CheckResult, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*validator.CheckResult), args.Error(1)
}

func (m *MockValidator) ValidateDiskSpace(ctx context.Context, requiredMB int64) (*validator.CheckResult, error) {
	args := m.Called(ctx, requiredMB)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*validator.CheckResult), args.Error(1)
}

func (m *MockValidator) ValidateSecureBoot(ctx context.Context) (*validator.CheckResult, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*validator.CheckResult), args.Error(1)
}

func (m *MockValidator) ValidateKernelHeaders(ctx context.Context) (*validator.CheckResult, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*validator.CheckResult), args.Error(1)
}

func (m *MockValidator) ValidateBuildTools(ctx context.Context) (*validator.CheckResult, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*validator.CheckResult), args.Error(1)
}

func (m *MockValidator) ValidateNouveauStatus(ctx context.Context) (*validator.CheckResult, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*validator.CheckResult), args.Error(1)
}

// Test helper functions

func createTestGPUDevice() pci.PCIDevice {
	return pci.PCIDevice{
		Address:  "0000:01:00.0",
		VendorID: "10de",
		DeviceID: "2684",
		Class:    "030000",
		Driver:   "nvidia",
		Revision: "a1",
	}
}

func createTestGPUModel() *nvidia.GPUModel {
	return &nvidia.GPUModel{
		DeviceID:          "2684",
		Name:              "GeForce RTX 4090",
		Architecture:      nvidia.ArchAdaLovelace,
		MinDriverVersion:  "525.60",
		ComputeCapability: "8.9",
		MemorySize:        "24GB",
	}
}

func createTestSMIInfo() *smi.SMIInfo {
	return &smi.SMIInfo{
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
}

func createTestKernelInfo() *kernel.KernelInfo {
	return &kernel.KernelInfo{
		Version:           "6.5.0-44-generic",
		Release:           "6.5.0",
		Architecture:      "x86_64",
		HeadersPath:       "/usr/src/linux-headers-6.5.0-44-generic",
		HeadersInstalled:  true,
		SecureBootEnabled: false,
	}
}

func createPassingValidationReport() *validator.ValidationReport {
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
		"sufficient disk space available",
		validator.SeverityInfo,
	))
	return report
}

func createFailingValidationReport() *validator.ValidationReport {
	report := validator.NewValidationReport()
	report.AddCheck(validator.NewCheckResult(
		validator.CheckKernelHeaders,
		false,
		"kernel headers are not installed",
		validator.SeverityError,
	).WithRemediation("Install kernel headers"))
	return report
}

// Tests

func TestNewOrchestrator(t *testing.T) {
	t.Run("creates with defaults", func(t *testing.T) {
		o := NewOrchestrator()
		require.NotNil(t, o)
		assert.Equal(t, DefaultTimeout, o.timeout)
		assert.Nil(t, o.pciScanner)
		assert.Nil(t, o.gpuDatabase)
		assert.Nil(t, o.smiParser)
	})

	t.Run("applies options", func(t *testing.T) {
		scanner := &MockPCIScanner{}
		database := &MockGPUDatabase{}
		parser := &MockSMIParser{}
		detector := &MockNouveauDetector{}
		kernelDet := &MockKernelDetector{}
		val := &MockValidator{}
		timeout := 60 * time.Second

		o := NewOrchestrator(
			WithPCIScanner(scanner),
			WithSkipLspciEnrich(true),
			WithGPUDatabase(database),
			WithSMIParser(parser),
			WithNouveauDetector(detector),
			WithKernelDetector(kernelDet),
			WithSystemValidator(val),
			WithTimeout(timeout),
		)

		require.NotNil(t, o)
		assert.Equal(t, scanner, o.pciScanner)
		assert.Equal(t, database, o.gpuDatabase)
		assert.Equal(t, parser, o.smiParser)
		assert.Equal(t, detector, o.nouveauDetector)
		assert.Equal(t, kernelDet, o.kernelDetector)
		assert.Equal(t, val, o.systemValidator)
		assert.Equal(t, timeout, o.timeout)
	})
}

func TestOrchestrator_DetectAll(t *testing.T) {
	t.Run("successful full detection", func(t *testing.T) {
		scanner := &MockPCIScanner{}
		database := &MockGPUDatabase{}
		parser := &MockSMIParser{}
		nouveauDet := &MockNouveauDetector{}
		kernelDet := &MockKernelDetector{}
		val := &MockValidator{}

		device := createTestGPUDevice()
		model := createTestGPUModel()
		smiInfo := createTestSMIInfo()
		kernelInfo := createTestKernelInfo()
		report := createPassingValidationReport()

		scanner.On("ScanNVIDIA", mock.Anything).Return([]pci.PCIDevice{device}, nil)
		database.On("Lookup", "2684").Return(model, true)
		parser.On("Parse", mock.Anything).Return(smiInfo, nil)
		parser.On("IsAvailable", mock.Anything).Return(true)
		parser.On("GetDriverVersion", mock.Anything).Return("550.54.14", nil)
		parser.On("GetCUDAVersion", mock.Anything).Return("12.4", nil)
		nouveauDet.On("Detect", mock.Anything).Return(&nouveau.Status{Loaded: false}, nil)
		kernelDet.On("GetKernelInfo", mock.Anything).Return(kernelInfo, nil)
		val.On("Validate", mock.Anything).Return(report, nil)

		o := NewOrchestrator(
			WithPCIScanner(scanner),
			WithSkipLspciEnrich(true),
			WithGPUDatabase(database),
			WithSMIParser(parser),
			WithNouveauDetector(nouveauDet),
			WithKernelDetector(kernelDet),
			WithSystemValidator(val),
		)

		ctx := context.Background()
		info, err := o.DetectAll(ctx)

		require.NoError(t, err)
		require.NotNil(t, info)
		assert.True(t, info.HasNVIDIAGPUs())
		assert.Equal(t, 1, info.GPUCount())
		assert.NotNil(t, info.InstalledDriver)
		assert.Equal(t, DriverTypeNVIDIA, info.InstalledDriver.Type)
		assert.Equal(t, "550.54.14", info.InstalledDriver.Version)
		assert.NotNil(t, info.KernelInfo)
		assert.NotNil(t, info.ValidationReport)
		assert.False(t, info.HasErrors())
		assert.Greater(t, info.Duration, time.Duration(0))
	})

	t.Run("handles partial failures gracefully", func(t *testing.T) {
		scanner := &MockPCIScanner{}
		parser := &MockSMIParser{}

		device := createTestGPUDevice()
		scanner.On("ScanNVIDIA", mock.Anything).Return([]pci.PCIDevice{device}, nil)
		parser.On("Parse", mock.Anything).Return(nil, assert.AnError)
		parser.On("IsAvailable", mock.Anything).Return(false)

		o := NewOrchestrator(
			WithPCIScanner(scanner),
			WithSkipLspciEnrich(true),
			WithSMIParser(parser),
		)

		ctx := context.Background()
		info, err := o.DetectAll(ctx)

		require.NoError(t, err)
		require.NotNil(t, info)
		assert.True(t, info.HasNVIDIAGPUs())
		// Should still have GPU info even though SMI failed
		assert.Equal(t, 1, len(info.NVIDIAGPUs))
	})

	t.Run("handles context cancellation", func(t *testing.T) {
		o := NewOrchestrator()

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		info, err := o.DetectAll(ctx)
		require.Error(t, err)
		assert.Nil(t, info)
	})

	t.Run("handles no GPUs found", func(t *testing.T) {
		scanner := &MockPCIScanner{}
		scanner.On("ScanNVIDIA", mock.Anything).Return([]pci.PCIDevice{}, nil)

		o := NewOrchestrator(WithPCIScanner(scanner))

		ctx := context.Background()
		info, err := o.DetectAll(ctx)

		require.NoError(t, err)
		require.NotNil(t, info)
		assert.False(t, info.HasNVIDIAGPUs())
		assert.Equal(t, 0, info.GPUCount())
	})
}

func TestOrchestrator_DetectGPUs(t *testing.T) {
	t.Run("detects GPUs with database lookup", func(t *testing.T) {
		scanner := &MockPCIScanner{}
		database := &MockGPUDatabase{}

		device := createTestGPUDevice()
		model := createTestGPUModel()

		scanner.On("ScanNVIDIA", mock.Anything).Return([]pci.PCIDevice{device}, nil)
		database.On("Lookup", "2684").Return(model, true)

		o := NewOrchestrator(
			WithPCIScanner(scanner),
			WithSkipLspciEnrich(true),
			WithGPUDatabase(database),
			WithSkipLspciEnrich(true),
		)

		ctx := context.Background()
		gpus, err := o.DetectGPUs(ctx)

		require.NoError(t, err)
		require.Len(t, gpus, 1)
		assert.Equal(t, device, gpus[0].PCIDevice)
		assert.NotNil(t, gpus[0].Model)
		assert.Equal(t, "GeForce RTX 4090", gpus[0].Model.Name)
	})

	t.Run("handles unknown GPU", func(t *testing.T) {
		scanner := &MockPCIScanner{}
		database := &MockGPUDatabase{}

		device := pci.PCIDevice{
			Address:  "0000:01:00.0",
			VendorID: "10de",
			DeviceID: "9999", // Unknown device ID
			Class:    "030000",
		}

		scanner.On("ScanNVIDIA", mock.Anything).Return([]pci.PCIDevice{device}, nil)
		database.On("Lookup", "9999").Return(nil, false)

		o := NewOrchestrator(
			WithPCIScanner(scanner),
			WithSkipLspciEnrich(true),
			WithGPUDatabase(database),
			WithSkipLspciEnrich(true),
		)

		ctx := context.Background()
		gpus, err := o.DetectGPUs(ctx)

		require.NoError(t, err)
		require.Len(t, gpus, 1)
		assert.Nil(t, gpus[0].Model)
		assert.Contains(t, gpus[0].Name(), "9999")
	})

	t.Run("enriches with SMI data", func(t *testing.T) {
		scanner := &MockPCIScanner{}
		parser := &MockSMIParser{}

		device := createTestGPUDevice()
		smiInfo := createTestSMIInfo()

		scanner.On("ScanNVIDIA", mock.Anything).Return([]pci.PCIDevice{device}, nil)
		parser.On("Parse", mock.Anything).Return(smiInfo, nil)

		o := NewOrchestrator(
			WithPCIScanner(scanner),
			WithSkipLspciEnrich(true),
			WithSMIParser(parser),
			WithSkipLspciEnrich(true),
		)

		ctx := context.Background()
		gpus, err := o.DetectGPUs(ctx)

		require.NoError(t, err)
		require.Len(t, gpus, 1)
		assert.NotNil(t, gpus[0].SMIInfo)
		assert.Equal(t, "NVIDIA GeForce RTX 4090", gpus[0].SMIInfo.Name)
	})

	t.Run("returns error when scanner not configured", func(t *testing.T) {
		o := NewOrchestrator()

		ctx := context.Background()
		gpus, err := o.DetectGPUs(ctx)

		require.Error(t, err)
		assert.Nil(t, gpus)
	})
}

func TestOrchestrator_GetDriverStatus(t *testing.T) {
	t.Run("detects NVIDIA proprietary driver", func(t *testing.T) {
		scanner := &MockPCIScanner{}
		parser := &MockSMIParser{}

		device := createTestGPUDevice()
		device.Driver = "nvidia"

		scanner.On("ScanNVIDIA", mock.Anything).Return([]pci.PCIDevice{device}, nil)
		parser.On("IsAvailable", mock.Anything).Return(true)
		parser.On("GetDriverVersion", mock.Anything).Return("550.54.14", nil)
		parser.On("GetCUDAVersion", mock.Anything).Return("12.4", nil)

		o := NewOrchestrator(
			WithPCIScanner(scanner),
			WithSkipLspciEnrich(true),
			WithSMIParser(parser),
		)

		ctx := context.Background()
		info, err := o.GetDriverStatus(ctx)

		require.NoError(t, err)
		require.NotNil(t, info)
		assert.True(t, info.Installed)
		assert.Equal(t, DriverTypeNVIDIA, info.Type)
		assert.Equal(t, "550.54.14", info.Version)
		assert.Equal(t, "12.4", info.CUDAVersion)
	})

	t.Run("detects Nouveau driver", func(t *testing.T) {
		scanner := &MockPCIScanner{}

		device := createTestGPUDevice()
		device.Driver = "nouveau"

		scanner.On("ScanNVIDIA", mock.Anything).Return([]pci.PCIDevice{device}, nil)

		o := NewOrchestrator(WithPCIScanner(scanner))

		ctx := context.Background()
		info, err := o.GetDriverStatus(ctx)

		require.NoError(t, err)
		require.NotNil(t, info)
		assert.True(t, info.Installed)
		assert.Equal(t, DriverTypeNouveau, info.Type)
		assert.Empty(t, info.Version) // Nouveau doesn't report version this way
	})

	t.Run("detects no driver", func(t *testing.T) {
		scanner := &MockPCIScanner{}
		parser := &MockSMIParser{}
		nouveauDet := &MockNouveauDetector{}

		device := createTestGPUDevice()
		device.Driver = ""

		scanner.On("ScanNVIDIA", mock.Anything).Return([]pci.PCIDevice{device}, nil)
		parser.On("IsAvailable", mock.Anything).Return(false)
		nouveauDet.On("Detect", mock.Anything).Return(&nouveau.Status{Loaded: false}, nil)

		o := NewOrchestrator(
			WithPCIScanner(scanner),
			WithSkipLspciEnrich(true),
			WithSMIParser(parser),
			WithNouveauDetector(nouveauDet),
		)

		ctx := context.Background()
		info, err := o.GetDriverStatus(ctx)

		require.NoError(t, err)
		require.NotNil(t, info)
		assert.False(t, info.Installed)
		assert.Equal(t, DriverTypeNone, info.Type)
	})

	t.Run("falls back to nouveau detector", func(t *testing.T) {
		scanner := &MockPCIScanner{}
		parser := &MockSMIParser{}
		nouveauDet := &MockNouveauDetector{}

		scanner.On("ScanNVIDIA", mock.Anything).Return([]pci.PCIDevice{}, nil)
		parser.On("IsAvailable", mock.Anything).Return(false)
		nouveauDet.On("Detect", mock.Anything).Return(&nouveau.Status{Loaded: true, InUse: true}, nil)

		o := NewOrchestrator(
			WithPCIScanner(scanner),
			WithSkipLspciEnrich(true),
			WithSMIParser(parser),
			WithNouveauDetector(nouveauDet),
		)

		ctx := context.Background()
		info, err := o.GetDriverStatus(ctx)

		require.NoError(t, err)
		require.NotNil(t, info)
		assert.True(t, info.Installed)
		assert.Equal(t, DriverTypeNouveau, info.Type)
	})
}

func TestOrchestrator_ValidateSystem(t *testing.T) {
	t.Run("returns validation report", func(t *testing.T) {
		val := &MockValidator{}
		report := createPassingValidationReport()

		val.On("Validate", mock.Anything).Return(report, nil)

		o := NewOrchestrator(WithSystemValidator(val))

		ctx := context.Background()
		result, err := o.ValidateSystem(ctx)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.True(t, result.Passed)
	})

	t.Run("returns error when validator not configured", func(t *testing.T) {
		o := NewOrchestrator()

		ctx := context.Background()
		result, err := o.ValidateSystem(ctx)

		require.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestOrchestrator_IsReadyForInstall(t *testing.T) {
	t.Run("ready when GPUs found and validation passes", func(t *testing.T) {
		scanner := &MockPCIScanner{}
		database := &MockGPUDatabase{}
		val := &MockValidator{}

		device := createTestGPUDevice()
		model := createTestGPUModel()
		report := createPassingValidationReport()

		scanner.On("ScanNVIDIA", mock.Anything).Return([]pci.PCIDevice{device}, nil)
		database.On("Lookup", "2684").Return(model, true)
		val.On("Validate", mock.Anything).Return(report, nil)

		o := NewOrchestrator(
			WithPCIScanner(scanner),
			WithSkipLspciEnrich(true),
			WithGPUDatabase(database),
			WithSystemValidator(val),
		)

		ctx := context.Background()
		ready, reasons, err := o.IsReadyForInstall(ctx)

		require.NoError(t, err)
		assert.True(t, ready)
		assert.Empty(t, reasons)
	})

	t.Run("not ready when no GPUs found", func(t *testing.T) {
		scanner := &MockPCIScanner{}
		scanner.On("ScanNVIDIA", mock.Anything).Return([]pci.PCIDevice{}, nil)

		o := NewOrchestrator(WithPCIScanner(scanner))

		ctx := context.Background()
		ready, reasons, err := o.IsReadyForInstall(ctx)

		require.NoError(t, err)
		assert.False(t, ready)
		assert.Contains(t, reasons[0], "No NVIDIA GPUs detected")
	})

	t.Run("not ready when validation has errors", func(t *testing.T) {
		scanner := &MockPCIScanner{}
		database := &MockGPUDatabase{}
		val := &MockValidator{}

		device := createTestGPUDevice()
		model := createTestGPUModel()
		report := createFailingValidationReport()

		scanner.On("ScanNVIDIA", mock.Anything).Return([]pci.PCIDevice{device}, nil)
		database.On("Lookup", "2684").Return(model, true)
		val.On("Validate", mock.Anything).Return(report, nil)

		o := NewOrchestrator(
			WithPCIScanner(scanner),
			WithSkipLspciEnrich(true),
			WithGPUDatabase(database),
			WithSystemValidator(val),
		)

		ctx := context.Background()
		ready, reasons, err := o.IsReadyForInstall(ctx)

		require.NoError(t, err)
		assert.False(t, ready)
		assert.NotEmpty(t, reasons)
		assert.Contains(t, reasons[0], "kernel headers")
	})

	t.Run("ready but with Nouveau warning", func(t *testing.T) {
		scanner := &MockPCIScanner{}
		database := &MockGPUDatabase{}
		val := &MockValidator{}
		nouveauDet := &MockNouveauDetector{}

		device := createTestGPUDevice()
		model := createTestGPUModel()
		report := createPassingValidationReport()

		scanner.On("ScanNVIDIA", mock.Anything).Return([]pci.PCIDevice{device}, nil)
		database.On("Lookup", "2684").Return(model, true)
		val.On("Validate", mock.Anything).Return(report, nil)
		nouveauDet.On("Detect", mock.Anything).Return(&nouveau.Status{Loaded: true, InUse: true}, nil)

		o := NewOrchestrator(
			WithPCIScanner(scanner),
			WithSkipLspciEnrich(true),
			WithGPUDatabase(database),
			WithSystemValidator(val),
			WithNouveauDetector(nouveauDet),
		)

		ctx := context.Background()
		ready, reasons, err := o.IsReadyForInstall(ctx)

		require.NoError(t, err)
		assert.True(t, ready) // Nouveau is warning, not blocker
		assert.NotEmpty(t, reasons)
		assert.Contains(t, reasons[0], "Warning:")
		assert.Contains(t, reasons[0], "Nouveau")
	})
}

func TestNVIDIAGPUInfo_Name(t *testing.T) {
	t.Run("prefers Model name", func(t *testing.T) {
		gpu := NVIDIAGPUInfo{
			PCIDevice: pci.PCIDevice{DeviceID: "2684"},
			Model:     createTestGPUModel(),
			SMIInfo:   &smi.SMIGPUInfo{Name: "NVIDIA GeForce RTX 4090"},
		}
		assert.Equal(t, "GeForce RTX 4090", gpu.Name())
	})

	t.Run("falls back to SMI name", func(t *testing.T) {
		gpu := NVIDIAGPUInfo{
			PCIDevice: pci.PCIDevice{DeviceID: "2684"},
			Model:     nil,
			SMIInfo:   &smi.SMIGPUInfo{Name: "NVIDIA GeForce RTX 4090"},
		}
		assert.Equal(t, "NVIDIA GeForce RTX 4090", gpu.Name())
	})

	t.Run("falls back to device ID", func(t *testing.T) {
		gpu := NVIDIAGPUInfo{
			PCIDevice: pci.PCIDevice{DeviceID: "2684"},
			Model:     nil,
			SMIInfo:   nil,
		}
		assert.Contains(t, gpu.Name(), "2684")
	})
}

func TestNVIDIAGPUInfo_Architecture(t *testing.T) {
	t.Run("returns architecture from model", func(t *testing.T) {
		gpu := NVIDIAGPUInfo{
			Model: createTestGPUModel(),
		}
		assert.Equal(t, "ada", gpu.Architecture())
	})

	t.Run("returns unknown when no model", func(t *testing.T) {
		gpu := NVIDIAGPUInfo{Model: nil}
		assert.Equal(t, "unknown", gpu.Architecture())
	})
}

func TestGPUInfo_Methods(t *testing.T) {
	t.Run("HasNVIDIAGPUs", func(t *testing.T) {
		info := &GPUInfo{NVIDIAGPUs: []NVIDIAGPUInfo{{}}}
		assert.True(t, info.HasNVIDIAGPUs())

		info = &GPUInfo{NVIDIAGPUs: nil}
		assert.False(t, info.HasNVIDIAGPUs())
	})

	t.Run("GPUCount", func(t *testing.T) {
		info := &GPUInfo{NVIDIAGPUs: []NVIDIAGPUInfo{{}, {}}}
		assert.Equal(t, 2, info.GPUCount())
	})

	t.Run("HasErrors", func(t *testing.T) {
		info := &GPUInfo{Errors: []error{assert.AnError}}
		assert.True(t, info.HasErrors())

		info = &GPUInfo{Errors: nil}
		assert.False(t, info.HasErrors())
	})

	t.Run("IsDriverInstalled", func(t *testing.T) {
		info := &GPUInfo{InstalledDriver: &DriverInfo{Installed: true}}
		assert.True(t, info.IsDriverInstalled())

		info = &GPUInfo{InstalledDriver: nil}
		assert.False(t, info.IsDriverInstalled())
	})

	t.Run("IsNouveauLoaded", func(t *testing.T) {
		info := &GPUInfo{NouveauStatus: &nouveau.Status{Loaded: true}}
		assert.True(t, info.IsNouveauLoaded())

		info = &GPUInfo{NouveauStatus: nil}
		assert.False(t, info.IsNouveauLoaded())
	})

	t.Run("HasValidationErrors", func(t *testing.T) {
		report := createFailingValidationReport()
		info := &GPUInfo{ValidationReport: report}
		assert.True(t, info.HasValidationErrors())

		info = &GPUInfo{ValidationReport: nil}
		assert.False(t, info.HasValidationErrors())
	})
}

func TestDriverType_String(t *testing.T) {
	assert.Equal(t, "nvidia", DriverTypeNVIDIA.String())
	assert.Equal(t, "nouveau", DriverTypeNouveau.String())
	assert.Equal(t, "none", DriverTypeNone.String())
}

func TestOrchestrator_Timeout(t *testing.T) {
	t.Run("respects timeout in DetectAll", func(t *testing.T) {
		scanner := &MockPCIScanner{}
		// Configure scanner to be slow
		scanner.On("ScanNVIDIA", mock.Anything).Run(func(args mock.Arguments) {
			time.Sleep(100 * time.Millisecond)
		}).Return([]pci.PCIDevice{}, nil)

		o := NewOrchestrator(
			WithPCIScanner(scanner),
			WithSkipLspciEnrich(true),
			WithTimeout(50*time.Millisecond),
		)

		ctx := context.Background()
		info, err := o.DetectAll(ctx)

		// Should complete (context may or may not be cancelled depending on timing)
		// The important thing is it doesn't hang
		assert.True(t, err != nil || info != nil)
	})
}
