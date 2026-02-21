// Package pptx provides CLI commands for working with .pptx files.
package pptx

import "github.com/spf13/cobra"

// NewCommand returns the pptx subcommand group.
func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pptx",
		Short: "Read and generate PowerPoint presentations (.pptx)",
		Long:  "Commands for working with Microsoft PowerPoint .pptx files — extract slide content and generate presentations from data.",
	}

	cmd.AddCommand(newReadCommand())
	cmd.AddCommand(newGenerateCommand())

	return cmd
}
