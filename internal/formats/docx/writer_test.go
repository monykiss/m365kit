package docx

import (
	"archive/zip"
	"bytes"
	"testing"
)

func TestWriteDocumentValidZIP(t *testing.T) {
	doc := &Document{
		Nodes: []Node{
			{Type: NodeHeading, Text: "Heading One", Level: 1},
			{Type: NodeHeading, Text: "Heading Two", Level: 2},
			{Type: NodeParagraph, Text: "First paragraph content."},
			{Type: NodeParagraph, Text: "Second paragraph content."},
			{Type: NodeParagraph, Text: "Third paragraph content."},
		},
	}

	data, err := WriteDocument(doc)
	if err != nil {
		t.Fatalf("WriteDocument failed: %v", err)
	}

	// Verify it's a valid ZIP
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatalf("output is not a valid ZIP: %v", err)
	}

	// Verify required files exist
	required := map[string]bool{
		"[Content_Types].xml":            false,
		"_rels/.rels":                    false,
		"word/document.xml":              false,
		"word/_rels/document.xml.rels":   false,
	}

	for _, f := range reader.File {
		if _, ok := required[f.Name]; ok {
			required[f.Name] = true
		}
	}

	for name, found := range required {
		if !found {
			t.Errorf("missing required file in .docx: %s", name)
		}
	}
}

func TestWriteDocumentRoundTrip(t *testing.T) {
	original := &Document{
		Nodes: []Node{
			{Type: NodeHeading, Text: "Title", Level: 1},
			{Type: NodeHeading, Text: "Section", Level: 2},
			{Type: NodeParagraph, Text: "Paragraph one."},
			{Type: NodeParagraph, Text: "Paragraph two."},
			{Type: NodeParagraph, Text: "Paragraph three."},
		},
	}

	data, err := WriteDocument(original)
	if err != nil {
		t.Fatalf("WriteDocument failed: %v", err)
	}

	parsed, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(parsed.Nodes) != len(original.Nodes) {
		t.Fatalf("round-trip: expected %d nodes, got %d", len(original.Nodes), len(parsed.Nodes))
	}

	for i, node := range parsed.Nodes {
		if node.Text != original.Nodes[i].Text {
			t.Errorf("node %d: expected text %q, got %q", i, original.Nodes[i].Text, node.Text)
		}
	}
}

func TestWriteDocumentWithFormattedRuns(t *testing.T) {
	doc := &Document{
		Nodes: []Node{
			{
				Type: NodeParagraph,
				Text: "Hello bold world",
				Runs: []Run{
					{Text: "Hello ", Bold: false},
					{Text: "bold", Bold: true},
					{Text: " world", Bold: false},
				},
			},
		},
	}

	data, err := WriteDocument(doc)
	if err != nil {
		t.Fatalf("WriteDocument failed: %v", err)
	}

	parsed, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(parsed.Nodes) != 1 {
		t.Fatalf("expected 1 node, got %d", len(parsed.Nodes))
	}

	if parsed.Nodes[0].Text != "Hello bold world" {
		t.Errorf("expected 'Hello bold world', got %q", parsed.Nodes[0].Text)
	}
}

func TestWriteEmptyDocument(t *testing.T) {
	doc := &Document{}

	data, err := WriteDocument(doc)
	if err != nil {
		t.Fatalf("WriteDocument failed: %v", err)
	}

	if len(data) == 0 {
		t.Fatal("WriteDocument returned empty data")
	}

	// Should still be a valid ZIP
	_, err = zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatalf("empty document is not a valid ZIP: %v", err)
	}
}
