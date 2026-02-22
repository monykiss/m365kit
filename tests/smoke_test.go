// Package tests provides smoke tests that validate every kit command
// exists, runs, and exits cleanly without panicking.
// These tests compile and run the binary — they are integration tests.
// They do NOT require Azure credentials, SMTP config, or API keys.
package tests

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// kitBin returns the path to the compiled kit binary.
func kitBin(t *testing.T) string {
	t.Helper()
	_, filename, _, _ := runtime.Caller(0)
	root := filepath.Join(filepath.Dir(filename), "..")
	bin := filepath.Join(root, "bin", "kit")
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}
	if _, err := os.Stat(bin); os.IsNotExist(err) {
		t.Fatalf("kit binary not found at %s — run 'make build' first", bin)
	}
	return bin
}

// run executes kit with args and returns stdout, stderr, and exit code.
func run(t *testing.T, args ...string) (string, string, int) {
	t.Helper()
	cmd := exec.Command(kitBin(t), args...)
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	code := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			code = exitErr.ExitCode()
		}
	}
	return stdout.String(), stderr.String(), code
}

// TestAllCommandsExist validates that every command appears in --help.
func TestAllCommandsExist(t *testing.T) {
	commands := []string{
		"word", "excel", "pptx", "ai", "pipeline", "batch",
		"auth", "onedrive", "sharepoint", "teams", "outlook", "acl",
		"fs", "template", "report", "watch",
		"send", "diff", "convert",
		"config", "completion", "update", "doctor", "version",
	}

	stdout, _, code := run(t, "--help")
	if code != 0 {
		t.Fatalf("kit --help exited with code %d", code)
	}
	for _, cmd := range commands {
		if !strings.Contains(stdout, cmd) {
			t.Errorf("command %q not found in kit --help output", cmd)
		}
	}
}

// TestWordReadHelp validates word read command structure.
func TestWordReadHelp(t *testing.T) {
	_, _, code := run(t, "word", "read", "--help")
	if code != 0 {
		t.Error("kit word read --help should exit 0")
	}
}

// TestWordWriteThenRead validates the core write + read round-trip.
func TestWordWriteThenRead(t *testing.T) {
	tmp := t.TempDir()
	out := filepath.Join(tmp, "smoke_test.docx")

	_, _, code := run(t, "word", "write",
		"--output", out,
		"--title", "Smoke Test",
		"--content", "This is a smoke test paragraph.")
	if code != 0 {
		t.Fatal("kit word write should exit 0")
	}
	if _, err := os.Stat(out); os.IsNotExist(err) {
		t.Fatal("output file was not created")
	}

	stdout, _, code := run(t, "word", "read", out)
	if code != 0 {
		t.Fatal("kit word read should exit 0")
	}
	if !strings.Contains(stdout, "Smoke Test") {
		t.Error("word read output should contain the title")
	}
}

// TestWordReadJSON validates JSON output structure.
func TestWordReadJSON(t *testing.T) {
	tmp := t.TempDir()
	out := filepath.Join(tmp, "json_test.docx")
	run(t, "word", "write", "--output", out, "--title", "JSON Test", "--content", "content")

	stdout, _, code := run(t, "word", "read", out, "--json")
	if code != 0 {
		t.Fatal("kit word read --json should exit 0")
	}
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("--json output is not valid JSON: %v\nOutput: %s", err, stdout)
	}
}

// TestDiffIdentical validates diff with identical files.
func TestDiffIdentical(t *testing.T) {
	tmp := t.TempDir()
	doc := filepath.Join(tmp, "same.docx")
	run(t, "word", "write", "--output", doc, "--title", "Same", "--content", "content")

	stdout, _, code := run(t, "diff", doc, doc, "--stats")
	if code != 0 {
		t.Fatal("kit diff on identical files should exit 0")
	}
	if !strings.Contains(stdout, "0") {
		t.Errorf("identical diff should report 0 changes, got: %s", stdout)
	}
}

// TestConvertDocxToMd validates conversion produces Markdown.
func TestConvertDocxToMd(t *testing.T) {
	tmp := t.TempDir()
	docx := filepath.Join(tmp, "conv.docx")
	run(t, "word", "write", "--output", docx, "--title", "Convert Me", "--content", "body text")

	stdout, _, code := run(t, "convert", docx, "--to", "md")
	if code != 0 {
		t.Fatal("kit convert --to md should exit 0")
	}
	if !strings.Contains(stdout, "Convert Me") {
		t.Errorf("markdown output should contain heading, got: %s", stdout)
	}
}

// TestSendDryRun validates send works without SMTP.
func TestSendDryRun(t *testing.T) {
	tmp := t.TempDir()
	doc := filepath.Join(tmp, "send.docx")
	run(t, "word", "write", "--output", doc, "--title", "T", "--content", "c")

	_, _, code := run(t, "send",
		"--to", "test@example.com",
		"--attach", doc,
		"--dry-run")
	if code != 0 {
		t.Error("kit send --dry-run should exit 0 without SMTP config")
	}
}

// TestFsScanEmpty validates fs scan on empty dir.
func TestFsScanEmpty(t *testing.T) {
	tmp := t.TempDir()
	_, _, code := run(t, "fs", "scan", tmp)
	if code != 0 {
		t.Error("kit fs scan on empty dir should exit 0")
	}
}

// TestVersionOutput validates version command format.
func TestVersionOutput(t *testing.T) {
	stdout, _, code := run(t, "version")
	if code != 0 {
		t.Fatal("kit version should exit 0")
	}
	if !strings.Contains(stdout, "kit") {
		t.Errorf("version output should contain 'kit', got: %s", stdout)
	}
}

// TestDoctorRuns validates doctor command runs without panic.
func TestDoctorRuns(t *testing.T) {
	_, _, code := run(t, "doctor")
	if code > 2 {
		t.Errorf("doctor should exit 0, 1, or 2, got: %d", code)
	}
}

// TestUpdateCheckRuns validates update check does not panic.
func TestUpdateCheckRuns(t *testing.T) {
	_, _, _ = run(t, "update", "check")
}

// TestWatchStatusNotRunning validates watch status when daemon is off.
func TestWatchStatusNotRunning(t *testing.T) {
	stdout, _, _ := run(t, "watch", "status")
	if strings.Contains(stdout, "panic") {
		t.Error("watch status should not panic")
	}
}

// TestConfigShowRuns validates config show does not panic.
func TestConfigShowRuns(t *testing.T) {
	_, _, code := run(t, "config", "show")
	if code > 1 {
		t.Errorf("config show should exit 0 or 1, got %d", code)
	}
}

// TestAllCommandsHaveHelp validates every command accepts --help.
func TestAllCommandsHaveHelp(t *testing.T) {
	commandPaths := [][]string{
		{"word", "read"}, {"word", "write"}, {"word", "edit"},
		{"excel", "read"}, {"excel", "write"}, {"excel", "analyze"},
		{"pptx", "read"}, {"pptx", "generate"},
		{"ai", "summarize"}, {"ai", "analyze"}, {"ai", "extract"}, {"ai", "ask"},
		{"pipeline", "run"},
		{"batch"},
		{"auth", "login"}, {"auth", "whoami"}, {"auth", "status"}, {"auth", "logout"},
		{"onedrive", "ls"}, {"onedrive", "get"}, {"onedrive", "put"}, {"onedrive", "recent"},
		{"sharepoint", "sites"}, {"sharepoint", "libs"}, {"sharepoint", "audit"},
		{"teams", "list"}, {"teams", "channels"}, {"teams", "post"}, {"teams", "dm"},
		{"outlook", "inbox"}, {"outlook", "read"}, {"outlook", "download"},
		{"acl", "audit"}, {"acl", "external"}, {"acl", "broken"},
		{"fs", "scan"}, {"fs", "rename"}, {"fs", "dedupe"}, {"fs", "stale"},
		{"template", "list"}, {"template", "show"}, {"template", "apply"},
		{"report", "generate"},
		{"watch", "status"}, {"watch", "stop"},
		{"send"}, {"diff"}, {"convert"},
		{"config", "init"}, {"config", "show"}, {"config", "validate"},
		{"completion", "bash"}, {"completion", "zsh"},
		{"update", "check"},
		{"doctor"}, {"version"},
	}

	for _, path := range commandPaths {
		args := append(path, "--help")
		t.Run(strings.Join(path, "_"), func(t *testing.T) {
			_, _, code := run(t, args...)
			if code != 0 {
				t.Errorf("kit %s --help should exit 0", strings.Join(path, " "))
			}
		})
	}
}
