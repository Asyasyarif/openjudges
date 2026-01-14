package config

import (
	"encoding/json"
	"os"
)

// Config represents the application configuration
type Config struct {
	Judges          []JudgeConfig      `json:"judges"`
	CustomProviders []LLMConfig        `json:"custom_providers,omitempty"`
	Options         Options            `json:"options,omitempty"`
	AutoPrompts     []AutoPromptConfig `json:"auto_prompts,omitempty"`
}

// AutoPromptConfig contains configuration for autonomous prompt engineer
type AutoPromptConfig struct {
	Name               string  `json:"name"`                 // Unique name for this config
	GetPromptVendor    string  `json:"get_prompt_vendor"`    // Vendor for downloading prompt
	UpdatePromptVendor string  `json:"update_prompt_vendor"` // Vendor for uploading prompt
	ChatVendor         string  `json:"chat_vendor"`          // Vendor for agent chat/analysis
	MaxIterations      int     `json:"max_iterations"`       // Max iterations (default 5)
	MinPassScore       float64 `json:"min_pass_score"`       // Minimum pass score (default 70)
}

// Options contains global options
type Options struct {
	DelayMs  int `json:"delay_ms"`  // Delay between iterations in ms
	JitterMs int `json:"jitter_ms"` // Random jitter to add (0 to jitter_ms)
	Parallel int `json:"parallel"`  // Max parallel judges (0 = unlimited)
}

// JudgeConfig contains configuration for a single judge
type JudgeConfig struct {
	Name         string            `json:"name"`
	LLM          LLMConfig         `json:"llm"`
	DataPath     string            `json:"data_path"`
	ResultPath   string            `json:"result_path"`
	PromptFile   string            `json:"prompt_file,omitempty"`
	PromptValues map[string]string `json:"prompt_values,omitempty"`
}

// LLMConfig contains LLM API configuration
type LLMConfig struct {
	Provider     string            `json:"provider"`
	APIKey       string            `json:"api_key"`
	APIURL       string            `json:"api_url"`
	Model        string            `json:"model"`
	Method       string            `json:"method,omitempty"`        // GET, POST, etc.
	Headers      map[string]string `json:"headers,omitempty"`       // Custom headers
	ResponsePath string            `json:"response_path,omitempty"` // JSONPath-like selector
	BodyTemplate string            `json:"body_template,omitempty"`
}

// DefaultLLMAPIURLs contains default API endpoints for providers
var DefaultLLMAPIURLs = map[string]string{
	"openai":     "https://api.openai.com/v1/chat/completions",
	"anthropic":  "https://api.anthropic.com/v1/messages",
	"google":     "https://generativelanguage.googleapis.com/v1beta/models",
	"groq":       "https://api.groq.com/openai/v1/chat/completions",
	"openrouter": "https://openrouter.ai/api/v1/chat/completions",
}

func (c *Config) UpdateJudge(originalName string, updated JudgeConfig) {
	for i, j := range c.Judges {
		if j.Name == originalName {
			c.Judges[i] = updated
			return
		}
	}
	c.Judges = append(c.Judges, updated)
}

// UpdateCustomProvider updates or appends a custom provider
func (c *Config) UpdateCustomProvider(originalName string, updated LLMConfig) {
	for i, p := range c.CustomProviders {
		if p.Provider == originalName {
			c.CustomProviders[i] = updated
			return
		}
	}
	c.CustomProviders = append(c.CustomProviders, updated)
}

// Load reads configuration from a JSON file and resolves API keys
func Load(path string) (*Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var cfg Config
	if err := json.NewDecoder(file).Decode(&cfg); err != nil {
		return nil, err
	}

	apiKeys, err := LoadAPIKeys()
	if err != nil {
		apiKeys = &APIKeys{}
	}

	for i := range cfg.Judges {
		cfg.Judges[i].LLM.APIKey = resolveAPIKey(
			cfg.Judges[i].LLM.APIKey,
			cfg.Judges[i].LLM.Provider,
			apiKeys,
		)
	}

	return &cfg, nil
}

// SaveConfig saves configuration to a JSON file
func SaveConfig(cfg *Config, path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(cfg)
}

// resolveEnvVar resolves ${ENV_VAR} syntax to actual env value
func resolveEnvVar(value string) string {
	if len(value) > 3 && value[:2] == "${" && value[len(value)-1] == '}' {
		envName := value[2 : len(value)-1]
		return os.Getenv(envName)
	}
	return value
}

func resolveAPIKey(configKey, provider string, apiKeys *APIKeys) string {
	if len(configKey) > 3 && configKey[:2] == "${" && configKey[len(configKey)-1] == '}' {
		envName := configKey[2 : len(configKey)-1]
		return os.Getenv(envName)
	}

	if configKey != "" {
		return configKey
	}

	if apiKeys != nil {
		if key := apiKeys.GetAPIKey(provider); key != "" {
			return key
		}
	}

	envKey := getEnvVarName(provider)
	return os.Getenv(envKey)
}
