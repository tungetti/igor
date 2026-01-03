package yum

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/tungetti/igor/internal/pkg"
)

// parseRpmQuery parses rpm -qa --queryformat '%{NAME}\t%{VERSION}-%{RELEASE}\t%{ARCH}\n' output.
// Returns a list of installed packages.
// This is identical to DNF since both use rpm for package queries.
func parseRpmQuery(output string) ([]pkg.Package, error) {
	var packages []pkg.Package

	lines := strings.Split(strings.TrimSpace(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Format: NAME\tVERSION-RELEASE\tARCH
		parts := strings.Split(line, "\t")
		if len(parts) < 2 {
			continue
		}

		name := strings.TrimSpace(parts[0])
		version := ""
		arch := ""

		if len(parts) >= 2 {
			version = strings.TrimSpace(parts[1])
		}
		if len(parts) >= 3 {
			arch = strings.TrimSpace(parts[2])
		}

		packages = append(packages, pkg.Package{
			Name:         name,
			Version:      version,
			Architecture: arch,
			Installed:    true,
		})
	}

	return packages, nil
}

// parseYumInfo parses yum info output.
// Returns package information for the first package found.
// YUM output format is similar to DNF but may have slight differences.
func parseYumInfo(output string) (*pkg.Package, error) {
	if strings.TrimSpace(output) == "" {
		return nil, nil
	}

	p := &pkg.Package{}

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Parse key: value format
		colonIdx := strings.Index(line, ":")
		if colonIdx == -1 {
			continue
		}

		key := strings.TrimSpace(line[:colonIdx])
		value := strings.TrimSpace(line[colonIdx+1:])

		switch strings.ToLower(key) {
		case "name":
			if p.Name == "" { // Only take the first occurrence
				p.Name = value
			}
		case "version":
			if p.Version == "" {
				p.Version = value
			}
		case "release":
			if p.Version != "" && !strings.Contains(p.Version, "-") {
				p.Version = p.Version + "-" + value
			}
		case "architecture", "arch":
			if p.Architecture == "" {
				p.Architecture = value
			}
		case "size":
			// Parse size (usually in format like "123 k" or "1.2 M")
			if p.Size == 0 {
				p.Size = parseSize(value)
			}
		case "summary", "description":
			if p.Description == "" {
				p.Description = value
			}
		case "repository", "from repo", "repo":
			if p.Repository == "" {
				p.Repository = value
			}
		}
	}

	return p, nil
}

// parseSize parses a size string like "123 k" or "1.2 M" to bytes.
func parseSize(sizeStr string) int64 {
	sizeStr = strings.TrimSpace(strings.ToLower(sizeStr))
	if sizeStr == "" {
		return 0
	}

	// Remove common suffixes and parse
	multiplier := int64(1)
	if strings.HasSuffix(sizeStr, "k") {
		multiplier = 1024
		sizeStr = strings.TrimSuffix(sizeStr, "k")
	} else if strings.HasSuffix(sizeStr, "m") {
		multiplier = 1024 * 1024
		sizeStr = strings.TrimSuffix(sizeStr, "m")
	} else if strings.HasSuffix(sizeStr, "g") {
		multiplier = 1024 * 1024 * 1024
		sizeStr = strings.TrimSuffix(sizeStr, "g")
	}

	sizeStr = strings.TrimSpace(sizeStr)
	if val, err := strconv.ParseFloat(sizeStr, 64); err == nil {
		return int64(val * float64(multiplier))
	}

	return 0
}

// parseYumSearch parses yum search output.
// YUM search output format is similar to DNF but may have slight differences.
// Format varies but typically:
// package-name.arch : Description
// or with summary sections
func parseYumSearch(output string) ([]pkg.Package, error) {
	var packages []pkg.Package

	lines := strings.Split(strings.TrimSpace(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Skip header lines and section separators
		if strings.HasPrefix(line, "=") ||
			strings.HasPrefix(line, "Loaded plugins") ||
			strings.HasPrefix(line, "N/S Matched") ||
			strings.HasPrefix(line, "Name Exactly Matched") ||
			strings.HasPrefix(line, "Name Matched") ||
			strings.HasPrefix(line, "Summary Matched") ||
			strings.HasPrefix(line, "Warning:") {
			continue
		}

		// Format: package-name.arch : Description
		// or just: package-name : Description
		colonIdx := strings.Index(line, " : ")
		if colonIdx == -1 {
			// Try without spaces around colon
			colonIdx = strings.Index(line, ":")
			if colonIdx == -1 {
				continue
			}
		}

		namePart := strings.TrimSpace(line[:colonIdx])
		description := ""
		if colonIdx+3 < len(line) {
			description = strings.TrimSpace(line[colonIdx+3:])
		} else if colonIdx+1 < len(line) {
			description = strings.TrimSpace(line[colonIdx+1:])
		}

		// Parse name and architecture
		// Format can be: package-name.x86_64 or just package-name
		name := namePart
		arch := ""
		if dotIdx := strings.LastIndex(namePart, "."); dotIdx != -1 {
			possibleArch := namePart[dotIdx+1:]
			// Check if it looks like an architecture
			if isArchitecture(possibleArch) {
				name = namePart[:dotIdx]
				arch = possibleArch
			}
		}

		packages = append(packages, pkg.Package{
			Name:         name,
			Description:  description,
			Architecture: arch,
		})
	}

	return packages, nil
}

// isArchitecture checks if a string looks like a package architecture.
func isArchitecture(s string) bool {
	archs := map[string]bool{
		"x86_64":  true,
		"i686":    true,
		"i386":    true,
		"noarch":  true,
		"aarch64": true,
		"armv7hl": true,
		"ppc64le": true,
		"ppc64":   true,
		"s390x":   true,
		"src":     true,
	}
	return archs[s]
}

// parseYumList parses yum list output.
// Format: package-name.arch    version    repository
func parseYumList(output string) ([]pkg.Package, error) {
	var packages []pkg.Package

	lines := strings.Split(strings.TrimSpace(output), "\n")
	inPackageSection := false

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Skip header lines
		if strings.HasPrefix(line, "Loaded plugins") ||
			strings.HasPrefix(line, "Installed Packages") ||
			strings.HasPrefix(line, "Available Packages") {
			inPackageSection = true
			continue
		}

		if !inPackageSection {
			continue
		}

		// Parse package line: package-name.arch    version    repository
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		namePart := fields[0]
		version := fields[1]
		repo := ""
		if len(fields) >= 3 {
			repo = fields[2]
		}

		// Parse name and architecture
		name := namePart
		arch := ""
		if dotIdx := strings.LastIndex(namePart, "."); dotIdx != -1 {
			possibleArch := namePart[dotIdx+1:]
			if isArchitecture(possibleArch) {
				name = namePart[:dotIdx]
				arch = possibleArch
			}
		}

		packages = append(packages, pkg.Package{
			Name:         name,
			Version:      version,
			Architecture: arch,
			Repository:   repo,
		})
	}

	return packages, nil
}

// parseYumRepolist parses yum repolist all output.
// Format: repo-id              repo-name                                status
func parseYumRepolist(output string) ([]pkg.Repository, error) {
	var repos []pkg.Repository

	lines := strings.Split(strings.TrimSpace(output), "\n")
	headerPassed := false

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Skip header and metadata lines
		if strings.HasPrefix(line, "Loaded plugins") ||
			strings.HasPrefix(line, "repo id") ||
			strings.Contains(strings.ToLower(line), "repo id") {
			headerPassed = true
			continue
		}

		// Skip status line at the end
		if strings.HasPrefix(line, "repolist:") {
			continue
		}

		if !headerPassed {
			continue
		}

		// Parse repository line
		// Format can be:
		// repo-id/arch              repo-name                                status
		// repo-id                   repo-name                                enabled/disabled
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		repoID := fields[0]
		// Remove architecture suffix if present (e.g., "base/7/x86_64" -> "base")
		if slashIdx := strings.Index(repoID, "/"); slashIdx != -1 {
			repoID = repoID[:slashIdx]
		}

		enabled := true

		// Check last field for status
		lastField := strings.ToLower(fields[len(fields)-1])
		if lastField == "disabled" {
			enabled = false
		}

		repos = append(repos, pkg.Repository{
			Name:    repoID,
			URL:     "", // URL not provided in repolist output
			Enabled: enabled,
			Type:    "rpm",
		})
	}

	return repos, nil
}

// parseYumCheckUpdate parses yum check-update output.
// Format: package-name.arch    version    repository
func parseYumCheckUpdate(output string) ([]pkg.Package, error) {
	var packages []pkg.Package

	lines := strings.Split(strings.TrimSpace(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Skip header/metadata lines
		if strings.HasPrefix(line, "Loaded plugins") ||
			strings.HasPrefix(line, "Obsoleting Packages") ||
			strings.HasPrefix(line, "Security:") ||
			strings.Contains(line, "set to be") {
			continue
		}

		// Parse package line: package-name.arch    version    repository
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		namePart := fields[0]
		version := fields[1]
		repo := ""
		if len(fields) >= 3 {
			repo = fields[2]
		}

		// Parse name and architecture
		name := namePart
		arch := ""
		if dotIdx := strings.LastIndex(namePart, "."); dotIdx != -1 {
			possibleArch := namePart[dotIdx+1:]
			if isArchitecture(possibleArch) {
				name = namePart[:dotIdx]
				arch = possibleArch
			}
		}

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

// parseRepoFile parses a .repo file content.
// Returns a list of repositories defined in the file.
func parseRepoFile(content string) ([]pkg.Repository, error) {
	var repos []pkg.Repository
	var currentRepo *pkg.Repository

	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Skip comments and empty lines
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Section header [repo-id]
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			// Save previous repo if exists
			if currentRepo != nil {
				repos = append(repos, *currentRepo)
			}
			repoID := line[1 : len(line)-1]
			currentRepo = &pkg.Repository{
				Name:    repoID,
				Enabled: true, // Default to enabled
				Type:    "rpm",
			}
			continue
		}

		// Key=value pairs
		if currentRepo != nil {
			if eqIdx := strings.Index(line, "="); eqIdx != -1 {
				key := strings.TrimSpace(line[:eqIdx])
				value := strings.TrimSpace(line[eqIdx+1:])

				switch strings.ToLower(key) {
				case "name":
					// Keep the ID as Name, but store display name elsewhere if needed
				case "baseurl":
					currentRepo.URL = value
				case "enabled":
					currentRepo.Enabled = value == "1"
				case "gpgkey":
					currentRepo.GPGKey = value
				}
			}
		}
	}

	// Don't forget the last repo
	if currentRepo != nil {
		repos = append(repos, *currentRepo)
	}

	return repos, nil
}

// buildRepoFileContent creates a .repo file content from a Repository.
func buildRepoFileContent(repo pkg.Repository) string {
	var sb strings.Builder

	// Section header
	repoID := sanitizeRepoID(repo.Name)
	sb.WriteString("[")
	sb.WriteString(repoID)
	sb.WriteString("]\n")

	// Name
	sb.WriteString("name=")
	if repo.Name != "" {
		sb.WriteString(repo.Name)
	} else {
		sb.WriteString(repoID)
	}
	sb.WriteString("\n")

	// Base URL
	if repo.URL != "" {
		sb.WriteString("baseurl=")
		sb.WriteString(repo.URL)
		sb.WriteString("\n")
	}

	// Enabled
	if repo.Enabled {
		sb.WriteString("enabled=1\n")
	} else {
		sb.WriteString("enabled=0\n")
	}

	// GPG settings
	if repo.GPGKey != "" {
		sb.WriteString("gpgcheck=1\n")
		sb.WriteString("gpgkey=")
		sb.WriteString(repo.GPGKey)
		sb.WriteString("\n")
	} else {
		sb.WriteString("gpgcheck=0\n")
	}

	return sb.String()
}

// sanitizeRepoID creates a valid repository ID from a name.
func sanitizeRepoID(name string) string {
	// Replace problematic characters
	name = strings.ReplaceAll(name, " ", "-")
	name = strings.ReplaceAll(name, "/", "-")
	name = strings.ReplaceAll(name, ":", "-")
	name = strings.ReplaceAll(name, ".", "-")

	// Remove leading/trailing dashes
	name = strings.Trim(name, "-")

	// Use regex to remove any remaining invalid characters
	reg := regexp.MustCompile(`[^a-zA-Z0-9_-]`)
	name = reg.ReplaceAllString(name, "")

	if name == "" {
		name = "custom-repo"
	}

	return name
}
