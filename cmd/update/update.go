// Package update provides CLI commands for checking and installing updates.
package update

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/klytics/m365kit/cmd/version"
	updatelib "github.com/klytics/m365kit/internal/update"
)

// NewCommand returns the update command group.
func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Check for and install updates",
	}

	cmd.AddCommand(newCheckCommand())
	cmd.AddCommand(newInstallCommand())

	return cmd
}

func newCheckCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "check",
		Short: "Check if a newer version is available",
		RunE: func(cmd *cobra.Command, args []string) error {
			currentVersion := version.Version

			release, err := updatelib.CheckLatest(currentVersion)
			if err != nil {
				return err
			}

			if release == nil {
				green := color.New(color.FgGreen)
				green.Printf("M365Kit %s is up to date.\n", currentVersion)
				return nil
			}

			fmt.Print(updatelib.FormatUpdateNotice(currentVersion, release))
			return nil
		},
	}
}

func newInstallCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "install",
		Short: "Install the latest version",
		RunE: func(cmd *cobra.Command, args []string) error {
			currentVersion := version.Version

			release, err := updatelib.CheckLatest(currentVersion)
			if err != nil {
				return err
			}

			if release == nil {
				color.New(color.FgGreen).Printf("M365Kit %s is already the latest version.\n", currentVersion)
				return nil
			}

			fmt.Printf("New version available: %s\n\n", release.Version)
			fmt.Println("To update, run one of:")
			fmt.Println("  brew upgrade monykiss/tap/m365kit  (Homebrew)")
			fmt.Println("  go install github.com/monykiss/m365kit@latest  (Go)")
			fmt.Printf("\nRelease notes: %s\n", release.HTMLURL)
			return nil
		},
	}
}
