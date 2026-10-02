// This file binds Go package type analysis to the current source buffer and its complete read set.

package operation

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"bytes"
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"sync"

	"semedit/internal/gocache"
	"semedit/internal/sourcebuf"

	"golang.org/x/tools/go/packages"
)

type structuralSnapshot struct {
	Root     string
	File     string
	Sources  map[string][]byte
	Packages []*packages.Package
	Buffer   *sourcebuf.Buffer
	Ctx      context.Context
	owned    bool
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
	captured := make(map[string][]byte)
	var mu sync.Mutex
	parse := func(fset *token.FileSet, filename string, src []byte) (*ast.File, error) {
		path, err := filepath.Abs(filename)
		if err != nil {
			return nil, fmt.Errorf("resolve package source %q: %w", filename, err)
		}
		content := bytes.Clone(src)
		if src == nil {
			content, err = sourcebuf.ReadFile(ctx, path)
			if err != nil {
				return nil, fmt.Errorf("read package source %q: %w", path, err)
			}
		}
		mu.Lock()
		if previous, ok := captured[path]; ok && !bytes.Equal(previous, content) {
			mu.Unlock()
			return nil, fmt.Errorf("package source %q changed during analysis", path)
		}
		captured[path] = bytes.Clone(content)
		mu.Unlock()
		file, err := parser.ParseFile(fset, path, content, parser.ParseComments|parser.AllErrors)
		if err != nil {
			return nil, fmt.Errorf("parse package source %q: %w", path, err)
		}
		return file, nil
	}
	overlay := sourcebuf.Overlay(ctx)
	if overlay == nil {
		overlay = make(map[string][]byte)
	}
	for path, content := range extra {
		overlay[path] = bytes.Clone(content)
	}
	for _, name := range []string{"go.mod", "go.sum"} {
		path := filepath.Join(root, name)
		content, err := sourcebuf.ReadFile(ctx, path)
		if errors.Is(err, os.ErrNotExist) && name == "go.sum" {
			continue
		}
		if err != nil {
			return nil, nil, fmt.Errorf("capture structural manifest %q before package loading: %w", path, err)
		}
		captured[path] = bytes.Clone(content)
	}
	environment, err := gocache.Environment(ctx, root)
	if err != nil {
		return nil, nil, fmt.Errorf("prepare structural Go package environment: %w", err)
	}
	mode := packages.NeedName | packages.NeedFiles | packages.NeedCompiledGoFiles | packages.NeedSyntax | packages.NeedTypes | packages.NeedTypesInfo | packages.NeedImports | packages.NeedDeps | packages.NeedModule
	loaded, err := packages.Load(&packages.Config{Context: ctx, Dir: root, Env: environment, Mode: mode, Tests: true, BuildFlags: []string{"-mod=readonly"}, Overlay: overlay, ParseFile: parse}, "./...")
	if err != nil {
		return nil, nil, fmt.Errorf("load structural package snapshot: %w", err)
	}
	if len(loaded) == 0 {
		return nil, nil, fmt.Errorf("load structural package snapshot: no packages in %q", root)
	}
	for _, pkg := range loaded {
		for _, files := range [][]string{pkg.GoFiles, pkg.CompiledGoFiles} {
			for _, filename := range files {
				path, err := filepath.Abs(filename)
				if err != nil {
					return nil, nil, fmt.Errorf("resolve package input %q: %w", filename, err)
				}
				relative, err := filepath.Rel(root, path)
				if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
					continue
				}
				content, err := sourcebuf.ReadFile(ctx, path)
				if err != nil {
					return nil, nil, fmt.Errorf("capture package input %q: %w", path, err)
				}
				mu.Lock()
				previous, analyzed := captured[path]
				candidate, replaced := extra[path]
				if replaced {
					content = bytes.Clone(candidate)
					if analyzed && !bytes.Equal(previous, candidate) {
						mu.Unlock()
						return nil, nil, fmt.Errorf("package input %q differs from its simulated output", path)
					}
				} else if analyzed && !bytes.Equal(previous, content) {
					mu.Unlock()
					return nil, nil, fmt.Errorf("package input %q differs from its analyzed snapshot", path)
				}
				captured[path] = bytes.Clone(content)
				mu.Unlock()
			}
		}
	}
	return loaded, captured, nil
}

func prepareStructuralSnapshot(ctx context.Context, cc CallContext, file string) (*structuralSnapshot, error) {
	selected := resolveWorkPath(cc.WorkDir, file)
	absolute, err := filepath.Abs(selected)
	if err != nil {
		return nil, fmt.Errorf("resolve structural target %q: %w", file, err)
	}
	root, err := structuralModuleRoot(absolute)
	if err != nil {
		return nil, err
	}
	buffer := sourcebuf.FromContext(ctx)
	owned := false
	if buffer == nil {
		buffer, err = sourcebuf.New(cc.WorkDir)
		if err != nil {
			return nil, fmt.Errorf("bind structural workspace: %w", err)
		}
		ctx = sourcebuf.WithBuffer(ctx, buffer)
		owned = true
	}
	if err := buffer.AuthorizeWrites(ctx, root); err != nil {
		return nil, fmt.Errorf("authorize structural module: %w", err)
	}
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
	return &structuralSnapshot{Root: root, File: absolute, Sources: sources, Packages: loaded, Buffer: buffer, Ctx: ctx, owned: owned}, nil
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
