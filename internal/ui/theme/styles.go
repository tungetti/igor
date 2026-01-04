package theme

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Styles contains pre-built lipgloss styles for the TUI.
// These styles are generated from a Theme and provide consistent
// styling across all UI components.
type Styles struct {
	// App-level styles
	App    lipgloss.Style
	Header lipgloss.Style
	Footer lipgloss.Style

	// Text styles
	Title     lipgloss.Style
	Subtitle  lipgloss.Style
	Paragraph lipgloss.Style
	Help      lipgloss.Style
	Code      lipgloss.Style

	// Component styles
	Panel            lipgloss.Style
	Card             lipgloss.Style
	List             lipgloss.Style
	ListItem         lipgloss.Style
	ListItemSelected lipgloss.Style
	ListItemFocused  lipgloss.Style

	// Button styles
	Button         lipgloss.Style
	ButtonFocused  lipgloss.Style
	ButtonDisabled lipgloss.Style
	ButtonPrimary  lipgloss.Style

	// Input styles
	Input        lipgloss.Style
	InputFocused lipgloss.Style
	InputError   lipgloss.Style

	// Status styles
	Success lipgloss.Style
	Warning lipgloss.Style
	Error   lipgloss.Style
	Info    lipgloss.Style

	// Progress styles
	ProgressBar    lipgloss.Style
	ProgressFilled lipgloss.Style
	ProgressEmpty  lipgloss.Style
	ProgressText   lipgloss.Style

	// Spinner styles
	Spinner lipgloss.Style

	// Border styles
	BorderNormal  lipgloss.Style
	BorderFocused lipgloss.Style
	BorderActive  lipgloss.Style

	// GPU/System info styles
	GPUName      lipgloss.Style
	GPUInfo      lipgloss.Style
	DriverInfo   lipgloss.Style
	SystemInfo   lipgloss.Style
	VersionLabel lipgloss.Style
	VersionValue lipgloss.Style

	// Logo/branding styles
	Logo    lipgloss.Style
	TagLine lipgloss.Style

	// Dialog styles
	Dialog       lipgloss.Style
	DialogTitle  lipgloss.Style
	DialogButton lipgloss.Style

	// Table styles
	TableHeader lipgloss.Style
	TableRow    lipgloss.Style
	TableRowAlt lipgloss.Style
	TableCell   lipgloss.Style
}

// NewStyles creates a Styles instance from a Theme.
// All styles are pre-computed for efficient rendering.
func NewStyles(t *Theme) Styles {
	return Styles{
		// App-level styles
		App: lipgloss.NewStyle(),

		Header: lipgloss.NewStyle().
			Bold(true).
			Foreground(t.Primary).
			Padding(1, 2).
			BorderStyle(lipgloss.NormalBorder()).
			BorderBottom(true).
			BorderForeground(t.Border),

		Footer: lipgloss.NewStyle().
			Foreground(t.TextMuted).
			Padding(0, 2).
			BorderStyle(lipgloss.NormalBorder()).
			BorderTop(true).
			BorderForeground(t.Border),

		// Text styles
		Title: lipgloss.NewStyle().
			Bold(true).
			Foreground(t.Primary).
			MarginBottom(1),

		Subtitle: lipgloss.NewStyle().
			Foreground(t.TextMuted).
			Italic(true),

		Paragraph: lipgloss.NewStyle().
			Foreground(t.Text),

		Help: lipgloss.NewStyle().
			Foreground(t.TextSubtle).
			Italic(true),

		Code: lipgloss.NewStyle().
			Foreground(t.Info).
			Background(t.BackgroundPanel).
			Padding(0, 1),

		// Component styles
		Panel: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(t.Border).
			Padding(1, 2),

		Card: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(t.Border).
			Padding(1, 2).
			MarginBottom(1),

		List: lipgloss.NewStyle().
			Padding(0, 1),

		ListItem: lipgloss.NewStyle().
			Foreground(t.Text).
			PaddingLeft(2),

		ListItemSelected: lipgloss.NewStyle().
			Foreground(t.Primary).
			Bold(true).
			PaddingLeft(0).
			SetString("> "),

		ListItemFocused: lipgloss.NewStyle().
			Foreground(t.TextInverse).
			Background(t.Primary).
			Bold(true).
			PaddingLeft(2).
			PaddingRight(2),

		// Button styles
		Button: lipgloss.NewStyle().
			Foreground(t.Text).
			Background(t.BackgroundPanel).
			Padding(0, 3).
			MarginRight(1),

		ButtonFocused: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(t.Primary).
			Bold(true).
			Padding(0, 3).
			MarginRight(1),

		ButtonDisabled: lipgloss.NewStyle().
			Foreground(t.TextSubtle).
			Background(t.BackgroundAlt).
			Padding(0, 3).
			MarginRight(1),

		ButtonPrimary: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(t.Primary).
			Bold(true).
			Padding(0, 3).
			MarginRight(1),

		// Input styles
		Input: lipgloss.NewStyle().
			Foreground(t.Text).
			Background(t.BackgroundAlt).
			Border(lipgloss.NormalBorder()).
			BorderForeground(t.Border).
			Padding(0, 1),

		InputFocused: lipgloss.NewStyle().
			Foreground(t.Text).
			Background(t.BackgroundAlt).
			Border(lipgloss.NormalBorder()).
			BorderForeground(t.BorderFocus).
			Padding(0, 1),

		InputError: lipgloss.NewStyle().
			Foreground(t.Text).
			Background(t.BackgroundAlt).
			Border(lipgloss.NormalBorder()).
			BorderForeground(t.Error).
			Padding(0, 1),

		// Status styles
		Success: lipgloss.NewStyle().
			Foreground(t.Success).
			Bold(true),

		Warning: lipgloss.NewStyle().
			Foreground(t.Warning).
			Bold(true),

		Error: lipgloss.NewStyle().
			Foreground(t.Error).
			Bold(true),

		Info: lipgloss.NewStyle().
			Foreground(t.Info),

		// Progress styles
		ProgressBar: lipgloss.NewStyle(),

		ProgressFilled: lipgloss.NewStyle().
			Foreground(t.Progress),

		ProgressEmpty: lipgloss.NewStyle().
			Foreground(t.ProgressBg),

		ProgressText: lipgloss.NewStyle().
			Foreground(t.TextMuted),

		// Spinner styles
		Spinner: lipgloss.NewStyle().
			Foreground(t.Primary),

		// Border styles
		BorderNormal: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(t.Border),

		BorderFocused: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(t.BorderFocus),

		BorderActive: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(t.BorderActive),

		// GPU/System info styles
		GPUName: lipgloss.NewStyle().
			Bold(true).
			Foreground(t.Primary),

		GPUInfo: lipgloss.NewStyle().
			Foreground(t.Text),

		DriverInfo: lipgloss.NewStyle().
			Foreground(t.Info),

		SystemInfo: lipgloss.NewStyle().
			Foreground(t.TextMuted),

		VersionLabel: lipgloss.NewStyle().
			Foreground(t.TextMuted).
			Bold(true),

		VersionValue: lipgloss.NewStyle().
			Foreground(t.Success),

		// Logo/branding styles
		Logo: lipgloss.NewStyle().
			Bold(true).
			Foreground(t.Primary),

		TagLine: lipgloss.NewStyle().
			Foreground(t.TextMuted).
			Italic(true),

		// Dialog styles
		Dialog: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(t.Primary).
			Padding(1, 2).
			Align(lipgloss.Center),

		DialogTitle: lipgloss.NewStyle().
			Bold(true).
			Foreground(t.Primary).
			MarginBottom(1).
			Align(lipgloss.Center),

		DialogButton: lipgloss.NewStyle().
			Foreground(t.Text).
			Background(t.BackgroundPanel).
			Padding(0, 2).
			MarginTop(1),

		// Table styles
		TableHeader: lipgloss.NewStyle().
			Bold(true).
			Foreground(t.Primary).
			BorderStyle(lipgloss.NormalBorder()).
			BorderBottom(true).
			BorderForeground(t.Border).
			Padding(0, 1),

		TableRow: lipgloss.NewStyle().
			Foreground(t.Text).
			Padding(0, 1),

		TableRowAlt: lipgloss.NewStyle().
			Foreground(t.Text).
			Background(t.BackgroundAlt).
			Padding(0, 1),

		TableCell: lipgloss.NewStyle().
			Foreground(t.Text).
			Padding(0, 1),
	}
}

// Copy returns a copy of the styles that can be modified.
// Since Styles contains value types (lipgloss.Style), this creates a shallow copy
// which is sufficient because lipgloss.Style is immutable.
func (s Styles) Copy() Styles {
	return s
}

// WithWidth returns styles adjusted for a specific width.
// This is useful for responsive layouts.
func (s Styles) WithWidth(width int) Styles {
	s.App = s.App.Width(width)
	s.Header = s.Header.Width(width)
	s.Footer = s.Footer.Width(width)
	s.Panel = s.Panel.Width(width - 4)   // Account for borders and padding
	s.Card = s.Card.Width(width - 4)     // Account for borders and padding
	s.Dialog = s.Dialog.Width(width - 8) // Account for borders and margin
	return s
}

// WithHeight returns styles adjusted for a specific height.
// This is useful for responsive layouts.
func (s Styles) WithHeight(height int) Styles {
	s.App = s.App.Height(height)
	return s
}

// RenderBox renders content inside a styled box with a title.
// The content is wrapped in a Panel style with the title styled using Title style.
func (s Styles) RenderBox(title, content string, width int) string {
	titleStyle := s.Title.Copy().Width(width - 4)
	contentStyle := lipgloss.NewStyle().Width(width - 4)

	box := s.Panel.Copy().Width(width)
	innerContent := titleStyle.Render(title) + "\n" + contentStyle.Render(content)
	return box.Render(innerContent)
}

// RenderStatusLine renders a status indicator with message.
// The status parameter should be one of: "success", "warning", "error", "info".
func (s Styles) RenderStatusLine(status, message string) string {
	var statusStyle lipgloss.Style
	switch status {
	case "success":
		statusStyle = s.Success
	case "warning":
		statusStyle = s.Warning
	case "error":
		statusStyle = s.Error
	case "info":
		statusStyle = s.Info
	default:
		statusStyle = s.Info
	}

	indicator := statusStyle.Render("●")
	return indicator + " " + message
}

// RenderProgressBar renders a progress bar with the given current value, total, and width.
// Returns an empty string if total is 0.
func (s Styles) RenderProgressBar(current, total, width int) string {
	if total == 0 || width <= 0 {
		return ""
	}

	// Clamp current to valid range
	if current < 0 {
		current = 0
	}
	if current > total {
		current = total
	}

	percent := float64(current) / float64(total)
	filled := int(float64(width) * percent)
	empty := width - filled

	// Ensure we don't have negative values due to rounding
	if filled < 0 {
		filled = 0
	}
	if empty < 0 {
		empty = 0
	}

	bar := s.ProgressFilled.Render(strings.Repeat("█", filled)) +
		s.ProgressEmpty.Render(strings.Repeat("░", empty))

	return bar
}

// RenderProgressBarWithLabel renders a progress bar with a percentage label.
func (s Styles) RenderProgressBarWithLabel(current, total, width int) string {
	if total == 0 || width <= 0 {
		return ""
	}

	// Reserve space for label " 100%"
	labelWidth := 5
	barWidth := width - labelWidth
	if barWidth < 1 {
		barWidth = 1
	}

	bar := s.RenderProgressBar(current, total, barWidth)
	percent := (current * 100) / total
	label := s.ProgressText.Render(formatPercent(percent))

	return bar + label
}

// formatPercent formats a percentage value (0-100) as a string with % suffix.
func formatPercent(percent int) string {
	if percent < 10 {
		return "  " + string(rune('0'+percent)) + "%"
	} else if percent < 100 {
		return " " + string(rune('0'+percent/10)) + string(rune('0'+percent%10)) + "%"
	}
	return "100%"
}

// RenderKeyValue renders a key-value pair with styled label and value.
func (s Styles) RenderKeyValue(key, value string) string {
	keyStyle := s.VersionLabel
	valueStyle := s.VersionValue
	return keyStyle.Render(key+": ") + valueStyle.Render(value)
}

// RenderGPUCard renders a styled GPU information card.
func (s Styles) RenderGPUCard(name, driver, memory string, width int) string {
	card := s.Card.Copy().Width(width)

	content := s.GPUName.Render(name) + "\n" +
		s.RenderKeyValue("Driver", driver) + "\n" +
		s.RenderKeyValue("Memory", memory)

	return card.Render(content)
}

// RenderButton renders a button with optional focus state.
func (s Styles) RenderButton(label string, focused bool) string {
	if focused {
		return s.ButtonFocused.Render(label)
	}
	return s.Button.Render(label)
}

// RenderDialog renders a centered dialog box with title, content, and buttons.
func (s Styles) RenderDialog(title, content string, buttons []string, focusedButton int, width int) string {
	dialog := s.Dialog.Copy().Width(width)

	titleRendered := s.DialogTitle.Render(title)
	contentRendered := lipgloss.NewStyle().Width(width - 4).Render(content)

	var buttonRow strings.Builder
	for i, btn := range buttons {
		if i > 0 {
			buttonRow.WriteString("  ")
		}
		if i == focusedButton {
			buttonRow.WriteString(s.ButtonFocused.Render(btn))
		} else {
			buttonRow.WriteString(s.Button.Render(btn))
		}
	}

	buttonStyle := lipgloss.NewStyle().Width(width - 4).Align(lipgloss.Center).MarginTop(1)
	buttonsRendered := buttonStyle.Render(buttonRow.String())

	return dialog.Render(titleRendered + "\n" + contentRendered + "\n" + buttonsRendered)
}

// RenderList renders a list of items with the specified selected index.
func (s Styles) RenderList(items []string, selectedIndex int) string {
	var result strings.Builder

	for i, item := range items {
		if i == selectedIndex {
			result.WriteString(s.ListItemSelected.Render(item))
		} else {
			result.WriteString(s.ListItem.Render(item))
		}
		if i < len(items)-1 {
			result.WriteString("\n")
		}
	}

	return s.List.Render(result.String())
}
