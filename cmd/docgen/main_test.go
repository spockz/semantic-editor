// Tests protect the generated documentation contract so published examples remain usable without manual presentation fixes.
package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderMarkdownIncludesCopyableExamplesAndSourceLinks(t *testing.T) {
	t.Parallel()

	const filename = "insert_specialized.txtar"
	page := renderMarkdown(nil, nil, []TxtarExample{{
		Filename: filename,
		Title:    "Specialized insertion",
		Steps: []TxtarStep{{
			Description: "Insert a type",
			Command:     "semedit insert-type --file api/server.go --source 'type Config struct{}'",
			MCPTool:     "semantic_insert_type",
			MCPArgsJSON: "{\"file\":\"api/server.go\"}",
		}},
	}})

	wants := []string{
		"```console",
		"```json",
		"[View the semedit repository on GitHub](https://github.com/spockz/semantic-editor)",
		`Source: [` + filename + `](https://github.com/spockz/semantic-editor/blob/main/testdata/scripts/` + filename + `)`,
		`[Download raw file](https://raw.githubusercontent.com/spockz/semantic-editor/main/testdata/scripts/` + filename + `)`,
	}
	for _, want := range wants {
		if !strings.Contains(page, want) {
			t.Errorf("rendered documentation does not contain %q", want)
		}
	}
}

func TestGeneratedDocWeightsPutGettingStartedFirst(t *testing.T) {
	t.Parallel()

	reference := renderMarkdown(nil, nil, nil)
	gettingStarted := renderGettingStarted()
	if !strings.Contains(reference, "weight: 10") {
		t.Fatal("reference page must sort after Getting Started in the docs sidebar")
	}
	if !strings.Contains(gettingStarted, "weight: 1") {
		t.Fatal("Getting Started page must sort first in the docs sidebar")
	}
}

func TestRenderGettingStartedTargetsSupportedPlatforms(t *testing.T) {
	t.Parallel()

	page := renderGettingStarted()
	macOS := strings.Index(page, "## macOS with Homebrew")
	linux := strings.Index(page, "## Linux")
	if macOS < 0 || linux < 0 || macOS > linux {
		t.Fatalf("expected macOS instructions before Linux instructions")
	}
	for _, unsupported := range []string{"Windows", "Chocolatey", "Scoop"} {
		if strings.Contains(page, unsupported) {
			t.Errorf("getting started page contains unsupported platform or package manager %q", unsupported)
		}
	}
	if strings.Contains(page, "\n# Getting Started\n") {
		t.Fatal("getting started content must not duplicate the theme-rendered page title")
	}
}

func TestWriteHugoConfigUsesDeploymentNeutralBaseURL(t *testing.T) {
	t.Parallel()

	outputDir := t.TempDir()
	for _, directory := range []string{
		filepath.Join(outputDir, "content", "docs"),
		filepath.Join(outputDir, "data"),
	} {
		if err := os.MkdirAll(directory, 0o750); err != nil {
			t.Fatalf("create Hugo source directory: %v", err)
		}
	}
	if err := writeHugoConfig(outputDir); err != nil {
		t.Fatalf("write Hugo config: %v", err)
	}
	config, err := os.ReadFile(filepath.Join(outputDir, "hugo.toml")) //nolint:gosec // outputDir is a test-owned temporary directory.
	if err != nil {
		t.Fatalf("read Hugo config: %v", err)
	}
	configText := string(config)
	for _, want := range []string{
		`baseURL = "/"`,
		`google_fonts = [["Inter", "300, 400, 600, 700"], ["Fira Code", "400, 500, 600, 700"]]`,
		`sans_serif_font = "Inter, -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif"`,
		`secondary_font = "Inter, -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif"`,
		`mono_font = "'Fira Code', SFMono-Regular, Menlo, Monaco, Consolas, monospace"`,
	} {
		if !strings.Contains(configText, want) {
			t.Errorf("Hugo config does not contain %q", want)
		}
	}
	if strings.Contains(configText, "spockz.github.io/semantic-editor") {
		t.Fatal("Hugo config must not hardcode the GitHub Pages deployment URL")
	}
}
