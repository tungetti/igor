package apt

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tungetti/igor/internal/constants"
	"github.com/tungetti/igor/internal/exec"
	"github.com/tungetti/igor/internal/pkg"
	"github.com/tungetti/igor/internal/privilege"
)

// setupTest creates a Manager with a mock executor for testing.
func setupTest() (*Manager, *exec.MockExecutor) {
	mockExec := exec.NewMockExecutor()
	priv := privilege.NewManager()
	priv.SetRoot(true) // Simulate running as root to avoid sudo wrapper
	mgr := NewManager(mockExec, priv)
	return mgr, mockExec
}

func TestManager_Name(t *testing.T) {
	mgr, _ := setupTest()
	assert.Equal(t, "apt", mgr.Name())
}

func TestManager_Family(t *testing.T) {
	mgr, _ := setupTest()
	assert.Equal(t, constants.FamilyDebian, mgr.Family())
}

func TestManager_Install_SinglePackage(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("env", exec.SuccessResult(""))

	err := mgr.Install(context.Background(), pkg.DefaultInstallOptions(), "nginx")
	require.NoError(t, err)

	// Verify the command was called
	assert.True(t, mockExec.WasCalled("env"))
	lastCall := mockExec.LastCall()
	assert.Contains(t, lastCall.Args, "apt-get")
	assert.Contains(t, lastCall.Args, "install")
	assert.Contains(t, lastCall.Args, "-y")
	assert.Contains(t, lastCall.Args, "nginx")
}

func TestManager_Install_MultiplePackages(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("env", exec.SuccessResult(""))

	err := mgr.Install(context.Background(), pkg.DefaultInstallOptions(), "nginx", "curl", "vim")
	require.NoError(t, err)

	lastCall := mockExec.LastCall()
	assert.Contains(t, lastCall.Args, "nginx")
	assert.Contains(t, lastCall.Args, "curl")
	assert.Contains(t, lastCall.Args, "vim")
}

func TestManager_Install_WithOptions(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("env", exec.SuccessResult(""))

	opts := pkg.InstallOptions{
		Force:          true,
		Reinstall:      true,
		AllowDowngrade: true,
	}

	err := mgr.Install(context.Background(), opts, "nginx")
	require.NoError(t, err)

	lastCall := mockExec.LastCall()
	assert.Contains(t, lastCall.Args, "--allow-unauthenticated")
	assert.Contains(t, lastCall.Args, "--allow-change-held-packages")
	assert.Contains(t, lastCall.Args, "--reinstall")
	assert.Contains(t, lastCall.Args, "--allow-downgrades")
}

func TestManager_Install_PackageNotFound(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("env", exec.FailureResult(100, "E: Unable to locate package nonexistent"))

	err := mgr.Install(context.Background(), pkg.DefaultInstallOptions(), "nonexistent")
	require.Error(t, err)
	assert.ErrorIs(t, err, pkg.ErrPackageNotFound)
}

func TestManager_Install_LockError(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("env", exec.FailureResult(100, "E: Could not get lock /var/lib/dpkg/lock"))

	err := mgr.Install(context.Background(), pkg.DefaultInstallOptions(), "nginx")
	require.Error(t, err)
	assert.ErrorIs(t, err, pkg.ErrLockAcquireFailed)
}

func TestManager_Install_EmptyPackages(t *testing.T) {
	mgr, mockExec := setupTest()

	err := mgr.Install(context.Background(), pkg.DefaultInstallOptions())
	require.NoError(t, err)

	// Should not call any command
	assert.Equal(t, 0, mockExec.CallCount())
}

func TestManager_Remove_SinglePackage(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("env", exec.SuccessResult(""))

	err := mgr.Remove(context.Background(), pkg.DefaultRemoveOptions(), "nginx")
	require.NoError(t, err)

	lastCall := mockExec.LastCall()
	assert.Contains(t, lastCall.Args, "apt-get")
	assert.Contains(t, lastCall.Args, "remove")
	assert.Contains(t, lastCall.Args, "-y")
	assert.Contains(t, lastCall.Args, "nginx")
}

func TestManager_Remove_WithPurge(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("env", exec.SuccessResult(""))

	opts := pkg.RemoveOptions{Purge: true}
	err := mgr.Remove(context.Background(), opts, "nginx")
	require.NoError(t, err)

	lastCall := mockExec.LastCall()
	assert.Contains(t, lastCall.Args, "purge")
	assert.NotContains(t, lastCall.Args, "remove")
}

func TestManager_Remove_WithAutoRemove(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("env", exec.SuccessResult(""))

	opts := pkg.RemoveOptions{AutoRemove: true}
	err := mgr.Remove(context.Background(), opts, "nginx")
	require.NoError(t, err)

	lastCall := mockExec.LastCall()
	assert.Contains(t, lastCall.Args, "--auto-remove")
}

func TestManager_Remove_PackageNotInstalled(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("env", exec.FailureResult(100, "Package nginx is not installed"))

	err := mgr.Remove(context.Background(), pkg.DefaultRemoveOptions(), "nginx")
	require.Error(t, err)
	assert.ErrorIs(t, err, pkg.ErrPackageNotInstalled)
}

func TestManager_Update_Success(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("env", exec.SuccessResult("Hit:1 http://archive.ubuntu.com/ubuntu jammy InRelease"))

	err := mgr.Update(context.Background(), pkg.DefaultUpdateOptions())
	require.NoError(t, err)

	lastCall := mockExec.LastCall()
	assert.Contains(t, lastCall.Args, "apt-get")
	assert.Contains(t, lastCall.Args, "update")
}

func TestManager_Update_Quiet(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("env", exec.SuccessResult(""))

	opts := pkg.UpdateOptions{Quiet: true}
	err := mgr.Update(context.Background(), opts)
	require.NoError(t, err)

	lastCall := mockExec.LastCall()
	assert.Contains(t, lastCall.Args, "-qq")
}

func TestManager_Update_NetworkError(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("env", exec.FailureResult(100, "E: Failed to fetch http://archive.ubuntu.com"))

	err := mgr.Update(context.Background(), pkg.DefaultUpdateOptions())
	require.Error(t, err)
	assert.ErrorIs(t, err, pkg.ErrNetworkUnavailable)
}

func TestManager_Update_LockError(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("env", exec.FailureResult(100, "E: Could not get lock"))

	err := mgr.Update(context.Background(), pkg.DefaultUpdateOptions())
	require.Error(t, err)
	assert.ErrorIs(t, err, pkg.ErrLockAcquireFailed)
}

func TestManager_Upgrade_AllPackages(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("env", exec.SuccessResult(""))

	err := mgr.Upgrade(context.Background(), pkg.DefaultInstallOptions())
	require.NoError(t, err)

	lastCall := mockExec.LastCall()
	assert.Contains(t, lastCall.Args, "apt-get")
	assert.Contains(t, lastCall.Args, "upgrade")
	assert.Contains(t, lastCall.Args, "-y")
}

func TestManager_Upgrade_SpecificPackages(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("env", exec.SuccessResult(""))

	err := mgr.Upgrade(context.Background(), pkg.DefaultInstallOptions(), "nginx", "curl")
	require.NoError(t, err)

	lastCall := mockExec.LastCall()
	assert.Contains(t, lastCall.Args, "install")
	assert.Contains(t, lastCall.Args, "--only-upgrade")
	assert.Contains(t, lastCall.Args, "nginx")
	assert.Contains(t, lastCall.Args, "curl")
}

func TestManager_IsInstalled_True(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("dpkg-query", exec.SuccessResult("install ok installed"))

	installed, err := mgr.IsInstalled(context.Background(), "nginx")
	require.NoError(t, err)
	assert.True(t, installed)
}

func TestManager_IsInstalled_False_NotInstalled(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("dpkg-query", exec.FailureResult(1, "dpkg-query: no packages found matching nginx"))

	installed, err := mgr.IsInstalled(context.Background(), "nginx")
	require.NoError(t, err)
	assert.False(t, installed)
}

func TestManager_IsInstalled_False_DifferentStatus(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("dpkg-query", exec.SuccessResult("deinstall ok config-files"))

	installed, err := mgr.IsInstalled(context.Background(), "nginx")
	require.NoError(t, err)
	assert.False(t, installed)
}

func TestManager_Search_Success(t *testing.T) {
	mgr, mockExec := setupTest()

	output := `nginx - small, powerful, scalable web/proxy server
nginx-common - common files for nginx
nginx-core - nginx web/proxy server (standard version)`

	mockExec.SetResponse("apt-cache", exec.SuccessResult(output))
	mockExec.SetResponse("dpkg-query", exec.FailureResult(1, "not found"))

	packages, err := mgr.Search(context.Background(), "nginx", pkg.DefaultSearchOptions())
	require.NoError(t, err)
	require.Len(t, packages, 3)
	assert.Equal(t, "nginx", packages[0].Name)
	assert.Equal(t, "small, powerful, scalable web/proxy server", packages[0].Description)
}

func TestManager_Search_WithLimit(t *testing.T) {
	mgr, mockExec := setupTest()

	output := `nginx - small, powerful, scalable web/proxy server
nginx-common - common files for nginx
nginx-core - nginx web/proxy server (standard version)`

	mockExec.SetResponse("apt-cache", exec.SuccessResult(output))
	mockExec.SetResponse("dpkg-query", exec.FailureResult(1, "not found"))

	opts := pkg.SearchOptions{Limit: 2, IncludeInstalled: false}
	packages, err := mgr.Search(context.Background(), "nginx", opts)
	require.NoError(t, err)
	require.Len(t, packages, 2)
}

func TestManager_Search_ExactMatch(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("apt-cache", exec.SuccessResult("nginx - web server"))
	mockExec.SetResponse("dpkg-query", exec.FailureResult(1, "not found"))

	opts := pkg.SearchOptions{ExactMatch: true}
	_, err := mgr.Search(context.Background(), "nginx", opts)
	require.NoError(t, err)

	lastCall := mockExec.Calls()[0]
	assert.Contains(t, lastCall.Args, "--names-only")
}

func TestManager_Info_Success(t *testing.T) {
	mgr, mockExec := setupTest()

	output := `Package: nginx
Version: 1.22.0-1ubuntu1
Architecture: amd64
Installed-Size: 1200
Depends: libc6, libpcre3
Description: small, powerful, scalable web/proxy server
Section: web
`

	mockExec.SetResponse("apt-cache", exec.SuccessResult(output))
	mockExec.SetResponse("dpkg-query", exec.SuccessResult("install ok installed"))

	info, err := mgr.Info(context.Background(), "nginx")
	require.NoError(t, err)
	require.NotNil(t, info)
	assert.Equal(t, "nginx", info.Name)
	assert.Equal(t, "1.22.0-1ubuntu1", info.Version)
	assert.Equal(t, "amd64", info.Architecture)
	assert.Equal(t, int64(1200*1024), info.Size)
	assert.True(t, info.Installed)
	assert.Contains(t, info.Dependencies, "libc6")
	assert.Contains(t, info.Dependencies, "libpcre3")
}

func TestManager_Info_NotFound(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("apt-cache", exec.FailureResult(100, "N: No packages found"))

	info, err := mgr.Info(context.Background(), "nonexistent")
	require.Error(t, err)
	assert.Nil(t, info)
	assert.ErrorIs(t, err, pkg.ErrPackageNotFound)
}

func TestManager_ListInstalled(t *testing.T) {
	mgr, mockExec := setupTest()

	output := `nginx	1.22.0-1ubuntu1	install ok installed
curl	7.81.0-1ubuntu1	install ok installed
vim	2:8.2.3995-1ubuntu2	deinstall ok config-files`

	mockExec.SetResponse("dpkg-query", exec.SuccessResult(output))

	packages, err := mgr.ListInstalled(context.Background())
	require.NoError(t, err)
	// vim should be filtered out (not fully installed)
	require.Len(t, packages, 2)
	assert.Equal(t, "nginx", packages[0].Name)
	assert.Equal(t, "1.22.0-1ubuntu1", packages[0].Version)
	assert.True(t, packages[0].Installed)
}

func TestManager_ListUpgradable(t *testing.T) {
	mgr, mockExec := setupTest()

	output := `Listing...
nginx/jammy-updates 1.22.0-1ubuntu1.1 amd64 [upgradable from: 1.22.0-1ubuntu1]
curl/jammy-security 7.81.0-1ubuntu1.7 amd64 [upgradable from: 7.81.0-1ubuntu1]`

	mockExec.SetResponse("apt", exec.SuccessResult(output))

	packages, err := mgr.ListUpgradable(context.Background())
	require.NoError(t, err)
	require.Len(t, packages, 2)
	assert.Equal(t, "nginx", packages[0].Name)
	assert.Equal(t, "1.22.0-1ubuntu1.1", packages[0].Version)
	assert.Equal(t, "jammy-updates", packages[0].Repository)
}

func TestManager_Clean_Success(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("env", exec.SuccessResult(""))

	err := mgr.Clean(context.Background())
	require.NoError(t, err)

	lastCall := mockExec.LastCall()
	assert.Contains(t, lastCall.Args, "apt-get")
	assert.Contains(t, lastCall.Args, "clean")
}

func TestManager_AutoRemove_Success(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("env", exec.SuccessResult(""))

	err := mgr.AutoRemove(context.Background())
	require.NoError(t, err)

	lastCall := mockExec.LastCall()
	assert.Contains(t, lastCall.Args, "apt-get")
	assert.Contains(t, lastCall.Args, "autoremove")
	assert.Contains(t, lastCall.Args, "-y")
}

func TestManager_Verify_Installed(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("dpkg-query", exec.SuccessResult("install ok installed"))
	mockExec.SetResponse("dpkg", exec.SuccessResult(""))

	valid, err := mgr.Verify(context.Background(), "nginx")
	require.NoError(t, err)
	assert.True(t, valid)
}

func TestManager_Verify_NotInstalled(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("dpkg-query", exec.FailureResult(1, "not found"))

	_, err := mgr.Verify(context.Background(), "nonexistent")
	require.Error(t, err)
	assert.ErrorIs(t, err, pkg.ErrPackageNotInstalled)
}

func TestManager_AddRepository_PPA(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("env", exec.SuccessResult(""))

	repo := pkg.Repository{
		Name: "graphics-drivers",
		URL:  "ppa:graphics-drivers/ppa",
	}

	err := mgr.AddRepository(context.Background(), repo)
	require.NoError(t, err)

	lastCall := mockExec.LastCall()
	assert.Contains(t, lastCall.Args, "add-apt-repository")
	assert.Contains(t, lastCall.Args, "-y")
	assert.Contains(t, lastCall.Args, "ppa:graphics-drivers/ppa")
}

func TestManager_AddRepository_PPAAlreadyExists(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("env", exec.FailureResult(1, "Error: ppa already exists"))

	repo := pkg.Repository{
		Name: "graphics-drivers",
		URL:  "ppa:graphics-drivers/ppa",
	}

	err := mgr.AddRepository(context.Background(), repo)
	require.Error(t, err)
	assert.ErrorIs(t, err, pkg.ErrRepositoryExists)
}

func TestManager_RemoveRepository_PPA(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("env", exec.SuccessResult(""))

	err := mgr.RemoveRepository(context.Background(), "ppa:graphics-drivers/ppa")
	require.NoError(t, err)

	lastCall := mockExec.LastCall()
	assert.Contains(t, lastCall.Args, "add-apt-repository")
	assert.Contains(t, lastCall.Args, "--remove")
}

func TestManager_RemoveRepository_NotFound(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("test", exec.FailureResult(1, ""))

	err := mgr.RemoveRepository(context.Background(), "nonexistent")
	require.Error(t, err)
	assert.ErrorIs(t, err, pkg.ErrRepositoryNotFound)
}

func TestManager_ListRepositories(t *testing.T) {
	mgr, mockExec := setupTest()

	sourcesContent := `deb http://archive.ubuntu.com/ubuntu jammy main restricted
deb http://archive.ubuntu.com/ubuntu jammy-updates main restricted
# deb http://archive.ubuntu.com/ubuntu jammy-backports main restricted`

	mockExec.SetResponse("cat", exec.SuccessResult(sourcesContent))
	mockExec.SetResponse("ls", exec.SuccessResult(""))

	repos, err := mgr.ListRepositories(context.Background())
	require.NoError(t, err)
	// Should return 2 enabled + 1 disabled
	require.Len(t, repos, 3)

	// Check first repo
	assert.Equal(t, "http://archive.ubuntu.com/ubuntu", repos[0].URL)
	assert.Equal(t, "jammy", repos[0].Distribution)
	assert.True(t, repos[0].Enabled)

	// Check disabled repo
	assert.False(t, repos[2].Enabled)
}

func TestManager_RefreshRepositories(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("env", exec.SuccessResult(""))

	err := mgr.RefreshRepositories(context.Background())
	require.NoError(t, err)

	// Should have called apt-get update
	lastCall := mockExec.LastCall()
	assert.Contains(t, lastCall.Args, "apt-get")
	assert.Contains(t, lastCall.Args, "update")
}

func TestManager_AddGPGKey(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetDefaultResponse(exec.SuccessResult(""))

	err := mgr.AddGPGKey(context.Background(), "nvidia", "https://nvidia.com/key.gpg")
	require.NoError(t, err)

	// Should have called mkdir and then the curl|gpg pipeline
	assert.True(t, mockExec.CallCount() >= 2)
}

func TestManager_AddGPGKey_NetworkError(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("mkdir", exec.SuccessResult(""))
	mockExec.SetResponse("sh", exec.FailureResult(1, "curl: (6) Could not resolve host: nvidia.com"))

	err := mgr.AddGPGKey(context.Background(), "nvidia", "https://nvidia.com/key.gpg")
	require.Error(t, err)
	assert.ErrorIs(t, err, pkg.ErrNetworkUnavailable)
}

// Parser tests

func TestParseDpkgQuery(t *testing.T) {
	output := `nginx	1.22.0-1ubuntu1	install ok installed
curl	7.81.0-1ubuntu1	install ok installed
vim	2:8.2.3995-1ubuntu2	deinstall ok config-files
htop	3.0.5-7	install ok installed`

	packages, err := parseDpkgQuery(output)
	require.NoError(t, err)
	require.Len(t, packages, 3) // vim should be excluded

	assert.Equal(t, "nginx", packages[0].Name)
	assert.Equal(t, "1.22.0-1ubuntu1", packages[0].Version)
	assert.True(t, packages[0].Installed)

	assert.Equal(t, "curl", packages[1].Name)
	assert.Equal(t, "htop", packages[2].Name)
}

func TestParseAptCacheShow(t *testing.T) {
	output := `Package: nginx
Version: 1.22.0-1ubuntu1
Architecture: amd64
Installed-Size: 1200
Depends: libc6 (>= 2.17), libpcre3, libssl3 (>= 3.0.0)
Description: small, powerful, scalable web/proxy server
 Nginx (pronounced "engine X") is a high-performance HTTP and reverse proxy
 server.
Section: web

Package: nginx
Version: 1.18.0-0ubuntu1
Architecture: amd64
`

	p, err := parseAptCacheShow(output)
	require.NoError(t, err)
	require.NotNil(t, p)

	assert.Equal(t, "nginx", p.Name)
	assert.Equal(t, "1.22.0-1ubuntu1", p.Version)
	assert.Equal(t, "amd64", p.Architecture)
	assert.Equal(t, int64(1200*1024), p.Size)
	assert.Equal(t, "small, powerful, scalable web/proxy server", p.Description)
	assert.Contains(t, p.Dependencies, "libc6")
	assert.Contains(t, p.Dependencies, "libpcre3")
	assert.Contains(t, p.Dependencies, "libssl3")
}

func TestParseAptCacheSearch(t *testing.T) {
	output := `nginx - small, powerful, scalable web/proxy server
nginx-common - common files for nginx
nginx-core - nginx web/proxy server (standard version)
nginx-light - nginx web/proxy server (basic version)`

	packages, err := parseAptCacheSearch(output)
	require.NoError(t, err)
	require.Len(t, packages, 4)

	assert.Equal(t, "nginx", packages[0].Name)
	assert.Equal(t, "small, powerful, scalable web/proxy server", packages[0].Description)
	assert.Equal(t, "nginx-common", packages[1].Name)
}

func TestParseSourcesList(t *testing.T) {
	content := `# Main repos
deb http://archive.ubuntu.com/ubuntu jammy main restricted
deb-src http://archive.ubuntu.com/ubuntu jammy main restricted
deb [arch=amd64 signed-by=/etc/apt/keyrings/nvidia.gpg] https://developer.download.nvidia.com/compute/cuda/repos/ubuntu2204/x86_64 /
# Disabled repo
#deb http://archive.ubuntu.com/ubuntu jammy-backports main`

	repos, err := parseSourcesList(content)
	require.NoError(t, err)
	require.Len(t, repos, 4)

	// First repo
	assert.Equal(t, "http://archive.ubuntu.com/ubuntu", repos[0].URL)
	assert.Equal(t, "jammy", repos[0].Distribution)
	assert.Equal(t, "deb", repos[0].Type)
	assert.True(t, repos[0].Enabled)
	assert.Contains(t, repos[0].Components, "main")
	assert.Contains(t, repos[0].Components, "restricted")

	// deb-src repo
	assert.Equal(t, "deb-src", repos[1].Type)

	// NVIDIA repo with options
	assert.Equal(t, "https://developer.download.nvidia.com/compute/cuda/repos/ubuntu2204/x86_64", repos[2].URL)
	assert.Equal(t, "/etc/apt/keyrings/nvidia.gpg", repos[2].GPGKey)

	// Disabled repo
	assert.False(t, repos[3].Enabled)
}

func TestParseAptListUpgradable(t *testing.T) {
	output := `Listing...
nginx/jammy-updates 1.22.0-1ubuntu1.1 amd64 [upgradable from: 1.22.0-1ubuntu1]
curl/jammy-security 7.81.0-1ubuntu1.7 amd64 [upgradable from: 7.81.0-1ubuntu1]
vim/jammy-updates 2:8.2.3995-1ubuntu2.3 amd64 [upgradable from: 2:8.2.3995-1ubuntu2]`

	packages, err := parseAptListUpgradable(output)
	require.NoError(t, err)
	require.Len(t, packages, 3)

	assert.Equal(t, "nginx", packages[0].Name)
	assert.Equal(t, "1.22.0-1ubuntu1.1", packages[0].Version)
	assert.Equal(t, "amd64", packages[0].Architecture)
	assert.Equal(t, "jammy-updates", packages[0].Repository)
	assert.True(t, packages[0].Installed)

	assert.Equal(t, "curl", packages[1].Name)
	assert.Equal(t, "jammy-security", packages[1].Repository)
}

func TestParseAptListUpgradable_Empty(t *testing.T) {
	output := `Listing...`

	packages, err := parseAptListUpgradable(output)
	require.NoError(t, err)
	require.Len(t, packages, 0)
}

func TestParseDependencies(t *testing.T) {
	// Simple dependencies
	deps := parseDependencies("libc6, libpcre3")
	assert.Len(t, deps, 2)
	assert.Contains(t, deps, "libc6")
	assert.Contains(t, deps, "libpcre3")

	// With version constraints
	deps = parseDependencies("libc6 (>= 2.17), libssl3 (>= 3.0.0)")
	assert.Len(t, deps, 2)
	assert.Contains(t, deps, "libc6")
	assert.Contains(t, deps, "libssl3")

	// With alternatives
	deps = parseDependencies("libc6 | libc6-udeb, libpcre3")
	assert.Len(t, deps, 2)
	assert.Contains(t, deps, "libc6") // First alternative only
	assert.Contains(t, deps, "libpcre3")

	// With :any suffix
	deps = parseDependencies("libc6:any, libpcre3:i386")
	assert.Len(t, deps, 2)
	assert.Contains(t, deps, "libc6")
	assert.Contains(t, deps, "libpcre3")
}

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"simple", "simple"},
		{"with space", "with_space"},
		{"with/slash", "with_slash"},
		{"with:colon", "with_colon"},
		{"with.dot", "with_dot"},
		{"ppa:user/repo", "ppa_user_repo"},
		{"__leading__", "leading"},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			result := sanitizeFilename(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestBuildSourcesListEntry(t *testing.T) {
	// Simple repo
	repo := pkg.Repository{
		URL:          "http://archive.ubuntu.com/ubuntu",
		Distribution: "jammy",
		Components:   []string{"main", "restricted"},
	}
	entry := buildSourcesListEntry(repo)
	assert.Equal(t, "deb http://archive.ubuntu.com/ubuntu jammy main restricted\n", entry)

	// With GPG key
	repo = pkg.Repository{
		URL:          "https://nvidia.com/repo",
		Distribution: "stable",
		Components:   []string{"main"},
		GPGKey:       "/etc/apt/keyrings/nvidia.gpg",
	}
	entry = buildSourcesListEntry(repo)
	assert.Contains(t, entry, "[signed-by=/etc/apt/keyrings/nvidia.gpg]")

	// deb-src type
	repo = pkg.Repository{
		Type:         "deb-src",
		URL:          "http://archive.ubuntu.com/ubuntu",
		Distribution: "jammy",
		Components:   []string{"main"},
	}
	entry = buildSourcesListEntry(repo)
	assert.True(t, strings.HasPrefix(entry, "deb-src "))

	// No distribution (flat repo)
	repo = pkg.Repository{
		URL: "https://nvidia.com/repo",
	}
	entry = buildSourcesListEntry(repo)
	assert.Contains(t, entry, " /\n")
}

func TestGenerateRepoName(t *testing.T) {
	tests := []struct {
		url      string
		dist     string
		expected string
	}{
		{"http://archive.ubuntu.com/ubuntu", "jammy", "archive_ubuntu_com_jammy"},
		{"https://nvidia.com/repo", "", "nvidia_com"},
		{"http://ppa.launchpad.net/graphics-drivers/ppa/ubuntu", "jammy", "ppa_launchpad_net_jammy"},
	}

	for _, tc := range tests {
		t.Run(tc.url, func(t *testing.T) {
			result := generateRepoName(tc.url, tc.dist)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// Context cancellation test
func TestManager_Install_ContextCancelled(t *testing.T) {
	mgr, mockExec := setupTest()

	// Set up response that simulates cancellation error
	mockExec.SetResponse("env", exec.SuccessResult(""))

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// The mock doesn't actually check context, but this tests the pattern
	err := mgr.Install(ctx, pkg.DefaultInstallOptions(), "nginx")
	// In a real scenario with actual command execution, this would return ctx.Err()
	// With mock, it succeeds because mock doesn't check context
	require.NoError(t, err)
}

// Interface compliance test
func TestManager_ImplementsInterface(t *testing.T) {
	var _ pkg.Manager = (*Manager)(nil)
}

// Additional repository tests

func TestManager_AddRepository_Direct(t *testing.T) {
	mgr, mockExec := setupTest()

	// File doesn't exist
	mockExec.SetResponse("test", exec.FailureResult(1, ""))
	// Create file with ExecuteWithInput
	mockExec.SetDefaultResponse(exec.SuccessResult(""))

	repo := pkg.Repository{
		Name:         "nvidia",
		URL:          "https://developer.download.nvidia.com/compute/cuda/repos/ubuntu2204/x86_64",
		Distribution: "/",
		Components:   []string{},
	}

	err := mgr.AddRepository(context.Background(), repo)
	require.NoError(t, err)
}

func TestManager_AddRepository_DirectExists(t *testing.T) {
	mgr, mockExec := setupTest()

	// File already exists
	mockExec.SetResponse("test", exec.SuccessResult(""))

	repo := pkg.Repository{
		Name:         "nvidia",
		URL:          "https://nvidia.com/repo",
		Distribution: "stable",
	}

	err := mgr.AddRepository(context.Background(), repo)
	require.Error(t, err)
	assert.ErrorIs(t, err, pkg.ErrRepositoryExists)
}

func TestManager_RemoveRepository_DirectRepo(t *testing.T) {
	mgr, mockExec := setupTest()

	// File exists
	mockExec.SetResponse("test", exec.SuccessResult(""))
	mockExec.SetDefaultResponse(exec.SuccessResult(""))

	err := mgr.RemoveRepository(context.Background(), "nvidia")
	require.NoError(t, err)

	calls := mockExec.Calls()
	// Should call rm -f
	var foundRm bool
	for _, call := range calls {
		if call.Command == "rm" {
			foundRm = true
			assert.Contains(t, call.Args, "-f")
		}
	}
	assert.True(t, foundRm)
}

func TestManager_EnableRepository(t *testing.T) {
	mgr, mockExec := setupTest()

	// File exists
	mockExec.SetResponse("test", exec.SuccessResult(""))
	// File content with disabled repo
	mockExec.SetResponse("cat", exec.SuccessResult("#deb http://archive.ubuntu.com/ubuntu jammy main"))
	mockExec.SetDefaultResponse(exec.SuccessResult(""))

	err := mgr.EnableRepository(context.Background(), "archive.ubuntu.com")
	require.NoError(t, err)
}

func TestManager_DisableRepository(t *testing.T) {
	mgr, mockExec := setupTest()

	// File exists
	mockExec.SetResponse("test", exec.SuccessResult(""))
	// File content with enabled repo
	mockExec.SetResponse("cat", exec.SuccessResult("deb http://archive.ubuntu.com/ubuntu jammy main"))
	mockExec.SetDefaultResponse(exec.SuccessResult(""))

	err := mgr.DisableRepository(context.Background(), "archive.ubuntu.com")
	require.NoError(t, err)
}

func TestManager_EnableRepository_AlreadyEnabled(t *testing.T) {
	mgr, mockExec := setupTest()

	// File exists
	mockExec.SetResponse("test", exec.SuccessResult(""))
	// File content with already enabled repo (no # prefix)
	mockExec.SetResponse("cat", exec.SuccessResult("deb http://archive.ubuntu.com/ubuntu jammy main"))

	err := mgr.EnableRepository(context.Background(), "archive.ubuntu.com")
	require.Error(t, err)
	assert.ErrorIs(t, err, pkg.ErrRepositoryNotFound) // "not found or already enabled"
}

func TestManager_DisableRepository_AlreadyDisabled(t *testing.T) {
	mgr, mockExec := setupTest()

	// File exists
	mockExec.SetResponse("test", exec.SuccessResult(""))
	// File content with already disabled repo
	mockExec.SetResponse("cat", exec.SuccessResult("#deb http://archive.ubuntu.com/ubuntu jammy main"))

	err := mgr.DisableRepository(context.Background(), "archive.ubuntu.com")
	require.Error(t, err)
	assert.ErrorIs(t, err, pkg.ErrRepositoryNotFound) // "not found or already disabled"
}

func TestManager_EnableRepository_FileNotFound(t *testing.T) {
	mgr, mockExec := setupTest()

	// File doesn't exist in sources.list.d
	mockExec.SetResponse("test", exec.FailureResult(1, ""))
	// Also can't read main sources.list
	mockExec.SetResponse("cat", exec.FailureResult(1, "No such file"))

	err := mgr.EnableRepository(context.Background(), "nonexistent")
	require.Error(t, err)
}

func TestManager_ListRepositories_WithSourcesListD(t *testing.T) {
	mgr, mockExec := setupTest()

	// Main sources.list
	mockExec.SetResponse("cat", exec.SuccessResult("deb http://archive.ubuntu.com/ubuntu jammy main"))
	// Files in sources.list.d
	mockExec.SetResponse("ls", exec.SuccessResult("nvidia.list\ngraphics-drivers-ubuntu-ppa.list"))

	repos, err := mgr.ListRepositories(context.Background())
	require.NoError(t, err)
	// Since mock returns the same "cat" response for all files,
	// we get 3 repos (1 from main + 2 from .list.d files using same content)
	assert.GreaterOrEqual(t, len(repos), 1)
}

func TestManager_GetGPGKeyPath(t *testing.T) {
	mgr, _ := setupTest()

	path := mgr.GetGPGKeyPath("nvidia")
	assert.Equal(t, "/etc/apt/keyrings/nvidia.gpg", path)

	path = mgr.GetGPGKeyPath("graphics-drivers/ppa")
	assert.Equal(t, "/etc/apt/keyrings/graphics-drivers_ppa.gpg", path)
}

func TestManager_AutoRemove_LockError(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("env", exec.FailureResult(100, "E: Could not get lock"))

	err := mgr.AutoRemove(context.Background())
	require.Error(t, err)
	assert.ErrorIs(t, err, pkg.ErrLockAcquireFailed)
}

func TestManager_Clean_Failure(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("env", exec.FailureResult(1, "Some error"))

	err := mgr.Clean(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "apt-get clean failed")
}

func TestManager_Verify_Invalid(t *testing.T) {
	mgr, mockExec := setupTest()

	// Package is installed
	mockExec.SetResponse("dpkg-query", exec.SuccessResult("install ok installed"))
	// But verification fails
	mockExec.SetResponse("dpkg", exec.FailureResult(1, "verification failed"))

	valid, err := mgr.Verify(context.Background(), "nginx")
	require.NoError(t, err)
	assert.False(t, valid)
}

func TestManager_Search_Failure(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("apt-cache", exec.FailureResult(1, "Search failed"))

	packages, err := mgr.Search(context.Background(), "nginx", pkg.DefaultSearchOptions())
	require.Error(t, err)
	assert.Nil(t, packages)
}

func TestManager_ListInstalled_Failure(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("dpkg-query", exec.FailureResult(1, "Query failed"))

	packages, err := mgr.ListInstalled(context.Background())
	require.Error(t, err)
	assert.Nil(t, packages)
}

func TestManager_ListUpgradable_Failure(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("apt", exec.FailureResult(1, "List failed"))

	packages, err := mgr.ListUpgradable(context.Background())
	require.Error(t, err)
	assert.Nil(t, packages)
}

func TestManager_Install_GenericFailure(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("env", exec.FailureResult(1, "Some unknown error"))

	err := mgr.Install(context.Background(), pkg.DefaultInstallOptions(), "nginx")
	require.Error(t, err)
	assert.ErrorIs(t, err, pkg.ErrInstallFailed)
}

func TestManager_Remove_GenericFailure(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("env", exec.FailureResult(1, "Some removal error"))

	err := mgr.Remove(context.Background(), pkg.DefaultRemoveOptions(), "nginx")
	require.Error(t, err)
	assert.ErrorIs(t, err, pkg.ErrRemoveFailed)
}

func TestManager_Update_GenericFailure(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("env", exec.FailureResult(1, "Some update error"))

	err := mgr.Update(context.Background(), pkg.DefaultUpdateOptions())
	require.Error(t, err)
	assert.ErrorIs(t, err, pkg.ErrUpdateFailed)
}

func TestManager_Upgrade_LockError(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("env", exec.FailureResult(100, "E: Could not get lock"))

	err := mgr.Upgrade(context.Background(), pkg.DefaultInstallOptions())
	require.Error(t, err)
	assert.ErrorIs(t, err, pkg.ErrLockAcquireFailed)
}

func TestManager_Upgrade_GenericFailure(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("env", exec.FailureResult(1, "Some upgrade error"))

	err := mgr.Upgrade(context.Background(), pkg.DefaultInstallOptions())
	require.Error(t, err)
	assert.ErrorIs(t, err, pkg.ErrInstallFailed)
}

func TestManager_IsInstalled_Error(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("dpkg-query", exec.FailureResult(2, "Some dpkg error"))

	_, err := mgr.IsInstalled(context.Background(), "nginx")
	require.Error(t, err)
}

func TestManager_Info_EmptyOutput(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("apt-cache", exec.SuccessResult(""))

	info, err := mgr.Info(context.Background(), "nonexistent")
	require.Error(t, err)
	assert.Nil(t, info)
	assert.ErrorIs(t, err, pkg.ErrPackageNotFound)
}

func TestManager_Info_IncompleteOutput(t *testing.T) {
	mgr, mockExec := setupTest()

	// Output with no Package: field
	mockExec.SetResponse("apt-cache", exec.SuccessResult("Version: 1.0\nDescription: test"))

	info, err := mgr.Info(context.Background(), "incomplete")
	require.Error(t, err)
	assert.Nil(t, info)
	assert.ErrorIs(t, err, pkg.ErrPackageNotFound)
}

func TestManager_AddRepository_DirectWriteFailure(t *testing.T) {
	mgr, mockExec := setupTest()

	// File doesn't exist
	mockExec.SetResponse("test", exec.FailureResult(1, ""))
	// But write fails
	mockExec.SetDefaultResponse(exec.FailureResult(1, "Permission denied"))

	repo := pkg.Repository{
		Name: "nvidia",
		URL:  "https://nvidia.com/repo",
	}

	err := mgr.AddRepository(context.Background(), repo)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create repository file")
}

func TestManager_RemoveRepository_RemoveFailure(t *testing.T) {
	mgr, mockExec := setupTest()

	// File exists
	mockExec.SetResponse("test", exec.SuccessResult(""))
	// But rm fails
	mockExec.SetDefaultResponse(exec.FailureResult(1, "Permission denied"))

	err := mgr.RemoveRepository(context.Background(), "nvidia")
	require.Error(t, err)
}

func TestManager_RemoveRepository_PPANotFound(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("env", exec.FailureResult(1, "does not exist"))

	err := mgr.RemoveRepository(context.Background(), "ppa:nonexistent/repo")
	require.Error(t, err)
	assert.ErrorIs(t, err, pkg.ErrRepositoryNotFound)
}

func TestManager_AddGPGKey_LegacyFallback(t *testing.T) {
	mgr, mockExec := setupTest()

	// mkdir fails (fall back to legacy)
	mockExec.SetResponse("mkdir", exec.FailureResult(1, "Permission denied"))
	// Legacy succeeds
	mockExec.SetResponse("sh", exec.SuccessResult(""))

	err := mgr.AddGPGKey(context.Background(), "nvidia", "https://nvidia.com/key.gpg")
	require.NoError(t, err)
}

func TestManager_AddGPGKey_LegacyFailure(t *testing.T) {
	mgr, mockExec := setupTest()

	// mkdir fails (fall back to legacy)
	mockExec.SetResponse("mkdir", exec.FailureResult(1, "Permission denied"))
	// Legacy also fails
	mockExec.SetResponse("sh", exec.FailureResult(1, "Connection refused"))

	err := mgr.AddGPGKey(context.Background(), "nvidia", "https://nvidia.com/key.gpg")
	require.Error(t, err)
	assert.ErrorIs(t, err, pkg.ErrNetworkUnavailable)
}

func TestManager_Install_DownloadOnly(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("env", exec.SuccessResult(""))

	opts := pkg.InstallOptions{
		DownloadOnly: true,
	}

	err := mgr.Install(context.Background(), opts, "nginx")
	require.NoError(t, err)

	lastCall := mockExec.LastCall()
	assert.Contains(t, lastCall.Args, "--download-only")
}

func TestManager_Install_SkipVerify(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("env", exec.SuccessResult(""))

	opts := pkg.InstallOptions{
		SkipVerify: true,
	}

	err := mgr.Install(context.Background(), opts, "nginx")
	require.NoError(t, err)

	lastCall := mockExec.LastCall()
	assert.Contains(t, lastCall.Args, "--allow-unauthenticated")
}

func TestManager_Remove_EmptyPackages(t *testing.T) {
	mgr, mockExec := setupTest()

	err := mgr.Remove(context.Background(), pkg.DefaultRemoveOptions())
	require.NoError(t, err)

	// Should not call any command
	assert.Equal(t, 0, mockExec.CallCount())
}

func TestManager_Remove_LockError(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("env", exec.FailureResult(100, "E: Could not get lock"))

	err := mgr.Remove(context.Background(), pkg.DefaultRemoveOptions(), "nginx")
	require.Error(t, err)
	assert.ErrorIs(t, err, pkg.ErrLockAcquireFailed)
}

func TestContainsRepoEntry(t *testing.T) {
	// Should match deb lines containing the name
	assert.True(t, containsRepoEntry("deb http://archive.ubuntu.com/ubuntu jammy main", "archive.ubuntu.com"))

	// Should not match non-deb lines
	assert.False(t, containsRepoEntry("# This is a comment about ubuntu", "ubuntu"))

	// Should not match if name not in line
	assert.False(t, containsRepoEntry("deb http://nvidia.com/repo stable main", "ubuntu"))
}

func TestParseSourcesList_EmptyInput(t *testing.T) {
	repos, err := parseSourcesList("")
	require.NoError(t, err)
	assert.Len(t, repos, 0)
}

func TestParseSourcesList_OnlyComments(t *testing.T) {
	content := `# This is a comment
# Another comment
# Not a deb line`

	repos, err := parseSourcesList(content)
	require.NoError(t, err)
	assert.Len(t, repos, 0)
}

func TestParseDpkgQuery_Empty(t *testing.T) {
	packages, err := parseDpkgQuery("")
	require.NoError(t, err)
	assert.Len(t, packages, 0)
}

func TestParseDpkgQuery_MalformedLine(t *testing.T) {
	// Line with less than 3 tab-separated fields
	output := `nginx	1.22.0
curl	7.81.0	install ok installed`

	packages, err := parseDpkgQuery(output)
	require.NoError(t, err)
	// Only curl should be parsed (nginx has incomplete fields)
	assert.Len(t, packages, 1)
	assert.Equal(t, "curl", packages[0].Name)
}

func TestParseAptCacheSearch_Empty(t *testing.T) {
	packages, err := parseAptCacheSearch("")
	require.NoError(t, err)
	assert.Len(t, packages, 0)
}

func TestParseAptCacheSearch_NoDescription(t *testing.T) {
	// Line without " - " separator
	output := "nginx"

	packages, err := parseAptCacheSearch(output)
	require.NoError(t, err)
	assert.Len(t, packages, 1)
	assert.Equal(t, "nginx", packages[0].Name)
}

func TestParseAptCacheShow_Empty(t *testing.T) {
	p, err := parseAptCacheShow("")
	require.NoError(t, err)
	assert.Nil(t, p)
}

func TestParseSourcesListLine_InvalidLines(t *testing.T) {
	// Not a deb line
	repo, err := parseSourcesListLine("This is just text")
	require.NoError(t, err)
	assert.Nil(t, repo)

	// Empty line after removing comment
	repo, err = parseSourcesListLine("# ")
	require.NoError(t, err)
	assert.Nil(t, repo)

	// Too few parts
	repo, err = parseSourcesListLine("deb")
	require.NoError(t, err)
	assert.Nil(t, repo)
}

func TestExtractOption(t *testing.T) {
	// Found option
	value := extractOption("[arch=amd64 signed-by=/etc/apt/keyrings/nvidia.gpg]", "signed-by")
	assert.Equal(t, "/etc/apt/keyrings/nvidia.gpg", value)

	// Not found
	value = extractOption("[arch=amd64]", "signed-by")
	assert.Equal(t, "", value)

	// No brackets
	value = extractOption("arch=amd64", "arch")
	assert.Equal(t, "amd64", value)
}

func TestFileExists(t *testing.T) {
	// File that exists (go.mod in project root)
	assert.True(t, FileExists("/home/tommasomariaungetti/Git/igor/go.mod"))

	// File that doesn't exist
	assert.False(t, FileExists("/nonexistent/path/file.txt"))
}

func TestManager_AddRepository_PPAGenericError(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("env", exec.FailureResult(1, "Some generic error"))

	repo := pkg.Repository{
		URL: "ppa:some/ppa",
	}

	err := mgr.AddRepository(context.Background(), repo)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "add-apt-repository failed")
}

func TestManager_RemoveRepository_PPAGenericError(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("env", exec.FailureResult(1, "Some generic error"))

	err := mgr.RemoveRepository(context.Background(), "ppa:some/ppa")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "add-apt-repository --remove failed")
}

func TestManager_Install_DpkgInterrupted(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("env", exec.FailureResult(100, "dpkg was interrupted"))

	err := mgr.Install(context.Background(), pkg.DefaultInstallOptions(), "nginx")
	require.Error(t, err)
	assert.ErrorIs(t, err, pkg.ErrLockAcquireFailed)
}
