// Package mcp owns volatile MCP staging journals and retains only evidence needed to replay accepted edits.
package mcp

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/spockz/semantic-editor/internal/backend"
	"github.com/spockz/semantic-editor/internal/backend/pathutil"
	"github.com/spockz/semantic-editor/internal/godiagnostics"
	"github.com/spockz/semantic-editor/internal/operation"
	"github.com/spockz/semantic-editor/internal/projectverify"
	"github.com/spockz/semantic-editor/internal/sourcebuf"
)

const (
	defaultStageStepLimit       = 50
	defaultStageByteLimit int64 = 10 << 20
	defaultStageAge             = 30 * time.Minute
	trustWorkspaceField         = "trust_workspace"
)

type stageLimits struct {
	steps                          int
	requestBytes, dirtySourceBytes int64
	maxAge                         time.Duration
}
type stageKey struct{ root, module string }
type stageStep struct {
	id, key, tool string
	params        json.RawMessage
	bytes         int64
	paths         []string
}
type stageState struct {
	id                string
	key               stageKey
	baseline, current *sourcebuf.Buffer
	workspaceBefore   map[string]string
	verification      batchVerificationBaseline
	project           backend.ProjectContext
	started           time.Time
	steps             []stageStep
	requestBytes      int64
	nextOrdinal       int
	terminal          string
}
type stageManager struct {
	server *Server
	limits stageLimits
	mu     sync.Mutex
	stages map[stageKey]*stageState
}

func newStageManager(s *Server) *stageManager {
	return &stageManager{server: s, limits: s.stageLimits, stages: make(map[stageKey]*stageState)}
}

func stageModule(root string) (string, error) {
	var mavenRoot string
	for dir := root; ; dir = filepath.Dir(dir) {
		manifest := filepath.Join(dir, "go.mod")
		info, err := os.Lstat(manifest)
		if err == nil {
			if !info.Mode().IsRegular() {
				return "", fmt.Errorf("go module manifest %q is not regular", manifest)
			}
			return filepath.EvalSymlinks(dir)
		}
		if !os.IsNotExist(err) {
			return "", fmt.Errorf("inspect Go module manifest: %w", err)
		}
		pom := filepath.Join(dir, "pom.xml")
		pomInfo, pomErr := os.Lstat(pom)
		if pomErr == nil {
			if !pomInfo.Mode().IsRegular() {
				return "", fmt.Errorf("maven POM %q is not regular", pom)
			}
			if mavenRoot == "" {
				mavenRoot = dir
			}
		} else if !os.IsNotExist(pomErr) {
			return "", fmt.Errorf("inspect Maven POM: %w", pomErr)
		}
		if parent := filepath.Dir(dir); parent == dir {
			if mavenRoot != "" {
				return filepath.EvalSymlinks(mavenRoot)
			}
			return "", fmt.Errorf("no enclosing Go module or Maven project for %q", root)
		}
	}
}
func stageScope(root string) (stageKey, error) {
	canonical, err := stageCanonicalRoot(root)
	if err != nil {
		return stageKey{}, err
	}
	module, err := stageModule(canonical)
	if err != nil {
		return stageKey{}, err
	}
	return stageKey{canonical, module}, nil
}
func stageCanonicalRoot(root string) (string, error) {
	canonical, err := filepath.EvalSymlinks(root)
	if err != nil {
		return "", fmt.Errorf("canonicalize staging root: %w", err)
	}
	return canonical, nil
}
func overlap(a, b string) bool {
	rel, err := filepath.Rel(a, b)
	return err == nil && (rel == "." || (!strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel)))
}
func (m *stageManager) expired(state *stageState) bool {
	return m.limits.maxAge > 0 && time.Since(state.started) > m.limits.maxAge
}

func (m *stageManager) expireLocked(ctx context.Context) error {
	expired := false
	for key, state := range m.stages {
		if !m.expired(state) {
			continue
		}
		delete(m.stages, key)
		expired = true
	}
	if !expired {
		return nil
	}
	if err := godiagnostics.InvalidateWithContext(ctx); err != nil {
		return fmt.Errorf("invalidate diagnostics after expired staging session: %w", err)
	}
	return nil
}

func stageLimitError(resource string, current, limit int64) error {
	return fmt.Errorf("staging %s limit exceeded: current=%d limit=%d", resource, current, limit)
}
func (m *stageManager) receipt(s *stageState, status, step string) operation.StageReceipt {
	steps := make([]operation.StageStep, len(s.steps))
	for i, staged := range s.steps {
		steps[i] = operation.StageStep{StepID: staged.id}
	}
	return operation.StageReceipt{Status: status, StageID: s.id, StepID: step, Steps: steps, StepCount: len(s.steps), RequestBytes: s.requestBytes, DirtySourceBytes: s.current.DirtySourceBytes(), Limits: operation.StageLimits{Steps: m.limits.steps, RequestBytes: m.limits.requestBytes, DirtySourceBytes: m.limits.dirtySourceBytes, MaxAge: m.limits.maxAge}}
}
func (m *stageManager) Start(ctx context.Context, root string) (operation.StageReceipt, error) {
	key, err := stageScope(root)
	if err != nil {
		return operation.StageReceipt{}, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if expiryErr := m.expireLocked(ctx); expiryErr != nil {
		return operation.StageReceipt{}, expiryErr
	}
	for _, other := range m.stages {
		currentModule, moduleErr := stageModule(other.key.root)
		if moduleErr != nil {
			return operation.StageReceipt{}, fmt.Errorf("active staging owner %s no longer has a valid module: %w", other.key.root, moduleErr)
		}
		if other.key.module == key.module || currentModule == key.module || overlap(other.key.root, key.root) || overlap(key.root, other.key.root) {
			return operation.StageReceipt{}, fmt.Errorf("a staging session already owns module workspace %s", other.key.root)
		}
	}
	b, err := sourcebuf.New(key.root)
	if err != nil {
		return operation.StageReceipt{}, err
	}
	workspaceBefore, err := snapshotBatchWorkspace(key.root)
	if err != nil {
		return operation.StageReceipt{}, fmt.Errorf("snapshot staging workspace: %w", err)
	}
	plan, err := projectverify.Discover(key.root)
	if err != nil {
		return operation.StageReceipt{}, fmt.Errorf("discover staging verification plan: %w", err)
	}
	verification, err := collectBatchVerificationBaseline(ctx, key.root, plan, m.server.enabledLanguages, nil)
	if err != nil {
		return operation.StageReceipt{}, fmt.Errorf("capture staging diagnostic baseline: %w", err)
	}
	var nonce [16]byte
	if _, err = rand.Read(nonce[:]); err != nil {
		return operation.StageReceipt{}, err
	}
	state := &stageState{id: hex.EncodeToString(nonce[:]), key: key, baseline: b.ForkOriginal(), current: b, workspaceBefore: workspaceBefore, verification: verification, nextOrdinal: 1, started: time.Now()}
	m.stages[key] = state
	return m.receipt(state, "staged", ""), nil
}
func (m *stageManager) state(ctx context.Context, root string) (stageKey, *stageState, error) {
	canonical, err := stageCanonicalRoot(root)
	if err != nil {
		return stageKey{}, nil, err
	}
	var key stageKey
	var state *stageState
	for candidateKey, candidate := range m.stages {
		if candidateKey.root == canonical {
			key, state = candidateKey, candidate
			break
		}
	}
	if state == nil {
		return stageKey{root: canonical}, nil, fmt.Errorf("no staging session is active for %s", canonical)
	}
	if m.expired(state) {
		delete(m.stages, state.key)
		expiredErr := fmt.Errorf("staging session expired after %s (max_age=%s); no staged bytes were published", time.Since(state.started).Round(time.Millisecond), m.limits.maxAge)
		return key, nil, errors.Join(expiredErr, godiagnostics.InvalidateWithContext(ctx))
	}
	if module, moduleErr := stageModule(state.key.root); moduleErr != nil || module != state.key.module {
		return key, nil, fmt.Errorf("staging module ownership changed for %s", state.key.root)
	}
	if state.terminal != "" {
		return key, nil, fmt.Errorf("staging session is terminal (%s); abort it", state.terminal)
	}
	return key, state, nil
}

func (m *stageManager) ownerLocked(root string) (*stageState, bool, error) {
	canonical, err := stageCanonicalRoot(root)
	if err != nil {
		return nil, false, err
	}
	module, moduleErr := stageModule(canonical)
	for _, state := range m.stages {
		if state.key.root == canonical {
			if moduleErr != nil || module != state.key.module {
				return nil, false, fmt.Errorf("staging module ownership changed for %s", state.key.root)
			}
			return state, true, nil
		}
		if overlap(state.key.root, canonical) || overlap(canonical, state.key.root) {
			return nil, false, fmt.Errorf("request workspace %s overlaps active staging workspace %s", canonical, state.key.root)
		}
		ownerModule, ownerErr := stageModule(state.key.root)
		if ownerErr != nil {
			return nil, false, fmt.Errorf("active staging owner %s no longer has a valid module: %w", state.key.root, ownerErr)
		}
		if moduleErr != nil {
			return nil, false, fmt.Errorf("resolve request module while staging is active: %w", moduleErr)
		}
		if ownerModule == module || state.key.module == module {
			return nil, false, fmt.Errorf("request workspace %s is in active staging module %s", canonical, state.key.root)
		}
	}
	return nil, false, nil
}

func (m *stageManager) BlockBatch(ctx context.Context, root string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.expireLocked(ctx); err != nil {
		return err
	}
	owner, found, err := m.ownerLocked(root)
	if err != nil {
		return err
	}
	if found {
		return fmt.Errorf("semantic_batch is unavailable while staging is active for %s", owner.key.root)
	}
	return nil
}
func (m *stageManager) Abort(ctx context.Context, root string) (operation.StageReceipt, error) {
	m.mu.Lock()
	if err := m.expireLocked(ctx); err != nil {
		m.mu.Unlock()
		return operation.StageReceipt{}, err
	}
	canonical, err := stageCanonicalRoot(root)
	if err != nil {
		m.mu.Unlock()
		return operation.StageReceipt{}, err
	}
	var key stageKey
	var state *stageState
	for candidateKey, candidate := range m.stages {
		if candidateKey.root == canonical {
			key, state = candidateKey, candidate
			break
		}
	}
	if state == nil {
		m.mu.Unlock()
		return operation.StageReceipt{}, fmt.Errorf("no staging session is active for %s", canonical)
	}
	result := m.receipt(state, "aborted", "")
	delete(m.stages, key)
	m.mu.Unlock()
	if err := godiagnostics.InvalidateWithContext(ctx); err != nil {
		return result, err
	}
	return result, nil
}

func invalidateStagedView(ctx context.Context, cause error) error {
	return errors.Join(cause, godiagnostics.InvalidateWithContext(ctx))
}

func (m *stageManager) Drop(ctx context.Context, root, stepID string) (operation.StageReceipt, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, state, err := m.state(ctx, root)
	if err != nil {
		return operation.StageReceipt{}, err
	}
	drop := -1
	for i, step := range state.steps {
		if step.id == stepID {
			drop = i
		}
	}
	if drop < 0 {
		return operation.StageReceipt{}, fmt.Errorf("staged step %q does not exist", stepID)
	}
	base := state.baseline.ForkOriginal()
	current := base.Fork()
	survivors := make([]stageStep, 0, len(state.steps)-1)
	for i, step := range state.steps {
		if i == drop {
			continue
		}
		candidate := current.Fork()
		entry, ok := m.server.registry.LookupKey(step.key)
		if !ok {
			return operation.StageReceipt{}, fmt.Errorf("replay staged step %s: operation no longer registered", step.id)
		}
		raw := map[string]any{}
		if err := json.Unmarshal(step.params, &raw); err != nil {
			return operation.StageReceipt{}, err
		}
		call := operation.CallContext{Ctx: sourcebuf.WithBuffer(ctx, candidate), WorkDir: state.key.root, Project: state.project, Registry: m.server.registry, Service: m.server.service, InBatch: true, DeferImports: true, DeferVerification: true, AllowedLanguages: m.server.enabledLanguages}
		if _, err := m.server.registry.Dispatch(call, entry.Key, raw); err != nil {
			return operation.StageReceipt{}, invalidateStagedView(ctx, fmt.Errorf("replay staged step %s: %w", step.id, err))
		}
		if err := base.MergeReadSetFrom(candidate); err != nil {
			return operation.StageReceipt{}, invalidateStagedView(ctx, err)
		}
		step.paths = stageChangedPaths(current, candidate)
		current = candidate
		survivors = append(survivors, step)
	}
	if err := current.AssertDisk(); err != nil {
		return operation.StageReceipt{}, invalidateStagedView(ctx, err)
	}
	if current.DirtySourceBytes() > m.limits.dirtySourceBytes {
		return operation.StageReceipt{}, invalidateStagedView(ctx, stageLimitError("dirty_source_bytes", current.DirtySourceBytes(), m.limits.dirtySourceBytes))
	}
	if err := godiagnostics.SynchronizeWithContext(sourcebuf.WithBuffer(ctx, current)); err != nil {
		return operation.StageReceipt{}, invalidateStagedView(ctx, fmt.Errorf("synchronize replayed staging view: %w", err))
	}
	var requestBytes int64
	for _, survivor := range survivors {
		requestBytes += survivor.bytes
	}
	state.baseline, state.current, state.steps, state.requestBytes = base, current, survivors, requestBytes
	return m.receipt(state, "dropped", ""), nil
}
func (m *stageManager) Commit(ctx context.Context, root string, trusted, skipHooks bool) (operation.StageCommitResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key, state, err := m.state(ctx, root)
	if err != nil {
		return operation.StageCommitResult{}, err
	}
	verifyOptions := map[string]any{trustWorkspaceField: trusted}
	for _, step := range state.steps {
		var raw map[string]any
		if err := json.Unmarshal(step.params, &raw); err != nil {
			return operation.StageCommitResult{}, fmt.Errorf("decode staged verification options: %w", err)
		}
		if err := mergeBatchVerificationOptions(verifyOptions, raw); err != nil {
			return operation.StageCommitResult{}, err
		}
	}
	if skipHooks {
		verifyOptions["skip_hooks"] = true
	}
	before := state.workspaceBefore
	requestServer, candidate, paths, candidateErr := m.commitCandidate(ctx, key, state, trusted)
	if candidateErr != nil {
		return operation.StageCommitResult{}, invalidateStagedView(ctx, candidateErr)
	}
	published, publishErr := candidate.PublishRetained(ctx)
	if publishErr != nil {
		return stagePublicationFailure(ctx, key, state, before, published, publishErr)
	}
	delete(m.stages, key)
	response := &BatchResponse{Status: "ok", WorkspaceRoot: key.root, Results: stageBatchRows(state.steps)}
	if requestServer.goEnabled() {
		if err := godiagnostics.SynchronizeWithContext(sourcebuf.WithoutBuffer(ctx)); err != nil {
			response.Status = batchStatusAppliedWithDiagnostics
			response.Results = append(response.Results, BatchResult{Tool: semanticVerifyToolName, Status: "error", Error: fmt.Sprintf("synchronize diagnostics after staged publication: %v", err)})
			captureAppliedBatchDiff(response, key.root, before)
			result := stageCommitFromResponse(state, response, published)
			result.ApplicationState = "persisted"
			return result, &operation.StageCommitError{Result: result, Err: fmt.Errorf("synchronize diagnostics after staged publication: %w", err)}
		}
	}
	verified, verifyErr := requestServer.verifyFinishedBatch(sourcebuf.WithoutBuffer(ctx), response, before, state.verification, verifyOptions, paths)
	if verifyErr != nil {
		response.Status = batchStatusAppliedWithDiagnostics
		response.Results = append(response.Results, BatchResult{Tool: semanticVerifyToolName, Status: "error", Error: fmt.Sprintf("verify staged publication: %v", verifyErr)})
		captureAppliedBatchDiff(response, key.root, before)
		result := stageCommitFromResponse(state, response, published)
		result.ApplicationState = "persisted"
		return result, &operation.StageCommitError{Result: result, Err: fmt.Errorf("verify staged publication: %w", verifyErr)}
	}
	result := stageCommitFromResponse(state, verified, published)
	if result.Status == batchStatusAppliedWithDiagnostics {
		result.ApplicationState = "persisted"
		return result, &operation.StageCommitError{Result: result, Err: stageCommitResponseError(verified)}
	}
	return result, nil
}

func stagePublicationFailure(ctx context.Context, key stageKey, state *stageState, before map[string]string, published int, publishErr error) (operation.StageCommitResult, error) {
	if published == 0 {
		outcomeErr := invalidateStagedView(ctx, publishErr)
		result := stageCommitResult(state, "failed")
		result.ApplicationState = "not_applied"
		result.PublicationError = outcomeErr.Error()
		return result, &operation.StageCommitError{Result: result, Err: outcomeErr}
	}
	state.terminal = "failed_partial"
	result := stageCommitResult(state, "failed_partial")
	result.ApplicationState = "partially_published"
	result.PublishedFiles = published
	outcomeErr := publishErr
	if invalidateErr := godiagnostics.InvalidateWithContext(ctx); invalidateErr != nil {
		outcomeErr = errors.Join(outcomeErr, fmt.Errorf("invalidate diagnostics after partial staged publication: %w", invalidateErr))
	}
	if diff, diffErr := finalBatchDiff(key.root, before); diffErr != nil {
		outcomeErr = errors.Join(outcomeErr, fmt.Errorf("capture partial staged diff: %w", diffErr))
	} else {
		result.FinalDiff = diff
	}
	result.PublicationError = outcomeErr.Error()
	return result, &operation.StageCommitError{Result: result, Err: outcomeErr}
}

func (m *stageManager) commitCandidate(ctx context.Context, key stageKey, state *stageState, trusted bool) (*Server, *sourcebuf.Buffer, []string, error) {
	module, moduleErr := stageModule(key.root)
	if moduleErr != nil {
		return nil, nil, nil, fmt.Errorf("resolve staged module ownership before commit: %w", moduleErr)
	}
	if module != key.module {
		return nil, nil, nil, fmt.Errorf("staged module ownership changed before commit")
	}
	candidate := state.current.Fork()
	requestServer := m.server.requestView(key.root, trusted)
	paths, normalizeErr := requestServer.normalizeBufferedBatch(sourcebuf.WithBuffer(ctx, candidate), candidate, state.workspaceBefore, true)
	if normalizeErr != nil {
		return nil, nil, nil, normalizeErr
	}
	if candidate.DirtySourceBytes() > m.limits.dirtySourceBytes {
		return nil, nil, nil, stageLimitError("dirty_source_bytes", candidate.DirtySourceBytes(), m.limits.dirtySourceBytes)
	}
	for _, path := range candidate.ChangedPaths() {
		owner, ownerErr := stageModule(filepath.Dir(path))
		if ownerErr != nil {
			return nil, nil, nil, fmt.Errorf("resolve staged output ownership for %s: %w", path, ownerErr)
		}
		if owner != key.module {
			return nil, nil, nil, fmt.Errorf("staged output ownership changed for %s", path)
		}
	}
	return requestServer, candidate, paths, nil
}
func (m *stageManager) Execute(ctx context.Context, root string, entry operation.Entry, raw map[string]any, call operation.CallContext) (any, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if entry.Key == "stage_abort" {
		return nil, false, nil
	}
	if err := m.expireLocked(ctx); err != nil {
		return nil, true, err
	}
	state, found, ownerErr := m.ownerLocked(root)
	if ownerErr != nil {
		return nil, true, ownerErr
	}
	if !found {
		return nil, false, nil
	}
	if entry.Key == "stage_start" || entry.Key == "stage_drop" || entry.Key == "stage_abort" || entry.Key == "stage_commit" {
		return nil, false, nil
	}
	if state.terminal != "" {
		return nil, true, fmt.Errorf("staging session is terminal (%s); abort it", state.terminal)
	}
	canonical, err := stageCanonicalRoot(root)
	if err != nil {
		return nil, true, err
	}
	if canonical != state.key.root {
		return nil, true, fmt.Errorf("request workspace %s is owned by active staging workspace %s", canonical, state.key.root)
	}
	if !entry.BufferedBatch {
		return nil, true, fmt.Errorf("operation %q is unavailable while staging is active", entry.MCPName)
	}
	value, executeErr := m.executeStagedOperation(ctx, state, entry, raw, call)
	if executeErr != nil {
		return nil, true, executeErr
	}
	return value, true, nil
}

func (m *stageManager) executeStagedOperation(ctx context.Context, state *stageState, entry operation.Entry, raw map[string]any, call operation.CallContext) (any, error) {
	normalized, err := entry.NormalizedInput(raw)
	if err != nil {
		return nil, fmt.Errorf("normalize staged operation %q: %w", entry.MCPName, err)
	}
	raw = normalized
	supported, err := entry.SupportsBufferedBatch(call, raw)
	if err != nil || !supported {
		return nil, fmt.Errorf("operation %q is not buffer-safe while staging is active", entry.MCPName)
	}
	params, err := json.Marshal(raw)
	if err != nil {
		return nil, err
	}
	size := int64(len(entry.MCPName) + 1 + len(params))
	if len(state.steps) >= m.limits.steps {
		return nil, stageLimitError("steps", int64(len(state.steps)+1), int64(m.limits.steps))
	}
	if state.requestBytes+size > m.limits.requestBytes {
		return nil, stageLimitError("request_bytes", state.requestBytes+size, m.limits.requestBytes)
	}
	candidate := state.current.Fork()
	ctx = sourcebuf.WithBuffer(ctx, candidate)
	call.Ctx = ctx
	call.InBatch = true
	call.DeferImports = true
	call.DeferVerification = true
	value, err := m.server.registry.Dispatch(call, entry.Key, raw)
	if err != nil {
		return nil, invalidateStagedView(ctx, err)
	}
	if err := candidate.AssertDisk(); err != nil {
		return nil, invalidateStagedView(ctx, err)
	}
	if candidate.DirtySourceBytes() > m.limits.dirtySourceBytes {
		return nil, invalidateStagedView(ctx, stageLimitError("dirty_source_bytes", candidate.DirtySourceBytes(), m.limits.dirtySourceBytes))
	}
	base := state.baseline.ForkOriginal()
	if err := base.MergeReadSetFrom(candidate); err != nil {
		return nil, invalidateStagedView(ctx, err)
	}
	id := stageStepID(state.id, state.nextOrdinal, entry.Key, params)
	step := stageStep{id: id, key: entry.Key, tool: entry.MCPName, params: params, bytes: size, paths: stageChangedPaths(state.current, candidate)}
	state.current, state.baseline = candidate, base
	state.project = call.Project
	state.steps = append(state.steps, step)
	state.requestBytes += size
	state.nextOrdinal++
	return stagedOperationResult{Result: value, Receipt: m.receipt(state, "staged", id)}, nil
}

type stagedOperationResult struct {
	Result  any
	Receipt operation.StageReceipt
}

func stageStepID(stage string, ordinal int, key string, params []byte) string {
	h := sha256.New()
	_, _ = fmt.Fprintf(h, "%s\x00%d\x00%s\x00", stage, ordinal, key)
	_, _ = h.Write(params)
	return hex.EncodeToString(h.Sum(nil))
}
func stageChangedPaths(before, after *sourcebuf.Buffer) []string {
	prior := map[string]string{}
	for _, c := range before.Changes() {
		prior[c.Path] = string(c.After)
	}
	var paths []string
	for _, c := range after.Changes() {
		if prior[c.Path] != string(c.After) {
			paths = append(paths, c.Path)
		}
		delete(prior, c.Path)
	}
	for p := range prior {
		paths = append(paths, p)
	}
	slices.Sort(paths)
	return slices.Compact(paths)
}
func stageBatchRows(steps []stageStep) []BatchResult {
	rows := make([]BatchResult, len(steps))
	for i, s := range steps {
		rows[i] = BatchResult{Tool: s.tool, Status: "ok"}
	}
	return rows
}
func stageCommitResult(state *stageState, status string) operation.StageCommitResult {
	return operation.StageCommitResult{Status: status, WorkspaceRoot: state.key.root, Results: stageCommitRows(state.steps), PublishedFiles: 0, FinalDiff: ""}
}
func stageCommitRows(steps []stageStep) []operation.StageCommitRow {
	rows := make([]operation.StageCommitRow, len(steps))
	for i, s := range steps {
		rows[i] = operation.StageCommitRow{StepID: s.id, Tool: s.tool, Status: "ok"}
	}
	return rows
}
func stageCommitFromResponse(state *stageState, response *BatchResponse, published int) operation.StageCommitResult {
	result := stageCommitResult(state, response.Status)
	result.FinalDiff = response.FinalDiff
	result.PublishedFiles = published
	result.Verification = response.Verification
	if response.DiagnosticDelta != nil {
		result.DiagnosticDelta = &operation.StageDiagnosticDelta{Introduced: stageDiagnosticStrings(response.DiagnosticDelta.Introduced), Resolved: stageDiagnosticStrings(response.DiagnosticDelta.Resolved), NetDelta: response.DiagnosticDelta.NetDelta, Candidates: make([]operation.StageDiagnosticCandidate, 0, len(response.DiagnosticDelta.Introduced))}
		for _, message := range response.DiagnosticDelta.Introduced {
			result.DiagnosticDelta.Candidates = append(result.DiagnosticDelta.Candidates, operation.StageDiagnosticCandidate{Diagnostic: message, CandidateStepIDs: stageDiagnosticStepIDs(state.steps, response.Verification, message)})
		}
	}
	return result
}

func stageCommitResponseError(response *BatchResponse) error {
	for _, row := range response.Results {
		if row.Status == "error" && row.Error != "" {
			return errors.New(row.Error)
		}
	}
	return errors.New("staged publication completed with diagnostics")
}

func stageDiagnosticStrings(values []string) []string {
	if values == nil {
		return []string{}
	}
	return slices.Clone(values)
}

func stageDiagnosticStepIDs(steps []stageStep, verification *operation.VerifyRes, message string) []string {
	if verification == nil {
		return []string{}
	}
	ids := make(map[string]struct{})
	for _, diagnostic := range verification.Diagnostics {
		if diagnostic.Message != message || diagnostic.Location == nil {
			continue
		}
		path, err := pathutil.FilePathFromURI(diagnostic.Location.URI)
		if err != nil {
			continue
		}
		for _, step := range steps {
			if slices.Contains(step.paths, path) {
				ids[step.id] = struct{}{}
			}
		}
	}
	return stageDiagnosticStrings(slices.Sorted(maps.Keys(ids)))
}
func (m *stageManager) reset(ctx context.Context) error {
	m.mu.Lock()
	clear(m.stages)
	m.mu.Unlock()
	return godiagnostics.InvalidateWithContext(ctx)
}
