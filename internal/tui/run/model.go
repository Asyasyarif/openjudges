package run

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/tree"
	"github.com/gen2brain/beeep"

	"openjudges/config"
	"openjudges/internal/runner"
	"openjudges/internal/styles"
	"openjudges/internal/tui/components"
	"openjudges/internal/vendor"
	"openjudges/judge"
	"openjudges/testcase"
)

// Step represents the current step in the run flow
type Step int

const (
	StepSelectJudges   Step = iota
	StepSelectProvider      // Choose builtin or vendor
	StepSelectVendor        // Select vendor from list
	StepRunning
)

type RunState int

const (
	StateGenerating RunState = iota
	StateEvaluating
)

type IterationResult struct {
	Iteration    int
	Question     string
	Elapsed      time.Duration
	RoastElapsed time.Duration
}

var monkeySpinner = spinner.Spinner{
	Frames: []string{"🙈", "🙉", "🙊", "🙉"},
	FPS:    time.Second / 6,
}

const (
	maxResponseLines = 3
	maxEvalLines     = 5
)

// sendCompletionNotification sends a desktop notification and terminal bell when run completes
func sendCompletionNotification(totalTests, passed, failed int, duration time.Duration) {
	// Terminal bell (always happens, works everywhere)
	fmt.Print("\a")

	// Desktop notification (goroutine to avoid blocking)
	go func() {
		title := "OpenJudges - Complete"
		message := fmt.Sprintf(
			"Completed %d tests in %.2fs\nPassed: %d | Failed: %d",
			totalTests, duration.Seconds(), passed, failed,
		)

		// Try desktop notification
		if err := beeep.Notify(title, message, ""); err != nil {
			// Silent fail - terminal bell already happened
		}
	}()
}

// RunModel is the main model for the execution UI
type RunModel struct {
	Step      Step
	Config    *config.Config
	Executor  *runner.Executor
	EventChan <-chan runner.Event

	// Selection Components
	JudgeSelect        components.SimpleSelect
	ProviderTypeSelect components.SimpleSelect // Builtin vs Vendor choice
	VendorSelect       components.SimpleSelect // Vendor selection
	SelectedJudges     []string                // Store selected judges before provider selection
	SelectedVendor     *vendor.VendorConfig    // Selected vendor (nil = use builtin)
	UseVendor          bool                    // Flag to indicate vendor mode
	Vendors            []vendor.VendorConfig   // Loaded vendors

	// State
	Judges      map[string]*JudgeState
	ActiveJudge string

	// Animations
	Spinner     spinner.Model
	EvalSpinner spinner.Model
	StartTime   time.Time
	Elapsed     float64

	// Layout
	Width  int
	Height int

	// Flags
	Quitting    bool
	Done        bool
	RunnerError error
	TotalTime   time.Duration
}

// JudgeState tracks the state of a single judge execution
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
	Results       []IterationResult
	IterStartTime time.Time
	EvalStartTime time.Time
	ExcelPath     string // NEW: Store Excel report path
}

func NewRunModel(cfg *config.Config, exec *runner.Executor, autoStartJudges []string) *RunModel {
	s := spinner.New()
	s.Spinner = spinner.Points
	s.Style = lipgloss.NewStyle().Foreground(styles.White)

	es := spinner.New()
	es.Spinner = monkeySpinner
	es.Style = lipgloss.NewStyle().Foreground(styles.White)

	m := RunModel{
		Step:        StepSelectJudges,
		Config:      cfg,
		Executor:    exec,
		EventChan:   exec.Events(),
		Judges:      make(map[string]*JudgeState),
		Spinner:     s,
		EvalSpinner: es,
		StartTime:   time.Now(),
	}

	// Initialize Judge Selection with all judges
	var judgeItems []components.SimpleSelectItem
	for _, j := range cfg.Judges {
		judgeItems = append(judgeItems, components.SimpleSelectItem{
			Title:       j.Name,
			Value:       j.Name,
			Description: fmt.Sprintf("%s (%s)", j.LLM.Provider, j.LLM.Model),
		})
	}

	m.JudgeSelect = components.NewSimpleSelect("Select Judge", judgeItems)
	m.JudgeSelect.Focus()

	// Auto-load vendors
	m.loadVendors()

	if len(autoStartJudges) > 0 {
		m.Step = StepRunning
		m.SelectedJudges = autoStartJudges
	}

	return &m
}

func (m *RunModel) Init() tea.Cmd {
	var cmds []tea.Cmd
	cmds = append(cmds, m.Spinner.Tick, m.EvalSpinner.Tick)

	if m.Step == StepRunning {
		m.StartTime = time.Now()
		cmds = append(cmds, waitForEvent(m.EventChan))
		// Auto-trigger executor run (all by default if empty autoStart but in Running step)
		cmds = append(cmds, func() tea.Msg {
			go func() {
				// Use SelectedJudges if provided, otherwise run all (fallback)
				names := m.SelectedJudges
				if len(names) == 0 {
					for _, j := range m.Config.Judges {
						names = append(names, j.Name)
					}
				}
				m.Executor.Run(names)
			}()
			return nil
		})
	}
	return tea.Batch(cmds...)
}

func (m *RunModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	// Process spinners first to ensure they always animate
	if sMsg, ok := msg.(spinner.TickMsg); ok {
		var cmd1, cmd2 tea.Cmd
		if sMsg.ID == m.Spinner.ID() {
			m.Spinner, cmd1 = m.Spinner.Update(msg)
		} else if sMsg.ID == m.EvalSpinner.ID() {
			m.EvalSpinner, cmd2 = m.EvalSpinner.Update(msg)
		}
		return m, tea.Batch(cmd1, cmd2)
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		m.VendorSelect.SetWidth(msg.Width)
		return m, nil

	case tea.KeyMsg:
		if m.Done {
			switch msg.String() {
			case "esc", "q", "ctrl+c":
				return m, tea.Quit
			}
		}
		switch msg.String() {
		case "ctrl+c", "q":
			if m.Step == StepRunning {
				m.Quitting = true
				m.Executor.Cancel() // Cancel execution
				return m, tea.Quit
			} else {
				return m, tea.Quit
			}
		case "tab", "right", "n":
			m.cycleJudge(1)
		case "shift+tab", "left", "p":
			m.cycleJudge(-1)
		}
	}

	// Handle steps
	switch m.Step {
	case StepSelectJudges:
		if key, ok := msg.(tea.KeyMsg); ok && key.String() == "esc" {
			return m, tea.Quit
		}
		m.JudgeSelect, cmd = m.JudgeSelect.Update(msg)
		if msgTypeIsEnter(msg) {
			val := m.JudgeSelect.SelectedValue()
			if val != "" {
				m.SelectedJudges = []string{val}

				// Check if the selected judge already specifies a vendor
				for _, jc := range m.Config.Judges {
					if jc.Name == val && strings.HasPrefix(strings.ToLower(jc.LLM.Provider), "vendor:") {
						vendorName := strings.TrimPrefix(strings.ToLower(jc.LLM.Provider), "vendor:")
						v := vendor.GetVendorByName(m.Vendors, vendorName)
						if v != nil {
							m.SelectedVendor = v
							m.Executor.SetVendor(v)
							m.Step = StepRunning
							m.StartTime = time.Now()
							go func() {
								m.Executor.Run(m.SelectedJudges)
							}()
							return m, waitForEvent(m.EventChan)
						}
					}
				}

				// Move to provider selection
				m.Step = StepSelectProvider
				m.initProviderSelection()
				return m, nil
			}
		}
		return m, cmd

	case StepSelectProvider:
		if key, ok := msg.(tea.KeyMsg); ok && key.String() == "esc" {
			m.Step = StepSelectJudges
			return m, nil
		}
		m.ProviderTypeSelect, cmd = m.ProviderTypeSelect.Update(msg)
		if msgTypeIsEnter(msg) {
			val := m.ProviderTypeSelect.SelectedValue()
			if val == "vendor" {
				// User selected vendor mode
				m.UseVendor = true
				if m.initVendorSelection() {
					m.Step = StepSelectVendor
					return m, nil
				}
				// No vendors found, show error message
				m.Step = StepRunning
				m.RunnerError = fmt.Errorf("no vendors found in ./vendors folder. Create vendor JSON files first")
				m.Done = true
				return m, nil
			}

			// Use builtin AI - move to running
			m.Step = StepRunning
			m.StartTime = time.Now()

			// Start execution in background
			go func() {
				if err := m.Executor.Run(m.SelectedJudges); err != nil {
					// Error handling
				}
			}()

			return m, tea.Batch(
				waitForEvent(m.EventChan),
			)
		}
		return m, cmd

	case StepSelectVendor:
		if key, ok := msg.(tea.KeyMsg); ok && key.String() == "esc" {
			m.Step = StepSelectProvider
			return m, nil
		}
		m.VendorSelect, cmd = m.VendorSelect.Update(msg)
		if msgTypeIsEnter(msg) {
			// Get selected vendor
			selectedVendorName := m.VendorSelect.SelectedValue()
			m.SelectedVendor = vendor.GetVendorByName(m.Vendors, selectedVendorName)

			// Set vendor on executor
			if m.SelectedVendor != nil {
				m.Executor.SetVendor(m.SelectedVendor)
			}

			// Move to running
			m.Step = StepRunning
			m.StartTime = time.Now()

			// Start execution in background
			go func() {
				if err := m.Executor.Run(m.SelectedJudges); err != nil {
					// Error handling
				}
			}()

			return m, tea.Batch(
				waitForEvent(m.EventChan),
			)
		}
		return m, cmd

	case StepRunning:
		// Handle runner events
		switch msg := msg.(type) {
		case runner.JudgeStartMsg:
			if _, exists := m.Judges[msg.Name]; !exists {
				m.Judges[msg.Name] = &JudgeState{
					Total:     msg.Total,
					Config:    msg.Config,
					IsRunning: true,
				}
			}
			m.ActiveJudge = msg.Name
			return m, waitForEvent(m.EventChan)

		case runner.TestStartMsg:
			if state, ok := m.Judges[msg.JudgeName]; ok {
				state.CurrentTest = &msg.TestData
				state.StreamBuffer.Reset()
				state.EvalBuffer.Reset()
				state.HoldOnMessage = ""
				state.IterStartTime = time.Now()
				state.State = StateGenerating
				if m.ActiveJudge == "" {
					m.ActiveJudge = msg.JudgeName
				}
			}
			return m, waitForEvent(m.EventChan)

		case runner.TokenStreamMsg:
			if state, ok := m.Judges[msg.JudgeName]; ok {
				state.CurrentUsage = msg.Usage
				if state.State == StateEvaluating {
					state.EvalBuffer.WriteString(msg.Token)
				} else {
					state.StreamBuffer.WriteString(msg.Token)
				}
			}
			return m, waitForEvent(m.EventChan)

		case runner.HoldOnMsg:
			if state, ok := m.Judges[msg.JudgeName]; ok {
				state.HoldOnMessage = msg.Message
			}
			return m, waitForEvent(m.EventChan)

		case runner.RoastingStartMsg:
			if state, ok := m.Judges[msg.JudgeName]; ok {
				state.HoldOnMessage = ""
				state.EvalStartTime = time.Now()
				state.State = StateEvaluating
			}
			return m, waitForEvent(m.EventChan)

		case runner.StreamCompleteMsg:
			return m, waitForEvent(m.EventChan)

		case runner.TestCompleteMsg:
			if state, ok := m.Judges[msg.JudgeName]; ok {
				// Handle SUMMARY result separately to capture Excel path
				if msg.TestID == "SUMMARY" {
					const prefix = "Excel report generated: "
					if strings.HasPrefix(msg.Result.Summary, prefix) {
						state.ExcelPath = strings.TrimPrefix(msg.Result.Summary, prefix)
					}
					return m, waitForEvent(m.EventChan)
				}

				state.Completed++
				if msg.Result.Passed {
					state.Passed++
				} else {
					state.Failed++
				}

				// Add to results for tree view
				state.Results = append(state.Results, IterationResult{
					Iteration:    state.Completed,
					Question:     state.CurrentTest.Prompt,
					Elapsed:      time.Since(state.IterStartTime),
					RoastElapsed: time.Since(state.EvalStartTime),
				})

				// Accumulate total usage
				state.TotalUsage.PromptTokens += state.CurrentUsage.PromptTokens
				state.TotalUsage.CompletionTokens += state.CurrentUsage.CompletionTokens
				state.TotalUsage.TotalTokens += state.CurrentUsage.TotalTokens
				state.CurrentUsage = judge.TokenUsage{} // Reset for next test

				// Format evaluation with structured output
				judgment := formatStructuredJudgment(&msg.Result)
				state.EvalBuffer.Reset()
				state.EvalBuffer.WriteString(judgment)
			}
			return m, waitForEvent(m.EventChan)

		case runner.JudgeCompleteMsg:
			if state, ok := m.Judges[msg.JudgeName]; ok {
				state.IsRunning = false
			}
			m.Done = true
			m.TotalTime = time.Since(m.StartTime)

			// Calculate totals from all judges
			totalTests := 0
			totalPassed := 0
			totalFailed := 0
			for _, j := range m.Judges {
				totalTests += j.Total
				totalPassed += j.Passed
				totalFailed += j.Failed
			}

			// Send completion notification
			sendCompletionNotification(totalTests, totalPassed, totalFailed, m.TotalTime)

			return m, waitForEvent(m.EventChan)

		case runner.ErrorMsg:
			m.RunnerError = msg.Error
			m.Done = true
			m.TotalTime = time.Since(m.StartTime)
			return m, waitForEvent(m.EventChan)
		}
	}

	return m, cmd
}

// formatStructuredJudgment formats the judge result with structured output
func formatStructuredJudgment(result *testcase.JudgeResult) string {
	var b strings.Builder

	// Overall score and pass/fail status
	passIcon := "✓"
	if !result.Passed {
		passIcon = "✗"
	}
	b.WriteString(fmt.Sprintf("Overall Score: %.1f/100 | Grade: %s | %s %v\n",
		result.OverallScore, result.OverallGrade, passIcon, result.Passed))

	// Dimension scores breakdown
	if len(result.DimensionScores) > 0 {
		b.WriteString("\nDimensions:\n")
		// Sorted keys for consistent display
		dims := []string{"accuracy", "completeness", "clarity", "relevance", "actionability"}
		for _, name := range dims {
			if ds, ok := result.DimensionScores[name]; ok {
				b.WriteString(fmt.Sprintf("  • %-13s: %.1f/100 (w: %.0f%%)\n",
					strings.Title(name), ds.Score, ds.Weight*100))
			}
		}
	}

	// Summary
	if result.Summary != "" {
		b.WriteString(fmt.Sprintf("\nSummary: %s\n", result.Summary))
	}

	return b.String()
}

func (m RunModel) View() string {
	if m.Quitting {
		return ""
	}

	// Calculate width for header
	width := m.Width
	if width <= 0 {
		width = 80
	}

	switch m.Step {
	case StepSelectJudges:
		header := components.RenderHeader(width)
		view := m.JudgeSelect.View()
		content := lipgloss.JoinVertical(lipgloss.Left, header, "", view)
		if m.Width > 0 && m.Height > 0 {
			return lipgloss.Place(m.Width, m.Height, lipgloss.Center, lipgloss.Center, content)
		}
		return lipgloss.NewStyle().Padding(1, 2).Render(content)
	case StepSelectProvider:
		return m.renderProviderSelection()
	case StepSelectVendor:
		return m.renderVendorSelection()
	case StepRunning:
		return m.renderRunning()
	}

	return ""
}

// initProviderSelection initializes the provider type selection menu
func (m *RunModel) initProviderSelection() {
	items := []components.SimpleSelectItem{
		{
			Title:       "Use Builtin AI",
			Description: "Use judge's configured provider",
			Value:       "builtin",
		},
		{
			Title:       "Use Vendor",
			Description: "Use custom vendor API for response generation",
			Value:       "vendor",
		},
	}
	m.ProviderTypeSelect = components.NewSimpleSelect("Select Provider", items)
	m.ProviderTypeSelect.Focus()
}

func (m *RunModel) loadVendors() {
	// Auto-detect vendors directory location
	vendors, err := vendor.LoadAllVendorsAuto()
	if err == nil && len(vendors) > 0 {
		m.Vendors = vendors
	}
}

// initVendorSelection initializes the vendor selection menu
// Returns true if vendors were loaded successfully, false otherwise
func (m *RunModel) initVendorSelection() bool {
	if len(m.Vendors) == 0 {
		m.loadVendors()
	}

	if len(m.Vendors) == 0 {
		return false
	}

	var items []components.SimpleSelectItem
	for _, v := range m.Vendors {
		// Truncate URL for display if too long
		displayURL := v.URL
		if len(displayURL) > 50 {
			displayURL = displayURL[:47] + "..."
		}
		items = append(items, components.SimpleSelectItem{
			Title:       v.Name,
			Description: fmt.Sprintf("File: %s | %s - %s", v.Filename, v.Method, displayURL),
			Value:       v.Name,
		})
	}
	m.VendorSelect = components.NewSimpleSelect("Select Vendor", items)
	m.VendorSelect.Focus()
	if m.Width > 0 {
		m.VendorSelect.SetWidth(m.Width)
	}
	return true
}

// renderVendorSelection renders the vendor selection menu
func (m RunModel) renderVendorSelection() string {
	view := m.VendorSelect.View()
	footer := styles.DimStyle.Render("Press Enter to select | Esc to go back")
	content := lipgloss.JoinVertical(lipgloss.Left, view, "", footer)

	return content
}

// renderProviderSelection renders the provider selection menu
func (m RunModel) renderProviderSelection() string {
	width := m.Width
	if width <= 0 {
		width = 80
	}

	header := components.RenderHeader(width)
	view := m.ProviderTypeSelect.View()
	footer := "\n" + styles.DimStyle.Render("Press Enter to continue | Esc to go back")
	content := lipgloss.JoinVertical(lipgloss.Left, header, "", view, footer)

	if m.Width > 0 && m.Height > 0 {
		return lipgloss.Place(m.Width, m.Height, lipgloss.Center, lipgloss.Center, content)
	}

	return lipgloss.NewStyle().Padding(1, 2).Render(content)
}

func (m RunModel) renderRunning() string {
	fullWidth := m.Width
	if fullWidth <= 0 {
		fullWidth = 80
	}

	boxWidth := fullWidth
	if boxWidth > 100 {
		boxWidth = 100
	}

	contentWidth := boxWidth - 4
	if contentWidth < 20 {
		contentWidth = 20
	}

	var sections []string

	// HEADER
	sections = append(sections, components.RenderHeader(boxWidth))

	// Handle Runner Error
	if m.RunnerError != nil {
		errorView := lipgloss.JoinVertical(lipgloss.Left,
			styles.ErrorTextStyle.Render("Runner Error:"),
			styles.ErrorBoxStyle.Render(m.RunnerError.Error()),
			"",
			styles.DimStyle.Render("Press esc to return"),
		)
		if m.Width > 0 && m.Height > 0 {
			centeredError := lipgloss.Place(boxWidth, 10, lipgloss.Center, lipgloss.Center, errorView)
			sections = append(sections, centeredError)
		} else {
			sections = append(sections, errorView)
		}
		return lipgloss.JoinVertical(lipgloss.Left, sections...)
	}

	state, ok := m.Judges[m.ActiveJudge]
	if !ok || state.CurrentTest == nil {
		loadingMsg := "Waiting for runner to initialize..."
		if m.ActiveJudge != "" {
			loadingMsg = fmt.Sprintf("Judge %s started, loading data...", m.ActiveJudge)
		}

		loadingView := lipgloss.NewStyle().Foreground(styles.Gray).Render(m.Spinner.View() + " " + loadingMsg)

		return lipgloss.JoinVertical(lipgloss.Left, sections...) + "\n" + loadingView
	}

	// TREE VIEW SECTION
	t := tree.New().
		Root(styles.HighlightStyle.Render(m.ActiveJudge))

	for i, res := range state.Results {
		questionText := res.Question
		if len(questionText) > 40 {
			questionText = questionText[:37] + "..."
		}

		elapsedStr := fmt.Sprintf("%.2fs", res.Elapsed.Seconds())
		roastElapsedStr := fmt.Sprintf("%.2fs", res.RoastElapsed.Seconds())
		roastText := lipgloss.NewStyle().Foreground(styles.Orange).Faint(true).Render(fmt.Sprintf("roasted in %s", roastElapsedStr))

		item := fmt.Sprintf("%d. %s    %s    %s",
			i+1,
			lipgloss.NewStyle().Foreground(styles.Gray).Render(questionText),
			lipgloss.NewStyle().Foreground(styles.Gray).Render(elapsedStr),
			roastText,
		)
		t.Child(item)
	}

	if len(state.Results) > 0 {
		sections = append(sections, t.String())
	}

	// Add warning only when running and no results yet
	if !m.Done && len(state.Results) == 0 {
		warningStyle := lipgloss.NewStyle().Foreground(styles.Yellow).Italic(true)
		sections = append(sections, warningStyle.Render("Process is running, please do not close the terminal"))
	}

	// PREPARE EVAL DATA
	summaryText := ""
	scoreText := ""
	if state.EvalBuffer.Len() > 0 {
		evalText := state.EvalBuffer.String()
		// Score
		if scoreIdx := strings.Index(evalText, "\"overall_score\":"); scoreIdx != -1 {
			offsetScore := scoreIdx + 16
			if offsetScore <= len(evalText) {
				sub := evalText[offsetScore:]
				if endIdx := strings.IndexAny(sub, ",}\n"); endIdx != -1 {
					scoreText = strings.TrimSpace(sub[:endIdx])
				}
			}
		}
		// Summary
		if summaryIdx := strings.Index(evalText, "\"summary\":"); summaryIdx != -1 {
			offsetSummary := summaryIdx + 11
			if offsetSummary <= len(evalText) {
				sub := evalText[offsetSummary:]
				if startQuote := strings.Index(sub, "\""); startQuote != -1 {
					sub = sub[startQuote+1:]
					if endQuote := strings.Index(sub, "\""); endQuote != -1 {
						summaryText = sub[:endQuote]
					}
				}
			}
		}
	}

	if m.RunnerError != nil {
		// ERROR VIEW
		errorView := lipgloss.JoinVertical(lipgloss.Left,
			styles.ErrorTextStyle.Render("Runner Error:"),
			styles.ErrorTextStyle.Render(m.RunnerError.Error()),
			"",
			styles.DimStyle.Render("Press esc to return"),
		)
		if m.Width > 0 && m.Height > 0 {
			centeredError := lipgloss.Place(boxWidth, 10, lipgloss.Center, lipgloss.Center, errorView)
			sections = append(sections, centeredError)
		} else {
			sections = append(sections, errorView)
		}
		sections = append(sections, "")
		sections = append(sections, styles.DimStyle.Render("esc: back to menu"))
		return strings.Join(sections, "\n")
	}

	if m.Done {
		// COMPLETION VIEW
		doneStyle := lipgloss.NewStyle().
			Align(lipgloss.Left).
			Padding(1, 0)

		totalPassed := 0
		totalFailed := 0
		totalPrompt := 0
		totalCompletion := 0
		totalTokens := 0
		for _, j := range m.Judges {
			totalPassed += j.Passed
			totalFailed += j.Failed
			totalPrompt += j.TotalUsage.PromptTokens
			totalCompletion += j.TotalUsage.CompletionTokens
			totalTokens += j.TotalUsage.TotalTokens
		}

		totalSeconds := m.TotalTime.Seconds()

		completionHeader := lipgloss.NewStyle().
			Bold(true).
			Foreground(styles.Green).
			Render("EVALUATION COMPLETE")

		content := lipgloss.JoinVertical(lipgloss.Left,
			completionHeader,
			"",
			fmt.Sprintf("%s %s", styles.DimStyle.Render("Total Time:"), styles.ValueStyle.Render(fmt.Sprintf("%.2fs", totalSeconds))),
			fmt.Sprintf("%s %s %s", styles.DimStyle.Render("Total Tokens:"), styles.ValueStyle.Render(fmt.Sprintf("%d", totalTokens)),
				styles.DimStyle.Render(fmt.Sprintf("(P:%d / C:%d)", totalPrompt, totalCompletion))),
			"",
			fmt.Sprintf("%s %s", styles.DimStyle.Render("Passed:"), styles.SuccessBgStyle.Render(fmt.Sprintf(" %d ", totalPassed))),
			fmt.Sprintf("%s %s", styles.DimStyle.Render("Failed:"), styles.ErrorBgStyle.Render(fmt.Sprintf(" %d ", totalFailed))),
			"",
		)

		for _, j := range m.Judges {
			if j.ExcelPath != "" {
				excelLine := fmt.Sprintf("%s %s",
					styles.DimStyle.Render("Report:"),
					styles.HighlightStyle.Render(filepath.Base(j.ExcelPath)))
				pathLine := styles.DimStyle.Render(fmt.Sprintf("       (%s)", j.ExcelPath))
				content = lipgloss.JoinVertical(lipgloss.Left, content, excelLine, pathLine, "")
			}
		}

		content = lipgloss.JoinVertical(lipgloss.Left, content,
			lipgloss.JoinHorizontal(lipgloss.Top,
				styles.HighlightStyle.Render("esc"), styles.DimStyle.Render(" back to menu"),
			),
		)

		sections = append(sections, doneStyle.Render(content))
	} else {
		// STREAMING VIEW with TREE LAYOUT
		usableWidth := contentWidth - 2
		if usableWidth < 0 {
			usableWidth = 0
		}

		// Status Line
		var statusText string
		ctaStyle := lipgloss.NewStyle().Foreground(styles.Cyan)

		if state.State == StateGenerating {
			text := "Generating response..."
			if state.HoldOnMessage != "" {
				text = state.HoldOnMessage
			}

			renderedText := ctaStyle.Render(text)

			// Add ParseAs info if available
			if m.SelectedVendor != nil && m.SelectedVendor.ParseAs != "" {
				renderedText += styles.DimStyle.Render(fmt.Sprintf(" parsed from %s", m.SelectedVendor.ParseAs))
			}

			statusText = fmt.Sprintf("%s %s", m.Spinner.View(), renderedText)
		} else {
			text := "Roasting with Judges ..."
			if state.HoldOnMessage != "" {
				text = state.HoldOnMessage
			}
			evalText := lipgloss.NewStyle().Foreground(styles.Orange).Render(text)
			statusText = fmt.Sprintf("%s %s", m.EvalSpinner.View(), evalText)
		}

		// Create TREE VIEW for Question and Streaming Answer
		questionTree := tree.New()

		// Format the question/status for tree root
		var rootLabel string
		if summaryText == "" && state.CurrentTest != nil {
			// Truncate question for display
			questionText := state.CurrentTest.Prompt
			if len(questionText) > 50 {
				questionText = questionText[:47] + "..."
			}
			rootLabel = statusText + " | " + questionText
		} else {
			rootLabel = statusText
		}

		questionTree.Root(lipgloss.NewStyle().
			Foreground(styles.Purple).
			Bold(true).
			Render(rootLabel))

		// Response streaming as first child
		wrappedResponse := wrapText(state.StreamBuffer.String(), usableWidth)
		respLines := strings.Split(wrappedResponse, "\n")
		if len(respLines) > maxResponseLines {
			respLines = respLines[len(respLines)-maxResponseLines:]
		}
		for len(respLines) < maxResponseLines {
			respLines = append(respLines, "")
		}
		questionTree.Child(strings.Join(respLines, "\n"))

		// Add LIVE ANALYSIS as child tree if evaluating
		if state.State == StateEvaluating || state.EvalBuffer.Len() > 0 {
			evalText := state.EvalBuffer.String()

			liveScore := ""
			if scoreText != "" {
				liveScore = lipgloss.NewStyle().
					Foreground(styles.Green).
					Bold(true).
					Render("Score: " + scoreText + "/100")
			}

			liveSummary := ""
			if summaryText != "" {
				liveSummary = lipgloss.NewStyle().
					Foreground(styles.Cyan).
					Render("Summary: " + summaryText)
			}

			coloredJSON := colorizeJSON(evalText, usableWidth)
			evalLines := strings.Split(coloredJSON, "\n")

			if len(evalLines) > maxEvalLines {
				evalLines = evalLines[len(evalLines)-maxEvalLines:]
			}
			for len(evalLines) < maxEvalLines {
				evalLines = append(evalLines, "")
			}

			// Create LIVE ANALYSIS as subtree child
			analysisTree := tree.New().
				Root(lipgloss.NewStyle().
					Foreground(styles.Orange).
					Bold(true).
					Render("LIVE ANALYSIS"))

			if liveScore != "" {
				analysisTree.Child(liveScore)
			}
			if liveSummary != "" {
				analysisTree.Child(liveSummary)
			}
			analysisTree.Child(lipgloss.NewStyle().
				Foreground(styles.Gray).
				Faint(true).
				Render("RAW DATA STREAM"))
			analysisTree.Child(strings.Join(evalLines, "\n"))

			// Add as child of main tree
			questionTree.Child(analysisTree.String())
		}

		// Add gap between completed results and current running test
		if len(state.Results) > 0 {
			sections = append(sections, "")
		}

		sections = append(sections, questionTree.String())
	}

	// Add gap before footer
	sections = append(sections, "")

	// FOOTER - LINE 1: Judge info & Tests
	infoLabelStyle := lipgloss.NewStyle().Foreground(styles.Gray).Faint(true)
	infoValueStyle := lipgloss.NewStyle().Foreground(styles.White)

	line1 := fmt.Sprintf("%s %s  %s  %s %s/%s  %s  %s %s",
		infoLabelStyle.Render("Judge:"), infoValueStyle.Render(m.ActiveJudge),
		lipgloss.NewStyle().Foreground(styles.Gray).Render("•"),
		infoLabelStyle.Render("Model:"), infoValueStyle.Render(state.Config.LLM.Provider), infoValueStyle.Render(state.Config.LLM.Model),
		lipgloss.NewStyle().Foreground(styles.Gray).Render("•"),
		infoLabelStyle.Render("Dataset:"), infoValueStyle.Render(state.Config.DataPath))

	// FOOTER - LINE 2: Tokens & Help
	usageText := fmt.Sprintf("%s In:%d / Out:%d  %s %d",
		infoLabelStyle.Render("Token"),
		state.TotalUsage.PromptTokens, state.TotalUsage.CompletionTokens,
		infoLabelStyle.Render("Σ:"), state.TotalUsage.TotalTokens)

	helpText := lipgloss.NewStyle().Foreground(styles.Gray).Faint(true).Render("esc: stop")
	if len(m.Judges) > 1 {
		helpText = lipgloss.NewStyle().Foreground(styles.Gray).Faint(true).Render(fmt.Sprintf("tab/arrows: switch judge (%d/%d) | esc: stop", m.getJudgeIndex()+1, len(m.Judges)))
	}

	usageWidth := lipgloss.Width(usageText)
	helpWidth := lipgloss.Width(helpText)

	// Spacer for line 2
	spacer2 := boxWidth - usageWidth - helpWidth - 2
	if spacer2 < 0 {
		spacer2 = 0
	}

	line2 := lipgloss.JoinHorizontal(lipgloss.Top, usageText, strings.Repeat(" ", spacer2), helpText)

	sections = append(sections, line1)
	sections = append(sections, line2)

	return strings.Join(sections, "\n")
}

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

func colorizeJSON(text string, maxWidth int) string {
	if text == "" {
		return ""
	}

	// 1. Numbers/Booleans: only match after : or [ or , to avoid ANSI codes
	reNums := regexp.MustCompile(`(:|,|\[)\s*(true|false|null|\d+(\.\d+)?)`)
	text = reNums.ReplaceAllStringFunc(text, func(s string) string {
		// Keep the prefix (:|,|[)
		prefix := s[:1]
		val := s[1:]
		return prefix + lipgloss.NewStyle().Foreground(styles.Yellow).Render(val)
	})

	// 2. Keys: "key":
	reKeys := regexp.MustCompile(`"([^"]+)":`)
	text = reKeys.ReplaceAllStringFunc(text, func(s string) string {
		return lipgloss.NewStyle().Foreground(styles.Purple).Render(s)
	})

	// 3. String values (that are not keys)
	// This is tricky but we can match "value" if not followed by :
	// For simplicity, let's just do keys and numbers/bools for now to avoid complexity
	// and add a dim style to the whole thing first.

	return lipgloss.NewStyle().Width(maxWidth).Render(text)
}

func (m *RunModel) cycleJudge(delta int) {
	if len(m.Judges) <= 1 {
		return
	}

	// Get sorted keys for consistent cycling
	var keys []string
	// We use the order of judges in config if possible
	for _, j := range m.Config.Judges {
		if _, ok := m.Judges[j.Name]; ok {
			keys = append(keys, j.Name)
		}
	}

	if len(keys) == 0 {
		return
	}

	currentIndex := -1
	for i, k := range keys {
		if k == m.ActiveJudge {
			currentIndex = i
			break
		}
	}

	if currentIndex == -1 {
		m.ActiveJudge = keys[0]
		return
	}

	nextIndex := (currentIndex + delta + len(keys)) % len(keys)
	m.ActiveJudge = keys[nextIndex]
}

func (m *RunModel) getJudgeIndex() int {
	var keys []string
	for _, j := range m.Config.Judges {
		if _, ok := m.Judges[j.Name]; ok {
			keys = append(keys, j.Name)
		}
	}

	for i, k := range keys {
		if k == m.ActiveJudge {
			return i
		}
	}
	return 0
}

func msgTypeIsEnter(msg tea.Msg) bool {
	if key, ok := msg.(tea.KeyMsg); ok {
		return key.String() == "enter"
	}
	return false
}

func waitForEvent(ch <-chan runner.Event) tea.Cmd {
	return func() tea.Msg {
		if event, ok := <-ch; ok {
			return event
		}
		return nil
	}
}
