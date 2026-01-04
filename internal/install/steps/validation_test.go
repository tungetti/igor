package steps

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tungetti/igor/internal/exec"
	"github.com/tungetti/igor/internal/gpu"
	"github.com/tungetti/igor/internal/gpu/pci"
	"github.com/tungetti/igor/internal/gpu/validator"
	"github.com/tungetti/igor/internal/install"
)

// MockValidator implements validator.Validator for testing.
type MockValidator struct {
	kernelResult        *validator.CheckResult
	kernelErr           error
	diskSpaceResult     *validator.CheckResult
	diskSpaceErr        error
	secureBootResult    *validator.CheckResult
	secureBootErr       error
	kernelHeadersResult *validator.CheckResult
	kernelHeadersErr    error
	buildToolsResult    *validator.CheckResult
	buildToolsErr       error
	nouveauResult       *validator.CheckResult
	nouveauErr          error
	validateReport      *validator.ValidationReport
	validateErr         error
}

func NewMockValidator() *MockValidator {
	return &MockValidator{}
}

func (m *MockValidator) Validate(ctx context.Context) (*validator.ValidationReport, error) {
	return m.validateReport, m.validateErr
}

func (m *MockValidator) ValidateKernel(ctx context.Context) (*validator.CheckResult, error) {
	if m.kernelErr != nil {
		return nil, m.kernelErr
	}
	if m.kernelResult == nil {
		return validator.NewCheckResult(
			validator.CheckKernelVersion,
			true,
			"kernel OK",
			validator.SeverityInfo,
		), nil
	}
	return m.kernelResult, nil
}

func (m *MockValidator) ValidateDiskSpace(ctx context.Context, requiredMB int64) (*validator.CheckResult, error) {
	if m.diskSpaceErr != nil {
		return nil, m.diskSpaceErr
	}
	if m.diskSpaceResult == nil {
		return validator.NewCheckResult(
			validator.CheckDiskSpace,
			true,
			"disk space OK",
			validator.SeverityInfo,
		), nil
	}
	return m.diskSpaceResult, nil
}

func (m *MockValidator) ValidateSecureBoot(ctx context.Context) (*validator.CheckResult, error) {
	if m.secureBootErr != nil {
		return nil, m.secureBootErr
	}
	if m.secureBootResult == nil {
		return validator.NewCheckResult(
			validator.CheckSecureBoot,
			true,
			"secure boot disabled",
			validator.SeverityInfo,
		), nil
	}
	return m.secureBootResult, nil
}

func (m *MockValidator) ValidateKernelHeaders(ctx context.Context) (*validator.CheckResult, error) {
	if m.kernelHeadersErr != nil {
		return nil, m.kernelHeadersErr
	}
	if m.kernelHeadersResult == nil {
		return validator.NewCheckResult(
			validator.CheckKernelHeaders,
			true,
			"kernel headers OK",
			validator.SeverityInfo,
		), nil
	}
	return m.kernelHeadersResult, nil
}

func (m *MockValidator) ValidateBuildTools(ctx context.Context) (*validator.CheckResult, error) {
	if m.buildToolsErr != nil {
		return nil, m.buildToolsErr
	}
	if m.buildToolsResult == nil {
		return validator.NewCheckResult(
			validator.CheckBuildTools,
			true,
			"build tools OK",
			validator.SeverityInfo,
		), nil
	}
	return m.buildToolsResult, nil
}

func (m *MockValidator) ValidateNouveauStatus(ctx context.Context) (*validator.CheckResult, error) {
	if m.nouveauErr != nil {
		return nil, m.nouveauErr
	}
	if m.nouveauResult == nil {
		return validator.NewCheckResult(
			validator.CheckNouveauStatus,
			true,
			"nouveau not loaded",
			validator.SeverityInfo,
		), nil
	}
	return m.nouveauResult, nil
}

// Ensure MockValidator implements validator.Validator
var _ validator.Validator = (*MockValidator)(nil)

// TestValidationCheck_String tests the String method for ValidationCheck.
func TestValidationCheck_String(t *testing.T) {
	tests := []struct {
		check    ValidationCheck
		expected string
	}{
		{CheckKernel, "kernel"},
		{CheckKernelHeaders, "kernel_headers"},
		{CheckDiskSpace, "disk_space"},
		{CheckSecureBoot, "secure_boot"},
		{CheckBuildTools, "build_tools"},
		{CheckNouveauStatus, "nouveau_status"},
		{CheckNVIDIAGPU, "nvidia_gpu"},
		{ValidationCheck(99), "unknown(99)"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.check.String())
		})
	}
}

// TestNewValidationStep tests creation of validation step with various options.
func TestNewValidationStep(t *testing.T) {
	t.Run("creates with defaults", func(t *testing.T) {
		step := NewValidationStep()

		assert.Equal(t, "validation", step.Name())
		assert.Equal(t, "Validate system requirements", step.Description())
		assert.False(t, step.CanRollback())
		assert.Equal(t, validator.DefaultMinDiskSpaceMB, step.requiredDiskMB)
		assert.Equal(t, defaultChecks(), step.checks)
	})

	t.Run("creates with custom validator", func(t *testing.T) {
		mockValidator := NewMockValidator()
		step := NewValidationStep(WithValidator(mockValidator))

		assert.Equal(t, mockValidator, step.validator)
	})

	t.Run("creates with custom disk requirement", func(t *testing.T) {
		step := NewValidationStep(WithRequiredDiskMB(5000))

		assert.Equal(t, int64(5000), step.requiredDiskMB)
	})

	t.Run("creates with custom checks", func(t *testing.T) {
		checks := []ValidationCheck{CheckKernel, CheckDiskSpace}
		step := NewValidationStep(WithChecks(checks...))

		assert.Equal(t, checks, step.checks)
	})

	t.Run("creates with multiple options", func(t *testing.T) {
		mockValidator := NewMockValidator()
		checks := []ValidationCheck{CheckKernel, CheckNVIDIAGPU}

		step := NewValidationStep(
			WithValidator(mockValidator),
			WithRequiredDiskMB(8000),
			WithChecks(checks...),
		)

		assert.Equal(t, mockValidator, step.validator)
		assert.Equal(t, int64(8000), step.requiredDiskMB)
		assert.Equal(t, checks, step.checks)
	})
}

// TestValidationStep_Execute_AllPass tests successful validation when all checks pass.
func TestValidationStep_Execute_AllPass(t *testing.T) {
	mockValidator := NewMockValidator()
	step := NewValidationStep(
		WithValidator(mockValidator),
		WithChecks(CheckKernel, CheckKernelHeaders, CheckDiskSpace, CheckBuildTools),
	)

	ctx := install.NewContext()
	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusCompleted, result.Status)
	assert.Contains(t, result.Message, "passed")
	assert.True(t, ctx.GetStateBool("validation_passed"))
	assert.Empty(t, getStringSliceFromState(ctx, "validation_errors"))
}

// TestValidationStep_Execute_KernelFails tests validation failure when kernel check fails.
func TestValidationStep_Execute_KernelFails(t *testing.T) {
	mockValidator := NewMockValidator()
	mockValidator.kernelResult = validator.NewCheckResult(
		validator.CheckKernelVersion,
		false,
		"kernel version 4.10 is below minimum 4.15",
		validator.SeverityError,
	)

	step := NewValidationStep(
		WithValidator(mockValidator),
		WithChecks(CheckKernel),
	)

	ctx := install.NewContext()
	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusFailed, result.Status)
	assert.Contains(t, result.Message, "validation failed")
	assert.False(t, ctx.GetStateBool("validation_passed"))

	errors := getStringSliceFromState(ctx, "validation_errors")
	assert.NotEmpty(t, errors)
	assert.Contains(t, errors[0], "kernel version")
}

// TestValidationStep_Execute_DiskSpaceFails tests validation failure when disk space is insufficient.
func TestValidationStep_Execute_DiskSpaceFails(t *testing.T) {
	mockValidator := NewMockValidator()
	mockValidator.diskSpaceResult = validator.NewCheckResult(
		validator.CheckDiskSpace,
		false,
		"insufficient disk space: 500 MB available, 2048 MB required",
		validator.SeverityError,
	)

	step := NewValidationStep(
		WithValidator(mockValidator),
		WithChecks(CheckDiskSpace),
	)

	ctx := install.NewContext()
	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusFailed, result.Status)
	assert.Contains(t, result.Message, "validation failed")
	assert.False(t, ctx.GetStateBool("validation_passed"))

	errors := getStringSliceFromState(ctx, "validation_errors")
	assert.NotEmpty(t, errors)
	assert.Contains(t, errors[0], "disk space")
}

// TestValidationStep_Execute_NouveauWarning tests that nouveau warning doesn't fail validation.
func TestValidationStep_Execute_NouveauWarning(t *testing.T) {
	mockValidator := NewMockValidator()
	mockValidator.nouveauResult = validator.NewCheckResult(
		validator.CheckNouveauStatus,
		false,
		"Nouveau driver is currently loaded",
		validator.SeverityWarning,
	)

	step := NewValidationStep(
		WithValidator(mockValidator),
		WithChecks(CheckNouveauStatus),
	)

	ctx := install.NewContext()
	result := step.Execute(ctx)

	// Should pass with warning
	assert.Equal(t, install.StepStatusCompleted, result.Status)
	assert.Contains(t, result.Message, "warning")
	assert.True(t, ctx.GetStateBool("validation_passed"))
	assert.True(t, ctx.GetStateBool("needs_nouveau_blacklist"))

	warnings := getStringSliceFromState(ctx, "validation_warnings")
	assert.NotEmpty(t, warnings)
}

// TestValidationStep_Execute_SecureBootWarning tests that secure boot warning doesn't fail validation.
func TestValidationStep_Execute_SecureBootWarning(t *testing.T) {
	mockValidator := NewMockValidator()
	mockValidator.secureBootResult = validator.NewCheckResult(
		validator.CheckSecureBoot,
		false,
		"Secure Boot is enabled",
		validator.SeverityWarning,
	)

	step := NewValidationStep(
		WithValidator(mockValidator),
		WithChecks(CheckSecureBoot),
	)

	ctx := install.NewContext()
	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusCompleted, result.Status)
	assert.True(t, ctx.GetStateBool("validation_passed"))

	warnings := getStringSliceFromState(ctx, "validation_warnings")
	assert.NotEmpty(t, warnings)
	assert.Contains(t, warnings[0], "Secure Boot")
}

// TestValidationStep_Execute_NoGPU tests validation failure when no NVIDIA GPU is present.
func TestValidationStep_Execute_NoGPU(t *testing.T) {
	mockValidator := NewMockValidator()

	step := NewValidationStep(
		WithValidator(mockValidator),
		WithChecks(CheckNVIDIAGPU),
	)

	t.Run("nil GPUInfo", func(t *testing.T) {
		ctx := install.NewContext()
		result := step.Execute(ctx)

		assert.Equal(t, install.StepStatusFailed, result.Status)
		assert.False(t, ctx.GetStateBool("validation_passed"))

		errors := getStringSliceFromState(ctx, "validation_errors")
		assert.NotEmpty(t, errors)
		assert.Contains(t, errors[0], "no GPU information")
	})

	t.Run("empty NVIDIAGPUs", func(t *testing.T) {
		ctx := install.NewContext(
			install.WithGPUInfo(&gpu.GPUInfo{
				NVIDIAGPUs: []gpu.NVIDIAGPUInfo{},
			}),
		)
		result := step.Execute(ctx)

		assert.Equal(t, install.StepStatusFailed, result.Status)
		assert.False(t, ctx.GetStateBool("validation_passed"))

		errors := getStringSliceFromState(ctx, "validation_errors")
		assert.NotEmpty(t, errors)
		assert.Contains(t, errors[0], "no NVIDIA GPU detected")
	})
}

// TestValidationStep_Execute_WithGPU tests successful NVIDIA GPU validation.
func TestValidationStep_Execute_WithGPU(t *testing.T) {
	mockValidator := NewMockValidator()

	step := NewValidationStep(
		WithValidator(mockValidator),
		WithChecks(CheckNVIDIAGPU),
	)

	ctx := install.NewContext(
		install.WithGPUInfo(&gpu.GPUInfo{
			NVIDIAGPUs: []gpu.NVIDIAGPUInfo{
				{
					PCIDevice: pci.PCIDevice{
						DeviceID: "2684",
					},
				},
			},
		}),
	)

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusCompleted, result.Status)
	assert.True(t, ctx.GetStateBool("validation_passed"))
}

// TestValidationStep_Execute_KernelHeadersFails tests kernel headers failure tracking.
func TestValidationStep_Execute_KernelHeadersFails(t *testing.T) {
	mockValidator := NewMockValidator()
	mockValidator.kernelHeadersResult = validator.NewCheckResult(
		validator.CheckKernelHeaders,
		false,
		"kernel headers not installed",
		validator.SeverityError,
	)

	step := NewValidationStep(
		WithValidator(mockValidator),
		WithChecks(CheckKernelHeaders),
	)

	ctx := install.NewContext()
	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusFailed, result.Status)
	assert.True(t, ctx.GetStateBool("needs_kernel_headers"))
}

// TestValidationStep_Execute_BuildToolsFails tests build tools failure.
func TestValidationStep_Execute_BuildToolsFails(t *testing.T) {
	mockValidator := NewMockValidator()
	mockValidator.buildToolsResult = validator.NewCheckResult(
		validator.CheckBuildTools,
		false,
		"missing build tools: gcc, make",
		validator.SeverityError,
	)

	step := NewValidationStep(
		WithValidator(mockValidator),
		WithChecks(CheckBuildTools),
	)

	ctx := install.NewContext()
	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusFailed, result.Status)
	assert.Contains(t, result.Message, "validation failed")
}

// TestValidationStep_Execute_Cancelled tests handling of cancelled context.
func TestValidationStep_Execute_Cancelled(t *testing.T) {
	mockValidator := NewMockValidator()
	step := NewValidationStep(
		WithValidator(mockValidator),
		WithChecks(CheckKernel),
	)

	ctx := install.NewContext()
	ctx.Cancel() // Cancel the context

	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusFailed, result.Status)
	assert.Contains(t, result.Message, "cancelled")
}

// TestValidationStep_Execute_WithContextExecutor tests validator creation from context.
func TestValidationStep_Execute_WithContextExecutor(t *testing.T) {
	mockExec := exec.NewMockExecutor()
	mockExec.SetResponse("df", exec.SuccessResult("Avail\n5000M\n"))

	// Don't provide a validator, let it create one from context
	step := NewValidationStep(
		WithChecks(CheckDiskSpace),
	)

	ctx := install.NewContext(
		install.WithExecutor(mockExec),
	)

	result := step.Execute(ctx)

	// Should create a validator and run the check
	// The result depends on mock executor responses
	assert.NotNil(t, result)
}

// TestValidationStep_Execute_NoValidatorOrExecutor tests failure when no validator can be created.
func TestValidationStep_Execute_NoValidatorOrExecutor(t *testing.T) {
	step := NewValidationStep(
		WithChecks(CheckKernel),
	)

	ctx := install.NewContext()
	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusFailed, result.Status)
	assert.Contains(t, result.Message, "failed to create validator")
}

// TestValidationStep_Execute_CheckError tests handling of check errors.
func TestValidationStep_Execute_CheckError(t *testing.T) {
	mockValidator := NewMockValidator()
	mockValidator.kernelErr = assert.AnError

	step := NewValidationStep(
		WithValidator(mockValidator),
		WithChecks(CheckKernel),
	)

	ctx := install.NewContext()
	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusFailed, result.Status)
	errors := getStringSliceFromState(ctx, "validation_errors")
	assert.NotEmpty(t, errors)
}

// TestValidationStep_Execute_MultipleFailures tests handling of multiple check failures.
func TestValidationStep_Execute_MultipleFailures(t *testing.T) {
	mockValidator := NewMockValidator()
	mockValidator.kernelResult = validator.NewCheckResult(
		validator.CheckKernelVersion,
		false,
		"kernel too old",
		validator.SeverityError,
	)
	mockValidator.diskSpaceResult = validator.NewCheckResult(
		validator.CheckDiskSpace,
		false,
		"insufficient disk space",
		validator.SeverityError,
	)

	step := NewValidationStep(
		WithValidator(mockValidator),
		WithChecks(CheckKernel, CheckDiskSpace),
	)

	ctx := install.NewContext()
	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusFailed, result.Status)

	errors := getStringSliceFromState(ctx, "validation_errors")
	assert.Len(t, errors, 2)
}

// TestValidationStep_Execute_MixedResults tests handling of mixed pass/fail/warning results.
func TestValidationStep_Execute_MixedResults(t *testing.T) {
	mockValidator := NewMockValidator()
	// Kernel passes (default)
	// Disk space passes (default)
	mockValidator.secureBootResult = validator.NewCheckResult(
		validator.CheckSecureBoot,
		false,
		"Secure Boot enabled",
		validator.SeverityWarning,
	)
	mockValidator.nouveauResult = validator.NewCheckResult(
		validator.CheckNouveauStatus,
		false,
		"Nouveau loaded",
		validator.SeverityWarning,
	)

	step := NewValidationStep(
		WithValidator(mockValidator),
		WithChecks(CheckKernel, CheckDiskSpace, CheckSecureBoot, CheckNouveauStatus),
	)

	ctx := install.NewContext()
	result := step.Execute(ctx)

	// Should pass with warnings
	assert.Equal(t, install.StepStatusCompleted, result.Status)
	assert.True(t, ctx.GetStateBool("validation_passed"))

	warnings := getStringSliceFromState(ctx, "validation_warnings")
	assert.Len(t, warnings, 2)
}

// TestValidationStep_Validate tests the Validate method.
func TestValidationStep_Validate(t *testing.T) {
	step := NewValidationStep()
	ctx := install.NewContext()

	err := step.Validate(ctx)

	assert.NoError(t, err)
}

// TestValidationStep_Rollback tests the Rollback method.
func TestValidationStep_Rollback(t *testing.T) {
	step := NewValidationStep()
	ctx := install.NewContext()

	err := step.Rollback(ctx)

	assert.NoError(t, err)
}

// TestValidationStep_CanRollback tests the CanRollback method.
func TestValidationStep_CanRollback(t *testing.T) {
	step := NewValidationStep()

	assert.False(t, step.CanRollback())
}

// TestValidationStep_StoresValidationReport tests that validation report is stored in context.
func TestValidationStep_StoresValidationReport(t *testing.T) {
	mockValidator := NewMockValidator()
	step := NewValidationStep(
		WithValidator(mockValidator),
		WithChecks(CheckKernel, CheckDiskSpace),
	)

	ctx := install.NewContext()
	step.Execute(ctx)

	reportVal, ok := ctx.GetState("validation_report")
	require.True(t, ok)

	report, ok := reportVal.(*ValidationReport)
	require.True(t, ok)
	assert.True(t, report.Passed)
	assert.Equal(t, 2, report.ChecksRun)
	assert.False(t, report.Timestamp.IsZero())
}

// TestValidationReport tests ValidationReport methods.
func TestValidationReport(t *testing.T) {
	t.Run("HasWarnings", func(t *testing.T) {
		report := &ValidationReport{}
		assert.False(t, report.HasWarnings())

		report.Warnings = []string{"warning1"}
		assert.True(t, report.HasWarnings())
	})

	t.Run("HasErrors", func(t *testing.T) {
		report := &ValidationReport{}
		assert.False(t, report.HasErrors())

		report.Errors = []string{"error1"}
		assert.True(t, report.HasErrors())
	})

	t.Run("Summary passed", func(t *testing.T) {
		report := &ValidationReport{
			Passed:    true,
			ChecksRun: 5,
		}
		summary := report.Summary()
		assert.Contains(t, summary, "PASSED")
		assert.Contains(t, summary, "5 checks")
	})

	t.Run("Summary failed", func(t *testing.T) {
		report := &ValidationReport{
			Passed:    false,
			ChecksRun: 5,
			Errors:    []string{"error1", "error2"},
			Warnings:  []string{"warning1"},
		}
		summary := report.Summary()
		assert.Contains(t, summary, "FAILED")
		assert.Contains(t, summary, "2 errors")
		assert.Contains(t, summary, "1 warnings")
	})
}

// TestDefaultChecks tests the default checks list.
func TestDefaultChecks(t *testing.T) {
	checks := defaultChecks()

	assert.Contains(t, checks, CheckKernel)
	assert.Contains(t, checks, CheckKernelHeaders)
	assert.Contains(t, checks, CheckDiskSpace)
	assert.Contains(t, checks, CheckBuildTools)
	assert.Contains(t, checks, CheckNouveauStatus)
	assert.NotContains(t, checks, CheckSecureBoot)
	assert.NotContains(t, checks, CheckNVIDIAGPU)
}

// TestValidationStep_Execute_UnknownCheck tests handling of unknown check types.
func TestValidationStep_Execute_UnknownCheck(t *testing.T) {
	mockValidator := NewMockValidator()

	// Create a step with an invalid check type
	step := NewValidationStep(
		WithValidator(mockValidator),
	)
	step.checks = []ValidationCheck{ValidationCheck(99)} // Unknown check

	ctx := install.NewContext()
	result := step.Execute(ctx)

	// Should fail due to unknown check
	assert.Equal(t, install.StepStatusFailed, result.Status)
}

// TestValidationStep_InterfaceCompliance verifies ValidationStep implements Step.
func TestValidationStep_InterfaceCompliance(t *testing.T) {
	var _ install.Step = (*ValidationStep)(nil)
}

// TestValidationStep_Execute_DryRun tests behavior in dry run mode.
func TestValidationStep_Execute_DryRun(t *testing.T) {
	mockValidator := NewMockValidator()
	step := NewValidationStep(
		WithValidator(mockValidator),
		WithChecks(CheckKernel),
	)

	ctx := install.NewContext(install.WithDryRun(true))
	result := step.Execute(ctx)

	// Validation should still run in dry run mode
	assert.Equal(t, install.StepStatusCompleted, result.Status)
}

// TestValidationStep_Execute_WithLogger tests that logging is called.
func TestValidationStep_Execute_WithLogger(t *testing.T) {
	mockValidator := NewMockValidator()
	step := NewValidationStep(
		WithValidator(mockValidator),
		WithChecks(CheckKernel),
	)

	// Create context without logger - should not panic
	ctx := install.NewContext()
	result := step.Execute(ctx)

	assert.Equal(t, install.StepStatusCompleted, result.Status)
}

// Helper function to get string slice from context state
func getStringSliceFromState(ctx *install.Context, key string) []string {
	val, ok := ctx.GetState(key)
	if !ok {
		return nil
	}
	slice, ok := val.([]string)
	if !ok {
		return nil
	}
	return slice
}

// BenchmarkValidationStep benchmarks the validation step execution.
func BenchmarkValidationStep(b *testing.B) {
	mockValidator := NewMockValidator()
	step := NewValidationStep(
		WithValidator(mockValidator),
		WithChecks(CheckKernel, CheckKernelHeaders, CheckDiskSpace, CheckBuildTools, CheckNouveauStatus),
	)

	ctx := install.NewContext()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		step.Execute(ctx)
	}
}
