// Package steps provides uninstallation step implementations for Igor.
// Each step represents a discrete phase of the NVIDIA driver uninstallation process.
package steps

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/tungetti/igor/internal/constants"
	"github.com/tungetti/igor/internal/gpu/kernel"
	"github.com/tungetti/igor/internal/install"
)

// State keys for Nouveau restoration.
const (
	// StateNouveauRestored indicates nouveau was restored.
	StateNouveauRestored = "nouveau_restored"
	// StateNouveauBlacklistRemoved indicates blacklist files were removed.
	StateNouveauBlacklistRemoved = "nouveau_blacklist_removed"
	// StateInitramfsRegenerated indicates initramfs was regenerated.
	StateInitramfsRegenerated = "initramfs_regenerated"
	// StateNouveauModuleLoaded indicates nouveau module was loaded.
	StateNouveauModuleLoaded = "nouveau_module_loaded"
)

// Common blacklist file paths that may be created by NVIDIA installers.
var defaultBlacklistPaths = []string{
	"/etc/modprobe.d/blacklist-nouveau.conf",
	"/etc/modprobe.d/nvidia-blacklists-nouveau.conf",
	"/etc/modprobe.d/nvidia-installer-disable-nouveau.conf",
	"/etc/modprobe.d/nouveau-blacklist.conf",
}

// blacklistContent is the content of the Nouveau blacklist configuration file
// used during rollback to re-blacklist nouveau.
const blacklistContent = `# Blacklist Nouveau driver for NVIDIA proprietary driver
# Re-created by Igor NVIDIA Installer rollback
blacklist nouveau
options nouveau modeset=0
`

// NouveauRestoreStep restores the nouveau driver after NVIDIA uninstallation.
// This ensures the system has a working graphics driver after removing NVIDIA.
type NouveauRestoreStep struct {
	install.BaseStep
	removeBlacklist     bool            // Remove nouveau blacklist files
	loadModule          bool            // Load nouveau kernel module
	regenerateInitramfs bool            // Regenerate initramfs to include nouveau
	kernelDetector      kernel.Detector // For checking module status
	blacklistPaths      []string        // Paths to check for blacklist files
}

// NouveauRestoreStepOption configures the NouveauRestoreStep.
type NouveauRestoreStepOption func(*NouveauRestoreStep)

// WithRemoveNouveauBlacklist enables/disables blacklist removal.
// Default is true.
func WithRemoveNouveauBlacklist(remove bool) NouveauRestoreStepOption {
	return func(s *NouveauRestoreStep) {
		s.removeBlacklist = remove
	}
}

// WithLoadNouveauModule enables/disables loading the nouveau module.
// Default is true.
func WithLoadNouveauModule(load bool) NouveauRestoreStepOption {
	return func(s *NouveauRestoreStep) {
		s.loadModule = load
	}
}

// WithRegenerateInitramfs enables/disables initramfs regeneration.
// Default is true (required for nouveau to load at boot).
func WithRegenerateInitramfs(regenerate bool) NouveauRestoreStepOption {
	return func(s *NouveauRestoreStep) {
		s.regenerateInitramfs = regenerate
	}
}

// WithNouveauKernelDetector sets a custom kernel detector.
func WithNouveauKernelDetector(detector kernel.Detector) NouveauRestoreStepOption {
	return func(s *NouveauRestoreStep) {
		s.kernelDetector = detector
	}
}

// WithBlacklistPaths sets custom paths to check for blacklist files.
// Default is the common paths used by NVIDIA installers.
func WithBlacklistPaths(paths []string) NouveauRestoreStepOption {
	return func(s *NouveauRestoreStep) {
		s.blacklistPaths = append([]string{}, paths...)
	}
}

// NewNouveauRestoreStep creates a new NouveauRestoreStep with the given options.
func NewNouveauRestoreStep(opts ...NouveauRestoreStepOption) *NouveauRestoreStep {
	s := &NouveauRestoreStep{
		BaseStep:            install.NewBaseStep("nouveau_restore", "Restore Nouveau driver", true),
		removeBlacklist:     true,
		loadModule:          true,
		regenerateInitramfs: true,
		blacklistPaths:      append([]string{}, defaultBlacklistPaths...),
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

// Execute restores the nouveau driver.
// It performs the following steps:
//  1. Checks for cancellation
//  2. Validates prerequisites (executor available)
//  3. Removes nouveau blacklist files (if enabled)
//  4. Regenerates initramfs (if enabled) using distribution-appropriate command
//  5. Loads nouveau module (if enabled) via modprobe
//  6. Verifies nouveau is loaded
//  7. Stores state for rollback
func (s *NouveauRestoreStep) Execute(ctx *install.Context) install.StepResult {
	startTime := time.Now()

	// Check for cancellation
	if ctx.IsCancelled() {
		return install.FailStep("step cancelled", context.Canceled)
	}

	ctx.LogDebug("starting Nouveau driver restoration")

	// Validate prerequisites
	if err := s.Validate(ctx); err != nil {
		return install.FailStep("validation failed", err).WithDuration(time.Since(startTime))
	}

	// Check for cancellation
	if ctx.IsCancelled() {
		return install.FailStep("step cancelled", context.Canceled).WithDuration(time.Since(startTime))
	}

	// Check if nouveau is already loaded
	if s.loadModule {
		loaded, err := s.isNouveauLoaded(ctx)
		if err != nil {
			ctx.LogWarn("failed to check if nouveau is loaded, proceeding anyway", "error", err)
		} else if loaded {
			ctx.Log("nouveau driver is already loaded")
			ctx.SetState(StateNouveauRestored, true)
			ctx.SetState(StateNouveauModuleLoaded, false) // We didn't load it
			return install.SkipStep("nouveau driver is already loaded").
				WithDuration(time.Since(startTime))
		}
	}

	// Dry run mode
	if ctx.DryRun {
		ctx.Log("dry run: would restore nouveau driver")
		if s.removeBlacklist {
			ctx.Log("dry run: would remove blacklist files", "paths", s.blacklistPaths)
		}
		if s.regenerateInitramfs {
			cmd, args := getInitramfsCommand(s.getDistroFamily(ctx))
			ctx.Log("dry run: would regenerate initramfs", "command", cmd, "args", args)
		}
		if s.loadModule {
			ctx.Log("dry run: would load nouveau module via modprobe")
		}
		return install.CompleteStep("dry run: nouveau would be restored").
			WithDuration(time.Since(startTime))
	}

	// Check for cancellation
	if ctx.IsCancelled() {
		return install.FailStep("step cancelled", context.Canceled).WithDuration(time.Since(startTime))
	}

	// Remove blacklist files (if enabled)
	blacklistRemoved := false
	if s.removeBlacklist {
		ctx.Log("removing nouveau blacklist files")
		removed, err := s.removeBlacklistFiles(ctx)
		if err != nil {
			ctx.LogError("failed to remove blacklist files", "error", err)
			return install.FailStep("failed to remove blacklist files", err).
				WithDuration(time.Since(startTime))
		}
		blacklistRemoved = removed
		if removed {
			ctx.LogDebug("blacklist files removed successfully")
		} else {
			ctx.LogDebug("no blacklist files found to remove")
		}
	}

	// Check for cancellation before initramfs regeneration
	if ctx.IsCancelled() {
		// Try to rollback if we removed blacklist files
		if blacklistRemoved {
			_ = s.reCreateBlacklistFile(ctx)
		}
		return install.FailStep("step cancelled", context.Canceled).
			WithDuration(time.Since(startTime))
	}

	// Regenerate initramfs (if enabled)
	initramfsRegenerated := false
	if s.regenerateInitramfs {
		ctx.Log("regenerating initramfs")
		if err := s.regenerateInitramfsCmd(ctx); err != nil {
			ctx.LogError("failed to regenerate initramfs", "error", err)
			// Try to rollback
			if blacklistRemoved {
				_ = s.reCreateBlacklistFile(ctx)
			}
			return install.FailStep("failed to regenerate initramfs", err).
				WithDuration(time.Since(startTime))
		}
		initramfsRegenerated = true
		ctx.LogDebug("initramfs regenerated successfully")
	}

	// Check for cancellation before loading module
	if ctx.IsCancelled() {
		return install.FailStep("step cancelled", context.Canceled).
			WithDuration(time.Since(startTime))
	}

	// Load nouveau module (if enabled)
	moduleLoaded := false
	if s.loadModule {
		ctx.Log("loading nouveau kernel module")
		if err := s.loadNouveauModule(ctx); err != nil {
			ctx.LogWarn("failed to load nouveau module", "error", err)
			// This is not necessarily a fatal error - nouveau will load on reboot
			// Continue and mark as successful but warn the user
			ctx.Log("nouveau will be available after reboot")
		} else {
			moduleLoaded = true
			ctx.LogDebug("nouveau module loaded successfully")
		}
	}

	// Verify nouveau is loaded (if we tried to load it)
	if s.loadModule && moduleLoaded {
		loaded, err := s.isNouveauLoaded(ctx)
		if err != nil {
			ctx.LogWarn("failed to verify nouveau is loaded", "error", err)
		} else if !loaded {
			ctx.LogWarn("nouveau module load reported success but verification failed")
		}
	}

	// Store state for rollback
	ctx.SetState(StateNouveauRestored, true)
	ctx.SetState(StateNouveauBlacklistRemoved, blacklistRemoved)
	ctx.SetState(StateInitramfsRegenerated, initramfsRegenerated)
	ctx.SetState(StateNouveauModuleLoaded, moduleLoaded)

	ctx.Log("nouveau driver restored successfully")
	return install.CompleteStep("nouveau driver restored successfully").
		WithDuration(time.Since(startTime)).
		WithCanRollback(true)
}

// Rollback re-blacklists nouveau and unloads the module.
// This is called when a later step fails and we need to undo the restoration.
func (s *NouveauRestoreStep) Rollback(ctx *install.Context) error {
	// Check if we actually restored nouveau
	if !ctx.GetStateBool(StateNouveauRestored) {
		ctx.LogDebug("nouveau was not restored, nothing to rollback")
		return nil
	}

	// Validate executor
	if ctx.Executor == nil {
		return fmt.Errorf("executor not available for rollback")
	}

	ctx.Log("rolling back nouveau restoration")

	// Unload nouveau module if we loaded it
	if ctx.GetStateBool(StateNouveauModuleLoaded) {
		ctx.Log("unloading nouveau module")
		if err := s.unloadNouveauModule(ctx); err != nil {
			ctx.LogWarn("failed to unload nouveau module", "error", err)
			// Continue with rollback even if unload fails
		}
	}

	// Re-create blacklist file if we removed it
	if ctx.GetStateBool(StateNouveauBlacklistRemoved) {
		ctx.Log("re-creating nouveau blacklist file")
		if err := s.reCreateBlacklistFile(ctx); err != nil {
			ctx.LogError("failed to re-create blacklist file", "error", err)
			return fmt.Errorf("failed to re-create blacklist file: %w", err)
		}
	}

	// Regenerate initramfs if we regenerated it
	if ctx.GetStateBool(StateInitramfsRegenerated) {
		ctx.Log("regenerating initramfs after rollback")
		if err := s.regenerateInitramfsCmd(ctx); err != nil {
			ctx.LogError("failed to regenerate initramfs during rollback", "error", err)
			return fmt.Errorf("failed to regenerate initramfs: %w", err)
		}
	}

	// Clear state
	ctx.DeleteState(StateNouveauRestored)
	ctx.DeleteState(StateNouveauBlacklistRemoved)
	ctx.DeleteState(StateInitramfsRegenerated)
	ctx.DeleteState(StateNouveauModuleLoaded)

	ctx.LogDebug("nouveau restoration rollback completed")
	return nil
}

// Validate checks if the step can be executed with the given context.
// It ensures the Executor is available.
func (s *NouveauRestoreStep) Validate(ctx *install.Context) error {
	if ctx == nil {
		return fmt.Errorf("context is nil")
	}
	if ctx.Executor == nil {
		return fmt.Errorf("executor is required for nouveau restoration")
	}
	return nil
}

// CanRollback returns true since nouveau restoration can be rolled back.
func (s *NouveauRestoreStep) CanRollback() bool {
	return true
}

// getDistroFamily returns the distribution family from the context.
func (s *NouveauRestoreStep) getDistroFamily(ctx *install.Context) constants.DistroFamily {
	if ctx.DistroInfo != nil {
		return ctx.DistroInfo.Family
	}
	return constants.FamilyUnknown
}

// getInitramfsCommand returns the distribution-appropriate initramfs command.
func getInitramfsCommand(family constants.DistroFamily) (string, []string) {
	switch family {
	case constants.FamilyDebian:
		return "update-initramfs", []string{"-u"}
	case constants.FamilyRHEL, constants.FamilySUSE:
		return "dracut", []string{"--force"}
	case constants.FamilyArch:
		return "mkinitcpio", []string{"-P"}
	default:
		// Fallback to Debian-style command
		return "update-initramfs", []string{"-u"}
	}
}

// isNouveauLoaded checks if the nouveau kernel module is loaded.
func (s *NouveauRestoreStep) isNouveauLoaded(ctx *install.Context) (bool, error) {
	// Use kernel detector if available
	if s.kernelDetector != nil {
		return s.kernelDetector.IsModuleLoaded(ctx.Context(), "nouveau")
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

	// Check if nouveau appears in output
	output := string(result.Stdout)
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) > 0 && fields[0] == "nouveau" {
			return true, nil
		}
	}

	return false, nil
}

// removeBlacklistFiles removes all nouveau blacklist files.
// Returns true if at least one file was removed.
func (s *NouveauRestoreStep) removeBlacklistFiles(ctx *install.Context) (bool, error) {
	removed := false

	for _, path := range s.blacklistPaths {
		// Check if file exists first
		checkResult := ctx.Executor.Execute(ctx.Context(), "test", "-f", path)
		if checkResult.ExitCode != 0 {
			// File doesn't exist, skip
			continue
		}

		ctx.LogDebug("removing blacklist file", "path", path)
		result := ctx.Executor.ExecuteElevated(ctx.Context(), "rm", "-f", path)
		if result.ExitCode != 0 {
			errMsg := strings.TrimSpace(string(result.Stderr))
			if errMsg == "" {
				errMsg = "unknown error"
			}
			return removed, fmt.Errorf("failed to remove blacklist file '%s': %s", path, errMsg)
		}
		removed = true
	}

	return removed, nil
}

// reCreateBlacklistFile creates a nouveau blacklist file for rollback.
func (s *NouveauRestoreStep) reCreateBlacklistFile(ctx *install.Context) error {
	// Use the first path as the default location
	path := defaultBlacklistPaths[0]

	ctx.LogDebug("re-creating blacklist file", "path", path)

	result := ctx.Executor.ExecuteWithInput(
		ctx.Context(),
		[]byte(blacklistContent),
		"tee",
		path,
	)

	if result.ExitCode != 0 {
		errMsg := strings.TrimSpace(string(result.Stderr))
		if errMsg == "" {
			errMsg = "unknown error"
		}
		return fmt.Errorf("failed to create blacklist file: %s", errMsg)
	}

	return nil
}

// regenerateInitramfsCmd regenerates the initramfs using the distribution-appropriate command.
func (s *NouveauRestoreStep) regenerateInitramfsCmd(ctx *install.Context) error {
	cmd, args := getInitramfsCommand(s.getDistroFamily(ctx))

	ctx.LogDebug("executing initramfs command", "command", cmd, "args", args)

	result := ctx.Executor.ExecuteElevated(ctx.Context(), cmd, args...)

	if result.ExitCode != 0 {
		errMsg := strings.TrimSpace(string(result.Stderr))
		if errMsg == "" {
			errMsg = strings.TrimSpace(string(result.Stdout))
		}
		if errMsg == "" {
			errMsg = "unknown error"
		}
		return fmt.Errorf("initramfs command failed: %s", errMsg)
	}

	return nil
}

// loadNouveauModule loads the nouveau kernel module.
func (s *NouveauRestoreStep) loadNouveauModule(ctx *install.Context) error {
	result := ctx.Executor.ExecuteElevated(ctx.Context(), "modprobe", "nouveau")

	if result.ExitCode != 0 {
		errMsg := strings.TrimSpace(string(result.Stderr))
		if errMsg == "" {
			errMsg = strings.TrimSpace(string(result.Stdout))
		}
		if errMsg == "" {
			errMsg = "unknown error"
		}
		return fmt.Errorf("modprobe nouveau failed: %s", errMsg)
	}

	return nil
}

// unloadNouveauModule unloads the nouveau kernel module.
func (s *NouveauRestoreStep) unloadNouveauModule(ctx *install.Context) error {
	result := ctx.Executor.ExecuteElevated(ctx.Context(), "modprobe", "-r", "nouveau")

	if result.ExitCode != 0 {
		errMsg := strings.TrimSpace(string(result.Stderr))
		if errMsg == "" {
			errMsg = strings.TrimSpace(string(result.Stdout))
		}
		if errMsg == "" {
			errMsg = "unknown error"
		}
		return fmt.Errorf("modprobe -r nouveau failed: %s", errMsg)
	}

	return nil
}

// Ensure NouveauRestoreStep implements the Step interface.
var _ install.Step = (*NouveauRestoreStep)(nil)
