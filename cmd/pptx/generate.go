package pptx

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/klytics/m365kit/internal/bridge"
)

type generateRequest struct {
	Action  string                 `json:"action"`
	Output  string                 `json:"output"`
	Options map[string]string      `json:"options,omitempty"`
	Slides  []map[string]any       `json:"slides"`
}

func newGenerateCommand() *cobra.Command {
	var (
		outputPath string
		dataFile   string
		title      string
	)

	cmd := &cobra.Command{
		Use:   "generate",
		Short: "Generate a PowerPoint deck from structured data",
		Long: `Creates a .pptx file from a JSON data file containing slide definitions.
Uses the pptxgenjs engine via the TypeScript bridge.

The --data flag accepts a JSON file with this structure:
  {
    "title": "My Deck",
    "author": "Jane",
    "slides": [
      {
        "title": "Slide 1",
        "content": [{"text": "Hello World", "fontSize": 24, "bold": true}],
        "notes": "Speaker notes here"
      }
    ]
  }

Each slide's content array supports: text, fontSize, bold, italic, color, x, y, w, h.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			jsonFlag, _ := cmd.Flags().GetBool("json")

			if dataFile == "" {
				return fmt.Errorf("--data is required — provide a JSON file with slide definitions")
			}

			// Read and parse the data file
			raw, err := os.ReadFile(dataFile)
			if err != nil {
				if os.IsNotExist(err) {
					return fmt.Errorf("data file not found: %s", dataFile)
				}
				return fmt.Errorf("could not read data file: %w", err)
			}

			var data struct {
				Title  string           `json:"title"`
				Author string           `json:"author"`
				Slides []map[string]any `json:"slides"`
			}
			if err := json.Unmarshal(raw, &data); err != nil {
				return fmt.Errorf("invalid JSON in data file: %w", err)
			}

			if len(data.Slides) == 0 {
				return fmt.Errorf("data file contains no slides")
			}

			// Determine output path
			out := outputPath
			if out == "" {
				base := strings.TrimSuffix(filepath.Base(dataFile), filepath.Ext(dataFile))
				out = base + ".pptx"
			}

			// Resolve to absolute path for the Node bridge
			absOut, err := filepath.Abs(out)
			if err != nil {
				return fmt.Errorf("could not resolve output path: %w", err)
			}

			// Override title from flag if provided
			deckTitle := data.Title
			if title != "" {
				deckTitle = title
			}

			opts := map[string]string{}
			if deckTitle != "" {
				opts["title"] = deckTitle
			}
			if data.Author != "" {
				opts["author"] = data.Author
			}

			req := generateRequest{
				Action:  "pptx.generate",
				Output:  absOut,
				Options: opts,
				Slides:  data.Slides,
			}

			result, err := bridge.Invoke(req)
			if err != nil {
				return err
			}

			slidesCount := int(0)
			if sc, ok := result["slidesCount"].(float64); ok {
				slidesCount = int(sc)
			}

			if jsonFlag {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(map[string]any{
					"path":        absOut,
					"slidesCount": slidesCount,
				})
			}

			fmt.Printf("Generated %s (%d slides)\n", absOut, slidesCount)
			return nil
		},
	}

	cmd.Flags().StringVarP(&outputPath, "output", "o", "", "Output file path (default: <data-basename>.pptx)")
	cmd.Flags().StringVarP(&dataFile, "data", "d", "", "JSON file with slide definitions (required)")
	cmd.Flags().StringVar(&title, "title", "", "Presentation title (overrides data file)")

	return cmd
}
