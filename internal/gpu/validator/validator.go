package validator

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/tungetti/igor/internal/errors"
	"github.com/tungetti/igor/internal/exec"
	"github.com/tungetti/igor/internal/gpu/kernel"
	"github.com/tungetti/igor/internal/gpu/nouveau"
)

// Validator interface defines the contract for system requirements validation.
type Validator interface {
	// Validate runs all checks and returns a comprehensive report.
	Validate(ctx context.Context) (*ValidationReport, error)

	// ValidateKernel checks kernel version compatibility.
	ValidateKernel(ctx context.Context) (*CheckResult, error)

	// ValidateDiskSpace checks available disk space against requirements.
	ValidateDiskSpace(ctx context.Context, requiredMB int64) (*CheckResult, error)

	// ValidateSecureBoot checks Secure Boot configuration.
	ValidateSecureBoot(ctx context.Context) (*CheckResult, error)

	// ValidateKernelHeaders checks if kernel headers are installed.
	ValidateKernelHeaders(ctx context.Context) (*CheckResult, error)

	// ValidateBuildTools checks for required build tools (gcc, make, dkms).
	ValidateBuildTools(ctx context.Context) (*CheckResult, error)

	// ValidateNouveauStatus checks if Nouveau driver needs to be disabled.
	ValidateNouveauStatus(ctx context.Context) (*CheckResult, error)
}

// FileSystem abstracts filesystem operations for testing.
type FileSystem interface {
	// Stat returns the FileInfo structure describing file.
	Stat(name string) (fs.FileInfo, error)
}

// RealFileSystem implements FileSystem using the actual operating system.
type RealFileSystem struct{}

// Stat returns the FileInfo structure describing file.
func (RealFileSystem) Stat(name string) (fs.FileInfo, error) {
	return os.Stat(name)
}

// ValidatorImpl is the production implementation of the Validator interface.
type ValidatorImpl struct {
	executor        exec.Executor
	kernelDetector  kernel.Detector
	nouveauDetector nouveau.Detector
	fs              FileSystem
	requiredDiskMB  int64
	minKernelMajor  int
	minKernelMinor  int
	requiredTools   []string
	diskCheckPaths  []string
}

// ValidatorOption configures the validator.
type ValidatorOption func(*ValidatorImpl)

// WithExecutor sets a custom executor.
func WithExecutor(executor exec.Executor) ValidatorOption {
	return func(v *ValidatorImpl) {
		v.executor = executor
	}
}

// WithKernelDetector sets a custom kernel detector.
func WithKernelDetector(detector kernel.Detector) ValidatorOption {
	return func(v *ValidatorImpl) {
		v.kernelDetector = detector
	}
}

// WithNouveauDetector sets a custom nouveau detector.
func WithNouveauDetector(detector nouveau.Detector) ValidatorOption {
	return func(v *ValidatorImpl) {
		v.nouveauDetector = detector
	}
}

// WithFileSystem sets a custom filesystem implementation.
func WithFileSystem(fs FileSystem) ValidatorOption {
	return func(v *ValidatorImpl) {
		v.fs = fs
	}
}

// WithRequiredDiskSpace sets the required disk space in MB.
func WithRequiredDiskSpace(mb int64) ValidatorOption {
	return func(v *ValidatorImpl) {
		v.requiredDiskMB = mb
	}
}

// WithMinKernelVersion sets the minimum kernel version requirement.
func WithMinKernelVersion(major, minor int) ValidatorOption {
	return func(v *ValidatorImpl) {
		v.minKernelMajor = major
		v.minKernelMinor = minor
	}
}

// WithRequiredTools sets the list of required build tools.
func WithRequiredTools(tools []string) ValidatorOption {
	return func(v *ValidatorImpl) {
		v.requiredTools = tools
	}
}

// WithDiskCheckPaths sets the paths to check for disk space.
func WithDiskCheckPaths(paths []string) ValidatorOption {
	return func(v *ValidatorImpl) {
		v.diskCheckPaths = paths
	}
}

// NewValidator creates a new validator with the given options.
func NewValidator(opts ...ValidatorOption) *ValidatorImpl {
	v := &ValidatorImpl{
		fs:             RealFileSystem{},
		requiredDiskMB: DefaultMinDiskSpaceMB,
		minKernelMajor: MinKernelMajor,
		minKernelMinor: MinKernelMinor,
		requiredTools:  RequiredBuildTools,
		diskCheckPaths: DiskSpaceCheckPaths,
	}
	for _, opt := range opts {
		opt(v)
	}
	return v
}

// Validate runs all validation checks and returns a comprehensive report.
func (v *ValidatorImpl) Validate(ctx context.Context) (*ValidationReport, error) {
	const op = "validator.Validate"

	// Check context cancellation
	select {
	case <-ctx.Done():
		return nil, errors.Wrap(errors.Validation, "validation cancelled", ctx.Err()).WithOp(op)
	default:
	}

	startTime := time.Now()
	report := NewValidationReport()

	// Run all checks, collecting results
	checks := []struct {
		name string
		fn   func(context.Context) (*CheckResult, error)
	}{
		{"kernel", v.validateKernelInternal},
		{"disk_space", func(ctx context.Context) (*CheckResult, error) {
			return v.ValidateDiskSpace(ctx, v.requiredDiskMB)
		}},
		{"kernel_headers", v.ValidateKernelHeaders},
		{"build_tools", v.ValidateBuildTools},
		{"secure_boot", v.ValidateSecureBoot},
		{"nouveau", v.ValidateNouveauStatus},
	}

	for _, check := range checks {
		// Check context before each check
		select {
		case <-ctx.Done():
			return nil, errors.Wrap(errors.Validation, "validation cancelled", ctx.Err()).WithOp(op)
		default:
		}

		result, err := check.fn(ctx)
		if err != nil {
			// If a check fails with an error, create a failed result
			result = NewCheckResult(
				CheckName(check.name),
				false,
				fmt.Sprintf("check failed: %v", err),
				SeverityError,
			)
		}
		if result != nil {
			report.AddCheck(result)
		}
	}

	report.Duration = time.Since(startTime)
	return report, nil
}

// validateKernelInternal wraps ValidateKernel to match the expected function signature.
func (v *ValidatorImpl) validateKernelInternal(ctx context.Context) (*CheckResult, error) {
	return v.ValidateKernel(ctx)
}

// ValidateKernel checks kernel version compatibility.
func (v *ValidatorImpl) ValidateKernel(ctx context.Context) (*CheckResult, error) {
	const op = "validator.ValidateKernel"

	// Check context cancellation
	select {
	case <-ctx.Done():
		return nil, errors.Wrap(errors.Validation, "kernel validation cancelled", ctx.Err()).WithOp(op)
	default:
	}

	if v.kernelDetector == nil {
		return NewCheckResult(
			CheckKernelVersion,
			false,
			"kernel detector not available",
			SeverityError,
		).WithRemediation("Internal error: kernel detector not configured"), nil
	}

	info, err := v.kernelDetector.GetKernelInfo(ctx)
	if err != nil {
		return NewCheckResult(
			CheckKernelVersion,
			false,
			fmt.Sprintf("failed to get kernel info: %v", err),
			SeverityError,
		), nil
	}

	// Parse kernel version
	major, minor, patch, err := parseKernelVersion(info.Release)
	if err != nil {
		return NewCheckResult(
			CheckKernelVersion,
			false,
			fmt.Sprintf("failed to parse kernel version %q: %v", info.Release, err),
			SeverityError,
		), nil
	}

	// Check minimum version
	if !isKernelVersionSufficient(major, minor, v.minKernelMajor, v.minKernelMinor) {
		return NewCheckResult(
			CheckKernelVersion,
			false,
			fmt.Sprintf("kernel version %s is below minimum required %d.%d",
				info.Version, v.minKernelMajor, v.minKernelMinor),
			SeverityError,
		).WithRemediation(fmt.Sprintf("Upgrade kernel to version %d.%d or newer",
			v.minKernelMajor, v.minKernelMinor)).
			WithDetail("current_version", info.Version).
			WithDetail("minimum_version", fmt.Sprintf("%d.%d", v.minKernelMajor, v.minKernelMinor)), nil
	}

	return NewCheckResult(
		CheckKernelVersion,
		true,
		fmt.Sprintf("kernel version %s is compatible", info.Version),
		SeverityInfo,
	).WithDetail("version", info.Version).
		WithDetail("release", info.Release).
		WithDetail("major", strconv.Itoa(major)).
		WithDetail("minor", strconv.Itoa(minor)).
		WithDetail("patch", strconv.Itoa(patch)), nil
}

// ValidateDiskSpace checks available disk space against requirements.
func (v *ValidatorImpl) ValidateDiskSpace(ctx context.Context, requiredMB int64) (*CheckResult, error) {
	const op = "validator.ValidateDiskSpace"

	// Check context cancellation
	select {
	case <-ctx.Done():
		return nil, errors.Wrap(errors.Validation, "disk space validation cancelled", ctx.Err()).WithOp(op)
	default:
	}

	if v.executor == nil {
		return NewCheckResult(
			CheckDiskSpace,
			false,
			"executor not available",
			SeverityError,
		).WithRemediation("Internal error: executor not configured"), nil
	}

	// Check disk space on each path
	var lowestAvailableMB int64 = -1
	var lowestPath string

	for _, path := range v.diskCheckPaths {
		// Check if path exists first
		if _, err := v.fs.Stat(path); err != nil {
			continue
		}

		availableMB, err := v.getDiskSpaceMB(ctx, path)
		if err != nil {
			continue
		}

		if lowestAvailableMB == -1 || availableMB < lowestAvailableMB {
			lowestAvailableMB = availableMB
			lowestPath = path
		}
	}

	if lowestAvailableMB == -1 {
		return NewCheckResult(
			CheckDiskSpace,
			false,
			"could not determine available disk space",
			SeverityError,
		).WithRemediation("Ensure disk is accessible and has read permissions"), nil
	}

	if lowestAvailableMB < requiredMB {
		return NewCheckResult(
			CheckDiskSpace,
			false,
			fmt.Sprintf("insufficient disk space: %d MB available, %d MB required on %s",
				lowestAvailableMB, requiredMB, lowestPath),
			SeverityError,
		).WithRemediation(fmt.Sprintf("Free up at least %d MB of disk space on %s",
			requiredMB-lowestAvailableMB, lowestPath)).
			WithDetail("available_mb", strconv.FormatInt(lowestAvailableMB, 10)).
			WithDetail("required_mb", strconv.FormatInt(requiredMB, 10)).
			WithDetail("path", lowestPath), nil
	}

	return NewCheckResult(
		CheckDiskSpace,
		true,
		fmt.Sprintf("sufficient disk space: %d MB available (%d MB required)",
			lowestAvailableMB, requiredMB),
		SeverityInfo,
	).WithDetail("available_mb", strconv.FormatInt(lowestAvailableMB, 10)).
		WithDetail("required_mb", strconv.FormatInt(requiredMB, 10)).
		WithDetail("path", lowestPath), nil
}

// getDiskSpaceMB returns available disk space in MB for the given path.
func (v *ValidatorImpl) getDiskSpaceMB(ctx context.Context, path string) (int64, error) {
	// Use df command to get available space
	result := v.executor.Execute(ctx, "df", "-BM", "--output=avail", path)
	if !result.Success() {
		return 0, fmt.Errorf("df command failed: %s", result.StderrString())
	}

	lines := result.StdoutLines()
	if len(lines) < 2 {
		return 0, fmt.Errorf("unexpected df output format")
	}

	// Second line contains the value (e.g., "12345M")
	valueStr := strings.TrimSpace(lines[1])
	valueStr = strings.TrimSuffix(valueStr, "M")

	availMB, err := strconv.ParseInt(valueStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse available space %q: %w", valueStr, err)
	}

	return availMB, nil
}

// ValidateSecureBoot checks Secure Boot configuration.
func (v *ValidatorImpl) ValidateSecureBoot(ctx context.Context) (*CheckResult, error) {
	const op = "validator.ValidateSecureBoot"

	// Check context cancellation
	select {
	case <-ctx.Done():
		return nil, errors.Wrap(errors.Validation, "secure boot validation cancelled", ctx.Err()).WithOp(op)
	default:
	}

	if v.kernelDetector == nil {
		return NewCheckResult(
			CheckSecureBoot,
			true,
			"kernel detector not available, skipping Secure Boot check",
			SeverityInfo,
		), nil
	}

	enabled, err := v.kernelDetector.IsSecureBootEnabled(ctx)
	if err != nil {
		// If we can't determine Secure Boot status, treat as info
		return NewCheckResult(
			CheckSecureBoot,
			true,
			"could not determine Secure Boot status",
			SeverityInfo,
		).WithDetail("error", err.Error()), nil
	}

	if enabled {
		return NewCheckResult(
			CheckSecureBoot,
			false,
			"Secure Boot is enabled - unsigned kernel modules may not load",
			SeverityWarning,
		).WithRemediation("Either disable Secure Boot in BIOS/UEFI settings, or use pre-signed NVIDIA drivers from your distribution").
			WithDetail("secure_boot", "enabled"), nil
	}

	return NewCheckResult(
		CheckSecureBoot,
		true,
		"Secure Boot is disabled or not supported",
		SeverityInfo,
	).WithDetail("secure_boot", "disabled"), nil
}

// ValidateKernelHeaders checks if kernel headers are installed.
func (v *ValidatorImpl) ValidateKernelHeaders(ctx context.Context) (*CheckResult, error) {
	const op = "validator.ValidateKernelHeaders"

	// Check context cancellation
	select {
	case <-ctx.Done():
		return nil, errors.Wrap(errors.Validation, "kernel headers validation cancelled", ctx.Err()).WithOp(op)
	default:
	}

	if v.kernelDetector == nil {
		return NewCheckResult(
			CheckKernelHeaders,
			false,
			"kernel detector not available",
			SeverityError,
		).WithRemediation("Internal error: kernel detector not configured"), nil
	}

	installed, err := v.kernelDetector.AreHeadersInstalled(ctx)
	if err != nil {
		return NewCheckResult(
			CheckKernelHeaders,
			false,
			fmt.Sprintf("failed to check kernel headers: %v", err),
			SeverityError,
		), nil
	}

	if !installed {
		// Try to get the recommended package name
		pkgName, _ := v.kernelDetector.GetHeadersPackage(ctx)
		remediation := "Install kernel headers for your current kernel"
		if pkgName != "" {
			remediation = fmt.Sprintf("Install kernel headers: sudo apt install %s (or equivalent for your distribution)", pkgName)
		}

		return NewCheckResult(
			CheckKernelHeaders,
			false,
			"kernel headers are not installed",
			SeverityError,
		).WithRemediation(remediation).
			WithDetail("headers_package", pkgName), nil
	}

	return NewCheckResult(
		CheckKernelHeaders,
		true,
		"kernel headers are installed",
		SeverityInfo,
	), nil
}

// ValidateBuildTools checks for required build tools (gcc, make, dkms).
func (v *ValidatorImpl) ValidateBuildTools(ctx context.Context) (*CheckResult, error) {
	const op = "validator.ValidateBuildTools"

	// Check context cancellation
	select {
	case <-ctx.Done():
		return nil, errors.Wrap(errors.Validation, "build tools validation cancelled", ctx.Err()).WithOp(op)
	default:
	}

	if v.executor == nil {
		return NewCheckResult(
			CheckBuildTools,
			false,
			"executor not available",
			SeverityError,
		).WithRemediation("Internal error: executor not configured"), nil
	}

	var missingTools []string
	var foundTools []string

	for _, tool := range v.requiredTools {
		if v.isToolAvailable(ctx, tool) {
			foundTools = append(foundTools, tool)
		} else {
			missingTools = append(missingTools, tool)
		}
	}

	if len(missingTools) > 0 {
		return NewCheckResult(
			CheckBuildTools,
			false,
			fmt.Sprintf("missing required build tools: %s", strings.Join(missingTools, ", ")),
			SeverityError,
		).WithRemediation(fmt.Sprintf("Install missing tools: sudo apt install %s (or equivalent for your distribution)",
			strings.Join(missingTools, " "))).
			WithDetail("missing_tools", strings.Join(missingTools, ",")).
			WithDetail("found_tools", strings.Join(foundTools, ",")), nil
	}

	return NewCheckResult(
		CheckBuildTools,
		true,
		fmt.Sprintf("all required build tools are available: %s", strings.Join(foundTools, ", ")),
		SeverityInfo,
	).WithDetail("found_tools", strings.Join(foundTools, ",")), nil
}

// isToolAvailable checks if a command-line tool is available in PATH.
func (v *ValidatorImpl) isToolAvailable(ctx context.Context, tool string) bool {
	result := v.executor.Execute(ctx, "which", tool)
	return result.Success() && len(result.StdoutLines()) > 0
}

// ValidateNouveauStatus checks if Nouveau driver needs to be disabled.
func (v *ValidatorImpl) ValidateNouveauStatus(ctx context.Context) (*CheckResult, error) {
	const op = "validator.ValidateNouveauStatus"

	// Check context cancellation
	select {
	case <-ctx.Done():
		return nil, errors.Wrap(errors.Validation, "nouveau validation cancelled", ctx.Err()).WithOp(op)
	default:
	}

	if v.nouveauDetector == nil {
		return NewCheckResult(
			CheckNouveauStatus,
			true,
			"nouveau detector not available, skipping check",
			SeverityInfo,
		), nil
	}

	status, err := v.nouveauDetector.Detect(ctx)
	if err != nil {
		return NewCheckResult(
			CheckNouveauStatus,
			false,
			fmt.Sprintf("failed to check Nouveau status: %v", err),
			SeverityWarning,
		), nil
	}

	if status.Loaded {
		remediation := "Blacklist the nouveau driver and reboot before installing NVIDIA drivers"
		if !status.BlacklistExists {
			remediation = "Create /etc/modprobe.d/blacklist-nouveau.conf with 'blacklist nouveau' and 'options nouveau modeset=0', then run 'sudo update-initramfs -u' and reboot"
		}

		return NewCheckResult(
			CheckNouveauStatus,
			false,
			"Nouveau driver is currently loaded",
			SeverityWarning,
		).WithRemediation(remediation).
			WithDetail("loaded", "true").
			WithDetail("in_use", strconv.FormatBool(status.InUse)).
			WithDetail("blacklist_exists", strconv.FormatBool(status.BlacklistExists)), nil
	}

	if !status.BlacklistExists {
		return NewCheckResult(
			CheckNouveauStatus,
			true,
			"Nouveau driver is not loaded, but blacklist configuration not found",
			SeverityInfo,
		).WithDetail("loaded", "false").
			WithDetail("blacklist_exists", "false"), nil
	}

	return NewCheckResult(
		CheckNouveauStatus,
		true,
		"Nouveau driver is not loaded and is blacklisted",
		SeverityInfo,
	).WithDetail("loaded", "false").
		WithDetail("blacklist_exists", "true").
		WithDetail("blacklist_files", strings.Join(status.BlacklistFiles, ",")), nil
}

// parseKernelVersion extracts major, minor, patch from a kernel version string.
// Examples: "6.5.0" -> (6, 5, 0), "5.15.0" -> (5, 15, 0)
func parseKernelVersion(version string) (major, minor, patch int, err error) {
	// Match major.minor.patch at the beginning
	re := regexp.MustCompile(`^(\d+)\.(\d+)\.(\d+)`)
	matches := re.FindStringSubmatch(version)
	if len(matches) < 4 {
		// Try major.minor only
		re = regexp.MustCompile(`^(\d+)\.(\d+)`)
		matches = re.FindStringSubmatch(version)
		if len(matches) < 3 {
			return 0, 0, 0, fmt.Errorf("invalid kernel version format: %s", version)
		}
		major, _ = strconv.Atoi(matches[1])
		minor, _ = strconv.Atoi(matches[2])
		return major, minor, 0, nil
	}

	major, _ = strconv.Atoi(matches[1])
	minor, _ = strconv.Atoi(matches[2])
	patch, _ = strconv.Atoi(matches[3])
	return major, minor, patch, nil
}

// isKernelVersionSufficient checks if the current kernel meets minimum requirements.
func isKernelVersionSufficient(major, minor, minMajor, minMinor int) bool {
	if major > minMajor {
		return true
	}
	if major == minMajor && minor >= minMinor {
		return true
	}
	return false
}

// Helper function to get absolute path for disk space check.
func getAbsolutePath(path string) string {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return path
	}
	return absPath
}

// Ensure ValidatorImpl implements Validator interface.
var _ Validator = (*ValidatorImpl)(nil)
