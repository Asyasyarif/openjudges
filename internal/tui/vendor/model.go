package vendor

import (
	"openllmjudge/config"
	"openllmjudge/internal/tui/components"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Step int

const (
	StepMenu Step = iota
	StepCreateWizard
	StepList
	StepView
)

type Model struct {
	Step   Step
	Config *config.Config
	Menu   components.SimpleSelect

	// Sub-models
	CreateModel CreateModel
	ListModel   ListModel
	ViewModel   ViewModel

	Width    int
	Height   int
	Quitting bool
}

func NewModel(cfg *config.Config, initialStep Step) Model {
	m := Model{
		Step:   initialStep,
		Config: cfg,
	}

	items := []components.SimpleSelectItem{
		{Title: "Create New Vendor", Description: "Add a custom LLM API provider", Value: "create"},
		{Title: "List Vendors", Description: "View and manage configured custom vendors", Value: "list"},
	}
	m.Menu = components.NewSimpleSelect("Vendor Management", items)
	m.Menu.Focus()

	// Pre-initialize sub-models if starting in a specific step
	if m.Step == StepCreateWizard {
		m.CreateModel = NewCreateModel(m.Config, nil)
	} else if m.Step == StepList {
		m.ListModel = NewListModel(m.Config)
	}

	return m
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			m.Quitting = true
			return m, tea.Quit
		}
		if msg.String() == "esc" {
			if m.Step == StepMenu {
				return m, tea.Quit
			}
			if m.Step == StepView {
				// Go back to list
				m.Step = StepList
				m.ListModel.refreshList()
				return m, nil
			}
			// For other steps (CreateWizard, etc.), go back to menu
			m.Step = StepMenu
			m.Menu.Focus()
			return m, nil
		}

	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		m.Menu.SetWidth(msg.Width)
		m.CreateModel.Width = msg.Width
		m.CreateModel.Height = msg.Height
		m.ListModel.Width = msg.Width
		m.ListModel.Height = msg.Height
		m.ListModel.List.SetWidth(msg.Width)
	}

	switch m.Step {
	case StepMenu:
		m.Menu, cmd = m.Menu.Update(msg)
		if msgTypeIsEnter(msg) {
			val := m.Menu.SelectedValue()
			if val == "create" {
				m.Step = StepCreateWizard
				m.CreateModel = NewCreateModel(m.Config, nil)
				m.CreateModel.Width = m.Width
				m.CreateModel.Height = m.Height
				return m, m.CreateModel.Init()
			} else if val == "list" {
				m.Step = StepList
				m.ListModel = NewListModel(m.Config)
				m.ListModel.Width = m.Width
				m.ListModel.Height = m.Height
				return m, m.ListModel.Init()
			}
		}
		return m, cmd

	case StepCreateWizard:
		m.CreateModel, cmd = m.CreateModel.Update(msg)
		if m.CreateModel.Done {
			// Save to config
			origName := ""
			if m.CreateModel.IsEdit {
				origName = m.CreateModel.OriginalName
			}
			m.Config.UpdateCustomProvider(origName, m.CreateModel.Answers)
			config.SaveConfig(m.Config, "config.json")
			m.Step = StepMenu
			m.Menu.Focus()
		}
		return m, cmd

	case StepList:
		m.ListModel, cmd = m.ListModel.Update(msg)
		if m.ListModel.Selected != nil {
			// Transition to view step
			m.Step = StepView
			m.ViewModel = NewViewModel(*m.ListModel.Selected)
			m.ViewModel.Width = m.Width
			m.ViewModel.Height = m.Height
			m.ListModel.Selected = nil
			return m, nil
		}
		return m, cmd

	case StepView:
		m.ViewModel, cmd = m.ViewModel.Update(msg)
		// Esc is handled at top level, just process ViewModel
		return m, cmd
	}

	return m, nil
}

func (m Model) View() string {
	if m.Quitting {
		return ""
	}

	var content string
	switch m.Step {
	case StepMenu:
		content = m.Menu.View()
	case StepCreateWizard:
		content = m.CreateModel.View()
	case StepList:
		content = m.ListModel.View()
	case StepView:
		content = m.ViewModel.View()
	}

	// For absolute full width, we avoid all DocStyle or parent padding/margins.
	// We just render the content directly with a small top/bottom padding.
	return lipgloss.NewStyle().
		Padding(1, 0, 1, 0).
		Render(content)
}
