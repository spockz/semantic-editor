// Package rust captures rust-analyzer's bounded extract-function assist.
// WHY: This file validates scope, callee resolution, the complete candidate, and the helper rename before staging.
package rust

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/spockz/semantic-editor/internal/backend"
	"github.com/spockz/semantic-editor/internal/sourcebuf"
)

func (b *RustBackend) applyRustExtractFunction(ctx context.Context, request backend.CodeActionRequest) (result *backend.CodeActionResult, returnErr error) {
	if err := validateRustExtractFunctionRequest(request); err != nil {
		return nil, err
	}
	if ctx == nil {
		ctx = context.Background()
	}
	root, file, source, err := resolveRustProject(ctx, request.Project)
	if err != nil {
		return nil, err
	}
	if !request.Project.WorkspaceTrust.Allows(root) {
		return nil, &backend.WorkspaceTrustError{Operation: request.Operation, Language: backend.LanguageRust, Workspace: root}
	}
	ctx, source, ownsScopeBuffer, err := prepareRustCallableSource(ctx, root, file, source)
	if err != nil {
		return nil, err
	}
	if scopeErr := validateRustCallableFileScope(ctx, root, file, source); scopeErr != nil {
		return nil, scopeErr
	}
	if request.ExpectedRevision != "" && request.ExpectedRevision != rustSourceRevision(source) {
		return nil, fmt.Errorf("rust extraction source revision changed")
	}
	selectionStart, selectionEnd, callee, err := validateRustExtractFunctionSelection(source, request)
	if err != nil {
		return nil, err
	}
	if rustIdentifierPresent(source, request.NewName) {
		return nil, fmt.Errorf("requested Rust function name %q already appears in the selected file", request.NewName)
	}

	b.mu.Lock()
	defer b.mu.Unlock()
	if sessionErr := b.ensureRustSessionLocked(ctx, root); sessionErr != nil {
		return nil, sessionErr
	}
	session := b.session
	updated, err := runRustDocumentLifecycle(ctx, b, session, file, source, func(ctx context.Context, uri string) ([]byte, error) {
		return computeRustExtractFunction(ctx, session, root, file, uri, source, request, selectionStart, selectionEnd, callee)
	})
	if err != nil {
		return nil, err
	}
	if err := publishRustCallableExtraction(ctx, root, file, source, updated, ownsScopeBuffer); err != nil {
		return nil, err
	}
	return &backend.CodeActionResult{File: file, Before: source, After: updated, Files: []backend.CodeActionFileEdit{{File: file, Before: bytes.Clone(source), After: bytes.Clone(updated)}}}, nil
}

func prepareRustCallableSource(ctx context.Context, root, file string, source []byte) (context.Context, []byte, bool, error) {
	ctx, owned, err := prepareRustCallableScope(ctx, root)
	if err != nil {
		return nil, nil, false, err
	}
	bufferedSource, err := sourcebuf.ReadFile(ctx, file)
	if err != nil {
		return nil, nil, false, fmt.Errorf("read Rust extraction source through workspace buffer: %w", err)
	}
	if !bytes.Equal(source, bufferedSource) {
		return nil, nil, false, fmt.Errorf("rust extraction source changed while preparing package scope")
	}
	return ctx, bufferedSource, owned, nil
}

func publishRustCallableExtraction(ctx context.Context, root, file string, before, after []byte, owned bool) error {
	if err := stageRustExtraction(ctx, root, file, before, after); err != nil {
		return err
	}
	if owned {
		if err := sourcebuf.FromContext(ctx).Flush(ctx); err != nil {
			return fmt.Errorf("publish Rust extraction result: %w", err)
		}
	}
	return nil
}

func prepareRustCallableScope(ctx context.Context, root string) (context.Context, bool, error) {
	editCtx, _, owned, err := sourcebuf.Prepare(ctx, root)
	if err != nil {
		return nil, false, fmt.Errorf("prepare Rust callable workspace buffer: %w", err)
	}
	if err := observeRustCallableScope(editCtx, root); err != nil {
		return nil, false, fmt.Errorf("observe Rust package scope: %w", err)
	}
	if _, err := sourcebuf.ReadFile(editCtx, filepath.Join(root, "Cargo.toml")); err != nil {
		return nil, false, fmt.Errorf("observe Rust Cargo manifest: %w", err)
	}
	return editCtx, owned, nil
}

func observeRustCallableScope(ctx context.Context, root string) error {
	if err := sourcebuf.ObserveTree(ctx, root, ".rs"); err != nil {
		return err
	}
	if err := sourcebuf.ObserveSourceDirectory(ctx, root, ".rs"); err != nil {
		return err
	}
	for _, relative := range []string{"", "src", "examples", "tests", "benches", "src/bin"} {
		path := root
		if relative != "" {
			path = filepath.Join(root, relative)
		}
		info, err := os.Lstat(path)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			continue
		}
		if relative == "src" {
			if err := sourcebuf.ObserveSourceDirectory(ctx, path, ".rs"); err != nil {
				return err
			}
		}
		if relative != "" {
			if err := sourcebuf.ObserveSourceTree(ctx, path, ".rs"); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateRustCallableFileScope(ctx context.Context, root, file string, source []byte) error {
	if !rustConventionalCrateRoot(root, file) {
		return fmt.Errorf("rust callable extraction requires a conventional crate-root source file so cross-file name effects are bounded")
	}
	if other, err := rustOtherSourceFile(ctx, root, file); err != nil {
		return fmt.Errorf("inspect Rust package source files: %w", err)
	} else if other != "" {
		return fmt.Errorf("rust callable extraction currently requires a single-source Cargo package; %q can introduce cross-file name conflicts", other)
	}
	if rustHasUnresolvedNameExpansion(source) {
		return fmt.Errorf("rust callable extraction refuses files with unresolved imported or macro-expanded names because introduced-name conflicts cannot be proven")
	}
	if rustTokenPresent(source, "mod") {
		return fmt.Errorf("rust callable extraction refuses crate roots that declare modules because child-module name conflicts cannot be proven")
	}
	return nil
}

func rustOtherSourceFile(ctx context.Context, root, selected string) (string, error) {
	selected = filepath.Clean(selected)
	if other, err := rustExplicitCargoTargetSource(ctx, root, selected); err != nil || other != "" {
		return other, err
	}
	other := ""
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		return inspectRustPackageEntry(root, selected, path, entry, &other)
	})
	for path := range sourcebuf.Overlay(ctx, ".rs") {
		if path != selected && (other == "" || path < other) {
			other = path
		}
	}
	return other, err
}

func inspectRustPackageEntry(root, selected, path string, entry fs.DirEntry, other *string) error {
	relative, err := filepath.Rel(root, path)
	if err != nil {
		return err
	}
	autoTargetPath := rustCargoAutoTargetPath(filepath.ToSlash(relative))
	if entry.IsDir() && path != root {
		skip, err := rustSkipPackageDirectory(path, relative, entry.Name(), autoTargetPath)
		if err != nil {
			return err
		}
		if skip {
			return filepath.SkipDir
		}
	}
	if entry.Type()&os.ModeSymlink != 0 && autoTargetPath {
		*other = path
	}
	if !entry.IsDir() && strings.EqualFold(filepath.Ext(path), ".rs") && filepath.Clean(path) != selected && *other == "" {
		*other = path
	}
	return nil
}

func rustSkipPackageDirectory(path, relative, name string, autoTargetPath bool) (bool, error) {
	if name == ".git" || name == ".scratch" || !autoTargetPath && (relative == "target" || relative == "vendor") {
		return true, nil
	}
	if _, err := os.Stat(filepath.Join(path, "Cargo.toml")); err == nil && !autoTargetPath {
		return true, nil
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return false, err
	}
	return false, nil
}

func rustCargoAutoTargetPath(relative string) bool {
	if relative == "src" {
		return true
	}
	for _, prefix := range []string{"src/bin", "examples", "tests", "benches"} {
		if relative == prefix || strings.HasPrefix(relative, prefix+"/") || strings.HasPrefix(prefix, relative+"/") {
			return true
		}
	}
	return false
}

func rustExplicitCargoTargetSource(ctx context.Context, root, selected string) (string, error) {
	var manifest struct {
		Package struct {
			Build any `toml:"build"`
		} `toml:"package"`
		Lib struct {
			Path string `toml:"path"`
		} `toml:"lib"`
		Bin []struct {
			Path string `toml:"path"`
		} `toml:"bin"`
		Example []struct {
			Path string `toml:"path"`
		} `toml:"example"`
		Test []struct {
			Path string `toml:"path"`
		} `toml:"test"`
		Bench []struct {
			Path string `toml:"path"`
		} `toml:"bench"`
	}
	manifestPath := filepath.Join(root, "Cargo.toml")
	manifestSource, err := sourcebuf.ReadFile(ctx, manifestPath)
	if err != nil {
		return "", fmt.Errorf("read Cargo target configuration %q: %w", manifestPath, err)
	}
	if _, err := toml.Decode(string(manifestSource), &manifest); err != nil {
		return "", fmt.Errorf("read Cargo target configuration %q: %w", manifestPath, err)
	}
	switch build := manifest.Package.Build.(type) {
	case nil:
	case string:
		if build != "" {
			return filepath.Join(root, build), nil
		}
	case bool:
		if build {
			return filepath.Join(root, "build.rs"), nil
		}
	default:
		return "", fmt.Errorf("unsupported Cargo package build setting type %T", build)
	}
	paths := []string{manifest.Lib.Path}
	for _, targets := range [][]struct {
		Path string `toml:"path"`
	}{manifest.Bin, manifest.Example, manifest.Test, manifest.Bench} {
		for _, target := range targets {
			paths = append(paths, target.Path)
		}
	}
	for _, path := range paths {
		if path == "" {
			continue
		}
		resolved := filepath.Clean(filepath.Join(root, path))
		if resolved != selected {
			return resolved, nil
		}
	}
	return "", nil
}

func rustConventionalCrateRoot(root, file string) bool {
	relative, err := filepath.Rel(root, file)
	if err != nil {
		return false
	}
	switch filepath.ToSlash(relative) {
	case "lib.rs", "main.rs", "src/lib.rs", "src/main.rs":
		return true
	default:
		return false
	}
}

func computeRustExtractFunction(ctx context.Context, session RustSession, root, file, uri string, source []byte, request backend.CodeActionRequest, selectionStart, selectionEnd int, callee string) ([]byte, error) {
	symbolsRaw, err := session.Request(ctx, "textDocument/documentSymbol", map[string]any{lspTextDocumentKey: map[string]string{"uri": uri}})
	if err != nil {
		return nil, &RustError{Op: rustOpDocumentSymbol, File: file, Err: err}
	}
	symbols, err := decodeRustDocumentSymbols(symbolsRaw, root, source)
	if err != nil {
		return nil, &RustError{Op: rustOpDocumentSymbol, File: file, Err: err}
	}
	owner, ok := selectRustTopLevelFunction(symbols, source, request.EnclosingSymbol, selectionStart, selectionEnd)
	if !ok {
		return nil, &RustError{Op: rustOpDocumentSymbol, File: file, Err: fmt.Errorf("selected statements must belong to a top-level Rust function named %q", request.EnclosingSymbol)}
	}
	calleeSymbol, ok := selectRustTopLevelFunctionByName(symbols, callee)
	if !ok {
		return nil, &RustError{Op: rustOpDocumentSymbol, File: file, Err: fmt.Errorf("callee %q is not a same-file top-level Rust function", callee)}
	}
	if ownerErr := validateRustCallableOwner(owner, source, request.EnclosingSymbol, callee); ownerErr != nil {
		return nil, &RustError{Op: rustOpDocumentSymbol, File: file, Err: ownerErr}
	}
	if calleeErr := validateRustUnitCallee(calleeSymbol, source, callee); calleeErr != nil {
		return nil, &RustError{Op: rustOpDocumentSymbol, File: file, Err: calleeErr}
	}
	edits, action, err := requestRustExtractFunctionEdits(ctx, session, file, uri, request.Range)
	if err != nil {
		return nil, err
	}
	candidate, renamePosition, useSpan, bindingSpan, err := prepareRustExtractFunctionCandidate(source, edits, action.Edit, owner, request.Expression, selectionStart, selectionEnd)
	if err != nil {
		return nil, &RustError{Op: rustOpCodeAction, File: file, Err: fmt.Errorf("validate extract-function workspace edit: %w", err)}
	}
	if err := session.Notify(ctx, "textDocument/didChange", map[string]any{
		lspTextDocumentKey: map[string]any{"uri": uri, "version": 2},
		"contentChanges":   []map[string]string{{"text": string(candidate)}},
	}); err != nil {
		return nil, &RustError{Op: "didChange", File: file, Err: err}
	}
	return renameRustExtractBinding(ctx, session, file, uri, candidate, renamePosition, useSpan, bindingSpan, request.NewName)
}

func validateRustExtractFunctionRequest(request backend.CodeActionRequest) error {
	if request.Operation != backend.OperationExtractFunction || request.Kind != rustRefactorExtract || request.EnclosingSymbol == "" || !isSafeRustBindingIdentifier(request.NewName) || request.Range.End.Line < request.Range.Start.Line || request.Range.End.Line == request.Range.Start.Line && request.Range.End.Character <= request.Range.Start.Character {
		return fmt.Errorf("unsupported rust extract-function request")
	}
	return nil
}

func validateRustExtractFunctionSelection(source []byte, request backend.CodeActionRequest) (int, int, string, error) {
	start, err := rustByteOffset(source, rustPosition(request.Range.Start))
	if err != nil {
		return 0, 0, "", fmt.Errorf("resolve rust extraction selection: %w", err)
	}
	end, err := rustByteOffset(source, rustPosition(request.Range.End))
	if err != nil || end <= start || string(source[start:end]) != request.Expression {
		return 0, 0, "", fmt.Errorf("rust extraction selection must exactly match the selected source statement")
	}
	callee := ""
	statements := strings.SplitSeq(request.Expression, "\n")
	for statement := range statements {
		statement = strings.TrimSuffix(statement, "\r")
		currentCallee, ok := rustDirectCallStatement(statement)
		if !ok || callee != "" && callee != currentCallee {
			return 0, 0, "", fmt.Errorf("rust callable extraction requires contiguous direct calls to the same callee with literal-only arguments")
		}
		callee = currentCallee
	}
	return start, end, callee, nil
}

func rustDirectCallStatement(statement string) (string, bool) {
	trimmed := strings.TrimSpace(statement)
	if !strings.HasSuffix(trimmed, ");") || strings.ContainsAny(trimmed, "{}!?") {
		return "", false
	}
	body := strings.TrimSuffix(trimmed, ";")
	open := strings.IndexByte(body, '(')
	if open <= 0 || !strings.HasSuffix(body, ")") {
		return "", false
	}
	callee := strings.TrimSpace(body[:open])
	if !isSimpleRustIdentifier(callee) {
		return "", false
	}
	arguments := strings.TrimSpace(body[open+1 : len(body)-1])
	if arguments != "" && !rustSuffixedIntegerExpression(arguments) {
		return "", false
	}
	return callee, true
}

func selectRustTopLevelFunction(symbols []rustDocumentSymbol, source []byte, name string, selectedStart, selectedEnd int) (rustDocumentSymbol, bool) {
	for _, symbol := range symbols {
		if symbol.Name != name || symbol.Kind != 12 {
			continue
		}
		start, startErr := rustByteOffset(source, symbol.Range.Start)
		end, endErr := rustByteOffset(source, symbol.Range.End)
		if startErr == nil && endErr == nil && start < selectedStart && selectedEnd < end {
			return symbol, true
		}
	}
	return rustDocumentSymbol{}, false
}

func selectRustTopLevelFunctionByName(symbols []rustDocumentSymbol, name string) (rustDocumentSymbol, bool) {
	for _, symbol := range symbols {
		if symbol.Name == name && symbol.Kind == 12 {
			return symbol, true
		}
	}
	return rustDocumentSymbol{}, false
}

func validateRustCallableOwner(owner rustDocumentSymbol, source []byte, name, callee string) error {
	start, startErr := rustByteOffset(source, owner.Range.Start)
	end, endErr := rustByteOffset(source, owner.Range.End)
	if startErr != nil || endErr != nil || end <= start {
		return ErrRustMalformedResponse
	}
	ownerSource := source[start:end]
	open := bytes.IndexByte(ownerSource, '{')
	close := bytes.LastIndexByte(ownerSource, '}')
	if open <= 0 || close <= open {
		return fmt.Errorf("rust extraction owner must have a simple function body")
	}
	header := string(ownerSource[:open])
	if !strings.HasPrefix(strings.TrimSpace(header), "fn ") || !strings.Contains(header, "fn "+name+"(") || strings.ContainsAny(header, "<>") || strings.Contains(header, "async") || strings.Contains(header, "unsafe") || strings.Contains(header, "const") || strings.Contains(header, "extern") || strings.Contains(header, "where") {
		return fmt.Errorf("rust extraction owner must be an ordinary non-generic free function")
	}
	if rustTokenPresent([]byte(header), callee) {
		return fmt.Errorf("rust extraction refuses an owner whose parameters could shadow callee %q", callee)
	}
	body := string(ownerSource[open+1 : close])
	for _, forbidden := range []string{"let", "use", "fn", "const", "static", "struct", "enum", "type", "macro_rules", "if", "for", "while", "match", "loop", "return", "break", "continue", "await", "yield", "unsafe", "?"} {
		if rustTokenPresent([]byte(body), forbidden) {
			return fmt.Errorf("rust extraction owner contains a local binding, item, or control-flow construct that can change name resolution or scope")
		}
	}
	return nil
}

func rustTokenPresent(source []byte, expected string) bool {
	if expected == "?" {
		return bytes.Contains(source, []byte(expected))
	}
	for offset := 0; offset < len(source); {
		if !isRustIdentifierByte(source[offset]) || source[offset] >= '0' && source[offset] <= '9' {
			offset++
			continue
		}
		start := offset
		for offset < len(source) && isRustIdentifierByte(source[offset]) {
			offset++
		}
		if string(source[start:offset]) == expected {
			return true
		}
	}
	return false
}

func validateRustUnitCallee(symbol rustDocumentSymbol, source []byte, name string) error {
	start, startErr := rustByteOffset(source, symbol.Range.Start)
	end, endErr := rustByteOffset(source, symbol.Range.End)
	if startErr != nil || endErr != nil || end <= start {
		return ErrRustMalformedResponse
	}
	declaration := source[start:end]
	bodyStart := bytes.IndexByte(declaration, '{')
	if bodyStart <= 0 {
		return fmt.Errorf("rust extraction cannot prove callee %q returns unit", name)
	}
	header := string(declaration[:bodyStart])
	if !strings.HasPrefix(strings.TrimSpace(header), "fn ") || !strings.Contains(header, "fn "+name+"(") || strings.ContainsAny(header, "<>") || strings.Contains(header, "->") || strings.Contains(header, "async") || strings.Contains(header, "unsafe") || strings.Contains(header, "const") || strings.Contains(header, "extern") || strings.Contains(header, "where") {
		return fmt.Errorf("rust extraction requires same-file ordinary unit-returning callee %q", name)
	}
	return nil
}

func requestRustExtractFunctionEdits(ctx context.Context, session RustSession, file, uri string, selected backend.Range) ([]rustTextEdit, rustCodeAction, error) {
	actionRaw, err := session.Request(ctx, "textDocument/codeAction", map[string]any{
		lspTextDocumentKey: map[string]string{"uri": uri},
		"range":            selected,
		"context":          map[string]any{rustLSPDiagnosticsKey: []any{}, "only": []string{rustRefactorExtract}},
	})
	if err != nil {
		return nil, rustCodeAction{}, &RustError{Op: rustOpCodeAction, File: file, Err: err}
	}
	action, encoded, err := selectRustExtractFunctionAction(actionRaw)
	if err != nil {
		return nil, rustCodeAction{}, &RustError{Op: rustOpCodeAction, File: file, Err: err}
	}
	if rustCodeActionDisabled(action.Disabled) || hasRustCommand(action.Command) {
		return nil, rustCodeAction{}, &RustError{Op: rustOpCodeAction, File: file, Err: ErrRustRenameInvalidEdit}
	}
	if len(action.Edit) == 0 || string(action.Edit) == "null" {
		if len(action.Data) == 0 || string(action.Data) == "null" {
			return nil, rustCodeAction{}, ErrRustRenameInvalidEdit
		}
		resolved, resolveErr := session.Request(ctx, rustOpResolveAction, encoded)
		if resolveErr != nil {
			return nil, rustCodeAction{}, &RustError{Op: rustOpResolveAction, File: file, Err: resolveErr}
		}
		original := action
		if decodeErr := json.Unmarshal(resolved, &action); decodeErr != nil || rustCodeActionDisabled(action.Disabled) || action.Kind != original.Kind || !sameRustCodeActionIdentity(action.Data, original.Data, "extract_function:") || hasRustCommand(action.Command) || len(action.Edit) == 0 || string(action.Edit) == "null" {
			return nil, rustCodeAction{}, ErrRustRenameInvalidEdit
		}
	}
	edits, err := decodeRustSingleFileWorkspaceEdit(file, action.Edit, 1)
	if err != nil {
		return nil, rustCodeAction{}, fmt.Errorf("decode Rust extract-function edit: %w", err)
	}
	return edits, action, nil
}

func selectRustExtractFunctionAction(raw json.RawMessage) (rustCodeAction, json.RawMessage, error) {
	return selectUniqueRustCodeAction(raw, "extract-function", func(action rustCodeAction) bool {
		var data rustCodeActionData
		return json.Unmarshal(action.Data, &data) == nil &&
			action.Kind == rustRefactorExtract &&
			strings.HasPrefix(data.ID, "extract_function:")
	})
}

type rustFunctionEditEnvelope struct {
	spans      []rustEditSpan
	insertion  string
	insertAt   int
	helperName string
}

func decodeRustFunctionEditEnvelope(source []byte, edits []rustTextEdit, rawEdit json.RawMessage, owner rustDocumentSymbol, statement string, selectionStart, selectionEnd int) (rustFunctionEditEnvelope, error) {
	selectedStatements := strings.Split(strings.TrimSpace(statement), "\n")
	maxEdits := 2*len(selectedStatements) + 1
	if len(selectedStatements) == 0 || !validRustFunctionEditEnvelope(rawEdit, edits, maxEdits) {
		return rustFunctionEditEnvelope{}, fmt.Errorf("raw envelope and decoded edits differ: %w", ErrRustRenameInvalidEdit)
	}
	ownerEnd, err := rustByteOffset(source, owner.Range.End)
	if err != nil {
		return rustFunctionEditEnvelope{}, fmt.Errorf("owner end is invalid: %w", ErrRustRenameInvalidEdit)
	}
	spans, insertion, spanErr := rustFunctionEditSpans(source, edits, ownerEnd, selectionStart)
	if spanErr != nil {
		return rustFunctionEditEnvelope{}, spanErr
	}
	if !rustSafeRustHelperInsertion(insertion.NewText, statement) {
		return rustFunctionEditEnvelope{}, fmt.Errorf("native helper text is invalid: %w", ErrRustRenameInvalidEdit)
	}
	helperName, ok := rustExtractedHelperName(insertion.NewText)
	if !ok || !isSafeRustBindingIdentifier(helperName) {
		return rustFunctionEditEnvelope{}, fmt.Errorf("native helper name is invalid: %w", ErrRustRenameInvalidEdit)
	}
	insertAt, err := rustByteOffset(source, insertion.Range.Start)
	if err != nil {
		return rustFunctionEditEnvelope{}, fmt.Errorf("insertion coordinate is invalid: %w", ErrRustRenameInvalidEdit)
	}
	if insertAt <= selectionEnd {
		return rustFunctionEditEnvelope{}, fmt.Errorf("insertion precedes the selection: %w", ErrRustRenameInvalidEdit)
	}
	if !nonOverlappingRustEditSpans(spans) {
		return rustFunctionEditEnvelope{}, fmt.Errorf("extraction edits overlap: %w", ErrRustRenameInvalidEdit)
	}
	sort.Slice(spans, func(i, j int) bool { return spans[i].start > spans[j].start })
	return rustFunctionEditEnvelope{spans: spans, insertion: insertion.NewText, insertAt: insertAt, helperName: helperName}, nil
}

func validRustFunctionEditEnvelope(rawEdit json.RawMessage, edits []rustTextEdit, maxEdits int) bool {
	if len(edits) < 2 || len(edits) > maxEdits {
		return false
	}
	var object struct {
		DocumentChanges []struct {
			Edits []rustTextEdit `json:"edits"`
		} `json:"documentChanges"`
	}
	if json.Unmarshal(rawEdit, &object) != nil || len(object.DocumentChanges) != 1 || len(object.DocumentChanges[0].Edits) != len(edits) {
		return false
	}
	for i := range edits {
		if object.DocumentChanges[0].Edits[i] != edits[i] {
			return false
		}
	}
	return true
}

func rustFunctionEditSpans(source []byte, edits []rustTextEdit, ownerEnd, selectionStart int) ([]rustEditSpan, *rustTextEdit, error) {
	var insertion *rustTextEdit
	spans := make([]rustEditSpan, 0, len(edits)-1)
	for _, edit := range edits {
		start, startErr := rustByteOffset(source, edit.Range.Start)
		end, endErr := rustByteOffset(source, edit.Range.End)
		if startErr != nil || endErr != nil {
			return nil, nil, fmt.Errorf("invalid LSP edit range start=%+v (%w), end=%+v (%w): %w", edit.Range.Start, startErr, edit.Range.End, endErr, ErrRustRenameInvalidEdit)
		}
		if start == end {
			if insertion != nil || start < ownerEnd || start > len(source) || strings.TrimSpace(string(source[ownerEnd:start])) != "" {
				return nil, nil, fmt.Errorf("helper insertion is outside the owner boundary: %w", ErrRustRenameInvalidEdit)
			}
			insertion = &edit
			continue
		}
		if start < selectionStart || end > ownerEnd || end <= start {
			return nil, nil, fmt.Errorf("replacement edit [%d,%d) escapes the selected owner range [%d,%d): %w", start, end, selectionStart, ownerEnd, ErrRustRenameInvalidEdit)
		}
		spans = append(spans, rustEditSpan{start: start, end: end, text: edit.NewText})
	}
	if insertion == nil || len(spans) != len(edits)-1 {
		return nil, nil, fmt.Errorf("workspace edit does not contain one helper insertion and selection replacements: %w", ErrRustRenameInvalidEdit)
	}
	return spans, insertion, nil
}

func nonOverlappingRustEditSpans(spans []rustEditSpan) bool {
	for i := range spans {
		for j := i + 1; j < len(spans); j++ {
			if spans[i].start < spans[j].end && spans[j].start < spans[i].end {
				return false
			}
			if spans[i].start == spans[i].end && spans[j].start == spans[j].end && spans[i].start == spans[j].start {
				return false
			}
		}
	}
	return true
}

func prepareRustExtractFunctionCandidate(source []byte, edits []rustTextEdit, rawEdit json.RawMessage, owner rustDocumentSymbol, statement string, selectionStart, selectionEnd int) ([]byte, backend.Position, rustEditSpan, rustEditSpan, error) {
	envelope, err := decodeRustFunctionEditEnvelope(source, edits, rawEdit, owner, statement, selectionStart, selectionEnd)
	if err != nil {
		return nil, backend.Position{}, rustEditSpan{}, rustEditSpan{}, fmt.Errorf("decode validated edit envelope: %w", err)
	}
	selectedAfter := []byte(envelope.helperName + "();")
	spans := append([]rustEditSpan(nil), envelope.spans...)
	spans = append(spans, rustEditSpan{start: envelope.insertAt, end: envelope.insertAt, text: envelope.insertion})
	for i := 1; i < len(spans); i++ {
		for j := i; j > 0 && spans[j].start > spans[j-1].start; j-- {
			spans[j], spans[j-1] = spans[j-1], spans[j]
		}
	}
	candidate := applyRustEditSpans(source, spans)
	expected := make([]byte, 0, len(source)+len(envelope.insertion)+len(selectedAfter)-(selectionEnd-selectionStart))
	expected = append(expected, source[:selectionStart]...)
	expected = append(expected, selectedAfter...)
	expected = append(expected, source[selectionEnd:envelope.insertAt]...)
	expected = append(expected, envelope.insertion...)
	expected = append(expected, source[envelope.insertAt:]...)
	if !bytes.Equal(candidate, expected) {
		return nil, backend.Position{}, rustEditSpan{}, rustEditSpan{}, fmt.Errorf("edit does not match the exact extraction postimage: %w", ErrRustRenameInvalidEdit)
	}
	insertedStart := envelope.insertAt + len(selectedAfter) - (selectionEnd - selectionStart)
	functionNeedle := []byte("fn " + envelope.helperName + "() {")
	functionStart := bytes.Index(candidate[insertedStart:], functionNeedle)
	if functionStart < 0 {
		return nil, backend.Position{}, rustEditSpan{}, rustEditSpan{}, fmt.Errorf("inserted helper declaration was not found: %w", ErrRustRenameInvalidEdit)
	}
	bindingStart := insertedStart + functionStart + len("fn ")
	binding := rustEditSpan{start: bindingStart, end: bindingStart + len(envelope.helperName), text: envelope.helperName}
	use := rustEditSpan{start: selectionStart, end: selectionStart + len(envelope.helperName), text: envelope.helperName}
	position, err := rustPositionAtOffset(candidate, use.start)
	if err != nil || string(candidate[use.start:use.end]) != envelope.helperName || string(candidate[binding.start:binding.end]) != envelope.helperName {
		return nil, backend.Position{}, rustEditSpan{}, rustEditSpan{}, fmt.Errorf("helper rename anchors are invalid: %w", ErrRustRenameInvalidEdit)
	}
	return candidate, position, use, binding, nil
}

func rustSafeRustHelperInsertion(helper, statement string) bool {
	lines := strings.Split(strings.Trim(helper, "\n"), "\n")
	selected := strings.Split(strings.TrimSpace(statement), "\n")
	if len(lines) != len(selected)+2 || !strings.HasPrefix(lines[0], "fn ") || lines[len(lines)-1] != "}" {
		return false
	}
	for index, sourceLine := range selected {
		if strings.TrimSpace(lines[index+1]) != strings.TrimSpace(sourceLine) {
			return false
		}
	}
	return true
}

func rustExtractedHelperName(helper string) (string, bool) {
	line, _, _ := strings.Cut(strings.Trim(helper, "\n"), "\n")
	if !strings.HasPrefix(line, "fn ") || !strings.HasSuffix(line, "() {") {
		return "", false
	}
	return strings.TrimSuffix(strings.TrimPrefix(line, "fn "), "() {"), true
}
