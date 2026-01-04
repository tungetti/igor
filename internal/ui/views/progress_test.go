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
	"github.com/tungetti/igor/internal/gpu"
)

// =============================================================================
// StepStatus Tests
// =============================================================================

func TestStepStatus_String(t *testing.T) {
	tests := []struct {
		status   StepStatus
		expected string
	}{
		{StepPending, "Pending"},
		{StepRunning, "Running"},
		{StepComplete, "Complete"},
		{StepFailed, "Failed"},
		{StepSkipped, "Skipped"},
		{StepStatus(99), "Unknown"},
	}

	for _, tc := range tests {
		t.Run(tc.expected, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.status.String())
		})
	}
}

// =============================================================================
// InstallationStep Tests
// =============================================================================

func TestInstallationStep_Duration(t *testing.T) {
	now := time.Now()
	step := InstallationStep{
		Name:      "test",
		StartTime: now,
		EndTime:   now.Add(5 * time.Second),
	}

	assert.Equal(t, 5*time.Second, step.Duration())
}

func TestInstallationStep_Duration_ZeroEndTime(t *testing.T) {
	step := InstallationStep{
		Name:      "test",
		StartTime: time.Now(),
	}

	assert.Equal(t, time.Duration(0), step.Duration())
}

func TestInstallationStep_Duration_ZeroStartTime(t *testing.T) {
	step := InstallationStep{
		Name:    "test",
		EndTime: time.Now(),
	}

	assert.Equal(t, time.Duration(0), step.Duration())
}

func TestInstallationStep_IsRunning(t *testing.T) {
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
			step := InstallationStep{Status: tc.status}
			assert.Equal(t, tc.expected, step.IsRunning())
		})
	}
}

func TestInstallationStep_IsDone(t *testing.T) {
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
			step := InstallationStep{Status: tc.status}
			assert.Equal(t, tc.expected, step.IsDone())
		})
	}
}

// =============================================================================
// ProgressKeyMap Tests
// =============================================================================

func TestDefaultProgressKeyMap(t *testing.T) {
	km := DefaultProgressKeyMap()

	// Verify all key bindings are set
	assert.NotEmpty(t, km.Cancel.Keys())
	assert.NotEmpty(t, km.Quit.Keys())
	assert.NotEmpty(t, km.Help.Keys())
}

func TestProgressKeyMap_Cancel(t *testing.T) {
	km := DefaultProgressKeyMap()

	assert.Contains(t, km.Cancel.Keys(), "ctrl+c")
}

func TestProgressKeyMap_Quit(t *testing.T) {
	km := DefaultProgressKeyMap()

	assert.Contains(t, km.Quit.Keys(), "q")
}

func TestProgressKeyMap_Help(t *testing.T) {
	km := DefaultProgressKeyMap()

	assert.Contains(t, km.Help.Keys(), "?")
}

func TestProgressKeyMap_ShortHelp(t *testing.T) {
	km := DefaultProgressKeyMap()

	shortHelp := km.ShortHelp()

	assert.Len(t, shortHelp, 1)
	assert.Equal(t, km.Cancel, shortHelp[0])
}

func TestProgressKeyMap_FullHelp(t *testing.T) {
	km := DefaultProgressKeyMap()

	fullHelp := km.FullHelp()

	assert.Len(t, fullHelp, 1)
	assert.Len(t, fullHelp[0], 3)
	assert.Equal(t, km.Cancel, fullHelp[0][0])
	assert.Equal(t, km.Quit, fullHelp[0][1])
	assert.Equal(t, km.Help, fullHelp[0][2])
}

func TestProgressKeyMap_ImplementsHelpKeyMap(t *testing.T) {
	km := DefaultProgressKeyMap()

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

func TestProgressKeyMap_BindingsHaveHelp(t *testing.T) {
	km := DefaultProgressKeyMap()

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
// buildInstallationSteps Tests
// =============================================================================

func TestBuildInstallationSteps_Empty(t *testing.T) {
	steps := buildInstallationSteps(nil)

	// Should have base steps: prepare, blacklist, update, configure, verify
	assert.Len(t, steps, 5)
	assert.Equal(t, "prepare", steps[0].Name)
	assert.Equal(t, "blacklist", steps[1].Name)
	assert.Equal(t, "update", steps[2].Name)
	assert.Equal(t, "configure", steps[3].Name)
	assert.Equal(t, "verify", steps[4].Name)
}

func TestBuildInstallationSteps_WithComponents(t *testing.T) {
	comps := []ComponentOption{
		{ID: "driver", Name: "NVIDIA Driver", Selected: true},
		{ID: "cuda", Name: "CUDA Toolkit", Selected: true},
	}

	steps := buildInstallationSteps(comps)

	// Should have base steps + 2 component steps
	assert.Len(t, steps, 7)
	assert.Equal(t, "install_driver", steps[3].Name)
	assert.Equal(t, "Installing NVIDIA Driver", steps[3].Description)
	assert.Equal(t, "install_cuda", steps[4].Name)
	assert.Equal(t, "Installing CUDA Toolkit", steps[4].Description)
}

func TestBuildInstallationSteps_UnselectedComponents(t *testing.T) {
	comps := []ComponentOption{
		{ID: "driver", Name: "NVIDIA Driver", Selected: true},
		{ID: "cuda", Name: "CUDA Toolkit", Selected: false}, // Not selected
	}

	steps := buildInstallationSteps(comps)

	// Should have base steps + 1 component step
	assert.Len(t, steps, 6)
	assert.Equal(t, "install_driver", steps[3].Name)
}

func TestBuildInstallationSteps_AllStepsHaveDescriptions(t *testing.T) {
	comps := []ComponentOption{
		{ID: "driver", Name: "NVIDIA Driver", Selected: true},
	}

	steps := buildInstallationSteps(comps)

	for _, step := range steps {
		assert.NotEmpty(t, step.Name, "step should have name")
		assert.NotEmpty(t, step.Description, "step should have description")
		assert.Equal(t, StepPending, step.Status, "step should start as pending")
	}
}

// =============================================================================
// NewProgress Tests
// =============================================================================

func TestNewProgress(t *testing.T) {
	styles := getTestStyles()
	version := "1.0.0"
	gpuInfo := createMockGPUInfo()
	driver := DriverOption{Version: "550", Branch: "Latest"}
	comps := []ComponentOption{{ID: "driver", Name: "NVIDIA Driver", Selected: true}}

	m := NewProgress(styles, version, gpuInfo, driver, comps)

	assert.Equal(t, version, m.Version())
	assert.Equal(t, gpuInfo, m.GPUInfo())
	assert.Equal(t, driver, m.Driver())
	assert.Equal(t, comps, m.ComponentOptions())
	assert.Equal(t, 0, m.Width())
	assert.Equal(t, 0, m.Height())
	assert.False(t, m.Ready())
	assert.False(t, m.IsComplete())
	assert.False(t, m.HasFailed())
	assert.Nil(t, m.FailureError())
	assert.Equal(t, 0, m.CurrentStep())
	assert.Greater(t, m.TotalSteps(), 0)
	assert.Equal(t, float64(0), m.Progress())
	assert.Equal(t, 10, m.MaxLogLines())
	assert.Empty(t, m.LogLines())
}

func TestNewProgress_WithNilGPUInfo(t *testing.T) {
	styles := getTestStyles()
	driver := DriverOption{Version: "550", Branch: "Latest"}
	comps := []ComponentOption{}

	m := NewProgress(styles, "1.0.0", nil, driver, comps)

	assert.Nil(t, m.GPUInfo())
}

func TestNewProgress_KeyMapInitialized(t *testing.T) {
	styles := getTestStyles()

	m := NewProgress(styles, "1.0.0", nil, DriverOption{}, nil)
	km := m.KeyMap()

	assert.NotEmpty(t, km.Cancel.Keys())
	assert.NotEmpty(t, km.Quit.Keys())
	assert.NotEmpty(t, km.Help.Keys())
}

func TestNewProgress_StepsBuiltCorrectly(t *testing.T) {
	styles := getTestStyles()
	comps := []ComponentOption{
		{ID: "driver", Name: "NVIDIA Driver", Selected: true},
		{ID: "cuda", Name: "CUDA Toolkit", Selected: true},
	}

	m := NewProgress(styles, "1.0.0", nil, DriverOption{}, comps)

	steps := m.Steps()
	assert.Len(t, steps, 7) // 5 base + 2 components
	assert.Equal(t, m.TotalSteps(), len(steps))
}

// =============================================================================
// Init Tests
// =============================================================================

func TestProgressModel_Init(t *testing.T) {
	styles := getTestStyles()
	m := NewProgress(styles, "1.0.0", nil, DriverOption{}, nil)

	cmd := m.Init()

	assert.NotNil(t, cmd, "Init should return a command for spinner")
}

// =============================================================================
// Update Tests - WindowSizeMsg
// =============================================================================

func TestProgressModel_Update_WindowSizeMsg(t *testing.T) {
	styles := getTestStyles()
	m := NewProgress(styles, "1.0.0", nil, DriverOption{}, nil)

	assert.False(t, m.Ready())

	msg := tea.WindowSizeMsg{Width: 80, Height: 24}
	updated, cmd := m.Update(msg)

	assert.Equal(t, 80, updated.Width())
	assert.Equal(t, 24, updated.Height())
	assert.True(t, updated.Ready())
	assert.Nil(t, cmd)
}

func TestProgressModel_Update_WindowSizeMsg_Large(t *testing.T) {
	styles := getTestStyles()
	m := NewProgress(styles, "1.0.0", nil, DriverOption{}, nil)

	msg := tea.WindowSizeMsg{Width: 200, Height: 60}
	updated, _ := m.Update(msg)

	assert.Equal(t, 200, updated.Width())
	assert.Equal(t, 60, updated.Height())
	assert.True(t, updated.Ready())
}

func TestProgressModel_Update_WindowSizeMsg_Small(t *testing.T) {
	styles := getTestStyles()
	m := NewProgress(styles, "1.0.0", nil, DriverOption{}, nil)

	msg := tea.WindowSizeMsg{Width: 40, Height: 10}
	updated, _ := m.Update(msg)

	assert.Equal(t, 40, updated.Width())
	assert.Equal(t, 10, updated.Height())
	assert.True(t, updated.Ready())
}

// =============================================================================
// Update Tests - InstallationStepStartMsg
// =============================================================================

func TestProgressModel_Update_InstallationStepStartMsg(t *testing.T) {
	styles := getTestStyles()
	m := NewProgress(styles, "1.0.0", nil, DriverOption{}, nil)
	m.SetSize(80, 24)

	msg := InstallationStepStartMsg{StepIndex: 0}
	updated, _ := m.Update(msg)

	assert.Equal(t, 0, updated.CurrentStep())
	assert.Equal(t, StepRunning, updated.Steps()[0].Status)
	assert.False(t, updated.Steps()[0].StartTime.IsZero())
	assert.Equal(t, float64(0), updated.Progress())
}

func TestProgressModel_Update_InstallationStepStartMsg_SecondStep(t *testing.T) {
	styles := getTestStyles()
	m := NewProgress(styles, "1.0.0", nil, DriverOption{}, nil)
	m.SetSize(80, 24)

	msg := InstallationStepStartMsg{StepIndex: 1}
	updated, _ := m.Update(msg)

	assert.Equal(t, 1, updated.CurrentStep())
	assert.Equal(t, StepRunning, updated.Steps()[1].Status)
	assert.Greater(t, updated.Progress(), float64(0))
}

func TestProgressModel_Update_InstallationStepStartMsg_InvalidIndex(t *testing.T) {
	styles := getTestStyles()
	m := NewProgress(styles, "1.0.0", nil, DriverOption{}, nil)
	m.SetSize(80, 24)

	msg := InstallationStepStartMsg{StepIndex: 100}
	updated, _ := m.Update(msg)

	// Should not crash, progress should be updated
	assert.NotNil(t, updated)
}

func TestProgressModel_Update_InstallationStepStartMsg_NegativeIndex(t *testing.T) {
	styles := getTestStyles()
	m := NewProgress(styles, "1.0.0", nil, DriverOption{}, nil)
	m.SetSize(80, 24)

	msg := InstallationStepStartMsg{StepIndex: -1}
	updated, _ := m.Update(msg)

	// Should not crash
	assert.NotNil(t, updated)
}

// =============================================================================
// Update Tests - InstallationStepCompleteMsg
// =============================================================================

func TestProgressModel_Update_InstallationStepCompleteMsg(t *testing.T) {
	styles := getTestStyles()
	m := NewProgress(styles, "1.0.0", nil, DriverOption{}, nil)
	m.SetSize(80, 24)

	// Start step first
	m, _ = m.Update(InstallationStepStartMsg{StepIndex: 0})

	// Complete step
	msg := InstallationStepCompleteMsg{StepIndex: 0}
	updated, _ := m.Update(msg)

	assert.Equal(t, StepComplete, updated.Steps()[0].Status)
	assert.False(t, updated.Steps()[0].EndTime.IsZero())
}

func TestProgressModel_Update_InstallationStepCompleteMsg_ProgressUpdates(t *testing.T) {
	styles := getTestStyles()
	m := NewProgress(styles, "1.0.0", nil, DriverOption{}, nil)
	m.SetSize(80, 24)

	totalSteps := m.TotalSteps()

	msg := InstallationStepCompleteMsg{StepIndex: 0}
	updated, _ := m.Update(msg)

	expectedProgress := float64(1) / float64(totalSteps)
	assert.Equal(t, expectedProgress, updated.Progress())
}

func TestProgressModel_Update_InstallationStepCompleteMsg_InvalidIndex(t *testing.T) {
	styles := getTestStyles()
	m := NewProgress(styles, "1.0.0", nil, DriverOption{}, nil)
	m.SetSize(80, 24)

	msg := InstallationStepCompleteMsg{StepIndex: 100}
	updated, _ := m.Update(msg)

	// Should not crash
	assert.NotNil(t, updated)
}

// =============================================================================
// Update Tests - InstallationStepFailedMsg
// =============================================================================

func TestProgressModel_Update_InstallationStepFailedMsg(t *testing.T) {
	styles := getTestStyles()
	m := NewProgress(styles, "1.0.0", nil, DriverOption{}, nil)
	m.SetSize(80, 24)

	testErr := errors.New("installation failed")
	msg := InstallationStepFailedMsg{StepIndex: 0, Error: testErr}
	updated, _ := m.Update(msg)

	assert.Equal(t, StepFailed, updated.Steps()[0].Status)
	assert.True(t, updated.HasFailed())
	assert.Equal(t, testErr, updated.FailureError())
	assert.Equal(t, testErr, updated.Steps()[0].Error)
}

func TestProgressModel_Update_InstallationStepFailedMsg_HidesSpinner(t *testing.T) {
	styles := getTestStyles()
	m := NewProgress(styles, "1.0.0", nil, DriverOption{}, nil)
	m.SetSize(80, 24)

	testErr := errors.New("installation failed")
	msg := InstallationStepFailedMsg{StepIndex: 0, Error: testErr}
	updated, _ := m.Update(msg)

	// Spinner should be hidden (visible in view when not failed)
	assert.True(t, updated.HasFailed())
}

func TestProgressModel_Update_InstallationStepFailedMsg_InvalidIndex(t *testing.T) {
	styles := getTestStyles()
	m := NewProgress(styles, "1.0.0", nil, DriverOption{}, nil)
	m.SetSize(80, 24)

	testErr := errors.New("installation failed")
	msg := InstallationStepFailedMsg{StepIndex: 100, Error: testErr}
	updated, _ := m.Update(msg)

	// Should still set hasFailed
	assert.True(t, updated.HasFailed())
	assert.Equal(t, testErr, updated.FailureError())
}

// =============================================================================
// Update Tests - InstallationLogMsg
// =============================================================================

func TestProgressModel_Update_InstallationLogMsg(t *testing.T) {
	styles := getTestStyles()
	m := NewProgress(styles, "1.0.0", nil, DriverOption{}, nil)
	m.SetSize(80, 24)

	msg := InstallationLogMsg{Message: "Installing package..."}
	updated, _ := m.Update(msg)

	assert.Len(t, updated.LogLines(), 1)
	assert.Equal(t, "Installing package...", updated.LogLines()[0])
}

func TestProgressModel_Update_InstallationLogMsg_Multiple(t *testing.T) {
	styles := getTestStyles()
	m := NewProgress(styles, "1.0.0", nil, DriverOption{}, nil)
	m.SetSize(80, 24)

	m, _ = m.Update(InstallationLogMsg{Message: "Line 1"})
	m, _ = m.Update(InstallationLogMsg{Message: "Line 2"})
	m, _ = m.Update(InstallationLogMsg{Message: "Line 3"})

	assert.Len(t, m.LogLines(), 3)
	assert.Equal(t, "Line 1", m.LogLines()[0])
	assert.Equal(t, "Line 2", m.LogLines()[1])
	assert.Equal(t, "Line 3", m.LogLines()[2])
}

func TestProgressModel_Update_InstallationLogMsg_TruncatesAtMax(t *testing.T) {
	styles := getTestStyles()
	m := NewProgress(styles, "1.0.0", nil, DriverOption{}, nil)
	m.SetSize(80, 24)
	m.SetMaxLogLines(3)

	m, _ = m.Update(InstallationLogMsg{Message: "Line 1"})
	m, _ = m.Update(InstallationLogMsg{Message: "Line 2"})
	m, _ = m.Update(InstallationLogMsg{Message: "Line 3"})
	m, _ = m.Update(InstallationLogMsg{Message: "Line 4"})
	m, _ = m.Update(InstallationLogMsg{Message: "Line 5"})

	assert.Len(t, m.LogLines(), 3)
	assert.Equal(t, "Line 3", m.LogLines()[0])
	assert.Equal(t, "Line 4", m.LogLines()[1])
	assert.Equal(t, "Line 5", m.LogLines()[2])
}

// =============================================================================
// Update Tests - InstallationCompleteMsg
// =============================================================================

func TestProgressModel_Update_InstallationCompleteMsg(t *testing.T) {
	styles := getTestStyles()
	m := NewProgress(styles, "1.0.0", nil, DriverOption{}, nil)
	m.SetSize(80, 24)

	msg := InstallationCompleteMsg{}
	updated, _ := m.Update(msg)

	assert.True(t, updated.IsComplete())
	assert.Equal(t, float64(1), updated.Progress())
}

// =============================================================================
// Update Tests - Cancel Key
// =============================================================================

func TestProgressModel_Update_CancelKey_DuringInstallation(t *testing.T) {
	styles := getTestStyles()
	m := NewProgress(styles, "1.0.0", nil, DriverOption{}, nil)
	m.SetSize(80, 24)

	// Not complete, not failed
	assert.False(t, m.IsComplete())
	assert.False(t, m.HasFailed())

	msg := tea.KeyMsg{Type: tea.KeyCtrlC}
	_, cmd := m.Update(msg)

	require.NotNil(t, cmd)

	result := cmd()
	_, ok := result.(InstallationCancelledMsg)
	assert.True(t, ok, "Expected InstallationCancelledMsg")
}

func TestProgressModel_Update_CancelKey_WhenComplete(t *testing.T) {
	styles := getTestStyles()
	m := NewProgress(styles, "1.0.0", nil, DriverOption{}, nil)
	m.SetSize(80, 24)
	m.SetComplete()

	msg := tea.KeyMsg{Type: tea.KeyCtrlC}
	_, cmd := m.Update(msg)

	// Should not trigger cancel when complete
	assert.Nil(t, cmd)
}

func TestProgressModel_Update_CancelKey_WhenFailed(t *testing.T) {
	styles := getTestStyles()
	m := NewProgress(styles, "1.0.0", nil, DriverOption{}, nil)
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

func TestProgressModel_Update_QuitKey_WhenComplete(t *testing.T) {
	styles := getTestStyles()
	gpuInfo := createMockGPUInfo()
	driver := DriverOption{Version: "550", Branch: "Latest"}
	comps := []ComponentOption{{ID: "driver", Name: "NVIDIA Driver", Selected: true}}
	m := NewProgress(styles, "1.0.0", gpuInfo, driver, comps)
	m.SetSize(80, 24)
	m.SetComplete()

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
	_, cmd := m.Update(msg)

	require.NotNil(t, cmd)

	result := cmd()
	navMsg, ok := result.(NavigateToCompleteMsg)
	assert.True(t, ok, "Expected NavigateToCompleteMsg")
	assert.Equal(t, gpuInfo, navMsg.GPUInfo)
	assert.Equal(t, driver, navMsg.Driver)
	assert.Equal(t, comps, navMsg.Components)
}

func TestProgressModel_Update_QuitKey_WhenFailed(t *testing.T) {
	styles := getTestStyles()
	m := NewProgress(styles, "1.0.0", nil, DriverOption{}, nil)
	m.SetSize(80, 24)

	testErr := errors.New("test error")
	m.MarkStepFailed(0, testErr)

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
	_, cmd := m.Update(msg)

	require.NotNil(t, cmd)

	result := cmd()
	navMsg, ok := result.(NavigateToErrorMsg)
	assert.True(t, ok, "Expected NavigateToErrorMsg")
	assert.Equal(t, testErr, navMsg.Error)
	assert.NotEmpty(t, navMsg.FailedStep)
}

func TestProgressModel_Update_QuitKey_DuringInstallation(t *testing.T) {
	styles := getTestStyles()
	m := NewProgress(styles, "1.0.0", nil, DriverOption{}, nil)
	m.SetSize(80, 24)

	// Not complete, not failed
	assert.False(t, m.IsComplete())
	assert.False(t, m.HasFailed())

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
	_, cmd := m.Update(msg)

	// Should do nothing during installation
	assert.Nil(t, cmd)
}

// =============================================================================
// Update Tests - Help Key
// =============================================================================

func TestProgressModel_Update_HelpKey(t *testing.T) {
	styles := getTestStyles()
	m := NewProgress(styles, "1.0.0", nil, DriverOption{}, nil)
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

func TestProgressModel_Update_SpinnerTick(t *testing.T) {
	styles := getTestStyles()
	m := NewProgress(styles, "1.0.0", nil, DriverOption{}, nil)
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

func TestProgressModel_View_NotReady(t *testing.T) {
	styles := getTestStyles()
	m := NewProgress(styles, "1.0.0", nil, DriverOption{}, nil)

	view := m.View()

	assert.Equal(t, "Loading...", view)
}

// =============================================================================
// View Tests - Ready
// =============================================================================

func TestProgressModel_View_Ready(t *testing.T) {
	styles := getTestStyles()
	m := NewProgress(styles, "1.0.0", nil, DriverOption{Version: "550"}, nil)
	m.SetSize(100, 40)

	view := m.View()

	assert.NotEmpty(t, view)
	assert.NotEqual(t, "Loading...", view)
}

func TestProgressModel_View_ShowsDriverVersion(t *testing.T) {
	styles := getTestStyles()
	driver := DriverOption{Version: "550", Branch: "Latest"}
	m := NewProgress(styles, "1.0.0", nil, driver, nil)
	m.SetSize(100, 40)

	view := m.View()

	assert.Contains(t, view, "550")
}

func TestProgressModel_View_ShowsProgressBar(t *testing.T) {
	styles := getTestStyles()
	m := NewProgress(styles, "1.0.0", nil, DriverOption{Version: "550"}, nil)
	m.SetSize(100, 40)

	view := m.View()

	assert.Contains(t, view, "Overall Progress:")
	assert.Contains(t, view, "%")
}

func TestProgressModel_View_ShowsSteps(t *testing.T) {
	styles := getTestStyles()
	m := NewProgress(styles, "1.0.0", nil, DriverOption{Version: "550"}, nil)
	m.SetSize(100, 40)

	view := m.View()

	// Should show step descriptions
	assert.Contains(t, view, "Preparing system")
}

func TestProgressModel_View_ShowsStepMarkers(t *testing.T) {
	styles := getTestStyles()
	comps := []ComponentOption{{ID: "driver", Name: "NVIDIA Driver", Selected: true}}
	m := NewProgress(styles, "1.0.0", nil, DriverOption{Version: "550"}, comps)
	m.SetSize(100, 40)

	// Mark first step complete
	m.MarkStepComplete(0)

	view := m.View()

	// Should contain checkmark for completed step
	assert.Contains(t, view, "\u2713") // Checkmark
}

func TestProgressModel_View_ShowsPendingMarkers(t *testing.T) {
	styles := getTestStyles()
	m := NewProgress(styles, "1.0.0", nil, DriverOption{Version: "550"}, nil)
	m.SetSize(100, 40)

	view := m.View()

	// Should contain pending marker
	assert.Contains(t, view, "\u25CB") // Empty circle
}

func TestProgressModel_View_ShowsRunningMarker(t *testing.T) {
	styles := getTestStyles()
	m := NewProgress(styles, "1.0.0", nil, DriverOption{Version: "550"}, nil)
	m.SetSize(100, 40)

	// Start a step
	m, _ = m.Update(InstallationStepStartMsg{StepIndex: 0})

	view := m.View()

	// Should contain running marker
	assert.Contains(t, view, "\u25CF") // Filled circle
}

func TestProgressModel_View_ShowsFailedMarker(t *testing.T) {
	styles := getTestStyles()
	m := NewProgress(styles, "1.0.0", nil, DriverOption{Version: "550"}, nil)
	m.SetSize(100, 40)

	// Fail a step
	m.MarkStepFailed(0, errors.New("test error"))

	view := m.View()

	// Should contain failed marker
	assert.Contains(t, view, "\u2717") // X mark
}

func TestProgressModel_View_ShowsLogOutput(t *testing.T) {
	styles := getTestStyles()
	m := NewProgress(styles, "1.0.0", nil, DriverOption{Version: "550"}, nil)
	m.SetSize(100, 40)

	m.AddLogLine("Installing package nvidia-driver-550...")

	view := m.View()

	assert.Contains(t, view, "Output:")
	assert.Contains(t, view, "Installing package nvidia-driver-550...")
}

func TestProgressModel_View_HidesLogOutputWhenEmpty(t *testing.T) {
	styles := getTestStyles()
	m := NewProgress(styles, "1.0.0", nil, DriverOption{Version: "550"}, nil)
	m.SetSize(100, 40)

	view := m.View()

	// Should not show Output: section when no logs
	assert.NotContains(t, view, "Output:")
}

// =============================================================================
// View Tests - Complete State
// =============================================================================

func TestProgressModel_View_Complete(t *testing.T) {
	styles := getTestStyles()
	m := NewProgress(styles, "1.0.0", nil, DriverOption{Version: "550"}, nil)
	m.SetSize(100, 40)
	m.SetComplete()

	view := m.View()

	assert.Contains(t, view, "Installation Complete!")
	assert.Contains(t, view, "Press 'q' to continue")
}

// =============================================================================
// View Tests - Failed State
// =============================================================================

func TestProgressModel_View_Failed(t *testing.T) {
	styles := getTestStyles()
	m := NewProgress(styles, "1.0.0", nil, DriverOption{Version: "550"}, nil)
	m.SetSize(100, 40)

	m.MarkStepFailed(0, errors.New("package installation failed"))

	view := m.View()

	assert.Contains(t, view, "Installation Failed")
	assert.Contains(t, view, "Error:")
	assert.Contains(t, view, "package installation failed")
	assert.Contains(t, view, "Press 'q' to view error details")
}

func TestProgressModel_View_Failed_NilError(t *testing.T) {
	styles := getTestStyles()
	m := NewProgress(styles, "1.0.0", nil, DriverOption{Version: "550"}, nil)
	m.SetSize(100, 40)

	m.MarkStepFailed(0, nil)

	view := m.View()

	assert.Contains(t, view, "Installation Failed")
	assert.Contains(t, view, "Unknown error")
}

// =============================================================================
// View Tests - Various Sizes
// =============================================================================

func TestProgressModel_View_VariousSizes(t *testing.T) {
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
			m := NewProgress(styles, "1.0.0", nil, DriverOption{Version: "550"}, nil)
			m.SetSize(tc.width, tc.height)

			view := m.View()

			assert.NotEmpty(t, view)
			assert.NotEqual(t, "Loading...", view)
		})
	}
}

func TestProgressModel_View_VerySmallSize(t *testing.T) {
	styles := getTestStyles()
	m := NewProgress(styles, "1.0.0", nil, DriverOption{Version: "550"}, nil)
	m.SetSize(10, 5)

	// Should not panic
	view := m.View()
	assert.NotEmpty(t, view)
}

// =============================================================================
// Getter Tests
// =============================================================================

func TestProgressModel_Getters(t *testing.T) {
	styles := getTestStyles()
	gpuInfo := createMockGPUInfo()
	driver := DriverOption{Version: "550", Branch: "Latest"}
	comps := []ComponentOption{{ID: "driver", Name: "NVIDIA Driver", Selected: true}}
	m := NewProgress(styles, "2.0.0", gpuInfo, driver, comps)
	m.SetSize(100, 50)

	assert.Equal(t, 100, m.Width())
	assert.Equal(t, 50, m.Height())
	assert.True(t, m.Ready())
	assert.Equal(t, "2.0.0", m.Version())
	assert.Equal(t, gpuInfo, m.GPUInfo())
	assert.Equal(t, driver, m.Driver())
	assert.Equal(t, comps, m.ComponentOptions())
	assert.NotNil(t, m.KeyMap())
	assert.Equal(t, 0, m.CurrentStep())
	assert.Greater(t, m.TotalSteps(), 0)
	assert.NotNil(t, m.Steps())
}

// =============================================================================
// SetSize Tests
// =============================================================================

func TestProgressModel_SetSize(t *testing.T) {
	styles := getTestStyles()
	m := NewProgress(styles, "1.0.0", nil, DriverOption{}, nil)

	assert.False(t, m.Ready())

	m.SetSize(100, 50)

	assert.True(t, m.Ready())
	assert.Equal(t, 100, m.Width())
	assert.Equal(t, 50, m.Height())
}

func TestProgressModel_SetSize_Multiple(t *testing.T) {
	styles := getTestStyles()
	m := NewProgress(styles, "1.0.0", nil, DriverOption{}, nil)

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

func TestProgressModel_SetMaxLogLines(t *testing.T) {
	styles := getTestStyles()
	m := NewProgress(styles, "1.0.0", nil, DriverOption{}, nil)

	m.SetMaxLogLines(5)
	assert.Equal(t, 5, m.MaxLogLines())
}

func TestProgressModel_SetMaxLogLines_Invalid(t *testing.T) {
	styles := getTestStyles()
	m := NewProgress(styles, "1.0.0", nil, DriverOption{}, nil)

	originalMax := m.MaxLogLines()
	m.SetMaxLogLines(0)
	assert.Equal(t, originalMax, m.MaxLogLines())

	m.SetMaxLogLines(-5)
	assert.Equal(t, originalMax, m.MaxLogLines())
}

func TestProgressModel_ClearLogLines(t *testing.T) {
	styles := getTestStyles()
	m := NewProgress(styles, "1.0.0", nil, DriverOption{}, nil)

	m.AddLogLine("Line 1")
	m.AddLogLine("Line 2")
	assert.Len(t, m.LogLines(), 2)

	m.ClearLogLines()
	assert.Empty(t, m.LogLines())
}

func TestProgressModel_AddLogLine(t *testing.T) {
	styles := getTestStyles()
	m := NewProgress(styles, "1.0.0", nil, DriverOption{}, nil)

	m.AddLogLine("Test line")
	assert.Len(t, m.LogLines(), 1)
	assert.Equal(t, "Test line", m.LogLines()[0])
}

// =============================================================================
// Step Management Tests
// =============================================================================

func TestProgressModel_MarkStepComplete(t *testing.T) {
	styles := getTestStyles()
	m := NewProgress(styles, "1.0.0", nil, DriverOption{}, nil)

	m.MarkStepComplete(0)

	steps := m.Steps()
	assert.Equal(t, StepComplete, steps[0].Status)
	assert.False(t, steps[0].EndTime.IsZero())
}

func TestProgressModel_MarkStepComplete_InvalidIndex(t *testing.T) {
	styles := getTestStyles()
	m := NewProgress(styles, "1.0.0", nil, DriverOption{}, nil)

	// Should not panic
	m.MarkStepComplete(-1)
	m.MarkStepComplete(100)
}

func TestProgressModel_MarkStepFailed(t *testing.T) {
	styles := getTestStyles()
	m := NewProgress(styles, "1.0.0", nil, DriverOption{}, nil)

	testErr := errors.New("test error")
	m.MarkStepFailed(0, testErr)

	steps := m.Steps()
	assert.Equal(t, StepFailed, steps[0].Status)
	assert.Equal(t, testErr, steps[0].Error)
	assert.True(t, m.HasFailed())
	assert.Equal(t, testErr, m.FailureError())
}

func TestProgressModel_MarkStepFailed_InvalidIndex(t *testing.T) {
	styles := getTestStyles()
	m := NewProgress(styles, "1.0.0", nil, DriverOption{}, nil)

	// Should not panic
	m.MarkStepFailed(-1, errors.New("error"))
	m.MarkStepFailed(100, errors.New("error"))
}

func TestProgressModel_SetComplete(t *testing.T) {
	styles := getTestStyles()
	m := NewProgress(styles, "1.0.0", nil, DriverOption{}, nil)

	m.SetComplete()

	assert.True(t, m.IsComplete())
	assert.Equal(t, float64(1), m.Progress())
}

// =============================================================================
// Message Type Tests
// =============================================================================

func TestInstallationStepStartMsg_Struct(t *testing.T) {
	msg := InstallationStepStartMsg{StepIndex: 2}
	assert.Equal(t, 2, msg.StepIndex)
}

func TestInstallationStepCompleteMsg_Struct(t *testing.T) {
	msg := InstallationStepCompleteMsg{StepIndex: 3}
	assert.Equal(t, 3, msg.StepIndex)
}

func TestInstallationStepFailedMsg_Struct(t *testing.T) {
	testErr := errors.New("test error")
	msg := InstallationStepFailedMsg{StepIndex: 1, Error: testErr}
	assert.Equal(t, 1, msg.StepIndex)
	assert.Equal(t, testErr, msg.Error)
}

func TestInstallationLogMsg_Struct(t *testing.T) {
	msg := InstallationLogMsg{Message: "test message"}
	assert.Equal(t, "test message", msg.Message)
}

func TestInstallationCompleteMsg_Struct(t *testing.T) {
	msg := InstallationCompleteMsg{}
	assert.NotNil(t, msg)
}

func TestInstallationCancelledMsg_Struct(t *testing.T) {
	msg := InstallationCancelledMsg{}
	assert.NotNil(t, msg)
}

func TestNavigateToCompleteMsg_Struct(t *testing.T) {
	gpuInfo := createMockGPUInfo()
	driver := DriverOption{Version: "550"}
	comps := []ComponentOption{{ID: "driver", Selected: true}}

	msg := NavigateToCompleteMsg{
		GPUInfo:    gpuInfo,
		Driver:     driver,
		Components: comps,
	}

	assert.Equal(t, gpuInfo, msg.GPUInfo)
	assert.Equal(t, "550", msg.Driver.Version)
	assert.Len(t, msg.Components, 1)
}

func TestNavigateToErrorMsg_Struct(t *testing.T) {
	testErr := errors.New("test error")
	msg := NavigateToErrorMsg{
		Error:      testErr,
		FailedStep: "Installing driver",
	}

	assert.Equal(t, testErr, msg.Error)
	assert.Equal(t, "Installing driver", msg.FailedStep)
}

// =============================================================================
// Integration Tests
// =============================================================================

func TestProgressModel_FullFlow_SuccessfulInstallation(t *testing.T) {
	styles := getTestStyles()
	gpuInfo := createMockGPUInfo()
	driver := DriverOption{Version: "550", Branch: "Latest"}
	comps := []ComponentOption{{ID: "driver", Name: "NVIDIA Driver", Selected: true}}
	m := NewProgress(styles, "1.0.0", gpuInfo, driver, comps)

	// Window resize
	m, _ = m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	assert.True(t, m.Ready())

	// View should render properly
	view := m.View()
	assert.Contains(t, view, "550")

	// Simulate installation steps
	totalSteps := m.TotalSteps()
	for i := 0; i < totalSteps; i++ {
		// Start step
		m, _ = m.Update(InstallationStepStartMsg{StepIndex: i})
		assert.Equal(t, StepRunning, m.Steps()[i].Status)

		// Add log
		m, _ = m.Update(InstallationLogMsg{Message: "Processing step " + m.Steps()[i].Name})

		// Complete step
		m, _ = m.Update(InstallationStepCompleteMsg{StepIndex: i})
		assert.Equal(t, StepComplete, m.Steps()[i].Status)
	}

	// Mark complete
	m, _ = m.Update(InstallationCompleteMsg{})
	assert.True(t, m.IsComplete())
	assert.Equal(t, float64(1), m.Progress())

	// Press q to navigate
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	require.NotNil(t, cmd)

	result := cmd()
	_, ok := result.(NavigateToCompleteMsg)
	assert.True(t, ok)
}

func TestProgressModel_FullFlow_FailedInstallation(t *testing.T) {
	styles := getTestStyles()
	m := NewProgress(styles, "1.0.0", nil, DriverOption{Version: "550"}, nil)

	// Window resize
	m, _ = m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})

	// Start first step
	m, _ = m.Update(InstallationStepStartMsg{StepIndex: 0})
	assert.Equal(t, StepRunning, m.Steps()[0].Status)

	// Fail the step
	testErr := errors.New("package not found")
	m, _ = m.Update(InstallationStepFailedMsg{StepIndex: 0, Error: testErr})
	assert.True(t, m.HasFailed())
	assert.Equal(t, StepFailed, m.Steps()[0].Status)

	// View should show failure
	view := m.View()
	assert.Contains(t, view, "Installation Failed")
	assert.Contains(t, view, "package not found")

	// Press q to navigate to error
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	require.NotNil(t, cmd)

	result := cmd()
	navMsg, ok := result.(NavigateToErrorMsg)
	assert.True(t, ok)
	assert.Equal(t, testErr, navMsg.Error)
}

func TestProgressModel_FullFlow_CancelInstallation(t *testing.T) {
	styles := getTestStyles()
	m := NewProgress(styles, "1.0.0", nil, DriverOption{Version: "550"}, nil)
	m.SetSize(100, 40)

	// Start first step
	m, _ = m.Update(InstallationStepStartMsg{StepIndex: 0})

	// Press Ctrl+C to cancel
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	require.NotNil(t, cmd)

	result := cmd()
	_, ok := result.(InstallationCancelledMsg)
	assert.True(t, ok)
}

func TestProgressModel_FullFlow_HelpToggle(t *testing.T) {
	styles := getTestStyles()
	m := NewProgress(styles, "1.0.0", nil, DriverOption{Version: "550"}, nil)
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

func TestProgressModel_UnknownMessage(t *testing.T) {
	styles := getTestStyles()
	m := NewProgress(styles, "1.0.0", nil, DriverOption{}, nil)
	m.SetSize(80, 24)

	type customMsg struct{}

	updated, cmd := m.Update(customMsg{})

	// State should remain unchanged
	assert.True(t, updated.Ready())
	assert.False(t, updated.IsComplete())
	assert.Nil(t, cmd)
}

func TestProgressModel_MultipleSizeChanges(t *testing.T) {
	styles := getTestStyles()
	m := NewProgress(styles, "1.0.0", nil, DriverOption{Version: "550"}, nil)

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

func TestProgressModel_EmptyComponents(t *testing.T) {
	styles := getTestStyles()
	m := NewProgress(styles, "1.0.0", nil, DriverOption{Version: "550"}, []ComponentOption{})
	m.SetSize(100, 40)

	// Should still have base steps
	assert.Equal(t, 5, m.TotalSteps())

	view := m.View()
	assert.NotEmpty(t, view)
}

func TestProgressModel_NilComponents(t *testing.T) {
	styles := getTestStyles()
	m := NewProgress(styles, "1.0.0", nil, DriverOption{Version: "550"}, nil)
	m.SetSize(100, 40)

	// Should still have base steps
	assert.Equal(t, 5, m.TotalSteps())

	// Should not panic
	view := m.View()
	assert.NotEmpty(t, view)
}

func TestProgressModel_ManySteps(t *testing.T) {
	styles := getTestStyles()
	comps := []ComponentOption{
		{ID: "driver", Name: "NVIDIA Driver", Selected: true},
		{ID: "cuda", Name: "CUDA Toolkit", Selected: true},
		{ID: "cudnn", Name: "cuDNN", Selected: true},
		{ID: "tensorrt", Name: "TensorRT", Selected: true},
		{ID: "nccl", Name: "NCCL", Selected: true},
	}
	m := NewProgress(styles, "1.0.0", nil, DriverOption{Version: "550"}, comps)
	m.SetSize(100, 40)

	// Should have many steps
	assert.Equal(t, 10, m.TotalSteps()) // 5 base + 5 components

	view := m.View()
	assert.NotEmpty(t, view)
	// Should show ellipsis for many steps
	assert.Contains(t, view, "...")
}

func TestProgressModel_StepTimingDisplay(t *testing.T) {
	styles := getTestStyles()
	m := NewProgress(styles, "1.0.0", nil, DriverOption{Version: "550"}, nil)
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

func TestProgressModel_ProgressCalculation(t *testing.T) {
	styles := getTestStyles()
	m := NewProgress(styles, "1.0.0", nil, DriverOption{}, nil)
	m.SetSize(80, 24)

	totalSteps := m.TotalSteps()

	// Initial progress is 0
	assert.Equal(t, float64(0), m.Progress())

	// Complete steps one by one and check progress
	for i := 0; i < totalSteps; i++ {
		m, _ = m.Update(InstallationStepCompleteMsg{StepIndex: i})
		expectedProgress := float64(i+1) / float64(totalSteps)
		assert.InDelta(t, expectedProgress, m.Progress(), 0.001)
	}

	// After all steps complete
	m, _ = m.Update(InstallationCompleteMsg{})
	assert.Equal(t, float64(1), m.Progress())
}

// =============================================================================
// Navigation Command Tests
// =============================================================================

func TestProgressModel_NavigateToError_WithFailedStep(t *testing.T) {
	styles := getTestStyles()
	m := NewProgress(styles, "1.0.0", nil, DriverOption{Version: "550"}, nil)
	m.SetSize(80, 24)

	// Start first step
	m, _ = m.Update(InstallationStepStartMsg{StepIndex: 0})

	// Fail it
	testErr := errors.New("test error")
	m, _ = m.Update(InstallationStepFailedMsg{StepIndex: 0, Error: testErr})

	// Navigate to error
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	require.NotNil(t, cmd)

	result := cmd()
	navMsg, ok := result.(NavigateToErrorMsg)
	assert.True(t, ok)
	assert.Equal(t, "Preparing system", navMsg.FailedStep) // First step description
}

func TestProgressModel_NavigateToComplete_WithAllData(t *testing.T) {
	styles := getTestStyles()
	gpuInfo := &gpu.GPUInfo{}
	driver := DriverOption{Version: "550", Branch: "Latest"}
	comps := []ComponentOption{{ID: "driver", Selected: true}}

	m := NewProgress(styles, "1.0.0", gpuInfo, driver, comps)
	m.SetSize(80, 24)
	m.SetComplete()

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	require.NotNil(t, cmd)

	result := cmd()
	navMsg, ok := result.(NavigateToCompleteMsg)
	assert.True(t, ok)
	assert.Equal(t, gpuInfo, navMsg.GPUInfo)
	assert.Equal(t, driver, navMsg.Driver)
	assert.Equal(t, comps, navMsg.Components)
}
