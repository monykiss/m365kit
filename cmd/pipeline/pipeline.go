// Package pipeline provides CLI commands for running pipeline workflows.
package pipeline

import "github.com/spf13/cobra"

// NewCommand returns the pipeline subcommand group.
func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pipeline",
		Short: "Run multi-step document workflows defined in YAML",
		Long:  "Execute automated pipelines that chain together read, write, analyze, and transform operations across document formats.",
	}

	cmd.AddCommand(newRunCommand())

	return cmd
}
