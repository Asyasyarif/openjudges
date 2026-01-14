package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"openjudges/internal/tui/results"
)

var (
	resultsList   bool
	resultsFormat string
)

var resultsCmd = &cobra.Command{
	Use:   "results [file]",
	Short: "View and analyze evaluation results",
	Long: `View detailed results from judge evaluations.

If no file is specified, opens an interactive file browser (TUI).`,
	RunE: runResults,
}

func init() {
	resultsCmd.Flags().BoolVarP(&resultsList, "list", "l", false, "List available result files")
	resultsCmd.Flags().StringVarP(&resultsFormat, "format", "f", "table", "Output format (table, json)")
}

func runResults(cmd *cobra.Command, args []string) error {
	// Find result files
	resultFiles, err := findResultFiles()
	if err != nil {
		return err
	}

	/*
		if len(resultFiles) == 0 {
			fmt.Println("No result files found in 'results/' directory.")
			return nil
		}
	*/

	// If --list flag is set, just list the files
	if resultsList {
		return listResultFiles(resultFiles)
	}

	// Use TUI for table format (interactive)
	if resultsFormat == "table" {
		initialFile := ""
		if len(args) > 0 {
			initialFile = args[0]
			// Validate existence if provided
			if _, err := os.Stat(initialFile); os.IsNotExist(err) {
				return fmt.Errorf("result file not found: %s", initialFile)
			}
		}

		m := results.NewModel(initialFile)
		p := tea.NewProgram(m, tea.WithAltScreen())
		if _, err := p.Run(); err != nil {
			return fmt.Errorf("error running results TUI: %v", err)
		}
		return nil
	}

	// JSON output flow
	var resultFile string
	if len(args) > 0 {
		resultFile = args[0]
		if _, err := os.Stat(resultFile); os.IsNotExist(err) {
			return fmt.Errorf("result file not found: %s", resultFile)
		}
	} else {
		return fmt.Errorf("result file argument required for JSON format")
	}

	// Load and display the result
	return displayResult(resultFile)
}

func findResultFiles() ([]string, error) {
	var files []string

	// Create results directory if not exists
	os.MkdirAll("results", 0755)

	err := filepath.Walk("results", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

func listResultFiles(files []string) error {
	fmt.Println("Available result files:")
	for _, f := range files {
		info, _ := os.Stat(f)
		size := "0B"
		timeStr := ""
		if info != nil {
			size = fmt.Sprintf("%d bytes", info.Size())
			timeStr = info.ModTime().Format(time.RFC822)
		}
		fmt.Printf("  %s (%s, %s)\n", f, size, timeStr)
	}
	return nil
}

func displayResult(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read result file: %w", err)
	}

	if resultsFormat == "json" {
		fmt.Println(string(data))
		return nil
	}
	return nil
}
