package ui

import (
	"context"
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
	"github.com/tungetti/igor/internal/ui/views"
)

// =============================================================================
// Test Helpers for Bubble Tea Integration Testing
// =============================================================================

// simulateKeyPress simulates a key press by returning a KeyMsg.
func simulateKeyPress(key string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}
}

// simulateEnter simulates the Enter key press.
func simulateEnter() tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyEnter}
}

// simulateEscape simulates the Escape key press.
func simulateEscape() tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyEsc}
}

// simulateUp simulates the Up arrow key press.
func simulateUp() tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyUp}
}

// simulateDown simulates the Down arrow key press.
func simulateDown() tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyDown}
}

// simulateLeft simulates the Left arrow key press.
func simulateLeft() tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyLeft}
}

// simulateRight simulates the Right arrow key press.
func simulateRight() tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRight}
}

// simulateTab simulates the Tab key press.
func simulateTab() tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyTab}
}

// simulateCtrlC simulates the Ctrl+C key combination.
func simulateCtrlC() tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyCtrlC}
}

// simulateWindowSize simulates a window resize event.
func simulateWindowSize(width, height int) tea.WindowSizeMsg {
	return tea.WindowSizeMsg{Width: width, Height: height}
}

// updateModel is a helper that updates a model and returns the updated model.
func updateModel(m Model, msg tea.Msg) Model {
	newModel, _ := m.Update(msg)
	return newModel.(Model)
}

// createTestGPUInfo creates a test GPU info structure.
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

// =============================================================================
// TestApp_CompleteUserFlow - Complete Installation User Flow
// =============================================================================

func TestApp_CompleteUserFlow(t *testing.T) {
	t.Run("full happy path from welcome to complete", func(t *testing.T) {
		m := NewWithVersion("1.0.0")

		// Step 1: Initialize with window size (Welcome screen)
		m = updateModel(m, simulateWindowSize(100, 30))
		assert.Equal(t, ViewWelcome, m.CurrentView)
		assert.True(t, m.Ready)
		view := m.View()
		assert.Contains(t, view, "IGOR")
		assert.Contains(t, view, "Start Installation")

		// Step 2: Navigate to detection (user presses Enter on Start)
		newModel, _ := m.Update(views.StartDetectionMsg{})
		m = newModel.(Model)
		assert.Equal(t, ViewDetecting, m.CurrentView)
		view = m.View()
		assert.Contains(t, view, "Detecting")

		// Step 3: Detection completes - navigate to driver selection
		gpuInfo := createTestGPUInfo()
		newModel, _ = m.Update(views.NavigateToDriverSelectionMsg{GPUInfo: gpuInfo})
		m = newModel.(Model)
		assert.Equal(t, ViewDriverSelection, m.CurrentView)
		assert.Equal(t, gpuInfo, m.gpuInfo)
		view = m.View()
		assert.Contains(t, view, "Driver")

		// Step 4: User selects driver and navigates to confirmation
		driver := views.DriverOption{Version: "550", Branch: "Latest", Recommended: true}
		components := []views.ComponentOption{
			{Name: "NVIDIA Driver", ID: "driver", Selected: true, Required: true},
			{Name: "NVIDIA Settings", ID: "settings", Selected: true},
		}
		newModel, _ = m.Update(views.NavigateToConfirmationMsg{
			GPUInfo:            gpuInfo,
			SelectedDriver:     driver,
			SelectedComponents: components,
		})
		m = newModel.(Model)
		assert.Equal(t, ViewConfirmation, m.CurrentView)
		assert.Equal(t, driver, m.driver)
		assert.Equal(t, components, m.components)
		view = m.View()
		assert.Contains(t, view, "Installation Summary")

		// Step 5: User confirms - start installation
		newModel, cmd := m.Update(views.StartInstallationMsg{
			GPUInfo:    gpuInfo,
			Driver:     driver,
			Components: components,
		})
		m = newModel.(Model)
		assert.Equal(t, ViewInstalling, m.CurrentView)
		assert.NotNil(t, cmd) // Should return spinner init command
		view = m.View()
		assert.Contains(t, view, "Installing")

		// Step 6: Installation completes - navigate to completion
		newModel, _ = m.Update(views.NavigateToCompleteMsg{
			GPUInfo:    gpuInfo,
			Driver:     driver,
			Components: components,
		})
		m = newModel.(Model)
		assert.Equal(t, ViewComplete, m.CurrentView)
		view = m.View()
		assert.Contains(t, view, "Complete")
	})

	t.Run("user flow with error during installation", func(t *testing.T) {
		m := NewWithVersion("1.0.0")

		// Initialize
		m = updateModel(m, simulateWindowSize(100, 30))

		// Navigate through to installation
		gpuInfo := createTestGPUInfo()
		driver := views.DriverOption{Version: "550", Branch: "Latest"}
		components := []views.ComponentOption{{Name: "Driver", ID: "driver", Selected: true}}

		m = updateModel(m, views.StartDetectionMsg{})
		m = updateModel(m, views.NavigateToDriverSelectionMsg{GPUInfo: gpuInfo})
		m = updateModel(m, views.NavigateToConfirmationMsg{
			GPUInfo:            gpuInfo,
			SelectedDriver:     driver,
			SelectedComponents: components,
		})
		m = updateModel(m, views.StartInstallationMsg{
			GPUInfo:    gpuInfo,
			Driver:     driver,
			Components: components,
		})
		assert.Equal(t, ViewInstalling, m.CurrentView)

		// Installation fails - navigate to error
		installErr := errors.New("package installation failed: nvidia-driver-550")
		newModel, _ := m.Update(views.NavigateToErrorMsg{
			Error:      installErr,
			FailedStep: "Installing NVIDIA Driver",
		})
		m = newModel.(Model)
		assert.Equal(t, ViewError, m.CurrentView)
		assert.Equal(t, installErr, m.Error)

		view := m.View()
		assert.Contains(t, view, "Installation Failed")
		assert.Contains(t, view, "package installation failed")
	})

	t.Run("user retries after error", func(t *testing.T) {
		m := NewWithVersion("1.0.0")
		m = updateModel(m, simulateWindowSize(100, 30))

		// Navigate to error state
		m = updateModel(m, views.NavigateToErrorMsg{
			Error:      errors.New("test error"),
			FailedStep: "Test Step",
		})
		assert.Equal(t, ViewError, m.CurrentView)

		// User requests retry
		newModel, _ := m.Update(views.RetryRequestedMsg{})
		m = newModel.(Model)
		assert.Equal(t, ViewWelcome, m.CurrentView)
		assert.Nil(t, m.Error)
	})
}

// =============================================================================
// TestApp_NavigationFlow - Navigation Between Views
// =============================================================================

func TestApp_NavigationFlow(t *testing.T) {
	t.Run("forward navigation through all views", func(t *testing.T) {
		m := NewWithVersion("1.0.0")
		m = updateModel(m, simulateWindowSize(100, 30))

		gpuInfo := createTestGPUInfo()
		driver := views.DriverOption{Version: "550", Branch: "Latest"}
		components := []views.ComponentOption{{Name: "Driver", ID: "driver", Selected: true}}

		// Welcome -> Detecting
		assert.Equal(t, ViewWelcome, m.CurrentView)
		m = updateModel(m, views.StartDetectionMsg{})
		assert.Equal(t, ViewDetecting, m.CurrentView)

		// Detecting -> DriverSelection
		m = updateModel(m, views.NavigateToDriverSelectionMsg{GPUInfo: gpuInfo})
		assert.Equal(t, ViewDriverSelection, m.CurrentView)

		// DriverSelection -> Confirmation
		m = updateModel(m, views.NavigateToConfirmationMsg{
			GPUInfo:            gpuInfo,
			SelectedDriver:     driver,
			SelectedComponents: components,
		})
		assert.Equal(t, ViewConfirmation, m.CurrentView)

		// Confirmation -> Installing
		m = updateModel(m, views.StartInstallationMsg{
			GPUInfo:    gpuInfo,
			Driver:     driver,
			Components: components,
		})
		assert.Equal(t, ViewInstalling, m.CurrentView)

		// Installing -> Complete
		m = updateModel(m, views.NavigateToCompleteMsg{
			GPUInfo:    gpuInfo,
			Driver:     driver,
			Components: components,
		})
		assert.Equal(t, ViewComplete, m.CurrentView)
	})

	t.Run("backward navigation with escape key", func(t *testing.T) {
		m := NewWithVersion("1.0.0")
		m = updateModel(m, simulateWindowSize(100, 30))

		// Navigate forward
		gpuInfo := createTestGPUInfo()
		m = updateModel(m, views.StartDetectionMsg{})
		m = updateModel(m, views.NavigateToDriverSelectionMsg{GPUInfo: gpuInfo})
		assert.Equal(t, ViewDriverSelection, m.CurrentView)

		// Press escape to go back to welcome
		m = updateModel(m, simulateEscape())
		assert.Equal(t, ViewWelcome, m.CurrentView)
	})

	t.Run("NavigateBackToSelectionMsg returns to selection", func(t *testing.T) {
		m := NewWithVersion("1.0.0")
		m = updateModel(m, simulateWindowSize(100, 30))

		gpuInfo := createTestGPUInfo()
		driver := views.DriverOption{Version: "550"}
		components := []views.ComponentOption{{Name: "Driver", ID: "driver", Selected: true}}

		// Go to selection first
		m = updateModel(m, views.StartDetectionMsg{})
		m = updateModel(m, views.NavigateToDriverSelectionMsg{GPUInfo: gpuInfo})
		assert.Equal(t, ViewDriverSelection, m.CurrentView)

		// Go to confirmation
		m = updateModel(m, views.NavigateToConfirmationMsg{
			GPUInfo:            gpuInfo,
			SelectedDriver:     driver,
			SelectedComponents: components,
		})
		assert.Equal(t, ViewConfirmation, m.CurrentView)

		// Navigate back to selection
		m = updateModel(m, views.NavigateBackToSelectionMsg{})
		assert.Equal(t, ViewDriverSelection, m.CurrentView)
	})

	t.Run("escape from welcome triggers quit", func(t *testing.T) {
		m := NewWithVersion("1.0.0")
		m = updateModel(m, simulateWindowSize(100, 30))
		assert.Equal(t, ViewWelcome, m.CurrentView)

		// Press escape from welcome
		_, cmd := m.Update(simulateEscape())
		require.NotNil(t, cmd)

		// Command should return QuitMsg
		result := cmd()
		_, ok := result.(QuitMsg)
		assert.True(t, ok)
	})
}

// =============================================================================
// TestApp_KeyboardInteraction - Keyboard Controls
// =============================================================================

func TestApp_KeyboardInteraction(t *testing.T) {
	t.Run("q key triggers quit", func(t *testing.T) {
		m := NewWithVersion("1.0.0")
		m = updateModel(m, simulateWindowSize(100, 30))

		_, cmd := m.Update(simulateKeyPress("q"))
		require.NotNil(t, cmd)

		result := cmd()
		_, ok := result.(QuitMsg)
		assert.True(t, ok)
	})

	t.Run("ctrl+c triggers quit", func(t *testing.T) {
		m := NewWithVersion("1.0.0")
		m = updateModel(m, simulateWindowSize(100, 30))

		_, cmd := m.Update(simulateCtrlC())
		require.NotNil(t, cmd)

		result := cmd()
		_, ok := result.(QuitMsg)
		assert.True(t, ok)
	})

	t.Run("navigation keys are delegated to active view", func(t *testing.T) {
		m := NewWithVersion("1.0.0")
		m = updateModel(m, simulateWindowSize(100, 30))

		// Navigate to selection view
		gpuInfo := createTestGPUInfo()
		m = updateModel(m, views.StartDetectionMsg{})
		m = updateModel(m, views.NavigateToDriverSelectionMsg{GPUInfo: gpuInfo})

		// Arrow keys should be delegated to selection view without panicking
		// and the view should still render correctly
		assert.Equal(t, ViewDriverSelection, m.CurrentView)
		initialView := m.View()

		m = updateModel(m, simulateDown())
		newView := m.View()

		// View should update (selection may or may not change depending on available options)
		assert.NotEmpty(t, newView)
		assert.NotEmpty(t, initialView)
	})

	t.Run("tab key for focus switching", func(t *testing.T) {
		m := NewWithVersion("1.0.0")
		m = updateModel(m, simulateWindowSize(100, 30))

		// Navigate to selection view
		gpuInfo := createTestGPUInfo()
		m = updateModel(m, views.StartDetectionMsg{})
		m = updateModel(m, views.NavigateToDriverSelectionMsg{GPUInfo: gpuInfo})

		// Tab should be processed without panicking and the view should update
		assert.Equal(t, ViewDriverSelection, m.CurrentView)
		initialView := m.View()

		m = updateModel(m, simulateTab())
		newView := m.View()

		// View should continue to render correctly
		assert.NotEmpty(t, newView)
		assert.NotEmpty(t, initialView)
	})

	t.Run("space key toggles selection", func(t *testing.T) {
		m := NewWithVersion("1.0.0")
		m = updateModel(m, simulateWindowSize(100, 30))

		// Navigate to selection view and switch to components section
		gpuInfo := createTestGPUInfo()
		m = updateModel(m, views.StartDetectionMsg{})
		m = updateModel(m, views.NavigateToDriverSelectionMsg{GPUInfo: gpuInfo})

		// Switch to components section and navigate
		m = updateModel(m, simulateTab())
		m = updateModel(m, simulateDown())

		// Space key should be processed without panicking
		spaceMsg := tea.KeyMsg{Type: tea.KeySpace}
		m = updateModel(m, spaceMsg)

		// View should continue to render correctly
		view := m.View()
		assert.NotEmpty(t, view)
		assert.Equal(t, ViewDriverSelection, m.CurrentView)
	})
}

// =============================================================================
// TestApp_ErrorRecovery - Error Handling in UI
// =============================================================================

func TestApp_ErrorRecovery(t *testing.T) {
	t.Run("displays errors gracefully", func(t *testing.T) {
		m := NewWithVersion("1.0.0")
		m = updateModel(m, simulateWindowSize(100, 30))

		testErr := errors.New("connection timeout: unable to reach package server")
		m = updateModel(m, views.NavigateToErrorMsg{
			Error:      testErr,
			FailedStep: "Updating package lists",
		})

		assert.Equal(t, ViewError, m.CurrentView)
		view := m.View()
		assert.Contains(t, view, "Installation Failed")
		assert.Contains(t, view, "connection timeout")
		assert.Contains(t, view, "Updating package lists")
	})

	t.Run("nil error shows unknown error", func(t *testing.T) {
		m := NewWithVersion("1.0.0")
		m = updateModel(m, simulateWindowSize(100, 30))

		m = updateModel(m, views.NavigateToErrorMsg{
			Error:      nil,
			FailedStep: "",
		})

		view := m.View()
		assert.Contains(t, view, "Unknown error")
	})

	t.Run("retry navigates back to welcome", func(t *testing.T) {
		m := NewWithVersion("1.0.0")
		m = updateModel(m, simulateWindowSize(100, 30))

		m = updateModel(m, views.NavigateToErrorMsg{
			Error:      errors.New("test error"),
			FailedStep: "Test Step",
		})
		assert.Equal(t, ViewError, m.CurrentView)

		m = updateModel(m, views.RetryRequestedMsg{})
		assert.Equal(t, ViewWelcome, m.CurrentView)
		assert.Nil(t, m.Error)
	})

	t.Run("error exit quits application", func(t *testing.T) {
		m := NewWithVersion("1.0.0")
		m = updateModel(m, simulateWindowSize(100, 30))

		m = updateModel(m, views.NavigateToErrorMsg{
			Error:      errors.New("fatal error"),
			FailedStep: "Critical Step",
		})

		newModel, cmd := m.Update(views.ErrorExitRequestedMsg{})
		m = newModel.(Model)

		assert.True(t, m.Quitting)
		assert.NotNil(t, cmd)
	})

	t.Run("ErrorMsg transitions to error view", func(t *testing.T) {
		m := NewWithVersion("1.0.0")
		m = updateModel(m, simulateWindowSize(100, 30))

		testErr := errors.New("generic error")
		m = updateModel(m, ErrorMsg{Err: testErr})

		assert.Equal(t, ViewError, m.CurrentView)
		assert.Equal(t, testErr, m.Error)
	})
}

// =============================================================================
// TestApp_ResizeHandling - Terminal Resize
// =============================================================================

func TestApp_ResizeHandling(t *testing.T) {
	terminalSizes := []struct {
		name   string
		width  int
		height int
	}{
		{"minimum_80x24", 80, 24},
		{"standard_100x30", 100, 30},
		{"wide_200x24", 200, 24},
		{"tall_80x60", 80, 60},
		{"very_large_250x80", 250, 80},
		{"small_40x15", 40, 15},
		{"very_small_20x10", 20, 10},
	}

	for _, tc := range terminalSizes {
		t.Run(tc.name, func(t *testing.T) {
			m := NewWithVersion("1.0.0")

			// Set initial size
			m = updateModel(m, simulateWindowSize(tc.width, tc.height))

			assert.Equal(t, tc.width, m.Width)
			assert.Equal(t, tc.height, m.Height)
			assert.True(t, m.Ready)

			// View should render without panic
			view := m.View()
			assert.NotEmpty(t, view)
		})
	}

	t.Run("resize during navigation", func(t *testing.T) {
		m := NewWithVersion("1.0.0")
		m = updateModel(m, simulateWindowSize(100, 30))

		gpuInfo := createTestGPUInfo()
		m = updateModel(m, views.StartDetectionMsg{})
		m = updateModel(m, views.NavigateToDriverSelectionMsg{GPUInfo: gpuInfo})

		// Resize while in selection view
		m = updateModel(m, simulateWindowSize(150, 40))
		assert.Equal(t, 150, m.Width)
		assert.Equal(t, 40, m.Height)
		assert.Equal(t, 150, m.selectionView.Width())
		assert.Equal(t, 40, m.selectionView.Height())

		// View should still render correctly
		view := m.View()
		assert.NotEmpty(t, view)
		assert.Contains(t, view, "Driver")
	})

	t.Run("multiple rapid resizes", func(t *testing.T) {
		m := NewWithVersion("1.0.0")

		sizes := []tea.WindowSizeMsg{
			{Width: 80, Height: 24},
			{Width: 100, Height: 30},
			{Width: 50, Height: 20},
			{Width: 200, Height: 50},
			{Width: 80, Height: 24},
		}

		for _, size := range sizes {
			m = updateModel(m, size)
			assert.Equal(t, size.Width, m.Width)
			assert.Equal(t, size.Height, m.Height)

			// View should render without panic
			view := m.View()
			assert.NotEmpty(t, view)
		}
	})

	t.Run("resize propagates to all views", func(t *testing.T) {
		m := NewWithVersion("1.0.0")
		m = updateModel(m, simulateWindowSize(100, 30))

		gpuInfo := createTestGPUInfo()
		driver := views.DriverOption{Version: "550", Branch: "Latest"}
		components := []views.ComponentOption{{Name: "Driver", ID: "driver", Selected: true}}

		// Initialize all views by navigating through them
		m = updateModel(m, views.StartDetectionMsg{})
		m = updateModel(m, views.NavigateToDriverSelectionMsg{GPUInfo: gpuInfo})
		m = updateModel(m, views.NavigateToConfirmationMsg{
			GPUInfo:            gpuInfo,
			SelectedDriver:     driver,
			SelectedComponents: components,
		})

		// Resize
		m = updateModel(m, simulateWindowSize(120, 40))

		// Check welcome view (always initialized)
		assert.Equal(t, 120, m.welcomeView.Width())
		assert.Equal(t, 40, m.welcomeView.Height())

		// Check confirmation view (currently active)
		assert.Equal(t, 120, m.confirmationView.Width())
		assert.Equal(t, 40, m.confirmationView.Height())
	})
}

// =============================================================================
// TestApp_SpinnerHandling - Spinner Animation
// =============================================================================

func TestApp_SpinnerHandling(t *testing.T) {
	t.Run("spinner ticks are delegated to detection view", func(t *testing.T) {
		m := NewWithVersion("1.0.0")
		m = updateModel(m, simulateWindowSize(100, 30))

		m = updateModel(m, views.StartDetectionMsg{})
		assert.Equal(t, ViewDetecting, m.CurrentView)

		// Spinner tick should be handled without error
		tickMsg := spinner.TickMsg{}
		newModel, cmd := m.Update(tickMsg)
		m = newModel.(Model)

		// Should still be in detecting view
		assert.Equal(t, ViewDetecting, m.CurrentView)
		_ = cmd // cmd may be nil or another tick command
	})

	t.Run("spinner ticks are delegated to progress view", func(t *testing.T) {
		m := NewWithVersion("1.0.0")
		m = updateModel(m, simulateWindowSize(100, 30))

		gpuInfo := createTestGPUInfo()
		driver := views.DriverOption{Version: "550"}
		components := []views.ComponentOption{{Name: "Driver", ID: "driver", Selected: true}}

		m = updateModel(m, views.StartInstallationMsg{
			GPUInfo:    gpuInfo,
			Driver:     driver,
			Components: components,
		})
		assert.Equal(t, ViewInstalling, m.CurrentView)

		// Spinner tick should be handled without error
		tickMsg := spinner.TickMsg{}
		newModel, _ := m.Update(tickMsg)
		m = newModel.(Model)

		assert.Equal(t, ViewInstalling, m.CurrentView)
	})
}

// =============================================================================
// TestApp_ContextManagement - Context Cancellation
// =============================================================================

func TestApp_ContextManagement(t *testing.T) {
	t.Run("context is cancelled on quit", func(t *testing.T) {
		m := NewWithVersion("1.0.0")
		m = updateModel(m, simulateWindowSize(100, 30))

		// Context should not be cancelled initially
		select {
		case <-m.ctx.Done():
			t.Fatal("Context should not be cancelled initially")
		default:
			// Expected
		}

		// Quit
		m = updateModel(m, QuitMsg{})

		// Context should be cancelled after quit
		select {
		case <-m.ctx.Done():
			// Expected
		default:
			t.Fatal("Context should be cancelled after quit")
		}
	})

	t.Run("parent context cancellation propagates", func(t *testing.T) {
		parentCtx, parentCancel := context.WithCancel(context.Background())
		defer parentCancel()

		m := NewWithContext(parentCtx)

		// Cancel parent
		parentCancel()

		// Child context should be cancelled
		select {
		case <-m.ctx.Done():
			// Expected
		case <-time.After(100 * time.Millisecond):
			t.Fatal("Context should be cancelled when parent is cancelled")
		}
	})

	t.Run("shutdown cancels context", func(t *testing.T) {
		m := NewWithVersion("1.0.0")

		// Context should not be cancelled
		assert.NoError(t, m.ctx.Err())

		m.Shutdown()

		// Context should be cancelled
		assert.Error(t, m.ctx.Err())
	})
}

// =============================================================================
// TestApp_ViewDelegation - Message Delegation to Active View
// =============================================================================

func TestApp_ViewDelegation(t *testing.T) {
	testCases := []struct {
		name       string
		setupView  func(m Model) Model
		view       ViewState
		assertView func(t *testing.T, m Model)
	}{
		{
			name: "welcome view",
			setupView: func(m Model) Model {
				return updateModel(m, simulateWindowSize(100, 30))
			},
			view: ViewWelcome,
			assertView: func(t *testing.T, m Model) {
				assert.True(t, m.welcomeView.Ready())
			},
		},
		{
			name: "detection view",
			setupView: func(m Model) Model {
				m = updateModel(m, simulateWindowSize(100, 30))
				return updateModel(m, views.StartDetectionMsg{})
			},
			view: ViewDetecting,
			assertView: func(t *testing.T, m Model) {
				assert.True(t, m.detectionView.Ready())
			},
		},
		{
			name: "selection view",
			setupView: func(m Model) Model {
				m = updateModel(m, simulateWindowSize(100, 30))
				m = updateModel(m, views.StartDetectionMsg{})
				return updateModel(m, views.NavigateToDriverSelectionMsg{GPUInfo: createTestGPUInfo()})
			},
			view: ViewDriverSelection,
			assertView: func(t *testing.T, m Model) {
				assert.True(t, m.selectionView.Ready())
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			m := NewWithVersion("1.0.0")
			m = tc.setupView(m)
			assert.Equal(t, tc.view, m.CurrentView)
			tc.assertView(t, m)
		})
	}
}

// =============================================================================
// TestApp_ExitScenarios - Various Exit Scenarios
// =============================================================================

func TestApp_ExitScenarios(t *testing.T) {
	t.Run("reboot requested from complete view", func(t *testing.T) {
		m := NewWithVersion("1.0.0")
		m = updateModel(m, simulateWindowSize(100, 30))

		gpuInfo := createTestGPUInfo()
		driver := views.DriverOption{Version: "550"}
		components := []views.ComponentOption{{Name: "Driver", ID: "driver", Selected: true}}

		m = updateModel(m, views.NavigateToCompleteMsg{
			GPUInfo:    gpuInfo,
			Driver:     driver,
			Components: components,
		})
		assert.Equal(t, ViewComplete, m.CurrentView)

		newModel, cmd := m.Update(views.RebootRequestedMsg{})
		m = newModel.(Model)

		assert.True(t, m.Quitting)
		assert.NotNil(t, cmd)
	})

	t.Run("exit requested from complete view", func(t *testing.T) {
		m := NewWithVersion("1.0.0")
		m = updateModel(m, simulateWindowSize(100, 30))

		m = updateModel(m, views.NavigateToCompleteMsg{
			GPUInfo:    createTestGPUInfo(),
			Driver:     views.DriverOption{Version: "550"},
			Components: []views.ComponentOption{},
		})

		newModel, cmd := m.Update(views.ExitRequestedMsg{})
		m = newModel.(Model)

		assert.True(t, m.Quitting)
		assert.NotNil(t, cmd)
	})

	t.Run("quit view rendering", func(t *testing.T) {
		m := NewWithVersion("1.0.0")
		m = updateModel(m, simulateWindowSize(100, 30))
		m.Quitting = true

		view := m.View()
		assert.Equal(t, "Goodbye!\n", view)
	})
}

// =============================================================================
// Benchmarks
// =============================================================================

func BenchmarkApp_Init(b *testing.B) {
	for i := 0; i < b.N; i++ {
		m := NewWithVersion("1.0.0")
		_ = m.Init()
	}
}

func BenchmarkApp_Update(b *testing.B) {
	m := NewWithVersion("1.0.0")
	m = updateModel(m, simulateWindowSize(100, 30))
	msg := simulateKeyPress("j")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = m.Update(msg)
	}
}

func BenchmarkApp_View(b *testing.B) {
	m := NewWithVersion("1.0.0")
	m = updateModel(m, simulateWindowSize(100, 30))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = m.View()
	}
}

func BenchmarkApp_NavigationFlow(b *testing.B) {
	gpuInfo := createTestGPUInfo()
	driver := views.DriverOption{Version: "550", Branch: "Latest"}
	components := []views.ComponentOption{{Name: "Driver", ID: "driver", Selected: true}}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m := NewWithVersion("1.0.0")
		m = updateModel(m, simulateWindowSize(100, 30))
		m = updateModel(m, views.StartDetectionMsg{})
		m = updateModel(m, views.NavigateToDriverSelectionMsg{GPUInfo: gpuInfo})
		m = updateModel(m, views.NavigateToConfirmationMsg{
			GPUInfo:            gpuInfo,
			SelectedDriver:     driver,
			SelectedComponents: components,
		})
		m = updateModel(m, views.StartInstallationMsg{
			GPUInfo:    gpuInfo,
			Driver:     driver,
			Components: components,
		})
		m = updateModel(m, views.NavigateToCompleteMsg{
			GPUInfo:    gpuInfo,
			Driver:     driver,
			Components: components,
		})
	}
}

func BenchmarkApp_Resize(b *testing.B) {
	m := NewWithVersion("1.0.0")
	m = updateModel(m, simulateWindowSize(100, 30))

	sizes := []tea.WindowSizeMsg{
		{Width: 80, Height: 24},
		{Width: 120, Height: 40},
		{Width: 60, Height: 20},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		size := sizes[i%len(sizes)]
		m = updateModel(m, size)
	}
}
