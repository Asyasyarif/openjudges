package autoprompt

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"openllmjudge/config"
	"openllmjudge/internal/styles"
	"openllmjudge/internal/tools"
	"openllmjudge/internal/tui/components"
	"openllmjudge/internal/vendor"
	"openllmjudge/judge"
	"openllmjudge/testcase"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Step int

const (
	StepSelectGetVendor Step = iota
	StepSelectUpdateVendor
	StepSelectChatVendor
	StepSelectJudges
	StepAgentRunning
	StepAgentImproving
	StepAgentComplete
	StepAgentError
)

type RunState int

const (
	StateGenerating RunState = iota
	StateEvaluating
	StateImproving
	StateIdle
)

type JudgeState struct {
	Config        config.JudgeConfig
	Total         int
	Completed     int
	Passed        int
	Failed        int
	CurrentTest   *testcase.TestCase
	StreamBuffer  strings.Builder
	EvalBuffer    strings.Builder
	HoldOnMessage string
	IsRunning     bool
	State         RunState
	CurrentUsage  judge.TokenUsage
	TotalUsage    judge.TokenUsage
	Results       []TestResult
	IterStartTime time.Time
	EvalStartTime time.Time
	ExcelPath     string
}

type TestResult struct {
	TestID       string
	Passed       bool
	Score        float64
	Summary      string
	Elapsed      time.Duration
	PromptTokens int
	OutputTokens int
}

type LogEntry struct {
	Timestamp time.Time
	Level     string
	Message   string
}

type AutoPromptModel struct {
	Config *config.Config

	GetVendorSelect    components.SimpleSelect
	UpdateVendorSelect components.SimpleSelect
	ChatVendorSelect   components.SimpleSelect
	JudgeSelect        components.SimpleSelect

	Vendors []vendor.VendorConfig

	SelectedJudge *config.JudgeConfig
	GetVendor     *vendor.VendorConfig
	UpdateVendor  *vendor.VendorConfig
	ChatVendor    *vendor.VendorConfig

	Step      Step
	State     RunState
	StartTime time.Time
	Elapsed   float64

	JudgeState *JudgeState

	Iteration     int
	MaxIterations int
	MinPassScore  float64

	AllResults []testcase.JudgeResult
	TokenUsage []map[string]int

	Spinner spinner.Model

	Width  int
	Height int

	Quitting  bool
	Done      bool
	Error     error
	ExcelPath string

	Thoughts     []Thought
	Observations []Observation
}

type Thought struct {
	Timestamp time.Time
	Iteration int
	Content   string
	Step      ReActAgentStep
}

type Observation struct {
	Timestamp  time.Time
	Iteration  int
	ActionName string
	Result     string
	Insight    string
}

type ReActAgentStep int

const (
	ReActThinking ReActAgentStep = iota
	ReActActing
	ReActObserving
	ReActReasoning
)

const (
	maxResponseLines = 10
	maxEvalLines     = 5
)

func NewInitialModel(cfg *config.Config) *AutoPromptModel {
	s := spinner.New()
	s.Spinner = spinner.Points
	s.Style = lipgloss.NewStyle().Foreground(styles.White)

	vendors, _ := vendor.LoadAllVendorsAuto()

	var vendorItems []components.SimpleSelectItem
	for _, v := range vendors {
		vendorItems = append(vendorItems, components.SimpleSelectItem{
			Title:       v.Name,
			Value:       v.Name,
			Description: fmt.Sprintf("%s %s", v.Method, v.URL),
		})
	}

	var judgeItems []components.SimpleSelectItem
	for _, j := range cfg.Judges {
		judgeItems = append(judgeItems, components.SimpleSelectItem{
			Title:       j.Name,
			Value:       j.Name,
			Description: fmt.Sprintf("%s (%s)", j.LLM.Provider, j.LLM.Model),
		})
	}

	m := AutoPromptModel{
		Config:             cfg,
		Step:               StepSelectGetVendor,
		Iteration:          1,
		MaxIterations:      5,
		MinPassScore:       70,
		Vendors:            vendors,
		Spinner:            s,
		GetVendorSelect:    components.NewSimpleSelect("Select GET Vendor (for downloading prompt)", vendorItems),
		UpdateVendorSelect: components.NewSimpleSelect("Select UPDATE Vendor (for uploading prompt)", vendorItems),
		ChatVendorSelect:   components.NewSimpleSelect("Select CHAT Vendor (for agent analysis)", vendorItems),
		JudgeSelect:        components.NewSimpleSelect("Select Judge", judgeItems),
	}

	m.GetVendorSelect.Focus()

	return &m
}

func (m AutoPromptModel) Init() tea.Cmd {
	return m.Spinner.Tick
}

func (m *AutoPromptModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	if _, ok := msg.(spinner.TickMsg); ok {
		m.Spinner, cmd = m.Spinner.Update(msg)
		return m, cmd
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		m.GetVendorSelect.SetWidth(msg.Width)
		m.UpdateVendorSelect.SetWidth(msg.Width)
		m.ChatVendorSelect.SetWidth(msg.Width)
		m.JudgeSelect.SetWidth(msg.Width)
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			m.Quitting = true
			return m, tea.Quit
		}
	}

	switch m.Step {
	case StepSelectGetVendor:
		m.GetVendorSelect, cmd = m.GetVendorSelect.Update(msg)
		if isEnter(msg) {
			val := m.GetVendorSelect.SelectedValue()
			if val != "" {
				m.GetVendor = vendor.GetVendorByName(m.Vendors, val)
				m.Step = StepSelectUpdateVendor
				m.UpdateVendorSelect.Focus()
				return m, nil
			}
		}
		return m, cmd

	case StepSelectUpdateVendor:
		m.UpdateVendorSelect, cmd = m.UpdateVendorSelect.Update(msg)
		if isEnter(msg) {
			val := m.UpdateVendorSelect.SelectedValue()
			if val != "" {
				m.UpdateVendor = vendor.GetVendorByName(m.Vendors, val)
				m.Step = StepSelectChatVendor
				m.ChatVendorSelect.Focus()
				return m, nil
			}
		}
		return m, cmd

	case StepSelectChatVendor:
		m.ChatVendorSelect, cmd = m.ChatVendorSelect.Update(msg)
		if isEnter(msg) {
			val := m.ChatVendorSelect.SelectedValue()
			if val != "" {
				m.ChatVendor = vendor.GetVendorByName(m.Vendors, val)
				m.Step = StepSelectJudges
				m.JudgeSelect.Focus()
				return m, nil
			}
		}
		return m, cmd

	case StepSelectJudges:
		m.JudgeSelect, cmd = m.JudgeSelect.Update(msg)
		if isEnter(msg) {
			val := m.JudgeSelect.SelectedValue()
			if val != "" {
				for _, j := range m.Config.Judges {
					if j.Name == val {
						m.SelectedJudge = &j
						break
					}
				}

				if m.SelectedJudge != nil {
					m.JudgeState = &JudgeState{
						Config:        *m.SelectedJudge,
						State:         StateIdle,
						HoldOnMessage: "Initializing agent...",
						IsRunning:     true,
					}

					m.Step = StepAgentRunning
					m.StartTime = time.Now()
					return m, m.startAgentWorkflow()
				}
			}
		}
		return m, cmd

	case StepAgentRunning, StepAgentImproving:
		if key, ok := msg.(tea.KeyMsg); ok {
			switch key.String() {
			case "esc":
				m.Quitting = true
				return m, tea.Quit
			}
		}
		return m, nil

	case StepAgentComplete:
		if key, ok := msg.(tea.KeyMsg); ok {
			switch key.String() {
			case "enter", "esc", "q":
				m.Quitting = true
				return m, tea.Quit
			}
		}
		return m, nil

	case StepAgentError:
		if key, ok := msg.(tea.KeyMsg); ok {
			switch key.String() {
			case "enter", "esc", "q":
				m.Quitting = true
				return m, tea.Quit
			}
		}
		return m, nil
	}

	return m, cmd
}

func (m AutoPromptModel) View() string {
	if m.Quitting {
		return ""
	}

	width := m.Width
	if width <= 0 {
		width = 80
	}

	switch m.Step {
	case StepSelectGetVendor:
		header := components.RenderHeader(width)
		view := m.GetVendorSelect.View()
		footer := styles.DimStyle.Render("\nPress Enter to select | Esc to quit")
		content := lipgloss.JoinVertical(lipgloss.Left, header, "", view, footer)
		if m.Width > 0 && m.Height > 0 {
			return lipgloss.Place(m.Width, m.Height, lipgloss.Center, lipgloss.Center, content)
		}
		return lipgloss.NewStyle().Padding(1, 2).Render(content)

	case StepSelectUpdateVendor:
		header := components.RenderHeader(width)
		view := m.UpdateVendorSelect.View()
		footer := styles.DimStyle.Render("\nPress Enter to select | Esc to go back")
		content := lipgloss.JoinVertical(lipgloss.Left, header, "", view, footer)
		if m.Width > 0 && m.Height > 0 {
			return lipgloss.Place(m.Width, m.Height, lipgloss.Center, lipgloss.Center, content)
		}
		return lipgloss.NewStyle().Padding(1, 2).Render(content)

	case StepSelectChatVendor:
		header := components.RenderHeader(width)
		view := m.ChatVendorSelect.View()
		footer := styles.DimStyle.Render("\nPress Enter to select | Esc to go back")
		content := lipgloss.JoinVertical(lipgloss.Left, header, "", view, footer)
		if m.Width > 0 && m.Height > 0 {
			return lipgloss.Place(m.Width, m.Height, lipgloss.Center, lipgloss.Center, content)
		}
		return lipgloss.NewStyle().Padding(1, 2).Render(content)

	case StepSelectJudges:
		header := components.RenderHeader(width)
		view := m.JudgeSelect.View()
		footer := styles.DimStyle.Render("\nPress Enter to select | Esc to go back")
		content := lipgloss.JoinVertical(lipgloss.Left, header, "", view, footer)
		if m.Width > 0 && m.Height > 0 {
			return lipgloss.Place(m.Width, m.Height, lipgloss.Center, lipgloss.Center, content)
		}
		return lipgloss.NewStyle().Padding(1, 2).Render(content)

	case StepAgentRunning, StepAgentImproving:
		return m.renderAgentExecution()

	case StepAgentComplete:
		return m.renderComplete()

	case StepAgentError:
		return m.renderError()
	}

	return ""
}

func (m AutoPromptModel) renderAgentExecution() string {
	state := m.JudgeState
	if state == nil {
		return ""
	}

	fullWidth := m.Width
	if fullWidth <= 0 {
		fullWidth = 80
	}

	contentWidth := fullWidth - 4
	if contentWidth > 90 {
		contentWidth = 90
	}

	var sections []string

	sections = append(sections, components.RenderHeader(contentWidth))
	sections = append(sections, "")

	if !m.Done && !m.Quitting {
		warningStyle := lipgloss.NewStyle().Foreground(styles.Yellow).Italic(true)
		sections = append(sections, warningStyle.Render("Process is running, please do not close the terminal"))
		sections = append(sections, "")
	}

	responseStyle := lipgloss.NewStyle().
		Width(contentWidth).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.Purple).
		Padding(0, 1)

	if state.State == StateEvaluating {
		responseStyle = responseStyle.BorderForeground(styles.Orange)
	}

	usableWidth := contentWidth - 2
	if usableWidth < 0 {
		usableWidth = 0
	}

	var statusText string
	ctaStyle := lipgloss.NewStyle().Foreground(styles.Cyan)
	statsStyle := lipgloss.NewStyle().Foreground(styles.Gray).Faint(true)

	elapsed := time.Since(m.StartTime).Seconds()

	if state.State == StateGenerating {
		text := "Agent generating response..."
		if state.HoldOnMessage != "" {
			text = state.HoldOnMessage
		}
		renderedText := ctaStyle.Render(text)
		statusText = fmt.Sprintf("%s %s", m.Spinner.View(), renderedText)
	} else if state.State == StateEvaluating {
		text := "Agent evaluating response..."
		if state.HoldOnMessage != "" {
			text = state.HoldOnMessage
		}
		evalText := lipgloss.NewStyle().Foreground(styles.Orange).Render(text)
		statusText = fmt.Sprintf("%s %s", m.Spinner.View(), evalText)
	} else if state.State == StateImproving {
		text := "Agent improving prompt..."
		improveText := lipgloss.NewStyle().Foreground(styles.Green).Render(text)
		statusText = fmt.Sprintf("%s %s", m.Spinner.View(), improveText)
	} else {
		statusText = ctaStyle.Render("Agent running...")
	}

	statsText := statsStyle.Render(fmt.Sprintf("%.2fs ∞ %d/%d", elapsed, state.Completed+1, state.Total))

	statusWidth := lipgloss.Width(statusText)
	statsWidth := lipgloss.Width(statsText)
	spaceWidth := usableWidth - statusWidth - statsWidth
	if spaceWidth < 0 {
		spaceWidth = 0
	}
	boxHeader := statusText + strings.Repeat(" ", spaceWidth) + statsText

	wrappedResponse := wrapText(state.StreamBuffer.String(), usableWidth)
	respLines := strings.Split(wrappedResponse, "\n")
	if len(respLines) > maxResponseLines {
		respLines = respLines[len(respLines)-maxResponseLines:]
	}
	for len(respLines) < maxResponseLines {
		respLines = append(respLines, "")
	}

	evalView := ""
	if state.State == StateEvaluating || state.EvalBuffer.Len() > 0 {
		evalText := state.EvalBuffer.String()

		liveScore := ""
		for _, result := range state.Results {
			if result.Summary != "" {
				liveScore = lipgloss.NewStyle().
					Foreground(styles.Green).
					Bold(true).
					Render(fmt.Sprintf("  Score: %.0f/100", result.Score))
				break
			}
		}

		liveSummary := ""
		for _, result := range state.Results {
			if result.Summary != "" {
				truncated := result.Summary
				if len(truncated) > 100 {
					truncated = truncated[:100] + "..."
				}
				liveSummary = "\n" + lipgloss.NewStyle().
					Italic(true).
					Foreground(styles.Cyan).
					Render("  Summary: "+truncated)
				break
			}
		}

		evalLines := strings.Split(evalText, "\n")
		if len(evalLines) > maxEvalLines {
			evalLines = evalLines[len(evalLines)-maxEvalLines:]
		}
		for len(evalLines) < maxEvalLines {
			evalLines = append(evalLines, "")
		}

		analysisHeader := lipgloss.NewStyle().
			Foreground(styles.Orange).
			Bold(true).
			Render("  LIVE EVALUATION") + liveScore

		rawLabel := lipgloss.NewStyle().
			Foreground(styles.Gray).
			Faint(true).
			Render("  RAW DATA")

		contentLines := []string{analysisHeader}
		if liveSummary != "" {
			contentLines = append(contentLines, liveSummary)
		}
		contentLines = append(contentLines, "", rawLabel, strings.Join(evalLines, "\n"))

		evalView = "\n" + lipgloss.NewStyle().
			PaddingTop(1).
			Width(usableWidth).
			Border(lipgloss.NormalBorder(), true, false, false, false).
			BorderForeground(styles.DarkGray).
			Render(lipgloss.JoinVertical(lipgloss.Left, contentLines...))
	}

	reasoningView := ""
	if len(m.Thoughts) > 0 {
		lastThought := m.Thoughts[len(m.Thoughts)-1]
		reasoningText := fmt.Sprintf("Agent: %s", lastThought.Content)
		reasoningHeader := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00BFFF")).
			Bold(true).
			Render("  THINKING")
		reasoningContent := lipgloss.NewStyle().
			Width(usableWidth).
			Render(reasoningHeader + "\n  " + reasoningText)
		reasoningView = "\n" + reasoningContent
	}

	boxContent := boxHeader + "\n\n" + strings.Join(respLines, "\n") + evalView + reasoningView
	sections = append(sections, responseStyle.Render(boxContent))
	sections = append(sections, "")

	infoLabelStyle := lipgloss.NewStyle().Foreground(styles.Gray).Faint(true)
	infoValueStyle := lipgloss.NewStyle().Foreground(styles.White)

	line1 := fmt.Sprintf("%s %s  %s  %s %s/%s  %s  %s %s",
		infoLabelStyle.Render("Judge:"), infoValueStyle.Render(state.Config.Name),
		lipgloss.NewStyle().Foreground(styles.Gray).Render("•"),
		infoLabelStyle.Render("Model:"), infoValueStyle.Render(state.Config.LLM.Provider), infoValueStyle.Render(state.Config.LLM.Model),
		lipgloss.NewStyle().Foreground(styles.Gray).Render("•"),
		infoLabelStyle.Render("Dataset:"), infoValueStyle.Render(state.Config.DataPath))

	line2 := fmt.Sprintf("%s %d/%d  %s  %s %s  %s  %s %.0f/%.0f  %s %d/%d",
		infoLabelStyle.Render("Iteration"),
		m.Iteration, m.MaxIterations,
		lipgloss.NewStyle().Foreground(styles.Gray).Render("•"),
		infoLabelStyle.Render("Tests:"), infoValueStyle.Render(fmt.Sprintf("%d/%d passed", state.Passed, state.Completed)),
		lipgloss.NewStyle().Foreground(styles.Gray).Render("•"),
		infoLabelStyle.Render("Score:"), state.TotalUsage.CompletionTokens, infoValueStyle.Render(fmt.Sprintf("%.0f", m.MinPassScore)),
		infoLabelStyle.Render("Token In/Out:"), state.TotalUsage.PromptTokens, state.TotalUsage.CompletionTokens)

	sections = append(sections, line1)
	sections = append(sections, line2)

	if m.Width > 0 && m.Height > 0 {
		return lipgloss.Place(m.Width, m.Height, lipgloss.Center, lipgloss.Center,
			lipgloss.JoinVertical(lipgloss.Left, sections...))
	}
	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

func (m AutoPromptModel) renderComplete() string {
	state := m.JudgeState
	if state == nil {
		state = &JudgeState{}
	}

	fullWidth := m.Width
	if fullWidth <= 0 {
		fullWidth = 80
	}

	boxWidth := fullWidth
	if boxWidth > 100 {
		boxWidth = 100
	}

	var sections []string

	sections = append(sections, components.RenderHeader(boxWidth))
	sections = append(sections, "")

	var status string
	var statusColor lipgloss.Color
	if state.Failed == 0 && state.Passed > 0 {
		status = "AGENT COMPLETE - ALL TESTS PASSED"
		statusColor = styles.Green
	} else if m.Iteration >= m.MaxIterations {
		status = "AGENT COMPLETE - MAX ITERATIONS REACHED"
		statusColor = styles.Red
	} else {
		status = "AGENT COMPLETE WITH FAILURES"
		statusColor = styles.Yellow
	}

	statusLine := lipgloss.NewStyle().
		Foreground(statusColor).
		Bold(true).
		Render(status)
	sections = append(sections, statusLine)
	sections = append(sections, "")

	summaryStyle := lipgloss.NewStyle().
		Width(60).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.Cyan).
		Padding(1, 2)

	summaryContent := fmt.Sprintf(`Iterations: %d/%d
Tests Run: %d | Passed: %d | Failed: %d
Final Score: %.0f/%.0f
Results File: %s`,
		m.Iteration, m.MaxIterations,
		state.Completed, state.Passed, state.Failed,
		state.TotalUsage.CompletionTokens, m.MinPassScore,
		m.ExcelPath)

	sections = append(sections, summaryStyle.Render(summaryContent))
	sections = append(sections, "")

	footer := lipgloss.JoinHorizontal(lipgloss.Top,
		styles.HighlightStyle.Render("esc"),
		styles.DimStyle.Render(" back to menu"),
	)
	sections = append(sections, footer)

	if m.Width > 0 && m.Height > 0 {
		return lipgloss.Place(m.Width, m.Height, lipgloss.Center, lipgloss.Center,
			lipgloss.JoinVertical(lipgloss.Left, sections...))
	}
	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

func (m AutoPromptModel) renderError() string {
	fullWidth := m.Width
	if fullWidth <= 0 {
		fullWidth = 80
	}

	boxWidth := fullWidth
	if boxWidth > 100 {
		boxWidth = 100
	}

	var sections []string

	sections = append(sections, components.RenderHeader(boxWidth))
	sections = append(sections, "")

	errorMsg := "Unknown error"
	if m.Error != nil {
		errorMsg = m.Error.Error()
	}
	errorLine := lipgloss.NewStyle().
		Foreground(styles.Red).
		Bold(true).
		Render("Error:")
	sections = append(sections, errorLine)
	sections = append(sections, errorMsg)
	sections = append(sections, "")

	footer := lipgloss.JoinHorizontal(lipgloss.Top,
		styles.HighlightStyle.Render("esc"),
		styles.DimStyle.Render(" back to menu"),
	)
	sections = append(sections, footer)

	if m.Width > 0 && m.Height > 0 {
		return lipgloss.Place(m.Width, m.Height, lipgloss.Center, lipgloss.Center,
			lipgloss.JoinVertical(lipgloss.Left, sections...))
	}
	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

func (m *AutoPromptModel) startAgentWorkflow() tea.Cmd {
	go func() {
		ctx := context.Background()

		state := m.JudgeState
		state.Total = 5
		state.State = StateIdle

		promptsDir := "prompts/refined"
		if err := os.MkdirAll(promptsDir, 0755); err != nil {
			m.Step = StepAgentError
			return
		}

		testCases, err := testcase.Load(m.SelectedJudge.DataPath)
		if err != nil {
			m.Step = StepAgentError
			return
		}
		state.Total = len(testCases)

		var promptContent string

		for m.Iteration <= m.MaxIterations {
			state.Results = nil
			state.TotalUsage = judge.TokenUsage{}

			state.HoldOnMessage = "Running tests..."
			state.State = StateGenerating
			state.StreamBuffer.Reset()
			// Only reset stream buffer per test - keep EvalBuffer for display

			passed := 0
			failed := 0
			totalScore := 0.0

			for i, tc := range testCases {
				testNum := i + 1
				state.Completed = testNum - 1
				state.CurrentTest = &tc
				state.IterStartTime = time.Now()

				// Only reset stream buffer (not eval buffer - we want to see evaluation results)
				state.StreamBuffer.Reset()

				state.HoldOnMessage = fmt.Sprintf("Generating response (%d/%d)...", testNum, len(testCases))
				state.State = StateGenerating

				jg := judge.NewJudge(*m.SelectedJudge)

				var chatFullText string
				var tokens judge.TokenUsage

				chatFullText, tokens, err = jg.EvaluateStreaming(ctx, tc.Prompt, judge.StreamConfig{
					OnToken: func(token string, tokenType judge.TokenType, usage judge.TokenUsage) {
						state.StreamBuffer.WriteString(token)
						state.CurrentUsage = usage
					},
				})

				if err != nil {
					state.EvalBuffer.WriteString(fmt.Sprintf("Error: %v", err))
					failed++
					state.EvalStartTime = time.Now()
					state.State = StateEvaluating
					continue
				}

				tc.AIResponse = chatFullText

				state.HoldOnMessage = fmt.Sprintf("Evaluating (%d/%d)...", testNum, len(testCases))
				state.State = StateEvaluating
				state.EvalStartTime = time.Now()

				evalResult, err := jg.Evaluate(ctx, tc)
				if err != nil {
					state.EvalBuffer.WriteString(fmt.Sprintf("Evaluation error: %v", err))
					failed++
					continue
				}

				// Keep evaluation result in buffer for display
				state.EvalBuffer.WriteString(evalResult.Result.Summary)
				passed++
				totalScore += evalResult.Result.OverallScore

				state.Results = append(state.Results, TestResult{
					TestID:       tc.ID,
					Passed:       evalResult.Result.Passed,
					Score:        evalResult.Result.OverallScore,
					Summary:      evalResult.Result.Summary,
					Elapsed:      time.Since(state.IterStartTime),
					PromptTokens: tokens.PromptTokens,
					OutputTokens: tokens.CompletionTokens,
				})

				state.TotalUsage.PromptTokens += tokens.PromptTokens
				state.TotalUsage.CompletionTokens += tokens.CompletionTokens
				state.TotalUsage.TotalTokens += tokens.TotalTokens

				// Apply delay + jitter between API calls (from config or defaults)
				m.applyDelayJitter()
			}

			state.Completed = len(testCases)
			avgScore := 0.0
			if len(testCases) > 0 {
				avgScore = totalScore / float64(len(testCases))
			}

			if passed == len(testCases) && avgScore >= m.MinPassScore {
				m.Done = true
				m.Step = StepAgentComplete
				m.saveResults()
				return
			}

			if m.Iteration >= m.MaxIterations {
				m.Done = true
				m.Step = StepAgentComplete
				m.saveResults()
				return
			}

			state.HoldOnMessage = "Downloading prompt..."
			state.State = StateGenerating
			state.StreamBuffer.Reset()
			state.EvalBuffer.Reset()

			promptContent, err = m.downloadPromptFromAPI(ctx)
			if err != nil {
				m.Step = StepAgentError
				return
			}

			timestamp := time.Now().Format("20060102_150405")
			localPath := filepath.Join(promptsDir, fmt.Sprintf("prompt_iter%d_%s.md", m.Iteration, timestamp))
			if err := tools.Write(localPath, promptContent); err != nil {
				m.Step = StepAgentError
				return
			}

			state.HoldOnMessage = "Agent improving prompt..."
			state.State = StateImproving
			state.StreamBuffer.Reset()
			state.EvalBuffer.Reset()

			var failedResults []failedTestResult
			for _, result := range state.Results {
				if !result.Passed {
					failedResults = append(failedResults, failedTestResult{
						TC:     testcase.TestCase{ID: result.TestID},
						Result: testcase.JudgeResult{OverallScore: result.Score, Summary: result.Summary},
					})
				}
			}

			suggestion, err := m.agentAnalyzeAndSuggest(ctx, promptContent, failedResults)
			if err != nil {
				m.Step = StepAgentError
				return
			}

			if suggestion != "" {
				newContent, err := m.editPrompt(promptContent, suggestion)
				if err != nil {
					m.Step = StepAgentError
					return
				}
				promptContent = newContent
			}

			state.HoldOnMessage = "Uploading improved prompt..."
			state.State = StateImproving

			if err := m.uploadPromptToAPI(ctx, promptContent); err != nil {
				m.Step = StepAgentError
				return
			}

			m.Iteration++
			m.Thoughts = append(m.Thoughts, Thought{
				Timestamp: time.Now(),
				Iteration: m.Iteration - 1,
				Content:   fmt.Sprintf("Iteration %d complete, testing again", m.Iteration-1),
				Step:      ReActThinking,
			})

			state.StreamBuffer.Reset()
			state.EvalBuffer.Reset()
		}

		m.Done = true
		m.Step = StepAgentComplete
		m.saveResults()
	}()

	return tickCmd
}

func (m *AutoPromptModel) downloadPromptFromAPI(ctx context.Context) (string, error) {
	if m.GetVendor == nil {
		return "Default prompt for testing", nil
	}

	tc := testcase.TestCase{
		ID:          "download",
		Name:        "Download Prompt",
		Prompt:      "Return the current prompt content",
		AIResponse:  "",
		Expectation: "Return the prompt content",
	}

	return m.GetVendor.CallVendorAPI(ctx, tc)
}

func (m *AutoPromptModel) uploadPromptToAPI(ctx context.Context, content string) error {
	if m.UpdateVendor == nil {
		return nil
	}

	url := m.UpdateVendor.URL
	if strings.Contains(url, "{{prompt}}") {
		url = strings.ReplaceAll(url, "{{prompt}}", content)
	}

	req, err := http.NewRequestWithContext(ctx, m.UpdateVendor.Method, url, nil)
	if err != nil {
		return err
	}

	for key, val := range m.UpdateVendor.Headers {
		req.Header.Set(key, val)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("upload failed with status: %d", resp.StatusCode)
	}

	return nil
}

func (m *AutoPromptModel) saveResults() error {
	state := m.JudgeState
	if state == nil {
		state = &JudgeState{}
	}

	resultsDir := "results"
	if err := os.MkdirAll(resultsDir, 0755); err != nil {
		return err
	}

	timestamp := time.Now().Format("20060102_150405")
	excelPath := filepath.Join(resultsDir, fmt.Sprintf("AutoPrompt_%s.xlsx", timestamp))

	f, err := os.Create(excelPath)
	if err != nil {
		return err
	}
	defer f.Close()

	header := "Test ID,Passed,Score,Summary,Elapsed,Prompt Tokens,Output Tokens\n"
	f.WriteString(header)

	for _, result := range state.Results {
		passed := "Yes"
		if !result.Passed {
			passed = "No"
		}
		line := fmt.Sprintf("%s,%s,%.0f,%s,%v,%d,%d\n",
			result.TestID, passed, result.Score, result.Summary,
			result.Elapsed, result.PromptTokens, result.OutputTokens)
		f.WriteString(line)
	}

	m.ExcelPath = excelPath
	state.ExcelPath = excelPath

	return nil
}

func (m *AutoPromptModel) agentAnalyzeAndSuggest(ctx context.Context, currentPrompt string, failedResults []failedTestResult) (string, error) {
	if m.ChatVendor == nil {
		return m.defaultImprovementSuggestion(failedResults), nil
	}

	var failedDetails string
	for _, fr := range failedResults {
		failedDetails += fmt.Sprintf("Test: %s, Score: %.0f, Summary: %s\n",
			fr.TC.ID, fr.Result.OverallScore, fr.Result.Summary)
	}

	analysisPrompt := fmt.Sprintf(`You are a prompt engineering expert. Analyze the failed tests and suggest improvements to the prompt.

Current Prompt:
%s

Failed Tests:
%s

Your task:
1. Identify the root causes of the failures
2. Suggest specific text changes to fix these issues
3. Be precise: use EXACT text from the prompt

Return suggestions in this exact format:
Change: [exact text to replace] -> [exact replacement text]
Reason: [brief explanation why this helps]
`, currentPrompt, failedDetails)

	tc := testcase.TestCase{
		ID:          "analysis",
		Name:        "Prompt Analysis",
		Prompt:      analysisPrompt,
		AIResponse:  "",
		Expectation: "Provide prompt improvement suggestions",
	}

	response, err := m.ChatVendor.CallVendorAPI(ctx, tc)
	if err != nil {
		return "", err
	}

	return response, nil
}

func (m *AutoPromptModel) editPrompt(currentPrompt, suggestions string) (string, error) {
	return currentPrompt + "\n\n## Improvements\n" + suggestions, nil
}

type failedTestResult struct {
	TC     testcase.TestCase
	Result testcase.JudgeResult
}

func (m *AutoPromptModel) defaultImprovementSuggestion(failedResults []failedTestResult) string {
	if len(failedResults) == 0 {
		return "Add more specific evaluation criteria and examples to the prompt."
	}
	return "Make the evaluation criteria more specific and measurable. Add examples of good and bad responses."
}

// applyDelayJitter applies configured delay + random jitter between API calls
// Falls back to defaults: 2000ms delay + up to 5000ms jitter
func (m *AutoPromptModel) applyDelayJitter() {
	delayMs := m.Config.Options.DelayMs
	jitterMs := m.Config.Options.JitterMs

	// Fallback if not configured (same as executor.go)
	if delayMs <= 0 {
		delayMs = 2000
	}
	if jitterMs <= 0 {
		jitterMs = 5000
	}

	totalDelay := time.Duration(delayMs) * time.Millisecond
	if jitterMs > 0 {
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		totalDelay += time.Duration(r.Intn(jitterMs)) * time.Millisecond
	}

	time.Sleep(totalDelay)
}

func isEnter(msg tea.Msg) bool {
	if key, ok := msg.(tea.KeyMsg); ok {
		return key.String() == "enter"
	}
	return false
}

func tickCmd() tea.Msg {
	return tickMsg{}
}

type tickMsg struct{}

func wrapText(text string, maxWidth int) string {
	if maxWidth <= 0 {
		return text
	}
	var wrappedLines []string
	splitLines := strings.Split(text, "\n")
	for _, line := range splitLines {
		if lipgloss.Width(line) <= maxWidth {
			wrappedLines = append(wrappedLines, line)
		} else {
			words := strings.Fields(line)
			if len(words) == 0 {
				wrappedLines = append(wrappedLines, "")
				continue
			}
			currentLine := ""
			for _, word := range words {
				testLine := currentLine
				if testLine != "" {
					testLine += " "
				}
				testLine += word
				if lipgloss.Width(testLine) <= maxWidth {
					currentLine = testLine
				} else {
					if currentLine != "" {
						wrappedLines = append(wrappedLines, currentLine)
					}
					currentLine = word
				}
			}
			if currentLine != "" {
				wrappedLines = append(wrappedLines, currentLine)
			}
		}
	}
	return strings.Join(wrappedLines, "\n")
}
