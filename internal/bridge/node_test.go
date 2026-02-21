package bridge

import (
	"os"
	"path/filepath"
	"testing"
)

func bridgePath() string {
	// Walk up from the test directory to find the project root
	dir, _ := os.Getwd()
	for {
		candidate := filepath.Join(dir, "packages", "core", "dist", "cli.js")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

func TestFindCLI(t *testing.T) {
	bp := bridgePath()
	if bp == "" {
		t.Skip("CLI bridge not built, skipping")
	}
	t.Setenv("KIT_BRIDGE_PATH", bp)

	cliPath, err := findCLI()
	if err != nil {
		t.Skipf("CLI bridge not found: %v", err)
	}

	if !filepath.IsAbs(cliPath) {
		t.Errorf("expected absolute path, got %q", cliPath)
	}

	if _, err := os.Stat(cliPath); err != nil {
		t.Errorf("CLI path does not exist: %s", cliPath)
	}
}

func TestFindCLIWithEnvOverride(t *testing.T) {
	t.Setenv("KIT_BRIDGE_PATH", "/nonexistent/path/cli.js")
	_, err := findCLI()
	if err == nil {
		t.Fatal("expected error for nonexistent KIT_BRIDGE_PATH")
	}
}

func TestInvokeGeneratePPTX(t *testing.T) {
	bp := bridgePath()
	if bp == "" {
		t.Skip("CLI bridge not built, skipping")
	}
	t.Setenv("KIT_BRIDGE_PATH", bp)

	outPath := filepath.Join(t.TempDir(), "test_output.pptx")
	req := map[string]any{
		"action": "pptx.generate",
		"output": outPath,
		"options": map[string]string{
			"title": "Test Deck",
		},
		"slides": []map[string]any{
			{
				"title": "Slide 1",
				"content": []map[string]any{
					{"text": "Hello from Go test"},
				},
			},
		},
	}

	result, err := Invoke(req)
	if err != nil {
		t.Fatalf("Invoke failed: %v", err)
	}

	if result["success"] != true {
		t.Errorf("expected success=true, got %v", result["success"])
	}

	info, err := os.Stat(outPath)
	if err != nil {
		t.Fatalf("output file not found: %v", err)
	}
	if info.Size() < 1000 {
		t.Errorf("output file too small: %d bytes", info.Size())
	}
}

func TestInvokeInvalidAction(t *testing.T) {
	bp := bridgePath()
	if bp == "" {
		t.Skip("CLI bridge not built, skipping")
	}
	t.Setenv("KIT_BRIDGE_PATH", bp)

	_, err := Invoke(map[string]any{
		"action": "nonexistent.action",
	})
	if err == nil {
		t.Fatal("expected error for unknown action")
	}
}

func TestInvokeMissingSlides(t *testing.T) {
	bp := bridgePath()
	if bp == "" {
		t.Skip("CLI bridge not built, skipping")
	}
	t.Setenv("KIT_BRIDGE_PATH", bp)

	_, err := Invoke(map[string]any{
		"action": "pptx.generate",
		"output": "/tmp/should_not_exist.pptx",
	})
	if err == nil {
		t.Fatal("expected error for missing slides")
	}
}
