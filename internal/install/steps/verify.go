// Package steps provides installation step implementations for Igor.
// Each step represents a discrete phase of the NVIDIA driver installation process.
package steps

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/tungetti/igor/internal/gpu/kernel"
	"github.com/tungetti/igor/internal/install"
)

// State keys for verification results.
const (
	// StateVerificationPassed indicates whether all critical verification checks passed.
	StateVerificationPassed = "verification_passed"
	// StateDriverVersion stores the detected NVIDIA driver version.
	StateDriverVersion = "driver_version"
	// StateNvidiaSmiAvailable indicates whether nvidia-smi is available and working.
	StateNvidiaSmiAvailable = "nvidia_smi_available"
	// StateModuleLoaded indicates whether the nvidia kernel module is loaded.
	StateModuleLoaded = "module_loaded"
	// StateGPUDetected indicates whether a GPU was detected by nvidia-smi.
	StateGPUDetected = "gpu_detected"
	// StateVerificationErrors stores a list of verification failure messages.
	StateVerificationErrors = "verification_errors"
)

// VerificationCheck represents a single verification check result.
type VerificationCheck struct {
	// Name is the identifier for this check (e.g., "nvidia-smi").
	Name string
	// Description is a human-readable description of what is being checked.
	Description string
	// Passed indicates whether the check passed.
	Passed bool
	// Message contains the result message or error description.
	Message string
	// Critical indicates whether failure of this check should fail the whole step.
	Critical bool
}

// CustomCheckFunc is a function that performs a custom verification check.
type CustomCheckFunc func(ctx *install.Context) VerificationCheck

// VerificationStep verifies that the NVIDIA driver installation was successful.
// It performs various checks to ensure the driver is properly installed and working.
type VerificationStep struct {
	install.BaseStep
	checkNvidiaSmi    bool              // Check nvidia-smi availability (default: true)
	checkModuleLoaded bool              // Check nvidia kernel module loaded (default: true)
	checkGPUDetected  bool              // Check GPU detected via nvidia-smi (default: true)
	checkXorgConfig   bool              // Check X.org config exists (default: false)
	failOnWarning     bool              // Treat warnings as failures (default: false)
	kernelDetector    kernel.Detector   // For module detection
	customChecks      []CustomCheckFunc // Custom verification functions
}

// VerificationStepOption configures the VerificationStep.
type VerificationStepOption func(*VerificationStep)

// WithCheckNvidiaSmi sets whether to check nvidia-smi availability.
func WithCheckNvidiaSmi(check bool) VerificationStepOption {
	return func(s *VerificationStep) {
		s.checkNvidiaSmi = check
	}
}

// WithCheckModuleLoaded sets whether to check if the nvidia kernel module is loaded.
func WithCheckModuleLoaded(check bool) VerificationStepOption {
	return func(s *VerificationStep) {
		s.checkModuleLoaded = check
	}
}

// WithCheckGPUDetected sets whether to check if a GPU is detected by nvidia-smi.
func WithCheckGPUDetected(check bool) VerificationStepOption {
	return func(s *VerificationStep) {
		s.checkGPUDetected = check
	}
}

// WithCheckXorgConfig sets whether to check if X.org config exists.
func WithCheckXorgConfig(check bool) VerificationStepOption {
	return func(s *VerificationStep) {
		s.checkXorgConfig = check
	}
}

// WithFailOnWarning sets whether to treat warnings as failures.
func WithFailOnWarning(fail bool) VerificationStepOption {
	return func(s *VerificationStep) {
		s.failOnWarning = fail
	}
}

// WithVerificationKernelDetector sets a custom kernel detector for module checking.
// This is primarily used for testing.
func WithVerificationKernelDetector(detector kernel.Detector) VerificationStepOption {
	return func(s *VerificationStep) {
		s.kernelDetector = detector
	}
}

// WithCustomCheck adds a custom verification check function.
func WithCustomCheck(check CustomCheckFunc) VerificationStepOption {
	return func(s *VerificationStep) {
		s.customChecks = append(s.customChecks, check)
	}
}

// NewVerificationStep creates a new VerificationStep with the given options.
func NewVerificationStep(opts ...VerificationStepOption) *VerificationStep {
	s := &VerificationStep{
		BaseStep:          install.NewBaseStep("verification", "Verify NVIDIA driver installation", false),
		checkNvidiaSmi:    true,
		checkModuleLoaded: true,
		checkGPUDetected:  true,
		checkXorgConfig:   false,
		failOnWarning:     false,
		customChecks:      make([]CustomCheckFunc, 0),
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

// Execute performs the post-installation verification checks.
// It runs enabled checks and determines if the installation was successful.
func (s *VerificationStep) Execute(ctx *install.Context) install.StepResult {
	startTime := time.Now()

	// Check for cancellation
	if ctx.IsCancelled() {
		return install.FailStep("verification cancelled", context.Canceled)
	}

	ctx.LogDebug("starting post-installation verification")

	// Validate prerequisites
	if err := s.Validate(ctx); err != nil {
		return install.FailStep("validation failed", err).WithDuration(time.Since(startTime))
	}

	// Dry run mode - report what checks would be performed
	if ctx.DryRun {
		return s.executeDryRun(ctx, startTime)
	}

	// Check for cancellation
	if ctx.IsCancelled() {
		return install.FailStep("verification cancelled", context.Canceled).WithDuration(time.Since(startTime))
	}

	// Collect all check results
	var results []VerificationCheck
	var verificationErrors []string

	// Run nvidia-smi availability check
	if s.checkNvidiaSmi {
		if ctx.IsCancelled() {
			return install.FailStep("verification cancelled", context.Canceled).WithDuration(time.Since(startTime))
		}
		check := s.checkNvidiaSmiAvailable(ctx)
		results = append(results, check)
		s.logCheckResult(ctx, check)
		if !check.Passed {
			verificationErrors = append(verificationErrors, check.Message)
		}
	}

	// Run module loaded check
	if s.checkModuleLoaded {
		if ctx.IsCancelled() {
			return install.FailStep("verification cancelled", context.Canceled).WithDuration(time.Since(startTime))
		}
		check := s.checkNvidiaModuleLoaded(ctx)
		results = append(results, check)
		s.logCheckResult(ctx, check)
		if !check.Passed {
			verificationErrors = append(verificationErrors, check.Message)
		}
	}

	// Run GPU detected check
	if s.checkGPUDetected {
		if ctx.IsCancelled() {
			return install.FailStep("verification cancelled", context.Canceled).WithDuration(time.Since(startTime))
		}
		check := s.checkGPUDetectedBySmi(ctx)
		results = append(results, check)
		s.logCheckResult(ctx, check)
		if !check.Passed {
			verificationErrors = append(verificationErrors, check.Message)
		}
	}

	// Run X.org config check
	if s.checkXorgConfig {
		if ctx.IsCancelled() {
			return install.FailStep("verification cancelled", context.Canceled).WithDuration(time.Since(startTime))
		}
		check := s.checkXorgConfigExists(ctx)
		results = append(results, check)
		s.logCheckResult(ctx, check)
		if !check.Passed {
			verificationErrors = append(verificationErrors, check.Message)
		}
	}

	// Run custom checks
	for _, customCheck := range s.customChecks {
		if ctx.IsCancelled() {
			return install.FailStep("verification cancelled", context.Canceled).WithDuration(time.Since(startTime))
		}
		check := customCheck(ctx)
		results = append(results, check)
		s.logCheckResult(ctx, check)
		if !check.Passed {
			verificationErrors = append(verificationErrors, check.Message)
		}
	}

	// Determine overall pass/fail based on critical checks
	verificationPassed := s.calculateOverallResult(results)

	// Store state with results
	s.storeResults(ctx, results, verificationErrors, verificationPassed)

	duration := time.Since(startTime)

	// Build result message
	passedCount := 0
	criticalFailedCount := 0
	warningCount := 0
	for _, check := range results {
		if check.Passed {
			passedCount++
		} else if check.Critical {
			criticalFailedCount++
		} else {
			warningCount++
		}
	}

	if !verificationPassed {
		errMsg := fmt.Sprintf("verification failed: %d/%d critical checks failed",
			criticalFailedCount, len(results))
		if len(verificationErrors) > 0 {
			errMsg = fmt.Sprintf("%s (%s)", errMsg, strings.Join(verificationErrors, "; "))
		}
		return install.FailStep(errMsg, fmt.Errorf("critical verification checks failed")).
			WithDuration(duration)
	}

	// Success, possibly with warnings
	msg := fmt.Sprintf("all %d verification checks passed", len(results))
	if warningCount > 0 {
		msg = fmt.Sprintf("%d/%d verification checks passed with %d warning(s)",
			passedCount, len(results), warningCount)
	}

	ctx.Log("post-installation verification completed successfully")
	return install.CompleteStep(msg).WithDuration(duration)
}

// executeDryRun reports what checks would be performed in dry run mode.
func (s *VerificationStep) executeDryRun(ctx *install.Context, startTime time.Time) install.StepResult {
	ctx.Log("dry run: would perform the following verification checks:")

	if s.checkNvidiaSmi {
		ctx.Log("dry run: would check nvidia-smi availability")
	}
	if s.checkModuleLoaded {
		ctx.Log("dry run: would check if nvidia kernel module is loaded")
	}
	if s.checkGPUDetected {
		ctx.Log("dry run: would check if GPU is detected by nvidia-smi")
	}
	if s.checkXorgConfig {
		ctx.Log("dry run: would check if X.org config exists")
	}
	for i := range s.customChecks {
		ctx.Log("dry run: would run custom check", "index", i+1)
	}

	return install.CompleteStep("dry run: verification checks would be performed").
		WithDuration(time.Since(startTime))
}

// logCheckResult logs the result of a verification check.
func (s *VerificationStep) logCheckResult(ctx *install.Context, check VerificationCheck) {
	if check.Passed {
		ctx.LogDebug("verification check passed", "check", check.Name, "message", check.Message)
	} else if check.Critical {
		ctx.LogError("verification check failed (critical)", "check", check.Name, "message", check.Message)
	} else {
		ctx.LogWarn("verification check failed (warning)", "check", check.Name, "message", check.Message)
	}
}

// calculateOverallResult determines if verification passed based on check results.
func (s *VerificationStep) calculateOverallResult(results []VerificationCheck) bool {
	for _, check := range results {
		if !check.Passed {
			if check.Critical {
				return false
			}
			if s.failOnWarning {
				return false
			}
		}
	}
	return true
}

// storeResults stores all verification results in the context state.
func (s *VerificationStep) storeResults(ctx *install.Context, results []VerificationCheck, errors []string, passed bool) {
	ctx.SetState(StateVerificationPassed, passed)
	ctx.SetState(StateVerificationErrors, errors)

	// Store individual check results in state
	for _, check := range results {
		switch check.Name {
		case "nvidia-smi":
			ctx.SetState(StateNvidiaSmiAvailable, check.Passed)
		case "nvidia-module":
			ctx.SetState(StateModuleLoaded, check.Passed)
		case "gpu-detected":
			ctx.SetState(StateGPUDetected, check.Passed)
		}
	}

	// Extract and store driver version if we have it
	if driverVersion := ctx.GetStateString(StateDriverVersion); driverVersion != "" {
		ctx.LogDebug("detected driver version", "version", driverVersion)
	}
}

// checkNvidiaSmiAvailable checks if nvidia-smi is available and working.
func (s *VerificationStep) checkNvidiaSmiAvailable(ctx *install.Context) VerificationCheck {
	check := VerificationCheck{
		Name:        "nvidia-smi",
		Description: "Check nvidia-smi availability",
		Critical:    true,
	}

	// Run nvidia-smi and check exit code
	result := ctx.Executor.Execute(ctx.Context(), "nvidia-smi", "--query-gpu=driver_version", "--format=csv,noheader")

	if result.ExitCode != 0 {
		check.Passed = false
		errMsg := strings.TrimSpace(string(result.Stderr))
		if errMsg == "" {
			errMsg = "nvidia-smi command failed"
		}
		check.Message = fmt.Sprintf("nvidia-smi not available: %s", errMsg)
		return check
	}

	// Parse driver version from output
	output := strings.TrimSpace(string(result.Stdout))
	if output == "" {
		check.Passed = false
		check.Message = "nvidia-smi returned empty driver version"
		return check
	}

	driverVersion := s.parseDriverVersion(output)
	if driverVersion != "" {
		ctx.SetState(StateDriverVersion, driverVersion)
	}

	check.Passed = true
	check.Message = fmt.Sprintf("nvidia-smi available, driver version: %s", driverVersion)
	return check
}

// checkNvidiaModuleLoaded checks if the nvidia kernel module is loaded.
func (s *VerificationStep) checkNvidiaModuleLoaded(ctx *install.Context) VerificationCheck {
	check := VerificationCheck{
		Name:        "nvidia-module",
		Description: "Check nvidia kernel module loaded",
		Critical:    true,
	}

	var loaded bool
	var err error

	// Use kernel detector if available
	if s.kernelDetector != nil {
		loaded, err = s.kernelDetector.IsModuleLoaded(ctx.Context(), "nvidia")
		if err != nil {
			ctx.LogWarn("failed to check module via detector, falling back to lsmod", "error", err)
			loaded, err = s.checkModuleViaLsmod(ctx, "nvidia")
		}
	} else {
		loaded, err = s.checkModuleViaLsmod(ctx, "nvidia")
	}

	if err != nil {
		check.Passed = false
		check.Message = fmt.Sprintf("failed to check nvidia module: %v", err)
		return check
	}

	if !loaded {
		check.Passed = false
		check.Message = "nvidia kernel module is not loaded"
		return check
	}

	check.Passed = true
	check.Message = "nvidia kernel module is loaded"
	return check
}

// checkModuleViaLsmod checks if a module is loaded using lsmod.
func (s *VerificationStep) checkModuleViaLsmod(ctx *install.Context, moduleName string) (bool, error) {
	result := ctx.Executor.Execute(ctx.Context(), "lsmod")
	if result.ExitCode != 0 {
		errMsg := strings.TrimSpace(string(result.Stderr))
		if errMsg == "" {
			errMsg = "lsmod command failed"
		}
		return false, fmt.Errorf("failed to list modules: %s", errMsg)
	}

	output := string(result.Stdout)
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) > 0 && fields[0] == moduleName {
			return true, nil
		}
	}

	return false, nil
}

// checkGPUDetectedBySmi checks if a GPU is detected by nvidia-smi.
func (s *VerificationStep) checkGPUDetectedBySmi(ctx *install.Context) VerificationCheck {
	check := VerificationCheck{
		Name:        "gpu-detected",
		Description: "Check GPU detected by nvidia-smi",
		Critical:    true,
	}

	// Run nvidia-smi with GPU query
	result := ctx.Executor.Execute(ctx.Context(),
		"nvidia-smi", "--query-gpu=name,memory.total", "--format=csv,noheader")

	if result.ExitCode != 0 {
		check.Passed = false
		errMsg := strings.TrimSpace(string(result.Stderr))
		if errMsg == "" {
			errMsg = "nvidia-smi GPU query failed"
		}
		check.Message = fmt.Sprintf("GPU detection failed: %s", errMsg)
		return check
	}

	output := strings.TrimSpace(string(result.Stdout))
	if output == "" {
		check.Passed = false
		check.Message = "no GPU detected by nvidia-smi"
		return check
	}

	// Count GPUs (one per line)
	lines := strings.Split(output, "\n")
	gpuCount := 0
	var gpuNames []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		gpuCount++
		// Extract GPU name from "name, memory" format
		parts := strings.SplitN(line, ",", 2)
		if len(parts) > 0 {
			gpuNames = append(gpuNames, strings.TrimSpace(parts[0]))
		}
	}

	if gpuCount == 0 {
		check.Passed = false
		check.Message = "no GPU detected by nvidia-smi"
		return check
	}

	check.Passed = true
	check.Message = fmt.Sprintf("detected %d GPU(s): %s", gpuCount, strings.Join(gpuNames, ", "))
	return check
}

// checkXorgConfigExists checks if the X.org NVIDIA config file exists.
func (s *VerificationStep) checkXorgConfigExists(ctx *install.Context) VerificationCheck {
	check := VerificationCheck{
		Name:        "xorg-config",
		Description: "Check X.org NVIDIA config exists",
		Critical:    false, // X.org config is not critical (might be Wayland)
	}

	// Check if the config file exists
	configPath := DefaultXorgConfPath
	result := ctx.Executor.Execute(ctx.Context(), "test", "-f", configPath)

	if result.ExitCode != 0 {
		check.Passed = false
		check.Message = fmt.Sprintf("X.org config not found at %s", configPath)
		return check
	}

	check.Passed = true
	check.Message = fmt.Sprintf("X.org config exists at %s", configPath)
	return check
}

// parseDriverVersion extracts the driver version from nvidia-smi output.
// Input example: "550.54.14" or "550.54.14, NVIDIA GeForce RTX 3080, 10240 MiB"
func (s *VerificationStep) parseDriverVersion(output string) string {
	output = strings.TrimSpace(output)
	if output == "" {
		return ""
	}

	// Handle multi-line output (take first line)
	lines := strings.Split(output, "\n")
	if len(lines) == 0 {
		return ""
	}
	firstLine := strings.TrimSpace(lines[0])

	// Handle comma-separated values (take first value)
	parts := strings.Split(firstLine, ",")
	if len(parts) == 0 {
		return ""
	}

	return strings.TrimSpace(parts[0])
}

// Validate checks if the step can be executed with the given context.
// It ensures the Executor is available for running verification commands.
func (s *VerificationStep) Validate(ctx *install.Context) error {
	if ctx.Executor == nil {
		return fmt.Errorf("executor is required for verification")
	}
	return nil
}

// Rollback is a no-op for verification since it doesn't modify the system.
func (s *VerificationStep) Rollback(ctx *install.Context) error {
	// Verification is read-only, nothing to rollback
	return nil
}

// CanRollback returns false since verification doesn't modify the system.
func (s *VerificationStep) CanRollback() bool {
	return false
}

// Ensure VerificationStep implements the Step interface.
var _ install.Step = (*VerificationStep)(nil)
