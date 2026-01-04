// Package ui provides the Bubble Tea TUI application for Igor.
// It implements the main model, message handling, and view rendering
// for the interactive NVIDIA driver installation interface.
package ui

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ViewState represents the current view in the application.
type ViewState int

const (
	// ViewWelcome is the initial welcome screen.
	ViewWelcome ViewState = iota
	// ViewDetecting is shown while detecting system information.
	ViewDetecting
	// ViewSystemInfo displays detected system information.
	ViewSystemInfo
	// ViewDriverSelection allows the user to select a driver version.
	ViewDriverSelection
	// ViewConfirmation asks for confirmation before installation.
	ViewConfirmation
	// ViewInstalling is shown during driver installation.
	ViewInstalling
	// ViewComplete is shown when installation is finished.
	ViewComplete
	// ViewError is shown when an error occurs.
	ViewError
)

// String returns the string representation of a ViewState.
func (v ViewState) String() string {
	switch v {
	case ViewWelcome:
		return "Welcome"
	case ViewDetecting:
		return "Detecting"
	case ViewSystemInfo:
		return "SystemInfo"
	case ViewDriverSelection:
		return "DriverSelection"
	case ViewConfirmation:
		return "Confirmation"
	case ViewInstalling:
		return "Installing"
	case ViewComplete:
		return "Complete"
	case ViewError:
		return "Error"
	default:
		return "Unknown"
	}
}

// Model is the main Bubble Tea model for the Igor TUI.
type Model struct {
	// CurrentView is the current view state.
	CurrentView ViewState

	// Width is the terminal width.
	Width int
	// Height is the terminal height.
	Height int

	// Ready indicates if the TUI is ready to render.
	Ready bool
	// Quitting indicates if the application is quitting.
	Quitting bool

	// Error holds any error that occurred.
	Error error

	// ctx is the context for cancellation.
	ctx context.Context
	// cancel is the cancel function for the context.
	cancel context.CancelFunc

	// keyMap holds the key bindings.
	keyMap KeyMap

	// Sub-models (will be added in later sprints)
	// welcomeModel    *WelcomeModel
	// detectionModel  *DetectionModel
	// etc.
}

// New creates a new TUI application model.
func New() Model {
	ctx, cancel := context.WithCancel(context.Background())
	return Model{
		CurrentView: ViewWelcome,
		ctx:         ctx,
		cancel:      cancel,
		keyMap:      DefaultKeyMap(),
	}
}

// NewWithContext creates a new TUI application model with a custom context.
func NewWithContext(ctx context.Context) Model {
	childCtx, cancel := context.WithCancel(ctx)
	return Model{
		CurrentView: ViewWelcome,
		ctx:         childCtx,
		cancel:      cancel,
		keyMap:      DefaultKeyMap(),
	}
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		tea.EnterAltScreen,
		waitForWindowSize,
	)
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKeyPress(msg)

	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		m.Ready = true
		return m, nil

	case QuitMsg:
		m.Quitting = true
		m.cancel()
		return m, tea.Quit

	case ErrorMsg:
		m.Error = msg.Err
		m.CurrentView = ViewError
		return m, nil

	case NavigateMsg:
		m.CurrentView = msg.View
		return m, nil

	case WindowReadyMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		m.Ready = true
		return m, nil
	}

	return m, nil
}

// View implements tea.Model.
func (m Model) View() string {
	if m.Quitting {
		return "Goodbye!\n"
	}

	if !m.Ready {
		return "Initializing..."
	}

	// Placeholder views - will be replaced by actual view models
	switch m.CurrentView {
	case ViewWelcome:
		return m.renderPlaceholder("Welcome to Igor - NVIDIA Driver Installer")
	case ViewDetecting:
		return m.renderPlaceholder("Detecting system...")
	case ViewSystemInfo:
		return m.renderPlaceholder("System Information")
	case ViewDriverSelection:
		return m.renderPlaceholder("Select Driver Version")
	case ViewConfirmation:
		return m.renderPlaceholder("Confirm Installation")
	case ViewInstalling:
		return m.renderPlaceholder("Installing...")
	case ViewComplete:
		return m.renderPlaceholder("Installation Complete")
	case ViewError:
		return m.renderError()
	default:
		return m.renderPlaceholder("Unknown View")
	}
}

// renderPlaceholder renders a centered placeholder view.
func (m Model) renderPlaceholder(title string) string {
	// Simple centered placeholder
	style := lipgloss.NewStyle().
		Width(m.Width).
		Height(m.Height).
		Align(lipgloss.Center, lipgloss.Center)

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("86"))

	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241"))

	content := titleStyle.Render(title) + "\n\n" +
		helpStyle.Render("Press 'q' to quit, '?' for help")

	return style.Render(content)
}

// renderError renders the error view.
func (m Model) renderError() string {
	errMsg := "Unknown error"
	if m.Error != nil {
		errMsg = m.Error.Error()
	}

	style := lipgloss.NewStyle().
		Width(m.Width).
		Height(m.Height).
		Align(lipgloss.Center, lipgloss.Center)

	errStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("196"))

	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241"))

	content := errStyle.Render("Error: "+errMsg) + "\n\n" +
		helpStyle.Render("Press 'q' to quit, 'esc' to go back")

	return style.Render(content)
}

// handleKeyPress handles key press events.
func (m Model) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, func() tea.Msg { return QuitMsg{} }

	case "esc":
		// Go back to previous view or quit from welcome
		if m.CurrentView == ViewWelcome {
			return m, func() tea.Msg { return QuitMsg{} }
		}
		// For now, go back to welcome
		m.CurrentView = ViewWelcome
		m.Error = nil // Clear any error when navigating away
		return m, nil
	}

	return m, nil
}

// waitForWindowSize returns a command that signals the window is ready.
// This is a placeholder that will be triggered by the actual WindowSizeMsg.
func waitForWindowSize() tea.Msg {
	return nil
}

// Context returns the application context.
func (m Model) Context() context.Context {
	return m.ctx
}

// Shutdown cancels the context and performs cleanup.
func (m *Model) Shutdown() {
	if m.cancel != nil {
		m.cancel()
	}
}

// KeyMap returns the current key bindings.
func (m Model) KeyMap() KeyMap {
	return m.keyMap
}

// SetKeyMap sets custom key bindings.
func (m *Model) SetKeyMap(km KeyMap) {
	m.keyMap = km
}

// IsReady returns whether the TUI is ready to render.
func (m Model) IsReady() bool {
	return m.Ready
}

// IsQuitting returns whether the application is quitting.
func (m Model) IsQuitting() bool {
	return m.Quitting
}

// SetError sets an error and transitions to the error view.
func (m *Model) SetError(err error) {
	m.Error = err
	m.CurrentView = ViewError
}

// ClearError clears the error and returns to the welcome view.
func (m *Model) ClearError() {
	m.Error = nil
	m.CurrentView = ViewWelcome
}

// NavigateTo transitions to the specified view.
func (m *Model) NavigateTo(view ViewState) {
	m.CurrentView = view
}
