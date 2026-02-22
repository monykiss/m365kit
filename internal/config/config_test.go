package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/viper"
)

func setupTestConfig(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	viper.Reset()
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(dir)
	viper.Set("provider", "anthropic")
	viper.Set("model", "claude-sonnet-4-20250514")
	viper.Set("output.color", true)

	// Override configDir for tests
	os.Setenv("HOME", dir)
	t.Cleanup(func() {
		viper.Reset()
	})
}

func TestLoadDefaults(t *testing.T) {
	viper.Reset()
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Provider != "anthropic" {
		t.Errorf("default provider = %q", cfg.Provider)
	}
}

func TestValidateNoAPIKey(t *testing.T) {
	setupTestConfig(t)
	t.Setenv("ANTHROPIC_API_KEY", "")
	viper.Set("api_keys.anthropic", "")
	viper.Set("provider", "anthropic")

	issues := Validate()
	hasError := false
	for _, issue := range issues {
		if issue.Severity == "error" && strings.Contains(issue.Message, "ANTHROPIC_API_KEY") {
			hasError = true
		}
	}
	if !hasError {
		t.Error("expected error about missing API key")
	}
}

func TestValidateWithAPIKey(t *testing.T) {
	setupTestConfig(t)
	t.Setenv("ANTHROPIC_API_KEY", "sk-ant-test-key")
	viper.Set("provider", "anthropic")

	issues := Validate()
	for _, issue := range issues {
		if issue.Key == "ai.provider" && issue.Severity == "error" {
			t.Errorf("unexpected error: %s", issue.Message)
		}
	}
}

func TestValidateSMTPWarning(t *testing.T) {
	setupTestConfig(t)
	t.Setenv("KIT_SMTP_HOST", "")
	viper.Set("smtp.host", "")

	issues := Validate()
	hasWarning := false
	for _, issue := range issues {
		if issue.Severity == "warning" && strings.Contains(issue.Message, "SMTP") {
			hasWarning = true
		}
	}
	if !hasWarning {
		t.Error("expected SMTP warning")
	}
}

func TestToEnv(t *testing.T) {
	setupTestConfig(t)
	viper.Set("provider", "anthropic")
	viper.Set("model", "claude-opus-4-6")
	viper.Set("api_keys.anthropic", "sk-ant-test")
	viper.Set("azure.client_id", "test-client-id")

	env := ToEnv()
	if env["KIT_AI_PROVIDER"] != "anthropic" {
		t.Errorf("KIT_AI_PROVIDER = %q", env["KIT_AI_PROVIDER"])
	}
	if env["KIT_AI_MODEL"] != "claude-opus-4-6" {
		t.Errorf("KIT_AI_MODEL = %q", env["KIT_AI_MODEL"])
	}
	if env["ANTHROPIC_API_KEY"] != "sk-ant-test" {
		t.Errorf("ANTHROPIC_API_KEY = %q", env["ANTHROPIC_API_KEY"])
	}
	if env["KIT_AZURE_CLIENT_ID"] != "test-client-id" {
		t.Errorf("KIT_AZURE_CLIENT_ID = %q", env["KIT_AZURE_CLIENT_ID"])
	}
}

func TestSetAndGet(t *testing.T) {
	dir := t.TempDir()
	os.Setenv("HOME", dir)
	viper.Reset()
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(filepath.Join(dir, ".kit"))

	// Create .kit directory
	os.MkdirAll(filepath.Join(dir, ".kit"), 0700)

	if err := Set("provider", "openai"); err != nil {
		t.Fatal(err)
	}

	got := Get("provider")
	if got != "openai" {
		t.Errorf("Get(provider) = %q, want %q", got, "openai")
	}
}

func TestShowConfig(t *testing.T) {
	setupTestConfig(t)
	viper.Set("provider", "anthropic")
	viper.Set("model", "claude-opus-4-6")

	output := ShowConfig()
	if !strings.Contains(output, "anthropic") {
		t.Error("ShowConfig should contain provider")
	}
	if !strings.Contains(output, "claude-opus-4-6") {
		t.Error("ShowConfig should contain model")
	}
}

func TestWizardNonInteractive(t *testing.T) {
	dir := t.TempDir()
	os.Setenv("HOME", dir)
	viper.Reset()
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")

	if err := WizardNonInteractive(); err != nil {
		t.Fatal(err)
	}

	if viper.GetString("provider") != "anthropic" {
		t.Errorf("provider = %q", viper.GetString("provider"))
	}
}

func TestWizardInteractive(t *testing.T) {
	dir := t.TempDir()
	os.Setenv("HOME", dir)
	viper.Reset()
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")

	// Simulate user input: choice 4 (skip), n (skip M365), n (skip SMTP)
	input := strings.NewReader("4\nn\nn\n")
	if err := Wizard(input); err != nil {
		t.Fatal(err)
	}
}

func TestConfigPath(t *testing.T) {
	path := ConfigPath()
	if !strings.Contains(path, ".kit") || !strings.Contains(path, "config.yaml") {
		t.Errorf("unexpected path: %q", path)
	}
}

func TestResetConfig(t *testing.T) {
	dir := t.TempDir()
	os.Setenv("HOME", dir)
	viper.Reset()
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")

	// Create config
	viper.Set("provider", "openai")
	SaveConfig()

	if err := ResetConfig(); err != nil {
		t.Fatal(err)
	}

	if viper.GetString("provider") != "anthropic" {
		t.Errorf("provider should reset to default, got %q", viper.GetString("provider"))
	}
}
