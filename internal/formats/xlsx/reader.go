// Package xlsx provides reading and writing capabilities for .xlsx (Excel) files.
package xlsx

import (
	"bytes"
	"fmt"
	"os"

	"github.com/xuri/excelize/v2"
)

// Sheet represents a single worksheet's data.
type Sheet struct {
	Name string     `json:"name"`
	Rows [][]string `json:"rows"`
}

// Workbook represents a parsed Excel file with all its sheets.
type Workbook struct {
	Sheets []Sheet `json:"sheets"`
}

// ReadFile reads an .xlsx file and returns its structured data.
func ReadFile(path string) (*Workbook, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf("file not found: %s — check that the path is correct", path)
	}

	f, err := excelize.OpenFile(path)
	if err != nil {
		return nil, fmt.Errorf("could not open %s — is this a valid .xlsx file? %w", path, err)
	}
	defer f.Close()

	return readWorkbook(f)
}

// ReadBytes reads an .xlsx file from a byte slice and returns its structured data.
func ReadBytes(data []byte) (*Workbook, error) {
	f, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("could not read Excel data: %w", err)
	}
	defer f.Close()

	return readWorkbook(f)
}

func readWorkbook(f *excelize.File) (*Workbook, error) {
	wb := &Workbook{}

	for _, name := range f.GetSheetList() {
		rows, err := f.GetRows(name)
		if err != nil {
			return nil, fmt.Errorf("could not read sheet %q: %w", name, err)
		}

		sheet := Sheet{
			Name: name,
			Rows: rows,
		}
		wb.Sheets = append(wb.Sheets, sheet)
	}

	return wb, nil
}

// GetSheet returns a specific sheet by name. Returns an error if the sheet is not found.
func (wb *Workbook) GetSheet(name string) (*Sheet, error) {
	for i := range wb.Sheets {
		if wb.Sheets[i].Name == name {
			return &wb.Sheets[i], nil
		}
	}

	available := make([]string, len(wb.Sheets))
	for i, s := range wb.Sheets {
		available[i] = s.Name
	}
	return nil, fmt.Errorf("sheet %q not found — available sheets: %v", name, available)
}

// ToCSV converts a sheet's data to CSV format.
func (s *Sheet) ToCSV() string {
	var result string
	for _, row := range s.Rows {
		for j, cell := range row {
			if j > 0 {
				result += ","
			}
			// Quote cells that contain commas or quotes
			if needsQuoting(cell) {
				result += "\"" + escapeCSV(cell) + "\""
			} else {
				result += cell
			}
		}
		result += "\n"
	}
	return result
}

func needsQuoting(s string) bool {
	for _, c := range s {
		if c == ',' || c == '"' || c == '\n' || c == '\r' {
			return true
		}
	}
	return false
}

func escapeCSV(s string) string {
	result := ""
	for _, c := range s {
		if c == '"' {
			result += "\"\""
		} else {
			result += string(c)
		}
	}
	return result
}

// RowCount returns the total number of data rows (excluding empty rows).
func (s *Sheet) RowCount() int {
	count := 0
	for _, row := range s.Rows {
		hasData := false
		for _, cell := range row {
			if cell != "" {
				hasData = true
				break
			}
		}
		if hasData {
			count++
		}
	}
	return count
}
