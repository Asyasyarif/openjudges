package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// APIKeys stores centralized API keys for all LLM providers
type APIKeys struct {
	OpenAI     string            `json:"openai,omitempty"`
	Anthropic  string            `json:"anthropic,omitempty"`
	Google     string            `json:"google,omitempty"`
	Groq       string            `json:"groq,omitempty"`
	OpenRouter string            `json:"openrouter,omitempty"`
	Custom     map[string]string `json:"custom,omitempty"`
}

// APIKeysConfig wraps APIKeys for config file structure
type APIKeysConfig struct {
	Version string  `json:"version"`
	Keys    APIKeys `json:"keys"`
}

// LoadAPIKeys loads API keys from centralized storage.
// Searches in order: .apikeys.json (local), ~/.openllmjudge/apikeys.json (user home),
// returns empty struct if not found.
func LoadAPIKeys() (*APIKeys, error) {
	localPath := ".apikeys.json"
	if _, err := os.Stat(localPath); err == nil {
		return loadAPIKeysFromFile(localPath)
	}

	homeDir, err := os.UserHomeDir()
	if err == nil {
		configPath := filepath.Join(homeDir, ".openllmjudge", "apikeys.json")
		if _, err := os.Stat(configPath); err == nil {
			return loadAPIKeysFromFile(configPath)
		}
	}

	return &APIKeys{}, nil
}

func loadAPIKeysFromFile(path string) (*APIKeys, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var cfg APIKeysConfig
	if err := json.NewDecoder(file).Decode(&cfg); err != nil {
		return nil, err
	}

	return &cfg.Keys, nil
}

// SaveAPIKeys saves API keys to specified location with secure file permissions (0600)
func SaveAPIKeys(keys *APIKeys, path string) error {
	cfg := APIKeysConfig{
		Version: "1.0",
		Keys:    *keys,
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0600)
}

// GetAPIKey retrieves API key for provider with fallback:
// 1. Local stored key, 2. Environment variable, 3. Empty string
func (k *APIKeys) GetAPIKey(provider string) string {
	var storedKey string

	switch provider {
	case "openai":
		storedKey = k.OpenAI
	case "anthropic":
		storedKey = k.Anthropic
	case "google":
		storedKey = k.Google
	case "groq":
		storedKey = k.Groq
	case "openrouter":
		storedKey = k.OpenRouter
	default:
		if k.Custom != nil {
			storedKey = k.Custom[provider]
		}
	}

	if storedKey == "" {
		envKey := getEnvVarName(provider)
		storedKey = os.Getenv(envKey)
	}

	return storedKey
}

func getEnvVarName(provider string) string {
	switch provider {
	case "openai":
		return "OPENAI_API_KEY"
	case "anthropic":
		return "ANTHROPIC_API_KEY"
	case "google":
		return "GOOGLE_API_KEY"
	case "groq":
		return "GROQ_API_KEY"
	case "openrouter":
		return "OPENROUTER_API_KEY"
	default:
		return provider + "_API_KEY"
	}
}
