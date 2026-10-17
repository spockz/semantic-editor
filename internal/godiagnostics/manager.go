// Package godiagnostics owns shared gopls analysis sessions so diagnostics, rename, and AST refactoring synchronization observe the same revision.
package godiagnostics

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"io"
	"maps"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/spockz/semantic-editor/internal/backend/pathutil"
	"github.com/spockz/semantic-editor/internal/gocache"
	"github.com/spockz/semantic-editor/internal/lsp"
	"github.com/spockz/semantic-editor/internal/telemetry"

	"encoding/json"

	"github.com/spockz/semantic-editor/internal/sourcebuf"
	"github.com/spockz/semantic-editor/internal/sourcefiles"

	"hash"

	"golang.org/x/tools/go/packages"
)

// Session exposes synchronized Go analysis during one serialized operation.
type Session struct {
	ctx            context.Context
	Root           string
	session        *managedSession
	freshDiscovery bool
}

// ErrSourceChanged marks an optimistic source conflict that can recover on the next synchronization.
var ErrSourceChanged = errors.New("go analysis inputs changed")

type scopeLoadResult struct {
	scope diagnosticsScope
	fresh bool
}

type scopeRevision struct {
	discovery [32]byte
	boundary  [32]byte
}

func encodeStructuralGoInput(file *ast.File) []byte {
	encoded := append([]byte(file.Name.Name), 0)
	for _, spec := range file.Imports {
		encoded = append(encoded, spec.Path.Value...)
		encoded = append(encoded, 0)
	}
	for _, group := range file.Comments {
		for _, comment := range group.List {
			if strings.HasPrefix(comment.Text, "//go:") || strings.HasPrefix(comment.Text, "// +build") {
				encoded = append(encoded, comment.Text...)
				encoded = append(encoded, 0)
			}
		}
	}
	for _, spec := range file.Imports {
		importPath, err := strconv.Unquote(spec.Path.Value)
		if err != nil || importPath != "C" {
			continue
		}
		for _, group := range file.Comments {
			if group.End() < spec.Pos() {
				encoded = append(encoded, group.Text()...)
				encoded = append(encoded, 0)
			}
		}
	}
	return encoded
}

type fingerprintInputs struct {
	ctx      context.Context
	sources  map[string]fingerprintInput
	retained map[string]fingerprintInput
	boundary hash.Hash
}

type fingerprintInput struct {
	digest    [32]byte
	structure []byte
	boundary  []byte
	patterns  []embedPattern
	valid     bool
}

func (s *managedSession) syncDocument(ctx context.Context, path string, document selectedDocument) (bool, error) {
	previous, opened := s.documents[path]
	dirty := sourcebuf.Dirty(ctx, path)
	if opened && previous.digest == document.digest && (previous.saved || dirty) {
		return false, nil
	}
	diskInfo, err := os.Stat(path)
	if err != nil && (!dirty || !errors.Is(err, os.ErrNotExist)) {
		return false, fmt.Errorf("inspect synchronized Go source %q: %w", path, err)
	}
	uri := (&url.URL{Scheme: "file", Path: filepath.ToSlash(path)}).String()
	version := previous.version + 1
	if !opened {
		if err := s.client.Notify(ctx, "textDocument/didOpen", map[string]any{
			textDocumentKey: map[string]any{"uri": uri, "languageId": "go", "version": version, "text": string(document.content)},
		}); err != nil {
			return false, fmt.Errorf("open Go source %q in gopls: %w", path, err)
		}
	} else {
		if err := s.client.Notify(ctx, "textDocument/didChange", map[string]any{
			textDocumentKey:  map[string]any{"uri": uri, "version": version},
			"contentChanges": []map[string]any{{"text": string(document.content)}},
		}); err != nil {
			return false, fmt.Errorf("synchronize Go source %q in gopls: %w", path, err)
		}
		// A preserved disk identity and mtime can leave gopls's disk-content cache stale.
		if !dirty && previous.diskInfo != nil && diskInfo != nil &&
			os.SameFile(previous.diskInfo, diskInfo) && previous.diskInfo.ModTime().Equal(diskInfo.ModTime()) {
			if err := s.client.Notify(ctx, "textDocument/didSave", map[string]any{textDocumentKey: map[string]any{"uri": uri}}); err != nil {
				return false, fmt.Errorf("save externally synchronized Go source %q: %w", path, err)
			}
		}
	}
	s.documents[path] = documentSnapshot{version: version, digest: document.digest, saved: !dirty, published: previous.published || !dirty, diskInfo: diskInfo}
	return !previous.published && !dirty, nil
}

type incrementalDiagnosticsContextKey struct{}

// WithIncrementalDiagnostics opts into revision-validated diagnostic reuse.
func WithIncrementalDiagnostics(ctx context.Context) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, incrementalDiagnosticsContextKey{}, true)
}

// Document is an immutable source preimage tied to an open LSP document version.
type Document struct {
	Path    string
	URI     string
	Version int
	Content []byte
	Info    os.FileInfo
}

// Request sends a protocol request while the owning manager serializes the operation.
func (s *Session) Request(ctx context.Context, method string, params any) (json.RawMessage, error) {
	return s.session.client.Request(ctx, method, params)
}

// Snapshot returns the synchronized revision and rejects changes since synchronization.
func (s *Session) Snapshot(path string) (Document, error) {
	defer telemetry.StartPhase(s.ctx, "go.session.snapshot")()
	path, err := filepath.Abs(path)
	if err != nil {
		return Document{}, fmt.Errorf("resolve document: %w", err)
	}
	path, err = sourcebuf.CanonicalPath(s.ctx, path)
	if err != nil {
		return Document{}, fmt.Errorf("canonicalize document: %w", err)
	}
	snapshot, ok := s.session.documents[path]
	if !ok {
		return Document{}, fmt.Errorf("go source %q is outside the synchronized scope", path)
	}
	// #nosec G304 -- only retained Go source paths are read.
	content, err := sourcebuf.ReadFile(s.ctx, path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Document{}, fmt.Errorf("%w: read synchronized Go source: %w", ErrSourceChanged, err)
		}
		return Document{}, fmt.Errorf("read synchronized Go source: %w", err)
	}
	if sha256.Sum256(content) != snapshot.digest {
		return Document{}, fmt.Errorf("%w: Go source %q changed since synchronization", ErrSourceChanged, path)
	}
	info, err := os.Stat(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return Document{}, fmt.Errorf("stat source preimage: %w", err)
	}
	if err == nil && !info.Mode().IsRegular() {
		return Document{}, fmt.Errorf("%w: source preimage %q is not regular", ErrSourceChanged, path)
	}
	if errors.Is(err, os.ErrNotExist) {
		if bufferedPath, canonicalErr := sourcebuf.CanonicalPath(s.ctx, path); canonicalErr == nil && bufferedPath == path {
			return Document{Path: path, URI: (&url.URL{Scheme: "file", Path: filepath.ToSlash(path)}).String(), Version: snapshot.version, Content: content}, nil
		}
		return Document{}, fmt.Errorf("%w: source preimage %q was removed", ErrSourceChanged, path)
	}
	return Document{Info: info, Path: path, URI: (&url.URL{Scheme: "file", Path: filepath.ToSlash(path)}).String(), Version: snapshot.version, Content: content}, nil
}

// CanonicalSourcePath validates a source path against this session's retained source view.
func (s *Session) CanonicalSourcePath(path string) (string, error) {
	canonical, err := sourcebuf.CanonicalPath(s.ctx, path)
	if err != nil {
		return "", fmt.Errorf("canonicalize Go source path: %w", err)
	}
	return canonical, nil
}

// SessionManager owns one reusable gopls analysis session per Go module.
type SessionManager struct {
	mu                 sync.Mutex
	sessions           map[string]*managedSession
	scopes             map[string]diagnosticsScope
	fingerprintSources map[string]map[string]fingerprintInput
	closed             bool
}

// Close stops every session owned by this MCP instance.
func (m *SessionManager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return nil
	}
	m.closed = true
	var resultErr error
	for root, session := range m.sessions {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		err := closeManagedSession(ctx, session)
		cancel()
		if err != nil {
			resultErr = errors.Join(resultErr, fmt.Errorf("close Go diagnostics session for %q: %w", root, err))
		}
		delete(m.sessions, root)
	}
	m.scopes = nil
	m.fingerprintSources = nil
	return resultErr
}

// CheckDetailed synchronizes selected files before requesting current diagnostics.
func (m *SessionManager) CheckDetailed(ctx context.Context, root, executable string, environment []string, patterns ...string) ([]Diagnostic, error) {
	root, resolveErr := filepath.Abs(root)
	if resolveErr != nil {
		return nil, fmt.Errorf("resolve Go diagnostics root: %w", resolveErr)
	}
	root, resolveErr = filepath.EvalSymlinks(root)
	if resolveErr != nil {
		return nil, fmt.Errorf("canonicalize Go diagnostics root: %w", resolveErr)
	}
	var diagnostics []Diagnostic
	err := m.use(ctx, root, executable, environment, func(ctx context.Context, session *Session) error {
		var err error
		diagnostics, err = checkSessionDiagnostics(ctx, root, environment, patterns, session)
		return err
	})
	if err != nil {
		return nil, err
	}
	slices.SortFunc(diagnostics, func(a, b Diagnostic) int { return strings.Compare(a.String(), b.String()) })
	return diagnostics, nil
}

func checkSessionDiagnostics(ctx context.Context, root string, environment []string, patterns []string, session *Session) ([]Diagnostic, error) {
	managed := session.session
	resetDiagnosticRoot(managed, root)
	incremental := incrementalDiagnostics(ctx) && len(patterns) == 0
	files, findings, err := selectDiagnosticInputs(ctx, root, environment, patterns, session, incremental)
	if err != nil {
		return nil, err
	}
	seedFull := !incremental && len(patterns) == 0 && canSeedFullDiagnostics(ctx, root, environment, files, findings, session)
	requestFiles := files
	if incremental {
		requestFiles = cachedDiagnosticFiles(managed, managed.scope, files)
		if !session.freshDiscovery {
			findings, err = updatePackageFindingCache(ctx, root, environment, managed, requestFiles, findings)
			if err != nil {
				return nil, err
			}
		}
	}
	diagnostics, err := requestDiagnosticFiles(ctx, root, requestFiles, session, incremental || seedFull)
	if err != nil {
		return nil, err
	}
	diagnostics = append(diagnostics, findings...)
	if incremental {
		if err := appendCachedDiagnostics(managed, files, requestFiles, &diagnostics); err != nil {
			return nil, err
		}
	}
	if incremental || seedFull {
		managed.diagnosticInputDigests = currentDocumentDigests(managed.documents)
		if session.freshDiscovery || seedFull {
			managed.packageFindingCache = map[string][]Diagnostic{root: slices.Clone(findings)}
			managed.packageFindingsDirty = false
		}
	}
	if seedFull {
		telemetry.StartPhase(ctx, "go.diagnostics.seed_full")()
	}
	return diagnostics, nil
}

func resetDiagnosticRoot(session *managedSession, root string) {
	if session.diagnosticRoot == root {
		return
	}
	session.diagnosticCache = make(map[string]diagnosticCacheEntry)
	session.diagnosticRoot = root
}

func incrementalDiagnostics(ctx context.Context) bool {
	incremental, _ := ctx.Value(incrementalDiagnosticsContextKey{}).(bool)
	return incremental
}

func selectDiagnosticInputs(ctx context.Context, root string, environment []string, patterns []string, session *Session, incremental bool) ([]string, []Diagnostic, error) {
	if !incremental {
		return loadGoFiles(ctx, root, environment, sourcebuf.Overlay(ctx, ".go"), patterns...)
	}
	files, findings, err := selectPackageFilesWithPolicy(root, session.session.scope.packages, buildCacheFromEnvironment(environment), false)
	if err != nil {
		return nil, nil, err
	}
	if session.freshDiscovery {
		return files, findings, nil
	}
	if cached, ok := session.session.packageFindingCache[root]; ok {
		return files, slices.Clone(cached), nil
	}
	if !session.session.packageFindingsDirty {
		return files, findings, nil
	}
	_, findings, err = loadGoFiles(ctx, root, environment, sourcebuf.Overlay(ctx, ".go"), "./...")
	return files, findings, err
}

func updatePackageFindingCache(ctx context.Context, root string, environment []string, session *managedSession, requestFiles []string, findings []Diagnostic) ([]Diagnostic, error) {
	if !hasChangedInputDigest(session) {
		return findings, nil
	}
	session.packageFindingsDirty = true
	session.packageFindingCache = map[string][]Diagnostic{root: slices.Clone(findings)}
	updated, err := refreshAffectedPackageFindings(ctx, root, environment, session.scope, requestFiles, findings)
	if err != nil {
		return nil, err
	}
	session.packageFindingCache[root] = slices.Clone(updated)
	return updated, nil
}

func requestDiagnosticFiles(ctx context.Context, root string, files []string, session *Session, cacheResults bool) ([]Diagnostic, error) {
	defer telemetry.StartPhase(ctx, "go.diagnostics.collect")()
	if diagnosticWorkerLimit < 1 {
		return nil, errors.New("go diagnostic worker limit must be positive")
	}
	if len(files) == 0 {
		return nil, nil
	}
	pullCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	results := make([][]Diagnostic, len(files))
	jobs := make(chan int)
	var workers sync.WaitGroup
	var firstErr error
	var failure sync.Once
	for range min(diagnosticWorkerLimit, len(files)) {
		workers.Go(func() {
			for index := range jobs {
				if pullCtx.Err() != nil {
					return
				}
				found, err := requestDiagnostics(pullCtx, session.session.client, root, files[index])
				if err != nil {
					failure.Do(func() { firstErr = err; cancel() })
					return
				}
				results[index] = found
			}
		})
	}
submit:
	for index := range files {
		select {
		case jobs <- index:
		case <-pullCtx.Done():
			break submit
		}
	}
	close(jobs)
	workers.Wait()
	if firstErr != nil {
		return nil, firstErr
	}
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("collect Go diagnostics: %w", err)
	}
	var diagnostics []Diagnostic
	for index, path := range files {
		found := results[index]
		diagnostics = append(diagnostics, found...)
		if cacheResults {
			snapshot := session.session.documents[path]
			session.session.diagnosticCache[path] = diagnosticCacheEntry{digest: snapshot.digest, diagnostics: slices.Clone(found)}
		}
	}
	return diagnostics, nil
}

func appendCachedDiagnostics(session *managedSession, files, requestFiles []string, diagnostics *[]Diagnostic) error {
	for _, path := range files {
		if slices.Contains(requestFiles, path) {
			continue
		}
		entry, ok := session.diagnosticCache[path]
		if !ok {
			return fmt.Errorf("cached Go diagnostics for %q disappeared during synchronization", path)
		}
		*diagnostics = append(*diagnostics, entry.diagnostics...)
	}
	return nil
}

func cachedDiagnosticFiles(session *managedSession, scope diagnosticsScope, files []string) []string {
	var changed []string
	for path, snapshot := range session.diagnosticInputDigests {
		current, retained := session.documents[path]
		if !retained || current.digest != snapshot {
			changed = append(changed, path)
		}
	}
	for path, snapshot := range session.documents {
		previous, retained := session.diagnosticInputDigests[path]
		if !retained || previous != snapshot.digest {
			changed = append(changed, path)
		}
	}
	for _, path := range files {
		entry, ok := session.diagnosticCache[path]
		if !ok || entry.digest != session.documents[path].digest {
			changed = append(changed, path)
		}
	}
	changed = slices.Compact(slices.Sorted(slices.Values(changed)))
	if len(changed) == 0 {
		return nil
	}
	affected, complete := packageGraphClosure(scope.packages, changed)
	if !complete {
		return slices.Clone(files)
	}
	selected := make(map[string]struct{})
	for _, pkg := range affected {
		for _, path := range pkg.GoFiles {
			selected[path] = struct{}{}
		}
	}
	result := make([]string, 0, len(selected))
	for _, path := range files {
		if _, ok := selected[path]; ok {
			result = append(result, path)
		}
	}
	if len(result) == 0 {
		return slices.Clone(files)
	}
	return result
}

func packageGraphClosure(pkgs []*packages.Package, targets []string) (map[string]*packages.Package, bool) {
	byID := make(map[string]*packages.Package)
	byPackagePath := make(map[string][]string)
	owners := make(map[string][]string)
	for _, pkg := range pkgs {
		byID[pkg.ID] = pkg
		byPackagePath[pkg.PkgPath] = append(byPackagePath[pkg.PkgPath], pkg.ID)
		for _, path := range pkg.GoFiles {
			owners[path] = append(owners[path], pkg.ID)
		}
	}
	queue := make([]string, 0, len(targets))
	for _, path := range targets {
		ids := owners[path]
		if len(ids) == 0 {
			return nil, false
		}
		queue = append(queue, ids...)
	}
	reverse := make(map[string][]string)
	for _, pkg := range pkgs {
		for _, imported := range pkg.Imports {
			if imported != nil {
				reverse[imported.ID] = append(reverse[imported.ID], pkg.ID)
			}
		}
		if _, right, ok := strings.Cut(pkg.ID, " ["); ok {
			testPath := strings.TrimSuffix(strings.TrimSuffix(right, "]"), ".test")
			for _, baseID := range byPackagePath[testPath] {
				reverse[baseID] = append(reverse[baseID], pkg.ID)
			}
			if byID[testPath] != nil {
				reverse[testPath] = append(reverse[testPath], pkg.ID)
			}
		} else if pkg.Name == "main" && strings.HasSuffix(pkg.ID, ".test") {
			testPath := strings.TrimSuffix(pkg.ID, ".test")
			for _, baseID := range byPackagePath[testPath] {
				reverse[baseID] = append(reverse[baseID], pkg.ID)
			}
		}
	}
	affected := make(map[string]*packages.Package)
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		if _, exists := affected[id]; exists {
			continue
		}
		pkg := byID[id]
		if pkg == nil {
			return nil, false
		}
		affected[id] = pkg
		queue = append(queue, reverse[id]...)
	}
	return affected, len(affected) > 0
}

func hasChangedInputDigest(session *managedSession) bool {
	if len(session.diagnosticInputDigests) == 0 {
		return false
	}
	for path, snapshot := range session.diagnosticInputDigests {
		current, retained := session.documents[path]
		if !retained || current.digest != snapshot {
			return true
		}
	}
	for path, snapshot := range session.documents {
		previous, retained := session.diagnosticInputDigests[path]
		if !retained || previous != snapshot.digest {
			return true
		}
	}
	return false
}

func currentDocumentDigests(documents map[string]documentSnapshot) map[string][32]byte {
	digests := make(map[string][32]byte, len(documents))
	for path, snapshot := range documents {
		digests[path] = snapshot.digest
	}
	return digests
}

func refreshAffectedPackageFindings(ctx context.Context, root string, environment []string, scope diagnosticsScope, files []string, current []Diagnostic) ([]Diagnostic, error) {
	changedIDs := make(map[string]struct{})
	patterns := make(map[string]struct{})
	for _, pkg := range scope.packages {
		for _, path := range pkg.GoFiles {
			if !slices.Contains(files, path) {
				continue
			}
			changedIDs[pkg.ID] = struct{}{}
			directory, err := filepath.Rel(root, filepath.Dir(path))
			if err != nil || !filepath.IsLocal(directory) {
				_, refreshed, refreshErr := loadGoFiles(ctx, root, environment, sourcebuf.Overlay(ctx, ".go"), "./...")
				return refreshed, refreshErr
			}
			patterns["./"+filepath.ToSlash(directory)] = struct{}{}
		}
	}
	if len(patterns) == 0 {
		_, refreshed, err := loadGoFiles(ctx, root, environment, sourcebuf.Overlay(ctx, ".go"), "./...")
		return refreshed, err
	}
	selected := make([]string, 0, len(patterns))
	for pattern := range patterns {
		selected = append(selected, pattern)
	}
	slices.Sort(selected)
	_, refreshed, err := loadGoFiles(ctx, root, environment, sourcebuf.Overlay(ctx, ".go"), selected...)
	if err != nil {
		return nil, err
	}
	result := make([]Diagnostic, 0, len(current)+len(refreshed))
	for _, finding := range current {
		if finding.Source == "package" {
			if _, changed := changedIDs[finding.Path]; changed {
				continue
			}
		}
		result = append(result, finding)
	}
	return append(result, refreshed...), nil
}

func buildCacheFromEnvironment(environment []string) string {
	for _, entry := range environment {
		key, value, ok := strings.Cut(entry, "=")
		if ok && key == "GOCACHE" {
			return value
		}
	}
	return ""
}

// NewSessionManager creates an MCP-scoped Go analysis manager.
func NewSessionManager() *SessionManager {
	return &SessionManager{sessions: make(map[string]*managedSession), scopes: make(map[string]diagnosticsScope)}
}

// WithSessionManager attaches MCP-owned Go analysis state to a request.
func WithSessionManager(ctx context.Context, manager *SessionManager) context.Context {
	return context.WithValue(ctx, managerContextKey{}, manager)
}

// AffectedPackagePatternsWithContext returns the reverse dependency closure for Go source targets.
func AffectedPackagePatternsWithContext(ctx context.Context, root string, targets []string) ([]string, error) {
	full := []string{"./..."}
	if ctx == nil || len(targets) == 0 {
		return full, nil
	}
	root, err := canonicalPackageGraphRoot(root)
	if err != nil {
		return full, err
	}
	manager, selected, err := selectCachedScope(ctx, root)
	if err != nil {
		return full, err
	}
	if manager == nil || selected == nil {
		return full, nil
	}
	targetPaths, err := canonicalTargetPaths(root, targets)
	if err != nil {
		return full, err
	}
	return packagePatternsForTargets(root, *selected, targetPaths)
}

func selectCachedScope(ctx context.Context, root string) (*SessionManager, *diagnosticsScope, error) {
	manager, _ := ctx.Value(managerContextKey{}).(*SessionManager)
	if manager == nil {
		return nil, nil, nil
	}
	environment, err := gocache.Environment(ctx, root)
	if err != nil {
		return nil, nil, fmt.Errorf("prepare Go package graph environment: %w", err)
	}
	manager.mu.Lock()
	defer manager.mu.Unlock()
	loaded, err := manager.loadScope(ctx, root, environment)
	if err != nil {
		return nil, nil, fmt.Errorf("initialize Go package graph: %w", err)
	}
	return manager, &loaded.scope, nil
}

func canonicalPackageGraphRoot(root string) (string, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("resolve Go package graph root: %w", err)
	}
	canonical, err := filepath.EvalSymlinks(root)
	if err != nil {
		return "", fmt.Errorf("canonicalize Go package graph root: %w", err)
	}
	return canonical, nil
}

func canonicalTargetPaths(root string, targets []string) ([]string, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve Go package graph root: %w", err)
	}
	paths := make([]string, 0, len(targets))
	for _, target := range targets {
		path := target
		if !filepath.IsAbs(path) {
			path = filepath.Join(root, path)
		}
		path, err = filepath.Abs(path)
		if err != nil {
			return nil, fmt.Errorf("resolve Go package target: %w", err)
		}
		if canonical, evalErr := filepath.EvalSymlinks(path); evalErr == nil {
			path = canonical
		}
		paths = append(paths, path)
	}
	return paths, nil
}

func packagePatternsForTargets(root string, selected diagnosticsScope, targetPaths []string) ([]string, error) {
	full := []string{"./..."}
	affected, complete := packageGraphClosure(selected.packages, targetPaths)
	if !complete {
		return full, nil
	}
	patterns, complete := packagePatternsForPackages(root, selected, affected)
	if !complete || len(patterns) == 0 {
		return full, nil
	}
	result := make([]string, 0, len(patterns))
	for pattern := range patterns {
		result = append(result, pattern)
	}
	slices.Sort(result)
	return result, nil
}

func packagePatternsForPackages(root string, selected diagnosticsScope, affected map[string]*packages.Package) (map[string]struct{}, bool) {
	patterns := make(map[string]struct{})
	for _, pkg := range affected {
		if len(pkg.GoFiles) == 0 {
			return nil, false
		}
		localFiles := 0
		for _, path := range pkg.GoFiles {
			if IsSyntheticTestRunnerFile(pkg, path, selected.buildCache) {
				continue
			}
			directory := filepath.Dir(path)
			if !pathutil.PathWithin(root, directory) {
				return nil, false
			}
			relative, err := filepath.Rel(root, directory)
			if err != nil || !filepath.IsLocal(relative) {
				return nil, false
			}
			patterns["./"+filepath.ToSlash(relative)] = struct{}{}
			localFiles++
		}
		if localFiles == 0 && !strings.HasSuffix(pkg.ID, ".test") {
			return nil, false
		}
	}
	return patterns, true
}

// CheckDetailedWithContext reuses an attached manager or runs a one-shot check.
func CheckDetailedWithContext(ctx context.Context, root, executable string, environment []string) ([]Diagnostic, error) {
	return CheckDetailedWithPatternsWithContext(ctx, root, executable, environment, nil)
}

// CheckDetailedWithPatternsWithContext requests diagnostics for explicit package patterns when provided.
func CheckDetailedWithPatternsWithContext(ctx context.Context, root, executable string, environment []string, patterns []string) ([]Diagnostic, error) {
	if ctx != nil {
		if manager, ok := ctx.Value(managerContextKey{}).(*SessionManager); ok && manager != nil {
			return manager.CheckDetailed(ctx, root, executable, environment, patterns...)
		}
	}
	return CheckDetailedForPatterns(ctx, root, executable, environment, patterns...)
}

// UseWithContext runs a Go LSP operation on the MCP session or a temporary CLI session.
func UseWithContext(ctx context.Context, root, executable string, environment []string, action func(context.Context, *Session) error) (resultErr error) {
	if ctx != nil {
		if manager, ok := ctx.Value(managerContextKey{}).(*SessionManager); ok && manager != nil {
			return manager.use(ctx, root, executable, environment, action)
		}
	}
	manager := NewSessionManager()
	defer func() { resultErr = errors.Join(resultErr, manager.Close()) }()
	return manager.use(ctx, root, executable, environment, action)
}

// SynchronizeWithContext refreshes published Go sessions. Buffered revisions synchronize on demand before LSP queries.
func SynchronizeWithContext(ctx context.Context) error {
	if ctx == nil || sourcebuf.FromContext(ctx) != nil {
		return nil
	}
	return synchronizeSessionsWithContext(ctx, "", false)
}

// SynchronizeScopeWithContext restores retained Go sessions overlapping root to their current disk source view.
func SynchronizeScopeWithContext(ctx context.Context, root string) error {
	if ctx == nil || root == "" {
		return nil
	}
	diskCtx, cancel := context.WithTimeout(sourcebuf.WithoutBuffer(context.WithoutCancel(ctx)), 5*time.Second)
	defer cancel()
	return synchronizeSessionsWithContext(diskCtx, root, true)
}

func synchronizeSessionsWithContext(ctx context.Context, root string, scoped bool) error {
	manager, ok := ctx.Value(managerContextKey{}).(*SessionManager)
	if !ok || manager == nil {
		return nil
	}
	type retained struct {
		root, executable string
		environment      []string
	}
	manager.mu.Lock()
	sessions := make([]retained, 0, len(manager.sessions))
	for sessionRoot, session := range manager.sessions {
		if scoped && !pathutil.PathWithin(sessionRoot, root) && !pathutil.PathWithin(root, sessionRoot) {
			continue
		}
		sessions = append(sessions, retained{root: sessionRoot, executable: session.executable, environment: slices.Clone(session.environment)})
	}
	manager.mu.Unlock()
	var resultErr error
	for _, session := range sessions {
		if err := manager.use(ctx, session.root, session.executable, session.environment, func(context.Context, *Session) error { return nil }); err != nil {
			if scoped {
				manager.mu.Lock()
				discardErr := manager.discardOverlappingScopes(session.root, nil)
				manager.mu.Unlock()
				resultErr = errors.Join(resultErr, fmt.Errorf("synchronize Go refactoring state for %s: %w", root, errors.Join(err, discardErr)))
				continue
			}
			return fmt.Errorf("synchronize Go refactoring state: %w", err)
		}
	}
	return resultErr
}

// EnsureSessionManager reuses MCP ownership or creates a temporary batch session owner.
func EnsureSessionManager(ctx context.Context) (context.Context, func() error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if manager, ok := ctx.Value(managerContextKey{}).(*SessionManager); ok && manager != nil {
		return ctx, func() error { return nil }
	}
	manager := NewSessionManager()
	return WithSessionManager(ctx, manager), manager.Close
}

// InvalidateWithContext discards retained analysis after failed source publication.
func InvalidateWithContext(ctx context.Context) error {
	if ctx == nil {
		return nil
	}
	manager, ok := ctx.Value(managerContextKey{}).(*SessionManager)
	if !ok || manager == nil {
		return nil
	}
	manager.mu.Lock()
	defer manager.mu.Unlock()
	var resultErr error
	for root := range manager.sessions {
		resultErr = errors.Join(resultErr, manager.discard(root, nil))
	}
	return resultErr
}

type documentSnapshot struct {
	saved     bool
	published bool
	version   int
	digest    [32]byte
	diskInfo  os.FileInfo
}

type diagnosticsScope struct {
	root        string
	files       []string
	modules     []string
	packages    []*packages.Package
	buildCache  string
	manifests   map[string]manifestState
	fingerprint scopeRevision
}

func (m *SessionManager) loadScope(ctx context.Context, root string, environment []string) (scopeLoadResult, error) {
	key := root + "\x00" + strings.Join(environment, "\x00")
	if scope, ok := m.scopes[key]; ok {
		if err := bindCachedScopeInputs(ctx, scope); err != nil {
			return scopeLoadResult{}, err
		}
		fingerprint, err := m.fingerprintScope(ctx, key, scope)
		if err == nil && fingerprint == scope.fingerprint {
			return scopeLoadResult{scope: scope}, nil
		}
	}
	scope, err := loadManagedScope(ctx, root, environment)
	if err != nil {
		return scopeLoadResult{}, err
	}
	fingerprint, err := m.fingerprintScope(ctx, key, scope)
	if err != nil {
		return scopeLoadResult{}, err
	}
	scope.fingerprint = fingerprint
	if m.scopes == nil {
		m.scopes = make(map[string]diagnosticsScope)
	}
	m.scopes[key] = scope
	return scopeLoadResult{scope: scope, fresh: true}, nil
}

func bindCachedScopeInputs(ctx context.Context, scope diagnosticsScope) error {
	defer telemetry.StartPhase(ctx, "go.scope.bind_inputs")()
	buffer := sourcebuf.FromContext(ctx)
	if buffer == nil {
		return nil
	}
	if err := buffer.AuthorizeWrites(ctx, scope.root); err != nil {
		return fmt.Errorf("authorize cached Go analysis scope: %w", err)
	}
	if err := sourcebuf.ObserveTree(ctx, scope.root, ".go"); err != nil {
		return fmt.Errorf("bind cached Go analysis scope: %w", err)
	}
	bound := map[string]bool{scope.root: true}
	if _, err := bindScopeInputs(ctx, scope, bound); err != nil {
		return err
	}
	if err := buffer.AssertDisk(); err != nil {
		return fmt.Errorf("validate cached Go analysis scope: %w", err)
	}
	return nil
}

func (m *SessionManager) discard(root string, cause error) error {
	session := m.sessions[root]
	delete(m.sessions, root)
	if session == nil {
		return cause
	}
	if err := session.client.Close(); err != nil {
		return errors.Join(cause, fmt.Errorf("discard Go diagnostics session for %q: %w", root, err))
	}
	return cause
}

func (m *SessionManager) use(ctx context.Context, root, executable string, environment []string, action func(context.Context, *Session) error) error {
	defer telemetry.StartPhase(ctx, "go.session.use")()
	if m == nil {
		return errors.New("gopls session manager is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	sessionCtx, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()
	root, executableInfo, err := validateManagedRequest(root, executable)
	if err != nil {
		return err
	}
	lockWait := telemetry.StartPhase(ctx, "go.session.lock_wait")
	m.mu.Lock()
	lockWait()
	defer m.mu.Unlock()
	if m.closed {
		return errors.New("gopls session manager is closed")
	}
	loaded, err := m.loadScope(sessionCtx, root, environment)
	if err != nil {
		return m.discardOverlappingScopes(root, err)
	}
	scope := loaded.scope
	if err = validateBufferedScope(sessionCtx, scope); err != nil {
		return err
	}
	session, err := m.prepareSession(sessionCtx, root, executable, executableInfo, environment, scope)
	if err != nil {
		return err
	}
	if err := action(sessionCtx, &Session{ctx: sessionCtx, Root: scope.root, session: session, freshDiscovery: loaded.fresh}); err != nil {
		if errors.Is(err, ErrSourceChanged) {
			return err
		}
		return m.discard(scope.root, err)
	}
	if err := session.syncDocuments(sessionCtx, scope.files); err != nil {
		return m.discard(scope.root, err)
	}
	session.files = slices.Clone(scope.files)
	if err := sessionCtx.Err(); err != nil {
		return m.discard(scope.root, fmt.Errorf("gopls session: %w", err))
	}
	return nil
}

func (m *SessionManager) prepareSession(ctx context.Context, root, executable string, executableInfo os.FileInfo, environment []string, scope diagnosticsScope) (*managedSession, error) {
	session := m.sessions[scope.root]
	if session != nil && !managedSessionMatches(session, executable, executableInfo, environment, scope) {
		delete(m.sessions, scope.root)
		closeCtx, closeCancel := context.WithTimeout(ctx, 5*time.Second)
		closeErr := closeManagedSession(closeCtx, session)
		closeCancel()
		if closeErr != nil {
			return nil, fmt.Errorf("replace stale Go LSP session for %q: %w", root, closeErr)
		}
		session = nil
	}
	metadataErrorTransition := session != nil && diagnosticGraphErrorTransition(session.scope, scope)
	reloadWorkspaceFolder := metadataErrorTransition && session.workspaceFolderChanges
	if metadataErrorTransition && !session.workspaceFolderChanges {
		delete(m.sessions, scope.root)
		closeCtx, closeCancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		closeErr := closeManagedSession(closeCtx, session)
		closeCancel()
		if closeErr != nil {
			return nil, fmt.Errorf("restart Go LSP without workspace-folder recovery for %q: %w", root, closeErr)
		}
		session = nil
	}
	if session == nil {
		started, err := startDiagnosticsClient(ctx, scope.root, executable, environment)
		if err != nil {
			return nil, err
		}
		session = &managedSession{
			client: started.client, workspaceFolderChanges: started.workspaceFolderChanges,
			executable: executable, executableInfo: executableInfo, environment: slices.Clone(environment),
			files: slices.Clone(scope.files), documents: make(map[string]documentSnapshot, len(scope.files)),
			manifests: scope.manifests, diagnosticCache: make(map[string]diagnosticCacheEntry), diagnosticInputDigests: make(map[string][32]byte),
		}
		if m.sessions == nil {
			m.sessions = make(map[string]*managedSession)
		}
		m.sessions[scope.root] = session
	}
	if diagnosticScopeRequiresFullRefresh(session.scope, scope) {
		session.diagnosticCache = make(map[string]diagnosticCacheEntry)
		session.packageFindingCache = make(map[string][]Diagnostic)
		session.packageFindingsDirty = false
		session.diagnosticInputDigests = nil
	}
	session.scope = scope
	if session.diagnosticCache == nil {
		session.diagnosticCache = make(map[string]diagnosticCacheEntry)
	}
	if err := session.syncDocuments(ctx, scope.files); err != nil {
		return nil, m.discard(scope.root, err)
	}
	if reloadWorkspaceFolder {
		reloadCtx, reloadCancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		reloadErr := session.reloadWorkspaceFolder(reloadCtx, scope.root)
		reloadCancel()
		if reloadErr != nil {
			return nil, m.discard(scope.root, fmt.Errorf("reload Go workspace metadata for %q: %w", root, reloadErr))
		}
	}
	session.files = slices.Clone(scope.files)
	return session, nil
}

func validateManagedRequest(root, executable string) (string, os.FileInfo, error) {
	canonical, err := filepath.Abs(root)
	if err != nil {
		return "", nil, fmt.Errorf("resolve Go LSP root: %w", err)
	}
	canonical, err = filepath.EvalSymlinks(canonical)
	if err != nil {
		return "", nil, fmt.Errorf("canonicalize Go LSP root: %w", err)
	}
	info, err := os.Stat(canonical)
	if err != nil {
		return "", nil, fmt.Errorf("stat Go LSP root: %w", err)
	}
	if !info.IsDir() {
		return "", nil, fmt.Errorf("gopls root %q is not a directory", canonical)
	}
	if executable == "" {
		return "", nil, errors.New("gopls executable is required")
	}
	executableInfo, err := os.Stat(executable)
	if err != nil {
		return "", nil, fmt.Errorf("stat gopls executable: %w", err)
	}
	return canonical, executableInfo, nil
}

func (m *SessionManager) discardOverlappingScopes(root string, cause error) error {
	for retainedRoot := range m.sessions {
		if pathutil.PathWithin(retainedRoot, root) || pathutil.PathWithin(root, retainedRoot) {
			cause = m.discard(retainedRoot, cause)
		}
	}
	return cause
}

func (m *SessionManager) fingerprintScope(ctx context.Context, key string, scope diagnosticsScope) (scopeRevision, error) {
	inputs := fingerprintInputs{sources: make(map[string]fingerprintInput), retained: m.fingerprintSources[key]}
	revision, err := scopeFingerprint(ctx, scope, inputs)
	if err != nil {
		return scopeRevision{}, err
	}
	if m.fingerprintSources == nil {
		m.fingerprintSources = make(map[string]map[string]fingerprintInput)
	}
	m.fingerprintSources[key] = inputs.sources
	return revision, nil
}

func validateBufferedScope(ctx context.Context, scope diagnosticsScope) error {
	buffer := sourcebuf.FromContext(ctx)
	if buffer == nil {
		return nil
	}
	for path := range scope.manifests {
		if _, err := sourcebuf.ReadFile(ctx, path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		if filepath.Base(path) == "go.mod" {
			if err := sourcebuf.ObserveTree(ctx, filepath.Dir(path), ".go"); err != nil {
				return err
			}
		}
	}
	return buffer.AssertDisk()
}

func managedSessionMatches(session *managedSession, executable string, executableInfo os.FileInfo, environment []string, scope diagnosticsScope) bool {
	return session.executable == executable &&
		os.SameFile(session.executableInfo, executableInfo) &&
		session.executableInfo.Size() == executableInfo.Size() &&
		session.executableInfo.ModTime().Equal(executableInfo.ModTime()) &&
		slices.Equal(session.environment, environment) &&
		maps.Equal(session.manifests, scope.manifests)
}

func (s *managedSession) syncDocuments(ctx context.Context, files []string) error {
	defer telemetry.StartPhase(ctx, "go.session.sync_documents")()
	initialSync := !s.documentsInitialized
	selected, err := selectedDocuments(ctx, files)
	if err != nil {
		return err
	}
	changedFiles, err := s.closeRemovedDocuments(ctx, selected)
	if err != nil {
		return err
	}
	changedFiles, err = s.syncSelectedDocuments(ctx, files, selected, changedFiles, initialSync)
	if err != nil {
		return err
	}
	s.documentsInitialized = true
	if len(changedFiles) > 0 {
		if err := notifyWatchedFileChanges(ctx, s.client, changedFiles); err != nil {
			return fmt.Errorf("notify changed Go file membership: %w", err)
		}
	}
	return nil
}

type selectedDocument struct {
	content []byte
	digest  [32]byte
}

func selectedDocuments(ctx context.Context, files []string) (map[string]selectedDocument, error) {
	selected := make(map[string]selectedDocument, len(files))
	for _, path := range files {
		// #nosec G304 -- packages.Load selected this source from the project root.
		content, err := sourcebuf.ReadFile(ctx, path)
		if err != nil {
			return nil, fmt.Errorf("read Go source %q: %w", path, err)
		}
		selected[path] = selectedDocument{content: content, digest: sha256.Sum256(content)}
	}
	return selected, nil
}

func (s *managedSession) closeRemovedDocuments(ctx context.Context, selected map[string]selectedDocument) ([]map[string]any, error) {
	var changedFiles []map[string]any
	var removed []string
	for path := range s.documents {
		if _, retained := selected[path]; retained {
			continue
		}
		removed = append(removed, path)
	}
	slices.Sort(removed)
	for _, path := range removed {
		previous := s.documents[path]
		uri := (&url.URL{Scheme: "file", Path: filepath.ToSlash(path)}).String()
		if err := s.client.Notify(ctx, "textDocument/didClose", map[string]any{textDocumentKey: map[string]any{"uri": uri}}); err != nil {
			return nil, fmt.Errorf("close removed Go source %q in gopls: %w", path, err)
		}
		if s.documentsInitialized && previous.published {
			changedFiles = append(changedFiles, map[string]any{"uri": uri, "type": 3})
		}
		delete(s.documents, path)
		delete(s.diagnosticCache, path)
	}
	return changedFiles, nil
}

func (s *managedSession) syncSelectedDocuments(ctx context.Context, files []string, selected map[string]selectedDocument, changedFiles []map[string]any, initialSync bool) ([]map[string]any, error) {
	for _, path := range files {
		created, err := s.syncDocument(ctx, path, selected[path])
		if err != nil {
			return nil, err
		}
		if created && !initialSync {
			uri := (&url.URL{Scheme: "file", Path: filepath.ToSlash(path)}).String()
			changedFiles = append(changedFiles, map[string]any{"uri": uri, "type": 1})
		}
	}
	return changedFiles, nil
}

func notifyWatchedFileChanges(ctx context.Context, client *lsp.Client, changes []map[string]any) error {
	return client.Notify(ctx, "workspace/didChangeWatchedFiles", map[string]any{"changes": changes})
}

func (s *managedSession) reloadWorkspaceFolder(ctx context.Context, root string) error {
	uri := (&url.URL{Scheme: "file", Path: filepath.ToSlash(root)}).String()
	folder := map[string]any{"uri": uri, "name": filepath.Base(root)}
	if err := s.client.Notify(ctx, "workspace/didChangeWorkspaceFolders", map[string]any{
		"event": map[string]any{"removed": []map[string]any{folder}, "added": []map[string]any{}},
	}); err != nil {
		return fmt.Errorf("remove Go workspace folder %q: %w", root, err)
	}
	if err := s.client.Notify(ctx, "workspace/didChangeWorkspaceFolders", map[string]any{
		"event": map[string]any{"removed": []map[string]any{}, "added": []map[string]any{folder}},
	}); err != nil {
		return fmt.Errorf("restore Go workspace folder %q: %w", root, err)
	}
	return nil
}

type managerContextKey struct{}

type manifestState struct {
	exists bool
	digest [32]byte
}

type managedSession struct {
	client                 *lsp.Client
	workspaceFolderChanges bool
	executableInfo         os.FileInfo
	executable             string
	environment            []string
	files                  []string
	documents              map[string]documentSnapshot
	documentsInitialized   bool
	manifests              map[string]manifestState
	scope                  diagnosticsScope
	diagnosticCache        map[string]diagnosticCacheEntry
	diagnosticRoot         string
	diagnosticInputDigests map[string][32]byte
	packageFindingCache    map[string][]Diagnostic
	packageFindingsDirty   bool
}

type diagnosticCacheEntry struct {
	digest      [32]byte
	diagnostics []Diagnostic
}

func readManifestState(root string) (map[string]manifestState, error) {
	state := make(map[string]manifestState, 2)
	for _, name := range []string{"go.mod", "go.sum"} {
		path := filepath.Join(root, name)
		input, err := readInputState(path)
		if err != nil {
			return nil, err
		}
		state[path] = input
	}
	return state, nil
}

func closeManagedSession(ctx context.Context, session *managedSession) error {
	var resultErr error
	if _, err := session.client.Request(ctx, "shutdown", map[string]any{}); err != nil && !errors.Is(err, lsp.ErrClosed) {
		resultErr = errors.Join(resultErr, fmt.Errorf("shutdown gopls diagnostics session: %w", err))
	}
	if err := session.client.Notify(ctx, "exit", nil); err != nil && !errors.Is(err, lsp.ErrClosed) {
		resultErr = errors.Join(resultErr, fmt.Errorf("exit gopls diagnostics session: %w", err))
	}
	if err := session.client.Close(); err != nil && !errors.Is(err, lsp.ErrClosed) {
		resultErr = errors.Join(resultErr, fmt.Errorf("close gopls diagnostics session: %w", err))
	}
	return resultErr
}

func loadManagedScope(ctx context.Context, root string, environment []string) (diagnosticsScope, error) {
	defer telemetry.StartPhase(ctx, "go.scope.load")()
	buffer := sourcebuf.FromContext(ctx)
	bound := make(map[string]bool)
	for {
		scope, err := discoverManagedScope(ctx, root, environment)
		if err != nil {
			return diagnosticsScope{}, err
		}
		scope.buildCache = buildCacheFromEnvironment(environment)
		if buffer == nil {
			return scope, nil
		}
		bound[scope.root] = true
		reload, err := bindScopeInputs(ctx, scope, bound)
		if err != nil {
			return diagnosticsScope{}, err
		}
		if err := buffer.AssertDisk(); err != nil {
			return diagnosticsScope{}, err
		}
		if !reload {
			return scope, nil
		}
		if err := ctx.Err(); err != nil {
			return diagnosticsScope{}, fmt.Errorf("bind Go analysis scope: %w", err)
		}
	}
}

func bindScopeInputs(ctx context.Context, scope diagnosticsScope, bound map[string]bool) (bool, error) {
	reload := false
	for path, expected := range scope.manifests {
		data, err := sourcebuf.ReadFile(ctx, path)
		actual := manifestState{}
		if err == nil {
			actual = manifestState{exists: true, digest: sha256.Sum256(data)}
		} else if !errors.Is(err, os.ErrNotExist) {
			return false, fmt.Errorf("bind Go analysis input: %w", err)
		}
		if actual != expected {
			reload = true
		}
		if filepath.Base(path) == "go.mod" && !bound[filepath.Dir(path)] {
			if err := sourcebuf.ObserveTree(ctx, filepath.Dir(path), ".go"); err != nil {
				return false, err
			}
			bound[filepath.Dir(path)] = true
			reload = true
		}
	}
	return reload, nil
}

func readInputState(path string) (manifestState, error) {
	// #nosec G304 -- paths come from Go package metadata or fixed module manifest names.
	content, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return manifestState{}, nil
	}
	if err != nil {
		return manifestState{}, fmt.Errorf("read Go diagnostics input %q: %w", path, err)
	}
	return manifestState{exists: true, digest: sha256.Sum256(content)}, nil
}

func discoverManagedScope(ctx context.Context, root string, environment []string) (diagnosticsScope, error) {
	moduleRoot, err := resolveDiagnosticsModule(ctx, root, environment)
	if err != nil {
		return diagnosticsScope{}, err
	}
	loaded, err := loadDiagnosticsPackages(ctx, moduleRoot, environment)
	if err != nil {
		return diagnosticsScope{}, err
	}
	return assembleDiagnosticsScope(ctx, moduleRoot, loaded)
}

func resolveDiagnosticsModule(ctx context.Context, root string, environment []string) (string, error) {
	defer telemetry.StartPhase(ctx, "go.scope.resolve_module")()
	cmd := exec.CommandContext(ctx, "go", "env", "GOMOD")
	cmd.Dir, cmd.Env = root, environment
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := errors.AsType[*exec.ExitError](err); ok {
			if detail := strings.TrimSpace(string(exitErr.Stderr)); detail != "" {
				return "", fmt.Errorf("resolve Go diagnostics module: %w: %s", err, detail)
			}
		}
		return "", fmt.Errorf("resolve Go diagnostics module: %w", err)
	}
	moduleFile := strings.TrimSpace(string(output))
	if moduleFile == "" || moduleFile == os.DevNull {
		return "", errors.New("go diagnostics require an enclosing module")
	}
	moduleRoot := filepath.Dir(moduleFile)
	if err := bindMainManifests(ctx, moduleRoot); err != nil {
		return "", err
	}
	return moduleRoot, nil
}

func bindMainManifests(ctx context.Context, moduleRoot string) error {
	buffer := sourcebuf.FromContext(ctx)
	if buffer == nil {
		return nil
	}
	for _, name := range []string{"go.mod", "go.sum"} {
		if _, err := sourcebuf.ReadFile(ctx, filepath.Join(moduleRoot, name)); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("bind main Go manifest: %w", err)
		}
	}
	return buffer.AuthorizeWrites(ctx, moduleRoot)
}

func loadDiagnosticsPackages(ctx context.Context, moduleRoot string, environment []string) ([]*packages.Package, error) {
	defer telemetry.StartPhase(ctx, "go.packages.load_scope")()
	loaded, err := packages.Load(&packages.Config{
		Context: ctx, Dir: moduleRoot, Env: environment, Overlay: sourcebuf.Overlay(ctx, ".go"),
		Mode:  packages.NeedName | packages.NeedFiles | packages.NeedModule | packages.NeedImports | packages.NeedDeps | packages.NeedEmbedFiles,
		Tests: true, BuildFlags: []string{"-mod=readonly"},
	}, "./...")
	if err != nil {
		return nil, fmt.Errorf("load Go diagnostics session scope: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("load Go diagnostics session scope: %w", err)
	}
	return loaded, nil
}

func assembleDiagnosticsScope(ctx context.Context, moduleRoot string, loaded []*packages.Package) (diagnosticsScope, error) {
	fileSet := make(map[string]struct{})
	manifestRoots := map[string]struct{}{moduleRoot: {}}
	auxiliary := make(map[string]manifestState)
	var visitErr error
	packages.Visit(loaded, func(pkg *packages.Package) bool {
		if visitErr != nil {
			return false
		}
		if pkg.Module == nil {
			return true
		}
		module := pkg.Module
		if module.Replace != nil {
			module = module.Replace
		}
		if module.Version != "" || module.Dir == "" {
			return true
		}
		manifestRoots[module.Dir] = struct{}{}
		for _, path := range pkg.GoFiles {
			canonicalPath, err := canonicalGoSourcePath(ctx, path)
			if err != nil {
				visitErr = err
				return false
			}
			fileSet[canonicalPath] = struct{}{}
		}
		for _, path := range append(slices.Clone(pkg.OtherFiles), pkg.EmbedFiles...) {
			state, err := readInputState(path)
			if err != nil {
				visitErr = err
				return false
			}
			auxiliary[path] = state
		}
		return true
	}, nil)
	if visitErr != nil {
		return diagnosticsScope{}, visitErr
	}
	if err := addOverlaySources(ctx, moduleRoot, fileSet); err != nil {
		return diagnosticsScope{}, err
	}
	scope := diagnosticsScope{root: moduleRoot, manifests: auxiliary, packages: loaded}
	for path := range fileSet {
		scope.files = append(scope.files, path)
	}
	slices.Sort(scope.files)
	for directory := range manifestRoots {
		scope.modules = append(scope.modules, directory)
		state, err := readManifestState(directory)
		if err != nil {
			return diagnosticsScope{}, err
		}
		maps.Copy(scope.manifests, state)
	}
	slices.Sort(scope.modules)
	return scope, nil
}

func scopeFingerprint(ctx context.Context, scope diagnosticsScope, inputs fingerprintInputs) (scopeRevision, error) {
	defer telemetry.StartPhase(ctx, "go.scope.fingerprint")()
	hash := sha256.New()
	boundary := sha256.New()
	write := func(value string) {
		_, _ = hash.Write([]byte(value + "\x00"))
		_, _ = boundary.Write([]byte(value + "\x00"))
	}
	overlays := sourcebuf.Overlay(ctx, ".go")
	inputs.ctx = ctx
	inputs.boundary = boundary
	embedPatterns, err := scopeEmbedPatterns(ctx, scope.files, inputs)
	if err != nil {
		return scopeRevision{}, err
	}
	defer telemetry.StartPhase(ctx, "go.scope.hash_inputs")()
	seen := make(map[string]struct{})
	for _, moduleRoot := range scope.modules {
		write(moduleRoot)
		if err := hashModuleInputs(hash, moduleRoot, scope, overlays, embedPatterns, seen, write, inputs); err != nil {
			return scopeRevision{}, err
		}
		if err := hashOverlayInputs(hash, moduleRoot, overlays, seen, write, inputs); err != nil {
			return scopeRevision{}, err
		}
	}
	if err := hashManifestInputs(scope.manifests, write); err != nil {
		return scopeRevision{}, err
	}
	var fingerprint scopeRevision
	copy(fingerprint.discovery[:], hash.Sum(nil))
	copy(fingerprint.boundary[:], boundary.Sum(nil))
	return fingerprint, nil
}

func hashModuleInputs(hash interface{ Write([]byte) (int, error) }, moduleRoot string, scope diagnosticsScope, overlays map[string][]byte, embedPatterns map[string][]embedPattern, seen map[string]struct{}, write func(string), inputs fingerprintInputs) error {
	return filepath.WalkDir(moduleRoot, func(path string, entry os.DirEntry, walkErr error) error {
		return hashModuleWalkEntry(hash, moduleRoot, path, entry, walkErr, scope, overlays, embedPatterns, seen, write, inputs)
	})
}

func hashModuleWalkEntry(hash interface{ Write([]byte) (int, error) }, moduleRoot, path string, entry os.DirEntry, walkErr error, scope diagnosticsScope, overlays map[string][]byte, embedPatterns map[string][]embedPattern, seen map[string]struct{}, write func(string), inputs fingerprintInputs) error {
	if walkErr != nil {
		return fmt.Errorf("inspect Go module structure %q: %w", path, walkErr)
	}
	if entry.IsDir() {
		if path != moduleRoot && (entry.Name() == ".git" || entry.Name() == ".scratch" || entry.Name() == "node_modules" || sourcefiles.IsBenchmarkDataDirectory(path)) {
			return filepath.SkipDir
		}
		return nil
	}
	if !fingerprintInputSelected(path, scope, embedPatterns) {
		return nil
	}
	info, err := entry.Info()
	if err != nil {
		return fmt.Errorf("inspect Go module input %q: %w", path, err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return hashAnalysisInputSymlink(hash, moduleRoot, path, overlays, seen, write, inputs)
	}
	if !info.Mode().IsRegular() {
		return nil
	}
	content, exists := overlays[path]
	if exists && content == nil {
		return nil
	}
	relative, err := filepath.Rel(moduleRoot, path)
	if err != nil {
		return fmt.Errorf("resolve Go module input %q: %w", path, err)
	}
	seen[path] = struct{}{}
	write(filepath.ToSlash(relative))
	if exists {
		return hashStructuralInput(hash, path, content, inputs)
	}
	content, err = os.ReadFile(path) // #nosec G304 -- path is enumerated below the selected Go module root.
	if err != nil {
		return fmt.Errorf("read Go module input %q: %w", path, err)
	}
	return hashStructuralInput(hash, path, content, inputs)
}

func hashAnalysisInputSymlink(hash interface{ Write([]byte) (int, error) }, moduleRoot, alias string, overlays map[string][]byte, seen map[string]struct{}, write func(string), inputs fingerprintInputs) error {
	target, err := filepath.EvalSymlinks(alias)
	if err != nil {
		return fmt.Errorf("resolve Go source symlink %q: %w", alias, err)
	}
	info, err := os.Stat(target)
	if err != nil {
		return fmt.Errorf("inspect Go source symlink target %q: %w", alias, err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("go source symlink %q has unsupported target %q", alias, target)
	}
	relative, err := filepath.Rel(moduleRoot, alias)
	if err != nil {
		return fmt.Errorf("resolve Go source symlink %q: %w", alias, err)
	}
	seen[alias] = struct{}{}
	write(filepath.ToSlash(relative))
	write(target)
	content, exists := overlays[alias]
	if !exists {
		content, exists = overlays[target]
	}
	if exists {
		if content == nil {
			write("deleted-overlay")
			return nil
		}
		return hashStructuralInput(hash, alias, content, inputs)
	}
	content, err = os.ReadFile(target) // #nosec G304 -- resolved target of a selected Go source symlink.
	if err != nil {
		return fmt.Errorf("read Go source symlink target %q: %w", alias, err)
	}
	return hashStructuralInput(hash, alias, content, inputs)
}

func fingerprintInputSelected(path string, scope diagnosticsScope, embedPatterns map[string][]embedPattern) bool {
	if filepath.Ext(path) == ".go" || isManifestInput(path) || matchesEmbedInput(path, embedPatterns) || compilerInputFilename(filepath.Base(path)) {
		return true
	}
	_, knownInput := scope.manifests[path]
	return knownInput
}

func hashOverlayInputs(hash interface{ Write([]byte) (int, error) }, moduleRoot string, overlays map[string][]byte, seen map[string]struct{}, write func(string), inputs fingerprintInputs) error {
	paths := make([]string, 0, len(overlays))
	for path := range overlays {
		paths = append(paths, path)
	}
	slices.Sort(paths)
	for _, path := range paths {
		content := overlays[path]
		if _, exists := seen[path]; exists || !pathutil.PathWithin(moduleRoot, path) || content == nil {
			continue
		}
		relative, err := filepath.Rel(moduleRoot, path)
		if err != nil {
			return fmt.Errorf("resolve buffered Go module input %q: %w", path, err)
		}
		write(filepath.ToSlash(relative))
		if err := hashStructuralInput(hash, path, content, inputs); err != nil {
			return err
		}
	}
	return nil
}

func hashManifestInputs(manifests map[string]manifestState, write func(string)) error {
	paths := make([]string, 0, len(manifests))
	for path := range manifests {
		paths = append(paths, path)
	}
	slices.Sort(paths)
	for _, path := range paths {
		state, err := readInputState(path)
		if err != nil {
			return err
		}
		write(path)
		write(fmt.Sprintf("%t:%x", state.exists, state.digest))
	}
	return nil
}

type embedPattern struct {
	value         string
	includeHidden bool
}

func scopeEmbedPatterns(ctx context.Context, files []string, inputs fingerprintInputs) (map[string][]embedPattern, error) {
	defer telemetry.StartPhase(ctx, "go.scope.embed_patterns")()
	patterns := make(map[string][]embedPattern)
	for _, file := range files {
		content, err := sourcebuf.ReadFile(ctx, file)
		if err != nil {
			return nil, fmt.Errorf("read Go embed directives from %q: %w", file, err)
		}
		input := structuralGoInput(file, content, inputs)
		if input.valid && len(input.patterns) > 0 {
			patterns[filepath.Dir(file)] = append(patterns[filepath.Dir(file)], input.patterns...)
		}
	}
	return patterns, nil
}

func matchesEmbedInput(file string, patterns map[string][]embedPattern) bool {
	for directory, candidates := range patterns {
		relative, err := filepath.Rel(directory, file)
		if err != nil || !filepath.IsLocal(relative) {
			continue
		}
		parts := strings.Split(filepath.ToSlash(relative), "/")
		for _, candidate := range candidates {
			if !candidate.includeHidden && hasHiddenEmbedPath(parts) {
				continue
			}
			for end := 1; end <= len(parts); end++ {
				matched, matchErr := path.Match(candidate.value, strings.Join(parts[:end], "/"))
				if matchErr == nil && matched {
					return true
				}
			}
		}
	}
	return false
}

func hasHiddenEmbedPath(parts []string) bool {
	for _, part := range parts {
		if strings.HasPrefix(part, ".") || strings.HasPrefix(part, "_") {
			return true
		}
	}
	return false
}

func isManifestInput(file string) bool {
	switch filepath.Base(file) {
	case "go.mod", "go.sum", "go.work", "go.work.sum":
		return true
	default:
		return false
	}
}

func hashStructuralInput(hash interface{ Write([]byte) (int, error) }, path string, content []byte, inputs fingerprintInputs) error {
	_, _ = hash.Write([]byte(filepath.Ext(path) + "\x00"))
	_, _ = inputs.boundary.Write([]byte(filepath.Ext(path) + "\x00"))
	if filepath.Ext(path) != ".go" {
		_, _ = hash.Write(content)
		_, _ = inputs.boundary.Write(content)
		return nil
	}
	input := structuralGoInput(path, content, inputs)
	if input.valid {
		_, _ = hash.Write(input.structure)
		_, _ = inputs.boundary.Write([]byte("valid\x00"))
		_, _ = inputs.boundary.Write(input.boundary)
	} else {
		_, _ = hash.Write(content)
		_, _ = inputs.boundary.Write([]byte("invalid\x00"))
		_, _ = inputs.boundary.Write(content)
	}
	return nil
}

func parseStructuralGoFile(path string, content []byte) (*ast.File, bool) {
	file, err := parser.ParseFile(token.NewFileSet(), path, content, parser.ParseComments|parser.AllErrors)
	return file, err == nil
}

func addOverlaySources(ctx context.Context, moduleRoot string, files map[string]struct{}) error {
	canonicalRoot, err := sourcebuf.CanonicalPath(ctx, moduleRoot)
	if err != nil {
		return fmt.Errorf("canonicalize Go diagnostics module root: %w", err)
	}
	for path := range sourcebuf.Overlay(ctx, ".go") {
		canonical, err := canonicalGoSourcePath(ctx, path)
		if err != nil {
			return err
		}
		if _, statErr := os.Lstat(canonical); statErr == nil {
			continue
		} else if !errors.Is(statErr, os.ErrNotExist) {
			return fmt.Errorf("inspect buffered Go source %q: %w", path, statErr)
		}
		managed, err := managedOverlaySource(canonicalRoot, canonical)
		if err != nil {
			return err
		}
		if managed {
			files[canonical] = struct{}{}
		}
	}
	return nil
}

func managedOverlaySource(moduleRoot, path string) (bool, error) {
	if filepath.Ext(path) != ".go" || strings.HasSuffix(path, "_test.go") || !pathutil.PathWithin(moduleRoot, path) {
		return false, nil
	}
	relative, err := filepath.Rel(moduleRoot, path)
	if err != nil {
		return false, fmt.Errorf("resolve buffered Go source path %q: %w", path, err)
	}
	if !filepath.IsLocal(relative) {
		return false, nil
	}
	for directory := filepath.Dir(relative); directory != "."; directory = filepath.Dir(directory) {
		base := filepath.Base(directory)
		if base == "vendor" || base == "testdata" || strings.HasPrefix(base, ".") {
			return false, nil
		}
	}
	for directory := filepath.Dir(path); directory != moduleRoot; directory = filepath.Dir(directory) {
		manifest := filepath.Join(directory, "go.mod")
		_, statErr := os.Stat(manifest)
		switch {
		case statErr == nil:
			return false, nil
		case errors.Is(statErr, os.ErrNotExist):
			continue
		default:
			return false, fmt.Errorf("inspect buffered Go module boundary %q: %w", directory, statErr)
		}
	}
	return true, nil
}

func canonicalGoSourcePath(ctx context.Context, path string) (string, error) {
	canonical, err := sourcebuf.CanonicalPath(ctx, path)
	if err != nil {
		return "", fmt.Errorf("canonicalize Go source path %q: %w", path, err)
	}
	return canonical, nil
}

func compilerInputFilename(name string) bool {
	// Classify compiler extensions independently of the host build tags and file contents.
	classification := build.Default
	classification.UseAllFiles = true
	classification.OpenFile = func(string) (io.ReadCloser, error) { return io.NopCloser(strings.NewReader("")), nil }
	matched, err := classification.MatchFile("", name)
	return matched || err != nil
}

func encodeDiagnosticBoundaryGoInput(file *ast.File) []byte {
	boundary := *file
	boundary.Imports = nil
	for _, spec := range file.Imports {
		importPath, err := strconv.Unquote(spec.Path.Value)
		if err == nil && importPath == "C" {
			boundary.Imports = append(boundary.Imports, spec)
		}
	}
	return encodeStructuralGoInput(&boundary)
}

func structuralGoInput(path string, content []byte, inputs fingerprintInputs) fingerprintInput {
	digest := sha256.Sum256(content)
	if input, ok := inputs.sources[path]; ok && input.digest == digest {
		return input
	}
	if input, ok := inputs.retained[path]; ok && input.digest == digest {
		inputs.sources[path] = input
		return input
	}
	defer telemetry.StartPhase(inputs.ctx, "go.scope.parse")()
	file, valid := parseStructuralGoFile(path, content)
	input := fingerprintInput{digest: digest, valid: valid}
	if valid {
		input.structure = encodeStructuralGoInput(file)
		input.boundary = encodeDiagnosticBoundaryGoInput(file)
		input.patterns = structuralEmbedPatterns(file)
	}
	inputs.sources[path] = input
	return input
}

func structuralEmbedPatterns(file *ast.File) []embedPattern {
	var patterns []embedPattern
	for _, group := range file.Comments {
		for _, comment := range group.List {
			text := strings.TrimSpace(strings.TrimPrefix(comment.Text, "//go:embed"))
			if text == comment.Text {
				continue
			}
			for field := range strings.FieldsSeq(text) {
				includeHidden := strings.HasPrefix(field, "all:")
				field = strings.TrimPrefix(field, "all:")
				if unquoted, err := strconv.Unquote(field); err == nil {
					field = unquoted
				}
				patterns = append(patterns, embedPattern{value: field, includeHidden: includeHidden})
			}
		}
	}
	return patterns
}

const diagnosticWorkerLimit = 1

func canSeedFullDiagnostics(ctx context.Context, root string, environment []string, files []string, findings []Diagnostic, session *Session) bool {
	if sourcebuf.FromContext(ctx) != nil {
		telemetry.StartPhase(ctx, "go.diagnostics.seed_skipped.buffered")()
		return false
	}
	scopeFiles, scopeFindings, err := selectPackageFilesWithPolicy(root, session.session.scope.packages, buildCacheFromEnvironment(environment), false)
	if err != nil {
		telemetry.StartPhase(ctx, "go.diagnostics.seed_skipped.scope_selection_error")()
		return false
	}
	if !slices.Equal(files, scopeFiles) || !slices.Equal(findings, scopeFindings) {
		telemetry.StartPhase(ctx, "go.diagnostics.seed_skipped.coverage_mismatch")()
		return false
	}
	return true
}

func diagnosticGraphHasErrors(pkgs []*packages.Package) bool {
	found := false
	packages.Visit(pkgs, func(pkg *packages.Package) bool {
		if len(pkg.Errors) != 0 {
			found = true
		}
		return true
	}, nil)
	return found
}

func diagnosticScopeRequiresFullRefresh(previous, current diagnosticsScope) bool {
	if previous.fingerprint.boundary != current.fingerprint.boundary || !slices.Equal(previous.files, current.files) || !slices.Equal(previous.modules, current.modules) {
		return true
	}
	return diagnosticGraphErrorTransition(previous, current)
}

func diagnosticGraphErrorTransition(previous, current diagnosticsScope) bool {
	return previous.fingerprint.discovery != current.fingerprint.discovery && (diagnosticGraphHasErrors(previous.packages) || diagnosticGraphHasErrors(current.packages))
}
