// This file executes ordered configured verification hooks and publishes staged normalization output to the selected project root.

package projectverify

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"semedit/internal/gocache"
	"semedit/internal/pipeline"
	"semedit/internal/sourcefiles"
	"slices"
	"strings"
	"time"
)

func (o *limitedOutput) Write(data []byte) (int, error) {
	remaining := o.limit - o.buffer.Len()
	if remaining > 0 {
		if remaining > len(data) {
			remaining = len(data)
		}
		_, _ = o.buffer.Write(data[:remaining])
	}
	if remaining < len(data) {
		o.truncated = true
	}
	return len(data), nil
}

// RunPhase executes the selected hooks in order and records their outcomes.
func RunPhase(ctx context.Context, plan Plan, phase Phase, scope Scope, lsp LSPRunner) (Report, error) {
	if ctx == nil {
		return Report{}, fmt.Errorf("verification context is nil")
	}
	if phase != PhaseNormalize && phase != PhaseCheck {
		return Report{}, fmt.Errorf("unsupported verification phase %q", phase)
	}
	root, err := canonicalVerificationRoot(plan.Root)
	if err != nil {
		return Report{}, err
	}
	plan.Root = root
	scope, err = normalizeScope(root, scope)
	if err != nil {
		return Report{}, err
	}
	report := Report{Root: root, ConfigPath: plan.ConfigPath, Phase: phase, Scope: scope}
	hooks := plan.Checks
	if phase == PhaseNormalize {
		hooks = plan.Normalize
	}
	completedCount := 0
	for _, hook := range hooks {
		if ctxErr := ctx.Err(); ctxErr != nil {
			report.Partial = completedCount > 0
			return report, fmt.Errorf("verification phase cancelled: %w", ctxErr)
		}
		result, runErr := executeVerificationHook(ctx, plan, phase, scope, root, hook, lsp)
		report.Results = append(report.Results, result)
		if errors.Is(runErr, errNoApplicablePackages) {
			continue
		}
		if runErr != nil {
			report.Partial = completedCount > 0 || len(result.Changed) > 0
			return report, runErr
		}
		if result.Status == HookSkipped {
			continue
		}
		completedCount++
	}
	if ctxErr := ctx.Err(); ctxErr != nil {
		report.Partial = completedCount > 0
		return report, fmt.Errorf("verification phase cancelled: %w", ctxErr)
	}
	report.Completed = true
	return report, nil
}

func canonicalVerificationRoot(root string) (string, error) {
	if strings.TrimSpace(root) == "" {
		return "", fmt.Errorf("verification plan has no project root")
	}
	canonical, err := filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("resolve verification root: %w", err)
	}
	canonical, err = filepath.EvalSymlinks(canonical)
	if err != nil {
		return "", fmt.Errorf("resolve verification root symlinks: %w", err)
	}
	info, err := os.Stat(canonical)
	if err != nil {
		return "", fmt.Errorf("stat verification root: %w", err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("verification root %q is not a directory", canonical)
	}
	return canonical, nil
}

func executeVerificationHook(ctx context.Context, plan Plan, phase Phase, requested Scope, root string, hook Hook, lsp LSPRunner) (HookResult, error) {
	result := HookResult{ID: hook.ID, Signal: hook.Signal, ConfigPath: hook.ConfigPath, ScopePaths: slices.Clone(hook.ScopePaths)}
	selected, applies, err := SelectHookScope(plan, hook, requested)
	if err != nil {
		result.Status, result.Error = HookFailed, err.Error()
		return result, fmt.Errorf("select scope for verification hook %q: %w", hook.ID, err)
	}
	if hook.Disabled || !applies {
		result.Status, result.ScopePaths = HookSkipped, selected.Paths
		return result, nil
	}
	runErr := executeSelectedHook(ctx, root, phase, requested, selected, hook, lsp, &result)
	finalizeHookResult(&result, hook, selected)
	if errors.Is(runErr, errNoApplicablePackages) {
		result.Status, result.ScopePaths = HookSkipped, slices.Clone(selected.Paths)
		return result, runErr
	}
	if runErr != nil {
		result.Status, result.Error = HookFailed, runErr.Error()
		return result, fmt.Errorf("verification hook %q failed: %w", hook.ID, runErr)
	}
	result.Status = HookSucceeded
	return result, nil
}

func executeSelectedHook(ctx context.Context, root string, phase Phase, requested, selected Scope, hook Hook, lsp LSPRunner, result *HookResult) error {
	started := time.Now()
	result.StartedAt = started
	timeout := hook.Timeout
	if timeout <= 0 {
		timeout = defaultHookTimeout
	}
	hookCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	runErr := runConfiguredHook(hookCtx, root, phase, requested, selected, hook, lsp, started, result)
	if runErr == nil && hookCtx.Err() != nil {
		runErr = hookCtx.Err()
	}
	return runErr
}

func runConfiguredHook(ctx context.Context, root string, phase Phase, requested, selected Scope, hook Hook, lsp LSPRunner, started time.Time, result *HookResult) error {
	if hook.LSP != nil {
		return runLSPHook(ctx, hook, selected, lsp, started, result)
	}
	if phase == PhaseNormalize {
		return runNormalization(ctx, root, hook, selected, requested, result)
	}
	return runCheck(ctx, root, hook, selected, result)
}

func runLSPHook(ctx context.Context, hook Hook, selected Scope, lsp LSPRunner, started time.Time, result *HookResult) error {
	if lsp == nil {
		return fmt.Errorf("no LSP runner is available for configured action")
	}
	lspResult, runErr := lsp(ctx, hook, selected)
	if lspResult.ID == "" {
		lspResult.ID = hook.ID
	}
	if lspResult.Kind == "" {
		lspResult.Kind = "lsp"
	}
	if lspResult.Signal == "" {
		lspResult.Signal = hook.Signal
	}
	if lspResult.ConfigPath == "" {
		lspResult.ConfigPath = hook.ConfigPath
	}
	lspResult.ScopePaths = slices.Clone(selected.Paths)
	lspResult.StartedAt, lspResult.FinishedAt = started, time.Now()
	*result = lspResult
	if runErr == nil && result.Status != HookSucceeded {
		if result.Error != "" {
			runErr = errors.New(result.Error)
		} else {
			runErr = fmt.Errorf("LSP runner returned status %q", result.Status)
		}
	}
	return runErr
}

func finalizeHookResult(result *HookResult, hook Hook, selected Scope) {
	result.FinishedAt = time.Now()
	if result.StartedAt.IsZero() {
		result.StartedAt = result.FinishedAt
	}
	if result.Kind == "" {
		//nolint:gocritic // Result kind follows the configured execution form.
		if hook.Shell != nil {
			result.Kind = "shell"
		} else if hook.LSP != nil {
			result.Kind = "lsp"
		} else {
			result.Kind = "exec"
		}
	}
	if result.ID == "" {
		result.ID = hook.ID
	}
	if result.Signal == "" {
		result.Signal = hook.Signal
	}
	if result.ConfigPath == "" {
		result.ConfigPath = hook.ConfigPath
	}
	if !slices.Equal(result.ScopePaths, selected.Paths) {
		result.ScopePaths = slices.Clone(selected.Paths)
	}
}

var errNoApplicablePackages = errors.New("no buildable Go packages are selected")

const (
	defaultHookTimeout = 2 * time.Minute
	outputLimit        = 64 * 1024
)

type fileSnapshot struct {
	digest [32]byte
	mode   fs.FileMode
}

type limitedOutput struct {
	buffer    bytes.Buffer
	limit     int
	truncated bool
}

// SelectHookScope returns the paths a hook would cover and whether it applies. Disabled hooks never apply.
func SelectHookScope(plan Plan, hook Hook, requested Scope) (Scope, bool, error) {
	if hook.Disabled {
		return Scope{}, false, nil
	}
	requestPaths := make([]string, 0, len(requested.Paths))
	for _, path := range requested.Paths {
		rel, err := scopePath(plan.Root, path)
		if err != nil {
			return Scope{}, false, err
		}
		requestPaths = append(requestPaths, rel)
	}
	languages := make(map[string]bool, len(hook.Languages))
	for _, language := range hook.Languages {
		languages[language] = true
	}
	var selected []string
	for _, source := range plan.Sources {
		if len(languages) > 0 && !languages[source.Language] {
			continue
		}
		if len(hook.ScopePaths) > 0 && !matchesAny(source.Path, hook.ScopePaths) {
			continue
		}
		if len(requestPaths) > 0 && !matchesAny(source.Path, requestPaths) {
			continue
		}
		selected = append(selected, source.Path)
	}
	if len(selected) == 0 {
		return Scope{}, false, nil
	}
	slices.Sort(selected)
	return Scope{Paths: selected}, true, nil
}

func scopePath(root, path string) (string, error) {
	if filepath.IsAbs(path) {
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return "", fmt.Errorf("resolve requested scope %q: %w", path, err)
		}
		if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return "", fmt.Errorf("requested scope %q is outside project root", path)
		}
		path = rel
	}
	clean, err := safeRelativePath(path)
	if err != nil {
		return "", fmt.Errorf("invalid requested scope %q: %w", path, err)
	}
	if err := ensureNoSymlinkPath(root, clean); err != nil {
		return "", fmt.Errorf("invalid requested scope %q: %w", path, err)
	}
	return filepath.ToSlash(clean), nil
}

func matchesAny(path string, candidates []string) bool {
	for _, candidate := range candidates {
		clean := filepath.ToSlash(filepath.Clean(filepath.FromSlash(candidate)))
		if clean == "." || path == clean || strings.HasPrefix(path, strings.TrimSuffix(clean, "/")+"/") {
			return true
		}
	}
	return false
}

func runCheck(ctx context.Context, root string, hook Hook, scope Scope, result *HookResult) error {
	executable, args, err := invocationForRoot(ctx, root, hook, scope)
	if err != nil {
		return err
	}
	cwd, err := hookWorkingDirectory(root, hook)
	if err != nil {
		return err
	}
	return runHookCommand(ctx, root, hook, cwd, executable, args, result)
}

func runCommand(ctx context.Context, cwd, executable string, args []string, result *HookResult) error {
	return runCommandWithEnv(ctx, cwd, executable, args, result, nil)
}

func hookWorkingDirectory(root string, hook Hook) (string, error) {
	rel := hook.Cwd
	if rel == "" {
		rel = "."
	}
	clean, err := safeRelativePath(rel)
	if err != nil {
		return "", fmt.Errorf("invalid hook working directory: %w", err)
	}
	if symlinkErr := ensureNoSymlinkPath(root, clean); symlinkErr != nil {
		return "", fmt.Errorf("invalid hook working directory: %w", symlinkErr)
	}
	dir := filepath.Join(root, clean)
	info, err := os.Stat(dir)
	if err != nil {
		return "", fmt.Errorf("resolve hook working directory: %w", err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("hook working directory %q is not a directory", rel)
	}
	return dir, nil
}

func hookInvocation(hook Hook, scope Scope) (string, []string, error) {
	if len(scope.Paths) == 0 {
		return "", nil, errNoApplicablePackages
	}
	if hook.LSP != nil {
		return "", nil, fmt.Errorf("LSP hook must be run through the LSP callback")
	}
	if hook.Shell != nil {
		args := append([]string(nil), hook.Shell.Args...)
		args = append(args, hook.Shell.Script)
		return hook.Shell.Executable, args, nil
	}
	if len(hook.Exec) == 0 {
		return "", nil, fmt.Errorf("hook %q has no executable arguments", hook.ID)
	}
	return hook.Exec[0], append([]string(nil), hook.Exec[1:]...), nil
}

func runNormalization(ctx context.Context, root string, hook Hook, selected, requested Scope, result *HookResult) error {
	workspace, err := prepareNormalization(ctx, root, hook, selected, result)
	if err != nil {
		return err
	}
	defer func() { _ = os.RemoveAll(workspace.stage) }()
	changed, err := normalizationChanges(ctx, workspace.before, workspace.after)
	if err != nil {
		return err
	}
	if err := validateNormalizationChanges(ctx, root, changed, hook, requested); err != nil {
		return err
	}
	return publishNormalizationChanges(ctx, root, workspace, changed, result)
}

type normalizationWorkspace struct {
	stage, stageRoot string
	before, after    map[string]fileSnapshot
}

func prepareNormalization(ctx context.Context, root string, hook Hook, selected Scope, result *HookResult) (normalizationWorkspace, error) {
	if ctxErr := ctx.Err(); ctxErr != nil {
		return normalizationWorkspace{}, fmt.Errorf("normalization was cancelled before staging: %w", ctxErr)
	}
	if err := ensureNoSymlinkPath(root, ".scratch"); err != nil {
		return normalizationWorkspace{}, err
	}
	scratch := filepath.Join(root, ".scratch")
	if err := os.MkdirAll(scratch, 0o700); err != nil {
		return normalizationWorkspace{}, fmt.Errorf("create verification scratch directory: %w", err)
	}
	stage, stageErr := os.MkdirTemp(scratch, "projectverify-")
	if stageErr != nil {
		return normalizationWorkspace{}, fmt.Errorf("create normalization staging area: %w", stageErr)
	}
	keepStage := false
	defer func() {
		if !keepStage {
			_ = os.RemoveAll(stage)
		}
	}()
	stageRoot := filepath.Join(stage, "workspace")
	if err := os.Mkdir(stageRoot, 0o700); err != nil {
		return normalizationWorkspace{}, fmt.Errorf("create staged workspace: %w", err)
	}
	if err := copyWorkspaceContext(ctx, root, stageRoot); err != nil {
		return normalizationWorkspace{}, fmt.Errorf("stage project for normalization: %w", err)
	}
	before, beforeErr := snapshotWorkspaceContext(ctx, stageRoot)
	if beforeErr != nil {
		return normalizationWorkspace{}, fmt.Errorf("snapshot staged workspace: %w", beforeErr)
	}
	cwd, cwdErr := hookWorkingDirectory(stageRoot, hook)
	if cwdErr != nil {
		return normalizationWorkspace{}, cwdErr
	}
	executable, args, invocationErr := invocationForRoot(ctx, stageRoot, hook, selected)
	if invocationErr != nil {
		return normalizationWorkspace{}, invocationErr
	}
	if err := runHookCommand(ctx, root, hook, cwd, executable, args, result); err != nil {
		return normalizationWorkspace{}, err
	}
	if err := ctx.Err(); err != nil {
		return normalizationWorkspace{}, fmt.Errorf("normalization was cancelled after command completion: %w", err)
	}
	after, afterErr := snapshotWorkspaceContext(ctx, stageRoot)
	if afterErr != nil {
		return normalizationWorkspace{}, fmt.Errorf("inspect staged normalization result: %w", afterErr)
	}
	keepStage = true
	return normalizationWorkspace{stage: stage, stageRoot: stageRoot, before: before, after: after}, nil
}

func normalizationChanges(ctx context.Context, before, after map[string]fileSnapshot) ([]string, error) {
	var changed []string
	for path, old := range before {
		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("normalization was cancelled while inspecting changes: %w", err)
		}
		current, exists := after[path]
		if !exists {
			return nil, fmt.Errorf("normalization hook deleted %q; deletions are not supported", path)
		}
		if old.digest != current.digest || old.mode.Perm() != current.mode.Perm() {
			changed = append(changed, path)
		}
	}
	for path := range after {
		if _, exists := before[path]; !exists {
			changed = append(changed, path)
		}
	}
	slices.Sort(changed)
	return changed, nil
}

func validateNormalizationChanges(ctx context.Context, root string, changed []string, hook Hook, requested Scope) error {
	for _, path := range changed {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("normalization was cancelled while validating changes: %w", err)
		}
		if err := validatePublishPath(root, path, hook, requested); err != nil {
			return err
		}
	}
	return nil
}

func publishNormalizationChanges(ctx context.Context, root string, workspace normalizationWorkspace, changed []string, result *HookResult) error {
	for _, path := range changed {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("normalization was cancelled before publishing changes: %w", err)
		}
		relative := filepath.FromSlash(path)
		stagedPath := filepath.Join(workspace.stageRoot, relative)
		target := filepath.Join(root, relative)
		data, err := os.ReadFile(stagedPath) // #nosec G304 -- path is within the private staging tree
		if err != nil {
			return fmt.Errorf("read staged change %q: %w", path, err)
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o750); err != nil {
			return fmt.Errorf("create parent for staged change %q: %w", path, err)
		}
		if err := ensureRegularTarget(root, relative); err != nil {
			return err
		}
		if err := pipeline.WriteAtomic(target, data); err != nil {
			return fmt.Errorf("publish staged change %q: %w", path, err)
		}
		result.Changed = append(result.Changed, filepath.ToSlash(path))
		if err := os.Chmod(target, workspace.after[path].mode.Perm()); err != nil {
			return fmt.Errorf("preserve mode for published change %q: %w", path, err)
		}
	}
	return nil
}

//nolint:unused // Kept as the context-free counterpart for focused filesystem callers.
func copyWorkspace(source, destination string) error {
	return copyWorkspaceContext(context.Background(), source, destination)
}

func stagedExcludedDirectory(name string) bool {
	return sourcefiles.IsIgnoredDirectory(name) || name == "__pycache__" || name == ".pytest_cache" || name == ".mypy_cache"
}

//nolint:unused // Kept as the context-free counterpart for focused filesystem callers.
func snapshotWorkspace(root string) (map[string]fileSnapshot, error) {
	return snapshotWorkspaceContext(context.Background(), root)
}

func validatePublishPath(root, path string, hook Hook, requested Scope) error {
	rel, err := safeRelativePath(filepath.FromSlash(path))
	if err != nil || rel == "." {
		return fmt.Errorf("normalization produced unsafe path %q", path)
	}
	path = filepath.ToSlash(rel)
	if protectedManifest(path) {
		return fmt.Errorf("normalization hook changed protected manifest %q", path)
	}
	if !pathAllowed(path, hook.ScopePaths) || !pathAllowed(path, requested.Paths) {
		return fmt.Errorf("normalization hook changed out-of-scope path %q", path)
	}
	return ensureRegularTarget(root, rel)
}

func pathAllowed(path string, scopes []string) bool {
	if len(scopes) == 0 {
		return true
	}
	for _, scope := range scopes {
		clean, err := safeRelativePath(filepath.FromSlash(scope))
		if err != nil {
			continue
		}
		normalized := filepath.ToSlash(clean)
		if normalized == "." || path == normalized || strings.HasPrefix(path, strings.TrimSuffix(normalized, "/")+"/") {
			return true
		}
	}
	return false
}

func protectedManifest(path string) bool {
	switch strings.ToLower(filepath.Base(path)) {
	case "go.mod", "go.sum", "go.work", "go.work.sum", "cargo.toml", "cargo.lock", "package.json", "package-lock.json", "pnpm-lock.yaml", "yarn.lock", "bun.lock", "bun.lockb", "pyproject.toml", "poetry.lock", "pdm.lock", "composer.json", "composer.lock", "gemfile", "gemfile.lock", "mix.exs", "mix.lock":
		return true
	default:
		return false
	}
}

func ensureRegularTarget(root, relative string) error {
	clean, pathErr := safeRelativePath(relative)
	if pathErr != nil || clean == "." {
		return fmt.Errorf("unsafe publication target %q", relative)
	}
	parent := filepath.Dir(clean)
	for current := parent; current != "."; current = filepath.Dir(current) {
		info, err := os.Lstat(filepath.Join(root, current))
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return fmt.Errorf("inspect publication parent %q: %w", current, err)
		}
		if info.Mode()&fs.ModeSymlink != 0 || !info.IsDir() {
			return fmt.Errorf("publication parent %q is not a regular directory", current)
		}
	}
	target := filepath.Join(root, clean)
	info, err := os.Lstat(target)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("inspect publication target %q: %w", clean, err)
	}
	if info.Mode()&fs.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return fmt.Errorf("publication target %q is not a regular file", clean)
	}
	return nil
}

func normalizeScope(root string, scope Scope) (Scope, error) {
	seen := make(map[string]bool, len(scope.Paths))
	paths := make([]string, 0, len(scope.Paths))
	for _, path := range scope.Paths {
		normalized, err := scopePath(root, path)
		if err != nil {
			return Scope{}, err
		}
		if !seen[normalized] {
			seen[normalized] = true
			paths = append(paths, normalized)
		}
	}
	slices.Sort(paths)
	return Scope{Paths: paths}, nil
}

func copyWorkspaceContext(ctx context.Context, source, destination string) error {
	return filepath.WalkDir(source, func(path string, entry fs.DirEntry, walkErr error) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		if entry.IsDir() && stagedExcludedDirectory(entry.Name()) {
			return filepath.SkipDir
		}
		if entry.Type()&fs.ModeSymlink != 0 {
			return nil
		}
		return copyWorkspaceEntry(path, rel, entry, destination)
	})
}

func copyWorkspaceEntry(path, rel string, entry fs.DirEntry, destination string) error {
	target := filepath.Join(destination, rel)
	info, err := entry.Info()
	if err != nil {
		return err
	}
	if entry.IsDir() {
		return os.MkdirAll(target, info.Mode().Perm())
	}
	if !entry.Type().IsRegular() {
		return fmt.Errorf("cannot stage non-regular project entry %q", filepath.ToSlash(rel))
	}
	if mkdirErr := os.MkdirAll(filepath.Dir(target), 0o750); mkdirErr != nil {
		return mkdirErr
	}
	input, err := os.Open(path) // #nosec G304,G122 -- path is produced by WalkDir beneath the private staging root
	if err != nil {
		return err
	}
	output, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, info.Mode().Perm()) // #nosec G304 -- target is a path beneath the private staging root
	if err != nil {
		_ = input.Close()
		return err
	}
	_, copyErr := io.Copy(output, input)
	closeInputErr := input.Close()
	closeOutputErr := output.Close()
	if copyErr != nil {
		return copyErr
	}
	if closeInputErr != nil {
		return closeInputErr
	}
	if closeOutputErr != nil {
		return closeOutputErr
	}
	return os.Chmod(target, info.Mode().Perm())
}

func snapshotWorkspaceContext(ctx context.Context, root string) (map[string]fileSnapshot, error) {
	files := make(map[string]fileSnapshot)
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		if entry.IsDir() && stagedExcludedDirectory(entry.Name()) {
			return filepath.SkipDir
		}
		if entry.Type()&fs.ModeSymlink != 0 {
			return fmt.Errorf("normalization hook created symlink %q", filepath.ToSlash(rel))
		}
		if entry.IsDir() {
			return nil
		}
		if !entry.Type().IsRegular() {
			return fmt.Errorf("normalization hook created unsupported filesystem entry %q", filepath.ToSlash(rel))
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		data, err := os.ReadFile(path) // #nosec G304,G122 -- path is produced by WalkDir beneath the private staging root
		if err != nil {
			return err
		}
		files[filepath.ToSlash(rel)] = fileSnapshot{digest: sha256.Sum256(data), mode: info.Mode()}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return files, nil
}

func ensureNoSymlinkPath(root, relative string) error {
	clean, err := safeRelativePath(relative)
	if err != nil {
		return err
	}
	if clean == "." {
		return nil
	}
	parts := strings.Split(clean, string(filepath.Separator))
	current := root
	for i, part := range parts {
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("inspect path component %q: %w", current, err)
		}
		if info.Mode()&fs.ModeSymlink != 0 {
			return fmt.Errorf("path component %q is a symlink", current)
		}
		if i < len(parts)-1 && !info.IsDir() {
			return fmt.Errorf("path component %q is not a directory", current)
		}
	}
	return nil
}

func invocationForRoot(ctx context.Context, root string, hook Hook, scope Scope) (string, []string, error) {
	executable, args, err := hookInvocation(hook, scope)
	if err != nil {
		return "", nil, err
	}
	if hook.Signal != "golangci-config" {
		return executable, args, nil
	}
	packages, err := golangciPackages(ctx, root, hook, scope)
	if err != nil {
		return "", nil, err
	}
	if len(packages) == 0 {
		return "", nil, errNoApplicablePackages
	}
	configPath := filepath.Join(root, filepath.FromSlash(hook.ConfigPath))
	return executable, append([]string{"run", "--config", configPath}, packages...), nil
}

func golangciPackages(ctx context.Context, root string, hook Hook, scope Scope) ([]string, error) {
	cwd, err := hookWorkingDirectory(root, hook)
	if err != nil {
		return nil, err
	}
	env, err := commandEnvironment(ctx, root, hook)
	if err != nil {
		return nil, err
	}
	cmd := exec.CommandContext(ctx, "go", "list", "-mod=readonly", "-f", "{{.Dir}}", "./...")
	cmd.Dir = cwd
	cmd.Env = env
	cmd.WaitDelay = 250 * time.Millisecond
	output, err := cmd.Output()
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, fmt.Errorf("resolve buildable Go package scope: %w", ctxErr)
		}
		if exitErr, ok := errors.AsType[*exec.ExitError](err); ok {
			return nil, fmt.Errorf("resolve buildable Go package scope: go list exited with status %d: %s", exitErr.ExitCode(), strings.TrimSpace(string(exitErr.Stderr)))
		}
		return nil, fmt.Errorf("resolve buildable Go package scope: %w", err)
	}
	if len(output) > outputLimit {
		return nil, fmt.Errorf("resolve buildable Go package scope: go list output exceeds %d bytes", outputLimit)
	}
	return filterGolangciPackages(root, cwd, hook.ScopePaths, scope.Paths, output)
}

func filterGolangciPackages(root, cwd string, configuredPaths, selectedPaths []string, output []byte) ([]string, error) {
	selected := make(map[string]bool, len(selectedPaths))
	for _, path := range selectedPaths {
		selected[filepath.ToSlash(filepath.Clean(filepath.FromSlash(path)))] = true
	}
	configuredScope := make(map[string]bool, len(configuredPaths))
	for _, path := range configuredPaths {
		configuredScope[filepath.ToSlash(filepath.Clean(filepath.FromSlash(path)))] = true
	}
	var packages []string
	seen := make(map[string]bool)
	for line := range strings.SplitSeq(strings.TrimSpace(string(output)), "\n") {
		if line == "" {
			continue
		}
		dir, err := filepath.Abs(line)
		if err != nil {
			return nil, fmt.Errorf("resolve Go package directory %q: %w", line, err)
		}
		relDir, err := filepath.Rel(cwd, dir)
		if err != nil || relDir == ".." || strings.HasPrefix(relDir, ".."+string(filepath.Separator)) {
			continue
		}
		relDir = filepath.ToSlash(relDir)
		if !containsSelectedGoFile(root, dir, selected, configuredScope) {
			continue
		}
		pattern := "./"
		if relDir != "." {
			pattern += strings.TrimSuffix(relDir, "/")
		}
		if !seen[pattern] {
			seen[pattern] = true
			packages = append(packages, pattern)
		}
	}
	slices.Sort(packages)
	return packages, nil
}

func containsSelectedGoFile(root, packageDir string, selected, configured map[string]bool) bool {
	for path := range selected {
		absoluteFile := filepath.Join(root, filepath.FromSlash(path))
		if filepath.Dir(absoluteFile) == packageDir && (len(configured) == 0 || configured[path]) {
			return true
		}
	}
	return false
}

func commandEnvironment(ctx context.Context, root string, hook Hook) ([]string, error) {
	for _, language := range hook.Languages {
		if language == "go" {
			env, err := gocache.Environment(ctx, root)
			if err != nil {
				return nil, fmt.Errorf("prepare Go verification environment: %w", err)
			}
			return env, nil
		}
	}
	return nil, nil
}

func runCommandWithEnv(ctx context.Context, cwd, executable string, args []string, result *HookResult, env []string) error {
	if strings.TrimSpace(executable) == "" {
		return fmt.Errorf("hook executable is empty")
	}
	stdout := &limitedOutput{limit: outputLimit}
	stderr := &limitedOutput{limit: outputLimit}
	cmd := exec.CommandContext(ctx, executable, args...) // #nosec G204 -- execution form is an explicit project verification hook
	cmd.Dir = cwd
	if env != nil {
		cmd.Env = env
	}
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.WaitDelay = 250 * time.Millisecond
	err := cmd.Run()
	result.Stdout = stdout.buffer.String()
	result.Stderr = stderr.buffer.String()
	result.Truncated = stdout.truncated || stderr.truncated
	if err == nil {
		result.ExitCode = 0
		return nil
	}
	if ctxErr := ctx.Err(); ctxErr != nil {
		if errors.Is(ctxErr, context.DeadlineExceeded) {
			return fmt.Errorf("hook exceeded its timeout: %w", ctxErr)
		}
		return fmt.Errorf("hook was cancelled: %w", ctxErr)
	}
	if exitErr, ok := errors.AsType[*exec.ExitError](err); ok {
		result.ExitCode = exitErr.ExitCode()
		return fmt.Errorf("command exited with status %d: %w", exitErr.ExitCode(), err)
	}
	return fmt.Errorf("start or wait for command: %w", err)
}

func runHookCommand(ctx context.Context, root string, hook Hook, cwd, executable string, args []string, result *HookResult) error {
	env, err := commandEnvironment(ctx, root, hook)
	if err != nil {
		return err
	}
	if env == nil {
		return runCommand(ctx, cwd, executable, args, result)
	}
	return runCommandWithEnv(ctx, cwd, executable, args, result, env)
}
