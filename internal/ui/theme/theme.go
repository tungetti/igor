package theme

import "github.com/charmbracelet/lipgloss"

// ThemeName identifies a theme variant.
type ThemeName string

const (
	// ThemeNVIDIADark is the default NVIDIA-themed dark theme.
	ThemeNVIDIADark ThemeName = "nvidia-dark"

	// ThemeNVIDIALight is the NVIDIA-themed light theme.
	ThemeNVIDIALight ThemeName = "nvidia-light"

	// ThemeHighContrast is the high-contrast accessibility theme.
	ThemeHighContrast ThemeName = "high-contrast"
)

// Theme represents the complete visual theme for the TUI.
// It contains all color definitions and pre-built styles for consistent UI rendering.
type Theme struct {
	// Name is the theme identifier.
	Name ThemeName

	// Primary colors - NVIDIA Green palette
	Primary      lipgloss.Color
	PrimaryDark  lipgloss.Color
	PrimaryLight lipgloss.Color

	// Secondary colors
	Secondary     lipgloss.Color
	SecondaryDark lipgloss.Color

	// Semantic colors (using TerminalColor interface for adaptive support)
	Success lipgloss.TerminalColor
	Warning lipgloss.TerminalColor
	Error   lipgloss.TerminalColor
	Info    lipgloss.TerminalColor

	// Text colors
	Text        lipgloss.TerminalColor
	TextMuted   lipgloss.TerminalColor
	TextSubtle  lipgloss.TerminalColor
	TextInverse lipgloss.TerminalColor

	// Background colors
	Background          lipgloss.TerminalColor
	BackgroundAlt       lipgloss.TerminalColor
	BackgroundPanel     lipgloss.TerminalColor
	BackgroundHighlight lipgloss.TerminalColor
	BackgroundSelected  lipgloss.TerminalColor

	// Border colors
	Border       lipgloss.TerminalColor
	BorderFocus  lipgloss.TerminalColor
	BorderActive lipgloss.TerminalColor
	BorderMuted  lipgloss.TerminalColor

	// Progress bar colors
	Progress         lipgloss.TerminalColor
	ProgressBg       lipgloss.TerminalColor
	ProgressComplete lipgloss.TerminalColor

	// Styles contains pre-built lipgloss styles using theme colors.
	Styles Styles
}

// DefaultTheme returns the default NVIDIA-themed dark theme.
// This theme uses NVIDIA's brand green (#76B900) as the primary color
// with a dark background optimized for most terminal environments.
func DefaultTheme() *Theme {
	t := &Theme{
		Name:          ThemeNVIDIADark,
		Primary:       NVIDIAGreen,
		PrimaryDark:   NVIDIAGreenDark,
		PrimaryLight:  NVIDIAGreenLight,
		Secondary:     NVIDIAGray,
		SecondaryDark: NVIDIAGrayDark,

		// Semantic colors
		Success: ColorSuccess,
		Warning: ColorWarning,
		Error:   ColorError,
		Info:    ColorInfo,

		// Text colors
		Text:        ColorText,
		TextMuted:   ColorTextMuted,
		TextSubtle:  ColorTextSubtle,
		TextInverse: ColorTextInverse,

		// Background colors
		Background:          ColorBackground,
		BackgroundAlt:       ColorBackgroundAlt,
		BackgroundPanel:     ColorBackgroundPanel,
		BackgroundHighlight: ColorBackgroundHighlight,
		BackgroundSelected:  ColorBackgroundSelected,

		// Border colors
		Border:       ColorBorder,
		BorderFocus:  ColorBorderFocus,
		BorderActive: ColorBorderActive,
		BorderMuted:  ColorBorderMuted,

		// Progress colors
		Progress:         ColorProgress,
		ProgressBg:       ColorProgressBg,
		ProgressComplete: ColorProgressComplete,
	}
	t.Styles = NewStyles(t)
	return t
}

// LightTheme returns an NVIDIA-themed light theme.
// This theme is optimized for light terminal backgrounds while
// maintaining the NVIDIA brand identity.
func LightTheme() *Theme {
	t := &Theme{
		Name:          ThemeNVIDIALight,
		Primary:       NVIDIAGreen,
		PrimaryDark:   NVIDIAGreenDark,
		PrimaryLight:  NVIDIAGreenLight,
		Secondary:     NVIDIAGray,
		SecondaryDark: NVIDIAGrayDark,

		// Semantic colors - same as default, AdaptiveColor handles light theme
		Success: ColorSuccess,
		Warning: ColorWarning,
		Error:   ColorError,
		Info:    ColorInfo,

		// Text colors - inverted for light backgrounds
		Text:        lipgloss.AdaptiveColor{Light: "#1F2937", Dark: "#1F2937"},
		TextMuted:   lipgloss.AdaptiveColor{Light: "#6B7280", Dark: "#6B7280"},
		TextSubtle:  lipgloss.AdaptiveColor{Light: "#9CA3AF", Dark: "#9CA3AF"},
		TextInverse: lipgloss.AdaptiveColor{Light: "#FFFFFF", Dark: "#FFFFFF"},

		// Background colors - light variants
		Background:          lipgloss.AdaptiveColor{Light: "#FFFFFF", Dark: "#FFFFFF"},
		BackgroundAlt:       lipgloss.AdaptiveColor{Light: "#F3F4F6", Dark: "#F3F4F6"},
		BackgroundPanel:     lipgloss.AdaptiveColor{Light: "#E5E7EB", Dark: "#E5E7EB"},
		BackgroundHighlight: lipgloss.AdaptiveColor{Light: "#D1FAE5", Dark: "#D1FAE5"},
		BackgroundSelected:  lipgloss.AdaptiveColor{Light: "#DCFCE7", Dark: "#DCFCE7"},

		// Border colors - light variants
		Border:       lipgloss.AdaptiveColor{Light: "#D1D5DB", Dark: "#D1D5DB"},
		BorderFocus:  lipgloss.AdaptiveColor{Light: "#76B900", Dark: "#76B900"},
		BorderActive: lipgloss.AdaptiveColor{Light: "#5A8F00", Dark: "#5A8F00"},
		BorderMuted:  lipgloss.AdaptiveColor{Light: "#E5E7EB", Dark: "#E5E7EB"},

		// Progress colors
		Progress:         ColorProgress,
		ProgressBg:       lipgloss.AdaptiveColor{Light: "#E5E7EB", Dark: "#E5E7EB"},
		ProgressComplete: ColorProgressComplete,
	}
	t.Styles = NewStyles(t)
	return t
}

// HighContrastTheme returns a high-contrast accessible theme.
// This theme uses maximum contrast colors for better visibility
// and accessibility compliance.
func HighContrastTheme() *Theme {
	t := &Theme{
		Name:          ThemeHighContrast,
		Primary:       lipgloss.Color("#00FF00"), // Bright green
		PrimaryDark:   lipgloss.Color("#008000"), // Dark green
		PrimaryLight:  lipgloss.Color("#00FF00"), // Same as primary
		Secondary:     lipgloss.Color("#FFFFFF"), // White
		SecondaryDark: lipgloss.Color("#808080"), // Gray

		// High contrast semantic colors
		Success: lipgloss.AdaptiveColor{Light: "#008000", Dark: "#00FF00"},
		Warning: lipgloss.AdaptiveColor{Light: "#FFD700", Dark: "#FFFF00"},
		Error:   lipgloss.AdaptiveColor{Light: "#FF0000", Dark: "#FF0000"},
		Info:    lipgloss.AdaptiveColor{Light: "#0000FF", Dark: "#00FFFF"},

		// High contrast text
		Text:        ColorHighContrastText,
		TextMuted:   ColorHighContrastText, // No muted text in high contrast
		TextSubtle:  ColorHighContrastText, // No subtle text in high contrast
		TextInverse: lipgloss.AdaptiveColor{Light: "#FFFFFF", Dark: "#000000"},

		// High contrast backgrounds
		Background:          ColorHighContrastBg,
		BackgroundAlt:       ColorHighContrastBg,
		BackgroundPanel:     lipgloss.AdaptiveColor{Light: "#C0C0C0", Dark: "#333333"},
		BackgroundHighlight: lipgloss.AdaptiveColor{Light: "#FFFF00", Dark: "#0000FF"},
		BackgroundSelected:  lipgloss.AdaptiveColor{Light: "#00FFFF", Dark: "#000080"},

		// High contrast borders
		Border:       ColorHighContrastBorder,
		BorderFocus:  ColorHighContrastFocus,
		BorderActive: ColorHighContrastFocus,
		BorderMuted:  ColorHighContrastBorder,

		// High contrast progress
		Progress:         lipgloss.AdaptiveColor{Light: "#00FF00", Dark: "#00FF00"},
		ProgressBg:       lipgloss.AdaptiveColor{Light: "#808080", Dark: "#333333"},
		ProgressComplete: lipgloss.AdaptiveColor{Light: "#00FF00", Dark: "#00FF00"},
	}
	t.Styles = NewStyles(t)
	return t
}

// GetTheme returns a theme by name. Returns DefaultTheme if name is not recognized.
func GetTheme(name ThemeName) *Theme {
	switch name {
	case ThemeNVIDIADark:
		return DefaultTheme()
	case ThemeNVIDIALight:
		return LightTheme()
	case ThemeHighContrast:
		return HighContrastTheme()
	default:
		return DefaultTheme()
	}
}

// AvailableThemes returns a list of all available theme names.
func AvailableThemes() []ThemeName {
	return []ThemeName{
		ThemeNVIDIADark,
		ThemeNVIDIALight,
		ThemeHighContrast,
	}
}

// IsDarkTheme returns true if the theme is designed for dark backgrounds.
func (t *Theme) IsDarkTheme() bool {
	return t.Name == ThemeNVIDIADark || t.Name == ThemeHighContrast
}

// Copy returns a deep copy of the theme that can be modified independently.
func (t *Theme) Copy() *Theme {
	newTheme := *t
	newTheme.Styles = NewStyles(&newTheme)
	return &newTheme
}

// WithPrimary returns a copy of the theme with a custom primary color.
func (t *Theme) WithPrimary(primary lipgloss.Color) *Theme {
	newTheme := t.Copy()
	newTheme.Primary = primary
	newTheme.Styles = NewStyles(newTheme)
	return newTheme
}
