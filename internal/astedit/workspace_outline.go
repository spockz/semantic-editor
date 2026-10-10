// Package astedit discovers package layouts, synopses, and module boundaries across a Go workspace.
package astedit

import (
	"fmt"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"golang.org/x/mod/modfile"

	"semedit/internal/backend"
	"semedit/internal/sourcefiles"
)

// WorkspaceOutlineGo discovers modules and packages under target path within root.
func WorkspaceOutlineGo(root, path string, includeTests bool) (*backend.WorkspaceOutlineResult, error) {
	if root == "" {
		root = "."
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve workspace root %q: %w", root, err)
	}
	canonicalRoot, err := filepath.EvalSymlinks(absRoot)
	if err == nil {
		absRoot = canonicalRoot
	}

	targetDir := absRoot
	if strings.TrimSpace(path) != "" && path != "." {
		if filepath.IsAbs(path) {
			targetDir = path
		} else {
			targetDir = filepath.Join(absRoot, path)
		}
	}
	targetDir = filepath.Clean(targetDir)
	canonicalTarget, err := filepath.EvalSymlinks(targetDir)
	if err == nil {
		targetDir = canonicalTarget
	}

	targetInfo, err := os.Stat(targetDir)
	if err != nil {
		return nil, fmt.Errorf("stat target directory %q: %w", targetDir, err)
	}
	if !targetInfo.IsDir() {
		return nil, fmt.Errorf("target path %q is not a directory", targetDir)
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

	moduleScanDir := absRoot
	if !strings.HasPrefix(targetDir, absRoot) {
		moduleScanDir = targetDir
	}
	modules, err := discoverGoModules(absRoot, moduleScanDir)
	if err != nil {
		return nil, err
	}
	result.Modules = modules

	packages, err := discoverGoPackages(absRoot, targetDir, includeTests)
	if err != nil {
		return nil, err
	}
	result.Packages = packages

	return result, nil
}

func discoverGoModules(root, scanDir string) ([]backend.WorkspaceModule, error) {
	modules := make([]backend.WorkspaceModule, 0)
	err := filepath.WalkDir(scanDir, func(current string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			if current != scanDir && sourcefiles.IsIgnoredDirectory(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if d.Name() != "go.mod" {
			return nil
		}
		modDir := filepath.Dir(current)
		rel, relErr := filepath.Rel(root, modDir)
		if relErr != nil {
			rel = modDir
		}
		displayPath := filepath.ToSlash(rel)
		if displayPath == "" {
			displayPath = "."
		}

		cleanPath := filepath.Clean(current)
		data, readErr := os.ReadFile(cleanPath) // #nosec G122,G304 -- WalkDir visits validated workspace paths.
		if readErr != nil {
			return fmt.Errorf("read Go module manifest %s: %w", cleanPath, readErr)
		}
		modPath, goVer := parseGoMod(cleanPath, data)
		modules = append(modules, backend.WorkspaceModule{
			Path:       displayPath,
			ModulePath: modPath,
			GoVersion:  goVer,
			IsRoot:     modDir == root,
		})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("discover Go modules: %w", err)
	}

	slices.SortFunc(modules, func(a, b backend.WorkspaceModule) int {
		if a.IsRoot != b.IsRoot {
			if a.IsRoot {
				return -1
			}
			return 1
		}
		return strings.Compare(a.Path, b.Path)
	})
	return modules, nil
}

func parseGoMod(path string, data []byte) (string, string) {
	file, err := modfile.Parse(path, data, nil)
	if err == nil {
		var modPath, goVer string
		if file.Module != nil {
			modPath = file.Module.Mod.Path
		}
		if file.Go != nil {
			goVer = file.Go.Version
		}
		if modPath != "" {
			return modPath, goVer
		}
	}

	var modPath, goVer string
	for line := range strings.SplitSeq(string(data), "\n") {
		trimmed := strings.TrimSpace(line)
		if after, ok := strings.CutPrefix(trimmed, "module "); ok {
			modPath = strings.TrimSpace(after)
		} else if after, ok := strings.CutPrefix(trimmed, "go "); ok {
			goVer = strings.TrimSpace(after)
		}
	}
	return modPath, goVer
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
		if current != targetDir && sourcefiles.IsIgnoredDirectory(d.Name()) {
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
