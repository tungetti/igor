package factory

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tungetti/igor/internal/constants"
	"github.com/tungetti/igor/internal/distro"
	"github.com/tungetti/igor/internal/exec"
	"github.com/tungetti/igor/internal/pkg/apt"
	"github.com/tungetti/igor/internal/pkg/dnf"
	"github.com/tungetti/igor/internal/pkg/pacman"
	"github.com/tungetti/igor/internal/pkg/yum"
	"github.com/tungetti/igor/internal/pkg/zypper"
	"github.com/tungetti/igor/internal/privilege"
)

// MockFileReader for testing
type MockFileReader struct {
	files map[string][]byte
}

func NewMockFileReader(files map[string][]byte) *MockFileReader {
	if files == nil {
		files = make(map[string][]byte)
	}
	return &MockFileReader{files: files}
}

func (m *MockFileReader) ReadFile(path string) ([]byte, error) {
	if content, ok := m.files[path]; ok {
		return content, nil
	}
	return nil, &fileNotFoundError{path: path}
}

func (m *MockFileReader) FileExists(path string) bool {
	_, ok := m.files[path]
	return ok
}

type fileNotFoundError struct {
	path string
}

func (e *fileNotFoundError) Error() string {
	return "file not found: " + e.path
}

// Sample os-release content for different distributions
var (
	ubuntu2404OSRelease = `PRETTY_NAME="Ubuntu 24.04 LTS"
NAME="Ubuntu"
VERSION_ID="24.04"
ID=ubuntu
ID_LIKE=debian
`

	debian12OSRelease = `PRETTY_NAME="Debian GNU/Linux 12 (bookworm)"
NAME="Debian GNU/Linux"
VERSION_ID="12"
ID=debian
`

	fedora40OSRelease = `NAME="Fedora Linux"
VERSION="40 (Workstation Edition)"
ID=fedora
VERSION_ID=40
`

	rhel9OSRelease = `NAME="Red Hat Enterprise Linux"
VERSION="9.0 (Plow)"
ID="rhel"
ID_LIKE="fedora"
VERSION_ID="9.0"
`

	rhel7OSRelease = `NAME="Red Hat Enterprise Linux"
VERSION="7.9 (Maipo)"
ID="rhel"
ID_LIKE="fedora"
VERSION_ID="7.9"
`

	centos7OSRelease = `NAME="CentOS Linux"
VERSION="7 (Core)"
ID="centos"
ID_LIKE="rhel fedora"
VERSION_ID="7"
`

	centos8OSRelease = `NAME="CentOS Stream"
VERSION="8"
ID="centos"
ID_LIKE="rhel fedora"
VERSION_ID="8"
`

	rocky9OSRelease = `NAME="Rocky Linux"
VERSION="9.0 (Blue Onyx)"
ID="rocky"
ID_LIKE="rhel centos fedora"
VERSION_ID="9.0"
`

	almalinux9OSRelease = `NAME="AlmaLinux"
VERSION="9.0 (Emerald Puma)"
ID="almalinux"
ID_LIKE="rhel centos fedora"
VERSION_ID="9.0"
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
`

	opensuse155OSRelease = `NAME="openSUSE Leap"
VERSION="15.5"
ID="opensuse-leap"
ID_LIKE="suse opensuse"
VERSION_ID="15.5"
`

	opensuseTumbleweedOSRelease = `NAME="openSUSE Tumbleweed"
ID="opensuse-tumbleweed"
ID_LIKE="opensuse suse"
VERSION_ID="20240101"
`

	linuxMintOSRelease = `NAME="Linux Mint"
VERSION="21.2 (Victoria)"
ID=linuxmint
ID_LIKE="ubuntu debian"
VERSION_ID="21.2"
`

	popOSRelease = `NAME="Pop!_OS"
VERSION="22.04 LTS"
ID=pop
ID_LIKE="ubuntu debian"
VERSION_ID="22.04"
`

	unknownOSRelease = `NAME="Unknown Linux"
ID="unknownos"
VERSION_ID="1.0"
`
)

func TestNewFactory(t *testing.T) {
	mockExec := exec.NewMockExecutor()
	priv := privilege.NewManager()
	mockFS := NewMockFileReader(nil)
	detector := distro.NewDetector(mockExec, mockFS)

	factory := NewFactory(mockExec, priv, detector)

	assert.NotNil(t, factory)
	assert.Equal(t, mockExec, factory.executor)
	assert.Equal(t, priv, factory.privilege)
	assert.Equal(t, detector, factory.detector)
}

func TestFactory_CreateForFamily(t *testing.T) {
	mockExec := exec.NewMockExecutor()
	priv := privilege.NewManager()
	factory := NewFactory(mockExec, priv, nil)

	tests := []struct {
		name           string
		family         constants.DistroFamily
		expectError    bool
		expectedName   string
		expectedFamily constants.DistroFamily
	}{
		{
			name:           "Debian family returns APT",
			family:         constants.FamilyDebian,
			expectError:    false,
			expectedName:   "apt",
			expectedFamily: constants.FamilyDebian,
		},
		{
			name:           "RHEL family returns DNF",
			family:         constants.FamilyRHEL,
			expectError:    false,
			expectedName:   "dnf",
			expectedFamily: constants.FamilyRHEL,
		},
		{
			name:           "Arch family returns Pacman",
			family:         constants.FamilyArch,
			expectError:    false,
			expectedName:   "pacman",
			expectedFamily: constants.FamilyArch,
		},
		{
			name:           "SUSE family returns Zypper",
			family:         constants.FamilySUSE,
			expectError:    false,
			expectedName:   "zypper",
			expectedFamily: constants.FamilySUSE,
		},
		{
			name:        "Unknown family returns error",
			family:      constants.FamilyUnknown,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mgr, err := factory.CreateForFamily(tt.family)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, mgr)
				assert.ErrorIs(t, err, ErrUnsupportedDistro)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, mgr)
				assert.Equal(t, tt.expectedName, mgr.Name())
				assert.Equal(t, tt.expectedFamily, mgr.Family())
			}
		})
	}
}

func TestFactory_CreateForFamily_ReturnsCorrectTypes(t *testing.T) {
	mockExec := exec.NewMockExecutor()
	priv := privilege.NewManager()
	factory := NewFactory(mockExec, priv, nil)

	t.Run("Debian family returns APT Manager", func(t *testing.T) {
		mgr, err := factory.CreateForFamily(constants.FamilyDebian)
		require.NoError(t, err)
		_, ok := mgr.(*apt.Manager)
		assert.True(t, ok, "expected *apt.Manager")
	})

	t.Run("RHEL family returns DNF Manager", func(t *testing.T) {
		mgr, err := factory.CreateForFamily(constants.FamilyRHEL)
		require.NoError(t, err)
		_, ok := mgr.(*dnf.Manager)
		assert.True(t, ok, "expected *dnf.Manager")
	})

	t.Run("Arch family returns Pacman Manager", func(t *testing.T) {
		mgr, err := factory.CreateForFamily(constants.FamilyArch)
		require.NoError(t, err)
		_, ok := mgr.(*pacman.Manager)
		assert.True(t, ok, "expected *pacman.Manager")
	})

	t.Run("SUSE family returns Zypper Manager", func(t *testing.T) {
		mgr, err := factory.CreateForFamily(constants.FamilySUSE)
		require.NoError(t, err)
		_, ok := mgr.(*zypper.Manager)
		assert.True(t, ok, "expected *zypper.Manager")
	})
}

func TestFactory_CreateForDistribution(t *testing.T) {
	mockExec := exec.NewMockExecutor()
	priv := privilege.NewManager()
	factory := NewFactory(mockExec, priv, nil)

	tests := []struct {
		name           string
		distro         *distro.Distribution
		expectError    bool
		expectedName   string
		expectedFamily constants.DistroFamily
	}{
		{
			name: "Ubuntu returns APT",
			distro: &distro.Distribution{
				ID:        "ubuntu",
				VersionID: "24.04",
				Family:    constants.FamilyDebian,
			},
			expectError:    false,
			expectedName:   "apt",
			expectedFamily: constants.FamilyDebian,
		},
		{
			name: "Debian returns APT",
			distro: &distro.Distribution{
				ID:        "debian",
				VersionID: "12",
				Family:    constants.FamilyDebian,
			},
			expectError:    false,
			expectedName:   "apt",
			expectedFamily: constants.FamilyDebian,
		},
		{
			name: "Linux Mint returns APT",
			distro: &distro.Distribution{
				ID:        "linuxmint",
				VersionID: "21.2",
				Family:    constants.FamilyDebian,
			},
			expectError:    false,
			expectedName:   "apt",
			expectedFamily: constants.FamilyDebian,
		},
		{
			name: "Pop!_OS returns APT",
			distro: &distro.Distribution{
				ID:        "pop",
				VersionID: "22.04",
				Family:    constants.FamilyDebian,
			},
			expectError:    false,
			expectedName:   "apt",
			expectedFamily: constants.FamilyDebian,
		},
		{
			name: "Fedora returns DNF",
			distro: &distro.Distribution{
				ID:        "fedora",
				VersionID: "40",
				Family:    constants.FamilyRHEL,
			},
			expectError:    false,
			expectedName:   "dnf",
			expectedFamily: constants.FamilyRHEL,
		},
		{
			name: "RHEL 9 returns DNF",
			distro: &distro.Distribution{
				ID:        "rhel",
				VersionID: "9.0",
				Family:    constants.FamilyRHEL,
			},
			expectError:    false,
			expectedName:   "dnf",
			expectedFamily: constants.FamilyRHEL,
		},
		{
			name: "Rocky 9 returns DNF",
			distro: &distro.Distribution{
				ID:        "rocky",
				VersionID: "9.0",
				Family:    constants.FamilyRHEL,
			},
			expectError:    false,
			expectedName:   "dnf",
			expectedFamily: constants.FamilyRHEL,
		},
		{
			name: "AlmaLinux 9 returns DNF",
			distro: &distro.Distribution{
				ID:        "almalinux",
				VersionID: "9.0",
				Family:    constants.FamilyRHEL,
			},
			expectError:    false,
			expectedName:   "dnf",
			expectedFamily: constants.FamilyRHEL,
		},
		{
			name: "CentOS 8 returns DNF",
			distro: &distro.Distribution{
				ID:        "centos",
				VersionID: "8",
				Family:    constants.FamilyRHEL,
			},
			expectError:    false,
			expectedName:   "dnf",
			expectedFamily: constants.FamilyRHEL,
		},
		{
			name: "Arch Linux returns Pacman",
			distro: &distro.Distribution{
				ID:     "arch",
				Family: constants.FamilyArch,
			},
			expectError:    false,
			expectedName:   "pacman",
			expectedFamily: constants.FamilyArch,
		},
		{
			name: "Manjaro returns Pacman",
			distro: &distro.Distribution{
				ID:        "manjaro",
				VersionID: "23.1.0",
				Family:    constants.FamilyArch,
			},
			expectError:    false,
			expectedName:   "pacman",
			expectedFamily: constants.FamilyArch,
		},
		{
			name: "EndeavourOS returns Pacman",
			distro: &distro.Distribution{
				ID:        "endeavouros",
				VersionID: "2024.01.25",
				Family:    constants.FamilyArch,
			},
			expectError:    false,
			expectedName:   "pacman",
			expectedFamily: constants.FamilyArch,
		},
		{
			name: "openSUSE Leap returns Zypper",
			distro: &distro.Distribution{
				ID:        "opensuse-leap",
				VersionID: "15.5",
				Family:    constants.FamilySUSE,
			},
			expectError:    false,
			expectedName:   "zypper",
			expectedFamily: constants.FamilySUSE,
		},
		{
			name: "openSUSE Tumbleweed returns Zypper",
			distro: &distro.Distribution{
				ID:        "opensuse-tumbleweed",
				VersionID: "20240101",
				Family:    constants.FamilySUSE,
			},
			expectError:    false,
			expectedName:   "zypper",
			expectedFamily: constants.FamilySUSE,
		},
		{
			name: "Unknown distribution returns error",
			distro: &distro.Distribution{
				ID:        "unknownos",
				VersionID: "1.0",
				Family:    constants.FamilyUnknown,
			},
			expectError: true,
		},
		{
			name:        "Nil distribution returns error",
			distro:      nil,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mgr, err := factory.CreateForDistribution(tt.distro)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, mgr)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, mgr)
				assert.Equal(t, tt.expectedName, mgr.Name())
				assert.Equal(t, tt.expectedFamily, mgr.Family())
			}
		})
	}
}

func TestFactory_CreateForDistribution_CentOS7VsCentOS8(t *testing.T) {
	mockExec := exec.NewMockExecutor()
	priv := privilege.NewManager()
	factory := NewFactory(mockExec, priv, nil)

	t.Run("CentOS 7 returns YUM", func(t *testing.T) {
		dist := &distro.Distribution{
			ID:        "centos",
			VersionID: "7",
			Family:    constants.FamilyRHEL,
		}
		mgr, err := factory.CreateForDistribution(dist)
		require.NoError(t, err)
		assert.Equal(t, "yum", mgr.Name())
		_, ok := mgr.(*yum.Manager)
		assert.True(t, ok, "expected *yum.Manager for CentOS 7")
	})

	t.Run("CentOS 7.9 returns YUM", func(t *testing.T) {
		dist := &distro.Distribution{
			ID:        "centos",
			VersionID: "7.9",
			Family:    constants.FamilyRHEL,
		}
		mgr, err := factory.CreateForDistribution(dist)
		require.NoError(t, err)
		assert.Equal(t, "yum", mgr.Name())
		_, ok := mgr.(*yum.Manager)
		assert.True(t, ok, "expected *yum.Manager for CentOS 7.9")
	})

	t.Run("CentOS 8 returns DNF", func(t *testing.T) {
		dist := &distro.Distribution{
			ID:        "centos",
			VersionID: "8",
			Family:    constants.FamilyRHEL,
		}
		mgr, err := factory.CreateForDistribution(dist)
		require.NoError(t, err)
		assert.Equal(t, "dnf", mgr.Name())
		_, ok := mgr.(*dnf.Manager)
		assert.True(t, ok, "expected *dnf.Manager for CentOS 8")
	})

	t.Run("CentOS 8.5 returns DNF", func(t *testing.T) {
		dist := &distro.Distribution{
			ID:        "centos",
			VersionID: "8.5",
			Family:    constants.FamilyRHEL,
		}
		mgr, err := factory.CreateForDistribution(dist)
		require.NoError(t, err)
		assert.Equal(t, "dnf", mgr.Name())
	})

	t.Run("CentOS Stream 9 returns DNF", func(t *testing.T) {
		dist := &distro.Distribution{
			ID:        "centos",
			VersionID: "9",
			Family:    constants.FamilyRHEL,
		}
		mgr, err := factory.CreateForDistribution(dist)
		require.NoError(t, err)
		assert.Equal(t, "dnf", mgr.Name())
	})
}

func TestFactory_CreateForDistribution_RHEL7VsRHEL8(t *testing.T) {
	mockExec := exec.NewMockExecutor()
	priv := privilege.NewManager()
	factory := NewFactory(mockExec, priv, nil)

	t.Run("RHEL 7 returns YUM", func(t *testing.T) {
		dist := &distro.Distribution{
			ID:        "rhel",
			VersionID: "7.9",
			Family:    constants.FamilyRHEL,
		}
		mgr, err := factory.CreateForDistribution(dist)
		require.NoError(t, err)
		assert.Equal(t, "yum", mgr.Name())
		_, ok := mgr.(*yum.Manager)
		assert.True(t, ok, "expected *yum.Manager for RHEL 7")
	})

	t.Run("RHEL 8 returns DNF", func(t *testing.T) {
		dist := &distro.Distribution{
			ID:        "rhel",
			VersionID: "8.0",
			Family:    constants.FamilyRHEL,
		}
		mgr, err := factory.CreateForDistribution(dist)
		require.NoError(t, err)
		assert.Equal(t, "dnf", mgr.Name())
		_, ok := mgr.(*dnf.Manager)
		assert.True(t, ok, "expected *dnf.Manager for RHEL 8")
	})

	t.Run("RHEL 9 returns DNF", func(t *testing.T) {
		dist := &distro.Distribution{
			ID:        "rhel",
			VersionID: "9.0",
			Family:    constants.FamilyRHEL,
		}
		mgr, err := factory.CreateForDistribution(dist)
		require.NoError(t, err)
		assert.Equal(t, "dnf", mgr.Name())
	})
}

func TestFactory_CreateForDistribution_OracleLinux(t *testing.T) {
	mockExec := exec.NewMockExecutor()
	priv := privilege.NewManager()
	factory := NewFactory(mockExec, priv, nil)

	t.Run("Oracle Linux 7 returns YUM", func(t *testing.T) {
		dist := &distro.Distribution{
			ID:        "ol",
			VersionID: "7.9",
			Family:    constants.FamilyRHEL,
		}
		mgr, err := factory.CreateForDistribution(dist)
		require.NoError(t, err)
		assert.Equal(t, "yum", mgr.Name())
	})

	t.Run("Oracle Linux 8 returns DNF", func(t *testing.T) {
		dist := &distro.Distribution{
			ID:        "ol",
			VersionID: "8.0",
			Family:    constants.FamilyRHEL,
		}
		mgr, err := factory.CreateForDistribution(dist)
		require.NoError(t, err)
		assert.Equal(t, "dnf", mgr.Name())
	})
}

func TestFactory_CreateForDistribution_FedoraAlwaysDNF(t *testing.T) {
	mockExec := exec.NewMockExecutor()
	priv := privilege.NewManager()
	factory := NewFactory(mockExec, priv, nil)

	// Fedora 1 (the very first) should still return DNF
	// (Even though it predates DNF, the current Fedora always has DNF)
	versions := []string{"1", "7", "20", "30", "38", "39", "40"}

	for _, version := range versions {
		t.Run("Fedora "+version+" returns DNF", func(t *testing.T) {
			dist := &distro.Distribution{
				ID:        "fedora",
				VersionID: version,
				Family:    constants.FamilyRHEL,
			}
			mgr, err := factory.CreateForDistribution(dist)
			require.NoError(t, err)
			assert.Equal(t, "dnf", mgr.Name())
			_, ok := mgr.(*dnf.Manager)
			assert.True(t, ok, "expected *dnf.Manager for Fedora")
		})
	}
}

func TestFactory_CreateForDistribution_NoVersion(t *testing.T) {
	mockExec := exec.NewMockExecutor()
	priv := privilege.NewManager()
	factory := NewFactory(mockExec, priv, nil)

	t.Run("RHEL family without version returns DNF", func(t *testing.T) {
		dist := &distro.Distribution{
			ID:        "centos",
			VersionID: "", // No version
			Family:    constants.FamilyRHEL,
		}
		mgr, err := factory.CreateForDistribution(dist)
		require.NoError(t, err)
		// Without version info, default to DNF (modern default)
		assert.Equal(t, "dnf", mgr.Name())
	})

	t.Run("RHEL family with unparseable version returns DNF", func(t *testing.T) {
		dist := &distro.Distribution{
			ID:        "centos",
			VersionID: "stream", // Not a number
			Family:    constants.FamilyRHEL,
		}
		mgr, err := factory.CreateForDistribution(dist)
		require.NoError(t, err)
		// Unparseable version defaults to DNF
		assert.Equal(t, "dnf", mgr.Name())
	})
}

func TestFactory_Create_AutoDetect(t *testing.T) {
	mockExec := exec.NewMockExecutor()
	priv := privilege.NewManager()

	tests := []struct {
		name           string
		osRelease      string
		expectedName   string
		expectedFamily constants.DistroFamily
	}{
		{
			name:           "Ubuntu auto-detection",
			osRelease:      ubuntu2404OSRelease,
			expectedName:   "apt",
			expectedFamily: constants.FamilyDebian,
		},
		{
			name:           "Debian auto-detection",
			osRelease:      debian12OSRelease,
			expectedName:   "apt",
			expectedFamily: constants.FamilyDebian,
		},
		{
			name:           "Fedora auto-detection",
			osRelease:      fedora40OSRelease,
			expectedName:   "dnf",
			expectedFamily: constants.FamilyRHEL,
		},
		{
			name:           "RHEL 9 auto-detection",
			osRelease:      rhel9OSRelease,
			expectedName:   "dnf",
			expectedFamily: constants.FamilyRHEL,
		},
		{
			name:           "RHEL 7 auto-detection",
			osRelease:      rhel7OSRelease,
			expectedName:   "yum",
			expectedFamily: constants.FamilyRHEL,
		},
		{
			name:           "CentOS 7 auto-detection",
			osRelease:      centos7OSRelease,
			expectedName:   "yum",
			expectedFamily: constants.FamilyRHEL,
		},
		{
			name:           "CentOS 8 auto-detection",
			osRelease:      centos8OSRelease,
			expectedName:   "dnf",
			expectedFamily: constants.FamilyRHEL,
		},
		{
			name:           "Rocky 9 auto-detection",
			osRelease:      rocky9OSRelease,
			expectedName:   "dnf",
			expectedFamily: constants.FamilyRHEL,
		},
		{
			name:           "AlmaLinux 9 auto-detection",
			osRelease:      almalinux9OSRelease,
			expectedName:   "dnf",
			expectedFamily: constants.FamilyRHEL,
		},
		{
			name:           "Arch auto-detection",
			osRelease:      archOSRelease,
			expectedName:   "pacman",
			expectedFamily: constants.FamilyArch,
		},
		{
			name:           "Manjaro auto-detection",
			osRelease:      manjaroOSRelease,
			expectedName:   "pacman",
			expectedFamily: constants.FamilyArch,
		},
		{
			name:           "EndeavourOS auto-detection",
			osRelease:      endeavourOSRelease,
			expectedName:   "pacman",
			expectedFamily: constants.FamilyArch,
		},
		{
			name:           "openSUSE Leap auto-detection",
			osRelease:      opensuse155OSRelease,
			expectedName:   "zypper",
			expectedFamily: constants.FamilySUSE,
		},
		{
			name:           "openSUSE Tumbleweed auto-detection",
			osRelease:      opensuseTumbleweedOSRelease,
			expectedName:   "zypper",
			expectedFamily: constants.FamilySUSE,
		},
		{
			name:           "Linux Mint auto-detection",
			osRelease:      linuxMintOSRelease,
			expectedName:   "apt",
			expectedFamily: constants.FamilyDebian,
		},
		{
			name:           "Pop!_OS auto-detection",
			osRelease:      popOSRelease,
			expectedName:   "apt",
			expectedFamily: constants.FamilyDebian,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockFS := NewMockFileReader(map[string][]byte{
				"/etc/os-release": []byte(tt.osRelease),
			})
			detector := distro.NewDetector(mockExec, mockFS)
			factory := NewFactory(mockExec, priv, detector)

			mgr, err := factory.Create(context.Background())
			require.NoError(t, err)
			require.NotNil(t, mgr)

			assert.Equal(t, tt.expectedName, mgr.Name())
			assert.Equal(t, tt.expectedFamily, mgr.Family())
		})
	}
}

func TestFactory_Create_NilDetector(t *testing.T) {
	mockExec := exec.NewMockExecutor()
	priv := privilege.NewManager()
	factory := NewFactory(mockExec, priv, nil) // nil detector

	mgr, err := factory.Create(context.Background())
	assert.Error(t, err)
	assert.Nil(t, mgr)
	assert.Contains(t, err.Error(), "detector is nil")
}

func TestFactory_Create_DetectionFails(t *testing.T) {
	mockExec := exec.NewMockExecutor()
	mockExec.SetDefaultResponse(exec.FailureResult(1, "command not found"))
	priv := privilege.NewManager()

	mockFS := NewMockFileReader(nil) // No files
	detector := distro.NewDetector(mockExec, mockFS)
	factory := NewFactory(mockExec, priv, detector)

	mgr, err := factory.Create(context.Background())
	assert.Error(t, err)
	assert.Nil(t, mgr)
	assert.Contains(t, err.Error(), "failed to detect distribution")
}

func TestFactory_Create_UnknownDistribution(t *testing.T) {
	mockExec := exec.NewMockExecutor()
	priv := privilege.NewManager()

	mockFS := NewMockFileReader(map[string][]byte{
		"/etc/os-release": []byte(unknownOSRelease),
	})
	detector := distro.NewDetector(mockExec, mockFS)
	factory := NewFactory(mockExec, priv, detector)

	mgr, err := factory.Create(context.Background())
	assert.Error(t, err)
	assert.Nil(t, mgr)
	assert.ErrorIs(t, err, ErrUnsupportedDistro)
}

func TestFactory_Create_ContextCancellation(t *testing.T) {
	mockExec := exec.NewMockExecutor()
	priv := privilege.NewManager()
	mockFS := NewMockFileReader(nil)
	detector := distro.NewDetector(mockExec, mockFS)
	factory := NewFactory(mockExec, priv, detector)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	mgr, err := factory.Create(ctx)
	assert.Error(t, err)
	assert.Nil(t, mgr)
}

func TestFactory_Create_ContextTimeout(t *testing.T) {
	mockExec := exec.NewMockExecutor()
	priv := privilege.NewManager()
	mockFS := NewMockFileReader(nil)
	detector := distro.NewDetector(mockExec, mockFS)
	factory := NewFactory(mockExec, priv, detector)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()
	time.Sleep(10 * time.Millisecond) // Ensure timeout

	mgr, err := factory.Create(ctx)
	assert.Error(t, err)
	assert.Nil(t, mgr)
}

func TestAvailableManagers(t *testing.T) {
	managers := AvailableManagers()

	assert.Len(t, managers, 5)
	assert.Contains(t, managers, "apt")
	assert.Contains(t, managers, "dnf")
	assert.Contains(t, managers, "yum")
	assert.Contains(t, managers, "pacman")
	assert.Contains(t, managers, "zypper")
}

func TestSupportedFamilies(t *testing.T) {
	families := SupportedFamilies()

	assert.Len(t, families, 4)
	assert.Contains(t, families, constants.FamilyDebian)
	assert.Contains(t, families, constants.FamilyRHEL)
	assert.Contains(t, families, constants.FamilyArch)
	assert.Contains(t, families, constants.FamilySUSE)

	// Should NOT contain FamilyUnknown
	assert.NotContains(t, families, constants.FamilyUnknown)
}

func TestFactory_CreateForDistribution_ManagerIsAvailable(t *testing.T) {
	// This test verifies that all created managers have the IsAvailable method
	// and it returns a boolean (not an error)
	mockExec := exec.NewMockExecutor()
	priv := privilege.NewManager()
	factory := NewFactory(mockExec, priv, nil)

	distributions := []*distro.Distribution{
		{ID: "ubuntu", VersionID: "24.04", Family: constants.FamilyDebian},
		{ID: "fedora", VersionID: "40", Family: constants.FamilyRHEL},
		{ID: "centos", VersionID: "7", Family: constants.FamilyRHEL},
		{ID: "arch", Family: constants.FamilyArch},
		{ID: "opensuse-leap", VersionID: "15.5", Family: constants.FamilySUSE},
	}

	for _, dist := range distributions {
		t.Run(dist.ID, func(t *testing.T) {
			mgr, err := factory.CreateForDistribution(dist)
			require.NoError(t, err)

			// IsAvailable should not panic and return a bool
			available := mgr.IsAvailable()
			// We can't guarantee availability in test environment,
			// just check it returns without panic
			_ = available
		})
	}
}

func TestFactory_EdgeCases(t *testing.T) {
	mockExec := exec.NewMockExecutor()
	priv := privilege.NewManager()
	factory := NewFactory(mockExec, priv, nil)

	t.Run("RHEL with invalid major version", func(t *testing.T) {
		dist := &distro.Distribution{
			ID:        "centos",
			VersionID: "abc", // Invalid version
			Family:    constants.FamilyRHEL,
		}
		mgr, err := factory.CreateForDistribution(dist)
		require.NoError(t, err)
		// Falls back to DNF for unparseable versions
		assert.Equal(t, "dnf", mgr.Name())
	})

	t.Run("RHEL with version starting with non-numeric", func(t *testing.T) {
		dist := &distro.Distribution{
			ID:        "centos",
			VersionID: "v8", // Leading 'v'
			Family:    constants.FamilyRHEL,
		}
		mgr, err := factory.CreateForDistribution(dist)
		require.NoError(t, err)
		// Falls back to DNF for unparseable versions
		assert.Equal(t, "dnf", mgr.Name())
	})

	t.Run("Amazon Linux (uses DNF)", func(t *testing.T) {
		dist := &distro.Distribution{
			ID:        "amzn",
			VersionID: "2023", // Amazon Linux 2023 uses DNF
			Family:    constants.FamilyRHEL,
		}
		mgr, err := factory.CreateForDistribution(dist)
		require.NoError(t, err)
		assert.Equal(t, "dnf", mgr.Name())
	})

	t.Run("Amazon Linux 2 (version 2 < 8, but should use yum)", func(t *testing.T) {
		dist := &distro.Distribution{
			ID:        "amzn",
			VersionID: "2", // Amazon Linux 2
			Family:    constants.FamilyRHEL,
		}
		mgr, err := factory.CreateForDistribution(dist)
		require.NoError(t, err)
		// Version 2 < 8, so it returns YUM
		assert.Equal(t, "yum", mgr.Name())
	})
}

func TestFactory_CreateForFamily_UnknownFamily(t *testing.T) {
	mockExec := exec.NewMockExecutor()
	priv := privilege.NewManager()
	factory := NewFactory(mockExec, priv, nil)

	// Test with an invalid family string cast
	invalidFamily := constants.DistroFamily("invalid")
	mgr, err := factory.CreateForFamily(invalidFamily)

	assert.Error(t, err)
	assert.Nil(t, mgr)
	assert.ErrorIs(t, err, ErrUnsupportedDistro)
}

func TestFactory_LazyInitialization(t *testing.T) {
	// This test verifies that managers are created on-demand (lazy initialization)
	// not all upfront
	mockExec := exec.NewMockExecutor()
	priv := privilege.NewManager()
	factory := NewFactory(mockExec, priv, nil)

	// Factory should not have any managers pre-created
	// Each call to CreateFor* creates a new manager instance

	mgr1, err := factory.CreateForFamily(constants.FamilyDebian)
	require.NoError(t, err)

	mgr2, err := factory.CreateForFamily(constants.FamilyDebian)
	require.NoError(t, err)

	// Each call should create a new instance (not cached)
	// They should be different pointer instances
	assert.NotSame(t, mgr1, mgr2, "managers should be new instances on each call")
}
