// Package watch provides a file system watcher for automated document processing.
// It monitors directories for new/modified Office documents and triggers configured actions.
package watch

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// Action defines what to do when a file event is detected.
type Action struct {
	Name    string `json:"name"`
	Type    string `json:"type"`    // "template", "ai", "copy", "command"
	Options map[string]string `json:"options,omitempty"`
}

// Rule defines a watch rule: which files to match and what action to take.
type Rule struct {
	ID         string   `json:"id"`
	Pattern    string   `json:"pattern"`    // Glob pattern (e.g., "*.docx", "contracts/*.xlsx")
	Extensions []string `json:"extensions"` // File extensions to match
	Action     Action   `json:"action"`
	Enabled    bool     `json:"enabled"`
}

// WatchConfig holds the complete watcher configuration.
type WatchConfig struct {
	Directories []string `json:"directories"`
	Rules       []Rule   `json:"rules"`
	Recursive   bool     `json:"recursive"`
	Debounce    int      `json:"debounceMs"` // Milliseconds to wait before processing
}

// Event represents a file event that was detected and processed.
type Event struct {
	Time      time.Time `json:"time"`
	Path      string    `json:"path"`
	Operation string    `json:"operation"` // "create", "modify", "rename"
	RuleID    string    `json:"ruleId,omitempty"`
	Action    string    `json:"action,omitempty"`
	Status    string    `json:"status"` // "processed", "error", "skipped"
	Error     string    `json:"error,omitempty"`
}

// Watcher monitors directories for file changes and triggers actions.
type Watcher struct {
	Config   WatchConfig
	Logger   *log.Logger
	Events   []Event
	Handler  EventHandler
	mu       sync.Mutex
	watcher  *fsnotify.Watcher
	debounce map[string]*time.Timer
}

// EventHandler is called when a matching file event occurs.
type EventHandler func(path string, rule Rule) error

// Status represents the current watcher status.
type Status struct {
	Running     bool     `json:"running"`
	Directories []string `json:"directories"`
	Rules       int      `json:"rules"`
	EventCount  int      `json:"eventCount"`
	StartedAt   string   `json:"startedAt,omitempty"`
}

// officeExtensions are the standard Office file extensions.
var officeExtensions = map[string]bool{
	".docx": true, ".xlsx": true, ".pptx": true,
	".doc": true, ".xls": true, ".ppt": true,
	".pdf": true, ".csv": true, ".json": true,
}

// New creates a new Watcher with the given configuration.
func New(config WatchConfig) (*Watcher, error) {
	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("could not create file watcher: %w", err)
	}

	if config.Debounce <= 0 {
		config.Debounce = 500
	}

	w := &Watcher{
		Config:   config,
		Logger:   log.New(os.Stderr, "[watch] ", log.LstdFlags),
		watcher:  fsw,
		debounce: make(map[string]*time.Timer),
	}

	return w, nil
}

// Start begins watching the configured directories. It blocks until the context is cancelled.
func (w *Watcher) Start(ctx context.Context) error {
	// Add directories
	for _, dir := range w.Config.Directories {
		absDir, err := filepath.Abs(dir)
		if err != nil {
			return fmt.Errorf("could not resolve %s: %w", dir, err)
		}

		if w.Config.Recursive {
			if err := w.addRecursive(absDir); err != nil {
				return err
			}
		} else {
			if err := w.watcher.Add(absDir); err != nil {
				return fmt.Errorf("could not watch %s: %w", absDir, err)
			}
		}
	}

	w.Logger.Printf("Watching %d directory(ies) with %d rule(s)", len(w.Config.Directories), len(w.Config.Rules))

	// Event loop
	for {
		select {
		case <-ctx.Done():
			w.Logger.Println("Stopping watcher")
			return w.watcher.Close()
		case event, ok := <-w.watcher.Events:
			if !ok {
				return nil
			}
			w.handleEvent(event)
		case err, ok := <-w.watcher.Errors:
			if !ok {
				return nil
			}
			w.Logger.Printf("Error: %v", err)
		}
	}
}

func (w *Watcher) addRecursive(dir string) error {
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}
		if info.IsDir() {
			// Skip hidden directories
			if strings.HasPrefix(filepath.Base(path), ".") && path != dir {
				return filepath.SkipDir
			}
			return w.watcher.Add(path)
		}
		return nil
	})
}

func (w *Watcher) handleEvent(event fsnotify.Event) {
	// Only process create and write events
	if !event.Has(fsnotify.Create) && !event.Has(fsnotify.Write) {
		return
	}

	path := event.Name
	ext := strings.ToLower(filepath.Ext(path))

	// Check if it's an office-type file
	if !officeExtensions[ext] {
		return
	}

	// Skip temp files
	base := filepath.Base(path)
	if strings.HasPrefix(base, "~$") || strings.HasPrefix(base, ".~") {
		return
	}

	// Debounce: wait before processing to avoid rapid fire
	w.mu.Lock()
	if timer, ok := w.debounce[path]; ok {
		timer.Stop()
	}
	w.debounce[path] = time.AfterFunc(time.Duration(w.Config.Debounce)*time.Millisecond, func() {
		w.processFile(path, event.Op.String())
	})
	w.mu.Unlock()
}

func (w *Watcher) processFile(path string, operation string) {
	// Find matching rule
	for _, rule := range w.Config.Rules {
		if !rule.Enabled {
			continue
		}
		if !w.matchesRule(path, rule) {
			continue
		}

		evt := Event{
			Time:      time.Now(),
			Path:      path,
			Operation: operation,
			RuleID:    rule.ID,
			Action:    rule.Action.Name,
		}

		if w.Handler != nil {
			if err := w.Handler(path, rule); err != nil {
				evt.Status = "error"
				evt.Error = err.Error()
				w.Logger.Printf("Error processing %s: %v", path, err)
			} else {
				evt.Status = "processed"
				w.Logger.Printf("Processed %s (rule: %s, action: %s)", path, rule.ID, rule.Action.Name)
			}
		} else {
			evt.Status = "processed"
			w.Logger.Printf("Matched %s (rule: %s, action: %s) [no handler]", path, rule.ID, rule.Action.Name)
		}

		w.mu.Lock()
		w.Events = append(w.Events, evt)
		w.mu.Unlock()
		return
	}

	// No rule matched — still log
	w.mu.Lock()
	w.Events = append(w.Events, Event{
		Time:      time.Now(),
		Path:      path,
		Operation: operation,
		Status:    "skipped",
	})
	w.mu.Unlock()
}

func (w *Watcher) matchesRule(path string, rule Rule) bool {
	ext := strings.ToLower(filepath.Ext(path))

	// Check extensions
	if len(rule.Extensions) > 0 {
		matched := false
		for _, e := range rule.Extensions {
			if !strings.HasPrefix(e, ".") {
				e = "." + e
			}
			if strings.ToLower(e) == ext {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	// Check glob pattern
	if rule.Pattern != "" {
		matched, _ := filepath.Match(rule.Pattern, filepath.Base(path))
		if !matched {
			return false
		}
	}

	return true
}

// GetStatus returns the current watcher status.
func (w *Watcher) GetStatus() Status {
	w.mu.Lock()
	defer w.mu.Unlock()
	return Status{
		Running:     true,
		Directories: w.Config.Directories,
		Rules:       len(w.Config.Rules),
		EventCount:  len(w.Events),
	}
}

// GetEvents returns all recorded events.
func (w *Watcher) GetEvents() []Event {
	w.mu.Lock()
	defer w.mu.Unlock()
	events := make([]Event, len(w.Events))
	copy(events, w.Events)
	return events
}

// Daemon manages a persistent watcher process with PID file tracking.

const pidFile = ".kit-watch.pid"

// WritePIDFile writes the current process ID to the PID file in the given directory.
func WritePIDFile(dir string) error {
	path := filepath.Join(dir, pidFile)
	return os.WriteFile(path, []byte(fmt.Sprintf("%d", os.Getpid())), 0644)
}

// ReadPIDFile reads the PID from the PID file.
func ReadPIDFile(dir string) (int, error) {
	path := filepath.Join(dir, pidFile)
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	var pid int
	if _, err := fmt.Sscanf(string(data), "%d", &pid); err != nil {
		return 0, fmt.Errorf("invalid PID file: %w", err)
	}
	return pid, nil
}

// RemovePIDFile removes the PID file.
func RemovePIDFile(dir string) error {
	return os.Remove(filepath.Join(dir, pidFile))
}

// SaveConfig writes the watcher config to a JSON file.
func SaveConfig(dir string, config WatchConfig) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "watch-config.json"), data, 0644)
}

// LoadConfig reads the watcher config from a JSON file.
func LoadConfig(dir string) (*WatchConfig, error) {
	data, err := os.ReadFile(filepath.Join(dir, "watch-config.json"))
	if err != nil {
		return nil, err
	}
	var config WatchConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("invalid watch config: %w", err)
	}
	return &config, nil
}

// DefaultConfigDir returns the default config directory for the watcher.
func DefaultConfigDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".kit")
}
