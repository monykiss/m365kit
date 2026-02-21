package word

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/klytics/m365kit/internal/formats/docx"
)

type editJSONOutput struct {
	ReplacementsMade int    `json:"replacements_made"`
	Output           string `json:"output"`
}

func newEditCommand() *cobra.Command {
	var (
		find           string
		replace        string
		replacements   string
		inPlace        bool
		outputPath     string
	)

	cmd := &cobra.Command{
		Use:   "edit <file.docx>",
		Short: "Find and replace text in a Word document",
		Long: `Performs find-and-replace operations on a .docx file.

By default writes to {basename}.edited.docx. Use --in-place to overwrite the source file.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			jsonFlag, _ := cmd.Flags().GetBool("json")
			inputPath := args[0]

			if !strings.HasSuffix(strings.ToLower(inputPath), ".docx") {
				return fmt.Errorf("expected a .docx file, got %q", inputPath)
			}

			// Build replacements map
			replMap := make(map[string]string)

			if find != "" {
				if replace == "" {
					return fmt.Errorf("--replace is required when using --find")
				}
				replMap[find] = replace
			}

			if replacements != "" {
				fileMap, err := loadReplacementsFile(replacements)
				if err != nil {
					return err
				}
				for k, v := range fileMap {
					replMap[k] = v
				}
			}

			if len(replMap) == 0 {
				return fmt.Errorf("no replacements specified — use --find/--replace or --replacements\n\nExample: kit word edit doc.docx --find \"old\" --replace \"new\"")
			}

			// Determine output path
			outPath := outputPath
			if outPath == "" {
				if inPlace {
					outPath = inputPath
				} else {
					ext := filepath.Ext(inputPath)
					base := strings.TrimSuffix(inputPath, ext)
					outPath = base + ".edited" + ext
				}
			}

			// If editing in-place, we need to read first then write
			if outPath == inputPath {
				data, err := os.ReadFile(inputPath)
				if err != nil {
					return fmt.Errorf("could not read %s: %w", inputPath, err)
				}
				edited, count, err := docx.EditBytes(data, replMap)
				if err != nil {
					return err
				}
				if err := os.WriteFile(outPath, edited, 0644); err != nil {
					return fmt.Errorf("could not write %s: %w", outPath, err)
				}
				return outputEditResult(jsonFlag, count, outPath)
			}

			result, err := docx.EditFile(inputPath, replMap, outPath)
			if err != nil {
				return err
			}

			return outputEditResult(jsonFlag, result.ReplacementsMade, result.OutputPath)
		},
	}

	cmd.Flags().StringVar(&find, "find", "", "Text to find")
	cmd.Flags().StringVar(&replace, "replace", "", "Replacement text")
	cmd.Flags().StringVar(&replacements, "replacements", "", "Path to JSON replacements map {\"find\": \"replace\", ...}")
	cmd.Flags().BoolVar(&inPlace, "in-place", false, "Overwrite the source file (use with caution)")
	cmd.Flags().StringVar(&outputPath, "output", "", "Explicit output file path")

	return cmd
}

func loadReplacementsFile(path string) (map[string]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("could not read replacements file %s: %w", path, err)
	}

	var m map[string]string
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("invalid replacements JSON: %w — expected {\"find\": \"replace\", ...}", err)
	}

	return m, nil
}

func outputEditResult(jsonFlag bool, count int, path string) error {
	if jsonFlag {
		out := editJSONOutput{
			ReplacementsMade: count,
			Output:           path,
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(out)
	}

	if count == 0 {
		fmt.Printf("No replacements made in %s\n", path)
	} else {
		fmt.Printf("Made %d replacement(s) → %s\n", count, path)
	}
	return nil
}
