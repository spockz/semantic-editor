// Package main builds browser-ready benchmark data and bundles the viewer assets.
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const benchmarkBrowserShortcode = `<link rel="stylesheet" href="/vendor/perspective/css/pro.css">
<link rel="stylesheet" href="/vendor/perspective/css/pro-dark.css">
<link rel="stylesheet" href="/vendor/perspective/css/perspective-viewer-datagrid.css">

<div class="benchmark-browser">
  <p>Results are grouped by target, context, and arm. Grouped cost, turns, elapsed time, and token counts use averages by default. Input tokens are uncached; cached input is shown separately. Use a column’s Edit control to choose average, minimum, or maximum.</p>
  <div class="benchmark-browser-filters">
    <label>Requirement met
      <select id="benchmark-browser-oracle-filter">
        <option value="">All</option>
        <option value="true">Pass</option>
        <option value="false">Fail</option>
      </select>
    </label>
    <label>Expected semantic tools used
      <select id="benchmark-browser-expected-tools-filter">
        <option value="">All</option>
        <option value="true">Pass</option>
        <option value="false">Fail</option>
      </select>
    </label>
    <label>Prompt variant
      <select id="benchmark-browser-prompt-filter">
        <option value="">All</option>
      </select>
    </label>
    <label>MCP instructions
      <select id="benchmark-browser-instructions-filter">
        <option value="">All</option>
      </select>
    </label>
  </div>
  <p id="benchmark-browser-status" role="status">Loading benchmark results…</p>
  <perspective-viewer id="benchmark-browser-viewer" theme="Pro Light" settings></perspective-viewer>
  <h2>Baseline vs semedit</h2>
  <p>Baseline and semedit values are compared within each target and context group. Negative changes are improvements for these lower-is-better metrics; positive changes are degradations.</p>
  <perspective-viewer id="benchmark-browser-comparison-viewer" theme="Pro Light" settings></perspective-viewer>
</div>

<style>
.benchmark-browser { width: 100%; }
.benchmark-browser-filters { display: flex; flex-wrap: wrap; gap: 1rem; margin: 1rem 0; }
.benchmark-browser-filters label { display: flex; align-items: center; gap: 0.5rem; }
.benchmark-browser perspective-viewer { display: block; width: 100%; height: min(75vh, 56rem); min-height: 34rem; }
.benchmark-browser #benchmark-browser-comparison-viewer { height: min(75vh, 56rem); min-height: 34rem; margin-top: 1rem; }
.benchmark-browser #benchmark-browser-status:empty { display: none; }
html.dark .benchmark-browser { color-scheme: dark; }
</style>

<script type="module">
import perspective from "/vendor/perspective/cdn/perspective.js";
import "/vendor/perspective/cdn/perspective-viewer.js";
import "/vendor/perspective/cdn/perspective-viewer-datagrid.js";

const viewer = document.querySelector("#benchmark-browser-viewer");
const comparisonViewer = document.querySelector("#benchmark-browser-comparison-viewer");
const viewers = [viewer, comparisonViewer];
const status = document.querySelector("#benchmark-browser-status");
const oracleFilter = document.querySelector("#benchmark-browser-oracle-filter");
const expectedToolsFilter = document.querySelector("#benchmark-browser-expected-tools-filter");
const promptFilter = document.querySelector("#benchmark-browser-prompt-filter");
const instructionsFilter = document.querySelector("#benchmark-browser-instructions-filter");
const syncTheme = () => {
  const dark = document.documentElement.classList.contains("dark");
  for (const currentViewer of viewers) currentViewer.setAttribute("theme", dark ? "Pro Dark" : "Pro Light");
};
const themeObserver = new MutationObserver(syncTheme);
themeObserver.observe(document.documentElement, { attributes: true, attributeFilter: ["class"] });
syncTheme();

try {
  await customElements.whenDefined("perspective-viewer");
  const response = await fetch("/data/benchmarks.json");
  if (!response.ok) throw new Error("Benchmark data request failed: " + response.status);
  const rows = await response.json();
  const populateDimensionFilter = (control, column) => {
    const values = [...new Set(rows.map(row => row[column]).filter(value => typeof value === "string" && value !== ""))].sort((a, b) => a.localeCompare(b));
    for (const value of values) {
      const option = document.createElement("option");
      option.value = value;
      option.textContent = column === "mcp_server_instructions" && value === "none" ? "None" : value;
      control.append(option);
    }
  };
  populateDimensionFilter(promptFilter, "prompt_variant");
  populateDimensionFilter(instructionsFilter, "mcp_server_instructions");
  const worker = await perspective.worker();
  const table = await worker.table(rows);
  await Promise.all(viewers.map(currentViewer => currentViewer.load(table)));
  await viewer.restore({
    plugin: "Datagrid",
    group_by: ["target__harness", "target__model", "target__effort", "context_variant", "arm"],
    columns: ["cost", "turns", "wall_clock_seconds", "input_tokens", "cached_input_tokens", "output_tokens", "reasoning_tokens"],
    aggregates: { cost: "avg", turns: "avg", wall_clock_seconds: "avg", input_tokens: "avg", cached_input_tokens: "avg", output_tokens: "avg", reasoning_tokens: "avg" },
  });
  await comparisonViewer.restore({
    plugin: "Datagrid",
    group_by: ["target__harness", "target__model", "target__effort", "context_variant"],
    columns: ["baseline_cost", "semedit_cost", "cost_delta", "baseline_turns", "semedit_turns", "turns_delta", "baseline_wall_clock_seconds", "semedit_wall_clock_seconds", "wall_clock_seconds_delta", "baseline_input_tokens", "semedit_input_tokens", "input_tokens_delta", "baseline_cached_input_tokens", "semedit_cached_input_tokens", "cached_input_tokens_delta", "baseline_output_tokens", "semedit_output_tokens", "output_tokens_delta", "baseline_reasoning_tokens", "semedit_reasoning_tokens", "reasoning_tokens_delta"],
    aggregates: { baseline_cost: "avg", semedit_cost: "avg", cost_delta: "avg", baseline_turns: "avg", semedit_turns: "avg", turns_delta: "avg", baseline_wall_clock_seconds: "avg", semedit_wall_clock_seconds: "avg", wall_clock_seconds_delta: "avg", baseline_input_tokens: "avg", semedit_input_tokens: "avg", input_tokens_delta: "avg", baseline_cached_input_tokens: "avg", semedit_cached_input_tokens: "avg", cached_input_tokens_delta: "avg", baseline_output_tokens: "avg", semedit_output_tokens: "avg", output_tokens_delta: "avg", baseline_reasoning_tokens: "avg", semedit_reasoning_tokens: "avg", reasoning_tokens_delta: "avg" },
    columns_config: {
      cost_delta: { number_bg_mode: "color", neg_bg_color: "#b7e4c7", pos_bg_color: "#f7b6b2" },
      turns_delta: { number_bg_mode: "color", neg_bg_color: "#b7e4c7", pos_bg_color: "#f7b6b2" },
      wall_clock_seconds_delta: { number_bg_mode: "color", neg_bg_color: "#b7e4c7", pos_bg_color: "#f7b6b2" },
      input_tokens_delta: { number_bg_mode: "color", neg_bg_color: "#b7e4c7", pos_bg_color: "#f7b6b2" },
      cached_input_tokens_delta: { number_bg_mode: "color", neg_bg_color: "#b7e4c7", pos_bg_color: "#f7b6b2" },
      output_tokens_delta: { number_bg_mode: "color", neg_bg_color: "#b7e4c7", pos_bg_color: "#f7b6b2" },
      reasoning_tokens_delta: { number_bg_mode: "color", neg_bg_color: "#b7e4c7", pos_bg_color: "#f7b6b2" },
    },
  });
  const updateFilters = async () => {
    const filters = [];
    for (const [control, column] of [[oracleFilter, "oracle_pass"], [expectedToolsFilter, "expected_semantic_tools_used"]]) {
      if (control.value !== "") filters.push([column, "==", control.value === "true"]);
    }
    for (const [control, column] of [[promptFilter, "prompt_variant"], [instructionsFilter, "mcp_server_instructions"]]) {
      if (control.value !== "") filters.push([column, "==", control.value]);
    }
    await Promise.all(viewers.map(currentViewer => currentViewer.restore({ filter: filters })));
  };
  oracleFilter.addEventListener("change", updateFilters);
  expectedToolsFilter.addEventListener("change", updateFilters);
  promptFilter.addEventListener("change", updateFilters);
  instructionsFilter.addEventListener("change", updateFilters);
  status.textContent = "";
} catch (error) {
  status.textContent = "Could not load benchmark results: " + error.message;
  status.setAttribute("role", "alert");
}
</script>
`

func writeBenchmarkBrowserAssets(rootDir, outputDir string) error {
	resultsDir := filepath.Join(rootDir, "data", "benchmarks", "results")
	var rows []map[string]json.RawMessage
	resultsRoot, err := os.OpenRoot(resultsDir)
	if os.IsNotExist(err) {
		err = nil
	} else if err != nil {
		return fmt.Errorf("open benchmark results directory: %w", err)
	}
	if resultsRoot != nil {
		err = filepath.WalkDir(resultsDir, func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() || filepath.Ext(path) != ".json" {
				return nil
			}
			relativePath, err := filepath.Rel(resultsDir, path)
			if err != nil {
				return fmt.Errorf("make benchmark result path relative: %w", err)
			}
			resultFile, err := resultsRoot.Open(relativePath)
			if err != nil {
				return fmt.Errorf("open benchmark result %s: %w", relativePath, err)
			}
			data, readErr := io.ReadAll(resultFile)
			closeErr := resultFile.Close()
			if readErr != nil {
				return fmt.Errorf("read benchmark result %s: %w", relativePath, readErr)
			}
			if closeErr != nil {
				return fmt.Errorf("close benchmark result %s: %w", relativePath, closeErr)
			}
			var report map[string]json.RawMessage
			if err := json.Unmarshal(data, &report); err != nil {
				return fmt.Errorf("decode benchmark result %s: %w", path, err)
			}
			var formatVersion int
			formatVersionValue := bytes.TrimSpace(report["format_version"])
			if len(formatVersionValue) > 0 && !bytes.Equal(formatVersionValue, []byte("null")) {
				if err := json.Unmarshal(formatVersionValue, &formatVersion); err != nil {
					return fmt.Errorf("decode benchmark format version in %s: %w", path, err)
				}
			}
			legacyDurations := formatVersion == 0
			runID, err := benchmarkRunID(resultsDir, path)
			if err != nil {
				return err
			}
			records := report["comparisons"]
			recordType := "comparison"
			if len(records) == 0 || string(records) == "null" {
				records = report["runs"]
				recordType = "run"
			}
			var entries []json.RawMessage
			if len(records) > 0 && string(records) != "null" {
				if err := json.Unmarshal(records, &entries); err != nil {
					return fmt.Errorf("decode benchmark %s records in %s: %w", recordType, path, err)
				}
			}
			for _, entry := range entries {
				var sourceRow map[string]json.RawMessage
				if err := json.Unmarshal(entry, &sourceRow); err != nil {
					return fmt.Errorf("decode benchmark %s in %s: %w", recordType, path, err)
				}
				if sourceRow == nil {
					return fmt.Errorf("decode benchmark %s in %s: expected a JSON object", recordType, path)
				}
				if recordType == "comparison" {
					comparisonRows, err := splitBenchmarkComparison(sourceRow, legacyDurations)
					if err != nil {
						return fmt.Errorf("split benchmark comparison in %s: %w", path, err)
					}
					for _, row := range comparisonRows {
						if err := addBenchmarkBrowserMetadata(row, runID, relativePath, "result"); err != nil {
							return err
						}
						rows = append(rows, row)
					}
					continue
				}
				row := make(map[string]json.RawMessage, len(sourceRow))
				for key, value := range sourceRow {
					if err := flattenBenchmarkField(key, value, row); err != nil {
						return fmt.Errorf("flatten benchmark field %s in %s: %w", key, path, err)
					}
				}
				if err := addBenchmarkBrowserSemanticToolPass(row); err != nil {
					return fmt.Errorf("derive semantic tool pass in %s: %w", path, err)
				}
				if err := addBenchmarkBrowserCost(entry, report["target"], row); err != nil {
					return fmt.Errorf("calculate benchmark cost in %s: %w", path, err)
				}
				if err := normalizeLegacyBenchmarkDurations(row, legacyDurations); err != nil {
					return fmt.Errorf("normalize benchmark durations in %s: %w", path, err)
				}
				if err := addBenchmarkBrowserWallClockSeconds(row); err != nil {
					return fmt.Errorf("derive benchmark wall-clock seconds in %s: %w", path, err)
				}
				if err := addBenchmarkBrowserMetadata(row, runID, relativePath, recordType); err != nil {
					return err
				}
				rows = append(rows, row)
			}
			return nil
		})
		closeErr := resultsRoot.Close()
		if err == nil {
			err = closeErr
		}
	}
	if err != nil {
		return fmt.Errorf("aggregate benchmark results: %w", err)
	}
	if rows == nil {
		rows = make([]map[string]json.RawMessage, 0)
	}
	if err := addBenchmarkBrowserComparisonDeltas(rows); err != nil {
		return fmt.Errorf("derive benchmark comparison changes: %w", err)
	}
	columns := make(map[string]struct{})
	for _, row := range rows {
		for column := range row {
			columns[column] = struct{}{}
		}
	}
	for _, row := range rows {
		for column := range columns {
			if _, ok := row[column]; !ok {
				row[column] = json.RawMessage("null")
			}
		}
	}
	encodedRows, err := json.Marshal(rows)
	if err != nil {
		return fmt.Errorf("encode benchmark browser data: %w", err)
	}
	if err := os.MkdirAll(filepath.Join(outputDir, "static", "data"), 0o750); err != nil {
		return fmt.Errorf("create benchmark browser data directory: %w", err)
	}
	dataPath := filepath.Join(outputDir, "static", "data", "benchmarks.json")
	if err := writeGeneratedFile(dataPath, encodedRows); err != nil {
		return fmt.Errorf("write benchmark browser data: %w", err)
	}

	vendorRoot := filepath.Join(rootDir, "cmd", "docgen", "assets", "vendor", "perspective")
	vendor, err := os.OpenRoot(vendorRoot)
	if err != nil {
		return fmt.Errorf("open Perspective vendor assets: %w", err)
	}
	err = filepath.WalkDir(vendorRoot, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relativePath, err := filepath.Rel(vendorRoot, path)
		if err != nil {
			return fmt.Errorf("make Perspective asset path relative: %w", err)
		}
		if entry.IsDir() {
			if relativePath == "." {
				return nil
			}
			if err := os.MkdirAll(filepath.Join(outputDir, "static", "vendor", "perspective", relativePath), 0o750); err != nil {
				return fmt.Errorf("create Perspective asset directory: %w", err)
			}
			return nil
		}
		if !entry.Type().IsRegular() {
			return nil
		}
		assetFile, err := vendor.Open(relativePath)
		if err != nil {
			return fmt.Errorf("open Perspective asset %s: %w", relativePath, err)
		}
		asset, readErr := io.ReadAll(assetFile)
		closeErr := assetFile.Close()
		if readErr != nil {
			return fmt.Errorf("read Perspective asset %s: %w", relativePath, readErr)
		}
		if closeErr != nil {
			return fmt.Errorf("close Perspective asset %s: %w", relativePath, closeErr)
		}
		targetPath := filepath.Join(outputDir, "static", "vendor", "perspective", relativePath)
		if err := writeGeneratedFile(targetPath, asset); err != nil {
			return fmt.Errorf("write Perspective asset %s: %w", relativePath, err)
		}
		return nil
	})
	closeErr := vendor.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		return fmt.Errorf("unpack Perspective vendor assets: %w", err)
	}

	shortcodeDir := filepath.Join(outputDir, "layouts", "shortcodes")
	if err := os.MkdirAll(shortcodeDir, 0o750); err != nil {
		return fmt.Errorf("create benchmark browser shortcode directory: %w", err)
	}
	shortcodePath := filepath.Join(shortcodeDir, "benchmark-browser.html")
	if err := writeGeneratedFile(shortcodePath, []byte(benchmarkBrowserShortcode)); err != nil {
		return fmt.Errorf("write benchmark browser shortcode: %w", err)
	}
	page := `---
title: "Benchmark browser"
description: "Explore benchmark observations across tasks, models, and experimental conditions."
draft: false
weight: 20
---

{{< benchmark-browser >}}
`
	pagePath := filepath.Join(outputDir, "content", "docs", "benchmarks", "browser", "index.md")
	if err := os.MkdirAll(filepath.Dir(pagePath), 0o750); err != nil {
		return fmt.Errorf("create benchmark browser page directory: %w", err)
	}
	if err := writeGeneratedFile(pagePath, []byte(page)); err != nil {
		return fmt.Errorf("write benchmark browser page: %w", err)
	}
	return nil
}

func addBenchmarkBrowserComparisonDeltas(rows []map[string]json.RawMessage) error {
	metrics := []string{"cost", "turns", "wall_clock_seconds", "input_tokens", "cached_input_tokens", "output_tokens", "reasoning_tokens"}
	targetDimensions := make(map[string]struct{})
	for _, row := range rows {
		for key := range row {
			if strings.HasPrefix(key, "target__") {
				targetDimensions[key] = struct{}{}
			}
		}
	}
	dimensions := make([]string, 0, 8+len(targetDimensions))
	dimensions = append(dimensions, "_run_id", "_source_file", "_record_type", "task_id", "context_variant", "variant", "prompt_variant", "mcp_server_instructions")
	for dimension := range targetDimensions {
		dimensions = append(dimensions, dimension)
	}
	sort.Strings(dimensions)
	type pair struct {
		baseline []int
		semedit  []int
	}
	pairs := make(map[string]*pair)
	for index := range rows {
		row := rows[index]
		for _, metric := range metrics {
			row[metric+"_delta"] = json.RawMessage("null")
			row["baseline_"+metric] = json.RawMessage("null")
			row["semedit_"+metric] = json.RawMessage("null")
		}
		var arm string
		if err := json.Unmarshal(row["arm"], &arm); err != nil || (arm != "baseline" && arm != "semedit") {
			continue
		}
		values := make([]json.RawMessage, 0, len(dimensions))
		for _, dimension := range dimensions {
			values = append(values, row[dimension])
		}
		encodedKey, err := json.Marshal(values)
		if err != nil {
			return fmt.Errorf("encode pair dimensions: %w", err)
		}
		key := string(encodedKey)
		if pairs[key] == nil {
			pairs[key] = &pair{}
		}
		if arm == "baseline" {
			pairs[key].baseline = append(pairs[key].baseline, index)
		} else {
			pairs[key].semedit = append(pairs[key].semedit, index)
		}
	}
	for _, matched := range pairs {
		if len(matched.baseline) != 1 || len(matched.semedit) != 1 {
			continue
		}
		baseline, semedit := rows[matched.baseline[0]], rows[matched.semedit[0]]
		for _, metric := range metrics {
			var before, after float64
			if !isBenchmarkNumber(baseline[metric], &before) || !isBenchmarkNumber(semedit[metric], &after) {
				continue
			}
			semedit["baseline_"+metric] = append(json.RawMessage(nil), baseline[metric]...)
			semedit["semedit_"+metric] = append(json.RawMessage(nil), semedit[metric]...)
			encoded, err := json.Marshal(after - before)
			if err != nil {
				return fmt.Errorf("encode %s change: %w", metric, err)
			}
			semedit[metric+"_delta"] = encoded
		}
	}
	return nil
}

func isBenchmarkNumber(raw json.RawMessage, value *float64) bool {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || bytes.Equal(raw, []byte("null")) {
		return false
	}
	return json.Unmarshal(raw, value) == nil
}

var benchmarkComparisonArms = []struct {
	field   string
	context string
	arm     string
}{
	{field: "small_baseline", context: "small", arm: "baseline"},
	{field: "small_semedit", context: "small", arm: "semedit"},
	{field: "small_verified_baseline", context: "small_verified", arm: "baseline"},
	{field: "small_verified_semedit", context: "small_verified", arm: "semedit"},
	{field: "large_baseline", context: "large", arm: "baseline"},
	{field: "large_semedit", context: "large", arm: "semedit"},
	{field: "large_verified_baseline", context: "large_verified", arm: "baseline"},
	{field: "large_verified_semedit", context: "large_verified", arm: "semedit"},
}

func splitBenchmarkComparison(comparison map[string]json.RawMessage, legacyDurations bool) ([]map[string]json.RawMessage, error) {
	base := make(map[string]json.RawMessage, len(comparison))
	for key, value := range comparison {
		if isBenchmarkComparisonArm(key) || isBenchmarkComparisonPrompt(key) {
			continue
		}
		if err := flattenBenchmarkField(key, value, base); err != nil {
			return nil, fmt.Errorf("flatten comparison field %s: %w", key, err)
		}
	}

	var rows []map[string]json.RawMessage
	for _, arm := range benchmarkComparisonArms {
		raw, ok := comparison[arm.field]
		if !ok || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
			continue
		}
		var run map[string]json.RawMessage
		if err := json.Unmarshal(raw, &run); err != nil {
			return nil, fmt.Errorf("decode %s result: %w", arm.field, err)
		}
		if run == nil {
			return nil, fmt.Errorf("decode %s result: expected a JSON object", arm.field)
		}
		row := make(map[string]json.RawMessage, len(base)+len(run)+2)
		maps.Copy(row, base)
		for key, value := range run {
			if isBenchmarkRunDetail(key) {
				continue
			}
			if err := flattenBenchmarkField(key, value, row); err != nil {
				return nil, fmt.Errorf("flatten %s result field %s: %w", arm.field, key, err)
			}
		}
		if err := addBenchmarkBrowserCost(raw, comparison["target"], row); err != nil {
			return nil, fmt.Errorf("calculate %s result cost: %w", arm.field, err)
		}
		for key, value := range map[string]string{"context_variant": arm.context, "arm": arm.arm} {
			encoded, err := json.Marshal(value)
			if err != nil {
				return nil, fmt.Errorf("encode benchmark %s: %w", key, err)
			}
			row[key] = encoded
		}
		if err := addBenchmarkBrowserSemanticToolPass(row); err != nil {
			return nil, fmt.Errorf("derive %s semantic tool pass: %w", arm.field, err)
		}
		if err := normalizeLegacyBenchmarkDurations(row, legacyDurations); err != nil {
			return nil, fmt.Errorf("normalize %s result durations: %w", arm.field, err)
		}
		if err := addBenchmarkBrowserWallClockSeconds(row); err != nil {
			return nil, fmt.Errorf("derive %s wall-clock seconds: %w", arm.field, err)
		}
		rows = append(rows, row)
	}
	return rows, nil
}

func addBenchmarkBrowserWallClockSeconds(row map[string]json.RawMessage) error {
	raw, ok := row["wall_clock_ms"]
	if !ok || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		row["wall_clock_seconds"] = json.RawMessage("null")
		return nil
	}
	var milliseconds float64
	if err := json.Unmarshal(raw, &milliseconds); err != nil {
		return fmt.Errorf("decode wall_clock_ms: %w", err)
	}
	seconds, err := json.Marshal(milliseconds / 1000)
	if err != nil {
		return fmt.Errorf("encode wall_clock_seconds: %w", err)
	}
	row["wall_clock_seconds"] = seconds
	return nil
}

func addBenchmarkBrowserSemanticToolPass(row map[string]json.RawMessage) error {
	var arm string
	armValue, ok := row["arm"]
	if !ok || bytes.Equal(bytes.TrimSpace(armValue), []byte("null")) {
		delete(row, "expected_semantic_tools_used")
		return nil
	}
	if err := json.Unmarshal(armValue, &arm); err != nil {
		return fmt.Errorf("decode arm: %w", err)
	}
	if arm != "semedit" {
		delete(row, "expected_semantic_tools_used")
		return nil
	}
	verified, ok := row["mcp_verified"]
	if !ok {
		row["expected_semantic_tools_used"] = json.RawMessage("null")
		return nil
	}
	row["expected_semantic_tools_used"] = append(json.RawMessage(nil), verified...)
	return nil
}

func addBenchmarkBrowserCost(runData, fallbackTarget json.RawMessage, row map[string]json.RawMessage) error {
	var run BenchRunResult
	if err := json.Unmarshal(runData, &run); err != nil {
		return fmt.Errorf("decode run for cost: %w", err)
	}
	if run.Target.Model == "" && len(fallbackTarget) > 0 {
		if err := json.Unmarshal(fallbackTarget, &run.Target); err != nil {
			return fmt.Errorf("decode target for cost: %w", err)
		}
	}
	var runFields map[string]json.RawMessage
	if err := json.Unmarshal(runData, &runFields); err != nil {
		return fmt.Errorf("decode token fields for cost: %w", err)
	}
	for _, key := range []string{"prompt_tokens", "uncached_prompt_tokens", "cached_prompt_tokens", "output_tokens", "reasoning_tokens"} {
		if _, ok := runFields[key]; ok {
			cost, available := benchmarkCost(&run)
			if !available {
				row["cost"] = json.RawMessage("null")
				return nil
			}
			encoded, err := json.Marshal(cost)
			if err != nil {
				return fmt.Errorf("encode cost: %w", err)
			}
			row["cost"] = encoded
			return nil
		}
	}
	row["cost"] = json.RawMessage("null")
	return nil
}

func normalizeLegacyBenchmarkDurations(row map[string]json.RawMessage, legacy bool) error {
	if !legacy {
		return nil
	}
	for _, key := range []string{
		"wall_clock_ms",
		"process_start_to_first_event_ms",
		"first_event_to_first_tool_call_ms",
		"mcp_initialize_to_first_semantic_call_ms",
		"mcp_server_start_to_initialize_ms",
		"oracle__duration_ms",
	} {
		raw, ok := row[key]
		if !ok || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
			continue
		}
		var nanoseconds int64
		if err := json.Unmarshal(raw, &nanoseconds); err != nil {
			return fmt.Errorf("decode %s: %w", key, err)
		}
		row[key] = json.RawMessage(fmt.Sprintf("%d", nanoseconds/int64(1e6)))
	}
	return nil
}

func isBenchmarkComparisonArm(name string) bool {
	for _, arm := range benchmarkComparisonArms {
		if name == arm.field {
			return true
		}
	}
	return false
}

func isBenchmarkComparisonPrompt(name string) bool {
	switch name {
	case "vanilla_prompt", "mcp_prompt", "vanilla_verified_prompt", "mcp_verified_prompt", "before_state":
		return true
	default:
		return false
	}
}

func isBenchmarkRunDetail(name string) bool {
	switch name {
	case "prompt", "before_state", "diff", "tools_used", "tool_calls", "interaction_steps", "semantic_tool_reflection":
		return true
	default:
		return false
	}
}

func addBenchmarkBrowserMetadata(row map[string]json.RawMessage, runID, relativePath, recordType string) error {
	for key, value := range map[string]string{
		"_run_id": runID, "_source_file": filepath.ToSlash(relativePath), "_record_type": recordType,
	} {
		encoded, err := json.Marshal(value)
		if err != nil {
			return fmt.Errorf("encode benchmark metadata %s: %w", key, err)
		}
		row[key] = encoded
	}
	return nil
}

func flattenBenchmarkField(name string, value json.RawMessage, row map[string]json.RawMessage) error {
	value = bytes.TrimSpace(value)
	if len(value) == 0 {
		return fmt.Errorf("empty JSON value")
	}
	if value[0] == '{' {
		var nested map[string]json.RawMessage
		if err := json.Unmarshal(value, &nested); err != nil {
			return err
		}
		if len(nested) == 0 {
			row[name] = json.RawMessage(`"{}"`)
			return nil
		}
		for key, child := range nested {
			if err := flattenBenchmarkField(name+"__"+key, child, row); err != nil {
				return err
			}
		}
		return nil
	}
	if value[0] == '[' {
		encoded, err := json.Marshal(string(value))
		if err != nil {
			return fmt.Errorf("encode array value: %w", err)
		}
		row[name] = encoded
		return nil
	}
	row[name] = value
	if name == "oracle__passed" {
		row["oracle_pass"] = value
	}
	if name == "uncached_prompt_tokens" {
		row["input_tokens"] = value
	}
	if name == "cached_prompt_tokens" {
		row["cached_input_tokens"] = value
	}
	return nil
}
