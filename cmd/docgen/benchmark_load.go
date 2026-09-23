// Package main loads benchmark snapshots and converts versioned report durations.
package main

import (
	"cmp"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

func loadAllBenchmarkComparisons(rootDir string) ([]*BenchComparisonSummary, error) {
	runs, err := loadBenchmarkDocumentationRuns(rootDir)
	if err != nil {
		return nil, err
	}
	return collectBenchmarkComparisons(runs), nil
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
	resultsDir := filepath.Join(rootDir, "data", "benchmarks", "results")
	if _, err := os.Stat(resultsDir); os.IsNotExist(err) {
		return nil, nil
	} else if err != nil {
		return nil, fmt.Errorf("stat benchmark results directory: %w", err)
	}

	runsByID := make(map[string]*benchmarkDocumentationRun)
	err := filepath.WalkDir(resultsDir, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() || filepath.Ext(path) != ".json" {
			return nil
		}

		// #nosec G304,G122 -- reading benchmark result JSON
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read benchmark file %s: %w", path, err)
		}

		var report BenchReport
		if err := unmarshalBenchmarkReport(data, &report); err != nil {
			return fmt.Errorf("unmarshal benchmark json %s: %w", path, err)
		}
		runID, err := benchmarkRunID(resultsDir, path)
		if err != nil {
			return err
		}
		run := runsByID[runID]
		if run == nil {
			run = &benchmarkDocumentationRun{ID: runID}
			runsByID[runID] = run
		}

		for _, comp := range report.Comparisons {
			comp = filterPublishableBenchmarkComparison(comp)
			if comp == nil {
				continue
			}
			if comp.TxtarPath != "" {
				comp.TxtarProvenance = resolveTxtarProvenance(rootDir, comp.TxtarPath)
			}
			run.Comparisons = append(run.Comparisons, comp)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	runs := make([]benchmarkDocumentationRun, 0, len(runsByID))
	for _, run := range runsByID {
		sortBenchmarkComparisons(run.Comparisons)
		runs = append(runs, *run)
	}
	slices.SortFunc(runs, func(a, b benchmarkDocumentationRun) int {
		return cmp.Compare(a.ID, b.ID)
	})
	return runs, nil
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
		for _, item := range value {
			if err := convertBenchmarkDurationMillisecondsToNanoseconds(item); err != nil {
				return err
			}
		}
		for _, key := range []string{
			"wall_clock_ms",
			"process_start_to_first_event_ms",
			"first_event_to_first_tool_call_ms",
			"mcp_initialize_to_first_semantic_call_ms",
			"mcp_server_start_to_initialize_ms",
		} {
			if raw, ok := value[key]; ok {
				nanoseconds, err := benchmarkMillisecondsToNanoseconds(raw)
				if err != nil {
					return fmt.Errorf("convert %s: %w", key, err)
				}
				value[key] = nanoseconds
			}
		}
		if _, isOracle := value["level_1_policy"]; isOracle {
			if raw, ok := value["duration_ms"]; ok {
				nanoseconds, err := benchmarkMillisecondsToNanoseconds(raw)
				if err != nil {
					return fmt.Errorf("convert oracle duration_ms: %w", err)
				}
				value["duration_ms"] = nanoseconds
			}
		}
	}
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
