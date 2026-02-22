// Package cmd contains all CLI commands for the kit binary.
package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	auditpkg "github.com/klytics/m365kit/internal/audit"
	"github.com/klytics/m365kit/internal/config"
	shellpkg "github.com/klytics/m365kit/internal/shell"

	"github.com/klytics/m365kit/cmd/acl"
	cmdadmin "github.com/klytics/m365kit/cmd/admin"
	"github.com/klytics/m365kit/cmd/ai"
	cmdaudit "github.com/klytics/m365kit/cmd/audit"
	cmdauth "github.com/klytics/m365kit/cmd/auth"
	"github.com/klytics/m365kit/cmd/batch"
	"github.com/klytics/m365kit/cmd/completion"
	cmdconfig "github.com/klytics/m365kit/cmd/config"
	cmdconvert "github.com/klytics/m365kit/cmd/convert"
	"github.com/klytics/m365kit/cmd/diff"
	"github.com/klytics/m365kit/cmd/doctor"
	"github.com/klytics/m365kit/cmd/excel"
	cmdffs "github.com/klytics/m365kit/cmd/fs"
	"github.com/klytics/m365kit/cmd/onedrive"
	cmdorg "github.com/klytics/m365kit/cmd/org"
	"github.com/klytics/m365kit/cmd/outlook"
	"github.com/klytics/m365kit/cmd/pipeline"
	cmdplugin "github.com/klytics/m365kit/cmd/plugin"
	"github.com/klytics/m365kit/cmd/pptx"
	"github.com/klytics/m365kit/cmd/report"
	"github.com/klytics/m365kit/cmd/send"
	cmdshell "github.com/klytics/m365kit/cmd/shell"
	"github.com/klytics/m365kit/cmd/sharepoint"
	"github.com/klytics/m365kit/cmd/teams"
	cmdtemplate "github.com/klytics/m365kit/cmd/template"
	"github.com/klytics/m365kit/cmd/update"
	"github.com/klytics/m365kit/cmd/version"
	cmdwatch "github.com/klytics/m365kit/cmd/watch"
	"github.com/klytics/m365kit/cmd/word"
)

// Exit codes for consistent error reporting.
const (
	ExitOK          = 0 // success
	ExitUserError   = 1 // bad flags, missing file, auth required
	ExitSystemError = 2 // network failure, IO error, API error
)

var (
	jsonOutput bool
	verbose    bool
	modelName  string
	provider   string
	noColor    bool
	noProgress bool
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
	rootCmd.PersistentFlags().BoolVar(&noProgress, "no-progress", false, "Disable progress bars")

	// Register subcommands
	rootCmd.AddCommand(word.NewCommand())
	rootCmd.AddCommand(excel.NewCommand())
	rootCmd.AddCommand(pptx.NewCommand())
	rootCmd.AddCommand(ai.NewCommand())
	rootCmd.AddCommand(cmdauth.NewCommand())
	rootCmd.AddCommand(pipeline.NewCommand())
	rootCmd.AddCommand(batch.NewCommand())
	rootCmd.AddCommand(cmdconfig.NewCommand())
	rootCmd.AddCommand(cmdconvert.NewCommand())
	rootCmd.AddCommand(diff.NewCommand())
	rootCmd.AddCommand(doctor.NewCommand())
	rootCmd.AddCommand(cmdffs.NewCommand())
	rootCmd.AddCommand(send.NewCommand())
	rootCmd.AddCommand(onedrive.NewCommand())
	rootCmd.AddCommand(outlook.NewCommand())
	rootCmd.AddCommand(sharepoint.NewCommand())
	rootCmd.AddCommand(acl.NewCommand())
	rootCmd.AddCommand(teams.NewCommand())
	rootCmd.AddCommand(cmdtemplate.NewCommand())
	rootCmd.AddCommand(report.NewCommand())
	rootCmd.AddCommand(update.NewCommand())
	rootCmd.AddCommand(cmdwatch.NewCommand())
	rootCmd.AddCommand(completion.NewCommand(rootCmd))
	rootCmd.AddCommand(version.NewCommand())

	// Enterprise commands (v1.1)
	rootCmd.AddCommand(cmdorg.NewCommand())
	rootCmd.AddCommand(cmdaudit.NewCommand())
	rootCmd.AddCommand(cmdadmin.NewCommand())

	// Platform commands (v1.2)
	rootCmd.AddCommand(cmdplugin.NewCommand())
	rootCmd.AddCommand(cmdshell.NewCommand())

	// Wire shell runner: the shell REPL creates a fresh root command per eval
	shellpkg.DefaultRunner = func(ctx context.Context, args []string, stdout, stderr io.Writer) error {
		inner := NewRootCommand()
		inner.SetOut(stdout)
		inner.SetErr(stderr)
		inner.SetArgs(args)
		return inner.ExecuteContext(ctx)
	}

	// Audit logging: wrap PersistentPreRun to capture start time
	origPreRun := rootCmd.PersistentPreRun
	rootCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		if origPreRun != nil {
			origPreRun(cmd, args)
		}
		if noProgress {
			os.Setenv("KIT_NO_PROGRESS", "1")
		}
		cmd.SetContext(context.WithValue(cmd.Context(), auditStartKey, time.Now()))
	}

	rootCmd.PersistentPostRun = func(cmd *cobra.Command, args []string) {
		orgCfg, _ := config.LoadOrgConfig()
		if orgCfg == nil || !orgCfg.Audit.Enabled {
			return
		}
		logger := auditpkg.NewLogger(orgCfg.AuditLogPath(), orgCfg.Audit.Endpoint, orgCfg.Audit.Level, true)
		start, _ := cmd.Context().Value(auditStartKey).(time.Time)
		entry := auditpkg.Entry{
			Timestamp:  start,
			Command:    cmd.CommandPath(),
			Args:       auditpkg.Redact(args),
			DurationMs: time.Since(start).Milliseconds(),
			UserID:     os.Getenv("KIT_USER"),
			Machine:    hostname(),
		}
		_ = logger.Log(cmd.Context(), entry)
	}

	return rootCmd
}

type contextKey string

const auditStartKey contextKey = "audit_start"

func hostname() string {
	h, _ := os.Hostname()
	parts := strings.SplitN(h, ".", 2)
	return parts[0]
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
