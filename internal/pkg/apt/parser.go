package apt

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/tungetti/igor/internal/pkg"
)

// parseDpkgQuery parses the output of dpkg-query -W -f='${Package}\t${Version}\t${Status}\n'.
// Returns a list of installed packages.
func parseDpkgQuery(output string) ([]pkg.Package, error) {
	var packages []pkg.Package

	lines := strings.Split(strings.TrimSpace(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Format: package\tversion\tstatus
		parts := strings.Split(line, "\t")
		if len(parts) < 3 {
			continue
		}

		name := strings.TrimSpace(parts[0])
		version := strings.TrimSpace(parts[1])
		status := strings.TrimSpace(parts[2])

		// Only include packages that are fully installed
		if !strings.Contains(status, "install ok installed") {
			continue
		}

		packages = append(packages, pkg.Package{
			Name:      name,
			Version:   version,
			Installed: true,
		})
	}

	return packages, nil
}

// parseAptCacheShow parses the output of apt-cache show.
// Returns package information for the first package found.
func parseAptCacheShow(output string) (*pkg.Package, error) {
	if strings.TrimSpace(output) == "" {
		return nil, nil
	}

	p := &pkg.Package{}
	var deps []string

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		// Stop at first empty line (separates multiple versions)
		if strings.TrimSpace(line) == "" && p.Name != "" {
			break
		}

		if strings.HasPrefix(line, "Package:") {
			p.Name = strings.TrimSpace(strings.TrimPrefix(line, "Package:"))
		} else if strings.HasPrefix(line, "Version:") {
			p.Version = strings.TrimSpace(strings.TrimPrefix(line, "Version:"))
		} else if strings.HasPrefix(line, "Description:") {
			p.Description = strings.TrimSpace(strings.TrimPrefix(line, "Description:"))
		} else if strings.HasPrefix(line, "Architecture:") {
			p.Architecture = strings.TrimSpace(strings.TrimPrefix(line, "Architecture:"))
		} else if strings.HasPrefix(line, "Installed-Size:") {
			sizeStr := strings.TrimSpace(strings.TrimPrefix(line, "Installed-Size:"))
			if size, err := strconv.ParseInt(sizeStr, 10, 64); err == nil {
				// Installed-Size is in KB, convert to bytes
				p.Size = size * 1024
			}
		} else if strings.HasPrefix(line, "Section:") {
			// Use section as repository info if no explicit source
			if p.Repository == "" {
				p.Repository = strings.TrimSpace(strings.TrimPrefix(line, "Section:"))
			}
		} else if strings.HasPrefix(line, "Depends:") {
			depStr := strings.TrimSpace(strings.TrimPrefix(line, "Depends:"))
			deps = append(deps, parseDependencies(depStr)...)
		}
	}

	p.Dependencies = deps
	return p, nil
}

// parseDependencies parses a dependency string from apt-cache show.
// Format: pkg1, pkg2 (>= version), pkg3 | pkg4
func parseDependencies(depStr string) []string {
	var deps []string

	// Split by comma first
	parts := strings.Split(depStr, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		// Handle alternatives (pkg1 | pkg2) - just take the first one
		if idx := strings.Index(part, "|"); idx != -1 {
			part = strings.TrimSpace(part[:idx])
		}

		// Remove version constraint
		if idx := strings.Index(part, "("); idx != -1 {
			part = strings.TrimSpace(part[:idx])
		}

		// Remove :any suffix
		if idx := strings.Index(part, ":"); idx != -1 {
			part = part[:idx]
		}

		if part != "" {
			deps = append(deps, part)
		}
	}

	return deps
}

// parseAptCacheSearch parses the output of apt-cache search.
// Format: package-name - description
func parseAptCacheSearch(output string) ([]pkg.Package, error) {
	var packages []pkg.Package

	lines := strings.Split(strings.TrimSpace(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Format: package-name - description
		// Note: package name can contain hyphens, so we look for " - "
		idx := strings.Index(line, " - ")
		if idx == -1 {
			// Fallback: just use the whole line as package name
			packages = append(packages, pkg.Package{
				Name: line,
			})
			continue
		}

		name := strings.TrimSpace(line[:idx])
		description := strings.TrimSpace(line[idx+3:])

		packages = append(packages, pkg.Package{
			Name:        name,
			Description: description,
		})
	}

	return packages, nil
}

// parseSourcesList parses sources.list format content.
// Format: deb [options] URL distribution [components...]
// Also supports: deb-src for source packages
// Commented lines starting with #deb or # deb are parsed as disabled repos.
func parseSourcesList(content string) ([]pkg.Repository, error) {
	var repos []pkg.Repository

	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Skip empty lines
		if line == "" {
			continue
		}

		// Skip pure comments (not disabled repos)
		// Disabled repos look like: #deb ... or # deb ...
		if strings.HasPrefix(line, "#") {
			// Check if this is a disabled repo (commented out deb line)
			trimmed := strings.TrimPrefix(line, "#")
			trimmed = strings.TrimSpace(trimmed)
			if !strings.HasPrefix(trimmed, "deb") {
				// This is a regular comment, skip it
				continue
			}
			// This is a disabled repo, parse it
		}

		repo, err := parseSourcesListLine(line)
		if err != nil {
			continue // Skip malformed lines
		}
		if repo != nil {
			repos = append(repos, *repo)
		}
	}

	return repos, nil
}

// parseSourcesListLine parses a single sources.list line.
func parseSourcesListLine(line string) (*pkg.Repository, error) {
	// Check for disabled lines (commented out but with # followed by deb)
	enabled := true
	if strings.HasPrefix(line, "#") {
		enabled = false
		line = strings.TrimPrefix(line, "#")
		line = strings.TrimSpace(line)
	}

	// Must start with deb or deb-src
	if !strings.HasPrefix(line, "deb ") && !strings.HasPrefix(line, "deb-src ") {
		return nil, nil
	}

	parts := strings.Fields(line)
	if len(parts) < 3 {
		return nil, nil
	}

	repoType := parts[0]
	idx := 1

	// Check for options in brackets [arch=amd64 signed-by=/path/to/key.gpg]
	var gpgKey string
	if strings.HasPrefix(parts[idx], "[") {
		// Find the closing bracket
		optStr := ""
		for ; idx < len(parts); idx++ {
			optStr += parts[idx] + " "
			if strings.Contains(parts[idx], "]") {
				idx++
				break
			}
		}
		// Extract signed-by option if present
		gpgKey = extractOption(optStr, "signed-by")
	}

	if idx >= len(parts) {
		return nil, nil
	}

	url := parts[idx]
	idx++

	var dist string
	if idx < len(parts) {
		dist = parts[idx]
		idx++
	}

	var components []string
	for ; idx < len(parts); idx++ {
		components = append(components, parts[idx])
	}

	// Generate a name from the URL
	name := generateRepoName(url, dist)

	return &pkg.Repository{
		Name:         name,
		URL:          url,
		Enabled:      enabled,
		GPGKey:       gpgKey,
		Type:         repoType,
		Distribution: dist,
		Components:   components,
	}, nil
}

// extractOption extracts a value from bracketed options.
// Format: [arch=amd64 signed-by=/path/to/key.gpg]
func extractOption(optStr, key string) string {
	// Look for key=value pattern
	pattern := regexp.MustCompile(key + `=([^\s\]]+)`)
	matches := pattern.FindStringSubmatch(optStr)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

// generateRepoName generates a repository name from URL and distribution.
func generateRepoName(url, dist string) string {
	// Extract hostname from URL
	name := url
	if idx := strings.Index(url, "://"); idx != -1 {
		name = url[idx+3:]
	}
	if idx := strings.Index(name, "/"); idx != -1 {
		name = name[:idx]
	}

	// Replace dots with underscores
	name = strings.ReplaceAll(name, ".", "_")

	if dist != "" {
		name = name + "_" + dist
	}

	return name
}

// parseAptListUpgradable parses the output of apt list --upgradable.
// Format: package/distribution version arch [upgradable from: old-version]
func parseAptListUpgradable(output string) ([]pkg.Package, error) {
	var packages []pkg.Package

	lines := strings.Split(strings.TrimSpace(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Skip the "Listing..." header line
		if strings.HasPrefix(line, "Listing...") {
			continue
		}

		// Format: package/distribution version arch [upgradable from: old-version]
		// Example: nginx/jammy-updates 1.22.0-1ubuntu1.1 amd64 [upgradable from: 1.22.0-1ubuntu1]

		// Split by space to get parts
		parts := strings.Fields(line)
		if len(parts) < 3 {
			continue
		}

		// First part is package/distribution
		pkgPart := parts[0]
		slashIdx := strings.Index(pkgPart, "/")
		var name, repo string
		if slashIdx != -1 {
			name = pkgPart[:slashIdx]
			repo = pkgPart[slashIdx+1:]
		} else {
			name = pkgPart
		}

		version := parts[1]
		arch := parts[2]

		packages = append(packages, pkg.Package{
			Name:         name,
			Version:      version,
			Architecture: arch,
			Repository:   repo,
			Installed:    true, // If upgradable, it's installed
		})
	}

	return packages, nil
}
