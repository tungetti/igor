// Package ui provides the Bubble Tea TUI application for Igor.
// It implements the main model, message handling, and view rendering
// for the interactive NVIDIA driver installation interface.
package ui

import (
	"context"
	"regexp"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/tungetti/igor/internal/distro"
	"github.com/tungetti/igor/internal/exec"
	"github.com/tungetti/igor/internal/gpu"
	"github.com/tungetti/igor/internal/gpu/kernel"
	"github.com/tungetti/igor/internal/gpu/nouveau"
	"github.com/tungetti/igor/internal/gpu/nvidia"
	"github.com/tungetti/igor/internal/gpu/pci"
	"github.com/tungetti/igor/internal/gpu/smi"
	"github.com/tungetti/igor/internal/gpu/validator"
	"github.com/tungetti/igor/internal/pkg"
	"github.com/tungetti/igor/internal/pkg/factory"
	"github.com/tungetti/igor/internal/ui/theme"
	"github.com/tungetti/igor/internal/ui/views"
)

// ViewState represents the current view in the application.
type ViewState int

const (
	// ViewWelcome is the initial welcome screen.
	ViewWelcome ViewState = iota
	// ViewDetecting is shown while detecting system information.
	ViewDetecting
	// ViewSystemInfo displays detected system information.
	ViewSystemInfo
	// ViewDriverSelection allows the user to select a driver version.
	ViewDriverSelection
	// ViewConfirmation asks for confirmation before installation.
	ViewConfirmation
	// ViewInstalling is shown during driver installation.
	ViewInstalling
	// ViewComplete is shown when installation is finished.
	ViewComplete
	// ViewError is shown when an error occurs.
	ViewError
)

// String returns the string representation of a ViewState.
func (v ViewState) String() string {
	switch v {
	case ViewWelcome:
		return "Welcome"
	case ViewDetecting:
		return "Detecting"
	case ViewSystemInfo:
		return "SystemInfo"
	case ViewDriverSelection:
		return "DriverSelection"
	case ViewConfirmation:
		return "Confirmation"
	case ViewInstalling:
		return "Installing"
	case ViewComplete:
		return "Complete"
	case ViewError:
		return "Error"
	default:
		return "Unknown"
	}
}

// Model is the main Bubble Tea model for the Igor TUI.
type Model struct {
	// CurrentView is the current view state.
	CurrentView ViewState

	// Width is the terminal width.
	Width int
	// Height is the terminal height.
	Height int

	// Ready indicates if the TUI is ready to render.
	Ready bool
	// Quitting indicates if the application is quitting.
	Quitting bool

	// Error holds any error that occurred.
	Error error

	// ctx is the context for cancellation.
	ctx context.Context
	// cancel is the cancel function for the context.
	cancel context.CancelFunc

	// keyMap holds the key bindings.
	keyMap KeyMap

	// Theme and styles
	theme  *theme.Theme
	styles theme.Styles

	// Version string
	version string

	// View models (initialized lazily or on navigation)
	welcomeView      views.WelcomeModel
	detectionView    views.DetectionModel
	selectionView    views.SelectionModel
	confirmationView views.ConfirmationModel
	progressView     views.ProgressModel
	completeView     views.CompleteModel
	errorView        views.ErrorModel

	// Shared state passed between views
	gpuInfo    *gpu.GPUInfo
	driver     views.DriverOption
	components []views.ComponentOption
}

// New creates a new TUI application model.
func New() Model {
	return NewWithVersion("dev")
}

// NewWithVersion creates a new TUI application model with a specified version.
func NewWithVersion(version string) Model {
	ctx, cancel := context.WithCancel(context.Background())
	t := theme.DefaultTheme()
	s := theme.NewStyles(t)

	return Model{
		CurrentView: ViewWelcome,
		ctx:         ctx,
		cancel:      cancel,
		keyMap:      DefaultKeyMap(),
		theme:       t,
		styles:      s,
		version:     version,
		welcomeView: views.NewWelcome(s, version),
	}
}

// NewWithContext creates a new TUI application model with a custom context.
func NewWithContext(ctx context.Context) Model {
	return NewWithContextAndVersion(ctx, "dev")
}

// NewWithContextAndVersion creates a new TUI application model with a custom context and version.
func NewWithContextAndVersion(ctx context.Context, version string) Model {
	childCtx, cancel := context.WithCancel(ctx)
	t := theme.DefaultTheme()
	s := theme.NewStyles(t)

	return Model{
		CurrentView: ViewWelcome,
		ctx:         childCtx,
		cancel:      cancel,
		keyMap:      DefaultKeyMap(),
		theme:       t,
		styles:      s,
		version:     version,
		welcomeView: views.NewWelcome(s, version),
	}
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		tea.EnterAltScreen,
		waitForWindowSize,
	)
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKeyPress(msg)

	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		m.Ready = true
		// Propagate window size to all initialized views
		m.propagateWindowSize(msg)
		return m, tea.Batch(cmds...)

	case QuitMsg:
		m.Quitting = true
		m.cancel()
		return m, tea.Quit

	case ErrorMsg:
		m.Error = msg.Err
		m.CurrentView = ViewError
		return m, nil

	case NavigateMsg:
		m.CurrentView = msg.View
		return m, nil

	case WindowReadyMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		m.Ready = true
		return m, nil

	// View navigation messages from views package
	case views.StartDetectionMsg:
		m.CurrentView = ViewDetecting
		m.detectionView = m.initDetectionView()
		// Start both the spinner animation and the actual detection
		return m, tea.Batch(m.detectionView.Init(), m.runDetection())

	case views.DetectionCompleteMsg:
		m.gpuInfo = msg.GPUInfo
		m.CurrentView = ViewDriverSelection
		m.selectionView = m.initSelectionView(msg.GPUInfo)
		sizeMsg := tea.WindowSizeMsg{Width: m.Width, Height: m.Height}
		m.selectionView.SetSize(sizeMsg.Width, sizeMsg.Height)
		return m, nil

	case views.NavigateToDriverSelectionMsg:
		m.gpuInfo = msg.GPUInfo
		m.CurrentView = ViewDriverSelection
		m.selectionView = m.initSelectionView(msg.GPUInfo)
		sizeMsg := tea.WindowSizeMsg{Width: m.Width, Height: m.Height}
		m.selectionView.SetSize(sizeMsg.Width, sizeMsg.Height)
		return m, nil

	case views.NavigateToWelcomeMsg:
		m.CurrentView = ViewWelcome
		return m, nil

	case views.NavigateToDetectionMsg:
		m.CurrentView = ViewDetecting
		m.detectionView = m.initDetectionView()
		return m, m.detectionView.Init()

	case views.NavigateToConfirmationMsg:
		m.gpuInfo = msg.GPUInfo
		m.driver = msg.SelectedDriver
		m.components = msg.SelectedComponents
		m.CurrentView = ViewConfirmation
		m.confirmationView = m.initConfirmationView(msg.GPUInfo, msg.SelectedDriver, msg.SelectedComponents)
		sizeMsg := tea.WindowSizeMsg{Width: m.Width, Height: m.Height}
		m.confirmationView.SetSize(sizeMsg.Width, sizeMsg.Height)
		return m, nil

	case views.NavigateBackToSelectionMsg:
		m.CurrentView = ViewDriverSelection
		// Re-use existing selection view if available
		sizeMsg := tea.WindowSizeMsg{Width: m.Width, Height: m.Height}
		m.selectionView.SetSize(sizeMsg.Width, sizeMsg.Height)
		return m, nil

	case views.StartInstallationMsg:
		m.gpuInfo = msg.GPUInfo
		m.driver = msg.Driver
		m.components = msg.Components
		m.CurrentView = ViewInstalling
		m.progressView = m.initProgressView(msg.GPUInfo, msg.Driver, msg.Components)
		return m, m.progressView.Init()

	case views.NavigateToCompleteMsg:
		m.gpuInfo = msg.GPUInfo
		m.driver = msg.Driver
		m.components = msg.Components
		m.CurrentView = ViewComplete
		m.completeView = m.initCompleteView(msg.GPUInfo, msg.Driver, msg.Components)
		sizeMsg := tea.WindowSizeMsg{Width: m.Width, Height: m.Height}
		m.completeView.SetSize(sizeMsg.Width, sizeMsg.Height)
		return m, nil

	case views.NavigateToErrorMsg:
		m.Error = msg.Error
		m.CurrentView = ViewError
		m.errorView = m.initErrorView(msg.Error, msg.FailedStep)
		sizeMsg := tea.WindowSizeMsg{Width: m.Width, Height: m.Height}
		m.errorView.SetSize(sizeMsg.Width, sizeMsg.Height)
		return m, nil

	case views.RebootRequestedMsg:
		// For now, just quit. In a full implementation, this would trigger a reboot.
		m.Quitting = true
		m.cancel()
		return m, tea.Quit

	case views.ExitRequestedMsg:
		m.Quitting = true
		m.cancel()
		return m, tea.Quit

	case views.ErrorExitRequestedMsg:
		m.Quitting = true
		m.cancel()
		return m, tea.Quit

	case views.RetryRequestedMsg:
		m.CurrentView = ViewWelcome
		m.Error = nil
		return m, nil

	// Spinner tick messages for animation
	case spinner.TickMsg:
		return m.handleSpinnerTick(msg)
	}

	// Delegate to active view's Update for view-specific handling
	return m.delegateToActiveView(msg)
}

// View implements tea.Model.
func (m Model) View() string {
	if m.Quitting {
		return "Goodbye!\n"
	}

	if !m.Ready {
		return "Initializing..."
	}

	// Render actual view models
	switch m.CurrentView {
	case ViewWelcome:
		return m.welcomeView.View()
	case ViewDetecting:
		return m.detectionView.View()
	case ViewSystemInfo:
		return m.renderPlaceholder("System Information")
	case ViewDriverSelection:
		return m.selectionView.View()
	case ViewConfirmation:
		return m.confirmationView.View()
	case ViewInstalling:
		return m.progressView.View()
	case ViewComplete:
		return m.completeView.View()
	case ViewError:
		return m.errorView.View()
	default:
		return m.renderPlaceholder("Unknown View")
	}
}

// renderPlaceholder renders a centered placeholder view.
func (m Model) renderPlaceholder(title string) string {
	// Simple centered placeholder
	style := lipgloss.NewStyle().
		Width(m.Width).
		Height(m.Height).
		Align(lipgloss.Center, lipgloss.Center)

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("86"))

	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241"))

	content := titleStyle.Render(title) + "\n\n" +
		helpStyle.Render("Press 'q' to quit, '?' for help")

	return style.Render(content)
}

// handleKeyPress handles key press events.
func (m Model) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Only handle ctrl+c globally - let views handle other keys
	switch msg.String() {
	case "ctrl+c":
		return m, func() tea.Msg { return QuitMsg{} }
	}

	// Delegate all other key presses to the active view
	return m.delegateToActiveView(msg)
}

// waitForWindowSize returns a command that signals the window is ready.
// This is a placeholder that will be triggered by the actual WindowSizeMsg.
func waitForWindowSize() tea.Msg {
	return nil
}

// Context returns the application context.
func (m Model) Context() context.Context {
	return m.ctx
}

// Shutdown cancels the context and performs cleanup.
func (m *Model) Shutdown() {
	if m.cancel != nil {
		m.cancel()
	}
}

// KeyMap returns the current key bindings.
func (m Model) KeyMap() KeyMap {
	return m.keyMap
}

// SetKeyMap sets custom key bindings.
func (m *Model) SetKeyMap(km KeyMap) {
	m.keyMap = km
}

// IsReady returns whether the TUI is ready to render.
func (m Model) IsReady() bool {
	return m.Ready
}

// IsQuitting returns whether the application is quitting.
func (m Model) IsQuitting() bool {
	return m.Quitting
}

// SetError sets an error and transitions to the error view.
func (m *Model) SetError(err error) {
	m.Error = err
	m.CurrentView = ViewError
}

// ClearError clears the error and returns to the welcome view.
func (m *Model) ClearError() {
	m.Error = nil
	m.CurrentView = ViewWelcome
}

// NavigateTo transitions to the specified view.
func (m *Model) NavigateTo(view ViewState) {
	m.CurrentView = view
}

// Version returns the application version.
func (m Model) Version() string {
	return m.version
}

// =============================================================================
// View Initialization Helpers
// =============================================================================

// initDetectionView creates and initializes the detection view.
func (m Model) initDetectionView() views.DetectionModel {
	view := views.NewDetection(m.styles, m.version)
	view.SetSize(m.Width, m.Height)
	return view
}

// initSelectionView creates and initializes the selection view with GPU info.
func (m Model) initSelectionView(gpuInfo *gpu.GPUInfo) views.SelectionModel {
	view := views.NewSelection(m.styles, m.version, gpuInfo)
	view.SetSize(m.Width, m.Height)
	return view
}

// initConfirmationView creates and initializes the confirmation view.
func (m Model) initConfirmationView(gpuInfo *gpu.GPUInfo, driver views.DriverOption, components []views.ComponentOption) views.ConfirmationModel {
	view := views.NewConfirmation(m.styles, m.version, gpuInfo, driver, components)
	view.SetSize(m.Width, m.Height)
	return view
}

// initProgressView creates and initializes the progress view.
func (m Model) initProgressView(gpuInfo *gpu.GPUInfo, driver views.DriverOption, components []views.ComponentOption) views.ProgressModel {
	view := views.NewProgress(m.styles, m.version, gpuInfo, driver, components)
	view.SetSize(m.Width, m.Height)
	return view
}

// initCompleteView creates and initializes the complete view.
func (m Model) initCompleteView(gpuInfo *gpu.GPUInfo, driver views.DriverOption, components []views.ComponentOption) views.CompleteModel {
	view := views.NewComplete(m.styles, m.version, gpuInfo, driver, components)
	view.SetSize(m.Width, m.Height)
	return view
}

// initErrorView creates and initializes the error view.
func (m Model) initErrorView(err error, failedStep string) views.ErrorModel {
	view := views.NewError(m.styles, m.version, err, failedStep)
	view.SetSize(m.Width, m.Height)
	return view
}

// =============================================================================
// Window Size Propagation
// =============================================================================

// propagateWindowSize propagates window size to all initialized views.
// Views are updated directly; no commands are returned as SetSize is synchronous.
func (m *Model) propagateWindowSize(msg tea.WindowSizeMsg) {
	// Update welcome view (always initialized)
	m.welcomeView.SetSize(msg.Width, msg.Height)

	// Update detection view if initialized
	if m.detectionView.Ready() || m.CurrentView == ViewDetecting {
		m.detectionView.SetSize(msg.Width, msg.Height)
	}

	// Update selection view if initialized
	if m.selectionView.Ready() || m.CurrentView == ViewDriverSelection {
		m.selectionView.SetSize(msg.Width, msg.Height)
	}

	// Update confirmation view if initialized
	if m.confirmationView.Ready() || m.CurrentView == ViewConfirmation {
		m.confirmationView.SetSize(msg.Width, msg.Height)
	}

	// Update progress view if initialized
	if m.progressView.Ready() || m.CurrentView == ViewInstalling {
		m.progressView.SetSize(msg.Width, msg.Height)
	}

	// Update complete view if initialized
	if m.completeView.Ready() || m.CurrentView == ViewComplete {
		m.completeView.SetSize(msg.Width, msg.Height)
	}

	// Update error view if initialized
	if m.errorView.Ready() || m.CurrentView == ViewError {
		m.errorView.SetSize(msg.Width, msg.Height)
	}
}

// =============================================================================
// Spinner and View Delegation
// =============================================================================

// handleSpinnerTick handles spinner tick messages by delegating to the active view.
func (m Model) handleSpinnerTick(msg spinner.TickMsg) (tea.Model, tea.Cmd) {
	switch m.CurrentView {
	case ViewDetecting:
		var cmd tea.Cmd
		m.detectionView, cmd = m.detectionView.Update(msg)
		return m, cmd
	case ViewInstalling:
		var cmd tea.Cmd
		m.progressView, cmd = m.progressView.Update(msg)
		return m, cmd
	}
	return m, nil
}

// delegateToActiveView delegates messages to the currently active view.
func (m Model) delegateToActiveView(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m.CurrentView {
	case ViewWelcome:
		var cmd tea.Cmd
		m.welcomeView, cmd = m.welcomeView.Update(msg)
		return m, cmd
	case ViewDetecting:
		var cmd tea.Cmd
		m.detectionView, cmd = m.detectionView.Update(msg)
		return m, cmd
	case ViewDriverSelection:
		var cmd tea.Cmd
		m.selectionView, cmd = m.selectionView.Update(msg)
		return m, cmd
	case ViewConfirmation:
		var cmd tea.Cmd
		m.confirmationView, cmd = m.confirmationView.Update(msg)
		return m, cmd
	case ViewInstalling:
		var cmd tea.Cmd
		m.progressView, cmd = m.progressView.Update(msg)
		return m, cmd
	case ViewComplete:
		var cmd tea.Cmd
		m.completeView, cmd = m.completeView.Update(msg)
		return m, cmd
	case ViewError:
		var cmd tea.Cmd
		m.errorView, cmd = m.errorView.Update(msg)
		return m, cmd
	}
	return m, nil
}

// =============================================================================
// GPU Detection
// =============================================================================

// runDetection runs the GPU detection asynchronously and returns a command.
func (m Model) runDetection() tea.Cmd {
	return func() tea.Msg {
		// Create the executor with default options
		executor := exec.NewExecutor(exec.DefaultOptions(), nil)

		// Create individual detectors
		pciScanner := pci.NewScanner()
		gpuDatabase := nvidia.NewDatabase()
		smiParser := smi.NewParser(executor)
		nouveauDetector := nouveau.NewDetector()
		kernelDetector := kernel.NewDetector(kernel.WithExecutor(executor))
		systemValidator := validator.NewValidator(validator.WithExecutor(executor))

		// Create the orchestrator with all detectors
		orchestrator := gpu.NewOrchestrator(
			gpu.WithPCIScanner(pciScanner),
			gpu.WithGPUDatabase(gpuDatabase),
			gpu.WithSMIParser(smiParser),
			gpu.WithNouveauDetector(nouveauDetector),
			gpu.WithKernelDetector(kernelDetector),
			gpu.WithSystemValidator(systemValidator),
		)

		// Run detection
		gpuInfo, err := orchestrator.DetectAll(m.ctx)
		if err != nil {
			return views.DetectionErrorMsg{Error: err}
		}

		// Query available drivers from the package manager
		availableDrivers := queryAvailableDrivers(m.ctx, executor)
		if len(availableDrivers) > 0 {
			gpuInfo.AvailableDrivers = availableDrivers
		}

		return views.DetectionCompleteMsg{GPUInfo: gpuInfo}
	}
}

// queryAvailableDrivers queries the package manager for available NVIDIA drivers.
func queryAvailableDrivers(ctx context.Context, executor exec.Executor) []gpu.AvailableDriver {
	// Detect distribution
	detector := distro.NewDetector(executor, nil)
	dist, err := detector.Detect(ctx)
	if err != nil {
		return nil
	}

	// Create package manager for this distribution
	pkgFactory := factory.NewFactory(executor, nil, detector)
	pkgManager, err := pkgFactory.CreateForDistribution(dist)
	if err != nil {
		return nil
	}

	// Search for NVIDIA driver packages
	searchOpts := pkg.DefaultSearchOptions()

	packages, err := pkgManager.Search(ctx, "nvidia", searchOpts)
	if err != nil {
		return nil
	}

	// Parse driver versions from package names
	return parseDriverPackages(packages, dist)
}

// parseDriverPackages extracts driver versions from package search results.
func parseDriverPackages(packages []pkg.Package, dist *distro.Distribution) []gpu.AvailableDriver {
	// Regex patterns for different distros
	var versionPattern *regexp.Regexp

	switch dist.Family.String() {
	case "arch":
		// Arch: nvidia, nvidia-lts, nvidia-dkms, nvidia-open, etc.
		// Also check AUR for specific versions
		versionPattern = regexp.MustCompile(`nvidia[-_]?(\d{3})`)
	case "debian":
		// Debian/Ubuntu: nvidia-driver-550, nvidia-driver-545, etc.
		versionPattern = regexp.MustCompile(`nvidia-driver-(\d{3})`)
	case "rhel":
		// RHEL/Fedora: nvidia-driver or akmod-nvidia with version in description
		versionPattern = regexp.MustCompile(`nvidia.*?(\d{3})`)
	case "suse":
		// openSUSE: nvidia-driver-G06, nvidia-driver-G05, etc. or version numbers
		versionPattern = regexp.MustCompile(`nvidia.*?(\d{3})`)
	default:
		versionPattern = regexp.MustCompile(`nvidia.*?(\d{3})`)
	}

	// Track unique versions
	versionMap := make(map[string]*gpu.AvailableDriver)

	for _, p := range packages {
		// Check package name
		matches := versionPattern.FindStringSubmatch(strings.ToLower(p.Name))
		if len(matches) >= 2 {
			version := matches[1]
			if _, exists := versionMap[version]; !exists {
				versionMap[version] = &gpu.AvailableDriver{
					Version:     version,
					FullVersion: p.Version,
					PackageName: p.Name,
					Branch:      classifyDriverBranch(version),
				}
			}
		}

		// Also check version field for full version
		if existing, exists := versionMap[extractMajorVersion(p.Version)]; exists {
			if existing.FullVersion == "" || len(p.Version) > len(existing.FullVersion) {
				existing.FullVersion = p.Version
			}
		}
	}

	// Handle Arch Linux special case - check for main nvidia package
	if dist.Family.String() == "arch" {
		for _, p := range packages {
			if p.Name == "nvidia" || p.Name == "nvidia-dkms" || p.Name == "nvidia-open" {
				// Get version from the package version field
				majorVer := extractMajorVersion(p.Version)
				if majorVer != "" {
					if _, exists := versionMap[majorVer]; !exists {
						versionMap[majorVer] = &gpu.AvailableDriver{
							Version:     majorVer,
							FullVersion: p.Version,
							PackageName: p.Name,
							Branch:      "Latest",
							Recommended: true,
						}
					} else {
						// Update existing with full version
						versionMap[majorVer].FullVersion = p.Version
						if p.Name == "nvidia" || p.Name == "nvidia-dkms" {
							versionMap[majorVer].Recommended = true
						}
					}
				}
			}
		}
	}

	// Convert map to slice and sort by version (descending)
	drivers := make([]gpu.AvailableDriver, 0, len(versionMap))
	for _, d := range versionMap {
		drivers = append(drivers, *d)
	}

	sort.Slice(drivers, func(i, j int) bool {
		return drivers[i].Version > drivers[j].Version
	})

	// Mark the first (highest version) as recommended if none marked
	hasRecommended := false
	for _, d := range drivers {
		if d.Recommended {
			hasRecommended = true
			break
		}
	}
	if !hasRecommended && len(drivers) > 0 {
		drivers[0].Recommended = true
	}

	return drivers
}

// extractMajorVersion extracts the major version (e.g., "560" from "560.35.03").
func extractMajorVersion(version string) string {
	parts := strings.Split(version, ".")
	if len(parts) > 0 {
		// Get first 3 digits if available
		major := parts[0]
		if len(major) >= 3 {
			return major[:3]
		}
		return major
	}
	return ""
}

// classifyDriverBranch classifies a driver version into a branch.
func classifyDriverBranch(version string) string {
	switch {
	case version >= "560":
		return "Latest"
	case version >= "550":
		return "Production"
	case version >= "535":
		return "LTS"
	case version >= "470":
		return "Legacy"
	default:
		return "Legacy"
	}
}
