package telemetry

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRecordAppends(t *testing.T) {
	dir := t.TempDir()
	s := &Store{Path: filepath.Join(dir, "telemetry.jsonl"), MaxSize: 10 * 1024 * 1024}

	s.Record(Event{Timestamp: time.Now(), Command: "word read", DurationMs: 10, ExitCode: 0})
	s.Record(Event{Timestamp: time.Now(), Command: "excel read", DurationMs: 20, ExitCode: 0})

	data, err := os.ReadFile(s.Path)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) == 0 {
		t.Error("expected non-empty telemetry file")
	}
}

func TestSummaryAggregates(t *testing.T) {
	dir := t.TempDir()
	s := &Store{Path: filepath.Join(dir, "telemetry.jsonl"), MaxSize: 10 * 1024 * 1024}

	s.Record(Event{Command: "word read", DurationMs: 10, ExitCode: 0})
	s.Record(Event{Command: "word read", DurationMs: 20, ExitCode: 0})
	s.Record(Event{Command: "excel read", DurationMs: 30, ExitCode: 1})

	stats, err := s.Summary()
	if err != nil {
		t.Fatal(err)
	}
	if stats.TotalCommands != 3 {
		t.Errorf("expected 3 commands, got %d", stats.TotalCommands)
	}
	if stats.TopCommands["word read"] != 2 {
		t.Errorf("expected 2 word read, got %d", stats.TopCommands["word read"])
	}
	if stats.ErrorCount != 1 {
		t.Errorf("expected 1 error, got %d", stats.ErrorCount)
	}
	if stats.AvgDuration != 20.0 {
		t.Errorf("expected avg 20ms, got %.1f", stats.AvgDuration)
	}
}

func TestSummaryEmptyStore(t *testing.T) {
	s := &Store{Path: "/nonexistent/telemetry.jsonl", MaxSize: 10 * 1024 * 1024}
	stats, err := s.Summary()
	if err != nil {
		t.Fatal(err)
	}
	if stats.TotalCommands != 0 {
		t.Errorf("expected 0 commands, got %d", stats.TotalCommands)
	}
}

func TestRotate(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "telemetry.jsonl")

	// Write more than MaxSize
	data := make([]byte, 1024)
	for i := range data {
		data[i] = 'x'
	}
	os.WriteFile(path, data, 0644)

	s := &Store{Path: path, MaxSize: 100} // 100 byte limit
	if err := s.Rotate(); err != nil {
		t.Fatal(err)
	}

	info, _ := os.Stat(path)
	if info.Size() != 0 {
		t.Errorf("expected truncated file, got %d bytes", info.Size())
	}
}

func TestRotateUnderLimit(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "telemetry.jsonl")
	os.WriteFile(path, []byte("small"), 0644)

	s := &Store{Path: path, MaxSize: 10 * 1024 * 1024}
	s.Rotate()

	data, _ := os.ReadFile(path)
	if string(data) != "small" {
		t.Error("should not truncate file under limit")
	}
}

func TestStoreCreatesDirectory(t *testing.T) {
	dir := t.TempDir()
	s := &Store{Path: filepath.Join(dir, "sub", "deep", "telemetry.jsonl"), MaxSize: 10 * 1024 * 1024}
	s.Record(Event{Command: "test"})

	if _, err := os.Stat(s.Path); os.IsNotExist(err) {
		t.Error("expected file to be created in nested directory")
	}
}

func TestClear(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "telemetry.jsonl")
	os.WriteFile(path, []byte("data\n"), 0644)

	s := &Store{Path: path}
	if err := s.Clear(); err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(path)
	if len(data) != 0 {
		t.Error("expected empty file after clear")
	}
}
