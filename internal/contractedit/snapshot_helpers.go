// Package contractedit owns execution-time source capture, validation, and publication for Go contract edits.
// WHY: This file keeps source parsing, module revision hashing, and package graph validation beside contract snapshots.
package contractedit

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"slices"
	"strings"

	"golang.org/x/tools/go/packages"

	"github.com/spockz/semantic-editor/internal/sourcebuf"
)

// ParseSource captures and parses one file from the current workspace view.
func ParseSource(ctx context.Context, root string, fset *token.FileSet, filename string, source []byte, captured map[string][]byte, onCapture func(string)) (*ast.File, error) {
	absolute, err := filepath.Abs(filename)
	if err != nil {
		return nil, fmt.Errorf("resolve parsed contract file %q: %w", filename, err)
	}
	canonical, err := filepath.EvalSymlinks(absolute)
	if err != nil {
		return nil, fmt.Errorf("canonicalize parsed contract file %q: %w", filename, err)
	}
	data := bytes.Clone(source)
	if source == nil {
		data, err = sourcebuf.ReadFile(ctx, canonical)
		if err != nil {
			return nil, fmt.Errorf("read parsed contract file %q: %w", canonical, err)
		}
	}
	if relative, relErr := filepath.Rel(root, canonical); relErr == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		captured[canonical] = bytes.Clone(data)
		if onCapture != nil {
			onCapture(canonical)
		}
	}
	file, err := parser.ParseFile(fset, filename, data, parser.ParseComments|parser.AllErrors)
	if err != nil {
		return nil, fmt.Errorf("parse current contract source %q: %w", filename, err)
	}
	return file, nil
}

// ValidatePackageGraph verifies that required packages have complete compiler facts.
func ValidatePackageGraph(packageSet map[string]*packages.Package, required map[string]bool) error {
	queue := make([]string, 0, len(required))
	for path := range required {
		queue = append(queue, path)
	}
	slices.Sort(queue)
	visited := make(map[string]bool, len(queue))
	for len(queue) > 0 {
		path := queue[0]
		queue = queue[1:]
		if visited[path] {
			continue
		}
		visited[path] = true
		pkg := packageSet[path]
		if pkg == nil {
			return fmt.Errorf("required package %q is absent from the current compiler graph", path)
		}
		if pkg.Types == nil || (required[path] && pkg.TypesInfo == nil) {
			return fmt.Errorf("required package %q has incomplete type information", path)
		}
		if len(pkg.Errors) > 0 {
			return fmt.Errorf("required package %q has type errors: %s", path, pkg.Errors[0].Msg)
		}
		imports := make([]string, 0, len(pkg.Imports))
		for imported := range pkg.Imports {
			imports = append(imports, imported)
		}
		slices.Sort(imports)
		for _, imported := range imports {
			if !visited[imported] {
				queue = append(queue, imported)
			}
		}
	}
	return nil
}

// Digest returns the SHA-256 revision identifier used by contract snapshots.
func Digest(data []byte) string {
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:])
}
