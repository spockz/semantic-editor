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
	"semedit/internal/telemetry"
)

// ScaffoldOptions configures new file scaffolding behavior.
type ScaffoldOptions struct {
	Overwrite           bool
	AutoOrganizeImports bool // accepted for schema uniformity; no-op
	PurposeHeader       string
}

// ScaffoldFile creates a new Go source file initialized with a package clause.
func ScaffoldFile(ctx context.Context, filePath, packageName string, opts ScaffoldOptions) (string, error) {
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
	if opts.PurposeHeader != "" {
		src = opts.PurposeHeader
		if !strings.HasSuffix(src, "\n") {
			src += "\n"
		}
		src += fmt.Sprintf("\npackage %s\n", resolvedPkg)
		fset := token.NewFileSet()
		node, err := parser.ParseFile(fset, "scaffold.go", src, parser.ParseComments)
		if err != nil || len(node.Comments) == 0 || node.Name == nil || node.Name.Name != resolvedPkg || len(node.Decls) != 0 {
			if err != nil {
				return "", fmt.Errorf("%w: invalid purpose header: %w", ErrSyntax, err)
			}
			return "", fmt.Errorf("%w: purpose header must contain only Go comments", ErrSyntax)
		}
	} else {
		finishFormatting := telemetry.Start(ctx, telemetry.PhaseFormattingAST)
		formatted, err := format.Source([]byte(src))
		finishFormatting()
		if err != nil {
			return "", fmt.Errorf("format scaffold: %w", err)
		}
		src = string(formatted)
	}

	if err := os.MkdirAll(filepath.Dir(cleanPath), 0o750); err != nil {
		return "", fmt.Errorf("create parent directory: %w", err)
	}

	if err := pipeline.WriteAtomic(cleanPath, []byte(src)); err != nil {
		return "", fmt.Errorf("write atomic: %w", err)
	}

	return resolvedPkg, nil
}
