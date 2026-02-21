// Package version provides the version command for the kit CLI.
package version

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Version is set at build time via ldflags.
var Version = "dev"

// NewCommand returns the version subcommand.
func NewCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the kit version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("kit %s\n", Version)
		},
	}
}
