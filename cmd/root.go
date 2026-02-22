// Package cmd contains all CLI commands for the kit binary.
package cmd

import (
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/klytics/m365kit/cmd/ai"
	cmdauth "github.com/klytics/m365kit/cmd/auth"
	"github.com/klytics/m365kit/cmd/batch"
	"github.com/klytics/m365kit/cmd/completion"
	cmdconfig "github.com/klytics/m365kit/cmd/config"
	"github.com/klytics/m365kit/cmd/diff"
	"github.com/klytics/m365kit/cmd/doctor"
	"github.com/klytics/m365kit/cmd/excel"
	cmdffs "github.com/klytics/m365kit/cmd/fs"
	"github.com/klytics/m365kit/cmd/onedrive"
	"github.com/klytics/m365kit/cmd/pipeline"
	"github.com/klytics/m365kit/cmd/pptx"
	"github.com/klytics/m365kit/cmd/report"
	"github.com/klytics/m365kit/cmd/send"
	"github.com/klytics/m365kit/cmd/sharepoint"
	"github.com/klytics/m365kit/cmd/teams"
	cmdtemplate "github.com/klytics/m365kit/cmd/template"
	"github.com/klytics/m365kit/cmd/update"
	"github.com/klytics/m365kit/cmd/version"
	cmdwatch "github.com/klytics/m365kit/cmd/watch"
	"github.com/klytics/m365kit/cmd/word"
)

var (
	jsonOutput bool
	verbose    bool
	modelName  string
	provider   string
	noColor    bool
)

// NewRootCommand creates and returns the root cobra command with all subcommands registered.
func NewRootCommand() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "kit",
		Short: "AI-native CLI for Microsoft 365 documents",
		Long: `M365Kit — The terminal is the new Office.

A unified programmatic interface to every Microsoft 365 document format.
Read, write, analyze, transform, and automate .docx .xlsx .pptx from your terminal.`,
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			if noColor {
				color.NoColor = true
			}
		},
	}

	// Global persistent flags
	rootCmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "Output as machine-readable JSON")
	rootCmd.PersistentFlags().BoolVar(&verbose, "verbose", false, "Enable debug logging")
	rootCmd.PersistentFlags().StringVar(&modelName, "model", defaultModel(), "AI model name override")
	rootCmd.PersistentFlags().StringVar(&provider, "provider", defaultProvider(), "AI provider: anthropic | openai | ollama")
	rootCmd.PersistentFlags().BoolVar(&noColor, "no-color", false, "Disable ANSI color output")

	// Register subcommands
	rootCmd.AddCommand(word.NewCommand())
	rootCmd.AddCommand(excel.NewCommand())
	rootCmd.AddCommand(pptx.NewCommand())
	rootCmd.AddCommand(ai.NewCommand())
	rootCmd.AddCommand(cmdauth.NewCommand())
	rootCmd.AddCommand(pipeline.NewCommand())
	rootCmd.AddCommand(batch.NewCommand())
	rootCmd.AddCommand(cmdconfig.NewCommand())
	rootCmd.AddCommand(diff.NewCommand())
	rootCmd.AddCommand(doctor.NewCommand())
	rootCmd.AddCommand(cmdffs.NewCommand())
	rootCmd.AddCommand(send.NewCommand())
	rootCmd.AddCommand(onedrive.NewCommand())
	rootCmd.AddCommand(sharepoint.NewCommand())
	rootCmd.AddCommand(teams.NewCommand())
	rootCmd.AddCommand(cmdtemplate.NewCommand())
	rootCmd.AddCommand(report.NewCommand())
	rootCmd.AddCommand(update.NewCommand())
	rootCmd.AddCommand(cmdwatch.NewCommand())
	rootCmd.AddCommand(completion.NewCommand(rootCmd))
	rootCmd.AddCommand(version.NewCommand())

	return rootCmd
}

// Execute runs the root command and handles any returned errors.
func Execute() {
	rootCmd := NewRootCommand()
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}
}

func defaultModel() string {
	if m := os.Getenv("KIT_MODEL"); m != "" {
		return m
	}
	return "claude-sonnet-4-20250514"
}

func defaultProvider() string {
	if p := os.Getenv("KIT_PROVIDER"); p != "" {
		return p
	}
	return "anthropic"
}
