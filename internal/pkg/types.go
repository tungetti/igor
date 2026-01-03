// Package pkg provides the package manager interface and types for Igor.
// It abstracts package operations across different Linux package managers
// (APT, DNF, YUM, Pacman, Zypper) and provides a unified API for package
// installation, removal, and repository management.
package pkg

import (
	"fmt"
)

// Package represents a software package with its metadata.
// This struct contains all information needed to identify and describe
// a package across different package managers.
type Package struct {
	// Name is the unique identifier for the package (e.g., "nvidia-driver-535").
	Name string

	// Version is the package version string (e.g., "535.154.05-1").
	Version string

	// Installed indicates whether the package is currently installed on the system.
	Installed bool

	// Repository is the source repository for the package (e.g., "nvidia-driver", "main").
	Repository string

	// Description is a human-readable description of the package.
	Description string

	// Size is the package size in bytes. For installed packages, this is typically
	// the installed size; for available packages, it may be the download size.
	Size int64

	// Architecture is the package architecture (e.g., "amd64", "x86_64", "all").
	Architecture string

	// Dependencies lists the packages this package depends on.
	Dependencies []string
}

// String returns a human-readable representation of the package.
func (p Package) String() string {
	status := "not installed"
	if p.Installed {
		status = "installed"
	}
	if p.Version != "" {
		return fmt.Sprintf("%s (%s) [%s]", p.Name, p.Version, status)
	}
	return fmt.Sprintf("%s [%s]", p.Name, status)
}

// FullName returns the package name with version if available.
func (p Package) FullName() string {
	if p.Version != "" {
		return fmt.Sprintf("%s-%s", p.Name, p.Version)
	}
	return p.Name
}

// IsEmpty returns true if the package has no name set.
func (p Package) IsEmpty() bool {
	return p.Name == ""
}

// Repository represents a package repository with its configuration.
// Different package managers have different repository formats, but this
// struct provides a unified representation.
type Repository struct {
	// Name is the unique identifier for the repository.
	Name string

	// URL is the repository URL or baseurl.
	URL string

	// Enabled indicates whether the repository is currently enabled.
	Enabled bool

	// GPGKey is the URL or path to the GPG key for package verification.
	GPGKey string

	// Type is the repository type (e.g., "deb", "rpm", "pkg").
	Type string

	// Components are repository components (e.g., "main", "contrib" for Debian).
	Components []string

	// Distribution is the distribution codename (e.g., "jammy", "bookworm" for Debian).
	Distribution string

	// Priority is the repository priority (lower = higher priority for most managers).
	Priority int
}

// String returns a human-readable representation of the repository.
func (r Repository) String() string {
	status := "disabled"
	if r.Enabled {
		status = "enabled"
	}
	return fmt.Sprintf("%s (%s) [%s]", r.Name, r.URL, status)
}

// IsEmpty returns true if the repository has no name set.
func (r Repository) IsEmpty() bool {
	return r.Name == ""
}

// HasGPGKey returns true if the repository has a GPG key configured.
func (r Repository) HasGPGKey() bool {
	return r.GPGKey != ""
}

// InstallOptions configures package installation behavior.
type InstallOptions struct {
	// Force forces installation even if the package is already installed
	// or if there are dependency issues.
	Force bool

	// NoConfirm suppresses interactive confirmation prompts.
	NoConfirm bool

	// SkipVerify skips GPG signature verification (use with caution).
	SkipVerify bool

	// DownloadOnly downloads the package without installing it.
	DownloadOnly bool

	// Reinstall reinstalls the package even if already installed.
	Reinstall bool

	// AllowDowngrade allows installing an older version of the package.
	AllowDowngrade bool
}

// DefaultInstallOptions returns the default installation options.
// By default, installation is interactive with signature verification enabled.
func DefaultInstallOptions() InstallOptions {
	return InstallOptions{
		Force:          false,
		NoConfirm:      false,
		SkipVerify:     false,
		DownloadOnly:   false,
		Reinstall:      false,
		AllowDowngrade: false,
	}
}

// NonInteractiveInstallOptions returns options suitable for automated installations.
func NonInteractiveInstallOptions() InstallOptions {
	return InstallOptions{
		Force:          false,
		NoConfirm:      true,
		SkipVerify:     false,
		DownloadOnly:   false,
		Reinstall:      false,
		AllowDowngrade: false,
	}
}

// UpdateOptions configures repository update behavior.
type UpdateOptions struct {
	// Quiet suppresses progress output during update.
	Quiet bool

	// ForceRefresh forces a refresh even if the cache is fresh.
	ForceRefresh bool
}

// DefaultUpdateOptions returns the default update options.
func DefaultUpdateOptions() UpdateOptions {
	return UpdateOptions{
		Quiet:        false,
		ForceRefresh: false,
	}
}

// RemoveOptions configures package removal behavior.
type RemoveOptions struct {
	// Purge removes configuration files along with the package.
	Purge bool

	// AutoRemove removes automatically installed dependencies that are no longer needed.
	AutoRemove bool

	// NoConfirm suppresses interactive confirmation prompts.
	NoConfirm bool
}

// DefaultRemoveOptions returns the default removal options.
func DefaultRemoveOptions() RemoveOptions {
	return RemoveOptions{
		Purge:      false,
		AutoRemove: false,
		NoConfirm:  false,
	}
}

// SearchOptions configures package search behavior.
type SearchOptions struct {
	// IncludeInstalled includes installed packages in search results.
	IncludeInstalled bool

	// ExactMatch requires exact name match instead of partial matching.
	ExactMatch bool

	// Limit is the maximum number of results to return (0 = no limit).
	Limit int
}

// DefaultSearchOptions returns the default search options.
func DefaultSearchOptions() SearchOptions {
	return SearchOptions{
		IncludeInstalled: true,
		ExactMatch:       false,
		Limit:            0,
	}
}
