// Package textlsp adapts bounded read-only LSP document symbols for selected text files.
package textlsp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"semedit/internal/backend"
	"semedit/internal/backend/pathutil"
	"semedit/internal/backend/readlsp"
	"semedit/internal/lsp"
	"semedit/internal/pipeline"
)

var (
	// ErrFileRequired reports lookup or verification requests without a selected file.
	ErrFileRequired = errors.New("selected source file is required")
	// ErrFileOutsideRoot reports selected paths outside the trusted workspace.
	ErrFileOutsideRoot = errors.New("selected source file is outside workspace root")
	// ErrSymbolNotFound reports symbols absent from the selected source file.
	ErrSymbolNotFound = errors.New("symbol not found in selected file")
	// ErrMalformedSymbols reports LSP symbol records that cannot be mapped safely.
	ErrMalformedSymbols = errors.New("malformed document symbols response")
	// ErrDiagnosticsTimeout reports that the matching diagnostics receipt was absent.
	ErrDiagnosticsTimeout = errors.New("matching diagnostics report was not received before timeout")
	// ErrOperationTimeout reports that an LSP operation exceeded its deadline.
	ErrOperationTimeout = errors.New("language-server operation exceeded its bounded deadline")
	// ErrScratchOutsideRoot reports scratch paths that escape the trusted workspace.
	ErrScratchOutsideRoot = errors.New("isolated LSP scratch path escapes workspace root")
)

const textDocumentKey = "textDocument"

const readOnlyPreview = "Read-only preview"

// Config describes a selected-file LSP backend without language-specific mutation support.
type Config struct {
	Language     backend.LanguageID
	Binary       string
	Args         []string
	Extensions   []string
	Basenames    []string
	LanguageID   string
	Diagnostics  bool
	Timeout      time.Duration
	SymbolSyntax string
}

// Session is the bounded request and notification surface used by an adapter.
type Session interface {
	Request(context.Context, string, any) (json.RawMessage, error)
	Notify(context.Context, string, any) error
	WaitDiagnostics(context.Context, string, int) ([]backend.Diagnostic, error)
	Close() error
}

// Factory starts a language server with a private working directory and environment.
type Factory func(context.Context, string, string, []string) (Session, error)

// Adapter provides trusted selected-file LSP lookup and optional explicit diagnostics.
type Adapter struct {
	Config  Config
	Factory Factory
}

// Language returns the configured language identifier.
func (a *Adapter) Language() backend.LanguageID { return a.Config.Language }

// CapabilityMatrix reports the operations proven for this adapter.
func (a *Adapter) CapabilityMatrix() backend.LanguageMatrix {
	name, file, server := "Bash", ".sh or .bash", "bash-language-server"
	if a.Language() == backend.LanguageMake {
		name, file, server = "Makefile", "Makefile, makefile, GNUmakefile, or .mk", "make-ls"
	}
	lookupDescription := "Trusted selected-file symbol lookup from the preinstalled " + server + " using UTF-16 positions."
	if a.Language() == backend.LanguageMake {
		lookupDescription = "Trusted selected-file Make target and assignment-variable lookup from source-mapped make-ls symbols."
	}
	ops := map[string]backend.OpCapability{"lookup": {Supported: true, Description: lookupDescription, CLICommand: "semedit lookup --language " + string(a.Language()) + " --file <path> --symbol <sym>", MCPTool: "semantic_lookup"}}
	ops[string(backend.OperationOutline)] = backend.OpCapability{
		Supported: true, Description: "Trusted selected-file outline from validated " + server + " document symbols; directory scopes and visibility filtering are Go-only.",
		CLICommand: "semedit outline --language " + string(a.Language()) + " --path <file>", MCPTool: "semantic_outline", Level: "symbol", ReadOnly: true,
	}
	if a.Language() == backend.LanguageMake {
		ops[string(backend.OperationInspect)] = backend.OpCapability{
			Supported: true, Description: "Inspect matching Make targets and assignment variables from exact selected-file make-ls source ranges.",
			CLICommand: "semedit inspect-symbol --language make --file <path> --symbol <name>", MCPTool: "semantic_inspect_symbol", Level: "symbol", ReadOnly: true,
		}
	}
	if a.Config.Diagnostics {
		ops["verify"] = backend.OpCapability{Supported: true, Description: "Trusted selected-file diagnostics from a matching publishDiagnostics report; formatting and imports are unsupported.", CLICommand: "semedit verify --language bash --file <path>", MCPTool: "semantic_verify"}
	}
	limitations := []backend.Constraint{
		{Title: "Read Only", Description: name + " rename, formatting, imports, and structural edits are unavailable.", Severity: "error"},
		{Title: "Trusted Explicit Tools", Description: "Lookup requires a selected " + file + " file inside the canonical workspace, request-scoped trust, and a preinstalled " + server + " executable.", Severity: "error"},
		{Title: "Selected-File Reads", Description: "External outlines and inspection are limited to one selected file; visibility filtering and directory scopes are Go-only.", Severity: "info"},
	}
	if a.Language() == backend.LanguageMake {
		limitations = append(limitations, backend.Constraint{Title: "Incomplete Make Projection", Description: "Make outlines and inspections are incomplete: continued declarations fail explicitly, while conditional blocks, include-remapped symbols, define blocks, and interrupted recipe ranges remain bounded server projections.", Severity: "info"})
		limitations = append(limitations, backend.Constraint{Title: "Make Includes", Description: "The isolated make-ls workspace does not constrain include paths outside its copied source; absolute or traversing relative includes may read files after workspace trust is granted.", Severity: "info"})
		limitations = append(limitations, backend.Constraint{Title: "Make Conditionals", Description: "Conditional DocumentSymbol records span whole blocks and cannot be mapped to a selected-source declaration, so conditional lookup is omitted.", Severity: "info"})
	}
	if a.Config.Diagnostics {
		limitations = append(limitations, backend.Constraint{Title: "Matching Diagnostics", Description: "Only an explicit matching publishDiagnostics report is accepted; missing reports time out as errors.", Severity: "info"})
	}
	if a.Language() == backend.LanguageBash {
		limitations = append(limitations, backend.Constraint{Title: "Flat Bash Symbols", Description: "bash-language-server provides flat function and variable locations; declaration extents and nested symbol hierarchy are not inferred.", Severity: "info"})
		limitations = append(limitations, backend.Constraint{Title: "Optional Shell Tools", Description: "bash-language-server may invoke an installed ShellCheck against the copied selected source; the adapter does not execute the shell script.", Severity: "info"})
	}
	return backend.LanguageMatrix{Language: string(a.Language()), DisplayName: name, Maturity: readOnlyPreview, Operations: ops, Limitations: limitations}
}

// Capabilities returns the operation registry for this adapter.
func (a *Adapter) Capabilities() backend.Capabilities {
	ops := []backend.Operation{backend.OperationLookup, backend.OperationOutline}
	if a.Language() == backend.LanguageMake {
		ops = append(ops, backend.OperationInspect)
	}
	if a.Config.Diagnostics {
		ops = append(ops, backend.OperationVerify)
	}
	return backend.NewCapabilitiesRequiringWorkspaceTrust(ops...)
}

// Rename reports that textual LSP ranges are not a supported mutation primitive.
func (a *Adapter) Rename(context.Context, backend.RenameRequest) (*backend.RenameResult, error) {
	return nil, &backend.Error{Operation: backend.OperationRename, Language: a.Language(), Err: backend.ErrUnsupportedOperation}
}

// Verify returns diagnostics only after receiving a matching explicit report.
func (a *Adapter) Verify(ctx context.Context, request backend.VerifyRequest) (result []backend.Diagnostic, retErr error) {
	if !a.Config.Diagnostics || request.FormatSelectedFile || request.OrganizeImports {
		return nil, &backend.Error{Operation: backend.OperationVerify, Language: a.Language(), Err: backend.ErrUnsupportedOperation}
	}
	root, file, source, err := a.resolve(request.Project)
	if err != nil {
		return nil, err
	}
	if !request.Project.WorkspaceTrust.Allows(root) {
		return nil, &backend.WorkspaceTrustError{Operation: backend.OperationVerify, Language: a.Language(), Workspace: root}
	}
	callerCtx := nonNilContext(ctx)
	ctx, cancel := context.WithTimeout(callerCtx, a.timeout())
	defer func() {
		if callerCtx.Err() == nil && errors.Is(ctx.Err(), context.DeadlineExceeded) && retErr != nil {
			retErr = errors.Join(retErr, ErrOperationTimeout)
		}
		cancel()
	}()
	session, scratch, uri, err := a.open(ctx, root, file, source, request.Project)
	if err != nil {
		return nil, err
	}
	defer func() {
		retErr = errors.Join(retErr, session.Close())
		if cleanupErr := os.RemoveAll(scratch); cleanupErr != nil {
			retErr = errors.Join(retErr, fmt.Errorf("remove isolated LSP workspace: %w", cleanupErr))
		}
	}()
	if err := session.Notify(ctx, "textDocument/didOpen", map[string]any{textDocumentKey: map[string]any{"uri": uri, "languageId": a.Config.LanguageID, "version": 1, "text": string(source)}}); err != nil {
		return nil, fmt.Errorf("open selected file for diagnostics: %w", err)
	}
	waitCtx, waitCancel := context.WithTimeout(ctx, 5*time.Second)
	defer waitCancel()
	diagnostics, err := session.WaitDiagnostics(waitCtx, uri, 1)
	if err != nil {
		return nil, fmt.Errorf("%s verify %s: %w", a.Language(), file, errors.Join(ErrDiagnosticsTimeout, err))
	}
	for i := range diagnostics {
		if diagnostics[i].Location != nil {
			diagnostics[i].Location.URI = pathutil.FileURI(file)
		}
	}
	return diagnostics, nil
}

// Lookup resolves a symbol from one trusted selected source file.
func (a *Adapter) Lookup(ctx context.Context, project backend.ProjectContext, query string) (result *backend.LookupResult, retErr error) {
	root, file, source, err := a.resolve(project)
	if err != nil {
		return nil, err
	}
	if !project.WorkspaceTrust.Allows(root) {
		return nil, &backend.WorkspaceTrustError{Operation: backend.OperationLookup, Language: a.Language(), Workspace: root}
	}
	if strings.TrimSpace(query) == "" {
		return nil, ErrMalformedSymbols
	}
	callerCtx := nonNilContext(ctx)
	ctx, cancel := context.WithTimeout(callerCtx, a.timeout())
	defer func() {
		if callerCtx.Err() == nil && errors.Is(ctx.Err(), context.DeadlineExceeded) && retErr != nil {
			retErr = errors.Join(retErr, ErrOperationTimeout)
		}
		cancel()
	}()
	session, scratch, uri, err := a.open(ctx, root, file, source, project)
	if err != nil {
		return nil, err
	}
	defer func() {
		retErr = errors.Join(retErr, session.Close())
		if cleanupErr := os.RemoveAll(scratch); cleanupErr != nil {
			retErr = errors.Join(retErr, fmt.Errorf("remove isolated LSP workspace: %w", cleanupErr))
		}
	}()
	if err := session.Notify(ctx, "textDocument/didOpen", map[string]any{textDocumentKey: map[string]any{"uri": uri, "languageId": a.Config.LanguageID, "version": 1, "text": string(source)}}); err != nil {
		return nil, fmt.Errorf("open selected file: %w", err)
	}
	raw, err := session.Request(ctx, "textDocument/documentSymbol", map[string]any{textDocumentKey: map[string]string{"uri": uri}})
	if err != nil {
		return nil, fmt.Errorf("request document symbols: %w", err)
	}
	symbols, err := decodeSymbols(raw, source, a.Config.SymbolSyntax)
	if err != nil {
		return nil, err
	}
	return selectSymbol(query, root, file, source, symbols)
}

// Outline returns a validated selected-file projection from the configured language server.
func (a *Adapter) Outline(ctx context.Context, request backend.OutlineRequest) (*backend.OutlineResult, error) {
	if a.Language() != backend.LanguageBash && a.Language() != backend.LanguageMake {
		return nil, &backend.Error{Operation: backend.OperationOutline, Language: a.Language(), Err: backend.ErrUnsupportedOperation}
	}
	if err := readlsp.ValidateOutlineRequest(request, false); err != nil {
		return nil, &backend.Error{Operation: backend.OperationOutline, Language: a.Language(), Err: err}
	}
	root := request.Project.RootDir
	if root == "" {
		root = "."
	}
	root, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve outline workspace: %w", err)
	}
	selected := request.Path
	if !filepath.IsAbs(selected) {
		selected = filepath.Join(root, selected)
	}
	selected, err = filepath.Abs(selected)
	if err != nil {
		return nil, fmt.Errorf("resolve outline path: %w", err)
	}
	info, err := os.Stat(selected)
	if err != nil {
		return nil, fmt.Errorf("stat outline path %q: %w", selected, err)
	}
	if info.IsDir() {
		err := readlsp.ValidateOutlineRequest(request, true)
		return nil, &backend.Error{Operation: backend.OperationOutline, Language: a.Language(), Err: err}
	}
	if !info.Mode().IsRegular() {
		return nil, &backend.Error{Operation: backend.OperationOutline, Language: a.Language(), Err: fmt.Errorf("outline path %q is not a regular source file", request.Path)}
	}
	project := request.Project
	project.File = request.Path
	root, file, source, symbols, limitations, err := a.readSymbols(ctx, project, backend.OperationOutline)
	if err != nil {
		return nil, err
	}
	complete := a.Language() == backend.LanguageBash
	if a.Language() == backend.LanguageMake {
		limitations = append(makeOutlineLimitations(), limitations...)
	}
	document := readlsp.Document{Root: root, File: file, Language: a.Language(), Source: source, Symbols: symbols, Complete: complete, Limitations: limitations}
	result, err := readlsp.Outline(document, request)
	if err != nil {
		return nil, &backend.Error{Operation: backend.OperationOutline, Language: a.Language(), Err: fmt.Errorf("outline %s file %q: %w", a.Language(), request.Path, err)}
	}
	return result, nil
}

// Inspect returns exact selected-file source ranges from the configured language server.
func (a *Adapter) Inspect(ctx context.Context, request backend.InspectRequest) (*backend.InspectResult, error) {
	if a.Language() != backend.LanguageMake {
		return nil, &backend.Error{Operation: backend.OperationInspect, Language: a.Language(), Err: backend.ErrUnsupportedOperation}
	}
	root, file, source, symbols, limitations, err := a.readSymbols(ctx, request.Project, backend.OperationInspect)
	if err != nil {
		return nil, err
	}
	limitations = append(makeOutlineLimitations(), limitations...)
	matches := matchReadSymbols(request.Symbol, symbols)
	if len(matches) == 0 {
		return nil, &backend.Error{Operation: backend.OperationInspect, Language: a.Language(), Err: fmt.Errorf("%s: %w", request.Symbol, ErrSymbolNotFound)}
	}
	selected := make([]readlsp.InspectMatch, 0, len(matches))
	for _, symbol := range matches {
		selected = append(selected, readlsp.InspectMatch{Symbol: symbol, SourceExtent: "server_range"})
	}
	document := readlsp.Document{Root: root, File: file, Language: a.Language(), Source: source, Symbols: symbols, Complete: false, Limitations: limitations}
	result, err := readlsp.Inspect(document, selected)
	if err != nil {
		return nil, &backend.Error{Operation: backend.OperationInspect, Language: a.Language(), Err: fmt.Errorf("inspect %s file %q symbol %q: %w", a.Language(), file, request.Symbol, err)}
	}
	return result, nil
}

func (a *Adapter) timeout() time.Duration {
	if a.Config.Timeout > 0 {
		return a.Config.Timeout
	}
	return 30 * time.Second
}
func nonNilContext(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return ctx
}
func (a *Adapter) resolve(project backend.ProjectContext) (string, string, []byte, error) {
	root := project.RootDir
	if root == "" {
		root = "."
	}
	root, err := filepath.Abs(root)
	if err != nil {
		return "", "", nil, err
	}
	root = backend.CanonicalWorkspaceRoot(root)
	selected := project.File
	if selected == "" {
		selected = project.RootDir
	}
	if selected == "" || selected == "." {
		return "", "", nil, ErrFileRequired
	}
	if !filepath.IsAbs(selected) {
		selected = filepath.Join(root, selected)
	}
	file, err := filepath.Abs(selected)
	if err != nil {
		return "", "", nil, err
	}
	file, err = filepath.EvalSymlinks(file)
	if err != nil {
		return "", "", nil, fmt.Errorf("resolve selected file: %w", err)
	}
	if !pathutil.PathWithin(root, file) {
		return "", "", nil, ErrFileOutsideRoot
	}
	base := strings.ToLower(filepath.Base(file))
	ext := strings.ToLower(filepath.Ext(file))
	allowed := false
	for _, v := range a.Config.Extensions {
		if ext == v {
			allowed = true
		}
	}
	for _, v := range a.Config.Basenames {
		if base == strings.ToLower(v) {
			allowed = true
		}
	}
	if !allowed {
		return "", "", nil, ErrFileRequired
	}
	source, err := os.ReadFile(file)
	if err != nil {
		return "", "", nil, fmt.Errorf("read selected file: %w", err)
	}
	return root, file, source, nil
}
func (a *Adapter) open(ctx context.Context, root, file string, source []byte, project backend.ProjectContext) (Session, string, string, error) {
	if err := ctx.Err(); err != nil {
		return nil, "", "", err
	}
	parent, err := scratchParent(root)
	if err != nil {
		return nil, "", "", err
	}
	scratch, err := os.MkdirTemp(parent, string(a.Language())+"-lsp-")
	if err != nil {
		return nil, "", "", fmt.Errorf("create isolated LSP root: %w", err)
	}
	name := filepath.Base(file)
	target := filepath.Join(scratch, name)
	if err := pipeline.WriteAtomic(target, source); err != nil {
		return nil, "", "", errors.Join(fmt.Errorf("write isolated selected source: %w", err), removeScratch(scratch))
	}
	factory := a.Factory
	if factory == nil {
		factory = processFactory
	}
	binary := a.Config.Binary
	if a.Language() == backend.LanguageBash && project.Bash.BashBin != "" {
		binary = project.Bash.BashBin
	}
	if a.Language() == backend.LanguageMake && project.Make.MakeBin != "" {
		binary = project.Make.MakeBin
	}
	session, err := factory(ctx, scratch, binary, a.Config.Args)
	if err != nil {
		return nil, "", "", errors.Join(fmt.Errorf("start %s language server: %w", a.Language(), err), removeScratch(scratch))
	}
	rootURI := pathutil.FileURI(scratch)
	if _, err = session.Request(ctx, "initialize", map[string]any{"processId": nil, "rootUri": rootURI, "workspaceFolders": []map[string]string{{"uri": rootURI, "name": filepath.Base(root)}}, "capabilities": map[string]any{"general": map[string]any{"positionEncodings": []string{"utf-16"}}, textDocumentKey: map[string]any{"documentSymbol": map[string]any{"hierarchicalDocumentSymbolSupport": true}}}}); err != nil {
		return nil, "", "", errors.Join(fmt.Errorf("initialize %s server: %w", a.Language(), err), session.Close(), removeScratch(scratch))
	}
	if err := session.Notify(ctx, "initialized", map[string]any{}); err != nil {
		return nil, "", "", errors.Join(fmt.Errorf("initialize %s server: %w", a.Language(), err), session.Close(), removeScratch(scratch))
	}
	return session, scratch, pathutil.FileURI(target), nil
}

func (a *Adapter) readSymbols(ctx context.Context, project backend.ProjectContext, operation backend.Operation) (root, file string, source []byte, symbols []backend.ReadSymbol, limitations []string, retErr error) {
	root, file, source, err := a.resolve(project)
	if err != nil {
		return "", "", nil, nil, nil, &backend.Error{Operation: operation, Language: a.Language(), Err: err}
	}
	if !project.WorkspaceTrust.Allows(root) {
		return "", "", nil, nil, nil, &backend.WorkspaceTrustError{Operation: operation, Language: a.Language(), Workspace: root}
	}
	if a.Language() == backend.LanguageMake {
		if line := makeContinuationLine(source); line >= 0 {
			return "", "", nil, nil, nil, &backend.Error{Operation: operation, Language: a.Language(), Err: fmt.Errorf("make read projection cannot map continued declaration at line %d", line+1)}
		}
	}
	callerCtx := nonNilContext(ctx)
	requestCtx, cancel := context.WithTimeout(callerCtx, a.timeout())
	defer func() {
		if callerCtx.Err() == nil && errors.Is(requestCtx.Err(), context.DeadlineExceeded) && retErr != nil {
			retErr = errors.Join(retErr, ErrOperationTimeout)
		}
		cancel()
	}()
	session, scratch, uri, err := a.open(requestCtx, root, file, source, project)
	if err != nil {
		return "", "", nil, nil, nil, &backend.Error{Operation: operation, Language: a.Language(), Err: err}
	}
	defer func() {
		retErr = errors.Join(retErr, session.Close())
		if cleanupErr := os.RemoveAll(scratch); cleanupErr != nil {
			retErr = errors.Join(retErr, fmt.Errorf("remove isolated LSP workspace: %w", cleanupErr))
		}
	}()
	if err := session.Notify(requestCtx, "textDocument/didOpen", map[string]any{textDocumentKey: map[string]any{"uri": uri, "languageId": a.Config.LanguageID, "version": 1, "text": string(source)}}); err != nil {
		return "", "", nil, nil, nil, fmt.Errorf("open selected file for read: %w", err)
	}
	raw, err := session.Request(requestCtx, "textDocument/documentSymbol", map[string]any{textDocumentKey: map[string]string{"uri": uri}})
	if err != nil {
		return "", "", nil, nil, nil, fmt.Errorf("request read symbols: %w", err)
	}
	mapped, readLimitations, err := decodeReadSymbols(raw, source, a.Config.SymbolSyntax, uri, file)
	if err != nil {
		return "", "", nil, nil, nil, fmt.Errorf("decode %s read symbols: %w", a.Language(), err)
	}
	return root, file, source, mapped, readLimitations, nil
}

func scratchParent(root string) (string, error) {
	parent := filepath.Join(root, ".scratch")
	if err := os.Mkdir(parent, 0700); err != nil && !errors.Is(err, os.ErrExist) {
		return "", fmt.Errorf("create scratch root: %w", err)
	}
	info, err := os.Lstat(parent)
	if err != nil {
		return "", fmt.Errorf("inspect scratch root: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return "", ErrScratchOutsideRoot
	}
	resolved, err := filepath.EvalSymlinks(parent)
	if err != nil {
		return "", fmt.Errorf("resolve scratch root: %w", err)
	}
	if !pathutil.PathWithin(root, resolved) {
		return "", ErrScratchOutsideRoot
	}
	return resolved, nil
}

func removeScratch(path string) error {
	if err := os.RemoveAll(path); err != nil {
		return fmt.Errorf("remove isolated LSP workspace: %w", err)
	}
	return nil
}

func processFactory(ctx context.Context, root, binary string, args []string) (Session, error) {
	if binary == "" {
		return nil, errors.New("server executable is not configured")
	}
	path, err := exec.LookPath(binary)
	if err != nil {
		return nil, fmt.Errorf("find server executable %q: %w", binary, err)
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(abs)
	if err != nil || info.IsDir() || info.Mode()&0111 == 0 {
		return nil, fmt.Errorf("invalid server executable %q", binary)
	}
	cmd := exec.CommandContext(ctx, abs, args...) // #nosec G204 -- executable is explicitly configured or PATH-resolved after trust.
	cmd.Dir = root
	home := filepath.Join(root, ".home")
	config := filepath.Join(root, ".config")
	cache := filepath.Join(root, ".cache")
	for _, dir := range []string{home, config, cache} {
		if err := os.MkdirAll(dir, 0700); err != nil {
			return nil, err
		}
	}
	cmd.Env = []string{"PATH=" + os.Getenv("PATH"), "HOME=" + home, "XDG_CONFIG_HOME=" + config, "XDG_CACHE_HOME=" + cache, "LANG=C.UTF-8", "LC_ALL=C.UTF-8"}
	session := &processSession{reports: make(map[string][]backend.Diagnostic), versions: make(map[string]*int), wake: make(chan struct{}, 1)}
	client, err := lsp.NewProcessClient(cmd, lsp.WithRequestHandler(rejectRequest), lsp.WithNotificationHandler(session.record))
	if err != nil {
		return nil, err
	}
	session.client = client
	return session, nil
}
func rejectRequest(context.Context, lsp.Request) (json.RawMessage, *lsp.ErrorObject) {
	return nil, &lsp.ErrorObject{Code: -32601, Message: "server request is unsupported by read-only selected-file backend"}
}

type processSession struct {
	client   *lsp.Client
	mu       sync.Mutex
	reports  map[string][]backend.Diagnostic
	versions map[string]*int
	wake     chan struct{}
}

func (s *processSession) Request(c context.Context, m string, p any) (json.RawMessage, error) {
	return s.client.Request(c, m, p)
}
func (s *processSession) Notify(c context.Context, m string, p any) error {
	return s.client.Notify(c, m, p)
}
func (s *processSession) record(n lsp.Notification) {
	if n.Method != "textDocument/publishDiagnostics" {
		return
	}
	var raw struct {
		URI         string          `json:"uri"`
		Version     *int            `json:"version"`
		Diagnostics json.RawMessage `json:"diagnostics"`
	}
	if json.Unmarshal(n.Params, &raw) != nil || raw.URI == "" || len(raw.Diagnostics) == 0 || raw.Diagnostics[0] != '[' {
		return
	}
	var list []struct {
		Message  string      `json:"message"`
		Severity int         `json:"severity"`
		Range    sourceRange `json:"range"`
	}
	if json.Unmarshal(raw.Diagnostics, &list) != nil {
		return
	}
	ds := make([]backend.Diagnostic, 0, len(list))
	for _, d := range list {
		ds = append(ds, backend.Diagnostic{Message: d.Message, Severity: d.Severity, Location: &backend.SourceLocation{URI: raw.URI, Range: backend.Range{Start: backend.Position(d.Range.Start), End: backend.Position(d.Range.End)}}})
	}
	s.mu.Lock()
	s.reports[raw.URI] = ds
	s.versions[raw.URI] = raw.Version
	s.mu.Unlock()
	select {
	case s.wake <- struct{}{}:
	default:
	}
}
func (s *processSession) WaitDiagnostics(c context.Context, u string, v int) ([]backend.Diagnostic, error) {
	for {
		s.mu.Lock()
		ds, ok := s.reports[u]
		version := s.versions[u]
		if ok && (version == nil || *version >= v) {
			out := append([]backend.Diagnostic{}, ds...)
			s.mu.Unlock()
			return out, nil
		}
		s.mu.Unlock()
		select {
		case <-c.Done():
			return nil, c.Err()
		case <-s.wake:
		}
	}
}
func (s *processSession) Close() error { return s.client.Close() }

type position struct {
	Line      int `json:"line"`
	Character int `json:"character"`
}
type sourceRange struct {
	Start position `json:"start"`
	End   position `json:"end"`
}
type symbol struct {
	Name      string      `json:"name"`
	Kind      int         `json:"kind"`
	URI       string      `json:"uri"`
	Range     sourceRange `json:"range"`
	Selection sourceRange `json:"selectionRange"`
	Location  *struct {
		URI   string      `json:"uri"`
		Range sourceRange `json:"range"`
	} `json:"location"`
	Children  []symbol `json:"children"`
	Container string   `json:"containerName"`
}

// decodeSymbols accepts both Bash's flat SymbolInformation-like locations and Make's
// hierarchical DocumentSymbol shape; both are read-only lookup records from one URI.
func decodeSymbols(raw json.RawMessage, source []byte, syntax string) ([]symbol, error) {
	var values []symbol
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}
	if err := json.Unmarshal(raw, &values); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrMalformedSymbols, err)
	}
	var walk func([]symbol) ([]symbol, bool)
	walk = func(items []symbol) ([]symbol, bool) {
		out := make([]symbol, 0, len(items))
		for _, s := range items {
			r := s.Range
			if s.Location != nil {
				r = s.Location.Range
			}
			if s.Name == "" || s.Kind < 1 || !validRange(source, r) || (s.Selection != sourceRange{} && !validRange(source, s.Selection)) {
				if syntax == "make" {
					continue
				}
				return nil, false
			}
			if syntax == "make" && !validMakeSymbol(source, s) {
				continue
			}
			if syntax == "bash" && !validBashSymbol(source, s) {
				return nil, false
			}
			children, ok := walk(s.Children)
			if !ok {
				return nil, false
			}
			s.Children = children
			out = append(out, s)
		}
		return out, true
	}
	values, ok := walk(values)
	if !ok {
		return nil, ErrMalformedSymbols
	}
	return values, nil
}

func validMakeSymbol(source []byte, s symbol) bool {
	if s.Location != nil || s.Selection == (sourceRange{}) || s.Selection.Start.Line != s.Range.Start.Line || s.Selection.End.Line != s.Selection.Start.Line {
		return false
	}
	start, startErr := byteOffset(source, s.Selection.Start)
	end, endErr := byteOffset(source, s.Selection.End)
	rangeStart, rangeStartErr := byteOffset(source, s.Range.Start)
	rangeEnd, rangeEndErr := byteOffset(source, s.Range.End)
	if startErr != nil || endErr != nil || rangeStartErr != nil || rangeEndErr != nil || start < rangeStart || end > rangeEnd || end < start {
		return false
	}
	if string(source[start:end]) != s.Name {
		return false
	}
	lineStart := bytesLastNewline(source, start) + 1
	lineEnd := start
	for lineEnd < len(source) && source[lineEnd] != '\n' {
		lineEnd++
	}
	line := strings.TrimSpace(string(source[lineStart:lineEnd]))
	selected := string(source[start:end])
	if selected != s.Name {
		return false
	}
	switch s.Kind {
	case 12:
		colon := strings.IndexByte(line, ':')
		return colon >= 0 && strings.TrimSpace(line[:colon]) == s.Name
	case 13:
		if !strings.HasPrefix(line, s.Name) || len(line) < len(s.Name) {
			return false
		}
		rest := strings.TrimLeft(line[len(s.Name):], " \t")
		return strings.HasPrefix(rest, "=") || strings.HasPrefix(rest, ":=") || strings.HasPrefix(rest, "?=") || strings.HasPrefix(rest, "+=") || strings.HasPrefix(rest, "!=")
	}
	return false
}

func validBashSymbol(source []byte, s symbol) bool {
	r := s.Range
	if s.Location != nil {
		r = s.Location.Range
	}
	start, err := byteOffset(source, r.Start)
	if err != nil {
		return false
	}
	lineStart := bytesLastNewline(source, start) + 1
	lineEnd := start
	for lineEnd < len(source) && source[lineEnd] != '\n' {
		lineEnd++
	}
	line := strings.TrimLeft(string(source[lineStart:lineEnd]), " \t")
	for offset := 0; offset < len(line); {
		relative := strings.Index(line[offset:], s.Name)
		if relative < 0 {
			break
		}
		nameAt := offset + relative
		offset = nameAt + len(s.Name)
		if (nameAt > 0 && isIdentifierByte(line[nameAt-1])) || (nameAt+len(s.Name) < len(line) && isIdentifierByte(line[nameAt+len(s.Name)])) {
			continue
		}
		prefix := strings.TrimSpace(line[:nameAt])
		suffix := strings.TrimSpace(line[nameAt+len(s.Name):])
		switch s.Kind {
		case 12:
			if (prefix == "" || prefix == "function") && (strings.HasPrefix(suffix, "(") || strings.HasPrefix(suffix, "{")) {
				return true
			}
		case 13:
			if (prefix == "" || prefix == "local" || prefix == "declare" || prefix == "typeset" || prefix == "readonly" || prefix == "export") && (suffix == "" || strings.HasPrefix(suffix, "=")) {
				return true
			}
		}
	}
	return false
}

func bytesLastNewline(source []byte, before int) int {
	for i := before - 1; i >= 0; i-- {
		if source[i] == '\n' {
			return i
		}
	}
	return -1
}

func isIdentifierByte(b byte) bool {
	return b == '_' || b >= 'a' && b <= 'z' || b >= 'A' && b <= 'Z' || b >= '0' && b <= '9'
}
func validRange(src []byte, r sourceRange) bool {
	a, e := byteOffset(src, r.Start)
	b, f := byteOffset(src, r.End)
	return e == nil && f == nil && a <= b
}
func byteOffset(src []byte, p position) (int, error) {
	if p.Line < 0 || p.Character < 0 {
		return 0, errors.New("negative position")
	}
	line, start := 0, 0
	for i, b := range src {
		if b == '\n' {
			if line == p.Line {
				offset, err := utf16Offset(src[start:i], p.Character)
				return start + offset, err
			}
			line++
			start = i + 1
		}
	}
	if line != p.Line {
		return 0, errors.New("line outside source")
	}
	offset, err := utf16Offset(src[start:], p.Character)
	return start + offset, err
}
func utf16Offset(src []byte, want int) (int, error) {
	units := 0
	for off := 0; off < len(src); {
		r, n := utf8.DecodeRune(src[off:])
		if r == utf8.RuneError && n == 1 {
			return 0, errors.New("invalid utf-8")
		}
		u := 1
		if r > 0xffff {
			u = 2
		}
		if want == units {
			return off, nil
		}
		if want > units && want < units+u {
			return 0, errors.New("position splits codepoint")
		}
		units += u
		off += n
	}
	if want == units {
		return len(src), nil
	}
	return 0, errors.New("character outside line")
}
func selectSymbol(query, root, file string, src []byte, items []symbol) (*backend.LookupResult, error) {
	parts := strings.Split(strings.Trim(strings.TrimSpace(query), `"'`), ".")
	if slices.Contains(parts, "") {
		return nil, ErrMalformedSymbols
	}
	type match struct {
		s    symbol
		path []string
	}
	var matches []match
	var visit func([]string, []symbol)
	visit = func(parent []string, items []symbol) {
		for _, s := range items {
			base := append([]string{}, parent...)
			if len(base) == 0 && s.Container != "" {
				base = append(base, strings.Split(s.Container, ".")...)
			}
			path := append(append([]string{}, base...), strings.Split(s.Name, ".")...)
			if len(parts) <= len(path) && strings.Join(path[len(path)-len(parts):], ".") == strings.Join(parts, ".") {
				matches = append(matches, match{s, path})
			}
			visit(path, s.Children)
		}
	}
	visit(nil, items)
	if len(matches) == 0 {
		return nil, fmt.Errorf("%s: %w", query, ErrSymbolNotFound)
	}
	candidates := make([]*backend.SymbolCandidate, 0, len(matches))
	for _, m := range matches {
		r := m.s.Selection
		if r == (sourceRange{}) {
			r = m.s.Range
		}
		if m.s.Location != nil {
			r = m.s.Location.Range
		}
		line, col := r.Start.Line+1, r.Start.Character+1
		off, _ := byteOffset(src, r.Start)
		rel, _ := filepath.Rel(root, file)
		kind := symbolKind(m.s.Kind)
		candidate := &backend.SymbolCandidate{Name: m.s.Name, QualifiedName: strings.Join(m.path, "."), Kind: kind, File: rel, Line: line, Column: col, Offset: off, Location: backend.SourceLocation{URI: pathutil.FileURI(file), Range: backend.Range{Start: backend.Position(r.Start), End: backend.Position(r.End)}}}
		if len(m.path) > 1 {
			candidate.Receiver = strings.Join(m.path[:len(m.path)-1], ".")
		}
		candidates = append(candidates, candidate)
	}
	if len(candidates) > 1 {
		return &backend.LookupResult{Symbol: query, File: file, Ambiguous: true, Candidates: candidates}, nil
	}
	c := candidates[0]
	return &backend.LookupResult{Symbol: c.QualifiedName, File: c.File, Line: c.Line, Column: c.Column, Offset: c.Offset, Kind: c.Kind, Receiver: c.Receiver, Location: c.Location}, nil
}
func symbolKind(k int) string {
	switch k {
	case 5:
		return "class"
	case 6:
		return "method"
	case 7:
		return "property"
	case 8:
		return "field"
	case 9:
		return "constructor"
	case 10:
		return "enum"
	case 11:
		return "interface"
	case 12:
		return "function"
	case 13:
		return "variable"
	case 14:
		return "constant"
	default:
		return fmt.Sprintf("symbol-%d", k)
	}
}

func makeOutlineLimitations() []string {
	return []string{"continued declarations are rejected; conditionals, include-remapped symbols, define blocks, and interrupted recipe spans remain incomplete server projections"}
}

func makeContinuationLine(source []byte) int {
	lineNumber := 0
	start := 0
	for offset := 0; offset <= len(source); {
		end := offset
		for end < len(source) && source[end] != 10 && source[end] != 13 {
			end++
		}
		line := string(source[start:end])
		if len(line) > 0 && hasTrailingContinuation(line) {
			return lineNumber
		}
		if end == len(source) {
			break
		}
		offset = end + 1
		if source[end] == 13 && offset < len(source) && source[offset] == 10 {
			offset++
		}
		start = offset
		lineNumber++
	}
	return -1
}

func hasTrailingContinuation(line string) bool {
	return strings.HasSuffix(line, string(byte(92)))
}

func physicalSourceLine(source []byte, requested int) ([]byte, bool) {
	if requested < 0 {
		return nil, false
	}
	lineNumber := 0
	start := 0
	for offset := 0; offset <= len(source); {
		end := offset
		for end < len(source) && source[end] != 10 && source[end] != 13 {
			end++
		}
		if lineNumber == requested {
			return source[start:end], true
		}
		if end == len(source) {
			break
		}
		offset = end + 1
		if source[end] == 13 && offset < len(source) && source[offset] == 10 {
			offset++
		}
		start = offset
		lineNumber++
	}
	return nil, false
}

func isMakeIncludeLine(source []byte, line int) bool {
	text, ok := physicalSourceLine(source, line)
	if !ok {
		return false
	}
	fields := strings.Fields(strings.TrimSpace(string(text)))
	if len(fields) == 0 {
		return false
	}
	switch fields[0] {
	case "include", "-include", "sinclude":
		return true
	}
	return false
}

func matchReadSymbols(query string, symbols []backend.ReadSymbol) []backend.ReadSymbol {
	parts := strings.Split(strings.TrimSpace(query), ".")
	if len(parts) == 0 || slices.Contains(parts, "") {
		return nil
	}
	matches := make([]backend.ReadSymbol, 0)
	var visit func([]backend.ReadSymbol)
	visit = func(items []backend.ReadSymbol) {
		for _, item := range items {
			qualified := item.QualifiedName
			if qualified == "" {
				qualified = item.Name
			}
			path := strings.Split(qualified, ".")
			if len(parts) <= len(path) && strings.Join(path[len(path)-len(parts):], ".") == strings.Join(parts, ".") {
				matches = append(matches, item)
			}
			visit(item.Children)
		}
	}
	visit(symbols)
	return matches
}

func decodeReadSymbols(raw json.RawMessage, source []byte, syntax, uri, file string) ([]backend.ReadSymbol, []string, error) {
	var values []symbol
	if len(raw) == 0 || string(raw) == "null" {
		return []backend.ReadSymbol{}, []string{}, nil
	}
	if err := json.Unmarshal(raw, &values); err != nil {
		return nil, nil, fmt.Errorf("%w: %w", ErrMalformedSymbols, err)
	}
	if syntax != "bash" && syntax != "make" {
		return nil, nil, fmt.Errorf("%w: unsupported symbol syntax %q", ErrMalformedSymbols, syntax)
	}
	if err := validateReadURIs(values, uri); err != nil {
		return nil, nil, err
	}
	limitations := make(map[string]bool)
	var walk func([]symbol, []string) ([]backend.ReadSymbol, error)
	walk = func(items []symbol, parents []string) ([]backend.ReadSymbol, error) {
		out := make([]backend.ReadSymbol, 0, len(items))
		for _, item := range items {
			if item.URI != "" && item.URI != uri {
				return nil, fmt.Errorf("%w: symbol %q has a foreign URI", ErrMalformedSymbols, item.Name)
			}
			if item.Location != nil && item.Location.URI != uri {
				return nil, fmt.Errorf("%w: symbol %q has a foreign URI", ErrMalformedSymbols, item.Name)
			}
			if syntax == "make" && item.Kind == 3 {
				if !validMakeContainer(source, item) {
					return nil, fmt.Errorf("%w: unexpected Make namespace %q", ErrMalformedSymbols, item.Name)
				}
				limitations["conditional blocks are omitted because server ranges cover multiple source declarations"] = true
				continue
			}
			kind := ""
			switch item.Kind {
			case 12:
				if syntax == "bash" {
					kind = "function"
				} else {
					kind = "target"
				}
			case 13:
				kind = "variable"
			}
			if kind == "" {
				return nil, fmt.Errorf("%w: unsupported %s symbol kind %d for %q", ErrMalformedSymbols, syntax, item.Kind, item.Name)
			}
			var sourceRangeValue sourceRange
			if syntax == "bash" {
				if item.Location == nil || item.Location.URI != uri {
					return nil, fmt.Errorf("%w: Bash symbol %q has a missing or foreign URI", ErrMalformedSymbols, item.Name)
				}
				sourceRangeValue = item.Location.Range
				if item.Name == "" || !validBashReadSymbol(source, item) {
					return nil, fmt.Errorf("%w: invalid Bash symbol %q", ErrMalformedSymbols, item.Name)
				}
			} else {
				if item.Location != nil && item.Location.URI != uri {
					return nil, fmt.Errorf("%w: Make symbol %q has a foreign URI", ErrMalformedSymbols, item.Name)
				}
				sourceRangeValue = item.Range
				if item.Name == "" || item.Selection == (sourceRange{}) || !validMakeReadSymbol(source, item) {
					return nil, fmt.Errorf("%w: invalid or unmappable Make symbol %q", ErrMalformedSymbols, item.Name)
				}
				if isMakeIncludeLine(source, item.Selection.Start.Line) {
					limitations["included targets mapped onto include lines are omitted"] = true
					continue
				}
			}
			qualified := item.Name
			if syntax == "make" && len(parents) > 0 {
				qualified = strings.Join(append(append([]string{}, parents...), item.Name), ".")
			}
			readSymbol := backend.ReadSymbol{
				Name: item.Name, QualifiedName: qualified, Kind: kind, File: filepath.ToSlash(file),
				Range: backend.Range{Start: backend.Position(sourceRangeValue.Start), End: backend.Position(sourceRangeValue.End)},
			}
			if syntax == "bash" {
				readSymbol.SelectionRange = readSymbol.Range
			} else {
				readSymbol.SelectionRange = backend.Range{Start: backend.Position(item.Selection.Start), End: backend.Position(item.Selection.End)}
			}
			children, err := walk(item.Children, append(append([]string{}, parents...), item.Name))
			if err != nil {
				return nil, err
			}
			readSymbol.Children = children
			out = append(out, readSymbol)
		}
		return out, nil
	}
	mapped, err := walk(values, nil)
	if err != nil {
		return nil, nil, err
	}
	readLimitations := make([]string, 0, len(limitations))
	for limitation := range limitations {
		readLimitations = append(readLimitations, limitation)
	}
	slices.Sort(readLimitations)
	return mapped, readLimitations, nil
}

func validMakeReadSymbol(source []byte, s symbol) bool {
	if s.Location != nil || s.Selection == (sourceRange{}) || s.Selection.Start.Line != s.Range.Start.Line || s.Selection.End.Line != s.Selection.Start.Line {
		return false
	}
	start, startErr := readlsp.ByteOffset(source, backend.Position(s.Selection.Start))
	end, endErr := readlsp.ByteOffset(source, backend.Position(s.Selection.End))
	rangeStart, rangeStartErr := readlsp.ByteOffset(source, backend.Position(s.Range.Start))
	rangeEnd, rangeEndErr := readlsp.ByteOffset(source, backend.Position(s.Range.End))
	if startErr != nil || endErr != nil || rangeStartErr != nil || rangeEndErr != nil || start < rangeStart || end > rangeEnd || end <= start {
		return false
	}
	if isMakeIncludeLine(source, s.Selection.Start.Line) {
		return true
	}
	if string(source[start:end]) != s.Name {
		return false
	}
	line, ok := physicalSourceLine(source, s.Selection.Start.Line)
	if !ok {
		return false
	}
	text := strings.TrimSpace(string(line))
	switch s.Kind {
	case 12:
		colon := strings.IndexByte(text, byte(58))
		return colon >= 0 && strings.TrimSpace(text[:colon]) == s.Name
	case 13:
		if !strings.HasPrefix(text, s.Name) {
			return false
		}
		rest := strings.TrimLeft(text[len(s.Name):], " \t")
		for _, operator := range []string{"=", ":=", "?=", "+=", "!="} {
			if strings.HasPrefix(rest, operator) {
				return true
			}
		}
		return false
	}
	return false
}

func validMakeContainer(source []byte, item symbol) bool {
	if item.Name == "" || item.Location != nil {
		return false
	}
	start, startErr := readlsp.ByteOffset(source, backend.Position(item.Range.Start))
	end, endErr := readlsp.ByteOffset(source, backend.Position(item.Range.End))
	selectionStart, selectionStartErr := readlsp.ByteOffset(source, backend.Position(item.Selection.Start))
	selectionEnd, selectionEndErr := readlsp.ByteOffset(source, backend.Position(item.Selection.End))
	if startErr != nil || endErr != nil || selectionStartErr != nil || selectionEndErr != nil || start >= end || selectionStart < start || selectionEnd > end || selectionStart >= selectionEnd {
		return false
	}
	if !isMakeConditionalLine(source, item.Range.Start.Line) {
		return false
	}
	line, ok := physicalSourceLine(source, item.Range.Start.Line)
	if !ok {
		return false
	}
	sourceFields := strings.Fields(strings.TrimSpace(string(line)))
	nameFields := strings.Fields(strings.TrimSpace(item.Name))
	return len(sourceFields) > 0 && len(nameFields) > 0 && strings.EqualFold(sourceFields[0], nameFields[0])
}

func isMakeConditionalLine(source []byte, line int) bool {
	text, ok := physicalSourceLine(source, line)
	if !ok {
		return false
	}
	fields := strings.Fields(strings.TrimSpace(string(text)))
	if len(fields) == 0 {
		return false
	}
	switch strings.ToLower(fields[0]) {
	case "ifeq", "ifneq", "ifdef", "ifndef":
		return true
	}
	return false
}

func validateReadURIs(items []symbol, selectedURI string) error {
	for _, item := range items {
		if item.URI != "" && item.URI != selectedURI {
			return fmt.Errorf("%w: symbol %q has a foreign URI", ErrMalformedSymbols, item.Name)
		}
		if item.Location != nil && item.Location.URI != selectedURI {
			return fmt.Errorf("%w: symbol %q has a foreign URI", ErrMalformedSymbols, item.Name)
		}
		if err := validateReadURIs(item.Children, selectedURI); err != nil {
			return err
		}
	}
	return nil
}

func validBashReadSymbol(source []byte, s symbol) bool {
	if s.Location == nil {
		return false
	}
	start, startErr := readlsp.ByteOffset(source, backend.Position(s.Location.Range.Start))
	end, endErr := readlsp.ByteOffset(source, backend.Position(s.Location.Range.End))
	lineStart, lineStartErr := readlsp.ByteOffset(source, backend.Position{Line: s.Location.Range.Start.Line, Character: 0})
	if startErr != nil || endErr != nil || lineStartErr != nil || start >= end {
		return false
	}
	rawLine, ok := physicalSourceLine(source, s.Location.Range.Start.Line)
	if !ok {
		return false
	}
	leading := len(rawLine) - len(strings.TrimLeft(string(rawLine), " \t"))
	line := strings.TrimLeft(string(rawLine), " \t")
	lowerBound := start - lineStart - leading
	upperBound := len(line)
	if s.Location.Range.End.Line == s.Location.Range.Start.Line {
		upperBound = end - lineStart - leading
	}
	for offset := 0; offset < len(line); {
		relative := strings.Index(line[offset:], s.Name)
		if relative < 0 {
			break
		}
		nameAt := offset + relative
		offset = nameAt + len(s.Name)
		if nameAt < lowerBound || nameAt+len(s.Name) > upperBound || nameAt+len(s.Name) > len(line) {
			continue
		}
		if (nameAt > 0 && isIdentifierByte(line[nameAt-1])) || (nameAt+len(s.Name) < len(line) && isIdentifierByte(line[nameAt+len(s.Name)])) {
			continue
		}
		prefix := strings.TrimSpace(line[:nameAt])
		suffix := strings.TrimSpace(line[nameAt+len(s.Name):])
		switch s.Kind {
		case 12:
			if (prefix == "" || prefix == "function") && (strings.HasPrefix(suffix, "(") || strings.HasPrefix(suffix, "{")) {
				return true
			}
		case 13:
			if (prefix == "" || prefix == "local" || prefix == "declare" || prefix == "typeset" || prefix == "readonly" || prefix == "export") && (suffix == "" || strings.HasPrefix(suffix, "=")) {
				return true
			}
		}
	}
	return false
}
