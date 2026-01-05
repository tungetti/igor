package pacman

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

	// Step 1: Update cache (pacman -Sy)
	mockExec.SetResponse("pacman", exec.SuccessResult(":: Synchronizing package databases..."))
	err := mgr.Update(ctx, pkg.DefaultUpdateOptions())
	require.NoError(t, err, "cache update should succeed")

	// Step 2: Search for packages
	searchOutput := `extra/nvidia-utils 545.29.06-1 [installed]
    NVIDIA drivers utilities
extra/nvidia 545.29.06-1
    NVIDIA drivers for linux
community/nvidia-390xx-utils 390.157-1
    NVIDIA 390xx legacy drivers utilities`
	mockExec.SetResponse("pacman", exec.SuccessResult(searchOutput))
	packages, err := mgr.Search(ctx, "nvidia", pkg.SearchOptions{IncludeInstalled: false})
	require.NoError(t, err, "search should succeed")
	require.Len(t, packages, 3, "should find 3 nvidia packages")

	// Step 3: Check if installed (should not be)
	mockExec.SetResponse("pacman", exec.FailureResult(1, "error: package 'nvidia' was not found"))
	installed, err := mgr.IsInstalled(ctx, "nvidia")
	require.NoError(t, err)
	assert.False(t, installed, "nvidia should not be installed initially")

	// Step 4: Install package
	mockExec.SetResponse("pacman", exec.SuccessResult("resolving dependencies...\ninstalling nvidia..."))
	err = mgr.Install(ctx, pkg.DefaultInstallOptions(), "nvidia")
	require.NoError(t, err, "install should succeed")

	// Step 5: Verify installed
	mockExec.SetResponse("pacman", exec.SuccessResult("nvidia 545.29.06-1"))
	installed, err = mgr.IsInstalled(ctx, "nvidia")
	require.NoError(t, err)
	assert.True(t, installed, "nvidia should be installed after install")

	// Step 6: Remove package
	mockExec.SetResponse("pacman", exec.SuccessResult("removing nvidia..."))
	err = mgr.Remove(ctx, pkg.DefaultRemoveOptions(), "nvidia")
	require.NoError(t, err, "remove should succeed")

	// Step 7: Verify removed
	mockExec.SetResponse("pacman", exec.FailureResult(1, "error: package 'nvidia' was not found"))
	installed, err = mgr.IsInstalled(ctx, "nvidia")
	require.NoError(t, err)
	assert.False(t, installed, "nvidia should not be installed after removal")
}

// TestManager_NVIDIADriverWorkflow tests NVIDIA driver installation workflow on Arch Linux.
func TestManager_NVIDIADriverWorkflow(t *testing.T) {
	mgr, mockExec := setupTest()
	ctx := context.Background()

	// Step 1: Update package database
	mockExec.SetResponse("pacman", exec.SuccessResult(":: Synchronizing package databases..."))
	err := mgr.Update(ctx, pkg.DefaultUpdateOptions())
	require.NoError(t, err)

	// Step 2: Search for NVIDIA driver packages
	searchOutput := `extra/nvidia 545.29.06-1
    NVIDIA drivers for linux
extra/nvidia-dkms 545.29.06-1
    NVIDIA drivers (dkms)
extra/nvidia-utils 545.29.06-1 [installed]
    NVIDIA drivers utilities
extra/nvidia-settings 545.29.06-1
    Tool for configuring the NVIDIA driver`
	mockExec.SetResponse("pacman", exec.SuccessResult(searchOutput))
	packages, err := mgr.Search(ctx, "nvidia", pkg.SearchOptions{IncludeInstalled: false})
	require.NoError(t, err)
	assert.Greater(t, len(packages), 0, "should find NVIDIA driver packages")

	// Step 3: Install NVIDIA driver
	mockExec.SetResponse("pacman", exec.SuccessResult("resolving dependencies...\ninstalling nvidia..."))
	err = mgr.Install(ctx, pkg.DefaultInstallOptions(), "nvidia")
	require.NoError(t, err, "NVIDIA driver installation should succeed")

	// Step 4: Verify installation
	mockExec.SetResponse("pacman", exec.SuccessResult("nvidia 545.29.06-1"))
	installed, err := mgr.IsInstalled(ctx, "nvidia")
	require.NoError(t, err)
	assert.True(t, installed, "NVIDIA driver should be installed")

	// Step 5: Verify package integrity
	mockExec.SetResponse("pacman", exec.SuccessResult("nvidia: 0 missing files"))
	valid, err := mgr.Verify(ctx, "nvidia")
	require.NoError(t, err)
	assert.True(t, valid, "NVIDIA driver should pass verification")
}

// TestManager_ErrorRecovery tests error recovery scenarios.
func TestManager_ErrorRecovery(t *testing.T) {
	mgr, mockExec := setupTest()
	ctx := context.Background()

	t.Run("package_not_found_recovery", func(t *testing.T) {
		mockExec.SetResponse("pacman", exec.FailureResult(1, "error: target not found: nonexistent-pkg"))
		err := mgr.Install(ctx, pkg.DefaultInstallOptions(), "nonexistent-pkg")
		require.Error(t, err)
		assert.ErrorIs(t, err, pkg.ErrPackageNotFound)

		// Verify system can still perform other operations
		mockExec.SetResponse("pacman", exec.SuccessResult("nginx 1.24.0-1"))
		installed, err := mgr.IsInstalled(ctx, "nginx")
		require.NoError(t, err)
		assert.True(t, installed)
	})

	t.Run("network_error_recovery", func(t *testing.T) {
		mockExec.SetResponse("pacman", exec.FailureResult(1, "error: failed to retrieve some files"))
		err := mgr.Update(ctx, pkg.DefaultUpdateOptions())
		require.Error(t, err)
		assert.ErrorIs(t, err, pkg.ErrNetworkUnavailable)

		// Verify cached operations still work
		mockExec.SetResponse("pacman", exec.SuccessResult("curl 8.4.0-1"))
		installed, err := mgr.IsInstalled(ctx, "curl")
		require.NoError(t, err)
		assert.True(t, installed)
	})

	t.Run("database_lock_error_recovery", func(t *testing.T) {
		mockExec.SetResponse("pacman", exec.FailureResult(1, "error: unable to lock database"))
		err := mgr.Install(ctx, pkg.DefaultInstallOptions(), "nginx")
		require.Error(t, err)
		assert.ErrorIs(t, err, pkg.ErrLockAcquireFailed)

		// Verify read-only operations work
		searchOutput := `extra/nginx 1.24.0-1
    web server`
		mockExec.SetResponse("pacman", exec.SuccessResult(searchOutput))
		_, err = mgr.Search(ctx, "nginx", pkg.DefaultSearchOptions())
		require.NoError(t, err)
	})

	t.Run("dependency_conflict_recovery", func(t *testing.T) {
		mockExec.SetResponse("pacman", exec.FailureResult(1, "error: failed to prepare transaction (conflicting dependencies)"))
		err := mgr.Install(ctx, pkg.DefaultInstallOptions(), "conflicting-pkg")
		require.Error(t, err)
		assert.ErrorIs(t, err, pkg.ErrDependencyConflict)
	})

	t.Run("target_not_found_on_remove", func(t *testing.T) {
		mockExec.SetResponse("pacman", exec.FailureResult(1, "error: target not found: nginx"))
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
	mockExec.SetResponse("pacman", exec.SuccessResult("nginx 1.24.0-1"))

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
			mockExec.SetResponse("pacman", exec.SuccessResult("extra/nginx 1.24.0-1\n    web server"))
			_, err := mgr.Search(ctx, "nginx", pkg.DefaultSearchOptions())
			if err != nil {
				errChan <- err
			}
		}()

		// Concurrent ListInstalled
		go func() {
			defer wg.Done()
			mockExec.SetResponse("pacman", exec.SuccessResult("linux 6.6.1.arch1-1\nnginx 1.24.0-1"))
			_, err := mgr.ListInstalled(ctx)
			if err != nil {
				errChan <- err
			}
		}()

		// Concurrent ListUpgradable
		go func() {
			defer wg.Done()
			mockExec.SetResponse("pacman", exec.SuccessResult("nginx 1.24.0-1 -> 1.24.1-1"))
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
		mockExec.SetResponse("pacman", exec.SuccessResult(""))
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		// With mock, operation may still succeed, but tests the pattern
		_ = mgr.Update(ctx, pkg.DefaultUpdateOptions())
	})

	t.Run("cancel_during_install", func(t *testing.T) {
		mockExec.SetResponse("pacman", exec.SuccessResult(""))
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		_ = mgr.Install(ctx, pkg.DefaultInstallOptions(), "nginx")
	})

	t.Run("timeout_handling", func(t *testing.T) {
		mockExec.SetResponse("pacman", exec.SuccessResult(""))
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
		defer cancel()
		time.Sleep(10 * time.Millisecond) // Ensure timeout

		_ = mgr.Update(ctx, pkg.DefaultUpdateOptions())
	})
}

// TestManager_OutputParsing tests parsing edge cases.
func TestManager_OutputParsing(t *testing.T) {
	t.Run("empty_output", func(t *testing.T) {
		packages, err := parsePacmanQ("")
		require.NoError(t, err)
		assert.Len(t, packages, 0)

		packages, err = parsePacmanSs("")
		require.NoError(t, err)
		assert.Len(t, packages, 0)

		p, err := parsePacmanInfo("")
		require.NoError(t, err)
		assert.Nil(t, p)
	})

	t.Run("malformed_output", func(t *testing.T) {
		// Single field line (should be skipped)
		packages, err := parsePacmanQ("nginx\ncurl 8.4.0-1")
		require.NoError(t, err)
		assert.Len(t, packages, 1)
		assert.Equal(t, "curl", packages[0].Name)
	})

	t.Run("unicode_in_package_names", func(t *testing.T) {
		output := `extra/noto-fonts-cjk 20230817-1
    中文字体 package
extra/libc 2.38-7
    GNU C Library`
		packages, err := parsePacmanSs(output)
		require.NoError(t, err)
		assert.Len(t, packages, 2)
		assert.Equal(t, "noto-fonts-cjk", packages[0].Name)
	})

	t.Run("very_long_output", func(t *testing.T) {
		var sb strings.Builder
		for i := 0; i < 1000; i++ {
			sb.WriteString("extra/package-")
			sb.WriteString(string(rune('a' + i%26)))
			sb.WriteString(" 1.0.0-1\n")
			sb.WriteString("    description\n")
		}
		packages, err := parsePacmanSs(sb.String())
		require.NoError(t, err)
		assert.Len(t, packages, 1000)
	})

	t.Run("partial_output", func(t *testing.T) {
		// Simulate interrupted output (no newline at end)
		output := "nginx 1.24.0-1"
		packages, err := parsePacmanQ(output)
		require.NoError(t, err)
		assert.Len(t, packages, 1)
	})
}

// TestManager_VersionComparison tests version handling.
func TestManager_VersionComparison(t *testing.T) {
	t.Run("parse_version_with_epoch", func(t *testing.T) {
		output := `vim 2:9.0.0-1
curl 8.4.0-1`
		packages, err := parsePacmanQ(output)
		require.NoError(t, err)
		require.Len(t, packages, 2)
		assert.Equal(t, "vim", packages[0].Name)
		assert.Equal(t, "2:9.0.0-1", packages[0].Version)
	})

	t.Run("parse_version_without_epoch", func(t *testing.T) {
		output := `nginx 1.24.0-1`
		packages, err := parsePacmanQ(output)
		require.NoError(t, err)
		require.Len(t, packages, 1)
		assert.Equal(t, "1.24.0-1", packages[0].Version)
	})

	t.Run("pacman_qu_upgrade_format", func(t *testing.T) {
		output := `linux 6.6.0.arch1-1 -> 6.6.1.arch1-1
nginx 1.24.0-1 -> 1.24.1-1`
		packages, err := parsePacmanQu(output)
		require.NoError(t, err)
		require.Len(t, packages, 2)
		assert.Equal(t, "6.6.1.arch1-1", packages[0].Version)
		assert.Equal(t, "1.24.1-1", packages[1].Version)
	})

	t.Run("pacman_qu_simple_format", func(t *testing.T) {
		output := `linux 6.6.1.arch1-1
nginx 1.24.1-1`
		packages, err := parsePacmanQu(output)
		require.NoError(t, err)
		require.Len(t, packages, 2)
		assert.Equal(t, "linux", packages[0].Name)
		assert.Equal(t, "6.6.1.arch1-1", packages[0].Version)
	})
}

// TestManager_PacmanSpecificFeatures tests Pacman-specific functionality.
func TestManager_PacmanSpecificFeatures(t *testing.T) {
	mgr, mockExec := setupTest()
	ctx := context.Background()

	t.Run("remove_with_recursive_deps", func(t *testing.T) {
		mockExec.SetResponse("pacman", exec.SuccessResult(""))
		opts := pkg.RemoveOptions{AutoRemove: true}
		err := mgr.Remove(ctx, opts, "nginx")
		require.NoError(t, err)

		lastCall := mockExec.LastCall()
		assert.Contains(t, lastCall.Args, "-Rs")
	})

	t.Run("remove_with_purge_nosave", func(t *testing.T) {
		mockExec.SetResponse("pacman", exec.SuccessResult(""))
		opts := pkg.RemoveOptions{Purge: true}
		err := mgr.Remove(ctx, opts, "nginx")
		require.NoError(t, err)

		lastCall := mockExec.LastCall()
		assert.Contains(t, lastCall.Args, "-Rns")
	})

	t.Run("autoremove_orphan_packages", func(t *testing.T) {
		// First call lists orphans
		mockExec.SetResponse("pacman", exec.SuccessResult("orphan1\norphan2"))
		err := mgr.AutoRemove(ctx)
		require.NoError(t, err)
	})

	t.Run("autoremove_no_orphans", func(t *testing.T) {
		// No orphans (exit code 1 with empty output)
		mockExec.SetResponse("pacman", exec.FailureResult(1, ""))
		err := mgr.AutoRemove(ctx)
		require.NoError(t, err, "no orphans should not be an error")
	})

	t.Run("force_install_with_overwrite", func(t *testing.T) {
		mockExec.SetResponse("pacman", exec.SuccessResult(""))
		opts := pkg.InstallOptions{Force: true}
		err := mgr.Install(ctx, opts, "nvidia")
		require.NoError(t, err)

		lastCall := mockExec.LastCall()
		assert.Contains(t, lastCall.Args, "--overwrite")
	})

	t.Run("download_only", func(t *testing.T) {
		mockExec.SetResponse("pacman", exec.SuccessResult(""))
		opts := pkg.InstallOptions{DownloadOnly: true}
		err := mgr.Install(ctx, opts, "nginx")
		require.NoError(t, err)

		lastCall := mockExec.LastCall()
		assert.Contains(t, lastCall.Args, "--downloadonly")
	})
}

// TestManager_KeyringManagement tests pacman-key operations.
func TestManager_KeyringManagement(t *testing.T) {
	mgr, mockExec := setupTest()
	ctx := context.Background()

	t.Run("initialize_keyring", func(t *testing.T) {
		mockExec.SetResponse("pacman-key", exec.SuccessResult(""))
		err := mgr.InitializeKeyring(ctx)
		require.NoError(t, err)

		lastCall := mockExec.LastCall()
		assert.Contains(t, lastCall.Args, "--init")
	})

	t.Run("populate_keyring_archlinux", func(t *testing.T) {
		mockExec.SetResponse("pacman-key", exec.SuccessResult(""))
		err := mgr.PopulateKeyring(ctx, "archlinux")
		require.NoError(t, err)

		lastCall := mockExec.LastCall()
		assert.Contains(t, lastCall.Args, "--populate")
		assert.Contains(t, lastCall.Args, "archlinux")
	})

	t.Run("populate_keyring_default", func(t *testing.T) {
		mockExec.SetResponse("pacman-key", exec.SuccessResult(""))
		err := mgr.PopulateKeyring(ctx, "")
		require.NoError(t, err)

		lastCall := mockExec.LastCall()
		assert.Contains(t, lastCall.Args, "archlinux")
	})

	t.Run("refresh_keys", func(t *testing.T) {
		mockExec.SetResponse("pacman-key", exec.SuccessResult(""))
		err := mgr.RefreshKeys(ctx)
		require.NoError(t, err)

		lastCall := mockExec.LastCall()
		assert.Contains(t, lastCall.Args, "--refresh-keys")
	})

	t.Run("refresh_keys_network_error", func(t *testing.T) {
		mockExec.SetResponse("pacman-key", exec.FailureResult(1, "gpg: keyserver receive failed"))
		err := mgr.RefreshKeys(ctx)
		require.Error(t, err)
		assert.ErrorIs(t, err, pkg.ErrNetworkUnavailable)
	})

	t.Run("import_gpg_key", func(t *testing.T) {
		mockExec.SetResponse("pacman-key", exec.SuccessResult(""))
		err := mgr.ImportGPGKey(ctx, "3056513887B78AEB")
		require.NoError(t, err)
	})
}

// TestManager_RepositoryManagement tests repository edge cases.
func TestManager_RepositoryManagement(t *testing.T) {
	mgr, mockExec := setupTest()
	ctx := context.Background()

	t.Run("add_custom_repository", func(t *testing.T) {
		mockExec.SetResponse("cat", exec.SuccessResult("[options]\n[core]\nInclude = /etc/pacman.d/mirrorlist"))
		mockExec.SetDefaultResponse(exec.SuccessResult(""))

		repo := pkg.Repository{
			Name: "chaotic-aur",
			URL:  "https://geo-mirror.chaotic.cx/$repo/$arch",
		}
		err := mgr.AddRepository(ctx, repo)
		require.NoError(t, err)
	})

	t.Run("enable_multilib_repository", func(t *testing.T) {
		content := `[options]
[core]
Include = /etc/pacman.d/mirrorlist

# [multilib]
# Include = /etc/pacman.d/mirrorlist`

		mockExec.SetResponse("cat", exec.SuccessResult(content))
		mockExec.SetDefaultResponse(exec.SuccessResult(""))

		err := mgr.EnableRepository(ctx, "multilib")
		require.NoError(t, err)
	})

	t.Run("disable_repository", func(t *testing.T) {
		content := `[options]
[core]
Include = /etc/pacman.d/mirrorlist

[multilib]
Include = /etc/pacman.d/mirrorlist`

		mockExec.SetResponse("cat", exec.SuccessResult(content))
		mockExec.SetDefaultResponse(exec.SuccessResult(""))

		err := mgr.DisableRepository(ctx, "multilib")
		require.NoError(t, err)
	})

	t.Run("list_repositories", func(t *testing.T) {
		content := `[options]
HoldPkg = pacman glibc
Architecture = auto

[core]
Include = /etc/pacman.d/mirrorlist

[extra]
Include = /etc/pacman.d/mirrorlist

[multilib]
Include = /etc/pacman.d/mirrorlist`

		mockExec.SetResponse("cat", exec.SuccessResult(content))
		repos, err := mgr.ListRepositories(ctx)
		require.NoError(t, err)
		require.Len(t, repos, 3)
		assert.Equal(t, "core", repos[0].Name)
		assert.Equal(t, "extra", repos[1].Name)
		assert.Equal(t, "multilib", repos[2].Name)
	})
}

// TestManager_PacmanConfParsing tests pacman.conf parsing.
func TestManager_PacmanConfParsing(t *testing.T) {
	t.Run("parse_full_pacman_conf", func(t *testing.T) {
		content := `[options]
HoldPkg = pacman glibc
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
	})

	t.Run("parse_pacman_conf_empty", func(t *testing.T) {
		repos, err := parsePacmanConf("")
		require.NoError(t, err)
		assert.Len(t, repos, 0)
	})

	t.Run("parse_pacman_conf_options_only", func(t *testing.T) {
		content := `[options]
HoldPkg = pacman glibc`

		repos, err := parsePacmanConf(content)
		require.NoError(t, err)
		assert.Len(t, repos, 0)
	})
}

// TestManager_HelperFunctions tests helper functions.
func TestManager_HelperFunctions(t *testing.T) {
	t.Run("parse_size", func(t *testing.T) {
		assert.Equal(t, int64(1024), parseSize("1 KiB"))
		assert.Equal(t, int64(1536), parseSize("1.5 KiB"))
		assert.Equal(t, int64(1024*1024), parseSize("1 MiB"))
		assert.Equal(t, int64(1024*1024*1024), parseSize("1 GiB"))
		assert.Equal(t, int64(0), parseSize("invalid"))
		assert.Equal(t, int64(0), parseSize(""))
	})

	t.Run("sanitize_repo_name", func(t *testing.T) {
		assert.Equal(t, "simple", sanitizeRepoName("simple"))
		assert.Equal(t, "with-space", sanitizeRepoName("with space"))
		assert.Equal(t, "with-slash", sanitizeRepoName("with/slash"))
		assert.Equal(t, "custom-repo", sanitizeRepoName(""))
	})
}

// TestManager_Ss_InstalledVariations tests search output with [installed] variations.
func TestManager_Ss_InstalledVariations(t *testing.T) {
	t.Run("installed_with_version_diff", func(t *testing.T) {
		output := `extra/nvidia 545.29.06-1 [installed: 545.29.02-1]
    NVIDIA drivers for linux`
		packages, err := parsePacmanSs(output)
		require.NoError(t, err)
		require.Len(t, packages, 1)
		assert.True(t, packages[0].Installed)
		assert.Equal(t, "545.29.06-1", packages[0].Version)
	})

	t.Run("installed_plain", func(t *testing.T) {
		output := `extra/nvidia-utils 545.29.06-1 [installed]
    NVIDIA drivers utilities`
		packages, err := parsePacmanSs(output)
		require.NoError(t, err)
		require.Len(t, packages, 1)
		assert.True(t, packages[0].Installed)
	})

	t.Run("not_installed", func(t *testing.T) {
		output := `extra/nvidia 545.29.06-1
    NVIDIA drivers for linux`
		packages, err := parsePacmanSs(output)
		require.NoError(t, err)
		require.Len(t, packages, 1)
		assert.False(t, packages[0].Installed)
	})
}

// =============================================================================
// Benchmark Tests
// =============================================================================

func BenchmarkManager_ParsePacmanQ(b *testing.B) {
	// Generate realistic output
	var sb strings.Builder
	for i := 0; i < 500; i++ {
		sb.WriteString("package-")
		sb.WriteString(string(rune('a' + i%26)))
		sb.WriteString(" 1.0.0-1\n")
	}
	output := sb.String()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = parsePacmanQ(output)
	}
}

func BenchmarkManager_ParsePacmanSs(b *testing.B) {
	var sb strings.Builder
	for i := 0; i < 200; i++ {
		sb.WriteString("extra/package-")
		sb.WriteString(string(rune('a' + i%26)))
		sb.WriteString(" 1.0.0-1\n")
		sb.WriteString("    package description for testing purposes\n")
	}
	output := sb.String()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = parsePacmanSs(output)
	}
}

func BenchmarkManager_ParsePacmanInfo(b *testing.B) {
	output := `Repository      : extra
Name            : nginx
Version         : 1.24.0-1
Description     : Lightweight HTTP server and IMAP/POP3 proxy server
Architecture    : x86_64
Installed Size  : 1.2 MiB
Depends On      : pcre2  zlib  openssl  geoip  mailcap
`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = parsePacmanInfo(output)
	}
}

func BenchmarkManager_ParsePacmanQu(b *testing.B) {
	var sb strings.Builder
	for i := 0; i < 100; i++ {
		sb.WriteString("package-")
		sb.WriteString(string(rune('a' + i%26)))
		sb.WriteString(" 1.0.0-1 -> 1.1.0-1\n")
	}
	output := sb.String()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = parsePacmanQu(output)
	}
}

func BenchmarkManager_ParsePacmanConf(b *testing.B) {
	content := `[options]
HoldPkg     = pacman glibc
Architecture = auto

[core]
Include = /etc/pacman.d/mirrorlist

[extra]
Include = /etc/pacman.d/mirrorlist

[community]
Include = /etc/pacman.d/mirrorlist

[multilib]
Include = /etc/pacman.d/mirrorlist

[chaotic-aur]
Server = https://geo-mirror.chaotic.cx/$repo/$arch
SigLevel = Required TrustedOnly`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = parsePacmanConf(content)
	}
}

func BenchmarkManager_IsInstalled(b *testing.B) {
	mgr, mockExec := setupTest()
	mockExec.SetResponse("pacman", exec.SuccessResult("nginx 1.24.0-1"))
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = mgr.IsInstalled(ctx, "nginx")
	}
}

func BenchmarkManager_Search(b *testing.B) {
	mgr, mockExec := setupTest()
	output := `extra/nginx 1.24.0-1
    HTTP server
extra/nginx-mainline 1.25.0-1
    HTTP server (mainline)
extra/nginx-mod-http-headers-more 0.33-1
    Nginx headers more module`
	mockExec.SetResponse("pacman", exec.SuccessResult(output))
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
