// Package audit provides the "kit audit" CLI commands for viewing audit logs.
package audit

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	auditpkg "github.com/klytics/m365kit/internal/audit"
	"github.com/klytics/m365kit/internal/config"
)

// NewCommand creates the "audit" command with all subcommands.
func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "audit",
		Short: "View and manage audit logs",
		Long:  "View command audit logs, export to CSV, and manage log files.",
	}

	cmd.AddCommand(newLogCmd())
	cmd.AddCommand(newClearCmd())
	cmd.AddCommand(newStatusCmd())

	return cmd
}

func auditLogPath() string {
	orgCfg, _ := config.LoadOrgConfig()
	if orgCfg != nil {
		return orgCfg.AuditLogPath()
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".kit", "audit.log")
}

func newLogCmd() *cobra.Command {
	var (
		last    int
		command string
		since   string
		userID  string
	)

	cmd := &cobra.Command{
		Use:   "log",
		Short: "Show recent audit log entries",
		RunE: func(cmd *cobra.Command, args []string) error {
			path := auditLogPath()
			entries, err := auditpkg.ReadEntries(path)
			if err != nil {
				return err
			}

			var sinceTime, untilTime time.Time
			if since != "" {
				t, err := time.Parse("2006-01-02", since)
				if err != nil {
					return fmt.Errorf("invalid --since date: %w (use YYYY-MM-DD)", err)
				}
				sinceTime = t
			}

			filtered := auditpkg.FilterEntries(entries, sinceTime, untilTime, command, userID)

			if last > 0 && len(filtered) > last {
				filtered = filtered[len(filtered)-last:]
			}

			jsonOut, _ := cmd.Flags().GetBool("json")
			if jsonOut {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(filtered)
			}

			if len(filtered) == 0 {
				fmt.Println("No audit log entries found.")
				return nil
			}

			fmt.Printf("Audit Log — %d Entries\n", len(filtered))
			fmt.Printf("File: %s\n\n", path)

			tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintf(tw, "TIMESTAMP\tUSER\tCOMMAND\tDURATION\tEXIT\n")
			for _, e := range filtered {
				ts := e.Timestamp.Format("2006-01-02 15:04:05")
				dur := fmt.Sprintf("%dms", e.DurationMs)
				if e.DurationMs >= 1000 {
					dur = fmt.Sprintf("%.1fs", float64(e.DurationMs)/1000)
				}
				user := e.UserID
				if user == "" {
					user = "-"
				}
				fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%d\n", ts, user, e.Command, dur, e.ExitCode)
			}
			tw.Flush()
			return nil
		},
	}

	cmd.Flags().IntVar(&last, "last", 20, "Show last N entries")
	cmd.Flags().StringVar(&command, "command", "", "Filter by command name")
	cmd.Flags().StringVar(&since, "since", "", "Filter entries since date (YYYY-MM-DD)")
	cmd.Flags().StringVar(&userID, "user", "", "Filter by user email")
	return cmd
}

func newClearCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "clear",
		Short: "Clear the audit log",
		RunE: func(cmd *cobra.Command, args []string) error {
			path := auditLogPath()
			if err := auditpkg.Clear(path); err != nil {
				return err
			}
			jsonOut, _ := cmd.Flags().GetBool("json")
			if jsonOut {
				return json.NewEncoder(os.Stdout).Encode(map[string]string{"cleared": path})
			}
			fmt.Printf("Audit log cleared: %s\n", path)
			return nil
		},
	}
}

func newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show audit log path and size",
		RunE: func(cmd *cobra.Command, args []string) error {
			path := auditLogPath()
			size := auditpkg.LogSize(path)

			jsonOut, _ := cmd.Flags().GetBool("json")
			if jsonOut {
				return json.NewEncoder(os.Stdout).Encode(map[string]interface{}{
					"path": path,
					"size": size,
				})
			}

			fmt.Printf("Audit log: %s\n", path)
			if size == 0 {
				fmt.Println("Size:      empty (no entries)")
			} else {
				fmt.Printf("Size:      %s\n", formatSize(size))
			}

			entries, _ := auditpkg.ReadEntries(path)
			fmt.Printf("Entries:   %d\n", len(entries))
			return nil
		},
	}
}

func formatSize(bytes int64) string {
	if bytes < 1024 {
		return fmt.Sprintf("%d B", bytes)
	}
	if bytes < 1024*1024 {
		return fmt.Sprintf("%.1f KB", float64(bytes)/1024)
	}
	return fmt.Sprintf("%.1f MB", float64(bytes)/(1024*1024))
}
