// Package main loads benchmark snapshots and converts versioned report durations.
package main

import (
	"cmp"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

type benchmarkSource struct {
	RunID  string
	Report BenchReport
	Rows   []map[string]json.RawMessage
}

func loadAllBenchmarkComparisons(rootDir string) ([]*BenchComparisonSummary, error) {
	runs, err := loadBenchmarkDocumentationRuns(rootDir)
	return collectBenchmarkComparisons(runs), err
}

func collectBenchmarkComparisons(runs []benchmarkDocumentationRun) []*BenchComparisonSummary {
	var comparisons []*BenchComparisonSummary
	for _, run := range runs {
		comparisons = append(comparisons, run.Comparisons...)
	}
	sortBenchmarkComparisons(comparisons)
	return comparisons
}

func loadBenchmarkDocumentationRuns(rootDir string) ([]benchmarkDocumentationRun, error) {
	sources, err := loadBenchmarkSources(filepath.Join(rootDir, "data", "benchmarks", "results"))
	return collectBenchmarkDocumentationRuns(rootDir, sources), err
}

func unmarshalBenchmarkReport(data []byte, report *BenchReport) error {
	var header struct {
		FormatVersion int    `json:"format_version"`
		DurationUnit  string `json:"duration_unit"`
	}
	if err := json.Unmarshal(data, &header); err != nil {
		return err
	}
	if header.FormatVersion == 0 {
		return json.Unmarshal(data, report)
	}
	if header.FormatVersion != 2 || header.DurationUnit != "milliseconds" {
		return fmt.Errorf("unsupported benchmark report format version %d with duration unit %q", header.FormatVersion, header.DurationUnit)
	}

	var value any
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	if err := convertBenchmarkDurationMillisecondsToNanoseconds(value); err != nil {
		return err
	}
	normalized, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return json.Unmarshal(normalized, report)
}

func convertBenchmarkDurationMillisecondsToNanoseconds(value any) error {
	switch value := value.(type) {
	case []any:
		for _, item := range value {
			if err := convertBenchmarkDurationMillisecondsToNanoseconds(item); err != nil {
				return err
			}
		}
	case map[string]any:
		return convertBenchmarkDurationMap(value)
	}
	return nil
}

func convertBenchmarkDurationMap(value map[string]any) error {
	for _, item := range value {
		if err := convertBenchmarkDurationMillisecondsToNanoseconds(item); err != nil {
			return err
		}
	}
	for _, key := range benchmarkDurationKeys {
		if err := convertBenchmarkDurationField(value, key); err != nil {
			return err
		}
	}
	if _, isOracle := value["level_1_policy"]; isOracle {
		if err := convertBenchmarkDurationField(value, "duration_ms"); err != nil {
			return fmt.Errorf("convert oracle %w", err)
		}
	}
	return nil
}

var benchmarkDurationKeys = []string{
	"wall_clock_ms",
	"process_start_to_first_event_ms",
	"first_event_to_first_tool_call_ms",
	"mcp_initialize_to_first_semantic_call_ms",
	"mcp_server_start_to_initialize_ms",
}

func convertBenchmarkDurationField(value map[string]any, key string) error {
	raw, ok := value[key]
	if !ok {
		return nil
	}
	nanoseconds, err := benchmarkMillisecondsToNanoseconds(raw)
	if err != nil {
		return fmt.Errorf("convert %s: %w", key, err)
	}
	value[key] = nanoseconds
	return nil
}

func benchmarkMillisecondsToNanoseconds(raw any) (int64, error) {
	milliseconds, ok := raw.(float64)
	if !ok || milliseconds != float64(int64(milliseconds)) {
		return 0, fmt.Errorf("expected integral duration, got %T (%v)", raw, raw)
	}
	return int64(milliseconds) * int64(time.Millisecond), nil
}

func benchmarkRunID(resultsDir, resultPath string) (string, error) {
	relPath, err := filepath.Rel(resultsDir, resultPath)
	if err != nil {
		return "", fmt.Errorf("determine benchmark result path %s: %w", resultPath, err)
	}
	parts := strings.Split(filepath.ToSlash(relPath), "/")
	if len(parts) == 1 {
		return "legacy", nil
	}
	if parts[0] == "" || parts[0] == "." || parts[0] == ".." {
		return "", fmt.Errorf("invalid benchmark run identifier in %s", resultPath)
	}
	return parts[0], nil
}

func loadBenchmarkSources(resultsDir string) ([]benchmarkSource, error) {
	resultsRoot, err := os.OpenRoot(resultsDir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("open benchmark results directory: %w", err)
	}
	var sources []benchmarkSource
	var failures []error
	walkErr := filepath.WalkDir(resultsDir, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			failures = append(failures, fmt.Errorf("walk benchmark path %s: %w", path, walkErr))
			return nil
		}
		if entry.IsDir() && (entry.Name() == "jobs" || entry.Name() == "evidence") {
			return filepath.SkipDir
		}
		if entry.IsDir() || filepath.Ext(path) != ".json" || entry.Name() == "manifest.json" || entry.Name() == "smoke-summary.json" {
			return nil
		}
		source, loadErr := loadBenchmarkSource(resultsDir, path, resultsRoot)
		if loadErr != nil {
			failures = append(failures, loadErr)
		} else {
			sources = append(sources, source)
		}
		return nil
	})
	if walkErr != nil {
		failures = append(failures, fmt.Errorf("walk benchmark results: %w", walkErr))
	}
	if closeErr := resultsRoot.Close(); closeErr != nil {
		failures = append(failures, fmt.Errorf("close benchmark results directory: %w", closeErr))
	}
	return sources, errors.Join(failures...)
}

func collectBenchmarkDocumentationRuns(rootDir string, sources []benchmarkSource) []benchmarkDocumentationRun {
	runsByID := make(map[string]*benchmarkDocumentationRun)
	for _, source := range sources {
		if len(source.Rows) == 0 {
			continue
		}
		run := runsByID[source.RunID]
		if run == nil {
			run = &benchmarkDocumentationRun{ID: source.RunID}
			runsByID[source.RunID] = run
		}
		for _, comp := range source.Report.Comparisons {
			comp = filterPublishableBenchmarkComparison(comp)
			if comp == nil {
				continue
			}
			if comp.TxtarPath != "" {
				comp.TxtarProvenance = resolveTxtarProvenance(rootDir, comp.TxtarPath)
			}
			run.Comparisons = append(run.Comparisons, comp)
		}
	}
	runs := make([]benchmarkDocumentationRun, 0, len(runsByID))
	for _, run := range runsByID {
		sortBenchmarkComparisons(run.Comparisons)
		runs = append(runs, *run)
	}
	slices.SortFunc(runs, func(a, b benchmarkDocumentationRun) int { return cmp.Compare(a.ID, b.ID) })
	return runs
}

func loadBenchmarkSource(resultsDir, path string, resultsRoot *os.Root) (benchmarkSource, error) {
	relativePath, err := filepath.Rel(resultsDir, path)
	if err != nil {
		return benchmarkSource{}, fmt.Errorf("make benchmark result path relative %s: %w", path, err)
	}
	data, err := resultsRoot.ReadFile(relativePath)
	if err != nil {
		return benchmarkSource{}, fmt.Errorf("read benchmark result %s: %w", path, err)
	}
	var source benchmarkSource
	if decodeErr := unmarshalBenchmarkReport(data, &source.Report); decodeErr != nil {
		return benchmarkSource{}, fmt.Errorf("decode benchmark report %s: %w", path, decodeErr)
	}
	source.RunID, err = benchmarkRunID(resultsDir, path)
	if err != nil {
		return benchmarkSource{}, err
	}
	source.Rows, err = loadBenchmarkBrowserResult(resultsDir, path, relativePath, data)
	if err != nil {
		return benchmarkSource{}, fmt.Errorf("load benchmark browser report: %w", err)
	}
	return source, nil
}
