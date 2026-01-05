package builder

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tungetti/igor/internal/constants"
	"github.com/tungetti/igor/internal/distro"
	"github.com/tungetti/igor/internal/install"
	"github.com/tungetti/igor/internal/install/steps"
)

// Test distributions for different families
var (
	ubuntuDistro = &distro.Distribution{
		ID:        "ubuntu",
		Name:      "Ubuntu",
		VersionID: "24.04",
		Family:    constants.FamilyDebian,
	}

	fedoraDistro = &distro.Distribution{
		ID:        "fedora",
		Name:      "Fedora",
		VersionID: "40",
		Family:    constants.FamilyRHEL,
	}

	archDistro = &distro.Distribution{
		ID:        "arch",
		Name:      "Arch Linux",
		VersionID: "",
		Family:    constants.FamilyArch,
	}

	openSUSEDistro = &distro.Distribution{
		ID:        "opensuse-tumbleweed",
		Name:      "openSUSE Tumbleweed",
		VersionID: "",
		Family:    constants.FamilySUSE,
	}

	unknownDistro = &distro.Distribution{
		ID:        "unknown",
		Name:      "Unknown Distro",
		VersionID: "1.0",
		Family:    constants.FamilyUnknown,
	}
)

// TestNewWorkflowBuilder tests the NewWorkflowBuilder factory function.
func TestNewWorkflowBuilder(t *testing.T) {
	t.Run("creates builder with distribution", func(t *testing.T) {
		builder := NewWorkflowBuilder(ubuntuDistro)

		assert.NotNil(t, builder)
		assert.Equal(t, ubuntuDistro, builder.Distribution())
	})

	t.Run("creates builder with nil distribution", func(t *testing.T) {
		builder := NewWorkflowBuilder(nil)

		assert.NotNil(t, builder)
		assert.Nil(t, builder.Distribution())
	})

	t.Run("creates builder with default config", func(t *testing.T) {
		builder := NewWorkflowBuilder(ubuntuDistro)

		config := builder.Config()
		assert.False(t, config.SkipValidation)
		assert.False(t, config.SkipRepository)
		assert.False(t, config.SkipNouveau)
		assert.False(t, config.SkipDKMS)
		assert.False(t, config.SkipModuleLoad)
		assert.False(t, config.SkipXorgConfig)
		assert.False(t, config.SkipVerification)
		assert.Nil(t, config.CustomSteps)
		assert.Nil(t, config.ValidationChecks)
		assert.Equal(t, int64(0), config.RequiredDiskMB)
	})

	t.Run("applies functional options", func(t *testing.T) {
		builder := NewWorkflowBuilder(ubuntuDistro,
			WithSkipValidation(true),
			WithSkipRepository(true),
		)

		config := builder.Config()
		assert.True(t, config.SkipValidation)
		assert.True(t, config.SkipRepository)
	})
}

// TestDefaultBuilderConfig tests the DefaultBuilderConfig function.
func TestDefaultBuilderConfig(t *testing.T) {
	config := DefaultBuilderConfig()

	assert.False(t, config.SkipValidation)
	assert.False(t, config.SkipRepository)
	assert.False(t, config.SkipNouveau)
	assert.False(t, config.SkipDKMS)
	assert.False(t, config.SkipModuleLoad)
	assert.False(t, config.SkipXorgConfig)
	assert.False(t, config.SkipVerification)
	assert.Nil(t, config.CustomSteps)
	assert.Nil(t, config.ValidationChecks)
	assert.Equal(t, int64(0), config.RequiredDiskMB)
}

// TestWorkflowBuilder_FunctionalOptions tests all functional options.
func TestWorkflowBuilder_FunctionalOptions(t *testing.T) {
	t.Run("WithSkipValidation", func(t *testing.T) {
		builder := NewWorkflowBuilder(ubuntuDistro, WithSkipValidation(true))
		assert.True(t, builder.Config().SkipValidation)

		builder = NewWorkflowBuilder(ubuntuDistro, WithSkipValidation(false))
		assert.False(t, builder.Config().SkipValidation)
	})

	t.Run("WithSkipRepository", func(t *testing.T) {
		builder := NewWorkflowBuilder(ubuntuDistro, WithSkipRepository(true))
		assert.True(t, builder.Config().SkipRepository)
	})

	t.Run("WithSkipNouveau", func(t *testing.T) {
		builder := NewWorkflowBuilder(ubuntuDistro, WithSkipNouveau(true))
		assert.True(t, builder.Config().SkipNouveau)
	})

	t.Run("WithSkipDKMS", func(t *testing.T) {
		builder := NewWorkflowBuilder(ubuntuDistro, WithSkipDKMS(true))
		assert.True(t, builder.Config().SkipDKMS)
	})

	t.Run("WithSkipModuleLoad", func(t *testing.T) {
		builder := NewWorkflowBuilder(ubuntuDistro, WithSkipModuleLoad(true))
		assert.True(t, builder.Config().SkipModuleLoad)
	})

	t.Run("WithSkipXorgConfig", func(t *testing.T) {
		builder := NewWorkflowBuilder(ubuntuDistro, WithSkipXorgConfig(true))
		assert.True(t, builder.Config().SkipXorgConfig)
	})

	t.Run("WithSkipVerification", func(t *testing.T) {
		builder := NewWorkflowBuilder(ubuntuDistro, WithSkipVerification(true))
		assert.True(t, builder.Config().SkipVerification)
	})

	t.Run("WithCustomSteps", func(t *testing.T) {
		customStep := install.NewFuncStep("custom", "Custom step", func(ctx *install.Context) install.StepResult {
			return install.CompleteStep("done")
		})

		builder := NewWorkflowBuilder(ubuntuDistro, WithCustomSteps(customStep))
		config := builder.Config()
		require.Len(t, config.CustomSteps, 1)
		assert.Equal(t, "custom", config.CustomSteps[0].Name())
	})

	t.Run("WithValidationChecks", func(t *testing.T) {
		checks := []steps.ValidationCheck{steps.CheckKernel, steps.CheckDiskSpace}
		builder := NewWorkflowBuilder(ubuntuDistro, WithValidationChecks(checks...))
		config := builder.Config()
		assert.Equal(t, checks, config.ValidationChecks)
	})

	t.Run("WithRequiredDiskMB", func(t *testing.T) {
		builder := NewWorkflowBuilder(ubuntuDistro, WithRequiredDiskMB(5000))
		assert.Equal(t, int64(5000), builder.Config().RequiredDiskMB)
	})

	t.Run("WithBuilderConfig", func(t *testing.T) {
		config := BuilderConfig{
			SkipValidation:   true,
			SkipRepository:   true,
			SkipNouveau:      true,
			SkipDKMS:         true,
			SkipModuleLoad:   true,
			SkipXorgConfig:   true,
			SkipVerification: true,
			RequiredDiskMB:   10000,
		}
		builder := NewWorkflowBuilder(ubuntuDistro, WithBuilderConfig(config))
		resultConfig := builder.Config()

		assert.True(t, resultConfig.SkipValidation)
		assert.True(t, resultConfig.SkipRepository)
		assert.True(t, resultConfig.SkipNouveau)
		assert.True(t, resultConfig.SkipDKMS)
		assert.True(t, resultConfig.SkipModuleLoad)
		assert.True(t, resultConfig.SkipXorgConfig)
		assert.True(t, resultConfig.SkipVerification)
		assert.Equal(t, int64(10000), resultConfig.RequiredDiskMB)
	})

	t.Run("multiple options applied in order", func(t *testing.T) {
		builder := NewWorkflowBuilder(ubuntuDistro,
			WithSkipValidation(true),
			WithSkipValidation(false), // Override the first one
		)
		assert.False(t, builder.Config().SkipValidation)
	})
}

// TestWorkflowBuilder_Build tests the Build method for different distributions.
func TestWorkflowBuilder_Build(t *testing.T) {
	t.Run("builds workflow for Debian family", func(t *testing.T) {
		builder := NewWorkflowBuilder(ubuntuDistro)
		workflow, err := builder.Build()

		require.NoError(t, err)
		require.NotNil(t, workflow)

		assert.Equal(t, "debian-nvidia-installation", workflow.Name())

		// Debian should have all 8 steps
		steps := workflow.Steps()
		assert.Len(t, steps, 8)

		// Verify step order
		stepNames := getStepNames(steps)
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

	t.Run("builds workflow for RHEL family", func(t *testing.T) {
		builder := NewWorkflowBuilder(fedoraDistro)
		workflow, err := builder.Build()

		require.NoError(t, err)
		require.NotNil(t, workflow)

		assert.Equal(t, "rhel-nvidia-installation", workflow.Name())

		// RHEL should have all 8 steps
		steps := workflow.Steps()
		assert.Len(t, steps, 8)

		// Verify step order
		stepNames := getStepNames(steps)
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

	t.Run("builds workflow for Arch family - skips repository", func(t *testing.T) {
		builder := NewWorkflowBuilder(archDistro)
		workflow, err := builder.Build()

		require.NoError(t, err)
		require.NotNil(t, workflow)

		assert.Equal(t, "arch-nvidia-installation", workflow.Name())

		// Arch should have 7 steps (no repository step)
		steps := workflow.Steps()
		assert.Len(t, steps, 7)

		// Verify repository step is NOT present
		stepNames := getStepNames(steps)
		assert.NotContains(t, stepNames, "repository")

		// Verify other steps are present in order
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

	t.Run("builds workflow for SUSE family", func(t *testing.T) {
		builder := NewWorkflowBuilder(openSUSEDistro)
		workflow, err := builder.Build()

		require.NoError(t, err)
		require.NotNil(t, workflow)

		assert.Equal(t, "suse-nvidia-installation", workflow.Name())

		// SUSE should have all 8 steps
		steps := workflow.Steps()
		assert.Len(t, steps, 8)
	})

	t.Run("returns error for nil distribution", func(t *testing.T) {
		builder := NewWorkflowBuilder(nil)
		workflow, err := builder.Build()

		assert.Error(t, err)
		assert.Nil(t, workflow)
		assert.Contains(t, err.Error(), "distribution is nil")
	})

	t.Run("returns error for unknown family", func(t *testing.T) {
		builder := NewWorkflowBuilder(unknownDistro)
		workflow, err := builder.Build()

		assert.Error(t, err)
		assert.Nil(t, workflow)
		assert.Contains(t, err.Error(), "unsupported distribution family: unknown")
	})
}

// TestWorkflowBuilder_Build_SkipOptions tests that skip options work correctly.
func TestWorkflowBuilder_Build_SkipOptions(t *testing.T) {
	t.Run("skip validation", func(t *testing.T) {
		builder := NewWorkflowBuilder(ubuntuDistro, WithSkipValidation(true))
		workflow, err := builder.Build()

		require.NoError(t, err)

		stepNames := getStepNames(workflow.Steps())
		assert.NotContains(t, stepNames, "validation")
		assert.Len(t, stepNames, 7) // 8 - 1
	})

	t.Run("skip repository", func(t *testing.T) {
		builder := NewWorkflowBuilder(ubuntuDistro, WithSkipRepository(true))
		workflow, err := builder.Build()

		require.NoError(t, err)

		stepNames := getStepNames(workflow.Steps())
		assert.NotContains(t, stepNames, "repository")
		assert.Len(t, stepNames, 7) // 8 - 1
	})

	t.Run("skip nouveau", func(t *testing.T) {
		builder := NewWorkflowBuilder(ubuntuDistro, WithSkipNouveau(true))
		workflow, err := builder.Build()

		require.NoError(t, err)

		stepNames := getStepNames(workflow.Steps())
		assert.NotContains(t, stepNames, "nouveau_blacklist")
		assert.Len(t, stepNames, 7) // 8 - 1
	})

	t.Run("skip DKMS", func(t *testing.T) {
		builder := NewWorkflowBuilder(ubuntuDistro, WithSkipDKMS(true))
		workflow, err := builder.Build()

		require.NoError(t, err)

		stepNames := getStepNames(workflow.Steps())
		assert.NotContains(t, stepNames, "dkms_build")
		assert.Len(t, stepNames, 7) // 8 - 1
	})

	t.Run("skip module load", func(t *testing.T) {
		builder := NewWorkflowBuilder(ubuntuDistro, WithSkipModuleLoad(true))
		workflow, err := builder.Build()

		require.NoError(t, err)

		stepNames := getStepNames(workflow.Steps())
		assert.NotContains(t, stepNames, "module_load")
		assert.Len(t, stepNames, 7) // 8 - 1
	})

	t.Run("skip xorg config", func(t *testing.T) {
		builder := NewWorkflowBuilder(ubuntuDistro, WithSkipXorgConfig(true))
		workflow, err := builder.Build()

		require.NoError(t, err)

		stepNames := getStepNames(workflow.Steps())
		assert.NotContains(t, stepNames, "xorg_config")
		assert.Len(t, stepNames, 7) // 8 - 1
	})

	t.Run("skip verification", func(t *testing.T) {
		builder := NewWorkflowBuilder(ubuntuDistro, WithSkipVerification(true))
		workflow, err := builder.Build()

		require.NoError(t, err)

		stepNames := getStepNames(workflow.Steps())
		assert.NotContains(t, stepNames, "verification")
		assert.Len(t, stepNames, 7) // 8 - 1
	})

	t.Run("skip all optional steps", func(t *testing.T) {
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

		// Only package installation should remain
		steps := workflow.Steps()
		assert.Len(t, steps, 1)
		assert.Equal(t, "packages", steps[0].Name())
	})

	t.Run("skip repository combined with Arch (no double skip)", func(t *testing.T) {
		// For Arch, repository is already skipped by default
		builder := NewWorkflowBuilder(archDistro, WithSkipRepository(true))
		workflow, err := builder.Build()

		require.NoError(t, err)

		stepNames := getStepNames(workflow.Steps())
		assert.NotContains(t, stepNames, "repository")
		assert.Len(t, stepNames, 7) // Same as default Arch
	})
}

// TestWorkflowBuilder_Build_CustomSteps tests custom step injection.
func TestWorkflowBuilder_Build_CustomSteps(t *testing.T) {
	t.Run("adds single custom step", func(t *testing.T) {
		customStep := install.NewFuncStep("custom1", "Custom step 1", func(ctx *install.Context) install.StepResult {
			return install.CompleteStep("done")
		})

		builder := NewWorkflowBuilder(ubuntuDistro, WithCustomSteps(customStep))
		workflow, err := builder.Build()

		require.NoError(t, err)

		steps := workflow.Steps()
		assert.Len(t, steps, 9) // 8 standard + 1 custom
		assert.Equal(t, "custom1", steps[8].Name())
	})

	t.Run("adds multiple custom steps", func(t *testing.T) {
		customStep1 := install.NewFuncStep("custom1", "Custom 1", func(ctx *install.Context) install.StepResult {
			return install.CompleteStep("done")
		})
		customStep2 := install.NewFuncStep("custom2", "Custom 2", func(ctx *install.Context) install.StepResult {
			return install.CompleteStep("done")
		})

		builder := NewWorkflowBuilder(ubuntuDistro, WithCustomSteps(customStep1, customStep2))
		workflow, err := builder.Build()

		require.NoError(t, err)

		steps := workflow.Steps()
		assert.Len(t, steps, 10) // 8 standard + 2 custom
		assert.Equal(t, "custom1", steps[8].Name())
		assert.Equal(t, "custom2", steps[9].Name())
	})

	t.Run("custom steps added after standard steps", func(t *testing.T) {
		customStep := install.NewFuncStep("custom", "Custom step", func(ctx *install.Context) install.StepResult {
			return install.CompleteStep("done")
		})

		builder := NewWorkflowBuilder(ubuntuDistro, WithCustomSteps(customStep))
		workflow, err := builder.Build()

		require.NoError(t, err)

		steps := workflow.Steps()
		// Last standard step should be verification
		assert.Equal(t, "verification", steps[7].Name())
		// Custom step should be after verification
		assert.Equal(t, "custom", steps[8].Name())
	})
}

// TestBuilderForFamily tests the convenience function.
func TestBuilderForFamily(t *testing.T) {
	t.Run("creates builder for Debian family", func(t *testing.T) {
		builder, err := BuilderForFamily(constants.FamilyDebian)

		require.NoError(t, err)
		require.NotNil(t, builder)

		assert.Equal(t, constants.FamilyDebian, builder.Distribution().Family)
		assert.Equal(t, "debian", builder.Distribution().ID)
	})

	t.Run("creates builder for RHEL family", func(t *testing.T) {
		builder, err := BuilderForFamily(constants.FamilyRHEL)

		require.NoError(t, err)
		require.NotNil(t, builder)

		assert.Equal(t, constants.FamilyRHEL, builder.Distribution().Family)
	})

	t.Run("creates builder for Arch family", func(t *testing.T) {
		builder, err := BuilderForFamily(constants.FamilyArch)

		require.NoError(t, err)
		require.NotNil(t, builder)

		assert.Equal(t, constants.FamilyArch, builder.Distribution().Family)
	})

	t.Run("creates builder for SUSE family", func(t *testing.T) {
		builder, err := BuilderForFamily(constants.FamilySUSE)

		require.NoError(t, err)
		require.NotNil(t, builder)

		assert.Equal(t, constants.FamilySUSE, builder.Distribution().Family)
	})

	t.Run("returns error for unknown family", func(t *testing.T) {
		builder, err := BuilderForFamily(constants.FamilyUnknown)

		assert.Error(t, err)
		assert.Nil(t, builder)
		assert.Contains(t, err.Error(), "unsupported distribution family: unknown")
	})

	t.Run("can build workflow from convenience function", func(t *testing.T) {
		builder, err := BuilderForFamily(constants.FamilyDebian)
		require.NoError(t, err)

		workflow, err := builder.Build()
		require.NoError(t, err)
		assert.Equal(t, "debian-nvidia-installation", workflow.Name())
	})
}

// TestWorkflowBuilder_WorkflowNameFormat tests the workflow naming convention.
func TestWorkflowBuilder_WorkflowNameFormat(t *testing.T) {
	testCases := []struct {
		name         string
		distro       *distro.Distribution
		expectedName string
	}{
		{
			name:         "Debian family workflow name",
			distro:       ubuntuDistro,
			expectedName: "debian-nvidia-installation",
		},
		{
			name:         "RHEL family workflow name",
			distro:       fedoraDistro,
			expectedName: "rhel-nvidia-installation",
		},
		{
			name:         "Arch family workflow name",
			distro:       archDistro,
			expectedName: "arch-nvidia-installation",
		},
		{
			name:         "SUSE family workflow name",
			distro:       openSUSEDistro,
			expectedName: "suse-nvidia-installation",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			builder := NewWorkflowBuilder(tc.distro)
			workflow, err := builder.Build()

			require.NoError(t, err)
			assert.Equal(t, tc.expectedName, workflow.Name())
		})
	}
}

// TestWorkflowBuilder_StepCount tests step counts for different families.
func TestWorkflowBuilder_StepCount(t *testing.T) {
	testCases := []struct {
		name          string
		distro        *distro.Distribution
		expectedCount int
	}{
		{
			name:          "Debian has 8 steps",
			distro:        ubuntuDistro,
			expectedCount: 8,
		},
		{
			name:          "RHEL has 8 steps",
			distro:        fedoraDistro,
			expectedCount: 8,
		},
		{
			name:          "Arch has 7 steps (no repository)",
			distro:        archDistro,
			expectedCount: 7,
		},
		{
			name:          "SUSE has 8 steps",
			distro:        openSUSEDistro,
			expectedCount: 8,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			builder := NewWorkflowBuilder(tc.distro)
			workflow, err := builder.Build()

			require.NoError(t, err)
			assert.Len(t, workflow.Steps(), tc.expectedCount)
		})
	}
}

// TestWorkflowBuilder_Config tests the Config method.
func TestWorkflowBuilder_Config(t *testing.T) {
	t.Run("returns copy of config", func(t *testing.T) {
		customStep := install.NewFuncStep("custom", "Custom", func(ctx *install.Context) install.StepResult {
			return install.CompleteStep("done")
		})
		checks := []steps.ValidationCheck{steps.CheckKernel}

		builder := NewWorkflowBuilder(ubuntuDistro,
			WithCustomSteps(customStep),
			WithValidationChecks(checks...),
		)

		config1 := builder.Config()
		config2 := builder.Config()

		// Should be equal but different instances
		assert.Equal(t, config1.SkipValidation, config2.SkipValidation)
		assert.Len(t, config1.CustomSteps, 1)
		assert.Len(t, config2.CustomSteps, 1)

		// Modifying the returned config should not affect the builder
		config1.SkipValidation = true
		assert.False(t, builder.Config().SkipValidation)
	})
}

// TestWorkflowBuilder_Distribution tests the Distribution method.
func TestWorkflowBuilder_Distribution(t *testing.T) {
	t.Run("returns the distribution", func(t *testing.T) {
		builder := NewWorkflowBuilder(ubuntuDistro)

		dist := builder.Distribution()
		assert.Equal(t, ubuntuDistro, dist)
		assert.Equal(t, "ubuntu", dist.ID)
		assert.Equal(t, constants.FamilyDebian, dist.Family)
	})

	t.Run("returns nil for nil distribution", func(t *testing.T) {
		builder := NewWorkflowBuilder(nil)
		assert.Nil(t, builder.Distribution())
	})
}

// TestWorkflowBuilder_ArchSkipsRepository specifically tests Arch repository handling.
func TestWorkflowBuilder_ArchSkipsRepository(t *testing.T) {
	t.Run("Arch family automatically skips repository step", func(t *testing.T) {
		builder := NewWorkflowBuilder(archDistro)
		workflow, err := builder.Build()

		require.NoError(t, err)

		// Verify no repository step
		for _, step := range workflow.Steps() {
			assert.NotEqual(t, "repository", step.Name(),
				"Arch workflow should not contain repository step")
		}
	})

	t.Run("Arch with explicit WithSkipRepository(false) still skips", func(t *testing.T) {
		// Even if user explicitly sets SkipRepository to false,
		// Arch should still skip (architecture decision)
		builder := NewWorkflowBuilder(archDistro, WithSkipRepository(false))
		workflow, err := builder.Build()

		require.NoError(t, err)

		stepNames := getStepNames(workflow.Steps())
		assert.NotContains(t, stepNames, "repository",
			"Arch should skip repository regardless of option")
	})

	t.Run("Manjaro (Arch derivative) also skips repository", func(t *testing.T) {
		manjaroDistro := &distro.Distribution{
			ID:        "manjaro",
			Name:      "Manjaro Linux",
			VersionID: "",
			Family:    constants.FamilyArch,
		}

		builder := NewWorkflowBuilder(manjaroDistro)
		workflow, err := builder.Build()

		require.NoError(t, err)

		stepNames := getStepNames(workflow.Steps())
		assert.NotContains(t, stepNames, "repository")
	})
}

// TestWorkflowBuilder_ValidationCheckConfiguration tests validation check customization.
func TestWorkflowBuilder_ValidationCheckConfiguration(t *testing.T) {
	t.Run("uses custom validation checks when specified", func(t *testing.T) {
		checks := []steps.ValidationCheck{
			steps.CheckKernel,
			steps.CheckDiskSpace,
		}

		builder := NewWorkflowBuilder(ubuntuDistro, WithValidationChecks(checks...))
		config := builder.Config()

		assert.Equal(t, checks, config.ValidationChecks)
	})

	t.Run("uses default validation checks when not specified", func(t *testing.T) {
		builder := NewWorkflowBuilder(ubuntuDistro)
		config := builder.Config()

		assert.Nil(t, config.ValidationChecks) // Nil means use defaults
	})

	t.Run("builds workflow with custom validation checks", func(t *testing.T) {
		checks := []steps.ValidationCheck{
			steps.CheckKernel,
			steps.CheckDiskSpace,
		}

		builder := NewWorkflowBuilder(ubuntuDistro, WithValidationChecks(checks...))
		workflow, err := builder.Build()

		require.NoError(t, err)
		assert.NotNil(t, workflow)

		// Verify validation step is included
		stepNames := getStepNames(workflow.Steps())
		assert.Contains(t, stepNames, "validation")
	})
}

// TestWorkflowBuilder_RequiredDiskMBConfiguration tests disk space requirement customization.
func TestWorkflowBuilder_RequiredDiskMBConfiguration(t *testing.T) {
	t.Run("uses custom required disk space when specified", func(t *testing.T) {
		builder := NewWorkflowBuilder(ubuntuDistro, WithRequiredDiskMB(8000))
		config := builder.Config()

		assert.Equal(t, int64(8000), config.RequiredDiskMB)
	})

	t.Run("uses zero (default) when not specified", func(t *testing.T) {
		builder := NewWorkflowBuilder(ubuntuDistro)
		config := builder.Config()

		assert.Equal(t, int64(0), config.RequiredDiskMB)
	})

	t.Run("builds workflow with custom required disk space", func(t *testing.T) {
		builder := NewWorkflowBuilder(ubuntuDistro, WithRequiredDiskMB(10000))
		workflow, err := builder.Build()

		require.NoError(t, err)
		assert.NotNil(t, workflow)

		// Verify validation step is included
		stepNames := getStepNames(workflow.Steps())
		assert.Contains(t, stepNames, "validation")
	})

	t.Run("builds workflow with both validation checks and required disk space", func(t *testing.T) {
		checks := []steps.ValidationCheck{steps.CheckKernel}
		builder := NewWorkflowBuilder(ubuntuDistro,
			WithValidationChecks(checks...),
			WithRequiredDiskMB(5000),
		)
		workflow, err := builder.Build()

		require.NoError(t, err)
		assert.NotNil(t, workflow)

		// Config should reflect both settings
		config := builder.Config()
		assert.Equal(t, checks, config.ValidationChecks)
		assert.Equal(t, int64(5000), config.RequiredDiskMB)

		// Validation step should be first step
		steps := workflow.Steps()
		assert.Equal(t, "validation", steps[0].Name())
	})
}

// TestWorkflowBuilder_ConcurrentBuilds tests that building is safe for concurrent use.
func TestWorkflowBuilder_ConcurrentBuilds(t *testing.T) {
	builder := NewWorkflowBuilder(ubuntuDistro)

	// Build workflow multiple times concurrently
	done := make(chan struct{})
	for i := 0; i < 10; i++ {
		go func() {
			defer func() { done <- struct{}{} }()
			workflow, err := builder.Build()
			assert.NoError(t, err)
			assert.NotNil(t, workflow)
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}

// TestWorkflowBuilder_MultipleBuilds tests that builder can be reused.
func TestWorkflowBuilder_MultipleBuilds(t *testing.T) {
	builder := NewWorkflowBuilder(ubuntuDistro)

	// Build multiple times
	workflow1, err1 := builder.Build()
	workflow2, err2 := builder.Build()

	require.NoError(t, err1)
	require.NoError(t, err2)

	// Both workflows should be valid and independent
	assert.Equal(t, workflow1.Name(), workflow2.Name())
	assert.Len(t, workflow1.Steps(), len(workflow2.Steps()))
}

// TestWorkflowBuilder_WorkflowIsExecutable tests that built workflows have proper structure.
func TestWorkflowBuilder_WorkflowIsExecutable(t *testing.T) {
	builder := NewWorkflowBuilder(ubuntuDistro)
	workflow, err := builder.Build()

	require.NoError(t, err)

	// Verify workflow has valid structure
	assert.NotEmpty(t, workflow.Name())
	assert.NotEmpty(t, workflow.Steps())

	// Each step should have a name and description
	for _, step := range workflow.Steps() {
		assert.NotEmpty(t, step.Name(), "Step should have a name")
		assert.NotEmpty(t, step.Description(), "Step should have a description")
	}
}

// Helper function to extract step names from a slice of steps.
func getStepNames(steps []install.Step) []string {
	names := make([]string, len(steps))
	for i, step := range steps {
		names[i] = step.Name()
	}
	return names
}

// TestWorkflowBuilder_AllFamilies tests build for all supported families.
func TestWorkflowBuilder_AllFamilies(t *testing.T) {
	families := []constants.DistroFamily{
		constants.FamilyDebian,
		constants.FamilyRHEL,
		constants.FamilyArch,
		constants.FamilySUSE,
	}

	for _, family := range families {
		t.Run(family.String(), func(t *testing.T) {
			dist := &distro.Distribution{
				ID:     family.String(),
				Name:   family.String(),
				Family: family,
			}

			builder := NewWorkflowBuilder(dist)
			workflow, err := builder.Build()

			require.NoError(t, err)
			require.NotNil(t, workflow)
			assert.Contains(t, workflow.Name(), family.String())
			assert.NotEmpty(t, workflow.Steps())
		})
	}
}

// BenchmarkWorkflowBuilder_Build benchmarks the build process.
func BenchmarkWorkflowBuilder_Build(b *testing.B) {
	builder := NewWorkflowBuilder(ubuntuDistro)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = builder.Build()
	}
}

// BenchmarkNewWorkflowBuilder benchmarks builder creation.
func BenchmarkNewWorkflowBuilder(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = NewWorkflowBuilder(ubuntuDistro,
			WithSkipValidation(true),
			WithRequiredDiskMB(5000),
		)
	}
}
