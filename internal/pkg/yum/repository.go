package yum

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/tungetti/igor/internal/pkg"
)

// Repository paths for YUM.
const (
	repoDir = "/etc/yum.repos.d"
)

// AddRepository adds a new package repository.
// Uses yum-config-manager --add-repo for URLs, or creates a .repo file.
func (m *Manager) AddRepository(ctx context.Context, repo pkg.Repository) error {
	if repo.URL == "" {
		return fmt.Errorf("repository URL is required")
	}

	// If URL ends with .repo, use yum-config-manager to add it directly
	if strings.HasSuffix(repo.URL, ".repo") {
		return m.addRepoFromURL(ctx, repo.URL)
	}

	// Otherwise, create a .repo file
	return m.addRepoFile(ctx, repo)
}

// addRepoFromURL adds a repository from a .repo URL using yum-config-manager.
func (m *Manager) addRepoFromURL(ctx context.Context, repoURL string) error {
	result := m.executor.ExecuteElevated(ctx, "yum-config-manager", "--add-repo", repoURL)

	if result.Failed() {
		stderr := result.StderrString()
		stdout := result.StdoutString()
		combined := stderr + stdout

		if strings.Contains(combined, "already exists") {
			return pkg.Wrap(pkg.ErrRepositoryExists, fmt.Errorf("repository already exists: %s", repoURL))
		}

		// If yum-config-manager is not available, try alternative method
		if strings.Contains(stderr, "command not found") ||
			strings.Contains(stderr, "No such file") {
			// yum-config-manager is part of yum-utils, try to install it
			installResult := m.executor.ExecuteElevated(ctx, "yum", "install", "-y", "yum-utils")
			if installResult.ExitCode == 0 {
				// Retry the add-repo
				result = m.executor.ExecuteElevated(ctx, "yum-config-manager", "--add-repo", repoURL)
				if result.ExitCode == 0 {
					return nil
				}
			}
			// Fall back to downloading the repo file manually
			return m.downloadRepoFile(ctx, repoURL)
		}

		return fmt.Errorf("yum-config-manager --add-repo failed: %s", stderr)
	}

	return nil
}

// downloadRepoFile downloads a .repo file using curl.
func (m *Manager) downloadRepoFile(ctx context.Context, repoURL string) error {
	// Extract filename from URL
	parts := strings.Split(repoURL, "/")
	filename := parts[len(parts)-1]
	if filename == "" {
		filename = "custom.repo"
	}

	repoFilePath := filepath.Join(repoDir, filename)

	// Use curl to download the repo file
	result := m.executor.ExecuteElevated(ctx, "curl", "-o", repoFilePath, "-L", repoURL)
	if result.Failed() {
		return fmt.Errorf("failed to download repository file: %s", result.StderrString())
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
// Uses yum repolist all.
func (m *Manager) ListRepositories(ctx context.Context) ([]pkg.Repository, error) {
	result := m.executor.Execute(ctx, "yum", "repolist", "all")

	if result.Failed() {
		return nil, fmt.Errorf("yum repolist failed: %s", result.StderrString())
	}

	return parseYumRepolist(result.StdoutString())
}

// EnableRepository enables a disabled repository.
// Uses yum-config-manager --enable.
func (m *Manager) EnableRepository(ctx context.Context, name string) error {
	result := m.executor.ExecuteElevated(ctx, "yum-config-manager", "--enable", name)

	if result.Failed() {
		stderr := result.StderrString()

		// If yum-config-manager is not available, try modifying the repo file directly
		if strings.Contains(stderr, "command not found") ||
			strings.Contains(stderr, "No such file") {
			return m.setRepoEnabled(ctx, name, true)
		}

		if strings.Contains(stderr, "No matching repo") ||
			strings.Contains(stderr, "Error:") {
			return pkg.Wrap(pkg.ErrRepositoryNotFound, fmt.Errorf("repository not found: %s", name))
		}
		return fmt.Errorf("yum-config-manager --enable failed: %s", stderr)
	}

	return nil
}

// DisableRepository disables an enabled repository.
// Uses yum-config-manager --disable.
func (m *Manager) DisableRepository(ctx context.Context, name string) error {
	result := m.executor.ExecuteElevated(ctx, "yum-config-manager", "--disable", name)

	if result.Failed() {
		stderr := result.StderrString()

		// If yum-config-manager is not available, try modifying the repo file directly
		if strings.Contains(stderr, "command not found") ||
			strings.Contains(stderr, "No such file") {
			return m.setRepoEnabled(ctx, name, false)
		}

		if strings.Contains(stderr, "No matching repo") ||
			strings.Contains(stderr, "Error:") {
			return pkg.Wrap(pkg.ErrRepositoryNotFound, fmt.Errorf("repository not found: %s", name))
		}
		return fmt.Errorf("yum-config-manager --disable failed: %s", stderr)
	}

	return nil
}

// setRepoEnabled modifies a .repo file to set enabled=0 or enabled=1.
// This is a fallback when yum-config-manager is not available.
func (m *Manager) setRepoEnabled(ctx context.Context, name string, enabled bool) error {
	// Find the repo file
	filename := sanitizeRepoID(name) + ".repo"
	repoFilePath := filepath.Join(repoDir, filename)

	// Check if file exists
	result := m.executor.Execute(ctx, "test", "-f", repoFilePath)
	if result.ExitCode != 0 {
		// Try with the name as-is
		repoFilePath = filepath.Join(repoDir, name+".repo")
		result = m.executor.Execute(ctx, "test", "-f", repoFilePath)
		if result.ExitCode != 0 {
			return pkg.Wrap(pkg.ErrRepositoryNotFound, fmt.Errorf("repository not found: %s", name))
		}
	}

	// Use sed to modify the enabled setting
	enabledValue := "0"
	if enabled {
		enabledValue = "1"
	}

	// sed -i 's/enabled=./enabled=1/' /etc/yum.repos.d/repo.repo
	sedExpr := fmt.Sprintf("s/enabled=./enabled=%s/", enabledValue)
	result = m.executor.ExecuteElevated(ctx, "sed", "-i", sedExpr, repoFilePath)
	if result.Failed() {
		return fmt.Errorf("failed to modify repository file: %s", result.StderrString())
	}

	return nil
}

// RefreshRepositories refreshes all repository metadata.
// Uses yum makecache.
func (m *Manager) RefreshRepositories(ctx context.Context) error {
	result := m.executor.ExecuteElevated(ctx, "yum", "makecache")

	if result.Failed() {
		stderr := result.StderrString()
		if strings.Contains(stderr, "Could not resolve") ||
			strings.Contains(stderr, "Cannot retrieve") ||
			strings.Contains(stderr, "Cannot find a valid baseurl") {
			return pkg.Wrap(pkg.ErrNetworkUnavailable, fmt.Errorf("yum makecache failed: %s", stderr))
		}
		return fmt.Errorf("yum makecache failed: %s", stderr)
	}

	return nil
}

// AddRPMFusionEL enables RPM Fusion repositories for EL7 systems.
// Note: RPM Fusion support for EL7 is limited compared to Fedora.
func (m *Manager) AddRPMFusionEL(ctx context.Context, free, nonfree bool) error {
	var packages []string

	// First ensure EPEL is installed (required for RPM Fusion)
	if err := m.AddEPEL(ctx); err != nil {
		// EPEL installation failure is not fatal, continue anyway
		_ = err
	}

	if free {
		packages = append(packages,
			"https://download1.rpmfusion.org/free/el/rpmfusion-free-release-7.noarch.rpm")
	}

	if nonfree {
		packages = append(packages,
			"https://download1.rpmfusion.org/nonfree/el/rpmfusion-nonfree-release-7.noarch.rpm")
	}

	if len(packages) == 0 {
		return nil
	}

	// Install the RPM Fusion release packages
	args := append([]string{"install", "-y"}, packages...)
	result := m.executor.ExecuteElevated(ctx, "yum", args...)

	if result.Failed() {
		stderr := result.StderrString()
		stdout := result.StdoutString()
		// Check if already installed
		if strings.Contains(stderr, "already installed") ||
			strings.Contains(stdout, "already installed") {
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

// InstallYumUtils installs yum-utils package which provides yum-config-manager.
func (m *Manager) InstallYumUtils(ctx context.Context) error {
	result := m.executor.ExecuteElevated(ctx, "yum", "install", "-y", "yum-utils")

	if result.Failed() {
		stderr := result.StderrString()
		stdout := result.StdoutString()
		// Check if already installed
		if strings.Contains(stderr, "already installed") ||
			strings.Contains(stdout, "already installed") {
			return nil
		}
		return fmt.Errorf("failed to install yum-utils: %s", stderr)
	}

	return nil
}

// AddNvidiaRepo adds the NVIDIA CUDA repository for EL7.
func (m *Manager) AddNvidiaRepo(ctx context.Context) error {
	repoURL := "https://developer.download.nvidia.com/compute/cuda/repos/rhel7/x86_64/cuda-rhel7.repo"
	return m.addRepoFromURL(ctx, repoURL)
}

// AddElrepo adds the ELRepo repository which provides additional drivers and kernel modules.
// ELRepo is commonly used for NVIDIA drivers on RHEL/CentOS 7.
func (m *Manager) AddElrepo(ctx context.Context) error {
	// Import the GPG key first
	keyResult := m.executor.ExecuteElevated(ctx, "rpm", "--import", "https://www.elrepo.org/RPM-GPG-KEY-elrepo.org")
	if keyResult.Failed() {
		// Continue anyway, the install will prompt
		_ = keyResult
	}

	// Install the elrepo-release package
	result := m.executor.ExecuteElevated(ctx, "yum", "install", "-y",
		"https://www.elrepo.org/elrepo-release-7.el7.elrepo.noarch.rpm")

	if result.Failed() {
		stderr := result.StderrString()
		stdout := result.StdoutString()
		// Check if already installed
		if strings.Contains(stderr, "already installed") ||
			strings.Contains(stdout, "already installed") {
			return nil
		}
		return fmt.Errorf("failed to add ELRepo repository: %s", stderr)
	}

	return nil
}
