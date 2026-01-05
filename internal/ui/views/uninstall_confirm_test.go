package views

import (
	"testing"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// UninstallConfirmKeyMap Tests
// =============================================================================

func TestDefaultUninstallConfirmKeyMap(t *testing.T) {
	km := DefaultUninstallConfirmKeyMap()

	// Verify all key bindings are set
	assert.NotEmpty(t, km.Confirm.Keys())
	assert.NotEmpty(t, km.Back.Keys())
	assert.NotEmpty(t, km.Quit.Keys())
	assert.NotEmpty(t, km.Left.Keys())
	assert.NotEmpty(t, km.Right.Keys())
	assert.NotEmpty(t, km.Help.Keys())
}

func TestUninstallConfirmKeyMap_Confirm(t *testing.T) {
	km := DefaultUninstallConfirmKeyMap()

	assert.Contains(t, km.Confirm.Keys(), "enter")
	assert.Contains(t, km.Confirm.Keys(), "y")
}

func TestUninstallConfirmKeyMap_Back(t *testing.T) {
	km := DefaultUninstallConfirmKeyMap()

	assert.Contains(t, km.Back.Keys(), "esc")
	assert.Contains(t, km.Back.Keys(), "n")
	assert.Contains(t, km.Back.Keys(), "backspace")
}

func TestUninstallConfirmKeyMap_Quit(t *testing.T) {
	km := DefaultUninstallConfirmKeyMap()

	assert.Contains(t, km.Quit.Keys(), "q")
	assert.Contains(t, km.Quit.Keys(), "ctrl+c")
}

func TestUninstallConfirmKeyMap_Left(t *testing.T) {
	km := DefaultUninstallConfirmKeyMap()

	assert.Contains(t, km.Left.Keys(), "left")
	assert.Contains(t, km.Left.Keys(), "h")
}

func TestUninstallConfirmKeyMap_Right(t *testing.T) {
	km := DefaultUninstallConfirmKeyMap()

	assert.Contains(t, km.Right.Keys(), "right")
	assert.Contains(t, km.Right.Keys(), "l")
	assert.Contains(t, km.Right.Keys(), "tab")
}

func TestUninstallConfirmKeyMap_Help(t *testing.T) {
	km := DefaultUninstallConfirmKeyMap()

	assert.Contains(t, km.Help.Keys(), "?")
}

func TestUninstallConfirmKeyMap_ShortHelp(t *testing.T) {
	km := DefaultUninstallConfirmKeyMap()

	shortHelp := km.ShortHelp()

	assert.Len(t, shortHelp, 3)
	assert.Equal(t, km.Confirm, shortHelp[0])
	assert.Equal(t, km.Back, shortHelp[1])
	assert.Equal(t, km.Quit, shortHelp[2])
}

func TestUninstallConfirmKeyMap_FullHelp(t *testing.T) {
	km := DefaultUninstallConfirmKeyMap()

	fullHelp := km.FullHelp()

	assert.Len(t, fullHelp, 3)

	// First row: confirm and back
	assert.Len(t, fullHelp[0], 2)
	assert.Equal(t, km.Confirm, fullHelp[0][0])
	assert.Equal(t, km.Back, fullHelp[0][1])

	// Second row: left and right
	assert.Len(t, fullHelp[1], 2)
	assert.Equal(t, km.Left, fullHelp[1][0])
	assert.Equal(t, km.Right, fullHelp[1][1])

	// Third row: quit and help
	assert.Len(t, fullHelp[2], 2)
	assert.Equal(t, km.Quit, fullHelp[2][0])
	assert.Equal(t, km.Help, fullHelp[2][1])
}

func TestUninstallConfirmKeyMap_ImplementsHelpKeyMap(t *testing.T) {
	km := DefaultUninstallConfirmKeyMap()

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

func TestUninstallConfirmKeyMap_BindingsHaveHelp(t *testing.T) {
	km := DefaultUninstallConfirmKeyMap()

	bindings := []struct {
		name    string
		binding key.Binding
	}{
		{"Confirm", km.Confirm},
		{"Back", km.Back},
		{"Quit", km.Quit},
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
// NewUninstallConfirm Tests
// =============================================================================

func TestNewUninstallConfirm(t *testing.T) {
	styles := getTestStyles()
	version := "1.0.0"

	m := NewUninstallConfirm(styles, version)

	assert.Equal(t, version, m.Version())
	assert.Equal(t, 0, m.Width())
	assert.Equal(t, 0, m.Height())
	assert.False(t, m.Ready())
	assert.Equal(t, 0, m.FocusedButtonIndex())
	assert.Empty(t, m.InstalledDriver())
	assert.Empty(t, m.PackagesToRemove())
	assert.Empty(t, m.ConfigsToRemove())
	assert.False(t, m.RestoreNouveau())
	assert.Empty(t, m.Warnings())
}

func TestNewUninstallConfirm_WithOptions(t *testing.T) {
	styles := getTestStyles()

	packages := []string{"nvidia-driver-550", "nvidia-utils-550"}
	configs := []string{"/etc/modprobe.d/blacklist-nouveau.conf"}
	warnings := []string{"A reboot will be required"}

	m := NewUninstallConfirm(styles, "1.0.0",
		WithInstalledDriver("550.78"),
		WithPackagesToRemove(packages),
		WithConfigsToRemove(configs),
		WithRestoreNouveau(true),
		WithUninstallWarnings(warnings),
	)

	assert.Equal(t, "550.78", m.InstalledDriver())
	assert.Equal(t, packages, m.PackagesToRemove())
	assert.Equal(t, configs, m.ConfigsToRemove())
	assert.True(t, m.RestoreNouveau())
	assert.Equal(t, warnings, m.Warnings())
}

func TestNewUninstallConfirm_KeyMapInitialized(t *testing.T) {
	styles := getTestStyles()

	m := NewUninstallConfirm(styles, "1.0.0")
	km := m.KeyMap()

	assert.NotEmpty(t, km.Confirm.Keys())
	assert.NotEmpty(t, km.Back.Keys())
	assert.NotEmpty(t, km.Quit.Keys())
}

// =============================================================================
// Init Tests
// =============================================================================

func TestUninstallConfirmModel_Init(t *testing.T) {
	styles := getTestStyles()
	m := NewUninstallConfirm(styles, "1.0.0")

	cmd := m.Init()

	assert.Nil(t, cmd, "Init should return nil for uninstall confirmation view")
}

// =============================================================================
// Update Tests - WindowSizeMsg
// =============================================================================

func TestUninstallConfirmModel_Update_WindowSizeMsg(t *testing.T) {
	styles := getTestStyles()
	m := NewUninstallConfirm(styles, "1.0.0")

	assert.False(t, m.Ready())

	msg := tea.WindowSizeMsg{Width: 80, Height: 24}
	updated, cmd := m.Update(msg)

	assert.Equal(t, 80, updated.Width())
	assert.Equal(t, 24, updated.Height())
	assert.True(t, updated.Ready())
	assert.Nil(t, cmd)
}

func TestUninstallConfirmModel_Update_WindowSizeMsg_Large(t *testing.T) {
	styles := getTestStyles()
	m := NewUninstallConfirm(styles, "1.0.0")

	msg := tea.WindowSizeMsg{Width: 200, Height: 60}
	updated, _ := m.Update(msg)

	assert.Equal(t, 200, updated.Width())
	assert.Equal(t, 60, updated.Height())
	assert.True(t, updated.Ready())
}

func TestUninstallConfirmModel_Update_WindowSizeMsg_Small(t *testing.T) {
	styles := getTestStyles()
	m := NewUninstallConfirm(styles, "1.0.0")

	msg := tea.WindowSizeMsg{Width: 40, Height: 10}
	updated, _ := m.Update(msg)

	assert.Equal(t, 40, updated.Width())
	assert.Equal(t, 10, updated.Height())
	assert.True(t, updated.Ready())
}

// =============================================================================
// Update Tests - Button Navigation (Left/Right)
// =============================================================================

func TestUninstallConfirmModel_Update_LeftRight_ButtonNavigation(t *testing.T) {
	styles := getTestStyles()
	m := NewUninstallConfirm(styles, "1.0.0")
	m.SetSize(80, 24)

	assert.Equal(t, 0, m.FocusedButtonIndex()) // Confirm Uninstall button

	// Move right to Cancel button
	msg := tea.KeyMsg{Type: tea.KeyRight}
	m, _ = m.Update(msg)
	assert.Equal(t, 1, m.FocusedButtonIndex())

	// Move left back to Confirm
	msg = tea.KeyMsg{Type: tea.KeyLeft}
	m, _ = m.Update(msg)
	assert.Equal(t, 0, m.FocusedButtonIndex())
}

func TestUninstallConfirmModel_Update_HLKeys_ButtonNavigation(t *testing.T) {
	styles := getTestStyles()
	m := NewUninstallConfirm(styles, "1.0.0")
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

func TestUninstallConfirmModel_Update_TabKey_ButtonNavigation(t *testing.T) {
	styles := getTestStyles()
	m := NewUninstallConfirm(styles, "1.0.0")
	m.SetSize(80, 24)

	// Tab should move right
	msg := tea.KeyMsg{Type: tea.KeyTab}
	m, _ = m.Update(msg)
	assert.Equal(t, 1, m.FocusedButtonIndex())
}

func TestUninstallConfirmModel_Update_ButtonsWrap(t *testing.T) {
	styles := getTestStyles()
	m := NewUninstallConfirm(styles, "1.0.0")
	m.SetSize(80, 24)

	// Move right twice (should wrap)
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRight})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRight})
	assert.Equal(t, 0, m.FocusedButtonIndex())

	// Move left (should wrap)
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyLeft})
	assert.Equal(t, 1, m.FocusedButtonIndex())
}

// =============================================================================
// Update Tests - Confirm on "Confirm Uninstall" Triggers UninstallConfirmedMsg
// =============================================================================

func TestUninstallConfirmModel_Update_Confirm_Uninstall_TriggersUninstallConfirmedMsg(t *testing.T) {
	styles := getTestStyles()
	m := NewUninstallConfirm(styles, "1.0.0",
		WithInstalledDriver("550.78"),
	)
	m.SetSize(80, 24)

	assert.Equal(t, 0, m.FocusedButtonIndex()) // Confirm Uninstall button

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	_, cmd := m.Update(msg)

	require.NotNil(t, cmd)

	result := cmd()
	_, ok := result.(UninstallConfirmedMsg)
	assert.True(t, ok, "Expected UninstallConfirmedMsg")
}

func TestUninstallConfirmModel_Update_Confirm_Uninstall_WithYKey(t *testing.T) {
	styles := getTestStyles()
	m := NewUninstallConfirm(styles, "1.0.0")
	m.SetSize(80, 24)

	// Press 'y' to confirm
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}}
	_, cmd := m.Update(msg)

	require.NotNil(t, cmd)

	result := cmd()
	_, ok := result.(UninstallConfirmedMsg)
	assert.True(t, ok, "Expected UninstallConfirmedMsg")
}

// =============================================================================
// Update Tests - Confirm on "Cancel" Triggers UninstallCancelledMsg
// =============================================================================

func TestUninstallConfirmModel_Update_Confirm_Cancel_TriggersUninstallCancelledMsg(t *testing.T) {
	styles := getTestStyles()
	m := NewUninstallConfirm(styles, "1.0.0")
	m.SetSize(80, 24)

	// Move to Cancel button
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRight})
	assert.Equal(t, 1, m.FocusedButtonIndex())

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	_, cmd := m.Update(msg)

	require.NotNil(t, cmd)

	result := cmd()
	_, ok := result.(UninstallCancelledMsg)
	assert.True(t, ok, "Expected UninstallCancelledMsg")
}

// =============================================================================
// Update Tests - Back Key Triggers UninstallCancelledMsg
// =============================================================================

func TestUninstallConfirmModel_Update_BackKey_Esc(t *testing.T) {
	styles := getTestStyles()
	m := NewUninstallConfirm(styles, "1.0.0")
	m.SetSize(80, 24)

	msg := tea.KeyMsg{Type: tea.KeyEscape}
	_, cmd := m.Update(msg)

	require.NotNil(t, cmd)

	result := cmd()
	_, ok := result.(UninstallCancelledMsg)
	assert.True(t, ok, "Expected UninstallCancelledMsg")
}

func TestUninstallConfirmModel_Update_BackKey_N(t *testing.T) {
	styles := getTestStyles()
	m := NewUninstallConfirm(styles, "1.0.0")
	m.SetSize(80, 24)

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}}
	_, cmd := m.Update(msg)

	require.NotNil(t, cmd)

	result := cmd()
	_, ok := result.(UninstallCancelledMsg)
	assert.True(t, ok, "Expected UninstallCancelledMsg")
}

func TestUninstallConfirmModel_Update_BackKey_Backspace(t *testing.T) {
	styles := getTestStyles()
	m := NewUninstallConfirm(styles, "1.0.0")
	m.SetSize(80, 24)

	msg := tea.KeyMsg{Type: tea.KeyBackspace}
	_, cmd := m.Update(msg)

	require.NotNil(t, cmd)

	result := cmd()
	_, ok := result.(UninstallCancelledMsg)
	assert.True(t, ok, "Expected UninstallCancelledMsg")
}

// =============================================================================
// Update Tests - Quit Key
// =============================================================================

func TestUninstallConfirmModel_Update_QuitKey(t *testing.T) {
	styles := getTestStyles()
	m := NewUninstallConfirm(styles, "1.0.0")
	m.SetSize(80, 24)

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
	_, cmd := m.Update(msg)

	require.NotNil(t, cmd, "q key should return quit command")
}

func TestUninstallConfirmModel_Update_CtrlC(t *testing.T) {
	styles := getTestStyles()
	m := NewUninstallConfirm(styles, "1.0.0")
	m.SetSize(80, 24)

	msg := tea.KeyMsg{Type: tea.KeyCtrlC}
	_, cmd := m.Update(msg)

	require.NotNil(t, cmd, "ctrl+c should return quit command")
}

// =============================================================================
// Update Tests - Help Key
// =============================================================================

func TestUninstallConfirmModel_Update_HelpKey(t *testing.T) {
	styles := getTestStyles()
	m := NewUninstallConfirm(styles, "1.0.0")
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

func TestUninstallConfirmModel_View_NotReady(t *testing.T) {
	styles := getTestStyles()
	m := NewUninstallConfirm(styles, "1.0.0")

	view := m.View()

	assert.Equal(t, "Loading...", view)
}

// =============================================================================
// View Tests - Ready
// =============================================================================

func TestUninstallConfirmModel_View_Ready(t *testing.T) {
	styles := getTestStyles()
	m := NewUninstallConfirm(styles, "1.0.0")
	m.SetSize(100, 40)

	view := m.View()

	assert.NotEmpty(t, view)
	assert.NotEqual(t, "Loading...", view)
}

func TestUninstallConfirmModel_View_ShowsUninstallTitle(t *testing.T) {
	styles := getTestStyles()
	m := NewUninstallConfirm(styles, "1.0.0")
	m.SetSize(100, 40)

	view := m.View()

	assert.Contains(t, view, "Uninstall NVIDIA Drivers")
}

func TestUninstallConfirmModel_View_ShowsDriverSection(t *testing.T) {
	styles := getTestStyles()
	m := NewUninstallConfirm(styles, "1.0.0",
		WithInstalledDriver("550.78"),
	)
	m.SetSize(100, 40)

	view := m.View()

	assert.Contains(t, view, "Driver Version:")
	assert.Contains(t, view, "550.78")
}

func TestUninstallConfirmModel_View_ShowsPackagesSection(t *testing.T) {
	styles := getTestStyles()
	packages := []string{"nvidia-driver-550", "nvidia-utils-550", "libnvidia-gl-550"}
	m := NewUninstallConfirm(styles, "1.0.0",
		WithPackagesToRemove(packages),
	)
	m.SetSize(100, 40)

	view := m.View()

	assert.Contains(t, view, "Packages:")
	assert.Contains(t, view, "nvidia-driver-550")
	assert.Contains(t, view, "nvidia-utils-550")
	assert.Contains(t, view, "libnvidia-gl-550")
}

func TestUninstallConfirmModel_View_ShowsConfigsSection(t *testing.T) {
	styles := getTestStyles()
	configs := []string{
		"/etc/X11/xorg.conf.d/20-nvidia.conf",
		"/etc/modprobe.d/blacklist-nouveau.conf",
	}
	m := NewUninstallConfirm(styles, "1.0.0",
		WithConfigsToRemove(configs),
	)
	m.SetSize(100, 40)

	view := m.View()

	assert.Contains(t, view, "Configuration:")
	assert.Contains(t, view, "/etc/X11/xorg.conf.d/20-nvidia.conf")
	assert.Contains(t, view, "/etc/modprobe.d/blacklist-nouveau.conf")
}

func TestUninstallConfirmModel_View_ShowsNouveauRestore(t *testing.T) {
	styles := getTestStyles()
	m := NewUninstallConfirm(styles, "1.0.0",
		WithRestoreNouveau(true),
	)
	m.SetSize(100, 40)

	view := m.View()

	assert.Contains(t, view, "Restore nouveau driver")
}

func TestUninstallConfirmModel_View_HidesNouveauRestoreWhenFalse(t *testing.T) {
	styles := getTestStyles()
	m := NewUninstallConfirm(styles, "1.0.0",
		WithRestoreNouveau(false),
	)
	m.SetSize(100, 40)

	view := m.View()

	assert.NotContains(t, view, "Restore nouveau driver")
}

func TestUninstallConfirmModel_View_ShowsButtons(t *testing.T) {
	styles := getTestStyles()
	m := NewUninstallConfirm(styles, "1.0.0")
	m.SetSize(100, 40)

	view := m.View()

	assert.Contains(t, view, "Confirm Uninstall")
	assert.Contains(t, view, "Cancel")
}

// =============================================================================
// View Tests - Warnings
// =============================================================================

func TestUninstallConfirmModel_View_ShowsWarningsWhenPresent(t *testing.T) {
	styles := getTestStyles()
	warnings := []string{"A reboot will be required after uninstall"}
	m := NewUninstallConfirm(styles, "1.0.0",
		WithUninstallWarnings(warnings),
	)
	m.SetSize(100, 40)

	view := m.View()

	assert.Contains(t, view, "A reboot will be required after uninstall")
	assert.Contains(t, view, "\u26A0") // Warning symbol
}

func TestUninstallConfirmModel_View_HidesWarningsWhenEmpty(t *testing.T) {
	styles := getTestStyles()
	m := NewUninstallConfirm(styles, "1.0.0")
	m.SetSize(100, 40)

	view := m.View()

	// Should not show extra warning markers beyond title
	// The view will have the warning emoji in the title, but not extra warning lines
	assert.NotContains(t, view, "reboot will be required")
}

func TestUninstallConfirmModel_View_ShowsMultipleWarnings(t *testing.T) {
	styles := getTestStyles()
	warnings := []string{
		"A reboot will be required",
		"Display may reset during uninstall",
	}
	m := NewUninstallConfirm(styles, "1.0.0",
		WithUninstallWarnings(warnings),
	)
	m.SetSize(100, 40)

	view := m.View()

	assert.Contains(t, view, "A reboot will be required")
	assert.Contains(t, view, "Display may reset during uninstall")
}

// =============================================================================
// View Tests - Various Sizes
// =============================================================================

func TestUninstallConfirmModel_View_VariousSizes(t *testing.T) {
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
			m := NewUninstallConfirm(styles, "1.0.0")
			m.SetSize(tc.width, tc.height)

			view := m.View()

			assert.NotEmpty(t, view)
			assert.NotEqual(t, "Loading...", view)
		})
	}
}

func TestUninstallConfirmModel_View_VerySmallSize(t *testing.T) {
	styles := getTestStyles()
	m := NewUninstallConfirm(styles, "1.0.0")
	m.SetSize(10, 5)

	// Should not panic
	view := m.View()
	assert.NotEmpty(t, view)
}

// =============================================================================
// Getter Tests
// =============================================================================

func TestUninstallConfirmModel_Getters(t *testing.T) {
	styles := getTestStyles()
	packages := []string{"nvidia-driver-550"}
	configs := []string{"/etc/modprobe.d/blacklist-nouveau.conf"}
	warnings := []string{"Reboot required"}

	m := NewUninstallConfirm(styles, "2.0.0",
		WithInstalledDriver("550.78"),
		WithPackagesToRemove(packages),
		WithConfigsToRemove(configs),
		WithRestoreNouveau(true),
		WithUninstallWarnings(warnings),
	)
	m.SetSize(100, 50)

	assert.Equal(t, 100, m.Width())
	assert.Equal(t, 50, m.Height())
	assert.True(t, m.Ready())
	assert.Equal(t, "2.0.0", m.Version())
	assert.Equal(t, "550.78", m.InstalledDriver())
	assert.Equal(t, packages, m.PackagesToRemove())
	assert.Equal(t, configs, m.ConfigsToRemove())
	assert.True(t, m.RestoreNouveau())
	assert.Equal(t, warnings, m.Warnings())
	assert.NotNil(t, m.KeyMap())
	assert.Equal(t, 0, m.FocusedButtonIndex())
}

// =============================================================================
// SetSize Tests
// =============================================================================

func TestUninstallConfirmModel_SetSize(t *testing.T) {
	styles := getTestStyles()
	m := NewUninstallConfirm(styles, "1.0.0")

	assert.False(t, m.Ready())

	m.SetSize(100, 50)

	assert.True(t, m.Ready())
	assert.Equal(t, 100, m.Width())
	assert.Equal(t, 50, m.Height())
}

func TestUninstallConfirmModel_SetSize_Multiple(t *testing.T) {
	styles := getTestStyles()
	m := NewUninstallConfirm(styles, "1.0.0")

	m.SetSize(80, 24)
	assert.Equal(t, 80, m.Width())
	assert.Equal(t, 24, m.Height())

	m.SetSize(120, 40)
	assert.Equal(t, 120, m.Width())
	assert.Equal(t, 40, m.Height())
}

// =============================================================================
// Setter Tests
// =============================================================================

func TestUninstallConfirmModel_SetInstalledDriver(t *testing.T) {
	styles := getTestStyles()
	m := NewUninstallConfirm(styles, "1.0.0")

	m.SetInstalledDriver("550.78")

	assert.Equal(t, "550.78", m.InstalledDriver())
}

func TestUninstallConfirmModel_SetPackagesToRemove(t *testing.T) {
	styles := getTestStyles()
	m := NewUninstallConfirm(styles, "1.0.0")

	packages := []string{"nvidia-driver-550", "nvidia-utils-550"}
	m.SetPackagesToRemove(packages)

	assert.Equal(t, packages, m.PackagesToRemove())
}

func TestUninstallConfirmModel_SetConfigsToRemove(t *testing.T) {
	styles := getTestStyles()
	m := NewUninstallConfirm(styles, "1.0.0")

	configs := []string{"/etc/modprobe.d/blacklist-nouveau.conf"}
	m.SetConfigsToRemove(configs)

	assert.Equal(t, configs, m.ConfigsToRemove())
}

func TestUninstallConfirmModel_SetRestoreNouveau(t *testing.T) {
	styles := getTestStyles()
	m := NewUninstallConfirm(styles, "1.0.0")

	m.SetRestoreNouveau(true)

	assert.True(t, m.RestoreNouveau())

	m.SetRestoreNouveau(false)

	assert.False(t, m.RestoreNouveau())
}

func TestUninstallConfirmModel_SetWarnings(t *testing.T) {
	styles := getTestStyles()
	m := NewUninstallConfirm(styles, "1.0.0")

	warnings := []string{"Reboot required"}
	m.SetWarnings(warnings)

	assert.Equal(t, warnings, m.Warnings())
}

// =============================================================================
// Message Type Tests
// =============================================================================

func TestUninstallConfirmedMsg_Struct(t *testing.T) {
	msg := UninstallConfirmedMsg{}
	assert.NotNil(t, msg)
}

func TestUninstallCancelledMsg_Struct(t *testing.T) {
	msg := UninstallCancelledMsg{}
	assert.NotNil(t, msg)
}

// =============================================================================
// Integration Tests
// =============================================================================

func TestUninstallConfirmModel_FullFlow_ConfirmUninstall(t *testing.T) {
	styles := getTestStyles()
	packages := []string{"nvidia-driver-550"}
	m := NewUninstallConfirm(styles, "1.0.0",
		WithInstalledDriver("550.78"),
		WithPackagesToRemove(packages),
		WithRestoreNouveau(true),
	)

	// Window resize
	m, _ = m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	assert.True(t, m.Ready())

	// View should render properly
	view := m.View()
	assert.Contains(t, view, "Uninstall NVIDIA Drivers")
	assert.Contains(t, view, "550.78")
	assert.Contains(t, view, "nvidia-driver-550")

	// Press enter to confirm (Confirm Uninstall button is focused by default)
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.NotNil(t, cmd)

	result := cmd()
	_, ok := result.(UninstallConfirmedMsg)
	assert.True(t, ok)
}

func TestUninstallConfirmModel_FullFlow_Cancel(t *testing.T) {
	styles := getTestStyles()
	m := NewUninstallConfirm(styles, "1.0.0")
	m.SetSize(100, 40)

	// Navigate to Cancel button
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRight})
	assert.Equal(t, 1, m.FocusedButtonIndex())

	// Press enter
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.NotNil(t, cmd)

	result := cmd()
	_, ok := result.(UninstallCancelledMsg)
	assert.True(t, ok)
}

func TestUninstallConfirmModel_FullFlow_EscToBack(t *testing.T) {
	styles := getTestStyles()
	m := NewUninstallConfirm(styles, "1.0.0")
	m.SetSize(100, 40)

	// Press escape to go back
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	require.NotNil(t, cmd)

	result := cmd()
	_, ok := result.(UninstallCancelledMsg)
	assert.True(t, ok)
}

func TestUninstallConfirmModel_FullFlow_HelpToggle(t *testing.T) {
	styles := getTestStyles()
	m := NewUninstallConfirm(styles, "1.0.0")
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

func TestUninstallConfirmModel_UnknownMessage(t *testing.T) {
	styles := getTestStyles()
	m := NewUninstallConfirm(styles, "1.0.0")
	m.SetSize(80, 24)

	type customMsg struct{}

	updated, cmd := m.Update(customMsg{})

	// State should remain unchanged
	assert.True(t, updated.Ready())
	assert.Equal(t, 0, updated.FocusedButtonIndex())
	assert.Nil(t, cmd)
}

func TestUninstallConfirmModel_MultipleSizeChanges(t *testing.T) {
	styles := getTestStyles()
	m := NewUninstallConfirm(styles, "1.0.0")

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

func TestUninstallConfirmModel_EmptyPackages(t *testing.T) {
	styles := getTestStyles()
	m := NewUninstallConfirm(styles, "1.0.0",
		WithPackagesToRemove([]string{}),
	)
	m.SetSize(100, 40)

	view := m.View()

	// Should not show Packages: section
	assert.NotContains(t, view, "Packages:")
}

func TestUninstallConfirmModel_EmptyConfigs(t *testing.T) {
	styles := getTestStyles()
	m := NewUninstallConfirm(styles, "1.0.0",
		WithConfigsToRemove([]string{}),
	)
	m.SetSize(100, 40)

	view := m.View()

	// Should not show Configuration: section
	assert.NotContains(t, view, "Configuration:")
}

func TestUninstallConfirmModel_NoDriver(t *testing.T) {
	styles := getTestStyles()
	m := NewUninstallConfirm(styles, "1.0.0")
	m.SetSize(100, 40)

	view := m.View()

	// Should not show Driver Version: section
	assert.NotContains(t, view, "Driver Version:")
}

func TestUninstallConfirmModel_FullContent(t *testing.T) {
	styles := getTestStyles()
	packages := []string{"nvidia-driver-550", "nvidia-utils-550", "libnvidia-gl-550"}
	configs := []string{
		"/etc/X11/xorg.conf.d/20-nvidia.conf",
		"/etc/modprobe.d/blacklist-nouveau.conf",
	}
	warnings := []string{"A reboot will be required after uninstall"}

	m := NewUninstallConfirm(styles, "1.0.0",
		WithInstalledDriver("550.78"),
		WithPackagesToRemove(packages),
		WithConfigsToRemove(configs),
		WithRestoreNouveau(true),
		WithUninstallWarnings(warnings),
	)
	m.SetSize(100, 50)

	view := m.View()

	// All sections should be present
	assert.Contains(t, view, "550.78")
	assert.Contains(t, view, "nvidia-driver-550")
	assert.Contains(t, view, "/etc/X11/xorg.conf.d/20-nvidia.conf")
	assert.Contains(t, view, "Restore nouveau driver")
	assert.Contains(t, view, "A reboot will be required after uninstall")
	assert.Contains(t, view, "Confirm Uninstall")
	assert.Contains(t, view, "Cancel")
}
