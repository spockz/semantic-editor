// Tests ensure documentation generation preserves valid output while reporting every independent failure.

package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestGenerateDocumentationCollectsFailures(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name           string
		valid          bool
		blockedOutputs bool
	}{
		{name: "valid siblings", valid: true},
		{name: "no valid reports"},
		{name: "independent output failures", valid: true, blockedOutputs: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rootDir := prepareErrorCollectionRoot(t, tc.valid)
			outputDir := prepareErrorCollectionOutput(t, tc.blockedOutputs)
			err := generateDocumentation(rootDir, outputDir, false)
			assertErrorCollectionDiagnostics(t, err, tc.blockedOutputs)
			assertErrorCollectionArtifacts(t, outputDir, tc.valid)
		})
	}
}

func prepareErrorCollectionRoot(t *testing.T, valid bool) string {
	t.Helper()
	rootDir := t.TempDir()
	files := map[string]string{
		"internal/astedit/insert.go":                            "package astedit\n",
		"testdata/scripts/example.txtar":                        "# Example\n",
		"cmd/docgen/assets/vendor/perspective/css/pro-dark.css": "body {}",
		"data/benchmarks/results/bad-json/report.json":          "{",
		"data/benchmarks/results/bad-version/report.json":       `{"format_version":1}`,
		"data/benchmarks/results/bad-browser/report.json":       `{"comparisons":[null]}`,
	}
	for path, content := range files {
		target := filepath.Join(rootDir, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(target), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := writeGeneratedFile(target, []byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if valid {
		writeBenchmarkDocumentationReport(t, rootDir, "valid-run", &BenchComparisonSummary{
			TaskID: "valid-task", Target: BenchTarget{Harness: "test", Model: "test-model"},
			SmallBaseline: benchmarkDocumentationRunResult(true, time.Second, 100),
			SmallSemedit:  benchmarkDocumentationRunResult(true, time.Second, 100),
		})
	}
	return rootDir
}

func prepareErrorCollectionOutput(t *testing.T, blocked bool) string {
	t.Helper()
	outputDir := t.TempDir()
	if err := prepareOutputDirectories(outputDir); err != nil {
		t.Fatal(err)
	}
	stalePath := filepath.Join(outputDir, "static", "data", "benchmarks", "runs", "bad-version.json")
	if err := os.MkdirAll(filepath.Dir(stalePath), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := writeGeneratedFile(stalePath, []byte(`[]`)); err != nil {
		t.Fatal(err)
	}
	if err := removeStaleDocumentation(outputDir); err != nil {
		t.Fatal(err)
	}
	if blocked {
		for _, path := range []string{"content/docs/reference.md", "hugo.toml"} {
			if err := os.MkdirAll(filepath.Join(outputDir, filepath.FromSlash(path)), 0o750); err != nil {
				t.Fatal(err)
			}
		}
	}
	return outputDir
}

func assertErrorCollectionDiagnostics(t *testing.T, err error, blocked bool) {
	t.Helper()
	if err == nil {
		t.Fatal("expected collected failures")
	}
	previous := -1
	for _, path := range []string{"bad-browser/report.json", "bad-json/report.json", "bad-version/report.json"} {
		index := strings.Index(err.Error(), path)
		if index <= previous {
			t.Errorf("missing or unordered failure %s: %v", path, err)
		}
		previous = index
	}
	if blocked {
		for _, phase := range []string{"writing Markdown output", "writing Hugo configuration"} {
			if !strings.Contains(err.Error(), phase) {
				t.Errorf("error missing %s: %v", phase, err)
			}
		}
	}
}

func assertErrorCollectionArtifacts(t *testing.T, outputDir string, valid bool) {
	t.Helper()
	paths := []string{"content/docs/getting-started.md", "content/docs/benchmarks/_index.md", "content/_index.md", "layouts/shortcodes/benchmark-browser.html", "static/data/benchmarks.json"}
	if valid {
		paths = append(paths, "content/docs/benchmarks/runs/valid-run/index.md", "static/data/benchmarks/runs/valid-run.json")
	}
	for _, path := range paths {
		if _, err := os.Stat(filepath.Join(outputDir, filepath.FromSlash(path))); err != nil {
			t.Errorf("independent output %s missing: %v", path, err)
		}
	}
	data, readErr := os.ReadFile(filepath.Join(outputDir, "static", "data", "benchmarks.json"))
	if readErr != nil {
		t.Fatal(readErr)
	}
	var rows []map[string]json.RawMessage
	if err := json.Unmarshal(data, &rows); err != nil {
		t.Fatal(err)
	}
	if valid != (len(rows) > 0) {
		t.Fatalf("browser rows do not match valid reports: %s", data)
	}
	for _, runID := range []string{"bad-json", "bad-version", "bad-browser"} {
		for _, path := range []string{filepath.Join(outputDir, "content", "docs", "benchmarks", "runs", runID, "index.md"), filepath.Join(outputDir, "static", "data", "benchmarks", "runs", runID+".json")} {
			if _, err := os.Stat(path); !os.IsNotExist(err) {
				t.Errorf("rejected report output exists or stat failed: %s: %v", path, err)
			}
		}
	}
	shortcode, readErr := os.ReadFile(filepath.Join(outputDir, "layouts", "shortcodes", "benchmark-browser.html"))
	if readErr != nil {
		t.Fatal(readErr)
	}
	if !strings.Contains(string(shortcode), "No valid benchmark results are available.") {
		t.Fatal("empty dataset message missing")
	}
}

func TestGenerateDocumentationContinuesAfterCapabilityFailure(t *testing.T) {
	t.Parallel()
	rootDir := prepareErrorCollectionRoot(t, true)
	if err := os.Remove(filepath.Join(rootDir, "internal", "astedit", "insert.go")); err != nil {
		t.Fatal(err)
	}
	outputDir := prepareErrorCollectionOutput(t, false)
	referencePath := filepath.Join(outputDir, "content", "docs", "reference.md")
	if err := writeGeneratedFile(referencePath, []byte("stale reference")); err != nil {
		t.Fatal(err)
	}
	if err := removeStaleDocumentation(outputDir); err != nil {
		t.Fatal(err)
	}
	err := generateDocumentation(rootDir, outputDir, false)
	if err == nil || !strings.Contains(err.Error(), "extracting code capabilities") {
		t.Fatalf("capability error missing: %v", err)
	}
	if _, err := os.Stat(referencePath); !os.IsNotExist(err) {
		t.Errorf("stale reference remains or stat failed: %v", err)
	}
	assertErrorCollectionArtifacts(t, outputDir, true)
}

func TestGenerateDocumentationIgnoresBenchmarkOperationalArtifacts(t *testing.T) {
	t.Parallel()
	rootDir := prepareErrorCollectionRoot(t, true)
	beforeOutput := prepareErrorCollectionOutput(t, false)
	beforeErr := generateDocumentation(rootDir, beforeOutput, false)
	assertErrorCollectionDiagnostics(t, beforeErr, false)
	beforeRows, err := os.ReadFile(filepath.Join(beforeOutput, "static", "data", "benchmarks.json"))
	if err != nil {
		t.Fatal(err)
	}
	report, err := os.ReadFile(filepath.Join(rootDir, "data", "benchmarks", "results", "valid-run", "report.json"))
	if err != nil {
		t.Fatal(err)
	}
	artifacts := map[string]any{
		"manifest.json":                        map[string]any{"format_version": 1, "job_ids": []string{"job-1"}},
		"jobs/job-1.json":                      map[string]any{"format_version": 1, "report": json.RawMessage(report)},
		"jobs/nested/report.json":              json.RawMessage(report),
		"evidence/nested/report.json":          json.RawMessage(report),
		"evidence/transcript.json":             []any{map[string]any{"response": "finished"}},
		"evidence/transcript.mcp-metrics.json": []any{map[string]any{"tool": "semantic_verify"}},
		"smoke-summary.json":                   []any{map[string]any{"status": "completed"}},
	}
	for path, value := range artifacts {
		data, marshalErr := json.Marshal(value)
		if marshalErr != nil {
			t.Fatal(marshalErr)
		}
		target := filepath.Join(rootDir, "data", "benchmarks", "results", "valid-run", filepath.FromSlash(path))
		if mkdirErr := os.MkdirAll(filepath.Dir(target), 0o750); mkdirErr != nil {
			t.Fatal(mkdirErr)
		}
		if writeErr := writeGeneratedFile(target, data); writeErr != nil {
			t.Fatal(writeErr)
		}
	}
	afterOutput := prepareErrorCollectionOutput(t, false)
	afterErr := generateDocumentation(rootDir, afterOutput, false)
	if afterErr == nil || afterErr.Error() != beforeErr.Error() {
		t.Fatalf("operational artifacts changed report failures: before %v; after %v", beforeErr, afterErr)
	}
	assertErrorCollectionArtifacts(t, afterOutput, true)
	afterRows, err := os.ReadFile(filepath.Join(afterOutput, "static", "data", "benchmarks.json"))
	if err != nil {
		t.Fatal(err)
	}
	if string(afterRows) != string(beforeRows) {
		t.Fatal("operational artifacts changed published browser rows")
	}
}
