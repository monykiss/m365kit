package report

import (
	"archive/zip"
	"bytes"
	"encoding/csv"
	"encoding/json"
	"encoding/xml"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// makeDocx creates a minimal .docx with the given body content.
func makeDocx(bodyContent string) []byte {
	buf := new(bytes.Buffer)
	zw := zip.NewWriter(buf)

	w, _ := zw.Create("[Content_Types].xml")
	w.Write([]byte(xml.Header + `<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
  <Default Extension="xml" ContentType="application/xml"/>
  <Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>
</Types>`))

	w, _ = zw.Create("_rels/.rels")
	w.Write([]byte(xml.Header + `<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/>
</Relationships>`))

	w, _ = zw.Create("word/document.xml")
	w.Write([]byte(xml.Header + `<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:body>` +
		bodyContent +
		`</w:body></w:document>`))

	zw.Close()
	return buf.Bytes()
}

func makeCSV(t *testing.T, dir string, headers []string, rows [][]string) string {
	t.Helper()
	path := filepath.Join(dir, "data.csv")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	w := csv.NewWriter(f)
	w.Write(headers)
	for _, row := range rows {
		w.Write(row)
	}
	w.Flush()
	return path
}

func TestLoadCSV(t *testing.T) {
	dir := t.TempDir()
	path := makeCSV(t, dir, []string{"name", "amount"}, [][]string{
		{"Alice", "100"},
		{"Bob", "200"},
		{"Charlie", "150"},
	})

	ds, err := LoadData(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(ds.Columns) != 2 {
		t.Errorf("expected 2 columns, got %d", len(ds.Columns))
	}
	if len(ds.Rows) != 3 {
		t.Errorf("expected 3 rows, got %d", len(ds.Rows))
	}
	if ds.Rows[0]["name"] != "Alice" {
		t.Errorf("expected Alice, got %q", ds.Rows[0]["name"])
	}
}

func TestLoadJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "data.json")

	data := []map[string]any{
		{"name": "Alice", "amount": 100},
		{"name": "Bob", "amount": 200},
	}
	jsonData, _ := json.Marshal(data)
	os.WriteFile(path, jsonData, 0644)

	ds, err := LoadData(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(ds.Rows) != 2 {
		t.Errorf("expected 2 rows, got %d", len(ds.Rows))
	}
}

func TestLoadJSONSingleObject(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "data.json")

	data := map[string]any{"name": "Alice", "amount": 100}
	jsonData, _ := json.Marshal(data)
	os.WriteFile(path, jsonData, 0644)

	ds, err := LoadData(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(ds.Rows) != 1 {
		t.Errorf("expected 1 row, got %d", len(ds.Rows))
	}
}

func TestComputeAggregates(t *testing.T) {
	ds := &DataSource{
		Columns: []string{"revenue", "name"},
		Rows: []map[string]string{
			{"revenue": "100", "name": "Alice"},
			{"revenue": "200", "name": "Bob"},
			{"revenue": "300", "name": "Charlie"},
		},
	}

	agg := ComputeAggregates(ds)

	if agg["sum_revenue"] != "600" {
		t.Errorf("sum_revenue = %q, want 600", agg["sum_revenue"])
	}
	if agg["avg_revenue"] != "200" {
		t.Errorf("avg_revenue = %q, want 200", agg["avg_revenue"])
	}
	if agg["min_revenue"] != "100" {
		t.Errorf("min_revenue = %q, want 100", agg["min_revenue"])
	}
	if agg["max_revenue"] != "300" {
		t.Errorf("max_revenue = %q, want 300", agg["max_revenue"])
	}
	if agg["count_revenue"] != "3" {
		t.Errorf("count_revenue = %q, want 3", agg["count_revenue"])
	}

	// Non-numeric column should not appear
	if _, ok := agg["sum_name"]; ok {
		t.Error("non-numeric column 'name' should not have aggregates")
	}
}

func TestComputeAggregatesDecimal(t *testing.T) {
	ds := &DataSource{
		Columns: []string{"price"},
		Rows: []map[string]string{
			{"price": "10.5"},
			{"price": "20.3"},
		},
	}

	agg := ComputeAggregates(ds)
	if agg["sum_price"] != "30.80" {
		t.Errorf("sum_price = %q, want 30.80", agg["sum_price"])
	}
	if agg["avg_price"] != "15.40" {
		t.Errorf("avg_price = %q, want 15.40", agg["avg_price"])
	}
}

func TestSanitizeVarName(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"Revenue", "revenue"},
		{"Total Amount", "total_amount"},
		{"cost-per-unit", "cost_per_unit"},
		{"Hello World!", "hello_world"},
		{"123abc", "123abc"},
	}
	for _, tt := range tests {
		got := sanitizeVarName(tt.in)
		if got != tt.want {
			t.Errorf("sanitizeVarName(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestFormatNumber(t *testing.T) {
	tests := []struct {
		in   float64
		want string
	}{
		{100, "100"},
		{100.5, "100.50"},
		{0, "0"},
		{1234.56, "1234.56"},
	}
	for _, tt := range tests {
		got := formatNumber(tt.in)
		if got != tt.want {
			t.Errorf("formatNumber(%v) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestGenerateReport(t *testing.T) {
	dir := t.TempDir()

	// Create template
	body := `<w:p><w:r><w:t>Total revenue: {{sum_revenue}}. Average: {{avg_revenue}}. Rows: {{row_count}}.</w:t></w:r></w:p>`
	templatePath := filepath.Join(dir, "template.docx")
	os.WriteFile(templatePath, makeDocx(body), 0644)

	// Create CSV data
	dataPath := makeCSV(t, dir, []string{"name", "revenue"}, [][]string{
		{"Alice", "1000"},
		{"Bob", "2000"},
		{"Charlie", "3000"},
	})

	outputPath := filepath.Join(dir, "report.docx")

	result, err := Generate(GenerateOptions{
		TemplatePath: templatePath,
		DataPath:     dataPath,
		OutputPath:   outputPath,
	})
	if err != nil {
		t.Fatal(err)
	}

	if result.DataRows != 3 {
		t.Errorf("expected 3 data rows, got %d", result.DataRows)
	}
	if result.VariablesApplied != 3 {
		t.Errorf("expected 3 applied, got %d", result.VariablesApplied)
	}

	// Verify output content
	data, _ := os.ReadFile(outputPath)
	reader, _ := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	for _, f := range reader.File {
		if f.Name == "word/document.xml" {
			rc, _ := f.Open()
			content, _ := io.ReadAll(rc)
			rc.Close()
			text := string(content)
			if !strings.Contains(text, "6000") {
				t.Error("expected sum_revenue=6000 in output")
			}
			if !strings.Contains(text, "2000") {
				t.Error("expected avg_revenue=2000 in output")
			}
			if !strings.Contains(text, "Rows: 3") {
				t.Error("expected row_count=3 in output")
			}
		}
	}
}

func TestGenerateWithExtraValues(t *testing.T) {
	dir := t.TempDir()

	body := `<w:p><w:r><w:t>Report by {{author}}. Total: {{sum_amount}}.</w:t></w:r></w:p>`
	templatePath := filepath.Join(dir, "template.docx")
	os.WriteFile(templatePath, makeDocx(body), 0644)

	dataPath := makeCSV(t, dir, []string{"amount"}, [][]string{{"50"}, {"100"}})
	outputPath := filepath.Join(dir, "report.docx")

	result, err := Generate(GenerateOptions{
		TemplatePath: templatePath,
		DataPath:     dataPath,
		OutputPath:   outputPath,
		ExtraValues:  map[string]string{"author": "Alice"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.VariablesApplied != 2 {
		t.Errorf("expected 2 applied, got %d", result.VariablesApplied)
	}
}

func TestPreviewVariables(t *testing.T) {
	dir := t.TempDir()
	dataPath := makeCSV(t, dir, []string{"score"}, [][]string{{"80"}, {"90"}, {"100"}})

	vars, err := PreviewVariables(dataPath, map[string]string{"title": "Report"})
	if err != nil {
		t.Fatal(err)
	}
	if vars["sum_score"] != "270" {
		t.Errorf("sum_score = %q, want 270", vars["sum_score"])
	}
	if vars["title"] != "Report" {
		t.Errorf("title = %q, want Report", vars["title"])
	}
	if vars["row_count"] != "3" {
		t.Errorf("row_count = %q, want 3", vars["row_count"])
	}
}

func TestUnsupportedDataFormat(t *testing.T) {
	_, err := LoadData("data.xyz")
	if err == nil {
		t.Error("expected error for unsupported format")
	}
}
