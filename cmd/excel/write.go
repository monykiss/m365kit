package excel

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/klytics/m365kit/internal/formats/xlsx"
)

type excelWriteInput struct {
	Sheets []excelWriteSheet `json:"sheets"`
}

type excelWriteSheet struct {
	Name    string     `json:"name"`
	Headers []string   `json:"headers,omitempty"`
	Rows    [][]interface{} `json:"rows"`
}

type excelWriteJSONOutput struct {
	File   string `json:"file"`
	Sheets int    `json:"sheets"`
	Rows   int    `json:"rows"`
}

func newWriteCommand() *cobra.Command {
	var (
		output    string
		dataPath  string
		sheetName string
	)

	cmd := &cobra.Command{
		Use:   "write",
		Short: "Generate an Excel spreadsheet from data",
		Long: `Creates an .xlsx file from structured JSON data.

JSON format:
  {"sheets": [{"name": "Sheet1", "headers": ["A","B"], "rows": [["a1","b1"]]}]}`,
		RunE: func(cmd *cobra.Command, args []string) error {
			jsonFlag, _ := cmd.Flags().GetBool("json")

			if output == "" {
				return fmt.Errorf("--output is required — specify the output .xlsx path\n\nExample: kit excel write --output data.xlsx --data input.json")
			}

			if !strings.HasSuffix(strings.ToLower(output), ".xlsx") {
				output += ".xlsx"
			}

			if dataPath == "" {
				return fmt.Errorf("--data is required — provide a JSON data file or - for stdin\n\nExample: kit excel write --output data.xlsx --data input.json")
			}

			// Read input data
			var raw []byte
			var err error
			if dataPath == "-" {
				raw, err = io.ReadAll(os.Stdin)
			} else {
				raw, err = os.ReadFile(dataPath)
			}
			if err != nil {
				return fmt.Errorf("could not read data: %w", err)
			}

			// Try to parse as our structured format first
			var input excelWriteInput
			if err := json.Unmarshal(raw, &input); err != nil {
				// Try parsing as an array of sheets (from kit excel read --json)
				var sheets []xlsx.Sheet
				if err2 := json.Unmarshal(raw, &sheets); err2 == nil {
					return writeFromSheets(sheets, output, jsonFlag)
				}
				return fmt.Errorf("invalid JSON data: %w — expected {\"sheets\": [...]}", err)
			}

			// Override sheet name if specified
			if sheetName != "" && len(input.Sheets) == 1 {
				input.Sheets[0].Name = sheetName
			}

			// Build workbook
			wb := &xlsx.Workbook{}
			totalRows := 0
			for _, s := range input.Sheets {
				name := s.Name
				if name == "" {
					name = fmt.Sprintf("Sheet%d", len(wb.Sheets)+1)
				}

				var rows [][]string

				// Add headers as first row if present
				if len(s.Headers) > 0 {
					rows = append(rows, s.Headers)
				}

				// Convert rows
				for _, row := range s.Rows {
					var strRow []string
					for _, cell := range row {
						strRow = append(strRow, fmt.Sprintf("%v", cell))
					}
					rows = append(rows, strRow)
				}

				totalRows += len(rows)
				wb.Sheets = append(wb.Sheets, xlsx.Sheet{
					Name: name,
					Rows: rows,
				})
			}

			if err := xlsx.WriteFile(wb, output); err != nil {
				return fmt.Errorf("could not write file: %w", err)
			}

			if jsonFlag {
				out := excelWriteJSONOutput{
					File:   output,
					Sheets: len(wb.Sheets),
					Rows:   totalRows,
				}
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(out)
			}

			fmt.Printf("Wrote %s (%d sheets, %d rows)\n", output, len(wb.Sheets), totalRows)
			return nil
		},
	}

	cmd.Flags().StringVar(&output, "output", "", "Output .xlsx file path (required)")
	cmd.Flags().StringVar(&dataPath, "data", "", "Path to JSON data file (or - for stdin)")
	cmd.Flags().StringVar(&sheetName, "sheet", "", "Sheet name (shortcut for single-sheet files)")

	return cmd
}

func writeFromSheets(sheets []xlsx.Sheet, output string, jsonFlag bool) error {
	wb := &xlsx.Workbook{Sheets: sheets}
	totalRows := 0
	for _, s := range sheets {
		totalRows += len(s.Rows)
	}

	if err := xlsx.WriteFile(wb, output); err != nil {
		return fmt.Errorf("could not write file: %w", err)
	}

	if jsonFlag {
		out := excelWriteJSONOutput{
			File:   output,
			Sheets: len(sheets),
			Rows:   totalRows,
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(out)
	}

	fmt.Printf("Wrote %s (%d sheets, %d rows)\n", output, len(sheets), totalRows)
	return nil
}
