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

// UninstallConfirmKeyMap defines key bindings for the uninstall confirmation screen.
type UninstallConfirmKeyMap struct {
	Confirm key.Binding
	Back    key.Binding
	Quit    key.Binding
	Left    key.Binding
	Right   key.Binding
	Help    key.Binding
}

// DefaultUninstallConfirmKeyMap returns the default key bindings for the uninstall confirmation screen.
func DefaultUninstallConfirmKeyMap() UninstallConfirmKeyMap {
	return UninstallConfirmKeyMap{
		Confirm: key.NewBinding(
			key.WithKeys("enter", "y"),
			key.WithHelp("enter/y", "confirm"),
		),
		Back: key.NewBinding(
			key.WithKeys("esc", "n", "backspace"),
			key.WithHelp("esc/n", "go back"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
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
	}
}

// ShortHelp implements help.KeyMap interface.
// Returns key bindings for the short help view.
func (k UninstallConfirmKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Confirm, k.Back, k.Quit}
}

// FullHelp implements help.KeyMap interface.
// Returns key bindings for the full help view.
func (k UninstallConfirmKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Confirm, k.Back},
		{k.Left, k.Right},
		{k.Quit, k.Help},
	}
}

// UninstallConfirmModel represents the uninstall confirmation view.
// This screen displays what will be removed and asks for confirmation.
type UninstallConfirmModel struct {
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
	keyMap UninstallConfirmKeyMap

	// Uninstall details
	installedDriver  string   // Currently installed driver version
	packagesToRemove []string // Packages that will be removed
	configsToRemove  []string // Config files that will be removed
	restoreNouveau   bool     // Whether nouveau will be restored

	// Warnings
	warnings []string

	// App info
	version string
}

// UninstallConfirmOption is a functional option for configuring UninstallConfirmModel.
type UninstallConfirmOption func(*UninstallConfirmModel)

// WithInstalledDriver sets the installed driver version.
func WithInstalledDriver(version string) UninstallConfirmOption {
	return func(m *UninstallConfirmModel) {
		m.installedDriver = version
	}
}

// WithPackagesToRemove sets the packages that will be removed.
func WithPackagesToRemove(packages []string) UninstallConfirmOption {
	return func(m *UninstallConfirmModel) {
		m.packagesToRemove = packages
	}
}

// WithConfigsToRemove sets the config files that will be removed.
func WithConfigsToRemove(configs []string) UninstallConfirmOption {
	return func(m *UninstallConfirmModel) {
		m.configsToRemove = configs
	}
}

// WithRestoreNouveau sets whether nouveau will be restored.
func WithRestoreNouveau(restore bool) UninstallConfirmOption {
	return func(m *UninstallConfirmModel) {
		m.restoreNouveau = restore
	}
}

// WithUninstallWarnings sets the warnings to display.
func WithUninstallWarnings(warnings []string) UninstallConfirmOption {
	return func(m *UninstallConfirmModel) {
		m.warnings = warnings
	}
}

// NewUninstallConfirm creates a new uninstall confirmation view.
func NewUninstallConfirm(styles theme.Styles, version string, opts ...UninstallConfirmOption) UninstallConfirmModel {
	keyMap := DefaultUninstallConfirmKeyMap()

	header := components.NewHeader(styles, "IGOR", "Uninstall NVIDIA Drivers", version)
	footer := components.NewFooter(styles, keyMap)
	buttons := components.NewButtonGroup(styles, "Confirm Uninstall", "Cancel")

	m := UninstallConfirmModel{
		header:  header,
		footer:  footer,
		buttons: buttons,
		styles:  styles,
		keyMap:  keyMap,
		version: version,
	}

	// Apply options
	for _, opt := range opts {
		opt(&m)
	}

	return m
}

// Init implements tea.Model. Uninstall confirmation screen doesn't need initialization.
func (m UninstallConfirmModel) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model and handles messages for the uninstall confirmation screen.
func (m UninstallConfirmModel) Update(msg tea.Msg) (UninstallConfirmModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keyMap.Quit):
			return m, tea.Quit

		case key.Matches(msg, m.keyMap.Back):
			return m, m.cancelUninstall

		case key.Matches(msg, m.keyMap.Confirm):
			if m.buttons.FocusedIndex() == 0 {
				return m, m.confirmUninstall
			}
			return m, m.cancelUninstall

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
	}

	return m, nil
}

// View implements tea.Model and renders the uninstall confirmation screen.
func (m UninstallConfirmModel) View() string {
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
func (m UninstallConfirmModel) renderContent(height int) string {
	// Warning title
	title := m.styles.Warning.Render("\u26A0\uFE0F  Uninstall NVIDIA Drivers")

	// Driver info
	driverSection := m.renderDriverSection()

	// Packages section
	packagesSection := m.renderPackagesSection()

	// Configs section
	configsSection := m.renderConfigsSection()

	// Nouveau restore
	nouveauSection := m.renderNouveauSection()

	// Warnings (if any)
	warningsSection := m.renderWarningsSection()

	// Buttons
	buttonRow := lipgloss.NewStyle().
		Width(m.width).
		Align(lipgloss.Center).
		MarginTop(1).
		Render(m.buttons.View())

	sections := []string{title, ""}
	if driverSection != "" {
		sections = append(sections, driverSection)
	}
	if packagesSection != "" {
		sections = append(sections, packagesSection)
	}
	if configsSection != "" {
		sections = append(sections, configsSection)
	}
	if nouveauSection != "" {
		sections = append(sections, nouveauSection)
	}
	if warningsSection != "" {
		sections = append(sections, warningsSection)
	}
	sections = append(sections, buttonRow)

	content := lipgloss.JoinVertical(lipgloss.Left, sections...)

	return lipgloss.NewStyle().
		Width(m.width).
		Height(height).
		Padding(1, 2).
		Render(content)
}

// renderDriverSection renders the installed driver section.
func (m UninstallConfirmModel) renderDriverSection() string {
	if m.installedDriver == "" {
		return ""
	}

	subtitle := m.styles.Subtitle.Render("Driver Version:")
	driverInfo := m.styles.Info.Render(m.installedDriver)

	return subtitle + " " + driverInfo
}

// renderPackagesSection renders the packages to be removed section.
func (m UninstallConfirmModel) renderPackagesSection() string {
	if len(m.packagesToRemove) == 0 {
		return ""
	}

	subtitle := m.styles.Subtitle.Render("Packages:")

	var items []string
	for _, pkg := range m.packagesToRemove {
		marker := m.styles.Error.Render("\u2022")
		items = append(items, "  "+marker+" "+pkg)
	}

	return subtitle + "\n" + lipgloss.JoinVertical(lipgloss.Left, items...)
}

// renderConfigsSection renders the config files to be removed section.
func (m UninstallConfirmModel) renderConfigsSection() string {
	if len(m.configsToRemove) == 0 {
		return ""
	}

	subtitle := m.styles.Subtitle.Render("Configuration:")

	var items []string
	for _, cfg := range m.configsToRemove {
		marker := m.styles.Error.Render("\u2022")
		items = append(items, "  "+marker+" "+cfg)
	}

	return subtitle + "\n" + lipgloss.JoinVertical(lipgloss.Left, items...)
}

// renderNouveauSection renders the nouveau restore section.
func (m UninstallConfirmModel) renderNouveauSection() string {
	if !m.restoreNouveau {
		return ""
	}

	marker := m.styles.Success.Render("\u2611")
	return marker + " Restore nouveau driver"
}

// renderWarningsSection renders the warnings section.
func (m UninstallConfirmModel) renderWarningsSection() string {
	if len(m.warnings) == 0 {
		return ""
	}

	var items []string
	for _, w := range m.warnings {
		marker := m.styles.Warning.Render("\u26A0")
		items = append(items, marker+" "+w)
	}

	return "\n" + lipgloss.JoinVertical(lipgloss.Left, items...)
}

// Navigation command functions

// confirmUninstall returns a command that signals uninstallation should begin.
func (m UninstallConfirmModel) confirmUninstall() tea.Msg {
	return UninstallConfirmedMsg{}
}

// cancelUninstall returns a command that signals uninstallation was cancelled.
func (m UninstallConfirmModel) cancelUninstall() tea.Msg {
	return UninstallCancelledMsg{}
}

// Message types

// UninstallConfirmedMsg is sent when user confirms uninstallation.
type UninstallConfirmedMsg struct{}

// UninstallCancelledMsg is sent when user cancels uninstallation.
type UninstallCancelledMsg struct{}

// Getters

// InstalledDriver returns the installed driver version.
func (m UninstallConfirmModel) InstalledDriver() string {
	return m.installedDriver
}

// PackagesToRemove returns the packages that will be removed.
func (m UninstallConfirmModel) PackagesToRemove() []string {
	return m.packagesToRemove
}

// ConfigsToRemove returns the config files that will be removed.
func (m UninstallConfirmModel) ConfigsToRemove() []string {
	return m.configsToRemove
}

// RestoreNouveau returns whether nouveau will be restored.
func (m UninstallConfirmModel) RestoreNouveau() bool {
	return m.restoreNouveau
}

// Warnings returns the list of warnings.
func (m UninstallConfirmModel) Warnings() []string {
	return m.warnings
}

// Width returns the current width of the view.
func (m UninstallConfirmModel) Width() int {
	return m.width
}

// Height returns the current height of the view.
func (m UninstallConfirmModel) Height() int {
	return m.height
}

// Ready returns whether the view is ready to render.
func (m UninstallConfirmModel) Ready() bool {
	return m.ready
}

// Version returns the application version.
func (m UninstallConfirmModel) Version() string {
	return m.version
}

// KeyMap returns the uninstall confirmation screen key bindings.
func (m UninstallConfirmModel) KeyMap() UninstallConfirmKeyMap {
	return m.keyMap
}

// FocusedButtonIndex returns the index of the currently focused button.
func (m UninstallConfirmModel) FocusedButtonIndex() int {
	return m.buttons.FocusedIndex()
}

// IsFullHelpShown returns whether full help is currently displayed.
func (m UninstallConfirmModel) IsFullHelpShown() bool {
	return m.footer.IsFullHelpShown()
}

// SetSize updates the view dimensions.
func (m *UninstallConfirmModel) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.header.SetWidth(width)
	m.footer.SetWidth(width)
	m.ready = true
}

// SetInstalledDriver sets the installed driver version.
func (m *UninstallConfirmModel) SetInstalledDriver(version string) {
	m.installedDriver = version
}

// SetPackagesToRemove sets the packages to be removed.
func (m *UninstallConfirmModel) SetPackagesToRemove(packages []string) {
	m.packagesToRemove = packages
}

// SetConfigsToRemove sets the config files to be removed.
func (m *UninstallConfirmModel) SetConfigsToRemove(configs []string) {
	m.configsToRemove = configs
}

// SetRestoreNouveau sets whether nouveau will be restored.
func (m *UninstallConfirmModel) SetRestoreNouveau(restore bool) {
	m.restoreNouveau = restore
}

// SetWarnings sets the warnings to display.
func (m *UninstallConfirmModel) SetWarnings(warnings []string) {
	m.warnings = warnings
}
