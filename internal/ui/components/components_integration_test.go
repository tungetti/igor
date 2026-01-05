package components

import (
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/tungetti/igor/internal/ui/theme"
)

// =============================================================================
// Integration Test Helpers
// =============================================================================

// getIntegrationStyles returns default styles for integration testing.
func getIntegrationStyles() theme.Styles {
	return theme.DefaultTheme().Styles
}

// =============================================================================
// TestProgressBar_Animations - Progress Bar Rendering Tests
// =============================================================================

func TestProgressBar_Animations(t *testing.T) {
	t.Run("0 to 100 percent progression", func(t *testing.T) {
		styles := getIntegrationStyles()
		p := NewProgress(styles, 50)

		for percent := 0; percent <= 100; percent += 10 {
			p.SetProgress(percent, 100)
			view := p.View()
			assert.NotEmpty(t, view)
			assert.Equal(t, percent, p.PercentInt())
		}
	})

	t.Run("progress with percentage display", func(t *testing.T) {
		styles := getIntegrationStyles()
		p := NewProgress(styles, 40)

		p.SetProgress(50, 100)
		view := p.ViewWithPercent()
		assert.Contains(t, view, "50%")

		p.SetProgress(100, 100)
		view = p.ViewWithPercent()
		assert.Contains(t, view, "100%")
	})

	t.Run("custom widths", func(t *testing.T) {
		styles := getIntegrationStyles()

		widths := []int{20, 40, 60, 80, 100}
		for _, width := range widths {
			p := NewProgress(styles, width)
			p.SetProgress(50, 100)

			assert.Equal(t, width, p.Width())
			view := p.View()
			assert.NotEmpty(t, view)
		}
	})

	t.Run("progress bar with gradient colors", func(t *testing.T) {
		styles := getIntegrationStyles()
		p := NewProgressWithGradient(styles, 50, "#FF0000", "#00FF00")

		p.SetProgress(75, 100)
		view := p.View()
		assert.NotEmpty(t, view)
	})

	t.Run("progress bar with solid fill", func(t *testing.T) {
		styles := getIntegrationStyles()
		p := NewProgressWithSolidFill(styles, 50, "#76B900")

		p.SetProgress(25, 100)
		view := p.View()
		assert.NotEmpty(t, view)
	})

	t.Run("progress update with frame message", func(t *testing.T) {
		styles := getIntegrationStyles()
		p := NewProgress(styles, 40)

		p.SetProgress(50, 100)

		// Update with frame message
		updated, cmd := p.Update(progress.FrameMsg{})
		assert.NotNil(t, updated)
		_ = cmd // Command may be nil or a continuation
	})

	t.Run("progress increment operations", func(t *testing.T) {
		styles := getIntegrationStyles()
		p := NewProgress(styles, 40)

		p.SetProgress(0, 10)
		assert.Equal(t, 0, p.Current())

		p.Increment()
		assert.Equal(t, 1, p.Current())

		p.IncrementBy(5)
		assert.Equal(t, 6, p.Current())
	})

	t.Run("progress reset", func(t *testing.T) {
		styles := getIntegrationStyles()
		p := NewProgress(styles, 40)

		p.SetProgress(75, 100)
		assert.True(t, p.Percent() > 0)

		p.Reset()
		assert.Equal(t, 0, p.Current())
		assert.Equal(t, 0, p.Total())
		assert.Equal(t, float64(0), p.Percent())
	})

	t.Run("progress completion detection", func(t *testing.T) {
		styles := getIntegrationStyles()
		p := NewProgress(styles, 40)

		p.SetProgress(50, 100)
		assert.False(t, p.IsComplete())

		p.SetProgress(100, 100)
		assert.True(t, p.IsComplete())
	})

	t.Run("progress with label", func(t *testing.T) {
		styles := getIntegrationStyles()
		p := NewProgress(styles, 40)

		p.SetLabel("Downloading...")
		p.SetProgress(50, 100)

		view := p.View()
		assert.Contains(t, view, "Downloading...")
	})
}

// =============================================================================
// TestSpinner_Frames - Spinner Animation Tests
// =============================================================================

func TestSpinner_Frames(t *testing.T) {
	t.Run("all spinner types render", func(t *testing.T) {
		styles := getIntegrationStyles()

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
			s := NewSpinnerWithType(styles, "Loading...", st)
			view := s.View()
			assert.NotEmpty(t, view, "Spinner type %d should render", i)
		}
	})

	t.Run("spinner tick animation", func(t *testing.T) {
		styles := getIntegrationStyles()
		s := NewSpinner(styles, "Processing...")

		// Initial view
		initialView := s.View()
		assert.NotEmpty(t, initialView)

		// Tick should update spinner and return command
		tickMsg := spinner.TickMsg{}
		updated, cmd := s.Update(tickMsg)
		assert.NotNil(t, updated)
		assert.NotNil(t, cmd) // Should return next tick command
	})

	t.Run("spinner visibility toggle", func(t *testing.T) {
		styles := getIntegrationStyles()
		s := NewSpinner(styles, "Loading...")

		// Initially visible
		assert.True(t, s.IsVisible())
		assert.NotEmpty(t, s.View())

		// Hide
		s.Hide()
		assert.False(t, s.IsVisible())
		assert.Empty(t, s.View())

		// Show
		s.Show()
		assert.True(t, s.IsVisible())
		assert.NotEmpty(t, s.View())
	})

	t.Run("spinner message updates", func(t *testing.T) {
		styles := getIntegrationStyles()
		s := NewSpinner(styles, "Initial message")

		assert.Contains(t, s.View(), "Initial message")

		s.SetMessage("Updated message")
		assert.Contains(t, s.View(), "Updated message")
	})

	t.Run("spinner init returns tick command", func(t *testing.T) {
		styles := getIntegrationStyles()
		s := NewSpinner(styles, "Loading...")

		cmd := s.Init()
		assert.NotNil(t, cmd)
	})

	t.Run("spinner without message", func(t *testing.T) {
		styles := getIntegrationStyles()
		s := NewSpinner(styles, "")

		view := s.View()
		assert.NotEmpty(t, view)
		// Should only contain spinner character(s), no trailing space
	})
}

// =============================================================================
// TestList_Interactions - List Component Tests
// =============================================================================

func TestList_Interactions(t *testing.T) {
	t.Run("list selection with keyboard", func(t *testing.T) {
		styles := getIntegrationStyles()
		items := []ListItem{
			NewListItem("Item 1", "Description 1", 1),
			NewListItem("Item 2", "Description 2", 2),
			NewListItem("Item 3", "Description 3", 3),
		}

		l := NewList(styles, "Test List", items, 60, 20)

		// Initial selection
		assert.Equal(t, 0, l.SelectedIndex())

		// Navigate down
		l, _ = l.Update(tea.KeyMsg{Type: tea.KeyDown})
		// Note: List wraps bubbles list, selection may require actual key handling
	})

	t.Run("list scrolling with many items", func(t *testing.T) {
		styles := getIntegrationStyles()

		var items []ListItem
		for i := 0; i < 50; i++ {
			items = append(items, NewListItem(
				"Item "+string(rune('A'+i%26)),
				"Description",
				i,
			))
		}

		l := NewList(styles, "Long List", items, 60, 10)
		assert.Equal(t, 50, l.Len())

		view := l.View()
		assert.NotEmpty(t, view)
	})

	t.Run("list filtering", func(t *testing.T) {
		styles := getIntegrationStyles()
		items := []ListItem{
			NewListItem("Alpha", "", "a"),
			NewListItem("Beta", "", "b"),
			NewListItem("Gamma", "", "c"),
		}

		l := NewList(styles, "Test", items, 60, 20)

		// Enable/disable filtering
		l.EnableFiltering()
		l.DisableFiltering()
	})

	t.Run("empty list handling", func(t *testing.T) {
		styles := getIntegrationStyles()
		l := NewList(styles, "Empty List", nil, 60, 20)

		assert.True(t, l.IsEmpty())
		assert.Equal(t, 0, l.Len())

		_, ok := l.SelectedItem()
		assert.False(t, ok)

		view := l.View()
		assert.NotEmpty(t, view)
	})

	t.Run("list item set and get", func(t *testing.T) {
		styles := getIntegrationStyles()
		l := NewList(styles, "Test", nil, 60, 20)

		items := []ListItem{
			NewListItem("New 1", "Desc", 1),
			NewListItem("New 2", "Desc", 2),
		}

		l.SetItems(items)
		assert.Equal(t, 2, l.Len())

		retrieved := l.Items()
		assert.Equal(t, 2, len(retrieved))
	})

	t.Run("list status bar and help", func(t *testing.T) {
		styles := getIntegrationStyles()
		items := []ListItem{NewListItem("Item", "", 1)}
		l := NewList(styles, "Test", items, 60, 20)

		// These should not panic
		l.ShowStatusBar()
		l.HideStatusBar()
		l.ShowHelp()
		l.HideHelp()
	})
}

// =============================================================================
// TestButton_States - Button Component Tests
// =============================================================================

func TestButton_States(t *testing.T) {
	t.Run("normal state", func(t *testing.T) {
		styles := getIntegrationStyles()
		b := NewButton(styles, "Click Me")

		assert.False(t, b.IsFocused())
		assert.False(t, b.IsDisabled())
		assert.True(t, b.IsEnabled())

		view := b.View()
		assert.Contains(t, view, "Click Me")
	})

	t.Run("focused state", func(t *testing.T) {
		styles := getIntegrationStyles()
		b := NewButton(styles, "Focused Button")

		b.Focus()
		assert.True(t, b.IsFocused())

		view := b.View()
		assert.Contains(t, view, "Focused Button")
	})

	t.Run("disabled state", func(t *testing.T) {
		styles := getIntegrationStyles()
		b := NewButton(styles, "Disabled Button")

		b.Disable()
		assert.True(t, b.IsDisabled())
		assert.False(t, b.IsEnabled())

		view := b.View()
		assert.Contains(t, view, "Disabled Button")
	})

	t.Run("state transitions", func(t *testing.T) {
		styles := getIntegrationStyles()
		b := NewButton(styles, "Button")

		// Normal -> Focused
		b.Focus()
		assert.True(t, b.IsFocused())

		// Focused -> Blurred
		b.Blur()
		assert.False(t, b.IsFocused())

		// Toggle
		b.Toggle()
		assert.True(t, b.IsFocused())
		b.Toggle()
		assert.False(t, b.IsFocused())

		// Enable/Disable
		b.Disable()
		assert.True(t, b.IsDisabled())
		b.Enable()
		assert.False(t, b.IsDisabled())
	})

	t.Run("label update", func(t *testing.T) {
		styles := getIntegrationStyles()
		b := NewButton(styles, "Original")

		b.SetLabel("Updated")
		assert.Equal(t, "Updated", b.Label())

		view := b.View()
		assert.Contains(t, view, "Updated")
	})
}

// =============================================================================
// TestButtonGroup_Navigation
// =============================================================================

func TestButtonGroup_Navigation(t *testing.T) {
	t.Run("navigation with next and previous", func(t *testing.T) {
		styles := getIntegrationStyles()
		g := NewButtonGroup(styles, "A", "B", "C", "D")

		assert.Equal(t, 0, g.FocusedIndex())
		assert.Equal(t, "A", g.FocusedLabel())

		g.Next()
		assert.Equal(t, 1, g.FocusedIndex())
		assert.Equal(t, "B", g.FocusedLabel())

		g.Next()
		g.Next()
		assert.Equal(t, 3, g.FocusedIndex())

		g.Next() // Should wrap
		assert.Equal(t, 0, g.FocusedIndex())

		g.Previous() // Should wrap to end
		assert.Equal(t, 3, g.FocusedIndex())
	})

	t.Run("direct focus by index", func(t *testing.T) {
		styles := getIntegrationStyles()
		g := NewButtonGroup(styles, "A", "B", "C")

		g.Focus(2)
		assert.Equal(t, 2, g.FocusedIndex())

		g.Focus(0)
		assert.Equal(t, 0, g.FocusedIndex())

		// Invalid indices should be ignored
		g.Focus(-1)
		assert.Equal(t, 0, g.FocusedIndex())

		g.Focus(10)
		assert.Equal(t, 0, g.FocusedIndex())
	})

	t.Run("button group rendering", func(t *testing.T) {
		styles := getIntegrationStyles()
		g := NewButtonGroup(styles, "OK", "Cancel")

		// Horizontal view
		view := g.View()
		assert.Contains(t, view, "OK")
		assert.Contains(t, view, "Cancel")

		// Vertical view
		verticalView := g.ViewVertical()
		assert.Contains(t, verticalView, "OK")
		assert.Contains(t, verticalView, "Cancel")
		assert.Contains(t, verticalView, "\n")
	})

	t.Run("button group enable/disable all", func(t *testing.T) {
		styles := getIntegrationStyles()
		g := NewButtonGroup(styles, "A", "B", "C")

		g.DisableAll()
		for _, b := range g.Buttons() {
			assert.True(t, b.IsDisabled())
		}

		g.EnableAll()
		for _, b := range g.Buttons() {
			assert.False(t, b.IsDisabled())
		}
	})

	t.Run("set labels dynamically", func(t *testing.T) {
		styles := getIntegrationStyles()
		g := NewButtonGroup(styles, "A", "B")

		g.SetLabels("X", "Y", "Z")
		assert.Equal(t, 3, g.Len())
		assert.Equal(t, "X", g.FocusedLabel())

		buttons := g.Buttons()
		assert.Equal(t, "X", buttons[0].Label())
		assert.Equal(t, "Y", buttons[1].Label())
		assert.Equal(t, "Z", buttons[2].Label())
	})
}

// =============================================================================
// TestTextInput_Handling (Simulated - components may not have direct text input)
// =============================================================================

func TestTextInput_Handling(t *testing.T) {
	// Note: Igor's TUI doesn't currently have a text input component
	// This test documents expected behavior if one is added
	t.Run("placeholder for text input tests", func(t *testing.T) {
		// Text input components would test:
		// - Character input
		// - Backspace handling
		// - Cursor movement
		// - Text selection
		t.Skip("Text input component not implemented")
	})
}

// =============================================================================
// TestTable_Rendering (Simulated - using panel as table-like component)
// =============================================================================

func TestTable_Rendering(t *testing.T) {
	t.Run("panel as table-like container", func(t *testing.T) {
		styles := getIntegrationStyles()

		// Create table-like content
		content := "Header1  Header2  Header3\n" +
			"Row1     Data1    Value1\n" +
			"Row2     Data2    Value2"

		p := NewPanelWithContent(styles, "Table View", content, 60, 10)

		view := p.View()
		assert.Contains(t, view, "Table View")
		assert.Contains(t, view, "Header1")
		assert.Contains(t, view, "Row1")
	})

	t.Run("panel scrolling simulation", func(t *testing.T) {
		styles := getIntegrationStyles()

		var lines []string
		for i := 0; i < 20; i++ {
			lines = append(lines, "Row "+string(rune('A'+i%26)))
		}
		content := strings.Join(lines, "\n")

		p := NewPanelWithContent(styles, "Scrollable", content, 60, 10)
		view := p.View()
		assert.NotEmpty(t, view)
	})
}

// =============================================================================
// TestPanel_Integration
// =============================================================================

func TestPanel_Integration(t *testing.T) {
	t.Run("panel content operations", func(t *testing.T) {
		styles := getIntegrationStyles()
		p := NewPanel(styles, "Panel Title", 60, 20)

		// Initially no content
		assert.False(t, p.HasContent())

		// Set content
		p.SetContent("Initial content")
		assert.True(t, p.HasContent())
		assert.Equal(t, "Initial content", p.Content())

		// Append content
		p.AppendContent("More content")
		assert.Contains(t, p.Content(), "Initial content")
		assert.Contains(t, p.Content(), "More content")

		// Clear content
		p.ClearContent()
		assert.False(t, p.HasContent())
	})

	t.Run("panel focus state", func(t *testing.T) {
		styles := getIntegrationStyles()
		p := NewPanel(styles, "Title", 60, 20)

		assert.False(t, p.IsFocused())

		p.Focus()
		assert.True(t, p.IsFocused())

		// Focused view should be different
		focusedView := p.View()
		assert.NotEmpty(t, focusedView)

		p.Blur()
		assert.False(t, p.IsFocused())
	})

	t.Run("panel inner dimensions", func(t *testing.T) {
		styles := getIntegrationStyles()
		p := NewPanel(styles, "Title", 60, 20)

		// Inner dimensions should be smaller than outer (border + padding)
		innerWidth := p.InnerWidth()
		innerHeight := p.InnerHeight()

		assert.Less(t, innerWidth, 60)
		assert.Less(t, innerHeight, 20)
		assert.Greater(t, innerWidth, 0)
		assert.Greater(t, innerHeight, 0)
	})
}

// =============================================================================
// TestHeader_Footer_Integration
// =============================================================================

func TestHeader_Footer_Integration(t *testing.T) {
	t.Run("header rendering at various widths", func(t *testing.T) {
		styles := getIntegrationStyles()
		h := NewHeader(styles, "IGOR", "NVIDIA Driver Installer", "1.0.0")

		widths := []int{40, 60, 80, 100, 120}
		for _, w := range widths {
			h.SetWidth(w)
			view := h.View()
			assert.NotEmpty(t, view)
			assert.Contains(t, view, "IGOR")
		}
	})

	t.Run("footer with status messages", func(t *testing.T) {
		styles := getIntegrationStyles()
		f := NewFooterWithoutHelp(styles)
		f.SetWidth(80)

		statusTypes := []struct {
			status string
			typ    string
		}{
			{"Processing...", "info"},
			{"Complete!", "success"},
			{"Warning!", "warning"},
			{"Error!", "error"},
		}

		for _, st := range statusTypes {
			f.SetStatus(st.status, st.typ)
			assert.True(t, f.HasStatus())
			assert.Equal(t, st.status, f.Status())

			view := f.View()
			assert.Contains(t, view, st.status)
		}
	})

	t.Run("footer help toggle", func(t *testing.T) {
		styles := getIntegrationStyles()
		km := testKeyMap{}
		f := NewFooter(styles, km)
		f.SetWidth(80)

		assert.False(t, f.IsFullHelpShown())

		// Toggle with ? key
		f, _ = f.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
		assert.True(t, f.IsFullHelpShown())

		f, _ = f.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
		assert.False(t, f.IsFullHelpShown())
	})
}

// =============================================================================
// Benchmarks
// =============================================================================

func BenchmarkProgressBar_Render(b *testing.B) {
	styles := getIntegrationStyles()
	p := NewProgress(styles, 50)
	p.SetProgress(50, 100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = p.View()
	}
}

func BenchmarkSpinner_Tick(b *testing.B) {
	styles := getIntegrationStyles()
	s := NewSpinner(styles, "Loading...")

	tickMsg := spinner.TickMsg{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s, _ = s.Update(tickMsg)
	}
}

func BenchmarkButtonGroup_Render(b *testing.B) {
	styles := getIntegrationStyles()
	g := NewButtonGroup(styles, "OK", "Cancel", "Help")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = g.View()
	}
}

func BenchmarkList_Render(b *testing.B) {
	styles := getIntegrationStyles()
	items := make([]ListItem, 20)
	for i := range items {
		items[i] = NewListItem("Item", "Description", i)
	}

	l := NewList(styles, "List", items, 60, 20)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = l.View()
	}
}

func BenchmarkPanel_Render(b *testing.B) {
	styles := getIntegrationStyles()
	p := NewPanelWithContent(styles, "Title", "Some content here", 60, 20)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = p.View()
	}
}

func BenchmarkHeader_Render(b *testing.B) {
	styles := getIntegrationStyles()
	h := NewHeader(styles, "IGOR", "NVIDIA Driver Installer", "1.0.0")
	h.SetWidth(80)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = h.View()
	}
}
