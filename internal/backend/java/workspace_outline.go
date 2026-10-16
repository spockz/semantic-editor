// Package java implements Java language-server operations and source transformations.
// WHY: This file describes Maven reactors and Java packages without starting a compiler or server.
package java

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/spockz/semantic-editor/internal/backend"
	"github.com/spockz/semantic-editor/internal/sourcefiles"
)

type javaWorkspacePOM struct {
	GroupID     string `xml:"groupId"`
	ArtifactID  string `xml:"artifactId"`
	Version     string `xml:"version"`
	Packaging   string `xml:"packaging"`
	Name        string `xml:"name"`
	Description string `xml:"description"`
	Parent      struct {
		GroupID string `xml:"groupId"`
		Version string `xml:"version"`
	} `xml:"parent"`
	Modules []string `xml:"modules>module"`
}

// LooksLikeJavaWorkspace reports whether the selected root contains a Java build marker or source.
func LooksLikeJavaWorkspace(root, path string) (bool, error) {
	root, target, err := javaOutlinePaths(root, path)
	if err != nil {
		return false, err
	}
	explicitScope := path != "" && path != "."
	scanRoot := target
	if !explicitScope {
		scanRoot = root
	}
	found := false
	err = filepath.WalkDir(scanRoot, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() && path != scanRoot && sourcefiles.IsIgnoredDirectoryPath(path) {
			return filepath.SkipDir
		}
		if entry.IsDir() {
			return nil
		}
		name := strings.ToLower(entry.Name())
		if name == "pom.xml" || name == "build.gradle" || name == "build.gradle.kts" {
			found = true
			return fs.SkipAll
		}
		if strings.EqualFold(filepath.Ext(entry.Name()), ".java") && (explicitScope || filepath.Dir(path) == scanRoot) {
			found = true
			return fs.SkipAll
		}
		return nil
	})
	if err != nil && !errorsIsSkipAll(err) {
		return false, fmt.Errorf("scan Java workspace %q: %w", scanRoot, err)
	}
	return found, nil
}

// WorkspaceOutline returns Maven modules and Java packages beneath the selected workspace path.
func WorkspaceOutline(root, path string, includeTests bool) (*backend.WorkspaceOutlineResult, error) {
	root, target, err := javaOutlinePaths(root, path)
	if err != nil {
		return nil, err
	}
	relScope, err := filepath.Rel(root, target)
	if err != nil || relScope == ".." || strings.HasPrefix(relScope, ".."+string(filepath.Separator)) {
		return nil, fmt.Errorf("java outline path %q is outside workspace root %q", target, root)
	}
	if relScope == "" {
		relScope = "."
	}
	result := &backend.WorkspaceOutlineResult{
		Scope:    backend.ReadScope{Kind: "workspace", Path: filepath.ToSlash(relScope), Complete: true},
		Modules:  make([]backend.WorkspaceModule, 0),
		Packages: make([]backend.WorkspacePackage, 0),
	}
	modules, err := discoverJavaModules(root, target)
	if err != nil {
		return nil, err
	}
	result.Modules = scopedJavaModules(modules, filepath.ToSlash(relScope))
	packages, err := discoverJavaPackages(root, target, includeTests)
	if err != nil {
		return nil, err
	}
	result.Packages = packages
	for i := range result.Modules {
		result.Modules[i].Language = string(backend.LanguageJava)
	}
	for i := range result.Packages {
		pkg := &result.Packages[i]
		pkg.Language = string(backend.LanguageJava)
		moduleIndex := -1
		modulePathLength := -1
		for j := range result.Modules {
			modulePath := result.Modules[j].Path
			if modulePath == "." || pkg.Path == modulePath || strings.HasPrefix(pkg.Path, modulePath+"/") {
				if len(modulePath) > modulePathLength {
					moduleIndex = j
					modulePathLength = len(modulePath)
				}
			}
		}
		if moduleIndex >= 0 {
			module := result.Modules[moduleIndex]
			pkg.Module = module.Path
			if module.GroupID != "" && module.ArtifactID != "" {
				pkg.Module = module.GroupID + ":" + module.ArtifactID
			}
			result.Modules[moduleIndex].PackagesCount++
		}
	}
	return result, nil
}

func scopedJavaModules(modules []backend.WorkspaceModule, scope string) []backend.WorkspaceModule {
	if scope == "." {
		return modules
	}
	nearest := ""
	for _, module := range modules {
		if module.Path == "." {
			nearest = "."
		}
	}
	for _, module := range modules {
		if module.Path != "." && (module.Path == scope || strings.HasPrefix(scope, module.Path+"/")) {
			if len(module.Path) > len(nearest) {
				nearest = module.Path
			}
		}
	}
	result := make([]backend.WorkspaceModule, 0, len(modules))
	for _, module := range modules {
		if module.Path == nearest || module.Path == scope || strings.HasPrefix(module.Path, scope+"/") {
			result = append(result, module)
		}
	}
	return result
}

func javaOutlinePaths(root, path string) (string, string, error) {
	if root == "" {
		root = "."
	}
	absoluteRoot, err := filepath.Abs(root)
	if err != nil {
		return "", "", fmt.Errorf("resolve Java workspace root %q: %w", root, err)
	}
	if canonical, evalErr := filepath.EvalSymlinks(absoluteRoot); evalErr == nil {
		absoluteRoot = canonical
	}
	target := absoluteRoot
	if strings.TrimSpace(path) != "" && path != "." {
		if filepath.IsAbs(path) {
			target = path
		} else {
			target = filepath.Join(absoluteRoot, path)
		}
	}
	target = filepath.Clean(target)
	if canonical, evalErr := filepath.EvalSymlinks(target); evalErr == nil {
		target = canonical
	}
	rel, err := filepath.Rel(absoluteRoot, target)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", "", fmt.Errorf("java outline path %q is outside workspace root %q", target, absoluteRoot)
	}
	info, err := os.Stat(target)
	if err != nil {
		return "", "", fmt.Errorf("stat Java outline path %q: %w", target, err)
	}
	if !info.IsDir() {
		return "", "", fmt.Errorf("java outline path %q is not a directory", target)
	}
	return absoluteRoot, target, nil
}

func javaSourceFiles(target, root string) ([]string, error) {
	files := make([]string, 0)
	err := filepath.WalkDir(target, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if path != target && sourcefiles.IsIgnoredDirectoryPath(path) {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.EqualFold(filepath.Ext(entry.Name()), ".java") {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return fmt.Errorf("resolve Java source path %q: %w", path, err)
		}
		if !filepath.IsLocal(rel) {
			return fmt.Errorf("java source path %q escapes workspace root %q", path, root)
		}
		files = append(files, path)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("scan Java source directory %q: %w", target, err)
	}
	sort.Strings(files)
	return files, nil
}

func discoverJavaModules(root, target string) ([]backend.WorkspaceModule, error) {
	discovery := javaModuleDiscovery{root: root, target: target, visited: make(map[string]bool)}
	var ancestors []string
	for dir := target; ; dir = filepath.Dir(dir) {
		ancestors = append(ancestors, dir)
		if dir == root {
			break
		}
	}
	for _, dir := range slices.Backward(ancestors) {

		pomPath := filepath.Join(dir, "pom.xml")
		if _, statErr := os.Stat(pomPath); statErr == nil {
			if err := discovery.visit(pomPath, "", ""); err != nil {
				return nil, err
			}
		} else if !os.IsNotExist(statErr) {
			return nil, fmt.Errorf("inspect Maven module POM %q: %w", pomPath, statErr)
		}
	}
	walkErr := filepath.WalkDir(target, func(current string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return fmt.Errorf("scan Maven modules at %q: %w", current, walkErr)
		}
		if entry.IsDir() && current != target && sourcefiles.IsIgnoredDirectoryPath(current) {
			return filepath.SkipDir
		}
		if !entry.IsDir() && entry.Name() == "pom.xml" {
			return discovery.visit(current, "", "")
		}
		return nil
	})
	if walkErr != nil {
		return nil, fmt.Errorf("scan Maven modules under %q: %w", target, walkErr)
	}
	sort.Slice(discovery.modules, func(i, j int) bool {
		if discovery.modules[i].Path == "." {
			return true
		}
		if discovery.modules[j].Path == "." {
			return false
		}
		return discovery.modules[i].Path < discovery.modules[j].Path
	})
	return discovery.modules, nil
}

type javaModuleDiscovery struct {
	root    string
	target  string
	modules []backend.WorkspaceModule
	visited map[string]bool
}

func (d *javaModuleDiscovery) visit(pomPath, inheritedGroup, inheritedVersion string) error {
	pomPath = filepath.Clean(pomPath)
	rel, err := filepath.Rel(d.root, filepath.Dir(pomPath))
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return fmt.Errorf("maven module POM %q is outside workspace root", pomPath)
	}
	if d.visited[pomPath] {
		return nil
	}
	d.visited[pomPath] = true
	data, err := os.ReadFile(pomPath)
	if err != nil {
		return fmt.Errorf("read Maven module POM %q: %w", pomPath, err)
	}
	var pom javaWorkspacePOM
	if err := xml.Unmarshal(data, &pom); err != nil {
		return fmt.Errorf("parse Maven module POM %q: %w", pomPath, err)
	}
	groupID := firstNonEmpty(pom.GroupID, pom.Parent.GroupID, inheritedGroup)
	version := firstNonEmpty(pom.Version, pom.Parent.Version, inheritedVersion)
	modulePath := filepath.ToSlash(rel)
	if modulePath == "." || pathWithin(d.target, filepath.Dir(pomPath)) || pathWithin(filepath.Dir(pomPath), d.target) {
		var doc *string
		if pom.Description != "" {
			description := pom.Description
			doc = &description
		}
		d.modules = append(d.modules, backend.WorkspaceModule{Path: modulePath, GroupID: groupID, ArtifactID: pom.ArtifactID, Version: firstNonEmpty(pom.Version, pom.Parent.Version, inheritedVersion), Packaging: firstNonEmpty(pom.Packaging, "jar"), IsRoot: modulePath == ".", Language: string(backend.LanguageJava), Synopsis: javaModuleSynopsis(pom.Description, pom.Name), Doc: doc})
	}
	for _, declared := range pom.Modules {
		if err := d.visitJavaDeclaredModule(pomPath, declared, groupID, version); err != nil {
			return err
		}
	}
	return nil
}

func (d *javaModuleDiscovery) visitJavaDeclaredModule(parentPOM, declared, groupID, version string) error {
	moduleDir := filepath.Clean(filepath.Join(filepath.Dir(parentPOM), declared))
	moduleRel, err := filepath.Rel(d.root, moduleDir)
	if err != nil || moduleRel == ".." || strings.HasPrefix(moduleRel, ".."+string(filepath.Separator)) {
		return fmt.Errorf("maven module path %q escapes workspace root", declared)
	}
	if !pathWithin(moduleDir, d.target) && !pathWithin(d.target, moduleDir) {
		return nil
	}
	return d.visit(filepath.Join(moduleDir, "pom.xml"), groupID, version)
}

func discoverJavaPackages(root, target string, includeTests bool) ([]backend.WorkspacePackage, error) {
	rootDir, openErr := os.OpenRoot(root)
	if openErr != nil {
		return nil, fmt.Errorf("open Java workspace root %q: %w", root, openErr)
	}
	discovery := javaPackageDiscovery{root: root, target: target, includeTests: includeTests, rootDir: rootDir, packages: make(map[string]*backend.WorkspacePackage)}
	walkErr := filepath.WalkDir(target, discovery.visit)
	closeErr := rootDir.Close()
	if walkErr != nil {
		return nil, fmt.Errorf("scan Java packages under %q: %w", target, walkErr)
	}
	if closeErr != nil {
		return nil, fmt.Errorf("close Java workspace root %q: %w", root, closeErr)
	}
	result := make([]backend.WorkspacePackage, 0, len(discovery.packages))
	for _, pkg := range discovery.packages {
		result = append(result, *pkg)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Depth == result[j].Depth {
			return result[i].Path < result[j].Path
		}
		return result[i].Depth < result[j].Depth
	})
	return result, nil
}

type javaPackageDiscovery struct {
	root         string
	target       string
	includeTests bool
	rootDir      *os.Root
	packages     map[string]*backend.WorkspacePackage
}

func (d *javaPackageDiscovery) visit(path string, entry fs.DirEntry, walkErr error) error {
	if walkErr != nil {
		return walkErr
	}
	if entry.IsDir() {
		return d.visitDirectory(path)
	}
	if !strings.EqualFold(filepath.Ext(entry.Name()), ".java") {
		return nil
	}
	rel, err := filepath.Rel(d.root, path)
	if err != nil {
		return fmt.Errorf("resolve Java source path %q: %w", path, err)
	}
	if javaTestSource(rel) && !d.includeTests {
		return nil
	}
	data, err := d.readSource(path, rel)
	if err != nil {
		return err
	}
	d.addPackage(entry.Name(), rel, data)
	return nil
}

func (d *javaPackageDiscovery) visitDirectory(path string) error {
	if path != d.target && sourcefiles.IsIgnoredDirectoryPath(path) {
		return filepath.SkipDir
	}
	return nil
}

func javaTestSource(relative string) bool {
	path := filepath.ToSlash(relative)
	return strings.Contains(path, "/src/test/java/") || strings.HasPrefix(path, "src/test/java/")
}

func (d *javaPackageDiscovery) readSource(path, relative string) ([]byte, error) {
	file, err := d.rootDir.Open(filepath.ToSlash(relative))
	if err != nil {
		return nil, fmt.Errorf("read Java source %q: %w", path, err)
	}
	data, readErr := io.ReadAll(file)
	closeErr := file.Close()
	if readErr != nil || closeErr != nil {
		return nil, fmt.Errorf("read Java source %q: %w", path, errors.Join(readErr, closeErr))
	}
	return data, nil
}

func (d *javaPackageDiscovery) addPackage(fileName, relative string, data []byte) {
	packageName := javaPackageName(data)
	if packageName == "" {
		packageName = "(default)"
	}
	packagePath := javaPackageRelativePath(filepath.ToSlash(filepath.Dir(relative)))
	if existing := d.packages[packagePath]; existing != nil {
		existing.FilesCount++
		if fileName == "package-info.java" {
			existing.Doc, existing.Synopsis = nil, ""
			setJavaPackageDoc(existing, javaPackageInfoDoc(data))
		}
		if existing.Synopsis == "" {
			setJavaPackageDoc(existing, javaPrimaryTypeDoc(data))
		}
		return
	}
	depth := 0
	if packagePath != "." {
		depth = len(strings.Split(packagePath, "/"))
	}
	pkg := &backend.WorkspacePackage{Path: packagePath, Name: packageName, Depth: depth, FilesCount: 1}
	d.packages[packagePath] = pkg
	if fileName == "package-info.java" {
		setJavaPackageDoc(pkg, javaPackageInfoDoc(data))
		return
	}
	setJavaPackageDoc(pkg, javaPrimaryTypeDoc(data))
}

func javaPackageName(source []byte) string {
	for line := range strings.SplitSeq(string(source), "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "package ") {
			continue
		}
		name := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(line, "package "), ";"))
		return name
	}
	return ""
}

func javaPrimaryTypeDoc(source []byte) string {
	text := string(source)
	offset := 0
	inBlockComment := false
	for _, line := range strings.SplitAfter(text, "\n") {
		trimmed := strings.TrimSpace(line)
		if inBlockComment {
			if strings.Contains(trimmed, "*/") {
				inBlockComment = false
			}
			offset += len(line)
			continue
		}
		if strings.HasPrefix(trimmed, "/*") {
			inBlockComment = !strings.Contains(trimmed, "*/")
			offset += len(line)
			continue
		}
		if strings.HasPrefix(trimmed, "//") {
			offset += len(line)
			continue
		}
		if strings.Contains(trimmed, "public") {
			for _, kind := range []string{"class ", "interface ", "enum ", "record "} {
				if strings.Contains(trimmed, kind) {
					return javaDocComment(source, offset)
				}
			}
		}
		offset += len(line)
	}
	return ""
}

func javaPackageRelativePath(relativeDirectory string) string {
	for _, sourceRoot := range []string{"/src/main/java/", "/src/test/java/"} {
		if before, after, ok := strings.Cut(relativeDirectory, sourceRoot); ok {
			module := before
			packagePath := after
			if module == "" {
				return packagePath
			}
			if packagePath == "" {
				return module
			}
			return filepath.ToSlash(filepath.Join(module, packagePath))
		}
	}
	return relativeDirectory
}

func javaDocComment(source []byte, before int) string {
	text := string(source)
	if before < 0 || before > len(text) {
		before = len(text)
	}
	end := strings.LastIndex(text[:before], "*/")
	if end < 0 {
		return ""
	}
	start := strings.LastIndex(text[:end], "/**")
	if start < 0 || !javaDocAttachmentOnly(text[end+2:before]) {
		return ""
	}
	comment := text[start+3 : end]
	lines := strings.Split(comment, "\n")
	for i, line := range lines {
		line = strings.TrimSpace(line)
		line = strings.TrimPrefix(line, "*")
		lines[i] = strings.TrimSpace(line)
	}
	return strings.TrimSpace(strings.Join(lines, " "))
}

func javaPackageInfoDoc(source []byte) string {
	offset := 0
	for _, line := range strings.SplitAfter(string(source), "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "package ") {
			return javaDocComment(source, offset)
		}
		offset += len(line)
	}
	return ""
}

func javaDocAttachmentOnly(text string) bool {
	for line := range strings.SplitSeq(text, "\n") {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "@") {
			return false
		}
	}
	return true
}

func setJavaPackageDoc(pkg *backend.WorkspacePackage, doc string) {
	if pkg == nil || doc == "" || pkg.Doc != nil {
		return
	}
	pkg.Doc = &doc
	end := strings.IndexAny(doc, ".!?\n")
	if end >= 0 {
		pkg.Synopsis = strings.TrimSpace(doc[:end+1])
	} else {
		pkg.Synopsis = doc
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func javaModuleSynopsis(description, name string) string {
	return strings.Join(strings.Fields(firstNonEmpty(description, name)), " ")
}

func pathWithin(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func errorsIsSkipAll(err error) bool {
	return errors.Is(err, fs.SkipAll)
}
