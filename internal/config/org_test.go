package config

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestLoadOrgConfigMissing(t *testing.T) {
	cfg, err := LoadOrgConfigFrom("/nonexistent/org.yaml")
	if err != nil {
		t.Fatalf("expected nil error for missing file, got: %v", err)
	}
	if cfg != nil {
		t.Error("expected nil config for missing file")
	}
}

func TestLoadOrgConfigValid(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "org.yaml")
	content := `
org_name: "Test Corp"
org_domain: "test.com"
org_id: "test-001"
azure:
  client_id: "abc-123"
ai:
  provider: anthropic
allowed_commands:
  - word
  - excel
locked:
  azure_client_id: true
audit:
  enabled: true
  level: command
`
	os.WriteFile(path, []byte(content), 0644)

	cfg, err := LoadOrgConfigFrom(path)
	if err != nil {
		t.Fatalf("LoadOrgConfigFrom failed: %v", err)
	}
	if cfg.OrgName != "Test Corp" {
		t.Errorf("OrgName = %q", cfg.OrgName)
	}
	if cfg.Azure.ClientID != "abc-123" {
		t.Errorf("Azure.ClientID = %q", cfg.Azure.ClientID)
	}
	if !cfg.Locked.AzureClientID {
		t.Error("expected Locked.AzureClientID = true")
	}
	if len(cfg.AllowedCommands) != 2 {
		t.Errorf("expected 2 allowed commands, got %d", len(cfg.AllowedCommands))
	}
}

func TestIsCommandAllowedEmptyList(t *testing.T) {
	cfg := &OrgConfig{}
	if !IsCommandAllowed(cfg, "kit ai summarize") {
		t.Error("empty allowed_commands should allow all")
	}
}

func TestIsCommandAllowedNilConfig(t *testing.T) {
	if !IsCommandAllowed(nil, "kit word read") {
		t.Error("nil config should allow all")
	}
}

func TestIsCommandAllowedRestricted(t *testing.T) {
	cfg := &OrgConfig{
		AllowedCommands: []string{"word", "excel"},
	}
	if !IsCommandAllowed(cfg, "kit word read") {
		t.Error("word should be allowed")
	}
	if !IsCommandAllowed(cfg, "kit excel write") {
		t.Error("excel should be allowed")
	}
	if IsCommandAllowed(cfg, "kit ai summarize") {
		t.Error("ai should be blocked")
	}
	if IsCommandAllowed(cfg, "kit acl audit") {
		t.Error("acl should be blocked")
	}
}

func TestValidateOrgConfigValid(t *testing.T) {
	cfg := &OrgConfig{OrgName: "Test", OrgDomain: "test.com"}
	cfg.AI.Provider = "anthropic"
	cfg.Audit.Level = "command"
	issues := ValidateOrgConfig(cfg)
	if len(issues) != 0 {
		t.Errorf("expected no issues, got: %v", issues)
	}
}

func TestValidateOrgConfigMissing(t *testing.T) {
	cfg := &OrgConfig{}
	issues := ValidateOrgConfig(cfg)
	if len(issues) < 2 {
		t.Errorf("expected at least 2 issues for empty config, got %d", len(issues))
	}
}

func TestValidateOrgConfigBadProvider(t *testing.T) {
	cfg := &OrgConfig{OrgName: "X", OrgDomain: "x.com"}
	cfg.AI.Provider = "gpt5"
	issues := ValidateOrgConfig(cfg)
	found := false
	for _, issue := range issues {
		if issue != "" {
			found = true
		}
	}
	if !found {
		t.Error("expected validation issue for bad provider")
	}
}

func TestOrgConfigPath(t *testing.T) {
	path := OrgConfigPath()
	if runtime.GOOS == "windows" {
		if path == "" {
			t.Error("expected non-empty path on Windows")
		}
	} else {
		if path != "/etc/kit/org.yaml" {
			t.Errorf("expected /etc/kit/org.yaml, got %q", path)
		}
	}
}

func TestGenerateOrgTemplate(t *testing.T) {
	tmpl := GenerateOrgTemplate("Acme Corp", "acme.com")
	if tmpl == "" {
		t.Fatal("expected non-empty template")
	}
	if !contains(tmpl, "Acme Corp") {
		t.Error("template should contain org name")
	}
	if !contains(tmpl, "acme.com") {
		t.Error("template should contain domain")
	}
}

func TestAuditLogPath(t *testing.T) {
	cfg := &OrgConfig{}
	path := cfg.AuditLogPath()
	if path == "" {
		t.Error("expected non-empty audit log path")
	}
}

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && (s == substr || len(s) > len(substr) && containsStr(s, substr))
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
