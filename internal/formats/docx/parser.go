// Package docx provides parsing and writing capabilities for .docx (OOXML) files.
package docx

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"strings"
)

// NodeType identifies the kind of content node in a document.
type NodeType int

const (
	// NodeParagraph represents a text paragraph.
	NodeParagraph NodeType = iota
	// NodeHeading represents a heading paragraph with a level (1-9).
	NodeHeading
	// NodeTable represents a table with rows and cells.
	NodeTable
	// NodeListItem represents a list item (bulleted or numbered).
	NodeListItem
)

// Node represents a single structural element in a document.
type Node struct {
	Type     NodeType   `json:"type"`
	Text     string     `json:"text"`
	Level    int        `json:"level,omitempty"`    // Heading level (1-9) or list nesting level
	Style    string     `json:"style,omitempty"`    // Original OOXML style name
	Children []Node     `json:"children,omitempty"` // For tables: rows containing cells
	Runs     []Run      `json:"runs,omitempty"`     // Individual text runs with formatting
	ListInfo *ListInfo  `json:"listInfo,omitempty"` // List numbering info
}

// Run represents a contiguous run of text with consistent formatting.
type Run struct {
	Text   string `json:"text"`
	Bold   bool   `json:"bold,omitempty"`
	Italic bool   `json:"italic,omitempty"`
}

// ListInfo holds numbering details for list items.
type ListInfo struct {
	NumID string `json:"numId"`
	Level int    `json:"level"`
}

// Metadata holds document-level metadata extracted from core.xml.
type Metadata struct {
	Title       string `json:"title,omitempty"`
	Creator     string `json:"creator,omitempty"`
	Description string `json:"description,omitempty"`
	Created     string `json:"created,omitempty"`
	Modified    string `json:"modified,omitempty"`
}

// Document is the top-level parsed representation of a .docx file.
type Document struct {
	Nodes    []Node   `json:"nodes"`
	Metadata Metadata `json:"metadata"`
}

// OOXML internal types for unmarshalling

type xmlParagraph struct {
	Properties xmlParagraphProps `xml:"pPr"`
	Runs       []xmlRun          `xml:"r"`
	Hyperlinks []xmlHyperlink    `xml:"hyperlink"`
}

type xmlParagraphProps struct {
	Style   xmlStyleVal  `xml:"pStyle"`
	NumPr   xmlNumPr     `xml:"numPr"`
	Heading xmlStyleVal  `xml:"outlineLvl"`
}

type xmlStyleVal struct {
	Val string `xml:"val,attr"`
}

type xmlNumPr struct {
	ILevel xmlStyleVal `xml:"ilvl"`
	NumID  xmlStyleVal `xml:"numId"`
}

type xmlRun struct {
	Properties xmlRunProps `xml:"rPr"`
	Text       []xmlText  `xml:"t"`
}

type xmlRunProps struct {
	Bold   *struct{} `xml:"b"`
	Italic *struct{} `xml:"i"`
}

type xmlText struct {
	Space string `xml:"space,attr"`
	Value string `xml:",chardata"`
}

type xmlHyperlink struct {
	Runs []xmlRun `xml:"r"`
}

type xmlTable struct {
	Rows []xmlTableRow `xml:"tr"`
}

type xmlTableRow struct {
	Cells []xmlTableCell `xml:"tc"`
}

type xmlTableCell struct {
	Paragraphs []xmlParagraph `xml:"p"`
}

// Core properties XML types
type xmlCoreProperties struct {
	Title       string `xml:"title"`
	Creator     string `xml:"creator"`
	Description string `xml:"description"`
	Created     string `xml:"created"`
	Modified    string `xml:"modified"`
}

// ParseFile reads and parses a .docx file from the given path.
func ParseFile(path string) (*Document, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("file not found: %s — check that the path is correct", path)
		}
		if os.IsPermission(err) {
			return nil, fmt.Errorf("permission denied reading %s — check file permissions or close the file if it is open in another application", path)
		}
		return nil, fmt.Errorf("could not read %s: %w", path, err)
	}
	return Parse(data)
}

// Parse reads and parses a .docx file from the given byte slice.
func Parse(data []byte) (*Document, error) {
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("invalid .docx file — the file does not appear to be a valid ZIP archive: %w", err)
	}

	doc := &Document{}

	// Parse core properties (metadata) — non-fatal if missing
	_ = parseCoreProperties(reader, doc)

	// Parse document body
	if err := parseDocumentBody(reader, doc); err != nil {
		return nil, err
	}

	return doc, nil
}

// ParseReader reads and parses a .docx file from a reader.
func ParseReader(r io.Reader) (*Document, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("could not read input: %w", err)
	}
	return Parse(data)
}

func parseCoreProperties(reader *zip.Reader, doc *Document) error {
	for _, f := range reader.File {
		if f.Name == "docProps/core.xml" {
			rc, err := f.Open()
			if err != nil {
				return err
			}
			defer rc.Close()

			data, err := io.ReadAll(rc)
			if err != nil {
				return err
			}

			var props xmlCoreProperties
			if err := xml.Unmarshal(data, &props); err != nil {
				return err
			}

			doc.Metadata = Metadata(props)
			return nil
		}
	}
	return nil
}

func parseDocumentBody(reader *zip.Reader, doc *Document) error {
	for _, f := range reader.File {
		if f.Name == "word/document.xml" {
			rc, err := f.Open()
			if err != nil {
				return fmt.Errorf("could not open document.xml inside .docx archive: %w", err)
			}
			defer rc.Close()

			data, err := io.ReadAll(rc)
			if err != nil {
				return fmt.Errorf("could not read document.xml: %w", err)
			}

			return parseXMLBody(data, doc)
		}
	}
	return fmt.Errorf("invalid .docx file — missing word/document.xml")
}

func parseXMLBody(data []byte, doc *Document) error {
	// We need to parse the body element and iterate over its children.
	// Due to OOXML namespace complexity, we use a streaming approach.
	decoder := xml.NewDecoder(bytes.NewReader(data))

	// Find the body element
	for {
		tok, err := decoder.Token()
		if err == io.EOF {
			return fmt.Errorf("invalid .docx file — no body element found in document.xml")
		}
		if err != nil {
			return fmt.Errorf("XML parse error in document.xml: %w", err)
		}

		if se, ok := tok.(xml.StartElement); ok && se.Name.Local == "body" {
			break
		}
	}

	// Now parse children of body
	for {
		tok, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("XML parse error: %w", err)
		}

		se, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}

		switch se.Name.Local {
		case "p":
			node, err := decodeParagraph(decoder, se)
			if err != nil {
				return err
			}
			if node != nil {
				doc.Nodes = append(doc.Nodes, *node)
			}
		case "tbl":
			node, err := decodeTable(decoder, se)
			if err != nil {
				return err
			}
			if node != nil {
				doc.Nodes = append(doc.Nodes, *node)
			}
		default:
			// Skip unknown elements
			if err := decoder.Skip(); err != nil {
				return err
			}
		}
	}

	return nil
}

func decodeParagraph(decoder *xml.Decoder, start xml.StartElement) (*Node, error) {
	var p xmlParagraph
	if err := decoder.DecodeElement(&p, &start); err != nil {
		return nil, fmt.Errorf("could not parse paragraph: %w", err)
	}

	// Collect all runs including from hyperlinks
	allRuns := make([]xmlRun, 0, len(p.Runs))
	allRuns = append(allRuns, p.Runs...)
	for _, h := range p.Hyperlinks {
		allRuns = append(allRuns, h.Runs...)
	}

	// Build text and runs
	var textBuilder strings.Builder
	runs := make([]Run, 0, len(allRuns))

	for _, r := range allRuns {
		for _, t := range r.Text {
			textBuilder.WriteString(t.Value)
		}
		runText := ""
		for _, t := range r.Text {
			runText += t.Value
		}
		if runText != "" {
			runs = append(runs, Run{
				Text:   runText,
				Bold:   r.Properties.Bold != nil,
				Italic: r.Properties.Italic != nil,
			})
		}
	}

	text := textBuilder.String()

	// Skip empty paragraphs
	if strings.TrimSpace(text) == "" {
		return nil, nil
	}

	node := &Node{
		Type: NodeParagraph,
		Text: text,
		Runs: runs,
	}

	// Detect heading style
	styleName := p.Properties.Style.Val
	node.Style = styleName
	if strings.HasPrefix(styleName, "Heading") || strings.HasPrefix(styleName, "heading") {
		node.Type = NodeHeading
		// Extract level from style name like "Heading1", "Heading2"
		level := 1
		if len(styleName) > 7 {
			ch := styleName[7]
			if ch >= '1' && ch <= '9' {
				level = int(ch - '0')
			}
		}
		node.Level = level
	}

	// Detect outline level
	if p.Properties.Heading.Val != "" {
		node.Type = NodeHeading
		level := 1
		if len(p.Properties.Heading.Val) > 0 {
			ch := p.Properties.Heading.Val[0]
			if ch >= '0' && ch <= '9' {
				level = int(ch-'0') + 1
			}
		}
		node.Level = level
	}

	// Detect list items
	if p.Properties.NumPr.NumID.Val != "" {
		node.Type = NodeListItem
		level := 0
		if p.Properties.NumPr.ILevel.Val != "" {
			ch := p.Properties.NumPr.ILevel.Val[0]
			if ch >= '0' && ch <= '9' {
				level = int(ch - '0')
			}
		}
		node.Level = level
		node.ListInfo = &ListInfo{
			NumID: p.Properties.NumPr.NumID.Val,
			Level: level,
		}
	}

	return node, nil
}

func decodeTable(decoder *xml.Decoder, start xml.StartElement) (*Node, error) {
	var t xmlTable
	if err := decoder.DecodeElement(&t, &start); err != nil {
		return nil, fmt.Errorf("could not parse table: %w", err)
	}

	node := &Node{
		Type:     NodeTable,
		Children: make([]Node, 0, len(t.Rows)),
	}

	for _, row := range t.Rows {
		rowNode := Node{
			Children: make([]Node, 0, len(row.Cells)),
		}
		for _, cell := range row.Cells {
			var cellTexts []string
			for _, p := range cell.Paragraphs {
				var text string
				for _, r := range p.Runs {
					for _, t := range r.Text {
						text += t.Value
					}
				}
				if text != "" {
					cellTexts = append(cellTexts, text)
				}
			}
			rowNode.Children = append(rowNode.Children, Node{
				Type: NodeParagraph,
				Text: strings.Join(cellTexts, "\n"),
			})
		}
		node.Children = append(node.Children, rowNode)
	}

	return node, nil
}

// PlainText returns the document content as plain text with section headers.
func (d *Document) PlainText() string {
	var b strings.Builder
	for _, n := range d.Nodes {
		writeNodePlainText(&b, n, 0)
	}
	return b.String()
}

func writeNodePlainText(b *strings.Builder, n Node, indent int) {
	prefix := strings.Repeat("  ", indent)
	switch n.Type {
	case NodeHeading:
		b.WriteString("\n")
		b.WriteString(prefix)
		b.WriteString(strings.Repeat("#", n.Level))
		b.WriteString(" ")
		b.WriteString(n.Text)
		b.WriteString("\n\n")
	case NodeParagraph:
		b.WriteString(prefix)
		b.WriteString(n.Text)
		b.WriteString("\n")
	case NodeListItem:
		b.WriteString(prefix)
		b.WriteString("- ")
		b.WriteString(n.Text)
		b.WriteString("\n")
	case NodeTable:
		for _, row := range n.Children {
			b.WriteString(prefix)
			cells := make([]string, 0, len(row.Children))
			for _, cell := range row.Children {
				cells = append(cells, cell.Text)
			}
			b.WriteString("| ")
			b.WriteString(strings.Join(cells, " | "))
			b.WriteString(" |")
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}
}

// Markdown returns the document content formatted as Markdown.
func (d *Document) Markdown() string {
	var b strings.Builder
	for _, n := range d.Nodes {
		writeNodeMarkdown(&b, n)
	}
	return b.String()
}

func writeNodeMarkdown(b *strings.Builder, n Node) {
	switch n.Type {
	case NodeHeading:
		b.WriteString(strings.Repeat("#", n.Level))
		b.WriteString(" ")
		writeRunsMarkdown(b, n)
		b.WriteString("\n\n")
	case NodeParagraph:
		writeRunsMarkdown(b, n)
		b.WriteString("\n\n")
	case NodeListItem:
		b.WriteString(strings.Repeat("  ", n.Level))
		b.WriteString("- ")
		writeRunsMarkdown(b, n)
		b.WriteString("\n")
	case NodeTable:
		if len(n.Children) == 0 {
			return
		}
		// Header row
		if len(n.Children) > 0 {
			row := n.Children[0]
			b.WriteString("| ")
			cells := make([]string, 0, len(row.Children))
			for _, cell := range row.Children {
				cells = append(cells, cell.Text)
			}
			b.WriteString(strings.Join(cells, " | "))
			b.WriteString(" |\n")
			// Separator
			b.WriteString("|")
			for range row.Children {
				b.WriteString(" --- |")
			}
			b.WriteString("\n")
		}
		// Data rows
		for i := 1; i < len(n.Children); i++ {
			row := n.Children[i]
			b.WriteString("| ")
			cells := make([]string, 0, len(row.Children))
			for _, cell := range row.Children {
				cells = append(cells, cell.Text)
			}
			b.WriteString(strings.Join(cells, " | "))
			b.WriteString(" |\n")
		}
		b.WriteString("\n")
	}
}

func writeRunsMarkdown(b *strings.Builder, n Node) {
	if len(n.Runs) == 0 {
		b.WriteString(n.Text)
		return
	}
	for _, r := range n.Runs {
		text := r.Text
		if r.Bold && r.Italic {
			b.WriteString("***")
			b.WriteString(text)
			b.WriteString("***")
		} else if r.Bold {
			b.WriteString("**")
			b.WriteString(text)
			b.WriteString("**")
		} else if r.Italic {
			b.WriteString("*")
			b.WriteString(text)
			b.WriteString("*")
		} else {
			b.WriteString(text)
		}
	}
}

// WordCount returns the total number of words across all text nodes.
func (d *Document) WordCount() int {
	count := 0
	for _, n := range d.Nodes {
		count += countWords(n)
	}
	return count
}

func countWords(n Node) int {
	count := len(strings.Fields(n.Text))
	for _, child := range n.Children {
		count += countWords(child)
	}
	return count
}

// Paragraphs returns just the text strings from all paragraph and heading nodes.
func (d *Document) Paragraphs() []string {
	var result []string
	for _, n := range d.Nodes {
		collectParagraphs(n, &result)
	}
	return result
}

func collectParagraphs(n Node, result *[]string) {
	switch n.Type {
	case NodeParagraph, NodeHeading, NodeListItem:
		if n.Text != "" {
			*result = append(*result, n.Text)
		}
	case NodeTable:
		for _, row := range n.Children {
			for _, cell := range row.Children {
				if cell.Text != "" {
					*result = append(*result, cell.Text)
				}
			}
		}
	}
}
