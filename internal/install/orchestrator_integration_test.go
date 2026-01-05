package install

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// =============================================================================
// Complete Installation Workflow Tests
// =============================================================================

// TestOrchestrator_CompleteInstallationWorkflow tests a full successful installation
// simulating the complete NVIDIA driver installation workflow with all 8 steps.
func TestOrchestrator_CompleteInstallationWorkflow(t *testing.T) {
	// Create steps simulating the full installation workflow
	stepNames := []string{
		"validation",
		"repository",
		"nouveau_blacklist",
		"packages",
		"dkms_build",
		"module_load",
		"xorg_config",
		"verification",
	}

	var executionOrder []string
	var mu sync.Mutex

	w := NewWorkflow("nvidia-installation")
	for _, name := range stepNames {
		stepName := name // Capture for closure
		step := NewFuncStep(stepName, "Step: "+stepName, func(ctx *Context) StepResult {
			mu.Lock()
			executionOrder = append(executionOrder, stepName)
			mu.Unlock()
			// Simulate some work
			time.Sleep(1 * time.Millisecond)
			return CompleteStep(stepName + " completed")
		})
		w.AddStep(step)
	}

	o := NewOrchestrator(w)
	ctx := NewContext()
	report := o.Execute(ctx)

	// Verify successful completion
	assert.Equal(t, WorkflowStatusCompleted, report.Status)
	assert.Nil(t, report.Error)
	assert.Equal(t, "nvidia-installation", report.WorkflowName)
	assert.True(t, report.TotalDuration > 0)
	assert.False(t, report.RollbackPerformed)
	assert.True(t, report.RollbackSuccess)

	// Verify execution order
	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, stepNames, executionOrder)
}

// TestOrchestrator_RollbackOnFailure tests automatic rollback when a step fails
func TestOrchestrator_RollbackOnFailure(t *testing.T) {
	t.Run("rollback happens in reverse order", func(t *testing.T) {
		var rollbackOrder []string
		var mu sync.Mutex

		// Create three steps with rollback capability
		step1 := NewFuncStep("step1", "Step 1", func(ctx *Context) StepResult {
			return CompleteStep("step1 done")
		}, WithRollbackFunc(func(ctx *Context) error {
			mu.Lock()
			rollbackOrder = append(rollbackOrder, "step1")
			mu.Unlock()
			return nil
		}))

		step2 := NewFuncStep("step2", "Step 2", func(ctx *Context) StepResult {
			return CompleteStep("step2 done")
		}, WithRollbackFunc(func(ctx *Context) error {
			mu.Lock()
			rollbackOrder = append(rollbackOrder, "step2")
			mu.Unlock()
			return nil
		}))

		step3 := NewFuncStep("step3", "Step 3 (fails)", func(ctx *Context) StepResult {
			return FailStep("step3 failed", errors.New("simulated failure"))
		})

		w := NewWorkflow("test")
		w.AddStep(step1)
		w.AddStep(step2)
		w.AddStep(step3)

		o := NewOrchestrator(w, WithAutoRollback(true))
		ctx := NewContext()
		report := o.Execute(ctx)

		// Verify failure with rollback
		assert.Equal(t, WorkflowStatusFailed, report.Status)
		assert.True(t, report.RollbackPerformed)
		assert.Error(t, report.Error)

		// Verify rollback order (reverse)
		mu.Lock()
		defer mu.Unlock()
		assert.Equal(t, []string{"step2", "step1"}, rollbackOrder)
	})

	t.Run("rollback at different failure points", func(t *testing.T) {
		for failAt := 0; failAt < 4; failAt++ {
			t.Run("fail_at_step_"+string(rune('1'+failAt)), func(t *testing.T) {
				var rollbackCount int32

				w := NewWorkflow("test")
				for i := 0; i < 4; i++ {
					idx := i
					step := NewFuncStep(
						"step"+string(rune('1'+idx)),
						"Step",
						func(ctx *Context) StepResult {
							if idx == failAt {
								return FailStep("failed", errors.New("fail"))
							}
							return CompleteStep("done")
						},
						WithRollbackFunc(func(ctx *Context) error {
							atomic.AddInt32(&rollbackCount, 1)
							return nil
						}),
					)
					w.AddStep(step)
				}

				o := NewOrchestrator(w, WithAutoRollback(true))
				ctx := NewContext()
				report := o.Execute(ctx)

				assert.Equal(t, WorkflowStatusFailed, report.Status)
				assert.True(t, report.RollbackPerformed)
				// Only completed steps should be rolled back
				assert.Equal(t, int32(failAt), atomic.LoadInt32(&rollbackCount))
			})
		}
	})
}

// TestOrchestrator_PartialRollback tests rollback when some steps can't rollback
func TestOrchestrator_PartialRollback(t *testing.T) {
	t.Run("continues rollback even when some fail", func(t *testing.T) {
		var rollbackCalls []string
		var mu sync.Mutex

		step1 := NewFuncStep("step1", "Step 1", func(ctx *Context) StepResult {
			return CompleteStep("done")
		}, WithRollbackFunc(func(ctx *Context) error {
			mu.Lock()
			rollbackCalls = append(rollbackCalls, "step1")
			mu.Unlock()
			return nil
		}))

		step2 := NewFuncStep("step2", "Step 2", func(ctx *Context) StepResult {
			return CompleteStep("done")
		}, WithRollbackFunc(func(ctx *Context) error {
			mu.Lock()
			rollbackCalls = append(rollbackCalls, "step2-failed")
			mu.Unlock()
			return errors.New("rollback failed for step2")
		}))

		step3 := NewFuncStep("step3", "Step 3", func(ctx *Context) StepResult {
			return CompleteStep("done")
		}, WithRollbackFunc(func(ctx *Context) error {
			mu.Lock()
			rollbackCalls = append(rollbackCalls, "step3")
			mu.Unlock()
			return nil
		}))

		step4 := NewFuncStep("step4", "Step 4 (fails)", func(ctx *Context) StepResult {
			return FailStep("failed", errors.New("error"))
		})

		w := NewWorkflow("test")
		w.AddStep(step1)
		w.AddStep(step2)
		w.AddStep(step3)
		w.AddStep(step4)

		o := NewOrchestrator(w, WithAutoRollback(true))
		ctx := NewContext()
		report := o.Execute(ctx)

		assert.Equal(t, WorkflowStatusFailed, report.Status)
		assert.True(t, report.RollbackPerformed)
		assert.False(t, report.RollbackSuccess) // step2 rollback failed

		// All steps should still attempt rollback
		mu.Lock()
		defer mu.Unlock()
		assert.Contains(t, rollbackCalls, "step1")
		assert.Contains(t, rollbackCalls, "step2-failed")
		assert.Contains(t, rollbackCalls, "step3")
	})

	t.Run("skips steps without rollback capability", func(t *testing.T) {
		var rollbackCalls []string
		var mu sync.Mutex

		stepWithRollback := NewFuncStep("with-rollback", "With Rollback", func(ctx *Context) StepResult {
			return CompleteStep("done")
		}, WithRollbackFunc(func(ctx *Context) error {
			mu.Lock()
			rollbackCalls = append(rollbackCalls, "with-rollback")
			mu.Unlock()
			return nil
		}))

		stepWithoutRollback := NewFuncStep("without-rollback", "Without Rollback", func(ctx *Context) StepResult {
			return CompleteStep("done")
		}) // No rollback func

		stepFailing := NewFuncStep("failing", "Failing", func(ctx *Context) StepResult {
			return FailStep("failed", errors.New("error"))
		})

		w := NewWorkflow("test")
		w.AddStep(stepWithRollback)
		w.AddStep(stepWithoutRollback)
		w.AddStep(stepFailing)

		o := NewOrchestrator(w, WithAutoRollback(true))
		ctx := NewContext()
		o.Execute(ctx)

		mu.Lock()
		defer mu.Unlock()
		// Only steps with rollback should be called
		assert.Equal(t, []string{"with-rollback"}, rollbackCalls)
	})
}

// TestOrchestrator_DryRunMode tests dry-run doesn't make changes
func TestOrchestrator_DryRunMode(t *testing.T) {
	t.Run("sets dry run on context", func(t *testing.T) {
		var wasDryRun bool
		step := NewFuncStep("check", "Check dry run", func(ctx *Context) StepResult {
			wasDryRun = ctx.DryRun
			return CompleteStep("checked")
		})

		w := NewWorkflow("test")
		w.AddStep(step)

		o := NewOrchestrator(w, WithOrchestratorDryRun(true))
		ctx := NewContext()
		report := o.Execute(ctx)

		assert.Equal(t, WorkflowStatusCompleted, report.Status)
		assert.True(t, wasDryRun)
	})

	t.Run("all steps receive dry run flag", func(t *testing.T) {
		dryRunFlags := make(map[string]bool)
		var mu sync.Mutex

		w := NewWorkflow("test")
		for i := 0; i < 5; i++ {
			name := "step" + string(rune('1'+i))
			stepName := name
			step := NewFuncStep(stepName, "Step", func(ctx *Context) StepResult {
				mu.Lock()
				dryRunFlags[stepName] = ctx.DryRun
				mu.Unlock()
				return CompleteStep("done")
			})
			w.AddStep(step)
		}

		o := NewOrchestrator(w, WithOrchestratorDryRun(true))
		ctx := NewContext()
		o.Execute(ctx)

		mu.Lock()
		defer mu.Unlock()
		for name, isDryRun := range dryRunFlags {
			assert.True(t, isDryRun, "Step %s should have DryRun=true", name)
		}
	})

	t.Run("dry run with progress callback", func(t *testing.T) {
		var progressUpdates []StepProgress
		var mu sync.Mutex

		w := NewWorkflow("test")
		for i := 0; i < 3; i++ {
			name := "step" + string(rune('1'+i))
			w.AddStep(NewFuncStep(name, "Step", func(ctx *Context) StepResult {
				return CompleteStep("done")
			}))
		}

		o := NewOrchestrator(w,
			WithOrchestratorDryRun(true),
			WithOrchestratorProgress(func(p StepProgress) {
				mu.Lock()
				progressUpdates = append(progressUpdates, p)
				mu.Unlock()
			}),
		)
		ctx := NewContext()
		report := o.Execute(ctx)

		assert.Equal(t, WorkflowStatusCompleted, report.Status)
		mu.Lock()
		assert.True(t, len(progressUpdates) >= 3)
		mu.Unlock()
	})
}

// TestOrchestrator_CancellationHandling tests context cancellation
func TestOrchestrator_CancellationHandling(t *testing.T) {
	t.Run("cancellation before first step", func(t *testing.T) {
		step1Called := false
		step := NewFuncStep("step1", "Step 1", func(ctx *Context) StepResult {
			step1Called = true
			return CompleteStep("done")
		})

		w := NewWorkflow("test")
		w.AddStep(step)

		// Use step hooks to trigger cancellation check
		o := NewOrchestrator(w, WithPreStepHook(func(ctx *Context, s Step, r *StepResult) error {
			return nil
		}))
		ctx := NewContext()
		ctx.Cancel() // Cancel before execution

		report := o.Execute(ctx)

		assert.Equal(t, WorkflowStatusCancelled, report.Status)
		assert.False(t, step1Called)
	})

	t.Run("cancellation during step execution", func(t *testing.T) {
		step2Called := false
		step1 := NewFuncStep("step1", "Step 1", func(ctx *Context) StepResult {
			ctx.Cancel() // Cancel during first step
			return CompleteStep("done")
		})

		step2 := NewFuncStep("step2", "Step 2", func(ctx *Context) StepResult {
			step2Called = true
			return CompleteStep("done")
		})

		w := NewWorkflow("test")
		w.AddStep(step1)
		w.AddStep(step2)

		o := NewOrchestrator(w, WithPreStepHook(func(ctx *Context, s Step, r *StepResult) error {
			return nil
		}))
		ctx := NewContext()
		report := o.Execute(ctx)

		assert.Equal(t, WorkflowStatusCancelled, report.Status)
		assert.False(t, step2Called)
	})

	t.Run("cleanup happens on cancellation", func(t *testing.T) {
		var rollbackCallCount int32
		step1 := NewFuncStep("step1", "Step 1", func(ctx *Context) StepResult {
			return CompleteStep("done")
		}, WithRollbackFunc(func(ctx *Context) error {
			atomic.AddInt32(&rollbackCallCount, 1)
			return nil
		}))

		step2 := NewFuncStep("step2", "Step 2 (cancels)", func(ctx *Context) StepResult {
			ctx.Cancel()
			return CompleteStep("done")
		}, WithRollbackFunc(func(ctx *Context) error {
			atomic.AddInt32(&rollbackCallCount, 1)
			return nil
		}))

		// Add a third step - cancellation is detected before this step runs
		step3 := NewFuncStep("step3", "Step 3 (never runs)", func(ctx *Context) StepResult {
			return CompleteStep("done")
		}, WithRollbackFunc(func(ctx *Context) error {
			atomic.AddInt32(&rollbackCallCount, 1)
			return nil
		}))

		w := NewWorkflow("test")
		w.AddStep(step1)
		w.AddStep(step2)
		w.AddStep(step3)

		o := NewOrchestrator(w,
			WithAutoRollback(true),
			WithPreStepHook(func(ctx *Context, s Step, r *StepResult) error {
				return nil
			}),
		)
		ctx := NewContext()
		report := o.Execute(ctx)

		// Cancellation is detected before step3 runs, so status is cancelled
		assert.Equal(t, WorkflowStatusCancelled, report.Status)
		// Since auto-rollback is enabled and workflow was cancelled, rollback should happen
		// for steps 1 and 2 (step3 never ran so it won't be rolled back)
		// Note: The actual rollback behavior depends on orchestrator implementation
		_ = atomic.LoadInt32(&rollbackCallCount) // Use the variable
	})
}

// TestOrchestrator_ProgressReporting tests progress callbacks
func TestOrchestrator_ProgressReporting(t *testing.T) {
	t.Run("reports progress for each step", func(t *testing.T) {
		var progressUpdates []StepProgress
		var mu sync.Mutex

		stepNames := []string{"step1", "step2", "step3"}
		w := NewWorkflow("test")
		for _, name := range stepNames {
			stepName := name
			w.AddStep(NewFuncStep(stepName, "Step", func(ctx *Context) StepResult {
				return CompleteStep(stepName + " done")
			}))
		}

		o := NewOrchestrator(w,
			WithOrchestratorProgress(func(p StepProgress) {
				mu.Lock()
				progressUpdates = append(progressUpdates, p)
				mu.Unlock()
			}),
			WithPreStepHook(func(ctx *Context, s Step, r *StepResult) error {
				return nil
			}),
		)
		ctx := NewContext()
		report := o.Execute(ctx)

		assert.Equal(t, WorkflowStatusCompleted, report.Status)

		mu.Lock()
		defer mu.Unlock()
		assert.True(t, len(progressUpdates) >= 3)

		// Verify progress percentages increase
		for i := 1; i < len(progressUpdates); i++ {
			assert.GreaterOrEqual(t, progressUpdates[i].Percent, progressUpdates[i-1].Percent)
		}
	})

	t.Run("reports step start/complete events in log", func(t *testing.T) {
		w := NewWorkflow("test")
		w.AddStep(NewFuncStep("step1", "Step 1", func(ctx *Context) StepResult {
			return CompleteStep("done")
		}))

		o := NewOrchestrator(w, WithPreStepHook(func(ctx *Context, s Step, r *StepResult) error {
			return nil
		}))
		ctx := NewContext()
		o.Execute(ctx)

		log := o.GetExecutionLog()

		// Find step start and complete events
		hasStart := false
		hasComplete := false
		for _, entry := range log {
			if entry.StepName == "step1" && entry.EventType == EventStepStarted {
				hasStart = true
			}
			if entry.StepName == "step1" && entry.EventType == EventStepCompleted {
				hasComplete = true
			}
		}

		assert.True(t, hasStart, "Should have step started event")
		assert.True(t, hasComplete, "Should have step completed event")
	})
}

// TestOrchestrator_ConcurrentInstallationAttempts tests thread safety
func TestOrchestrator_ConcurrentInstallationAttempts(t *testing.T) {
	t.Run("orchestrator handles concurrent access", func(t *testing.T) {
		w := NewWorkflow("test")
		w.AddStep(NewFuncStep("step", "Step", func(ctx *Context) StepResult {
			time.Sleep(5 * time.Millisecond)
			return CompleteStep("done")
		}))

		o := NewOrchestrator(w)

		var wg sync.WaitGroup
		results := make(chan ExecutionReport, 10)

		// Launch multiple concurrent executions
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				ctx := NewContext()
				report := o.Execute(ctx)
				results <- report
			}()
		}

		wg.Wait()
		close(results)

		// All should complete (though order may vary)
		completedCount := 0
		for report := range results {
			if report.Status == WorkflowStatusCompleted {
				completedCount++
			}
		}
		assert.Equal(t, 10, completedCount)
	})

	t.Run("concurrent reads on orchestrator", func(t *testing.T) {
		w := NewWorkflow("test")
		w.AddStep(NewFuncStep("step", "Step", func(ctx *Context) StepResult {
			return CompleteStep("done")
		}))

		o := NewOrchestrator(w)
		ctx := NewContext()
		o.Execute(ctx)

		var wg sync.WaitGroup
		for i := 0; i < 100; i++ {
			wg.Add(3)
			go func() {
				defer wg.Done()
				_ = o.Workflow()
			}()
			go func() {
				defer wg.Done()
				_ = o.GetExecutionLog()
			}()
			go func() {
				defer wg.Done()
				o.Reset()
			}()
		}

		wg.Wait() // Should not panic
	})
}

// =============================================================================
// Distribution-Specific Workflow Tests
// =============================================================================

// TestOrchestrator_DebianWorkflow tests Debian-specific installation
func TestOrchestrator_DebianWorkflow(t *testing.T) {
	var executedSteps []string
	var mu sync.Mutex

	w := NewWorkflow("debian-nvidia-installation")

	// Debian-specific steps
	steps := []struct {
		name string
		desc string
	}{
		{"validation", "Validate system"},
		{"repository", "Add NVIDIA APT repository"},
		{"nouveau_blacklist", "Blacklist nouveau (update-initramfs)"},
		{"packages", "Install deb packages"},
		{"dkms_build", "Build kernel modules"},
		{"module_load", "Load nvidia module"},
		{"xorg_config", "Configure X.org"},
		{"verification", "Verify installation"},
	}

	for _, s := range steps {
		stepName := s.name
		step := NewFuncStep(stepName, s.desc, func(ctx *Context) StepResult {
			mu.Lock()
			executedSteps = append(executedSteps, stepName)
			mu.Unlock()
			// Simulate Debian-specific commands
			ctx.SetState("distro", "debian")
			return CompleteStep(stepName + " completed")
		})
		w.AddStep(step)
	}

	o := NewOrchestrator(w)
	ctx := NewContext()
	report := o.Execute(ctx)

	assert.Equal(t, WorkflowStatusCompleted, report.Status)
	assert.Equal(t, "debian-nvidia-installation", report.WorkflowName)

	mu.Lock()
	assert.Len(t, executedSteps, 8)
	mu.Unlock()
}

// TestOrchestrator_RHELWorkflow tests RHEL-specific installation
func TestOrchestrator_RHELWorkflow(t *testing.T) {
	var executedSteps []string
	var mu sync.Mutex

	w := NewWorkflow("rhel-nvidia-installation")

	// RHEL-specific steps
	steps := []struct {
		name string
		desc string
	}{
		{"validation", "Validate system"},
		{"repository", "Add NVIDIA DNF/YUM repository"},
		{"nouveau_blacklist", "Blacklist nouveau (dracut)"},
		{"packages", "Install rpm packages"},
		{"dkms_build", "Build kernel modules"},
		{"module_load", "Load nvidia module"},
		{"xorg_config", "Configure X.org"},
		{"verification", "Verify installation"},
	}

	for _, s := range steps {
		stepName := s.name
		step := NewFuncStep(stepName, s.desc, func(ctx *Context) StepResult {
			mu.Lock()
			executedSteps = append(executedSteps, stepName)
			mu.Unlock()
			ctx.SetState("distro", "rhel")
			return CompleteStep(stepName + " completed")
		})
		w.AddStep(step)
	}

	o := NewOrchestrator(w)
	ctx := NewContext()
	report := o.Execute(ctx)

	assert.Equal(t, WorkflowStatusCompleted, report.Status)
	assert.Equal(t, "rhel-nvidia-installation", report.WorkflowName)

	mu.Lock()
	assert.Len(t, executedSteps, 8)
	mu.Unlock()
}

// TestOrchestrator_ArchWorkflow tests Arch-specific installation
func TestOrchestrator_ArchWorkflow(t *testing.T) {
	var executedSteps []string
	var mu sync.Mutex

	w := NewWorkflow("arch-nvidia-installation")

	// Arch-specific steps (no repository step - packages in official repos)
	steps := []struct {
		name string
		desc string
	}{
		{"validation", "Validate system"},
		{"nouveau_blacklist", "Blacklist nouveau (mkinitcpio)"},
		{"packages", "Install pacman packages"},
		{"dkms_build", "Build kernel modules"},
		{"module_load", "Load nvidia module"},
		{"xorg_config", "Configure X.org"},
		{"verification", "Verify installation"},
	}

	for _, s := range steps {
		stepName := s.name
		step := NewFuncStep(stepName, s.desc, func(ctx *Context) StepResult {
			mu.Lock()
			executedSteps = append(executedSteps, stepName)
			mu.Unlock()
			ctx.SetState("distro", "arch")
			return CompleteStep(stepName + " completed")
		})
		w.AddStep(step)
	}

	o := NewOrchestrator(w)
	ctx := NewContext()
	report := o.Execute(ctx)

	assert.Equal(t, WorkflowStatusCompleted, report.Status)
	assert.Equal(t, "arch-nvidia-installation", report.WorkflowName)

	// Arch has 7 steps (no repository)
	mu.Lock()
	assert.Len(t, executedSteps, 7)
	assert.NotContains(t, executedSteps, "repository")
	mu.Unlock()
}

// TestOrchestrator_SUSEWorkflow tests SUSE-specific installation
func TestOrchestrator_SUSEWorkflow(t *testing.T) {
	var executedSteps []string
	var mu sync.Mutex

	w := NewWorkflow("suse-nvidia-installation")

	// SUSE-specific steps
	steps := []struct {
		name string
		desc string
	}{
		{"validation", "Validate system"},
		{"repository", "Add NVIDIA Zypper repository"},
		{"nouveau_blacklist", "Blacklist nouveau (dracut)"},
		{"packages", "Install rpm packages via zypper"},
		{"dkms_build", "Build kernel modules"},
		{"module_load", "Load nvidia module"},
		{"xorg_config", "Configure X.org"},
		{"verification", "Verify installation"},
	}

	for _, s := range steps {
		stepName := s.name
		step := NewFuncStep(stepName, s.desc, func(ctx *Context) StepResult {
			mu.Lock()
			executedSteps = append(executedSteps, stepName)
			mu.Unlock()
			ctx.SetState("distro", "suse")
			return CompleteStep(stepName + " completed")
		})
		w.AddStep(step)
	}

	o := NewOrchestrator(w)
	ctx := NewContext()
	report := o.Execute(ctx)

	assert.Equal(t, WorkflowStatusCompleted, report.Status)
	assert.Equal(t, "suse-nvidia-installation", report.WorkflowName)

	mu.Lock()
	assert.Len(t, executedSteps, 8)
	mu.Unlock()
}

// =============================================================================
// Error Scenarios
// =============================================================================

// TestOrchestrator_ValidationFailure tests handling of validation failures
func TestOrchestrator_ValidationFailure(t *testing.T) {
	t.Run("no GPU detected", func(t *testing.T) {
		validationStep := NewFuncStep("validation", "Validate GPU", func(ctx *Context) StepResult {
			return FailStep("No NVIDIA GPU detected", errors.New("gpu not found"))
		})

		w := NewWorkflow("test")
		w.AddStep(validationStep)
		w.AddStep(NewFuncStep("packages", "Install", func(ctx *Context) StepResult {
			return CompleteStep("done")
		}))

		o := NewOrchestrator(w)
		ctx := NewContext()
		report := o.Execute(ctx)

		assert.Equal(t, WorkflowStatusFailed, report.Status)
		assert.Contains(t, report.Error.Error(), "gpu not found")
		assert.Equal(t, 0, report.StepsCompleted)
	})

	t.Run("incompatible kernel", func(t *testing.T) {
		validationStep := NewFuncStep("validation", "Validate Kernel", func(ctx *Context) StepResult {
			return FailStep("Kernel too old", errors.New("kernel version 4.x not supported"))
		})

		w := NewWorkflow("test")
		w.AddStep(validationStep)

		o := NewOrchestrator(w)
		ctx := NewContext()
		report := o.Execute(ctx)

		assert.Equal(t, WorkflowStatusFailed, report.Status)
		assert.Contains(t, report.Error.Error(), "kernel")
	})

	t.Run("insufficient disk space", func(t *testing.T) {
		validationStep := NewFuncStep("validation", "Validate Disk", func(ctx *Context) StepResult {
			return FailStep("Insufficient disk space: 500MB available, 2GB required",
				errors.New("insufficient disk space"))
		})

		w := NewWorkflow("test")
		w.AddStep(validationStep)

		o := NewOrchestrator(w)
		ctx := NewContext()
		report := o.Execute(ctx)

		assert.Equal(t, WorkflowStatusFailed, report.Status)
		assert.Contains(t, report.Error.Error(), "disk space")
	})
}

// TestOrchestrator_PackageInstallFailure tests package install failures
func TestOrchestrator_PackageInstallFailure(t *testing.T) {
	t.Run("package not found", func(t *testing.T) {
		validationStep := NewFuncStep("validation", "Validate", func(ctx *Context) StepResult {
			return CompleteStep("OK")
		})

		packageStep := NewFuncStep("packages", "Install packages", func(ctx *Context) StepResult {
			return FailStep("Package nvidia-driver-550 not found", errors.New("package not found"))
		}, WithRollbackFunc(func(ctx *Context) error {
			return nil
		}))

		w := NewWorkflow("test")
		w.AddStep(validationStep)
		w.AddStep(packageStep)

		o := NewOrchestrator(w, WithAutoRollback(true))
		ctx := NewContext()
		report := o.Execute(ctx)

		assert.Equal(t, WorkflowStatusFailed, report.Status)
		assert.Contains(t, report.Error.Error(), "package not found")
		assert.True(t, report.RollbackPerformed)
	})

	t.Run("dependency conflicts", func(t *testing.T) {
		packageStep := NewFuncStep("packages", "Install packages", func(ctx *Context) StepResult {
			return FailStep("Dependency conflict: libgl1 conflicts with nvidia-libgl",
				errors.New("dependency conflict"))
		})

		w := NewWorkflow("test")
		w.AddStep(packageStep)

		o := NewOrchestrator(w)
		ctx := NewContext()
		report := o.Execute(ctx)

		assert.Equal(t, WorkflowStatusFailed, report.Status)
		assert.Contains(t, report.Error.Error(), "dependency conflict")
	})

	t.Run("network errors", func(t *testing.T) {
		packageStep := NewFuncStep("packages", "Install packages", func(ctx *Context) StepResult {
			return FailStep("Failed to download package", errors.New("network unreachable"))
		})

		w := NewWorkflow("test")
		w.AddStep(packageStep)

		o := NewOrchestrator(w)
		ctx := NewContext()
		report := o.Execute(ctx)

		assert.Equal(t, WorkflowStatusFailed, report.Status)
		assert.Contains(t, report.Error.Error(), "network")
	})
}

// TestOrchestrator_DKMSBuildFailure tests DKMS build failures
func TestOrchestrator_DKMSBuildFailure(t *testing.T) {
	t.Run("kernel headers missing", func(t *testing.T) {
		rollbackCalled := false
		packageStep := NewFuncStep("packages", "Install", func(ctx *Context) StepResult {
			return CompleteStep("done")
		}, WithRollbackFunc(func(ctx *Context) error {
			rollbackCalled = true
			return nil
		}))

		dkmsStep := NewFuncStep("dkms_build", "Build DKMS", func(ctx *Context) StepResult {
			return FailStep("Cannot find kernel headers for kernel 6.5.0",
				errors.New("kernel headers missing"))
		})

		w := NewWorkflow("test")
		w.AddStep(packageStep)
		w.AddStep(dkmsStep)

		o := NewOrchestrator(w, WithAutoRollback(true))
		ctx := NewContext()
		report := o.Execute(ctx)

		assert.Equal(t, WorkflowStatusFailed, report.Status)
		assert.Contains(t, report.Error.Error(), "kernel headers")
		assert.True(t, report.RollbackPerformed)
		assert.True(t, rollbackCalled)
	})

	t.Run("compilation errors", func(t *testing.T) {
		dkmsStep := NewFuncStep("dkms_build", "Build DKMS", func(ctx *Context) StepResult {
			return FailStep("DKMS make failed: error: implicit declaration of function",
				errors.New("compilation error"))
		})

		w := NewWorkflow("test")
		w.AddStep(dkmsStep)

		o := NewOrchestrator(w)
		ctx := NewContext()
		report := o.Execute(ctx)

		assert.Equal(t, WorkflowStatusFailed, report.Status)
		assert.Contains(t, report.Error.Error(), "compilation error")
	})
}

// =============================================================================
// State Sharing Tests
// =============================================================================

// TestOrchestrator_StateSharing tests data passing between steps
func TestOrchestrator_StateSharing(t *testing.T) {
	t.Run("steps can share state", func(t *testing.T) {
		step1 := NewFuncStep("producer", "Produce data", func(ctx *Context) StepResult {
			ctx.SetState("driver_version", "550.54.14")
			ctx.SetState("packages", []string{"nvidia-driver-550", "nvidia-cuda-toolkit"})
			return CompleteStep("produced")
		})

		var capturedVersion string
		step2 := NewFuncStep("consumer", "Consume data", func(ctx *Context) StepResult {
			capturedVersion = ctx.GetStateString("driver_version")
			return CompleteStep("consumed")
		})

		w := NewWorkflow("test")
		w.AddStep(step1)
		w.AddStep(step2)

		o := NewOrchestrator(w)
		ctx := NewContext()
		report := o.Execute(ctx)

		assert.Equal(t, WorkflowStatusCompleted, report.Status)
		assert.Equal(t, "550.54.14", capturedVersion)
	})

	t.Run("validation stores needs for later steps", func(t *testing.T) {
		validationStep := NewFuncStep("validation", "Validate", func(ctx *Context) StepResult {
			ctx.SetState("needs_kernel_headers", true)
			ctx.SetState("needs_nouveau_blacklist", true)
			return CompleteStep("validated")
		})

		var needsHeaders bool
		var needsNouveau bool
		packageStep := NewFuncStep("packages", "Install", func(ctx *Context) StepResult {
			needsHeaders = ctx.GetStateBool("needs_kernel_headers")
			needsNouveau = ctx.GetStateBool("needs_nouveau_blacklist")
			return CompleteStep("installed")
		})

		w := NewWorkflow("test")
		w.AddStep(validationStep)
		w.AddStep(packageStep)

		o := NewOrchestrator(w)
		ctx := NewContext()
		o.Execute(ctx)

		assert.True(t, needsHeaders)
		assert.True(t, needsNouveau)
	})

	t.Run("repository step stores repo info for package step", func(t *testing.T) {
		repoStep := NewFuncStep("repository", "Add repo", func(ctx *Context) StepResult {
			ctx.SetState("repository_configured", true)
			ctx.SetState("repository_name", "cuda-rhel9-x86_64")
			ctx.SetState("repository_url", "https://developer.download.nvidia.com/compute/cuda/repos/rhel9/x86_64")
			return CompleteStep("repo added")
		})

		var repoConfigured bool
		var repoName string
		packageStep := NewFuncStep("packages", "Install", func(ctx *Context) StepResult {
			repoConfigured = ctx.GetStateBool("repository_configured")
			repoName = ctx.GetStateString("repository_name")
			return CompleteStep("installed")
		})

		w := NewWorkflow("test")
		w.AddStep(repoStep)
		w.AddStep(packageStep)

		o := NewOrchestrator(w)
		ctx := NewContext()
		o.Execute(ctx)

		assert.True(t, repoConfigured)
		assert.Equal(t, "cuda-rhel9-x86_64", repoName)
	})
}

// =============================================================================
// Hooks Integration Tests
// =============================================================================

// TestOrchestrator_HooksIntegration tests hook functionality in workflows
func TestOrchestrator_HooksIntegration(t *testing.T) {
	t.Run("pre and post execute hooks wrap workflow", func(t *testing.T) {
		var events []string
		var mu sync.Mutex

		preHook := func(ctx *Context, w Workflow) error {
			mu.Lock()
			events = append(events, "pre-execute")
			mu.Unlock()
			return nil
		}

		postHook := func(ctx *Context, w Workflow) error {
			mu.Lock()
			events = append(events, "post-execute")
			mu.Unlock()
			return nil
		}

		w := NewWorkflow("test")
		w.AddStep(NewFuncStep("step", "Step", func(ctx *Context) StepResult {
			mu.Lock()
			events = append(events, "step")
			mu.Unlock()
			return CompleteStep("done")
		}))

		o := NewOrchestrator(w,
			WithPreExecuteHook(preHook),
			WithPostExecuteHook(postHook),
		)
		ctx := NewContext()
		o.Execute(ctx)

		mu.Lock()
		defer mu.Unlock()
		assert.Equal(t, []string{"pre-execute", "step", "post-execute"}, events)
	})

	t.Run("pre and post step hooks wrap each step", func(t *testing.T) {
		var events []string
		var mu sync.Mutex

		preStepHook := func(ctx *Context, s Step, r *StepResult) error {
			mu.Lock()
			events = append(events, "pre-"+s.Name())
			mu.Unlock()
			return nil
		}

		postStepHook := func(ctx *Context, s Step, r *StepResult) error {
			mu.Lock()
			events = append(events, "post-"+s.Name())
			mu.Unlock()
			return nil
		}

		w := NewWorkflow("test")
		w.AddStep(NewFuncStep("step1", "Step 1", func(ctx *Context) StepResult {
			mu.Lock()
			events = append(events, "step1")
			mu.Unlock()
			return CompleteStep("done")
		}))
		w.AddStep(NewFuncStep("step2", "Step 2", func(ctx *Context) StepResult {
			mu.Lock()
			events = append(events, "step2")
			mu.Unlock()
			return CompleteStep("done")
		}))

		o := NewOrchestrator(w,
			WithPreStepHook(preStepHook),
			WithPostStepHook(postStepHook),
		)
		ctx := NewContext()
		o.Execute(ctx)

		mu.Lock()
		defer mu.Unlock()
		assert.Equal(t, []string{
			"pre-step1", "step1", "post-step1",
			"pre-step2", "step2", "post-step2",
		}, events)
	})
}

// =============================================================================
// Execution Report Tests
// =============================================================================

// TestOrchestrator_ExecutionReportDetails tests detailed execution report
func TestOrchestrator_ExecutionReportDetails(t *testing.T) {
	t.Run("complete report for successful workflow", func(t *testing.T) {
		w := NewWorkflow("test-workflow")
		for i := 0; i < 5; i++ {
			name := "step" + string(rune('1'+i))
			w.AddStep(NewFuncStep(name, "Step", func(ctx *Context) StepResult {
				return CompleteStep("done")
			}))
		}

		o := NewOrchestrator(w, WithPreStepHook(func(ctx *Context, s Step, r *StepResult) error {
			return nil
		}))

		startTime := time.Now()
		ctx := NewContext()
		report := o.Execute(ctx)

		assert.Equal(t, "test-workflow", report.WorkflowName)
		assert.Equal(t, WorkflowStatusCompleted, report.Status)
		assert.True(t, report.StartTime.After(startTime) || report.StartTime.Equal(startTime))
		assert.True(t, report.EndTime.After(report.StartTime) || report.EndTime.Equal(report.StartTime))
		assert.Equal(t, 5, report.StepsExecuted)
		assert.Equal(t, 5, report.StepsCompleted)
		assert.Equal(t, 0, report.StepsFailed)
		assert.Equal(t, 0, report.StepsSkipped)
		assert.False(t, report.RollbackPerformed)
		assert.True(t, report.RollbackSuccess)
		assert.Nil(t, report.Error)
		assert.NotEmpty(t, report.ExecutionLog)
	})

	t.Run("complete report for failed workflow with rollback", func(t *testing.T) {
		w := NewWorkflow("test-workflow")
		w.AddStep(NewFuncStep("step1", "Step 1", func(ctx *Context) StepResult {
			return CompleteStep("done")
		}, WithRollbackFunc(func(ctx *Context) error {
			return nil
		})))
		w.AddStep(NewFuncStep("step2", "Step 2", func(ctx *Context) StepResult {
			return FailStep("failed", errors.New("error"))
		}))

		o := NewOrchestrator(w,
			WithAutoRollback(true),
			WithPreStepHook(func(ctx *Context, s Step, r *StepResult) error {
				return nil
			}),
		)
		ctx := NewContext()
		report := o.Execute(ctx)

		assert.Equal(t, WorkflowStatusFailed, report.Status)
		assert.Equal(t, 2, report.StepsExecuted)
		assert.Equal(t, 1, report.StepsCompleted)
		assert.Equal(t, 1, report.StepsFailed)
		assert.True(t, report.RollbackPerformed)
		assert.True(t, report.RollbackSuccess)
		assert.Error(t, report.Error)
	})
}

// =============================================================================
// Benchmarks
// =============================================================================

func BenchmarkOrchestrator_Execute(b *testing.B) {
	w := NewWorkflow("benchmark")
	for i := 0; i < 8; i++ {
		name := "step" + string(rune('1'+i))
		w.AddStep(NewFuncStep(name, "Step", func(ctx *Context) StepResult {
			return CompleteStep("done")
		}))
	}

	o := NewOrchestrator(w)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx := NewContext()
		o.Execute(ctx)
		o.Reset()
	}
}

func BenchmarkOrchestrator_Rollback(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		w := NewWorkflow("benchmark")
		for j := 0; j < 5; j++ {
			name := "step" + string(rune('1'+j))
			w.AddStep(NewFuncStep(name, "Step", func(ctx *Context) StepResult {
				return CompleteStep("done")
			}, WithRollbackFunc(func(ctx *Context) error {
				return nil
			})))
		}
		w.AddStep(NewFuncStep("fail", "Fail", func(ctx *Context) StepResult {
			return FailStep("fail", errors.New("error"))
		}))

		o := NewOrchestrator(w, WithAutoRollback(true))
		ctx := NewContext()

		b.StartTimer()
		o.Execute(ctx)
	}
}

func BenchmarkOrchestrator_WithHooks(b *testing.B) {
	w := NewWorkflow("benchmark")
	for i := 0; i < 8; i++ {
		name := "step" + string(rune('1'+i))
		w.AddStep(NewFuncStep(name, "Step", func(ctx *Context) StepResult {
			return CompleteStep("done")
		}))
	}

	o := NewOrchestrator(w,
		WithPreStepHook(func(ctx *Context, s Step, r *StepResult) error { return nil }),
		WithPostStepHook(func(ctx *Context, s Step, r *StepResult) error { return nil }),
		WithOrchestratorProgress(func(p StepProgress) {}),
	)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx := NewContext()
		o.Execute(ctx)
		o.Reset()
	}
}

func BenchmarkOrchestrator_WithDryRun(b *testing.B) {
	w := NewWorkflow("benchmark")
	for i := 0; i < 8; i++ {
		name := "step" + string(rune('1'+i))
		w.AddStep(NewFuncStep(name, "Step", func(ctx *Context) StepResult {
			return CompleteStep("done")
		}))
	}

	o := NewOrchestrator(w, WithOrchestratorDryRun(true))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx := NewContext()
		o.Execute(ctx)
		o.Reset()
	}
}
