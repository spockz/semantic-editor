// This file binds Go package type analysis to the current source buffer and its complete read set.

package operation

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"bytes"
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"sync"

	"github.com/spockz/semantic-editor/internal/gocache"
	"github.com/spockz/semantic-editor/internal/sourcebuf"

	"golang.org/x/tools/go/packages"
)

type structuralSnapshot struct {
	Root          string
	WorkspaceRoot string
	File          string
	Sources       map[string][]byte
	Packages      []*packages.Package
	Buffer        *sourcebuf.Buffer
	Ctx           context.Context
	owned         bool
}

func structuralModuleRoot(selected string) (string, error) {
	absolute, err := filepath.Abs(selected)
	if err != nil {
		return "", fmt.Errorf("resolve selected Go file: %w", err)
	}
	current := filepath.Dir(absolute)
	for {
		manifest := filepath.Join(current, "go.mod")
		info, err := os.Stat(manifest)
		if err == nil {
			if info.IsDir() || !info.Mode().IsRegular() {
				return "", fmt.Errorf("module manifest %q is not a regular file", manifest)
			}
			return current, nil
		}
		if !errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("inspect module manifest %q: %w", manifest, err)
		}
		parent := filepath.Dir(current)
		if parent == current {
			return "", fmt.Errorf("no Go module contains %q", selected)
		}
		current = parent
	}
}

func loadStructuralPackages(ctx context.Context, root string, extra map[string][]byte) ([]*packages.Package, map[string][]byte, error) {
	capture := &structuralPackageCapture{ctx: ctx, sources: make(map[string][]byte)}
	overlay := structuralPackageOverlay(ctx, extra)
	if err := captureManifests(capture, root); err != nil {
		return nil, nil, err
	}
	loaded, err := loadStructuralPackageGraph(capture, root, overlay)
	if err != nil {
		return nil, nil, err
	}
	if len(loaded) == 0 {
		return nil, nil, fmt.Errorf("load structural package snapshot: no packages in %q", root)
	}
	if err := captureLoadedStructuralInputs(capture, root, loaded, extra); err != nil {
		return nil, nil, err
	}
	return loaded, capture.sources, nil
}

type structuralPackageCapture struct {
	ctx     context.Context
	sources map[string][]byte
	mu      sync.Mutex
}

func (capture *structuralPackageCapture) parse(fset *token.FileSet, filename string, source []byte) (*ast.File, error) {
	path, err := filepath.Abs(filename)
	if err != nil {
		return nil, fmt.Errorf("resolve package source %q: %w", filename, err)
	}
	content := bytes.Clone(source)
	if source == nil {
		content, err = sourcebuf.ReadFile(capture.ctx, path)
		if err != nil {
			return nil, fmt.Errorf("read package source %q: %w", path, err)
		}
	}
	capture.mu.Lock()
	previous, exists := capture.sources[path]
	if exists && !bytes.Equal(previous, content) {
		capture.mu.Unlock()
		return nil, fmt.Errorf("package source %q changed during analysis", path)
	}
	capture.sources[path] = bytes.Clone(content)
	capture.mu.Unlock()
	file, err := parser.ParseFile(fset, path, content, parser.ParseComments|parser.AllErrors)
	if err != nil {
		return nil, fmt.Errorf("parse package source %q: %w", path, err)
	}
	return file, nil
}

func structuralPackageOverlay(ctx context.Context, extra map[string][]byte) map[string][]byte {
	overlay := sourcebuf.Overlay(ctx, ".go")
	if overlay == nil {
		overlay = make(map[string][]byte)
	}
	for path, content := range extra {
		overlay[path] = bytes.Clone(content)
	}
	return overlay
}

func captureManifests(capture *structuralPackageCapture, root string) error {
	for _, name := range []string{"go.mod", "go.sum"} {
		path := filepath.Join(root, name)
		content, err := sourcebuf.ReadFile(capture.ctx, path)
		if errors.Is(err, os.ErrNotExist) && name == "go.sum" {
			continue
		}
		if err != nil {
			return fmt.Errorf("capture structural manifest %q before package loading: %w", path, err)
		}
		capture.sources[path] = bytes.Clone(content)
	}
	return nil
}

func loadStructuralPackageGraph(capture *structuralPackageCapture, root string, overlay map[string][]byte) ([]*packages.Package, error) {
	environment, err := gocache.Environment(capture.ctx, root)
	if err != nil {
		return nil, fmt.Errorf("prepare structural Go package environment: %w", err)
	}
	mode := packages.NeedName | packages.NeedFiles | packages.NeedCompiledGoFiles | packages.NeedEmbedFiles | packages.NeedSyntax | packages.NeedTypes | packages.NeedTypesInfo | packages.NeedImports | packages.NeedDeps | packages.NeedModule
	loaded, err := packages.Load(&packages.Config{
		Context: capture.ctx, Dir: root, Env: environment, Mode: mode,
		Tests: true, BuildFlags: []string{"-mod=readonly"}, Overlay: overlay, ParseFile: capture.parse,
	}, "./...")
	if err != nil {
		return nil, fmt.Errorf("load structural package snapshot: %w", err)
	}
	return loaded, nil
}

func captureLoadedStructuralInputs(capture *structuralPackageCapture, root string, loaded []*packages.Package, extra map[string][]byte) error {
	root, err := filepath.Abs(root)
	if err != nil {
		return fmt.Errorf("resolve structural module root %q: %w", root, err)
	}
	moduleRoots := map[string]bool{filepath.Clean(root): true}
	var captureErr error
	packages.Visit(loaded, func(pkg *packages.Package) bool {
		if captureErr != nil {
			return false
		}
		moduleRoot, local, err := localStructuralModuleRoot(pkg)
		if err != nil {
			captureErr = err
			return false
		}
		if !local {
			return true
		}
		moduleRoots[moduleRoot] = true
		captureErr = captureStructuralPackageInputs(capture, moduleRoot, pkg, extra)
		return captureErr == nil
	}, nil)
	if captureErr != nil {
		return captureErr
	}
	return captureStructuralModuleManifests(capture, moduleRoots)
}

func captureStructuralModuleManifests(capture *structuralPackageCapture, roots map[string]bool) error {
	orderedRoots := make([]string, 0, len(roots))
	for root := range roots {
		orderedRoots = append(orderedRoots, root)
	}
	slices.Sort(orderedRoots)
	for _, root := range orderedRoots {
		for _, name := range []string{"go.mod", "go.sum"} {
			path := filepath.Join(root, name)
			content, err := sourcebuf.ReadFile(capture.ctx, path)
			if errors.Is(err, os.ErrNotExist) && name == "go.sum" {
				continue
			}
			if err != nil {
				return fmt.Errorf("capture local structural manifest %q: %w", path, err)
			}
			if previous, exists := capture.sources[path]; exists && !bytes.Equal(previous, content) {
				return fmt.Errorf("local structural manifest %q changed during analysis", path)
			}
			capture.sources[path] = bytes.Clone(content)
		}
	}
	return nil
}

func captureStructuralPackageInputs(capture *structuralPackageCapture, root string, pkg *packages.Package, extra map[string][]byte) error {
	for _, paths := range [][]string{pkg.GoFiles, pkg.CompiledGoFiles, pkg.OtherFiles, pkg.EmbedFiles} {
		for _, filename := range paths {
			if err := captureLoadedStructuralFile(capture, root, filename, extra); err != nil {
				return err
			}
		}
	}
	return nil
}

func localStructuralModuleRoot(pkg *packages.Package) (string, bool, error) {
	if pkg.Module == nil {
		return "", false, nil
	}
	module := pkg.Module
	if module.Replace != nil {
		module = module.Replace
	}
	if module.Dir == "" || module.Version != "" {
		return "", false, nil
	}
	root, err := filepath.Abs(module.Dir)
	if err != nil {
		return "", false, fmt.Errorf("resolve local module input root %q: %w", module.Dir, err)
	}
	return filepath.Clean(root), true, nil
}

func captureLoadedStructuralFile(capture *structuralPackageCapture, root, filename string, extra map[string][]byte) error {
	path, err := filepath.Abs(filename)
	if err != nil {
		return fmt.Errorf("resolve package input %q: %w", filename, err)
	}
	relative, err := filepath.Rel(root, path)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return nil
	}
	content, err := sourcebuf.ReadFile(capture.ctx, path)
	if err != nil {
		return fmt.Errorf("capture package input %q: %w", path, err)
	}
	capture.mu.Lock()
	defer capture.mu.Unlock()
	previous, analyzed := capture.sources[path]
	candidate, replaced := extra[path]
	if replaced {
		content = bytes.Clone(candidate)
		if analyzed && !bytes.Equal(previous, candidate) {
			return fmt.Errorf("package input %q differs from its simulated output", path)
		}
	} else if analyzed && !bytes.Equal(previous, content) {
		return fmt.Errorf("package input %q differs from its analyzed snapshot", path)
	}
	capture.sources[path] = bytes.Clone(content)
	return nil
}

func prepareStructuralSnapshot(ctx context.Context, cc CallContext, file string) (*structuralSnapshot, error) {
	workspaceRoot, err := filepath.Abs(cc.WorkDir)
	if err != nil {
		return nil, fmt.Errorf("resolve structural workspace root %q: %w", cc.WorkDir, err)
	}
	selected := resolveWorkPath(cc.WorkDir, file)
	absolute, err := filepath.Abs(selected)
	if err != nil {
		return nil, fmt.Errorf("resolve structural target %q: %w", file, err)
	}
	root, err := structuralModuleRoot(absolute)
	if err != nil {
		return nil, err
	}
	preparedCtx, buffer, owned, err := sourcebuf.Prepare(ctx, root)
	if err != nil {
		return nil, fmt.Errorf("prepare structural module buffer: %w", err)
	}
	ctx = preparedCtx
	selectedSource, err := sourcebuf.ReadFile(ctx, absolute)
	if err != nil {
		return nil, fmt.Errorf("read structural target %q: %w", file, err)
	}
	loaded, sources, err := loadStructuralPackages(ctx, root, nil)
	if err != nil {
		return nil, err
	}
	if analyzed, ok := sources[absolute]; !ok || !bytes.Equal(analyzed, selectedSource) {
		return nil, fmt.Errorf("structural target %q is not bound to the analyzed package snapshot", file)
	}
	return &structuralSnapshot{Root: root, WorkspaceRoot: workspaceRoot, File: absolute, Sources: sources, Packages: loaded, Buffer: buffer, Ctx: ctx, owned: owned}, nil
}

func (snapshot *structuralSnapshot) validateOutput(output []byte) error {
	loaded, _, err := loadStructuralPackages(snapshot.Ctx, snapshot.Root, map[string][]byte{snapshot.File: bytes.Clone(output)})
	if err != nil {
		return err
	}
	foundSelectedPackage := false
	for _, pkg := range loaded {
		contains, err := packageContainsStructuralFile(pkg, snapshot.File)
		if err != nil {
			return err
		}
		if !contains {
			continue
		}
		foundSelectedPackage = true
		if len(pkg.Errors) > 0 {
			return fmt.Errorf("structural output fails package type analysis in %s: %s", pkg.PkgPath, pkg.Errors[0].Msg)
		}
		if pkg.TypesInfo == nil || pkg.Fset == nil {
			return fmt.Errorf("structural output has no type information for package %s", pkg.PkgPath)
		}
	}
	if !foundSelectedPackage {
		return fmt.Errorf("structural output package does not contain selected file %q", snapshot.File)
	}
	return nil
}

func (snapshot *structuralSnapshot) applyOutput(output []byte) (bool, error) {
	before := snapshot.Sources[snapshot.File]
	current, err := sourcebuf.ReadFile(snapshot.Ctx, snapshot.File)
	if err != nil {
		return false, fmt.Errorf("revalidate selected structural source: %w", err)
	}
	if !bytes.Equal(current, before) {
		return false, fmt.Errorf("selected structural source changed after analysis")
	}
	if err := snapshot.Buffer.AssertDisk(); err != nil {
		return false, fmt.Errorf("revalidate structural read set: %w", err)
	}
	if bytes.Equal(before, output) {
		return false, nil
	}
	if err := snapshot.Buffer.Apply(snapshot.Ctx, []sourcebuf.Change{{Path: snapshot.File, Before: before, After: output}}); err != nil {
		return false, fmt.Errorf("apply structural output: %w", err)
	}
	if snapshot.owned {
		if err := snapshot.Buffer.Flush(snapshot.Ctx); err != nil {
			return false, fmt.Errorf("publish structural output: %w", err)
		}
	}
	return true, nil
}

func packageContainsStructuralFile(pkg *packages.Package, selected string) (bool, error) {
	for _, filename := range pkg.CompiledGoFiles {
		absolute, err := filepath.Abs(filename)
		if err != nil {
			return false, fmt.Errorf("resolve typed source %q: %w", filename, err)
		}
		if absolute == selected {
			return true, nil
		}
	}
	return false, nil
}
