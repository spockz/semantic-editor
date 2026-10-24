// These process-boundary tests ensure the public CLI executes only its printed plan and isolates arm policies.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/spockz/semantic-editor/internal/pipeline"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

func TestBenchmarkProcessHelper(t *testing.T) {
	mode := os.Getenv("SEMEDIT_TEST_PROCESS")
	if mode == "" {
		return
	}
	marker := slices.Index(os.Args, "--")
	if marker < 0 {
		t.Fatal("missing subprocess argument boundary")
	}
	args := os.Args[marker+1:]
	if mode == "cli" {
		os.Args = append([]string{os.Args[0]}, args...)
		flag.CommandLine = flag.NewFlagSet("benchmark-harness", flag.ExitOnError)
		os.Exit(run())
	}
	if mode != "provider" {
		t.Fatalf("invalid subprocess mode %q", mode)
	}
	if len(args) == 0 {
		t.Fatal("provider missing args")
	}
	if args[0] == "app-server" {
		runFakeCodexAppServer(t)
		return
	}
	plan, err := os.ReadFile(os.Getenv("SEMEDIT_TEST_OUTPUT"))
	if err != nil {
		t.Fatal(err)
	}
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	invocation := cliProviderInvocation{Args: args, Prompt: args[len(args)-1], Plan: string(plan), Directory: cwd}
	data, err := json.Marshal(invocation)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(os.Getenv("SEMEDIT_TEST_EVIDENCE"), fmt.Sprintf("%d.json", time.Now().UnixNano()))
	if err := pipeline.WriteAtomic(path, data); err != nil {
		t.Fatal(err)
	}
	if err := pipeline.WriteAtomic(filepath.Join(cwd, "main.go"), []byte("package main\nfunc Done() {}\nfunc main() {}\n")); err != nil {
		t.Fatal(err)
	}
	events := []map[string]any{{"type": "thread.started", "thread_id": "fake-session"}, {"type": "turn.started"}}
	if !slices.Contains(args, "mcp_servers.semedit.enabled=false") {
		events = append(events, map[string]any{"type": "item.completed", "item": map[string]any{"id": "edit", "type": "mcp_tool_call", "server": "semedit", "tool": "semantic_insert_construct", "status": "completed"}})
	}
	events = append(events, map[string]any{"type": "item.completed", "item": map[string]any{"id": "answer", "type": "agent_message", "text": "DONE"}}, map[string]any{"type": "turn.completed", "usage": map[string]any{"input_tokens": 5, "output_tokens": 3}})
	for _, event := range events {
		if err := json.NewEncoder(os.Stdout).Encode(event); err != nil {
			t.Fatal(err)
		}
	}
	os.Exit(0)
}

func TestCLIPlansBeforeExecutionAndIsolatesRestrictionPolicies(t *testing.T) {
	var baselinePrompt string
	for _, policy := range []string{"read", "write", "readwrite", "default"} {
		t.Run(policy, func(t *testing.T) {
			assertCLIRestrictionPolicy(t, policy, &baselinePrompt)
		})
	}
}

func assertCLIRestrictionPolicy(t *testing.T, policy string, baselinePrompt *string) {
	t.Helper()
	flags, resolved := cliRestrictionFlags(policy)
	trial := runCLITrial(t, flags, false)
	if trial.Err != nil {
		t.Fatalf("CLI failed: %v\n%s", trial.Err, trial.Output)
	}
	assertCLIPairedResults(t, trial)
	for _, invocation := range trial.Invocations {
		assertCLIPlanWasResolved(t, invocation)
		if invocation.Prompt == semanticIntegrationReflectionPrompt {
			assertCLIIntegrationFollowup(t, invocation)
			continue
		}
		if slices.Contains(invocation.Args, "mcp_servers.semedit.enabled=false") {
			assertCLIBaselinePolicyIsolation(t, policy, baselinePrompt, invocation.Prompt)
			continue
		}
		assertCLISemeditPolicy(t, policy, resolved, invocation.Prompt, invocation.Args)
	}
	if trial.Report.Runs[1].SemanticIntegrationReflection == nil {
		t.Fatal("CLI report did not retain integration feedback")
	}
	assertCLIResultConditions(t, trial, resolved)
	if trial.Report.Runs[0].ComparisonPairID != trial.Report.Runs[1].ComparisonPairID {
		t.Error("agent arms not paired")
	}
	if !strings.Contains(trial.Output, "[2/2 completed · 0 remaining]") {
		t.Error("completion accounting missing")
	}
}

func cliRestrictionFlags(policy string) ([]string, string) {
	if policy == "default" {
		return nil, "write"
	}
	return []string{"--semedit-arm-restrict=" + policy}, policy
}

func assertCLIPairedResults(t *testing.T, trial cliTrial) {
	t.Helper()
	if len(trial.Invocations) != 3 || len(trial.Report.Runs) != 2 {
		t.Fatalf("invocations=%d results=%d, want two task jobs and one integration follow-up\n%s", len(trial.Invocations), len(trial.Report.Runs), trial.Output)
	}
}

func assertCLIIntegrationFollowup(t *testing.T, invocation cliProviderInvocation) {
	t.Helper()
	arguments := strings.Join(invocation.Args, " ")
	if !strings.Contains(arguments, "mcp_servers.semedit.enabled=true") || !strings.Contains(arguments, "threadId=persistent-thread") {
		t.Error("integration question did not continue the original semedit thread")
	}
	if !strings.Contains(arguments, `mcp_servers.semedit.default_tools_approval_mode="approve"`) {
		t.Error("semedit app-server did not enable approval for its default tools")
	}
}

func assertCLIPlanWasResolved(t *testing.T, invocation cliProviderInvocation) {
	t.Helper()
	if !strings.Contains(invocation.Plan, "Resolved benchmark plan: 2 jobs") || !strings.Contains(invocation.Plan, "arm=baseline-diff") || !strings.Contains(invocation.Plan, "arm=semedit") {
		t.Errorf("provider started before complete plan:\n%s", invocation.Plan)
	}
}

func assertCLIBaselinePolicyIsolation(t *testing.T, policy string, baselinePrompt *string, prompt string) {
	t.Helper()
	if *baselinePrompt == "" {
		*baselinePrompt = prompt
	}
	if prompt != *baselinePrompt {
		t.Errorf("policy %s altered baseline prompt", policy)
	}
	if strings.Contains(prompt, "Do not inspect Go implementation files (non-test *.go)") || strings.Contains(prompt, "Shell commands for builds") {
		t.Errorf("baseline received policy steering: %s", prompt)
	}
}

func assertCLISemeditPolicy(t *testing.T, policy, resolved, prompt string, args []string) {
	t.Helper()
	if !slices.Contains(args, "mcp_servers.semedit.enabled=true") {
		t.Error("semedit MCP not enabled")
	}
	read := strings.Contains(prompt, "Do not inspect Go implementation files (non-test *.go)")
	write := strings.Contains(prompt, "supported source code modifications")
	if read != (resolved != "write") || write != (resolved != "read") {
		t.Errorf("policy %s wrong steering: %s", policy, prompt)
	}
	if !strings.Contains(prompt, "Shell commands for builds and tests are allowed") {
		t.Error("shell build/test permission missing")
	}
}

func TestBenchmarkCLIPrewarmDefaultAndOptOut(t *testing.T) {
	defaults, err := parseBenchmarkCLI([]string{"--out-dir="})
	if err != nil {
		t.Fatal(err)
	}
	if !defaults.Plan.SemeditPrewarmVerify {
		t.Fatal("semedit verify prewarm should be enabled by default")
	}
	if defaults.Plan.SemeditBatching {
		t.Fatal("semedit batching should be disabled by default")
	}
	enabled, err := parseBenchmarkCLI([]string{"--out-dir=", "--semedit-batching=true"})
	if err != nil {
		t.Fatal(err)
	}
	if !enabled.Plan.SemeditBatching {
		t.Fatal("explicit semedit batching opt-in was ignored")
	}
	disabled, err := parseBenchmarkCLI([]string{"--out-dir=", "--semedit-prewarm-verify=false"})
	if err != nil {
		t.Fatal(err)
	}
	if disabled.Plan.SemeditPrewarmVerify {
		t.Fatal("explicit semedit verify opt-out was ignored")
	}
}

func TestCLIPlanningFailuresNeverStartAProvider(t *testing.T) {
	for _, tc := range []struct {
		name      string
		flags     []string
		malformed bool
	}{
		{name: "invalid policy", flags: []string{"--semedit-arm-restrict=invalid"}},
		{name: "explicit empty policy", flags: []string{"--semedit-arm-restrict="}},
		{name: "malformed fixture", malformed: true},
		{name: "unsupported arm selection", flags: []string{"--arm=semedit"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			trial := runCLITrial(t, tc.flags, tc.malformed)
			if trial.Err == nil {
				t.Errorf("planning should fail\n%s", trial.Output)
			}
			if len(trial.Invocations) != 0 {
				t.Errorf("planning failure started %d providers", len(trial.Invocations))
			}
		})
	}
}

func TestBenchmarkMakeTargetsForwardRestrictionPolicy(t *testing.T) {
	for _, target := range []string{"bench", "bench-all"} {
		for _, policy := range []string{"read", "write", "readwrite", ""} {
			args := []string{"-n", "-f", "Makefile", target}
			expected := policy
			if policy == "" {
				expected = "write"
			} else {
				args = append(args, "SEMEDIT_ARM_RESTRICT="+policy)
			}
			// #nosec G204 -- dry-run repository benchmark Make targets with controlled values.
			out, err := exec.Command("make", args...).CombinedOutput()
			if err != nil {
				t.Fatalf("make %s: %v\n%s", target, err, out)
			}
			if !strings.Contains(string(out), "--semedit-arm-restrict \""+expected+"\"") {
				t.Errorf("%s did not forward %s:\n%s", target, expected, out)
			}
		}
	}
}

func TestBenchmarkMakeTargetsForwardResumeFlag(t *testing.T) {
	for _, target := range []string{"bench", "bench-all"} {
		args := []string{"-n", "-f", "Makefile", target, "RESUME=true", "RUN_ID=resume-run"}
		// #nosec G204 -- dry-run repository benchmark Make targets with controlled values.
		output, err := exec.Command("make", args...).CombinedOutput()
		if err != nil {
			t.Fatalf("make %s: %v\n%s", target, err, output)
		}
		if !strings.Contains(string(output), "--resume") || !strings.Contains(string(output), "--run-id \"resume-run\"") {
			t.Errorf("%s did not forward resume and the selected run ID:\n%s", target, output)
		}
	}
}

func TestBenchmarkMakeTargetsForwardRAMRootAndModuleCache(t *testing.T) {
	for _, target := range []string{"bench", "bench-all"} {
		args := []string{"-n", "-f", "Makefile", target, "BENCH_RAM_ROOT=/mnt/benchmark-ram", "BENCH_MODCACHE=/ssd/go-mod"}
		// #nosec G204 -- dry-run repository benchmark Make targets with controlled values.
		output, err := exec.Command("make", args...).CombinedOutput()
		if err != nil {
			t.Fatalf("make %s: %v\n%s", target, err, output)
		}
		for _, argument := range []string{`--ram-fixture-root "/mnt/benchmark-ram"`, `--module-cache "/ssd/go-mod"`} {
			if !strings.Contains(string(output), argument) {
				t.Errorf("%s did not forward %s:\n%s", target, argument, output)
			}
		}
	}
}

func TestBenchmarkMakeTargetsDefaultToRAMAndPersistentModuleCache(t *testing.T) {
	for _, target := range []string{"bench", "bench-all"} {
		args := []string{"-n", "-f", "Makefile", target, "BENCH_RAM_DEFAULT=/mnt/benchmark-ram", "GOMODCACHE=/ssd/go-mod"}
		// #nosec G204 -- dry-run repository benchmark Make targets with controlled values.
		output, err := exec.Command("make", args...).CombinedOutput()
		if err != nil {
			t.Fatalf("make %s: %v\n%s", target, err, output)
		}
		for _, argument := range []string{`--ram-fixture-root "/mnt/benchmark-ram"`, `--module-cache "/ssd/go-mod"`} {
			if !strings.Contains(string(output), argument) {
				t.Errorf("%s default did not forward %s:\n%s", target, argument, output)
			}
		}
	}
}

func TestCLIControlSharesSchedulerAndRetainsFailure(t *testing.T) {
	trial := runCLITrial(t, []string{"--target=control"}, false)
	if trial.Err == nil {
		t.Error("unsupported control task should return an error")
	}
	if len(trial.Report.Runs) != 3 {
		t.Fatalf("results=%d, want two agent arms and failed control\n%s", len(trial.Report.Runs), trial.Output)
	}
	if len(trial.Invocations) != 3 {
		t.Errorf("provider invocations=%d, want two agent turns and one semedit reflection; control must not invoke one", len(trial.Invocations))
	}
	controlCount := 0
	for _, run := range trial.Report.Runs {
		if run.Arm != ArmControl {
			continue
		}
		controlCount++
		if run.Error == "" || run.Success {
			t.Errorf("control failure was lost: %+v", run)
		}
		if run.SemeditArmRestrict != "" || run.SemeditArmRestrictionApplied || run.ComparisonPairID != "" {
			t.Error("control inherited agent comparison policy")
		}
		if run.JobID == "" || !strings.Contains(trial.Output, run.JobID) {
			t.Error("control missing planned identity")
		}
	}
	if controlCount != 1 {
		t.Errorf("control results=%d, want 1", controlCount)
	}
	if !strings.Contains(trial.Output, "Resolved benchmark plan: 3 jobs") || !strings.Contains(trial.Output, "[3/3 completed · 0 remaining]") {
		t.Errorf("control not accounted in shared scheduler:\n%s", trial.Output)
	}
}

func TestCLIMCPInstructionModeMatrixUsesPerJobRuntimeSettings(t *testing.T) {
	trial := runCLITrial(t, []string{"--mcp-server-instructions=none,descriptive,directive,prescriptive"}, false)
	if trial.Err != nil {
		t.Fatalf("CLI failed: %v\n%s", trial.Err, trial.Output)
	}
	assertMCPInstructionModeMatrix(t, trial)
}

func TestCLIDefaultMCPInstructionsArePrescriptive(t *testing.T) {
	trial := runCLITrial(t, nil, false)
	if trial.Err != nil {
		t.Fatalf("CLI failed: %v\n%s", trial.Err, trial.Output)
	}
	if !strings.Contains(trial.Output, "MCP-instructions=prescriptive") {
		t.Fatalf("resolved CLI plan omitted the prescriptive default:\n%s", trial.Output)
	}
	if len(trial.Report.Runs) == 0 {
		t.Fatal("CLI report contained no runs")
	}
	for _, run := range trial.Report.Runs {
		if run.MCPServerInstructions != MCPServerInstructionsPrescriptive {
			t.Errorf("run instruction mode = %q, want prescriptive", run.MCPServerInstructions)
		}
	}
}

func assertMCPInstructionModeMatrix(t *testing.T, trial cliTrial) {
	t.Helper()
	if len(trial.Invocations) != 9 || len(trial.Report.Runs) != 6 {
		t.Fatalf("invocations=%d results=%d, want three paired mode cells plus their integration reflections\n%s", len(trial.Invocations), len(trial.Report.Runs), trial.Output)
	}
	if len(trial.Report.Comparisons) != 3 {
		t.Errorf("comparisons=%d, want one per canonical mode", len(trial.Report.Comparisons))
	}
	wantModes := []MCPServerInstructionMode{MCPServerInstructionsNone, MCPServerInstructionsDescriptive, MCPServerInstructionsPrescriptive}
	pairedModes := collectMCPInstructionRunModes(t, trial.Report.Runs)
	assertMCPInstructionModePairs(t, pairedModes, wantModes)
	assertMCPInstructionPromptModes(t, trial.Invocations)
}

type mcpInstructionRunModes struct {
	pairByMode map[MCPServerInstructionMode]string
	counts     map[MCPServerInstructionMode]int
	jobIDs     map[string]bool
}

func collectMCPInstructionRunModes(t *testing.T, runs []*RunResult) mcpInstructionRunModes {
	t.Helper()
	modes := mcpInstructionRunModes{
		pairByMode: make(map[MCPServerInstructionMode]string),
		counts:     make(map[MCPServerInstructionMode]int),
		jobIDs:     make(map[string]bool),
	}
	for _, run := range runs {
		if run.Arm == ArmSemedit && run.SemanticIntegrationReflection == nil {
			t.Error("semedit mode result missing integration feedback")
		}
		mode := run.MCPServerInstructions
		modes.counts[mode]++
		if modes.counts[mode] == 1 {
			modes.pairByMode[mode] = run.ComparisonPairID
		} else if modes.pairByMode[mode] != run.ComparisonPairID {
			t.Errorf("mode %q has mismatched pair ids", mode)
		}
		if run.JobID == "" || modes.jobIDs[run.JobID] {
			t.Errorf("missing or duplicate job id %q", run.JobID)
		}
		modes.jobIDs[run.JobID] = true
	}
	return modes
}

func assertMCPInstructionModePairs(t *testing.T, modes mcpInstructionRunModes, wantModes []MCPServerInstructionMode) {
	t.Helper()
	if len(modes.pairByMode) != len(wantModes) {
		t.Errorf("mode pairs=%d, want %d", len(modes.pairByMode), len(wantModes))
	}
	for _, mode := range wantModes {
		if modes.counts[mode] != 2 || modes.pairByMode[mode] == "" {
			t.Errorf("mode %q has %d runs and pair %q, want two paired arms", mode, modes.counts[mode], modes.pairByMode[mode])
		}
		for _, other := range wantModes {
			if mode != other && modes.pairByMode[mode] != "" && modes.pairByMode[mode] == modes.pairByMode[other] {
				t.Errorf("modes %q and %q share a comparison pair", mode, other)
			}
		}
	}
}

func assertMCPInstructionPromptModes(t *testing.T, invocations []cliProviderInvocation) {
	t.Helper()
	descriptive := 0
	prescriptive := 0
	for _, invocation := range invocations {
		if invocation.Prompt == semanticIntegrationReflectionPrompt {
			continue
		}
		arguments := strings.Join(invocation.Args, " ")
		if strings.Contains(arguments, "mcp_servers.semedit.enabled=false") {
			continue
		}
		if strings.Contains(arguments, "Semedit semantic tools are available") {
			descriptive++
		}
		if strings.Contains(arguments, "Inspect the complete tool inventory") {
			prescriptive++
		}
	}
	if descriptive != 1 || prescriptive != 1 {
		t.Errorf("runtime instruction args: descriptive=%d prescriptive=%d, want 1 each", descriptive, prescriptive)
	}
}

type cliTrial struct {
	Output      string
	Invocations []cliProviderInvocation
	Report      BenchmarkReport
	Err         error
}

const cliPolicyFixture = "task_id: fixture-policy\ncategory: test\ninstruction: Add Done to main.go.\ncontexts:\n  - small\nprompt_variants:\n  default: Add Done to main.go.\noracle:\n  level_1_mutation_policy:\n    disallowed_files:\n      - go.mod\n  level_2_ast:\n    file: main.go\n    must_contain_symbols:\n      - Done\n  level_3_build:\n    clean_compile: false\n  level_4_test:\n    pass_tests: false\n\n-- go.mod --\nmodule example.com/policy\n\ngo 1.23\n-- main.go --\npackage main\nfunc main() {}\n"

type cliProviderInvocation struct {
	Args      []string
	Prompt    string
	Plan      string
	Directory string
}

type cliTrialWorkspace struct {
	root       string
	bin        string
	fixtures   string
	evidence   string
	outputPath string
	resultPath string
	executable string
}

type cliProcessOutput struct {
	output string
	err    error
}

func runCLITrial(t *testing.T, extra []string, malformed bool) cliTrial {
	t.Helper()
	workspace := prepareCLITrialWorkspace(t, malformed)
	process := runCLITrialProcess(t, workspace, extra)
	return readCLITrialResult(t, workspace, process)
}

func prepareCLITrialWorkspace(t *testing.T, malformed bool) cliTrialWorkspace {
	t.Helper()
	root := t.TempDir()
	bin := filepath.Join(root, "bin")
	fixtures := filepath.Join(root, "fixtures")
	evidence := filepath.Join(root, "evidence")
	for _, dir := range []string{bin, fixtures, evidence} {
		if err := os.MkdirAll(dir, 0o750); err != nil {
			t.Fatal(err)
		}
	}
	if err := pipeline.WriteAtomic(filepath.Join(fixtures, "fixture_policy.txtar"), []byte(cliPolicyFixture)); err != nil {
		t.Fatal(err)
	}
	if malformed {
		if err := pipeline.WriteAtomic(filepath.Join(fixtures, "broken.txtar"), []byte("not a benchmark")); err != nil {
			t.Fatal(err)
		}
	}
	executable, executableErr := os.Executable()
	if executableErr != nil {
		t.Fatal(executableErr)
	}
	script := "#!/bin/sh\nSEMEDIT_TEST_PROCESS=provider exec '" + strings.ReplaceAll(executable, "'", "'\\''") + "' -test.run=^TestBenchmarkProcessHelper$ -- \"$@\"\n"
	provider := filepath.Join(bin, "codex")
	if err := pipeline.WriteAtomic(provider, []byte(script)); err != nil {
		t.Fatal(err)
	}
	// #nosec G302 -- executable test double under the test's temporary directory.
	if err := os.Chmod(provider, 0o700); err != nil {
		t.Fatal(err)
	}
	return cliTrialWorkspace{
		root:       root,
		bin:        bin,
		fixtures:   fixtures,
		evidence:   evidence,
		outputPath: filepath.Join(root, "output.log"),
		resultPath: filepath.Join(root, "result.json"),
		executable: executable,
	}
}

func runCLITrialProcess(t *testing.T, workspace cliTrialWorkspace, extra []string) cliProcessOutput {
	t.Helper()
	output, outputErr := os.Create(workspace.outputPath)
	if outputErr != nil {
		t.Fatal(outputErr)
	}
	args := make([]string, 0, 13+len(extra))
	args = append(args, "-test.run=^TestBenchmarkProcessHelper$", "--", "--dir", workspace.fixtures, "--target", "codex", "--variants", "small", "--concurrency", "1", "--out-dir=", "--semedit-prewarm-verify=false", "--out-json", workspace.resultPath)
	args = append(args, extra...)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	// #nosec G204 -- execute this test binary with controlled arguments and a fake provider.
	command := exec.CommandContext(ctx, workspace.executable, args...)
	command.Dir = workspace.root
	command.Env = append(os.Environ(), "SEMEDIT_TEST_PROCESS=cli", "SEMEDIT_TEST_OUTPUT="+workspace.outputPath, "SEMEDIT_TEST_EVIDENCE="+workspace.evidence, "SEMEDIT_APP_SERVER_LOG="+filepath.Join(workspace.root, "app-server.jsonl"), "SEMEDIT_APP_SERVER_PROCESSES="+filepath.Join(workspace.root, "app-server.processes"), "PATH="+workspace.bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	command.Stdout = output
	command.Stderr = output
	runErr := command.Run()
	if closeErr := output.Close(); closeErr != nil {
		t.Fatal(closeErr)
	}
	data, readErr := os.ReadFile(workspace.outputPath)
	if readErr != nil {
		t.Fatal(readErr)
	}
	return cliProcessOutput{output: string(data), err: runErr}
}

func readCLITrialResult(t *testing.T, workspace cliTrialWorkspace, process cliProcessOutput) cliTrial {
	t.Helper()
	trial := cliTrial{Output: process.output, Err: process.err}
	trial.Invocations = readCLIProviderInvocations(t, workspace.evidence)
	data, readErr := os.ReadFile(workspace.resultPath)
	if readErr == nil {
		if err := json.Unmarshal(data, &trial.Report); err != nil {
			t.Fatal(err)
		}
	} else if !os.IsNotExist(readErr) {
		t.Fatal(readErr)
	}
	if process.err == nil && len(trial.Report.Runs) > 0 {
		assertCLIAppServerLifecycle(t, workspace, len(trial.Report.Runs), len(trial.Invocations))
	}
	return trial
}

func readCLIProviderInvocations(t *testing.T, evidence string) []cliProviderInvocation {
	t.Helper()
	logs, globErr := filepath.Glob(filepath.Join(evidence, "*.json"))
	if globErr != nil {
		t.Fatal(globErr)
	}
	invocations := make([]cliProviderInvocation, 0, len(logs))
	for _, log := range logs {
		data, err := os.ReadFile(log)
		if err != nil {
			t.Fatal(err)
		}
		var invocation cliProviderInvocation
		if err := json.Unmarshal(data, &invocation); err != nil {
			t.Fatal(err)
		}
		invocations = append(invocations, invocation)
	}
	return invocations
}

func assertCLIResultConditions(t *testing.T, trial cliTrial, resolved string) {
	t.Helper()
	for _, result := range trial.Report.Runs {
		if !result.Success || result.Error != "" {
			t.Errorf("failed result: %+v", result)
		}
		if string(result.SemeditArmRestrict) != resolved {
			t.Errorf("policy result=%s want %s", result.SemeditArmRestrict, resolved)
		}
		if result.JobID == "" || result.ComparisonPairID == "" || !strings.Contains(trial.Output, result.JobID) || !strings.Contains(trial.Output, result.ComparisonPairID) {
			t.Error("executed result identity differs from plan")
		}
		if result.SemeditArmRestrictionApplied != isSemeditArm(result.Arm) {
			t.Error("policy applicability wrong")
		}
	}
}

func assertCLIAppServerLifecycle(t *testing.T, workspace cliTrialWorkspace, wantSessions, wantTurns int) {
	t.Helper()
	processes, err := os.ReadFile(filepath.Join(workspace.root, "app-server.processes"))
	if err != nil {
		t.Fatalf("app-server process log: %v", err)
	}
	if got := strings.Count(string(processes), "process"); got != wantSessions {
		t.Fatalf("app-server processes = %d, want one per benchmark task; runs=%d", got, wantSessions)
	}
	data, err := os.ReadFile(filepath.Join(workspace.root, "app-server.jsonl"))
	if err != nil {
		t.Fatalf("app-server request log: %v", err)
	}
	requests := string(data)
	for _, method := range []string{"initialize", "thread/start"} {
		if got := strings.Count(requests, "\"method\":\""+method+"\""); got != wantSessions {
			t.Errorf("%s requests = %d, want one per benchmark task", method, got)
		}
	}
	if got := strings.Count(requests, "\"method\":\"turn/start\""); got != wantTurns {
		t.Errorf("turn/start requests = %d, want %d recorded turns", got, wantTurns)
	}
}
