package components

import (
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tungetti/igor/internal/ui/theme"
)

// Helper to get default styles for testing
func getTestStyles() theme.Styles {
	return theme.DefaultTheme().Styles
}

// ============================================================================
// Spinner Tests
// ============================================================================

func TestNewSpinner(t *testing.T) {
	styles := getTestStyles()

	tests := []struct {
		name    string
		message string
	}{
		{"with message", "Loading..."},
		{"empty message", ""},
		{"long message", "This is a very long loading message that spans multiple words"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewSpinner(styles, tt.message)
			assert.Equal(t, tt.message, s.Message())
			assert.True(t, s.IsVisible())
		})
	}
}

func TestNewSpinnerWithType(t *testing.T) {
	styles := getTestStyles()

	spinnerTypes := []spinner.Spinner{
		spinner.Dot,
		spinner.Line,
		spinner.MiniDot,
		spinner.Jump,
		spinner.Pulse,
		spinner.Points,
		spinner.Globe,
		spinner.Moon,
		spinner.Monkey,
	}

	for i, st := range spinnerTypes {
		s := NewSpinnerWithType(styles, "Test", st)
		assert.Equal(t, "Test", s.Message())
		assert.True(t, s.IsVisible(), "spinner type %d should be visible", i)
	}
}

func TestSpinnerVisibility(t *testing.T) {
	styles := getTestStyles()
	s := NewSpinner(styles, "Loading")

	// Initially visible
	assert.True(t, s.IsVisible())
	view := s.View()
	assert.NotEmpty(t, view)
	assert.Contains(t, view, "Loading")

	// Hide
	s.Hide()
	assert.False(t, s.IsVisible())
	assert.Empty(t, s.View())

	// Show again
	s.Show()
	assert.True(t, s.IsVisible())
	assert.NotEmpty(t, s.View())
}

func TestSpinnerSetMessage(t *testing.T) {
	styles := getTestStyles()
	s := NewSpinner(styles, "Initial")

	assert.Equal(t, "Initial", s.Message())

	s.SetMessage("Updated")
	assert.Equal(t, "Updated", s.Message())
	assert.Contains(t, s.View(), "Updated")

	s.SetMessage("")
	assert.Equal(t, "", s.Message())
}

func TestSpinnerInit(t *testing.T) {
	styles := getTestStyles()
	s := NewSpinner(styles, "Loading")

	cmd := s.Init()
	assert.NotNil(t, cmd, "Init should return a tick command")
}

func TestSpinnerUpdate(t *testing.T) {
	styles := getTestStyles()
	s := NewSpinner(styles, "Loading")

	// Update with a spinner tick message
	tickMsg := spinner.TickMsg{}
	updated, cmd := s.Update(tickMsg)

	// Should not panic and should return updated model
	assert.NotNil(t, updated)
	_ = cmd // cmd may or may not be nil depending on spinner state
}

func TestSpinnerTick(t *testing.T) {
	styles := getTestStyles()
	s := NewSpinner(styles, "Loading")

	cmd := s.Tick()
	assert.NotNil(t, cmd, "Tick should return a command")
}

func TestSpinnerViewWithoutMessage(t *testing.T) {
	styles := getTestStyles()
	s := NewSpinner(styles, "")

	view := s.View()
	// Should only show the spinner, no trailing space
	assert.NotEmpty(t, view)
}

// ============================================================================
// Progress Tests
// ============================================================================

func TestNewProgress(t *testing.T) {
	styles := getTestStyles()
	p := NewProgress(styles, 40)

	assert.Equal(t, 40, p.Width())
	assert.Equal(t, 0, p.Current())
	assert.Equal(t, 0, p.Total())
	assert.Equal(t, "", p.Label())
}

func TestNewProgressWithGradient(t *testing.T) {
	styles := getTestStyles()
	p := NewProgressWithGradient(styles, 50, "#FF0000", "#00FF00")

	assert.Equal(t, 50, p.Width())
}

func TestNewProgressWithSolidFill(t *testing.T) {
	styles := getTestStyles()
	p := NewProgressWithSolidFill(styles, 30, "#76B900")

	assert.Equal(t, 30, p.Width())
}

func TestProgressSetProgress(t *testing.T) {
	styles := getTestStyles()
	p := NewProgress(styles, 40)

	tests := []struct {
		name            string
		current         int
		total           int
		expectedPercent float64
	}{
		{"zero progress", 0, 100, 0.0},
		{"half progress", 50, 100, 0.5},
		{"full progress", 100, 100, 1.0},
		{"quarter progress", 25, 100, 0.25},
		{"zero total", 50, 0, 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := p.SetProgress(tt.current, tt.total)
			assert.Equal(t, tt.current, p.Current())
			assert.Equal(t, tt.total, p.Total())
			assert.InDelta(t, tt.expectedPercent, p.Percent(), 0.001)
			// SetProgress returns a command for animation
			_ = cmd
		})
	}
}

func TestProgressPercentInt(t *testing.T) {
	styles := getTestStyles()
	p := NewProgress(styles, 40)

	p.SetProgress(33, 100)
	assert.Equal(t, 33, p.PercentInt())

	p.SetProgress(66, 100)
	assert.Equal(t, 66, p.PercentInt())

	p.SetProgress(100, 100)
	assert.Equal(t, 100, p.PercentInt())
}

func TestProgressIsComplete(t *testing.T) {
	styles := getTestStyles()
	p := NewProgress(styles, 40)

	p.SetProgress(50, 100)
	assert.False(t, p.IsComplete())

	p.SetProgress(100, 100)
	assert.True(t, p.IsComplete())

	p.SetProgress(0, 0)
	assert.False(t, p.IsComplete())
}

func TestProgressLabel(t *testing.T) {
	styles := getTestStyles()
	p := NewProgress(styles, 40)

	assert.Equal(t, "", p.Label())

	p.SetLabel("Downloading...")
	assert.Equal(t, "Downloading...", p.Label())
	assert.Contains(t, p.View(), "Downloading...")
}

func TestProgressReset(t *testing.T) {
	styles := getTestStyles()
	p := NewProgress(styles, 40)

	p.SetProgress(50, 100)
	p.Reset()

	assert.Equal(t, 0, p.Current())
	assert.Equal(t, 0, p.Total())
	assert.Equal(t, float64(0), p.Percent())
}

func TestProgressIncrement(t *testing.T) {
	styles := getTestStyles()
	p := NewProgress(styles, 40)

	p.SetProgress(0, 10)
	p.Increment()
	assert.Equal(t, 1, p.Current())

	p.IncrementBy(5)
	assert.Equal(t, 6, p.Current())
}

func TestProgressSetWidth(t *testing.T) {
	styles := getTestStyles()
	p := NewProgress(styles, 40)

	p.SetWidth(60)
	assert.Equal(t, 60, p.Width())
}

func TestProgressView(t *testing.T) {
	styles := getTestStyles()
	p := NewProgress(styles, 40)

	p.SetProgress(50, 100)
	view := p.View()
	assert.NotEmpty(t, view)

	// With label
	p.SetLabel("Progress")
	view = p.View()
	assert.Contains(t, view, "Progress")
}

func TestProgressViewWithPercent(t *testing.T) {
	styles := getTestStyles()
	p := NewProgress(styles, 40)

	p.SetProgress(50, 100)
	view := p.ViewWithPercent()
	assert.Contains(t, view, "50%")
}

func TestProgressInit(t *testing.T) {
	styles := getTestStyles()
	p := NewProgress(styles, 40)

	cmd := p.Init()
	assert.Nil(t, cmd, "Progress Init should return nil")
}

func TestProgressUpdate(t *testing.T) {
	styles := getTestStyles()
	p := NewProgress(styles, 40)

	// Update with a progress frame message
	msg := progress.FrameMsg{}
	updated, _ := p.Update(msg)
	assert.NotNil(t, updated)

	// Update with unrelated message
	updated2, _ := p.Update(tea.KeyMsg{})
	assert.NotNil(t, updated2)
}

// ============================================================================
// List Tests
// ============================================================================

func TestNewListItem(t *testing.T) {
	item := NewListItem("Title", "Description", 42)

	assert.Equal(t, "Title", item.Title())
	assert.Equal(t, "Description", item.Description())
	assert.Equal(t, "Title", item.FilterValue())
	assert.Equal(t, 42, item.Value())
}

func TestNewSimpleListItem(t *testing.T) {
	item := NewSimpleListItem("Title", "value")

	assert.Equal(t, "Title", item.Title())
	assert.Equal(t, "", item.Description())
	assert.Equal(t, "value", item.Value())
}

func TestNewList(t *testing.T) {
	styles := getTestStyles()
	items := []ListItem{
		NewListItem("Item 1", "Desc 1", 1),
		NewListItem("Item 2", "Desc 2", 2),
		NewListItem("Item 3", "Desc 3", 3),
	}

	l := NewList(styles, "Test List", items, 40, 10)

	assert.Equal(t, "Test List", l.Title())
	assert.Equal(t, 40, l.Width())
	assert.Equal(t, 10, l.Height())
	assert.Equal(t, 3, l.Len())
	assert.False(t, l.IsEmpty())
}

func TestNewListEmpty(t *testing.T) {
	styles := getTestStyles()
	l := NewList(styles, "Empty List", nil, 40, 10)

	assert.Equal(t, 0, l.Len())
	assert.True(t, l.IsEmpty())
}

func TestListSelectedItem(t *testing.T) {
	styles := getTestStyles()
	items := []ListItem{
		NewListItem("Item 1", "Desc 1", 1),
		NewListItem("Item 2", "Desc 2", 2),
	}

	l := NewList(styles, "Test", items, 40, 10)

	item, ok := l.SelectedItem()
	assert.True(t, ok)
	assert.Equal(t, "Item 1", item.Title())
	assert.Equal(t, 0, l.SelectedIndex())
}

func TestListSelectedItemEmpty(t *testing.T) {
	styles := getTestStyles()
	l := NewList(styles, "Empty", nil, 40, 10)

	_, ok := l.SelectedItem()
	assert.False(t, ok)
}

func TestListSetItems(t *testing.T) {
	styles := getTestStyles()
	l := NewList(styles, "Test", nil, 40, 10)

	items := []ListItem{
		NewListItem("New 1", "", 1),
		NewListItem("New 2", "", 2),
	}

	l.SetItems(items)
	assert.Equal(t, 2, l.Len())

	retrievedItems := l.Items()
	assert.Equal(t, 2, len(retrievedItems))
	assert.Equal(t, "New 1", retrievedItems[0].Title())
}

func TestListSetSize(t *testing.T) {
	styles := getTestStyles()
	l := NewList(styles, "Test", nil, 40, 10)

	l.SetSize(60, 20)
	assert.Equal(t, 60, l.Width())
	assert.Equal(t, 20, l.Height())
}

func TestListSetTitle(t *testing.T) {
	styles := getTestStyles()
	l := NewList(styles, "Initial", nil, 40, 10)

	l.SetTitle("Updated")
	assert.Equal(t, "Updated", l.Title())
}

func TestListSelect(t *testing.T) {
	styles := getTestStyles()
	items := []ListItem{
		NewListItem("Item 1", "", 1),
		NewListItem("Item 2", "", 2),
		NewListItem("Item 3", "", 3),
	}

	l := NewList(styles, "Test", items, 40, 10)

	l.Select(2)
	assert.Equal(t, 2, l.SelectedIndex())
}

func TestListView(t *testing.T) {
	styles := getTestStyles()
	items := []ListItem{
		NewListItem("Item 1", "Desc 1", 1),
	}

	l := NewList(styles, "Test", items, 40, 10)
	view := l.View()
	assert.NotEmpty(t, view)
}

func TestListInit(t *testing.T) {
	styles := getTestStyles()
	l := NewList(styles, "Test", nil, 40, 10)

	cmd := l.Init()
	assert.Nil(t, cmd)
}

func TestListUpdate(t *testing.T) {
	styles := getTestStyles()
	items := []ListItem{
		NewListItem("Item 1", "", 1),
		NewListItem("Item 2", "", 2),
	}

	l := NewList(styles, "Test", items, 40, 10)

	updated, _ := l.Update(tea.KeyMsg{Type: tea.KeyDown})
	assert.NotNil(t, updated)
}

func TestListFilteringAndHelp(t *testing.T) {
	styles := getTestStyles()
	l := NewList(styles, "Test", nil, 40, 10)

	// These should not panic
	l.EnableFiltering()
	l.DisableFiltering()
	l.ShowStatusBar()
	l.HideStatusBar()
	l.ShowHelp()
	l.HideHelp()
}

// ============================================================================
// Button Tests
// ============================================================================

func TestNewButton(t *testing.T) {
	styles := getTestStyles()
	b := NewButton(styles, "Click Me")

	assert.Equal(t, "Click Me", b.Label())
	assert.False(t, b.IsFocused())
	assert.False(t, b.IsDisabled())
	assert.True(t, b.IsEnabled())
}

func TestButtonFocus(t *testing.T) {
	styles := getTestStyles()
	b := NewButton(styles, "Button")

	b.Focus()
	assert.True(t, b.IsFocused())

	b.Blur()
	assert.False(t, b.IsFocused())

	b.Toggle()
	assert.True(t, b.IsFocused())

	b.Toggle()
	assert.False(t, b.IsFocused())
}

func TestButtonDisable(t *testing.T) {
	styles := getTestStyles()
	b := NewButton(styles, "Button")

	b.Disable()
	assert.True(t, b.IsDisabled())
	assert.False(t, b.IsEnabled())

	b.Enable()
	assert.False(t, b.IsDisabled())
	assert.True(t, b.IsEnabled())
}

func TestButtonSetLabel(t *testing.T) {
	styles := getTestStyles()
	b := NewButton(styles, "Initial")

	b.SetLabel("Updated")
	assert.Equal(t, "Updated", b.Label())
}

func TestButtonView(t *testing.T) {
	styles := getTestStyles()
	b := NewButton(styles, "Button")

	// Normal state
	view := b.View()
	assert.Contains(t, view, "Button")

	// Focused state
	b.Focus()
	focusedView := b.View()
	assert.Contains(t, focusedView, "Button")

	// Disabled state
	b.Blur()
	b.Disable()
	disabledView := b.View()
	assert.Contains(t, disabledView, "Button")
}

// ============================================================================
// ButtonGroup Tests
// ============================================================================

func TestNewButtonGroup(t *testing.T) {
	styles := getTestStyles()
	g := NewButtonGroup(styles, "OK", "Cancel")

	assert.Equal(t, 2, g.Len())
	assert.False(t, g.IsEmpty())
	assert.Equal(t, 0, g.FocusedIndex())
	assert.Equal(t, "OK", g.FocusedLabel())
}

func TestNewButtonGroupEmpty(t *testing.T) {
	styles := getTestStyles()
	g := NewButtonGroup(styles)

	assert.Equal(t, 0, g.Len())
	assert.True(t, g.IsEmpty())
	assert.Equal(t, "", g.FocusedLabel())
}

func TestNewButtonGroupFromButtons(t *testing.T) {
	styles := getTestStyles()
	b1 := NewButton(styles, "One")
	b2 := NewButton(styles, "Two")

	g := NewButtonGroupFromButtons(styles, b1, b2)

	assert.Equal(t, 2, g.Len())
	assert.Equal(t, "One", g.FocusedLabel())
}

func TestButtonGroupNavigation(t *testing.T) {
	styles := getTestStyles()
	g := NewButtonGroup(styles, "A", "B", "C")

	assert.Equal(t, 0, g.FocusedIndex())

	g.Next()
	assert.Equal(t, 1, g.FocusedIndex())
	assert.Equal(t, "B", g.FocusedLabel())

	g.Next()
	assert.Equal(t, 2, g.FocusedIndex())

	g.Next() // Should wrap around
	assert.Equal(t, 0, g.FocusedIndex())

	g.Previous() // Should wrap to end
	assert.Equal(t, 2, g.FocusedIndex())

	g.Previous()
	assert.Equal(t, 1, g.FocusedIndex())
}

func TestButtonGroupNavigationEmpty(t *testing.T) {
	styles := getTestStyles()
	g := NewButtonGroup(styles)

	// Should not panic
	g.Next()
	g.Previous()
}

func TestButtonGroupFocus(t *testing.T) {
	styles := getTestStyles()
	g := NewButtonGroup(styles, "A", "B", "C")

	g.Focus(2)
	assert.Equal(t, 2, g.FocusedIndex())

	g.Focus(1)
	assert.Equal(t, 1, g.FocusedIndex())

	// Invalid index should be ignored
	g.Focus(-1)
	assert.Equal(t, 1, g.FocusedIndex())

	g.Focus(10)
	assert.Equal(t, 1, g.FocusedIndex())
}

func TestButtonGroupButton(t *testing.T) {
	styles := getTestStyles()
	g := NewButtonGroup(styles, "A", "B")

	b, ok := g.Button(0)
	assert.True(t, ok)
	assert.Equal(t, "A", b.Label())

	b, ok = g.Button(1)
	assert.True(t, ok)
	assert.Equal(t, "B", b.Label())

	_, ok = g.Button(-1)
	assert.False(t, ok)

	_, ok = g.Button(10)
	assert.False(t, ok)
}

func TestButtonGroupButtons(t *testing.T) {
	styles := getTestStyles()
	g := NewButtonGroup(styles, "A", "B", "C")

	buttons := g.Buttons()
	assert.Equal(t, 3, len(buttons))
	assert.Equal(t, "A", buttons[0].Label())
	assert.Equal(t, "B", buttons[1].Label())
	assert.Equal(t, "C", buttons[2].Label())
}

func TestButtonGroupFocusedButton(t *testing.T) {
	styles := getTestStyles()
	g := NewButtonGroup(styles, "A", "B")

	fb := g.FocusedButton()
	require.NotNil(t, fb)
	assert.Equal(t, "A", fb.Label())

	g.Next()
	fb = g.FocusedButton()
	assert.Equal(t, "B", fb.Label())
}

func TestButtonGroupDisableEnable(t *testing.T) {
	styles := getTestStyles()
	g := NewButtonGroup(styles, "A", "B")

	g.DisableAll()
	for _, b := range g.Buttons() {
		assert.True(t, b.IsDisabled())
	}

	g.EnableAll()
	for _, b := range g.Buttons() {
		assert.False(t, b.IsDisabled())
	}
}

func TestButtonGroupSetLabels(t *testing.T) {
	styles := getTestStyles()
	g := NewButtonGroup(styles, "A", "B")

	g.SetLabels("X", "Y", "Z")
	assert.Equal(t, 3, g.Len())
	assert.Equal(t, "X", g.FocusedLabel())

	g.SetLabels()
	assert.Equal(t, 0, g.Len())
}

func TestButtonGroupView(t *testing.T) {
	styles := getTestStyles()
	g := NewButtonGroup(styles, "OK", "Cancel")

	view := g.View()
	assert.Contains(t, view, "OK")
	assert.Contains(t, view, "Cancel")
}

func TestButtonGroupViewVertical(t *testing.T) {
	styles := getTestStyles()
	g := NewButtonGroup(styles, "OK", "Cancel")

	view := g.ViewVertical()
	assert.Contains(t, view, "OK")
	assert.Contains(t, view, "Cancel")
	assert.Contains(t, view, "\n")
}

// ============================================================================
// Panel Tests
// ============================================================================

func TestNewPanel(t *testing.T) {
	styles := getTestStyles()
	p := NewPanel(styles, "Title", 40, 10)

	assert.Equal(t, "Title", p.Title())
	assert.Equal(t, 40, p.Width())
	assert.Equal(t, 10, p.Height())
	assert.Equal(t, "", p.Content())
	assert.False(t, p.IsFocused())
}

func TestNewPanelWithContent(t *testing.T) {
	styles := getTestStyles()
	p := NewPanelWithContent(styles, "Title", "Content", 40, 10)

	assert.Equal(t, "Title", p.Title())
	assert.Equal(t, "Content", p.Content())
}

func TestPanelContent(t *testing.T) {
	styles := getTestStyles()
	p := NewPanel(styles, "Title", 40, 10)

	p.SetContent("New content")
	assert.Equal(t, "New content", p.Content())
	assert.True(t, p.HasContent())

	p.AppendContent("More content")
	assert.Contains(t, p.Content(), "New content")
	assert.Contains(t, p.Content(), "More content")

	p.ClearContent()
	assert.Equal(t, "", p.Content())
	assert.False(t, p.HasContent())
}

func TestPanelTitle(t *testing.T) {
	styles := getTestStyles()
	p := NewPanel(styles, "Title", 40, 10)

	assert.True(t, p.HasTitle())

	p.SetTitle("New Title")
	assert.Equal(t, "New Title", p.Title())

	p.SetTitle("")
	assert.False(t, p.HasTitle())
}

func TestPanelSize(t *testing.T) {
	styles := getTestStyles()
	p := NewPanel(styles, "Title", 40, 10)

	p.SetSize(60, 20)
	assert.Equal(t, 60, p.Width())
	assert.Equal(t, 20, p.Height())
}

func TestPanelFocus(t *testing.T) {
	styles := getTestStyles()
	p := NewPanel(styles, "Title", 40, 10)

	p.Focus()
	assert.True(t, p.IsFocused())

	p.Blur()
	assert.False(t, p.IsFocused())
}

func TestPanelView(t *testing.T) {
	styles := getTestStyles()
	p := NewPanelWithContent(styles, "Title", "Content", 40, 10)

	view := p.View()
	assert.Contains(t, view, "Title")
	assert.Contains(t, view, "Content")

	// Focused panel
	p.Focus()
	focusedView := p.View()
	assert.NotEmpty(t, focusedView)
}

func TestPanelViewWithoutBorder(t *testing.T) {
	styles := getTestStyles()
	p := NewPanelWithContent(styles, "Title", "Content", 40, 10)

	view := p.ViewWithoutBorder()
	assert.Contains(t, view, "Title")
	assert.Contains(t, view, "Content")
}

func TestPanelInnerDimensions(t *testing.T) {
	styles := getTestStyles()
	p := NewPanel(styles, "Title", 40, 20)

	innerWidth := p.InnerWidth()
	assert.True(t, innerWidth > 0 && innerWidth < 40)

	innerHeight := p.InnerHeight()
	assert.True(t, innerHeight > 0 && innerHeight < 20)
}

func TestPanelInnerDimensionsSmall(t *testing.T) {
	styles := getTestStyles()
	p := NewPanel(styles, "Title", 4, 2)

	// Should return 0 for very small dimensions
	innerWidth := p.InnerWidth()
	assert.GreaterOrEqual(t, innerWidth, 0)

	innerHeight := p.InnerHeight()
	assert.GreaterOrEqual(t, innerHeight, 0)
}

func TestPanelNoHeight(t *testing.T) {
	styles := getTestStyles()
	p := NewPanel(styles, "Title", 40, 0)

	// Should not set height constraint when height is 0
	view := p.View()
	assert.NotEmpty(t, view)
}

// ============================================================================
// Header Tests
// ============================================================================

func TestNewHeader(t *testing.T) {
	styles := getTestStyles()
	h := NewHeader(styles, "Igor", "NVIDIA Driver Installer", "1.0.0")

	assert.Equal(t, "Igor", h.Title())
	assert.Equal(t, "NVIDIA Driver Installer", h.Subtitle())
	assert.Equal(t, "1.0.0", h.Version())
	assert.True(t, h.HasSubtitle())
	assert.True(t, h.HasVersion())
}

func TestNewSimpleHeader(t *testing.T) {
	styles := getTestStyles()
	h := NewSimpleHeader(styles, "Igor")

	assert.Equal(t, "Igor", h.Title())
	assert.Equal(t, "", h.Subtitle())
	assert.Equal(t, "", h.Version())
	assert.False(t, h.HasSubtitle())
	assert.False(t, h.HasVersion())
}

func TestHeaderSetters(t *testing.T) {
	styles := getTestStyles()
	h := NewSimpleHeader(styles, "Initial")

	h.SetTitle("New Title")
	assert.Equal(t, "New Title", h.Title())

	h.SetSubtitle("Subtitle")
	assert.Equal(t, "Subtitle", h.Subtitle())
	assert.True(t, h.HasSubtitle())

	h.SetVersion("2.0.0")
	assert.Equal(t, "2.0.0", h.Version())
	assert.True(t, h.HasVersion())
}

func TestHeaderWidth(t *testing.T) {
	styles := getTestStyles()
	h := NewHeader(styles, "Igor", "Subtitle", "1.0.0")

	h.SetWidth(80)
	assert.Equal(t, 80, h.Width())
}

func TestHeaderView(t *testing.T) {
	styles := getTestStyles()
	h := NewHeader(styles, "Igor", "NVIDIA Driver Installer", "1.0.0")
	h.SetWidth(80)

	view := h.View()
	assert.Contains(t, view, "Igor")
	assert.Contains(t, view, "v1.0.0")
}

func TestHeaderViewNoWidth(t *testing.T) {
	styles := getTestStyles()
	h := NewHeader(styles, "Igor", "Subtitle", "1.0.0")

	// Should render minimal header without width set
	view := h.View()
	assert.Contains(t, view, "Igor")
}

func TestHeaderViewCentered(t *testing.T) {
	styles := getTestStyles()
	h := NewHeader(styles, "Igor", "Subtitle", "1.0.0")
	h.SetWidth(80)

	view := h.ViewCentered()
	assert.Contains(t, view, "Igor")
}

func TestHeaderViewVariousWidths(t *testing.T) {
	styles := getTestStyles()
	h := NewHeader(styles, "Igor", "NVIDIA Driver Installer", "1.0.0")

	widths := []int{20, 40, 60, 80, 100, 120}
	for _, w := range widths {
		h.SetWidth(w)
		view := h.View()
		assert.NotEmpty(t, view, "Header should render at width %d", w)
	}
}

func TestHeaderViewNarrow(t *testing.T) {
	styles := getTestStyles()
	h := NewHeader(styles, "Igor", "NVIDIA Driver Installer", "1.0.0")
	h.SetWidth(10) // Very narrow

	// Should still render without panic
	view := h.View()
	assert.NotEmpty(t, view)
}

// ============================================================================
// Footer Tests
// ============================================================================

// testKeyMap implements help.KeyMap for testing
type testKeyMap struct{}

func (k testKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{
		key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "quit")),
		key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
	}
}

func (k testKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{
			key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "quit")),
			key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
		},
	}
}

func TestNewFooter(t *testing.T) {
	styles := getTestStyles()
	km := testKeyMap{}
	f := NewFooter(styles, km)

	assert.True(t, f.IsHelpShown())
	assert.False(t, f.IsFullHelpShown())
	assert.False(t, f.HasStatus())
}

func TestNewFooterWithoutHelp(t *testing.T) {
	styles := getTestStyles()
	f := NewFooterWithoutHelp(styles)

	assert.False(t, f.IsHelpShown())
}

func TestFooterStatus(t *testing.T) {
	styles := getTestStyles()
	f := NewFooterWithoutHelp(styles)

	tests := []struct {
		name       string
		status     string
		statusType string
	}{
		{"info", "Processing...", "info"},
		{"success", "Complete!", "success"},
		{"warning", "Low disk space", "warning"},
		{"error", "Failed!", "error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f.SetStatus(tt.status, tt.statusType)
			assert.Equal(t, tt.status, f.Status())
			assert.Equal(t, tt.statusType, f.StatusType())
			assert.True(t, f.HasStatus())
		})
	}
}

func TestFooterStatusShortcuts(t *testing.T) {
	styles := getTestStyles()
	f := NewFooterWithoutHelp(styles)

	f.SetInfoStatus("Info")
	assert.Equal(t, "info", f.StatusType())

	f.SetSuccessStatus("Success")
	assert.Equal(t, "success", f.StatusType())

	f.SetWarningStatus("Warning")
	assert.Equal(t, "warning", f.StatusType())

	f.SetErrorStatus("Error")
	assert.Equal(t, "error", f.StatusType())
}

func TestFooterClearStatus(t *testing.T) {
	styles := getTestStyles()
	f := NewFooterWithoutHelp(styles)

	f.SetStatus("Status", "info")
	assert.True(t, f.HasStatus())

	f.ClearStatus()
	assert.False(t, f.HasStatus())
	assert.Equal(t, "", f.Status())
	assert.Equal(t, "", f.StatusType())
}

func TestFooterWidth(t *testing.T) {
	styles := getTestStyles()
	f := NewFooterWithoutHelp(styles)

	f.SetWidth(80)
	assert.Equal(t, 80, f.Width())
}

func TestFooterShowHelp(t *testing.T) {
	styles := getTestStyles()
	km := testKeyMap{}
	f := NewFooter(styles, km)

	f.ShowHelp(false)
	assert.False(t, f.IsHelpShown())

	f.ShowHelp(true)
	assert.True(t, f.IsHelpShown())
}

func TestFooterToggleFullHelp(t *testing.T) {
	styles := getTestStyles()
	km := testKeyMap{}
	f := NewFooter(styles, km)

	assert.False(t, f.IsFullHelpShown())

	f.ToggleFullHelp()
	assert.True(t, f.IsFullHelpShown())

	f.ToggleFullHelp()
	assert.False(t, f.IsFullHelpShown())
}

func TestFooterSetShowAll(t *testing.T) {
	styles := getTestStyles()
	km := testKeyMap{}
	f := NewFooter(styles, km)

	f.SetShowAll(true)
	assert.True(t, f.IsFullHelpShown())

	f.SetShowAll(false)
	assert.False(t, f.IsFullHelpShown())
}

func TestFooterSetKeyMap(t *testing.T) {
	styles := getTestStyles()
	f := NewFooterWithoutHelp(styles)

	km := testKeyMap{}
	f.SetKeyMap(km)
	f.ShowHelp(true)

	// Should not panic and should render help
	view := f.View()
	assert.NotEmpty(t, view)
}

func TestFooterView(t *testing.T) {
	styles := getTestStyles()
	km := testKeyMap{}
	f := NewFooter(styles, km)
	f.SetWidth(80)

	view := f.View()
	assert.NotEmpty(t, view)
}

func TestFooterViewWithStatus(t *testing.T) {
	styles := getTestStyles()
	km := testKeyMap{}
	f := NewFooter(styles, km)
	f.SetWidth(80)
	f.SetStatus("Processing...", "info")

	view := f.View()
	assert.Contains(t, view, "Processing...")
	assert.Contains(t, view, "●")
}

func TestFooterViewStatusOnly(t *testing.T) {
	styles := getTestStyles()
	f := NewFooterWithoutHelp(styles)
	f.SetWidth(80)

	// No status - should return empty
	view := f.ViewStatusOnly()
	assert.Empty(t, view)

	// With status
	f.SetStatus("Status", "success")
	view = f.ViewStatusOnly()
	assert.Contains(t, view, "Status")
}

func TestFooterViewHelpOnly(t *testing.T) {
	styles := getTestStyles()
	km := testKeyMap{}
	f := NewFooter(styles, km)
	f.SetWidth(80)

	view := f.ViewHelpOnly()
	assert.NotEmpty(t, view)

	// Without help
	f.ShowHelp(false)
	view = f.ViewHelpOnly()
	assert.Empty(t, view)
}

func TestFooterInit(t *testing.T) {
	styles := getTestStyles()
	km := testKeyMap{}
	f := NewFooter(styles, km)

	cmd := f.Init()
	assert.Nil(t, cmd)
}

func TestFooterUpdate(t *testing.T) {
	styles := getTestStyles()
	km := testKeyMap{}
	f := NewFooter(styles, km)

	// Update with ? key to toggle help
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}}
	updated, _ := f.Update(msg)
	assert.True(t, updated.IsFullHelpShown())

	// Toggle again
	updated, _ = updated.Update(msg)
	assert.False(t, updated.IsFullHelpShown())
}

func TestFooterUpdateUnrelatedKey(t *testing.T) {
	styles := getTestStyles()
	km := testKeyMap{}
	f := NewFooter(styles, km)

	initialState := f.IsFullHelpShown()

	// Update with unrelated key
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}}
	updated, _ := f.Update(msg)

	// State should not change
	assert.Equal(t, initialState, updated.IsFullHelpShown())
}

// ============================================================================
// Edge Cases and Integration Tests
// ============================================================================

func TestComponentsWithNilStyles(t *testing.T) {
	// This test verifies components don't panic with zero-value styles
	var styles theme.Styles

	// These should not panic
	_ = NewSpinner(styles, "Test")
	_ = NewProgress(styles, 40)
	_ = NewList(styles, "Test", nil, 40, 10)
	_ = NewButton(styles, "Test")
	_ = NewPanel(styles, "Test", 40, 10)
	_ = NewHeader(styles, "Test", "", "")
	_ = NewFooterWithoutHelp(styles)
}

func TestComponentsWithEmptyStrings(t *testing.T) {
	styles := getTestStyles()

	// Spinner with empty message
	s := NewSpinner(styles, "")
	assert.NotEmpty(t, s.View())

	// List with empty title
	l := NewList(styles, "", nil, 40, 10)
	_ = l.View()

	// Button with empty label
	b := NewButton(styles, "")
	_ = b.View()

	// Panel with empty title and content
	p := NewPanel(styles, "", 40, 10)
	_ = p.View()

	// Header with empty values
	h := NewHeader(styles, "", "", "")
	_ = h.View()
}

func TestComponentsWithUnicodeContent(t *testing.T) {
	styles := getTestStyles()

	// Test with Unicode characters
	unicodeContent := "🚀 Installing NVIDIA Driver 日本語 中文 한국어"

	s := NewSpinner(styles, unicodeContent)
	view := s.View()
	assert.Contains(t, view, "🚀")

	p := NewPanelWithContent(styles, "Unicode Test 🎮", unicodeContent, 80, 10)
	view = p.View()
	assert.Contains(t, view, "🎮")

	h := NewHeader(styles, "Igor 🖥️", "NVIDIA ⚡", "1.0")
	h.SetWidth(80)
	view = h.View()
	assert.Contains(t, view, "🖥️")
}

func TestListItemWithNilValue(t *testing.T) {
	item := NewListItem("Title", "Description", nil)
	assert.Nil(t, item.Value())
}

func TestButtonGroupWithSingleButton(t *testing.T) {
	styles := getTestStyles()
	g := NewButtonGroup(styles, "Only")

	assert.Equal(t, 1, g.Len())
	assert.Equal(t, "Only", g.FocusedLabel())

	// Navigation should stay on the same button
	g.Next()
	assert.Equal(t, 0, g.FocusedIndex())

	g.Previous()
	assert.Equal(t, 0, g.FocusedIndex())
}

func TestProgressWithLargeValues(t *testing.T) {
	styles := getTestStyles()
	p := NewProgress(styles, 40)

	// Test with large values
	p.SetProgress(999999999, 1000000000)
	assert.InDelta(t, 0.999999999, p.Percent(), 0.0001)

	// Test with equal current and total
	p.SetProgress(1000000000, 1000000000)
	assert.True(t, p.IsComplete())
}

func TestPanelAppendToEmptyContent(t *testing.T) {
	styles := getTestStyles()
	p := NewPanel(styles, "Title", 40, 10)

	p.AppendContent("First line")
	assert.Equal(t, "First line", p.Content())

	p.AppendContent("Second line")
	assert.True(t, strings.Contains(p.Content(), "First line"))
	assert.True(t, strings.Contains(p.Content(), "Second line"))
}
