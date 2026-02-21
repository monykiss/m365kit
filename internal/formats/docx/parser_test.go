package docx

import (
	"testing"
)

func TestParseAndRoundTrip(t *testing.T) {
	// Create a document, write it, then parse it back
	original := &Document{
		Nodes: []Node{
			{Type: NodeHeading, Text: "Test Document", Level: 1},
			{Type: NodeParagraph, Text: "This is a test paragraph."},
			{Type: NodeParagraph, Text: "Second paragraph with more content."},
			{Type: NodeListItem, Text: "First item", Level: 0, ListInfo: &ListInfo{NumID: "1", Level: 0}},
			{Type: NodeListItem, Text: "Second item", Level: 0, ListInfo: &ListInfo{NumID: "1", Level: 0}},
		},
	}

	// Write to bytes
	data, err := WriteDocument(original)
	if err != nil {
		t.Fatalf("WriteDocument failed: %v", err)
	}

	if len(data) == 0 {
		t.Fatal("WriteDocument returned empty data")
	}

	// Parse back
	parsed, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(parsed.Nodes) != len(original.Nodes) {
		t.Fatalf("expected %d nodes, got %d", len(original.Nodes), len(parsed.Nodes))
	}

	for i, node := range parsed.Nodes {
		if node.Text != original.Nodes[i].Text {
			t.Errorf("node %d: expected text %q, got %q", i, original.Nodes[i].Text, node.Text)
		}
		if node.Type != original.Nodes[i].Type {
			t.Errorf("node %d: expected type %d, got %d", i, original.Nodes[i].Type, node.Type)
		}
	}
}

func TestPlainText(t *testing.T) {
	doc := &Document{
		Nodes: []Node{
			{Type: NodeHeading, Text: "Title", Level: 1},
			{Type: NodeParagraph, Text: "Body text here."},
		},
	}

	text := doc.PlainText()
	if text == "" {
		t.Fatal("PlainText returned empty string")
	}
	if !contains(text, "Title") {
		t.Error("PlainText missing heading text")
	}
	if !contains(text, "Body text here.") {
		t.Error("PlainText missing paragraph text")
	}
}

func TestMarkdown(t *testing.T) {
	doc := &Document{
		Nodes: []Node{
			{Type: NodeHeading, Text: "Title", Level: 1},
			{Type: NodeParagraph, Text: "Normal text.", Runs: []Run{
				{Text: "Normal ", Bold: false},
				{Text: "bold", Bold: true},
				{Text: " text.", Bold: false},
			}},
		},
	}

	md := doc.Markdown()
	if !contains(md, "# Title") {
		t.Error("Markdown missing heading markup")
	}
	if !contains(md, "**bold**") {
		t.Error("Markdown missing bold markup")
	}
}

func TestWordCount(t *testing.T) {
	doc := &Document{
		Nodes: []Node{
			{Type: NodeParagraph, Text: "one two three"},
			{Type: NodeParagraph, Text: "four five"},
		},
	}

	if wc := doc.WordCount(); wc != 5 {
		t.Errorf("expected word count 5, got %d", wc)
	}
}

func TestParagraphs(t *testing.T) {
	doc := &Document{
		Nodes: []Node{
			{Type: NodeHeading, Text: "Heading", Level: 1},
			{Type: NodeParagraph, Text: "Para 1"},
			{Type: NodeParagraph, Text: "Para 2"},
		},
	}

	paras := doc.Paragraphs()
	if len(paras) != 3 {
		t.Fatalf("expected 3 paragraphs, got %d", len(paras))
	}
}

func TestParseInvalidData(t *testing.T) {
	_, err := Parse([]byte("not a zip file"))
	if err == nil {
		t.Fatal("expected error for invalid data")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
