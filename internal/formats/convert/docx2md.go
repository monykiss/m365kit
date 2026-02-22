package convert

import (
	"fmt"
	"strings"

	"github.com/klytics/m365kit/internal/formats/docx"
)

// DocxToMarkdown converts a .docx file to Markdown.
func DocxToMarkdown(inputPath string) (string, error) {
	doc, err := docx.ParseFile(inputPath)
	if err != nil {
		return "", fmt.Errorf("could not parse docx: %w", err)
	}
	return doc.Markdown(), nil
}

// DocxToText converts a .docx file to plain text.
func DocxToText(inputPath string) (string, error) {
	doc, err := docx.ParseFile(inputPath)
	if err != nil {
		return "", fmt.Errorf("could not parse docx: %w", err)
	}
	return doc.PlainText(), nil
}

// DocxToHTML converts a .docx file to a self-contained HTML5 document.
func DocxToHTML(inputPath string) (string, error) {
	doc, err := docx.ParseFile(inputPath)
	if err != nil {
		return "", fmt.Errorf("could not parse docx: %w", err)
	}

	var b strings.Builder
	b.WriteString(`<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>`)
	if doc.Metadata.Title != "" {
		b.WriteString(htmlEscape(doc.Metadata.Title))
	} else {
		b.WriteString("Document")
	}
	b.WriteString(`</title>
  <style>
    body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif; max-width: 800px; margin: 2rem auto; line-height: 1.6; padding: 0 1rem; }
    h1, h2, h3 { margin-top: 2rem; }
    table { border-collapse: collapse; width: 100%; margin: 1rem 0; }
    td, th { border: 1px solid #ddd; padding: 8px; text-align: left; }
    th { background-color: #f5f5f5; }
    ul, ol { padding-left: 2rem; }
  </style>
</head>
<body>
`)

	for _, node := range doc.Nodes {
		writeNodeHTML(&b, node)
	}

	b.WriteString(`</body>
</html>`)

	return b.String(), nil
}

func writeNodeHTML(b *strings.Builder, n docx.Node) {
	switch n.Type {
	case docx.NodeHeading:
		level := n.Level
		if level < 1 {
			level = 1
		}
		if level > 6 {
			level = 6
		}
		fmt.Fprintf(b, "<h%d>", level)
		writeRunsHTML(b, n)
		fmt.Fprintf(b, "</h%d>\n", level)

	case docx.NodeParagraph:
		b.WriteString("<p>")
		writeRunsHTML(b, n)
		b.WriteString("</p>\n")

	case docx.NodeListItem:
		b.WriteString("<ul><li>")
		writeRunsHTML(b, n)
		b.WriteString("</li></ul>\n")

	case docx.NodeTable:
		b.WriteString("<table>\n")
		for i, row := range n.Children {
			b.WriteString("<tr>")
			tag := "td"
			if i == 0 {
				tag = "th"
			}
			for _, cell := range row.Children {
				fmt.Fprintf(b, "<%s>%s</%s>", tag, htmlEscape(cell.Text), tag)
			}
			b.WriteString("</tr>\n")
		}
		b.WriteString("</table>\n")
	}
}

func writeRunsHTML(b *strings.Builder, n docx.Node) {
	if len(n.Runs) == 0 {
		b.WriteString(htmlEscape(n.Text))
		return
	}
	for _, r := range n.Runs {
		text := htmlEscape(r.Text)
		if r.Bold && r.Italic {
			b.WriteString("<strong><em>")
			b.WriteString(text)
			b.WriteString("</em></strong>")
		} else if r.Bold {
			b.WriteString("<strong>")
			b.WriteString(text)
			b.WriteString("</strong>")
		} else if r.Italic {
			b.WriteString("<em>")
			b.WriteString(text)
			b.WriteString("</em>")
		} else {
			b.WriteString(text)
		}
	}
}

func htmlEscape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	return s
}
