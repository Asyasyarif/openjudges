package components

import (
	"fmt"
	"io"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"openjudges/internal/styles"
)

// SelectItem represents an item in the select list
type SelectItem struct {
	title       string
	description string
	value       string
	disabled    bool
}

// NewSelectItem creates a new select item
func NewSelectItem(title, description, value string) SelectItem {
	return SelectItem{
		title:       title,
		description: description,
		value:       value,
		disabled:    false,
	}
}

// NewDisabledSelectItem creates a disabled select item
func NewDisabledSelectItem(title, description string) SelectItem {
	return SelectItem{
		title:       title,
		description: description,
		disabled:    true,
	}
}

// Implement list.Item interface
func (i SelectItem) Title() string       { return i.title }
func (i SelectItem) Description() string { return i.description }
func (i SelectItem) FilterValue() string { return i.title }

// Value returns the item's value
func (i SelectItem) Value() string { return i.value }

// IsDisabled returns whether the item is disabled
func (i SelectItem) IsDisabled() bool { return i.disabled }

// SelectDelegate is a custom delegate for the select list
type SelectDelegate struct {
	list.DefaultDelegate
}

// NewSelectDelegate creates a new select delegate
func NewSelectDelegate() SelectDelegate {
	d := list.NewDefaultDelegate()
	d.ShowDescription = true
	d.SetHeight(2)

	// Customize styles
	d.Styles.SelectedTitle = lipgloss.NewStyle().
		Foreground(styles.Purple).
		Bold(true).
		BorderLeft(true).
		BorderStyle(lipgloss.ThickBorder()).
		BorderForeground(styles.Purple).
		Padding(0, 0, 0, 1)

	d.Styles.SelectedDesc = lipgloss.NewStyle().
		Foreground(styles.Cyan).
		BorderLeft(true).
		BorderStyle(lipgloss.ThickBorder()).
		BorderForeground(styles.Purple).
		Padding(0, 0, 0, 1)

	d.Styles.NormalTitle = lipgloss.NewStyle().
		Foreground(styles.White).
		Padding(0, 0, 0, 2)

	d.Styles.NormalDesc = lipgloss.NewStyle().
		Foreground(styles.Gray).
		Padding(0, 0, 0, 2)

	d.Styles.DimmedTitle = lipgloss.NewStyle().
		Foreground(styles.Gray).
		Padding(0, 0, 0, 2)

	d.Styles.DimmedDesc = lipgloss.NewStyle().
		Foreground(styles.DarkGray).
		Padding(0, 0, 0, 2)

	return SelectDelegate{DefaultDelegate: d}
}

// Render implements list.ItemDelegate
func (d SelectDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	if selectItem, ok := item.(SelectItem); ok {
		if selectItem.IsDisabled() {
			// Render disabled item
			title := d.Styles.DimmedTitle.Render(selectItem.Title())
			desc := d.Styles.DimmedDesc.Render(selectItem.Description())
			fmt.Fprintf(w, "%s\n%s", title, desc)
			return
		}
	}

	// Use default rendering for normal items
	d.DefaultDelegate.Render(w, m, index, item)
}

// Select is a reusable select component
type Select struct {
	list     list.Model
	title    string
	selected *SelectItem
	showBack bool
	focused  bool
}

// NewSelect creates a new select component
func NewSelect(title string, items []SelectItem, width, height int) Select {
	// Convert to list.Item
	listItems := make([]list.Item, len(items))
	for i, item := range items {
		listItems[i] = item
	}

	delegate := NewSelectDelegate()
	l := list.New(listItems, delegate, width, height)
	l.Title = title
	l.SetShowTitle(true)
	l.SetShowHelp(true)
	l.SetFilteringEnabled(true)
	l.SetShowStatusBar(true)

	// Customize help styles
	l.Styles.Title = lipgloss.NewStyle().
		Foreground(styles.Purple).
		Bold(true).
		Padding(0, 0, 1, 0)

	l.Styles.HelpStyle = lipgloss.NewStyle().
		Foreground(styles.Gray)

	return Select{
		list:     l,
		title:    title,
		showBack: false,
	}
}

// WithBackOption adds a "Back" option to the select
func (s Select) WithBackOption() Select {
	s.showBack = true

	// Prepend back option
	items := s.list.Items()
	backItem := NewSelectItem("← Back", "Return to previous menu", "__back__")
	newItems := append([]list.Item{backItem}, items...)
	s.list.SetItems(newItems)

	return s
}

func (s *Select) SetSize(width, height int) {
	s.list.SetSize(width, height)
}

// SetTitle updates the list title
func (s *Select) SetTitle(title string) {
	s.list.Title = title
}

// Focus focuses the component
func (s *Select) Focus() {
	s.focused = true
}

// Blur blurs the component
func (s *Select) Blur() {
	s.focused = false
}

// IsFocused returns whether the component is focused
func (s Select) IsFocused() bool {
	return s.focused
}

// SelectedItem returns the currently selected item
func (s Select) SelectedItem() *SelectItem {
	if s.selected != nil {
		return s.selected
	}

	item := s.list.SelectedItem()
	if selectItem, ok := item.(SelectItem); ok {
		return &selectItem
	}

	return nil
}

// IsBackSelected returns true if the back option was selected
func (s Select) IsBackSelected() bool {
	item := s.SelectedItem()
	return item != nil && item.Value() == "__back__"
}

// Init implements tea.Model
func (s Select) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model
func (s Select) Update(msg tea.Msg) (Select, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			// Store selected item
			item := s.list.SelectedItem()
			if selectItem, ok := item.(SelectItem); ok {
				if selectItem.IsDisabled() {
					// Don't allow selecting disabled items
					return s, nil
				}
				s.selected = &selectItem
			}
			return s, nil

		case "esc":
			// If back option exists, select it
			if s.showBack {
				items := s.list.Items()
				if len(items) > 0 {
					if backItem, ok := items[0].(SelectItem); ok && backItem.Value() == "__back__" {
						s.selected = &backItem
						return s, tea.Quit
					}
				}
			}
			return s, tea.Quit
		}

	case tea.WindowSizeMsg:
		s.list.SetSize(msg.Width, msg.Height-4)
	}

	var cmd tea.Cmd
	s.list, cmd = s.list.Update(msg)
	return s, cmd
}

// View implements tea.Model
func (s Select) View() string {
	view := s.list.View()
	if s.focused {
		return lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(styles.Purple).
			Padding(0, 1).
			Render(view)
	}
	return lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(styles.DarkGray).
		Padding(0, 1).
		Render(view)
}

// SelectedValue returns the value of the currently selected item
func (s Select) SelectedValue() string {
	item := s.SelectedItem()
	if item != nil {
		return item.Value()
	}
	return ""
}
