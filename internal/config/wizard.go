package config

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

// ConfigIssue represents a validation finding.
type ConfigIssue struct {
	Key      string `json:"key"`
	Severity string `json:"severity"` // "error", "warning", "info"
	Message  string `json:"message"`
	Fix      string `json:"fix"`
}

// Wizard runs the interactive setup wizard.
// If reader is nil, reads from os.Stdin.
func Wizard(reader io.Reader) error {
	if reader == nil {
		reader = os.Stdin
	}
	scanner := bufio.NewScanner(reader)

	fmt.Println("M365Kit Setup Wizard")
	fmt.Println()
	fmt.Println("Let's get you set up in about 60 seconds.")
	fmt.Println()
	fmt.Println(strings.Repeat("-", 48))
	fmt.Println()

	// Step 1: AI Provider
	fmt.Println("Step 1/4: AI Provider")
	fmt.Println("  Which AI provider do you want to use?")
	fmt.Println("  [1] Anthropic Claude (recommended)")
	fmt.Println("  [2] OpenAI GPT-4o")
	fmt.Println("  [3] Ollama (local, free)")
	fmt.Println("  [4] Skip for now")
	fmt.Print("  Choice: ")

	scanner.Scan()
	choice := strings.TrimSpace(scanner.Text())

	switch choice {
	case "1":
		viper.Set("provider", "anthropic")
		fmt.Print("  Paste your Anthropic API key (sk-ant-...): ")
		scanner.Scan()
		key := strings.TrimSpace(scanner.Text())
		if key != "" {
			viper.Set("api_keys.anthropic", key)
			fmt.Println("  API key saved")
		}
	case "2":
		viper.Set("provider", "openai")
		fmt.Print("  Paste your OpenAI API key (sk-...): ")
		scanner.Scan()
		key := strings.TrimSpace(scanner.Text())
		if key != "" {
			viper.Set("api_keys.openai", key)
			fmt.Println("  API key saved")
		}
	case "3":
		viper.Set("provider", "ollama")
		fmt.Print("  Ollama host (default: http://localhost:11434): ")
		scanner.Scan()
		host := strings.TrimSpace(scanner.Text())
		if host != "" {
			viper.Set("ollama.host", host)
		} else {
			viper.Set("ollama.host", "http://localhost:11434")
		}
		fmt.Println("  Ollama configured")
	default:
		fmt.Println("  Skipped")
	}
	fmt.Println()

	// Step 2: Microsoft 365
	fmt.Println("Step 2/4: Microsoft 365 (optional)")
	fmt.Print("  Set up OneDrive/SharePoint/Teams access? [Y/n]: ")
	scanner.Scan()
	m365Choice := strings.TrimSpace(strings.ToLower(scanner.Text()))
	if m365Choice == "" || m365Choice == "y" || m365Choice == "yes" {
		fmt.Print("  Paste your Azure App Client ID: ")
		scanner.Scan()
		clientID := strings.TrimSpace(scanner.Text())
		if clientID != "" {
			viper.Set("azure.client_id", clientID)
			fmt.Println("  Client ID saved")
			fmt.Println("  -> Run: kit auth login  (when ready to authenticate)")
		}
	} else {
		fmt.Println("  Skipped")
	}
	fmt.Println()

	// Step 3: Email
	fmt.Println("Step 3/4: Email (optional)")
	fmt.Print("  Set up SMTP for kit send? [y/N]: ")
	scanner.Scan()
	smtpChoice := strings.TrimSpace(strings.ToLower(scanner.Text()))
	if smtpChoice == "y" || smtpChoice == "yes" {
		fmt.Print("  SMTP host: ")
		scanner.Scan()
		viper.Set("smtp.host", strings.TrimSpace(scanner.Text()))
		fmt.Print("  SMTP port (default: 587): ")
		scanner.Scan()
		port := strings.TrimSpace(scanner.Text())
		if port == "" {
			port = "587"
		}
		viper.Set("smtp.port", port)
		fmt.Print("  SMTP username: ")
		scanner.Scan()
		viper.Set("smtp.username", strings.TrimSpace(scanner.Text()))
		fmt.Println("  SMTP configured")
	} else {
		fmt.Println("  Skipped")
	}
	fmt.Println()

	// Save config
	if err := SaveConfig(); err != nil {
		return fmt.Errorf("could not save config: %w", err)
	}

	// Step 4: Done
	fmt.Println("Step 4/4: Done!")
	fmt.Println(strings.Repeat("-", 48))
	fmt.Println()
	fmt.Println("M365Kit is ready!")
	fmt.Println()
	fmt.Println("Quick start:")
	fmt.Println("  kit word read document.docx | kit ai summarize")
	fmt.Println("  kit auth login                  (Microsoft 365)")
	fmt.Println("  kit onedrive ls                 (after auth)")
	fmt.Println("  kit fs scan ~/Documents -r      (local files)")
	fmt.Println()
	fmt.Printf("Config file: %s\n", ConfigPath())
	fmt.Println("Type 'kit config show' to see all settings.")

	return nil
}

// WizardNonInteractive sets up config with defaults only (no user input).
func WizardNonInteractive() error {
	viper.Set("provider", "anthropic")
	viper.Set("output.color", true)
	viper.Set("output.format", "text")
	return SaveConfig()
}

// Validate checks config values and returns a list of issues.
func Validate() []ConfigIssue {
	var issues []ConfigIssue

	provider := viper.GetString("provider")

	// Check AI provider key
	switch provider {
	case "anthropic":
		key := os.Getenv("ANTHROPIC_API_KEY")
		if key == "" {
			key = viper.GetString("api_keys.anthropic")
		}
		if key == "" {
			issues = append(issues, ConfigIssue{
				Key:      "ai.provider",
				Severity: "error",
				Message:  fmt.Sprintf("provider is %q but ANTHROPIC_API_KEY is not set", provider),
				Fix:      "export ANTHROPIC_API_KEY=sk-ant-...\nOr: kit config set api_keys.anthropic sk-ant-...",
			})
		} else {
			issues = append(issues, ConfigIssue{
				Key:      "ai.provider",
				Severity: "info",
				Message:  "Anthropic API key configured",
			})
		}
	case "openai":
		key := os.Getenv("OPENAI_API_KEY")
		if key == "" {
			key = viper.GetString("api_keys.openai")
		}
		if key == "" {
			issues = append(issues, ConfigIssue{
				Key:      "ai.provider",
				Severity: "error",
				Message:  fmt.Sprintf("provider is %q but OPENAI_API_KEY is not set", provider),
				Fix:      "export OPENAI_API_KEY=sk-...",
			})
		}
	case "ollama":
		issues = append(issues, ConfigIssue{
			Key:      "ai.provider",
			Severity: "info",
			Message:  "Ollama configured (no API key needed)",
		})
	}

	// Check Microsoft 365
	clientID := os.Getenv("KIT_AZURE_CLIENT_ID")
	if clientID == "" {
		clientID = viper.GetString("azure.client_id")
	}
	if clientID != "" {
		issues = append(issues, ConfigIssue{
			Key:      "azure.client_id",
			Severity: "info",
			Message:  "Microsoft 365 client ID configured",
		})
	}

	// Check SMTP
	smtpHost := os.Getenv("KIT_SMTP_HOST")
	if smtpHost == "" {
		smtpHost = viper.GetString("smtp.host")
	}
	if smtpHost == "" {
		issues = append(issues, ConfigIssue{
			Key:      "smtp.host",
			Severity: "warning",
			Message:  "SMTP host is not set — kit send will not work",
			Fix:      "kit config set smtp.host your-smtp-host",
		})
	}

	return issues
}

// ToEnv returns all config values as a map of env var name -> value.
func ToEnv() map[string]string {
	env := make(map[string]string)

	if p := viper.GetString("provider"); p != "" {
		env["KIT_AI_PROVIDER"] = p
	}
	if m := viper.GetString("model"); m != "" {
		env["KIT_AI_MODEL"] = m
	}
	if k := viper.GetString("api_keys.anthropic"); k != "" {
		env["ANTHROPIC_API_KEY"] = k
	}
	if k := viper.GetString("api_keys.openai"); k != "" {
		env["OPENAI_API_KEY"] = k
	}
	if h := viper.GetString("ollama.host"); h != "" {
		env["KIT_OLLAMA_HOST"] = h
	}
	if c := viper.GetString("azure.client_id"); c != "" {
		env["KIT_AZURE_CLIENT_ID"] = c
	}
	if h := viper.GetString("smtp.host"); h != "" {
		env["KIT_SMTP_HOST"] = h
	}
	if p := viper.GetString("smtp.port"); p != "" {
		env["KIT_SMTP_PORT"] = p
	}
	if u := viper.GetString("smtp.username"); u != "" {
		env["KIT_SMTP_USERNAME"] = u
	}

	return env
}

// Set sets a config value and saves to disk.
func Set(key, value string) error {
	viper.Set(key, value)
	return SaveConfig()
}

// Get retrieves a config value.
func Get(key string) string {
	return viper.GetString(key)
}

// ResetConfig resets all config to defaults.
func ResetConfig() error {
	path := ConfigPath()
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("could not delete config: %w", err)
	}
	// Reset viper defaults
	viper.Set("provider", "anthropic")
	viper.Set("model", "claude-sonnet-4-20250514")
	viper.Set("output.color", true)
	viper.Set("output.format", "text")
	return nil
}

// SaveConfig writes the current config to ~/.kit/config.yaml.
func SaveConfig() error {
	dir := configDir()
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("could not create config directory: %w", err)
	}

	path := filepath.Join(dir, "config.yaml")
	if err := viper.WriteConfigAs(path); err != nil {
		return fmt.Errorf("could not write config: %w", err)
	}

	// Set secure permissions
	os.Chmod(path, 0600)
	return nil
}

// ConfigPath returns the path to the config file.
func ConfigPath() string {
	return filepath.Join(configDir(), "config.yaml")
}

// ShowConfig returns a formatted string of the current configuration.
func ShowConfig() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Config: %s\n\n", ConfigPath()))

	sb.WriteString("AI\n")
	sb.WriteString(fmt.Sprintf("  provider:  %s\n", viper.GetString("provider")))
	sb.WriteString(fmt.Sprintf("  model:     %s\n", viper.GetString("model")))
	if k := viper.GetString("api_keys.anthropic"); k != "" {
		sb.WriteString(fmt.Sprintf("  key:       %s****\n", k[:min(10, len(k))]))
	}
	if k := viper.GetString("api_keys.openai"); k != "" {
		sb.WriteString(fmt.Sprintf("  key:       %s****\n", k[:min(10, len(k))]))
	}
	sb.WriteString("\n")

	// Microsoft 365
	clientID := viper.GetString("azure.client_id")
	if clientID == "" {
		clientID = os.Getenv("KIT_AZURE_CLIENT_ID")
	}
	if clientID != "" {
		sb.WriteString("Microsoft 365\n")
		sb.WriteString(fmt.Sprintf("  client_id: %s\n", clientID))
		sb.WriteString("\n")
	}

	// SMTP
	smtpHost := viper.GetString("smtp.host")
	if smtpHost != "" {
		sb.WriteString("Email (SMTP)\n")
		sb.WriteString(fmt.Sprintf("  host:      %s\n", smtpHost))
		sb.WriteString(fmt.Sprintf("  port:      %s\n", viper.GetString("smtp.port")))
		sb.WriteString(fmt.Sprintf("  username:  %s\n", viper.GetString("smtp.username")))
		sb.WriteString("\n")
	}

	return sb.String()
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
