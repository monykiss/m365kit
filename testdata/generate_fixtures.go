//go:build ignore

// This program generates test fixture files for M365Kit.
package main

import (
	"fmt"
	"os"

	"github.com/klytics/m365kit/internal/formats/docx"
	"github.com/klytics/m365kit/internal/formats/xlsx"
)

func main() {
	if err := generateDocx(); err != nil {
		fmt.Fprintf(os.Stderr, "Error generating sample.docx: %v\n", err)
		os.Exit(1)
	}

	if err := generateXlsx(); err != nil {
		fmt.Fprintf(os.Stderr, "Error generating sample.xlsx: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Test fixtures generated successfully.")
}

func generateDocx() error {
	doc := &docx.Document{
		Nodes: []docx.Node{
			{Type: docx.NodeHeading, Text: "M365Kit Sample Document", Level: 1},
			{Type: docx.NodeParagraph, Text: "This is a sample document for testing the M365Kit CLI tool. It contains various formatting elements to verify parser correctness."},
			{Type: docx.NodeHeading, Text: "Project Overview", Level: 2},
			{Type: docx.NodeParagraph, Text: "M365Kit provides a unified programmatic interface to Microsoft 365 document formats. It supports reading, writing, and AI-powered analysis of .docx, .xlsx, and .pptx files."},
			{Type: docx.NodeHeading, Text: "Key Features", Level: 2},
			{Type: docx.NodeListItem, Text: "Parse Word documents with full OOXML support", Level: 0, ListInfo: &docx.ListInfo{NumID: "1", Level: 0}},
			{Type: docx.NodeListItem, Text: "Read Excel spreadsheets with multi-sheet support", Level: 0, ListInfo: &docx.ListInfo{NumID: "1", Level: 0}},
			{Type: docx.NodeListItem, Text: "AI-powered document summarization and analysis", Level: 0, ListInfo: &docx.ListInfo{NumID: "1", Level: 0}},
			{Type: docx.NodeListItem, Text: "Pipeline YAML workflows for automation", Level: 0, ListInfo: &docx.ListInfo{NumID: "1", Level: 0}},
			{Type: docx.NodeHeading, Text: "Technical Details", Level: 2},
			{Type: docx.NodeParagraph, Text: "The project is built with Go for the CLI and TypeScript for the PPTX generation engine. It uses the Cobra framework for command-line parsing and supports multiple AI providers including Anthropic, OpenAI, and Ollama."},
			{Type: docx.NodeHeading, Text: "Next Steps", Level: 2},
			{Type: docx.NodeParagraph, Text: "The team will focus on expanding format support, improving AI integration, and building out the pipeline execution engine. Target release date is Q1 2025."},
		},
	}

	data, err := docx.WriteDocument(doc)
	if err != nil {
		return err
	}

	return os.WriteFile("testdata/sample.docx", data, 0644)
}

func generateXlsx() error {
	wb := &xlsx.Workbook{
		Sheets: []xlsx.Sheet{
			{
				Name: "Revenue",
				Rows: [][]string{
					{"Quarter", "Product", "Revenue", "Growth"},
					{"Q1 2024", "Enterprise", "1250000", "12%"},
					{"Q1 2024", "SMB", "450000", "8%"},
					{"Q1 2024", "Consumer", "320000", "15%"},
					{"Q2 2024", "Enterprise", "1380000", "10%"},
					{"Q2 2024", "SMB", "520000", "16%"},
					{"Q2 2024", "Consumer", "350000", "9%"},
					{"Q3 2024", "Enterprise", "1450000", "5%"},
					{"Q3 2024", "SMB", "580000", "12%"},
					{"Q3 2024", "Consumer", "410000", "17%"},
					{"Q4 2024", "Enterprise", "1620000", "12%"},
					{"Q4 2024", "SMB", "640000", "10%"},
					{"Q4 2024", "Consumer", "480000", "17%"},
				},
			},
			{
				Name: "Summary",
				Rows: [][]string{
					{"Metric", "Value"},
					{"Total Revenue", "8450000"},
					{"YoY Growth", "12.3%"},
					{"Top Product", "Enterprise"},
					{"Fastest Growth", "Consumer"},
				},
			},
		},
	}

	return xlsx.WriteFile(wb, "testdata/sample.xlsx")
}
