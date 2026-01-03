package zypper

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/tungetti/igor/internal/pkg"
)

// Repository paths for Zypper.
const (
	repoDir = "/etc/zypp/repos.d"
)

// NVIDIA repository URLs for different openSUSE versions.
const (
	// NvidiaRepoTumbleweed is the NVIDIA repository URL for openSUSE Tumbleweed.
	NvidiaRepoTumbleweed = "https://download.nvidia.com/opensuse/tumbleweed"
	// NvidiaRepoLeap15 is the NVIDIA repository URL template for openSUSE Leap 15.x.
	// Replace the version number as needed (e.g., leap/15.5, leap/15.6).
	NvidiaRepoLeap15 = "https://download.nvidia.com/opensuse/leap/%s"
)

// AddRepository adds a new package repository.
// Uses zypper addrepo to add the repository.
func (m *Manager) AddRepository(ctx context.Context, repo pkg.Repository) error {
	if repo.URL == "" {
		return fmt.Errorf("repository URL is required")
	}

	args := []string{"addrepo", "--refresh"}

	if !repo.Enabled {
		args = append(args, "--disable")
	}

	if repo.GPGKey == "" {
		args = append(args, "--no-gpgcheck")
	}

	args = append(args, repo.URL, repo.Name)

	result := m.executor.ExecuteElevated(ctx, "zypper", args...)

	if result.Failed() {
		stderr := result.StderrString()
		if strings.Contains(stderr, "already exists") ||
			strings.Contains(stderr, "Repository named") {
			return pkg.Wrap(pkg.ErrRepositoryExists, fmt.Errorf("repository already exists: %s", repo.Name))
		}
		return fmt.Errorf("zypper addrepo failed: %s", stderr)
	}

	// Import GPG key if provided
	if repo.GPGKey != "" {
		if err := m.ImportGPGKey(ctx, repo.GPGKey); err != nil {
			// Log the error but don't fail the operation
			_ = err
		}
	}

	return nil
}

// RemoveRepository removes a package repository.
// Uses zypper removerepo to remove the repository.
func (m *Manager) RemoveRepository(ctx context.Context, name string) error {
	result := m.executor.ExecuteElevated(ctx, "zypper", "removerepo", name)

	if result.Failed() {
		stderr := result.StderrString()
		if strings.Contains(stderr, "not found") ||
			strings.Contains(stderr, "No repository") {
			return pkg.Wrap(pkg.ErrRepositoryNotFound, fmt.Errorf("repository not found: %s", name))
		}
		return fmt.Errorf("zypper removerepo failed: %s", stderr)
	}

	return nil
}

// ListRepositories returns a list of configured repositories.
// Uses zypper repos.
func (m *Manager) ListRepositories(ctx context.Context) ([]pkg.Repository, error) {
	result := m.executor.Execute(ctx, "zypper", "repos")

	if result.Failed() {
		stderr := result.StderrString()
		// No repositories is not an error
		if strings.Contains(stderr, "No repositories defined") ||
			strings.Contains(result.StdoutString(), "No repositories defined") {
			return []pkg.Repository{}, nil
		}
		return nil, fmt.Errorf("zypper repos failed: %s", stderr)
	}

	return parseZypperRepos(result.StdoutString())
}

// EnableRepository enables a disabled repository.
// Uses zypper modifyrepo --enable.
func (m *Manager) EnableRepository(ctx context.Context, name string) error {
	result := m.executor.ExecuteElevated(ctx, "zypper", "modifyrepo", "--enable", name)

	if result.Failed() {
		stderr := result.StderrString()
		if strings.Contains(stderr, "not found") ||
			strings.Contains(stderr, "No repository") {
			return pkg.Wrap(pkg.ErrRepositoryNotFound, fmt.Errorf("repository not found: %s", name))
		}
		return fmt.Errorf("zypper modifyrepo --enable failed: %s", stderr)
	}

	return nil
}

// DisableRepository disables an enabled repository.
// Uses zypper modifyrepo --disable.
func (m *Manager) DisableRepository(ctx context.Context, name string) error {
	result := m.executor.ExecuteElevated(ctx, "zypper", "modifyrepo", "--disable", name)

	if result.Failed() {
		stderr := result.StderrString()
		if strings.Contains(stderr, "not found") ||
			strings.Contains(stderr, "No repository") {
			return pkg.Wrap(pkg.ErrRepositoryNotFound, fmt.Errorf("repository not found: %s", name))
		}
		return fmt.Errorf("zypper modifyrepo --disable failed: %s", stderr)
	}

	return nil
}

// RefreshRepositories refreshes all repository metadata.
// Uses zypper refresh.
func (m *Manager) RefreshRepositories(ctx context.Context) error {
	result := m.executor.ExecuteElevated(ctx, "zypper", "refresh")

	if result.Failed() {
		stderr := result.StderrString()
		if strings.Contains(stderr, "Could not resolve") ||
			strings.Contains(stderr, "Timeout") ||
			strings.Contains(stderr, "Network") {
			return pkg.Wrap(pkg.ErrNetworkUnavailable, fmt.Errorf("zypper refresh failed: %s", stderr))
		}
		return fmt.Errorf("zypper refresh failed: %s", stderr)
	}

	return nil
}

// AddNvidiaRepo adds the official NVIDIA repository for openSUSE.
// The version parameter should be either "tumbleweed" or a Leap version like "15.5".
func (m *Manager) AddNvidiaRepo(ctx context.Context, version string) error {
	var repoURL string

	version = strings.ToLower(strings.TrimSpace(version))
	if version == "tumbleweed" || version == "" {
		repoURL = NvidiaRepoTumbleweed
	} else {
		// Assume it's a Leap version
		repoURL = fmt.Sprintf(NvidiaRepoLeap15, version)
	}

	// Check if repository already exists
	repos, err := m.ListRepositories(ctx)
	if err == nil {
		for _, r := range repos {
			if strings.ToLower(r.Name) == "nvidia" {
				// Repository already exists, just ensure it's enabled
				return m.EnableRepository(ctx, r.Name)
			}
		}
	}

	// Add the repository with --refresh and --no-gpgcheck (NVIDIA repo uses its own key)
	args := []string{"addrepo", "--refresh", "--no-gpgcheck", repoURL, "NVIDIA"}
	result := m.executor.ExecuteElevated(ctx, "zypper", args...)

	if result.Failed() {
		stderr := result.StderrString()
		if strings.Contains(stderr, "already exists") {
			return nil // Already exists is fine
		}
		return fmt.Errorf("failed to add NVIDIA repository: %s", stderr)
	}

	// Refresh to get the repository metadata
	_ = m.RefreshRepositories(ctx)

	return nil
}

// AddNvidiaRepoTumbleweed adds the NVIDIA repository for openSUSE Tumbleweed.
func (m *Manager) AddNvidiaRepoTumbleweed(ctx context.Context) error {
	return m.AddNvidiaRepo(ctx, "tumbleweed")
}

// AddNvidiaRepoLeap adds the NVIDIA repository for openSUSE Leap.
// The version should be the Leap version number (e.g., "15.5", "15.6").
func (m *Manager) AddNvidiaRepoLeap(ctx context.Context, version string) error {
	return m.AddNvidiaRepo(ctx, version)
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

// SetRepositoryPriority sets the priority for a repository.
// Lower numbers mean higher priority.
// Uses zypper modifyrepo --priority.
func (m *Manager) SetRepositoryPriority(ctx context.Context, name string, priority int) error {
	result := m.executor.ExecuteElevated(ctx, "zypper", "modifyrepo", "--priority",
		fmt.Sprintf("%d", priority), name)

	if result.Failed() {
		stderr := result.StderrString()
		if strings.Contains(stderr, "not found") ||
			strings.Contains(stderr, "No repository") {
			return pkg.Wrap(pkg.ErrRepositoryNotFound, fmt.Errorf("repository not found: %s", name))
		}
		return fmt.Errorf("zypper modifyrepo --priority failed: %s", stderr)
	}

	return nil
}

// SetRepositoryRefresh sets whether a repository should be automatically refreshed.
// Uses zypper modifyrepo --refresh or --no-refresh.
func (m *Manager) SetRepositoryRefresh(ctx context.Context, name string, refresh bool) error {
	var flag string
	if refresh {
		flag = "--refresh"
	} else {
		flag = "--no-refresh"
	}

	result := m.executor.ExecuteElevated(ctx, "zypper", "modifyrepo", flag, name)

	if result.Failed() {
		stderr := result.StderrString()
		if strings.Contains(stderr, "not found") ||
			strings.Contains(stderr, "No repository") {
			return pkg.Wrap(pkg.ErrRepositoryNotFound, fmt.Errorf("repository not found: %s", name))
		}
		return fmt.Errorf("zypper modifyrepo %s failed: %s", flag, stderr)
	}

	return nil
}
