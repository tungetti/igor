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

// ConfirmationKeyMap defines key bindings for the confirmation screen.
type ConfirmationKeyMap struct {
	Confirm key.Binding
	Back    key.Binding
	Quit    key.Binding
	Left    key.Binding
	Right   key.Binding
	Help    key.Binding
}

// DefaultConfirmationKeyMap returns the default key bindings for the confirmation screen.
func DefaultConfirmationKeyMap() ConfirmationKeyMap {
	return ConfirmationKeyMap{
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
func (k ConfirmationKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Confirm, k.Back, k.Quit}
}

// FullHelp implements help.KeyMap interface.
// Returns key bindings for the full help view.
func (k ConfirmationKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Confirm, k.Back},
		{k.Left, k.Right},
		{k.Quit, k.Help},
	}
}

// ConfirmationModel represents the installation confirmation view.
// This screen displays a summary of what will be installed and asks the user
// to confirm before proceeding.
type ConfirmationModel struct {
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
	keyMap ConfirmationKeyMap

	// Installation details
	gpuInfo            *gpu.GPUInfo
	selectedDriver     DriverOption
	selectedComponents []ComponentOption

	// Warnings
	warnings []string

	// App info
	version string
}

// NewConfirmation creates a new confirmation view.
func NewConfirmation(
	styles theme.Styles,
	version string,
	gpuInfo *gpu.GPUInfo,
	driver DriverOption,
	comps []ComponentOption,
) ConfirmationModel {
	keyMap := DefaultConfirmationKeyMap()

	header := components.NewHeader(styles, "IGOR", "Confirm Installation", version)
	footer := components.NewFooter(styles, keyMap)
	buttons := components.NewButtonGroup(styles, "Install", "Go Back")

	// Build warnings based on system state
	warnings := buildWarnings(gpuInfo, driver)

	return ConfirmationModel{
		header:             header,
		footer:             footer,
		buttons:            buttons,
		styles:             styles,
		keyMap:             keyMap,
		version:            version,
		gpuInfo:            gpuInfo,
		selectedDriver:     driver,
		selectedComponents: comps,
		warnings:           warnings,
	}
}

// buildWarnings builds warnings based on the detected GPU and system state.
func buildWarnings(gpuInfo *gpu.GPUInfo, _ DriverOption) []string {
	var warnings []string

	if gpuInfo != nil {
		// Check for Nouveau
		if gpuInfo.NouveauStatus != nil && gpuInfo.NouveauStatus.Loaded {
			warnings = append(warnings, "Nouveau driver will be blacklisted and system will require reboot")
		}

		// Check for Secure Boot
		if gpuInfo.KernelInfo != nil && gpuInfo.KernelInfo.SecureBootEnabled {
			warnings = append(warnings, "Secure Boot is enabled - driver signing may be required")
		}

		// Check kernel headers
		if gpuInfo.KernelInfo != nil && !gpuInfo.KernelInfo.HeadersInstalled {
			warnings = append(warnings, "Kernel headers will be installed for DKMS")
		}

		// Check for existing driver
		if gpuInfo.InstalledDriver != nil && gpuInfo.InstalledDriver.Installed {
			warnings = append(warnings, "Existing driver will be replaced")
		}
	}

	return warnings
}

// Init implements tea.Model. Confirmation screen doesn't need initialization.
func (m ConfirmationModel) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model and handles messages for the confirmation screen.
func (m ConfirmationModel) Update(msg tea.Msg) (ConfirmationModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keyMap.Quit):
			return m, tea.Quit

		case key.Matches(msg, m.keyMap.Back):
			return m, m.navigateToSelection

		case key.Matches(msg, m.keyMap.Confirm):
			if m.buttons.FocusedIndex() == 0 {
				return m, m.startInstallation
			}
			return m, m.navigateToSelection

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

// View implements tea.Model and renders the confirmation screen.
func (m ConfirmationModel) View() string {
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
func (m ConfirmationModel) renderContent(height int) string {
	// Summary title
	title := m.styles.Title.Render("Installation Summary")

	// GPU info
	gpuSection := m.renderGPUSection()

	// Driver info
	driverSection := m.renderDriverSection()

	// Components
	componentsSection := m.renderComponentsSection()

	// Warnings (if any)
	warningsSection := m.renderWarningsSection()

	// Confirmation message
	confirmMsg := m.styles.Paragraph.Render(
		"\nAre you sure you want to proceed with the installation?",
	)

	// Buttons
	buttonRow := lipgloss.NewStyle().
		Width(m.width).
		Align(lipgloss.Center).
		MarginTop(1).
		Render(m.buttons.View())

	sections := []string{title, "", gpuSection, driverSection, componentsSection}
	if warningsSection != "" {
		sections = append(sections, warningsSection)
	}
	sections = append(sections, confirmMsg, buttonRow)

	content := lipgloss.JoinVertical(lipgloss.Left, sections...)

	return lipgloss.NewStyle().
		Width(m.width).
		Height(height).
		Padding(1, 2).
		Render(content)
}

// renderGPUSection renders the GPU information section.
func (m ConfirmationModel) renderGPUSection() string {
	subtitle := m.styles.Subtitle.Render("Target GPU:")

	gpuName := "Unknown GPU"
	if m.gpuInfo != nil && len(m.gpuInfo.NVIDIAGPUs) > 0 {
		// Use the Name() method which prioritizes lspci name > Model name > SMI name
		gpuName = m.gpuInfo.NVIDIAGPUs[0].Name()
	}

	return subtitle + " " + m.styles.GPUName.Render(gpuName)
}

// renderDriverSection renders the driver information section.
func (m ConfirmationModel) renderDriverSection() string {
	subtitle := m.styles.Subtitle.Render("Driver Version:")

	version := m.selectedDriver.Version
	branch := m.selectedDriver.Branch

	driverInfo := m.styles.Info.Render(version) + " (" + branch + ")"

	return subtitle + " " + driverInfo
}

// renderComponentsSection renders the components section.
func (m ConfirmationModel) renderComponentsSection() string {
	subtitle := m.styles.Subtitle.Render("Components to install:")

	var items []string
	for _, comp := range m.selectedComponents {
		marker := m.styles.Success.Render("\u2713")
		items = append(items, "  "+marker+" "+comp.Name)
	}

	return subtitle + "\n" + lipgloss.JoinVertical(lipgloss.Left, items...)
}

// renderWarningsSection renders the warnings section.
func (m ConfirmationModel) renderWarningsSection() string {
	if len(m.warnings) == 0 {
		return ""
	}

	subtitle := m.styles.Warning.Render("Warnings:")

	var items []string
	for _, w := range m.warnings {
		marker := m.styles.Warning.Render("\u26A0")
		items = append(items, "  "+marker+" "+w)
	}

	return subtitle + "\n" + lipgloss.JoinVertical(lipgloss.Left, items...)
}

// Navigation command functions

// startInstallation returns a command that signals installation should begin.
func (m ConfirmationModel) startInstallation() tea.Msg {
	return StartInstallationMsg{
		GPUInfo:    m.gpuInfo,
		Driver:     m.selectedDriver,
		Components: m.selectedComponents,
	}
}

// navigateToSelection returns a command that signals navigation back to selection.
func (m ConfirmationModel) navigateToSelection() tea.Msg {
	return NavigateBackToSelectionMsg{
		GPUInfo: m.gpuInfo,
	}
}

// Message types

// StartInstallationMsg signals that installation should begin.
type StartInstallationMsg struct {
	GPUInfo    *gpu.GPUInfo
	Driver     DriverOption
	Components []ComponentOption
}

// NavigateBackToSelectionMsg signals navigation back to selection.
type NavigateBackToSelectionMsg struct {
	GPUInfo *gpu.GPUInfo
}

// Getters

// GPUInfo returns the GPU info passed to this view.
func (m ConfirmationModel) GPUInfo() *gpu.GPUInfo {
	return m.gpuInfo
}

// SelectedDriver returns the selected driver option.
func (m ConfirmationModel) SelectedDriver() DriverOption {
	return m.selectedDriver
}

// SelectedComponents returns the selected component options.
func (m ConfirmationModel) SelectedComponents() []ComponentOption {
	return m.selectedComponents
}

// Warnings returns the list of warnings.
func (m ConfirmationModel) Warnings() []string {
	return m.warnings
}

// Width returns the current width of the view.
func (m ConfirmationModel) Width() int {
	return m.width
}

// Height returns the current height of the view.
func (m ConfirmationModel) Height() int {
	return m.height
}

// Ready returns whether the view is ready to render.
func (m ConfirmationModel) Ready() bool {
	return m.ready
}

// Version returns the application version.
func (m ConfirmationModel) Version() string {
	return m.version
}

// KeyMap returns the confirmation screen key bindings.
func (m ConfirmationModel) KeyMap() ConfirmationKeyMap {
	return m.keyMap
}

// FocusedButtonIndex returns the index of the currently focused button.
func (m ConfirmationModel) FocusedButtonIndex() int {
	return m.buttons.FocusedIndex()
}

// IsFullHelpShown returns whether full help is currently displayed.
func (m ConfirmationModel) IsFullHelpShown() bool {
	return m.footer.IsFullHelpShown()
}

// SetSize updates the view dimensions.
func (m *ConfirmationModel) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.header.SetWidth(width)
	m.footer.SetWidth(width)
	m.ready = true
}
