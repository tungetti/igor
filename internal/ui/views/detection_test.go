package views

import (
	"errors"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tungetti/igor/internal/gpu"
	"github.com/tungetti/igor/internal/gpu/kernel"
	"github.com/tungetti/igor/internal/gpu/nouveau"
	"github.com/tungetti/igor/internal/gpu/nvidia"
	"github.com/tungetti/igor/internal/gpu/pci"
	"github.com/tungetti/igor/internal/gpu/validator"
	"github.com/tungetti/igor/internal/ui/theme"
)

// =============================================================================
// Test Helpers
// =============================================================================

// createMockGPUInfo creates a mock GPUInfo for testing.
func createMockGPUInfo() *gpu.GPUInfo {
	return &gpu.GPUInfo{
		PCIDevices: []pci.PCIDevice{
			{
				Address:  "0000:01:00.0",
				VendorID: "10de",
				DeviceID: "2684",
				Class:    "0300",
			},
		},
		NVIDIAGPUs: []gpu.NVIDIAGPUInfo{
			{
				PCIDevice: pci.PCIDevice{
					Address:  "0000:01:00.0",
					VendorID: "10de",
					DeviceID: "2684",
					Class:    "0300",
				},
				Model: &nvidia.GPUModel{
					DeviceID:     "2684",
					Name:         "NVIDIA GeForce RTX 4090",
					Architecture: nvidia.ArchAdaLovelace,
				},
			},
		},
		InstalledDriver: &gpu.DriverInfo{
			Installed:   true,
			Type:        gpu.DriverTypeNVIDIA,
			Version:     "550.54.14",
			CUDAVersion: "12.4",
		},
		KernelInfo: &kernel.KernelInfo{
			Version:           "6.5.0-44-generic",
			Release:           "6.5.0",
			Architecture:      "x86_64",
			HeadersInstalled:  true,
			SecureBootEnabled: false,
		},
		NouveauStatus: &nouveau.Status{
			Loaded:          false,
			InUse:           false,
			BlacklistExists: true,
		},
		ValidationReport: &validator.ValidationReport{
			Passed:   true,
			Errors:   []validator.CheckResult{},
			Warnings: []validator.CheckResult{},
		},
		DetectionTime: time.Now(),
		Duration:      2 * time.Second,
	}
}

// createMockGPUInfoWithNoDriver creates GPUInfo without driver installed.
func createMockGPUInfoWithNoDriver() *gpu.GPUInfo {
	info := createMockGPUInfo()
	info.InstalledDriver = &gpu.DriverInfo{
		Installed: false,
		Type:      gpu.DriverTypeNone,
	}
	return info
}

// createMockGPUInfoWithNouveauLoaded creates GPUInfo with Nouveau loaded.
func createMockGPUInfoWithNouveauLoaded() *gpu.GPUInfo {
	info := createMockGPUInfo()
	info.NouveauStatus = &nouveau.Status{
		Loaded:          true,
		InUse:           true,
		BlacklistExists: false,
	}
	return info
}

// createMockGPUInfoWithValidationErrors creates GPUInfo with validation errors.
func createMockGPUInfoWithValidationErrors() *gpu.GPUInfo {
	info := createMockGPUInfo()
	info.ValidationReport = &validator.ValidationReport{
		Passed: false,
		Errors: []validator.CheckResult{
			{
				Name:     validator.CheckKernelHeaders,
				Passed:   false,
				Message:  "Kernel headers not installed",
				Severity: validator.SeverityError,
			},
		},
		Warnings: []validator.CheckResult{
			{
				Name:     validator.CheckSecureBoot,
				Passed:   false,
				Message:  "Secure Boot is enabled",
				Severity: validator.SeverityWarning,
			},
		},
	}
	return info
}

// createMockGPUInfoWithMultipleGPUs creates GPUInfo with multiple GPUs.
func createMockGPUInfoWithMultipleGPUs() *gpu.GPUInfo {
	info := createMockGPUInfo()
	info.NVIDIAGPUs = append(info.NVIDIAGPUs, gpu.NVIDIAGPUInfo{
		PCIDevice: pci.PCIDevice{
			Address:  "0000:02:00.0",
			VendorID: "10de",
			DeviceID: "2204",
			Class:    "0300",
		},
		Model: &nvidia.GPUModel{
			DeviceID:     "2204",
			Name:         "NVIDIA GeForce RTX 3090",
			Architecture: nvidia.ArchAmpere,
		},
	})
	return info
}

// =============================================================================
// DetectionState Tests
// =============================================================================

func TestDetectionState_String(t *testing.T) {
	tests := []struct {
		state    DetectionState
		expected string
	}{
		{StateDetecting, "Detecting"},
		{StateComplete, "Complete"},
		{StateError, "Error"},
		{DetectionState(99), "Unknown"},
	}

	for _, tc := range tests {
		t.Run(tc.expected, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.state.String())
		})
	}
}

// =============================================================================
// DetectionKeyMap Tests
// =============================================================================

func TestDefaultDetectionKeyMap(t *testing.T) {
	km := DefaultDetectionKeyMap()

	// Verify all key bindings are set
	assert.NotEmpty(t, km.Continue.Keys())
	assert.NotEmpty(t, km.Back.Keys())
	assert.NotEmpty(t, km.Quit.Keys())
	assert.NotEmpty(t, km.Help.Keys())
}

func TestDetectionKeyMap_Continue(t *testing.T) {
	km := DefaultDetectionKeyMap()

	// Continue should have enter and space
	assert.Contains(t, km.Continue.Keys(), "enter")
	assert.Contains(t, km.Continue.Keys(), " ")
}

func TestDetectionKeyMap_Back(t *testing.T) {
	km := DefaultDetectionKeyMap()

	// Back should have esc and backspace
	assert.Contains(t, km.Back.Keys(), "esc")
	assert.Contains(t, km.Back.Keys(), "backspace")
}

func TestDetectionKeyMap_Quit(t *testing.T) {
	km := DefaultDetectionKeyMap()

	// Quit should have q and ctrl+c
	assert.Contains(t, km.Quit.Keys(), "q")
	assert.Contains(t, km.Quit.Keys(), "ctrl+c")
}

func TestDetectionKeyMap_Help(t *testing.T) {
	km := DefaultDetectionKeyMap()

	assert.Contains(t, km.Help.Keys(), "?")
}

func TestDetectionKeyMap_ShortHelp(t *testing.T) {
	km := DefaultDetectionKeyMap()

	shortHelp := km.ShortHelp()

	assert.Len(t, shortHelp, 3)
	assert.Equal(t, km.Continue, shortHelp[0])
	assert.Equal(t, km.Back, shortHelp[1])
	assert.Equal(t, km.Quit, shortHelp[2])
}

func TestDetectionKeyMap_FullHelp(t *testing.T) {
	km := DefaultDetectionKeyMap()

	fullHelp := km.FullHelp()

	assert.Len(t, fullHelp, 2)

	// First row: Continue and Back
	assert.Len(t, fullHelp[0], 2)
	assert.Equal(t, km.Continue, fullHelp[0][0])
	assert.Equal(t, km.Back, fullHelp[0][1])

	// Second row: Quit and Help
	assert.Len(t, fullHelp[1], 2)
	assert.Equal(t, km.Quit, fullHelp[1][0])
	assert.Equal(t, km.Help, fullHelp[1][1])
}

// =============================================================================
// NewDetection Tests
// =============================================================================

func TestNewDetection(t *testing.T) {
	styles := getTestStyles()
	version := "1.0.0"

	m := NewDetection(styles, version)

	assert.Equal(t, version, m.Version())
	assert.Equal(t, 0, m.Width())
	assert.Equal(t, 0, m.Height())
	assert.False(t, m.Ready())
	assert.Equal(t, StateDetecting, m.State())
	assert.Nil(t, m.GPUInfo())
	assert.Nil(t, m.Error())
	assert.Equal(t, 0, m.StepIndex())
	assert.NotEmpty(t, m.Steps())
	assert.NotEmpty(t, m.CurrentStep())
}

func TestNewDetection_WithEmptyVersion(t *testing.T) {
	styles := getTestStyles()

	m := NewDetection(styles, "")

	assert.Equal(t, "", m.Version())
}

func TestNewDetection_StepsInitialized(t *testing.T) {
	styles := getTestStyles()

	m := NewDetection(styles, "1.0.0")

	steps := m.Steps()
	assert.NotEmpty(t, steps)
	assert.Equal(t, steps[0], m.CurrentStep())
}

func TestNewDetection_KeyMapInitialized(t *testing.T) {
	styles := getTestStyles()

	m := NewDetection(styles, "1.0.0")
	km := m.KeyMap()

	assert.NotEmpty(t, km.Continue.Keys())
	assert.NotEmpty(t, km.Back.Keys())
	assert.NotEmpty(t, km.Quit.Keys())
	assert.NotEmpty(t, km.Help.Keys())
}

// =============================================================================
// Init Tests
// =============================================================================

func TestDetectionModel_Init(t *testing.T) {
	styles := getTestStyles()
	m := NewDetection(styles, "1.0.0")

	cmd := m.Init()

	assert.NotNil(t, cmd, "Init should return spinner tick command")
}

// =============================================================================
// Update Tests - WindowSizeMsg
// =============================================================================

func TestDetectionModel_Update_WindowSizeMsg(t *testing.T) {
	styles := getTestStyles()
	m := NewDetection(styles, "1.0.0")

	assert.False(t, m.Ready())

	msg := tea.WindowSizeMsg{Width: 80, Height: 24}
	updated, cmd := m.Update(msg)

	assert.Equal(t, 80, updated.Width())
	assert.Equal(t, 24, updated.Height())
	assert.True(t, updated.Ready())
	assert.Nil(t, cmd)
}

func TestDetectionModel_Update_WindowSizeMsg_Large(t *testing.T) {
	styles := getTestStyles()
	m := NewDetection(styles, "1.0.0")

	msg := tea.WindowSizeMsg{Width: 200, Height: 60}
	updated, _ := m.Update(msg)

	assert.Equal(t, 200, updated.Width())
	assert.Equal(t, 60, updated.Height())
	assert.True(t, updated.Ready())
}

func TestDetectionModel_Update_WindowSizeMsg_Small(t *testing.T) {
	styles := getTestStyles()
	m := NewDetection(styles, "1.0.0")

	msg := tea.WindowSizeMsg{Width: 40, Height: 10}
	updated, _ := m.Update(msg)

	assert.Equal(t, 40, updated.Width())
	assert.Equal(t, 10, updated.Height())
	assert.True(t, updated.Ready())
}

// =============================================================================
// Update Tests - Spinner Ticks
// =============================================================================

func TestDetectionModel_Update_SpinnerTick(t *testing.T) {
	styles := getTestStyles()
	m := NewDetection(styles, "1.0.0")
	m.SetSize(80, 24)

	// Create a spinner tick message
	msg := spinner.TickMsg{
		ID:   0,
		Time: time.Now(),
	}

	// Update with spinner tick
	updated, cmd := m.Update(msg)

	// Should still be in detecting state
	assert.Equal(t, StateDetecting, updated.State())

	// Should return a command for next tick
	assert.NotNil(t, cmd)
}

// =============================================================================
// Update Tests - DetectionStepMsg
// =============================================================================

func TestDetectionModel_Update_DetectionStepMsg(t *testing.T) {
	styles := getTestStyles()
	m := NewDetection(styles, "1.0.0")
	m.SetSize(80, 24)

	steps := m.Steps()

	msg := DetectionStepMsg{Step: 2}
	updated, cmd := m.Update(msg)

	assert.Equal(t, 2, updated.StepIndex())
	assert.Equal(t, steps[2], updated.CurrentStep())
	assert.Nil(t, cmd)
}

func TestDetectionModel_Update_DetectionStepMsg_AllSteps(t *testing.T) {
	styles := getTestStyles()
	m := NewDetection(styles, "1.0.0")
	m.SetSize(80, 24)

	steps := m.Steps()

	for i := range steps {
		msg := DetectionStepMsg{Step: i}
		m, _ = m.Update(msg)

		assert.Equal(t, i, m.StepIndex())
		assert.Equal(t, steps[i], m.CurrentStep())
	}
}

func TestDetectionModel_Update_DetectionStepMsg_OutOfBounds(t *testing.T) {
	styles := getTestStyles()
	m := NewDetection(styles, "1.0.0")
	m.SetSize(80, 24)

	// Step beyond bounds
	msg := DetectionStepMsg{Step: 100}
	updated, _ := m.Update(msg)

	// Step index is updated but current step should stay as is
	assert.Equal(t, 100, updated.StepIndex())
}

// =============================================================================
// Update Tests - DetectionCompleteMsg
// =============================================================================

func TestDetectionModel_Update_DetectionCompleteMsg(t *testing.T) {
	styles := getTestStyles()
	m := NewDetection(styles, "1.0.0")
	m.SetSize(80, 24)

	gpuInfo := createMockGPUInfo()

	msg := DetectionCompleteMsg{GPUInfo: gpuInfo}
	updated, cmd := m.Update(msg)

	assert.Equal(t, StateComplete, updated.State())
	assert.Equal(t, gpuInfo, updated.GPUInfo())
	assert.Nil(t, updated.Error())
	assert.Nil(t, cmd)
}

func TestDetectionModel_Update_DetectionCompleteMsg_NilGPUInfo(t *testing.T) {
	styles := getTestStyles()
	m := NewDetection(styles, "1.0.0")
	m.SetSize(80, 24)

	msg := DetectionCompleteMsg{GPUInfo: nil}
	updated, _ := m.Update(msg)

	assert.Equal(t, StateComplete, updated.State())
	assert.Nil(t, updated.GPUInfo())
}

// =============================================================================
// Update Tests - DetectionErrorMsg
// =============================================================================

func TestDetectionModel_Update_DetectionErrorMsg(t *testing.T) {
	styles := getTestStyles()
	m := NewDetection(styles, "1.0.0")
	m.SetSize(80, 24)

	testErr := errors.New("detection failed: no PCI bus access")

	msg := DetectionErrorMsg{Error: testErr}
	updated, cmd := m.Update(msg)

	assert.Equal(t, StateError, updated.State())
	assert.Equal(t, testErr, updated.Error())
	assert.Nil(t, updated.GPUInfo())
	assert.Nil(t, cmd)
}

func TestDetectionModel_Update_DetectionErrorMsg_NilError(t *testing.T) {
	styles := getTestStyles()
	m := NewDetection(styles, "1.0.0")
	m.SetSize(80, 24)

	msg := DetectionErrorMsg{Error: nil}
	updated, _ := m.Update(msg)

	assert.Equal(t, StateError, updated.State())
	assert.Nil(t, updated.Error())
}

// =============================================================================
// Update Tests - Key Navigation in StateComplete
// =============================================================================

func TestDetectionModel_Update_EnterKey_StateComplete(t *testing.T) {
	styles := getTestStyles()
	m := NewDetection(styles, "1.0.0")
	m.SetSize(80, 24)
	m.SetGPUInfo(createMockGPUInfo())

	assert.Equal(t, StateComplete, m.State())

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	_, cmd := m.Update(msg)

	require.NotNil(t, cmd)

	result := cmd()
	navMsg, ok := result.(NavigateToDriverSelectionMsg)
	assert.True(t, ok, "Expected NavigateToDriverSelectionMsg")
	assert.NotNil(t, navMsg.GPUInfo)
}

func TestDetectionModel_Update_SpaceKey_StateComplete(t *testing.T) {
	styles := getTestStyles()
	m := NewDetection(styles, "1.0.0")
	m.SetSize(80, 24)
	m.SetGPUInfo(createMockGPUInfo())

	msg := tea.KeyMsg{Type: tea.KeySpace}
	_, cmd := m.Update(msg)

	require.NotNil(t, cmd)

	result := cmd()
	_, ok := result.(NavigateToDriverSelectionMsg)
	assert.True(t, ok, "Expected NavigateToDriverSelectionMsg from space key")
}

func TestDetectionModel_Update_EscKey_StateComplete(t *testing.T) {
	styles := getTestStyles()
	m := NewDetection(styles, "1.0.0")
	m.SetSize(80, 24)
	m.SetGPUInfo(createMockGPUInfo())

	msg := tea.KeyMsg{Type: tea.KeyEscape}
	_, cmd := m.Update(msg)

	require.NotNil(t, cmd)

	result := cmd()
	_, ok := result.(NavigateToWelcomeMsg)
	assert.True(t, ok, "Expected NavigateToWelcomeMsg")
}

func TestDetectionModel_Update_BackspaceKey_StateComplete(t *testing.T) {
	styles := getTestStyles()
	m := NewDetection(styles, "1.0.0")
	m.SetSize(80, 24)
	m.SetGPUInfo(createMockGPUInfo())

	msg := tea.KeyMsg{Type: tea.KeyBackspace}
	_, cmd := m.Update(msg)

	require.NotNil(t, cmd)

	result := cmd()
	_, ok := result.(NavigateToWelcomeMsg)
	assert.True(t, ok, "Expected NavigateToWelcomeMsg from backspace")
}

func TestDetectionModel_Update_QuitKey_StateComplete(t *testing.T) {
	styles := getTestStyles()
	m := NewDetection(styles, "1.0.0")
	m.SetSize(80, 24)
	m.SetGPUInfo(createMockGPUInfo())

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
	_, cmd := m.Update(msg)

	require.NotNil(t, cmd, "q key should return quit command")
}

func TestDetectionModel_Update_HelpKey_StateComplete(t *testing.T) {
	styles := getTestStyles()
	m := NewDetection(styles, "1.0.0")
	m.SetSize(80, 24)
	m.SetGPUInfo(createMockGPUInfo())

	assert.False(t, m.IsFullHelpShown())

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}}
	updated, cmd := m.Update(msg)

	assert.True(t, updated.IsFullHelpShown())
	assert.Nil(t, cmd)
}

// =============================================================================
// Update Tests - Key Navigation in StateError
// =============================================================================

func TestDetectionModel_Update_EscKey_StateError(t *testing.T) {
	styles := getTestStyles()
	m := NewDetection(styles, "1.0.0")
	m.SetSize(80, 24)
	m.SetError(errors.New("test error"))

	assert.Equal(t, StateError, m.State())

	msg := tea.KeyMsg{Type: tea.KeyEscape}
	_, cmd := m.Update(msg)

	require.NotNil(t, cmd)

	result := cmd()
	_, ok := result.(NavigateToWelcomeMsg)
	assert.True(t, ok, "Expected NavigateToWelcomeMsg")
}

func TestDetectionModel_Update_QuitKey_StateError(t *testing.T) {
	styles := getTestStyles()
	m := NewDetection(styles, "1.0.0")
	m.SetSize(80, 24)
	m.SetError(errors.New("test error"))

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
	_, cmd := m.Update(msg)

	require.NotNil(t, cmd, "q key should return quit command")
}

func TestDetectionModel_Update_HelpKey_StateError(t *testing.T) {
	styles := getTestStyles()
	m := NewDetection(styles, "1.0.0")
	m.SetSize(80, 24)
	m.SetError(errors.New("test error"))

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}}
	updated, cmd := m.Update(msg)

	assert.True(t, updated.IsFullHelpShown())
	assert.Nil(t, cmd)
}

// =============================================================================
// Update Tests - Key Navigation in StateDetecting
// =============================================================================

func TestDetectionModel_Update_QuitKey_StateDetecting(t *testing.T) {
	styles := getTestStyles()
	m := NewDetection(styles, "1.0.0")
	m.SetSize(80, 24)

	assert.Equal(t, StateDetecting, m.State())

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
	_, cmd := m.Update(msg)

	require.NotNil(t, cmd, "q key should return quit command even during detection")
}

func TestDetectionModel_Update_CtrlC_StateDetecting(t *testing.T) {
	styles := getTestStyles()
	m := NewDetection(styles, "1.0.0")
	m.SetSize(80, 24)

	msg := tea.KeyMsg{Type: tea.KeyCtrlC}
	_, cmd := m.Update(msg)

	require.NotNil(t, cmd, "ctrl+c should return quit command")
}

func TestDetectionModel_Update_OtherKeys_StateDetecting(t *testing.T) {
	styles := getTestStyles()
	m := NewDetection(styles, "1.0.0")
	m.SetSize(80, 24)

	// Enter key should be ignored during detection
	msg := tea.KeyMsg{Type: tea.KeyEnter}
	updated, cmd := m.Update(msg)

	assert.Equal(t, StateDetecting, updated.State())
	// Should return nil or batch command (not a navigation command)
	if cmd != nil {
		result := cmd()
		_, isNav := result.(NavigateToDriverSelectionMsg)
		assert.False(t, isNav, "Should not navigate during detection")
	}
}

// =============================================================================
// View Tests - Not Ready
// =============================================================================

func TestDetectionModel_View_NotReady(t *testing.T) {
	styles := getTestStyles()
	m := NewDetection(styles, "1.0.0")

	view := m.View()

	assert.Equal(t, "Loading...", view)
}

// =============================================================================
// View Tests - StateDetecting
// =============================================================================

func TestDetectionModel_View_Detecting(t *testing.T) {
	styles := getTestStyles()
	m := NewDetection(styles, "1.0.0")
	m.SetSize(80, 24)

	view := m.View()

	assert.NotEmpty(t, view)
	assert.NotEqual(t, "Loading...", view)
	assert.Contains(t, view, "Detecting System Configuration")
	assert.Contains(t, view, "Please wait")
}

func TestDetectionModel_View_Detecting_ShowsStep(t *testing.T) {
	styles := getTestStyles()
	m := NewDetection(styles, "1.0.0")
	m.SetSize(80, 24)

	view := m.View()

	// Should contain the spinner with some initialization message
	// The spinner message includes "Initializing detection..." at step 0
	assert.Contains(t, view, "Initializing detection...")
}

// =============================================================================
// View Tests - StateComplete
// =============================================================================

func TestDetectionModel_View_Complete(t *testing.T) {
	styles := getTestStyles()
	m := NewDetection(styles, "1.0.0")
	m.SetSize(100, 40)
	m.SetGPUInfo(createMockGPUInfo())

	view := m.View()

	assert.NotEmpty(t, view)
	assert.Contains(t, view, "NVIDIA GPUs")
	assert.Contains(t, view, "Driver Status")
	assert.Contains(t, view, "System Information")
}

func TestDetectionModel_View_Complete_ShowsGPUName(t *testing.T) {
	styles := getTestStyles()
	m := NewDetection(styles, "1.0.0")
	m.SetSize(100, 40)
	m.SetGPUInfo(createMockGPUInfo())

	view := m.View()

	assert.Contains(t, view, "RTX 4090")
}

func TestDetectionModel_View_Complete_ShowsDriverInfo(t *testing.T) {
	styles := getTestStyles()
	m := NewDetection(styles, "1.0.0")
	m.SetSize(100, 40)
	m.SetGPUInfo(createMockGPUInfo())

	view := m.View()

	assert.Contains(t, view, "550.54.14")
	assert.Contains(t, view, "12.4")
}

func TestDetectionModel_View_Complete_ShowsKernelInfo(t *testing.T) {
	styles := getTestStyles()
	m := NewDetection(styles, "1.0.0")
	m.SetSize(100, 40)
	m.SetGPUInfo(createMockGPUInfo())

	view := m.View()

	assert.Contains(t, view, "6.5.0-44-generic")
}

func TestDetectionModel_View_Complete_ShowsValidation(t *testing.T) {
	styles := getTestStyles()
	m := NewDetection(styles, "1.0.0")
	m.SetSize(100, 40)
	m.SetGPUInfo(createMockGPUInfo())

	view := m.View()

	assert.Contains(t, view, "Validation")
	assert.Contains(t, view, "ready for installation")
}

func TestDetectionModel_View_Complete_NoDriver(t *testing.T) {
	styles := getTestStyles()
	m := NewDetection(styles, "1.0.0")
	m.SetSize(100, 40)
	m.SetGPUInfo(createMockGPUInfoWithNoDriver())

	view := m.View()

	assert.Contains(t, view, "No NVIDIA driver installed")
}

func TestDetectionModel_View_Complete_NouveauLoaded(t *testing.T) {
	styles := getTestStyles()
	m := NewDetection(styles, "1.0.0")
	m.SetSize(100, 40)
	m.SetGPUInfo(createMockGPUInfoWithNouveauLoaded())

	view := m.View()

	assert.Contains(t, view, "Nouveau")
	assert.Contains(t, view, "Loaded")
}

func TestDetectionModel_View_Complete_ValidationErrors(t *testing.T) {
	styles := getTestStyles()
	m := NewDetection(styles, "1.0.0")
	m.SetSize(100, 40)
	m.SetGPUInfo(createMockGPUInfoWithValidationErrors())

	view := m.View()

	assert.Contains(t, view, "Errors: 1")
	assert.Contains(t, view, "Warnings: 1")
}

func TestDetectionModel_View_Complete_MultipleGPUs(t *testing.T) {
	styles := getTestStyles()
	m := NewDetection(styles, "1.0.0")
	m.SetSize(100, 40)
	m.SetGPUInfo(createMockGPUInfoWithMultipleGPUs())

	view := m.View()

	assert.Contains(t, view, "RTX 4090")
	assert.Contains(t, view, "RTX 3090")
}

func TestDetectionModel_View_Complete_ShowsInstructions(t *testing.T) {
	styles := getTestStyles()
	m := NewDetection(styles, "1.0.0")
	m.SetSize(100, 40)
	m.SetGPUInfo(createMockGPUInfo())

	view := m.View()

	assert.Contains(t, view, "Press Enter to continue")
	assert.Contains(t, view, "Esc to go back")
}

// =============================================================================
// View Tests - StateComplete with No GPU
// =============================================================================

func TestDetectionModel_View_Complete_NoGPU(t *testing.T) {
	styles := getTestStyles()
	m := NewDetection(styles, "1.0.0")
	m.SetSize(100, 40)

	// Complete with nil GPU info
	msg := DetectionCompleteMsg{GPUInfo: nil}
	m, _ = m.Update(msg)

	view := m.View()

	assert.Contains(t, view, "No NVIDIA GPUs Detected")
	assert.Contains(t, view, "ensure your GPU is properly installed")
}

func TestDetectionModel_View_Complete_EmptyGPUList(t *testing.T) {
	styles := getTestStyles()
	m := NewDetection(styles, "1.0.0")
	m.SetSize(100, 40)

	emptyInfo := &gpu.GPUInfo{
		NVIDIAGPUs: []gpu.NVIDIAGPUInfo{},
	}
	m.SetGPUInfo(emptyInfo)

	view := m.View()

	assert.Contains(t, view, "No NVIDIA GPUs Detected")
}

// =============================================================================
// View Tests - StateError
// =============================================================================

func TestDetectionModel_View_Error(t *testing.T) {
	styles := getTestStyles()
	m := NewDetection(styles, "1.0.0")
	m.SetSize(80, 24)
	m.SetError(errors.New("PCI bus access denied"))

	view := m.View()

	assert.Contains(t, view, "Detection Failed")
	assert.Contains(t, view, "PCI bus access denied")
}

func TestDetectionModel_View_Error_NilError(t *testing.T) {
	styles := getTestStyles()
	m := NewDetection(styles, "1.0.0")
	m.SetSize(80, 24)

	msg := DetectionErrorMsg{Error: nil}
	m, _ = m.Update(msg)

	view := m.View()

	assert.Contains(t, view, "Detection Failed")
	assert.Contains(t, view, "unknown error")
}

func TestDetectionModel_View_Error_ShowsInstructions(t *testing.T) {
	styles := getTestStyles()
	m := NewDetection(styles, "1.0.0")
	m.SetSize(80, 24)
	m.SetError(errors.New("test error"))

	view := m.View()

	assert.Contains(t, view, "Esc to go back")
	assert.Contains(t, view, "q to quit")
}

// =============================================================================
// View Tests - Various Sizes
// =============================================================================

func TestDetectionModel_View_VariousSizes(t *testing.T) {
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
			m := NewDetection(styles, "1.0.0")
			m.SetSize(tc.width, tc.height)
			m.SetGPUInfo(createMockGPUInfo())

			view := m.View()

			assert.NotEmpty(t, view)
			assert.NotEqual(t, "Loading...", view)
		})
	}
}

func TestDetectionModel_View_VerySmallSize(t *testing.T) {
	styles := getTestStyles()
	m := NewDetection(styles, "1.0.0")
	m.SetSize(10, 5)
	m.SetGPUInfo(createMockGPUInfo())

	// Should not panic
	view := m.View()
	assert.NotEmpty(t, view)
}

func TestDetectionModel_View_ZeroHeight(t *testing.T) {
	styles := getTestStyles()
	m := NewDetection(styles, "1.0.0")
	m.SetSize(80, 0)

	// Should not panic
	view := m.View()
	assert.NotEmpty(t, view)
}

// =============================================================================
// SetSize Tests
// =============================================================================

func TestDetectionModel_SetSize(t *testing.T) {
	styles := getTestStyles()
	m := NewDetection(styles, "1.0.0")

	assert.False(t, m.Ready())

	m.SetSize(100, 50)

	assert.True(t, m.Ready())
	assert.Equal(t, 100, m.Width())
	assert.Equal(t, 50, m.Height())
}

func TestDetectionModel_SetSize_Multiple(t *testing.T) {
	styles := getTestStyles()
	m := NewDetection(styles, "1.0.0")

	m.SetSize(80, 24)
	assert.Equal(t, 80, m.Width())
	assert.Equal(t, 24, m.Height())

	m.SetSize(120, 40)
	assert.Equal(t, 120, m.Width())
	assert.Equal(t, 40, m.Height())
}

// =============================================================================
// SetGPUInfo Tests
// =============================================================================

func TestDetectionModel_SetGPUInfo(t *testing.T) {
	styles := getTestStyles()
	m := NewDetection(styles, "1.0.0")
	m.SetSize(80, 24)

	assert.Equal(t, StateDetecting, m.State())

	gpuInfo := createMockGPUInfo()
	m.SetGPUInfo(gpuInfo)

	assert.Equal(t, StateComplete, m.State())
	assert.Equal(t, gpuInfo, m.GPUInfo())
}

func TestDetectionModel_SetGPUInfo_Nil(t *testing.T) {
	styles := getTestStyles()
	m := NewDetection(styles, "1.0.0")
	m.SetSize(80, 24)

	m.SetGPUInfo(nil)

	assert.Equal(t, StateComplete, m.State())
	assert.Nil(t, m.GPUInfo())
}

// =============================================================================
// SetError Tests
// =============================================================================

func TestDetectionModel_SetError(t *testing.T) {
	styles := getTestStyles()
	m := NewDetection(styles, "1.0.0")
	m.SetSize(80, 24)

	assert.Equal(t, StateDetecting, m.State())

	testErr := errors.New("test error")
	m.SetError(testErr)

	assert.Equal(t, StateError, m.State())
	assert.Equal(t, testErr, m.Error())
}

func TestDetectionModel_SetError_Nil(t *testing.T) {
	styles := getTestStyles()
	m := NewDetection(styles, "1.0.0")
	m.SetSize(80, 24)

	m.SetError(nil)

	assert.Equal(t, StateError, m.State())
	assert.Nil(t, m.Error())
}

// =============================================================================
// SetStep Tests
// =============================================================================

func TestDetectionModel_SetStep(t *testing.T) {
	styles := getTestStyles()
	m := NewDetection(styles, "1.0.0")

	steps := m.Steps()

	m.SetStep(3)

	assert.Equal(t, 3, m.StepIndex())
	assert.Equal(t, steps[3], m.CurrentStep())
}

func TestDetectionModel_SetStep_OutOfBounds(t *testing.T) {
	styles := getTestStyles()
	m := NewDetection(styles, "1.0.0")

	initialStep := m.CurrentStep()

	// Negative index
	m.SetStep(-1)
	assert.Equal(t, -1, m.StepIndex())
	assert.Equal(t, initialStep, m.CurrentStep()) // Should not change

	// Beyond bounds
	m.SetStep(100)
	assert.Equal(t, 100, m.StepIndex())
}

// =============================================================================
// Getter Tests
// =============================================================================

func TestDetectionModel_Getters(t *testing.T) {
	styles := getTestStyles()
	m := NewDetection(styles, "2.0.0")
	m.SetSize(100, 50)

	assert.Equal(t, 100, m.Width())
	assert.Equal(t, 50, m.Height())
	assert.True(t, m.Ready())
	assert.Equal(t, "2.0.0", m.Version())
	assert.NotNil(t, m.KeyMap())
	assert.Equal(t, StateDetecting, m.State())
}

// =============================================================================
// Message Type Tests
// =============================================================================

func TestDetectionStepMsg(t *testing.T) {
	msg := DetectionStepMsg{Step: 3}
	assert.Equal(t, 3, msg.Step)
}

func TestDetectionCompleteMsg(t *testing.T) {
	gpuInfo := createMockGPUInfo()
	msg := DetectionCompleteMsg{GPUInfo: gpuInfo}
	assert.Equal(t, gpuInfo, msg.GPUInfo)
}

func TestDetectionErrorMsg(t *testing.T) {
	err := errors.New("test error")
	msg := DetectionErrorMsg{Error: err}
	assert.Equal(t, err, msg.Error)
}

func TestNavigateToDriverSelectionMsg(t *testing.T) {
	gpuInfo := createMockGPUInfo()
	msg := NavigateToDriverSelectionMsg{GPUInfo: gpuInfo}
	assert.Equal(t, gpuInfo, msg.GPUInfo)
}

func TestNavigateToWelcomeMsg(t *testing.T) {
	msg := NavigateToWelcomeMsg{}
	assert.NotNil(t, msg)
}

// =============================================================================
// Integration Tests
// =============================================================================

func TestDetectionModel_FullFlow_Success(t *testing.T) {
	styles := getTestStyles()
	m := NewDetection(styles, "1.0.0")

	// Simulate window resize
	m, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	assert.True(t, m.Ready())
	assert.Equal(t, StateDetecting, m.State())

	// Render detecting state
	view := m.View()
	assert.Contains(t, view, "Detecting")

	// Simulate step updates
	for i := 0; i < len(m.Steps()); i++ {
		m, _ = m.Update(DetectionStepMsg{Step: i})
		assert.Equal(t, i, m.StepIndex())
	}

	// Simulate detection complete
	gpuInfo := createMockGPUInfo()
	m, _ = m.Update(DetectionCompleteMsg{GPUInfo: gpuInfo})
	assert.Equal(t, StateComplete, m.State())

	// Render complete state
	view = m.View()
	assert.Contains(t, view, "RTX 4090")

	// Press enter to continue
	m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.NotNil(t, cmd)

	result := cmd()
	_, ok := result.(NavigateToDriverSelectionMsg)
	assert.True(t, ok)
}

func TestDetectionModel_FullFlow_Error(t *testing.T) {
	styles := getTestStyles()
	m := NewDetection(styles, "1.0.0")

	// Simulate window resize
	m, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	// Simulate detection error
	testErr := errors.New("PCI bus not accessible")
	m, _ = m.Update(DetectionErrorMsg{Error: testErr})
	assert.Equal(t, StateError, m.State())

	// Render error state
	view := m.View()
	assert.Contains(t, view, "Detection Failed")
	assert.Contains(t, view, "PCI bus not accessible")

	// Press esc to go back
	m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	require.NotNil(t, cmd)

	result := cmd()
	_, ok := result.(NavigateToWelcomeMsg)
	assert.True(t, ok)
}

func TestDetectionModel_HelpToggleFlow(t *testing.T) {
	styles := getTestStyles()
	m := NewDetection(styles, "1.0.0")
	m.SetSize(80, 24)
	m.SetGPUInfo(createMockGPUInfo())

	// Help is initially not shown in full
	assert.False(t, m.IsFullHelpShown())

	// Toggle help on
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	assert.True(t, m.IsFullHelpShown())

	// Render with full help - should not panic
	view := m.View()
	assert.NotEmpty(t, view)

	// Toggle help off
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	assert.False(t, m.IsFullHelpShown())
}

// =============================================================================
// KeyMap Interface Compliance Tests
// =============================================================================

func TestDetectionKeyMap_ImplementsHelpKeyMap(t *testing.T) {
	km := DefaultDetectionKeyMap()

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

func TestDetectionKeyMap_BindingsHaveHelp(t *testing.T) {
	km := DefaultDetectionKeyMap()

	// All bindings should have help text
	t.Run("Continue", func(t *testing.T) {
		help := km.Continue.Help()
		assert.NotEmpty(t, help.Key, "binding should have key help")
		assert.NotEmpty(t, help.Desc, "binding should have description")
	})

	t.Run("Back", func(t *testing.T) {
		help := km.Back.Help()
		assert.NotEmpty(t, help.Key, "binding should have key help")
		assert.NotEmpty(t, help.Desc, "binding should have description")
	})

	t.Run("Quit", func(t *testing.T) {
		help := km.Quit.Help()
		assert.NotEmpty(t, help.Key, "binding should have key help")
		assert.NotEmpty(t, help.Desc, "binding should have description")
	})

	t.Run("Help", func(t *testing.T) {
		help := km.Help.Help()
		assert.NotEmpty(t, help.Key, "binding should have key help")
		assert.NotEmpty(t, help.Desc, "binding should have description")
	})
}

// =============================================================================
// Edge Cases
// =============================================================================

func TestDetectionModel_RapidStateChanges(t *testing.T) {
	styles := getTestStyles()
	m := NewDetection(styles, "1.0.0")
	m.SetSize(80, 24)

	// Rapid state changes
	for i := 0; i < 10; i++ {
		m.SetGPUInfo(createMockGPUInfo())
		m.SetError(errors.New("error"))
		m.SetGPUInfo(nil)
	}

	// Should be in a valid state
	assert.Contains(t, []DetectionState{StateComplete, StateError}, m.State())
}

func TestDetectionModel_MultipleSizeChanges(t *testing.T) {
	styles := getTestStyles()
	m := NewDetection(styles, "1.0.0")
	m.SetGPUInfo(createMockGPUInfo())

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

func TestDetectionModel_EmptyStyles(t *testing.T) {
	// Test with zero-value styles
	var styles theme.Styles
	m := NewDetection(styles, "1.0.0")
	m.SetSize(80, 24)
	m.SetGPUInfo(createMockGPUInfo())

	// Should not panic
	view := m.View()
	assert.NotEmpty(t, view)
}

func TestDetectionModel_UnknownMessage(t *testing.T) {
	styles := getTestStyles()
	m := NewDetection(styles, "1.0.0")
	m.SetSize(80, 24)

	type customMsg struct{}

	updated, cmd := m.Update(customMsg{})

	// State should remain unchanged
	assert.True(t, updated.Ready())
	assert.Equal(t, StateDetecting, updated.State())
	// Only command from batch should be nil or spinner-related
	if cmd != nil {
		// Execute and verify it's not a navigation message
		result := cmd()
		_, isNav := result.(NavigateToDriverSelectionMsg)
		assert.False(t, isNav)
	}
}

// =============================================================================
// GPU Info Edge Cases
// =============================================================================

func TestDetectionModel_GPUInfo_NoModel(t *testing.T) {
	styles := getTestStyles()
	m := NewDetection(styles, "1.0.0")
	m.SetSize(100, 40)

	// GPU without model info
	info := &gpu.GPUInfo{
		NVIDIAGPUs: []gpu.NVIDIAGPUInfo{
			{
				PCIDevice: pci.PCIDevice{
					Address:  "0000:01:00.0",
					VendorID: "10de",
					DeviceID: "9999",
					Class:    "0300",
				},
				Model: nil, // No model info
			},
		},
	}
	m.SetGPUInfo(info)

	view := m.View()

	// Should show device ID instead of name
	assert.Contains(t, view, "9999")
}

func TestDetectionModel_GPUInfo_NoKernelInfo(t *testing.T) {
	styles := getTestStyles()
	m := NewDetection(styles, "1.0.0")
	m.SetSize(100, 40)

	info := createMockGPUInfo()
	info.KernelInfo = nil
	m.SetGPUInfo(info)

	view := m.View()

	// Should still render without kernel info
	assert.Contains(t, view, "System Information")
}

func TestDetectionModel_GPUInfo_NoValidationReport(t *testing.T) {
	styles := getTestStyles()
	m := NewDetection(styles, "1.0.0")
	m.SetSize(100, 40)

	info := createMockGPUInfo()
	info.ValidationReport = nil
	m.SetGPUInfo(info)

	view := m.View()

	// Should still render without validation section
	assert.NotEmpty(t, view)
}
