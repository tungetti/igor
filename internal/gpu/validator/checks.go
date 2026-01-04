// Package validator provides system requirements validation for NVIDIA driver installation.
// It checks disk space, kernel compatibility, required packages, and Secure Boot status
// before allowing installation to proceed.
package validator

import (
	"fmt"
	"time"
)

// Severity represents the severity level of a validation issue.
type Severity string

const (
	// SeverityError indicates the installation cannot proceed.
	SeverityError Severity = "error"
	// SeverityWarning indicates the issue may cause problems.
	SeverityWarning Severity = "warning"
	// SeverityInfo is for informational messages only.
	SeverityInfo Severity = "info"
)

// String returns the string representation of the severity level.
func (s Severity) String() string {
	return string(s)
}

// IsError returns true if this is an error severity.
func (s Severity) IsError() bool {
	return s == SeverityError
}

// IsWarning returns true if this is a warning severity.
func (s Severity) IsWarning() bool {
	return s == SeverityWarning
}

// IsInfo returns true if this is an info severity.
func (s Severity) IsInfo() bool {
	return s == SeverityInfo
}

// CheckName identifies specific validation checks.
type CheckName string

const (
	// CheckKernelVersion validates kernel version compatibility.
	CheckKernelVersion CheckName = "kernel_version"
	// CheckKernelHeaders validates kernel headers installation.
	CheckKernelHeaders CheckName = "kernel_headers"
	// CheckDiskSpace validates available disk space.
	CheckDiskSpace CheckName = "disk_space"
	// CheckSecureBoot validates Secure Boot configuration.
	CheckSecureBoot CheckName = "secure_boot"
	// CheckBuildTools validates build tool availability.
	CheckBuildTools CheckName = "build_tools"
	// CheckNouveauStatus validates Nouveau driver status.
	CheckNouveauStatus CheckName = "nouveau_status"
)

// String returns the string representation of the check name.
func (c CheckName) String() string {
	return string(c)
}

// CheckResult represents the result of a single validation check.
type CheckResult struct {
	// Name identifies the check that was performed.
	Name CheckName

	// Passed indicates whether the check passed.
	Passed bool

	// Message describes the result of the check.
	Message string

	// Severity indicates how critical a failure is.
	Severity Severity

	// Remediation provides instructions on how to fix a failed check.
	Remediation string

	// Details contains additional information about the check result.
	Details map[string]string
}

// NewCheckResult creates a new CheckResult with the given parameters.
func NewCheckResult(name CheckName, passed bool, message string, severity Severity) *CheckResult {
	return &CheckResult{
		Name:     name,
		Passed:   passed,
		Message:  message,
		Severity: severity,
		Details:  make(map[string]string),
	}
}

// WithRemediation adds remediation instructions to the check result.
func (c *CheckResult) WithRemediation(remediation string) *CheckResult {
	c.Remediation = remediation
	return c
}

// WithDetail adds a detail key-value pair to the check result.
func (c *CheckResult) WithDetail(key, value string) *CheckResult {
	if c.Details == nil {
		c.Details = make(map[string]string)
	}
	c.Details[key] = value
	return c
}

// String returns a human-readable representation of the check result.
func (c *CheckResult) String() string {
	status := "PASS"
	if !c.Passed {
		status = "FAIL"
	}
	return fmt.Sprintf("[%s] %s: %s", status, c.Name, c.Message)
}

// ValidationReport contains the results of all validation checks.
type ValidationReport struct {
	// Passed indicates whether all required checks passed (no errors).
	Passed bool

	// Checks contains all check results.
	Checks []CheckResult

	// Errors contains only the failed checks with error severity.
	Errors []CheckResult

	// Warnings contains only the failed checks with warning severity.
	Warnings []CheckResult

	// Infos contains informational check results.
	Infos []CheckResult

	// Timestamp is when the validation was performed.
	Timestamp time.Time

	// Duration is how long the validation took.
	Duration time.Duration
}

// NewValidationReport creates a new empty ValidationReport.
func NewValidationReport() *ValidationReport {
	return &ValidationReport{
		Passed:    true,
		Checks:    make([]CheckResult, 0),
		Errors:    make([]CheckResult, 0),
		Warnings:  make([]CheckResult, 0),
		Infos:     make([]CheckResult, 0),
		Timestamp: time.Now(),
	}
}

// AddCheck adds a check result to the report and updates the summary.
func (r *ValidationReport) AddCheck(result *CheckResult) {
	if result == nil {
		return
	}

	r.Checks = append(r.Checks, *result)

	// Categorize by severity
	switch result.Severity {
	case SeverityError:
		if !result.Passed {
			r.Errors = append(r.Errors, *result)
			r.Passed = false
		}
	case SeverityWarning:
		if !result.Passed {
			r.Warnings = append(r.Warnings, *result)
		}
	case SeverityInfo:
		r.Infos = append(r.Infos, *result)
	}
}

// HasErrors returns true if there are any error-severity failures.
func (r *ValidationReport) HasErrors() bool {
	return len(r.Errors) > 0
}

// HasWarnings returns true if there are any warning-severity issues.
func (r *ValidationReport) HasWarnings() bool {
	return len(r.Warnings) > 0
}

// ErrorCount returns the number of error-severity failures.
func (r *ValidationReport) ErrorCount() int {
	return len(r.Errors)
}

// WarningCount returns the number of warning-severity issues.
func (r *ValidationReport) WarningCount() int {
	return len(r.Warnings)
}

// TotalChecks returns the total number of checks performed.
func (r *ValidationReport) TotalChecks() int {
	return len(r.Checks)
}

// PassedChecks returns the number of checks that passed.
func (r *ValidationReport) PassedChecks() int {
	count := 0
	for _, check := range r.Checks {
		if check.Passed {
			count++
		}
	}
	return count
}

// FailedChecks returns the number of checks that failed.
func (r *ValidationReport) FailedChecks() int {
	return r.TotalChecks() - r.PassedChecks()
}

// GetCheck returns the result for a specific check by name.
// Returns nil if the check was not performed.
func (r *ValidationReport) GetCheck(name CheckName) *CheckResult {
	for i := range r.Checks {
		if r.Checks[i].Name == name {
			return &r.Checks[i]
		}
	}
	return nil
}

// Summary returns a human-readable summary of the validation.
func (r *ValidationReport) Summary() string {
	status := "PASSED"
	if !r.Passed {
		status = "FAILED"
	}
	return fmt.Sprintf("Validation %s: %d/%d checks passed, %d errors, %d warnings",
		status, r.PassedChecks(), r.TotalChecks(), r.ErrorCount(), r.WarningCount())
}

// Disk space constants for installation requirements.
const (
	// DefaultDriverDiskSpaceMB is the minimum disk space needed for driver only (2GB).
	DefaultDriverDiskSpaceMB int64 = 2048

	// DefaultCUDADiskSpaceMB is the minimum disk space needed for CUDA toolkit (5GB).
	DefaultCUDADiskSpaceMB int64 = 5120

	// DefaultMinDiskSpaceMB is the default minimum disk space requirement.
	DefaultMinDiskSpaceMB int64 = 2048
)

// Minimum kernel version requirements.
const (
	// MinKernelMajor is the minimum required kernel major version.
	MinKernelMajor = 4

	// MinKernelMinor is the minimum required kernel minor version (for kernel 4.x).
	MinKernelMinor = 15
)

// Build tool requirements.
var (
	// RequiredBuildTools is the list of build tools required for DKMS compilation.
	RequiredBuildTools = []string{"gcc", "make", "dkms"}

	// OptionalBuildTools is the list of optional but recommended build tools.
	OptionalBuildTools = []string{"pkg-config"}
)

// Common paths for disk space checks.
var (
	// DiskSpaceCheckPaths contains the paths to check for available disk space.
	DiskSpaceCheckPaths = []string{"/usr", "/var", "/"}
)
