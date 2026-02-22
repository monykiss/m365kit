package convert

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/klytics/m365kit/internal/formats/xlsx"
)

// XlsxToCSV converts an XLSX sheet to CSV format.
func XlsxToCSV(inputPath, sheetName string) (string, error) {
	sheet, err := getSheet(inputPath, sheetName)
	if err != nil {
		return "", err
	}
	return sheet.ToCSV(), nil
}

// XlsxToJSON converts an XLSX sheet to JSON format.
// First row is treated as headers, remaining rows become objects.
func XlsxToJSON(inputPath, sheetName string) (string, error) {
	sheet, err := getSheet(inputPath, sheetName)
	if err != nil {
		return "", err
	}

	if len(sheet.Rows) < 1 {
		return "[]", nil
	}

	headers := sheet.Rows[0]
	var records []map[string]string

	for _, row := range sheet.Rows[1:] {
		record := make(map[string]string)
		for i, h := range headers {
			if i < len(row) {
				record[h] = row[i]
			} else {
				record[h] = ""
			}
		}
		records = append(records, record)
	}

	result, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		return "", err
	}
	return string(result), nil
}

// XlsxToMarkdown converts an XLSX sheet to a GFM Markdown table.
// First row is treated as headers.
func XlsxToMarkdown(inputPath, sheetName string) (string, error) {
	sheet, err := getSheet(inputPath, sheetName)
	if err != nil {
		return "", err
	}

	if len(sheet.Rows) < 1 {
		return "", nil
	}

	headers := sheet.Rows[0]
	var b strings.Builder

	// Header row
	b.WriteString("| ")
	b.WriteString(strings.Join(headers, " | "))
	b.WriteString(" |\n")

	// Separator row
	b.WriteString("|")
	for range headers {
		b.WriteString(" --- |")
	}
	b.WriteString("\n")

	// Data rows
	for _, row := range sheet.Rows[1:] {
		b.WriteString("| ")
		cells := make([]string, len(headers))
		for i := range headers {
			if i < len(row) {
				cells[i] = row[i]
			}
		}
		b.WriteString(strings.Join(cells, " | "))
		b.WriteString(" |\n")
	}

	return b.String(), nil
}

func getSheet(inputPath, sheetName string) (*xlsx.Sheet, error) {
	wb, err := xlsx.ReadFile(inputPath)
	if err != nil {
		return nil, fmt.Errorf("could not read xlsx: %w", err)
	}

	if sheetName != "" {
		return wb.GetSheet(sheetName)
	}

	if len(wb.Sheets) == 0 {
		return nil, fmt.Errorf("no sheets found in %s", inputPath)
	}
	return &wb.Sheets[0], nil
}
