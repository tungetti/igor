package components

import (
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/tungetti/igor/internal/ui/theme"
)

// FooterModel represents the application footer with help and status.
// It displays contextual help and optional status messages.
type FooterModel struct {
	help       help.Model
	keyMap     help.KeyMap
	status     string
	statusType string // "info", "success", "warning", "error"
	width      int
	styles     theme.Styles
	showHelp   bool
}

// NewFooter creates a new footer with help integration.
func NewFooter(styles theme.Styles, keyMap help.KeyMap) FooterModel {
	h := help.New()
	h.ShowAll = false

	return FooterModel{
		help:     h,
		keyMap:   keyMap,
		styles:   styles,
		showHelp: true,
	}
}

// NewFooterWithoutHelp creates a new footer without help display.
func NewFooterWithoutHelp(styles theme.Styles) FooterModel {
	h := help.New()
	h.ShowAll = false

	return FooterModel{
		help:     h,
		styles:   styles,
		showHelp: false,
	}
}

// Init implements tea.Model. Footer doesn't need initialization.
func (m FooterModel) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model and handles help toggle.
func (m FooterModel) Update(msg tea.Msg) (FooterModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if key.Matches(msg, key.NewBinding(key.WithKeys("?"))) {
			m.help.ShowAll = !m.help.ShowAll
		}
	}
	return m, nil
}

// View renders the footer with status and help.
func (m FooterModel) View() string {
	var content string

	// Status line if present
	if m.status != "" {
		var statusStyle = m.styles.Info
		switch m.statusType {
		case "success":
			statusStyle = m.styles.Success
		case "warning":
			statusStyle = m.styles.Warning
		case "error":
			statusStyle = m.styles.Error
		}
		content = statusStyle.Render("● ") + m.status + "\n"
	}

	// Help text
	if m.showHelp && m.keyMap != nil {
		m.help.Width = m.width
		content += m.help.View(m.keyMap)
	}

	return m.styles.Footer.Copy().Width(m.width).Render(content)
}

// ViewStatusOnly renders only the status line without help.
func (m FooterModel) ViewStatusOnly() string {
	if m.status == "" {
		return ""
	}

	var statusStyle = m.styles.Info
	switch m.statusType {
	case "success":
		statusStyle = m.styles.Success
	case "warning":
		statusStyle = m.styles.Warning
	case "error":
		statusStyle = m.styles.Error
	}

	content := statusStyle.Render("● ") + m.status
	return m.styles.Footer.Copy().Width(m.width).Render(content)
}

// ViewHelpOnly renders only the help without status.
func (m FooterModel) ViewHelpOnly() string {
	if !m.showHelp || m.keyMap == nil {
		return ""
	}

	m.help.Width = m.width
	content := m.help.View(m.keyMap)
	return m.styles.Footer.Copy().Width(m.width).Render(content)
}

// SetStatus sets the status message with a type.
// Valid types: "info", "success", "warning", "error".
func (m *FooterModel) SetStatus(status, statusType string) {
	m.status = status
	m.statusType = statusType
}

// SetInfoStatus sets an info status message.
func (m *FooterModel) SetInfoStatus(status string) {
	m.SetStatus(status, "info")
}

// SetSuccessStatus sets a success status message.
func (m *FooterModel) SetSuccessStatus(status string) {
	m.SetStatus(status, "success")
}

// SetWarningStatus sets a warning status message.
func (m *FooterModel) SetWarningStatus(status string) {
	m.SetStatus(status, "warning")
}

// SetErrorStatus sets an error status message.
func (m *FooterModel) SetErrorStatus(status string) {
	m.SetStatus(status, "error")
}

// ClearStatus clears the status message.
func (m *FooterModel) ClearStatus() {
	m.status = ""
	m.statusType = ""
}

// Status returns the current status message.
func (m FooterModel) Status() string {
	return m.status
}

// StatusType returns the current status type.
func (m FooterModel) StatusType() string {
	return m.statusType
}

// HasStatus returns true if there is a status message.
func (m FooterModel) HasStatus() bool {
	return m.status != ""
}

// SetWidth updates the footer's width.
func (m *FooterModel) SetWidth(width int) {
	m.width = width
}

// Width returns the footer's width.
func (m FooterModel) Width() int {
	return m.width
}

// ShowHelp shows or hides the help text.
func (m *FooterModel) ShowHelp(show bool) {
	m.showHelp = show
}

// IsHelpShown returns whether help is currently shown.
func (m FooterModel) IsHelpShown() bool {
	return m.showHelp
}

// ToggleFullHelp toggles between short and full help display.
func (m *FooterModel) ToggleFullHelp() {
	m.help.ShowAll = !m.help.ShowAll
}

// IsFullHelpShown returns whether full help is shown.
func (m FooterModel) IsFullHelpShown() bool {
	return m.help.ShowAll
}

// SetKeyMap updates the key map used for help display.
func (m *FooterModel) SetKeyMap(keyMap help.KeyMap) {
	m.keyMap = keyMap
}

// SetShowAll sets whether to show all help bindings.
func (m *FooterModel) SetShowAll(showAll bool) {
	m.help.ShowAll = showAll
}
