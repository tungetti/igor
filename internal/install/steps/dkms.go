// Package steps provides installation step implementations for Igor.
// Each step represents a discrete phase of the NVIDIA driver installation process.
package steps

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/tungetti/igor/internal/gpu/kernel"
	"github.com/tungetti/igor/internal/install"
)

// State keys for DKMS module build configuration.
const (
	// StateDKMSBuilt indicates whether the DKMS module was successfully built by this step.
	StateDKMSBuilt = "dkms_built"
	// StateDKMSModuleName stores the name of the DKMS module that was built.
	StateDKMSModuleName = "dkms_module_name"
	// StateDKMSModuleVersion stores the version of the DKMS module that was built.
	StateDKMSModuleVersion = "dkms_module_version"
	// StateDKMSKernelVersion stores the kernel version the module was built for.
	StateDKMSKernelVersion = "dkms_kernel_version"
	// StateDKMSBuildTime stores the duration of the DKMS build process.
	StateDKMSBuildTime = "dkms_build_time"
)

// Default values for DKMS build step.
const (
	// DefaultDKMSModuleName is the default NVIDIA DKMS module name.
	DefaultDKMSModuleName = "nvidia"
	// DefaultDKMSTimeout is the default timeout for DKMS build operations (10 minutes).
	DefaultDKMSTimeout = 10 * time.Minute
)

// DKMSBuildStep builds NVIDIA kernel modules using DKMS (Dynamic Kernel Module Support).
// It checks if the module is already built for the current kernel and builds it if not.
type DKMSBuildStep struct {
	install.BaseStep
	moduleName      string          // NVIDIA module name (default: "nvidia")
	moduleVersion   string          // Specific version (optional, auto-detected if empty)
	kernelVersion   string          // Specific kernel version (optional, uses current if empty)
	skipStatusCheck bool            // Skip checking if module is already built
	kernelDetector  kernel.Detector // For getting kernel info
	timeout         time.Duration   // Build timeout (default: 10 minutes)
}

// DKMSBuildStepOption configures the DKMSBuildStep.
type DKMSBuildStepOption func(*DKMSBuildStep)

// WithModuleName sets the DKMS module name (default: "nvidia").
func WithModuleName(name string) DKMSBuildStepOption {
	return func(s *DKMSBuildStep) {
		s.moduleName = name
	}
}

// WithModuleVersion sets the specific module version to build.
// If not set, the version is auto-detected from dkms status.
func WithModuleVersion(version string) DKMSBuildStepOption {
	return func(s *DKMSBuildStep) {
		s.moduleVersion = version
	}
}

// WithKernelVersion sets the specific kernel version to build for.
// If not set, the current running kernel version is used.
func WithKernelVersion(version string) DKMSBuildStepOption {
	return func(s *DKMSBuildStep) {
		s.kernelVersion = version
	}
}

// WithSkipStatusCheck configures whether to skip checking if the module is already built.
func WithSkipStatusCheck(skip bool) DKMSBuildStepOption {
	return func(s *DKMSBuildStep) {
		s.skipStatusCheck = skip
	}
}

// WithKernelDetector sets a custom kernel detector.
// This is primarily used for testing.
func WithKernelDetector(detector kernel.Detector) DKMSBuildStepOption {
	return func(s *DKMSBuildStep) {
		s.kernelDetector = detector
	}
}

// WithDKMSTimeout sets the timeout for DKMS build operations.
func WithDKMSTimeout(timeout time.Duration) DKMSBuildStepOption {
	return func(s *DKMSBuildStep) {
		s.timeout = timeout
	}
}

// NewDKMSBuildStep creates a new DKMSBuildStep with the given options.
func NewDKMSBuildStep(opts ...DKMSBuildStepOption) *DKMSBuildStep {
	s := &DKMSBuildStep{
		BaseStep:        install.NewBaseStep("dkms_build", "Build NVIDIA kernel modules with DKMS", true),
		moduleName:      DefaultDKMSModuleName,
		skipStatusCheck: false,
		timeout:         DefaultDKMSTimeout,
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

// Execute builds the NVIDIA kernel modules using DKMS.
// It performs the following steps:
//  1. Checks for cancellation
//  2. Validates prerequisites (executor available)
//  3. Gets kernel version from kernel.Detector or context
//  4. Checks if DKMS is available
//  5. Gets NVIDIA module version from dkms status (if not specified)
//  6. Checks if module is already built for current kernel (unless skipStatusCheck)
//  7. If already built, skip with success message
//  8. In dry-run mode, logs what would be built
//  9. Runs dkms build nvidia/<version> -k <kernel_version>
//  10. Runs dkms install nvidia/<version> -k <kernel_version>
//  11. Verifies installation via dkms status
//  12. Stores state for rollback
func (s *DKMSBuildStep) Execute(ctx *install.Context) install.StepResult {
	startTime := time.Now()

	// Check for cancellation
	if ctx.IsCancelled() {
		return install.FailStep("step cancelled", context.Canceled)
	}

	ctx.LogDebug("starting DKMS module build")

	// Validate prerequisites
	if err := s.Validate(ctx); err != nil {
		return install.FailStep("validation failed", err).WithDuration(time.Since(startTime))
	}

	// Check if DKMS is available
	if !s.isDKMSAvailable(ctx) {
		ctx.Log("DKMS is not available, skipping module build")
		return install.SkipStep("DKMS is not available").WithDuration(time.Since(startTime))
	}

	// Check for cancellation
	if ctx.IsCancelled() {
		return install.FailStep("step cancelled", context.Canceled).WithDuration(time.Since(startTime))
	}

	// Get kernel version
	kernelVersion, err := s.getKernelVersion(ctx)
	if err != nil {
		ctx.LogError("failed to get kernel version", "error", err)
		return install.FailStep("failed to get kernel version", err).WithDuration(time.Since(startTime))
	}
	ctx.LogDebug("using kernel version", "kernel", kernelVersion)

	// Get module version
	moduleVersion := s.moduleVersion
	if moduleVersion == "" {
		moduleVersion, err = s.getModuleVersion(ctx)
		if err != nil {
			ctx.LogError("failed to get module version", "error", err)
			return install.FailStep("failed to get NVIDIA module version from DKMS", err).WithDuration(time.Since(startTime))
		}
	}
	if moduleVersion == "" {
		ctx.Log("no NVIDIA DKMS module found, skipping build")
		return install.SkipStep("no NVIDIA DKMS module found").WithDuration(time.Since(startTime))
	}
	ctx.LogDebug("using module version", "version", moduleVersion)

	// Check for cancellation
	if ctx.IsCancelled() {
		return install.FailStep("step cancelled", context.Canceled).WithDuration(time.Since(startTime))
	}

	// Check if module is already built (unless skipStatusCheck)
	if !s.skipStatusCheck {
		isBuilt, err := s.isModuleBuilt(ctx, moduleVersion, kernelVersion)
		if err != nil {
			ctx.LogWarn("failed to check if module is built, proceeding anyway", "error", err)
		} else if isBuilt {
			ctx.Log("NVIDIA module is already built for kernel", "version", moduleVersion, "kernel", kernelVersion)
			return install.SkipStep(fmt.Sprintf("module %s/%s is already built for kernel %s", s.moduleName, moduleVersion, kernelVersion)).
				WithDuration(time.Since(startTime))
		}
	}

	// Dry run mode
	if ctx.DryRun {
		ctx.Log("dry run: would build DKMS module", "module", s.moduleName, "version", moduleVersion, "kernel", kernelVersion)
		ctx.Log("dry run: would run: dkms build %s/%s -k %s", s.moduleName, moduleVersion, kernelVersion)
		ctx.Log("dry run: would run: dkms install %s/%s -k %s", s.moduleName, moduleVersion, kernelVersion)
		return install.CompleteStep("dry run: DKMS module would be built").WithDuration(time.Since(startTime))
	}

	// Check for cancellation before build
	if ctx.IsCancelled() {
		return install.FailStep("step cancelled", context.Canceled).WithDuration(time.Since(startTime))
	}

	// Build the module
	ctx.Log("building NVIDIA DKMS module", "module", s.moduleName, "version", moduleVersion, "kernel", kernelVersion)
	if err := s.buildModule(ctx, moduleVersion, kernelVersion); err != nil {
		ctx.LogError("failed to build DKMS module", "error", err)
		return install.FailStep("failed to build DKMS module", err).WithDuration(time.Since(startTime))
	}

	// Check for cancellation after build
	if ctx.IsCancelled() {
		// Try to clean up the built module
		ctx.LogDebug("cleaning up after cancellation")
		_ = s.removeModule(ctx, moduleVersion, kernelVersion)
		return install.FailStep("step cancelled", context.Canceled).WithDuration(time.Since(startTime))
	}

	// Install the module
	ctx.Log("installing NVIDIA DKMS module", "module", s.moduleName, "version", moduleVersion, "kernel", kernelVersion)
	if err := s.installModule(ctx, moduleVersion, kernelVersion); err != nil {
		ctx.LogError("failed to install DKMS module", "error", err)
		// Try to rollback the build
		if rollbackErr := s.removeModule(ctx, moduleVersion, kernelVersion); rollbackErr != nil {
			ctx.LogWarn("failed to rollback module build", "error", rollbackErr)
		}
		return install.FailStep("failed to install DKMS module", err).WithDuration(time.Since(startTime))
	}

	// Store state for rollback
	buildDuration := time.Since(startTime)
	ctx.SetState(StateDKMSBuilt, true)
	ctx.SetState(StateDKMSModuleName, s.moduleName)
	ctx.SetState(StateDKMSModuleVersion, moduleVersion)
	ctx.SetState(StateDKMSKernelVersion, kernelVersion)
	ctx.SetState(StateDKMSBuildTime, buildDuration)

	ctx.Log("NVIDIA DKMS module built and installed successfully",
		"module", s.moduleName,
		"version", moduleVersion,
		"kernel", kernelVersion,
		"duration", buildDuration)

	return install.CompleteStep("NVIDIA DKMS module built and installed successfully").
		WithDuration(buildDuration).
		WithCanRollback(true)
}

// Rollback removes the DKMS module that was built during execution.
// If no module was built by this step, this is a no-op.
func (s *DKMSBuildStep) Rollback(ctx *install.Context) error {
	// Check if we actually built a module
	if !ctx.GetStateBool(StateDKMSBuilt) {
		ctx.LogDebug("no DKMS module was built, nothing to rollback")
		return nil
	}

	moduleVersion := ctx.GetStateString(StateDKMSModuleVersion)
	kernelVersion := ctx.GetStateString(StateDKMSKernelVersion)

	if moduleVersion == "" {
		ctx.LogDebug("module version not found in state, nothing to rollback")
		return nil
	}

	// Validate executor
	if ctx.Executor == nil {
		return fmt.Errorf("executor not available for rollback")
	}

	ctx.Log("rolling back DKMS module build", "module", s.moduleName, "version", moduleVersion, "kernel", kernelVersion)

	// Remove the module
	if err := s.removeModule(ctx, moduleVersion, kernelVersion); err != nil {
		ctx.LogError("failed to remove DKMS module during rollback", "error", err)
		return fmt.Errorf("failed to remove DKMS module '%s/%s' for kernel '%s': %w",
			s.moduleName, moduleVersion, kernelVersion, err)
	}

	// Clear state
	ctx.DeleteState(StateDKMSBuilt)
	ctx.DeleteState(StateDKMSModuleName)
	ctx.DeleteState(StateDKMSModuleVersion)
	ctx.DeleteState(StateDKMSKernelVersion)
	ctx.DeleteState(StateDKMSBuildTime)

	ctx.LogDebug("DKMS module rollback completed")
	return nil
}

// Validate checks if the step can be executed with the given context.
// It ensures the Executor is available for running commands and validates
// that module names and versions contain only safe characters.
func (s *DKMSBuildStep) Validate(ctx *install.Context) error {
	if ctx.Executor == nil {
		return fmt.Errorf("executor is required for DKMS module build")
	}
	// Validate module name contains only safe characters
	if s.moduleName != "" && !isValidDKMSModuleName(s.moduleName) {
		return fmt.Errorf("invalid DKMS module name: %q", s.moduleName)
	}
	// Validate module version if specified
	if s.moduleVersion != "" && !isValidDKMSVersion(s.moduleVersion) {
		return fmt.Errorf("invalid module version: %q", s.moduleVersion)
	}
	// Validate kernel version if specified
	if s.kernelVersion != "" && !isValidKernelVersion(s.kernelVersion) {
		return fmt.Errorf("invalid kernel version: %q", s.kernelVersion)
	}
	return nil
}

// isValidDKMSModuleName checks if the module name contains only safe characters.
// Valid characters are alphanumeric, hyphen, and underscore.
func isValidDKMSModuleName(name string) bool {
	if name == "" {
		return false
	}
	for _, c := range name {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
			(c >= '0' && c <= '9') || c == '-' || c == '_') {
			return false
		}
	}
	return true
}

// isValidDKMSVersion checks if the version contains only safe characters.
// Valid characters are digits, dots, hyphens, and alphanumeric suffixes.
func isValidDKMSVersion(version string) bool {
	if version == "" {
		return false
	}
	for _, c := range version {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
			(c >= '0' && c <= '9') || c == '.' || c == '-' || c == '_') {
			return false
		}
	}
	return true
}

// isValidKernelVersion checks if the kernel version contains only safe characters.
// Valid characters are digits, dots, hyphens, alphanumeric, and plus signs.
func isValidKernelVersion(version string) bool {
	if version == "" {
		return false
	}
	for _, c := range version {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
			(c >= '0' && c <= '9') || c == '.' || c == '-' || c == '_' || c == '+') {
			return false
		}
	}
	return true
}

// CanRollback returns true since DKMS module build can be rolled back.
func (s *DKMSBuildStep) CanRollback() bool {
	return true
}

// isDKMSAvailable checks if the dkms command is available on the system.
func (s *DKMSBuildStep) isDKMSAvailable(ctx *install.Context) bool {
	result := ctx.Executor.Execute(ctx.Context(), "which", "dkms")
	if result.ExitCode == 0 {
		return true
	}

	// Try command -v as fallback
	result = ctx.Executor.Execute(ctx.Context(), "command", "-v", "dkms")
	return result.ExitCode == 0
}

// getKernelVersion returns the kernel version to build for.
// If a specific kernel version is set, it uses that.
// Otherwise, it uses the kernel detector or falls back to uname -r.
func (s *DKMSBuildStep) getKernelVersion(ctx *install.Context) (string, error) {
	// Use specified kernel version if set
	if s.kernelVersion != "" {
		return s.kernelVersion, nil
	}

	// Try kernel detector if available
	if s.kernelDetector != nil {
		info, err := s.kernelDetector.GetKernelInfo(ctx.Context())
		if err == nil && info.Version != "" {
			return info.Version, nil
		}
	}

	// Fallback to uname -r
	result := ctx.Executor.Execute(ctx.Context(), "uname", "-r")
	if result.ExitCode != 0 {
		errMsg := strings.TrimSpace(string(result.Stderr))
		if errMsg == "" {
			errMsg = "unknown error"
		}
		return "", fmt.Errorf("failed to get kernel version: %s", errMsg)
	}

	version := strings.TrimSpace(string(result.Stdout))
	if version == "" {
		return "", fmt.Errorf("empty kernel version returned")
	}

	return version, nil
}

// getModuleVersion extracts the NVIDIA module version from dkms status.
// It parses the output to find the nvidia module version.
func (s *DKMSBuildStep) getModuleVersion(ctx *install.Context) (string, error) {
	result := ctx.Executor.Execute(ctx.Context(), "dkms", "status", s.moduleName)
	if result.ExitCode != 0 {
		// dkms status returns non-zero if no modules found, which is not an error
		return "", nil
	}

	output := string(result.Stdout)
	if output == "" {
		return "", nil
	}

	return s.parseModuleVersion(output), nil
}

// parseModuleVersion extracts the version from dkms status output.
// Example outputs:
//   - "nvidia/550.54.14, 6.5.0-44-generic, x86_64: installed"
//   - "nvidia/550.54.14: added"
func (s *DKMSBuildStep) parseModuleVersion(output string) string {
	// Match module/version pattern
	re := regexp.MustCompile(`^` + regexp.QuoteMeta(s.moduleName) + `/([^,:\s]+)`)

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		matches := re.FindStringSubmatch(line)
		if len(matches) >= 2 {
			return matches[1]
		}
	}

	return ""
}

// isModuleBuilt checks if the NVIDIA module is already built for the specified kernel.
// It parses dkms status output to check for "installed" status for the given kernel.
func (s *DKMSBuildStep) isModuleBuilt(ctx *install.Context, version, kernelVersion string) (bool, error) {
	result := ctx.Executor.Execute(ctx.Context(), "dkms", "status", s.moduleName)
	if result.ExitCode != 0 {
		return false, nil
	}

	output := string(result.Stdout)
	return s.parseIsModuleBuilt(output, version, kernelVersion), nil
}

// parseIsModuleBuilt checks if the output indicates the module is built for the kernel.
// Example: "nvidia/550.54.14, 6.5.0-44-generic, x86_64: installed"
func (s *DKMSBuildStep) parseIsModuleBuilt(output, version, kernelVersion string) bool {
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Check if line contains the version and kernel version
		expectedPrefix := fmt.Sprintf("%s/%s", s.moduleName, version)
		if !strings.HasPrefix(line, expectedPrefix) {
			continue
		}

		// Check if line contains the kernel version and is installed
		if strings.Contains(line, kernelVersion) && strings.Contains(line, ": installed") {
			return true
		}
	}

	return false
}

// buildModule runs dkms build for the specified module version and kernel.
func (s *DKMSBuildStep) buildModule(ctx *install.Context, version, kernelVersion string) error {
	moduleSpec := fmt.Sprintf("%s/%s", s.moduleName, version)

	// Build with timeout context
	buildCtx, cancel := context.WithTimeout(ctx.Context(), s.timeout)
	defer cancel()

	args := []string{"build", moduleSpec}
	if kernelVersion != "" {
		args = append(args, "-k", kernelVersion)
	}

	result := ctx.Executor.ExecuteElevated(buildCtx, "dkms", args...)

	if result.ExitCode != 0 {
		errMsg := strings.TrimSpace(string(result.Stderr))
		if errMsg == "" {
			errMsg = strings.TrimSpace(string(result.Stdout))
		}
		if errMsg == "" {
			errMsg = "unknown error"
		}
		return fmt.Errorf("dkms build failed: %s", errMsg)
	}

	return nil
}

// installModule runs dkms install for the specified module version and kernel.
func (s *DKMSBuildStep) installModule(ctx *install.Context, version, kernelVersion string) error {
	moduleSpec := fmt.Sprintf("%s/%s", s.moduleName, version)

	args := []string{"install", moduleSpec}
	if kernelVersion != "" {
		args = append(args, "-k", kernelVersion)
	}

	result := ctx.Executor.ExecuteElevated(ctx.Context(), "dkms", args...)

	if result.ExitCode != 0 {
		errMsg := strings.TrimSpace(string(result.Stderr))
		if errMsg == "" {
			errMsg = strings.TrimSpace(string(result.Stdout))
		}
		if errMsg == "" {
			errMsg = "unknown error"
		}
		return fmt.Errorf("dkms install failed: %s", errMsg)
	}

	return nil
}

// removeModule runs dkms remove for the specified module version and kernel.
func (s *DKMSBuildStep) removeModule(ctx *install.Context, version, kernelVersion string) error {
	moduleSpec := fmt.Sprintf("%s/%s", s.moduleName, version)

	args := []string{"remove", moduleSpec}
	if kernelVersion != "" {
		args = append(args, "-k", kernelVersion)
	}
	args = append(args, "--all") // Remove all instances for this version

	result := ctx.Executor.ExecuteElevated(ctx.Context(), "dkms", args...)

	if result.ExitCode != 0 {
		errMsg := strings.TrimSpace(string(result.Stderr))
		if errMsg == "" {
			errMsg = strings.TrimSpace(string(result.Stdout))
		}
		if errMsg == "" {
			errMsg = "unknown error"
		}
		return fmt.Errorf("dkms remove failed: %s", errMsg)
	}

	return nil
}

// Ensure DKMSBuildStep implements the Step interface.
var _ install.Step = (*DKMSBuildStep)(nil)
