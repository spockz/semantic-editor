// Package main provides the standalone benchmarking harness for semedit.
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"sync"
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
	var extractTo string
	var evalDir string
	var timeout time.Duration
	var concurrency int
	var listMode bool
	var mcpServerInstructionsRaw string
	var provenance ProvenanceSet

	flag.StringVar(&taskID, "task", "", "Specific benchmark task ID to run (e.g. 'task-01-rename-local', empty for all)")
	flag.StringVar(&tasksFlag, "tasks", "", "Comma-separated list of task base names to run in matrix mode")
	flag.StringVar(&benchDir, "dir", "testdata/bench", "Path to benchmark fixtures directory containing txtar archives")
	flag.StringVar(&arm, "arm", "control", "Evaluation arm to execute (control, semedit, baseline-diff)")
	flag.StringVar(&harness, "harness", "control", "Agent harness to drive (control, codex, agy)")
	flag.Var(&targets, "target", "Execution target harness[/model[/effort]] (repeatable, e.g. -target codex/gpt-5.6-luna/high)")
	flag.StringVar(&variantsFlag, "variants", "small,large", "Comma-separated context variants to run in matrix mode (small, large)")
	flag.BoolVar(&matrixMode, "matrix", false, "Execute full combinatorial matrix across targets, tasks, variants, and arms")
	flag.IntVar(&concurrency, "concurrency", 4, "Number of concurrent matrix benchmark workers")
	flag.StringVar(&outJSON, "out-json", "", "Optional path to save telemetry stats JSON")
	flag.StringVar(&outMD, "out-md", "", "Optional path to save rendered Markdown report")
	flag.StringVar(&outDir, "out-dir", "data/benchmarks/results", "Output directory to store individual benchmark results JSON and MD")
	flag.StringVar(&extractTo, "extract-to", "", "Extract fixture to target directory and exit (for agent eval trials)")
	flag.StringVar(&evalDir, "eval-dir", "", "Evaluate target directory with task oracle (for agent eval trials)")
	flag.DurationVar(&timeout, "timeout", 5*time.Minute, "Timeout per benchmark task")
	flag.StringVar(&mcpServerInstructionsRaw, "mcp-server-instructions", "none", "Server-wide semedit MCP instruction mode (none, descriptive, prescriptive; Codex only)")
	flag.Var(&provenance, "provenance", "Technical execution provenance key=value (repeatable; does not group results)")
	flag.Var(&provenance, "classifier", "Deprecated alias for -provenance")
	flag.BoolVar(&listMode, "list", false, "List all available benchmark tasks and their prompt variants")
	flag.Parse()

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

	if matrixMode || len(targets) > 0 {
		return runMatrix(runner, benchDir, targets, tasksFlag, taskID, variantsFlag, outDir, outJSON, outMD, timeout, concurrency)
	}

	// Legacy single-run path
	return runSingle(runner, benchDir, taskID, harness, arm, outJSON, outMD, timeout)
}

func runMatrix(runner *Runner, benchDir string, targets []Target, tasksFlag, singleTask, variantsFlag, outDir, outJSON, outMD string, timeout time.Duration, concurrency int) int {
	if len(targets) == 0 {
		targets = []Target{{Harness: "codex"}, {Harness: "agy"}}
	}

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
		sort.Strings(taskBases)
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

	fmt.Printf("==> Starting Matrix Benchmark (Concurrency: %d)\n", concurrency)
	fmt.Printf("    Targets:  %v\n", targets)
	fmt.Printf("    Tasks:    %v\n", taskBases)
	fmt.Printf("    Variants: %v\n", variants)
	fmt.Printf("    Arms:     %v\n\n", arms)
	fmt.Printf("    MCP server instructions: %s\n", runner.mcpServerInstructionsMode)
	if runner.provenance.String() != "" {
		fmt.Printf("    Provenance: %s\n", runner.provenance)
	}
	fmt.Println()

	type matrixJob struct {
		taskBase string
		variant  string
		target   Target
		arm      ArmType
		task     *Task
	}

	var jobs []matrixJob
	for _, taskBase := range taskBases {
		for _, target := range targets {
			fixtureFile := resolveFixturePath(benchDir, taskBase, "small")
			// #nosec G304 -- reading benchmark fixture
			data, err := os.ReadFile(fixtureFile)
			if err != nil {
				fmt.Printf("❌ Failed reading fixture %s: %v\n", fixtureFile, err)
				continue
			}

			task, err := ParseTask(data)
			if err != nil {
				fmt.Printf("❌ Failed parsing fixture %s: %v\n", fixtureFile, err)
				continue
			}

			if len(declaredPromptVariants(task)) == 0 {
				fmt.Printf("Skipping benchmark %s: no non-empty prompt_variants declared in its txtar fixture\n", task.Metadata.TaskID)
				continue
			}

			for _, variant := range variants {
				effectiveVariants := matrixPromptVariants(task, variant)
				if len(effectiveVariants) == 0 {
					fmt.Printf("Skipping benchmark %s variant %s: prompt variant is not declared in its txtar fixture\n", task.Metadata.TaskID, variant)
					continue
				}

				for _, effVar := range effectiveVariants {
					for _, arm := range arms {
						jobs = append(jobs, matrixJob{
							taskBase: taskBase,
							variant:  effVar,
							target:   target,
							arm:      arm,
							task:     task,
						})
					}
				}
			}
		}
	}

	var mu sync.Mutex
	var allRuns []*RunResult

	jobChan := make(chan matrixJob, len(jobs))
	for _, j := range jobs {
		jobChan <- j
	}
	close(jobChan)

	var wg sync.WaitGroup
	for w := 0; w < concurrency; w++ {
		wg.Go(func() {
			for j := range jobChan {
				ctx, cancel := context.WithTimeout(context.Background(), timeout)
				res, execErr := runner.ExecuteAgentDriver(ctx, j.task, j.target, j.arm, j.variant)
				cancel()

				if execErr != nil {
					mu.Lock()
					fmt.Printf("❌ [Target=%s Task=%s Variant=%s Arm=%s] System error: %v\n",
						j.target.String(), j.taskBase, j.variant, j.arm, execErr)
					mu.Unlock()
					continue
				}

				res.Variant = j.variant
				status := "PASS"
				if !res.Success {
					status = "FAIL"
				}

				mu.Lock()
				allRuns = append(allRuns, res)
				fmt.Printf("==> [Target=%s Task=%s Variant=%s Arm=%s] %s (took %v, turns: %d, out_tokens: %d)\n",
					j.target.String(), j.taskBase, j.variant, j.arm, status,
					res.WallClock.Round(time.Millisecond), res.Turns, res.OutputTokens)
				mu.Unlock()
			}
		})
	}

	wg.Wait()

	report := &BenchmarkReport{
		Timestamp:   time.Now(),
		Runs:        allRuns,
		Comparisons: BuildComparisons(allRuns),
	}

	if outDir != "" {
		repoRoot, _ := os.Getwd()
		type benchKey struct {
			taskBase              string
			target                string
			mcpServerInstructions MCPServerInstructionMode
		}
		benchGroups := make(map[benchKey][]*RunResult)
		for _, r := range allRuns {
			base := normalizeTaskBase(r.TaskID)
			k := benchKey{taskBase: base, target: r.Target.String(), mcpServerInstructions: r.MCPServerInstructions}
			benchGroups[k] = append(benchGroups[k], r)
		}

		for k, runs := range benchGroups {
			fixtureFile := resolveFixturePath(benchDir, k.taskBase, "small")
			relFixture, err := filepath.Rel(repoRoot, fixtureFile)
			if err != nil {
				relFixture = fixtureFile
			}
			provenance := ResolveTxtarProvenance(repoRoot, relFixture)

			targetSlug := strings.ReplaceAll(k.target, "/", "-")
			if instructionMode := normalizeMCPServerInstructions(k.mcpServerInstructions); instructionMode != MCPServerInstructionsNone {
				targetSlug += "-mcp-server-instructions-" + string(instructionMode)
			}
			taskOutDir := filepath.Join(outDir, k.taskBase)
			if err := os.MkdirAll(taskOutDir, 0o750); err != nil {
				fmt.Printf("❌ Failed to create task out dir %s: %v\n", taskOutDir, err)
				continue
			}

			benchComparisons := BuildComparisons(runs)
			for _, comp := range benchComparisons {
				comp.TxtarPath = filepath.ToSlash(relFixture)
				comp.TxtarProvenance = provenance
			}

			singleReport := &BenchmarkReport{
				Timestamp:   time.Now(),
				Runs:        runs,
				Comparisons: benchComparisons,
			}

			jsonFile := filepath.Join(taskOutDir, targetSlug+".json")
			mdFile := filepath.Join(taskOutDir, targetSlug+".md")
			if err := SaveReport(singleReport, jsonFile, mdFile); err != nil {
				fmt.Printf("❌ Failed to save benchmark %s (%s): %v\n", k.taskBase, k.target, err)
			} else {
				fmt.Printf(" Saved benchmark results: %s\n", jsonFile)
			}
		}
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
	sort.Strings(variants)
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
	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".txtar") {
				continue
			}
			fixturePath := filepath.Join(dir, entry.Name())
			// #nosec G304 -- reading benchmark fixture for listing
			data, err := os.ReadFile(filepath.Clean(fixturePath))
			if err != nil {
				continue
			}
			task, err := ParseTask(data)
			if err != nil || task.Metadata.TaskID == "" {
				continue
			}
			if seen[task.Metadata.TaskID] {
				continue
			}
			seen[task.Metadata.TaskID] = true

			promptVars := make([]string, 0, len(task.Metadata.PromptVariants))
			for k := range task.Metadata.PromptVariants {
				promptVars = append(promptVars, k)
			}
			sort.Strings(promptVars)

			benchmarks = append(benchmarks, BenchmarkInfo{
				TaskID:         task.Metadata.TaskID,
				Category:       task.Metadata.Category,
				Instruction:    task.Metadata.Instruction,
				PromptVariants: promptVars,
				FixturePath:    filepath.ToSlash(fixturePath),
			})
		}
	}

	sort.Slice(benchmarks, func(i, j int) bool {
		return benchmarks[i].TaskID < benchmarks[j].TaskID
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
