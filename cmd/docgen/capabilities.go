// capabilities.go isolates AST and registry capability extraction from the docgen entrypoint.
package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"semedit/internal/backend"
	"semedit/internal/operation"
	"strings"
)

func extractCodeCapabilities(rootDir string) ([]CodeCapability, []string, error) {
	// Parse internal/astedit/insert.go for placement qualifier constants.
	// These are used by the docs template and are not exposed through the backend interface.

	insertFile := filepath.Join(rootDir, "internal", "astedit", "insert.go")
	fset := token.NewFileSet()
	insertNode, err := parser.ParseFile(fset, insertFile, nil, 0)

	var placements []string
	if err == nil {
		ast.Inspect(insertNode, func(n ast.Node) bool {
			gen, ok := n.(*ast.GenDecl)
			if !ok || gen.Tok != token.CONST {
				return true
			}
			for _, spec := range gen.Specs {
				valSpec, ok := spec.(*ast.ValueSpec)
				if !ok {
					continue
				}
				for _, val := range valSpec.Values {
					if basic, ok := val.(*ast.BasicLit); ok && basic.Kind == token.STRING {
						valStr := strings.Trim(basic.Value, `"`)
						if strings.Contains(valStr, "_") || valStr == "file_start" || valStr == "file_end" {
							placements = append(placements, valStr)
						}
					}
				}
			}
			return true
		})
	}

	if len(placements) == 0 {
		placements = []string{
			"file_start", "file_end",
			"public_start", "public_end",
			"private_start", "private_end",
			"before_symbol", "after_symbol",
		}
	}

	// The operation registry is the documentation source of truth. Language
	// profiles provide the static limitations that accompany its handlers.
	registry := operation.DefaultRegistry()
	langOrder := []backend.LanguageID{
		backend.LanguageGo,
		backend.LanguageRust,
		backend.LanguageJava,
		backend.LanguageScala,
		backend.LanguageHaskell,
	}
	var capabilities []CodeCapability
	for _, lang := range langOrder {
		m, err := registry.Matrix(lang)
		if err != nil {
			return nil, nil, fmt.Errorf("operation capability matrix for %s: %w", lang, err)
		}
		ops := make(map[string]OpMetadata, len(m.Operations))
		for name, op := range m.Operations {
			ops[name] = OpMetadata{
				Supported:    op.Supported,
				Description:  op.Description,
				CLICommand:   op.CLICommand,
				MCPTool:      op.MCPTool,
				PlacementKey: op.PlacementKey,
			}
		}
		constraints := make([]Constraint, len(m.Limitations))
		for i, lim := range m.Limitations {
			constraints[i] = Constraint{
				Title:       lim.Title,
				Description: lim.Description,
				Severity:    lim.Severity,
			}
		}
		capabilities = append(capabilities, CodeCapability{
			Language:           m.Language,
			DisplayName:        m.DisplayName,
			Maturity:           m.Maturity,
			SupportedModifiers: m.SupportedModifiers,
			Operations:         ops,
			Limitations:        constraints,
		})
	}

	return capabilities, placements, nil
}
