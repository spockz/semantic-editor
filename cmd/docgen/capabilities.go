// capabilities.go isolates AST and registry capability extraction from the docgen entrypoint.
package main

import (
	"fmt"
	"github.com/spockz/semantic-editor/internal/backend"
	"github.com/spockz/semantic-editor/internal/capability"
	"github.com/spockz/semantic-editor/internal/operation"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
)

func extractCodeCapabilities(rootDir string) ([]CodeCapability, []string, error) {
	insertFile := filepath.Join(rootDir, "internal", "astedit", "insert.go")
	placements, err := extractPlacementConstants(insertFile)
	if err != nil {
		return nil, nil, err
	}
	if len(placements) == 0 {
		placements = defaultPlacementConstants()
	}
	capabilities, err := extractOperationCapabilities()
	if err != nil {
		return nil, nil, err
	}
	return capabilities, placements, nil
}

func extractPlacementConstants(insertFile string) ([]string, error) {
	fset := token.NewFileSet()
	insertNode, err := parser.ParseFile(fset, insertFile, nil, 0)
	if err != nil {
		return nil, fmt.Errorf("parse placement constants %s: %w", insertFile, err)
	}
	var placements []string
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
				appendPlacementConstant(&placements, val)
			}
		}
		return true
	})
	return placements, nil
}

func appendPlacementConstant(placements *[]string, value ast.Expr) {
	basic, ok := value.(*ast.BasicLit)
	if !ok || basic.Kind != token.STRING {
		return
	}
	valueText := strings.Trim(basic.Value, `"`)
	if strings.Contains(valueText, "_") || valueText == "file_start" || valueText == "file_end" {
		*placements = append(*placements, valueText)
	}
}

func defaultPlacementConstants() []string {
	return []string{
		"file_start", "file_end",
		"public_start", "public_end",
		"private_start", "private_end",
		"before_symbol", "after_symbol",
	}
}

func extractOperationCapabilities() ([]CodeCapability, error) {
	registry := operation.DefaultRegistry()
	langOrder := []backend.LanguageID{
		backend.LanguageGo,
		backend.LanguageRust,
		backend.LanguageJava,
		backend.LanguageScala,
		backend.LanguageHaskell,
		backend.LanguageKotlin,
		backend.LanguageBash,
		backend.LanguageMake,
	}
	var capabilities []CodeCapability
	for _, lang := range langOrder {
		matrix, err := registry.Matrix(lang)
		if err != nil {
			return nil, fmt.Errorf("operation capability matrix for %s: %w", lang, err)
		}
		capabilities = append(capabilities, capabilityFromMatrix(matrix))
	}
	return capabilities, nil
}

func capabilityFromMatrix(matrix capability.LanguageMatrix) CodeCapability {
	ops := make(map[string]OpMetadata, len(matrix.Operations))
	for name, op := range matrix.Operations {
		ops[name] = OpMetadata{
			Supported:        op.Supported,
			Description:      op.Description,
			CLICommand:       op.CLICommand,
			MCPTool:          op.MCPTool,
			PlacementKey:     op.PlacementKey,
			LanguageContract: op.LanguageContract,
		}
	}
	constraints := make([]Constraint, len(matrix.Limitations))
	for i, limitation := range matrix.Limitations {
		constraints[i] = Constraint{
			Title:       limitation.Title,
			Description: limitation.Description,
			Severity:    limitation.Severity,
		}
	}
	return CodeCapability{
		Language:           matrix.Language,
		DisplayName:        matrix.DisplayName,
		Maturity:           matrix.Maturity,
		SupportedModifiers: matrix.SupportedModifiers,
		Operations:         ops,
		Limitations:        constraints,
	}
}
