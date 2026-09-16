// Package astedit provides AST-level code transformation routines including declaration insertion, body replacement, and visibility management.
package astedit

import (
	"context"
	"fmt"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"semedit/internal/pipeline"
)

// ScaffoldOptions configures new file scaffolding behavior.
type ScaffoldOptions struct {
	Overwrite           bool
	AutoOrganizeImports bool // accepted for schema uniformity; no-op
}

// ScaffoldFile creates a new Go source file initialized with a package clause.
func ScaffoldFile(_ context.Context, filePath, packageName string, opts ScaffoldOptions) (string, error) {
	cleanPath := filepath.Clean(filePath)
	if cleanPath == "" || cleanPath == "." {
		return "", fmt.Errorf("invalid file path: %q", filePath)
	}

	if _, err := os.Stat(cleanPath); err == nil && !opts.Overwrite {
		return "", fmt.Errorf("%w: %s", ErrFileExists, cleanPath)
	}

	resolvedPkg := strings.TrimSpace(packageName)
	if resolvedPkg == "" || resolvedPkg == "infer" {
		dir := filepath.Dir(cleanPath)
		entries, err := os.ReadDir(dir)
		if err != nil {
			return "", fmt.Errorf("%w: %s", ErrInferNoSiblings, dir)
		}

		baseName := filepath.Base(cleanPath)
		var inferred string
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			name := entry.Name()
			if strings.HasSuffix(name, ".go") && !strings.HasSuffix(name, "_test.go") && name != baseName {
				sibPath := filepath.Join(dir, name)
				fset := token.NewFileSet()
				node, err := parser.ParseFile(fset, sibPath, nil, parser.PackageClauseOnly)
				if err == nil && node.Name != nil && node.Name.Name != "" {
					inferred = node.Name.Name
					break
				}
			}
		}

		if inferred == "" {
			return "", fmt.Errorf("%w: in %s", ErrInferNoSiblings, dir)
		}
		resolvedPkg = inferred
	} else {
		fset := token.NewFileSet()
		if _, err := parser.ParseFile(fset, "dummy.go", "package "+resolvedPkg+"\n", parser.PackageClauseOnly); err != nil {
			return "", fmt.Errorf("%w: invalid package name %q", ErrSyntax, resolvedPkg)
		}
	}

	src := fmt.Sprintf("package %s\n", resolvedPkg)
	formatted, err := format.Source([]byte(src))
	if err != nil {
		return "", fmt.Errorf("format scaffold: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(cleanPath), 0o750); err != nil {
		return "", fmt.Errorf("create parent directory: %w", err)
	}

	if err := pipeline.WriteAtomic(cleanPath, formatted); err != nil {
		return "", fmt.Errorf("write atomic: %w", err)
	}

	return resolvedPkg, nil
}
