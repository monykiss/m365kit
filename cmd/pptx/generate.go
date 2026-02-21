package pptx

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newGenerateCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "generate",
		Short: "Generate a PowerPoint deck from template and data",
		Long:  "Creates a .pptx file from a template and structured data. Uses the TypeScript pptxgenjs engine.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("pptx generate is not yet implemented — this feature is coming in a future release")
		},
	}
}
