// Package factory provides a package manager factory that creates the appropriate
// package manager implementation based on the detected Linux distribution.
package factory

import (
	"context"
	"fmt"
	"strconv"

	"github.com/tungetti/igor/internal/constants"
	"github.com/tungetti/igor/internal/distro"
	"github.com/tungetti/igor/internal/exec"
	"github.com/tungetti/igor/internal/pkg"
	"github.com/tungetti/igor/internal/pkg/apt"
	"github.com/tungetti/igor/internal/pkg/dnf"
	"github.com/tungetti/igor/internal/pkg/pacman"
	"github.com/tungetti/igor/internal/pkg/yum"
	"github.com/tungetti/igor/internal/pkg/zypper"
	"github.com/tungetti/igor/internal/privilege"
)

// ErrUnsupportedDistro is returned when the distribution is not supported.
var ErrUnsupportedDistro = pkg.NewPackageError(0, "unsupported distribution family")

// Factory creates package managers based on distribution.
// It uses lazy initialization, creating managers only when requested.
type Factory struct {
	executor  exec.Executor
	privilege *privilege.Manager
	detector  *distro.Detector
}

// NewFactory creates a new package manager factory.
// The executor is used for running shell commands.
// The privilege manager handles privilege elevation for operations requiring root access.
// The detector is used for distribution detection when auto-detecting the package manager.
func NewFactory(executor exec.Executor, privilege *privilege.Manager, detector *distro.Detector) *Factory {
	return &Factory{
		executor:  executor,
		privilege: privilege,
		detector:  detector,
	}
}

// Create returns the appropriate package manager for the current system.
// It auto-detects the distribution and returns the matching manager.
// Returns an error if the distribution cannot be detected or is not supported.
func (f *Factory) Create(ctx context.Context) (pkg.Manager, error) {
	if f.detector == nil {
		return nil, fmt.Errorf("factory: detector is nil, cannot auto-detect distribution")
	}

	dist, err := f.detector.Detect(ctx)
	if err != nil {
		return nil, fmt.Errorf("factory: failed to detect distribution: %w", err)
	}

	return f.CreateForDistribution(dist)
}

// CreateForFamily returns a package manager for a specific distribution family.
// Useful when you already know the target distribution family.
// For RHEL family, this returns DNF (suitable for modern systems).
// Use CreateForDistribution for more precise control over YUM vs DNF selection.
func (f *Factory) CreateForFamily(family constants.DistroFamily) (pkg.Manager, error) {
	switch family {
	case constants.FamilyDebian:
		return apt.NewManager(f.executor, f.privilege), nil

	case constants.FamilyRHEL:
		// Default to DNF for RHEL family when version is unknown
		// This is appropriate for Fedora and modern RHEL/CentOS/Rocky/Alma (v8+)
		return dnf.NewManager(f.executor, f.privilege), nil

	case constants.FamilyArch:
		return pacman.NewManager(f.executor, f.privilege), nil

	case constants.FamilySUSE:
		return zypper.NewManager(f.executor, f.privilege), nil

	case constants.FamilyUnknown:
		return nil, fmt.Errorf("factory: %w: unknown", ErrUnsupportedDistro)

	default:
		return nil, fmt.Errorf("factory: %w: %s", ErrUnsupportedDistro, family)
	}
}

// CreateForDistribution returns a package manager for a specific distribution.
// Handles special cases like CentOS 7 (YUM) vs CentOS 8+ (DNF).
// This is the most precise method for creating a package manager.
func (f *Factory) CreateForDistribution(dist *distro.Distribution) (pkg.Manager, error) {
	if dist == nil {
		return nil, fmt.Errorf("factory: distribution is nil")
	}

	switch dist.Family {
	case constants.FamilyDebian:
		return apt.NewManager(f.executor, f.privilege), nil

	case constants.FamilyRHEL:
		return f.createRHELManager(dist), nil

	case constants.FamilyArch:
		return pacman.NewManager(f.executor, f.privilege), nil

	case constants.FamilySUSE:
		return zypper.NewManager(f.executor, f.privilege), nil

	case constants.FamilyUnknown:
		return nil, fmt.Errorf("factory: %w: unknown (%s)", ErrUnsupportedDistro, dist.ID)

	default:
		return nil, fmt.Errorf("factory: %w: %s", ErrUnsupportedDistro, dist.Family)
	}
}

// createRHELManager determines whether to use DNF or YUM based on distribution version.
// RHEL/CentOS 7 and earlier use YUM, while RHEL/CentOS 8+ and Fedora use DNF.
func (f *Factory) createRHELManager(dist *distro.Distribution) pkg.Manager {
	// Fedora always uses DNF
	if dist.ID == "fedora" {
		return dnf.NewManager(f.executor, f.privilege)
	}

	// Check the major version to determine DNF vs YUM
	majorVersion := dist.MajorVersion()
	if majorVersion != "" {
		v, err := strconv.Atoi(majorVersion)
		if err == nil && v < 8 {
			// CentOS 7, RHEL 7, Oracle Linux 7, etc. use YUM
			return yum.NewManager(f.executor, f.privilege)
		}
	}

	// Fedora, CentOS 8+, RHEL 8+, Rocky, Alma, Amazon Linux 2023+ use DNF
	return dnf.NewManager(f.executor, f.privilege)
}

// AvailableManagers returns a list of all package manager names.
// These are the package manager implementations supported by the factory.
func AvailableManagers() []string {
	return []string{
		"apt",
		"dnf",
		"yum",
		"pacman",
		"zypper",
	}
}

// SupportedFamilies returns all supported distribution families.
// These are the distribution families that the factory can create managers for.
func SupportedFamilies() []constants.DistroFamily {
	return []constants.DistroFamily{
		constants.FamilyDebian,
		constants.FamilyRHEL,
		constants.FamilyArch,
		constants.FamilySUSE,
	}
}
