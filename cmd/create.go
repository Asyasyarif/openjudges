package cmd

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"openllmjudge/config"
	"openllmjudge/internal/tui/create"
)

// LLM Provider and Model configurations
var (
	llmProviders = []string{"openai", "anthropic", "google", "groq", "openrouter"}

	llmModels = map[string][]string{
		"openai":     {"gpt-5.2", "gpt-5.2-thinking", "gpt-5.1", "gpt-5", "gpt-4o", "gpt-4o-mini", "gpt-4-turbo", "o1", "o1-mini"},
		"anthropic":  {"claude-sonnet-4-5-20250929", "claude-opus-4-1-20250101", "claude-sonnet-4-20250514", "claude-3-5-sonnet-20241022", "claude-3-5-haiku-20241022"},
		"google":     {"gemini-3-flash-exp", "gemini-3-pro", "gemini-3-flash", "gemini-2.5-pro", "gemini-2.5-flash", "gemini-2.0-flash", "gemini-1.5-pro", "gemini-1.5-flash"},
		"groq":       {"llama-3.3-70b-versatile", "llama-3.1-8b-instant", "llama-3.3-70b-specdec", "mixtral-8x7b-32768", "deepseek-r1-distill-llama-70b"},
		"openrouter": {"anthropic/claude-sonnet-4-5-20250929", "openai/gpt-5.2", "google/gemini-3-pro", "deepseek/deepseek-chat"},
	}
)

var (
	createName     string
	createProvider string
	createModel    string
	createAPIKey   string
	createDataset  string
	createResult   string
)

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new judge configuration",
	Long: `Create a new LLM judge configuration interactively or with flags.

Examples:
  # Interactive mode
  openllmjudge create

  # Non-interactive mode (all flags)
  openllmjudge create --name my-judge --provider openai --model gpt-5.2 \
    --dataset datasets/testcases.csv \
    --result results/my-judge.json

  # Partial flags (prompts for missing)
  openllmjudge create --name my-judge --provider anthropic`,
	RunE: runCreate,
}

func init() {
	createCmd.Flags().StringVar(&createName, "name", "", "Judge name")
	createCmd.Flags().StringVar(&createProvider, "provider", "", "LLM provider (openai, anthropic, google, groq, openrouter)")
	createCmd.Flags().StringVar(&createModel, "model", "", "LLM model")
	createCmd.Flags().StringVar(&createAPIKey, "api-key", "", "API key (or ${ENV_VAR})")
	createCmd.Flags().StringVar(&createDataset, "dataset", "", "Dataset file path")
	createCmd.Flags().StringVar(&createResult, "result", "", "Result file path")
}

func runCreate(cmd *cobra.Command, args []string) error {
	// Load existing configuration
	cfg, err := config.Load(configFile)
	if err != nil {
		// If config doesn't exist, create a new one
		cfg = &config.Config{
			Judges: []config.JudgeConfig{},
			Options: config.Options{
				DelayMs:  1000,
				JitterMs: 500,
			},
		}
	}

	// Check if all flags are provided (non-interactive mode)
	if allFlagsProvided() {
		return createJudgeNonInteractive(cfg)
	}

	// Interactive mode
	return createJudgeInteractive(cfg)
}

func allFlagsProvided() bool {
	return createName != "" &&
		createProvider != "" &&
		createModel != "" &&
		createAPIKey != "" &&
		createDataset != "" &&
		createResult != ""
}

func createJudgeInteractive(cfg *config.Config) error {
	p := tea.NewProgram(create.NewModel(nil))
	_, err := p.Run()
	if err != nil {
		return fmt.Errorf("error running create wizard: %v", err)
	}
	return nil
}

func createJudgeNonInteractive(cfg *config.Config) error {
	// Validate provider
	if !contains(llmProviders, createProvider) {
		return fmt.Errorf("invalid provider: %s (valid: %s)", createProvider, strings.Join(llmProviders, ", "))
	}

	// Validate model
	models := llmModels[createProvider]
	if !contains(models, createModel) {
		return fmt.Errorf("invalid model: %s for provider %s (valid: %s)", createModel, createProvider, strings.Join(models, ", "))
	}

	// Check if judge already exists
	for i, j := range cfg.Judges {
		if j.Name == createName {
			fmt.Printf("Warning: Overwriting existing judge: %s\n", createName)
			cfg.Judges = append(cfg.Judges[:i], cfg.Judges[i+1:]...)
			break
		}
	}

	// Create the judge config
	newJudge := config.JudgeConfig{
		Name: createName,
		LLM: config.LLMConfig{
			Provider: createProvider,
			APIKey:   createAPIKey,
			APIURL:   config.DefaultLLMAPIURLs[createProvider],
			Model:    createModel,
		},
		DataPath:   createDataset,
		ResultPath: createResult,
		PromptFile: "prompts/judges_prompt_dont_change.md",
	}

	// Add to config
	cfg.Judges = append(cfg.Judges, newJudge)

	// Save config
	if err := saveConfig(cfg); err != nil {
		return err
	}

	fmt.Printf("Judge '%s' created successfully!\n", createName)
	fmt.Printf("Provider: %s\nModel: %s\nDataset: %s\nResult: %s\n", newJudge.LLM.Provider, newJudge.LLM.Model, newJudge.DataPath, newJudge.ResultPath)

	return nil
}

func saveConfig(cfg *config.Config) error {
	// Ensure directories exist
	os.MkdirAll("datasets", 0755)
	os.MkdirAll("results", 0755)

	return config.SaveConfig(cfg, configFile)
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
