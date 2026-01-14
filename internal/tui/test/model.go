package test

import (
	"fmt"
	"strings"
	"time"

	"openjudges/internal/styles"
	"openjudges/internal/tui/components"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/tree"
)

const (
	totalIterations  = 10
	maxQuestionLen   = 100
	maxResponseLines = 3
	dummyQuestion    = "What are the key differences between supervised and unsupervised learning in machine learning, and how do you decide which approach to use?"
	dummyResponse    = "Supervised learning uses labeled data where the algorithm learns from input-output pairs to make predictions on new data. The model is trained with examples that have known correct answers. Unsupervised learning works with unlabeled data, finding patterns and structures without predefined outputs. It's useful for clustering and dimensionality reduction. The choice depends on your data availability and problem type: use supervised when you have labeled data and need predictions, and unsupervised when exploring data patterns or when labels are unavailable or expensive to obtain."
	maxEvalLines     = 2
	dummyEval        = "Evaluating the response against criteria... Checking for clarity and technical accuracy. The explanation of supervised and unsupervised learning is correct. Decision criteria are well-formulated. Score: 95/100. Recommendations: Keep up the good work!"
)

type TestState int

const (
	StateGenerating TestState = iota
	StateEvaluating
)

var monkeySpinner = spinner.Spinner{
	Frames: []string{"🙈", "🙉", "🙊", "🙉"},
	FPS:    time.Second / 6,
}

type tickMsg time.Time

type IterationResult struct {
	Iteration    int
	Question     string
	Elapsed      time.Duration
	RoastElapsed time.Duration
}

type TestModel struct {
	iteration    int
	total        int
	question     string
	streamBuffer strings.Builder
	streamIndex  int
	evalBuffer   strings.Builder
	evalIndex    int
	state        TestState

	results       []IterationResult
	iterStartTime time.Time
	evalStartTime time.Time

	isDone    bool
	totalTime time.Duration

	spinner     spinner.Model
	evalSpinner spinner.Model
	startTime   time.Time
	elapsed     float64
	judgeName   string
	judgeInfo   string
	vendorInfo  string
	promptFile  string
	datasetFile string

	help     help.Model
	quitting bool
	width    int
	height   int
	keys     KeyMap
}

type KeyMap struct {
	Quit key.Binding
}

var keys = KeyMap{
	Quit: key.NewBinding(
		key.WithKeys("esc", "ctrl+c"),
		key.WithHelp("esc", "stop"),
	),
}

func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Quit}
}

func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{{k.Quit}}
}

func NewTestModel() *TestModel {
	s := spinner.New()
	s.Spinner = spinner.Points
	s.Style = lipgloss.NewStyle().Foreground(styles.Purple)

	es := spinner.New()
	es.Spinner = monkeySpinner
	es.Style = lipgloss.NewStyle().Foreground(styles.Orange)

	// Truncate question if needed
	question := dummyQuestion
	if len(question) > maxQuestionLen {
		question = question[:maxQuestionLen-3] + "..."
	}

	now := time.Now()

	return &TestModel{
		iteration:     1,
		total:         totalIterations,
		question:      question,
		spinner:       s,
		startTime:     now,
		iterStartTime: now,
		judgeName:     "Judges-21320",
		judgeInfo:     "GPT-4o",
		vendorInfo:    "OpenAI API",
		help:          help.New(),
		keys:          keys,
		state:         StateGenerating,
		evalSpinner:   es,
		promptFile:    "~super.md",
		datasetFile:   "~maindata.csv",
	}
}

func (m *TestModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		m.evalSpinner.Tick,
		tickCmd(),
	)
}

func (m *TestModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.isDone {
			switch msg.String() {
			case "v", "enter":
				// For now just stay here, in real app might go to results
				return m, nil
			case "b", "esc":
				return m, tea.Quit
			}
		}

		switch {
		case key.Matches(msg, m.keys.Quit):
			m.quitting = true
			return m, tea.Quit
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tickMsg:
		// Update elapsed time (never reset until all iterations complete)
		m.elapsed = time.Since(m.startTime).Seconds()

		if m.iteration > m.total {
			m.quitting = true
			return m, tea.Quit
		}

		// Simulate streaming response character by character
		if m.state == StateGenerating {
			if m.streamIndex < len(dummyResponse) {
				// Add 2-5 characters per tick for smoother streaming
				charsToAdd := 2 + (m.streamIndex % 4)
				end := m.streamIndex + charsToAdd
				if end > len(dummyResponse) {
					end = len(dummyResponse)
				}
				m.streamBuffer.WriteString(dummyResponse[m.streamIndex:end])
				m.streamIndex = end

				cmds = append(cmds, tickCmd())
			} else {
				// Move to Evaluating state
				m.state = StateEvaluating
				m.evalStartTime = time.Now()
				cmds = append(cmds, tickCmd())
			}
		} else if m.state == StateEvaluating {
			if m.evalIndex < len(dummyEval) {
				// Add 2-5 characters per tick for smoother streaming
				charsToAdd := 2 + (m.evalIndex % 4)
				end := m.evalIndex + charsToAdd
				if end > len(dummyEval) {
					end = len(dummyEval)
				}
				m.evalBuffer.WriteString(dummyEval[m.evalIndex:end])
				m.evalIndex = end

				cmds = append(cmds, tickCmd())
			} else {
				// Completed one iteration, move to next
				m.results = append(m.results, IterationResult{
					Iteration:    m.iteration,
					Question:     m.question,
					Elapsed:      time.Since(m.iterStartTime),
					RoastElapsed: time.Since(m.evalStartTime),
				})

				m.iteration++
				if m.iteration <= m.total {
					// Reset for next iteration (DON'T reset startTime or elapsed)
					m.streamBuffer.Reset()
					m.streamIndex = 0
					m.evalBuffer.Reset()
					m.evalIndex = 0
					m.state = StateGenerating
					m.iterStartTime = time.Now()
					cmds = append(cmds, tickCmd())
				} else {
					// All iterations complete
					m.isDone = true
					m.totalTime = time.Since(m.startTime)
					return m, nil
				}
			}
		}
	}

	// Handle spinner tick
	if sMsg, ok := msg.(spinner.TickMsg); ok {
		var cmd tea.Cmd
		if sMsg.ID == m.spinner.ID() {
			m.spinner, cmd = m.spinner.Update(msg)
		} else if sMsg.ID == m.evalSpinner.ID() {
			m.evalSpinner, cmd = m.evalSpinner.Update(msg)
		}
		return m, cmd
	}

	return m, tea.Batch(cmds...)
}

func (m *TestModel) View() string {
	if m.quitting {
		return ""
	}

	// Calculate box width (terminal width - potential outer margins if any, but sections are joined directly)
	fullWidth := m.width
	if fullWidth <= 0 {
		fullWidth = 80
	}

	boxWidth := fullWidth
	if boxWidth > 100 { // Cap max width for readability on very wide terminals
		boxWidth = 100
	}

	// Internal width for content (boxWidth - 2 for border - 2 for padding)
	contentWidth := boxWidth - 4
	if contentWidth < 20 {
		contentWidth = 20
	}

	var sections []string

	// HEADER: ASCII Art and Version
	// RenderHeader already uses width-4 internally for infoLine
	sections = append(sections, components.RenderHeader(boxWidth))
	sections = append(sections, "")

	// VIEW ASSEMBLY
	if !m.isDone {
		warningStyle := lipgloss.NewStyle().
			Foreground(styles.Yellow).
			Italic(true)
		warningText := warningStyle.Render("Process is running, please do not close the terminal")
		sections = append(sections, warningText)
		sections = append(sections, "")
	}

	// TREE VIEW SECTION
	t := tree.New().
		Root(styles.HighlightStyle.Render(m.judgeName))

	for _, res := range m.results {
		questionText := res.Question
		if len(questionText) > 50 {
			questionText = questionText[:47] + "..."
		}

		elapsedStr := fmt.Sprintf("%.2fs", res.Elapsed.Seconds())
		roastElapsedStr := fmt.Sprintf("%.2fs", res.RoastElapsed.Seconds())
		roastText := lipgloss.NewStyle().Foreground(styles.Orange).Faint(true).Render(fmt.Sprintf("roasted in %s", roastElapsedStr))

		item := fmt.Sprintf("%s    %s    %s",
			lipgloss.NewStyle().Foreground(styles.Gray).Render(questionText),
			lipgloss.NewStyle().Foreground(styles.Gray).Render(fmt.Sprintf("%s / %d", elapsedStr, res.Iteration)),
			roastText,
		)
		t.Child(item)
	}

	if len(m.results) > 0 {
		sections = append(sections, t.String())
		sections = append(sections, "")
	}

	// TOP LINE: Labeled Question
	questionLabelStyle := lipgloss.NewStyle().
		Foreground(styles.Gray)

	questionStyle := lipgloss.NewStyle().
		Foreground(styles.Purple).
		Bold(true)

	// Wrap question text
	wrappedQuestion := wrapText(m.question, boxWidth-11) // "Question: " is 10 chars
	question := questionLabelStyle.Render("Question: ") + questionStyle.Render(wrappedQuestion)

	if m.isDone {
		// COMPLETION BOX
		doneStyle := lipgloss.NewStyle().
			Width(contentWidth).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(styles.Green).
			Padding(1, 2)

		totalStr := fmt.Sprintf("%.2fs", m.totalTime.Seconds())
		content := lipgloss.JoinVertical(lipgloss.Left,
			styles.SuccessTextStyle.Render("Judges is done..."),
			"",
			fmt.Sprintf("%s %s", styles.DimStyle.Render("Total Time Elapsed:"), styles.ValueStyle.Render(totalStr)),
			"",
			lipgloss.JoinHorizontal(lipgloss.Top,
				styles.HighlightStyle.Render("v"), styles.DimStyle.Render(" view result   "),
				styles.HighlightStyle.Render("esc"), styles.DimStyle.Render(" back to menu"),
			),
		)
		sections = append(sections, doneStyle.Render(content))
	} else {
		sections = append(sections, question)
		sections = append(sections, "")

		// STREAMING RESPONSE SECTION
		responseStyle := lipgloss.NewStyle().
			Width(contentWidth).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(styles.Purple).
			Padding(0, 1)

		if m.state == StateEvaluating {
			responseStyle = responseStyle.BorderForeground(styles.Orange)
		}

		// Usable width inside the padded box
		usableWidth := contentWidth - 2
		if usableWidth < 0 {
			usableWidth = 0
		}

		responseText := m.streamBuffer.String()
		wrappedContent := wrapText(responseText, usableWidth)

		// Take last 3 lines (tail behavior)
		allWrappedLines := strings.Split(wrappedContent, "\n")
		displayLines := []string{}
		startIdx := len(allWrappedLines) - maxResponseLines
		if startIdx < 0 {
			startIdx = 0
		}

		for i := startIdx; i < len(allWrappedLines); i++ {
			displayLines = append(displayLines, allWrappedLines[i])
		}

		// Pad to ensure consistent 3-line height
		for len(displayLines) < maxResponseLines {
			displayLines = append(displayLines, "")
		}

		// BOX CONTENT ASSEMBLY
		ctaStyle := lipgloss.NewStyle().
			Foreground(styles.Cyan).
			Bold(false)

		statsStyle := lipgloss.NewStyle().
			Foreground(styles.Gray).
			Faint(true)

		var statusText string
		if m.state == StateGenerating {
			statusText = fmt.Sprintf("%s %s", m.spinner.View(), ctaStyle.Render("Generating response..."))
		} else {
			evalText := lipgloss.NewStyle().Foreground(styles.Orange).Render("Roasting with Judges ...")
			statusText = fmt.Sprintf("%s %s", m.evalSpinner.View(), evalText)
		}

		duration := time.Duration(m.elapsed * float64(time.Second)).Round(time.Second)
		statsText := statsStyle.Render(fmt.Sprintf("⏱︎ %s | ∞ %d/%d", duration.String(), m.iteration, m.total))

		// Calculate space for right alignment
		statusWidth := lipgloss.Width(statusText)
		statsWidth := lipgloss.Width(statsText)
		spaceWidth := usableWidth - statusWidth - statsWidth
		if spaceWidth < 0 {
			spaceWidth = 0
		}

		boxHeader := statusText + strings.Repeat(" ", spaceWidth) + statsText

		// Evaluation stream (if in evaluating state)
		evalView := ""
		if m.state == StateEvaluating {
			evalContent := m.evalBuffer.String()
			wrappedEval := wrapText(evalContent, usableWidth)
			evalLines := strings.Split(wrappedEval, "\n")
			tailEvalLines := []string{}
			evalStartIdx := len(evalLines) - maxEvalLines
			if evalStartIdx < 0 {
				evalStartIdx = 0
			}
			for i := evalStartIdx; i < len(evalLines); i++ {
				tailEvalLines = append(tailEvalLines, evalLines[i])
			}
			// Pad
			for len(tailEvalLines) < maxEvalLines {
				tailEvalLines = append(tailEvalLines, "")
			}
			evalView = "\n" + lipgloss.NewStyle().
				Foreground(styles.Gray).
				PaddingTop(1).
				Width(usableWidth).
				Border(lipgloss.NormalBorder(), true, false, false, false).
				BorderForeground(styles.DarkGray).
				Render(strings.Join(tailEvalLines, "\n"))
		}

		responseContent := boxHeader + "\n\n" + strings.Join(displayLines, "\n") + evalView

		sections = append(sections, responseStyle.Render(responseContent))
	}
	sections = append(sections, "")

	// BOTTOM INFO: Judge and vendor info (left) | Help (right)
	infoStyle := lipgloss.NewStyle().
		Foreground(styles.Gray)

	labelStyle := lipgloss.NewStyle().
		Foreground(styles.Gray).
		Faint(true)

	valueStyle := lipgloss.NewStyle().
		Foreground(styles.White)

	helpStyle := lipgloss.NewStyle().
		Foreground(styles.Gray).
		Faint(true)

	leftInfo := fmt.Sprintf("%s %s  %s  %s %s  %s  %s %s",
		labelStyle.Render("LLM:"),
		valueStyle.Render(m.judgeInfo),
		infoStyle.Render("•"),
		labelStyle.Render("Prompt:"),
		valueStyle.Render(m.promptFile),
		infoStyle.Render("•"),
		labelStyle.Render("Dataset:"),
		valueStyle.Render(m.datasetFile),
	)

	helpText := helpStyle.Render("Press esc to stop")

	leftInfoWidth := lipgloss.Width(leftInfo)
	helpWidth := lipgloss.Width(helpText)
	bottomSpacerWidth := boxWidth - leftInfoWidth - helpWidth - 2
	if bottomSpacerWidth < 0 {
		bottomSpacerWidth = 0
	}

	bottomLine := lipgloss.JoinHorizontal(lipgloss.Top,
		leftInfo,
		strings.Repeat(" ", bottomSpacerWidth),
		helpText,
	)

	sections = append(sections, bottomLine)

	return strings.Join(sections, "\n")
}

func tickCmd() tea.Cmd {
	return tea.Tick(time.Millisecond*50, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func wrapText(text string, maxWidth int) string {
	if maxWidth <= 0 {
		return text
	}
	var wrappedLines []string
	splitLines := strings.Split(text, "\n")

	for _, line := range splitLines {
		if len(line) <= maxWidth {
			wrappedLines = append(wrappedLines, line)
		} else {
			// Simple word wrap
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

				if len(testLine) <= maxWidth {
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
