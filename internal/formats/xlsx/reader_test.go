package xlsx

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteAndRead(t *testing.T) {
	// Create a workbook, write it, then read it back
	original := &Workbook{
		Sheets: []Sheet{
			{
				Name: "TestSheet",
				Rows: [][]string{
					{"Name", "Age", "City"},
					{"Alice", "30", "New York"},
					{"Bob", "25", "San Francisco"},
				},
			},
		},
	}

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.xlsx")

	if err := WriteFile(original, path); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatal("WriteFile did not create the file")
	}

	// Read back
	wb, err := ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}

	if len(wb.Sheets) != 1 {
		t.Fatalf("expected 1 sheet, got %d", len(wb.Sheets))
	}

	sheet := wb.Sheets[0]
	if sheet.Name != "TestSheet" {
		t.Errorf("expected sheet name 'TestSheet', got %q", sheet.Name)
	}

	if len(sheet.Rows) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(sheet.Rows))
	}

	if sheet.Rows[1][0] != "Alice" {
		t.Errorf("expected 'Alice', got %q", sheet.Rows[1][0])
	}
}

func TestSheetToCSV(t *testing.T) {
	sheet := Sheet{
		Name: "Test",
		Rows: [][]string{
			{"Name", "Value"},
			{"Test", "123"},
		},
	}

	csv := sheet.ToCSV()
	expected := "Name,Value\nTest,123\n"
	if csv != expected {
		t.Errorf("expected CSV %q, got %q", expected, csv)
	}
}

func TestGetSheet(t *testing.T) {
	wb := &Workbook{
		Sheets: []Sheet{
			{Name: "One"},
			{Name: "Two"},
		},
	}

	s, err := wb.GetSheet("Two")
	if err != nil {
		t.Fatalf("GetSheet failed: %v", err)
	}
	if s.Name != "Two" {
		t.Errorf("expected 'Two', got %q", s.Name)
	}

	_, err = wb.GetSheet("Missing")
	if err == nil {
		t.Error("expected error for missing sheet")
	}
}

func TestRowCount(t *testing.T) {
	sheet := Sheet{
		Rows: [][]string{
			{"A", "B"},
			{"C", "D"},
			{"", ""},
		},
	}

	if rc := sheet.RowCount(); rc != 2 {
		t.Errorf("expected 2 non-empty rows, got %d", rc)
	}
}

func TestReadFileNotFound(t *testing.T) {
	_, err := ReadFile("/nonexistent/file.xlsx")
	if err == nil {
		t.Error("expected error for missing file")
	}
}
