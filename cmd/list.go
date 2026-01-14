package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"openjudges/config"
	"openjudges/internal/tui/create"
	"openjudges/internal/tui/list"
)

var (
	listVerbose bool
	listFormat  string
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all configured judges",
	Long:  `Display a list of all configured LLM judges with their details.`,
	RunE:  runList,
}

func init() {
	listCmd.Flags().BoolVarP(&listVerbose, "verbose", "v", false, "Show detailed information")
	listCmd.Flags().StringVarP(&listFormat, "format", "f", "table", "Output format (table, json)")
}

func runList(cmd *cobra.Command, args []string) error {
	// Load configuration
	cfg, err := config.Load(configFile)
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		return err
	}

	// Output based on format
	switch listFormat {
	case "json":
		return outputJSON(cfg.Judges)
	case "table":
		// Launch TUI
		m := list.NewModel(cfg, configFile)
		p := tea.NewProgram(m, tea.WithAltScreen())
		tm, err := p.Run()
		if err != nil {
			return fmt.Errorf("error running list TUI: %v", err)
		}

		// Check if we need to edit
		if listModel, ok := tm.(list.Model); ok {
			if editConfig := listModel.GetEditResult(); editConfig != nil {
				// Launch create wizard in edit mode
				createModel := create.NewModel(editConfig)
				pEdit := tea.NewProgram(createModel)
				if _, err := pEdit.Run(); err != nil {
					return fmt.Errorf("error running edit wizard: %v", err)
				}
			}
		}
		return nil
	default:
		fmt.Printf("Unknown format: %s\n", listFormat)
		return fmt.Errorf("unknown format: %s", listFormat)
	}
}

func outputJSON(judges []config.JudgeConfig) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(judges)
}
