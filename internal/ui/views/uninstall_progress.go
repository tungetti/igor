// Package views provides the different screen views for the Igor TUI application.
// Each view represents a distinct screen in the user interface workflow.
package views

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/tungetti/igor/internal/ui/components"
	"github.com/tungetti/igor/internal/ui/theme"
)

// UninstallProgressKeyMap defines key bindings during uninstallation.
type UninstallProgressKeyMap struct {
	Cancel key.Binding
	Quit   key.Binding
	Help   key.Binding
}

// DefaultUninstallProgressKeyMap returns the default key bindings for the uninstall progress screen.
func DefaultUninstallProgressKeyMap() UninstallProgressKeyMap {
	return UninstallProgressKeyMap{
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
func (k UninstallProgressKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Cancel}
}

// FullHelp implements help.KeyMap interface.
// Returns key bindings for the full help view.
func (k UninstallProgressKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Cancel, k.Quit, k.Help},
	}
}

// UninstallStep represents a step in the uninstall process.
type UninstallStep struct {
	Name        string
	Description string
	Status      StepStatus // Reuse StepStatus from progress.go
	StartTime   time.Time
	EndTime     time.Time
	Error       error
}

// Duration returns the duration of the step if it has completed.
func (s UninstallStep) Duration() time.Duration {
	if s.EndTime.IsZero() || s.StartTime.IsZero() {
		return 0
	}
	return s.EndTime.Sub(s.StartTime)
}

// IsRunning returns true if the step is currently running.
func (s UninstallStep) IsRunning() bool {
	return s.Status == StepRunning
}

// IsDone returns true if the step has completed (success, failure, or skipped).
func (s UninstallStep) IsDone() bool {
	return s.Status == StepComplete || s.Status == StepFailed || s.Status == StepSkipped
}

// UninstallProgressModel represents the uninstall progress view.
type UninstallProgressModel struct {
	// Dimensions
	width  int
	height int

	// Components
	header  components.HeaderModel
	footer  components.FooterModel
	spinner components.SpinnerModel

	// State
	ready     bool
	styles    theme.Styles
	keyMap    UninstallProgressKeyMap
	startTime time.Time

	// Uninstall steps
	steps           []UninstallStep
	currentStep     int
	totalSteps      int
	overallProgress float64

	// Status
	completed bool
	failed    bool
	cancelled bool

	// Results
	removedPackages []string
	cleanedConfigs  []string
	nouveauRestored bool
	needsReboot     bool

	// Log output
	logLines    []string
	maxLogLines int

	// Failure info
	failureError error

	// App info
	version string
}

// UninstallProgressOption is a functional option for configuring UninstallProgressModel.
type UninstallProgressOption func(*UninstallProgressModel)

// WithUninstallSteps sets the uninstall steps.
func WithUninstallSteps(steps []UninstallStep) UninstallProgressOption {
	return func(m *UninstallProgressModel) {
		m.steps = steps
		m.totalSteps = len(steps)
	}
}

// NewUninstallProgress creates a new uninstall progress view.
func NewUninstallProgress(styles theme.Styles, version string, opts ...UninstallProgressOption) UninstallProgressModel {
	keyMap := DefaultUninstallProgressKeyMap()

	header := components.NewHeader(styles, "IGOR", "Uninstalling...", version)
	footer := components.NewFooter(styles, keyMap)
	spin := components.NewSpinner(styles, "Preparing uninstall...")

	m := UninstallProgressModel{
		header:      header,
		footer:      footer,
		spinner:     spin,
		styles:      styles,
		keyMap:      keyMap,
		version:     version,
		maxLogLines: 10,
		logLines:    make([]string, 0),
		startTime:   time.Now(),
	}

	// Apply options
	for _, opt := range opts {
		opt(&m)
	}

	// Build default steps if none provided
	if len(m.steps) == 0 {
		m.steps = buildDefaultUninstallSteps()
		m.totalSteps = len(m.steps)
	}

	return m
}

// buildDefaultUninstallSteps builds the default uninstall steps.
func buildDefaultUninstallSteps() []UninstallStep {
	return []UninstallStep{
		{Name: "unload_modules", Description: "Unload kernel modules"},
		{Name: "remove_packages", Description: "Remove packages"},
		{Name: "remove_configs", Description: "Remove configuration files"},
		{Name: "restore_nouveau", Description: "Restore nouveau driver"},
		{Name: "regenerate_initramfs", Description: "Regenerate initramfs"},
	}
}

// Init implements tea.Model and starts the spinner animation.
func (m UninstallProgressModel) Init() tea.Cmd {
	return m.spinner.Init()
}

// Update implements tea.Model and handles messages for the uninstall progress screen.
func (m UninstallProgressModel) Update(msg tea.Msg) (UninstallProgressModel, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keyMap.Cancel):
			if !m.completed && !m.failed {
				m.cancelled = true
				return m, m.cancelUninstall
			}
		case key.Matches(msg, m.keyMap.Quit):
			if m.completed || m.failed {
				if m.failed {
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
		m.ready = true

	case UninstallStepStartedMsg:
		if msg.StepIndex >= 0 && msg.StepIndex < len(m.steps) {
			m.currentStep = msg.StepIndex
			m.steps[msg.StepIndex].Status = StepRunning
			m.steps[msg.StepIndex].StartTime = time.Now()
			m.spinner.SetMessage(m.steps[msg.StepIndex].Description)
		}
		m.overallProgress = float64(msg.StepIndex) / float64(m.totalSteps)

	case UninstallStepCompletedMsg:
		if msg.StepIndex >= 0 && msg.StepIndex < len(m.steps) {
			if msg.Error != nil {
				m.steps[msg.StepIndex].Status = StepFailed
				m.steps[msg.StepIndex].Error = msg.Error
				m.failed = true
				m.failureError = msg.Error
				m.spinner.Hide()
			} else {
				m.steps[msg.StepIndex].Status = StepComplete
			}
			m.steps[msg.StepIndex].EndTime = time.Now()
		}
		m.overallProgress = float64(msg.StepIndex+1) / float64(m.totalSteps)

	case UninstallLogMsg:
		m.addLogLine(msg.Message)

	case UninstallCompleteMsg:
		m.completed = true
		m.overallProgress = 1.0
		m.spinner.Hide()
		m.header.SetSubtitle("Uninstall Complete")
		m.removedPackages = msg.RemovedPackages
		m.cleanedConfigs = msg.CleanedConfigs
		m.nouveauRestored = msg.NouveauRestored
		m.needsReboot = msg.NeedsReboot
		if msg.Error != nil {
			m.failed = true
			m.failureError = msg.Error
		}

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// addLogLine adds a log line to the log output, maintaining max lines.
func (m *UninstallProgressModel) addLogLine(line string) {
	m.logLines = append(m.logLines, line)
	if len(m.logLines) > m.maxLogLines {
		m.logLines = m.logLines[1:]
	}
}

// View implements tea.Model and renders the uninstall progress screen.
func (m UninstallProgressModel) View() string {
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
func (m UninstallProgressModel) renderContent(height int) string {
	// Title
	var title string
	if m.completed && !m.failed {
		title = m.styles.Success.Render("Uninstall Complete!")
	} else if m.failed {
		title = m.styles.Error.Render("Uninstall Failed")
	} else if m.cancelled {
		title = m.styles.Warning.Render("Uninstall Cancelled")
	} else {
		title = m.styles.Title.Render("Uninstalling NVIDIA Drivers...")
	}

	// Progress bar
	progressBar := m.renderProgressBar()

	// Steps list
	stepsList := m.renderStepsList()

	// Log output
	logOutput := m.renderLogOutput()

	// Current action (spinner) if still running
	var currentAction string
	if !m.completed && !m.failed && !m.cancelled {
		currentAction = m.spinner.View()
	}

	// Elapsed time
	elapsedSection := m.renderElapsedTime()

	sections := []string{title, "", progressBar, "", stepsList}
	if currentAction != "" {
		sections = append(sections, "", currentAction)
	}
	if logOutput != "" {
		sections = append(sections, "", logOutput)
	}
	sections = append(sections, "", elapsedSection)

	// Status message at end
	if m.completed && !m.failed {
		if m.needsReboot {
			sections = append(sections, "", m.styles.Warning.Render("\u26A0 A reboot is required to complete the uninstallation"))
		}
		sections = append(sections, m.styles.Help.Render("Press 'q' to continue"))
	} else if m.failed {
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
func (m UninstallProgressModel) renderProgressBar() string {
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

// renderStepsList renders the list of uninstall steps with status indicators.
func (m UninstallProgressModel) renderStepsList() string {
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
func (m UninstallProgressModel) renderLogOutput() string {
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

// renderElapsedTime renders the elapsed time.
func (m UninstallProgressModel) renderElapsedTime() string {
	elapsed := time.Since(m.startTime)
	minutes := int(elapsed.Minutes())
	seconds := int(elapsed.Seconds()) % 60
	return m.styles.Help.Render(fmt.Sprintf("Elapsed: %02d:%02d", minutes, seconds))
}

// Navigation command functions

// cancelUninstall returns a command that signals uninstall was cancelled.
func (m UninstallProgressModel) cancelUninstall() tea.Msg {
	return UninstallCancelledProgressMsg{}
}

// navigateToComplete returns a command that signals navigation to completion screen.
func (m UninstallProgressModel) navigateToComplete() tea.Msg {
	return NavigateToUninstallCompleteMsg{
		RemovedPackages: m.removedPackages,
		CleanedConfigs:  m.cleanedConfigs,
		NouveauRestored: m.nouveauRestored,
		NeedsReboot:     m.needsReboot,
	}
}

// navigateToError returns a command that signals navigation to error screen.
func (m UninstallProgressModel) navigateToError() tea.Msg {
	failedStep := ""
	if m.currentStep >= 0 && m.currentStep < len(m.steps) {
		failedStep = m.steps[m.currentStep].Description
	}
	return NavigateToUninstallErrorMsg{
		Error:      m.failureError,
		FailedStep: failedStep,
	}
}

// Message types

// UninstallStepStartedMsg is sent when an uninstall step starts.
type UninstallStepStartedMsg struct {
	StepIndex int
	StepName  string
}

// UninstallStepCompletedMsg is sent when an uninstall step completes.
type UninstallStepCompletedMsg struct {
	StepIndex int
	StepName  string
	Error     error
}

// UninstallLogMsg adds a log line to the output.
type UninstallLogMsg struct {
	Message string
}

// UninstallCompleteMsg is sent when uninstallation is complete.
type UninstallCompleteMsg struct {
	Success         bool
	RemovedPackages []string
	CleanedConfigs  []string
	NouveauRestored bool
	NeedsReboot     bool
	Error           error
}

// UninstallCancelledProgressMsg signals uninstall was cancelled during progress.
type UninstallCancelledProgressMsg struct{}

// NavigateToUninstallCompleteMsg signals navigation to uninstall completion screen.
type NavigateToUninstallCompleteMsg struct {
	RemovedPackages []string
	CleanedConfigs  []string
	NouveauRestored bool
	NeedsReboot     bool
}

// NavigateToUninstallErrorMsg signals navigation to uninstall error screen.
type NavigateToUninstallErrorMsg struct {
	Error      error
	FailedStep string
}

// Getters

// IsComplete returns whether the uninstall is complete.
func (m UninstallProgressModel) IsComplete() bool { return m.completed }

// HasFailed returns whether the uninstall has failed.
func (m UninstallProgressModel) HasFailed() bool { return m.failed }

// IsCancelled returns whether the uninstall was cancelled.
func (m UninstallProgressModel) IsCancelled() bool { return m.cancelled }

// FailureError returns the error that caused the uninstall to fail.
func (m UninstallProgressModel) FailureError() error { return m.failureError }

// CurrentStep returns the index of the current step.
func (m UninstallProgressModel) CurrentStep() int { return m.currentStep }

// TotalSteps returns the total number of steps.
func (m UninstallProgressModel) TotalSteps() int { return m.totalSteps }

// Progress returns the overall progress as a value between 0 and 1.
func (m UninstallProgressModel) Progress() float64 { return m.overallProgress }

// Steps returns the list of uninstall steps.
func (m UninstallProgressModel) Steps() []UninstallStep { return m.steps }

// LogLines returns the current log lines.
func (m UninstallProgressModel) LogLines() []string { return m.logLines }

// RemovedPackages returns the removed packages.
func (m UninstallProgressModel) RemovedPackages() []string { return m.removedPackages }

// CleanedConfigs returns the cleaned config files.
func (m UninstallProgressModel) CleanedConfigs() []string { return m.cleanedConfigs }

// NouveauRestored returns whether nouveau was restored.
func (m UninstallProgressModel) NouveauRestored() bool { return m.nouveauRestored }

// NeedsReboot returns whether a reboot is needed.
func (m UninstallProgressModel) NeedsReboot() bool { return m.needsReboot }

// Width returns the current width of the view.
func (m UninstallProgressModel) Width() int { return m.width }

// Height returns the current height of the view.
func (m UninstallProgressModel) Height() int { return m.height }

// Ready returns whether the view is ready to render.
func (m UninstallProgressModel) Ready() bool { return m.ready }

// Version returns the application version.
func (m UninstallProgressModel) Version() string { return m.version }

// KeyMap returns the uninstall progress screen key bindings.
func (m UninstallProgressModel) KeyMap() UninstallProgressKeyMap { return m.keyMap }

// MaxLogLines returns the maximum number of log lines to display.
func (m UninstallProgressModel) MaxLogLines() int { return m.maxLogLines }

// IsFullHelpShown returns whether full help is currently displayed.
func (m UninstallProgressModel) IsFullHelpShown() bool { return m.footer.IsFullHelpShown() }

// StartTime returns the start time of the uninstall.
func (m UninstallProgressModel) StartTime() time.Time { return m.startTime }

// SetSize updates the view dimensions.
func (m *UninstallProgressModel) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.header.SetWidth(width)
	m.footer.SetWidth(width)
	m.ready = true
}

// SetMaxLogLines sets the maximum number of log lines to display.
func (m *UninstallProgressModel) SetMaxLogLines(max int) {
	if max > 0 {
		m.maxLogLines = max
	}
}

// ClearLogLines clears all log lines.
func (m *UninstallProgressModel) ClearLogLines() {
	m.logLines = make([]string, 0)
}

// AddLogLine adds a log line (exposed for testing).
func (m *UninstallProgressModel) AddLogLine(line string) {
	m.addLogLine(line)
}

// MarkStepComplete marks a step as complete (for testing).
func (m *UninstallProgressModel) MarkStepComplete(index int) {
	if index >= 0 && index < len(m.steps) {
		m.steps[index].Status = StepComplete
		m.steps[index].EndTime = time.Now()
	}
}

// MarkStepFailed marks a step as failed (for testing).
func (m *UninstallProgressModel) MarkStepFailed(index int, err error) {
	if index >= 0 && index < len(m.steps) {
		m.steps[index].Status = StepFailed
		m.steps[index].EndTime = time.Now()
		m.steps[index].Error = err
		m.failed = true
		m.failureError = err
	}
}

// SetComplete marks the uninstall as complete (for testing).
func (m *UninstallProgressModel) SetComplete(needsReboot bool) {
	m.completed = true
	m.overallProgress = 1.0
	m.needsReboot = needsReboot
	m.spinner.Hide()
	m.header.SetSubtitle("Uninstall Complete")
}

// SetRemovedPackages sets the removed packages (for testing).
func (m *UninstallProgressModel) SetRemovedPackages(packages []string) {
	m.removedPackages = packages
}

// SetCleanedConfigs sets the cleaned configs (for testing).
func (m *UninstallProgressModel) SetCleanedConfigs(configs []string) {
	m.cleanedConfigs = configs
}

// SetNouveauRestored sets whether nouveau was restored (for testing).
func (m *UninstallProgressModel) SetNouveauRestored(restored bool) {
	m.nouveauRestored = restored
}

// SetNeedsReboot sets whether a reboot is needed (for testing).
func (m *UninstallProgressModel) SetNeedsReboot(needsReboot bool) {
	m.needsReboot = needsReboot
}

// SetStartTime sets the start time (for testing).
func (m *UninstallProgressModel) SetStartTime(t time.Time) {
	m.startTime = t
}
