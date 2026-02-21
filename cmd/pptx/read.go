package pptx

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	pptxformat "github.com/klytics/m365kit/internal/formats/pptx"
)

func newReadCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "read <file.pptx>",
		Short: "Extract slide content from a PowerPoint file",
		Long:  "Reads a .pptx file and outputs slide content as text, JSON, or Markdown.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			jsonFlag, _ := cmd.Flags().GetBool("json")

			filePath := args[0]
			if !strings.HasSuffix(strings.ToLower(filePath), ".pptx") {
				return fmt.Errorf("expected a .pptx file, got %q", filePath)
			}

			pres, err := pptxformat.ReadFile(filePath)
			if err != nil {
				return err
			}

			if jsonFlag {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(pres)
			}

			return outputPPTXPretty(pres)
		},
	}

	return cmd
}

func outputPPTXPretty(pres *pptxformat.Presentation) error {
	heading := color.New(color.Bold, color.FgCyan)
	dim := color.New(color.FgHiBlack)

	for _, slide := range pres.Slides {
		heading.Printf("Slide %d", slide.Number)
		if slide.Title != "" {
			heading.Printf(": %s", slide.Title)
		}
		heading.Println()

		for _, text := range slide.TextContent {
			fmt.Printf("  %s\n", text)
		}

		if len(slide.Notes) > 0 {
			dim.Println("  Notes:")
			for _, note := range slide.Notes {
				dim.Printf("    %s\n", note)
			}
		}
		fmt.Println()
	}

	dim.Printf("--- %d slides ---\n", len(pres.Slides))
	return nil
}
