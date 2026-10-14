// Package main orchestrates benchmark report loading and page rendering.
package main

import (
	"encoding/json"
	"fmt"
	"path/filepath"
)

func renderBenchmarksDoc(rootDir string) (string, error) {
	documentation, err := renderBenchmarkDocumentation(rootDir)
	return documentation.Index, err
}

func renderBenchmarkDocumentation(rootDir string) (benchmarkDocumentation, error) {
	sources, err := loadBenchmarkSources(filepath.Join(rootDir, "data", "benchmarks", "results"))
	if err != nil {
		err = fmt.Errorf("load benchmark runs: %w", err)
	}
	runs := collectBenchmarkDocumentationRuns(rootDir, sources)
	rows := make([]map[string]json.RawMessage, 0)
	for _, source := range sources {
		rows = append(rows, source.Rows...)
	}
	comparisons := collectBenchmarkComparisons(runs)
	return benchmarkDocumentation{
		Index:            renderBestBenchmarksDoc(selectBestBenchmarkComparisons(runs), runs),
		Aggregates:       renderBenchmarkAggregatesDoc(comparisons),
		AggregateDetails: renderBenchmarkAggregateDetails(comparisons),
		Runs:             runs,
		BrowserRows:      rows,
	}, err
}
