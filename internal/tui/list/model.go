package list

import (
	"fmt"

	"openjudges/config"
	"openjudges/internal/styles"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Mode represents the current state of the TUI
type Mode int

const (
	ModeNormal Mode = iota
	ModeSearch
)

// Item implements list.Item
type Item struct {
	Config config.JudgeConfig
}

func (i Item) Title() string { return i.Config.Name }
func (i Item) Description() string {
	return fmt.Sprintf("%s / %s • Data: %s",
		i.Config.LLM.Provider,
		i.Config.LLM.Model,
		i.Config.DataPath,
	)
}
func (i Item) FilterValue() string { return i.Config.Name + " " + i.Config.LLM.Provider }

// Model for the list TUI
type Model struct {
	list       list.Model
	cfg        *config.Config
	configPath string
	EditResult *config.JudgeConfig // Selected judge for editing

	// Search
	mode        Mode
	searchInput textinput.Model

	// State
	quitting bool
	width    int
	height   int
}

// NewModel creates a new list model
func NewModel(cfg *config.Config, configPath string) Model {
	items := make([]list.Item, len(cfg.Judges))
	for i, j := range cfg.Judges {
		items[i] = Item{Config: j}
	}

	// Create list delegate
	delegate := list.NewDefaultDelegate()

	// Simple list styles without border boxes
	delegate.Styles.SelectedTitle = lipgloss.NewStyle().
		Foreground(styles.Purple).
		Bold(true).
		Padding(0, 0, 0, 1)

	delegate.Styles.SelectedDesc = delegate.Styles.SelectedTitle.Copy().
		Foreground(styles.Gray).
		Bold(false)

	delegate.Styles.NormalTitle = lipgloss.NewStyle().
		Foreground(styles.White).
		Padding(0, 0, 0, 2)

	delegate.Styles.NormalDesc = delegate.Styles.NormalTitle.Copy().
		Foreground(styles.Gray)

	l := list.New(items, delegate, 0, 0)
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)
	l.SetShowFilter(false) // We use our own search UI
	l.SetShowTitle(false)  // Custom header instead
	l.SetShowHelp(false)   // Custom help at the bottom

	// Style the list components
	l.Styles.HelpStyle = styles.DimStyle.Copy().PaddingLeft(2)
	l.Styles.FilterPrompt = styles.HighlightStyle
	l.Styles.FilterCursor = styles.HighlightStyle

	// Setup search input
	ti := textinput.New()
	ti.Placeholder = "Search judges..."
	ti.PlaceholderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	ti.Prompt = ""  // Handled in View
	ti.Width = 40   // Initial width
	ti.SetValue("") // Explicitly clear any potential initial value

	l.SetFilterText("") // Explicitly clear list filter

	return Model{
		list:        l,
		cfg:         cfg,
		configPath:  configPath,
		mode:        ModeNormal,
		searchInput: ti,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		textinput.Blink,
		m.searchInput.Focus(),
	)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		h, v := styles.DocStyle.GetFrameSize()
		// Available width inside DocStyle
		availWidth := msg.Width - h
		// Search input width should be box width minus padding
		m.searchInput.Width = availWidth - 4

		// Adjust list height:
		// Header (Title + 2-line Search) ~ 5-6 lines
		// Status/Help ~ 2 lines
		m.list.SetSize(availWidth, msg.Height-v-8)
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			m.quitting = true
			return m, tea.Quit

		case "esc":
			if m.searchInput.Value() != "" {
				m.searchInput.SetValue("")
				m.list.SetFilterText("")
				return m, nil
			}
			m.quitting = true
			return m, tea.Quit

		case "up", "down", "pgup", "pgdown":
			// Explicitly handle list navigation
			m.list, cmd = m.list.Update(msg)
			return m, cmd

		case "enter":
			if len(m.list.VisibleItems()) > 0 {
				selected := m.list.SelectedItem().(Item)
				m.EditResult = &selected.Config
				m.quitting = true
				return m, tea.Quit
			}
			return m, nil

		case "d":
			if m.searchInput.Value() == "" { // Only delete when not searching
				if len(m.list.VisibleItems()) > 0 {
					idx := m.list.Index()
					selected := m.list.SelectedItem().(Item)
					if m.deleteJudge(selected.Config.Name) {
						m.list.RemoveItem(idx)
						m.list.ResetSelected()
					}
				}
				return m, nil
			}
		}

		// Handle other keys for search
		var tiCmd tea.Cmd
		m.searchInput, tiCmd = m.searchInput.Update(msg)
		m.list.SetFilterText(m.searchInput.Value())
		return m, tiCmd
	}

	return m, nil
}

func (m Model) View() string {
	if m.quitting {
		return ""
	}

	w := m.width
	if w <= 0 {
		w = 80
	}

	h, _ := styles.DocStyle.GetFrameSize()
	availWidth := w - h

	// 1. Header (Title + Search Box)
	title := styles.HeaderStyle.Render("JUDGE LIST")

	searchStyle := lipgloss.NewStyle().
		Foreground(styles.Gray).
		Bold(true)

	searchInputBox := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(styles.DarkGray).
		Padding(0, 1).
		Width(availWidth) // Use calculated width

	if m.searchInput.Focused() {
		searchInputBox = searchInputBox.BorderForeground(styles.Purple)
	}

	searchView := lipgloss.JoinVertical(lipgloss.Left,
		searchStyle.Render("SEARCH"),
		searchInputBox.Render(m.searchInput.View()),
	)

	header := lipgloss.JoinVertical(lipgloss.Left,
		title,
		"\n",
		searchView,
	)

	// 2. List or Empty State
	var listView string
	if len(m.list.Items()) == 0 {
		listView = lipgloss.NewStyle().
			Foreground(styles.Gray).
			Padding(4, 2).
			Render(
				lipgloss.JoinVertical(lipgloss.Left,
					styles.HighlightStyle.Render("No judges found."),
					"Type "+styles.HighlightStyle.Render("/create")+" to add your first judge.",
					"",
					styles.HighlightStyle.Render("Tips for getting started:"),
					"1. Create .md file for prompt and store in prompts folder",
					"2. Create a new judge using /create",
					"3. Run evaluations using /run",
					"4. View results using /results",
				),
			)
	} else {
		listView = m.list.View()
	}

	// 3. Footer / Help
	help := styles.DimStyle.Render("nav: ↑/↓ • edit: enter • delete: d (when empty search) • quit: esc")

	return styles.DocStyle.Render(lipgloss.JoinVertical(lipgloss.Left,
		header,
		"\n",
		listView,
		"\n",
		help,
	))
}

func (m Model) GetEditResult() *config.JudgeConfig {
	return m.EditResult
}

// Helper to delete judge from config
func (m *Model) deleteJudge(name string) bool {
	// Find index
	idx := -1
	for i, j := range m.cfg.Judges {
		if j.Name == name {
			idx = i
			break
		}
	}

	if idx != -1 {
		// Remove from slice
		m.cfg.Judges = append(m.cfg.Judges[:idx], m.cfg.Judges[idx+1:]...)

		// Save to file
		if err := config.SaveConfig(m.cfg, m.configPath); err != nil {
			return false
		}
		return true
	}
	return false
}
