// Package excel provides CLI commands for working with .xlsx files.
package excel

import "github.com/spf13/cobra"

// NewCommand returns the excel subcommand group.
func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "excel",
		Short: "Read, write, and analyze Excel spreadsheets (.xlsx)",
		Long:  "Commands for working with Microsoft Excel .xlsx files — extract data, generate spreadsheets, and run AI-powered analysis.",
	}

	cmd.AddCommand(newReadCommand())
	cmd.AddCommand(newWriteCommand())
	cmd.AddCommand(newAnalyzeCommand())

	return cmd
}
