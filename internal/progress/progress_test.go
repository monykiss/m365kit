package progress

import (
	"os"
	"testing"
	"time"
)

func TestNewDisabledInNonTTY(t *testing.T) {
	// In tests, stderr is typically not a TTY
	bar := New("test", 10)
	if bar.Enabled {
		t.Skip("TTY detected, skipping non-TTY test")
	}
	// When not a TTY, bar should be disabled
	if bar.Enabled {
		t.Error("expected bar to be disabled in non-TTY")
	}
}

func TestNewWithEnvDisable(t *testing.T) {
	t.Setenv("KIT_NO_PROGRESS", "1")
	bar := New("test", 10)
	if bar.Enabled {
		t.Error("expected bar to be disabled with KIT_NO_PROGRESS=1")
	}
}

func TestNewWithJSONDisable(t *testing.T) {
	t.Setenv("KIT_JSON", "true")
	bar := New("test", 10)
	if bar.Enabled {
		t.Error("expected bar to be disabled with KIT_JSON=true")
	}
}

func TestBarIncrement(t *testing.T) {
	bar := &Bar{Total: 10, Width: 40, Enabled: false}
	bar.Increment("test")
	if bar.Current != 1 {
		t.Errorf("expected current=1, got %d", bar.Current)
	}
	bar.Increment("test2")
	if bar.Current != 2 {
		t.Errorf("expected current=2, got %d", bar.Current)
	}
}

func TestBarOverIncrement(t *testing.T) {
	bar := &Bar{Total: 3, Width: 40, Enabled: false}
	bar.Increment("a")
	bar.Increment("b")
	bar.Increment("c")
	bar.Increment("d") // Over-increment
	bar.Increment("e") // Over-increment
	if bar.Current != 3 {
		t.Errorf("expected current capped at 3, got %d", bar.Current)
	}
}

func TestBarSet(t *testing.T) {
	bar := &Bar{Total: 100, Width: 40, Enabled: false}
	bar.Set(50, "halfway")
	if bar.Current != 50 {
		t.Errorf("expected current=50, got %d", bar.Current)
	}
}

func TestBarSetOverflow(t *testing.T) {
	bar := &Bar{Total: 10, Width: 40, Enabled: false}
	bar.Set(999, "overflow")
	if bar.Current != 10 {
		t.Errorf("expected current capped at 10, got %d", bar.Current)
	}
}

func TestBarPctZero(t *testing.T) {
	bar := &Bar{Total: 10, Width: 40, Enabled: false}
	pct := bar.Pct()
	if pct != 0 {
		t.Errorf("expected 0%%, got %.1f%%", pct)
	}
}

func TestBarPctFifty(t *testing.T) {
	bar := &Bar{Total: 10, Current: 5, Width: 40, Enabled: false}
	pct := bar.Pct()
	if pct != 50.0 {
		t.Errorf("expected 50%%, got %.1f%%", pct)
	}
}

func TestBarPctHundred(t *testing.T) {
	bar := &Bar{Total: 10, Current: 10, Width: 40, Enabled: false}
	pct := bar.Pct()
	if pct != 100.0 {
		t.Errorf("expected 100%%, got %.1f%%", pct)
	}
}

func TestBarPctZeroTotal(t *testing.T) {
	bar := &Bar{Total: 0, Width: 40, Enabled: false}
	pct := bar.Pct()
	if pct != 0 {
		t.Errorf("expected 0%% for zero total, got %.1f%%", pct)
	}
}

func TestBarFinishDisabled(t *testing.T) {
	bar := &Bar{Total: 10, Width: 40, Enabled: false}
	// Should not panic
	bar.Finish("done")
}

func TestSpinnerStartStopDisabled(t *testing.T) {
	s := &Spinner{Label: "test", Enabled: false, done: make(chan struct{})}
	// Start and stop should not panic when disabled
	s.Start()
	s.Stop("done")
}

func TestSpinnerStartStop(t *testing.T) {
	s := &Spinner{Label: "test", Enabled: true, done: make(chan struct{})}
	s.Start()
	time.Sleep(100 * time.Millisecond) // Let a few frames render
	s.Stop("complete")
	// If we get here without deadlock, test passes
}

func TestSpinnerUpdate(t *testing.T) {
	s := &Spinner{Label: "initial", Enabled: false, done: make(chan struct{})}
	s.Update("updated")
	if s.Label != "updated" {
		t.Errorf("expected label 'updated', got %q", s.Label)
	}
}

func TestDisabledBarDoesNotWrite(t *testing.T) {
	// Redirect stderr to verify no output
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	bar := &Bar{Total: 10, Width: 40, Enabled: false}
	bar.Increment("test")
	bar.Finish("done")

	w.Close()
	os.Stderr = oldStderr

	buf := make([]byte, 1024)
	n, _ := r.Read(buf)
	if n > 0 {
		t.Errorf("disabled bar should not write to stderr, wrote %d bytes", n)
	}
}

func TestNewSpinnerDisabled(t *testing.T) {
	t.Setenv("KIT_NO_PROGRESS", "1")
	s := NewSpinner("test")
	if s.Enabled {
		t.Error("expected spinner to be disabled")
	}
}
