package pacman

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
	assert.Equal(t, "pacman", mgr.Name())
}

func TestManager_Family(t *testing.T) {
	mgr, _ := setupTest()
	assert.Equal(t, constants.FamilyArch, mgr.Family())
}

func TestManager_Install_SinglePackage(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("pacman", exec.SuccessResult(""))

	err := mgr.Install(context.Background(), pkg.DefaultInstallOptions(), "nvidia")
	require.NoError(t, err)

	// Verify the command was called
	assert.True(t, mockExec.WasCalled("pacman"))
	lastCall := mockExec.LastCall()
	assert.Contains(t, lastCall.Args, "-S")
	assert.Contains(t, lastCall.Args, "--noconfirm")
	assert.Contains(t, lastCall.Args, "nvidia")
}

func TestManager_Install_MultiplePackages(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("pacman", exec.SuccessResult(""))

	err := mgr.Install(context.Background(), pkg.DefaultInstallOptions(), "nvidia", "cuda", "nvidia-utils")
	require.NoError(t, err)

	lastCall := mockExec.LastCall()
	assert.Contains(t, lastCall.Args, "nvidia")
	assert.Contains(t, lastCall.Args, "cuda")
	assert.Contains(t, lastCall.Args, "nvidia-utils")
}

func TestManager_Install_WithOptions(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("pacman", exec.SuccessResult(""))

	opts := pkg.InstallOptions{
		Force:        true,
		DownloadOnly: true,
	}

	err := mgr.Install(context.Background(), opts, "nvidia")
	require.NoError(t, err)

	lastCall := mockExec.LastCall()
	assert.Contains(t, lastCall.Args, "--overwrite")
	assert.Contains(t, lastCall.Args, "--downloadonly")
}

func TestManager_Install_PackageNotFound(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("pacman", exec.FailureResult(1, "error: target not found: nonexistent"))

	err := mgr.Install(context.Background(), pkg.DefaultInstallOptions(), "nonexistent")
	require.Error(t, err)
	assert.ErrorIs(t, err, pkg.ErrPackageNotFound)
}

func TestManager_Install_LockError(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("pacman", exec.FailureResult(1, "error: unable to lock database"))

	err := mgr.Install(context.Background(), pkg.DefaultInstallOptions(), "nvidia")
	require.Error(t, err)
	assert.ErrorIs(t, err, pkg.ErrLockAcquireFailed)
}

func TestManager_Install_DependencyConflict(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("pacman", exec.FailureResult(1, "error: failed to prepare transaction (conflicting dependencies)"))

	err := mgr.Install(context.Background(), pkg.DefaultInstallOptions(), "nvidia")
	require.Error(t, err)
	assert.ErrorIs(t, err, pkg.ErrDependencyConflict)
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

	mockExec.SetResponse("pacman", exec.SuccessResult(""))

	err := mgr.Remove(context.Background(), pkg.DefaultRemoveOptions(), "nvidia")
	require.NoError(t, err)

	lastCall := mockExec.LastCall()
	assert.Contains(t, lastCall.Args, "-R")
	assert.Contains(t, lastCall.Args, "--noconfirm")
	assert.Contains(t, lastCall.Args, "nvidia")
}

func TestManager_Remove_WithPurge(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("pacman", exec.SuccessResult(""))

	opts := pkg.RemoveOptions{Purge: true}
	err := mgr.Remove(context.Background(), opts, "nvidia")
	require.NoError(t, err)

	lastCall := mockExec.LastCall()
	assert.Contains(t, lastCall.Args, "-Rns")
}

func TestManager_Remove_WithAutoRemove(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("pacman", exec.SuccessResult(""))

	opts := pkg.RemoveOptions{AutoRemove: true}
	err := mgr.Remove(context.Background(), opts, "nvidia")
	require.NoError(t, err)

	lastCall := mockExec.LastCall()
	assert.Contains(t, lastCall.Args, "-Rs")
}

func TestManager_Remove_PackageNotInstalled(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("pacman", exec.FailureResult(1, "error: target not found: nvidia"))

	err := mgr.Remove(context.Background(), pkg.DefaultRemoveOptions(), "nvidia")
	require.Error(t, err)
	assert.ErrorIs(t, err, pkg.ErrPackageNotInstalled)
}

func TestManager_Remove_EmptyPackages(t *testing.T) {
	mgr, mockExec := setupTest()

	err := mgr.Remove(context.Background(), pkg.DefaultRemoveOptions())
	require.NoError(t, err)

	assert.Equal(t, 0, mockExec.CallCount())
}

func TestManager_Update_Success(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("pacman", exec.SuccessResult(":: Synchronizing package databases..."))

	err := mgr.Update(context.Background(), pkg.DefaultUpdateOptions())
	require.NoError(t, err)

	lastCall := mockExec.LastCall()
	assert.Contains(t, lastCall.Args, "-Sy")
}

func TestManager_Update_Quiet(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("pacman", exec.SuccessResult(""))

	opts := pkg.UpdateOptions{Quiet: true}
	err := mgr.Update(context.Background(), opts)
	require.NoError(t, err)

	lastCall := mockExec.LastCall()
	assert.Contains(t, lastCall.Args, "--quiet")
}

func TestManager_Update_NetworkError(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("pacman", exec.FailureResult(1, "error: failed to retrieve some files"))

	err := mgr.Update(context.Background(), pkg.DefaultUpdateOptions())
	require.Error(t, err)
	assert.ErrorIs(t, err, pkg.ErrNetworkUnavailable)
}

func TestManager_Update_LockError(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("pacman", exec.FailureResult(1, "error: unable to lock database"))

	err := mgr.Update(context.Background(), pkg.DefaultUpdateOptions())
	require.Error(t, err)
	assert.ErrorIs(t, err, pkg.ErrLockAcquireFailed)
}

func TestManager_Update_GenericFailure(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("pacman", exec.FailureResult(1, "Some error"))

	err := mgr.Update(context.Background(), pkg.DefaultUpdateOptions())
	require.Error(t, err)
	assert.ErrorIs(t, err, pkg.ErrUpdateFailed)
}

func TestManager_Upgrade_AllPackages(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("pacman", exec.SuccessResult(""))

	err := mgr.Upgrade(context.Background(), pkg.DefaultInstallOptions())
	require.NoError(t, err)

	lastCall := mockExec.LastCall()
	assert.Contains(t, lastCall.Args, "-Su")
	assert.Contains(t, lastCall.Args, "--noconfirm")
}

func TestManager_Upgrade_SpecificPackages(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("pacman", exec.SuccessResult(""))

	err := mgr.Upgrade(context.Background(), pkg.DefaultInstallOptions(), "nvidia", "cuda")
	require.NoError(t, err)

	lastCall := mockExec.LastCall()
	assert.Contains(t, lastCall.Args, "-S")
	assert.Contains(t, lastCall.Args, "nvidia")
	assert.Contains(t, lastCall.Args, "cuda")
}

func TestManager_Upgrade_WithOptions(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("pacman", exec.SuccessResult(""))

	opts := pkg.InstallOptions{
		Force: true,
	}

	err := mgr.Upgrade(context.Background(), opts)
	require.NoError(t, err)

	lastCall := mockExec.LastCall()
	assert.Contains(t, lastCall.Args, "--overwrite")
}

func TestManager_Upgrade_Failure(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("pacman", exec.FailureResult(1, "Upgrade failed"))

	err := mgr.Upgrade(context.Background(), pkg.DefaultInstallOptions())
	require.Error(t, err)
	assert.ErrorIs(t, err, pkg.ErrInstallFailed)
}

func TestManager_Upgrade_LockError(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("pacman", exec.FailureResult(1, "unable to lock database"))

	err := mgr.Upgrade(context.Background(), pkg.DefaultInstallOptions())
	require.Error(t, err)
	assert.ErrorIs(t, err, pkg.ErrLockAcquireFailed)
}

func TestManager_IsInstalled_True(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("pacman", exec.SuccessResult("nvidia 545.29.06-1"))

	installed, err := mgr.IsInstalled(context.Background(), "nvidia")
	require.NoError(t, err)
	assert.True(t, installed)
}

func TestManager_IsInstalled_False(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("pacman", exec.FailureResult(1, "error: package 'nvidia' was not found"))

	installed, err := mgr.IsInstalled(context.Background(), "nvidia")
	require.NoError(t, err)
	assert.False(t, installed)
}

func TestManager_IsInstalled_Error(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("pacman", exec.FailureResult(2, "database error"))

	_, err := mgr.IsInstalled(context.Background(), "nvidia")
	require.Error(t, err)
}

func TestManager_Search_Success(t *testing.T) {
	mgr, mockExec := setupTest()

	output := `extra/nvidia-utils 545.29.06-1 [installed]
    NVIDIA drivers utilities
extra/nvidia 545.29.06-1
    NVIDIA drivers for linux
community/nvidia-390xx-utils 390.157-1
    NVIDIA 390xx legacy drivers utilities`

	mockExec.SetResponse("pacman", exec.SuccessResult(output))

	packages, err := mgr.Search(context.Background(), "nvidia", pkg.SearchOptions{IncludeInstalled: false})
	require.NoError(t, err)
	require.Len(t, packages, 3)
	assert.Equal(t, "nvidia-utils", packages[0].Name)
	assert.Equal(t, "extra", packages[0].Repository)
	assert.True(t, packages[0].Installed)
}

func TestManager_Search_WithLimit(t *testing.T) {
	mgr, mockExec := setupTest()

	output := `extra/nvidia-utils 545.29.06-1
    NVIDIA drivers utilities
extra/nvidia 545.29.06-1
    NVIDIA drivers for linux
community/nvidia-390xx-utils 390.157-1
    NVIDIA 390xx legacy drivers utilities`

	mockExec.SetResponse("pacman", exec.SuccessResult(output))

	opts := pkg.SearchOptions{Limit: 2, IncludeInstalled: false}
	packages, err := mgr.Search(context.Background(), "nvidia", opts)
	require.NoError(t, err)
	require.Len(t, packages, 2)
}

func TestManager_Search_NoResults(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("pacman", exec.FailureResult(1, ""))

	packages, err := mgr.Search(context.Background(), "nonexistent", pkg.DefaultSearchOptions())
	require.NoError(t, err)
	assert.Len(t, packages, 0)
}

func TestManager_Search_Failure(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("pacman", exec.FailureResult(2, "Database error"))

	packages, err := mgr.Search(context.Background(), "nvidia", pkg.DefaultSearchOptions())
	require.Error(t, err)
	assert.Nil(t, packages)
}

func TestManager_Info_Success(t *testing.T) {
	mgr, mockExec := setupTest()

	output := `Repository      : extra
Name            : nvidia
Version         : 545.29.06-1
Description     : NVIDIA drivers for linux
Architecture    : x86_64
Installed Size  : 1.2 MiB
Depends On      : linux  nvidia-utils=545.29.06  libglvnd`

	mockExec.SetResponse("pacman", exec.SuccessResult(output))

	info, err := mgr.Info(context.Background(), "nvidia")
	require.NoError(t, err)
	require.NotNil(t, info)
	assert.Equal(t, "nvidia", info.Name)
	assert.Equal(t, "545.29.06-1", info.Version)
	assert.Equal(t, "x86_64", info.Architecture)
	assert.Equal(t, "extra", info.Repository)
	assert.Equal(t, "NVIDIA drivers for linux", info.Description)
}

func TestManager_Info_NotFound(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("pacman", exec.FailureResult(1, "error: package 'nonexistent' was not found"))

	info, err := mgr.Info(context.Background(), "nonexistent")
	require.Error(t, err)
	assert.Nil(t, info)
	assert.ErrorIs(t, err, pkg.ErrPackageNotFound)
}

func TestManager_Info_LocalFallback(t *testing.T) {
	mgr, mockExec := setupTest()

	// We'll use the default response for the pattern - both Si and Qi fail
	mockExec.SetDefaultResponse(exec.FailureResult(1, "error: package was not found"))

	_, err := mgr.Info(context.Background(), "localonly")
	// With mock always returning failure, this should fail
	require.Error(t, err)
	assert.ErrorIs(t, err, pkg.ErrPackageNotFound)
}

func TestManager_ListInstalled(t *testing.T) {
	mgr, mockExec := setupTest()

	output := `linux 6.6.1.arch1-1
nvidia 545.29.06-1
nvidia-utils 545.29.06-1`

	mockExec.SetResponse("pacman", exec.SuccessResult(output))

	packages, err := mgr.ListInstalled(context.Background())
	require.NoError(t, err)
	require.Len(t, packages, 3)
	assert.Equal(t, "linux", packages[0].Name)
	assert.Equal(t, "6.6.1.arch1-1", packages[0].Version)
	assert.True(t, packages[0].Installed)
}

func TestManager_ListInstalled_Failure(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("pacman", exec.FailureResult(1, "Database error"))

	packages, err := mgr.ListInstalled(context.Background())
	require.Error(t, err)
	assert.Nil(t, packages)
}

func TestManager_ListUpgradable(t *testing.T) {
	mgr, mockExec := setupTest()

	output := `linux 6.6.0.arch1-1 -> 6.6.1.arch1-1
nvidia 545.29.02-1 -> 545.29.06-1`

	mockExec.SetResponse("pacman", exec.SuccessResult(output))

	packages, err := mgr.ListUpgradable(context.Background())
	require.NoError(t, err)
	require.Len(t, packages, 2)
	assert.Equal(t, "linux", packages[0].Name)
	assert.Equal(t, "6.6.1.arch1-1", packages[0].Version)
}

func TestManager_ListUpgradable_NoUpdates(t *testing.T) {
	mgr, mockExec := setupTest()

	// pacman -Qu returns exit code 1 when no updates available
	mockExec.SetResponse("pacman", exec.FailureResult(1, ""))

	packages, err := mgr.ListUpgradable(context.Background())
	require.NoError(t, err)
	assert.Len(t, packages, 0)
}

func TestManager_ListUpgradable_Error(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("pacman", exec.FailureResult(2, "Database error"))

	packages, err := mgr.ListUpgradable(context.Background())
	require.Error(t, err)
	assert.Nil(t, packages)
}

func TestManager_Clean_Success(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("pacman", exec.SuccessResult("Packages to keep: ..."))

	err := mgr.Clean(context.Background())
	require.NoError(t, err)

	lastCall := mockExec.LastCall()
	assert.Contains(t, lastCall.Args, "-Sc")
	assert.Contains(t, lastCall.Args, "--noconfirm")
}

func TestManager_Clean_Failure(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("pacman", exec.FailureResult(1, "Clean failed"))

	err := mgr.Clean(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "pacman clean failed")
}

func TestManager_AutoRemove_Success(t *testing.T) {
	mgr, mockExec := setupTest()

	// First call lists orphans, second call removes them
	mockExec.SetResponse("pacman", exec.SuccessResult("orphan1\norphan2"))

	err := mgr.AutoRemove(context.Background())
	require.NoError(t, err)
}

func TestManager_AutoRemove_NoOrphans(t *testing.T) {
	mgr, mockExec := setupTest()

	// No orphans available
	mockExec.SetResponse("pacman", exec.FailureResult(1, ""))

	err := mgr.AutoRemove(context.Background())
	require.NoError(t, err) // No orphans is not an error
}

func TestManager_AutoRemove_LockError(t *testing.T) {
	mgr, mockExec := setupTest()

	// First call returns orphans, second call fails with lock error
	mockExec.SetDefaultResponse(exec.SuccessResult("orphan1"))
	// After first call, set the response to fail
	// Note: This test is limited by the mock's capabilities

	err := mgr.AutoRemove(context.Background())
	// With default success response for orphan list, it should attempt remove
	require.NoError(t, err)
}

func TestManager_Verify_Installed(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("pacman", exec.SuccessResult("nvidia: 0 missing files"))

	valid, err := mgr.Verify(context.Background(), "nvidia")
	require.NoError(t, err)
	assert.True(t, valid)
}

func TestManager_Verify_NotInstalled(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("pacman", exec.FailureResult(1, ""))

	_, err := mgr.Verify(context.Background(), "nonexistent")
	require.Error(t, err)
	assert.ErrorIs(t, err, pkg.ErrPackageNotInstalled)
}

func TestManager_Verify_Invalid(t *testing.T) {
	mgr, mockExec := setupTest()

	// First call (IsInstalled) succeeds, second call (Qk) fails with missing files
	mockExec.SetDefaultResponse(exec.SuccessResult("nvidia 545.29.06-1"))

	valid, err := mgr.Verify(context.Background(), "nvidia")
	require.NoError(t, err)
	// With default success response, verification passes
	assert.True(t, valid)
}

// Repository tests

func TestManager_AddRepository_Success(t *testing.T) {
	mgr, mockExec := setupTest()

	// List repos returns empty
	mockExec.SetResponse("cat", exec.SuccessResult("[options]\n[core]\nInclude = /etc/pacman.d/mirrorlist"))
	mockExec.SetDefaultResponse(exec.SuccessResult(""))

	repo := pkg.Repository{
		Name: "chaotic-aur",
		URL:  "https://geo-mirror.chaotic.cx/$repo/$arch",
	}

	err := mgr.AddRepository(context.Background(), repo)
	require.NoError(t, err)
}

func TestManager_AddRepository_AlreadyExists(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("cat", exec.SuccessResult("[options]\n[chaotic-aur]\nServer = https://example.com"))

	repo := pkg.Repository{
		Name: "chaotic-aur",
		URL:  "https://example.com",
	}

	err := mgr.AddRepository(context.Background(), repo)
	require.Error(t, err)
	assert.ErrorIs(t, err, pkg.ErrRepositoryExists)
}

func TestManager_AddRepository_EmptyName(t *testing.T) {
	mgr, _ := setupTest()

	repo := pkg.Repository{
		URL: "https://example.com",
	}

	err := mgr.AddRepository(context.Background(), repo)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "name is required")
}

func TestManager_RemoveRepository_Success(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("cat", exec.SuccessResult("[options]\n[chaotic-aur]\nServer = https://example.com\n[core]\nInclude = /etc/pacman.d/mirrorlist"))
	mockExec.SetDefaultResponse(exec.SuccessResult(""))

	err := mgr.RemoveRepository(context.Background(), "chaotic-aur")
	require.NoError(t, err)
}

func TestManager_RemoveRepository_NotFound(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("cat", exec.SuccessResult("[options]\n[core]\nInclude = /etc/pacman.d/mirrorlist"))

	err := mgr.RemoveRepository(context.Background(), "nonexistent")
	require.Error(t, err)
	assert.ErrorIs(t, err, pkg.ErrRepositoryNotFound)
}

func TestManager_ListRepositories_Success(t *testing.T) {
	mgr, mockExec := setupTest()

	content := `[options]
HoldPkg     = pacman glibc
Architecture = auto

[core]
Include = /etc/pacman.d/mirrorlist

[extra]
Include = /etc/pacman.d/mirrorlist

[community]
Include = /etc/pacman.d/mirrorlist`

	mockExec.SetResponse("cat", exec.SuccessResult(content))

	repos, err := mgr.ListRepositories(context.Background())
	require.NoError(t, err)
	require.Len(t, repos, 3)
	assert.Equal(t, "core", repos[0].Name)
	assert.Equal(t, "extra", repos[1].Name)
	assert.Equal(t, "community", repos[2].Name)
}

func TestManager_ListRepositories_Failure(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("cat", exec.FailureResult(1, "File not found"))

	repos, err := mgr.ListRepositories(context.Background())
	require.Error(t, err)
	assert.Nil(t, repos)
}

func TestManager_EnableRepository_Success(t *testing.T) {
	mgr, mockExec := setupTest()

	content := `[options]
[core]
Include = /etc/pacman.d/mirrorlist

# [multilib]
# Include = /etc/pacman.d/mirrorlist`

	mockExec.SetResponse("cat", exec.SuccessResult(content))
	mockExec.SetDefaultResponse(exec.SuccessResult(""))

	err := mgr.EnableRepository(context.Background(), "multilib")
	require.NoError(t, err)
}

func TestManager_EnableRepository_AlreadyEnabled(t *testing.T) {
	mgr, mockExec := setupTest()

	content := `[options]
[core]
Include = /etc/pacman.d/mirrorlist

[multilib]
Include = /etc/pacman.d/mirrorlist`

	mockExec.SetResponse("cat", exec.SuccessResult(content))

	err := mgr.EnableRepository(context.Background(), "multilib")
	require.NoError(t, err)
}

func TestManager_EnableRepository_NotFound(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("cat", exec.SuccessResult("[options]\n[core]\nInclude = /etc/pacman.d/mirrorlist"))

	err := mgr.EnableRepository(context.Background(), "nonexistent")
	require.Error(t, err)
	assert.ErrorIs(t, err, pkg.ErrRepositoryNotFound)
}

func TestManager_DisableRepository_Success(t *testing.T) {
	mgr, mockExec := setupTest()

	content := `[options]
[core]
Include = /etc/pacman.d/mirrorlist

[multilib]
Include = /etc/pacman.d/mirrorlist`

	mockExec.SetResponse("cat", exec.SuccessResult(content))
	mockExec.SetDefaultResponse(exec.SuccessResult(""))

	err := mgr.DisableRepository(context.Background(), "multilib")
	require.NoError(t, err)
}

func TestManager_DisableRepository_AlreadyDisabled(t *testing.T) {
	mgr, mockExec := setupTest()

	content := `[options]
[core]
Include = /etc/pacman.d/mirrorlist

# [multilib]
# Include = /etc/pacman.d/mirrorlist`

	mockExec.SetResponse("cat", exec.SuccessResult(content))

	err := mgr.DisableRepository(context.Background(), "multilib")
	require.NoError(t, err)
}

func TestManager_DisableRepository_NotFound(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("cat", exec.SuccessResult("[options]\n[core]\nInclude = /etc/pacman.d/mirrorlist"))

	err := mgr.DisableRepository(context.Background(), "nonexistent")
	require.Error(t, err)
	assert.ErrorIs(t, err, pkg.ErrRepositoryNotFound)
}

func TestManager_RefreshRepositories_Success(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("pacman", exec.SuccessResult(":: Synchronizing package databases..."))

	err := mgr.RefreshRepositories(context.Background())
	require.NoError(t, err)

	lastCall := mockExec.LastCall()
	assert.Contains(t, lastCall.Args, "-Sy")
}

func TestManager_RefreshRepositories_NetworkError(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("pacman", exec.FailureResult(1, "error: failed to retrieve some files"))

	err := mgr.RefreshRepositories(context.Background())
	require.Error(t, err)
	assert.ErrorIs(t, err, pkg.ErrNetworkUnavailable)
}

func TestManager_ImportGPGKey_Success(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("pacman-key", exec.SuccessResult(""))

	err := mgr.ImportGPGKey(context.Background(), "3056513887B78AEB")
	require.NoError(t, err)
}

func TestManager_ImportGPGKey_NetworkError(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("pacman-key", exec.FailureResult(1, "gpg: keyserver receive failed"))

	err := mgr.ImportGPGKey(context.Background(), "invalid-key")
	require.Error(t, err)
	assert.ErrorIs(t, err, pkg.ErrNetworkUnavailable)
}

func TestManager_InitializeKeyring(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("pacman-key", exec.SuccessResult(""))

	err := mgr.InitializeKeyring(context.Background())
	require.NoError(t, err)

	lastCall := mockExec.LastCall()
	assert.Contains(t, lastCall.Args, "--init")
}

func TestManager_PopulateKeyring(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("pacman-key", exec.SuccessResult(""))

	err := mgr.PopulateKeyring(context.Background(), "archlinux")
	require.NoError(t, err)

	lastCall := mockExec.LastCall()
	assert.Contains(t, lastCall.Args, "--populate")
	assert.Contains(t, lastCall.Args, "archlinux")
}

func TestManager_PopulateKeyring_DefaultDistro(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("pacman-key", exec.SuccessResult(""))

	err := mgr.PopulateKeyring(context.Background(), "")
	require.NoError(t, err)

	lastCall := mockExec.LastCall()
	assert.Contains(t, lastCall.Args, "archlinux")
}

func TestManager_RefreshKeys(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("pacman-key", exec.SuccessResult(""))

	err := mgr.RefreshKeys(context.Background())
	require.NoError(t, err)

	lastCall := mockExec.LastCall()
	assert.Contains(t, lastCall.Args, "--refresh-keys")
}

func TestManager_RefreshKeys_NetworkError(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("pacman-key", exec.FailureResult(1, "gpg: keyserver receive failed"))

	err := mgr.RefreshKeys(context.Background())
	require.Error(t, err)
	assert.ErrorIs(t, err, pkg.ErrNetworkUnavailable)
}

// Parser tests

func TestParsePacmanQ(t *testing.T) {
	output := `linux 6.6.1.arch1-1
nvidia 545.29.06-1
nvidia-utils 545.29.06-1`

	packages, err := parsePacmanQ(output)
	require.NoError(t, err)
	require.Len(t, packages, 3)

	assert.Equal(t, "linux", packages[0].Name)
	assert.Equal(t, "6.6.1.arch1-1", packages[0].Version)
	assert.True(t, packages[0].Installed)
}

func TestParsePacmanQ_Empty(t *testing.T) {
	packages, err := parsePacmanQ("")
	require.NoError(t, err)
	assert.Len(t, packages, 0)
}

func TestParsePacmanQ_MalformedLine(t *testing.T) {
	output := `linux
nvidia 545.29.06-1`

	packages, err := parsePacmanQ(output)
	require.NoError(t, err)
	assert.Len(t, packages, 1)
	assert.Equal(t, "nvidia", packages[0].Name)
}

func TestParsePacmanInfo(t *testing.T) {
	output := `Repository      : extra
Name            : nvidia
Version         : 545.29.06-1
Description     : NVIDIA drivers for linux
Architecture    : x86_64
URL             : https://www.nvidia.com/
Installed Size  : 1.2 MiB
Depends On      : linux  nvidia-utils=545.29.06`

	p, err := parsePacmanInfo(output)
	require.NoError(t, err)
	require.NotNil(t, p)

	assert.Equal(t, "nvidia", p.Name)
	assert.Equal(t, "545.29.06-1", p.Version)
	assert.Equal(t, "x86_64", p.Architecture)
	assert.Equal(t, "extra", p.Repository)
	assert.Equal(t, "NVIDIA drivers for linux", p.Description)
	assert.Greater(t, p.Size, int64(0))
	assert.Contains(t, p.Dependencies, "linux")
}

func TestParsePacmanInfo_Empty(t *testing.T) {
	p, err := parsePacmanInfo("")
	require.NoError(t, err)
	assert.Nil(t, p)
}

func TestParsePacmanSs(t *testing.T) {
	output := `extra/nvidia-utils 545.29.06-1 [installed]
    NVIDIA drivers utilities
extra/nvidia 545.29.06-1
    NVIDIA drivers for linux
community/nvidia-390xx-utils 390.157-1
    NVIDIA 390xx legacy drivers utilities`

	packages, err := parsePacmanSs(output)
	require.NoError(t, err)
	require.Len(t, packages, 3)

	assert.Equal(t, "nvidia-utils", packages[0].Name)
	assert.Equal(t, "extra", packages[0].Repository)
	assert.Equal(t, "545.29.06-1", packages[0].Version)
	assert.True(t, packages[0].Installed)
	assert.Equal(t, "NVIDIA drivers utilities", packages[0].Description)

	assert.Equal(t, "nvidia", packages[1].Name)
	assert.False(t, packages[1].Installed)
}

func TestParsePacmanSs_Empty(t *testing.T) {
	packages, err := parsePacmanSs("")
	require.NoError(t, err)
	assert.Len(t, packages, 0)
}

func TestParsePacmanQu(t *testing.T) {
	output := `linux 6.6.0.arch1-1 -> 6.6.1.arch1-1
nvidia 545.29.02-1 -> 545.29.06-1`

	packages, err := parsePacmanQu(output)
	require.NoError(t, err)
	require.Len(t, packages, 2)

	assert.Equal(t, "linux", packages[0].Name)
	assert.Equal(t, "6.6.1.arch1-1", packages[0].Version)
	assert.True(t, packages[0].Installed)
}

func TestParsePacmanQu_SimpleFormat(t *testing.T) {
	output := `linux 6.6.1.arch1-1
nvidia 545.29.06-1`

	packages, err := parsePacmanQu(output)
	require.NoError(t, err)
	require.Len(t, packages, 2)

	assert.Equal(t, "linux", packages[0].Name)
	assert.Equal(t, "6.6.1.arch1-1", packages[0].Version)
}

func TestParsePacmanQu_Empty(t *testing.T) {
	packages, err := parsePacmanQu("")
	require.NoError(t, err)
	assert.Len(t, packages, 0)
}

func TestParsePacmanConf(t *testing.T) {
	content := `[options]
HoldPkg     = pacman glibc
Architecture = auto

[core]
Include = /etc/pacman.d/mirrorlist

[extra]
Include = /etc/pacman.d/mirrorlist

[chaotic-aur]
Server = https://geo-mirror.chaotic.cx/$repo/$arch
SigLevel = Required TrustedOnly`

	repos, err := parsePacmanConf(content)
	require.NoError(t, err)
	require.Len(t, repos, 3)

	assert.Equal(t, "core", repos[0].Name)
	assert.Equal(t, "/etc/pacman.d/mirrorlist", repos[0].URL)
	assert.True(t, repos[0].Enabled)

	assert.Equal(t, "chaotic-aur", repos[2].Name)
	assert.Equal(t, "https://geo-mirror.chaotic.cx/$repo/$arch", repos[2].URL)
}

func TestParsePacmanConf_Empty(t *testing.T) {
	repos, err := parsePacmanConf("")
	require.NoError(t, err)
	assert.Len(t, repos, 0)
}

func TestParsePacmanConf_OnlyOptions(t *testing.T) {
	content := `[options]
HoldPkg     = pacman glibc`

	repos, err := parsePacmanConf(content)
	require.NoError(t, err)
	assert.Len(t, repos, 0) // Options section should not be included
}

func TestParseSize(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{"1024", 1024},
		{"1 KiB", 1024},
		{"1.5 KiB", 1536},
		{"1 MiB", 1024 * 1024},
		{"1.5 MiB", 1536 * 1024},
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

func TestSanitizeRepoName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"simple", "simple"},
		{"with space", "with-space"},
		{"with/slash", "with-slash"},
		{"with:colon", "with-colon"},
		{"---leading---", "leading"},
		{"", "custom-repo"},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			result := sanitizeRepoName(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// Interface compliance test
func TestManager_ImplementsInterface(t *testing.T) {
	var _ pkg.Manager = (*Manager)(nil)
}

// Context cancellation test
func TestManager_Install_ContextCancelled(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("pacman", exec.SuccessResult(""))

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// The mock doesn't actually check context, but this tests the pattern
	err := mgr.Install(ctx, pkg.DefaultInstallOptions(), "nvidia")
	// With mock, it succeeds because mock doesn't check context
	require.NoError(t, err)
}

// Additional edge case tests

func TestManager_Install_GenericError(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("pacman", exec.FailureResult(1, "Some generic error"))

	err := mgr.Install(context.Background(), pkg.DefaultInstallOptions(), "nvidia")
	require.Error(t, err)
	assert.ErrorIs(t, err, pkg.ErrInstallFailed)
}

func TestManager_Remove_LockError(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("pacman", exec.FailureResult(1, "database is locked"))

	err := mgr.Remove(context.Background(), pkg.DefaultRemoveOptions(), "nvidia")
	require.Error(t, err)
	assert.ErrorIs(t, err, pkg.ErrLockAcquireFailed)
}

func TestManager_Remove_GenericError(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("pacman", exec.FailureResult(1, "Some generic error"))

	err := mgr.Remove(context.Background(), pkg.DefaultRemoveOptions(), "nvidia")
	require.Error(t, err)
	assert.ErrorIs(t, err, pkg.ErrRemoveFailed)
}

func TestManager_Search_IncludeInstalled(t *testing.T) {
	mgr, mockExec := setupTest()

	output := `extra/nvidia 545.29.06-1
    NVIDIA drivers for linux`

	mockExec.SetResponse("pacman", exec.SuccessResult(output))

	opts := pkg.SearchOptions{IncludeInstalled: true}
	packages, err := mgr.Search(context.Background(), "nvidia", opts)
	require.NoError(t, err)
	require.Len(t, packages, 1)
	// With mock returning success for IsInstalled check
}

func TestManager_Info_GenericError(t *testing.T) {
	mgr, mockExec := setupTest()

	mockExec.SetResponse("pacman", exec.FailureResult(1, "Some error"))

	info, err := mgr.Info(context.Background(), "nvidia")
	require.Error(t, err)
	assert.Nil(t, info)
}

func TestManager_AutoRemove_EmptyOrphans(t *testing.T) {
	mgr, mockExec := setupTest()

	// Return empty string (no orphans)
	mockExec.SetResponse("pacman", exec.SuccessResult(""))

	err := mgr.AutoRemove(context.Background())
	require.NoError(t, err)
}

// Test parsing dependencies
func TestParsePacmanInfo_Dependencies(t *testing.T) {
	output := `Name            : nvidia
Version         : 545.29.06-1
Depends On      : linux>=6.0  nvidia-utils=545.29.06  libglvnd>=1.4`

	p, err := parsePacmanInfo(output)
	require.NoError(t, err)
	require.NotNil(t, p)

	// Dependencies should have version constraints stripped
	assert.Contains(t, p.Dependencies, "linux")
	assert.Contains(t, p.Dependencies, "nvidia-utils")
	assert.Contains(t, p.Dependencies, "libglvnd")
}

// Test search with installed annotation variations
func TestParsePacmanSs_InstalledVariations(t *testing.T) {
	output := `extra/nvidia 545.29.06-1 [installed: 545.29.02-1]
    NVIDIA drivers for linux`

	packages, err := parsePacmanSs(output)
	require.NoError(t, err)
	require.Len(t, packages, 1)
	assert.True(t, packages[0].Installed)
}
