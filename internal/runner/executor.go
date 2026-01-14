package runner

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"sync"
	"time"

	"openjudges/config"
	"openjudges/internal/export"
	"openjudges/internal/vendor"
	"openjudges/judge"
	"openjudges/testcase"
)

// Event represents a runner event
type Event interface {
	isEvent()
}

// Executor orchestrates judge execution with progress reporting via events
type Executor struct {
	cfg       *config.Config
	eventChan chan Event
	ctx       context.Context
	cancel    context.CancelFunc
	vendor    *vendor.VendorConfig // Optional vendor for response generation
}

var holdOnMessages = []string{
	"umm...hold on",
	"let me think for a second...",
	"wait, let me check that...",
	"hmm, interesting...",
	"hold on, processing...",
	"stay tuned...",
	"preparing the roast...",
	"checking the logic...",
	"one moment...",
	"please wait...",
	"hold on...",
	"cooking up the roast...",
	"let me think...",
}

// NewExecutor creates a new executor
func NewExecutor(cfg *config.Config) *Executor {
	ctx, cancel := context.WithCancel(context.Background())

	return &Executor{
		cfg:       cfg,
		ctx:       ctx,
		cancel:    cancel,
		eventChan: make(chan Event, 100), // Buffer events
	}
}

// Events returns the event channel
func (e *Executor) Events() <-chan Event {
	return e.eventChan
}

// Run executes the specified judges
func (e *Executor) Run(judgeNames []string) error {
	// Get judges to run
	judgesToRun := e.getJudgesToRun(judgeNames)
	if len(judgesToRun) == 0 {
		return fmt.Errorf("no judges found to run")
	}

	// Determine parallelism
	parallel := e.cfg.Options.Parallel
	if parallel <= 0 {
		// Default to a sensible parallel limit if 0,
		// but if they want to avoid spam, we should probably follow the config.
		// Let's treat 0 as "all in parallel" to maintain backward compatibility,
		// but provide a way to limit it.
		parallel = len(judgesToRun)
	}

	// Semaphore to limit parallelism
	sem := make(chan struct{}, parallel)

	var wg sync.WaitGroup
	for i, jc := range judgesToRun {
		wg.Add(1)
		go func(idx int, judgeConfig config.JudgeConfig) {
			defer wg.Done()

			// Stagger start to avoid simultaneous bursts across judges
			if parallel > 1 {
				time.Sleep(time.Duration(idx) * 200 * time.Millisecond)
			}

			sem <- struct{}{}        // Acquire
			defer func() { <-sem }() // Release
			e.runJudge(judgeConfig)
		}(i, jc)
	}

	// Wait for all judges to complete
	wg.Wait()

	return nil
}

// Cancel cancels the execution
func (e *Executor) Cancel() {
	e.cancel()
}

// SetVendor sets the vendor for response generation
// When set, the vendor API will be used instead of the judge's LLM
func (e *Executor) SetVendor(v *vendor.VendorConfig) {
	e.vendor = v
}

// getJudgesToRun returns the judges to execute based on the provided names
func (e *Executor) getJudgesToRun(judgeNames []string) []config.JudgeConfig {
	if len(judgeNames) == 0 {
		// Run all judges
		return e.cfg.Judges
	}

	// Run specific judges
	var result []config.JudgeConfig
	for _, name := range judgeNames {
		for _, jc := range e.cfg.Judges {
			if jc.Name == name {
				result = append(result, jc)
				break
			}
		}
	}
	return result
}

// runJudge executes a single judge
func (e *Executor) runJudge(jc config.JudgeConfig) {
	// Load test cases
	testCases, err := testcase.Load(jc.DataPath)
	if err != nil {
		e.emit(ErrorMsg{JudgeName: jc.Name, Error: fmt.Errorf("failed to load test cases: %v", err)})
		return
	}
	if len(testCases) == 0 {
		e.emit(ErrorMsg{JudgeName: jc.Name, Error: fmt.Errorf("no test cases found in %s", jc.DataPath)})
		return
	}

	e.emit(JudgeStartMsg{Name: jc.Name, Total: len(testCases), Config: jc})

	// Create judge instance
	jg := judge.NewJudge(jc)
	var results []testcase.JudgeResult
	var tokenData []export.TokenUsageData

	// Evaluate each test case
	for _, tc := range testCases {
		// Check if context is cancelled
		select {
		case <-e.ctx.Done():
			e.emit(ErrorMsg{JudgeName: jc.Name, Error: fmt.Errorf("cancelled")})
			return
		default:
		}

		e.emit(TestStartMsg{JudgeName: jc.Name, TestID: tc.ID, TestData: tc})

		// Configure streaming
		streamConfig := judge.StreamConfig{
			OnToken: func(token string, tokenType judge.TokenType, usage judge.TokenUsage) {
				e.emit(TokenStreamMsg{
					JudgeName: jc.Name,
					TestID:    tc.ID,
					Token:     token,
					TokenType: tokenType,
					Usage:     usage,
				})
			},
		}

		// Step 1: Generate AI response (Chat)
		// Use vendor API if set, otherwise use judge's LLM
		var chatFullText string
		if e.vendor != nil {
			// Use vendor API - pass the whole test case for flexible placeholders
			response, err := e.vendor.CallVendorAPI(e.ctx, tc)
			if err != nil {
				e.emit(ErrorMsg{JudgeName: jc.Name, TestID: tc.ID, Error: fmt.Errorf("vendor API failed: %v", err)})
				continue
			}
			chatFullText = response
			// Emit the response as a token stream for UI display
			e.emit(TokenStreamMsg{
				JudgeName: jc.Name,
				TestID:    tc.ID,
				Token:     chatFullText,
				TokenType: judge.TokenContent,
			})
		} else {
			// Use builtin AI (existing behavior)
			var err error
			chatFullText, _, err = jg.EvaluateStreaming(e.ctx, tc.Prompt, streamConfig)
			if err != nil {
				e.emit(ErrorMsg{JudgeName: jc.Name, TestID: tc.ID, Error: fmt.Errorf("generation failed: %v", err)})
				continue
			}
		}

		// Update AIResponse for evaluation
		tc.AIResponse = chatFullText

		// Step 2: Add delay and "hold on" message
		e.applyDelayWithMsg(tc.ID, jc.Name)

		// Step 3: Evaluate with streaming (Roasting)
		e.emit(RoastingStartMsg{JudgeName: jc.Name, TestID: tc.ID})

		// We use PrepareEvaluationPrompt which handles the template or default prompt
		evalPrompt := jg.PrepareEvaluationPrompt(tc)
		fullText, tokens, err := jg.EvaluateStreaming(e.ctx, evalPrompt, streamConfig)

		e.emit(StreamCompleteMsg{JudgeName: jc.Name, TestID: tc.ID})

		var evalResult *testcase.JudgeResult
		if err != nil {
			e.emit(ErrorMsg{JudgeName: jc.Name, TestID: tc.ID, Error: err})
			continue
		}

		// Parse result
		evalResult, err = jg.ParseJudgeResponse(fullText, tc.ID)
		if err != nil {
			e.emit(ErrorMsg{JudgeName: jc.Name, TestID: tc.ID, Error: fmt.Errorf("parse error: %v", err)})
			continue
		}

		// Update results
		evalResult.Prompt = tc.Prompt
		evalResult.AIResponse = tc.AIResponse
		evalResult.Expectation = tc.Expectation

		results = append(results, *evalResult)
		tokenData = append(tokenData, export.TokenUsageData{
			TestID:           tc.ID,
			PromptTokens:     tokens.PromptTokens,
			CompletionTokens: tokens.CompletionTokens,
			TotalTokens:      tokens.TotalTokens,
		})
		e.emit(TestCompleteMsg{
			JudgeName: jc.Name,
			TestID:    tc.ID,
			Result:    *evalResult,
			Tokens:    tokens.TotalTokens,
		})

		// Final delay after completion of each test case before the next one
		// to ensure visual separation and avoid rapid-fire requests
		e.applyDelayWithMsg(tc.ID, jc.Name)
	}

	// Save results
	excelPath, err := e.saveResults(jc.ResultPath, jc.Name, results, tokenData)
	if err != nil {
		e.emit(ErrorMsg{JudgeName: jc.Name, Error: fmt.Errorf("failed to save results: %v", err)})
		return
	} else {
		// Log the Excel path for the summary
		e.emit(TestCompleteMsg{
			JudgeName: jc.Name,
			TestID:    "SUMMARY",
			Result: testcase.JudgeResult{
				Summary: fmt.Sprintf("Excel report generated: %s", excelPath),
			},
		})
	}

	e.emit(JudgeCompleteMsg{JudgeName: jc.Name})
}

func (e *Executor) emit(event Event) {
	e.eventChan <- event
}

// saveResults saves evaluation results to a JSON file and Excel report
func (e *Executor) saveResults(path string, judgeName string, results []testcase.JudgeResult, tokenData []export.TokenUsageData) (string, error) {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return "", err
		}
	}

	// Create output structure
	output := struct {
		JudgeName string                 `json:"judge_name"`
		Timestamp string                 `json:"timestamp"`
		Total     int                    `json:"total"`
		Passed    int                    `json:"passed"`
		Failed    int                    `json:"failed"`
		Results   []testcase.JudgeResult `json:"results"`
	}{
		JudgeName: judgeName,
		Timestamp: time.Now().Format(time.RFC3339),
		Total:     len(results),
		Results:   results,
	}

	// Count passed/failed
	for _, r := range results {
		if r.Passed {
			output.Passed++
		} else {
			output.Failed++
		}
	}

	// Marshal to JSON
	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return "", err
	}

	// Write JSON file
	if err := os.WriteFile(path, data, 0644); err != nil {
		return "", err
	}

	// Export to Excel with timestamped filename
	excelFilename := export.GenerateFilename(judgeName, time.Now())
	excelPath := filepath.Join(dir, excelFilename)

	exportData := export.ExportData{
		JudgeName: judgeName,
		Results:   results,
		TokenData: tokenData,
	}

	if err := export.ExportToExcel(exportData, excelPath); err != nil {
		// Log warning but don't fail the run
		log.Printf("Warning: Excel export failed: %v", err)
	}

	return excelPath, nil
}

// applyDelayWithMsg emits a random hold on message and then applies the configured delay
func (e *Executor) applyDelayWithMsg(testID string, judgeName string) {
	randomMsg := holdOnMessages[rand.Intn(len(holdOnMessages))]
	e.emit(HoldOnMsg{JudgeName: judgeName, TestID: testID, Message: randomMsg})
	e.applyDelay()
}

func (e *Executor) applyDelay() {
	options := e.cfg.Options
	delayMs := options.DelayMs
	jitterMs := options.JitterMs

	// Fallback if not configured (User requested 2000/5000)
	if delayMs <= 0 {
		delayMs = 2000
	}
	if jitterMs <= 0 {
		jitterMs = 5000
	}

	totalDelay := time.Duration(delayMs) * time.Millisecond
	if jitterMs > 0 {
		// Use local rand generator to be safer (though global is seeded in 1.20+)
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		totalDelay += time.Duration(r.Intn(jitterMs)) * time.Millisecond
	}

	// Use a timer with context support
	timer := time.NewTimer(totalDelay)
	defer timer.Stop()

	select {
	case <-e.ctx.Done():
	case <-timer.C:
	}
}
