package cmd

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"openjudges/config"
	"openjudges/internal/tui/vendor"
)

var vendorCmd = &cobra.Command{
	Use:   "vendor",
	Short: "Manage custom LLM vendors",
	Long:  `Manage custom LLM API providers by creating new ones or listing/editing existing ones.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runVendor(cmd, args)
	},
}

func init() {
	rootCmd.AddCommand(vendorCmd)
}

func runVendor(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load(configFile)
	if err != nil {
		// If config doesn't exist, create an empty one
		cfg = &config.Config{}
	}

	initialStep := vendor.StepMenu
	if len(args) > 0 {
		switch args[0] {
		case "create":
			initialStep = vendor.StepCreateWizard
		case "list":
			initialStep = vendor.StepList
		}
	}

	p := tea.NewProgram(vendor.NewModel(cfg, initialStep), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running program: %v\n", err)
		os.Exit(1)
	}

	return nil
}
