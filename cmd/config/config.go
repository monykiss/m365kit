// Package config provides CLI commands for configuration management.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/klytics/m365kit/internal/config"
)

// NewCommand returns the config command group.
func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage M365Kit configuration",
		Long:  "Interactive setup, view, and modify M365Kit settings.",
	}

	cmd.AddCommand(newInitCommand())
	cmd.AddCommand(newShowCommand())
	cmd.AddCommand(newSetCommand())
	cmd.AddCommand(newGetCommand())
	cmd.AddCommand(newResetCommand())
	cmd.AddCommand(newPathCommand())
	cmd.AddCommand(newValidateCommand())
	cmd.AddCommand(newEnvCommand())

	return cmd
}

func newInitCommand() *cobra.Command {
	var noInteractive bool
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Interactive setup wizard",
		RunE: func(cmd *cobra.Command, args []string) error {
			if noInteractive {
				return config.WizardNonInteractive()
			}
			return config.Wizard(nil)
		},
	}
	cmd.Flags().BoolVar(&noInteractive, "no-interactive", false, "Skip prompts, use defaults")
	return cmd
}

func newShowCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Show current configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			jsonFlag, _ := cmd.Flags().GetBool("json")
			config.Load() // ensure loaded

			if jsonFlag {
				env := config.ToEnv()
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(env)
			}

			fmt.Print(config.ShowConfig())
			return nil
		},
	}
}

func newSetCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set a configuration value",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			config.Load() // ensure loaded
			if err := config.Set(args[0], args[1]); err != nil {
				return err
			}
			fmt.Printf("Set %s = %s\n", args[0], args[1])
			return nil
		},
	}
}

func newGetCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "get <key>",
		Short: "Get a configuration value",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			config.Load() // ensure loaded
			val := config.Get(args[0])
			if val == "" {
				fmt.Printf("%s: (not set)\n", args[0])
			} else {
				fmt.Printf("%s: %s\n", args[0], val)
			}
			return nil
		},
	}
}

func newResetCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "reset",
		Short: "Reset configuration to defaults",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := config.ResetConfig(); err != nil {
				return err
			}
			fmt.Println("Configuration reset to defaults")
			return nil
		},
	}
}

func newPathCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "path",
		Short: "Show config file path",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(config.ConfigPath())
		},
	}
}

func newValidateCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "validate",
		Short: "Validate current configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			jsonFlag, _ := cmd.Flags().GetBool("json")
			config.Load() // ensure loaded

			issues := config.Validate()

			if jsonFlag {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(issues)
			}

			errors := 0
			warnings := 0
			for _, issue := range issues {
				switch issue.Severity {
				case "error":
					errors++
				case "warning":
					warnings++
				}
			}

			if errors == 0 && warnings == 0 {
				color.New(color.FgGreen).Println("Configuration is valid")
				return nil
			}

			fmt.Printf("Config validation: %d errors, %d warnings\n\n", errors, warnings)

			for _, issue := range issues {
				switch issue.Severity {
				case "error":
					color.New(color.FgRed).Printf("  %s\n", issue.Message)
				case "warning":
					color.New(color.FgYellow).Printf("  %s\n", issue.Message)
				case "info":
					color.New(color.FgGreen).Printf("  %s\n", issue.Message)
				}
				if issue.Fix != "" {
					fmt.Printf("   Fix: %s\n", issue.Fix)
				}
			}
			return nil
		},
	}
}

func newEnvCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "env",
		Short: "Export configuration as environment variables",
		RunE: func(cmd *cobra.Command, args []string) error {
			jsonFlag, _ := cmd.Flags().GetBool("json")
			config.Load() // ensure loaded

			env := config.ToEnv()

			if jsonFlag {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(env)
			}

			// Sort keys for deterministic output
			keys := make([]string, 0, len(env))
			for k := range env {
				keys = append(keys, k)
			}
			sort.Strings(keys)

			for _, k := range keys {
				fmt.Printf("export %s=%q\n", k, env[k])
			}
			fmt.Println("# Add these to your ~/.zshrc or ~/.bashrc")
			return nil
		},
	}
}
