// Package report provides the "kit report" CLI command for generating reports.
package report

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	rpt "github.com/klytics/m365kit/internal/report"
)

// NewCommand creates the "report" command.
func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "report",
		Short: "Generate reports from data and templates",
		Long: `Generate document reports by combining a .docx template with a data source.

Data sources can be CSV or JSON files. Aggregate variables (sum, avg, min, max)
are automatically computed for numeric columns.

Example:
  kit report generate --template invoice.docx --data sales.csv -o report.docx
  kit report preview --data sales.csv`,
	}

	cmd.AddCommand(newGenerateCmd())
	cmd.AddCommand(newPreviewCmd())

	return cmd
}

func newGenerateCmd() *cobra.Command {
	var (
		templatePath string
		dataPath     string
		outputPath   string
		setValues    []string
	)

	cmd := &cobra.Command{
		Use:   "generate",
		Short: "Generate a report from a template and data source",
		RunE: func(cmd *cobra.Command, args []string) error {
			if templatePath == "" {
				return fmt.Errorf("--template is required")
			}
			if dataPath == "" {
				return fmt.Errorf("--data is required")
			}

			if outputPath == "" {
				base := strings.TrimSuffix(templatePath, ".docx")
				outputPath = base + "_report.docx"
			}

			extra := make(map[string]string)
			for _, s := range setValues {
				parts := strings.SplitN(s, "=", 2)
				if len(parts) != 2 {
					return fmt.Errorf("invalid --set format: %q (expected key=value)", s)
				}
				extra[parts[0]] = parts[1]
			}

			result, err := rpt.Generate(rpt.GenerateOptions{
				TemplatePath: templatePath,
				DataPath:     dataPath,
				OutputPath:   outputPath,
				ExtraValues:  extra,
			})
			if err != nil {
				return err
			}

			jsonOut, _ := cmd.Flags().GetBool("json")
			if jsonOut {
				return json.NewEncoder(os.Stdout).Encode(result)
			}

			fmt.Printf("Report generated → %s\n", result.OutputPath)
			fmt.Printf("  Data rows:    %d\n", result.DataRows)
			fmt.Printf("  Applied:      %d variable(s)\n", result.VariablesApplied)
			if result.VariablesMissing > 0 {
				fmt.Printf("  Missing:      %s\n", strings.Join(result.MissingNames, ", "))
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&templatePath, "template", "t", "", "Template .docx file path")
	cmd.Flags().StringVarP(&dataPath, "data", "d", "", "Data source file (.csv or .json)")
	cmd.Flags().StringVarP(&outputPath, "output", "o", "", "Output file path")
	cmd.Flags().StringSliceVar(&setValues, "set", nil, "Additional variable values (key=value)")

	return cmd
}

func newPreviewCmd() *cobra.Command {
	var (
		dataPath  string
		setValues []string
	)

	cmd := &cobra.Command{
		Use:   "preview",
		Short: "Preview the variables that would be available from a data source",
		RunE: func(cmd *cobra.Command, args []string) error {
			if dataPath == "" {
				return fmt.Errorf("--data is required")
			}

			extra := make(map[string]string)
			for _, s := range setValues {
				parts := strings.SplitN(s, "=", 2)
				if len(parts) == 2 {
					extra[parts[0]] = parts[1]
				}
			}

			vars, err := rpt.PreviewVariables(dataPath, extra)
			if err != nil {
				return err
			}

			jsonOut, _ := cmd.Flags().GetBool("json")
			if jsonOut {
				return json.NewEncoder(os.Stdout).Encode(vars)
			}

			// Sort and display
			keys := make([]string, 0, len(vars))
			for k := range vars {
				keys = append(keys, k)
			}
			sort.Strings(keys)

			tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintf(tw, "VARIABLE\tVALUE\n")
			for _, k := range keys {
				v := vars[k]
				if len(v) > 50 {
					v = v[:47] + "..."
				}
				fmt.Fprintf(tw, "{{%s}}\t%s\n", k, v)
			}
			tw.Flush()
			return nil
		},
	}

	cmd.Flags().StringVarP(&dataPath, "data", "d", "", "Data source file (.csv or .json)")
	cmd.Flags().StringSliceVar(&setValues, "set", nil, "Additional variable values (key=value)")

	return cmd
}
