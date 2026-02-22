// Package plugin provides the "kit plugin" CLI commands.
package plugin

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"

	pluginpkg "github.com/klytics/m365kit/internal/plugin"
)

// NewCommand creates the "plugin" command with all subcommands.
func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "plugin",
		Short: "Manage M365Kit plugins",
		Long:  "Install, list, run, and create custom M365Kit plugins.",
	}

	cmd.AddCommand(newListCmd())
	cmd.AddCommand(newInstallCmd())
	cmd.AddCommand(newRemoveCmd())
	cmd.AddCommand(newRunCmd())
	cmd.AddCommand(newShowCmd())
	cmd.AddCommand(newNewCmd())

	return cmd
}

func newListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List installed plugins",
		RunE: func(cmd *cobra.Command, args []string) error {
			plugins, err := pluginpkg.Discover()
			if err != nil {
				return err
			}

			jsonOut, _ := cmd.Flags().GetBool("json")
			if jsonOut {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(plugins)
			}

			if len(plugins) == 0 {
				dir, _ := pluginpkg.Dir()
				fmt.Println("No plugins installed.")
				fmt.Printf("Plugin dir: %s\n", dir)
				fmt.Println("\nInstall a plugin:")
				fmt.Println("  kit plugin install --local ./my-plugin/")
				fmt.Println("  kit plugin new --name my-plugin --type shell")
				return nil
			}

			fmt.Printf("Installed Plugins (%d)\n\n", len(plugins))
			tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintf(tw, "NAME\tVERSION\tTYPE\tDESCRIPTION\tSOURCE\n")
			for _, p := range plugins {
				ver := p.Version
				if ver == "" {
					ver = "-"
				}
				desc := p.Description
				if desc == "" {
					desc = "-"
				}
				src := p.Source
				if src == "" {
					src = "local"
				}
				fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n", p.Name, ver, p.Type, desc, src)
			}
			tw.Flush()
			fmt.Println()
			dir, _ := pluginpkg.Dir()
			fmt.Printf("Plugin dir: %s\n", dir)
			return nil
		},
	}
}

func newInstallCmd() *cobra.Command {
	var local bool

	cmd := &cobra.Command{
		Use:   "install <source>",
		Short: "Install a plugin from a local path",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			source := args[0]
			p, err := pluginpkg.Install(source)
			if err != nil {
				return err
			}

			jsonOut, _ := cmd.Flags().GetBool("json")
			if jsonOut {
				return json.NewEncoder(os.Stdout).Encode(p)
			}

			fmt.Printf("Installed plugin: %s", p.Name)
			if p.Version != "" {
				fmt.Printf(" v%s", p.Version)
			}
			fmt.Println()
			fmt.Printf("Run with: kit plugin run %s\n", p.Name)
			return nil
		},
	}

	cmd.Flags().BoolVar(&local, "local", false, "Install from local directory")
	return cmd
}

func newRemoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "remove <name>",
		Short: "Remove an installed plugin",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			if err := pluginpkg.Remove(name); err != nil {
				return err
			}

			jsonOut, _ := cmd.Flags().GetBool("json")
			if jsonOut {
				return json.NewEncoder(os.Stdout).Encode(map[string]string{"removed": name})
			}

			fmt.Printf("Removed plugin: %s\n", name)
			return nil
		},
	}
}

func newRunCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "run <name> [args...]",
		Short: "Run a plugin",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return pluginpkg.Run(cmd.Context(), args[0], args[1:])
		},
	}
}

func newShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show <name>",
		Short: "Show plugin details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			p, err := pluginpkg.Get(args[0])
			if err != nil {
				return err
			}

			jsonOut, _ := cmd.Flags().GetBool("json")
			if jsonOut {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(p)
			}

			fmt.Printf("Name:        %s\n", p.Name)
			if p.Version != "" {
				fmt.Printf("Version:     %s\n", p.Version)
			}
			if p.Description != "" {
				fmt.Printf("Description: %s\n", p.Description)
			}
			if p.Author != "" {
				fmt.Printf("Author:      %s\n", p.Author)
			}
			fmt.Printf("Type:        %s\n", p.Type)
			fmt.Printf("Path:        %s\n", p.Path)
			if !p.InstalledAt.IsZero() {
				fmt.Printf("Installed:   %s\n", p.InstalledAt.Format("2006-01-02 15:04"))
			}
			return nil
		},
	}
}

func newNewCmd() *cobra.Command {
	var (
		name       string
		pluginType string
		outputDir  string
	)

	cmd := &cobra.Command{
		Use:   "new",
		Short: "Create a new plugin scaffold",
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" {
				return fmt.Errorf("--name is required")
			}
			if pluginType == "" {
				pluginType = "shell"
			}
			if outputDir == "" {
				outputDir = "."
			}

			dir, err := pluginpkg.NewScaffold(name, pluginType, outputDir)
			if err != nil {
				return err
			}

			jsonOut, _ := cmd.Flags().GetBool("json")
			if jsonOut {
				return json.NewEncoder(os.Stdout).Encode(map[string]string{
					"name": name,
					"type": pluginType,
					"path": dir,
				})
			}

			fmt.Printf("Created plugin scaffold: %s/\n\n", dir)
			fmt.Println("Files:")
			fmt.Printf("  %s/plugin.yaml          — plugin metadata\n", name)
			fmt.Printf("  %s/kit-%s     — plugin executable\n", name, name)
			fmt.Printf("  %s/README.md             — usage docs\n", name)
			fmt.Println()
			fmt.Println("Next steps:")
			fmt.Printf("  1. Edit %s/kit-%s\n", name, name)
			fmt.Printf("  2. kit plugin install --local %s/\n", dir)
			fmt.Printf("  3. kit %s <args>\n", name)
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Plugin name (required)")
	cmd.Flags().StringVar(&pluginType, "type", "shell", "Plugin type: shell | go")
	cmd.Flags().StringVarP(&outputDir, "output", "o", ".", "Output directory")
	return cmd
}
