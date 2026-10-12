// Package astedit provides AST-level code transformation routines including declaration insertion, body replacement, and visibility management.
package astedit

import (
	"context"
	"errors"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"semedit/internal/pipeline"
	"semedit/internal/sourcebuf"
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

	if sourcebuf.FromContext(ctx) != nil {
		if _, err := sourcebuf.ReadFile(ctx, cleanPath); err == nil && !opts.Overwrite {
			return "", fmt.Errorf("%w: %s", ErrFileExists, cleanPath)
		} else if err != nil && !errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("inspect buffered scaffold target: %w", err)
		}
	} else if _, err := os.Stat(cleanPath); err == nil && !opts.Overwrite {
		return "", fmt.Errorf("%w: %s", ErrFileExists, cleanPath)
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("inspect scaffold target: %w", err)
	}

	resolvedPkg, err := resolveScaffoldPackageName(cleanPath, packageName)
	if err != nil {
		return "", err
	}
	src, err := formatScaffoldSource(ctx, resolvedPkg, opts.PurposeHeader)
	if err != nil {
		return "", err
	}

	if sourcebuf.FromContext(ctx) == nil {
		if err := os.MkdirAll(filepath.Dir(cleanPath), 0o750); err != nil {
			return "", fmt.Errorf("create parent directory: %w", err)
		}
	}

	if err := pipeline.WriteAtomicContext(ctx, cleanPath, []byte(src)); err != nil {
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
		scaffold, err := scaffoldSourceWithPurpose(packageName, purposeHeader)
		if err != nil {
			return "", fmt.Errorf("invalid purpose header: %w", err)
		}
		return scaffold, nil
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
	purposeHeader = strings.TrimPrefix(purposeHeader, "\ufeff")
	purposeHeader = strings.TrimSpace(purposeHeader)
	if purposeHeader == "" {
		return fmt.Sprintf("package %s\n", packageName), nil
	}
	if strings.ContainsRune(purposeHeader, '\ufeff') {
		return "", fmt.Errorf("%w: purpose header contains an interior byte order mark", ErrSyntax)
	}
	if purposeHeaderHasDirective(purposeHeader) {
		return "", fmt.Errorf("%w: purpose header must not contain Go directives", ErrSyntax)
	}
	source := purposeHeader
	if !strings.HasSuffix(source, "\n") {
		source += "\n"
	}
	source += fmt.Sprintf("\npackage %s\n", packageName)
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, "scaffold.go", source, parser.ParseComments|parser.AllErrors)
	if err == nil && len(node.Comments) > 0 && node.Name != nil && node.Name.Name == packageName && len(node.Decls) == 0 {
		return source, nil
	}
	if purposeHeaderIsDeclaration(purposeHeader) {
		return "", fmt.Errorf("%w: purpose header must contain only Go comments", ErrSyntax)
	}
	lines := strings.Split(purposeHeader, "\n")
	for index, line := range lines {
		trimmed := strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(trimmed, "//"):
			lines[index] = trimmed
		case trimmed == "":
			lines[index] = "//"
		default:
			lines[index] = "// " + trimmed
		}
	}
	source = strings.Join(lines, "\n") + "\n\n" + fmt.Sprintf("package %s\n", packageName)
	formatted, err := format.Source([]byte(source))
	if err != nil {
		return "", fmt.Errorf("format scaffold: %w", err)
	}
	fset = token.NewFileSet()
	node, err = parser.ParseFile(fset, "scaffold.go", formatted, parser.ParseComments)
	if err != nil {
		return "", fmt.Errorf("%w: invalid wrapped purpose header: %w", ErrSyntax, err)
	}
	if len(node.Comments) == 0 || node.Name == nil || node.Name.Name != packageName || len(node.Decls) != 0 {
		return "", fmt.Errorf("%w: purpose header must contain only Go comments", ErrSyntax)
	}
	return string(formatted), nil
}

func purposeHeaderHasDirective(purposeHeader string) bool {
	for line := range strings.SplitSeq(purposeHeader, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "//go:") ||
			strings.HasPrefix(trimmed, "//line ") ||
			strings.HasPrefix(trimmed, "// +build") {
			return true
		}
	}
	return false
}

func purposeHeaderIsDeclaration(purposeHeader string) bool {
	source := strings.TrimSpace(purposeHeader)
	if source == "package" {
		return true
	}
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "purpose.go", source, parser.ParseComments|parser.AllErrors)
	if err == nil && file.Name != nil {
		return true
	}
	if hasPartialPurposeDeclaration(file, source, fset) {
		return true
	}
	synthetic := "package purpose\n\n" + source + "\n"
	fset = token.NewFileSet()
	file, err = parser.ParseFile(fset, "purpose.go", synthetic, parser.ParseComments|parser.AllErrors)
	if err == nil && len(file.Decls) > 0 {
		return true
	}
	return hasPartialPurposeDeclaration(file, synthetic, fset)
}

func hasPartialPurposeDeclaration(file *ast.File, source string, fset *token.FileSet) bool {
	if file == nil {
		return false
	}
	for _, declaration := range file.Decls {
		switch declaration := declaration.(type) {
		case *ast.GenDecl:
			if declaration.Tok == token.TYPE || declaration.Tok == token.CONST ||
				declaration.Tok == token.VAR || declaration.Tok == token.IMPORT {
				return true
			}
		case *ast.FuncDecl:
			if declaration.Type == nil || declaration.Type.Params == nil {
				continue
			}
			position := fset.Position(declaration.Type.Params.Opening)
			if position.Offset >= 0 && position.Offset < len(source) && source[position.Offset] == '(' {
				return true
			}
		}
	}
	return false
}
