// Package plugin provides plugin discovery, installation, and execution.
// Plugins are executables named kit-<name> in ~/.kit/plugins/ or $PATH.
package plugin

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Plugin represents a discovered plugin.
type Plugin struct {
	Name        string    `json:"name"`
	Version     string    `json:"version"`
	Description string    `json:"description"`
	Author      string    `json:"author"`
	Path        string    `json:"path"`
	Type        string    `json:"type"`   // "shell" | "go" | "script"
	Source      string    `json:"source"` // install source or "local"
	InstalledAt time.Time `json:"installed_at"`
	Manifest    *Manifest `json:"-"`
}

// Manifest is the metadata file for a plugin (plugin.yaml alongside executable).
type Manifest struct {
	Name        string   `yaml:"name" json:"name"`
	Version     string   `yaml:"version" json:"version"`
	Description string   `yaml:"description" json:"description"`
	Author      string   `yaml:"author" json:"author"`
	MinVersion  string   `yaml:"min_version" json:"min_version"`
	Commands    []string `yaml:"commands" json:"commands"`
}

// Dir returns the plugin directory (~/.kit/plugins/).
func Dir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine home directory: %w", err)
	}
	return filepath.Join(home, ".kit", "plugins"), nil
}

// EnsureDir creates the plugin directory if it does not exist.
func EnsureDir() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("cannot create plugin directory: %w", err)
	}
	return dir, nil
}

// Discover returns all installed plugins.
func Discover() ([]Plugin, error) {
	dir, err := Dir()
	if err != nil {
		return nil, err
	}

	var plugins []Plugin

	// 1. Direct executables: ~/.kit/plugins/kit-<name>
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() {
			// Check subdirectory: ~/.kit/plugins/<name>/kit-<name>
			subExec := filepath.Join(dir, name, "kit-"+name)
			if isExecutable(subExec) {
				p := pluginFromPath(subExec, name)
				plugins = append(plugins, p)
			}
			continue
		}
		if strings.HasPrefix(name, "kit-") {
			fullPath := filepath.Join(dir, name)
			if isExecutable(fullPath) {
				pluginName := strings.TrimPrefix(name, "kit-")
				p := pluginFromPath(fullPath, pluginName)
				plugins = append(plugins, p)
			}
		}
	}

	return plugins, nil
}

// Get returns a specific plugin by name.
func Get(name string) (*Plugin, error) {
	plugins, err := Discover()
	if err != nil {
		return nil, err
	}
	for _, p := range plugins {
		if p.Name == name {
			return &p, nil
		}
	}

	// Also check $PATH for kit-<name>
	binName := "kit-" + name
	pathExec, err := exec.LookPath(binName)
	if err == nil {
		p := pluginFromPath(pathExec, name)
		return &p, nil
	}

	return nil, fmt.Errorf("plugin %q not found", name)
}

// Install installs a plugin from a local directory.
func Install(source string) (*Plugin, error) {
	dir, err := EnsureDir()
	if err != nil {
		return nil, err
	}

	// Local install: copy the directory contents
	source = strings.TrimPrefix(source, "file://")
	info, err := os.Stat(source)
	if err != nil {
		return nil, fmt.Errorf("source not found: %w", err)
	}

	if info.IsDir() {
		return installFromDir(source, dir)
	}
	return installFromFile(source, dir)
}

func installFromDir(source, pluginDir string) (*Plugin, error) {
	// Read manifest to determine plugin name
	manifest, err := LoadManifest(source)
	if err != nil {
		return nil, fmt.Errorf("cannot read plugin manifest: %w", err)
	}

	destDir := filepath.Join(pluginDir, manifest.Name)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return nil, err
	}

	// Copy all files from source to destDir
	entries, err := os.ReadDir(source)
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		srcFile := filepath.Join(source, entry.Name())
		dstFile := filepath.Join(destDir, entry.Name())
		data, err := os.ReadFile(srcFile)
		if err != nil {
			return nil, err
		}
		info, _ := entry.Info()
		if err := os.WriteFile(dstFile, data, info.Mode()); err != nil {
			return nil, err
		}
	}

	execPath := filepath.Join(destDir, "kit-"+manifest.Name)
	p := pluginFromPath(execPath, manifest.Name)
	p.Source = "local"
	return &p, nil
}

func installFromFile(source, pluginDir string) (*Plugin, error) {
	data, err := os.ReadFile(source)
	if err != nil {
		return nil, err
	}
	name := strings.TrimPrefix(filepath.Base(source), "kit-")
	dest := filepath.Join(pluginDir, filepath.Base(source))
	if err := os.WriteFile(dest, data, 0755); err != nil {
		return nil, err
	}
	p := pluginFromPath(dest, name)
	p.Source = "local"
	return &p, nil
}

// Remove uninstalls a plugin by name.
func Remove(name string) error {
	dir, err := Dir()
	if err != nil {
		return err
	}

	// Check direct executable
	direct := filepath.Join(dir, "kit-"+name)
	if _, err := os.Stat(direct); err == nil {
		return os.Remove(direct)
	}

	// Check subdirectory
	subDir := filepath.Join(dir, name)
	if _, err := os.Stat(subDir); err == nil {
		return os.RemoveAll(subDir)
	}

	return fmt.Errorf("plugin %q not found", name)
}

// Run executes a plugin with args, forwarding stdin/stdout/stderr.
func Run(ctx context.Context, name string, args []string) error {
	p, err := Get(name)
	if err != nil {
		return err
	}

	cmd := exec.CommandContext(ctx, p.Path, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Set M365Kit environment variables
	cmd.Env = append(os.Environ(), pluginEnv()...)

	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		return err
	}
	return nil
}

// LoadManifest reads plugin.yaml from a directory.
func LoadManifest(dir string) (*Manifest, error) {
	path := filepath.Join(dir, "plugin.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("plugin.yaml not found in %s", dir)
		}
		return nil, err
	}
	var m Manifest
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("invalid plugin.yaml: %w", err)
	}
	return &m, nil
}

// NewScaffold creates a new plugin scaffold in outputDir.
func NewScaffold(name, pluginType, outputDir string) (string, error) {
	dir := filepath.Join(outputDir, name)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}

	// Write plugin.yaml
	manifest := fmt.Sprintf(`name: %s
version: 0.1.0
description: "A custom M365Kit plugin"
author: ""
min_version: "1.2.0"
commands:
  - %s
`, name, name)
	if err := os.WriteFile(filepath.Join(dir, "plugin.yaml"), []byte(manifest), 0644); err != nil {
		return "", err
	}

	execName := "kit-" + name
	switch pluginType {
	case "shell":
		script := fmt.Sprintf(`#!/bin/bash
# M365Kit plugin: %s
# Usage: kit %s <args>
#
# Environment variables provided by M365Kit:
#   KIT_VERSION     — current M365Kit version
#   KIT_CONFIG_PATH — path to config.yaml
#   KIT_JSON        — "true" if --json output requested

set -euo pipefail

if [ $# -eq 0 ]; then
  echo "Usage: kit %s <args>" >&2
  exit 1
fi

echo "Plugin %s processing: $1"
`, name, name, name, name)
		if err := os.WriteFile(filepath.Join(dir, execName), []byte(script), 0755); err != nil {
			return "", err
		}
	case "go":
		goMain := fmt.Sprintf(`package main

import (
	"fmt"
	"os"
)

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: kit %s <args>")
		os.Exit(1)
	}

	version := os.Getenv("KIT_VERSION")
	jsonOutput := os.Getenv("KIT_JSON") == "true"

	_ = version
	_ = jsonOutput

	fmt.Printf("Plugin %s processing: %%s\n", args[0])
}
`, name, name)
		if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte(goMain), 0644); err != nil {
			return "", err
		}
		// Write a placeholder executable
		if err := os.WriteFile(filepath.Join(dir, execName), []byte("#!/bin/bash\ngo run . \"$@\"\n"), 0755); err != nil {
			return "", err
		}
	default:
		return "", fmt.Errorf("unsupported plugin type: %s (use shell or go)", pluginType)
	}

	// Write README.md
	readme := fmt.Sprintf("# kit-%s\n\nA custom M365Kit plugin.\n\n## Usage\n\n```bash\nkit %s <args>\n```\n", name, name)
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte(readme), 0644); err != nil {
		return "", err
	}

	return dir, nil
}

func pluginFromPath(path, name string) Plugin {
	p := Plugin{
		Name: name,
		Path: path,
		Type: detectType(path),
	}

	// Try to load manifest from same directory
	dir := filepath.Dir(path)
	if m, err := LoadManifest(dir); err == nil {
		p.Manifest = m
		p.Version = m.Version
		p.Description = m.Description
		p.Author = m.Author
	}

	info, err := os.Stat(path)
	if err == nil {
		p.InstalledAt = info.ModTime()
	}

	return p
}

func detectType(path string) string {
	data, err := os.ReadFile(path)
	if err != nil || len(data) == 0 {
		return "script"
	}
	if strings.HasPrefix(string(data), "#!/") {
		return "shell"
	}
	// Check for ELF or Mach-O magic bytes (compiled binary)
	if len(data) >= 4 {
		if data[0] == 0x7f && data[1] == 'E' && data[2] == 'L' && data[3] == 'F' {
			return "go"
		}
		// Mach-O
		if data[0] == 0xfe && data[1] == 0xed && data[2] == 0xfa {
			return "go"
		}
		if data[0] == 0xcf && data[1] == 0xfa && data[2] == 0xed && data[3] == 0xfe {
			return "go"
		}
	}
	return "script"
}

func isExecutable(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	if runtime.GOOS == "windows" {
		// On Windows, check for common executable extensions
		ext := strings.ToLower(filepath.Ext(path))
		return ext == ".exe" || ext == ".bat" || ext == ".cmd"
	}
	return info.Mode()&0111 != 0
}

func pluginEnv() []string {
	home, _ := os.UserHomeDir()
	return []string{
		"KIT_VERSION=1.2.0",
		"KIT_CONFIG_PATH=" + filepath.Join(home, ".kit", "config.yaml"),
		"KIT_TOKEN_PATH=" + filepath.Join(home, ".kit", "token.json"),
		"KIT_JSON=" + boolEnv(os.Getenv("KIT_JSON")),
		"KIT_VERBOSE=" + boolEnv(os.Getenv("KIT_VERBOSE")),
	}
}

func boolEnv(v string) string {
	if v == "true" || v == "1" {
		return "true"
	}
	return "false"
}
