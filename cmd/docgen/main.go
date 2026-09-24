// Package main synthesizes code-derived capability documentation and test-driven examples into a Hugo source tree.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	rootDir, err := findRepoRoot()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error locating repo root: %v\n", err)
		os.Exit(1)
	}

	outputDir := filepath.Join(rootDir, ".scratch", "docgen")
	flags := flag.NewFlagSet("docgen", flag.ExitOnError)
	flags.StringVar(&outputDir, "output-dir", outputDir, "directory for the generated Hugo source tree")
	if err := flags.Parse(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing flags: %v\n", err)
		os.Exit(1)
	}
	if err := os.MkdirAll(filepath.Join(outputDir, "content", "docs"), 0o750); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating documentation source directory: %v\n", err)
		os.Exit(1)
	}
	if err := os.MkdirAll(filepath.Join(outputDir, "data"), 0o750); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating Hugo data directory: %v\n", err)
		os.Exit(1)
	}
	shortcodesDir := filepath.Join(outputDir, "layouts", "shortcodes")
	if err := os.RemoveAll(shortcodesDir); err != nil {
		fmt.Fprintf(os.Stderr, "Error removing stale generated shortcodes: %v\n", err)
		os.Exit(1)
	}
	if err := os.MkdirAll(shortcodesDir, 0o750); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating Hugo shortcode directory: %v\n", err)
		os.Exit(1)
	}
	for _, stalePath := range []string{
		filepath.Join(outputDir, "content", "docs", "index.md"),
		filepath.Join(outputDir, "content", "docs", "getting-started"),
		filepath.Join(outputDir, "content", "docs", "reference"),
		filepath.Join(outputDir, "content", "docs", "benchmarks.md"),
		filepath.Join(outputDir, "content", "docs", "benchmarks"),
		filepath.Join(outputDir, "data", "landing.yaml"),
	} {
		if err := os.RemoveAll(stalePath); err != nil {
			fmt.Fprintf(os.Stderr, "Error removing stale documentation output: %v\n", err)
			os.Exit(1)
		}
	}

	// 1. Extract capabilities from code AST
	capabilities, placements, err := extractCodeCapabilities(rootDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error extracting code capabilities: %v\n", err)
		os.Exit(1)
	}

	// 2. Parse txtar test archives for real-world examples
	examples, err := extractTxtarExamples(rootDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error extracting txtar examples: %v\n", err)
		os.Exit(1)
	}

	// 3. Render Hugo source content
	markdownContent := renderMarkdown(capabilities, placements, examples)
	targetFile := filepath.Join(outputDir, "content", "docs", "reference.md")
	if err := writeGeneratedFile(targetFile, []byte(markdownContent)); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing Markdown output: %v\n", err)
		os.Exit(1)
	}
	gettingStartedFile := filepath.Join(outputDir, "content", "docs", "getting-started.md")
	if err := writeGeneratedFile(gettingStartedFile, []byte(renderGettingStarted())); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing getting started output: %v\n", err)
		os.Exit(1)
	}
	benchmarkDocumentation, err := renderBenchmarkDocumentation(rootDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error rendering benchmark documentation: %v\n", err)
		os.Exit(1)
	}
	if err := writeBenchmarkBrowserAssets(rootDir, outputDir); err != nil {
		fmt.Fprintf(os.Stderr, "Error generating benchmark browser assets: %v\n", err)
		os.Exit(1)
	}
	benchmarksDir := filepath.Join(outputDir, "content", "docs", "benchmarks")
	if err := os.MkdirAll(benchmarksDir, 0o750); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating benchmarks documentation directory: %v\n", err)
		os.Exit(1)
	}
	if err := writeGeneratedFile(filepath.Join(benchmarksDir, "_index.md"), []byte(benchmarkDocumentation.Index)); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing benchmarks index: %v\n", err)
		os.Exit(1)
	}
	aggregatesDir := filepath.Join(benchmarksDir, "aggregates")
	if err := os.MkdirAll(aggregatesDir, 0o750); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating benchmark aggregate directory: %v\n", err)
		os.Exit(1)
	}
	if err := writeGeneratedFile(filepath.Join(aggregatesDir, "index.md"), []byte(benchmarkDocumentation.Aggregates)); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing benchmark aggregates: %v\n", err)
		os.Exit(1)
	}
	runsDir := filepath.Join(benchmarksDir, "runs")
	if err := os.MkdirAll(runsDir, 0o750); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating benchmark runs directory: %v\n", err)
		os.Exit(1)
	}
	if err := writeGeneratedFile(filepath.Join(runsDir, "_index.md"), []byte(`---
title: "Benchmark runs"
draft: false
weight: 22
---

Individual benchmark observations are linked from the [empirical benchmark overview](/docs/benchmarks/).
`)); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing benchmark runs index: %v\n", err)
		os.Exit(1)
	}
	for _, run := range benchmarkDocumentation.Runs {
		runDir := filepath.Join(runsDir, run.ID)
		if err := os.MkdirAll(runDir, 0o750); err != nil {
			fmt.Fprintf(os.Stderr, "Error creating benchmark run directory: %v\n", err)
			os.Exit(1)
		}
		if err := writeGeneratedFile(filepath.Join(runDir, "index.md"), []byte(renderBenchmarkRunDoc(run))); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing benchmark run %s: %v\n", run.ID, err)
			os.Exit(1)
		}
	}
	if err := writeGeneratedFile(filepath.Join(shortcodesDir, "benchmark-code.html"), []byte(benchmarkCodeShortcode)); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing benchmark shortcode: %v\n", err)
		os.Exit(1)
	}
	if err := writeLandingAssets(outputDir); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing landing assets: %v\n", err)
		os.Exit(1)
	}
	if err := writeHugoConfig(outputDir); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing Hugo configuration: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Successfully synthesized Hugo documentation source: %s\n", targetFile)
	fmt.Printf("Extracted %d languages, %d placement modes, %d txtar workflows.\n", len(capabilities), len(placements), len(examples))
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
