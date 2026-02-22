package benchmarks

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/klytics/m365kit/internal/formats/convert"
	"github.com/klytics/m365kit/internal/formats/docx"
	"github.com/klytics/m365kit/internal/formats/xlsx"
)

var sampleDocx = filepath.Join("..", "testdata", "sample.docx")
var sampleXlsx = filepath.Join("..", "testdata", "sample.xlsx")

// --- DOCX Benchmarks ---

func BenchmarkDocxRead(b *testing.B) {
	if _, err := os.Stat(sampleDocx); os.IsNotExist(err) {
		b.Skip("sample.docx not found")
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := docx.ParseFile(sampleDocx)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDocxWrite(b *testing.B) {
	doc := &docx.Document{
		Nodes: []docx.Node{
			{Type: docx.NodeHeading, Text: "Benchmark Document", Level: 1},
			{Type: docx.NodeParagraph, Text: "This is a test paragraph for benchmarking."},
			{Type: docx.NodeParagraph, Text: "Second paragraph with more content to simulate a real document."},
			{Type: docx.NodeListItem, Text: "First item"},
			{Type: docx.NodeListItem, Text: "Second item"},
			{Type: docx.NodeListItem, Text: "Third item"},
		},
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := docx.WriteDocument(doc)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDocxWriteLarge(b *testing.B) {
	nodes := make([]docx.Node, 100)
	for i := range nodes {
		nodes[i] = docx.Node{Type: docx.NodeParagraph, Text: "Lorem ipsum dolor sit amet, consectetur adipiscing elit. Sed do eiusmod tempor incididunt ut labore et dolore magna aliqua."}
	}
	doc := &docx.Document{Nodes: nodes}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := docx.WriteDocument(doc)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDocxRoundTrip(b *testing.B) {
	if _, err := os.Stat(sampleDocx); os.IsNotExist(err) {
		b.Skip("sample.docx not found")
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		doc, err := docx.ParseFile(sampleDocx)
		if err != nil {
			b.Fatal(err)
		}
		_, err = docx.WriteDocument(doc)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDocxPlainText(b *testing.B) {
	if _, err := os.Stat(sampleDocx); os.IsNotExist(err) {
		b.Skip("sample.docx not found")
	}
	doc, err := docx.ParseFile(sampleDocx)
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = doc.PlainText()
	}
}

func BenchmarkDocxMarkdown(b *testing.B) {
	if _, err := os.Stat(sampleDocx); os.IsNotExist(err) {
		b.Skip("sample.docx not found")
	}
	doc, err := docx.ParseFile(sampleDocx)
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = doc.Markdown()
	}
}

// --- XLSX Benchmarks ---

func BenchmarkXlsxRead(b *testing.B) {
	if _, err := os.Stat(sampleXlsx); os.IsNotExist(err) {
		b.Skip("sample.xlsx not found")
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := xlsx.ReadFile(sampleXlsx)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkXlsxWrite(b *testing.B) {
	wb := &xlsx.Workbook{
		Sheets: []xlsx.Sheet{
			{
				Name: "Data",
				Rows: [][]string{
					{"Name", "Value", "Category"},
					{"Alpha", "100", "A"},
					{"Beta", "200", "B"},
					{"Gamma", "300", "A"},
					{"Delta", "400", "C"},
				},
			},
		},
	}
	dir := b.TempDir()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := xlsx.WriteFile(wb, filepath.Join(dir, "bench.xlsx"))
		if err != nil {
			b.Fatal(err)
		}
	}
}

// --- Convert Benchmarks ---

func BenchmarkConvertDocxToMd(b *testing.B) {
	if _, err := os.Stat(sampleDocx); os.IsNotExist(err) {
		b.Skip("sample.docx not found")
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := convert.DocxToMarkdown(sampleDocx)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkConvertDocxToHTML(b *testing.B) {
	if _, err := os.Stat(sampleDocx); os.IsNotExist(err) {
		b.Skip("sample.docx not found")
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := convert.DocxToHTML(sampleDocx)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkConvertDocxToText(b *testing.B) {
	if _, err := os.Stat(sampleDocx); os.IsNotExist(err) {
		b.Skip("sample.docx not found")
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := convert.DocxToText(sampleDocx)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkConvertXlsxToCSV(b *testing.B) {
	if _, err := os.Stat(sampleXlsx); os.IsNotExist(err) {
		b.Skip("sample.xlsx not found")
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := convert.XlsxToCSV(sampleXlsx, "")
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkConvertXlsxToJSON(b *testing.B) {
	if _, err := os.Stat(sampleXlsx); os.IsNotExist(err) {
		b.Skip("sample.xlsx not found")
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := convert.XlsxToJSON(sampleXlsx, "")
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkConvertMdToDocx(b *testing.B) {
	md := `# Benchmark
This is a **bold** paragraph with *italic* text.

## Section Two
- Item 1
- Item 2
- Item 3

| Header | Value |
|--------|-------|
| A      | 100   |
| B      | 200   |
`
	dir := b.TempDir()
	mdPath := filepath.Join(dir, "bench.md")
	os.WriteFile(mdPath, []byte(md), 0644)

	outPath := filepath.Join(dir, "bench.docx")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := convert.MarkdownToDocx(md, outPath)
		if err != nil {
			b.Fatal(err)
		}
	}
}
