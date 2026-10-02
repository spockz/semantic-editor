// Package astedit contains exact-node removals because structural deletion must preserve sibling syntax and attached comments.
package astedit

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"semedit/internal/pipeline"
	"semedit/internal/sourcebuf"
	"semedit/internal/symbol"
)

// RemoveOptions describes the exact node and selectors for a structural removal.
type RemoveOptions struct {
	Kind          string
	Symbol        string
	InFunction    string
	Discriminator string
	TargetCase    string
	ConstructPath string
	SafeDelete    bool
}

type safeDeleteSnapshot struct {
	root  string
	files map[string][32]byte
}

// SafeDeleteSnapshotError reports a source-set change after preflight and before publication.
type SafeDeleteSnapshotError struct{ Cause error }

// Error describes the changed source set.
func (e *SafeDeleteSnapshotError) Error() string {
	return fmt.Sprintf("module source set changed during preflight: %v", e.Cause)
}

// Unwrap returns the source-set change cause.
func (e *SafeDeleteSnapshotError) Unwrap() error { return e.Cause }

type removeTarget struct {
	node  ast.Node
	owner string
}

// Remove deletes one Go declaration or switch case after validating the resulting file syntax.
func Remove(ctx context.Context, filePath string, opts RemoveOptions, autoOrganizeImports bool) (string, []string, string, error) {
	cleanPath := filepath.Clean(filePath)
	content, err := sourcebuf.ReadFile(ctx, cleanPath)
	if err != nil {
		return "", nil, "", fmt.Errorf("read target file: %w", err)
	}
	fset := token.NewFileSet()
	fileNode, err := parser.ParseFile(fset, cleanPath, content, parser.ParseComments|parser.AllErrors)
	if err != nil {
		return "", nil, "", &SyntaxError{File: cleanPath, Pos: extractSyntaxPosition(fset, err), Cause: err, Err: ErrSyntax}
	}
	target, err := resolveRemoveTarget(fset, fileNode, opts, cleanPath)
	if err != nil {
		return "", nil, "", err
	}
	start, end, err := removalRange(fset, fileNode, target.node, len(content))
	if err != nil {
		return "", nil, "", err
	}
	checks := []string{"go_syntax_parse", "target_resolution"}
	var snapshot *safeDeleteSnapshot
	if opts.SafeDelete {
		var refusal string
		snapshot, checks, refusal = safeDeleteDeclaration(ctx, fset, fileNode, target.node, cleanPath, content, checks)
		if refusal != "" {
			return "", checks, refusal, nil
		}
	}
	oldText := strings.TrimSpace(string(content[start:end]))
	formatted, err := formatRemovedFile(cleanPath, content, start, end)
	if err != nil {
		return "", checks, "", err
	}
	if snapshot != nil {
		checks = append(checks, "module_source_unchanged_before_write")
	}
	var prepublish func() error
	if snapshot != nil {
		prepublish = func() error { return validateSafeDeleteSnapshot(ctx, snapshot) }
	}
	if err := publishRemovedFile(ctx, cleanPath, formatted, autoOrganizeImports, prepublish); err != nil {
		if snapshotErr, ok := errors.AsType[*SafeDeleteSnapshotError](err); ok {
			return "", checks, "", snapshotErr
		}
		return "", checks, "", err
	}
	targetName := opts.Symbol
	if opts.Kind == "case" {
		targetName = target.owner
	}
	return UnifiedDiff(targetName, oldText, ""), checks, "", nil
}

func safeDeleteDeclaration(ctx context.Context, fset *token.FileSet, file *ast.File, selected ast.Node, cleanPath string, content []byte, checks []string) (*safeDeleteSnapshot, []string, string) {
	checks = append(checks, "safe_delete_scope_check")
	target, refusal := classifySafeDeleteTarget(file, selected)
	if refusal != "" {
		return nil, checks, refusal
	}
	switch target.kind {
	case "constant":
		gen, value, ident := standaloneConstant(file, selected)
		if gen == nil || value == nil || ident == nil || gen.Lparen.IsValid() || len(gen.Specs) != 1 {
			return nil, checks, "safe_delete supports only an unexported standalone constant without grouped iota siblings"
		}
		checks = append(checks, "constant_resource_scope_check")
	case "value":
		checks = append(checks, "variable_effect_scope_check")
	default:
		checks = append(checks, "function_identity_scope_check")
	}
	root, err := nearestModuleRoot(filepath.Dir(cleanPath))
	if err != nil {
		return nil, checks, "cannot establish a complete Go module source set: " + err.Error()
	}
	files, referenceFound, directiveFound, err := scanModuleSources(ctx, root, cleanPath, fset.Position(target.pos).Offset, target.name, content)
	checks = append(checks, "module_go_source_scan")
	if err != nil {
		return nil, checks, "cannot complete module source scan: " + err.Error()
	}
	checks = append(checks, "module_reference_scan_complete")
	if directiveFound {
		checks = append(checks, "special_directive_scan")
		return nil, checks, "safe_delete refuses modules containing Go generation or link-name directives"
	}
	checks = append(checks, "special_directive_scan")
	if referenceFound {
		return nil, checks, fmt.Sprintf("safe_delete found another Go identifier named %q in the module source set", target.name)
	}
	if target.kind != "constant" {
		if refusal, err := verifySafeDeleteTarget(ctx, root, cleanPath, target); err != nil {
			return nil, checks, "cannot complete compiler reference preflight: " + err.Error()
		} else if refusal != "" {
			return nil, checks, refusal
		}
	}
	return &safeDeleteSnapshot{root: root, files: files}, checks, ""
}

func standaloneConstant(file *ast.File, selected ast.Node) (*ast.GenDecl, *ast.ValueSpec, *ast.Ident) {
	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.CONST || gen.Lparen.IsValid() || len(gen.Specs) != 1 || gen.Specs[0] != selected {
			continue
		}
		value, ok := selected.(*ast.ValueSpec)
		if !ok || len(value.Names) != 1 {
			return nil, nil, nil
		}
		return gen, value, value.Names[0]
	}
	return nil, nil, nil
}

func nearestModuleRoot(directory string) (string, error) {
	current, err := filepath.Abs(directory)
	if err != nil {
		return "", fmt.Errorf("resolve source directory: %w", err)
	}
	for {
		if info, statErr := os.Stat(filepath.Join(current, "go.mod")); statErr == nil && !info.IsDir() {
			return current, nil
		} else if statErr != nil && !os.IsNotExist(statErr) {
			return "", fmt.Errorf("inspect go.mod at %s: %w", current, statErr)
		}
		parent := filepath.Dir(current)
		if parent == current {
			return "", fmt.Errorf("no go.mod found above %s", directory)
		}
		current = parent
	}
}

func scanModuleSources(ctx context.Context, root, target string, targetOffset int, targetName string, targetContent []byte) (map[string][32]byte, bool, bool, error) {
	files := make(map[string][32]byte)
	referenceFound, directiveFound := false, false
	rootFS, err := os.OpenRoot(root)
	if err != nil {
		return nil, false, false, fmt.Errorf("open module root %s: %w", root, err)
	}
	// The selected file has already been resolved under the active workspace; root is its nearest module.
	// #nosec G703
	err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return fmt.Errorf("walk %s: %w", path, walkErr)
		}
		if entry.IsDir() && path != root {
			name := entry.Name()
			if strings.HasPrefix(name, ".") || strings.HasPrefix(name, "_") || name == "vendor" {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("symlink in module source set: %s", path)
		}
		if filepath.Base(path) == "go.mod" && filepath.Dir(path) != root {
			return fmt.Errorf("nested Go module in source set: %s", path)
		}
		extension := filepath.Ext(path)
		if extension == ".s" || extension == ".S" || extension == ".syso" || extension == ".c" || extension == ".h" || extension == ".cpp" || extension == ".cc" || extension == ".cxx" || extension == ".m" || extension == ".mm" {
			return fmt.Errorf("non-Go source in module prevents complete reference proof: %s", path)
		}
		if entry.IsDir() || extension != ".go" {
			return nil
		}
		if !entry.Type().IsRegular() {
			return fmt.Errorf("non-regular Go source in module source set: %s", path)
		}
		absolute, err := filepath.Abs(path)
		if err != nil {
			return fmt.Errorf("resolve source path %s: %w", path, err)
		}
		relative, err := filepath.Rel(root, absolute)
		if err != nil {
			return fmt.Errorf("resolve source path relative to module root: %w", err)
		}
		var data []byte
		if sourcebuf.FromContext(ctx) != nil {
			data, err = sourcebuf.ReadFile(ctx, absolute)
			if err != nil {
				return fmt.Errorf("read %s from current source revision: %w", absolute, err)
			}
		} else {
			dataFile, openErr := rootFS.Open(relative)
			if openErr != nil {
				return fmt.Errorf("open %s beneath module root: %w", absolute, openErr)
			}
			data, err = io.ReadAll(dataFile)
			closeErr := dataFile.Close()
			if err != nil {
				return fmt.Errorf("read %s: %w", absolute, err)
			}
			if closeErr != nil {
				return fmt.Errorf("close %s: %w", absolute, closeErr)
			}
		}
		if absolute == target && !bytes.Equal(data, targetContent) {
			return fmt.Errorf("selected file changed while scanning: %s", absolute)
		}
		files[absolute] = sha256.Sum256(data)
		scanSet := token.NewFileSet()
		parsed, err := parser.ParseFile(scanSet, absolute, data, parser.ParseComments|parser.AllErrors)
		if err != nil {
			return fmt.Errorf("parse %s: %w", absolute, err)
		}
		for _, group := range parsed.Comments {
			for _, comment := range group.List {
				text := strings.TrimSpace(comment.Text)
				if strings.HasPrefix(text, "//go:") && !strings.HasPrefix(text, "//go:build ") {
					directiveFound = true
				}
			}
		}
		ast.Inspect(parsed, func(node ast.Node) bool {
			ident, ok := node.(*ast.Ident)
			if !ok || ident.Name != targetName {
				return true
			}
			if absolute == target && scanSet.Position(ident.Pos()).Offset == targetOffset {
				return true
			}
			referenceFound = true
			return true
		})
		return nil
	})
	closeErr := rootFS.Close()
	if err == nil && closeErr != nil {
		err = fmt.Errorf("close module root %s: %w", root, closeErr)
	}
	return files, referenceFound, directiveFound, err
}

func validateSafeDeleteSnapshot(ctx context.Context, snapshot *safeDeleteSnapshot) error {
	if buffer := sourcebuf.FromContext(ctx); buffer != nil {
		if err := buffer.AssertDisk(); err != nil {
			return fmt.Errorf("disk preimage changed: %w", err)
		}
	}
	current, _, _, err := scanModuleSources(ctx, snapshot.root, "", -1, "", nil)
	if err != nil {
		return err
	}
	if len(current) != len(snapshot.files) {
		return fmt.Errorf("go source file set changed")
	}
	for path, expected := range snapshot.files {
		if actual, ok := current[path]; !ok || actual != expected {
			return fmt.Errorf("go source changed: %s", path)
		}
	}
	return nil
}

func resolveRemoveTarget(fset *token.FileSet, file *ast.File, opts RemoveOptions, cleanPath string) (removeTarget, error) {
	switch opts.Kind {
	case "function", "method", "type", "constant", "variable":
		node, err := findTopLevelTarget(file, opts, cleanPath)
		return removeTarget{node: node}, err
	case "case":
		return findCaseTarget(fset, file, opts, cleanPath)
	default:
		return removeTarget{}, fmt.Errorf("unsupported removal kind %q", opts.Kind)
	}
}

func findTopLevelTarget(file *ast.File, opts RemoveOptions, cleanPath string) (ast.Node, error) {
	receiver, name, err := symbol.ParseIdentifier(opts.Symbol)
	if err != nil {
		return nil, fmt.Errorf("parse symbol %q: %w", opts.Symbol, err)
	}
	var selected ast.Node
	for _, decl := range file.Decls {
		switch node := decl.(type) {
		case *ast.FuncDecl:
			if isSelectedFunction(node, opts.Kind, receiver, name) {
				if selected != nil {
					return nil, fmt.Errorf("symbol %q is ambiguous in %s", opts.Symbol, cleanPath)
				}
				selected = node
			}
		case *ast.GenDecl:
			for _, spec := range node.Specs {
				if matchesDeclarationSpec(node.Tok, spec, opts.Kind, name) {
					if value, ok := spec.(*ast.ValueSpec); ok && len(value.Names) != 1 {
						return nil, fmt.Errorf("cannot remove %q from a shared value spec; split the spec first", opts.Symbol)
					}
					if selected != nil {
						return nil, fmt.Errorf("symbol %q is ambiguous in %s", opts.Symbol, cleanPath)
					}
					selected = spec
				}
			}
		}
	}
	if selected == nil {
		return nil, fmt.Errorf("%s %q not found in %s", opts.Kind, opts.Symbol, cleanPath)
	}
	return selected, nil
}

func isSelectedFunction(fn *ast.FuncDecl, kind, receiver, name string) bool {
	if (kind != "function" && kind != "method") || fn.Name.Name != name || extractReceiverName(fn.Recv) != receiver {
		return false
	}
	return (kind == "method") == (fn.Recv != nil)
}

func matchesDeclarationSpec(tok token.Token, spec ast.Spec, kind, name string) bool {
	switch node := spec.(type) {
	case *ast.TypeSpec:
		return kind == "type" && node.Name.Name == name
	case *ast.ValueSpec:
		if kind == "constant" && tok != token.CONST || kind == "variable" && tok != token.VAR {
			return false
		}
		if kind != "constant" && kind != "variable" {
			return false
		}
		for _, ident := range node.Names {
			if ident.Name == name {
				return true
			}
		}
	}
	return false
}

func findCaseTarget(fset *token.FileSet, file *ast.File, opts RemoveOptions, cleanPath string) (removeTarget, error) {
	if opts.InFunction == "" {
		return removeTarget{}, fmt.Errorf("in_function is required for case removal")
	}
	receiver, name, err := symbol.ParseIdentifier(opts.InFunction)
	if err != nil {
		return removeTarget{}, fmt.Errorf("parse in_function %q: %w", opts.InFunction, err)
	}
	var owner *ast.FuncDecl
	for _, decl := range file.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok && fn.Name.Name == name && extractReceiverName(fn.Recv) == receiver {
			owner = fn
			break
		}
	}
	if owner == nil || owner.Body == nil {
		return removeTarget{}, fmt.Errorf("function %q not found or has no body in %s", opts.InFunction, cleanPath)
	}
	candidates := collectConstructCandidates(fset, owner.Body, ConstructCase)
	switchSelectors := caseSwitchSelectors(fset, owner.Body)
	matches := matchingCaseCandidates(fset, candidates, switchSelectors, opts)
	if len(matches) == 0 {
		return removeTarget{}, fmt.Errorf("case %q not found in %s; available candidates:\n%s", opts.TargetCase, opts.InFunction, formatConstructCandidates(fset, candidates, ConstructCase))
	}
	if len(matches) > 1 {
		return removeTarget{}, fmt.Errorf("ambiguous case %q in %s; set construct_path to one of:\n%s", opts.TargetCase, opts.InFunction, formatConstructCandidates(fset, matches, ConstructCase))
	}
	return removeTarget{node: matches[0].node, owner: opts.InFunction}, nil
}

func caseSwitchSelectors(fset *token.FileSet, body *ast.BlockStmt) map[*ast.CaseClause]string {
	selectors := make(map[*ast.CaseClause]string)
	ast.Inspect(body, func(node ast.Node) bool {
		var clauses *ast.BlockStmt
		var selector string
		switch stmt := node.(type) {
		case *ast.SwitchStmt:
			clauses = stmt.Body
			if stmt.Tag != nil {
				selector = nodeText(fset, stmt.Tag)
			}
		case *ast.TypeSwitchStmt:
			clauses = stmt.Body
			if stmt.Assign != nil {
				selector = nodeText(fset, stmt.Assign)
			}
		default:
			return true
		}
		if clauses != nil {
			for _, item := range clauses.List {
				if clause, ok := item.(*ast.CaseClause); ok {
					selectors[clause] = selector
				}
			}
		}
		return true
	})
	return selectors
}

func matchingCaseCandidates(fset *token.FileSet, candidates []constructCandidate, switchSelectors map[*ast.CaseClause]string, opts RemoveOptions) []constructCandidate {
	var matches []constructCandidate
	for _, candidate := range candidates {
		clause, ok := candidate.node.(*ast.CaseClause)
		if !ok || opts.TargetCase != "" && !candidateMatches(fset, candidate, ConstructCase, opts.TargetCase) {
			continue
		}
		if opts.Discriminator != "" && switchSelectors[clause] != opts.Discriminator {
			continue
		}
		if opts.ConstructPath != "" && candidate.path != opts.ConstructPath {
			continue
		}
		matches = append(matches, candidate)
	}
	return matches
}

func removalRange(fset *token.FileSet, file *ast.File, selected ast.Node, contentLength int) (int, int, error) {
	startNode, endNode := selected.Pos(), selected.End()
	for _, decl := range file.Decls {
		if gen, ok := decl.(*ast.GenDecl); ok {
			if startNode, endNode, found := declarationRemovalRange(gen, selected); found {
				return validateRemovalRange(fset, startNode, endNode, contentLength)
			}
		}
		if fn, ok := decl.(*ast.FuncDecl); ok && fn == selected && fn.Doc != nil {
			startNode = fn.Doc.Pos()
		}
	}
	return validateRemovalRange(fset, startNode, endNode, contentLength)
}

func declarationRemovalRange(gen *ast.GenDecl, selected ast.Node) (token.Pos, token.Pos, bool) {
	for _, spec := range gen.Specs {
		if spec != selected {
			continue
		}
		if len(gen.Specs) == 1 {
			start := gen.Pos()
			if gen.Doc != nil {
				start = gen.Doc.Pos()
			}
			return start, gen.End(), true
		}
		start, end := spec.Pos(), spec.End()
		var doc, comment *ast.CommentGroup
		switch item := spec.(type) {
		case *ast.TypeSpec:
			doc, comment = item.Doc, item.Comment
		case *ast.ValueSpec:
			doc, comment = item.Doc, item.Comment
		}
		if doc != nil {
			start = doc.Pos()
		}
		if comment != nil {
			end = comment.End()
		}
		return start, end, true
	}
	return token.NoPos, token.NoPos, false
}

func validateRemovalRange(fset *token.FileSet, startNode, endNode token.Pos, contentLength int) (int, int, error) {
	start, end := fset.Position(startNode).Offset, fset.Position(endNode).Offset
	if start < 0 || end < start || end > contentLength {
		return 0, 0, fmt.Errorf("invalid source range for removal")
	}
	return start, end, nil
}

func formatRemovedFile(cleanPath string, content []byte, start, end int) ([]byte, error) {
	updated := replaceBytes(content, start, end, nil)
	formatted, err := format.Source(updated)
	if err != nil {
		return nil, &SyntaxError{File: cleanPath, Cause: err, Err: ErrSyntax}
	}
	if _, err := parser.ParseFile(token.NewFileSet(), cleanPath, formatted, parser.ParseComments|parser.AllErrors); err != nil {
		return nil, &SyntaxError{File: cleanPath, Cause: err, Err: ErrSyntax}
	}
	return formatted, nil
}

func publishRemovedFile(ctx context.Context, cleanPath string, formatted []byte, autoOrganizeImports bool, check func() error) error {
	writeCheck := func() error {
		if check != nil {
			if err := check(); err != nil {
				return &SafeDeleteSnapshotError{Cause: err}
			}
		}
		return nil
	}
	if err := sourcebuf.WriteFileChecked(ctx, cleanPath, formatted, writeCheck); err != nil {
		return fmt.Errorf("write atomic: %w", err)
	}
	if autoOrganizeImports {
		if err := pipeline.OrganizeImports(ctx, filepath.Dir(cleanPath), cleanPath); err != nil {
			_ = pipeline.Format(ctx, filepath.Dir(cleanPath), cleanPath)
			return fmt.Errorf("organize imports: %w", err)
		}
		return nil
	}
	if err := pipeline.Format(ctx, filepath.Dir(cleanPath), cleanPath); err != nil {
		return fmt.Errorf("format file: %w", err)
	}
	return nil
}
