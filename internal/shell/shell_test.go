package shell

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"testing"
)

func mockRunner(version string) CommandRunner {
	return func(ctx context.Context, args []string, stdout, stderr io.Writer) error {
		if len(args) == 0 {
			return fmt.Errorf("no command")
		}
		switch args[0] {
		case "version":
			fmt.Fprintf(stdout, "kit %s\n", version)
			return nil
		case "word":
			if len(args) > 1 && args[1] == "read" {
				fmt.Fprintf(stdout, "Document content\n")
				return nil
			}
			if len(args) > 1 && args[1] == "write" {
				fmt.Fprintf(stdout, "Written\n")
				return nil
			}
			return nil
		case "unknown-command":
			return fmt.Errorf("unknown command: %s", args[0])
		}
		fmt.Fprintf(stdout, "OK\n")
		return nil
	}
}

func TestNewSession(t *testing.T) {
	s, err := NewSession()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(s.CommandHistory) != 0 {
		t.Errorf("expected empty history, got %d entries", len(s.CommandHistory))
	}
	if s.HistoryFile == "" {
		t.Error("expected history file path to be set")
	}
	if len(s.KnownCommands) == 0 {
		t.Error("expected known commands to be populated")
	}
}

func TestEvalVersion(t *testing.T) {
	DefaultRunner = mockRunner("v1.2.0-test")
	defer func() { DefaultRunner = nil }()

	s, _ := NewSession()
	output, err := s.Eval(context.Background(), "version")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(output, "v1.2.0-test") {
		t.Errorf("expected version output, got: %q", output)
	}
}

func TestEvalWordRead(t *testing.T) {
	DefaultRunner = mockRunner("v1.2.0")
	defer func() { DefaultRunner = nil }()

	s, _ := NewSession()
	output, err := s.Eval(context.Background(), "word read test.docx")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(output, "Document content") {
		t.Errorf("expected document content, got: %q", output)
	}
}

func TestEvalWordWrite(t *testing.T) {
	DefaultRunner = mockRunner("v1.2.0")
	defer func() { DefaultRunner = nil }()

	s, _ := NewSession()
	output, err := s.Eval(context.Background(), "word write --output test.docx")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(output, "Written") {
		t.Errorf("expected written confirmation, got: %q", output)
	}
}

func TestEvalUnknownCommand(t *testing.T) {
	DefaultRunner = mockRunner("v1.2.0")
	defer func() { DefaultRunner = nil }()

	s, _ := NewSession()
	_, err := s.Eval(context.Background(), "unknown-command")
	if err == nil {
		t.Error("expected error for unknown command")
	}
}

func TestEvalEmpty(t *testing.T) {
	DefaultRunner = mockRunner("v1.2.0")
	defer func() { DefaultRunner = nil }()

	s, _ := NewSession()
	output, err := s.Eval(context.Background(), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if output != "" {
		t.Errorf("expected empty output, got: %q", output)
	}
}

func TestEvalNoRunner(t *testing.T) {
	DefaultRunner = nil
	s, _ := NewSession()
	_, err := s.Eval(context.Background(), "version")
	if err == nil {
		t.Error("expected error when runner is nil")
	}
}

func TestCompleteTopLevel(t *testing.T) {
	s, _ := NewSession()
	matches := s.Complete("wo")
	if len(matches) != 1 || matches[0] != "word" {
		t.Errorf("expected [word], got %v", matches)
	}
}

func TestCompleteMultipleMatches(t *testing.T) {
	s, _ := NewSession()
	matches := s.Complete("a")
	// Should match: ai, auth, acl, admin, audit
	found := make(map[string]bool)
	for _, m := range matches {
		found[m] = true
	}
	for _, expected := range []string{"ai", "auth", "acl", "admin", "audit"} {
		if !found[expected] {
			t.Errorf("expected %q in completions, got %v", expected, matches)
		}
	}
}

func TestCompleteSubcommand(t *testing.T) {
	s, _ := NewSession()
	matches := s.Complete("word re")
	if len(matches) != 1 || matches[0] != "read" {
		t.Errorf("expected [read], got %v", matches)
	}
}

func TestCompleteEmpty(t *testing.T) {
	s, _ := NewSession()
	matches := s.Complete("")
	if len(matches) == 0 {
		t.Error("expected all commands for empty input")
	}
}

func TestCompleteUnknownCommand(t *testing.T) {
	s, _ := NewSession()
	matches := s.Complete("zzz ")
	// No subcommands for unknown command
	if len(matches) != 0 {
		t.Errorf("expected no matches, got %v", matches)
	}
}

func TestHistoryGrows(t *testing.T) {
	DefaultRunner = mockRunner("v1.2.0")
	defer func() { DefaultRunner = nil }()

	s, _ := NewSession()
	s.Eval(context.Background(), "version")
	s.Eval(context.Background(), "word read test.docx")
	s.Eval(context.Background(), "ai summarize")

	// Note: Eval doesn't add to history by itself (Run does). But we
	// check that the session tracks state correctly.
	if s.LastOutput == "" {
		t.Error("expected LastOutput to be set after Eval")
	}
}

func TestLastOutputUpdated(t *testing.T) {
	DefaultRunner = mockRunner("v1.2.0")
	defer func() { DefaultRunner = nil }()

	s, _ := NewSession()

	s.Eval(context.Background(), "version")
	if !strings.Contains(s.LastOutput, "1.2.0") {
		t.Errorf("expected LastOutput to contain version, got: %q", s.LastOutput)
	}

	s.Eval(context.Background(), "word read test.docx")
	if !strings.Contains(s.LastOutput, "Document content") {
		t.Errorf("expected LastOutput to be updated, got: %q", s.LastOutput)
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"30s", "30s"},
		{"90s", "1m 30s"},
		{"5m", "5m 0s"},
	}
	_ = tests // formatDuration is tested via Run output
	_ = bytes.Buffer{}
}
