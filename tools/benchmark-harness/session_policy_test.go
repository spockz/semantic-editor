// This file verifies the public arm policy values and their strict parsing contract.
package main

import (
	"strings"
	"testing"
)

func TestParseSemeditArmRestrictionIsStrict(t *testing.T) {
	for _, value := range []SemeditArmRestriction{SemeditArmRestrictRead, SemeditArmRestrictWrite, SemeditArmRestrictReadWrite} {
		parsed, err := ParseSemeditArmRestriction(string(value))
		if err != nil || parsed != value {
			t.Fatalf("ParseSemeditArmRestriction(%q) = %q, %v", value, parsed, err)
		}
	}
	for _, value := range []string{"", "READ", "none", "write "} {
		if _, err := ParseSemeditArmRestriction(value); err == nil {
			t.Fatalf("ParseSemeditArmRestriction(%q) unexpectedly succeeded", value)
		}
	}
}

func TestNewRunnerDefaultsArmRestrictionToWrite(t *testing.T) {
	runner := NewRunner(t.TempDir())
	if runner.semeditArmRestriction != SemeditArmRestrictWrite {
		t.Fatalf("default restriction = %q, want write", runner.semeditArmRestriction)
	}
}

func TestSemeditRestrictionSteeringAndPrompts(t *testing.T) {
	task := &Task{Metadata: TaskMetadata{Instruction: "Inspect and fix the task.", InteractiveFollowups: []string{"Continue with the correction."}}}
	tests := []struct {
		name             string
		policy           SemeditArmRestriction
		expectedSteering string
	}{
		{
			name:             "read",
			policy:           SemeditArmRestrictRead,
			expectedSteering: "Do not inspect Go implementation files (non-test *.go) with shell commands such as cat, sed, or rg. Use semantic code inspection tools for those files. Reading project documentation (docs/*.md), build manifests (go.mod, Makefile), and visible test files in the task workspace is permitted. Shell commands for builds and tests are allowed.",
		},
		{
			name:             "write",
			policy:           SemeditArmRestrictWrite,
			expectedSteering: "Use semedit semantic tools for supported source code modifications. Shell commands for builds and tests are allowed.",
		},
		{
			name:             "readwrite",
			policy:           SemeditArmRestrictReadWrite,
			expectedSteering: "Do not inspect Go implementation files (non-test *.go) with shell commands such as cat, sed, or rg. Use semantic code inspection tools for those files. Reading project documentation (docs/*.md), build manifests (go.mod, Makefile), and visible test files in the task workspace is permitted. Use semedit semantic tools for supported source code modifications. Shell commands for builds and tests are allowed.",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			execution, err := ResolveAgentExecution(task, Target{}, ArmSemedit, "", test.policy)
			if err != nil {
				t.Fatalf("ResolveAgentExecution: %v", err)
			}
			wantPrompt := "Inspect and fix the task. " + test.expectedSteering + " When done, output DONE."
			if execution.Prompt != wantPrompt {
				t.Errorf("initial prompt = %q, want %q", execution.Prompt, wantPrompt)
			}
			if len(execution.Followups) != 1 {
				t.Fatalf("followups = %d, want 1", len(execution.Followups))
			}
			wantFollowup := "Continue with the correction. " + test.expectedSteering + " When done, output DONE."
			if execution.Followups[0] != wantFollowup {
				t.Errorf("follow-up prompt = %q, want %q", execution.Followups[0], wantFollowup)
			}
		})
	}
}

func TestResolveStagedFollowupPrompts(t *testing.T) {
	task := &Task{Metadata: TaskMetadata{
		Instruction:     "Rename the first method.",
		InteractiveMode: "staged",
		StagedFollowups: []StagedFollowup{
			{TotalEdits: 2, Instruction: "Rename the second method.", Oracle: OracleConfig{AST: ASTConfig{File: "pipeline/workflow.go"}}},
			{TotalEdits: 4, Instruction: "Rename the next two methods.", Oracle: OracleConfig{AST: ASTConfig{File: "pipeline/workflow.go"}}},
		},
	}}
	for _, arm := range []ArmType{ArmBaseline, ArmSemedit} {
		execution, err := ResolveAgentExecution(task, Target{}, arm, "", SemeditArmRestrictWrite)
		if err != nil {
			t.Fatalf("%s: %v", arm, err)
		}
		if len(execution.Followups) != 0 || len(execution.StagedFollowups) != 2 {
			t.Fatalf("%s: corrective=%d staged=%d", arm, len(execution.Followups), len(execution.StagedFollowups))
		}
		if execution.StagedFollowups[0].TotalEdits != 2 || execution.StagedFollowups[1].TotalEdits != 4 {
			t.Fatalf("%s: stage totals = %+v", arm, execution.StagedFollowups)
		}
		if !strings.Contains(execution.StagedFollowups[0].Instruction, "Rename the second method.") {
			t.Fatalf("%s: missing stage instruction: %q", arm, execution.StagedFollowups[0].Instruction)
		}
		switch arm {
		case ArmBaseline:
			if !strings.Contains(execution.StagedFollowups[0].Instruction, "Do not use semantic editing MCP tools") {
				t.Fatalf("baseline stage lacks arm steering: %q", execution.StagedFollowups[0].Instruction)
			}
		case ArmSemedit:
			if !strings.Contains(execution.StagedFollowups[0].Instruction, "Use semedit semantic tools") {
				t.Fatalf("semedit stage lacks arm steering: %q", execution.StagedFollowups[0].Instruction)
			}
		default:
			t.Fatalf("unexpected arm %s", arm)
		}
	}
}
