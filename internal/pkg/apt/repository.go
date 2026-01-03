package apt

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/tungetti/igor/internal/pkg"
)

// Repository paths for APT.
const (
	sourcesListPath = "/etc/apt/sources.list"
	sourcesListDir  = "/etc/apt/sources.list.d"
	aptKeyringsDir  = "/etc/apt/keyrings"
	trustedGPGDir   = "/etc/apt/trusted.gpg.d"
)

// AddRepository adds a new package repository.
// For PPAs: uses add-apt-repository -y ppa:user/ppa
// For direct URLs: creates a .list file in /etc/apt/sources.list.d/
func (m *Manager) AddRepository(ctx context.Context, repo pkg.Repository) error {
	// Check if it's a PPA
	if strings.HasPrefix(repo.URL, "ppa:") {
		return m.addPPA(ctx, repo.URL)
	}

	// For direct repository URLs, create a .list file
	return m.addDirectRepository(ctx, repo)
}

// addPPA adds a PPA repository using add-apt-repository.
func (m *Manager) addPPA(ctx context.Context, ppa string) error {
	result := m.executeElevatedWithEnv(ctx, "add-apt-repository", "-y", ppa)

	if result.Failed() {
		stderr := result.StderrString()
		if strings.Contains(stderr, "already exists") {
			return pkg.Wrap(pkg.ErrRepositoryExists, fmt.Errorf("PPA already exists: %s", ppa))
		}
		return fmt.Errorf("add-apt-repository failed: %s", stderr)
	}

	return nil
}

// addDirectRepository adds a repository by creating a .list file.
func (m *Manager) addDirectRepository(ctx context.Context, repo pkg.Repository) error {
	// Generate filename from repo name
	filename := sanitizeFilename(repo.Name) + ".list"
	repoFilePath := filepath.Join(sourcesListDir, filename)

	// Check if file already exists
	result := m.executor.Execute(ctx, "test", "-f", repoFilePath)
	if result.ExitCode == 0 {
		return pkg.Wrap(pkg.ErrRepositoryExists, fmt.Errorf("repository file already exists: %s", repoFilePath))
	}

	// Build the sources.list entry
	entry := buildSourcesListEntry(repo)

	// Write the file using tee with elevated privileges
	result = m.executor.ExecuteWithInput(ctx, []byte(entry), "sudo", "tee", repoFilePath)
	if result.Failed() {
		return fmt.Errorf("failed to create repository file: %s", result.StderrString())
	}

	return nil
}

// buildSourcesListEntry creates a sources.list format entry from a Repository.
func buildSourcesListEntry(repo pkg.Repository) string {
	var sb strings.Builder

	// Type (deb or deb-src)
	repoType := repo.Type
	if repoType == "" {
		repoType = "deb"
	}
	sb.WriteString(repoType)
	sb.WriteString(" ")

	// Options (GPG key signing)
	if repo.GPGKey != "" {
		sb.WriteString("[")
		sb.WriteString(fmt.Sprintf("signed-by=%s", repo.GPGKey))
		sb.WriteString("] ")
	}

	// URL
	sb.WriteString(repo.URL)
	sb.WriteString(" ")

	// Distribution
	if repo.Distribution != "" {
		sb.WriteString(repo.Distribution)
	} else {
		sb.WriteString("/")
	}

	// Components
	if len(repo.Components) > 0 {
		sb.WriteString(" ")
		sb.WriteString(strings.Join(repo.Components, " "))
	}

	sb.WriteString("\n")
	return sb.String()
}

// RemoveRepository removes a package repository.
func (m *Manager) RemoveRepository(ctx context.Context, name string) error {
	// Check if it's a PPA format
	if strings.HasPrefix(name, "ppa:") {
		return m.removePPA(ctx, name)
	}

	// Try to find and remove the .list file
	filename := sanitizeFilename(name) + ".list"
	repoFilePath := filepath.Join(sourcesListDir, filename)

	// Check if file exists
	result := m.executor.Execute(ctx, "test", "-f", repoFilePath)
	if result.ExitCode != 0 {
		// Try without sanitization
		repoFilePath = filepath.Join(sourcesListDir, name+".list")
		result = m.executor.Execute(ctx, "test", "-f", repoFilePath)
		if result.ExitCode != 0 {
			return pkg.Wrap(pkg.ErrRepositoryNotFound, fmt.Errorf("repository not found: %s", name))
		}
	}

	// Remove the file
	result = m.executor.ExecuteElevated(ctx, "rm", "-f", repoFilePath)
	if result.Failed() {
		return fmt.Errorf("failed to remove repository file: %s", result.StderrString())
	}

	return nil
}

// removePPA removes a PPA repository using add-apt-repository --remove.
func (m *Manager) removePPA(ctx context.Context, ppa string) error {
	result := m.executeElevatedWithEnv(ctx, "add-apt-repository", "--remove", "-y", ppa)

	if result.Failed() {
		stderr := result.StderrString()
		if strings.Contains(stderr, "does not exist") || strings.Contains(stderr, "not found") {
			return pkg.Wrap(pkg.ErrRepositoryNotFound, fmt.Errorf("PPA not found: %s", ppa))
		}
		return fmt.Errorf("add-apt-repository --remove failed: %s", stderr)
	}

	return nil
}

// ListRepositories returns a list of configured repositories.
// Parses /etc/apt/sources.list and /etc/apt/sources.list.d/*.list files.
func (m *Manager) ListRepositories(ctx context.Context) ([]pkg.Repository, error) {
	var allRepos []pkg.Repository

	// Read main sources.list
	result := m.executor.Execute(ctx, "cat", sourcesListPath)
	if result.Success() {
		repos, _ := parseSourcesList(result.StdoutString())
		allRepos = append(allRepos, repos...)
	}

	// Read all .list files in sources.list.d
	result = m.executor.Execute(ctx, "ls", "-1", sourcesListDir)
	if result.Success() {
		files := strings.Split(strings.TrimSpace(result.StdoutString()), "\n")
		for _, file := range files {
			if !strings.HasSuffix(file, ".list") {
				continue
			}
			path := filepath.Join(sourcesListDir, file)
			result := m.executor.Execute(ctx, "cat", path)
			if result.Success() {
				repos, _ := parseSourcesList(result.StdoutString())
				allRepos = append(allRepos, repos...)
			}
		}
	}

	return allRepos, nil
}

// EnableRepository enables a disabled repository.
// This uncomments the repository line in its configuration file.
func (m *Manager) EnableRepository(ctx context.Context, name string) error {
	return m.toggleRepository(ctx, name, true)
}

// DisableRepository disables an enabled repository.
// This comments out the repository line in its configuration file.
func (m *Manager) DisableRepository(ctx context.Context, name string) error {
	return m.toggleRepository(ctx, name, false)
}

// toggleRepository enables or disables a repository by commenting/uncommenting its entry.
func (m *Manager) toggleRepository(ctx context.Context, name string, enable bool) error {
	// Try to find the repository file
	filename := sanitizeFilename(name) + ".list"
	repoPath := filepath.Join(sourcesListDir, filename)

	// Check if file exists
	result := m.executor.Execute(ctx, "test", "-f", repoPath)
	if result.ExitCode != 0 {
		// Try the main sources.list
		repoPath = sourcesListPath
	}

	// Read the current content
	result = m.executor.Execute(ctx, "cat", repoPath)
	if result.Failed() {
		return pkg.Wrap(pkg.ErrRepositoryNotFound, fmt.Errorf("cannot read repository file: %s", repoPath))
	}

	content := result.StdoutString()
	lines := strings.Split(content, "\n")
	modified := false

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Skip empty lines
		if trimmed == "" {
			continue
		}

		if enable {
			// Enable: remove leading #
			if strings.HasPrefix(trimmed, "#") && containsRepoEntry(trimmed[1:], name) {
				lines[i] = strings.TrimPrefix(line, "#")
				lines[i] = strings.TrimPrefix(lines[i], " ")
				modified = true
			}
		} else {
			// Disable: add leading #
			if !strings.HasPrefix(trimmed, "#") && containsRepoEntry(trimmed, name) {
				lines[i] = "# " + line
				modified = true
			}
		}
	}

	if !modified {
		if enable {
			return pkg.Wrap(pkg.ErrRepositoryNotFound, fmt.Errorf("repository not found or already enabled: %s", name))
		}
		return pkg.Wrap(pkg.ErrRepositoryNotFound, fmt.Errorf("repository not found or already disabled: %s", name))
	}

	// Write the modified content
	newContent := strings.Join(lines, "\n")
	result = m.executor.ExecuteWithInput(ctx, []byte(newContent), "sudo", "tee", repoPath)
	if result.Failed() {
		return fmt.Errorf("failed to update repository file: %s", result.StderrString())
	}

	return nil
}

// containsRepoEntry checks if a line contains a repository entry matching the name.
func containsRepoEntry(line, name string) bool {
	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, "deb") {
		return false
	}
	return strings.Contains(line, name)
}

// RefreshRepositories refreshes all repository metadata.
// This is an alias for Update.
func (m *Manager) RefreshRepositories(ctx context.Context) error {
	return m.Update(ctx, pkg.DefaultUpdateOptions())
}

// AddGPGKey imports a GPG key for repository verification.
// Uses the modern approach: curl -fsSL URL | gpg --dearmor -o /etc/apt/keyrings/name.gpg
// Falls back to apt-key add if keyrings directory doesn't exist.
func (m *Manager) AddGPGKey(ctx context.Context, name string, keyURL string) error {
	// Validate inputs to prevent shell injection
	if err := validateGPGKeyInputs(name, keyURL); err != nil {
		return err
	}

	// Ensure keyrings directory exists
	result := m.executor.ExecuteElevated(ctx, "mkdir", "-p", aptKeyringsDir)
	if result.Failed() {
		// Fall back to trusted.gpg.d
		return m.addGPGKeyLegacy(ctx, name, keyURL)
	}

	// Download and dearmor the key
	keyPath := filepath.Join(aptKeyringsDir, sanitizeFilename(name)+".gpg")

	// Use a pipeline: curl | gpg --dearmor | sudo tee
	// Since we can't do pipes directly, we'll use a shell command
	// Inputs have been validated above to prevent injection
	shellCmd := fmt.Sprintf("curl -fsSL '%s' | gpg --dearmor -o '%s'", keyURL, keyPath)
	result = m.executor.ExecuteElevated(ctx, "sh", "-c", shellCmd)

	if result.Failed() {
		stderr := result.StderrString()
		if strings.Contains(stderr, "Could not resolve") || strings.Contains(stderr, "Connection refused") {
			return pkg.Wrap(pkg.ErrNetworkUnavailable, fmt.Errorf("failed to download GPG key: %s", stderr))
		}
		return fmt.Errorf("failed to add GPG key: %s", stderr)
	}

	return nil
}

// validateGPGKeyInputs validates inputs to prevent shell injection attacks.
func validateGPGKeyInputs(name, keyURL string) error {
	// Characters that could be used for shell injection
	dangerousChars := "'\"\\$`|;&()<>!"

	if strings.ContainsAny(name, dangerousChars) {
		return fmt.Errorf("invalid characters in GPG key name: contains shell metacharacters")
	}
	if strings.ContainsAny(keyURL, dangerousChars) {
		return fmt.Errorf("invalid characters in GPG key URL: contains shell metacharacters")
	}

	// Validate URL format (basic check)
	if !strings.HasPrefix(keyURL, "http://") && !strings.HasPrefix(keyURL, "https://") {
		return fmt.Errorf("invalid GPG key URL: must start with http:// or https://")
	}

	return nil
}

// addGPGKeyLegacy adds a GPG key using the legacy apt-key method.
func (m *Manager) addGPGKeyLegacy(ctx context.Context, name string, keyURL string) error {
	// Inputs already validated by AddGPGKey caller
	// Download and add with apt-key (deprecated but works everywhere)
	shellCmd := fmt.Sprintf("curl -fsSL '%s' | apt-key add -", keyURL)
	result := m.executor.ExecuteElevated(ctx, "sh", "-c", shellCmd)

	if result.Failed() {
		stderr := result.StderrString()
		if strings.Contains(stderr, "Could not resolve") || strings.Contains(stderr, "Connection refused") {
			return pkg.Wrap(pkg.ErrNetworkUnavailable, fmt.Errorf("failed to download GPG key: %s", stderr))
		}
		return fmt.Errorf("failed to add GPG key: %s", stderr)
	}

	return nil
}

// GetGPGKeyPath returns the path where a GPG key should be stored.
// Useful when adding a repository that needs to reference the key.
func (m *Manager) GetGPGKeyPath(name string) string {
	return filepath.Join(aptKeyringsDir, sanitizeFilename(name)+".gpg")
}

// sanitizeFilename removes or replaces characters that are not safe for filenames.
func sanitizeFilename(name string) string {
	// Replace problematic characters
	name = strings.ReplaceAll(name, "/", "_")
	name = strings.ReplaceAll(name, ":", "_")
	name = strings.ReplaceAll(name, " ", "_")
	name = strings.ReplaceAll(name, ".", "_")

	// Remove leading/trailing underscores
	name = strings.Trim(name, "_")

	return name
}

// FileExists checks if a file exists (utility for testing).
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
