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
	if err := writeLandingAssets(outputDir); err != nil {
		t.Fatalf("write landing assets: %v", err)
	}
	config, err := os.ReadFile(filepath.Join(outputDir, "hugo.toml")) //nolint:gosec // outputDir is a test-owned temporary directory.
	if err != nil {
		t.Fatalf("read Hugo config: %v", err)
	}
	configText := string(config)
	for _, want := range []string{
		`baseURL = "/"`,
		`endLevel = 4`,
		`path = "github.com/imfing/hextra"`,
		`name = "Get started"`,
		`pageRef = "/docs/getting-started"`,
		`name = "Reference"`,
		`pageRef = "/docs/reference"`,
		`name = "Benchmarks"`,
		`pageRef = "/docs/benchmarks"`,
		`type = "search"`,
		`url = "https://github.com/spockz/semantic-editor"`,
		`type = "theme-toggle"`,
		`default = "system"`,
		`displayToggle = true`,
	} {
		if !strings.Contains(configText, want) {
			t.Errorf("Hugo config does not contain %q", want)
		}
	}
	for _, obsolete := range []string{"lotusdocs", "bootstrap", "menu.primary", "prismTheme"} {
		if strings.Contains(strings.ToLower(configText), strings.ToLower(obsolete)) {
			t.Errorf("Hugo config contains obsolete Lotus/Bootstrap setting %q", obsolete)
		}
	}
	if strings.Contains(configText, "spockz.github.io/semantic-editor") {
		t.Fatal("Hugo config must not hardcode the GitHub Pages deployment URL")
	}

	module, err := os.ReadFile(filepath.Join(outputDir, "go.mod")) //nolint:gosec // outputDir is a test-owned temporary directory.
	if err != nil {
		t.Fatalf("read Hugo module go.mod: %v", err)
	}
	moduleText := string(module)
	if !strings.Contains(moduleText, "github.com/imfing/hextra v0.12.3") {
		t.Errorf("Hugo module does not pin Hextra v0.12.3")
	}
	for _, obsolete := range []string{"lotusdocs", "bootstrap"} {
		if strings.Contains(strings.ToLower(moduleText), obsolete) {
			t.Errorf("Hugo module contains obsolete dependency %q", obsolete)
		}
	}

	landing, err := os.ReadFile(filepath.Join(outputDir, "content", "_index.md")) //nolint:gosec // outputDir is a test-owned temporary directory.
	if err != nil {
		t.Fatalf("read Hugo landing page: %v", err)
	}
	landingText := string(landing)
	for _, want := range []string{
		`hextra/hero-badge`,
		`hextra/hero-button`,
		`hextra/feature-grid`,
		`Get started`,
		`View on GitHub`,
		`semantic-workflow-banner.png`,
		`deterministic-edits.png`,
		`symbol-intent.png`,
		`structured-feedback.png`,
		`agent-contract.png`,
	} {
		if !strings.Contains(landingText, want) {
			t.Errorf("Hugo landing page does not contain %q", want)
		}
	}
	for _, name := range landingAssetNames {
		info, err := os.Stat(filepath.Join(outputDir, "static", "images", "landing", name)) //nolint:gosec // outputDir is a test-owned temporary directory.
		if err != nil {
			t.Errorf("stat generated landing asset %q: %v", name, err)
			continue
		}
		if info.Size() == 0 {
			t.Errorf("generated landing asset %q is empty", name)
		}
	}
}

func TestBenchmarkCodeShortcodeUsesHextraPartialsAndSafeDecoding(t *testing.T) {
	t.Parallel()

	for _, want := range []string{
		`{{- $encoded := .Get "content" -}}`,
		`{{- $content := $encoded | base64Decode -}}`,
		`partial "components/codeblock"`,
		`partialCached "components/codeblock-copy-button"`,
		`site.Params.highlight.copy.enable`,
	} {
		if !strings.Contains(benchmarkCodeShortcode, want) {
			t.Errorf("benchmark shortcode source missing %q", want)
		}
	}
	for _, forbidden := range []string{"benchmark-shell-command", "benchmark-tool-arguments", "hextra-code-copy-btn"} {
		if strings.Contains(benchmarkCodeShortcode, forbidden) {
			t.Errorf("benchmark shortcode must not duplicate theme implementation %q", forbidden)
		}
	}
}

func TestParseTxtarExpectedOutputState(t *testing.T) {
	t.Parallel()

	content := `# Scenario with want file and cmp command
exec semedit rename --file api/server.go --symbol Old --to New
cmp api/server.go want/api/server.go

-- api/server.go --
package api

type Old struct{}

-- want/api/server.go --
package api

type New struct{}
`
	ex := parseTxtarFile("test.txtar", content)
	if ex == nil {
		t.Fatal("expected non-nil TxtarExample")
	}
	if len(ex.Outputs) != 1 {
		t.Fatalf("expected 1 output file, got %d", len(ex.Outputs))
	}
	if ex.Outputs[0].Path != "api/server.go" {
		t.Errorf("expected output path %q, got %q", "api/server.go", ex.Outputs[0].Path)
	}
	if !strings.Contains(ex.Outputs[0].Content, "type New struct{}") {
		t.Errorf("expected output content to contain %q, got %q", "type New struct{}", ex.Outputs[0].Content)
	}
	if len(ex.Outputs[0].DiffLines) == 0 {
		t.Error("expected non-empty diff lines for transformation")
	}
}

func TestRenderMarkdownOutputsExpectedState(t *testing.T) {
	t.Parallel()

	page := renderMarkdown(nil, nil, []TxtarExample{{
		Filename: "rename.txtar",
		Title:    "Rename Symbol",
		Steps: []TxtarStep{{
			Description: "Rename symbol",
			Command:     "semedit rename --file api/server.go --symbol Old --to New",
			MCPTool:     "semantic_rename",
			MCPArgsJSON: `{"file":"api/server.go","symbol":"Old","to":"New"}`,
		}},
		Outputs: []TxtarFileOutput{{
			Path:    "api/server.go",
			Content: "package api\n\ntype New struct{}\n",
			DiffLines: []DiffLine{
				{Type: "del", Content: "- type Old struct{}"},
				{Type: "add", Content: "+ type New struct{}"},
			},
		}},
	}})

	wants := []string{
		"**Unified AST transformation diff**",
		"```diff",
		"+ type New struct{}",
		"**Expected output state (`api/server.go`)**",
		"```go",
		"type New struct{}",
	}
	for _, want := range wants {
		if !strings.Contains(page, want) {
			t.Errorf("rendered markdown missing expected string %q", want)
		}
	}
}
