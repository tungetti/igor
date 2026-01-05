// Package steps provides uninstallation step implementations for Igor.
// Each step represents a discrete phase of the NVIDIA driver uninstallation process.
package steps

import (
	"context"
	"fmt"
	"time"

	"github.com/tungetti/igor/internal/install"
	"github.com/tungetti/igor/internal/pkg"
	"github.com/tungetti/igor/internal/uninstall"
)

// State keys for package removal.
const (
	// StatePackagesRemoved indicates packages were removed successfully.
	StatePackagesRemoved = "packages_removed"
	// StateRemovedPackages is the list of successfully removed packages.
	StateRemovedPackages = "removed_packages"
	// StateFailedPackages is the list of packages that failed to remove.
	StateFailedPackages = "failed_packages"
	// StateRemovalPurged indicates if purge mode was used.
	StateRemovalPurged = "removal_purged"
)

// Discovery is the interface for discovering installed NVIDIA packages.
// This is an alias to the uninstall.Discovery interface for convenience.
type Discovery = uninstall.Discovery

// PackageRemovalStep removes NVIDIA packages from the system.
type PackageRemovalStep struct {
	install.BaseStep
	discovery        Discovery
	packagesToRemove []string // Specific packages to remove (optional)
	removeAll        bool     // Remove all discovered NVIDIA packages
	purge            bool     // Purge config files (apt purge vs apt remove)
	autoRemove       bool     // Remove orphaned dependencies
	batchSize        int      // How many packages to remove at once (0 = all at once)
}

// RemovalStepOption configures the package removal step.
type RemovalStepOption func(*PackageRemovalStep)

// WithPackagesToRemove sets specific packages to remove.
func WithPackagesToRemove(packages []string) RemovalStepOption {
	return func(s *PackageRemovalStep) {
		s.packagesToRemove = append(s.packagesToRemove, packages...)
	}
}

// WithRemoveAll removes all discovered NVIDIA packages.
func WithRemoveAll(removeAll bool) RemovalStepOption {
	return func(s *PackageRemovalStep) {
		s.removeAll = removeAll
	}
}

// WithPurge enables purging of configuration files (where supported).
func WithPurge(purge bool) RemovalStepOption {
	return func(s *PackageRemovalStep) {
		s.purge = purge
	}
}

// WithAutoRemove enables automatic removal of orphaned dependencies.
func WithAutoRemove(autoRemove bool) RemovalStepOption {
	return func(s *PackageRemovalStep) {
		s.autoRemove = autoRemove
	}
}

// WithRemovalBatchSize sets how many packages to remove at once.
// If set to 0 (default), all packages are removed in a single operation.
func WithRemovalBatchSize(size int) RemovalStepOption {
	return func(s *PackageRemovalStep) {
		s.batchSize = size
	}
}

// WithRemovalDiscovery sets the discovery instance for finding packages.
func WithRemovalDiscovery(discovery Discovery) RemovalStepOption {
	return func(s *PackageRemovalStep) {
		s.discovery = discovery
	}
}

// NewPackageRemovalStep creates a new package removal step.
func NewPackageRemovalStep(opts ...RemovalStepOption) *PackageRemovalStep {
	s := &PackageRemovalStep{
		BaseStep:         install.NewBaseStep("removal", "Remove NVIDIA packages", false),
		packagesToRemove: make([]string, 0),
		removeAll:        false,
		purge:            false,
		autoRemove:       true, // Default to auto-remove orphaned dependencies
		batchSize:        0,
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

// Execute removes the NVIDIA packages from the system.
// It performs the following steps:
//  1. Checks for cancellation
//  2. Validates context has package manager
//  3. If removeAll is true, uses discovery to find all NVIDIA packages
//  4. If specific packages provided, uses those
//  5. If no packages to remove, returns skip result
//  6. In dry-run mode, logs what would be removed and returns success
//  7. Removes packages using package manager (in batches if configured)
//  8. Tracks removed and failed packages in context state
//  9. Stores results for the UninstallResult
func (s *PackageRemovalStep) Execute(ctx *install.Context) install.StepResult {
	startTime := time.Now()

	// Check for cancellation
	if ctx.IsCancelled() {
		return install.FailStep("step cancelled", context.Canceled)
	}

	ctx.LogDebug("starting package removal")

	// Validate prerequisites
	if err := s.Validate(ctx); err != nil {
		return install.FailStep("validation failed", err).WithDuration(time.Since(startTime))
	}

	// Determine packages to remove
	packages, err := s.determinePackages(ctx)
	if err != nil {
		ctx.LogError("failed to determine packages to remove", "error", err)
		return install.FailStep("failed to determine packages", err).WithDuration(time.Since(startTime))
	}

	if len(packages) == 0 {
		ctx.Log("no packages to remove")
		return install.SkipStep("no packages to remove").WithDuration(time.Since(startTime))
	}

	ctx.LogDebug("packages to remove", "count", len(packages), "packages", packages)

	// Dry run mode
	if ctx.DryRun {
		ctx.Log("dry run: would remove packages", "packages", packages, "purge", s.purge)
		return install.CompleteStep("dry run: packages would be removed").WithDuration(time.Since(startTime))
	}

	// Check for cancellation before starting removal
	if ctx.IsCancelled() {
		return install.FailStep("step cancelled", context.Canceled).WithDuration(time.Since(startTime))
	}

	// Remove packages
	removedPackages, failedPackages, err := s.removePackages(ctx, packages)

	// Store state regardless of outcome
	ctx.SetState(StateRemovedPackages, removedPackages)
	ctx.SetState(StateFailedPackages, failedPackages)
	ctx.SetState(StateRemovalPurged, s.purge)

	// Determine result based on outcome
	removalDuration := time.Since(startTime)

	if len(removedPackages) == 0 && len(failedPackages) > 0 {
		// Complete failure - no packages removed
		ctx.LogError("package removal failed", "error", err, "failed_packages", failedPackages)
		ctx.SetState(StatePackagesRemoved, false)
		return install.FailStep("failed to remove any packages", err).WithDuration(removalDuration)
	}

	if len(failedPackages) > 0 {
		// Partial failure - some packages removed, some failed
		ctx.LogWarn("some packages failed to remove",
			"removed", len(removedPackages),
			"failed", len(failedPackages),
			"failed_packages", failedPackages)
		ctx.SetState(StatePackagesRemoved, true)
		return install.NewStepResult(install.StepStatusCompleted,
			fmt.Sprintf("partially removed packages: %d removed, %d failed", len(removedPackages), len(failedPackages))).
			WithDuration(removalDuration).
			WithCanRollback(false)
	}

	// Complete success
	ctx.SetState(StatePackagesRemoved, true)
	ctx.Log("packages removed successfully", "count", len(removedPackages), "duration", removalDuration)
	return install.CompleteStep("packages removed successfully").
		WithDuration(removalDuration).
		WithCanRollback(false)
}

// Rollback is a no-op for package removal as it would require re-installing packages.
// Package removal is not easily reversible during uninstall operations.
func (s *PackageRemovalStep) Rollback(ctx *install.Context) error {
	ctx.LogWarn("package removal rollback is not supported - packages cannot be automatically re-installed")
	return nil
}

// Validate checks if the step can be executed with the given context.
func (s *PackageRemovalStep) Validate(ctx *install.Context) error {
	if ctx == nil {
		return fmt.Errorf("context is nil")
	}
	if ctx.PackageManager == nil {
		return fmt.Errorf("package manager is required for package removal")
	}
	// Check that we have some way to determine packages
	if !s.removeAll && len(s.packagesToRemove) == 0 {
		return fmt.Errorf("either removeAll must be true or packagesToRemove must be specified")
	}
	// If removeAll is true, we need a discovery instance
	if s.removeAll && s.discovery == nil {
		return fmt.Errorf("discovery is required when removeAll is true")
	}
	return nil
}

// CanRollback returns false since package removal is not easily reversible.
func (s *PackageRemovalStep) CanRollback() bool {
	return false
}

// determinePackages determines which packages to remove based on configuration.
func (s *PackageRemovalStep) determinePackages(ctx *install.Context) ([]string, error) {
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

	// If removeAll is true, discover all NVIDIA packages
	if s.removeAll && s.discovery != nil {
		ctx.LogDebug("discovering all NVIDIA packages")
		discovered, err := s.discovery.Discover(ctx.Context())
		if err != nil {
			return nil, fmt.Errorf("failed to discover NVIDIA packages: %w", err)
		}
		if discovered != nil && len(discovered.AllPackages) > 0 {
			ctx.LogDebug("discovered packages", "count", len(discovered.AllPackages))
			addPackages(discovered.AllPackages)
		}
	}

	// Add specific packages if provided
	if len(s.packagesToRemove) > 0 {
		ctx.LogDebug("adding specified packages", "packages", s.packagesToRemove)
		addPackages(s.packagesToRemove)
	}

	return packages, nil
}

// removePackages removes the specified packages using the package manager.
// Returns the list of successfully removed packages, failed packages, and any error.
func (s *PackageRemovalStep) removePackages(ctx *install.Context, packages []string) ([]string, []string, error) {
	opts := pkg.RemoveOptions{
		Purge:      s.purge,
		AutoRemove: s.autoRemove,
		NoConfirm:  true,
	}

	// If we have no batch size, remove all at once
	if s.batchSize <= 0 {
		ctx.Log("removing packages", "count", len(packages), "purge", s.purge)
		if err := ctx.PackageManager.Remove(ctx.Context(), opts, packages...); err != nil {
			// Check for context cancellation
			if ctx.IsCancelled() {
				return nil, packages, context.Canceled
			}
			return nil, packages, err
		}
		return packages, nil, nil
	}

	// Remove in batches
	var removedPackages []string
	var failedPackages []string
	var lastError error

	for i := 0; i < len(packages); i += s.batchSize {
		// Check for cancellation before each batch
		if ctx.IsCancelled() {
			// Mark remaining packages as failed
			failedPackages = append(failedPackages, packages[i:]...)
			return removedPackages, failedPackages, context.Canceled
		}

		end := i + s.batchSize
		if end > len(packages) {
			end = len(packages)
		}
		batch := packages[i:end]

		ctx.Log("removing package batch", "batch", i/s.batchSize+1, "packages", batch)
		if err := ctx.PackageManager.Remove(ctx.Context(), opts, batch...); err != nil {
			ctx.LogError("failed to remove batch", "batch", i/s.batchSize+1, "error", err)
			failedPackages = append(failedPackages, batch...)
			lastError = err
			// Continue with next batch instead of stopping
			continue
		}

		removedPackages = append(removedPackages, batch...)
	}

	return removedPackages, failedPackages, lastError
}

// Ensure PackageRemovalStep implements the Step interface.
var _ install.Step = (*PackageRemovalStep)(nil)
