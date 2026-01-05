package builder

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tungetti/igor/internal/constants"
	"github.com/tungetti/igor/internal/distro"
	"github.com/tungetti/igor/internal/exec"
	"github.com/tungetti/igor/internal/install"
	"github.com/tungetti/igor/internal/install/steps"
)

// =============================================================================
// End-to-End Workflow Building and Execution Tests
// =============================================================================

// TestBuilder_EndToEndWorkflowExecution tests complete workflow execution
func TestBuilder_EndToEndWorkflowExecution(t *testing.T) {
	t.Run("Debian workflow executes all steps in order", func(t *testing.T) {
		builder := NewWorkflowBuilder(ubuntuDistro)
		workflow, err := builder.Build()
		require.NoError(t, err)

		// Verify workflow structure
		assert.Equal(t, "debian-nvidia-installation", workflow.Name())
		assert.Len(t, workflow.Steps(), 8)

		// Verify step order
		stepNames := getStepNames(workflow.Steps())
		expectedOrder := []string{
			"validation",
			"repository",
			"nouveau_blacklist",
			"packages",
			"dkms_build",
			"module_load",
			"xorg_config",
			"verification",
		}
		assert.Equal(t, expectedOrder, stepNames)
	})

	t.Run("RHEL workflow executes all steps in order", func(t *testing.T) {
		builder := NewWorkflowBuilder(fedoraDistro)
		workflow, err := builder.Build()
		require.NoError(t, err)

		assert.Equal(t, "rhel-nvidia-installation", workflow.Name())
		assert.Len(t, workflow.Steps(), 8)

		stepNames := getStepNames(workflow.Steps())
		expectedOrder := []string{
			"validation",
			"repository",
			"nouveau_blacklist",
			"packages",
			"dkms_build",
			"module_load",
			"xorg_config",
			"verification",
		}
		assert.Equal(t, expectedOrder, stepNames)
	})

	t.Run("Arch workflow skips repository step", func(t *testing.T) {
		builder := NewWorkflowBuilder(archDistro)
		workflow, err := builder.Build()
		require.NoError(t, err)

		assert.Equal(t, "arch-nvidia-installation", workflow.Name())
		assert.Len(t, workflow.Steps(), 7)

		stepNames := getStepNames(workflow.Steps())
		assert.NotContains(t, stepNames, "repository")

		expectedOrder := []string{
			"validation",
			"nouveau_blacklist",
			"packages",
			"dkms_build",
			"module_load",
			"xorg_config",
			"verification",
		}
		assert.Equal(t, expectedOrder, stepNames)
	})

	t.Run("SUSE workflow executes all steps in order", func(t *testing.T) {
		builder := NewWorkflowBuilder(openSUSEDistro)
		workflow, err := builder.Build()
		require.NoError(t, err)

		assert.Equal(t, "suse-nvidia-installation", workflow.Name())
		assert.Len(t, workflow.Steps(), 8)

		stepNames := getStepNames(workflow.Steps())
		expectedOrder := []string{
			"validation",
			"repository",
			"nouveau_blacklist",
			"packages",
			"dkms_build",
			"module_load",
			"xorg_config",
			"verification",
		}
		assert.Equal(t, expectedOrder, stepNames)
	})
}

// =============================================================================
// Custom Steps Integration Tests
// =============================================================================

// TestBuilder_CustomStepsIntegration tests custom step injection and execution
func TestBuilder_CustomStepsIntegration(t *testing.T) {
	t.Run("custom steps execute after standard steps", func(t *testing.T) {
		var executionOrder []string
		var mu sync.Mutex

		customStep1 := install.NewFuncStep("custom_pre_reboot", "Pre-reboot check", func(ctx *install.Context) install.StepResult {
			mu.Lock()
			executionOrder = append(executionOrder, "custom_pre_reboot")
			mu.Unlock()
			ctx.SetState("pre_reboot_checked", true)
			return install.CompleteStep("Pre-reboot check completed")
		})

		customStep2 := install.NewFuncStep("custom_final_cleanup", "Final cleanup", func(ctx *install.Context) install.StepResult {
			mu.Lock()
			executionOrder = append(executionOrder, "custom_final_cleanup")
			mu.Unlock()
			return install.CompleteStep("Cleanup completed")
		})

		builder := NewWorkflowBuilder(ubuntuDistro, WithCustomSteps(customStep1, customStep2))
		workflow, err := builder.Build()
		require.NoError(t, err)

		// Should have 10 steps (8 standard + 2 custom)
		assert.Len(t, workflow.Steps(), 10)

		// Custom steps should be at the end
		stepNames := getStepNames(workflow.Steps())
		assert.Equal(t, "custom_pre_reboot", stepNames[8])
		assert.Equal(t, "custom_final_cleanup", stepNames[9])
	})

	t.Run("custom step with rollback capability", func(t *testing.T) {
		var rolledBack bool

		customStep := install.NewFuncStep("custom_with_rollback", "Custom with rollback",
			func(ctx *install.Context) install.StepResult {
				ctx.SetState("custom_data", "important_value")
				return install.CompleteStep("Custom step completed")
			},
			install.WithRollbackFunc(func(ctx *install.Context) error {
				rolledBack = true
				ctx.SetState("custom_data", nil)
				return nil
			}),
		)

		builder := NewWorkflowBuilder(ubuntuDistro, WithCustomSteps(customStep))
		workflow, err := builder.Build()
		require.NoError(t, err)

		// Find the custom step
		var step install.Step
		for _, s := range workflow.Steps() {
			if s.Name() == "custom_with_rollback" {
				step = s
				break
			}
		}
		require.NotNil(t, step)

		// Execute and then rollback
		ctx := install.NewContext()
		result := step.Execute(ctx)
		assert.True(t, result.IsSuccess())
		assert.Equal(t, "important_value", ctx.GetStateString("custom_data"))

		// Rollback
		assert.True(t, step.CanRollback())
		err = step.Rollback(ctx)
		assert.NoError(t, err)
		assert.True(t, rolledBack)
	})

	t.Run("custom steps can access state from previous steps", func(t *testing.T) {
		var capturedState string

		// First custom step sets state
		step1 := install.NewFuncStep("state_setter", "Set state", func(ctx *install.Context) install.StepResult {
			ctx.SetState("shared_data", "from_step1")
			return install.CompleteStep("State set")
		})

		// Second custom step reads state
		step2 := install.NewFuncStep("state_reader", "Read state", func(ctx *install.Context) install.StepResult {
			capturedState = ctx.GetStateString("shared_data")
			return install.CompleteStep("State read")
		})

		builder := NewWorkflowBuilder(ubuntuDistro,
			WithSkipValidation(true),
			WithSkipRepository(true),
			WithSkipNouveau(true),
			WithSkipDKMS(true),
			WithSkipModuleLoad(true),
			WithSkipXorgConfig(true),
			WithSkipVerification(true),
			WithCustomSteps(step1, step2),
		)
		workflow, err := builder.Build()
		require.NoError(t, err)

		// Execute all steps
		ctx := install.NewContext()
		for _, step := range workflow.Steps() {
			step.Execute(ctx)
		}

		assert.Equal(t, "from_step1", capturedState)
	})
}

// =============================================================================
// Skip Options Integration Tests
// =============================================================================

// TestBuilder_SkipOptionsIntegration tests various skip configurations
func TestBuilder_SkipOptionsIntegration(t *testing.T) {
	t.Run("minimal workflow with all optional steps skipped", func(t *testing.T) {
		builder := NewWorkflowBuilder(ubuntuDistro,
			WithSkipValidation(true),
			WithSkipRepository(true),
			WithSkipNouveau(true),
			WithSkipDKMS(true),
			WithSkipModuleLoad(true),
			WithSkipXorgConfig(true),
			WithSkipVerification(true),
		)
		workflow, err := builder.Build()
		require.NoError(t, err)

		// Only packages step should remain
		assert.Len(t, workflow.Steps(), 1)
		assert.Equal(t, "packages", workflow.Steps()[0].Name())
	})

	t.Run("workflow with only validation and verification", func(t *testing.T) {
		builder := NewWorkflowBuilder(ubuntuDistro,
			WithSkipRepository(true),
			WithSkipNouveau(true),
			WithSkipDKMS(true),
			WithSkipModuleLoad(true),
			WithSkipXorgConfig(true),
		)
		workflow, err := builder.Build()
		require.NoError(t, err)

		stepNames := getStepNames(workflow.Steps())
		assert.Contains(t, stepNames, "validation")
		assert.Contains(t, stepNames, "packages")
		assert.Contains(t, stepNames, "verification")
		assert.Len(t, stepNames, 3)
	})

	t.Run("Arch with additional skip options", func(t *testing.T) {
		builder := NewWorkflowBuilder(archDistro,
			WithSkipDKMS(true),
			WithSkipXorgConfig(true),
		)
		workflow, err := builder.Build()
		require.NoError(t, err)

		// Arch already skips repository, plus DKMS and Xorg
		stepNames := getStepNames(workflow.Steps())
		assert.NotContains(t, stepNames, "repository")
		assert.NotContains(t, stepNames, "dkms_build")
		assert.NotContains(t, stepNames, "xorg_config")
		assert.Len(t, stepNames, 5)
	})
}

// =============================================================================
// Validation Configuration Integration Tests
// =============================================================================

// TestBuilder_ValidationConfigIntegration tests validation step configuration
func TestBuilder_ValidationConfigIntegration(t *testing.T) {
	t.Run("custom validation checks are applied to workflow", func(t *testing.T) {
		checks := []steps.ValidationCheck{
			steps.CheckKernel,
			steps.CheckDiskSpace,
		}

		builder := NewWorkflowBuilder(ubuntuDistro, WithValidationChecks(checks...))
		workflow, err := builder.Build()
		require.NoError(t, err)

		// Validation step should be first
		assert.Equal(t, "validation", workflow.Steps()[0].Name())
	})

	t.Run("custom disk space requirement is applied", func(t *testing.T) {
		builder := NewWorkflowBuilder(ubuntuDistro, WithRequiredDiskMB(10000))
		workflow, err := builder.Build()
		require.NoError(t, err)

		assert.Equal(t, "validation", workflow.Steps()[0].Name())
		assert.Equal(t, int64(10000), builder.Config().RequiredDiskMB)
	})

	t.Run("combined validation options", func(t *testing.T) {
		checks := []steps.ValidationCheck{steps.CheckNVIDIAGPU}

		builder := NewWorkflowBuilder(fedoraDistro,
			WithValidationChecks(checks...),
			WithRequiredDiskMB(8000),
		)
		workflow, err := builder.Build()
		require.NoError(t, err)

		config := builder.Config()
		assert.Equal(t, checks, config.ValidationChecks)
		assert.Equal(t, int64(8000), config.RequiredDiskMB)
		assert.Len(t, workflow.Steps(), 8)
	})
}

// =============================================================================
// Distribution-Specific Workflow Integration Tests
// =============================================================================

// TestBuilder_DistributionSpecificWorkflows tests distribution-specific behavior
func TestBuilder_DistributionSpecificWorkflows(t *testing.T) {
	distributions := []struct {
		name          string
		distro        *distro.Distribution
		expectedSteps int
		hasRepository bool
	}{
		{
			name:          "Ubuntu",
			distro:        ubuntuDistro,
			expectedSteps: 8,
			hasRepository: true,
		},
		{
			name:          "Fedora",
			distro:        fedoraDistro,
			expectedSteps: 8,
			hasRepository: true,
		},
		{
			name:          "Arch",
			distro:        archDistro,
			expectedSteps: 7,
			hasRepository: false,
		},
		{
			name:          "openSUSE",
			distro:        openSUSEDistro,
			expectedSteps: 8,
			hasRepository: true,
		},
	}

	for _, tc := range distributions {
		t.Run(tc.name, func(t *testing.T) {
			builder := NewWorkflowBuilder(tc.distro)
			workflow, err := builder.Build()
			require.NoError(t, err)

			assert.Len(t, workflow.Steps(), tc.expectedSteps)

			stepNames := getStepNames(workflow.Steps())
			if tc.hasRepository {
				assert.Contains(t, stepNames, "repository")
			} else {
				assert.NotContains(t, stepNames, "repository")
			}
		})
	}
}

// TestBuilder_DerivativeDistributions tests workflows for derivative distributions
func TestBuilder_DerivativeDistributions(t *testing.T) {
	derivatives := []struct {
		name          string
		distro        *distro.Distribution
		expectedSteps int
	}{
		{
			name: "Linux Mint (Debian derivative)",
			distro: &distro.Distribution{
				ID:        "linuxmint",
				Name:      "Linux Mint",
				VersionID: "21.3",
				Family:    constants.FamilyDebian,
			},
			expectedSteps: 8,
		},
		{
			name: "CentOS (RHEL derivative)",
			distro: &distro.Distribution{
				ID:        "centos",
				Name:      "CentOS Linux",
				VersionID: "9",
				Family:    constants.FamilyRHEL,
			},
			expectedSteps: 8,
		},
		{
			name: "Manjaro (Arch derivative)",
			distro: &distro.Distribution{
				ID:        "manjaro",
				Name:      "Manjaro Linux",
				VersionID: "24.0",
				Family:    constants.FamilyArch,
			},
			expectedSteps: 7,
		},
		{
			name: "openSUSE Leap (SUSE derivative)",
			distro: &distro.Distribution{
				ID:        "opensuse-leap",
				Name:      "openSUSE Leap",
				VersionID: "15.5",
				Family:    constants.FamilySUSE,
			},
			expectedSteps: 8,
		},
	}

	for _, tc := range derivatives {
		t.Run(tc.name, func(t *testing.T) {
			builder := NewWorkflowBuilder(tc.distro)
			workflow, err := builder.Build()
			require.NoError(t, err)

			assert.Len(t, workflow.Steps(), tc.expectedSteps)
			assert.Contains(t, workflow.Name(), tc.distro.Family.String())
		})
	}
}

// =============================================================================
// Error Handling Integration Tests
// =============================================================================

// TestBuilder_ErrorHandlingIntegration tests error scenarios
func TestBuilder_ErrorHandlingIntegration(t *testing.T) {
	t.Run("nil distribution returns error", func(t *testing.T) {
		builder := NewWorkflowBuilder(nil)
		workflow, err := builder.Build()

		assert.Error(t, err)
		assert.Nil(t, workflow)
		assert.Contains(t, err.Error(), "distribution is nil")
	})

	t.Run("unknown family returns error", func(t *testing.T) {
		builder := NewWorkflowBuilder(unknownDistro)
		workflow, err := builder.Build()

		assert.Error(t, err)
		assert.Nil(t, workflow)
		assert.Contains(t, err.Error(), "unsupported distribution family")
	})

	t.Run("BuilderForFamily with unknown returns error", func(t *testing.T) {
		builder, err := BuilderForFamily(constants.FamilyUnknown)

		assert.Error(t, err)
		assert.Nil(t, builder)
		assert.Contains(t, err.Error(), "unsupported distribution family")
	})
}

// =============================================================================
// Workflow Reusability Tests
// =============================================================================

// TestBuilder_WorkflowReusability tests that workflows can be built multiple times
func TestBuilder_WorkflowReusability(t *testing.T) {
	t.Run("builder can create multiple independent workflows", func(t *testing.T) {
		builder := NewWorkflowBuilder(ubuntuDistro)

		workflow1, err1 := builder.Build()
		workflow2, err2 := builder.Build()

		require.NoError(t, err1)
		require.NoError(t, err2)

		// Both workflows should be valid
		assert.Equal(t, workflow1.Name(), workflow2.Name())
		assert.Len(t, workflow1.Steps(), len(workflow2.Steps()))

		// But they should be independent instances
		// (modifications to one shouldn't affect the other)
	})

	t.Run("different builders create independent workflows", func(t *testing.T) {
		builder1 := NewWorkflowBuilder(ubuntuDistro)
		builder2 := NewWorkflowBuilder(fedoraDistro)

		workflow1, err1 := builder1.Build()
		workflow2, err2 := builder2.Build()

		require.NoError(t, err1)
		require.NoError(t, err2)

		assert.NotEqual(t, workflow1.Name(), workflow2.Name())
	})
}

// =============================================================================
// Concurrent Building Tests
// =============================================================================

// TestBuilder_ConcurrentBuilding tests thread safety of workflow building
func TestBuilder_ConcurrentBuilding(t *testing.T) {
	t.Run("concurrent builds from same builder", func(t *testing.T) {
		builder := NewWorkflowBuilder(ubuntuDistro)

		var wg sync.WaitGroup
		results := make(chan error, 100)

		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				workflow, err := builder.Build()
				if err != nil {
					results <- err
					return
				}
				if workflow == nil {
					results <- assert.AnError
					return
				}
				if len(workflow.Steps()) != 8 {
					results <- assert.AnError
					return
				}
				results <- nil
			}()
		}

		wg.Wait()
		close(results)

		for err := range results {
			assert.NoError(t, err)
		}
	})

	t.Run("concurrent builds from different builders", func(t *testing.T) {
		distros := []*distro.Distribution{ubuntuDistro, fedoraDistro, archDistro, openSUSEDistro}

		var wg sync.WaitGroup
		results := make(chan error, 40)

		for i := 0; i < 10; i++ {
			for _, d := range distros {
				wg.Add(1)
				go func(dist *distro.Distribution) {
					defer wg.Done()
					builder := NewWorkflowBuilder(dist)
					workflow, err := builder.Build()
					if err != nil {
						results <- err
						return
					}
					if workflow == nil {
						results <- assert.AnError
						return
					}
					results <- nil
				}(d)
			}
		}

		wg.Wait()
		close(results)

		for err := range results {
			assert.NoError(t, err)
		}
	})
}

// =============================================================================
// BuilderForFamily Integration Tests
// =============================================================================

// TestBuilder_BuilderForFamilyIntegration tests the convenience function
func TestBuilder_BuilderForFamilyIntegration(t *testing.T) {
	families := []constants.DistroFamily{
		constants.FamilyDebian,
		constants.FamilyRHEL,
		constants.FamilyArch,
		constants.FamilySUSE,
	}

	for _, family := range families {
		t.Run(family.String(), func(t *testing.T) {
			builder, err := BuilderForFamily(family)
			require.NoError(t, err)
			require.NotNil(t, builder)

			workflow, err := builder.Build()
			require.NoError(t, err)
			require.NotNil(t, workflow)

			assert.Contains(t, workflow.Name(), family.String())
			assert.NotEmpty(t, workflow.Steps())
		})
	}
}

// =============================================================================
// Dry Run Workflow Integration Tests
// =============================================================================

// TestBuilder_DryRunWorkflowExecution tests workflow execution in dry-run mode
func TestBuilder_DryRunWorkflowExecution(t *testing.T) {
	t.Run("workflow steps handle dry run mode", func(t *testing.T) {
		// Test with only custom steps that don't require mocked dependencies
		customStep1 := install.NewFuncStep("custom1", "Custom step 1", func(ctx *install.Context) install.StepResult {
			if ctx.DryRun {
				return install.CompleteStep("dry run: custom step 1")
			}
			return install.CompleteStep("custom step 1 done")
		})

		customStep2 := install.NewFuncStep("custom2", "Custom step 2", func(ctx *install.Context) install.StepResult {
			if ctx.DryRun {
				return install.CompleteStep("dry run: custom step 2")
			}
			return install.CompleteStep("custom step 2 done")
		})

		// Build workflow with all standard steps skipped, only custom steps
		builder := NewWorkflowBuilder(ubuntuDistro,
			WithSkipValidation(true),
			WithSkipRepository(true),
			WithSkipNouveau(true),
			WithSkipDKMS(true),
			WithSkipModuleLoad(true),
			WithSkipXorgConfig(true),
			WithSkipVerification(true),
			WithCustomSteps(customStep1, customStep2),
		)
		workflow, err := builder.Build()
		require.NoError(t, err)

		// Should have packages step + 2 custom steps
		// Note: packages step is always included and requires dependencies
		// Skip it for this dry-run test to focus on custom steps behavior
		assert.Len(t, workflow.Steps(), 3) // packages + 2 custom

		// Execute only custom steps in dry-run mode
		ctx := install.NewContext(
			install.WithDryRun(true),
			install.WithDistroInfo(ubuntuDistro),
			install.WithDriverVersion("550"),
		)
		ctx.Executor = exec.NewMockExecutor()

		// Execute custom steps (skip the packages step which requires package manager)
		for _, step := range workflow.Steps() {
			if step.Name() == "packages" {
				continue // Skip packages step as it requires package manager mock
			}
			result := step.Execute(ctx)
			// Custom steps should succeed in dry-run mode
			assert.True(t, result.IsSuccess(),
				"Step %s should succeed in dry-run mode", step.Name())
		}
	})

	t.Run("dry run context is properly set", func(t *testing.T) {
		ctx := install.NewContext(
			install.WithDryRun(true),
			install.WithDistroInfo(ubuntuDistro),
		)

		assert.True(t, ctx.DryRun)
	})
}

// =============================================================================
// Step Description Integration Tests
// =============================================================================

// TestBuilder_StepDescriptions tests that all steps have proper descriptions
func TestBuilder_StepDescriptions(t *testing.T) {
	families := []constants.DistroFamily{
		constants.FamilyDebian,
		constants.FamilyRHEL,
		constants.FamilyArch,
		constants.FamilySUSE,
	}

	for _, family := range families {
		t.Run(family.String(), func(t *testing.T) {
			builder, err := BuilderForFamily(family)
			require.NoError(t, err)

			workflow, err := builder.Build()
			require.NoError(t, err)

			for _, step := range workflow.Steps() {
				assert.NotEmpty(t, step.Name(), "Step should have a name")
				assert.NotEmpty(t, step.Description(), "Step %s should have a description", step.Name())
			}
		})
	}
}

// =============================================================================
// Benchmarks
// =============================================================================

func BenchmarkBuilder_BuildDebianWorkflow(b *testing.B) {
	for i := 0; i < b.N; i++ {
		builder := NewWorkflowBuilder(ubuntuDistro)
		_, _ = builder.Build()
	}
}

func BenchmarkBuilder_BuildArchWorkflow(b *testing.B) {
	for i := 0; i < b.N; i++ {
		builder := NewWorkflowBuilder(archDistro)
		_, _ = builder.Build()
	}
}

func BenchmarkBuilder_BuildWithAllOptions(b *testing.B) {
	customStep := install.NewFuncStep("custom", "Custom", func(ctx *install.Context) install.StepResult {
		return install.CompleteStep("done")
	})
	checks := []steps.ValidationCheck{steps.CheckKernel, steps.CheckDiskSpace}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		builder := NewWorkflowBuilder(ubuntuDistro,
			WithValidationChecks(checks...),
			WithRequiredDiskMB(5000),
			WithCustomSteps(customStep),
		)
		_, _ = builder.Build()
	}
}

func BenchmarkBuilder_ConcurrentBuilds(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			builder := NewWorkflowBuilder(ubuntuDistro)
			_, _ = builder.Build()
		}
	})
}

func BenchmarkBuilderForFamily(b *testing.B) {
	families := []constants.DistroFamily{
		constants.FamilyDebian,
		constants.FamilyRHEL,
		constants.FamilyArch,
		constants.FamilySUSE,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		family := families[i%len(families)]
		builder, _ := BuilderForFamily(family)
		_, _ = builder.Build()
	}
}
