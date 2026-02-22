// Package org provides the "kit org" CLI commands for org-wide configuration.
package org

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/klytics/m365kit/internal/config"
)

// NewCommand creates the "org" command with all subcommands.
func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "org",
		Short: "Manage organization-wide configuration",
		Long: `View, validate, and manage the org-wide M365Kit configuration.
Org config is deployed by IT admins to control defaults, lock settings,
and enable audit logging for all users.`,
	}

	cmd.AddCommand(newShowCmd())
	cmd.AddCommand(newValidateCmd())
	cmd.AddCommand(newInitCmd())
	cmd.AddCommand(newStatusCmd())

	return cmd
}

func newShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Show current org configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.LoadOrgConfig()
			if err != nil {
				return err
			}

			jsonOut, _ := cmd.Flags().GetBool("json")

			if cfg == nil {
				if jsonOut {
					return json.NewEncoder(os.Stdout).Encode(map[string]string{
						"status": "community",
						"path":   config.OrgConfigPath(),
					})
				}
				fmt.Printf("No org config found at %s\n", config.OrgConfigPath())
				fmt.Println("Running in community mode — all features available.")
				return nil
			}

			if jsonOut {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(cfg)
			}

			fmt.Printf("Organization: %s (%s)\n", cfg.OrgName, cfg.OrgDomain)
			fmt.Printf("Config:       %s\n", config.OrgConfigPath())
			fmt.Println()
			if cfg.AI.Provider != "" {
				locked := ""
				if cfg.Locked.AIProvider {
					locked = "  [LOCKED]"
				}
				fmt.Printf("AI Provider:  %s%s\n", cfg.AI.Provider, locked)
			}
			if cfg.Azure.ClientID != "" {
				locked := ""
				if cfg.Locked.AzureClientID {
					locked = "  [LOCKED]"
				}
				fmt.Printf("Azure App:    %s%s\n", cfg.Azure.ClientID, locked)
			}
			if len(cfg.AllowedCommands) > 0 {
				fmt.Printf("Commands:     %v\n", cfg.AllowedCommands)
			} else {
				fmt.Println("Commands:     all allowed")
			}
			fmt.Println()
			if cfg.Audit.Enabled {
				fmt.Printf("Audit:        enabled -> %s\n", cfg.AuditLogPath())
			} else {
				fmt.Println("Audit:        disabled")
			}
			if cfg.Telemetry.Enabled {
				fmt.Println("Telemetry:    enabled")
			} else {
				fmt.Println("Telemetry:    disabled")
			}
			return nil
		},
	}
}

func newValidateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "validate [file]",
		Short: "Validate an org config file",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.LoadOrgConfigFrom(args[0])
			if err != nil {
				return err
			}
			if cfg == nil {
				return fmt.Errorf("file not found: %s", args[0])
			}

			issues := config.ValidateOrgConfig(cfg)

			jsonOut, _ := cmd.Flags().GetBool("json")
			if jsonOut {
				return json.NewEncoder(os.Stdout).Encode(map[string]interface{}{
					"valid":  len(issues) == 0,
					"issues": issues,
				})
			}

			if len(issues) == 0 {
				fmt.Printf("Valid org config: %s (%s)\n", cfg.OrgName, cfg.OrgDomain)
				return nil
			}

			fmt.Printf("Validation failed (%d issues):\n", len(issues))
			for _, issue := range issues {
				fmt.Printf("  - %s\n", issue)
			}
			return fmt.Errorf("%d validation issues found", len(issues))
		},
	}
}

func newInitCmd() *cobra.Command {
	var (
		orgName string
		domain  string
	)

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Generate an org config template",
		RunE: func(cmd *cobra.Command, args []string) error {
			if orgName == "" {
				orgName = "My Organization"
			}
			if domain == "" {
				domain = "example.com"
			}
			fmt.Print(config.GenerateOrgTemplate(orgName, domain))
			return nil
		},
	}

	cmd.Flags().StringVar(&orgName, "org-name", "", "Organization name")
	cmd.Flags().StringVar(&domain, "domain", "", "Organization domain")
	return cmd
}

func newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show how org policy affects current user",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.LoadOrgConfig()
			if err != nil {
				return err
			}

			jsonOut, _ := cmd.Flags().GetBool("json")

			if cfg == nil {
				if jsonOut {
					return json.NewEncoder(os.Stdout).Encode(map[string]string{"mode": "community"})
				}
				fmt.Println("Running in community mode (no org config detected).")
				fmt.Println("All features are available.")
				return nil
			}

			if jsonOut {
				return json.NewEncoder(os.Stdout).Encode(map[string]interface{}{
					"mode":      "organization",
					"org_name":  cfg.OrgName,
					"audit":     cfg.Audit.Enabled,
					"telemetry": cfg.Telemetry.Enabled,
				})
			}

			fmt.Printf("You are operating under %s organization policy.\n\n", cfg.OrgName)
			if cfg.Azure.ClientID != "" {
				fmt.Printf("  Microsoft 365: Org Azure app configured (%s)\n", cfg.Azure.ClientID)
			}
			if cfg.AI.Provider != "" {
				status := ""
				if cfg.Locked.AIProvider {
					status = " (locked by org policy)"
				}
				fmt.Printf("  AI Provider:   %s%s\n", cfg.AI.Provider, status)
			}
			if cfg.Audit.Enabled {
				fmt.Printf("  Audit log:     enabled -> %s\n", cfg.AuditLogPath())
			}
			if cfg.Telemetry.Enabled {
				fmt.Println("  Telemetry:     enabled")
			}
			return nil
		},
	}
}
