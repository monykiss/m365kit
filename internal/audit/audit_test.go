package audit

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLogWritesEntry(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "audit.log")

	l := NewLogger(path, "", "command", true)
	entry := Entry{
		Timestamp:  time.Now(),
		Machine:    "test-host",
		Command:    "word read",
		Args:       []string{"test.docx", "--json"},
		ExitCode:   0,
		DurationMs: 42,
	}

	if err := l.Log(context.Background(), entry); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) == 0 {
		t.Error("expected non-empty log file")
	}
}

func TestLogDisabledIsNoop(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "audit.log")

	l := NewLogger(path, "", "command", false)
	entry := Entry{Command: "word read"}
	l.Log(context.Background(), entry)

	_, err := os.Stat(path)
	if err == nil {
		t.Error("disabled logger should not create file")
	}
}

func TestLogAppendsEntries(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "audit.log")

	l := NewLogger(path, "", "command", true)
	for i := 0; i < 3; i++ {
		l.Log(context.Background(), Entry{Command: "test", DurationMs: int64(i)})
	}

	entries, err := ReadEntries(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 3 {
		t.Errorf("expected 3 entries, got %d", len(entries))
	}
}

func TestLogCreatesDirectory(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "subdir", "deep", "audit.log")

	l := NewLogger(path, "", "command", true)
	l.Log(context.Background(), Entry{Command: "test"})

	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("expected log file to be created in nested directory")
	}
}

func TestRedactSensitiveFlags(t *testing.T) {
	args := []string{"--key", "my-secret-key", "--output", "file.docx"}
	result := Redact(args)

	if result[0] != "--key" {
		t.Errorf("expected --key preserved, got %q", result[0])
	}
	if result[1] != "[REDACTED]" {
		t.Errorf("expected [REDACTED], got %q", result[1])
	}
	if result[2] != "--output" {
		t.Errorf("expected --output preserved, got %q", result[2])
	}
	if result[3] != "file.docx" {
		t.Errorf("expected file.docx preserved, got %q", result[3])
	}
}

func TestRedactPatterns(t *testing.T) {
	args := []string{"sk-ant-api03-secret", "normal-arg", "Bearer token123"}
	result := Redact(args)

	if result[0] != "[REDACTED]" {
		t.Errorf("expected sk-ant- redacted, got %q", result[0])
	}
	if result[1] != "normal-arg" {
		t.Errorf("expected normal-arg preserved, got %q", result[1])
	}
	if result[2] != "[REDACTED]" {
		t.Errorf("expected Bearer redacted, got %q", result[2])
	}
}

func TestRedactPreservesFilePaths(t *testing.T) {
	args := []string{"/home/user/docs/contract.docx", "--to", "md"}
	result := Redact(args)
	if result[0] != "/home/user/docs/contract.docx" {
		t.Errorf("expected file path preserved, got %q", result[0])
	}
}

func TestReadEntriesMissingFile(t *testing.T) {
	entries, err := ReadEntries("/nonexistent/audit.log")
	if err != nil {
		t.Fatalf("expected nil error for missing file, got: %v", err)
	}
	if len(entries) != 0 {
		t.Error("expected empty entries for missing file")
	}
}

func TestFilterEntries(t *testing.T) {
	now := time.Now()
	entries := []Entry{
		{Timestamp: now.Add(-2 * time.Hour), Command: "word read", UserID: "alice@test.com"},
		{Timestamp: now.Add(-1 * time.Hour), Command: "acl audit", UserID: "bob@test.com"},
		{Timestamp: now, Command: "word write", UserID: "alice@test.com"},
	}

	// Filter by command
	result := FilterEntries(entries, time.Time{}, time.Time{}, "word", "")
	if len(result) != 2 {
		t.Errorf("expected 2 word entries, got %d", len(result))
	}

	// Filter by user
	result = FilterEntries(entries, time.Time{}, time.Time{}, "", "bob@test.com")
	if len(result) != 1 {
		t.Errorf("expected 1 bob entry, got %d", len(result))
	}

	// Filter by time
	result = FilterEntries(entries, now.Add(-90*time.Minute), time.Time{}, "", "")
	if len(result) != 2 {
		t.Errorf("expected 2 recent entries, got %d", len(result))
	}
}

func TestLogSize(t *testing.T) {
	size := LogSize("/nonexistent/audit.log")
	if size != 0 {
		t.Errorf("expected 0 for missing file, got %d", size)
	}
}

func TestClear(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "audit.log")
	os.WriteFile(path, []byte("some data\n"), 0644)

	if err := Clear(path); err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(path)
	if len(data) != 0 {
		t.Error("expected empty file after clear")
	}
}
