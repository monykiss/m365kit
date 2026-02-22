// Package audit provides command-level audit logging for compliance.
package audit

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Entry represents a single audit log entry.
type Entry struct {
	Timestamp  time.Time `json:"timestamp"`
	UserID     string    `json:"user_id,omitempty"`
	Machine    string    `json:"machine"`
	Command    string    `json:"command"`
	Args       []string  `json:"args"`
	ExitCode   int       `json:"exit_code"`
	DurationMs int64     `json:"duration_ms"`
	InputFile  string    `json:"input_file,omitempty"`
	OutputFile string    `json:"output_file,omitempty"`
}

// Logger writes audit entries to a file and/or HTTP endpoint.
type Logger struct {
	FilePath string
	Endpoint string
	Level    string
	Enabled  bool
}

// NewLogger creates a Logger. Returns a disabled logger if orgCfg is nil
// or audit is not enabled.
func NewLogger(filePath string, endpoint string, level string, enabled bool) *Logger {
	return &Logger{
		FilePath: filePath,
		Endpoint: endpoint,
		Level:    level,
		Enabled:  enabled,
	}
}

// Log writes a single audit entry. Non-blocking, best-effort.
func (l *Logger) Log(_ context.Context, entry Entry) error {
	if !l.Enabled || l.FilePath == "" {
		return nil
	}

	// Ensure parent directory exists
	dir := filepath.Dir(l.FilePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil // silently fail — never block commands
	}

	f, err := os.OpenFile(l.FilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil
	}
	defer f.Close()

	data, err := json.Marshal(entry)
	if err != nil {
		return nil
	}
	data = append(data, '\n')
	_, _ = f.Write(data)
	return nil
}

// ReadEntries reads all audit entries from the log file.
func ReadEntries(filePath string) ([]Entry, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var entries []Entry
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		if line == "" {
			continue
		}
		var e Entry
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			continue // skip malformed lines
		}
		entries = append(entries, e)
	}
	return entries, nil
}

// FilterEntries returns entries matching the given criteria.
func FilterEntries(entries []Entry, since, until time.Time, command, userID string) []Entry {
	var result []Entry
	for _, e := range entries {
		if !since.IsZero() && e.Timestamp.Before(since) {
			continue
		}
		if !until.IsZero() && e.Timestamp.After(until) {
			continue
		}
		if command != "" && !strings.Contains(e.Command, command) {
			continue
		}
		if userID != "" && e.UserID != userID {
			continue
		}
		result = append(result, e)
	}
	return result
}

// LogSize returns the size of the audit log in bytes, or 0 if not found.
func LogSize(filePath string) int64 {
	info, err := os.Stat(filePath)
	if err != nil {
		return 0
	}
	return info.Size()
}

// Clear truncates the audit log file.
func Clear(filePath string) error {
	return os.Truncate(filePath, 0)
}

// sensitiveFlags are flags whose following value should be redacted.
var sensitiveFlags = map[string]bool{
	"--key": true, "--token": true, "--password": true,
	"--secret": true, "--api-key": true, "--apikey": true,
}

// sensitivePatterns are value prefixes that indicate secrets.
var sensitivePatterns = []string{"sk-ant-", "sk-", "Bearer "}

// Redact sanitizes args to remove secrets.
func Redact(args []string) []string {
	result := make([]string, len(args))
	redactNext := false
	for i, arg := range args {
		if redactNext {
			result[i] = "[REDACTED]"
			redactNext = false
			continue
		}
		if sensitiveFlags[arg] {
			result[i] = arg
			redactNext = true
			continue
		}
		redacted := false
		for _, pat := range sensitivePatterns {
			if strings.HasPrefix(arg, pat) {
				result[i] = "[REDACTED]"
				redacted = true
				break
			}
		}
		if !redacted {
			result[i] = arg
		}
	}
	return result
}
