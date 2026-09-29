// Package main provides the standalone benchmarking harness for semedit.
package main

import (
	"bufio"
	"cmp"
	"context"
	"flag"
	"fmt"
	"io"
	"maps"
	"os"
	"os/signal"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
	"text/tabwriter"
	"time"
)

func main() {
	os.Exit(run())
}

func run() int {
	return runPlannedCLI()
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

func runPlannedCLI() int {
	cli, err := parseBenchmarkCLI(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "benchmark CLI: %v\n", err)
		return 1
	}
	if cli.ListFree {
		return listOpenRouterFreeModels()
	}
	if cli.List {
		return listBenchmarks(cli.Plan.BenchDir)
	}
	if cli.ExtractTo != "" || cli.EvalDir != "" {
		return runFixtureUtility(cli)
	}
	plan, err := BuildBenchmarkPlan(cli.Plan)
	if err != nil {
		fmt.Fprintf(os.Stderr, "planning error: %v\n", err)
		return 1
	}
	return executeBenchmarkPlan(plan)
}

type benchmarkCLIOptions struct {
	Plan      PlanOptions
	List      bool
	ListFree  bool
	ExtractTo string
	EvalDir   string
	TaskID    string
}

func parseBenchmarkCLI(args []string) (benchmarkCLIOptions, error) {
	fs := flag.NewFlagSet("benchmark-harness", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	var settings benchmarkCLIOptions
	var taskID, tasksRaw, arm, harness, variantsRaw string
	var targets TargetList
	var matrixMode, listMode, listFree, openRouterFreeTop10 bool
	var outJSON, outMD, outDir, runID string
	var timeout time.Duration
	var concurrency, repeats int
	var mcpRaw, policyRaw string
	var provenance ProvenanceSet
	fs.StringVar(&taskID, "task", "", "Benchmark task ID; default selects all discoverable fixtures")
	fs.StringVar(&tasksRaw, "tasks", "", "Comma-separated benchmark task IDs; default selects all fixtures")
	fs.StringVar(&settings.Plan.BenchDir, "dir", "testdata/bench", "Benchmark fixture directory")
	fs.StringVar(&arm, "arm", "", "Deprecated; use --target (explicit use is rejected)")
	fs.StringVar(&harness, "harness", "", "Deprecated single-target alias for --target")
	fs.Var(&targets, "target", "Execution target harness[/model[/effort]] (repeatable)")
	fs.IntVar(&repeats, "repeats", 1, "Number of independent trials")
	fs.StringVar(&variantsRaw, "variants", "small,large", "Comma-separated contexts with optional prompt")
	fs.BoolVar(&matrixMode, "matrix", false, "Deprecated no-op; all runs use the resolved plan pipeline")
	fs.IntVar(&concurrency, "concurrency", 4, "Concurrent benchmark jobs")
	fs.StringVar(&outJSON, "out-json", "", "Optional report JSON output path")
	fs.StringVar(&outMD, "out-md", "", "Optional report Markdown output path")
	fs.StringVar(&outDir, "out-dir", "data/benchmarks/results", "Parent directory for run-scoped results; empty disables run output")
	fs.StringVar(&runID, "run-id", "", "Required run identifier when --out-dir is non-empty")
	fs.StringVar(&settings.ExtractTo, "extract-to", "", "Extract fixture to target directory and exit")
	fs.StringVar(&settings.EvalDir, "eval-dir", "", "Evaluate target directory and exit")
	fs.DurationVar(&timeout, "timeout", 5*time.Minute, "Timeout per benchmark job")
	fs.StringVar(&mcpRaw, "mcp-server-instructions", "none", "Comma-separated server-wide MCP instruction modes: none, descriptive, prescriptive")
	fs.StringVar(&policyRaw, "semedit-arm-restrict", "write", "Semedit arm policy: read, write, readwrite")
	fs.BoolVar(&settings.Plan.SemeditPrewarmVerify, "semedit-prewarm-verify", false, "Prewarm the semedit arm fixture with a check-only verify CLI before agent execution")
	fs.Var(&provenance, "provenance", "Technical execution provenance key=value (repeatable)")
	fs.Var(&provenance, "classifier", "Deprecated alias for provenance")
	fs.BoolVar(&listMode, "list", false, "List benchmark fixtures and prompt variants")
	fs.BoolVar(&listFree, "list-openrouter-free", false, "List the OpenRouter free-model catalog")
	fs.BoolVar(&openRouterFreeTop10, "openrouter-free-top10", false, "Add the pinned free-model target")
	if err := fs.Parse(args); err != nil {
		return settings, fmt.Errorf("parse flags: %w", err)
	}
	explicit := make(map[string]bool)
	fs.Visit(func(f *flag.Flag) { explicit[f.Name] = true })
	if len(fs.Args()) > 0 && (len(fs.Args()) != 1 || fs.Args()[0] != "list") {
		return settings, fmt.Errorf("unexpected positional arguments: %s", strings.Join(fs.Args(), " "))
	}
	settings.ListFree = listFree
	settings.List = listMode || len(fs.Args()) == 1
	settings.TaskID = taskID
	if explicit["repeats"] && repeats < 1 {
		return settings, fmt.Errorf("--repeats must be at least 1")
	}
	if explicit["concurrency"] && concurrency < 1 {
		return settings, fmt.Errorf("--concurrency must be at least 1")
	}
	if explicit["timeout"] && timeout <= 0 {
		return settings, fmt.Errorf("--timeout must be positive")
	}
	if explicit["arm"] {
		return settings, fmt.Errorf("--arm is no longer supported; use --target to request paired baseline and semedit jobs")
	}
	if explicit["harness"] && len(targets) > 0 {
		return settings, fmt.Errorf("--harness cannot be combined with --target")
	}
	if explicit["task"] && tasksRaw != "" {
		return settings, fmt.Errorf("--task cannot be combined with --tasks")
	}
	if explicit["harness"] {
		target, err := ParseTarget(harness)
		if err != nil {
			return settings, fmt.Errorf("invalid --harness: %w", err)
		}
		targets = append(targets, target)
	}
	if openRouterFreeTop10 {
		targets = append(targets, Target{Harness: string(HarnessOpenCode), Model: "openrouter/free-top10"})
	}
	if explicit["harness"] && len(targets) > 1 {
		return settings, fmt.Errorf("--harness selects exactly one target")
	}
	tasks := make([]string, 0)
	if taskID != "" {
		tasks = append(tasks, taskID)
	}
	for value := range strings.SplitSeq(tasksRaw, ",") {
		value = strings.TrimSpace(value)
		if value != "" {
			tasks = append(tasks, value)
		}
	}
	variants := make([]string, 0)
	for value := range strings.SplitSeq(variantsRaw, ",") {
		value = strings.TrimSpace(value)
		if value != "" {
			variants = append(variants, value)
		}
	}
	policy, err := ParseSemeditArmRestriction(policyRaw)
	if err != nil {
		return settings, fmt.Errorf("invalid --semedit-arm-restrict: %w", err)
	}
	mcpModes, err := ParseMCPServerInstructionModes(mcpRaw)
	if err != nil {
		return settings, fmt.Errorf("invalid --mcp-server-instructions: %w", err)
	}
	settings.Plan.TaskIDs = tasks
	settings.Plan.Targets = targets
	settings.Plan.Variants = variants
	settings.Plan.Repeats = repeats
	settings.Plan.Concurrency = concurrency
	settings.Plan.Timeout = timeout
	settings.Plan.OutDir = outDir
	settings.Plan.RunID = runID
	settings.Plan.OutJSON = outJSON
	settings.Plan.OutMD = outMD
	settings.Plan.MCPServerInstructions = mcpModes
	settings.Plan.SemeditArmRestriction = policy
	settings.Plan.Provenance = provenance.Clone()
	return settings, nil
}

func runFixtureUtility(cli benchmarkCLIOptions) int {
	if cli.TaskID == "" {
		fmt.Fprintln(os.Stderr, "-task is required for extraction or evaluation")
		return 1
	}
	fixturePath := filepath.Join(cli.Plan.BenchDir, cli.TaskID+".txtar")
	if _, err := os.Stat(fixturePath); err != nil {
		fixturePath = filepath.Join(cli.Plan.BenchDir, strings.ReplaceAll(cli.TaskID, "-", "_")+".txtar")
	}
	if cli.ExtractTo != "" {
		if err := extractFixture(fixturePath, cli.ExtractTo); err != nil {
			fmt.Fprintf(os.Stderr, "extract error: %v\n", err)
			return 1
		}
		return 0
	}
	if err := evaluateDir(fixturePath, cli.EvalDir, []string{"api/server.go"}); err != nil {
		fmt.Fprintf(os.Stderr, "evaluate error: %v\n", err)
		return 1
	}
	return 0
}

func executeBenchmarkPlan(plan *BenchmarkPlan) int {
	buffered := bufio.NewWriter(os.Stdout)
	if err := RenderBenchmarkPlan(buffered, plan); err != nil {
		fmt.Fprintf(os.Stderr, "print plan: %v\n", err)
		return 1
	}
	if err := buffered.Flush(); err != nil {
		fmt.Fprintf(os.Stderr, "flush plan: %v\n", err)
		return 1
	}
	resultDir := ""
	var err error
	if plan.Options.OutDir != "" {
		resultDir, err = createBenchmarkRunDir(plan.Options.OutDir, plan.Options.RunID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "invalid benchmark result directory: %v\n", err)
			return 1
		}
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	outcomes, progressErr := ExecutePlan(ctx, plan, os.Stdout, func(ctx context.Context, job Job) (*RunResult, error) {
		runner := NewRunner(filepath.Join(".scratch", "benchmarks"), WithMCPServerInstructions(job.MCPServerInstructions), WithSemeditArmRestriction(plan.Options.SemeditArmRestriction), WithSemeditPrewarmVerify(plan.Options.SemeditPrewarmVerify), WithProvenance(plan.Options.Provenance))
		if job.Arm == ArmControl {
			return runner.ExecuteControlVariant(ctx, job.Task, job.Context)
		}
		return runner.ExecuteAgent(ctx, job.Execution)
	})
	results := make([]*RunResult, 0, len(outcomes))
	failed := progressErr != nil
	for _, outcome := range outcomes {
		results = append(results, outcome.Result)
		if outcome.Err != nil || outcome.Result == nil || !outcome.Result.Success {
			failed = true
		}
	}
	report := &BenchmarkReport{Timestamp: time.Now(), Runs: results, Comparisons: BuildComparisons(results)}
	if err := saveRunReports(resultDir, plan, results, report); err != nil {
		fmt.Fprintf(os.Stderr, "save benchmark reports: %v\n", err)
		return 1
	}
	if progressErr != nil {
		fmt.Fprintf(os.Stderr, "write progress: %v\n", progressErr)
	}
	if failed {
		return 1
	}
	return 0
}

func saveRunReports(resultDir string, plan *BenchmarkPlan, results []*RunResult, report *BenchmarkReport) error {
	if resultDir != "" {
		if err := saveMatrixReports(resultDir, plan.Options.BenchDir, results); err != nil {
			return fmt.Errorf("save run reports: %w", err)
		}
	}
	if plan.Options.OutJSON != "" || plan.Options.OutMD != "" {
		if err := SaveReport(report, plan.Options.OutJSON, plan.Options.OutMD); err != nil {
			return fmt.Errorf("save reports: %w", err)
		}
	}
	return nil
}
