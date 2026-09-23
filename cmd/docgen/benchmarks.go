// Package main orchestrates benchmark report loading and page rendering.
package main

import "fmt"

func renderBenchmarksDoc(rootDir string) (string, error) {
	documentation, err := renderBenchmarkDocumentation(rootDir)
	if err != nil {
		return "", err
	}
	return documentation.Index, nil
}

func renderBenchmarkDocumentation(rootDir string) (benchmarkDocumentation, error) {
	runs, err := loadBenchmarkDocumentationRuns(rootDir)
	if err != nil {
		return benchmarkDocumentation{}, fmt.Errorf("load benchmark runs: %w", err)
	}

	comparisons := collectBenchmarkComparisons(runs)

	return benchmarkDocumentation{
		Index:      renderBestBenchmarksDoc(selectBestBenchmarkComparisons(runs), runs),
		Aggregates: renderBenchmarkAggregatesDoc(comparisons),
		Runs:       runs,
	}, nil
}
