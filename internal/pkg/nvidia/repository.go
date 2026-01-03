package nvidia

import (
	"fmt"
	"strings"

	"github.com/tungetti/igor/internal/constants"
	"github.com/tungetti/igor/internal/distro"
	"github.com/tungetti/igor/internal/pkg"
)

// RepositoryInfo contains detailed information about an NVIDIA repository
// for a specific distribution, including setup instructions.
type RepositoryInfo struct {
	// Name is the repository name/identifier.
	Name string

	// URL is the base URL for the repository.
	URL string

	// GPGKey is the URL to the GPG key for package verification.
	GPGKey string

	// Enabled indicates if the repository should be enabled by default.
	Enabled bool

	// Description provides a human-readable description.
	Description string

	// Type is the repository type (e.g., "deb", "rpm", "pkg").
	Type string

	// Components are the repository components (e.g., "main" for Debian).
	Components []string

	// RequiresThirdParty indicates if a third-party repo (like RPM Fusion) is needed.
	RequiresThirdParty bool

	// ThirdPartyName is the name of the required third-party repository.
	ThirdPartyName string

	// SetupInstructions provides additional setup notes.
	SetupInstructions string
}

// NVIDIA repository URLs and GPG keys by distribution family.
const (
	// Debian/Ubuntu CUDA repository base URL template.
	// Format: ubuntu{version}/{arch} e.g., ubuntu2404/x86_64
	CUDARepoBaseURL = "https://developer.download.nvidia.com/compute/cuda/repos"

	// CUDA GPG key URL template.
	CUDAGPGKeyURL = "https://developer.download.nvidia.com/compute/cuda/repos/%s/x86_64/3bf863cc.pub"

	// Ubuntu graphics-drivers PPA.
	UbuntuGraphicsDriversPPA = "ppa:graphics-drivers/ppa"

	// RPM Fusion Nonfree repository URL template for Fedora.
	// The %fedora variable is expanded by rpm.
	RPMFusionNonfreeFedoraURL = "https://download1.rpmfusion.org/nonfree/fedora/rpmfusion-nonfree-release-%s.noarch.rpm"

	// RPM Fusion Free repository URL template for Fedora.
	RPMFusionFreeFedoraURL = "https://download1.rpmfusion.org/free/fedora/rpmfusion-free-release-%s.noarch.rpm"

	// RPM Fusion Nonfree repository URL template for RHEL/CentOS/Rocky.
	RPMFusionNonfreeELURL = "https://download1.rpmfusion.org/nonfree/el/rpmfusion-nonfree-release-%s.noarch.rpm"

	// RPM Fusion Free repository URL template for RHEL/CentOS/Rocky.
	RPMFusionFreeELURL = "https://download1.rpmfusion.org/free/el/rpmfusion-free-release-%s.noarch.rpm"

	// openSUSE Tumbleweed NVIDIA repository.
	OpenSUSETumbleweedNvidiaURL = "https://download.nvidia.com/opensuse/tumbleweed"

	// openSUSE Leap NVIDIA repository URL template.
	// Format: https://download.nvidia.com/opensuse/leap/{version}
	OpenSUSELeapNvidiaURL = "https://download.nvidia.com/opensuse/leap/%s"
)

// Ubuntu codename to CUDA repository path mapping.
var ubuntuCUDARepos = map[string]string{
	"noble":  "ubuntu2404", // Ubuntu 24.04
	"jammy":  "ubuntu2204", // Ubuntu 22.04
	"focal":  "ubuntu2004", // Ubuntu 20.04
	"bionic": "ubuntu1804", // Ubuntu 18.04
	"24.04":  "ubuntu2404",
	"22.04":  "ubuntu2204",
	"20.04":  "ubuntu2004",
	"18.04":  "ubuntu1804",
}

// Debian codename to CUDA repository path mapping.
var debianCUDARepos = map[string]string{
	"bookworm": "debian12", // Debian 12
	"bullseye": "debian11", // Debian 11
	"buster":   "debian10", // Debian 10
	"12":       "debian12",
	"11":       "debian11",
	"10":       "debian10",
}

// GetRepository returns the primary NVIDIA repository for a distribution.
// This is the repository needed to install the NVIDIA driver.
func GetRepository(dist *distro.Distribution) (*pkg.Repository, error) {
	if dist == nil {
		return nil, fmt.Errorf("distribution cannot be nil")
	}

	switch dist.Family {
	case constants.FamilyDebian:
		return getDebianRepository(dist)
	case constants.FamilyRHEL:
		return getRHELRepository(dist)
	case constants.FamilyArch:
		return getArchRepository(dist)
	case constants.FamilySUSE:
		return getSUSERepository(dist)
	default:
		return nil, fmt.Errorf("unsupported distribution family: %s", dist.Family)
	}
}

// GetCUDARepository returns the CUDA repository for a distribution.
// This repository provides the CUDA toolkit and related packages.
func GetCUDARepository(dist *distro.Distribution) (*pkg.Repository, error) {
	if dist == nil {
		return nil, fmt.Errorf("distribution cannot be nil")
	}

	switch dist.Family {
	case constants.FamilyDebian:
		return getDebianCUDARepository(dist)
	case constants.FamilyRHEL:
		return getRHELCUDARepository(dist)
	case constants.FamilyArch:
		// Arch uses official repos for CUDA
		return nil, nil
	case constants.FamilySUSE:
		// SUSE uses the same NVIDIA repo for CUDA
		return getSUSERepository(dist)
	default:
		return nil, fmt.Errorf("unsupported distribution family: %s", dist.Family)
	}
}

// GetRepositoryForFamily returns the default repository for a distribution family.
// This is useful when the specific distribution is not known.
func GetRepositoryForFamily(family constants.DistroFamily) (*pkg.Repository, error) {
	switch family {
	case constants.FamilyDebian:
		// Return Ubuntu 22.04 as default
		return &pkg.Repository{
			Name:         "nvidia-cuda",
			URL:          fmt.Sprintf("%s/ubuntu2204/x86_64", CUDARepoBaseURL),
			Enabled:      true,
			GPGKey:       fmt.Sprintf(CUDAGPGKeyURL, "ubuntu2204"),
			Type:         "deb",
			Distribution: "jammy",
			Components:   []string{"/"},
		}, nil

	case constants.FamilyRHEL:
		// RPM Fusion is needed, return a placeholder
		return &pkg.Repository{
			Name:    "rpmfusion-nonfree",
			URL:     "https://download1.rpmfusion.org/nonfree/fedora",
			Enabled: true,
			Type:    "rpm",
		}, nil

	case constants.FamilyArch:
		// Arch doesn't need an extra repository
		return nil, nil

	case constants.FamilySUSE:
		return &pkg.Repository{
			Name:    "nvidia",
			URL:     OpenSUSETumbleweedNvidiaURL,
			Enabled: true,
			Type:    "rpm",
		}, nil

	default:
		return nil, fmt.Errorf("unsupported distribution family: %s", family)
	}
}

// GetRepositoryInfo returns detailed repository information for a distribution.
func GetRepositoryInfo(dist *distro.Distribution) (*RepositoryInfo, error) {
	if dist == nil {
		return nil, fmt.Errorf("distribution cannot be nil")
	}

	switch dist.Family {
	case constants.FamilyDebian:
		return getDebianRepositoryInfo(dist)
	case constants.FamilyRHEL:
		return getRHELRepositoryInfo(dist)
	case constants.FamilyArch:
		return getArchRepositoryInfo(dist)
	case constants.FamilySUSE:
		return getSUSERepositoryInfo(dist)
	default:
		return nil, fmt.Errorf("unsupported distribution family: %s", dist.Family)
	}
}

// getDebianRepository returns the NVIDIA repository for Debian-based distributions.
func getDebianRepository(dist *distro.Distribution) (*pkg.Repository, error) {
	// For Ubuntu, use the graphics-drivers PPA or CUDA repo
	if dist.ID == "ubuntu" || dist.ID == "pop" || dist.ID == "linuxmint" {
		return &pkg.Repository{
			Name:    "graphics-drivers-ppa",
			URL:     UbuntuGraphicsDriversPPA,
			Enabled: true,
			Type:    "ppa",
		}, nil
	}

	// For Debian, use the CUDA repository
	repoPath := getDebianRepoPath(dist)
	return &pkg.Repository{
		Name:         "nvidia-cuda",
		URL:          fmt.Sprintf("%s/%s/x86_64", CUDARepoBaseURL, repoPath),
		Enabled:      true,
		GPGKey:       fmt.Sprintf(CUDAGPGKeyURL, repoPath),
		Type:         "deb",
		Distribution: "/",
		Components:   []string{},
	}, nil
}

// getDebianCUDARepository returns the CUDA repository for Debian-based distributions.
func getDebianCUDARepository(dist *distro.Distribution) (*pkg.Repository, error) {
	var repoPath string

	if dist.ID == "ubuntu" || dist.ID == "pop" || dist.ID == "linuxmint" {
		repoPath = getUbuntuRepoPath(dist)
	} else {
		repoPath = getDebianRepoPath(dist)
	}

	return &pkg.Repository{
		Name:         "nvidia-cuda",
		URL:          fmt.Sprintf("%s/%s/x86_64", CUDARepoBaseURL, repoPath),
		Enabled:      true,
		GPGKey:       fmt.Sprintf(CUDAGPGKeyURL, repoPath),
		Type:         "deb",
		Distribution: "/",
		Components:   []string{},
	}, nil
}

// getUbuntuRepoPath returns the CUDA repository path for Ubuntu.
func getUbuntuRepoPath(dist *distro.Distribution) string {
	// Try codename first
	if dist.VersionCodename != "" {
		if path, ok := ubuntuCUDARepos[strings.ToLower(dist.VersionCodename)]; ok {
			return path
		}
	}

	// Try version ID
	if dist.VersionID != "" {
		if path, ok := ubuntuCUDARepos[dist.VersionID]; ok {
			return path
		}
	}

	// Default to latest LTS
	return "ubuntu2204"
}

// getDebianRepoPath returns the CUDA repository path for Debian.
func getDebianRepoPath(dist *distro.Distribution) string {
	// Try codename first
	if dist.VersionCodename != "" {
		if path, ok := debianCUDARepos[strings.ToLower(dist.VersionCodename)]; ok {
			return path
		}
	}

	// Try major version
	majorVersion := dist.MajorVersion()
	if majorVersion != "" {
		if path, ok := debianCUDARepos[majorVersion]; ok {
			return path
		}
	}

	// Default to latest stable
	return "debian12"
}

// getRHELRepository returns the NVIDIA repository for RHEL-based distributions.
// For Fedora/RHEL, we need RPM Fusion.
func getRHELRepository(dist *distro.Distribution) (*pkg.Repository, error) {
	if dist.ID == "fedora" {
		version := dist.VersionID
		if version == "" {
			version = "40" // Default to Fedora 40
		}
		return &pkg.Repository{
			Name:    "rpmfusion-nonfree",
			URL:     fmt.Sprintf(RPMFusionNonfreeFedoraURL, version),
			Enabled: true,
			Type:    "rpm",
		}, nil
	}

	// For RHEL/CentOS/Rocky/AlmaLinux
	majorVersion := dist.MajorVersion()
	if majorVersion == "" {
		majorVersion = "9" // Default to RHEL 9
	}
	return &pkg.Repository{
		Name:    "rpmfusion-nonfree",
		URL:     fmt.Sprintf(RPMFusionNonfreeELURL, majorVersion),
		Enabled: true,
		Type:    "rpm",
	}, nil
}

// getRHELCUDARepository returns the CUDA repository for RHEL-based distributions.
func getRHELCUDARepository(dist *distro.Distribution) (*pkg.Repository, error) {
	var repoPath string

	if dist.ID == "fedora" {
		version := dist.VersionID
		if version == "" {
			version = "40"
		}
		repoPath = fmt.Sprintf("fedora%s", version)
	} else {
		majorVersion := dist.MajorVersion()
		if majorVersion == "" {
			majorVersion = "9"
		}
		repoPath = fmt.Sprintf("rhel%s", majorVersion)
	}

	return &pkg.Repository{
		Name:    "nvidia-cuda",
		URL:     fmt.Sprintf("%s/%s/x86_64", CUDARepoBaseURL, repoPath),
		Enabled: true,
		GPGKey:  fmt.Sprintf(CUDAGPGKeyURL, repoPath),
		Type:    "rpm",
	}, nil
}

// getArchRepository returns the NVIDIA repository for Arch-based distributions.
// Arch uses official repositories, so no extra repo is needed.
func getArchRepository(dist *distro.Distribution) (*pkg.Repository, error) {
	// Arch Linux uses official repos - no extra repository needed
	return nil, nil
}

// getSUSERepository returns the NVIDIA repository for SUSE-based distributions.
func getSUSERepository(dist *distro.Distribution) (*pkg.Repository, error) {
	var url string

	if dist.ID == "opensuse-tumbleweed" || strings.Contains(strings.ToLower(dist.Name), "tumbleweed") {
		url = OpenSUSETumbleweedNvidiaURL
	} else {
		// openSUSE Leap
		version := dist.VersionID
		if version == "" {
			version = "15.5" // Default to Leap 15.5
		}
		url = fmt.Sprintf(OpenSUSELeapNvidiaURL, version)
	}

	return &pkg.Repository{
		Name:    "nvidia",
		URL:     url,
		Enabled: true,
		Type:    "rpm",
	}, nil
}

// getDebianRepositoryInfo returns detailed repository info for Debian-based distributions.
func getDebianRepositoryInfo(dist *distro.Distribution) (*RepositoryInfo, error) {
	if dist.ID == "ubuntu" || dist.ID == "pop" || dist.ID == "linuxmint" {
		return &RepositoryInfo{
			Name:        "graphics-drivers-ppa",
			URL:         UbuntuGraphicsDriversPPA,
			Enabled:     true,
			Description: "Ubuntu Graphics Drivers PPA - provides latest NVIDIA drivers",
			Type:        "ppa",
			SetupInstructions: `To add the PPA:
  sudo add-apt-repository ppa:graphics-drivers/ppa
  sudo apt update`,
		}, nil
	}

	repoPath := getDebianRepoPath(dist)
	return &RepositoryInfo{
		Name:        "nvidia-cuda",
		URL:         fmt.Sprintf("%s/%s/x86_64", CUDARepoBaseURL, repoPath),
		GPGKey:      fmt.Sprintf(CUDAGPGKeyURL, repoPath),
		Enabled:     true,
		Description: "NVIDIA CUDA repository for Debian",
		Type:        "deb",
		Components:  []string{"/"},
		SetupInstructions: `To add the CUDA repository:
  wget https://developer.download.nvidia.com/compute/cuda/repos/` + repoPath + `/x86_64/cuda-keyring_1.1-1_all.deb
  sudo dpkg -i cuda-keyring_1.1-1_all.deb
  sudo apt update`,
	}, nil
}

// getRHELRepositoryInfo returns detailed repository info for RHEL-based distributions.
func getRHELRepositoryInfo(dist *distro.Distribution) (*RepositoryInfo, error) {
	var repoURL string
	var version string

	if dist.ID == "fedora" {
		version = dist.VersionID
		if version == "" {
			version = "40"
		}
		repoURL = fmt.Sprintf(RPMFusionNonfreeFedoraURL, version)
	} else {
		version = dist.MajorVersion()
		if version == "" {
			version = "9"
		}
		repoURL = fmt.Sprintf(RPMFusionNonfreeELURL, version)
	}

	return &RepositoryInfo{
		Name:               "rpmfusion-nonfree",
		URL:                repoURL,
		Enabled:            true,
		Description:        "RPM Fusion Nonfree repository - provides NVIDIA drivers",
		Type:               "rpm",
		RequiresThirdParty: true,
		ThirdPartyName:     "RPM Fusion",
		SetupInstructions: `To add RPM Fusion repositories:
  sudo dnf install ` + repoURL + `
  sudo dnf update`,
	}, nil
}

// getArchRepositoryInfo returns detailed repository info for Arch-based distributions.
func getArchRepositoryInfo(dist *distro.Distribution) (*RepositoryInfo, error) {
	return &RepositoryInfo{
		Name:               "official",
		URL:                "https://archlinux.org/packages/",
		Enabled:            true,
		Description:        "Arch Linux official repositories - NVIDIA packages included",
		Type:               "pkg",
		RequiresThirdParty: false,
		SetupInstructions: `No additional repository needed.
NVIDIA packages are available in the official extra repository.
  sudo pacman -S nvidia nvidia-utils`,
	}, nil
}

// getSUSERepositoryInfo returns detailed repository info for SUSE-based distributions.
func getSUSERepositoryInfo(dist *distro.Distribution) (*RepositoryInfo, error) {
	var url string
	var name string

	if dist.ID == "opensuse-tumbleweed" || strings.Contains(strings.ToLower(dist.Name), "tumbleweed") {
		url = OpenSUSETumbleweedNvidiaURL
		name = "NVIDIA-openSUSE-Tumbleweed"
	} else {
		version := dist.VersionID
		if version == "" {
			version = "15.5"
		}
		url = fmt.Sprintf(OpenSUSELeapNvidiaURL, version)
		name = fmt.Sprintf("NVIDIA-openSUSE-Leap-%s", version)
	}

	return &RepositoryInfo{
		Name:        name,
		URL:         url,
		Enabled:     true,
		Description: "Official NVIDIA repository for openSUSE",
		Type:        "rpm",
		SetupInstructions: `To add the NVIDIA repository:
  sudo zypper addrepo --refresh ` + url + ` ` + name + `
  sudo zypper refresh`,
	}, nil
}

// GetRPMFusionURLs returns both Free and Nonfree RPM Fusion URLs for a distribution.
func GetRPMFusionURLs(dist *distro.Distribution) (freeURL, nonfreeURL string, err error) {
	if dist == nil {
		return "", "", fmt.Errorf("distribution cannot be nil")
	}

	if dist.Family != constants.FamilyRHEL {
		return "", "", fmt.Errorf("RPM Fusion is only for RHEL-based distributions")
	}

	if dist.ID == "fedora" {
		version := dist.VersionID
		if version == "" {
			version = "40"
		}
		return fmt.Sprintf(RPMFusionFreeFedoraURL, version),
			fmt.Sprintf(RPMFusionNonfreeFedoraURL, version),
			nil
	}

	// For RHEL/CentOS/Rocky/AlmaLinux
	majorVersion := dist.MajorVersion()
	if majorVersion == "" {
		majorVersion = "9"
	}
	return fmt.Sprintf(RPMFusionFreeELURL, majorVersion),
		fmt.Sprintf(RPMFusionNonfreeELURL, majorVersion),
		nil
}

// RequiresThirdPartyRepo returns true if the distribution needs a third-party
// repository for NVIDIA drivers (like RPM Fusion for Fedora).
func RequiresThirdPartyRepo(dist *distro.Distribution) bool {
	if dist == nil {
		return false
	}

	switch dist.Family {
	case constants.FamilyRHEL:
		// RHEL family needs RPM Fusion
		return true
	case constants.FamilyDebian:
		// Ubuntu/Debian can use PPAs or CUDA repo
		return false
	case constants.FamilyArch:
		// Arch has NVIDIA in official repos
		return false
	case constants.FamilySUSE:
		// SUSE has official NVIDIA repo
		return false
	default:
		return true
	}
}

// GetGPGKeyURL returns the GPG key URL for NVIDIA repositories on a distribution.
func GetGPGKeyURL(dist *distro.Distribution) (string, error) {
	if dist == nil {
		return "", fmt.Errorf("distribution cannot be nil")
	}

	switch dist.Family {
	case constants.FamilyDebian:
		repoPath := "ubuntu2204"
		if dist.ID == "ubuntu" {
			repoPath = getUbuntuRepoPath(dist)
		} else if dist.ID == "debian" {
			repoPath = getDebianRepoPath(dist)
		}
		return fmt.Sprintf(CUDAGPGKeyURL, repoPath), nil

	case constants.FamilyRHEL:
		// RPM Fusion handles its own keys
		return "", nil

	case constants.FamilyArch:
		// Arch uses pacman-key for verification
		return "", nil

	case constants.FamilySUSE:
		// SUSE NVIDIA repo has its own key
		return "", nil

	default:
		return "", fmt.Errorf("unsupported distribution family: %s", dist.Family)
	}
}
