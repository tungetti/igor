// Package zypper implements the pkg.Manager interface for Zypper-based Linux distributions.
// It supports openSUSE Leap, openSUSE Tumbleweed, and SUSE Linux Enterprise (SLES).
package zypper

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/tungetti/igor/internal/constants"
	igorexec "github.com/tungetti/igor/internal/exec"
	"github.com/tungetti/igor/internal/pkg"
	"github.com/tungetti/igor/internal/privilege"
)

// Manager implements pkg.Manager for Zypper-based distributions.
// It provides package management functionality using zypper and rpm commands.
type Manager struct {
	executor  igorexec.Executor
	privilege *privilege.Manager
}

// NewManager creates a new Zypper package manager.
// The executor is used for running shell commands, and privilege manager
// handles privilege elevation for operations requiring root access.
func NewManager(executor igorexec.Executor, priv *privilege.Manager) *Manager {
	return &Manager{
		executor:  executor,
		privilege: priv,
	}
}

// Name returns the package manager name.
func (m *Manager) Name() string {
	return "zypper"
}

// Family returns the distribution family this manager supports.
func (m *Manager) Family() constants.DistroFamily {
	return constants.FamilySUSE
}

// IsAvailable checks if zypper is available on the system.
func (m *Manager) IsAvailable() bool {
	_, err := exec.LookPath("zypper")
	return err == nil
}

// Install installs one or more packages using zypper install.
// Uses --non-interactive for unattended operation.
func (m *Manager) Install(ctx context.Context, opts pkg.InstallOptions, packages ...string) error {
	if len(packages) == 0 {
		return nil
	}

	args := m.buildInstallArgs(opts, packages)
	result := m.executor.ExecuteElevated(ctx, "zypper", args...)

	if result.Failed() {
		stderr := result.StderrString()
		stdout := result.StdoutString()
		combined := stderr + stdout

		// Check for common error patterns
		if strings.Contains(combined, "No provider of") ||
			strings.Contains(combined, "not found in package names") ||
			strings.Contains(combined, "package not found") ||
			strings.Contains(combined, "Nothing to do") {
			// Try to extract the package name
			for _, p := range packages {
				if strings.Contains(combined, p) {
					return pkg.WrapWithPackage(pkg.ErrPackageNotFound, p, fmt.Errorf("zypper install failed: %s", combined))
				}
			}
			return pkg.Wrap(pkg.ErrPackageNotFound, fmt.Errorf("zypper install failed: %s", combined))
		}
		if strings.Contains(stderr, "System management is locked") ||
			strings.Contains(stderr, "another zypper is running") {
			return pkg.Wrap(pkg.ErrLockAcquireFailed, fmt.Errorf("zypper install failed: %s", stderr))
		}
		return pkg.Wrap(pkg.ErrInstallFailed, fmt.Errorf("zypper install failed (exit code %d): %s", result.ExitCode, combined))
	}

	return nil
}

// buildInstallArgs constructs the zypper install command arguments.
func (m *Manager) buildInstallArgs(opts pkg.InstallOptions, packages []string) []string {
	args := []string{"--non-interactive", "install"}

	if opts.Force || opts.AllowDowngrade {
		args = append(args, "--force")
	}
	if opts.Reinstall {
		args = append(args, "--force")
	}
	if opts.DownloadOnly {
		args = append(args, "--download-only")
	}
	if opts.SkipVerify {
		args = append(args, "--no-gpg-checks")
	}

	args = append(args, packages...)
	return args
}

// Remove removes one or more packages from the system.
// Uses --non-interactive for unattended operation.
func (m *Manager) Remove(ctx context.Context, opts pkg.RemoveOptions, packages ...string) error {
	if len(packages) == 0 {
		return nil
	}

	args := []string{"--non-interactive", "remove"}

	if opts.Purge {
		args = append(args, "--clean-deps")
	}

	args = append(args, packages...)

	result := m.executor.ExecuteElevated(ctx, "zypper", args...)

	if result.Failed() {
		stderr := result.StderrString()
		stdout := result.StdoutString()
		combined := stderr + stdout

		// Check if package is not installed
		if strings.Contains(combined, "not installed") ||
			strings.Contains(combined, "not found") ||
			strings.Contains(combined, "No packages to remove") {
			for _, p := range packages {
				if strings.Contains(combined, p) {
					return pkg.WrapWithPackage(pkg.ErrPackageNotInstalled, p, fmt.Errorf("zypper remove failed: %s", combined))
				}
			}
			return pkg.Wrap(pkg.ErrPackageNotInstalled, fmt.Errorf("zypper remove failed: %s", combined))
		}
		if strings.Contains(stderr, "System management is locked") ||
			strings.Contains(stderr, "another zypper is running") {
			return pkg.Wrap(pkg.ErrLockAcquireFailed, fmt.Errorf("zypper remove failed: %s", stderr))
		}
		return pkg.Wrap(pkg.ErrRemoveFailed, fmt.Errorf("zypper remove failed (exit code %d): %s", result.ExitCode, combined))
	}

	return nil
}

// Update refreshes the package database using zypper refresh.
func (m *Manager) Update(ctx context.Context, opts pkg.UpdateOptions) error {
	args := []string{"--non-interactive", "refresh"}

	if opts.ForceRefresh {
		args = append(args, "--force")
	}

	result := m.executor.ExecuteElevated(ctx, "zypper", args...)

	if result.Failed() {
		stderr := result.StderrString()
		if strings.Contains(stderr, "System management is locked") ||
			strings.Contains(stderr, "another zypper is running") {
			return pkg.Wrap(pkg.ErrLockAcquireFailed, fmt.Errorf("zypper refresh failed: %s", stderr))
		}
		// Network errors
		if strings.Contains(stderr, "Could not resolve") ||
			strings.Contains(stderr, "Timeout") ||
			strings.Contains(stderr, "Download") ||
			strings.Contains(stderr, "Network") {
			return pkg.Wrap(pkg.ErrNetworkUnavailable, fmt.Errorf("zypper refresh failed: %s", stderr))
		}
		return pkg.Wrap(pkg.ErrUpdateFailed, fmt.Errorf("zypper refresh failed (exit code %d): %s", result.ExitCode, stderr))
	}

	return nil
}

// Upgrade upgrades installed packages to their latest versions.
// If packages are specified, only those packages are upgraded using zypper update.
// If no packages are specified, a full distribution upgrade is performed using zypper dist-upgrade.
func (m *Manager) Upgrade(ctx context.Context, opts pkg.InstallOptions, packages ...string) error {
	var args []string

	if len(packages) == 0 {
		// Full system upgrade using dist-upgrade
		args = []string{"--non-interactive", "dist-upgrade"}
	} else {
		// Upgrade specific packages using update
		args = []string{"--non-interactive", "update"}
	}

	// Add options before packages
	if opts.Force || opts.AllowDowngrade {
		args = append(args, "--force")
	}
	if opts.SkipVerify {
		args = append(args, "--no-gpg-checks")
	}

	// Now add packages
	if len(packages) > 0 {
		args = append(args, packages...)
	}

	result := m.executor.ExecuteElevated(ctx, "zypper", args...)

	if result.Failed() {
		stderr := result.StderrString()
		if strings.Contains(stderr, "System management is locked") ||
			strings.Contains(stderr, "another zypper is running") {
			return pkg.Wrap(pkg.ErrLockAcquireFailed, fmt.Errorf("zypper upgrade failed: %s", stderr))
		}
		return pkg.Wrap(pkg.ErrInstallFailed, fmt.Errorf("zypper upgrade failed (exit code %d): %s", result.ExitCode, stderr))
	}

	return nil
}

// IsInstalled checks if a package is currently installed.
// Uses rpm -q to check the package status.
func (m *Manager) IsInstalled(ctx context.Context, pkgName string) (bool, error) {
	result := m.executor.Execute(ctx, "rpm", "-q", pkgName)

	if result.Failed() {
		// rpm -q returns exit code 1 if package is not installed
		if result.ExitCode == 1 {
			return false, nil
		}
		return false, fmt.Errorf("rpm -q failed: %s", result.StderrString())
	}

	return true, nil
}

// Search searches for packages matching the query.
// Uses zypper search to find matching packages.
func (m *Manager) Search(ctx context.Context, query string, opts pkg.SearchOptions) ([]pkg.Package, error) {
	args := []string{"search"}

	if opts.ExactMatch {
		args = append(args, "--match-exact")
	}

	args = append(args, query)

	result := m.executor.Execute(ctx, "zypper", args...)

	if result.Failed() {
		stderr := result.StderrString()
		stdout := result.StdoutString()
		// "No matching items found" is not an error, just empty results
		if strings.Contains(stderr, "No matching items") ||
			strings.Contains(stdout, "No matching items") ||
			result.ExitCode == 104 { // Zypper exit code for no results
			return []pkg.Package{}, nil
		}
		return nil, fmt.Errorf("zypper search failed: %s", stderr)
	}

	packages, err := parseZypperSearch(result.StdoutString())
	if err != nil {
		return nil, err
	}

	// Apply limit if specified
	if opts.Limit > 0 && len(packages) > opts.Limit {
		packages = packages[:opts.Limit]
	}

	// Check installed status if requested
	if opts.IncludeInstalled {
		for i := range packages {
			installed, _ := m.IsInstalled(ctx, packages[i].Name)
			packages[i].Installed = installed
		}
	}

	return packages, nil
}

// Info returns detailed information about a specific package.
// Uses zypper info to retrieve package metadata.
func (m *Manager) Info(ctx context.Context, pkgName string) (*pkg.Package, error) {
	result := m.executor.Execute(ctx, "zypper", "info", pkgName)

	if result.Failed() {
		stderr := result.StderrString()
		stdout := result.StdoutString()
		combined := stderr + stdout
		if strings.Contains(combined, "not found") ||
			strings.Contains(combined, "No matching items") {
			return nil, pkg.WrapWithPackage(pkg.ErrPackageNotFound, pkgName, fmt.Errorf("package not found"))
		}
		return nil, fmt.Errorf("zypper info failed: %s", stderr)
	}

	p, err := parseZypperInfo(result.StdoutString())
	if err != nil {
		return nil, err
	}

	if p == nil || p.Name == "" {
		return nil, pkg.WrapWithPackage(pkg.ErrPackageNotFound, pkgName, fmt.Errorf("package not found"))
	}

	// Check if installed
	installed, _ := m.IsInstalled(ctx, pkgName)
	p.Installed = installed

	return p, nil
}

// ListInstalled returns a list of all installed packages.
// Uses rpm -qa with custom query format for consistent output.
func (m *Manager) ListInstalled(ctx context.Context) ([]pkg.Package, error) {
	result := m.executor.Execute(ctx, "rpm", "-qa", "--queryformat", "%{NAME}\t%{VERSION}-%{RELEASE}\t%{ARCH}\n")

	if result.Failed() {
		return nil, fmt.Errorf("rpm -qa failed: %s", result.StderrString())
	}

	return parseRpmQuery(result.StdoutString())
}

// ListUpgradable returns a list of packages that can be upgraded.
// Uses zypper list-updates to get the list.
func (m *Manager) ListUpgradable(ctx context.Context) ([]pkg.Package, error) {
	result := m.executor.Execute(ctx, "zypper", "list-updates")

	if result.Failed() {
		stderr := result.StderrString()
		// No updates available is not an error
		if strings.Contains(stderr, "No updates found") ||
			result.ExitCode == 0 {
			return []pkg.Package{}, nil
		}
		return nil, fmt.Errorf("zypper list-updates failed: %s", stderr)
	}

	return parseZypperListUpdates(result.StdoutString())
}

// Clean removes cached package files to free disk space.
// Uses zypper clean.
func (m *Manager) Clean(ctx context.Context) error {
	result := m.executor.ExecuteElevated(ctx, "zypper", "clean", "--all")

	if result.Failed() {
		return fmt.Errorf("zypper clean failed: %s", result.StderrString())
	}

	return nil
}

// AutoRemove removes automatically installed packages that are no longer needed.
// Zypper doesn't have a direct autoremove command, so we use packages --unneeded
// to list unneeded packages and then remove them.
func (m *Manager) AutoRemove(ctx context.Context) error {
	// First, list unneeded packages
	result := m.executor.Execute(ctx, "zypper", "packages", "--unneeded")

	if result.Failed() {
		stderr := result.StderrString()
		if strings.Contains(stderr, "System management is locked") ||
			strings.Contains(stderr, "another zypper is running") {
			return pkg.Wrap(pkg.ErrLockAcquireFailed, fmt.Errorf("zypper packages failed: %s", stderr))
		}
		// No unneeded packages is not an error
		if strings.Contains(stderr, "No packages found") ||
			strings.Contains(result.StdoutString(), "No packages found") {
			return nil
		}
		return fmt.Errorf("zypper packages --unneeded failed: %s", stderr)
	}

	// Parse the output to get package names
	packages := parseUnneededPackages(result.StdoutString())

	if len(packages) == 0 {
		return nil
	}

	// Remove the unneeded packages
	args := []string{"--non-interactive", "remove"}
	args = append(args, packages...)

	result = m.executor.ExecuteElevated(ctx, "zypper", args...)

	if result.Failed() {
		stderr := result.StderrString()
		if strings.Contains(stderr, "System management is locked") ||
			strings.Contains(stderr, "another zypper is running") {
			return pkg.Wrap(pkg.ErrLockAcquireFailed, fmt.Errorf("zypper autoremove failed: %s", stderr))
		}
		return fmt.Errorf("zypper autoremove failed: %s", stderr)
	}

	return nil
}

// Verify checks package integrity using rpm -V.
// Returns true if the package passes verification, false otherwise.
func (m *Manager) Verify(ctx context.Context, pkgName string) (bool, error) {
	// First check if package is installed
	installed, err := m.IsInstalled(ctx, pkgName)
	if err != nil {
		return false, err
	}
	if !installed {
		return false, pkg.WrapWithPackage(pkg.ErrPackageNotInstalled, pkgName, fmt.Errorf("package not installed"))
	}

	// Use rpm -V for verification
	result := m.executor.Execute(ctx, "rpm", "-V", pkgName)
	// rpm -V returns 0 if package is OK, non-zero if files are missing/changed
	return result.ExitCode == 0, nil
}

// Ensure Manager implements pkg.Manager interface.
var _ pkg.Manager = (*Manager)(nil)
