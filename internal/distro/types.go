// Package distro provides Linux distribution detection and identification.
// It parses /etc/os-release, lsb_release, and other system files to determine
// the current Linux distribution and its family (Debian, RHEL, Arch, SUSE).
package distro

import (
	"fmt"
	"strings"

	"github.com/tungetti/igor/internal/constants"
)

// Distribution contains information about the detected Linux distribution.
// This struct is populated by parsing /etc/os-release or similar system files.
type Distribution struct {
	// ID is the lowercase identifier of the distribution (e.g., "ubuntu", "fedora", "arch").
	ID string

	// Name is the human-readable name of the distribution (e.g., "Ubuntu").
	Name string

	// Version is the version string of the distribution (e.g., "24.04 LTS (Noble Numbat)").
	Version string

	// VersionID is the version identifier (e.g., "24.04").
	VersionID string

	// VersionCodename is the version codename (e.g., "noble").
	VersionCodename string

	// PrettyName is the full human-readable name (e.g., "Ubuntu 24.04 LTS").
	PrettyName string

	// Family is the distribution family for package management.
	Family constants.DistroFamily

	// IDLike is a list of related distributions (e.g., ["debian"] for Ubuntu).
	IDLike []string

	// HomeURL is the distribution's home URL.
	HomeURL string

	// SupportURL is the distribution's support URL.
	SupportURL string

	// BuildID is the build identifier, typically used by rolling releases.
	BuildID string
}

// String returns a human-readable representation of the distribution.
func (d *Distribution) String() string {
	if d.PrettyName != "" {
		return d.PrettyName
	}
	if d.Name != "" {
		if d.VersionID != "" {
			return fmt.Sprintf("%s %s", d.Name, d.VersionID)
		}
		return d.Name
	}
	if d.ID != "" {
		return d.ID
	}
	return "Unknown Distribution"
}

// IsDebian returns true if the distribution is in the Debian family.
func (d *Distribution) IsDebian() bool {
	return d.Family == constants.FamilyDebian
}

// IsRHEL returns true if the distribution is in the RHEL/Fedora family.
func (d *Distribution) IsRHEL() bool {
	return d.Family == constants.FamilyRHEL
}

// IsArch returns true if the distribution is in the Arch Linux family.
func (d *Distribution) IsArch() bool {
	return d.Family == constants.FamilyArch
}

// IsSUSE returns true if the distribution is in the SUSE family.
func (d *Distribution) IsSUSE() bool {
	return d.Family == constants.FamilySUSE
}

// IsUnknown returns true if the distribution family could not be determined.
func (d *Distribution) IsUnknown() bool {
	return d.Family == constants.FamilyUnknown
}

// MajorVersion extracts the major version from VersionID.
// For example, "24.04" returns "24", "40" returns "40", "15.5" returns "15".
// Returns an empty string if VersionID is empty or cannot be parsed.
func (d *Distribution) MajorVersion() string {
	if d.VersionID == "" {
		return ""
	}

	// Split by common separators
	for _, sep := range []string{".", "-", "_"} {
		if idx := strings.Index(d.VersionID, sep); idx > 0 {
			return d.VersionID[:idx]
		}
	}

	// No separator found, return the whole version
	return d.VersionID
}

// MinorVersion extracts the minor version from VersionID.
// For example, "24.04" returns "04", "15.5" returns "5".
// Returns an empty string if there is no minor version.
func (d *Distribution) MinorVersion() string {
	if d.VersionID == "" {
		return ""
	}

	// Split by common separators
	for _, sep := range []string{".", "-", "_"} {
		if idx := strings.Index(d.VersionID, sep); idx >= 0 && idx < len(d.VersionID)-1 {
			remainder := d.VersionID[idx+1:]
			// Get just the minor version (up to next separator)
			for _, sep2 := range []string{".", "-", "_"} {
				if idx2 := strings.Index(remainder, sep2); idx2 > 0 {
					return remainder[:idx2]
				}
			}
			return remainder
		}
	}

	return ""
}

// IsRolling returns true if this is a rolling release distribution.
// Rolling releases typically use BUILD_ID instead of VERSION_ID.
func (d *Distribution) IsRolling() bool {
	// Arch and derivatives are typically rolling
	if d.IsArch() {
		return true
	}
	// openSUSE Tumbleweed is rolling
	if d.ID == "opensuse-tumbleweed" {
		return true
	}
	// If there's a BUILD_ID but no VERSION_ID, it's likely rolling
	if d.BuildID != "" && d.VersionID == "" {
		return true
	}
	return false
}
