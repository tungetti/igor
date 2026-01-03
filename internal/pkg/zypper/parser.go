package zypper

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/tungetti/igor/internal/pkg"
)

// parseRpmQuery parses rpm -qa --queryformat '%{NAME}\t%{VERSION}-%{RELEASE}\t%{ARCH}\n' output.
// Returns a list of installed packages. This format is shared with DNF/YUM.
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

// parseZypperInfo parses zypper info output.
// Returns package information for the first package found.
//
// Example output:
//
//	Information for package nginx:
//	-------------------------------
//	Repository     : openSUSE-Tumbleweed-Oss
//	Name           : nginx
//	Version        : 1.24.0-1.1
//	Arch           : x86_64
//	Vendor         : openSUSE
//	Installed Size : 1.2 MiB
//	Installed      : Yes
//	Status         : up-to-date
//	Source package : nginx-1.24.0-1.1.src
//	Summary        : A high performance web server and reverse proxy server
//	Description    : nginx is a web server...
func parseZypperInfo(output string) (*pkg.Package, error) {
	if strings.TrimSpace(output) == "" {
		return nil, nil
	}

	p := &pkg.Package{}

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "---") || strings.HasPrefix(line, "Information for") {
			continue
		}

		// Parse key: value format (zypper uses ": " with more spaces)
		colonIdx := strings.Index(line, ":")
		if colonIdx == -1 {
			continue
		}

		key := strings.TrimSpace(line[:colonIdx])
		value := strings.TrimSpace(line[colonIdx+1:])

		switch strings.ToLower(key) {
		case "name":
			if p.Name == "" {
				p.Name = value
			}
		case "version":
			if p.Version == "" {
				p.Version = value
			}
		case "arch", "architecture":
			if p.Architecture == "" {
				p.Architecture = value
			}
		case "installed size", "size":
			if p.Size == 0 {
				p.Size = parseSize(value)
			}
		case "summary", "description":
			if p.Description == "" {
				p.Description = value
			}
		case "repository":
			if p.Repository == "" {
				p.Repository = value
			}
		case "installed":
			if strings.ToLower(value) == "yes" {
				p.Installed = true
			}
		}
	}

	return p, nil
}

// parseSize parses a size string like "123 k", "1.2 M", "1.2 MiB" to bytes.
func parseSize(sizeStr string) int64 {
	sizeStr = strings.TrimSpace(strings.ToLower(sizeStr))
	if sizeStr == "" {
		return 0
	}

	// Remove common suffixes and parse
	multiplier := int64(1)

	// Handle MiB/KiB/GiB style
	if strings.HasSuffix(sizeStr, "gib") {
		multiplier = 1024 * 1024 * 1024
		sizeStr = strings.TrimSuffix(sizeStr, "gib")
	} else if strings.HasSuffix(sizeStr, "mib") {
		multiplier = 1024 * 1024
		sizeStr = strings.TrimSuffix(sizeStr, "mib")
	} else if strings.HasSuffix(sizeStr, "kib") {
		multiplier = 1024
		sizeStr = strings.TrimSuffix(sizeStr, "kib")
	} else if strings.HasSuffix(sizeStr, "g") {
		multiplier = 1024 * 1024 * 1024
		sizeStr = strings.TrimSuffix(sizeStr, "g")
	} else if strings.HasSuffix(sizeStr, "m") {
		multiplier = 1024 * 1024
		sizeStr = strings.TrimSuffix(sizeStr, "m")
	} else if strings.HasSuffix(sizeStr, "k") {
		multiplier = 1024
		sizeStr = strings.TrimSuffix(sizeStr, "k")
	}

	sizeStr = strings.TrimSpace(sizeStr)
	if val, err := strconv.ParseFloat(sizeStr, 64); err == nil {
		return int64(val * float64(multiplier))
	}

	return 0
}

// parseZypperSearch parses zypper search output.
// Format: S | Name | Summary | Type
//
// Example output:
//
//	Loading repository data...
//	Reading installed packages...
//	S | Name                      | Summary                                                  | Type
//	--+---------------------------+----------------------------------------------------------+--------
//	i | nginx                     | A high performance web server and reverse proxy server   | package
//	  | nginx-source              | Source code of nginx                                     | srcpackage
func parseZypperSearch(output string) ([]pkg.Package, error) {
	var packages []pkg.Package

	lines := strings.Split(strings.TrimSpace(output), "\n")
	inTable := false

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Skip header lines
		if strings.HasPrefix(line, "Loading") ||
			strings.HasPrefix(line, "Reading") ||
			strings.HasPrefix(line, "S |") ||
			strings.HasPrefix(line, "--+") ||
			strings.HasPrefix(line, "S  |") {
			inTable = true
			continue
		}

		if !inTable {
			continue
		}

		// Parse table row: S | Name | Summary | Type
		// S can be: i (installed), empty, v (different version available), etc.
		parts := strings.Split(line, "|")
		if len(parts) < 3 {
			continue
		}

		status := strings.TrimSpace(parts[0])
		name := strings.TrimSpace(parts[1])
		description := strings.TrimSpace(parts[2])

		if name == "" {
			continue
		}

		installed := strings.Contains(status, "i")

		packages = append(packages, pkg.Package{
			Name:        name,
			Description: description,
			Installed:   installed,
		})
	}

	return packages, nil
}

// parseZypperRepos parses zypper repos output.
// Format: # | Alias | Name | Enabled | GPG Check | Refresh
//
// Example output:
//
//	Repository priorities are without effect. All enabled repositories share the same priority.
//	# | Alias                     | Name                          | Enabled | GPG Check | Refresh
//	--+---------------------------+-------------------------------+---------+-----------+--------
//	1 | openSUSE-Tumbleweed-Oss   | openSUSE-Tumbleweed-Oss       | Yes     | (r ) Yes  | Yes
//	2 | openSUSE-Tumbleweed-Non-Oss | openSUSE-Tumbleweed-Non-Oss | Yes     | (r ) Yes  | Yes
//	3 | nvidia                    | NVIDIA Repository             | Yes     | (  ) Yes  | Yes
func parseZypperRepos(output string) ([]pkg.Repository, error) {
	var repos []pkg.Repository

	lines := strings.Split(strings.TrimSpace(output), "\n")
	inTable := false

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Skip header lines
		if strings.HasPrefix(line, "Repository priorities") ||
			strings.HasPrefix(line, "# |") ||
			strings.HasPrefix(line, "--+") {
			inTable = true
			continue
		}

		if !inTable {
			continue
		}

		// Parse table row: # | Alias | Name | Enabled | GPG Check | Refresh
		parts := strings.Split(line, "|")
		if len(parts) < 4 {
			continue
		}

		alias := strings.TrimSpace(parts[1])
		name := strings.TrimSpace(parts[2])
		enabledStr := strings.TrimSpace(parts[3])

		if alias == "" {
			continue
		}

		enabled := strings.ToLower(enabledStr) == "yes"

		repos = append(repos, pkg.Repository{
			Name:    alias,
			Enabled: enabled,
			Type:    "rpm",
		})

		// Use name if alias is different
		if name != "" && name != alias {
			repos[len(repos)-1].Name = alias // Keep alias as name for consistency
		}
	}

	return repos, nil
}

// parseZypperListUpdates parses zypper list-updates output.
// Format: S | Repository | Name | Current Version | Available Version | Arch
//
// Example output:
//
//	Loading repository data...
//	Reading installed packages...
//	S | Repository                | Name       | Current Version | Available Version | Arch
//	--+---------------------------+------------+-----------------+-------------------+--------
//	v | openSUSE-Tumbleweed-Oss   | nginx      | 1.22.0-1.1      | 1.24.0-1.1       | x86_64
//	v | openSUSE-Tumbleweed-Oss   | curl       | 8.0.0-1.1       | 8.1.0-1.1        | x86_64
func parseZypperListUpdates(output string) ([]pkg.Package, error) {
	var packages []pkg.Package

	lines := strings.Split(strings.TrimSpace(output), "\n")
	inTable := false

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Skip header lines
		if strings.HasPrefix(line, "Loading") ||
			strings.HasPrefix(line, "Reading") ||
			strings.HasPrefix(line, "S |") ||
			strings.HasPrefix(line, "--+") ||
			strings.HasPrefix(line, "No updates found") {
			inTable = true
			continue
		}

		if !inTable {
			continue
		}

		// Parse table row: S | Repository | Name | Current Version | Available Version | Arch
		parts := strings.Split(line, "|")
		if len(parts) < 5 {
			continue
		}

		repo := strings.TrimSpace(parts[1])
		name := strings.TrimSpace(parts[2])
		// currentVersion := strings.TrimSpace(parts[3]) // Not used, we want the available version
		availableVersion := strings.TrimSpace(parts[4])
		arch := ""
		if len(parts) >= 6 {
			arch = strings.TrimSpace(parts[5])
		}

		if name == "" {
			continue
		}

		packages = append(packages, pkg.Package{
			Name:         name,
			Version:      availableVersion,
			Architecture: arch,
			Repository:   repo,
			Installed:    true, // If upgradable, it's installed
		})
	}

	return packages, nil
}

// parseUnneededPackages parses zypper packages --unneeded output
// and returns a list of package names.
// Format: S | Repository | Name | Version | Arch
//
// Example output:
//
//	Loading repository data...
//	Reading installed packages...
//	S | Repository                | Name              | Version       | Arch
//	--+---------------------------+-------------------+---------------+--------
//	i | @System                   | orphan-package    | 1.0.0-1.1     | x86_64
func parseUnneededPackages(output string) []string {
	var packages []string

	lines := strings.Split(strings.TrimSpace(output), "\n")
	inTable := false

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Skip header lines
		if strings.HasPrefix(line, "Loading") ||
			strings.HasPrefix(line, "Reading") ||
			strings.HasPrefix(line, "S |") ||
			strings.HasPrefix(line, "--+") ||
			strings.HasPrefix(line, "No packages found") {
			inTable = true
			continue
		}

		if !inTable {
			continue
		}

		// Parse table row: S | Repository | Name | Version | Arch
		parts := strings.Split(line, "|")
		if len(parts) < 3 {
			continue
		}

		name := strings.TrimSpace(parts[2])
		if name == "" {
			continue
		}

		packages = append(packages, name)
	}

	return packages
}

// parseRepoFile parses a .repo file content.
// Returns a list of repositories defined in the file.
// Format is similar to YUM/DNF repo files:
//
//	[repo-id]
//	name=Repository Name
//	baseurl=https://...
//	enabled=1
//	gpgcheck=1
//	gpgkey=https://...
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

	// Autorefresh (zypper-specific)
	sb.WriteString("autorefresh=1\n")

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

// isArchitecture checks if a string looks like a package architecture.
func isArchitecture(s string) bool {
	archs := map[string]bool{
		"x86_64":  true,
		"i686":    true,
		"i386":    true,
		"i586":    true,
		"noarch":  true,
		"aarch64": true,
		"armv7hl": true,
		"ppc64le": true,
		"s390x":   true,
		"src":     true,
	}
	return archs[s]
}
