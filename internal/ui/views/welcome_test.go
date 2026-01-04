package views

import (
	"testing"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tungetti/igor/internal/ui/theme"
)

// Helper to get default styles for testing
func getTestStyles() theme.Styles {
	return theme.DefaultTheme().Styles
}

// =============================================================================
// WelcomeKeyMap Tests
// =============================================================================

func TestDefaultWelcomeKeyMap(t *testing.T) {
	km := DefaultWelcomeKeyMap()

	// Verify all key bindings are set
	assert.NotEmpty(t, km.Start.Keys())
	assert.NotEmpty(t, km.Quit.Keys())
	assert.NotEmpty(t, km.Left.Keys())
	assert.NotEmpty(t, km.Right.Keys())
	assert.NotEmpty(t, km.Help.Keys())
}

func TestWelcomeKeyMap_Start(t *testing.T) {
	km := DefaultWelcomeKeyMap()

	// Start should have enter and space
	assert.Contains(t, km.Start.Keys(), "enter")
	assert.Contains(t, km.Start.Keys(), " ")
}

func TestWelcomeKeyMap_Quit(t *testing.T) {
	km := DefaultWelcomeKeyMap()

	// Quit should have q, esc, and ctrl+c
	assert.Contains(t, km.Quit.Keys(), "q")
	assert.Contains(t, km.Quit.Keys(), "esc")
	assert.Contains(t, km.Quit.Keys(), "ctrl+c")
}

func TestWelcomeKeyMap_Navigation(t *testing.T) {
	km := DefaultWelcomeKeyMap()

	// Left navigation
	assert.Contains(t, km.Left.Keys(), "left")
	assert.Contains(t, km.Left.Keys(), "h")

	// Right navigation
	assert.Contains(t, km.Right.Keys(), "right")
	assert.Contains(t, km.Right.Keys(), "l")
	assert.Contains(t, km.Right.Keys(), "tab")
}

func TestWelcomeKeyMap_Help(t *testing.T) {
	km := DefaultWelcomeKeyMap()

	assert.Contains(t, km.Help.Keys(), "?")
}

func TestWelcomeKeyMap_ShortHelp(t *testing.T) {
	km := DefaultWelcomeKeyMap()

	shortHelp := km.ShortHelp()

	assert.Len(t, shortHelp, 3)
	assert.Equal(t, km.Start, shortHelp[0])
	assert.Equal(t, km.Quit, shortHelp[1])
	assert.Equal(t, km.Help, shortHelp[2])
}

func TestWelcomeKeyMap_FullHelp(t *testing.T) {
	km := DefaultWelcomeKeyMap()

	fullHelp := km.FullHelp()

	assert.Len(t, fullHelp, 2)

	// First row: Start and Quit
	assert.Len(t, fullHelp[0], 2)
	assert.Equal(t, km.Start, fullHelp[0][0])
	assert.Equal(t, km.Quit, fullHelp[0][1])

	// Second row: Left, Right, Help
	assert.Len(t, fullHelp[1], 3)
	assert.Equal(t, km.Left, fullHelp[1][0])
	assert.Equal(t, km.Right, fullHelp[1][1])
	assert.Equal(t, km.Help, fullHelp[1][2])
}

// =============================================================================
// NewWelcome Tests
// =============================================================================

func TestNewWelcome(t *testing.T) {
	styles := getTestStyles()
	version := "1.0.0"

	m := NewWelcome(styles, version)

	assert.Equal(t, version, m.Version())
	assert.Equal(t, 0, m.Width())
	assert.Equal(t, 0, m.Height())
	assert.False(t, m.Ready())
	assert.Equal(t, 0, m.FocusedButtonIndex())
}

func TestNewWelcome_WithEmptyVersion(t *testing.T) {
	styles := getTestStyles()

	m := NewWelcome(styles, "")

	assert.Equal(t, "", m.Version())
}

func TestNewWelcome_KeyMapInitialized(t *testing.T) {
	styles := getTestStyles()

	m := NewWelcome(styles, "1.0.0")
	km := m.KeyMap()

	assert.NotEmpty(t, km.Start.Keys())
	assert.NotEmpty(t, km.Quit.Keys())
	assert.NotEmpty(t, km.Left.Keys())
	assert.NotEmpty(t, km.Right.Keys())
	assert.NotEmpty(t, km.Help.Keys())
}

func TestNewWelcome_ButtonsInitialized(t *testing.T) {
	styles := getTestStyles()

	m := NewWelcome(styles, "1.0.0")

	// First button should be focused
	assert.Equal(t, 0, m.FocusedButtonIndex())
}

// =============================================================================
// Init Tests
// =============================================================================

func TestWelcomeModel_Init(t *testing.T) {
	styles := getTestStyles()
	m := NewWelcome(styles, "1.0.0")

	cmd := m.Init()

	assert.Nil(t, cmd, "Init should return nil")
}

// =============================================================================
// Update Tests - WindowSizeMsg
// =============================================================================

func TestWelcomeModel_Update_WindowSizeMsg(t *testing.T) {
	styles := getTestStyles()
	m := NewWelcome(styles, "1.0.0")

	assert.False(t, m.Ready())

	msg := tea.WindowSizeMsg{Width: 80, Height: 24}
	updated, cmd := m.Update(msg)

	assert.Equal(t, 80, updated.Width())
	assert.Equal(t, 24, updated.Height())
	assert.True(t, updated.Ready())
	assert.Nil(t, cmd)
}

func TestWelcomeModel_Update_WindowSizeMsg_Large(t *testing.T) {
	styles := getTestStyles()
	m := NewWelcome(styles, "1.0.0")

	msg := tea.WindowSizeMsg{Width: 200, Height: 60}
	updated, _ := m.Update(msg)

	assert.Equal(t, 200, updated.Width())
	assert.Equal(t, 60, updated.Height())
	assert.True(t, updated.Ready())
}

func TestWelcomeModel_Update_WindowSizeMsg_Small(t *testing.T) {
	styles := getTestStyles()
	m := NewWelcome(styles, "1.0.0")

	msg := tea.WindowSizeMsg{Width: 40, Height: 10}
	updated, _ := m.Update(msg)

	assert.Equal(t, 40, updated.Width())
	assert.Equal(t, 10, updated.Height())
	assert.True(t, updated.Ready())
}

// =============================================================================
// Update Tests - Key Navigation (Left/Right)
// =============================================================================

func TestWelcomeModel_Update_LeftKey(t *testing.T) {
	styles := getTestStyles()
	m := NewWelcome(styles, "1.0.0")

	// Initial focus is on button 0
	assert.Equal(t, 0, m.FocusedButtonIndex())

	// Press left - should wrap to last button
	msg := tea.KeyMsg{Type: tea.KeyLeft}
	updated, cmd := m.Update(msg)

	assert.Equal(t, 1, updated.FocusedButtonIndex())
	assert.Nil(t, cmd)
}

func TestWelcomeModel_Update_LeftKey_H(t *testing.T) {
	styles := getTestStyles()
	m := NewWelcome(styles, "1.0.0")

	// Press 'h' for left
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}}
	updated, cmd := m.Update(msg)

	assert.Equal(t, 1, updated.FocusedButtonIndex())
	assert.Nil(t, cmd)
}

func TestWelcomeModel_Update_RightKey(t *testing.T) {
	styles := getTestStyles()
	m := NewWelcome(styles, "1.0.0")

	// Initial focus is on button 0
	assert.Equal(t, 0, m.FocusedButtonIndex())

	// Press right
	msg := tea.KeyMsg{Type: tea.KeyRight}
	updated, cmd := m.Update(msg)

	assert.Equal(t, 1, updated.FocusedButtonIndex())
	assert.Nil(t, cmd)

	// Press right again - should wrap to first button
	updated, _ = updated.Update(msg)
	assert.Equal(t, 0, updated.FocusedButtonIndex())
}

func TestWelcomeModel_Update_RightKey_L(t *testing.T) {
	styles := getTestStyles()
	m := NewWelcome(styles, "1.0.0")

	// Press 'l' for right
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}}
	updated, cmd := m.Update(msg)

	assert.Equal(t, 1, updated.FocusedButtonIndex())
	assert.Nil(t, cmd)
}

func TestWelcomeModel_Update_TabKey(t *testing.T) {
	styles := getTestStyles()
	m := NewWelcome(styles, "1.0.0")

	// Press tab
	msg := tea.KeyMsg{Type: tea.KeyTab}
	updated, cmd := m.Update(msg)

	assert.Equal(t, 1, updated.FocusedButtonIndex())
	assert.Nil(t, cmd)
}

// =============================================================================
// Update Tests - Enter Key (Start Installation)
// =============================================================================

func TestWelcomeModel_Update_Enter_StartInstallation(t *testing.T) {
	styles := getTestStyles()
	m := NewWelcome(styles, "1.0.0")

	// Button 0 is "Start Installation"
	assert.Equal(t, 0, m.FocusedButtonIndex())

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	_, cmd := m.Update(msg)

	require.NotNil(t, cmd)

	// Execute the command and verify it returns StartDetectionMsg
	result := cmd()
	_, ok := result.(StartDetectionMsg)
	assert.True(t, ok, "Expected StartDetectionMsg from command")
}

func TestWelcomeModel_Update_Space_StartInstallation(t *testing.T) {
	styles := getTestStyles()
	m := NewWelcome(styles, "1.0.0")

	// Button 0 is "Start Installation"
	msg := tea.KeyMsg{Type: tea.KeySpace}
	_, cmd := m.Update(msg)

	require.NotNil(t, cmd)

	// Execute the command and verify it returns StartDetectionMsg
	result := cmd()
	_, ok := result.(StartDetectionMsg)
	assert.True(t, ok, "Expected StartDetectionMsg from space key")
}

// =============================================================================
// Update Tests - Enter Key (Exit)
// =============================================================================

func TestWelcomeModel_Update_Enter_Exit(t *testing.T) {
	styles := getTestStyles()
	m := NewWelcome(styles, "1.0.0")

	// Navigate to Exit button
	m.FocusButton(1)
	assert.Equal(t, 1, m.FocusedButtonIndex())

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	_, cmd := m.Update(msg)

	require.NotNil(t, cmd)

	// The command should be tea.Quit
	// tea.Quit returns a special quit message
	result := cmd()
	assert.NotNil(t, result)
}

// =============================================================================
// Update Tests - Quit Keys
// =============================================================================

func TestWelcomeModel_Update_QuitKey_Q(t *testing.T) {
	styles := getTestStyles()
	m := NewWelcome(styles, "1.0.0")

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
	_, cmd := m.Update(msg)

	require.NotNil(t, cmd, "q key should return quit command")
}

func TestWelcomeModel_Update_QuitKey_Escape(t *testing.T) {
	styles := getTestStyles()
	m := NewWelcome(styles, "1.0.0")

	msg := tea.KeyMsg{Type: tea.KeyEscape}
	_, cmd := m.Update(msg)

	require.NotNil(t, cmd, "esc key should return quit command")
}

func TestWelcomeModel_Update_QuitKey_CtrlC(t *testing.T) {
	styles := getTestStyles()
	m := NewWelcome(styles, "1.0.0")

	msg := tea.KeyMsg{Type: tea.KeyCtrlC}
	_, cmd := m.Update(msg)

	require.NotNil(t, cmd, "ctrl+c should return quit command")
}

// =============================================================================
// Update Tests - Help Toggle
// =============================================================================

func TestWelcomeModel_Update_HelpToggle(t *testing.T) {
	styles := getTestStyles()
	m := NewWelcome(styles, "1.0.0")

	assert.False(t, m.IsFullHelpShown())

	// Toggle help on
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}}
	updated, cmd := m.Update(msg)

	assert.True(t, updated.IsFullHelpShown())
	assert.Nil(t, cmd)

	// Toggle help off
	updated, _ = updated.Update(msg)
	assert.False(t, updated.IsFullHelpShown())
}

// =============================================================================
// Update Tests - Unknown Messages
// =============================================================================

func TestWelcomeModel_Update_UnknownKey(t *testing.T) {
	styles := getTestStyles()
	m := NewWelcome(styles, "1.0.0")
	m.SetSize(80, 24)

	initialFocus := m.FocusedButtonIndex()

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}}
	updated, cmd := m.Update(msg)

	// State should remain unchanged
	assert.Equal(t, initialFocus, updated.FocusedButtonIndex())
	assert.Nil(t, cmd)
}

func TestWelcomeModel_Update_UnknownMessage(t *testing.T) {
	styles := getTestStyles()
	m := NewWelcome(styles, "1.0.0")
	m.SetSize(80, 24)

	type customMsg struct{}

	updated, cmd := m.Update(customMsg{})

	// State should remain unchanged
	assert.True(t, updated.Ready())
	assert.Nil(t, cmd)
}

// =============================================================================
// View Tests
// =============================================================================

func TestWelcomeModel_View_NotReady(t *testing.T) {
	styles := getTestStyles()
	m := NewWelcome(styles, "1.0.0")

	view := m.View()

	assert.Equal(t, "Loading...", view)
}

func TestWelcomeModel_View_Ready(t *testing.T) {
	styles := getTestStyles()
	m := NewWelcome(styles, "1.0.0")
	m.SetSize(80, 24)

	view := m.View()

	assert.NotEmpty(t, view)
	assert.NotEqual(t, "Loading...", view)
}

func TestWelcomeModel_View_ContainsLogo(t *testing.T) {
	styles := getTestStyles()
	m := NewWelcome(styles, "1.0.0")
	m.SetSize(100, 40)

	view := m.View()

	// Check for part of the ASCII logo
	assert.Contains(t, view, "IGOR")
}

func TestWelcomeModel_View_ContainsDescription(t *testing.T) {
	styles := getTestStyles()
	m := NewWelcome(styles, "1.0.0")
	m.SetSize(100, 40)

	view := m.View()

	assert.Contains(t, view, "Welcome to Igor")
	assert.Contains(t, view, "NVIDIA driver")
}

func TestWelcomeModel_View_ContainsFeatures(t *testing.T) {
	styles := getTestStyles()
	m := NewWelcome(styles, "1.0.0")
	m.SetSize(100, 40)

	view := m.View()

	assert.Contains(t, view, "Automatic GPU detection")
	assert.Contains(t, view, "Distribution-aware")
	assert.Contains(t, view, "CUDA toolkit")
}

func TestWelcomeModel_View_ContainsButtons(t *testing.T) {
	styles := getTestStyles()
	m := NewWelcome(styles, "1.0.0")
	m.SetSize(100, 40)

	view := m.View()

	assert.Contains(t, view, "Start Installation")
	assert.Contains(t, view, "Exit")
}

func TestWelcomeModel_View_VariousSizes(t *testing.T) {
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
			m := NewWelcome(styles, "1.0.0")
			m.SetSize(tc.width, tc.height)

			view := m.View()

			assert.NotEmpty(t, view)
			assert.NotEqual(t, "Loading...", view)
		})
	}
}

func TestWelcomeModel_View_VerySmallSize(t *testing.T) {
	styles := getTestStyles()
	m := NewWelcome(styles, "1.0.0")
	m.SetSize(10, 5)

	// Should not panic
	view := m.View()
	assert.NotEmpty(t, view)
}

func TestWelcomeModel_View_ZeroHeight(t *testing.T) {
	styles := getTestStyles()
	m := NewWelcome(styles, "1.0.0")
	m.SetSize(80, 0)

	// Should not panic
	view := m.View()
	assert.NotEmpty(t, view)
}

// =============================================================================
// SetSize Tests
// =============================================================================

func TestWelcomeModel_SetSize(t *testing.T) {
	styles := getTestStyles()
	m := NewWelcome(styles, "1.0.0")

	assert.False(t, m.Ready())

	m.SetSize(100, 50)

	assert.True(t, m.Ready())
	assert.Equal(t, 100, m.Width())
	assert.Equal(t, 50, m.Height())
}

func TestWelcomeModel_SetSize_Multiple(t *testing.T) {
	styles := getTestStyles()
	m := NewWelcome(styles, "1.0.0")

	m.SetSize(80, 24)
	assert.Equal(t, 80, m.Width())
	assert.Equal(t, 24, m.Height())

	m.SetSize(120, 40)
	assert.Equal(t, 120, m.Width())
	assert.Equal(t, 40, m.Height())
}

// =============================================================================
// FocusButton Tests
// =============================================================================

func TestWelcomeModel_FocusButton(t *testing.T) {
	styles := getTestStyles()
	m := NewWelcome(styles, "1.0.0")

	m.FocusButton(1)
	assert.Equal(t, 1, m.FocusedButtonIndex())

	m.FocusButton(0)
	assert.Equal(t, 0, m.FocusedButtonIndex())
}

func TestWelcomeModel_FocusButton_InvalidIndex(t *testing.T) {
	styles := getTestStyles()
	m := NewWelcome(styles, "1.0.0")

	initialFocus := m.FocusedButtonIndex()

	// Invalid indices should be ignored by the ButtonGroup
	m.FocusButton(-1)
	assert.Equal(t, initialFocus, m.FocusedButtonIndex())

	m.FocusButton(10)
	assert.Equal(t, initialFocus, m.FocusedButtonIndex())
}

// =============================================================================
// Getter Tests
// =============================================================================

func TestWelcomeModel_Getters(t *testing.T) {
	styles := getTestStyles()
	m := NewWelcome(styles, "2.0.0")
	m.SetSize(100, 50)

	assert.Equal(t, 100, m.Width())
	assert.Equal(t, 50, m.Height())
	assert.True(t, m.Ready())
	assert.Equal(t, "2.0.0", m.Version())
	assert.NotNil(t, m.KeyMap())
}

// =============================================================================
// StartDetectionMsg Tests
// =============================================================================

func TestStartDetectionMsg(t *testing.T) {
	msg := StartDetectionMsg{}
	assert.NotNil(t, msg)
}

// =============================================================================
// Integration Tests
// =============================================================================

func TestWelcomeModel_NavigationFlow(t *testing.T) {
	styles := getTestStyles()
	m := NewWelcome(styles, "1.0.0")
	m.SetSize(80, 24)

	// Start at button 0 (Start Installation)
	assert.Equal(t, 0, m.FocusedButtonIndex())

	// Navigate right to Exit
	msg := tea.KeyMsg{Type: tea.KeyRight}
	m, _ = m.Update(msg)
	assert.Equal(t, 1, m.FocusedButtonIndex())

	// Navigate right again - should wrap to Start
	m, _ = m.Update(msg)
	assert.Equal(t, 0, m.FocusedButtonIndex())

	// Navigate left - should go to Exit
	msg = tea.KeyMsg{Type: tea.KeyLeft}
	m, _ = m.Update(msg)
	assert.Equal(t, 1, m.FocusedButtonIndex())
}

func TestWelcomeModel_FullUserFlow_StartInstallation(t *testing.T) {
	styles := getTestStyles()
	m := NewWelcome(styles, "1.0.0")

	// Simulate window resize
	m, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	assert.True(t, m.Ready())

	// Verify we can render
	view := m.View()
	assert.NotEmpty(t, view)
	assert.Contains(t, view, "Start Installation")

	// Press enter to start
	m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.NotNil(t, cmd)

	result := cmd()
	_, ok := result.(StartDetectionMsg)
	assert.True(t, ok)
}

func TestWelcomeModel_FullUserFlow_Exit(t *testing.T) {
	styles := getTestStyles()
	m := NewWelcome(styles, "1.0.0")

	// Simulate window resize
	m, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	// Navigate to Exit button
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRight})
	assert.Equal(t, 1, m.FocusedButtonIndex())

	// Press enter to exit
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.NotNil(t, cmd)
}

func TestWelcomeModel_HelpFlow(t *testing.T) {
	styles := getTestStyles()
	m := NewWelcome(styles, "1.0.0")
	m.SetSize(80, 24)

	// Help is initially not shown in full
	assert.False(t, m.IsFullHelpShown())

	// Toggle help
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

func TestWelcomeKeyMap_ImplementsHelpKeyMap(t *testing.T) {
	km := DefaultWelcomeKeyMap()

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

func TestWelcomeKeyMap_BindingsHaveHelp(t *testing.T) {
	km := DefaultWelcomeKeyMap()

	// All bindings should have help text
	bindings := []key.Binding{
		km.Start,
		km.Quit,
		km.Left,
		km.Right,
		km.Help,
	}

	for _, b := range bindings {
		help := b.Help()
		assert.NotEmpty(t, help.Key, "binding should have key help")
		assert.NotEmpty(t, help.Desc, "binding should have description")
	}
}

// =============================================================================
// Edge Cases
// =============================================================================

func TestWelcomeModel_RapidKeyPresses(t *testing.T) {
	styles := getTestStyles()
	m := NewWelcome(styles, "1.0.0")
	m.SetSize(80, 24)

	// Rapid navigation
	for i := 0; i < 10; i++ {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRight})
	}

	// Should still be in valid state
	assert.GreaterOrEqual(t, m.FocusedButtonIndex(), 0)
	assert.Less(t, m.FocusedButtonIndex(), 2)
}

func TestWelcomeModel_MultipleSizeChanges(t *testing.T) {
	styles := getTestStyles()
	m := NewWelcome(styles, "1.0.0")

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

func TestWelcomeModel_EmptyStyles(t *testing.T) {
	// Test with zero-value styles
	var styles theme.Styles
	m := NewWelcome(styles, "1.0.0")
	m.SetSize(80, 24)

	// Should not panic
	view := m.View()
	assert.NotEmpty(t, view)
}
