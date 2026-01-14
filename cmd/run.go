package cmd

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"openllmjudge/config"
	"openllmjudge/internal/runner"
	"openllmjudge/internal/styles"
	"openllmjudge/internal/tui/run"
)

var (
	runAll    bool
	runDelay  int
	runJitter int
)

var runCmd = &cobra.Command{
	Use:   "run [judge-name...]",
	Short: "Run judge evaluations",
	Long: `Execute LLM judge evaluations on test cases.

Examples:
  # Run specific judge
  openllmjudge run my-judge

  # Run multiple judges
  openllmjudge run judge1 judge2

  # Run all judges
  openllmjudge run --all

  # Override delay/jitter
  openllmjudge run --delay 2000 --jitter 1000`,
	RunE: runRun,
}

func init() {
	runCmd.Flags().BoolVarP(&runAll, "all", "a", false, "Run all judges")
	runCmd.Flags().IntVarP(&runDelay, "delay", "d", 0, "Delay between requests in ms (overrides config)")
	runCmd.Flags().IntVarP(&runJitter, "jitter", "j", 0, "Random jitter in ms (overrides config)")
}

func runRun(cmd *cobra.Command, args []string) error {
	// Load configuration
	cfg, err := config.Load(configFile)
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		return err
	}

	if len(cfg.Judges) == 0 {
		fmt.Printf("\n%s\n", styles.WarningBoxStyle.Render("No judges configured yet. Use /create to add one."))
		fmt.Println(styles.DimStyle.Render("Press Enter to continue..."))
		var input string
		fmt.Scanln(&input)
		return nil
	}

	// Override delay/jitter if specified
	if runDelay > 0 {
		cfg.Options.DelayMs = runDelay
	}
	if runJitter > 0 {
		cfg.Options.JitterMs = runJitter
	}

	// Determine which judges to run
	var judgeNames []string

	if runAll {
		// Run all judges
		for _, j := range cfg.Judges {
			judgeNames = append(judgeNames, j.Name)
		}
	} else if len(args) > 0 {
		// Run specified judges from args
		judgeNames = args

		// Validate judge names
		for _, name := range judgeNames {
			found := false
			for _, j := range cfg.Judges {
				if j.Name == name {
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("judge not found: %s", name)
			}
		}
	}

	// Create executor
	executor := runner.NewExecutor(cfg)

	// Create and run TUI
	// If judgeNames is empty, model starts in selection step.
	// If not empty, it auto-starts in StepRunning.
	model := run.NewRunModel(cfg, executor, judgeNames)

	p := tea.NewProgram(model, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		return fmt.Errorf("error running TUI: %v", err)
	}

	return nil // TUI finished
}
