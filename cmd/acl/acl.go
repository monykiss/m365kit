// Package acl provides the "kit acl" CLI commands for SharePoint permissions audit.
package acl

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/klytics/m365kit/internal/auth"
	"github.com/klytics/m365kit/internal/graph"
)

// NewCommand creates the "acl" command with all subcommands.
func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "acl",
		Short: "Audit SharePoint permissions and access controls",
		Long: `Audit SharePoint permissions to find external shares, broken inheritance,
and anonymous links. Read-only — never modifies permissions.

Example:
  kit acl audit --site <site-id>
  kit acl external --site <site-id>
  kit acl broken --site <site-id>`,
	}

	cmd.AddCommand(newAuditCmd())
	cmd.AddCommand(newExternalCmd())
	cmd.AddCommand(newBrokenCmd())
	cmd.AddCommand(newUsersCmd())
	cmd.AddCommand(newCheckCmd())

	return cmd
}

func newAuditCmd() *cobra.Command {
	var (
		siteID  string
		domain  string
		output  string
	)

	cmd := &cobra.Command{
		Use:   "audit",
		Short: "Audit all permissions on a SharePoint site",
		RunE: func(cmd *cobra.Command, args []string) error {
			if siteID == "" {
				return fmt.Errorf("--site is required")
			}

			client, err := auth.RequireAuth(cmd.Context())
			if err != nil {
				return err
			}

			a := graph.NewACL(client, domain)
			report, err := a.AuditSitePermissions(cmd.Context(), siteID)
			if err != nil {
				return err
			}

			jsonOut, _ := cmd.Flags().GetBool("json")
			if jsonOut || output != "" {
				data, _ := json.MarshalIndent(report, "", "  ")
				if output != "" {
					if err := os.WriteFile(output, data, 0644); err != nil {
						return err
					}
					fmt.Printf("Audit report saved to %s\n", output)
					return nil
				}
				fmt.Println(string(data))
				return nil
			}

			// Human-readable output
			fmt.Printf("SharePoint ACL Audit: %s\n", report.Site)
			fmt.Printf("Generated: %s\n\n", report.GeneratedAt.Format("2006-01-02 15:04 MST"))

			fmt.Println("Summary")
			fmt.Printf("  Total files scanned:           %d\n", report.TotalFiles)
			fmt.Printf("  Files with external sharing:   %d\n", report.ExternalShares)
			fmt.Printf("  Files with unique permissions: %d\n", report.BrokenInheritance)
			fmt.Printf("  Anonymous share links:         %d\n", report.AnonymousLinks)

			external := graph.FindExternalShares(report)
			if len(external) > 0 {
				fmt.Printf("\nExternal Shares (%d files)\n", len(external))
				tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
				fmt.Fprintf(tw, "FILE\tEXTERNAL USER\tROLE\n")
				for _, entry := range external {
					for _, email := range entry.ExternalUsers {
						roles := "view"
						for _, p := range entry.Permissions {
							if p.GetEmail() == email && len(p.Roles) > 0 {
								roles = strings.Join(p.Roles, ", ")
							}
						}
						fmt.Fprintf(tw, "%s\t%s\t%s\n", entry.Path, email, roles)
					}
				}
				tw.Flush()
			}

			broken := graph.FindBrokenInheritance(report)
			if len(broken) > 0 {
				fmt.Printf("\nBroken Inheritance (%d files)\n", len(broken))
				for _, entry := range broken {
					fmt.Printf("  %s\n", entry.Path)
				}
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&siteID, "site", "", "SharePoint site ID or URL")
	cmd.Flags().StringVar(&domain, "domain", "", "Organization domain for external detection (e.g., company.com)")
	cmd.Flags().StringVarP(&output, "output", "o", "", "Export report to file (JSON)")

	return cmd
}

func newExternalCmd() *cobra.Command {
	var (
		siteID string
		domain string
	)

	cmd := &cobra.Command{
		Use:   "external",
		Short: "Find files shared with external users",
		RunE: func(cmd *cobra.Command, args []string) error {
			if siteID == "" {
				return fmt.Errorf("--site is required")
			}

			client, err := auth.RequireAuth(cmd.Context())
			if err != nil {
				return err
			}

			a := graph.NewACL(client, domain)
			report, err := a.AuditSitePermissions(cmd.Context(), siteID)
			if err != nil {
				return err
			}

			external := graph.FindExternalShares(report)

			jsonOut, _ := cmd.Flags().GetBool("json")
			if jsonOut {
				return json.NewEncoder(os.Stdout).Encode(external)
			}

			if len(external) == 0 {
				fmt.Println("No external shares found.")
				return nil
			}

			tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintf(tw, "FILE\tEXTERNAL USERS\n")
			for _, entry := range external {
				fmt.Fprintf(tw, "%s\t%s\n", entry.Path, strings.Join(entry.ExternalUsers, ", "))
			}
			tw.Flush()
			return nil
		},
	}

	cmd.Flags().StringVar(&siteID, "site", "", "SharePoint site ID")
	cmd.Flags().StringVar(&domain, "domain", "", "Organization domain")
	return cmd
}

func newBrokenCmd() *cobra.Command {
	var siteID string

	cmd := &cobra.Command{
		Use:   "broken",
		Short: "Find files with broken permission inheritance",
		RunE: func(cmd *cobra.Command, args []string) error {
			if siteID == "" {
				return fmt.Errorf("--site is required")
			}

			client, err := auth.RequireAuth(cmd.Context())
			if err != nil {
				return err
			}

			a := graph.NewACL(client, "")
			report, err := a.AuditSitePermissions(cmd.Context(), siteID)
			if err != nil {
				return err
			}

			broken := graph.FindBrokenInheritance(report)

			jsonOut, _ := cmd.Flags().GetBool("json")
			if jsonOut {
				return json.NewEncoder(os.Stdout).Encode(broken)
			}

			if len(broken) == 0 {
				fmt.Println("No files with broken inheritance found.")
				return nil
			}

			fmt.Printf("Files with unique permissions (%d):\n", len(broken))
			for _, entry := range broken {
				fmt.Printf("  %s (%d permissions)\n", entry.Path, len(entry.Permissions))
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&siteID, "site", "", "SharePoint site ID")
	return cmd
}

func newUsersCmd() *cobra.Command {
	var siteID string

	cmd := &cobra.Command{
		Use:   "users",
		Short: "List all users with access to a site",
		RunE: func(cmd *cobra.Command, args []string) error {
			if siteID == "" {
				return fmt.Errorf("--site is required")
			}

			client, err := auth.RequireAuth(cmd.Context())
			if err != nil {
				return err
			}

			a := graph.NewACL(client, "")
			report, err := a.AuditSitePermissions(cmd.Context(), siteID)
			if err != nil {
				return err
			}

			// Collect unique users
			users := make(map[string]string)
			for _, entry := range report.Entries {
				for _, p := range entry.Permissions {
					email := p.GetEmail()
					if email != "" {
						name := ""
						if p.GrantedToV2 != nil && p.GrantedToV2.User != nil {
							name = p.GrantedToV2.User.DisplayName
						} else if p.GrantedTo != nil && p.GrantedTo.User != nil {
							name = p.GrantedTo.User.DisplayName
						}
						users[email] = name
					}
				}
			}

			jsonOut, _ := cmd.Flags().GetBool("json")
			if jsonOut {
				return json.NewEncoder(os.Stdout).Encode(users)
			}

			tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintf(tw, "EMAIL\tNAME\n")
			for email, name := range users {
				fmt.Fprintf(tw, "%s\t%s\n", email, name)
			}
			tw.Flush()
			return nil
		},
	}

	cmd.Flags().StringVar(&siteID, "site", "", "SharePoint site ID")
	return cmd
}

func newCheckCmd() *cobra.Command {
	var (
		siteID string
		file   string
	)

	cmd := &cobra.Command{
		Use:   "check",
		Short: "Check who has access to a specific file",
		RunE: func(cmd *cobra.Command, args []string) error {
			if siteID == "" || file == "" {
				return fmt.Errorf("--site and --file are required")
			}

			jsonOut, _ := cmd.Flags().GetBool("json")
			if jsonOut {
				return json.NewEncoder(os.Stdout).Encode(map[string]string{
					"message": "check requires audit data — use 'kit acl audit' for full results",
				})
			}

			fmt.Println("Use 'kit acl audit --site <id> --json' and search for the file path.")
			return nil
		},
	}

	cmd.Flags().StringVar(&siteID, "site", "", "SharePoint site ID")
	cmd.Flags().StringVar(&file, "file", "", "File path to check")
	return cmd
}
