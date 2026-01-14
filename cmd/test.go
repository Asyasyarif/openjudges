package cmd

import (
	"fmt"

	"openllmjudge/internal/tui/test"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

var testCmd = &cobra.Command{
	Use:   "test",
	Short: "Run dummy LLM streaming simulation",
	Long:  "Run a dummy LLM streaming simulation to test TUI components. Shows progress, timer, and streaming output.",
	RunE: func(cmd *cobra.Command, args []string) error {
		p := tea.NewProgram(test.NewTestModel(), tea.WithAltScreen())
		if _, err := p.Run(); err != nil {
			return fmt.Errorf("error running test TUI: %v", err)
		}
		return nil
	},
}

// init() removed - test command is hidden from the menu and slash commands
