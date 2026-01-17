package root

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	bubbletea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Mode int

const (
	ModeBlank Mode = iota
	ModeInitializing
	ModeCommand
)

type CommandSuggestion struct {
	Name        string
	Description string
}

type Model struct {
	Mode         Mode
	Input        textinput.Model
	Width        int
	Height       int
	Suggestions  []CommandSuggestion
	Filtered     []CommandSuggestion
	Selected     int
	Quitting     bool
	ExecutedCmd  string
	InitMessages []string
	InitComplete bool
}

func NewModel() Model {
	suggestions := []CommandSuggestion{
		{Name: "create", Description: "Create a new judge configuration"},
		{Name: "run", Description: "Run an evaluation task"},
		{Name: "list", Description: "List all existing judge configurations"},
		{Name: "results", Description: "View evaluation results"},
		{Name: "vendor", Description: "Manage custom LLM vendors"},
		{Name: "auto-prompt", Description: "Autonomous Prompt Engineer - Automatically test and improve prompts"},
		{Name: "quit", Description: "Exit the application"},
	}

	// Create input with prompt and placeholder
	ti := textinput.New()
	ti.Prompt = "> "
	ti.Placeholder = "Type command or press /"
	ti.PlaceholderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	ti.Width = 50
	ti.Focus()

	return Model{
		Mode:         ModeBlank,
		Input:        ti,
		Suggestions:  suggestions,
		Filtered:     nil,
		InitMessages: []string{},
		InitComplete: false,
	}
}

func (m Model) Init() bubbletea.Cmd {
	return textinput.Blink
}

func (m *Model) initializeProject() {
}

func (m *Model) FilterSuggestions(val string) {
	if val == "" {
		m.Filtered = nil
		return
	}

	clean := strings.TrimSpace(val)
	search := strings.ToLower(strings.TrimPrefix(clean, "/"))
	search = strings.TrimSpace(search)

	if search == "" {
		if strings.HasPrefix(val, "/") {
			m.Filtered = m.Suggestions
		} else {
			m.Filtered = nil
		}
		return
	}

	var filtered []CommandSuggestion
	for _, s := range m.Suggestions {
		if strings.HasPrefix(strings.ToLower(s.Name), search) {
			filtered = append(filtered, s)
		}
	}

	if strings.HasPrefix("vendor create", search) && search != "vendor create" {
		filtered = append(filtered, CommandSuggestion{Name: "vendor create", Description: "Create a new custom vendor"})
	}
	if strings.HasPrefix("vendor list", search) && search != "vendor list" {
		filtered = append(filtered, CommandSuggestion{Name: "vendor list", Description: "List all saved custom vendors"})
	}

	if search == "vendor" {
		filtered = append(filtered, CommandSuggestion{Name: "vendor create", Description: "Create a new custom vendor"})
		filtered = append(filtered, CommandSuggestion{Name: "vendor list", Description: "List all saved custom vendors"})
	}
	m.Filtered = filtered
	m.Selected = 0
}
