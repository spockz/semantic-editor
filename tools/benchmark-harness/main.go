// Package main provides the standalone benchmarking harness for semedit.
package main

import (
	"bufio"
	"cmp"
	"context"
	"errors"
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

	"github.com/spockz/semantic-editor/internal/gocache"
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
		if promptVariant == "explicit" {
			continue
		}
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
	if storageErr := validateDurableBenchmarkStorage(cli.Plan); storageErr != nil {
		fmt.Fprintf(os.Stderr, "benchmark storage: %v\n", storageErr)
		return 1
	}
	if cli.Plan.RAMFixtureRoot != "" {
		runner := NewRunner(filepath.Join(".scratch", "benchmarks"), WithFixtureRoot(cli.Plan.RAMFixtureRoot), WithModuleCache(cli.Plan.ModuleCache))
		if fixtureErr := runner.validateFixtureRoot(); fixtureErr != nil {
			fmt.Fprintf(os.Stderr, "benchmark storage: %v\n", fixtureErr)
			return 1
		}
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

type benchmarkCLIValidation struct {
	explicit          map[string]bool
	targets           []Target
	repeats           int
	concurrency       int
	timeout           time.Duration
	harness           string
	tasksRaw          string
	openRouterFreeTop bool
}

type benchmarkCLIFlags struct {
	settings            benchmarkCLIOptions
	explicit            map[string]bool
	args                []string
	taskID              string
	tasksRaw            string
	arm                 string
	harness             string
	variantsRaw         string
	targets             TargetList
	matrixMode          bool
	listMode            bool
	listFree            bool
	openRouterFreeTop10 bool
	outJSON             string
	outMD               string
	outDir              string
	runID               string
	resume              bool
	semeditSkill        bool
	timeout             time.Duration
	concurrency         int
	repeats             int
	mcpRaw              string
	policyRaw           string
	provenance          ProvenanceSet
	ramFixtureRoot      string
	moduleCache         string
}

func parseBenchmarkCLI(args []string) (benchmarkCLIOptions, error) {
	input, err := parseBenchmarkCLIFlags(args)
	if err != nil {
		return benchmarkCLIOptions{}, err
	}
	settings := input.settings
	targets, err := validateBenchmarkCLI(benchmarkCLIValidation{
		explicit:          input.explicit,
		targets:           input.targets,
		repeats:           input.repeats,
		concurrency:       input.concurrency,
		timeout:           input.timeout,
		harness:           input.harness,
		tasksRaw:          input.tasksRaw,
		openRouterFreeTop: input.openRouterFreeTop10,
	})
	if err != nil {
		return settings, err
	}
	tasks := parseCLIList(input.taskID, input.tasksRaw)
	variants := parseCLIList("", input.variantsRaw)
	policy, err := ParseSemeditArmRestriction(input.policyRaw)
	if err != nil {
		return settings, fmt.Errorf("invalid --semedit-arm-restrict: %w", err)
	}
	mcpModes, err := ParseMCPServerInstructionModes(input.mcpRaw)
	if err != nil {
		return settings, fmt.Errorf("invalid --mcp-server-instructions: %w", err)
	}
	applyBenchmarkCLIPlan(&settings, input, targets, tasks, variants, policy, mcpModes)
	return settings, nil
}

func parseBenchmarkCLIFlags(args []string) (benchmarkCLIFlags, error) {
	var input benchmarkCLIFlags
	fs := flag.NewFlagSet("benchmark-harness", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	registerBenchmarkCLIFlags(fs, &input)
	if err := fs.Parse(args); err != nil {
		return input, fmt.Errorf("parse flags: %w", err)
	}
	input.explicit = make(map[string]bool)
	fs.Visit(func(f *flag.Flag) { input.explicit[f.Name] = true })
	input.args = fs.Args()
	if len(input.args) > 0 && (len(input.args) != 1 || input.args[0] != "list") {
		return input, fmt.Errorf("unexpected positional arguments: %s", strings.Join(input.args, " "))
	}
	input.settings.ListFree = input.listFree
	input.settings.List = input.listMode || len(input.args) == 1
	input.settings.TaskID = input.taskID
	return input, nil
}

func registerBenchmarkCLIFlags(fs *flag.FlagSet, input *benchmarkCLIFlags) {
	fs.StringVar(&input.taskID, "task", "", "Benchmark task ID; default selects all discoverable fixtures")
	fs.StringVar(&input.tasksRaw, "tasks", "", "Comma-separated benchmark task IDs; default selects all fixtures")
	fs.StringVar(&input.settings.Plan.BenchDir, "dir", "testdata/bench", "Benchmark fixture directory")
	fs.StringVar(&input.arm, "arm", "", "Deprecated; use --target (explicit use is rejected)")
	fs.StringVar(&input.harness, "harness", "", "Deprecated single-target alias for --target")
	fs.Var(&input.targets, "target", "Execution target harness[/model[/effort]] (repeatable)")
	fs.IntVar(&input.repeats, "repeats", 1, "Number of independent trials")
	fs.StringVar(&input.variantsRaw, "variants", "small,large", "Comma-separated contexts with optional prompt")
	fs.BoolVar(&input.matrixMode, "matrix", false, "Deprecated no-op; all runs use the resolved plan pipeline")
	fs.IntVar(&input.concurrency, "concurrency", 4, "Concurrent benchmark jobs")
	fs.StringVar(&input.outJSON, "out-json", "", "Optional report JSON output path")
	fs.StringVar(&input.outMD, "out-md", "", "Optional report Markdown output path")
	fs.StringVar(&input.outDir, "out-dir", "data/benchmarks/results", "Parent directory for run-scoped results; empty disables run output")
	fs.StringVar(&input.runID, "run-id", "", "Required run identifier when --out-dir is non-empty")
	fs.BoolVar(&input.resume, "resume", false, "Resume an existing run directory and skip jobs with saved results")
	fs.BoolVar(&input.semeditSkill, "semedit-skill", false, "Install the semedit skill in both semedit benchmark arms")
	fs.StringVar(&input.settings.ExtractTo, "extract-to", "", "Extract fixture to target directory and exit")
	fs.StringVar(&input.settings.EvalDir, "eval-dir", "", "Evaluate target directory and exit")
	fs.DurationVar(&input.timeout, "timeout", 5*time.Minute, "Timeout per benchmark job")
	fs.StringVar(&input.mcpRaw, "mcp-server-instructions", "prescriptive", "Comma-separated server-wide MCP instruction modes: none, descriptive, prescriptive")
	fs.StringVar(&input.policyRaw, "semedit-arm-restrict", "write", "Semedit arm policy: read, write, readwrite")
	fs.BoolVar(&input.settings.Plan.SemeditPrewarmVerify, "semedit-prewarm-verify", true, "Prewarm the semedit arm fixture with a check-only verify CLI before agent execution")
	fs.StringVar(&input.ramFixtureRoot, "ram-fixture-root", "", "Mounted tmpfs root for disposable fixture workspaces")
	fs.StringVar(&input.moduleCache, "module-cache", "", "Persistent SSD Go module cache for offline benchmark fixtures")
	fs.Var(&input.provenance, "provenance", "Technical execution provenance key=value (repeatable)")
	fs.Var(&input.provenance, "classifier", "Deprecated alias for provenance")
	fs.BoolVar(&input.listMode, "list", false, "List benchmark fixtures and prompt variants")
	fs.BoolVar(&input.listFree, "list-openrouter-free", false, "List the OpenRouter free-model catalog")
	fs.BoolVar(&input.openRouterFreeTop10, "openrouter-free-top10", false, "Add the pinned free-model target")
}

func parseCLIList(first, second string) []string {
	values := make([]string, 0)
	if first != "" {
		values = append(values, first)
	}
	for value := range strings.SplitSeq(second, ",") {
		value = strings.TrimSpace(value)
		if value != "" {
			values = append(values, value)
		}
	}
	return values
}

func applyBenchmarkCLIPlan(settings *benchmarkCLIOptions, input benchmarkCLIFlags, targets []Target, tasks, variants []string, policy SemeditArmRestriction, mcpModes []MCPServerInstructionMode) {
	settings.Plan.TaskIDs = tasks
	settings.Plan.Targets = targets
	settings.Plan.Variants = variants
	settings.Plan.Repeats = input.repeats
	settings.Plan.Concurrency = input.concurrency
	settings.Plan.Timeout = input.timeout
	settings.Plan.OutDir = input.outDir
	settings.Plan.RunID = input.runID
	settings.Plan.Resume = input.resume
	settings.Plan.SemeditSkill = input.semeditSkill
	settings.Plan.OutJSON = input.outJSON
	settings.Plan.OutMD = input.outMD
	settings.Plan.MCPServerInstructions = mcpModes
	settings.Plan.SemeditArmRestriction = policy
	settings.Plan.Provenance = input.provenance.Clone()
	settings.Plan.RAMFixtureRoot = input.ramFixtureRoot
	settings.Plan.ModuleCache = input.moduleCache
	if settings.Plan.RAMFixtureRoot != "" || settings.Plan.ModuleCache != "" {
		if settings.Plan.Provenance == nil {
			settings.Plan.Provenance = make(ProvenanceSet)
		}
		storage := "disk"
		if settings.Plan.RAMFixtureRoot != "" {
			storage = "tmpfs"
		}
		settings.Plan.Provenance["fixture_storage"] = storage
		settings.Plan.Provenance["module_cache"] = settings.Plan.ModuleCache
		settings.Plan.Provenance["module_cache_network"] = "offline"
	}
}

func validateBenchmarkCLI(input benchmarkCLIValidation) ([]Target, error) {
	targets := append([]Target(nil), input.targets...)
	if input.explicit["repeats"] && input.repeats < 1 {
		return nil, fmt.Errorf("--repeats must be at least 1")
	}
	if input.explicit["concurrency"] && input.concurrency < 1 {
		return nil, fmt.Errorf("--concurrency must be at least 1")
	}
	if input.explicit["timeout"] && input.timeout <= 0 {
		return nil, fmt.Errorf("--timeout must be positive")
	}
	if input.explicit["arm"] {
		return nil, fmt.Errorf("--arm is no longer supported; use --target to request paired baseline and semedit jobs")
	}
	if input.explicit["harness"] && len(targets) > 0 {
		return nil, fmt.Errorf("--harness cannot be combined with --target")
	}
	if input.explicit["task"] && input.tasksRaw != "" {
		return nil, fmt.Errorf("--task cannot be combined with --tasks")
	}
	if input.explicit["harness"] {
		target, err := ParseTarget(input.harness)
		if err != nil {
			return nil, fmt.Errorf("invalid --harness: %w", err)
		}
		targets = append(targets, target)
	}
	if input.openRouterFreeTop {
		targets = append(targets, Target{Harness: string(HarnessOpenCode), Model: "openrouter/free-top10"})
	}
	if input.explicit["harness"] && len(targets) > 1 {
		return nil, fmt.Errorf("--harness selects exactly one target")
	}
	return targets, nil
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
	ctx := context.Background()
	if cli.Plan.ModuleCache != "" {
		ctx = gocache.WithModuleCache(ctx, cli.Plan.ModuleCache)
	}
	if err := evaluateDir(ctx, fixturePath, cli.EvalDir, []string{"api/server.go"}); err != nil {
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
	resultDir, err := prepareBenchmarkRunDir(plan)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid benchmark result directory: %v\n", err)
		return 1
	}
	completed := make(map[string]*RunResult)
	if plan.Options.Resume {
		completed, err = loadJobCheckpoints(resultDir, plan)
		if err != nil {
			fmt.Fprintf(os.Stderr, "load benchmark checkpoints: %v\n", err)
			return 1
		}
		fmt.Printf("Resuming run %s: %d saved jobs, %d jobs remaining\n", plan.Options.RunID, len(completed), len(plan.Jobs)-len(completed))
	}
	pending := pendingBenchmarkPlan(plan, completed)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	relayURL, modelLimits, closeRelay, relayErr := prepareLocalModelRunRelay(pending.Jobs)
	if relayErr != nil {
		fmt.Fprintf(os.Stderr, ("start local model relay: %v\n"), relayErr)
		return 1
	}
	defer func() {
		if closeErr := closeRelay(); closeErr != nil {
			fmt.Fprintf(os.Stderr, "close local model relay: %v\n", closeErr)
		}
	}()
	freshOutcomes, progressErr := ExecutePlanWithCompletion(ctx, pending, os.Stdout, func(ctx context.Context, job Job) (*RunResult, error) {
		runner := NewRunner(filepath.Join(".scratch", "benchmarks"), WithFixtureRoot(plan.Options.RAMFixtureRoot), WithModuleCache(plan.Options.ModuleCache), WithMCPServerInstructions(job.MCPServerInstructions), WithSemeditArmRestriction(plan.Options.SemeditArmRestriction), WithSemeditPrewarmVerify(plan.Options.SemeditPrewarmVerify), WithProvenance(plan.Options.Provenance), WithLocalModelRelayURL(relayURL), WithLocalModelLimits(modelLimits))
		if job.Arm == ArmControl {
			return runner.ExecuteControlVariant(ctx, job.Task, job.Context)
		}
		return runner.ExecuteAgent(ctx, job.Execution)
	}, func(outcome JobOutcome) error {
		if resultDir == "" {
			return nil
		}
		if ctx.Err() != nil && errors.Is(outcome.Err, context.Canceled) {
			return nil
		}
		path, err := saveJobOutcomeCheckpoint(resultDir, outcome)
		if err != nil {
			return err
		}
		if _, err := fmt.Fprintf(os.Stdout, " Saved individual job result: %s\n", path); err != nil {
			return fmt.Errorf("print job result path: %w", err)
		}
		return nil
	})
	outcomes := mergeJobOutcomes(plan, completed, freshOutcomes)
	results := make([]*RunResult, 0, len(outcomes))
	failed := progressErr != nil || len(outcomes) != len(plan.Jobs)
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
		fmt.Fprintf(os.Stderr, "save job checkpoint or write progress: %v\n", progressErr)
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
