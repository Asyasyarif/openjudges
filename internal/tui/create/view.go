package create

import (
	"fmt"
	"openllmjudge/internal/styles"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/tree"
)

// View renders the Create Wizard
func (m Model) View() string {
	if m.Quitting {
		return ""
	}

	var component string
	var title string

	switch m.CurrentStep {
	case StepName:
		title = "EnterJudge Name"
		component = m.InputName.View()
	case StepProvider:
		title = "Select Provider"
		component = m.SelectProvider.View()
	case StepModel:
		title = "Select Model"
		component = m.SelectModel.View()
	case StepAPIKey:
		title = "Enter API Key"
		component = m.InputAPIKey.View()
	case StepDataset:
		title = "Select Dataset (.xlsx only)"
		component = lipgloss.JoinVertical(lipgloss.Left,
			m.InputData.View(),
			m.renderDataSuggestions(),
		)
	case StepConfirm:
		title = "Confirm Judge Details"
		component = m.renderConfirmation()
	}

	header := m.renderStepHeader(title)
	help := m.renderHelp()

	return styles.DocStyle.Render(lipgloss.JoinVertical(lipgloss.Left,
		styles.HeaderStyle.Render("Create Judge"),
		"\n",
		header,
		"\n",
		component,
		"\n",
		help,
	))
}

func (m Model) renderDataSuggestions() string {
	if !m.ShowDataSuggestions || len(m.FilteredDataSuggestions) == 0 {
		return ""
	}

	var lines []string
	lines = append(lines, "\n"+styles.DimStyle.Render("Suggestions (↑/↓ navigate, Enter select):"))
	lines = append(lines, "")

	for i, s := range m.FilteredDataSuggestions {
		if i == m.SelectedDataSuggestion {
			lines = append(lines, styles.SelectedItemStyle.Render("▶ "+s))
		} else {
			lines = append(lines, styles.UnselectedItemStyle.Render(s))
		}
	}
	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

func (m Model) renderStepHeader(title string) string {
	w := m.Width
	if w <= 0 {
		w = 80
	}
	// DocStyle uses Margin(1, 2), so available width is w - 4
	availableWidth := w - 4

	stepNum := int(m.CurrentStep) + 1
	totalSteps := 6

	progress := styles.DimStyle.Render(fmt.Sprintf("Step %d/%d", stepNum, totalSteps))

	titleStyled := lipgloss.NewStyle().
		Bold(true).
		Foreground(styles.Purple).
		Render(title)

	// Join horizontal with space between title and progress
	return lipgloss.JoinHorizontal(lipgloss.Top,
		titleStyled,
		lipgloss.NewStyle().Width(availableWidth-lipgloss.Width(titleStyled)).Align(lipgloss.Right).Render(progress),
	)
}

func (m Model) renderHelp() string {
	items := []string{
		"enter: next",
		"esc: back",
		"ctrl+c: quit",
	}

	return styles.DimStyle.Render(" " + strings.Join(items, " • ") + " ")
}

func (m Model) renderConfirmation() string {
	t := tree.New().
		Root(styles.HighlightStyle.Render(m.Answers.Name)).
		Child(
			fmt.Sprintf("LLM Vendor: %s", styles.ValueStyle.Render(m.Answers.LLM.Provider)),
			fmt.Sprintf("LLM Model:  %s", styles.ValueStyle.Render(m.Answers.LLM.Model)),
		)

	t.Child(
		fmt.Sprintf("Dataset:    %s", styles.ValueStyle.Render(m.Answers.DataPath)),
	)

	w := m.Width - 4
	if w < 0 {
		w = 0
	}

	return lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(styles.Purple).
		Padding(1, 2).
		Width(w).
		Render(lipgloss.JoinVertical(lipgloss.Left,
			t.String(),
			"\n"+styles.DimStyle.Render("Press enter to save Judges"),
		))
}
