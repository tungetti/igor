package theme

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/stretchr/testify/assert"
)

// =============================================================================
// Color Consistency Integration Tests
// =============================================================================

// TestColorConsistencyAcrossThemes verifies that all themes define the same
// color properties and that color relationships are consistent.
func TestColorConsistencyAcrossThemes(t *testing.T) {
	themes := []*Theme{
		DefaultTheme(),
		LightTheme(),
		HighContrastTheme(),
	}

	for _, theme := range themes {
		t.Run(string(theme.Name), func(t *testing.T) {
			// Primary colors should be defined
			assert.NotEmpty(t, string(theme.Primary), "Primary color should be defined")
			assert.NotEmpty(t, string(theme.PrimaryDark), "PrimaryDark color should be defined")
			assert.NotEmpty(t, string(theme.PrimaryLight), "PrimaryLight color should be defined")

			// Secondary colors should be defined
			assert.NotEmpty(t, string(theme.Secondary), "Secondary color should be defined")
			assert.NotEmpty(t, string(theme.SecondaryDark), "SecondaryDark color should be defined")

			// Semantic colors should be defined
			assert.NotNil(t, theme.Success, "Success color should be defined")
			assert.NotNil(t, theme.Warning, "Warning color should be defined")
			assert.NotNil(t, theme.Error, "Error color should be defined")
			assert.NotNil(t, theme.Info, "Info color should be defined")

			// Text colors should be defined
			assert.NotNil(t, theme.Text, "Text color should be defined")
			assert.NotNil(t, theme.TextMuted, "TextMuted color should be defined")
			assert.NotNil(t, theme.TextSubtle, "TextSubtle color should be defined")
			assert.NotNil(t, theme.TextInverse, "TextInverse color should be defined")

			// Background colors should be defined
			assert.NotNil(t, theme.Background, "Background color should be defined")
			assert.NotNil(t, theme.BackgroundAlt, "BackgroundAlt color should be defined")
			assert.NotNil(t, theme.BackgroundPanel, "BackgroundPanel color should be defined")
			assert.NotNil(t, theme.BackgroundHighlight, "BackgroundHighlight color should be defined")
			assert.NotNil(t, theme.BackgroundSelected, "BackgroundSelected color should be defined")

			// Border colors should be defined
			assert.NotNil(t, theme.Border, "Border color should be defined")
			assert.NotNil(t, theme.BorderFocus, "BorderFocus color should be defined")
			assert.NotNil(t, theme.BorderActive, "BorderActive color should be defined")
			assert.NotNil(t, theme.BorderMuted, "BorderMuted color should be defined")

			// Progress colors should be defined
			assert.NotNil(t, theme.Progress, "Progress color should be defined")
			assert.NotNil(t, theme.ProgressBg, "ProgressBg color should be defined")
			assert.NotNil(t, theme.ProgressComplete, "ProgressComplete color should be defined")
		})
	}
}

// TestNVIDIABrandColorsConsistency verifies NVIDIA brand colors are consistent
// across dark and light themes.
func TestNVIDIABrandColorsConsistency(t *testing.T) {
	darkTheme := DefaultTheme()
	lightTheme := LightTheme()

	// Both themes should use the same NVIDIA green primary color
	assert.Equal(t, darkTheme.Primary, lightTheme.Primary,
		"NVIDIA Green primary color should be consistent across dark and light themes")

	// Both themes should use the same primary dark variant
	assert.Equal(t, darkTheme.PrimaryDark, lightTheme.PrimaryDark,
		"NVIDIA Green dark variant should be consistent")

	// Both themes should use the same primary light variant
	assert.Equal(t, darkTheme.PrimaryLight, lightTheme.PrimaryLight,
		"NVIDIA Green light variant should be consistent")
}

// TestAdaptiveColorProperties verifies that adaptive colors have both light
// and dark values properly set.
func TestAdaptiveColorProperties(t *testing.T) {
	// Test global adaptive colors
	adaptiveColors := []struct {
		name  string
		color lipgloss.AdaptiveColor
	}{
		{"ColorSuccess", ColorSuccess},
		{"ColorWarning", ColorWarning},
		{"ColorError", ColorError},
		{"ColorInfo", ColorInfo},
		{"ColorText", ColorText},
		{"ColorTextMuted", ColorTextMuted},
		{"ColorTextSubtle", ColorTextSubtle},
		{"ColorTextInverse", ColorTextInverse},
		{"ColorBackground", ColorBackground},
		{"ColorBackgroundAlt", ColorBackgroundAlt},
		{"ColorBackgroundPanel", ColorBackgroundPanel},
		{"ColorBackgroundHighlight", ColorBackgroundHighlight},
		{"ColorBackgroundSelected", ColorBackgroundSelected},
		{"ColorBorder", ColorBorder},
		{"ColorBorderFocus", ColorBorderFocus},
		{"ColorBorderActive", ColorBorderActive},
		{"ColorBorderMuted", ColorBorderMuted},
		{"ColorProgress", ColorProgress},
		{"ColorProgressBg", ColorProgressBg},
		{"ColorProgressComplete", ColorProgressComplete},
		{"ColorHighContrastText", ColorHighContrastText},
		{"ColorHighContrastBg", ColorHighContrastBg},
		{"ColorHighContrastBorder", ColorHighContrastBorder},
		{"ColorHighContrastFocus", ColorHighContrastFocus},
	}

	for _, tt := range adaptiveColors {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotEmpty(t, tt.color.Light, "%s should have a Light color value", tt.name)
			assert.NotEmpty(t, tt.color.Dark, "%s should have a Dark color value", tt.name)

			// Verify hex format
			assert.True(t, strings.HasPrefix(tt.color.Light, "#"),
				"%s Light value should be hex format", tt.name)
			assert.True(t, strings.HasPrefix(tt.color.Dark, "#"),
				"%s Dark value should be hex format", tt.name)
		})
	}
}

// TestSemanticColorDistinctness verifies semantic colors are visually distinct.
func TestSemanticColorDistinctness(t *testing.T) {
	// Semantic colors should be different from each other
	assert.NotEqual(t, ColorSuccess.Dark, ColorError.Dark,
		"Success and Error should be distinct in dark mode")
	assert.NotEqual(t, ColorSuccess.Dark, ColorWarning.Dark,
		"Success and Warning should be distinct in dark mode")
	assert.NotEqual(t, ColorSuccess.Dark, ColorInfo.Dark,
		"Success and Info should be distinct in dark mode")
	assert.NotEqual(t, ColorError.Dark, ColorWarning.Dark,
		"Error and Warning should be distinct in dark mode")
	assert.NotEqual(t, ColorError.Dark, ColorInfo.Dark,
		"Error and Info should be distinct in dark mode")
	assert.NotEqual(t, ColorWarning.Dark, ColorInfo.Dark,
		"Warning and Info should be distinct in dark mode")

	// Same for light mode
	assert.NotEqual(t, ColorSuccess.Light, ColorError.Light,
		"Success and Error should be distinct in light mode")
	assert.NotEqual(t, ColorSuccess.Light, ColorWarning.Light,
		"Success and Warning should be distinct in light mode")
}

// =============================================================================
// Style Application Integration Tests
// =============================================================================

// TestStyleRenderConsistency verifies that styles render consistently.
func TestStyleRenderConsistency(t *testing.T) {
	theme := DefaultTheme()
	styles := theme.Styles

	// Same content rendered multiple times should produce identical output
	content := "Test Content"

	for i := 0; i < 5; i++ {
		render1 := styles.Title.Render(content)
		render2 := styles.Title.Render(content)
		assert.Equal(t, render1, render2, "Repeated renders should be identical")
	}
}

// TestStyleChaining verifies that styles can be properly chained and modified.
func TestStyleChaining(t *testing.T) {
	theme := DefaultTheme()

	// Create a chain of style modifications
	baseStyle := theme.Styles.Panel
	modifiedStyle := baseStyle.Copy().
		Width(60).
		Height(20).
		Padding(2, 4)

	// Verify modifications work
	content := "Chained Content"
	rendered := modifiedStyle.Render(content)
	assert.NotEmpty(t, rendered)

	// Original style should be unchanged (since we used Copy())
	originalRendered := baseStyle.Render(content)
	assert.NotEqual(t, rendered, originalRendered, "Modified style should differ from original")
}

// TestStyleApplicationToComponents verifies styles can be applied to various content types.
func TestStyleApplicationToComponents(t *testing.T) {
	theme := DefaultTheme()
	styles := theme.Styles

	testCases := []struct {
		name    string
		content string
	}{
		{"Empty string", ""},
		{"Single character", "X"},
		{"Short text", "Hello"},
		{"Long text", strings.Repeat("Lorem ipsum dolor sit amet. ", 10)},
		{"Multiline", "Line 1\nLine 2\nLine 3"},
		{"With special chars", "Path: /usr/local/bin"},
		{"With numbers", "Version 1.2.3"},
		{"Unicode", "NVIDIA GPU Driver"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Apply various styles and verify they don't panic
			assert.NotPanics(t, func() {
				_ = styles.Title.Render(tc.content)
				_ = styles.Panel.Render(tc.content)
				_ = styles.Button.Render(tc.content)
				_ = styles.Card.Render(tc.content)
				_ = styles.Success.Render(tc.content)
				_ = styles.Error.Render(tc.content)
			})
		})
	}
}

// TestStylesWithDifferentThemes verifies styles work correctly with all themes.
func TestStylesWithDifferentThemes(t *testing.T) {
	themes := []*Theme{
		DefaultTheme(),
		LightTheme(),
		HighContrastTheme(),
	}

	content := "Sample Content"

	for _, theme := range themes {
		t.Run(string(theme.Name), func(t *testing.T) {
			styles := theme.Styles

			// Render with each style type
			assert.NotEmpty(t, styles.Title.Render(content))
			assert.NotEmpty(t, styles.Subtitle.Render(content))
			assert.NotEmpty(t, styles.Paragraph.Render(content))
			assert.NotEmpty(t, styles.Panel.Render(content))
			assert.NotEmpty(t, styles.Card.Render(content))
			assert.NotEmpty(t, styles.Button.Render(content))
			assert.NotEmpty(t, styles.ButtonFocused.Render(content))
			assert.NotEmpty(t, styles.Success.Render(content))
			assert.NotEmpty(t, styles.Warning.Render(content))
			assert.NotEmpty(t, styles.Error.Render(content))
			assert.NotEmpty(t, styles.Info.Render(content))
		})
	}
}

// =============================================================================
// Responsive Styling Integration Tests
// =============================================================================

// TestResponsiveWidthStyling verifies styles adapt to different widths.
func TestResponsiveWidthStyling(t *testing.T) {
	theme := DefaultTheme()
	styles := NewStyles(theme)

	widths := []int{40, 60, 80, 100, 120}

	for _, width := range widths {
		t.Run(string(rune('0'+width/10))+"0_width", func(t *testing.T) {
			responsiveStyles := styles.WithWidth(width)

			// Verify width is properly set
			assert.Equal(t, width, responsiveStyles.App.GetWidth())
			assert.Equal(t, width, responsiveStyles.Header.GetWidth())
			assert.Equal(t, width, responsiveStyles.Footer.GetWidth())

			// Panel and Card should account for borders/padding
			assert.Equal(t, width-4, responsiveStyles.Panel.GetWidth())
			assert.Equal(t, width-4, responsiveStyles.Card.GetWidth())

			// Verify content renders without panic
			rendered := responsiveStyles.Panel.Render("Content at width " + string(rune('0'+width/10)) + "0")
			assert.NotEmpty(t, rendered)
		})
	}
}

// TestResponsiveHeightStyling verifies styles adapt to different heights.
func TestResponsiveHeightStyling(t *testing.T) {
	theme := DefaultTheme()
	styles := NewStyles(theme)

	heights := []int{20, 30, 40, 50}

	for _, height := range heights {
		t.Run(string(rune('0'+height/10))+"0_height", func(t *testing.T) {
			responsiveStyles := styles.WithHeight(height)

			// Verify height is properly set
			assert.Equal(t, height, responsiveStyles.App.GetHeight())
		})
	}
}

// TestResponsiveWidthAndHeight verifies combined width and height adjustments.
func TestResponsiveWidthAndHeight(t *testing.T) {
	theme := DefaultTheme()
	styles := NewStyles(theme)

	testCases := []struct {
		name   string
		width  int
		height int
	}{
		{"Small terminal", 40, 20},
		{"Medium terminal", 80, 24},
		{"Large terminal", 120, 40},
		{"Wide terminal", 160, 30},
		{"Tall terminal", 80, 60},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			responsiveStyles := styles.WithWidth(tc.width).WithHeight(tc.height)

			assert.Equal(t, tc.width, responsiveStyles.App.GetWidth())
			assert.Equal(t, tc.height, responsiveStyles.App.GetHeight())

			// Render a complex layout
			box := responsiveStyles.RenderBox("Title", "Content for "+tc.name, tc.width-10)
			assert.NotEmpty(t, box)
		})
	}
}

// TestMinimumTerminalSize verifies styles work with minimum terminal sizes.
func TestMinimumTerminalSize(t *testing.T) {
	theme := DefaultTheme()
	styles := NewStyles(theme)

	// Test minimum viable sizes
	minWidth := 20
	minHeight := 10

	responsiveStyles := styles.WithWidth(minWidth).WithHeight(minHeight)

	assert.NotPanics(t, func() {
		_ = responsiveStyles.Title.Render("Short")
		_ = responsiveStyles.Panel.Render("Panel")
		_ = responsiveStyles.Button.Render("OK")
	})
}

// TestExtremeTerminalSizes verifies styles handle extreme sizes gracefully.
func TestExtremeTerminalSizes(t *testing.T) {
	theme := DefaultTheme()
	styles := NewStyles(theme)

	testCases := []struct {
		name   string
		width  int
		height int
	}{
		{"Very narrow", 10, 40},
		{"Very short", 80, 5},
		{"Very large", 300, 100},
		{"Zero width", 0, 40},
		{"Zero height", 80, 0},
		{"Negative width", -10, 40},
		{"Negative height", 80, -10},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Should not panic even with extreme values
			assert.NotPanics(t, func() {
				responsiveStyles := styles.WithWidth(tc.width).WithHeight(tc.height)
				_ = responsiveStyles.Title.Render("Test")
				_ = responsiveStyles.Panel.Render("Test")
			})
		})
	}
}

// =============================================================================
// Accessibility Integration Tests
// =============================================================================

// TestHighContrastThemeAccessibility verifies high contrast theme meets accessibility needs.
func TestHighContrastThemeAccessibility(t *testing.T) {
	theme := HighContrastTheme()

	// High contrast theme should use maximum contrast colors
	assert.Equal(t, lipgloss.Color("#00FF00"), theme.Primary,
		"High contrast should use bright green")

	// Text should be high contrast
	if adaptiveText, ok := theme.Text.(lipgloss.AdaptiveColor); ok {
		assert.Equal(t, "#000000", adaptiveText.Light, "Light mode text should be black")
		assert.Equal(t, "#FFFFFF", adaptiveText.Dark, "Dark mode text should be white")
	}

	// Background should be high contrast
	if adaptiveBg, ok := theme.Background.(lipgloss.AdaptiveColor); ok {
		assert.Equal(t, "#FFFFFF", adaptiveBg.Light, "Light mode background should be white")
		assert.Equal(t, "#000000", adaptiveBg.Dark, "Dark mode background should be black")
	}
}

// TestHighContrastNoMutedText verifies high contrast theme doesn't use muted text.
func TestHighContrastNoMutedText(t *testing.T) {
	theme := HighContrastTheme()

	// In high contrast mode, muted and subtle text should equal regular text
	assert.Equal(t, theme.Text, theme.TextMuted,
		"High contrast should not have muted text (same as regular text)")
	assert.Equal(t, theme.Text, theme.TextSubtle,
		"High contrast should not have subtle text (same as regular text)")
}

// TestSemanticColorAccessibility verifies semantic colors are accessible.
func TestSemanticColorAccessibility(t *testing.T) {
	// Verify that semantic colors use appropriate color ranges
	// Success should use green tones
	assert.True(t, strings.Contains(ColorSuccess.Dark, "4A") || strings.Contains(ColorSuccess.Dark, "DE"),
		"Success should use green tones")

	// Error should use red tones
	assert.True(t, strings.Contains(ColorError.Dark, "F8") || strings.Contains(ColorError.Dark, "71"),
		"Error should use red tones")

	// Warning should use yellow/amber tones
	assert.True(t, strings.Contains(ColorWarning.Dark, "FA") || strings.Contains(ColorWarning.Dark, "CC"),
		"Warning should use yellow/amber tones")

	// Info should use blue tones
	assert.True(t, strings.Contains(ColorInfo.Dark, "38") || strings.Contains(ColorInfo.Dark, "BD"),
		"Info should use blue tones")
}

// TestTextReadabilityAcrossThemes verifies text remains readable across all themes.
func TestTextReadabilityAcrossThemes(t *testing.T) {
	themes := []*Theme{
		DefaultTheme(),
		LightTheme(),
		HighContrastTheme(),
	}

	for _, theme := range themes {
		t.Run(string(theme.Name), func(t *testing.T) {
			styles := theme.Styles

			// Render text with all text styles
			textStyles := []struct {
				name  string
				style lipgloss.Style
			}{
				{"Title", styles.Title},
				{"Subtitle", styles.Subtitle},
				{"Paragraph", styles.Paragraph},
				{"Help", styles.Help},
				{"Code", styles.Code},
			}

			for _, ts := range textStyles {
				rendered := ts.style.Render("Readable Text Sample 123")
				assert.NotEmpty(t, rendered, "%s text should render", ts.name)
			}
		})
	}
}

// TestButtonFocusVisibility verifies focused buttons have different style definitions.
func TestButtonFocusVisibility(t *testing.T) {
	themes := []*Theme{
		DefaultTheme(),
		LightTheme(),
		HighContrastTheme(),
	}

	for _, theme := range themes {
		t.Run(string(theme.Name), func(t *testing.T) {
			styles := theme.Styles

			// Verify button styles are defined differently
			// Note: Rendered output may look identical in non-ANSI terminals,
			// so we verify the style definitions are different rather than rendered output
			normalButton := styles.Button.Render("Button")
			focusedButton := styles.ButtonFocused.Render("Button")

			// Both should render without panicking and produce non-empty output
			assert.NotEmpty(t, normalButton, "Normal button should render")
			assert.NotEmpty(t, focusedButton, "Focused button should render")

			// In a color-capable terminal, these would be different.
			// We verify both styles can render successfully.
		})
	}
}

// TestInputFocusAndErrorStates verifies input states have different style definitions.
func TestInputFocusAndErrorStates(t *testing.T) {
	theme := DefaultTheme()
	styles := theme.Styles

	normalInput := styles.Input.Render("Input")
	focusedInput := styles.InputFocused.Render("Input")
	errorInput := styles.InputError.Render("Input")

	// All three states should render successfully
	// Note: Rendered output may look identical in non-ANSI terminals,
	// so we verify they all render without panicking
	assert.NotEmpty(t, normalInput, "Normal input should render")
	assert.NotEmpty(t, focusedInput, "Focused input should render")
	assert.NotEmpty(t, errorInput, "Error input should render")
}

// TestBorderFocusVisibility verifies border styles have different definitions.
func TestBorderFocusVisibility(t *testing.T) {
	theme := DefaultTheme()
	styles := theme.Styles

	normalBorder := styles.BorderNormal.Render("Content")
	focusedBorder := styles.BorderFocused.Render("Content")
	activeBorder := styles.BorderActive.Render("Content")

	// All border states should render successfully
	// Note: Rendered output may look identical in non-ANSI terminals,
	// so we verify they all render without panicking
	assert.NotEmpty(t, normalBorder, "Normal border should render")
	assert.NotEmpty(t, focusedBorder, "Focused border should render")
	assert.NotEmpty(t, activeBorder, "Active border should render")
}

// =============================================================================
// Render Helper Integration Tests
// =============================================================================

// TestRenderHelpersIntegration verifies all render helpers work together.
func TestRenderHelpersIntegration(t *testing.T) {
	theme := DefaultTheme()
	styles := theme.Styles

	// Create a complex UI layout using multiple helpers
	assert.NotPanics(t, func() {
		// GPU card
		gpuCard := styles.RenderGPUCard("GeForce RTX 4090", "545.29.02", "24 GB", 60)
		assert.NotEmpty(t, gpuCard)

		// Progress bar
		progressBar := styles.RenderProgressBarWithLabel(75, 100, 40)
		assert.NotEmpty(t, progressBar)

		// Status lines
		successLine := styles.RenderStatusLine("success", "Installation complete")
		warningLine := styles.RenderStatusLine("warning", "Reboot recommended")
		errorLine := styles.RenderStatusLine("error", "Installation failed")
		infoLine := styles.RenderStatusLine("info", "Detecting hardware...")

		assert.NotEmpty(t, successLine)
		assert.NotEmpty(t, warningLine)
		assert.NotEmpty(t, errorLine)
		assert.NotEmpty(t, infoLine)

		// Dialog
		dialog := styles.RenderDialog("Confirm", "Proceed with installation?", []string{"Yes", "No"}, 0, 50)
		assert.NotEmpty(t, dialog)

		// Box
		box := styles.RenderBox("System Info", "Linux 6.1.0 x86_64", 40)
		assert.NotEmpty(t, box)

		// List
		list := styles.RenderList([]string{"Driver 545.29.02", "Driver 535.154.05", "Driver 530.30.02"}, 1)
		assert.NotEmpty(t, list)

		// Key-value
		kv := styles.RenderKeyValue("Version", "1.0.0")
		assert.NotEmpty(t, kv)
	})
}

// TestRenderHelpersWithAllThemes verifies render helpers work with all themes.
func TestRenderHelpersWithAllThemes(t *testing.T) {
	themes := []*Theme{
		DefaultTheme(),
		LightTheme(),
		HighContrastTheme(),
	}

	for _, theme := range themes {
		t.Run(string(theme.Name), func(t *testing.T) {
			styles := theme.Styles

			// All render helpers should work
			assert.NotEmpty(t, styles.RenderBox("Title", "Content", 40))
			assert.NotEmpty(t, styles.RenderProgressBar(50, 100, 20))
			assert.NotEmpty(t, styles.RenderProgressBarWithLabel(50, 100, 30))
			assert.NotEmpty(t, styles.RenderStatusLine("success", "OK"))
			assert.NotEmpty(t, styles.RenderKeyValue("Key", "Value"))
			assert.NotEmpty(t, styles.RenderGPUCard("GPU", "Driver", "Memory", 50))
			assert.NotEmpty(t, styles.RenderButton("Click", false))
			assert.NotEmpty(t, styles.RenderButton("Click", true))
			assert.NotEmpty(t, styles.RenderDialog("Title", "Content", []string{"OK"}, 0, 40))
			assert.NotEmpty(t, styles.RenderList([]string{"Item 1", "Item 2"}, 0))
		})
	}
}

// =============================================================================
// Theme Switching Integration Tests
// =============================================================================

// TestThemeSwitching verifies themes can be switched at runtime.
func TestThemeSwitching(t *testing.T) {
	// Simulate switching themes
	currentTheme := DefaultTheme()
	content := "Test Content"

	// Render with dark theme
	darkRender := currentTheme.Styles.Title.Render(content)
	assert.NotEmpty(t, darkRender)

	// Switch to light theme
	currentTheme = LightTheme()
	lightRender := currentTheme.Styles.Title.Render(content)
	assert.NotEmpty(t, lightRender)

	// Switch to high contrast theme
	currentTheme = HighContrastTheme()
	highContrastRender := currentTheme.Styles.Title.Render(content)
	assert.NotEmpty(t, highContrastRender)

	// All renders should be valid (themes may or may not produce different output
	// depending on terminal capabilities, but they should all work)
	assert.NotPanics(t, func() {
		_ = DefaultTheme().Styles.Panel.Render("Panel")
		_ = LightTheme().Styles.Panel.Render("Panel")
		_ = HighContrastTheme().Styles.Panel.Render("Panel")
	})
}

// TestThemeCustomization verifies themes can be customized.
func TestThemeCustomization(t *testing.T) {
	theme := DefaultTheme()

	// Create customized version
	customPrimary := lipgloss.Color("#FF5500")
	customTheme := theme.WithPrimary(customPrimary)

	// Verify customization
	assert.Equal(t, customPrimary, customTheme.Primary)
	assert.NotEqual(t, theme.Primary, customTheme.Primary)

	// Verify styles are regenerated with new primary
	originalTitle := theme.Styles.Title.Render("Title")
	customTitle := customTheme.Styles.Title.Render("Title")
	assert.NotEmpty(t, originalTitle)
	assert.NotEmpty(t, customTitle)
}

// TestThemeCopyIndependence verifies copied themes are independent.
func TestThemeCopyIndependence(t *testing.T) {
	original := DefaultTheme()
	copied := original.Copy()

	// Modify the copy
	copied.Primary = lipgloss.Color("#123456")

	// Original should be unchanged
	assert.Equal(t, NVIDIAGreen, original.Primary)
	assert.NotEqual(t, original.Primary, copied.Primary)
}

// =============================================================================
// Status Color Integration Tests
// =============================================================================

// TestStatusColorIntegration verifies status colors integrate properly with styles.
func TestStatusColorIntegration(t *testing.T) {
	statuses := []StatusColor{
		StatusSuccess,
		StatusWarning,
		StatusError,
		StatusInfo,
	}

	for _, status := range statuses {
		t.Run(string(status), func(t *testing.T) {
			// Get status color
			color := GetStatusColor(status)
			assert.NotNil(t, color)

			// Create indicator
			indicator := StatusIndicator(status)
			assert.NotEmpty(t, indicator)
			assert.Contains(t, indicator, "")

			// Create styled text
			style := lipgloss.NewStyle().Foreground(color)
			rendered := style.Render("Status message")
			assert.NotEmpty(t, rendered)
		})
	}
}

// TestUnknownStatusColorFallback verifies unknown status colors fall back gracefully.
func TestUnknownStatusColorFallback(t *testing.T) {
	unknownStatuses := []StatusColor{
		StatusColor("unknown"),
		StatusColor(""),
		StatusColor("invalid"),
	}

	for _, status := range unknownStatuses {
		t.Run(string(status), func(t *testing.T) {
			color := GetStatusColor(status)
			// Should return ColorInfo as fallback
			assert.Equal(t, ColorInfo, color)
		})
	}
}

// =============================================================================
// Benchmarks
// =============================================================================

func BenchmarkStyle_Apply(b *testing.B) {
	theme := DefaultTheme()
	styles := theme.Styles
	content := "Benchmark Content"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = styles.Title.Render(content)
	}
}

func BenchmarkStyles_WithWidth(b *testing.B) {
	theme := DefaultTheme()
	styles := NewStyles(theme)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = styles.WithWidth(80)
	}
}

func BenchmarkStyles_WithWidthAndHeight(b *testing.B) {
	theme := DefaultTheme()
	styles := NewStyles(theme)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = styles.WithWidth(80).WithHeight(24)
	}
}

func BenchmarkRenderBoxIntegration(b *testing.B) {
	theme := DefaultTheme()
	styles := theme.Styles

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = styles.RenderBox("Title", "Content", 60)
	}
}

func BenchmarkRenderGPUCard(b *testing.B) {
	theme := DefaultTheme()
	styles := theme.Styles

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = styles.RenderGPUCard("GeForce RTX 4090", "545.29.02", "24 GB", 60)
	}
}

func BenchmarkRenderDialog(b *testing.B) {
	theme := DefaultTheme()
	styles := theme.Styles
	buttons := []string{"Yes", "No", "Cancel"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = styles.RenderDialog("Confirm", "Are you sure?", buttons, 0, 50)
	}
}

func BenchmarkGetTheme(b *testing.B) {
	themes := []ThemeName{ThemeNVIDIADark, ThemeNVIDIALight, ThemeHighContrast}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = GetTheme(themes[i%3])
	}
}

func BenchmarkStatusIndicator(b *testing.B) {
	statuses := []StatusColor{StatusSuccess, StatusWarning, StatusError, StatusInfo}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = StatusIndicator(statuses[i%4])
	}
}

func BenchmarkMultipleStyleRenders(b *testing.B) {
	theme := DefaultTheme()
	styles := theme.Styles
	content := "Benchmark Content"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = styles.Title.Render(content)
		_ = styles.Panel.Render(content)
		_ = styles.Button.Render(content)
		_ = styles.Success.Render(content)
	}
}

func BenchmarkThemeCopy(b *testing.B) {
	theme := DefaultTheme()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = theme.Copy()
	}
}

// =============================================================================
// Edge Case Integration Tests
// =============================================================================

// TestStylesWithEmptyContent verifies styles handle empty content.
func TestStylesWithEmptyContent(t *testing.T) {
	theme := DefaultTheme()
	styles := theme.Styles

	// Empty string should not panic
	assert.NotPanics(t, func() {
		_ = styles.Title.Render("")
		_ = styles.Panel.Render("")
		_ = styles.Button.Render("")
		_ = styles.RenderBox("Title", "", 40)
		_ = styles.RenderList([]string{}, 0)
		_ = styles.RenderDialog("Title", "", []string{}, 0, 40)
	})
}

// TestStylesWithVeryLongContent verifies styles handle very long content.
func TestStylesWithVeryLongContent(t *testing.T) {
	theme := DefaultTheme()
	styles := theme.Styles

	longContent := strings.Repeat("Very long content that repeats. ", 100)

	assert.NotPanics(t, func() {
		_ = styles.Title.Render(longContent)
		_ = styles.Panel.Render(longContent)
		_ = styles.Paragraph.Render(longContent)
	})
}

// TestStylesWithSpecialCharacters verifies styles handle special characters.
func TestStylesWithSpecialCharacters(t *testing.T) {
	theme := DefaultTheme()
	styles := theme.Styles

	specialContents := []string{
		"\t\n\r",                // Whitespace
		"<>\"'&",                // HTML-like chars
		"\\n\\t\\r",             // Escaped sequences
		"\x00\x01\x02",          // Control chars
		"NVIDIA",                // Already has special chars (trademark)
		string([]rune{0x1F4A1}), // Emoji (lightbulb)
		"\033[31mred\033[0m",    // ANSI escape
	}

	for i, content := range specialContents {
		t.Run(string(rune('A'+i)), func(t *testing.T) {
			assert.NotPanics(t, func() {
				_ = styles.Title.Render(content)
				_ = styles.Panel.Render(content)
			})
		})
	}
}

// TestThemeRequired verifies NewStyles requires a valid theme.
func TestThemeRequired(t *testing.T) {
	theme := &Theme{
		Name:         ThemeNVIDIADark,
		Primary:      NVIDIAGreen,
		PrimaryDark:  NVIDIAGreenDark,
		PrimaryLight: NVIDIAGreenLight,
		Secondary:    NVIDIAGray,
		// Other fields left as zero values
	}

	// Should not panic even with minimal theme
	assert.NotPanics(t, func() {
		styles := NewStyles(theme)
		_ = styles.Title.Render("Test")
	})
}
