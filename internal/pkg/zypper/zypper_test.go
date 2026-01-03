package zypper

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
	assert.Equal(t, "zypper", mgr.Name())
}

func TestManager_Family(t *testing.T) {
	mgr, _ := setupTest()
	assert.Equal(t, constants.FamilySUSE, mgr.Family())
}

func TestManager_Install_SinglePackage(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("zypper", exec.SuccessResult(""))

	err := mgr.Install(context.Background(), pkg.DefaultInstallOptions(), "nginx")
	require.NoError(t, err)

	// Verify the command was called
	assert.True(t, mockExec.WasCalled("zypper"))
	lastCall := mockExec.LastCall()
	assert.Contains(t, lastCall.Args, "--non-interactive")
	assert.Contains(t, lastCall.Args, "install")
	assert.Contains(t, lastCall.Args, "nginx")
}

func TestManager_Install_MultiplePackages(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("zypper", exec.SuccessResult(""))

	err := mgr.Install(context.Background(), pkg.DefaultInstallOptions(), "nginx", "curl", "vim")
	require.NoError(t, err)

	lastCall := mockExec.LastCall()
	assert.Contains(t, lastCall.Args, "nginx")
	assert.Contains(t, lastCall.Args, "curl")
	assert.Contains(t, lastCall.Args, "vim")
}

func TestManager_Install_WithOptions(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("zypper", exec.SuccessResult(""))

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
	assert.Contains(t, lastCall.Args, "--force")
	assert.Contains(t, lastCall.Args, "--no-gpg-checks")
	assert.Contains(t, lastCall.Args, "--download-only")
}

func TestManager_Install_PackageNotFound(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("zypper", exec.FailureResult(1, "No provider of 'nonexistent' found."))

	err := mgr.Install(context.Background(), pkg.DefaultInstallOptions(), "nonexistent")
	require.Error(t, err)
	assert.ErrorIs(t, err, pkg.ErrPackageNotFound)
}

func TestManager_Install_LockError(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("zypper", exec.FailureResult(1, "System management is locked by the application with pid 1234"))

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

	mockExec.SetResponse("zypper", exec.SuccessResult(""))

	err := mgr.Remove(context.Background(), pkg.DefaultRemoveOptions(), "nginx")
	require.NoError(t, err)

	lastCall := mockExec.LastCall()
	assert.Contains(t, lastCall.Args, "--non-interactive")
	assert.Contains(t, lastCall.Args, "remove")
	assert.Contains(t, lastCall.Args, "nginx")
}

func TestManager_Remove_WithPurge(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("zypper", exec.SuccessResult(""))

	opts := pkg.RemoveOptions{Purge: true}
	err := mgr.Remove(context.Background(), opts, "nginx")
	require.NoError(t, err)

	lastCall := mockExec.LastCall()
	assert.Contains(t, lastCall.Args, "--clean-deps")
}

func TestManager_Remove_PackageNotInstalled(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("zypper", exec.FailureResult(1, "nginx is not installed"))

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

	mockExec.SetResponse("zypper", exec.SuccessResult(""))

	err := mgr.Update(context.Background(), pkg.DefaultUpdateOptions())
	require.NoError(t, err)

	lastCall := mockExec.LastCall()
	assert.Contains(t, lastCall.Args, "refresh")
}

func TestManager_Update_ForceRefresh(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("zypper", exec.SuccessResult(""))

	opts := pkg.UpdateOptions{ForceRefresh: true}
	err := mgr.Update(context.Background(), opts)
	require.NoError(t, err)

	lastCall := mockExec.LastCall()
	assert.Contains(t, lastCall.Args, "--force")
}

func TestManager_Update_NetworkError(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("zypper", exec.FailureResult(1, "Could not resolve host"))

	err := mgr.Update(context.Background(), pkg.DefaultUpdateOptions())
	require.Error(t, err)
	assert.ErrorIs(t, err, pkg.ErrNetworkUnavailable)
}

func TestManager_Update_LockError(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("zypper", exec.FailureResult(1, "System management is locked"))

	err := mgr.Update(context.Background(), pkg.DefaultUpdateOptions())
	require.Error(t, err)
	assert.ErrorIs(t, err, pkg.ErrLockAcquireFailed)
}

func TestManager_Update_GenericFailure(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("zypper", exec.FailureResult(1, "Some error"))

	err := mgr.Update(context.Background(), pkg.DefaultUpdateOptions())
	require.Error(t, err)
	assert.ErrorIs(t, err, pkg.ErrUpdateFailed)
}

func TestManager_Upgrade_AllPackages(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("zypper", exec.SuccessResult(""))

	err := mgr.Upgrade(context.Background(), pkg.DefaultInstallOptions())
	require.NoError(t, err)

	lastCall := mockExec.LastCall()
	assert.Contains(t, lastCall.Args, "--non-interactive")
	assert.Contains(t, lastCall.Args, "dist-upgrade")
}

func TestManager_Upgrade_SpecificPackages(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("zypper", exec.SuccessResult(""))

	err := mgr.Upgrade(context.Background(), pkg.DefaultInstallOptions(), "nginx", "curl")
	require.NoError(t, err)

	lastCall := mockExec.LastCall()
	assert.Contains(t, lastCall.Args, "update")
	assert.Contains(t, lastCall.Args, "nginx")
	assert.Contains(t, lastCall.Args, "curl")
}

func TestManager_Upgrade_WithOptions(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("zypper", exec.SuccessResult(""))

	opts := pkg.InstallOptions{
		Force:      true,
		SkipVerify: true,
	}

	err := mgr.Upgrade(context.Background(), opts)
	require.NoError(t, err)

	lastCall := mockExec.LastCall()
	assert.Contains(t, lastCall.Args, "--force")
	assert.Contains(t, lastCall.Args, "--no-gpg-checks")
}

func TestManager_Upgrade_Failure(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("zypper", exec.FailureResult(1, "Upgrade failed"))

	err := mgr.Upgrade(context.Background(), pkg.DefaultInstallOptions())
	require.Error(t, err)
	assert.ErrorIs(t, err, pkg.ErrInstallFailed)
}

func TestManager_Upgrade_LockError(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("zypper", exec.FailureResult(1, "System management is locked"))

	err := mgr.Upgrade(context.Background(), pkg.DefaultInstallOptions())
	require.Error(t, err)
	assert.ErrorIs(t, err, pkg.ErrLockAcquireFailed)
}

func TestManager_IsInstalled_True(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("rpm", exec.SuccessResult("nginx-1.24.0-1.1.x86_64"))

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

	output := `Loading repository data...
Reading installed packages...
S | Name                      | Summary                                                  | Type
--+---------------------------+----------------------------------------------------------+--------
i | nginx                     | A high performance web server and reverse proxy server   | package
  | nginx-source              | Source code of nginx                                     | srcpackage`

	mockExec.SetResponse("zypper", exec.SuccessResult(output))
	mockExec.SetResponse("rpm", exec.FailureResult(1, "not found"))

	packages, err := mgr.Search(context.Background(), "nginx", pkg.SearchOptions{IncludeInstalled: false})
	require.NoError(t, err)
	require.Len(t, packages, 2)
	assert.Equal(t, "nginx", packages[0].Name)
	assert.True(t, packages[0].Installed) // The 'i' in status column indicates installed
}

func TestManager_Search_ExactMatch(t *testing.T) {
	mgr, mockExec := setupTest()

	output := `Loading repository data...
Reading installed packages...
S | Name  | Summary                                                  | Type
--+-------+----------------------------------------------------------+--------
i | nginx | A high performance web server and reverse proxy server   | package`

	mockExec.SetResponse("zypper", exec.SuccessResult(output))
	mockExec.SetResponse("rpm", exec.SuccessResult("nginx-1.24.0-1.1.x86_64"))

	opts := pkg.SearchOptions{ExactMatch: true, IncludeInstalled: true}
	packages, err := mgr.Search(context.Background(), "nginx", opts)
	require.NoError(t, err)
	require.Len(t, packages, 1)
	assert.Equal(t, "nginx", packages[0].Name)
}

func TestManager_Search_WithLimit(t *testing.T) {
	mgr, mockExec := setupTest()

	output := `S | Name          | Summary                        | Type
--+---------------+--------------------------------+--------
  | nginx         | Web server                     | package
  | nginx-source  | Source code                    | srcpackage
  | nginx-docs    | Documentation                  | package`

	mockExec.SetResponse("zypper", exec.SuccessResult(output))

	opts := pkg.SearchOptions{Limit: 2, IncludeInstalled: false}
	packages, err := mgr.Search(context.Background(), "nginx", opts)
	require.NoError(t, err)
	require.Len(t, packages, 2)
}

func TestManager_Search_NoResults(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("zypper", exec.FailureResult(104, "No matching items found."))

	packages, err := mgr.Search(context.Background(), "nonexistent", pkg.DefaultSearchOptions())
	require.NoError(t, err)
	assert.Len(t, packages, 0)
}

func TestManager_Search_Failure(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("zypper", exec.FailureResult(1, "Database error"))

	packages, err := mgr.Search(context.Background(), "nginx", pkg.DefaultSearchOptions())
	require.Error(t, err)
	assert.Nil(t, packages)
}

func TestManager_Info_Success(t *testing.T) {
	mgr, mockExec := setupTest()

	output := `Information for package nginx:
-------------------------------
Repository     : openSUSE-Tumbleweed-Oss
Name           : nginx
Version        : 1.24.0-1.1
Arch           : x86_64
Vendor         : openSUSE
Installed Size : 1.2 MiB
Installed      : Yes
Status         : up-to-date
Summary        : A high performance web server and reverse proxy server`

	mockExec.SetResponse("zypper", exec.SuccessResult(output))
	mockExec.SetResponse("rpm", exec.SuccessResult("nginx-1.24.0-1.1.x86_64"))

	info, err := mgr.Info(context.Background(), "nginx")
	require.NoError(t, err)
	require.NotNil(t, info)
	assert.Equal(t, "nginx", info.Name)
	assert.Equal(t, "1.24.0-1.1", info.Version)
	assert.Equal(t, "x86_64", info.Architecture)
	assert.True(t, info.Installed)
}

func TestManager_Info_NotFound(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("zypper", exec.FailureResult(1, "No matching items found."))

	info, err := mgr.Info(context.Background(), "nonexistent")
	require.Error(t, err)
	assert.Nil(t, info)
	assert.ErrorIs(t, err, pkg.ErrPackageNotFound)
}

func TestManager_Info_GenericError(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("zypper", exec.FailureResult(1, "Some error"))

	info, err := mgr.Info(context.Background(), "nginx")
	require.Error(t, err)
	assert.Nil(t, info)
}

func TestManager_ListInstalled(t *testing.T) {
	mgr, mockExec := setupTest()

	output := `nginx	1.24.0-1.1	x86_64
curl	8.1.0-1.1	x86_64
vim	9.0.0-1.1	x86_64`

	mockExec.SetResponse("rpm", exec.SuccessResult(output))

	packages, err := mgr.ListInstalled(context.Background())
	require.NoError(t, err)
	require.Len(t, packages, 3)
	assert.Equal(t, "nginx", packages[0].Name)
	assert.Equal(t, "1.24.0-1.1", packages[0].Version)
	assert.Equal(t, "x86_64", packages[0].Architecture)
	assert.True(t, packages[0].Installed)
}

func TestManager_ListInstalled_Failure(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("rpm", exec.FailureResult(1, "RPM database error"))

	packages, err := mgr.ListInstalled(context.Background())
	require.Error(t, err)
	assert.Nil(t, packages)
}

func TestManager_ListUpgradable(t *testing.T) {
	mgr, mockExec := setupTest()

	output := `Loading repository data...
Reading installed packages...
S | Repository                | Name       | Current Version | Available Version | Arch
--+---------------------------+------------+-----------------+-------------------+--------
v | openSUSE-Tumbleweed-Oss   | nginx      | 1.22.0-1.1      | 1.24.0-1.1       | x86_64
v | openSUSE-Tumbleweed-Oss   | curl       | 8.0.0-1.1       | 8.1.0-1.1        | x86_64`

	mockExec.SetResponse("zypper", exec.SuccessResult(output))

	packages, err := mgr.ListUpgradable(context.Background())
	require.NoError(t, err)
	require.Len(t, packages, 2)
	assert.Equal(t, "nginx", packages[0].Name)
	assert.Equal(t, "1.24.0-1.1", packages[0].Version)
	assert.Equal(t, "openSUSE-Tumbleweed-Oss", packages[0].Repository)
}

func TestManager_ListUpgradable_NoUpdates(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("zypper", exec.SuccessResult("No updates found."))

	packages, err := mgr.ListUpgradable(context.Background())
	require.NoError(t, err)
	assert.Len(t, packages, 0)
}

func TestManager_ListUpgradable_Failure(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("zypper", exec.FailureResult(1, "List updates failed"))

	packages, err := mgr.ListUpgradable(context.Background())
	require.Error(t, err)
	assert.Nil(t, packages)
}

func TestManager_Clean_Success(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("zypper", exec.SuccessResult("All repositories have been cleaned up."))

	err := mgr.Clean(context.Background())
	require.NoError(t, err)

	lastCall := mockExec.LastCall()
	assert.Contains(t, lastCall.Args, "clean")
	assert.Contains(t, lastCall.Args, "--all")
}

func TestManager_Clean_Failure(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("zypper", exec.FailureResult(1, "Clean failed"))

	err := mgr.Clean(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "zypper clean failed")
}

func TestManager_AutoRemove_Success(t *testing.T) {
	mgr, mockExec := setupTest()

	// First call: list unneeded packages
	unneededOutput := `S | Repository | Name              | Version       | Arch
--+------------+-------------------+---------------+--------
i | @System    | orphan-package    | 1.0.0-1.1     | x86_64`

	mockExec.SetResponse("zypper", exec.SuccessResult(unneededOutput))

	err := mgr.AutoRemove(context.Background())
	require.NoError(t, err)
}

func TestManager_AutoRemove_NoUnneeded(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("zypper", exec.SuccessResult("No packages found."))

	err := mgr.AutoRemove(context.Background())
	require.NoError(t, err)
}

func TestManager_AutoRemove_LockError(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("zypper", exec.FailureResult(1, "System management is locked"))

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

	// rpm -q returns not installed
	mockExec.SetResponse("rpm", exec.FailureResult(1, "package not installed"))

	_, err := mgr.Verify(context.Background(), "nonexistent")
	require.Error(t, err)
	assert.ErrorIs(t, err, pkg.ErrPackageNotInstalled)
}

// Repository tests

func TestManager_AddRepository_Success(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("zypper", exec.SuccessResult(""))
	mockExec.SetResponse("rpm", exec.SuccessResult(""))

	repo := pkg.Repository{
		Name:    "nvidia",
		URL:     "https://download.nvidia.com/opensuse/tumbleweed",
		Enabled: true,
	}

	err := mgr.AddRepository(context.Background(), repo)
	require.NoError(t, err)

	lastCall := mockExec.LastCall()
	assert.Contains(t, lastCall.Args, "addrepo")
	assert.Contains(t, lastCall.Args, "--refresh")
}

func TestManager_AddRepository_Disabled(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("zypper", exec.SuccessResult(""))

	repo := pkg.Repository{
		Name:    "nvidia",
		URL:     "https://download.nvidia.com/opensuse/tumbleweed",
		Enabled: false,
	}

	err := mgr.AddRepository(context.Background(), repo)
	require.NoError(t, err)

	lastCall := mockExec.LastCall()
	assert.Contains(t, lastCall.Args, "--disable")
}

func TestManager_AddRepository_AlreadyExists(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("zypper", exec.FailureResult(1, "Repository 'nvidia' already exists"))

	repo := pkg.Repository{
		Name: "nvidia",
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

	mockExec.SetResponse("zypper", exec.SuccessResult(""))

	err := mgr.RemoveRepository(context.Background(), "nvidia")
	require.NoError(t, err)

	lastCall := mockExec.LastCall()
	assert.Contains(t, lastCall.Args, "removerepo")
	assert.Contains(t, lastCall.Args, "nvidia")
}

func TestManager_RemoveRepository_NotFound(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("zypper", exec.FailureResult(1, "Repository 'nonexistent' not found"))

	err := mgr.RemoveRepository(context.Background(), "nonexistent")
	require.Error(t, err)
	assert.ErrorIs(t, err, pkg.ErrRepositoryNotFound)
}

func TestManager_ListRepositories_Success(t *testing.T) {
	mgr, mockExec := setupTest()

	output := `Repository priorities are without effect. All enabled repositories share the same priority.
# | Alias                     | Name                          | Enabled | GPG Check | Refresh
--+---------------------------+-------------------------------+---------+-----------+--------
1 | openSUSE-Tumbleweed-Oss   | openSUSE-Tumbleweed-Oss       | Yes     | (r ) Yes  | Yes
2 | nvidia                    | NVIDIA Repository             | Yes     | (  ) Yes  | Yes
3 | disabled-repo             | Disabled Repository           | No      | (  ) Yes  | No`

	mockExec.SetResponse("zypper", exec.SuccessResult(output))

	repos, err := mgr.ListRepositories(context.Background())
	require.NoError(t, err)
	require.Len(t, repos, 3)
	assert.Equal(t, "openSUSE-Tumbleweed-Oss", repos[0].Name)
	assert.True(t, repos[0].Enabled)
	assert.Equal(t, "nvidia", repos[1].Name)
	assert.True(t, repos[1].Enabled)
	assert.Equal(t, "disabled-repo", repos[2].Name)
	assert.False(t, repos[2].Enabled)
}

func TestManager_ListRepositories_NoRepos(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("zypper", exec.FailureResult(1, "No repositories defined."))

	repos, err := mgr.ListRepositories(context.Background())
	require.NoError(t, err)
	assert.Len(t, repos, 0)
}

func TestManager_ListRepositories_Failure(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("zypper", exec.FailureResult(1, "Repos failed"))

	repos, err := mgr.ListRepositories(context.Background())
	require.Error(t, err)
	assert.Nil(t, repos)
}

func TestManager_EnableRepository_Success(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("zypper", exec.SuccessResult(""))

	err := mgr.EnableRepository(context.Background(), "nvidia")
	require.NoError(t, err)

	lastCall := mockExec.LastCall()
	assert.Contains(t, lastCall.Args, "modifyrepo")
	assert.Contains(t, lastCall.Args, "--enable")
	assert.Contains(t, lastCall.Args, "nvidia")
}

func TestManager_EnableRepository_NotFound(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("zypper", exec.FailureResult(1, "Repository 'nonexistent' not found"))

	err := mgr.EnableRepository(context.Background(), "nonexistent")
	require.Error(t, err)
	assert.ErrorIs(t, err, pkg.ErrRepositoryNotFound)
}

func TestManager_DisableRepository_Success(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("zypper", exec.SuccessResult(""))

	err := mgr.DisableRepository(context.Background(), "nvidia")
	require.NoError(t, err)

	lastCall := mockExec.LastCall()
	assert.Contains(t, lastCall.Args, "modifyrepo")
	assert.Contains(t, lastCall.Args, "--disable")
}

func TestManager_DisableRepository_NotFound(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("zypper", exec.FailureResult(1, "Repository 'nonexistent' not found"))

	err := mgr.DisableRepository(context.Background(), "nonexistent")
	require.Error(t, err)
	assert.ErrorIs(t, err, pkg.ErrRepositoryNotFound)
}

func TestManager_RefreshRepositories_Success(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("zypper", exec.SuccessResult("All repositories have been refreshed."))

	err := mgr.RefreshRepositories(context.Background())
	require.NoError(t, err)

	lastCall := mockExec.LastCall()
	assert.Contains(t, lastCall.Args, "refresh")
}

func TestManager_RefreshRepositories_NetworkError(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("zypper", exec.FailureResult(1, "Could not resolve host"))

	err := mgr.RefreshRepositories(context.Background())
	require.Error(t, err)
	assert.ErrorIs(t, err, pkg.ErrNetworkUnavailable)
}

func TestManager_AddNvidiaRepo_Tumbleweed(t *testing.T) {
	mgr, mockExec := setupTest()

	// First call: list repos (empty)
	mockExec.SetResponse("zypper", exec.SuccessResult(""))

	err := mgr.AddNvidiaRepoTumbleweed(context.Background())
	require.NoError(t, err)

	// Should have called addrepo
	assert.True(t, mockExec.WasCalled("zypper"))
}

func TestManager_AddNvidiaRepo_Leap(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("zypper", exec.SuccessResult(""))

	err := mgr.AddNvidiaRepoLeap(context.Background(), "15.5")
	require.NoError(t, err)
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

	path := mgr.GetRepoFilePath("nvidia")
	assert.Equal(t, "/etc/zypp/repos.d/nvidia.repo", path)

	path = mgr.GetRepoFilePath("my repo/test")
	assert.Equal(t, "/etc/zypp/repos.d/my-repo-test.repo", path)
}

func TestManager_SetRepositoryPriority(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("zypper", exec.SuccessResult(""))

	err := mgr.SetRepositoryPriority(context.Background(), "nvidia", 90)
	require.NoError(t, err)

	lastCall := mockExec.LastCall()
	assert.Contains(t, lastCall.Args, "modifyrepo")
	assert.Contains(t, lastCall.Args, "--priority")
	assert.Contains(t, lastCall.Args, "90")
}

func TestManager_SetRepositoryRefresh(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("zypper", exec.SuccessResult(""))

	err := mgr.SetRepositoryRefresh(context.Background(), "nvidia", true)
	require.NoError(t, err)

	lastCall := mockExec.LastCall()
	assert.Contains(t, lastCall.Args, "--refresh")

	err = mgr.SetRepositoryRefresh(context.Background(), "nvidia", false)
	require.NoError(t, err)

	lastCall = mockExec.LastCall()
	assert.Contains(t, lastCall.Args, "--no-refresh")
}

// Parser tests

func TestParseRpmQuery(t *testing.T) {
	output := `nginx	1.24.0-1.1	x86_64
curl	8.1.0-1.1	x86_64
vim	9.0.0-1.1	x86_64`

	packages, err := parseRpmQuery(output)
	require.NoError(t, err)
	require.Len(t, packages, 3)

	assert.Equal(t, "nginx", packages[0].Name)
	assert.Equal(t, "1.24.0-1.1", packages[0].Version)
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
curl	8.1.0-1.1	x86_64`

	packages, err := parseRpmQuery(output)
	require.NoError(t, err)
	// Only curl should be fully parsed
	assert.Len(t, packages, 1)
	assert.Equal(t, "curl", packages[0].Name)
}

func TestParseZypperInfo(t *testing.T) {
	output := `Information for package nginx:
-------------------------------
Repository     : openSUSE-Tumbleweed-Oss
Name           : nginx
Version        : 1.24.0-1.1
Arch           : x86_64
Installed Size : 1.2 MiB
Summary        : A high performance web server`

	p, err := parseZypperInfo(output)
	require.NoError(t, err)
	require.NotNil(t, p)

	assert.Equal(t, "nginx", p.Name)
	assert.Equal(t, "1.24.0-1.1", p.Version)
	assert.Equal(t, "x86_64", p.Architecture)
	assert.Equal(t, "A high performance web server", p.Description)
	assert.Equal(t, "openSUSE-Tumbleweed-Oss", p.Repository)
	assert.Greater(t, p.Size, int64(0))
}

func TestParseZypperInfo_Empty(t *testing.T) {
	p, err := parseZypperInfo("")
	require.NoError(t, err)
	assert.Nil(t, p)
}

func TestParseZypperSearch(t *testing.T) {
	output := `Loading repository data...
Reading installed packages...
S | Name                      | Summary                                                  | Type
--+---------------------------+----------------------------------------------------------+--------
i | nginx                     | A high performance web server and reverse proxy server   | package
  | nginx-source              | Source code of nginx                                     | srcpackage`

	packages, err := parseZypperSearch(output)
	require.NoError(t, err)
	require.Len(t, packages, 2)

	assert.Equal(t, "nginx", packages[0].Name)
	assert.True(t, packages[0].Installed)
	assert.Equal(t, "A high performance web server and reverse proxy server", packages[0].Description)

	assert.Equal(t, "nginx-source", packages[1].Name)
	assert.False(t, packages[1].Installed)
}

func TestParseZypperSearch_Empty(t *testing.T) {
	packages, err := parseZypperSearch("")
	require.NoError(t, err)
	assert.Len(t, packages, 0)
}

func TestParseZypperRepos(t *testing.T) {
	output := `Repository priorities are without effect. All enabled repositories share the same priority.
# | Alias                     | Name                          | Enabled | GPG Check | Refresh
--+---------------------------+-------------------------------+---------+-----------+--------
1 | openSUSE-Tumbleweed-Oss   | openSUSE-Tumbleweed-Oss       | Yes     | (r ) Yes  | Yes
2 | nvidia                    | NVIDIA Repository             | No      | (  ) Yes  | Yes`

	repos, err := parseZypperRepos(output)
	require.NoError(t, err)
	require.Len(t, repos, 2)

	assert.Equal(t, "openSUSE-Tumbleweed-Oss", repos[0].Name)
	assert.True(t, repos[0].Enabled)

	assert.Equal(t, "nvidia", repos[1].Name)
	assert.False(t, repos[1].Enabled)
}

func TestParseZypperRepos_Empty(t *testing.T) {
	repos, err := parseZypperRepos("")
	require.NoError(t, err)
	assert.Len(t, repos, 0)
}

func TestParseZypperListUpdates(t *testing.T) {
	output := `Loading repository data...
Reading installed packages...
S | Repository                | Name       | Current Version | Available Version | Arch
--+---------------------------+------------+-----------------+-------------------+--------
v | openSUSE-Tumbleweed-Oss   | nginx      | 1.22.0-1.1      | 1.24.0-1.1       | x86_64
v | openSUSE-Tumbleweed-Oss   | curl       | 8.0.0-1.1       | 8.1.0-1.1        | x86_64`

	packages, err := parseZypperListUpdates(output)
	require.NoError(t, err)
	require.Len(t, packages, 2)

	assert.Equal(t, "nginx", packages[0].Name)
	assert.Equal(t, "1.24.0-1.1", packages[0].Version)
	assert.Equal(t, "x86_64", packages[0].Architecture)
	assert.Equal(t, "openSUSE-Tumbleweed-Oss", packages[0].Repository)
	assert.True(t, packages[0].Installed)
}

func TestParseZypperListUpdates_Empty(t *testing.T) {
	packages, err := parseZypperListUpdates("No updates found.")
	require.NoError(t, err)
	assert.Len(t, packages, 0)
}

func TestParseUnneededPackages(t *testing.T) {
	output := `Loading repository data...
Reading installed packages...
S | Repository | Name              | Version       | Arch
--+------------+-------------------+---------------+--------
i | @System    | orphan-package    | 1.0.0-1.1     | x86_64
i | @System    | unused-lib        | 2.0.0-1.1     | x86_64`

	packages := parseUnneededPackages(output)
	require.Len(t, packages, 2)
	assert.Equal(t, "orphan-package", packages[0])
	assert.Equal(t, "unused-lib", packages[1])
}

func TestParseUnneededPackages_Empty(t *testing.T) {
	packages := parseUnneededPackages("No packages found.")
	assert.Len(t, packages, 0)
}

func TestParseRepoFile(t *testing.T) {
	content := `[nvidia]
name=NVIDIA Repository
baseurl=https://download.nvidia.com/opensuse/tumbleweed
enabled=1
gpgcheck=1
gpgkey=https://download.nvidia.com/opensuse/tumbleweed/repodata/repomd.xml.key

[nvidia-source]
name=NVIDIA Source Repository
baseurl=https://download.nvidia.com/opensuse/tumbleweed/SRPMS
enabled=0
gpgcheck=1`

	repos, err := parseRepoFile(content)
	require.NoError(t, err)
	require.Len(t, repos, 2)

	assert.Equal(t, "nvidia", repos[0].Name)
	assert.Equal(t, "https://download.nvidia.com/opensuse/tumbleweed", repos[0].URL)
	assert.True(t, repos[0].Enabled)
	assert.Equal(t, "https://download.nvidia.com/opensuse/tumbleweed/repodata/repomd.xml.key", repos[0].GPGKey)

	assert.Equal(t, "nvidia-source", repos[1].Name)
	assert.False(t, repos[1].Enabled)
}

func TestBuildRepoFileContent(t *testing.T) {
	repo := pkg.Repository{
		Name:    "nvidia",
		URL:     "https://download.nvidia.com/opensuse/tumbleweed",
		Enabled: true,
		GPGKey:  "https://download.nvidia.com/opensuse/tumbleweed/repodata/repomd.xml.key",
	}

	content := buildRepoFileContent(repo)

	assert.Contains(t, content, "[nvidia]")
	assert.Contains(t, content, "name=nvidia")
	assert.Contains(t, content, "baseurl=https://download.nvidia.com/opensuse/tumbleweed")
	assert.Contains(t, content, "enabled=1")
	assert.Contains(t, content, "gpgcheck=1")
	assert.Contains(t, content, "gpgkey=https://download.nvidia.com/opensuse/tumbleweed/repodata/repomd.xml.key")
	assert.Contains(t, content, "autorefresh=1")
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
		{"nvidia/opensuse", "nvidia-opensuse"},
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
		{"1 MiB", 1024 * 1024},
		{"1.2 MiB", 1258291}, // approximately 1.2 * 1024 * 1024
		{"1 G", 1024 * 1024 * 1024},
		{"1 GiB", 1024 * 1024 * 1024},
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
	assert.True(t, isArchitecture("i586"))
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

	mockExec.SetResponse("zypper", exec.SuccessResult(""))

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// The mock doesn't actually check context, but this tests the pattern
	err := mgr.Install(ctx, pkg.DefaultInstallOptions(), "nginx")
	// With mock, it succeeds because mock doesn't check context
	require.NoError(t, err)
}

// Edge case tests

func TestManager_Install_NothingToDo(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("zypper", exec.FailureResult(1, "Nothing to do."))

	err := mgr.Install(context.Background(), pkg.DefaultInstallOptions(), "already-installed")
	require.Error(t, err)
	assert.ErrorIs(t, err, pkg.ErrPackageNotFound)
}

func TestManager_Remove_GenericError(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("zypper", exec.FailureResult(1, "Some generic error"))

	err := mgr.Remove(context.Background(), pkg.DefaultRemoveOptions(), "pkg")
	require.Error(t, err)
	assert.ErrorIs(t, err, pkg.ErrRemoveFailed)
}

func TestManager_Remove_LockError(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("zypper", exec.FailureResult(1, "another zypper is running"))

	err := mgr.Remove(context.Background(), pkg.DefaultRemoveOptions(), "nginx")
	require.Error(t, err)
	assert.ErrorIs(t, err, pkg.ErrLockAcquireFailed)
}

func TestManager_EnableRepository_GenericError(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("zypper", exec.FailureResult(1, "Some error"))

	err := mgr.EnableRepository(context.Background(), "repo")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "zypper modifyrepo --enable failed")
}

func TestManager_DisableRepository_GenericError(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("zypper", exec.FailureResult(1, "Some error"))

	err := mgr.DisableRepository(context.Background(), "repo")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "zypper modifyrepo --disable failed")
}

func TestManager_RefreshRepositories_GenericError(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("zypper", exec.FailureResult(1, "Some error"))

	err := mgr.RefreshRepositories(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "zypper refresh failed")
}

func TestManager_ImportGPGKey_GenericError(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("rpm", exec.FailureResult(1, "Import failed"))

	err := mgr.ImportGPGKey(context.Background(), "https://example.com/key")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "rpm --import failed")
}

func TestManager_SetRepositoryPriority_NotFound(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("zypper", exec.FailureResult(1, "Repository 'nonexistent' not found"))

	err := mgr.SetRepositoryPriority(context.Background(), "nonexistent", 90)
	require.Error(t, err)
	assert.ErrorIs(t, err, pkg.ErrRepositoryNotFound)
}

func TestManager_SetRepositoryRefresh_NotFound(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("zypper", exec.FailureResult(1, "No repository"))

	err := mgr.SetRepositoryRefresh(context.Background(), "nonexistent", true)
	require.Error(t, err)
	assert.ErrorIs(t, err, pkg.ErrRepositoryNotFound)
}
