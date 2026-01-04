package validator

import (
	"context"
	"io/fs"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tungetti/igor/internal/exec"
	"github.com/tungetti/igor/internal/gpu/kernel"
	"github.com/tungetti/igor/internal/gpu/nouveau"
)

// Mock implementations for testing

// MockKernelDetector implements kernel.Detector for testing.
type MockKernelDetector struct {
	kernelInfo        *kernel.KernelInfo
	kernelInfoErr     error
	headersInstalled  bool
	headersErr        error
	headersPackage    string
	headersPackageErr error
	secureBootEnabled bool
	secureBootErr     error
}

func (m *MockKernelDetector) GetKernelInfo(ctx context.Context) (*kernel.KernelInfo, error) {
	return m.kernelInfo, m.kernelInfoErr
}

func (m *MockKernelDetector) IsModuleLoaded(ctx context.Context, name string) (bool, error) {
	return false, nil
}

func (m *MockKernelDetector) GetLoadedModules(ctx context.Context) ([]kernel.ModuleInfo, error) {
	return nil, nil
}

func (m *MockKernelDetector) GetModule(ctx context.Context, name string) (*kernel.ModuleInfo, error) {
	return nil, nil
}

func (m *MockKernelDetector) AreHeadersInstalled(ctx context.Context) (bool, error) {
	return m.headersInstalled, m.headersErr
}

func (m *MockKernelDetector) GetHeadersPackage(ctx context.Context) (string, error) {
	return m.headersPackage, m.headersPackageErr
}

func (m *MockKernelDetector) IsSecureBootEnabled(ctx context.Context) (bool, error) {
	return m.secureBootEnabled, m.secureBootErr
}

// MockNouveauDetector implements nouveau.Detector for testing.
type MockNouveauDetector struct {
	status          *nouveau.Status
	detectErr       error
	loaded          bool
	loadedErr       error
	blacklisted     bool
	blacklistedErr  error
	boundDevices    []string
	boundDevicesErr error
}

func (m *MockNouveauDetector) Detect(ctx context.Context) (*nouveau.Status, error) {
	return m.status, m.detectErr
}

func (m *MockNouveauDetector) IsLoaded(ctx context.Context) (bool, error) {
	return m.loaded, m.loadedErr
}

func (m *MockNouveauDetector) IsBlacklisted(ctx context.Context) (bool, error) {
	return m.blacklisted, m.blacklistedErr
}

func (m *MockNouveauDetector) GetBoundDevices(ctx context.Context) ([]string, error) {
	return m.boundDevices, m.boundDevicesErr
}

// MockFileSystem implements FileSystem for testing.
type MockFileSystem struct {
	stats map[string]fs.FileInfo
}

func NewMockFileSystem() *MockFileSystem {
	return &MockFileSystem{
		stats: make(map[string]fs.FileInfo),
	}
}

func (m *MockFileSystem) AddPath(path string) {
	m.stats[path] = &mockFileInfo{name: path, isDir: true}
}

func (m *MockFileSystem) Stat(name string) (fs.FileInfo, error) {
	if info, ok := m.stats[name]; ok {
		return info, nil
	}
	return nil, os.ErrNotExist
}

type mockFileInfo struct {
	name  string
	isDir bool
	size  int64
}

func (m *mockFileInfo) Name() string       { return m.name }
func (m *mockFileInfo) Size() int64        { return m.size }
func (m *mockFileInfo) Mode() fs.FileMode  { return 0755 }
func (m *mockFileInfo) ModTime() time.Time { return time.Now() }
func (m *mockFileInfo) IsDir() bool        { return m.isDir }
func (m *mockFileInfo) Sys() interface{}   { return nil }

// TestSeverity tests Severity type methods.
func TestSeverity(t *testing.T) {
	t.Run("String returns correct values", func(t *testing.T) {
		assert.Equal(t, "error", SeverityError.String())
		assert.Equal(t, "warning", SeverityWarning.String())
		assert.Equal(t, "info", SeverityInfo.String())
	})

	t.Run("IsError returns true only for error severity", func(t *testing.T) {
		assert.True(t, SeverityError.IsError())
		assert.False(t, SeverityWarning.IsError())
		assert.False(t, SeverityInfo.IsError())
	})

	t.Run("IsWarning returns true only for warning severity", func(t *testing.T) {
		assert.False(t, SeverityError.IsWarning())
		assert.True(t, SeverityWarning.IsWarning())
		assert.False(t, SeverityInfo.IsWarning())
	})

	t.Run("IsInfo returns true only for info severity", func(t *testing.T) {
		assert.False(t, SeverityError.IsInfo())
		assert.False(t, SeverityWarning.IsInfo())
		assert.True(t, SeverityInfo.IsInfo())
	})
}

// TestCheckName tests CheckName type methods.
func TestCheckName(t *testing.T) {
	assert.Equal(t, "kernel_version", CheckKernelVersion.String())
	assert.Equal(t, "kernel_headers", CheckKernelHeaders.String())
	assert.Equal(t, "disk_space", CheckDiskSpace.String())
	assert.Equal(t, "secure_boot", CheckSecureBoot.String())
	assert.Equal(t, "build_tools", CheckBuildTools.String())
	assert.Equal(t, "nouveau_status", CheckNouveauStatus.String())
}

// TestCheckResult tests CheckResult creation and methods.
func TestCheckResult(t *testing.T) {
	t.Run("NewCheckResult creates result correctly", func(t *testing.T) {
		result := NewCheckResult(CheckKernelVersion, true, "test message", SeverityInfo)
		assert.Equal(t, CheckKernelVersion, result.Name)
		assert.True(t, result.Passed)
		assert.Equal(t, "test message", result.Message)
		assert.Equal(t, SeverityInfo, result.Severity)
		assert.Empty(t, result.Remediation)
		assert.NotNil(t, result.Details)
	})

	t.Run("WithRemediation adds remediation", func(t *testing.T) {
		result := NewCheckResult(CheckDiskSpace, false, "low space", SeverityError).
			WithRemediation("Free up disk space")
		assert.Equal(t, "Free up disk space", result.Remediation)
	})

	t.Run("WithDetail adds details", func(t *testing.T) {
		result := NewCheckResult(CheckDiskSpace, false, "low space", SeverityError).
			WithDetail("available_mb", "1000").
			WithDetail("required_mb", "2000")
		assert.Equal(t, "1000", result.Details["available_mb"])
		assert.Equal(t, "2000", result.Details["required_mb"])
	})

	t.Run("String returns formatted result", func(t *testing.T) {
		passResult := NewCheckResult(CheckKernelVersion, true, "kernel OK", SeverityInfo)
		assert.Contains(t, passResult.String(), "PASS")
		assert.Contains(t, passResult.String(), "kernel_version")

		failResult := NewCheckResult(CheckDiskSpace, false, "low space", SeverityError)
		assert.Contains(t, failResult.String(), "FAIL")
		assert.Contains(t, failResult.String(), "disk_space")
	})

	t.Run("WithDetail handles nil Details map", func(t *testing.T) {
		result := &CheckResult{Name: CheckDiskSpace}
		result.WithDetail("key", "value")
		assert.Equal(t, "value", result.Details["key"])
	})
}

// TestValidationReport tests ValidationReport creation and methods.
func TestValidationReport(t *testing.T) {
	t.Run("NewValidationReport creates empty report", func(t *testing.T) {
		report := NewValidationReport()
		assert.True(t, report.Passed)
		assert.Empty(t, report.Checks)
		assert.Empty(t, report.Errors)
		assert.Empty(t, report.Warnings)
		assert.Empty(t, report.Infos)
		assert.False(t, report.Timestamp.IsZero())
	})

	t.Run("AddCheck adds passing check", func(t *testing.T) {
		report := NewValidationReport()
		result := NewCheckResult(CheckKernelVersion, true, "OK", SeverityInfo)
		report.AddCheck(result)

		assert.Len(t, report.Checks, 1)
		assert.True(t, report.Passed)
		assert.Empty(t, report.Errors)
		assert.Len(t, report.Infos, 1)
	})

	t.Run("AddCheck adds failing error check", func(t *testing.T) {
		report := NewValidationReport()
		result := NewCheckResult(CheckDiskSpace, false, "low space", SeverityError)
		report.AddCheck(result)

		assert.Len(t, report.Checks, 1)
		assert.False(t, report.Passed)
		assert.Len(t, report.Errors, 1)
	})

	t.Run("AddCheck adds failing warning check", func(t *testing.T) {
		report := NewValidationReport()
		result := NewCheckResult(CheckSecureBoot, false, "Secure Boot enabled", SeverityWarning)
		report.AddCheck(result)

		assert.Len(t, report.Checks, 1)
		assert.True(t, report.Passed) // Warnings don't fail the report
		assert.Len(t, report.Warnings, 1)
	})

	t.Run("AddCheck ignores nil result", func(t *testing.T) {
		report := NewValidationReport()
		report.AddCheck(nil)
		assert.Empty(t, report.Checks)
	})

	t.Run("HasErrors and HasWarnings work correctly", func(t *testing.T) {
		report := NewValidationReport()
		assert.False(t, report.HasErrors())
		assert.False(t, report.HasWarnings())

		report.AddCheck(NewCheckResult(CheckSecureBoot, false, "warning", SeverityWarning))
		assert.False(t, report.HasErrors())
		assert.True(t, report.HasWarnings())

		report.AddCheck(NewCheckResult(CheckDiskSpace, false, "error", SeverityError))
		assert.True(t, report.HasErrors())
		assert.True(t, report.HasWarnings())
	})

	t.Run("Count methods work correctly", func(t *testing.T) {
		report := NewValidationReport()
		report.AddCheck(NewCheckResult(CheckKernelVersion, true, "OK", SeverityInfo))
		report.AddCheck(NewCheckResult(CheckSecureBoot, false, "warning", SeverityWarning))
		report.AddCheck(NewCheckResult(CheckDiskSpace, false, "error", SeverityError))

		assert.Equal(t, 3, report.TotalChecks())
		assert.Equal(t, 1, report.PassedChecks())
		assert.Equal(t, 2, report.FailedChecks())
		assert.Equal(t, 1, report.ErrorCount())
		assert.Equal(t, 1, report.WarningCount())
	})

	t.Run("GetCheck returns correct result", func(t *testing.T) {
		report := NewValidationReport()
		report.AddCheck(NewCheckResult(CheckKernelVersion, true, "OK", SeverityInfo))
		report.AddCheck(NewCheckResult(CheckDiskSpace, false, "low", SeverityError))

		check := report.GetCheck(CheckKernelVersion)
		require.NotNil(t, check)
		assert.True(t, check.Passed)

		check = report.GetCheck(CheckDiskSpace)
		require.NotNil(t, check)
		assert.False(t, check.Passed)

		check = report.GetCheck(CheckBuildTools)
		assert.Nil(t, check)
	})

	t.Run("Summary returns formatted string", func(t *testing.T) {
		report := NewValidationReport()
		report.AddCheck(NewCheckResult(CheckKernelVersion, true, "OK", SeverityInfo))
		report.AddCheck(NewCheckResult(CheckDiskSpace, false, "low", SeverityError))

		summary := report.Summary()
		assert.Contains(t, summary, "FAILED")
		assert.Contains(t, summary, "1/2")
		assert.Contains(t, summary, "1 errors")
	})
}

// TestValidatorCreation tests validator creation with options.
func TestValidatorCreation(t *testing.T) {
	t.Run("NewValidator creates validator with defaults", func(t *testing.T) {
		v := NewValidator()
		assert.NotNil(t, v)
		assert.Equal(t, DefaultMinDiskSpaceMB, v.requiredDiskMB)
		assert.Equal(t, MinKernelMajor, v.minKernelMajor)
		assert.Equal(t, MinKernelMinor, v.minKernelMinor)
		assert.Equal(t, RequiredBuildTools, v.requiredTools)
	})

	t.Run("NewValidator applies options", func(t *testing.T) {
		mockExec := exec.NewMockExecutor()
		mockKernel := &MockKernelDetector{}
		mockNouveau := &MockNouveauDetector{}
		mockFS := NewMockFileSystem()

		v := NewValidator(
			WithExecutor(mockExec),
			WithKernelDetector(mockKernel),
			WithNouveauDetector(mockNouveau),
			WithFileSystem(mockFS),
			WithRequiredDiskSpace(10000),
			WithMinKernelVersion(5, 10),
			WithRequiredTools([]string{"gcc", "make"}),
			WithDiskCheckPaths([]string{"/home"}),
		)

		assert.Equal(t, mockExec, v.executor)
		assert.Equal(t, mockKernel, v.kernelDetector)
		assert.Equal(t, mockNouveau, v.nouveauDetector)
		assert.Equal(t, mockFS, v.fs)
		assert.Equal(t, int64(10000), v.requiredDiskMB)
		assert.Equal(t, 5, v.minKernelMajor)
		assert.Equal(t, 10, v.minKernelMinor)
		assert.Equal(t, []string{"gcc", "make"}, v.requiredTools)
		assert.Equal(t, []string{"/home"}, v.diskCheckPaths)
	})
}

// TestValidateKernel tests kernel version validation.
func TestValidateKernel(t *testing.T) {
	t.Run("passes for sufficient kernel version", func(t *testing.T) {
		mockKernel := &MockKernelDetector{
			kernelInfo: &kernel.KernelInfo{
				Version: "6.5.0-44-generic",
				Release: "6.5.0",
			},
		}

		v := NewValidator(WithKernelDetector(mockKernel))
		result, err := v.ValidateKernel(context.Background())

		require.NoError(t, err)
		assert.True(t, result.Passed)
		assert.Equal(t, CheckKernelVersion, result.Name)
		assert.Contains(t, result.Message, "compatible")
	})

	t.Run("fails for old kernel version", func(t *testing.T) {
		mockKernel := &MockKernelDetector{
			kernelInfo: &kernel.KernelInfo{
				Version: "4.10.0",
				Release: "4.10.0",
			},
		}

		v := NewValidator(WithKernelDetector(mockKernel))
		result, err := v.ValidateKernel(context.Background())

		require.NoError(t, err)
		assert.False(t, result.Passed)
		assert.Equal(t, SeverityError, result.Severity)
		assert.Contains(t, result.Message, "below minimum")
		assert.NotEmpty(t, result.Remediation)
	})

	t.Run("fails when kernel detector returns error", func(t *testing.T) {
		mockKernel := &MockKernelDetector{
			kernelInfoErr: assert.AnError,
		}

		v := NewValidator(WithKernelDetector(mockKernel))
		result, err := v.ValidateKernel(context.Background())

		require.NoError(t, err)
		assert.False(t, result.Passed)
		assert.Equal(t, SeverityError, result.Severity)
	})

	t.Run("fails when kernel detector is nil", func(t *testing.T) {
		v := NewValidator()
		result, err := v.ValidateKernel(context.Background())

		require.NoError(t, err)
		assert.False(t, result.Passed)
		assert.Contains(t, result.Message, "not available")
	})

	t.Run("handles context cancellation", func(t *testing.T) {
		v := NewValidator()
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		_, err := v.ValidateKernel(ctx)
		assert.Error(t, err)
	})

	t.Run("handles kernel 5.0+ versions", func(t *testing.T) {
		mockKernel := &MockKernelDetector{
			kernelInfo: &kernel.KernelInfo{
				Version: "5.4.0-91-generic",
				Release: "5.4.0",
			},
		}

		v := NewValidator(WithKernelDetector(mockKernel))
		result, err := v.ValidateKernel(context.Background())

		require.NoError(t, err)
		assert.True(t, result.Passed)
	})

	t.Run("handles kernel 4.15 exactly", func(t *testing.T) {
		mockKernel := &MockKernelDetector{
			kernelInfo: &kernel.KernelInfo{
				Version: "4.15.0-generic",
				Release: "4.15.0",
			},
		}

		v := NewValidator(WithKernelDetector(mockKernel))
		result, err := v.ValidateKernel(context.Background())

		require.NoError(t, err)
		assert.True(t, result.Passed)
	})
}

// TestValidateDiskSpace tests disk space validation.
func TestValidateDiskSpace(t *testing.T) {
	t.Run("passes with sufficient disk space", func(t *testing.T) {
		mockExec := exec.NewMockExecutor()
		mockExec.SetResponse("df", exec.SuccessResult("Avail\n5000M\n"))

		mockFS := NewMockFileSystem()
		mockFS.AddPath("/")

		v := NewValidator(
			WithExecutor(mockExec),
			WithFileSystem(mockFS),
			WithDiskCheckPaths([]string{"/"}),
		)

		result, err := v.ValidateDiskSpace(context.Background(), 2048)

		require.NoError(t, err)
		assert.True(t, result.Passed)
		assert.Contains(t, result.Message, "sufficient")
	})

	t.Run("fails with insufficient disk space", func(t *testing.T) {
		mockExec := exec.NewMockExecutor()
		mockExec.SetResponse("df", exec.SuccessResult("Avail\n1000M\n"))

		mockFS := NewMockFileSystem()
		mockFS.AddPath("/")

		v := NewValidator(
			WithExecutor(mockExec),
			WithFileSystem(mockFS),
			WithDiskCheckPaths([]string{"/"}),
		)

		result, err := v.ValidateDiskSpace(context.Background(), 2048)

		require.NoError(t, err)
		assert.False(t, result.Passed)
		assert.Equal(t, SeverityError, result.Severity)
		assert.Contains(t, result.Message, "insufficient")
		assert.NotEmpty(t, result.Remediation)
	})

	t.Run("fails when executor is nil", func(t *testing.T) {
		v := NewValidator()
		result, err := v.ValidateDiskSpace(context.Background(), 2048)

		require.NoError(t, err)
		assert.False(t, result.Passed)
		assert.Contains(t, result.Message, "executor not available")
	})

	t.Run("fails when df command fails", func(t *testing.T) {
		mockExec := exec.NewMockExecutor()
		mockExec.SetResponse("df", exec.FailureResult(1, "error"))

		mockFS := NewMockFileSystem()
		mockFS.AddPath("/")

		v := NewValidator(
			WithExecutor(mockExec),
			WithFileSystem(mockFS),
			WithDiskCheckPaths([]string{"/"}),
		)

		result, err := v.ValidateDiskSpace(context.Background(), 2048)

		require.NoError(t, err)
		assert.False(t, result.Passed)
	})

	t.Run("handles multiple paths", func(t *testing.T) {
		mockExec := exec.NewMockExecutor()
		// First path has enough, but we want to test that all are checked
		mockExec.SetResponse("df", exec.SuccessResult("Avail\n3000M\n"))

		mockFS := NewMockFileSystem()
		mockFS.AddPath("/usr")
		mockFS.AddPath("/var")

		v := NewValidator(
			WithExecutor(mockExec),
			WithFileSystem(mockFS),
			WithDiskCheckPaths([]string{"/usr", "/var"}),
		)

		result, err := v.ValidateDiskSpace(context.Background(), 2048)

		require.NoError(t, err)
		assert.True(t, result.Passed)
	})

	t.Run("handles context cancellation", func(t *testing.T) {
		v := NewValidator()
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		_, err := v.ValidateDiskSpace(ctx, 2048)
		assert.Error(t, err)
	})
}

// TestValidateSecureBoot tests Secure Boot validation.
func TestValidateSecureBoot(t *testing.T) {
	t.Run("passes when Secure Boot is disabled", func(t *testing.T) {
		mockKernel := &MockKernelDetector{
			secureBootEnabled: false,
		}

		v := NewValidator(WithKernelDetector(mockKernel))
		result, err := v.ValidateSecureBoot(context.Background())

		require.NoError(t, err)
		assert.True(t, result.Passed)
		assert.Contains(t, result.Message, "disabled")
	})

	t.Run("warns when Secure Boot is enabled", func(t *testing.T) {
		mockKernel := &MockKernelDetector{
			secureBootEnabled: true,
		}

		v := NewValidator(WithKernelDetector(mockKernel))
		result, err := v.ValidateSecureBoot(context.Background())

		require.NoError(t, err)
		assert.False(t, result.Passed)
		assert.Equal(t, SeverityWarning, result.Severity)
		assert.Contains(t, result.Message, "Secure Boot is enabled")
		assert.NotEmpty(t, result.Remediation)
	})

	t.Run("handles Secure Boot check error gracefully", func(t *testing.T) {
		mockKernel := &MockKernelDetector{
			secureBootErr: assert.AnError,
		}

		v := NewValidator(WithKernelDetector(mockKernel))
		result, err := v.ValidateSecureBoot(context.Background())

		require.NoError(t, err)
		assert.True(t, result.Passed) // Error is treated as info, not failure
		assert.Equal(t, SeverityInfo, result.Severity)
	})

	t.Run("passes when kernel detector is nil", func(t *testing.T) {
		v := NewValidator()
		result, err := v.ValidateSecureBoot(context.Background())

		require.NoError(t, err)
		assert.True(t, result.Passed)
		assert.Equal(t, SeverityInfo, result.Severity)
	})

	t.Run("handles context cancellation", func(t *testing.T) {
		v := NewValidator()
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		_, err := v.ValidateSecureBoot(ctx)
		assert.Error(t, err)
	})
}

// TestValidateKernelHeaders tests kernel headers validation.
func TestValidateKernelHeaders(t *testing.T) {
	t.Run("passes when headers are installed", func(t *testing.T) {
		mockKernel := &MockKernelDetector{
			headersInstalled: true,
		}

		v := NewValidator(WithKernelDetector(mockKernel))
		result, err := v.ValidateKernelHeaders(context.Background())

		require.NoError(t, err)
		assert.True(t, result.Passed)
		assert.Contains(t, result.Message, "installed")
	})

	t.Run("fails when headers are not installed", func(t *testing.T) {
		mockKernel := &MockKernelDetector{
			headersInstalled: false,
			headersPackage:   "linux-headers-6.5.0",
		}

		v := NewValidator(WithKernelDetector(mockKernel))
		result, err := v.ValidateKernelHeaders(context.Background())

		require.NoError(t, err)
		assert.False(t, result.Passed)
		assert.Equal(t, SeverityError, result.Severity)
		assert.Contains(t, result.Remediation, "linux-headers-6.5.0")
	})

	t.Run("fails when headers check returns error", func(t *testing.T) {
		mockKernel := &MockKernelDetector{
			headersErr: assert.AnError,
		}

		v := NewValidator(WithKernelDetector(mockKernel))
		result, err := v.ValidateKernelHeaders(context.Background())

		require.NoError(t, err)
		assert.False(t, result.Passed)
		assert.Equal(t, SeverityError, result.Severity)
	})

	t.Run("fails when kernel detector is nil", func(t *testing.T) {
		v := NewValidator()
		result, err := v.ValidateKernelHeaders(context.Background())

		require.NoError(t, err)
		assert.False(t, result.Passed)
	})

	t.Run("handles context cancellation", func(t *testing.T) {
		v := NewValidator()
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		_, err := v.ValidateKernelHeaders(ctx)
		assert.Error(t, err)
	})
}

// TestValidateBuildTools tests build tools validation.
func TestValidateBuildTools(t *testing.T) {
	t.Run("passes when all tools are available", func(t *testing.T) {
		mockExec := exec.NewMockExecutor()
		mockExec.SetResponse("which", exec.SuccessResult("/usr/bin/gcc\n"))

		v := NewValidator(
			WithExecutor(mockExec),
			WithRequiredTools([]string{"gcc", "make", "dkms"}),
		)

		result, err := v.ValidateBuildTools(context.Background())

		require.NoError(t, err)
		assert.True(t, result.Passed)
		assert.Contains(t, result.Message, "available")
	})

	t.Run("fails when tools are missing", func(t *testing.T) {
		mockExec := exec.NewMockExecutor()
		mockExec.SetDefaultResponse(exec.FailureResult(1, "not found"))

		v := NewValidator(
			WithExecutor(mockExec),
			WithRequiredTools([]string{"gcc", "make"}),
		)

		result, err := v.ValidateBuildTools(context.Background())

		require.NoError(t, err)
		assert.False(t, result.Passed)
		assert.Equal(t, SeverityError, result.Severity)
		assert.Contains(t, result.Message, "missing")
		assert.Contains(t, result.Message, "gcc")
	})

	t.Run("fails when executor is nil", func(t *testing.T) {
		v := NewValidator()
		result, err := v.ValidateBuildTools(context.Background())

		require.NoError(t, err)
		assert.False(t, result.Passed)
	})

	t.Run("handles context cancellation", func(t *testing.T) {
		v := NewValidator()
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		_, err := v.ValidateBuildTools(ctx)
		assert.Error(t, err)
	})
}

// TestValidateNouveauStatus tests Nouveau status validation.
func TestValidateNouveauStatus(t *testing.T) {
	t.Run("passes when Nouveau is not loaded and blacklisted", func(t *testing.T) {
		mockNouveau := &MockNouveauDetector{
			status: &nouveau.Status{
				Loaded:          false,
				BlacklistExists: true,
				BlacklistFiles:  []string{"/etc/modprobe.d/blacklist-nouveau.conf"},
			},
		}

		v := NewValidator(WithNouveauDetector(mockNouveau))
		result, err := v.ValidateNouveauStatus(context.Background())

		require.NoError(t, err)
		assert.True(t, result.Passed)
		assert.Contains(t, result.Message, "not loaded")
		assert.Contains(t, result.Message, "blacklisted")
	})

	t.Run("warns when Nouveau is loaded", func(t *testing.T) {
		mockNouveau := &MockNouveauDetector{
			status: &nouveau.Status{
				Loaded:          true,
				InUse:           true,
				BlacklistExists: false,
			},
		}

		v := NewValidator(WithNouveauDetector(mockNouveau))
		result, err := v.ValidateNouveauStatus(context.Background())

		require.NoError(t, err)
		assert.False(t, result.Passed)
		assert.Equal(t, SeverityWarning, result.Severity)
		assert.Contains(t, result.Message, "currently loaded")
		assert.NotEmpty(t, result.Remediation)
	})

	t.Run("info when Nouveau not loaded but not blacklisted", func(t *testing.T) {
		mockNouveau := &MockNouveauDetector{
			status: &nouveau.Status{
				Loaded:          false,
				BlacklistExists: false,
			},
		}

		v := NewValidator(WithNouveauDetector(mockNouveau))
		result, err := v.ValidateNouveauStatus(context.Background())

		require.NoError(t, err)
		assert.True(t, result.Passed)
		assert.Equal(t, SeverityInfo, result.Severity)
	})

	t.Run("handles detect error", func(t *testing.T) {
		mockNouveau := &MockNouveauDetector{
			detectErr: assert.AnError,
		}

		v := NewValidator(WithNouveauDetector(mockNouveau))
		result, err := v.ValidateNouveauStatus(context.Background())

		require.NoError(t, err)
		assert.False(t, result.Passed)
		assert.Equal(t, SeverityWarning, result.Severity)
	})

	t.Run("passes when nouveau detector is nil", func(t *testing.T) {
		v := NewValidator()
		result, err := v.ValidateNouveauStatus(context.Background())

		require.NoError(t, err)
		assert.True(t, result.Passed)
		assert.Equal(t, SeverityInfo, result.Severity)
	})

	t.Run("handles context cancellation", func(t *testing.T) {
		v := NewValidator()
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		_, err := v.ValidateNouveauStatus(ctx)
		assert.Error(t, err)
	})

	t.Run("warns with blacklist exists but still loaded", func(t *testing.T) {
		mockNouveau := &MockNouveauDetector{
			status: &nouveau.Status{
				Loaded:          true,
				BlacklistExists: true,
			},
		}

		v := NewValidator(WithNouveauDetector(mockNouveau))
		result, err := v.ValidateNouveauStatus(context.Background())

		require.NoError(t, err)
		assert.False(t, result.Passed)
		assert.Contains(t, result.Remediation, "reboot")
	})
}

// TestValidate tests the full validation workflow.
func TestValidate(t *testing.T) {
	t.Run("returns report with all checks", func(t *testing.T) {
		mockExec := exec.NewMockExecutor()
		mockExec.SetResponse("df", exec.SuccessResult("Avail\n5000M\n"))
		mockExec.SetResponse("which", exec.SuccessResult("/usr/bin/gcc\n"))

		mockKernel := &MockKernelDetector{
			kernelInfo: &kernel.KernelInfo{
				Version: "6.5.0-44-generic",
				Release: "6.5.0",
			},
			headersInstalled:  true,
			secureBootEnabled: false,
		}

		mockNouveau := &MockNouveauDetector{
			status: &nouveau.Status{
				Loaded:          false,
				BlacklistExists: true,
			},
		}

		mockFS := NewMockFileSystem()
		mockFS.AddPath("/")

		v := NewValidator(
			WithExecutor(mockExec),
			WithKernelDetector(mockKernel),
			WithNouveauDetector(mockNouveau),
			WithFileSystem(mockFS),
			WithDiskCheckPaths([]string{"/"}),
		)

		report, err := v.Validate(context.Background())

		require.NoError(t, err)
		assert.NotNil(t, report)
		assert.True(t, report.Passed)
		assert.GreaterOrEqual(t, len(report.Checks), 6)
		assert.False(t, report.Duration == 0)
	})

	t.Run("handles context cancellation", func(t *testing.T) {
		v := NewValidator()
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		_, err := v.Validate(ctx)
		assert.Error(t, err)
	})

	t.Run("continues with other checks when one fails", func(t *testing.T) {
		mockExec := exec.NewMockExecutor()
		mockExec.SetResponse("df", exec.SuccessResult("Avail\n100M\n")) // Low disk space
		mockExec.SetResponse("which", exec.SuccessResult("/usr/bin/gcc\n"))

		mockKernel := &MockKernelDetector{
			kernelInfo: &kernel.KernelInfo{
				Version: "6.5.0",
				Release: "6.5.0",
			},
			headersInstalled: true,
		}

		mockNouveau := &MockNouveauDetector{
			status: &nouveau.Status{Loaded: false, BlacklistExists: true},
		}

		mockFS := NewMockFileSystem()
		mockFS.AddPath("/")

		v := NewValidator(
			WithExecutor(mockExec),
			WithKernelDetector(mockKernel),
			WithNouveauDetector(mockNouveau),
			WithFileSystem(mockFS),
			WithDiskCheckPaths([]string{"/"}),
		)

		report, err := v.Validate(context.Background())

		require.NoError(t, err)
		assert.False(t, report.Passed)
		assert.GreaterOrEqual(t, len(report.Checks), 6)
		assert.True(t, report.HasErrors())
	})
}

// TestParseKernelVersion tests kernel version parsing.
func TestParseKernelVersion(t *testing.T) {
	tests := []struct {
		version string
		major   int
		minor   int
		patch   int
		wantErr bool
	}{
		{"6.5.0", 6, 5, 0, false},
		{"5.15.0", 5, 15, 0, false},
		{"4.15.0", 4, 15, 0, false},
		{"6.5", 6, 5, 0, false},
		{"6.5.0-44-generic", 6, 5, 0, false},
		{"5.14.0-284.el9.x86_64", 5, 14, 0, false},
		{"invalid", 0, 0, 0, true},
		{"", 0, 0, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			major, minor, patch, err := parseKernelVersion(tt.version)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.major, major)
				assert.Equal(t, tt.minor, minor)
				assert.Equal(t, tt.patch, patch)
			}
		})
	}
}

// TestIsKernelVersionSufficient tests kernel version comparison.
func TestIsKernelVersionSufficient(t *testing.T) {
	tests := []struct {
		major    int
		minor    int
		minMajor int
		minMinor int
		expected bool
	}{
		{6, 5, 4, 15, true},   // 6.5 > 4.15
		{5, 0, 4, 15, true},   // 5.0 > 4.15
		{4, 15, 4, 15, true},  // 4.15 = 4.15
		{4, 16, 4, 15, true},  // 4.16 > 4.15
		{4, 14, 4, 15, false}, // 4.14 < 4.15
		{3, 10, 4, 15, false}, // 3.10 < 4.15
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			result := isKernelVersionSufficient(tt.major, tt.minor, tt.minMajor, tt.minMinor)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestGetDiskSpaceMBParsing tests disk space parsing.
func TestGetDiskSpaceMBParsing(t *testing.T) {
	t.Run("parses valid df output", func(t *testing.T) {
		mockExec := exec.NewMockExecutor()
		mockExec.SetResponse("df", exec.SuccessResult("Avail\n12345M\n"))

		mockFS := NewMockFileSystem()
		mockFS.AddPath("/")

		v := NewValidator(
			WithExecutor(mockExec),
			WithFileSystem(mockFS),
		)

		space, err := v.getDiskSpaceMB(context.Background(), "/")
		require.NoError(t, err)
		assert.Equal(t, int64(12345), space)
	})

	t.Run("handles invalid df output", func(t *testing.T) {
		mockExec := exec.NewMockExecutor()
		mockExec.SetResponse("df", exec.SuccessResult("Avail\n"))

		v := NewValidator(WithExecutor(mockExec))

		_, err := v.getDiskSpaceMB(context.Background(), "/")
		assert.Error(t, err)
	})

	t.Run("handles non-numeric value", func(t *testing.T) {
		mockExec := exec.NewMockExecutor()
		mockExec.SetResponse("df", exec.SuccessResult("Avail\nnotanumber\n"))

		v := NewValidator(WithExecutor(mockExec))

		_, err := v.getDiskSpaceMB(context.Background(), "/")
		assert.Error(t, err)
	})
}

// TestConstants tests package constants.
func TestConstants(t *testing.T) {
	assert.Equal(t, int64(2048), DefaultDriverDiskSpaceMB)
	assert.Equal(t, int64(5120), DefaultCUDADiskSpaceMB)
	assert.Equal(t, int64(2048), DefaultMinDiskSpaceMB)
	assert.Equal(t, 4, MinKernelMajor)
	assert.Equal(t, 15, MinKernelMinor)
	assert.Contains(t, RequiredBuildTools, "gcc")
	assert.Contains(t, RequiredBuildTools, "make")
	assert.Contains(t, RequiredBuildTools, "dkms")
	assert.Contains(t, DiskSpaceCheckPaths, "/")
}

// TestValidatorInterfaceCompliance verifies ValidatorImpl implements Validator.
func TestValidatorInterfaceCompliance(t *testing.T) {
	var _ Validator = (*ValidatorImpl)(nil)
}

// TestRealFileSystem tests the RealFileSystem implementation exists.
func TestRealFileSystem(t *testing.T) {
	var _ FileSystem = RealFileSystem{}
}

// TestGetAbsolutePath tests the helper function.
func TestGetAbsolutePath(t *testing.T) {
	// Test with valid path
	result := getAbsolutePath("/usr")
	assert.Equal(t, "/usr", result)

	// Test with relative path
	result = getAbsolutePath(".")
	assert.NotEmpty(t, result)
}

// TestValidateWithFailingChecks tests error handling during validation.
func TestValidateWithFailingChecks(t *testing.T) {
	t.Run("handles kernel check failure gracefully", func(t *testing.T) {
		mockKernel := &MockKernelDetector{
			kernelInfoErr: assert.AnError,
		}

		v := NewValidator(WithKernelDetector(mockKernel))
		report, err := v.Validate(context.Background())

		require.NoError(t, err)
		assert.NotNil(t, report)
		assert.False(t, report.Passed)
	})
}

// TestDiskSpaceWithNonexistentPath tests disk space check with missing paths.
func TestDiskSpaceWithNonexistentPath(t *testing.T) {
	mockExec := exec.NewMockExecutor()
	mockFS := NewMockFileSystem()
	// Don't add any paths

	v := NewValidator(
		WithExecutor(mockExec),
		WithFileSystem(mockFS),
		WithDiskCheckPaths([]string{"/nonexistent"}),
	)

	result, err := v.ValidateDiskSpace(context.Background(), 2048)

	require.NoError(t, err)
	assert.False(t, result.Passed)
	assert.Contains(t, result.Message, "could not determine")
}

// TestKernelHeadersWithEmptyPackageName tests headers check without package name.
func TestKernelHeadersWithEmptyPackageName(t *testing.T) {
	mockKernel := &MockKernelDetector{
		headersInstalled:  false,
		headersPackage:    "",
		headersPackageErr: assert.AnError,
	}

	v := NewValidator(WithKernelDetector(mockKernel))
	result, err := v.ValidateKernelHeaders(context.Background())

	require.NoError(t, err)
	assert.False(t, result.Passed)
	assert.Contains(t, result.Remediation, "Install kernel headers")
}

// BenchmarkValidate benchmarks the full validation workflow.
func BenchmarkValidate(b *testing.B) {
	mockExec := exec.NewMockExecutor()
	mockExec.SetResponse("df", exec.SuccessResult("Avail\n5000M\n"))
	mockExec.SetResponse("which", exec.SuccessResult("/usr/bin/gcc\n"))

	mockKernel := &MockKernelDetector{
		kernelInfo: &kernel.KernelInfo{
			Version: "6.5.0",
			Release: "6.5.0",
		},
		headersInstalled:  true,
		secureBootEnabled: false,
	}

	mockNouveau := &MockNouveauDetector{
		status: &nouveau.Status{Loaded: false, BlacklistExists: true},
	}

	mockFS := NewMockFileSystem()
	mockFS.AddPath("/")

	v := NewValidator(
		WithExecutor(mockExec),
		WithKernelDetector(mockKernel),
		WithNouveauDetector(mockNouveau),
		WithFileSystem(mockFS),
		WithDiskCheckPaths([]string{"/"}),
	)

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = v.Validate(ctx)
	}
}
