package components

import (
	"github.com/charmbracelet/lipgloss"
)

// LayoutType defines the split layout
type LayoutType int

const (
	LayoutHorizontal LayoutType = iota // Two panels side by side
	LayoutVertical                     // Two panels stacked
	LayoutGrid                         // 2x2 grid
)

// Panel represents a content area in the split view
type Panel struct {
	Title   string
	Content string
	Style   lipgloss.Style
}

// SplitView is a component that renders panels in a split layout
type SplitView struct {
	Panels []Panel
	Layout LayoutType
	Width  int
	Height int

	// Styles
	BorderColor lipgloss.Color
	ActiveColor lipgloss.Color
	ActivePanel int
}

// NewSplitView creates a new split view
func NewSplitView(width, height int) SplitView {
	return SplitView{
		Panels:      make([]Panel, 4), // Default 4 panels
		Layout:      LayoutGrid,
		Width:       width,
		Height:      height,
		BorderColor: lipgloss.Color("#6B7280"), // Gray
		ActiveColor: lipgloss.Color("#A855F7"), // Purple
	}
}

// UpdatePanel updates the content of a specific panel
func (s *SplitView) UpdatePanel(index int, title, content string) {
	if index >= 0 && index < len(s.Panels) {
		s.Panels[index].Title = title
		s.Panels[index].Content = content
	}
}

// SetActivePanel sets the active panel index
func (s *SplitView) SetActivePanel(index int) {
	if index >= 0 && index < len(s.Panels) {
		s.ActivePanel = index
	}
}

// View renders the split view
func (s SplitView) View() string {
	if s.Width == 0 || s.Height == 0 {
		return ""
	}

	switch s.Layout {
	case LayoutGrid:
		return s.renderGrid()
	case LayoutHorizontal:
		return s.renderHorizontal()
	case LayoutVertical:
		return s.renderVertical()
	default:
		return "Invalid layout"
	}
}

// renderGrid renders a 2x2 grid
func (s SplitView) renderGrid() string {
	if len(s.Panels) < 4 {
		return "Need 4 panels for grid layout"
	}

	// Calculate dimensions
	// W: [ P0 | P1 ]
	//    -------
	//    [ P2 | P3 ]

	halfWidth := s.Width / 2
	halfHeight := s.Height / 2

	// Adjust for borders
	w1 := halfWidth - 1    // Left column
	w2 := s.Width - w1 - 2 // Right column (-2 for borders)

	h1 := halfHeight - 1    // Top row
	h2 := s.Height - h1 - 2 // Bottom row

	// Render panels
	p0 := s.renderPanel(0, w1, h1)
	p1 := s.renderPanel(1, w2, h1)
	p2 := s.renderPanel(2, w1, h2)
	p3 := s.renderPanel(3, w2, h2)

	topRow := lipgloss.JoinHorizontal(lipgloss.Top, p0, p1)
	bottomRow := lipgloss.JoinHorizontal(lipgloss.Top, p2, p3)

	return lipgloss.JoinVertical(lipgloss.Left, topRow, bottomRow)
}

func (s SplitView) renderHorizontal() string {
	// [ P0 | P1 ]
	halfWidth := s.Width / 2
	w1 := halfWidth - 1
	w2 := s.Width - w1 - 2

	p0 := s.renderPanel(0, w1, s.Height-2)
	p1 := s.renderPanel(1, w2, s.Height-2)

	return lipgloss.JoinHorizontal(lipgloss.Top, p0, p1)
}

func (s SplitView) renderVertical() string {
	// [ P0 ]
	// ------
	// [ P1 ]
	halfHeight := s.Height / 2
	h1 := halfHeight - 1
	h2 := s.Height - h1 - 2

	p0 := s.renderPanel(0, s.Width-2, h1)
	p1 := s.renderPanel(1, s.Width-2, h2)

	return lipgloss.JoinVertical(lipgloss.Left, p0, p1)
}

func (s SplitView) renderPanel(index, w, h int) string {
	if w <= 0 || h <= 0 {
		return ""
	}

	panel := s.Panels[index]
	borderColor := s.BorderColor
	if index == s.ActivePanel {
		borderColor = s.ActiveColor
	}

	style := lipgloss.NewStyle().
		Width(w).
		Height(h).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Padding(0, 1)

	// Handle content overflow simply for now (wrapping default)
	// Truncate if too long?
	// Content rendering could be more sophisticated (e.g., viewport)

	// Ideally, we should use a viewport model here for scrolling,
	// but for now let's just render the string and let lipgloss handle wrapping

	content := panel.Content

	// If title exists, prepend it
	if panel.Title != "" {
		titleStyle := lipgloss.NewStyle().Bold(true).Foreground(borderColor)
		header := titleStyle.Render(panel.Title)
		content = header + "\n" + content
	}

	return style.Render(content)
}
