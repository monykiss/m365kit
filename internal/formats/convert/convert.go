// Package convert provides format conversion between document types.
// All conversions are pure Go — no Word, LibreOffice, or subprocess calls.
package convert

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// SupportedConversions lists all supported from→to format pairs.
var SupportedConversions = map[string][]string{
	"docx": {"md", "html", "txt"},
	"md":   {"docx"},
	"html": {"docx"},
	"xlsx": {"csv", "json", "md"},
}

// Convert converts a file from one format to another.
// If outputPath is empty, returns the result as a string (for piping).
func Convert(inputPath, outputPath, toFmt string) (string, error) {
	fromFmt := detectFormat(inputPath)
	if fromFmt == "" {
		return "", fmt.Errorf("could not detect input format from extension: %s", filepath.Ext(inputPath))
	}

	// Validate conversion is supported
	supported := SupportedConversions[fromFmt]
	found := false
	for _, s := range supported {
		if s == toFmt {
			found = true
			break
		}
	}
	if !found {
		return "", fmt.Errorf("unsupported conversion: %s → %s (supported from %s: %v)", fromFmt, toFmt, fromFmt, supported)
	}

	var result string
	var err error

	switch fromFmt + "→" + toFmt {
	case "docx→md":
		result, err = DocxToMarkdown(inputPath)
	case "docx→html":
		result, err = DocxToHTML(inputPath)
	case "docx→txt":
		result, err = DocxToText(inputPath)
	case "md→docx":
		input, readErr := os.ReadFile(inputPath)
		if readErr != nil {
			return "", fmt.Errorf("could not read %s: %w", inputPath, readErr)
		}
		if outputPath == "" {
			outputPath = strings.TrimSuffix(inputPath, filepath.Ext(inputPath)) + ".docx"
		}
		return "", MarkdownToDocx(string(input), outputPath)
	case "html→docx":
		input, readErr := os.ReadFile(inputPath)
		if readErr != nil {
			return "", fmt.Errorf("could not read %s: %w", inputPath, readErr)
		}
		if outputPath == "" {
			outputPath = strings.TrimSuffix(inputPath, filepath.Ext(inputPath)) + ".docx"
		}
		return "", HTMLToDocx(string(input), outputPath)
	case "xlsx→csv":
		result, err = XlsxToCSV(inputPath, "")
	case "xlsx→json":
		result, err = XlsxToJSON(inputPath, "")
	case "xlsx→md":
		result, err = XlsxToMarkdown(inputPath, "")
	default:
		return "", fmt.Errorf("conversion %s → %s not implemented", fromFmt, toFmt)
	}

	if err != nil {
		return "", err
	}

	if outputPath != "" && result != "" {
		if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
			return "", err
		}
		if err := os.WriteFile(outputPath, []byte(result), 0644); err != nil {
			return "", fmt.Errorf("could not write %s: %w", outputPath, err)
		}
		return result, nil
	}

	return result, nil
}

func detectFormat(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".docx":
		return "docx"
	case ".md", ".markdown":
		return "md"
	case ".html", ".htm":
		return "html"
	case ".xlsx":
		return "xlsx"
	case ".txt":
		return "txt"
	default:
		return ""
	}
}
