package distro

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tungetti/igor/internal/constants"
	"github.com/tungetti/igor/internal/exec"
)

// NOTE: We cannot import github.com/tungetti/igor/internal/testing here
// because that package imports distro, which would create an import cycle.
// Instead, we define local test fixtures and mocks that mirror the patterns
// from internal/testing.

// MockFileReader is a mock implementation of FileReader for testing.
type MockFileReader struct {
	files map[string][]byte
}

// NewMockFileReader creates a new MockFileReader with the given files.
func NewMockFileReader(files map[string][]byte) *MockFileReader {
	if files == nil {
		files = make(map[string][]byte)
	}
	return &MockFileReader{files: files}
}

// ReadFile implements FileReader.
func (m *MockFileReader) ReadFile(path string) ([]byte, error) {
	if content, ok := m.files[path]; ok {
		return content, nil
	}
	return nil, &fileNotFoundError{path: path}
}

// FileExists implements FileReader.
func (m *MockFileReader) FileExists(path string) bool {
	_, ok := m.files[path]
	return ok
}

// SetFile adds or updates a file in the mock filesystem.
func (m *MockFileReader) SetFile(path string, content []byte) {
	m.files[path] = content
}

type fileNotFoundError struct {
	path string
}

func (e *fileNotFoundError) Error() string {
	return "file not found: " + e.path
}

// ============================================================================
// Test Context Helpers
// ============================================================================

// testContext creates a context with a default timeout suitable for testing.
// This mirrors the pattern from internal/testing.ContextWithTimeout.
func testContext(t *testing.T) (context.Context, context.CancelFunc) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	return ctx, cancel
}

// ============================================================================
// Sample OS Release Files for Testing
// ============================================================================

// Sample os-release files for testing
var (
	ubuntu2404OSRelease = `PRETTY_NAME="Ubuntu 24.04 LTS"
NAME="Ubuntu"
VERSION_ID="24.04"
VERSION="24.04 LTS (Noble Numbat)"
VERSION_CODENAME=noble
ID=ubuntu
ID_LIKE=debian
HOME_URL="https://www.ubuntu.com/"
SUPPORT_URL="https://help.ubuntu.com/"
`

	ubuntu2204OSRelease = `PRETTY_NAME="Ubuntu 22.04.3 LTS"
NAME="Ubuntu"
VERSION_ID="22.04"
VERSION="22.04.3 LTS (Jammy Jellyfish)"
VERSION_CODENAME=jammy
ID=ubuntu
ID_LIKE=debian
HOME_URL="https://www.ubuntu.com/"
`

	debian12OSRelease = `PRETTY_NAME="Debian GNU/Linux 12 (bookworm)"
NAME="Debian GNU/Linux"
VERSION_ID="12"
VERSION="12 (bookworm)"
VERSION_CODENAME=bookworm
ID=debian
HOME_URL="https://www.debian.org/"
`

	fedora40OSRelease = `NAME="Fedora Linux"
VERSION="40 (Workstation Edition)"
ID=fedora
VERSION_ID=40
PRETTY_NAME="Fedora Linux 40 (Workstation Edition)"
HOME_URL="https://fedoraproject.org/"
`

	rhel9OSRelease = `NAME="Red Hat Enterprise Linux"
VERSION="9.0 (Plow)"
ID="rhel"
ID_LIKE="fedora"
VERSION_ID="9.0"
PRETTY_NAME="Red Hat Enterprise Linux 9.0 (Plow)"
`

	rocky9OSRelease = `NAME="Rocky Linux"
VERSION="9.0 (Blue Onyx)"
ID="rocky"
ID_LIKE="rhel centos fedora"
VERSION_ID="9.0"
PRETTY_NAME="Rocky Linux 9.0 (Blue Onyx)"
`

	almalinux9OSRelease = `NAME="AlmaLinux"
VERSION="9.0 (Emerald Puma)"
ID="almalinux"
ID_LIKE="rhel centos fedora"
VERSION_ID="9.0"
PRETTY_NAME="AlmaLinux 9.0 (Emerald Puma)"
`

	centos7OSRelease = `NAME="CentOS Linux"
VERSION="7 (Core)"
ID="centos"
ID_LIKE="rhel fedora"
VERSION_ID="7"
PRETTY_NAME="CentOS Linux 7 (Core)"
`

	archOSRelease = `NAME="Arch Linux"
PRETTY_NAME="Arch Linux"
ID=arch
BUILD_ID=rolling
`

	manjaroOSRelease = `NAME="Manjaro Linux"
PRETTY_NAME="Manjaro Linux"
ID=manjaro
ID_LIKE=arch
VERSION_ID="23.1.0"
`

	endeavourOSRelease = `NAME="EndeavourOS"
PRETTY_NAME="EndeavourOS"
ID=endeavouros
ID_LIKE=arch
VERSION_ID="2024.01.25"
BUILD_ID=rolling
`

	opensuse155OSRelease = `NAME="openSUSE Leap"
VERSION="15.5"
ID="opensuse-leap"
ID_LIKE="suse opensuse"
VERSION_ID="15.5"
PRETTY_NAME="openSUSE Leap 15.5"
`

	opensuseTumbleweedOSRelease = `NAME="openSUSE Tumbleweed"
ID="opensuse-tumbleweed"
ID_LIKE="opensuse suse"
VERSION_ID="20240101"
PRETTY_NAME="openSUSE Tumbleweed"
`

	linuxMintOSRelease = `NAME="Linux Mint"
VERSION="21.2 (Victoria)"
ID=linuxmint
ID_LIKE="ubuntu debian"
VERSION_ID="21.2"
VERSION_CODENAME=victoria
PRETTY_NAME="Linux Mint 21.2"
`

	popOSRelease = `NAME="Pop!_OS"
VERSION="22.04 LTS"
ID=pop
ID_LIKE="ubuntu debian"
VERSION_ID="22.04"
VERSION_CODENAME=jammy
PRETTY_NAME="Pop!_OS 22.04 LTS"
`
)

func TestParseOSReleaseContent(t *testing.T) {
	tests := []struct {
		name            string
		content         string
		expectedID      string
		expectedName    string
		expectedVersion string
		expectedFamily  constants.DistroFamily
		expectedIDLike  []string
	}{
		{
			name:            "Ubuntu 24.04",
			content:         ubuntu2404OSRelease,
			expectedID:      "ubuntu",
			expectedName:    "Ubuntu",
			expectedVersion: "24.04",
			expectedFamily:  constants.FamilyDebian,
			expectedIDLike:  []string{"debian"},
		},
		{
			name:            "Debian 12",
			content:         debian12OSRelease,
			expectedID:      "debian",
			expectedName:    "Debian GNU/Linux",
			expectedVersion: "12",
			expectedFamily:  constants.FamilyDebian,
			expectedIDLike:  nil,
		},
		{
			name:            "Fedora 40",
			content:         fedora40OSRelease,
			expectedID:      "fedora",
			expectedName:    "Fedora Linux",
			expectedVersion: "40",
			expectedFamily:  constants.FamilyRHEL,
			expectedIDLike:  nil,
		},
		{
			name:            "RHEL 9",
			content:         rhel9OSRelease,
			expectedID:      "rhel",
			expectedName:    "Red Hat Enterprise Linux",
			expectedVersion: "9.0",
			expectedFamily:  constants.FamilyRHEL,
			expectedIDLike:  []string{"fedora"},
		},
		{
			name:            "Rocky 9",
			content:         rocky9OSRelease,
			expectedID:      "rocky",
			expectedName:    "Rocky Linux",
			expectedVersion: "9.0",
			expectedFamily:  constants.FamilyRHEL,
			expectedIDLike:  []string{"rhel", "centos", "fedora"},
		},
		{
			name:            "AlmaLinux 9",
			content:         almalinux9OSRelease,
			expectedID:      "almalinux",
			expectedName:    "AlmaLinux",
			expectedVersion: "9.0",
			expectedFamily:  constants.FamilyRHEL,
			expectedIDLike:  []string{"rhel", "centos", "fedora"},
		},
		{
			name:            "Arch Linux",
			content:         archOSRelease,
			expectedID:      "arch",
			expectedName:    "Arch Linux",
			expectedVersion: "",
			expectedFamily:  constants.FamilyArch,
			expectedIDLike:  nil,
		},
		{
			name:            "Manjaro",
			content:         manjaroOSRelease,
			expectedID:      "manjaro",
			expectedName:    "Manjaro Linux",
			expectedVersion: "23.1.0",
			expectedFamily:  constants.FamilyArch,
			expectedIDLike:  []string{"arch"},
		},
		{
			name:            "EndeavourOS",
			content:         endeavourOSRelease,
			expectedID:      "endeavouros",
			expectedName:    "EndeavourOS",
			expectedVersion: "2024.01.25",
			expectedFamily:  constants.FamilyArch,
			expectedIDLike:  []string{"arch"},
		},
		{
			name:            "openSUSE Leap 15.5",
			content:         opensuse155OSRelease,
			expectedID:      "opensuse-leap",
			expectedName:    "openSUSE Leap",
			expectedVersion: "15.5",
			expectedFamily:  constants.FamilySUSE,
			expectedIDLike:  []string{"suse", "opensuse"},
		},
		{
			name:            "openSUSE Tumbleweed",
			content:         opensuseTumbleweedOSRelease,
			expectedID:      "opensuse-tumbleweed",
			expectedName:    "openSUSE Tumbleweed",
			expectedVersion: "20240101",
			expectedFamily:  constants.FamilySUSE,
			expectedIDLike:  []string{"opensuse", "suse"},
		},
		{
			name:            "Linux Mint",
			content:         linuxMintOSRelease,
			expectedID:      "linuxmint",
			expectedName:    "Linux Mint",
			expectedVersion: "21.2",
			expectedFamily:  constants.FamilyDebian,
			expectedIDLike:  []string{"ubuntu", "debian"},
		},
		{
			name:            "Pop!_OS",
			content:         popOSRelease,
			expectedID:      "pop",
			expectedName:    "Pop!_OS",
			expectedVersion: "22.04",
			expectedFamily:  constants.FamilyDebian,
			expectedIDLike:  []string{"ubuntu", "debian"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dist, err := ParseOSReleaseContent(tt.content)
			require.NoError(t, err)
			require.NotNil(t, dist)

			assert.Equal(t, tt.expectedID, dist.ID)
			assert.Equal(t, tt.expectedName, dist.Name)
			assert.Equal(t, tt.expectedVersion, dist.VersionID)
			assert.Equal(t, tt.expectedFamily, dist.Family)
			assert.Equal(t, tt.expectedIDLike, dist.IDLike)
		})
	}
}

func TestParseOSReleaseContent_EdgeCases(t *testing.T) {
	t.Run("empty content", func(t *testing.T) {
		dist, err := ParseOSReleaseContent("")
		require.NoError(t, err)
		require.NotNil(t, dist)
		assert.Equal(t, "", dist.ID)
		assert.Equal(t, constants.FamilyUnknown, dist.Family)
	})

	t.Run("content with comments", func(t *testing.T) {
		content := `# This is a comment
ID=ubuntu
# Another comment
NAME="Ubuntu"
`
		dist, err := ParseOSReleaseContent(content)
		require.NoError(t, err)
		assert.Equal(t, "ubuntu", dist.ID)
		assert.Equal(t, "Ubuntu", dist.Name)
	})

	t.Run("content with single quotes", func(t *testing.T) {
		content := `ID='ubuntu'
NAME='Ubuntu'
VERSION_ID='24.04'
`
		dist, err := ParseOSReleaseContent(content)
		require.NoError(t, err)
		assert.Equal(t, "ubuntu", dist.ID)
		assert.Equal(t, "Ubuntu", dist.Name)
		assert.Equal(t, "24.04", dist.VersionID)
	})

	t.Run("content without quotes", func(t *testing.T) {
		content := `ID=arch
NAME=Arch Linux
`
		dist, err := ParseOSReleaseContent(content)
		require.NoError(t, err)
		assert.Equal(t, "arch", dist.ID)
	})

	t.Run("content with leading/trailing whitespace on lines", func(t *testing.T) {
		content := `  ID=ubuntu  
  NAME="Ubuntu"  
`
		dist, err := ParseOSReleaseContent(content)
		require.NoError(t, err)
		assert.Equal(t, "ubuntu", dist.ID)
		assert.Equal(t, "Ubuntu", dist.Name)
	})
}

func TestDetectFamily(t *testing.T) {
	tests := []struct {
		name     string
		id       string
		idLike   []string
		expected constants.DistroFamily
	}{
		// Debian family
		{"debian", "debian", nil, constants.FamilyDebian},
		{"ubuntu", "ubuntu", []string{"debian"}, constants.FamilyDebian},
		{"linuxmint", "linuxmint", []string{"ubuntu", "debian"}, constants.FamilyDebian},
		{"pop", "pop", []string{"ubuntu", "debian"}, constants.FamilyDebian},
		{"elementary", "elementary", []string{"ubuntu"}, constants.FamilyDebian},
		{"zorin", "zorin", []string{"ubuntu"}, constants.FamilyDebian},
		{"kali", "kali", []string{"debian"}, constants.FamilyDebian},

		// RHEL family
		{"fedora", "fedora", nil, constants.FamilyRHEL},
		{"rhel", "rhel", []string{"fedora"}, constants.FamilyRHEL},
		{"centos", "centos", []string{"rhel", "fedora"}, constants.FamilyRHEL},
		{"rocky", "rocky", []string{"rhel", "centos", "fedora"}, constants.FamilyRHEL},
		{"almalinux", "almalinux", []string{"rhel", "centos", "fedora"}, constants.FamilyRHEL},
		{"oracle", "ol", []string{"fedora"}, constants.FamilyRHEL},
		{"amazon", "amzn", []string{"fedora", "rhel"}, constants.FamilyRHEL},

		// Arch family
		{"arch", "arch", nil, constants.FamilyArch},
		{"manjaro", "manjaro", []string{"arch"}, constants.FamilyArch},
		{"endeavouros", "endeavouros", []string{"arch"}, constants.FamilyArch},
		{"garuda", "garuda", []string{"arch"}, constants.FamilyArch},
		{"artix", "artix", []string{"arch"}, constants.FamilyArch},

		// SUSE family
		{"opensuse-leap", "opensuse-leap", []string{"suse", "opensuse"}, constants.FamilySUSE},
		{"opensuse-tumbleweed", "opensuse-tumbleweed", []string{"opensuse", "suse"}, constants.FamilySUSE},
		{"sles", "sles", []string{"suse"}, constants.FamilySUSE},

		// Unknown
		{"unknown", "someunknowndistro", nil, constants.FamilyUnknown},
		{"unknown with unknown idlike", "someunknowndistro", []string{"somethingelse"}, constants.FamilyUnknown},

		// ID_LIKE based detection
		{"derivative from debian", "customdistro", []string{"debian"}, constants.FamilyDebian},
		{"derivative from ubuntu", "customdistro", []string{"ubuntu"}, constants.FamilyDebian},
		{"derivative from arch", "customdistro", []string{"arch"}, constants.FamilyArch},
		{"derivative from fedora", "customdistro", []string{"fedora"}, constants.FamilyRHEL},
		{"derivative from suse", "customdistro", []string{"suse"}, constants.FamilySUSE},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			family := DetectFamily(tt.id, tt.idLike)
			assert.Equal(t, tt.expected, family)
		})
	}
}

func TestDistribution_Methods(t *testing.T) {
	t.Run("String", func(t *testing.T) {
		tests := []struct {
			name     string
			dist     Distribution
			expected string
		}{
			{
				name:     "with PrettyName",
				dist:     Distribution{PrettyName: "Ubuntu 24.04 LTS"},
				expected: "Ubuntu 24.04 LTS",
			},
			{
				name:     "with Name and Version",
				dist:     Distribution{Name: "Ubuntu", VersionID: "24.04"},
				expected: "Ubuntu 24.04",
			},
			{
				name:     "with Name only",
				dist:     Distribution{Name: "Ubuntu"},
				expected: "Ubuntu",
			},
			{
				name:     "with ID only",
				dist:     Distribution{ID: "ubuntu"},
				expected: "ubuntu",
			},
			{
				name:     "empty",
				dist:     Distribution{},
				expected: "Unknown Distribution",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				assert.Equal(t, tt.expected, tt.dist.String())
			})
		}
	})

	t.Run("IsFamily methods", func(t *testing.T) {
		debianDist := Distribution{Family: constants.FamilyDebian}
		rhelDist := Distribution{Family: constants.FamilyRHEL}
		archDist := Distribution{Family: constants.FamilyArch}
		suseDist := Distribution{Family: constants.FamilySUSE}
		unknownDist := Distribution{Family: constants.FamilyUnknown}

		assert.True(t, debianDist.IsDebian())
		assert.False(t, debianDist.IsRHEL())
		assert.False(t, debianDist.IsArch())
		assert.False(t, debianDist.IsSUSE())
		assert.False(t, debianDist.IsUnknown())

		assert.True(t, rhelDist.IsRHEL())
		assert.True(t, archDist.IsArch())
		assert.True(t, suseDist.IsSUSE())
		assert.True(t, unknownDist.IsUnknown())
	})

	t.Run("MajorVersion", func(t *testing.T) {
		tests := []struct {
			versionID string
			expected  string
		}{
			{"24.04", "24"},
			{"22.04", "22"},
			{"9.0", "9"},
			{"15.5", "15"},
			{"40", "40"},
			{"2024.01.25", "2024"},
			{"15-sp5", "15"},
			{"", ""},
		}

		for _, tt := range tests {
			t.Run(tt.versionID, func(t *testing.T) {
				dist := Distribution{VersionID: tt.versionID}
				assert.Equal(t, tt.expected, dist.MajorVersion())
			})
		}
	})

	t.Run("MinorVersion", func(t *testing.T) {
		tests := []struct {
			versionID string
			expected  string
		}{
			{"24.04", "04"},
			{"9.0", "0"},
			{"15.5", "5"},
			{"40", ""},
			{"15.5.1", "5"},
			{"", ""},
		}

		for _, tt := range tests {
			t.Run(tt.versionID, func(t *testing.T) {
				dist := Distribution{VersionID: tt.versionID}
				assert.Equal(t, tt.expected, dist.MinorVersion())
			})
		}
	})

	t.Run("IsRolling", func(t *testing.T) {
		archDist := Distribution{ID: "arch", Family: constants.FamilyArch}
		manjaroDist := Distribution{ID: "manjaro", Family: constants.FamilyArch}
		tumbleweedDist := Distribution{ID: "opensuse-tumbleweed", Family: constants.FamilySUSE}
		ubuntuDist := Distribution{ID: "ubuntu", Family: constants.FamilyDebian, VersionID: "24.04"}
		rollingWithBuildID := Distribution{ID: "somerolling", BuildID: "20240101"}

		assert.True(t, archDist.IsRolling())
		assert.True(t, manjaroDist.IsRolling())
		assert.True(t, tumbleweedDist.IsRolling())
		assert.False(t, ubuntuDist.IsRolling())
		assert.True(t, rollingWithBuildID.IsRolling())
	})
}

func TestDetector_Detect(t *testing.T) {
	t.Run("detects from os-release", func(t *testing.T) {
		mockFS := NewMockFileReader(map[string][]byte{
			"/etc/os-release": []byte(ubuntu2404OSRelease),
		})
		mockExec := exec.NewMockExecutor()
		detector := NewDetector(mockExec, mockFS)

		dist, err := detector.Detect(context.Background())
		require.NoError(t, err)
		require.NotNil(t, dist)

		assert.Equal(t, "ubuntu", dist.ID)
		assert.Equal(t, "24.04", dist.VersionID)
		assert.Equal(t, constants.FamilyDebian, dist.Family)
	})

	t.Run("falls back to /usr/lib/os-release", func(t *testing.T) {
		mockFS := NewMockFileReader(map[string][]byte{
			"/usr/lib/os-release": []byte(fedora40OSRelease),
		})
		mockExec := exec.NewMockExecutor()
		detector := NewDetector(mockExec, mockFS)

		dist, err := detector.Detect(context.Background())
		require.NoError(t, err)
		require.NotNil(t, dist)

		assert.Equal(t, "fedora", dist.ID)
		assert.Equal(t, constants.FamilyRHEL, dist.Family)
	})

	t.Run("falls back to lsb-release", func(t *testing.T) {
		lsbRelease := `DISTRIB_ID=Ubuntu
DISTRIB_RELEASE=22.04
DISTRIB_CODENAME=jammy
DISTRIB_DESCRIPTION="Ubuntu 22.04 LTS"
`
		mockFS := NewMockFileReader(map[string][]byte{
			"/etc/lsb-release": []byte(lsbRelease),
		})
		mockExec := exec.NewMockExecutor()
		detector := NewDetector(mockExec, mockFS)

		dist, err := detector.Detect(context.Background())
		require.NoError(t, err)
		require.NotNil(t, dist)

		assert.Equal(t, "ubuntu", dist.ID)
		assert.Equal(t, "22.04", dist.VersionID)
		assert.Equal(t, "jammy", dist.VersionCodename)
	})

	t.Run("falls back to redhat-release", func(t *testing.T) {
		mockFS := NewMockFileReader(map[string][]byte{
			"/etc/redhat-release": []byte("Rocky Linux release 9.0 (Blue Onyx)"),
		})
		mockExec := exec.NewMockExecutor()
		detector := NewDetector(mockExec, mockFS)

		dist, err := detector.Detect(context.Background())
		require.NoError(t, err)
		require.NotNil(t, dist)

		assert.Equal(t, "rocky", dist.ID)
		assert.Equal(t, constants.FamilyRHEL, dist.Family)
	})

	t.Run("falls back to debian_version", func(t *testing.T) {
		mockFS := NewMockFileReader(map[string][]byte{
			"/etc/debian_version": []byte("12.0"),
		})
		mockExec := exec.NewMockExecutor()
		detector := NewDetector(mockExec, mockFS)

		dist, err := detector.Detect(context.Background())
		require.NoError(t, err)
		require.NotNil(t, dist)

		assert.Equal(t, "debian", dist.ID)
		assert.Equal(t, "12.0", dist.VersionID)
		assert.Equal(t, constants.FamilyDebian, dist.Family)
	})

	t.Run("falls back to arch-release", func(t *testing.T) {
		mockFS := NewMockFileReader(map[string][]byte{
			"/etc/arch-release": []byte(""),
		})
		mockExec := exec.NewMockExecutor()
		detector := NewDetector(mockExec, mockFS)

		dist, err := detector.Detect(context.Background())
		require.NoError(t, err)
		require.NotNil(t, dist)

		assert.Equal(t, "arch", dist.ID)
		assert.Equal(t, constants.FamilyArch, dist.Family)
	})

	t.Run("falls back to lsb_release command", func(t *testing.T) {
		mockFS := NewMockFileReader(nil)
		mockExec := exec.NewMockExecutor()
		mockExec.SetResponse("lsb_release", exec.SuccessResult(`Distributor ID:	Ubuntu
Description:	Ubuntu 22.04 LTS
Release:	22.04
Codename:	jammy
`))
		detector := NewDetector(mockExec, mockFS)

		dist, err := detector.Detect(context.Background())
		require.NoError(t, err)
		require.NotNil(t, dist)

		assert.Equal(t, "ubuntu", dist.ID)
		assert.Equal(t, "22.04", dist.VersionID)
	})

	t.Run("returns error when no detection method works", func(t *testing.T) {
		mockFS := NewMockFileReader(nil)
		mockExec := exec.NewMockExecutor()
		mockExec.SetDefaultResponse(exec.FailureResult(1, "command not found"))
		detector := NewDetector(mockExec, mockFS)

		dist, err := detector.Detect(context.Background())
		assert.Error(t, err)
		assert.Nil(t, dist)
	})

	t.Run("respects context cancellation", func(t *testing.T) {
		mockFS := NewMockFileReader(nil)
		mockExec := exec.NewMockExecutor()
		detector := NewDetector(mockExec, mockFS)

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		dist, err := detector.Detect(ctx)
		assert.Error(t, err)
		assert.Nil(t, dist)
	})

	t.Run("respects context timeout", func(t *testing.T) {
		mockFS := NewMockFileReader(nil)
		mockExec := exec.NewMockExecutor()
		detector := NewDetector(mockExec, mockFS)

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
		defer cancel()
		time.Sleep(10 * time.Millisecond) // Ensure timeout

		dist, err := detector.Detect(ctx)
		assert.Error(t, err)
		assert.Nil(t, dist)
	})
}

func TestDetector_AllDistributions(t *testing.T) {
	distributions := []struct {
		name           string
		content        string
		expectedID     string
		expectedFamily constants.DistroFamily
	}{
		{"Ubuntu 24.04", ubuntu2404OSRelease, "ubuntu", constants.FamilyDebian},
		{"Ubuntu 22.04", ubuntu2204OSRelease, "ubuntu", constants.FamilyDebian},
		{"Debian 12", debian12OSRelease, "debian", constants.FamilyDebian},
		{"Fedora 40", fedora40OSRelease, "fedora", constants.FamilyRHEL},
		{"RHEL 9", rhel9OSRelease, "rhel", constants.FamilyRHEL},
		{"Rocky 9", rocky9OSRelease, "rocky", constants.FamilyRHEL},
		{"AlmaLinux 9", almalinux9OSRelease, "almalinux", constants.FamilyRHEL},
		{"CentOS 7", centos7OSRelease, "centos", constants.FamilyRHEL},
		{"Arch Linux", archOSRelease, "arch", constants.FamilyArch},
		{"Manjaro", manjaroOSRelease, "manjaro", constants.FamilyArch},
		{"EndeavourOS", endeavourOSRelease, "endeavouros", constants.FamilyArch},
		{"openSUSE Leap 15.5", opensuse155OSRelease, "opensuse-leap", constants.FamilySUSE},
		{"openSUSE Tumbleweed", opensuseTumbleweedOSRelease, "opensuse-tumbleweed", constants.FamilySUSE},
		{"Linux Mint", linuxMintOSRelease, "linuxmint", constants.FamilyDebian},
		{"Pop!_OS", popOSRelease, "pop", constants.FamilyDebian},
	}

	for _, tt := range distributions {
		t.Run(tt.name, func(t *testing.T) {
			mockFS := NewMockFileReader(map[string][]byte{
				"/etc/os-release": []byte(tt.content),
			})
			mockExec := exec.NewMockExecutor()
			detector := NewDetector(mockExec, mockFS)

			dist, err := detector.Detect(context.Background())
			require.NoError(t, err)
			require.NotNil(t, dist)

			assert.Equal(t, tt.expectedID, dist.ID, "ID mismatch for %s", tt.name)
			assert.Equal(t, tt.expectedFamily, dist.Family, "Family mismatch for %s", tt.name)
		})
	}
}

func TestNewDetector_NilFileReader(t *testing.T) {
	detector := NewDetector(nil, nil)
	assert.NotNil(t, detector.fsReader)
	_, ok := detector.fsReader.(*DefaultFileReader)
	assert.True(t, ok, "should use DefaultFileReader when nil is passed")
}

func TestParseLSBReleaseContent(t *testing.T) {
	content := `DISTRIB_ID=Ubuntu
DISTRIB_RELEASE=22.04
DISTRIB_CODENAME=jammy
DISTRIB_DESCRIPTION="Ubuntu 22.04.3 LTS"
`
	dist, err := ParseLSBReleaseContent(content)
	require.NoError(t, err)
	require.NotNil(t, dist)

	assert.Equal(t, "ubuntu", dist.ID)
	assert.Equal(t, "Ubuntu", dist.Name)
	assert.Equal(t, "22.04", dist.VersionID)
	assert.Equal(t, "jammy", dist.VersionCodename)
	assert.Equal(t, "Ubuntu 22.04.3 LTS", dist.PrettyName)
	assert.Equal(t, constants.FamilyDebian, dist.Family)
}

func TestFallbackParsers(t *testing.T) {
	t.Run("parseRedHatRelease with various formats", func(t *testing.T) {
		mockFS := NewMockFileReader(nil)
		detector := NewDetector(nil, mockFS)

		tests := []struct {
			content    string
			expectedID string
		}{
			{"Red Hat Enterprise Linux release 9.0 (Plow)", "rhel"},
			{"CentOS Linux release 7.9.2009 (Core)", "centos"},
			{"Fedora release 40 (Forty)", "fedora"},
			{"Rocky Linux release 9.0 (Blue Onyx)", "rocky"},
			{"AlmaLinux release 9.0 (Emerald Puma)", "almalinux"},
		}

		for _, tt := range tests {
			t.Run(tt.content, func(t *testing.T) {
				dist := detector.parseRedHatRelease(tt.content)
				assert.Equal(t, tt.expectedID, dist.ID)
				assert.Equal(t, constants.FamilyRHEL, dist.Family)
			})
		}
	})

	t.Run("parseDebianVersion", func(t *testing.T) {
		mockFS := NewMockFileReader(nil)
		detector := NewDetector(nil, mockFS)

		tests := []struct {
			content          string
			expectedVersion  string
			expectedCodename string
		}{
			{"12.0", "12.0", ""},
			{"bookworm/sid", "", "bookworm"},
			{"11.7", "11.7", ""},
		}

		for _, tt := range tests {
			t.Run(tt.content, func(t *testing.T) {
				dist := detector.parseDebianVersion(tt.content)
				assert.Equal(t, "debian", dist.ID)
				assert.Equal(t, tt.expectedVersion, dist.VersionID)
				assert.Equal(t, tt.expectedCodename, dist.VersionCodename)
				assert.Equal(t, constants.FamilyDebian, dist.Family)
			})
		}
	})
}

func TestFallbackParsers_SuSERelease(t *testing.T) {
	mockFS := NewMockFileReader(nil)
	detector := NewDetector(nil, mockFS)

	tests := []struct {
		name       string
		content    string
		expectedID string
	}{
		{
			name: "openSUSE Leap",
			content: `openSUSE Leap 15.5
VERSION = 15.5
`,
			expectedID: "opensuse-leap",
		},
		{
			name: "openSUSE Tumbleweed",
			content: `openSUSE Tumbleweed
VERSION = 20240101
`,
			expectedID: "opensuse-tumbleweed",
		},
		{
			name: "SLES",
			content: `SUSE Linux Enterprise Server 15
VERSION = 15
`,
			expectedID: "sles",
		},
		{
			name:       "Generic openSUSE",
			content:    `openSUSE 13.2`,
			expectedID: "opensuse",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dist := detector.parseSuSERelease(tt.content)
			assert.Equal(t, tt.expectedID, dist.ID)
			assert.Equal(t, constants.FamilySUSE, dist.Family)
		})
	}
}

func TestFallbackFiles_SuSERelease(t *testing.T) {
	mockFS := NewMockFileReader(map[string][]byte{
		"/etc/SuSE-release": []byte(`openSUSE Leap 15.5
VERSION = 15.5
`),
	})
	mockExec := exec.NewMockExecutor()
	detector := NewDetector(mockExec, mockFS)

	dist, err := detector.Detect(context.Background())
	require.NoError(t, err)
	require.NotNil(t, dist)

	assert.Equal(t, "opensuse-leap", dist.ID)
	assert.Equal(t, constants.FamilySUSE, dist.Family)
}

func TestParseOSRelease_WithFilesystem(t *testing.T) {
	t.Run("returns error for non-existent file", func(t *testing.T) {
		_, err := ParseOSRelease("/nonexistent/os-release")
		assert.Error(t, err)
	})
}

func TestParseLSBRelease_WithFilesystem(t *testing.T) {
	t.Run("returns error for non-existent file", func(t *testing.T) {
		_, err := ParseLSBRelease("/nonexistent/lsb-release")
		assert.Error(t, err)
	})
}

func TestParseIDLike_EmptyValue(t *testing.T) {
	result := parseIDLike("")
	assert.Nil(t, result)
}

func TestUnquote_ShortString(t *testing.T) {
	// Test strings that are too short to have quotes
	assert.Equal(t, "a", unquote("a"))
	assert.Equal(t, "", unquote(""))
}

func TestDetector_tryOSRelease_FileReadError(t *testing.T) {
	// Create a mock that returns an error for ReadFile but exists for FileExists
	mockFS := &errorOnReadFileReader{}
	detector := NewDetector(nil, mockFS)

	_, err := detector.tryOSRelease("/etc/os-release")
	assert.Error(t, err)
}

type errorOnReadFileReader struct{}

func (e *errorOnReadFileReader) ReadFile(path string) ([]byte, error) {
	return nil, &fileNotFoundError{path: path}
}

func (e *errorOnReadFileReader) FileExists(path string) bool {
	return true // File exists but can't be read
}

func TestDetector_tryLSBReleaseCommand_NoExecutor(t *testing.T) {
	mockFS := NewMockFileReader(nil)
	detector := NewDetector(nil, mockFS) // nil executor

	_, err := detector.tryLSBReleaseCommand(context.Background())
	assert.Error(t, err)
}

func TestDetector_tryLSBReleaseCommand_EmptyID(t *testing.T) {
	mockFS := NewMockFileReader(nil)
	mockExec := exec.NewMockExecutor()
	mockExec.SetResponse("lsb_release", exec.SuccessResult(`Description:	Some Linux
Release:	1.0
`)) // No Distributor ID
	detector := NewDetector(mockExec, mockFS)

	_, err := detector.tryLSBReleaseCommand(context.Background())
	assert.Error(t, err)
}

func TestParseRedHatRelease_UnknownDistro(t *testing.T) {
	mockFS := NewMockFileReader(nil)
	detector := NewDetector(nil, mockFS)

	dist := detector.parseRedHatRelease("Some Unknown Red Hat Based Distro release 1.0")
	assert.Equal(t, "rhel", dist.ID)
	assert.Equal(t, constants.FamilyRHEL, dist.Family)
}

func TestParseRedHatRelease_NoVersion(t *testing.T) {
	mockFS := NewMockFileReader(nil)
	detector := NewDetector(nil, mockFS)

	dist := detector.parseRedHatRelease("Fedora release")
	assert.Equal(t, "fedora", dist.ID)
	assert.Equal(t, "", dist.VersionID)
}

func TestDefaultFileReader(t *testing.T) {
	reader := &DefaultFileReader{}

	t.Run("ReadFile returns error for non-existent file", func(t *testing.T) {
		_, err := reader.ReadFile("/nonexistent/path/to/file")
		assert.Error(t, err)
	})

	t.Run("FileExists returns false for non-existent file", func(t *testing.T) {
		exists := reader.FileExists("/nonexistent/path/to/file")
		assert.False(t, exists)
	})
}

func TestParseLSBReleaseContent_EmptyAndEdgeCases(t *testing.T) {
	t.Run("empty content", func(t *testing.T) {
		dist, err := ParseLSBReleaseContent("")
		require.NoError(t, err)
		assert.Equal(t, "", dist.ID)
	})

	t.Run("with comments", func(t *testing.T) {
		content := `# Comment
DISTRIB_ID=Ubuntu
`
		dist, err := ParseLSBReleaseContent(content)
		require.NoError(t, err)
		assert.Equal(t, "ubuntu", dist.ID)
	})

	t.Run("line without equals", func(t *testing.T) {
		content := `DISTRIB_ID=Ubuntu
SomeInvalidLine
DISTRIB_RELEASE=22.04
`
		dist, err := ParseLSBReleaseContent(content)
		require.NoError(t, err)
		assert.Equal(t, "ubuntu", dist.ID)
		assert.Equal(t, "22.04", dist.VersionID)
	})
}
