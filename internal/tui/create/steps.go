package create

import (
	"fmt"
	"openllmjudge/config"
	"openllmjudge/internal/tui/components"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// Update handles events
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		m.updateComponentSizes()
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			m.Quitting = true
			return m, tea.Quit
		case "esc":
			return m.handleBack()
		}
	}

	// Step-specific updates
	switch m.CurrentStep {
	case StepName:
		m.InputName, cmd = m.InputName.Update(msg)
		if msgTypeIsEnter(msg) && m.InputName.Error() == nil {
			m.Answers.Name = m.InputName.Value()
			m.CurrentStep = StepProvider
			m.SelectProvider.Focus()
			return m, nil
		}
		return m, cmd

	case StepProvider:
		oldProvider := m.SelectProvider.SelectedValue()
		m.SelectProvider, cmd = m.SelectProvider.Update(msg)
		newProvider := m.SelectProvider.SelectedValue()
		if newProvider != "" && newProvider != oldProvider {
			m.Answers.LLM.Provider = newProvider
			m.initModelStep()
		}
		if msgTypeIsEnter(msg) && newProvider != "" {
			m.CurrentStep = StepModel
			m.SelectModel.Focus()
			return m, nil
		}
		return m, cmd

	case StepModel:
		m.SelectModel, cmd = m.SelectModel.Update(msg)
		if msgTypeIsEnter(msg) && m.SelectModel.SelectedValue() != "" {
			m.Answers.LLM.Model = m.SelectModel.SelectedValue()
			m.CurrentStep = StepAPIKey
			m.InputAPIKey.Focus()
			return m, nil
		}
		return m, cmd

	case StepAPIKey:
		m.InputAPIKey, cmd = m.InputAPIKey.Update(msg)
		if msgTypeIsEnter(msg) {
			m.Answers.LLM.APIKey = m.InputAPIKey.Value()
			m.CurrentStep = StepDataset
			m.InputData.Focus()
			return m, m.InputData.Init()
		}
		return m, cmd

	case StepDataset:
		if m.ShowDataSuggestions {
			switch msg := msg.(type) {
			case tea.KeyMsg:
				switch msg.String() {
				case "up", "k":
					if m.SelectedDataSuggestion > 0 {
						m.SelectedDataSuggestion--
					}
					return m, nil
				case "down", "j":
					if m.SelectedDataSuggestion < len(m.FilteredDataSuggestions)-1 {
						m.SelectedDataSuggestion++
					}
					return m, nil
				case "enter":
					if len(m.FilteredDataSuggestions) > 0 {
						suggestion := m.FilteredDataSuggestions[m.SelectedDataSuggestion]
						val := m.InputData.Value()
						lastAt := strings.LastIndex(val, "@")
						if lastAt >= 0 {
							newVal := val[:lastAt] + suggestion
							m.InputData.SetValue(newVal)
						}
						m.ShowDataSuggestions = false
						return m, nil
					}
				case "esc":
					m.ShowDataSuggestions = false
					return m, nil
				}
			}
		}

		m.InputData, cmd = m.InputData.Update(msg)
		val := m.InputData.Value()

		if strings.Contains(val, "@") {
			m.ShowDataSuggestions = true
			m.updateDatasetSuggestions()
		} else {
			m.ShowDataSuggestions = false
		}

		if msgTypeIsEnter(msg) && !m.ShowDataSuggestions && m.InputData.Error() == nil {
			m.Answers.DataPath = m.InputData.Value()
			m.CurrentStep = StepConfirm
			return m, nil
		}
		return m, cmd

	case StepConfirm:
		if msgTypeIsEnter(msg) {
			return m.submit()
		}
	}

	return m, nil
}

func (m Model) handleBack() (tea.Model, tea.Cmd) {
	switch m.CurrentStep {
	case StepName:
		m.Quitting = true
		return m, tea.Quit
	case StepProvider:
		m.CurrentStep = StepName
		m.InputName.Focus()
	case StepModel:
		m.CurrentStep = StepProvider
		m.SelectProvider.Focus()
	case StepAPIKey:
		m.CurrentStep = StepModel
		m.SelectModel.Focus()
	case StepDataset:
		m.CurrentStep = StepAPIKey
		m.InputAPIKey.Focus()
	case StepConfirm:
		m.CurrentStep = StepDataset
		m.InputData.Focus()
	}
	return m, nil
}

func (m *Model) updateComponentSizes() {
	w := m.Width - 6
	m.InputName.SetWidth(w)
	m.InputAPIKey.SetWidth(w)
	m.InputData.SetWidth(w)
}

func (m Model) submit() (tea.Model, tea.Cmd) {
	m.Answers.Name = m.InputName.Value()
	m.Answers.LLM.APIKey = m.InputAPIKey.Value()
	m.Answers.DataPath = m.InputData.Value()

	// If it's a custom provider, copy its details
	cfg, err := config.Load(filepath.Join(os.Getenv("HOME"), ".openllmjudge", "config.json"))
	if err == nil {
		for _, p := range cfg.CustomProviders {
			if p.Provider == m.Answers.LLM.Provider {
				m.Answers.LLM.Method = p.Method
				m.Answers.LLM.Headers = p.Headers
				m.Answers.LLM.APIURL = p.APIURL
				m.Answers.LLM.ResponsePath = p.ResponsePath
				m.Answers.LLM.BodyTemplate = p.BodyTemplate
				if m.Answers.LLM.APIKey == "" {
					m.Answers.LLM.APIKey = p.APIKey
				}
				goto saved
			}
		}
	}

	m.Answers.LLM.APIURL = config.DefaultLLMAPIURLs[m.Answers.LLM.Provider]

saved:

	// Default result path
	m.Answers.ResultPath = fmt.Sprintf("results/%s_results.json", m.Answers.Name)

	m.saveConfig()
	m.Done = true
	m.Quitting = true
	return m, tea.Quit
}

func msgTypeIsEnter(msg tea.Msg) bool {
	if key, ok := msg.(tea.KeyMsg); ok {
		return key.Type == tea.KeyEnter
	}
	return false
}

func (m *Model) initModelStep() {
	var items []components.SimpleSelectItem

	switch m.Answers.LLM.Provider {
	case "openai":
		items = []components.SimpleSelectItem{
			{Title: "GPT-4o", Description: "Flagship model", Value: "gpt-4o"},
			{Title: "GPT-4o Mini", Description: "Fast and cheap", Value: "gpt-4o-mini"},
			{Title: "GPT-4 Turbo", Description: "Previous flagship", Value: "gpt-4-turbo"},
			{Title: "GPT-3.5 Turbo", Description: "Legacy", Value: "gpt-3.5-turbo"},
		}
	case "anthropic":
		items = []components.SimpleSelectItem{
			{Title: "Claude 3.5 Sonnet", Description: "Best balance", Value: "claude-3-5-sonnet-20240620"},
			{Title: "Claude 3.5 Haiku", Description: "Fastest", Value: "claude-3-5-haiku-20241022"},
			{Title: "Claude 3 Opus", Description: "Most capable legacy", Value: "claude-3-opus-20240229"},
		}
	case "google":
		items = []components.SimpleSelectItem{
			{Title: "Gemini 1.5 Pro", Description: "Flagship", Value: "gemini-1.5-pro"},
			{Title: "Gemini 1.5 Flash", Description: "Low latency", Value: "gemini-1.5-flash"},
		}
	case "groq":
		items = []components.SimpleSelectItem{
			{Title: "Llama 3 70B", Description: "High performance", Value: "llama3-70b-8192"},
			{Title: "Mixtral 8x7B", Description: "Fast and reliable", Value: "mixtral-8x7b-32768"},
		}
	default:
		// Check if it's a custom provider
		cfg, err := config.Load(filepath.Join(os.Getenv("HOME"), ".openllmjudge", "config.json"))
		if err == nil {
			for _, p := range cfg.CustomProviders {
				if p.Provider == m.Answers.LLM.Provider {
					items = []components.SimpleSelectItem{
						{Title: p.Model, Description: "Custom configured model", Value: p.Model},
					}
					if p.Model == "" {
						items[0].Title = "Default"
						items[0].Value = "default"
					}
					goto found
				}
			}
		}
		items = []components.SimpleSelectItem{
			{Title: "Default", Description: "Default model", Value: "default"},
		}
	}

found:

	m.SelectModel = components.NewSimpleSelect("Select Model", items)
	m.SelectModel.SetWidth(m.Width - 6)
}

func (m *Model) updateDatasetSuggestions() {
	if len(m.DatasetSuggestions) == 0 {
		m.loadDatasetFiles()
	}

	val := m.InputData.Value()
	lastAt := strings.LastIndex(val, "@")
	if lastAt < 0 {
		m.FilteredDataSuggestions = nil
		m.ShowDataSuggestions = false
		return
	}

	query := strings.ToLower(val[lastAt+1:])
	m.FilteredDataSuggestions = nil
	for _, s := range m.DatasetSuggestions {
		if strings.Contains(strings.ToLower(s), query) {
			m.FilteredDataSuggestions = append(m.FilteredDataSuggestions, s)
		}
	}

	if m.SelectedDataSuggestion >= len(m.FilteredDataSuggestions) {
		m.SelectedDataSuggestion = 0
	}
}

func (m *Model) loadDatasetFiles() {
	cwd, _ := os.Getwd()
	datasetsDir := filepath.Join(cwd, "datasets")

	filepath.Walk(datasetsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() {
			ext := filepath.Ext(path)
			if ext == ".xlsx" || ext == ".xlsm" {
				rel, err := filepath.Rel(cwd, path)
				if err == nil {
					m.DatasetSuggestions = append(m.DatasetSuggestions, rel)
				}
			}
		}
		return nil
	})
}

func (m Model) saveConfig() {
	cfg, err := config.Load("config.json")
	if err != nil {
		cfg = &config.Config{Judges: []config.JudgeConfig{}}
	}
	if m.IsEdit {
		cfg.UpdateJudge(m.OriginalName, m.Answers)
	} else {
		cfg.Judges = append(cfg.Judges, m.Answers)
	}
	config.SaveConfig(cfg, "config.json")
}
