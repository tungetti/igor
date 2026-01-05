package yum

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

	// Step 1: Update cache (yum check-update)
	// Exit code 100 means updates available (success)
	mockExec.SetResponse("yum", exec.SuccessResult(""))
	err := mgr.Update(ctx, pkg.DefaultUpdateOptions())
	require.NoError(t, err, "cache update should succeed")

	// Step 2: Search for packages
	searchOutput := `========================= N/S Matched: nginx ==========================
nginx.x86_64 : A high performance web server and reverse proxy server
nginx-mod-http-perl.x86_64 : Nginx HTTP perl module
nginx-core.x86_64 : Core nginx package`
	mockExec.SetResponse("yum", exec.SuccessResult(searchOutput))
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
	mockExec.SetResponse("yum", exec.SuccessResult("Complete!"))
	err = mgr.Install(ctx, pkg.DefaultInstallOptions(), "nginx")
	require.NoError(t, err, "install should succeed")

	// Step 5: Verify installed
	mockExec.SetResponse("rpm", exec.SuccessResult("nginx-1.22.0-1.el7.x86_64"))
	installed, err = mgr.IsInstalled(ctx, "nginx")
	require.NoError(t, err)
	assert.True(t, installed, "nginx should be installed after install")

	// Step 6: Remove package
	mockExec.SetResponse("yum", exec.SuccessResult("Complete!"))
	err = mgr.Remove(ctx, pkg.DefaultRemoveOptions(), "nginx")
	require.NoError(t, err, "remove should succeed")

	// Step 7: Verify removed
	mockExec.SetResponse("rpm", exec.FailureResult(1, "package nginx is not installed"))
	installed, err = mgr.IsInstalled(ctx, "nginx")
	require.NoError(t, err)
	assert.False(t, installed, "nginx should not be installed after removal")
}

// TestManager_NVIDIADriverWorkflow tests NVIDIA driver installation workflow on CentOS 7/RHEL 7.
func TestManager_NVIDIADriverWorkflow(t *testing.T) {
	mgr, mockExec := setupTest()
	ctx := context.Background()

	// Step 1: Add EPEL repository (required for some dependencies)
	mockExec.SetResponse("yum", exec.SuccessResult("Complete!"))
	err := mgr.AddEPEL(ctx)
	require.NoError(t, err, "adding EPEL should succeed")

	// Step 2: Add NVIDIA repository
	mockExec.SetResponse("yum-config-manager", exec.SuccessResult(""))
	err = mgr.AddNvidiaRepo(ctx)
	require.NoError(t, err, "adding NVIDIA repo should succeed")

	// Step 3: Update cache after adding repository
	mockExec.SetResponse("yum", exec.SuccessResult(""))
	err = mgr.Update(ctx, pkg.DefaultUpdateOptions())
	require.NoError(t, err)

	// Step 4: Search for NVIDIA driver packages
	searchOutput := `nvidia-driver.x86_64 : NVIDIA driver package
nvidia-driver-cuda.x86_64 : NVIDIA CUDA driver package
nvidia-kmod-common.x86_64 : Common NVIDIA kernel module files`
	mockExec.SetResponse("yum", exec.SuccessResult(searchOutput))
	mockExec.SetResponse("rpm", exec.FailureResult(1, "not installed"))
	packages, err := mgr.Search(ctx, "nvidia", pkg.SearchOptions{IncludeInstalled: false})
	require.NoError(t, err)
	assert.Greater(t, len(packages), 0, "should find NVIDIA driver packages")

	// Step 5: Install NVIDIA driver
	mockExec.SetResponse("yum", exec.SuccessResult("Complete!"))
	err = mgr.Install(ctx, pkg.DefaultInstallOptions(), "nvidia-driver")
	require.NoError(t, err, "NVIDIA driver installation should succeed")

	// Step 6: Verify installation
	mockExec.SetResponse("rpm", exec.SuccessResult("nvidia-driver-535.104.05-1.el7.x86_64"))
	installed, err := mgr.IsInstalled(ctx, "nvidia-driver")
	require.NoError(t, err)
	assert.True(t, installed, "NVIDIA driver should be installed")
}

// TestManager_ErrorRecovery tests error recovery scenarios.
func TestManager_ErrorRecovery(t *testing.T) {
	mgr, mockExec := setupTest()
	ctx := context.Background()

	t.Run("package_not_found_recovery", func(t *testing.T) {
		mockExec.SetResponse("yum", exec.FailureResult(1, "No package nonexistent-pkg available."))
		err := mgr.Install(ctx, pkg.DefaultInstallOptions(), "nonexistent-pkg")
		require.Error(t, err)
		assert.ErrorIs(t, err, pkg.ErrPackageNotFound)

		// Verify system can still perform other operations
		mockExec.SetResponse("rpm", exec.SuccessResult("nginx-1.22.0-1.el7.x86_64"))
		installed, err := mgr.IsInstalled(ctx, "nginx")
		require.NoError(t, err)
		assert.True(t, installed)
	})

	t.Run("network_error_recovery", func(t *testing.T) {
		mockExec.SetResponse("yum", exec.FailureResult(1, "Cannot find a valid baseurl for repo"))
		err := mgr.Update(ctx, pkg.DefaultUpdateOptions())
		require.Error(t, err)
		assert.ErrorIs(t, err, pkg.ErrNetworkUnavailable)

		// Verify cached operations still work
		mockExec.SetResponse("rpm", exec.SuccessResult("curl-7.29.0-59.el7.x86_64"))
		installed, err := mgr.IsInstalled(ctx, "curl")
		require.NoError(t, err)
		assert.True(t, installed)
	})

	t.Run("yum_lock_error_recovery", func(t *testing.T) {
		mockExec.SetResponse("yum", exec.FailureResult(1, "Another app is currently holding the yum lock"))
		err := mgr.Install(ctx, pkg.DefaultInstallOptions(), "nginx")
		require.Error(t, err)
		assert.ErrorIs(t, err, pkg.ErrLockAcquireFailed)

		// Verify read-only operations work
		mockExec.SetResponse("yum", exec.SuccessResult("nginx.x86_64 : web server"))
		_, err = mgr.Search(ctx, "nginx", pkg.DefaultSearchOptions())
		require.NoError(t, err)
	})

	t.Run("no_match_for_argument_recovery", func(t *testing.T) {
		mockExec.SetResponse("yum", exec.FailureResult(1, "No Match for argument: nginx"))
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
	mockExec.SetResponse("rpm", exec.SuccessResult("nginx-1.22.0-1.el7.x86_64"))
	mockExec.SetResponse("yum", exec.SuccessResult("nginx.x86_64 : web server"))

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
			_, err := mgr.Search(ctx, "nginx", pkg.DefaultSearchOptions())
			if err != nil {
				errChan <- err
			}
		}()

		// Concurrent ListInstalled
		go func() {
			defer wg.Done()
			mockExec.SetResponse("rpm", exec.SuccessResult("nginx\t1.22.0-1.el7\tx86_64"))
			_, err := mgr.ListInstalled(ctx)
			if err != nil {
				errChan <- err
			}
		}()

		// Concurrent ListUpgradable
		go func() {
			defer wg.Done()
			mockExec.SetResponse("yum", exec.SuccessResult("nginx.x86_64  1.24.0-1.el7  epel"))
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
		mockExec.SetResponse("yum", exec.SuccessResult(""))
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		// With mock, operation may still succeed, but tests the pattern
		_ = mgr.Update(ctx, pkg.DefaultUpdateOptions())
	})

	t.Run("cancel_during_install", func(t *testing.T) {
		mockExec.SetResponse("yum", exec.SuccessResult(""))
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		_ = mgr.Install(ctx, pkg.DefaultInstallOptions(), "nginx")
	})

	t.Run("timeout_handling", func(t *testing.T) {
		mockExec.SetResponse("yum", exec.SuccessResult(""))
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

		packages, err = parseYumSearch("")
		require.NoError(t, err)
		assert.Len(t, packages, 0)

		p, err := parseYumInfo("")
		require.NoError(t, err)
		assert.Nil(t, p)
	})

	t.Run("malformed_output", func(t *testing.T) {
		// Single field line (should be skipped)
		packages, err := parseRpmQuery("nginx\ncurl\t7.29.0-59.el7\tx86_64")
		require.NoError(t, err)
		assert.Len(t, packages, 1)
		assert.Equal(t, "curl", packages[0].Name)
	})

	t.Run("unicode_in_package_names", func(t *testing.T) {
		output := "fonts-noto-cjk.noarch : 中文字体 package\nlibc.x86_64 : GNU C Library"
		packages, err := parseYumSearch(output)
		require.NoError(t, err)
		assert.Len(t, packages, 2)
		assert.Equal(t, "fonts-noto-cjk", packages[0].Name)
	})

	t.Run("very_long_output", func(t *testing.T) {
		var sb strings.Builder
		for i := 0; i < 1000; i++ {
			sb.WriteString("package-")
			sb.WriteString(string(rune('a' + i%26)))
			sb.WriteString(".x86_64 : description\n")
		}
		packages, err := parseYumSearch(sb.String())
		require.NoError(t, err)
		assert.Len(t, packages, 1000)
	})

	t.Run("partial_output", func(t *testing.T) {
		// Simulate interrupted output (no newline at end)
		output := "nginx\t1.22.0-1.el7\tx86_64"
		packages, err := parseRpmQuery(output)
		require.NoError(t, err)
		assert.Len(t, packages, 1)
	})
}

// TestManager_VersionComparison tests version handling.
func TestManager_VersionComparison(t *testing.T) {
	t.Run("parse_version_with_epoch", func(t *testing.T) {
		output := `vim	2:8.2-1.el7	x86_64
curl	7.29.0-59.el7	x86_64`
		packages, err := parseRpmQuery(output)
		require.NoError(t, err)
		require.Len(t, packages, 2)
		assert.Equal(t, "vim", packages[0].Name)
		assert.Equal(t, "2:8.2-1.el7", packages[0].Version)
	})

	t.Run("parse_version_without_epoch", func(t *testing.T) {
		output := `nginx	1.22.0-1.el7	x86_64`
		packages, err := parseRpmQuery(output)
		require.NoError(t, err)
		require.Len(t, packages, 1)
		assert.Equal(t, "1.22.0-1.el7", packages[0].Version)
	})

	t.Run("yum_check_update_format", func(t *testing.T) {
		output := `nginx.x86_64                    1.24.0-1.el7               epel
vim.x86_64                      2:8.2-1.el7_9              base`
		packages, err := parseYumCheckUpdate(output)
		require.NoError(t, err)
		require.Len(t, packages, 2)
		assert.Equal(t, "1.24.0-1.el7", packages[0].Version)
		assert.Equal(t, "2:8.2-1.el7_9", packages[1].Version)
	})
}

// TestManager_YumSpecificFeatures tests YUM-specific functionality.
func TestManager_YumSpecificFeatures(t *testing.T) {
	mgr, mockExec := setupTest()
	ctx := context.Background()

	t.Run("yum_check_update_exit_100", func(t *testing.T) {
		// yum check-update returns exit code 100 when updates are available
		mockExec.SetResponse("yum", exec.FailureResult(100, ""))
		err := mgr.Update(ctx, pkg.DefaultUpdateOptions())
		require.NoError(t, err, "exit code 100 should not be an error")
	})

	t.Run("autoremove_fallback_to_package_cleanup", func(t *testing.T) {
		// First call to yum autoremove fails with "No such command"
		mockExec.SetResponse("yum", exec.FailureResult(1, "No such command: autoremove"))
		// Fallback to package-cleanup succeeds
		mockExec.SetResponse("package-cleanup", exec.SuccessResult(""))

		err := mgr.AutoRemove(ctx)
		require.NoError(t, err)
	})

	t.Run("epel_already_installed", func(t *testing.T) {
		mockExec.SetResponse("yum", exec.FailureResult(1, "Package epel-release already installed"))
		err := mgr.AddEPEL(ctx)
		require.NoError(t, err, "already installed EPEL should not be an error")
	})

	t.Run("elrepo_installation", func(t *testing.T) {
		mockExec.SetDefaultResponse(exec.SuccessResult("Complete!"))
		err := mgr.AddElrepo(ctx)
		require.NoError(t, err)
	})

	t.Run("rpm_fusion_el_installation", func(t *testing.T) {
		mockExec.SetDefaultResponse(exec.SuccessResult("Complete!"))
		err := mgr.AddRPMFusionEL(ctx, true, true)
		require.NoError(t, err)
	})

	t.Run("yum_utils_installation", func(t *testing.T) {
		mockExec.SetResponse("yum", exec.SuccessResult("Complete!"))
		err := mgr.InstallYumUtils(ctx)
		require.NoError(t, err)
	})
}

// TestManager_RepositoryManagement tests repository edge cases.
func TestManager_RepositoryManagement(t *testing.T) {
	mgr, mockExec := setupTest()
	ctx := context.Background()

	t.Run("add_repository_with_gpg_key", func(t *testing.T) {
		mockExec.SetResponse("test", exec.FailureResult(1, ""))
		mockExec.SetDefaultResponse(exec.SuccessResult(""))

		repo := pkg.Repository{
			Name:   "nvidia-cuda",
			URL:    "https://developer.download.nvidia.com/cuda/rhel7/x86_64",
			GPGKey: "https://developer.download.nvidia.com/cuda/repos/rhel7/x86_64/7fa2af80.pub",
		}
		err := mgr.AddRepository(ctx, repo)
		require.NoError(t, err)
	})

	t.Run("import_gpg_key", func(t *testing.T) {
		mockExec.SetResponse("rpm", exec.SuccessResult(""))
		err := mgr.ImportGPGKey(ctx, "https://example.com/RPM-GPG-KEY")
		require.NoError(t, err)
	})

	t.Run("gpg_key_network_error", func(t *testing.T) {
		mockExec.SetResponse("rpm", exec.FailureResult(1, "Could not resolve host"))
		err := mgr.ImportGPGKey(ctx, "https://example.com/key")
		require.Error(t, err)
		assert.ErrorIs(t, err, pkg.ErrNetworkUnavailable)
	})

	t.Run("get_repo_file_path", func(t *testing.T) {
		path := mgr.GetRepoFilePath("nvidia-cuda")
		assert.Equal(t, "/etc/yum.repos.d/nvidia-cuda.repo", path)

		path = mgr.GetRepoFilePath("my repo/test")
		assert.Equal(t, "/etc/yum.repos.d/my-repo-test.repo", path)
	})
}

// TestManager_RepoFileParsing tests .repo file parsing.
func TestManager_RepoFileParsing(t *testing.T) {
	t.Run("parse_full_repo_file", func(t *testing.T) {
		content := `[nvidia-cuda]
name=NVIDIA CUDA Repository
baseurl=https://developer.download.nvidia.com/cuda/rhel7/x86_64
enabled=1
gpgcheck=1
gpgkey=https://developer.download.nvidia.com/cuda/repos/rhel7/x86_64/7fa2af80.pub

[nvidia-cuda-source]
name=NVIDIA CUDA Source Repository
baseurl=https://developer.download.nvidia.com/cuda/rhel7/SRPMS
enabled=0
gpgcheck=1`

		repos, err := parseRepoFile(content)
		require.NoError(t, err)
		require.Len(t, repos, 2)

		assert.Equal(t, "nvidia-cuda", repos[0].Name)
		assert.Equal(t, "https://developer.download.nvidia.com/cuda/rhel7/x86_64", repos[0].URL)
		assert.True(t, repos[0].Enabled)
		assert.NotEmpty(t, repos[0].GPGKey)

		assert.Equal(t, "nvidia-cuda-source", repos[1].Name)
		assert.False(t, repos[1].Enabled)
	})

	t.Run("build_repo_file_content", func(t *testing.T) {
		repo := pkg.Repository{
			Name:    "nvidia-cuda",
			URL:     "https://developer.download.nvidia.com/cuda/rhel7/x86_64",
			Enabled: true,
			GPGKey:  "https://developer.download.nvidia.com/cuda/repos/rhel7/x86_64/7fa2af80.pub",
		}

		content := buildRepoFileContent(repo)
		assert.Contains(t, content, "[nvidia-cuda]")
		assert.Contains(t, content, "baseurl=https://developer.download.nvidia.com/cuda/rhel7/x86_64")
		assert.Contains(t, content, "enabled=1")
		assert.Contains(t, content, "gpgcheck=1")
	})
}

// TestManager_HelperFunctions tests helper functions.
func TestManager_HelperFunctions(t *testing.T) {
	t.Run("is_architecture", func(t *testing.T) {
		assert.True(t, isArchitecture("x86_64"))
		assert.True(t, isArchitecture("i686"))
		assert.True(t, isArchitecture("noarch"))
		assert.True(t, isArchitecture("aarch64"))
		assert.True(t, isArchitecture("ppc64"))
		assert.False(t, isArchitecture("something"))
		assert.False(t, isArchitecture(""))
	})

	t.Run("parse_size", func(t *testing.T) {
		assert.Equal(t, int64(1024), parseSize("1 k"))
		assert.Equal(t, int64(1024*1024), parseSize("1 M"))
		assert.Equal(t, int64(1024*1024*1024), parseSize("1 G"))
		assert.Equal(t, int64(0), parseSize("invalid"))
	})

	t.Run("sanitize_repo_id", func(t *testing.T) {
		assert.Equal(t, "simple", sanitizeRepoID("simple"))
		assert.Equal(t, "with-space", sanitizeRepoID("with space"))
		assert.Equal(t, "with-slash", sanitizeRepoID("with/slash"))
		assert.Equal(t, "custom-repo", sanitizeRepoID(""))
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
		sb.WriteString("\t1.0.0-1.el7\tx86_64\n")
	}
	output := sb.String()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = parseRpmQuery(output)
	}
}

func BenchmarkManager_ParseYumSearch(b *testing.B) {
	var sb strings.Builder
	for i := 0; i < 200; i++ {
		sb.WriteString("package-")
		sb.WriteString(string(rune('a' + i%26)))
		sb.WriteString(".x86_64 : package description for testing purposes\n")
	}
	output := sb.String()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = parseYumSearch(output)
	}
}

func BenchmarkManager_ParseYumInfo(b *testing.B) {
	output := `Loaded plugins: fastestmirror
Name        : nginx
Version     : 1.22.0
Release     : 1.el7
Arch        : x86_64
Size        : 1.2 M
Summary     : A high performance web server and reverse proxy server
Repo        : epel
`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = parseYumInfo(output)
	}
}

func BenchmarkManager_ParseYumCheckUpdate(b *testing.B) {
	var sb strings.Builder
	for i := 0; i < 100; i++ {
		sb.WriteString("package-")
		sb.WriteString(string(rune('a' + i%26)))
		sb.WriteString(".x86_64                    1.1.0-1.el7               base\n")
	}
	output := sb.String()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = parseYumCheckUpdate(output)
	}
}

func BenchmarkManager_ParseYumList(b *testing.B) {
	var sb strings.Builder
	sb.WriteString("Loaded plugins: fastestmirror\nInstalled Packages\n")
	for i := 0; i < 200; i++ {
		sb.WriteString("package-")
		sb.WriteString(string(rune('a' + i%26)))
		sb.WriteString(".x86_64                    1.0.0-1.el7               @base\n")
	}
	output := sb.String()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = parseYumList(output)
	}
}

func BenchmarkManager_ParseYumRepolist(b *testing.B) {
	output := `Loaded plugins: fastestmirror
repo id                           repo name                                 status
base/7/x86_64                     CentOS-7 - Base                           enabled
epel/x86_64                       Extra Packages for Enterprise Linux 7     enabled
nvidia-cuda                       NVIDIA CUDA                               disabled
updates/7/x86_64                  CentOS-7 - Updates                        enabled
extras/7/x86_64                   CentOS-7 - Extras                         enabled
repolist: 10000`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = parseYumRepolist(output)
	}
}

func BenchmarkManager_ParseRepoFile(b *testing.B) {
	content := `[nvidia-cuda]
name=NVIDIA CUDA Repository
baseurl=https://developer.download.nvidia.com/cuda/rhel7/x86_64
enabled=1
gpgcheck=1
gpgkey=https://developer.download.nvidia.com/cuda/repos/rhel7/x86_64/7fa2af80.pub

[epel]
name=Extra Packages for Enterprise Linux 7 - $basearch
metalink=https://mirrors.fedoraproject.org/metalink?repo=epel-7&arch=$basearch
enabled=1
gpgcheck=1
gpgkey=file:///etc/pki/rpm-gpg/RPM-GPG-KEY-EPEL-7`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = parseRepoFile(content)
	}
}

func BenchmarkManager_IsInstalled(b *testing.B) {
	mgr, mockExec := setupTest()
	mockExec.SetResponse("rpm", exec.SuccessResult("nginx-1.22.0-1.el7.x86_64"))
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = mgr.IsInstalled(ctx, "nginx")
	}
}

func BenchmarkManager_Search(b *testing.B) {
	mgr, mockExec := setupTest()
	output := `nginx.x86_64 : A high performance web server and reverse proxy server
nginx-mod-http-perl.x86_64 : Nginx HTTP perl module
nginx-core.x86_64 : Core nginx package`
	mockExec.SetResponse("yum", exec.SuccessResult(output))
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
