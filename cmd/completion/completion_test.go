package completion

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func testRootCmd() *cobra.Command {
	root := &cobra.Command{Use: "kit"}
	root.AddCommand(&cobra.Command{Use: "word", Short: "Word operations"})
	root.AddCommand(&cobra.Command{Use: "excel", Short: "Excel operations"})
	return root
}

func TestBashCompletion(t *testing.T) {
	root := testRootCmd()
	var buf bytes.Buffer
	root.SetOut(&buf)

	if err := root.GenBashCompletion(&buf); err != nil {
		t.Fatal(err)
	}

	output := buf.String()
	if !strings.Contains(output, "_kit") {
		t.Error("bash completion should contain _kit function")
	}
}

func TestZshCompletion(t *testing.T) {
	root := testRootCmd()
	var buf bytes.Buffer

	if err := root.GenZshCompletion(&buf); err != nil {
		t.Fatal(err)
	}

	output := buf.String()
	if !strings.Contains(output, "compdef") {
		t.Error("zsh completion should contain compdef")
	}
}

func TestFishCompletion(t *testing.T) {
	root := testRootCmd()
	var buf bytes.Buffer

	if err := root.GenFishCompletion(&buf, true); err != nil {
		t.Fatal(err)
	}

	output := buf.String()
	if !strings.Contains(output, "complete -c kit") {
		t.Error("fish completion should contain 'complete -c kit'")
	}
}

func TestPowerShellCompletion(t *testing.T) {
	root := testRootCmd()
	var buf bytes.Buffer

	if err := root.GenPowerShellCompletionWithDesc(&buf); err != nil {
		t.Fatal(err)
	}

	output := buf.String()
	if !strings.Contains(output, "kit") {
		t.Error("PowerShell completion should contain kit")
	}
}
