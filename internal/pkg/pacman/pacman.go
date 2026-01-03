// Package pacman implements the pkg.Manager interface for Pacman-based Linux distributions.
// It supports Arch Linux, Manjaro, EndeavourOS, Garuda Linux, and other Arch derivatives.
package pacman

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

// Manager implements pkg.Manager for Pacman-based distributions.
// It provides package management functionality using pacman and pacman-key commands.
type Manager struct {
	executor  igorexec.Executor
	privilege *privilege.Manager
}

// NewManager creates a new Pacman package manager.
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
	return "pacman"
}

// Family returns the distribution family this manager supports.
func (m *Manager) Family() constants.DistroFamily {
	return constants.FamilyArch
}

// IsAvailable checks if pacman is available on the system.
func (m *Manager) IsAvailable() bool {
	_, err := exec.LookPath("pacman")
	return err == nil
}

// Install installs one or more packages using pacman -S.
// Uses --noconfirm for non-interactive operation.
func (m *Manager) Install(ctx context.Context, opts pkg.InstallOptions, packages ...string) error {
	if len(packages) == 0 {
		return nil
	}

	args := m.buildInstallArgs(opts, packages)
	result := m.executor.ExecuteElevated(ctx, "pacman", args...)

	if result.Failed() {
		stderr := result.StderrString()
		stdout := result.StdoutString()
		combined := stderr + stdout

		// Check for common error patterns
		if strings.Contains(combined, "target not found") ||
			strings.Contains(combined, "could not find") {
			// Try to extract the package name
			for _, p := range packages {
				if strings.Contains(combined, p) {
					return pkg.WrapWithPackage(pkg.ErrPackageNotFound, p, fmt.Errorf("pacman install failed: %s", combined))
				}
			}
			return pkg.Wrap(pkg.ErrPackageNotFound, fmt.Errorf("pacman install failed: %s", combined))
		}
		if strings.Contains(stderr, "unable to lock database") ||
			strings.Contains(stderr, "database is locked") {
			return pkg.Wrap(pkg.ErrLockAcquireFailed, fmt.Errorf("pacman install failed: %s", stderr))
		}
		if strings.Contains(stderr, "conflicting dependencies") ||
			strings.Contains(stderr, "failed to prepare transaction") {
			return pkg.Wrap(pkg.ErrDependencyConflict, fmt.Errorf("pacman install failed: %s", stderr))
		}
		return pkg.Wrap(pkg.ErrInstallFailed, fmt.Errorf("pacman install failed (exit code %d): %s", result.ExitCode, stderr))
	}

	return nil
}

// buildInstallArgs constructs the pacman -S command arguments.
func (m *Manager) buildInstallArgs(opts pkg.InstallOptions, packages []string) []string {
	args := []string{"-S", "--noconfirm"}

	// --overwrite allows reinstalling/overwriting files (covers both Force and AllowDowngrade)
	if opts.Force || opts.AllowDowngrade {
		args = append(args, "--overwrite", "*")
	}
	if opts.DownloadOnly {
		args = append(args, "--downloadonly")
	}
	// Note: SkipVerify is not directly supported by pacman on command line
	// Signature verification is controlled via SigLevel in pacman.conf
	// We don't add any flag here as there's no safe per-command option

	args = append(args, packages...)
	return args
}

// Remove removes one or more packages from the system.
// Uses --noconfirm for non-interactive operation.
func (m *Manager) Remove(ctx context.Context, opts pkg.RemoveOptions, packages ...string) error {
	if len(packages) == 0 {
		return nil
	}

	args := []string{}

	// -Rs removes package with dependencies, -Rns also removes config files (purge)
	if opts.Purge {
		args = append(args, "-Rns", "--noconfirm")
	} else if opts.AutoRemove {
		args = append(args, "-Rs", "--noconfirm")
	} else {
		args = append(args, "-R", "--noconfirm")
	}

	args = append(args, packages...)

	result := m.executor.ExecuteElevated(ctx, "pacman", args...)

	if result.Failed() {
		stderr := result.StderrString()
		stdout := result.StdoutString()
		combined := stderr + stdout

		// Check if package is not installed
		if strings.Contains(combined, "target not found") ||
			strings.Contains(combined, "error: target not found") {
			for _, p := range packages {
				if strings.Contains(combined, p) {
					return pkg.WrapWithPackage(pkg.ErrPackageNotInstalled, p, fmt.Errorf("pacman remove failed: %s", combined))
				}
			}
			return pkg.Wrap(pkg.ErrPackageNotInstalled, fmt.Errorf("pacman remove failed: %s", combined))
		}
		if strings.Contains(stderr, "unable to lock database") ||
			strings.Contains(stderr, "database is locked") {
			return pkg.Wrap(pkg.ErrLockAcquireFailed, fmt.Errorf("pacman remove failed: %s", stderr))
		}
		return pkg.Wrap(pkg.ErrRemoveFailed, fmt.Errorf("pacman remove failed (exit code %d): %s", result.ExitCode, stderr))
	}

	return nil
}

// Update updates the package database using pacman -Sy.
func (m *Manager) Update(ctx context.Context, opts pkg.UpdateOptions) error {
	args := []string{"-Sy"}

	if opts.Quiet {
		args = append(args, "--quiet")
	}

	result := m.executor.ExecuteElevated(ctx, "pacman", args...)

	if result.Failed() {
		stderr := result.StderrString()
		if strings.Contains(stderr, "unable to lock database") ||
			strings.Contains(stderr, "database is locked") {
			return pkg.Wrap(pkg.ErrLockAcquireFailed, fmt.Errorf("pacman -Sy failed: %s", stderr))
		}
		// Network errors
		if strings.Contains(stderr, "failed to retrieve") ||
			strings.Contains(stderr, "failed to download") ||
			strings.Contains(stderr, "error: failed to synchronize") {
			return pkg.Wrap(pkg.ErrNetworkUnavailable, fmt.Errorf("pacman -Sy failed: %s", stderr))
		}
		return pkg.Wrap(pkg.ErrUpdateFailed, fmt.Errorf("pacman -Sy failed (exit code %d): %s", result.ExitCode, stderr))
	}

	return nil
}

// Upgrade upgrades installed packages to their latest versions.
// If packages are specified, only those packages are upgraded.
// If no packages are specified, all upgradable packages are upgraded.
func (m *Manager) Upgrade(ctx context.Context, opts pkg.InstallOptions, packages ...string) error {
	var args []string

	if len(packages) == 0 {
		// Upgrade all packages: pacman -Su --noconfirm
		args = []string{"-Su", "--noconfirm"}
	} else {
		// Upgrade specific packages: pacman -S --noconfirm [packages]
		// This reinstalls/upgrades the specified packages
		args = []string{"-S", "--noconfirm"}
		args = append(args, packages...)
	}

	if opts.Force || opts.AllowDowngrade {
		args = append(args, "--overwrite", "*")
	}
	// Note: SkipVerify is not directly supported by pacman on command line

	result := m.executor.ExecuteElevated(ctx, "pacman", args...)

	if result.Failed() {
		stderr := result.StderrString()
		if strings.Contains(stderr, "unable to lock database") ||
			strings.Contains(stderr, "database is locked") {
			return pkg.Wrap(pkg.ErrLockAcquireFailed, fmt.Errorf("pacman upgrade failed: %s", stderr))
		}
		return pkg.Wrap(pkg.ErrInstallFailed, fmt.Errorf("pacman upgrade failed (exit code %d): %s", result.ExitCode, stderr))
	}

	return nil
}

// IsInstalled checks if a package is currently installed.
// Uses pacman -Q to check the package status.
func (m *Manager) IsInstalled(ctx context.Context, pkgName string) (bool, error) {
	result := m.executor.Execute(ctx, "pacman", "-Q", pkgName)

	if result.Failed() {
		// pacman -Q returns exit code 1 if package is not installed
		if result.ExitCode == 1 {
			return false, nil
		}
		return false, fmt.Errorf("pacman -Q failed: %s", result.StderrString())
	}

	return true, nil
}

// Search searches for packages matching the query.
// Uses pacman -Ss to find matching packages.
func (m *Manager) Search(ctx context.Context, query string, opts pkg.SearchOptions) ([]pkg.Package, error) {
	args := []string{"-Ss", query}

	result := m.executor.Execute(ctx, "pacman", args...)

	if result.Failed() {
		stderr := result.StderrString()
		// No matches is not an error
		if result.ExitCode == 1 && stderr == "" {
			return []pkg.Package{}, nil
		}
		if strings.Contains(stderr, "no matches found") ||
			strings.Contains(result.StdoutString(), "no packages found") {
			return []pkg.Package{}, nil
		}
		return nil, fmt.Errorf("pacman search failed: %s", stderr)
	}

	packages, err := parsePacmanSs(result.StdoutString())
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
// Uses pacman -Si for remote packages or pacman -Qi for local packages.
func (m *Manager) Info(ctx context.Context, pkgName string) (*pkg.Package, error) {
	// First try remote info
	result := m.executor.Execute(ctx, "pacman", "-Si", pkgName)

	var p *pkg.Package
	var err error

	if result.Failed() {
		// Package might only be installed locally, try -Qi
		result = m.executor.Execute(ctx, "pacman", "-Qi", pkgName)
		if result.Failed() {
			stderr := result.StderrString()
			if strings.Contains(stderr, "was not found") ||
				strings.Contains(stderr, "package") && strings.Contains(stderr, "not found") {
				return nil, pkg.WrapWithPackage(pkg.ErrPackageNotFound, pkgName, fmt.Errorf("package not found"))
			}
			return nil, fmt.Errorf("pacman info failed: %s", stderr)
		}
		p, err = parsePacmanQi(result.StdoutString())
	} else {
		p, err = parsePacmanSi(result.StdoutString())
	}

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
// Uses pacman -Q.
func (m *Manager) ListInstalled(ctx context.Context) ([]pkg.Package, error) {
	result := m.executor.Execute(ctx, "pacman", "-Q")

	if result.Failed() {
		return nil, fmt.Errorf("pacman -Q failed: %s", result.StderrString())
	}

	return parsePacmanQ(result.StdoutString())
}

// ListUpgradable returns a list of packages that can be upgraded.
// Uses pacman -Qu.
func (m *Manager) ListUpgradable(ctx context.Context) ([]pkg.Package, error) {
	result := m.executor.Execute(ctx, "pacman", "-Qu")

	// pacman -Qu returns exit code 1 if no upgrades available
	if result.Failed() {
		if result.ExitCode == 1 {
			// No upgrades available
			return []pkg.Package{}, nil
		}
		return nil, fmt.Errorf("pacman -Qu failed: %s", result.StderrString())
	}

	return parsePacmanQu(result.StdoutString())
}

// Clean removes cached package files to free disk space.
// Uses pacman -Sc --noconfirm.
func (m *Manager) Clean(ctx context.Context) error {
	result := m.executor.ExecuteElevated(ctx, "pacman", "-Sc", "--noconfirm")

	if result.Failed() {
		return fmt.Errorf("pacman clean failed: %s", result.StderrString())
	}

	return nil
}

// AutoRemove removes automatically installed packages that are no longer needed.
// Uses pacman -Rns $(pacman -Qdtq) --noconfirm.
// This removes orphan packages (dependencies that are no longer required).
func (m *Manager) AutoRemove(ctx context.Context) error {
	// First, get the list of orphan packages
	result := m.executor.Execute(ctx, "pacman", "-Qdtq")

	// If no orphans, pacman -Qdtq returns exit code 1 and empty output
	if result.Failed() || strings.TrimSpace(result.StdoutString()) == "" {
		// No orphan packages to remove
		return nil
	}

	// Get the list of orphan packages
	orphans := strings.Fields(strings.TrimSpace(result.StdoutString()))
	if len(orphans) == 0 {
		return nil
	}

	// Remove the orphan packages
	args := append([]string{"-Rns", "--noconfirm"}, orphans...)
	result = m.executor.ExecuteElevated(ctx, "pacman", args...)

	if result.Failed() {
		stderr := result.StderrString()
		if strings.Contains(stderr, "unable to lock database") ||
			strings.Contains(stderr, "database is locked") {
			return pkg.Wrap(pkg.ErrLockAcquireFailed, fmt.Errorf("pacman autoremove failed: %s", stderr))
		}
		return fmt.Errorf("pacman autoremove failed: %s", stderr)
	}

	return nil
}

// Verify checks package integrity using pacman -Qk.
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

	// Use pacman -Qk for verification
	// -Qk checks for missing files, -Qkk also verifies file properties
	result := m.executor.Execute(ctx, "pacman", "-Qk", pkgName)

	// pacman -Qk returns 0 if package is OK
	// It returns 1 if there are missing files
	if result.Failed() {
		// Check if it's a verification failure or an error
		if result.ExitCode == 1 {
			// Files are missing
			return false, nil
		}
		return false, fmt.Errorf("pacman verify failed: %s", result.StderrString())
	}

	return true, nil
}

// Ensure Manager implements pkg.Manager interface.
var _ pkg.Manager = (*Manager)(nil)
