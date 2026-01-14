package export

import (
	"fmt"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"

	"openllmjudge/testcase"
)

// ExportData holds all data needed for Excel export
type ExportData struct {
	JudgeName string
	Results   []testcase.JudgeResult
	TokenData []TokenUsageData
}

// TokenUsageData holds token usage per test
type TokenUsageData struct {
	TestID           string
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}

// GenerateFilename creates a filename with timestamp format: JudgeName_DD-MM-YYYY_HH-MM-SS.xlsx
func GenerateFilename(judgeName string, t time.Time) string {
	timestamp := t.Format("2006-01-02_15-04-05")
	// Sanitize judge name for filename
	safeName := strings.ReplaceAll(judgeName, " ", "_")
	safeName = strings.ReplaceAll(safeName, "/", "-")
	return fmt.Sprintf("%s_%s.xlsx", safeName, timestamp)
}

// ExportToExcel generates an Excel report with results and charts
func ExportToExcel(data ExportData, outputPath string) error {
	f := excelize.NewFile()
	defer f.Close()

	// Create Results sheet
	resultsSheet := "Results"
	f.SetSheetName("Sheet1", resultsSheet)

	// Create Charts sheet
	chartsSheet := "Charts"
	if _, err := f.NewSheet(chartsSheet); err != nil {
		return fmt.Errorf("failed to create charts sheet: %w", err)
	}

	// Write results data
	if err := writeResultsSheet(f, resultsSheet, data); err != nil {
		return fmt.Errorf("failed to write results: %w", err)
	}

	// Create charts
	if err := createCharts(f, chartsSheet, data); err != nil {
		return fmt.Errorf("failed to create charts: %w", err)
	}

	// Set active sheet to Results
	idx, _ := f.GetSheetIndex(resultsSheet)
	f.SetActiveSheet(idx)

	// Save file
	if err := f.SaveAs(outputPath); err != nil {
		return fmt.Errorf("failed to save Excel file: %w", err)
	}

	return nil
}

func writeResultsSheet(f *excelize.File, sheet string, data ExportData) error {
	// Define headers
	headers := []string{
		"Test ID", "Question", "AI Response", "Expectation",
		"Passed", "Score", "Grade", "Summary",
		"Accuracy", "Completeness", "Clarity", "Relevance", "Actionability",
		"Critical Issues", "Improvements", "Exemplary Elements",
		"Confidence", "Prompt Tokens", "Completion Tokens", "Total Tokens",
	}

	// Header style
	headerStyle, err := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"4472C4"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true},
		Border: []excelize.Border{
			{Type: "left", Color: "000000", Style: 1},
			{Type: "top", Color: "000000", Style: 1},
			{Type: "bottom", Color: "000000", Style: 1},
			{Type: "right", Color: "000000", Style: 1},
		},
	})
	if err != nil {
		return err
	}

	// Write headers
	for i, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheet, cell, header)
		f.SetCellStyle(sheet, cell, cell, headerStyle)
	}

	// Alternating row styles
	evenStyle, _ := f.NewStyle(&excelize.Style{
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"E9EDF4"}, Pattern: 1},
		Alignment: &excelize.Alignment{Vertical: "top", WrapText: true},
	})
	oddStyle, _ := f.NewStyle(&excelize.Style{
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"FFFFFF"}, Pattern: 1},
		Alignment: &excelize.Alignment{Vertical: "top", WrapText: true},
	})

	// Pass/Fail specific styles
	passStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"22C55E"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})
	failStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"EF4444"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})

	// Build token lookup map
	tokenMap := make(map[string]TokenUsageData)
	for _, t := range data.TokenData {
		tokenMap[t.TestID] = t
	}

	// Helper to format dimension score
	getDimScore := func(dim testcase.EvaluationDimension) string {
		if dim.Score == 0 {
			return "-"
		}
		return fmt.Sprintf("%.1f\n%s", dim.Score, dim.Reasoning)
	}

	// Helper to format issues
	formatIssues := func(issues []testcase.CriticalIssue) string {
		var parts []string
		for _, i := range issues {
			parts = append(parts, fmt.Sprintf("[%s] %s: %s", i.Severity, i.Issue, i.Suggestion))
		}
		return strings.Join(parts, "\n\n")
	}

	// Helper to format improvements
	formatImprovements := func(imps []testcase.ImprovementRecommendation) string {
		var parts []string
		for _, i := range imps {
			parts = append(parts, fmt.Sprintf("[%s] %s -> %s", i.Priority, i.Area, i.SuggestedChange))
		}
		return strings.Join(parts, "\n\n")
	}

	// Write data rows
	for i, result := range data.Results {
		row := i + 2 // Start from row 2 (after header)

		// Passed indicator
		passedStr := "FAIL"
		if result.Passed {
			passedStr = "PASSED"
		}

		// Get token data
		tokens := tokenMap[result.TestCaseID]

		// Row data
		rowData := []interface{}{
			result.TestCaseID,
			result.Prompt,
			result.AIResponse,
			result.Expectation,
			passedStr,
			fmt.Sprintf("%.1f", result.OverallScore),
			result.OverallGrade,
			result.Summary,
			getDimScore(result.DimensionScores["accuracy"]),
			getDimScore(result.DimensionScores["completeness"]),
			getDimScore(result.DimensionScores["clarity"]),
			getDimScore(result.DimensionScores["relevance"]),
			getDimScore(result.DimensionScores["actionability"]),
			formatIssues(result.CriticalIssues),
			formatImprovements(result.Improvements),
			strings.Join(result.Exemplary, "\n"),
			fmt.Sprintf("%s\n%s", result.Confidence, result.ConfidenceReason),
			tokens.PromptTokens,
			tokens.CompletionTokens,
			tokens.TotalTokens,
		}

		// Write cells
		for j, val := range rowData {
			cell, _ := excelize.CoordinatesToCellName(j+1, row)
			f.SetCellValue(sheet, cell, val)

			// Apply alternating style
			style := oddStyle
			if i%2 == 0 {
				style = evenStyle
			}

			// SPECIAL CASE: Passed Column (index 4)
			if j == 4 {
				if result.Passed {
					style = passStyle
				} else {
					style = failStyle
				}
			}

			f.SetCellStyle(sheet, cell, cell, style)
		}
	}

	// Set column widths
	columnWidths := []float64{
		12, 40, 50, 30, // ID, Q, A, Expectation
		10, 8, 8, 40, // Passed, Score, Grade, Summary
		30, 30, 30, 30, 30, // Dimensions
		40, 40, 30, // Issues, Improvements, Exemplary
		20, 15, 18, 14, // Confidence, Tokens
	}

	for i, width := range columnWidths {
		colName, _ := excelize.CoordinatesToCellName(i+1, 1)
		colLetter := strings.TrimRight(colName, "0123456789")
		f.SetColWidth(sheet, colLetter, colLetter, width)
	}

	// Add summary row
	summaryRow := len(data.Results) + 3
	summaryStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"FFC000"}, Pattern: 1},
	})

	// Count passed/failed
	passed, failed := 0, 0
	totalPrompt, totalCompletion, totalTokens := 0, 0, 0
	for _, r := range data.Results {
		if r.Passed {
			passed++
		} else {
			failed++
		}
	}
	for _, t := range data.TokenData {
		totalPrompt += t.PromptTokens
		totalCompletion += t.CompletionTokens
		totalTokens += t.TotalTokens
	}

	// Write summary
	f.SetCellValue(sheet, fmt.Sprintf("A%d", summaryRow), "SUMMARY")
	f.SetCellValue(sheet, fmt.Sprintf("D%d", summaryRow), fmt.Sprintf("Total: %d", len(data.Results)))
	f.SetCellValue(sheet, fmt.Sprintf("E%d", summaryRow), fmt.Sprintf("PASSED:%d FAILED:%d", passed, failed))

	// Token totals (adjust columns based on new layout)
	// PromptTokens is at index 17 (col R), Completion at 18 (S), Total at 19 (T)
	f.SetCellValue(sheet, fmt.Sprintf("R%d", summaryRow), totalPrompt)
	f.SetCellValue(sheet, fmt.Sprintf("S%d", summaryRow), totalCompletion)
	f.SetCellValue(sheet, fmt.Sprintf("T%d", summaryRow), totalTokens)

	// Apply summary style (A to T)
	startCell := fmt.Sprintf("A%d", summaryRow)
	endCell := fmt.Sprintf("T%d", summaryRow)
	f.SetCellStyle(sheet, startCell, endCell, summaryStyle)

	return nil
}

func createCharts(f *excelize.File, chartsSheet string, data ExportData) error {
	// Prepare chart data in Charts sheet
	// Column A: Test IDs, Column B: Scores, Column C: Labels for pie
	f.SetCellValue(chartsSheet, "A1", "Test ID")
	f.SetCellValue(chartsSheet, "B1", "Score")

	for i, result := range data.Results {
		row := i + 2
		f.SetCellValue(chartsSheet, fmt.Sprintf("A%d", row), result.TestCaseID)
		f.SetCellValue(chartsSheet, fmt.Sprintf("B%d", row), result.OverallScore)
	}

	// Pie chart data
	passed, failed := 0, 0
	for _, r := range data.Results {
		if r.Passed {
			passed++
		} else {
			failed++
		}
	}

	pieDataRow := len(data.Results) + 4
	f.SetCellValue(chartsSheet, fmt.Sprintf("A%d", pieDataRow), "Status")
	f.SetCellValue(chartsSheet, fmt.Sprintf("B%d", pieDataRow), "Count")
	f.SetCellValue(chartsSheet, fmt.Sprintf("A%d", pieDataRow+1), "Passed")
	f.SetCellValue(chartsSheet, fmt.Sprintf("B%d", pieDataRow+1), passed)
	f.SetCellValue(chartsSheet, fmt.Sprintf("A%d", pieDataRow+2), "Failed")
	f.SetCellValue(chartsSheet, fmt.Sprintf("B%d", pieDataRow+2), failed)

	lastDataRow := len(data.Results) + 1

	// Create Bar Chart for Quality Scores
	if err := f.AddChart(chartsSheet, "D2", &excelize.Chart{
		Type: excelize.Col,
		Series: []excelize.ChartSeries{
			{
				Name:       "Score",
				Categories: fmt.Sprintf("Charts!$A$2:$A$%d", lastDataRow),
				Values:     fmt.Sprintf("Charts!$B$2:$B$%d", lastDataRow),
				Fill:       excelize.Fill{Type: "pattern", Color: []string{"4472C4"}, Pattern: 1},
			},
		},
		Title: []excelize.RichTextRun{
			{Text: "Quality Scores by Test"},
		},
		PlotArea: excelize.ChartPlotArea{
			ShowVal: true,
		},
		Dimension: excelize.ChartDimension{Width: 600, Height: 300},
	}); err != nil {
		return fmt.Errorf("failed to create bar chart: %w", err)
	}

	// Create Pie Chart for Pass/Fail ratio
	if err := f.AddChart(chartsSheet, "D20", &excelize.Chart{
		Type: excelize.Pie,
		Series: []excelize.ChartSeries{
			{
				Name:       "Pass/Fail",
				Categories: fmt.Sprintf("Charts!$A$%d:$A$%d", pieDataRow+1, pieDataRow+2),
				Values:     fmt.Sprintf("Charts!$B$%d:$B$%d", pieDataRow+1, pieDataRow+2),
			},
		},
		Title: []excelize.RichTextRun{
			{Text: "Pass / Fail Ratio"},
		},
		PlotArea: excelize.ChartPlotArea{
			ShowPercent: true,
			ShowCatName: true,
		},
		Dimension: excelize.ChartDimension{Width: 400, Height: 300},
	}); err != nil {
		return fmt.Errorf("failed to create pie chart: %w", err)
	}

	return nil
}
