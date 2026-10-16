// Package operation renders compact, source-oriented navigation summaries for semantic read results.
package operation

import (
	"fmt"
	"strings"

	"github.com/spockz/semantic-editor/internal/backend"
)

func renderLookupNavigation(result *LookupRes) string {
	if result == nil {
		return "Lookup returned no result."
	}
	view := result.SourceView
	if view == "" {
		view = "disk"
	}
	if result.Ambiguous {
		lines := []string{fmt.Sprintf("Ambiguous lookup for %q in %s source: %d candidates.", result.Symbol, view, len(result.Candidates))}
		for _, candidate := range result.Candidates {
			lines = append(lines, renderLookupTarget(candidate))
		}
		appendLookupLocalBindings(&lines, result.Usages, nil)
		return strings.Join(lines, "\n")
	}
	if result.Definition != nil {
		lines := []string{"Resolved lookup in " + view + " source:", renderLookupTarget(*result.Definition)}
		appendLookupLocalBindings(&lines, result.Usages, result.Definition)
		return strings.Join(lines, "\n")
	}
	lines := []string{fmt.Sprintf("Resolved lookup %q at %s:%d:%d (%s) in %s source.", result.Symbol, result.File, result.Line, result.Column, result.Kind, view)}
	appendLookupLocalBindings(&lines, result.Usages, nil)
	return strings.Join(lines, "\n")
}

func renderLookupTarget(target LookupTarget) string {
	startLine, endLine := target.Line, target.Line
	if target.DeclarationRange != nil {
		startLine, endLine = inclusiveRangeLines(*target.DeclarationRange)
	}
	return backend.RenderNavigationDeclaration(backend.NavigationDeclaration{File: target.File, Name: target.Name, Selector: target.QualifiedName, Description: backend.NavigationDescription(target.Kind, target.Name, target.Signature), StartLine: startLine, EndLine: endLine, StartColumn: target.Column, PointOnly: target.DeclarationRange == nil})
}

func appendLookupLocalBindings(lines *[]string, usages []LookupTarget, definition *LookupTarget) {
	bindings := make([]LookupTarget, 0, len(usages))
	for _, usage := range usages {
		if definition != nil && usage.File == definition.File && usage.Line == definition.Line && usage.Column == definition.Column {
			continue
		}
		bindings = append(bindings, usage)
	}
	if len(bindings) == 0 {
		return
	}
	*lines = append(*lines, "Local bindings (navigation only):")
	for _, binding := range bindings {
		binding.QualifiedName = ""
		binding.DeclarationRange = nil
		binding.Signature = nil
		*lines = append(*lines, "  "+renderLookupTarget(binding))
	}
}

func renderOutlineNavigation(result *backend.OutlineResult) string {
	if result == nil {
		return "Outline returned no result."
	}
	lines := []string{fmt.Sprintf("Outline of %s: %d files, %d declarations.", renderScope(result.Scope), len(result.Files), countReadSymbols(result.Files))}
	for _, file := range result.Files {
		view := file.SourceView
		if view == "" {
			view = "disk"
		}
		details := []string{string(file.Language), "source: " + view, "revision: " + file.Revision}
		if file.Package != "" {
			details = append(details, "package: "+file.Package)
		}
		lines = append(lines, file.File+" | "+strings.Join(details, " | "))
		for _, symbol := range file.Symbols {
			appendReadSymbolLines(&lines, symbol, 1)
		}
	}
	return strings.Join(lines, "\n")
}

func appendReadSymbolLines(lines *[]string, symbol backend.ReadSymbol, depth int) {
	indent := strings.Repeat("  ", depth)
	startLine, endLine := inclusiveRangeLines(symbol.Range)
	description := backend.NavigationDescription(symbol.Kind, symbol.Name, symbol.Signature)
	if len(symbol.Children) > 0 {
		description = strings.TrimSpace(symbol.Kind + " " + symbol.Name)
	}
	row := backend.RenderNavigationDeclaration(backend.NavigationDeclaration{File: symbol.File, Name: symbol.Name, Selector: symbol.QualifiedName, Description: description, StartLine: startLine, EndLine: endLine})
	*lines = append(*lines, indentNavigationRow(row, indent))
	for _, child := range symbol.Children {
		appendReadSymbolLines(lines, child, depth+1)
	}
}

func countReadSymbols(files []backend.OutlineFile) int {
	count := 0
	var visit func(backend.ReadSymbol)
	visit = func(symbol backend.ReadSymbol) {
		count++
		for _, child := range symbol.Children {
			visit(child)
		}
	}
	for _, file := range files {
		for _, symbol := range file.Symbols {
			visit(symbol)
		}
	}
	return count
}

func renderReferencesNavigation(result *backend.ReferencesResult) string {
	if result == nil {
		return "Reference search returned no result."
	}
	symbol := result.Symbol
	view := result.SourceView
	if view == "" {
		view = "disk"
	}
	lines := []string{fmt.Sprintf("References to %s in %s (source: %s): %d locations in %d files.", declarationIdentity(symbol.Name, symbol.QualifiedName), renderScope(result.Scope), view, len(result.References), countReferenceFiles(result.References))}
	groups := make(map[string][]backend.Reference)
	order := make([]string, 0)
	for _, reference := range result.References {
		group := reference.File + "\x00" + reference.EnclosingSymbol
		if _, exists := groups[group]; !exists {
			order = append(order, group)
		}
		groups[group] = append(groups[group], reference)
	}
	for _, group := range order {
		file, owner, _ := strings.Cut(group, "\x00")
		references := groups[group]
		if owner != "" {
			if declaration := references[0].EnclosingDeclaration; declaration != nil {
				startLine, endLine := inclusiveRangeLines(declaration.Range)
				row := backend.RenderNavigationDeclaration(backend.NavigationDeclaration{File: declaration.File, Name: declaration.Name, Selector: declaration.QualifiedName, Description: backend.NavigationDescription(declaration.Kind, declaration.Name, declaration.Signature), StartLine: startLine, EndLine: endLine})
				lines = append(lines, indentNavigationRow(row, "  "))
				lines = append(lines, "  references at "+renderReferencePositions(references))
				continue
			}
			lines = append(lines, file+" | symbol: "+owner, "  references at "+renderReferencePositions(references))
			continue
		}
		for _, reference := range references {
			start := reference.Range.Start
			line := fmt.Sprintf("  %s:L%d:C%d", reference.File, start.Line+1, start.Character+1)
			if snippet := compactReferenceContext(reference.Snippet); snippet != "" {
				line += " " + snippet
			}
			lines = append(lines, line)
		}
	}
	return strings.Join(lines, "\n")
}

func countReferenceFiles(references []backend.Reference) int {
	files := make(map[string]struct{}, len(references))
	for _, reference := range references {
		files[reference.File] = struct{}{}
	}
	return len(files)
}

func indentNavigationRow(row, indent string) string {
	return indent + strings.ReplaceAll(row, "\n", "\n"+indent)
}

func compactReferenceContext(snippet string) string {
	const maximumRunes = 160
	compact := strings.Join(strings.Fields(snippet), " ")
	runes := []rune(compact)
	if len(runes) <= maximumRunes {
		return compact
	}
	return string(runes[:maximumRunes]) + "…"
}

func renderReferencePositions(references []backend.Reference) string {
	positions := make([]string, 0, len(references))
	for _, reference := range references {
		positions = append(positions, fmt.Sprintf("L%d:C%d", reference.Range.Start.Line+1, reference.Range.Start.Character+1))
	}
	return strings.Join(positions, ", ")
}

func declarationIdentity(name, qualifiedName string) string {
	if qualifiedName != "" {
		return qualifiedName
	}
	return name
}

func renderScope(scope backend.ReadScope) string {
	state := "partial"
	if scope.Complete {
		state = "complete"
	}
	value := fmt.Sprintf("%s %q (%s)", scope.Kind, scope.Path, state)
	if len(scope.Limitations) > 0 {
		value += "; limitations: " + strings.Join(scope.Limitations, "; ")
	}
	return value
}

func inclusiveRangeLines(value backend.Range) (int, int) {
	start := value.Start.Line + 1
	end := value.End.Line + 1
	if value.End.Line > value.Start.Line && value.End.Character == 0 {
		end--
	}
	if end < start {
		end = start
	}
	return start, end
}
