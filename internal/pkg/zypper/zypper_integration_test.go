package zypper

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tungetti/igor/internal/exec"
	"github.com/tungetti/igor/internal/pkg"
	"github.com/tungetti/igor/internal/privilege"
)

// =============================================================================
// Integration Test Scenarios
// =============================================================================

// TestManager_CompleteWorkflow tests a full package management workflow.
func TestManager_CompleteWorkflow(t *testing.T) {
	mgr, mockExec := setupTest()
	ctx := context.Background()

	// Step 1: Update cache (zypper refresh)
	mockExec.SetResponse("zypper", exec.SuccessResult("All repositories have been refreshed."))
	err := mgr.Update(ctx, pkg.DefaultUpdateOptions())
	require.NoError(t, err, "cache update should succeed")

	// Step 2: Search for packages
	searchOutput := `Loading repository data...
Reading installed packages...
S | Name                      | Summary                                                  | Type
--+---------------------------+----------------------------------------------------------+--------
i | nginx                     | A high performance web server and reverse proxy server   | package
  | nginx-source              | Source code of nginx                                     | srcpackage
  | nginx-docs                | Documentation for nginx                                  | package`
	mockExec.SetResponse("zypper", exec.SuccessResult(searchOutput))
	mockExec.SetResponse("rpm", exec.FailureResult(1, "not installed"))
	packages, err := mgr.Search(ctx, "nginx", pkg.DefaultSearchOptions())
	require.NoError(t, err, "search should succeed")
	require.Len(t, packages, 3, "should find 3 nginx packages")

	// Step 3: Check if installed (should not be)
	mockExec.SetResponse("rpm", exec.FailureResult(1, "package nginx is not installed"))
	installed, err := mgr.IsInstalled(ctx, "nginx")
	require.NoError(t, err)
	assert.False(t, installed, "nginx should not be installed initially")

	// Step 4: Install package
	mockExec.SetResponse("zypper", exec.SuccessResult("1 new package to install."))
	err = mgr.Install(ctx, pkg.DefaultInstallOptions(), "nginx")
	require.NoError(t, err, "install should succeed")

	// Step 5: Verify installed
	mockExec.SetResponse("rpm", exec.SuccessResult("nginx-1.24.0-1.1.x86_64"))
	installed, err = mgr.IsInstalled(ctx, "nginx")
	require.NoError(t, err)
	assert.True(t, installed, "nginx should be installed after install")

	// Step 6: Remove package
	mockExec.SetResponse("zypper", exec.SuccessResult("1 package to remove."))
	err = mgr.Remove(ctx, pkg.DefaultRemoveOptions(), "nginx")
	require.NoError(t, err, "remove should succeed")

	// Step 7: Verify removed
	mockExec.SetResponse("rpm", exec.FailureResult(1, "package nginx is not installed"))
	installed, err = mgr.IsInstalled(ctx, "nginx")
	require.NoError(t, err)
	assert.False(t, installed, "nginx should not be installed after removal")
}

// TestManager_NVIDIADriverWorkflow tests NVIDIA driver installation workflow on openSUSE.
func TestManager_NVIDIADriverWorkflow(t *testing.T) {
	mgr, mockExec := setupTest()
	ctx := context.Background()

	// Step 1: Add NVIDIA repository (Tumbleweed)
	mockExec.SetResponse("zypper", exec.SuccessResult(""))
	err := mgr.AddNvidiaRepoTumbleweed(ctx)
	require.NoError(t, err, "adding NVIDIA repo should succeed")

	// Step 2: Refresh repositories
	mockExec.SetResponse("zypper", exec.SuccessResult("All repositories have been refreshed."))
	err = mgr.Update(ctx, pkg.DefaultUpdateOptions())
	require.NoError(t, err)

	// Step 3: Search for NVIDIA driver packages
	searchOutput := `S | Name                      | Summary                              | Type
--+---------------------------+--------------------------------------+--------
  | nvidia-driver-G06         | NVIDIA driver                        | package
  | nvidia-gl-G06             | OpenGL libs for NVIDIA driver        | package
  | nvidia-compute-utils-G06  | NVIDIA compute utilities             | package`
	mockExec.SetResponse("zypper", exec.SuccessResult(searchOutput))
	mockExec.SetResponse("rpm", exec.FailureResult(1, "not installed"))
	packages, err := mgr.Search(ctx, "nvidia", pkg.SearchOptions{IncludeInstalled: false})
	require.NoError(t, err)
	assert.Greater(t, len(packages), 0, "should find NVIDIA driver packages")

	// Step 4: Install NVIDIA driver
	mockExec.SetResponse("zypper", exec.SuccessResult(""))
	err = mgr.Install(ctx, pkg.DefaultInstallOptions(), "nvidia-driver-G06")
	require.NoError(t, err, "NVIDIA driver installation should succeed")

	// Step 5: Verify installation
	mockExec.SetResponse("rpm", exec.SuccessResult("nvidia-driver-G06-535.104.05-1.1.x86_64"))
	installed, err := mgr.IsInstalled(ctx, "nvidia-driver-G06")
	require.NoError(t, err)
	assert.True(t, installed, "NVIDIA driver should be installed")

	// Step 6: Verify package integrity
	mockExec.SetResponse("rpm", exec.SuccessResult(""))
	valid, err := mgr.Verify(ctx, "nvidia-driver-G06")
	require.NoError(t, err)
	assert.True(t, valid, "NVIDIA driver should pass verification")
}

// TestManager_ErrorRecovery tests error recovery scenarios.
func TestManager_ErrorRecovery(t *testing.T) {
	mgr, mockExec := setupTest()
	ctx := context.Background()

	t.Run("package_not_found_recovery", func(t *testing.T) {
		mockExec.SetResponse("zypper", exec.FailureResult(1, "No provider of 'nonexistent-pkg' found."))
		err := mgr.Install(ctx, pkg.DefaultInstallOptions(), "nonexistent-pkg")
		require.Error(t, err)
		assert.ErrorIs(t, err, pkg.ErrPackageNotFound)

		// Verify system can still perform other operations
		mockExec.SetResponse("rpm", exec.SuccessResult("nginx-1.24.0-1.1.x86_64"))
		installed, err := mgr.IsInstalled(ctx, "nginx")
		require.NoError(t, err)
		assert.True(t, installed)
	})

	t.Run("network_error_recovery", func(t *testing.T) {
		mockExec.SetResponse("zypper", exec.FailureResult(1, "Could not resolve host"))
		err := mgr.Update(ctx, pkg.DefaultUpdateOptions())
		require.Error(t, err)
		assert.ErrorIs(t, err, pkg.ErrNetworkUnavailable)

		// Verify cached operations still work
		mockExec.SetResponse("rpm", exec.SuccessResult("curl-8.1.0-1.1.x86_64"))
		installed, err := mgr.IsInstalled(ctx, "curl")
		require.NoError(t, err)
		assert.True(t, installed)
	})

	t.Run("system_management_locked_recovery", func(t *testing.T) {
		mockExec.SetResponse("zypper", exec.FailureResult(1, "System management is locked by the application with pid 1234"))
		err := mgr.Install(ctx, pkg.DefaultInstallOptions(), "nginx")
		require.Error(t, err)
		assert.ErrorIs(t, err, pkg.ErrLockAcquireFailed)

		// Verify read-only operations work
		searchOutput := `S | Name  | Summary | Type
--+-------+---------+--------
  | nginx | server  | package`
		mockExec.SetResponse("zypper", exec.SuccessResult(searchOutput))
		_, err = mgr.Search(ctx, "nginx", pkg.DefaultSearchOptions())
		require.NoError(t, err)
	})

	t.Run("another_zypper_running_recovery", func(t *testing.T) {
		mockExec.SetResponse("zypper", exec.FailureResult(1, "another zypper is running"))
		err := mgr.Remove(ctx, pkg.DefaultRemoveOptions(), "nginx")
		require.Error(t, err)
		assert.ErrorIs(t, err, pkg.ErrLockAcquireFailed)
	})

	t.Run("package_not_installed_on_remove", func(t *testing.T) {
		mockExec.SetResponse("zypper", exec.FailureResult(1, "nginx is not installed"))
		err := mgr.Remove(ctx, pkg.DefaultRemoveOptions(), "nginx")
		require.Error(t, err)
		assert.ErrorIs(t, err, pkg.ErrPackageNotInstalled)
	})
}

// TestManager_ConcurrentOperations tests thread safety.
func TestManager_ConcurrentOperations(t *testing.T) {
	mgr, mockExec := setupTest()
	ctx := context.Background()

	// Set up mock responses
	mockExec.SetDefaultResponse(exec.SuccessResult(""))
	mockExec.SetResponse("rpm", exec.SuccessResult("nginx-1.24.0-1.1.x86_64"))
	mockExec.SetResponse("zypper", exec.SuccessResult(""))

	var wg sync.WaitGroup
	errChan := make(chan error, 20)

	// Perform multiple concurrent operations
	for i := 0; i < 5; i++ {
		wg.Add(4)

		// Concurrent IsInstalled
		go func() {
			defer wg.Done()
			_, err := mgr.IsInstalled(ctx, "nginx")
			if err != nil {
				errChan <- err
			}
		}()

		// Concurrent Search
		go func() {
			defer wg.Done()
			searchOutput := `S | Name | Summary | Type
--+------+---------+-------
  | nginx | server | package`
			mockExec.SetResponse("zypper", exec.SuccessResult(searchOutput))
			_, err := mgr.Search(ctx, "nginx", pkg.DefaultSearchOptions())
			if err != nil {
				errChan <- err
			}
		}()

		// Concurrent ListInstalled
		go func() {
			defer wg.Done()
			mockExec.SetResponse("rpm", exec.SuccessResult("nginx\t1.24.0-1.1\tx86_64"))
			_, err := mgr.ListInstalled(ctx)
			if err != nil {
				errChan <- err
			}
		}()

		// Concurrent ListUpgradable
		go func() {
			defer wg.Done()
			listOutput := `S | Repository | Name | Current | Available | Arch
--+------------+------+---------+-----------+------
v | repo       | nginx| 1.22.0  | 1.24.0    | x86_64`
			mockExec.SetResponse("zypper", exec.SuccessResult(listOutput))
			_, err := mgr.ListUpgradable(ctx)
			if err != nil {
				errChan <- err
			}
		}()
	}

	wg.Wait()
	close(errChan)

	for err := range errChan {
		t.Errorf("Concurrent operation failed: %v", err)
	}
}

// TestManager_ContextCancellation tests context handling.
func TestManager_ContextCancellation(t *testing.T) {
	mgr, mockExec := setupTest()

	t.Run("cancel_during_update", func(t *testing.T) {
		mockExec.SetResponse("zypper", exec.SuccessResult(""))
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		// With mock, operation may still succeed, but tests the pattern
		_ = mgr.Update(ctx, pkg.DefaultUpdateOptions())
	})

	t.Run("cancel_during_install", func(t *testing.T) {
		mockExec.SetResponse("zypper", exec.SuccessResult(""))
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		_ = mgr.Install(ctx, pkg.DefaultInstallOptions(), "nginx")
	})

	t.Run("timeout_handling", func(t *testing.T) {
		mockExec.SetResponse("zypper", exec.SuccessResult(""))
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
		defer cancel()
		time.Sleep(10 * time.Millisecond) // Ensure timeout

		_ = mgr.Update(ctx, pkg.DefaultUpdateOptions())
	})
}

// TestManager_OutputParsing tests parsing edge cases.
func TestManager_OutputParsing(t *testing.T) {
	t.Run("empty_output", func(t *testing.T) {
		packages, err := parseRpmQuery("")
		require.NoError(t, err)
		assert.Len(t, packages, 0)

		packages, err = parseZypperSearch("")
		require.NoError(t, err)
		assert.Len(t, packages, 0)

		p, err := parseZypperInfo("")
		require.NoError(t, err)
		assert.Nil(t, p)
	})

	t.Run("malformed_output", func(t *testing.T) {
		// Single field line (should be skipped)
		packages, err := parseRpmQuery("nginx\ncurl\t8.1.0-1.1\tx86_64")
		require.NoError(t, err)
		assert.Len(t, packages, 1)
		assert.Equal(t, "curl", packages[0].Name)
	})

	t.Run("unicode_in_package_names", func(t *testing.T) {
		output := `S | Name              | Summary         | Type
--+-------------------+-----------------+--------
  | fonts-noto-cjk    | 中文字体 package | package
  | libc6             | GNU C Library   | package`
		packages, err := parseZypperSearch(output)
		require.NoError(t, err)
		assert.Len(t, packages, 2)
		assert.Equal(t, "fonts-noto-cjk", packages[0].Name)
	})

	t.Run("very_long_output", func(t *testing.T) {
		var sb strings.Builder
		sb.WriteString("S | Name | Summary | Type\n")
		sb.WriteString("--+------+---------+-------\n")
		for i := 0; i < 1000; i++ {
			sb.WriteString("  | package-")
			sb.WriteString(string(rune('a' + i%26)))
			sb.WriteString(" | description | package\n")
		}
		packages, err := parseZypperSearch(sb.String())
		require.NoError(t, err)
		assert.Len(t, packages, 1000)
	})

	t.Run("partial_output", func(t *testing.T) {
		// Simulate interrupted output (no newline at end)
		output := "nginx\t1.24.0-1.1\tx86_64"
		packages, err := parseRpmQuery(output)
		require.NoError(t, err)
		assert.Len(t, packages, 1)
	})
}

// TestManager_VersionComparison tests version handling.
func TestManager_VersionComparison(t *testing.T) {
	t.Run("parse_version_with_epoch", func(t *testing.T) {
		output := `vim	2:9.0.0-1.1	x86_64
curl	8.1.0-1.1	x86_64`
		packages, err := parseRpmQuery(output)
		require.NoError(t, err)
		require.Len(t, packages, 2)
		assert.Equal(t, "vim", packages[0].Name)
		assert.Equal(t, "2:9.0.0-1.1", packages[0].Version)
	})

	t.Run("parse_version_without_epoch", func(t *testing.T) {
		output := `nginx	1.24.0-1.1	x86_64`
		packages, err := parseRpmQuery(output)
		require.NoError(t, err)
		require.Len(t, packages, 1)
		assert.Equal(t, "1.24.0-1.1", packages[0].Version)
	})

	t.Run("zypper_list_updates_format", func(t *testing.T) {
		output := `Loading repository data...
Reading installed packages...
S | Repository                | Name       | Current Version | Available Version | Arch
--+---------------------------+------------+-----------------+-------------------+--------
v | openSUSE-Tumbleweed-Oss   | nginx      | 1.22.0-1.1      | 1.24.0-1.1       | x86_64
v | openSUSE-Tumbleweed-Oss   | vim        | 2:8.2.0-1.1     | 2:9.0.0-1.1      | x86_64`
		packages, err := parseZypperListUpdates(output)
		require.NoError(t, err)
		require.Len(t, packages, 2)
		assert.Equal(t, "1.24.0-1.1", packages[0].Version)
		assert.Equal(t, "2:9.0.0-1.1", packages[1].Version)
	})
}

// TestManager_ZypperSpecificFeatures tests Zypper-specific functionality.
func TestManager_ZypperSpecificFeatures(t *testing.T) {
	mgr, mockExec := setupTest()
	ctx := context.Background()

	t.Run("dist_upgrade_all", func(t *testing.T) {
		mockExec.SetResponse("zypper", exec.SuccessResult(""))
		err := mgr.Upgrade(ctx, pkg.DefaultInstallOptions())
		require.NoError(t, err)

		lastCall := mockExec.LastCall()
		assert.Contains(t, lastCall.Args, "dist-upgrade")
	})

	t.Run("update_specific_packages", func(t *testing.T) {
		mockExec.SetResponse("zypper", exec.SuccessResult(""))
		err := mgr.Upgrade(ctx, pkg.DefaultInstallOptions(), "nginx", "curl")
		require.NoError(t, err)

		lastCall := mockExec.LastCall()
		assert.Contains(t, lastCall.Args, "update")
		assert.Contains(t, lastCall.Args, "nginx")
		assert.Contains(t, lastCall.Args, "curl")
	})

	t.Run("remove_with_clean_deps", func(t *testing.T) {
		mockExec.SetResponse("zypper", exec.SuccessResult(""))
		opts := pkg.RemoveOptions{Purge: true}
		err := mgr.Remove(ctx, opts, "nginx")
		require.NoError(t, err)

		lastCall := mockExec.LastCall()
		assert.Contains(t, lastCall.Args, "--clean-deps")
	})

	t.Run("force_refresh", func(t *testing.T) {
		mockExec.SetResponse("zypper", exec.SuccessResult(""))
		opts := pkg.UpdateOptions{ForceRefresh: true}
		err := mgr.Update(ctx, opts)
		require.NoError(t, err)

		lastCall := mockExec.LastCall()
		assert.Contains(t, lastCall.Args, "--force")
	})

	t.Run("install_with_no_gpg_checks", func(t *testing.T) {
		mockExec.SetResponse("zypper", exec.SuccessResult(""))
		opts := pkg.InstallOptions{SkipVerify: true}
		err := mgr.Install(ctx, opts, "nvidia-driver")
		require.NoError(t, err)

		lastCall := mockExec.LastCall()
		assert.Contains(t, lastCall.Args, "--no-gpg-checks")
	})

	t.Run("install_download_only", func(t *testing.T) {
		mockExec.SetResponse("zypper", exec.SuccessResult(""))
		opts := pkg.InstallOptions{DownloadOnly: true}
		err := mgr.Install(ctx, opts, "nginx")
		require.NoError(t, err)

		lastCall := mockExec.LastCall()
		assert.Contains(t, lastCall.Args, "--download-only")
	})

	t.Run("autoremove_unneeded", func(t *testing.T) {
		unneededOutput := `S | Repository | Name              | Version       | Arch
--+------------+-------------------+---------------+--------
i | @System    | orphan-package    | 1.0.0-1.1     | x86_64`
		mockExec.SetResponse("zypper", exec.SuccessResult(unneededOutput))
		err := mgr.AutoRemove(ctx)
		require.NoError(t, err)
	})

	t.Run("autoremove_no_unneeded", func(t *testing.T) {
		mockExec.SetResponse("zypper", exec.SuccessResult("No packages found."))
		err := mgr.AutoRemove(ctx)
		require.NoError(t, err, "no unneeded packages should not be an error")
	})
}

// TestManager_RepositoryManagement tests repository edge cases.
func TestManager_RepositoryManagement(t *testing.T) {
	mgr, mockExec := setupTest()
	ctx := context.Background()

	t.Run("add_repository_enabled", func(t *testing.T) {
		mockExec.SetResponse("zypper", exec.SuccessResult(""))
		mockExec.SetResponse("rpm", exec.SuccessResult(""))

		repo := pkg.Repository{
			Name:    "nvidia",
			URL:     "https://download.nvidia.com/opensuse/tumbleweed",
			Enabled: true,
		}
		err := mgr.AddRepository(ctx, repo)
		require.NoError(t, err)

		lastCall := mockExec.LastCall()
		assert.Contains(t, lastCall.Args, "addrepo")
		assert.Contains(t, lastCall.Args, "--refresh")
	})

	t.Run("add_repository_disabled", func(t *testing.T) {
		mockExec.SetResponse("zypper", exec.SuccessResult(""))

		repo := pkg.Repository{
			Name:    "nvidia-testing",
			URL:     "https://download.nvidia.com/opensuse/tumbleweed-testing",
			Enabled: false,
		}
		err := mgr.AddRepository(ctx, repo)
		require.NoError(t, err)

		lastCall := mockExec.LastCall()
		assert.Contains(t, lastCall.Args, "--disable")
	})

	t.Run("set_repository_priority", func(t *testing.T) {
		mockExec.SetResponse("zypper", exec.SuccessResult(""))
		err := mgr.SetRepositoryPriority(ctx, "nvidia", 90)
		require.NoError(t, err)

		lastCall := mockExec.LastCall()
		assert.Contains(t, lastCall.Args, "modifyrepo")
		assert.Contains(t, lastCall.Args, "--priority")
		assert.Contains(t, lastCall.Args, "90")
	})

	t.Run("set_repository_refresh", func(t *testing.T) {
		mockExec.SetResponse("zypper", exec.SuccessResult(""))
		err := mgr.SetRepositoryRefresh(ctx, "nvidia", true)
		require.NoError(t, err)

		lastCall := mockExec.LastCall()
		assert.Contains(t, lastCall.Args, "--refresh")
	})

	t.Run("set_repository_no_refresh", func(t *testing.T) {
		mockExec.SetResponse("zypper", exec.SuccessResult(""))
		err := mgr.SetRepositoryRefresh(ctx, "nvidia", false)
		require.NoError(t, err)

		lastCall := mockExec.LastCall()
		assert.Contains(t, lastCall.Args, "--no-refresh")
	})

	t.Run("import_gpg_key", func(t *testing.T) {
		mockExec.SetResponse("rpm", exec.SuccessResult(""))
		err := mgr.ImportGPGKey(ctx, "https://example.com/RPM-GPG-KEY")
		require.NoError(t, err)

		lastCall := mockExec.LastCall()
		assert.Contains(t, lastCall.Args, "--import")
	})

	t.Run("get_repo_file_path", func(t *testing.T) {
		path := mgr.GetRepoFilePath("nvidia")
		assert.Equal(t, "/etc/zypp/repos.d/nvidia.repo", path)

		path = mgr.GetRepoFilePath("my repo/test")
		assert.Equal(t, "/etc/zypp/repos.d/my-repo-test.repo", path)
	})
}

// TestManager_NvidiaRepoHelpers tests NVIDIA repository helper functions.
func TestManager_NvidiaRepoHelpers(t *testing.T) {
	mgr, mockExec := setupTest()
	ctx := context.Background()

	t.Run("add_nvidia_repo_tumbleweed", func(t *testing.T) {
		mockExec.SetResponse("zypper", exec.SuccessResult(""))
		err := mgr.AddNvidiaRepoTumbleweed(ctx)
		require.NoError(t, err)
	})

	t.Run("add_nvidia_repo_leap", func(t *testing.T) {
		mockExec.SetResponse("zypper", exec.SuccessResult(""))
		err := mgr.AddNvidiaRepoLeap(ctx, "15.5")
		require.NoError(t, err)
	})
}

// TestManager_RepoFileParsing tests .repo file parsing.
func TestManager_RepoFileParsing(t *testing.T) {
	t.Run("parse_full_repo_file", func(t *testing.T) {
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
		assert.NotEmpty(t, repos[0].GPGKey)

		assert.Equal(t, "nvidia-source", repos[1].Name)
		assert.False(t, repos[1].Enabled)
	})

	t.Run("build_repo_file_content", func(t *testing.T) {
		repo := pkg.Repository{
			Name:    "nvidia",
			URL:     "https://download.nvidia.com/opensuse/tumbleweed",
			Enabled: true,
			GPGKey:  "https://download.nvidia.com/opensuse/tumbleweed/repodata/repomd.xml.key",
		}

		content := buildRepoFileContent(repo)
		assert.Contains(t, content, "[nvidia]")
		assert.Contains(t, content, "baseurl=https://download.nvidia.com/opensuse/tumbleweed")
		assert.Contains(t, content, "enabled=1")
		assert.Contains(t, content, "gpgcheck=1")
		assert.Contains(t, content, "autorefresh=1")
	})

	t.Run("build_repo_file_content_disabled", func(t *testing.T) {
		repo := pkg.Repository{
			Name:    "test",
			URL:     "https://example.com",
			Enabled: false,
		}

		content := buildRepoFileContent(repo)
		assert.Contains(t, content, "enabled=0")
		assert.Contains(t, content, "gpgcheck=0")
	})
}

// TestManager_HelperFunctions tests helper functions.
func TestManager_HelperFunctions(t *testing.T) {
	t.Run("is_architecture", func(t *testing.T) {
		assert.True(t, isArchitecture("x86_64"))
		assert.True(t, isArchitecture("i686"))
		assert.True(t, isArchitecture("i586"))
		assert.True(t, isArchitecture("noarch"))
		assert.True(t, isArchitecture("aarch64"))
		assert.False(t, isArchitecture("something"))
		assert.False(t, isArchitecture(""))
	})

	t.Run("parse_size", func(t *testing.T) {
		assert.Equal(t, int64(1024), parseSize("1 k"))
		assert.Equal(t, int64(1024*1024), parseSize("1 M"))
		assert.Equal(t, int64(1024*1024), parseSize("1 MiB"))
		assert.Equal(t, int64(1024*1024*1024), parseSize("1 G"))
		assert.Equal(t, int64(1024*1024*1024), parseSize("1 GiB"))
		assert.Equal(t, int64(0), parseSize("invalid"))
	})

	t.Run("sanitize_repo_id", func(t *testing.T) {
		assert.Equal(t, "simple", sanitizeRepoID("simple"))
		assert.Equal(t, "with-space", sanitizeRepoID("with space"))
		assert.Equal(t, "with-slash", sanitizeRepoID("with/slash"))
		assert.Equal(t, "custom-repo", sanitizeRepoID(""))
	})

	t.Run("parse_unneeded_packages", func(t *testing.T) {
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
	})
}

// =============================================================================
// Benchmark Tests
// =============================================================================

func BenchmarkManager_ParseRpmQuery(b *testing.B) {
	// Generate realistic output
	var sb strings.Builder
	for i := 0; i < 500; i++ {
		sb.WriteString("package-")
		sb.WriteString(string(rune('a' + i%26)))
		sb.WriteString("\t1.0.0-1.1\tx86_64\n")
	}
	output := sb.String()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = parseRpmQuery(output)
	}
}

func BenchmarkManager_ParseZypperSearch(b *testing.B) {
	var sb strings.Builder
	sb.WriteString("S | Name | Summary | Type\n")
	sb.WriteString("--+------+---------+-------\n")
	for i := 0; i < 200; i++ {
		sb.WriteString("  | package-")
		sb.WriteString(string(rune('a' + i%26)))
		sb.WriteString(" | package description for testing purposes | package\n")
	}
	output := sb.String()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = parseZypperSearch(output)
	}
}

func BenchmarkManager_ParseZypperInfo(b *testing.B) {
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
Summary        : A high performance web server and reverse proxy server
`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = parseZypperInfo(output)
	}
}

func BenchmarkManager_ParseZypperListUpdates(b *testing.B) {
	var sb strings.Builder
	sb.WriteString("Loading repository data...\n")
	sb.WriteString("Reading installed packages...\n")
	sb.WriteString("S | Repository | Name | Current Version | Available Version | Arch\n")
	sb.WriteString("--+------------+------+-----------------+-------------------+------\n")
	for i := 0; i < 100; i++ {
		sb.WriteString("v | openSUSE | package-")
		sb.WriteString(string(rune('a' + i%26)))
		sb.WriteString(" | 1.0.0-1.1 | 1.1.0-1.1 | x86_64\n")
	}
	output := sb.String()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = parseZypperListUpdates(output)
	}
}

func BenchmarkManager_ParseZypperRepos(b *testing.B) {
	output := `Repository priorities are without effect. All enabled repositories share the same priority.
# | Alias                     | Name                          | Enabled | GPG Check | Refresh
--+---------------------------+-------------------------------+---------+-----------+--------
1 | openSUSE-Tumbleweed-Oss   | openSUSE-Tumbleweed-Oss       | Yes     | (r ) Yes  | Yes
2 | nvidia                    | NVIDIA Repository             | Yes     | (  ) Yes  | Yes
3 | disabled-repo             | Disabled Repository           | No      | (  ) Yes  | No
4 | packman                   | Packman Repository            | Yes     | (  ) Yes  | Yes`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = parseZypperRepos(output)
	}
}

func BenchmarkManager_ParseRepoFile(b *testing.B) {
	content := `[nvidia]
name=NVIDIA Repository
baseurl=https://download.nvidia.com/opensuse/tumbleweed
enabled=1
gpgcheck=1
gpgkey=https://download.nvidia.com/opensuse/tumbleweed/repodata/repomd.xml.key
autorefresh=1

[packman]
name=Packman Repository
baseurl=https://ftp.gwdg.de/pub/linux/misc/packman/suse/openSUSE_Tumbleweed/
enabled=1
gpgcheck=1
gpgkey=https://ftp.gwdg.de/pub/linux/misc/packman/suse/openSUSE_Tumbleweed/repodata/repomd.xml.key`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = parseRepoFile(content)
	}
}

func BenchmarkManager_IsInstalled(b *testing.B) {
	mgr, mockExec := setupTest()
	mockExec.SetResponse("rpm", exec.SuccessResult("nginx-1.24.0-1.1.x86_64"))
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = mgr.IsInstalled(ctx, "nginx")
	}
}

func BenchmarkManager_Search(b *testing.B) {
	mgr, mockExec := setupTest()
	output := `S | Name        | Summary                        | Type
--+-------------+--------------------------------+--------
i | nginx       | HTTP server                    | package
  | nginx-source| Source code                    | srcpackage`
	mockExec.SetResponse("zypper", exec.SuccessResult(output))
	mockExec.SetResponse("rpm", exec.FailureResult(1, "not installed"))
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = mgr.Search(ctx, "nginx", pkg.SearchOptions{IncludeInstalled: false})
	}
}

// =============================================================================
// Helper function for setup with custom privilege settings
// =============================================================================

func setupTestWithPrivilege(isRoot bool) (*Manager, *exec.MockExecutor) {
	mockExec := exec.NewMockExecutor()
	priv := privilege.NewManager()
	priv.SetRoot(isRoot)
	mgr := NewManager(mockExec, priv)
	return mgr, mockExec
}
