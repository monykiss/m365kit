// Package ai provides CLI commands for AI-powered document analysis.
package ai

import "github.com/spf13/cobra"

// NewCommand returns the ai subcommand group.
func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ai",
		Short: "AI-powered document analysis and transformation",
		Long:  "Commands that use AI models to summarize, analyze, extract entities, and answer questions about documents.",
	}

	cmd.AddCommand(newSummarizeCommand())
	cmd.AddCommand(newAnalyzeCommand())
	cmd.AddCommand(newExtractCommand())
	cmd.AddCommand(newAskCommand())

	return cmd
}
