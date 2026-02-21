// Package pptx provides reading capabilities for .pptx (PowerPoint) files.
package pptx

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
)

// Slide represents a single slide's extracted content.
type Slide struct {
	Number      int      `json:"number"`
	Title       string   `json:"title,omitempty"`
	TextContent []string `json:"textContent"`
	Notes       []string `json:"notes,omitempty"`
}

// Presentation represents a parsed PowerPoint file.
type Presentation struct {
	Slides []Slide `json:"slides"`
}

// ReadFile reads and parses a .pptx file from the given path.
func ReadFile(path string) (*Presentation, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("file not found: %s — check that the path is correct", path)
		}
		return nil, fmt.Errorf("could not read %s: %w", path, err)
	}
	return Parse(data)
}

// Parse reads and parses a .pptx file from the given byte slice.
func Parse(data []byte) (*Presentation, error) {
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("invalid .pptx file — the file does not appear to be a valid ZIP archive: %w", err)
	}

	pres := &Presentation{}

	// Collect slide files
	type slideEntry struct {
		name string
		file *zip.File
	}
	var slideFiles []slideEntry

	for _, f := range reader.File {
		if strings.HasPrefix(f.Name, "ppt/slides/slide") && strings.HasSuffix(f.Name, ".xml") {
			slideFiles = append(slideFiles, slideEntry{name: f.Name, file: f})
		}
	}

	// Sort by name to maintain slide order
	sort.Slice(slideFiles, func(i, j int) bool {
		return slideFiles[i].name < slideFiles[j].name
	})

	for i, sf := range slideFiles {
		slide, err := parseSlide(sf.file, i+1)
		if err != nil {
			return nil, fmt.Errorf("could not parse %s: %w", sf.name, err)
		}
		pres.Slides = append(pres.Slides, *slide)
	}

	return pres, nil
}

func parseSlide(f *zip.File, number int) (*Slide, error) {
	rc, err := f.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()

	data, err := io.ReadAll(rc)
	if err != nil {
		return nil, err
	}

	slide := &Slide{Number: number}

	// Extract all text content using streaming XML parser
	decoder := xml.NewDecoder(bytes.NewReader(data))
	var inTitle bool
	var texts []string

	for {
		tok, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			break
		}

		switch t := tok.(type) {
		case xml.StartElement:
			if t.Name.Local == "ph" {
				for _, attr := range t.Attr {
					if attr.Name.Local == "type" && (attr.Value == "title" || attr.Value == "ctrTitle") {
						inTitle = true
					}
				}
			}
		case xml.CharData:
			text := strings.TrimSpace(string(t))
			if text != "" {
				if inTitle && slide.Title == "" {
					slide.Title = text
				}
				texts = append(texts, text)
			}
		case xml.EndElement:
			if t.Name.Local == "sp" {
				inTitle = false
			}
		}
	}

	slide.TextContent = texts
	return slide, nil
}

// PlainText returns all slide content as plain text.
func (p *Presentation) PlainText() string {
	var b strings.Builder
	for _, slide := range p.Slides {
		fmt.Fprintf(&b, "--- Slide %d ---\n", slide.Number)
		if slide.Title != "" {
			fmt.Fprintf(&b, "%s\n\n", slide.Title)
		}
		for _, text := range slide.TextContent {
			fmt.Fprintf(&b, "%s\n", text)
		}
		b.WriteString("\n")
	}
	return b.String()
}
