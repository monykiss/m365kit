// Package shell provides the "kit shell" interactive REPL command.
package shell

import (
	"fmt"

	"github.com/spf13/cobra"

	shellpkg "github.com/klytics/m365kit/internal/shell"
)

// NewCommand creates the "shell" command.
func NewCommand() *cobra.Command {
	var (
		evalCmd string
		siteURL string
	)

	cmd := &cobra.Command{
		Use:   "shell",
		Short: "Start an interactive M365Kit shell",
		Long: `Start an interactive REPL with persistent state and tab completion.

Commands run without re-paying startup cost. Auth tokens persist across
commands in the session. Tab completion works for all commands and flags.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			session, err := shellpkg.NewSession()
			if err != nil {
				return err
			}
			if siteURL != "" {
				session.DefaultSite = siteURL
			}
			if evalCmd != "" {
				output, err := session.Eval(cmd.Context(), evalCmd)
				if err != nil {
					return err
				}
				fmt.Print(output)
				return nil
			}
			return session.Run(cmd.Context())
		},
	}

	cmd.Flags().StringVar(&evalCmd, "eval", "", "Run a single command and exit")
	cmd.Flags().StringVar(&siteURL, "sharepoint", "", "Default SharePoint site URL")
	return cmd
}
