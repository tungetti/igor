// Package apt implements the pkg.Manager interface for APT-based Linux distributions.
// It supports Debian, Ubuntu, Linux Mint, Pop!_OS, and other Debian derivatives.
package apt

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

// Environment variables for non-interactive APT operations.
var nonInteractiveEnv = []string{
	"DEBIAN_FRONTEND=noninteractive",
	"DEBCONF_NONINTERACTIVE_SEEN=true",
}

// Manager implements pkg.Manager for APT-based distributions.
// It provides package management functionality using apt-get, apt-cache,
// and dpkg-query commands.
type Manager struct {
	executor  igorexec.Executor
	privilege *privilege.Manager
}

// NewManager creates a new APT package manager.
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
	return "apt"
}

// Family returns the distribution family this manager supports.
func (m *Manager) Family() constants.DistroFamily {
	return constants.FamilyDebian
}

// IsAvailable checks if apt-get is available on the system.
func (m *Manager) IsAvailable() bool {
	_, err := exec.LookPath("apt-get")
	return err == nil
}

// Install installs one or more packages using apt-get install.
// Uses DEBIAN_FRONTEND=noninteractive for unattended operation.
func (m *Manager) Install(ctx context.Context, opts pkg.InstallOptions, packages ...string) error {
	if len(packages) == 0 {
		return nil
	}

	args := m.buildInstallArgs(opts, packages)
	result := m.executeElevatedWithEnv(ctx, "apt-get", args...)

	if result.Failed() {
		// Check for common error patterns
		stderr := result.StderrString()
		if strings.Contains(stderr, "Unable to locate package") {
			// Try to extract the package name
			for _, p := range packages {
				if strings.Contains(stderr, p) {
					return pkg.WrapWithPackage(pkg.ErrPackageNotFound, p, fmt.Errorf("apt-get install failed: %s", stderr))
				}
			}
			return pkg.Wrap(pkg.ErrPackageNotFound, fmt.Errorf("apt-get install failed: %s", stderr))
		}
		if strings.Contains(stderr, "dpkg was interrupted") || strings.Contains(stderr, "Could not get lock") {
			return pkg.Wrap(pkg.ErrLockAcquireFailed, fmt.Errorf("apt-get install failed: %s", stderr))
		}
		return pkg.Wrap(pkg.ErrInstallFailed, fmt.Errorf("apt-get install failed (exit code %d): %s", result.ExitCode, stderr))
	}

	return nil
}

// buildInstallArgs constructs the apt-get install command arguments.
func (m *Manager) buildInstallArgs(opts pkg.InstallOptions, packages []string) []string {
	args := []string{"install", "-y"}

	if opts.Force {
		args = append(args, "--allow-unauthenticated", "--allow-downgrades", "--allow-change-held-packages")
	}
	if opts.Reinstall {
		args = append(args, "--reinstall")
	}
	if opts.DownloadOnly {
		args = append(args, "--download-only")
	}
	if opts.AllowDowngrade {
		args = append(args, "--allow-downgrades")
	}
	if opts.SkipVerify {
		args = append(args, "--allow-unauthenticated")
	}

	args = append(args, packages...)
	return args
}

// Remove removes one or more packages from the system.
// If opts.Purge is true, also removes configuration files.
func (m *Manager) Remove(ctx context.Context, opts pkg.RemoveOptions, packages ...string) error {
	if len(packages) == 0 {
		return nil
	}

	var cmd string
	if opts.Purge {
		cmd = "purge"
	} else {
		cmd = "remove"
	}

	args := []string{cmd, "-y"}
	if opts.AutoRemove {
		args = append(args, "--auto-remove")
	}
	args = append(args, packages...)

	result := m.executeElevatedWithEnv(ctx, "apt-get", args...)

	if result.Failed() {
		stderr := result.StderrString()
		// Check if package is not installed
		if strings.Contains(stderr, "is not installed") {
			for _, p := range packages {
				if strings.Contains(stderr, p) {
					return pkg.WrapWithPackage(pkg.ErrPackageNotInstalled, p, fmt.Errorf("apt-get %s failed: %s", cmd, stderr))
				}
			}
			return pkg.Wrap(pkg.ErrPackageNotInstalled, fmt.Errorf("apt-get %s failed: %s", cmd, stderr))
		}
		if strings.Contains(stderr, "Could not get lock") {
			return pkg.Wrap(pkg.ErrLockAcquireFailed, fmt.Errorf("apt-get %s failed: %s", cmd, stderr))
		}
		return pkg.Wrap(pkg.ErrRemoveFailed, fmt.Errorf("apt-get %s failed (exit code %d): %s", cmd, result.ExitCode, stderr))
	}

	return nil
}

// Update updates the package database using apt-get update.
func (m *Manager) Update(ctx context.Context, opts pkg.UpdateOptions) error {
	args := []string{"update"}

	if opts.Quiet {
		args = append(args, "-qq")
	}

	result := m.executeElevatedWithEnv(ctx, "apt-get", args...)

	if result.Failed() {
		stderr := result.StderrString()
		if strings.Contains(stderr, "Could not get lock") {
			return pkg.Wrap(pkg.ErrLockAcquireFailed, fmt.Errorf("apt-get update failed: %s", stderr))
		}
		// Network errors
		if strings.Contains(stderr, "Could not resolve") || strings.Contains(stderr, "Failed to fetch") {
			return pkg.Wrap(pkg.ErrNetworkUnavailable, fmt.Errorf("apt-get update failed: %s", stderr))
		}
		return pkg.Wrap(pkg.ErrUpdateFailed, fmt.Errorf("apt-get update failed (exit code %d): %s", result.ExitCode, stderr))
	}

	return nil
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
		// Upgrade specific packages - use install with --only-upgrade
		args = []string{"install", "-y", "--only-upgrade"}
		args = append(args, packages...)
	}

	if opts.Force {
		args = append(args, "--allow-unauthenticated", "--allow-change-held-packages")
	}
	if opts.AllowDowngrade {
		args = append(args, "--allow-downgrades")
	}

	result := m.executeElevatedWithEnv(ctx, "apt-get", args...)

	if result.Failed() {
		stderr := result.StderrString()
		if strings.Contains(stderr, "Could not get lock") {
			return pkg.Wrap(pkg.ErrLockAcquireFailed, fmt.Errorf("apt-get upgrade failed: %s", stderr))
		}
		return pkg.Wrap(pkg.ErrInstallFailed, fmt.Errorf("apt-get upgrade failed (exit code %d): %s", result.ExitCode, stderr))
	}

	return nil
}

// IsInstalled checks if a package is currently installed.
// Uses dpkg-query to check the package status.
func (m *Manager) IsInstalled(ctx context.Context, pkgName string) (bool, error) {
	result := m.executor.Execute(ctx, "dpkg-query", "-W", "-f=${Status}", pkgName)

	if result.Failed() {
		// dpkg-query returns non-zero if package is not found
		if result.ExitCode == 1 {
			return false, nil
		}
		return false, fmt.Errorf("dpkg-query failed: %s", result.StderrString())
	}

	// Check if status indicates installed
	status := strings.TrimSpace(result.StdoutString())
	return strings.Contains(status, "install ok installed"), nil
}

// Search searches for packages matching the query.
// Uses apt-cache search to find matching packages.
func (m *Manager) Search(ctx context.Context, query string, opts pkg.SearchOptions) ([]pkg.Package, error) {
	args := []string{"search"}

	if opts.ExactMatch {
		args = append(args, "--names-only", "^"+query+"$")
	} else {
		args = append(args, query)
	}

	result := m.executor.Execute(ctx, "apt-cache", args...)

	if result.Failed() {
		return nil, fmt.Errorf("apt-cache search failed: %s", result.StderrString())
	}

	packages, err := parseAptCacheSearch(result.StdoutString())
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
// Uses apt-cache show to retrieve package metadata.
func (m *Manager) Info(ctx context.Context, pkgName string) (*pkg.Package, error) {
	result := m.executor.Execute(ctx, "apt-cache", "show", pkgName)

	if result.Failed() {
		stderr := result.StderrString()
		if strings.Contains(stderr, "No packages found") || result.ExitCode == 100 {
			return nil, pkg.WrapWithPackage(pkg.ErrPackageNotFound, pkgName, fmt.Errorf("package not found"))
		}
		return nil, fmt.Errorf("apt-cache show failed: %s", stderr)
	}

	p, err := parseAptCacheShow(result.StdoutString())
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
// Uses dpkg-query to get the list of installed packages.
func (m *Manager) ListInstalled(ctx context.Context) ([]pkg.Package, error) {
	result := m.executor.Execute(ctx, "dpkg-query", "-W", "-f=${Package}\t${Version}\t${Status}\n")

	if result.Failed() {
		return nil, fmt.Errorf("dpkg-query failed: %s", result.StderrString())
	}

	return parseDpkgQuery(result.StdoutString())
}

// ListUpgradable returns a list of packages that can be upgraded.
// Uses apt list --upgradable to get the list.
func (m *Manager) ListUpgradable(ctx context.Context) ([]pkg.Package, error) {
	result := m.executor.Execute(ctx, "apt", "list", "--upgradable")

	if result.Failed() {
		return nil, fmt.Errorf("apt list --upgradable failed: %s", result.StderrString())
	}

	return parseAptListUpgradable(result.StdoutString())
}

// Clean removes cached package files to free disk space.
// Uses apt-get clean.
func (m *Manager) Clean(ctx context.Context) error {
	result := m.executeElevatedWithEnv(ctx, "apt-get", "clean")

	if result.Failed() {
		return fmt.Errorf("apt-get clean failed: %s", result.StderrString())
	}

	return nil
}

// AutoRemove removes automatically installed packages that are no longer needed.
// Uses apt-get autoremove -y.
func (m *Manager) AutoRemove(ctx context.Context) error {
	result := m.executeElevatedWithEnv(ctx, "apt-get", "autoremove", "-y")

	if result.Failed() {
		stderr := result.StderrString()
		if strings.Contains(stderr, "Could not get lock") {
			return pkg.Wrap(pkg.ErrLockAcquireFailed, fmt.Errorf("apt-get autoremove failed: %s", stderr))
		}
		return fmt.Errorf("apt-get autoremove failed: %s", stderr)
	}

	return nil
}

// Verify checks package integrity using debsums if available.
// Falls back to dpkg --verify if debsums is not installed.
func (m *Manager) Verify(ctx context.Context, pkgName string) (bool, error) {
	// First check if package is installed
	installed, err := m.IsInstalled(ctx, pkgName)
	if err != nil {
		return false, err
	}
	if !installed {
		return false, pkg.WrapWithPackage(pkg.ErrPackageNotInstalled, pkgName, fmt.Errorf("package not installed"))
	}

	// Try debsums first (more thorough verification)
	if _, err := exec.LookPath("debsums"); err == nil {
		result := m.executor.Execute(ctx, "debsums", "-s", pkgName)
		// debsums returns 0 if all files match, 2 if files are missing/changed
		return result.ExitCode == 0, nil
	}

	// Fallback: use dpkg --verify
	result := m.executor.Execute(ctx, "dpkg", "--verify", pkgName)
	// dpkg --verify returns 0 if package is OK
	return result.ExitCode == 0, nil
}

// executeElevatedWithEnv runs a command with root privileges and non-interactive environment.
func (m *Manager) executeElevatedWithEnv(ctx context.Context, cmd string, args ...string) *igorexec.Result {
	// Prepend environment variable setup using env command
	envArgs := append(nonInteractiveEnv, cmd)
	envArgs = append(envArgs, args...)

	return m.executor.ExecuteElevated(ctx, "env", envArgs...)
}

// Ensure Manager implements pkg.Manager interface.
var _ pkg.Manager = (*Manager)(nil)
