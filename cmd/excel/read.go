package excel

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/klytics/m365kit/internal/formats/xlsx"
)

func newReadCommand() *cobra.Command {
	var sheetName string
	var csvOutput bool

	cmd := &cobra.Command{
		Use:   "read <file.xlsx>",
		Short: "Extract data from an Excel spreadsheet",
		Long:  "Reads an .xlsx file and outputs its data. Supports JSON, CSV, and pretty-printed table output. Pass '-' to read from stdin.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			jsonFlag, _ := cmd.Flags().GetBool("json")

			var wb *xlsx.Workbook
			var err error

			if len(args) == 0 || args[0] == "-" {
				data, readErr := io.ReadAll(os.Stdin)
				if readErr != nil {
					return fmt.Errorf("could not read from stdin: %w", readErr)
				}
				if len(data) == 0 {
					return fmt.Errorf("no input provided — pass an .xlsx file path or pipe data to stdin")
				}
				wb, err = xlsx.ReadBytes(data)
			} else {
				filePath := args[0]
				if !strings.HasSuffix(strings.ToLower(filePath), ".xlsx") {
					return fmt.Errorf("expected an .xlsx file, got %q — use 'kit excel read <file.xlsx>'", filePath)
				}
				wb, err = xlsx.ReadFile(filePath)
			}

			if err != nil {
				return err
			}

			// Filter to specific sheet if requested
			if sheetName != "" {
				sheet, err := wb.GetSheet(sheetName)
				if err != nil {
					return err
				}
				wb = &xlsx.Workbook{Sheets: []xlsx.Sheet{*sheet}}
			}

			if jsonFlag {
				return outputExcelJSON(wb)
			}

			if csvOutput {
				return outputExcelCSV(wb)
			}

			return outputExcelPretty(wb)
		},
	}

	cmd.Flags().StringVar(&sheetName, "sheet", "", "Read only the named sheet")
	cmd.Flags().BoolVar(&csvOutput, "csv", false, "Output as CSV")

	return cmd
}

func outputExcelJSON(wb *xlsx.Workbook) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(wb.Sheets)
}

func outputExcelCSV(wb *xlsx.Workbook) error {
	for _, sheet := range wb.Sheets {
		if len(wb.Sheets) > 1 {
			fmt.Fprintf(os.Stderr, "--- %s ---\n", sheet.Name)
		}
		fmt.Print(sheet.ToCSV())
	}
	return nil
}

func outputExcelPretty(wb *xlsx.Workbook) error {
	headerStyle := color.New(color.Bold, color.FgCyan)
	dim := color.New(color.FgHiBlack)

	for _, sheet := range wb.Sheets {
		headerStyle.Printf("Sheet: %s\n", sheet.Name)

		if len(sheet.Rows) == 0 {
			dim.Println("  (empty)")
			continue
		}

		// Calculate column widths
		colWidths := make([]int, 0)
		for _, row := range sheet.Rows {
			for j, cell := range row {
				for len(colWidths) <= j {
					colWidths = append(colWidths, 0)
				}
				if len(cell) > colWidths[j] {
					colWidths[j] = len(cell)
				}
			}
		}

		// Cap column widths
		for i := range colWidths {
			if colWidths[i] > 40 {
				colWidths[i] = 40
			}
			if colWidths[i] < 3 {
				colWidths[i] = 3
			}
		}

		// Print header row
		if len(sheet.Rows) > 0 {
			printRow(sheet.Rows[0], colWidths, color.New(color.Bold))
			// Separator
			dim.Print("  ")
			for j, w := range colWidths {
				if j > 0 {
					dim.Print("+-")
				}
				dim.Print(strings.Repeat("-", w+1))
			}
			dim.Println()
		}

		// Print data rows
		for i := 1; i < len(sheet.Rows); i++ {
			printRow(sheet.Rows[i], colWidths, nil)
		}

		dim.Printf("  (%d rows)\n\n", len(sheet.Rows)-1)
	}

	return nil
}

func printRow(row []string, colWidths []int, style *color.Color) {
	fmt.Print("  ")
	for j := range colWidths {
		if j > 0 {
			fmt.Print("| ")
		}
		cell := ""
		if j < len(row) {
			cell = row[j]
		}
		if len(cell) > colWidths[j] {
			cell = cell[:colWidths[j]-1] + "~"
		}
		padded := cell + strings.Repeat(" ", colWidths[j]-len(cell)+1)
		if style != nil {
			style.Print(padded)
		} else {
			fmt.Print(padded)
		}
	}
	fmt.Println()
}
