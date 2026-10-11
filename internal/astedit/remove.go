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

// GoRemoveConstructKinds lists declaration and case kinds supported by Go removal.
var GoRemoveConstructKinds = []ConstructKind{ConstructFunction, ConstructMethod, ConstructType, ConstructConst, ConstructVar, ConstructCase, ConstructStatementOrExpr}

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
	class string
}

// Remove deletes one Go declaration or switch case after validating the resulting file syntax.
func Remove(ctx context.Context, filePath string, opts RemoveOptions, autoOrganizeImports bool) (string, []string, string, error) {
	if sourcebuf.FromContext(ctx) == nil {
		var diff string
		var checks []string
		var refusal string
		err := runBufferedFile(ctx, filePath, func(editCtx context.Context) error {
			var err error
			diff, checks, refusal, err = Remove(editCtx, filePath, opts, autoOrganizeImports)
			return err
		})
		return diff, checks, refusal, err
	}
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
	target, err := resolveRemoveTarget(fset, fileNode, content, opts, cleanPath)
	if err != nil {
		return "", nil, "", err
	}
	if opts.SafeDelete && target.class == "statement" {
		return "", nil, "", fmt.Errorf("safe_delete is supported only for eligible top-level declarations")
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
	if opts.Kind == "case" || opts.Kind == string(ConstructStatementOrExpr) {
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
	scan, err := scanModuleSources(ctx, root, cleanPath, fset.Position(target.pos).Offset, target.name, content)
	checks = append(checks, "module_go_source_scan")
	if err != nil {
		return nil, checks, "cannot complete module source scan: " + err.Error()
	}
	checks = append(checks, "module_reference_scan_complete")
	if scan.directiveFound {
		checks = append(checks, "special_directive_scan")
		return nil, checks, "safe_delete refuses modules containing Go generation or link-name directives"
	}
	checks = append(checks, "special_directive_scan")
	if scan.referenceFound {
		return nil, checks, fmt.Sprintf("safe_delete found another Go identifier named %q in the module source set", target.name)
	}
	if refusal, err := verifySafeDeleteTarget(ctx, root, cleanPath, target); err != nil {
		return nil, checks, "cannot complete compiler reference preflight: " + err.Error()
	} else if refusal != "" {
		return nil, checks, refusal
	}
	return &safeDeleteSnapshot{root: root, files: scan.revisions}, checks, ""
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

type moduleSourceScan struct {
	revisions      map[string][32]byte
	referenceFound bool
	directiveFound bool
}

type moduleSourcePath struct {
	absolute string
	relative string
}

func scanModuleSources(ctx context.Context, root, target string, targetOffset int, targetName string, targetContent []byte) (moduleSourceScan, error) {
	scan := moduleSourceScan{revisions: make(map[string][32]byte)}
	rootFS, err := os.OpenRoot(root)
	if err != nil {
		return moduleSourceScan{}, fmt.Errorf("open module root %s: %w", root, err)
	}
	// The selected file has already been resolved under the active workspace; root is its nearest module.
	// #nosec G703
	err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		return scanModuleSourceEntry(ctx, root, rootFS, target, targetOffset, targetName, targetContent, &scan, path, entry, walkErr)
	})
	closeErr := rootFS.Close()
	if closeErr != nil {
		err = errors.Join(err, fmt.Errorf("close module root %s: %w", root, closeErr))
	}
	return scan, err
}

func scanModuleSourceEntry(ctx context.Context, root string, rootFS *os.Root, target string, targetOffset int, targetName string, targetContent []byte, scan *moduleSourceScan, path string, entry fs.DirEntry, walkErr error) error {
	if walkErr != nil {
		return fmt.Errorf("walk %s: %w", path, walkErr)
	}
	if entry.IsDir() {
		return moduleDirectoryAction(root, path, entry.Name())
	}
	if err := validateModuleSourceEntry(root, path, entry); err != nil {
		return err
	}
	if filepath.Ext(path) != ".go" {
		return nil
	}
	sourcePath, err := resolveModuleSourcePath(root, path)
	if err != nil {
		return err
	}
	data, err := readModuleSource(ctx, rootFS, sourcePath)
	if err != nil {
		return err
	}
	if sourcePath.absolute == target && !bytes.Equal(data, targetContent) {
		return fmt.Errorf("selected file changed while scanning: %s", sourcePath.absolute)
	}
	scan.revisions[sourcePath.absolute] = sha256.Sum256(data)
	return inspectModuleSource(data, sourcePath.absolute, target, targetOffset, targetName, scan)
}

func moduleDirectoryAction(root, path, name string) error {
	if path == root {
		return nil
	}
	if strings.HasPrefix(name, ".") || strings.HasPrefix(name, "_") || name == "vendor" {
		return filepath.SkipDir
	}
	return nil
}

func validateModuleSourceEntry(root, path string, entry fs.DirEntry) error {
	if entry.Type()&os.ModeSymlink != 0 {
		return fmt.Errorf("symlink in module source set: %s", path)
	}
	if filepath.Base(path) == "go.mod" && filepath.Dir(path) != root {
		return fmt.Errorf("nested Go module in source set: %s", path)
	}
	if isUnsupportedModuleSource(path) {
		return fmt.Errorf("non-Go source in module prevents complete reference proof: %s", path)
	}
	if filepath.Ext(path) == ".go" && !entry.Type().IsRegular() {
		return fmt.Errorf("non-regular Go source in module source set: %s", path)
	}
	return nil
}

func isUnsupportedModuleSource(path string) bool {
	switch filepath.Ext(path) {
	case ".s", ".S", ".syso", ".c", ".h", ".cpp", ".cc", ".cxx", ".m", ".mm":
		return true
	default:
		return false
	}
}

func resolveModuleSourcePath(root, path string) (moduleSourcePath, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return moduleSourcePath{}, fmt.Errorf("resolve source path %s: %w", path, err)
	}
	relative, err := filepath.Rel(root, absolute)
	if err != nil {
		return moduleSourcePath{}, fmt.Errorf("resolve source path relative to module root: %w", err)
	}
	return moduleSourcePath{absolute: absolute, relative: relative}, nil
}

func readModuleSource(ctx context.Context, rootFS *os.Root, path moduleSourcePath) ([]byte, error) {
	if sourcebuf.FromContext(ctx) != nil {
		data, err := sourcebuf.ReadFile(ctx, path.absolute)
		if err != nil {
			return nil, fmt.Errorf("read %s from current source revision: %w", path.absolute, err)
		}
		return data, nil
	}
	file, err := rootFS.Open(path.relative)
	if err != nil {
		return nil, fmt.Errorf("open %s beneath module root: %w", path.absolute, err)
	}
	data, readErr := io.ReadAll(file)
	closeErr := file.Close()
	var resultErr error
	if readErr != nil {
		resultErr = fmt.Errorf("read %s: %w", path.absolute, readErr)
	}
	if closeErr != nil {
		resultErr = errors.Join(resultErr, fmt.Errorf("close %s: %w", path.absolute, closeErr))
	}
	if resultErr != nil {
		return nil, resultErr
	}
	return data, nil
}

func inspectModuleSource(data []byte, path, target string, targetOffset int, targetName string, scan *moduleSourceScan) error {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, data, parser.ParseComments|parser.AllErrors)
	if err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}
	collectModuleDirectives(file, scan)
	collectModuleReferences(file, fset, path, target, targetOffset, targetName, scan)
	return nil
}

func collectModuleDirectives(file *ast.File, scan *moduleSourceScan) {
	for _, group := range file.Comments {
		for _, comment := range group.List {
			text := strings.TrimSpace(comment.Text)
			if strings.HasPrefix(text, "//go:") && !strings.HasPrefix(text, "//go:build ") {
				scan.directiveFound = true
			}
		}
	}
}

func collectModuleReferences(file *ast.File, fset *token.FileSet, path, target string, targetOffset int, targetName string, scan *moduleSourceScan) {
	ast.Inspect(file, func(node ast.Node) bool {
		identifier, ok := node.(*ast.Ident)
		if !ok || identifier.Name != targetName {
			return true
		}
		if path == target && fset.Position(identifier.Pos()).Offset == targetOffset {
			return true
		}
		scan.referenceFound = true
		return true
	})
}

func validateSafeDeleteSnapshot(ctx context.Context, snapshot *safeDeleteSnapshot) error {
	if buffer := sourcebuf.FromContext(ctx); buffer != nil {
		if err := buffer.AssertDisk(); err != nil {
			return fmt.Errorf("disk preimage changed: %w", err)
		}
	}
	currentScan, err := scanModuleSources(ctx, snapshot.root, "", -1, "", nil)
	if err != nil {
		return err
	}
	current := currentScan.revisions
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

func resolveRemoveTarget(fset *token.FileSet, file *ast.File, content []byte, opts RemoveOptions, cleanPath string) (removeTarget, error) {
	if opts.Kind == string(ConstructConst) {
		opts.Kind = "constant"
	}
	if opts.Kind == string(ConstructVar) {
		opts.Kind = "variable"
	}
	switch opts.Kind {
	case "function", "method", "type", "constant", "variable":
		node, err := findTopLevelTarget(file, opts, cleanPath)
		return removeTarget{node: node}, err
	case "case":
		return findCaseTarget(fset, file, opts, cleanPath)
	case string(ConstructStatementOrExpr):
		return findStatementOrExpressionRemoveTarget(fset, file, content, opts)
	default:
		return removeTarget{}, fmt.Errorf("unsupported removal kind %q", opts.Kind)
	}
}

func findStatementOrExpressionRemoveTarget(fset *token.FileSet, file *ast.File, content []byte, opts RemoveOptions) (removeTarget, error) {
	ownerName := opts.InFunction
	if ownerName == "" {
		return removeTarget{}, fmt.Errorf("in_function is required for statement/expression removal")
	}
	if strings.TrimSpace(opts.Discriminator) == "" {
		return removeTarget{}, fmt.Errorf("discriminator is required for statement/expression removal")
	}
	owner, err := findConstructOwner(file, ownerName)
	if err != nil {
		return removeTarget{}, err
	}
	candidates, _ := collectStatementExpressionCandidates(fset, content, file, owner)
	var directMatches []constructCandidate
	var expressionMatches []constructCandidate
	for _, candidate := range candidates {
		if candidate.key != normalizeConstructSelector(opts.Discriminator) {
			continue
		}
		if isDirectOwnerStatementCandidate(candidate) {
			directMatches = append(directMatches, candidate)
		} else if candidate.class == "expression" {
			expressionMatches = append(expressionMatches, candidate)
		}
	}
	if opts.ConstructPath != "" {
		var selected *constructCandidate
		for _, candidate := range candidates {
			if candidate.key != normalizeConstructSelector(opts.Discriminator) {
				continue
			}
			if candidate.path == opts.ConstructPath {
				selected = &candidate
				break
			}
		}
		if selected == nil {
			return removeTarget{}, fmt.Errorf("construct_path %q does not select %q in %s; candidates:\n%s", opts.ConstructPath, opts.Discriminator, ownerName, formatSpanCandidates(fset, candidates))
		}
		if !isDirectOwnerStatementCandidate(*selected) {
			if selected.class == "expression" {
				return removeTarget{}, fmt.Errorf("cannot remove expression slot %q; select a direct statement", opts.Discriminator)
			}
			return removeTarget{}, fmt.Errorf("construct_path %q selects a nested statement; only direct owner statements can be removed", selected.path)
		}
		directMatches = []constructCandidate{*selected}
	}
	if len(directMatches) == 0 && len(expressionMatches) > 0 {
		return removeTarget{}, fmt.Errorf("cannot remove expression slot %q; select a direct statement", opts.Discriminator)
	}
	if len(directMatches) == 0 {
		return removeTarget{}, fmt.Errorf("statement/expression %q not found in %s; candidates:\n%s", opts.Discriminator, ownerName, formatSpanCandidates(fset, candidates))
	}
	if len(directMatches) > 1 {
		return removeTarget{}, fmt.Errorf("ambiguous direct statement %q in %s; set construct_path to one of:\n%s", opts.Discriminator, ownerName, formatSpanCandidates(fset, directMatches))
	}
	stmt, ok := directMatches[0].node.(ast.Stmt)
	if !ok {
		return removeTarget{}, fmt.Errorf("selected target is not a direct statement")
	}
	return removeTarget{node: stmt, owner: ownerName, class: "statement"}, nil
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
			if declarationStart, declarationEnd, found := declarationRemovalRange(gen, selected); found {
				return validateRemovalRange(fset, declarationStart, declarationEnd, contentLength)
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
