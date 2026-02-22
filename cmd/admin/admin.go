// Package admin provides the "kit admin" CLI commands for IT administrators.
package admin

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	adminpkg "github.com/klytics/m365kit/internal/admin"
	auditpkg "github.com/klytics/m365kit/internal/audit"
	"github.com/klytics/m365kit/internal/config"
	"github.com/klytics/m365kit/internal/telemetry"
)

// NewCommand creates the "admin" command with all subcommands.
func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "admin",
		Short: "IT administration: usage stats, user management",
		Long:  "Org-wide usage statistics, user activity, and telemetry management for IT administrators.",
	}

	cmd.AddCommand(newStatsCmd())
	cmd.AddCommand(newUsersCmd())
	cmd.AddCommand(newTelemetryCmd())

	return cmd
}

func newStatsCmd() *cobra.Command {
	var (
		since string
		by    string
	)

	cmd := &cobra.Command{
		Use:   "stats",
		Short: "Show usage statistics",
		RunE: func(cmd *cobra.Command, args []string) error {
			path := auditLogPath()
			entries, err := auditpkg.ReadEntries(path)
			if err != nil {
				return err
			}

			filter := adminpkg.StatsFilter{}
			if since != "" {
				t, err := parseDate(since)
				if err != nil {
					return err
				}
				filter.Since = t
			}

			stats := adminpkg.AggregateStats(entries, filter)

			jsonOut, _ := cmd.Flags().GetBool("json")
			if jsonOut {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(stats)
			}

			orgCfg, _ := config.LoadOrgConfig()
			orgName := "Local"
			if orgCfg != nil {
				orgName = orgCfg.OrgName
			}

			fmt.Printf("M365Kit Usage — %s\n\n", orgName)
			fmt.Println("SUMMARY")
			fmt.Printf("  Active users:    %d\n", stats.ActiveUsers)
			fmt.Printf("  Commands run:    %d\n", stats.CommandCount)
			fmt.Printf("  Error rate:      %.1f%%\n", stats.ErrorRate)
			fmt.Println()

			if by == "command" || by == "" {
				if len(stats.TopCommands) > 0 {
					fmt.Println("TOP COMMANDS")
					tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
					limit := 10
					if limit > len(stats.TopCommands) {
						limit = len(stats.TopCommands)
					}
					for _, c := range stats.TopCommands[:limit] {
						fmt.Fprintf(tw, "  %s\t%d\t(%.0f%%)\n", c.Command, c.Count, c.Pct)
					}
					tw.Flush()
					fmt.Println()
				}
			}

			if by == "user" || by == "" {
				if len(stats.TopUsers) > 0 {
					fmt.Println("TOP USERS")
					tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
					limit := 10
					if limit > len(stats.TopUsers) {
						limit = len(stats.TopUsers)
					}
					for _, u := range stats.TopUsers[:limit] {
						fmt.Fprintf(tw, "  %s\t%d commands\n", u.UserID, u.Count)
					}
					tw.Flush()
				}
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&since, "since", "", "Stats since date (YYYY-MM-DD)")
	cmd.Flags().StringVar(&by, "by", "", "Group by: command | user")
	return cmd
}

func newUsersCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "users",
		Short: "List all users who have used M365Kit",
		RunE: func(cmd *cobra.Command, args []string) error {
			path := auditLogPath()
			entries, _ := auditpkg.ReadEntries(path)

			users := make(map[string]int)
			for _, e := range entries {
				if e.UserID != "" {
					users[e.UserID]++
				}
			}

			jsonOut, _ := cmd.Flags().GetBool("json")
			if jsonOut {
				return json.NewEncoder(os.Stdout).Encode(users)
			}

			if len(users) == 0 {
				fmt.Println("No users found in audit log.")
				return nil
			}

			tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintf(tw, "USER\tCOMMANDS\n")
			for user, count := range users {
				fmt.Fprintf(tw, "%s\t%d\n", user, count)
			}
			tw.Flush()
			return nil
		},
	}
}

func newTelemetryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "telemetry",
		Short: "Manage local telemetry data",
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "status",
		Short: "Show telemetry store size",
		RunE: func(cmd *cobra.Command, args []string) error {
			store := telemetry.DefaultStore()
			size := store.Size()

			jsonOut, _ := cmd.Flags().GetBool("json")
			if jsonOut {
				return json.NewEncoder(os.Stdout).Encode(map[string]interface{}{
					"path": store.Path,
					"size": size,
				})
			}

			fmt.Printf("Telemetry store: %s\n", store.Path)
			if size == 0 {
				fmt.Println("Size:            empty")
			} else {
				fmt.Printf("Size:            %d bytes\n", size)
			}
			return nil
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "clear",
		Short: "Clear local telemetry data",
		RunE: func(cmd *cobra.Command, args []string) error {
			store := telemetry.DefaultStore()
			if err := store.Clear(); err != nil {
				return err
			}
			jsonOut, _ := cmd.Flags().GetBool("json")
			if jsonOut {
				return json.NewEncoder(os.Stdout).Encode(map[string]string{"cleared": store.Path})
			}
			fmt.Println("Telemetry data cleared.")
			return nil
		},
	})
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

func parseDate(s string) (time.Time, error) {
	return time.Parse("2006-01-02", s)
}
