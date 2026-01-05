package views

import (
	"errors"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// UninstallStep Tests
// =============================================================================

func TestUninstallStep_Duration(t *testing.T) {
	now := time.Now()
	step := UninstallStep{
		Name:      "test",
		StartTime: now,
		EndTime:   now.Add(5 * time.Second),
	}

	assert.Equal(t, 5*time.Second, step.Duration())
}

func TestUninstallStep_Duration_ZeroEndTime(t *testing.T) {
	step := UninstallStep{
		Name:      "test",
		StartTime: time.Now(),
	}

	assert.Equal(t, time.Duration(0), step.Duration())
}

func TestUninstallStep_Duration_ZeroStartTime(t *testing.T) {
	step := UninstallStep{
		Name:    "test",
		EndTime: time.Now(),
	}

	assert.Equal(t, time.Duration(0), step.Duration())
}

func TestUninstallStep_IsRunning(t *testing.T) {
	tests := []struct {
		status   StepStatus
		expected bool
	}{
		{StepPending, false},
		{StepRunning, true},
		{StepComplete, false},
		{StepFailed, false},
		{StepSkipped, false},
	}

	for _, tc := range tests {
		t.Run(tc.status.String(), func(t *testing.T) {
			step := UninstallStep{Status: tc.status}
			assert.Equal(t, tc.expected, step.IsRunning())
		})
	}
}

func TestUninstallStep_IsDone(t *testing.T) {
	tests := []struct {
		status   StepStatus
		expected bool
	}{
		{StepPending, false},
		{StepRunning, false},
		{StepComplete, true},
		{StepFailed, true},
		{StepSkipped, true},
	}

	for _, tc := range tests {
		t.Run(tc.status.String(), func(t *testing.T) {
			step := UninstallStep{Status: tc.status}
			assert.Equal(t, tc.expected, step.IsDone())
		})
	}
}

// =============================================================================
// UninstallProgressKeyMap Tests
// =============================================================================

func TestDefaultUninstallProgressKeyMap(t *testing.T) {
	km := DefaultUninstallProgressKeyMap()

	// Verify all key bindings are set
	assert.NotEmpty(t, km.Cancel.Keys())
	assert.NotEmpty(t, km.Quit.Keys())
	assert.NotEmpty(t, km.Help.Keys())
}

func TestUninstallProgressKeyMap_Cancel(t *testing.T) {
	km := DefaultUninstallProgressKeyMap()

	assert.Contains(t, km.Cancel.Keys(), "ctrl+c")
}

func TestUninstallProgressKeyMap_Quit(t *testing.T) {
	km := DefaultUninstallProgressKeyMap()

	assert.Contains(t, km.Quit.Keys(), "q")
}

func TestUninstallProgressKeyMap_Help(t *testing.T) {
	km := DefaultUninstallProgressKeyMap()

	assert.Contains(t, km.Help.Keys(), "?")
}

func TestUninstallProgressKeyMap_ShortHelp(t *testing.T) {
	km := DefaultUninstallProgressKeyMap()

	shortHelp := km.ShortHelp()

	assert.Len(t, shortHelp, 1)
	assert.Equal(t, km.Cancel, shortHelp[0])
}

func TestUninstallProgressKeyMap_FullHelp(t *testing.T) {
	km := DefaultUninstallProgressKeyMap()

	fullHelp := km.FullHelp()

	assert.Len(t, fullHelp, 1)
	assert.Len(t, fullHelp[0], 3)
	assert.Equal(t, km.Cancel, fullHelp[0][0])
	assert.Equal(t, km.Quit, fullHelp[0][1])
	assert.Equal(t, km.Help, fullHelp[0][2])
}

func TestUninstallProgressKeyMap_ImplementsHelpKeyMap(t *testing.T) {
	km := DefaultUninstallProgressKeyMap()

	// Should be able to call these methods as per help.KeyMap interface
	shortHelp := km.ShortHelp()
	fullHelp := km.FullHelp()

	assert.NotEmpty(t, shortHelp)
	assert.NotEmpty(t, fullHelp)

	// Verify types are correct
	for _, binding := range shortHelp {
		assert.NotEmpty(t, binding.Keys())
	}

	for _, row := range fullHelp {
		for _, binding := range row {
			assert.NotEmpty(t, binding.Keys())
		}
	}
}

func TestUninstallProgressKeyMap_BindingsHaveHelp(t *testing.T) {
	km := DefaultUninstallProgressKeyMap()

	bindings := []struct {
		name    string
		binding key.Binding
	}{
		{"Cancel", km.Cancel},
		{"Quit", km.Quit},
		{"Help", km.Help},
	}

	for _, b := range bindings {
		t.Run(b.name, func(t *testing.T) {
			help := b.binding.Help()
			assert.NotEmpty(t, help.Key, "binding should have key help")
			assert.NotEmpty(t, help.Desc, "binding should have description")
		})
	}
}

// =============================================================================
// buildDefaultUninstallSteps Tests
// =============================================================================

func TestBuildDefaultUninstallSteps(t *testing.T) {
	steps := buildDefaultUninstallSteps()

	// Should have default steps
	assert.Len(t, steps, 5)
	assert.Equal(t, "unload_modules", steps[0].Name)
	assert.Equal(t, "remove_packages", steps[1].Name)
	assert.Equal(t, "remove_configs", steps[2].Name)
	assert.Equal(t, "restore_nouveau", steps[3].Name)
	assert.Equal(t, "regenerate_initramfs", steps[4].Name)
}

func TestBuildDefaultUninstallSteps_AllStepsHaveDescriptions(t *testing.T) {
	steps := buildDefaultUninstallSteps()

	for _, step := range steps {
		assert.NotEmpty(t, step.Name, "step should have name")
		assert.NotEmpty(t, step.Description, "step should have description")
		assert.Equal(t, StepPending, step.Status, "step should start as pending")
	}
}

// =============================================================================
// NewUninstallProgress Tests
// =============================================================================

func TestNewUninstallProgress(t *testing.T) {
	styles := getTestStyles()
	version := "1.0.0"

	m := NewUninstallProgress(styles, version)

	assert.Equal(t, version, m.Version())
	assert.Equal(t, 0, m.Width())
	assert.Equal(t, 0, m.Height())
	assert.False(t, m.Ready())
	assert.False(t, m.IsComplete())
	assert.False(t, m.HasFailed())
	assert.False(t, m.IsCancelled())
	assert.Nil(t, m.FailureError())
	assert.Equal(t, 0, m.CurrentStep())
	assert.Greater(t, m.TotalSteps(), 0)
	assert.Equal(t, float64(0), m.Progress())
	assert.Equal(t, 10, m.MaxLogLines())
	assert.Empty(t, m.LogLines())
}

func TestNewUninstallProgress_WithCustomSteps(t *testing.T) {
	styles := getTestStyles()
	customSteps := []UninstallStep{
		{Name: "step1", Description: "Step 1"},
		{Name: "step2", Description: "Step 2"},
	}

	m := NewUninstallProgress(styles, "1.0.0",
		WithUninstallSteps(customSteps),
	)

	assert.Equal(t, 2, m.TotalSteps())
	assert.Equal(t, "step1", m.Steps()[0].Name)
	assert.Equal(t, "step2", m.Steps()[1].Name)
}

func TestNewUninstallProgress_KeyMapInitialized(t *testing.T) {
	styles := getTestStyles()

	m := NewUninstallProgress(styles, "1.0.0")
	km := m.KeyMap()

	assert.NotEmpty(t, km.Cancel.Keys())
	assert.NotEmpty(t, km.Quit.Keys())
	assert.NotEmpty(t, km.Help.Keys())
}

func TestNewUninstallProgress_DefaultSteps(t *testing.T) {
	styles := getTestStyles()

	m := NewUninstallProgress(styles, "1.0.0")

	steps := m.Steps()
	assert.Len(t, steps, 5)
	assert.Equal(t, m.TotalSteps(), len(steps))
}

// =============================================================================
// Init Tests
// =============================================================================

func TestUninstallProgressModel_Init(t *testing.T) {
	styles := getTestStyles()
	m := NewUninstallProgress(styles, "1.0.0")

	cmd := m.Init()

	assert.NotNil(t, cmd, "Init should return a command for spinner")
}

// =============================================================================
// Update Tests - WindowSizeMsg
// =============================================================================

func TestUninstallProgressModel_Update_WindowSizeMsg(t *testing.T) {
	styles := getTestStyles()
	m := NewUninstallProgress(styles, "1.0.0")

	assert.False(t, m.Ready())

	msg := tea.WindowSizeMsg{Width: 80, Height: 24}
	updated, cmd := m.Update(msg)

	assert.Equal(t, 80, updated.Width())
	assert.Equal(t, 24, updated.Height())
	assert.True(t, updated.Ready())
	assert.Nil(t, cmd)
}

func TestUninstallProgressModel_Update_WindowSizeMsg_Large(t *testing.T) {
	styles := getTestStyles()
	m := NewUninstallProgress(styles, "1.0.0")

	msg := tea.WindowSizeMsg{Width: 200, Height: 60}
	updated, _ := m.Update(msg)

	assert.Equal(t, 200, updated.Width())
	assert.Equal(t, 60, updated.Height())
	assert.True(t, updated.Ready())
}

func TestUninstallProgressModel_Update_WindowSizeMsg_Small(t *testing.T) {
	styles := getTestStyles()
	m := NewUninstallProgress(styles, "1.0.0")

	msg := tea.WindowSizeMsg{Width: 40, Height: 10}
	updated, _ := m.Update(msg)

	assert.Equal(t, 40, updated.Width())
	assert.Equal(t, 10, updated.Height())
	assert.True(t, updated.Ready())
}

// =============================================================================
// Update Tests - UninstallStepStartedMsg
// =============================================================================

func TestUninstallProgressModel_Update_UninstallStepStartedMsg(t *testing.T) {
	styles := getTestStyles()
	m := NewUninstallProgress(styles, "1.0.0")
	m.SetSize(80, 24)

	msg := UninstallStepStartedMsg{StepIndex: 0}
	updated, _ := m.Update(msg)

	assert.Equal(t, 0, updated.CurrentStep())
	assert.Equal(t, StepRunning, updated.Steps()[0].Status)
	assert.False(t, updated.Steps()[0].StartTime.IsZero())
	assert.Equal(t, float64(0), updated.Progress())
}

func TestUninstallProgressModel_Update_UninstallStepStartedMsg_SecondStep(t *testing.T) {
	styles := getTestStyles()
	m := NewUninstallProgress(styles, "1.0.0")
	m.SetSize(80, 24)

	msg := UninstallStepStartedMsg{StepIndex: 1}
	updated, _ := m.Update(msg)

	assert.Equal(t, 1, updated.CurrentStep())
	assert.Equal(t, StepRunning, updated.Steps()[1].Status)
	assert.Greater(t, updated.Progress(), float64(0))
}

func TestUninstallProgressModel_Update_UninstallStepStartedMsg_InvalidIndex(t *testing.T) {
	styles := getTestStyles()
	m := NewUninstallProgress(styles, "1.0.0")
	m.SetSize(80, 24)

	msg := UninstallStepStartedMsg{StepIndex: 100}
	updated, _ := m.Update(msg)

	// Should not crash, progress should be updated
	assert.NotNil(t, updated)
}

func TestUninstallProgressModel_Update_UninstallStepStartedMsg_NegativeIndex(t *testing.T) {
	styles := getTestStyles()
	m := NewUninstallProgress(styles, "1.0.0")
	m.SetSize(80, 24)

	msg := UninstallStepStartedMsg{StepIndex: -1}
	updated, _ := m.Update(msg)

	// Should not crash
	assert.NotNil(t, updated)
}

// =============================================================================
// Update Tests - UninstallStepCompletedMsg
// =============================================================================

func TestUninstallProgressModel_Update_UninstallStepCompletedMsg(t *testing.T) {
	styles := getTestStyles()
	m := NewUninstallProgress(styles, "1.0.0")
	m.SetSize(80, 24)

	// Start step first
	m, _ = m.Update(UninstallStepStartedMsg{StepIndex: 0})

	// Complete step
	msg := UninstallStepCompletedMsg{StepIndex: 0}
	updated, _ := m.Update(msg)

	assert.Equal(t, StepComplete, updated.Steps()[0].Status)
	assert.False(t, updated.Steps()[0].EndTime.IsZero())
}

func TestUninstallProgressModel_Update_UninstallStepCompletedMsg_WithError(t *testing.T) {
	styles := getTestStyles()
	m := NewUninstallProgress(styles, "1.0.0")
	m.SetSize(80, 24)

	// Start step first
	m, _ = m.Update(UninstallStepStartedMsg{StepIndex: 0})

	// Complete step with error
	testErr := errors.New("uninstall failed")
	msg := UninstallStepCompletedMsg{StepIndex: 0, Error: testErr}
	updated, _ := m.Update(msg)

	assert.Equal(t, StepFailed, updated.Steps()[0].Status)
	assert.True(t, updated.HasFailed())
	assert.Equal(t, testErr, updated.FailureError())
}

func TestUninstallProgressModel_Update_UninstallStepCompletedMsg_ProgressUpdates(t *testing.T) {
	styles := getTestStyles()
	m := NewUninstallProgress(styles, "1.0.0")
	m.SetSize(80, 24)

	totalSteps := m.TotalSteps()

	msg := UninstallStepCompletedMsg{StepIndex: 0}
	updated, _ := m.Update(msg)

	expectedProgress := float64(1) / float64(totalSteps)
	assert.Equal(t, expectedProgress, updated.Progress())
}

func TestUninstallProgressModel_Update_UninstallStepCompletedMsg_InvalidIndex(t *testing.T) {
	styles := getTestStyles()
	m := NewUninstallProgress(styles, "1.0.0")
	m.SetSize(80, 24)

	msg := UninstallStepCompletedMsg{StepIndex: 100}
	updated, _ := m.Update(msg)

	// Should not crash
	assert.NotNil(t, updated)
}

// =============================================================================
// Update Tests - UninstallLogMsg
// =============================================================================

func TestUninstallProgressModel_Update_UninstallLogMsg(t *testing.T) {
	styles := getTestStyles()
	m := NewUninstallProgress(styles, "1.0.0")
	m.SetSize(80, 24)

	msg := UninstallLogMsg{Message: "Removing package..."}
	updated, _ := m.Update(msg)

	assert.Len(t, updated.LogLines(), 1)
	assert.Equal(t, "Removing package...", updated.LogLines()[0])
}

func TestUninstallProgressModel_Update_UninstallLogMsg_Multiple(t *testing.T) {
	styles := getTestStyles()
	m := NewUninstallProgress(styles, "1.0.0")
	m.SetSize(80, 24)

	m, _ = m.Update(UninstallLogMsg{Message: "Line 1"})
	m, _ = m.Update(UninstallLogMsg{Message: "Line 2"})
	m, _ = m.Update(UninstallLogMsg{Message: "Line 3"})

	assert.Len(t, m.LogLines(), 3)
	assert.Equal(t, "Line 1", m.LogLines()[0])
	assert.Equal(t, "Line 2", m.LogLines()[1])
	assert.Equal(t, "Line 3", m.LogLines()[2])
}

func TestUninstallProgressModel_Update_UninstallLogMsg_TruncatesAtMax(t *testing.T) {
	styles := getTestStyles()
	m := NewUninstallProgress(styles, "1.0.0")
	m.SetSize(80, 24)
	m.SetMaxLogLines(3)

	m, _ = m.Update(UninstallLogMsg{Message: "Line 1"})
	m, _ = m.Update(UninstallLogMsg{Message: "Line 2"})
	m, _ = m.Update(UninstallLogMsg{Message: "Line 3"})
	m, _ = m.Update(UninstallLogMsg{Message: "Line 4"})
	m, _ = m.Update(UninstallLogMsg{Message: "Line 5"})

	assert.Len(t, m.LogLines(), 3)
	assert.Equal(t, "Line 3", m.LogLines()[0])
	assert.Equal(t, "Line 4", m.LogLines()[1])
	assert.Equal(t, "Line 5", m.LogLines()[2])
}

// =============================================================================
// Update Tests - UninstallCompleteMsg
// =============================================================================

func TestUninstallProgressModel_Update_UninstallCompleteMsg(t *testing.T) {
	styles := getTestStyles()
	m := NewUninstallProgress(styles, "1.0.0")
	m.SetSize(80, 24)

	msg := UninstallCompleteMsg{
		Success:         true,
		RemovedPackages: []string{"nvidia-driver-550"},
		CleanedConfigs:  []string{"/etc/modprobe.d/blacklist-nouveau.conf"},
		NouveauRestored: true,
		NeedsReboot:     true,
	}
	updated, _ := m.Update(msg)

	assert.True(t, updated.IsComplete())
	assert.Equal(t, float64(1), updated.Progress())
	assert.Equal(t, []string{"nvidia-driver-550"}, updated.RemovedPackages())
	assert.Equal(t, []string{"/etc/modprobe.d/blacklist-nouveau.conf"}, updated.CleanedConfigs())
	assert.True(t, updated.NouveauRestored())
	assert.True(t, updated.NeedsReboot())
}

func TestUninstallProgressModel_Update_UninstallCompleteMsg_WithError(t *testing.T) {
	styles := getTestStyles()
	m := NewUninstallProgress(styles, "1.0.0")
	m.SetSize(80, 24)

	testErr := errors.New("uninstall failed")
	msg := UninstallCompleteMsg{
		Success: false,
		Error:   testErr,
	}
	updated, _ := m.Update(msg)

	assert.True(t, updated.IsComplete())
	assert.True(t, updated.HasFailed())
	assert.Equal(t, testErr, updated.FailureError())
}

// =============================================================================
// Update Tests - Cancel Key
// =============================================================================

func TestUninstallProgressModel_Update_CancelKey_DuringUninstall(t *testing.T) {
	styles := getTestStyles()
	m := NewUninstallProgress(styles, "1.0.0")
	m.SetSize(80, 24)

	// Not complete, not failed
	assert.False(t, m.IsComplete())
	assert.False(t, m.HasFailed())

	msg := tea.KeyMsg{Type: tea.KeyCtrlC}
	updated, cmd := m.Update(msg)

	assert.True(t, updated.IsCancelled())
	require.NotNil(t, cmd)

	result := cmd()
	_, ok := result.(UninstallCancelledProgressMsg)
	assert.True(t, ok, "Expected UninstallCancelledProgressMsg")
}

func TestUninstallProgressModel_Update_CancelKey_WhenComplete(t *testing.T) {
	styles := getTestStyles()
	m := NewUninstallProgress(styles, "1.0.0")
	m.SetSize(80, 24)
	m.SetComplete(false)

	msg := tea.KeyMsg{Type: tea.KeyCtrlC}
	_, cmd := m.Update(msg)

	// Should not trigger cancel when complete
	assert.Nil(t, cmd)
}

func TestUninstallProgressModel_Update_CancelKey_WhenFailed(t *testing.T) {
	styles := getTestStyles()
	m := NewUninstallProgress(styles, "1.0.0")
	m.SetSize(80, 24)
	m.MarkStepFailed(0, errors.New("test error"))

	msg := tea.KeyMsg{Type: tea.KeyCtrlC}
	_, cmd := m.Update(msg)

	// Should not trigger cancel when failed
	assert.Nil(t, cmd)
}

// =============================================================================
// Update Tests - Quit Key
// =============================================================================

func TestUninstallProgressModel_Update_QuitKey_WhenComplete(t *testing.T) {
	styles := getTestStyles()
	m := NewUninstallProgress(styles, "1.0.0")
	m.SetSize(80, 24)
	m.SetComplete(true)
	m.SetRemovedPackages([]string{"nvidia-driver-550"})

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
	_, cmd := m.Update(msg)

	require.NotNil(t, cmd)

	result := cmd()
	navMsg, ok := result.(NavigateToUninstallCompleteMsg)
	assert.True(t, ok, "Expected NavigateToUninstallCompleteMsg")
	assert.Equal(t, []string{"nvidia-driver-550"}, navMsg.RemovedPackages)
	assert.True(t, navMsg.NeedsReboot)
}

func TestUninstallProgressModel_Update_QuitKey_WhenFailed(t *testing.T) {
	styles := getTestStyles()
	m := NewUninstallProgress(styles, "1.0.0")
	m.SetSize(80, 24)

	testErr := errors.New("test error")
	m.MarkStepFailed(0, testErr)

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
	_, cmd := m.Update(msg)

	require.NotNil(t, cmd)

	result := cmd()
	navMsg, ok := result.(NavigateToUninstallErrorMsg)
	assert.True(t, ok, "Expected NavigateToUninstallErrorMsg")
	assert.Equal(t, testErr, navMsg.Error)
	assert.NotEmpty(t, navMsg.FailedStep)
}

func TestUninstallProgressModel_Update_QuitKey_DuringUninstall(t *testing.T) {
	styles := getTestStyles()
	m := NewUninstallProgress(styles, "1.0.0")
	m.SetSize(80, 24)

	// Not complete, not failed
	assert.False(t, m.IsComplete())
	assert.False(t, m.HasFailed())

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
	_, cmd := m.Update(msg)

	// Should do nothing during uninstall
	assert.Nil(t, cmd)
}

// =============================================================================
// Update Tests - Help Key
// =============================================================================

func TestUninstallProgressModel_Update_HelpKey(t *testing.T) {
	styles := getTestStyles()
	m := NewUninstallProgress(styles, "1.0.0")
	m.SetSize(80, 24)

	assert.False(t, m.IsFullHelpShown())

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}}
	m, cmd := m.Update(msg)

	assert.True(t, m.IsFullHelpShown())
	assert.Nil(t, cmd)

	// Toggle off
	m, _ = m.Update(msg)
	assert.False(t, m.IsFullHelpShown())
}

// =============================================================================
// Update Tests - Spinner Tick
// =============================================================================

func TestUninstallProgressModel_Update_SpinnerTick(t *testing.T) {
	styles := getTestStyles()
	m := NewUninstallProgress(styles, "1.0.0")
	m.SetSize(80, 24)

	msg := spinner.TickMsg{}
	updated, cmd := m.Update(msg)

	// Should handle spinner tick without error
	assert.NotNil(t, updated)
	// May return a command for next tick
	_ = cmd
}

// =============================================================================
// View Tests - Not Ready
// =============================================================================

func TestUninstallProgressModel_View_NotReady(t *testing.T) {
	styles := getTestStyles()
	m := NewUninstallProgress(styles, "1.0.0")

	view := m.View()

	assert.Equal(t, "Loading...", view)
}

// =============================================================================
// View Tests - Ready
// =============================================================================

func TestUninstallProgressModel_View_Ready(t *testing.T) {
	styles := getTestStyles()
	m := NewUninstallProgress(styles, "1.0.0")
	m.SetSize(100, 40)

	view := m.View()

	assert.NotEmpty(t, view)
	assert.NotEqual(t, "Loading...", view)
}

func TestUninstallProgressModel_View_ShowsTitle(t *testing.T) {
	styles := getTestStyles()
	m := NewUninstallProgress(styles, "1.0.0")
	m.SetSize(100, 40)

	view := m.View()

	assert.Contains(t, view, "Uninstalling NVIDIA Drivers")
}

func TestUninstallProgressModel_View_ShowsProgressBar(t *testing.T) {
	styles := getTestStyles()
	m := NewUninstallProgress(styles, "1.0.0")
	m.SetSize(100, 40)

	view := m.View()

	assert.Contains(t, view, "Overall Progress:")
	assert.Contains(t, view, "%")
}

func TestUninstallProgressModel_View_ShowsSteps(t *testing.T) {
	styles := getTestStyles()
	m := NewUninstallProgress(styles, "1.0.0")
	m.SetSize(100, 40)

	view := m.View()

	// Should show step descriptions
	assert.Contains(t, view, "Unload kernel modules")
}

func TestUninstallProgressModel_View_ShowsStepMarkers(t *testing.T) {
	styles := getTestStyles()
	m := NewUninstallProgress(styles, "1.0.0")
	m.SetSize(100, 40)

	// Mark first step complete
	m.MarkStepComplete(0)

	view := m.View()

	// Should contain checkmark for completed step
	assert.Contains(t, view, "\u2713") // Checkmark
}

func TestUninstallProgressModel_View_ShowsPendingMarkers(t *testing.T) {
	styles := getTestStyles()
	m := NewUninstallProgress(styles, "1.0.0")
	m.SetSize(100, 40)

	view := m.View()

	// Should contain pending marker
	assert.Contains(t, view, "\u25CB") // Empty circle
}

func TestUninstallProgressModel_View_ShowsRunningMarker(t *testing.T) {
	styles := getTestStyles()
	m := NewUninstallProgress(styles, "1.0.0")
	m.SetSize(100, 40)

	// Start a step
	m, _ = m.Update(UninstallStepStartedMsg{StepIndex: 0})

	view := m.View()

	// Should contain running marker
	assert.Contains(t, view, "\u25CF") // Filled circle
}

func TestUninstallProgressModel_View_ShowsFailedMarker(t *testing.T) {
	styles := getTestStyles()
	m := NewUninstallProgress(styles, "1.0.0")
	m.SetSize(100, 40)

	// Fail a step
	m.MarkStepFailed(0, errors.New("test error"))

	view := m.View()

	// Should contain failed marker
	assert.Contains(t, view, "\u2717") // X mark
}

func TestUninstallProgressModel_View_ShowsLogOutput(t *testing.T) {
	styles := getTestStyles()
	m := NewUninstallProgress(styles, "1.0.0")
	m.SetSize(100, 40)

	m.AddLogLine("Removing package nvidia-driver-550...")

	view := m.View()

	assert.Contains(t, view, "Output:")
	assert.Contains(t, view, "Removing package nvidia-driver-550...")
}

func TestUninstallProgressModel_View_HidesLogOutputWhenEmpty(t *testing.T) {
	styles := getTestStyles()
	m := NewUninstallProgress(styles, "1.0.0")
	m.SetSize(100, 40)

	view := m.View()

	// Should not show Output: section when no logs
	assert.NotContains(t, view, "Output:")
}

func TestUninstallProgressModel_View_ShowsElapsedTime(t *testing.T) {
	styles := getTestStyles()
	m := NewUninstallProgress(styles, "1.0.0")
	m.SetSize(100, 40)

	view := m.View()

	assert.Contains(t, view, "Elapsed:")
}

// =============================================================================
// View Tests - Complete State
// =============================================================================

func TestUninstallProgressModel_View_Complete(t *testing.T) {
	styles := getTestStyles()
	m := NewUninstallProgress(styles, "1.0.0")
	m.SetSize(100, 40)
	m.SetComplete(false)

	view := m.View()

	assert.Contains(t, view, "Uninstall Complete!")
	assert.Contains(t, view, "Press 'q' to continue")
}

func TestUninstallProgressModel_View_Complete_NeedsReboot(t *testing.T) {
	styles := getTestStyles()
	m := NewUninstallProgress(styles, "1.0.0")
	m.SetSize(100, 40)
	m.SetComplete(true)

	view := m.View()

	assert.Contains(t, view, "Uninstall Complete!")
	assert.Contains(t, view, "reboot is required")
}

// =============================================================================
// View Tests - Failed State
// =============================================================================

func TestUninstallProgressModel_View_Failed(t *testing.T) {
	styles := getTestStyles()
	m := NewUninstallProgress(styles, "1.0.0")
	m.SetSize(100, 40)

	m.MarkStepFailed(0, errors.New("package removal failed"))

	view := m.View()

	assert.Contains(t, view, "Uninstall Failed")
	assert.Contains(t, view, "Error:")
	assert.Contains(t, view, "package removal failed")
	assert.Contains(t, view, "Press 'q' to view error details")
}

func TestUninstallProgressModel_View_Failed_NilError(t *testing.T) {
	styles := getTestStyles()
	m := NewUninstallProgress(styles, "1.0.0")
	m.SetSize(100, 40)

	m.MarkStepFailed(0, nil)

	view := m.View()

	assert.Contains(t, view, "Uninstall Failed")
	assert.Contains(t, view, "Unknown error")
}

// =============================================================================
// View Tests - Cancelled State
// =============================================================================

func TestUninstallProgressModel_View_Cancelled(t *testing.T) {
	styles := getTestStyles()
	m := NewUninstallProgress(styles, "1.0.0")
	m.SetSize(100, 40)

	// Cancel the uninstall
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})

	view := m.View()

	assert.Contains(t, view, "Uninstall Cancelled")
}

// =============================================================================
// View Tests - Various Sizes
// =============================================================================

func TestUninstallProgressModel_View_VariousSizes(t *testing.T) {
	styles := getTestStyles()

	testSizes := []struct {
		name   string
		width  int
		height int
	}{
		{"small", 40, 15},
		{"medium", 80, 24},
		{"large", 120, 40},
		{"wide", 200, 20},
		{"tall", 60, 50},
	}

	for _, tc := range testSizes {
		t.Run(tc.name, func(t *testing.T) {
			m := NewUninstallProgress(styles, "1.0.0")
			m.SetSize(tc.width, tc.height)

			view := m.View()

			assert.NotEmpty(t, view)
			assert.NotEqual(t, "Loading...", view)
		})
	}
}

func TestUninstallProgressModel_View_VerySmallSize(t *testing.T) {
	styles := getTestStyles()
	m := NewUninstallProgress(styles, "1.0.0")
	m.SetSize(10, 5)

	// Should not panic
	view := m.View()
	assert.NotEmpty(t, view)
}

// =============================================================================
// Getter Tests
// =============================================================================

func TestUninstallProgressModel_Getters(t *testing.T) {
	styles := getTestStyles()
	m := NewUninstallProgress(styles, "2.0.0")
	m.SetSize(100, 50)

	assert.Equal(t, 100, m.Width())
	assert.Equal(t, 50, m.Height())
	assert.True(t, m.Ready())
	assert.Equal(t, "2.0.0", m.Version())
	assert.NotNil(t, m.KeyMap())
	assert.Equal(t, 0, m.CurrentStep())
	assert.Greater(t, m.TotalSteps(), 0)
	assert.NotNil(t, m.Steps())
	assert.NotZero(t, m.StartTime())
}

// =============================================================================
// SetSize Tests
// =============================================================================

func TestUninstallProgressModel_SetSize(t *testing.T) {
	styles := getTestStyles()
	m := NewUninstallProgress(styles, "1.0.0")

	assert.False(t, m.Ready())

	m.SetSize(100, 50)

	assert.True(t, m.Ready())
	assert.Equal(t, 100, m.Width())
	assert.Equal(t, 50, m.Height())
}

func TestUninstallProgressModel_SetSize_Multiple(t *testing.T) {
	styles := getTestStyles()
	m := NewUninstallProgress(styles, "1.0.0")

	m.SetSize(80, 24)
	assert.Equal(t, 80, m.Width())
	assert.Equal(t, 24, m.Height())

	m.SetSize(120, 40)
	assert.Equal(t, 120, m.Width())
	assert.Equal(t, 40, m.Height())
}

// =============================================================================
// Log Line Management Tests
// =============================================================================

func TestUninstallProgressModel_SetMaxLogLines(t *testing.T) {
	styles := getTestStyles()
	m := NewUninstallProgress(styles, "1.0.0")

	m.SetMaxLogLines(5)
	assert.Equal(t, 5, m.MaxLogLines())
}

func TestUninstallProgressModel_SetMaxLogLines_Invalid(t *testing.T) {
	styles := getTestStyles()
	m := NewUninstallProgress(styles, "1.0.0")

	originalMax := m.MaxLogLines()
	m.SetMaxLogLines(0)
	assert.Equal(t, originalMax, m.MaxLogLines())

	m.SetMaxLogLines(-5)
	assert.Equal(t, originalMax, m.MaxLogLines())
}

func TestUninstallProgressModel_ClearLogLines(t *testing.T) {
	styles := getTestStyles()
	m := NewUninstallProgress(styles, "1.0.0")

	m.AddLogLine("Line 1")
	m.AddLogLine("Line 2")
	assert.Len(t, m.LogLines(), 2)

	m.ClearLogLines()
	assert.Empty(t, m.LogLines())
}

func TestUninstallProgressModel_AddLogLine(t *testing.T) {
	styles := getTestStyles()
	m := NewUninstallProgress(styles, "1.0.0")

	m.AddLogLine("Test line")
	assert.Len(t, m.LogLines(), 1)
	assert.Equal(t, "Test line", m.LogLines()[0])
}

// =============================================================================
// Step Management Tests
// =============================================================================

func TestUninstallProgressModel_MarkStepComplete(t *testing.T) {
	styles := getTestStyles()
	m := NewUninstallProgress(styles, "1.0.0")

	m.MarkStepComplete(0)

	steps := m.Steps()
	assert.Equal(t, StepComplete, steps[0].Status)
	assert.False(t, steps[0].EndTime.IsZero())
}

func TestUninstallProgressModel_MarkStepComplete_InvalidIndex(t *testing.T) {
	styles := getTestStyles()
	m := NewUninstallProgress(styles, "1.0.0")

	// Should not panic
	m.MarkStepComplete(-1)
	m.MarkStepComplete(100)
}

func TestUninstallProgressModel_MarkStepFailed(t *testing.T) {
	styles := getTestStyles()
	m := NewUninstallProgress(styles, "1.0.0")

	testErr := errors.New("test error")
	m.MarkStepFailed(0, testErr)

	steps := m.Steps()
	assert.Equal(t, StepFailed, steps[0].Status)
	assert.Equal(t, testErr, steps[0].Error)
	assert.True(t, m.HasFailed())
	assert.Equal(t, testErr, m.FailureError())
}

func TestUninstallProgressModel_MarkStepFailed_InvalidIndex(t *testing.T) {
	styles := getTestStyles()
	m := NewUninstallProgress(styles, "1.0.0")

	// Should not panic
	m.MarkStepFailed(-1, errors.New("error"))
	m.MarkStepFailed(100, errors.New("error"))
}

func TestUninstallProgressModel_SetComplete(t *testing.T) {
	styles := getTestStyles()
	m := NewUninstallProgress(styles, "1.0.0")

	m.SetComplete(true)

	assert.True(t, m.IsComplete())
	assert.Equal(t, float64(1), m.Progress())
	assert.True(t, m.NeedsReboot())
}

func TestUninstallProgressModel_SetComplete_NoReboot(t *testing.T) {
	styles := getTestStyles()
	m := NewUninstallProgress(styles, "1.0.0")

	m.SetComplete(false)

	assert.True(t, m.IsComplete())
	assert.False(t, m.NeedsReboot())
}

// =============================================================================
// Setter Tests
// =============================================================================

func TestUninstallProgressModel_SetRemovedPackages(t *testing.T) {
	styles := getTestStyles()
	m := NewUninstallProgress(styles, "1.0.0")

	packages := []string{"nvidia-driver-550", "nvidia-utils-550"}
	m.SetRemovedPackages(packages)

	assert.Equal(t, packages, m.RemovedPackages())
}

func TestUninstallProgressModel_SetCleanedConfigs(t *testing.T) {
	styles := getTestStyles()
	m := NewUninstallProgress(styles, "1.0.0")

	configs := []string{"/etc/modprobe.d/blacklist-nouveau.conf"}
	m.SetCleanedConfigs(configs)

	assert.Equal(t, configs, m.CleanedConfigs())
}

func TestUninstallProgressModel_SetNouveauRestored(t *testing.T) {
	styles := getTestStyles()
	m := NewUninstallProgress(styles, "1.0.0")

	m.SetNouveauRestored(true)

	assert.True(t, m.NouveauRestored())
}

func TestUninstallProgressModel_SetNeedsReboot(t *testing.T) {
	styles := getTestStyles()
	m := NewUninstallProgress(styles, "1.0.0")

	m.SetNeedsReboot(true)

	assert.True(t, m.NeedsReboot())
}

func TestUninstallProgressModel_SetStartTime(t *testing.T) {
	styles := getTestStyles()
	m := NewUninstallProgress(styles, "1.0.0")

	testTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	m.SetStartTime(testTime)

	assert.Equal(t, testTime, m.StartTime())
}

// =============================================================================
// Message Type Tests
// =============================================================================

func TestUninstallStepStartedMsg_Struct(t *testing.T) {
	msg := UninstallStepStartedMsg{StepIndex: 2, StepName: "test"}
	assert.Equal(t, 2, msg.StepIndex)
	assert.Equal(t, "test", msg.StepName)
}

func TestUninstallStepCompletedMsg_Struct(t *testing.T) {
	testErr := errors.New("test error")
	msg := UninstallStepCompletedMsg{StepIndex: 3, StepName: "test", Error: testErr}
	assert.Equal(t, 3, msg.StepIndex)
	assert.Equal(t, "test", msg.StepName)
	assert.Equal(t, testErr, msg.Error)
}

func TestUninstallLogMsg_Struct(t *testing.T) {
	msg := UninstallLogMsg{Message: "test message"}
	assert.Equal(t, "test message", msg.Message)
}

func TestUninstallCompleteMsg_Struct(t *testing.T) {
	packages := []string{"nvidia-driver-550"}
	configs := []string{"/etc/modprobe.d/blacklist-nouveau.conf"}

	msg := UninstallCompleteMsg{
		Success:         true,
		RemovedPackages: packages,
		CleanedConfigs:  configs,
		NouveauRestored: true,
		NeedsReboot:     true,
	}

	assert.True(t, msg.Success)
	assert.Equal(t, packages, msg.RemovedPackages)
	assert.Equal(t, configs, msg.CleanedConfigs)
	assert.True(t, msg.NouveauRestored)
	assert.True(t, msg.NeedsReboot)
}

func TestUninstallCancelledProgressMsg_Struct(t *testing.T) {
	msg := UninstallCancelledProgressMsg{}
	assert.NotNil(t, msg)
}

func TestNavigateToUninstallCompleteMsg_Struct(t *testing.T) {
	packages := []string{"nvidia-driver-550"}
	configs := []string{"/etc/modprobe.d/blacklist-nouveau.conf"}

	msg := NavigateToUninstallCompleteMsg{
		RemovedPackages: packages,
		CleanedConfigs:  configs,
		NouveauRestored: true,
		NeedsReboot:     true,
	}

	assert.Equal(t, packages, msg.RemovedPackages)
	assert.Equal(t, configs, msg.CleanedConfigs)
	assert.True(t, msg.NouveauRestored)
	assert.True(t, msg.NeedsReboot)
}

func TestNavigateToUninstallErrorMsg_Struct(t *testing.T) {
	testErr := errors.New("test error")
	msg := NavigateToUninstallErrorMsg{
		Error:      testErr,
		FailedStep: "Remove packages",
	}

	assert.Equal(t, testErr, msg.Error)
	assert.Equal(t, "Remove packages", msg.FailedStep)
}

// =============================================================================
// Integration Tests
// =============================================================================

func TestUninstallProgressModel_FullFlow_SuccessfulUninstall(t *testing.T) {
	styles := getTestStyles()
	m := NewUninstallProgress(styles, "1.0.0")

	// Window resize
	m, _ = m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	assert.True(t, m.Ready())

	// View should render properly
	view := m.View()
	assert.Contains(t, view, "Uninstalling NVIDIA Drivers")

	// Simulate uninstall steps
	totalSteps := m.TotalSteps()
	for i := 0; i < totalSteps; i++ {
		// Start step
		m, _ = m.Update(UninstallStepStartedMsg{StepIndex: i})
		assert.Equal(t, StepRunning, m.Steps()[i].Status)

		// Add log
		m, _ = m.Update(UninstallLogMsg{Message: "Processing step " + m.Steps()[i].Name})

		// Complete step
		m, _ = m.Update(UninstallStepCompletedMsg{StepIndex: i})
		assert.Equal(t, StepComplete, m.Steps()[i].Status)
	}

	// Mark complete
	m, _ = m.Update(UninstallCompleteMsg{
		Success:         true,
		RemovedPackages: []string{"nvidia-driver-550"},
		NouveauRestored: true,
		NeedsReboot:     true,
	})
	assert.True(t, m.IsComplete())
	assert.Equal(t, float64(1), m.Progress())

	// Press q to navigate
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	require.NotNil(t, cmd)

	result := cmd()
	_, ok := result.(NavigateToUninstallCompleteMsg)
	assert.True(t, ok)
}

func TestUninstallProgressModel_FullFlow_FailedUninstall(t *testing.T) {
	styles := getTestStyles()
	m := NewUninstallProgress(styles, "1.0.0")

	// Window resize
	m, _ = m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})

	// Start first step
	m, _ = m.Update(UninstallStepStartedMsg{StepIndex: 0})
	assert.Equal(t, StepRunning, m.Steps()[0].Status)

	// Fail the step
	testErr := errors.New("package not found")
	m, _ = m.Update(UninstallStepCompletedMsg{StepIndex: 0, Error: testErr})
	assert.True(t, m.HasFailed())
	assert.Equal(t, StepFailed, m.Steps()[0].Status)

	// View should show failure
	view := m.View()
	assert.Contains(t, view, "Uninstall Failed")
	assert.Contains(t, view, "package not found")

	// Press q to navigate to error
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	require.NotNil(t, cmd)

	result := cmd()
	navMsg, ok := result.(NavigateToUninstallErrorMsg)
	assert.True(t, ok)
	assert.Equal(t, testErr, navMsg.Error)
}

func TestUninstallProgressModel_FullFlow_CancelUninstall(t *testing.T) {
	styles := getTestStyles()
	m := NewUninstallProgress(styles, "1.0.0")
	m.SetSize(100, 40)

	// Start first step
	m, _ = m.Update(UninstallStepStartedMsg{StepIndex: 0})

	// Press Ctrl+C to cancel
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	require.NotNil(t, cmd)

	result := cmd()
	_, ok := result.(UninstallCancelledProgressMsg)
	assert.True(t, ok)
}

func TestUninstallProgressModel_FullFlow_HelpToggle(t *testing.T) {
	styles := getTestStyles()
	m := NewUninstallProgress(styles, "1.0.0")
	m.SetSize(100, 40)

	assert.False(t, m.IsFullHelpShown())

	// Toggle help on
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	assert.True(t, m.IsFullHelpShown())

	// View should still work
	view := m.View()
	assert.NotEmpty(t, view)

	// Toggle help off
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	assert.False(t, m.IsFullHelpShown())
}

// =============================================================================
// Edge Cases
// =============================================================================

func TestUninstallProgressModel_UnknownMessage(t *testing.T) {
	styles := getTestStyles()
	m := NewUninstallProgress(styles, "1.0.0")
	m.SetSize(80, 24)

	type customMsg struct{}

	updated, cmd := m.Update(customMsg{})

	// State should remain unchanged
	assert.True(t, updated.Ready())
	assert.False(t, updated.IsComplete())
	assert.Nil(t, cmd)
}

func TestUninstallProgressModel_MultipleSizeChanges(t *testing.T) {
	styles := getTestStyles()
	m := NewUninstallProgress(styles, "1.0.0")

	sizes := []tea.WindowSizeMsg{
		{Width: 80, Height: 24},
		{Width: 120, Height: 40},
		{Width: 40, Height: 15},
		{Width: 200, Height: 60},
		{Width: 80, Height: 24},
	}

	for _, size := range sizes {
		m, _ = m.Update(size)
		assert.Equal(t, size.Width, m.Width())
		assert.Equal(t, size.Height, m.Height())

		// View should still work
		view := m.View()
		assert.NotEmpty(t, view)
	}
}

func TestUninstallProgressModel_ManySteps(t *testing.T) {
	styles := getTestStyles()
	customSteps := []UninstallStep{
		{Name: "step1", Description: "Step 1"},
		{Name: "step2", Description: "Step 2"},
		{Name: "step3", Description: "Step 3"},
		{Name: "step4", Description: "Step 4"},
		{Name: "step5", Description: "Step 5"},
		{Name: "step6", Description: "Step 6"},
		{Name: "step7", Description: "Step 7"},
		{Name: "step8", Description: "Step 8"},
		{Name: "step9", Description: "Step 9"},
		{Name: "step10", Description: "Step 10"},
	}
	m := NewUninstallProgress(styles, "1.0.0",
		WithUninstallSteps(customSteps),
	)
	m.SetSize(100, 40)

	// Should have many steps
	assert.Equal(t, 10, m.TotalSteps())

	view := m.View()
	assert.NotEmpty(t, view)
	// Should show ellipsis for many steps
	assert.Contains(t, view, "...")
}

func TestUninstallProgressModel_StepTimingDisplay(t *testing.T) {
	styles := getTestStyles()
	m := NewUninstallProgress(styles, "1.0.0")
	m.SetSize(100, 40)

	// Start and complete a step with time difference
	m.steps[0].Status = StepComplete
	m.steps[0].StartTime = time.Now().Add(-2 * time.Second)
	m.steps[0].EndTime = time.Now()

	view := m.View()

	// Should show timing
	assert.Contains(t, view, "s)")
}

// =============================================================================
// Progress Calculation Tests
// =============================================================================

func TestUninstallProgressModel_ProgressCalculation(t *testing.T) {
	styles := getTestStyles()
	m := NewUninstallProgress(styles, "1.0.0")
	m.SetSize(80, 24)

	totalSteps := m.TotalSteps()

	// Initial progress is 0
	assert.Equal(t, float64(0), m.Progress())

	// Complete steps one by one and check progress
	for i := 0; i < totalSteps; i++ {
		m, _ = m.Update(UninstallStepCompletedMsg{StepIndex: i})
		expectedProgress := float64(i+1) / float64(totalSteps)
		assert.InDelta(t, expectedProgress, m.Progress(), 0.001)
	}

	// After all steps complete
	m, _ = m.Update(UninstallCompleteMsg{Success: true})
	assert.Equal(t, float64(1), m.Progress())
}

// =============================================================================
// Navigation Command Tests
// =============================================================================

func TestUninstallProgressModel_NavigateToError_WithFailedStep(t *testing.T) {
	styles := getTestStyles()
	m := NewUninstallProgress(styles, "1.0.0")
	m.SetSize(80, 24)

	// Start first step
	m, _ = m.Update(UninstallStepStartedMsg{StepIndex: 0})

	// Fail it
	testErr := errors.New("test error")
	m, _ = m.Update(UninstallStepCompletedMsg{StepIndex: 0, Error: testErr})

	// Navigate to error
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	require.NotNil(t, cmd)

	result := cmd()
	navMsg, ok := result.(NavigateToUninstallErrorMsg)
	assert.True(t, ok)
	assert.Equal(t, "Unload kernel modules", navMsg.FailedStep) // First step description
}

func TestUninstallProgressModel_NavigateToComplete_WithAllData(t *testing.T) {
	styles := getTestStyles()
	packages := []string{"nvidia-driver-550"}
	configs := []string{"/etc/modprobe.d/blacklist-nouveau.conf"}

	m := NewUninstallProgress(styles, "1.0.0")
	m.SetSize(80, 24)
	m.SetComplete(true)
	m.SetRemovedPackages(packages)
	m.SetCleanedConfigs(configs)
	m.SetNouveauRestored(true)

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	require.NotNil(t, cmd)

	result := cmd()
	navMsg, ok := result.(NavigateToUninstallCompleteMsg)
	assert.True(t, ok)
	assert.Equal(t, packages, navMsg.RemovedPackages)
	assert.Equal(t, configs, navMsg.CleanedConfigs)
	assert.True(t, navMsg.NouveauRestored)
	assert.True(t, navMsg.NeedsReboot)
}

// =============================================================================
// Additional Edge Case Tests for Coverage
// =============================================================================

func TestUninstallProgressModel_View_SkippedStepMarker(t *testing.T) {
	styles := getTestStyles()
	m := NewUninstallProgress(styles, "1.0.0")
	m.SetSize(100, 40)

	// Set a step to skipped status
	m.steps[0].Status = StepSkipped

	view := m.View()

	// Should contain skipped marker
	assert.Contains(t, view, "-")
}

func TestUninstallProgressModel_View_VeryNarrowWidth(t *testing.T) {
	styles := getTestStyles()
	m := NewUninstallProgress(styles, "1.0.0")
	m.SetSize(15, 40) // Very narrow - less than 20 for progress bar

	// Should not panic
	view := m.View()
	assert.NotEmpty(t, view)
}
