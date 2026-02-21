// Package docx provides parsing and writing capabilities for .docx (OOXML) files.
package docx

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"strings"
)

// WriteDocument generates a .docx file from a Document struct, returning the raw bytes.
func WriteDocument(doc *Document) ([]byte, error) {
	buf := new(bytes.Buffer)
	zw := zip.NewWriter(buf)

	// Write [Content_Types].xml
	if err := writeContentTypes(zw); err != nil {
		return nil, fmt.Errorf("could not write content types: %w", err)
	}

	// Write _rels/.rels
	if err := writeRels(zw); err != nil {
		return nil, fmt.Errorf("could not write relationships: %w", err)
	}

	// Write word/_rels/document.xml.rels
	if err := writeDocRels(zw); err != nil {
		return nil, fmt.Errorf("could not write document relationships: %w", err)
	}

	// Write word/document.xml
	if err := writeDocumentXML(zw, doc); err != nil {
		return nil, fmt.Errorf("could not write document body: %w", err)
	}

	if err := zw.Close(); err != nil {
		return nil, fmt.Errorf("could not finalize .docx archive: %w", err)
	}

	return buf.Bytes(), nil
}

func writeContentTypes(zw *zip.Writer) error {
	w, err := zw.Create("[Content_Types].xml")
	if err != nil {
		return err
	}
	_, err = w.Write([]byte(xml.Header + `<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
  <Default Extension="xml" ContentType="application/xml"/>
  <Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>
</Types>`))
	return err
}

func writeRels(zw *zip.Writer) error {
	w, err := zw.Create("_rels/.rels")
	if err != nil {
		return err
	}
	_, err = w.Write([]byte(xml.Header + `<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/>
</Relationships>`))
	return err
}

func writeDocRels(zw *zip.Writer) error {
	w, err := zw.Create("word/_rels/document.xml.rels")
	if err != nil {
		return err
	}
	_, err = w.Write([]byte(xml.Header + `<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
</Relationships>`))
	return err
}

func writeDocumentXML(zw *zip.Writer, doc *Document) error {
	w, err := zw.Create("word/document.xml")
	if err != nil {
		return err
	}

	var b strings.Builder
	b.WriteString(xml.Header)
	b.WriteString(`<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">`)
	b.WriteString(`<w:body>`)

	for _, node := range doc.Nodes {
		writeNodeXML(&b, node)
	}

	b.WriteString(`</w:body>`)
	b.WriteString(`</w:document>`)

	_, err = w.Write([]byte(b.String()))
	return err
}

func writeNodeXML(b *strings.Builder, n Node) {
	switch n.Type {
	case NodeHeading:
		b.WriteString(`<w:p><w:pPr><w:pStyle w:val="`)
		b.WriteString(fmt.Sprintf("Heading%d", n.Level))
		b.WriteString(`"/></w:pPr>`)
		writeRunsXML(b, n)
		b.WriteString(`</w:p>`)
	case NodeParagraph:
		b.WriteString(`<w:p>`)
		writeRunsXML(b, n)
		b.WriteString(`</w:p>`)
	case NodeListItem:
		b.WriteString(`<w:p><w:pPr><w:numPr>`)
		numID := "1"
		if n.ListInfo != nil {
			numID = n.ListInfo.NumID
		}
		b.WriteString(fmt.Sprintf(`<w:ilvl w:val="%d"/>`, n.Level))
		b.WriteString(fmt.Sprintf(`<w:numId w:val="%s"/>`, numID))
		b.WriteString(`</w:numPr></w:pPr>`)
		writeRunsXML(b, n)
		b.WriteString(`</w:p>`)
	case NodeTable:
		b.WriteString(`<w:tbl>`)
		for _, row := range n.Children {
			b.WriteString(`<w:tr>`)
			for _, cell := range row.Children {
				b.WriteString(`<w:tc><w:p>`)
				writeRunsXML(b, cell)
				b.WriteString(`</w:p></w:tc>`)
			}
			b.WriteString(`</w:tr>`)
		}
		b.WriteString(`</w:tbl>`)
	}
}

func writeRunsXML(b *strings.Builder, n Node) {
	if len(n.Runs) == 0 {
		// Write as a single unformatted run
		b.WriteString(`<w:r><w:t xml:space="preserve">`)
		b.WriteString(xmlEscape(n.Text))
		b.WriteString(`</w:t></w:r>`)
		return
	}
	for _, r := range n.Runs {
		b.WriteString(`<w:r>`)
		if r.Bold || r.Italic {
			b.WriteString(`<w:rPr>`)
			if r.Bold {
				b.WriteString(`<w:b/>`)
			}
			if r.Italic {
				b.WriteString(`<w:i/>`)
			}
			b.WriteString(`</w:rPr>`)
		}
		b.WriteString(`<w:t xml:space="preserve">`)
		b.WriteString(xmlEscape(r.Text))
		b.WriteString(`</w:t></w:r>`)
	}
}

func xmlEscape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	return s
}
