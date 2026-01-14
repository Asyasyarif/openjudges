package create

import (
	"fmt"
	"openjudges/config"
	"openjudges/internal/tui/components"
	"openjudges/judge"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
)

// Step represents a step in the wizard
type Step int

const (
	StepName Step = iota
	StepProvider
	StepModel
	StepAPIKey
	StepDataset
	StepConfirm
)

// Model is the create wizard model
type Model struct {
	CurrentStep Step
	Answers     config.JudgeConfig

	// Prompt data
	PromptTemplate *judge.PromptTemplate
	PromptVarInput map[string]string // Values for prompt variables
	PromptVarIndex int               // Current variable being edited

	// Components
	InputName      components.Input
	SelectProvider components.SimpleSelect
	SelectModel    components.SimpleSelect
	InputAPIKey    components.HiddenInput
	InputData      components.Input

	// Suggestions
	DatasetSuggestions      []string
	FilteredDataSuggestions []string
	SelectedDataSuggestion  int
	ShowDataSuggestions     bool

	// State
	IsEdit       bool
	OriginalName string
	Width        int
	Height       int
	Quitting     bool
	Done         bool
	Err          error
}

// NewModel creates a new Create Wizard model
func NewModel(existing *config.JudgeConfig) Model {
	m := Model{
		CurrentStep:    StepName,
		Answers:        config.JudgeConfig{},
		PromptVarInput: make(map[string]string),
	}

	if existing != nil {
		m.IsEdit = true
		m.OriginalName = existing.Name
		m.Answers = *existing
	}

	// Initialize inputs
	m.InputName = components.NewInputRequired("Judge Name", "e.g. gpt-4-judge")
	m.InputName = m.InputName.WithHelp("A unique name for this judge configuration")
	if m.IsEdit {
		m.InputName.SetValue(m.Answers.Name)
	}
	m.InputName.Focus()

	// Provider selection (SimpleSelect)
	providers := []components.SimpleSelectItem{
		{Title: "OpenAI", Description: "GPT-4, GPT-3.5, GPT-4o-Mini more", Value: "openai"},
		{Title: "Anthropic", Description: "Claude 3.5 Sonnet, Haiku, more", Value: "anthropic"},
		{Title: "Google", Description: "Gemini 1.5 Pro, Flash, more", Value: "google"},
		{Title: "Groq", Description: "Llama 3, Mixtral (Fast), more", Value: "groq"},
		{Title: "OpenRouter", Description: "Gateway LLM Provider", Value: "openrouter"},
	}

	// Add custom providers from config
	cfg, err := config.Load(filepath.Join(os.Getenv("HOME"), ".openjudges", "config.json"))
	if err == nil {
		for _, p := range cfg.CustomProviders {
			providers = append(providers, components.SimpleSelectItem{
				Title:       p.Provider,
				Description: fmt.Sprintf("Custom: %s", p.APIURL),
				Value:       p.Provider,
			})
		}
	}

	m.SelectProvider = components.NewSimpleSelect("Select LLM Provider", providers)
	if m.IsEdit {
		m.SelectProvider.SelectValue(m.Answers.LLM.Provider)
	}

	// API Key
	m.InputAPIKey = components.NewHiddenInput("API Key", "sk-...")
	m.InputAPIKey = m.InputAPIKey.WithHelp("Leave empty to use environment variables")
	if m.IsEdit {
		m.InputAPIKey.SetValue(m.Answers.LLM.APIKey)
	}

	// Initial model step
	m.initModelStep()
	if m.IsEdit {
		m.SelectModel.SelectValue(m.Answers.LLM.Model)
	}

	// Default Prompt
	m.Answers.PromptFile = "prompts/judges_prompt_dont_change.md"

	// Dataset input
	m.InputData = components.NewInputRequired("Dataset File", "type @ to see files...")
	m.InputData = m.InputData.WithHelp("Path to the dataset Excel file. Type @ to browse files in datasets/")
	if m.IsEdit {
		m.InputData.SetValue(m.Answers.DataPath)
	}

	return m
}

// Init implements tea.Model
func (m Model) Init() tea.Cmd {
	return m.InputName.Init()
}
