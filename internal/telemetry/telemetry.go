// Package telemetry provides local usage analytics with opt-in shipping.
// Privacy-first: no user ID, no file paths, no content.
package telemetry

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Event represents a single anonymous telemetry event.
type Event struct {
	Timestamp  time.Time `json:"ts"`
	Command    string    `json:"cmd"`
	DurationMs int64     `json:"ms"`
	ExitCode   int       `json:"exit"`
	FilesIn    int       `json:"fi,omitempty"`
	FilesOut   int       `json:"fo,omitempty"`
}

// Stats holds aggregated telemetry statistics.
type Stats struct {
	TotalCommands int            `json:"total_commands"`
	TopCommands   map[string]int `json:"top_commands"`
	AvgDuration   float64        `json:"avg_duration_ms"`
	ErrorCount    int            `json:"error_count"`
	Period        string         `json:"period"`
}

// Store manages the local telemetry store (~/.kit/telemetry.jsonl).
type Store struct {
	Path    string
	MaxSize int64 // default 10MB
}

// DefaultStore returns a Store at the default location.
func DefaultStore() *Store {
	home, _ := os.UserHomeDir()
	return &Store{
		Path:    filepath.Join(home, ".kit", "telemetry.jsonl"),
		MaxSize: 10 * 1024 * 1024,
	}
}

// Record appends an event to the local store. Non-blocking, best-effort.
func (s *Store) Record(e Event) {
	dir := filepath.Dir(s.Path)
	_ = os.MkdirAll(dir, 0755)

	f, err := os.OpenFile(s.Path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()

	data, err := json.Marshal(e)
	if err != nil {
		return
	}
	data = append(data, '\n')
	_, _ = f.Write(data)
}

// Summary returns aggregated stats from the local store.
func (s *Store) Summary() (*Stats, error) {
	data, err := os.ReadFile(s.Path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Stats{TopCommands: make(map[string]int)}, nil
		}
		return nil, err
	}

	stats := &Stats{TopCommands: make(map[string]int)}
	var totalDuration int64

	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		if line == "" {
			continue
		}
		var e Event
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			continue
		}
		stats.TotalCommands++
		stats.TopCommands[e.Command]++
		totalDuration += e.DurationMs
		if e.ExitCode != 0 {
			stats.ErrorCount++
		}
	}

	if stats.TotalCommands > 0 {
		stats.AvgDuration = float64(totalDuration) / float64(stats.TotalCommands)
	}

	return stats, nil
}

// Size returns the size of the telemetry store in bytes.
func (s *Store) Size() int64 {
	info, err := os.Stat(s.Path)
	if err != nil {
		return 0
	}
	return info.Size()
}

// Rotate truncates the store when it exceeds MaxSize.
func (s *Store) Rotate() error {
	info, err := os.Stat(s.Path)
	if err != nil {
		return nil
	}
	if info.Size() <= s.MaxSize {
		return nil
	}
	return os.Truncate(s.Path, 0)
}

// Clear removes all telemetry data.
func (s *Store) Clear() error {
	if _, err := os.Stat(s.Path); os.IsNotExist(err) {
		return nil
	}
	return os.Truncate(s.Path, 0)
}
