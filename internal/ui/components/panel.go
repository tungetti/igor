package components

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/tungetti/igor/internal/ui/theme"
)

// PanelModel represents a styled container/panel.
// It provides a bordered box with optional title and focus state.
type PanelModel struct {
	title   string
	content string
	width   int
	height  int
	styles  theme.Styles
	focused bool
}

// NewPanel creates a new panel with the specified dimensions.
func NewPanel(styles theme.Styles, title string, width, height int) PanelModel {
	return PanelModel{
		title:  title,
		width:  width,
		height: height,
		styles: styles,
	}
}

// NewPanelWithContent creates a new panel with initial content.
func NewPanelWithContent(styles theme.Styles, title, content string, width, height int) PanelModel {
	return PanelModel{
		title:   title,
		content: content,
		width:   width,
		height:  height,
		styles:  styles,
	}
}

// View renders the panel with its content.
func (m PanelModel) View() string {
	style := m.styles.Panel.Copy().Width(m.width)
	if m.height > 0 {
		style = style.Height(m.height)
	}

	if m.focused {
		style = style.BorderForeground(m.styles.BorderFocused.GetBorderBottomForeground())
	}

	var content string
	if m.title != "" {
		content = m.styles.Title.Render(m.title) + "\n\n"
	}
	content += m.content

	return style.Render(content)
}

// ViewWithoutBorder renders the panel content without the border.
func (m PanelModel) ViewWithoutBorder() string {
	style := lipgloss.NewStyle().Width(m.width)
	if m.height > 0 {
		style = style.Height(m.height)
	}

	var content string
	if m.title != "" {
		content = m.styles.Title.Render(m.title) + "\n\n"
	}
	content += m.content

	return style.Render(content)
}

// SetContent updates the panel's content.
func (m *PanelModel) SetContent(content string) {
	m.content = content
}

// Content returns the panel's current content.
func (m PanelModel) Content() string {
	return m.content
}

// SetTitle updates the panel's title.
func (m *PanelModel) SetTitle(title string) {
	m.title = title
}

// Title returns the panel's title.
func (m PanelModel) Title() string {
	return m.title
}

// SetSize updates the panel's dimensions.
func (m *PanelModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// Width returns the panel's width.
func (m PanelModel) Width() int {
	return m.width
}

// Height returns the panel's height.
func (m PanelModel) Height() int {
	return m.height
}

// Focus sets the panel to focused state.
func (m *PanelModel) Focus() {
	m.focused = true
}

// Blur removes focus from the panel.
func (m *PanelModel) Blur() {
	m.focused = false
}

// IsFocused returns whether the panel is focused.
func (m PanelModel) IsFocused() bool {
	return m.focused
}

// AppendContent appends text to the panel's content.
func (m *PanelModel) AppendContent(content string) {
	if m.content == "" {
		m.content = content
	} else {
		m.content += "\n" + content
	}
}

// ClearContent clears the panel's content.
func (m *PanelModel) ClearContent() {
	m.content = ""
}

// HasContent returns true if the panel has content.
func (m PanelModel) HasContent() bool {
	return m.content != ""
}

// HasTitle returns true if the panel has a title.
func (m PanelModel) HasTitle() bool {
	return m.title != ""
}

// InnerWidth returns the usable width inside the panel (accounting for borders/padding).
func (m PanelModel) InnerWidth() int {
	// Panel style has 2 padding on each side + 1 border on each side = 6
	innerWidth := m.width - 6
	if innerWidth < 0 {
		return 0
	}
	return innerWidth
}

// InnerHeight returns the usable height inside the panel (accounting for borders/padding).
func (m PanelModel) InnerHeight() int {
	// Panel style has 1 padding on top/bottom + 1 border on top/bottom = 4
	// Plus title takes 2 lines if present (title + empty line)
	innerHeight := m.height - 4
	if m.title != "" {
		innerHeight -= 2
	}
	if innerHeight < 0 {
		return 0
	}
	return innerHeight
}
