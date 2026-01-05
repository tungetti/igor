// Package uninstall provides the uninstallation workflow framework for Igor.
// It defines interfaces and types for creating, executing, and managing
// multi-step uninstallation workflows.
package uninstall

import (
	"fmt"
	"time"
)

// State keys for uninstallation workflow state management.
const (
	// StatePackagesRemoved indicates packages were removed.
	StatePackagesRemoved = "packages_removed"
	// StateRemovedPackages is the list of removed package names.
	StateRemovedPackages = "removed_packages"
	// StateConfigsCleaned indicates configs were cleaned.
	StateConfigsCleaned = "configs_cleaned"
	// StateCleanedConfigs is the list of cleaned config paths.
	StateCleanedConfigs = "cleaned_configs"
	// StateModulesUnloaded indicates kernel modules were unloaded.
	StateModulesUnloaded = "modules_unloaded"
	// StateNouveauUnblocked indicates nouveau was unblocked.
	StateNouveauUnblocked = "nouveau_unblocked"
	// StateNouveauRestored indicates nouveau was restored/loaded.
	StateNouveauRestored = "nouveau_restored"
)

// UninstallStatus represents the status of an uninstallation workflow.
type UninstallStatus int

const (
	// UninstallStatusPending indicates the uninstallation has not yet started.
	UninstallStatusPending UninstallStatus = iota
	// UninstallStatusRunning indicates the uninstallation is currently executing.
	UninstallStatusRunning
	// UninstallStatusCompleted indicates the uninstallation completed successfully.
	UninstallStatusCompleted
	// UninstallStatusPartial indicates some packages were removed, some failed.
	UninstallStatusPartial
	// UninstallStatusFailed indicates the uninstallation failed.
	UninstallStatusFailed
	// UninstallStatusCancelled indicates the uninstallation was cancelled.
	UninstallStatusCancelled
)

// String returns the string representation of the uninstall status.
func (s UninstallStatus) String() string {
	switch s {
	case UninstallStatusPending:
		return "pending"
	case UninstallStatusRunning:
		return "running"
	case UninstallStatusCompleted:
		return "completed"
	case UninstallStatusPartial:
		return "partial"
	case UninstallStatusFailed:
		return "failed"
	case UninstallStatusCancelled:
		return "cancelled"
	default:
		return fmt.Sprintf("unknown(%d)", int(s))
	}
}

// IsTerminal returns true if this status represents a terminal state.
func (s UninstallStatus) IsTerminal() bool {
	switch s {
	case UninstallStatusCompleted, UninstallStatusPartial, UninstallStatusFailed, UninstallStatusCancelled:
		return true
	default:
		return false
	}
}

// IsSuccess returns true if this status represents a successful outcome.
func (s UninstallStatus) IsSuccess() bool {
	return s == UninstallStatusCompleted
}

// IsPartial returns true if this status represents a partial uninstallation.
func (s UninstallStatus) IsPartial() bool {
	return s == UninstallStatusPartial
}

// UninstallResult contains the result of an uninstallation workflow.
type UninstallResult struct {
	// Status is the final status of the uninstallation.
	Status UninstallStatus

	// RemovedPackages lists packages that were successfully removed.
	RemovedPackages []string

	// FailedPackages lists packages that failed to remove.
	FailedPackages []string

	// CleanedConfigs lists configuration files that were removed.
	CleanedConfigs []string

	// CompletedSteps contains the names of successfully completed steps.
	CompletedSteps []string

	// FailedStep is the name of the step that failed, if any.
	FailedStep string

	// Error is the error that caused failure, if any.
	Error error

	// TotalDuration is the total time taken.
	TotalDuration time.Duration

	// NeedsReboot indicates whether a system reboot is required.
	NeedsReboot bool

	// NouveauRestored indicates if nouveau driver was restored.
	NouveauRestored bool
}

// NewUninstallResult creates a new uninstall result with the given status.
func NewUninstallResult(status UninstallStatus) UninstallResult {
	return UninstallResult{
		Status:          status,
		RemovedPackages: make([]string, 0),
		FailedPackages:  make([]string, 0),
		CleanedConfigs:  make([]string, 0),
		CompletedSteps:  make([]string, 0),
	}
}

// WithError adds an error and failed step to the uninstall result.
func (r UninstallResult) WithError(stepName string, err error) UninstallResult {
	r.FailedStep = stepName
	r.Error = err
	return r
}

// WithDuration adds a total duration to the uninstall result.
func (r UninstallResult) WithDuration(d time.Duration) UninstallResult {
	r.TotalDuration = d
	return r
}

// WithNeedsReboot sets whether a reboot is required.
func (r UninstallResult) WithNeedsReboot(needsReboot bool) UninstallResult {
	r.NeedsReboot = needsReboot
	return r
}

// WithNouveauRestored sets whether nouveau was restored.
func (r UninstallResult) WithNouveauRestored(restored bool) UninstallResult {
	r.NouveauRestored = restored
	return r
}

// AddCompletedStep adds a step name to the list of completed steps.
func (r *UninstallResult) AddCompletedStep(stepName string) {
	r.CompletedSteps = append(r.CompletedSteps, stepName)
}

// AddRemovedPackage adds a package name to the list of removed packages.
func (r *UninstallResult) AddRemovedPackage(pkg string) {
	r.RemovedPackages = append(r.RemovedPackages, pkg)
}

// AddFailedPackage adds a package name to the list of failed packages.
func (r *UninstallResult) AddFailedPackage(pkg string) {
	r.FailedPackages = append(r.FailedPackages, pkg)
}

// AddCleanedConfig adds a config path to the list of cleaned configs.
func (r *UninstallResult) AddCleanedConfig(path string) {
	r.CleanedConfigs = append(r.CleanedConfigs, path)
}

// IsSuccess returns true if the uninstallation completed successfully.
func (r UninstallResult) IsSuccess() bool {
	return r.Status.IsSuccess()
}

// IsFailure returns true if the uninstallation failed.
func (r UninstallResult) IsFailure() bool {
	return r.Status == UninstallStatusFailed
}

// IsPartial returns true if the uninstallation was partial.
func (r UninstallResult) IsPartial() bool {
	return r.Status == UninstallStatusPartial
}

// String returns a human-readable representation of the uninstall result.
func (r UninstallResult) String() string {
	if r.Error != nil {
		return fmt.Sprintf("%s: failed at step '%s' (error: %v)", r.Status, r.FailedStep, r.Error)
	}
	if r.Status == UninstallStatusPartial {
		return fmt.Sprintf("%s: removed %d packages, %d failed in %v",
			r.Status, len(r.RemovedPackages), len(r.FailedPackages), r.TotalDuration)
	}
	return fmt.Sprintf("%s: removed %d packages, cleaned %d configs in %v",
		r.Status, len(r.RemovedPackages), len(r.CleanedConfigs), r.TotalDuration)
}
