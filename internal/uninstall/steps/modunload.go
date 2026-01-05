// Package steps provides uninstallation step implementations for Igor.
// Each step represents a discrete phase of the NVIDIA driver uninstallation process.
package steps

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/tungetti/igor/internal/gpu/kernel"
	"github.com/tungetti/igor/internal/install"
)

// State keys for module unloading.
const (
	// StateModulesUnloaded indicates modules were unloaded.
	StateModulesUnloaded = "modules_unloaded"
	// StateUnloadedModules is the list of modules that were unloaded.
	StateUnloadedModules = "unloaded_modules"
	// StateModulesInUse is the list of modules that were in use and couldn't be unloaded.
	StateModulesInUse = "modules_in_use"
)

// Default NVIDIA kernel modules to unload.
// The order is the reverse of the install order (dependencies first).
var DefaultUnloadModules = []string{"nvidia-modeset", "nvidia-drm", "nvidia-uvm", "nvidia"}

// ModuleUnloadStep unloads NVIDIA kernel modules using modprobe -r.
// The modules are unloaded in order (dependencies first, base module last).
type ModuleUnloadStep struct {
	install.BaseStep
	moduleNames     []string        // Modules to unload
	skipIfNotLoaded bool            // Skip if modules not loaded
	kernelDetector  kernel.Detector // For checking if modules are loaded
	force           bool            // Force unload even if in use (dangerous)
	retryCount      int             // Number of retries for unloading
	retryDelay      time.Duration   // Delay between retries
}

// ModuleUnloadStepOption configures the ModuleUnloadStep.
type ModuleUnloadStepOption func(*ModuleUnloadStep)

// WithUnloadModuleNames sets the specific modules to unload.
func WithUnloadModuleNames(names []string) ModuleUnloadStepOption {
	return func(s *ModuleUnloadStep) {
		s.moduleNames = append([]string{}, names...)
	}
}

// WithSkipIfNotLoaded configures whether to skip if modules not loaded.
// Default is true.
func WithSkipIfNotLoaded(skip bool) ModuleUnloadStepOption {
	return func(s *ModuleUnloadStep) {
		s.skipIfNotLoaded = skip
	}
}

// WithUnloadKernelDetector sets a custom kernel detector.
func WithUnloadKernelDetector(detector kernel.Detector) ModuleUnloadStepOption {
	return func(s *ModuleUnloadStep) {
		s.kernelDetector = detector
	}
}

// WithForceUnload enables force unload (rmmod -f style, dangerous).
// Default is false.
func WithForceUnload(force bool) ModuleUnloadStepOption {
	return func(s *ModuleUnloadStep) {
		s.force = force
	}
}

// WithUnloadRetry sets retry count and delay for unload attempts.
// Default is 3 retries with 1 second delay.
func WithUnloadRetry(count int, delay time.Duration) ModuleUnloadStepOption {
	return func(s *ModuleUnloadStep) {
		s.retryCount = count
		s.retryDelay = delay
	}
}

// NewModuleUnloadStep creates a new ModuleUnloadStep with the given options.
func NewModuleUnloadStep(opts ...ModuleUnloadStepOption) *ModuleUnloadStep {
	s := &ModuleUnloadStep{
		BaseStep:        install.NewBaseStep("module_unload", "Unload NVIDIA kernel modules", true),
		moduleNames:     append([]string{}, DefaultUnloadModules...),
		skipIfNotLoaded: true,
		force:           false,
		retryCount:      3,
		retryDelay:      1 * time.Second,
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

// Execute unloads the NVIDIA kernel modules.
// It performs the following steps:
//  1. Checks for cancellation
//  2. Validates prerequisites (executor available)
//  3. Checks which modules are currently loaded
//  4. If no modules loaded and skipIfNotLoaded=true, returns skip
//  5. In dry-run mode, logs what would be unloaded
//  6. Unloads each module in order (dependencies first)
//  7. Handles modules in use with retry logic
//  8. Verifies modules are unloaded
//  9. Stores unloaded modules in state
func (s *ModuleUnloadStep) Execute(ctx *install.Context) install.StepResult {
	startTime := time.Now()

	// Check for cancellation
	if ctx.IsCancelled() {
		return install.FailStep("step cancelled", context.Canceled)
	}

	ctx.LogDebug("starting NVIDIA module unloading")

	// Validate prerequisites
	if err := s.Validate(ctx); err != nil {
		return install.FailStep("validation failed", err).WithDuration(time.Since(startTime))
	}

	// Check for empty module list
	if len(s.moduleNames) == 0 {
		ctx.LogDebug("no modules configured, skipping")
		return install.SkipStep("no modules configured to unload").
			WithDuration(time.Since(startTime))
	}

	// Check for cancellation
	if ctx.IsCancelled() {
		return install.FailStep("step cancelled", context.Canceled).WithDuration(time.Since(startTime))
	}

	// Check which modules are currently loaded
	loadedModules, err := s.getLoadedModules(ctx)
	if err != nil {
		ctx.LogWarn("failed to check loaded modules, proceeding anyway", "error", err)
		loadedModules = s.moduleNames // Assume all should be unloaded
	}

	// Filter to only modules that are actually loaded
	modulesToUnload := s.filterLoadedModules(s.moduleNames, loadedModules)

	// If no modules are loaded and skipIfNotLoaded=true, skip
	if len(modulesToUnload) == 0 && s.skipIfNotLoaded {
		ctx.Log("no NVIDIA modules are loaded, skipping")
		return install.SkipStep("no NVIDIA modules are loaded").
			WithDuration(time.Since(startTime))
	}

	// Check for cancellation
	if ctx.IsCancelled() {
		return install.FailStep("step cancelled", context.Canceled).WithDuration(time.Since(startTime))
	}

	// Dry run mode
	if ctx.DryRun {
		ctx.Log("dry run: would unload NVIDIA kernel modules")
		for _, mod := range modulesToUnload {
			ctx.Log("dry run: would run modprobe -r", "module", mod)
		}
		return install.CompleteStep("dry run: NVIDIA modules would be unloaded").
			WithDuration(time.Since(startTime))
	}

	// Check for cancellation before unloading
	if ctx.IsCancelled() {
		return install.FailStep("step cancelled", context.Canceled).WithDuration(time.Since(startTime))
	}

	// Unload each module
	ctx.Log("unloading NVIDIA kernel modules", "modules", strings.Join(modulesToUnload, ", "))
	unloadedModules := make([]string, 0, len(modulesToUnload))
	modulesInUse := make([]string, 0)

	for _, moduleName := range modulesToUnload {
		// Check for cancellation between module unloads
		if ctx.IsCancelled() {
			ctx.SetState(StateModulesUnloaded, len(unloadedModules) > 0)
			ctx.SetState(StateUnloadedModules, append([]string{}, unloadedModules...))
			return install.FailStep("step cancelled", context.Canceled).WithDuration(time.Since(startTime))
		}

		ctx.LogDebug("unloading module", "module", moduleName)

		// Try to unload with retries
		err := s.unloadModuleWithRetry(ctx, moduleName)
		if err != nil {
			// Check if module is in use
			inUse, _ := s.isModuleInUse(ctx, moduleName)
			if inUse {
				modulesInUse = append(modulesInUse, moduleName)
				ctx.LogWarn("module is in use", "module", moduleName)

				// If force is enabled, try rmmod -f
				if s.force {
					ctx.LogWarn("attempting force unload (dangerous)", "module", moduleName)
					if err := s.forceUnloadModule(ctx, moduleName); err != nil {
						ctx.LogError("force unload failed", "module", moduleName, "error", err)
						// Store state and return failure
						ctx.SetState(StateModulesUnloaded, len(unloadedModules) > 0)
						ctx.SetState(StateUnloadedModules, append([]string{}, unloadedModules...))
						ctx.SetState(StateModulesInUse, append([]string{}, modulesInUse...))
						return install.FailStep(fmt.Sprintf("failed to force unload module '%s'", moduleName), err).
							WithDuration(time.Since(startTime))
					}
					unloadedModules = append(unloadedModules, moduleName)
					continue
				}

				// Store state and return failure
				ctx.SetState(StateModulesUnloaded, len(unloadedModules) > 0)
				ctx.SetState(StateUnloadedModules, append([]string{}, unloadedModules...))
				ctx.SetState(StateModulesInUse, append([]string{}, modulesInUse...))
				return install.FailStep(
					fmt.Sprintf("module '%s' is in use and cannot be unloaded", moduleName),
					fmt.Errorf("module in use, try stopping GPU applications first"),
				).WithDuration(time.Since(startTime))
			}

			ctx.LogError("failed to unload module", "module", moduleName, "error", err)
			ctx.SetState(StateModulesUnloaded, len(unloadedModules) > 0)
			ctx.SetState(StateUnloadedModules, append([]string{}, unloadedModules...))
			return install.FailStep(fmt.Sprintf("failed to unload module '%s'", moduleName), err).
				WithDuration(time.Since(startTime))
		}
		unloadedModules = append(unloadedModules, moduleName)
	}

	// Verify modules are unloaded
	for _, moduleName := range unloadedModules {
		loaded, err := s.isModuleLoaded(ctx, moduleName)
		if err != nil {
			ctx.LogWarn("failed to verify module is unloaded", "module", moduleName, "error", err)
		} else if loaded {
			ctx.LogWarn("module reported success but verification failed", "module", moduleName)
		}
	}

	// Store state for rollback (defensive copy)
	ctx.SetState(StateModulesUnloaded, true)
	ctx.SetState(StateUnloadedModules, append([]string{}, unloadedModules...))

	ctx.Log("NVIDIA kernel modules unloaded successfully", "count", len(unloadedModules))
	return install.CompleteStep("NVIDIA kernel modules unloaded successfully").
		WithDuration(time.Since(startTime)).
		WithCanRollback(true)
}

// Rollback reloads the modules that were unloaded during execution.
// This allows the system to be returned to its previous state if needed.
func (s *ModuleUnloadStep) Rollback(ctx *install.Context) error {
	// Check if we actually unloaded modules
	if !ctx.GetStateBool(StateModulesUnloaded) {
		ctx.LogDebug("no modules were unloaded, nothing to rollback")
		return nil
	}

	// Get the list of unloaded modules
	unloadedModulesRaw, ok := ctx.GetState(StateUnloadedModules)
	if !ok {
		ctx.LogDebug("unloaded modules list not found in state, nothing to rollback")
		return nil
	}

	unloadedModules, ok := unloadedModulesRaw.([]string)
	if !ok {
		ctx.LogDebug("unloaded modules list has invalid type, nothing to rollback")
		return nil
	}

	if len(unloadedModules) == 0 {
		ctx.LogDebug("no modules in unloaded list, nothing to rollback")
		return nil
	}

	// Validate executor
	if ctx.Executor == nil {
		return fmt.Errorf("executor not available for rollback")
	}

	ctx.Log("rolling back module unloading (reloading modules)", "modules", strings.Join(unloadedModules, ", "))

	// Reload modules in reverse order (base module first, then dependencies)
	if err := s.reloadModules(ctx, unloadedModules); err != nil {
		ctx.LogError("failed to reload modules during rollback", "error", err)
		return fmt.Errorf("failed to reload modules: %w", err)
	}

	// Clear state
	ctx.DeleteState(StateModulesUnloaded)
	ctx.DeleteState(StateUnloadedModules)
	ctx.DeleteState(StateModulesInUse)

	ctx.LogDebug("module unloading rollback completed")
	return nil
}

// Validate checks if the step can be executed with the given context.
// It ensures the Executor is available and validates module names.
func (s *ModuleUnloadStep) Validate(ctx *install.Context) error {
	if ctx == nil {
		return fmt.Errorf("context is nil")
	}
	if ctx.Executor == nil {
		return fmt.Errorf("executor is required for module unloading")
	}

	// Validate module names contain only safe characters
	for _, name := range s.moduleNames {
		if name == "" {
			return fmt.Errorf("empty module name is not allowed")
		}
		if !isValidModuleName(name) {
			return fmt.Errorf("invalid module name: %q", name)
		}
	}

	return nil
}

// CanRollback returns true since we can reload modules after unloading.
func (s *ModuleUnloadStep) CanRollback() bool {
	return true
}

// isModuleLoaded checks if a kernel module is loaded.
// It uses the kernel detector if available, otherwise falls back to executor.
func (s *ModuleUnloadStep) isModuleLoaded(ctx *install.Context, name string) (bool, error) {
	// Use kernel detector if available
	if s.kernelDetector != nil {
		return s.kernelDetector.IsModuleLoaded(ctx.Context(), name)
	}

	// Fallback: use lsmod | grep
	result := ctx.Executor.Execute(ctx.Context(), "lsmod")
	if result.ExitCode != 0 {
		errMsg := strings.TrimSpace(string(result.Stderr))
		if errMsg == "" {
			errMsg = "lsmod command failed"
		}
		return false, fmt.Errorf("failed to list modules: %s", errMsg)
	}

	// Check if module name appears in output
	output := string(result.Stdout)
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) > 0 && fields[0] == name {
			return true, nil
		}
	}

	return false, nil
}

// getLoadedModules returns a list of all loaded NVIDIA-related modules.
func (s *ModuleUnloadStep) getLoadedModules(ctx *install.Context) ([]string, error) {
	// Use kernel detector if available
	if s.kernelDetector != nil {
		modules, err := s.kernelDetector.GetLoadedModules(ctx.Context())
		if err != nil {
			return nil, err
		}
		var loaded []string
		for _, mod := range modules {
			loaded = append(loaded, mod.Name)
		}
		return loaded, nil
	}

	// Fallback: use lsmod
	result := ctx.Executor.Execute(ctx.Context(), "lsmod")
	if result.ExitCode != 0 {
		errMsg := strings.TrimSpace(string(result.Stderr))
		if errMsg == "" {
			errMsg = "lsmod command failed"
		}
		return nil, fmt.Errorf("failed to list modules: %s", errMsg)
	}

	var loaded []string
	lines := strings.Split(string(result.Stdout), "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) > 0 {
			loaded = append(loaded, fields[0])
		}
	}

	return loaded, nil
}

// filterLoadedModules returns only the modules from 'toCheck' that are in 'loaded'.
func (s *ModuleUnloadStep) filterLoadedModules(toCheck, loaded []string) []string {
	loadedSet := make(map[string]bool)
	for _, name := range loaded {
		loadedSet[name] = true
	}

	var result []string
	for _, name := range toCheck {
		// Handle module names with hyphens/underscores interchangeably
		normalizedName := strings.ReplaceAll(name, "-", "_")
		if loadedSet[name] || loadedSet[normalizedName] {
			result = append(result, name)
		}
	}

	return result
}

// isModuleInUse checks if a kernel module is currently in use.
func (s *ModuleUnloadStep) isModuleInUse(ctx *install.Context, moduleName string) (bool, error) {
	// Try to read from /sys/module/<name>/refcnt
	normalizedName := strings.ReplaceAll(moduleName, "-", "_")
	result := ctx.Executor.Execute(ctx.Context(), "cat", fmt.Sprintf("/sys/module/%s/refcnt", normalizedName))
	if result.ExitCode == 0 {
		refcnt := strings.TrimSpace(string(result.Stdout))
		if refcnt != "0" && refcnt != "" {
			return true, nil
		}
		return false, nil
	}

	// Fallback: parse lsmod output for "Used by" column
	result = ctx.Executor.Execute(ctx.Context(), "lsmod")
	if result.ExitCode != 0 {
		return false, fmt.Errorf("failed to run lsmod")
	}

	lines := strings.Split(string(result.Stdout), "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) >= 3 && fields[0] == moduleName {
			// fields[2] is the "Used by" count
			usedBy := fields[2]
			if usedBy != "0" && usedBy != "" {
				return true, nil
			}
			return false, nil
		}
	}

	return false, nil
}

// getModuleHolders returns modules that depend on the given module.
func (s *ModuleUnloadStep) getModuleHolders(ctx *install.Context, moduleName string) ([]string, error) {
	// Read from /sys/module/<name>/holders
	normalizedName := strings.ReplaceAll(moduleName, "-", "_")
	result := ctx.Executor.Execute(ctx.Context(), "ls", fmt.Sprintf("/sys/module/%s/holders", normalizedName))
	if result.ExitCode != 0 {
		// No holders directory or module not loaded
		return nil, nil
	}

	output := strings.TrimSpace(string(result.Stdout))
	if output == "" {
		return nil, nil
	}

	return strings.Fields(output), nil
}

// unloadModule unloads a kernel module using modprobe -r.
func (s *ModuleUnloadStep) unloadModule(ctx *install.Context, name string) error {
	result := ctx.Executor.ExecuteElevated(ctx.Context(), "modprobe", "-r", name)

	if result.ExitCode != 0 {
		errMsg := strings.TrimSpace(string(result.Stderr))
		if errMsg == "" {
			errMsg = strings.TrimSpace(string(result.Stdout))
		}
		if errMsg == "" {
			errMsg = "unknown error"
		}
		return fmt.Errorf("modprobe -r failed: %s", errMsg)
	}

	return nil
}

// unloadModuleWithRetry attempts to unload a module with retries.
func (s *ModuleUnloadStep) unloadModuleWithRetry(ctx *install.Context, name string) error {
	var lastErr error

	for i := 0; i <= s.retryCount; i++ {
		// Check for cancellation before each retry
		if ctx.IsCancelled() {
			return context.Canceled
		}

		if i > 0 {
			ctx.LogDebug("retrying module unload", "module", name, "attempt", i+1)
			time.Sleep(s.retryDelay)
		}

		err := s.unloadModule(ctx, name)
		if err == nil {
			return nil
		}
		lastErr = err

		// Check if still in use
		inUse, _ := s.isModuleInUse(ctx, name)
		if !inUse {
			// Module is not in use but unload still failed - don't retry
			return lastErr
		}
	}

	return lastErr
}

// forceUnloadModule forcefully unloads a module using rmmod -f.
// This is dangerous and can cause system instability.
func (s *ModuleUnloadStep) forceUnloadModule(ctx *install.Context, name string) error {
	result := ctx.Executor.ExecuteElevated(ctx.Context(), "rmmod", "-f", name)

	if result.ExitCode != 0 {
		errMsg := strings.TrimSpace(string(result.Stderr))
		if errMsg == "" {
			errMsg = strings.TrimSpace(string(result.Stdout))
		}
		if errMsg == "" {
			errMsg = "unknown error"
		}
		return fmt.Errorf("rmmod -f failed: %s", errMsg)
	}

	return nil
}

// loadModule loads a kernel module using modprobe.
func (s *ModuleUnloadStep) loadModule(ctx *install.Context, name string) error {
	result := ctx.Executor.ExecuteElevated(ctx.Context(), "modprobe", name)

	if result.ExitCode != 0 {
		errMsg := strings.TrimSpace(string(result.Stderr))
		if errMsg == "" {
			errMsg = strings.TrimSpace(string(result.Stdout))
		}
		if errMsg == "" {
			errMsg = "unknown error"
		}
		return fmt.Errorf("modprobe failed: %s", errMsg)
	}

	return nil
}

// reloadModules reloads modules in reverse order (base module first).
func (s *ModuleUnloadStep) reloadModules(ctx *install.Context, modules []string) error {
	var firstErr error

	// Reload in reverse order (nvidia first, then dependent modules)
	for i := len(modules) - 1; i >= 0; i-- {
		moduleName := modules[i]
		ctx.LogDebug("reloading module", "module", moduleName)

		if err := s.loadModule(ctx, moduleName); err != nil {
			ctx.LogWarn("failed to reload module", "module", moduleName, "error", err)
			if firstErr == nil {
				firstErr = fmt.Errorf("failed to reload module '%s': %w", moduleName, err)
			}
		}
	}

	return firstErr
}

// isValidModuleName checks if the module name contains only safe characters.
// Valid characters are alphanumeric, hyphen, and underscore.
func isValidModuleName(name string) bool {
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

// Ensure ModuleUnloadStep implements the Step interface.
var _ install.Step = (*ModuleUnloadStep)(nil)
