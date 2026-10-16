// Package gobackend implements typed Go reference search using one captured source snapshot and go/types identities.
package gobackend

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	backend "github.com/spockz/semantic-editor/internal/backend"
	"github.com/spockz/semantic-editor/internal/backend/pathutil"
	"github.com/spockz/semantic-editor/internal/symbol"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"sync"

	"golang.org/x/tools/go/packages"
)

var errNoTypedReferenceTarget = errors.New("no typed reference target")

type referenceSourceSnapshot struct {
	mu    sync.Mutex
	files map[string][]byte
	err   error
}

type referenceTarget struct {
	object      types.Object
	identity    string
	candidate   *symbol.Symbol
	path        string
	ident       *ast.Ident
	fset        *token.FileSet
	source      []byte
	declaration ast.Node
}

func (snapshot *referenceSourceSnapshot) parseFile(fset *token.FileSet, filename string, source []byte) (*ast.File, error) {
	if source == nil {
		var err error
		// #nosec G304 -- go/packages supplies filenames for the active build.
		source, err = os.ReadFile(filename)
		if err != nil {
			return nil, fmt.Errorf("read source %s: %w", filename, err)
		}
	}
	path, err := cleanAbsolutePath(filename)
	if err != nil {
		return nil, err
	}
	captured := append([]byte(nil), source...)
	snapshot.mu.Lock()
	if existing, ok := snapshot.files[path]; ok && !bytes.Equal(existing, captured) {
		snapshot.err = fmt.Errorf("source changed while packages were loading: %s", path)
	}
	if _, ok := snapshot.files[path]; !ok {
		snapshot.files[path] = captured
	}
	captureErr := snapshot.err
	snapshot.mu.Unlock()
	if captureErr != nil {
		return nil, captureErr
	}
	return parser.ParseFile(fset, filename, captured, parser.ParseComments)
}

func cleanAbsolutePath(path string) (string, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve absolute source path %s: %w", path, err)
	}
	return filepath.Clean(absolute), nil
}

func (snapshot *referenceSourceSnapshot) source(path string) ([]byte, bool) {
	key, err := cleanAbsolutePath(path)
	if err != nil {
		return nil, false
	}
	snapshot.mu.Lock()
	defer snapshot.mu.Unlock()
	source, ok := snapshot.files[key]
	return source, ok
}

func (snapshot *referenceSourceSnapshot) verifyUnchanged() error {
	snapshot.mu.Lock()
	if snapshot.err != nil {
		err := snapshot.err
		snapshot.mu.Unlock()
		return err
	}
	files := make(map[string][]byte, len(snapshot.files))
	maps.Copy(files, snapshot.files)
	snapshot.mu.Unlock()

	for path, expected := range files {
		// #nosec G304 -- paths come from Go packages parsed for this module.
		current, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("recheck source snapshot %s: %w", path, err)
		}
		if !bytes.Equal(current, expected) {
			return fmt.Errorf("source changed during reference analysis: %s", path)
		}
	}
	return nil
}

func referencesModuleRoot(project backend.ProjectContext) (string, error) {
	workspaceRoot := project.RootDir
	if workspaceRoot == "" {
		workspaceRoot = "."
	}
	workspaceRoot, err := cleanAbsolutePath(workspaceRoot)
	if err != nil {
		return "", err
	}
	if project.File != "" {
		selected := project.File
		if !filepath.IsAbs(selected) {
			selected = filepath.Join(workspaceRoot, selected)
		}
		selected, err = cleanAbsolutePath(selected)
		if err != nil {
			return "", err
		}
		if !pathWithin(workspaceRoot, selected) {
			return "", fmt.Errorf("selected file %s is outside the project root %s", selected, workspaceRoot)
		}
		moduleRoot, found, moduleErr := enclosingGoModule(filepath.Dir(selected))
		if moduleErr != nil {
			return "", moduleErr
		}
		if found && pathWithin(moduleRoot, selected) {
			return moduleRoot, nil
		}
	}
	moduleRoot, found, err := enclosingGoModule(workspaceRoot)
	if err != nil {
		return "", err
	}
	if found {
		return moduleRoot, nil
	}
	return "", fmt.Errorf("no enclosing Go module found for %s", workspaceRoot)
}

func pathWithin(root, path string) bool {
	relative, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	return relative == "." || (relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)))
}

func enclosingGoModule(start string) (string, bool, error) {
	current := start
	for {
		manifest := filepath.Join(current, "go.mod")
		info, err := os.Stat(manifest)
		if err == nil {
			if info.IsDir() {
				return "", false, fmt.Errorf("go module manifest path is a directory: %s", manifest)
			}
			return current, true, nil
		}
		if !os.IsNotExist(err) {
			return "", false, fmt.Errorf("inspect go module manifest %s: %w", manifest, err)
		}
		parent := filepath.Dir(current)
		if parent == current {
			return "", false, nil
		}
		current = parent
	}
}

func rejectActiveGoWorkspace(ctx context.Context, root string) error {
	command := exec.CommandContext(ctx, "go", "env", "GOWORK")
	command.Dir = root
	output, err := command.Output()
	if err != nil {
		return fmt.Errorf("inspect active Go workspace: %w", err)
	}
	workspace := strings.TrimSpace(string(output))
	if workspace != "" && workspace != "off" {
		return fmt.Errorf("active go.work is unsupported for references: %s; set GOWORK=off to use one module", workspace)
	}
	return nil
}

func loadReferencePackages(ctx context.Context, root string, snapshot *referenceSourceSnapshot) ([]*packages.Package, error) {
	configuration := &packages.Config{
		Context: ctx,
		Dir:     root,
		Tests:   true,
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedCompiledGoFiles |
			packages.NeedSyntax | packages.NeedTypes | packages.NeedTypesInfo |
			packages.NeedTypesSizes | packages.NeedImports | packages.NeedDeps | packages.NeedModule,
		BuildFlags: []string{"-mod=readonly"},
		ParseFile:  snapshot.parseFile,
	}
	loaded, err := packages.Load(configuration, "./...")
	if err != nil {
		return nil, fmt.Errorf("load Go module packages: %w", err)
	}
	if len(loaded) == 0 {
		return nil, fmt.Errorf("load Go module packages: no packages found under %s", root)
	}
	snapshot.mu.Lock()
	captureErr := snapshot.err
	snapshot.mu.Unlock()
	if captureErr != nil {
		return nil, fmt.Errorf("capture package source: %w", captureErr)
	}
	for _, pkg := range loaded {
		if pkg.IllTyped || len(pkg.Errors) > 0 {
			message := packageTypeError(pkg, make(map[string]bool))
			if message == "" {
				message = "package or imported dependency is ill-typed"
			}
			return nil, fmt.Errorf("load Go package %s: %s", pkg.PkgPath, message)
		}
		if pkg.TypesInfo == nil || pkg.Fset == nil {
			return nil, fmt.Errorf("load Go package %s: type information is unavailable", pkg.PkgPath)
		}
	}
	return loaded, nil
}

func originReferenceObject(object types.Object) types.Object {
	switch typed := object.(type) {
	case *types.Func:
		return typed.Origin()
	case *types.Var:
		return typed.Origin()
	case *types.TypeName:
		switch typeValue := typed.Type().(type) {
		case *types.Alias:
			return typeValue.Origin().Obj()
		case *types.Named:
			return typeValue.Origin().Obj()
		}
	}
	return object
}

func referenceObjectIdentity(object types.Object, fset *token.FileSet) (string, bool, error) {
	if object == nil || fset == nil {
		return "", false, nil
	}
	origin := originReferenceObject(object)
	if origin == nil || !origin.Pos().IsValid() {
		return "", false, nil
	}
	position := fset.PositionFor(origin.Pos(), false)
	if !position.IsValid() || position.Filename == "" || position.Offset < 0 {
		return "", false, nil
	}
	path, err := cleanAbsolutePath(position.Filename)
	if err != nil {
		return "", false, err
	}
	return fmt.Sprintf("%s:%d", path, position.Offset), true, nil
}

func resolveTypedTarget(root, selectedFile, query string, loaded []*packages.Package, originals map[string]bool, sources *referenceSourceSnapshot) (*referenceTarget, error) {
	receiver, name, err := symbol.ParseIdentifier(query)
	if err != nil {
		return nil, err
	}
	selectedPath, err := absoluteSelectedTargetPath(root, selectedFile)
	if err != nil {
		return nil, err
	}
	var targets map[string]*referenceTarget
	if receiver != "" {
		targets, err = qualifiedReferenceTargets(root, selectedPath, receiver, name, loaded, originals, sources)
	} else {
		targets, err = declarationReferenceTargets(root, selectedPath, name, loaded, originals, sources)
		if err == nil && len(targets) == 0 {
			targets, err = localReferenceTargets(root, selectedPath, name, loaded, originals, sources)
		}
	}
	if err != nil {
		return nil, err
	}
	if len(targets) == 0 {
		return nil, fmt.Errorf("%w: %q", symbol.ErrNotFound, query)
	}
	if len(targets) > 1 {
		return nil, referenceTargetAmbiguity(root, query, targets)
	}
	for _, target := range targets {
		return target, nil
	}
	return nil, fmt.Errorf("%w: %q", symbol.ErrNotFound, query)
}

func referenceTargetAmbiguity(root, query string, targets map[string]*referenceTarget) error {
	keys := make([]string, 0, len(targets))
	for key := range targets {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	candidates := make([]*backend.NamedTarget, 0, len(keys))
	for _, key := range keys {
		target := targets[key]
		read, err := readTargetSymbol(root, target)
		if err != nil {
			return err
		}
		selection := read.SelectionRange
		declaration := read.Range
		candidates = append(candidates, &backend.NamedTarget{Name: read.Name, QualifiedName: read.QualifiedName, Kind: read.Kind, Location: backend.SourceLocation{URI: pathutil.FileURI(target.path), Range: selection}, DeclarationRange: &declaration, Signature: read.Signature})
	}
	locations, err := backend.RenderAmbiguousCandidates(root, candidates)
	if err != nil {
		return err
	}
	return fmt.Errorf("%w: references for %q match %d candidates in disk source:\n%s", backend.ErrAmbiguous, query, len(candidates), locations)
}

// FindReferences resolves one Go declaration and searches its active module source snapshot.
func (GoBackend) FindReferences(ctx context.Context, request backend.ReferencesRequest) (*backend.ReferencesResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	workspaceRoot := request.Project.RootDir
	if workspaceRoot == "" {
		workspaceRoot = "."
	}
	workspaceRoot, err := cleanAbsolutePath(workspaceRoot)
	if err != nil {
		return nil, fmt.Errorf("find references: %w", err)
	}
	root, err := referencesModuleRoot(request.Project)
	if err != nil {
		return nil, fmt.Errorf("find references: %w", err)
	}
	if workspaceErr := rejectActiveGoWorkspace(ctx, root); workspaceErr != nil {
		return nil, fmt.Errorf("find references: %w", workspaceErr)
	}
	selectedFile, err := selectedReferenceFile(request.Project, root)
	if err != nil {
		return nil, err
	}

	sources := &referenceSourceSnapshot{files: make(map[string][]byte)}
	loaded, err := loadReferencePackages(ctx, root, sources)
	if err != nil {
		return nil, err
	}
	originals, err := originalModuleSources(root, loaded)
	if err != nil {
		return nil, &backend.Error{Operation: backend.OperationFindReferences, Language: backend.LanguageGo, Err: err}
	}
	target, err := resolveTypedTarget(root, selectedFile, request.Symbol, loaded, originals, sources)
	if err != nil {
		return nil, &backend.Error{Operation: backend.OperationFindReferences, Language: backend.LanguageGo, Err: err}
	}
	readSymbol, err := readTargetSymbol(workspaceRoot, target)
	if err != nil {
		return nil, err
	}
	references, files, err := collectTypedReferencesInScope(root, workspaceRoot, target, loaded, originals, sources)
	if err != nil {
		return nil, &backend.Error{Operation: backend.OperationFindReferences, Language: backend.LanguageGo, Err: err}
	}
	if verifyErr := sources.verifyUnchanged(); verifyErr != nil {
		return nil, &backend.Error{Operation: backend.OperationFindReferences, Language: backend.LanguageGo, Err: verifyErr}
	}
	scopePath, err := referenceScopePath(workspaceRoot, root)
	if err != nil {
		return nil, err
	}
	return &backend.ReferencesResult{
		Scope: backend.ReadScope{
			Kind: "workspace_active_build", Path: scopePath, Complete: true,
			Limitations: []string{
				"Search uses the active build configuration and includes test variants; nested modules are excluded.",
				"Cgo and other transformed compiler inputs are unsupported because references cannot be mapped reliably to original source.",
			},
		},
		Symbol: readSymbol, References: references, Files: files,
	}, nil
}

func selectedReferenceFile(project backend.ProjectContext, moduleRoot string) (string, error) {
	if project.File == "" {
		return "", nil
	}
	selected := project.File
	if !filepath.IsAbs(selected) {
		base := project.RootDir
		if base == "" {
			base = "."
		}
		selected = filepath.Join(base, selected)
	}
	selected, err := cleanAbsolutePath(selected)
	if err != nil {
		return "", err
	}
	if !pathWithin(moduleRoot, selected) {
		return "", fmt.Errorf("selected file %s is outside the searched Go module %s", selected, moduleRoot)
	}
	relative, err := filepath.Rel(moduleRoot, selected)
	if err != nil {
		return "", fmt.Errorf("make selected file relative to Go module: %w", err)
	}
	return relative, nil
}

func sourceRange(source []byte, start, end int) (backend.Range, error) {
	if start < 0 || end < start || end > len(source) {
		return backend.Range{}, fmt.Errorf("source offsets [%d,%d) exceed snapshot size %d", start, end, len(source))
	}
	return backend.Range{
		Start: backend.PositionFromByteOffset(source, start),
		End:   backend.PositionFromByteOffset(source, end),
	}, nil
}

func relativeSourcePath(root, path string) (string, error) {
	absolute, err := cleanAbsolutePath(path)
	if err != nil {
		return "", err
	}
	if !pathWithin(root, absolute) {
		return filepath.ToSlash(absolute), nil
	}
	relative, err := filepath.Rel(root, absolute)
	if err != nil {
		return "", fmt.Errorf("make source path relative to output root: %w", err)
	}
	return filepath.ToSlash(relative), nil
}

func readTargetSymbol(root string, target *referenceTarget) (backend.ReadSymbol, error) {
	start := target.fset.PositionFor(target.ident.Pos(), false).Offset
	end := target.fset.PositionFor(target.ident.End(), false).Offset
	if target.declaration != nil {
		start = target.fset.PositionFor(target.declaration.Pos(), false).Offset
		end = target.fset.PositionFor(target.declaration.End(), false).Offset
	}
	rangeValue, err := sourceRange(target.source, start, end)
	if err != nil {
		return backend.ReadSymbol{}, err
	}
	file, err := relativeSourcePath(root, target.path)
	if err != nil {
		return backend.ReadSymbol{}, err
	}
	qualifiedName := target.candidate.QualifiedName
	if qualifiedName == "" {
		qualifiedName = target.candidate.BuildQualifiedName()
	}
	read := backend.ReadSymbol{
		Name: target.candidate.Name, QualifiedName: qualifiedName, Kind: target.candidate.Kind,
		File: file, Range: rangeValue,
	}
	selectionStart := target.fset.PositionFor(target.ident.Pos(), false).Offset
	selectionEnd := target.fset.PositionFor(target.ident.End(), false).Offset
	selection, err := sourceRange(target.source, selectionStart, selectionEnd)
	if err != nil {
		return backend.ReadSymbol{}, err
	}
	read.SelectionRange = selection
	if function, ok := target.declaration.(*ast.FuncDecl); ok {
		signatureEnd := function.End()
		if function.Body != nil {
			signatureEnd = function.Body.Pos()
		}
		signatureEndOffset := target.fset.PositionFor(signatureEnd, false).Offset
		if signatureEndOffset >= start && signatureEndOffset <= len(target.source) {
			signature := strings.TrimSpace(string(target.source[start:signatureEndOffset]))
			if signature != "" {
				read.Signature = &signature
			}
		}
	}
	return read, nil
}

type referenceRecord struct {
	file      string
	offset    int
	reference backend.Reference
}

func collectTypedReferencesInScope(moduleRoot, outputRoot string, target *referenceTarget, loaded []*packages.Package, originals map[string]bool, sources *referenceSourceSnapshot) ([]backend.Reference, []backend.FileRevision, error) {
	records := make(map[string]referenceRecord)
	files := make(map[string][]byte)
	for _, pkg := range loaded {
		for _, file := range pkg.Syntax {
			if err := collectTypedReferencesInFile(moduleRoot, outputRoot, target, pkg, file, originals, sources, records); err != nil {
				return nil, nil, err
			}
		}
	}
	for path := range originals {
		source, ok := sources.source(path)
		if !ok {
			return nil, nil, fmt.Errorf("original active source was not captured by parser: %s", path)
		}
		files[path] = source
	}

	references := orderReferenceRecords(records)
	revisions, err := revisionRecords(outputRoot, files)
	return references, revisions, err
}

func collectTypedReferencesInFile(moduleRoot, outputRoot string, target *referenceTarget, pkg *packages.Package, file *ast.File, originals map[string]bool, sources *referenceSourceSnapshot, records map[string]referenceRecord) error {
	filePath, err := packageSyntaxPath(pkg, file)
	if err != nil {
		return err
	}
	if !originals[filePath] || !pathWithin(moduleRoot, filePath) {
		return nil
	}
	owners := make(map[string]*backend.ReadSymbol)
	for ident, object := range pkg.TypesInfo.Uses {
		if !ident.Pos().IsValid() || ident.Pos() < file.Pos() || ident.Pos() >= file.End() {
			continue
		}
		match, err := typedReferenceRecord(outputRoot, target, pkg, file, ident, object, originals, sources, owners)
		if err != nil {
			return err
		}
		if match.include {
			if _, exists := records[match.key]; exists {
				continue
			}
			records[match.key] = match.record
		}
	}
	return nil
}

type typedReferenceMatch struct {
	record  referenceRecord
	key     string
	include bool
}

func typedReferenceRecord(outputRoot string, target *referenceTarget, pkg *packages.Package, file *ast.File, ident *ast.Ident, object types.Object, originals map[string]bool, sources *referenceSourceSnapshot, owners map[string]*backend.ReadSymbol) (typedReferenceMatch, error) {
	identity, valid, err := referenceObjectIdentity(object, pkg.Fset)
	if err != nil || !valid || identity != target.identity {
		return typedReferenceMatch{}, err
	}
	position := pkg.Fset.PositionFor(ident.Pos(), false)
	absolute, err := cleanAbsolutePath(position.Filename)
	if err != nil || !originals[absolute] {
		return typedReferenceMatch{}, err
	}
	source, ok := sources.source(absolute)
	if !ok {
		return typedReferenceMatch{}, fmt.Errorf("reference source was not captured by parser: %s", absolute)
	}
	start := position.Offset
	end := pkg.Fset.PositionFor(ident.End(), false).Offset
	rangeValue, err := sourceRange(source, start, end)
	if err != nil {
		return typedReferenceMatch{}, err
	}
	filePath, err := relativeSourcePath(outputRoot, absolute)
	if err != nil {
		return typedReferenceMatch{}, err
	}
	key := fmt.Sprintf("%s:%d:%d", absolute, start, end)
	ownerResult, ownerErr := enclosingReferenceDeclaration(file, ident.Pos(), source, pkg.Fset, filePath, owners)
	if ownerErr != nil {
		return typedReferenceMatch{}, ownerErr
	}
	enclosing := ""
	if ownerResult.Declaration != nil {
		enclosing = ownerResult.Declaration.QualifiedName
	}
	record := referenceRecord{file: absolute, offset: start, reference: backend.Reference{File: filePath, Range: rangeValue, EnclosingSymbol: enclosing, EnclosingDeclaration: ownerResult.Declaration, Snippet: sourceLine(source, start)}}
	return typedReferenceMatch{record: record, key: key, include: true}, nil
}

func orderReferenceRecords(records map[string]referenceRecord) []backend.Reference {
	ordered := make([]referenceRecord, 0, len(records))
	for _, record := range records {
		ordered = append(ordered, record)
	}
	sort.Slice(ordered, func(i, j int) bool {
		if ordered[i].reference.File != ordered[j].reference.File {
			return ordered[i].reference.File < ordered[j].reference.File
		}
		return ordered[i].offset < ordered[j].offset
	})
	references := make([]backend.Reference, 0, len(ordered))
	for _, record := range ordered {
		references = append(references, record.reference)
	}
	return references
}

func revisionRecords(outputRoot string, files map[string][]byte) ([]backend.FileRevision, error) {
	paths := make([]string, 0, len(files))
	for path := range files {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	revisions := make([]backend.FileRevision, 0, len(paths))
	for _, path := range paths {
		filePath, err := relativeSourcePath(outputRoot, path)
		if err != nil {
			return nil, err
		}
		digest := sha256.Sum256(files[path])
		revisions = append(revisions, backend.FileRevision{File: filePath, Revision: hex.EncodeToString(digest[:])})
	}
	return revisions, nil
}

func sourceLine(source []byte, offset int) string {
	if offset < 0 || offset > len(source) {
		return ""
	}
	start := bytes.LastIndexByte(source[:offset], '\n') + 1
	end := len(source)
	if next := bytes.IndexByte(source[offset:], '\n'); next >= 0 {
		end = offset + next
	}
	return strings.TrimSpace(string(source[start:end]))
}

type referenceDeclarationResult struct {
	Declaration *backend.ReadSymbol
}

func enclosingReferenceDeclaration(file *ast.File, position token.Pos, source []byte, fset *token.FileSet, filePath string, owners map[string]*backend.ReadSymbol) (referenceDeclarationResult, error) {
	if file == nil || !position.IsValid() {
		return referenceDeclarationResult{}, nil
	}
	var selected *ast.FuncDecl
	ast.Inspect(file, func(node ast.Node) bool {
		declaration, ok := node.(*ast.FuncDecl)
		if !ok || position < declaration.Pos() || position >= declaration.End() {
			return true
		}
		if selected == nil || declaration.End()-declaration.Pos() < selected.End()-selected.Pos() {
			selected = declaration
		}
		return true
	})
	if selected == nil || selected.Name == nil {
		return referenceDeclarationResult{}, nil
	}
	qualified := selected.Name.Name
	if selected.Recv != nil && len(selected.Recv.List) > 0 {
		if receiver := referenceReceiverName(selected.Recv.List[0].Type); receiver != "" {
			qualified = receiver + "." + selected.Name.Name
		}
	}
	if cached, ok := owners[qualified]; ok {
		return referenceDeclarationResult{Declaration: cached}, nil
	}
	start := fset.PositionFor(selected.Pos(), false).Offset
	end := fset.PositionFor(selected.End(), false).Offset
	rangeValue, err := sourceRange(source, start, end)
	if err != nil {
		return referenceDeclarationResult{}, err
	}
	selection, err := sourceRange(source, fset.PositionFor(selected.Name.Pos(), false).Offset, fset.PositionFor(selected.Name.End(), false).Offset)
	if err != nil {
		return referenceDeclarationResult{}, err
	}
	signatureEnd := selected.End()
	if selected.Body != nil {
		signatureEnd = selected.Body.Pos()
	}
	signatureEndOffset := fset.PositionFor(signatureEnd, false).Offset
	if signatureEndOffset < start || signatureEndOffset > len(source) {
		return referenceDeclarationResult{}, fmt.Errorf("capture enclosing declaration signature range for %s", filePath)
	}
	signature := strings.TrimSpace(string(source[start:signatureEndOffset]))
	owner := &backend.ReadSymbol{Name: selected.Name.Name, QualifiedName: qualified, Kind: "function", File: filePath, Range: rangeValue, SelectionRange: selection}
	if selected.Recv != nil {
		owner.Kind = "method"
	}
	if signature != "" {
		owner.Signature = &signature
	}
	owners[qualified] = owner
	return referenceDeclarationResult{Declaration: owner}, nil
}

func referenceReceiverName(receiver ast.Expr) string {
	for {
		switch expression := receiver.(type) {
		case *ast.StarExpr:
			receiver = expression.X
		case *ast.IndexExpr:
			receiver = expression.X
		case *ast.IndexListExpr:
			receiver = expression.X
		case *ast.Ident:
			return expression.Name
		case *ast.SelectorExpr:
			return expression.Sel.Name
		default:
			return ""
		}
	}
}

func referenceScopePath(workspaceRoot, moduleRoot string) (string, error) {
	relative, err := filepath.Rel(workspaceRoot, moduleRoot)
	if err != nil {
		return "", fmt.Errorf("describe searched module scope: %w", err)
	}
	if pathWithin(workspaceRoot, moduleRoot) {
		return filepath.ToSlash(relative), nil
	}
	return filepath.ToSlash(moduleRoot), nil
}

func packageSyntaxPath(pkg *packages.Package, file *ast.File) (string, error) {
	if pkg == nil || pkg.Fset == nil || file == nil {
		return "", fmt.Errorf("package syntax has no file set")
	}
	position := pkg.Fset.PositionFor(file.Pos(), false)
	if !position.IsValid() || position.Filename == "" {
		return "", fmt.Errorf("package syntax has no physical source path")
	}
	return cleanAbsolutePath(position.Filename)
}

func typedReferenceTarget(root string, pkg *packages.Package, ident *ast.Ident, object types.Object, receiver string, sources *referenceSourceSnapshot) (*referenceTarget, error) {
	if pkg == nil || pkg.Fset == nil || ident == nil || object == nil {
		return nil, errNoTypedReferenceTarget
	}
	identity, valid, err := referenceObjectIdentity(object, pkg.Fset)
	if err != nil {
		return nil, err
	}
	if !valid {
		return nil, errNoTypedReferenceTarget
	}
	position := pkg.Fset.PositionFor(ident.Pos(), false)
	path, err := cleanAbsolutePath(position.Filename)
	if err != nil {
		return nil, err
	}
	source, ok := sources.source(path)
	if !ok {
		return nil, fmt.Errorf("typed declaration source was not captured by package parser: %s", path)
	}
	file, err := relativeSourcePath(root, path)
	if err != nil {
		return nil, err
	}
	kind := "value"
	switch typed := object.(type) {
	case *types.Func:
		kind = "function"
		if receiver != "" {
			kind = "method"
		}
	case *types.TypeName:
		kind = "type"
		_, isInterface := types.Unalias(typed.Type()).Underlying().(*types.Interface)
		if isInterface {
			kind = "interface"
		}
	case *types.Const:
		kind = "constant"
	case *types.Var:
		kind = "variable"
		if typed.IsField() {
			kind = "field"
		}
	}
	candidate := &symbol.Symbol{
		Name: ident.Name, Receiver: receiver, QualifiedName: ident.Name, Kind: kind,
		File: file, Offset: position.Offset,
	}
	if receiver != "" {
		candidate.QualifiedName = receiver + "." + ident.Name
	}
	lineRange, err := sourceRange(source, position.Offset, pkg.Fset.PositionFor(ident.End(), false).Offset)
	if err != nil {
		return nil, err
	}
	candidate.Line = lineRange.Start.Line + 1
	candidate.Column = lineRange.Start.Character + 1
	return &referenceTarget{object: object, identity: identity, candidate: candidate, path: path, ident: ident, fset: pkg.Fset, source: source, declaration: referenceDeclarationNode(pkg, ident)}, nil
}

func referenceDeclarationNode(pkg *packages.Package, ident *ast.Ident) ast.Node {
	if pkg == nil || ident == nil {
		return nil
	}
	var selected ast.Node
	for _, file := range pkg.Syntax {
		if ident.Pos() < file.Pos() || ident.Pos() >= file.End() {
			continue
		}
		ast.Inspect(file, func(node ast.Node) bool {
			if node == nil || ident.Pos() < node.Pos() || ident.Pos() >= node.End() {
				return true
			}
			if declaration := referenceNodeForIdent(node, ident); declaration != nil {
				selected = declaration
			}
			return true
		})
		break
	}
	return selected
}

func referenceNodeForIdent(node ast.Node, ident *ast.Ident) ast.Node {
	switch typed := node.(type) {
	case *ast.FuncDecl:
		if typed.Name == ident {
			return typed
		}
	case *ast.Field:
		if referenceFieldHasIdent(typed, ident) {
			return typed
		}
	case *ast.TypeSpec:
		if typed.Name == ident {
			return typed
		}
	case *ast.ValueSpec:
		if referenceNamesContain(typed.Names, ident) {
			return typed
		}
	}
	return nil
}

func referenceFieldHasIdent(field *ast.Field, ident *ast.Ident) bool {
	return referenceNamesContain(field.Names, ident)
}

func referenceNamesContain(names []*ast.Ident, ident *ast.Ident) bool {
	return slices.Contains(names, ident)
}

func findTypedDefinition(root, identity, receiver, name string, loaded []*packages.Package, originals map[string]bool, sources *referenceSourceSnapshot) (*referenceTarget, error) {
	for _, pkg := range loaded {
		for ident, definition := range pkg.TypesInfo.Defs {
			definitionIdentity, valid, err := referenceObjectIdentity(definition, pkg.Fset)
			if err != nil {
				return nil, err
			}
			if !valid || definitionIdentity != identity || ident.Name != name {
				continue
			}
			position := pkg.Fset.PositionFor(ident.Pos(), false)
			path, err := cleanAbsolutePath(position.Filename)
			if err != nil {
				return nil, err
			}
			if !originals[path] {
				continue
			}
			return typedReferenceTarget(root, pkg, ident, definition, receiver, sources)
		}
	}
	return nil, errNoTypedReferenceTarget
}

func localReferenceBindingKind(file *ast.File, ident *ast.Ident) string {
	parents := make(map[ast.Node]ast.Node)
	var stack []ast.Node
	ast.Inspect(file, func(node ast.Node) bool {
		if node == nil {
			stack = stack[:len(stack)-1]
			return false
		}
		if len(stack) > 0 {
			parents[node] = stack[len(stack)-1]
		}
		stack = append(stack, node)
		return true
	})
	return localBindingKindForParent(file, ident, parents, parents[ident])
}

func localBindingKindForParent(file *ast.File, ident *ast.Ident, parents map[ast.Node]ast.Node, parent ast.Node) string {
	switch declaration := parent.(type) {
	case *ast.AssignStmt:
		if declaration.Tok == token.DEFINE {
			for _, lhs := range declaration.Lhs {
				if lhs == ident {
					return "variable"
				}
			}
		}
	case *ast.RangeStmt:
		if declaration.Tok == token.DEFINE && (declaration.Key == ident || declaration.Value == ident) {
			return "variable"
		}
	case *ast.ValueSpec:
		return valueSpecBindingKind(file, parents, declaration)
	case *ast.Field:
		return fieldBindingKind(parents, declaration)
	}
	return ""
}

func valueSpecBindingKind(file *ast.File, parents map[ast.Node]ast.Node, declaration *ast.ValueSpec) string {
	genDecl, ok := parents[declaration].(*ast.GenDecl)
	if !ok || isFileDeclaration(file, genDecl) {
		return ""
	}
	if genDecl.Tok == token.CONST {
		return "constant"
	}
	return "variable"
}

func fieldBindingKind(parents map[ast.Node]ast.Node, declaration *ast.Field) string {
	fieldList, ok := parents[declaration].(*ast.FieldList)
	if !ok {
		return ""
	}
	switch owner := parents[fieldList].(type) {
	case *ast.FuncDecl:
		if owner.Recv == fieldList {
			return "receiver"
		}
	case *ast.FuncType:
		return functionFieldBindingKind(parents, fieldList, owner)
	}
	return ""
}

func functionFieldBindingKind(parents map[ast.Node]ast.Node, fieldList *ast.FieldList, owner *ast.FuncType) string {
	if owner.TypeParams == fieldList {
		return ""
	}
	switch parents[owner].(type) {
	case *ast.FuncDecl, *ast.FuncLit:
		if owner.Params == fieldList {
			return "parameter"
		}
		if owner.Results == fieldList {
			return "result"
		}
	}
	return ""
}

func isFileDeclaration(file *ast.File, target ast.Node) bool {
	for _, declaration := range file.Decls {
		if declaration == target {
			return true
		}
	}
	return false
}

func originalModuleSources(root string, loaded []*packages.Package) (map[string]bool, error) {
	testedPackages, err := testedModulePackages(root, loaded)
	if err != nil {
		return nil, err
	}
	originals := make(map[string]bool)
	for _, pkg := range loaded {
		include, err := isOriginalModulePackage(root, pkg, testedPackages)
		if err != nil {
			return nil, err
		}
		if !include {
			continue
		}
		if err := addOriginalPackageFiles(root, pkg, originals); err != nil {
			return nil, err
		}
	}
	if err := validateOriginalCompilerFiles(root, loaded, testedPackages, originals); err != nil {
		return nil, err
	}
	if len(originals) == 0 {
		return nil, fmt.Errorf("go module %s has no original active Go source files", root)
	}
	return originals, nil
}

func testedModulePackages(root string, loaded []*packages.Package) (map[string]bool, error) {
	tested := make(map[string]bool)
	for _, pkg := range loaded {
		if pkg == nil || pkg.Module == nil {
			continue
		}
		moduleDir, err := cleanAbsolutePath(pkg.Module.Dir)
		if err != nil {
			return nil, err
		}
		if moduleDir == root && pkg.PkgPath != "" {
			tested[pkg.PkgPath] = true
		}
	}
	return tested, nil
}

func isOriginalModulePackage(root string, pkg *packages.Package, tested map[string]bool) (bool, error) {
	if pkg == nil || pkg.Module == nil || isGeneratedTestMain(pkg, tested) {
		return false, nil
	}
	moduleDir, err := cleanAbsolutePath(pkg.Module.Dir)
	if err != nil {
		return false, err
	}
	return moduleDir == root, nil
}

func isGeneratedTestMain(pkg *packages.Package, tested map[string]bool) bool {
	if pkg == nil || pkg.Name != "main" || len(pkg.GoFiles) != 1 || filepath.Ext(pkg.GoFiles[0]) == ".go" {
		return false
	}
	for _, suffix := range []string{".test", ".test.exe"} {
		if generatedTestPackageMatches(pkg.PkgPath, suffix, tested) || generatedTestPackageMatches(pkg.ID, suffix, tested) {
			return true
		}
	}
	return false
}

func generatedTestPackageMatches(packagePath, suffix string, tested map[string]bool) bool {
	candidate := strings.TrimSuffix(packagePath, suffix)
	return candidate != packagePath && tested[candidate]
}

func addOriginalPackageFiles(root string, pkg *packages.Package, originals map[string]bool) error {
	for _, name := range pkg.GoFiles {
		if filepath.Ext(name) != ".go" {
			continue
		}
		path, err := packageSourcePath(pkg.Module.Dir, name)
		if err != nil {
			return err
		}
		if pathWithin(root, path) {
			originals[path] = true
		}
	}
	return nil
}

func validateOriginalCompilerFiles(root string, loaded []*packages.Package, tested, originals map[string]bool) error {
	for _, pkg := range loaded {
		include, err := isOriginalModulePackage(root, pkg, tested)
		if err != nil {
			return err
		}
		if !include {
			continue
		}
		for _, name := range pkg.CompiledGoFiles {
			path, err := packageSourcePath(pkg.Module.Dir, name)
			if err != nil {
				return err
			}
			if !originals[path] {
				return fmt.Errorf("go reference analysis cannot guarantee source mappings for transformed or generated compiler input %s in package %s", path, pkg.PkgPath)
			}
		}
	}
	return nil
}

func packageSourcePath(moduleDir, name string) (string, error) {
	if !filepath.IsAbs(name) {
		name = filepath.Join(moduleDir, name)
	}
	return cleanAbsolutePath(name)
}

func packageTypeError(pkg *packages.Package, visited map[string]bool) string {
	if pkg == nil {
		return ""
	}
	if pkg.ID != "" && visited[pkg.ID] {
		return ""
	}
	if pkg.ID != "" {
		visited[pkg.ID] = true
	}
	messages := make(map[string]bool)
	for _, packageError := range pkg.Errors {
		message := packageError.Msg
		if packageError.Pos != "" {
			message = packageError.Pos + ": " + message
		}
		if pkg.PkgPath != "" {
			message = pkg.PkgPath + ": " + message
		}
		messages[message] = true
	}
	importPaths := make([]string, 0, len(pkg.Imports))
	for importPath := range pkg.Imports {
		importPaths = append(importPaths, importPath)
	}
	sort.Strings(importPaths)
	for _, importPath := range importPaths {
		for message := range strings.SplitSeq(packageTypeError(pkg.Imports[importPath], visited), "\n") {
			if message != "" {
				messages[message] = true
			}
		}
	}
	ordered := make([]string, 0, len(messages))
	for message := range messages {
		ordered = append(ordered, message)
	}
	sort.Strings(ordered)
	if len(ordered) > 0 {
		return strings.Join(ordered, "\n")
	}
	if pkg.IllTyped {
		return "package or imported dependency is ill-typed"
	}
	return ""
}

func extractReceiver(receiver *ast.FieldList) string {
	if receiver == nil || len(receiver.List) == 0 {
		return ""
	}
	typeExpr := receiver.List[0].Type
	for {
		switch value := typeExpr.(type) {
		case *ast.StarExpr:
			typeExpr = value.X
		case *ast.IndexExpr:
			typeExpr = value.X
		case *ast.IndexListExpr:
			typeExpr = value.X
		case *ast.Ident:
			return value.Name
		default:
			return ""
		}
	}
}

func absoluteSelectedTargetPath(root, selectedFile string) (string, error) {
	if selectedFile == "" {
		return "", nil
	}
	path := selectedFile
	if !filepath.IsAbs(path) {
		path = filepath.Join(root, path)
	}
	return cleanAbsolutePath(path)
}

func addTypedReferenceTarget(root string, targets map[string]*referenceTarget, pkg *packages.Package, ident *ast.Ident, object types.Object, receiver string, sources *referenceSourceSnapshot) error {
	target, err := typedReferenceTarget(root, pkg, ident, object, receiver, sources)
	if errors.Is(err, errNoTypedReferenceTarget) {
		return nil
	}
	if err != nil {
		return err
	}
	targets[target.identity] = target
	return nil
}

func qualifiedReferenceTargets(root, selectedPath, receiver, name string, loaded []*packages.Package, originals map[string]bool, sources *referenceSourceSnapshot) (map[string]*referenceTarget, error) {
	targets := make(map[string]*referenceTarget)
	for _, pkg := range loaded {
		for _, file := range pkg.Syntax {
			filePath, err := packageSyntaxPath(pkg, file)
			if err != nil {
				return nil, err
			}
			if !originals[filePath] {
				continue
			}
			if err := collectQualifiedTargetsInFile(root, selectedPath, receiver, name, filePath, pkg, file, loaded, originals, sources, targets); err != nil {
				return nil, err
			}
		}
	}
	return targets, nil
}

func collectQualifiedTargetsInFile(root, selectedPath, receiver, name, filePath string, pkg *packages.Package, file *ast.File, loaded []*packages.Package, originals map[string]bool, sources *referenceSourceSnapshot, targets map[string]*referenceTarget) error {
	for _, declaration := range file.Decls {
		general, ok := declaration.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, raw := range general.Specs {
			if err := collectQualifiedTargetSpec(root, selectedPath, receiver, name, filePath, pkg, raw, loaded, originals, sources, targets); err != nil {
				return err
			}
		}
	}
	return nil
}

func collectQualifiedTargetSpec(root, selectedPath, receiver, name, filePath string, pkg *packages.Package, raw ast.Spec, loaded []*packages.Package, originals map[string]bool, sources *referenceSourceSnapshot, targets map[string]*referenceTarget) error {
	spec, ok := raw.(*ast.TypeSpec)
	if !ok || spec.Name.Name != receiver {
		return nil
	}
	typeName, ok := pkg.TypesInfo.Defs[spec.Name].(*types.TypeName)
	if !ok {
		return nil
	}
	target, err := qualifiedMemberTarget(root, selectedPath, receiver, name, filePath, pkg, typeName, loaded, originals, sources)
	if errors.Is(err, errNoTypedReferenceTarget) {
		return nil
	}
	if err != nil {
		return err
	}
	targets[target.identity] = target
	return nil
}

func qualifiedMemberTarget(root, selectedPath, receiver, name, receiverPath string, pkg *packages.Package, typeName *types.TypeName, loaded []*packages.Package, originals map[string]bool, sources *referenceSourceSnapshot) (*referenceTarget, error) {
	_, isInterface := types.Unalias(typeName.Type()).Underlying().(*types.Interface)
	if isInterface && selectedPath != "" && receiverPath != selectedPath {
		return nil, errNoTypedReferenceTarget
	}
	member, index, _ := types.LookupFieldOrMethod(typeName.Type(), true, pkg.Types, name)
	if member == nil {
		return nil, errNoTypedReferenceTarget
	}
	if variable, ok := member.(*types.Var); ok && variable.IsField() && len(index) > 1 {
		return nil, errNoTypedReferenceTarget
	}
	if function, ok := member.(*types.Func); ok && !isInterface {
		if len(index) > 1 {
			return nil, errNoTypedReferenceTarget
		}
		declaredHere, err := methodDeclaredOnReceiver(function, typeName, pkg.Fset)
		if err != nil {
			return nil, err
		}
		if !declaredHere {
			return nil, errNoTypedReferenceTarget
		}
	}
	identity, valid, err := referenceObjectIdentity(member, pkg.Fset)
	if err != nil {
		return nil, err
	}
	if !valid {
		return nil, errNoTypedReferenceTarget
	}
	target, err := findTypedDefinition(root, identity, receiver, name, loaded, originals, sources)
	if err != nil {
		return nil, err
	}
	if selectedPath != "" && !isInterface && target.path != selectedPath {
		return nil, errNoTypedReferenceTarget
	}
	return target, nil
}

func methodDeclaredOnReceiver(method *types.Func, receiver *types.TypeName, fset *token.FileSet) (bool, error) {
	signature, ok := method.Type().(*types.Signature)
	if !ok || signature.Recv() == nil {
		return false, nil
	}
	receiverType := types.Unalias(signature.Recv().Type())
	if pointer, isPointer := receiverType.(*types.Pointer); isPointer {
		receiverType = types.Unalias(pointer.Elem())
	}
	named, ok := receiverType.(*types.Named)
	if !ok {
		return false, nil
	}
	receiverIdentity, valid, err := referenceObjectIdentity(receiver, fset)
	if err != nil {
		return false, err
	}
	if !valid {
		return false, nil
	}
	methodReceiverIdentity, valid, err := referenceObjectIdentity(named.Origin().Obj(), fset)
	if err != nil {
		return false, err
	}
	return valid && receiverIdentity == methodReceiverIdentity, nil
}

func declarationReferenceTargets(root, selectedPath, name string, loaded []*packages.Package, originals map[string]bool, sources *referenceSourceSnapshot) (map[string]*referenceTarget, error) {
	targets := make(map[string]*referenceTarget)
	for _, pkg := range loaded {
		for _, file := range pkg.Syntax {
			filePath, err := packageSyntaxPath(pkg, file)
			if err != nil {
				return nil, err
			}
			if !originals[filePath] || (selectedPath != "" && filePath != selectedPath) {
				continue
			}
			if err := collectDeclarationTargetsInFile(root, pkg, file, name, sources, targets); err != nil {
				return nil, err
			}
		}
	}
	return targets, nil
}

func collectDeclarationTargetsInFile(root string, pkg *packages.Package, file *ast.File, name string, sources *referenceSourceSnapshot, targets map[string]*referenceTarget) error {
	for _, declaration := range file.Decls {
		if err := collectDeclarationTarget(root, pkg, declaration, name, sources, targets); err != nil {
			return err
		}
	}
	return nil
}

func collectDeclarationTarget(root string, pkg *packages.Package, declaration ast.Decl, name string, sources *referenceSourceSnapshot, targets map[string]*referenceTarget) error {
	switch declaration := declaration.(type) {
	case *ast.FuncDecl:
		if declaration.Name.Name != name {
			return nil
		}
		return addTypedReferenceTarget(root, targets, pkg, declaration.Name, pkg.TypesInfo.Defs[declaration.Name], extractReceiver(declaration.Recv), sources)
	case *ast.GenDecl:
		return addDeclarationSpecs(root, targets, pkg, declaration, name, sources)
	default:
		return nil
	}
}

func addDeclarationSpecs(root string, targets map[string]*referenceTarget, pkg *packages.Package, declaration *ast.GenDecl, name string, sources *referenceSourceSnapshot) error {
	for _, raw := range declaration.Specs {
		switch spec := raw.(type) {
		case *ast.TypeSpec:
			if spec.Name.Name == name {
				if err := addTypedReferenceTarget(root, targets, pkg, spec.Name, pkg.TypesInfo.Defs[spec.Name], "", sources); err != nil {
					return err
				}
			}
		case *ast.ValueSpec:
			for _, ident := range spec.Names {
				if ident.Name == name {
					if err := addTypedReferenceTarget(root, targets, pkg, ident, pkg.TypesInfo.Defs[ident], "", sources); err != nil {
						return err
					}
				}
			}
		}
	}
	return nil
}

func localReferenceTargets(root, selectedPath, name string, loaded []*packages.Package, originals map[string]bool, sources *referenceSourceSnapshot) (map[string]*referenceTarget, error) {
	targets := make(map[string]*referenceTarget)
	for _, pkg := range loaded {
		for _, file := range pkg.Syntax {
			filePath, err := packageSyntaxPath(pkg, file)
			if err != nil {
				return nil, err
			}
			if !originals[filePath] || (selectedPath != "" && filePath != selectedPath) {
				continue
			}
			for ident, object := range pkg.TypesInfo.Defs {
				if ident.Name != name {
					continue
				}
				kind := localReferenceBindingKind(file, ident)
				if kind == "" {
					continue
				}
				target, err := typedReferenceTarget(root, pkg, ident, object, "", sources)
				if errors.Is(err, errNoTypedReferenceTarget) {
					continue
				}
				if err != nil {
					return nil, err
				}
				target.candidate.Kind = kind
				targets[target.identity] = target
			}
		}
	}
	return targets, nil
}
