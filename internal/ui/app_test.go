package ui

import (
	"context"
	"errors"
	"testing"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// ViewState Tests
// =============================================================================

func TestViewState_String(t *testing.T) {
	tests := []struct {
		state    ViewState
		expected string
	}{
		{ViewWelcome, "Welcome"},
		{ViewDetecting, "Detecting"},
		{ViewSystemInfo, "SystemInfo"},
		{ViewDriverSelection, "DriverSelection"},
		{ViewConfirmation, "Confirmation"},
		{ViewInstalling, "Installing"},
		{ViewComplete, "Complete"},
		{ViewError, "Error"},
		{ViewState(100), "Unknown"}, // Unknown state
	}

	for _, tc := range tests {
		t.Run(tc.expected, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.state.String())
		})
	}
}

// =============================================================================
// Model Creation Tests
// =============================================================================

func TestNew(t *testing.T) {
	m := New()

	assert.Equal(t, ViewWelcome, m.CurrentView)
	assert.Equal(t, 0, m.Width)
	assert.Equal(t, 0, m.Height)
	assert.False(t, m.Ready)
	assert.False(t, m.Quitting)
	assert.Nil(t, m.Error)
	assert.NotNil(t, m.ctx)
	assert.NotNil(t, m.cancel)
	assert.NotNil(t, m.keyMap.Quit.Keys())
}

func TestNewWithContext(t *testing.T) {
	parentCtx, parentCancel := context.WithCancel(context.Background())
	defer parentCancel()

	m := NewWithContext(parentCtx)

	assert.Equal(t, ViewWelcome, m.CurrentView)
	assert.NotNil(t, m.ctx)
	assert.NotNil(t, m.cancel)

	// Child context should be derived from parent
	parentCancel()

	// Check that context is cancelled
	select {
	case <-m.ctx.Done():
		// Expected - parent cancellation propagates to child
	default:
		t.Error("Expected context to be cancelled when parent is cancelled")
	}
}

func TestNew_InitialKeyMap(t *testing.T) {
	m := New()

	// Verify key map is initialized with default values
	assert.NotEmpty(t, m.keyMap.Quit.Keys())
	assert.NotEmpty(t, m.keyMap.Help.Keys())
	assert.NotEmpty(t, m.keyMap.Up.Keys())
	assert.NotEmpty(t, m.keyMap.Down.Keys())
}

// =============================================================================
// Init Tests
// =============================================================================

func TestModel_Init(t *testing.T) {
	m := New()

	cmd := m.Init()

	// Init should return a batch command
	assert.NotNil(t, cmd)

	// Execute the command to see what messages it produces
	// The batch should include EnterAltScreen
	// Note: We can't easily inspect batch commands, but we verify it's not nil
}

// =============================================================================
// Update Tests - WindowSizeMsg
// =============================================================================

func TestModel_Update_WindowSizeMsg(t *testing.T) {
	m := New()
	assert.False(t, m.Ready)

	msg := tea.WindowSizeMsg{Width: 80, Height: 24}
	newModel, cmd := m.Update(msg)

	updatedModel := newModel.(Model)
	assert.Equal(t, 80, updatedModel.Width)
	assert.Equal(t, 24, updatedModel.Height)
	assert.True(t, updatedModel.Ready)
	assert.Nil(t, cmd)
}

func TestModel_Update_WindowSizeMsg_Large(t *testing.T) {
	m := New()

	msg := tea.WindowSizeMsg{Width: 200, Height: 60}
	newModel, _ := m.Update(msg)

	updatedModel := newModel.(Model)
	assert.Equal(t, 200, updatedModel.Width)
	assert.Equal(t, 60, updatedModel.Height)
}

// =============================================================================
// Update Tests - QuitMsg
// =============================================================================

func TestModel_Update_QuitMsg(t *testing.T) {
	m := New()

	msg := QuitMsg{}
	newModel, cmd := m.Update(msg)

	updatedModel := newModel.(Model)
	assert.True(t, updatedModel.Quitting)
	assert.NotNil(t, cmd)

	// Verify context is cancelled
	select {
	case <-updatedModel.ctx.Done():
		// Expected
	default:
		t.Error("Expected context to be cancelled on quit")
	}
}

// =============================================================================
// Update Tests - ErrorMsg
// =============================================================================

func TestModel_Update_ErrorMsg(t *testing.T) {
	m := New()
	testErr := errors.New("test error")

	msg := ErrorMsg{Err: testErr}
	newModel, cmd := m.Update(msg)

	updatedModel := newModel.(Model)
	assert.Equal(t, ViewError, updatedModel.CurrentView)
	assert.Equal(t, testErr, updatedModel.Error)
	assert.Nil(t, cmd)
}

func TestModel_Update_ErrorMsg_NilError(t *testing.T) {
	m := New()

	msg := ErrorMsg{Err: nil}
	newModel, _ := m.Update(msg)

	updatedModel := newModel.(Model)
	assert.Equal(t, ViewError, updatedModel.CurrentView)
	assert.Nil(t, updatedModel.Error)
}

// =============================================================================
// Update Tests - NavigateMsg
// =============================================================================

func TestModel_Update_NavigateMsg(t *testing.T) {
	testCases := []struct {
		name   string
		toView ViewState
	}{
		{"Navigate to Detecting", ViewDetecting},
		{"Navigate to SystemInfo", ViewSystemInfo},
		{"Navigate to DriverSelection", ViewDriverSelection},
		{"Navigate to Confirmation", ViewConfirmation},
		{"Navigate to Installing", ViewInstalling},
		{"Navigate to Complete", ViewComplete},
		{"Navigate to Error", ViewError},
		{"Navigate to Welcome", ViewWelcome},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			m := New()
			msg := NavigateMsg{View: tc.toView}

			newModel, cmd := m.Update(msg)

			updatedModel := newModel.(Model)
			assert.Equal(t, tc.toView, updatedModel.CurrentView)
			assert.Nil(t, cmd)
		})
	}
}

// =============================================================================
// Update Tests - WindowReadyMsg
// =============================================================================

func TestModel_Update_WindowReadyMsg(t *testing.T) {
	m := New()

	msg := WindowReadyMsg{Width: 100, Height: 50}
	newModel, cmd := m.Update(msg)

	updatedModel := newModel.(Model)
	assert.Equal(t, 100, updatedModel.Width)
	assert.Equal(t, 50, updatedModel.Height)
	assert.True(t, updatedModel.Ready)
	assert.Nil(t, cmd)
}

// =============================================================================
// Update Tests - KeyMsg
// =============================================================================

func TestModel_Update_KeyMsg_Quit(t *testing.T) {
	m := New()
	m.Ready = true

	testCases := []struct {
		name string
		key  string
	}{
		{"q key", "q"},
		{"ctrl+c", "ctrl+c"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(tc.key)}
			if tc.key == "ctrl+c" {
				msg = tea.KeyMsg{Type: tea.KeyCtrlC}
			}

			newModel, cmd := m.Update(msg)
			_ = newModel.(Model)

			// Command should produce QuitMsg
			assert.NotNil(t, cmd)
			result := cmd()
			_, ok := result.(QuitMsg)
			assert.True(t, ok, "Expected QuitMsg from command")
		})
	}
}

func TestModel_Update_KeyMsg_Escape_FromWelcome(t *testing.T) {
	m := New()
	m.Ready = true
	m.CurrentView = ViewWelcome

	msg := tea.KeyMsg{Type: tea.KeyEscape}
	_, cmd := m.Update(msg)

	// Escape from welcome should produce QuitMsg
	assert.NotNil(t, cmd)
	result := cmd()
	_, ok := result.(QuitMsg)
	assert.True(t, ok, "Expected QuitMsg from escape at welcome screen")
}

func TestModel_Update_KeyMsg_Escape_FromOtherViews(t *testing.T) {
	views := []ViewState{
		ViewDetecting,
		ViewSystemInfo,
		ViewDriverSelection,
		ViewConfirmation,
		ViewInstalling,
		ViewComplete,
		ViewError,
	}

	for _, view := range views {
		t.Run(view.String(), func(t *testing.T) {
			m := New()
			m.Ready = true
			m.CurrentView = view
			m.Error = errors.New("test error") // Set an error to test clearing

			msg := tea.KeyMsg{Type: tea.KeyEscape}
			newModel, cmd := m.Update(msg)

			updatedModel := newModel.(Model)
			assert.Equal(t, ViewWelcome, updatedModel.CurrentView)
			assert.Nil(t, updatedModel.Error, "Error should be cleared on escape")
			assert.Nil(t, cmd)
		})
	}
}

func TestModel_Update_KeyMsg_Unknown(t *testing.T) {
	m := New()
	m.Ready = true

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")}
	newModel, cmd := m.Update(msg)

	updatedModel := newModel.(Model)
	// Unknown key should not change state
	assert.Equal(t, ViewWelcome, updatedModel.CurrentView)
	assert.Nil(t, cmd)
}

// =============================================================================
// View Tests
// =============================================================================

func TestModel_View_NotReady(t *testing.T) {
	m := New()
	m.Ready = false

	view := m.View()

	assert.Equal(t, "Initializing...", view)
}

func TestModel_View_Quitting(t *testing.T) {
	m := New()
	m.Quitting = true

	view := m.View()

	assert.Equal(t, "Goodbye!\n", view)
}

func TestModel_View_AllStates(t *testing.T) {
	testCases := []struct {
		state           ViewState
		expectedContain string
	}{
		{ViewWelcome, "Welcome to Igor"},
		{ViewDetecting, "Detecting system"},
		{ViewSystemInfo, "System Information"},
		{ViewDriverSelection, "Select Driver Version"},
		{ViewConfirmation, "Confirm Installation"},
		{ViewInstalling, "Installing"},
		{ViewComplete, "Installation Complete"},
	}

	for _, tc := range testCases {
		t.Run(tc.state.String(), func(t *testing.T) {
			m := New()
			m.Ready = true
			m.Width = 80
			m.Height = 24
			m.CurrentView = tc.state

			view := m.View()

			assert.Contains(t, view, tc.expectedContain)
			assert.Contains(t, view, "Press 'q' to quit")
		})
	}
}

func TestModel_View_Error_WithMessage(t *testing.T) {
	m := New()
	m.Ready = true
	m.Width = 80
	m.Height = 24
	m.CurrentView = ViewError
	m.Error = errors.New("something went wrong")

	view := m.View()

	assert.Contains(t, view, "Error:")
	assert.Contains(t, view, "something went wrong")
}

func TestModel_View_Error_NilError(t *testing.T) {
	m := New()
	m.Ready = true
	m.Width = 80
	m.Height = 24
	m.CurrentView = ViewError
	m.Error = nil

	view := m.View()

	assert.Contains(t, view, "Error:")
	assert.Contains(t, view, "Unknown error")
}

func TestModel_View_UnknownState(t *testing.T) {
	m := New()
	m.Ready = true
	m.Width = 80
	m.Height = 24
	m.CurrentView = ViewState(999) // Unknown state

	view := m.View()

	assert.Contains(t, view, "Unknown View")
}

// =============================================================================
// Helper Method Tests
// =============================================================================

func TestModel_Context(t *testing.T) {
	m := New()

	ctx := m.Context()

	assert.NotNil(t, ctx)
	assert.NoError(t, ctx.Err()) // Context should not be cancelled
}

func TestModel_Shutdown(t *testing.T) {
	m := New()

	// Verify context is not cancelled before shutdown
	select {
	case <-m.ctx.Done():
		t.Error("Context should not be cancelled before Shutdown")
	default:
		// Expected
	}

	m.Shutdown()

	// Verify context is cancelled after shutdown
	select {
	case <-m.ctx.Done():
		// Expected
	default:
		t.Error("Context should be cancelled after Shutdown")
	}
}

func TestModel_Shutdown_Multiple(t *testing.T) {
	m := New()

	// Calling Shutdown multiple times should not panic
	m.Shutdown()
	m.Shutdown()
	m.Shutdown()

	// Context should still be cancelled
	assert.Error(t, m.ctx.Err())
}

func TestModel_KeyMap(t *testing.T) {
	m := New()

	km := m.KeyMap()

	assert.NotEmpty(t, km.Quit.Keys())
	assert.NotEmpty(t, km.Help.Keys())
}

func TestModel_SetKeyMap(t *testing.T) {
	m := New()

	newKeyMap := KeyMap{
		Quit: key.NewBinding(
			key.WithKeys("x"),
			key.WithHelp("x", "exit"),
		),
	}

	m.SetKeyMap(newKeyMap)

	assert.Equal(t, []string{"x"}, m.keyMap.Quit.Keys())
}

func TestModel_IsReady(t *testing.T) {
	m := New()

	assert.False(t, m.IsReady())

	m.Ready = true
	assert.True(t, m.IsReady())
}

func TestModel_IsQuitting(t *testing.T) {
	m := New()

	assert.False(t, m.IsQuitting())

	m.Quitting = true
	assert.True(t, m.IsQuitting())
}

func TestModel_SetError(t *testing.T) {
	m := New()
	m.CurrentView = ViewWelcome
	testErr := errors.New("test error")

	m.SetError(testErr)

	assert.Equal(t, ViewError, m.CurrentView)
	assert.Equal(t, testErr, m.Error)
}

func TestModel_ClearError(t *testing.T) {
	m := New()
	m.Error = errors.New("test error")
	m.CurrentView = ViewError

	m.ClearError()

	assert.Nil(t, m.Error)
	assert.Equal(t, ViewWelcome, m.CurrentView)
}

func TestModel_NavigateTo(t *testing.T) {
	m := New()

	views := []ViewState{
		ViewDetecting,
		ViewSystemInfo,
		ViewDriverSelection,
		ViewConfirmation,
		ViewInstalling,
		ViewComplete,
		ViewError,
		ViewWelcome,
	}

	for _, view := range views {
		m.NavigateTo(view)
		assert.Equal(t, view, m.CurrentView)
	}
}

// =============================================================================
// Message Constructor Tests
// =============================================================================

func TestNavigate_Command(t *testing.T) {
	cmd := Navigate(ViewSystemInfo)
	require.NotNil(t, cmd)

	msg := cmd()
	navMsg, ok := msg.(NavigateMsg)

	assert.True(t, ok)
	assert.Equal(t, ViewSystemInfo, navMsg.View)
}

func TestReportError_Command(t *testing.T) {
	testErr := errors.New("test error")
	cmd := ReportError(testErr)
	require.NotNil(t, cmd)

	msg := cmd()
	errMsg, ok := msg.(ErrorMsg)

	assert.True(t, ok)
	assert.Equal(t, testErr, errMsg.Err)
}

func TestQuit_Command(t *testing.T) {
	cmd := Quit()
	require.NotNil(t, cmd)

	msg := cmd()
	_, ok := msg.(QuitMsg)

	assert.True(t, ok)
}

func TestSendProgress_Command(t *testing.T) {
	cmd := SendProgress(5, 10, "Processing...")
	require.NotNil(t, cmd)

	msg := cmd()
	progressMsg, ok := msg.(ProgressMsg)

	assert.True(t, ok)
	assert.Equal(t, 5, progressMsg.Current)
	assert.Equal(t, 10, progressMsg.Total)
	assert.Equal(t, "Processing...", progressMsg.Message)
}

func TestSendStatus_Command(t *testing.T) {
	cmd := SendStatus("Operation complete", false)
	require.NotNil(t, cmd)

	msg := cmd()
	statusMsg, ok := msg.(StatusMsg)

	assert.True(t, ok)
	assert.Equal(t, "Operation complete", statusMsg.Message)
	assert.False(t, statusMsg.IsError)

	// Test with error status
	cmd = SendStatus("Operation failed", true)
	msg = cmd()
	statusMsg, ok = msg.(StatusMsg)

	assert.True(t, ok)
	assert.Equal(t, "Operation failed", statusMsg.Message)
	assert.True(t, statusMsg.IsError)
}

func TestSendTick_Command(t *testing.T) {
	cmd := SendTick()
	require.NotNil(t, cmd)

	msg := cmd()
	_, ok := msg.(TickMsg)

	assert.True(t, ok)
}

func TestSendWindowReady_Command(t *testing.T) {
	cmd := SendWindowReady(120, 40)
	require.NotNil(t, cmd)

	msg := cmd()
	readyMsg, ok := msg.(WindowReadyMsg)

	assert.True(t, ok)
	assert.Equal(t, 120, readyMsg.Width)
	assert.Equal(t, 40, readyMsg.Height)
}

// =============================================================================
// KeyMap Tests
// =============================================================================

func TestDefaultKeyMap(t *testing.T) {
	km := DefaultKeyMap()

	// Verify all keys are set
	assert.NotEmpty(t, km.Quit.Keys())
	assert.NotEmpty(t, km.Help.Keys())
	assert.NotEmpty(t, km.Up.Keys())
	assert.NotEmpty(t, km.Down.Keys())
	assert.NotEmpty(t, km.Left.Keys())
	assert.NotEmpty(t, km.Right.Keys())
	assert.NotEmpty(t, km.Enter.Keys())
	assert.NotEmpty(t, km.Back.Keys())
	assert.NotEmpty(t, km.Tab.Keys())
	assert.NotEmpty(t, km.Space.Keys())
	assert.NotEmpty(t, km.PageUp.Keys())
	assert.NotEmpty(t, km.PageDown.Keys())
	assert.NotEmpty(t, km.Home.Keys())
	assert.NotEmpty(t, km.End.Keys())
}

func TestKeyMap_VimStyleNavigation(t *testing.T) {
	km := DefaultKeyMap()

	// Verify vim-style hjkl navigation
	assert.Contains(t, km.Up.Keys(), "k")
	assert.Contains(t, km.Down.Keys(), "j")
	assert.Contains(t, km.Left.Keys(), "h")
	assert.Contains(t, km.Right.Keys(), "l")
}

func TestKeyMap_ShortHelp(t *testing.T) {
	km := DefaultKeyMap()

	help := km.ShortHelp()

	assert.Len(t, help, 2)
	// Should contain Help and Quit
	assert.Equal(t, km.Help, help[0])
	assert.Equal(t, km.Quit, help[1])
}

func TestKeyMap_FullHelp(t *testing.T) {
	km := DefaultKeyMap()

	help := km.FullHelp()

	assert.Len(t, help, 4) // 4 rows of bindings

	// First row: navigation
	assert.Len(t, help[0], 4) // Up, Down, Left, Right

	// Second row: actions
	assert.Len(t, help[1], 4) // Enter, Back, Tab, Space

	// Third row: paging
	assert.Len(t, help[2], 4) // PageUp, PageDown, Home, End

	// Fourth row: general
	assert.Len(t, help[3], 2) // Help, Quit
}

func TestKeyMap_NavigationKeys(t *testing.T) {
	km := DefaultKeyMap()

	navKeys := km.NavigationKeys()

	assert.Len(t, navKeys, 8)
	assert.Contains(t, navKeys, km.Up)
	assert.Contains(t, navKeys, km.Down)
	assert.Contains(t, navKeys, km.Left)
	assert.Contains(t, navKeys, km.Right)
	assert.Contains(t, navKeys, km.PageUp)
	assert.Contains(t, navKeys, km.PageDown)
	assert.Contains(t, navKeys, km.Home)
	assert.Contains(t, navKeys, km.End)
}

func TestKeyMap_ActionKeys(t *testing.T) {
	km := DefaultKeyMap()

	actionKeys := km.ActionKeys()

	assert.Len(t, actionKeys, 4)
	assert.Contains(t, actionKeys, km.Enter)
	assert.Contains(t, actionKeys, km.Back)
	assert.Contains(t, actionKeys, km.Tab)
	assert.Contains(t, actionKeys, km.Space)
}

func TestKeyMap_DisableNavigation(t *testing.T) {
	km := DefaultKeyMap()

	km.DisableNavigation()

	assert.False(t, km.Up.Enabled())
	assert.False(t, km.Down.Enabled())
	assert.False(t, km.Left.Enabled())
	assert.False(t, km.Right.Enabled())
	assert.False(t, km.PageUp.Enabled())
	assert.False(t, km.PageDown.Enabled())
	assert.False(t, km.Home.Enabled())
	assert.False(t, km.End.Enabled())
}

func TestKeyMap_EnableNavigation(t *testing.T) {
	km := DefaultKeyMap()
	km.DisableNavigation()

	km.EnableNavigation()

	assert.True(t, km.Up.Enabled())
	assert.True(t, km.Down.Enabled())
	assert.True(t, km.Left.Enabled())
	assert.True(t, km.Right.Enabled())
	assert.True(t, km.PageUp.Enabled())
	assert.True(t, km.PageDown.Enabled())
	assert.True(t, km.Home.Enabled())
	assert.True(t, km.End.Enabled())
}

func TestKeyMap_DisableActions(t *testing.T) {
	km := DefaultKeyMap()

	km.DisableActions()

	assert.False(t, km.Enter.Enabled())
	assert.False(t, km.Back.Enabled())
	assert.False(t, km.Tab.Enabled())
	assert.False(t, km.Space.Enabled())
}

func TestKeyMap_EnableActions(t *testing.T) {
	km := DefaultKeyMap()
	km.DisableActions()

	km.EnableActions()

	assert.True(t, km.Enter.Enabled())
	assert.True(t, km.Back.Enabled())
	assert.True(t, km.Tab.Enabled())
	assert.True(t, km.Space.Enabled())
}

func TestKeyMap_DisableAll(t *testing.T) {
	km := DefaultKeyMap()

	km.DisableAll()

	// Navigation should be disabled
	assert.False(t, km.Up.Enabled())
	assert.False(t, km.Down.Enabled())

	// Actions should be disabled
	assert.False(t, km.Enter.Enabled())
	assert.False(t, km.Back.Enabled())

	// Help should be disabled
	assert.False(t, km.Help.Enabled())

	// Note: Quit is intentionally kept - should still work
}

func TestKeyMap_EnableAll(t *testing.T) {
	km := DefaultKeyMap()
	km.DisableAll()

	km.EnableAll()

	// All keys should be enabled
	assert.True(t, km.Up.Enabled())
	assert.True(t, km.Down.Enabled())
	assert.True(t, km.Enter.Enabled())
	assert.True(t, km.Back.Enabled())
	assert.True(t, km.Help.Enabled())
	assert.True(t, km.Quit.Enabled())
}

// =============================================================================
// Message Type Tests
// =============================================================================

func TestMessageTypes(t *testing.T) {
	// Verify message types can be created and contain expected data

	t.Run("QuitMsg", func(t *testing.T) {
		msg := QuitMsg{}
		assert.NotNil(t, msg)
	})

	t.Run("ErrorMsg", func(t *testing.T) {
		err := errors.New("test")
		msg := ErrorMsg{Err: err}
		assert.Equal(t, err, msg.Err)
	})

	t.Run("NavigateMsg", func(t *testing.T) {
		msg := NavigateMsg{View: ViewComplete}
		assert.Equal(t, ViewComplete, msg.View)
	})

	t.Run("WindowReadyMsg", func(t *testing.T) {
		msg := WindowReadyMsg{Width: 100, Height: 50}
		assert.Equal(t, 100, msg.Width)
		assert.Equal(t, 50, msg.Height)
	})

	t.Run("TickMsg", func(t *testing.T) {
		msg := TickMsg{}
		assert.NotNil(t, msg)
	})

	t.Run("StatusMsg", func(t *testing.T) {
		msg := StatusMsg{Message: "test", IsError: true}
		assert.Equal(t, "test", msg.Message)
		assert.True(t, msg.IsError)
	})

	t.Run("ProgressMsg", func(t *testing.T) {
		msg := ProgressMsg{Current: 5, Total: 10, Message: "progress"}
		assert.Equal(t, 5, msg.Current)
		assert.Equal(t, 10, msg.Total)
		assert.Equal(t, "progress", msg.Message)
	})

	t.Run("DetectionCompleteMsg", func(t *testing.T) {
		msg := DetectionCompleteMsg{Success: true}
		assert.True(t, msg.Success)
	})

	t.Run("InstallationCompleteMsg", func(t *testing.T) {
		msg := InstallationCompleteMsg{Success: true, Message: "done"}
		assert.True(t, msg.Success)
		assert.Equal(t, "done", msg.Message)
	})
}

// =============================================================================
// Integration Tests
// =============================================================================

func TestModel_FullFlow(t *testing.T) {
	m := New()

	// Simulate window size event
	newModel, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = newModel.(Model)
	assert.True(t, m.Ready)

	// Navigate through views
	newModel, _ = m.Update(NavigateMsg{View: ViewDetecting})
	m = newModel.(Model)
	assert.Equal(t, ViewDetecting, m.CurrentView)

	newModel, _ = m.Update(NavigateMsg{View: ViewSystemInfo})
	m = newModel.(Model)
	assert.Equal(t, ViewSystemInfo, m.CurrentView)

	// Trigger an error
	testErr := errors.New("test error")
	newModel, _ = m.Update(ErrorMsg{Err: testErr})
	m = newModel.(Model)
	assert.Equal(t, ViewError, m.CurrentView)
	assert.Equal(t, testErr, m.Error)

	// Go back to welcome with escape
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	m = newModel.(Model)
	assert.Equal(t, ViewWelcome, m.CurrentView)
	assert.Nil(t, m.Error)
}

func TestModel_RenderPlaceholder_Dimensions(t *testing.T) {
	m := New()
	m.Ready = true
	m.Width = 100
	m.Height = 30

	// Render should not panic with various dimensions
	view := m.renderPlaceholder("Test Title")
	assert.NotEmpty(t, view)

	// Test with minimal dimensions
	m.Width = 20
	m.Height = 5
	view = m.renderPlaceholder("Test Title")
	assert.NotEmpty(t, view)

	// Test with zero dimensions
	m.Width = 0
	m.Height = 0
	view = m.renderPlaceholder("Test Title")
	assert.NotEmpty(t, view)
}

func TestModel_RenderError_Dimensions(t *testing.T) {
	m := New()
	m.Ready = true
	m.Width = 100
	m.Height = 30
	m.Error = errors.New("test error")

	// Render should not panic with various dimensions
	view := m.renderError()
	assert.NotEmpty(t, view)
	assert.Contains(t, view, "test error")
}

// =============================================================================
// Edge Cases
// =============================================================================

func TestModel_Update_UnknownMessage(t *testing.T) {
	m := New()

	// Create a custom message type not handled by Update
	type customMsg struct{}

	newModel, cmd := m.Update(customMsg{})

	updatedModel := newModel.(Model)
	// State should remain unchanged
	assert.Equal(t, ViewWelcome, updatedModel.CurrentView)
	assert.Nil(t, cmd)
}

func TestModel_QuittingTakesPrecedence(t *testing.T) {
	m := New()
	m.Ready = true
	m.Quitting = true
	m.CurrentView = ViewError // Even with error view set

	view := m.View()

	// Quitting should take precedence
	assert.Equal(t, "Goodbye!\n", view)
}

func TestModel_NotReadyTakesPrecedence(t *testing.T) {
	m := New()
	m.Ready = false
	m.CurrentView = ViewComplete // Even with other view set

	view := m.View()

	// Not ready should take precedence
	assert.Equal(t, "Initializing...", view)
}

func TestWaitForWindowSize(t *testing.T) {
	// This function returns nil as a placeholder
	result := waitForWindowSize()
	assert.Nil(t, result)
}
