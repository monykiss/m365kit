package word

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/klytics/m365kit/internal/formats/docx"
)

type readOutput struct {
	Paragraphs []string       `json:"paragraphs"`
	Metadata   docx.Metadata  `json:"metadata"`
	WordCount  int            `json:"wordCount"`
}

func newReadCommand() *cobra.Command {
	var markdown bool

	cmd := &cobra.Command{
		Use:   "read <file.docx>",
		Short: "Extract text content from a Word document",
		Long:  "Reads a .docx file and outputs its text content. Supports plain text, JSON, and Markdown output formats. Pass '-' to read from stdin.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			jsonFlag, _ := cmd.Flags().GetBool("json")

			var doc *docx.Document
			var err error

			if len(args) == 0 || args[0] == "-" {
				// Read from stdin
				data, readErr := io.ReadAll(os.Stdin)
				if readErr != nil {
					return fmt.Errorf("could not read from stdin: %w", readErr)
				}
				if len(data) == 0 {
					return fmt.Errorf("no input provided — pass a .docx file path or pipe data to stdin")
				}
				doc, err = docx.Parse(data)
			} else {
				filePath := args[0]
				if !strings.HasSuffix(strings.ToLower(filePath), ".docx") {
					return fmt.Errorf("expected a .docx file, got %q — use 'kit word read <file.docx>'", filePath)
				}
				doc, err = docx.ParseFile(filePath)
			}

			if err != nil {
				return err
			}

			if jsonFlag {
				return outputJSON(doc)
			}

			if markdown {
				fmt.Print(doc.Markdown())
				return nil
			}

			return outputPretty(doc)
		},
	}

	cmd.Flags().BoolVar(&markdown, "markdown", false, "Output as clean Markdown")

	return cmd
}

func outputJSON(doc *docx.Document) error {
	out := readOutput{
		Paragraphs: doc.Paragraphs(),
		Metadata:   doc.Metadata,
		WordCount:  doc.WordCount(),
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

func outputPretty(doc *docx.Document) error {
	bold := color.New(color.Bold)
	heading := color.New(color.Bold, color.FgCyan)
	dim := color.New(color.FgHiBlack)

	for _, node := range doc.Nodes {
		switch node.Type {
		case docx.NodeHeading:
			heading.Printf("%s %s\n", strings.Repeat("#", node.Level), node.Text)
		case docx.NodeParagraph:
			hasBold := false
			for _, r := range node.Runs {
				if r.Bold {
					hasBold = true
					break
				}
			}
			if hasBold {
				for _, r := range node.Runs {
					if r.Bold {
						bold.Print(r.Text)
					} else {
						fmt.Print(r.Text)
					}
				}
				fmt.Println()
			} else {
				fmt.Println(node.Text)
			}
		case docx.NodeListItem:
			fmt.Printf("  %s %s\n", dim.Sprint("•"), node.Text)
		case docx.NodeTable:
			for _, row := range node.Children {
				cells := make([]string, 0, len(row.Children))
				for _, cell := range row.Children {
					cells = append(cells, cell.Text)
				}
				dim.Print("| ")
				fmt.Print(strings.Join(cells, " | "))
				dim.Println(" |")
			}
			fmt.Println()
		}
	}

	dim.Printf("\n--- %d words ---\n", doc.WordCount())
	return nil
}
