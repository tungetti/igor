// Package yum implements the pkg.Manager interface for YUM-based Linux distributions.
// It supports CentOS 7, RHEL 7, and other older Enterprise Linux systems that use
// YUM (Yellowdog Updater Modified) instead of DNF.
//
// YUM is the predecessor to DNF and is commonly found on:
// - CentOS 7
// - RHEL 7
// - Oracle Linux 7
// - Scientific Linux 7
// - Amazon Linux (pre-2023)
//
// Key differences from DNF:
// - Uses "yum update" instead of "dnf upgrade"
// - Uses "yum-config-manager" instead of "dnf config-manager"
// - Does not support --allowerasing flag
// - EPEL repository is critical for many packages
package yum

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

// Manager implements pkg.Manager for YUM-based distributions.
// It provides package management functionality using yum and rpm commands.
type Manager struct {
	executor  igorexec.Executor
	privilege *privilege.Manager
}

// NewManager creates a new YUM package manager.
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
	return "yum"
}

// Family returns the distribution family this manager supports.
func (m *Manager) Family() constants.DistroFamily {
	return constants.FamilyRHEL
}

// IsAvailable checks if yum is available on the system.
func (m *Manager) IsAvailable() bool {
	_, err := exec.LookPath("yum")
	return err == nil
}

// Install installs one or more packages using yum install.
// Uses -y for non-interactive operation.
func (m *Manager) Install(ctx context.Context, opts pkg.InstallOptions, packages ...string) error {
	if len(packages) == 0 {
		return nil
	}

	args := m.buildInstallArgs(opts, packages)
	result := m.executor.ExecuteElevated(ctx, "yum", args...)

	if result.Failed() {
		stderr := result.StderrString()
		stdout := result.StdoutString()
		combined := stderr + stdout

		// Check for common error patterns
		if strings.Contains(combined, "No package") ||
			strings.Contains(combined, "No match for argument") ||
			strings.Contains(combined, "Nothing to do") {
			// Try to extract the package name
			for _, p := range packages {
				if strings.Contains(combined, p) {
					return pkg.WrapWithPackage(pkg.ErrPackageNotFound, p, fmt.Errorf("yum install failed: %s", stderr))
				}
			}
			return pkg.Wrap(pkg.ErrPackageNotFound, fmt.Errorf("yum install failed: %s", stderr))
		}
		if strings.Contains(stderr, "lock") || strings.Contains(stderr, "Another app is currently holding the yum lock") {
			return pkg.Wrap(pkg.ErrLockAcquireFailed, fmt.Errorf("yum install failed: %s", stderr))
		}
		return pkg.Wrap(pkg.ErrInstallFailed, fmt.Errorf("yum install failed (exit code %d): %s", result.ExitCode, stderr))
	}

	return nil
}

// buildInstallArgs constructs the yum install command arguments.
// Note: YUM doesn't support --allowerasing like DNF.
func (m *Manager) buildInstallArgs(opts pkg.InstallOptions, packages []string) []string {
	args := []string{"install", "-y"}

	// YUM doesn't have --allowerasing, but we can use --skip-broken for Force
	if opts.Force {
		args = append(args, "--skip-broken")
	}
	// YUM doesn't have --reinstall flag like DNF, use reinstall command instead
	// For now, we proceed with install which will handle reinstalls
	if opts.DownloadOnly {
		args = append(args, "--downloadonly")
	}
	if opts.SkipVerify {
		args = append(args, "--nogpgcheck")
	}

	args = append(args, packages...)
	return args
}

// Remove removes one or more packages from the system.
// Uses -y for non-interactive operation.
func (m *Manager) Remove(ctx context.Context, opts pkg.RemoveOptions, packages ...string) error {
	if len(packages) == 0 {
		return nil
	}

	args := []string{"remove", "-y"}
	args = append(args, packages...)

	result := m.executor.ExecuteElevated(ctx, "yum", args...)

	if result.Failed() {
		stderr := result.StderrString()
		stdout := result.StdoutString()
		combined := stderr + stdout

		// Check if package is not installed
		if strings.Contains(combined, "No Match for argument") ||
			strings.Contains(combined, "No Packages marked for removal") ||
			strings.Contains(combined, "not installed") ||
			strings.Contains(combined, "No package") {
			for _, p := range packages {
				if strings.Contains(combined, p) {
					return pkg.WrapWithPackage(pkg.ErrPackageNotInstalled, p, fmt.Errorf("yum remove failed: %s", combined))
				}
			}
			return pkg.Wrap(pkg.ErrPackageNotInstalled, fmt.Errorf("yum remove failed: %s", combined))
		}
		if strings.Contains(stderr, "lock") || strings.Contains(stderr, "Another app is currently holding the yum lock") {
			return pkg.Wrap(pkg.ErrLockAcquireFailed, fmt.Errorf("yum remove failed: %s", stderr))
		}
		return pkg.Wrap(pkg.ErrRemoveFailed, fmt.Errorf("yum remove failed (exit code %d): %s", result.ExitCode, stderr))
	}

	return nil
}

// Update updates the package database using yum check-update.
// Note: yum check-update returns exit code 100 when updates are available (not an error).
func (m *Manager) Update(ctx context.Context, opts pkg.UpdateOptions) error {
	args := []string{"check-update"}

	if opts.Quiet {
		args = append(args, "-q")
	}

	result := m.executor.ExecuteElevated(ctx, "yum", args...)

	// yum check-update returns:
	// 0 - no updates available
	// 100 - updates available (not an error!)
	// 1 - error
	if result.ExitCode == 0 || result.ExitCode == 100 {
		return nil
	}

	stderr := result.StderrString()
	if strings.Contains(stderr, "lock") || strings.Contains(stderr, "Another app is currently holding the yum lock") {
		return pkg.Wrap(pkg.ErrLockAcquireFailed, fmt.Errorf("yum check-update failed: %s", stderr))
	}
	// Network errors
	if strings.Contains(stderr, "Could not resolve") ||
		strings.Contains(stderr, "Cannot retrieve") ||
		strings.Contains(stderr, "Cannot find a valid baseurl") {
		return pkg.Wrap(pkg.ErrNetworkUnavailable, fmt.Errorf("yum check-update failed: %s", stderr))
	}
	return pkg.Wrap(pkg.ErrUpdateFailed, fmt.Errorf("yum check-update failed (exit code %d): %s", result.ExitCode, stderr))
}

// Upgrade upgrades installed packages to their latest versions.
// If packages are specified, only those packages are upgraded.
// If no packages are specified, all upgradable packages are upgraded.
// Note: YUM uses "update" instead of "upgrade" (both work but update is traditional).
func (m *Manager) Upgrade(ctx context.Context, opts pkg.InstallOptions, packages ...string) error {
	var args []string

	if len(packages) == 0 {
		// Upgrade all packages - YUM uses "update" instead of "upgrade"
		args = []string{"update", "-y"}
	} else {
		// Upgrade specific packages
		args = []string{"update", "-y"}
		args = append(args, packages...)
	}

	// YUM doesn't support --allowerasing
	if opts.SkipVerify {
		args = append(args, "--nogpgcheck")
	}

	result := m.executor.ExecuteElevated(ctx, "yum", args...)

	if result.Failed() {
		stderr := result.StderrString()
		if strings.Contains(stderr, "lock") || strings.Contains(stderr, "Another app is currently holding the yum lock") {
			return pkg.Wrap(pkg.ErrLockAcquireFailed, fmt.Errorf("yum update failed: %s", stderr))
		}
		return pkg.Wrap(pkg.ErrInstallFailed, fmt.Errorf("yum update failed (exit code %d): %s", result.ExitCode, stderr))
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
// Uses yum search to find matching packages.
func (m *Manager) Search(ctx context.Context, query string, opts pkg.SearchOptions) ([]pkg.Package, error) {
	args := []string{"search"}

	if opts.ExactMatch {
		// For exact match, we use yum list instead
		args = []string{"list", query}
	} else {
		args = append(args, query)
	}

	result := m.executor.Execute(ctx, "yum", args...)

	if result.Failed() {
		stderr := result.StderrString()
		stdout := result.StdoutString()
		combined := stderr + stdout
		// "No matches found" or "No matching Packages" is not an error, just empty results
		if strings.Contains(combined, "No matches found") ||
			strings.Contains(combined, "No matching Packages") ||
			strings.Contains(combined, "Warning: No matches found") {
			return []pkg.Package{}, nil
		}
		return nil, fmt.Errorf("yum search failed: %s", stderr)
	}

	var packages []pkg.Package
	var err error

	if opts.ExactMatch {
		packages, err = parseYumList(result.StdoutString())
	} else {
		packages, err = parseYumSearch(result.StdoutString())
	}

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
// Uses yum info to retrieve package metadata.
func (m *Manager) Info(ctx context.Context, pkgName string) (*pkg.Package, error) {
	result := m.executor.Execute(ctx, "yum", "info", pkgName)

	if result.Failed() {
		stderr := result.StderrString()
		stdout := result.StdoutString()
		combined := stderr + stdout
		if strings.Contains(combined, "No matching Packages") ||
			strings.Contains(combined, "Error: No matching Packages") ||
			strings.Contains(combined, "No package") {
			return nil, pkg.WrapWithPackage(pkg.ErrPackageNotFound, pkgName, fmt.Errorf("package not found"))
		}
		return nil, fmt.Errorf("yum info failed: %s", stderr)
	}

	p, err := parseYumInfo(result.StdoutString())
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
// Uses rpm -qa with custom query format.
func (m *Manager) ListInstalled(ctx context.Context) ([]pkg.Package, error) {
	result := m.executor.Execute(ctx, "rpm", "-qa", "--queryformat", "%{NAME}\t%{VERSION}-%{RELEASE}\t%{ARCH}\n")

	if result.Failed() {
		return nil, fmt.Errorf("rpm -qa failed: %s", result.StderrString())
	}

	return parseRpmQuery(result.StdoutString())
}

// ListUpgradable returns a list of packages that can be upgraded.
// Uses yum check-update to get the list.
func (m *Manager) ListUpgradable(ctx context.Context) ([]pkg.Package, error) {
	result := m.executor.Execute(ctx, "yum", "check-update", "-q")

	// yum check-update returns 100 if updates are available, 0 if none
	if result.ExitCode != 0 && result.ExitCode != 100 {
		return nil, fmt.Errorf("yum check-update failed: %s", result.StderrString())
	}

	return parseYumCheckUpdate(result.StdoutString())
}

// Clean removes cached package files to free disk space.
// Uses yum clean all.
func (m *Manager) Clean(ctx context.Context) error {
	result := m.executor.ExecuteElevated(ctx, "yum", "clean", "all")

	if result.Failed() {
		return fmt.Errorf("yum clean failed: %s", result.StderrString())
	}

	return nil
}

// AutoRemove removes automatically installed packages that are no longer needed.
// Uses yum autoremove -y. For older YUM versions, this may not be available.
// Falls back to package-cleanup --leaves if autoremove is not supported.
func (m *Manager) AutoRemove(ctx context.Context) error {
	result := m.executor.ExecuteElevated(ctx, "yum", "autoremove", "-y")

	if result.Failed() {
		stderr := result.StderrString()

		// Check if autoremove is not supported (older YUM versions)
		if strings.Contains(stderr, "No such command") ||
			strings.Contains(stderr, "autoremove") {
			// Try package-cleanup as fallback
			return m.autoRemoveFallback(ctx)
		}

		if strings.Contains(stderr, "lock") || strings.Contains(stderr, "Another app is currently holding the yum lock") {
			return pkg.Wrap(pkg.ErrLockAcquireFailed, fmt.Errorf("yum autoremove failed: %s", stderr))
		}
		return fmt.Errorf("yum autoremove failed: %s", stderr)
	}

	return nil
}

// autoRemoveFallback uses package-cleanup for older YUM versions that don't support autoremove.
func (m *Manager) autoRemoveFallback(ctx context.Context) error {
	// package-cleanup is part of yum-utils
	result := m.executor.ExecuteElevated(ctx, "package-cleanup", "--leaves", "-y")

	if result.Failed() {
		// If package-cleanup is not available, just return without error
		// as autoremove is optional functionality
		return nil
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

// AddEPEL enables the EPEL (Extra Packages for Enterprise Linux) repository.
// EPEL is critical for many packages on CentOS 7/RHEL 7.
// For CentOS, it installs epel-release package.
// For RHEL, it downloads the release RPM directly from the Fedora project.
func (m *Manager) AddEPEL(ctx context.Context) error {
	// First try installing epel-release package (works on CentOS)
	result := m.executor.ExecuteElevated(ctx, "yum", "install", "-y", "epel-release")

	if result.ExitCode == 0 {
		return nil
	}

	// If that fails, try direct URL for RHEL 7
	result = m.executor.ExecuteElevated(ctx, "yum", "install", "-y",
		"https://dl.fedoraproject.org/pub/epel/epel-release-latest-7.noarch.rpm")

	if result.Failed() {
		stderr := result.StderrString()
		stdout := result.StdoutString()
		// Check if already installed
		if strings.Contains(stderr, "already installed") ||
			strings.Contains(stdout, "already installed") {
			return nil
		}
		return fmt.Errorf("failed to add EPEL repository: %s", stderr)
	}

	return nil
}

// Ensure Manager implements pkg.Manager interface.
var _ pkg.Manager = (*Manager)(nil)
