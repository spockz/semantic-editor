// Package astedit provides AST-level code transformation routines including declaration insertion, body replacement, and visibility management.
package astedit

import (
	"context"
	"errors"
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

var errInvalidSiblingGoSyntax = errors.New("invalid sibling Go syntax")

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

	resolvedPkg, err := resolveScaffoldPackageName(cleanPath, packageName)
	if err != nil {
		return "", err
	}
	src, err := formatScaffoldSource(ctx, resolvedPkg, opts.PurposeHeader)
	if err != nil {
		return "", err
	}

	if err := os.MkdirAll(filepath.Dir(cleanPath), 0o750); err != nil {
		return "", fmt.Errorf("create parent directory: %w", err)
	}

	if err := pipeline.WriteAtomic(cleanPath, []byte(src)); err != nil {
		return "", fmt.Errorf("write atomic: %w", err)
	}

	return resolvedPkg, nil
}

func resolveScaffoldPackageName(path, requested string) (string, error) {
	name := strings.TrimSpace(requested)
	if name == "" || name == "infer" {
		inferred, err := inferScaffoldPackageName(path)
		if err != nil {
			return "", err
		}
		return inferred, nil
	}
	if err := validateScaffoldPackageName(name); err != nil {
		return "", err
	}
	return name, nil
}

func inferScaffoldPackageName(path string) (string, error) {
	dir := filepath.Dir(path)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", fmt.Errorf("%w: read sibling directory %s: %w", ErrInferNoSiblings, dir, err)
	}
	baseName := filepath.Base(path)
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") || entry.Name() == baseName {
			continue
		}
		result := parseSiblingPackage(filepath.Join(dir, entry.Name()))
		if errors.Is(result.err, errInvalidSiblingGoSyntax) {
			continue
		}
		if result.err != nil {
			return "", result.err
		}
		if result.name != "" {
			return result.name, nil
		}
	}
	return "", fmt.Errorf("%w: in %s", ErrInferNoSiblings, dir)
}

type siblingPackageResult struct {
	name string
	err  error
}

func parseSiblingPackage(path string) siblingPackageResult {
	// #nosec G304 -- the path comes from an enumerated entry under the requested target directory; read errors propagate.
	source, err := os.ReadFile(path)
	if err != nil {
		return siblingPackageResult{err: fmt.Errorf("read sibling Go file %q: %w", path, err)}
	}
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, path, source, parser.PackageClauseOnly)
	if err != nil {
		return siblingPackageResult{err: fmt.Errorf("%w: %w", errInvalidSiblingGoSyntax, err)}
	}
	if node.Name == nil {
		return siblingPackageResult{}
	}
	return siblingPackageResult{name: node.Name.Name}
}

func validateScaffoldPackageName(name string) error {
	fset := token.NewFileSet()
	if _, err := parser.ParseFile(fset, "dummy.go", "package "+name+"\n", parser.PackageClauseOnly); err != nil {
		return fmt.Errorf("%w: invalid package name %q: %w", ErrSyntax, name, err)
	}
	return nil
}

func formatScaffoldSource(ctx context.Context, packageName, purposeHeader string) (string, error) {
	source := fmt.Sprintf("package %s\n", packageName)
	if purposeHeader != "" {
		return scaffoldSourceWithPurpose(packageName, purposeHeader)
	}
	finishFormatting := telemetry.Start(ctx, telemetry.PhaseFormattingAST)
	formatted, err := format.Source([]byte(source))
	finishFormatting()
	if err != nil {
		return "", fmt.Errorf("format scaffold: %w", err)
	}
	return string(formatted), nil
}

func scaffoldSourceWithPurpose(packageName, purposeHeader string) (string, error) {
	source := purposeHeader
	if !strings.HasSuffix(source, "\n") {
		source += "\n"
	}
	source += fmt.Sprintf("\npackage %s\n", packageName)
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, "scaffold.go", source, parser.ParseComments)
	if err != nil {
		return "", fmt.Errorf("%w: invalid purpose header: %w", ErrSyntax, err)
	}
	if len(node.Comments) == 0 || node.Name == nil || node.Name.Name != packageName || len(node.Decls) != 0 {
		return "", fmt.Errorf("%w: purpose header must contain only Go comments", ErrSyntax)
	}
	return source, nil
}
