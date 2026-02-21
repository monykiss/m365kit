package config

import (
	"fmt"
	"os"
)

// GetAPIKey retrieves the API key for the given provider, checking environment
// variables first and falling back to the config file.
func GetAPIKey(provider string) (string, error) {
	switch provider {
	case "anthropic":
		if key := os.Getenv("ANTHROPIC_API_KEY"); key != "" {
			return key, nil
		}
		cfg, err := Load()
		if err == nil && cfg.APIKeys.Anthropic != "" {
			return cfg.APIKeys.Anthropic, nil
		}
		return "", fmt.Errorf("ANTHROPIC_API_KEY not found — set it via environment variable or in ~/.kit/config.yaml")

	case "openai":
		if key := os.Getenv("OPENAI_API_KEY"); key != "" {
			return key, nil
		}
		cfg, err := Load()
		if err == nil && cfg.APIKeys.OpenAI != "" {
			return cfg.APIKeys.OpenAI, nil
		}
		return "", fmt.Errorf("OPENAI_API_KEY not found — set it via environment variable or in ~/.kit/config.yaml")

	default:
		return "", fmt.Errorf("no API key management for provider %q", provider)
	}
}
