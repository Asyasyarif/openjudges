package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// ProgressBar is a reusable progress bar component
type ProgressBar struct {
	Percentage float64
	Width      int
	Color      lipgloss.Color
	ShowLabel  bool
	Label      string
}

// NewProgressBar creates a new progress bar
func NewProgressBar(width int) ProgressBar {
	return ProgressBar{
		Width:     width,
		Color:     lipgloss.Color("#A855F7"), // Purple
		ShowLabel: true,
		Label:     "",
	}
}

// SetProgress sets the current progress (0.0 - 1.0)
func (p *ProgressBar) SetProgress(percentage float64) {
	if percentage < 0 {
		percentage = 0
	}
	if percentage > 1 {
		percentage = 1
	}
	p.Percentage = percentage
}

// View returns the rendered progress bar
func (p ProgressBar) View() string {
	if p.Width < 4 {
		return ""
	}

	barWidth := p.Width - 2 // Minus brackets/padding
	if p.ShowLabel {
		labelWidth := len(p.Label) + 1 + 5 // Label + space + "100%"
		barWidth -= labelWidth
	}

	if barWidth < 1 {
		barWidth = 1
	}

	filledWidth := int(float64(barWidth) * p.Percentage)
	emptyWidth := barWidth - filledWidth

	filled := strings.Repeat("█", filledWidth)
	empty := strings.Repeat("░", emptyWidth)

	barStyle := lipgloss.NewStyle().Foreground(p.Color)
	emptyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#374151")) // Dark Gray

	bar := barStyle.Render(filled) + emptyStyle.Render(empty)

	if p.ShowLabel {
		percent := int(p.Percentage * 100)
		label := ""
		if p.Label != "" {
			label = p.Label + " "
		}
		return lipgloss.JoinHorizontal(lipgloss.Center,
			label,
			bar,
			fmt.Sprintf(" %d%%", percent),
		)
	}

	return bar
}
