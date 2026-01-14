package vendor

import (
	"encoding/json"
	"openjudges/config"
	"openjudges/internal/styles"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type VendorJSONConfig struct {
	Name         string            `json:"name"`
	URL          string            `json:"url"`
	Method       string            `json:"method"`
	Headers      map[string]string `json:"headers"`
	Body         interface{}       `json:"body"`
	ResponsePath string            `json:"response_path"`
	APIKey       string            `json:"api_key"`
}

type CreateModel struct {
	Config   *config.Config
	TextArea textarea.Model
	Error    string

	Width        int
	Height       int
	IsEdit       bool
	OriginalName string
	Done         bool
	Answers      config.LLMConfig
}

func NewCreateModel(cfg *config.Config, existing *config.LLMConfig) CreateModel {
	ta := textarea.New()
	ta.Placeholder = "{\n  \"name\": \"my-vendor\",\n  \"url\": \"...\"\n}"
	ta.Focus()
	ta.SetHeight(15)

	m := CreateModel{
		Config:   cfg,
		TextArea: ta,
	}

	if existing != nil {
		m.IsEdit = true
		m.OriginalName = existing.Provider
		m.Answers = *existing

		// Pre-fill with existing config as JSON
		jsonConfig := VendorJSONConfig{
			Name:         existing.Provider,
			URL:          existing.APIURL,
			Method:       existing.Method,
			Headers:      existing.Headers,
			Body:         nil, // BodyTemplate needs string vs map handling
			ResponsePath: existing.ResponsePath,
			APIKey:       existing.APIKey,
		}

		if existing.BodyTemplate != "" {
			var bodyMap interface{}
			if err := json.Unmarshal([]byte(existing.BodyTemplate), &bodyMap); err == nil {
				jsonConfig.Body = bodyMap
			} else {
				jsonConfig.Body = existing.BodyTemplate
			}
		}

		b, _ := json.MarshalIndent(jsonConfig, "", "  ")
		m.TextArea.SetValue(string(b))
	}

	return m
}

func (m CreateModel) Init() tea.Cmd {
	return textarea.Blink
}

func (m CreateModel) Update(msg tea.Msg) (CreateModel, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		m.TextArea.SetWidth(msg.Width - 4)
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+s":
			return m.submit()
		case "esc":
			// Cancel
		}
	}

	m.TextArea, cmd = m.TextArea.Update(msg)

	// Ensure width is always set for the internal textarea
	// Interior width = TerminalWidth - Padding(2) - Border(2)
	if m.Width > 0 {
		m.TextArea.SetWidth(m.Width - 4)
	}

	// Live validation
	var jcfg VendorJSONConfig
	if err := json.Unmarshal([]byte(m.TextArea.Value()), &jcfg); err != nil {
		m.Error = "Invalid JSON: " + err.Error()
	} else {
		m.Error = ""
	}

	return m, cmd
}

func (m CreateModel) submit() (CreateModel, tea.Cmd) {
	var jcfg VendorJSONConfig
	err := json.Unmarshal([]byte(m.TextArea.Value()), &jcfg)
	if err != nil {
		m.Error = "Invalid JSON: " + err.Error()
		return m, nil
	}

	if jcfg.Name == "" {
		m.Error = "Field 'name' is required"
		return m, nil
	}
	if jcfg.URL == "" {
		m.Error = "Field 'url' is required"
		return m, nil
	}

	m.Answers.Provider = jcfg.Name
	m.Answers.APIURL = jcfg.URL
	m.Answers.Method = jcfg.Method
	if m.Answers.Method == "" {
		m.Answers.Method = "POST"
	}
	m.Answers.Headers = jcfg.Headers
	m.Answers.ResponsePath = jcfg.ResponsePath
	m.Answers.APIKey = jcfg.APIKey

	if jcfg.Body != nil {
		bodyBytes, _ := json.Marshal(jcfg.Body)
		m.Answers.BodyTemplate = string(bodyBytes)
	}

	m.Done = true
	return m, nil
}

func (m CreateModel) View() string {
	title := "Vendor Configuration (JSON)"
	if m.IsEdit {
		title = "Edit Vendor: " + m.OriginalName
	}

	// Double check width for the internal textarea to avoid lag on first frame
	if m.Width > 0 {
		m.TextArea.SetWidth(m.Width - 4)
	}

	example := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(styles.Cyan).
		Padding(0, 1).
		Width(m.Width - 4). // m.Width - 2(padding) - 2(border)
		Render(lipgloss.JoinVertical(lipgloss.Left,
			styles.LabelStyle.Foreground(styles.Cyan).Render("JSON Configuration Template:"),
			styles.DimStyle.Render(`{`),
			styles.DimStyle.Render(`  "name": "example-vendor",`),
			styles.DimStyle.Render(`  "url": "https://example.com/api?q={{input}}",`),
			styles.DimStyle.Render(`  "method": "GET",`),
			styles.DimStyle.Render(`  "headers": { "Authorization": "Bearer {{api_key}}" },`),
			styles.DimStyle.Render(`  "body": {}`),
			styles.DimStyle.Render(`}`),
		))
	// Note: You can still use "response_path": "path.to.text" if needed

	inputView := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.Purple).
		Padding(0, 1).
		Width(m.Width - 4). // m.Width - 2(padding) - 2(border)
		Render(m.TextArea.View())

	infoText := styles.DimStyle.Italic(true).Render("ℹ Tags within {{input}} will be automatically mapped during the evaluation process.")

	var errView string
	if m.Error != "" {
		errView = "\n" + styles.ErrorTextStyle.Render(m.Error)
	}

	help := styles.HelpBar([]string{"Ctrl+S: Save", "Esc: Cancel"}, m.Width)

	return lipgloss.JoinVertical(lipgloss.Left,
		styles.HeaderStyle.Render(title),
		"\n",
		example,
		"\n",
		inputView,
		"\n",
		infoText,
		errView,
		"\n",
		help,
	)
}
