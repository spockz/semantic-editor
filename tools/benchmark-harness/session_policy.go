// This file resolves semedit-arm restriction policy and task-specific prompts once so all agent turns use the same declared experiment condition.
package main

import (
	"fmt"
	"strings"
)

type AgentExecution struct {
	Task      *Task
	Target    Target
	Arm       ArmType
	Variant   string
	Prompt    string
	Followups []string
	Policy    SemeditArmRestriction
}

type SemeditArmRestriction string

const (
	SemeditArmRestrictRead      SemeditArmRestriction = "read"
	SemeditArmRestrictWrite     SemeditArmRestriction = "write"
	SemeditArmRestrictReadWrite SemeditArmRestriction = "readwrite"
)

func ParseSemeditArmRestriction(raw string) (SemeditArmRestriction, error) {
	policy := SemeditArmRestriction(raw)
	switch policy {
	case SemeditArmRestrictRead, SemeditArmRestrictWrite, SemeditArmRestrictReadWrite:
		return policy, nil
	default:
		return "", fmt.Errorf("unsupported semedit arm restriction %q (want read, write, or readwrite)", raw)
	}
}

func WithSemeditArmRestriction(policy SemeditArmRestriction) RunnerOption {
	return func(r *Runner) { r.semeditArmRestriction = policy }
}

func ResolveAgentExecution(task *Task, target Target, arm ArmType, variant string, policy SemeditArmRestriction) (AgentExecution, error) {
	if task == nil {
		return AgentExecution{}, fmt.Errorf("resolve agent execution: task is nil")
	}
	if policy == "" {
		policy = SemeditArmRestrictWrite
	}
	if _, err := ParseSemeditArmRestriction(string(policy)); err != nil {
		return AgentExecution{}, err
	}
	result := &RunResult{Target: target, Arm: arm}
	prompt, err := prepareAgentPromptWithPolicy(task, arm, variant, result, policy)
	if err != nil {
		return AgentExecution{}, err
	}
	followups := make([]string, 0, len(task.Metadata.InteractiveFollowups))
	for _, followup := range task.Metadata.InteractiveFollowups {
		followup = withMutationPolicyGuidance(followup, task, true)
		if arm == ArmSemedit {
			steering, err := semeditRestrictionSteering(policy)
			if err != nil {
				return AgentExecution{}, err
			}
			followup = strings.TrimSpace(followup + " " + steering + " When done, output DONE.")
		}
		followups = append(followups, followup)
	}
	return AgentExecution{Task: task, Target: target, Arm: arm, Variant: variant, Prompt: prompt, Followups: followups, Policy: policy}, nil
}

func semeditRestrictionSteering(policy SemeditArmRestriction) (string, error) {
	switch policy {
	case SemeditArmRestrictRead:
		return "Do not inspect source files with shell commands such as cat, sed, or rg. Use semantic or built-in code inspection tools. Shell commands for builds and tests are allowed.", nil
	case SemeditArmRestrictWrite:
		return "Use semedit semantic tools for supported source code modifications. Shell commands for builds and tests are allowed.", nil
	case SemeditArmRestrictReadWrite:
		return "Do not inspect source files with shell commands such as cat, sed, or rg. Use semantic or built-in code inspection tools, and use semedit semantic tools for supported source code modifications. Shell commands for builds and tests are allowed.", nil
	default:
		return "", fmt.Errorf("unsupported semedit arm restriction %q", policy)
	}
}

func promptVariantFromExecution(execution AgentExecution) string {
	if _, variant, ok := strings.Cut(execution.Variant, ":"); ok {
		return variant
	}
	return ""
}
