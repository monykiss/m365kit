package convert

import (
	"os"
	"regexp"
	"strings"

	"github.com/klytics/m365kit/internal/formats/docx"
)

var orderedListRe = regexp.MustCompile(`^\d+\.\s`)

// MarkdownToDocx converts a Markdown string to a .docx file.
func MarkdownToDocx(input, outputPath string) error {
	doc := parseMarkdown(input)
	data, err := docx.WriteDocument(doc)
	if err != nil {
		return err
	}
	return os.WriteFile(outputPath, data, 0644)
}

// HTMLToDocx converts an HTML string to a .docx file.
// Basic support: strips tags, preserves text structure.
func HTMLToDocx(input, outputPath string) error {
	doc := parseHTML(input)
	data, err := docx.WriteDocument(doc)
	if err != nil {
		return err
	}
	return os.WriteFile(outputPath, data, 0644)
}

func parseMarkdown(input string) *docx.Document {
	doc := &docx.Document{}
	lines := strings.Split(input, "\n")

	i := 0
	for i < len(lines) {
		line := lines[i]
		trimmed := strings.TrimSpace(line)

		// Skip empty lines
		if trimmed == "" {
			i++
			continue
		}

		// Horizontal rule
		if trimmed == "---" || trimmed == "***" || trimmed == "___" {
			i++
			continue
		}

		// Headings
		if strings.HasPrefix(trimmed, "#") {
			level := 0
			for _, c := range trimmed {
				if c == '#' {
					level++
				} else {
					break
				}
			}
			if level > 0 && level <= 6 {
				text := strings.TrimSpace(trimmed[level:])
				runs := parseInlineFormatting(text)
				doc.Nodes = append(doc.Nodes, docx.Node{
					Type:  docx.NodeHeading,
					Text:  stripFormatting(text),
					Level: level,
					Runs:  runs,
				})
				i++
				continue
			}
		}

		// Unordered list
		if strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* ") {
			text := trimmed[2:]
			runs := parseInlineFormatting(text)
			doc.Nodes = append(doc.Nodes, docx.Node{
				Type: docx.NodeListItem,
				Text: stripFormatting(text),
				Runs: runs,
			})
			i++
			continue
		}

		// Ordered list
		if orderedListRe.MatchString(trimmed) {
			idx := strings.Index(trimmed, ". ")
			text := trimmed[idx+2:]
			runs := parseInlineFormatting(text)
			doc.Nodes = append(doc.Nodes, docx.Node{
				Type: docx.NodeListItem,
				Text: stripFormatting(text),
				Runs: runs,
			})
			i++
			continue
		}

		// Table (GFM)
		if strings.HasPrefix(trimmed, "|") && strings.HasSuffix(trimmed, "|") {
			var tableRows [][]string
			for i < len(lines) {
				l := strings.TrimSpace(lines[i])
				if !strings.HasPrefix(l, "|") {
					break
				}
				// Skip separator rows
				if isSeparatorRow(l) {
					i++
					continue
				}
				cells := parseTableRow(l)
				tableRows = append(tableRows, cells)
				i++
			}
			if len(tableRows) > 0 {
				node := docx.Node{Type: docx.NodeTable}
				for _, row := range tableRows {
					rowNode := docx.Node{}
					for _, cell := range row {
						rowNode.Children = append(rowNode.Children, docx.Node{
							Type: docx.NodeParagraph,
							Text: cell,
						})
					}
					node.Children = append(node.Children, rowNode)
				}
				doc.Nodes = append(doc.Nodes, node)
			}
			continue
		}

		// Regular paragraph
		runs := parseInlineFormatting(trimmed)
		doc.Nodes = append(doc.Nodes, docx.Node{
			Type: docx.NodeParagraph,
			Text: stripFormatting(trimmed),
			Runs: runs,
		})
		i++
	}

	return doc
}

func parseInlineFormatting(text string) []docx.Run {
	var runs []docx.Run

	// Pattern for **bold**, *italic*, ***bold italic***
	boldItalicRe := regexp.MustCompile(`\*\*\*(.+?)\*\*\*`)
	boldRe := regexp.MustCompile(`\*\*(.+?)\*\*`)
	italicRe := regexp.MustCompile(`\*(.+?)\*`)

	// Simple approach: scan for formatting markers
	remaining := text
	for remaining != "" {
		// Try bold+italic first
		if loc := boldItalicRe.FindStringIndex(remaining); loc != nil && loc[0] == 0 {
			match := boldItalicRe.FindStringSubmatch(remaining)
			runs = append(runs, docx.Run{Text: match[1], Bold: true, Italic: true})
			remaining = remaining[loc[1]:]
			continue
		}

		// Try bold
		if loc := boldRe.FindStringIndex(remaining); loc != nil && loc[0] == 0 {
			match := boldRe.FindStringSubmatch(remaining)
			runs = append(runs, docx.Run{Text: match[1], Bold: true})
			remaining = remaining[loc[1]:]
			continue
		}

		// Try italic
		if loc := italicRe.FindStringIndex(remaining); loc != nil && loc[0] == 0 {
			match := italicRe.FindStringSubmatch(remaining)
			runs = append(runs, docx.Run{Text: match[1], Italic: true})
			remaining = remaining[loc[1]:]
			continue
		}

		// Find next formatting marker
		nextBoldItalic := boldItalicRe.FindStringIndex(remaining)
		nextBold := boldRe.FindStringIndex(remaining)
		nextItalic := italicRe.FindStringIndex(remaining)

		nextIdx := len(remaining)
		if nextBoldItalic != nil && nextBoldItalic[0] < nextIdx {
			nextIdx = nextBoldItalic[0]
		}
		if nextBold != nil && nextBold[0] < nextIdx {
			nextIdx = nextBold[0]
		}
		if nextItalic != nil && nextItalic[0] < nextIdx {
			nextIdx = nextItalic[0]
		}

		if nextIdx > 0 {
			runs = append(runs, docx.Run{Text: remaining[:nextIdx]})
			remaining = remaining[nextIdx:]
		} else {
			runs = append(runs, docx.Run{Text: remaining})
			remaining = ""
		}
	}

	if len(runs) == 0 {
		runs = append(runs, docx.Run{Text: text})
	}

	return runs
}

func stripFormatting(text string) string {
	text = regexp.MustCompile(`\*\*\*(.+?)\*\*\*`).ReplaceAllString(text, "$1")
	text = regexp.MustCompile(`\*\*(.+?)\*\*`).ReplaceAllString(text, "$1")
	text = regexp.MustCompile(`\*(.+?)\*`).ReplaceAllString(text, "$1")
	return text
}

func isSeparatorRow(line string) bool {
	stripped := strings.ReplaceAll(line, "|", "")
	stripped = strings.ReplaceAll(stripped, "-", "")
	stripped = strings.ReplaceAll(stripped, ":", "")
	stripped = strings.TrimSpace(stripped)
	return stripped == ""
}

func parseTableRow(line string) []string {
	// Remove leading/trailing pipes and split
	line = strings.TrimPrefix(line, "|")
	line = strings.TrimSuffix(line, "|")
	parts := strings.Split(line, "|")
	var cells []string
	for _, p := range parts {
		cells = append(cells, strings.TrimSpace(p))
	}
	return cells
}

func parseHTML(input string) *docx.Document {
	doc := &docx.Document{}

	// Simple HTML parser — strip tags and extract text
	// Handle <h1>-<h6>, <p>, <li>, basic tags
	tagRe := regexp.MustCompile(`<(/?)(\w+)[^>]*>`)

	// Remove script and style blocks
	scriptRe := regexp.MustCompile(`(?is)<script.*?</script>`)
	styleRe := regexp.MustCompile(`(?is)<style.*?</style>`)
	input = scriptRe.ReplaceAllString(input, "")
	input = styleRe.ReplaceAllString(input, "")

	var currentText strings.Builder
	var currentType docx.NodeType
	var currentLevel int
	inTag := false

	flush := func() {
		text := strings.TrimSpace(currentText.String())
		if text != "" {
			doc.Nodes = append(doc.Nodes, docx.Node{
				Type:  currentType,
				Text:  text,
				Level: currentLevel,
			})
		}
		currentText.Reset()
		currentType = docx.NodeParagraph
		currentLevel = 0
	}

	parts := tagRe.Split(input, -1)
	tags := tagRe.FindAllStringSubmatch(input, -1)

	for i, part := range parts {
		// Process text
		decoded := htmlDecode(part)
		decoded = strings.ReplaceAll(decoded, "\n", " ")
		decoded = strings.TrimSpace(decoded)
		if decoded != "" {
			if currentText.Len() > 0 {
				currentText.WriteString(" ")
			}
			currentText.WriteString(decoded)
		}

		// Process tag
		if i < len(tags) {
			isClose := tags[i][1] == "/"
			tagName := strings.ToLower(tags[i][2])
			_ = inTag

			if isClose {
				switch tagName {
				case "h1", "h2", "h3", "h4", "h5", "h6", "p", "li", "div":
					flush()
				}
				inTag = false
			} else {
				switch tagName {
				case "h1":
					flush()
					currentType = docx.NodeHeading
					currentLevel = 1
				case "h2":
					flush()
					currentType = docx.NodeHeading
					currentLevel = 2
				case "h3":
					flush()
					currentType = docx.NodeHeading
					currentLevel = 3
				case "h4", "h5", "h6":
					flush()
					currentType = docx.NodeHeading
					currentLevel = int(tagName[1] - '0')
				case "p", "div":
					flush()
					currentType = docx.NodeParagraph
				case "li":
					flush()
					currentType = docx.NodeListItem
				case "br":
					currentText.WriteString(" ")
				}
				inTag = true
			}
		}
	}
	flush()

	return doc
}

func htmlDecode(s string) string {
	s = strings.ReplaceAll(s, "&amp;", "&")
	s = strings.ReplaceAll(s, "&lt;", "<")
	s = strings.ReplaceAll(s, "&gt;", ">")
	s = strings.ReplaceAll(s, "&quot;", "\"")
	s = strings.ReplaceAll(s, "&#39;", "'")
	s = strings.ReplaceAll(s, "&nbsp;", " ")
	return s
}
