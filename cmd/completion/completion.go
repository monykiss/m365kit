// Package completion provides shell completion generation commands.
package completion

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// NewCommand returns the completion command.
func NewCommand(rootCmd *cobra.Command) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "completion [bash|zsh|fish|powershell]",
		Short: "Generate shell completions",
		Long: `Generate shell completion scripts for M365Kit.

Install instructions:
  Bash:       kit completion bash > /etc/bash_completion.d/kit
              echo 'source <(kit completion bash)' >> ~/.bashrc
  Zsh:        kit completion zsh > ~/.zsh/completions/_kit
  Fish:       kit completion fish > ~/.config/fish/completions/kit.fish
  PowerShell: kit completion powershell >> $PROFILE`,
		ValidArgs: []string{"bash", "zsh", "fish", "powershell"},
		Args:      cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			switch args[0] {
			case "bash":
				fmt.Fprintln(os.Stdout, "# M365Kit bash completion")
				fmt.Fprintln(os.Stdout, "# Install: kit completion bash > /etc/bash_completion.d/kit")
				fmt.Fprintln(os.Stdout, "# Or:      echo 'source <(kit completion bash)' >> ~/.bashrc")
				fmt.Fprintln(os.Stdout)
				return rootCmd.GenBashCompletion(os.Stdout)
			case "zsh":
				fmt.Fprintln(os.Stdout, "# M365Kit zsh completion")
				fmt.Fprintln(os.Stdout, "# Install: kit completion zsh > ~/.zsh/completions/_kit")
				fmt.Fprintln(os.Stdout)
				return rootCmd.GenZshCompletion(os.Stdout)
			case "fish":
				fmt.Fprintln(os.Stdout, "# M365Kit fish completion")
				fmt.Fprintln(os.Stdout, "# Install: kit completion fish > ~/.config/fish/completions/kit.fish")
				fmt.Fprintln(os.Stdout)
				return rootCmd.GenFishCompletion(os.Stdout, true)
			case "powershell":
				fmt.Fprintln(os.Stdout, "# M365Kit PowerShell completion")
				fmt.Fprintln(os.Stdout, "# Install: kit completion powershell >> $PROFILE")
				fmt.Fprintln(os.Stdout)
				return rootCmd.GenPowerShellCompletionWithDesc(os.Stdout)
			default:
				return fmt.Errorf("unsupported shell: %s (supported: bash, zsh, fish, powershell)", args[0])
			}
		},
	}
	return cmd
}
