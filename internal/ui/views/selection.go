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

// DriverOption represents a driver version option.
type DriverOption struct {
	Version     string // e.g., "550", "545", "535"
	Branch      string // e.g., "Latest", "Production", "LTS"
	Description string // e.g., "Recommended for most users"
	Recommended bool
}

// ComponentOption represents an optional component.
type ComponentOption struct {
	Name        string // e.g., "CUDA Toolkit"
	ID          string // e.g., "cuda"
	Description string
	Selected    bool
	Required    bool // Some components may be required
}

// SelectionKeyMap defines key bindings for the selection screen.
type SelectionKeyMap struct {
	Up       key.Binding
	Down     key.Binding
	Left     key.Binding
	Right    key.Binding
	Select   key.Binding // Toggle component selection
	Continue key.Binding
	Back     key.Binding
	Quit     key.Binding
	Tab      key.Binding // Switch sections
	Help     key.Binding
}

// DefaultSelectionKeyMap returns the default key bindings for the selection screen.
func DefaultSelectionKeyMap() SelectionKeyMap {
	return SelectionKeyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("up/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("down/j", "down"),
		),
		Left: key.NewBinding(
			key.WithKeys("left", "h"),
			key.WithHelp("left/h", "left"),
		),
		Right: key.NewBinding(
			key.WithKeys("right", "l"),
			key.WithHelp("right/l", "right"),
		),
		Select: key.NewBinding(
			key.WithKeys(" "),
			key.WithHelp("space", "toggle"),
		),
		Continue: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "continue"),
		),
		Back: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "back"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
		Tab: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "next section"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
	}
}

// ShortHelp implements help.KeyMap interface.
// Returns key bindings for the short help view.
func (k SelectionKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Select, k.Continue, k.Back}
}

// FullHelp implements help.KeyMap interface.
// Returns key bindings for the full help view.
func (k SelectionKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Left, k.Right},
		{k.Select, k.Tab, k.Continue, k.Back},
		{k.Quit, k.Help},
	}
}

// SelectionModel represents the driver selection view.
type SelectionModel struct {
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
	keyMap SelectionKeyMap

	// Focus management
	focusedSection int // 0 = drivers, 1 = components, 2 = buttons

	// Driver options
	driverOptions  []DriverOption
	selectedDriver int

	// Component options
	componentOptions  []ComponentOption
	selectedComponent int

	// GPU info from detection
	gpuInfo *gpu.GPUInfo

	// App info
	version string
}

// NewSelection creates a new driver selection view.
func NewSelection(styles theme.Styles, version string, gpuInfo *gpu.GPUInfo) SelectionModel {
	keyMap := DefaultSelectionKeyMap()

	header := components.NewHeader(styles, "IGOR", "Driver Selection", version)
	footer := components.NewFooter(styles, keyMap)
	buttons := components.NewButtonGroup(styles, "Continue", "Back")

	// Build driver options based on GPU
	driverOptions := buildDriverOptions(gpuInfo)

	// Build component options
	componentOptions := buildComponentOptions()

	return SelectionModel{
		header:            header,
		footer:            footer,
		buttons:           buttons,
		styles:            styles,
		keyMap:            keyMap,
		version:           version,
		gpuInfo:           gpuInfo,
		driverOptions:     driverOptions,
		componentOptions:  componentOptions,
		focusedSection:    0, // Start on driver selection
		selectedDriver:    0,
		selectedComponent: 0,
	}
}

// buildDriverOptions builds driver options based on the detected GPU.
func buildDriverOptions(gpuInfo *gpu.GPUInfo) []DriverOption {
	// Build options based on detected GPU architecture
	// In a full implementation, this would check GPU architecture
	// and filter compatible driver versions
	options := []DriverOption{
		{
			Version:     "550",
			Branch:      "Latest",
			Description: "Latest features and performance improvements",
			Recommended: true,
		},
		{
			Version:     "545",
			Branch:      "Production",
			Description: "Stable production release",
			Recommended: false,
		},
		{
			Version:     "535",
			Branch:      "LTS",
			Description: "Long-term support, maximum stability",
			Recommended: false,
		},
		{
			Version:     "470",
			Branch:      "Legacy",
			Description: "For older GPUs (Kepler, Maxwell)",
			Recommended: false,
		},
	}

	return options
}

// buildComponentOptions builds the list of component options.
func buildComponentOptions() []ComponentOption {
	return []ComponentOption{
		{
			Name:        "NVIDIA Driver",
			ID:          "driver",
			Description: "Core NVIDIA graphics driver",
			Selected:    true,
			Required:    true,
		},
		{
			Name:        "CUDA Toolkit",
			ID:          "cuda",
			Description: "GPU computing platform for developers",
			Selected:    false,
			Required:    false,
		},
		{
			Name:        "cuDNN",
			ID:          "cudnn",
			Description: "Deep learning primitives library",
			Selected:    false,
			Required:    false,
		},
		{
			Name:        "NVIDIA Settings",
			ID:          "settings",
			Description: "GUI configuration tool",
			Selected:    true,
			Required:    false,
		},
	}
}

// Init implements tea.Model. Selection screen doesn't need initialization.
func (m SelectionModel) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model and handles messages for the selection screen.
func (m SelectionModel) Update(msg tea.Msg) (SelectionModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keyMap.Quit):
			return m, tea.Quit

		case key.Matches(msg, m.keyMap.Back):
			return m, m.navigateToDetection

		case key.Matches(msg, m.keyMap.Continue):
			if m.focusedSection == 2 { // On buttons
				if m.buttons.FocusedIndex() == 0 {
					return m, m.navigateToConfirmation
				}
				return m, m.navigateToDetection
			}
			// Move to next section or continue
			m.focusedSection++
			if m.focusedSection > 2 {
				return m, m.navigateToConfirmation
			}
			return m, nil

		case key.Matches(msg, m.keyMap.Tab):
			m.focusedSection = (m.focusedSection + 1) % 3
			return m, nil

		case key.Matches(msg, m.keyMap.Up):
			m = m.moveUp()
			return m, nil

		case key.Matches(msg, m.keyMap.Down):
			m = m.moveDown()
			return m, nil

		case key.Matches(msg, m.keyMap.Left):
			if m.focusedSection == 2 {
				m.buttons.Previous()
			}
			return m, nil

		case key.Matches(msg, m.keyMap.Right):
			if m.focusedSection == 2 {
				m.buttons.Next()
			}
			return m, nil

		case key.Matches(msg, m.keyMap.Select):
			if m.focusedSection == 1 { // Components section
				m = m.toggleComponent()
			}
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

// moveUp moves the selection up within the current section.
func (m SelectionModel) moveUp() SelectionModel {
	switch m.focusedSection {
	case 0: // Drivers
		if m.selectedDriver > 0 {
			m.selectedDriver--
		}
	case 1: // Components
		if m.selectedComponent > 0 {
			m.selectedComponent--
		}
	}
	return m
}

// moveDown moves the selection down within the current section.
func (m SelectionModel) moveDown() SelectionModel {
	switch m.focusedSection {
	case 0: // Drivers
		if m.selectedDriver < len(m.driverOptions)-1 {
			m.selectedDriver++
		}
	case 1: // Components
		if m.selectedComponent < len(m.componentOptions)-1 {
			m.selectedComponent++
		}
	}
	return m
}

// toggleComponent toggles the selected state of the current component.
func (m SelectionModel) toggleComponent() SelectionModel {
	if m.selectedComponent >= 0 && m.selectedComponent < len(m.componentOptions) {
		comp := &m.componentOptions[m.selectedComponent]
		if !comp.Required { // Can't toggle required components
			comp.Selected = !comp.Selected
		}
	}
	return m
}

// View implements tea.Model and renders the selection screen.
func (m SelectionModel) View() string {
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
func (m SelectionModel) renderContent(height int) string {
	// Two-column layout: drivers on left, components on right
	halfWidth := (m.width - 6) / 2
	if halfWidth < 20 {
		halfWidth = 20
	}

	driversPanel := m.renderDriversPanel(halfWidth)
	componentsPanel := m.renderComponentsPanel(halfWidth)

	// Join panels horizontally
	panels := lipgloss.JoinHorizontal(lipgloss.Top, driversPanel, "  ", componentsPanel)

	// Buttons at bottom
	buttonRow := lipgloss.NewStyle().
		Width(m.width).
		Align(lipgloss.Center).
		MarginTop(1).
		Render(m.buttons.View())

	// Highlight buttons section if focused
	if m.focusedSection == 2 {
		buttonRow = lipgloss.NewStyle().
			Width(m.width).
			Align(lipgloss.Center).
			MarginTop(1).
			Render(m.buttons.View())
	}

	content := lipgloss.JoinVertical(lipgloss.Left, panels, buttonRow)

	return lipgloss.NewStyle().
		Width(m.width).
		Height(height).
		Padding(1, 2).
		Render(content)
}

// renderDriversPanel renders the driver selection panel.
func (m SelectionModel) renderDriversPanel(width int) string {
	isFocused := m.focusedSection == 0

	title := m.styles.Title.Render("Driver Version")

	var items []string
	for i, opt := range m.driverOptions {
		isSelected := i == m.selectedDriver
		item := m.renderDriverOption(opt, isSelected, isFocused)
		items = append(items, item)
	}

	content := title + "\n\n" + lipgloss.JoinVertical(lipgloss.Left, items...)

	panelStyle := m.styles.Panel.Copy().Width(width)
	if isFocused {
		panelStyle = panelStyle.BorderForeground(theme.NVIDIAGreen)
	}

	return panelStyle.Render(content)
}

// renderDriverOption renders a single driver option.
func (m SelectionModel) renderDriverOption(opt DriverOption, isSelected, isFocused bool) string {
	cursor := "  "
	if isSelected && isFocused {
		cursor = m.styles.Success.Render("> ")
	} else if isSelected {
		cursor = "* "
	}

	version := opt.Version
	if opt.Recommended {
		version += " " + m.styles.Success.Render("(Recommended)")
	}

	versionStyle := m.styles.GPUName
	if isSelected {
		versionStyle = versionStyle.Copy().Bold(true)
	}

	line := cursor + versionStyle.Render(version) + " - " + opt.Branch
	desc := "     " + m.styles.Help.Render(opt.Description)

	return line + "\n" + desc
}

// renderComponentsPanel renders the component selection panel.
func (m SelectionModel) renderComponentsPanel(width int) string {
	isFocused := m.focusedSection == 1

	title := m.styles.Title.Render("Components")

	var items []string
	for i, comp := range m.componentOptions {
		isSelected := i == m.selectedComponent
		item := m.renderComponentOption(comp, isSelected, isFocused)
		items = append(items, item)
	}

	content := title + "\n\n" + lipgloss.JoinVertical(lipgloss.Left, items...)

	panelStyle := m.styles.Panel.Copy().Width(width)
	if isFocused {
		panelStyle = panelStyle.BorderForeground(theme.NVIDIAGreen)
	}

	return panelStyle.Render(content)
}

// renderComponentOption renders a single component option.
func (m SelectionModel) renderComponentOption(comp ComponentOption, isSelected, isFocused bool) string {
	cursor := "  "
	if isSelected && isFocused {
		cursor = m.styles.Success.Render("> ")
	}

	checkbox := "[ ]"
	if comp.Selected {
		checkbox = m.styles.Success.Render("[x]")
	}
	if comp.Required {
		checkbox = m.styles.Info.Render("[*]") // Required, always selected
	}

	nameStyle := m.styles.Info
	if isSelected {
		nameStyle = nameStyle.Copy().Bold(true)
	}

	line := cursor + checkbox + " " + nameStyle.Render(comp.Name)
	desc := "       " + m.styles.Help.Render(comp.Description)

	return line + "\n" + desc
}

// Navigation command functions

// navigateToConfirmation returns a command that signals navigation to confirmation.
func (m SelectionModel) navigateToConfirmation() tea.Msg {
	return NavigateToConfirmationMsg{
		GPUInfo:            m.gpuInfo,
		SelectedDriver:     m.driverOptions[m.selectedDriver],
		SelectedComponents: m.getSelectedComponents(),
	}
}

// navigateToDetection returns a command that signals navigation back to detection.
func (m SelectionModel) navigateToDetection() tea.Msg {
	return NavigateToDetectionMsg{}
}

// getSelectedComponents returns all selected component options.
func (m SelectionModel) getSelectedComponents() []ComponentOption {
	var selected []ComponentOption
	for _, comp := range m.componentOptions {
		if comp.Selected {
			selected = append(selected, comp)
		}
	}
	return selected
}

// Message types

// NavigateToConfirmationMsg signals navigation to confirmation screen.
type NavigateToConfirmationMsg struct {
	GPUInfo            *gpu.GPUInfo
	SelectedDriver     DriverOption
	SelectedComponents []ComponentOption
}

// NavigateToDetectionMsg signals navigation back to detection.
type NavigateToDetectionMsg struct{}

// Getters

// SelectedDriverOption returns the currently selected driver option.
func (m SelectionModel) SelectedDriverOption() DriverOption {
	if m.selectedDriver >= 0 && m.selectedDriver < len(m.driverOptions) {
		return m.driverOptions[m.selectedDriver]
	}
	return DriverOption{}
}

// SelectedComponents returns all selected component options.
func (m SelectionModel) SelectedComponents() []ComponentOption {
	return m.getSelectedComponents()
}

// FocusedSection returns the currently focused section index.
func (m SelectionModel) FocusedSection() int {
	return m.focusedSection
}

// Width returns the current width of the view.
func (m SelectionModel) Width() int {
	return m.width
}

// Height returns the current height of the view.
func (m SelectionModel) Height() int {
	return m.height
}

// Ready returns whether the view is ready to render.
func (m SelectionModel) Ready() bool {
	return m.ready
}

// GPUInfo returns the GPU info passed to this view.
func (m SelectionModel) GPUInfo() *gpu.GPUInfo {
	return m.gpuInfo
}

// DriverOptions returns the available driver options.
func (m SelectionModel) DriverOptions() []DriverOption {
	return m.driverOptions
}

// ComponentOptions returns the available component options.
func (m SelectionModel) ComponentOptions() []ComponentOption {
	return m.componentOptions
}

// SelectedDriverIndex returns the index of the selected driver.
func (m SelectionModel) SelectedDriverIndex() int {
	return m.selectedDriver
}

// SelectedComponentIndex returns the index of the selected component.
func (m SelectionModel) SelectedComponentIndex() int {
	return m.selectedComponent
}

// KeyMap returns the selection screen key bindings.
func (m SelectionModel) KeyMap() SelectionKeyMap {
	return m.keyMap
}

// Version returns the application version.
func (m SelectionModel) Version() string {
	return m.version
}

// IsFullHelpShown returns whether full help is currently displayed.
func (m SelectionModel) IsFullHelpShown() bool {
	return m.footer.IsFullHelpShown()
}

// FocusedButtonIndex returns the index of the currently focused button.
func (m SelectionModel) FocusedButtonIndex() int {
	return m.buttons.FocusedIndex()
}

// SetSize updates the view dimensions.
func (m *SelectionModel) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.header.SetWidth(width)
	m.footer.SetWidth(width)
	m.ready = true
}

// SetFocusedSection sets the focused section.
func (m *SelectionModel) SetFocusedSection(section int) {
	if section >= 0 && section <= 2 {
		m.focusedSection = section
	}
}
