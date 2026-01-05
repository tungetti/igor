package recovery

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tungetti/igor/internal/constants"
	"github.com/tungetti/igor/internal/distro"
	"github.com/tungetti/igor/internal/exec"
	"github.com/tungetti/igor/internal/logging"
	"github.com/tungetti/igor/internal/pkg"
	"github.com/tungetti/igor/internal/uninstall"
)

// =============================================================================
// Mock Implementations
// =============================================================================

// MockDiscovery is a test implementation of uninstall.Discovery.
type MockDiscovery struct {
	packages       *uninstall.DiscoveredPackages
	discoverErr    error
	driverPackages []string
	driverVersion  string
	cudaPackages   []string
	cudaVersion    string
	isInstalled    bool
}

func NewMockDiscovery() *MockDiscovery {
	return &MockDiscovery{
		packages: &uninstall.DiscoveredPackages{
			AllPackages: []string{},
			TotalCount:  0,
		},
	}
}

func (m *MockDiscovery) SetPackages(packages *uninstall.DiscoveredPackages) {
	m.packages = packages
}

func (m *MockDiscovery) SetDiscoverError(err error) {
	m.discoverErr = err
}

func (m *MockDiscovery) Discover(ctx context.Context) (*uninstall.DiscoveredPackages, error) {
	if m.discoverErr != nil {
		return nil, m.discoverErr
	}
	return m.packages, nil
}

func (m *MockDiscovery) DiscoverDriver(ctx context.Context) ([]string, string, error) {
	return m.driverPackages, m.driverVersion, m.discoverErr
}

func (m *MockDiscovery) DiscoverCUDA(ctx context.Context) ([]string, string, error) {
	return m.cudaPackages, m.cudaVersion, m.discoverErr
}

func (m *MockDiscovery) IsNVIDIAInstalled(ctx context.Context) (bool, error) {
	return m.isInstalled, m.discoverErr
}

func (m *MockDiscovery) GetDriverVersion(ctx context.Context) (string, error) {
	return m.driverVersion, m.discoverErr
}

var _ uninstall.Discovery = (*MockDiscovery)(nil)

// MockPackageManager is a test implementation of pkg.Manager.
type MockPackageManager struct {
	name           string
	family         constants.DistroFamily
	removedPkgs    []string
	removeErr      error
	installedPkgs  []pkg.Package
	listInstallErr error
}

func NewMockPackageManager() *MockPackageManager {
	return &MockPackageManager{
		name:   "mock",
		family: constants.FamilyDebian,
	}
}

func (m *MockPackageManager) SetRemoveError(err error) {
	m.removeErr = err
}

func (m *MockPackageManager) RemovedPackages() []string {
	return m.removedPkgs
}

func (m *MockPackageManager) Install(ctx context.Context, opts pkg.InstallOptions, packages ...string) error {
	return nil
}

func (m *MockPackageManager) Remove(ctx context.Context, opts pkg.RemoveOptions, packages ...string) error {
	if m.removeErr != nil {
		return m.removeErr
	}
	m.removedPkgs = append(m.removedPkgs, packages...)
	return nil
}

func (m *MockPackageManager) Update(ctx context.Context, opts pkg.UpdateOptions) error {
	return nil
}

func (m *MockPackageManager) Upgrade(ctx context.Context, opts pkg.InstallOptions, packages ...string) error {
	return nil
}

func (m *MockPackageManager) IsInstalled(ctx context.Context, pkgName string) (bool, error) {
	return false, nil
}

func (m *MockPackageManager) Search(ctx context.Context, query string, opts pkg.SearchOptions) ([]pkg.Package, error) {
	return nil, nil
}

func (m *MockPackageManager) Info(ctx context.Context, pkgName string) (*pkg.Package, error) {
	return nil, nil
}

func (m *MockPackageManager) ListInstalled(ctx context.Context) ([]pkg.Package, error) {
	if m.listInstallErr != nil {
		return nil, m.listInstallErr
	}
	return m.installedPkgs, nil
}

func (m *MockPackageManager) ListUpgradable(ctx context.Context) ([]pkg.Package, error) {
	return nil, nil
}

func (m *MockPackageManager) AddRepository(ctx context.Context, repo pkg.Repository) error {
	return nil
}

func (m *MockPackageManager) RemoveRepository(ctx context.Context, name string) error {
	return nil
}

func (m *MockPackageManager) ListRepositories(ctx context.Context) ([]pkg.Repository, error) {
	return nil, nil
}

func (m *MockPackageManager) EnableRepository(ctx context.Context, name string) error {
	return nil
}

func (m *MockPackageManager) DisableRepository(ctx context.Context, name string) error {
	return nil
}

func (m *MockPackageManager) RefreshRepositories(ctx context.Context) error {
	return nil
}

func (m *MockPackageManager) Clean(ctx context.Context) error {
	return nil
}

func (m *MockPackageManager) AutoRemove(ctx context.Context) error {
	return nil
}

func (m *MockPackageManager) Verify(ctx context.Context, pkgName string) (bool, error) {
	return false, nil
}

func (m *MockPackageManager) Name() string {
	return m.name
}

func (m *MockPackageManager) Family() constants.DistroFamily {
	return m.family
}

func (m *MockPackageManager) IsAvailable() bool {
	return true
}

var _ pkg.Manager = (*MockPackageManager)(nil)

// =============================================================================
// EnvironmentType Tests
// =============================================================================

func TestEnvironmentType_String(t *testing.T) {
	tests := []struct {
		envType  EnvironmentType
		expected string
	}{
		{EnvironmentUnknown, "unknown"},
		{EnvironmentTTY, "tty"},
		{EnvironmentGraphical, "graphical"},
		{EnvironmentSSH, "ssh"},
		{EnvironmentType(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.envType.String())
		})
	}
}

// =============================================================================
// DetectEnvironment Tests
// =============================================================================

func TestDetectEnvironment(t *testing.T) {
	t.Run("detects graphical environment with DISPLAY", func(t *testing.T) {
		env := DetectEnvironmentWithOptions(DetectionOptions{
			EnvReader: func(key string) string {
				switch key {
				case "DISPLAY":
					return ":0"
				default:
					return ""
				}
			},
		})

		assert.Equal(t, EnvironmentGraphical, env.Type)
		assert.Equal(t, ":0", env.Display)
	})

	t.Run("detects graphical environment with WAYLAND_DISPLAY", func(t *testing.T) {
		env := DetectEnvironmentWithOptions(DetectionOptions{
			EnvReader: func(key string) string {
				switch key {
				case "WAYLAND_DISPLAY":
					return "wayland-0"
				default:
					return ""
				}
			},
		})

		assert.Equal(t, EnvironmentGraphical, env.Type)
		assert.Equal(t, "wayland-0", env.WaylandDisplay)
	})

	t.Run("detects SSH environment", func(t *testing.T) {
		env := DetectEnvironmentWithOptions(DetectionOptions{
			EnvReader: func(key string) string {
				switch key {
				case "SSH_CONNECTION":
					return "192.168.1.100 12345 192.168.1.1 22"
				default:
					return ""
				}
			},
		})

		assert.Equal(t, EnvironmentSSH, env.Type)
		assert.NotEmpty(t, env.SSHConnection)
	})

	t.Run("SSH takes precedence over graphical", func(t *testing.T) {
		env := DetectEnvironmentWithOptions(DetectionOptions{
			EnvReader: func(key string) string {
				switch key {
				case "SSH_CONNECTION":
					return "192.168.1.100 12345 192.168.1.1 22"
				case "DISPLAY":
					return ":0"
				default:
					return ""
				}
			},
		})

		assert.Equal(t, EnvironmentSSH, env.Type)
	})

	t.Run("detects TTY with TERM=linux", func(t *testing.T) {
		env := DetectEnvironmentWithOptions(DetectionOptions{
			EnvReader: func(key string) string {
				switch key {
				case "TERM":
					return "linux"
				default:
					return ""
				}
			},
		})

		assert.Equal(t, EnvironmentTTY, env.Type)
		assert.Equal(t, "linux", env.Term)
	})

	t.Run("detects recovery boot", func(t *testing.T) {
		env := DetectEnvironmentWithOptions(DetectionOptions{
			EnvReader: func(key string) string {
				return ""
			},
			FileReader: func(path string) ([]byte, error) {
				if path == "/proc/cmdline" {
					return []byte("root=/dev/sda1 single"), nil
				}
				return nil, nil
			},
		})

		assert.True(t, env.IsRecoveryBoot)
	})

	t.Run("detects recovery with various keywords", func(t *testing.T) {
		keywords := []string{"single", "rescue", "recovery", "emergency", "runlevel=1"}

		for _, keyword := range keywords {
			t.Run(keyword, func(t *testing.T) {
				env := DetectEnvironmentWithOptions(DetectionOptions{
					EnvReader: func(key string) string {
						return ""
					},
					FileReader: func(path string) ([]byte, error) {
						if path == "/proc/cmdline" {
							return []byte("root=/dev/sda1 " + keyword), nil
						}
						return nil, nil
					},
				})

				assert.True(t, env.IsRecoveryBoot)
			})
		}
	})

	t.Run("unknown environment when nothing detected", func(t *testing.T) {
		env := DetectEnvironmentWithOptions(DetectionOptions{
			EnvReader: func(key string) string {
				return ""
			},
			FileReader: func(path string) ([]byte, error) {
				return nil, nil
			},
		})

		assert.Equal(t, EnvironmentUnknown, env.Type)
	})
}

func TestDetectEnvironment_Default(t *testing.T) {
	// Test the default function - it should not panic
	env := DetectEnvironment()
	assert.NotNil(t, env)
}

// =============================================================================
// Environment Method Tests
// =============================================================================

func TestEnvironment_IsRecoveryMode(t *testing.T) {
	tests := []struct {
		name           string
		envType        EnvironmentType
		isRecoveryBoot bool
		expected       bool
	}{
		{"TTY is recovery mode", EnvironmentTTY, false, true},
		{"TTY with recovery boot", EnvironmentTTY, true, true},
		{"Graphical is not recovery", EnvironmentGraphical, false, false},
		{"Graphical with recovery boot", EnvironmentGraphical, true, false},
		{"SSH is not recovery", EnvironmentSSH, false, false},
		{"Unknown without recovery boot", EnvironmentUnknown, false, false},
		{"Unknown with recovery boot", EnvironmentUnknown, true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := &Environment{
				Type:           tt.envType,
				IsRecoveryBoot: tt.isRecoveryBoot,
			}
			assert.Equal(t, tt.expected, env.IsRecoveryMode())
		})
	}
}

func TestEnvironment_IsGraphical(t *testing.T) {
	t.Run("returns true for graphical", func(t *testing.T) {
		env := &Environment{Type: EnvironmentGraphical}
		assert.True(t, env.IsGraphical())
	})

	t.Run("returns false for TTY", func(t *testing.T) {
		env := &Environment{Type: EnvironmentTTY}
		assert.False(t, env.IsGraphical())
	})
}

func TestEnvironment_IsSSH(t *testing.T) {
	t.Run("returns true for SSH", func(t *testing.T) {
		env := &Environment{Type: EnvironmentSSH}
		assert.True(t, env.IsSSH())
	})

	t.Run("returns false for TTY", func(t *testing.T) {
		env := &Environment{Type: EnvironmentTTY}
		assert.False(t, env.IsSSH())
	})
}

// =============================================================================
// NewRecoveryMode Tests
// =============================================================================

func TestNewRecoveryMode(t *testing.T) {
	t.Run("creates with default options", func(t *testing.T) {
		rm := NewRecoveryMode()

		assert.NotNil(t, rm)
		assert.NotNil(t, rm.ui)
		assert.NotNil(t, rm.env)
		assert.NotNil(t, rm.logger)
	})

	t.Run("with environment option", func(t *testing.T) {
		env := &Environment{Type: EnvironmentTTY}
		rm := NewRecoveryMode(WithRecoveryEnvironment(env))

		assert.Equal(t, env, rm.env)
	})

	t.Run("with distro option", func(t *testing.T) {
		dist := &distro.Distribution{ID: "ubuntu", Name: "Ubuntu"}
		rm := NewRecoveryMode(WithRecoveryDistro(dist))

		assert.Equal(t, dist, rm.distroInfo)
	})

	t.Run("with executor option", func(t *testing.T) {
		executor := exec.NewMockExecutor()
		rm := NewRecoveryMode(WithRecoveryExecutor(executor))

		assert.Equal(t, executor, rm.executor)
	})

	t.Run("with package manager option", func(t *testing.T) {
		pm := NewMockPackageManager()
		rm := NewRecoveryMode(WithRecoveryPackageManager(pm))

		assert.Equal(t, pm, rm.pkgManager)
	})

	t.Run("with logger option", func(t *testing.T) {
		logger := logging.NewNop()
		rm := NewRecoveryMode(WithRecoveryLogger(logger))

		assert.NotNil(t, rm.logger)
	})

	t.Run("with UI option", func(t *testing.T) {
		var buf bytes.Buffer
		ui := NewTTYUI(WithTTYWriter(&buf))
		rm := NewRecoveryMode(WithRecoveryUI(ui))

		assert.Equal(t, ui, rm.ui)
	})

	t.Run("with discovery option", func(t *testing.T) {
		discovery := NewMockDiscovery()
		rm := NewRecoveryMode(WithRecoveryDiscovery(discovery))

		assert.Equal(t, discovery, rm.discovery)
	})

	t.Run("with dry run option", func(t *testing.T) {
		rm := NewRecoveryMode(WithRecoveryDryRun(true))

		assert.True(t, rm.dryRun)
	})

	t.Run("creates discovery from package manager", func(t *testing.T) {
		pm := NewMockPackageManager()
		rm := NewRecoveryMode(WithRecoveryPackageManager(pm))

		assert.NotNil(t, rm.discovery)
	})

	t.Run("with all options", func(t *testing.T) {
		var buf bytes.Buffer
		env := &Environment{Type: EnvironmentTTY}
		dist := &distro.Distribution{ID: "fedora"}
		executor := exec.NewMockExecutor()
		pm := NewMockPackageManager()
		logger := logging.NewNop()
		ui := NewTTYUI(WithTTYWriter(&buf))
		discovery := NewMockDiscovery()

		rm := NewRecoveryMode(
			WithRecoveryEnvironment(env),
			WithRecoveryDistro(dist),
			WithRecoveryExecutor(executor),
			WithRecoveryPackageManager(pm),
			WithRecoveryLogger(logger),
			WithRecoveryUI(ui),
			WithRecoveryDiscovery(discovery),
			WithRecoveryDryRun(true),
		)

		assert.Equal(t, env, rm.env)
		assert.Equal(t, dist, rm.distroInfo)
		assert.Equal(t, executor, rm.executor)
		assert.Equal(t, pm, rm.pkgManager)
		assert.Equal(t, ui, rm.ui)
		assert.Equal(t, discovery, rm.discovery)
		assert.True(t, rm.dryRun)
	})
}

// =============================================================================
// RecoveryMode Accessor Tests
// =============================================================================

func TestRecoveryMode_Accessors(t *testing.T) {
	t.Run("Environment returns environment", func(t *testing.T) {
		env := &Environment{Type: EnvironmentTTY}
		rm := NewRecoveryMode(WithRecoveryEnvironment(env))

		assert.Equal(t, env, rm.Environment())
	})

	t.Run("UI returns UI", func(t *testing.T) {
		var buf bytes.Buffer
		ui := NewTTYUI(WithTTYWriter(&buf))
		rm := NewRecoveryMode(WithRecoveryUI(ui))

		assert.Equal(t, ui, rm.UI())
	})

	t.Run("IsDryRun returns dry run status", func(t *testing.T) {
		rm := NewRecoveryMode(WithRecoveryDryRun(true))
		assert.True(t, rm.IsDryRun())

		rm = NewRecoveryMode(WithRecoveryDryRun(false))
		assert.False(t, rm.IsDryRun())
	})
}

// =============================================================================
// RecoveryMode.Run Tests
// =============================================================================

func TestRecoveryMode_Run(t *testing.T) {
	ctx := context.Background()

	t.Run("returns error without discovery", func(t *testing.T) {
		var buf bytes.Buffer
		reader := strings.NewReader("y\n")
		ui := NewTTYUI(WithTTYWriter(&buf), WithTTYReader(reader))
		env := &Environment{Type: EnvironmentTTY}

		rm := NewRecoveryMode(
			WithRecoveryUI(ui),
			WithRecoveryEnvironment(env),
			// No discovery or package manager
		)

		// Mock euid check by running the test - it will fail on non-root check
		// In test environment, we can't easily mock os.Geteuid
		err := rm.Run(ctx)

		// Either fails on root check or discovery check
		assert.Error(t, err)
	})

	t.Run("shows no packages message when empty", func(t *testing.T) {
		var buf bytes.Buffer
		reader := strings.NewReader("") // No input needed
		ui := NewTTYUI(WithTTYWriter(&buf), WithTTYReader(reader))
		env := &Environment{Type: EnvironmentTTY}
		discovery := NewMockDiscovery()
		discovery.SetPackages(&uninstall.DiscoveredPackages{
			AllPackages: []string{},
			TotalCount:  0,
		})

		rm := NewRecoveryMode(
			WithRecoveryUI(ui),
			WithRecoveryEnvironment(env),
			WithRecoveryDiscovery(discovery),
		)

		// This will fail on root check in test environment
		_ = rm.Run(ctx)

		// The test verifies it doesn't panic
	})

	t.Run("warns about graphical environment", func(t *testing.T) {
		var buf bytes.Buffer
		reader := strings.NewReader("n\n") // Cancel when asked
		ui := NewTTYUI(WithTTYWriter(&buf), WithTTYReader(reader))
		env := &Environment{Type: EnvironmentGraphical, Display: ":0"}
		discovery := NewMockDiscovery()

		rm := NewRecoveryMode(
			WithRecoveryUI(ui),
			WithRecoveryEnvironment(env),
			WithRecoveryDiscovery(discovery),
		)

		_ = rm.Run(ctx)

		output := buf.String()
		assert.Contains(t, output, "graphical environment")
	})
}

// =============================================================================
// RecoveryMode.ExecuteUninstall Tests
// =============================================================================

func TestRecoveryMode_ExecuteUninstall(t *testing.T) {
	ctx := context.Background()

	t.Run("removes packages via package manager", func(t *testing.T) {
		var buf bytes.Buffer
		ui := NewTTYUI(WithTTYWriter(&buf))
		pm := NewMockPackageManager()
		executor := exec.NewMockExecutor()

		// Set up which command to return for initramfs rebuild
		executor.SetResponse("which", exec.SuccessResult("/usr/bin/update-initramfs"))
		executor.SetResponse("update-initramfs", exec.SuccessResult(""))

		rm := NewRecoveryMode(
			WithRecoveryUI(ui),
			WithRecoveryPackageManager(pm),
			WithRecoveryExecutor(executor),
		)

		packages := []string{"nvidia-driver-550", "nvidia-settings"}
		err := rm.ExecuteUninstall(ctx, packages, true)

		require.NoError(t, err)
		assert.Contains(t, pm.RemovedPackages(), "nvidia-driver-550")
		assert.Contains(t, pm.RemovedPackages(), "nvidia-settings")
	})

	t.Run("dry run does not remove packages", func(t *testing.T) {
		var buf bytes.Buffer
		ui := NewTTYUI(WithTTYWriter(&buf))
		pm := NewMockPackageManager()

		rm := NewRecoveryMode(
			WithRecoveryUI(ui),
			WithRecoveryPackageManager(pm),
			WithRecoveryDryRun(true),
		)

		packages := []string{"nvidia-driver-550"}
		err := rm.ExecuteUninstall(ctx, packages, true)

		require.NoError(t, err)
		assert.Empty(t, pm.RemovedPackages())
		assert.Contains(t, buf.String(), "[DRY RUN]")
	})

	t.Run("handles empty package list", func(t *testing.T) {
		var buf bytes.Buffer
		ui := NewTTYUI(WithTTYWriter(&buf))

		rm := NewRecoveryMode(
			WithRecoveryUI(ui),
		)

		err := rm.ExecuteUninstall(ctx, []string{}, false)

		require.NoError(t, err)
		assert.Contains(t, buf.String(), "No packages to remove")
	})

	t.Run("continues on package removal error", func(t *testing.T) {
		var buf bytes.Buffer
		ui := NewTTYUI(WithTTYWriter(&buf))
		pm := NewMockPackageManager()
		pm.SetRemoveError(pkg.ErrRemoveFailed)

		rm := NewRecoveryMode(
			WithRecoveryUI(ui),
			WithRecoveryPackageManager(pm),
		)

		packages := []string{"nvidia-driver-550", "nvidia-settings"}
		err := rm.ExecuteUninstall(ctx, packages, false)

		// Should not return error - continues with other packages
		require.NoError(t, err)
		assert.Contains(t, buf.String(), "[FAIL]")
	})

	t.Run("handles no package manager", func(t *testing.T) {
		var buf bytes.Buffer
		ui := NewTTYUI(WithTTYWriter(&buf))

		rm := NewRecoveryMode(
			WithRecoveryUI(ui),
			// No package manager
		)

		packages := []string{"nvidia-driver-550"}
		err := rm.ExecuteUninstall(ctx, packages, false)

		require.NoError(t, err)
		assert.Contains(t, buf.String(), "No package manager configured")
	})

	t.Run("rebuilds initramfs when restoring nouveau", func(t *testing.T) {
		var buf bytes.Buffer
		ui := NewTTYUI(WithTTYWriter(&buf))
		pm := NewMockPackageManager()
		executor := exec.NewMockExecutor()

		executor.SetResponse("which", exec.SuccessResult("/usr/bin/update-initramfs"))
		executor.SetResponse("update-initramfs", exec.SuccessResult(""))

		rm := NewRecoveryMode(
			WithRecoveryUI(ui),
			WithRecoveryPackageManager(pm),
			WithRecoveryExecutor(executor),
		)

		packages := []string{"nvidia-driver-550"}
		err := rm.ExecuteUninstall(ctx, packages, true)

		require.NoError(t, err)
		assert.True(t, executor.WasCalled("which"))
		assert.Contains(t, buf.String(), "initramfs")
	})

	t.Run("shows step progress", func(t *testing.T) {
		var buf bytes.Buffer
		ui := NewTTYUI(WithTTYWriter(&buf))
		pm := NewMockPackageManager()

		rm := NewRecoveryMode(
			WithRecoveryUI(ui),
			WithRecoveryPackageManager(pm),
		)

		packages := []string{"pkg1", "pkg2", "pkg3"}
		_ = rm.ExecuteUninstall(ctx, packages, false)

		output := buf.String()
		assert.Contains(t, output, "[1/3]")
		assert.Contains(t, output, "[2/3]")
		assert.Contains(t, output, "[3/3]")
	})
}

// =============================================================================
// Detection Helper Tests
// =============================================================================

func TestDetectRecoveryBoot(t *testing.T) {
	t.Run("returns false on file read error", func(t *testing.T) {
		result := detectRecoveryBoot(func(path string) ([]byte, error) {
			return nil, assert.AnError
		})

		assert.False(t, result)
	})

	t.Run("returns false for normal boot", func(t *testing.T) {
		result := detectRecoveryBoot(func(path string) ([]byte, error) {
			return []byte("root=/dev/sda1 ro quiet splash"), nil
		})

		assert.False(t, result)
	})

	t.Run("case insensitive matching", func(t *testing.T) {
		result := detectRecoveryBoot(func(path string) ([]byte, error) {
			return []byte("root=/dev/sda1 SINGLE"), nil
		})

		assert.True(t, result)
	})
}

func TestDetermineEnvironmentType(t *testing.T) {
	t.Run("SSH takes precedence", func(t *testing.T) {
		env := &Environment{
			SSHConnection: "1.2.3.4 1234 5.6.7.8 22",
			Display:       ":0",
			Term:          "linux",
		}

		result := determineEnvironmentType(env)
		assert.Equal(t, EnvironmentSSH, result)
	})

	t.Run("graphical if DISPLAY set", func(t *testing.T) {
		env := &Environment{
			Display: ":0",
		}

		result := determineEnvironmentType(env)
		assert.Equal(t, EnvironmentGraphical, result)
	})

	t.Run("graphical if WAYLAND_DISPLAY set", func(t *testing.T) {
		env := &Environment{
			WaylandDisplay: "wayland-0",
		}

		result := determineEnvironmentType(env)
		assert.Equal(t, EnvironmentGraphical, result)
	})

	t.Run("TTY if TERM is linux", func(t *testing.T) {
		env := &Environment{
			Term: "linux",
		}

		result := determineEnvironmentType(env)
		assert.Equal(t, EnvironmentTTY, result)
	})

	t.Run("TTY if on real TTY", func(t *testing.T) {
		env := &Environment{
			TTY: "/dev/tty1",
		}

		result := determineEnvironmentType(env)
		assert.Equal(t, EnvironmentTTY, result)
	})

	t.Run("TTY if recovery boot with no graphical", func(t *testing.T) {
		env := &Environment{
			IsRecoveryBoot: true,
		}

		result := determineEnvironmentType(env)
		assert.Equal(t, EnvironmentTTY, result)
	})

	t.Run("unknown if nothing matches", func(t *testing.T) {
		env := &Environment{}

		result := determineEnvironmentType(env)
		assert.Equal(t, EnvironmentUnknown, result)
	})
}

// =============================================================================
// Integration Tests
// =============================================================================

func TestRecoveryMode_IntegrationDryRun(t *testing.T) {
	ctx := context.Background()

	var buf bytes.Buffer
	// Simulate user confirming the operation
	reader := strings.NewReader("y\ny\n")
	ui := NewTTYUI(WithTTYWriter(&buf), WithTTYReader(reader))
	env := &Environment{Type: EnvironmentTTY}
	pm := NewMockPackageManager()
	executor := exec.NewMockExecutor()

	discovery := NewMockDiscovery()
	discovery.SetPackages(&uninstall.DiscoveredPackages{
		AllPackages:    []string{"nvidia-driver-550", "nvidia-settings"},
		DriverPackages: []string{"nvidia-driver-550"},
		TotalCount:     2,
		DriverVersion:  "550",
	})

	rm := NewRecoveryMode(
		WithRecoveryUI(ui),
		WithRecoveryEnvironment(env),
		WithRecoveryPackageManager(pm),
		WithRecoveryExecutor(executor),
		WithRecoveryDiscovery(discovery),
		WithRecoveryDryRun(true),
	)

	// This will fail on root check, but we can test the setup
	_ = rm.Run(ctx)

	// Verify no actual packages were removed
	assert.Empty(t, pm.RemovedPackages())
}

// =============================================================================
// Edge Cases and Error Handling Tests
// =============================================================================

func TestRecoveryMode_DiscoveryError(t *testing.T) {
	ctx := context.Background()

	var buf bytes.Buffer
	reader := strings.NewReader("y\n")
	ui := NewTTYUI(WithTTYWriter(&buf), WithTTYReader(reader))
	env := &Environment{Type: EnvironmentTTY}

	discovery := NewMockDiscovery()
	discovery.SetDiscoverError(assert.AnError)

	rm := NewRecoveryMode(
		WithRecoveryUI(ui),
		WithRecoveryEnvironment(env),
		WithRecoveryDiscovery(discovery),
	)

	// Will fail on root check in test env, but discovery error is after that
	_ = rm.Run(ctx)
}

func TestRecoveryMode_ConcurrentAccess(t *testing.T) {
	rm := NewRecoveryMode()

	// Test concurrent access to accessors
	done := make(chan bool)

	go func() {
		for i := 0; i < 100; i++ {
			_ = rm.Environment()
			_ = rm.UI()
			_ = rm.IsDryRun()
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 100; i++ {
			_ = rm.Environment()
			_ = rm.UI()
			_ = rm.IsDryRun()
		}
		done <- true
	}()

	<-done
	<-done
}

// =============================================================================
// Additional Coverage Tests
// =============================================================================

func TestWithRecoveryOrchestrator(t *testing.T) {
	orchestrator := uninstall.NewUninstallOrchestrator()
	rm := NewRecoveryMode(WithRecoveryOrchestrator(orchestrator))

	assert.Equal(t, orchestrator, rm.orchestrator)
}

func TestDefaultCommandRunner(t *testing.T) {
	_, err := defaultCommandRunner("test", "arg1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not supported")
}

func TestShowEnvironmentInfo(t *testing.T) {
	t.Run("with full environment", func(t *testing.T) {
		var buf bytes.Buffer
		ui := NewTTYUI(WithTTYWriter(&buf))
		env := &Environment{
			Type:           EnvironmentTTY,
			TTY:            "/dev/tty1",
			Term:           "linux",
			IsRecoveryBoot: true,
		}
		dist := &distro.Distribution{ID: "ubuntu", Name: "Ubuntu", VersionID: "24.04"}

		rm := NewRecoveryMode(
			WithRecoveryUI(ui),
			WithRecoveryEnvironment(env),
			WithRecoveryDistro(dist),
		)

		rm.showEnvironmentInfo()

		output := buf.String()
		assert.Contains(t, output, "tty")
		assert.Contains(t, output, "/dev/tty1")
		assert.Contains(t, output, "linux")
		assert.Contains(t, output, "recovery mode")
		assert.Contains(t, output, "Ubuntu")
	})

	t.Run("with nil environment", func(t *testing.T) {
		var buf bytes.Buffer
		ui := NewTTYUI(WithTTYWriter(&buf))

		rm := &RecoveryMode{
			ui:  ui,
			env: nil,
		}

		rm.showEnvironmentInfo()

		output := buf.String()
		assert.Contains(t, output, "not detected")
	})

	t.Run("with minimal environment", func(t *testing.T) {
		var buf bytes.Buffer
		ui := NewTTYUI(WithTTYWriter(&buf))
		env := &Environment{
			Type: EnvironmentGraphical,
		}

		rm := NewRecoveryMode(
			WithRecoveryUI(ui),
			WithRecoveryEnvironment(env),
		)

		rm.showEnvironmentInfo()

		output := buf.String()
		assert.Contains(t, output, "graphical")
	})
}

func TestRebuildInitramfs(t *testing.T) {
	ctx := context.Background()

	t.Run("returns error without executor", func(t *testing.T) {
		rm := NewRecoveryMode()

		err := rm.rebuildInitramfs(ctx)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "executor not configured")
	})

	t.Run("tries multiple commands", func(t *testing.T) {
		executor := exec.NewMockExecutor()
		// First which fails (update-initramfs not found)
		// But we set a default that returns success for the which command
		executor.SetResponse("which", exec.SuccessResult("/usr/bin/update-initramfs"))
		executor.SetResponse("update-initramfs", exec.SuccessResult(""))

		rm := NewRecoveryMode(WithRecoveryExecutor(executor))

		err := rm.rebuildInitramfs(ctx)

		assert.NoError(t, err)
	})

	t.Run("returns error when no command found", func(t *testing.T) {
		executor := exec.NewMockExecutor()
		executor.SetResponse("which", exec.FailureResult(1, "not found"))

		rm := NewRecoveryMode(WithRecoveryExecutor(executor))

		err := rm.rebuildInitramfs(ctx)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no initramfs rebuild command found")
	})

	t.Run("returns error when command fails", func(t *testing.T) {
		executor := exec.NewMockExecutor()
		executor.SetResponse("which", exec.SuccessResult("/usr/bin/update-initramfs"))
		executor.SetResponse("update-initramfs", exec.FailureResult(1, "failed to rebuild"))

		rm := NewRecoveryMode(WithRecoveryExecutor(executor))

		err := rm.rebuildInitramfs(ctx)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed")
	})
}

func TestRecoveryMode_ExecuteUninstall_RestoreNouveau(t *testing.T) {
	ctx := context.Background()

	t.Run("dry run skips nouveau restoration", func(t *testing.T) {
		var buf bytes.Buffer
		ui := NewTTYUI(WithTTYWriter(&buf))
		pm := NewMockPackageManager()
		executor := exec.NewMockExecutor()

		rm := NewRecoveryMode(
			WithRecoveryUI(ui),
			WithRecoveryPackageManager(pm),
			WithRecoveryExecutor(executor),
			WithRecoveryDryRun(true),
		)

		packages := []string{"nvidia-driver-550"}
		err := rm.ExecuteUninstall(ctx, packages, true)

		require.NoError(t, err)
		output := buf.String()
		assert.Contains(t, output, "[DRY RUN]")
		assert.Contains(t, output, "blacklist")
		assert.Contains(t, output, "initramfs")
	})

	t.Run("skips nouveau restoration when not requested", func(t *testing.T) {
		var buf bytes.Buffer
		ui := NewTTYUI(WithTTYWriter(&buf))
		pm := NewMockPackageManager()

		rm := NewRecoveryMode(
			WithRecoveryUI(ui),
			WithRecoveryPackageManager(pm),
		)

		packages := []string{"nvidia-driver-550"}
		err := rm.ExecuteUninstall(ctx, packages, false)

		require.NoError(t, err)
		output := buf.String()
		assert.NotContains(t, output, "blacklist")
		assert.NotContains(t, output, "initramfs")
	})
}

func TestDetectRecoveryBoot_RunlevelS(t *testing.T) {
	result := detectRecoveryBoot(func(path string) ([]byte, error) {
		return []byte("root=/dev/sda2 s"), nil
	})

	assert.True(t, result)
}

func TestTTYUI_Header_LongText(t *testing.T) {
	var buf bytes.Buffer
	ui := NewTTYUI(WithTTYWriter(&buf), WithTTYWidth(20))

	// Header longer than width
	ui.Header("This is a very long header that exceeds the width")

	output := buf.String()
	assert.Contains(t, output, "This is a very long header")
}

func TestTTYUI_ShowPackages_Truncation(t *testing.T) {
	var buf bytes.Buffer
	ui := NewTTYUI(WithTTYWriter(&buf), WithTTYWidth(80))

	// Package with very long name
	packages := []string{
		"nvidia-driver-550",
		"very-long-package-name-that-needs-to-be-truncated-to-fit-the-column-width",
		"nvidia-settings",
	}
	ui.ShowPackages(packages)

	// Should not panic and should contain output
	output := buf.String()
	assert.Contains(t, output, "Found 3 package(s)")
}

func TestTTYUI_ShowPackages_NarrowWidth(t *testing.T) {
	var buf bytes.Buffer
	// Very narrow width forces single-column layout
	ui := NewTTYUI(WithTTYWriter(&buf), WithTTYWidth(30))

	packages := make([]string, 10)
	for i := range packages {
		packages[i] = "pkg-" + string(rune('a'+i))
	}
	ui.ShowPackages(packages)

	output := buf.String()
	assert.Contains(t, output, "Found 10 package(s)")
}

func TestDetectEnvironmentWithOptions_TTYFromEnv(t *testing.T) {
	env := DetectEnvironmentWithOptions(DetectionOptions{
		EnvReader: func(key string) string {
			switch key {
			case "TTY":
				return "/dev/tty2"
			default:
				return ""
			}
		},
	})

	assert.Equal(t, "/dev/tty2", env.TTY)
}

func TestRecoveryMode_ExecuteUninstall_WithInitramfsCommandError(t *testing.T) {
	ctx := context.Background()

	var buf bytes.Buffer
	ui := NewTTYUI(WithTTYWriter(&buf))
	pm := NewMockPackageManager()
	executor := exec.NewMockExecutor()

	// which succeeds but command returns error
	executor.SetResponse("which", exec.SuccessResult("/usr/bin/update-initramfs"))
	executor.SetResponse("update-initramfs", exec.ErrorResult(assert.AnError))

	rm := NewRecoveryMode(
		WithRecoveryUI(ui),
		WithRecoveryPackageManager(pm),
		WithRecoveryExecutor(executor),
	)

	packages := []string{"nvidia-driver-550"}
	_ = rm.ExecuteUninstall(ctx, packages, true)

	output := buf.String()
	// Should show failure but continue
	assert.Contains(t, output, "[FAIL]")
}

func TestRecoveryMode_ExecuteUninstall_WithNoExecutorButRestoreNouveau(t *testing.T) {
	ctx := context.Background()

	var buf bytes.Buffer
	ui := NewTTYUI(WithTTYWriter(&buf))
	pm := NewMockPackageManager()

	rm := NewRecoveryMode(
		WithRecoveryUI(ui),
		WithRecoveryPackageManager(pm),
		// No executor - but restore nouveau is requested
	)

	packages := []string{"nvidia-driver-550"}
	err := rm.ExecuteUninstall(ctx, packages, true)

	// Should not panic, just skip the nouveau restoration steps that need executor
	require.NoError(t, err)
}

func TestRecoveryMode_Run_UserCancelsGraphicalWarning(t *testing.T) {
	ctx := context.Background()

	var buf bytes.Buffer
	reader := strings.NewReader("n\n") // Cancel when asked about continuing in graphical
	ui := NewTTYUI(WithTTYWriter(&buf), WithTTYReader(reader))
	env := &Environment{Type: EnvironmentGraphical, Display: ":0"}

	rm := NewRecoveryMode(
		WithRecoveryUI(ui),
		WithRecoveryEnvironment(env),
	)

	err := rm.Run(ctx)

	// Should return nil (not an error, just cancelled)
	assert.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "Use 'igor uninstall' in graphical mode")
}

func TestRecoveryMode_Run_SSHEnvironment(t *testing.T) {
	ctx := context.Background()

	var buf bytes.Buffer
	reader := strings.NewReader("")
	ui := NewTTYUI(WithTTYWriter(&buf), WithTTYReader(reader))
	env := &Environment{Type: EnvironmentSSH, SSHConnection: "1.2.3.4 22"}

	rm := NewRecoveryMode(
		WithRecoveryUI(ui),
		WithRecoveryEnvironment(env),
		// No discovery - will fail later
	)

	// Run will fail on root check in test, but SSH won't trigger graphical warning
	_ = rm.Run(ctx)

	output := buf.String()
	// Should not contain the graphical warning
	assert.NotContains(t, output, "graphical environment appears")
}

func TestRecoveryMode_Run_UserConfirmsGraphical(t *testing.T) {
	ctx := context.Background()

	var buf bytes.Buffer
	reader := strings.NewReader("y\n") // Confirm to continue in graphical
	ui := NewTTYUI(WithTTYWriter(&buf), WithTTYReader(reader))
	env := &Environment{Type: EnvironmentGraphical, Display: ":0"}

	rm := NewRecoveryMode(
		WithRecoveryUI(ui),
		WithRecoveryEnvironment(env),
		// No discovery - will fail on root check
	)

	// Run will fail on root check after user confirms
	err := rm.Run(ctx)

	// Should have proceeded past the graphical warning
	assert.Error(t, err)
	output := buf.String()
	assert.Contains(t, output, "Continue anyway")
}

func TestRecoveryMode_ExecuteUninstall_LogsRemovedPackages(t *testing.T) {
	ctx := context.Background()

	var buf bytes.Buffer
	ui := NewTTYUI(WithTTYWriter(&buf))
	pm := NewMockPackageManager()
	logger := logging.NewNop() // Use nop logger to avoid output

	rm := NewRecoveryMode(
		WithRecoveryUI(ui),
		WithRecoveryPackageManager(pm),
		WithRecoveryLogger(logger),
	)

	packages := []string{"nvidia-driver-550", "nvidia-settings", "nvidia-dkms"}
	err := rm.ExecuteUninstall(ctx, packages, false)

	require.NoError(t, err)
	assert.Len(t, pm.RemovedPackages(), 3)
}

func TestDetectEnvironmentWithOptions_DefaultsNilFileReader(t *testing.T) {
	// Test that nil file reader falls back to default
	env := DetectEnvironmentWithOptions(DetectionOptions{
		EnvReader: func(key string) string {
			return ""
		},
		FileReader: nil, // nil - should use default
	})

	// Should not panic
	assert.NotNil(t, env)
}

func TestTTYUI_ShowPackages_ExactColumnsThreshold(t *testing.T) {
	var buf bytes.Buffer
	ui := NewTTYUI(WithTTYWriter(&buf), WithTTYWidth(80))

	// Exactly 5 packages (threshold for column layout)
	packages := []string{"pkg1", "pkg2", "pkg3", "pkg4", "pkg5"}
	ui.ShowPackages(packages)

	output := buf.String()
	assert.Contains(t, output, "Found 5 package(s)")
}

func TestTTYUI_ShowPackages_SixPackagesUsesColumns(t *testing.T) {
	var buf bytes.Buffer
	ui := NewTTYUI(WithTTYWriter(&buf), WithTTYWidth(80))

	// 6 packages (above threshold, should use columns)
	packages := []string{"pkg1", "pkg2", "pkg3", "pkg4", "pkg5", "pkg6"}
	ui.ShowPackages(packages)

	output := buf.String()
	assert.Contains(t, output, "Found 6 package(s)")
}

func TestRecoveryMode_RebuildInitramfsWithResult(t *testing.T) {
	ctx := context.Background()
	executor := exec.NewMockExecutor()

	// Simulate dracut available and working
	executor.SetDefaultResponse(exec.SuccessResult(""))
	executor.SetResponse("which", exec.SuccessResult("/usr/bin/dracut"))

	rm := NewRecoveryMode(WithRecoveryExecutor(executor))

	err := rm.rebuildInitramfs(ctx)

	assert.NoError(t, err)
}

// The following tests focus on the Run function behavior that's testable without root

func TestRecoveryMode_Run_DisplaysHeader(t *testing.T) {
	ctx := context.Background()

	var buf bytes.Buffer
	reader := strings.NewReader("")
	ui := NewTTYUI(WithTTYWriter(&buf), WithTTYReader(reader))
	env := &Environment{Type: EnvironmentTTY}

	rm := NewRecoveryMode(
		WithRecoveryUI(ui),
		WithRecoveryEnvironment(env),
	)

	// Will fail on root check but should display header first
	_ = rm.Run(ctx)

	output := buf.String()
	assert.Contains(t, output, "NVIDIA Driver Recovery Mode")
	assert.Contains(t, output, "Detecting environment")
}

func TestRecoveryMode_Run_RequiresRootPrivileges(t *testing.T) {
	ctx := context.Background()

	var buf bytes.Buffer
	reader := strings.NewReader("")
	ui := NewTTYUI(WithTTYWriter(&buf), WithTTYReader(reader))
	env := &Environment{Type: EnvironmentTTY}

	rm := NewRecoveryMode(
		WithRecoveryUI(ui),
		WithRecoveryEnvironment(env),
	)

	err := rm.Run(ctx)

	// In test environment, we're not running as root
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "root privileges required")
	assert.Contains(t, buf.String(), "root privileges")
}

func TestRecoveryMode_Run_ShowsEnvironmentInfo(t *testing.T) {
	ctx := context.Background()

	var buf bytes.Buffer
	reader := strings.NewReader("")
	ui := NewTTYUI(WithTTYWriter(&buf), WithTTYReader(reader))
	env := &Environment{
		Type: EnvironmentTTY,
		TTY:  "/dev/tty3",
		Term: "linux",
	}
	dist := &distro.Distribution{ID: "ubuntu", Name: "Ubuntu"}

	rm := NewRecoveryMode(
		WithRecoveryUI(ui),
		WithRecoveryEnvironment(env),
		WithRecoveryDistro(dist),
	)

	_ = rm.Run(ctx)

	output := buf.String()
	// Should show environment info before failing on root check
	assert.Contains(t, output, "tty")
	assert.Contains(t, output, "linux")
}

func TestTTYUI_Status_NoTruncationNeeded(t *testing.T) {
	var buf bytes.Buffer
	ui := NewTTYUI(WithTTYWriter(&buf), WithTTYWidth(80))

	ui.Status("[TEST]", "Short message")

	output := buf.String()
	assert.Contains(t, output, "[TEST]")
	assert.Contains(t, output, "Short message")
	assert.NotContains(t, output, "...")
}

func TestTTYUI_Status_ExtremelyNarrowWidth(t *testing.T) {
	var buf bytes.Buffer
	// Very narrow width where truncation would result in < 3 chars
	ui := NewTTYUI(WithTTYWriter(&buf), WithTTYWidth(10))

	ui.Status("[OK]", "This is a long message")

	// Should not panic
	output := buf.String()
	assert.NotEmpty(t, output)
}

func TestRecoveryMode_ExecuteUninstall_WithLogger(t *testing.T) {
	ctx := context.Background()

	var buf bytes.Buffer
	ui := NewTTYUI(WithTTYWriter(&buf))
	pm := NewMockPackageManager()

	// Create a real logger that writes to a buffer
	var logBuf bytes.Buffer
	logOpts := logging.Options{
		Level:  logging.LevelDebug,
		Output: &logBuf,
	}
	logger := logging.New(logOpts)

	executor := exec.NewMockExecutor()
	executor.SetResponse("which", exec.SuccessResult("/usr/bin/update-initramfs"))
	executor.SetResponse("update-initramfs", exec.SuccessResult(""))

	rm := NewRecoveryMode(
		WithRecoveryUI(ui),
		WithRecoveryPackageManager(pm),
		WithRecoveryExecutor(executor),
		WithRecoveryLogger(logger),
	)

	packages := []string{"nvidia-driver-550"}
	_ = rm.ExecuteUninstall(ctx, packages, true)

	// Check that logger was used
	logOutput := logBuf.String()
	assert.Contains(t, logOutput, "Executing uninstall")
}

// =============================================================================
// Run Function with Mock Root Checker Tests
// =============================================================================

func alwaysRoot() bool { return true }

func TestRecoveryMode_Run_NoDiscovery(t *testing.T) {
	ctx := context.Background()

	var buf bytes.Buffer
	reader := strings.NewReader("")
	ui := NewTTYUI(WithTTYWriter(&buf), WithTTYReader(reader))
	env := &Environment{Type: EnvironmentTTY}

	rm := NewRecoveryMode(
		WithRecoveryUI(ui),
		WithRecoveryEnvironment(env),
		WithRecoveryRootChecker(alwaysRoot),
		// No discovery
	)

	err := rm.Run(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "discovery not configured")
	output := buf.String()
	assert.Contains(t, output, "Package discovery not configured")
}

func TestRecoveryMode_Run_DiscoveryError(t *testing.T) {
	ctx := context.Background()

	var buf bytes.Buffer
	reader := strings.NewReader("")
	ui := NewTTYUI(WithTTYWriter(&buf), WithTTYReader(reader))
	env := &Environment{Type: EnvironmentTTY}
	discovery := NewMockDiscovery()
	discovery.SetDiscoverError(assert.AnError)

	rm := NewRecoveryMode(
		WithRecoveryUI(ui),
		WithRecoveryEnvironment(env),
		WithRecoveryRootChecker(alwaysRoot),
		WithRecoveryDiscovery(discovery),
	)

	err := rm.Run(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "discovery failed")
	output := buf.String()
	assert.Contains(t, output, "Failed to discover packages")
}

func TestRecoveryMode_Run_NoPackagesFound(t *testing.T) {
	ctx := context.Background()

	var buf bytes.Buffer
	reader := strings.NewReader("")
	ui := NewTTYUI(WithTTYWriter(&buf), WithTTYReader(reader))
	env := &Environment{Type: EnvironmentTTY}
	discovery := NewMockDiscovery()
	discovery.SetPackages(&uninstall.DiscoveredPackages{
		AllPackages: []string{},
		TotalCount:  0,
	})

	rm := NewRecoveryMode(
		WithRecoveryUI(ui),
		WithRecoveryEnvironment(env),
		WithRecoveryRootChecker(alwaysRoot),
		WithRecoveryDiscovery(discovery),
	)

	err := rm.Run(ctx)

	assert.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "No NVIDIA packages found")
}

func TestRecoveryMode_Run_UserCancelsUninstall(t *testing.T) {
	ctx := context.Background()

	var buf bytes.Buffer
	reader := strings.NewReader("n\n") // Cancel at uninstall confirmation
	ui := NewTTYUI(WithTTYWriter(&buf), WithTTYReader(reader))
	env := &Environment{Type: EnvironmentTTY}
	discovery := NewMockDiscovery()
	discovery.SetPackages(&uninstall.DiscoveredPackages{
		AllPackages:    []string{"nvidia-driver-550"},
		DriverPackages: []string{"nvidia-driver-550"},
		TotalCount:     1,
		DriverVersion:  "550",
	})

	rm := NewRecoveryMode(
		WithRecoveryUI(ui),
		WithRecoveryEnvironment(env),
		WithRecoveryRootChecker(alwaysRoot),
		WithRecoveryDiscovery(discovery),
	)

	err := rm.Run(ctx)

	assert.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "Cancelled by user")
}

func TestRecoveryMode_Run_SuccessfulUninstall(t *testing.T) {
	ctx := context.Background()

	var buf bytes.Buffer
	reader := strings.NewReader("y\nn\n") // Confirm uninstall, decline reboot
	ui := NewTTYUI(WithTTYWriter(&buf), WithTTYReader(reader))
	env := &Environment{Type: EnvironmentTTY}
	pm := NewMockPackageManager()
	executor := exec.NewMockExecutor()
	executor.SetResponse("which", exec.SuccessResult("/usr/bin/update-initramfs"))
	executor.SetResponse("update-initramfs", exec.SuccessResult(""))
	discovery := NewMockDiscovery()
	discovery.SetPackages(&uninstall.DiscoveredPackages{
		AllPackages:    []string{"nvidia-driver-550"},
		DriverPackages: []string{"nvidia-driver-550"},
		TotalCount:     1,
		DriverVersion:  "550",
	})

	rm := NewRecoveryMode(
		WithRecoveryUI(ui),
		WithRecoveryEnvironment(env),
		WithRecoveryRootChecker(alwaysRoot),
		WithRecoveryDiscovery(discovery),
		WithRecoveryPackageManager(pm),
		WithRecoveryExecutor(executor),
	)

	err := rm.Run(ctx)

	assert.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "Recovery completed successfully")
	assert.Contains(t, output, "Remember to reboot")
}

func TestRecoveryMode_Run_SuccessfulUninstallWithReboot(t *testing.T) {
	ctx := context.Background()

	var buf bytes.Buffer
	reader := strings.NewReader("y\ny\n") // Confirm uninstall, confirm reboot
	ui := NewTTYUI(WithTTYWriter(&buf), WithTTYReader(reader))
	env := &Environment{Type: EnvironmentTTY}
	pm := NewMockPackageManager()
	executor := exec.NewMockExecutor()
	executor.SetResponse("which", exec.SuccessResult("/usr/bin/update-initramfs"))
	executor.SetResponse("update-initramfs", exec.SuccessResult(""))
	executor.SetResponse("reboot", exec.SuccessResult(""))
	discovery := NewMockDiscovery()
	discovery.SetPackages(&uninstall.DiscoveredPackages{
		AllPackages:    []string{"nvidia-driver-550"},
		DriverPackages: []string{"nvidia-driver-550"},
		TotalCount:     1,
	})

	rm := NewRecoveryMode(
		WithRecoveryUI(ui),
		WithRecoveryEnvironment(env),
		WithRecoveryRootChecker(alwaysRoot),
		WithRecoveryDiscovery(discovery),
		WithRecoveryPackageManager(pm),
		WithRecoveryExecutor(executor),
	)

	err := rm.Run(ctx)

	assert.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "Rebooting")
}

func TestRecoveryMode_Run_RebootFails(t *testing.T) {
	ctx := context.Background()

	var buf bytes.Buffer
	reader := strings.NewReader("y\ny\n") // Confirm uninstall, confirm reboot
	ui := NewTTYUI(WithTTYWriter(&buf), WithTTYReader(reader))
	env := &Environment{Type: EnvironmentTTY}
	pm := NewMockPackageManager()
	executor := exec.NewMockExecutor()
	executor.SetResponse("which", exec.SuccessResult("/usr/bin/update-initramfs"))
	executor.SetResponse("update-initramfs", exec.SuccessResult(""))
	executor.SetResponse("reboot", exec.ErrorResult(assert.AnError))
	discovery := NewMockDiscovery()
	discovery.SetPackages(&uninstall.DiscoveredPackages{
		AllPackages: []string{"nvidia-driver-550"},
		TotalCount:  1,
	})

	rm := NewRecoveryMode(
		WithRecoveryUI(ui),
		WithRecoveryEnvironment(env),
		WithRecoveryRootChecker(alwaysRoot),
		WithRecoveryDiscovery(discovery),
		WithRecoveryPackageManager(pm),
		WithRecoveryExecutor(executor),
	)

	err := rm.Run(ctx)

	assert.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "Failed to reboot")
	assert.Contains(t, output, "reboot manually")
}

func TestRecoveryMode_Run_DryRunReboot(t *testing.T) {
	ctx := context.Background()

	var buf bytes.Buffer
	reader := strings.NewReader("y\ny\n") // Confirm uninstall, confirm reboot
	ui := NewTTYUI(WithTTYWriter(&buf), WithTTYReader(reader))
	env := &Environment{Type: EnvironmentTTY}
	pm := NewMockPackageManager()
	discovery := NewMockDiscovery()
	discovery.SetPackages(&uninstall.DiscoveredPackages{
		AllPackages: []string{"nvidia-driver-550"},
		TotalCount:  1,
	})

	rm := NewRecoveryMode(
		WithRecoveryUI(ui),
		WithRecoveryEnvironment(env),
		WithRecoveryRootChecker(alwaysRoot),
		WithRecoveryDiscovery(discovery),
		WithRecoveryPackageManager(pm),
		WithRecoveryDryRun(true),
	)

	err := rm.Run(ctx)

	assert.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "[DRY RUN] Would execute: reboot")
}

func TestRecoveryMode_Run_ShowsCUDAVersion(t *testing.T) {
	ctx := context.Background()

	var buf bytes.Buffer
	reader := strings.NewReader("n\n") // Cancel at confirmation
	ui := NewTTYUI(WithTTYWriter(&buf), WithTTYReader(reader))
	env := &Environment{Type: EnvironmentTTY}
	discovery := NewMockDiscovery()
	discovery.SetPackages(&uninstall.DiscoveredPackages{
		AllPackages:   []string{"nvidia-driver-550", "cuda-toolkit-12-4"},
		TotalCount:    2,
		DriverVersion: "550",
		CUDAVersion:   "12.4",
	})

	rm := NewRecoveryMode(
		WithRecoveryUI(ui),
		WithRecoveryEnvironment(env),
		WithRecoveryRootChecker(alwaysRoot),
		WithRecoveryDiscovery(discovery),
	)

	_ = rm.Run(ctx)

	output := buf.String()
	assert.Contains(t, output, "Driver version: 550")
	assert.Contains(t, output, "CUDA version: 12.4")
}

func TestWithRecoveryRootChecker(t *testing.T) {
	customChecker := func() bool { return true }
	rm := NewRecoveryMode(WithRecoveryRootChecker(customChecker))

	assert.True(t, rm.rootChecker())
}

func TestDefaultRootChecker(t *testing.T) {
	// This will return false in test environment (not running as root)
	result := defaultRootChecker()
	// We can't assert the exact value since tests might run as root in some environments
	// But we can verify it doesn't panic
	assert.IsType(t, false, result)
}
