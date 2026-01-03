package dnf

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/tungetti/igor/internal/pkg"
)

// Repository paths for DNF/YUM.
const (
	repoDir = "/etc/yum.repos.d"
)

// AddRepository adds a new package repository.
// Uses dnf config-manager --add-repo for URLs, or creates a .repo file.
func (m *Manager) AddRepository(ctx context.Context, repo pkg.Repository) error {
	if repo.URL == "" {
		return fmt.Errorf("repository URL is required")
	}

	// If URL ends with .repo, use config-manager to add it directly
	if strings.HasSuffix(repo.URL, ".repo") {
		return m.addRepoFromURL(ctx, repo.URL)
	}

	// Otherwise, create a .repo file
	return m.addRepoFile(ctx, repo)
}

// addRepoFromURL adds a repository from a .repo URL using dnf config-manager.
func (m *Manager) addRepoFromURL(ctx context.Context, repoURL string) error {
	result := m.executor.ExecuteElevated(ctx, "dnf", "config-manager", "--add-repo", repoURL)

	if result.Failed() {
		stderr := result.StderrString()
		if strings.Contains(stderr, "already exists") {
			return pkg.Wrap(pkg.ErrRepositoryExists, fmt.Errorf("repository already exists: %s", repoURL))
		}
		return fmt.Errorf("dnf config-manager --add-repo failed: %s", stderr)
	}

	return nil
}

// addRepoFile creates a .repo file in /etc/yum.repos.d/.
func (m *Manager) addRepoFile(ctx context.Context, repo pkg.Repository) error {
	// Generate filename from repo name
	filename := sanitizeRepoID(repo.Name) + ".repo"
	repoFilePath := filepath.Join(repoDir, filename)

	// Check if file already exists
	result := m.executor.Execute(ctx, "test", "-f", repoFilePath)
	if result.ExitCode == 0 {
		return pkg.Wrap(pkg.ErrRepositoryExists, fmt.Errorf("repository file already exists: %s", repoFilePath))
	}

	// Build the repo file content
	content := buildRepoFileContent(repo)

	// Write the file using tee with elevated privileges
	result = m.executor.ExecuteWithInput(ctx, []byte(content), "sudo", "tee", repoFilePath)
	if result.Failed() {
		return fmt.Errorf("failed to create repository file: %s", result.StderrString())
	}

	return nil
}

// RemoveRepository removes a package repository.
func (m *Manager) RemoveRepository(ctx context.Context, name string) error {
	// Try to find and remove the .repo file
	filename := sanitizeRepoID(name) + ".repo"
	repoFilePath := filepath.Join(repoDir, filename)

	// Check if file exists
	result := m.executor.Execute(ctx, "test", "-f", repoFilePath)
	if result.ExitCode != 0 {
		// Try with the name as-is
		repoFilePath = filepath.Join(repoDir, name+".repo")
		result = m.executor.Execute(ctx, "test", "-f", repoFilePath)
		if result.ExitCode != 0 {
			// Try to find any repo file containing the name
			repoFilePath = filepath.Join(repoDir, name)
			result = m.executor.Execute(ctx, "test", "-f", repoFilePath)
			if result.ExitCode != 0 {
				return pkg.Wrap(pkg.ErrRepositoryNotFound, fmt.Errorf("repository not found: %s", name))
			}
		}
	}

	// Remove the file
	result = m.executor.ExecuteElevated(ctx, "rm", "-f", repoFilePath)
	if result.Failed() {
		return fmt.Errorf("failed to remove repository file: %s", result.StderrString())
	}

	return nil
}

// ListRepositories returns a list of configured repositories.
// Uses dnf repolist --all.
func (m *Manager) ListRepositories(ctx context.Context) ([]pkg.Repository, error) {
	result := m.executor.Execute(ctx, "dnf", "repolist", "--all")

	if result.Failed() {
		return nil, fmt.Errorf("dnf repolist failed: %s", result.StderrString())
	}

	return parseDnfRepolist(result.StdoutString())
}

// EnableRepository enables a disabled repository.
// Uses dnf config-manager --set-enabled.
func (m *Manager) EnableRepository(ctx context.Context, name string) error {
	result := m.executor.ExecuteElevated(ctx, "dnf", "config-manager", "--set-enabled", name)

	if result.Failed() {
		stderr := result.StderrString()
		if strings.Contains(stderr, "No matching repo") ||
			strings.Contains(stderr, "Error:") {
			return pkg.Wrap(pkg.ErrRepositoryNotFound, fmt.Errorf("repository not found: %s", name))
		}
		return fmt.Errorf("dnf config-manager --set-enabled failed: %s", stderr)
	}

	return nil
}

// DisableRepository disables an enabled repository.
// Uses dnf config-manager --set-disabled.
func (m *Manager) DisableRepository(ctx context.Context, name string) error {
	result := m.executor.ExecuteElevated(ctx, "dnf", "config-manager", "--set-disabled", name)

	if result.Failed() {
		stderr := result.StderrString()
		if strings.Contains(stderr, "No matching repo") ||
			strings.Contains(stderr, "Error:") {
			return pkg.Wrap(pkg.ErrRepositoryNotFound, fmt.Errorf("repository not found: %s", name))
		}
		return fmt.Errorf("dnf config-manager --set-disabled failed: %s", stderr)
	}

	return nil
}

// RefreshRepositories refreshes all repository metadata.
// Uses dnf makecache.
func (m *Manager) RefreshRepositories(ctx context.Context) error {
	result := m.executor.ExecuteElevated(ctx, "dnf", "makecache")

	if result.Failed() {
		stderr := result.StderrString()
		if strings.Contains(stderr, "Could not resolve") ||
			strings.Contains(stderr, "Failed to download") {
			return pkg.Wrap(pkg.ErrNetworkUnavailable, fmt.Errorf("dnf makecache failed: %s", stderr))
		}
		return fmt.Errorf("dnf makecache failed: %s", stderr)
	}

	return nil
}

// AddRPMFusion enables RPM Fusion repositories (common for NVIDIA drivers on Fedora).
// Installs the free and/or nonfree release packages.
func (m *Manager) AddRPMFusion(ctx context.Context, free, nonfree bool) error {
	// First, detect if we're on Fedora or RHEL-like
	// We'll use the fedora-release package detection

	var packages []string

	if free {
		// RPM Fusion Free repository
		packages = append(packages,
			"https://download1.rpmfusion.org/free/fedora/rpmfusion-free-release-$(rpm -E %fedora).noarch.rpm")
	}

	if nonfree {
		// RPM Fusion Nonfree repository (needed for NVIDIA drivers)
		packages = append(packages,
			"https://download1.rpmfusion.org/nonfree/fedora/rpmfusion-nonfree-release-$(rpm -E %fedora).noarch.rpm")
	}

	if len(packages) == 0 {
		return nil
	}

	// Build the shell command since we need variable expansion
	shellCmd := fmt.Sprintf("dnf install -y %s", strings.Join(packages, " "))
	result := m.executor.ExecuteElevated(ctx, "sh", "-c", shellCmd)

	if result.Failed() {
		stderr := result.StderrString()
		// Check if already installed
		if strings.Contains(stderr, "already installed") ||
			strings.Contains(result.StdoutString(), "already installed") {
			return nil // Already installed is not an error
		}
		return fmt.Errorf("failed to add RPM Fusion repositories: %s", stderr)
	}

	return nil
}

// AddRPMFusionEL enables RPM Fusion repositories for RHEL/Rocky/AlmaLinux.
// This uses different URLs than Fedora.
func (m *Manager) AddRPMFusionEL(ctx context.Context, free, nonfree bool) error {
	var packages []string

	if free {
		packages = append(packages,
			"https://download1.rpmfusion.org/free/el/rpmfusion-free-release-$(rpm -E %rhel).noarch.rpm")
	}

	if nonfree {
		packages = append(packages,
			"https://download1.rpmfusion.org/nonfree/el/rpmfusion-nonfree-release-$(rpm -E %rhel).noarch.rpm")
	}

	if len(packages) == 0 {
		return nil
	}

	// Also need EPEL on EL systems for some dependencies
	// First install EPEL
	epelResult := m.executor.ExecuteElevated(ctx, "dnf", "install", "-y", "epel-release")
	if epelResult.Failed() {
		// EPEL might not be needed or already installed, continue anyway
		_ = epelResult
	}

	// Build the shell command since we need variable expansion
	shellCmd := fmt.Sprintf("dnf install -y %s", strings.Join(packages, " "))
	result := m.executor.ExecuteElevated(ctx, "sh", "-c", shellCmd)

	if result.Failed() {
		stderr := result.StderrString()
		// Check if already installed
		if strings.Contains(stderr, "already installed") ||
			strings.Contains(result.StdoutString(), "already installed") {
			return nil
		}
		return fmt.Errorf("failed to add RPM Fusion repositories: %s", stderr)
	}

	return nil
}

// ImportGPGKey imports a GPG key for repository verification.
// Uses rpm --import.
func (m *Manager) ImportGPGKey(ctx context.Context, keyURL string) error {
	result := m.executor.ExecuteElevated(ctx, "rpm", "--import", keyURL)

	if result.Failed() {
		stderr := result.StderrString()
		if strings.Contains(stderr, "Could not resolve") ||
			strings.Contains(stderr, "not retriev") {
			return pkg.Wrap(pkg.ErrNetworkUnavailable, fmt.Errorf("failed to import GPG key: %s", stderr))
		}
		return fmt.Errorf("rpm --import failed: %s", stderr)
	}

	return nil
}

// GetRepoFilePath returns the path where a repository file should be stored.
func (m *Manager) GetRepoFilePath(name string) string {
	return filepath.Join(repoDir, sanitizeRepoID(name)+".repo")
}
