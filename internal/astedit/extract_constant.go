// WHY: Applies proven constant extraction plans to an existing Go source overlay.

package astedit

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"sort"

	"github.com/spockz/semantic-editor/internal/sourcebuf"
	"github.com/spockz/semantic-editor/internal/sourcefiles"
	"github.com/spockz/semantic-editor/internal/symbol"

	"strings"

	"golang.org/x/tools/go/packages"
)

// ExtractConstantOptions selects exact compile-time expressions in one owner.
type ExtractConstantOptions struct {
	Discriminator     string
	OccurrenceIndices []int
	ExpectedRevision  string
}

// ExtractConstant plans and applies one package-level constant extraction.
func ExtractConstant(ctx context.Context, filePath, owner, name string, opts ExtractConstantOptions) (string, error) {
	if opts.Discriminator == "" {
		return "", fmt.Errorf("discriminator must contain the exact source text of one constant expression")
	}
	if len(opts.OccurrenceIndices) > 0 && opts.ExpectedRevision == "" {
		return "", fmt.Errorf("expected revision is required with occurrence indices")
	}
	cleanPath, err := filepath.Abs(filepath.Clean(filePath))
	if err != nil {
		return "", fmt.Errorf("resolve constant extraction target: %w", err)
	}
	ctx, ownedBuffer, standalone, err := prepareExtractionBuffer(ctx, cleanPath)
	if err != nil {
		return "", err
	}
	content, err := sourcebuf.ReadFile(ctx, cleanPath)
	if err != nil {
		return "", fmt.Errorf("read constant extraction target: %w", err)
	}
	if opts.ExpectedRevision != "" && goReadRevision(content) != opts.ExpectedRevision {
		return "", fmt.Errorf("selected source revision mismatch: expected %q, current %q", opts.ExpectedRevision, goReadRevision(content))
	}
	overlay := sourcebuf.Overlay(ctx, ".go")
	if overlay == nil {
		overlay = make(map[string][]byte)
	}
	overlay[cleanPath] = bytes.Clone(content)
	loaded, err := loadConstantPackage(ctx, filepath.Dir(cleanPath), overlay)
	if err != nil {
		return "", err
	}
	inputs, err := selectConstantPackageInputs(loaded, cleanPath)
	if err != nil {
		return "", err
	}
	receiver, ownerName, err := symbol.ParseIdentifier(owner)
	if err != nil {
		return "", fmt.Errorf("parse in_function selector: %w", err)
	}
	target := findExtractOwner(inputs.syntax, ownerName, receiver)
	if target == nil || target.Body == nil {
		return "", fmt.Errorf("constant extraction owner %q was not found or has no body", owner)
	}
	selected, err := selectConstantExpressions(inputs.fset, target.Body, content, opts.Discriminator, opts.OccurrenceIndices)
	if err != nil {
		return "", err
	}
	plan, err := PlanConstantExtraction(content, inputs.fset, inputs.syntax, inputs.info, inputs.scope, selected, name)
	if err != nil {
		return "", fmt.Errorf("plan constant extraction: %w", err)
	}
	updated, err := constantExtractionSource(cleanPath, content, plan)
	if err != nil {
		return "", err
	}
	diff, err := publishConstantExtraction(ctx, cleanPath, content, updated, plan, ownedBuffer, standalone)
	if err != nil {
		return "", err
	}
	return diff, nil
}

type constantPackageInputs struct {
	fset   *token.FileSet
	syntax *ast.File
	info   *types.Info
	scope  *types.Scope
	path   string
}

func selectConstantPackageInputs(loaded []*packages.Package, targetPath string) (constantPackageInputs, error) {
	var selected constantPackageInputs
	for _, pkg := range loaded {
		if pkg.Types == nil || pkg.TypesInfo == nil || pkg.Fset == nil {
			continue
		}
		for index, path := range pkg.CompiledGoFiles {
			absolute, err := filepath.Abs(path)
			if err != nil {
				return constantPackageInputs{}, fmt.Errorf("resolve constant extraction package path %q: %w", path, err)
			}
			if filepath.Clean(absolute) != targetPath || index >= len(pkg.Syntax) {
				continue
			}
			selected = constantPackageInputs{fset: pkg.Fset, syntax: pkg.Syntax[index], info: pkg.TypesInfo, scope: pkg.Types.Scope(), path: filepath.Clean(absolute)}
			break
		}
	}
	if selected.syntax == nil {
		return constantPackageInputs{}, fmt.Errorf("constant extraction target %q is not in a buildable Go package", targetPath)
	}
	return selected, nil
}

func selectConstantExpressions(fset *token.FileSet, body *ast.BlockStmt, source []byte, discriminator string, indices []int) ([]ast.Expr, error) {
	matches := make([]ast.Expr, 0)
	ast.Inspect(body, func(node ast.Node) bool {
		if _, ok := node.(*ast.FuncLit); ok {
			return false
		}
		expression, ok := node.(ast.Expr)
		if ok && matchesConstantSource(fset, source, discriminator, expression) {
			matches = append(matches, expression)
		}
		return true
	})
	sort.Slice(matches, func(i, j int) bool { return matches[i].Pos() < matches[j].Pos() })
	if len(indices) == 0 {
		if len(matches) != 1 {
			return nil, fmt.Errorf("constant expression selection matched %d current expressions; choose exact source text that identifies one expression or supply occurrence_indices", len(matches))
		}
		return matches, nil
	}
	selected := make([]ast.Expr, 0, len(indices))
	for _, index := range indices {
		if index < 0 || index >= len(matches) {
			return nil, fmt.Errorf("occurrence_indices entry %d is outside the %d exact matches", index, len(matches))
		}
		selected = append(selected, matches[index])
	}
	return selected, nil
}

func matchesConstantSource(fset *token.FileSet, source []byte, discriminator string, expression ast.Expr) bool {
	start, end := fset.Position(expression.Pos()).Offset, fset.Position(expression.End()).Offset
	return start >= 0 && end >= start && end <= len(source) && string(source[start:end]) == discriminator
}

func constantExtractionSource(path string, source []byte, plan ConstantExtractionPlan) ([]byte, error) {
	updated := make([]byte, 0, len(source)+len(plan.Declaration)+8)
	position := 0
	for _, occurrence := range plan.Occurrences {
		if occurrence.Start < position || occurrence.End <= occurrence.Start || occurrence.End > len(source) {
			return nil, fmt.Errorf("selected constant expression ranges are inconsistent")
		}
		updated = append(updated, source[position:occurrence.Start]...)
		updated = append(updated, plan.Replacement...)
		position = occurrence.End
	}
	updated = append(updated, source[position:]...)
	if len(updated) > 0 && updated[len(updated)-1] != '\n' {
		updated = append(updated, '\n')
	}
	updated = append(updated, '\n')
	updated = append(updated, []byte(plan.Declaration)...)
	updated = append(updated, '\n')
	if _, err := parser.ParseFile(token.NewFileSet(), path, updated, parser.ParseComments|parser.AllErrors); err != nil {
		return nil, &SyntaxError{File: path, Snippet: plan.Declaration, Cause: err, Err: ErrSyntax}
	}
	return updated, nil
}

func publishConstantExtraction(ctx context.Context, path string, before, after []byte, plan ConstantExtractionPlan, ownedBuffer *sourcebuf.Buffer, standalone bool) (string, error) {
	validated := sourcebuf.Overlay(ctx, ".go")
	if validated == nil {
		validated = make(map[string][]byte)
	}
	validated[path] = after
	loaded, err := loadConstantPackage(ctx, filepath.Dir(path), validated)
	if err != nil {
		return "", fmt.Errorf("validate extracted constant source: %w", err)
	}
	if err := validateExtractedConstantReferences(loaded, path, plan); err != nil {
		return "", fmt.Errorf("validate extracted constant references: %w", err)
	}
	return publishPreparedSource(ctx, sourcePublication{
		path:        path,
		diffPath:    filepath.Base(path),
		before:      before,
		after:       after,
		ownedBuffer: ownedBuffer,
		standalone:  standalone,
		operation:   "constant extraction",
	})
}

func validateExtractedConstantReferences(loaded []*packages.Package, path string, plan ConstantExtractionPlan) error {
	inputs, err := selectConstantPackageInputs(loaded, path)
	if err != nil {
		return err
	}
	constant, ok := inputs.scope.Lookup(plan.Replacement).(*types.Const)
	if !ok || !types.Identical(constant.Type(), plan.resolvedType) || constant.Val().ExactString() != plan.Value {
		return fmt.Errorf("generated package constant does not preserve the selected type and exact value")
	}
	expected := make(map[int]bool, len(plan.Occurrences))
	delta := 0
	for _, occurrence := range plan.Occurrences {
		start := occurrence.Start + delta
		expected[start] = false
		delta += len(plan.Replacement) - (occurrence.End - occurrence.Start)
	}
	for identifier, object := range inputs.info.Uses {
		position := inputs.fset.PositionFor(identifier.Pos(), false)
		if filepath.Clean(position.Filename) != inputs.path {
			continue
		}
		start := position.Offset
		if _, selected := expected[start]; !selected {
			continue
		}
		constantUse, ok := object.(*types.Const)
		typed, hasType := inputs.info.Types[identifier]
		if identifier.Name != plan.Replacement || !ok || constantUse != constant || !hasType || !types.Identical(typed.Type, plan.resolvedType) || typed.Value == nil || typed.Value.ExactString() != plan.Value {
			return fmt.Errorf("selected occurrence at byte %d does not resolve to the generated package constant", start)
		}
		expected[start] = true
	}
	missing := make([]int, 0)
	for start, found := range expected {
		if !found {
			missing = append(missing, start)
		}
	}
	if len(missing) > 0 {
		sort.Ints(missing)
		return fmt.Errorf("selected occurrence at byte %d was not resolved as the generated package constant", missing[0])
	}
	return nil
}

func loadConstantPackage(ctx context.Context, dir string, overlay map[string][]byte) ([]*packages.Package, error) {
	loaded, err := loadExtractionPackages(ctx, dir, overlay, extractionLoadOptions{tests: true, typeInfo: true})
	if err != nil {
		return nil, err
	}
	primary := make([]*packages.Package, 0, len(loaded))
	for _, pkg := range loaded {
		if !hasConstantTestFiles(pkg.CompiledGoFiles) {
			primary = append(primary, pkg)
		}
	}
	if len(primary) == 0 {
		return nil, fmt.Errorf("load Go package for constant extraction: no primary package found")
	}
	return primary, nil
}

func hasConstantTestFiles(files []string) bool {
	for _, path := range files {
		if strings.HasSuffix(filepath.Base(path), "_test.go") {
			return true
		}
	}
	return false
}

func bindExtractionPackageInputs(ctx context.Context, loaded []*packages.Package, bound map[string]bool) (bool, error) {
	inputs, err := collectExtractionPackageInputs(loaded)
	if err != nil {
		return false, err
	}
	return bindCollectedExtractionInputs(ctx, inputs, bound)
}

type extractionPackageInputs struct {
	roots       map[string]bool
	directories map[string]bool
	auxiliary   map[string]bool
	files       map[string]bool
	manifests   map[string]bool
}

func newExtractionPackageInputs() extractionPackageInputs {
	return extractionPackageInputs{
		roots: make(map[string]bool), directories: make(map[string]bool),
		auxiliary: make(map[string]bool), files: make(map[string]bool), manifests: make(map[string]bool),
	}
}

func collectExtractionPackageInputs(loaded []*packages.Package) (extractionPackageInputs, error) {
	inputs := newExtractionPackageInputs()
	var visitErr error
	packages.Visit(loaded, func(pkg *packages.Package) bool {
		if visitErr != nil {
			return false
		}
		if collectionErr := collectPackageInputRecord(pkg, &inputs); collectionErr != nil {
			visitErr = collectionErr
			return false
		}
		return true
	}, nil)
	if visitErr != nil {
		return extractionPackageInputs{}, visitErr
	}
	return inputs, nil
}

func collectPackageInputRecord(pkg *packages.Package, inputs *extractionPackageInputs) error {
	if pkg.Module == nil {
		return nil
	}
	module := pkg.Module
	if module.Replace != nil {
		module = module.Replace
	}
	if module.Version != "" || module.Dir == "" {
		return nil
	}
	if err := collectModuleInput(module, inputs); err != nil {
		return err
	}
	goFiles := append(append(append([]string(nil), pkg.GoFiles...), pkg.CompiledGoFiles...), pkg.OtherFiles...)
	if err := addExtractionInputPaths(goFiles, inputs.files, inputs.directories, "resolve analysis source path"); err != nil {
		return err
	}
	return addExtractionInputPaths(pkg.EmbedFiles, inputs.auxiliary, inputs.directories, "resolve embed input path")
}

func collectModuleInput(module *packages.Module, inputs *extractionPackageInputs) error {
	if module == nil {
		return nil
	}
	if module.Replace != nil {
		module = module.Replace
	}
	if module.Version != "" || module.Dir == "" {
		return nil
	}
	root, err := filepath.Abs(module.Dir)
	if err != nil {
		return fmt.Errorf("resolve analysis module directory: %w", err)
	}
	root = filepath.Clean(root)
	inputs.roots[root] = true
	for _, name := range []string{"go.mod", "go.sum"} {
		inputs.manifests[filepath.Join(root, name)] = true
	}
	return nil
}

func addExtractionInputPaths(paths []string, files, directories map[string]bool, operation string) error {
	for _, path := range paths {
		if path == "" {
			continue
		}
		absolute, err := filepath.Abs(path)
		if err != nil {
			return fmt.Errorf("%s %q: %w", operation, path, err)
		}
		absolute = filepath.Clean(absolute)
		files[absolute] = true
		directories[filepath.Dir(absolute)] = true
	}
	return nil
}

func bindCollectedExtractionInputs(ctx context.Context, inputs extractionPackageInputs, bound map[string]bool) (bool, error) {
	added, err := bindExtractionModuleRoots(ctx, inputs, bound)
	if err != nil {
		return false, err
	}
	directoryAdded, err := bindExtractionDirectories(ctx, inputs.directories, bound)
	if err != nil {
		return false, err
	}
	added = added || directoryAdded
	manifestAdded, err := bindExtractionManifests(ctx, inputs.manifests, bound)
	if err != nil {
		return false, err
	}
	added = added || manifestAdded
	if err := observeExtractionInputs(ctx, inputs.files, "compiled Go input"); err != nil {
		return false, err
	}
	if err := observeExtractionInputs(ctx, inputs.auxiliary, "embedded input"); err != nil {
		return false, err
	}
	return added, nil
}

func bindExtractionModuleRoots(ctx context.Context, inputs extractionPackageInputs, bound map[string]bool) (bool, error) {
	added := false
	for _, root := range sortedExtractionKeys(inputs.roots) {
		key := "module:" + root
		if bound[key] {
			continue
		}
		if err := rejectExtractionModuleSymlinks(root); err != nil {
			return false, err
		}
		if err := sourcebuf.ObserveTree(ctx, root, ".go"); err != nil {
			return false, fmt.Errorf("observe dependency module %q: %w", root, err)
		}
		bound[key] = true
		added = true
		for directory := range inputs.directories {
			if moveWithin(root, directory) {
				bound["dir:"+directory] = true
			}
		}
	}
	return added, nil
}

func bindExtractionDirectories(ctx context.Context, directories, bound map[string]bool) (bool, error) {
	added := false
	for _, directory := range sortedExtractionKeys(directories) {
		key := "dir:" + directory
		if bound[key] {
			continue
		}
		if err := rejectExtractionPackageSymlinks(directory); err != nil {
			return false, err
		}
		if err := sourcebuf.ObserveTree(ctx, directory, ".go"); err != nil {
			return false, fmt.Errorf("observe package source directory %q: %w", directory, err)
		}
		bound[key] = true
		added = true
	}
	return added, nil
}

func bindExtractionManifests(ctx context.Context, manifests, bound map[string]bool) (bool, error) {
	added := false
	for _, path := range sortedExtractionKeys(manifests) {
		key := "manifest:" + filepath.Clean(path)
		if bound[key] {
			continue
		}
		if _, err := sourcebuf.ReadFile(ctx, path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return false, fmt.Errorf("read module manifest %q: %w", path, err)
		}
		bound[key] = true
		added = true
	}
	return added, nil
}

func observeExtractionInputs(ctx context.Context, inputs map[string]bool, description string) error {
	for _, path := range sortedExtractionKeys(inputs) {
		if _, err := sourcebuf.ReadFile(ctx, path); err != nil {
			return fmt.Errorf("read %s %q: %w", description, path, err)
		}
	}
	return nil
}

func rejectExtractionPackageSymlinks(dir string) error {
	absolute, err := filepath.Abs(dir)
	if err != nil {
		return fmt.Errorf("resolve constant package source directory: %w", err)
	}
	resolved, err := filepath.EvalSymlinks(absolute)
	if err != nil {
		return fmt.Errorf("resolve constant package source directory %q: %w", absolute, err)
	}
	if resolved != absolute {
		return fmt.Errorf("extraction refuses symlinked package source directory %q", absolute)
	}
	entries, err := os.ReadDir(absolute)
	if err != nil {
		return fmt.Errorf("read constant package source directory %q: %w", absolute, err)
	}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".go" {
			continue
		}
		path := filepath.Join(absolute, entry.Name())
		info, err := os.Lstat(path)
		if err != nil {
			return fmt.Errorf("inspect constant package source %q: %w", path, err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("extraction refuses symlinked Go package source %q", path)
		}
	}
	return nil
}

func sortedExtractionKeys(values map[string]bool) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func rejectExtractionModuleSymlinks(root string) error {
	absolute, err := filepath.Abs(root)
	if err != nil {
		return fmt.Errorf("resolve dependency module source root: %w", err)
	}
	resolved, err := filepath.EvalSymlinks(absolute)
	if err != nil {
		return fmt.Errorf("resolve dependency module source root %q: %w", absolute, err)
	}
	if resolved != absolute {
		return fmt.Errorf("extraction refuses symlinked dependency module %q", absolute)
	}
	return filepath.WalkDir(absolute, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return fmt.Errorf("inspect dependency module source %q: %w", path, walkErr)
		}
		if entry.IsDir() {
			if path != absolute && (entry.Name() == ".git" || entry.Name() == ".scratch" || entry.Name() == "vendor" || entry.Name() == "node_modules" || sourcefiles.IsBenchmarkDataDirectory(path)) {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(entry.Name()) != ".go" || entry.Type()&os.ModeSymlink == 0 {
			return nil
		}
		return fmt.Errorf("extraction refuses symlinked Go package source %q", path)
	})
}
