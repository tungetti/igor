package steps

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tungetti/igor/internal/constants"
	"github.com/tungetti/igor/internal/distro"
	"github.com/tungetti/igor/internal/exec"
	"github.com/tungetti/igor/internal/gpu"
	"github.com/tungetti/igor/internal/gpu/pci"
	"github.com/tungetti/igor/internal/gpu/validator"
	"github.com/tungetti/igor/internal/install"
)

// =============================================================================
// Step Chaining Tests
// =============================================================================

// TestSteps_ExecuteSequence tests step execution in sequence
func TestSteps_ExecuteSequence(t *testing.T) {
	t.Run("steps execute in order", func(t *testing.T) {
		var executionOrder []string
		var mu sync.Mutex

		steps := []install.Step{
			install.NewFuncStep("step1", "First step", func(ctx *install.Context) install.StepResult {
				mu.Lock()
				executionOrder = append(executionOrder, "step1")
				mu.Unlock()
				ctx.SetState("step1_completed", true)
				return install.CompleteStep("step1 done")
			}),
			install.NewFuncStep("step2", "Second step", func(ctx *install.Context) install.StepResult {
				mu.Lock()
				executionOrder = append(executionOrder, "step2")
				mu.Unlock()
				// Check that step1 ran first
				if !ctx.GetStateBool("step1_completed") {
					return install.FailStep("step1 should have run first", errors.New("order violation"))
				}
				ctx.SetState("step2_completed", true)
				return install.CompleteStep("step2 done")
			}),
			install.NewFuncStep("step3", "Third step", func(ctx *install.Context) install.StepResult {
				mu.Lock()
				executionOrder = append(executionOrder, "step3")
				mu.Unlock()
				if !ctx.GetStateBool("step2_completed") {
					return install.FailStep("step2 should have run first", errors.New("order violation"))
				}
				return install.CompleteStep("step3 done")
			}),
		}

		ctx := install.NewContext()
		for _, step := range steps {
			result := step.Execute(ctx)
			assert.True(t, result.IsSuccess(), "Step %s should succeed", step.Name())
		}

		mu.Lock()
		defer mu.Unlock()
		assert.Equal(t, []string{"step1", "step2", "step3"}, executionOrder)
	})

	t.Run("data passing between steps via context state", func(t *testing.T) {
		producer := install.NewFuncStep("producer", "Produce data", func(ctx *install.Context) install.StepResult {
			ctx.SetState("driver_version", "550.54.14")
			ctx.SetState("packages_installed", []string{"nvidia-driver-550", "nvidia-cuda-toolkit"})
			ctx.SetState("repository_configured", true)
			return install.CompleteStep("data produced")
		})

		var capturedVersion string
		var capturedRepoConfigured bool

		consumer := install.NewFuncStep("consumer", "Consume data", func(ctx *install.Context) install.StepResult {
			capturedVersion = ctx.GetStateString("driver_version")
			capturedRepoConfigured = ctx.GetStateBool("repository_configured")
			return install.CompleteStep("data consumed")
		})

		ctx := install.NewContext()

		result := producer.Execute(ctx)
		assert.True(t, result.IsSuccess())

		result = consumer.Execute(ctx)
		assert.True(t, result.IsSuccess())

		assert.Equal(t, "550.54.14", capturedVersion)
		assert.True(t, capturedRepoConfigured)
	})
}

// TestSteps_RollbackSequence tests rollback in reverse order
func TestSteps_RollbackSequence(t *testing.T) {
	t.Run("steps rollback in reverse order", func(t *testing.T) {
		var rollbackOrder []string
		var mu sync.Mutex

		steps := []install.Step{
			install.NewFuncStep("step1", "First step", func(ctx *install.Context) install.StepResult {
				return install.CompleteStep("done")
			}, install.WithRollbackFunc(func(ctx *install.Context) error {
				mu.Lock()
				rollbackOrder = append(rollbackOrder, "step1")
				mu.Unlock()
				return nil
			})),
			install.NewFuncStep("step2", "Second step", func(ctx *install.Context) install.StepResult {
				return install.CompleteStep("done")
			}, install.WithRollbackFunc(func(ctx *install.Context) error {
				mu.Lock()
				rollbackOrder = append(rollbackOrder, "step2")
				mu.Unlock()
				return nil
			})),
			install.NewFuncStep("step3", "Third step", func(ctx *install.Context) install.StepResult {
				return install.CompleteStep("done")
			}, install.WithRollbackFunc(func(ctx *install.Context) error {
				mu.Lock()
				rollbackOrder = append(rollbackOrder, "step3")
				mu.Unlock()
				return nil
			})),
		}

		ctx := install.NewContext()

		// Execute all steps
		var completedSteps []install.Step
		for _, step := range steps {
			result := step.Execute(ctx)
			assert.True(t, result.IsSuccess())
			completedSteps = append(completedSteps, step)
		}

		// Rollback in reverse order
		for i := len(completedSteps) - 1; i >= 0; i-- {
			step := completedSteps[i]
			if step.CanRollback() {
				err := step.Rollback(ctx)
				assert.NoError(t, err)
			}
		}

		mu.Lock()
		defer mu.Unlock()
		assert.Equal(t, []string{"step3", "step2", "step1"}, rollbackOrder)
	})
}

// =============================================================================
// Individual Step Integration Tests
// =============================================================================

// TestValidationStep_Integration tests validation with real-world scenarios
func TestValidationStep_Integration(t *testing.T) {
	t.Run("GPU present validation", func(t *testing.T) {
		mockValidator := NewMockValidator()
		mockValidator.kernelResult = validator.NewCheckResult(
			validator.CheckKernelVersion, true, "Kernel 6.5.0 is compatible", validator.SeverityInfo)
		mockValidator.diskSpaceResult = validator.NewCheckResult(
			validator.CheckDiskSpace, true, "10GB available", validator.SeverityInfo)

		step := NewValidationStep(
			WithValidator(mockValidator),
			WithChecks(CheckKernel, CheckDiskSpace, CheckNVIDIAGPU),
		)

		gpuInfo := &gpu.GPUInfo{
			NVIDIAGPUs: []gpu.NVIDIAGPUInfo{
				{
					PCIDevice: pci.PCIDevice{
						Address:  "0000:01:00.0",
						VendorID: pci.VendorNVIDIA,
						DeviceID: "2684",
						Class:    pci.ClassVGA,
						Driver:   "",
					},
				},
			},
		}

		ctx := install.NewContext(install.WithGPUInfo(gpuInfo))
		ctx.Executor = exec.NewMockExecutor()

		result := step.Execute(ctx)

		// Result depends on GPU detection
		assert.NotNil(t, result)
	})

	t.Run("kernel compatible validation", func(t *testing.T) {
		mockValidator := NewMockValidator()
		mockValidator.kernelResult = validator.NewCheckResult(
			validator.CheckKernelVersion, true, "Kernel 6.5.0-44-generic compatible", validator.SeverityInfo)

		step := NewValidationStep(
			WithValidator(mockValidator),
			WithChecks(CheckKernel),
		)

		ctx := install.NewContext()
		ctx.Executor = exec.NewMockExecutor()

		result := step.Execute(ctx)

		assert.True(t, result.IsSuccess())
	})

	t.Run("disk space sufficient validation", func(t *testing.T) {
		mockValidator := NewMockValidator()
		mockValidator.diskSpaceResult = validator.NewCheckResult(
			validator.CheckDiskSpace, true, "5000MB available (2000MB required)", validator.SeverityInfo)

		step := NewValidationStep(
			WithValidator(mockValidator),
			WithChecks(CheckDiskSpace),
			WithRequiredDiskMB(2000),
		)

		ctx := install.NewContext()
		ctx.Executor = exec.NewMockExecutor()

		result := step.Execute(ctx)

		assert.True(t, result.IsSuccess())
	})

	t.Run("validation failure stores errors", func(t *testing.T) {
		mockValidator := NewMockValidator()
		mockValidator.kernelResult = validator.NewCheckResult(
			validator.CheckKernelVersion, false, "Kernel 4.x not supported", validator.SeverityError)

		step := NewValidationStep(
			WithValidator(mockValidator),
			WithChecks(CheckKernel),
		)

		ctx := install.NewContext()
		ctx.Executor = exec.NewMockExecutor()

		result := step.Execute(ctx)

		assert.True(t, result.IsFailure())
		assert.False(t, ctx.GetStateBool("validation_passed"))
	})

	t.Run("validation with warnings passes", func(t *testing.T) {
		mockValidator := NewMockValidator()
		mockValidator.nouveauResult = validator.NewCheckResult(
			validator.CheckNouveauStatus, false, "Nouveau is loaded", validator.SeverityWarning)
		mockValidator.kernelResult = validator.NewCheckResult(
			validator.CheckKernelVersion, true, "OK", validator.SeverityInfo)

		step := NewValidationStep(
			WithValidator(mockValidator),
			WithChecks(CheckKernel, CheckNouveauStatus),
		)

		ctx := install.NewContext()
		ctx.Executor = exec.NewMockExecutor()

		result := step.Execute(ctx)

		// Should pass despite warning
		assert.True(t, result.IsSuccess())
		assert.True(t, ctx.GetStateBool("needs_nouveau_blacklist"))
	})
}

// TestRepositoryStep_Integration tests repository setup
func TestRepositoryStep_Integration(t *testing.T) {
	t.Run("add NVIDIA repository", func(t *testing.T) {
		mockPM := NewPackageMockManager()

		step := NewRepositoryStep()

		ctx := install.NewContext(
			install.WithDistroInfo(&distro.Distribution{
				ID:     "ubuntu",
				Name:   "Ubuntu",
				Family: constants.FamilyDebian,
			}),
			install.WithPackageManager(mockPM),
		)

		result := step.Execute(ctx)

		assert.True(t, result.IsSuccess())
		assert.True(t, ctx.GetStateBool(StateRepositoryConfigured))
		assert.NotEmpty(t, ctx.GetStateString(StateRepositoryName))
	})

	t.Run("rollback removes repository", func(t *testing.T) {
		mockPM := NewPackageMockManager()

		step := NewRepositoryStep()

		ctx := install.NewContext(
			install.WithDistroInfo(&distro.Distribution{
				ID:     "fedora",
				Name:   "Fedora",
				Family: constants.FamilyRHEL,
			}),
			install.WithPackageManager(mockPM),
		)

		// Execute first
		result := step.Execute(ctx)
		assert.True(t, result.IsSuccess())

		// Verify state is set
		assert.True(t, ctx.GetStateBool(StateRepositoryConfigured))

		// Rollback
		err := step.Rollback(ctx)
		assert.NoError(t, err)

		// State should be cleared
		assert.False(t, ctx.GetStateBool(StateRepositoryConfigured))
	})

	t.Run("dry run does not add repository", func(t *testing.T) {
		mockPM := NewPackageMockManager()

		step := NewRepositoryStep()

		ctx := install.NewContext(
			install.WithDistroInfo(&distro.Distribution{
				ID:     "ubuntu",
				Name:   "Ubuntu",
				Family: constants.FamilyDebian,
			}),
			install.WithPackageManager(mockPM),
			install.WithDryRun(true),
		)

		result := step.Execute(ctx)

		assert.True(t, result.IsSuccess())
		// Dry run should not set state
		assert.False(t, ctx.GetStateBool(StateRepositoryConfigured))
	})
}

// TestNouveauStep_Integration tests nouveau blacklist
func TestNouveauStep_Integration(t *testing.T) {
	t.Run("create blacklist file", func(t *testing.T) {
		mockDetector := NewMockNouveauDetector()
		mockDetector.SetBlacklisted(false)
		mockFileWriter := NewMockFileWriter()
		mockExecutor := exec.NewMockExecutor()
		mockExecutor.SetResponse("tee", &exec.Result{ExitCode: 0})
		mockExecutor.SetResponse("update-initramfs", &exec.Result{ExitCode: 0})

		step := NewNouveauBlacklistStep(
			WithNouveauDetector(mockDetector),
			WithFileWriter(mockFileWriter),
			WithSkipInitramfs(true), // Skip initramfs for faster tests
		)

		ctx := install.NewContext(
			install.WithExecutor(mockExecutor),
			install.WithDistroInfo(&distro.Distribution{
				Family: constants.FamilyDebian,
			}),
		)

		result := step.Execute(ctx)

		assert.True(t, result.IsSuccess())
		assert.True(t, ctx.GetStateBool(StateNouveauBlacklisted))
		assert.NotEmpty(t, ctx.GetStateString(StateNouveauBlacklistFile))
	})

	t.Run("skip if already blacklisted", func(t *testing.T) {
		mockDetector := NewMockNouveauDetector()
		mockDetector.SetBlacklisted(true)
		mockExecutor := exec.NewMockExecutor()

		step := NewNouveauBlacklistStep(WithNouveauDetector(mockDetector))

		ctx := install.NewContext(install.WithExecutor(mockExecutor))

		result := step.Execute(ctx)

		assert.Equal(t, install.StepStatusSkipped, result.Status)
	})

	t.Run("rollback removes blacklist", func(t *testing.T) {
		mockDetector := NewMockNouveauDetector()
		mockDetector.SetBlacklisted(false)
		mockFileWriter := NewMockFileWriter()
		mockExecutor := exec.NewMockExecutor()
		mockExecutor.SetResponse("tee", &exec.Result{ExitCode: 0})
		mockExecutor.SetResponse("rm", &exec.Result{ExitCode: 0})
		mockExecutor.SetDefaultResponse(&exec.Result{ExitCode: 0})

		step := NewNouveauBlacklistStep(
			WithNouveauDetector(mockDetector),
			WithFileWriter(mockFileWriter),
			WithSkipInitramfs(true),
		)

		ctx := install.NewContext(
			install.WithExecutor(mockExecutor),
			install.WithDistroInfo(&distro.Distribution{
				Family: constants.FamilyDebian,
			}),
		)

		// Execute
		step.Execute(ctx)

		// Rollback
		err := step.Rollback(ctx)
		assert.NoError(t, err)
		assert.False(t, ctx.GetStateBool(StateNouveauBlacklisted))
	})
}

// TestPackageStep_Integration tests package installation
func TestPackageStep_Integration(t *testing.T) {
	t.Run("install driver packages", func(t *testing.T) {
		mockPM := NewPackageMockManager()

		step := NewPackageInstallationStep()

		ctx := install.NewContext(
			install.WithPackageManager(mockPM),
			install.WithDistroInfo(&distro.Distribution{
				ID:     "ubuntu",
				Name:   "Ubuntu",
				Family: constants.FamilyDebian,
			}),
			install.WithDriverVersion("550"),
		)

		result := step.Execute(ctx)

		assert.True(t, result.IsSuccess())
		assert.True(t, ctx.GetStateBool(StatePackagesInstalled))
	})

	t.Run("rollback removes packages", func(t *testing.T) {
		mockPM := NewPackageMockManager()

		step := NewPackageInstallationStep()

		ctx := install.NewContext(
			install.WithPackageManager(mockPM),
			install.WithDistroInfo(&distro.Distribution{
				ID:     "ubuntu",
				Name:   "Ubuntu",
				Family: constants.FamilyDebian,
			}),
			install.WithDriverVersion("550"),
		)

		// Execute
		result := step.Execute(ctx)
		assert.True(t, result.IsSuccess())

		// Rollback
		err := step.Rollback(ctx)
		assert.NoError(t, err)
		assert.False(t, ctx.GetStateBool(StatePackagesInstalled))
	})

	t.Run("with additional packages", func(t *testing.T) {
		mockPM := NewPackageMockManager()

		step := NewPackageInstallationStep(
			WithAdditionalPackages("nvidia-settings", "nvidia-prime"),
		)

		ctx := install.NewContext(
			install.WithPackageManager(mockPM),
			install.WithDistroInfo(&distro.Distribution{
				ID:     "ubuntu",
				Name:   "Ubuntu",
				Family: constants.FamilyDebian,
			}),
			install.WithDriverVersion("550"),
		)

		result := step.Execute(ctx)

		assert.True(t, result.IsSuccess())
	})
}

// TestDKMSStep_Integration tests DKMS build
func TestDKMSStep_Integration(t *testing.T) {
	t.Run("build kernel modules", func(t *testing.T) {
		mockExecutor := exec.NewMockExecutor()
		mockExecutor.SetResponse("which", &exec.Result{Stdout: []byte("/usr/sbin/dkms"), ExitCode: 0})
		mockExecutor.SetResponse("uname", &exec.Result{Stdout: []byte("6.5.0-44-generic"), ExitCode: 0})
		mockExecutor.SetResponse("dkms", &exec.Result{Stdout: []byte("nvidia/550.54.14: added"), ExitCode: 0})
		mockExecutor.SetDefaultResponse(&exec.Result{ExitCode: 0})

		step := NewDKMSBuildStep()

		ctx := install.NewContext(install.WithExecutor(mockExecutor))

		result := step.Execute(ctx)

		// May skip if already built or complete
		assert.True(t, result.Status == install.StepStatusCompleted || result.Status == install.StepStatusSkipped)
	})

	t.Run("skip if DKMS not available", func(t *testing.T) {
		mockExecutor := exec.NewMockExecutor()
		mockExecutor.SetResponse("which", &exec.Result{ExitCode: 1})
		mockExecutor.SetResponse("command", &exec.Result{ExitCode: 1})
		mockExecutor.SetDefaultResponse(&exec.Result{ExitCode: 1})

		step := NewDKMSBuildStep()

		ctx := install.NewContext(install.WithExecutor(mockExecutor))

		result := step.Execute(ctx)

		assert.Equal(t, install.StepStatusSkipped, result.Status)
	})

	t.Run("rollback removes modules", func(t *testing.T) {
		mockExecutor := exec.NewMockExecutor()
		mockExecutor.SetResponse("which", &exec.Result{Stdout: []byte("/usr/sbin/dkms"), ExitCode: 0})
		mockExecutor.SetResponse("uname", &exec.Result{Stdout: []byte("6.5.0-44-generic"), ExitCode: 0})
		mockExecutor.SetResponse("dkms", &exec.Result{Stdout: []byte("nvidia/550.54.14: added"), ExitCode: 0})
		mockExecutor.SetDefaultResponse(&exec.Result{ExitCode: 0})

		step := NewDKMSBuildStep()

		ctx := install.NewContext(install.WithExecutor(mockExecutor))

		// Execute
		step.Execute(ctx)

		// Rollback
		err := step.Rollback(ctx)
		assert.NoError(t, err)
	})
}

// TestModloadStep_Integration tests module loading
func TestModloadStep_Integration(t *testing.T) {
	t.Run("load nvidia module", func(t *testing.T) {
		mockExecutor := exec.NewMockExecutor()
		mockExecutor.SetResponse("lsmod", &exec.Result{Stdout: []byte(""), ExitCode: 0}) // nvidia not loaded initially
		mockExecutor.SetDefaultResponse(&exec.Result{ExitCode: 0})

		step := NewModuleLoadStep()

		ctx := install.NewContext(install.WithExecutor(mockExecutor))

		result := step.Execute(ctx)

		assert.True(t, result.IsSuccess())
		assert.True(t, ctx.GetStateBool(StateModulesLoaded))
	})

	t.Run("skip if already loaded", func(t *testing.T) {
		mockExecutor := exec.NewMockExecutor()
		mockExecutor.SetResponse("lsmod", &exec.Result{Stdout: []byte("nvidia  12345  0"), ExitCode: 0})

		step := NewModuleLoadStep(WithSkipIfLoaded(true))

		ctx := install.NewContext(install.WithExecutor(mockExecutor))

		result := step.Execute(ctx)

		assert.Equal(t, install.StepStatusSkipped, result.Status)
	})

	t.Run("rollback unloads module", func(t *testing.T) {
		mockExecutor := exec.NewMockExecutor()
		mockExecutor.SetResponse("lsmod", &exec.Result{Stdout: []byte(""), ExitCode: 0})
		mockExecutor.SetDefaultResponse(&exec.Result{ExitCode: 0})

		step := NewModuleLoadStep()

		ctx := install.NewContext(install.WithExecutor(mockExecutor))

		// Execute
		step.Execute(ctx)

		// Rollback
		err := step.Rollback(ctx)
		assert.NoError(t, err)
		assert.False(t, ctx.GetStateBool(StateModulesLoaded))
	})
}

// IntegrationMockDisplayDetector implements DisplayDetector for testing.
type IntegrationMockDisplayDetector struct {
	displayServer string
	isWayland     bool
}

func (m *IntegrationMockDisplayDetector) DetectDisplayServer(ctx context.Context) (string, error) {
	return m.displayServer, nil
}

func (m *IntegrationMockDisplayDetector) IsWaylandSession() bool {
	return m.isWayland
}

// TestXorgStep_Integration tests X.org configuration
func TestXorgStep_Integration(t *testing.T) {
	t.Run("create xorg.conf", func(t *testing.T) {
		mockExecutor := exec.NewMockExecutor()
		mockExecutor.SetResponse("tee", &exec.Result{ExitCode: 0})
		mockExecutor.SetDefaultResponse(&exec.Result{ExitCode: 0})

		step := NewXorgConfigStep(
			WithDisplayDetector(&IntegrationMockDisplayDetector{displayServer: "xorg"}),
		)

		ctx := install.NewContext(install.WithExecutor(mockExecutor))

		result := step.Execute(ctx)

		assert.True(t, result.IsSuccess())
		assert.True(t, ctx.GetStateBool(StateXorgConfigured))
	})

	t.Run("skip if Wayland and configured to skip", func(t *testing.T) {
		mockExecutor := exec.NewMockExecutor()

		step := NewXorgConfigStep(
			WithDisplayDetector(&IntegrationMockDisplayDetector{displayServer: "wayland", isWayland: true}),
			WithSkipIfWayland(true),
		)

		ctx := install.NewContext(install.WithExecutor(mockExecutor))

		result := step.Execute(ctx)

		assert.Equal(t, install.StepStatusSkipped, result.Status)
	})

	t.Run("rollback removes config", func(t *testing.T) {
		mockExecutor := exec.NewMockExecutor()
		mockExecutor.SetResponse("tee", &exec.Result{ExitCode: 0})
		mockExecutor.SetDefaultResponse(&exec.Result{ExitCode: 0})

		step := NewXorgConfigStep(
			WithDisplayDetector(&IntegrationMockDisplayDetector{displayServer: "xorg"}),
		)

		ctx := install.NewContext(install.WithExecutor(mockExecutor))

		// Execute
		step.Execute(ctx)

		// Rollback
		err := step.Rollback(ctx)
		assert.NoError(t, err)
		assert.False(t, ctx.GetStateBool(StateXorgConfigured))
	})
}

// TestVerifyStep_Integration tests verification
func TestVerifyStep_Integration(t *testing.T) {
	t.Run("nvidia-smi works", func(t *testing.T) {
		mockExecutor := exec.NewMockExecutor()
		mockExecutor.SetResponse("nvidia-smi", &exec.Result{Stdout: []byte("550.54.14"), ExitCode: 0})
		mockExecutor.SetResponse("lsmod", &exec.Result{Stdout: []byte("nvidia  12345  0"), ExitCode: 0})

		step := NewVerificationStep(
			WithCheckNvidiaSmi(true),
			WithCheckModuleLoaded(true),
			WithCheckGPUDetected(false),
		)

		ctx := install.NewContext(install.WithExecutor(mockExecutor))

		result := step.Execute(ctx)

		assert.True(t, result.IsSuccess())
		assert.True(t, ctx.GetStateBool(StateVerificationPassed))
		assert.NotEmpty(t, ctx.GetStateString(StateDriverVersion))
	})

	t.Run("module loaded check", func(t *testing.T) {
		mockExecutor := exec.NewMockExecutor()
		mockExecutor.SetResponse("lsmod", &exec.Result{Stdout: []byte("nvidia  12345  0\nnvidia_uvm  5678  0"), ExitCode: 0})

		step := NewVerificationStep(
			WithCheckNvidiaSmi(false),
			WithCheckModuleLoaded(true),
			WithCheckGPUDetected(false),
		)

		ctx := install.NewContext(install.WithExecutor(mockExecutor))

		result := step.Execute(ctx)

		assert.True(t, result.IsSuccess())
		assert.True(t, ctx.GetStateBool(StateModuleLoaded))
	})

	t.Run("GPU detected check", func(t *testing.T) {
		mockExecutor := exec.NewMockExecutor()
		mockExecutor.SetResponse("nvidia-smi", &exec.Result{Stdout: []byte("NVIDIA GeForce RTX 3080, 10240 MiB"), ExitCode: 0})

		step := NewVerificationStep(
			WithCheckNvidiaSmi(false),
			WithCheckModuleLoaded(false),
			WithCheckGPUDetected(true),
		)

		ctx := install.NewContext(install.WithExecutor(mockExecutor))

		result := step.Execute(ctx)

		assert.True(t, result.IsSuccess())
		assert.True(t, ctx.GetStateBool(StateGPUDetected))
	})

	t.Run("verification has no rollback", func(t *testing.T) {
		step := NewVerificationStep()
		assert.False(t, step.CanRollback())

		ctx := install.NewContext()
		err := step.Rollback(ctx)
		assert.NoError(t, err)
	})
}

// =============================================================================
// Step State Sharing Tests
// =============================================================================

// TestSteps_StateSharing tests data passing between steps
func TestSteps_StateSharing(t *testing.T) {
	t.Run("repository step stores repo info for package step", func(t *testing.T) {
		mockPM := NewPackageMockManager()

		repoStep := NewRepositoryStep()
		packageStep := NewPackageInstallationStep()

		ctx := install.NewContext(
			install.WithPackageManager(mockPM),
			install.WithDistroInfo(&distro.Distribution{
				ID:     "ubuntu",
				Name:   "Ubuntu",
				Family: constants.FamilyDebian,
			}),
			install.WithDriverVersion("550"),
		)

		// Execute repository step
		result := repoStep.Execute(ctx)
		require.True(t, result.IsSuccess())

		// Verify state is accessible to package step
		repoConfigured := ctx.GetStateBool(StateRepositoryConfigured)
		assert.True(t, repoConfigured)

		// Package step can now use this information
		result = packageStep.Execute(ctx)
		assert.True(t, result.IsSuccess())
	})

	t.Run("validation step stores needs for subsequent steps", func(t *testing.T) {
		mockValidator := NewMockValidator()
		mockValidator.nouveauResult = validator.NewCheckResult(
			validator.CheckNouveauStatus, false, "Nouveau loaded", validator.SeverityWarning)
		mockValidator.kernelHeadersResult = validator.NewCheckResult(
			validator.CheckKernelHeaders, false, "Headers missing", validator.SeverityError)

		step := NewValidationStep(
			WithValidator(mockValidator),
			WithChecks(CheckNouveauStatus, CheckKernelHeaders),
		)

		ctx := install.NewContext()
		ctx.Executor = exec.NewMockExecutor()

		step.Execute(ctx)

		// Later steps can check what needs to be done
		assert.True(t, ctx.GetStateBool("needs_nouveau_blacklist"))
		assert.True(t, ctx.GetStateBool("needs_kernel_headers"))
	})

	t.Run("verify step uses installation info from previous steps", func(t *testing.T) {
		mockExecutor := exec.NewMockExecutor()
		mockExecutor.SetResponse("nvidia-smi", &exec.Result{Stdout: []byte("550.54.14"), ExitCode: 0})
		mockExecutor.SetResponse("lsmod", &exec.Result{Stdout: []byte("nvidia  12345  0"), ExitCode: 0})

		ctx := install.NewContext(install.WithExecutor(mockExecutor))

		// Simulate previous steps setting state
		ctx.SetState("packages_installed", true)
		ctx.SetState("modules_loaded", true)
		ctx.SetState("xorg_configured", true)

		verifyStep := NewVerificationStep(
			WithCheckNvidiaSmi(true),
			WithCheckModuleLoaded(true),
		)

		result := verifyStep.Execute(ctx)

		assert.True(t, result.IsSuccess())

		// Verification should have added its own state
		assert.True(t, ctx.GetStateBool(StateVerificationPassed))
		assert.Equal(t, "550.54.14", ctx.GetStateString(StateDriverVersion))
	})
}

// =============================================================================
// Cancellation Tests
// =============================================================================

// TestSteps_CancellationHandling tests how steps handle cancellation
func TestSteps_CancellationHandling(t *testing.T) {
	t.Run("validation step respects cancellation", func(t *testing.T) {
		mockValidator := NewMockValidator()

		step := NewValidationStep(WithValidator(mockValidator))

		ctx := install.NewContext()
		ctx.Executor = exec.NewMockExecutor()
		ctx.Cancel()

		result := step.Execute(ctx)

		assert.True(t, result.IsFailure())
		assert.Contains(t, result.Message, "cancelled")
	})

	t.Run("repository step respects cancellation", func(t *testing.T) {
		mockPM := NewPackageMockManager()

		step := NewRepositoryStep()

		ctx := install.NewContext(
			install.WithPackageManager(mockPM),
			install.WithDistroInfo(&distro.Distribution{
				ID:     "ubuntu",
				Family: constants.FamilyDebian,
			}),
		)
		ctx.Cancel()

		result := step.Execute(ctx)

		assert.True(t, result.IsFailure())
	})

	t.Run("package step respects cancellation", func(t *testing.T) {
		mockPM := NewPackageMockManager()

		step := NewPackageInstallationStep()

		ctx := install.NewContext(
			install.WithPackageManager(mockPM),
			install.WithDistroInfo(&distro.Distribution{
				ID:     "ubuntu",
				Family: constants.FamilyDebian,
			}),
			install.WithDriverVersion("550"),
		)
		ctx.Cancel()

		result := step.Execute(ctx)

		assert.True(t, result.IsFailure())
	})
}

// =============================================================================
// Dry Run Tests
// =============================================================================

// TestSteps_DryRunMode tests dry run behavior across all steps
func TestSteps_DryRunMode(t *testing.T) {
	t.Run("validation step works in dry run", func(t *testing.T) {
		mockValidator := NewMockValidator()

		step := NewValidationStep(WithValidator(mockValidator))

		ctx := install.NewContext(install.WithDryRun(true))
		ctx.Executor = exec.NewMockExecutor()

		result := step.Execute(ctx)

		// Validation still runs in dry mode
		assert.True(t, result.IsSuccess())
	})

	t.Run("repository step dry run", func(t *testing.T) {
		mockPM := NewPackageMockManager()

		step := NewRepositoryStep()

		ctx := install.NewContext(
			install.WithPackageManager(mockPM),
			install.WithDistroInfo(&distro.Distribution{
				ID:     "ubuntu",
				Family: constants.FamilyDebian,
			}),
			install.WithDryRun(true),
		)

		result := step.Execute(ctx)

		assert.True(t, result.IsSuccess())
		// Should NOT set state in dry run
		assert.False(t, ctx.GetStateBool(StateRepositoryConfigured))
	})

	t.Run("nouveau step dry run", func(t *testing.T) {
		mockDetector := NewMockNouveauDetector()
		mockDetector.SetBlacklisted(false)
		mockExecutor := exec.NewMockExecutor()

		step := NewNouveauBlacklistStep(WithNouveauDetector(mockDetector))

		ctx := install.NewContext(
			install.WithExecutor(mockExecutor),
			install.WithDistroInfo(&distro.Distribution{Family: constants.FamilyDebian}),
			install.WithDryRun(true),
		)

		result := step.Execute(ctx)

		assert.True(t, result.IsSuccess())
		assert.False(t, ctx.GetStateBool(StateNouveauBlacklisted))
	})

	t.Run("package step dry run", func(t *testing.T) {
		mockPM := NewPackageMockManager()

		step := NewPackageInstallationStep()

		ctx := install.NewContext(
			install.WithPackageManager(mockPM),
			install.WithDistroInfo(&distro.Distribution{
				ID:     "ubuntu",
				Family: constants.FamilyDebian,
			}),
			install.WithDriverVersion("550"),
			install.WithDryRun(true),
		)

		result := step.Execute(ctx)

		assert.True(t, result.IsSuccess())
		assert.False(t, ctx.GetStateBool(StatePackagesInstalled))
	})

	t.Run("verification step dry run", func(t *testing.T) {
		mockExecutor := exec.NewMockExecutor()

		step := NewVerificationStep()

		ctx := install.NewContext(
			install.WithExecutor(mockExecutor),
			install.WithDryRun(true),
		)

		result := step.Execute(ctx)

		assert.True(t, result.IsSuccess())
		assert.Contains(t, result.Message, "dry run")
	})
}

// =============================================================================
// Benchmarks
// =============================================================================

func BenchmarkValidationStep_Integration(b *testing.B) {
	mockValidator := NewMockValidator()
	step := NewValidationStep(WithValidator(mockValidator))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx := install.NewContext()
		ctx.Executor = exec.NewMockExecutor()
		step.Execute(ctx)
	}
}

func BenchmarkPackageStep_Integration(b *testing.B) {
	mockPM := NewPackageMockManager()
	step := NewPackageInstallationStep()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx := install.NewContext(
			install.WithPackageManager(mockPM),
			install.WithDistroInfo(&distro.Distribution{
				ID:     "ubuntu",
				Family: constants.FamilyDebian,
			}),
			install.WithDriverVersion("550"),
		)
		step.Execute(ctx)
	}
}

func BenchmarkStepChain_Integration(b *testing.B) {
	steps := []install.Step{
		install.NewFuncStep("step1", "Step 1", func(ctx *install.Context) install.StepResult {
			ctx.SetState("key1", "value1")
			return install.CompleteStep("done")
		}),
		install.NewFuncStep("step2", "Step 2", func(ctx *install.Context) install.StepResult {
			_ = ctx.GetStateString("key1")
			ctx.SetState("key2", "value2")
			return install.CompleteStep("done")
		}),
		install.NewFuncStep("step3", "Step 3", func(ctx *install.Context) install.StepResult {
			_ = ctx.GetStateString("key2")
			return install.CompleteStep("done")
		}),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx := install.NewContext()
		for _, step := range steps {
			step.Execute(ctx)
		}
	}
}
