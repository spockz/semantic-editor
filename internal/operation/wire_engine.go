// Package operation wires host-engine editing operations into the central registry.
package operation

import (
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"

	"semedit/internal/pipeline"
	"semedit/internal/sourcebuf"
	"semedit/internal/telemetry"
)

var (
	structurePlacementEnum = []string{"file_start", "file_end", "public_start", "public_end", "private_start", "private_end", "before_symbol", "after_symbol", "first", "last", "before", "after", "before_default"}
	placementEnum          = []string{"file_start", "file_end", "public_start", "public_end", "private_start", "private_end", "before_symbol", "after_symbol"}
	accessEnum             = []string{"infer", "public", "private", "protected", "package-private"}
	visibilityEnum         = []string{"public", "private"}
	groupEnum              = []string{"append", "standalone"}
	caseEnum               = []string{"first", "last", "before_default", "before", "after"}
)

// AppendDiagnosticDelta appends a compiler diagnostic delta to a response base string.
// It mirrors the MCP server helper retired in the ingress migration step.
func AppendDiagnosticDelta(base string, delta pipeline.DiagnosticDelta) string {
	if len(delta.Introduced) == 0 && len(delta.Resolved) == 0 && len(delta.Suggestions) == 0 {
		return base
	}
	var sb strings.Builder
	sb.WriteString(base)
	if len(delta.Introduced) > 0 || len(delta.Resolved) > 0 {
		fmt.Fprintf(&sb, "\nDiagnostics delta (net %d):\n", delta.NetDelta)
		if len(delta.Resolved) > 0 {
			fmt.Fprintf(&sb, "Resolved:\n- %s\n", strings.Join(delta.Resolved, "\n- "))
		}
		if len(delta.Introduced) > 0 {
			fmt.Fprintf(&sb, "Introduced:\n- %s\n", strings.Join(delta.Introduced, "\n- "))
		}
	}
	if len(delta.Suggestions) > 0 {
		fmt.Fprintf(&sb, "\nActionable suggestions:\n- %s\n", strings.Join(delta.Suggestions, "\n- "))
	}
	return sb.String()
}

// resolveWorkPath mirrors the MCP server path resolution: relative paths join the workspace root.
func resolveWorkPath(workDir, file string) string {
	if file == "" {
		return workDir
	}
	if workDir != "" && !filepath.IsAbs(file) {
		return filepath.Join(workDir, file)
	}
	return file
}

func displayWorkPath(workDir, file string) string {
	if strings.TrimSpace(file) == "" {
		return ""
	}
	root, err := filepath.Abs(workDir)
	if err != nil {
		return filepath.ToSlash(filepath.Clean(file))
	}
	target := resolveWorkPath(root, file)
	absolute, err := filepath.Abs(target)
	if err != nil {
		return filepath.ToSlash(filepath.Clean(file))
	}
	relative, err := filepath.Rel(root, absolute)
	if err == nil && relative != "." && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return filepath.ToSlash(relative)
	}
	return filepath.ToSlash(filepath.Clean(file))
}

// effectiveAutoOrganize applies the batch deferral flag over the parsed default.
func effectiveAutoOrganize(cc CallContext, parsed bool) bool {
	if cc.DeferImports {
		return false
	}
	return parsed
}

const automaticVerificationGuidance = " This standalone operation automatically verifies diagnostics and has no verification opt-out; do not call semantic_verify separately."

func surroundingDelta(ctx context.Context, workDir string, deferVerification bool) func() (pipeline.DiagnosticDelta, string) {
	if deferVerification {
		return func() (pipeline.DiagnosticDelta, string) { return pipeline.DiagnosticDelta{}, "deferred" }
	}
	before, beforeErr := pipeline.CheckDiagnostics(telemetry.WithPhase(ctx, telemetry.PhaseVerificationBefore), workDir)
	return func() (pipeline.DiagnosticDelta, string) {
		after, afterErr := pipeline.CheckDiagnostics(telemetry.WithPhase(ctx, telemetry.PhaseVerificationAfter), workDir)
		if beforeErr != nil || afterErr != nil {
			return pipeline.DiagnosticDelta{}, "unavailable"
		}
		delta := pipeline.ComputeDelta(before, after)
		if len(delta.Introduced)+len(delta.Resolved)+len(delta.Suggestions) == 0 {
			return delta, "unchanged"
		}
		return delta, "changed"
	}
}

// FileEditRes reports a single-file mutation for response rendering and batch post-processing.
type FileEditRes struct {
	File               string
	DestinationFile    string
	DestinationDisplay string
	Display            string
	Symbol             string
	InType             string
	Line               int
	InFunction         string
	Discriminator      string
	DiagnosticStatus   string
	Diff               string
	Detail             string
	Delta              pipeline.DiagnosticDelta
	HasDelta           bool
}

func formatFileEdit(res FileEditRes) (string, error) {
	text := res.Detail
	if res.Display != "" {
		text += fmt.Sprintf(" file: %s.", res.Display)
	}
	if res.DestinationDisplay != "" {
		text += fmt.Sprintf(" destination_file: %s.", res.DestinationDisplay)
	}
	if res.Symbol != "" {
		text += fmt.Sprintf(" symbol: %s.", res.Symbol)
	}
	if res.InType != "" {
		text += fmt.Sprintf(" in_type: %s.", res.InType)
	}
	if res.Line > 0 {
		text += fmt.Sprintf(" line (navigation only): %d.", res.Line)
	}
	if res.InFunction != "" {
		text += fmt.Sprintf(" in_function: %s.", res.InFunction)
	}
	if res.Discriminator != "" {
		text += fmt.Sprintf(" discriminator: %s.", res.Discriminator)
	}
	if res.Diff != "" {
		text += "\n" + res.Diff
	}
	if res.HasDelta {
		text += " diagnostic_status: " + res.DiagnosticStatus + "."
		if res.DiagnosticStatus == "changed" {
			text = AppendDiagnosticDelta(text, res.Delta)
		}
	}
	return text, nil
}

func receiverNameForDecl(decl *ast.FuncDecl) string {
	if decl.Recv == nil || len(decl.Recv.List) != 1 {
		return ""
	}
	return receiverName(decl.Recv.List[0].Type)
}

func receiverName(expr ast.Expr) string {
	switch node := expr.(type) {
	case *ast.StarExpr:
		return receiverName(node.X)
	case *ast.Ident:
		return node.Name
	case *ast.IndexExpr:
		return receiverName(node.X)
	case *ast.IndexListExpr:
		return receiverName(node.X)
	}
	return ""
}

func qualifiedGoTarget(receiver, name string) string {
	if receiver == "" {
		return name
	}
	return receiver + "." + name
}

// registerEngineOps adds the host-engine file and build operations to the registry.
func registerEngineOps(registry *Registry) error {
	if err := Register(registry, assertionModeDef()); err != nil {
		return err
	}
	if err := Register(registry, insertStructureDef()); err != nil {
		return err
	}
	if err := Register(registry, insertDeclarationDef()); err != nil {
		return err
	}
	if err := Register(registry, insertFunctionDef()); err != nil {
		return err
	}
	if err := Register(registry, insertTypeDef()); err != nil {
		return err
	}
	if err := Register(registry, insertDeclDef()); err != nil {
		return err
	}
	if err := Register(registry, replaceDeclDef()); err != nil {
		return err
	}
	if err := Register(registry, removeDef()); err != nil {
		return err
	}
	if err := Register(registry, organizeImportsDef()); err != nil {
		return err
	}
	if err := Register(registry, addBuildDependencyDef()); err != nil {
		return err
	}
	if err := Register(registry, replaceBodyDef()); err != nil {
		return err
	}
	if err := Register(registry, replaceConstructDef()); err != nil {
		return err
	}
	if err := Register(registry, moveConstructDef()); err != nil {
		return err
	}
	if err := registerExtractOps(registry); err != nil {
		return err
	}
	if err := Register(registry, inlineVariableDef()); err != nil {
		return err
	}
	if err := Register(registry, migrateTypeDef()); err != nil {
		return err
	}
	if err := Register(registry, inlineFunctionDef()); err != nil {
		return err
	}
	if err := Register(registry, scaffoldFileDef()); err != nil {
		return err
	}
	if err := Register(registry, insertCaseDef()); err != nil {
		return err
	}
	return registerStructuralPlanOps(registry)
}

func goTargetAfterEdit(ctx context.Context, filePath, source string) (string, int, error) {
	prefix := "package p\n"
	if strings.HasPrefix(strings.TrimSpace(source), "package ") {
		prefix = ""
	}
	snippet, err := parser.ParseFile(token.NewFileSet(), "snippet.go", prefix+source, parser.ParseComments|parser.AllErrors)
	if err != nil {
		return "", 0, fmt.Errorf("parse inserted declaration after write: %w", err)
	}
	if len(snippet.Decls) != 1 {
		return "", 0, nil
	}
	var name, receiver string
	switch decl := snippet.Decls[0].(type) {
	case *ast.FuncDecl:
		name = decl.Name.Name
		if decl.Recv != nil && len(decl.Recv.List) == 1 {
			receiver = receiverName(decl.Recv.List[0].Type)
			if receiver == "" {
				return "", 0, nil
			}
		}
	case *ast.GenDecl:
		if len(decl.Specs) != 1 {
			return "", 0, nil
		}
		switch spec := decl.Specs[0].(type) {
		case *ast.TypeSpec:
			name = spec.Name.Name
		case *ast.ValueSpec:
			if len(spec.Names) != 1 {
				return "", 0, nil
			}
			name = spec.Names[0].Name
		}
	}
	if name == "" {
		return "", 0, nil
	}
	content, err := sourcebuf.ReadFile(ctx, filePath) // #nosec G304 -- filePath is the resolved target of a validated edit operation.
	if err != nil {
		return "", 0, fmt.Errorf("read edited file %q: %w", filePath, err)
	}
	fset := token.NewFileSet()
	final, err := parser.ParseFile(fset, filePath, content, parser.ParseComments|parser.AllErrors)
	if err != nil {
		return "", 0, fmt.Errorf("parse edited file %q: %w", filePath, err)
	}
	var foundSymbol string
	var foundLine int
	for _, decl := range final.Decls {
		switch node := decl.(type) {
		case *ast.FuncDecl:
			if node.Name.Name == name && receiverNameForDecl(node) == receiver {
				if foundSymbol != "" {
					return "", 0, nil
				}
				foundSymbol, foundLine = qualifiedGoTarget(receiver, name), fset.Position(node.Pos()).Line
			}
		case *ast.GenDecl:
			for _, spec := range node.Specs {
				var found bool
				switch item := spec.(type) {
				case *ast.TypeSpec:
					found = item.Name.Name == name
				case *ast.ValueSpec:
					for _, candidate := range item.Names {
						found = found || candidate.Name == name
					}
				}
				if found {
					if foundSymbol != "" {
						return "", 0, nil
					}
					foundSymbol, foundLine = name, fset.Position(spec.Pos()).Line
				}
			}
		}
	}
	if foundSymbol != "" {
		return foundSymbol, foundLine, nil
	}
	return "", 0, fmt.Errorf("inserted declaration target %q is absent from edited file %q", name, filePath)
}

func goExistingTargetAfterEdit(ctx context.Context, filePath, symbol string) (string, int, error) {
	content, err := sourcebuf.ReadFile(ctx, filePath) // #nosec G304 -- filePath is the resolved target of a validated edit operation.
	if err != nil {
		return "", 0, fmt.Errorf("read edited file %q: %w", filePath, err)
	}
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filePath, content, parser.ParseComments|parser.AllErrors)
	if err != nil {
		return "", 0, fmt.Errorf("parse edited file %q: %w", filePath, err)
	}
	var line int
	count := 0
	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, raw := range gen.Specs {
			matched := false
			switch spec := raw.(type) {
			case *ast.TypeSpec:
				matched = spec.Name.Name == symbol
			case *ast.ValueSpec:
				for _, name := range spec.Names {
					matched = matched || name.Name == symbol
				}
			}
			if matched {
				count++
				line = fset.Position(raw.Pos()).Line
			}
		}
	}
	if count != 1 {
		return "", 0, nil
	}
	return symbol, line, nil
}
