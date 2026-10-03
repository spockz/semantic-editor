// Package contractedit owns execution-time source capture, validation, and publication for Go contract edits.
// WHY: This file turns a typed contract plan into one formatted source candidate.
package contractedit

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"strconv"

	"golang.org/x/tools/go/packages"

	"semedit/internal/contractplan"
)

// ImportBindings resolves the explicit local names of imports in the target file.
func ImportBindings(pkg *packages.Package, file *ast.File) ([]contractplan.ImportBinding, error) {
	if pkg == nil || pkg.Types == nil || file == nil {
		return nil, fmt.Errorf("destination package and syntax are required")
	}
	bindings := make([]contractplan.ImportBinding, 0, len(file.Imports))
	for _, spec := range file.Imports {
		path, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			return nil, fmt.Errorf("decode destination import path: %w", err)
		}
		name := ""
		if spec.Name != nil {
			name = spec.Name.Name
		} else if imported := pkg.Imports[path]; imported != nil {
			name = imported.Name
		}
		if name == "" {
			return nil, fmt.Errorf("resolve destination import name for %q", path)
		}
		bindings = append(bindings, contractplan.ImportBinding{Path: path, Name: name})
	}
	return bindings, nil
}

// RenderCandidate applies one typed contract plan to the destination source bytes.
func RenderCandidate(source []byte, destination *ast.File, fset *token.FileSet, plan contractplan.Plan) ([]byte, error) {
	if destination == nil || fset == nil {
		return nil, fmt.Errorf("destination syntax and file set are required")
	}
	if plan.Declaration == "" {
		return bytes.Clone(source), nil
	}
	packageFile := fset.File(destination.Pos())
	if packageFile == nil {
		return nil, fmt.Errorf("destination package position has no source file")
	}
	packageEnd := packageFile.Offset(destination.Name.End())
	if packageEnd < 0 || packageEnd > len(source) {
		return nil, fmt.Errorf("destination package clause is outside its source bytes")
	}
	var output bytes.Buffer
	output.Grow(len(source) + len(plan.Declaration) + len(plan.Imports)*48 + 4)
	output.Write(source[:packageEnd])
	if len(plan.Imports) > 0 {
		output.WriteString("\n\nimport (\n")
		for _, imported := range plan.Imports {
			fmt.Fprintf(&output, "\t%s %s\n", imported.Name, strconv.Quote(imported.Path))
		}
		output.WriteString(")\n")
	}
	output.Write(source[packageEnd:])
	output.WriteString("\n\n")
	output.WriteString(plan.Declaration)
	output.WriteByte('\n')
	candidate, err := format.Source(output.Bytes())
	if err != nil {
		return nil, fmt.Errorf("format generated contract destination: %w", err)
	}
	if _, err := parser.ParseFile(fset, packageFile.Name(), candidate, parser.ParseComments|parser.AllErrors); err != nil {
		return nil, fmt.Errorf("parse generated destination source: %w", err)
	}
	return bytes.Clone(candidate), nil
}
