// This file verifies the public arm policy values and their strict parsing contract.
package main

import "testing"

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
