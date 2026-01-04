// Package views provides the different screen views for the Igor TUI application.
// Each view represents a distinct screen in the user interface workflow.
package views

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/tungetti/igor/internal/ui/components"
	"github.com/tungetti/igor/internal/ui/theme"
)

// StartDetectionMsg signals that detection should begin.
// This message is sent when the user selects "Start Installation" on the welcome screen.
type StartDetectionMsg struct{}

// WelcomeKeyMap defines key bindings for the welcome screen.
type WelcomeKeyMap struct {
	Start key.Binding
	Quit  key.Binding
	Left  key.Binding
	Right key.Binding
	Help  key.Binding
}

// DefaultWelcomeKeyMap returns the default key bindings for the welcome screen.
func DefaultWelcomeKeyMap() WelcomeKeyMap {
	return WelcomeKeyMap{
		Start: key.NewBinding(
			key.WithKeys("enter", " "),
			key.WithHelp("enter", "start"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "esc", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
		Left: key.NewBinding(
			key.WithKeys("left", "h"),
			key.WithHelp("<-/h", "previous"),
		),
		Right: key.NewBinding(
			key.WithKeys("right", "l", "tab"),
			key.WithHelp("->/l", "next"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
	}
}

// ShortHelp implements help.KeyMap interface.
// Returns key bindings for the short help view.
func (k WelcomeKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Start, k.Quit, k.Help}
}

// FullHelp implements help.KeyMap interface.
// Returns key bindings for the full help view.
func (k WelcomeKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Start, k.Quit},
		{k.Left, k.Right, k.Help},
	}
}

// WelcomeModel represents the welcome screen view.
// This is the first screen users see when launching Igor.
type WelcomeModel struct {
	// Dimensions
	width  int
	height int

	// Components
	header  components.HeaderModel
	footer  components.FooterModel
	buttons components.ButtonGroup

	// State
	ready  bool
	styles theme.Styles
	keyMap WelcomeKeyMap

	// App info
	version string
}

// NewWelcome creates a new welcome screen model.
func NewWelcome(styles theme.Styles, version string) WelcomeModel {
	keyMap := DefaultWelcomeKeyMap()

	header := components.NewHeader(styles, "IGOR", "NVIDIA Driver Installer", version)
	footer := components.NewFooter(styles, keyMap)
	buttons := components.NewButtonGroup(styles, "Start Installation", "Exit")

	return WelcomeModel{
		header:  header,
		footer:  footer,
		buttons: buttons,
		styles:  styles,
		keyMap:  keyMap,
		version: version,
	}
}

// Init implements tea.Model.
// The welcome screen doesn't need any initialization commands.
func (m WelcomeModel) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model and handles messages for the welcome screen.
func (m WelcomeModel) Update(msg tea.Msg) (WelcomeModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keyMap.Start):
			// Return command based on selected button
			if m.buttons.FocusedIndex() == 0 {
				// Start installation - navigate to detection
				return m, m.startDetection
			}
			// Exit
			return m, tea.Quit

		case key.Matches(msg, m.keyMap.Quit):
			return m, tea.Quit

		case key.Matches(msg, m.keyMap.Left):
			m.buttons.Previous()
			return m, nil

		case key.Matches(msg, m.keyMap.Right):
			m.buttons.Next()
			return m, nil

		case key.Matches(msg, m.keyMap.Help):
			m.footer.ToggleFullHelp()
			return m, nil
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.header.SetWidth(msg.Width)
		m.footer.SetWidth(msg.Width)
		m.ready = true
		return m, nil
	}

	return m, nil
}

// View implements tea.Model and renders the welcome screen.
func (m WelcomeModel) View() string {
	if !m.ready {
		return "Loading..."
	}

	// Build the welcome screen layout
	header := m.header.View()
	footer := m.footer.View()

	// Calculate content area height
	headerHeight := lipgloss.Height(header)
	footerHeight := lipgloss.Height(footer)
	contentHeight := m.height - headerHeight - footerHeight

	// Main content
	content := m.renderContent(contentHeight)

	return lipgloss.JoinVertical(lipgloss.Left,
		header,
		content,
		footer,
	)
}

// renderContent renders the main content area of the welcome screen.
func (m WelcomeModel) renderContent(height int) string {
	// ASCII art logo
	logo := m.renderLogo()

	// Description
	description := m.styles.Paragraph.Render(
		"Welcome to Igor, the NVIDIA driver installation assistant.\n" +
			"This tool will help you install the appropriate NVIDIA drivers\n" +
			"for your system with minimal hassle.",
	)

	// Features list
	features := m.renderFeatures()

	// Buttons
	buttonRow := lipgloss.NewStyle().
		Width(m.width).
		Align(lipgloss.Center).
		MarginTop(2).
		Render(m.buttons.View())

	// Combine content
	content := lipgloss.JoinVertical(lipgloss.Center,
		logo,
		"",
		description,
		"",
		features,
		buttonRow,
	)

	// Ensure height is at least 1
	if height < 1 {
		height = 1
	}

	// Center vertically in available space
	contentStyle := lipgloss.NewStyle().
		Width(m.width).
		Height(height).
		Align(lipgloss.Center, lipgloss.Center)

	return contentStyle.Render(content)
}

// renderLogo renders the ASCII art logo.
func (m WelcomeModel) renderLogo() string {
	logo := ` _____ _____  ____  _____  
|_   _/ ____|/ __ \|  __ \ 
  | || |  __| |  | | |__) |
  | || | |_ | |  | |  _  / 
 _| || |__| | |__| | | \ \ 
|_____\_____|\____/|_|  \_\`

	return m.styles.Logo.Render(logo)
}

// renderFeatures renders the feature list.
func (m WelcomeModel) renderFeatures() string {
	features := []string{
		"  * Automatic GPU detection",
		"  * Distribution-aware package management",
		"  * Safe driver installation with rollback support",
		"  * CUDA toolkit installation (optional)",
	}

	featureStyle := m.styles.Help
	var rendered []string
	for _, f := range features {
		rendered = append(rendered, featureStyle.Render(f))
	}

	return lipgloss.JoinVertical(lipgloss.Left, rendered...)
}

// startDetection returns a command that signals detection should begin.
func (m WelcomeModel) startDetection() tea.Msg {
	return StartDetectionMsg{}
}

// Width returns the current width of the view.
func (m WelcomeModel) Width() int {
	return m.width
}

// Height returns the current height of the view.
func (m WelcomeModel) Height() int {
	return m.height
}

// Ready returns whether the view is ready to render.
func (m WelcomeModel) Ready() bool {
	return m.ready
}

// SetSize updates the view dimensions.
func (m *WelcomeModel) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.header.SetWidth(width)
	m.footer.SetWidth(width)
	m.ready = true
}

// KeyMap returns the welcome screen key bindings.
func (m WelcomeModel) KeyMap() WelcomeKeyMap {
	return m.keyMap
}

// Version returns the application version.
func (m WelcomeModel) Version() string {
	return m.version
}

// FocusedButtonIndex returns the index of the currently focused button.
func (m WelcomeModel) FocusedButtonIndex() int {
	return m.buttons.FocusedIndex()
}

// FocusButton sets focus to the button at the specified index.
func (m *WelcomeModel) FocusButton(index int) {
	m.buttons.Focus(index)
}

// IsFullHelpShown returns whether full help is currently displayed.
func (m WelcomeModel) IsFullHelpShown() bool {
	return m.footer.IsFullHelpShown()
}
