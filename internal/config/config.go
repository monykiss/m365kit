// Package config manages application configuration from files and environment.
package config

import (
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// Config holds the application configuration.
type Config struct {
	Provider string `mapstructure:"provider"`
	Model    string `mapstructure:"model"`
	APIKeys  struct {
		Anthropic string `mapstructure:"anthropic"`
		OpenAI    string `mapstructure:"openai"`
	} `mapstructure:"api_keys"`
	Ollama struct {
		Host string `mapstructure:"host"`
	} `mapstructure:"ollama"`
	Output struct {
		Format string `mapstructure:"format"`
		Color  bool   `mapstructure:"color"`
	} `mapstructure:"output"`
}

// Load reads the configuration from ~/.kit/config.yaml and environment variables.
func Load() (*Config, error) {
	configDir := configDir()

	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(configDir)

	// Defaults
	viper.SetDefault("provider", "anthropic")
	viper.SetDefault("model", "claude-sonnet-4-20250514")
	viper.SetDefault("output.color", true)
	viper.SetDefault("output.format", "text")

	// Environment variable overrides
	viper.SetEnvPrefix("KIT")
	viper.AutomaticEnv()

	// Read config file (non-fatal if missing)
	_ = viper.ReadInConfig()

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func configDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".kit"
	}
	return filepath.Join(home, ".kit")
}
