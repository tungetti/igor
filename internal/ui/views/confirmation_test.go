package views

import (
	"testing"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tungetti/igor/internal/gpu"
	"github.com/tungetti/igor/internal/gpu/kernel"
	"github.com/tungetti/igor/internal/gpu/nouveau"
	"github.com/tungetti/igor/internal/gpu/nvidia"
	"github.com/tungetti/igor/internal/gpu/pci"
)

// =============================================================================
// ConfirmationKeyMap Tests
// =============================================================================

func TestDefaultConfirmationKeyMap(t *testing.T) {
	km := DefaultConfirmationKeyMap()

	// Verify all key bindings are set
	assert.NotEmpty(t, km.Confirm.Keys())
	assert.NotEmpty(t, km.Back.Keys())
	assert.NotEmpty(t, km.Quit.Keys())
	assert.NotEmpty(t, km.Left.Keys())
	assert.NotEmpty(t, km.Right.Keys())
	assert.NotEmpty(t, km.Help.Keys())
}

func TestConfirmationKeyMap_Confirm(t *testing.T) {
	km := DefaultConfirmationKeyMap()

	assert.Contains(t, km.Confirm.Keys(), "enter")
	assert.Contains(t, km.Confirm.Keys(), "y")
}

func TestConfirmationKeyMap_Back(t *testing.T) {
	km := DefaultConfirmationKeyMap()

	assert.Contains(t, km.Back.Keys(), "esc")
	assert.Contains(t, km.Back.Keys(), "n")
	assert.Contains(t, km.Back.Keys(), "backspace")
}

func TestConfirmationKeyMap_Quit(t *testing.T) {
	km := DefaultConfirmationKeyMap()

	assert.Contains(t, km.Quit.Keys(), "q")
	assert.Contains(t, km.Quit.Keys(), "ctrl+c")
}

func TestConfirmationKeyMap_Left(t *testing.T) {
	km := DefaultConfirmationKeyMap()

	assert.Contains(t, km.Left.Keys(), "left")
	assert.Contains(t, km.Left.Keys(), "h")
}

func TestConfirmationKeyMap_Right(t *testing.T) {
	km := DefaultConfirmationKeyMap()

	assert.Contains(t, km.Right.Keys(), "right")
	assert.Contains(t, km.Right.Keys(), "l")
	assert.Contains(t, km.Right.Keys(), "tab")
}

func TestConfirmationKeyMap_Help(t *testing.T) {
	km := DefaultConfirmationKeyMap()

	assert.Contains(t, km.Help.Keys(), "?")
}

func TestConfirmationKeyMap_ShortHelp(t *testing.T) {
	km := DefaultConfirmationKeyMap()

	shortHelp := km.ShortHelp()

	assert.Len(t, shortHelp, 3)
	assert.Equal(t, km.Confirm, shortHelp[0])
	assert.Equal(t, km.Back, shortHelp[1])
	assert.Equal(t, km.Quit, shortHelp[2])
}

func TestConfirmationKeyMap_FullHelp(t *testing.T) {
	km := DefaultConfirmationKeyMap()

	fullHelp := km.FullHelp()

	assert.Len(t, fullHelp, 3)

	// First row: confirm and back
	assert.Len(t, fullHelp[0], 2)
	assert.Equal(t, km.Confirm, fullHelp[0][0])
	assert.Equal(t, km.Back, fullHelp[0][1])

	// Second row: left and right
	assert.Len(t, fullHelp[1], 2)
	assert.Equal(t, km.Left, fullHelp[1][0])
	assert.Equal(t, km.Right, fullHelp[1][1])

	// Third row: quit and help
	assert.Len(t, fullHelp[2], 2)
	assert.Equal(t, km.Quit, fullHelp[2][0])
	assert.Equal(t, km.Help, fullHelp[2][1])
}

func TestConfirmationKeyMap_ImplementsHelpKeyMap(t *testing.T) {
	km := DefaultConfirmationKeyMap()

	// Should be able to call these methods as per help.KeyMap interface
	shortHelp := km.ShortHelp()
	fullHelp := km.FullHelp()

	assert.NotEmpty(t, shortHelp)
	assert.NotEmpty(t, fullHelp)

	// Verify types are correct
	for _, binding := range shortHelp {
		assert.NotEmpty(t, binding.Keys())
	}

	for _, row := range fullHelp {
		for _, binding := range row {
			assert.NotEmpty(t, binding.Keys())
		}
	}
}

func TestConfirmationKeyMap_BindingsHaveHelp(t *testing.T) {
	km := DefaultConfirmationKeyMap()

	bindings := []struct {
		name    string
		binding key.Binding
	}{
		{"Confirm", km.Confirm},
		{"Back", km.Back},
		{"Quit", km.Quit},
		{"Left", km.Left},
		{"Right", km.Right},
		{"Help", km.Help},
	}

	for _, b := range bindings {
		t.Run(b.name, func(t *testing.T) {
			help := b.binding.Help()
			assert.NotEmpty(t, help.Key, "binding should have key help")
			assert.NotEmpty(t, help.Desc, "binding should have description")
		})
	}
}

// =============================================================================
// BuildWarnings Tests
// =============================================================================

func TestBuildWarnings_NilGPUInfo(t *testing.T) {
	driver := DriverOption{Version: "550", Branch: "Latest"}
	warnings := buildWarnings(nil, driver)

	assert.Empty(t, warnings)
}

func TestBuildWarnings_NoWarnings(t *testing.T) {
	gpuInfo := &gpu.GPUInfo{
		NouveauStatus: &nouveau.Status{Loaded: false},
		KernelInfo:    &kernel.KernelInfo{SecureBootEnabled: false, HeadersInstalled: true},
		InstalledDriver: &gpu.DriverInfo{
			Installed: false,
		},
	}
	driver := DriverOption{Version: "550", Branch: "Latest"}

	warnings := buildWarnings(gpuInfo, driver)

	assert.Empty(t, warnings)
}

func TestBuildWarnings_NouveauLoaded(t *testing.T) {
	gpuInfo := &gpu.GPUInfo{
		NouveauStatus: &nouveau.Status{Loaded: true},
	}
	driver := DriverOption{Version: "550", Branch: "Latest"}

	warnings := buildWarnings(gpuInfo, driver)

	assert.Len(t, warnings, 1)
	assert.Contains(t, warnings[0], "Nouveau")
	assert.Contains(t, warnings[0], "reboot")
}

func TestBuildWarnings_SecureBootEnabled(t *testing.T) {
	gpuInfo := &gpu.GPUInfo{
		KernelInfo: &kernel.KernelInfo{SecureBootEnabled: true},
	}
	driver := DriverOption{Version: "550", Branch: "Latest"}

	warnings := buildWarnings(gpuInfo, driver)

	assert.Len(t, warnings, 2) // Also includes headers warning
	assert.Contains(t, warnings[0], "Secure Boot")
	assert.Contains(t, warnings[0], "signing")
}

func TestBuildWarnings_KernelHeadersNotInstalled(t *testing.T) {
	gpuInfo := &gpu.GPUInfo{
		KernelInfo: &kernel.KernelInfo{HeadersInstalled: false},
	}
	driver := DriverOption{Version: "550", Branch: "Latest"}

	warnings := buildWarnings(gpuInfo, driver)

	assert.Len(t, warnings, 1)
	assert.Contains(t, warnings[0], "Kernel headers")
	assert.Contains(t, warnings[0], "DKMS")
}

func TestBuildWarnings_ExistingDriverInstalled(t *testing.T) {
	gpuInfo := &gpu.GPUInfo{
		KernelInfo: &kernel.KernelInfo{HeadersInstalled: true},
		InstalledDriver: &gpu.DriverInfo{
			Installed: true,
			Version:   "535.113.01",
		},
	}
	driver := DriverOption{Version: "550", Branch: "Latest"}

	warnings := buildWarnings(gpuInfo, driver)

	assert.Len(t, warnings, 1)
	assert.Contains(t, warnings[0], "Existing driver")
	assert.Contains(t, warnings[0], "replaced")
}

func TestBuildWarnings_AllWarnings(t *testing.T) {
	gpuInfo := &gpu.GPUInfo{
		NouveauStatus: &nouveau.Status{Loaded: true},
		KernelInfo:    &kernel.KernelInfo{SecureBootEnabled: true, HeadersInstalled: false},
		InstalledDriver: &gpu.DriverInfo{
			Installed: true,
			Version:   "535.113.01",
		},
	}
	driver := DriverOption{Version: "550", Branch: "Latest"}

	warnings := buildWarnings(gpuInfo, driver)

	assert.Len(t, warnings, 4)
}

func TestBuildWarnings_NilNouveauStatus(t *testing.T) {
	gpuInfo := &gpu.GPUInfo{
		NouveauStatus: nil,
		KernelInfo:    &kernel.KernelInfo{HeadersInstalled: true},
	}
	driver := DriverOption{Version: "550", Branch: "Latest"}

	warnings := buildWarnings(gpuInfo, driver)

	assert.Empty(t, warnings)
}

func TestBuildWarnings_NilKernelInfo(t *testing.T) {
	gpuInfo := &gpu.GPUInfo{
		NouveauStatus: &nouveau.Status{Loaded: false},
		KernelInfo:    nil,
	}
	driver := DriverOption{Version: "550", Branch: "Latest"}

	warnings := buildWarnings(gpuInfo, driver)

	assert.Empty(t, warnings)
}

func TestBuildWarnings_NilInstalledDriver(t *testing.T) {
	gpuInfo := &gpu.GPUInfo{
		KernelInfo:      &kernel.KernelInfo{HeadersInstalled: true},
		InstalledDriver: nil,
	}
	driver := DriverOption{Version: "550", Branch: "Latest"}

	warnings := buildWarnings(gpuInfo, driver)

	assert.Empty(t, warnings)
}

// =============================================================================
// NewConfirmation Tests
// =============================================================================

func TestNewConfirmation(t *testing.T) {
	styles := getTestStyles()
	version := "1.0.0"
	gpuInfo := createMockGPUInfo()
	driver := DriverOption{Version: "550", Branch: "Latest"}
	comps := []ComponentOption{{ID: "driver", Name: "NVIDIA Driver", Selected: true}}

	m := NewConfirmation(styles, version, gpuInfo, driver, comps)

	assert.Equal(t, version, m.Version())
	assert.Equal(t, gpuInfo, m.GPUInfo())
	assert.Equal(t, driver, m.SelectedDriver())
	assert.Equal(t, comps, m.SelectedComponents())
	assert.Equal(t, 0, m.Width())
	assert.Equal(t, 0, m.Height())
	assert.False(t, m.Ready())
	assert.Equal(t, 0, m.FocusedButtonIndex())
}

func TestNewConfirmation_WithNilGPUInfo(t *testing.T) {
	styles := getTestStyles()
	driver := DriverOption{Version: "550", Branch: "Latest"}
	comps := []ComponentOption{}

	m := NewConfirmation(styles, "1.0.0", nil, driver, comps)

	assert.Nil(t, m.GPUInfo())
	assert.Empty(t, m.Warnings())
}

func TestNewConfirmation_WithWarnings(t *testing.T) {
	styles := getTestStyles()
	gpuInfo := &gpu.GPUInfo{
		NouveauStatus: &nouveau.Status{Loaded: true},
	}
	driver := DriverOption{Version: "550", Branch: "Latest"}
	comps := []ComponentOption{}

	m := NewConfirmation(styles, "1.0.0", gpuInfo, driver, comps)

	assert.NotEmpty(t, m.Warnings())
}

func TestNewConfirmation_KeyMapInitialized(t *testing.T) {
	styles := getTestStyles()

	m := NewConfirmation(styles, "1.0.0", nil, DriverOption{}, nil)
	km := m.KeyMap()

	assert.NotEmpty(t, km.Confirm.Keys())
	assert.NotEmpty(t, km.Back.Keys())
	assert.NotEmpty(t, km.Quit.Keys())
}

// =============================================================================
// Init Tests
// =============================================================================

func TestConfirmationModel_Init(t *testing.T) {
	styles := getTestStyles()
	m := NewConfirmation(styles, "1.0.0", nil, DriverOption{}, nil)

	cmd := m.Init()

	assert.Nil(t, cmd, "Init should return nil for confirmation view")
}

// =============================================================================
// Update Tests - WindowSizeMsg
// =============================================================================

func TestConfirmationModel_Update_WindowSizeMsg(t *testing.T) {
	styles := getTestStyles()
	m := NewConfirmation(styles, "1.0.0", nil, DriverOption{}, nil)

	assert.False(t, m.Ready())

	msg := tea.WindowSizeMsg{Width: 80, Height: 24}
	updated, cmd := m.Update(msg)

	assert.Equal(t, 80, updated.Width())
	assert.Equal(t, 24, updated.Height())
	assert.True(t, updated.Ready())
	assert.Nil(t, cmd)
}

func TestConfirmationModel_Update_WindowSizeMsg_Large(t *testing.T) {
	styles := getTestStyles()
	m := NewConfirmation(styles, "1.0.0", nil, DriverOption{}, nil)

	msg := tea.WindowSizeMsg{Width: 200, Height: 60}
	updated, _ := m.Update(msg)

	assert.Equal(t, 200, updated.Width())
	assert.Equal(t, 60, updated.Height())
	assert.True(t, updated.Ready())
}

func TestConfirmationModel_Update_WindowSizeMsg_Small(t *testing.T) {
	styles := getTestStyles()
	m := NewConfirmation(styles, "1.0.0", nil, DriverOption{}, nil)

	msg := tea.WindowSizeMsg{Width: 40, Height: 10}
	updated, _ := m.Update(msg)

	assert.Equal(t, 40, updated.Width())
	assert.Equal(t, 10, updated.Height())
	assert.True(t, updated.Ready())
}

// =============================================================================
// Update Tests - Button Navigation (Left/Right)
// =============================================================================

func TestConfirmationModel_Update_LeftRight_ButtonNavigation(t *testing.T) {
	styles := getTestStyles()
	m := NewConfirmation(styles, "1.0.0", nil, DriverOption{}, nil)
	m.SetSize(80, 24)

	assert.Equal(t, 0, m.FocusedButtonIndex()) // Install button

	// Move right to Go Back button
	msg := tea.KeyMsg{Type: tea.KeyRight}
	m, _ = m.Update(msg)
	assert.Equal(t, 1, m.FocusedButtonIndex())

	// Move left back to Install
	msg = tea.KeyMsg{Type: tea.KeyLeft}
	m, _ = m.Update(msg)
	assert.Equal(t, 0, m.FocusedButtonIndex())
}

func TestConfirmationModel_Update_HLKeys_ButtonNavigation(t *testing.T) {
	styles := getTestStyles()
	m := NewConfirmation(styles, "1.0.0", nil, DriverOption{}, nil)
	m.SetSize(80, 24)

	// Use l for right
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}}
	m, _ = m.Update(msg)
	assert.Equal(t, 1, m.FocusedButtonIndex())

	// Use h for left
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}}
	m, _ = m.Update(msg)
	assert.Equal(t, 0, m.FocusedButtonIndex())
}

func TestConfirmationModel_Update_TabKey_ButtonNavigation(t *testing.T) {
	styles := getTestStyles()
	m := NewConfirmation(styles, "1.0.0", nil, DriverOption{}, nil)
	m.SetSize(80, 24)

	// Tab should move right
	msg := tea.KeyMsg{Type: tea.KeyTab}
	m, _ = m.Update(msg)
	assert.Equal(t, 1, m.FocusedButtonIndex())
}

func TestConfirmationModel_Update_ButtonsWrap(t *testing.T) {
	styles := getTestStyles()
	m := NewConfirmation(styles, "1.0.0", nil, DriverOption{}, nil)
	m.SetSize(80, 24)

	// Move right twice (should wrap)
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRight})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRight})
	assert.Equal(t, 0, m.FocusedButtonIndex())

	// Move left (should wrap)
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyLeft})
	assert.Equal(t, 1, m.FocusedButtonIndex())
}

// =============================================================================
// Update Tests - Confirm on "Install" Triggers StartInstallationMsg
// =============================================================================

func TestConfirmationModel_Update_Confirm_Install_TriggersStartInstallationMsg(t *testing.T) {
	styles := getTestStyles()
	gpuInfo := createMockGPUInfo()
	driver := DriverOption{Version: "550", Branch: "Latest"}
	comps := []ComponentOption{{ID: "driver", Name: "NVIDIA Driver", Selected: true}}
	m := NewConfirmation(styles, "1.0.0", gpuInfo, driver, comps)
	m.SetSize(80, 24)

	assert.Equal(t, 0, m.FocusedButtonIndex()) // Install button

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	_, cmd := m.Update(msg)

	require.NotNil(t, cmd)

	result := cmd()
	startMsg, ok := result.(StartInstallationMsg)
	assert.True(t, ok, "Expected StartInstallationMsg")
	assert.Equal(t, gpuInfo, startMsg.GPUInfo)
	assert.Equal(t, driver, startMsg.Driver)
	assert.Equal(t, comps, startMsg.Components)
}

func TestConfirmationModel_Update_Confirm_Install_WithYKey(t *testing.T) {
	styles := getTestStyles()
	gpuInfo := createMockGPUInfo()
	driver := DriverOption{Version: "550", Branch: "Latest"}
	comps := []ComponentOption{{ID: "driver", Name: "NVIDIA Driver", Selected: true}}
	m := NewConfirmation(styles, "1.0.0", gpuInfo, driver, comps)
	m.SetSize(80, 24)

	// Press 'y' to confirm
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}}
	_, cmd := m.Update(msg)

	require.NotNil(t, cmd)

	result := cmd()
	_, ok := result.(StartInstallationMsg)
	assert.True(t, ok, "Expected StartInstallationMsg")
}

// =============================================================================
// Update Tests - Confirm on "Go Back" Triggers NavigateBackToSelectionMsg
// =============================================================================

func TestConfirmationModel_Update_Confirm_GoBack_TriggersNavigateBackToSelectionMsg(t *testing.T) {
	styles := getTestStyles()
	gpuInfo := createMockGPUInfo()
	m := NewConfirmation(styles, "1.0.0", gpuInfo, DriverOption{}, nil)
	m.SetSize(80, 24)

	// Move to Go Back button
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRight})
	assert.Equal(t, 1, m.FocusedButtonIndex())

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	_, cmd := m.Update(msg)

	require.NotNil(t, cmd)

	result := cmd()
	navMsg, ok := result.(NavigateBackToSelectionMsg)
	assert.True(t, ok, "Expected NavigateBackToSelectionMsg")
	assert.Equal(t, gpuInfo, navMsg.GPUInfo)
}

// =============================================================================
// Update Tests - Back Key Triggers NavigateBackToSelectionMsg
// =============================================================================

func TestConfirmationModel_Update_BackKey_Esc(t *testing.T) {
	styles := getTestStyles()
	gpuInfo := createMockGPUInfo()
	m := NewConfirmation(styles, "1.0.0", gpuInfo, DriverOption{}, nil)
	m.SetSize(80, 24)

	msg := tea.KeyMsg{Type: tea.KeyEscape}
	_, cmd := m.Update(msg)

	require.NotNil(t, cmd)

	result := cmd()
	_, ok := result.(NavigateBackToSelectionMsg)
	assert.True(t, ok, "Expected NavigateBackToSelectionMsg")
}

func TestConfirmationModel_Update_BackKey_N(t *testing.T) {
	styles := getTestStyles()
	gpuInfo := createMockGPUInfo()
	m := NewConfirmation(styles, "1.0.0", gpuInfo, DriverOption{}, nil)
	m.SetSize(80, 24)

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}}
	_, cmd := m.Update(msg)

	require.NotNil(t, cmd)

	result := cmd()
	_, ok := result.(NavigateBackToSelectionMsg)
	assert.True(t, ok, "Expected NavigateBackToSelectionMsg")
}

func TestConfirmationModel_Update_BackKey_Backspace(t *testing.T) {
	styles := getTestStyles()
	gpuInfo := createMockGPUInfo()
	m := NewConfirmation(styles, "1.0.0", gpuInfo, DriverOption{}, nil)
	m.SetSize(80, 24)

	msg := tea.KeyMsg{Type: tea.KeyBackspace}
	_, cmd := m.Update(msg)

	require.NotNil(t, cmd)

	result := cmd()
	_, ok := result.(NavigateBackToSelectionMsg)
	assert.True(t, ok, "Expected NavigateBackToSelectionMsg")
}

// =============================================================================
// Update Tests - Quit Key
// =============================================================================

func TestConfirmationModel_Update_QuitKey(t *testing.T) {
	styles := getTestStyles()
	m := NewConfirmation(styles, "1.0.0", nil, DriverOption{}, nil)
	m.SetSize(80, 24)

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
	_, cmd := m.Update(msg)

	require.NotNil(t, cmd, "q key should return quit command")
}

func TestConfirmationModel_Update_CtrlC(t *testing.T) {
	styles := getTestStyles()
	m := NewConfirmation(styles, "1.0.0", nil, DriverOption{}, nil)
	m.SetSize(80, 24)

	msg := tea.KeyMsg{Type: tea.KeyCtrlC}
	_, cmd := m.Update(msg)

	require.NotNil(t, cmd, "ctrl+c should return quit command")
}

// =============================================================================
// Update Tests - Help Key
// =============================================================================

func TestConfirmationModel_Update_HelpKey(t *testing.T) {
	styles := getTestStyles()
	m := NewConfirmation(styles, "1.0.0", nil, DriverOption{}, nil)
	m.SetSize(80, 24)

	assert.False(t, m.IsFullHelpShown())

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}}
	m, cmd := m.Update(msg)

	assert.True(t, m.IsFullHelpShown())
	assert.Nil(t, cmd)

	// Toggle off
	m, _ = m.Update(msg)
	assert.False(t, m.IsFullHelpShown())
}

// =============================================================================
// View Tests - Not Ready
// =============================================================================

func TestConfirmationModel_View_NotReady(t *testing.T) {
	styles := getTestStyles()
	m := NewConfirmation(styles, "1.0.0", nil, DriverOption{}, nil)

	view := m.View()

	assert.Equal(t, "Loading...", view)
}

// =============================================================================
// View Tests - Ready
// =============================================================================

func TestConfirmationModel_View_Ready(t *testing.T) {
	styles := getTestStyles()
	m := NewConfirmation(styles, "1.0.0", nil, DriverOption{}, nil)
	m.SetSize(100, 40)

	view := m.View()

	assert.NotEmpty(t, view)
	assert.NotEqual(t, "Loading...", view)
}

func TestConfirmationModel_View_ShowsInstallationSummary(t *testing.T) {
	styles := getTestStyles()
	m := NewConfirmation(styles, "1.0.0", nil, DriverOption{}, nil)
	m.SetSize(100, 40)

	view := m.View()

	assert.Contains(t, view, "Installation Summary")
}

func TestConfirmationModel_View_ShowsGPUSection(t *testing.T) {
	styles := getTestStyles()
	gpuInfo := createMockGPUInfoWithModel()
	m := NewConfirmation(styles, "1.0.0", gpuInfo, DriverOption{}, nil)
	m.SetSize(100, 40)

	view := m.View()

	assert.Contains(t, view, "Target GPU:")
	assert.Contains(t, view, "GeForce RTX 3080")
}

func TestConfirmationModel_View_ShowsUnknownGPU(t *testing.T) {
	styles := getTestStyles()
	m := NewConfirmation(styles, "1.0.0", nil, DriverOption{}, nil)
	m.SetSize(100, 40)

	view := m.View()

	assert.Contains(t, view, "Target GPU:")
	assert.Contains(t, view, "Unknown GPU")
}

func TestConfirmationModel_View_ShowsDriverSection(t *testing.T) {
	styles := getTestStyles()
	driver := DriverOption{Version: "550", Branch: "Latest"}
	m := NewConfirmation(styles, "1.0.0", nil, driver, nil)
	m.SetSize(100, 40)

	view := m.View()

	assert.Contains(t, view, "Driver Version:")
	assert.Contains(t, view, "550")
	assert.Contains(t, view, "Latest")
}

func TestConfirmationModel_View_ShowsComponentsSection(t *testing.T) {
	styles := getTestStyles()
	comps := []ComponentOption{
		{ID: "driver", Name: "NVIDIA Driver", Selected: true},
		{ID: "cuda", Name: "CUDA Toolkit", Selected: true},
	}
	m := NewConfirmation(styles, "1.0.0", nil, DriverOption{}, comps)
	m.SetSize(100, 40)

	view := m.View()

	assert.Contains(t, view, "Components to install:")
	assert.Contains(t, view, "NVIDIA Driver")
	assert.Contains(t, view, "CUDA Toolkit")
}

func TestConfirmationModel_View_ShowsComponentCheckmarks(t *testing.T) {
	styles := getTestStyles()
	comps := []ComponentOption{
		{ID: "driver", Name: "NVIDIA Driver", Selected: true},
	}
	m := NewConfirmation(styles, "1.0.0", nil, DriverOption{}, comps)
	m.SetSize(100, 40)

	view := m.View()

	// Checkmark character should be present
	assert.Contains(t, view, "\u2713")
}

func TestConfirmationModel_View_ShowsButtons(t *testing.T) {
	styles := getTestStyles()
	m := NewConfirmation(styles, "1.0.0", nil, DriverOption{}, nil)
	m.SetSize(100, 40)

	view := m.View()

	assert.Contains(t, view, "Install")
	assert.Contains(t, view, "Go Back")
}

func TestConfirmationModel_View_ShowsConfirmationMessage(t *testing.T) {
	styles := getTestStyles()
	m := NewConfirmation(styles, "1.0.0", nil, DriverOption{}, nil)
	m.SetSize(100, 40)

	view := m.View()

	assert.Contains(t, view, "Are you sure you want to proceed with the installation?")
}

// =============================================================================
// View Tests - Warnings
// =============================================================================

func TestConfirmationModel_View_ShowsWarningsWhenPresent(t *testing.T) {
	styles := getTestStyles()
	gpuInfo := &gpu.GPUInfo{
		NouveauStatus: &nouveau.Status{Loaded: true},
	}
	m := NewConfirmation(styles, "1.0.0", gpuInfo, DriverOption{}, nil)
	m.SetSize(100, 40)

	view := m.View()

	assert.Contains(t, view, "Warnings:")
	assert.Contains(t, view, "Nouveau")
	assert.Contains(t, view, "\u26A0") // Warning symbol
}

func TestConfirmationModel_View_HidesWarningsWhenEmpty(t *testing.T) {
	styles := getTestStyles()
	gpuInfo := &gpu.GPUInfo{
		NouveauStatus: &nouveau.Status{Loaded: false},
		KernelInfo:    &kernel.KernelInfo{HeadersInstalled: true, SecureBootEnabled: false},
	}
	m := NewConfirmation(styles, "1.0.0", gpuInfo, DriverOption{}, nil)
	m.SetSize(100, 40)

	view := m.View()

	assert.NotContains(t, view, "Warnings:")
}

func TestConfirmationModel_View_ShowsMultipleWarnings(t *testing.T) {
	styles := getTestStyles()
	gpuInfo := &gpu.GPUInfo{
		NouveauStatus: &nouveau.Status{Loaded: true},
		KernelInfo:    &kernel.KernelInfo{SecureBootEnabled: true, HeadersInstalled: false},
	}
	m := NewConfirmation(styles, "1.0.0", gpuInfo, DriverOption{}, nil)
	m.SetSize(100, 40)

	view := m.View()

	assert.Contains(t, view, "Nouveau")
	assert.Contains(t, view, "Secure Boot")
	assert.Contains(t, view, "Kernel headers")
}

// =============================================================================
// View Tests - Various Sizes
// =============================================================================

func TestConfirmationModel_View_VariousSizes(t *testing.T) {
	styles := getTestStyles()

	testSizes := []struct {
		name   string
		width  int
		height int
	}{
		{"small", 40, 15},
		{"medium", 80, 24},
		{"large", 120, 40},
		{"wide", 200, 20},
		{"tall", 60, 50},
	}

	for _, tc := range testSizes {
		t.Run(tc.name, func(t *testing.T) {
			m := NewConfirmation(styles, "1.0.0", nil, DriverOption{}, nil)
			m.SetSize(tc.width, tc.height)

			view := m.View()

			assert.NotEmpty(t, view)
			assert.NotEqual(t, "Loading...", view)
		})
	}
}

func TestConfirmationModel_View_VerySmallSize(t *testing.T) {
	styles := getTestStyles()
	m := NewConfirmation(styles, "1.0.0", nil, DriverOption{}, nil)
	m.SetSize(10, 5)

	// Should not panic
	view := m.View()
	assert.NotEmpty(t, view)
}

// =============================================================================
// Getter Tests
// =============================================================================

func TestConfirmationModel_Getters(t *testing.T) {
	styles := getTestStyles()
	gpuInfo := createMockGPUInfo()
	driver := DriverOption{Version: "550", Branch: "Latest"}
	comps := []ComponentOption{{ID: "driver", Name: "NVIDIA Driver", Selected: true}}
	m := NewConfirmation(styles, "2.0.0", gpuInfo, driver, comps)
	m.SetSize(100, 50)

	assert.Equal(t, 100, m.Width())
	assert.Equal(t, 50, m.Height())
	assert.True(t, m.Ready())
	assert.Equal(t, "2.0.0", m.Version())
	assert.Equal(t, gpuInfo, m.GPUInfo())
	assert.Equal(t, driver, m.SelectedDriver())
	assert.Equal(t, comps, m.SelectedComponents())
	assert.NotNil(t, m.KeyMap())
	assert.Equal(t, 0, m.FocusedButtonIndex())
}

func TestConfirmationModel_Warnings_Getter(t *testing.T) {
	styles := getTestStyles()
	gpuInfo := &gpu.GPUInfo{
		NouveauStatus: &nouveau.Status{Loaded: true},
	}
	m := NewConfirmation(styles, "1.0.0", gpuInfo, DriverOption{}, nil)

	warnings := m.Warnings()

	assert.NotEmpty(t, warnings)
	assert.Contains(t, warnings[0], "Nouveau")
}

// =============================================================================
// SetSize Tests
// =============================================================================

func TestConfirmationModel_SetSize(t *testing.T) {
	styles := getTestStyles()
	m := NewConfirmation(styles, "1.0.0", nil, DriverOption{}, nil)

	assert.False(t, m.Ready())

	m.SetSize(100, 50)

	assert.True(t, m.Ready())
	assert.Equal(t, 100, m.Width())
	assert.Equal(t, 50, m.Height())
}

func TestConfirmationModel_SetSize_Multiple(t *testing.T) {
	styles := getTestStyles()
	m := NewConfirmation(styles, "1.0.0", nil, DriverOption{}, nil)

	m.SetSize(80, 24)
	assert.Equal(t, 80, m.Width())
	assert.Equal(t, 24, m.Height())

	m.SetSize(120, 40)
	assert.Equal(t, 120, m.Width())
	assert.Equal(t, 40, m.Height())
}

// =============================================================================
// Message Type Tests
// =============================================================================

func TestStartInstallationMsg_Struct(t *testing.T) {
	gpuInfo := createMockGPUInfo()
	driver := DriverOption{Version: "550", Branch: "Latest"}
	comps := []ComponentOption{{ID: "driver", Selected: true}}

	msg := StartInstallationMsg{
		GPUInfo:    gpuInfo,
		Driver:     driver,
		Components: comps,
	}

	assert.Equal(t, gpuInfo, msg.GPUInfo)
	assert.Equal(t, "550", msg.Driver.Version)
	assert.Len(t, msg.Components, 1)
}

func TestNavigateBackToSelectionMsg_Struct(t *testing.T) {
	gpuInfo := createMockGPUInfo()

	msg := NavigateBackToSelectionMsg{
		GPUInfo: gpuInfo,
	}

	assert.Equal(t, gpuInfo, msg.GPUInfo)
}

// =============================================================================
// Integration Tests
// =============================================================================

func TestConfirmationModel_FullFlow_ConfirmInstall(t *testing.T) {
	styles := getTestStyles()
	gpuInfo := createMockGPUInfo()
	driver := DriverOption{Version: "550", Branch: "Latest"}
	comps := []ComponentOption{{ID: "driver", Name: "NVIDIA Driver", Selected: true}}
	m := NewConfirmation(styles, "1.0.0", gpuInfo, driver, comps)

	// Window resize
	m, _ = m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	assert.True(t, m.Ready())

	// View should render properly
	view := m.View()
	assert.Contains(t, view, "Installation Summary")
	assert.Contains(t, view, "550")
	assert.Contains(t, view, "NVIDIA Driver")

	// Press enter to confirm (Install button is focused by default)
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.NotNil(t, cmd)

	result := cmd()
	startMsg, ok := result.(StartInstallationMsg)
	assert.True(t, ok)
	assert.Equal(t, "550", startMsg.Driver.Version)
}

func TestConfirmationModel_FullFlow_GoBack(t *testing.T) {
	styles := getTestStyles()
	gpuInfo := createMockGPUInfo()
	m := NewConfirmation(styles, "1.0.0", gpuInfo, DriverOption{}, nil)
	m.SetSize(100, 40)

	// Navigate to Go Back button
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRight})
	assert.Equal(t, 1, m.FocusedButtonIndex())

	// Press enter
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.NotNil(t, cmd)

	result := cmd()
	_, ok := result.(NavigateBackToSelectionMsg)
	assert.True(t, ok)
}

func TestConfirmationModel_FullFlow_EscToBack(t *testing.T) {
	styles := getTestStyles()
	m := NewConfirmation(styles, "1.0.0", nil, DriverOption{}, nil)
	m.SetSize(100, 40)

	// Press escape to go back
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	require.NotNil(t, cmd)

	result := cmd()
	_, ok := result.(NavigateBackToSelectionMsg)
	assert.True(t, ok)
}

func TestConfirmationModel_FullFlow_HelpToggle(t *testing.T) {
	styles := getTestStyles()
	m := NewConfirmation(styles, "1.0.0", nil, DriverOption{}, nil)
	m.SetSize(100, 40)

	assert.False(t, m.IsFullHelpShown())

	// Toggle help on
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	assert.True(t, m.IsFullHelpShown())

	// View should still work
	view := m.View()
	assert.NotEmpty(t, view)

	// Toggle help off
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	assert.False(t, m.IsFullHelpShown())
}

// =============================================================================
// Edge Cases
// =============================================================================

func TestConfirmationModel_UnknownMessage(t *testing.T) {
	styles := getTestStyles()
	m := NewConfirmation(styles, "1.0.0", nil, DriverOption{}, nil)
	m.SetSize(80, 24)

	type customMsg struct{}

	updated, cmd := m.Update(customMsg{})

	// State should remain unchanged
	assert.True(t, updated.Ready())
	assert.Equal(t, 0, updated.FocusedButtonIndex())
	assert.Nil(t, cmd)
}

func TestConfirmationModel_MultipleSizeChanges(t *testing.T) {
	styles := getTestStyles()
	m := NewConfirmation(styles, "1.0.0", nil, DriverOption{}, nil)

	sizes := []tea.WindowSizeMsg{
		{Width: 80, Height: 24},
		{Width: 120, Height: 40},
		{Width: 40, Height: 15},
		{Width: 200, Height: 60},
		{Width: 80, Height: 24},
	}

	for _, size := range sizes {
		m, _ = m.Update(size)
		assert.Equal(t, size.Width, m.Width())
		assert.Equal(t, size.Height, m.Height())

		// View should still work
		view := m.View()
		assert.NotEmpty(t, view)
	}
}

func TestConfirmationModel_EmptyComponents(t *testing.T) {
	styles := getTestStyles()
	m := NewConfirmation(styles, "1.0.0", nil, DriverOption{}, []ComponentOption{})
	m.SetSize(100, 40)

	view := m.View()

	assert.Contains(t, view, "Components to install:")
	// No components listed
}

func TestConfirmationModel_NilComponents(t *testing.T) {
	styles := getTestStyles()
	m := NewConfirmation(styles, "1.0.0", nil, DriverOption{}, nil)
	m.SetSize(100, 40)

	// Should not panic
	view := m.View()
	assert.NotEmpty(t, view)
}

// =============================================================================
// GPU Name Rendering Tests
// =============================================================================

func TestConfirmationModel_GPUNameFromModel(t *testing.T) {
	styles := getTestStyles()
	gpuInfo := createMockGPUInfoWithModel()
	m := NewConfirmation(styles, "1.0.0", gpuInfo, DriverOption{}, nil)
	m.SetSize(100, 40)

	view := m.View()

	assert.Contains(t, view, "GeForce RTX 3080")
}

func TestConfirmationModel_GPUNameFallbackToName(t *testing.T) {
	styles := getTestStyles()
	gpuInfo := &gpu.GPUInfo{
		NVIDIAGPUs: []gpu.NVIDIAGPUInfo{
			{
				PCIDevice: pci.PCIDevice{
					Address:  "0000:01:00.0",
					VendorID: "10de",
					DeviceID: "2204",
				},
				Model: nil, // No model info
			},
		},
	}
	m := NewConfirmation(styles, "1.0.0", gpuInfo, DriverOption{}, nil)
	m.SetSize(100, 40)

	view := m.View()

	// Should use fallback name from NVIDIAGPUInfo.Name()
	assert.Contains(t, view, "Target GPU:")
}

func TestConfirmationModel_GPUNameUnknownWhenNoGPUs(t *testing.T) {
	styles := getTestStyles()
	gpuInfo := &gpu.GPUInfo{
		NVIDIAGPUs: []gpu.NVIDIAGPUInfo{}, // Empty
	}
	m := NewConfirmation(styles, "1.0.0", gpuInfo, DriverOption{}, nil)
	m.SetSize(100, 40)

	view := m.View()

	assert.Contains(t, view, "Unknown GPU")
}

// =============================================================================
// Helper Functions
// =============================================================================

// createMockGPUInfoWithModel creates a GPUInfo with model information.
func createMockGPUInfoWithModel() *gpu.GPUInfo {
	return &gpu.GPUInfo{
		NVIDIAGPUs: []gpu.NVIDIAGPUInfo{
			{
				PCIDevice: pci.PCIDevice{
					Address:  "0000:01:00.0",
					VendorID: "10de",
					DeviceID: "2204",
				},
				Model: &nvidia.GPUModel{
					Name:         "GeForce RTX 3080",
					Architecture: nvidia.ArchAmpere,
				},
			},
		},
	}
}
