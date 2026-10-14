// Package maven runs bounded, explicitly trusted Maven verification actions.
package maven

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/spockz/semantic-editor/internal/backend"
)

const (
	captureLimit     = 64 << 10
	logTailLimit     = 4 << 10
	executionTimeout = 2 * time.Minute
)

var (
	// ErrTrustRequired indicates missing request-scoped trust.
	ErrTrustRequired = errors.New("workspace trust required for Maven execution")
	// ErrRootPOMRequired indicates the root pom.xml is missing.
	ErrRootPOMRequired = errors.New("root Maven pom.xml is required")
	// ErrRootOutsideTrust indicates that the selected root escapes the trusted workspace.
	ErrRootOutsideTrust = errors.New("maven root is outside the trusted workspace")
	// ErrRootPOMOutsideTrust indicates that a root pom symlink escapes its workspace.
	ErrRootPOMOutsideTrust = errors.New("root Maven pom.xml is outside the trusted workspace")
	// ErrUnsafeWrapper indicates a wrapper escaping the root.
	ErrUnsafeWrapper = errors.New("maven wrapper must resolve within the workspace root")
	// ErrToolUnavailable indicates the selected binary cannot be executed.
	ErrToolUnavailable = errors.New("maven executable is unavailable")
)

// Tool selects the Maven launcher.
type Tool string

const (
	// ToolAuto selects the root wrapper when available.
	ToolAuto Tool = "auto"
	// ToolWrapper selects mvnw.
	ToolWrapper Tool = "wrapper"
	// ToolSystem selects a system Maven executable.
	ToolSystem Tool = "system"
)

// Request contains one bounded Maven invocation.
type Request struct {
	Root      string
	Trust     backend.WorkspaceTrust
	Tool      Tool
	SystemBin string
	// AllowNetwork opts into Maven network access. The zero value is isolated.
	AllowNetwork bool
	Goal         string
}

// Result reports bounded Maven process output.
type Result struct {
	Goal            string   `json:"goal"`
	Tool            string   `json:"tool"`
	Command         []string `json:"command"`
	ExitCode        int      `json:"exit_code"`
	Stdout          string   `json:"stdout,omitempty"`
	Stderr          string   `json:"stderr,omitempty"`
	StdoutTruncated bool     `json:"stdout_truncated,omitempty"`
	StderrTruncated bool     `json:"stderr_truncated,omitempty"`
	LogTail         string   `json:"log_tail,omitempty"`
	Error           string   `json:"error,omitempty"`
	DurationMS      int64    `json:"duration_ms"`
}

// Error identifies a typed Maven runner failure.
type Error struct {
	Kind   string
	Err    error
	Result *Result
}

func (e *Error) Error() string { return e.Kind + ": " + e.Err.Error() }
func (e *Error) Unwrap() error { return e.Err }

// MavenResult returns the process result captured before a Maven error.
func (e *Error) MavenResult() *Result { return e.Result }

type boundedBuffer struct {
	b         bytes.Buffer
	truncated bool
}

type cancellationResult struct {
	observed bool
	err      error
}

type processResult struct {
	stdout       boundedBuffer
	stderr       boundedBuffer
	err          error
	cancellation cancellationResult
	durationMS   int64
}

func (b *boundedBuffer) Write(p []byte) (int, error) {
	length := len(p)
	if b.b.Len() < captureLimit {
		n := captureLimit - b.b.Len()
		if len(p) > n {
			p = p[:n]
			b.truncated = true
		}
		_, _ = b.b.Write(p)
	} else {
		b.truncated = true
	}
	return length, nil
}
func (b *boundedBuffer) String() string {
	s := b.b.String()
	if b.truncated {
		s += "\n[output truncated]"
	}
	return s
}

// Run executes one fixed Maven goal without a shell.
func Run(ctx context.Context, req Request) (Result, error) {
	root, pom, err := validateRunRequest(req)
	if err != nil {
		return Result{}, err
	}
	if mkdirErr := os.MkdirAll(filepath.Join(root, ".scratch"), 0o700); mkdirErr != nil {
		return Result{}, &Error{Kind: "scratch", Err: mkdirErr}
	}
	if req.Tool == "" {
		req.Tool = ToolAuto
	}
	bin, selected, err := resolveTool(root, req.Tool, req.SystemBin)
	if err != nil {
		return Result{}, err
	}
	home, err := os.MkdirTemp(filepath.Join(root, ".scratch"), "maven-home-")
	if err != nil {
		return Result{}, &Error{Kind: "scratch", Err: err}
	}
	defer func() { _ = os.RemoveAll(home) }()
	repo, err := os.MkdirTemp(filepath.Join(root, ".scratch"), "maven-repo-")
	if err != nil {
		return Result{}, &Error{Kind: "scratch", Err: err}
	}
	defer func() { _ = os.RemoveAll(repo) }()
	args := []string{"--batch-mode", "--no-transfer-progress", "-f", pom}
	if !req.AllowNetwork {
		args = append(args, "-o")
	}
	args = append(args, "-Dmaven.repo.local="+repo, req.Goal)
	if deadline, ok := ctx.Deadline(); !ok || time.Until(deadline) > executionTimeout {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, executionTimeout)
		defer cancel()
	}
	process := runMavenProcess(ctx, bin, args, root, home)
	return processResultForRequest(ctx, req, bin, selected, args, process)
}

func processResultForRequest(ctx context.Context, req Request, bin, selected string, args []string, process processResult) (Result, error) {
	err := process.err
	stdout, stderr := process.stdout.String(), process.stderr.String()
	result := Result{Goal: req.Goal, Tool: selected, Command: append([]string{bin}, args...), Stdout: stdout, Stderr: stderr, StdoutTruncated: process.stdout.truncated, StderrTruncated: process.stderr.truncated, LogTail: logTail(stdout, stderr), DurationMS: process.durationMS}
	if err != nil {
		result.ExitCode = 1
		if exit, ok := errors.AsType[*exec.ExitError](err); ok {
			result.ExitCode = exit.ExitCode()
		}
	}
	if process.cancellation.observed {
		ctxErr := ctx.Err()
		if ctxErr == nil {
			ctxErr = context.Canceled
		}
		result.Error = ctxErr.Error()
		cancelCause := errors.Join(ctxErr, process.cancellation.err)
		if process.cancellation.err != nil {
			result.Error += ": " + process.cancellation.err.Error()
		}
		return result, &Error{Kind: "Maven canceled", Err: cancelCause, Result: &result}
	}
	if err == nil {
		result.ExitCode = 0
		return result, nil
	}
	if _, ok := errors.AsType[*exec.ExitError](err); ok {
		result.Error = fmt.Sprintf("exit code %d", result.ExitCode)
		return result, &Error{Kind: "Maven failed", Err: fmt.Errorf("exit code %d: %w", result.ExitCode, err), Result: &result}
	}
	result.Error = err.Error()
	return result, &Error{Kind: "Maven execution", Err: err, Result: &result}
}

func runMavenProcess(ctx context.Context, bin string, args []string, root, home string) processResult {
	//nolint:noctx // Cancellation must kill the process group until Wait drains inherited output pipes.
	cmd := exec.Command(bin, args...) // #nosec G204 -- validated wrapper/system executable.
	cmd.Dir = root
	cmd.Env = isolatedEnvironment(home)
	var stdout, stderr boundedBuffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	configureMavenProcess(cmd)
	started := time.Now()
	cancellation := cancellationResult{}
	err := ctx.Err()
	if err == nil {
		err = cmd.Start()
	} else {
		cancellation.observed = true
	}
	if err == nil {
		processDone := make(chan struct{})
		watchDone := make(chan cancellationResult, 1)
		var processMu sync.Mutex
		finished := false
		go func() {
			select {
			case <-ctx.Done():
				processMu.Lock()
				result := cancellationResult{}
				if !finished {
					result = cancellationResult{observed: true, err: stopMavenProcess(cmd)}
				}
				processMu.Unlock()
				watchDone <- result
			case <-processDone:
				watchDone <- cancellationResult{}
			}
		}()
		err = cmd.Wait()
		processMu.Lock()
		finished = true
		processMu.Unlock()
		close(processDone)
		cancellation = <-watchDone
	}
	return processResult{stdout: stdout, stderr: stderr, err: err, cancellation: cancellation, durationMS: time.Since(started).Milliseconds()}
}

func validateRunRequest(req Request) (string, string, error) {
	root := backend.CanonicalWorkspaceRoot(req.Root)
	trustRoot := backend.CanonicalWorkspaceRoot(req.Trust.WorkspaceRoot)
	if !req.Trust.Trusted || trustRoot == "" || !within(trustRoot, root) {
		return "", "", &Error{Kind: "trust", Err: fmt.Errorf("%w: %w", ErrTrustRequired, ErrRootOutsideTrust)}
	}
	if req.Goal != "test-compile" && req.Goal != "test" {
		return "", "", &Error{Kind: "goal", Err: fmt.Errorf("unsupported Maven goal %q", req.Goal)}
	}
	pom := filepath.Join(root, "pom.xml")
	canonicalPOM, resolveErr := filepath.EvalSymlinks(pom)
	if resolveErr != nil {
		return "", "", &Error{Kind: "root pom", Err: ErrRootPOMRequired}
	}
	canonicalPOM = filepath.Clean(canonicalPOM)
	if !within(root, canonicalPOM) {
		return "", "", &Error{Kind: "root pom", Err: ErrRootPOMOutsideTrust}
	}
	if info, statErr := os.Stat(canonicalPOM); statErr != nil || info.IsDir() {
		return "", "", &Error{Kind: "root pom", Err: ErrRootPOMRequired}
	}
	return root, canonicalPOM, nil
}

func isolatedEnvironment(home string) []string {
	blocked := map[string]bool{"HOME": true, "USERPROFILE": true, "MAVEN_USER_HOME": true, "MAVEN_CONFIG": true, "M2_HOME": true, "M2": true, "MAVEN_OPTS": true, "MAVEN_ARGS": true, "MAVEN_ARGS_APPEND": true, "MAVEN_PROJECTBASEDIR": true, "JAVA_TOOL_OPTIONS": true, "_JAVA_OPTIONS": true}
	original := os.Environ()
	env := make([]string, 0, len(original)+3)
	for _, value := range original {
		key, _, ok := strings.Cut(value, "=")
		if !ok || blocked[key] {
			continue
		}
		env = append(env, value)
	}
	env = append(env, "HOME="+home, "USERPROFILE="+home, "MAVEN_USER_HOME="+home, "MAVEN_CONFIG="+home)
	return env
}

func logTail(stdout, stderr string) string {
	combined := strings.TrimSpace(stdout)
	if stderr != "" {
		if combined != "" {
			combined += "\n"
		}
		combined += strings.TrimSpace(stderr)
	}
	if len(combined) <= logTailLimit {
		return combined
	}
	return combined[len(combined)-logTailLimit:]
}

func resolveTool(root string, requested Tool, system string) (string, string, error) {
	wrapper := filepath.Join(root, "mvnw")
	if requested == ToolAuto {
		if _, err := os.Stat(wrapper); err == nil {
			requested = ToolWrapper
		} else {
			requested = ToolSystem
		}
	}
	if requested == ToolWrapper {
		resolved, err := filepath.EvalSymlinks(wrapper)
		if err != nil {
			return "", "", &Error{Kind: "wrapper", Err: fmt.Errorf("%w: %w", ErrUnsafeWrapper, err)}
		}
		if !within(root, resolved) {
			return "", "", &Error{Kind: "wrapper", Err: ErrUnsafeWrapper}
		}
		if info, err := os.Stat(resolved); err != nil || info.IsDir() || info.Mode()&0o111 == 0 {
			return "", "", &Error{Kind: "wrapper", Err: ErrToolUnavailable}
		}
		return resolved, string(ToolWrapper), nil
	}
	if requested != ToolSystem {
		return "", "", &Error{Kind: "tool", Err: fmt.Errorf("unknown Maven tool %q", requested)}
	}
	if strings.TrimSpace(system) != "" {
		if !filepath.IsAbs(system) {
			return "", "", &Error{Kind: "system", Err: fmt.Errorf("system Maven binary must be absolute: %w", ErrToolUnavailable)}
		}
		resolved, err := filepath.EvalSymlinks(system)
		if err != nil {
			return "", "", &Error{Kind: "system", Err: fmt.Errorf("%w: %w", ErrToolUnavailable, err)}
		}
		info, err := os.Stat(resolved)
		if err != nil || info.IsDir() || info.Mode()&0o111 == 0 {
			return "", "", &Error{Kind: "system", Err: ErrToolUnavailable}
		}
		return resolved, string(ToolSystem), nil
	}
	path, err := exec.LookPath("mvn")
	if err != nil {
		return "", "", &Error{Kind: "system", Err: fmt.Errorf("%w: %w", ErrToolUnavailable, err)}
	}
	return path, string(ToolSystem), nil
}
func within(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

var _ io.Writer = (*boundedBuffer)(nil)
