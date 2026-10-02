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
	"maps"
	"os"
	"path/filepath"
	"sort"

	"semedit/internal/sourcebuf"
	"semedit/internal/symbol"

	"semedit/internal/gocache"

	"strings"

	"golang.org/x/tools/go/packages"
)

// ExtractConstantOptions selects one exact compile-time expression.
type ExtractConstantOptions struct {
	Discriminator    string
	ExpectedRevision string
}

// ExtractConstant plans and applies one package-level constant extraction.
func ExtractConstant(ctx context.Context, filePath, owner, name string, opts ExtractConstantOptions) (string, error) {
	if opts.Discriminator == "" {
		return "", fmt.Errorf("discriminator must contain the exact source text of one constant expression")
	}
	cleanPath, err := filepath.Abs(filepath.Clean(filePath))
	if err != nil {
		return "", fmt.Errorf("resolve constant extraction target: %w", err)
	}
	content, err := sourcebuf.ReadFile(ctx, cleanPath)
	if err != nil {
		return "", fmt.Errorf("read constant extraction target: %w", err)
	}
	if opts.ExpectedRevision != "" && goReadRevision(content) != opts.ExpectedRevision {
		return "", fmt.Errorf("selected source revision mismatch: expected %q, current %q", opts.ExpectedRevision, goReadRevision(content))
	}
	standalone := sourcebuf.FromContext(ctx) == nil
	var ownedBuffer *sourcebuf.Buffer
	if standalone {
		moduleRoot, rootErr := moveModuleRoot(cleanPath)
		if rootErr != nil {
			return "", fmt.Errorf("locate constant extraction module: %w", rootErr)
		}
		ownedBuffer, err = sourcebuf.New(filepath.Dir(cleanPath))
		if err != nil {
			return "", fmt.Errorf("capture constant extraction source package: %w", err)
		}
		ctx = sourcebuf.WithBuffer(ctx, ownedBuffer)
		if err := ownedBuffer.AuthorizeWrites(ctx, moduleRoot); err != nil {
			return "", fmt.Errorf("authorize constant extraction output: %w", err)
		}
		buffered, readErr := sourcebuf.ReadFile(ctx, cleanPath)
		if readErr != nil {
			return "", fmt.Errorf("read captured constant extraction target: %w", readErr)
		}
		if !bytes.Equal(buffered, content) {
			return "", fmt.Errorf("selected source changed while capturing constant extraction inputs")
		}
	}
	if err := sourcebuf.ObserveTree(ctx, filepath.Dir(cleanPath)); err != nil {
		return "", fmt.Errorf("capture package source set for constant extraction: %w", err)
	}
	overlay := sourcebuf.Overlay(ctx)
	if overlay == nil {
		overlay = make(map[string][]byte)
	}
	overlay[cleanPath] = bytes.Clone(content)
	loaded, err := loadConstantPackage(ctx, filepath.Dir(cleanPath), overlay)
	if err != nil {
		return "", err
	}
	var fset *token.FileSet
	var syntax *ast.File
	var info *types.Info
	var scope *types.Scope
	for _, pkg := range loaded {
		if pkg.Types == nil || pkg.TypesInfo == nil || pkg.Fset == nil {
			continue
		}
		for index, path := range pkg.CompiledGoFiles {
			absolute, absErr := filepath.Abs(path)
			if absErr != nil || filepath.Clean(absolute) != cleanPath {
				continue
			}
			fset, syntax, info, scope = pkg.Fset, pkg.Syntax[index], pkg.TypesInfo, pkg.Types.Scope()
			break
		}
	}
	if syntax == nil {
		return "", fmt.Errorf("constant extraction target %q is not in a buildable Go package", cleanPath)
	}
	receiver, ownerName, err := symbol.ParseIdentifier(owner)
	if err != nil {
		return "", fmt.Errorf("parse in_function selector: %w", err)
	}
	target := findExtractOwner(syntax, ownerName, receiver)
	if target == nil || target.Body == nil {
		return "", fmt.Errorf("constant extraction owner %q was not found or has no body", owner)
	}
	selected := make([]ast.Expr, 0, 1)
	ast.Inspect(target.Body, func(node ast.Node) bool {
		expr, ok := node.(ast.Expr)
		if !ok {
			return true
		}
		start, end := fset.Position(expr.Pos()).Offset, fset.Position(expr.End()).Offset
		if start >= 0 && end >= start && end <= len(content) && string(content[start:end]) == opts.Discriminator {
			selected = append(selected, expr)
		}
		return true
	})
	if len(selected) != 1 {
		return "", fmt.Errorf("constant expression selection matched %d current expressions; choose exact source text that identifies one expression", len(selected))
	}
	plan, err := PlanConstantExtraction(content, fset, syntax, info, scope, selected[0], name)
	if err != nil {
		return "", fmt.Errorf("plan constant extraction: %w", err)
	}
	updated := make([]byte, 0, len(content)+len(plan.Declaration)+8)
	updated = append(updated, content[:plan.Start]...)
	updated = append(updated, plan.Replacement...)
	updated = append(updated, content[plan.End:]...)
	if len(updated) > 0 && updated[len(updated)-1] != '\n' {
		updated = append(updated, '\n')
	}
	updated = append(updated, '\n')
	updated = append(updated, []byte(plan.Declaration)...)
	updated = append(updated, '\n')
	if _, err := parser.ParseFile(token.NewFileSet(), cleanPath, updated, parser.ParseComments|parser.AllErrors); err != nil {
		return "", &SyntaxError{File: cleanPath, Snippet: plan.Declaration, Cause: err, Err: ErrSyntax}
	}
	validated := sourcebuf.Overlay(ctx)
	if validated == nil {
		validated = make(map[string][]byte)
	}
	validated[cleanPath] = updated
	if _, err := loadConstantPackage(ctx, filepath.Dir(cleanPath), validated); err != nil {
		return "", fmt.Errorf("validate extracted constant source: %w", err)
	}
	buffer := sourcebuf.FromContext(ctx)
	if buffer == nil {
		return "", fmt.Errorf("constant extraction requires a source buffer before apply")
	}
	if err := buffer.Apply(ctx, []sourcebuf.Change{{Path: cleanPath, Before: content, After: updated}}); err != nil {
		return "", fmt.Errorf("apply buffered constant extraction: %w", err)
	}
	if standalone {
		if err := ownedBuffer.Flush(ctx); err != nil {
			return "", fmt.Errorf("publish constant extraction: %w", err)
		}
	}
	return UnifiedDiff(filepath.Base(cleanPath), string(content), string(updated)), nil
}

func loadConstantPackage(ctx context.Context, dir string, overlay map[string][]byte) ([]*packages.Package, error) {
	if err := rejectConstantPackageSymlinks(dir); err != nil {
		return nil, err
	}
	environment, err := gocache.Environment(ctx, dir)
	if err != nil {
		return nil, fmt.Errorf("prepare constant extraction Go environment: %w", err)
	}
	moduleRoot, err := moveModuleRoot(filepath.Join(dir, "constant.go"))
	if err != nil {
		return nil, fmt.Errorf("locate constant extraction module: %w", err)
	}
	buffer := sourcebuf.FromContext(ctx)
	if buffer != nil {
		if err := buffer.AuthorizeWrites(ctx, moduleRoot); err != nil {
			return nil, fmt.Errorf("bind constant extraction module: %w", err)
		}
	}
	bound := map[string]bool{"dir:" + filepath.Clean(dir): true, "module:" + filepath.Clean(moduleRoot): true}
	for {
		current := sourcebuf.Overlay(ctx)
		if current == nil {
			current = make(map[string][]byte)
		}
		maps.Copy(current, overlay)
		config := &packages.Config{
			Context: ctx, Dir: dir, Env: environment, BuildFlags: []string{"-mod=readonly"},
			Tests: true, Overlay: current,
			Mode: packages.NeedName | packages.NeedFiles | packages.NeedCompiledGoFiles | packages.NeedSyntax |
				packages.NeedTypes | packages.NeedTypesInfo | packages.NeedImports | packages.NeedDeps |
				packages.NeedModule | packages.NeedEmbedFiles,
		}
		loaded, err := packages.Load(config, ".")
		if err != nil {
			return nil, fmt.Errorf("load Go package for constant extraction: %w", err)
		}
		if len(loaded) == 0 {
			return nil, fmt.Errorf("load Go package for constant extraction: no package found")
		}
		for _, pkg := range loaded {
			for _, item := range pkg.Errors {
				return nil, fmt.Errorf("load Go package for constant extraction: %s", item.Error())
			}
		}
		added, err := bindConstantPackageInputs(ctx, loaded, bound)
		if err != nil {
			return nil, fmt.Errorf("capture constant extraction analysis inputs: %w", err)
		}
		if added {
			continue
		}
		if buffer != nil {
			if err := buffer.AssertDisk(); err != nil {
				return nil, fmt.Errorf("revalidate constant extraction analysis inputs: %w", err)
			}
		}
		primary := make([]*packages.Package, 0, 1)
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
}

func hasConstantTestFiles(files []string) bool {
	for _, path := range files {
		if strings.HasSuffix(filepath.Base(path), "_test.go") {
			return true
		}
	}
	return false
}

func bindConstantPackageInputs(ctx context.Context, loaded []*packages.Package, bound map[string]bool) (bool, error) {
	roots := make(map[string]bool)
	directories := make(map[string]bool)
	auxiliary := make(map[string]bool)
	files := make(map[string]bool)
	manifests := make(map[string]bool)
	var visitErr error
	packages.Visit(loaded, func(pkg *packages.Package) bool {
		module := pkg.Module
		if module != nil {
			if module.Replace != nil {
				module = module.Replace
			}
			if module.Dir != "" {
				root, err := filepath.Abs(module.Dir)
				if err != nil {
					visitErr = fmt.Errorf("resolve analysis module directory: %w", err)
					return false
				}
				root = filepath.Clean(root)
				roots[root] = true
				for _, name := range []string{"go.mod", "go.sum"} {
					manifests[filepath.Join(root, name)] = true
				}
			}
		}
		for _, path := range append(append(append([]string(nil), pkg.GoFiles...), pkg.CompiledGoFiles...), pkg.OtherFiles...) {
			if path == "" {
				continue
			}
			absolute, err := filepath.Abs(path)
			if err != nil {
				visitErr = fmt.Errorf("resolve analysis source path %q: %w", path, err)
				return false
			}
			absolute = filepath.Clean(absolute)
			files[absolute] = true
			directories[filepath.Dir(absolute)] = true
		}
		for _, path := range pkg.EmbedFiles {
			if path == "" {
				continue
			}
			absolute, err := filepath.Abs(path)
			if err != nil {
				visitErr = fmt.Errorf("resolve embed input path %q: %w", path, err)
				return false
			}
			absolute = filepath.Clean(absolute)
			auxiliary[absolute] = true
			directories[filepath.Dir(absolute)] = true
		}
		return true
	}, nil)
	if visitErr != nil {
		return false, visitErr
	}
	added := false
	for _, root := range sortedConstantKeys(roots) {
		key := "module:" + root
		if bound[key] {
			continue
		}
		if err := rejectConstantModuleSymlinks(root); err != nil {
			return false, err
		}
		if err := sourcebuf.ObserveTree(ctx, root); err != nil {
			return false, fmt.Errorf("observe dependency module %q: %w", root, err)
		}
		bound[key] = true
		added = true
		for directory := range directories {
			if moveWithin(root, directory) {
				bound["dir:"+directory] = true
			}
		}
	}
	for _, directory := range sortedConstantKeys(directories) {
		key := "dir:" + directory
		if bound[key] {
			continue
		}
		if err := rejectConstantPackageSymlinks(directory); err != nil {
			return false, err
		}
		if err := sourcebuf.ObserveTree(ctx, directory); err != nil {
			return false, fmt.Errorf("observe package source directory %q: %w", directory, err)
		}
		bound[key] = true
		added = true
	}
	for _, path := range sortedConstantKeys(manifests) {
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
	for _, path := range sortedConstantKeys(files) {
		if _, err := sourcebuf.ReadFile(ctx, path); err != nil {
			return false, fmt.Errorf("read compiled Go input %q: %w", path, err)
		}
	}
	for _, path := range sortedConstantKeys(auxiliary) {
		if _, err := sourcebuf.ReadFile(ctx, path); err != nil {
			return false, fmt.Errorf("read embedded input %q: %w", path, err)
		}
	}
	return added, nil
}

func rejectConstantPackageSymlinks(dir string) error {
	absolute, err := filepath.Abs(dir)
	if err != nil {
		return fmt.Errorf("resolve constant package source directory: %w", err)
	}
	resolved, err := filepath.EvalSymlinks(absolute)
	if err != nil {
		return fmt.Errorf("resolve constant package source directory %q: %w", absolute, err)
	}
	if resolved != absolute {
		return fmt.Errorf("constant extraction refuses symlinked package source directory %q", absolute)
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
			return fmt.Errorf("constant extraction refuses symlinked Go package source %q", path)
		}
	}
	return nil
}

func sortedConstantKeys(values map[string]bool) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func rejectConstantModuleSymlinks(root string) error {
	absolute, err := filepath.Abs(root)
	if err != nil {
		return fmt.Errorf("resolve dependency module source root: %w", err)
	}
	resolved, err := filepath.EvalSymlinks(absolute)
	if err != nil {
		return fmt.Errorf("resolve dependency module source root %q: %w", absolute, err)
	}
	if resolved != absolute {
		return fmt.Errorf("constant extraction refuses symlinked dependency module %q", absolute)
	}
	return filepath.WalkDir(absolute, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return fmt.Errorf("inspect dependency module source %q: %w", path, walkErr)
		}
		if entry.IsDir() {
			if path != absolute && (entry.Name() == ".git" || entry.Name() == ".scratch" || entry.Name() == "vendor" || entry.Name() == "node_modules") {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(entry.Name()) != ".go" || entry.Type()&os.ModeSymlink == 0 {
			return nil
		}
		return fmt.Errorf("constant extraction refuses symlinked dependency Go source %q", path)
	})
}
