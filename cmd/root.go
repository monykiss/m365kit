// Package cmd contains all CLI commands for the kit binary.
package cmd

import (
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/klytics/m365kit/cmd/ai"
	"github.com/klytics/m365kit/cmd/batch"
	"github.com/klytics/m365kit/cmd/diff"
	"github.com/klytics/m365kit/cmd/excel"
	"github.com/klytics/m365kit/cmd/pipeline"
	"github.com/klytics/m365kit/cmd/pptx"
	"github.com/klytics/m365kit/cmd/send"
	"github.com/klytics/m365kit/cmd/version"
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
	rootCmd.AddCommand(pipeline.NewCommand())
	rootCmd.AddCommand(batch.NewCommand())
	rootCmd.AddCommand(diff.NewCommand())
	rootCmd.AddCommand(send.NewCommand())
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
