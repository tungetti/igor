package views

import (
	"testing"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// DriverOption Tests
// =============================================================================

func TestDriverOption_Struct(t *testing.T) {
	opt := DriverOption{
		Version:     "550",
		Branch:      "Latest",
		Description: "Latest features",
		Recommended: true,
	}

	assert.Equal(t, "550", opt.Version)
	assert.Equal(t, "Latest", opt.Branch)
	assert.Equal(t, "Latest features", opt.Description)
	assert.True(t, opt.Recommended)
}

// =============================================================================
// ComponentOption Tests
// =============================================================================

func TestComponentOption_Struct(t *testing.T) {
	comp := ComponentOption{
		Name:        "CUDA Toolkit",
		ID:          "cuda",
		Description: "GPU computing platform",
		Selected:    true,
		Required:    false,
	}

	assert.Equal(t, "CUDA Toolkit", comp.Name)
	assert.Equal(t, "cuda", comp.ID)
	assert.Equal(t, "GPU computing platform", comp.Description)
	assert.True(t, comp.Selected)
	assert.False(t, comp.Required)
}

// =============================================================================
// SelectionKeyMap Tests
// =============================================================================

func TestDefaultSelectionKeyMap(t *testing.T) {
	km := DefaultSelectionKeyMap()

	// Verify all key bindings are set
	assert.NotEmpty(t, km.Up.Keys())
	assert.NotEmpty(t, km.Down.Keys())
	assert.NotEmpty(t, km.Left.Keys())
	assert.NotEmpty(t, km.Right.Keys())
	assert.NotEmpty(t, km.Select.Keys())
	assert.NotEmpty(t, km.Continue.Keys())
	assert.NotEmpty(t, km.Back.Keys())
	assert.NotEmpty(t, km.Quit.Keys())
	assert.NotEmpty(t, km.Tab.Keys())
	assert.NotEmpty(t, km.Help.Keys())
}

func TestSelectionKeyMap_Up(t *testing.T) {
	km := DefaultSelectionKeyMap()

	assert.Contains(t, km.Up.Keys(), "up")
	assert.Contains(t, km.Up.Keys(), "k")
}

func TestSelectionKeyMap_Down(t *testing.T) {
	km := DefaultSelectionKeyMap()

	assert.Contains(t, km.Down.Keys(), "down")
	assert.Contains(t, km.Down.Keys(), "j")
}

func TestSelectionKeyMap_Left(t *testing.T) {
	km := DefaultSelectionKeyMap()

	assert.Contains(t, km.Left.Keys(), "left")
	assert.Contains(t, km.Left.Keys(), "h")
}

func TestSelectionKeyMap_Right(t *testing.T) {
	km := DefaultSelectionKeyMap()

	assert.Contains(t, km.Right.Keys(), "right")
	assert.Contains(t, km.Right.Keys(), "l")
}

func TestSelectionKeyMap_Select(t *testing.T) {
	km := DefaultSelectionKeyMap()

	assert.Contains(t, km.Select.Keys(), " ")
}

func TestSelectionKeyMap_Continue(t *testing.T) {
	km := DefaultSelectionKeyMap()

	assert.Contains(t, km.Continue.Keys(), "enter")
}

func TestSelectionKeyMap_Back(t *testing.T) {
	km := DefaultSelectionKeyMap()

	assert.Contains(t, km.Back.Keys(), "esc")
}

func TestSelectionKeyMap_Quit(t *testing.T) {
	km := DefaultSelectionKeyMap()

	assert.Contains(t, km.Quit.Keys(), "q")
	assert.Contains(t, km.Quit.Keys(), "ctrl+c")
}

func TestSelectionKeyMap_Tab(t *testing.T) {
	km := DefaultSelectionKeyMap()

	assert.Contains(t, km.Tab.Keys(), "tab")
}

func TestSelectionKeyMap_Help(t *testing.T) {
	km := DefaultSelectionKeyMap()

	assert.Contains(t, km.Help.Keys(), "?")
}

func TestSelectionKeyMap_ShortHelp(t *testing.T) {
	km := DefaultSelectionKeyMap()

	shortHelp := km.ShortHelp()

	assert.Len(t, shortHelp, 5)
	assert.Equal(t, km.Up, shortHelp[0])
	assert.Equal(t, km.Down, shortHelp[1])
	assert.Equal(t, km.Select, shortHelp[2])
	assert.Equal(t, km.Continue, shortHelp[3])
	assert.Equal(t, km.Back, shortHelp[4])
}

func TestSelectionKeyMap_FullHelp(t *testing.T) {
	km := DefaultSelectionKeyMap()

	fullHelp := km.FullHelp()

	assert.Len(t, fullHelp, 3)

	// First row: navigation
	assert.Len(t, fullHelp[0], 4)
	assert.Equal(t, km.Up, fullHelp[0][0])
	assert.Equal(t, km.Down, fullHelp[0][1])
	assert.Equal(t, km.Left, fullHelp[0][2])
	assert.Equal(t, km.Right, fullHelp[0][3])

	// Second row: actions
	assert.Len(t, fullHelp[1], 4)
	assert.Equal(t, km.Select, fullHelp[1][0])
	assert.Equal(t, km.Tab, fullHelp[1][1])
	assert.Equal(t, km.Continue, fullHelp[1][2])
	assert.Equal(t, km.Back, fullHelp[1][3])

	// Third row: quit and help
	assert.Len(t, fullHelp[2], 2)
	assert.Equal(t, km.Quit, fullHelp[2][0])
	assert.Equal(t, km.Help, fullHelp[2][1])
}

func TestSelectionKeyMap_ImplementsHelpKeyMap(t *testing.T) {
	km := DefaultSelectionKeyMap()

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

func TestSelectionKeyMap_BindingsHaveHelp(t *testing.T) {
	km := DefaultSelectionKeyMap()

	bindings := []struct {
		name    string
		binding key.Binding
	}{
		{"Up", km.Up},
		{"Down", km.Down},
		{"Left", km.Left},
		{"Right", km.Right},
		{"Select", km.Select},
		{"Continue", km.Continue},
		{"Back", km.Back},
		{"Quit", km.Quit},
		{"Tab", km.Tab},
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
// BuildDriverOptions Tests
// =============================================================================

func TestBuildDriverOptions(t *testing.T) {
	gpuInfo := createMockGPUInfo()
	options := buildDriverOptions(gpuInfo)

	assert.NotEmpty(t, options)
	assert.Len(t, options, 4)
}

func TestBuildDriverOptions_HasRecommended(t *testing.T) {
	options := buildDriverOptions(nil)

	hasRecommended := false
	for _, opt := range options {
		if opt.Recommended {
			hasRecommended = true
			break
		}
	}
	assert.True(t, hasRecommended, "Should have at least one recommended option")
}

func TestBuildDriverOptions_VersionsMatch(t *testing.T) {
	options := buildDriverOptions(nil)

	// Static fallback versions: 560 (Latest), 550 (Production), 535 (LTS), 470 (Legacy)
	expectedVersions := []string{"560", "550", "535", "470"}
	for i, opt := range options {
		assert.Equal(t, expectedVersions[i], opt.Version)
	}
}

func TestBuildDriverOptions_NilGPUInfo(t *testing.T) {
	options := buildDriverOptions(nil)

	assert.NotEmpty(t, options)
	assert.Len(t, options, 4)
}

// =============================================================================
// BuildComponentOptions Tests
// =============================================================================

func TestBuildComponentOptions(t *testing.T) {
	options := buildComponentOptions()

	assert.NotEmpty(t, options)
	assert.Len(t, options, 4)
}

func TestBuildComponentOptions_HasRequiredComponent(t *testing.T) {
	options := buildComponentOptions()

	hasRequired := false
	for _, opt := range options {
		if opt.Required {
			hasRequired = true
			assert.True(t, opt.Selected, "Required component should be selected")
			assert.Equal(t, "driver", opt.ID)
		}
	}
	assert.True(t, hasRequired, "Should have at least one required component")
}

func TestBuildComponentOptions_ComponentIDs(t *testing.T) {
	options := buildComponentOptions()

	expectedIDs := []string{"driver", "cuda", "cudnn", "settings"}
	for i, opt := range options {
		assert.Equal(t, expectedIDs[i], opt.ID)
	}
}

func TestBuildComponentOptions_InitialSelection(t *testing.T) {
	options := buildComponentOptions()

	// Driver and Settings should be selected by default
	assert.True(t, options[0].Selected)  // driver
	assert.False(t, options[1].Selected) // cuda
	assert.False(t, options[2].Selected) // cudnn
	assert.True(t, options[3].Selected)  // settings
}

// =============================================================================
// NewSelection Tests
// =============================================================================

func TestNewSelection(t *testing.T) {
	styles := getTestStyles()
	version := "1.0.0"
	gpuInfo := createMockGPUInfo()

	m := NewSelection(styles, version, gpuInfo)

	assert.Equal(t, version, m.Version())
	assert.Equal(t, gpuInfo, m.GPUInfo())
	assert.Equal(t, 0, m.Width())
	assert.Equal(t, 0, m.Height())
	assert.False(t, m.Ready())
	assert.Equal(t, 0, m.FocusedSection()) // Starts on drivers
	assert.Equal(t, 0, m.SelectedDriverIndex())
	assert.Equal(t, 0, m.SelectedComponentIndex())
	assert.NotEmpty(t, m.DriverOptions())
	assert.NotEmpty(t, m.ComponentOptions())
}

func TestNewSelection_WithNilGPUInfo(t *testing.T) {
	styles := getTestStyles()

	m := NewSelection(styles, "1.0.0", nil)

	assert.Nil(t, m.GPUInfo())
	assert.NotEmpty(t, m.DriverOptions())
}

func TestNewSelection_KeyMapInitialized(t *testing.T) {
	styles := getTestStyles()

	m := NewSelection(styles, "1.0.0", nil)
	km := m.KeyMap()

	assert.NotEmpty(t, km.Up.Keys())
	assert.NotEmpty(t, km.Down.Keys())
	assert.NotEmpty(t, km.Continue.Keys())
}

// =============================================================================
// Init Tests
// =============================================================================

func TestSelectionModel_Init(t *testing.T) {
	styles := getTestStyles()
	m := NewSelection(styles, "1.0.0", nil)

	cmd := m.Init()

	assert.Nil(t, cmd, "Init should return nil for selection view")
}

// =============================================================================
// Update Tests - WindowSizeMsg
// =============================================================================

func TestSelectionModel_Update_WindowSizeMsg(t *testing.T) {
	styles := getTestStyles()
	m := NewSelection(styles, "1.0.0", nil)

	assert.False(t, m.Ready())

	msg := tea.WindowSizeMsg{Width: 80, Height: 24}
	updated, cmd := m.Update(msg)

	assert.Equal(t, 80, updated.Width())
	assert.Equal(t, 24, updated.Height())
	assert.True(t, updated.Ready())
	assert.Nil(t, cmd)
}

func TestSelectionModel_Update_WindowSizeMsg_Large(t *testing.T) {
	styles := getTestStyles()
	m := NewSelection(styles, "1.0.0", nil)

	msg := tea.WindowSizeMsg{Width: 200, Height: 60}
	updated, _ := m.Update(msg)

	assert.Equal(t, 200, updated.Width())
	assert.Equal(t, 60, updated.Height())
	assert.True(t, updated.Ready())
}

func TestSelectionModel_Update_WindowSizeMsg_Small(t *testing.T) {
	styles := getTestStyles()
	m := NewSelection(styles, "1.0.0", nil)

	msg := tea.WindowSizeMsg{Width: 40, Height: 10}
	updated, _ := m.Update(msg)

	assert.Equal(t, 40, updated.Width())
	assert.Equal(t, 10, updated.Height())
	assert.True(t, updated.Ready())
}

// =============================================================================
// Update Tests - Section Navigation with Tab
// =============================================================================

func TestSelectionModel_Update_Tab_NavigatesSections(t *testing.T) {
	styles := getTestStyles()
	m := NewSelection(styles, "1.0.0", nil)
	m.SetSize(80, 24)

	assert.Equal(t, 0, m.FocusedSection())

	// Tab to components section
	msg := tea.KeyMsg{Type: tea.KeyTab}
	m, _ = m.Update(msg)
	assert.Equal(t, 1, m.FocusedSection())

	// Tab to buttons section
	m, _ = m.Update(msg)
	assert.Equal(t, 2, m.FocusedSection())

	// Tab wraps back to drivers
	m, _ = m.Update(msg)
	assert.Equal(t, 0, m.FocusedSection())
}

func TestSelectionModel_Update_Tab_FullCycle(t *testing.T) {
	styles := getTestStyles()
	m := NewSelection(styles, "1.0.0", nil)
	m.SetSize(80, 24)

	msg := tea.KeyMsg{Type: tea.KeyTab}

	// Full cycle through sections
	for i := 0; i < 6; i++ {
		expected := (i + 1) % 3
		m, _ = m.Update(msg)
		assert.Equal(t, expected, m.FocusedSection())
	}
}

// =============================================================================
// Update Tests - Up/Down Navigation in Drivers Section
// =============================================================================

func TestSelectionModel_Update_DownKey_DriversSection(t *testing.T) {
	styles := getTestStyles()
	m := NewSelection(styles, "1.0.0", nil)
	m.SetSize(80, 24)

	assert.Equal(t, 0, m.SelectedDriverIndex())

	msg := tea.KeyMsg{Type: tea.KeyDown}
	m, _ = m.Update(msg)
	assert.Equal(t, 1, m.SelectedDriverIndex())

	m, _ = m.Update(msg)
	assert.Equal(t, 2, m.SelectedDriverIndex())

	m, _ = m.Update(msg)
	assert.Equal(t, 3, m.SelectedDriverIndex())

	// Can't go past last option
	m, _ = m.Update(msg)
	assert.Equal(t, 3, m.SelectedDriverIndex())
}

func TestSelectionModel_Update_UpKey_DriversSection(t *testing.T) {
	styles := getTestStyles()
	m := NewSelection(styles, "1.0.0", nil)
	m.SetSize(80, 24)

	// Move down first
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	assert.Equal(t, 2, m.SelectedDriverIndex())

	// Move up
	msg := tea.KeyMsg{Type: tea.KeyUp}
	m, _ = m.Update(msg)
	assert.Equal(t, 1, m.SelectedDriverIndex())

	m, _ = m.Update(msg)
	assert.Equal(t, 0, m.SelectedDriverIndex())

	// Can't go past first option
	m, _ = m.Update(msg)
	assert.Equal(t, 0, m.SelectedDriverIndex())
}

func TestSelectionModel_Update_JKKeys_DriversSection(t *testing.T) {
	styles := getTestStyles()
	m := NewSelection(styles, "1.0.0", nil)
	m.SetSize(80, 24)

	// Use j for down
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
	m, _ = m.Update(msg)
	assert.Equal(t, 1, m.SelectedDriverIndex())

	// Use k for up
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
	m, _ = m.Update(msg)
	assert.Equal(t, 0, m.SelectedDriverIndex())
}

// =============================================================================
// Update Tests - Up/Down Navigation in Components Section
// =============================================================================

func TestSelectionModel_Update_DownKey_ComponentsSection(t *testing.T) {
	styles := getTestStyles()
	m := NewSelection(styles, "1.0.0", nil)
	m.SetSize(80, 24)

	// Switch to components section
	m.SetFocusedSection(1)

	assert.Equal(t, 0, m.SelectedComponentIndex())

	msg := tea.KeyMsg{Type: tea.KeyDown}
	m, _ = m.Update(msg)
	assert.Equal(t, 1, m.SelectedComponentIndex())

	m, _ = m.Update(msg)
	assert.Equal(t, 2, m.SelectedComponentIndex())

	m, _ = m.Update(msg)
	assert.Equal(t, 3, m.SelectedComponentIndex())

	// Can't go past last option
	m, _ = m.Update(msg)
	assert.Equal(t, 3, m.SelectedComponentIndex())
}

func TestSelectionModel_Update_UpKey_ComponentsSection(t *testing.T) {
	styles := getTestStyles()
	m := NewSelection(styles, "1.0.0", nil)
	m.SetSize(80, 24)

	// Switch to components section
	m.SetFocusedSection(1)

	// Move down first
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	assert.Equal(t, 2, m.SelectedComponentIndex())

	// Move up
	msg := tea.KeyMsg{Type: tea.KeyUp}
	m, _ = m.Update(msg)
	assert.Equal(t, 1, m.SelectedComponentIndex())
}

// =============================================================================
// Update Tests - Space Toggles Component Selection
// =============================================================================

func TestSelectionModel_Update_Space_TogglesComponent(t *testing.T) {
	styles := getTestStyles()
	m := NewSelection(styles, "1.0.0", nil)
	m.SetSize(80, 24)

	// Switch to components section
	m.SetFocusedSection(1)

	// Move to CUDA (index 1)
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	assert.Equal(t, 1, m.SelectedComponentIndex())

	// CUDA should not be selected initially
	assert.False(t, m.ComponentOptions()[1].Selected)

	// Toggle with space
	msg := tea.KeyMsg{Type: tea.KeySpace}
	m, _ = m.Update(msg)

	// Now it should be selected
	assert.True(t, m.ComponentOptions()[1].Selected)

	// Toggle again
	m, _ = m.Update(msg)
	assert.False(t, m.ComponentOptions()[1].Selected)
}

func TestSelectionModel_Update_Space_CannotToggleRequired(t *testing.T) {
	styles := getTestStyles()
	m := NewSelection(styles, "1.0.0", nil)
	m.SetSize(80, 24)

	// Switch to components section
	m.SetFocusedSection(1)

	// First component (driver) is required and selected
	assert.True(t, m.ComponentOptions()[0].Required)
	assert.True(t, m.ComponentOptions()[0].Selected)

	// Try to toggle with space
	msg := tea.KeyMsg{Type: tea.KeySpace}
	m, _ = m.Update(msg)

	// Should still be selected
	assert.True(t, m.ComponentOptions()[0].Selected)
}

func TestSelectionModel_Update_Space_NotInDriversSection(t *testing.T) {
	styles := getTestStyles()
	m := NewSelection(styles, "1.0.0", nil)
	m.SetSize(80, 24)

	// Stay in drivers section (section 0)
	assert.Equal(t, 0, m.FocusedSection())

	// Get initial component states
	initialSelected := m.ComponentOptions()[1].Selected

	// Press space in drivers section
	msg := tea.KeyMsg{Type: tea.KeySpace}
	m, _ = m.Update(msg)

	// Component selection should not change
	assert.Equal(t, initialSelected, m.ComponentOptions()[1].Selected)
}

// =============================================================================
// Update Tests - Left/Right in Buttons Section
// =============================================================================

func TestSelectionModel_Update_LeftRight_ButtonsSection(t *testing.T) {
	styles := getTestStyles()
	m := NewSelection(styles, "1.0.0", nil)
	m.SetSize(80, 24)

	// Navigate to buttons section
	m.SetFocusedSection(2)

	assert.Equal(t, 0, m.FocusedButtonIndex()) // Continue button

	// Move right to Back button
	msg := tea.KeyMsg{Type: tea.KeyRight}
	m, _ = m.Update(msg)
	assert.Equal(t, 1, m.FocusedButtonIndex())

	// Move left back to Continue
	msg = tea.KeyMsg{Type: tea.KeyLeft}
	m, _ = m.Update(msg)
	assert.Equal(t, 0, m.FocusedButtonIndex())
}

// =============================================================================
// Update Tests - Continue Key
// =============================================================================

func TestSelectionModel_Update_Enter_FromDrivers_MovesToComponents(t *testing.T) {
	styles := getTestStyles()
	m := NewSelection(styles, "1.0.0", nil)
	m.SetSize(80, 24)

	assert.Equal(t, 0, m.FocusedSection())

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	m, cmd := m.Update(msg)

	assert.Equal(t, 1, m.FocusedSection())
	assert.Nil(t, cmd) // No navigation command, just section change
}

func TestSelectionModel_Update_Enter_FromComponents_MovesToButtons(t *testing.T) {
	styles := getTestStyles()
	m := NewSelection(styles, "1.0.0", nil)
	m.SetSize(80, 24)

	m.SetFocusedSection(1)

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	m, cmd := m.Update(msg)

	assert.Equal(t, 2, m.FocusedSection())
	assert.Nil(t, cmd)
}

func TestSelectionModel_Update_Enter_FromButtons_Continue_NavigatesToConfirmation(t *testing.T) {
	styles := getTestStyles()
	gpuInfo := createMockGPUInfo()
	m := NewSelection(styles, "1.0.0", gpuInfo)
	m.SetSize(80, 24)

	m.SetFocusedSection(2)
	assert.Equal(t, 0, m.FocusedButtonIndex()) // Continue button

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	_, cmd := m.Update(msg)

	require.NotNil(t, cmd)

	result := cmd()
	navMsg, ok := result.(NavigateToConfirmationMsg)
	assert.True(t, ok, "Expected NavigateToConfirmationMsg")
	assert.Equal(t, gpuInfo, navMsg.GPUInfo)
	assert.NotEmpty(t, navMsg.SelectedDriver.Version)
	assert.NotEmpty(t, navMsg.SelectedComponents)
}

func TestSelectionModel_Update_Enter_FromButtons_Back_NavigatesToDetection(t *testing.T) {
	styles := getTestStyles()
	m := NewSelection(styles, "1.0.0", nil)
	m.SetSize(80, 24)

	m.SetFocusedSection(2)
	// Move to Back button
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRight})
	assert.Equal(t, 1, m.FocusedButtonIndex()) // Back button

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	_, cmd := m.Update(msg)

	require.NotNil(t, cmd)

	result := cmd()
	_, ok := result.(NavigateToDetectionMsg)
	assert.True(t, ok, "Expected NavigateToDetectionMsg")
}

// =============================================================================
// Update Tests - Back Key
// =============================================================================

func TestSelectionModel_Update_Esc_NavigatesToDetection(t *testing.T) {
	styles := getTestStyles()
	m := NewSelection(styles, "1.0.0", nil)
	m.SetSize(80, 24)

	msg := tea.KeyMsg{Type: tea.KeyEscape}
	_, cmd := m.Update(msg)

	require.NotNil(t, cmd)

	result := cmd()
	_, ok := result.(NavigateToDetectionMsg)
	assert.True(t, ok, "Expected NavigateToDetectionMsg")
}

// =============================================================================
// Update Tests - Quit Key
// =============================================================================

func TestSelectionModel_Update_QuitKey(t *testing.T) {
	styles := getTestStyles()
	m := NewSelection(styles, "1.0.0", nil)
	m.SetSize(80, 24)

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
	_, cmd := m.Update(msg)

	require.NotNil(t, cmd, "q key should return quit command")
}

func TestSelectionModel_Update_CtrlC(t *testing.T) {
	styles := getTestStyles()
	m := NewSelection(styles, "1.0.0", nil)
	m.SetSize(80, 24)

	msg := tea.KeyMsg{Type: tea.KeyCtrlC}
	_, cmd := m.Update(msg)

	require.NotNil(t, cmd, "ctrl+c should return quit command")
}

// =============================================================================
// Update Tests - Help Key
// =============================================================================

func TestSelectionModel_Update_HelpKey(t *testing.T) {
	styles := getTestStyles()
	m := NewSelection(styles, "1.0.0", nil)
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
// View Tests - Not Ready
// =============================================================================

func TestSelectionModel_View_NotReady(t *testing.T) {
	styles := getTestStyles()
	m := NewSelection(styles, "1.0.0", nil)

	view := m.View()

	assert.Equal(t, "Loading...", view)
}

// =============================================================================
// View Tests - Ready
// =============================================================================

func TestSelectionModel_View_Ready(t *testing.T) {
	styles := getTestStyles()
	m := NewSelection(styles, "1.0.0", nil)
	m.SetSize(100, 40)

	view := m.View()

	assert.NotEmpty(t, view)
	assert.NotEqual(t, "Loading...", view)
}

func TestSelectionModel_View_ShowsDriversPanel(t *testing.T) {
	styles := getTestStyles()
	m := NewSelection(styles, "1.0.0", nil)
	m.SetSize(100, 40)

	view := m.View()

	assert.Contains(t, view, "Driver Version")
	assert.Contains(t, view, "550")
	assert.Contains(t, view, "Latest")
	assert.Contains(t, view, "Recommended")
}

func TestSelectionModel_View_ShowsComponentsPanel(t *testing.T) {
	styles := getTestStyles()
	m := NewSelection(styles, "1.0.0", nil)
	m.SetSize(100, 40)

	view := m.View()

	assert.Contains(t, view, "Components")
	assert.Contains(t, view, "NVIDIA Driver")
	assert.Contains(t, view, "CUDA Toolkit")
	assert.Contains(t, view, "cuDNN")
	assert.Contains(t, view, "NVIDIA Settings")
}

func TestSelectionModel_View_ShowsButtons(t *testing.T) {
	styles := getTestStyles()
	m := NewSelection(styles, "1.0.0", nil)
	m.SetSize(100, 40)

	view := m.View()

	assert.Contains(t, view, "Continue")
	assert.Contains(t, view, "Back")
}

func TestSelectionModel_View_ShowsAllDriverVersions(t *testing.T) {
	styles := getTestStyles()
	m := NewSelection(styles, "1.0.0", nil)
	m.SetSize(100, 40)

	view := m.View()

	// Static fallback versions: 560 (Latest), 550 (Production), 535 (LTS), 470 (Legacy)
	assert.Contains(t, view, "560")
	assert.Contains(t, view, "550")
	assert.Contains(t, view, "535")
	assert.Contains(t, view, "470")
}

func TestSelectionModel_View_ShowsBranches(t *testing.T) {
	styles := getTestStyles()
	m := NewSelection(styles, "1.0.0", nil)
	m.SetSize(100, 40)

	view := m.View()

	assert.Contains(t, view, "Latest")
	assert.Contains(t, view, "Production")
	assert.Contains(t, view, "LTS")
	assert.Contains(t, view, "Legacy")
}

// =============================================================================
// View Tests - Various Sizes
// =============================================================================

func TestSelectionModel_View_VariousSizes(t *testing.T) {
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
			m := NewSelection(styles, "1.0.0", nil)
			m.SetSize(tc.width, tc.height)

			view := m.View()

			assert.NotEmpty(t, view)
			assert.NotEqual(t, "Loading...", view)
		})
	}
}

func TestSelectionModel_View_VerySmallSize(t *testing.T) {
	styles := getTestStyles()
	m := NewSelection(styles, "1.0.0", nil)
	m.SetSize(10, 5)

	// Should not panic
	view := m.View()
	assert.NotEmpty(t, view)
}

// =============================================================================
// SelectedDriverOption Tests
// =============================================================================

func TestSelectionModel_SelectedDriverOption(t *testing.T) {
	styles := getTestStyles()
	m := NewSelection(styles, "1.0.0", nil)

	opt := m.SelectedDriverOption()

	// First option is 560 (Latest, Recommended)
	assert.Equal(t, "560", opt.Version)
	assert.Equal(t, "Latest", opt.Branch)
	assert.True(t, opt.Recommended)
}

func TestSelectionModel_SelectedDriverOption_AfterNavigation(t *testing.T) {
	styles := getTestStyles()
	m := NewSelection(styles, "1.0.0", nil)
	m.SetSize(80, 24)

	// Navigate to 535 LTS (index 2)
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})

	opt := m.SelectedDriverOption()

	assert.Equal(t, "535", opt.Version)
	assert.Equal(t, "LTS", opt.Branch)
	assert.False(t, opt.Recommended)
}

// =============================================================================
// SelectedComponents Tests
// =============================================================================

func TestSelectionModel_SelectedComponents_Initial(t *testing.T) {
	styles := getTestStyles()
	m := NewSelection(styles, "1.0.0", nil)

	selected := m.SelectedComponents()

	// Driver and Settings are selected by default
	assert.Len(t, selected, 2)

	ids := make([]string, len(selected))
	for i, c := range selected {
		ids[i] = c.ID
	}
	assert.Contains(t, ids, "driver")
	assert.Contains(t, ids, "settings")
}

func TestSelectionModel_SelectedComponents_AfterToggle(t *testing.T) {
	styles := getTestStyles()
	m := NewSelection(styles, "1.0.0", nil)
	m.SetSize(80, 24)

	// Switch to components and toggle CUDA
	m.SetFocusedSection(1)
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})  // Move to CUDA
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeySpace}) // Toggle

	selected := m.SelectedComponents()

	assert.Len(t, selected, 3)

	ids := make([]string, len(selected))
	for i, c := range selected {
		ids[i] = c.ID
	}
	assert.Contains(t, ids, "driver")
	assert.Contains(t, ids, "cuda")
	assert.Contains(t, ids, "settings")
}

// =============================================================================
// SetSize Tests
// =============================================================================

func TestSelectionModel_SetSize(t *testing.T) {
	styles := getTestStyles()
	m := NewSelection(styles, "1.0.0", nil)

	assert.False(t, m.Ready())

	m.SetSize(100, 50)

	assert.True(t, m.Ready())
	assert.Equal(t, 100, m.Width())
	assert.Equal(t, 50, m.Height())
}

func TestSelectionModel_SetSize_Multiple(t *testing.T) {
	styles := getTestStyles()
	m := NewSelection(styles, "1.0.0", nil)

	m.SetSize(80, 24)
	assert.Equal(t, 80, m.Width())
	assert.Equal(t, 24, m.Height())

	m.SetSize(120, 40)
	assert.Equal(t, 120, m.Width())
	assert.Equal(t, 40, m.Height())
}

// =============================================================================
// SetFocusedSection Tests
// =============================================================================

func TestSelectionModel_SetFocusedSection(t *testing.T) {
	styles := getTestStyles()
	m := NewSelection(styles, "1.0.0", nil)

	assert.Equal(t, 0, m.FocusedSection())

	m.SetFocusedSection(1)
	assert.Equal(t, 1, m.FocusedSection())

	m.SetFocusedSection(2)
	assert.Equal(t, 2, m.FocusedSection())
}

func TestSelectionModel_SetFocusedSection_InvalidValues(t *testing.T) {
	styles := getTestStyles()
	m := NewSelection(styles, "1.0.0", nil)

	m.SetFocusedSection(1)
	assert.Equal(t, 1, m.FocusedSection())

	// Invalid values should be ignored
	m.SetFocusedSection(-1)
	assert.Equal(t, 1, m.FocusedSection())

	m.SetFocusedSection(10)
	assert.Equal(t, 1, m.FocusedSection())
}

// =============================================================================
// Getter Tests
// =============================================================================

func TestSelectionModel_Getters(t *testing.T) {
	styles := getTestStyles()
	gpuInfo := createMockGPUInfo()
	m := NewSelection(styles, "2.0.0", gpuInfo)
	m.SetSize(100, 50)

	assert.Equal(t, 100, m.Width())
	assert.Equal(t, 50, m.Height())
	assert.True(t, m.Ready())
	assert.Equal(t, "2.0.0", m.Version())
	assert.Equal(t, gpuInfo, m.GPUInfo())
	assert.NotNil(t, m.KeyMap())
	assert.Equal(t, 0, m.FocusedSection())
	assert.Equal(t, 0, m.SelectedDriverIndex())
	assert.Equal(t, 0, m.SelectedComponentIndex())
	assert.NotEmpty(t, m.DriverOptions())
	assert.NotEmpty(t, m.ComponentOptions())
}

// =============================================================================
// Message Type Tests
// =============================================================================

func TestNavigateToConfirmationMsg(t *testing.T) {
	gpuInfo := createMockGPUInfo()
	driver := DriverOption{Version: "550", Branch: "Latest"}
	components := []ComponentOption{{ID: "driver", Selected: true}}

	msg := NavigateToConfirmationMsg{
		GPUInfo:            gpuInfo,
		SelectedDriver:     driver,
		SelectedComponents: components,
	}

	assert.Equal(t, gpuInfo, msg.GPUInfo)
	assert.Equal(t, "550", msg.SelectedDriver.Version)
	assert.Len(t, msg.SelectedComponents, 1)
}

func TestNavigateToDetectionMsg_Struct(t *testing.T) {
	msg := NavigateToDetectionMsg{}
	assert.NotNil(t, msg)
}

// =============================================================================
// Integration Tests
// =============================================================================

func TestSelectionModel_FullFlow(t *testing.T) {
	styles := getTestStyles()
	gpuInfo := createMockGPUInfo()
	m := NewSelection(styles, "1.0.0", gpuInfo)

	// Window resize
	m, _ = m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	assert.True(t, m.Ready())

	// Navigate drivers - select 535 LTS
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	assert.Equal(t, 2, m.SelectedDriverIndex())
	assert.Equal(t, "535", m.SelectedDriverOption().Version)

	// Tab to components
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	assert.Equal(t, 1, m.FocusedSection())

	// Select CUDA
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})  // Move to CUDA
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeySpace}) // Toggle CUDA on
	assert.True(t, m.ComponentOptions()[1].Selected)

	// Tab to buttons
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	assert.Equal(t, 2, m.FocusedSection())

	// Press enter to continue
	m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.NotNil(t, cmd)

	result := cmd()
	navMsg, ok := result.(NavigateToConfirmationMsg)
	assert.True(t, ok)
	assert.Equal(t, "535", navMsg.SelectedDriver.Version)
	assert.Len(t, navMsg.SelectedComponents, 3) // driver, cuda, settings
}

func TestSelectionModel_BackFlow(t *testing.T) {
	styles := getTestStyles()
	m := NewSelection(styles, "1.0.0", nil)
	m.SetSize(100, 40)

	// Press escape to go back
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	require.NotNil(t, cmd)

	result := cmd()
	_, ok := result.(NavigateToDetectionMsg)
	assert.True(t, ok)
}

func TestSelectionModel_HelpToggleFlow(t *testing.T) {
	styles := getTestStyles()
	m := NewSelection(styles, "1.0.0", nil)
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

func TestSelectionModel_EmptyStyles(t *testing.T) {
	var styles = getTestStyles()
	m := NewSelection(styles, "1.0.0", nil)
	m.SetSize(80, 24)

	// Should not panic
	view := m.View()
	assert.NotEmpty(t, view)
}

func TestSelectionModel_UnknownMessage(t *testing.T) {
	styles := getTestStyles()
	m := NewSelection(styles, "1.0.0", nil)
	m.SetSize(80, 24)

	type customMsg struct{}

	updated, cmd := m.Update(customMsg{})

	// State should remain unchanged
	assert.True(t, updated.Ready())
	assert.Equal(t, 0, updated.FocusedSection())
	assert.Nil(t, cmd)
}

func TestSelectionModel_MultipleSizeChanges(t *testing.T) {
	styles := getTestStyles()
	m := NewSelection(styles, "1.0.0", nil)

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

// =============================================================================
// Component Option Toggle Edge Cases
// =============================================================================

func TestSelectionModel_ToggleAllOptionalComponents(t *testing.T) {
	styles := getTestStyles()
	m := NewSelection(styles, "1.0.0", nil)
	m.SetSize(80, 24)

	// Switch to components section
	m.SetFocusedSection(1)

	// Toggle all optional components
	for i := 0; i < 4; i++ {
		// Move to component
		for j := 0; j < i; j++ {
			m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
		}

		// Try to toggle
		comp := m.ComponentOptions()[i]
		if !comp.Required {
			m, _ = m.Update(tea.KeyMsg{Type: tea.KeySpace})
		}

		// Reset position
		m, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
		m.SetFocusedSection(1)
	}

	// Verify required component is still selected
	assert.True(t, m.ComponentOptions()[0].Selected)
}

func TestSelectionModel_SelectAllThenDeselectAll(t *testing.T) {
	styles := getTestStyles()
	m := NewSelection(styles, "1.0.0", nil)
	m.SetSize(80, 24)

	m.SetFocusedSection(1)

	// Select CUDA and cuDNN
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown}) // CUDA
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeySpace})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown}) // cuDNN
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeySpace})

	// Verify all optional selected
	assert.True(t, m.ComponentOptions()[1].Selected)
	assert.True(t, m.ComponentOptions()[2].Selected)

	// Deselect them
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeySpace}) // cuDNN
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})    // CUDA
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeySpace})

	// Verify deselected
	assert.False(t, m.ComponentOptions()[1].Selected)
	assert.False(t, m.ComponentOptions()[2].Selected)
}
