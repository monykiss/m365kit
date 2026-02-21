package xlsx

import (
	"fmt"

	"github.com/xuri/excelize/v2"
)

// WriteFile creates a new .xlsx file from the given workbook data.
func WriteFile(wb *Workbook, path string) error {
	f := excelize.NewFile()
	defer f.Close()

	for i, sheet := range wb.Sheets {
		sheetName := sheet.Name
		if sheetName == "" {
			sheetName = fmt.Sprintf("Sheet%d", i+1)
		}

		if i == 0 {
			// Rename default sheet
			defaultSheet := f.GetSheetName(0)
			if err := f.SetSheetName(defaultSheet, sheetName); err != nil {
				return fmt.Errorf("could not rename sheet: %w", err)
			}
		} else {
			if _, err := f.NewSheet(sheetName); err != nil {
				return fmt.Errorf("could not create sheet %q: %w", sheetName, err)
			}
		}

		for rowIdx, row := range sheet.Rows {
			for colIdx, cell := range row {
				cellName, err := excelize.CoordinatesToCellName(colIdx+1, rowIdx+1)
				if err != nil {
					return fmt.Errorf("invalid cell coordinates: %w", err)
				}
				if err := f.SetCellValue(sheetName, cellName, cell); err != nil {
					return fmt.Errorf("could not set cell %s: %w", cellName, err)
				}
			}
		}
	}

	if err := f.SaveAs(path); err != nil {
		return fmt.Errorf("could not save %s: %w", path, err)
	}

	return nil
}
