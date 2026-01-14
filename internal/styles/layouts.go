package styles

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Panel represents a panel in a layout
type Panel struct {
	Title   string
	Content string
	Style   lipgloss.Style
}

// SplitHorizontal creates a horizontal split layout (left | right)
func SplitHorizontal(left, right string, width, height int, ratio float64) string {
	if ratio <= 0 || ratio >= 1 {
		ratio = 0.5 // Default to 50/50
	}

	leftWidth := int(float64(width) * ratio)
	rightWidth := width - leftWidth - 2 // Account for borders

	leftBox := lipgloss.NewStyle().
		Width(leftWidth).
		Height(height).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(Purple).
		Padding(1).
		Render(left)

	rightBox := lipgloss.NewStyle().
		Width(rightWidth).
		Height(height).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(Cyan).
		Padding(1).
		Render(right)

	return lipgloss.JoinHorizontal(lipgloss.Top, leftBox, rightBox)
}

// SplitVertical creates a vertical split layout (top / bottom)
func SplitVertical(top, bottom string, width, height int, ratio float64) string {
	if ratio <= 0 || ratio >= 1 {
		ratio = 0.5
	}

	topHeight := int(float64(height) * ratio)
	bottomHeight := height - topHeight - 2 // Account for borders

	topBox := lipgloss.NewStyle().
		Width(width).
		Height(topHeight).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(Purple).
		Padding(1).
		Render(top)

	bottomBox := lipgloss.NewStyle().
		Width(width).
		Height(bottomHeight).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(Cyan).
		Padding(1).
		Render(bottom)

	return lipgloss.JoinVertical(lipgloss.Left, topBox, bottomBox)
}

// QuadLayout creates a 2x2 grid layout
// ┌─────────┬─────────┐
// │ TL      │ TR      │
// ├─────────┼─────────┤
// │ BL      │ BR      │
// └─────────┴─────────┘
func QuadLayout(tl, tr, bl, br string, width, height int) string {
	halfHeight := height / 2
	topRow := SplitHorizontal(tl, tr, width, halfHeight, 0.5)
	bottomRow := SplitHorizontal(bl, br, width, halfHeight, 0.5)
	return lipgloss.JoinVertical(lipgloss.Left, topRow, bottomRow)
}

// QuadLayoutWithTitles creates a 2x2 grid with panel titles
func QuadLayoutWithTitles(panels [4]Panel, width, height int) string {
	if len(panels) != 4 {
		return ""
	}

	halfWidth := width/2 - 2
	halfHeight := height/2 - 2

	// Create styled panels with titles
	tl := renderPanel(panels[0], halfWidth, halfHeight, Purple)
	tr := renderPanel(panels[1], halfWidth, halfHeight, Cyan)
	bl := renderPanel(panels[2], halfWidth, halfHeight, Green)
	br := renderPanel(panels[3], halfWidth, halfHeight, Yellow)

	return QuadLayout(tl, tr, bl, br, width, height)
}

// renderPanel creates a panel with title and content
func renderPanel(panel Panel, width, height int, borderColor lipgloss.Color) string {
	titleStyle := lipgloss.NewStyle().
		Foreground(borderColor).
		Bold(true).
		PaddingBottom(1)

	title := titleStyle.Render(panel.Title)

	// Truncate content if too long
	content := panel.Content
	lines := strings.Split(content, "\n")
	if len(lines) > height-3 {
		lines = lines[:height-3]
		lines = append(lines, DimStyle.Render("..."))
		content = strings.Join(lines, "\n")
	}

	body := lipgloss.JoinVertical(lipgloss.Left, title, content)

	return lipgloss.NewStyle().
		Width(width).
		Height(height).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Padding(1).
		Render(body)
}

// CenterText centers text horizontally within a given width
func CenterText(text string, width int) string {
	return lipgloss.NewStyle().
		Width(width).
		Align(lipgloss.Center).
		Render(text)
}

// CenterBlock centers a block of content both horizontally and vertically
func CenterBlock(content string, width, height int) string {
	return lipgloss.Place(width, height,
		lipgloss.Center, lipgloss.Center,
		content)
}

// ASCIIArt renders ASCII art centered
func ASCIIArt(text string, width int) string {
	style := lipgloss.NewStyle().
		Foreground(Purple).
		Bold(true).
		Width(width).
		Align(lipgloss.Center)
	return style.Render(text)
}

// HelpBar renders a help bar at the bottom of the screen
func HelpBar(hints []string, width int) string {
	separator := DimStyle.Render(" • ")
	helpText := strings.Join(hints, separator)

	return lipgloss.NewStyle().
		Foreground(Gray).
		Width(width).
		Align(lipgloss.Center).
		Padding(1, 0).
		Render(helpText)
}

// ListItem renders a list item with optional selection indicator
func ListItem(text string, selected bool) string {
	if selected {
		return SelectedItemStyle.Render("▸ " + text)
	}
	return UnselectedItemStyle.Render(text)
}

// ProgressBar renders a simple progress bar
func ProgressBar(current, total int, width int) string {
	if total == 0 {
		return ""
	}

	percentage := float64(current) / float64(total)
	filled := int(percentage * float64(width))

	bar := strings.Repeat("█", filled) + strings.Repeat("░", width-filled)

	return ProgressBarStyle.Render(bar)
}
