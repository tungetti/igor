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

// State keys for repository configuration.
const (
	// StateRepositoryConfigured indicates whether a repository was successfully configured.
	StateRepositoryConfigured = "repository_configured"
	// StateRepositoryName stores the name of the configured repository.
	StateRepositoryName = "repository_name"
)

// RepositoryStep configures NVIDIA repositories for the detected Linux distribution.
// It adds the appropriate NVIDIA repository based on the distribution family and
// updates the package lists to make NVIDIA packages available for installation.
type RepositoryStep struct {
	install.BaseStep
	skipUpdate bool
}

// RepositoryStepOption configures the repository step.
type RepositoryStepOption func(*RepositoryStep)

// WithSkipUpdate configures whether to skip the package list update after adding
// the repository. This can be useful when multiple repositories are being added
// and you want to update only once at the end.
func WithSkipUpdate(skip bool) RepositoryStepOption {
	return func(s *RepositoryStep) {
		s.skipUpdate = skip
	}
}

// NewRepositoryStep creates a new repository configuration step with the given options.
func NewRepositoryStep(opts ...RepositoryStepOption) *RepositoryStep {
	s := &RepositoryStep{
		BaseStep:   install.NewBaseStep("repository", "Configure NVIDIA repository", true),
		skipUpdate: false,
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

// Execute configures the NVIDIA repository for the current distribution.
// It performs the following steps:
//  1. Gets the appropriate repository for the distribution
//  2. For Arch Linux, skips as no external repository is needed
//  3. In dry-run mode, logs the action without making changes
//  4. Adds the repository using the package manager
//  5. Updates package lists (unless skipUpdate is set)
//  6. Stores state for potential rollback
func (s *RepositoryStep) Execute(ctx *install.Context) install.StepResult {
	startTime := time.Now()

	// Check for cancellation
	if ctx.IsCancelled() {
		return install.FailStep("step cancelled", context.Canceled)
	}

	ctx.LogDebug("starting repository configuration")

	// Validate prerequisites
	if err := s.Validate(ctx); err != nil {
		return install.FailStep("validation failed", err).WithDuration(time.Since(startTime))
	}

	// Get repository for this distribution
	repo, err := nvidia.GetRepository(ctx.DistroInfo)
	if err != nil {
		ctx.LogError("failed to get repository info", "error", err)
		return install.FailStep("failed to get repository info", err).WithDuration(time.Since(startTime))
	}

	// Arch Linux doesn't need an external repository
	if repo == nil {
		ctx.Log("no external repository needed for this distribution", "distro", ctx.DistroInfo.ID)
		return install.CompleteStep("no repository configuration required").WithDuration(time.Since(startTime))
	}

	ctx.LogDebug("repository selected", "name", repo.Name, "url", repo.URL, "type", repo.Type)

	// Dry run mode
	if ctx.DryRun {
		ctx.Log("dry run: would add repository", "name", repo.Name, "url", repo.URL)
		if !s.skipUpdate {
			ctx.Log("dry run: would update package lists")
		}
		return install.CompleteStep("dry run: repository would be added").WithDuration(time.Since(startTime))
	}

	// Check for cancellation before adding repository
	if ctx.IsCancelled() {
		return install.FailStep("step cancelled", context.Canceled).WithDuration(time.Since(startTime))
	}

	// Add repository
	ctx.Log("adding repository", "name", repo.Name)
	if err := ctx.PackageManager.AddRepository(ctx.Context(), *repo); err != nil {
		ctx.LogError("failed to add repository", "name", repo.Name, "error", err)
		return install.FailStep("failed to add repository", err).WithDuration(time.Since(startTime))
	}

	ctx.LogDebug("repository added successfully", "name", repo.Name)

	// Update package lists unless skipped
	if !s.skipUpdate {
		// Check for cancellation before updating
		if ctx.IsCancelled() {
			// Try to rollback the repository we just added
			_ = ctx.PackageManager.RemoveRepository(ctx.Context(), repo.Name)
			return install.FailStep("step cancelled", context.Canceled).WithDuration(time.Since(startTime))
		}

		ctx.Log("updating package lists")
		if err := ctx.PackageManager.Update(ctx.Context(), pkg.DefaultUpdateOptions()); err != nil {
			ctx.LogError("failed to update package lists", "error", err)
			// Try to rollback the repository we just added
			rollbackErr := ctx.PackageManager.RemoveRepository(ctx.Context(), repo.Name)
			if rollbackErr != nil {
				ctx.LogError("failed to rollback repository", "name", repo.Name, "error", rollbackErr)
			}
			return install.FailStep("failed to update package lists", err).WithDuration(time.Since(startTime))
		}
		ctx.LogDebug("package lists updated successfully")
	}

	// Store state for rollback
	ctx.SetState(StateRepositoryConfigured, true)
	ctx.SetState(StateRepositoryName, repo.Name)

	ctx.Log("repository configured successfully", "name", repo.Name)
	return install.CompleteStep("repository configured successfully").
		WithDuration(time.Since(startTime)).
		WithCanRollback(true)
}

// Rollback removes the repository that was added during execution.
// If no repository was configured, this is a no-op.
func (s *RepositoryStep) Rollback(ctx *install.Context) error {
	// Check if we actually configured a repository
	if !ctx.GetStateBool(StateRepositoryConfigured) {
		ctx.LogDebug("no repository was configured, nothing to rollback")
		return nil
	}

	repoName := ctx.GetStateString(StateRepositoryName)
	if repoName == "" {
		ctx.LogDebug("repository name not found in state, nothing to rollback")
		return nil
	}

	// Validate package manager
	if ctx.PackageManager == nil {
		return fmt.Errorf("package manager not available for rollback")
	}

	ctx.Log("rolling back repository configuration", "name", repoName)

	// Remove the repository
	if err := ctx.PackageManager.RemoveRepository(ctx.Context(), repoName); err != nil {
		ctx.LogError("failed to remove repository during rollback", "name", repoName, "error", err)
		return fmt.Errorf("failed to remove repository '%s': %w", repoName, err)
	}

	// Clear state
	ctx.DeleteState(StateRepositoryConfigured)
	ctx.DeleteState(StateRepositoryName)

	ctx.LogDebug("repository rollback completed", "name", repoName)
	return nil
}

// Validate checks if the step can be executed with the given context.
// It ensures both PackageManager and DistroInfo are available.
func (s *RepositoryStep) Validate(ctx *install.Context) error {
	if ctx.PackageManager == nil {
		return fmt.Errorf("package manager is required for repository configuration")
	}
	if ctx.DistroInfo == nil {
		return fmt.Errorf("distribution info is required for repository configuration")
	}
	return nil
}

// CanRollback returns true since repository configuration can be rolled back.
func (s *RepositoryStep) CanRollback() bool {
	return true
}

// Ensure RepositoryStep implements the Step interface.
var _ install.Step = (*RepositoryStep)(nil)
