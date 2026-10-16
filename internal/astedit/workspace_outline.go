// Package astedit discovers package layouts, synopses, and module boundaries across a Go workspace.
package astedit

import (
	"errors"
	"fmt"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"golang.org/x/mod/modfile"

	"github.com/spockz/semantic-editor/internal/backend"
	"github.com/spockz/semantic-editor/internal/sourcefiles"
)

// WorkspaceOutlineGo discovers modules and packages under target path within root.
func WorkspaceOutlineGo(root, path string, includeTests bool) (*backend.WorkspaceOutlineResult, error) {
	absRoot, targetDir, err := resolveOutlineTarget(root, path)
	if err != nil {
		return nil, err
	}

	relScope, err := filepath.Rel(absRoot, targetDir)
	if err != nil {
		relScope = targetDir
	}
	displayScope := filepath.ToSlash(relScope)
	if displayScope == "" {
		displayScope = "."
	}

	result := &backend.WorkspaceOutlineResult{
		Scope: backend.ReadScope{
			Kind:     "workspace",
			Path:     displayScope,
			Complete: true,
		},
		Modules:  make([]backend.WorkspaceModule, 0),
		Packages: make([]backend.WorkspacePackage, 0),
	}

	modules, err := discoverScopedGoModules(absRoot, targetDir)
	if err != nil {
		return nil, err
	}
	result.Modules = modules

	packages, err := discoverGoPackages(absRoot, targetDir, includeTests)
	if err != nil {
		return nil, err
	}
	result.Packages = packages
	assignGoPackageModules(result.Packages, result.Modules)
	for i := range result.Modules {
		result.Modules[i].PackagesCount = countModulePackages(result.Packages, result.Modules[i].ModulePath)
		result.Modules[i].Language = "go"
		if result.Modules[i].Doc == nil {
			if doc := fallbackModuleDoc(result.Packages, result.Modules[i]); doc != nil {
				result.Modules[i].Doc = doc
				result.Modules[i].Synopsis = formatPackageSynopsis(*doc)
			}
		}
	}

	return result, nil
}

func discoverScopedGoModules(root, target string) ([]backend.WorkspaceModule, error) {
	rootScope := filepath.Clean(target) == filepath.Clean(root)
	modules := make(map[string]backend.WorkspaceModule)
	add := func(dir string, isRoot bool) error {
		dir = filepath.Clean(dir)
		if _, exists := modules[dir]; exists {
			return nil
		}
		manifest := filepath.Join(dir, "go.mod")
		data, err := fs.ReadFile(os.DirFS(dir), "go.mod")
		if err != nil {
			return fmt.Errorf("read Go module manifest %q: %w", manifest, err)
		}
		modPath, goVersion, doc, err := parseGoModDetails(manifest, data)
		if err != nil {
			return fmt.Errorf("parse Go module manifest %q: %w", manifest, err)
		}
		rel, err := filepath.Rel(root, dir)
		if err != nil {
			return fmt.Errorf("relativize Go module %q: %w", dir, err)
		}
		display := filepath.ToSlash(rel)
		if display == "" {
			display = "."
		}
		modules[dir] = backend.WorkspaceModule{Path: display, ModulePath: modPath, GoVersion: goVersion, IsRoot: isRoot, Language: "go", Doc: doc}
		if doc != nil {
			modules[dir] = withModuleSynopsis(modules[dir], *doc)
		}
		return nil
	}

	if rootScope {
		if err := walkGoModuleDirs(root, func(dir string) error { return add(dir, dir == root) }); err != nil {
			return nil, err
		}
		if err := addGoWorkModules(root, add); err != nil {
			return nil, err
		}
	} else {
		owner, err := nearestGoModule(root, target)
		if err != nil {
			return nil, err
		}
		if owner != "" {
			if err := add(owner, owner == root); err != nil {
				return nil, err
			}
		}
		if err := walkGoModuleDirs(target, func(dir string) error { return add(dir, dir == root) }); err != nil {
			return nil, err
		}
	}
	out := make([]backend.WorkspaceModule, 0, len(modules))
	for _, module := range modules {
		out = append(out, module)
	}
	slices.SortFunc(out, func(a, b backend.WorkspaceModule) int {
		if a.IsRoot != b.IsRoot {
			if a.IsRoot {
				return -1
			}
			return 1
		}
		return strings.Compare(a.Path, b.Path)
	})
	return out, nil
}

func addGoWorkModules(root string, add func(string, bool) error) error {
	workPath := filepath.Join(root, "go.work")
	data, err := fs.ReadFile(os.DirFS(root), "go.work")
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read Go workspace manifest %q: %w", workPath, err)
	}
	work, err := modfile.ParseWork(workPath, data, nil)
	if err != nil {
		return fmt.Errorf("parse Go workspace manifest %q: %w", workPath, err)
	}
	for _, use := range work.Use {
		dir := use.Path
		if !filepath.IsAbs(dir) {
			dir = filepath.Join(root, dir)
		}
		dir, err = filepath.Abs(dir)
		if err != nil {
			return fmt.Errorf("resolve Go workspace module %q: %w", use.Path, err)
		}
		dir, err = filepath.EvalSymlinks(dir)
		if err != nil {
			return fmt.Errorf("resolve Go workspace module symlinks %q: %w", use.Path, err)
		}
		if err := add(dir, false); err != nil {
			return fmt.Errorf("load go.work use %q: %w", use.Path, err)
		}
	}
	return nil
}

func walkGoModuleDirs(start string, visit func(string) error) error {
	err := filepath.WalkDir(start, func(current string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return fmt.Errorf("scan Go modules at %q: %w", current, walkErr)
		}
		if entry.IsDir() {
			if current != start && sourcefiles.IsIgnoredDirectoryPath(current) {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.Name() == "go.mod" {
			return visit(filepath.Dir(current))
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("discover Go modules: %w", err)
	}
	return nil
}

func nearestGoModule(root, target string) (string, error) {
	for dir := target; ; dir = filepath.Dir(dir) {
		if info, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil && !info.IsDir() {
			return dir, nil
		} else if err != nil && !os.IsNotExist(err) {
			return "", fmt.Errorf("inspect Go module manifest in %q: %w", dir, err)
		}
		if dir == root {
			break
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
	}
	return "", nil
}

func parseGoModDetails(name string, data []byte) (string, string, *string, error) {
	file, err := modfile.Parse(name, data, nil)
	if err != nil {
		return "", "", nil, err
	}
	modPath, goVersion := "", ""
	var doc *string
	if file.Module != nil {
		modPath = file.Module.Mod.Path
		if file.Module.Syntax != nil && len(file.Module.Syntax.Before) > 0 {
			lines := make([]string, 0, len(file.Module.Syntax.Before))
			for _, comment := range file.Module.Syntax.Before {
				lines = append(lines, strings.TrimSpace(strings.TrimPrefix(comment.Token, "//")))
			}
			text := strings.TrimSpace(strings.Join(lines, "\n"))
			if text != "" {
				doc = &text
			}
		}
	}
	if file.Go != nil {
		goVersion = file.Go.Version
	}
	if modPath == "" {
		return "", "", nil, fmt.Errorf("module directive is missing")
	}
	return modPath, goVersion, doc, nil
}

func withModuleSynopsis(module backend.WorkspaceModule, doc string) backend.WorkspaceModule {
	module.Synopsis = formatPackageSynopsis(doc)
	return module
}

func assignGoPackageModules(packages []backend.WorkspacePackage, modules []backend.WorkspaceModule) {
	for i := range packages {
		bestDepth := -1
		for _, module := range modules {
			if module.ModulePath == "" {
				continue
			}
			modPath := filepath.Clean(filepath.FromSlash(module.Path))
			pkgPath := filepath.Clean(filepath.FromSlash(packages[i].Path))
			if modPath == "." || pkgPath == modPath || strings.HasPrefix(pkgPath, modPath+string(filepath.Separator)) {
				depth := strings.Count(modPath, string(filepath.Separator))
				if depth > bestDepth {
					bestDepth = depth
					packages[i].Module = module.ModulePath
				}
			}
		}
		packages[i].Language = "go"
	}
}

func countModulePackages(packages []backend.WorkspacePackage, modulePath string) int {
	count := 0
	for _, pkg := range packages {
		if pkg.Module == modulePath {
			count++
		}
	}
	return count
}

func fallbackModuleDoc(packages []backend.WorkspacePackage, module backend.WorkspaceModule) *string {
	for _, pkg := range packages {
		if pkg.Path == module.Path && (pkg.Name == "main" || pkg.Name == path.Base(module.ModulePath)) {
			return pkg.Doc
		}
	}
	return nil
}

func discoverGoPackages(root, targetDir string, includeTests bool) ([]backend.WorkspacePackage, error) {
	packages := make([]backend.WorkspacePackage, 0)
	err := filepath.WalkDir(targetDir, func(current string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if !d.IsDir() {
			return nil
		}
		if current != targetDir && sourcefiles.IsIgnoredDirectoryPath(current) {
			return filepath.SkipDir
		}

		pkg, ok, inspectErr := inspectGoPackageDirectory(root, current, includeTests)
		if inspectErr != nil {
			return inspectErr
		}
		if ok {
			packages = append(packages, pkg)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("discover Go packages: %w", err)
	}

	slices.SortFunc(packages, func(a, b backend.WorkspacePackage) int {
		if a.Depth != b.Depth {
			return a.Depth - b.Depth
		}
		return strings.Compare(a.Path, b.Path)
	})
	return packages, nil
}

func inspectGoPackageDirectory(root, dir string, includeTests bool) (backend.WorkspacePackage, bool, error) {
	nonTestFiles, testFiles, docFilePath, err := collectPackageFiles(dir)
	if err != nil {
		return backend.WorkspacePackage{}, false, err
	}
	if !includeTests && len(nonTestFiles) == 0 {
		return backend.WorkspacePackage{}, false, nil
	}
	if len(nonTestFiles) == 0 && len(testFiles) == 0 {
		return backend.WorkspacePackage{}, false, nil
	}

	candidateFiles := make([]string, 0, len(nonTestFiles)+len(testFiles))
	candidateFiles = append(candidateFiles, nonTestFiles...)
	if includeTests {
		candidateFiles = append(candidateFiles, testFiles...)
	}

	packageName, docText := resolvePackageIdentity(docFilePath, candidateFiles)
	if packageName == "" {
		return backend.WorkspacePackage{}, false, nil
	}

	rel, err := filepath.Rel(root, dir)
	if err != nil {
		rel = dir
	}
	displayPath := filepath.ToSlash(rel)
	if displayPath == "" {
		displayPath = "."
	}

	depth := 0
	if displayPath != "." {
		depth = strings.Count(displayPath, "/") + 1
	}

	totalFiles := len(nonTestFiles)
	if includeTests {
		totalFiles += len(testFiles)
	}

	synopsis := formatPackageSynopsis(docText)
	var docPtr *string
	if docText != "" {
		docPtr = &docText
	}

	return backend.WorkspacePackage{
		Path:       displayPath,
		Name:       packageName,
		Synopsis:   synopsis,
		Doc:        docPtr,
		Depth:      depth,
		FilesCount: totalFiles,
	}, true, nil
}

func collectPackageFiles(dir string) (nonTest []string, test []string, docFile string, err error) {
	entries, readErr := os.ReadDir(dir)
	if readErr != nil {
		return nil, nil, "", fmt.Errorf("read directory %q: %w", dir, readErr)
	}
	for _, entry := range entries {
		if !entry.Type().IsRegular() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasPrefix(name, ".") {
			continue
		}
		full := filepath.Join(dir, name)
		if strings.HasSuffix(name, "_test.go") {
			test = append(test, full)
		} else {
			nonTest = append(nonTest, full)
			if name == "doc.go" {
				docFile = full
			}
		}
	}
	return nonTest, test, docFile, nil
}

func resolvePackageIdentity(docFilePath string, candidateFiles []string) (string, string) {
	packageName := ""
	docText := ""
	fset := token.NewFileSet()

	if docFilePath != "" {
		if parsed, parseErr := parser.ParseFile(fset, docFilePath, nil, parser.PackageClauseOnly|parser.ParseComments); parseErr == nil && parsed != nil {
			packageName = parsed.Name.Name
			if parsed.Doc != nil {
				docText = strings.TrimSpace(parsed.Doc.Text())
			}
		}
	}

	for _, filePath := range candidateFiles {
		if packageName != "" && docText != "" {
			break
		}
		if filePath == docFilePath {
			continue
		}
		parsed, parseErr := parser.ParseFile(fset, filePath, nil, parser.PackageClauseOnly|parser.ParseComments)
		if parseErr != nil || parsed == nil {
			continue
		}
		if packageName == "" {
			packageName = parsed.Name.Name
		}
		if docText == "" && parsed.Doc != nil {
			docText = strings.TrimSpace(parsed.Doc.Text())
		}
	}

	return packageName, docText
}

func formatPackageSynopsis(doc string) string {
	trimmed := strings.TrimSpace(doc)
	if trimmed == "" {
		return "(no package doc)"
	}
	lines := strings.Split(trimmed, "\n")
	var first strings.Builder
	first.WriteString(strings.TrimSpace(lines[0]))
	for i := 1; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			break
		}
		first.WriteString(" " + line)
		if strings.HasSuffix(line, ".") {
			break
		}
	}
	fields := strings.Fields(first.String())
	result := strings.Join(fields, " ")
	if len(result) > 160 {
		return result[:157] + "..."
	}
	return result
}

// LooksLikeGoWorkspace lets auto routing recognize Go module and go.work scopes.
func LooksLikeGoWorkspace(root, path string) (bool, error) {
	absRoot, target, err := resolveOutlineTarget(root, path)
	if err != nil {
		return false, err
	}
	moduleFound, err := hasEnclosingGoModule(absRoot, target)
	if err != nil {
		return false, err
	}
	if moduleFound {
		return true, nil
	}
	matched, err := matchesGoWorkUse(absRoot, target)
	if err != nil {
		return false, err
	}
	if matched {
		return true, nil
	}
	return directoryHasGoSource(target)
}

func resolveOutlineTarget(root, requested string) (string, string, error) {
	if root == "" {
		root = "."
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return "", "", fmt.Errorf("resolve Go workspace root %q: %w", root, err)
	}
	if canonicalRoot, evalErr := filepath.EvalSymlinks(absRoot); evalErr == nil {
		absRoot = canonicalRoot
	} else if !os.IsNotExist(evalErr) {
		return "", "", fmt.Errorf("resolve Go workspace root symlinks %q: %w", absRoot, evalErr)
	}
	target := absRoot
	if strings.TrimSpace(requested) != "" && requested != "." {
		if filepath.IsAbs(requested) {
			target = requested
		} else {
			target = filepath.Join(absRoot, requested)
		}
	}
	target = filepath.Clean(target)
	if canonicalTarget, evalErr := filepath.EvalSymlinks(target); evalErr == nil {
		target = canonicalTarget
	} else if !os.IsNotExist(evalErr) {
		return "", "", fmt.Errorf("resolve Go outline path symlinks %q: %w", target, evalErr)
	}
	rel, err := filepath.Rel(absRoot, target)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", "", fmt.Errorf("go outline path %q is outside workspace root %q", target, absRoot)
	}
	info, err := os.Stat(target)
	if err != nil {
		return "", "", fmt.Errorf("stat Go outline path %q: %w", target, err)
	}
	if !info.IsDir() {
		return "", "", fmt.Errorf("go outline path %q is not a directory", target)
	}
	return absRoot, target, nil
}

func matchesGoWorkUse(absRoot, target string) (bool, error) {
	workPath := filepath.Join(absRoot, "go.work")
	if data, readErr := fs.ReadFile(os.DirFS(absRoot), "go.work"); readErr == nil {
		work, parseErr := modfile.ParseWork(workPath, data, nil)
		if parseErr != nil {
			return false, fmt.Errorf("parse Go workspace manifest %q: %w", workPath, parseErr)
		}
		for _, use := range work.Use {
			moduleDir := use.Path
			if !filepath.IsAbs(moduleDir) {
				moduleDir = filepath.Join(absRoot, moduleDir)
			}
			moduleDir = filepath.Clean(moduleDir)
			moduleToTarget, relErr := filepath.Rel(moduleDir, target)
			if relErr == nil && moduleToTarget != ".." && !strings.HasPrefix(moduleToTarget, ".."+string(filepath.Separator)) {
				return true, nil
			}
			targetToModule, relErr := filepath.Rel(target, moduleDir)
			if relErr == nil && targetToModule != ".." && !strings.HasPrefix(targetToModule, ".."+string(filepath.Separator)) {
				return true, nil
			}
		}
	} else if !os.IsNotExist(readErr) {
		return false, fmt.Errorf("read Go workspace manifest %q: %w", workPath, readErr)
	}
	return false, nil
}

func directoryHasGoSource(target string) (bool, error) {
	found := false
	err := filepath.WalkDir(target, func(current string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return fmt.Errorf("scan Go source at %q: %w", current, walkErr)
		}
		if entry.IsDir() && current != target && sourcefiles.IsIgnoredDirectoryPath(current) {
			return filepath.SkipDir
		}
		if !entry.IsDir() && (filepath.Ext(entry.Name()) == ".go" || entry.Name() == "go.mod") {
			found = true
			return fs.SkipAll
		}
		return nil
	})
	if err != nil && !errors.Is(err, fs.SkipAll) {
		return false, fmt.Errorf("scan Go workspace %q: %w", target, err)
	}
	return found, nil
}

func hasEnclosingGoModule(root, target string) (bool, error) {
	for dir := target; ; dir = filepath.Dir(dir) {
		manifest := filepath.Join(dir, "go.mod")
		if info, err := os.Stat(manifest); err == nil && !info.IsDir() {
			return true, nil
		} else if err != nil && !os.IsNotExist(err) {
			return false, fmt.Errorf("inspect Go module manifest %q: %w", manifest, err)
		}
		if dir == root {
			return false, nil
		}
	}
}
