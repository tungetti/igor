package components

import (
	"fmt"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/tungetti/igor/internal/ui/theme"
)

// ProgressModel wraps bubbles progress with Igor styling.
// It provides a progress bar with labels and percentage tracking.
type ProgressModel struct {
	progress progress.Model
	width    int
	label    string
	current  int
	total    int
	styles   theme.Styles
}

// NewProgress creates a new progress bar with the specified width.
// The progress bar uses a gradient fill based on the theme's primary color.
func NewProgress(styles theme.Styles, width int) ProgressModel {
	p := progress.New(
		progress.WithDefaultGradient(),
		progress.WithWidth(width),
		progress.WithoutPercentage(),
	)
	return ProgressModel{
		progress: p,
		width:    width,
		styles:   styles,
	}
}

// NewProgressWithGradient creates a new progress bar with custom gradient colors.
func NewProgressWithGradient(styles theme.Styles, width int, colorA, colorB string) ProgressModel {
	p := progress.New(
		progress.WithGradient(colorA, colorB),
		progress.WithWidth(width),
		progress.WithoutPercentage(),
	)
	return ProgressModel{
		progress: p,
		width:    width,
		styles:   styles,
	}
}

// NewProgressWithSolidFill creates a new progress bar with a solid fill color.
func NewProgressWithSolidFill(styles theme.Styles, width int, color string) ProgressModel {
	p := progress.New(
		progress.WithSolidFill(color),
		progress.WithWidth(width),
		progress.WithoutPercentage(),
	)
	return ProgressModel{
		progress: p,
		width:    width,
		styles:   styles,
	}
}

// Init implements tea.Model. Progress bar doesn't need initialization.
func (m ProgressModel) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model and handles progress frame messages.
func (m ProgressModel) Update(msg tea.Msg) (ProgressModel, tea.Cmd) {
	switch msg := msg.(type) {
	case progress.FrameMsg:
		progressModel, cmd := m.progress.Update(msg)
		m.progress = progressModel.(progress.Model)
		return m, cmd
	}
	return m, nil
}

// View implements tea.Model and renders the progress bar.
func (m ProgressModel) View() string {
	percent := 0.0
	if m.total > 0 {
		percent = float64(m.current) / float64(m.total)
	}

	bar := m.progress.ViewAs(percent)
	if m.label != "" {
		return m.styles.ProgressText.Render(m.label) + "\n" + bar
	}
	return bar
}

// ViewWithPercent renders the progress bar with a percentage label.
func (m ProgressModel) ViewWithPercent() string {
	percent := m.Percent()
	bar := m.progress.ViewAs(percent)
	percentStr := fmt.Sprintf(" %3.0f%%", percent*100)

	if m.label != "" {
		return m.styles.ProgressText.Render(m.label) + "\n" + bar + m.styles.ProgressText.Render(percentStr)
	}
	return bar + m.styles.ProgressText.Render(percentStr)
}

// SetProgress updates the current progress and returns a command for animation.
func (m *ProgressModel) SetProgress(current, total int) tea.Cmd {
	m.current = current
	m.total = total
	percent := 0.0
	if total > 0 {
		percent = float64(current) / float64(total)
	}
	return m.progress.SetPercent(percent)
}

// SetLabel updates the progress bar's label.
func (m *ProgressModel) SetLabel(label string) {
	m.label = label
}

// Label returns the current label.
func (m ProgressModel) Label() string {
	return m.label
}

// SetWidth updates the progress bar's width.
func (m *ProgressModel) SetWidth(width int) {
	m.width = width
	m.progress.Width = width
}

// Width returns the current width.
func (m ProgressModel) Width() int {
	return m.width
}

// Percent returns the current progress as a percentage (0.0 to 1.0).
func (m ProgressModel) Percent() float64 {
	if m.total == 0 {
		return 0
	}
	return float64(m.current) / float64(m.total)
}

// PercentInt returns the current progress as an integer percentage (0 to 100).
func (m ProgressModel) PercentInt() int {
	return int(m.Percent() * 100)
}

// Current returns the current progress value.
func (m ProgressModel) Current() int {
	return m.current
}

// Total returns the total progress value.
func (m ProgressModel) Total() int {
	return m.total
}

// IsComplete returns true if the progress is at 100%.
func (m ProgressModel) IsComplete() bool {
	return m.total > 0 && m.current >= m.total
}

// Reset resets the progress to zero.
func (m *ProgressModel) Reset() {
	m.current = 0
	m.total = 0
}

// IncrementBy increases the current progress by the specified amount.
func (m *ProgressModel) IncrementBy(amount int) tea.Cmd {
	return m.SetProgress(m.current+amount, m.total)
}

// Increment increases the current progress by 1.
func (m *ProgressModel) Increment() tea.Cmd {
	return m.IncrementBy(1)
}
