package excel

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newAnalyzeCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "analyze <file.xlsx>",
		Short: "AI-powered analysis of Excel data",
		Long:  "Uses AI to identify trends, anomalies, and insights in spreadsheet data.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("excel analyze is not yet implemented — use 'kit excel read <file> --json | kit ai analyze' instead")
		},
	}
}
