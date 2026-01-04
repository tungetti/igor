// Package views provides the different screen views for the Igor TUI application.
// Each view represents a distinct screen in the user interface workflow.
package views

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/tungetti/igor/internal/gpu"
	"github.com/tungetti/igor/internal/ui/components"
	"github.com/tungetti/igor/internal/ui/theme"
)

// DetectionState represents the current state of detection.
type DetectionState int

const (
	// StateDetecting indicates detection is in progress.
	StateDetecting DetectionState = iota
	// StateComplete indicates detection completed successfully.
	StateComplete
	// StateError indicates detection failed with an error.
	StateError
)

// String returns the string representation of a DetectionState.
func (s DetectionState) String() string {
	switch s {
	case StateDetecting:
		return "Detecting"
	case StateComplete:
		return "Complete"
	case StateError:
		return "Error"
	default:
		return "Unknown"
	}
}

// DetectionKeyMap defines key bindings for the detection screen.
type DetectionKeyMap struct {
	Continue key.Binding
	Back     key.Binding
	Quit     key.Binding
	Help     key.Binding
}

// DefaultDetectionKeyMap returns the default key bindings for the detection screen.
func DefaultDetectionKeyMap() DetectionKeyMap {
	return DetectionKeyMap{
		Continue: key.NewBinding(
			key.WithKeys("enter", " "),
			key.WithHelp("enter", "continue"),
		),
		Back: key.NewBinding(
			key.WithKeys("esc", "backspace"),
			key.WithHelp("esc", "back"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
	}
}

// ShortHelp implements help.KeyMap interface.
// Returns key bindings for the short help view.
func (k DetectionKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Continue, k.Back, k.Quit}
}

// FullHelp implements help.KeyMap interface.
// Returns key bindings for the full help view.
func (k DetectionKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Continue, k.Back},
		{k.Quit, k.Help},
	}
}

// DetectionModel represents the system detection view.
// This screen shows detection progress with a spinner, then displays
// detected system information including GPU(s), distribution, kernel, and driver status.
type DetectionModel struct {
	// Dimensions
	width  int
	height int

	// Components
	header  components.HeaderModel
	footer  components.FooterModel
	spinner components.SpinnerModel

	// State
	state  DetectionState
	ready  bool
	styles theme.Styles
	keyMap DetectionKeyMap

	// Detection results
	gpuInfo *gpu.GPUInfo
	err     error

	// Detection progress messages
	currentStep string
	steps       []string
	stepIndex   int

	// App info
	version string
}

// NewDetection creates a new detection view model.
func NewDetection(styles theme.Styles, version string) DetectionModel {
	keyMap := DefaultDetectionKeyMap()

	header := components.NewHeader(styles, "IGOR", "System Detection", version)
	footer := components.NewFooter(styles, keyMap)
	spin := components.NewSpinner(styles, "Initializing detection...")

	steps := []string{
		"Scanning PCI devices...",
		"Identifying NVIDIA GPUs...",
		"Checking installed drivers...",
		"Detecting kernel version...",
		"Checking Nouveau status...",
		"Validating system requirements...",
	}

	return DetectionModel{
		header:      header,
		footer:      footer,
		spinner:     spin,
		state:       StateDetecting,
		styles:      styles,
		keyMap:      keyMap,
		version:     version,
		steps:       steps,
		stepIndex:   0,
		currentStep: steps[0],
	}
}

// Init implements tea.Model and starts the spinner animation.
func (m DetectionModel) Init() tea.Cmd {
	return m.spinner.Init()
}

// Update implements tea.Model and handles messages for the detection screen.
func (m DetectionModel) Update(msg tea.Msg) (DetectionModel, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch m.state {
		case StateComplete:
			switch {
			case key.Matches(msg, m.keyMap.Continue):
				return m, m.navigateToDriverSelection
			case key.Matches(msg, m.keyMap.Back):
				return m, m.navigateToWelcome
			case key.Matches(msg, m.keyMap.Quit):
				return m, tea.Quit
			case key.Matches(msg, m.keyMap.Help):
				m.footer.ToggleFullHelp()
				return m, nil
			}
		case StateError:
			switch {
			case key.Matches(msg, m.keyMap.Back):
				return m, m.navigateToWelcome
			case key.Matches(msg, m.keyMap.Quit):
				return m, tea.Quit
			case key.Matches(msg, m.keyMap.Help):
				m.footer.ToggleFullHelp()
				return m, nil
			}
		case StateDetecting:
			if key.Matches(msg, m.keyMap.Quit) {
				return m, tea.Quit
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.header.SetWidth(msg.Width)
		m.footer.SetWidth(msg.Width)
		m.ready = true
		return m, nil

	case DetectionStepMsg:
		m.stepIndex = msg.Step
		if msg.Step < len(m.steps) {
			m.currentStep = m.steps[msg.Step]
			m.spinner.SetMessage(m.currentStep)
		}
		return m, nil

	case DetectionCompleteMsg:
		m.state = StateComplete
		m.gpuInfo = msg.GPUInfo
		m.spinner.Hide()
		return m, nil

	case DetectionErrorMsg:
		m.state = StateError
		m.err = msg.Error
		m.spinner.Hide()
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// View implements tea.Model and renders the detection screen.
func (m DetectionModel) View() string {
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

	var content string
	switch m.state {
	case StateDetecting:
		content = m.renderDetecting(contentHeight)
	case StateComplete:
		content = m.renderComplete(contentHeight)
	case StateError:
		content = m.renderError(contentHeight)
	}

	return lipgloss.JoinVertical(lipgloss.Left, header, content, footer)
}

// renderDetecting renders the detection in progress screen.
func (m DetectionModel) renderDetecting(height int) string {
	spinnerView := m.spinner.View()

	content := lipgloss.JoinVertical(lipgloss.Center,
		"",
		m.styles.Title.Render("Detecting System Configuration"),
		"",
		spinnerView,
		"",
		m.styles.Help.Render("Please wait..."),
	)

	return lipgloss.NewStyle().
		Width(m.width).
		Height(height).
		Align(lipgloss.Center, lipgloss.Center).
		Render(content)
}

// renderComplete renders the detection completed screen.
func (m DetectionModel) renderComplete(height int) string {
	if m.gpuInfo == nil || len(m.gpuInfo.NVIDIAGPUs) == 0 {
		return m.renderNoGPU(height)
	}

	// Build system info display
	var sections []string

	// GPU section
	gpuSection := m.renderGPUSection()
	sections = append(sections, gpuSection)

	// Driver section
	driverSection := m.renderDriverSection()
	sections = append(sections, driverSection)

	// System section (kernel, etc.)
	systemSection := m.renderSystemSection()
	sections = append(sections, systemSection)

	// Validation summary
	if m.gpuInfo.ValidationReport != nil {
		validationSection := m.renderValidationSection()
		sections = append(sections, validationSection)
	}

	content := lipgloss.JoinVertical(lipgloss.Left, sections...)

	// Add instruction
	instruction := m.styles.Help.Render("\nPress Enter to continue to driver selection, or Esc to go back.")
	content = content + instruction

	return lipgloss.NewStyle().
		Width(m.width).
		Height(height).
		Padding(1, 2).
		Render(content)
}

// renderGPUSection renders the GPU information section.
func (m DetectionModel) renderGPUSection() string {
	title := m.styles.Title.Render("Detected NVIDIA GPUs")

	if m.gpuInfo == nil || len(m.gpuInfo.NVIDIAGPUs) == 0 {
		return title + "\n" + m.styles.Warning.Render("  No NVIDIA GPUs detected")
	}

	var gpuLines []string
	for i, g := range m.gpuInfo.NVIDIAGPUs {
		name := g.Name()
		line := fmt.Sprintf("  %d. %s", i+1, m.styles.GPUName.Render(name))
		gpuLines = append(gpuLines, line)
	}

	return title + "\n" + lipgloss.JoinVertical(lipgloss.Left, gpuLines...)
}

// renderDriverSection renders the driver status section.
func (m DetectionModel) renderDriverSection() string {
	title := m.styles.Title.Render("Driver Status")

	if m.gpuInfo == nil || m.gpuInfo.InstalledDriver == nil || !m.gpuInfo.InstalledDriver.Installed {
		return title + "\n" + m.styles.Warning.Render("  No NVIDIA driver installed")
	}

	driver := m.gpuInfo.InstalledDriver
	driverType := string(driver.Type)
	version := driver.Version

	var lines []string
	lines = append(lines, fmt.Sprintf("  Type: %s", m.styles.Info.Render(driverType)))
	lines = append(lines, fmt.Sprintf("  Version: %s", m.styles.Info.Render(version)))

	if driver.CUDAVersion != "" {
		lines = append(lines, fmt.Sprintf("  CUDA: %s", m.styles.Info.Render(driver.CUDAVersion)))
	}

	return title + "\n" + lipgloss.JoinVertical(lipgloss.Left, lines...)
}

// renderSystemSection renders the system information section.
func (m DetectionModel) renderSystemSection() string {
	title := m.styles.Title.Render("System Information")

	var lines []string

	if m.gpuInfo != nil && m.gpuInfo.KernelInfo != nil {
		kernelInfo := m.gpuInfo.KernelInfo
		lines = append(lines, fmt.Sprintf("  Kernel: %s", m.styles.Info.Render(kernelInfo.Version)))

		headersStatus := "Not installed"
		if kernelInfo.HeadersInstalled {
			headersStatus = "Installed"
		}
		lines = append(lines, fmt.Sprintf("  Headers: %s", m.styles.Info.Render(headersStatus)))

		secureBootStatus := "Disabled"
		if kernelInfo.SecureBootEnabled {
			secureBootStatus = "Enabled"
		}
		lines = append(lines, fmt.Sprintf("  Secure Boot: %s", m.styles.Info.Render(secureBootStatus)))
	}

	if m.gpuInfo != nil && m.gpuInfo.NouveauStatus != nil && m.gpuInfo.NouveauStatus.Loaded {
		lines = append(lines, fmt.Sprintf("  Nouveau: %s", m.styles.Warning.Render("Loaded (will be blacklisted)")))
	}

	if len(lines) == 0 {
		lines = append(lines, m.styles.Help.Render("  No system information available"))
	}

	return title + "\n" + lipgloss.JoinVertical(lipgloss.Left, lines...)
}

// renderValidationSection renders the validation summary section.
func (m DetectionModel) renderValidationSection() string {
	if m.gpuInfo == nil || m.gpuInfo.ValidationReport == nil {
		return ""
	}

	report := m.gpuInfo.ValidationReport
	title := m.styles.Title.Render("System Validation")

	var lines []string

	if len(report.Errors) > 0 {
		lines = append(lines, m.styles.Error.Render(fmt.Sprintf("  Errors: %d", len(report.Errors))))
	}
	if len(report.Warnings) > 0 {
		lines = append(lines, m.styles.Warning.Render(fmt.Sprintf("  Warnings: %d", len(report.Warnings))))
	}
	if len(report.Errors) == 0 && len(report.Warnings) == 0 {
		lines = append(lines, m.styles.Success.Render("  System ready for installation"))
	}

	return title + "\n" + lipgloss.JoinVertical(lipgloss.Left, lines...)
}

// renderNoGPU renders the no GPU detected screen.
func (m DetectionModel) renderNoGPU(height int) string {
	content := lipgloss.JoinVertical(lipgloss.Center,
		m.styles.Error.Render("No NVIDIA GPUs Detected"),
		"",
		m.styles.Paragraph.Render("Igor could not find any NVIDIA GPUs in your system."),
		m.styles.Paragraph.Render("Please ensure your GPU is properly installed."),
		"",
		m.styles.Help.Render("Press Esc to go back."),
	)

	return lipgloss.NewStyle().
		Width(m.width).
		Height(height).
		Align(lipgloss.Center, lipgloss.Center).
		Render(content)
}

// renderError renders the error screen.
func (m DetectionModel) renderError(height int) string {
	errMsg := "An unknown error occurred"
	if m.err != nil {
		errMsg = m.err.Error()
	}

	content := lipgloss.JoinVertical(lipgloss.Center,
		m.styles.Error.Render("Detection Failed"),
		"",
		m.styles.Paragraph.Render(errMsg),
		"",
		m.styles.Help.Render("Press Esc to go back or q to quit."),
	)

	return lipgloss.NewStyle().
		Width(m.width).
		Height(height).
		Align(lipgloss.Center, lipgloss.Center).
		Render(content)
}

// Navigation command functions

// navigateToDriverSelection returns a command that signals navigation to driver selection.
func (m DetectionModel) navigateToDriverSelection() tea.Msg {
	return NavigateToDriverSelectionMsg{GPUInfo: m.gpuInfo}
}

// navigateToWelcome returns a command that signals navigation back to welcome.
func (m DetectionModel) navigateToWelcome() tea.Msg {
	return NavigateToWelcomeMsg{}
}

// Message types

// DetectionStepMsg indicates progress through detection steps.
type DetectionStepMsg struct {
	Step int
}

// DetectionCompleteMsg signals detection completed successfully.
type DetectionCompleteMsg struct {
	GPUInfo *gpu.GPUInfo
}

// DetectionErrorMsg signals detection failed.
type DetectionErrorMsg struct {
	Error error
}

// NavigateToDriverSelectionMsg signals navigation to driver selection.
type NavigateToDriverSelectionMsg struct {
	GPUInfo *gpu.GPUInfo
}

// NavigateToWelcomeMsg signals navigation back to welcome.
type NavigateToWelcomeMsg struct{}

// Getters

// State returns the current detection state.
func (m DetectionModel) State() DetectionState {
	return m.state
}

// GPUInfo returns the detected GPU information.
func (m DetectionModel) GPUInfo() *gpu.GPUInfo {
	return m.gpuInfo
}

// Error returns any error that occurred during detection.
func (m DetectionModel) Error() error {
	return m.err
}

// Width returns the current width of the view.
func (m DetectionModel) Width() int {
	return m.width
}

// Height returns the current height of the view.
func (m DetectionModel) Height() int {
	return m.height
}

// Ready returns whether the view is ready to render.
func (m DetectionModel) Ready() bool {
	return m.ready
}

// Version returns the application version.
func (m DetectionModel) Version() string {
	return m.version
}

// KeyMap returns the detection screen key bindings.
func (m DetectionModel) KeyMap() DetectionKeyMap {
	return m.keyMap
}

// CurrentStep returns the current detection step message.
func (m DetectionModel) CurrentStep() string {
	return m.currentStep
}

// StepIndex returns the current step index.
func (m DetectionModel) StepIndex() int {
	return m.stepIndex
}

// Steps returns the list of detection steps.
func (m DetectionModel) Steps() []string {
	return m.steps
}

// IsFullHelpShown returns whether full help is currently displayed.
func (m DetectionModel) IsFullHelpShown() bool {
	return m.footer.IsFullHelpShown()
}

// Setters

// SetSize updates the view dimensions.
func (m *DetectionModel) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.header.SetWidth(width)
	m.footer.SetWidth(width)
	m.ready = true
}

// SetGPUInfo sets the detection results (called by main app after detection).
func (m *DetectionModel) SetGPUInfo(info *gpu.GPUInfo) {
	m.gpuInfo = info
	m.state = StateComplete
	m.spinner.Hide()
}

// SetError sets the error state (called by main app on detection failure).
func (m *DetectionModel) SetError(err error) {
	m.err = err
	m.state = StateError
	m.spinner.Hide()
}

// SetStep updates the current detection step.
func (m *DetectionModel) SetStep(step int) {
	m.stepIndex = step
	if step >= 0 && step < len(m.steps) {
		m.currentStep = m.steps[step]
		m.spinner.SetMessage(m.currentStep)
	}
}
