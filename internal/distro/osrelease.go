package distro

import (
	"bufio"
	"os"
	"strings"

	"github.com/tungetti/igor/internal/errors"
)

// Standard paths for os-release file
const (
	primaryOSReleasePath   = "/etc/os-release"
	secondaryOSReleasePath = "/usr/lib/os-release"
)

// ParseOSRelease reads and parses /etc/os-release from the filesystem.
// It tries the primary path first, then falls back to the secondary path.
func ParseOSRelease(path string) (*Distribution, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, errors.Wrap(errors.NotFound, "failed to read os-release file", err).WithOp("distro.ParseOSRelease")
	}
	return ParseOSReleaseContent(string(content))
}

// ParseOSReleaseContent parses os-release content from a string.
// This is useful for testing without filesystem access.
func ParseOSReleaseContent(content string) (*Distribution, error) {
	dist := &Distribution{}
	scanner := bufio.NewScanner(strings.NewReader(content))

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse KEY=VALUE format
		key, value, found := strings.Cut(line, "=")
		if !found {
			continue
		}

		// Remove quotes from value
		value = unquote(value)

		switch key {
		case "ID":
			dist.ID = strings.ToLower(value)
		case "NAME":
			dist.Name = value
		case "VERSION":
			dist.Version = value
		case "VERSION_ID":
			dist.VersionID = value
		case "VERSION_CODENAME":
			dist.VersionCodename = value
		case "PRETTY_NAME":
			dist.PrettyName = value
		case "ID_LIKE":
			dist.IDLike = parseIDLike(value)
		case "HOME_URL":
			dist.HomeURL = value
		case "SUPPORT_URL":
			dist.SupportURL = value
		case "BUILD_ID":
			dist.BuildID = value
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, errors.Wrap(errors.DistroDetection, "failed to parse os-release content", err).WithOp("distro.ParseOSReleaseContent")
	}

	// Determine the distribution family
	dist.Family = DetectFamily(dist.ID, dist.IDLike)

	return dist, nil
}

// unquote removes surrounding quotes (single or double) from a string.
func unquote(s string) string {
	s = strings.TrimSpace(s)
	if len(s) < 2 {
		return s
	}

	// Check for double quotes
	if s[0] == '"' && s[len(s)-1] == '"' {
		return s[1 : len(s)-1]
	}

	// Check for single quotes
	if s[0] == '\'' && s[len(s)-1] == '\'' {
		return s[1 : len(s)-1]
	}

	return s
}

// parseIDLike parses the ID_LIKE field which contains space-separated IDs.
func parseIDLike(value string) []string {
	if value == "" {
		return nil
	}

	parts := strings.Fields(value)
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.ToLower(strings.TrimSpace(part))
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// ParseLSBRelease parses the /etc/lsb-release file format.
// This is an older format used by some Ubuntu systems.
func ParseLSBRelease(path string) (*Distribution, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, errors.Wrap(errors.NotFound, "failed to read lsb-release file", err).WithOp("distro.ParseLSBRelease")
	}
	return ParseLSBReleaseContent(string(content))
}

// ParseLSBReleaseContent parses lsb-release content from a string.
func ParseLSBReleaseContent(content string) (*Distribution, error) {
	dist := &Distribution{}
	scanner := bufio.NewScanner(strings.NewReader(content))

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		key, value, found := strings.Cut(line, "=")
		if !found {
			continue
		}

		value = unquote(value)

		switch key {
		case "DISTRIB_ID":
			dist.ID = strings.ToLower(value)
			dist.Name = value
		case "DISTRIB_RELEASE":
			dist.VersionID = value
		case "DISTRIB_CODENAME":
			dist.VersionCodename = value
		case "DISTRIB_DESCRIPTION":
			dist.PrettyName = value
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, errors.Wrap(errors.DistroDetection, "failed to parse lsb-release content", err).WithOp("distro.ParseLSBReleaseContent")
	}

	// Determine the distribution family
	dist.Family = DetectFamily(dist.ID, dist.IDLike)

	return dist, nil
}
