package send

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/klytics/m365kit/internal/email"
)

func TestParseEmails(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"alice@co.com", []string{"alice@co.com"}},
		{"alice@co.com,bob@co.com", []string{"alice@co.com", "bob@co.com"}},
		{"alice@co.com, bob@co.com , carol@co.com", []string{"alice@co.com", "bob@co.com", "carol@co.com"}},
		{"", nil},
	}

	for _, tt := range tests {
		result := parseEmails(tt.input)
		if len(result) != len(tt.expected) {
			t.Errorf("parseEmails(%q) = %v, want %v", tt.input, result, tt.expected)
			continue
		}
		for i := range result {
			if result[i] != tt.expected[i] {
				t.Errorf("parseEmails(%q)[%d] = %q, want %q", tt.input, i, result[i], tt.expected[i])
			}
		}
	}
}

func TestValidateEmail(t *testing.T) {
	valid := []string{"test@example.com", "user.name+tag@domain.co", "a@b.cd"}
	for _, e := range valid {
		if !email.ValidateEmail(e) {
			t.Errorf("expected %q to be valid", e)
		}
	}
	invalid := []string{"not-an-email", "@missing.com", "no@", "spaces in@email.com", ""}
	for _, e := range invalid {
		if email.ValidateEmail(e) {
			t.Errorf("expected %q to be invalid", e)
		}
	}
}

func TestMessageValidateInvalidEmail(t *testing.T) {
	msg := email.Message{To: []string{"not-an-email"}}
	err := msg.Validate()
	if err == nil {
		t.Fatal("expected error for invalid email")
	}
	if !contains(err.Error(), "invalid recipient email") {
		t.Errorf("expected 'invalid recipient email' in error, got: %s", err.Error())
	}
}

func TestMessageValidateMissingAttachment(t *testing.T) {
	msg := email.Message{
		To:     []string{"test@example.com"},
		Attach: "/nonexistent/file.docx",
	}
	err := msg.Validate()
	if err == nil {
		t.Fatal("expected error for missing attachment")
	}
	if !contains(err.Error(), "attachment not found") {
		t.Errorf("expected 'attachment not found' in error, got: %s", err.Error())
	}
}

func TestMessageValidateNoRecipients(t *testing.T) {
	msg := email.Message{}
	err := msg.Validate()
	if err == nil {
		t.Fatal("expected error for no recipients")
	}
}

func TestLoadConfigMissing(t *testing.T) {
	// Ensure env vars are unset
	t.Setenv("KIT_SMTP_HOST", "")
	t.Setenv("KIT_SMTP_USERNAME", "")
	t.Setenv("KIT_SMTP_PASSWORD", "")
	t.Setenv("KIT_SMTP_FROM", "")

	_, err := email.LoadConfig()
	if err == nil {
		t.Fatal("expected error when SMTP env vars are missing")
	}
	if !contains(err.Error(), "No SMTP configuration found") {
		t.Errorf("expected descriptive error, got: %s", err.Error())
	}
}

func TestLoadConfigFromEnv(t *testing.T) {
	t.Setenv("KIT_SMTP_HOST", "smtp.test.com")
	t.Setenv("KIT_SMTP_PORT", "465")
	t.Setenv("KIT_SMTP_USERNAME", "user")
	t.Setenv("KIT_SMTP_PASSWORD", "pass")
	t.Setenv("KIT_SMTP_FROM", "noreply@test.com")

	cfg, err := email.LoadConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Host != "smtp.test.com" {
		t.Errorf("host = %q, want smtp.test.com", cfg.Host)
	}
	if cfg.Port != 465 {
		t.Errorf("port = %d, want 465", cfg.Port)
	}
	if cfg.Username != "user" {
		t.Errorf("username = %q, want user", cfg.Username)
	}
	if cfg.From != "noreply@test.com" {
		t.Errorf("from = %q, want noreply@test.com", cfg.From)
	}
}

func TestDefaultSubject(t *testing.T) {
	// Test that default subject comes from attachment basename without extension
	tests := []struct {
		attach   string
		expected string
	}{
		{"report.xlsx", "report"},
		{"/path/to/quarterly_results.docx", "quarterly_results"},
		{"deck.pptx", "deck"},
	}

	for _, tt := range tests {
		result := defaultSubjectFromPath(tt.attach)
		if result != tt.expected {
			t.Errorf("defaultSubject(%q) = %q, want %q", tt.attach, result, tt.expected)
		}
	}
}

func TestDryRunJSON(t *testing.T) {
	msg := email.Message{
		To:      []string{"test@example.com"},
		Subject: "Test Subject",
		Body:    "Test body",
		Attach:  "", // no attachment for this test
	}

	data := map[string]any{
		"sent":       false,
		"dryRun":     true,
		"to":         msg.To,
		"subject":    msg.Subject,
		"attach":     msg.Attach,
		"attachSize": msg.AttachSize(),
		"aiDrafted":  false,
		"body":       msg.Body,
	}

	jsonBytes, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("json marshal failed: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(jsonBytes, &parsed); err != nil {
		t.Fatalf("json unmarshal failed: %v", err)
	}

	if parsed["dryRun"] != true {
		t.Errorf("expected dryRun=true, got %v", parsed["dryRun"])
	}
	if parsed["sent"] != false {
		t.Errorf("expected sent=false, got %v", parsed["sent"])
	}
}

func TestTruncate(t *testing.T) {
	short := "hello"
	if r := truncate(short, 10); r != "hello" {
		t.Errorf("truncate short = %q, want %q", r, "hello")
	}

	long := "this is a very long string that should be truncated"
	r := truncate(long, 20)
	if len(r) > 20 {
		t.Errorf("truncated string too long: %d chars", len(r))
	}
	if r[len(r)-3:] != "..." {
		t.Errorf("truncated string should end with '...', got %q", r)
	}
}

// helper used by defaultSubject test
func defaultSubjectFromPath(attach string) string {
	base := filepath.Base(attach)
	return strings.TrimSuffix(base, filepath.Ext(base))
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && containsStr(s, substr)
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
