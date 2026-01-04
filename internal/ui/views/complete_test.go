package views

import (
	"testing"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tungetti/igor/internal/gpu"
	"github.com/tungetti/igor/internal/gpu/nouveau"
	"github.com/tungetti/igor/internal/gpu/nvidia"
	"github.com/tungetti/igor/internal/gpu/pci"
)

// =============================================================================
// CompleteKeyMap Tests
// =============================================================================

func TestDefaultCompleteKeyMap(t *testing.T) {
	km := DefaultCompleteKeyMap()

	// Verify all key bindings are set
	assert.NotEmpty(t, km.Reboot.Keys())
	assert.NotEmpty(t, km.Exit.Keys())
	assert.NotEmpty(t, km.Left.Keys())
	assert.NotEmpty(t, km.Right.Keys())
	assert.NotEmpty(t, km.Help.Keys())
}

func TestCompleteKeyMap_Reboot(t *testing.T) {
	km := DefaultCompleteKeyMap()

	assert.Contains(t, km.Reboot.Keys(), "enter")
	assert.Contains(t, km.Reboot.Keys(), "r")
}

func TestCompleteKeyMap_Exit(t *testing.T) {
	km := DefaultCompleteKeyMap()

	assert.Contains(t, km.Exit.Keys(), "q")
	assert.Contains(t, km.Exit.Keys(), "esc")
}

func TestCompleteKeyMap_Left(t *testing.T) {
	km := DefaultCompleteKeyMap()

	assert.Contains(t, km.Left.Keys(), "left")
	assert.Contains(t, km.Left.Keys(), "h")
}

func TestCompleteKeyMap_Right(t *testing.T) {
	km := DefaultCompleteKeyMap()

	assert.Contains(t, km.Right.Keys(), "right")
	assert.Contains(t, km.Right.Keys(), "l")
	assert.Contains(t, km.Right.Keys(), "tab")
}

func TestCompleteKeyMap_Help(t *testing.T) {
	km := DefaultCompleteKeyMap()

	assert.Contains(t, km.Help.Keys(), "?")
}

func TestCompleteKeyMapShortHelp(t *testing.T) {
	km := DefaultCompleteKeyMap()

	shortHelp := km.ShortHelp()

	assert.Len(t, shortHelp, 2)
	assert.Equal(t, km.Reboot, shortHelp[0])
	assert.Equal(t, km.Exit, shortHelp[1])
}

func TestCompleteKeyMapFullHelp(t *testing.T) {
	km := DefaultCompleteKeyMap()

	fullHelp := km.FullHelp()

	assert.Len(t, fullHelp, 3)

	// First row: reboot and exit
	assert.Len(t, fullHelp[0], 2)
	assert.Equal(t, km.Reboot, fullHelp[0][0])
	assert.Equal(t, km.Exit, fullHelp[0][1])

	// Second row: left and right
	assert.Len(t, fullHelp[1], 2)
	assert.Equal(t, km.Left, fullHelp[1][0])
	assert.Equal(t, km.Right, fullHelp[1][1])

	// Third row: help
	assert.Len(t, fullHelp[2], 1)
	assert.Equal(t, km.Help, fullHelp[2][0])
}

func TestCompleteKeyMap_ImplementsHelpKeyMap(t *testing.T) {
	km := DefaultCompleteKeyMap()

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

func TestCompleteKeyMap_BindingsHaveHelp(t *testing.T) {
	km := DefaultCompleteKeyMap()

	bindings := []struct {
		name    string
		binding key.Binding
	}{
		{"Reboot", km.Reboot},
		{"Exit", km.Exit},
		{"Left", km.Left},
		{"Right", km.Right},
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
// NewComplete Tests
// =============================================================================

func TestNewComplete(t *testing.T) {
	styles := getTestStyles()
	version := "1.0.0"
	gpuInfo := createMockGPUInfo()
	driver := DriverOption{Version: "550", Branch: "Latest"}
	comps := []ComponentOption{{ID: "driver", Name: "NVIDIA Driver", Selected: true}}

	m := NewComplete(styles, version, gpuInfo, driver, comps)

	assert.Equal(t, version, m.Version())
	assert.Equal(t, gpuInfo, m.GPUInfo())
	assert.Equal(t, driver, m.Driver())
	assert.Equal(t, comps, m.ComponentOptions())
	assert.Equal(t, 0, m.Width())
	assert.Equal(t, 0, m.Height())
	assert.False(t, m.Ready())
	assert.Equal(t, 0, m.FocusedButtonIndex())
}

func TestNewComplete_WithNilGPUInfo(t *testing.T) {
	styles := getTestStyles()
	driver := DriverOption{Version: "550", Branch: "Latest"}
	comps := []ComponentOption{}

	m := NewComplete(styles, "1.0.0", nil, driver, comps)

	assert.Nil(t, m.GPUInfo())
	assert.False(t, m.NeedsReboot())
}

func TestNewComplete_KeyMapInitialized(t *testing.T) {
	styles := getTestStyles()

	m := NewComplete(styles, "1.0.0", nil, DriverOption{}, nil)
	km := m.KeyMap()

	assert.NotEmpty(t, km.Reboot.Keys())
	assert.NotEmpty(t, km.Exit.Keys())
	assert.NotEmpty(t, km.Left.Keys())
	assert.NotEmpty(t, km.Right.Keys())
	assert.NotEmpty(t, km.Help.Keys())
}

func TestNewComplete_ButtonsInitialized(t *testing.T) {
	styles := getTestStyles()

	m := NewComplete(styles, "1.0.0", nil, DriverOption{}, nil)

	// First button (Reboot Now) should be focused by default
	assert.Equal(t, 0, m.FocusedButtonIndex())
}

// =============================================================================
// Init Tests
// =============================================================================

func TestCompleteInit(t *testing.T) {
	styles := getTestStyles()
	m := NewComplete(styles, "1.0.0", nil, DriverOption{}, nil)

	cmd := m.Init()

	assert.Nil(t, cmd, "Init should return nil for completion view")
}

// =============================================================================
// Update Tests - WindowSizeMsg
// =============================================================================

func TestCompleteUpdate_WindowSize(t *testing.T) {
	styles := getTestStyles()
	m := NewComplete(styles, "1.0.0", nil, DriverOption{}, nil)

	assert.False(t, m.Ready())

	msg := tea.WindowSizeMsg{Width: 80, Height: 24}
	updated, cmd := m.Update(msg)

	assert.Equal(t, 80, updated.Width())
	assert.Equal(t, 24, updated.Height())
	assert.True(t, updated.Ready())
	assert.Nil(t, cmd)
}

func TestCompleteUpdate_WindowSize_Large(t *testing.T) {
	styles := getTestStyles()
	m := NewComplete(styles, "1.0.0", nil, DriverOption{}, nil)

	msg := tea.WindowSizeMsg{Width: 200, Height: 60}
	updated, _ := m.Update(msg)

	assert.Equal(t, 200, updated.Width())
	assert.Equal(t, 60, updated.Height())
	assert.True(t, updated.Ready())
}

func TestCompleteUpdate_WindowSize_Small(t *testing.T) {
	styles := getTestStyles()
	m := NewComplete(styles, "1.0.0", nil, DriverOption{}, nil)

	msg := tea.WindowSizeMsg{Width: 40, Height: 10}
	updated, _ := m.Update(msg)

	assert.Equal(t, 40, updated.Width())
	assert.Equal(t, 10, updated.Height())
	assert.True(t, updated.Ready())
}

// =============================================================================
// Update Tests - Key Presses
// =============================================================================

func TestCompleteUpdate_KeyPresses_LeftRight_ButtonNavigation(t *testing.T) {
	styles := getTestStyles()
	m := NewComplete(styles, "1.0.0", nil, DriverOption{}, nil)
	m.SetSize(80, 24)

	assert.Equal(t, 0, m.FocusedButtonIndex()) // Reboot Now button

	// Move right to Exit button
	msg := tea.KeyMsg{Type: tea.KeyRight}
	m, _ = m.Update(msg)
	assert.Equal(t, 1, m.FocusedButtonIndex())

	// Move left back to Reboot Now
	msg = tea.KeyMsg{Type: tea.KeyLeft}
	m, _ = m.Update(msg)
	assert.Equal(t, 0, m.FocusedButtonIndex())
}

func TestCompleteUpdate_KeyPresses_HLKeys_ButtonNavigation(t *testing.T) {
	styles := getTestStyles()
	m := NewComplete(styles, "1.0.0", nil, DriverOption{}, nil)
	m.SetSize(80, 24)

	// Use l for right
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}}
	m, _ = m.Update(msg)
	assert.Equal(t, 1, m.FocusedButtonIndex())

	// Use h for left
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}}
	m, _ = m.Update(msg)
	assert.Equal(t, 0, m.FocusedButtonIndex())
}

func TestCompleteUpdate_KeyPresses_TabKey_ButtonNavigation(t *testing.T) {
	styles := getTestStyles()
	m := NewComplete(styles, "1.0.0", nil, DriverOption{}, nil)
	m.SetSize(80, 24)

	// Tab should move right
	msg := tea.KeyMsg{Type: tea.KeyTab}
	m, _ = m.Update(msg)
	assert.Equal(t, 1, m.FocusedButtonIndex())
}

func TestCompleteUpdate_KeyPresses_ButtonsWrap(t *testing.T) {
	styles := getTestStyles()
	m := NewComplete(styles, "1.0.0", nil, DriverOption{}, nil)
	m.SetSize(80, 24)

	// Move right twice (should wrap)
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRight})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRight})
	assert.Equal(t, 0, m.FocusedButtonIndex())

	// Move left (should wrap)
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyLeft})
	assert.Equal(t, 1, m.FocusedButtonIndex())
}

func TestCompleteUpdate_KeyPresses_Reboot_TriggersRebootRequestedMsg(t *testing.T) {
	styles := getTestStyles()
	gpuInfo := createMockGPUInfo()
	driver := DriverOption{Version: "550", Branch: "Latest"}
	comps := []ComponentOption{{ID: "driver", Name: "NVIDIA Driver", Selected: true}}
	m := NewComplete(styles, "1.0.0", gpuInfo, driver, comps)
	m.SetSize(80, 24)

	assert.Equal(t, 0, m.FocusedButtonIndex()) // Reboot Now button

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	_, cmd := m.Update(msg)

	require.NotNil(t, cmd)

	result := cmd()
	_, ok := result.(RebootRequestedMsg)
	assert.True(t, ok, "Expected RebootRequestedMsg")
}

func TestCompleteUpdate_KeyPresses_Reboot_WithRKey(t *testing.T) {
	styles := getTestStyles()
	m := NewComplete(styles, "1.0.0", nil, DriverOption{}, nil)
	m.SetSize(80, 24)

	// Press 'r' to reboot
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}}
	_, cmd := m.Update(msg)

	require.NotNil(t, cmd)

	result := cmd()
	_, ok := result.(RebootRequestedMsg)
	assert.True(t, ok, "Expected RebootRequestedMsg")
}

func TestCompleteUpdate_KeyPresses_Exit_TriggersExitRequestedMsg(t *testing.T) {
	styles := getTestStyles()
	m := NewComplete(styles, "1.0.0", nil, DriverOption{}, nil)
	m.SetSize(80, 24)

	// Navigate to Exit button
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRight})
	assert.Equal(t, 1, m.FocusedButtonIndex())

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	_, cmd := m.Update(msg)

	require.NotNil(t, cmd)

	result := cmd()
	_, ok := result.(ExitRequestedMsg)
	assert.True(t, ok, "Expected ExitRequestedMsg")
}

func TestCompleteUpdate_KeyPresses_Exit_WithQKey(t *testing.T) {
	styles := getTestStyles()
	m := NewComplete(styles, "1.0.0", nil, DriverOption{}, nil)
	m.SetSize(80, 24)

	// Press 'q' to exit
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
	_, cmd := m.Update(msg)

	require.NotNil(t, cmd)

	result := cmd()
	_, ok := result.(ExitRequestedMsg)
	assert.True(t, ok, "Expected ExitRequestedMsg from q key")
}

func TestCompleteUpdate_KeyPresses_Exit_WithEscKey(t *testing.T) {
	styles := getTestStyles()
	m := NewComplete(styles, "1.0.0", nil, DriverOption{}, nil)
	m.SetSize(80, 24)

	// Press esc to exit
	msg := tea.KeyMsg{Type: tea.KeyEscape}
	_, cmd := m.Update(msg)

	require.NotNil(t, cmd)

	result := cmd()
	_, ok := result.(ExitRequestedMsg)
	assert.True(t, ok, "Expected ExitRequestedMsg from esc key")
}

func TestCompleteUpdate_KeyPresses_Help(t *testing.T) {
	styles := getTestStyles()
	m := NewComplete(styles, "1.0.0", nil, DriverOption{}, nil)
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
// View Tests
// =============================================================================

func TestCompleteView_NotReady(t *testing.T) {
	styles := getTestStyles()
	m := NewComplete(styles, "1.0.0", nil, DriverOption{}, nil)

	view := m.View()

	assert.Equal(t, "Loading...", view)
}

func TestCompleteView(t *testing.T) {
	styles := getTestStyles()
	driver := DriverOption{Version: "550", Branch: "Latest"}
	comps := []ComponentOption{
		{ID: "driver", Name: "NVIDIA Driver", Selected: true},
		{ID: "cuda", Name: "CUDA Toolkit", Selected: true},
	}
	m := NewComplete(styles, "1.0.0", nil, driver, comps)
	m.SetSize(100, 40)

	view := m.View()

	assert.NotEmpty(t, view)
	assert.NotEqual(t, "Loading...", view)
	assert.Contains(t, view, "Installation Complete")
	assert.Contains(t, view, "\u2713") // Checkmark
	assert.Contains(t, view, "550")
	assert.Contains(t, view, "Latest")
	assert.Contains(t, view, "NVIDIA Driver")
	assert.Contains(t, view, "CUDA Toolkit")
	assert.Contains(t, view, "nvidia-smi")
	assert.Contains(t, view, "Reboot Now")
	assert.Contains(t, view, "Exit")
}

func TestCompleteView_ShowsDriverInfo(t *testing.T) {
	styles := getTestStyles()
	driver := DriverOption{Version: "545", Branch: "Production"}
	m := NewComplete(styles, "1.0.0", nil, driver, nil)
	m.SetSize(100, 40)

	view := m.View()

	assert.Contains(t, view, "Driver Installed:")
	assert.Contains(t, view, "545")
	assert.Contains(t, view, "Production")
}

func TestCompleteView_ShowsComponents(t *testing.T) {
	styles := getTestStyles()
	comps := []ComponentOption{
		{ID: "driver", Name: "NVIDIA Driver", Selected: true},
		{ID: "settings", Name: "NVIDIA Settings", Selected: true},
	}
	m := NewComplete(styles, "1.0.0", nil, DriverOption{}, comps)
	m.SetSize(100, 40)

	view := m.View()

	assert.Contains(t, view, "Installed Components:")
	assert.Contains(t, view, "NVIDIA Driver")
	assert.Contains(t, view, "NVIDIA Settings")
}

func TestCompleteView_ShowsNextSteps(t *testing.T) {
	styles := getTestStyles()
	m := NewComplete(styles, "1.0.0", nil, DriverOption{}, nil)
	m.SetSize(100, 40)

	view := m.View()

	assert.Contains(t, view, "Next Steps:")
	assert.Contains(t, view, "nvidia-smi")
}

func TestCompleteView_ShowsRebootRecommendation(t *testing.T) {
	styles := getTestStyles()
	gpuInfo := &gpu.GPUInfo{
		NouveauStatus: &nouveau.Status{Loaded: true},
	}
	m := NewComplete(styles, "1.0.0", gpuInfo, DriverOption{}, nil)
	m.SetSize(100, 40)

	view := m.View()

	assert.Contains(t, view, "Reboot is recommended")
}

func TestCompleteView_HidesRebootRecommendation(t *testing.T) {
	styles := getTestStyles()
	gpuInfo := &gpu.GPUInfo{
		NouveauStatus: &nouveau.Status{Loaded: false},
	}
	m := NewComplete(styles, "1.0.0", gpuInfo, DriverOption{}, nil)
	m.SetSize(100, 40)

	view := m.View()

	assert.NotContains(t, view, "Reboot is recommended")
}

func TestCompleteView_ShowsButtons(t *testing.T) {
	styles := getTestStyles()
	m := NewComplete(styles, "1.0.0", nil, DriverOption{}, nil)
	m.SetSize(100, 40)

	view := m.View()

	assert.Contains(t, view, "Reboot Now")
	assert.Contains(t, view, "Exit")
}

func TestCompleteView_VariousSizes(t *testing.T) {
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
			m := NewComplete(styles, "1.0.0", nil, DriverOption{}, nil)
			m.SetSize(tc.width, tc.height)

			view := m.View()

			assert.NotEmpty(t, view)
			assert.NotEqual(t, "Loading...", view)
		})
	}
}

func TestCompleteView_VerySmallSize(t *testing.T) {
	styles := getTestStyles()
	m := NewComplete(styles, "1.0.0", nil, DriverOption{}, nil)
	m.SetSize(10, 5)

	// Should not panic
	view := m.View()
	assert.NotEmpty(t, view)
}

// =============================================================================
// Getter Tests
// =============================================================================

func TestCompleteGetters(t *testing.T) {
	styles := getTestStyles()
	gpuInfo := createMockGPUInfo()
	driver := DriverOption{Version: "550", Branch: "Latest"}
	comps := []ComponentOption{{ID: "driver", Name: "NVIDIA Driver", Selected: true}}
	m := NewComplete(styles, "2.0.0", gpuInfo, driver, comps)
	m.SetSize(100, 50)

	assert.Equal(t, 100, m.Width())
	assert.Equal(t, 50, m.Height())
	assert.True(t, m.Ready())
	assert.Equal(t, "2.0.0", m.Version())
	assert.Equal(t, gpuInfo, m.GPUInfo())
	assert.Equal(t, driver, m.Driver())
	assert.Equal(t, comps, m.ComponentOptions())
	assert.NotNil(t, m.KeyMap())
	assert.Equal(t, 0, m.FocusedButtonIndex())
	assert.False(t, m.IsFullHelpShown())
}

// =============================================================================
// SetSize Tests
// =============================================================================

func TestCompleteSetSize(t *testing.T) {
	styles := getTestStyles()
	m := NewComplete(styles, "1.0.0", nil, DriverOption{}, nil)

	assert.False(t, m.Ready())

	m.SetSize(100, 50)

	assert.True(t, m.Ready())
	assert.Equal(t, 100, m.Width())
	assert.Equal(t, 50, m.Height())
}

func TestCompleteSetSize_Multiple(t *testing.T) {
	styles := getTestStyles()
	m := NewComplete(styles, "1.0.0", nil, DriverOption{}, nil)

	m.SetSize(80, 24)
	assert.Equal(t, 80, m.Width())
	assert.Equal(t, 24, m.Height())

	m.SetSize(120, 40)
	assert.Equal(t, 120, m.Width())
	assert.Equal(t, 40, m.Height())
}

// =============================================================================
// NeedsReboot Tests
// =============================================================================

func TestCompleteNeedsReboot_WithNouveauLoaded(t *testing.T) {
	styles := getTestStyles()
	gpuInfo := &gpu.GPUInfo{
		NouveauStatus: &nouveau.Status{Loaded: true, InUse: true},
	}
	m := NewComplete(styles, "1.0.0", gpuInfo, DriverOption{}, nil)

	assert.True(t, m.NeedsReboot())
}

func TestCompleteNeedsReboot_WithNouveauNotLoaded(t *testing.T) {
	styles := getTestStyles()
	gpuInfo := &gpu.GPUInfo{
		NouveauStatus: &nouveau.Status{Loaded: false, InUse: false},
	}
	m := NewComplete(styles, "1.0.0", gpuInfo, DriverOption{}, nil)

	assert.False(t, m.NeedsReboot())
}

func TestCompleteNeedsReboot_WithNilNouveauStatus(t *testing.T) {
	styles := getTestStyles()
	gpuInfo := &gpu.GPUInfo{
		NouveauStatus: nil,
	}
	m := NewComplete(styles, "1.0.0", gpuInfo, DriverOption{}, nil)

	assert.False(t, m.NeedsReboot())
}

func TestCompleteNeedsReboot_WithNilGPUInfo(t *testing.T) {
	styles := getTestStyles()
	m := NewComplete(styles, "1.0.0", nil, DriverOption{}, nil)

	assert.False(t, m.NeedsReboot())
}

// =============================================================================
// Message Type Tests
// =============================================================================

func TestRebootRequestedMsg_Struct(t *testing.T) {
	msg := RebootRequestedMsg{}
	assert.NotNil(t, msg)
}

func TestExitRequestedMsg_Struct(t *testing.T) {
	msg := ExitRequestedMsg{}
	assert.NotNil(t, msg)
}

// =============================================================================
// Integration Tests
// =============================================================================

func TestComplete_FullFlow_Reboot(t *testing.T) {
	styles := getTestStyles()
	gpuInfo := &gpu.GPUInfo{
		NouveauStatus: &nouveau.Status{Loaded: true},
		NVIDIAGPUs: []gpu.NVIDIAGPUInfo{
			{
				PCIDevice: pci.PCIDevice{Address: "0000:01:00.0"},
				Model:     &nvidia.GPUModel{Name: "GeForce RTX 3080"},
			},
		},
	}
	driver := DriverOption{Version: "550", Branch: "Latest"}
	comps := []ComponentOption{{ID: "driver", Name: "NVIDIA Driver", Selected: true}}
	m := NewComplete(styles, "1.0.0", gpuInfo, driver, comps)

	// Window resize
	m, _ = m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	assert.True(t, m.Ready())
	assert.True(t, m.NeedsReboot())

	// View should render properly
	view := m.View()
	assert.Contains(t, view, "Installation Complete")
	assert.Contains(t, view, "Reboot is recommended")

	// Press enter to reboot (Reboot Now button is focused by default)
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.NotNil(t, cmd)

	result := cmd()
	_, ok := result.(RebootRequestedMsg)
	assert.True(t, ok)
}

func TestComplete_FullFlow_Exit(t *testing.T) {
	styles := getTestStyles()
	m := NewComplete(styles, "1.0.0", nil, DriverOption{}, nil)
	m.SetSize(100, 40)

	// Navigate to Exit button
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRight})
	assert.Equal(t, 1, m.FocusedButtonIndex())

	// Press enter
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.NotNil(t, cmd)

	result := cmd()
	_, ok := result.(ExitRequestedMsg)
	assert.True(t, ok)
}

func TestComplete_FullFlow_EscToExit(t *testing.T) {
	styles := getTestStyles()
	m := NewComplete(styles, "1.0.0", nil, DriverOption{}, nil)
	m.SetSize(100, 40)

	// Press escape to exit
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	require.NotNil(t, cmd)

	result := cmd()
	_, ok := result.(ExitRequestedMsg)
	assert.True(t, ok)
}

func TestComplete_FullFlow_HelpToggle(t *testing.T) {
	styles := getTestStyles()
	m := NewComplete(styles, "1.0.0", nil, DriverOption{}, nil)
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

func TestComplete_UnknownMessage(t *testing.T) {
	styles := getTestStyles()
	m := NewComplete(styles, "1.0.0", nil, DriverOption{}, nil)
	m.SetSize(80, 24)

	type customMsg struct{}

	updated, cmd := m.Update(customMsg{})

	// State should remain unchanged
	assert.True(t, updated.Ready())
	assert.Equal(t, 0, updated.FocusedButtonIndex())
	assert.Nil(t, cmd)
}

func TestComplete_MultipleSizeChanges(t *testing.T) {
	styles := getTestStyles()
	m := NewComplete(styles, "1.0.0", nil, DriverOption{}, nil)

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

func TestComplete_EmptyComponents(t *testing.T) {
	styles := getTestStyles()
	m := NewComplete(styles, "1.0.0", nil, DriverOption{}, []ComponentOption{})
	m.SetSize(100, 40)

	view := m.View()

	assert.Contains(t, view, "Installed Components:")
	// No components listed but should not panic
}

func TestComplete_NilComponents(t *testing.T) {
	styles := getTestStyles()
	m := NewComplete(styles, "1.0.0", nil, DriverOption{}, nil)
	m.SetSize(100, 40)

	// Should not panic
	view := m.View()
	assert.NotEmpty(t, view)
}

func TestComplete_RapidKeyPresses(t *testing.T) {
	styles := getTestStyles()
	m := NewComplete(styles, "1.0.0", nil, DriverOption{}, nil)
	m.SetSize(80, 24)

	// Rapid navigation
	for i := 0; i < 10; i++ {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRight})
	}

	// Should still be in valid state
	assert.GreaterOrEqual(t, m.FocusedButtonIndex(), 0)
	assert.Less(t, m.FocusedButtonIndex(), 2)
}

func TestComplete_EmptyVersion(t *testing.T) {
	styles := getTestStyles()
	m := NewComplete(styles, "", nil, DriverOption{}, nil)
	m.SetSize(80, 24)

	assert.Equal(t, "", m.Version())

	// Should not panic
	view := m.View()
	assert.NotEmpty(t, view)
}

func TestComplete_EmptyDriver(t *testing.T) {
	styles := getTestStyles()
	m := NewComplete(styles, "1.0.0", nil, DriverOption{}, nil)
	m.SetSize(100, 40)

	view := m.View()

	// Should still show driver section even with empty values
	assert.Contains(t, view, "Driver Installed:")
}

func TestComplete_ComponentsWithCheckmarks(t *testing.T) {
	styles := getTestStyles()
	comps := []ComponentOption{
		{ID: "driver", Name: "NVIDIA Driver", Selected: true},
		{ID: "cuda", Name: "CUDA Toolkit", Selected: true},
		{ID: "settings", Name: "NVIDIA Settings", Selected: true},
	}
	m := NewComplete(styles, "1.0.0", nil, DriverOption{Version: "550", Branch: "Latest"}, comps)
	m.SetSize(100, 40)

	view := m.View()

	// Should have checkmarks for each component
	// The view should contain multiple checkmarks
	assert.Contains(t, view, "\u2713")
}

// =============================================================================
// Additional Coverage Tests
// =============================================================================

func TestComplete_NavigationFlow(t *testing.T) {
	styles := getTestStyles()
	m := NewComplete(styles, "1.0.0", nil, DriverOption{}, nil)
	m.SetSize(80, 24)

	// Start at button 0 (Reboot Now)
	assert.Equal(t, 0, m.FocusedButtonIndex())

	// Navigate right to Exit
	msg := tea.KeyMsg{Type: tea.KeyRight}
	m, _ = m.Update(msg)
	assert.Equal(t, 1, m.FocusedButtonIndex())

	// Navigate right again - should wrap to Reboot Now
	m, _ = m.Update(msg)
	assert.Equal(t, 0, m.FocusedButtonIndex())

	// Navigate left - should go to Exit
	msg = tea.KeyMsg{Type: tea.KeyLeft}
	m, _ = m.Update(msg)
	assert.Equal(t, 1, m.FocusedButtonIndex())
}

func TestComplete_AllRenderMethods(t *testing.T) {
	styles := getTestStyles()
	gpuInfo := &gpu.GPUInfo{
		NouveauStatus: &nouveau.Status{Loaded: true},
	}
	driver := DriverOption{Version: "550", Branch: "Latest"}
	comps := []ComponentOption{
		{ID: "driver", Name: "NVIDIA Driver", Selected: true},
		{ID: "cuda", Name: "CUDA Toolkit", Selected: true},
	}
	m := NewComplete(styles, "1.0.0", gpuInfo, driver, comps)
	m.SetSize(100, 40)

	view := m.View()

	// Verify all sections are rendered
	assert.Contains(t, view, "Installation Complete!") // renderSuccessBanner
	assert.Contains(t, view, "Driver Installed:")      // renderDriverInfo
	assert.Contains(t, view, "Installed Components:")  // renderInstalledComponents
	assert.Contains(t, view, "Next Steps:")            // renderNextSteps
	assert.Contains(t, view, "nvidia-smi")             // renderNextSteps
	assert.Contains(t, view, "Reboot is recommended")  // renderNextSteps with reboot
}
