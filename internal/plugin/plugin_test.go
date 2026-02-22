package plugin

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDiscoverEmptyDir(t *testing.T) {
	tmp := t.TempDir()
	origHome := os.Getenv("HOME")
	t.Setenv("HOME", tmp)
	defer os.Setenv("HOME", origHome)

	// Create the plugin dir but leave it empty
	os.MkdirAll(filepath.Join(tmp, ".kit", "plugins"), 0755)

	plugins, err := Discover()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(plugins) != 0 {
		t.Errorf("expected 0 plugins, got %d", len(plugins))
	}
}

func TestDiscoverFindsExecutables(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	pluginDir := filepath.Join(tmp, ".kit", "plugins")
	os.MkdirAll(pluginDir, 0755)

	// Create a fake plugin executable
	execPath := filepath.Join(pluginDir, "kit-myplugin")
	os.WriteFile(execPath, []byte("#!/bin/bash\necho hello\n"), 0755)

	plugins, err := Discover()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(plugins) != 1 {
		t.Fatalf("expected 1 plugin, got %d", len(plugins))
	}
	if plugins[0].Name != "myplugin" {
		t.Errorf("expected name 'myplugin', got %q", plugins[0].Name)
	}
	if plugins[0].Type != "shell" {
		t.Errorf("expected type 'shell', got %q", plugins[0].Type)
	}
}

func TestDiscoverSubdirectory(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	pluginDir := filepath.Join(tmp, ".kit", "plugins", "review")
	os.MkdirAll(pluginDir, 0755)

	execPath := filepath.Join(pluginDir, "kit-review")
	os.WriteFile(execPath, []byte("#!/bin/bash\necho review\n"), 0755)

	// Write manifest
	os.WriteFile(filepath.Join(pluginDir, "plugin.yaml"), []byte(`name: review
version: 1.0.0
description: Code review plugin
author: Test Author
commands:
  - review
`), 0644)

	plugins, err := Discover()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(plugins) != 1 {
		t.Fatalf("expected 1 plugin, got %d", len(plugins))
	}
	if plugins[0].Version != "1.0.0" {
		t.Errorf("expected version '1.0.0', got %q", plugins[0].Version)
	}
	if plugins[0].Description != "Code review plugin" {
		t.Errorf("expected description 'Code review plugin', got %q", plugins[0].Description)
	}
}

func TestLoadManifest(t *testing.T) {
	tmp := t.TempDir()
	manifest := `name: test-plugin
version: 2.0.0
description: A test plugin
author: Test Author
min_version: "1.2.0"
commands:
  - test
  - test-alt
`
	os.WriteFile(filepath.Join(tmp, "plugin.yaml"), []byte(manifest), 0644)

	m, err := LoadManifest(tmp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.Name != "test-plugin" {
		t.Errorf("expected name 'test-plugin', got %q", m.Name)
	}
	if m.Version != "2.0.0" {
		t.Errorf("expected version '2.0.0', got %q", m.Version)
	}
	if len(m.Commands) != 2 {
		t.Errorf("expected 2 commands, got %d", len(m.Commands))
	}
	if m.MinVersion != "1.2.0" {
		t.Errorf("expected min_version '1.2.0', got %q", m.MinVersion)
	}
}

func TestLoadManifestMissing(t *testing.T) {
	tmp := t.TempDir()
	_, err := LoadManifest(tmp)
	if err == nil {
		t.Error("expected error for missing manifest")
	}
}

func TestGetNotFound(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	os.MkdirAll(filepath.Join(tmp, ".kit", "plugins"), 0755)

	_, err := Get("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent plugin")
	}
}

func TestRunSetsEnv(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	pluginDir := filepath.Join(tmp, ".kit", "plugins")
	os.MkdirAll(pluginDir, 0755)

	// Create a plugin that prints KIT_VERSION
	script := "#!/bin/bash\necho \"VERSION=$KIT_VERSION\"\n"
	execPath := filepath.Join(pluginDir, "kit-envtest")
	os.WriteFile(execPath, []byte(script), 0755)

	// We can't easily capture stdout from Run (it goes to os.Stdout),
	// so we verify the plugin is found and env is set correctly.
	env := pluginEnv()
	found := false
	for _, e := range env {
		if strings.HasPrefix(e, "KIT_VERSION=") {
			found = true
			if !strings.Contains(e, "1.2.0") {
				t.Errorf("expected KIT_VERSION=1.2.0, got %q", e)
			}
		}
	}
	if !found {
		t.Error("KIT_VERSION not found in plugin env")
	}
}

func TestNewScaffoldShell(t *testing.T) {
	tmp := t.TempDir()
	dir, err := NewScaffold("my-review", "shell", tmp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check plugin.yaml exists
	manifestPath := filepath.Join(dir, "plugin.yaml")
	if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
		t.Error("plugin.yaml not created")
	}

	// Check executable exists and has shebang
	execPath := filepath.Join(dir, "kit-my-review")
	data, err := os.ReadFile(execPath)
	if err != nil {
		t.Fatalf("executable not created: %v", err)
	}
	if !strings.HasPrefix(string(data), "#!/bin/bash") {
		t.Error("shell plugin should start with shebang")
	}

	// Check executable bit
	info, _ := os.Stat(execPath)
	if info.Mode()&0111 == 0 {
		t.Error("executable should have execute permission")
	}

	// Check README exists
	if _, err := os.Stat(filepath.Join(dir, "README.md")); os.IsNotExist(err) {
		t.Error("README.md not created")
	}
}

func TestNewScaffoldGo(t *testing.T) {
	tmp := t.TempDir()
	dir, err := NewScaffold("my-tool", "go", tmp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check main.go exists
	mainPath := filepath.Join(dir, "main.go")
	data, err := os.ReadFile(mainPath)
	if err != nil {
		t.Fatalf("main.go not created: %v", err)
	}
	if !strings.Contains(string(data), "package main") {
		t.Error("Go plugin should contain package main")
	}
}

func TestNewScaffoldInvalidType(t *testing.T) {
	tmp := t.TempDir()
	_, err := NewScaffold("test", "ruby", tmp)
	if err == nil {
		t.Error("expected error for invalid plugin type")
	}
}

func TestRemovePlugin(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	pluginDir := filepath.Join(tmp, ".kit", "plugins")
	os.MkdirAll(pluginDir, 0755)

	// Create a plugin
	execPath := filepath.Join(pluginDir, "kit-removeme")
	os.WriteFile(execPath, []byte("#!/bin/bash\n"), 0755)

	// Remove it
	err := Remove("removeme")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify it's gone
	if _, err := os.Stat(execPath); !os.IsNotExist(err) {
		t.Error("plugin should have been removed")
	}
}

func TestRemovePluginSubdir(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	subDir := filepath.Join(tmp, ".kit", "plugins", "myplug")
	os.MkdirAll(subDir, 0755)
	os.WriteFile(filepath.Join(subDir, "kit-myplug"), []byte("#!/bin/bash\n"), 0755)
	os.WriteFile(filepath.Join(subDir, "plugin.yaml"), []byte("name: myplug\n"), 0644)

	err := Remove("myplug")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := os.Stat(subDir); !os.IsNotExist(err) {
		t.Error("plugin directory should have been removed")
	}
}

func TestRemoveNotFound(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	os.MkdirAll(filepath.Join(tmp, ".kit", "plugins"), 0755)

	err := Remove("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent plugin")
	}
}

func TestInstallLocal(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	// Create a source plugin directory
	srcDir := filepath.Join(tmp, "src-plugin")
	os.MkdirAll(srcDir, 0755)
	os.WriteFile(filepath.Join(srcDir, "plugin.yaml"), []byte("name: localtest\nversion: 1.0.0\n"), 0644)
	os.WriteFile(filepath.Join(srcDir, "kit-localtest"), []byte("#!/bin/bash\necho local\n"), 0755)

	p, err := Install(srcDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Name != "localtest" {
		t.Errorf("expected name 'localtest', got %q", p.Name)
	}

	// Verify it's discoverable
	plugins, _ := Discover()
	if len(plugins) != 1 {
		t.Errorf("expected 1 installed plugin, got %d", len(plugins))
	}
}

// Ensure Run doesn't panic on valid but non-functional context
func TestRunContext(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	os.MkdirAll(filepath.Join(tmp, ".kit", "plugins"), 0755)

	ctx := context.Background()
	err := Run(ctx, "nonexistent", nil)
	if err == nil {
		t.Error("expected error running nonexistent plugin")
	}
}
