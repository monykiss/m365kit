package pipeline

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"
)

func TestInterpolateDateToday(t *testing.T) {
	e := NewExecutor(false)
	result := e.interpolate("Today is ${{ date.today }}")
	today := time.Now().Format("2006-01-02")

	if !strings.Contains(result, today) {
		t.Errorf("expected today's date %q in result %q", today, result)
	}
}

func TestInterpolateDateTimestamp(t *testing.T) {
	e := NewExecutor(false)
	result := e.interpolate("Now: ${{ date.timestamp }}")

	if strings.Contains(result, "${{") {
		t.Errorf("timestamp was not interpolated: %q", result)
	}

	// Should contain a year
	year := time.Now().Format("2006")
	if !strings.Contains(result, year) {
		t.Errorf("expected year %q in result %q", year, result)
	}
}

func TestInterpolateEnvVar(t *testing.T) {
	os.Setenv("KIT_TEST_VAR", "hello_world")
	defer os.Unsetenv("KIT_TEST_VAR")

	e := NewExecutor(false)
	result := e.interpolate("Value: ${{ env.KIT_TEST_VAR }}")

	if !strings.Contains(result, "hello_world") {
		t.Errorf("expected 'hello_world' in result %q", result)
	}
}

func TestInterpolateEnvVarEmpty(t *testing.T) {
	os.Unsetenv("KIT_MISSING_VAR")

	e := NewExecutor(false)
	result := e.interpolate("Value: ${{ env.KIT_MISSING_VAR }}")

	// Should resolve to empty string
	if strings.Contains(result, "${{") {
		t.Errorf("env var was not interpolated: %q", result)
	}
	if !strings.Contains(result, "Value: ") {
		t.Errorf("unexpected result: %q", result)
	}
}

func TestStepOutputFlowsToNextStep(t *testing.T) {
	e := NewExecutor(false)

	// Register two actions: step1 produces output, step2 uses it
	e.RegisterAction("produce", func(ctx context.Context, step Step, input string) (string, error) {
		return "produced_data", nil
	})
	e.RegisterAction("consume", func(ctx context.Context, step Step, input string) (string, error) {
		return "received:" + input, nil
	})

	p := &Pipeline{
		Name:    "test",
		Version: "1.0",
		Steps: []Step{
			{ID: "step1", Action: "produce", Input: "initial"},
			{ID: "step2", Action: "consume", Input: "${{ steps.step1.output }}"},
		},
	}

	results, err := e.Run(context.Background(), p)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	if results[0].Output != "produced_data" {
		t.Errorf("step1: expected 'produced_data', got %q", results[0].Output)
	}

	if results[1].Output != "received:produced_data" {
		t.Errorf("step2: expected 'received:produced_data', got %q", results[1].Output)
	}
}

func TestUnknownActionReturnsError(t *testing.T) {
	e := NewExecutor(false)

	p := &Pipeline{
		Name:    "test",
		Version: "1.0",
		Steps: []Step{
			{ID: "bad_step", Action: "nonexistent.action"},
		},
	}

	_, err := e.Run(context.Background(), p)
	if err == nil {
		t.Fatal("expected error for unknown action")
	}

	if !strings.Contains(err.Error(), "unknown action") {
		t.Errorf("expected 'unknown action' in error, got: %s", err.Error())
	}

	if !strings.Contains(err.Error(), "nonexistent.action") {
		t.Errorf("expected action name in error, got: %s", err.Error())
	}
}

func TestUnknownActionSkipOnFailure(t *testing.T) {
	e := NewExecutor(false)
	e.RegisterAction("ok_action", func(ctx context.Context, step Step, input string) (string, error) {
		return "ok", nil
	})

	p := &Pipeline{
		Name:    "test",
		Version: "1.0",
		Steps: []Step{
			{ID: "skip_me", Action: "nonexistent", OnFailure: "skip"},
			{ID: "after_skip", Action: "ok_action"},
		},
	}

	results, err := e.Run(context.Background(), p)
	if err != nil {
		t.Fatalf("Run should not fail with on_failure=skip: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	if results[0].Error == nil {
		t.Error("first step should have an error")
	}

	if results[1].Output != "ok" {
		t.Errorf("second step should have run, got output %q", results[1].Output)
	}
}

func TestDryRunSkipsAISteps(t *testing.T) {
	e := NewExecutor(false)
	e.SetDryRun(true)

	called := false
	e.RegisterAction("ai.summarize", func(ctx context.Context, step Step, input string) (string, error) {
		called = true
		return "summary", nil
	})
	e.RegisterAction("word.read", func(ctx context.Context, step Step, input string) (string, error) {
		return "document text", nil
	})

	p := &Pipeline{
		Name:    "test",
		Version: "1.0",
		Steps: []Step{
			{ID: "read", Action: "word.read", Input: "test.docx"},
			{ID: "summarize", Action: "ai.summarize", Input: "${{ steps.read.output }}"},
		},
	}

	results, err := e.Run(context.Background(), p)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if called {
		t.Error("ai.summarize action should NOT have been called in dry-run mode")
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	if !strings.Contains(results[1].Output, "DRY-RUN") {
		t.Errorf("expected DRY-RUN in output, got %q", results[1].Output)
	}
}
