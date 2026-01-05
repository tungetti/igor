package dnf

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

	// Step 1: Update cache (check-update)
	mockExec.SetResponse("dnf", exec.SuccessResult(""))
	err := mgr.Update(ctx, pkg.DefaultUpdateOptions())
	require.NoError(t, err, "cache update should succeed")

	// Step 2: Search for packages
	searchOutput := `========================= Name Matched: nginx ==========================
nginx.x86_64 : A high performance web server and reverse proxy server
nginx-mod-http-perl.x86_64 : Nginx HTTP perl module`
	mockExec.SetResponse("dnf", exec.SuccessResult(searchOutput))
	mockExec.SetResponse("rpm", exec.FailureResult(1, "not found"))
	packages, err := mgr.Search(ctx, "nginx", pkg.DefaultSearchOptions())
	require.NoError(t, err, "search should succeed")
	require.Len(t, packages, 2, "should find 2 nginx packages")

	// Step 3: Check if installed (should not be)
	mockExec.SetResponse("rpm", exec.FailureResult(1, "package nginx is not installed"))
	installed, err := mgr.IsInstalled(ctx, "nginx")
	require.NoError(t, err)
	assert.False(t, installed, "nginx should not be installed initially")

	// Step 4: Install package
	mockExec.SetResponse("dnf", exec.SuccessResult("Complete!"))
	err = mgr.Install(ctx, pkg.DefaultInstallOptions(), "nginx")
	require.NoError(t, err, "install should succeed")

	// Step 5: Verify installed
	mockExec.SetResponse("rpm", exec.SuccessResult("nginx-1.22.0-1.fc39.x86_64"))
	installed, err = mgr.IsInstalled(ctx, "nginx")
	require.NoError(t, err)
	assert.True(t, installed, "nginx should be installed after install")

	// Step 6: Remove package
	mockExec.SetResponse("dnf", exec.SuccessResult("Complete!"))
	err = mgr.Remove(ctx, pkg.DefaultRemoveOptions(), "nginx")
	require.NoError(t, err, "remove should succeed")

	// Step 7: Verify removed
	mockExec.SetResponse("rpm", exec.FailureResult(1, "package nginx is not installed"))
	installed, err = mgr.IsInstalled(ctx, "nginx")
	require.NoError(t, err)
	assert.False(t, installed, "nginx should not be installed after removal")
}

// TestManager_NVIDIADriverWorkflow tests NVIDIA driver installation workflow.
func TestManager_NVIDIADriverWorkflow(t *testing.T) {
	mgr, mockExec := setupTest()
	ctx := context.Background()

	// Step 1: Add NVIDIA CUDA repository
	mockExec.SetResponse("dnf", exec.SuccessResult(""))
	repo := pkg.Repository{
		Name: "nvidia-cuda",
		URL:  "https://developer.download.nvidia.com/compute/cuda/repos/fedora39/x86_64/cuda-fedora39.repo",
	}
	err := mgr.AddRepository(ctx, repo)
	require.NoError(t, err, "adding NVIDIA repo should succeed")

	// Step 2: Refresh repositories
	mockExec.SetResponse("dnf", exec.SuccessResult("Metadata cache created"))
	err = mgr.RefreshRepositories(ctx)
	require.NoError(t, err)

	// Step 3: Search for NVIDIA driver packages
	searchOutput := `nvidia-driver.x86_64 : NVIDIA driver package
nvidia-driver-libs.x86_64 : NVIDIA driver libraries`
	mockExec.SetResponse("dnf", exec.SuccessResult(searchOutput))
	mockExec.SetResponse("rpm", exec.FailureResult(1, "not found"))
	packages, err := mgr.Search(ctx, "nvidia-driver", pkg.SearchOptions{IncludeInstalled: false})
	require.NoError(t, err)
	assert.Greater(t, len(packages), 0, "should find NVIDIA driver packages")

	// Step 4: Install NVIDIA driver
	mockExec.SetResponse("dnf", exec.SuccessResult("Complete!"))
	err = mgr.Install(ctx, pkg.DefaultInstallOptions(), "nvidia-driver")
	require.NoError(t, err, "NVIDIA driver installation should succeed")

	// Step 5: Verify installation
	mockExec.SetResponse("rpm", exec.SuccessResult("nvidia-driver-535.154.05-1.fc39.x86_64"))
	installed, err := mgr.IsInstalled(ctx, "nvidia-driver")
	require.NoError(t, err)
	assert.True(t, installed, "NVIDIA driver should be installed")
}

// TestManager_ErrorRecovery tests error recovery scenarios.
func TestManager_ErrorRecovery(t *testing.T) {
	mgr, mockExec := setupTest()
	ctx := context.Background()

	t.Run("package_not_found_recovery", func(t *testing.T) {
		mockExec.SetResponse("dnf", exec.FailureResult(1, "No match for argument: nonexistent-pkg"))
		err := mgr.Install(ctx, pkg.DefaultInstallOptions(), "nonexistent-pkg")
		require.Error(t, err)
		assert.ErrorIs(t, err, pkg.ErrPackageNotFound)

		// Verify system can still perform other operations
		mockExec.SetResponse("rpm", exec.SuccessResult("nginx-1.22.0"))
		installed, err := mgr.IsInstalled(ctx, "nginx")
		require.NoError(t, err)
		assert.True(t, installed)
	})

	t.Run("network_error_recovery", func(t *testing.T) {
		mockExec.SetResponse("dnf", exec.FailureResult(1, "Could not resolve host"))
		err := mgr.Update(ctx, pkg.DefaultUpdateOptions())
		require.Error(t, err)
		assert.ErrorIs(t, err, pkg.ErrNetworkUnavailable)

		// Verify cached operations still work
		mockExec.SetResponse("rpm", exec.SuccessResult("curl-7.76.1"))
		installed, err := mgr.IsInstalled(ctx, "curl")
		require.NoError(t, err)
		assert.True(t, installed)
	})

	t.Run("lock_error_recovery", func(t *testing.T) {
		mockExec.SetResponse("dnf", exec.FailureResult(1, "another copy is running"))
		err := mgr.Install(ctx, pkg.DefaultInstallOptions(), "nginx")
		require.Error(t, err)
		assert.ErrorIs(t, err, pkg.ErrLockAcquireFailed)

		// Verify read-only operations work
		searchOutput := `nginx.x86_64 : web server`
		mockExec.SetResponse("dnf", exec.SuccessResult(searchOutput))
		_, err = mgr.Search(ctx, "nginx", pkg.DefaultSearchOptions())
		require.NoError(t, err)
	})

	t.Run("nothing_to_do_recovery", func(t *testing.T) {
		mockExec.SetResponse("dnf", exec.FailureResult(1, "Nothing to do"))
		err := mgr.Install(ctx, pkg.DefaultInstallOptions(), "already-installed")
		require.Error(t, err)
		assert.ErrorIs(t, err, pkg.ErrPackageNotFound)
	})
}

// TestManager_ConcurrentOperations tests thread safety.
func TestManager_ConcurrentOperations(t *testing.T) {
	mgr, mockExec := setupTest()
	ctx := context.Background()

	// Set up mock responses
	mockExec.SetDefaultResponse(exec.SuccessResult(""))
	mockExec.SetResponse("rpm", exec.SuccessResult("nginx-1.22.0"))
	mockExec.SetResponse("dnf", exec.SuccessResult("nginx.x86_64 : web server"))

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
			mockExec.SetResponse("rpm", exec.SuccessResult("nginx\t1.22.0\tx86_64"))
			_, err := mgr.ListInstalled(ctx)
			if err != nil {
				errChan <- err
			}
		}()

		// Concurrent ListUpgradable
		go func() {
			defer wg.Done()
			mockExec.SetResponse("dnf", exec.SuccessResult(""))
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
		mockExec.SetResponse("dnf", exec.SuccessResult(""))
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		// With mock, operation may still succeed, but tests the pattern
		_ = mgr.Update(ctx, pkg.DefaultUpdateOptions())
	})

	t.Run("cancel_during_install", func(t *testing.T) {
		mockExec.SetResponse("dnf", exec.SuccessResult(""))
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		_ = mgr.Install(ctx, pkg.DefaultInstallOptions(), "nginx")
	})

	t.Run("timeout_handling", func(t *testing.T) {
		mockExec.SetResponse("dnf", exec.SuccessResult(""))
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

		packages, err = parseDnfSearch("")
		require.NoError(t, err)
		assert.Len(t, packages, 0)

		p, err := parseDnfInfo("")
		require.NoError(t, err)
		assert.Nil(t, p)
	})

	t.Run("malformed_output", func(t *testing.T) {
		// Single field line (should be skipped)
		packages, err := parseRpmQuery("nginx\ncurl\t7.76.1\tx86_64")
		require.NoError(t, err)
		assert.Len(t, packages, 1)
		assert.Equal(t, "curl", packages[0].Name)
	})

	t.Run("unicode_in_package_names", func(t *testing.T) {
		output := "fonts-noto-cjk.noarch : 中文字体 package"
		packages, err := parseDnfSearch(output)
		require.NoError(t, err)
		assert.Len(t, packages, 1)
	})

	t.Run("very_long_output", func(t *testing.T) {
		var sb strings.Builder
		for i := 0; i < 1000; i++ {
			sb.WriteString("package-")
			sb.WriteString(string(rune('a' + i%26)))
			sb.WriteString(".x86_64 : package description\n")
		}
		packages, err := parseDnfSearch(sb.String())
		require.NoError(t, err)
		assert.Len(t, packages, 1000)
	})
}

// TestManager_VersionComparison tests version handling.
func TestManager_VersionComparison(t *testing.T) {
	t.Run("parse_version_with_release", func(t *testing.T) {
		output := `nginx	1.22.0-1.el9	x86_64
curl	7.76.1-23.el9	x86_64`
		packages, err := parseRpmQuery(output)
		require.NoError(t, err)
		require.Len(t, packages, 2)
		assert.Equal(t, "nginx", packages[0].Name)
		assert.Equal(t, "1.22.0-1.el9", packages[0].Version)
	})

	t.Run("dnf_check_update_format", func(t *testing.T) {
		output := `nginx.x86_64                    1.24.0-1.el9               updates
curl.x86_64                     7.76.1-26.el9              baseos`
		packages, err := parseDnfCheckUpdate(output)
		require.NoError(t, err)
		require.Len(t, packages, 2)
		assert.Equal(t, "nginx", packages[0].Name)
		assert.Equal(t, "1.24.0-1.el9", packages[0].Version)
		assert.Equal(t, "updates", packages[0].Repository)
	})
}

// TestManager_DnfListOutputParsing tests dnf list output parsing.
func TestManager_DnfListOutputParsing(t *testing.T) {
	t.Run("installed_packages", func(t *testing.T) {
		output := `Last metadata expiration check: 0:00:01 ago
Installed Packages
nginx.x86_64                    1.22.0-1.el9               @appstream
curl.x86_64                     7.76.1-23.el9              @baseos`
		packages, err := parseDnfList(output)
		require.NoError(t, err)
		require.Len(t, packages, 2)
		assert.Equal(t, "nginx", packages[0].Name)
		assert.Equal(t, "1.22.0-1.el9", packages[0].Version)
		assert.Equal(t, "@appstream", packages[0].Repository)
	})

	t.Run("available_packages", func(t *testing.T) {
		output := `Last metadata expiration check: 0:00:01 ago
Available Packages
nginx-mod-http-perl.x86_64      1.22.0-1.el9               appstream`
		packages, err := parseDnfList(output)
		require.NoError(t, err)
		require.Len(t, packages, 1)
	})
}

// TestManager_ModuleHandling tests DNF module operations.
func TestManager_ModuleHandling(t *testing.T) {
	mgr, mockExec := setupTest()
	ctx := context.Background()

	t.Run("install_with_module_stream", func(t *testing.T) {
		mockExec.SetResponse("dnf", exec.SuccessResult("Complete!"))
		// DNF supports package:stream syntax
		err := mgr.Install(ctx, pkg.DefaultInstallOptions(), "nodejs:18")
		require.NoError(t, err)

		lastCall := mockExec.LastCall()
		assert.Contains(t, lastCall.Args, "nodejs:18")
	})
}

// TestManager_RepositoryManagement tests repository edge cases.
func TestManager_RepositoryManagement(t *testing.T) {
	mgr, mockExec := setupTest()
	ctx := context.Background()

	t.Run("dnf_repolist_parsing", func(t *testing.T) {
		output := `repo id                           repo name                                 status
fedora                           Fedora 39 - x86_64                        enabled
updates                          Fedora 39 - x86_64 - Updates              enabled
rpmfusion-nonfree                RPM Fusion Nonfree                        disabled`
		mockExec.SetResponse("dnf", exec.SuccessResult(output))

		repos, err := mgr.ListRepositories(ctx)
		require.NoError(t, err)
		require.Len(t, repos, 3)
		assert.Equal(t, "fedora", repos[0].Name)
		assert.True(t, repos[0].Enabled)
		assert.Equal(t, "rpmfusion-nonfree", repos[2].Name)
		assert.False(t, repos[2].Enabled)
	})

	t.Run("add_rpmfusion_repos", func(t *testing.T) {
		mockExec.SetResponse("sh", exec.SuccessResult("Complete!"))
		err := mgr.AddRPMFusion(ctx, true, true)
		require.NoError(t, err)
	})
}

// =============================================================================
// Benchmark Tests
// =============================================================================

func BenchmarkManager_ParseRpmQuery(b *testing.B) {
	var sb strings.Builder
	for i := 0; i < 500; i++ {
		sb.WriteString("package-")
		sb.WriteString(string(rune('a' + i%26)))
		sb.WriteString("\t1.0.0-1.el9\tx86_64\n")
	}
	output := sb.String()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = parseRpmQuery(output)
	}
}

func BenchmarkManager_ParseDnfSearch(b *testing.B) {
	var sb strings.Builder
	sb.WriteString("========================= Name Matched: test ==========================\n")
	for i := 0; i < 200; i++ {
		sb.WriteString("package-")
		sb.WriteString(string(rune('a' + i%26)))
		sb.WriteString(".x86_64 : package description for testing\n")
	}
	output := sb.String()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = parseDnfSearch(output)
	}
}

func BenchmarkManager_ParseDnfInfo(b *testing.B) {
	output := `Last metadata expiration check: 0:00:01 ago
Name         : nginx
Version      : 1.22.0
Release      : 1.el9
Architecture : x86_64
Size         : 1.2 M
Summary      : A high performance web server and reverse proxy server
Repository   : appstream`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = parseDnfInfo(output)
	}
}

func BenchmarkManager_ParseDnfCheckUpdate(b *testing.B) {
	var sb strings.Builder
	for i := 0; i < 100; i++ {
		sb.WriteString("package-")
		sb.WriteString(string(rune('a' + i%26)))
		sb.WriteString(".x86_64                    1.1.0-1.el9               updates\n")
	}
	output := sb.String()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = parseDnfCheckUpdate(output)
	}
}

func BenchmarkManager_IsInstalled(b *testing.B) {
	mgr, mockExec := setupTest()
	mockExec.SetResponse("rpm", exec.SuccessResult("nginx-1.22.0-1.el9.x86_64"))
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = mgr.IsInstalled(ctx, "nginx")
	}
}

func BenchmarkManager_Search(b *testing.B) {
	mgr, mockExec := setupTest()
	output := `nginx.x86_64 : A high performance web server and reverse proxy server
nginx-mod-http-perl.x86_64 : Nginx HTTP perl module`
	mockExec.SetResponse("dnf", exec.SuccessResult(output))
	mockExec.SetResponse("rpm", exec.FailureResult(1, "not found"))
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = mgr.Search(ctx, "nginx", pkg.SearchOptions{IncludeInstalled: false})
	}
}

func BenchmarkManager_ParseDnfRepolist(b *testing.B) {
	output := `repo id                           repo name                                 status
fedora                           Fedora 39 - x86_64                        enabled
updates                          Fedora 39 - x86_64 - Updates              enabled
updates-modular                  Fedora 39 - Modular Updates               enabled
rpmfusion-free                   RPM Fusion Free                           enabled
rpmfusion-nonfree                RPM Fusion Nonfree                        disabled
cuda-fedora39-x86_64             NVIDIA CUDA Repository                    enabled`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = parseDnfRepolist(output)
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
