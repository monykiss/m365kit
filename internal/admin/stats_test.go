package admin

import (
	"testing"
	"time"

	"github.com/klytics/m365kit/internal/audit"
)

func TestAggregateStatsBasic(t *testing.T) {
	now := time.Now()
	entries := []audit.Entry{
		{Timestamp: now, Command: "word read", UserID: "alice@test.com", ExitCode: 0},
		{Timestamp: now, Command: "word read", UserID: "bob@test.com", ExitCode: 0},
		{Timestamp: now, Command: "excel read", UserID: "alice@test.com", ExitCode: 0},
		{Timestamp: now, Command: "ai summarize", UserID: "alice@test.com", ExitCode: 1},
	}

	stats := AggregateStats(entries, StatsFilter{})
	if stats.CommandCount != 4 {
		t.Errorf("expected 4 commands, got %d", stats.CommandCount)
	}
	if stats.ActiveUsers != 2 {
		t.Errorf("expected 2 users, got %d", stats.ActiveUsers)
	}
	if stats.UserErrors != 1 {
		t.Errorf("expected 1 user error, got %d", stats.UserErrors)
	}
}

func TestAggregateStatsFilterByDate(t *testing.T) {
	now := time.Now()
	entries := []audit.Entry{
		{Timestamp: now.Add(-48 * time.Hour), Command: "old", UserID: "alice@test.com"},
		{Timestamp: now.Add(-1 * time.Hour), Command: "recent", UserID: "alice@test.com"},
		{Timestamp: now, Command: "now", UserID: "alice@test.com"},
	}

	stats := AggregateStats(entries, StatsFilter{Since: now.Add(-2 * time.Hour)})
	if stats.CommandCount != 2 {
		t.Errorf("expected 2 recent entries, got %d", stats.CommandCount)
	}
}

func TestAggregateStatsEmpty(t *testing.T) {
	stats := AggregateStats(nil, StatsFilter{})
	if stats.CommandCount != 0 {
		t.Errorf("expected 0 commands, got %d", stats.CommandCount)
	}
	if stats.ActiveUsers != 0 {
		t.Error("expected 0 active users")
	}
}

func TestTopCommandsSorted(t *testing.T) {
	entries := []audit.Entry{
		{Command: "word read"},
		{Command: "word read"},
		{Command: "word read"},
		{Command: "excel read"},
		{Command: "ai summarize"},
		{Command: "ai summarize"},
	}

	stats := AggregateStats(entries, StatsFilter{})
	if len(stats.TopCommands) < 3 {
		t.Fatalf("expected at least 3 commands, got %d", len(stats.TopCommands))
	}
	if stats.TopCommands[0].Command != "word read" {
		t.Errorf("expected word read as top command, got %q", stats.TopCommands[0].Command)
	}
	if stats.TopCommands[0].Count != 3 {
		t.Errorf("expected 3 for top command, got %d", stats.TopCommands[0].Count)
	}
}

func TestErrorRate(t *testing.T) {
	entries := []audit.Entry{
		{Command: "a", ExitCode: 0},
		{Command: "b", ExitCode: 1},
		{Command: "c", ExitCode: 0},
		{Command: "d", ExitCode: 2},
		{Command: "e", ExitCode: 0},
	}

	stats := AggregateStats(entries, StatsFilter{})
	if stats.ErrorRate != 40.0 {
		t.Errorf("expected 40%% error rate, got %.1f%%", stats.ErrorRate)
	}
}
