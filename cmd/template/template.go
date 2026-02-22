// Package template provides the "kit template" CLI commands for document template management.
package template

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	tmpl "github.com/klytics/m365kit/internal/template"
)

// NewCommand creates the "template" command with all subcommands.
func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "template",
		Aliases: []string{"tmpl"},
		Short:   "Manage document templates with variable substitution",
		Long:    "Create, manage, and apply document templates with {{variable}} placeholders.",
	}

	cmd.AddCommand(newListCmd())
	cmd.AddCommand(newShowCmd())
	cmd.AddCommand(newApplyCmd())
	cmd.AddCommand(newAddCmd())
	cmd.AddCommand(newRemoveCmd())
	cmd.AddCommand(newVarsCmd())

	return cmd
}

func newListCmd() *cobra.Command {
	var libraryDir string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all registered templates",
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := libraryDir
			if dir == "" {
				dir = tmpl.DefaultLibraryDir()
			}

			lib, err := tmpl.LoadLibrary(dir)
			if err != nil {
				return err
			}

			templates := lib.List()
			jsonOut, _ := cmd.Flags().GetBool("json")

			if jsonOut {
				return json.NewEncoder(os.Stdout).Encode(templates)
			}

			if len(templates) == 0 {
				fmt.Println("No templates registered. Use 'kit template add' to register one.")
				return nil
			}

			tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintf(tw, "NAME\tVARIABLES\tDESCRIPTION\n")
			for _, t := range templates {
				varNames := make([]string, len(t.Variables))
				for i, v := range t.Variables {
					varNames[i] = v.Name
				}
				fmt.Fprintf(tw, "%s\t%s\t%s\n", t.Name, strings.Join(varNames, ", "), t.Description)
			}
			tw.Flush()
			return nil
		},
	}

	cmd.Flags().StringVar(&libraryDir, "dir", "", "Template library directory (default: ~/.kit/templates)")
	return cmd
}

func newShowCmd() *cobra.Command {
	var libraryDir string

	cmd := &cobra.Command{
		Use:   "show <name>",
		Short: "Show details of a registered template",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := libraryDir
			if dir == "" {
				dir = tmpl.DefaultLibraryDir()
			}

			lib, err := tmpl.LoadLibrary(dir)
			if err != nil {
				return err
			}

			t, err := lib.Get(args[0])
			if err != nil {
				return err
			}

			jsonOut, _ := cmd.Flags().GetBool("json")
			if jsonOut {
				return json.NewEncoder(os.Stdout).Encode(t)
			}

			fmt.Printf("Name:        %s\n", t.Name)
			fmt.Printf("Description: %s\n", t.Description)
			fmt.Printf("Path:        %s\n", t.Path)
			fmt.Printf("Created:     %s\n", t.CreatedAt.Format("2006-01-02 15:04"))
			fmt.Printf("Updated:     %s\n", t.UpdatedAt.Format("2006-01-02 15:04"))
			fmt.Printf("Variables:   %d\n", len(t.Variables))
			for _, v := range t.Variables {
				req := ""
				if v.Required {
					req = " (required)"
				}
				fmt.Printf("  - %s%s\n", v.Name, req)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&libraryDir, "dir", "", "Template library directory")
	return cmd
}

func newApplyCmd() *cobra.Command {
	var (
		outputPath string
		setValues  []string
		dryRun     bool
	)

	cmd := &cobra.Command{
		Use:   "apply <template.docx|name> [--set key=value ...]",
		Short: "Apply variable substitution to a template",
		Long: `Apply variable values to a document template.

Variables can be provided via --set flags:
  kit template apply contract.docx --set name="John Doe" --set date="2025-01-01" -o filled.docx

Or apply a registered template by name:
  kit template apply invoice --set client="Acme Corp" --set amount="$5,000" -o invoice.docx`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Parse --set values
			values := make(map[string]string)
			for _, s := range setValues {
				parts := strings.SplitN(s, "=", 2)
				if len(parts) != 2 {
					return fmt.Errorf("invalid --set format: %q (expected key=value)", s)
				}
				values[parts[0]] = parts[1]
			}

			input := args[0]
			templatePath := input

			// Check if it's a library template name (no file extension)
			if !strings.HasSuffix(input, ".docx") {
				lib, err := tmpl.LoadLibrary(tmpl.DefaultLibraryDir())
				if err == nil {
					if t, err := lib.Get(input); err == nil {
						templatePath = t.Path
					}
				}
			}

			if outputPath == "" {
				base := strings.TrimSuffix(templatePath, ".docx")
				outputPath = base + "_filled.docx"
			}

			jsonOut, _ := cmd.Flags().GetBool("json")

			if dryRun {
				vars, err := tmpl.ExtractVariables(templatePath)
				if err != nil {
					return err
				}
				if jsonOut {
					result := map[string]any{
						"dryRun":    true,
						"template":  templatePath,
						"variables": vars,
						"provided":  values,
					}
					return json.NewEncoder(os.Stdout).Encode(result)
				}
				fmt.Printf("Template: %s\n", templatePath)
				fmt.Printf("Output:   %s (dry run — no file written)\n", outputPath)
				fmt.Printf("Variables found:\n")
				for _, v := range vars {
					val, ok := values[v.Name]
					if ok {
						fmt.Printf("  %s = %q\n", v.Name, val)
					} else {
						fmt.Printf("  %s = (not provided)\n", v.Name)
					}
				}
				return nil
			}

			result, err := tmpl.Apply(templatePath, values, outputPath)
			if err != nil {
				return err
			}

			if jsonOut {
				return json.NewEncoder(os.Stdout).Encode(result)
			}

			fmt.Printf("Applied %d variable(s) → %s\n", result.VariablesApplied, result.OutputPath)
			if result.VariablesMissing > 0 {
				fmt.Printf("Warning: %d variable(s) not provided: %s\n",
					result.VariablesMissing, strings.Join(result.MissingNames, ", "))
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&outputPath, "output", "o", "", "Output file path (default: <input>_filled.docx)")
	cmd.Flags().StringSliceVar(&setValues, "set", nil, "Set variable value (key=value)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be substituted without writing")

	return cmd
}

func newAddCmd() *cobra.Command {
	var (
		description string
		libraryDir  string
	)

	cmd := &cobra.Command{
		Use:   "add <name> <file.docx>",
		Short: "Register a document as a template in the library",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := libraryDir
			if dir == "" {
				dir = tmpl.DefaultLibraryDir()
			}

			lib, err := tmpl.LoadLibrary(dir)
			if err != nil {
				return err
			}

			t, err := lib.Add(args[0], description, args[1])
			if err != nil {
				return err
			}

			jsonOut, _ := cmd.Flags().GetBool("json")
			if jsonOut {
				return json.NewEncoder(os.Stdout).Encode(t)
			}

			fmt.Printf("Added template %q with %d variable(s)\n", t.Name, len(t.Variables))
			for _, v := range t.Variables {
				fmt.Printf("  - %s\n", v.Name)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&description, "description", "", "Template description")
	cmd.Flags().StringVar(&libraryDir, "dir", "", "Template library directory")
	return cmd
}

func newRemoveCmd() *cobra.Command {
	var libraryDir string

	cmd := &cobra.Command{
		Use:   "remove <name>",
		Short: "Remove a template from the library",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := libraryDir
			if dir == "" {
				dir = tmpl.DefaultLibraryDir()
			}

			lib, err := tmpl.LoadLibrary(dir)
			if err != nil {
				return err
			}

			if err := lib.Remove(args[0]); err != nil {
				return err
			}

			jsonOut, _ := cmd.Flags().GetBool("json")
			if jsonOut {
				return json.NewEncoder(os.Stdout).Encode(map[string]string{
					"removed": args[0],
				})
			}

			fmt.Printf("Removed template %q\n", args[0])
			return nil
		},
	}

	cmd.Flags().StringVar(&libraryDir, "dir", "", "Template library directory")
	return cmd
}

func newVarsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "vars <file.docx>",
		Short: "Extract and list template variables from a document",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			vars, err := tmpl.ExtractVariables(args[0])
			if err != nil {
				return err
			}

			jsonOut, _ := cmd.Flags().GetBool("json")
			if jsonOut {
				return json.NewEncoder(os.Stdout).Encode(vars)
			}

			if len(vars) == 0 {
				fmt.Println("No template variables found.")
				return nil
			}

			fmt.Printf("Variables in %s:\n", args[0])
			for _, v := range vars {
				fmt.Printf("  {{%s}}\n", v.Name)
			}
			return nil
		},
	}

	return cmd
}
