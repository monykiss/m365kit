package word

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newSummarizeCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "summarize <file.docx>",
		Short: "Generate an AI summary of a Word document",
		Long:  "Uses AI to produce a concise summary of a .docx file. Equivalent to 'kit word read <file> | kit ai summarize'.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("word summarize is not yet implemented — use 'kit word read <file> | kit ai summarize' instead")
		},
	}
}
