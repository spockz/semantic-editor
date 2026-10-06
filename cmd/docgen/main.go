// Package main synthesizes code-derived capability documentation and test-driven examples into a Hugo source tree.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	if err := runDocgen(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error %v\n", err)
		os.Exit(1)
	}
}

type docgenOptions struct {
	outputDir string
	enableRSS bool
}

func runDocgen(args []string) error {
	rootDir, rootErr := findRepoRoot()
	if rootErr != nil {
		return fmt.Errorf("locating repo root: %w", rootErr)
	}
	opts, parseErr := parseDocgenOptions(rootDir, args)
	if parseErr != nil {
		return parseErr
	}
	if err := prepareOutputDirectories(opts.outputDir); err != nil {
		return err
	}
	if err := removeStaleDocumentation(opts.outputDir); err != nil {
		return err
	}
	return generateDocumentation(rootDir, opts.outputDir, opts.enableRSS)
}

func parseDocgenOptions(rootDir string, args []string) (docgenOptions, error) {
	opts := docgenOptions{
		outputDir: filepath.Join(rootDir, ".scratch", "docgen"),
	}
	flags := flag.NewFlagSet("docgen", flag.ExitOnError)
	flags.StringVar(&opts.outputDir, "output-dir", opts.outputDir, "directory for the generated Hugo source tree")
	flags.BoolVar(&opts.enableRSS, "enable-rss", false, "enable RSS feed generation in Hugo configuration")
	if err := flags.Parse(args); err != nil {
		return docgenOptions{}, fmt.Errorf("parsing flags: %w", err)
	}
	return opts, nil
}

func prepareOutputDirectories(outputDir string) error {
	if err := os.MkdirAll(filepath.Join(outputDir, "content", "docs"), 0o750); err != nil {
		return fmt.Errorf("creating documentation source directory: %w", err)
	}
	if err := os.MkdirAll(filepath.Join(outputDir, "data"), 0o750); err != nil {
		return fmt.Errorf("creating Hugo data directory: %w", err)
	}
	shortcodesDir := filepath.Join(outputDir, "layouts", "shortcodes")
	if err := os.RemoveAll(shortcodesDir); err != nil {
		return fmt.Errorf("removing stale generated shortcodes: %w", err)
	}
	if err := os.MkdirAll(shortcodesDir, 0o750); err != nil {
		return fmt.Errorf("creating Hugo shortcode directory: %w", err)
	}
	return nil
}

func removeStaleDocumentation(outputDir string) error {
	for _, stalePath := range []string{
		filepath.Join(outputDir, "content", "docs", "index.md"),
		filepath.Join(outputDir, "content", "docs", "getting-started"),
		filepath.Join(outputDir, "content", "docs", "reference"),
		filepath.Join(outputDir, "content", "docs", "benchmarks.md"),
		filepath.Join(outputDir, "content", "docs", "benchmarks"),
		filepath.Join(outputDir, "data", "landing.yaml"),
	} {
		if err := os.RemoveAll(stalePath); err != nil {
			return fmt.Errorf("removing stale documentation output: %w", err)
		}
	}
	return nil
}

func generateDocumentation(rootDir, outputDir string, enableRSS bool) error {
	capabilities, placements, capabilitiesErr := extractCodeCapabilities(rootDir)
	if capabilitiesErr != nil {
		return fmt.Errorf("extracting code capabilities: %w", capabilitiesErr)
	}
	examples, examplesErr := extractTxtarExamples(rootDir)
	if examplesErr != nil {
		return fmt.Errorf("extracting txtar examples: %w", examplesErr)
	}
	if writeErr := writeReferenceDocumentation(outputDir, capabilities, placements, examples); writeErr != nil {
		return writeErr
	}
	documentation, renderErr := renderBenchmarkDocumentation(rootDir)
	if renderErr != nil {
		return fmt.Errorf("rendering benchmark documentation: %w", renderErr)
	}
	if writeErr := writeBenchmarkDocumentation(rootDir, outputDir, documentation); writeErr != nil {
		return writeErr
	}
	if err := writeLandingAssets(outputDir); err != nil {
		return fmt.Errorf("writing landing assets: %w", err)
	}
	if err := writeHugoConfig(outputDir, enableRSS); err != nil {
		return fmt.Errorf("writing Hugo configuration: %w", err)
	}
	fmt.Printf("Successfully synthesized Hugo documentation source: %s\n", filepath.Join(outputDir, "content", "docs", "reference.md"))
	fmt.Printf("Extracted %d languages, %d placement modes, %d txtar workflows.\n", len(capabilities), len(placements), len(examples))
	return nil
}

func writeReferenceDocumentation(outputDir string, capabilities []CodeCapability, placements []string, examples []TxtarExample) error {
	docsDir := filepath.Join(outputDir, "content", "docs")
	if err := writeGeneratedFile(filepath.Join(docsDir, "reference.md"), []byte(renderMarkdown(capabilities, placements, examples))); err != nil {
		return fmt.Errorf("writing Markdown output: %w", err)
	}
	if err := writeGeneratedFile(filepath.Join(docsDir, "getting-started.md"), []byte(renderGettingStarted())); err != nil {
		return fmt.Errorf("writing getting started output: %w", err)
	}
	return nil
}

func writeBenchmarkDocumentation(rootDir, outputDir string, documentation benchmarkDocumentation) error {
	if err := writeBenchmarkBrowserAssets(rootDir, outputDir); err != nil {
		return fmt.Errorf("generating benchmark browser assets: %w", err)
	}
	benchmarksDir := filepath.Join(outputDir, "content", "docs", "benchmarks")
	if err := os.MkdirAll(benchmarksDir, 0o750); err != nil {
		return fmt.Errorf("creating benchmarks documentation directory: %w", err)
	}
	if err := writeGeneratedFile(filepath.Join(benchmarksDir, "_index.md"), []byte(documentation.Index)); err != nil {
		return fmt.Errorf("writing benchmarks index: %w", err)
	}
	if err := writeBenchmarkAggregates(benchmarksDir, documentation.Aggregates); err != nil {
		return err
	}
	if err := writeBenchmarkAggregateDetails(benchmarksDir, documentation.AggregateDetails); err != nil {
		return err
	}
	if err := writeBenchmarkRuns(benchmarksDir, documentation.Runs); err != nil {
		return err
	}
	return writeBenchmarkShortcode(outputDir)
}

func writeBenchmarkAggregates(benchmarksDir, content string) error {
	aggregatesDir := filepath.Join(benchmarksDir, "aggregates")
	if err := os.MkdirAll(aggregatesDir, 0o750); err != nil {
		return fmt.Errorf("creating benchmark aggregate directory: %w", err)
	}
	if err := writeGeneratedFile(filepath.Join(aggregatesDir, "index.md"), []byte(content)); err != nil {
		return fmt.Errorf("writing benchmark aggregates: %w", err)
	}
	return nil
}

func writeBenchmarkAggregateDetails(benchmarksDir string, pages []benchmarkAggregatePage) error {
	tasksDir := filepath.Join(benchmarksDir, "aggregates", "tasks")
	for _, page := range pages {
		pageDir := filepath.Join(tasksDir, page.ID)
		if err := os.MkdirAll(pageDir, 0o750); err != nil {
			return fmt.Errorf("creating benchmark aggregate detail directory for %s: %w", page.Task, err)
		}
		if err := writeGeneratedFile(filepath.Join(pageDir, "index.md"), []byte(page.Content)); err != nil {
			return fmt.Errorf("writing benchmark aggregate detail for %s: %w", page.Task, err)
		}
	}
	return nil
}

func writeBenchmarkRuns(benchmarksDir string, runs []benchmarkDocumentationRun) error {
	runsDir := filepath.Join(benchmarksDir, "runs")
	if err := os.MkdirAll(runsDir, 0o750); err != nil {
		return fmt.Errorf("creating benchmark runs directory: %w", err)
	}
	if err := writeGeneratedFile(filepath.Join(runsDir, "_index.md"), []byte(`---
title: "Benchmark runs"
draft: false
weight: 22
---

The [empirical benchmark overview](/docs/benchmarks/) links individual benchmark observations.
`)); err != nil {
		return fmt.Errorf("writing benchmark runs index: %w", err)
	}
	for _, run := range runs {
		runDir := filepath.Join(runsDir, run.ID)
		if err := os.MkdirAll(runDir, 0o750); err != nil {
			return fmt.Errorf("creating benchmark run directory: %w", err)
		}
		if err := writeGeneratedFile(filepath.Join(runDir, "index.md"), []byte(renderBenchmarkRunDoc(run))); err != nil {
			return fmt.Errorf("writing benchmark run %s: %w", run.ID, err)
		}
	}
	return nil
}

func writeBenchmarkShortcode(outputDir string) error {
	shortcodesDir := filepath.Join(outputDir, "layouts", "shortcodes")
	if err := writeGeneratedFile(filepath.Join(shortcodesDir, "benchmark-code.html"), []byte(benchmarkCodeShortcode)); err != nil {
		return fmt.Errorf("writing benchmark shortcode: %w", err)
	}
	return nil
}

func findRepoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return os.Getwd()
}
