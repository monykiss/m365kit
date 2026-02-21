package output

import (
	"os"
	"os/exec"
	"strings"
)

// ShouldPage returns true if output should be piped through a pager.
// This checks if stdout is a terminal and the content exceeds terminal height.
func ShouldPage(content string, termHeight int) bool {
	if !isTerminal() {
		return false
	}
	lines := strings.Count(content, "\n")
	return lines > termHeight
}

// Page pipes content through the user's preferred pager (PAGER env, or "less").
func Page(content string) error {
	pager := os.Getenv("PAGER")
	if pager == "" {
		pager = "less"
	}

	cmd := exec.Command(pager)
	cmd.Stdin = strings.NewReader(content)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func isTerminal() bool {
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}
