package pipeline

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"
)

// ActionFunc is the signature for pipeline action handlers.
type ActionFunc func(ctx context.Context, step Step, input string) (string, error)

// Executor runs pipeline steps sequentially, resolving variable interpolation between steps.
type Executor struct {
	actions map[string]ActionFunc
	results map[string]*StepResult
	verbose bool
	dryRun  bool
}

// NewExecutor creates a new pipeline executor with the given options.
func NewExecutor(verbose bool) *Executor {
	return &Executor{
		actions: make(map[string]ActionFunc),
		results: make(map[string]*StepResult),
		verbose: verbose,
	}
}

// SetDryRun enables dry-run mode. Non-AI steps execute normally; AI steps are skipped
// with a description of what they would do.
func (e *Executor) SetDryRun(dryRun bool) {
	e.dryRun = dryRun
}

// RegisterAction adds an action handler to the executor's registry.
func (e *Executor) RegisterAction(name string, fn ActionFunc) {
	e.actions[name] = fn
}

// Run executes all steps in the pipeline sequentially.
func (e *Executor) Run(ctx context.Context, p *Pipeline) ([]StepResult, error) {
	var results []StepResult

	if e.verbose {
		fmt.Printf("Running pipeline: %s (v%s)\n", p.Name, p.Version)
		if e.dryRun {
			fmt.Println("  (dry-run mode — AI steps will be skipped)")
		}
	}

	for i, step := range p.Steps {
		if e.verbose {
			fmt.Printf("[%d/%d] Running step: %s (%s)\n", i+1, len(p.Steps), step.ID, step.Action)
		}

		// Resolve variable interpolation in all string fields
		resolvedStep := e.resolveStepVariables(step)

		// In dry-run mode, skip AI steps
		if e.dryRun && isAIAction(resolvedStep.Action) {
			input := resolvedStep.Input
			if input == "" {
				input = resolvedStep.Data
			}
			preview := truncateStr(input, 100)
			msg := fmt.Sprintf("[DRY-RUN] Would call %s with %d chars of input", resolvedStep.Action, len(input))
			if e.verbose {
				fmt.Printf("  %s\n", msg)
				if preview != "" {
					fmt.Printf("  Input preview: %s\n", preview)
				}
				if opts := resolvedStep.Options; len(opts) > 0 {
					fmt.Printf("  Options: %v\n", opts)
				}
			}
			result := StepResult{StepID: resolvedStep.ID, Output: msg}
			results = append(results, result)
			e.results[resolvedStep.ID] = &result
			continue
		}

		// Look up action handler
		action, ok := e.actions[resolvedStep.Action]
		if !ok {
			err := fmt.Errorf("unknown action %q in step %q — registered actions: %v",
				resolvedStep.Action, resolvedStep.ID, e.actionNames())

			if resolvedStep.OnFailure == "skip" {
				if e.verbose {
					fmt.Printf("  Skipping step %s: %s\n", resolvedStep.ID, err)
				}
				result := StepResult{StepID: resolvedStep.ID, Error: err}
				results = append(results, result)
				e.results[resolvedStep.ID] = &result
				continue
			}
			return results, err
		}

		// Determine input
		input := resolvedStep.Input
		if input == "" && resolvedStep.Data != "" {
			input = resolvedStep.Data
		}

		// Execute the action
		start := time.Now()
		output, err := action(ctx, resolvedStep, input)
		duration := time.Since(start)

		result := StepResult{
			StepID: resolvedStep.ID,
			Output: output,
			Error:  err,
		}
		results = append(results, result)
		e.results[resolvedStep.ID] = &result

		if e.verbose {
			fmt.Printf("  Completed in %s\n", duration.Round(time.Millisecond))
		}

		if err != nil {
			if resolvedStep.OnFailure == "skip" {
				if e.verbose {
					fmt.Printf("  Step %s failed (skipping): %s\n", resolvedStep.ID, err)
				}
				continue
			}
			return results, fmt.Errorf("step %q failed: %w", resolvedStep.ID, err)
		}
	}

	return results, nil
}

func isAIAction(action string) bool {
	return strings.HasPrefix(action, "ai.")
}

var interpolationPattern = regexp.MustCompile(`\$\{\{\s*([^}]+)\s*\}\}`)

func (e *Executor) resolveStepVariables(step Step) Step {
	resolved := step
	resolved.Input = e.interpolate(step.Input)
	resolved.Template = e.interpolate(step.Template)
	resolved.Data = e.interpolate(step.Data)
	resolved.To = e.interpolate(step.To)
	resolved.Subject = e.interpolate(step.Subject)
	resolved.Attach = e.interpolate(step.Attach)

	if resolved.Options != nil {
		newOpts := make(map[string]string, len(resolved.Options))
		for k, v := range resolved.Options {
			newOpts[k] = e.interpolate(v)
		}
		resolved.Options = newOpts
	}

	return resolved
}

func (e *Executor) interpolate(s string) string {
	return interpolationPattern.ReplaceAllStringFunc(s, func(match string) string {
		inner := interpolationPattern.FindStringSubmatch(match)
		if len(inner) < 2 {
			return match
		}
		expr := strings.TrimSpace(inner[1])

		// Handle steps.<id>.output
		if strings.HasPrefix(expr, "steps.") {
			parts := strings.Split(expr, ".")
			if len(parts) >= 3 && parts[2] == "output" {
				stepID := parts[1]
				if result, ok := e.results[stepID]; ok {
					return result.Output
				}
			}
		}

		// Handle date.today
		if expr == "date.today" {
			return time.Now().Format("2006-01-02")
		}

		// Handle date.now or date.timestamp
		if expr == "date.now" || expr == "date.timestamp" {
			return time.Now().Format(time.RFC3339)
		}

		// Handle env.VAR_NAME
		if strings.HasPrefix(expr, "env.") {
			varName := strings.TrimPrefix(expr, "env.")
			return os.Getenv(varName)
		}

		return match
	})
}

func (e *Executor) actionNames() []string {
	names := make([]string, 0, len(e.actions))
	for name := range e.actions {
		names = append(names, name)
	}
	return names
}

func truncateStr(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
