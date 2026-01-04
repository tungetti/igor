package components

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/tungetti/igor/internal/ui/theme"
)

// HeaderModel represents the application header.
// It displays the application title, subtitle, and version.
type HeaderModel struct {
	title    string
	subtitle string
	version  string
	width    int
	styles   theme.Styles
}

// NewHeader creates a new header with title, subtitle, and version.
func NewHeader(styles theme.Styles, title, subtitle, version string) HeaderModel {
	return HeaderModel{
		title:    title,
		subtitle: subtitle,
		version:  version,
		styles:   styles,
	}
}

// NewSimpleHeader creates a new header with just a title.
func NewSimpleHeader(styles theme.Styles, title string) HeaderModel {
	return HeaderModel{
		title:  title,
		styles: styles,
	}
}

// View renders the header with title, subtitle, and version.
func (m HeaderModel) View() string {
	if m.width <= 0 {
		// Return minimal header if width not set
		return m.renderMinimal()
	}

	// Logo/title on the left
	title := m.styles.Logo.Render(m.title)

	// Subtitle if present
	var subtitle string
	if m.subtitle != "" {
		subtitle = m.styles.Subtitle.Render(m.subtitle)
	}

	// Version on the right
	var version string
	if m.version != "" {
		version = m.styles.VersionValue.Render("v" + m.version)
	}

	// Build left content
	leftContent := title
	if subtitle != "" {
		leftContent += " " + subtitle
	}

	// Calculate spacing
	leftWidth := lipgloss.Width(leftContent)
	rightWidth := lipgloss.Width(version)

	// Account for header padding (2 on each side)
	availableWidth := m.width - 4
	spacerWidth := availableWidth - leftWidth - rightWidth
	if spacerWidth < 1 {
		spacerWidth = 1
	}

	spacer := lipgloss.NewStyle().Width(spacerWidth).Render("")

	content := leftContent + spacer + version

	return m.styles.Header.Copy().Width(m.width).Render(content)
}

// renderMinimal renders a minimal header without width constraints.
func (m HeaderModel) renderMinimal() string {
	title := m.styles.Logo.Render(m.title)

	var parts []string
	parts = append(parts, title)

	if m.subtitle != "" {
		parts = append(parts, m.styles.Subtitle.Render(m.subtitle))
	}

	if m.version != "" {
		parts = append(parts, m.styles.VersionValue.Render("v"+m.version))
	}

	content := title
	if m.subtitle != "" {
		content += " " + m.styles.Subtitle.Render(m.subtitle)
	}
	if m.version != "" {
		content += "  " + m.styles.VersionValue.Render("v"+m.version)
	}

	return m.styles.Header.Render(content)
}

// ViewCentered renders the header with centered content.
func (m HeaderModel) ViewCentered() string {
	title := m.styles.Logo.Render(m.title)

	var content string
	if m.subtitle != "" {
		content = title + " " + m.styles.Subtitle.Render(m.subtitle)
	} else {
		content = title
	}

	if m.version != "" {
		content += "\n" + m.styles.VersionValue.Render("v"+m.version)
	}

	return m.styles.Header.Copy().
		Width(m.width).
		Align(lipgloss.Center).
		Render(content)
}

// SetWidth updates the header's width.
func (m *HeaderModel) SetWidth(width int) {
	m.width = width
}

// Width returns the header's width.
func (m HeaderModel) Width() int {
	return m.width
}

// SetTitle updates the header's title.
func (m *HeaderModel) SetTitle(title string) {
	m.title = title
}

// Title returns the header's title.
func (m HeaderModel) Title() string {
	return m.title
}

// SetSubtitle updates the header's subtitle.
func (m *HeaderModel) SetSubtitle(subtitle string) {
	m.subtitle = subtitle
}

// Subtitle returns the header's subtitle.
func (m HeaderModel) Subtitle() string {
	return m.subtitle
}

// SetVersion updates the header's version.
func (m *HeaderModel) SetVersion(version string) {
	m.version = version
}

// Version returns the header's version.
func (m HeaderModel) Version() string {
	return m.version
}

// HasSubtitle returns true if the header has a subtitle.
func (m HeaderModel) HasSubtitle() bool {
	return m.subtitle != ""
}

// HasVersion returns true if the header has a version.
func (m HeaderModel) HasVersion() bool {
	return m.version != ""
}
