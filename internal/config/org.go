package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"gopkg.in/yaml.v3"
)

// OrgConfig represents the org-wide configuration layer.
// Read from /etc/kit/org.yaml (macOS/Linux) or
// C:\ProgramData\M365Kit\org.yaml (Windows).
type OrgConfig struct {
	OrgName   string `yaml:"org_name" json:"org_name"`
	OrgDomain string `yaml:"org_domain" json:"org_domain"`
	OrgID     string `yaml:"org_id" json:"org_id"`

	Azure struct {
		ClientID string `yaml:"client_id" json:"client_id"`
		TenantID string `yaml:"tenant_id" json:"tenant_id"`
	} `yaml:"azure" json:"azure"`

	AI struct {
		Provider string `yaml:"provider" json:"provider"`
		Model    string `yaml:"model" json:"model"`
	} `yaml:"ai" json:"ai"`

	AllowedCommands []string `yaml:"allowed_commands" json:"allowed_commands"`

	Locked struct {
		AzureClientID bool `yaml:"azure_client_id" json:"azure_client_id"`
		AIProvider    bool `yaml:"ai_provider" json:"ai_provider"`
		Commands      bool `yaml:"commands" json:"commands"`
	} `yaml:"locked" json:"locked"`

	Audit struct {
		Enabled  bool   `yaml:"enabled" json:"enabled"`
		FilePath string `yaml:"file_path" json:"file_path"`
		Endpoint string `yaml:"endpoint" json:"endpoint"`
		Level    string `yaml:"level" json:"level"`
	} `yaml:"audit" json:"audit"`

	Telemetry struct {
		Enabled  bool   `yaml:"enabled" json:"enabled"`
		Endpoint string `yaml:"endpoint" json:"endpoint"`
	} `yaml:"telemetry" json:"telemetry"`
}

// OrgConfigPath returns the platform-specific path for org config.
func OrgConfigPath() string {
	if runtime.GOOS == "windows" {
		return filepath.Join(os.Getenv("ProgramData"), "M365Kit", "org.yaml")
	}
	return "/etc/kit/org.yaml"
}

// LoadOrgConfig reads the org config file. Returns nil (not error) if file does not exist.
func LoadOrgConfig() (*OrgConfig, error) {
	path := OrgConfigPath()
	return LoadOrgConfigFrom(path)
}

// LoadOrgConfigFrom reads the org config from a specific path.
func LoadOrgConfigFrom(path string) (*OrgConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("could not read org config at %s: %w", path, err)
	}

	var cfg OrgConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("invalid org config at %s: %w", path, err)
	}

	return &cfg, nil
}

// ValidateOrgConfig checks that an org config is valid.
func ValidateOrgConfig(cfg *OrgConfig) []string {
	var issues []string
	if cfg.OrgName == "" {
		issues = append(issues, "org_name is required")
	}
	if cfg.OrgDomain == "" {
		issues = append(issues, "org_domain is required")
	}
	if cfg.Audit.Level != "" && cfg.Audit.Level != "command" && cfg.Audit.Level != "verbose" {
		issues = append(issues, fmt.Sprintf("audit.level must be 'command' or 'verbose', got %q", cfg.Audit.Level))
	}
	if cfg.AI.Provider != "" {
		valid := map[string]bool{"anthropic": true, "openai": true, "ollama": true}
		if !valid[cfg.AI.Provider] {
			issues = append(issues, fmt.Sprintf("ai.provider must be anthropic, openai, or ollama, got %q", cfg.AI.Provider))
		}
	}
	return issues
}

// IsCommandAllowed checks if a command is allowed by org policy.
// Empty allowed_commands means all commands are allowed.
func IsCommandAllowed(orgCfg *OrgConfig, commandPath string) bool {
	if orgCfg == nil || len(orgCfg.AllowedCommands) == 0 {
		return true
	}
	// commandPath is like "kit word read" — check if "word" or "word read" is allowed
	parts := strings.Fields(commandPath)
	for _, allowed := range orgCfg.AllowedCommands {
		for _, part := range parts {
			if part == allowed {
				return true
			}
		}
		if strings.Contains(commandPath, allowed) {
			return true
		}
	}
	return false
}

// GenerateOrgTemplate returns a YAML template for org config.
func GenerateOrgTemplate(orgName, domain string) string {
	return fmt.Sprintf(`# M365Kit Organization Configuration
# Deploy to: %s
# Permissions: readable by all users, writable only by root/Administrators

org_name: %q
org_domain: %q
org_id: ""

azure:
  client_id: ""
  tenant_id: ""

ai:
  provider: anthropic
  model: claude-sonnet-4-20250514

# allowed_commands: []  # empty = all allowed

locked:
  azure_client_id: false
  ai_provider: false
  commands: false

audit:
  enabled: true
  file_path: "~/.kit/audit.log"
  endpoint: ""
  level: command

telemetry:
  enabled: false
  endpoint: ""
`, OrgConfigPath(), orgName, domain)
}

// AuditLogPath returns the resolved audit log path from org config.
func (o *OrgConfig) AuditLogPath() string {
	if o.Audit.FilePath == "" {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, ".kit", "audit.log")
	}
	path := o.Audit.FilePath
	if strings.HasPrefix(path, "~/") {
		home, _ := os.UserHomeDir()
		path = filepath.Join(home, path[2:])
	}
	return path
}
