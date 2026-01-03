package distro

import (
	"context"
	"os"
	"strings"

	"github.com/tungetti/igor/internal/constants"
	"github.com/tungetti/igor/internal/errors"
	"github.com/tungetti/igor/internal/exec"
)

// FileReader interface for reading files (allows mocking in tests).
type FileReader interface {
	// ReadFile reads the contents of a file.
	ReadFile(path string) ([]byte, error)

	// FileExists checks if a file exists.
	FileExists(path string) bool
}

// DefaultFileReader uses the real filesystem.
type DefaultFileReader struct{}

// ReadFile implements FileReader.
func (r *DefaultFileReader) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

// FileExists implements FileReader.
func (r *DefaultFileReader) FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// Detector detects the Linux distribution.
type Detector struct {
	executor exec.Executor
	fsReader FileReader
}

// NewDetector creates a new distribution detector.
// If fsReader is nil, DefaultFileReader is used.
func NewDetector(executor exec.Executor, fsReader FileReader) *Detector {
	if fsReader == nil {
		fsReader = &DefaultFileReader{}
	}
	return &Detector{
		executor: executor,
		fsReader: fsReader,
	}
}

// Detect detects the current Linux distribution.
// Detection order:
// 1. /etc/os-release (standard, most reliable)
// 2. /usr/lib/os-release (fallback location)
// 3. /etc/lsb-release (older Ubuntu systems)
// 4. /etc/redhat-release (RHEL/CentOS fallback)
// 5. /etc/debian_version (Debian fallback)
// 6. /etc/arch-release (Arch fallback)
// 7. /etc/SuSE-release (SUSE fallback)
// 8. lsb_release command (if available)
func (d *Detector) Detect(ctx context.Context) (*Distribution, error) {
	const op = "distro.Detect"

	// Check for context cancellation
	select {
	case <-ctx.Done():
		return nil, errors.Wrap(errors.Timeout, "detection cancelled", ctx.Err()).WithOp(op)
	default:
	}

	// Try /etc/os-release first (standard location)
	if dist, err := d.tryOSRelease(constants.OSReleasePath); err == nil && dist != nil {
		return dist, nil
	}

	// Try /usr/lib/os-release (fallback location)
	if dist, err := d.tryOSRelease("/usr/lib/os-release"); err == nil && dist != nil {
		return dist, nil
	}

	// Try /etc/lsb-release (older Ubuntu)
	if dist, err := d.tryLSBRelease(constants.LSBReleasePath); err == nil && dist != nil {
		return dist, nil
	}

	// Check for context cancellation before fallback files
	select {
	case <-ctx.Done():
		return nil, errors.Wrap(errors.Timeout, "detection cancelled", ctx.Err()).WithOp(op)
	default:
	}

	// Try distribution-specific fallback files
	if dist := d.tryFallbackFiles(); dist != nil {
		return dist, nil
	}

	// Last resort: try lsb_release command
	if dist, err := d.tryLSBReleaseCommand(ctx); err == nil && dist != nil {
		return dist, nil
	}

	return nil, errors.New(errors.DistroDetection, "could not detect Linux distribution").WithOp(op)
}

// tryOSRelease attempts to parse an os-release file.
func (d *Detector) tryOSRelease(path string) (*Distribution, error) {
	if !d.fsReader.FileExists(path) {
		return nil, errors.Newf(errors.NotFound, "file not found: %s", path)
	}

	content, err := d.fsReader.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return ParseOSReleaseContent(string(content))
}

// tryLSBRelease attempts to parse an lsb-release file.
func (d *Detector) tryLSBRelease(path string) (*Distribution, error) {
	if !d.fsReader.FileExists(path) {
		return nil, errors.Newf(errors.NotFound, "file not found: %s", path)
	}

	content, err := d.fsReader.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return ParseLSBReleaseContent(string(content))
}

// tryFallbackFiles checks for distribution-specific release files.
func (d *Detector) tryFallbackFiles() *Distribution {
	// Check /etc/redhat-release (RHEL/CentOS/Fedora)
	if d.fsReader.FileExists("/etc/redhat-release") {
		if content, err := d.fsReader.ReadFile("/etc/redhat-release"); err == nil {
			return d.parseRedHatRelease(string(content))
		}
	}

	// Check /etc/debian_version (Debian)
	if d.fsReader.FileExists("/etc/debian_version") {
		if content, err := d.fsReader.ReadFile("/etc/debian_version"); err == nil {
			return d.parseDebianVersion(string(content))
		}
	}

	// Check /etc/arch-release (Arch Linux)
	if d.fsReader.FileExists("/etc/arch-release") {
		return &Distribution{
			ID:         "arch",
			Name:       "Arch Linux",
			PrettyName: "Arch Linux",
			Family:     constants.FamilyArch,
		}
	}

	// Check /etc/SuSE-release (older SUSE)
	if d.fsReader.FileExists("/etc/SuSE-release") {
		if content, err := d.fsReader.ReadFile("/etc/SuSE-release"); err == nil {
			return d.parseSuSERelease(string(content))
		}
	}

	return nil
}

// parseRedHatRelease parses /etc/redhat-release content.
func (d *Detector) parseRedHatRelease(content string) *Distribution {
	content = strings.TrimSpace(content)
	dist := &Distribution{
		PrettyName: content,
		Family:     constants.FamilyRHEL,
	}

	// Common patterns:
	// "Red Hat Enterprise Linux release 9.0 (Plow)"
	// "CentOS Linux release 7.9.2009 (Core)"
	// "Fedora release 40 (Forty)"
	// "Rocky Linux release 9.0 (Blue Onyx)"
	// "AlmaLinux release 9.0 (Emerald Puma)"

	lowerContent := strings.ToLower(content)
	switch {
	case strings.Contains(lowerContent, "red hat"):
		dist.ID = "rhel"
		dist.Name = "Red Hat Enterprise Linux"
	case strings.Contains(lowerContent, "centos"):
		dist.ID = "centos"
		dist.Name = "CentOS"
	case strings.Contains(lowerContent, "fedora"):
		dist.ID = "fedora"
		dist.Name = "Fedora"
	case strings.Contains(lowerContent, "rocky"):
		dist.ID = "rocky"
		dist.Name = "Rocky Linux"
		dist.IDLike = []string{"rhel", "centos", "fedora"}
	case strings.Contains(lowerContent, "alma"):
		dist.ID = "almalinux"
		dist.Name = "AlmaLinux"
		dist.IDLike = []string{"rhel", "centos", "fedora"}
	default:
		dist.ID = "rhel"
		dist.Name = "RHEL-based"
	}

	// Try to extract version
	if idx := strings.Index(lowerContent, "release"); idx != -1 {
		remaining := strings.TrimSpace(content[idx+len("release"):])
		if spaceIdx := strings.Index(remaining, " "); spaceIdx > 0 {
			dist.VersionID = remaining[:spaceIdx]
		} else if remaining != "" {
			dist.VersionID = remaining
		}
	}

	return dist
}

// parseDebianVersion parses /etc/debian_version content.
func (d *Detector) parseDebianVersion(content string) *Distribution {
	content = strings.TrimSpace(content)
	dist := &Distribution{
		ID:        "debian",
		Name:      "Debian GNU/Linux",
		VersionID: content,
		Family:    constants.FamilyDebian,
	}

	// Version can be like "12.0" or a codename like "bookworm/sid"
	if strings.Contains(content, "/") {
		parts := strings.Split(content, "/")
		dist.VersionCodename = parts[0]
		dist.VersionID = "" // Codename only, no version
	}

	if dist.VersionID != "" {
		dist.PrettyName = "Debian GNU/Linux " + dist.VersionID
	} else if dist.VersionCodename != "" {
		dist.PrettyName = "Debian GNU/Linux (" + dist.VersionCodename + ")"
	} else {
		dist.PrettyName = "Debian GNU/Linux"
	}

	return dist
}

// parseSuSERelease parses /etc/SuSE-release content.
func (d *Detector) parseSuSERelease(content string) *Distribution {
	lines := strings.Split(content, "\n")
	dist := &Distribution{
		Family: constants.FamilySUSE,
	}

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if dist.PrettyName == "" {
			dist.PrettyName = line
			dist.Name = line
		}

		if strings.HasPrefix(line, "VERSION") {
			if _, value, found := strings.Cut(line, "="); found {
				dist.VersionID = strings.TrimSpace(value)
			}
		}
	}

	// Determine ID
	lowerName := strings.ToLower(dist.Name)
	switch {
	case strings.Contains(lowerName, "tumbleweed"):
		dist.ID = "opensuse-tumbleweed"
	case strings.Contains(lowerName, "leap"):
		dist.ID = "opensuse-leap"
	case strings.Contains(lowerName, "sles") || strings.Contains(lowerName, "enterprise"):
		dist.ID = "sles"
	default:
		dist.ID = "opensuse"
	}

	return dist
}

// tryLSBReleaseCommand tries to get distribution info from the lsb_release command.
func (d *Detector) tryLSBReleaseCommand(ctx context.Context) (*Distribution, error) {
	if d.executor == nil {
		return nil, errors.New(errors.DistroDetection, "no executor available")
	}

	result := d.executor.Execute(ctx, "lsb_release", "-a")
	if !result.Success() {
		return nil, errors.Newf(errors.DistroDetection, "lsb_release command failed")
	}

	dist := &Distribution{}
	lines := strings.Split(result.StdoutString(), "\n")

	for _, line := range lines {
		key, value, found := strings.Cut(line, ":")
		if !found {
			continue
		}

		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)

		switch key {
		case "Distributor ID":
			dist.ID = strings.ToLower(value)
			dist.Name = value
		case "Description":
			dist.PrettyName = value
		case "Release":
			dist.VersionID = value
		case "Codename":
			if value != "n/a" {
				dist.VersionCodename = value
			}
		}
	}

	if dist.ID == "" {
		return nil, errors.New(errors.DistroDetection, "lsb_release returned no distribution ID")
	}

	dist.Family = DetectFamily(dist.ID, dist.IDLike)
	return dist, nil
}

// DetectFamily determines the distribution family from ID and ID_LIKE.
func DetectFamily(id string, idLike []string) constants.DistroFamily {
	id = strings.ToLower(id)

	// Direct ID matches
	switch id {
	case "debian", "ubuntu", "linuxmint", "pop", "elementary", "zorin", "kali", "mx", "lmde", "raspbian", "devuan":
		return constants.FamilyDebian
	case "fedora", "rhel", "centos", "rocky", "almalinux", "ol", "amzn", "scientific", "oracle":
		return constants.FamilyRHEL
	case "arch", "manjaro", "endeavouros", "garuda", "artix", "arcolinux", "archcraft", "archbang":
		return constants.FamilyArch
	case "opensuse", "opensuse-leap", "opensuse-tumbleweed", "sles", "suse":
		return constants.FamilySUSE
	}

	// Check ID_LIKE for derivatives
	for _, like := range idLike {
		like = strings.ToLower(like)
		switch like {
		case "debian", "ubuntu":
			return constants.FamilyDebian
		case "fedora", "rhel", "centos":
			return constants.FamilyRHEL
		case "arch":
			return constants.FamilyArch
		case "suse", "opensuse":
			return constants.FamilySUSE
		}
	}

	return constants.FamilyUnknown
}
