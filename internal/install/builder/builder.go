// Package builder provides workflow construction for NVIDIA driver installation.
// It assembles distribution-specific installation workflows by selecting and
// configuring appropriate steps based on the detected Linux distribution family.
package builder

import (
	"fmt"

	"github.com/tungetti/igor/internal/constants"
	"github.com/tungetti/igor/internal/distro"
	"github.com/tungetti/igor/internal/install"
	"github.com/tungetti/igor/internal/install/steps"
)

// BuilderConfig contains configuration options for workflow building.
type BuilderConfig struct {
	// SkipValidation skips the pre-installation validation step
	SkipValidation bool
	// SkipRepository skips repository configuration
	SkipRepository bool
	// SkipNouveau skips nouveau blacklisting
	SkipNouveau bool
	// SkipDKMS skips DKMS module building
	SkipDKMS bool
	// SkipModuleLoad skips kernel module loading
	SkipModuleLoad bool
	// SkipXorgConfig skips X.org configuration
	SkipXorgConfig bool
	// SkipVerification skips post-installation verification
	SkipVerification bool
	// CustomSteps allows injecting custom steps (for extensibility)
	CustomSteps []install.Step
	// ValidationChecks allows customizing which validation checks to run
	ValidationChecks []steps.ValidationCheck
	// RequiredDiskMB overrides default required disk space
	RequiredDiskMB int64
}

// WorkflowBuilder builds installation workflows for different distributions.
type WorkflowBuilder struct {
	config BuilderConfig
	distro *distro.Distribution
}

// WorkflowBuilderOption is a functional option for WorkflowBuilder.
type WorkflowBuilderOption func(*WorkflowBuilder)

// DefaultBuilderConfig returns the default builder configuration.
func DefaultBuilderConfig() BuilderConfig {
	return BuilderConfig{
		SkipValidation:   false,
		SkipRepository:   false,
		SkipNouveau:      false,
		SkipDKMS:         false,
		SkipModuleLoad:   false,
		SkipXorgConfig:   false,
		SkipVerification: false,
		CustomSteps:      nil,
		ValidationChecks: nil,
		RequiredDiskMB:   0, // Use default from validator
	}
}

// NewWorkflowBuilder creates a new workflow builder for the given distribution.
func NewWorkflowBuilder(dist *distro.Distribution, opts ...WorkflowBuilderOption) *WorkflowBuilder {
	b := &WorkflowBuilder{
		config: DefaultBuilderConfig(),
		distro: dist,
	}

	for _, opt := range opts {
		opt(b)
	}

	return b
}

// WithSkipValidation sets whether to skip the validation step.
func WithSkipValidation(skip bool) WorkflowBuilderOption {
	return func(b *WorkflowBuilder) {
		b.config.SkipValidation = skip
	}
}

// WithSkipRepository sets whether to skip the repository step.
func WithSkipRepository(skip bool) WorkflowBuilderOption {
	return func(b *WorkflowBuilder) {
		b.config.SkipRepository = skip
	}
}

// WithSkipNouveau sets whether to skip the nouveau blacklist step.
func WithSkipNouveau(skip bool) WorkflowBuilderOption {
	return func(b *WorkflowBuilder) {
		b.config.SkipNouveau = skip
	}
}

// WithSkipDKMS sets whether to skip the DKMS build step.
func WithSkipDKMS(skip bool) WorkflowBuilderOption {
	return func(b *WorkflowBuilder) {
		b.config.SkipDKMS = skip
	}
}

// WithSkipModuleLoad sets whether to skip the module load step.
func WithSkipModuleLoad(skip bool) WorkflowBuilderOption {
	return func(b *WorkflowBuilder) {
		b.config.SkipModuleLoad = skip
	}
}

// WithSkipXorgConfig sets whether to skip the xorg config step.
func WithSkipXorgConfig(skip bool) WorkflowBuilderOption {
	return func(b *WorkflowBuilder) {
		b.config.SkipXorgConfig = skip
	}
}

// WithSkipVerification sets whether to skip the verification step.
func WithSkipVerification(skip bool) WorkflowBuilderOption {
	return func(b *WorkflowBuilder) {
		b.config.SkipVerification = skip
	}
}

// WithCustomSteps adds custom steps to the workflow.
// Custom steps are added after the standard steps.
func WithCustomSteps(customSteps ...install.Step) WorkflowBuilderOption {
	return func(b *WorkflowBuilder) {
		b.config.CustomSteps = append(b.config.CustomSteps, customSteps...)
	}
}

// WithValidationChecks sets the validation checks to run.
// If not set, the default validation checks are used.
func WithValidationChecks(checks ...steps.ValidationCheck) WorkflowBuilderOption {
	return func(b *WorkflowBuilder) {
		b.config.ValidationChecks = append([]steps.ValidationCheck{}, checks...)
	}
}

// WithRequiredDiskMB sets the required disk space in megabytes.
// If set to 0, the default from the validator is used.
func WithRequiredDiskMB(mb int64) WorkflowBuilderOption {
	return func(b *WorkflowBuilder) {
		b.config.RequiredDiskMB = mb
	}
}

// WithBuilderConfig sets the entire builder configuration at once.
func WithBuilderConfig(config BuilderConfig) WorkflowBuilderOption {
	return func(b *WorkflowBuilder) {
		b.config = config
	}
}

// Build creates a workflow with the appropriate steps for the distribution.
// The workflow name will be formatted as "{distro-family}-nvidia-installation".
// Returns an error if the distribution is nil or unknown.
func (b *WorkflowBuilder) Build() (install.Workflow, error) {
	// Validate distribution
	if b.distro == nil {
		return nil, fmt.Errorf("distribution is nil")
	}

	// Check for unknown family
	if b.distro.Family == constants.FamilyUnknown {
		return nil, fmt.Errorf("unsupported distribution family: unknown")
	}

	// Create workflow with appropriate name
	workflowName := fmt.Sprintf("%s-nvidia-installation", b.distro.Family.String())
	workflow := install.NewWorkflow(workflowName)

	// Add steps based on distribution family
	if err := b.addSteps(workflow); err != nil {
		return nil, err
	}

	return workflow, nil
}

// addSteps adds the appropriate steps to the workflow based on distribution family.
func (b *WorkflowBuilder) addSteps(workflow *install.BaseWorkflow) error {
	// Step order (as per spec):
	// 1. ValidationStep
	// 2. RepositoryStep (skipped for Arch)
	// 3. NouveauBlacklistStep
	// 4. PackageInstallationStep
	// 5. DKMSBuildStep
	// 6. ModuleLoadStep
	// 7. XorgConfigStep
	// 8. VerificationStep

	// 1. Validation step
	if !b.config.SkipValidation {
		workflow.AddStep(b.buildValidationStep())
	}

	// 2. Repository step (skipped for Arch family)
	if !b.config.SkipRepository && !b.shouldSkipRepository() {
		workflow.AddStep(b.buildRepositoryStep())
	}

	// 3. Nouveau blacklist step
	if !b.config.SkipNouveau {
		workflow.AddStep(b.buildNouveauBlacklistStep())
	}

	// 4. Package installation step
	workflow.AddStep(b.buildPackageInstallationStep())

	// 5. DKMS build step
	if !b.config.SkipDKMS {
		workflow.AddStep(b.buildDKMSBuildStep())
	}

	// 6. Module load step
	if !b.config.SkipModuleLoad {
		workflow.AddStep(b.buildModuleLoadStep())
	}

	// 7. X.org config step
	if !b.config.SkipXorgConfig {
		workflow.AddStep(b.buildXorgConfigStep())
	}

	// 8. Verification step
	if !b.config.SkipVerification {
		workflow.AddStep(b.buildVerificationStep())
	}

	// Add custom steps
	for _, step := range b.config.CustomSteps {
		workflow.AddStep(step)
	}

	return nil
}

// shouldSkipRepository returns true if the repository step should be skipped
// for the current distribution family.
func (b *WorkflowBuilder) shouldSkipRepository() bool {
	// Arch family doesn't need external repositories - NVIDIA packages are in official repos
	return b.distro.Family == constants.FamilyArch
}

// buildValidationStep creates the validation step with appropriate options.
func (b *WorkflowBuilder) buildValidationStep() install.Step {
	var opts []steps.ValidationStepOption

	// Add custom validation checks if specified
	if len(b.config.ValidationChecks) > 0 {
		opts = append(opts, steps.WithChecks(b.config.ValidationChecks...))
	}

	// Add custom required disk space if specified
	if b.config.RequiredDiskMB > 0 {
		opts = append(opts, steps.WithRequiredDiskMB(b.config.RequiredDiskMB))
	}

	return steps.NewValidationStep(opts...)
}

// buildRepositoryStep creates the repository configuration step.
func (b *WorkflowBuilder) buildRepositoryStep() install.Step {
	return steps.NewRepositoryStep()
}

// buildNouveauBlacklistStep creates the nouveau blacklist step.
func (b *WorkflowBuilder) buildNouveauBlacklistStep() install.Step {
	return steps.NewNouveauBlacklistStep()
}

// buildPackageInstallationStep creates the package installation step.
func (b *WorkflowBuilder) buildPackageInstallationStep() install.Step {
	return steps.NewPackageInstallationStep()
}

// buildDKMSBuildStep creates the DKMS build step.
func (b *WorkflowBuilder) buildDKMSBuildStep() install.Step {
	return steps.NewDKMSBuildStep()
}

// buildModuleLoadStep creates the module load step.
func (b *WorkflowBuilder) buildModuleLoadStep() install.Step {
	return steps.NewModuleLoadStep()
}

// buildXorgConfigStep creates the X.org configuration step.
func (b *WorkflowBuilder) buildXorgConfigStep() install.Step {
	return steps.NewXorgConfigStep()
}

// buildVerificationStep creates the verification step.
func (b *WorkflowBuilder) buildVerificationStep() install.Step {
	return steps.NewVerificationStep()
}

// BuilderForFamily is a convenience function that creates a workflow builder
// for the given distribution family with default configuration.
func BuilderForFamily(family constants.DistroFamily) (*WorkflowBuilder, error) {
	if family == constants.FamilyUnknown {
		return nil, fmt.Errorf("unsupported distribution family: unknown")
	}

	// Create a minimal distribution with the specified family
	dist := &distro.Distribution{
		Family: family,
		ID:     family.String(),
		Name:   family.String(),
	}

	return NewWorkflowBuilder(dist), nil
}

// Config returns a copy of the current builder configuration.
func (b *WorkflowBuilder) Config() BuilderConfig {
	// Return a copy to prevent mutation
	config := b.config
	if b.config.CustomSteps != nil {
		config.CustomSteps = append([]install.Step{}, b.config.CustomSteps...)
	}
	if b.config.ValidationChecks != nil {
		config.ValidationChecks = append([]steps.ValidationCheck{}, b.config.ValidationChecks...)
	}
	return config
}

// Distribution returns the distribution this builder is configured for.
func (b *WorkflowBuilder) Distribution() *distro.Distribution {
	return b.distro
}

// Ensure BaseWorkflow implements Workflow interface (compile-time check).
var _ install.Workflow = (*install.BaseWorkflow)(nil)
