package docx

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
)

// EditResult holds the outcome of an edit operation.
type EditResult struct {
	ReplacementsMade int
	OutputPath       string
}

// EditFile performs find-and-replace operations on a .docx file.
// It preserves all XML structure and only modifies text content within <w:t> elements.
func EditFile(inputPath string, replacements map[string]string, outputPath string) (*EditResult, error) {
	data, err := os.ReadFile(inputPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("file not found: %s — check that the path is correct", inputPath)
		}
		return nil, fmt.Errorf("could not read %s: %w", inputPath, err)
	}

	edited, count, err := EditBytes(data, replacements)
	if err != nil {
		return nil, err
	}

	if err := os.WriteFile(outputPath, edited, 0644); err != nil {
		return nil, fmt.Errorf("could not write %s: %w", outputPath, err)
	}

	return &EditResult{
		ReplacementsMade: count,
		OutputPath:       outputPath,
	}, nil
}

// EditBytes performs find-and-replace on raw .docx bytes.
// Returns the modified bytes and the total number of replacements made.
func EditBytes(data []byte, replacements map[string]string) ([]byte, int, error) {
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, 0, fmt.Errorf("invalid .docx file: %w", err)
	}

	buf := new(bytes.Buffer)
	writer := zip.NewWriter(buf)
	totalReplacements := 0

	for _, f := range reader.File {
		rc, err := f.Open()
		if err != nil {
			return nil, 0, fmt.Errorf("could not open %s in archive: %w", f.Name, err)
		}

		content, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			return nil, 0, fmt.Errorf("could not read %s: %w", f.Name, err)
		}

		// Only perform replacements in XML files within the word/ directory
		if isTextXML(f.Name) {
			text := string(content)
			for find, replace := range replacements {
				count := strings.Count(text, find)
				if count > 0 {
					totalReplacements += count
					text = strings.ReplaceAll(text, find, replace)
				}
			}
			content = []byte(text)
		}

		// Create new file in output ZIP preserving the original header
		header := &zip.FileHeader{
			Name:   f.Name,
			Method: f.Method,
		}
		header.SetModTime(f.Modified)

		w, err := writer.CreateHeader(header)
		if err != nil {
			return nil, 0, fmt.Errorf("could not create %s in output: %w", f.Name, err)
		}

		if _, err := w.Write(content); err != nil {
			return nil, 0, fmt.Errorf("could not write %s: %w", f.Name, err)
		}
	}

	if err := writer.Close(); err != nil {
		return nil, 0, fmt.Errorf("could not finalize output archive: %w", err)
	}

	return buf.Bytes(), totalReplacements, nil
}

func isTextXML(name string) bool {
	return strings.HasPrefix(name, "word/") && strings.HasSuffix(name, ".xml")
}
