// external_repo.go hydrates pinned benchmark repositories before timing begins.
package main

import (
	"bytes"
	"context"
	"errors"
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
	cacheDir := filepath.Join(scratchRoot, "cache", "repos", name)
	if err := ensureRepositoryCache(ctx, repository.URL, repository.BaseCommit, cacheDir); err != nil {
		return err
	}
	if err := copyRepositoryCache(ctx, cacheDir, workDir, repository.BaseCommit); err != nil {
		return fmt.Errorf("hydrate benchmark workspace from repository cache: %w", err)
	}
	if output, err := runGit(ctx, workDir, "checkout", "--detach", "-f", repository.BaseCommit); err != nil {
		return fmt.Errorf("checkout benchmark base commit %s: %w: %s", repository.BaseCommit, err, strings.TrimSpace(string(output)))
	}
	return nil
}

func ensureRepositoryCache(ctx context.Context, repoURL, baseCommit, cacheDir string) (retErr error) {
	lockValue, _ := repoCacheLocks.LoadOrStore(cacheDir, &sync.Mutex{})
	lock := lockValue.(*sync.Mutex)
	lock.Lock()
	defer lock.Unlock()
	_, statErr := os.Stat(filepath.Join(cacheDir, ".git"))
	if os.IsNotExist(statErr) {
		if parentErr := os.MkdirAll(filepath.Dir(cacheDir), 0o750); parentErr != nil {
			return fmt.Errorf("create repository cache directory: %w", parentErr)
		}
		if removeErr := os.RemoveAll(cacheDir); removeErr != nil {
			return fmt.Errorf("remove incomplete repository cache %s: %w", cacheDir, removeErr)
		}
		stagingRoot, createErr := os.MkdirTemp(filepath.Dir(cacheDir), ".repo-cache-*")
		if createErr != nil {
			return fmt.Errorf("create staging directory for repository cache: %w", createErr)
		}
		defer func() {
			if err := os.RemoveAll(stagingRoot); err != nil {
				retErr = errors.Join(retErr, fmt.Errorf("remove repository cache staging directory: %w", err))
			}
		}()
		stagingCache := filepath.Join(stagingRoot, "repo")
		cloneArgs := []string{"clone", "--no-checkout"}
		if remoteRepositoryURL(repoURL) {
			cloneArgs = append(cloneArgs, "--filter=blob:none")
		}
		cloneArgs = append(cloneArgs, repoURL, stagingCache)
		if output, cloneErr := runGit(ctx, "", cloneArgs...); cloneErr != nil {
			return fmt.Errorf("clone benchmark repository %s: %w: %s", repoURL, cloneErr, strings.TrimSpace(string(output)))
		}
		if renameErr := os.Rename(stagingCache, cacheDir); renameErr != nil {
			return fmt.Errorf("publish repository cache %s: %w", cacheDir, renameErr)
		}
	} else if statErr != nil {
		return fmt.Errorf("inspect repository cache %s: %w", cacheDir, statErr)
	}
	if _, err := runGit(ctx, cacheDir, "cat-file", "-e", baseCommit+"^{tree}"); err == nil {
		return nil
	}
	fetchArgs := []string{"fetch"}
	if remoteRepositoryURL(repoURL) {
		fetchArgs = append(fetchArgs, "--filter=blob:none")
	}
	fetchArgs = append(fetchArgs, "--depth=1", "origin", baseCommit)
	if output, err := runGit(ctx, cacheDir, fetchArgs...); err != nil {
		return fmt.Errorf("fetch benchmark base commit %s: %w: %s", baseCommit, err, strings.TrimSpace(string(output)))
	}
	if output, err := runGit(ctx, cacheDir, "cat-file", "-e", baseCommit+"^{tree}"); err != nil {
		return fmt.Errorf("verify cached benchmark commit %s: %w: %s", baseCommit, err, strings.TrimSpace(string(output)))
	}
	return nil
}

func remoteRepositoryURL(repoURL string) bool {
	return strings.HasPrefix(repoURL, "https://") || strings.HasPrefix(repoURL, "http://") || strings.HasPrefix(repoURL, "ssh://") || strings.HasPrefix(repoURL, "git@")
}

func copyRepositoryCache(ctx context.Context, cacheDir, workDir, baseCommit string) error {
	var cloneArgs []string
	switch runtime.GOOS {
	case "darwin":
		cloneArgs = []string{"-cR", filepath.Join(cacheDir, "."), workDir}
	case "linux":
		cloneArgs = []string{"-a", "--reflink=auto", filepath.Join(cacheDir, "."), workDir}
	default:
		cloneArgs = []string{"-R", filepath.Join(cacheDir, "."), workDir}
	}
	if _, copyErr := runCommand(ctx, "cp", "", cloneArgs...); copyErr != nil && ctx.Err() != nil {
		return fmt.Errorf("copy-on-write repository clone: %w", ctx.Err())
	}
	if _, err := runGit(ctx, workDir, "cat-file", "-e", baseCommit+"^{tree}"); err == nil {
		return nil
	}
	if err := os.RemoveAll(workDir); err != nil {
		return fmt.Errorf("remove incomplete copy-on-write repository clone: %w", err)
	}
	if output, err := runGit(ctx, "", "clone", "--shared", "--no-checkout", cacheDir, workDir); err != nil {
		return fmt.Errorf("clone repository cache into workspace: %w: %s", err, strings.TrimSpace(string(output)))
	}
	if output, err := runGit(ctx, workDir, "cat-file", "-e", baseCommit+"^{tree}"); err != nil {
		return fmt.Errorf("verify fallback repository clone: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

func applyRepositoryPatch(ctx context.Context, workDir string, patch []byte, description string) error {
	if len(patch) == 0 {
		return nil
	}
	// #nosec G204 -- executable and subcommand are fixed; the patch is provided over stdin.
	cmd := exec.CommandContext(ctx, "git", "-C", workDir, "apply", "--whitespace=nowarn", "-")
	cmd.Stdin = bytes.NewReader(patch)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("apply %s: %w: %s", description, err, strings.TrimSpace(string(output)))
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
		return result, nil
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
		return result, nil
	}
	result.Oracle = oracle
	result.Success = oracle.Passed
	if !oracle.Passed {
		result.Error = oracle.ErrorMessage
	}
	return result, nil
}
