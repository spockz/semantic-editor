// Package main provides the standalone benchmarking harness for semedit.
package main

import (
	"cmp"
	"context"
	"flag"
	"fmt"
	"io"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"text/tabwriter"
	"time"
)

func main() {
	os.Exit(run())
}

func run() int {
	var taskID string
	var tasksFlag string
	var benchDir string
	var arm string
	var harness string
	var variantsFlag string
	var matrixMode bool
	var targets TargetList
	var outJSON string
	var outMD string
	var outDir string
	var runID string
	var extractTo string
	var evalDir string
	var timeout time.Duration
	var concurrency int
	var repeats int
	var listMode bool
	var listOpenRouterFree bool
	var openRouterFreeTop10 bool
	var mcpServerInstructionsRaw string
	var provenance ProvenanceSet

	flag.StringVar(&taskID, "task", "", "Specific benchmark task ID to run (e.g. 'task-01-rename-local', empty for all)")
	flag.StringVar(&tasksFlag, "tasks", "", "Comma-separated list of task base names to run in matrix mode")
	flag.StringVar(&benchDir, "dir", "testdata/bench", "Path to benchmark fixtures directory containing txtar archives")
	flag.StringVar(&arm, "arm", "control", "Evaluation arm to execute (control, semedit, baseline-diff)")
	flag.StringVar(&harness, "harness", "control", "Agent harness to drive (control, codex, agy, opencode)")
	flag.Var(&targets, "target", "Execution target harness[/model[/effort]] (repeatable, e.g. -target codex/gpt-5.6-luna/high)")
	flag.IntVar(&repeats, "repeats", 1, "Number of independent trials for each selected target/task/variant/arm")
	flag.StringVar(&variantsFlag, "variants", "small,large", "Comma-separated context variants to run in matrix mode (small, large)")
	flag.BoolVar(&matrixMode, "matrix", false, "Execute full combinatorial matrix across targets, tasks, variants, and arms")
	flag.IntVar(&concurrency, "concurrency", 4, "Number of concurrent matrix benchmark workers")
	flag.StringVar(&outJSON, "out-json", "", "Optional path to save telemetry stats JSON")
	flag.StringVar(&outMD, "out-md", "", "Optional path to save rendered Markdown report")
	flag.StringVar(&outDir, "out-dir", "data/benchmarks/results", "Parent directory for run-scoped benchmark results JSON and MD")
	flag.StringVar(&runID, "run-id", "", "Required identifier for a matrix result directory below -out-dir")
	flag.StringVar(&extractTo, "extract-to", "", "Extract fixture to target directory and exit (for agent eval trials)")
	flag.StringVar(&evalDir, "eval-dir", "", "Evaluate target directory with task oracle (for agent eval trials)")
	flag.DurationVar(&timeout, "timeout", 5*time.Minute, "Timeout per benchmark task")
	flag.StringVar(&mcpServerInstructionsRaw, "mcp-server-instructions", "none", "Server-wide semedit MCP instruction mode (none, descriptive, prescriptive; Codex and OpenCode)")
	flag.Var(&provenance, "provenance", "Technical execution provenance key=value (repeatable; does not group results)")
	flag.Var(&provenance, "classifier", "Deprecated alias for -provenance")
	flag.BoolVar(&listMode, "list", false, "List all available benchmark tasks and their prompt variants")
	flag.BoolVar(&listOpenRouterFree, "list-openrouter-free", false, "List the pinned top coding-oriented OpenRouter free-model catalog")
	flag.BoolVar(&openRouterFreeTop10, "openrouter-free-top10", false, "Add the pinned top 10 coding-oriented OpenRouter free models to the matrix")
	flag.Parse()

	if listOpenRouterFree {
		return listOpenRouterFreeModels()
	}
	if listMode || (len(flag.Args()) > 0 && flag.Args()[0] == "list") {
		return listBenchmarks(benchDir)
	}

	if extractTo != "" {
		if taskID == "" {
			fmt.Fprintln(os.Stderr, "-task is required when using -extract-to")
			return 1
		}
		fixtureFile := filepath.Join(benchDir, taskID+".txtar")
		if _, err := os.Stat(fixtureFile); err != nil {
			fixtureFile = filepath.Join(benchDir, strings.ReplaceAll(taskID, "-", "_")+".txtar")
		}
		if err := extractFixture(fixtureFile, extractTo); err != nil {
			fmt.Fprintf(os.Stderr, "extract error: %v\n", err)
			return 1
		}
		return 0
	}

	if evalDir != "" {
		if taskID == "" {
			fmt.Fprintln(os.Stderr, "-task is required when using -eval-dir")
			return 1
		}
		fixtureFile := filepath.Join(benchDir, taskID+".txtar")
		if _, err := os.Stat(fixtureFile); err != nil {
			fixtureFile = filepath.Join(benchDir, strings.ReplaceAll(taskID, "-", "_")+".txtar")
		}
		if err := evaluateDir(fixtureFile, evalDir, []string{"api/server.go"}); err != nil {
			fmt.Fprintf(os.Stderr, "evaluate error: %v\n", err)
			return 1
		}
		return 0
	}

	mcpServerInstructions, err := ParseMCPServerInstructionMode(mcpServerInstructionsRaw)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid -mcp-server-instructions: %v\n", err)
		return 1
	}

	scratchDir := filepath.Join(".scratch", "benchmarks")
	runner := NewRunner(scratchDir, WithMCPServerInstructions(mcpServerInstructions), WithProvenance(provenance))

	if openRouterFreeTop10 {
		targets = append(targets, Target{Harness: string(HarnessOpenCode), Model: "openrouter/free-top10"})
	}
	if matrixMode || len(targets) > 0 {
		return runMatrix(runner, benchDir, targets, tasksFlag, taskID, variantsFlag, outDir, runID, outJSON, outMD, timeout, concurrency, repeats)
	}

	// Legacy single-run path
	return runSingle(runner, benchDir, taskID, harness, arm, outJSON, outMD, timeout)
}

func runMatrix(runner *Runner, benchDir string, targets []Target, tasksFlag, singleTask, variantsFlag, outDir, runID, outJSON, outMD string, timeout time.Duration, concurrency, repeats int) int {
	if repeats < 1 {
		fmt.Fprintln(os.Stderr, "-repeats must be at least 1")
		return 1
	}
	if len(targets) == 0 {
		targets = []Target{{Harness: "codex"}, {Harness: "agy"}}
	}
	targets = expandOpenRouterFreeTargets(targets)

	var taskBases []string
	switch {
	case tasksFlag == "all" || tasksFlag == "*":
		entries, err := os.ReadDir(benchDir)
		if err == nil {
			for _, e := range entries {
				if strings.HasSuffix(e.Name(), ".txtar") && !strings.Contains(e.Name(), "large_context") {
					taskBases = append(taskBases, strings.TrimSuffix(e.Name(), ".txtar"))
				}
			}
		}
		taskBases = append(taskBases, "generate_template_main")
		slices.Sort(taskBases)
		taskBases = slices.Compact(taskBases)
	case tasksFlag != "":
		for t := range strings.SplitSeq(tasksFlag, ",") {
			t = strings.TrimSpace(t)
			if t != "" {
				taskBases = append(taskBases, t)
			}
		}
	case singleTask != "":
		taskBases = []string{singleTask}
	default:
		taskBases = []string{"task-01-rename-local"}
	}

	var variants []string
	for v := range strings.SplitSeq(variantsFlag, ",") {
		v = strings.TrimSpace(strings.ToLower(v))
		if v != "" {
			variants = append(variants, v)
		}
	}

	arms := []ArmType{ArmBaseline, ArmSemedit}

	if concurrency < 1 {
		concurrency = 1
	}

	resultDir := ""
	if outDir != "" {
		var err error
		resultDir, err = createBenchmarkRunDir(outDir, runID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "invalid benchmark result directory: %v\n", err)
			return 1
		}
	}

	fmt.Printf("==> Starting Matrix Benchmark (Concurrency: %d)\n", concurrency)
	fmt.Printf("    Targets:  %v\n", targets)
	fmt.Printf("    Tasks:    %v\n", taskBases)
	fmt.Printf("    Variants: %v\n", variants)
	fmt.Printf("    Arms:     %v\n\n", arms)
	fmt.Printf("    Repeats:  %d\n", repeats)
	fmt.Printf("    MCP server instructions: %s\n", runner.mcpServerInstructionsMode)
	if resultDir != "" {
		fmt.Printf("    Result run: %s\n", resultDir)
	}
	if runner.provenance.String() != "" {
		fmt.Printf("    Provenance: %s\n", runner.provenance)
	}
	fmt.Println()

	jobs := buildMatrixJobs(benchDir, taskBases, targets, variants, arms, repeats)
	allRuns := executeMatrixJobs(runner, jobs, timeout, concurrency)

	report := &BenchmarkReport{
		Timestamp:   time.Now(),
		Runs:        allRuns,
		Comparisons: BuildComparisons(allRuns),
	}

	if resultDir != "" {
		saveMatrixReports(resultDir, benchDir, allRuns)
	}

	if outJSON != "" || outMD != "" {
		if err := SaveReport(report, outJSON, outMD); err != nil {
			fmt.Printf("❌ Failed to save reports: %v\n", err)
			return 1
		}
		if outMD != "" {
			fmt.Printf("\n==> Rendered Markdown Comparison Table:\n\n%s\n", report.RenderMarkdown())
		}
	}

	return 0
}

func createBenchmarkRunDir(outDir, runID string) (string, error) {
	if err := validateBenchmarkRunID(runID); err != nil {
		return "", err
	}
	if err := os.MkdirAll(outDir, 0o750); err != nil {
		return "", fmt.Errorf("create results parent directory: %w", err)
	}
	runDir := filepath.Join(outDir, runID)
	if err := os.Mkdir(runDir, 0o750); err != nil {
		if os.IsExist(err) {
			return "", fmt.Errorf("run id %q already exists", runID)
		}
		return "", fmt.Errorf("create run directory: %w", err)
	}
	return runDir, nil
}

func expandOpenRouterFreeTargets(targets []Target) []Target {
	expanded := make([]Target, 0, len(targets))
	for _, target := range targets {
		if target.Harness == string(HarnessOpenCode) && target.Model == "openrouter/free-top10" {
			for _, catalogTarget := range openRouterTopTargets() {
				catalogTarget.Effort = target.Effort
				expanded = append(expanded, catalogTarget)
			}
			continue
		}
		expanded = append(expanded, target)
	}
	return expanded
}

func listOpenRouterFreeModels() int {
	fmt.Printf("OpenRouter free coding-oriented catalog (as of %s)\n", openRouterFreeCatalogAsOf)
	fmt.Printf("Source: %s\n", openRouterFreeCatalogURL)
	fmt.Println("Rank\tModel ID\tProgramming rank\tTool calling\tOpenCode target")
	for _, model := range openRouterFreeCatalog {
		programmingRank := "-"
		if model.ProgrammingRank > 0 {
			programmingRank = fmt.Sprintf("#%d", model.ProgrammingRank)
		}
		fmt.Printf("%d\t%s\t%s\t%t\topencode/openrouter/%s\n", model.Rank, model.ID, programmingRank, model.SupportsToolCall, model.ID)
	}
	return 0
}

func validateBenchmarkRunID(runID string) error {
	if runID == "" {
		return fmt.Errorf("-run-id is required when -out-dir is set")
	}
	if runID == "." || runID == ".." || strings.ContainsAny(runID, "/\\") {
		return fmt.Errorf("invalid run id %q", runID)
	}
	for _, char := range runID {
		if (char < 'a' || char > 'z') && (char < 'A' || char > 'Z') && (char < '0' || char > '9') && char != '-' && char != '_' && char != '.' {
			return fmt.Errorf("invalid run id %q", runID)
		}
	}
	return nil
}

func runSingle(runner *Runner, benchDir, taskSelector, harnessStr, armStr, outJSON, outMD string, timeout time.Duration) int {
	entries, err := os.ReadDir(benchDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading bench dir: %v\n", err)
		return 1
	}

	harness := HarnessType(strings.ToLower(harnessStr))
	arm := ArmType(strings.ToLower(armStr))
	target := Target{Harness: string(harness)}

	var reportRuns []*RunResult

	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".txtar") {
			continue
		}

		fixturePath := filepath.Join(benchDir, entry.Name())
		// #nosec G304 -- reading benchmark txtar fixture files
		data, err := os.ReadFile(filepath.Clean(fixturePath))
		if err != nil {
			fmt.Printf("read fixture error: %v\n", err)
			return 1
		}

		task, err := ParseTask(data)
		if err != nil {
			fmt.Printf("parse fixture error: %v\n", err)
			return 1
		}

		if taskSelector != "" && task.Metadata.TaskID != taskSelector && !strings.Contains(task.Metadata.TaskID, taskSelector) {
			continue
		}

		variant := "small"
		if strings.Contains(task.Metadata.TaskID, "large") {
			variant = "large"
		}
		variants := []string{variant}
		if harness != HarnessControl && arm != ArmControl {
			variants = matrixPromptVariants(task, variant)
			if len(variants) == 0 {
				fmt.Printf("Skipping benchmark %s: no non-empty prompt_variants declared in its txtar fixture\n", task.Metadata.TaskID)
				continue
			}
		}

		for _, variant := range variants {
			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			var res *RunResult
			if harness == HarnessControl || arm == ArmControl {
				res, err = runner.ExecuteControl(ctx, task)
			} else {
				res, err = runner.ExecuteAgentDriver(ctx, task, target, arm, variant)
			}
			cancel()

			if err != nil {
				fmt.Printf("❌ Task %s failed: %v\n", task.Metadata.TaskID, err)
				continue
			}
			reportRuns = append(reportRuns, res)
		}
	}

	if outJSON != "" || outMD != "" {
		report := &BenchmarkReport{
			Timestamp: time.Now(),
			Runs:      reportRuns,
		}
		if err := SaveReport(report, outJSON, outMD); err != nil {
			fmt.Printf("error saving report: %v\n", err)
			return 1
		}
	}

	return 0
}

func resolveFixturePath(benchDir, taskBase, _ string) string {
	cleanBase := strings.ReplaceAll(taskBase, "-", "_")

	strippedBase := cleanBase
	if strings.HasPrefix(cleanBase, "task_") {
		if idx := strings.Index(cleanBase[5:], "_"); idx >= 0 {
			strippedBase = cleanBase[5+idx+1:]
		}
	}

	bases := []string{cleanBase}
	if strippedBase != cleanBase {
		bases = append(bases, strippedBase)
	}

	for _, b := range bases {
		candidates := []string{
			filepath.Join(benchDir, b+".txtar"),
			filepath.Join(benchDir, strings.ReplaceAll(b, "_", "-")+".txtar"),
			filepath.Join("testdata", "scripts", b+".txtar"),
			filepath.Join("testdata", "scripts", strings.ReplaceAll(b, "_", "-")+".txtar"),
			filepath.Join("testdata", "bench", b+".txtar"),
			filepath.Join("testdata", "bench", strings.ReplaceAll(b, "_", "-")+".txtar"),
		}

		for _, cand := range candidates {
			if _, err := os.Stat(cand); err == nil {
				return cand
			}
		}
	}

	return filepath.Join(benchDir, taskBase+".txtar")
}

func declaredPromptVariants(task *Task) []string {
	if task == nil {
		return nil
	}

	variants := make([]string, 0, len(task.Metadata.PromptVariants))
	for name, prompt := range task.Metadata.PromptVariants {
		if strings.TrimSpace(name) != "" && strings.TrimSpace(prompt) != "" {
			variants = append(variants, name)
		}
	}
	slices.Sort(variants)
	return variants
}

func matrixPromptVariants(task *Task, contextVariant string) []string {
	declared := declaredPromptVariants(task)
	if len(declared) == 0 {
		return nil
	}

	if _, promptVariant, explicit := strings.Cut(contextVariant, ":"); explicit {
		if slices.Contains(declared, promptVariant) {
			return []string{contextVariant}
		}
		return nil
	}

	variants := make([]string, 0, len(declared))
	for _, promptVariant := range declared {
		variants = append(variants, contextVariant+":"+promptVariant)
	}
	return variants
}

// BenchmarkInfo summarizes an available benchmark task fixture.
type BenchmarkInfo struct {
	TaskID         string   `json:"task_id"`
	Category       string   `json:"category"`
	Instruction    string   `json:"instruction"`
	PromptVariants []string `json:"prompt_variants,omitempty"`
	FixturePath    string   `json:"fixture_path"`
}

// CollectAvailableBenchmarks scans fixture directories and returns sorted benchmark metadata.
func CollectAvailableBenchmarks(benchDir string) ([]BenchmarkInfo, error) {
	var benchmarks []BenchmarkInfo
	seen := make(map[string]bool)

	dirs := []string{benchDir}
	scriptsDir := filepath.Join(filepath.Dir(benchDir), "scripts")
	if scriptsDir != benchDir {
		dirs = append(dirs, scriptsDir)
	}
	for i, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			if i > 0 && os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("read benchmark directory %s: %w", dir, err)
		}
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".txtar") {
				continue
			}
			fixturePath := filepath.Join(dir, entry.Name())
			// #nosec G304 -- reading benchmark fixture for listing
			data, err := os.ReadFile(filepath.Clean(fixturePath))
			if err != nil {
				return nil, fmt.Errorf("read benchmark fixture %s: %w", fixturePath, err)
			}
			task, err := ParseTask(data)
			if err != nil {
				if i > 0 {
					continue // The optional scripts directory contains non-benchmark CLI archives.
				}
				return nil, fmt.Errorf("parse benchmark fixture %s: %w", fixturePath, err)
			}
			if seen[task.Metadata.TaskID] {
				continue
			}
			seen[task.Metadata.TaskID] = true

			promptVars := slices.Sorted(maps.Keys(task.Metadata.PromptVariants))

			benchmarks = append(benchmarks, BenchmarkInfo{
				TaskID:         task.Metadata.TaskID,
				Category:       task.Metadata.Category,
				Instruction:    task.Metadata.Instruction,
				PromptVariants: promptVars,
				FixturePath:    filepath.ToSlash(fixturePath),
			})
		}
	}

	slices.SortFunc(benchmarks, func(a, b BenchmarkInfo) int {
		return cmp.Compare(a.TaskID, b.TaskID)
	})

	return benchmarks, nil
}

// PrintBenchmarksList outputs a formatted table of available benchmarks to the given writer.
func PrintBenchmarksList(out io.Writer, benchmarks []BenchmarkInfo) {
	_, _ = fmt.Fprintf(out, "Available Benchmarks (%d tasks):\n\n", len(benchmarks))
	w := tabwriter.NewWriter(out, 0, 0, 3, ' ', 0)
	_, _ = fmt.Fprintln(w, "TASK ID\tCATEGORY\tPROMPT VARIANTS\tFIXTURE PATH")
	for _, b := range benchmarks {
		pv := "default"
		if len(b.PromptVariants) > 0 {
			pv = strings.Join(b.PromptVariants, ", ")
		}
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", b.TaskID, b.Category, pv, b.FixturePath)
	}
	_ = w.Flush()
}

func listBenchmarks(benchDir string) int {
	benchmarks, err := CollectAvailableBenchmarks(benchDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error listing benchmarks: %v\n", err)
		return 1
	}
	PrintBenchmarksList(os.Stdout, benchmarks)
	return 0
}
