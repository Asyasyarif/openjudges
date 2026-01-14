package chat

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"openllmjudge/config"
	"openllmjudge/internal/styles"
	"openllmjudge/internal/tui/components"
	"openllmjudge/judge"
)

type Step int

const (
	StepSelectProvider Step = iota
	StepChat
)

type Message struct {
	Role    string
	Content string
}

type Model struct {
	Step           Step
	Config         *config.Config
	ProviderSelect components.SimpleSelect
	Input          textinput.Model
	Messages       []Message
	Loading        bool
	Error          error
	Width          int
	Height         int
	Spinner        spinner.Model
	Judge          *judge.Judge
	Quitting       bool
}

func NewModel(cfg *config.Config) Model {
	// Setup provider options from config and defaults
	var options []components.SimpleSelectItem
	for _, j := range cfg.Judges {
		options = append(options, components.SimpleSelectItem{
			Value:       j.Name,
			Title:       j.Name,
			Description: fmt.Sprintf("%s (%s)", j.LLM.Provider, j.LLM.Model),
		})
	}

	// Add manual entry option? Maybe later.

	ps := components.NewSimpleSelect("Select Judge to Chat With", options)

	ti := textinput.New()
	ti.Placeholder = "Type a message..."
	ti.Focus()

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(styles.Purple)

	return Model{
		Step:           StepSelectProvider,
		Config:         cfg,
		ProviderSelect: ps,
		Input:          ti,
		Spinner:        s,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.Spinner.Tick, textinput.Blink)
}

type chatResponseMsg struct {
	content string
	err     error
}

func (m *Model) sendChat() tea.Cmd {
	m.Loading = true
	m.Error = nil
	prompt := m.Input.Value()
	m.Messages = append(m.Messages, Message{Role: "user", Content: prompt})
	m.Input.SetValue("")

	return func() tea.Msg {
		ctx := context.Background()
		resp, err := m.Judge.Chat(ctx, prompt)
		return chatResponseMsg{content: resp, err: err}
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			if m.Step == StepSelectProvider {
				return m, tea.Quit
			}
			m.Step = StepSelectProvider
			return m, nil

		case "enter":
			if m.Step == StepSelectProvider {
				selectedValue := m.ProviderSelect.SelectedValue()
				if selectedValue != "" {
					// Find config
					var judgeCfg config.JudgeConfig
					for _, j := range m.Config.Judges {
						if j.Name == selectedValue {
							judgeCfg = j
							break
						}
					}
					m.Judge = judge.NewJudge(judgeCfg)
					m.Step = StepChat
					return m, nil
				}
			} else if m.Step == StepChat && !m.Loading && m.Input.Value() != "" {
				return m, m.sendChat()
			}
		}

	case chatResponseMsg:
		m.Loading = false
		if msg.err != nil {
			m.Error = msg.err
		} else {
			m.Messages = append(m.Messages, Message{Role: "assistant", Content: msg.content})
		}
		return m, nil

	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height

	case spinner.TickMsg:
		m.Spinner, cmd = m.Spinner.Update(msg)
		return m, cmd
	}

	if m.Step == StepSelectProvider {
		newPS, psCmd := m.ProviderSelect.Update(msg)
		m.ProviderSelect = newPS
		return m, psCmd
	} else {
		m.Input, cmd = m.Input.Update(msg)
		return m, cmd
	}
}

func (m Model) View() string {
	if m.Step == StepSelectProvider {
		return styles.DocStyle.Render(m.ProviderSelect.View())
	}

	// Chat View
	var sb strings.Builder
	sb.WriteString(styles.HeaderStyle.Render("Chatting with " + m.Judge.Name()))
	sb.WriteString("\n\n")

	// Render messages
	for _, msg := range m.Messages {
		role := lipgloss.NewStyle().Foreground(styles.Purple).Render("User: ")
		if msg.Role == "assistant" {
			role = styles.HighlightStyle.Render("AI:   ")
		}
		sb.WriteString(role + "\n")
		sb.WriteString(msg.Content + "\n\n")
	}

	if m.Loading {
		sb.WriteString(m.Spinner.View() + " Thinking...\n")
	}

	if m.Error != nil {
		sb.WriteString(styles.RenderError(m.Error.Error()) + "\n")
	}

	sb.WriteString("\n" + m.Input.View())
	sb.WriteString("\n" + styles.DimStyle.Render("(Esc to back, Ctrl+C to quit)"))

	return styles.DocStyle.Render(sb.String())
}
