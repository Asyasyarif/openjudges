package results

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"openjudges/internal/styles"
	"openjudges/internal/tui/components"
	"openjudges/testcase"
)

type State int

const (
	StateBrowsing State = iota
	StateList
	StateDetail
)

// ResultData represents the loaded result file
type ResultData struct {
	JudgeName string                 `json:"judge_name"`
	Timestamp string                 `json:"timestamp"`
	Total     int                    `json:"total"`
	Passed    int                    `json:"passed"`
	Failed    int                    `json:"failed"`
	Results   []testcase.JudgeResult `json:"results"`
}

// ResultItem implements list.Item for test results
type ResultItem struct {
	Result testcase.JudgeResult
}

func (i ResultItem) Title() string {
	if i.Result.Passed {
		return "✓ " + i.Result.TestCaseID
	}
	return "✗ " + i.Result.TestCaseID
}

func (i ResultItem) Description() string {
	score := i.Result.OverallScore
	if score == 0 && i.Result.Score > 0 {
		score = i.Result.Score
	}
	summary := i.Result.Summary
	if summary == "" {
		summary = i.Result.Reasoning
	}
	if len(summary) > 50 {
		summary = summary[:50] + "..."
	}
	return fmt.Sprintf("Score: %.1f/10 - %s", score, summary)
}

func (i ResultItem) FilterValue() string { return i.Result.TestCaseID + " " + i.Result.Reasoning }

// Model for results TUI
type Model struct {
	State          State
	FileBrowser    components.FileBrowser
	ResultList     list.Model
	DetailViewport viewport.Model

	CurrentResult *ResultData
	SelectedTest  *testcase.JudgeResult

	width  int
	height int
}

func NewModel(initialFile string) Model {
	// File browser
	fb := components.NewFileBrowser("results", "", 0, 0)
	fb = fb.WithBackOption()

	// Result List
	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = styles.SelectedItemStyle
	delegate.Styles.SelectedDesc = styles.DimStyle.Copy().Foreground(lipgloss.Color(styles.Purple))

	l := list.New([]list.Item{}, delegate, 0, 0)
	l.Title = "Test Results"
	l.SetShowStatusBar(true)
	l.Styles.Title = styles.HeaderStyle

	m := Model{
		State:       StateBrowsing,
		FileBrowser: fb,
		ResultList:  l,
	}

	// Load initial file if provided
	if initialFile != "" {
		data, err := m.loadResult(initialFile)
		if err == nil {
			m.CurrentResult = data
			m.State = StateList
			m.populateList(data)
		}
	}

	return m
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		// Update sub-models sizes
		h, v := styles.DocStyle.GetFrameSize()
		listWidth := msg.Width - h
		listHeight := msg.Height - v

		// File browser uses select component which needs sizing
		m.FileBrowser.Select.SetSize(listWidth, listHeight)
		m.ResultList.SetSize(listWidth, listHeight)

		// Viewport size
		m.DetailViewport = viewport.New(listWidth, listHeight)
		// Note: Viewport content needs render to set properly, we do it when setting content

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "esc":
			if m.State == StateList {
				m.State = StateBrowsing
				m.CurrentResult = nil
				return m, nil
			} else if m.State == StateDetail {
				m.State = StateList
				return m, nil
			} else {
				return m, tea.Quit
			}
		case "q":
			if m.State != StateBrowsing { // Allow 'q' to go back/quit except in browsing where it might filter? No filebrowser filters by typing.
				// Actually bubbles/list uses 'q' to quit by default? No, usually not mapped unless requested.
				// FileBrowser uses Select which uses bubbles/list.
			}
		}
	}

	// Handle updates based on state
	switch m.State {
	case StateBrowsing:
		newFB, newCmd := m.FileBrowser.Update(msg)
		m.FileBrowser = newFB
		cmds = append(cmds, newCmd)

		// Check selection
		if m.FileBrowser.SelectedPath() != "" {
			path := m.FileBrowser.SelectedPath()
			if path == "__back__" {
				return m, tea.Quit
			}

			// Load file
			data, err := m.loadResult(path)
			if err == nil {
				m.CurrentResult = data
				m.State = StateList
				m.populateList(data)
			}
		}

	case StateList:
		newList, newCmd := m.ResultList.Update(msg)
		m.ResultList = newList
		cmds = append(cmds, newCmd)

		// Check enter
		if msg, ok := msg.(tea.KeyMsg); ok && msg.String() == "enter" {
			if m.ResultList.SelectedItem() != nil {
				item := m.ResultList.SelectedItem().(ResultItem)
				m.SelectedTest = &item.Result
				m.State = StateDetail
				m.renderDetail()
			}
		}

	case StateDetail:
		newVP, cmd := m.DetailViewport.Update(msg)
		m.DetailViewport = newVP
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m *Model) loadResult(path string) (*ResultData, error) {
	// Read file
	bytes, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var data ResultData
	if err := json.Unmarshal(bytes, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

func (m *Model) populateList(data *ResultData) {
	items := make([]list.Item, len(data.Results))
	for i, r := range data.Results {
		items[i] = ResultItem{Result: r}
	}
	m.ResultList.SetItems(items)
	m.ResultList.Title = fmt.Sprintf("Results: %s (%d tests)", data.JudgeName, data.Total)
}

func (m *Model) renderDetail() {
	if m.SelectedTest == nil {
		return
	}
	r := m.SelectedTest

	var content strings.Builder

	// Prompt
	content.WriteString(styles.HeaderStyle.Render("Prompt"))
	content.WriteString("\n")
	content.WriteString(r.Prompt)
	content.WriteString("\n\n")

	// AI Response
	content.WriteString(styles.HeaderStyle.Render("AI Response"))
	content.WriteString("\n")
	content.WriteString(r.AIResponse)
	content.WriteString("\n\n")

	// Expected
	content.WriteString(styles.HeaderStyle.Render("Expected Behavior"))
	content.WriteString("\n")
	content.WriteString(r.Expectation)
	content.WriteString("\n\n")

	// Evaluation - Overall Score
	overallScore := r.OverallScore
	if overallScore == 0 && r.Score > 0 {
		overallScore = r.Score
	}
	content.WriteString(styles.HeaderStyle.Render("Evaluation"))
	content.WriteString("\n")
	passIcon := "✓"
	if !r.Passed {
		passIcon = "✗"
	}
	content.WriteString(fmt.Sprintf("Overall Score: %.1f/10 | %s %v\n\n", overallScore, passIcon, r.Passed))

	// Criteria Scores
	if len(r.CriteriaScores) > 0 {
		content.WriteString(styles.SubheaderStyle.Render("Criteria Scores"))
		content.WriteString("\n")
		for name, score := range r.CriteriaScores {
			content.WriteString(fmt.Sprintf("• %s: %.1f/%.0f\n", name, score.Score, score.MaxScore))
			content.WriteString(fmt.Sprintf("  %s\n", score.Reasoning))
		}
		content.WriteString("\n")
	}

	// Strengths
	if len(r.Strengths) > 0 {
		content.WriteString(styles.SuccessTextStyle.Render("Strengths"))
		content.WriteString("\n")
		for _, s := range r.Strengths {
			content.WriteString(fmt.Sprintf("✓ %s\n", s))
		}
		content.WriteString("\n")
	}

	// Weaknesses
	if len(r.Weaknesses) > 0 {
		content.WriteString(styles.ErrorTextStyle.Render("Weaknesses"))
		content.WriteString("\n")
		for _, w := range r.Weaknesses {
			content.WriteString(fmt.Sprintf("✗ %s\n", w))
		}
		content.WriteString("\n")
	}

	// Suggestions
	if len(r.Suggestions) > 0 {
		content.WriteString(styles.WarningTextStyle.Render("Suggestions"))
		content.WriteString("\n")
		for _, s := range r.Suggestions {
			content.WriteString(fmt.Sprintf("→ %s\n", s))
		}
		content.WriteString("\n")
	}

	// Summary
	if r.Summary != "" {
		content.WriteString(styles.SubheaderStyle.Render("Summary"))
		content.WriteString("\n")
		content.WriteString(r.Summary)
		content.WriteString("\n")
	}

	m.DetailViewport.SetContent(content.String())
}

func (m Model) View() string {
	var content string

	switch m.State {
	case StateBrowsing:
		m.FileBrowser.Select.SetTitle("Evaluation Results")
		content = m.FileBrowser.View()

	case StateList:
		content = m.ResultList.View()

	case StateDetail:
		header := styles.HeaderStyle.Render("Test Case: " + m.SelectedTest.TestCaseID)
		content = fmt.Sprintf("%s\n%s", header, m.DetailViewport.View())
	}

	return styles.DocStyle.Render(content)
}
