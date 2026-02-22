package convert

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/klytics/m365kit/internal/formats/docx"
	"github.com/klytics/m365kit/internal/formats/xlsx"
)

// createTestDocx creates a test .docx with given nodes.
func createTestDocx(t *testing.T, dir string, nodes []docx.Node) string {
	t.Helper()
	doc := &docx.Document{Nodes: nodes}
	data, err := docx.WriteDocument(doc)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "test.docx")
	os.WriteFile(path, data, 0644)
	return path
}

func TestDocxToMarkdownHeadings(t *testing.T) {
	dir := t.TempDir()
	path := createTestDocx(t, dir, []docx.Node{
		{Type: docx.NodeHeading, Level: 1, Text: "Title"},
		{Type: docx.NodeHeading, Level: 2, Text: "Subtitle"},
		{Type: docx.NodeParagraph, Text: "Body text"},
	})

	result, err := DocxToMarkdown(path)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(result, "# Title") {
		t.Error("expected # Title in output")
	}
	if !strings.Contains(result, "## Subtitle") {
		t.Error("expected ## Subtitle in output")
	}
	if !strings.Contains(result, "Body text") {
		t.Error("expected body text in output")
	}
}

func TestDocxToMarkdownBoldItalic(t *testing.T) {
	dir := t.TempDir()
	path := createTestDocx(t, dir, []docx.Node{
		{Type: docx.NodeParagraph, Text: "Hello world", Runs: []docx.Run{
			{Text: "Hello ", Bold: false},
			{Text: "world", Bold: true},
		}},
	})

	result, err := DocxToMarkdown(path)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(result, "**world**") {
		t.Errorf("expected **world**, got: %s", result)
	}
}

func TestDocxToMarkdownTable(t *testing.T) {
	dir := t.TempDir()
	path := createTestDocx(t, dir, []docx.Node{
		{Type: docx.NodeTable, Children: []docx.Node{
			{Children: []docx.Node{
				{Type: docx.NodeParagraph, Text: "Name"},
				{Type: docx.NodeParagraph, Text: "Age"},
			}},
			{Children: []docx.Node{
				{Type: docx.NodeParagraph, Text: "Alice"},
				{Type: docx.NodeParagraph, Text: "30"},
			}},
		}},
	})

	result, err := DocxToMarkdown(path)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(result, "| Name | Age |") {
		t.Errorf("expected markdown table header, got: %s", result)
	}
	if !strings.Contains(result, "| --- |") {
		t.Error("expected markdown table separator")
	}
}

func TestMarkdownToDocxHeadings(t *testing.T) {
	dir := t.TempDir()
	output := filepath.Join(dir, "output.docx")

	md := "## Section Title\n\nParagraph text.\n"
	if err := MarkdownToDocx(md, output); err != nil {
		t.Fatal(err)
	}

	// Read back and verify
	doc, err := docx.ParseFile(output)
	if err != nil {
		t.Fatal(err)
	}

	foundHeading := false
	for _, n := range doc.Nodes {
		if n.Type == docx.NodeHeading && strings.Contains(n.Text, "Section Title") {
			foundHeading = true
		}
	}
	if !foundHeading {
		t.Error("expected Heading2 in output docx")
	}
}

func TestMarkdownToDocxBold(t *testing.T) {
	dir := t.TempDir()
	output := filepath.Join(dir, "output.docx")

	md := "This is **bold** text.\n"
	if err := MarkdownToDocx(md, output); err != nil {
		t.Fatal(err)
	}

	doc, err := docx.ParseFile(output)
	if err != nil {
		t.Fatal(err)
	}

	foundBold := false
	for _, n := range doc.Nodes {
		for _, r := range n.Runs {
			if r.Bold && r.Text == "bold" {
				foundBold = true
			}
		}
	}
	if !foundBold {
		t.Error("expected bold run in output docx")
	}
}

func TestMarkdownToDocxList(t *testing.T) {
	dir := t.TempDir()
	output := filepath.Join(dir, "output.docx")

	md := "- Item one\n- Item two\n"
	if err := MarkdownToDocx(md, output); err != nil {
		t.Fatal(err)
	}

	doc, err := docx.ParseFile(output)
	if err != nil {
		t.Fatal(err)
	}

	listCount := 0
	for _, n := range doc.Nodes {
		if n.Type == docx.NodeListItem {
			listCount++
		}
	}
	if listCount != 2 {
		t.Errorf("expected 2 list items, got %d", listCount)
	}
}

func TestMarkdownRoundTrip(t *testing.T) {
	dir := t.TempDir()
	docxPath := filepath.Join(dir, "roundtrip.docx")

	md := "# Main Title\n\nSome paragraph.\n\n## Sub Section\n\nMore text.\n"
	if err := MarkdownToDocx(md, docxPath); err != nil {
		t.Fatal(err)
	}

	result, err := DocxToMarkdown(docxPath)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(result, "# Main Title") {
		t.Error("round-trip: expected # Main Title")
	}
	if !strings.Contains(result, "## Sub Section") {
		t.Error("round-trip: expected ## Sub Section")
	}
}

func TestDocxToHTMLValid(t *testing.T) {
	dir := t.TempDir()
	path := createTestDocx(t, dir, []docx.Node{
		{Type: docx.NodeHeading, Level: 1, Text: "Title"},
		{Type: docx.NodeParagraph, Text: "Paragraph"},
	})

	result, err := DocxToHTML(path)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(result, "<!DOCTYPE html>") {
		t.Error("expected <!DOCTYPE html>")
	}
	if !strings.Contains(result, "<h1>Title</h1>") {
		t.Error("expected <h1>Title</h1>")
	}
	if !strings.Contains(result, "<p>Paragraph</p>") {
		t.Error("expected <p>Paragraph</p>")
	}
}

func TestDocxToHTMLHeadingLevels(t *testing.T) {
	dir := t.TempDir()
	path := createTestDocx(t, dir, []docx.Node{
		{Type: docx.NodeHeading, Level: 1, Text: "H1"},
		{Type: docx.NodeHeading, Level: 2, Text: "H2"},
		{Type: docx.NodeHeading, Level: 3, Text: "H3"},
	})

	result, err := DocxToHTML(path)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(result, "<h1>H1</h1>") {
		t.Error("expected <h1>")
	}
	if !strings.Contains(result, "<h2>H2</h2>") {
		t.Error("expected <h2>")
	}
	if !strings.Contains(result, "<h3>H3</h3>") {
		t.Error("expected <h3>")
	}
}

func TestXlsxToMarkdownTable(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.xlsx")

	wb := &xlsx.Workbook{
		Sheets: []xlsx.Sheet{
			{Name: "Sheet1", Rows: [][]string{
				{"Name", "Score"},
				{"Alice", "95"},
				{"Bob", "87"},
			}},
		},
	}
	if err := xlsx.WriteFile(wb, path); err != nil {
		t.Fatal(err)
	}

	result, err := XlsxToMarkdown(path, "")
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(result, "| Name | Score |") {
		t.Errorf("expected header row, got: %s", result)
	}
	if !strings.Contains(result, "| --- |") {
		t.Error("expected separator row")
	}
	if !strings.Contains(result, "| Alice | 95 |") {
		t.Error("expected data row")
	}
}

func TestUnsupportedConversion(t *testing.T) {
	_, err := Convert("test.txt", "", "docx")
	if err == nil {
		t.Error("expected error for unsupported conversion")
	}
	if !strings.Contains(err.Error(), "unsupported") {
		t.Errorf("expected 'unsupported' in error, got: %s", err)
	}
}

func TestDetectFormat(t *testing.T) {
	tests := []struct {
		path, want string
	}{
		{"report.docx", "docx"},
		{"README.md", "md"},
		{"page.html", "html"},
		{"data.xlsx", "xlsx"},
		{"file.txt", "txt"},
		{"unknown.xyz", ""},
	}
	for _, tt := range tests {
		got := detectFormat(tt.path)
		if got != tt.want {
			t.Errorf("detectFormat(%q) = %q, want %q", tt.path, got, tt.want)
		}
	}
}

func TestHTMLToDocx(t *testing.T) {
	dir := t.TempDir()
	output := filepath.Join(dir, "output.docx")

	html := `<!DOCTYPE html><html><body><h1>Hello</h1><p>World</p></body></html>`
	if err := HTMLToDocx(html, output); err != nil {
		t.Fatal(err)
	}

	doc, err := docx.ParseFile(output)
	if err != nil {
		t.Fatal(err)
	}

	foundHeading := false
	for _, n := range doc.Nodes {
		if n.Type == docx.NodeHeading && strings.Contains(n.Text, "Hello") {
			foundHeading = true
		}
	}
	if !foundHeading {
		t.Error("expected heading 'Hello' in docx")
	}
}

// TestConvertDetectFormatUnknown verifies that Convert returns an error when
// the input file has an unrecognized extension.
func TestConvertDetectFormatUnknown(t *testing.T) {
	_, err := Convert("archive.zip", "", "txt")
	if err == nil {
		t.Fatal("expected error for unknown input extension")
	}
	if !strings.Contains(err.Error(), "could not detect input format") {
		t.Errorf("expected 'could not detect input format' in error, got: %s", err)
	}
}

// TestConvertDocxToTextContent creates a .docx with known content, converts it
// to plain text via Convert, and verifies the output contains the expected text.
func TestConvertDocxToTextContent(t *testing.T) {
	dir := t.TempDir()
	path := createTestDocx(t, dir, []docx.Node{
		{Type: docx.NodeHeading, Level: 1, Text: "Report Title"},
		{Type: docx.NodeParagraph, Text: "This is the body of the report."},
		{Type: docx.NodeParagraph, Text: "It has multiple paragraphs."},
	})

	result, err := Convert(path, "", "txt")
	if err != nil {
		t.Fatalf("Convert docx to txt failed: %v", err)
	}

	if !strings.Contains(result, "Report Title") {
		t.Error("expected 'Report Title' in text output")
	}
	if !strings.Contains(result, "body of the report") {
		t.Error("expected 'body of the report' in text output")
	}
	if !strings.Contains(result, "multiple paragraphs") {
		t.Error("expected 'multiple paragraphs' in text output")
	}
}
