package word

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/klytics/m365kit/internal/formats/docx"
)

type writeInput struct {
	Title    string         `json:"title,omitempty"`
	Sections []writeSection `json:"sections,omitempty"`
}

type writeSection struct {
	Heading string `json:"heading"`
	Body    string `json:"body"`
}

type writeJSONOutput struct {
	File      string `json:"file"`
	WordCount int    `json:"wordCount"`
	Size      int    `json:"size"`
}

func newWriteCommand() *cobra.Command {
	var (
		output   string
		title    string
		content  string
		dataPath string
		template string
	)

	cmd := &cobra.Command{
		Use:   "write",
		Short: "Generate a Word document from template and data",
		Long: `Creates a .docx file from structured data or inline content.

Provide content via --title/--content flags, or pass a JSON file with --data.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			jsonFlag, _ := cmd.Flags().GetBool("json")

			if output == "" {
				return fmt.Errorf("--output is required — specify the output .docx path\n\nExample: kit word write --output report.docx --title \"Report\" --content \"Body text\"")
			}

			if !strings.HasSuffix(strings.ToLower(output), ".docx") {
				output += ".docx"
			}

			var doc *docx.Document

			if dataPath != "" {
				d, err := buildFromDataFile(dataPath)
				if err != nil {
					return err
				}
				doc = d
			} else if title != "" || content != "" {
				doc = buildFromFlags(title, content, template)
			} else {
				return fmt.Errorf("provide content via --title/--content or --data\n\nExample: kit word write --output report.docx --title \"Report\" --content \"Body text\"")
			}

			data, err := docx.WriteDocument(doc)
			if err != nil {
				return fmt.Errorf("could not generate document: %w", err)
			}

			if err := os.WriteFile(output, data, 0644); err != nil {
				return fmt.Errorf("could not write file %s: %w", output, err)
			}

			if jsonFlag {
				out := writeJSONOutput{
					File:      output,
					WordCount: doc.WordCount(),
					Size:      len(data),
				}
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(out)
			}

			fmt.Printf("Wrote %s (%d words, %d bytes)\n", output, doc.WordCount(), len(data))
			return nil
		},
	}

	cmd.Flags().StringVar(&output, "output", "", "Output .docx file path (required)")
	cmd.Flags().StringVar(&title, "title", "", "Document title")
	cmd.Flags().StringVar(&content, "content", "", "Document body text")
	cmd.Flags().StringVar(&dataPath, "data", "", "Path to JSON data file (or - for stdin)")
	cmd.Flags().StringVar(&template, "template", "simple", "Template style: simple | report | memo")

	return cmd
}

func buildFromDataFile(path string) (*docx.Document, error) {
	var raw []byte
	var err error

	if path == "-" {
		raw, err = os.ReadFile("/dev/stdin")
	} else {
		raw, err = os.ReadFile(path)
	}
	if err != nil {
		return nil, fmt.Errorf("could not read data file %s: %w", path, err)
	}

	var input writeInput
	if err := json.Unmarshal(raw, &input); err != nil {
		return nil, fmt.Errorf("invalid JSON data: %w — expected {\"title\": \"...\", \"sections\": [...]}", err)
	}

	doc := &docx.Document{}

	if input.Title != "" {
		doc.Nodes = append(doc.Nodes, docx.Node{
			Type:  docx.NodeHeading,
			Text:  input.Title,
			Level: 1,
		})
	}

	for _, section := range input.Sections {
		if section.Heading != "" {
			doc.Nodes = append(doc.Nodes, docx.Node{
				Type:  docx.NodeHeading,
				Text:  section.Heading,
				Level: 2,
			})
		}
		if section.Body != "" {
			// Split body on double newlines into separate paragraphs
			for _, para := range splitParagraphs(section.Body) {
				doc.Nodes = append(doc.Nodes, docx.Node{
					Type: docx.NodeParagraph,
					Text: para,
				})
			}
		}
	}

	return doc, nil
}

func buildFromFlags(title, content, template string) *docx.Document {
	doc := &docx.Document{}

	if title != "" {
		doc.Nodes = append(doc.Nodes, docx.Node{
			Type:  docx.NodeHeading,
			Text:  title,
			Level: 1,
		})
	}

	if content != "" {
		for _, para := range splitParagraphs(content) {
			doc.Nodes = append(doc.Nodes, docx.Node{
				Type: docx.NodeParagraph,
				Text: para,
			})
		}
	}

	return doc
}

func splitParagraphs(text string) []string {
	parts := strings.Split(text, "\n\n")
	var result []string
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	if len(result) == 0 && strings.TrimSpace(text) != "" {
		result = append(result, strings.TrimSpace(text))
	}
	return result
}
