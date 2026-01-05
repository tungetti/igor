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
	"github.com/tungetti/igor/internal/gpu/pci"
	"github.com/tungetti/igor/internal/ui/theme"
)

// =============================================================================
// Test Helpers
// =============================================================================

// getIntegrationTestStyles returns default styles for integration testing.
func getIntegrationTestStyles() theme.Styles {
	return theme.DefaultTheme().Styles
}

// simulateWindowSize creates a window size message.
func simulateWindowSize(width, height int) tea.WindowSizeMsg {
	return tea.WindowSizeMsg{Width: width, Height: height}
}

// simulateEnterKey creates an Enter key message.
func simulateEnterKey() tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyEnter}
}

// simulateEscapeKey creates an Escape key message.
func simulateEscapeKey() tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyEsc}
}

// simulateUpKey creates an Up arrow key message.
func simulateUpKey() tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyUp}
}

// simulateDownKey creates a Down arrow key message.
func simulateDownKey() tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyDown}
}

// simulateLeftKey creates a Left arrow key message.
func simulateLeftKey() tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyLeft}
}

// simulateRightKey creates a Right arrow key message.
func simulateRightKey() tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRight}
}

// simulateTabKey creates a Tab key message.
func simulateTabKey() tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyTab}
}

// simulateSpaceKey creates a Space key message.
func simulateSpaceKey() tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeySpace}
}

// simulateRuneKey creates a key message for a single rune.
func simulateRuneKey(r rune) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}
}

// createTestGPUInfo creates a test GPUInfo structure.
func createTestGPUInfo() *gpu.GPUInfo {
	return &gpu.GPUInfo{
		NVIDIAGPUs: []gpu.NVIDIAGPUInfo{
			{
				PCIDevice: pci.PCIDevice{
					Address:  "0000:01:00.0",
					VendorID: "10de",
					DeviceID: "2684",
					Class:    "0300",
				},
			},
		},
		InstalledDriver: &gpu.DriverInfo{
			Installed: false,
			Type:      gpu.DriverTypeNone,
		},
		KernelInfo: &kernel.KernelInfo{
			Version:          "6.1.0-generic",
			HeadersInstalled: true,
		},
		NouveauStatus: &nouveau.Status{
			Loaded: false,
		},
	}
}

// createTestGPUInfoWithNouveauLoaded creates GPU info with Nouveau loaded.
func createTestGPUInfoWithNouveauLoaded() *gpu.GPUInfo {
	info := createTestGPUInfo()
	info.NouveauStatus.Loaded = true
	return info
}

// createTestGPUInfoWithDriver creates GPU info with driver installed.
func createTestGPUInfoWithDriver() *gpu.GPUInfo {
	info := createTestGPUInfo()
	info.InstalledDriver = &gpu.DriverInfo{
		Installed:   true,
		Type:        gpu.DriverTypeNVIDIA,
		Version:     "550.54.14",
		CUDAVersion: "12.4",
	}
	return info
}

// =============================================================================
// TestWelcomeView_Interactions
// =============================================================================

func TestWelcomeView_Interactions(t *testing.T) {
	t.Run("enter on start button triggers detection", func(t *testing.T) {
		styles := getIntegrationTestStyles()
		m := NewWelcome(styles, "1.0.0")
		m.SetSize(100, 30)

		// Ensure focus is on Start button
		assert.Equal(t, 0, m.FocusedButtonIndex())

		// Press enter
		m, cmd := m.Update(simulateEnterKey())
		require.NotNil(t, cmd)

		result := cmd()
		_, ok := result.(StartDetectionMsg)
		assert.True(t, ok, "Expected StartDetectionMsg")
	})

	t.Run("enter on exit button triggers quit", func(t *testing.T) {
		styles := getIntegrationTestStyles()
		m := NewWelcome(styles, "1.0.0")
		m.SetSize(100, 30)

		// Navigate to Exit button
		m.FocusButton(1)
		assert.Equal(t, 1, m.FocusedButtonIndex())

		// Press enter
		_, cmd := m.Update(simulateEnterKey())
		require.NotNil(t, cmd)
	})

	t.Run("q key triggers quit", func(t *testing.T) {
		styles := getIntegrationTestStyles()
		m := NewWelcome(styles, "1.0.0")
		m.SetSize(100, 30)

		_, cmd := m.Update(simulateRuneKey('q'))
		require.NotNil(t, cmd)
	})

	t.Run("display branding elements", func(t *testing.T) {
		styles := getIntegrationTestStyles()
		m := NewWelcome(styles, "2.0.0")
		m.SetSize(100, 40)

		view := m.View()
		assert.Contains(t, view, "IGOR")
		assert.Contains(t, view, "NVIDIA")
		assert.Contains(t, view, "Welcome to Igor")
		assert.Contains(t, view, "Start Installation")
		assert.Contains(t, view, "Exit")
	})

	t.Run("navigation between buttons", func(t *testing.T) {
		styles := getIntegrationTestStyles()
		m := NewWelcome(styles, "1.0.0")
		m.SetSize(100, 30)

		// Start at button 0
		assert.Equal(t, 0, m.FocusedButtonIndex())

		// Navigate right
		m, _ = m.Update(simulateRightKey())
		assert.Equal(t, 1, m.FocusedButtonIndex())

		// Navigate left
		m, _ = m.Update(simulateLeftKey())
		assert.Equal(t, 0, m.FocusedButtonIndex())

		// Tab also navigates
		m, _ = m.Update(simulateTabKey())
		assert.Equal(t, 1, m.FocusedButtonIndex())
	})

	t.Run("help toggle", func(t *testing.T) {
		styles := getIntegrationTestStyles()
		m := NewWelcome(styles, "1.0.0")
		m.SetSize(100, 30)

		assert.False(t, m.IsFullHelpShown())

		m, _ = m.Update(simulateRuneKey('?'))
		assert.True(t, m.IsFullHelpShown())

		m, _ = m.Update(simulateRuneKey('?'))
		assert.False(t, m.IsFullHelpShown())
	})
}

// =============================================================================
// TestDetectionView_AnimationFlow
// =============================================================================

func TestDetectionView_AnimationFlow(t *testing.T) {
	t.Run("spinner animation during detection", func(t *testing.T) {
		styles := getIntegrationTestStyles()
		m := NewDetection(styles, "1.0.0")
		m.SetSize(100, 30)

		// Initially in detecting state
		assert.Equal(t, StateDetecting, m.State())

		// Spinner should be visible
		view := m.View()
		assert.Contains(t, view, "Detecting")

		// Spinner tick should update
		tickMsg := spinner.TickMsg{}
		m, cmd := m.Update(tickMsg)
		assert.NotNil(t, cmd) // Should return next tick command
	})

	t.Run("progress through detection steps", func(t *testing.T) {
		styles := getIntegrationTestStyles()
		m := NewDetection(styles, "1.0.0")
		m.SetSize(100, 30)

		// Update step
		m, _ = m.Update(DetectionStepMsg{Step: 1})
		assert.Equal(t, 1, m.StepIndex())

		m, _ = m.Update(DetectionStepMsg{Step: 2})
		assert.Equal(t, 2, m.StepIndex())
	})

	t.Run("detection complete transition", func(t *testing.T) {
		styles := getIntegrationTestStyles()
		m := NewDetection(styles, "1.0.0")
		m.SetSize(100, 30)

		gpuInfo := createTestGPUInfo()
		m, _ = m.Update(DetectionCompleteMsg{GPUInfo: gpuInfo})

		assert.Equal(t, StateComplete, m.State())
		assert.Equal(t, gpuInfo, m.GPUInfo())
	})

	t.Run("detection error handling", func(t *testing.T) {
		styles := getIntegrationTestStyles()
		m := NewDetection(styles, "1.0.0")
		m.SetSize(100, 30)

		testErr := errors.New("detection failed")
		m, _ = m.Update(DetectionErrorMsg{Error: testErr})

		assert.Equal(t, StateError, m.State())
		assert.Equal(t, testErr, m.Error())
	})

	t.Run("continue after detection complete", func(t *testing.T) {
		styles := getIntegrationTestStyles()
		m := NewDetection(styles, "1.0.0")
		m.SetSize(100, 30)

		gpuInfo := createTestGPUInfo()
		m, _ = m.Update(DetectionCompleteMsg{GPUInfo: gpuInfo})

		// Press enter to continue
		_, cmd := m.Update(simulateEnterKey())
		require.NotNil(t, cmd)

		result := cmd()
		navMsg, ok := result.(NavigateToDriverSelectionMsg)
		assert.True(t, ok)
		assert.Equal(t, gpuInfo, navMsg.GPUInfo)
	})

	t.Run("back navigation from complete state", func(t *testing.T) {
		styles := getIntegrationTestStyles()
		m := NewDetection(styles, "1.0.0")
		m.SetSize(100, 30)

		m, _ = m.Update(DetectionCompleteMsg{GPUInfo: createTestGPUInfo()})

		_, cmd := m.Update(simulateEscapeKey())
		require.NotNil(t, cmd)

		result := cmd()
		_, ok := result.(NavigateToWelcomeMsg)
		assert.True(t, ok)
	})
}

// =============================================================================
// TestSelectionView_ListNavigation
// =============================================================================

func TestSelectionView_ListNavigation(t *testing.T) {
	t.Run("up down navigation in drivers list", func(t *testing.T) {
		styles := getIntegrationTestStyles()
		gpuInfo := createTestGPUInfo()
		m := NewSelection(styles, "1.0.0", gpuInfo)
		m.SetSize(100, 30)

		// Start at driver 0
		assert.Equal(t, 0, m.SelectedDriverIndex())
		assert.Equal(t, 0, m.FocusedSection())

		// Navigate down
		m, _ = m.Update(simulateDownKey())
		assert.Equal(t, 1, m.SelectedDriverIndex())

		m, _ = m.Update(simulateDownKey())
		assert.Equal(t, 2, m.SelectedDriverIndex())

		// Navigate up
		m, _ = m.Update(simulateUpKey())
		assert.Equal(t, 1, m.SelectedDriverIndex())
	})

	t.Run("tab switches sections", func(t *testing.T) {
		styles := getIntegrationTestStyles()
		m := NewSelection(styles, "1.0.0", createTestGPUInfo())
		m.SetSize(100, 30)

		// Start in drivers section
		assert.Equal(t, 0, m.FocusedSection())

		// Tab to components
		m, _ = m.Update(simulateTabKey())
		assert.Equal(t, 1, m.FocusedSection())

		// Tab to buttons
		m, _ = m.Update(simulateTabKey())
		assert.Equal(t, 2, m.FocusedSection())

		// Tab wraps to drivers
		m, _ = m.Update(simulateTabKey())
		assert.Equal(t, 0, m.FocusedSection())
	})

	t.Run("component selection toggle", func(t *testing.T) {
		styles := getIntegrationTestStyles()
		m := NewSelection(styles, "1.0.0", createTestGPUInfo())
		m.SetSize(100, 30)

		// Switch to components section
		m, _ = m.Update(simulateTabKey())
		assert.Equal(t, 1, m.FocusedSection())

		// Move to CUDA (second component)
		m, _ = m.Update(simulateDownKey())
		assert.Equal(t, 1, m.SelectedComponentIndex())

		// Toggle with space
		components := m.ComponentOptions()
		initialSelected := components[1].Selected

		m, _ = m.Update(simulateSpaceKey())
		updatedComponents := m.ComponentOptions()
		assert.NotEqual(t, initialSelected, updatedComponents[1].Selected)
	})

	t.Run("required component cannot be toggled", func(t *testing.T) {
		styles := getIntegrationTestStyles()
		m := NewSelection(styles, "1.0.0", createTestGPUInfo())
		m.SetSize(100, 30)

		// Switch to components section
		m, _ = m.Update(simulateTabKey())

		// First component (driver) is required
		components := m.ComponentOptions()
		assert.True(t, components[0].Required)
		assert.True(t, components[0].Selected)

		// Try to toggle
		m, _ = m.Update(simulateSpaceKey())
		updatedComponents := m.ComponentOptions()
		assert.True(t, updatedComponents[0].Selected) // Still selected
	})

	t.Run("recommended driver marking", func(t *testing.T) {
		styles := getIntegrationTestStyles()
		m := NewSelection(styles, "1.0.0", createTestGPUInfo())
		m.SetSize(100, 30)

		drivers := m.DriverOptions()
		var hasRecommended bool
		for _, d := range drivers {
			if d.Recommended {
				hasRecommended = true
				break
			}
		}
		assert.True(t, hasRecommended)

		view := m.View()
		assert.Contains(t, view, "Recommended")
	})

	t.Run("continue to confirmation", func(t *testing.T) {
		styles := getIntegrationTestStyles()
		gpuInfo := createTestGPUInfo()
		m := NewSelection(styles, "1.0.0", gpuInfo)
		m.SetSize(100, 30)

		// Navigate to buttons section
		m, _ = m.Update(simulateTabKey()) // components
		m, _ = m.Update(simulateTabKey()) // buttons

		// Press enter on Continue button
		_, cmd := m.Update(simulateEnterKey())
		require.NotNil(t, cmd)

		result := cmd()
		navMsg, ok := result.(NavigateToConfirmationMsg)
		assert.True(t, ok)
		assert.Equal(t, gpuInfo, navMsg.GPUInfo)
	})

	t.Run("back navigation", func(t *testing.T) {
		styles := getIntegrationTestStyles()
		m := NewSelection(styles, "1.0.0", createTestGPUInfo())
		m.SetSize(100, 30)

		_, cmd := m.Update(simulateEscapeKey())
		require.NotNil(t, cmd)

		result := cmd()
		_, ok := result.(NavigateToDetectionMsg)
		assert.True(t, ok)
	})
}

// =============================================================================
// TestConfirmationView_Summary
// =============================================================================

func TestConfirmationView_Summary(t *testing.T) {
	t.Run("display selected options", func(t *testing.T) {
		styles := getIntegrationTestStyles()
		gpuInfo := createTestGPUInfo()
		driver := DriverOption{Version: "550", Branch: "Latest", Recommended: true}
		components := []ComponentOption{
			{Name: "NVIDIA Driver", ID: "driver", Selected: true, Required: true},
			{Name: "CUDA Toolkit", ID: "cuda", Selected: true},
		}

		m := NewConfirmation(styles, "1.0.0", gpuInfo, driver, components)
		m.SetSize(100, 40)

		view := m.View()
		assert.Contains(t, view, "Installation Summary")
		assert.Contains(t, view, "550")
		assert.Contains(t, view, "NVIDIA Driver")
		assert.Contains(t, view, "CUDA Toolkit")
	})

	t.Run("confirm and cancel buttons", func(t *testing.T) {
		styles := getIntegrationTestStyles()
		m := NewConfirmation(styles, "1.0.0", createTestGPUInfo(),
			DriverOption{Version: "550"},
			[]ComponentOption{{Name: "Driver", ID: "driver", Selected: true}})
		m.SetSize(100, 30)

		view := m.View()
		assert.Contains(t, view, "Install")
		assert.Contains(t, view, "Go Back")
	})

	t.Run("confirm starts installation", func(t *testing.T) {
		styles := getIntegrationTestStyles()
		gpuInfo := createTestGPUInfo()
		driver := DriverOption{Version: "550", Branch: "Latest"}
		components := []ComponentOption{{Name: "Driver", ID: "driver", Selected: true}}

		m := NewConfirmation(styles, "1.0.0", gpuInfo, driver, components)
		m.SetSize(100, 30)

		// Focus should be on Confirm button
		assert.Equal(t, 0, m.FocusedButtonIndex())

		_, cmd := m.Update(simulateEnterKey())
		require.NotNil(t, cmd)

		result := cmd()
		installMsg, ok := result.(StartInstallationMsg)
		assert.True(t, ok)
		assert.Equal(t, gpuInfo, installMsg.GPUInfo)
		assert.Equal(t, driver, installMsg.Driver)
	})

	t.Run("cancel returns to selection", func(t *testing.T) {
		styles := getIntegrationTestStyles()
		m := NewConfirmation(styles, "1.0.0", createTestGPUInfo(),
			DriverOption{Version: "550"},
			[]ComponentOption{})
		m.SetSize(100, 30)

		// Navigate to Cancel button
		m, _ = m.Update(simulateRightKey())
		assert.Equal(t, 1, m.FocusedButtonIndex())

		_, cmd := m.Update(simulateEnterKey())
		require.NotNil(t, cmd)

		result := cmd()
		_, ok := result.(NavigateBackToSelectionMsg)
		assert.True(t, ok)
	})
}

// =============================================================================
// TestProgressView_Updates
// =============================================================================

func TestProgressView_Updates(t *testing.T) {
	t.Run("progress bar updates", func(t *testing.T) {
		styles := getIntegrationTestStyles()
		gpuInfo := createTestGPUInfo()
		driver := DriverOption{Version: "550"}
		components := []ComponentOption{{Name: "Driver", ID: "driver", Selected: true}}

		m := NewProgress(styles, "1.0.0", gpuInfo, driver, components)
		m.SetSize(100, 30)

		// Update progress through step messages
		m, _ = m.Update(InstallationStepStartMsg{StepIndex: 0})
		assert.Equal(t, 0, m.CurrentStep())

		m, _ = m.Update(InstallationStepCompleteMsg{StepIndex: 0})
		assert.Greater(t, m.Progress(), 0.0)
	})

	t.Run("step completion", func(t *testing.T) {
		styles := getIntegrationTestStyles()
		m := NewProgress(styles, "1.0.0", createTestGPUInfo(),
			DriverOption{Version: "550"},
			[]ComponentOption{{Name: "Driver", ID: "driver", Selected: true}})
		m.SetSize(100, 30)

		steps := m.Steps()
		require.Greater(t, len(steps), 0)

		// Complete first step
		m, _ = m.Update(InstallationStepStartMsg{StepIndex: 0})
		m, _ = m.Update(InstallationStepCompleteMsg{StepIndex: 0})

		assert.Equal(t, StepComplete, m.Steps()[0].Status)
	})

	t.Run("log scrolling", func(t *testing.T) {
		styles := getIntegrationTestStyles()
		m := NewProgress(styles, "1.0.0", createTestGPUInfo(),
			DriverOption{Version: "550"},
			[]ComponentOption{})
		m.SetSize(100, 30)

		// Add log lines
		for i := 0; i < 15; i++ {
			m, _ = m.Update(InstallationLogMsg{Message: "Log line " + string(rune('A'+i))})
		}

		// Should be capped at max log lines
		assert.LessOrEqual(t, len(m.LogLines()), m.MaxLogLines())
	})

	t.Run("error display", func(t *testing.T) {
		styles := getIntegrationTestStyles()
		m := NewProgress(styles, "1.0.0", createTestGPUInfo(),
			DriverOption{Version: "550"},
			[]ComponentOption{{Name: "Driver", ID: "driver", Selected: true}})
		m.SetSize(100, 30)

		testErr := errors.New("installation failed")
		m, _ = m.Update(InstallationStepFailedMsg{StepIndex: 0, Error: testErr})

		assert.True(t, m.HasFailed())
		assert.Equal(t, testErr, m.FailureError())

		view := m.View()
		assert.Contains(t, view, "installation failed")
	})

	t.Run("installation complete", func(t *testing.T) {
		styles := getIntegrationTestStyles()
		m := NewProgress(styles, "1.0.0", createTestGPUInfo(),
			DriverOption{Version: "550"},
			[]ComponentOption{})
		m.SetSize(100, 30)

		m, _ = m.Update(InstallationCompleteMsg{})

		assert.True(t, m.IsComplete())
		assert.Equal(t, 1.0, m.Progress())
	})

	t.Run("navigate to complete on success", func(t *testing.T) {
		styles := getIntegrationTestStyles()
		m := NewProgress(styles, "1.0.0", createTestGPUInfo(),
			DriverOption{Version: "550"},
			[]ComponentOption{})
		m.SetSize(100, 30)

		m, _ = m.Update(InstallationCompleteMsg{})

		// Press q to continue
		_, cmd := m.Update(simulateRuneKey('q'))
		require.NotNil(t, cmd)

		result := cmd()
		_, ok := result.(NavigateToCompleteMsg)
		assert.True(t, ok)
	})

	t.Run("navigate to error on failure", func(t *testing.T) {
		styles := getIntegrationTestStyles()
		m := NewProgress(styles, "1.0.0", createTestGPUInfo(),
			DriverOption{Version: "550"},
			[]ComponentOption{{Name: "Driver", ID: "driver", Selected: true}})
		m.SetSize(100, 30)

		m, _ = m.Update(InstallationStepStartMsg{StepIndex: 0})
		m, _ = m.Update(InstallationStepFailedMsg{StepIndex: 0, Error: errors.New("failed")})

		// Press q to view error
		_, cmd := m.Update(simulateRuneKey('q'))
		require.NotNil(t, cmd)

		result := cmd()
		_, ok := result.(NavigateToErrorMsg)
		assert.True(t, ok)
	})
}

// =============================================================================
// TestCompleteView_Results
// =============================================================================

func TestCompleteView_Results(t *testing.T) {
	t.Run("success display", func(t *testing.T) {
		styles := getIntegrationTestStyles()
		gpuInfo := createTestGPUInfo()
		driver := DriverOption{Version: "550", Branch: "Latest"}
		components := []ComponentOption{
			{Name: "NVIDIA Driver", ID: "driver", Selected: true},
			{Name: "CUDA Toolkit", ID: "cuda", Selected: true},
		}

		m := NewComplete(styles, "1.0.0", gpuInfo, driver, components)
		m.SetSize(100, 40)

		view := m.View()
		assert.Contains(t, view, "Complete")
		assert.Contains(t, view, "550")
		assert.Contains(t, view, "NVIDIA Driver")
		assert.Contains(t, view, "CUDA Toolkit")
	})

	t.Run("reboot prompt when nouveau was loaded", func(t *testing.T) {
		styles := getIntegrationTestStyles()
		gpuInfo := createTestGPUInfoWithNouveauLoaded()
		driver := DriverOption{Version: "550"}
		components := []ComponentOption{}

		m := NewComplete(styles, "1.0.0", gpuInfo, driver, components)
		m.SetSize(100, 30)

		assert.True(t, m.NeedsReboot())

		view := m.View()
		assert.Contains(t, view, "Reboot")
	})

	t.Run("reboot button triggers reboot request", func(t *testing.T) {
		styles := getIntegrationTestStyles()
		m := NewComplete(styles, "1.0.0", createTestGPUInfo(),
			DriverOption{Version: "550"},
			[]ComponentOption{})
		m.SetSize(100, 30)

		// Reboot button should be focused
		assert.Equal(t, 0, m.FocusedButtonIndex())

		_, cmd := m.Update(simulateEnterKey())
		require.NotNil(t, cmd)

		result := cmd()
		_, ok := result.(RebootRequestedMsg)
		assert.True(t, ok)
	})

	t.Run("exit button triggers exit request", func(t *testing.T) {
		styles := getIntegrationTestStyles()
		m := NewComplete(styles, "1.0.0", createTestGPUInfo(),
			DriverOption{Version: "550"},
			[]ComponentOption{})
		m.SetSize(100, 30)

		// Navigate to Exit button
		m, _ = m.Update(simulateRightKey())
		assert.Equal(t, 1, m.FocusedButtonIndex())

		_, cmd := m.Update(simulateEnterKey())
		require.NotNil(t, cmd)

		result := cmd()
		_, ok := result.(ExitRequestedMsg)
		assert.True(t, ok)
	})

	t.Run("q key exits", func(t *testing.T) {
		styles := getIntegrationTestStyles()
		m := NewComplete(styles, "1.0.0", createTestGPUInfo(),
			DriverOption{Version: "550"},
			[]ComponentOption{})
		m.SetSize(100, 30)

		_, cmd := m.Update(simulateRuneKey('q'))
		require.NotNil(t, cmd)
	})
}

// =============================================================================
// TestErrorView_Details
// =============================================================================

func TestErrorView_Details(t *testing.T) {
	t.Run("error message display", func(t *testing.T) {
		styles := getIntegrationTestStyles()
		testErr := errors.New("package installation failed: nvidia-driver-550 not found")
		m := NewError(styles, "1.0.0", testErr, "Installing NVIDIA Driver")
		m.SetSize(100, 40)

		view := m.View()
		assert.Contains(t, view, "Installation Failed")
		assert.Contains(t, view, "package installation failed")
		assert.Contains(t, view, "Installing NVIDIA Driver")
	})

	t.Run("troubleshooting tips for install error", func(t *testing.T) {
		styles := getIntegrationTestStyles()
		m := NewError(styles, "1.0.0", errors.New("error"), "Installing driver")
		m.SetSize(100, 40)

		tips := m.TroubleshootingTips()
		assert.Greater(t, len(tips), 0)

		view := m.View()
		assert.Contains(t, view, "Troubleshooting")
	})

	t.Run("troubleshooting tips for update error", func(t *testing.T) {
		styles := getIntegrationTestStyles()
		// Note: Use "update" not "Updating" - "updating" does not contain "update"
		m := NewError(styles, "1.0.0", errors.New("network error"), "update package lists")
		m.SetSize(100, 40)

		tips := m.TroubleshootingTips()
		var hasNetworkTip bool
		for _, tip := range tips {
			if tip == "Check network connectivity" {
				hasNetworkTip = true
				break
			}
		}
		assert.True(t, hasNetworkTip)
	})

	t.Run("retry option", func(t *testing.T) {
		styles := getIntegrationTestStyles()
		m := NewError(styles, "1.0.0", errors.New("error"), "Test Step")
		m.SetSize(100, 30)

		// Retry button should be focused
		assert.Equal(t, 0, m.FocusedButtonIndex())

		_, cmd := m.Update(simulateEnterKey())
		require.NotNil(t, cmd)

		result := cmd()
		_, ok := result.(RetryRequestedMsg)
		assert.True(t, ok)
	})

	t.Run("exit option", func(t *testing.T) {
		styles := getIntegrationTestStyles()
		m := NewError(styles, "1.0.0", errors.New("error"), "Test Step")
		m.SetSize(100, 30)

		// Navigate to Exit button
		m, _ = m.Update(simulateRightKey())
		assert.Equal(t, 1, m.FocusedButtonIndex())

		_, cmd := m.Update(simulateEnterKey())
		require.NotNil(t, cmd)

		result := cmd()
		_, ok := result.(ErrorExitRequestedMsg)
		assert.True(t, ok)
	})

	t.Run("nil error shows unknown error", func(t *testing.T) {
		styles := getIntegrationTestStyles()
		m := NewError(styles, "1.0.0", nil, "")
		m.SetSize(100, 30)

		view := m.View()
		assert.Contains(t, view, "Unknown error")
	})
}

// =============================================================================
// TestUninstallViews
// =============================================================================

func TestUninstallViews(t *testing.T) {
	t.Run("uninstall confirmation view", func(t *testing.T) {
		styles := getIntegrationTestStyles()
		m := NewUninstallConfirm(styles, "1.0.0")
		m.SetSize(100, 30)

		view := m.View()
		assert.Contains(t, view, "Uninstall")
	})

	t.Run("uninstall confirmation buttons", func(t *testing.T) {
		styles := getIntegrationTestStyles()
		m := NewUninstallConfirm(styles, "1.0.0")
		m.SetSize(100, 30)

		// Should have confirm and cancel buttons
		view := m.View()
		assert.Contains(t, view, "Confirm")
		assert.Contains(t, view, "Cancel")
	})

	t.Run("uninstall progress view", func(t *testing.T) {
		styles := getIntegrationTestStyles()
		m := NewUninstallProgress(styles, "1.0.0")
		m.SetSize(100, 30)

		view := m.View()
		assert.Contains(t, view, "Uninstalling")
	})
}

// =============================================================================
// View Rendering Tests for Various Sizes
// =============================================================================

func TestViews_RenderingAtVariousSizes(t *testing.T) {
	styles := getIntegrationTestStyles()
	gpuInfo := createTestGPUInfo()
	driver := DriverOption{Version: "550"}
	components := []ComponentOption{{Name: "Driver", ID: "driver", Selected: true}}

	testSizes := []struct {
		name   string
		width  int
		height int
	}{
		{"small_40x15", 40, 15},
		{"standard_80x24", 80, 24},
		{"medium_100x30", 100, 30},
		{"wide_200x24", 200, 24},
		{"tall_80x50", 80, 50},
		{"large_150x40", 150, 40},
	}

	viewFactories := []struct {
		name   string
		create func() interface {
			SetSize(int, int)
			View() string
		}
	}{
		{
			name: "welcome",
			create: func() interface {
				SetSize(int, int)
				View() string
			} {
				return &WelcomeModelWrapper{NewWelcome(styles, "1.0.0")}
			},
		},
		{
			name: "detection",
			create: func() interface {
				SetSize(int, int)
				View() string
			} {
				return &DetectionModelWrapper{NewDetection(styles, "1.0.0")}
			},
		},
		{
			name: "selection",
			create: func() interface {
				SetSize(int, int)
				View() string
			} {
				return &SelectionModelWrapper{NewSelection(styles, "1.0.0", gpuInfo)}
			},
		},
		{
			name: "confirmation",
			create: func() interface {
				SetSize(int, int)
				View() string
			} {
				return &ConfirmationModelWrapper{NewConfirmation(styles, "1.0.0", gpuInfo, driver, components)}
			},
		},
		{
			name: "progress",
			create: func() interface {
				SetSize(int, int)
				View() string
			} {
				return &ProgressModelWrapper{NewProgress(styles, "1.0.0", gpuInfo, driver, components)}
			},
		},
		{
			name: "complete",
			create: func() interface {
				SetSize(int, int)
				View() string
			} {
				return &CompleteModelWrapper{NewComplete(styles, "1.0.0", gpuInfo, driver, components)}
			},
		},
		{
			name: "error",
			create: func() interface {
				SetSize(int, int)
				View() string
			} {
				return &ErrorModelWrapper{NewError(styles, "1.0.0", errors.New("test"), "Test")}
			},
		},
	}

	for _, viewFactory := range viewFactories {
		for _, size := range testSizes {
			t.Run(viewFactory.name+"_"+size.name, func(t *testing.T) {
				view := viewFactory.create()
				view.SetSize(size.width, size.height)

				rendered := view.View()
				assert.NotEmpty(t, rendered)
				assert.NotEqual(t, "Loading...", rendered)
			})
		}
	}
}

// Wrapper types to satisfy interface requirements
type WelcomeModelWrapper struct{ WelcomeModel }

func (w *WelcomeModelWrapper) SetSize(width, height int) { w.WelcomeModel.SetSize(width, height) }
func (w *WelcomeModelWrapper) View() string              { return w.WelcomeModel.View() }

type DetectionModelWrapper struct{ DetectionModel }

func (w *DetectionModelWrapper) SetSize(width, height int) { w.DetectionModel.SetSize(width, height) }
func (w *DetectionModelWrapper) View() string              { return w.DetectionModel.View() }

type SelectionModelWrapper struct{ SelectionModel }

func (w *SelectionModelWrapper) SetSize(width, height int) { w.SelectionModel.SetSize(width, height) }
func (w *SelectionModelWrapper) View() string              { return w.SelectionModel.View() }

type ConfirmationModelWrapper struct{ ConfirmationModel }

func (w *ConfirmationModelWrapper) SetSize(width, height int) {
	w.ConfirmationModel.SetSize(width, height)
}
func (w *ConfirmationModelWrapper) View() string { return w.ConfirmationModel.View() }

type ProgressModelWrapper struct{ ProgressModel }

func (w *ProgressModelWrapper) SetSize(width, height int) { w.ProgressModel.SetSize(width, height) }
func (w *ProgressModelWrapper) View() string              { return w.ProgressModel.View() }

type CompleteModelWrapper struct{ CompleteModel }

func (w *CompleteModelWrapper) SetSize(width, height int) { w.CompleteModel.SetSize(width, height) }
func (w *CompleteModelWrapper) View() string              { return w.CompleteModel.View() }

type ErrorModelWrapper struct{ ErrorModel }

func (w *ErrorModelWrapper) SetSize(width, height int) { w.ErrorModel.SetSize(width, height) }
func (w *ErrorModelWrapper) View() string              { return w.ErrorModel.View() }

// =============================================================================
// Step and Status Integration Tests
// =============================================================================

func TestInstallationStep_IntegrationDuration(t *testing.T) {
	t.Run("duration calculation with timing", func(t *testing.T) {
		step := InstallationStep{
			Name:      "test",
			StartTime: time.Now().Add(-5 * time.Second),
			EndTime:   time.Now(),
		}

		duration := step.Duration()
		assert.InDelta(t, 5.0, duration.Seconds(), 0.5)
	})

	t.Run("zero duration when times not set", func(t *testing.T) {
		step := InstallationStep{Name: "test"}
		assert.Equal(t, time.Duration(0), step.Duration())
	})

	t.Run("step state checks", func(t *testing.T) {
		step := InstallationStep{Name: "test", Status: StepRunning}
		assert.True(t, step.IsRunning())
		assert.False(t, step.IsDone())

		step.Status = StepComplete
		assert.False(t, step.IsRunning())
		assert.True(t, step.IsDone())
	})
}

func TestStepStatus_IntegrationString(t *testing.T) {
	// Test all status values render to non-empty strings
	statuses := []StepStatus{StepPending, StepRunning, StepComplete, StepFailed, StepSkipped}

	for _, status := range statuses {
		t.Run(status.String(), func(t *testing.T) {
			assert.NotEmpty(t, status.String())
		})
	}
}

func TestDetectionState_IntegrationString(t *testing.T) {
	// Test all states render to non-empty strings
	states := []DetectionState{StateDetecting, StateComplete, StateError}

	for _, state := range states {
		t.Run(state.String(), func(t *testing.T) {
			assert.NotEmpty(t, state.String())
		})
	}
}

// =============================================================================
// Benchmarks
// =============================================================================

func BenchmarkView_Render(b *testing.B) {
	styles := getIntegrationTestStyles()
	gpuInfo := createTestGPUInfo()
	driver := DriverOption{Version: "550"}
	components := []ComponentOption{{Name: "Driver", ID: "driver", Selected: true}}

	views := []struct {
		name string
		view func() string
	}{
		{"welcome", func() string {
			m := NewWelcome(styles, "1.0.0")
			m.SetSize(100, 30)
			return m.View()
		}},
		{"detection", func() string {
			m := NewDetection(styles, "1.0.0")
			m.SetSize(100, 30)
			return m.View()
		}},
		{"selection", func() string {
			m := NewSelection(styles, "1.0.0", gpuInfo)
			m.SetSize(100, 30)
			return m.View()
		}},
		{"confirmation", func() string {
			m := NewConfirmation(styles, "1.0.0", gpuInfo, driver, components)
			m.SetSize(100, 30)
			return m.View()
		}},
		{"progress", func() string {
			m := NewProgress(styles, "1.0.0", gpuInfo, driver, components)
			m.SetSize(100, 30)
			return m.View()
		}},
		{"complete", func() string {
			m := NewComplete(styles, "1.0.0", gpuInfo, driver, components)
			m.SetSize(100, 30)
			return m.View()
		}},
		{"error", func() string {
			m := NewError(styles, "1.0.0", errors.New("test error"), "Test Step")
			m.SetSize(100, 30)
			return m.View()
		}},
	}

	for _, v := range views {
		b.Run(v.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_ = v.view()
			}
		})
	}
}
