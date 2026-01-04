package ui

import "github.com/charmbracelet/bubbles/key"

// KeyMap defines the key bindings for the application.
type KeyMap struct {
	Quit     key.Binding
	Help     key.Binding
	Up       key.Binding
	Down     key.Binding
	Left     key.Binding
	Right    key.Binding
	Enter    key.Binding
	Back     key.Binding
	Tab      key.Binding
	Space    key.Binding
	PageUp   key.Binding
	PageDown key.Binding
	Home     key.Binding
	End      key.Binding
}

// DefaultKeyMap returns the default key bindings.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		Left: key.NewBinding(
			key.WithKeys("left", "h"),
			key.WithHelp("←/h", "left"),
		),
		Right: key.NewBinding(
			key.WithKeys("right", "l"),
			key.WithHelp("→/l", "right"),
		),
		Enter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "select"),
		),
		Back: key.NewBinding(
			key.WithKeys("esc", "backspace"),
			key.WithHelp("esc", "back"),
		),
		Tab: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "next field"),
		),
		Space: key.NewBinding(
			key.WithKeys(" "),
			key.WithHelp("space", "toggle"),
		),
		PageUp: key.NewBinding(
			key.WithKeys("pgup"),
			key.WithHelp("pgup", "page up"),
		),
		PageDown: key.NewBinding(
			key.WithKeys("pgdown"),
			key.WithHelp("pgdn", "page down"),
		),
		Home: key.NewBinding(
			key.WithKeys("home"),
			key.WithHelp("home", "go to start"),
		),
		End: key.NewBinding(
			key.WithKeys("end"),
			key.WithHelp("end", "go to end"),
		),
	}
}

// ShortHelp returns key bindings for the short help view.
// This implements the help.KeyMap interface.
func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Help, k.Quit}
}

// FullHelp returns key bindings for the full help view.
// This implements the help.KeyMap interface.
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Left, k.Right},
		{k.Enter, k.Back, k.Tab, k.Space},
		{k.PageUp, k.PageDown, k.Home, k.End},
		{k.Help, k.Quit},
	}
}

// NavigationKeys returns the navigation-related key bindings.
func (k KeyMap) NavigationKeys() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Left, k.Right, k.PageUp, k.PageDown, k.Home, k.End}
}

// ActionKeys returns the action-related key bindings.
func (k KeyMap) ActionKeys() []key.Binding {
	return []key.Binding{k.Enter, k.Back, k.Tab, k.Space}
}

// SetEnabled enables or disables a set of key bindings.
func (k *KeyMap) SetEnabled(enabled bool, bindings ...*key.Binding) {
	for _, b := range bindings {
		b.SetEnabled(enabled)
	}
}

// DisableNavigation disables all navigation keys.
func (k *KeyMap) DisableNavigation() {
	k.Up.SetEnabled(false)
	k.Down.SetEnabled(false)
	k.Left.SetEnabled(false)
	k.Right.SetEnabled(false)
	k.PageUp.SetEnabled(false)
	k.PageDown.SetEnabled(false)
	k.Home.SetEnabled(false)
	k.End.SetEnabled(false)
}

// EnableNavigation enables all navigation keys.
func (k *KeyMap) EnableNavigation() {
	k.Up.SetEnabled(true)
	k.Down.SetEnabled(true)
	k.Left.SetEnabled(true)
	k.Right.SetEnabled(true)
	k.PageUp.SetEnabled(true)
	k.PageDown.SetEnabled(true)
	k.Home.SetEnabled(true)
	k.End.SetEnabled(true)
}

// DisableActions disables all action keys.
func (k *KeyMap) DisableActions() {
	k.Enter.SetEnabled(false)
	k.Back.SetEnabled(false)
	k.Tab.SetEnabled(false)
	k.Space.SetEnabled(false)
}

// EnableActions enables all action keys.
func (k *KeyMap) EnableActions() {
	k.Enter.SetEnabled(true)
	k.Back.SetEnabled(true)
	k.Tab.SetEnabled(true)
	k.Space.SetEnabled(true)
}

// DisableAll disables all key bindings except Quit.
func (k *KeyMap) DisableAll() {
	k.DisableNavigation()
	k.DisableActions()
	k.Help.SetEnabled(false)
}

// EnableAll enables all key bindings.
func (k *KeyMap) EnableAll() {
	k.EnableNavigation()
	k.EnableActions()
	k.Help.SetEnabled(true)
	k.Quit.SetEnabled(true)
}
