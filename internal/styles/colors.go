package styles

import "github.com/charmbracelet/lipgloss"

// Color definitions matching the current pterm color scheme
var (
	// Primary palette
	Purple   = lipgloss.Color("#A855F7") // Primary brand color
	Cyan     = lipgloss.Color("#06B6D4") // Info and highlights
	Green    = lipgloss.Color("#22C55E") // Success states
	Yellow   = lipgloss.Color("#EAB308") // Warning states
	Orange   = lipgloss.Color("#F97316") // Evaluating states
	Red      = lipgloss.Color("#EF4444") // Error states
	Gray     = lipgloss.Color("#6B7280") // Muted text
	DarkGray = lipgloss.Color("#374151") // Background elements
	White    = lipgloss.Color("#F9FAFB") // Primary text

	// Semantic color aliases for better readability
	ColorPrimary = Purple
	ColorInfo    = Cyan
	ColorSuccess = Green
	ColorWarning = Yellow
	ColorError   = Red
	ColorMuted   = Gray
	ColorText    = White
)

// AdaptiveColor returns a color that adapts to light/dark terminal backgrounds
func AdaptiveColor(light, dark string) lipgloss.AdaptiveColor {
	return lipgloss.AdaptiveColor{
		Light: light,
		Dark:  dark,
	}
}
