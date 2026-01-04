// Package steps provides installation step implementations for Igor.
// Each step represents a discrete phase of the NVIDIA driver installation process.
package steps

import (
	"context"
	"fmt"
	"time"

	"github.com/tungetti/igor/internal/install"
	"github.com/tungetti/igor/internal/pkg"
	"github.com/tungetti/igor/internal/pkg/nvidia"
)

// State keys for package installation.
const (
	// StatePackagesInstalled indicates whether packages were successfully installed.
	StatePackagesInstalled = "packages_installed"
	// StateInstalledPackages stores the list of installed package names.
	StateInstalledPackages = "installed_packages"
	// StatePackageInstallTime stores the duration of the installation.
	StatePackageInstallTime = "package_install_time"
)

// PackageInstallationStep installs NVIDIA packages using the package manager.
// It computes the required packages based on the selected driver version and
// components, then installs them via the configured package manager.
type PackageInstallationStep struct {
	install.BaseStep
	additionalPackages []string                     // Extra packages to install beyond computed ones
	skipDependencies   bool                         // TODO: Implement to pass --nodeps or equivalent to package manager
	batchSize          int                          // How many packages to install at once (0 = all)
	preInstallHook     func(*install.Context) error // Hook before installation
	postInstallHook    func(*install.Context) error // Hook after installation
}

// PackageInstallationStepOption configures the PackageInstallationStep.
type PackageInstallationStepOption func(*PackageInstallationStep)

// WithAdditionalPackages adds extra packages to install beyond the computed ones.
// These packages are appended to the list of packages determined from the
// driver version and selected components.
func WithAdditionalPackages(packages ...string) PackageInstallationStepOption {
	return func(s *PackageInstallationStep) {
		s.additionalPackages = append(s.additionalPackages, packages...)
	}
}

// WithSkipDependencies configures whether to skip dependency checking.
// This is primarily useful for testing purposes.
func WithSkipDependencies(skip bool) PackageInstallationStepOption {
	return func(s *PackageInstallationStep) {
		s.skipDependencies = skip
	}
}

// WithBatchSize sets how many packages to install at once.
// If set to 0 (default), all packages are installed in a single operation.
// Setting a batch size can be useful for large installations or when
// needing to check for cancellation between batches.
func WithBatchSize(size int) PackageInstallationStepOption {
	return func(s *PackageInstallationStep) {
		s.batchSize = size
	}
}

// WithPreInstallHook sets a function to be called before package installation.
// If the hook returns an error, installation is aborted.
func WithPreInstallHook(fn func(*install.Context) error) PackageInstallationStepOption {
	return func(s *PackageInstallationStep) {
		s.preInstallHook = fn
	}
}

// WithPostInstallHook sets a function to be called after successful package installation.
// If the hook returns an error, the installation is marked as failed.
func WithPostInstallHook(fn func(*install.Context) error) PackageInstallationStepOption {
	return func(s *PackageInstallationStep) {
		s.postInstallHook = fn
	}
}

// NewPackageInstallationStep creates a new package installation step with the given options.
func NewPackageInstallationStep(opts ...PackageInstallationStepOption) *PackageInstallationStep {
	s := &PackageInstallationStep{
		BaseStep:           install.NewBaseStep("packages", "Install NVIDIA packages", true),
		additionalPackages: make([]string, 0),
		skipDependencies:   false,
		batchSize:          0,
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

// Execute installs the NVIDIA packages for the current distribution.
// It performs the following steps:
//  1. Validates prerequisites (package manager, distro info)
//  2. Computes the packages to install based on driver version and components
//  3. In dry-run mode, logs what would be installed
//  4. Runs pre-install hook if configured
//  5. Installs packages (in batches if configured)
//  6. Runs post-install hook if configured
//  7. Stores state for potential rollback
func (s *PackageInstallationStep) Execute(ctx *install.Context) install.StepResult {
	startTime := time.Now()

	// Check for cancellation
	if ctx.IsCancelled() {
		return install.FailStep("step cancelled", context.Canceled)
	}

	ctx.LogDebug("starting package installation")

	// Validate prerequisites
	if err := s.Validate(ctx); err != nil {
		return install.FailStep("validation failed", err).WithDuration(time.Since(startTime))
	}

	// Compute packages to install
	packages, err := s.computePackages(ctx)
	if err != nil {
		ctx.LogError("failed to compute packages", "error", err)
		return install.FailStep("failed to compute packages", err).WithDuration(time.Since(startTime))
	}

	if len(packages) == 0 {
		ctx.Log("no packages to install")
		return install.SkipStep("no packages to install").WithDuration(time.Since(startTime))
	}

	ctx.LogDebug("packages to install", "count", len(packages), "packages", packages)

	// Dry run mode
	if ctx.DryRun {
		ctx.Log("dry run: would install packages", "packages", packages)
		return install.CompleteStep("dry run: packages would be installed").WithDuration(time.Since(startTime))
	}

	// Check for cancellation before starting installation
	if ctx.IsCancelled() {
		return install.FailStep("step cancelled", context.Canceled).WithDuration(time.Since(startTime))
	}

	// Run pre-install hook if configured
	if s.preInstallHook != nil {
		ctx.LogDebug("running pre-install hook")
		if err := s.preInstallHook(ctx); err != nil {
			ctx.LogError("pre-install hook failed", "error", err)
			return install.FailStep("pre-install hook failed", err).WithDuration(time.Since(startTime))
		}
		ctx.LogDebug("pre-install hook completed successfully")
	}

	// Check for cancellation after pre-install hook
	if ctx.IsCancelled() {
		return install.FailStep("step cancelled", context.Canceled).WithDuration(time.Since(startTime))
	}

	// Install packages
	installedPackages, err := s.installPackages(ctx, packages)
	if err != nil {
		ctx.LogError("package installation failed", "error", err)
		// If some packages were installed, try to roll them back
		if len(installedPackages) > 0 {
			ctx.SetState(StateInstalledPackages, installedPackages)
			if rollbackErr := s.removePackages(ctx, installedPackages); rollbackErr != nil {
				ctx.LogWarn("failed to rollback partially installed packages", "error", rollbackErr)
			}
		}
		return install.FailStep("failed to install packages", err).WithDuration(time.Since(startTime))
	}

	// Store installed packages in state for rollback
	ctx.SetState(StateInstalledPackages, installedPackages)

	// Check for cancellation before post-install hook
	if ctx.IsCancelled() {
		// Try to rollback what we installed
		if rollbackErr := s.removePackages(ctx, installedPackages); rollbackErr != nil {
			ctx.LogWarn("failed to rollback packages after cancellation", "error", rollbackErr)
		}
		ctx.DeleteState(StateInstalledPackages)
		return install.FailStep("step cancelled", context.Canceled).WithDuration(time.Since(startTime))
	}

	// Run post-install hook if configured
	if s.postInstallHook != nil {
		ctx.LogDebug("running post-install hook")
		if err := s.postInstallHook(ctx); err != nil {
			ctx.LogError("post-install hook failed", "error", err)
			// Try to rollback installed packages
			if rollbackErr := s.removePackages(ctx, installedPackages); rollbackErr != nil {
				ctx.LogWarn("failed to rollback packages after post-install hook failure", "error", rollbackErr)
			}
			ctx.DeleteState(StateInstalledPackages)
			return install.FailStep("post-install hook failed", err).WithDuration(time.Since(startTime))
		}
		ctx.LogDebug("post-install hook completed successfully")
	}

	// Store state for rollback
	installDuration := time.Since(startTime)
	ctx.SetState(StatePackagesInstalled, true)
	ctx.SetState(StatePackageInstallTime, installDuration)

	ctx.Log("packages installed successfully", "count", len(installedPackages), "duration", installDuration)
	return install.CompleteStep("packages installed successfully").
		WithDuration(installDuration).
		WithCanRollback(true)
}

// Rollback removes the packages that were installed during execution.
// If no packages were installed, this is a no-op.
func (s *PackageInstallationStep) Rollback(ctx *install.Context) error {
	// Check if we actually installed packages
	if !ctx.GetStateBool(StatePackagesInstalled) {
		ctx.LogDebug("no packages were installed, nothing to rollback")
		return nil
	}

	// Get installed packages from state
	packagesRaw, ok := ctx.GetState(StateInstalledPackages)
	if !ok {
		ctx.LogDebug("installed packages list not found in state, nothing to rollback")
		return nil
	}

	packages, ok := packagesRaw.([]string)
	if !ok || len(packages) == 0 {
		ctx.LogDebug("no packages to rollback")
		return nil
	}

	// Validate package manager
	if ctx.PackageManager == nil {
		return fmt.Errorf("package manager not available for rollback")
	}

	ctx.Log("rolling back package installation", "count", len(packages))

	// Remove the packages
	if err := s.removePackages(ctx, packages); err != nil {
		ctx.LogError("failed to remove packages during rollback", "error", err)
		return fmt.Errorf("failed to remove packages: %w", err)
	}

	// Clear state
	ctx.DeleteState(StatePackagesInstalled)
	ctx.DeleteState(StateInstalledPackages)
	ctx.DeleteState(StatePackageInstallTime)

	ctx.LogDebug("package rollback completed", "count", len(packages))
	return nil
}

// Validate checks if the step can be executed with the given context.
// It ensures PackageManager, DistroInfo are available, and at least one
// component or driver version is specified.
func (s *PackageInstallationStep) Validate(ctx *install.Context) error {
	if ctx.PackageManager == nil {
		return fmt.Errorf("package manager is required for package installation")
	}
	if ctx.DistroInfo == nil {
		return fmt.Errorf("distribution info is required for package installation")
	}
	// Check that we have something to install
	if ctx.DriverVersion == "" && len(ctx.Components) == 0 && len(s.additionalPackages) == 0 {
		return fmt.Errorf("at least one component, driver version, or additional package is required")
	}
	return nil
}

// CanRollback returns true since package installation can be rolled back.
func (s *PackageInstallationStep) CanRollback() bool {
	return true
}

// computePackages determines which packages to install based on the context.
// It uses nvidia.GetPackageSet to get distribution-specific package names,
// then adds packages for the specified driver version and components.
func (s *PackageInstallationStep) computePackages(ctx *install.Context) ([]string, error) {
	if ctx.DistroInfo == nil {
		return nil, fmt.Errorf("no package set available: distribution info is nil")
	}

	packageSet := nvidia.GetPackageSet(ctx.DistroInfo)
	if packageSet == nil {
		return nil, fmt.Errorf("no package set available for distribution: %s", ctx.DistroInfo.ID)
	}

	// Use a map for deduplication
	seen := make(map[string]bool)
	var packages []string

	// Helper to add unique packages
	addPackages := func(pkgs []string) {
		for _, p := range pkgs {
			if p != "" && !seen[p] {
				seen[p] = true
				packages = append(packages, p)
			}
		}
	}

	// If driver version is specified, get version-specific packages
	if ctx.DriverVersion != "" {
		ctx.LogDebug("adding driver packages for version", "version", ctx.DriverVersion)
		versionPackages := packageSet.GetPackagesForVersion(ctx.DriverVersion)
		addPackages(versionPackages)
	}

	// Map component strings to nvidia.Component and get packages
	for _, componentStr := range ctx.Components {
		component := nvidia.Component(componentStr)
		if !component.IsValid() {
			ctx.LogWarn("unknown component, skipping", "component", componentStr)
			continue
		}

		ctx.LogDebug("adding packages for component", "component", componentStr)
		componentPackages := packageSet.GetPackages(component)
		addPackages(componentPackages)
	}

	// Add any additional packages
	if len(s.additionalPackages) > 0 {
		ctx.LogDebug("adding additional packages", "packages", s.additionalPackages)
		addPackages(s.additionalPackages)
	}

	return packages, nil
}

// installPackages installs the specified packages using the package manager.
// If batchSize is set, packages are installed in batches.
// Returns the list of packages that were successfully installed.
func (s *PackageInstallationStep) installPackages(ctx *install.Context, packages []string) ([]string, error) {
	opts := pkg.NonInteractiveInstallOptions()

	// If we have no batch size, install all at once
	if s.batchSize <= 0 {
		ctx.Log("installing packages", "count", len(packages))
		if err := ctx.PackageManager.Install(ctx.Context(), opts, packages...); err != nil {
			return nil, err
		}
		return packages, nil
	}

	// Install in batches
	var installedPackages []string
	for i := 0; i < len(packages); i += s.batchSize {
		// Check for cancellation before each batch
		if ctx.IsCancelled() {
			return installedPackages, context.Canceled
		}

		end := i + s.batchSize
		if end > len(packages) {
			end = len(packages)
		}
		batch := packages[i:end]

		ctx.Log("installing package batch", "batch", i/s.batchSize+1, "packages", batch)
		if err := ctx.PackageManager.Install(ctx.Context(), opts, batch...); err != nil {
			return installedPackages, fmt.Errorf("failed to install batch %d: %w", i/s.batchSize+1, err)
		}

		installedPackages = append(installedPackages, batch...)
	}

	return installedPackages, nil
}

// removePackages removes the specified packages using the package manager.
func (s *PackageInstallationStep) removePackages(ctx *install.Context, packages []string) error {
	if ctx.PackageManager == nil {
		return fmt.Errorf("package manager not available")
	}

	opts := pkg.RemoveOptions{
		NoConfirm:  true,
		AutoRemove: true,
	}

	ctx.Log("removing packages", "count", len(packages))
	return ctx.PackageManager.Remove(ctx.Context(), opts, packages...)
}

// Ensure PackageInstallationStep implements the Step interface.
var _ install.Step = (*PackageInstallationStep)(nil)
