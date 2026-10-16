// Package godiagnostics owns shared gopls analysis sessions so diagnostics, rename, and AST refactoring synchronization observe the same revision.
package godiagnostics

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"github.com/spockz/semantic-editor/internal/backend/pathutil"
	"github.com/spockz/semantic-editor/internal/lsp"
	"maps"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"encoding/json"

	"github.com/spockz/semantic-editor/internal/sourcebuf"

	"golang.org/x/tools/go/packages"
)

// Session exposes synchronized Go analysis during one serialized operation.
type Session struct {
	ctx     context.Context
	Root    string
	session *managedSession
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
		return Document{}, fmt.Errorf("read synchronized Go source: %w", err)
	}
	if sha256.Sum256(content) != snapshot.digest {
		return Document{}, fmt.Errorf("go source %q changed since synchronization", path)
	}
	info, err := os.Stat(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return Document{}, fmt.Errorf("stat source preimage: %w", err)
	}
	if err == nil && !info.Mode().IsRegular() {
		return Document{}, fmt.Errorf("source preimage %q is not regular", path)
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

// AssertCurrent rejects disk changes anywhere in the analysis scope before publication.
func (s *Session) AssertCurrent() error {
	return s.AssertRevision(nil)
}

// AssertRevision rechecks all analysis inputs while accounting for files published by the current rename.
func (s *Session) AssertRevision(published map[string][]byte) error {
	for _, path := range s.session.files {
		if expected, ok := published[path]; ok {
			// #nosec G304 -- these are retained Go analysis input paths.
			current, err := sourcebuf.ReadFile(s.ctx, path)
			if err != nil {
				return fmt.Errorf("read published Go source: %w", err)
			}
			if !bytes.Equal(current, expected) {
				return fmt.Errorf("published Go source %q changed during rename", path)
			}
		} else if _, err := s.Snapshot(path); err != nil {
			return err
		}
	}
	for path, expected := range s.session.manifests {
		current, err := readInputState(path)
		if err != nil {
			return err
		}
		if current != expected {
			return fmt.Errorf("gopls input %q changed since synchronization", path)
		}
	}
	if buffer := sourcebuf.FromContext(s.ctx); buffer != nil {
		return buffer.AssertDisk()
	}
	return nil
}

// SessionManager owns one reusable gopls analysis session per Go module.
type SessionManager struct {
	mu       sync.Mutex
	sessions map[string]*managedSession
	closed   bool
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
		files, findings, err := loadGoFiles(ctx, root, environment, patterns...)
		if err != nil {
			return err
		}
		diagnostics = append(diagnostics, findings...)
		for _, path := range files {
			found, err := requestDiagnostics(ctx, session.session.client, root, path)
			if err != nil {
				return err
			}
			diagnostics = append(diagnostics, found...)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	slices.SortFunc(diagnostics, func(a, b Diagnostic) int { return strings.Compare(a.String(), b.String()) })
	return diagnostics, nil
}

// NewSessionManager creates an MCP-scoped Go analysis manager.
func NewSessionManager() *SessionManager {
	return &SessionManager{sessions: make(map[string]*managedSession)}
}

// WithSessionManager attaches MCP-owned Go analysis state to a request.
func WithSessionManager(ctx context.Context, manager *SessionManager) context.Context {
	return context.WithValue(ctx, managerContextKey{}, manager)
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

// SynchronizeWithContext refreshes retained Go sessions after every mutation, including deferred batch edits.
func SynchronizeWithContext(ctx context.Context) error {
	if ctx == nil {
		return nil
	}
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
	for root, session := range manager.sessions {
		sessions = append(sessions, retained{root: root, executable: session.executable, environment: slices.Clone(session.environment)})
	}
	manager.mu.Unlock()
	for _, session := range sessions {
		if err := manager.use(ctx, session.root, session.executable, session.environment, func(context.Context, *Session) error { return nil }); err != nil {
			return fmt.Errorf("synchronize Go refactoring state: %w", err)
		}
	}
	return nil
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
}

type diagnosticsScope struct {
	root      string
	files     []string
	manifests map[string]manifestState
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
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return errors.New("gopls session manager is closed")
	}
	scope, err := loadManagedScope(sessionCtx, root, environment)
	if err != nil {
		return m.discardOverlappingScopes(root, err)
	}
	if err := validateBufferedScope(sessionCtx, scope); err != nil {
		return err
	}
	session := m.sessions[scope.root]
	if session != nil && !managedSessionMatches(session, executable, executableInfo, environment, scope) {
		delete(m.sessions, scope.root)
		closeCtx, closeCancel := context.WithTimeout(sessionCtx, 5*time.Second)
		closeErr := closeManagedSession(closeCtx, session)
		closeCancel()
		if closeErr != nil {
			return fmt.Errorf("replace stale Go LSP session for %q: %w", root, closeErr)
		}
		session = nil
	}
	if session == nil {
		client, err := startDiagnosticsClient(sessionCtx, scope.root, executable, environment)
		if err != nil {
			return err
		}
		session = &managedSession{
			client: client, executable: executable, executableInfo: executableInfo, environment: slices.Clone(environment),
			files: slices.Clone(scope.files), documents: make(map[string]documentSnapshot, len(scope.files)),
			manifests: scope.manifests,
		}
		if m.sessions == nil {
			m.sessions = make(map[string]*managedSession)
		}
		m.sessions[scope.root] = session
	}
	if err := session.syncDocuments(sessionCtx, scope.files); err != nil {
		return m.discard(scope.root, err)
	}

	if err := action(sessionCtx, &Session{ctx: sessionCtx, Root: scope.root, session: session}); err != nil {
		return m.discard(scope.root, err)
	}
	if err := session.syncDocuments(sessionCtx, scope.files); err != nil {
		return m.discard(scope.root, err)
	}
	if err := sessionCtx.Err(); err != nil {
		return m.discard(scope.root, fmt.Errorf("gopls session: %w", err))
	}
	return nil
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
			if err := sourcebuf.ObserveTree(ctx, filepath.Dir(path)); err != nil {
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

type selectedDocument struct {
	content []byte
	digest  [32]byte
}

type watchedFileChange struct {
	uri  string
	kind int
}

func (s *managedSession) syncDocuments(ctx context.Context, files []string) error {
	selected, err := readSelectedDocuments(ctx, files)
	if err != nil {
		return err
	}
	watchChanges, err := s.closeRemovedDocuments(ctx, selected)
	if err != nil {
		return err
	}
	for _, path := range files {
		change, err := s.syncDocument(ctx, path, selected[path])
		if err != nil {
			return err
		}
		if change.kind != 0 {
			watchChanges = append(watchChanges, change)
		}
	}
	if err := s.notifyMembershipChanges(ctx, watchChanges); err != nil {
		return err
	}
	s.documentsInitialized = true
	s.files = slices.Clone(files)
	return nil
}

func readSelectedDocuments(ctx context.Context, files []string) (map[string]selectedDocument, error) {
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

func (s *managedSession) closeRemovedDocuments(ctx context.Context, selected map[string]selectedDocument) ([]watchedFileChange, error) {
	watchChanges := make([]watchedFileChange, 0)
	removed := make([]string, 0)
	for path := range s.documents {
		if _, ok := selected[path]; ok {
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
			watchChanges = append(watchChanges, watchedFileChange{uri: uri, kind: 3})
		}
		delete(s.documents, path)
	}
	return watchChanges, nil
}

func (s *managedSession) syncDocument(ctx context.Context, path string, document selectedDocument) (watchedFileChange, error) {
	uri := (&url.URL{Scheme: "file", Path: filepath.ToSlash(path)}).String()
	previous, opened := s.documents[path]
	if !opened {
		if err := s.client.Notify(ctx, "textDocument/didOpen", map[string]any{
			textDocumentKey: map[string]any{"uri": uri, "languageId": "go", "version": 1, "text": string(document.content)},
		}); err != nil {
			return watchedFileChange{}, fmt.Errorf("open Go source %q in gopls: %w", path, err)
		}
		saved := !sourcebuf.Dirty(ctx, path)
		s.documents[path] = documentSnapshot{version: 1, digest: document.digest, saved: saved, published: saved}
		if s.documentsInitialized && saved {
			return watchedFileChange{uri: uri, kind: 1}, nil
		}
		return watchedFileChange{}, nil
	}
	if previous.digest == document.digest && previous.saved {
		return watchedFileChange{}, nil
	}
	version := previous.version
	if previous.digest != document.digest {
		version++
		if err := s.client.Notify(ctx, "textDocument/didChange", map[string]any{textDocumentKey: map[string]any{"uri": uri, "version": version}, "contentChanges": []map[string]any{{"text": string(document.content)}}}); err != nil {
			return watchedFileChange{}, fmt.Errorf("synchronize Go source %q in gopls: %w", path, err)
		}
	}
	if sourcebuf.Dirty(ctx, path) {
		s.documents[path] = documentSnapshot{version: version, digest: document.digest, saved: false, published: previous.published}
		return watchedFileChange{}, nil
	}
	if err := s.client.Notify(ctx, "textDocument/didSave", map[string]any{textDocumentKey: map[string]any{"uri": uri}}); err != nil {
		return watchedFileChange{}, fmt.Errorf("save Go source %q in gopls: %w", path, err)
	}
	changeType := 2
	if !previous.published {
		changeType = 1
	}
	if err := s.client.Notify(ctx, "workspace/didChangeWatchedFiles", map[string]any{"changes": []map[string]any{{"uri": uri, "type": changeType}}}); err != nil {
		return watchedFileChange{}, fmt.Errorf("notify changed Go source %q: %w", path, err)
	}
	s.documents[path] = documentSnapshot{version: version, digest: document.digest, saved: true, published: true}
	return watchedFileChange{}, nil
}

func (s *managedSession) notifyMembershipChanges(ctx context.Context, watchChanges []watchedFileChange) error {
	if len(watchChanges) > 0 {
		changes := make([]map[string]any, 0, len(watchChanges))
		for _, change := range watchChanges {
			changes = append(changes, map[string]any{"uri": change.uri, "type": change.kind})
		}
		if err := s.client.Notify(ctx, "workspace/didChangeWatchedFiles", map[string]any{"changes": changes}); err != nil {
			return fmt.Errorf("notify changed Go source membership: %w", err)
		}
	}
	return nil
}

type managerContextKey struct{}

type manifestState struct {
	exists bool
	digest [32]byte
}

type managedSession struct {
	client               *lsp.Client
	executableInfo       os.FileInfo
	executable           string
	environment          []string
	files                []string
	documents            map[string]documentSnapshot
	documentsInitialized bool
	manifests            map[string]manifestState
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
	buffer := sourcebuf.FromContext(ctx)
	bound := make(map[string]bool)
	for {
		scope, err := discoverManagedScope(ctx, root, environment)
		if err != nil {
			return diagnosticsScope{}, err
		}
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
			if err := sourcebuf.ObserveTree(ctx, filepath.Dir(path)); err != nil {
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
	loaded, err := packages.Load(&packages.Config{
		Context: ctx, Dir: moduleRoot, Env: environment, Overlay: sourcebuf.Overlay(ctx),
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
	scope := diagnosticsScope{root: moduleRoot, manifests: auxiliary}
	for path := range fileSet {
		scope.files = append(scope.files, path)
	}
	slices.Sort(scope.files)
	for directory := range manifestRoots {
		state, err := readManifestState(directory)
		if err != nil {
			return diagnosticsScope{}, err
		}
		maps.Copy(scope.manifests, state)
	}
	return scope, nil
}

func addOverlaySources(ctx context.Context, moduleRoot string, files map[string]struct{}) error {
	canonicalRoot, err := sourcebuf.CanonicalPath(ctx, moduleRoot)
	if err != nil {
		return fmt.Errorf("canonicalize Go diagnostics module root: %w", err)
	}
	for path := range sourcebuf.Overlay(ctx) {
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
