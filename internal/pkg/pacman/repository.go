package pacman

import (
	"context"
	"fmt"
	"strings"

	"github.com/tungetti/igor/internal/pkg"
)

// Repository paths for Pacman.
const (
	pacmanConfPath = "/etc/pacman.conf"
)

// AddRepository adds a new package repository to pacman.conf.
// For Pacman, repositories are configured in /etc/pacman.conf.
// Note: This is a simplified implementation. For full control, users should
// edit pacman.conf directly or use pacman.d includes.
func (m *Manager) AddRepository(ctx context.Context, repo pkg.Repository) error {
	if repo.Name == "" {
		return fmt.Errorf("repository name is required")
	}

	// Check if repository already exists
	repos, err := m.ListRepositories(ctx)
	if err == nil {
		for _, existingRepo := range repos {
			if existingRepo.Name == repo.Name {
				return pkg.Wrap(pkg.ErrRepositoryExists, fmt.Errorf("repository already exists: %s", repo.Name))
			}
		}
	}

	// Build the repository configuration block
	var repoBlock strings.Builder
	repoBlock.WriteString("\n# Added by Igor\n")
	repoBlock.WriteString(fmt.Sprintf("[%s]\n", repo.Name))

	if repo.GPGKey != "" {
		repoBlock.WriteString(fmt.Sprintf("SigLevel = Required TrustedOnly\n"))
	} else {
		repoBlock.WriteString("SigLevel = Optional TrustAll\n")
	}

	if repo.URL != "" {
		repoBlock.WriteString(fmt.Sprintf("Server = %s\n", repo.URL))
	}

	// Append the repository to pacman.conf using tee
	result := m.executor.ExecuteWithInput(
		ctx,
		[]byte(repoBlock.String()),
		"sudo",
		"tee",
		"-a",
		pacmanConfPath,
	)

	if result.Failed() {
		return fmt.Errorf("failed to add repository to pacman.conf: %s", result.StderrString())
	}

	// Import GPG key if provided
	if repo.GPGKey != "" {
		if err := m.ImportGPGKey(ctx, repo.GPGKey); err != nil {
			// Don't fail completely, just warn
			// The user might need to handle this manually
			_ = err
		}
	}

	return nil
}

// RemoveRepository removes a package repository from pacman.conf.
// This is a simplified implementation that comments out the repository section.
func (m *Manager) RemoveRepository(ctx context.Context, name string) error {
	// Read current pacman.conf
	result := m.executor.Execute(ctx, "cat", pacmanConfPath)
	if result.Failed() {
		return fmt.Errorf("failed to read pacman.conf: %s", result.StderrString())
	}

	content := result.StdoutString()
	lines := strings.Split(content, "\n")
	var newContent strings.Builder
	inTargetRepo := false
	repoFound := false

	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)

		// Check if we're entering a new section
		if strings.HasPrefix(trimmedLine, "[") && strings.HasSuffix(trimmedLine, "]") {
			sectionName := trimmedLine[1 : len(trimmedLine)-1]
			if sectionName == name {
				inTargetRepo = true
				repoFound = true
				// Comment out this section header
				newContent.WriteString("# ")
				newContent.WriteString(line)
				newContent.WriteString("\n")
				continue
			} else {
				inTargetRepo = false
			}
		}

		if inTargetRepo && trimmedLine != "" && !strings.HasPrefix(trimmedLine, "#") {
			// Comment out lines in the target repo section
			newContent.WriteString("# ")
			newContent.WriteString(line)
			newContent.WriteString("\n")
		} else {
			newContent.WriteString(line)
			newContent.WriteString("\n")
		}
	}

	if !repoFound {
		return pkg.Wrap(pkg.ErrRepositoryNotFound, fmt.Errorf("repository not found: %s", name))
	}

	// Write the modified content back
	result = m.executor.ExecuteWithInput(
		ctx,
		[]byte(newContent.String()),
		"sudo",
		"tee",
		pacmanConfPath,
	)

	if result.Failed() {
		return fmt.Errorf("failed to update pacman.conf: %s", result.StderrString())
	}

	return nil
}

// ListRepositories returns a list of configured repositories.
// Parses /etc/pacman.conf to extract repository information.
func (m *Manager) ListRepositories(ctx context.Context) ([]pkg.Repository, error) {
	result := m.executor.Execute(ctx, "cat", pacmanConfPath)

	if result.Failed() {
		return nil, fmt.Errorf("failed to read pacman.conf: %s", result.StderrString())
	}

	return parsePacmanConf(result.StdoutString())
}

// EnableRepository enables a repository in pacman.conf.
// This uncomments a previously commented repository section.
func (m *Manager) EnableRepository(ctx context.Context, name string) error {
	// Read current pacman.conf
	result := m.executor.Execute(ctx, "cat", pacmanConfPath)
	if result.Failed() {
		return fmt.Errorf("failed to read pacman.conf: %s", result.StderrString())
	}

	content := result.StdoutString()
	lines := strings.Split(content, "\n")
	var newContent strings.Builder
	inTargetRepo := false
	repoFound := false

	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)

		// Check for commented section header
		commentedSection := fmt.Sprintf("# [%s]", name)
		if strings.Contains(trimmedLine, commentedSection) || trimmedLine == fmt.Sprintf("#[%s]", name) {
			inTargetRepo = true
			repoFound = true
			// Uncomment this line
			newContent.WriteString(strings.TrimPrefix(strings.TrimPrefix(line, "# "), "#"))
			newContent.WriteString("\n")
			continue
		}

		// Check for active section (already enabled)
		if trimmedLine == fmt.Sprintf("[%s]", name) {
			// Already enabled
			return nil
		}

		// Check if we're entering a different section
		if strings.HasPrefix(trimmedLine, "[") || strings.HasPrefix(trimmedLine, "# [") {
			if !strings.Contains(trimmedLine, name) {
				inTargetRepo = false
			}
		}

		if inTargetRepo && strings.HasPrefix(trimmedLine, "#") {
			// Uncomment lines in the target repo section (but not actual comments)
			// Check if this looks like a config line
			withoutComment := strings.TrimPrefix(strings.TrimPrefix(trimmedLine, "# "), "#")
			if strings.Contains(withoutComment, "=") || strings.HasPrefix(strings.ToLower(withoutComment), "siglevel") {
				newContent.WriteString(withoutComment)
				newContent.WriteString("\n")
				continue
			}
		}

		newContent.WriteString(line)
		newContent.WriteString("\n")
	}

	if !repoFound {
		return pkg.Wrap(pkg.ErrRepositoryNotFound, fmt.Errorf("repository not found: %s", name))
	}

	// Write the modified content back
	result = m.executor.ExecuteWithInput(
		ctx,
		[]byte(newContent.String()),
		"sudo",
		"tee",
		pacmanConfPath,
	)

	if result.Failed() {
		return fmt.Errorf("failed to update pacman.conf: %s", result.StderrString())
	}

	return nil
}

// DisableRepository disables a repository in pacman.conf.
// This comments out the repository section.
func (m *Manager) DisableRepository(ctx context.Context, name string) error {
	// Read current pacman.conf
	result := m.executor.Execute(ctx, "cat", pacmanConfPath)
	if result.Failed() {
		return fmt.Errorf("failed to read pacman.conf: %s", result.StderrString())
	}

	content := result.StdoutString()
	lines := strings.Split(content, "\n")
	var newContent strings.Builder
	inTargetRepo := false
	repoFound := false

	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)

		// Check if we're entering a new section
		if strings.HasPrefix(trimmedLine, "[") && strings.HasSuffix(trimmedLine, "]") {
			sectionName := trimmedLine[1 : len(trimmedLine)-1]
			if sectionName == name {
				inTargetRepo = true
				repoFound = true
				// Comment out this section header
				newContent.WriteString("# ")
				newContent.WriteString(line)
				newContent.WriteString("\n")
				continue
			} else {
				inTargetRepo = false
			}
		}

		// Check for already commented section
		if strings.Contains(trimmedLine, fmt.Sprintf("[%s]", name)) && strings.HasPrefix(trimmedLine, "#") {
			// Already disabled
			return nil
		}

		if inTargetRepo && trimmedLine != "" && !strings.HasPrefix(trimmedLine, "#") {
			// Comment out lines in the target repo section
			newContent.WriteString("# ")
			newContent.WriteString(line)
			newContent.WriteString("\n")
		} else {
			newContent.WriteString(line)
			newContent.WriteString("\n")
		}
	}

	if !repoFound {
		return pkg.Wrap(pkg.ErrRepositoryNotFound, fmt.Errorf("repository not found: %s", name))
	}

	// Write the modified content back
	result = m.executor.ExecuteWithInput(
		ctx,
		[]byte(newContent.String()),
		"sudo",
		"tee",
		pacmanConfPath,
	)

	if result.Failed() {
		return fmt.Errorf("failed to update pacman.conf: %s", result.StderrString())
	}

	return nil
}

// RefreshRepositories refreshes the package database.
// Uses pacman -Sy.
func (m *Manager) RefreshRepositories(ctx context.Context) error {
	result := m.executor.ExecuteElevated(ctx, "pacman", "-Sy")

	if result.Failed() {
		stderr := result.StderrString()
		if strings.Contains(stderr, "failed to retrieve") ||
			strings.Contains(stderr, "failed to download") ||
			strings.Contains(stderr, "error: failed to synchronize") {
			return pkg.Wrap(pkg.ErrNetworkUnavailable, fmt.Errorf("pacman -Sy failed: %s", stderr))
		}
		return fmt.Errorf("pacman -Sy failed: %s", stderr)
	}

	return nil
}

// ImportGPGKey imports a GPG key for package signing verification.
// Uses pacman-key --recv-keys and pacman-key --lsign-key.
func (m *Manager) ImportGPGKey(ctx context.Context, keyID string) error {
	// Receive the key
	result := m.executor.ExecuteElevated(ctx, "pacman-key", "--recv-keys", keyID)

	if result.Failed() {
		stderr := result.StderrString()
		if strings.Contains(stderr, "keyserver receive failed") ||
			strings.Contains(stderr, "No data") {
			return pkg.Wrap(pkg.ErrNetworkUnavailable, fmt.Errorf("failed to receive GPG key: %s", stderr))
		}
		return fmt.Errorf("pacman-key --recv-keys failed: %s", stderr)
	}

	// Locally sign the key
	result = m.executor.ExecuteElevated(ctx, "pacman-key", "--lsign-key", keyID)

	if result.Failed() {
		return fmt.Errorf("pacman-key --lsign-key failed: %s", result.StderrString())
	}

	return nil
}

// InitializeKeyring initializes the pacman keyring.
// Uses pacman-key --init.
func (m *Manager) InitializeKeyring(ctx context.Context) error {
	result := m.executor.ExecuteElevated(ctx, "pacman-key", "--init")

	if result.Failed() {
		return fmt.Errorf("pacman-key --init failed: %s", result.StderrString())
	}

	return nil
}

// PopulateKeyring populates the pacman keyring with official keys.
// Uses pacman-key --populate archlinux.
func (m *Manager) PopulateKeyring(ctx context.Context, distro string) error {
	if distro == "" {
		distro = "archlinux"
	}

	result := m.executor.ExecuteElevated(ctx, "pacman-key", "--populate", distro)

	if result.Failed() {
		return fmt.Errorf("pacman-key --populate failed: %s", result.StderrString())
	}

	return nil
}

// RefreshKeys refreshes all expired keys.
// Uses pacman-key --refresh-keys.
func (m *Manager) RefreshKeys(ctx context.Context) error {
	result := m.executor.ExecuteElevated(ctx, "pacman-key", "--refresh-keys")

	if result.Failed() {
		stderr := result.StderrString()
		if strings.Contains(stderr, "keyserver receive failed") ||
			strings.Contains(stderr, "No data") {
			return pkg.Wrap(pkg.ErrNetworkUnavailable, fmt.Errorf("failed to refresh keys: %s", stderr))
		}
		return fmt.Errorf("pacman-key --refresh-keys failed: %s", stderr)
	}

	return nil
}
