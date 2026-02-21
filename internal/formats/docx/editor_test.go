package docx

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEditBytesReplacesText(t *testing.T) {
	// Create a document with known text
	doc := &Document{
		Nodes: []Node{
			{Type: NodeHeading, Text: "M365Kit Report", Level: 1},
			{Type: NodeParagraph, Text: "M365Kit is a great tool. Use M365Kit daily."},
		},
	}

	original, err := WriteDocument(doc)
	if err != nil {
		t.Fatalf("WriteDocument failed: %v", err)
	}

	// Edit: replace M365Kit → EditedKit
	edited, count, err := EditBytes(original, map[string]string{"M365Kit": "EditedKit"})
	if err != nil {
		t.Fatalf("EditBytes failed: %v", err)
	}

	if count != 3 {
		t.Errorf("expected 3 replacements, got %d", count)
	}

	// Parse and verify
	parsed, err := Parse(edited)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	for _, node := range parsed.Nodes {
		if containsString(node.Text, "M365Kit") {
			t.Errorf("found unreplaced 'M365Kit' in text: %q", node.Text)
		}
		if node.Type == NodeHeading && !containsString(node.Text, "EditedKit") {
			t.Errorf("expected 'EditedKit' in heading, got %q", node.Text)
		}
	}
}

func TestEditBytesNoMatch(t *testing.T) {
	doc := &Document{
		Nodes: []Node{
			{Type: NodeParagraph, Text: "Hello World"},
		},
	}

	original, err := WriteDocument(doc)
	if err != nil {
		t.Fatalf("WriteDocument failed: %v", err)
	}

	_, count, err := EditBytes(original, map[string]string{"NOTFOUND": "replacement"})
	if err != nil {
		t.Fatalf("EditBytes failed: %v", err)
	}

	if count != 0 {
		t.Errorf("expected 0 replacements, got %d", count)
	}
}

func TestEditFileEndToEnd(t *testing.T) {
	// Create a temporary source file
	doc := &Document{
		Nodes: []Node{
			{Type: NodeParagraph, Text: "Replace PLACEHOLDER with value"},
		},
	}

	data, err := WriteDocument(doc)
	if err != nil {
		t.Fatalf("WriteDocument failed: %v", err)
	}

	tmpDir := t.TempDir()
	srcPath := filepath.Join(tmpDir, "source.docx")
	outPath := filepath.Join(tmpDir, "output.docx")

	if err := os.WriteFile(srcPath, data, 0644); err != nil {
		t.Fatalf("could not write source: %v", err)
	}

	result, err := EditFile(srcPath, map[string]string{"PLACEHOLDER": "ACTUAL"}, outPath)
	if err != nil {
		t.Fatalf("EditFile failed: %v", err)
	}

	if result.ReplacementsMade != 1 {
		t.Errorf("expected 1 replacement, got %d", result.ReplacementsMade)
	}

	// Parse the output
	parsed, err := ParseFile(outPath)
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}

	if len(parsed.Nodes) == 0 {
		t.Fatal("parsed doc has no nodes")
	}

	if !containsString(parsed.Nodes[0].Text, "ACTUAL") {
		t.Errorf("expected 'ACTUAL' in output, got %q", parsed.Nodes[0].Text)
	}
}

func TestEditMultipleReplacements(t *testing.T) {
	doc := &Document{
		Nodes: []Node{
			{Type: NodeParagraph, Text: "Year: 2023, Company: OldCo"},
		},
	}

	original, err := WriteDocument(doc)
	if err != nil {
		t.Fatalf("WriteDocument failed: %v", err)
	}

	replacements := map[string]string{
		"2023":  "2024",
		"OldCo": "NewCo",
	}

	edited, count, err := EditBytes(original, replacements)
	if err != nil {
		t.Fatalf("EditBytes failed: %v", err)
	}

	if count != 2 {
		t.Errorf("expected 2 replacements, got %d", count)
	}

	parsed, err := Parse(edited)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(parsed.Nodes) == 0 {
		t.Fatal("parsed doc has no nodes")
	}

	text := parsed.Nodes[0].Text
	if !containsString(text, "2024") || !containsString(text, "NewCo") {
		t.Errorf("expected '2024' and 'NewCo' in text, got %q", text)
	}
}

func containsString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
