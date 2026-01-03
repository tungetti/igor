package pacman

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/tungetti/igor/internal/pkg"
)

// parsePacmanQ parses pacman -Q output.
// Format: package-name version
// Example: linux 6.6.1.arch1-1
func parsePacmanQ(output string) ([]pkg.Package, error) {
	var packages []pkg.Package

	lines := strings.Split(strings.TrimSpace(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Format: package-name version
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}

		name := parts[0]
		version := parts[1]

		packages = append(packages, pkg.Package{
			Name:      name,
			Version:   version,
			Installed: true,
		})
	}

	return packages, nil
}

// parsePacmanQi parses pacman -Qi output (local package info).
// This is the detailed query format for installed packages.
func parsePacmanQi(output string) (*pkg.Package, error) {
	return parsePacmanInfo(output)
}

// parsePacmanSi parses pacman -Si output (remote package info).
// This is the detailed query format for sync database packages.
func parsePacmanSi(output string) (*pkg.Package, error) {
	return parsePacmanInfo(output)
}

// parsePacmanInfo parses pacman info output (both -Qi and -Si).
// The format is the same for both local and remote queries.
func parsePacmanInfo(output string) (*pkg.Package, error) {
	if strings.TrimSpace(output) == "" {
		return nil, nil
	}

	p := &pkg.Package{}
	var currentKey string
	var multilineValue strings.Builder

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		// Check if this is a key: value line
		colonIdx := strings.Index(line, ":")
		if colonIdx != -1 && !strings.HasPrefix(line, " ") {
			// Save previous multiline value if any
			if currentKey == "Description" && p.Description == "" {
				p.Description = strings.TrimSpace(multilineValue.String())
			}

			key := strings.TrimSpace(line[:colonIdx])
			value := strings.TrimSpace(line[colonIdx+1:])

			currentKey = key
			multilineValue.Reset()

			switch key {
			case "Name":
				if p.Name == "" {
					p.Name = value
				}
			case "Version":
				if p.Version == "" {
					p.Version = value
				}
			case "Architecture":
				if p.Architecture == "" {
					p.Architecture = value
				}
			case "Repository":
				if p.Repository == "" {
					p.Repository = value
				}
			case "Description":
				if p.Description == "" {
					p.Description = value
				}
			case "Installed Size", "Download Size":
				if p.Size == 0 {
					p.Size = parseSize(value)
				}
			case "Depends On":
				if len(p.Dependencies) == 0 && value != "None" {
					deps := strings.Fields(value)
					for _, dep := range deps {
						// Remove version constraints like >=1.0
						if idx := strings.IndexAny(dep, ">=<"); idx != -1 {
							dep = dep[:idx]
						}
						p.Dependencies = append(p.Dependencies, dep)
					}
				}
			}
		} else if strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t") {
			// Continuation of previous value (multiline)
			multilineValue.WriteString(" ")
			multilineValue.WriteString(strings.TrimSpace(line))
		}
	}

	// Handle any remaining multiline value
	if currentKey == "Description" && p.Description == "" {
		p.Description = strings.TrimSpace(multilineValue.String())
	}

	return p, nil
}

// parsePacmanSs parses pacman -Ss output (search results).
// Format:
// repo/package-name version [installed]
//
//	Description text
func parsePacmanSs(output string) ([]pkg.Package, error) {
	var packages []pkg.Package

	lines := strings.Split(strings.TrimSpace(output), "\n")
	var currentPkg *pkg.Package

	for _, line := range lines {
		if line == "" {
			continue
		}

		// Check if this is a package header line (starts with repo/name)
		if !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") {
			// Save previous package if exists
			if currentPkg != nil {
				packages = append(packages, *currentPkg)
			}

			// Parse header line: repo/package-name version [installed]
			// Example: extra/nvidia-utils 545.29.06-1 [installed]
			currentPkg = &pkg.Package{}

			// Check if installed
			if strings.Contains(line, "[installed") {
				currentPkg.Installed = true
				// Remove the [installed] part
				line = regexp.MustCompile(`\s*\[installed.*?\]`).ReplaceAllString(line, "")
			}

			parts := strings.Fields(line)
			if len(parts) >= 1 {
				// Parse repo/name
				repoPkg := parts[0]
				if slashIdx := strings.Index(repoPkg, "/"); slashIdx != -1 {
					currentPkg.Repository = repoPkg[:slashIdx]
					currentPkg.Name = repoPkg[slashIdx+1:]
				} else {
					currentPkg.Name = repoPkg
				}
			}
			if len(parts) >= 2 {
				currentPkg.Version = parts[1]
			}
		} else if currentPkg != nil {
			// This is a description line (indented)
			desc := strings.TrimSpace(line)
			if currentPkg.Description == "" {
				currentPkg.Description = desc
			} else {
				currentPkg.Description += " " + desc
			}
		}
	}

	// Don't forget the last package
	if currentPkg != nil {
		packages = append(packages, *currentPkg)
	}

	return packages, nil
}

// parsePacmanQu parses pacman -Qu output (upgradable packages).
// Format: package-name old-version -> new-version
// Example: linux 6.6.0.arch1-1 -> 6.6.1.arch1-1
func parsePacmanQu(output string) ([]pkg.Package, error) {
	var packages []pkg.Package

	lines := strings.Split(strings.TrimSpace(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Format: package-name old-version -> new-version
		parts := strings.Fields(line)
		if len(parts) < 4 {
			// Simple format without arrow: package-name version
			if len(parts) >= 2 {
				packages = append(packages, pkg.Package{
					Name:      parts[0],
					Version:   parts[1],
					Installed: true,
				})
			}
			continue
		}

		// parts[0] = name, parts[1] = old-version, parts[2] = "->", parts[3] = new-version
		name := parts[0]
		newVersion := ""
		if len(parts) >= 4 && parts[2] == "->" {
			newVersion = parts[3]
		} else {
			newVersion = parts[1]
		}

		packages = append(packages, pkg.Package{
			Name:      name,
			Version:   newVersion,
			Installed: true, // If upgradable, it's installed
		})
	}

	return packages, nil
}

// parsePacmanConf parses /etc/pacman.conf for repositories.
// Returns a list of configured repositories.
func parsePacmanConf(content string) ([]pkg.Repository, error) {
	var repos []pkg.Repository
	var currentRepo *pkg.Repository

	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Skip comments and empty lines
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Section header [repo-name]
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			// Save previous repo if exists and is not [options]
			if currentRepo != nil && currentRepo.Name != "options" {
				repos = append(repos, *currentRepo)
			}
			repoName := line[1 : len(line)-1]
			currentRepo = &pkg.Repository{
				Name:    repoName,
				Enabled: true, // Repositories in pacman.conf are enabled by default
				Type:    "pacman",
			}
			continue
		}

		// Key = value pairs
		if currentRepo != nil && currentRepo.Name != "options" {
			if eqIdx := strings.Index(line, "="); eqIdx != -1 {
				key := strings.TrimSpace(line[:eqIdx])
				value := strings.TrimSpace(line[eqIdx+1:])

				switch strings.ToLower(key) {
				case "server":
					if currentRepo.URL == "" {
						currentRepo.URL = value
					}
				case "include":
					// Include directives point to mirrorlist files
					if currentRepo.URL == "" {
						currentRepo.URL = value // Store the include path
					}
				case "siglevel":
					// Could parse signature level here if needed
				}
			}
		}
	}

	// Don't forget the last repo
	if currentRepo != nil && currentRepo.Name != "options" {
		repos = append(repos, *currentRepo)
	}

	return repos, nil
}

// parseSize parses a size string like "123.45 KiB" or "1.2 MiB" to bytes.
func parseSize(sizeStr string) int64 {
	sizeStr = strings.TrimSpace(sizeStr)
	if sizeStr == "" {
		return 0
	}

	// Pacman uses KiB, MiB, GiB, etc.
	multiplier := int64(1)
	lowerStr := strings.ToLower(sizeStr)

	if strings.Contains(lowerStr, "kib") || strings.HasSuffix(lowerStr, "k") {
		multiplier = 1024
	} else if strings.Contains(lowerStr, "mib") || strings.HasSuffix(lowerStr, "m") {
		multiplier = 1024 * 1024
	} else if strings.Contains(lowerStr, "gib") || strings.HasSuffix(lowerStr, "g") {
		multiplier = 1024 * 1024 * 1024
	} else if strings.Contains(lowerStr, "tib") || strings.HasSuffix(lowerStr, "t") {
		multiplier = 1024 * 1024 * 1024 * 1024
	}

	// Extract numeric value
	numStr := regexp.MustCompile(`[\d.]+`).FindString(sizeStr)
	if numStr == "" {
		return 0
	}

	if val, err := strconv.ParseFloat(numStr, 64); err == nil {
		return int64(val * float64(multiplier))
	}

	return 0
}

// sanitizeRepoName creates a valid repository name for file operations.
func sanitizeRepoName(name string) string {
	// Replace problematic characters
	name = strings.ReplaceAll(name, " ", "-")
	name = strings.ReplaceAll(name, "/", "-")
	name = strings.ReplaceAll(name, ":", "-")

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
