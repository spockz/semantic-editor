// This package isolates project verification discovery and execution from language backends so CLI and MCP callers can share one plan.

// Package projectverify discovers configured verification signals and runs ordered project hooks.
package projectverify

import (
	"context"
	"time"
)

// LSPRunner delegates configured language-server actions to the owning backend.
type LSPRunner func(context.Context, Hook, Scope) (HookResult, error)

// Report records the ordered outcome of one verification phase.
type Report struct {
	Root       string
	ConfigPath string
	Phase      Phase
	Scope      Scope
	Results    []HookResult
	Completed  bool
	Partial    bool
}

// HookResult is the receipt for one verification hook.
type HookResult struct {
	ID          string
	Kind        string
	Status      HookStatus
	ExitCode    int
	Stdout      string
	Stderr      string
	Truncated   bool
	Changed     []string
	Diagnostics []string
	StartedAt   time.Time
	FinishedAt  time.Time
	Error       string
	Signal      string
	ConfigPath  string
	ScopePaths  []string
}

// HookStatus describes whether a hook succeeded, failed, or was skipped.
type HookStatus string

// Plan contains discovered source files and configured phase hooks.
type Plan struct {
	Root       string
	ConfigPath string
	Sources    []Source
	Normalize  []Hook
	Checks     []Hook
}

// Hook describes one executable or language-server verification action.
type Hook struct {
	ID         string
	Languages  []string
	Disabled   bool
	Cwd        string
	Timeout    time.Duration
	Exec       []string
	Shell      *ShellSpec
	LSP        *LSPSpec
	Signal     string
	ConfigPath string
	ScopePaths []string
}

// LSPSpec identifies an explicit language-server action.
type LSPSpec struct {
	Language   string
	ActionKind string
	Command    string
	Arguments  any
}

// ShellSpec describes an explicit shell executable, its arguments, and script.
type ShellSpec struct {
	Executable string
	Args       []string
	Script     string
}

// Source identifies a recognized project source path and language.
type Source struct {
	Path     string
	Language string
}

// Scope selects project-relative paths for a verification phase.
type Scope struct {
	Paths []string
}

// PhaseNormalize and PhaseCheck identify supported verification phases.
const (
	PhaseNormalize Phase      = "normalize"
	PhaseCheck     Phase      = "check"
	HookSucceeded  HookStatus = "succeeded"
	HookFailed     HookStatus = "failed"
	HookSkipped    HookStatus = "skipped"
)

// Phase identifies whether normalization or checks are being run.
type Phase string
