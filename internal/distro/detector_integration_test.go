package distro

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tungetti/igor/internal/constants"
	"github.com/tungetti/igor/internal/exec"
)

// ============================================================================
// OS Release Fixtures for Integration Tests
// (Defined here to avoid import cycle with internal/testing)
// ============================================================================

// Integration test fixtures - using realistic os-release content
var (
	integrationUbuntuOSRelease = `NAME="Ubuntu"
VERSION="22.04.3 LTS (Jammy Jellyfish)"
ID=ubuntu
ID_LIKE=debian
PRETTY_NAME="Ubuntu 22.04.3 LTS"
VERSION_ID="22.04"
VERSION_CODENAME=jammy
HOME_URL="https://www.ubuntu.com/"
SUPPORT_URL="https://help.ubuntu.com/"`

	integrationFedoraOSRelease = `NAME="Fedora Linux"
VERSION="39 (Workstation Edition)"
ID=fedora
VERSION_ID=39
PRETTY_NAME="Fedora Linux 39 (Workstation Edition)"
HOME_URL="https://fedoraproject.org/"
SUPPORT_URL="https://ask.fedoraproject.org/"`

	integrationArchOSRelease = `NAME="Arch Linux"
PRETTY_NAME="Arch Linux"
ID=arch
BUILD_ID=rolling
HOME_URL="https://archlinux.org/"
SUPPORT_URL="https://wiki.archlinux.org/"`

	integrationOpenSUSEOSRelease = `NAME="openSUSE Leap"
VERSION="15.5"
ID="opensuse-leap"
ID_LIKE="suse opensuse"
VERSION_ID="15.5"
PRETTY_NAME="openSUSE Leap 15.5"
HOME_URL="https://www.opensuse.org/"
SUPPORT_URL="https://en.opensuse.org/Portal:Support"`

	integrationOpenSUSETumbleweedOSRelease = `NAME="openSUSE Tumbleweed"
ID="opensuse-tumbleweed"
ID_LIKE="opensuse suse"
PRETTY_NAME="openSUSE Tumbleweed"
BUILD_ID=20231215
HOME_URL="https://www.opensuse.org/"`

	integrationDebianOSRelease = `NAME="Debian GNU/Linux"
VERSION="12 (bookworm)"
ID=debian
VERSION_ID="12"
VERSION_CODENAME=bookworm
PRETTY_NAME="Debian GNU/Linux 12 (bookworm)"
HOME_URL="https://www.debian.org/"
SUPPORT_URL="https://www.debian.org/support"`

	integrationRHELOSRelease = `NAME="Red Hat Enterprise Linux"
VERSION="9.3 (Plow)"
ID=rhel
ID_LIKE="fedora"
VERSION_ID="9.3"
PRETTY_NAME="Red Hat Enterprise Linux 9.3 (Plow)"
HOME_URL="https://www.redhat.com/"
SUPPORT_URL="https://access.redhat.com/support"`

	integrationCentOSOSRelease = `NAME="CentOS Stream"
VERSION="9"
ID=centos
ID_LIKE="rhel fedora"
VERSION_ID="9"
PRETTY_NAME="CentOS Stream 9"
HOME_URL="https://centos.org/"
SUPPORT_URL="https://centos.org/help/"`

	integrationRockyOSRelease = `NAME="Rocky Linux"
VERSION="9.3 (Blue Onyx)"
ID=rocky
ID_LIKE="rhel centos fedora"
VERSION_ID="9.3"
PRETTY_NAME="Rocky Linux 9.3 (Blue Onyx)"
HOME_URL="https://rockylinux.org/"
SUPPORT_URL="https://wiki.rockylinux.org/"`

	integrationManjaroOSRelease = `NAME="Manjaro Linux"
PRETTY_NAME="Manjaro Linux"
ID=manjaro
ID_LIKE=arch
BUILD_ID=rolling
HOME_URL="https://manjaro.org/"
SUPPORT_URL="https://wiki.manjaro.org/"`

	integrationPopOSOSRelease = `NAME="Pop!_OS"
VERSION="22.04 LTS"
ID=pop
ID_LIKE="ubuntu debian"
VERSION_ID="22.04"
PRETTY_NAME="Pop!_OS 22.04 LTS"
HOME_URL="https://pop.system76.com/"
SUPPORT_URL="https://support.system76.com/"`
)

// ============================================================================
// Integration Tests - Complete Detection Workflows
// ============================================================================

// TestDetector_IntegrationScenarios tests complete detection workflows
// for each major distribution family.
func TestDetector_IntegrationScenarios(t *testing.T) {
	t.Run("Debian family detection workflow", func(t *testing.T) {
		// Test the complete workflow for Debian-based distributions
		mockFS := NewMockFileReader(map[string][]byte{
			"/etc/os-release": []byte(integrationUbuntuOSRelease),
		})
		mockExec := exec.NewMockExecutor()
		detector := NewDetector(mockExec, mockFS)

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		dist, err := detector.Detect(ctx)
		require.NoError(t, err)
		require.NotNil(t, dist)

		// Verify the complete detection result
		assert.Equal(t, "ubuntu", dist.ID)
		assert.Equal(t, constants.FamilyDebian, dist.Family)
		assert.True(t, dist.IsDebian())
		assert.False(t, dist.IsRHEL())
		assert.False(t, dist.IsArch())
		assert.False(t, dist.IsSUSE())
		assert.Contains(t, dist.IDLike, "debian")
	})

	t.Run("RHEL family detection workflow", func(t *testing.T) {
		mockFS := NewMockFileReader(map[string][]byte{
			"/etc/os-release": []byte(integrationFedoraOSRelease),
		})
		mockExec := exec.NewMockExecutor()
		detector := NewDetector(mockExec, mockFS)

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		dist, err := detector.Detect(ctx)
		require.NoError(t, err)
		require.NotNil(t, dist)

		assert.Equal(t, "fedora", dist.ID)
		assert.Equal(t, constants.FamilyRHEL, dist.Family)
		assert.True(t, dist.IsRHEL())
		assert.False(t, dist.IsDebian())
	})

	t.Run("Arch family detection workflow", func(t *testing.T) {
		mockFS := NewMockFileReader(map[string][]byte{
			"/etc/os-release": []byte(integrationArchOSRelease),
		})
		mockExec := exec.NewMockExecutor()
		detector := NewDetector(mockExec, mockFS)

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		dist, err := detector.Detect(ctx)
		require.NoError(t, err)
		require.NotNil(t, dist)

		assert.Equal(t, "arch", dist.ID)
		assert.Equal(t, constants.FamilyArch, dist.Family)
		assert.True(t, dist.IsArch())
		assert.True(t, dist.IsRolling())
	})

	t.Run("SUSE family detection workflow", func(t *testing.T) {
		mockFS := NewMockFileReader(map[string][]byte{
			"/etc/os-release": []byte(integrationOpenSUSEOSRelease),
		})
		mockExec := exec.NewMockExecutor()
		detector := NewDetector(mockExec, mockFS)

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		dist, err := detector.Detect(ctx)
		require.NoError(t, err)
		require.NotNil(t, dist)

		assert.Equal(t, "opensuse-leap", dist.ID)
		assert.Equal(t, constants.FamilySUSE, dist.Family)
		assert.True(t, dist.IsSUSE())
		assert.False(t, dist.IsRolling())
	})

	t.Run("Fallback chain detection workflow", func(t *testing.T) {
		// Test that fallback detection chain works correctly
		// when primary os-release is missing
		mockFS := NewMockFileReader(map[string][]byte{
			"/usr/lib/os-release": []byte(integrationDebianOSRelease),
		})
		mockExec := exec.NewMockExecutor()
		detector := NewDetector(mockExec, mockFS)

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		dist, err := detector.Detect(ctx)
		require.NoError(t, err)
		require.NotNil(t, dist)

		assert.Equal(t, "debian", dist.ID)
		assert.Equal(t, constants.FamilyDebian, dist.Family)
	})
}

// TestDetector_AllSupportedDistributions tests detection for all supported distributions.
func TestDetector_AllSupportedDistributions(t *testing.T) {
	testCases := []struct {
		name             string
		osReleaseContent string
		expectedFamily   constants.DistroFamily
		expectedID       string
	}{
		{"Ubuntu 22.04", integrationUbuntuOSRelease, constants.FamilyDebian, "ubuntu"},
		{"Fedora 39", integrationFedoraOSRelease, constants.FamilyRHEL, "fedora"},
		{"Arch Linux", integrationArchOSRelease, constants.FamilyArch, "arch"},
		{"openSUSE Leap", integrationOpenSUSEOSRelease, constants.FamilySUSE, "opensuse-leap"},
		{"openSUSE Tumbleweed", integrationOpenSUSETumbleweedOSRelease, constants.FamilySUSE, "opensuse-tumbleweed"},
		{"Debian 12", integrationDebianOSRelease, constants.FamilyDebian, "debian"},
		{"RHEL 9", integrationRHELOSRelease, constants.FamilyRHEL, "rhel"},
		{"CentOS Stream", integrationCentOSOSRelease, constants.FamilyRHEL, "centos"},
		{"Rocky Linux", integrationRockyOSRelease, constants.FamilyRHEL, "rocky"},
		{"Manjaro", integrationManjaroOSRelease, constants.FamilyArch, "manjaro"},
		{"Pop!_OS", integrationPopOSOSRelease, constants.FamilyDebian, "pop"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockFS := NewMockFileReader(map[string][]byte{
				"/etc/os-release": []byte(tc.osReleaseContent),
			})
			mockExec := exec.NewMockExecutor()
			detector := NewDetector(mockExec, mockFS)

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			dist, err := detector.Detect(ctx)
			require.NoError(t, err, "Detection should succeed for %s", tc.name)
			require.NotNil(t, dist, "Distribution should not be nil for %s", tc.name)

			assert.Equal(t, tc.expectedID, dist.ID, "ID mismatch for %s", tc.name)
			assert.Equal(t, tc.expectedFamily, dist.Family, "Family mismatch for %s", tc.name)
		})
	}
}

// TestDetector_FileSystemErrors tests handling of various file system errors.
func TestDetector_FileSystemErrors(t *testing.T) {
	t.Run("missing all release files", func(t *testing.T) {
		mockFS := NewMockFileReader(nil)
		mockExec := exec.NewMockExecutor()
		mockExec.SetDefaultResponse(exec.FailureResult(1, "command not found"))
		detector := NewDetector(mockExec, mockFS)

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		dist, err := detector.Detect(ctx)
		assert.Error(t, err)
		assert.Nil(t, dist)
	})

	t.Run("corrupted os-release with valid fallback", func(t *testing.T) {
		mockFS := NewMockFileReader(map[string][]byte{
			"/etc/os-release": []byte(""),
			"/etc/lsb-release": []byte(`DISTRIB_ID=Ubuntu
DISTRIB_RELEASE=22.04
DISTRIB_CODENAME=jammy
DISTRIB_DESCRIPTION="Ubuntu 22.04 LTS"
`),
		})
		mockExec := exec.NewMockExecutor()
		detector := NewDetector(mockExec, mockFS)

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		dist, err := detector.Detect(ctx)
		require.NoError(t, err)
		require.NotNil(t, dist)
	})

	t.Run("file exists but read fails - uses arch fallback", func(t *testing.T) {
		// When os-release exists but can't be read, the detector
		// falls back to other detection methods. The errorOnReadFileReader
		// reports all files as existing, so it falls back to arch-release.
		mockFS := &errorOnReadFileReader{}
		mockExec := exec.NewMockExecutor()
		mockExec.SetDefaultResponse(exec.FailureResult(1, "command not found"))
		detector := NewDetector(mockExec, mockFS)

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		dist, err := detector.Detect(ctx)
		// This should succeed with arch fallback because errorOnReadFileReader
		// reports all files as existing, including /etc/arch-release
		require.NoError(t, err)
		require.NotNil(t, dist)
		assert.Equal(t, "arch", dist.ID)
	})
}

// TestDetector_MalformedOSRelease tests handling of malformed os-release files.
func TestDetector_MalformedOSRelease(t *testing.T) {
	t.Run("empty file", func(t *testing.T) {
		mockFS := NewMockFileReader(map[string][]byte{
			"/etc/os-release": []byte(""),
		})
		mockExec := exec.NewMockExecutor()
		detector := NewDetector(mockExec, mockFS)

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		dist, err := detector.Detect(ctx)
		require.NoError(t, err)
		require.NotNil(t, dist)
		assert.Equal(t, "", dist.ID)
		assert.Equal(t, constants.FamilyUnknown, dist.Family)
	})

	t.Run("missing required fields", func(t *testing.T) {
		content := `# This is a comment
# Another comment

`
		mockFS := NewMockFileReader(map[string][]byte{
			"/etc/os-release": []byte(content),
		})
		mockExec := exec.NewMockExecutor()
		detector := NewDetector(mockExec, mockFS)

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		dist, err := detector.Detect(ctx)
		require.NoError(t, err)
		assert.Equal(t, "", dist.ID)
	})

	t.Run("invalid format - no equals sign", func(t *testing.T) {
		content := `ID ubuntu
NAME Ubuntu
VERSION 22.04
`
		mockFS := NewMockFileReader(map[string][]byte{
			"/etc/os-release": []byte(content),
		})
		mockExec := exec.NewMockExecutor()
		detector := NewDetector(mockExec, mockFS)

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		dist, err := detector.Detect(ctx)
		require.NoError(t, err)
		assert.Equal(t, "", dist.ID)
	})

	t.Run("binary garbage content", func(t *testing.T) {
		binaryContent := []byte{0x00, 0x01, 0x02, 0xFF, 0xFE, 0xFD}
		mockFS := NewMockFileReader(map[string][]byte{
			"/etc/os-release": binaryContent,
		})
		mockExec := exec.NewMockExecutor()
		detector := NewDetector(mockExec, mockFS)

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		dist, err := detector.Detect(ctx)
		require.NoError(t, err)
		require.NotNil(t, dist)
	})

	t.Run("mixed valid and invalid lines", func(t *testing.T) {
		content := `ID=ubuntu
INVALID_LINE_WITHOUT_EQUALS
NAME="Ubuntu"
=empty_key
VERSION_ID="22.04"
`
		mockFS := NewMockFileReader(map[string][]byte{
			"/etc/os-release": []byte(content),
		})
		mockExec := exec.NewMockExecutor()
		detector := NewDetector(mockExec, mockFS)

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		dist, err := detector.Detect(ctx)
		require.NoError(t, err)
		assert.Equal(t, "ubuntu", dist.ID)
		assert.Equal(t, "Ubuntu", dist.Name)
		assert.Equal(t, "22.04", dist.VersionID)
	})

	t.Run("extremely long lines", func(t *testing.T) {
		longValue := make([]byte, 10000)
		for i := range longValue {
			longValue[i] = 'a'
		}
		content := "ID=ubuntu\nNAME=\"" + string(longValue) + "\"\n"
		mockFS := NewMockFileReader(map[string][]byte{
			"/etc/os-release": []byte(content),
		})
		mockExec := exec.NewMockExecutor()
		detector := NewDetector(mockExec, mockFS)

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		dist, err := detector.Detect(ctx)
		require.NoError(t, err)
		assert.Equal(t, "ubuntu", dist.ID)
		assert.Len(t, dist.Name, 10000)
	})
}

// TestDetector_FallbackDetection tests the fallback detection chain.
func TestDetector_FallbackDetection(t *testing.T) {
	t.Run("fallback to lsb-release when os-release missing", func(t *testing.T) {
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

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		dist, err := detector.Detect(ctx)
		require.NoError(t, err)
		require.NotNil(t, dist)
		assert.Equal(t, "ubuntu", dist.ID)
		assert.Equal(t, "22.04", dist.VersionID)
		assert.Equal(t, "jammy", dist.VersionCodename)
	})

	t.Run("fallback to redhat-release", func(t *testing.T) {
		mockFS := NewMockFileReader(map[string][]byte{
			"/etc/redhat-release": []byte("Rocky Linux release 9.3 (Blue Onyx)"),
		})
		mockExec := exec.NewMockExecutor()
		detector := NewDetector(mockExec, mockFS)

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		dist, err := detector.Detect(ctx)
		require.NoError(t, err)
		require.NotNil(t, dist)
		assert.Equal(t, "rocky", dist.ID)
		assert.Equal(t, constants.FamilyRHEL, dist.Family)
	})

	t.Run("fallback to debian_version", func(t *testing.T) {
		mockFS := NewMockFileReader(map[string][]byte{
			"/etc/debian_version": []byte("12.4"),
		})
		mockExec := exec.NewMockExecutor()
		detector := NewDetector(mockExec, mockFS)

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		dist, err := detector.Detect(ctx)
		require.NoError(t, err)
		require.NotNil(t, dist)
		assert.Equal(t, "debian", dist.ID)
		assert.Equal(t, "12.4", dist.VersionID)
		assert.Equal(t, constants.FamilyDebian, dist.Family)
	})

	t.Run("fallback to arch-release", func(t *testing.T) {
		mockFS := NewMockFileReader(map[string][]byte{
			"/etc/arch-release": []byte(""),
		})
		mockExec := exec.NewMockExecutor()
		detector := NewDetector(mockExec, mockFS)

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		dist, err := detector.Detect(ctx)
		require.NoError(t, err)
		require.NotNil(t, dist)
		assert.Equal(t, "arch", dist.ID)
		assert.Equal(t, constants.FamilyArch, dist.Family)
	})

	t.Run("fallback to SuSE-release", func(t *testing.T) {
		mockFS := NewMockFileReader(map[string][]byte{
			"/etc/SuSE-release": []byte(`openSUSE Leap 15.5
VERSION = 15.5
`),
		})
		mockExec := exec.NewMockExecutor()
		detector := NewDetector(mockExec, mockFS)

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		dist, err := detector.Detect(ctx)
		require.NoError(t, err)
		require.NotNil(t, dist)
		assert.Equal(t, "opensuse-leap", dist.ID)
		assert.Equal(t, constants.FamilySUSE, dist.Family)
	})

	t.Run("fallback to lsb_release command", func(t *testing.T) {
		mockFS := NewMockFileReader(nil)
		mockExec := exec.NewMockExecutor()
		mockExec.SetResponse("lsb_release", exec.SuccessResult(`Distributor ID:	Fedora
Description:	Fedora release 39 (Thirty Nine)
Release:	39
Codename:	n/a
`))
		detector := NewDetector(mockExec, mockFS)

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		dist, err := detector.Detect(ctx)
		require.NoError(t, err)
		require.NotNil(t, dist)
		assert.Equal(t, "fedora", dist.ID)
		assert.Equal(t, "39", dist.VersionID)
	})

	t.Run("priority order - os-release over lsb-release", func(t *testing.T) {
		mockFS := NewMockFileReader(map[string][]byte{
			"/etc/os-release": []byte(integrationFedoraOSRelease),
			"/etc/lsb-release": []byte(`DISTRIB_ID=Ubuntu
DISTRIB_RELEASE=22.04
`),
		})
		mockExec := exec.NewMockExecutor()
		detector := NewDetector(mockExec, mockFS)

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		dist, err := detector.Detect(ctx)
		require.NoError(t, err)
		assert.Equal(t, "fedora", dist.ID)
	})
}

// TestDetector_ContextCancellation tests context cancellation handling.
func TestDetector_ContextCancellation(t *testing.T) {
	t.Run("cancelled context before detection", func(t *testing.T) {
		mockFS := NewMockFileReader(map[string][]byte{
			"/etc/os-release": []byte(integrationUbuntuOSRelease),
		})
		mockExec := exec.NewMockExecutor()
		detector := NewDetector(mockExec, mockFS)

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		dist, err := detector.Detect(ctx)
		assert.Error(t, err)
		assert.Nil(t, dist)
	})

	t.Run("expired context timeout", func(t *testing.T) {
		mockFS := NewMockFileReader(nil)
		mockExec := exec.NewMockExecutor()
		detector := NewDetector(mockExec, mockFS)

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
		defer cancel()
		time.Sleep(10 * time.Millisecond)

		dist, err := detector.Detect(ctx)
		assert.Error(t, err)
		assert.Nil(t, dist)
	})

	t.Run("context with deadline", func(t *testing.T) {
		mockFS := NewMockFileReader(nil)
		mockExec := exec.NewMockExecutor()
		detector := NewDetector(mockExec, mockFS)

		deadline := time.Now().Add(-1 * time.Second)
		ctx, cancel := context.WithDeadline(context.Background(), deadline)
		defer cancel()

		dist, err := detector.Detect(ctx)
		assert.Error(t, err)
		assert.Nil(t, dist)
	})
}

// TestDetector_ConcurrentAccess tests thread safety of the detector.
func TestDetector_ConcurrentAccess(t *testing.T) {
	t.Run("multiple goroutines calling Detect simultaneously", func(t *testing.T) {
		mockFS := NewMockFileReader(map[string][]byte{
			"/etc/os-release": []byte(integrationUbuntuOSRelease),
		})
		mockExec := exec.NewMockExecutor()
		detector := NewDetector(mockExec, mockFS)

		const numGoroutines = 100
		var wg sync.WaitGroup
		wg.Add(numGoroutines)

		errors := make(chan error, numGoroutines)
		results := make(chan *Distribution, numGoroutines)

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		for i := 0; i < numGoroutines; i++ {
			go func() {
				defer wg.Done()
				dist, err := detector.Detect(ctx)
				if err != nil {
					errors <- err
					return
				}
				results <- dist
			}()
		}

		wg.Wait()
		close(errors)
		close(results)

		for err := range errors {
			t.Errorf("Unexpected error in concurrent detection: %v", err)
		}

		var firstDist *Distribution
		for dist := range results {
			if firstDist == nil {
				firstDist = dist
				continue
			}
			assert.Equal(t, firstDist.ID, dist.ID)
			assert.Equal(t, firstDist.Family, dist.Family)
		}
	})

	t.Run("concurrent detection with different mocks", func(t *testing.T) {
		distros := []struct {
			osRelease  string
			expectedID string
		}{
			{integrationUbuntuOSRelease, "ubuntu"},
			{integrationFedoraOSRelease, "fedora"},
			{integrationArchOSRelease, "arch"},
			{integrationDebianOSRelease, "debian"},
		}

		const goroutinesPerDistro = 25
		var wg sync.WaitGroup
		wg.Add(len(distros) * goroutinesPerDistro)

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		for _, d := range distros {
			for i := 0; i < goroutinesPerDistro; i++ {
				go func(osRelease, expectedID string) {
					defer wg.Done()

					mockFS := NewMockFileReader(map[string][]byte{
						"/etc/os-release": []byte(osRelease),
					})
					mockExec := exec.NewMockExecutor()
					detector := NewDetector(mockExec, mockFS)

					dist, err := detector.Detect(ctx)
					assert.NoError(t, err)
					assert.Equal(t, expectedID, dist.ID)
				}(d.osRelease, d.expectedID)
			}
		}

		wg.Wait()
	})
}

// TestDetector_EdgeCases tests additional edge cases.
func TestDetector_EdgeCases(t *testing.T) {
	t.Run("distribution with only ID field", func(t *testing.T) {
		content := `ID=customdistro`
		mockFS := NewMockFileReader(map[string][]byte{
			"/etc/os-release": []byte(content),
		})
		mockExec := exec.NewMockExecutor()
		detector := NewDetector(mockExec, mockFS)

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		dist, err := detector.Detect(ctx)
		require.NoError(t, err)
		assert.Equal(t, "customdistro", dist.ID)
		assert.Equal(t, constants.FamilyUnknown, dist.Family)
	})

	t.Run("distribution with unicode characters", func(t *testing.T) {
		content := `ID=testdistro
NAME="Test Distribution"
PRETTY_NAME="Test Distribution 1.0"
VERSION="1.0"
`
		mockFS := NewMockFileReader(map[string][]byte{
			"/etc/os-release": []byte(content),
		})
		mockExec := exec.NewMockExecutor()
		detector := NewDetector(mockExec, mockFS)

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		dist, err := detector.Detect(ctx)
		require.NoError(t, err)
		assert.Equal(t, "testdistro", dist.ID)
	})

	t.Run("version parsing edge cases", func(t *testing.T) {
		testCases := []struct {
			versionID     string
			expectedMajor string
			expectedMinor string
		}{
			{"24.04", "24", "04"},
			{"9.3.1", "9", "3"},
			{"15-sp5", "15", "sp5"},
			{"rolling", "rolling", ""},
			{"", "", ""},
			{"1", "1", ""},
		}

		for _, tc := range testCases {
			t.Run("version_"+tc.versionID, func(t *testing.T) {
				dist := &Distribution{VersionID: tc.versionID}
				assert.Equal(t, tc.expectedMajor, dist.MajorVersion())
				assert.Equal(t, tc.expectedMinor, dist.MinorVersion())
			})
		}
	})

	t.Run("ID_LIKE with multiple values", func(t *testing.T) {
		content := `ID=customdistro
ID_LIKE="ubuntu debian fedora"
`
		mockFS := NewMockFileReader(map[string][]byte{
			"/etc/os-release": []byte(content),
		})
		mockExec := exec.NewMockExecutor()
		detector := NewDetector(mockExec, mockFS)

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		dist, err := detector.Detect(ctx)
		require.NoError(t, err)
		assert.Equal(t, 3, len(dist.IDLike))
		assert.Equal(t, constants.FamilyDebian, dist.Family)
	})
}

// TestDetector_RollingReleaseDetection tests rolling release identification.
func TestDetector_RollingReleaseDetection(t *testing.T) {
	t.Run("Arch Linux is rolling", func(t *testing.T) {
		mockFS := NewMockFileReader(map[string][]byte{
			"/etc/os-release": []byte(integrationArchOSRelease),
		})
		mockExec := exec.NewMockExecutor()
		detector := NewDetector(mockExec, mockFS)

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		dist, err := detector.Detect(ctx)
		require.NoError(t, err)
		assert.True(t, dist.IsRolling())
	})

	t.Run("Manjaro is rolling", func(t *testing.T) {
		mockFS := NewMockFileReader(map[string][]byte{
			"/etc/os-release": []byte(integrationManjaroOSRelease),
		})
		mockExec := exec.NewMockExecutor()
		detector := NewDetector(mockExec, mockFS)

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		dist, err := detector.Detect(ctx)
		require.NoError(t, err)
		assert.True(t, dist.IsRolling())
	})

	t.Run("openSUSE Tumbleweed is rolling", func(t *testing.T) {
		mockFS := NewMockFileReader(map[string][]byte{
			"/etc/os-release": []byte(integrationOpenSUSETumbleweedOSRelease),
		})
		mockExec := exec.NewMockExecutor()
		detector := NewDetector(mockExec, mockFS)

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		dist, err := detector.Detect(ctx)
		require.NoError(t, err)
		assert.True(t, dist.IsRolling())
	})

	t.Run("Ubuntu is not rolling", func(t *testing.T) {
		mockFS := NewMockFileReader(map[string][]byte{
			"/etc/os-release": []byte(integrationUbuntuOSRelease),
		})
		mockExec := exec.NewMockExecutor()
		detector := NewDetector(mockExec, mockFS)

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		dist, err := detector.Detect(ctx)
		require.NoError(t, err)
		assert.False(t, dist.IsRolling())
	})

	t.Run("openSUSE Leap is not rolling", func(t *testing.T) {
		mockFS := NewMockFileReader(map[string][]byte{
			"/etc/os-release": []byte(integrationOpenSUSEOSRelease),
		})
		mockExec := exec.NewMockExecutor()
		detector := NewDetector(mockExec, mockFS)

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		dist, err := detector.Detect(ctx)
		require.NoError(t, err)
		assert.False(t, dist.IsRolling())
	})
}
