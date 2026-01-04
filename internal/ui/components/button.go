package components

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/tungetti/igor/internal/ui/theme"
)

// ButtonModel represents a clickable button.
// It supports focus, disabled states, and consistent styling.
type ButtonModel struct {
	label    string
	focused  bool
	disabled bool
	styles   theme.Styles
}

// NewButton creates a new button with the specified label.
func NewButton(styles theme.Styles, label string) ButtonModel {
	return ButtonModel{
		label:  label,
		styles: styles,
	}
}

// View renders the button with appropriate styling based on its state.
func (m ButtonModel) View() string {
	var style lipgloss.Style
	switch {
	case m.disabled:
		style = m.styles.ButtonDisabled
	case m.focused:
		style = m.styles.ButtonFocused
	default:
		style = m.styles.Button
	}
	return style.Render(m.label)
}

// Focus sets the button to focused state.
func (m *ButtonModel) Focus() {
	m.focused = true
}

// Blur removes focus from the button.
func (m *ButtonModel) Blur() {
	m.focused = false
}

// Enable enables the button.
func (m *ButtonModel) Enable() {
	m.disabled = false
}

// Disable disables the button.
func (m *ButtonModel) Disable() {
	m.disabled = true
}

// IsFocused returns whether the button is focused.
func (m ButtonModel) IsFocused() bool {
	return m.focused
}

// IsDisabled returns whether the button is disabled.
func (m ButtonModel) IsDisabled() bool {
	return m.disabled
}

// IsEnabled returns whether the button is enabled.
func (m ButtonModel) IsEnabled() bool {
	return !m.disabled
}

// Label returns the button's label.
func (m ButtonModel) Label() string {
	return m.label
}

// SetLabel updates the button's label.
func (m *ButtonModel) SetLabel(label string) {
	m.label = label
}

// Toggle toggles the button's focus state.
func (m *ButtonModel) Toggle() {
	m.focused = !m.focused
}

// ButtonGroup manages a group of buttons with navigation.
// It tracks which button is currently focused.
type ButtonGroup struct {
	buttons []ButtonModel
	focused int
	styles  theme.Styles
}

// NewButtonGroup creates a new button group from the specified labels.
// The first button is focused by default.
func NewButtonGroup(styles theme.Styles, labels ...string) ButtonGroup {
	buttons := make([]ButtonModel, len(labels))
	for i, label := range labels {
		buttons[i] = NewButton(styles, label)
	}
	if len(buttons) > 0 {
		buttons[0].Focus()
	}
	return ButtonGroup{
		buttons: buttons,
		focused: 0,
		styles:  styles,
	}
}

// NewButtonGroupFromButtons creates a button group from existing buttons.
func NewButtonGroupFromButtons(styles theme.Styles, buttons ...ButtonModel) ButtonGroup {
	// Ensure only the first button is focused
	for i := range buttons {
		buttons[i].Blur()
	}
	if len(buttons) > 0 {
		buttons[0].Focus()
	}
	return ButtonGroup{
		buttons: buttons,
		focused: 0,
		styles:  styles,
	}
}

// View renders all buttons in the group horizontally.
func (g ButtonGroup) View() string {
	var result string
	for i, btn := range g.buttons {
		if i > 0 {
			result += "  "
		}
		result += btn.View()
	}
	return result
}

// ViewVertical renders all buttons in the group vertically.
func (g ButtonGroup) ViewVertical() string {
	var result string
	for i, btn := range g.buttons {
		if i > 0 {
			result += "\n"
		}
		result += btn.View()
	}
	return result
}

// Next moves focus to the next button (wraps around).
func (g *ButtonGroup) Next() {
	if len(g.buttons) == 0 {
		return
	}
	g.buttons[g.focused].Blur()
	g.focused = (g.focused + 1) % len(g.buttons)
	g.buttons[g.focused].Focus()
}

// Previous moves focus to the previous button (wraps around).
func (g *ButtonGroup) Previous() {
	if len(g.buttons) == 0 {
		return
	}
	g.buttons[g.focused].Blur()
	g.focused--
	if g.focused < 0 {
		g.focused = len(g.buttons) - 1
	}
	g.buttons[g.focused].Focus()
}

// FocusedIndex returns the index of the currently focused button.
func (g ButtonGroup) FocusedIndex() int {
	return g.focused
}

// FocusedLabel returns the label of the currently focused button.
func (g ButtonGroup) FocusedLabel() string {
	if g.focused >= 0 && g.focused < len(g.buttons) {
		return g.buttons[g.focused].Label()
	}
	return ""
}

// FocusedButton returns a pointer to the currently focused button.
func (g *ButtonGroup) FocusedButton() *ButtonModel {
	if g.focused >= 0 && g.focused < len(g.buttons) {
		return &g.buttons[g.focused]
	}
	return nil
}

// Focus sets focus to the button at the specified index.
func (g *ButtonGroup) Focus(index int) {
	if index < 0 || index >= len(g.buttons) {
		return
	}
	g.buttons[g.focused].Blur()
	g.focused = index
	g.buttons[g.focused].Focus()
}

// Len returns the number of buttons in the group.
func (g ButtonGroup) Len() int {
	return len(g.buttons)
}

// IsEmpty returns true if the group has no buttons.
func (g ButtonGroup) IsEmpty() bool {
	return len(g.buttons) == 0
}

// Button returns the button at the specified index.
func (g ButtonGroup) Button(index int) (ButtonModel, bool) {
	if index < 0 || index >= len(g.buttons) {
		return ButtonModel{}, false
	}
	return g.buttons[index], true
}

// Buttons returns all buttons in the group.
func (g ButtonGroup) Buttons() []ButtonModel {
	result := make([]ButtonModel, len(g.buttons))
	copy(result, g.buttons)
	return result
}

// DisableAll disables all buttons in the group.
func (g *ButtonGroup) DisableAll() {
	for i := range g.buttons {
		g.buttons[i].Disable()
	}
}

// EnableAll enables all buttons in the group.
func (g *ButtonGroup) EnableAll() {
	for i := range g.buttons {
		g.buttons[i].Enable()
	}
}

// SetLabels updates the labels of all buttons.
// If there are more labels than buttons, new buttons are created.
// If there are fewer labels than buttons, extra buttons are removed.
func (g *ButtonGroup) SetLabels(labels ...string) {
	if len(labels) == 0 {
		g.buttons = nil
		g.focused = 0
		return
	}

	buttons := make([]ButtonModel, len(labels))
	for i, label := range labels {
		buttons[i] = NewButton(g.styles, label)
	}

	// Preserve focus if possible
	if g.focused >= len(buttons) {
		g.focused = len(buttons) - 1
	}
	if g.focused >= 0 && g.focused < len(buttons) {
		buttons[g.focused].Focus()
	}

	g.buttons = buttons
}
