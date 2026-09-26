// These process-boundary tests ensure the public CLI executes only its printed plan and isolates arm policies.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"semedit/internal/pipeline"
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
			flags := []string{}
			resolved := policy
			if policy == "default" {
				resolved = "write"
			} else {
				flags = append(flags, "--semedit-arm-restrict="+policy)
			}
			trial := runCLITrial(t, flags, false)
			if trial.Err != nil {
				t.Fatalf("CLI failed: %v\n%s", trial.Err, trial.Output)
			}
			if len(trial.Invocations) != 2 || len(trial.Report.Runs) != 2 {
				t.Fatalf("invocations=%d results=%d, want paired jobs\n%s", len(trial.Invocations), len(trial.Report.Runs), trial.Output)
			}
			for _, invocation := range trial.Invocations {
				if !strings.Contains(invocation.Plan, "Resolved benchmark plan: 2 jobs") || !strings.Contains(invocation.Plan, "arm=baseline-diff") || !strings.Contains(invocation.Plan, "arm=semedit") {
					t.Errorf("provider started before complete plan:\n%s", invocation.Plan)
				}
				baseline := slices.Contains(invocation.Args, "mcp_servers.semedit.enabled=false")
				if baseline {
					if baselinePrompt == "" {
						baselinePrompt = invocation.Prompt
					}
					if invocation.Prompt != baselinePrompt {
						t.Errorf("policy %s altered baseline prompt", policy)
					}
					if strings.Contains(invocation.Prompt, "Do not inspect source") || strings.Contains(invocation.Prompt, "Shell commands for builds") {
						t.Errorf("baseline received policy steering: %s", invocation.Prompt)
					}
				} else {
					if !slices.Contains(invocation.Args, "mcp_servers.semedit.enabled=true") {
						t.Error("semedit MCP not enabled")
					}
					read := strings.Contains(invocation.Prompt, "Do not inspect source")
					write := strings.Contains(invocation.Prompt, "supported source code modifications")
					if read != (resolved != "write") || write != (resolved != "read") {
						t.Errorf("policy %s wrong steering: %s", policy, invocation.Prompt)
					}
					if !strings.Contains(invocation.Prompt, "Shell commands for builds and tests are allowed") {
						t.Error("shell build/test permission missing")
					}
				}
			}
			assertCLIResultConditions(t, trial, resolved)
			if trial.Report.Runs[0].ComparisonPairID != trial.Report.Runs[1].ComparisonPairID {
				t.Error("agent arms not paired")
			}
			if !strings.Contains(trial.Output, "[2/2 completed · 0 remaining]") {
				t.Error("completion accounting missing")
			}
		})
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

func TestCLIControlSharesSchedulerAndRetainsFailure(t *testing.T) {
	trial := runCLITrial(t, []string{"--target=control"}, false)
	if trial.Err == nil {
		t.Error("unsupported control task should return an error")
	}
	if len(trial.Report.Runs) != 3 {
		t.Fatalf("results=%d, want two agent arms and failed control\n%s", len(trial.Report.Runs), trial.Output)
	}
	if len(trial.Invocations) != 2 {
		t.Errorf("provider invocations=%d, control must not invoke one", len(trial.Invocations))
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

func runCLITrial(t *testing.T, extra []string, malformed bool) cliTrial {
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
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	script := "#!/bin/sh\nSEMEDIT_TEST_PROCESS=provider exec '" + strings.ReplaceAll(exe, "'", "'\\''") + "' -test.run=^TestBenchmarkProcessHelper$ -- \"$@\"\n"
	provider := filepath.Join(bin, "codex")
	if err := pipeline.WriteAtomic(provider, []byte(script)); err != nil {
		t.Fatal(err)
	}
	// #nosec G302 -- executable test double under the test's temporary directory.
	if err := os.Chmod(provider, 0o700); err != nil {
		t.Fatal(err)
	}
	outputPath := filepath.Join(root, "output.log")
	output, err := os.Create(outputPath)
	if err != nil {
		t.Fatal(err)
	}
	resultPath := filepath.Join(root, "result.json")
	args := make([]string, 0, 13+len(extra))
	args = append(args, "-test.run=^TestBenchmarkProcessHelper$", "--", "--dir", fixtures, "--target", "codex", "--variants", "small", "--concurrency", "1", "--out-dir=", "--out-json", resultPath)
	args = append(args, extra...)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	// #nosec G204 -- execute this test binary with controlled arguments and a fake provider.
	command := exec.CommandContext(ctx, exe, args...)
	command.Dir = root
	command.Env = append(os.Environ(), "SEMEDIT_TEST_PROCESS=cli", "SEMEDIT_TEST_OUTPUT="+outputPath, "SEMEDIT_TEST_EVIDENCE="+evidence, "PATH="+bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	command.Stdout = output
	command.Stderr = output
	runErr := command.Run()
	if err := output.Close(); err != nil {
		t.Fatal(err)
	}
	out, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatal(err)
	}
	trial := cliTrial{Output: string(out), Err: runErr}
	logs, err := filepath.Glob(filepath.Join(evidence, "*.json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, log := range logs {
		data, err := os.ReadFile(log)
		if err != nil {
			t.Fatal(err)
		}
		var invocation cliProviderInvocation
		if err := json.Unmarshal(data, &invocation); err != nil {
			t.Fatal(err)
		}
		trial.Invocations = append(trial.Invocations, invocation)
	}
	data, readErr := os.ReadFile(resultPath)
	if readErr == nil {
		if err := json.Unmarshal(data, &trial.Report); err != nil {
			t.Fatal(err)
		}
	} else if !os.IsNotExist(readErr) {
		t.Fatal(readErr)
	}
	return trial
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
		if result.SemeditArmRestrictionApplied != (result.Arm == ArmSemedit) {
			t.Error("policy applicability wrong")
		}
	}
}
