package constants

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestExitCode_Int(t *testing.T) {
	tests := []struct {
		name     string
		code     ExitCode
		expected int
	}{
		{"ExitSuccess", ExitSuccess, 0},
		{"ExitError", ExitError, 1},
		{"ExitPermission", ExitPermission, 2},
		{"ExitValidation", ExitValidation, 3},
		{"ExitInstallation", ExitInstallation, 4},
		{"ExitUserAbort", ExitUserAbort, 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.code.Int())
		})
	}
}

func TestDistroFamily_String(t *testing.T) {
	tests := []struct {
		family   DistroFamily
		expected string
	}{
		{FamilyDebian, "debian"},
		{FamilyRHEL, "rhel"},
		{FamilyArch, "arch"},
		{FamilySUSE, "suse"},
		{FamilyUnknown, "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.family.String())
		})
	}
}

func TestAppMetadata(t *testing.T) {
	// Verify app metadata constants are set correctly
	assert.Equal(t, "igor", AppName)
	assert.Equal(t, "NVIDIA TUI Installer for Linux", AppDescription)
}

func TestTimeouts(t *testing.T) {
	// Verify timeout values are reasonable
	assert.Equal(t, 5*time.Minute, DefaultTimeout)
	assert.Equal(t, 30*time.Second, ShortTimeout)
	assert.Equal(t, 15*time.Minute, LongTimeout)
	assert.Equal(t, 60*time.Second, NetworkTimeout)
	assert.Equal(t, 2*time.Minute, CommandTimeout)

	// Verify timeout ordering makes sense
	assert.Less(t, ShortTimeout, DefaultTimeout)
	assert.Less(t, DefaultTimeout, LongTimeout)
}

func TestFilePaths(t *testing.T) {
	// Verify file path constants are non-empty
	assert.NotEmpty(t, DefaultConfigDir)
	assert.NotEmpty(t, DefaultCacheDir)
	assert.NotEmpty(t, DefaultLogFile)
	assert.NotEmpty(t, ConfigFileName)
}

func TestSystemPaths(t *testing.T) {
	// Verify system path constants start with /
	assert.True(t, OSReleasePath[0] == '/')
	assert.True(t, LSBReleasePath[0] == '/')
	assert.True(t, ModprobeDir[0] == '/')
	assert.True(t, XorgConfDir[0] == '/')
	assert.True(t, SysClassDRM[0] == '/')
	assert.True(t, ProcModules[0] == '/')
}

func TestNvidiaConstants(t *testing.T) {
	// Verify NVIDIA-specific constants are set
	assert.Equal(t, "nouveau", NouveauModuleName)
	assert.Equal(t, "nvidia", NvidiaModuleName)
	assert.Equal(t, "nvidia_drm", NvidiaDRMModule)
	assert.Equal(t, "nvidia_modeset", NvidiaModeset)
	assert.Equal(t, "blacklist-nouveau.conf", NvidiaBlacklistFile)
}

func TestPackageManagerConstants(t *testing.T) {
	// Verify package manager constants are set
	pkgManagers := []string{AptGet, Apt, Dpkg, Dnf, Yum, Rpm, Pacman, Zypper}
	for _, pm := range pkgManagers {
		assert.NotEmpty(t, pm)
	}
}

func TestDistroFamily_Custom(t *testing.T) {
	// Test that DistroFamily can be used as a type
	var family DistroFamily = "custom"
	assert.Equal(t, "custom", family.String())
}

func TestConstantsAreTyped(t *testing.T) {
	// Verify that constants are typed (compile-time check)
	var _ string = AppName
	var _ string = AppDescription
	var _ time.Duration = DefaultTimeout
	var _ time.Duration = ShortTimeout
	var _ time.Duration = LongTimeout
	var _ time.Duration = NetworkTimeout
	var _ time.Duration = CommandTimeout
	var _ string = DefaultConfigDir
	var _ string = DefaultCacheDir
	var _ string = DefaultLogFile
	var _ string = ConfigFileName
	var _ string = OSReleasePath
	var _ string = LSBReleasePath
	var _ string = ModprobeDir
	var _ string = XorgConfDir
	var _ string = SysClassDRM
	var _ string = ProcModules
	var _ string = NouveauModuleName
	var _ string = NvidiaModuleName
	var _ string = NvidiaDRMModule
	var _ string = NvidiaModeset
	var _ string = NvidiaBlacklistFile
	var _ string = AptGet
	var _ string = Apt
	var _ string = Dpkg
	var _ string = Dnf
	var _ string = Yum
	var _ string = Rpm
	var _ string = Pacman
	var _ string = Zypper
	var _ DistroFamily = FamilyDebian
	var _ DistroFamily = FamilyRHEL
	var _ DistroFamily = FamilyArch
	var _ DistroFamily = FamilySUSE
	var _ DistroFamily = FamilyUnknown
	var _ ExitCode = ExitSuccess
	var _ ExitCode = ExitError
	var _ ExitCode = ExitPermission
	var _ ExitCode = ExitValidation
	var _ ExitCode = ExitInstallation
	var _ ExitCode = ExitUserAbort
}
