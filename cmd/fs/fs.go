// Package fs provides CLI commands for local file system intelligence.
package fs

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	fslib "github.com/klytics/m365kit/internal/fs"
)

// NewCommand returns the fs command group.
func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "fs",
		Short: "Local file system intelligence for Office documents",
		Long:  "Scan, rename, deduplicate, and organize Office documents on the local filesystem.",
	}

	cmd.AddCommand(newScanCommand())
	cmd.AddCommand(newRenameCommand())
	cmd.AddCommand(newDedupeCommand())
	cmd.AddCommand(newStaleCommand())
	cmd.AddCommand(newOrganizeCommand())
	cmd.AddCommand(newManifestCommand())

	return cmd
}

func newScanCommand() *cobra.Command {
	var (
		recursive bool
		exts      []string
		withHash  bool
	)
	cmd := &cobra.Command{
		Use:   "scan [directory]",
		Short: "Scan for Office documents",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			jsonFlag, _ := cmd.Flags().GetBool("json")

			dir := "."
			if len(args) > 0 {
				dir = args[0]
			}

			result, err := fslib.Scan(dir, fslib.ScanOptions{
				Recursive:  recursive,
				Extensions: exts,
				WithHash:   withHash,
			})
			if err != nil {
				return err
			}

			if jsonFlag {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(result)
			}

			fmt.Printf("Scanned: %s\n", result.RootDir)
			fmt.Printf("Found: %d Office documents (%s)\n\n", len(result.Files), fslib.FormatSize(result.TotalSize))

			if len(result.ByFormat) > 0 {
				bold := color.New(color.Bold)
				bold.Println("By format:")
				for format, count := range result.ByFormat {
					fmt.Printf("  %-20s %d files\n", format, count)
				}
				fmt.Println()
			}

			if len(result.Files) > 0 {
				w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
				fmt.Fprintf(w, "NAME\tSIZE\tMODIFIED\tPATH\n")
				for _, f := range result.Files {
					fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
						f.Name,
						fslib.FormatSize(f.Size),
						f.ModifiedAt.Format("2006-01-02"),
						f.Path)
				}
				w.Flush()
			}
			return nil
		},
	}
	cmd.Flags().BoolVarP(&recursive, "recursive", "r", false, "Scan subdirectories")
	cmd.Flags().StringSliceVar(&exts, "ext", nil, "Filter by extension (e.g., .docx,.xlsx)")
	cmd.Flags().BoolVar(&withHash, "hash", false, "Compute SHA-256 hashes (needed for dedupe)")
	return cmd
}

func newRenameCommand() *cobra.Command {
	var (
		pattern   string
		dryRun    bool
		recursive bool
	)
	cmd := &cobra.Command{
		Use:   "rename [directory]",
		Short: "Rename Office documents with consistent naming",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			jsonFlag, _ := cmd.Flags().GetBool("json")

			dir := "."
			if len(args) > 0 {
				dir = args[0]
			}

			result, err := fslib.Scan(dir, fslib.ScanOptions{Recursive: recursive})
			if err != nil {
				return err
			}

			results := fslib.Rename(result.Files, fslib.RenameRule{
				Pattern: pattern,
				DryRun:  dryRun,
			})

			if jsonFlag {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(results)
			}

			changed := 0
			for _, r := range results {
				if r.OldPath == r.NewPath {
					continue
				}
				status := "would rename"
				if r.Applied {
					status = "renamed"
					changed++
				}
				if r.Error != "" {
					status = "error: " + r.Error
				}
				fmt.Printf("[%s] %s → %s\n", status, r.OldPath, r.NewPath)
			}

			if dryRun {
				renames := 0
				for _, r := range results {
					if r.OldPath != r.NewPath {
						renames++
					}
				}
				fmt.Printf("\nDry run: %d files would be renamed (use without --dry-run to apply)\n", renames)
			} else {
				fmt.Printf("\n%d files renamed\n", changed)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&pattern, "pattern", "kebab", "Naming pattern: kebab | snake | lower | date-prefix")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview changes without applying")
	cmd.Flags().BoolVarP(&recursive, "recursive", "r", false, "Include subdirectories")
	return cmd
}

func newDedupeCommand() *cobra.Command {
	var (
		dryRun    bool
		recursive bool
	)
	cmd := &cobra.Command{
		Use:   "dedupe [directory]",
		Short: "Find and remove duplicate Office documents",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			jsonFlag, _ := cmd.Flags().GetBool("json")

			dir := "."
			if len(args) > 0 {
				dir = args[0]
			}

			result, err := fslib.Scan(dir, fslib.ScanOptions{
				Recursive: recursive,
				WithHash:  true,
			})
			if err != nil {
				return err
			}

			dupes := fslib.FindDuplicates(result.Files)

			if jsonFlag {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(dupes)
			}

			if len(dupes.Groups) == 0 {
				fmt.Println("No duplicates found")
				return nil
			}

			fmt.Print(fslib.FormatDedupeReport(dupes))

			if !dryRun {
				results := fslib.RemoveDuplicates(dupes.Groups, false)
				removed := 0
				for _, r := range results {
					if r.Applied {
						removed++
					}
				}
				fmt.Printf("Removed %d duplicate files\n", removed)
			} else {
				fmt.Printf("Dry run: %d duplicate files would be removed\n", dupes.TotalDupes)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview without deleting")
	cmd.Flags().BoolVarP(&recursive, "recursive", "r", false, "Include subdirectories")
	return cmd
}

func newStaleCommand() *cobra.Command {
	var (
		days      int
		recursive bool
	)
	cmd := &cobra.Command{
		Use:   "stale [directory]",
		Short: "Find Office documents not modified in N days",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			jsonFlag, _ := cmd.Flags().GetBool("json")

			dir := "."
			if len(args) > 0 {
				dir = args[0]
			}

			result, err := fslib.Scan(dir, fslib.ScanOptions{Recursive: recursive})
			if err != nil {
				return err
			}

			stale := fslib.StaleFiles(result.Files, time.Duration(days)*24*time.Hour)

			if jsonFlag {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(stale)
			}

			if len(stale) == 0 {
				fmt.Printf("No files older than %d days\n", days)
				return nil
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintf(w, "LAST MODIFIED\tDAYS AGO\tNAME\tPATH\n")
			for _, f := range stale {
				daysAgo := int(time.Since(f.ModifiedAt).Hours() / 24)
				fmt.Fprintf(w, "%s\t%d\t%s\t%s\n",
					f.ModifiedAt.Format("2006-01-02"),
					daysAgo, f.Name, f.Path)
			}
			return w.Flush()
		},
	}
	cmd.Flags().IntVar(&days, "days", 90, "Days since last modification")
	cmd.Flags().BoolVarP(&recursive, "recursive", "r", false, "Include subdirectories")
	return cmd
}

func newOrganizeCommand() *cobra.Command {
	var (
		strategy  string
		dryRun    bool
		recursive bool
	)
	cmd := &cobra.Command{
		Use:   "organize [directory]",
		Short: "Organize Office documents into folders",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			jsonFlag, _ := cmd.Flags().GetBool("json")

			dir := "."
			if len(args) > 0 {
				dir = args[0]
			}

			result, err := fslib.Scan(dir, fslib.ScanOptions{Recursive: recursive})
			if err != nil {
				return err
			}

			results := fslib.OrganizeFile(result.Files, result.RootDir, fslib.OrganizeRule{
				Strategy: strategy,
				DryRun:   dryRun,
			})

			if jsonFlag {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(results)
			}

			moved := 0
			for _, r := range results {
				if r.OldPath == r.NewPath {
					continue
				}
				status := "would move"
				if r.Applied {
					status = "moved"
					moved++
				}
				if r.Error != "" {
					status = "error: " + r.Error
				}
				fmt.Printf("[%s] %s → %s\n", status, r.OldPath, r.NewPath)
			}

			if dryRun {
				moves := 0
				for _, r := range results {
					if r.OldPath != r.NewPath {
						moves++
					}
				}
				fmt.Printf("\nDry run: %d files would be moved (use without --dry-run to apply)\n", moves)
			} else {
				fmt.Printf("\n%d files organized\n", moved)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&strategy, "strategy", "by-type", "Organization: by-type | by-year | by-month")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview changes without applying")
	cmd.Flags().BoolVarP(&recursive, "recursive", "r", false, "Include subdirectories")
	return cmd
}

func newManifestCommand() *cobra.Command {
	var recursive bool
	cmd := &cobra.Command{
		Use:   "manifest [directory]",
		Short: "Generate a JSON manifest of Office documents",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := "."
			if len(args) > 0 {
				dir = args[0]
			}

			result, err := fslib.Scan(dir, fslib.ScanOptions{
				Recursive: recursive,
				WithHash:  true,
			})
			if err != nil {
				return err
			}

			data, err := fslib.Manifest(result)
			if err != nil {
				return err
			}

			fmt.Println(string(data))
			return nil
		},
	}
	cmd.Flags().BoolVarP(&recursive, "recursive", "r", false, "Include subdirectories")
	return cmd
}
