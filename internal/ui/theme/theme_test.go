package theme

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Color Tests
// =============================================================================

func TestNVIDIAColors(t *testing.T) {
	tests := []struct {
		name     string
		color    lipgloss.Color
		expected string
	}{
		{"NVIDIAGreen", NVIDIAGreen, "#76B900"},
		{"NVIDIAGreenDark", NVIDIAGreenDark, "#5A8F00"},
		{"NVIDIAGreenLight", NVIDIAGreenLight, "#8BD000"},
		{"NVIDIABlack", NVIDIABlack, "#1A1A1A"},
		{"NVIDIAWhite", NVIDIAWhite, "#FFFFFF"},
		{"NVIDIAGray", NVIDIAGray, "#666666"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, string(tt.color))
		})
	}
}

func TestAdaptiveColors(t *testing.T) {
	// Test that adaptive colors have valid Light and Dark values
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
		{"ColorBackground", ColorBackground},
		{"ColorBackgroundAlt", ColorBackgroundAlt},
		{"ColorBackgroundPanel", ColorBackgroundPanel},
		{"ColorBorder", ColorBorder},
		{"ColorBorderFocus", ColorBorderFocus},
		{"ColorProgress", ColorProgress},
		{"ColorProgressBg", ColorProgressBg},
	}

	for _, tt := range adaptiveColors {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotEmpty(t, tt.color.Light, "Light color should not be empty")
			assert.NotEmpty(t, tt.color.Dark, "Dark color should not be empty")
		})
	}
}

func TestColorFormats(t *testing.T) {
	// Test that hex colors start with #
	hexColors := []lipgloss.Color{
		NVIDIAGreen,
		NVIDIAGreenDark,
		NVIDIAGreenLight,
		NVIDIABlack,
		NVIDIAWhite,
		NVIDIAGray,
	}

	for i, color := range hexColors {
		colorStr := string(color)
		assert.True(t, strings.HasPrefix(colorStr, "#"),
			"Color %d should be a hex color starting with #", i)
		assert.Equal(t, 7, len(colorStr),
			"Color %d should be 7 characters (#RRGGBB)", i)
	}
}

func TestStatusColors(t *testing.T) {
	tests := []struct {
		status StatusColor
		color  lipgloss.AdaptiveColor
	}{
		{StatusSuccess, ColorSuccess},
		{StatusWarning, ColorWarning},
		{StatusError, ColorError},
		{StatusInfo, ColorInfo},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			result := GetStatusColor(tt.status)
			assert.Equal(t, tt.color, result)
		})
	}
}

func TestGetStatusColorDefault(t *testing.T) {
	// Unknown status should return ColorInfo
	result := GetStatusColor(StatusColor("unknown"))
	assert.Equal(t, ColorInfo, result)
}

func TestStatusIndicator(t *testing.T) {
	tests := []StatusColor{
		StatusSuccess,
		StatusWarning,
		StatusError,
		StatusInfo,
	}

	for _, status := range tests {
		t.Run(string(status), func(t *testing.T) {
			indicator := StatusIndicator(status)
			assert.NotEmpty(t, indicator)
			// The indicator should contain the bullet character
			assert.Contains(t, indicator, "●")
		})
	}
}

// =============================================================================
// Theme Tests
// =============================================================================

func TestDefaultTheme(t *testing.T) {
	theme := DefaultTheme()

	require.NotNil(t, theme)
	assert.Equal(t, ThemeNVIDIADark, theme.Name)
	assert.Equal(t, NVIDIAGreen, theme.Primary)
	assert.Equal(t, NVIDIAGreenDark, theme.PrimaryDark)
	assert.Equal(t, NVIDIAGreenLight, theme.PrimaryLight)

	// Check semantic colors
	assert.Equal(t, ColorSuccess, theme.Success)
	assert.Equal(t, ColorWarning, theme.Warning)
	assert.Equal(t, ColorError, theme.Error)
	assert.Equal(t, ColorInfo, theme.Info)

	// Check text colors
	assert.Equal(t, ColorText, theme.Text)
	assert.Equal(t, ColorTextMuted, theme.TextMuted)
	assert.Equal(t, ColorTextSubtle, theme.TextSubtle)

	// Check that Styles are initialized
	assert.NotEmpty(t, theme.Styles.Title.String())
}

func TestLightTheme(t *testing.T) {
	theme := LightTheme()

	require.NotNil(t, theme)
	assert.Equal(t, ThemeNVIDIALight, theme.Name)
	assert.Equal(t, NVIDIAGreen, theme.Primary)

	// Light theme should have light-optimized colors
	assert.NotNil(t, theme.Text)
	assert.NotNil(t, theme.Background)
}

func TestHighContrastTheme(t *testing.T) {
	theme := HighContrastTheme()

	require.NotNil(t, theme)
	assert.Equal(t, ThemeHighContrast, theme.Name)

	// High contrast theme should use bright colors
	assert.Equal(t, lipgloss.Color("#00FF00"), theme.Primary)
}

func TestGetTheme(t *testing.T) {
	tests := []struct {
		name     ThemeName
		expected ThemeName
	}{
		{ThemeNVIDIADark, ThemeNVIDIADark},
		{ThemeNVIDIALight, ThemeNVIDIALight},
		{ThemeHighContrast, ThemeHighContrast},
		{"unknown", ThemeNVIDIADark}, // Unknown should return default
	}

	for _, tt := range tests {
		t.Run(string(tt.name), func(t *testing.T) {
			theme := GetTheme(tt.name)
			assert.Equal(t, tt.expected, theme.Name)
		})
	}
}

func TestAvailableThemes(t *testing.T) {
	themes := AvailableThemes()

	assert.Len(t, themes, 3)
	assert.Contains(t, themes, ThemeNVIDIADark)
	assert.Contains(t, themes, ThemeNVIDIALight)
	assert.Contains(t, themes, ThemeHighContrast)
}

func TestThemeIsDarkTheme(t *testing.T) {
	tests := []struct {
		theme    *Theme
		expected bool
	}{
		{DefaultTheme(), true},
		{LightTheme(), false},
		{HighContrastTheme(), true},
	}

	for _, tt := range tests {
		t.Run(string(tt.theme.Name), func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.theme.IsDarkTheme())
		})
	}
}

func TestThemeCopy(t *testing.T) {
	original := DefaultTheme()
	copied := original.Copy()

	// Verify it's a different instance
	assert.NotSame(t, original, copied)
	assert.Equal(t, original.Name, copied.Name)
	assert.Equal(t, original.Primary, copied.Primary)

	// Modifying the copy shouldn't affect the original
	copied.Primary = lipgloss.Color("#FF0000")
	assert.NotEqual(t, original.Primary, copied.Primary)
}

func TestThemeWithPrimary(t *testing.T) {
	original := DefaultTheme()
	customPrimary := lipgloss.Color("#FF5500")
	modified := original.WithPrimary(customPrimary)

	// Verify the original is unchanged
	assert.Equal(t, NVIDIAGreen, original.Primary)

	// Verify the modified theme has the new primary
	assert.Equal(t, customPrimary, modified.Primary)
	assert.NotSame(t, original, modified)
}

// =============================================================================
// Styles Tests
// =============================================================================

func TestNewStyles(t *testing.T) {
	theme := DefaultTheme()
	styles := NewStyles(theme)

	// Test that basic styles are created
	assert.NotZero(t, styles.Title)
	assert.NotZero(t, styles.Subtitle)
	assert.NotZero(t, styles.Paragraph)
	assert.NotZero(t, styles.Panel)
	assert.NotZero(t, styles.Card)
	assert.NotZero(t, styles.Button)
	assert.NotZero(t, styles.ButtonFocused)
	assert.NotZero(t, styles.Success)
	assert.NotZero(t, styles.Warning)
	assert.NotZero(t, styles.Error)
	assert.NotZero(t, styles.Info)
}

func TestStylesCopy(t *testing.T) {
	theme := DefaultTheme()
	original := NewStyles(theme)
	copied := original.Copy()

	// Since Styles contains value types, this should create a copy
	assert.Equal(t, original.Title.String(), copied.Title.String())

	// Modifying the copy's width should not affect original
	copied = copied.WithWidth(100)
	// The styles should now be different (copied has width set)
	assert.Equal(t, 100, copied.App.GetWidth())
}

func TestStylesWithWidth(t *testing.T) {
	theme := DefaultTheme()
	styles := NewStyles(theme)
	width := 80

	adjusted := styles.WithWidth(width)

	assert.Equal(t, width, adjusted.App.GetWidth())
	assert.Equal(t, width, adjusted.Header.GetWidth())
	assert.Equal(t, width, adjusted.Footer.GetWidth())
	assert.Equal(t, width-4, adjusted.Panel.GetWidth())
	assert.Equal(t, width-4, adjusted.Card.GetWidth())
}

func TestStylesWithHeight(t *testing.T) {
	theme := DefaultTheme()
	styles := NewStyles(theme)
	height := 40

	adjusted := styles.WithHeight(height)

	assert.Equal(t, height, adjusted.App.GetHeight())
}

func TestStylesWithWidthAndHeight(t *testing.T) {
	theme := DefaultTheme()
	styles := NewStyles(theme)

	adjusted := styles.WithWidth(100).WithHeight(50)

	assert.Equal(t, 100, adjusted.App.GetWidth())
	assert.Equal(t, 50, adjusted.App.GetHeight())
}

// =============================================================================
// Render Helper Tests
// =============================================================================

func TestRenderBox(t *testing.T) {
	theme := DefaultTheme()
	styles := NewStyles(theme)

	result := styles.RenderBox("Test Title", "Test Content", 40)

	assert.NotEmpty(t, result)
	assert.Contains(t, result, "Test Title")
	assert.Contains(t, result, "Test Content")
}

func TestRenderBoxEmptyContent(t *testing.T) {
	theme := DefaultTheme()
	styles := NewStyles(theme)

	result := styles.RenderBox("Title", "", 40)

	assert.NotEmpty(t, result)
	assert.Contains(t, result, "Title")
}

func TestRenderStatusLine(t *testing.T) {
	theme := DefaultTheme()
	styles := NewStyles(theme)

	tests := []struct {
		status   string
		message  string
		contains string
	}{
		{"success", "Operation succeeded", "●"},
		{"warning", "Be careful", "●"},
		{"error", "Something failed", "●"},
		{"info", "Information", "●"},
		{"unknown", "Unknown status", "●"}, // Should default to info style
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			result := styles.RenderStatusLine(tt.status, tt.message)
			assert.Contains(t, result, tt.contains)
			assert.Contains(t, result, tt.message)
		})
	}
}

func TestRenderProgressBar(t *testing.T) {
	theme := DefaultTheme()
	styles := NewStyles(theme)

	tests := []struct {
		name     string
		current  int
		total    int
		width    int
		expected int // expected length of filled chars
	}{
		{"empty", 0, 100, 20, 0},
		{"half", 50, 100, 20, 10},
		{"full", 100, 100, 20, 20},
		{"quarter", 25, 100, 20, 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := styles.RenderProgressBar(tt.current, tt.total, tt.width)
			assert.NotEmpty(t, result)
		})
	}
}

func TestRenderProgressBarEdgeCases(t *testing.T) {
	theme := DefaultTheme()
	styles := NewStyles(theme)

	// Zero total should return empty string
	assert.Empty(t, styles.RenderProgressBar(50, 0, 20))

	// Zero width should return empty string
	assert.Empty(t, styles.RenderProgressBar(50, 100, 0))

	// Negative width should return empty string
	assert.Empty(t, styles.RenderProgressBar(50, 100, -5))

	// Current greater than total should clamp to total
	result := styles.RenderProgressBar(150, 100, 20)
	assert.NotEmpty(t, result)

	// Negative current should clamp to zero
	result = styles.RenderProgressBar(-10, 100, 20)
	assert.NotEmpty(t, result)
}

func TestRenderProgressBarWithLabel(t *testing.T) {
	theme := DefaultTheme()
	styles := NewStyles(theme)

	result := styles.RenderProgressBarWithLabel(50, 100, 30)
	assert.NotEmpty(t, result)

	// Zero total should return empty
	assert.Empty(t, styles.RenderProgressBarWithLabel(50, 0, 30))

	// Zero width should return empty
	assert.Empty(t, styles.RenderProgressBarWithLabel(50, 100, 0))
}

func TestFormatPercent(t *testing.T) {
	tests := []struct {
		percent  int
		expected string
	}{
		{0, "  0%"},
		{5, "  5%"},
		{10, " 10%"},
		{50, " 50%"},
		{99, " 99%"},
		{100, "100%"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := formatPercent(tt.percent)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRenderKeyValue(t *testing.T) {
	theme := DefaultTheme()
	styles := NewStyles(theme)

	result := styles.RenderKeyValue("Driver", "535.154.05")

	assert.NotEmpty(t, result)
	assert.Contains(t, result, "Driver")
	assert.Contains(t, result, "535.154.05")
}

func TestRenderGPUCard(t *testing.T) {
	theme := DefaultTheme()
	styles := NewStyles(theme)

	result := styles.RenderGPUCard("GeForce RTX 4090", "535.154.05", "24GB", 50)

	assert.NotEmpty(t, result)
	assert.Contains(t, result, "GeForce RTX 4090")
	assert.Contains(t, result, "535.154.05")
	assert.Contains(t, result, "24GB")
}

func TestRenderButton(t *testing.T) {
	theme := DefaultTheme()
	styles := NewStyles(theme)

	// Test normal button
	normalBtn := styles.RenderButton("OK", false)
	assert.NotEmpty(t, normalBtn)
	assert.Contains(t, normalBtn, "OK")

	// Test focused button
	focusedBtn := styles.RenderButton("Cancel", true)
	assert.NotEmpty(t, focusedBtn)
	assert.Contains(t, focusedBtn, "Cancel")

	// Focused and normal should be different
	assert.NotEqual(t, normalBtn, focusedBtn)
}

func TestRenderDialog(t *testing.T) {
	theme := DefaultTheme()
	styles := NewStyles(theme)

	buttons := []string{"OK", "Cancel"}
	result := styles.RenderDialog("Confirm", "Are you sure?", buttons, 0, 50)

	assert.NotEmpty(t, result)
	assert.Contains(t, result, "Confirm")
	assert.Contains(t, result, "Are you sure?")
	assert.Contains(t, result, "OK")
	assert.Contains(t, result, "Cancel")
}

func TestRenderDialogEmptyButtons(t *testing.T) {
	theme := DefaultTheme()
	styles := NewStyles(theme)

	result := styles.RenderDialog("Info", "Some message", []string{}, 0, 50)

	assert.NotEmpty(t, result)
	assert.Contains(t, result, "Info")
	assert.Contains(t, result, "Some message")
}

func TestRenderList(t *testing.T) {
	theme := DefaultTheme()
	styles := NewStyles(theme)

	items := []string{"Item 1", "Item 2", "Item 3"}
	result := styles.RenderList(items, 1)

	assert.NotEmpty(t, result)
	assert.Contains(t, result, "Item 1")
	assert.Contains(t, result, "Item 2")
	assert.Contains(t, result, "Item 3")
}

func TestRenderListEmpty(t *testing.T) {
	theme := DefaultTheme()
	styles := NewStyles(theme)

	result := styles.RenderList([]string{}, 0)
	assert.NotEmpty(t, result) // Should still render the list container
}

func TestRenderListSingleItem(t *testing.T) {
	theme := DefaultTheme()
	styles := NewStyles(theme)

	items := []string{"Only Item"}
	result := styles.RenderList(items, 0)

	assert.NotEmpty(t, result)
	assert.Contains(t, result, "Only Item")
}

// =============================================================================
// Style Property Tests
// =============================================================================

func TestStylePropertiesForeground(t *testing.T) {
	theme := DefaultTheme()
	styles := NewStyles(theme)

	// Title should have Primary color as foreground
	titleStyle := styles.Title
	// We verify the style can render without panicking
	rendered := titleStyle.Render("Test")
	assert.NotEmpty(t, rendered)
}

func TestStylePropertiesBorders(t *testing.T) {
	theme := DefaultTheme()
	styles := NewStyles(theme)

	// Panel should have rounded borders
	panelRendered := styles.Panel.Render("Content")
	assert.NotEmpty(t, panelRendered)

	// BorderNormal should have rounded borders
	borderRendered := styles.BorderNormal.Render("Content")
	assert.NotEmpty(t, borderRendered)
}

func TestStylePropertiesPadding(t *testing.T) {
	theme := DefaultTheme()
	styles := NewStyles(theme)

	// Header should have padding
	headerStyle := styles.Header
	rendered := headerStyle.Render("Header")
	// Rendered output should be longer than content due to padding
	assert.Greater(t, len(rendered), len("Header"))
}

func TestStylePropertiesBold(t *testing.T) {
	theme := DefaultTheme()
	styles := NewStyles(theme)

	// Title should be bold
	titleStyle := styles.Title
	rendered := titleStyle.Render("Bold Title")
	assert.NotEmpty(t, rendered)
}

// =============================================================================
// Integration Tests
// =============================================================================

func TestThemeStylesIntegration(t *testing.T) {
	// Test that theme.Styles is properly initialized
	for _, themeName := range AvailableThemes() {
		t.Run(string(themeName), func(t *testing.T) {
			theme := GetTheme(themeName)
			require.NotNil(t, theme)

			// Verify styles can render content without panicking
			rendered := theme.Styles.Title.Render("Test Title")
			assert.NotEmpty(t, rendered)

			rendered = theme.Styles.Panel.Render("Panel Content")
			assert.NotEmpty(t, rendered)

			rendered = theme.Styles.Button.Render("Button")
			assert.NotEmpty(t, rendered)

			rendered = theme.Styles.Success.Render("Success")
			assert.NotEmpty(t, rendered)
		})
	}
}

func TestAllStylesRenderWithoutPanic(t *testing.T) {
	theme := DefaultTheme()
	styles := theme.Styles

	// Test all styles can render without panicking
	styleTests := []struct {
		name  string
		style lipgloss.Style
	}{
		{"App", styles.App},
		{"Header", styles.Header},
		{"Footer", styles.Footer},
		{"Title", styles.Title},
		{"Subtitle", styles.Subtitle},
		{"Paragraph", styles.Paragraph},
		{"Help", styles.Help},
		{"Code", styles.Code},
		{"Panel", styles.Panel},
		{"Card", styles.Card},
		{"List", styles.List},
		{"ListItem", styles.ListItem},
		{"ListItemSelected", styles.ListItemSelected},
		{"ListItemFocused", styles.ListItemFocused},
		{"Button", styles.Button},
		{"ButtonFocused", styles.ButtonFocused},
		{"ButtonDisabled", styles.ButtonDisabled},
		{"ButtonPrimary", styles.ButtonPrimary},
		{"Input", styles.Input},
		{"InputFocused", styles.InputFocused},
		{"InputError", styles.InputError},
		{"Success", styles.Success},
		{"Warning", styles.Warning},
		{"Error", styles.Error},
		{"Info", styles.Info},
		{"ProgressBar", styles.ProgressBar},
		{"ProgressFilled", styles.ProgressFilled},
		{"ProgressEmpty", styles.ProgressEmpty},
		{"ProgressText", styles.ProgressText},
		{"Spinner", styles.Spinner},
		{"BorderNormal", styles.BorderNormal},
		{"BorderFocused", styles.BorderFocused},
		{"BorderActive", styles.BorderActive},
		{"GPUName", styles.GPUName},
		{"GPUInfo", styles.GPUInfo},
		{"DriverInfo", styles.DriverInfo},
		{"SystemInfo", styles.SystemInfo},
		{"VersionLabel", styles.VersionLabel},
		{"VersionValue", styles.VersionValue},
		{"Logo", styles.Logo},
		{"TagLine", styles.TagLine},
		{"Dialog", styles.Dialog},
		{"DialogTitle", styles.DialogTitle},
		{"DialogButton", styles.DialogButton},
		{"TableHeader", styles.TableHeader},
		{"TableRow", styles.TableRow},
		{"TableRowAlt", styles.TableRowAlt},
		{"TableCell", styles.TableCell},
	}

	for _, tt := range styleTests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotPanics(t, func() {
				rendered := tt.style.Render("Test content")
				assert.NotEmpty(t, rendered)
			})
		})
	}
}

func TestThemeConsistency(t *testing.T) {
	// Ensure all themes have the same structure
	themes := []*Theme{
		DefaultTheme(),
		LightTheme(),
		HighContrastTheme(),
	}

	for _, theme := range themes {
		t.Run(string(theme.Name), func(t *testing.T) {
			// All themes should have valid colors
			assert.NotEqual(t, lipgloss.Color(""), theme.Primary)
			assert.NotEqual(t, lipgloss.Color(""), theme.PrimaryDark)

			// All themes should have semantic colors
			assert.NotNil(t, theme.Success)
			assert.NotNil(t, theme.Warning)
			assert.NotNil(t, theme.Error)
			assert.NotNil(t, theme.Info)

			// All themes should have text colors
			assert.NotNil(t, theme.Text)
			assert.NotNil(t, theme.TextMuted)
			assert.NotNil(t, theme.TextSubtle)

			// All themes should have initialized styles
			rendered := theme.Styles.Title.Render("Test")
			assert.NotEmpty(t, rendered)
		})
	}
}

// =============================================================================
// Benchmark Tests
// =============================================================================

func BenchmarkDefaultTheme(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = DefaultTheme()
	}
}

func BenchmarkNewStyles(b *testing.B) {
	theme := DefaultTheme()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = NewStyles(theme)
	}
}

func BenchmarkRenderProgressBar(b *testing.B) {
	theme := DefaultTheme()
	styles := NewStyles(theme)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = styles.RenderProgressBar(50, 100, 40)
	}
}

func BenchmarkRenderBox(b *testing.B) {
	theme := DefaultTheme()
	styles := NewStyles(theme)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = styles.RenderBox("Title", "Content", 60)
	}
}

func BenchmarkStyleRender(b *testing.B) {
	theme := DefaultTheme()
	styles := NewStyles(theme)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = styles.Title.Render("Test Title")
	}
}
