// Package bridge provides the Go→Node subprocess bridge for TypeScript-powered features.
package bridge

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// Invoke sends a JSON request to the Node.js CLI bridge and returns the parsed response.
func Invoke(request any) (map[string]any, error) {
	cliPath, err := findCLI()
	if err != nil {
		return nil, err
	}

	input, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("could not marshal bridge request: %w", err)
	}

	cmd := exec.Command("node", cliPath)
	cmd.Stdin = bytes.NewReader(input)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		msg := stderr.String()
		if msg == "" {
			msg = err.Error()
		}
		return nil, fmt.Errorf("node bridge failed: %s", msg)
	}

	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		return nil, fmt.Errorf("could not parse bridge response: %w\nraw output: %s", err, stdout.String())
	}

	if errMsg, ok := result["error"]; ok {
		return nil, fmt.Errorf("bridge error: %v", errMsg)
	}

	return result, nil
}

// findCLI locates the compiled Node CLI bridge (dist/cli.js).
// It searches in order:
//  1. KIT_BRIDGE_PATH env var (for testing/overrides)
//  2. Relative to the running binary (../../packages/core/dist/cli.js)
//  3. Relative to the current working directory (packages/core/dist/cli.js)
func findCLI() (string, error) {
	// 1. Environment override
	if p := os.Getenv("KIT_BRIDGE_PATH"); p != "" {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
		return "", fmt.Errorf("KIT_BRIDGE_PATH=%q does not exist", p)
	}

	candidates := []string{}

	// 2. Relative to the binary
	if exePath, err := os.Executable(); err == nil {
		binDir := filepath.Dir(exePath)
		// Binary could be in bin/, project root, or installed globally
		candidates = append(candidates,
			filepath.Join(binDir, "..", "packages", "core", "dist", "cli.js"),
			filepath.Join(binDir, "packages", "core", "dist", "cli.js"),
		)
		// If on macOS, resolve symlinks
		if runtime.GOOS == "darwin" {
			if resolved, err := filepath.EvalSymlinks(exePath); err == nil {
				resolvedDir := filepath.Dir(resolved)
				candidates = append(candidates,
					filepath.Join(resolvedDir, "..", "packages", "core", "dist", "cli.js"),
				)
			}
		}
	}

	// 3. Relative to working directory
	if wd, err := os.Getwd(); err == nil {
		candidates = append(candidates,
			filepath.Join(wd, "packages", "core", "dist", "cli.js"),
		)
	}

	for _, c := range candidates {
		abs, err := filepath.Abs(c)
		if err != nil {
			continue
		}
		if _, err := os.Stat(abs); err == nil {
			return abs, nil
		}
	}

	return "", fmt.Errorf("could not find the M365Kit Node bridge — run 'cd packages/core && npm install && npm run build' first")
}
