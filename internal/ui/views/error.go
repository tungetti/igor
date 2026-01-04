// Package views provides the different screen views for the Igor TUI application.
// Each view represents a distinct screen in the user interface workflow.
package views

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/tungetti/igor/internal/ui/components"
	"github.com/tungetti/igor/internal/ui/theme"
)

// ErrorKeyMap defines key bindings for the error/help screen.
type ErrorKeyMap struct {
	Retry key.Binding
	Exit  key.Binding
	Left  key.Binding
	Right key.Binding
	Help  key.Binding
	Copy  key.Binding
}

// DefaultErrorKeyMap returns the default key bindings for the error screen.
func DefaultErrorKeyMap() ErrorKeyMap {
	return ErrorKeyMap{
		Retry: key.NewBinding(
			key.WithKeys("enter", "r"),
			key.WithHelp("enter/r", "retry installation"),
		),
		Exit: key.NewBinding(
			key.WithKeys("q", "esc"),
			key.WithHelp("q/esc", "exit"),
		),
		Left: key.NewBinding(
			key.WithKeys("left", "h"),
			key.WithHelp("\u2190/h", "previous"),
		),
		Right: key.NewBinding(
			key.WithKeys("right", "l", "tab"),
			key.WithHelp("\u2192/l", "next"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
		Copy: key.NewBinding(
			key.WithKeys("c"),
			key.WithHelp("c", "copy error to clipboard"),
		),
	}
}

// ShortHelp implements help.KeyMap interface.
// Returns key bindings for the short help view.
func (k ErrorKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Retry, k.Exit}
}

// FullHelp implements help.KeyMap interface.
// Returns key bindings for the full help view.
func (k ErrorKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Retry, k.Exit},
		{k.Left, k.Right},
		{k.Help, k.Copy},
	}
}

// ErrorModel represents the error/help view.
// This screen is displayed when installation fails and provides
// troubleshooting information and options to retry or exit.
type ErrorModel struct {
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
	keyMap ErrorKeyMap

	// Error details
	err        error
	failedStep string

	// Help content
	troubleshootingTips []string

	// App info
	version string
}

// NewError creates a new error view.
func NewError(styles theme.Styles, version string, err error, failedStep string) ErrorModel {
	keyMap := DefaultErrorKeyMap()

	header := components.NewHeader(styles, "IGOR", "Installation Failed", version)
	footer := components.NewFooter(styles, keyMap)
	buttons := components.NewButtonGroup(styles, "Retry", "Exit")

	// Build troubleshooting tips based on failed step
	tips := buildTroubleshootingTips(failedStep)

	return ErrorModel{
		header:              header,
		footer:              footer,
		buttons:             buttons,
		styles:              styles,
		keyMap:              keyMap,
		version:             version,
		err:                 err,
		failedStep:          failedStep,
		troubleshootingTips: tips,
	}
}

// buildTroubleshootingTips generates troubleshooting tips based on the failed step.
func buildTroubleshootingTips(failedStep string) []string {
	var tips []string

	// Normalize the failed step for matching
	step := strings.ToLower(failedStep)

	switch {
	case strings.Contains(step, "blacklist"):
		tips = append(tips, "Check if Nouveau driver can be unloaded")
		tips = append(tips, "Try rebooting and running again")

	case strings.Contains(step, "update"):
		tips = append(tips, "Check network connectivity")
		tips = append(tips, "Verify repository access")
		tips = append(tips, "Try running 'sudo apt update' manually")

	case strings.HasPrefix(step, "install") || strings.Contains(step, "install_"):
		tips = append(tips, "Check disk space")
		tips = append(tips, "Verify package availability")
		tips = append(tips, "Check for package conflicts")

	case strings.Contains(step, "configure"):
		tips = append(tips, "Check system permissions")
		tips = append(tips, "Verify configuration files are writable")

	case strings.Contains(step, "verify"):
		tips = append(tips, "Check driver loaded correctly")
		tips = append(tips, "Run 'dmesg | grep nvidia' for kernel messages")
		tips = append(tips, "Check if nvidia-smi returns expected output")

	default:
		// Default tips for unknown steps
		tips = append(tips, "Check system logs for more details")
		tips = append(tips, "Run with --verbose for more details")
	}

	return tips
}

// Init implements tea.Model. Error screen doesn't need initialization.
func (m ErrorModel) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model and handles messages for the error screen.
func (m ErrorModel) Update(msg tea.Msg) (ErrorModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keyMap.Exit):
			// Exit key pressed - emit ErrorExitRequestedMsg
			return m, m.exit

		case key.Matches(msg, m.keyMap.Retry):
			// Only trigger retry if "Retry" button is focused
			if m.buttons.FocusedIndex() == 0 {
				return m, m.retry
			}
			// If Exit button is focused and enter is pressed, exit
			return m, m.exit

		case key.Matches(msg, m.keyMap.Left):
			m.buttons.Previous()
			return m, nil

		case key.Matches(msg, m.keyMap.Right):
			m.buttons.Next()
			return m, nil

		case key.Matches(msg, m.keyMap.Help):
			m.footer.ToggleFullHelp()
			return m, nil

		case key.Matches(msg, m.keyMap.Copy):
			// TODO(P5): Implement clipboard copy functionality
			// Consider using github.com/atotto/clipboard
			return m, nil
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.header.SetWidth(msg.Width)
		m.footer.SetWidth(msg.Width)
		m.ready = true
	}

	return m, nil
}

// View implements tea.Model and renders the error screen.
func (m ErrorModel) View() string {
	if !m.ready {
		return "Loading..."
	}

	header := m.header.View()
	footer := m.footer.View()

	headerHeight := lipgloss.Height(header)
	footerHeight := lipgloss.Height(footer)
	contentHeight := m.height - headerHeight - footerHeight

	// Ensure minimum content height
	if contentHeight < 1 {
		contentHeight = 1
	}

	content := m.renderContent(contentHeight)

	return lipgloss.JoinVertical(lipgloss.Left, header, content, footer)
}

// renderContent renders the main content area.
func (m ErrorModel) renderContent(height int) string {
	// Error banner
	banner := m.renderErrorBanner()

	// Failed step (if provided)
	failedStepSection := m.renderFailedStep()

	// Error details
	errorDetails := m.renderErrorDetails()

	// Troubleshooting tips
	tipsSection := m.renderTroubleshootingTips()

	// Buttons
	buttonRow := lipgloss.NewStyle().
		Width(m.width).
		Align(lipgloss.Center).
		MarginTop(1).
		Render(m.buttons.View())

	sections := []string{banner, ""}
	if failedStepSection != "" {
		sections = append(sections, failedStepSection)
	}
	sections = append(sections, errorDetails, "", tipsSection, buttonRow)

	content := lipgloss.JoinVertical(lipgloss.Left, sections...)

	return lipgloss.NewStyle().
		Width(m.width).
		Height(height).
		Padding(1, 2).
		Render(content)
}

// renderErrorBanner renders the error banner with X icon.
func (m ErrorModel) renderErrorBanner() string {
	xIcon := m.styles.Error.Render("\u2717")
	title := m.styles.Error.Render("Installation Failed")

	banner := xIcon + "  " + title

	return lipgloss.NewStyle().
		Width(m.width - 4).
		Align(lipgloss.Center).
		MarginBottom(1).
		Render(banner)
}

// renderFailedStep renders the failed step name if provided.
func (m ErrorModel) renderFailedStep() string {
	if m.failedStep == "" {
		return ""
	}

	subtitle := m.styles.Subtitle.Render("Failed Step:")
	stepName := m.styles.Error.Render(m.failedStep)

	return subtitle + " " + stepName
}

// renderErrorDetails renders the error message details.
func (m ErrorModel) renderErrorDetails() string {
	subtitle := m.styles.Subtitle.Render("Error Details:")

	var errorMsg string
	if m.err != nil {
		errorMsg = m.err.Error()
	} else {
		errorMsg = "Unknown error occurred"
	}

	// Style the error message
	errorText := m.styles.Paragraph.Render(errorMsg)

	return subtitle + "\n  " + errorText
}

// renderTroubleshootingTips renders the troubleshooting tips section.
func (m ErrorModel) renderTroubleshootingTips() string {
	subtitle := m.styles.Subtitle.Render("Troubleshooting Tips:")

	var items []string
	for _, tip := range m.troubleshootingTips {
		marker := m.styles.Info.Render("\u2022")
		items = append(items, "  "+marker+" "+tip)
	}

	if len(items) == 0 {
		return subtitle + "\n  " + m.styles.Help.Render("(no tips available)")
	}

	return subtitle + "\n" + lipgloss.JoinVertical(lipgloss.Left, items...)
}

// Navigation command functions

// retry returns a command that signals the user wants to retry installation.
func (m ErrorModel) retry() tea.Msg {
	return RetryRequestedMsg{}
}

// exit returns a command that signals the user wants to exit from error screen.
func (m ErrorModel) exit() tea.Msg {
	return ErrorExitRequestedMsg{}
}

// Message types

// RetryRequestedMsg signals that the user wants to retry installation.
type RetryRequestedMsg struct{}

// ErrorExitRequestedMsg signals that the user wants to exit from error screen.
type ErrorExitRequestedMsg struct{}

// Getters

// Error returns the error that caused the installation to fail.
func (m ErrorModel) Error() error {
	return m.err
}

// FailedStep returns the name of the step that failed.
func (m ErrorModel) FailedStep() string {
	return m.failedStep
}

// TroubleshootingTips returns the list of troubleshooting tips.
func (m ErrorModel) TroubleshootingTips() []string {
	return m.troubleshootingTips
}

// Width returns the current width of the view.
func (m ErrorModel) Width() int {
	return m.width
}

// Height returns the current height of the view.
func (m ErrorModel) Height() int {
	return m.height
}

// Ready returns whether the view is ready to render.
func (m ErrorModel) Ready() bool {
	return m.ready
}

// Version returns the application version.
func (m ErrorModel) Version() string {
	return m.version
}

// KeyMap returns the error screen key bindings.
func (m ErrorModel) KeyMap() ErrorKeyMap {
	return m.keyMap
}

// FocusedButtonIndex returns the index of the currently focused button.
func (m ErrorModel) FocusedButtonIndex() int {
	return m.buttons.FocusedIndex()
}

// IsFullHelpShown returns whether full help is currently displayed.
func (m ErrorModel) IsFullHelpShown() bool {
	return m.footer.IsFullHelpShown()
}

// SetSize updates the view dimensions.
func (m *ErrorModel) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.header.SetWidth(width)
	m.footer.SetWidth(width)
	m.ready = true
}
