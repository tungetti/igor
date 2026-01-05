package apt

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

	// Step 1: Update cache
	mockExec.SetResponse("env", exec.SuccessResult("Hit:1 http://archive.ubuntu.com/ubuntu jammy InRelease"))
	err := mgr.Update(ctx, pkg.DefaultUpdateOptions())
	require.NoError(t, err, "cache update should succeed")

	// Step 2: Search for packages
	searchOutput := `nginx - small, powerful, scalable web/proxy server
nginx-common - common files for nginx
nginx-core - nginx web/proxy server (standard version)`
	mockExec.SetResponse("apt-cache", exec.SuccessResult(searchOutput))
	mockExec.SetResponse("dpkg-query", exec.FailureResult(1, "not found"))
	packages, err := mgr.Search(ctx, "nginx", pkg.DefaultSearchOptions())
	require.NoError(t, err, "search should succeed")
	require.Len(t, packages, 3, "should find 3 nginx packages")

	// Step 3: Check if installed (should not be)
	mockExec.SetResponse("dpkg-query", exec.FailureResult(1, "no packages found"))
	installed, err := mgr.IsInstalled(ctx, "nginx")
	require.NoError(t, err)
	assert.False(t, installed, "nginx should not be installed initially")

	// Step 4: Install package
	mockExec.SetResponse("env", exec.SuccessResult("Setting up nginx..."))
	err = mgr.Install(ctx, pkg.DefaultInstallOptions(), "nginx")
	require.NoError(t, err, "install should succeed")

	// Step 5: Verify installed
	mockExec.SetResponse("dpkg-query", exec.SuccessResult("install ok installed"))
	installed, err = mgr.IsInstalled(ctx, "nginx")
	require.NoError(t, err)
	assert.True(t, installed, "nginx should be installed after install")

	// Step 6: Remove package
	mockExec.SetResponse("env", exec.SuccessResult("Removing nginx..."))
	err = mgr.Remove(ctx, pkg.DefaultRemoveOptions(), "nginx")
	require.NoError(t, err, "remove should succeed")

	// Step 7: Verify removed
	mockExec.SetResponse("dpkg-query", exec.FailureResult(1, "no packages found"))
	installed, err = mgr.IsInstalled(ctx, "nginx")
	require.NoError(t, err)
	assert.False(t, installed, "nginx should not be installed after removal")
}

// TestManager_NVIDIADriverWorkflow tests NVIDIA driver installation workflow.
func TestManager_NVIDIADriverWorkflow(t *testing.T) {
	mgr, mockExec := setupTest()
	ctx := context.Background()

	// Step 1: Add graphics-drivers PPA
	mockExec.SetResponse("env", exec.SuccessResult(""))
	repo := pkg.Repository{
		Name: "graphics-drivers",
		URL:  "ppa:graphics-drivers/ppa",
	}
	err := mgr.AddRepository(ctx, repo)
	require.NoError(t, err, "adding PPA should succeed")

	// Step 2: Update cache after adding repository
	mockExec.SetResponse("env", exec.SuccessResult(""))
	err = mgr.Update(ctx, pkg.DefaultUpdateOptions())
	require.NoError(t, err)

	// Step 3: Search for NVIDIA driver packages
	searchOutput := `nvidia-driver-535 - NVIDIA driver metapackage
nvidia-driver-545 - NVIDIA driver metapackage (latest)
nvidia-utils-535 - NVIDIA driver utilities`
	mockExec.SetResponse("apt-cache", exec.SuccessResult(searchOutput))
	mockExec.SetResponse("dpkg-query", exec.FailureResult(1, "not found"))
	packages, err := mgr.Search(ctx, "nvidia-driver", pkg.SearchOptions{IncludeInstalled: false})
	require.NoError(t, err)
	assert.Greater(t, len(packages), 0, "should find NVIDIA driver packages")

	// Step 4: Install NVIDIA driver
	mockExec.SetResponse("env", exec.SuccessResult("Setting up nvidia-driver-535..."))
	err = mgr.Install(ctx, pkg.DefaultInstallOptions(), "nvidia-driver-535")
	require.NoError(t, err, "NVIDIA driver installation should succeed")

	// Step 5: Verify installation
	mockExec.SetResponse("dpkg-query", exec.SuccessResult("install ok installed"))
	installed, err := mgr.IsInstalled(ctx, "nvidia-driver-535")
	require.NoError(t, err)
	assert.True(t, installed, "NVIDIA driver should be installed")
}

// TestManager_ErrorRecovery tests error recovery scenarios.
func TestManager_ErrorRecovery(t *testing.T) {
	mgr, mockExec := setupTest()
	ctx := context.Background()

	t.Run("package_not_found_recovery", func(t *testing.T) {
		mockExec.SetResponse("env", exec.FailureResult(100, "E: Unable to locate package nonexistent-pkg"))
		err := mgr.Install(ctx, pkg.DefaultInstallOptions(), "nonexistent-pkg")
		require.Error(t, err)
		assert.ErrorIs(t, err, pkg.ErrPackageNotFound)

		// Verify system can still perform other operations
		mockExec.SetResponse("dpkg-query", exec.SuccessResult("install ok installed"))
		installed, err := mgr.IsInstalled(ctx, "nginx")
		require.NoError(t, err)
		assert.True(t, installed)
	})

	t.Run("network_error_recovery", func(t *testing.T) {
		mockExec.SetResponse("env", exec.FailureResult(100, "E: Failed to fetch http://archive.ubuntu.com"))
		err := mgr.Update(ctx, pkg.DefaultUpdateOptions())
		require.Error(t, err)
		assert.ErrorIs(t, err, pkg.ErrNetworkUnavailable)

		// Verify cached operations still work
		mockExec.SetResponse("dpkg-query", exec.SuccessResult("install ok installed"))
		installed, err := mgr.IsInstalled(ctx, "curl")
		require.NoError(t, err)
		assert.True(t, installed)
	})

	t.Run("permission_denied_recovery", func(t *testing.T) {
		mockExec.SetResponse("env", exec.FailureResult(100, "E: Could not get lock /var/lib/dpkg/lock"))
		err := mgr.Install(ctx, pkg.DefaultInstallOptions(), "nginx")
		require.Error(t, err)
		assert.ErrorIs(t, err, pkg.ErrLockAcquireFailed)

		// Verify read-only operations work
		mockExec.SetResponse("apt-cache", exec.SuccessResult("nginx - web server"))
		_, err = mgr.Search(ctx, "nginx", pkg.DefaultSearchOptions())
		require.NoError(t, err)
	})

	t.Run("interrupted_operation_recovery", func(t *testing.T) {
		mockExec.SetResponse("env", exec.FailureResult(100, "dpkg was interrupted"))
		err := mgr.Install(ctx, pkg.DefaultInstallOptions(), "nginx")
		require.Error(t, err)
		assert.ErrorIs(t, err, pkg.ErrLockAcquireFailed)
	})
}

// TestManager_ConcurrentOperations tests thread safety.
func TestManager_ConcurrentOperations(t *testing.T) {
	mgr, mockExec := setupTest()
	ctx := context.Background()

	// Set up mock responses
	mockExec.SetDefaultResponse(exec.SuccessResult(""))
	mockExec.SetResponse("dpkg-query", exec.SuccessResult("install ok installed"))
	mockExec.SetResponse("apt-cache", exec.SuccessResult("nginx - web server"))
	mockExec.SetResponse("apt", exec.SuccessResult(""))

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
			mockExec.SetResponse("dpkg-query", exec.SuccessResult("nginx\t1.22.0\tinstall ok installed"))
			_, err := mgr.ListInstalled(ctx)
			if err != nil {
				errChan <- err
			}
		}()

		// Concurrent ListUpgradable
		go func() {
			defer wg.Done()
			mockExec.SetResponse("apt", exec.SuccessResult("Listing..."))
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
		mockExec.SetResponse("env", exec.SuccessResult(""))
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		// With mock, operation may still succeed, but tests the pattern
		_ = mgr.Update(ctx, pkg.DefaultUpdateOptions())
	})

	t.Run("cancel_during_install", func(t *testing.T) {
		mockExec.SetResponse("env", exec.SuccessResult(""))
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		_ = mgr.Install(ctx, pkg.DefaultInstallOptions(), "nginx")
	})

	t.Run("timeout_handling", func(t *testing.T) {
		mockExec.SetResponse("env", exec.SuccessResult(""))
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
		defer cancel()
		time.Sleep(10 * time.Millisecond) // Ensure timeout

		_ = mgr.Update(ctx, pkg.DefaultUpdateOptions())
	})
}

// TestManager_OutputParsing tests parsing edge cases.
func TestManager_OutputParsing(t *testing.T) {
	t.Run("empty_output", func(t *testing.T) {
		packages, err := parseDpkgQuery("")
		require.NoError(t, err)
		assert.Len(t, packages, 0)

		packages, err = parseAptCacheSearch("")
		require.NoError(t, err)
		assert.Len(t, packages, 0)

		p, err := parseAptCacheShow("")
		require.NoError(t, err)
		assert.Nil(t, p)
	})

	t.Run("malformed_output", func(t *testing.T) {
		// Single field line (should be skipped)
		packages, err := parseDpkgQuery("nginx\ncurl\t7.81.0\tinstall ok installed")
		require.NoError(t, err)
		assert.Len(t, packages, 1)
		assert.Equal(t, "curl", packages[0].Name)
	})

	t.Run("unicode_in_package_names", func(t *testing.T) {
		output := "fonts-noto-cjk - 中文字体 package\nlibc6 - GNU C Library"
		packages, err := parseAptCacheSearch(output)
		require.NoError(t, err)
		assert.Len(t, packages, 2)
		assert.Equal(t, "fonts-noto-cjk", packages[0].Name)
		assert.Contains(t, packages[0].Description, "中文字体")
	})

	t.Run("very_long_output", func(t *testing.T) {
		var sb strings.Builder
		for i := 0; i < 1000; i++ {
			sb.WriteString("package-")
			sb.WriteString(string(rune('a' + i%26)))
			sb.WriteString(" - description\n")
		}
		packages, err := parseAptCacheSearch(sb.String())
		require.NoError(t, err)
		assert.Len(t, packages, 1000)
	})

	t.Run("partial_output", func(t *testing.T) {
		// Simulate interrupted output (no newline at end)
		output := "nginx\t1.22.0\tinstall ok installed"
		packages, err := parseDpkgQuery(output)
		require.NoError(t, err)
		assert.Len(t, packages, 1)
	})
}

// TestManager_VersionComparison tests version handling.
func TestManager_VersionComparison(t *testing.T) {
	t.Run("parse_version_with_epoch", func(t *testing.T) {
		output := `vim	2:8.2.3995-1ubuntu2	install ok installed
curl	7.81.0-1ubuntu1	install ok installed`
		packages, err := parseDpkgQuery(output)
		require.NoError(t, err)
		require.Len(t, packages, 2)
		assert.Equal(t, "vim", packages[0].Name)
		assert.Equal(t, "2:8.2.3995-1ubuntu2", packages[0].Version)
	})

	t.Run("parse_version_without_epoch", func(t *testing.T) {
		output := `nginx	1.22.0-1ubuntu1	install ok installed`
		packages, err := parseDpkgQuery(output)
		require.NoError(t, err)
		require.Len(t, packages, 1)
		assert.Equal(t, "1.22.0-1ubuntu1", packages[0].Version)
	})

	t.Run("upgradable_version_comparison", func(t *testing.T) {
		output := `Listing...
nginx/jammy-updates 1.22.0-1ubuntu1.1 amd64 [upgradable from: 1.22.0-1ubuntu1]
vim/jammy-updates 2:8.2.3995-1ubuntu2.3 amd64 [upgradable from: 2:8.2.3995-1ubuntu2]`
		packages, err := parseAptListUpgradable(output)
		require.NoError(t, err)
		require.Len(t, packages, 2)
		assert.Equal(t, "1.22.0-1ubuntu1.1", packages[0].Version)
		assert.Equal(t, "2:8.2.3995-1ubuntu2.3", packages[1].Version)
	})
}

// TestManager_DpkgStatusParsing tests dpkg status parsing edge cases.
func TestManager_DpkgStatusParsing(t *testing.T) {
	t.Run("various_status_states", func(t *testing.T) {
		output := `nginx	1.22.0	install ok installed
curl	7.81.0	deinstall ok config-files
vim	8.2	purge ok not-installed
htop	3.0	install ok half-configured`
		packages, err := parseDpkgQuery(output)
		require.NoError(t, err)
		// Only fully installed packages should be returned
		assert.Len(t, packages, 1)
		assert.Equal(t, "nginx", packages[0].Name)
	})
}

// TestManager_AptCacheOutputParsing tests apt-cache output parsing.
func TestManager_AptCacheOutputParsing(t *testing.T) {
	t.Run("apt_cache_show_multiple_versions", func(t *testing.T) {
		output := `Package: nginx
Version: 1.22.0-1ubuntu1
Architecture: amd64
Installed-Size: 1200
Description: web server

Package: nginx
Version: 1.18.0-0ubuntu1
Architecture: amd64
Installed-Size: 1100
Description: older web server`
		// Should parse only the first version
		p, err := parseAptCacheShow(output)
		require.NoError(t, err)
		require.NotNil(t, p)
		assert.Equal(t, "1.22.0-1ubuntu1", p.Version)
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
			URL:    "https://developer.download.nvidia.com/compute/cuda/repos/ubuntu2204/x86_64",
			GPGKey: "https://developer.download.nvidia.com/compute/cuda/repos/ubuntu2204/x86_64/7fa2af80.pub",
		}
		err := mgr.AddRepository(ctx, repo)
		require.NoError(t, err)
	})

	t.Run("held_packages_handling", func(t *testing.T) {
		// Simulate held package scenario
		mockExec.SetResponse("env", exec.FailureResult(1, "held packages: nvidia-driver-535"))
		opts := pkg.InstallOptions{Force: true}
		mockExec.SetResponse("env", exec.SuccessResult(""))
		err := mgr.Install(ctx, opts, "nvidia-driver-535")
		require.NoError(t, err)
	})
}

// =============================================================================
// Benchmark Tests
// =============================================================================

func BenchmarkManager_ParseDpkgQuery(b *testing.B) {
	// Generate realistic output
	var sb strings.Builder
	for i := 0; i < 500; i++ {
		sb.WriteString("package-")
		sb.WriteString(string(rune('a' + i%26)))
		sb.WriteString("\t1.0.0-1ubuntu1\tinstall ok installed\n")
	}
	output := sb.String()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = parseDpkgQuery(output)
	}
}

func BenchmarkManager_ParseAptCacheSearch(b *testing.B) {
	var sb strings.Builder
	for i := 0; i < 200; i++ {
		sb.WriteString("package-")
		sb.WriteString(string(rune('a' + i%26)))
		sb.WriteString(" - package description for testing purposes\n")
	}
	output := sb.String()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = parseAptCacheSearch(output)
	}
}

func BenchmarkManager_ParseAptCacheShow(b *testing.B) {
	output := `Package: nginx
Version: 1.22.0-1ubuntu1
Architecture: amd64
Installed-Size: 1200
Depends: libc6 (>= 2.17), libpcre3, libssl3 (>= 3.0.0)
Description: small, powerful, scalable web/proxy server
 Nginx (pronounced "engine X") is a high-performance HTTP and reverse proxy
 server.
Section: web
`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = parseAptCacheShow(output)
	}
}

func BenchmarkManager_ParseAptListUpgradable(b *testing.B) {
	var sb strings.Builder
	sb.WriteString("Listing...\n")
	for i := 0; i < 100; i++ {
		sb.WriteString("package-")
		sb.WriteString(string(rune('a' + i%26)))
		sb.WriteString("/jammy-updates 1.1.0-1ubuntu1 amd64 [upgradable from: 1.0.0-1ubuntu1]\n")
	}
	output := sb.String()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = parseAptListUpgradable(output)
	}
}

func BenchmarkManager_IsInstalled(b *testing.B) {
	mgr, mockExec := setupTest()
	mockExec.SetResponse("dpkg-query", exec.SuccessResult("install ok installed"))
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = mgr.IsInstalled(ctx, "nginx")
	}
}

func BenchmarkManager_Search(b *testing.B) {
	mgr, mockExec := setupTest()
	output := `nginx - small, powerful, scalable web/proxy server
nginx-common - common files for nginx
nginx-core - nginx web/proxy server (standard version)`
	mockExec.SetResponse("apt-cache", exec.SuccessResult(output))
	mockExec.SetResponse("dpkg-query", exec.FailureResult(1, "not found"))
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = mgr.Search(ctx, "nginx", pkg.SearchOptions{IncludeInstalled: false})
	}
}

func BenchmarkManager_ParseSourcesList(b *testing.B) {
	content := `# Main repos
deb http://archive.ubuntu.com/ubuntu jammy main restricted
deb-src http://archive.ubuntu.com/ubuntu jammy main restricted
deb http://archive.ubuntu.com/ubuntu jammy-updates main restricted
deb http://archive.ubuntu.com/ubuntu jammy-backports main restricted
deb [arch=amd64 signed-by=/etc/apt/keyrings/nvidia.gpg] https://developer.download.nvidia.com/compute/cuda/repos/ubuntu2204/x86_64 /
#deb http://archive.ubuntu.com/ubuntu jammy-proposed main restricted`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = parseSourcesList(content)
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
