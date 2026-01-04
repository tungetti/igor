// Package components provides reusable UI components for the Igor TUI.
// These components wrap charmbracelet/bubbles with Igor-specific styling and behavior.
package components

import (
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/tungetti/igor/internal/ui/theme"
)

// SpinnerModel wraps bubbles spinner with Igor styling.
// It provides a configurable loading indicator with an optional message.
type SpinnerModel struct {
	spinner spinner.Model
	message string
	styles  theme.Styles
	visible bool
}

// NewSpinner creates a new spinner with optional message.
// The spinner uses the theme's spinner style for consistent appearance.
func NewSpinner(styles theme.Styles, message string) SpinnerModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = styles.Spinner
	return SpinnerModel{
		spinner: s,
		message: message,
		styles:  styles,
		visible: true,
	}
}

// NewSpinnerWithType creates a new spinner with a specific spinner type.
// Available types: spinner.Dot, spinner.Line, spinner.MiniDot, spinner.Jump,
// spinner.Pulse, spinner.Points, spinner.Globe, spinner.Moon, spinner.Monkey.
func NewSpinnerWithType(styles theme.Styles, message string, spinnerType spinner.Spinner) SpinnerModel {
	s := spinner.New()
	s.Spinner = spinnerType
	s.Style = styles.Spinner
	return SpinnerModel{
		spinner: s,
		message: message,
		styles:  styles,
		visible: true,
	}
}

// Init implements tea.Model and starts the spinner animation.
func (m SpinnerModel) Init() tea.Cmd {
	return m.spinner.Tick
}

// Update implements tea.Model and handles spinner tick messages.
func (m SpinnerModel) Update(msg tea.Msg) (SpinnerModel, tea.Cmd) {
	var cmd tea.Cmd
	m.spinner, cmd = m.spinner.Update(msg)
	return m, cmd
}

// View implements tea.Model and renders the spinner with its message.
func (m SpinnerModel) View() string {
	if !m.visible {
		return ""
	}
	if m.message == "" {
		return m.spinner.View()
	}
	return m.spinner.View() + " " + m.message
}

// SetMessage updates the spinner's message.
func (m *SpinnerModel) SetMessage(msg string) {
	m.message = msg
}

// Message returns the current spinner message.
func (m SpinnerModel) Message() string {
	return m.message
}

// Show makes the spinner visible.
func (m *SpinnerModel) Show() {
	m.visible = true
}

// Hide makes the spinner invisible.
func (m *SpinnerModel) Hide() {
	m.visible = false
}

// IsVisible returns whether the spinner is visible.
func (m SpinnerModel) IsVisible() bool {
	return m.visible
}

// SetSpinnerType changes the spinner animation type.
func (m *SpinnerModel) SetSpinnerType(spinnerType spinner.Spinner) {
	m.spinner.Spinner = spinnerType
}

// Tick returns a command that sends a spinner tick.
// This is useful for manually triggering spinner animation.
func (m SpinnerModel) Tick() tea.Cmd {
	return m.spinner.Tick
}
