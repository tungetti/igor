package factory

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tungetti/igor/internal/constants"
	"github.com/tungetti/igor/internal/distro"
	"github.com/tungetti/igor/internal/exec"
	"github.com/tungetti/igor/internal/pkg"
	"github.com/tungetti/igor/internal/pkg/apt"
	"github.com/tungetti/igor/internal/pkg/dnf"
	"github.com/tungetti/igor/internal/pkg/pacman"
	"github.com/tungetti/igor/internal/pkg/yum"
	"github.com/tungetti/igor/internal/pkg/zypper"
	"github.com/tungetti/igor/internal/privilege"
)

// =============================================================================
// Integration Test Scenarios
// =============================================================================

// TestFactory_CompleteDistributionWorkflow tests full detection and creation workflow.
func TestFactory_CompleteDistributionWorkflow(t *testing.T) {
	mockExec := exec.NewMockExecutor()
	priv := privilege.NewManager()

	tests := []struct {
		name           string
		osRelease      string
		expectedName   string
		expectedFamily constants.DistroFamily
	}{
		{
			name:           "Ubuntu 24.04 workflow",
			osRelease:      ubuntu2404OSRelease,
			expectedName:   "apt",
			expectedFamily: constants.FamilyDebian,
		},
		{
			name:           "Fedora 40 workflow",
			osRelease:      fedora40OSRelease,
			expectedName:   "dnf",
			expectedFamily: constants.FamilyRHEL,
		},
		{
			name:           "CentOS 7 workflow (YUM)",
			osRelease:      centos7OSRelease,
			expectedName:   "yum",
			expectedFamily: constants.FamilyRHEL,
		},
		{
			name:           "CentOS 8 workflow (DNF)",
			osRelease:      centos8OSRelease,
			expectedName:   "dnf",
			expectedFamily: constants.FamilyRHEL,
		},
		{
			name:           "Arch Linux workflow",
			osRelease:      archOSRelease,
			expectedName:   "pacman",
			expectedFamily: constants.FamilyArch,
		},
		{
			name:           "openSUSE Tumbleweed workflow",
			osRelease:      opensuseTumbleweedOSRelease,
			expectedName:   "zypper",
			expectedFamily: constants.FamilySUSE,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockFS := NewMockFileReader(map[string][]byte{
				"/etc/os-release": []byte(tt.osRelease),
			})
			detector := distro.NewDetector(mockExec, mockFS)
			factory := NewFactory(mockExec, priv, detector)

			// Step 1: Auto-detect distribution
			mgr, err := factory.Create(context.Background())
			require.NoError(t, err)
			require.NotNil(t, mgr)

			// Step 2: Verify manager properties
			assert.Equal(t, tt.expectedName, mgr.Name())
			assert.Equal(t, tt.expectedFamily, mgr.Family())

			// Step 3: Verify manager is usable (basic operations don't panic)
			_ = mgr.IsAvailable()
		})
	}
}

// TestFactory_FallbackBehavior tests fallback scenarios.
func TestFactory_FallbackBehavior(t *testing.T) {
	mockExec := exec.NewMockExecutor()
	priv := privilege.NewManager()

	t.Run("rhel_family_without_version_defaults_to_dnf", func(t *testing.T) {
		factory := NewFactory(mockExec, priv, nil)
		dist := &distro.Distribution{
			ID:        "rhel",
			VersionID: "", // No version
			Family:    constants.FamilyRHEL,
		}
		mgr, err := factory.CreateForDistribution(dist)
		require.NoError(t, err)
		assert.Equal(t, "dnf", mgr.Name())
	})

	t.Run("rhel_family_with_unparseable_version_defaults_to_dnf", func(t *testing.T) {
		factory := NewFactory(mockExec, priv, nil)
		dist := &distro.Distribution{
			ID:        "centos",
			VersionID: "stream", // Not a number
			Family:    constants.FamilyRHEL,
		}
		mgr, err := factory.CreateForDistribution(dist)
		require.NoError(t, err)
		assert.Equal(t, "dnf", mgr.Name())
	})

	t.Run("amazon_linux_2_uses_yum", func(t *testing.T) {
		factory := NewFactory(mockExec, priv, nil)
		dist := &distro.Distribution{
			ID:        "amzn",
			VersionID: "2",
			Family:    constants.FamilyRHEL,
		}
		mgr, err := factory.CreateForDistribution(dist)
		require.NoError(t, err)
		assert.Equal(t, "yum", mgr.Name())
	})

	t.Run("amazon_linux_2023_uses_dnf", func(t *testing.T) {
		factory := NewFactory(mockExec, priv, nil)
		dist := &distro.Distribution{
			ID:        "amzn",
			VersionID: "2023",
			Family:    constants.FamilyRHEL,
		}
		mgr, err := factory.CreateForDistribution(dist)
		require.NoError(t, err)
		assert.Equal(t, "dnf", mgr.Name())
	})
}

// TestFactory_ConcurrentCreation tests thread safety of factory.
func TestFactory_ConcurrentCreation(t *testing.T) {
	mockExec := exec.NewMockExecutor()
	priv := privilege.NewManager()
	factory := NewFactory(mockExec, priv, nil)

	var wg sync.WaitGroup
	errChan := make(chan error, 50)
	mgrChan := make(chan pkg.Manager, 50)

	families := []constants.DistroFamily{
		constants.FamilyDebian,
		constants.FamilyRHEL,
		constants.FamilyArch,
		constants.FamilySUSE,
	}

	// Create many managers concurrently
	for i := 0; i < 10; i++ {
		for _, family := range families {
			wg.Add(1)
			go func(f constants.DistroFamily) {
				defer wg.Done()
				mgr, err := factory.CreateForFamily(f)
				if err != nil {
					errChan <- err
					return
				}
				mgrChan <- mgr
			}(family)
		}
	}

	wg.Wait()
	close(errChan)
	close(mgrChan)

	// Check for errors
	for err := range errChan {
		t.Errorf("Concurrent creation failed: %v", err)
	}

	// Count created managers
	count := 0
	for range mgrChan {
		count++
	}
	assert.Equal(t, 40, count, "should create 40 managers (10 iterations x 4 families)")
}

// TestFactory_DistributionAutoDetection tests auto-detection scenarios.
func TestFactory_DistributionAutoDetection(t *testing.T) {
	mockExec := exec.NewMockExecutor()
	priv := privilege.NewManager()

	t.Run("detection_with_id_like", func(t *testing.T) {
		// Linux Mint (id_like: ubuntu debian)
		mockFS := NewMockFileReader(map[string][]byte{
			"/etc/os-release": []byte(linuxMintOSRelease),
		})
		detector := distro.NewDetector(mockExec, mockFS)
		factory := NewFactory(mockExec, priv, detector)

		mgr, err := factory.Create(context.Background())
		require.NoError(t, err)
		assert.Equal(t, "apt", mgr.Name())
		assert.Equal(t, constants.FamilyDebian, mgr.Family())
	})

	t.Run("detection_with_arch_derivative", func(t *testing.T) {
		// Manjaro (id_like: arch)
		mockFS := NewMockFileReader(map[string][]byte{
			"/etc/os-release": []byte(manjaroOSRelease),
		})
		detector := distro.NewDetector(mockExec, mockFS)
		factory := NewFactory(mockExec, priv, detector)

		mgr, err := factory.Create(context.Background())
		require.NoError(t, err)
		assert.Equal(t, "pacman", mgr.Name())
	})

	t.Run("detection_with_rhel_derivative", func(t *testing.T) {
		// Rocky Linux (id_like: rhel centos fedora)
		mockFS := NewMockFileReader(map[string][]byte{
			"/etc/os-release": []byte(rocky9OSRelease),
		})
		detector := distro.NewDetector(mockExec, mockFS)
		factory := NewFactory(mockExec, priv, detector)

		mgr, err := factory.Create(context.Background())
		require.NoError(t, err)
		assert.Equal(t, "dnf", mgr.Name())
	})
}

// TestFactory_ErrorHandling tests error scenarios.
func TestFactory_ErrorHandling(t *testing.T) {
	mockExec := exec.NewMockExecutor()
	priv := privilege.NewManager()

	t.Run("nil_detector_returns_error", func(t *testing.T) {
		factory := NewFactory(mockExec, priv, nil)
		mgr, err := factory.Create(context.Background())
		assert.Error(t, err)
		assert.Nil(t, mgr)
		assert.Contains(t, err.Error(), "detector is nil")
	})

	t.Run("nil_distribution_returns_error", func(t *testing.T) {
		factory := NewFactory(mockExec, priv, nil)
		mgr, err := factory.CreateForDistribution(nil)
		assert.Error(t, err)
		assert.Nil(t, mgr)
		assert.Contains(t, err.Error(), "distribution is nil")
	})

	t.Run("unknown_family_returns_error", func(t *testing.T) {
		factory := NewFactory(mockExec, priv, nil)
		mgr, err := factory.CreateForFamily(constants.FamilyUnknown)
		assert.Error(t, err)
		assert.Nil(t, mgr)
		assert.ErrorIs(t, err, ErrUnsupportedDistro)
	})

	t.Run("invalid_family_returns_error", func(t *testing.T) {
		factory := NewFactory(mockExec, priv, nil)
		invalidFamily := constants.DistroFamily("invalid")
		mgr, err := factory.CreateForFamily(invalidFamily)
		assert.Error(t, err)
		assert.Nil(t, mgr)
		assert.ErrorIs(t, err, ErrUnsupportedDistro)
	})

	t.Run("unknown_distribution_returns_error", func(t *testing.T) {
		mockFS := NewMockFileReader(map[string][]byte{
			"/etc/os-release": []byte(unknownOSRelease),
		})
		detector := distro.NewDetector(mockExec, mockFS)
		factory := NewFactory(mockExec, priv, detector)

		mgr, err := factory.Create(context.Background())
		assert.Error(t, err)
		assert.Nil(t, mgr)
		assert.ErrorIs(t, err, ErrUnsupportedDistro)
	})

	t.Run("detection_failure_returns_error", func(t *testing.T) {
		mockExec.SetDefaultResponse(exec.FailureResult(1, "command not found"))
		mockFS := NewMockFileReader(nil) // No files
		detector := distro.NewDetector(mockExec, mockFS)
		factory := NewFactory(mockExec, priv, detector)

		mgr, err := factory.Create(context.Background())
		assert.Error(t, err)
		assert.Nil(t, mgr)
		assert.Contains(t, err.Error(), "failed to detect distribution")
	})
}

// TestFactory_ContextHandling tests context cancellation and timeout.
func TestFactory_ContextHandling(t *testing.T) {
	mockExec := exec.NewMockExecutor()
	priv := privilege.NewManager()

	t.Run("context_cancellation", func(t *testing.T) {
		mockFS := NewMockFileReader(nil)
		detector := distro.NewDetector(mockExec, mockFS)
		factory := NewFactory(mockExec, priv, detector)

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		mgr, err := factory.Create(ctx)
		assert.Error(t, err)
		assert.Nil(t, mgr)
	})

	t.Run("context_timeout", func(t *testing.T) {
		mockFS := NewMockFileReader(nil)
		detector := distro.NewDetector(mockExec, mockFS)
		factory := NewFactory(mockExec, priv, detector)

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
		defer cancel()
		time.Sleep(10 * time.Millisecond) // Ensure timeout

		mgr, err := factory.Create(ctx)
		assert.Error(t, err)
		assert.Nil(t, mgr)
	})
}

// TestFactory_ManagerTypeVerification tests correct manager type creation.
func TestFactory_ManagerTypeVerification(t *testing.T) {
	mockExec := exec.NewMockExecutor()
	priv := privilege.NewManager()
	factory := NewFactory(mockExec, priv, nil)

	t.Run("apt_manager_type", func(t *testing.T) {
		mgr, err := factory.CreateForFamily(constants.FamilyDebian)
		require.NoError(t, err)
		_, ok := mgr.(*apt.Manager)
		assert.True(t, ok, "expected *apt.Manager")
	})

	t.Run("dnf_manager_type", func(t *testing.T) {
		mgr, err := factory.CreateForFamily(constants.FamilyRHEL)
		require.NoError(t, err)
		_, ok := mgr.(*dnf.Manager)
		assert.True(t, ok, "expected *dnf.Manager")
	})

	t.Run("yum_manager_type_for_centos7", func(t *testing.T) {
		dist := &distro.Distribution{
			ID:        "centos",
			VersionID: "7",
			Family:    constants.FamilyRHEL,
		}
		mgr, err := factory.CreateForDistribution(dist)
		require.NoError(t, err)
		_, ok := mgr.(*yum.Manager)
		assert.True(t, ok, "expected *yum.Manager for CentOS 7")
	})

	t.Run("pacman_manager_type", func(t *testing.T) {
		mgr, err := factory.CreateForFamily(constants.FamilyArch)
		require.NoError(t, err)
		_, ok := mgr.(*pacman.Manager)
		assert.True(t, ok, "expected *pacman.Manager")
	})

	t.Run("zypper_manager_type", func(t *testing.T) {
		mgr, err := factory.CreateForFamily(constants.FamilySUSE)
		require.NoError(t, err)
		_, ok := mgr.(*zypper.Manager)
		assert.True(t, ok, "expected *zypper.Manager")
	})
}

// TestFactory_AllDistributionFamilies tests all distribution families.
func TestFactory_AllDistributionFamilies(t *testing.T) {
	mockExec := exec.NewMockExecutor()
	priv := privilege.NewManager()
	factory := NewFactory(mockExec, priv, nil)

	supportedFamilies := SupportedFamilies()
	assert.Len(t, supportedFamilies, 4)

	for _, family := range supportedFamilies {
		t.Run(string(family), func(t *testing.T) {
			mgr, err := factory.CreateForFamily(family)
			require.NoError(t, err)
			require.NotNil(t, mgr)
			assert.Equal(t, family, mgr.Family())
		})
	}
}

// TestFactory_AllAvailableManagers tests that all managers are available.
func TestFactory_AllAvailableManagers(t *testing.T) {
	managers := AvailableManagers()
	assert.Len(t, managers, 5)
	assert.Contains(t, managers, "apt")
	assert.Contains(t, managers, "dnf")
	assert.Contains(t, managers, "yum")
	assert.Contains(t, managers, "pacman")
	assert.Contains(t, managers, "zypper")
}

// TestFactory_RHELVersionMapping tests RHEL version to package manager mapping.
func TestFactory_RHELVersionMapping(t *testing.T) {
	mockExec := exec.NewMockExecutor()
	priv := privilege.NewManager()
	factory := NewFactory(mockExec, priv, nil)

	versions := []struct {
		version  string
		expected string
		useYUM   bool
		distroID string
	}{
		{"6", "yum", true, "centos"},
		{"6.10", "yum", true, "centos"},
		{"7", "yum", true, "centos"},
		{"7.9", "yum", true, "centos"},
		{"7.9.2009", "yum", true, "centos"},
		{"8", "dnf", false, "centos"},
		{"8.5", "dnf", false, "centos"},
		{"9", "dnf", false, "centos"},
		{"9.0", "dnf", false, "centos"},
		{"40", "dnf", false, "fedora"}, // Fedora always DNF
		{"1", "dnf", false, "fedora"},  // Even old Fedora returns DNF
	}

	for _, v := range versions {
		t.Run(v.distroID+"_"+v.version, func(t *testing.T) {
			dist := &distro.Distribution{
				ID:        v.distroID,
				VersionID: v.version,
				Family:    constants.FamilyRHEL,
			}
			mgr, err := factory.CreateForDistribution(dist)
			require.NoError(t, err)
			assert.Equal(t, v.expected, mgr.Name())

			if v.useYUM {
				_, ok := mgr.(*yum.Manager)
				assert.True(t, ok, "expected *yum.Manager for version %s", v.version)
			} else {
				_, ok := mgr.(*dnf.Manager)
				assert.True(t, ok, "expected *dnf.Manager for version %s", v.version)
			}
		})
	}
}

// TestFactory_LazyCreation tests that managers are created on demand.
func TestFactory_LazyCreation(t *testing.T) {
	mockExec := exec.NewMockExecutor()
	priv := privilege.NewManager()
	factory := NewFactory(mockExec, priv, nil)

	// Each call should create a new instance
	mgr1, err := factory.CreateForFamily(constants.FamilyDebian)
	require.NoError(t, err)

	mgr2, err := factory.CreateForFamily(constants.FamilyDebian)
	require.NoError(t, err)

	// They should be different instances (not cached)
	assert.NotSame(t, mgr1, mgr2, "managers should be new instances on each call")

	// But they should have the same properties
	assert.Equal(t, mgr1.Name(), mgr2.Name())
	assert.Equal(t, mgr1.Family(), mgr2.Family())
}

// =============================================================================
// Benchmark Tests
// =============================================================================

func BenchmarkFactory_CreateForFamily(b *testing.B) {
	mockExec := exec.NewMockExecutor()
	priv := privilege.NewManager()
	factory := NewFactory(mockExec, priv, nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = factory.CreateForFamily(constants.FamilyDebian)
	}
}

func BenchmarkFactory_CreateForDistribution(b *testing.B) {
	mockExec := exec.NewMockExecutor()
	priv := privilege.NewManager()
	factory := NewFactory(mockExec, priv, nil)
	dist := &distro.Distribution{
		ID:        "ubuntu",
		VersionID: "24.04",
		Family:    constants.FamilyDebian,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = factory.CreateForDistribution(dist)
	}
}

func BenchmarkFactory_CreateWithDetection(b *testing.B) {
	mockExec := exec.NewMockExecutor()
	priv := privilege.NewManager()
	mockFS := NewMockFileReader(map[string][]byte{
		"/etc/os-release": []byte(ubuntu2404OSRelease),
	})
	detector := distro.NewDetector(mockExec, mockFS)
	factory := NewFactory(mockExec, priv, detector)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = factory.Create(ctx)
	}
}

func BenchmarkFactory_CreateAllFamilies(b *testing.B) {
	mockExec := exec.NewMockExecutor()
	priv := privilege.NewManager()
	factory := NewFactory(mockExec, priv, nil)
	families := SupportedFamilies()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, family := range families {
			_, _ = factory.CreateForFamily(family)
		}
	}
}

func BenchmarkFactory_ConcurrentCreation(b *testing.B) {
	mockExec := exec.NewMockExecutor()
	priv := privilege.NewManager()
	factory := NewFactory(mockExec, priv, nil)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		families := []constants.DistroFamily{
			constants.FamilyDebian,
			constants.FamilyRHEL,
			constants.FamilyArch,
			constants.FamilySUSE,
		}
		i := 0
		for pb.Next() {
			_, _ = factory.CreateForFamily(families[i%len(families)])
			i++
		}
	})
}

func BenchmarkAvailableManagers(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = AvailableManagers()
	}
}

func BenchmarkSupportedFamilies(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = SupportedFamilies()
	}
}

// =============================================================================
// Edge Case Tests
// =============================================================================

func TestFactory_IntegrationEdgeCases(t *testing.T) {
	mockExec := exec.NewMockExecutor()
	priv := privilege.NewManager()
	factory := NewFactory(mockExec, priv, nil)

	t.Run("oracle_linux_7_uses_yum", func(t *testing.T) {
		dist := &distro.Distribution{
			ID:        "ol",
			VersionID: "7.9",
			Family:    constants.FamilyRHEL,
		}
		mgr, err := factory.CreateForDistribution(dist)
		require.NoError(t, err)
		assert.Equal(t, "yum", mgr.Name())
	})

	t.Run("oracle_linux_8_uses_dnf", func(t *testing.T) {
		dist := &distro.Distribution{
			ID:        "ol",
			VersionID: "8.0",
			Family:    constants.FamilyRHEL,
		}
		mgr, err := factory.CreateForDistribution(dist)
		require.NoError(t, err)
		assert.Equal(t, "dnf", mgr.Name())
	})

	t.Run("version_with_leading_v_uses_dnf", func(t *testing.T) {
		dist := &distro.Distribution{
			ID:        "centos",
			VersionID: "v8",
			Family:    constants.FamilyRHEL,
		}
		mgr, err := factory.CreateForDistribution(dist)
		require.NoError(t, err)
		assert.Equal(t, "dnf", mgr.Name()) // Unparseable defaults to DNF
	})

	t.Run("version_with_only_text_uses_dnf", func(t *testing.T) {
		dist := &distro.Distribution{
			ID:        "centos",
			VersionID: "rolling",
			Family:    constants.FamilyRHEL,
		}
		mgr, err := factory.CreateForDistribution(dist)
		require.NoError(t, err)
		assert.Equal(t, "dnf", mgr.Name())
	})

	t.Run("scientific_linux_7_uses_yum", func(t *testing.T) {
		dist := &distro.Distribution{
			ID:        "scientific",
			VersionID: "7.9",
			Family:    constants.FamilyRHEL,
		}
		mgr, err := factory.CreateForDistribution(dist)
		require.NoError(t, err)
		assert.Equal(t, "yum", mgr.Name())
	})

	t.Run("endeavouros_uses_pacman", func(t *testing.T) {
		dist := &distro.Distribution{
			ID:        "endeavouros",
			VersionID: "2024.01.25",
			Family:    constants.FamilyArch,
		}
		mgr, err := factory.CreateForDistribution(dist)
		require.NoError(t, err)
		assert.Equal(t, "pacman", mgr.Name())
	})

	t.Run("garuda_uses_pacman", func(t *testing.T) {
		dist := &distro.Distribution{
			ID:        "garuda",
			VersionID: "rolling",
			Family:    constants.FamilyArch,
		}
		mgr, err := factory.CreateForDistribution(dist)
		require.NoError(t, err)
		assert.Equal(t, "pacman", mgr.Name())
	})

	t.Run("sles_uses_zypper", func(t *testing.T) {
		dist := &distro.Distribution{
			ID:        "sles",
			VersionID: "15.5",
			Family:    constants.FamilySUSE,
		}
		mgr, err := factory.CreateForDistribution(dist)
		require.NoError(t, err)
		assert.Equal(t, "zypper", mgr.Name())
	})
}

// TestFactory_ManagerMethodsWork verifies created managers have working methods.
func TestFactory_ManagerMethodsWork(t *testing.T) {
	mockExec := exec.NewMockExecutor()
	priv := privilege.NewManager()
	factory := NewFactory(mockExec, priv, nil)

	distributions := []*distro.Distribution{
		{ID: "ubuntu", VersionID: "24.04", Family: constants.FamilyDebian},
		{ID: "fedora", VersionID: "40", Family: constants.FamilyRHEL},
		{ID: "centos", VersionID: "7", Family: constants.FamilyRHEL},
		{ID: "arch", Family: constants.FamilyArch},
		{ID: "opensuse-leap", VersionID: "15.5", Family: constants.FamilySUSE},
	}

	for _, dist := range distributions {
		t.Run(dist.ID, func(t *testing.T) {
			mgr, err := factory.CreateForDistribution(dist)
			require.NoError(t, err)

			// These should not panic
			_ = mgr.Name()
			_ = mgr.Family()
			_ = mgr.IsAvailable()
		})
	}
}

// =============================================================================
// Helper for building strings
// =============================================================================

func buildOSRelease(id, versionID, name string, idLike ...string) string {
	var sb strings.Builder
	sb.WriteString("NAME=\"" + name + "\"\n")
	sb.WriteString("ID=" + id + "\n")
	if versionID != "" {
		sb.WriteString("VERSION_ID=\"" + versionID + "\"\n")
	}
	if len(idLike) > 0 {
		sb.WriteString("ID_LIKE=\"" + strings.Join(idLike, " ") + "\"\n")
	}
	return sb.String()
}
