package cmd

import (
	"fmt"

	"openllmjudge/config"
	"openllmjudge/internal/tui/autoprompt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

var autopromptCmd = &cobra.Command{
	Use:   "auto-prompt",
	Short: "Autonomous Prompt Engineer - Automatically test and improve prompts",
	Long: `Autonomous Prompt Engineer that iteratively:
1. Downloads prompt from API (GET vendor)
2. Runs test cases with selected judge
3. If tests fail, analyzes and improves the prompt
4. Uploads improved prompt to API (UPDATE vendor)
5. Repeats until all tests pass or max iterations reached

Examples:
  openllmjudge auto-prompt                        # Interactive TUI mode
  openllmjudge auto-prompt --judge=gpt-4 --config=production  # Direct run`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load(configPath)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		judgeName, _ := cmd.Flags().GetString("judge")
		configName, _ := cmd.Flags().GetString("config")

		if judgeName == "" && configName == "" {
			// Interactive mode - launch TUI
			p := tea.NewProgram(autoprompt.NewInitialModel(cfg), tea.WithAltScreen())
			if _, err := p.Run(); err != nil {
				return fmt.Errorf("error running auto-prompt TUI: %w", err)
			}
			return nil
		}

		// Direct mode - run with specified judge and config
		if judgeName == "" {
			return fmt.Errorf("judge is required when running in direct mode")
		}
		if configName == "" {
			return fmt.Errorf("auto-prompt config is required when running in direct mode")
		}

		// Find judge and auto-prompt config
		foundJudge := false
		foundConfig := false

		for _, j := range cfg.Judges {
			if j.Name == judgeName {
				foundJudge = true
				break
			}
		}

		for _, ap := range cfg.AutoPrompts {
			if ap.Name == configName {
				foundConfig = true
				break
			}
		}

		if !foundJudge {
			return fmt.Errorf("judge '%s' not found in config", judgeName)
		}
		if !foundConfig {
			return fmt.Errorf("auto-prompt config '%s' not found in config", configName)
		}

		// TODO: Implement direct run mode
		fmt.Printf("Direct run mode - Judge: %s, Config: %s\n", judgeName, configName)
		fmt.Println("This feature will be implemented in a future update.")
		return nil
	},
}

var (
	judgeName  string
	configName string
	configPath string
)

func init() {
	rootCmd.AddCommand(autopromptCmd)

	autopromptCmd.Flags().StringVarP(&judgeName, "judge", "j", "", "Judge name to use")
	autopromptCmd.Flags().StringVarP(&configName, "config", "c", "", "Auto-prompt config name to use")
	autopromptCmd.Flags().StringVarP(&configPath, "config-path", "f", "config.json", "Path to config file")
}
