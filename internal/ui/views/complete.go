// Package views provides the different screen views for the Igor TUI application.
// Each view represents a distinct screen in the user interface workflow.
package views

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/tungetti/igor/internal/gpu"
	"github.com/tungetti/igor/internal/ui/components"
	"github.com/tungetti/igor/internal/ui/theme"
)

// CompleteKeyMap defines key bindings for the completion screen.
type CompleteKeyMap struct {
	Reboot key.Binding
	Exit   key.Binding
	Left   key.Binding
	Right  key.Binding
	Help   key.Binding
}

// DefaultCompleteKeyMap returns the default key bindings for the completion screen.
func DefaultCompleteKeyMap() CompleteKeyMap {
	return CompleteKeyMap{
		Reboot: key.NewBinding(
			key.WithKeys("enter", "r"),
			key.WithHelp("enter/r", "reboot now"),
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
	}
}

// ShortHelp implements help.KeyMap interface.
// Returns key bindings for the short help view.
func (k CompleteKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Reboot, k.Exit}
}

// FullHelp implements help.KeyMap interface.
// Returns key bindings for the full help view.
func (k CompleteKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Reboot, k.Exit},
		{k.Left, k.Right},
		{k.Help},
	}
}

// CompleteModel represents the installation completion view.
// This screen displays a success message and options to reboot or exit.
type CompleteModel struct {
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
	keyMap CompleteKeyMap

	// Installation details
	gpuInfo    *gpu.GPUInfo
	driver     DriverOption
	components []ComponentOption

	// Reboot recommendation
	needsReboot bool

	// App info
	version string
}

// NewComplete creates a new completion view.
func NewComplete(
	styles theme.Styles,
	version string,
	gpuInfo *gpu.GPUInfo,
	driver DriverOption,
	comps []ComponentOption,
) CompleteModel {
	keyMap := DefaultCompleteKeyMap()

	header := components.NewHeader(styles, "IGOR", "Installation Complete", version)
	footer := components.NewFooter(styles, keyMap)
	buttons := components.NewButtonGroup(styles, "Reboot Now", "Exit")

	// Determine if reboot is needed
	needsReboot := false
	if gpuInfo != nil && gpuInfo.NouveauStatus != nil && gpuInfo.NouveauStatus.Loaded {
		needsReboot = true
	}

	return CompleteModel{
		header:      header,
		footer:      footer,
		buttons:     buttons,
		styles:      styles,
		keyMap:      keyMap,
		version:     version,
		gpuInfo:     gpuInfo,
		driver:      driver,
		components:  comps,
		needsReboot: needsReboot,
	}
}

// Init implements tea.Model. Completion screen doesn't need initialization.
func (m CompleteModel) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model and handles messages for the completion screen.
func (m CompleteModel) Update(msg tea.Msg) (CompleteModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keyMap.Exit):
			// Exit key pressed - emit ExitRequestedMsg
			return m, m.exit

		case key.Matches(msg, m.keyMap.Reboot):
			// Only trigger reboot if "Reboot Now" button is focused
			if m.buttons.FocusedIndex() == 0 {
				return m, m.reboot
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

// View implements tea.Model and renders the completion screen.
func (m CompleteModel) View() string {
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
func (m CompleteModel) renderContent(height int) string {
	// Success banner
	banner := m.renderSuccessBanner()

	// Installed driver info
	driverInfo := m.renderDriverInfo()

	// Installed components
	componentsSection := m.renderInstalledComponents()

	// Next steps
	nextSteps := m.renderNextSteps()

	// Buttons
	buttonRow := lipgloss.NewStyle().
		Width(m.width).
		Align(lipgloss.Center).
		MarginTop(1).
		Render(m.buttons.View())

	sections := []string{banner, "", driverInfo, componentsSection, "", nextSteps, buttonRow}

	content := lipgloss.JoinVertical(lipgloss.Left, sections...)

	return lipgloss.NewStyle().
		Width(m.width).
		Height(height).
		Padding(1, 2).
		Render(content)
}

// renderSuccessBanner renders the success banner with checkmark icon.
func (m CompleteModel) renderSuccessBanner() string {
	checkmark := m.styles.Success.Render("\u2713")
	title := m.styles.Success.Render("Installation Complete!")

	banner := checkmark + "  " + title

	return lipgloss.NewStyle().
		Width(m.width - 4).
		Align(lipgloss.Center).
		MarginBottom(1).
		Render(banner)
}

// renderDriverInfo renders the installed driver information.
func (m CompleteModel) renderDriverInfo() string {
	subtitle := m.styles.Subtitle.Render("Driver Installed:")

	version := m.driver.Version
	branch := m.driver.Branch

	driverInfo := m.styles.Info.Render(version) + " (" + branch + ")"

	return subtitle + " " + driverInfo
}

// renderInstalledComponents renders the list of installed components.
func (m CompleteModel) renderInstalledComponents() string {
	subtitle := m.styles.Subtitle.Render("Installed Components:")

	var items []string
	for _, comp := range m.components {
		marker := m.styles.Success.Render("\u2713")
		items = append(items, "  "+marker+" "+comp.Name)
	}

	if len(items) == 0 {
		return subtitle + "\n  " + m.styles.Help.Render("(none)")
	}

	return subtitle + "\n" + lipgloss.JoinVertical(lipgloss.Left, items...)
}

// renderNextSteps renders the next steps section.
func (m CompleteModel) renderNextSteps() string {
	subtitle := m.styles.Subtitle.Render("Next Steps:")

	var steps []string

	// Always show nvidia-smi verification step
	step1 := "  " + m.styles.Info.Render("\u2022") + " Run 'nvidia-smi' to verify installation"
	steps = append(steps, step1)

	// Show reboot recommendation if needed
	if m.needsReboot {
		step2 := "  " + m.styles.Warning.Render("\u2022") + " " + m.styles.Warning.Render("Reboot is recommended to complete driver loading")
		steps = append(steps, step2)
	}

	return subtitle + "\n" + lipgloss.JoinVertical(lipgloss.Left, steps...)
}

// Navigation command functions

// reboot returns a command that signals the user wants to reboot.
func (m CompleteModel) reboot() tea.Msg {
	return RebootRequestedMsg{}
}

// exit returns a command that signals the user wants to exit.
func (m CompleteModel) exit() tea.Msg {
	return ExitRequestedMsg{}
}

// Message types

// RebootRequestedMsg signals that the user wants to reboot the system.
type RebootRequestedMsg struct{}

// ExitRequestedMsg signals that the user wants to exit the application.
type ExitRequestedMsg struct{}

// Getters

// GPUInfo returns the GPU info passed to this view.
func (m CompleteModel) GPUInfo() *gpu.GPUInfo {
	return m.gpuInfo
}

// Driver returns the installed driver option.
func (m CompleteModel) Driver() DriverOption {
	return m.driver
}

// ComponentOptions returns the installed component options.
func (m CompleteModel) ComponentOptions() []ComponentOption {
	return m.components
}

// NeedsReboot returns whether a reboot is recommended.
func (m CompleteModel) NeedsReboot() bool {
	return m.needsReboot
}

// Width returns the current width of the view.
func (m CompleteModel) Width() int {
	return m.width
}

// Height returns the current height of the view.
func (m CompleteModel) Height() int {
	return m.height
}

// Ready returns whether the view is ready to render.
func (m CompleteModel) Ready() bool {
	return m.ready
}

// Version returns the application version.
func (m CompleteModel) Version() string {
	return m.version
}

// KeyMap returns the completion screen key bindings.
func (m CompleteModel) KeyMap() CompleteKeyMap {
	return m.keyMap
}

// FocusedButtonIndex returns the index of the currently focused button.
func (m CompleteModel) FocusedButtonIndex() int {
	return m.buttons.FocusedIndex()
}

// IsFullHelpShown returns whether full help is currently displayed.
func (m CompleteModel) IsFullHelpShown() bool {
	return m.footer.IsFullHelpShown()
}

// SetSize updates the view dimensions.
func (m *CompleteModel) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.header.SetWidth(width)
	m.footer.SetWidth(width)
	m.ready = true
}
