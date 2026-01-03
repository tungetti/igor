// Package dnf implements the pkg.Manager interface for DNF-based Linux distributions.
// It supports Fedora, RHEL 8+, Rocky Linux 8+, AlmaLinux 8+, and other RHEL derivatives.
package dnf

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

// Manager implements pkg.Manager for DNF-based distributions.
// It provides package management functionality using dnf and rpm commands.
type Manager struct {
	executor  igorexec.Executor
	privilege *privilege.Manager
}

// NewManager creates a new DNF package manager.
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
	return "dnf"
}

// Family returns the distribution family this manager supports.
func (m *Manager) Family() constants.DistroFamily {
	return constants.FamilyRHEL
}

// IsAvailable checks if dnf is available on the system.
func (m *Manager) IsAvailable() bool {
	_, err := exec.LookPath("dnf")
	return err == nil
}

// Install installs one or more packages using dnf install.
// Uses -y for non-interactive operation.
func (m *Manager) Install(ctx context.Context, opts pkg.InstallOptions, packages ...string) error {
	if len(packages) == 0 {
		return nil
	}

	args := m.buildInstallArgs(opts, packages)
	result := m.executor.ExecuteElevated(ctx, "dnf", args...)

	if result.Failed() {
		stderr := result.StderrString()
		// Check for common error patterns
		if strings.Contains(stderr, "No match for argument") ||
			strings.Contains(stderr, "No package") ||
			strings.Contains(stderr, "Nothing to do") {
			// Try to extract the package name
			for _, p := range packages {
				if strings.Contains(stderr, p) {
					return pkg.WrapWithPackage(pkg.ErrPackageNotFound, p, fmt.Errorf("dnf install failed: %s", stderr))
				}
			}
			return pkg.Wrap(pkg.ErrPackageNotFound, fmt.Errorf("dnf install failed: %s", stderr))
		}
		if strings.Contains(stderr, "lock") || strings.Contains(stderr, "another copy is running") {
			return pkg.Wrap(pkg.ErrLockAcquireFailed, fmt.Errorf("dnf install failed: %s", stderr))
		}
		return pkg.Wrap(pkg.ErrInstallFailed, fmt.Errorf("dnf install failed (exit code %d): %s", result.ExitCode, stderr))
	}

	return nil
}

// buildInstallArgs constructs the dnf install command arguments.
func (m *Manager) buildInstallArgs(opts pkg.InstallOptions, packages []string) []string {
	args := []string{"install", "-y"}

	// --allowerasing allows removing conflicting packages (covers both Force and AllowDowngrade)
	if opts.Force || opts.AllowDowngrade {
		args = append(args, "--allowerasing")
	}
	if opts.Reinstall {
		args = append(args, "--reinstall")
	}
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

	result := m.executor.ExecuteElevated(ctx, "dnf", args...)

	if result.Failed() {
		stderr := result.StderrString()
		stdout := result.StdoutString()
		combined := stderr + stdout

		// Check if package is not installed
		if strings.Contains(combined, "No match for argument") ||
			strings.Contains(combined, "No packages marked for removal") ||
			strings.Contains(combined, "not installed") {
			for _, p := range packages {
				if strings.Contains(combined, p) {
					return pkg.WrapWithPackage(pkg.ErrPackageNotInstalled, p, fmt.Errorf("dnf remove failed: %s", combined))
				}
			}
			return pkg.Wrap(pkg.ErrPackageNotInstalled, fmt.Errorf("dnf remove failed: %s", combined))
		}
		if strings.Contains(stderr, "lock") || strings.Contains(stderr, "another copy is running") {
			return pkg.Wrap(pkg.ErrLockAcquireFailed, fmt.Errorf("dnf remove failed: %s", stderr))
		}
		return pkg.Wrap(pkg.ErrRemoveFailed, fmt.Errorf("dnf remove failed (exit code %d): %s", result.ExitCode, stderr))
	}

	return nil
}

// Update updates the package database using dnf check-update.
// Note: dnf check-update returns exit code 100 when updates are available (not an error).
func (m *Manager) Update(ctx context.Context, opts pkg.UpdateOptions) error {
	args := []string{"check-update"}

	if opts.Quiet {
		args = append(args, "-q")
	}

	result := m.executor.ExecuteElevated(ctx, "dnf", args...)

	// dnf check-update returns:
	// 0 - no updates available
	// 100 - updates available (not an error!)
	// 1 - error
	if result.ExitCode == 0 || result.ExitCode == 100 {
		return nil
	}

	stderr := result.StderrString()
	if strings.Contains(stderr, "lock") || strings.Contains(stderr, "another copy is running") {
		return pkg.Wrap(pkg.ErrLockAcquireFailed, fmt.Errorf("dnf check-update failed: %s", stderr))
	}
	// Network errors
	if strings.Contains(stderr, "Could not resolve") ||
		strings.Contains(stderr, "Failed to download") ||
		strings.Contains(stderr, "Cannot download") {
		return pkg.Wrap(pkg.ErrNetworkUnavailable, fmt.Errorf("dnf check-update failed: %s", stderr))
	}
	return pkg.Wrap(pkg.ErrUpdateFailed, fmt.Errorf("dnf check-update failed (exit code %d): %s", result.ExitCode, stderr))
}

// Upgrade upgrades installed packages to their latest versions.
// If packages are specified, only those packages are upgraded.
// If no packages are specified, all upgradable packages are upgraded.
func (m *Manager) Upgrade(ctx context.Context, opts pkg.InstallOptions, packages ...string) error {
	var args []string

	if len(packages) == 0 {
		// Upgrade all packages
		args = []string{"upgrade", "-y"}
	} else {
		// Upgrade specific packages
		args = []string{"upgrade", "-y"}
		args = append(args, packages...)
	}

	// --allowerasing allows removing conflicting packages (covers both Force and AllowDowngrade)
	if opts.Force || opts.AllowDowngrade {
		args = append(args, "--allowerasing")
	}
	if opts.SkipVerify {
		args = append(args, "--nogpgcheck")
	}

	result := m.executor.ExecuteElevated(ctx, "dnf", args...)

	if result.Failed() {
		stderr := result.StderrString()
		if strings.Contains(stderr, "lock") || strings.Contains(stderr, "another copy is running") {
			return pkg.Wrap(pkg.ErrLockAcquireFailed, fmt.Errorf("dnf upgrade failed: %s", stderr))
		}
		return pkg.Wrap(pkg.ErrInstallFailed, fmt.Errorf("dnf upgrade failed (exit code %d): %s", result.ExitCode, stderr))
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
// Uses dnf search to find matching packages.
func (m *Manager) Search(ctx context.Context, query string, opts pkg.SearchOptions) ([]pkg.Package, error) {
	args := []string{"search"}

	if opts.ExactMatch {
		// For exact match, we use dnf list instead
		args = []string{"list", query}
	} else {
		args = append(args, query)
	}

	result := m.executor.Execute(ctx, "dnf", args...)

	if result.Failed() {
		stderr := result.StderrString()
		// "No matches found" is not an error, just empty results
		if strings.Contains(stderr, "No matches found") || strings.Contains(stderr, "No matching Packages") {
			return []pkg.Package{}, nil
		}
		return nil, fmt.Errorf("dnf search failed: %s", stderr)
	}

	var packages []pkg.Package
	var err error

	if opts.ExactMatch {
		packages, err = parseDnfList(result.StdoutString())
	} else {
		packages, err = parseDnfSearch(result.StdoutString())
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
// Uses dnf info to retrieve package metadata.
func (m *Manager) Info(ctx context.Context, pkgName string) (*pkg.Package, error) {
	result := m.executor.Execute(ctx, "dnf", "info", pkgName)

	if result.Failed() {
		stderr := result.StderrString()
		if strings.Contains(stderr, "No matching Packages") ||
			strings.Contains(stderr, "Error: No matching Packages") {
			return nil, pkg.WrapWithPackage(pkg.ErrPackageNotFound, pkgName, fmt.Errorf("package not found"))
		}
		return nil, fmt.Errorf("dnf info failed: %s", stderr)
	}

	p, err := parseDnfInfo(result.StdoutString())
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
// Uses dnf check-update to get the list.
func (m *Manager) ListUpgradable(ctx context.Context) ([]pkg.Package, error) {
	result := m.executor.Execute(ctx, "dnf", "check-update", "-q")

	// dnf check-update returns 100 if updates are available, 0 if none
	if result.ExitCode != 0 && result.ExitCode != 100 {
		return nil, fmt.Errorf("dnf check-update failed: %s", result.StderrString())
	}

	return parseDnfCheckUpdate(result.StdoutString())
}

// Clean removes cached package files to free disk space.
// Uses dnf clean all.
func (m *Manager) Clean(ctx context.Context) error {
	result := m.executor.ExecuteElevated(ctx, "dnf", "clean", "all")

	if result.Failed() {
		return fmt.Errorf("dnf clean failed: %s", result.StderrString())
	}

	return nil
}

// AutoRemove removes automatically installed packages that are no longer needed.
// Uses dnf autoremove -y.
func (m *Manager) AutoRemove(ctx context.Context) error {
	result := m.executor.ExecuteElevated(ctx, "dnf", "autoremove", "-y")

	if result.Failed() {
		stderr := result.StderrString()
		if strings.Contains(stderr, "lock") || strings.Contains(stderr, "another copy is running") {
			return pkg.Wrap(pkg.ErrLockAcquireFailed, fmt.Errorf("dnf autoremove failed: %s", stderr))
		}
		return fmt.Errorf("dnf autoremove failed: %s", stderr)
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
