package root

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// Mode represents the current state of the root TUI
type Mode int

const (
	ModeBlank Mode = iota
	ModeCommand
)

// CommandSuggestion represents a command with a description
type CommandSuggestion struct {
	Name        string
	Description string
}

// Model holds the state for the root TUI
type Model struct {
	Mode        Mode
	Input       textinput.Model
	Width       int
	Height      int
	Suggestions []CommandSuggestion
	Filtered    []CommandSuggestion
	Selected    int
	Quitting    bool
	ExecutedCmd string // Command to execute after TUI exits
}

// NewModel creates a new root model
func NewModel() Model {
	ti := textinput.New()
	ti.Placeholder = "Type a command or '/' to show all menu"
	ti.Prompt = "► "
	ti.CharLimit = 50
	ti.Width = 100

	// Default suggestions
	suggestions := []CommandSuggestion{
		{Name: "create", Description: "Create a new judge configuration"},
		{Name: "run", Description: "Run an evaluation task"},
		{Name: "list", Description: "List all existing judge configurations"},
		{Name: "results", Description: "View evaluation results"},
		{Name: "vendor", Description: "Manage custom LLM vendors"},
		{Name: "auto-prompt", Description: "Autonomous Prompt Engineer - Automatically test and improve prompts"},
		{Name: "quit", Description: "Exit the application"},
	}

	return Model{
		Mode:        ModeBlank,
		Input:       ti,
		Suggestions: suggestions,
		Filtered:    nil, // Start with no suggestions visible
	}
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return textinput.Blink
}

// FilterSuggestions filters the suggestion list based on input
func (m *Model) FilterSuggestions(val string) {
	if val == "" {
		m.Filtered = nil
		return
	}

	// Clean input: trim spaces and leading slash
	clean := strings.TrimSpace(val)
	search := strings.ToLower(strings.TrimPrefix(clean, "/"))
	search = strings.TrimSpace(search)

	if search == "" {
		// If just "/", show all
		if strings.HasPrefix(val, "/") {
			m.Filtered = m.Suggestions
		} else {
			m.Filtered = nil
		}
		return
	}

	var filtered []CommandSuggestion
	for _, s := range m.Suggestions {
		// All suggestions are lowercase in model init
		if strings.HasPrefix(strings.ToLower(s.Name), search) {
			filtered = append(filtered, s)
		}
	}

	// Contextual suggestions for /vendor subcommands
	if strings.HasPrefix("vendor create", search) && search != "vendor create" {
		filtered = append(filtered, CommandSuggestion{Name: "vendor create", Description: "Create a new custom vendor"})
	}
	if strings.HasPrefix("vendor list", search) && search != "vendor list" {
		filtered = append(filtered, CommandSuggestion{Name: "vendor list", Description: "List all saved custom vendors"})
	}

	// If the user hasn't typed anything else after "vendor", show sub-commands as options
	if search == "vendor" {
		// Check if not already added by prefix (it should be, but let's be explicit and put them first or second)
		filtered = append(filtered, CommandSuggestion{Name: "vendor create", Description: "Create a new custom vendor"})
		filtered = append(filtered, CommandSuggestion{Name: "vendor list", Description: "List all saved custom vendors"})
	}

	m.Filtered = filtered
	m.Selected = 0 // Reset selection
}
