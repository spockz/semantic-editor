// external_repo.go hydrates pinned benchmark repositories before timing begins.
package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

var repoCacheLocks sync.Map

func hydrateExternalRepository(ctx context.Context, task *Task, scratchRoot, workDir string) error {
	repository := task.Metadata.Repository
	if repository.URL == "" || repository.BaseCommit == "" {
		return fmt.Errorf("external repository metadata requires URL and base commit")
	}
	name := strings.TrimSuffix(filepath.Base(repository.URL), ".git")
	cacheDir := filepath.Join(scratchRoot, "cache", "repos", name, repository.BaseCommit)
	if err := ensureRepositoryCache(ctx, repository.URL, repository.BaseCommit, cacheDir); err != nil {
		return err
	}
	if err := copyRepositoryCache(ctx, cacheDir, workDir, repository.BaseCommit); err != nil {
		return fmt.Errorf("hydrate benchmark workspace from repository cache: %w", err)
	}
	return nil
}

func ensureRepositoryCache(ctx context.Context, repoURL, baseCommit, cacheDir string) (retErr error) {
	lockValue, _ := repoCacheLocks.LoadOrStore(cacheDir, &sync.Mutex{})
	lock := lockValue.(*sync.Mutex)
	lock.Lock()
	defer lock.Unlock()

	gitDir := filepath.Join(cacheDir, ".git")
	info, err := os.Stat(gitDir)
	if err != nil {
		return fmt.Errorf("inspect prechecked repository cache %s for %s: %w", cacheDir, repoURL, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("prechecked repository cache %s is not a standalone checkout", cacheDir)
	}
	treeOutput, treeErr := runGit(ctx, cacheDir, "cat-file", "-e", baseCommit+"^{tree}")
	if treeErr != nil {
		return fmt.Errorf("pinned benchmark commit %s is missing from prechecked cache %s: %w: %s", baseCommit, cacheDir, treeErr, strings.TrimSpace(string(treeOutput)))
	}
	checkoutOutput, checkoutErr := runGit(ctx, cacheDir, "checkout", "--detach", "-f", baseCommit)
	if checkoutErr != nil {
		return fmt.Errorf("checkout benchmark base commit %s from local cache: %w: %s", baseCommit, checkoutErr, strings.TrimSpace(string(checkoutOutput)))
	}
	remotes, err := runGit(ctx, cacheDir, "remote")
	if err != nil {
		return fmt.Errorf("list cached repository remotes: %w", err)
	}
	for remote := range strings.FieldsSeq(string(remotes)) {
		output, removeErr := runGit(ctx, cacheDir, "remote", "remove", remote)
		if removeErr != nil {
			return fmt.Errorf("disable remote %s in benchmark cache: %w: %s", remote, removeErr, strings.TrimSpace(string(output)))
		}
	}
	return nil
}

func copyRepositoryCache(ctx context.Context, cacheDir, workDir, baseCommit string) error {
	source := cacheDir + string(filepath.Separator) + "."
	var copyArgs []string
	switch runtime.GOOS {
	case "darwin":
		copyArgs = []string{"-cR", source, workDir}
	case "linux":
		copyArgs = []string{"-a", "--reflink=auto", source, workDir}
	default:
		copyArgs = []string{"-R", source, workDir}
	}
	output, err := runCommand(ctx, "cp", "", copyArgs...)
	if err != nil {
		if ctx.Err() != nil {
			return fmt.Errorf("copy checked-out repository into fixture: %w", ctx.Err())
		}
		return fmt.Errorf("copy checked-out repository into fixture: %w: %s", err, strings.TrimSpace(string(output)))
	}
	head, err := runGit(ctx, workDir, "rev-parse", "--verify", "HEAD")
	if err != nil {
		return fmt.Errorf("read copied repository checkout: %w: %s", err, strings.TrimSpace(string(head)))
	}
	if strings.TrimSpace(string(head)) != baseCommit {
		return fmt.Errorf("copied repository HEAD %s does not match pinned commit %s", strings.TrimSpace(string(head)), baseCommit)
	}
	remotes, err := runGit(ctx, workDir, "remote")
	if err != nil {
		return fmt.Errorf("inspect copied repository remotes: %w", err)
	}
	if names := strings.Fields(string(remotes)); len(names) > 0 {
		return fmt.Errorf("copied repository has configured remotes: %s", strings.Join(names, ", "))
	}
	return nil
}

func applyRepositoryPatch(ctx context.Context, workDir string, patch []byte, description string) error {
	if len(patch) == 0 {
		return nil
	}
	root, err := filepath.Abs(workDir)
	if err != nil {
		return fmt.Errorf("resolve %s workspace: %w", description, err)
	}
	env := make([]string, 0, len(os.Environ())+2)
	for _, entry := range os.Environ() {
		key, _, _ := strings.Cut(entry, "=")
		if !strings.HasPrefix(key, "GIT_") && key != "LC_ALL" {
			env = append(env, entry)
		}
	}
	env = append(env, "GIT_CEILING_DIRECTORIES="+filepath.Dir(root), "LC_ALL=C")
	stages := []struct {
		name string
		args []string
	}{
		{name: "apply", args: []string{"apply", "--verbose", "--whitespace=nowarn", "-"}},
		{name: "verify applied", args: []string{"apply", "--verbose", "--reverse", "--check", "--whitespace=nowarn", "-"}},
	}
	for _, stage := range stages {
		// #nosec G204 -- executable and arguments are fixed; fixture patch is provided over stdin.
		cmd := exec.CommandContext(ctx, "git", stage.args...)
		cmd.Dir = root
		cmd.Env = env
		cmd.Stdin = bytes.NewReader(patch)
		output, runErr := cmd.CombinedOutput()
		if runErr != nil {
			return fmt.Errorf("%s %s: %w: %s", stage.name, description, runErr, strings.TrimSpace(string(output)))
		}
		if bytes.Contains(output, []byte("Skipped patch ")) {
			return fmt.Errorf("%s %s: git skipped patch despite exit status 0: %s", stage.name, description, strings.TrimSpace(string(output)))
		}
	}
	return nil
}

func runGit(ctx context.Context, workDir string, args ...string) ([]byte, error) {
	return runCommand(ctx, "git", workDir, args...)
}

func runCommand(ctx context.Context, name, workDir string, args ...string) ([]byte, error) {
	// #nosec G204 -- callers use fixed git/cp executables and harness-built arguments.
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = workDir
	output, err := cmd.CombinedOutput()
	return output, err
}

func (r *Runner) ExecutePatchReplay(ctx context.Context, task *Task, target Target, variant string) (*RunResult, error) {
	if task == nil {
		return nil, fmt.Errorf("execute patch replay: task is nil")
	}
	if task.Metadata.Repository.URL == "" || len(task.FixPatch) == 0 {
		return nil, fmt.Errorf("execute patch replay: external repository and fix patch are required")
	}
	runID := fmt.Sprintf("run_patch_replay_%s_%d", safePathFragment(task.Metadata.TaskID), time.Now().UnixNano())
	workDir := filepath.Join(r.baseScratchDir, runID)
	if err := os.MkdirAll(workDir, 0o750); err != nil {
		return nil, fmt.Errorf("create patch replay workspace: %w", err)
	}
	defer func() {
		_ = os.RemoveAll(workDir)
	}()
	if err := hydrateExternalRepository(ctx, task, r.baseScratchDir, workDir); err != nil {
		return nil, fmt.Errorf("hydrate patch replay workspace: %w", err)
	}
	if err := task.ExtractVariantTo(workDir, variant); err != nil {
		return nil, fmt.Errorf("extract patch replay fixture metadata: %w", err)
	}
	started := time.Now()
	result := &RunResult{
		TaskID: task.Metadata.TaskID, Target: target, Arm: ArmSemeditExplicit,
		Prompt: task.Metadata.Instruction, ExecutionMode: "ground_truth_fix_patch_replay",
		MCPServerInstructions: MCPServerInstructionsNone, Provenance: r.provenanceFor(),
	}
	if err := applyRepositoryPatch(ctx, workDir, task.FixPatch, "ground-truth fix patch"); err != nil {
		result.Error = err.Error()
		result.WallClock = time.Since(started)
		return result, err
	}
	changed, err := runGit(ctx, workDir, "diff", "--name-only", task.Metadata.Repository.BaseCommit)
	if err != nil {
		return nil, fmt.Errorf("list patch replay changes: %w: %s", err, strings.TrimSpace(string(changed)))
	}
	modifiedFiles := strings.Fields(string(changed))
	diff, err := runGit(ctx, workDir, "diff", "--no-ext-diff", task.Metadata.Repository.BaseCommit)
	if err != nil {
		return nil, fmt.Errorf("capture patch replay diff: %w: %s", err, strings.TrimSpace(string(diff)))
	}
	result.Diff = string(diff)
	oracle, err := Evaluate(ctx, task, workDir, modifiedFiles)
	result.WallClock = time.Since(started)
	if err != nil {
		result.Error = fmt.Sprintf("oracle evaluation: %v", err)
		return result, fmt.Errorf("oracle evaluation: %w", err)
	}
	result.Oracle = oracle
	result.Success = oracle.Passed
	if !oracle.Passed {
		result.Error = oracle.ErrorMessage
	}
	return result, nil
}
