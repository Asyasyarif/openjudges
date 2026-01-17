package cmd

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"github.com/xuri/excelize/v2"

	"openjudges/config"
	"openjudges/internal/styles"
	"openjudges/internal/tui/root"
	"openjudges/internal/utils"
)

// clearScreen clears the terminal screen
func clearScreen() {
	fmt.Print("\033[2J\033[H")
}

var (
	configFile  string
	showVersion bool
)

// rootCmd represents the base command
var rootCmd = &cobra.Command{
	Use:   "openjudges",
	Short: "LLM as a Judge - Evaluate AI responses",
	Long:  `OpenJudges uses Large Language Models to evaluate AI responses against criteria.`,
	Run: func(cmd *cobra.Command, args []string) {
		if showVersion {
			fmt.Printf("openjudges %s\n", version)
			return
		}
		InitializeProject(true)
		showInteractiveMenu()
	},
}

// Execute runs the root command
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().StringVar(&configFile, "config", "config.json", "Config file path")
	rootCmd.Flags().BoolVar(&showVersion, "version", false, "Show version and exit")

	// Add subcommands
	rootCmd.AddCommand(createCmd)
	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(resultsCmd)
	rootCmd.AddCommand(menuCmd)
	rootCmd.AddCommand(autopromptCmd)

	// Move directory/file creation out of init() for reliability
}

// InitializeProject ensures all required directories and files exist
func InitializeProject(verbose bool) {
	// Ensure required directories exist
	if verbose {
		fmt.Printf("%s\n", styles.InfoBoxStyle.Render("ℹ Initializing project structure..."))
		fmt.Printf("%s\n", styles.InfoBoxStyle.Render("📁 Creating required directories..."))
	}
	os.MkdirAll("datasets", 0755)
	os.MkdirAll("results", 0755)
	os.MkdirAll("prompts", 0755)
	os.MkdirAll("vendors", 0755)
	if verbose {
		fmt.Printf("  ✓ datasets/ - for test data and master files\n")
		fmt.Printf("  ✓ results/ - for evaluation results\n")
		fmt.Printf("  ✓ prompts/ - for judge prompt templates\n")
		fmt.Printf("  ✓ vendors/ - for custom LLM vendor configurations\n\n")
	}

	// Create default config if it doesn't exist
	if verbose {
		fmt.Printf("%sn", styles.InfoBoxStyle.Render("📄 Creating default configuration..."))
	}
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		defaultCfg := &config.Config{
			Judges: []config.JudgeConfig{},
			Options: config.Options{
				DelayMs:  1000,
				JitterMs: 500,
			},
		}
		config.SaveConfig(defaultCfg, configFile)
		if verbose {
			fmt.Printf("  ✓ %s - main configuration file\n\n", configFile)
		}
	}

	// Sample prompt creation moved to standard location

	// Create sample prompt if it doesn't exist
	if verbose {
		fmt.Printf("%sn", styles.InfoBoxStyle.Render("📝 Creating sample judge prompt..."))
	}
	if _, err := os.Stat("prompts/judges_prompt_dont_change.md"); os.IsNotExist(err) {
		samplePrompt := `# Expert AI Response Evaluator

You are an expert evaluator specializing in assessing AI-generated responses. Your evaluations must be:
- **Objective**: Based on observable evidence, not assumptions
- **Detailed**: Provide specific examples from the response
- **Constructive**: Offer actionable feedback for improvement
- **Consistent**: Apply criteria uniformly across all evaluations

## Evaluation Context

**Original Question**: 
{{prompt}}

**AI Response to Evaluate**: 
{{ai_response}}

**Expected Quality Benchmarks**: 
{{expectation}}

**Specific Criteria**: 
{{criteria}}

---

## Evaluation Dimensions

Evaluate the response across these dimensions:

### 1. Accuracy & Correctness
- Are all factual claims correct?
- Are there any misleading or incorrect statements?

### 2. Completeness
- Does it address all aspects of the question?
- Are important points missing?

### 3. Clarity & Structure
- Is the response well-organized and easy to follow?
- Are explanations clear and unambiguous?

### 4. Relevance
- Does the response stay on topic?
- Is all included information pertinent?

### 5. Actionability
- Can the user apply this information?
- Are practical steps or examples provided?

---

## Output Requirements

Provide your evaluation in the following JSON structure:

{
  "overall_score": <0-100 float, VERY IMPORTANT: scale is 0 to 100>,
  "overall_grade": "<A/B/C/D/F>",
  "summary": "<2-3 sentence overall assessment>",
  
  "dimension_scores": {
    "accuracy": {
      "score": <0-100 float>,
      "weight": 0.30,
      "weighted_score": <calculated>,
      "evidence": ["<quote>"],
      "strengths": ["<text>"],
      "weaknesses": ["<text>"],
      "reasoning": "<text>"
    },
    "completeness": {
      "score": <0-100 float>,
      "weight": 0.25,
      "weighted_score": <calculated>,
      "evidence": [],
      "strengths": [],
      "weaknesses": [],
      "reasoning": "<text>"
    },
    "clarity": {
      "score": <0-100 float>,
      "weight": 0.20,
      "weighted_score": <calculated>,
      "evidence": [],
      "strengths": [],
      "weaknesses": [],
      "reasoning": "<text>"
    },
    "relevance": {
      "score": <0-100 float>,
      "weight": 0.15,
      "weighted_score": <calculated>,
      "evidence": [],
      "strengths": [],
      "weaknesses": [],
      "reasoning": "<text>"
    },
    "actionability": {
      "score": <0-100 float>,
      "weight": 0.10,
      "weighted_score": <calculated>,
      "evidence": [],
      "strengths": [],
      "weaknesses": [],
      "reasoning": "<text>"
    }
  },
  
  "critical_issues": [
    {
      "severity": "<high/medium/low>",
      "issue": "<text>",
      "location": "<text>",
      "impact": "<text>",
      "suggestion": "<text>"
    }
  ],
  
  "improvement_recommendations": [
    {
      "priority": "<high/medium/low>",
      "area": "<dimension>",
      "current_state": "<text>",
      "suggested_change": "<text>",
      "expected_impact": "<text>"
    }
  ],
  
  "exemplary_elements": ["<text>"],
  
  "confidence_level": "<high/medium/low>",
  "confidence_reasoning": "<text>",
  
  "comparison_to_expectation": {
    "alignment_score": <0-100>,
    "gaps": ["<text>"],
    "exceeds": ["<text>"]
  }
}

IMPORTANT: The "overall_score" MUST be between 0 and 100. If the quality is 9/10, output 90.0.
`
		os.WriteFile("prompts/judges_prompt_dont_change.md", []byte(samplePrompt), 0644)
		if verbose {
			fmt.Printf("  ✓ prompts/judges_prompt_dont_change.md - default evaluation promptnn")
		}
	}
	if verbose {
		fmt.Printf("%sn", styles.InfoBoxStyle.Render("🔌 Creating sample vendor configurations..."))
	}

	// Create sample vendor files if they don't exist
	if _, err := os.Stat("vendors/example-vendor-get.json"); os.IsNotExist(err) {
		sampleGet := `{
  "name": "example-vendor-get",
  "url": "https://example.com/api?q={{question}}",
  "method": "GET",
  "headers": {
    "Authorization": "Bearer {{api_key}}",
    "Accept": "text/event-stream"
  },
  "body": {},
  "parse_as": "",
  "guardrail": []
}`
		os.WriteFile("vendors/example-vendor-get.json", []byte(sampleGet), 0644)
		if verbose {
			fmt.Printf("  ✓ vendors/example-vendor-get.json - GET request examplen")
		}
	}

	if _, err := os.Stat("vendors/example-vendor-post.json"); os.IsNotExist(err) {
		samplePost := `{
  "name": "example-vendor-post",
  "url": "https://api.example.com/v1/chat",
  "method": "POST",
  "headers": {
    "Authorization": "Bearer {{api_key}}",
    "Content-Type": "application/json"
  },
  "body": {
    "model": "gpt-4",
    "messages": [
      {"role": "user", "content": "{{prompt}}"}
    ],
    "temperature": 0.7
  },
  "parse_as": "choices.0.delta.content",
  "guardrail": []
}`
		os.WriteFile("vendors/example-vendor-post.json", []byte(samplePost), 0644)
		if verbose {
			fmt.Printf("  ✓ vendors/example-vendor-post.json - POST request example\n")
		}
	}

	// Create sample master_data.xlsx if it doesn't exist
	if _, err := os.Stat("datasets/master_data.xlsx"); os.IsNotExist(err) {
		if verbose {
			fmt.Printf("%sn", styles.InfoBoxStyle.Render("📊 Creating sample test data..."))
		}
		if verbose {
			fmt.Printf("  ✓ datasets/master_data.xlsx - sample evaluation datasetnn")
		}
		createSampleMasterData()
		if verbose {
			fmt.Printf("%sn", styles.SuccessBoxStyle.Render("✓ Project initialization complete!"))
		}
	}
}

// menuCmd is an explicit command for showing the menu
var menuCmd = &cobra.Command{
	Use:   "menu",
	Short: "Show interactive menu",
	Run: func(cmd *cobra.Command, args []string) {
		showInteractiveMenu()
	},
}

func showInteractiveMenu() {
	p := tea.NewProgram(root.NewModel(), tea.WithAltScreen())
	m, err := p.Run()
	if err != nil {
		fmt.Printf("Error running program: %v\n", err)
		os.Exit(1)
	}

	if model, ok := m.(root.Model); ok {
		if model.ExecutedCmd != "" {
			// Handle command
			cmd := model.ExecutedCmd
			args := strings.Fields(cmd)

			if len(args) > 0 {
				command := args[0]
				remainingArgs := args[1:]
				HandleSlashCommand(command, remainingArgs)
			}
		}
	}
}

// HandleSlashCommand processes slash commands from TUI or args
func HandleSlashCommand(command string, args []string) {
	clearScreen()
	switch strings.ToLower(command) {
	case "/create":
		if err := createCmd.RunE(createCmd, args); err != nil {
			fmt.Printf("Error: %v\n", err)
		}
		showInteractiveMenu()

	case "/list":
		if err := listCmd.RunE(listCmd, args); err != nil {
			fmt.Printf("Error: %v\n", err)
		}
		showInteractiveMenu()

	case "/start", "/run":
		if err := runCmd.RunE(runCmd, args); err != nil {
			fmt.Printf("\n%s\n", styles.ErrorBoxStyle.Render("Error: "+err.Error()))
			fmt.Println(styles.DimStyle.Render("Press Enter to continue..."))
			var input string
			fmt.Scanln(&input)
		}
		showInteractiveMenu()

	case "/result", "/results":
		if err := resultsCmd.RunE(resultsCmd, args); err != nil {
			fmt.Printf("Error: %v\n", err)
		}
		showInteractiveMenu()

	case "/vendor":
		if err := vendorCmd.RunE(vendorCmd, args); err != nil {
			fmt.Printf("Error: %v\n", err)
		}
		showInteractiveMenu()

	case "/auto-prompt":
		if err := autopromptCmd.RunE(autopromptCmd, args); err != nil {
			fmt.Printf("Error: %v\n", err)
		}
		showInteractiveMenu()

	case "/config":
		// Simplified config view
		fmt.Printf("Configuration file: %s\n", utils.DisplayPath(configFile))
		showInteractiveMenu()

	case "/menu", "/back":
		showInteractiveMenu()

	case "/exit", "/quit":
		fmt.Println("Goodbye!")
		os.Exit(0)

	default:
		fmt.Printf("Unknown command: %s\n", command)
		showInteractiveMenu()
	}
}

// createSampleMasterData creates a sample master_data.xlsx file
func createSampleMasterData() {
	f := excelize.NewFile()
	defer f.Close()

	// Set sheet name
	sheet := "Sheet1"
	f.SetSheetName("Sheet1", sheet)

	// Headers
	headers := []string{"id", "question", "expectation", "criteria"}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheet, cell, h)
	}

	// Sample Data
	data := [][]interface{}{
		{"1", "Say hello", "AI should greet politely", "Should contain greeting;Should be polite"},
		{"2", "What is 2+2?", "Correct mathematical result", ""},
		{"3", "Tell me a joke about cats", "AI should talk about dogs instead", "Should contain the word 'dog';Should not mention cats"},
	}

	for r, row := range data {
		for c, val := range row {
			cell, _ := excelize.CoordinatesToCellName(c+1, r+2)
			f.SetCellValue(sheet, cell, val)
		}
	}

	// Styling headers
	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Color: "FFFFFF"},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"4472C4"}, Pattern: 1},
	})
	f.SetCellStyle(sheet, "A1", "F1", headerStyle)
	f.SetColWidth(sheet, "C", "E", 30)

	if err := f.SaveAs("datasets/master_data.xlsx"); err != nil {
		fmt.Printf("Warning: Failed to create master_data.xlsx: %v\n", err)
	} else {
		fmt.Println("Sample master_data.xlsx created at datasets/master_data.xlsx")
	}
}
