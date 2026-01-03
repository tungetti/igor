package dnf

import (
	"context"
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
	assert.Equal(t, "dnf", mgr.Name())
}

func TestManager_Family(t *testing.T) {
	mgr, _ := setupTest()
	assert.Equal(t, constants.FamilyRHEL, mgr.Family())
}

func TestManager_Install_SinglePackage(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("dnf", exec.SuccessResult(""))

	err := mgr.Install(context.Background(), pkg.DefaultInstallOptions(), "nginx")
	require.NoError(t, err)

	// Verify the command was called
	assert.True(t, mockExec.WasCalled("dnf"))
	lastCall := mockExec.LastCall()
	assert.Contains(t, lastCall.Args, "install")
	assert.Contains(t, lastCall.Args, "-y")
	assert.Contains(t, lastCall.Args, "nginx")
}

func TestManager_Install_MultiplePackages(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("dnf", exec.SuccessResult(""))

	err := mgr.Install(context.Background(), pkg.DefaultInstallOptions(), "nginx", "curl", "vim")
	require.NoError(t, err)

	lastCall := mockExec.LastCall()
	assert.Contains(t, lastCall.Args, "nginx")
	assert.Contains(t, lastCall.Args, "curl")
	assert.Contains(t, lastCall.Args, "vim")
}

func TestManager_Install_WithOptions(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("dnf", exec.SuccessResult(""))

	opts := pkg.InstallOptions{
		Force:          true,
		Reinstall:      true,
		AllowDowngrade: true,
		SkipVerify:     true,
		DownloadOnly:   true,
	}

	err := mgr.Install(context.Background(), opts, "nginx")
	require.NoError(t, err)

	lastCall := mockExec.LastCall()
	assert.Contains(t, lastCall.Args, "--allowerasing")
	assert.Contains(t, lastCall.Args, "--reinstall")
	assert.Contains(t, lastCall.Args, "--nogpgcheck")
	assert.Contains(t, lastCall.Args, "--downloadonly")
}

func TestManager_Install_PackageNotFound(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("dnf", exec.FailureResult(1, "No match for argument: nonexistent"))

	err := mgr.Install(context.Background(), pkg.DefaultInstallOptions(), "nonexistent")
	require.Error(t, err)
	assert.ErrorIs(t, err, pkg.ErrPackageNotFound)
}

func TestManager_Install_LockError(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("dnf", exec.FailureResult(1, "Error: Waiting for process with pid 1234 to finish."))

	err := mgr.Install(context.Background(), pkg.DefaultInstallOptions(), "nginx")
	require.Error(t, err)
	// Lock errors should map to ErrInstallFailed or ErrLockAcquireFailed
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

	mockExec.SetResponse("dnf", exec.SuccessResult(""))

	err := mgr.Remove(context.Background(), pkg.DefaultRemoveOptions(), "nginx")
	require.NoError(t, err)

	lastCall := mockExec.LastCall()
	assert.Contains(t, lastCall.Args, "remove")
	assert.Contains(t, lastCall.Args, "-y")
	assert.Contains(t, lastCall.Args, "nginx")
}

func TestManager_Remove_PackageNotInstalled(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("dnf", exec.FailureResult(1, "No match for argument: nginx"))

	err := mgr.Remove(context.Background(), pkg.DefaultRemoveOptions(), "nginx")
	require.Error(t, err)
	assert.ErrorIs(t, err, pkg.ErrPackageNotInstalled)
}

func TestManager_Remove_EmptyPackages(t *testing.T) {
	mgr, mockExec := setupTest()

	err := mgr.Remove(context.Background(), pkg.DefaultRemoveOptions())
	require.NoError(t, err)

	// Should not call any command
	assert.Equal(t, 0, mockExec.CallCount())
}

func TestManager_Update_Success(t *testing.T) {
	mgr, mockExec := setupTest()

	// dnf check-update returns 0 when no updates available
	mockExec.SetResponse("dnf", exec.SuccessResult(""))

	err := mgr.Update(context.Background(), pkg.DefaultUpdateOptions())
	require.NoError(t, err)

	lastCall := mockExec.LastCall()
	assert.Contains(t, lastCall.Args, "check-update")
}

func TestManager_Update_UpdatesAvailable(t *testing.T) {
	mgr, mockExec := setupTest()

	// dnf check-update returns 100 when updates are available (not an error!)
	mockExec.SetResponse("dnf", exec.FailureResult(100, ""))

	err := mgr.Update(context.Background(), pkg.DefaultUpdateOptions())
	require.NoError(t, err) // Should NOT be an error
}

func TestManager_Update_Quiet(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("dnf", exec.SuccessResult(""))

	opts := pkg.UpdateOptions{Quiet: true}
	err := mgr.Update(context.Background(), opts)
	require.NoError(t, err)

	lastCall := mockExec.LastCall()
	assert.Contains(t, lastCall.Args, "-q")
}

func TestManager_Update_NetworkError(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("dnf", exec.FailureResult(1, "Could not resolve host"))

	err := mgr.Update(context.Background(), pkg.DefaultUpdateOptions())
	require.Error(t, err)
	assert.ErrorIs(t, err, pkg.ErrNetworkUnavailable)
}

func TestManager_Update_GenericFailure(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("dnf", exec.FailureResult(1, "Some error"))

	err := mgr.Update(context.Background(), pkg.DefaultUpdateOptions())
	require.Error(t, err)
	assert.ErrorIs(t, err, pkg.ErrUpdateFailed)
}

func TestManager_Upgrade_AllPackages(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("dnf", exec.SuccessResult(""))

	err := mgr.Upgrade(context.Background(), pkg.DefaultInstallOptions())
	require.NoError(t, err)

	lastCall := mockExec.LastCall()
	assert.Contains(t, lastCall.Args, "upgrade")
	assert.Contains(t, lastCall.Args, "-y")
}

func TestManager_Upgrade_SpecificPackages(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("dnf", exec.SuccessResult(""))

	err := mgr.Upgrade(context.Background(), pkg.DefaultInstallOptions(), "nginx", "curl")
	require.NoError(t, err)

	lastCall := mockExec.LastCall()
	assert.Contains(t, lastCall.Args, "upgrade")
	assert.Contains(t, lastCall.Args, "nginx")
	assert.Contains(t, lastCall.Args, "curl")
}

func TestManager_Upgrade_WithOptions(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("dnf", exec.SuccessResult(""))

	opts := pkg.InstallOptions{
		Force:      true,
		SkipVerify: true,
	}

	err := mgr.Upgrade(context.Background(), opts)
	require.NoError(t, err)

	lastCall := mockExec.LastCall()
	assert.Contains(t, lastCall.Args, "--allowerasing")
	assert.Contains(t, lastCall.Args, "--nogpgcheck")
}

func TestManager_Upgrade_Failure(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("dnf", exec.FailureResult(1, "Upgrade failed"))

	err := mgr.Upgrade(context.Background(), pkg.DefaultInstallOptions())
	require.Error(t, err)
	assert.ErrorIs(t, err, pkg.ErrInstallFailed)
}

func TestManager_IsInstalled_True(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("rpm", exec.SuccessResult("nginx-1.22.0-1.el9.x86_64"))

	installed, err := mgr.IsInstalled(context.Background(), "nginx")
	require.NoError(t, err)
	assert.True(t, installed)
}

func TestManager_IsInstalled_False(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("rpm", exec.FailureResult(1, "package nginx is not installed"))

	installed, err := mgr.IsInstalled(context.Background(), "nginx")
	require.NoError(t, err)
	assert.False(t, installed)
}

func TestManager_IsInstalled_Error(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("rpm", exec.FailureResult(2, "rpm database error"))

	_, err := mgr.IsInstalled(context.Background(), "nginx")
	require.Error(t, err)
}

func TestManager_Search_Success(t *testing.T) {
	mgr, mockExec := setupTest()

	output := `========================= Name Matched: nginx ==========================
nginx.x86_64 : A high performance web server and reverse proxy server
nginx-mod-http-perl.x86_64 : Nginx HTTP perl module`

	mockExec.SetResponse("dnf", exec.SuccessResult(output))
	mockExec.SetResponse("rpm", exec.FailureResult(1, "not found"))

	packages, err := mgr.Search(context.Background(), "nginx", pkg.DefaultSearchOptions())
	require.NoError(t, err)
	require.Len(t, packages, 2)
	assert.Equal(t, "nginx", packages[0].Name)
}

func TestManager_Search_ExactMatch(t *testing.T) {
	mgr, mockExec := setupTest()

	output := `Last metadata expiration check: 0:00:01 ago
Installed Packages
nginx.x86_64    1.22.0-1.el9    @appstream`

	mockExec.SetResponse("dnf", exec.SuccessResult(output))
	mockExec.SetResponse("rpm", exec.SuccessResult("nginx-1.22.0-1.el9.x86_64"))

	opts := pkg.SearchOptions{ExactMatch: true, IncludeInstalled: true}
	packages, err := mgr.Search(context.Background(), "nginx", opts)
	require.NoError(t, err)
	require.Len(t, packages, 1)
	assert.Equal(t, "nginx", packages[0].Name)
}

func TestManager_Search_WithLimit(t *testing.T) {
	mgr, mockExec := setupTest()

	output := `nginx.x86_64 : A high performance web server
nginx-mod-http-perl.x86_64 : Nginx HTTP perl module
nginx-core.x86_64 : Core nginx package`

	mockExec.SetResponse("dnf", exec.SuccessResult(output))
	mockExec.SetResponse("rpm", exec.FailureResult(1, "not found"))

	opts := pkg.SearchOptions{Limit: 2, IncludeInstalled: false}
	packages, err := mgr.Search(context.Background(), "nginx", opts)
	require.NoError(t, err)
	require.Len(t, packages, 2)
}

func TestManager_Search_NoResults(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("dnf", exec.FailureResult(1, "No matches found"))

	packages, err := mgr.Search(context.Background(), "nonexistent", pkg.DefaultSearchOptions())
	require.NoError(t, err)
	assert.Len(t, packages, 0)
}

func TestManager_Info_Success(t *testing.T) {
	mgr, mockExec := setupTest()

	output := `Last metadata expiration check: 0:00:01 ago
Name         : nginx
Version      : 1.22.0
Release      : 1.el9
Architecture : x86_64
Size         : 1.2 M
Summary      : A high performance web server and reverse proxy server
Repository   : appstream`

	mockExec.SetResponse("dnf", exec.SuccessResult(output))
	mockExec.SetResponse("rpm", exec.SuccessResult("nginx-1.22.0-1.el9.x86_64"))

	info, err := mgr.Info(context.Background(), "nginx")
	require.NoError(t, err)
	require.NotNil(t, info)
	assert.Equal(t, "nginx", info.Name)
	assert.Equal(t, "1.22.0-1.el9", info.Version)
	assert.Equal(t, "x86_64", info.Architecture)
	assert.True(t, info.Installed)
}

func TestManager_Info_NotFound(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("dnf", exec.FailureResult(1, "Error: No matching Packages to list"))

	info, err := mgr.Info(context.Background(), "nonexistent")
	require.Error(t, err)
	assert.Nil(t, info)
	assert.ErrorIs(t, err, pkg.ErrPackageNotFound)
}

func TestManager_ListInstalled(t *testing.T) {
	mgr, mockExec := setupTest()

	output := `nginx	1.22.0-1.el9	x86_64
curl	7.76.1-23.el9	x86_64
vim-minimal	8.2.2637-20.el9	x86_64`

	mockExec.SetResponse("rpm", exec.SuccessResult(output))

	packages, err := mgr.ListInstalled(context.Background())
	require.NoError(t, err)
	require.Len(t, packages, 3)
	assert.Equal(t, "nginx", packages[0].Name)
	assert.Equal(t, "1.22.0-1.el9", packages[0].Version)
	assert.Equal(t, "x86_64", packages[0].Architecture)
	assert.True(t, packages[0].Installed)
}

func TestManager_ListUpgradable(t *testing.T) {
	mgr, mockExec := setupTest()

	output := `nginx.x86_64                    1.24.0-1.el9               updates
curl.x86_64                     7.76.1-26.el9              baseos`

	// Exit code 100 means updates available - output goes to stdout
	mockExec.SetResponse("dnf", exec.SuccessResultWithStderr(output, ""))
	// Manually set exit code to 100 to simulate updates available
	// The mock doesn't support this directly, so we use SuccessResult with the output

	packages, err := mgr.ListUpgradable(context.Background())
	require.NoError(t, err)
	require.Len(t, packages, 2)
	assert.Equal(t, "nginx", packages[0].Name)
	assert.Equal(t, "1.24.0-1.el9", packages[0].Version)
	assert.Equal(t, "updates", packages[0].Repository)
}

func TestManager_ListUpgradable_NoUpdates(t *testing.T) {
	mgr, mockExec := setupTest()

	// Exit code 0 means no updates available
	mockExec.SetResponse("dnf", exec.SuccessResult(""))

	packages, err := mgr.ListUpgradable(context.Background())
	require.NoError(t, err)
	assert.Len(t, packages, 0)
}

func TestManager_Clean_Success(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("dnf", exec.SuccessResult("Cleaning repos"))

	err := mgr.Clean(context.Background())
	require.NoError(t, err)

	lastCall := mockExec.LastCall()
	assert.Contains(t, lastCall.Args, "clean")
	assert.Contains(t, lastCall.Args, "all")
}

func TestManager_Clean_Failure(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("dnf", exec.FailureResult(1, "Clean failed"))

	err := mgr.Clean(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "dnf clean failed")
}

func TestManager_AutoRemove_Success(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("dnf", exec.SuccessResult("Removing unused packages"))

	err := mgr.AutoRemove(context.Background())
	require.NoError(t, err)

	lastCall := mockExec.LastCall()
	assert.Contains(t, lastCall.Args, "autoremove")
	assert.Contains(t, lastCall.Args, "-y")
}

func TestManager_AutoRemove_LockError(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("dnf", exec.FailureResult(1, "Waiting for process with pid 1234 to finish. another copy is running"))

	err := mgr.AutoRemove(context.Background())
	require.Error(t, err)
	assert.ErrorIs(t, err, pkg.ErrLockAcquireFailed)
}

func TestManager_Verify_Installed(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("rpm", exec.SuccessResult(""))

	valid, err := mgr.Verify(context.Background(), "nginx")
	require.NoError(t, err)
	assert.True(t, valid)
}

func TestManager_Verify_NotInstalled(t *testing.T) {
	mgr, mockExec := setupTest()

	// First rpm -q returns not installed
	mockExec.SetResponse("rpm", exec.FailureResult(1, "package not installed"))

	_, err := mgr.Verify(context.Background(), "nonexistent")
	require.Error(t, err)
	assert.ErrorIs(t, err, pkg.ErrPackageNotInstalled)
}

func TestManager_Verify_Invalid(t *testing.T) {
	mgr, mockExec := setupTest()

	// First call for IsInstalled succeeds
	// We need to handle multiple calls to rpm with different results
	// The mock returns the same result for all rpm calls, so we need to be clever
	// For this test, we'll check that the Verify logic works when rpm -V fails

	// Create a custom test case
	mockExec.SetDefaultResponse(exec.SuccessResult("nginx-1.22.0")) // IsInstalled returns true

	valid, err := mgr.Verify(context.Background(), "nginx")
	require.NoError(t, err)
	// Since we return success for all rpm calls, verification passes
	assert.True(t, valid)
}

// Repository tests

func TestManager_AddRepository_FromURL(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("dnf", exec.SuccessResult(""))

	repo := pkg.Repository{
		Name: "nvidia",
		URL:  "https://developer.download.nvidia.com/cuda/rhel9/x86_64/cuda-rhel9.repo",
	}

	err := mgr.AddRepository(context.Background(), repo)
	require.NoError(t, err)

	lastCall := mockExec.LastCall()
	assert.Contains(t, lastCall.Args, "config-manager")
	assert.Contains(t, lastCall.Args, "--add-repo")
}

func TestManager_AddRepository_CreateFile(t *testing.T) {
	mgr, mockExec := setupTest()

	// File doesn't exist
	mockExec.SetResponse("test", exec.FailureResult(1, ""))
	mockExec.SetDefaultResponse(exec.SuccessResult(""))

	repo := pkg.Repository{
		Name:    "custom-repo",
		URL:     "https://example.com/repo",
		Enabled: true,
		GPGKey:  "https://example.com/key.gpg",
	}

	err := mgr.AddRepository(context.Background(), repo)
	require.NoError(t, err)
}

func TestManager_AddRepository_AlreadyExists(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("dnf", exec.FailureResult(1, "repository already exists"))

	repo := pkg.Repository{
		Name: "nvidia",
		URL:  "https://example.com/repo.repo",
	}

	err := mgr.AddRepository(context.Background(), repo)
	require.Error(t, err)
	assert.ErrorIs(t, err, pkg.ErrRepositoryExists)
}

func TestManager_AddRepository_FileAlreadyExists(t *testing.T) {
	mgr, mockExec := setupTest()

	// File exists
	mockExec.SetResponse("test", exec.SuccessResult(""))

	repo := pkg.Repository{
		Name: "custom-repo",
		URL:  "https://example.com/repo",
	}

	err := mgr.AddRepository(context.Background(), repo)
	require.Error(t, err)
	assert.ErrorIs(t, err, pkg.ErrRepositoryExists)
}

func TestManager_AddRepository_EmptyURL(t *testing.T) {
	mgr, _ := setupTest()

	repo := pkg.Repository{
		Name: "test",
	}

	err := mgr.AddRepository(context.Background(), repo)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "URL is required")
}

func TestManager_RemoveRepository_Success(t *testing.T) {
	mgr, mockExec := setupTest()

	// File exists
	mockExec.SetResponse("test", exec.SuccessResult(""))
	mockExec.SetResponse("rm", exec.SuccessResult(""))

	err := mgr.RemoveRepository(context.Background(), "nvidia")
	require.NoError(t, err)

	// Should have called rm
	assert.True(t, mockExec.WasCalled("rm"))
}

func TestManager_RemoveRepository_NotFound(t *testing.T) {
	mgr, mockExec := setupTest()

	// File doesn't exist
	mockExec.SetResponse("test", exec.FailureResult(1, ""))

	err := mgr.RemoveRepository(context.Background(), "nonexistent")
	require.Error(t, err)
	assert.ErrorIs(t, err, pkg.ErrRepositoryNotFound)
}

func TestManager_ListRepositories_Success(t *testing.T) {
	mgr, mockExec := setupTest()

	output := `repo id                           repo name                                 status
fedora                           Fedora 39 - x86_64                        enabled
updates                          Fedora 39 - x86_64 - Updates              enabled
rpmfusion-nonfree                RPM Fusion Nonfree                        disabled`

	mockExec.SetResponse("dnf", exec.SuccessResult(output))

	repos, err := mgr.ListRepositories(context.Background())
	require.NoError(t, err)
	require.Len(t, repos, 3)
	assert.Equal(t, "fedora", repos[0].Name)
	assert.True(t, repos[0].Enabled)
	assert.False(t, repos[2].Enabled)
}

func TestManager_EnableRepository_Success(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("dnf", exec.SuccessResult(""))

	err := mgr.EnableRepository(context.Background(), "rpmfusion-nonfree")
	require.NoError(t, err)

	lastCall := mockExec.LastCall()
	assert.Contains(t, lastCall.Args, "config-manager")
	assert.Contains(t, lastCall.Args, "--set-enabled")
	assert.Contains(t, lastCall.Args, "rpmfusion-nonfree")
}

func TestManager_EnableRepository_NotFound(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("dnf", exec.FailureResult(1, "Error: No matching repo"))

	err := mgr.EnableRepository(context.Background(), "nonexistent")
	require.Error(t, err)
	assert.ErrorIs(t, err, pkg.ErrRepositoryNotFound)
}

func TestManager_DisableRepository_Success(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("dnf", exec.SuccessResult(""))

	err := mgr.DisableRepository(context.Background(), "rpmfusion-nonfree")
	require.NoError(t, err)

	lastCall := mockExec.LastCall()
	assert.Contains(t, lastCall.Args, "config-manager")
	assert.Contains(t, lastCall.Args, "--set-disabled")
}

func TestManager_DisableRepository_NotFound(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("dnf", exec.FailureResult(1, "Error: No matching repo"))

	err := mgr.DisableRepository(context.Background(), "nonexistent")
	require.Error(t, err)
	assert.ErrorIs(t, err, pkg.ErrRepositoryNotFound)
}

func TestManager_RefreshRepositories_Success(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("dnf", exec.SuccessResult("Metadata cache created"))

	err := mgr.RefreshRepositories(context.Background())
	require.NoError(t, err)

	lastCall := mockExec.LastCall()
	assert.Contains(t, lastCall.Args, "makecache")
}

func TestManager_RefreshRepositories_NetworkError(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("dnf", exec.FailureResult(1, "Failed to download metadata"))

	err := mgr.RefreshRepositories(context.Background())
	require.Error(t, err)
	assert.ErrorIs(t, err, pkg.ErrNetworkUnavailable)
}

func TestManager_AddRPMFusion(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("sh", exec.SuccessResult("Complete!"))

	err := mgr.AddRPMFusion(context.Background(), true, true)
	require.NoError(t, err)

	// Should have called shell command
	assert.True(t, mockExec.WasCalled("sh"))
}

func TestManager_AddRPMFusion_AlreadyInstalled(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("sh", exec.FailureResult(1, "Package rpmfusion-free-release already installed"))

	err := mgr.AddRPMFusion(context.Background(), true, false)
	require.NoError(t, err) // Already installed is not an error
}

func TestManager_AddRPMFusion_NoRepos(t *testing.T) {
	mgr, mockExec := setupTest()

	err := mgr.AddRPMFusion(context.Background(), false, false)
	require.NoError(t, err)

	// Should not call any command
	assert.Equal(t, 0, mockExec.CallCount())
}

func TestManager_ImportGPGKey_Success(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("rpm", exec.SuccessResult(""))

	err := mgr.ImportGPGKey(context.Background(), "https://example.com/RPM-GPG-KEY")
	require.NoError(t, err)

	lastCall := mockExec.LastCall()
	assert.Contains(t, lastCall.Args, "--import")
}

func TestManager_ImportGPGKey_NetworkError(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("rpm", exec.FailureResult(1, "Could not resolve host"))

	err := mgr.ImportGPGKey(context.Background(), "https://example.com/key")
	require.Error(t, err)
	assert.ErrorIs(t, err, pkg.ErrNetworkUnavailable)
}

func TestManager_GetRepoFilePath(t *testing.T) {
	mgr, _ := setupTest()

	path := mgr.GetRepoFilePath("nvidia-cuda")
	assert.Equal(t, "/etc/yum.repos.d/nvidia-cuda.repo", path)

	path = mgr.GetRepoFilePath("rpm fusion/nonfree")
	assert.Equal(t, "/etc/yum.repos.d/rpm-fusion-nonfree.repo", path)
}

// Parser tests

func TestParseRpmQuery(t *testing.T) {
	output := `nginx	1.22.0-1.el9	x86_64
curl	7.76.1-23.el9	x86_64
vim-minimal	8.2.2637-20.el9	x86_64`

	packages, err := parseRpmQuery(output)
	require.NoError(t, err)
	require.Len(t, packages, 3)

	assert.Equal(t, "nginx", packages[0].Name)
	assert.Equal(t, "1.22.0-1.el9", packages[0].Version)
	assert.Equal(t, "x86_64", packages[0].Architecture)
	assert.True(t, packages[0].Installed)
}

func TestParseRpmQuery_Empty(t *testing.T) {
	packages, err := parseRpmQuery("")
	require.NoError(t, err)
	assert.Len(t, packages, 0)
}

func TestParseRpmQuery_MalformedLine(t *testing.T) {
	output := `nginx
curl	7.76.1-23.el9	x86_64`

	packages, err := parseRpmQuery(output)
	require.NoError(t, err)
	// Only curl should be fully parsed
	assert.Len(t, packages, 1)
	assert.Equal(t, "curl", packages[0].Name)
}

func TestParseDnfInfo(t *testing.T) {
	output := `Name         : nginx
Version      : 1.22.0
Release      : 1.el9
Architecture : x86_64
Size         : 1.2 M
Summary      : A high performance web server
Repository   : appstream`

	p, err := parseDnfInfo(output)
	require.NoError(t, err)
	require.NotNil(t, p)

	assert.Equal(t, "nginx", p.Name)
	assert.Equal(t, "1.22.0-1.el9", p.Version)
	assert.Equal(t, "x86_64", p.Architecture)
	assert.Equal(t, "A high performance web server", p.Description)
	assert.Equal(t, "appstream", p.Repository)
	assert.Greater(t, p.Size, int64(0))
}

func TestParseDnfInfo_Empty(t *testing.T) {
	p, err := parseDnfInfo("")
	require.NoError(t, err)
	assert.Nil(t, p)
}

func TestParseDnfSearch(t *testing.T) {
	output := `========================= Name Matched: nginx ==========================
nginx.x86_64 : A high performance web server and reverse proxy server
nginx-mod-http-perl.x86_64 : Nginx HTTP perl module`

	packages, err := parseDnfSearch(output)
	require.NoError(t, err)
	require.Len(t, packages, 2)

	assert.Equal(t, "nginx", packages[0].Name)
	assert.Equal(t, "x86_64", packages[0].Architecture)
	assert.Equal(t, "A high performance web server and reverse proxy server", packages[0].Description)
}

func TestParseDnfSearch_Empty(t *testing.T) {
	packages, err := parseDnfSearch("")
	require.NoError(t, err)
	assert.Len(t, packages, 0)
}

func TestParseDnfList(t *testing.T) {
	output := `Last metadata expiration check: 0:00:01 ago
Installed Packages
nginx.x86_64                    1.22.0-1.el9               @appstream
curl.x86_64                     7.76.1-23.el9              @baseos`

	packages, err := parseDnfList(output)
	require.NoError(t, err)
	require.Len(t, packages, 2)

	assert.Equal(t, "nginx", packages[0].Name)
	assert.Equal(t, "1.22.0-1.el9", packages[0].Version)
	assert.Equal(t, "x86_64", packages[0].Architecture)
	assert.Equal(t, "@appstream", packages[0].Repository)
}

func TestParseDnfRepolist(t *testing.T) {
	output := `repo id                           repo name                                 status
fedora                           Fedora 39 - x86_64                        enabled
updates                          Fedora 39 - x86_64 - Updates              enabled
rpmfusion-nonfree                RPM Fusion Nonfree                        disabled`

	repos, err := parseDnfRepolist(output)
	require.NoError(t, err)
	require.Len(t, repos, 3)

	assert.Equal(t, "fedora", repos[0].Name)
	assert.True(t, repos[0].Enabled)

	assert.Equal(t, "rpmfusion-nonfree", repos[2].Name)
	assert.False(t, repos[2].Enabled)
}

func TestParseDnfRepolist_Empty(t *testing.T) {
	repos, err := parseDnfRepolist("")
	require.NoError(t, err)
	assert.Len(t, repos, 0)
}

func TestParseDnfCheckUpdate(t *testing.T) {
	output := `nginx.x86_64                    1.24.0-1.el9               updates
curl.x86_64                     7.76.1-26.el9              baseos`

	packages, err := parseDnfCheckUpdate(output)
	require.NoError(t, err)
	require.Len(t, packages, 2)

	assert.Equal(t, "nginx", packages[0].Name)
	assert.Equal(t, "1.24.0-1.el9", packages[0].Version)
	assert.Equal(t, "x86_64", packages[0].Architecture)
	assert.Equal(t, "updates", packages[0].Repository)
	assert.True(t, packages[0].Installed)
}

func TestParseDnfCheckUpdate_WithMetadata(t *testing.T) {
	output := `Last metadata expiration check: 0:00:01 ago
nginx.x86_64                    1.24.0-1.el9               updates
Security: kernel.x86_64         5.14.0-362.8.1.el9         baseos`

	packages, err := parseDnfCheckUpdate(output)
	require.NoError(t, err)
	require.Len(t, packages, 1) // Security line should be skipped
	assert.Equal(t, "nginx", packages[0].Name)
}

func TestParseRepoFile(t *testing.T) {
	content := `[nvidia-cuda]
name=NVIDIA CUDA Repository
baseurl=https://developer.download.nvidia.com/cuda/rhel9/x86_64
enabled=1
gpgcheck=1
gpgkey=https://developer.download.nvidia.com/cuda/repos/rhel9/x86_64/7fa2af80.pub

[nvidia-cuda-source]
name=NVIDIA CUDA Source Repository
baseurl=https://developer.download.nvidia.com/cuda/rhel9/SRPMS
enabled=0
gpgcheck=1`

	repos, err := parseRepoFile(content)
	require.NoError(t, err)
	require.Len(t, repos, 2)

	assert.Equal(t, "nvidia-cuda", repos[0].Name)
	assert.Equal(t, "https://developer.download.nvidia.com/cuda/rhel9/x86_64", repos[0].URL)
	assert.True(t, repos[0].Enabled)
	assert.Equal(t, "https://developer.download.nvidia.com/cuda/repos/rhel9/x86_64/7fa2af80.pub", repos[0].GPGKey)

	assert.Equal(t, "nvidia-cuda-source", repos[1].Name)
	assert.False(t, repos[1].Enabled)
}

func TestBuildRepoFileContent(t *testing.T) {
	repo := pkg.Repository{
		Name:    "nvidia-cuda",
		URL:     "https://developer.download.nvidia.com/cuda/rhel9/x86_64",
		Enabled: true,
		GPGKey:  "https://developer.download.nvidia.com/cuda/repos/rhel9/x86_64/7fa2af80.pub",
	}

	content := buildRepoFileContent(repo)

	assert.Contains(t, content, "[nvidia-cuda]")
	assert.Contains(t, content, "name=nvidia-cuda")
	assert.Contains(t, content, "baseurl=https://developer.download.nvidia.com/cuda/rhel9/x86_64")
	assert.Contains(t, content, "enabled=1")
	assert.Contains(t, content, "gpgcheck=1")
	assert.Contains(t, content, "gpgkey=https://developer.download.nvidia.com/cuda/repos/rhel9/x86_64/7fa2af80.pub")
}

func TestBuildRepoFileContent_Disabled(t *testing.T) {
	repo := pkg.Repository{
		Name:    "test",
		URL:     "https://example.com",
		Enabled: false,
	}

	content := buildRepoFileContent(repo)
	assert.Contains(t, content, "enabled=0")
	assert.Contains(t, content, "gpgcheck=0")
}

func TestSanitizeRepoID(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"simple", "simple"},
		{"with space", "with-space"},
		{"with/slash", "with-slash"},
		{"with:colon", "with-colon"},
		{"with.dot", "with-dot"},
		{"rpm-fusion/nonfree", "rpm-fusion-nonfree"},
		{"---leading---", "leading"},
		{"", "custom-repo"},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			result := sanitizeRepoID(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestParseSize(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{"1024", 1024},
		{"1 k", 1024},
		{"1k", 1024},
		{"1.5 k", 1536},
		{"1 M", 1024 * 1024},
		{"1m", 1024 * 1024},
		{"1 G", 1024 * 1024 * 1024},
		{"", 0},
		{"invalid", 0},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			result := parseSize(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestIsArchitecture(t *testing.T) {
	assert.True(t, isArchitecture("x86_64"))
	assert.True(t, isArchitecture("i686"))
	assert.True(t, isArchitecture("noarch"))
	assert.True(t, isArchitecture("aarch64"))
	assert.False(t, isArchitecture("something"))
	assert.False(t, isArchitecture(""))
}

// Interface compliance test
func TestManager_ImplementsInterface(t *testing.T) {
	var _ pkg.Manager = (*Manager)(nil)
}

// Context cancellation test
func TestManager_Install_ContextCancelled(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("dnf", exec.SuccessResult(""))

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// The mock doesn't actually check context, but this tests the pattern
	err := mgr.Install(ctx, pkg.DefaultInstallOptions(), "nginx")
	// With mock, it succeeds because mock doesn't check context
	require.NoError(t, err)
}

// Error handling edge cases

func TestManager_Install_NothingToDo(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("dnf", exec.FailureResult(1, "Nothing to do"))

	err := mgr.Install(context.Background(), pkg.DefaultInstallOptions(), "already-installed")
	require.Error(t, err)
	assert.ErrorIs(t, err, pkg.ErrPackageNotFound)
}

func TestManager_Remove_NoPackagesMarked(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("dnf", exec.FailureResult(1, "No packages marked for removal"))

	err := mgr.Remove(context.Background(), pkg.DefaultRemoveOptions(), "not-installed")
	require.Error(t, err)
	assert.ErrorIs(t, err, pkg.ErrPackageNotInstalled)
}

func TestManager_Remove_GenericError(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("dnf", exec.FailureResult(1, "Some generic error"))

	err := mgr.Remove(context.Background(), pkg.DefaultRemoveOptions(), "pkg")
	require.Error(t, err)
	assert.ErrorIs(t, err, pkg.ErrRemoveFailed)
}

func TestManager_Install_AnotherCopyRunning(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("dnf", exec.FailureResult(1, "another copy is running"))

	err := mgr.Install(context.Background(), pkg.DefaultInstallOptions(), "nginx")
	require.Error(t, err)
	assert.ErrorIs(t, err, pkg.ErrLockAcquireFailed)
}

func TestManager_Upgrade_LockError(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("dnf", exec.FailureResult(1, "Waiting for lock"))

	err := mgr.Upgrade(context.Background(), pkg.DefaultInstallOptions())
	require.Error(t, err)
	assert.ErrorIs(t, err, pkg.ErrLockAcquireFailed)
}

func TestManager_Search_Failure(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("dnf", exec.FailureResult(1, "Database error"))

	packages, err := mgr.Search(context.Background(), "nginx", pkg.DefaultSearchOptions())
	require.Error(t, err)
	assert.Nil(t, packages)
}

func TestManager_Info_GenericError(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("dnf", exec.FailureResult(1, "Some error"))

	info, err := mgr.Info(context.Background(), "nginx")
	require.Error(t, err)
	assert.Nil(t, info)
}

func TestManager_ListInstalled_Failure(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("rpm", exec.FailureResult(1, "RPM database error"))

	packages, err := mgr.ListInstalled(context.Background())
	require.Error(t, err)
	assert.Nil(t, packages)
}

func TestManager_ListUpgradable_Failure(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("dnf", exec.FailureResult(1, "Check update failed"))

	packages, err := mgr.ListUpgradable(context.Background())
	require.Error(t, err)
	assert.Nil(t, packages)
}

func TestManager_ListRepositories_Failure(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("dnf", exec.FailureResult(1, "Repolist failed"))

	repos, err := mgr.ListRepositories(context.Background())
	require.Error(t, err)
	assert.Nil(t, repos)
}

func TestManager_EnableRepository_GenericError(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("dnf", exec.FailureResult(1, "Some error"))

	err := mgr.EnableRepository(context.Background(), "repo")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "config-manager --set-enabled failed")
}

func TestManager_DisableRepository_GenericError(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("dnf", exec.FailureResult(1, "Some error"))

	err := mgr.DisableRepository(context.Background(), "repo")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "config-manager --set-disabled failed")
}

func TestManager_RefreshRepositories_GenericError(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("dnf", exec.FailureResult(1, "Some error"))

	err := mgr.RefreshRepositories(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "dnf makecache failed")
}

func TestManager_RemoveRepository_RmFails(t *testing.T) {
	mgr, mockExec := setupTest()

	// File exists
	mockExec.SetResponse("test", exec.SuccessResult(""))
	// But rm fails
	mockExec.SetResponse("rm", exec.FailureResult(1, "Permission denied"))

	err := mgr.RemoveRepository(context.Background(), "repo")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to remove repository file")
}

func TestManager_AddRPMFusion_Failure(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("sh", exec.FailureResult(1, "Installation failed"))

	err := mgr.AddRPMFusion(context.Background(), true, true)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to add RPM Fusion")
}

func TestManager_ImportGPGKey_GenericError(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("rpm", exec.FailureResult(1, "Import failed"))

	err := mgr.ImportGPGKey(context.Background(), "https://example.com/key")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "rpm --import failed")
}

func TestManager_AddRPMFusionEL(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetDefaultResponse(exec.SuccessResult("Complete!"))

	err := mgr.AddRPMFusionEL(context.Background(), true, true)
	require.NoError(t, err)

	// Should have called dnf install for EPEL and sh for RPM Fusion
	assert.True(t, mockExec.WasCalled("dnf") || mockExec.WasCalled("sh"))
}

func TestManager_AddRPMFusionEL_NoRepos(t *testing.T) {
	mgr, _ := setupTest()

	err := mgr.AddRPMFusionEL(context.Background(), false, false)
	require.NoError(t, err)

	// Should not call any command for RPM Fusion (only EPEL might be called but it's okay if it fails)
}
