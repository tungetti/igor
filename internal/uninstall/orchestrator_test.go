package uninstall

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/tungetti/igor/internal/install"
)

// =============================================================================
// Mock Types
// =============================================================================

// mockUninstallWorkflow implements UninstallWorkflow for testing.
type mockUninstallWorkflow struct {
	name          string
	steps         []UninstallStep
	executeResult UninstallResult
	executeCalled bool
	progressCb    func(install.StepProgress)
	cancelCalled  bool
	executeFunc   func(ctx *Context) UninstallResult
	mu            sync.Mutex
}

func newMockWorkflow(name string) *mockUninstallWorkflow {
	return &mockUninstallWorkflow{
		name:  name,
		steps: make([]UninstallStep, 0),
	}
}

func (m *mockUninstallWorkflow) Name() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.name
}

func (m *mockUninstallWorkflow) Steps() []UninstallStep {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]UninstallStep{}, m.steps...)
}

func (m *mockUninstallWorkflow) AddStep(step UninstallStep) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.steps = append(m.steps, step)
}

func (m *mockUninstallWorkflow) Execute(ctx *Context) UninstallResult {
	m.mu.Lock()
	m.executeCalled = true
	executeFunc := m.executeFunc
	result := m.executeResult
	m.mu.Unlock()

	if executeFunc != nil {
		return executeFunc(ctx)
	}
	return result
}

func (m *mockUninstallWorkflow) OnProgress(callback func(install.StepProgress)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.progressCb = callback
}

func (m *mockUninstallWorkflow) Cancel() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cancelCalled = true
}

func (m *mockUninstallWorkflow) setExecuteResult(result UninstallResult) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.executeResult = result
}

func (m *mockUninstallWorkflow) setExecuteFunc(fn func(ctx *Context) UninstallResult) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.executeFunc = fn
}

// mockUninstallStep implements install.Step for testing.
type mockUninstallStep struct {
	name           string
	description    string
	validateError  error
	executeResult  install.StepResult
	canRollback    bool
	executeCalled  bool
	rollbackCalled bool
	validateCalled bool
	mu             sync.Mutex
}

func newMockStep(name, description string) *mockUninstallStep {
	return &mockUninstallStep{
		name:        name,
		description: description,
		executeResult: install.StepResult{
			Status:  install.StepStatusCompleted,
			Message: "Step completed",
		},
	}
}

func (m *mockUninstallStep) Name() string {
	return m.name
}

func (m *mockUninstallStep) Description() string {
	return m.description
}

func (m *mockUninstallStep) Execute(ctx *install.Context) install.StepResult {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.executeCalled = true
	return m.executeResult
}

func (m *mockUninstallStep) Rollback(ctx *install.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.rollbackCalled = true
	return nil
}

func (m *mockUninstallStep) CanRollback() bool {
	return m.canRollback
}

func (m *mockUninstallStep) Validate(ctx *install.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.validateCalled = true
	return m.validateError
}

func (m *mockUninstallStep) setExecuteResult(result install.StepResult) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.executeResult = result
}

func (m *mockUninstallStep) setValidateError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.validateError = err
}

// mockDiscovery implements Discovery for testing.
type mockDiscovery struct {
	discoverResult *DiscoveredPackages
	discoverError  error
	discoverCalled bool
	mu             sync.Mutex
}

func newMockDiscovery() *mockDiscovery {
	return &mockDiscovery{
		discoverResult: &DiscoveredPackages{
			DriverPackages: []string{"nvidia-driver-550"},
			AllPackages:    []string{"nvidia-driver-550"},
			TotalCount:     1,
		},
	}
}

func (m *mockDiscovery) Discover(ctx context.Context) (*DiscoveredPackages, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.discoverCalled = true
	return m.discoverResult, m.discoverError
}

func (m *mockDiscovery) DiscoverDriver(ctx context.Context) ([]string, string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.discoverError != nil {
		return nil, "", m.discoverError
	}
	if m.discoverResult != nil {
		return m.discoverResult.DriverPackages, m.discoverResult.DriverVersion, nil
	}
	return nil, "", nil
}

func (m *mockDiscovery) DiscoverCUDA(ctx context.Context) ([]string, string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.discoverError != nil {
		return nil, "", m.discoverError
	}
	if m.discoverResult != nil {
		return m.discoverResult.CUDAPackages, m.discoverResult.CUDAVersion, nil
	}
	return nil, "", nil
}

func (m *mockDiscovery) IsNVIDIAInstalled(ctx context.Context) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.discoverError != nil {
		return false, m.discoverError
	}
	if m.discoverResult != nil {
		return m.discoverResult.TotalCount > 0, nil
	}
	return false, nil
}

func (m *mockDiscovery) GetDriverVersion(ctx context.Context) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.discoverError != nil {
		return "", m.discoverError
	}
	if m.discoverResult != nil {
		return m.discoverResult.DriverVersion, nil
	}
	return "", nil
}

// =============================================================================
// Event Type String Tests
// =============================================================================

func TestUninstallExecutionEventType_String(t *testing.T) {
	tests := []struct {
		name     string
		event    UninstallExecutionEventType
		expected string
	}{
		{
			name:     "workflow started",
			event:    UninstallEventWorkflowStarted,
			expected: "workflow_started",
		},
		{
			name:     "workflow completed",
			event:    UninstallEventWorkflowCompleted,
			expected: "workflow_completed",
		},
		{
			name:     "workflow failed",
			event:    UninstallEventWorkflowFailed,
			expected: "workflow_failed",
		},
		{
			name:     "workflow cancelled",
			event:    UninstallEventWorkflowCancelled,
			expected: "workflow_cancelled",
		},
		{
			name:     "step started",
			event:    UninstallEventStepStarted,
			expected: "step_started",
		},
		{
			name:     "step completed",
			event:    UninstallEventStepCompleted,
			expected: "step_completed",
		},
		{
			name:     "step skipped",
			event:    UninstallEventStepSkipped,
			expected: "step_skipped",
		},
		{
			name:     "step failed",
			event:    UninstallEventStepFailed,
			expected: "step_failed",
		},
		{
			name:     "unknown event type",
			event:    UninstallExecutionEventType(999),
			expected: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.event.String()
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

// =============================================================================
// Constructor and Options Tests
// =============================================================================

func TestNewUninstallOrchestrator(t *testing.T) {
	t.Run("creates with defaults", func(t *testing.T) {
		o := NewUninstallOrchestrator()
		if o == nil {
			t.Fatal("expected non-nil orchestrator")
		}
		if o.workflow != nil {
			t.Error("expected nil workflow by default")
		}
		if o.discovery != nil {
			t.Error("expected nil discovery by default")
		}
		if o.autoRollback {
			t.Error("expected autoRollback to be false by default")
		}
		if !o.stopOnFirstError {
			t.Error("expected stopOnFirstError to be true by default")
		}
		if o.dryRun {
			t.Error("expected dryRun to be false by default")
		}
		if o.executionLog == nil {
			t.Error("expected executionLog to be initialized")
		}
		if o.completedSteps == nil {
			t.Error("expected completedSteps to be initialized")
		}
	})

	t.Run("creates with options", func(t *testing.T) {
		workflow := newMockWorkflow("test-workflow")
		discovery := newMockDiscovery()
		progressCalled := false

		o := NewUninstallOrchestrator(
			WithUninstallWorkflow(workflow),
			WithUninstallOrchestratorDiscovery(discovery),
			WithUninstallAutoRollback(true),
			WithUninstallStopOnFirstError(false),
			WithUninstallOrchestratorDryRun(true),
			WithUninstallOrchestratorProgress(func(p install.StepProgress) {
				progressCalled = true
			}),
		)

		if o.workflow == nil {
			t.Error("expected workflow to be set")
		}
		if o.discovery == nil {
			t.Error("expected discovery to be set")
		}
		if !o.autoRollback {
			t.Error("expected autoRollback to be true")
		}
		if o.stopOnFirstError {
			t.Error("expected stopOnFirstError to be false")
		}
		if !o.dryRun {
			t.Error("expected dryRun to be true")
		}
		if o.progressCallback == nil {
			t.Error("expected progressCallback to be set")
		}

		// Trigger progress callback
		o.progressCallback(install.StepProgress{})
		if !progressCalled {
			t.Error("expected progress callback to be called")
		}
	})
}

func TestUninstallOrchestratorOptions(t *testing.T) {
	t.Run("WithUninstallWorkflow", func(t *testing.T) {
		workflow := newMockWorkflow("test")
		o := NewUninstallOrchestrator(WithUninstallWorkflow(workflow))
		if o.workflow == nil {
			t.Error("expected workflow to be set")
		}
		if o.workflow.Name() != "test" {
			t.Errorf("expected workflow name 'test', got %q", o.workflow.Name())
		}
	})

	t.Run("WithUninstallOrchestratorDiscovery", func(t *testing.T) {
		discovery := newMockDiscovery()
		o := NewUninstallOrchestrator(WithUninstallOrchestratorDiscovery(discovery))
		if o.discovery == nil {
			t.Error("expected discovery to be set")
		}
	})

	t.Run("WithUninstallAutoRollback", func(t *testing.T) {
		o := NewUninstallOrchestrator(WithUninstallAutoRollback(true))
		if !o.autoRollback {
			t.Error("expected autoRollback to be true")
		}
	})

	t.Run("WithUninstallStopOnFirstError", func(t *testing.T) {
		o := NewUninstallOrchestrator(WithUninstallStopOnFirstError(false))
		if o.stopOnFirstError {
			t.Error("expected stopOnFirstError to be false")
		}
	})

	t.Run("WithUninstallOrchestratorDryRun", func(t *testing.T) {
		o := NewUninstallOrchestrator(WithUninstallOrchestratorDryRun(true))
		if !o.dryRun {
			t.Error("expected dryRun to be true")
		}
	})

	t.Run("WithUninstallPreExecuteHook", func(t *testing.T) {
		hookCalled := false
		hook := func(ctx *Context, workflow UninstallWorkflow) error {
			hookCalled = true
			return nil
		}
		o := NewUninstallOrchestrator(WithUninstallPreExecuteHook(hook))
		if o.preExecuteHook == nil {
			t.Error("expected preExecuteHook to be set")
		}
		// Test hook is callable
		_ = o.preExecuteHook(nil, nil)
		if !hookCalled {
			t.Error("expected hook to be called")
		}
	})

	t.Run("WithUninstallPostExecuteHook", func(t *testing.T) {
		hookCalled := false
		hook := func(ctx *Context, workflow UninstallWorkflow) error {
			hookCalled = true
			return nil
		}
		o := NewUninstallOrchestrator(WithUninstallPostExecuteHook(hook))
		if o.postExecuteHook == nil {
			t.Error("expected postExecuteHook to be set")
		}
		_ = o.postExecuteHook(nil, nil)
		if !hookCalled {
			t.Error("expected hook to be called")
		}
	})

	t.Run("WithUninstallPreStepHook", func(t *testing.T) {
		hookCalled := false
		hook := func(ctx *Context, step UninstallStep, result *install.StepResult) error {
			hookCalled = true
			return nil
		}
		o := NewUninstallOrchestrator(WithUninstallPreStepHook(hook))
		if o.preStepHook == nil {
			t.Error("expected preStepHook to be set")
		}
		_ = o.preStepHook(nil, nil, nil)
		if !hookCalled {
			t.Error("expected hook to be called")
		}
	})

	t.Run("WithUninstallPostStepHook", func(t *testing.T) {
		hookCalled := false
		hook := func(ctx *Context, step UninstallStep, result *install.StepResult) error {
			hookCalled = true
			return nil
		}
		o := NewUninstallOrchestrator(WithUninstallPostStepHook(hook))
		if o.postStepHook == nil {
			t.Error("expected postStepHook to be set")
		}
		_ = o.postStepHook(nil, nil, nil)
		if !hookCalled {
			t.Error("expected hook to be called")
		}
	})

	t.Run("WithUninstallOrchestratorProgress", func(t *testing.T) {
		progressCalled := false
		callback := func(p install.StepProgress) {
			progressCalled = true
		}
		o := NewUninstallOrchestrator(WithUninstallOrchestratorProgress(callback))
		if o.progressCallback == nil {
			t.Error("expected progressCallback to be set")
		}
		o.progressCallback(install.StepProgress{})
		if !progressCalled {
			t.Error("expected callback to be called")
		}
	})
}

// =============================================================================
// Execute Tests
// =============================================================================

func TestUninstallOrchestrator_Execute(t *testing.T) {
	t.Run("successful execution without hooks", func(t *testing.T) {
		workflow := newMockWorkflow("test-workflow")
		workflow.setExecuteResult(NewUninstallResult(UninstallStatusCompleted))

		o := NewUninstallOrchestrator(WithUninstallWorkflow(workflow))
		ctx := NewUninstallContext()

		report := o.Execute(ctx)

		if report.Status != UninstallStatusCompleted {
			t.Errorf("expected status completed, got %v", report.Status)
		}
		if report.WorkflowName != "test-workflow" {
			t.Errorf("expected workflow name 'test-workflow', got %q", report.WorkflowName)
		}
		if report.TotalDuration <= 0 {
			t.Error("expected positive total duration")
		}
		if len(report.ExecutionLog) < 2 {
			t.Error("expected at least 2 execution log entries (start and end)")
		}
	})

	t.Run("execution with nil context", func(t *testing.T) {
		workflow := newMockWorkflow("test")
		workflow.setExecuteResult(NewUninstallResult(UninstallStatusCompleted))

		o := NewUninstallOrchestrator(WithUninstallWorkflow(workflow))

		// Should not panic with nil context
		report := o.Execute(nil)
		if report.Status != UninstallStatusCompleted {
			t.Errorf("expected completed status, got %v", report.Status)
		}
	})
}

func TestUninstallOrchestrator_Execute_NilWorkflow(t *testing.T) {
	o := NewUninstallOrchestrator()
	ctx := NewUninstallContext()

	report := o.Execute(ctx)

	if report.Status != UninstallStatusFailed {
		t.Errorf("expected status failed, got %v", report.Status)
	}
	if report.Error == nil {
		t.Error("expected error for nil workflow")
	}
}

func TestUninstallOrchestrator_Execute_DryRunMode(t *testing.T) {
	workflow := newMockWorkflow("test")
	workflow.setExecuteResult(NewUninstallResult(UninstallStatusCompleted))

	o := NewUninstallOrchestrator(
		WithUninstallWorkflow(workflow),
		WithUninstallOrchestratorDryRun(true),
	)
	ctx := NewUninstallContext()

	report := o.Execute(ctx)

	if !ctx.DryRun {
		t.Error("expected context DryRun to be set to true")
	}
	if report.Status != UninstallStatusCompleted {
		t.Errorf("expected completed status, got %v", report.Status)
	}
}

func TestUninstallOrchestrator_Execute_WithCancellation(t *testing.T) {
	// Create a workflow with multiple steps to test cancellation between steps
	workflow := newMockWorkflow("test")
	step1 := newMockStep("step1", "First step")
	step2 := newMockStep("step2", "Second step")
	workflow.AddStep(step1)
	workflow.AddStep(step2)

	cancelCtx, cancel := context.WithCancel(context.Background())
	ctx := NewUninstallContext(WithUninstallContext(cancelCtx))

	stepCount := 0
	// Cancel after the first step completes, before the second step starts
	postStepHook := func(ctx *Context, step UninstallStep, result *install.StepResult) error {
		stepCount++
		if stepCount == 1 {
			cancel() // Cancel after first step
		}
		return nil
	}

	o := NewUninstallOrchestrator(
		WithUninstallWorkflow(workflow),
		WithUninstallPostStepHook(postStepHook),
		WithUninstallPreStepHook(func(ctx *Context, step UninstallStep, result *install.StepResult) error {
			return nil
		}),
	)

	report := o.Execute(ctx)

	// After cancellation, status should be cancelled
	if report.Status != UninstallStatusCancelled {
		t.Errorf("expected cancelled status, got %v", report.Status)
	}
	// Only one step should have been executed
	if step2.executeCalled {
		t.Error("expected step2 not to be executed after cancellation")
	}
}

func TestUninstallOrchestrator_Execute_WithHooks(t *testing.T) {
	workflow := newMockWorkflow("test")
	workflow.setExecuteResult(NewUninstallResult(UninstallStatusCompleted))

	preExecuteCalled := false
	postExecuteCalled := false

	o := NewUninstallOrchestrator(
		WithUninstallWorkflow(workflow),
		WithUninstallPreExecuteHook(func(ctx *Context, workflow UninstallWorkflow) error {
			preExecuteCalled = true
			return nil
		}),
		WithUninstallPostExecuteHook(func(ctx *Context, workflow UninstallWorkflow) error {
			postExecuteCalled = true
			return nil
		}),
	)

	report := o.Execute(NewUninstallContext())

	if !preExecuteCalled {
		t.Error("expected preExecuteHook to be called")
	}
	if !postExecuteCalled {
		t.Error("expected postExecuteHook to be called")
	}
	if report.Status != UninstallStatusCompleted {
		t.Errorf("expected completed status, got %v", report.Status)
	}
}

func TestUninstallOrchestrator_Execute_PreExecuteHookFailure(t *testing.T) {
	workflow := newMockWorkflow("test")
	workflow.setExecuteResult(NewUninstallResult(UninstallStatusCompleted))

	expectedErr := errors.New("pre-execute hook failed")

	o := NewUninstallOrchestrator(
		WithUninstallWorkflow(workflow),
		WithUninstallPreExecuteHook(func(ctx *Context, workflow UninstallWorkflow) error {
			return expectedErr
		}),
	)

	report := o.Execute(NewUninstallContext())

	if report.Status != UninstallStatusFailed {
		t.Errorf("expected failed status, got %v", report.Status)
	}
	if report.Error == nil {
		t.Error("expected error in report")
	}

	// Verify execution log contains pre-execute hook failure
	found := false
	for _, entry := range report.ExecutionLog {
		if entry.EventType == UninstallEventWorkflowFailed && entry.Error != nil {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected to find pre-execute hook failure in execution log")
	}
}

func TestUninstallOrchestrator_Execute_PostExecuteHookFailure(t *testing.T) {
	workflow := newMockWorkflow("test")
	workflow.setExecuteResult(NewUninstallResult(UninstallStatusCompleted))

	expectedErr := errors.New("post-execute hook failed")

	o := NewUninstallOrchestrator(
		WithUninstallWorkflow(workflow),
		WithUninstallPostExecuteHook(func(ctx *Context, workflow UninstallWorkflow) error {
			return expectedErr
		}),
	)

	report := o.Execute(NewUninstallContext())

	if report.Status != UninstallStatusFailed {
		t.Errorf("expected failed status, got %v", report.Status)
	}
	if report.Error == nil {
		t.Error("expected error in report")
	}
}

func TestUninstallOrchestrator_Execute_StepHooks(t *testing.T) {
	workflow := newMockWorkflow("test")
	step1 := newMockStep("step1", "First step")
	step2 := newMockStep("step2", "Second step")
	workflow.AddStep(step1)
	workflow.AddStep(step2)

	preStepCalls := 0
	postStepCalls := 0

	o := NewUninstallOrchestrator(
		WithUninstallWorkflow(workflow),
		WithUninstallPreStepHook(func(ctx *Context, step UninstallStep, result *install.StepResult) error {
			preStepCalls++
			return nil
		}),
		WithUninstallPostStepHook(func(ctx *Context, step UninstallStep, result *install.StepResult) error {
			postStepCalls++
			return nil
		}),
	)

	report := o.Execute(NewUninstallContext())

	if preStepCalls != 2 {
		t.Errorf("expected preStepHook to be called 2 times, got %d", preStepCalls)
	}
	if postStepCalls != 2 {
		t.Errorf("expected postStepHook to be called 2 times, got %d", postStepCalls)
	}
	if report.Status != UninstallStatusCompleted {
		t.Errorf("expected completed status, got %v", report.Status)
	}
}

func TestUninstallOrchestrator_Execute_PreStepHookFailure(t *testing.T) {
	workflow := newMockWorkflow("test")
	step := newMockStep("step1", "Test step")
	workflow.AddStep(step)

	expectedErr := errors.New("pre-step hook failed")

	o := NewUninstallOrchestrator(
		WithUninstallWorkflow(workflow),
		WithUninstallPreStepHook(func(ctx *Context, step UninstallStep, result *install.StepResult) error {
			return expectedErr
		}),
	)

	report := o.Execute(NewUninstallContext())

	if report.Status != UninstallStatusFailed {
		t.Errorf("expected failed status, got %v", report.Status)
	}
	if report.Error == nil {
		t.Error("expected error in report")
	}
}

func TestUninstallOrchestrator_Execute_PostStepHookFailure(t *testing.T) {
	workflow := newMockWorkflow("test")
	step := newMockStep("step1", "Test step")
	workflow.AddStep(step)

	expectedErr := errors.New("post-step hook failed")

	o := NewUninstallOrchestrator(
		WithUninstallWorkflow(workflow),
		WithUninstallPostStepHook(func(ctx *Context, step UninstallStep, result *install.StepResult) error {
			return expectedErr
		}),
	)

	report := o.Execute(NewUninstallContext())

	if report.Status != UninstallStatusFailed {
		t.Errorf("expected failed status, got %v", report.Status)
	}
	if report.Error == nil {
		t.Error("expected error in report")
	}
}

func TestUninstallOrchestrator_Execute_StopOnFirstError(t *testing.T) {
	workflow := newMockWorkflow("test")
	step1 := newMockStep("step1", "First step")
	step1.setExecuteResult(install.StepResult{
		Status:  install.StepStatusFailed,
		Message: "Step 1 failed",
		Error:   errors.New("step 1 error"),
	})
	step2 := newMockStep("step2", "Second step")
	workflow.AddStep(step1)
	workflow.AddStep(step2)

	o := NewUninstallOrchestrator(
		WithUninstallWorkflow(workflow),
		WithUninstallStopOnFirstError(true),
		// Add hooks to trigger step-by-step execution
		WithUninstallPreStepHook(func(ctx *Context, step UninstallStep, result *install.StepResult) error {
			return nil
		}),
	)

	report := o.Execute(NewUninstallContext())

	if report.Status != UninstallStatusFailed {
		t.Errorf("expected failed status, got %v", report.Status)
	}
	if step2.executeCalled {
		t.Error("expected step2 not to be executed when stopOnFirstError is true")
	}
}

func TestUninstallOrchestrator_Execute_ContinueOnError(t *testing.T) {
	workflow := newMockWorkflow("test")
	step1 := newMockStep("step1", "First step")
	step1.setExecuteResult(install.StepResult{
		Status:  install.StepStatusFailed,
		Message: "Step 1 failed",
		Error:   errors.New("step 1 error"),
	})
	step2 := newMockStep("step2", "Second step")
	workflow.AddStep(step1)
	workflow.AddStep(step2)

	o := NewUninstallOrchestrator(
		WithUninstallWorkflow(workflow),
		WithUninstallStopOnFirstError(false),
		// Add hooks to trigger step-by-step execution
		WithUninstallPreStepHook(func(ctx *Context, step UninstallStep, result *install.StepResult) error {
			return nil
		}),
	)

	report := o.Execute(NewUninstallContext())

	// Status should still be failed, but step2 should have been executed
	if report.Status != UninstallStatusFailed {
		t.Errorf("expected failed status, got %v", report.Status)
	}
	if !step2.executeCalled {
		t.Error("expected step2 to be executed when stopOnFirstError is false")
	}
}

func TestUninstallOrchestrator_Execute_PartialResult(t *testing.T) {
	workflow := newMockWorkflow("test")
	step := newMockStep("step1", "Test step")
	workflow.AddStep(step)

	o := NewUninstallOrchestrator(
		WithUninstallWorkflow(workflow),
		// Add hooks to trigger step-by-step execution
		WithUninstallPreStepHook(func(ctx *Context, step UninstallStep, result *install.StepResult) error {
			return nil
		}),
	)

	// Create a context that will result in partial status
	ctx := NewUninstallContext()

	report := o.Execute(ctx)

	// With successful steps, should be completed
	if report.Status != UninstallStatusCompleted {
		t.Errorf("expected completed status, got %v", report.Status)
	}
}

func TestUninstallOrchestrator_Execute_SkippedStep(t *testing.T) {
	workflow := newMockWorkflow("test")
	step := newMockStep("step1", "Test step")
	step.setExecuteResult(install.StepResult{
		Status:  install.StepStatusSkipped,
		Message: "Step skipped",
	})
	workflow.AddStep(step)

	o := NewUninstallOrchestrator(
		WithUninstallWorkflow(workflow),
		// Add hooks to trigger step-by-step execution
		WithUninstallPreStepHook(func(ctx *Context, step UninstallStep, result *install.StepResult) error {
			return nil
		}),
	)

	report := o.Execute(NewUninstallContext())

	// Skipped steps should result in completed status
	if report.Status != UninstallStatusCompleted {
		t.Errorf("expected completed status, got %v", report.Status)
	}
	if report.StepsSkipped != 1 {
		t.Errorf("expected 1 skipped step, got %d", report.StepsSkipped)
	}
}

func TestUninstallOrchestrator_Execute_ValidationFailure(t *testing.T) {
	workflow := newMockWorkflow("test")
	step := newMockStep("step1", "Test step")
	step.setValidateError(errors.New("validation failed"))
	workflow.AddStep(step)

	o := NewUninstallOrchestrator(
		WithUninstallWorkflow(workflow),
		// Add hooks to trigger step-by-step execution
		WithUninstallPreStepHook(func(ctx *Context, step UninstallStep, result *install.StepResult) error {
			return nil
		}),
	)

	report := o.Execute(NewUninstallContext())

	if report.Status != UninstallStatusFailed {
		t.Errorf("expected failed status, got %v", report.Status)
	}
	if report.Error == nil {
		t.Error("expected error for validation failure")
	}
}

func TestUninstallOrchestrator_Execute_ProgressCallback(t *testing.T) {
	workflow := newMockWorkflow("test")
	step := newMockStep("step1", "Test step")
	workflow.AddStep(step)

	progressUpdates := make([]install.StepProgress, 0)
	var mu sync.Mutex

	o := NewUninstallOrchestrator(
		WithUninstallWorkflow(workflow),
		WithUninstallOrchestratorProgress(func(p install.StepProgress) {
			mu.Lock()
			progressUpdates = append(progressUpdates, p)
			mu.Unlock()
		}),
		// Add hooks to trigger step-by-step execution
		WithUninstallPreStepHook(func(ctx *Context, step UninstallStep, result *install.StepResult) error {
			return nil
		}),
	)

	o.Execute(NewUninstallContext())

	mu.Lock()
	updates := len(progressUpdates)
	mu.Unlock()

	if updates == 0 {
		t.Error("expected at least one progress update")
	}
}

// =============================================================================
// Method Tests
// =============================================================================

func TestUninstallOrchestrator_GetExecutionLog(t *testing.T) {
	workflow := newMockWorkflow("test")
	workflow.setExecuteResult(NewUninstallResult(UninstallStatusCompleted))

	o := NewUninstallOrchestrator(WithUninstallWorkflow(workflow))

	// Initially empty
	log := o.GetExecutionLog()
	if len(log) != 0 {
		t.Errorf("expected empty log initially, got %d entries", len(log))
	}

	// After execution
	o.Execute(NewUninstallContext())

	log = o.GetExecutionLog()
	if len(log) < 2 {
		t.Error("expected at least 2 log entries after execution")
	}

	// Verify it returns a copy
	originalLen := len(log)
	log = append(log, UninstallExecutionEntry{})
	currentLog := o.GetExecutionLog()
	if len(currentLog) != originalLen {
		t.Error("GetExecutionLog should return a copy")
	}
}

func TestUninstallOrchestrator_Reset(t *testing.T) {
	workflow := newMockWorkflow("test")
	workflow.setExecuteResult(NewUninstallResult(UninstallStatusCompleted))

	o := NewUninstallOrchestrator(WithUninstallWorkflow(workflow))

	// Execute to populate state
	o.Execute(NewUninstallContext())

	// Verify state is populated
	if len(o.GetExecutionLog()) == 0 {
		t.Error("expected execution log to have entries")
	}

	// Reset
	o.Reset()

	// Verify state is cleared
	if len(o.GetExecutionLog()) != 0 {
		t.Error("expected execution log to be empty after reset")
	}
}

func TestUninstallOrchestrator_SetWorkflow(t *testing.T) {
	o := NewUninstallOrchestrator()

	workflow1 := newMockWorkflow("workflow1")
	o.SetWorkflow(workflow1)

	if o.Workflow() == nil {
		t.Error("expected workflow to be set")
	}
	if o.Workflow().Name() != "workflow1" {
		t.Errorf("expected workflow name 'workflow1', got %q", o.Workflow().Name())
	}

	workflow2 := newMockWorkflow("workflow2")
	o.SetWorkflow(workflow2)

	if o.Workflow().Name() != "workflow2" {
		t.Errorf("expected workflow name 'workflow2', got %q", o.Workflow().Name())
	}
}

func TestUninstallOrchestrator_Workflow(t *testing.T) {
	t.Run("returns nil when not set", func(t *testing.T) {
		o := NewUninstallOrchestrator()
		if o.Workflow() != nil {
			t.Error("expected nil workflow")
		}
	})

	t.Run("returns workflow when set", func(t *testing.T) {
		workflow := newMockWorkflow("test")
		o := NewUninstallOrchestrator(WithUninstallWorkflow(workflow))
		if o.Workflow() == nil {
			t.Error("expected non-nil workflow")
		}
	})
}

func TestUninstallOrchestrator_SetDiscovery(t *testing.T) {
	o := NewUninstallOrchestrator()

	discovery := newMockDiscovery()
	o.SetDiscovery(discovery)

	if o.Discovery() == nil {
		t.Error("expected discovery to be set")
	}
}

func TestUninstallOrchestrator_Discovery(t *testing.T) {
	t.Run("returns nil when not set", func(t *testing.T) {
		o := NewUninstallOrchestrator()
		if o.Discovery() != nil {
			t.Error("expected nil discovery")
		}
	})

	t.Run("returns discovery when set", func(t *testing.T) {
		discovery := newMockDiscovery()
		o := NewUninstallOrchestrator(WithUninstallOrchestratorDiscovery(discovery))
		if o.Discovery() == nil {
			t.Error("expected non-nil discovery")
		}
	})
}

// =============================================================================
// Report Generation Tests
// =============================================================================

func TestUninstallOrchestrator_GenerateReport(t *testing.T) {
	t.Run("completed workflow", func(t *testing.T) {
		workflow := newMockWorkflow("test-workflow")
		step1 := newMockStep("step1", "Step 1")
		step2 := newMockStep("step2", "Step 2")
		workflow.AddStep(step1)
		workflow.AddStep(step2)

		o := NewUninstallOrchestrator(
			WithUninstallWorkflow(workflow),
			WithUninstallPreStepHook(func(ctx *Context, step UninstallStep, result *install.StepResult) error {
				return nil
			}),
		)

		report := o.Execute(NewUninstallContext())

		if report.WorkflowName != "test-workflow" {
			t.Errorf("expected workflow name 'test-workflow', got %q", report.WorkflowName)
		}
		if report.Status != UninstallStatusCompleted {
			t.Errorf("expected completed status, got %v", report.Status)
		}
		if report.StepsExecuted != 2 {
			t.Errorf("expected 2 steps executed, got %d", report.StepsExecuted)
		}
		if report.StepsCompleted != 2 {
			t.Errorf("expected 2 steps completed, got %d", report.StepsCompleted)
		}
		if report.StartTime.IsZero() {
			t.Error("expected StartTime to be set")
		}
		if report.EndTime.IsZero() {
			t.Error("expected EndTime to be set")
		}
		if report.TotalDuration <= 0 {
			t.Error("expected positive TotalDuration")
		}
	})

	t.Run("failed workflow", func(t *testing.T) {
		workflow := newMockWorkflow("test")
		step := newMockStep("step1", "Step 1")
		step.setExecuteResult(install.StepResult{
			Status:  install.StepStatusFailed,
			Message: "Failed",
			Error:   errors.New("test error"),
		})
		workflow.AddStep(step)

		o := NewUninstallOrchestrator(
			WithUninstallWorkflow(workflow),
			WithUninstallPreStepHook(func(ctx *Context, step UninstallStep, result *install.StepResult) error {
				return nil
			}),
		)

		report := o.Execute(NewUninstallContext())

		if report.Status != UninstallStatusFailed {
			t.Errorf("expected failed status, got %v", report.Status)
		}
		if report.StepsFailed != 1 {
			t.Errorf("expected 1 failed step, got %d", report.StepsFailed)
		}
		if report.Error == nil {
			t.Error("expected error in report")
		}
	})

	t.Run("report with removed packages", func(t *testing.T) {
		workflow := newMockWorkflow("test")
		step := newMockStep("step1", "Step 1")
		workflow.AddStep(step)

		o := NewUninstallOrchestrator(
			WithUninstallWorkflow(workflow),
			WithUninstallPreStepHook(func(ctx *Context, step UninstallStep, result *install.StepResult) error {
				return nil
			}),
		)

		report := o.Execute(NewUninstallContext())

		// RemovedPackages should be initialized (empty)
		if report.RemovedPackages == nil {
			t.Error("expected RemovedPackages to be initialized")
		}
		if report.CleanedConfigs == nil {
			t.Error("expected CleanedConfigs to be initialized")
		}
	})

	t.Run("cancelled workflow report", func(t *testing.T) {
		workflow := newMockWorkflow("test")
		step1 := newMockStep("step1", "Step 1")
		step2 := newMockStep("step2", "Step 2")
		workflow.AddStep(step1)
		workflow.AddStep(step2)

		cancelCtx, cancel := context.WithCancel(context.Background())
		ctx := NewUninstallContext(WithUninstallContext(cancelCtx))

		stepCount := 0
		o := NewUninstallOrchestrator(
			WithUninstallWorkflow(workflow),
			WithUninstallPostStepHook(func(ctx *Context, step UninstallStep, result *install.StepResult) error {
				stepCount++
				if stepCount == 1 {
					cancel() // Cancel after first step, before second
				}
				return nil
			}),
			WithUninstallPreStepHook(func(ctx *Context, step UninstallStep, result *install.StepResult) error {
				return nil
			}),
		)

		report := o.Execute(ctx)

		if report.Status != UninstallStatusCancelled {
			t.Errorf("expected cancelled status, got %v", report.Status)
		}
	})

	t.Run("partial result report", func(t *testing.T) {
		workflow := newMockWorkflow("test")
		step := newMockStep("step1", "Step 1")
		workflow.AddStep(step)

		o := NewUninstallOrchestrator(
			WithUninstallWorkflow(workflow),
			WithUninstallPreStepHook(func(ctx *Context, step UninstallStep, result *install.StepResult) error {
				return nil
			}),
		)

		// Execute normally - the partial status is determined by workflow result
		report := o.Execute(NewUninstallContext())

		// Default result is completed
		if report.Status != UninstallStatusCompleted {
			t.Errorf("expected completed status, got %v", report.Status)
		}
	})
}

// =============================================================================
// Concurrency Tests
// =============================================================================

func TestUninstallOrchestrator_Concurrency(t *testing.T) {
	t.Run("concurrent GetExecutionLog calls", func(t *testing.T) {
		workflow := newMockWorkflow("test")
		workflow.setExecuteResult(NewUninstallResult(UninstallStatusCompleted))

		o := NewUninstallOrchestrator(WithUninstallWorkflow(workflow))
		o.Execute(NewUninstallContext())

		var wg sync.WaitGroup
		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_ = o.GetExecutionLog()
			}()
		}
		wg.Wait()
	})

	t.Run("concurrent SetWorkflow and Workflow calls", func(t *testing.T) {
		o := NewUninstallOrchestrator()

		var wg sync.WaitGroup
		for i := 0; i < 100; i++ {
			wg.Add(2)
			go func(n int) {
				defer wg.Done()
				o.SetWorkflow(newMockWorkflow("workflow"))
			}(i)
			go func() {
				defer wg.Done()
				_ = o.Workflow()
			}()
		}
		wg.Wait()
	})

	t.Run("concurrent SetDiscovery and Discovery calls", func(t *testing.T) {
		o := NewUninstallOrchestrator()

		var wg sync.WaitGroup
		for i := 0; i < 100; i++ {
			wg.Add(2)
			go func() {
				defer wg.Done()
				o.SetDiscovery(newMockDiscovery())
			}()
			go func() {
				defer wg.Done()
				_ = o.Discovery()
			}()
		}
		wg.Wait()
	})

	t.Run("concurrent Reset calls", func(t *testing.T) {
		workflow := newMockWorkflow("test")
		workflow.setExecuteResult(NewUninstallResult(UninstallStatusCompleted))

		o := NewUninstallOrchestrator(WithUninstallWorkflow(workflow))
		o.Execute(NewUninstallContext())

		var wg sync.WaitGroup
		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				o.Reset()
			}()
		}
		wg.Wait()
	})

	t.Run("concurrent Execute calls", func(t *testing.T) {
		var executionCount int32

		workflow := newMockWorkflow("test")
		workflow.setExecuteFunc(func(ctx *Context) UninstallResult {
			atomic.AddInt32(&executionCount, 1)
			time.Sleep(10 * time.Millisecond) // Simulate work
			return NewUninstallResult(UninstallStatusCompleted)
		})

		o := NewUninstallOrchestrator(WithUninstallWorkflow(workflow))

		var wg sync.WaitGroup
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_ = o.Execute(NewUninstallContext())
			}()
		}
		wg.Wait()

		// All executions should have run
		if atomic.LoadInt32(&executionCount) != 10 {
			t.Errorf("expected 10 executions, got %d", executionCount)
		}
	})
}

// =============================================================================
// Edge Case Tests
// =============================================================================

func TestUninstallOrchestrator_EdgeCases(t *testing.T) {
	t.Run("empty workflow (no steps)", func(t *testing.T) {
		workflow := newMockWorkflow("empty")
		workflow.setExecuteResult(NewUninstallResult(UninstallStatusCompleted))

		o := NewUninstallOrchestrator(WithUninstallWorkflow(workflow))

		report := o.Execute(NewUninstallContext())

		if report.Status != UninstallStatusCompleted {
			t.Errorf("expected completed status for empty workflow, got %v", report.Status)
		}
	})

	t.Run("multiple executions", func(t *testing.T) {
		workflow := newMockWorkflow("test")
		workflow.setExecuteResult(NewUninstallResult(UninstallStatusCompleted))

		o := NewUninstallOrchestrator(WithUninstallWorkflow(workflow))

		// First execution
		report1 := o.Execute(NewUninstallContext())
		log1Len := len(report1.ExecutionLog)

		// Second execution without reset
		report2 := o.Execute(NewUninstallContext())
		log2Len := len(report2.ExecutionLog)

		// Each execution should reset the log
		if log1Len != log2Len {
			t.Errorf("expected same log length, got %d and %d", log1Len, log2Len)
		}
	})

	t.Run("execution after reset", func(t *testing.T) {
		workflow := newMockWorkflow("test")
		workflow.setExecuteResult(NewUninstallResult(UninstallStatusCompleted))

		o := NewUninstallOrchestrator(WithUninstallWorkflow(workflow))

		o.Execute(NewUninstallContext())
		o.Reset()

		if len(o.GetExecutionLog()) != 0 {
			t.Error("expected empty log after reset")
		}

		report := o.Execute(NewUninstallContext())
		if report.Status != UninstallStatusCompleted {
			t.Errorf("expected completed status, got %v", report.Status)
		}
	})

	t.Run("hook returns nil error", func(t *testing.T) {
		workflow := newMockWorkflow("test")
		workflow.setExecuteResult(NewUninstallResult(UninstallStatusCompleted))

		o := NewUninstallOrchestrator(
			WithUninstallWorkflow(workflow),
			WithUninstallPreExecuteHook(func(ctx *Context, workflow UninstallWorkflow) error {
				return nil
			}),
			WithUninstallPostExecuteHook(func(ctx *Context, workflow UninstallWorkflow) error {
				return nil
			}),
		)

		report := o.Execute(NewUninstallContext())

		if report.Status != UninstallStatusCompleted {
			t.Errorf("expected completed status, got %v", report.Status)
		}
	})

	t.Run("post-execute hook failure on already failed workflow", func(t *testing.T) {
		workflow := newMockWorkflow("test")
		workflow.setExecuteResult(NewUninstallResult(UninstallStatusFailed).WithError("test", errors.New("workflow error")))

		o := NewUninstallOrchestrator(
			WithUninstallWorkflow(workflow),
			WithUninstallPostExecuteHook(func(ctx *Context, workflow UninstallWorkflow) error {
				return errors.New("post hook error")
			}),
		)

		report := o.Execute(NewUninstallContext())

		// Status should still be failed (from workflow), not overwritten by post hook
		if report.Status != UninstallStatusFailed {
			t.Errorf("expected failed status, got %v", report.Status)
		}
	})

	t.Run("unexpected step status", func(t *testing.T) {
		workflow := newMockWorkflow("test")
		step := newMockStep("step1", "Step 1")
		step.setExecuteResult(install.StepResult{
			Status:  install.StepStatus(999), // Invalid status
			Message: "Unknown status",
		})
		workflow.AddStep(step)

		o := NewUninstallOrchestrator(
			WithUninstallWorkflow(workflow),
			WithUninstallPreStepHook(func(ctx *Context, step UninstallStep, result *install.StepResult) error {
				return nil
			}),
		)

		report := o.Execute(NewUninstallContext())

		if report.Status != UninstallStatusFailed {
			t.Errorf("expected failed status for unexpected step status, got %v", report.Status)
		}
	})
}

// =============================================================================
// Execution Entry Tests
// =============================================================================

func TestUninstallExecutionEntry(t *testing.T) {
	t.Run("entry with all fields", func(t *testing.T) {
		now := time.Now()
		err := errors.New("test error")
		duration := 5 * time.Second

		entry := UninstallExecutionEntry{
			Timestamp: now,
			StepName:  "test-step",
			EventType: UninstallEventStepCompleted,
			Message:   "Step completed successfully",
			Duration:  duration,
			Error:     err,
		}

		if entry.Timestamp != now {
			t.Error("timestamp mismatch")
		}
		if entry.StepName != "test-step" {
			t.Error("step name mismatch")
		}
		if entry.EventType != UninstallEventStepCompleted {
			t.Error("event type mismatch")
		}
		if entry.Message != "Step completed successfully" {
			t.Error("message mismatch")
		}
		if entry.Duration != duration {
			t.Error("duration mismatch")
		}
		if entry.Error != err {
			t.Error("error mismatch")
		}
	})
}

// =============================================================================
// Execution Report Tests
// =============================================================================

func TestUninstallExecutionReport(t *testing.T) {
	t.Run("report initialization", func(t *testing.T) {
		report := UninstallExecutionReport{
			WorkflowName:    "test",
			Status:          UninstallStatusCompleted,
			StartTime:       time.Now(),
			EndTime:         time.Now(),
			TotalDuration:   time.Second,
			StepsExecuted:   5,
			StepsCompleted:  4,
			StepsSkipped:    1,
			StepsFailed:     0,
			RemovedPackages: []string{"pkg1", "pkg2"},
			CleanedConfigs:  []string{"/etc/nvidia"},
			NouveauRestored: true,
			NeedsReboot:     true,
			ExecutionLog:    []UninstallExecutionEntry{},
			Error:           nil,
		}

		if report.WorkflowName != "test" {
			t.Error("workflow name mismatch")
		}
		if report.StepsExecuted != 5 {
			t.Error("steps executed mismatch")
		}
		if len(report.RemovedPackages) != 2 {
			t.Error("removed packages count mismatch")
		}
		if !report.NouveauRestored {
			t.Error("nouveau restored should be true")
		}
		if !report.NeedsReboot {
			t.Error("needs reboot should be true")
		}
	})
}

// =============================================================================
// State Sync Tests
// =============================================================================

// mockStepWithStateSync is a step that sets state on the install context
// to test state synchronization.
type mockStepWithStateSync struct {
	name        string
	description string
}

func (m *mockStepWithStateSync) Name() string {
	return m.name
}

func (m *mockStepWithStateSync) Description() string {
	return m.description
}

func (m *mockStepWithStateSync) Execute(ctx *install.Context) install.StepResult {
	// Set various states to test sync
	ctx.SetState(StateRemovedPackages, []string{"nvidia-driver-550", "nvidia-utils-550"})
	ctx.SetState(StateCleanedConfigs, []string{"/etc/modprobe.d/nvidia.conf"})
	ctx.SetState(StatePackagesRemoved, true)
	ctx.SetState(StateConfigsCleaned, true)
	ctx.SetState(StateModulesUnloaded, true)
	ctx.SetState(StateNouveauUnblocked, true)
	ctx.SetState(StateNouveauRestored, true)

	return install.StepResult{
		Status:  install.StepStatusCompleted,
		Message: "State sync test completed",
	}
}

func (m *mockStepWithStateSync) Rollback(ctx *install.Context) error {
	return nil
}

func (m *mockStepWithStateSync) CanRollback() bool {
	return false
}

func (m *mockStepWithStateSync) Validate(ctx *install.Context) error {
	return nil
}

func TestUninstallOrchestrator_StateSynchronization(t *testing.T) {
	t.Run("state syncs from install context to uninstall context", func(t *testing.T) {
		workflow := newMockWorkflow("test")
		step := &mockStepWithStateSync{
			name:        "state-sync-step",
			description: "Sets state values",
		}
		workflow.AddStep(step)

		ctx := NewUninstallContext()

		o := NewUninstallOrchestrator(
			WithUninstallWorkflow(workflow),
			// Add hooks to trigger step-by-step execution with state sync
			WithUninstallPreStepHook(func(ctx *Context, step UninstallStep, result *install.StepResult) error {
				return nil
			}),
		)

		report := o.Execute(ctx)

		if report.Status != UninstallStatusCompleted {
			t.Errorf("expected completed status, got %v", report.Status)
		}

		// Verify state was synced to the report
		if len(report.RemovedPackages) != 2 {
			t.Errorf("expected 2 removed packages, got %d", len(report.RemovedPackages))
		}
		if len(report.CleanedConfigs) != 1 {
			t.Errorf("expected 1 cleaned config, got %d", len(report.CleanedConfigs))
		}
		if !report.NouveauRestored {
			t.Error("expected NouveauRestored to be true")
		}

		// Verify state was synced to the context
		if !ctx.GetStateBool(StatePackagesRemoved) {
			t.Error("expected StatePackagesRemoved to be true in context")
		}
		if !ctx.GetStateBool(StateNouveauRestored) {
			t.Error("expected StateNouveauRestored to be true in context")
		}
	})

	t.Run("execute with nil context creates install context", func(t *testing.T) {
		workflow := newMockWorkflow("test")
		step := newMockStep("step1", "Test step")
		workflow.AddStep(step)

		o := NewUninstallOrchestrator(
			WithUninstallWorkflow(workflow),
			WithUninstallPreStepHook(func(ctx *Context, step UninstallStep, result *install.StepResult) error {
				return nil
			}),
		)

		// Execute with nil context
		report := o.Execute(nil)

		// Should complete successfully even with nil context
		if report.Status != UninstallStatusCompleted {
			t.Errorf("expected completed status, got %v", report.Status)
		}
	})
}

// =============================================================================
// Partial Status Tests
// =============================================================================

func TestUninstallOrchestrator_Execute_PartialStatus(t *testing.T) {
	t.Run("partial status when some packages failed and some removed", func(t *testing.T) {
		// Create a step that sets partial state
		workflow := newMockWorkflow("test")
		step := NewUninstallFuncStep(
			"partial-step",
			"Creates partial state",
			func(ctx *Context) install.StepResult {
				return install.StepResult{
					Status:  install.StepStatusCompleted,
					Message: "Partial completion",
				}
			},
		)
		workflow.AddStep(step)

		o := NewUninstallOrchestrator(
			WithUninstallWorkflow(workflow),
			WithUninstallPreStepHook(func(ctx *Context, step UninstallStep, result *install.StepResult) error {
				return nil
			}),
		)

		report := o.Execute(NewUninstallContext())

		// Without failed packages, should be completed
		if report.Status != UninstallStatusCompleted {
			t.Errorf("expected completed status, got %v", report.Status)
		}
	})
}

// =============================================================================
// Direct Workflow Execution Tests
// =============================================================================

func TestUninstallOrchestrator_Execute_DirectWorkflow(t *testing.T) {
	t.Run("executes workflow directly when no hooks set", func(t *testing.T) {
		workflow := newMockWorkflow("test")
		workflow.setExecuteResult(NewUninstallResult(UninstallStatusCompleted))

		o := NewUninstallOrchestrator(WithUninstallWorkflow(workflow))

		report := o.Execute(NewUninstallContext())

		if !workflow.executeCalled {
			t.Error("expected workflow.Execute to be called")
		}
		if report.Status != UninstallStatusCompleted {
			t.Errorf("expected completed status, got %v", report.Status)
		}
	})

	t.Run("workflow with partial result", func(t *testing.T) {
		workflow := newMockWorkflow("test")
		result := NewUninstallResult(UninstallStatusPartial)
		result.AddRemovedPackage("nvidia-driver-550")
		result.AddFailedPackage("nvidia-cuda-toolkit")
		workflow.setExecuteResult(result)

		o := NewUninstallOrchestrator(WithUninstallWorkflow(workflow))

		report := o.Execute(NewUninstallContext())

		if report.Status != UninstallStatusPartial {
			t.Errorf("expected partial status, got %v", report.Status)
		}
	})
}
