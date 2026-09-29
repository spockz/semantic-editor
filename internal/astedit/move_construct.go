// Package astedit provides zero-token declaration construct relocation.
package astedit

import (
	"bytes"
	"context"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"semedit/internal/pipeline"
	"semedit/internal/symbol"
)

// MoveConstructOptions configures declaration relocation.
type MoveConstructOptions struct {
	Placement           Placement
	TargetSymbol        string
	DestinationFile     string
	AutoOrganizeImports bool
}

// MoveConstruct relocates an existing declaration construct to a new placement within filePath.
func MoveConstruct(ctx context.Context, filePath, symbolQuery string, opts MoveConstructOptions) (string, error) {
	cleanPath := filepath.Clean(filePath)
	destinationPath := cleanPath
	if opts.DestinationFile != "" {
		destinationPath = filepath.Clean(opts.DestinationFile)
	}
	if destinationPath != cleanPath {
		return moveConstructTo(ctx, cleanPath, destinationPath, symbolQuery, opts)
	}
	content, err := os.ReadFile(cleanPath)
	if err != nil {
		return "", fmt.Errorf("read target file: %w", err)
	}

	fset := token.NewFileSet()
	fileNode, err := parser.ParseFile(fset, cleanPath, content, parser.ParseComments|parser.AllErrors)
	if err != nil {
		return "", &SyntaxError{File: cleanPath, Pos: extractSyntaxPosition(fset, err), Cause: err, Err: ErrSyntax}
	}

	recv, name, _ := symbol.ParseIdentifier(symbolQuery)
	var targetDecl ast.Decl
	for _, decl := range fileNode.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok {
			if fn.Name.Name == name && extractReceiverName(fn.Recv) == recv {
				targetDecl = fn
				break
			}
		}
		if gen, ok := decl.(*ast.GenDecl); ok {
			matched := false
			for _, spec := range gen.Specs {
				switch s := spec.(type) {
				case *ast.TypeSpec:
					if s.Name.Name == symbolQuery {
						targetDecl = gen
						matched = true
						break
					}
				case *ast.ValueSpec:
					for _, ident := range s.Names {
						if ident.Name == symbolQuery {
							targetDecl = gen
							matched = true
							break
						}
					}
				}
				if matched {
					break
				}
			}
			if matched {
				break
			}
		}
	}

	if targetDecl == nil {
		return "", &symbol.SymbolError{Op: "move_construct", File: filePath, Symbol: symbolQuery, Err: symbol.ErrNotFound}
	}
	if group, ok := targetDecl.(*ast.GenDecl); ok && len(group.Specs) > 1 {
		return "", fmt.Errorf("cannot move one member of a grouped declaration; split the group first")
	}
	if opts.TargetSymbol == symbolQuery && (opts.Placement == PlacementBeforeSymbol || opts.Placement == PlacementAfterSymbol) {
		return "", nil
	}

	var startPos, endPos token.Pos
	switch d := targetDecl.(type) {
	case *ast.FuncDecl:
		if d.Doc != nil {
			startPos = d.Doc.Pos()
		} else {
			startPos = d.Pos()
		}
		endPos = d.End()
	case *ast.GenDecl:
		if d.Doc != nil {
			startPos = d.Doc.Pos()
		} else {
			startPos = d.Pos()
		}
		endPos = d.End()
	}

	startOffset := fset.Position(startPos).Offset
	endOffset := fset.Position(endPos).Offset
	declText := string(content[startOffset:endOffset])

	cutEnd := endOffset
	for cutEnd < len(content) && (content[cutEnd] == ' ' || content[cutEnd] == '\t' || content[cutEnd] == '\r') {
		cutEnd++
	}
	if cutEnd < len(content) && content[cutEnd] == '\n' {
		cutEnd++
	}

	remaining := make([]byte, 0, len(content)-(cutEnd-startOffset))
	remaining = append(remaining, content[:startOffset]...)
	remaining = append(remaining, content[cutEnd:]...)

	fset2 := token.NewFileSet()
	fileNode2, err := parser.ParseFile(fset2, cleanPath, remaining, parser.ParseComments|parser.AllErrors)
	if err != nil {
		return "", fmt.Errorf("parse file after cutting %s: %w", symbolQuery, err)
	}

	insertOffset, err := calculateMoveOffset(fset2, fileNode2, remaining, opts)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	buf.Write(remaining[:insertOffset])
	if insertOffset > 0 && !bytes.HasSuffix(buf.Bytes(), []byte("\n\n")) {
		if bytes.HasSuffix(buf.Bytes(), []byte("\n")) {
			buf.WriteString("\n")
		} else {
			buf.WriteString("\n\n")
		}
	}
	rest := remaining[insertOffset:]
	buf.WriteString(declText)
	if len(bytes.TrimLeft(rest, "\r\n")) == 0 {
		buf.WriteByte('\n')
	} else {
		buf.WriteString("\n\n")
	}
	buf.Write(bytes.TrimLeft(rest, "\r\n"))

	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		return "", fmt.Errorf("format moved construct: %w", err)
	}

	if _, err := parser.ParseFile(token.NewFileSet(), cleanPath, formatted, parser.ParseComments); err != nil {
		return "", fmt.Errorf("verify moved syntax: %w", err)
	}

	if err := verifyMoveFileUnchanged(cleanPath, content); err != nil {
		return "", err
	}
	if err := pipeline.WriteAtomic(cleanPath, formatted); err != nil {
		return "", fmt.Errorf("write atomic: %w", err)
	}
	if opts.AutoOrganizeImports {
		if err := pipeline.OrganizeImports(ctx, filepath.Dir(cleanPath), cleanPath); err != nil {
			_ = pipeline.Format(ctx, filepath.Dir(cleanPath), cleanPath)
			return "", fmt.Errorf("organize imports: %w", err)
		}
	} else if err := pipeline.Format(ctx, filepath.Dir(cleanPath), cleanPath); err != nil {
		return "", fmt.Errorf("format file: %w", err)
	}
	finalContent, err := os.ReadFile(cleanPath)
	if err != nil {
		return "", fmt.Errorf("read moved file after formatting: %w", err)
	}
	return UnifiedDiff(symbolQuery, string(content), string(finalContent)), nil
}

func calculateMoveOffset(fset *token.FileSet, fileNode *ast.File, content []byte, opts MoveConstructOptions) (int, error) {
	switch opts.Placement {
	case PlacementFileStart:
		return offsetAfterImports(fset, fileNode), nil

	case PlacementFileEnd, "":
		return len(content), nil

	case PlacementBeforeSymbol, PlacementAfterSymbol:
		return findSymbolBoundaryOffset(fset, fileNode, opts.TargetSymbol, opts.Placement)

	case PlacementPublicStart, PlacementPublicEnd, PlacementPrivateStart, PlacementPrivateEnd:
		return calculateInsertionOffset(fset, fileNode, content, Options{
			Placement:    opts.Placement,
			TargetSymbol: opts.TargetSymbol,
		})

	default:
		return 0, fmt.Errorf("unsupported placement %q for move_construct", opts.Placement)
	}
}

func findSymbolBoundaryOffset(fset *token.FileSet, fileNode *ast.File, targetSymbol string, placement Placement) (int, error) {
	if targetSymbol == "" {
		return 0, &PlacementError{
			Strategy: placement,
			Err:      ErrSymbolNotFound,
		}
	}
	targetRecv, targetName, _ := symbol.ParseIdentifier(targetSymbol)
	for _, decl := range fileNode.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok {
			if fn.Name.Name == targetName && extractReceiverName(fn.Recv) == targetRecv {
				if placement == PlacementBeforeSymbol {
					if fn.Doc != nil {
						return fset.Position(fn.Doc.Pos()).Offset, nil
					}
					return fset.Position(fn.Pos()).Offset, nil
				}
				return fset.Position(fn.End()).Offset, nil
			}
		}
		if gen, ok := decl.(*ast.GenDecl); ok {
			for _, spec := range gen.Specs {
				switch s := spec.(type) {
				case *ast.TypeSpec:
					if s.Name.Name == targetSymbol {
						if placement == PlacementBeforeSymbol {
							if gen.Doc != nil {
								return fset.Position(gen.Doc.Pos()).Offset, nil
							}
							return fset.Position(gen.Pos()).Offset, nil
						}
						return fset.Position(gen.End()).Offset, nil
					}
				case *ast.ValueSpec:
					for _, ident := range s.Names {
						if ident.Name == targetSymbol {
							if placement == PlacementBeforeSymbol {
								if gen.Doc != nil {
									return fset.Position(gen.Doc.Pos()).Offset, nil
								}
								return fset.Position(gen.Pos()).Offset, nil
							}
							return fset.Position(gen.End()).Offset, nil
						}
					}
				}
			}
		}
	}
	return 0, fmt.Errorf("target symbol %q not found: %w", targetSymbol, ErrSymbolNotFound)
}

func moveConstructTo(ctx context.Context, sourcePath, destinationPath, symbolQuery string, opts MoveConstructOptions) (string, error) {
	sourcePath, destinationPath = filepath.Clean(sourcePath), filepath.Clean(destinationPath)
	if filepath.Dir(sourcePath) != filepath.Dir(destinationPath) {
		return moveConstructAcrossPackages(ctx, sourcePath, destinationPath, symbolQuery, opts)
	}
	source, err := os.ReadFile(sourcePath)
	if err != nil {
		return "", fmt.Errorf("read source file: %w", err)
	}
	// #nosec G304 -- destinationPath comes from a workspace-relative operation request.
	destination, err := os.ReadFile(destinationPath)
	if err != nil {
		return "", fmt.Errorf("read destination file: %w", err)
	}
	fset := token.NewFileSet()
	srcFile, err := parser.ParseFile(fset, sourcePath, source, parser.ParseComments|parser.AllErrors)
	if err != nil {
		return "", fmt.Errorf("parse source file: %w", err)
	}
	dstFset := token.NewFileSet()
	dstFile, err := parser.ParseFile(dstFset, destinationPath, destination, parser.ParseComments|parser.AllErrors)
	if err != nil {
		return "", fmt.Errorf("parse destination file: %w", err)
	}
	if srcFile.Name.Name != dstFile.Name.Name {
		return "", fmt.Errorf("source and destination must belong to the same Go package")
	}
	decl := findMoveDeclaration(srcFile, symbolQuery)
	if decl == nil {
		return "", &symbol.SymbolError{Op: "move_construct", File: sourcePath, Symbol: symbolQuery, Err: symbol.ErrNotFound}
	}
	if group, ok := decl.(*ast.GenDecl); ok && len(group.Specs) > 1 {
		return "", fmt.Errorf("cannot move one member of a grouped declaration; split the group first")
	}
	if declarationNameCollision(dstFile, decl) {
		return "", fmt.Errorf("destination already declares a name from %q", symbolQuery)
	}
	if moveBuildConstraint(srcFile)+moveFilenameBuildConstraint(sourcePath) != moveBuildConstraint(dstFile)+moveFilenameBuildConstraint(destinationPath) {
		return "", fmt.Errorf("source and destination have different Go build constraints")
	}
	start, end := moveDeclOffsets(fset, decl, srcFile, source)
	declText := string(source[start:end])
	remaining := append(append([]byte(nil), source[:start]...), source[end:]...)
	if len(remaining) > 0 && remaining[len(remaining)-1] != '\n' {
		remaining = append(remaining, '\n')
	}
	if _, err := parser.ParseFile(token.NewFileSet(), sourcePath, remaining, parser.ParseComments|parser.AllErrors); err != nil {
		return "", fmt.Errorf("validate source after move: %w", err)
	}
	dstOffset, err := calculateMoveOffset(dstFset, dstFile, destination, opts)
	if err != nil {
		return "", err
	}
	var destBuf bytes.Buffer
	destBuf.Write(destination[:dstOffset])
	if dstOffset > 0 && !bytes.HasSuffix(destBuf.Bytes(), []byte("\n\n")) {
		if bytes.HasSuffix(destBuf.Bytes(), []byte("\n")) {
			destBuf.WriteByte('\n')
		} else {
			destBuf.WriteString("\n\n")
		}
	}
	destinationRest := bytes.TrimLeft(destination[dstOffset:], "\r\n")
	destBuf.WriteString(declText)
	if len(destinationRest) == 0 {
		destBuf.WriteByte('\n')
	} else {
		destBuf.WriteString("\n\n")
	}
	destBuf.Write(destinationRest)
	formattedDestination, err := format.Source(destBuf.Bytes())
	if err != nil {
		return "", fmt.Errorf("format destination after move: %w", err)
	}
	destinationFset := token.NewFileSet()
	destinationAST, err := parser.ParseFile(destinationFset, destinationPath, formattedDestination, parser.ParseComments|parser.AllErrors)
	if err != nil {
		return "", fmt.Errorf("parse destination after declaration insertion: %w", err)
	}
	neededImports, err := moveImportsForDeclaration(ctx, filepath.Dir(sourcePath), decl, srcFile)
	if err != nil {
		return "", err
	}
	if err := mergeMoveImports(destinationAST, neededImports); err != nil {
		return "", err
	}
	var mergedDestination bytes.Buffer
	if err := format.Node(&mergedDestination, destinationFset, destinationAST); err != nil {
		return "", fmt.Errorf("format destination imports: %w", err)
	}
	formattedDestination = mergedDestination.Bytes()
	formattedSource, err := format.Source(remaining)
	if err != nil {
		return "", fmt.Errorf("format source after move: %w", err)
	}
	if _, err := parser.ParseFile(token.NewFileSet(), destinationPath, formattedDestination, parser.ParseComments|parser.AllErrors); err != nil {
		return "", fmt.Errorf("validate destination after move: %w", err)
	}
	if err := verifyMoveFileUnchanged(sourcePath, source); err != nil {
		return "", err
	}
	if err := verifyMoveFileUnchanged(destinationPath, destination); err != nil {
		return "", err
	}
	// Publish destination first. If the second per-file atomic write fails, the
	// error names the already-published destination so callers can recover.
	if err := pipeline.WriteAtomic(destinationPath, formattedDestination); err != nil {
		return "", fmt.Errorf("write destination atomically: %w", err)
	}
	if err := pipeline.WriteAtomic(sourcePath, formattedSource); err != nil {
		return "", fmt.Errorf("destination %s was written but source %s remains unchanged: %w", destinationPath, sourcePath, err)
	}
	if opts.AutoOrganizeImports {
		for _, path := range []string{sourcePath, destinationPath} {
			if err := pipeline.OrganizeImports(ctx, filepath.Dir(path), path); err != nil {
				return "", fmt.Errorf("organize imports in %s after move: %w", path, err)
			}
		}
	}
	finalSource, err := os.ReadFile(sourcePath)
	if err != nil {
		return "", fmt.Errorf("read source after move: %w", err)
	}
	// #nosec G304 -- destinationPath is the workspace-relative destination selected by the operation.
	finalDestination, err := os.ReadFile(destinationPath)
	if err != nil {
		return "", fmt.Errorf("read destination after move: %w", err)
	}
	return UnifiedDiff(symbolQuery, string(source), string(finalSource)) + "\n" + UnifiedDiff(symbolQuery, string(destination), string(finalDestination)), nil
}

func findMoveDeclaration(file *ast.File, query string) ast.Decl {
	recv, name, _ := symbol.ParseIdentifier(query)
	for _, decl := range file.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok && fn.Name.Name == name && extractReceiverName(fn.Recv) == recv {
			return fn
		}
		if gen, ok := decl.(*ast.GenDecl); ok {
			for _, spec := range gen.Specs {
				switch item := spec.(type) {
				case *ast.TypeSpec:
					if item.Name.Name == query {
						return gen
					}
				case *ast.ValueSpec:
					for _, ident := range item.Names {
						if ident.Name == query {
							return gen
						}
					}
				}
			}
		}
	}
	return nil
}

func moveDeclOffsets(fset *token.FileSet, decl ast.Decl, file *ast.File, source []byte) (int, int) {
	startPos := decl.Pos()
	endPos := decl.End()
	switch item := decl.(type) {
	case *ast.FuncDecl:
		if item.Doc != nil {
			startPos = item.Doc.Pos()
		}
	case *ast.GenDecl:
		if item.Doc != nil {
			startPos = item.Doc.Pos()
		}
	}
	start := fset.Position(startPos).Offset
	end := fset.Position(endPos).Offset
	endLine := fset.Position(endPos).Line
	for _, group := range file.Comments {
		groupStart := fset.Position(group.Pos())
		groupEnd := fset.Position(group.End()).Offset
		if groupStart.Offset < end || groupStart.Line != endLine {
			continue
		}
		if len(bytes.TrimSpace(source[end:groupStart.Offset])) == 0 {
			end = groupEnd
		}
	}
	return start, end
}

func declarationNameCollision(file *ast.File, moved ast.Decl) bool {
	names := map[string]bool{}
	switch decl := moved.(type) {
	case *ast.FuncDecl:
		names[decl.Name.Name] = true
	case *ast.GenDecl:
		for _, spec := range decl.Specs {
			switch item := spec.(type) {
			case *ast.TypeSpec:
				names[item.Name.Name] = true
			case *ast.ValueSpec:
				for _, name := range item.Names {
					names[name.Name] = true
				}
			}
		}
	}
	for _, decl := range file.Decls {
		switch item := decl.(type) {
		case *ast.FuncDecl:
			if names[item.Name.Name] {
				return true
			}
		case *ast.GenDecl:
			for _, spec := range item.Specs {
				switch s := spec.(type) {
				case *ast.TypeSpec:
					if names[s.Name.Name] {
						return true
					}
				case *ast.ValueSpec:
					for _, name := range s.Names {
						if names[name.Name] {
							return true
						}
					}
				}
			}
		}
	}
	return false
}

func moveImportsForDeclaration(ctx context.Context, workingDir string, decl ast.Decl, source *ast.File) (*ast.File, error) {
	used := map[string]bool{}
	ast.Inspect(decl, func(node ast.Node) bool {
		selector, ok := node.(*ast.SelectorExpr)
		if ok {
			if name, ok := selector.X.(*ast.Ident); ok {
				used[name.Name] = true
			}
		}
		return true
	})
	imports := &ast.File{}
	for _, spec := range source.Imports {
		pathValue, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			return nil, fmt.Errorf("decode source import: %w", err)
		}
		if spec.Name != nil && spec.Name.Name == "." {
			return nil, fmt.Errorf("cannot move declaration from a file with a dot import")
		}
		var local string
		if spec.Name != nil {
			local = spec.Name.Name
		} else {
			resolved, resolveErr := resolveMoveImportName(ctx, workingDir, pathValue)
			if resolveErr != nil {
				return nil, resolveErr
			}
			local = resolved
		}
		if used[local] {
			imports.Imports = append(imports.Imports, spec)
		}
	}
	return imports, nil
}

func moveFilenameBuildConstraint(path string) string {
	name := strings.TrimSuffix(filepath.Base(path), ".go")
	parts := strings.Split(name, "_")
	if len(parts) < 2 {
		return ""
	}
	known := map[string]bool{"aix": true, "android": true, "darwin": true, "dragonfly": true, "freebsd": true, "illumos": true, "ios": true, "js": true, "linux": true, "netbsd": true, "openbsd": true, "plan9": true, "solaris": true, "wasip1": true, "windows": true, "386": true, "amd64": true, "arm": true, "arm64": true, "loong64": true, "mips": true, "mipsle": true, "mips64": true, "mips64le": true, "ppc64": true, "ppc64le": true, "riscv64": true, "s390x": true, "wasm": true}
	if !known[parts[len(parts)-1]] {
		return ""
	}
	last := parts[len(parts)-1]
	if len(parts) > 2 && known[parts[len(parts)-2]] {
		return parts[len(parts)-2] + "_" + last
	}
	return last
}

func moveBuildConstraint(file *ast.File) string {
	var goBuild, plusBuild []string
	for _, group := range file.Comments {
		for _, comment := range group.List {
			text := strings.TrimSpace(comment.Text)
			if after, ok := strings.CutPrefix(text, "//go:build "); ok {
				goBuild = append(goBuild, after)
			}
			if after, ok := strings.CutPrefix(text, "// +build "); ok {
				plusBuild = append(plusBuild, after)
			}
		}
	}
	if len(goBuild) > 0 {
		return strings.Join(goBuild, "\n")
	}
	if len(plusBuild) > 0 {
		return strings.Join(plusBuild, "\n")
	}
	return ""
}

func resolveMoveImportName(ctx context.Context, workingDir, importPath string) (string, error) {
	// #nosec G204 -- executable is fixed to go; importPath is one parsed import literal.
	command := exec.CommandContext(ctx, "go", "list", "-f", "{{.Name}}", importPath)
	command.Dir = workingDir
	output, err := command.Output()
	if err != nil {
		return "", fmt.Errorf("cannot resolve package name for import %q; refusing move: %w", importPath, err)
	}
	name := strings.TrimSpace(string(output))
	if name == "" || !token.IsIdentifier(name) {
		return "", fmt.Errorf("go list returned an invalid package name %q for import %q; refusing move", name, importPath)
	}
	return name, nil
}

func verifyMoveFileUnchanged(path string, expected []byte) error {
	// #nosec G304 -- path is an operation-selected source or destination file.
	current, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("recheck %s before move publication: %w", path, err)
	}
	if !bytes.Equal(current, expected) {
		return fmt.Errorf("%s changed while move was being prepared; refusing to overwrite concurrent edits", path)
	}
	return nil
}

func mergeMoveImports(destination, source *ast.File) error {
	imports := make(map[string]string, len(destination.Imports))
	names := make(map[string]string, len(destination.Imports))
	for _, spec := range destination.Imports {
		pathValue, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			return fmt.Errorf("decode destination import: %w", err)
		}
		local := filepath.Base(pathValue)
		if spec.Name != nil {
			local = spec.Name.Name
		}
		imports[pathValue], names[local] = local, pathValue
	}
	var additions []ast.Spec
	for _, spec := range source.Imports {
		pathValue, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			return fmt.Errorf("decode source import: %w", err)
		}
		local := filepath.Base(pathValue)
		var alias *ast.Ident
		if spec.Name != nil {
			local = spec.Name.Name
			alias = &ast.Ident{Name: local}
		}
		if existing, exists := imports[pathValue]; exists {
			if existing != local {
				return fmt.Errorf("cannot move declaration because destination imports %q as %q, source uses %q", pathValue, existing, local)
			}
			continue
		}
		if existing, exists := names[local]; exists && existing != pathValue {
			return fmt.Errorf("cannot move declaration because import name %q is already used for %q", local, existing)
		}
		additions = append(additions, &ast.ImportSpec{Name: alias, Path: &ast.BasicLit{Kind: token.STRING, Value: strconv.Quote(pathValue)}})
		imports[pathValue], names[local] = local, pathValue
	}
	if len(additions) == 0 {
		return nil
	}
	for _, declaration := range destination.Decls {
		gen, ok := declaration.(*ast.GenDecl)
		if ok && gen.Tok == token.IMPORT {
			gen.Specs = append(gen.Specs, additions...)
			return nil
		}
	}
	destination.Decls = append([]ast.Decl{&ast.GenDecl{Tok: token.IMPORT, Lparen: token.Pos(1), Specs: additions}}, destination.Decls...)
	return nil
}
