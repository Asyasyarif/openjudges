package components

import (
	"fmt"
	"openjudges/internal/styles"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// SimpleSelectItem represents an item in the simple select
type SimpleSelectItem struct {
	Title       string
	Description string
	Value       string
}

// SimpleSelect is a lightweight selection component
type SimpleSelect struct {
	Title    string
	Items    []SimpleSelectItem
	Cursor   int
	Selected *SimpleSelectItem
	Focused  bool
	Width    int
}

// NewSimpleSelect creates a new simple select component
func NewSimpleSelect(title string, items []SimpleSelectItem) SimpleSelect {
	return SimpleSelect{
		Title: title,
		Items: items,
	}
}

// Focus focuses the component
func (s *SimpleSelect) Focus() tea.Cmd {
	s.Focused = true
	return nil
}

// Blur blurs the component
func (s *SimpleSelect) Blur() {
	s.Focused = false
}

// Update handles events
func (s SimpleSelect) Update(msg tea.Msg) (SimpleSelect, tea.Cmd) {
	if !s.Focused {
		return s, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if s.Cursor > 0 {
				s.Cursor--
			}
		case "down", "j":
			if s.Cursor < len(s.Items)-1 {
				s.Cursor++
			}
		case "enter":
			if len(s.Items) > 0 {
				s.Selected = &s.Items[s.Cursor]
			}
		}
	}
	return s, nil
}

// View renders the component
func (s *SimpleSelect) SelectValue(val string) {
	for i, item := range s.Items {
		if item.Value == val {
			s.Cursor = i
			return
		}
	}
}

func (s SimpleSelect) View() string {
	var b strings.Builder

	// Header
	if s.Title != "" {
		b.WriteString(styles.HeaderStyle.MarginBottom(0).Render(s.Title) + "\n\n")
	}

	// Calculate max title width for alignment
	maxTitleWidth := 0
	for _, item := range s.Items {
		if len(item.Title) > maxTitleWidth {
			maxTitleWidth = len(item.Title)
		}
	}

	for i, item := range s.Items {
		cursor := "  "
		style := lipgloss.NewStyle()

		if s.Cursor == i && s.Focused {
			cursor = lipgloss.NewStyle().Foreground(styles.Purple).Render("▸ ")
			style = style.Foreground(styles.Purple).Bold(true)
		} else {
			cursor = "  "
			style = style.Foreground(styles.Gray)
		}

		title := item.Title
		// Pad title to maxTitleWidth
		paddedTitle := fmt.Sprintf("%-*s", maxTitleWidth, title)

		line := lipgloss.JoinHorizontal(lipgloss.Center,
			cursor,
			style.Render(paddedTitle),
		)

		if item.Description != "" {
			desc := "    " + styles.DimStyle.Render(item.Description)
			line += desc
		}

		b.WriteString(line + "\n")
	}

	content := b.String()

	if s.Width > 0 {
		// Full width with border
		style := lipgloss.NewStyle().
			Padding(1, 0).
			Width(s.Width).
			MaxWidth(s.Width).
			Border(lipgloss.RoundedBorder())
		return style.Render(content)
	}

	return content
}

// SelectedValue returns the value of the selected item
func (s SimpleSelect) SelectedValue() string {
	if s.Selected != nil {
		return s.Selected.Value
	}
	return ""
}

// SetWidth sets the component width
func (s *SimpleSelect) SetWidth(w int) {
	s.Width = w
}
