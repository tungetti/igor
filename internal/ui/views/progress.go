// Package views provides the different screen views for the Igor TUI application.
// Each view represents a distinct screen in the user interface workflow.
package views

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/tungetti/igor/internal/gpu"
	"github.com/tungetti/igor/internal/ui/components"
	"github.com/tungetti/igor/internal/ui/theme"
)

// StepStatus represents the status of an installation step.
type StepStatus int

// Step status constants.
const (
	StepPending StepStatus = iota
	StepRunning
	StepComplete
	StepFailed
	StepSkipped
)

// String returns the string representation of the step status.
func (s StepStatus) String() string {
	switch s {
	case StepPending:
		return "Pending"
	case StepRunning:
		return "Running"
	case StepComplete:
		return "Complete"
	case StepFailed:
		return "Failed"
	case StepSkipped:
		return "Skipped"
	default:
		return "Unknown"
	}
}

// InstallationStep represents a step in the installation process.
type InstallationStep struct {
	Name        string
	Description string
	Status      StepStatus
	StartTime   time.Time
	EndTime     time.Time
	Error       error
}

// Duration returns the duration of the step if it has completed.
func (s InstallationStep) Duration() time.Duration {
	if s.EndTime.IsZero() || s.StartTime.IsZero() {
		return 0
	}
	return s.EndTime.Sub(s.StartTime)
}

// IsRunning returns true if the step is currently running.
func (s InstallationStep) IsRunning() bool {
	return s.Status == StepRunning
}

// IsDone returns true if the step has completed (success, failure, or skipped).
func (s InstallationStep) IsDone() bool {
	return s.Status == StepComplete || s.Status == StepFailed || s.Status == StepSkipped
}

// ProgressKeyMap defines key bindings during installation.
type ProgressKeyMap struct {
	Cancel key.Binding
	Quit   key.Binding
	Help   key.Binding
}

// DefaultProgressKeyMap returns the default key bindings for the progress screen.
func DefaultProgressKeyMap() ProgressKeyMap {
	return ProgressKeyMap{
		Cancel: key.NewBinding(
			key.WithKeys("ctrl+c"),
			key.WithHelp("ctrl+c", "cancel"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q"),
			key.WithHelp("q", "quit (when complete)"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
	}
}

// ShortHelp implements help.KeyMap interface.
// Returns key bindings for the short help view.
func (k ProgressKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Cancel}
}

// FullHelp implements help.KeyMap interface.
// Returns key bindings for the full help view.
func (k ProgressKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Cancel, k.Quit, k.Help},
	}
}

// ProgressModel represents the installation progress view.
// This screen displays the progress of driver installation with a progress bar,
// step indicators, and log output.
type ProgressModel struct {
	// Dimensions
	width  int
	height int

	// Components
	header   components.HeaderModel
	footer   components.FooterModel
	spinner  components.SpinnerModel
	progress components.ProgressModel

	// State
	ready  bool
	styles theme.Styles
	keyMap ProgressKeyMap

	// Installation state
	steps           []InstallationStep
	currentStep     int
	totalSteps      int
	overallProgress float64

	// Log output
	logLines    []string
	maxLogLines int

	// Status
	isComplete   bool
	hasFailed    bool
	failureError error

	// Installation details (from confirmation)
	gpuInfo    *gpu.GPUInfo
	driver     DriverOption
	components []ComponentOption

	// App info
	version string
}

// NewProgress creates a new progress view.
func NewProgress(
	styles theme.Styles,
	version string,
	gpuInfo *gpu.GPUInfo,
	driver DriverOption,
	comps []ComponentOption,
) ProgressModel {
	keyMap := DefaultProgressKeyMap()

	header := components.NewHeader(styles, "IGOR", "Installing...", version)
	footer := components.NewFooter(styles, keyMap)
	spin := components.NewSpinner(styles, "Preparing installation...")
	prog := components.NewProgress(styles, 50) // Default width, will be adjusted

	// Build installation steps
	steps := buildInstallationSteps(comps)

	return ProgressModel{
		header:      header,
		footer:      footer,
		spinner:     spin,
		progress:    prog,
		styles:      styles,
		keyMap:      keyMap,
		version:     version,
		gpuInfo:     gpuInfo,
		driver:      driver,
		components:  comps,
		steps:       steps,
		totalSteps:  len(steps),
		maxLogLines: 10,
		logLines:    make([]string, 0),
	}
}

// buildInstallationSteps builds the installation steps based on selected components.
func buildInstallationSteps(comps []ComponentOption) []InstallationStep {
	steps := []InstallationStep{
		{Name: "prepare", Description: "Preparing system"},
		{Name: "blacklist", Description: "Blacklisting Nouveau driver"},
		{Name: "update", Description: "Updating package lists"},
	}

	// Add component-specific steps
	for _, comp := range comps {
		if comp.Selected {
			steps = append(steps, InstallationStep{
				Name:        "install_" + comp.ID,
				Description: "Installing " + comp.Name,
			})
		}
	}

	steps = append(steps, InstallationStep{
		Name:        "configure",
		Description: "Configuring drivers",
	})

	steps = append(steps, InstallationStep{
		Name:        "verify",
		Description: "Verifying installation",
	})

	return steps
}

// Init implements tea.Model and starts the spinner animation.
func (m ProgressModel) Init() tea.Cmd {
	return m.spinner.Init()
}

// Update implements tea.Model and handles messages for the progress screen.
func (m ProgressModel) Update(msg tea.Msg) (ProgressModel, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keyMap.Cancel):
			if !m.isComplete && !m.hasFailed {
				return m, m.cancelInstallation
			}
		case key.Matches(msg, m.keyMap.Quit):
			if m.isComplete || m.hasFailed {
				if m.hasFailed {
					return m, m.navigateToError
				}
				return m, m.navigateToComplete
			}
		case key.Matches(msg, m.keyMap.Help):
			m.footer.ToggleFullHelp()
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.header.SetWidth(msg.Width)
		m.footer.SetWidth(msg.Width)
		m.progress.SetWidth(msg.Width - 10)
		m.ready = true

	case InstallationStepStartMsg:
		if msg.StepIndex >= 0 && msg.StepIndex < len(m.steps) {
			m.currentStep = msg.StepIndex
			m.steps[msg.StepIndex].Status = StepRunning
			m.steps[msg.StepIndex].StartTime = time.Now()
			m.spinner.SetMessage(m.steps[msg.StepIndex].Description)
		}
		m.overallProgress = float64(msg.StepIndex) / float64(m.totalSteps)

	case InstallationStepCompleteMsg:
		if msg.StepIndex >= 0 && msg.StepIndex < len(m.steps) {
			m.steps[msg.StepIndex].Status = StepComplete
			m.steps[msg.StepIndex].EndTime = time.Now()
		}
		m.overallProgress = float64(msg.StepIndex+1) / float64(m.totalSteps)

	case InstallationStepFailedMsg:
		if msg.StepIndex >= 0 && msg.StepIndex < len(m.steps) {
			m.steps[msg.StepIndex].Status = StepFailed
			m.steps[msg.StepIndex].EndTime = time.Now()
			m.steps[msg.StepIndex].Error = msg.Error
		}
		m.hasFailed = true
		m.failureError = msg.Error
		m.spinner.Hide()

	case InstallationLogMsg:
		m.addLogLine(msg.Message)

	case InstallationCompleteMsg:
		m.isComplete = true
		m.overallProgress = 1.0
		m.spinner.Hide()
		m.header.SetSubtitle("Installation Complete")

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)

	case progress.FrameMsg:
		var cmd tea.Cmd
		m.progress, cmd = m.progress.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// addLogLine adds a log line to the log output, maintaining max lines.
func (m *ProgressModel) addLogLine(line string) {
	m.logLines = append(m.logLines, line)
	if len(m.logLines) > m.maxLogLines {
		m.logLines = m.logLines[1:]
	}
}

// View implements tea.Model and renders the progress screen.
func (m ProgressModel) View() string {
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
func (m ProgressModel) renderContent(height int) string {
	// Title
	var title string
	if m.isComplete {
		title = m.styles.Success.Render("Installation Complete!")
	} else if m.hasFailed {
		title = m.styles.Error.Render("Installation Failed")
	} else {
		title = m.styles.Title.Render("Installing NVIDIA Driver " + m.driver.Version)
	}

	// Progress bar
	progressBar := m.renderProgressBar()

	// Steps list
	stepsList := m.renderStepsList()

	// Log output
	logOutput := m.renderLogOutput()

	// Current action (spinner) if still running
	var currentAction string
	if !m.isComplete && !m.hasFailed {
		currentAction = m.spinner.View()
	}

	sections := []string{title, "", progressBar, "", stepsList}
	if currentAction != "" {
		sections = append(sections, "", currentAction)
	}
	if logOutput != "" {
		sections = append(sections, "", logOutput)
	}

	// Status message at end
	if m.isComplete {
		sections = append(sections, "", m.styles.Help.Render("Press 'q' to continue"))
	} else if m.hasFailed {
		errMsg := "Unknown error"
		if m.failureError != nil {
			errMsg = m.failureError.Error()
		}
		sections = append(sections, "", m.styles.Error.Render("Error: "+errMsg))
		sections = append(sections, m.styles.Help.Render("Press 'q' to view error details"))
	}

	content := lipgloss.JoinVertical(lipgloss.Left, sections...)

	return lipgloss.NewStyle().
		Width(m.width).
		Height(height).
		Padding(1, 2).
		Render(content)
}

// renderProgressBar renders the overall progress bar.
func (m ProgressModel) renderProgressBar() string {
	percent := int(m.overallProgress * 100)
	label := fmt.Sprintf("Overall Progress: %d%%", percent)

	// Simple text-based progress bar
	barWidth := m.width - 20
	if barWidth < 10 {
		barWidth = 10
	}

	filled := int(float64(barWidth) * m.overallProgress)
	empty := barWidth - filled

	// Ensure non-negative values
	if filled < 0 {
		filled = 0
	}
	if empty < 0 {
		empty = 0
	}

	bar := m.styles.ProgressFilled.Render(strings.Repeat("\u2588", filled)) +
		m.styles.ProgressEmpty.Render(strings.Repeat("\u2591", empty))

	return label + "\n" + bar
}

// renderStepsList renders the list of installation steps with status indicators.
func (m ProgressModel) renderStepsList() string {
	var lines []string

	displayLimit := 7 // Show at most 7 steps before showing ellipsis

	for i, step := range m.steps {
		var marker string
		var nameStyle lipgloss.Style

		switch step.Status {
		case StepComplete:
			marker = m.styles.Success.Render("\u2713")
			nameStyle = m.styles.Success
		case StepRunning:
			marker = m.styles.Info.Render("\u25CF")
			nameStyle = m.styles.Info.Bold(true)
		case StepFailed:
			marker = m.styles.Error.Render("\u2717")
			nameStyle = m.styles.Error
		case StepSkipped:
			marker = m.styles.Help.Render("-")
			nameStyle = m.styles.Help
		default: // Pending
			marker = m.styles.Help.Render("\u25CB")
			nameStyle = m.styles.Help
		}

		line := fmt.Sprintf("  %s %s", marker, nameStyle.Render(step.Description))

		// Add timing for completed steps
		if step.Status == StepComplete && !step.EndTime.IsZero() {
			duration := step.EndTime.Sub(step.StartTime)
			line += m.styles.Help.Render(fmt.Sprintf(" (%.1fs)", duration.Seconds()))
		}

		lines = append(lines, line)

		// Only show limited steps to save space
		if i >= displayLimit-1 && i < len(m.steps)-1 {
			lines = append(lines, "  ...")
			break
		}
	}

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

// renderLogOutput renders the log output area.
func (m ProgressModel) renderLogOutput() string {
	if len(m.logLines) == 0 {
		return ""
	}

	title := m.styles.Subtitle.Render("Output:")

	var lines []string
	for _, line := range m.logLines {
		lines = append(lines, "  "+m.styles.Help.Render(line))
	}

	return title + "\n" + lipgloss.JoinVertical(lipgloss.Left, lines...)
}

// Navigation command functions

// cancelInstallation returns a command that signals installation was cancelled.
func (m ProgressModel) cancelInstallation() tea.Msg {
	return InstallationCancelledMsg{}
}

// navigateToComplete returns a command that signals navigation to completion screen.
func (m ProgressModel) navigateToComplete() tea.Msg {
	return NavigateToCompleteMsg{
		GPUInfo:    m.gpuInfo,
		Driver:     m.driver,
		Components: m.components,
	}
}

// navigateToError returns a command that signals navigation to error screen.
func (m ProgressModel) navigateToError() tea.Msg {
	failedStep := ""
	if m.currentStep >= 0 && m.currentStep < len(m.steps) {
		failedStep = m.steps[m.currentStep].Description
	}
	return NavigateToErrorMsg{
		Error:      m.failureError,
		FailedStep: failedStep,
	}
}

// Message types

// InstallationStepStartMsg signals a step has started.
type InstallationStepStartMsg struct {
	StepIndex int
}

// InstallationStepCompleteMsg signals a step has completed.
type InstallationStepCompleteMsg struct {
	StepIndex int
}

// InstallationStepFailedMsg signals a step has failed.
type InstallationStepFailedMsg struct {
	StepIndex int
	Error     error
}

// InstallationLogMsg adds a log line to the output.
type InstallationLogMsg struct {
	Message string
}

// InstallationCompleteMsg signals installation is complete.
type InstallationCompleteMsg struct{}

// InstallationCancelledMsg signals installation was cancelled.
type InstallationCancelledMsg struct{}

// NavigateToCompleteMsg signals navigation to completion screen.
type NavigateToCompleteMsg struct {
	GPUInfo    *gpu.GPUInfo
	Driver     DriverOption
	Components []ComponentOption
}

// NavigateToErrorMsg signals navigation to error screen.
type NavigateToErrorMsg struct {
	Error      error
	FailedStep string
}

// Getters

// IsComplete returns whether the installation is complete.
func (m ProgressModel) IsComplete() bool { return m.isComplete }

// HasFailed returns whether the installation has failed.
func (m ProgressModel) HasFailed() bool { return m.hasFailed }

// FailureError returns the error that caused the installation to fail.
func (m ProgressModel) FailureError() error { return m.failureError }

// CurrentStep returns the index of the current step.
func (m ProgressModel) CurrentStep() int { return m.currentStep }

// TotalSteps returns the total number of steps.
func (m ProgressModel) TotalSteps() int { return m.totalSteps }

// Progress returns the overall progress as a value between 0 and 1.
func (m ProgressModel) Progress() float64 { return m.overallProgress }

// Steps returns the list of installation steps.
func (m ProgressModel) Steps() []InstallationStep { return m.steps }

// LogLines returns the current log lines.
func (m ProgressModel) LogLines() []string { return m.logLines }

// Width returns the current width of the view.
func (m ProgressModel) Width() int { return m.width }

// Height returns the current height of the view.
func (m ProgressModel) Height() int { return m.height }

// Ready returns whether the view is ready to render.
func (m ProgressModel) Ready() bool { return m.ready }

// GPUInfo returns the GPU info passed to this view.
func (m ProgressModel) GPUInfo() *gpu.GPUInfo { return m.gpuInfo }

// Driver returns the selected driver option.
func (m ProgressModel) Driver() DriverOption { return m.driver }

// Components returns the selected component options.
func (m ProgressModel) ComponentOptions() []ComponentOption { return m.components }

// Version returns the application version.
func (m ProgressModel) Version() string { return m.version }

// KeyMap returns the progress screen key bindings.
func (m ProgressModel) KeyMap() ProgressKeyMap { return m.keyMap }

// MaxLogLines returns the maximum number of log lines to display.
func (m ProgressModel) MaxLogLines() int { return m.maxLogLines }

// IsFullHelpShown returns whether full help is currently displayed.
func (m ProgressModel) IsFullHelpShown() bool { return m.footer.IsFullHelpShown() }

// SetSize updates the view dimensions.
func (m *ProgressModel) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.header.SetWidth(width)
	m.footer.SetWidth(width)
	m.progress.SetWidth(width - 10)
	m.ready = true
}

// SetMaxLogLines sets the maximum number of log lines to display.
func (m *ProgressModel) SetMaxLogLines(max int) {
	if max > 0 {
		m.maxLogLines = max
	}
}

// ClearLogLines clears all log lines.
func (m *ProgressModel) ClearLogLines() {
	m.logLines = make([]string, 0)
}

// AddLogLine adds a log line (exposed for testing).
func (m *ProgressModel) AddLogLine(line string) {
	m.addLogLine(line)
}

// MarkStepComplete marks a step as complete (for testing).
func (m *ProgressModel) MarkStepComplete(index int) {
	if index >= 0 && index < len(m.steps) {
		m.steps[index].Status = StepComplete
		m.steps[index].EndTime = time.Now()
	}
}

// MarkStepFailed marks a step as failed (for testing).
func (m *ProgressModel) MarkStepFailed(index int, err error) {
	if index >= 0 && index < len(m.steps) {
		m.steps[index].Status = StepFailed
		m.steps[index].EndTime = time.Now()
		m.steps[index].Error = err
		m.hasFailed = true
		m.failureError = err
	}
}

// SetComplete marks the installation as complete (for testing).
func (m *ProgressModel) SetComplete() {
	m.isComplete = true
	m.overallProgress = 1.0
	m.spinner.Hide()
	m.header.SetSubtitle("Installation Complete")
}
