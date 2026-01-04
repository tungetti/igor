// Package steps provides installation step implementations for Igor.
// Each step represents a discrete phase of the NVIDIA driver installation process.
package steps

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/tungetti/igor/internal/gpu/validator"
	"github.com/tungetti/igor/internal/install"
)

// ValidationCheck represents a single validation check to perform.
type ValidationCheck int

const (
	// CheckKernel validates kernel version compatibility.
	CheckKernel ValidationCheck = iota
	// CheckKernelHeaders validates kernel headers are installed.
	CheckKernelHeaders
	// CheckDiskSpace validates sufficient disk space is available.
	CheckDiskSpace
	// CheckSecureBoot validates Secure Boot configuration.
	CheckSecureBoot
	// CheckBuildTools validates required build tools are available.
	CheckBuildTools
	// CheckNouveauStatus validates Nouveau driver status.
	CheckNouveauStatus
	// CheckNVIDIAGPU validates that an NVIDIA GPU is present.
	CheckNVIDIAGPU
)

// String returns the string representation of a ValidationCheck.
func (c ValidationCheck) String() string {
	switch c {
	case CheckKernel:
		return "kernel"
	case CheckKernelHeaders:
		return "kernel_headers"
	case CheckDiskSpace:
		return "disk_space"
	case CheckSecureBoot:
		return "secure_boot"
	case CheckBuildTools:
		return "build_tools"
	case CheckNouveauStatus:
		return "nouveau_status"
	case CheckNVIDIAGPU:
		return "nvidia_gpu"
	default:
		return fmt.Sprintf("unknown(%d)", int(c))
	}
}

// ValidationStep validates system requirements before installation.
type ValidationStep struct {
	install.BaseStep
	validator      validator.Validator
	requiredDiskMB int64
	checks         []ValidationCheck
}

// ValidationStepOption configures the validation step.
type ValidationStepOption func(*ValidationStep)

// WithValidator sets a custom validator implementation.
func WithValidator(v validator.Validator) ValidationStepOption {
	return func(s *ValidationStep) {
		s.validator = v
	}
}

// WithRequiredDiskMB sets the required disk space in megabytes.
func WithRequiredDiskMB(mb int64) ValidationStepOption {
	return func(s *ValidationStep) {
		s.requiredDiskMB = mb
	}
}

// WithChecks sets the specific validation checks to perform.
func WithChecks(checks ...ValidationCheck) ValidationStepOption {
	return func(s *ValidationStep) {
		s.checks = checks
	}
}

// defaultChecks returns the default set of validation checks.
func defaultChecks() []ValidationCheck {
	return []ValidationCheck{
		CheckKernel,
		CheckKernelHeaders,
		CheckDiskSpace,
		CheckBuildTools,
		CheckNouveauStatus,
	}
}

// NewValidationStep creates a new validation step with the given options.
func NewValidationStep(opts ...ValidationStepOption) *ValidationStep {
	s := &ValidationStep{
		BaseStep:       install.NewBaseStep("validation", "Validate system requirements", false),
		requiredDiskMB: validator.DefaultMinDiskSpaceMB,
		checks:         defaultChecks(),
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

// Execute runs all validation checks and stores results in the context.
func (s *ValidationStep) Execute(ctx *install.Context) install.StepResult {
	startTime := time.Now()

	// Check for cancellation
	if ctx.IsCancelled() {
		return install.FailStep("validation cancelled", context.Canceled)
	}

	ctx.LogDebug("starting validation", "checks", len(s.checks))

	// Get or create validator
	v := s.getValidator(ctx)
	if v == nil {
		return install.FailStep("failed to create validator", fmt.Errorf("no validator available"))
	}

	// Run validation checks
	var errors []string
	var warnings []string
	needsKernelHeaders := false
	needsNouveauBlacklist := false

	for _, check := range s.checks {
		if ctx.IsCancelled() {
			return install.FailStep("validation cancelled", context.Canceled)
		}

		checkResult, err := s.runCheck(ctx.Context(), v, check, ctx)
		if err != nil {
			ctx.LogError("check failed", "check", check.String(), "error", err)
			errors = append(errors, fmt.Sprintf("%s: %v", check.String(), err))
			continue
		}

		if checkResult == nil {
			continue
		}

		// Process check result
		if !checkResult.Passed {
			switch checkResult.Severity {
			case validator.SeverityError:
				errors = append(errors, checkResult.Message)
				// Track specific needs
				if check == CheckKernelHeaders {
					needsKernelHeaders = true
				}
			case validator.SeverityWarning:
				warnings = append(warnings, checkResult.Message)
				// Track nouveau needs
				if check == CheckNouveauStatus {
					needsNouveauBlacklist = true
				}
			}
		}

		ctx.LogDebug("check completed",
			"check", check.String(),
			"passed", checkResult.Passed,
			"severity", checkResult.Severity.String(),
		)
	}

	// Store results in context state
	validationPassed := len(errors) == 0
	s.storeResults(ctx, validationPassed, warnings, errors, needsKernelHeaders, needsNouveauBlacklist)

	duration := time.Since(startTime)

	// Determine result status
	if !validationPassed {
		errMsg := fmt.Sprintf("validation failed: %s", strings.Join(errors, "; "))
		return install.FailStep(errMsg, fmt.Errorf("critical validation checks failed")).
			WithDuration(duration)
	}

	// Success, possibly with warnings
	msg := "all validation checks passed"
	if len(warnings) > 0 {
		msg = fmt.Sprintf("validation passed with %d warning(s)", len(warnings))
	}

	return install.CompleteStep(msg).WithDuration(duration)
}

// getValidator returns the configured validator or creates one from context.
func (s *ValidationStep) getValidator(ctx *install.Context) validator.Validator {
	if s.validator != nil {
		return s.validator
	}

	// Create validator from context executor
	if ctx.Executor != nil {
		return validator.NewValidator(
			validator.WithExecutor(ctx.Executor),
			validator.WithRequiredDiskSpace(s.requiredDiskMB),
		)
	}

	return nil
}

// runCheck executes a single validation check.
func (s *ValidationStep) runCheck(ctx context.Context, v validator.Validator, check ValidationCheck, installCtx *install.Context) (*validator.CheckResult, error) {
	switch check {
	case CheckKernel:
		return v.ValidateKernel(ctx)
	case CheckKernelHeaders:
		return v.ValidateKernelHeaders(ctx)
	case CheckDiskSpace:
		return v.ValidateDiskSpace(ctx, s.requiredDiskMB)
	case CheckSecureBoot:
		return v.ValidateSecureBoot(ctx)
	case CheckBuildTools:
		return v.ValidateBuildTools(ctx)
	case CheckNouveauStatus:
		return v.ValidateNouveauStatus(ctx)
	case CheckNVIDIAGPU:
		return s.checkNVIDIAGPU(installCtx)
	default:
		return nil, fmt.Errorf("unknown check: %s", check.String())
	}
}

// checkNVIDIAGPU validates that at least one NVIDIA GPU is present in the system.
func (s *ValidationStep) checkNVIDIAGPU(ctx *install.Context) (*validator.CheckResult, error) {
	if ctx.GPUInfo == nil {
		return validator.NewCheckResult(
			"nvidia_gpu",
			false,
			"no GPU information available",
			validator.SeverityError,
		).WithRemediation("Run GPU detection before validation"), nil
	}

	if !ctx.GPUInfo.HasNVIDIAGPUs() {
		return validator.NewCheckResult(
			"nvidia_gpu",
			false,
			"no NVIDIA GPU detected in the system",
			validator.SeverityError,
		).WithRemediation("Ensure an NVIDIA GPU is installed and properly connected"), nil
	}

	gpuCount := ctx.GPUInfo.GPUCount()
	gpuNames := make([]string, 0, gpuCount)
	for _, gpu := range ctx.GPUInfo.NVIDIAGPUs {
		gpuNames = append(gpuNames, gpu.Name())
	}

	return validator.NewCheckResult(
		"nvidia_gpu",
		true,
		fmt.Sprintf("found %d NVIDIA GPU(s): %s", gpuCount, strings.Join(gpuNames, ", ")),
		validator.SeverityInfo,
	).WithDetail("gpu_count", fmt.Sprintf("%d", gpuCount)), nil
}

// storeResults stores validation results in the context state.
func (s *ValidationStep) storeResults(ctx *install.Context, passed bool, warnings, errors []string, needsHeaders, needsNouveau bool) {
	ctx.SetState("validation_passed", passed)
	ctx.SetState("validation_warnings", warnings)
	ctx.SetState("validation_errors", errors)
	ctx.SetState("needs_kernel_headers", needsHeaders)
	ctx.SetState("needs_nouveau_blacklist", needsNouveau)

	// Build and store a validation report
	report := &ValidationReport{
		Passed:    passed,
		Warnings:  warnings,
		Errors:    errors,
		ChecksRun: len(s.checks),
		Timestamp: time.Now(),
	}
	ctx.SetState("validation_report", report)
}

// Validate checks if the step can be executed (always true for validation).
func (s *ValidationStep) Validate(ctx *install.Context) error {
	// Validation step can always be executed
	return nil
}

// Rollback is a no-op for validation (nothing to undo).
func (s *ValidationStep) Rollback(ctx *install.Context) error {
	// Validation has no side effects, nothing to rollback
	return nil
}

// CanRollback returns false since validation has no side effects.
func (s *ValidationStep) CanRollback() bool {
	return false
}

// ValidationReport contains summarized validation results.
type ValidationReport struct {
	Passed    bool
	Warnings  []string
	Errors    []string
	ChecksRun int
	Timestamp time.Time
}

// HasWarnings returns true if there are any warnings.
func (r *ValidationReport) HasWarnings() bool {
	return len(r.Warnings) > 0
}

// HasErrors returns true if there are any errors.
func (r *ValidationReport) HasErrors() bool {
	return len(r.Errors) > 0
}

// Summary returns a human-readable summary of the validation.
func (r *ValidationReport) Summary() string {
	status := "PASSED"
	if !r.Passed {
		status = "FAILED"
	}
	return fmt.Sprintf("Validation %s: %d checks run, %d errors, %d warnings",
		status, r.ChecksRun, len(r.Errors), len(r.Warnings))
}

// Ensure ValidationStep implements the Step interface.
var _ install.Step = (*ValidationStep)(nil)
