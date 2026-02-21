// Package word provides CLI commands for working with .docx files.
package word

import "github.com/spf13/cobra"

// NewCommand returns the word subcommand group.
func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "word",
		Short: "Read, write, and transform Word documents (.docx)",
		Long:  "Commands for working with Microsoft Word .docx files — extract text, generate documents, find-and-replace, and AI-powered analysis.",
	}

	cmd.AddCommand(newReadCommand())
	cmd.AddCommand(newWriteCommand())
	cmd.AddCommand(newEditCommand())
	cmd.AddCommand(newSummarizeCommand())

	return cmd
}
