package pipeline

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	pipelinepkg "github.com/klytics/m365kit/internal/pipeline"
	"github.com/klytics/m365kit/internal/pipeline/actions"
)

func newRunCommand() *cobra.Command {
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "run <workflow.yaml>",
		Short: "Execute a pipeline workflow from a YAML file",
		Long: `Runs a multi-step pipeline defined in a YAML file.

Steps are executed sequentially with variable interpolation between steps.
Use --dry-run to execute non-AI steps and preview what AI steps would do.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			jsonFlag, _ := cmd.Flags().GetBool("json")
			verbose, _ := cmd.Flags().GetBool("verbose")

			p, err := pipelinepkg.LoadPipeline(args[0])
			if err != nil {
				return err
			}

			executor := pipelinepkg.NewExecutor(verbose)
			executor.SetDryRun(dryRun)
			actions.RegisterAll(executor)

			ctx := context.Background()
			results, execErr := executor.Run(ctx, p)

			if jsonFlag {
				// Build JSON-safe output (errors don't serialize well)
				type jsonResult struct {
					StepID string `json:"stepId"`
					Output string `json:"output,omitempty"`
					Error  string `json:"error,omitempty"`
				}
				out := make([]jsonResult, len(results))
				for i, r := range results {
					out[i] = jsonResult{StepID: r.StepID, Output: r.Output}
					if r.Error != nil {
						out[i].Error = r.Error.Error()
					}
				}
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				_ = enc.Encode(out)
			} else {
				for _, r := range results {
					if r.Error != nil {
						fmt.Fprintf(os.Stderr, "Step %s: FAILED — %s\n", r.StepID, r.Error)
					} else {
						fmt.Printf("Step %s: OK\n", r.StepID)
						if verbose && r.Output != "" {
							fmt.Printf("  Output: %s\n", truncate(r.Output, 200))
						}
					}
				}
			}

			return execErr
		},
	}

	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview pipeline execution without calling AI APIs")

	return cmd
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
