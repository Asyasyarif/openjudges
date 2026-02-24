package root

import (
	"fmt"
	"openjudges/internal/styles"
	"openjudges/internal/tui/components"

	"github.com/charmbracelet/lipgloss"
)

// View renders the UI
// View renders the UI
func (m Model) View() string {
	if m.Quitting {
		return ""
	}

	if m.Mode == ModeInitializing {
		return m.renderInitializing()
	}

	header := m.renderHeader()
	tips := m.renderTips()
	input := m.renderInputBox()

	var footer string
	if m.Mode == ModeCommand {
		footer = m.renderSuggestions()
	} else {
		footer = styles.DimStyle.Render("? for help")
	}

	return styles.DocStyle.Render(lipgloss.JoinVertical(lipgloss.Left,
		header,
		"\n",
		tips,
		"\n",
		input,
		footer,
	))
}

func (m Model) renderHeader() string {
	return components.RenderHeader(m.Width)
}

func (m Model) renderTips() string {
	tips := []string{
		"Pro Tip for " + styles.HighlightStyle.Render("getting started:"),
		"1. Create a new judge using " + styles.HighlightStyle.Render("/create"),
		"2. Run evaluations using " + styles.HighlightStyle.Render("/run"),
		"3. View results using " + styles.HighlightStyle.Render("/results"),
	}
	return lipgloss.JoinVertical(lipgloss.Left, tips...)
}

func (m Model) renderInputBox() string {
	inputView := m.Input.View()
	w := m.Width
	if w <= 0 {
		w = 80
	}

	inputWidth := w - 6
	if inputWidth < 0 {
		inputWidth = 0
	}

	// Input with border
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.Purple).
		Padding(0, 1).
		Width(inputWidth).
		Render(inputView)
}

func (m Model) renderSuggestions() string {
	if len(m.Filtered) == 0 {
		return ""
	}

	var lines []string
	lines = append(lines, "\n"+styles.DimStyle.Render("Suggestions (↑/↓ navigate, Enter select):"))
	lines = append(lines, "")

	for i, s := range m.Filtered {
		name := s.Name
		desc := styles.DimStyle.Render(s.Description)

		var line string
		if i == m.Selected {
			line = styles.SelectedItemStyle.Render(fmt.Sprintf("▶ %-10s  %s", name, desc))
		} else {
			line = fmt.Sprintf("  %-10s  %s", name, desc)
		}
		lines = append(lines, line)
	}

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

// renderInitializing shows the initialization progress
func (m Model) renderInitializing() string {
	header := components.RenderHeader(m.Width)

	var lines []string
	lines = append(lines, header)
	lines = append(lines, "")

	for _, msg := range m.InitMessages {
		lines = append(lines, msg)
	}

	if !m.InitComplete {
		lines = append(lines, "")
		lines = append(lines, styles.DimStyle.Render("Initializing..."))
	}

	return styles.DocStyle.Render(lipgloss.JoinVertical(lipgloss.Left, lines...))
}
