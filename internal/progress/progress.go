// Package progress provides terminal progress bars and spinners.
// All output goes to stderr to avoid polluting stdout/pipes.
package progress

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"
)

// Bar renders an ASCII progress bar to stderr.
type Bar struct {
	Total   int
	Current int
	Label   string
	Width   int
	Enabled bool

	mu sync.Mutex
}

// New creates a progress bar.
// Automatically disabled if not a TTY, if --json is set, or KIT_NO_PROGRESS=1.
func New(label string, total int) *Bar {
	return &Bar{
		Total:   total,
		Label:   label,
		Width:   40,
		Enabled: shouldEnable(),
	}
}

// Increment advances the bar by 1 and redraws.
func (b *Bar) Increment(status string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.Current++
	if b.Current > b.Total {
		b.Current = b.Total
	}
	b.render(status)
}

// Set sets the bar to a specific value.
func (b *Bar) Set(n int, status string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.Current = n
	if b.Current > b.Total {
		b.Current = b.Total
	}
	b.render(status)
}

// Finish prints a final completion line.
func (b *Bar) Finish(summary string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if !b.Enabled {
		return
	}
	// Clear the line and print summary
	fmt.Fprintf(os.Stderr, "\r\033[K✓ %s\n", summary)
}

func (b *Bar) render(status string) {
	if !b.Enabled {
		return
	}

	pct := 0.0
	if b.Total > 0 {
		pct = float64(b.Current) / float64(b.Total)
	}

	filled := int(pct * float64(b.Width))
	if filled > b.Width {
		filled = b.Width
	}

	bar := strings.Repeat("=", filled) + strings.Repeat(" ", b.Width-filled)
	fmt.Fprintf(os.Stderr, "\r\033[K%s [%s] %d/%d  %s",
		b.Label, bar, b.Current, b.Total, status)
}

// Pct returns the current percentage (0-100) of the bar.
func (b *Bar) Pct() float64 {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.Total == 0 {
		return 0
	}
	return float64(b.Current) / float64(b.Total) * 100
}

// Spinner shows a spinner for operations where total is unknown.
type Spinner struct {
	Label   string
	Enabled bool

	mu      sync.Mutex
	done    chan struct{}
	stopped bool
}

// NewSpinner creates a spinner.
func NewSpinner(label string) *Spinner {
	return &Spinner{
		Label:   label,
		Enabled: shouldEnable(),
		done:    make(chan struct{}),
	}
}

// Start begins the spinner animation.
func (s *Spinner) Start() {
	if !s.Enabled {
		return
	}

	s.mu.Lock()
	s.stopped = false
	s.done = make(chan struct{})
	s.mu.Unlock()

	go func() {
		frames := []rune{'⠋', '⠙', '⠹', '⠸', '⠼', '⠴', '⠦', '⠧', '⠇', '⠏'}
		i := 0
		ticker := time.NewTicker(80 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-s.done:
				return
			case <-ticker.C:
				s.mu.Lock()
				if !s.stopped {
					fmt.Fprintf(os.Stderr, "\r\033[K%c %s", frames[i%len(frames)], s.Label)
					i++
				}
				s.mu.Unlock()
			}
		}
	}()
}

// Stop stops the spinner and prints a result.
func (s *Spinner) Stop(result string) {
	s.mu.Lock()
	s.stopped = true
	s.mu.Unlock()

	select {
	case <-s.done:
	default:
		close(s.done)
	}

	if s.Enabled {
		fmt.Fprintf(os.Stderr, "\r\033[K✓ %s\n", result)
	}
}

// Update changes the spinner label while it's running.
func (s *Spinner) Update(label string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Label = label
}

func shouldEnable() bool {
	// Disabled via env var
	if os.Getenv("KIT_NO_PROGRESS") == "1" {
		return false
	}
	// Disabled when JSON output is requested
	if os.Getenv("KIT_JSON") == "true" {
		return false
	}
	// Check if stderr is a TTY
	return isTTY()
}

func isTTY() bool {
	stat, err := os.Stderr.Stat()
	if err != nil {
		return false
	}
	return (stat.Mode() & os.ModeCharDevice) != 0
}
