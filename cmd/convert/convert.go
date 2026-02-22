// Package convert provides the "kit convert" CLI command for format conversion.
package convert

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	conv "github.com/klytics/m365kit/internal/formats/convert"
)

// NewCommand creates the "convert" command.
func NewCommand() *cobra.Command {
	var (
		toFmt    string
		output   string
		sheet    string
		outDir   string
	)

	cmd := &cobra.Command{
		Use:   "convert <file> --to <format>",
		Short: "Convert between document formats (no Word required)",
		Long: `Convert between document formats using pure Go. No Word, LibreOffice, or
external tools required.

Supported conversions:
  .docx → .md, .html, .txt
  .md   → .docx
  .html → .docx
  .xlsx → .csv, .json, .md

Examples:
  kit convert document.docx --to md
  kit convert README.md --to docx --output README.docx
  kit convert data.xlsx --to csv --sheet Revenue
  kit convert '*.docx' --to md --out-dir ./markdown/`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if toFmt == "" {
				return fmt.Errorf("--to is required (e.g., md, html, txt, docx, csv, json)")
			}

			inputPattern := args[0]

			// Check for glob pattern
			if strings.Contains(inputPattern, "*") {
				return batchConvert(inputPattern, toFmt, outDir)
			}

			// Single file conversion
			outPath := output
			if outPath == "" && outDir != "" {
				base := strings.TrimSuffix(filepath.Base(inputPattern), filepath.Ext(inputPattern))
				outPath = filepath.Join(outDir, base+"."+toFmt)
			}

			result, err := conv.Convert(inputPattern, outPath, toFmt)
			if err != nil {
				return err
			}

			jsonOut, _ := cmd.Flags().GetBool("json")
			if jsonOut {
				return json.NewEncoder(os.Stdout).Encode(map[string]string{
					"input":  inputPattern,
					"output": outPath,
					"format": toFmt,
				})
			}

			if outPath != "" {
				fmt.Printf("Converted: %s → %s\n", inputPattern, outPath)
			} else if result != "" {
				fmt.Print(result)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&toFmt, "to", "", "Target format (md, html, txt, docx, csv, json)")
	cmd.Flags().StringVarP(&output, "output", "o", "", "Output file path")
	cmd.Flags().StringVar(&sheet, "sheet", "", "Sheet name for XLSX conversion")
	cmd.Flags().StringVar(&outDir, "out-dir", "", "Output directory for batch conversion")

	return cmd
}

func batchConvert(pattern, toFmt, outDir string) error {
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return fmt.Errorf("invalid pattern: %w", err)
	}

	if len(matches) == 0 {
		fmt.Println("No files matched the pattern.")
		return nil
	}

	if outDir == "" {
		outDir = "."
	}

	for _, inputPath := range matches {
		base := strings.TrimSuffix(filepath.Base(inputPath), filepath.Ext(inputPath))
		outPath := filepath.Join(outDir, base+"."+toFmt)

		_, err := conv.Convert(inputPath, outPath, toFmt)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not convert %s: %v\n", inputPath, err)
			continue
		}
		fmt.Printf("Converted: %s → %s\n", inputPath, outPath)
	}

	return nil
}
