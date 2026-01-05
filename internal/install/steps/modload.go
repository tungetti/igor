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

// State keys for module loading configuration.
const (
	// StateModulesLoaded indicates whether modules were loaded by this step.
	StateModulesLoaded = "modules_loaded"
	// StateLoadedModules stores the list of modules that were loaded.
	StateLoadedModules = "loaded_modules"
)

// Default NVIDIA kernel modules to load.
// The order is important: nvidia must be loaded first, then dependent modules.
var DefaultNvidiaModules = []string{"nvidia", "nvidia-uvm", "nvidia-drm", "nvidia-modeset"}

// ModuleLoadStep loads NVIDIA kernel modules using modprobe.
// It supports loading a configurable list of modules and can skip
// loading if modules are already loaded.
type ModuleLoadStep struct {
	install.BaseStep
	moduleNames    []string        // Modules to load (default: nvidia family)
	skipIfLoaded   bool            // Skip if nvidia module already loaded
	kernelDetector kernel.Detector // For checking if modules are loaded
	forceReload    bool            // Unload and reload even if loaded
}

// ModuleLoadStepOption configures the ModuleLoadStep.
type ModuleLoadStepOption func(*ModuleLoadStep)

// WithModuleNames sets the specific modules to load.
func WithModuleNames(names []string) ModuleLoadStepOption {
	return func(s *ModuleLoadStep) {
		s.moduleNames = append([]string{}, names...)
	}
}

// WithSkipIfLoaded configures whether to skip loading if modules are already loaded.
// Default is true.
func WithSkipIfLoaded(skip bool) ModuleLoadStepOption {
	return func(s *ModuleLoadStep) {
		s.skipIfLoaded = skip
	}
}

// WithModuleKernelDetector sets a custom kernel detector for module checking.
// This is primarily used for testing.
func WithModuleKernelDetector(detector kernel.Detector) ModuleLoadStepOption {
	return func(s *ModuleLoadStep) {
		s.kernelDetector = detector
	}
}

// WithForceReload configures whether to force reload modules even if already loaded.
// When true, modules will be unloaded first, then reloaded.
func WithForceReload(force bool) ModuleLoadStepOption {
	return func(s *ModuleLoadStep) {
		s.forceReload = force
	}
}

// NewModuleLoadStep creates a new ModuleLoadStep with the given options.
func NewModuleLoadStep(opts ...ModuleLoadStepOption) *ModuleLoadStep {
	s := &ModuleLoadStep{
		BaseStep:     install.NewBaseStep("module_load", "Load NVIDIA kernel modules", true),
		moduleNames:  append([]string{}, DefaultNvidiaModules...),
		skipIfLoaded: true,
		forceReload:  false,
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

// Execute loads the NVIDIA kernel modules.
// It performs the following steps:
//  1. Checks for cancellation
//  2. Validates prerequisites (executor available)
//  3. Checks if nvidia module is already loaded
//  4. If loaded and skipIfLoaded=true, returns SkipStep
//  5. If forceReload and loaded, unloads first
//  6. In dry-run mode, logs what would be loaded
//  7. Loads each module via modprobe
//  8. Verifies modules are loaded
//  9. Stores state for rollback
func (s *ModuleLoadStep) Execute(ctx *install.Context) install.StepResult {
	startTime := time.Now()

	// Check for cancellation
	if ctx.IsCancelled() {
		return install.FailStep("step cancelled", context.Canceled)
	}

	ctx.LogDebug("starting NVIDIA module loading")

	// Validate prerequisites
	if err := s.Validate(ctx); err != nil {
		return install.FailStep("validation failed", err).WithDuration(time.Since(startTime))
	}

	// Check for empty module list
	if len(s.moduleNames) == 0 {
		ctx.LogDebug("no modules configured, skipping")
		return install.SkipStep("no modules configured to load").
			WithDuration(time.Since(startTime))
	}

	// Check for cancellation
	if ctx.IsCancelled() {
		return install.FailStep("step cancelled", context.Canceled).WithDuration(time.Since(startTime))
	}

	// Check if nvidia module is already loaded
	nvidiaLoaded, err := s.isModuleLoaded(ctx, "nvidia")
	if err != nil {
		ctx.LogWarn("failed to check if nvidia module is loaded, proceeding anyway", "error", err)
		nvidiaLoaded = false
	}

	// If loaded and skipIfLoaded, skip the step
	if nvidiaLoaded && s.skipIfLoaded && !s.forceReload {
		ctx.Log("NVIDIA module is already loaded, skipping")
		return install.SkipStep("NVIDIA module is already loaded").
			WithDuration(time.Since(startTime))
	}

	// Check for cancellation
	if ctx.IsCancelled() {
		return install.FailStep("step cancelled", context.Canceled).WithDuration(time.Since(startTime))
	}

	// If force reload and modules are loaded, unload them first
	if s.forceReload && nvidiaLoaded {
		ctx.Log("force reload requested, unloading existing modules first")

		// Dry run mode for unload
		if ctx.DryRun {
			ctx.Log("dry run: would unload modules in reverse order")
			for i := len(s.moduleNames) - 1; i >= 0; i-- {
				ctx.Log("dry run: would run modprobe -r", "module", s.moduleNames[i])
			}
		} else {
			// Unload in reverse order
			if err := s.unloadModules(ctx, s.moduleNames); err != nil {
				ctx.LogError("failed to unload modules for reload", "error", err)
				return install.FailStep("failed to unload modules for reload", err).
					WithDuration(time.Since(startTime))
			}
		}
	}

	// Dry run mode
	if ctx.DryRun {
		ctx.Log("dry run: would load NVIDIA kernel modules")
		for _, mod := range s.moduleNames {
			ctx.Log("dry run: would run modprobe", "module", mod)
		}
		return install.CompleteStep("dry run: NVIDIA modules would be loaded").
			WithDuration(time.Since(startTime))
	}

	// Check for cancellation before loading
	if ctx.IsCancelled() {
		return install.FailStep("step cancelled", context.Canceled).WithDuration(time.Since(startTime))
	}

	// Load each module
	ctx.Log("loading NVIDIA kernel modules", "modules", strings.Join(s.moduleNames, ", "))
	loadedModules := make([]string, 0, len(s.moduleNames))

	for _, moduleName := range s.moduleNames {
		// Check for cancellation between module loads
		if ctx.IsCancelled() {
			// Attempt to unload modules we've loaded so far
			if len(loadedModules) > 0 {
				ctx.LogDebug("cleaning up after cancellation")
				_ = s.unloadModules(ctx, loadedModules)
			}
			return install.FailStep("step cancelled", context.Canceled).WithDuration(time.Since(startTime))
		}

		ctx.LogDebug("loading module", "module", moduleName)
		if err := s.loadModule(ctx, moduleName); err != nil {
			ctx.LogError("failed to load module", "module", moduleName, "error", err)
			// Attempt to unload modules we've loaded so far
			if len(loadedModules) > 0 {
				ctx.LogDebug("rolling back loaded modules")
				if rollbackErr := s.unloadModules(ctx, loadedModules); rollbackErr != nil {
					ctx.LogWarn("failed to rollback loaded modules", "error", rollbackErr)
				}
			}
			return install.FailStep(fmt.Sprintf("failed to load module '%s'", moduleName), err).
				WithDuration(time.Since(startTime))
		}
		loadedModules = append(loadedModules, moduleName)
	}

	// Verify modules are loaded
	for _, moduleName := range s.moduleNames {
		loaded, err := s.isModuleLoaded(ctx, moduleName)
		if err != nil {
			ctx.LogWarn("failed to verify module is loaded", "module", moduleName, "error", err)
		} else if !loaded {
			ctx.LogWarn("module reported success but verification failed", "module", moduleName)
		}
	}

	// Store state for rollback (defensive copy)
	ctx.SetState(StateModulesLoaded, true)
	ctx.SetState(StateLoadedModules, append([]string{}, loadedModules...))

	ctx.Log("NVIDIA kernel modules loaded successfully", "count", len(loadedModules))
	return install.CompleteStep("NVIDIA kernel modules loaded successfully").
		WithDuration(time.Since(startTime)).
		WithCanRollback(true)
}

// Rollback unloads the modules that were loaded during execution.
// If no modules were loaded by this step, this is a no-op.
func (s *ModuleLoadStep) Rollback(ctx *install.Context) error {
	// Check if we actually loaded modules
	if !ctx.GetStateBool(StateModulesLoaded) {
		ctx.LogDebug("no modules were loaded, nothing to rollback")
		return nil
	}

	// Get the list of loaded modules
	loadedModulesRaw, ok := ctx.GetState(StateLoadedModules)
	if !ok {
		ctx.LogDebug("loaded modules list not found in state, nothing to rollback")
		return nil
	}

	loadedModules, ok := loadedModulesRaw.([]string)
	if !ok {
		ctx.LogDebug("loaded modules list has invalid type, nothing to rollback")
		return nil
	}

	if len(loadedModules) == 0 {
		ctx.LogDebug("no modules in loaded list, nothing to rollback")
		return nil
	}

	// Validate executor
	if ctx.Executor == nil {
		return fmt.Errorf("executor not available for rollback")
	}

	ctx.Log("rolling back module loading", "modules", strings.Join(loadedModules, ", "))

	// Unload modules in reverse order
	if err := s.unloadModules(ctx, loadedModules); err != nil {
		ctx.LogError("failed to unload modules during rollback", "error", err)
		return fmt.Errorf("failed to unload modules: %w", err)
	}

	// Clear state
	ctx.DeleteState(StateModulesLoaded)
	ctx.DeleteState(StateLoadedModules)

	ctx.LogDebug("module loading rollback completed")
	return nil
}

// Validate checks if the step can be executed with the given context.
// It ensures the Executor is available and validates module names.
func (s *ModuleLoadStep) Validate(ctx *install.Context) error {
	if ctx.Executor == nil {
		return fmt.Errorf("executor is required for module loading")
	}

	// Validate module names contain only safe characters
	for _, name := range s.moduleNames {
		if name == "" {
			return fmt.Errorf("empty module name is not allowed")
		}
		if !isValidDKMSModuleName(name) {
			return fmt.Errorf("invalid module name: %q", name)
		}
	}

	return nil
}

// CanRollback returns true since module loading can be rolled back.
func (s *ModuleLoadStep) CanRollback() bool {
	return true
}

// isModuleLoaded checks if a kernel module is loaded.
// It uses the kernel detector if available, otherwise falls back to executor.
func (s *ModuleLoadStep) isModuleLoaded(ctx *install.Context, name string) (bool, error) {
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

// loadModule loads a kernel module using modprobe.
func (s *ModuleLoadStep) loadModule(ctx *install.Context, name string) error {
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

// unloadModule unloads a kernel module using modprobe -r.
func (s *ModuleLoadStep) unloadModule(ctx *install.Context, name string) error {
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

// unloadModules unloads modules in reverse order.
// It attempts to unload all modules and returns the first error encountered.
func (s *ModuleLoadStep) unloadModules(ctx *install.Context, modules []string) error {
	var firstErr error

	// Unload in reverse order (dependent modules first)
	for i := len(modules) - 1; i >= 0; i-- {
		moduleName := modules[i]
		ctx.LogDebug("unloading module", "module", moduleName)

		if err := s.unloadModule(ctx, moduleName); err != nil {
			ctx.LogWarn("failed to unload module", "module", moduleName, "error", err)
			if firstErr == nil {
				firstErr = fmt.Errorf("failed to unload module '%s': %w", moduleName, err)
			}
		}
	}

	return firstErr
}

// Ensure ModuleLoadStep implements the Step interface.
var _ install.Step = (*ModuleLoadStep)(nil)
