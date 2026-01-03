package nvidia

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tungetti/igor/internal/constants"
	"github.com/tungetti/igor/internal/distro"
)

func TestGetRepository_Debian(t *testing.T) {
	tests := []struct {
		name     string
		dist     *distro.Distribution
		wantNil  bool
		wantType string
		wantURL  string
	}{
		{
			name: "ubuntu",
			dist: &distro.Distribution{
				ID:              "ubuntu",
				VersionID:       "22.04",
				VersionCodename: "jammy",
				Family:          constants.FamilyDebian,
			},
			wantNil:  false,
			wantType: "ppa",
			wantURL:  "ppa:",
		},
		{
			name: "pop_os",
			dist: &distro.Distribution{
				ID:     "pop",
				Family: constants.FamilyDebian,
			},
			wantNil:  false,
			wantType: "ppa",
		},
		{
			name: "debian",
			dist: &distro.Distribution{
				ID:              "debian",
				VersionID:       "12",
				VersionCodename: "bookworm",
				Family:          constants.FamilyDebian,
			},
			wantNil:  false,
			wantType: "deb",
			wantURL:  "developer.download.nvidia.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, err := GetRepository(tt.dist)

			if tt.wantNil {
				assert.Nil(t, repo)
			} else {
				require.NoError(t, err)
				require.NotNil(t, repo)
				assert.Equal(t, tt.wantType, repo.Type)
				if tt.wantURL != "" {
					assert.Contains(t, repo.URL, tt.wantURL)
				}
			}
		})
	}
}

func TestGetRepository_RHEL(t *testing.T) {
	tests := []struct {
		name        string
		dist        *distro.Distribution
		wantContain string
	}{
		{
			name: "fedora",
			dist: &distro.Distribution{
				ID:        "fedora",
				VersionID: "40",
				Family:    constants.FamilyRHEL,
			},
			wantContain: "rpmfusion",
		},
		{
			name: "rhel",
			dist: &distro.Distribution{
				ID:        "rhel",
				VersionID: "9",
				Family:    constants.FamilyRHEL,
			},
			wantContain: "rpmfusion",
		},
		{
			name: "rocky",
			dist: &distro.Distribution{
				ID:        "rocky",
				VersionID: "9.0",
				Family:    constants.FamilyRHEL,
			},
			wantContain: "rpmfusion",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, err := GetRepository(tt.dist)
			require.NoError(t, err)
			require.NotNil(t, repo)
			assert.Contains(t, repo.URL, tt.wantContain)
			assert.Equal(t, "rpm", repo.Type)
		})
	}
}

func TestGetRepository_Arch(t *testing.T) {
	dist := &distro.Distribution{
		ID:     "arch",
		Family: constants.FamilyArch,
	}

	repo, err := GetRepository(dist)
	require.NoError(t, err)
	// Arch doesn't need an extra repository
	assert.Nil(t, repo)
}

func TestGetRepository_SUSE(t *testing.T) {
	tests := []struct {
		name        string
		dist        *distro.Distribution
		wantContain string
	}{
		{
			name: "tumbleweed",
			dist: &distro.Distribution{
				ID:     "opensuse-tumbleweed",
				Family: constants.FamilySUSE,
			},
			wantContain: "tumbleweed",
		},
		{
			name: "leap",
			dist: &distro.Distribution{
				ID:        "opensuse-leap",
				VersionID: "15.5",
				Family:    constants.FamilySUSE,
			},
			wantContain: "15.5",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, err := GetRepository(tt.dist)
			require.NoError(t, err)
			require.NotNil(t, repo)
			assert.Contains(t, repo.URL, tt.wantContain)
		})
	}
}

func TestGetRepository_NilDistribution(t *testing.T) {
	repo, err := GetRepository(nil)
	assert.Error(t, err)
	assert.Nil(t, repo)
}

func TestGetRepository_UnsupportedFamily(t *testing.T) {
	dist := &distro.Distribution{
		ID:     "unknown",
		Family: constants.FamilyUnknown,
	}

	repo, err := GetRepository(dist)
	assert.Error(t, err)
	assert.Nil(t, repo)
	assert.Contains(t, err.Error(), "unsupported")
}

func TestGetCUDARepository_Debian(t *testing.T) {
	tests := []struct {
		name        string
		dist        *distro.Distribution
		wantContain string
	}{
		{
			name: "ubuntu_2204",
			dist: &distro.Distribution{
				ID:              "ubuntu",
				VersionID:       "22.04",
				VersionCodename: "jammy",
				Family:          constants.FamilyDebian,
			},
			wantContain: "ubuntu2204",
		},
		{
			name: "ubuntu_2404",
			dist: &distro.Distribution{
				ID:              "ubuntu",
				VersionID:       "24.04",
				VersionCodename: "noble",
				Family:          constants.FamilyDebian,
			},
			wantContain: "ubuntu2404",
		},
		{
			name: "debian_12",
			dist: &distro.Distribution{
				ID:              "debian",
				VersionID:       "12",
				VersionCodename: "bookworm",
				Family:          constants.FamilyDebian,
			},
			wantContain: "debian12",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, err := GetCUDARepository(tt.dist)
			require.NoError(t, err)
			require.NotNil(t, repo)
			assert.Contains(t, repo.URL, tt.wantContain)
			assert.Contains(t, repo.URL, "developer.download.nvidia.com")
			assert.NotEmpty(t, repo.GPGKey)
		})
	}
}

func TestGetCUDARepository_RHEL(t *testing.T) {
	dist := &distro.Distribution{
		ID:        "fedora",
		VersionID: "40",
		Family:    constants.FamilyRHEL,
	}

	repo, err := GetCUDARepository(dist)
	require.NoError(t, err)
	require.NotNil(t, repo)
	assert.Contains(t, repo.URL, "fedora40")
}

func TestGetCUDARepository_Arch(t *testing.T) {
	dist := &distro.Distribution{
		ID:     "arch",
		Family: constants.FamilyArch,
	}

	// Arch uses official repos for CUDA
	repo, err := GetCUDARepository(dist)
	require.NoError(t, err)
	assert.Nil(t, repo)
}

func TestGetRepositoryForFamily(t *testing.T) {
	tests := []struct {
		family  constants.DistroFamily
		wantNil bool
		wantErr bool
	}{
		{constants.FamilyDebian, false, false},
		{constants.FamilyRHEL, false, false},
		{constants.FamilyArch, true, false},
		{constants.FamilySUSE, false, false},
		{constants.FamilyUnknown, false, true},
	}

	for _, tt := range tests {
		t.Run(string(tt.family), func(t *testing.T) {
			repo, err := GetRepositoryForFamily(tt.family)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, repo)
			} else if tt.wantNil {
				assert.NoError(t, err)
				assert.Nil(t, repo)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, repo)
			}
		})
	}
}

func TestGetRepositoryInfo(t *testing.T) {
	tests := []struct {
		name        string
		dist        *distro.Distribution
		wantName    string
		wantContain string
	}{
		{
			name: "ubuntu",
			dist: &distro.Distribution{
				ID:     "ubuntu",
				Family: constants.FamilyDebian,
			},
			wantName:    "graphics-drivers-ppa",
			wantContain: "add-apt-repository",
		},
		{
			name: "debian",
			dist: &distro.Distribution{
				ID:              "debian",
				VersionCodename: "bookworm",
				Family:          constants.FamilyDebian,
			},
			wantName:    "nvidia-cuda",
			wantContain: "dpkg",
		},
		{
			name: "fedora",
			dist: &distro.Distribution{
				ID:        "fedora",
				VersionID: "40",
				Family:    constants.FamilyRHEL,
			},
			wantName:    "rpmfusion-nonfree",
			wantContain: "dnf install",
		},
		{
			name: "arch",
			dist: &distro.Distribution{
				ID:     "arch",
				Family: constants.FamilyArch,
			},
			wantName:    "official",
			wantContain: "pacman",
		},
		{
			name: "opensuse",
			dist: &distro.Distribution{
				ID:     "opensuse-tumbleweed",
				Family: constants.FamilySUSE,
			},
			wantContain: "zypper addrepo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info, err := GetRepositoryInfo(tt.dist)
			require.NoError(t, err)
			require.NotNil(t, info)

			if tt.wantName != "" {
				assert.Equal(t, tt.wantName, info.Name)
			}
			assert.Contains(t, info.SetupInstructions, tt.wantContain)
		})
	}
}

func TestGetRepositoryInfo_NilDistribution(t *testing.T) {
	info, err := GetRepositoryInfo(nil)
	assert.Error(t, err)
	assert.Nil(t, info)
}

func TestGetRPMFusionURLs(t *testing.T) {
	tests := []struct {
		name        string
		dist        *distro.Distribution
		wantFree    string
		wantNonfree string
		wantErr     bool
	}{
		{
			name: "fedora_40",
			dist: &distro.Distribution{
				ID:        "fedora",
				VersionID: "40",
				Family:    constants.FamilyRHEL,
			},
			wantFree:    "free/fedora/rpmfusion-free-release-40",
			wantNonfree: "nonfree/fedora/rpmfusion-nonfree-release-40",
		},
		{
			name: "rhel_9",
			dist: &distro.Distribution{
				ID:        "rhel",
				VersionID: "9",
				Family:    constants.FamilyRHEL,
			},
			wantFree:    "free/el/rpmfusion-free-release-9",
			wantNonfree: "nonfree/el/rpmfusion-nonfree-release-9",
		},
		{
			name: "rocky_9",
			dist: &distro.Distribution{
				ID:        "rocky",
				VersionID: "9.3",
				Family:    constants.FamilyRHEL,
			},
			wantFree:    "free/el/rpmfusion-free-release-9",
			wantNonfree: "nonfree/el/rpmfusion-nonfree-release-9",
		},
		{
			name: "not_rhel",
			dist: &distro.Distribution{
				ID:     "ubuntu",
				Family: constants.FamilyDebian,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			freeURL, nonfreeURL, err := GetRPMFusionURLs(tt.dist)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Contains(t, freeURL, tt.wantFree)
				assert.Contains(t, nonfreeURL, tt.wantNonfree)
			}
		})
	}
}

func TestGetRPMFusionURLs_NilDistribution(t *testing.T) {
	_, _, err := GetRPMFusionURLs(nil)
	assert.Error(t, err)
}

func TestRequiresThirdPartyRepo(t *testing.T) {
	tests := []struct {
		family   constants.DistroFamily
		requires bool
	}{
		{constants.FamilyDebian, false},
		{constants.FamilyRHEL, true},
		{constants.FamilyArch, false},
		{constants.FamilySUSE, false},
		{constants.FamilyUnknown, true},
	}

	for _, tt := range tests {
		t.Run(string(tt.family), func(t *testing.T) {
			dist := &distro.Distribution{Family: tt.family}
			assert.Equal(t, tt.requires, RequiresThirdPartyRepo(dist))
		})
	}
}

func TestRequiresThirdPartyRepo_NilDistribution(t *testing.T) {
	assert.False(t, RequiresThirdPartyRepo(nil))
}

func TestGetGPGKeyURL(t *testing.T) {
	tests := []struct {
		name        string
		dist        *distro.Distribution
		wantContain string
		wantEmpty   bool
	}{
		{
			name: "ubuntu",
			dist: &distro.Distribution{
				ID:              "ubuntu",
				VersionCodename: "jammy",
				Family:          constants.FamilyDebian,
			},
			wantContain: "3bf863cc.pub",
		},
		{
			name: "debian",
			dist: &distro.Distribution{
				ID:              "debian",
				VersionCodename: "bookworm",
				Family:          constants.FamilyDebian,
			},
			wantContain: "3bf863cc.pub",
		},
		{
			name: "rhel",
			dist: &distro.Distribution{
				ID:     "fedora",
				Family: constants.FamilyRHEL,
			},
			wantEmpty: true,
		},
		{
			name: "arch",
			dist: &distro.Distribution{
				ID:     "arch",
				Family: constants.FamilyArch,
			},
			wantEmpty: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url, err := GetGPGKeyURL(tt.dist)

			if tt.wantEmpty {
				assert.NoError(t, err)
				assert.Empty(t, url)
			} else {
				require.NoError(t, err)
				assert.Contains(t, url, tt.wantContain)
			}
		})
	}
}

func TestGetGPGKeyURL_NilDistribution(t *testing.T) {
	url, err := GetGPGKeyURL(nil)
	assert.Error(t, err)
	assert.Empty(t, url)
}

func TestRepositoryConstants(t *testing.T) {
	// Verify repository URL constants are well-formed
	assert.True(t, strings.HasPrefix(CUDARepoBaseURL, "https://"))
	assert.Contains(t, CUDARepoBaseURL, "developer.download.nvidia.com")

	assert.Contains(t, CUDAGPGKeyURL, "3bf863cc.pub")

	assert.True(t, strings.HasPrefix(UbuntuGraphicsDriversPPA, "ppa:"))

	assert.True(t, strings.HasPrefix(OpenSUSETumbleweedNvidiaURL, "https://"))
	assert.Contains(t, OpenSUSETumbleweedNvidiaURL, "tumbleweed")

	assert.Contains(t, OpenSUSELeapNvidiaURL, "leap")
	assert.Contains(t, OpenSUSELeapNvidiaURL, "%s")
}

func TestUbuntuCUDARepos(t *testing.T) {
	// Verify Ubuntu codename mappings
	assert.Equal(t, "ubuntu2404", ubuntuCUDARepos["noble"])
	assert.Equal(t, "ubuntu2204", ubuntuCUDARepos["jammy"])
	assert.Equal(t, "ubuntu2004", ubuntuCUDARepos["focal"])

	// Version mappings should match
	assert.Equal(t, ubuntuCUDARepos["noble"], ubuntuCUDARepos["24.04"])
	assert.Equal(t, ubuntuCUDARepos["jammy"], ubuntuCUDARepos["22.04"])
}

func TestDebianCUDARepos(t *testing.T) {
	// Verify Debian codename mappings
	assert.Equal(t, "debian12", debianCUDARepos["bookworm"])
	assert.Equal(t, "debian11", debianCUDARepos["bullseye"])
	assert.Equal(t, "debian10", debianCUDARepos["buster"])

	// Version mappings should match
	assert.Equal(t, debianCUDARepos["bookworm"], debianCUDARepos["12"])
	assert.Equal(t, debianCUDARepos["bullseye"], debianCUDARepos["11"])
}

func TestRepositoryInfo_Fields(t *testing.T) {
	dist := &distro.Distribution{
		ID:        "fedora",
		VersionID: "40",
		Family:    constants.FamilyRHEL,
	}

	info, err := GetRepositoryInfo(dist)
	require.NoError(t, err)
	require.NotNil(t, info)

	// RHEL should require third-party repo
	assert.True(t, info.RequiresThirdParty)
	assert.Equal(t, "RPM Fusion", info.ThirdPartyName)
	assert.NotEmpty(t, info.SetupInstructions)
	assert.NotEmpty(t, info.Description)
}

func TestArchRepositoryInfo_NoThirdParty(t *testing.T) {
	dist := &distro.Distribution{
		ID:     "arch",
		Family: constants.FamilyArch,
	}

	info, err := GetRepositoryInfo(dist)
	require.NoError(t, err)
	require.NotNil(t, info)

	// Arch should not require third-party repo
	assert.False(t, info.RequiresThirdParty)
}

func TestGetRepository_LeapVersions(t *testing.T) {
	tests := []struct {
		version     string
		wantContain string
	}{
		{"15.5", "15.5"},
		{"15.6", "15.6"},
		{"15.4", "15.4"},
	}

	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			dist := &distro.Distribution{
				ID:        "opensuse-leap",
				VersionID: tt.version,
				Family:    constants.FamilySUSE,
			}

			repo, err := GetRepository(dist)
			require.NoError(t, err)
			require.NotNil(t, repo)
			assert.Contains(t, repo.URL, tt.wantContain)
		})
	}
}

func TestGetRepository_FedoraVersions(t *testing.T) {
	tests := []struct {
		version     string
		wantContain string
	}{
		{"40", "40"},
		{"39", "39"},
		{"38", "38"},
	}

	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			dist := &distro.Distribution{
				ID:        "fedora",
				VersionID: tt.version,
				Family:    constants.FamilyRHEL,
			}

			repo, err := GetRepository(dist)
			require.NoError(t, err)
			require.NotNil(t, repo)
			assert.Contains(t, repo.URL, tt.wantContain)
		})
	}
}

func TestGetCUDARepository_SUSE(t *testing.T) {
	dist := &distro.Distribution{
		ID:     "opensuse-tumbleweed",
		Family: constants.FamilySUSE,
	}

	// SUSE uses the same repo for CUDA
	repo, err := GetCUDARepository(dist)
	require.NoError(t, err)
	require.NotNil(t, repo)
	assert.Contains(t, repo.URL, "tumbleweed")
}

func TestGetCUDARepository_NilDistribution(t *testing.T) {
	repo, err := GetCUDARepository(nil)
	assert.Error(t, err)
	assert.Nil(t, repo)
}

func TestGetCUDARepository_UnsupportedFamily(t *testing.T) {
	dist := &distro.Distribution{
		ID:     "unknown",
		Family: constants.FamilyUnknown,
	}

	repo, err := GetCUDARepository(dist)
	assert.Error(t, err)
	assert.Nil(t, repo)
}

func TestGetRepository_RHELNoVersion(t *testing.T) {
	// Test RHEL with no version specified (should use default)
	dist := &distro.Distribution{
		ID:     "rhel",
		Family: constants.FamilyRHEL,
	}

	repo, err := GetRepository(dist)
	require.NoError(t, err)
	require.NotNil(t, repo)
	// Should default to version 9
	assert.Contains(t, repo.URL, "9")
}

func TestGetRepository_FedoraNoVersion(t *testing.T) {
	// Test Fedora with no version specified
	dist := &distro.Distribution{
		ID:     "fedora",
		Family: constants.FamilyRHEL,
	}

	repo, err := GetRepository(dist)
	require.NoError(t, err)
	require.NotNil(t, repo)
	// Should default to version 40
	assert.Contains(t, repo.URL, "40")
}

func TestGetRepository_SUSENoVersion(t *testing.T) {
	// Test Leap with no version specified
	dist := &distro.Distribution{
		ID:     "opensuse-leap",
		Family: constants.FamilySUSE,
	}

	repo, err := GetRepository(dist)
	require.NoError(t, err)
	require.NotNil(t, repo)
	// Should default to version 15.5
	assert.Contains(t, repo.URL, "15.5")
}

func TestGetRepository_TumbleweedByName(t *testing.T) {
	// Test detection by name containing "tumbleweed"
	dist := &distro.Distribution{
		ID:     "opensuse",
		Name:   "openSUSE Tumbleweed",
		Family: constants.FamilySUSE,
	}

	repo, err := GetRepository(dist)
	require.NoError(t, err)
	require.NotNil(t, repo)
	assert.Contains(t, repo.URL, "tumbleweed")
}

func TestGetRepositoryInfo_SUSE(t *testing.T) {
	tests := []struct {
		name        string
		dist        *distro.Distribution
		wantContain string
	}{
		{
			name: "tumbleweed",
			dist: &distro.Distribution{
				ID:     "opensuse-tumbleweed",
				Family: constants.FamilySUSE,
			},
			wantContain: "Tumbleweed",
		},
		{
			name: "leap",
			dist: &distro.Distribution{
				ID:        "opensuse-leap",
				VersionID: "15.5",
				Family:    constants.FamilySUSE,
			},
			wantContain: "Leap",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info, err := GetRepositoryInfo(tt.dist)
			require.NoError(t, err)
			require.NotNil(t, info)
			assert.Contains(t, info.Name, tt.wantContain)
		})
	}
}

func TestGetRPMFusionURLs_DefaultVersions(t *testing.T) {
	// Test with no version specified
	distFedora := &distro.Distribution{
		ID:     "fedora",
		Family: constants.FamilyRHEL,
	}

	freeURL, nonfreeURL, err := GetRPMFusionURLs(distFedora)
	require.NoError(t, err)
	assert.Contains(t, freeURL, "40")
	assert.Contains(t, nonfreeURL, "40")

	distRHEL := &distro.Distribution{
		ID:     "rhel",
		Family: constants.FamilyRHEL,
	}

	freeURL, nonfreeURL, err = GetRPMFusionURLs(distRHEL)
	require.NoError(t, err)
	assert.Contains(t, freeURL, "9")
	assert.Contains(t, nonfreeURL, "9")
}

func TestGetCUDARepository_RHELVersions(t *testing.T) {
	tests := []struct {
		name        string
		dist        *distro.Distribution
		wantContain string
	}{
		{
			name: "rhel9",
			dist: &distro.Distribution{
				ID:        "rhel",
				VersionID: "9",
				Family:    constants.FamilyRHEL,
			},
			wantContain: "rhel9",
		},
		{
			name: "rhel_no_version",
			dist: &distro.Distribution{
				ID:     "rhel",
				Family: constants.FamilyRHEL,
			},
			wantContain: "rhel9", // default
		},
		{
			name: "fedora_no_version",
			dist: &distro.Distribution{
				ID:     "fedora",
				Family: constants.FamilyRHEL,
			},
			wantContain: "fedora40", // default
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, err := GetCUDARepository(tt.dist)
			require.NoError(t, err)
			require.NotNil(t, repo)
			assert.Contains(t, repo.URL, tt.wantContain)
		})
	}
}

func TestGetUbuntuRepoPath_Fallback(t *testing.T) {
	// Test with unknown codename - should fallback to default
	dist := &distro.Distribution{
		ID:              "ubuntu",
		VersionCodename: "unknown-codename",
		VersionID:       "99.99",
		Family:          constants.FamilyDebian,
	}

	repo, err := GetCUDARepository(dist)
	require.NoError(t, err)
	require.NotNil(t, repo)
	// Should fallback to ubuntu2204
	assert.Contains(t, repo.URL, "ubuntu2204")
}

func TestGetDebianRepoPath_Fallback(t *testing.T) {
	// Test with unknown codename - should fallback to default
	dist := &distro.Distribution{
		ID:              "debian",
		VersionCodename: "unknown-codename",
		VersionID:       "99",
		Family:          constants.FamilyDebian,
	}

	repo, err := GetCUDARepository(dist)
	require.NoError(t, err)
	require.NotNil(t, repo)
	// Should fallback to debian12
	assert.Contains(t, repo.URL, "debian12")
}

func TestGetRepositoryInfo_RHELNoVersion(t *testing.T) {
	dist := &distro.Distribution{
		ID:     "rhel",
		Family: constants.FamilyRHEL,
	}

	info, err := GetRepositoryInfo(dist)
	require.NoError(t, err)
	require.NotNil(t, info)
	// Should default to version 9
	assert.Contains(t, info.URL, "9")
}

func TestGetRepositoryInfo_SUSENoVersion(t *testing.T) {
	dist := &distro.Distribution{
		ID:     "opensuse-leap",
		Family: constants.FamilySUSE,
	}

	info, err := GetRepositoryInfo(dist)
	require.NoError(t, err)
	require.NotNil(t, info)
	// Should default to 15.5
	assert.Contains(t, info.URL, "15.5")
}

func TestGetRepositoryInfo_UnsupportedFamily(t *testing.T) {
	dist := &distro.Distribution{
		ID:     "unknown",
		Family: constants.FamilyUnknown,
	}

	info, err := GetRepositoryInfo(dist)
	assert.Error(t, err)
	assert.Nil(t, info)
}

func TestGetGPGKeyURL_UnsupportedFamily(t *testing.T) {
	dist := &distro.Distribution{
		ID:     "unknown",
		Family: constants.FamilyUnknown,
	}

	url, err := GetGPGKeyURL(dist)
	assert.Error(t, err)
	assert.Empty(t, url)
}
