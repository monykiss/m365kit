package watch

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewWatcher(t *testing.T) {
	w, err := New(WatchConfig{
		Directories: []string{t.TempDir()},
		Debounce:    100,
	})
	if err != nil {
		t.Fatal(err)
	}
	if w == nil {
		t.Fatal("expected non-nil watcher")
	}
	w.watcher.Close()
}

func TestMatchesRuleExtension(t *testing.T) {
	w, _ := New(WatchConfig{})
	defer w.watcher.Close()

	rule := Rule{
		ID:         "r1",
		Extensions: []string{".docx", ".xlsx"},
		Enabled:    true,
	}

	if !w.matchesRule("/tmp/report.docx", rule) {
		t.Error("should match .docx")
	}
	if !w.matchesRule("/tmp/data.xlsx", rule) {
		t.Error("should match .xlsx")
	}
	if w.matchesRule("/tmp/image.png", rule) {
		t.Error("should not match .png")
	}
}

func TestMatchesRulePattern(t *testing.T) {
	w, _ := New(WatchConfig{})
	defer w.watcher.Close()

	rule := Rule{
		ID:      "r1",
		Pattern: "report_*.docx",
		Enabled: true,
	}

	if !w.matchesRule("/tmp/report_2024.docx", rule) {
		t.Error("should match report_2024.docx")
	}
	if w.matchesRule("/tmp/invoice.docx", rule) {
		t.Error("should not match invoice.docx")
	}
}

func TestMatchesRuleExtensionAndPattern(t *testing.T) {
	w, _ := New(WatchConfig{})
	defer w.watcher.Close()

	rule := Rule{
		ID:         "r1",
		Pattern:    "invoice_*",
		Extensions: []string{".docx"},
		Enabled:    true,
	}

	if !w.matchesRule("/tmp/invoice_001.docx", rule) {
		t.Error("should match invoice_001.docx")
	}
	if w.matchesRule("/tmp/invoice_001.xlsx", rule) {
		t.Error("should not match .xlsx")
	}
	if w.matchesRule("/tmp/report_001.docx", rule) {
		t.Error("should not match report_001.docx")
	}
}

func TestMatchesRuleDisabled(t *testing.T) {
	w, _ := New(WatchConfig{})
	defer w.watcher.Close()

	rule := Rule{
		ID:         "r1",
		Extensions: []string{".docx"},
		Enabled:    false,
	}

	// Even though extension matches, rule is disabled
	// (matchesRule doesn't check Enabled — that's done in processFile)
	if !w.matchesRule("/tmp/test.docx", rule) {
		t.Error("matchesRule should match regardless of Enabled flag")
	}
}

func TestWatcherEvents(t *testing.T) {
	dir := t.TempDir()

	w, err := New(WatchConfig{
		Directories: []string{dir},
		Rules: []Rule{
			{
				ID:         "test-rule",
				Extensions: []string{".docx"},
				Action:     Action{Name: "log", Type: "command"},
				Enabled:    true,
			},
		},
		Debounce: 50,
	})
	if err != nil {
		t.Fatal(err)
	}

	handlerCalled := make(chan string, 1)
	w.Handler = func(path string, rule Rule) error {
		handlerCalled <- path
		return nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		w.Start(ctx)
	}()

	// Give the watcher time to start
	time.Sleep(100 * time.Millisecond)

	// Create a matching file
	testFile := filepath.Join(dir, "test.docx")
	os.WriteFile(testFile, []byte("test"), 0644)

	// Wait for handler
	select {
	case path := <-handlerCalled:
		if path != testFile {
			t.Errorf("expected %q, got %q", testFile, path)
		}
	case <-time.After(2 * time.Second):
		t.Error("timeout waiting for handler call")
	}

	cancel()
}

func TestWatcherSkipsNonOffice(t *testing.T) {
	dir := t.TempDir()

	w, err := New(WatchConfig{
		Directories: []string{dir},
		Rules: []Rule{
			{ID: "r1", Extensions: []string{".docx"}, Enabled: true, Action: Action{Name: "test"}},
		},
		Debounce: 50,
	})
	if err != nil {
		t.Fatal(err)
	}

	handlerCalled := false
	w.Handler = func(path string, rule Rule) error {
		handlerCalled = true
		return nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go w.Start(ctx)
	time.Sleep(100 * time.Millisecond)

	// Create a non-office file
	os.WriteFile(filepath.Join(dir, "readme.txt"), []byte("test"), 0644)
	time.Sleep(200 * time.Millisecond)

	if handlerCalled {
		t.Error("handler should not be called for .txt files")
	}

	cancel()
}

func TestPIDFile(t *testing.T) {
	dir := t.TempDir()

	if err := WritePIDFile(dir); err != nil {
		t.Fatal(err)
	}

	pid, err := ReadPIDFile(dir)
	if err != nil {
		t.Fatal(err)
	}
	if pid != os.Getpid() {
		t.Errorf("expected PID %d, got %d", os.Getpid(), pid)
	}

	if err := RemovePIDFile(dir); err != nil {
		t.Fatal(err)
	}

	_, err = ReadPIDFile(dir)
	if err == nil {
		t.Error("expected error after removing PID file")
	}
}

func TestSaveAndLoadConfig(t *testing.T) {
	dir := t.TempDir()

	config := WatchConfig{
		Directories: []string{"/tmp/docs"},
		Rules: []Rule{
			{ID: "r1", Extensions: []string{".docx"}, Enabled: true},
		},
		Recursive: true,
		Debounce:  500,
	}

	if err := SaveConfig(dir, config); err != nil {
		t.Fatal(err)
	}

	loaded, err := LoadConfig(dir)
	if err != nil {
		t.Fatal(err)
	}

	if len(loaded.Directories) != 1 || loaded.Directories[0] != "/tmp/docs" {
		t.Errorf("directories mismatch: %v", loaded.Directories)
	}
	if !loaded.Recursive {
		t.Error("expected recursive=true")
	}
	if len(loaded.Rules) != 1 {
		t.Errorf("expected 1 rule, got %d", len(loaded.Rules))
	}
}

func TestGetStatus(t *testing.T) {
	w, _ := New(WatchConfig{
		Directories: []string{"/tmp/a", "/tmp/b"},
		Rules:       []Rule{{ID: "r1"}, {ID: "r2"}},
	})
	defer w.watcher.Close()

	status := w.GetStatus()
	if !status.Running {
		t.Error("expected running=true")
	}
	if len(status.Directories) != 2 {
		t.Errorf("expected 2 directories, got %d", len(status.Directories))
	}
	if status.Rules != 2 {
		t.Errorf("expected 2 rules, got %d", status.Rules)
	}
}

func TestEventJSON(t *testing.T) {
	evt := Event{
		Time:      time.Now(),
		Path:      "/tmp/test.docx",
		Operation: "CREATE",
		RuleID:    "r1",
		Status:    "processed",
	}

	data, err := json.Marshal(evt)
	if err != nil {
		t.Fatal(err)
	}

	var decoded Event
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Path != "/tmp/test.docx" {
		t.Errorf("Path = %q", decoded.Path)
	}
	if decoded.Status != "processed" {
		t.Errorf("Status = %q", decoded.Status)
	}
}

func TestDefaultDebounce(t *testing.T) {
	w, _ := New(WatchConfig{Debounce: 0})
	defer w.watcher.Close()

	if w.Config.Debounce != 500 {
		t.Errorf("expected default debounce 500, got %d", w.Config.Debounce)
	}
}
