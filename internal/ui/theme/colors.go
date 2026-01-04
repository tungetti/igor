// Package theme provides the theming and styling system for the Igor TUI.
// It includes NVIDIA-inspired color palettes, adaptive colors for light/dark
// terminal themes, and pre-built lipgloss styles for consistent UI appearance.
package theme

import "github.com/charmbracelet/lipgloss"

// NVIDIA-inspired primary colors
var (
	// NVIDIAGreen is the primary NVIDIA brand green color.
	NVIDIAGreen = lipgloss.Color("#76B900")

	// NVIDIAGreenDark is a darker variant of NVIDIA green for hover/active states.
	NVIDIAGreenDark = lipgloss.Color("#5A8F00")

	// NVIDIAGreenLight is a lighter variant of NVIDIA green for highlights.
	NVIDIAGreenLight = lipgloss.Color("#8BD000")

	// NVIDIABlack is the NVIDIA brand black color.
	NVIDIABlack = lipgloss.Color("#1A1A1A")

	// NVIDIAWhite is the NVIDIA brand white color.
	NVIDIAWhite = lipgloss.Color("#FFFFFF")

	// NVIDIAGray is a neutral gray for secondary elements.
	NVIDIAGray = lipgloss.Color("#666666")

	// NVIDIAGrayLight is a light gray for backgrounds.
	NVIDIAGrayLight = lipgloss.Color("#9CA3AF")

	// NVIDIAGrayDark is a dark gray for muted elements.
	NVIDIAGrayDark = lipgloss.Color("#404040")
)

// Semantic colors using AdaptiveColor for automatic light/dark theme support.
// AdaptiveColor automatically selects the appropriate color based on the
// terminal's background brightness.
var (
	// ColorSuccess represents successful operations (green tones).
	ColorSuccess = lipgloss.AdaptiveColor{Light: "#22C55E", Dark: "#4ADE80"}

	// ColorWarning represents warnings and cautions (yellow/amber tones).
	ColorWarning = lipgloss.AdaptiveColor{Light: "#EAB308", Dark: "#FACC15"}

	// ColorError represents errors and failures (red tones).
	ColorError = lipgloss.AdaptiveColor{Light: "#DC2626", Dark: "#F87171"}

	// ColorInfo represents informational messages (blue tones).
	ColorInfo = lipgloss.AdaptiveColor{Light: "#0EA5E9", Dark: "#38BDF8"}
)

// Text colors with adaptive support for readability across terminal themes.
var (
	// ColorText is the primary text color.
	ColorText = lipgloss.AdaptiveColor{Light: "#1F2937", Dark: "#F9FAFB"}

	// ColorTextMuted is for secondary/less important text.
	ColorTextMuted = lipgloss.AdaptiveColor{Light: "#6B7280", Dark: "#9CA3AF"}

	// ColorTextSubtle is for tertiary/placeholder text.
	ColorTextSubtle = lipgloss.AdaptiveColor{Light: "#9CA3AF", Dark: "#6B7280"}

	// ColorTextInverse is for text on colored backgrounds.
	ColorTextInverse = lipgloss.AdaptiveColor{Light: "#FFFFFF", Dark: "#1A1A1A"}
)

// Background colors with adaptive support.
var (
	// ColorBackground is the main background color.
	ColorBackground = lipgloss.AdaptiveColor{Light: "#FFFFFF", Dark: "#1A1A1A"}

	// ColorBackgroundAlt is an alternate background for visual separation.
	ColorBackgroundAlt = lipgloss.AdaptiveColor{Light: "#F3F4F6", Dark: "#262626"}

	// ColorBackgroundPanel is for panels and cards.
	ColorBackgroundPanel = lipgloss.AdaptiveColor{Light: "#E5E7EB", Dark: "#333333"}

	// ColorBackgroundHighlight is for highlighted/hovered items.
	ColorBackgroundHighlight = lipgloss.AdaptiveColor{Light: "#D1FAE5", Dark: "#064E3B"}

	// ColorBackgroundSelected is for selected items.
	ColorBackgroundSelected = lipgloss.AdaptiveColor{Light: "#DCFCE7", Dark: "#14532D"}
)

// Border colors with adaptive support.
var (
	// ColorBorder is the default border color.
	ColorBorder = lipgloss.AdaptiveColor{Light: "#D1D5DB", Dark: "#404040"}

	// ColorBorderFocus is for focused element borders (uses NVIDIA green).
	ColorBorderFocus = lipgloss.AdaptiveColor{Light: "#76B900", Dark: "#76B900"}

	// ColorBorderActive is for active element borders.
	ColorBorderActive = lipgloss.AdaptiveColor{Light: "#5A8F00", Dark: "#8BD000"}

	// ColorBorderMuted is for subtle/muted borders.
	ColorBorderMuted = lipgloss.AdaptiveColor{Light: "#E5E7EB", Dark: "#333333"}
)

// Progress and loading indicator colors.
var (
	// ColorProgress is the progress bar fill color (NVIDIA green).
	ColorProgress = lipgloss.AdaptiveColor{Light: "#76B900", Dark: "#76B900"}

	// ColorProgressBg is the progress bar background color.
	ColorProgressBg = lipgloss.AdaptiveColor{Light: "#E5E7EB", Dark: "#404040"}

	// ColorProgressComplete is for completed progress sections.
	ColorProgressComplete = lipgloss.AdaptiveColor{Light: "#22C55E", Dark: "#4ADE80"}
)

// High contrast colors for accessibility.
var (
	// ColorHighContrastText is high contrast text color.
	ColorHighContrastText = lipgloss.AdaptiveColor{Light: "#000000", Dark: "#FFFFFF"}

	// ColorHighContrastBg is high contrast background color.
	ColorHighContrastBg = lipgloss.AdaptiveColor{Light: "#FFFFFF", Dark: "#000000"}

	// ColorHighContrastBorder is high contrast border color.
	ColorHighContrastBorder = lipgloss.AdaptiveColor{Light: "#000000", Dark: "#FFFFFF"}

	// ColorHighContrastFocus is high contrast focus indicator.
	ColorHighContrastFocus = lipgloss.AdaptiveColor{Light: "#0000FF", Dark: "#FFFF00"}
)

// StatusColor represents a status indicator color.
type StatusColor string

const (
	// StatusSuccess indicates success.
	StatusSuccess StatusColor = "success"
	// StatusWarning indicates warning.
	StatusWarning StatusColor = "warning"
	// StatusError indicates error.
	StatusError StatusColor = "error"
	// StatusInfo indicates informational.
	StatusInfo StatusColor = "info"
)

// GetStatusColor returns the appropriate AdaptiveColor for a status.
func GetStatusColor(status StatusColor) lipgloss.AdaptiveColor {
	switch status {
	case StatusSuccess:
		return ColorSuccess
	case StatusWarning:
		return ColorWarning
	case StatusError:
		return ColorError
	case StatusInfo:
		return ColorInfo
	default:
		return ColorInfo
	}
}

// StatusIndicator returns a colored status indicator character.
func StatusIndicator(status StatusColor) string {
	style := lipgloss.NewStyle().Foreground(GetStatusColor(status))
	return style.Render("●")
}
