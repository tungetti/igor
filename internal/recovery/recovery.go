// Package recovery provides TTY-based recovery mode for Igor.
// This module enables emergency NVIDIA driver uninstallation from a TTY/virtual
// console when X.org fails to start after a bad driver installation.
package recovery

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/tungetti/igor/internal/distro"
	"github.com/tungetti/igor/internal/exec"
	"github.com/tungetti/igor/internal/logging"
	"github.com/tungetti/igor/internal/pkg"
	"github.com/tungetti/igor/internal/uninstall"
)

// EnvironmentType represents where Igor is running.
type EnvironmentType int

const (
	// EnvironmentUnknown indicates the environment could not be determined.
	EnvironmentUnknown EnvironmentType = iota
	// EnvironmentTTY indicates a virtual console (no X/Wayland).
	EnvironmentTTY
	// EnvironmentGraphical indicates X11 or Wayland is running.
	EnvironmentGraphical
	// EnvironmentSSH indicates a remote SSH session.
	EnvironmentSSH
)

// String returns the string representation of the environment type.
func (e EnvironmentType) String() string {
	switch e {
	case EnvironmentTTY:
		return "tty"
	case EnvironmentGraphical:
		return "graphical"
	case EnvironmentSSH:
		return "ssh"
	default:
		return "unknown"
	}
}

// Environment contains information about the running environment.
type Environment struct {
	// Type indicates the detected environment type.
	Type EnvironmentType

	// Display is the DISPLAY environment variable (empty if no X11).
	Display string

	// WaylandDisplay is the WAYLAND_DISPLAY environment variable.
	WaylandDisplay string

	// TTY is the current TTY device (e.g., /dev/tty1).
	TTY string

	// IsRecoveryBoot indicates if the system booted in recovery/single-user mode.
	IsRecoveryBoot bool

	// Term is the TERM environment variable.
	Term string

	// SSHConnection is the SSH_CONNECTION environment variable.
	SSHConnection string
}

// EnvReader allows injecting environment variable reading for testing.
type EnvReader func(key string) string

// defaultEnvReader uses os.Getenv.
func defaultEnvReader(key string) string {
	return os.Getenv(key)
}

// FileReader allows injecting file reading for testing.
type FileReader func(path string) ([]byte, error)

// defaultFileReader uses os.ReadFile.
func defaultFileReader(path string) ([]byte, error) {
	return os.ReadFile(path)
}

// CommandRunner allows injecting command execution for testing.
type CommandRunner func(name string, args ...string) ([]byte, error)

// defaultCommandRunner uses exec.Command.
func defaultCommandRunner(name string, args ...string) ([]byte, error) {
	return nil, fmt.Errorf("command execution not supported in detection")
}

// DetectionOptions configures environment detection.
type DetectionOptions struct {
	EnvReader     EnvReader
	FileReader    FileReader
	CommandRunner CommandRunner
}

// DetectEnvironment checks the current running environment.
// It examines environment variables and system state to determine
// whether Igor is running in a TTY, graphical session, or SSH.
func DetectEnvironment() *Environment {
	return DetectEnvironmentWithOptions(DetectionOptions{})
}

// DetectEnvironmentWithOptions detects the environment using the provided options.
// This is useful for testing with mock environment variables.
func DetectEnvironmentWithOptions(opts DetectionOptions) *Environment {
	if opts.EnvReader == nil {
		opts.EnvReader = defaultEnvReader
	}
	if opts.FileReader == nil {
		opts.FileReader = defaultFileReader
	}

	env := &Environment{
		Display:        opts.EnvReader("DISPLAY"),
		WaylandDisplay: opts.EnvReader("WAYLAND_DISPLAY"),
		Term:           opts.EnvReader("TERM"),
		SSHConnection:  opts.EnvReader("SSH_CONNECTION"),
	}

	// Detect TTY
	tty := opts.EnvReader("TTY")
	if tty == "" {
		// Try to read from /proc/self/fd/0
		if link, err := os.Readlink("/proc/self/fd/0"); err == nil {
			if strings.HasPrefix(link, "/dev/tty") || strings.HasPrefix(link, "/dev/pts") {
				tty = link
			}
		}
	}
	env.TTY = tty

	// Check for recovery boot
	env.IsRecoveryBoot = detectRecoveryBoot(opts.FileReader)

	// Determine environment type
	env.Type = determineEnvironmentType(env)

	return env
}

// detectRecoveryBoot checks if the system was booted in recovery mode.
func detectRecoveryBoot(fileReader FileReader) bool {
	cmdline, err := fileReader("/proc/cmdline")
	if err != nil {
		return false
	}

	cmdlineStr := strings.ToLower(string(cmdline))

	// Keywords that can appear anywhere
	containsKeywords := []string{"single", "rescue", "recovery", "emergency", "init=/bin/sh", "init=/bin/bash", "runlevel=1"}

	for _, keyword := range containsKeywords {
		if strings.Contains(cmdlineStr, keyword) {
			return true
		}
	}

	// Check for standalone "1" as a boot parameter (runlevel 1)
	// Must be a separate word, not part of another parameter like "sda1"
	fields := strings.Fields(cmdlineStr)
	for _, field := range fields {
		if field == "1" || field == "s" {
			return true
		}
	}

	return false
}

// determineEnvironmentType determines the environment type from the collected info.
func determineEnvironmentType(env *Environment) EnvironmentType {
	// Check for SSH first
	if env.SSHConnection != "" {
		return EnvironmentSSH
	}

	// Check for graphical environment
	if env.Display != "" || env.WaylandDisplay != "" {
		return EnvironmentGraphical
	}

	// Check for TTY indicators
	// TERM=linux is typical for virtual consoles
	if env.Term == "linux" {
		return EnvironmentTTY
	}

	// Check if we're on a real TTY (not pts)
	if strings.HasPrefix(env.TTY, "/dev/tty") && !strings.HasPrefix(env.TTY, "/dev/tty/") {
		return EnvironmentTTY
	}

	// If recovery boot and no graphical, assume TTY
	if env.IsRecoveryBoot {
		return EnvironmentTTY
	}

	return EnvironmentUnknown
}

// IsRecoveryMode returns true if the environment indicates we should use
// recovery/TTY mode for interaction.
func (e *Environment) IsRecoveryMode() bool {
	// Use recovery mode if:
	// 1. We're in a TTY (no graphical session)
	// 2. We booted in recovery mode
	// 3. We're in an unknown environment (play it safe)
	switch e.Type {
	case EnvironmentTTY:
		return true
	case EnvironmentUnknown:
		return e.IsRecoveryBoot
	default:
		return false
	}
}

// IsGraphical returns true if a graphical environment is available.
func (e *Environment) IsGraphical() bool {
	return e.Type == EnvironmentGraphical
}

// IsSSH returns true if we're in an SSH session.
func (e *Environment) IsSSH() bool {
	return e.Type == EnvironmentSSH
}

// RootChecker is a function that checks if the process has root privileges.
type RootChecker func() bool

// defaultRootChecker uses os.Geteuid to check for root.
func defaultRootChecker() bool {
	return os.Geteuid() == 0
}

// RecoveryMode handles emergency uninstallation from TTY.
// It provides a text-based interface for removing NVIDIA drivers
// when the graphical environment is unavailable.
type RecoveryMode struct {
	mu           sync.RWMutex
	env          *Environment
	ui           *TTYUI
	orchestrator *uninstall.UninstallOrchestrator
	discovery    uninstall.Discovery
	distroInfo   *distro.Distribution
	executor     exec.Executor
	pkgManager   pkg.Manager
	logger       logging.Logger
	dryRun       bool
	rootChecker  RootChecker
}

// RecoveryOption is a functional option for RecoveryMode.
type RecoveryOption func(*RecoveryMode)

// NewRecoveryMode creates a new recovery mode instance with the given options.
func NewRecoveryMode(opts ...RecoveryOption) *RecoveryMode {
	r := &RecoveryMode{
		ui:          NewTTYUI(),
		rootChecker: defaultRootChecker,
	}

	for _, opt := range opts {
		opt(r)
	}

	// Set up default logger if not provided
	if r.logger == nil {
		r.logger = logging.NewNop()
	}

	// Detect environment if not provided
	if r.env == nil {
		r.env = DetectEnvironment()
	}

	// Set up discovery if we have a package manager
	if r.discovery == nil && r.pkgManager != nil {
		r.discovery = uninstall.NewPackageDiscovery(
			r.pkgManager,
			uninstall.WithDiscoveryDistro(r.distroInfo),
			uninstall.WithDiscoveryExecutor(r.executor),
		)
	}

	return r
}

// WithRecoveryEnvironment sets the environment for recovery mode.
func WithRecoveryEnvironment(env *Environment) RecoveryOption {
	return func(r *RecoveryMode) {
		r.env = env
	}
}

// WithRecoveryDistro sets the distribution information.
func WithRecoveryDistro(d *distro.Distribution) RecoveryOption {
	return func(r *RecoveryMode) {
		r.distroInfo = d
	}
}

// WithRecoveryExecutor sets the command executor.
func WithRecoveryExecutor(e exec.Executor) RecoveryOption {
	return func(r *RecoveryMode) {
		r.executor = e
	}
}

// WithRecoveryPackageManager sets the package manager.
func WithRecoveryPackageManager(pm pkg.Manager) RecoveryOption {
	return func(r *RecoveryMode) {
		r.pkgManager = pm
	}
}

// WithRecoveryLogger sets the logger.
func WithRecoveryLogger(l logging.Logger) RecoveryOption {
	return func(r *RecoveryMode) {
		r.logger = l
	}
}

// WithRecoveryUI sets the TTY UI.
func WithRecoveryUI(ui *TTYUI) RecoveryOption {
	return func(r *RecoveryMode) {
		r.ui = ui
	}
}

// WithRecoveryDiscovery sets the package discovery instance.
func WithRecoveryDiscovery(d uninstall.Discovery) RecoveryOption {
	return func(r *RecoveryMode) {
		r.discovery = d
	}
}

// WithRecoveryOrchestrator sets the uninstall orchestrator.
func WithRecoveryOrchestrator(o *uninstall.UninstallOrchestrator) RecoveryOption {
	return func(r *RecoveryMode) {
		r.orchestrator = o
	}
}

// WithRecoveryDryRun enables dry-run mode (no actual changes).
func WithRecoveryDryRun(dryRun bool) RecoveryOption {
	return func(r *RecoveryMode) {
		r.dryRun = dryRun
	}
}

// WithRecoveryRootChecker sets a custom root privilege checker (for testing).
func WithRecoveryRootChecker(checker RootChecker) RecoveryOption {
	return func(r *RecoveryMode) {
		r.rootChecker = checker
	}
}

// Run executes the recovery mode workflow.
// It guides the user through discovering and removing NVIDIA packages.
// Returns an error if recovery fails.
func (r *RecoveryMode) Run(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.logger.Info("Starting recovery mode")

	// Display header
	r.ui.Header("NVIDIA Driver Recovery Mode")
	r.ui.Blank()

	// Check if we should be in recovery mode
	if r.env != nil && !r.env.IsRecoveryMode() && r.env.Type != EnvironmentSSH {
		r.ui.Warning("A graphical environment appears to be running.")
		r.ui.Info("Recovery mode is designed for TTY/console use.")
		if !r.ui.Confirm("Continue anyway?", false) {
			r.ui.Info("Cancelled. Use 'igor uninstall' in graphical mode instead.")
			return nil
		}
	}

	// Step 1: Show environment info
	r.ui.Info("Detecting environment...")
	r.showEnvironmentInfo()
	r.ui.Blank()

	// Step 2: Check for root privileges
	if !r.rootChecker() {
		r.ui.Error("Recovery mode requires root privileges.")
		r.ui.Info("Please run with: sudo igor recovery")
		return fmt.Errorf("root privileges required")
	}
	r.ui.Success("Running as root")
	r.ui.Blank()

	// Step 3: Discover packages
	r.ui.Info("Scanning for NVIDIA packages...")
	if r.discovery == nil {
		r.ui.Error("Package discovery not configured.")
		return fmt.Errorf("discovery not configured: package manager required")
	}

	packages, err := r.discovery.Discover(ctx)
	if err != nil {
		r.ui.Error(fmt.Sprintf("Failed to discover packages: %v", err))
		return fmt.Errorf("discovery failed: %w", err)
	}

	if packages.IsEmpty() {
		r.ui.Warning("No NVIDIA packages found on this system.")
		r.ui.Info("There may be nothing to uninstall.")
		return nil
	}

	r.ui.Success(fmt.Sprintf("Found %d NVIDIA package(s)", packages.TotalCount))
	r.ui.Blank()

	// Step 4: Show what will be removed
	r.ui.ShowPackages(packages.AllPackages)

	// Step 5: Show additional info
	if packages.DriverVersion != "" {
		r.ui.Info(fmt.Sprintf("Driver version: %s", packages.DriverVersion))
	}
	if packages.CUDAVersion != "" {
		r.ui.Info(fmt.Sprintf("CUDA version: %s", packages.CUDAVersion))
	}
	r.ui.Blank()

	// Step 6: Confirm with user
	r.ui.ShowWarning("This will remove all NVIDIA packages and restore the nouveau driver.")
	r.ui.Print("Your system will need to reboot after uninstallation.")
	r.ui.Blank()

	if r.dryRun {
		r.ui.Info("[DRY RUN] No changes will be made.")
		r.ui.Blank()
	}

	if !r.ui.Confirm("Remove these packages and restore nouveau?", true) {
		r.ui.Warning("Cancelled by user")
		r.logger.Info("Recovery cancelled by user")
		return nil
	}

	// Step 7: Execute uninstall
	r.ui.Blank()
	r.ui.Info("Starting uninstallation...")
	r.ui.Separator()

	err = r.ExecuteUninstall(ctx, packages.AllPackages, true)
	if err != nil {
		r.ui.ShowError("Uninstallation failed", []string{
			err.Error(),
			"Check the logs for more details.",
			"You may need to manually remove packages.",
		})
		return err
	}

	// Step 8: Show success
	details := []string{
		"All NVIDIA packages have been removed.",
		"The nouveau driver should be restored.",
		"Please reboot your system to complete the recovery.",
	}
	r.ui.ShowResult(true, "Recovery completed successfully!", details)

	r.ui.Blank()
	if r.ui.Confirm("Reboot now?", false) {
		r.ui.Info("Rebooting...")
		if r.executor != nil && !r.dryRun {
			result := r.executor.ExecuteElevated(ctx, "reboot")
			if result.Error != nil {
				r.ui.Warning(fmt.Sprintf("Failed to reboot: %v", result.Error))
				r.ui.Info("Please reboot manually: sudo reboot")
			}
		} else {
			r.ui.Info("[DRY RUN] Would execute: reboot")
		}
	} else {
		r.ui.Info("Remember to reboot before starting X.org.")
	}

	return nil
}

// showEnvironmentInfo displays detected environment information.
func (r *RecoveryMode) showEnvironmentInfo() {
	if r.env == nil {
		r.ui.Warning("Environment not detected")
		return
	}

	r.ui.Print(fmt.Sprintf("  Environment: %s", r.env.Type))
	if r.env.TTY != "" {
		r.ui.Print(fmt.Sprintf("  TTY: %s", r.env.TTY))
	}
	if r.env.Term != "" {
		r.ui.Print(fmt.Sprintf("  TERM: %s", r.env.Term))
	}
	if r.env.IsRecoveryBoot {
		r.ui.Warning("System booted in recovery mode")
	}
	if r.distroInfo != nil {
		r.ui.Print(fmt.Sprintf("  Distribution: %s", r.distroInfo.String()))
	}
}

// ExecuteUninstall performs the actual uninstallation.
// packages is the list of package names to remove.
// restoreNouveau indicates whether to restore the nouveau driver.
func (r *RecoveryMode) ExecuteUninstall(ctx context.Context, packages []string, restoreNouveau bool) error {
	r.logger.Info("Executing uninstall", "packages", len(packages), "restoreNouveau", restoreNouveau)

	if len(packages) == 0 {
		r.ui.Warning("No packages to remove")
		return nil
	}

	totalSteps := len(packages)
	if restoreNouveau {
		totalSteps += 2 // unblock + rebuild initramfs
	}

	currentStep := 0

	// Remove packages one by one (safer for recovery)
	if r.pkgManager != nil {
		for i, pkgName := range packages {
			currentStep++
			r.ui.ShowStep(currentStep, totalSteps, fmt.Sprintf("Removing %s...", pkgName))

			if r.dryRun {
				r.ui.StepSuccess(currentStep, totalSteps, fmt.Sprintf("[DRY RUN] Would remove %s", pkgName))
				continue
			}

			removeOpts := pkg.RemoveOptions{
				Purge:      true,
				NoConfirm:  true,
				AutoRemove: false,
			}

			err := r.pkgManager.Remove(ctx, removeOpts, pkgName)
			if err != nil {
				r.ui.StepFailed(currentStep, totalSteps, fmt.Sprintf("Failed to remove %s: %v", pkgName, err))
				r.logger.Error("Package removal failed", "package", pkgName, "error", err)
				// Continue with other packages - we want to remove as much as possible
			} else {
				r.ui.StepSuccess(currentStep, totalSteps, fmt.Sprintf("Removed %s", pkgName))
				r.logger.Info("Package removed", "package", pkgName, "progress", fmt.Sprintf("%d/%d", i+1, len(packages)))
			}
		}
	} else {
		r.ui.Warning("No package manager configured - skipping package removal")
	}

	// Restore nouveau if requested
	if restoreNouveau && r.executor != nil {
		// Step: Remove nouveau from blacklist
		currentStep++
		r.ui.ShowStep(currentStep, totalSteps, "Removing nouveau blacklist...")

		blacklistFiles := []string{
			"/etc/modprobe.d/nvidia-installer-disable-nouveau.conf",
			"/etc/modprobe.d/blacklist-nouveau.conf",
			"/etc/modprobe.d/nvidia.conf",
		}

		if r.dryRun {
			r.ui.StepSuccess(currentStep, totalSteps, "[DRY RUN] Would remove blacklist files")
		} else {
			for _, f := range blacklistFiles {
				if _, err := os.Stat(f); err == nil {
					if err := os.Remove(f); err != nil {
						r.logger.Warn("Failed to remove blacklist file", "file", f, "error", err)
					} else {
						r.logger.Info("Removed blacklist file", "file", f)
					}
				}
			}
			r.ui.StepSuccess(currentStep, totalSteps, "Nouveau blacklist removed")
		}

		// Step: Rebuild initramfs
		currentStep++
		r.ui.ShowStep(currentStep, totalSteps, "Rebuilding initramfs...")

		if r.dryRun {
			r.ui.StepSuccess(currentStep, totalSteps, "[DRY RUN] Would rebuild initramfs")
		} else {
			err := r.rebuildInitramfs(ctx)
			if err != nil {
				r.ui.StepFailed(currentStep, totalSteps, fmt.Sprintf("Failed to rebuild initramfs: %v", err))
				r.logger.Error("Initramfs rebuild failed", "error", err)
				// This is not fatal - user can do it manually
			} else {
				r.ui.StepSuccess(currentStep, totalSteps, "Initramfs rebuilt")
			}
		}
	}

	return nil
}

// rebuildInitramfs rebuilds the initial ramdisk to include nouveau.
func (r *RecoveryMode) rebuildInitramfs(ctx context.Context) error {
	if r.executor == nil {
		return fmt.Errorf("executor not configured")
	}

	// Try different initramfs rebuild commands based on what's available
	commands := []struct {
		cmd  string
		args []string
	}{
		{"update-initramfs", []string{"-u"}}, // Debian/Ubuntu
		{"dracut", []string{"--force"}},      // Fedora/RHEL
		{"mkinitcpio", []string{"-P"}},       // Arch
		{"mkinitrd", []string{}},             // SUSE
	}

	for _, c := range commands {
		// Check if command exists
		checkResult := r.executor.Execute(ctx, "which", c.cmd)
		if checkResult.ExitCode != 0 {
			continue
		}

		// Execute the command
		r.logger.Info("Rebuilding initramfs", "command", c.cmd)
		result := r.executor.ExecuteElevated(ctx, c.cmd, c.args...)
		if result.Error != nil {
			return result.Error
		}
		if result.ExitCode != 0 {
			return fmt.Errorf("%s failed with exit code %d: %s", c.cmd, result.ExitCode, string(result.Stderr))
		}

		return nil
	}

	return fmt.Errorf("no initramfs rebuild command found")
}

// Environment returns the detected environment.
func (r *RecoveryMode) Environment() *Environment {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.env
}

// UI returns the TTY UI instance.
func (r *RecoveryMode) UI() *TTYUI {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.ui
}

// IsDryRun returns whether dry-run mode is enabled.
func (r *RecoveryMode) IsDryRun() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.dryRun
}
