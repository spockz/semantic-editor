// Package main implements target parsing and representation for benchmark harnesses.
package main

import (
	"fmt"
	"strings"
)

// Target models a benchmark execution target: harness, model, and reasoning effort.
type Target struct {
	Harness string `json:"harness"` // "codex", "agy", or "control"
	Model   string `json:"model,omitempty"`
	Effort  string `json:"effort,omitempty"` // "low", "medium", "high"
}

// String formats the target as canonical harness/model/effort or harness.
func (t Target) String() string {
	if t.Harness == "" {
		return "control"
	}
	parts := []string{t.Harness}
	if t.Model != "" {
		parts = append(parts, t.Model)
		if t.Effort != "" {
			parts = append(parts, t.Effort)
		}
	}
	return strings.Join(parts, "/")
}

// ParseTarget parses a target string into Target.
// Formats supported:
// - "harness" (e.g. "codex", "agy", "control")
// - "harness/model" (e.g. "agy/gemini-3.8-flash-high")
// - "harness/model/effort" (e.g. "codex/gpt-5.6-luna/high")
func ParseTarget(raw string) (Target, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return Target{}, fmt.Errorf("empty target specification")
	}

	parts := strings.Split(raw, "/")
	switch len(parts) {
	case 1:
		harness := strings.ToLower(parts[0])
		return Target{Harness: harness}, nil
	case 2:
		harness := strings.ToLower(parts[0])
		return Target{Harness: harness, Model: parts[1]}, nil
	case 3:
		harness := strings.ToLower(parts[0])
		return Target{Harness: harness, Model: parts[1], Effort: strings.ToLower(parts[2])}, nil
	default:
		return Target{}, fmt.Errorf("invalid target specification %q: expected harness[/model[/effort]]", raw)
	}
}

// TargetList implements flag.Value to allow repeatable --target flags.
type TargetList []Target

// String returns the comma-separated target strings.
func (tl *TargetList) String() string {
	if tl == nil || len(*tl) == 0 {
		return ""
	}
	items := make([]string, len(*tl))
	for i, t := range *tl {
		items[i] = t.String()
	}
	return strings.Join(items, ", ")
}

// Set appends or parses targets from the command-line flag value.
func (tl *TargetList) Set(value string) error {
	for raw := range strings.SplitSeq(value, ",") {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		target, err := ParseTarget(raw)
		if err != nil {
			return err
		}
		*tl = append(*tl, target)
	}
	return nil
}
