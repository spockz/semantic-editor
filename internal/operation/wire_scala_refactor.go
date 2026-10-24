// Package operation routes the verified Scala extraction slice through the shared code-action service.
// WHY: CLI, MCP, and buffered batch calls need one selector, safety contract, and receipt path.
package operation

import (
	"bytes"
	"context"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"unicode/utf16"

	"github.com/spockz/semantic-editor/internal/astedit"
	"github.com/spockz/semantic-editor/internal/backend"
	"github.com/spockz/semantic-editor/internal/backends"
	"github.com/spockz/semantic-editor/internal/sourcebuf"
)

func runExtractVariableScalaWithReceipt(ctx context.Context, cc CallContext, req ExtractVariableReq) (ExtractionResult, error) {
	if req.Kind != "variable" {
		return ExtractionResult{}, fmt.Errorf("scala extract_value currently supports kind=variable only")
	}
	if req.ConstructPath != "" {
		return ExtractionResult{}, fmt.Errorf("scala extraction does not accept construct_path; provide one exact discriminator")
	}
	service := cc.Service
	if service == nil {
		service = backends.NewDefaultService()
	}
	project := effectiveProject(cc, req.Project)
	file := resolveWorkPath(cc.WorkDir, req.File)
	project.File = file
	source, err := sourcebuf.ReadFile(ctx, file)
	if err != nil {
		return ExtractionResult{}, fmt.Errorf("read scala extraction source: %w", err)
	}
	selection, err := scalaExtractionSelection(source, req.InFunction, req.Discriminator)
	if err != nil {
		return ExtractionResult{}, err
	}
	result, err := service.ApplyCodeAction(ctx, backend.CodeActionRequest{
		Operation: backend.OperationExtractVariable, Project: project, File: file,
		Range: selection, Kind: "refactor.extract", NewName: req.Symbol,
		Symbol: req.InFunction, ExpectedRevision: req.ExpectedRevision,
	})
	if err != nil {
		return ExtractionResult{}, err
	}
	if result == nil || result.File != file {
		return ExtractionResult{}, fmt.Errorf("scala extraction backend returned an unexpected target file")
	}
	readSet, planned, err := codeActionFileRevisions(cc, result)
	if err != nil {
		return ExtractionResult{}, err
	}
	relative, err := relativeWorkPath(cc.WorkDir, file)
	if err != nil {
		return ExtractionResult{}, fmt.Errorf("derive Scala extraction continuation path for %q: %w", file, err)
	}
	revision := structuralRevision(result.After)
	draft := EditReceiptDraft{
		Operation: string(backend.OperationExtractVariable), Status: EditPersisted,
		ReadSet: readSet, PlannedChanges: planned,
		Continuations: []SymbolTarget{{File: relative, Symbol: req.InFunction, ExpectedRevision: &revision}},
	}
	return ExtractionResult{Edit: FileEditRes{
		File: file, Display: displayWorkPath(cc.WorkDir, req.File), Symbol: req.Symbol,
		InFunction: req.InFunction, Discriminator: req.Discriminator,
		Diff:             astedit.UnifiedDiff(displayWorkPath(cc.WorkDir, req.File), string(result.Before), string(result.After)),
		Detail:           "Extracted the Scala integer expression through a captured Metals action and local rename.",
		DiagnosticStatus: diagnosticStatusUnavailable, HasDelta: !bytes.Equal(result.Before, result.After),
	}, Draft: draft}, nil
}

func runExtractFunctionScalaWithReceipt(ctx context.Context, cc CallContext, req ExtractFunctionReq) (ExtractionResult, error) {
	if req.Kind != astedit.ExtractKindFunction {
		return ExtractionResult{}, fmt.Errorf("scala extract_callable currently supports kind=function only")
	}
	service := cc.Service
	if service == nil {
		service = backends.NewDefaultService()
	}
	project := effectiveProject(cc, req.Project)
	file := resolveWorkPath(cc.WorkDir, req.File)
	project.File = file
	owner, err := service.Lookup(ctx, project, req.InFunction)
	if err != nil {
		return ExtractionResult{}, err
	}
	if owner == nil || owner.Definition == nil || owner.Definition.DeclarationRange == nil || owner.Definition.Kind != "method" && owner.Definition.Kind != "function" {
		return ExtractionResult{}, fmt.Errorf("scala extraction owner %q must resolve to one method", req.InFunction)
	}
	source, err := sourcebuf.ReadFile(ctx, file)
	if err != nil {
		return ExtractionResult{}, fmt.Errorf("read Scala extraction preimage: %w", err)
	}
	selection, statement, err := scalaCallableStatementRange(source, *owner.Definition.DeclarationRange, req.StartLine, req.EndLine)
	if err != nil {
		return ExtractionResult{}, err
	}
	result, err := service.ApplyCodeAction(ctx, backend.CodeActionRequest{
		Operation: backend.OperationExtractFunction, Project: project, File: file,
		Range: selection, Kind: "refactor.extract", NewName: req.Symbol,
		Expression: statement, EnclosingSymbol: owner.Definition.QualifiedName,
	})
	if err != nil {
		return ExtractionResult{}, err
	}
	if result == nil || result.File != file || !filepath.IsAbs(result.File) || !bytes.Equal(result.Before, source) {
		return ExtractionResult{}, fmt.Errorf("scala callable extraction backend returned an unexpected target or source preimage")
	}
	readSet, planned, err := codeActionFileRevisions(cc, result)
	if err != nil {
		return ExtractionResult{}, err
	}
	relative, err := relativeWorkPath(cc.WorkDir, file)
	if err != nil {
		return ExtractionResult{}, fmt.Errorf("derive Scala callable extraction continuation path for %q: %w", file, err)
	}
	revision := structuralRevision(result.After)
	draft := EditReceiptDraft{
		Operation: string(backend.OperationExtractFunction), Status: EditPersisted,
		ReadSet: readSet, PlannedChanges: planned,
		Continuations: []SymbolTarget{{File: relative, Symbol: req.Symbol, ExpectedRevision: &revision}},
	}
	return ExtractionResult{Edit: FileEditRes{
		File: file, Display: displayWorkPath(cc.WorkDir, req.File), Symbol: req.Symbol,
		InFunction:       owner.Definition.QualifiedName,
		Diff:             astedit.UnifiedDiff(displayWorkPath(cc.WorkDir, req.File), string(result.Before), string(result.After)),
		Detail:           "Extracted the Scala statement through a resolved Metals method-local action.",
		DiagnosticStatus: diagnosticStatusUnavailable, HasDelta: !bytes.Equal(result.Before, result.After),
	}, Draft: draft}, nil
}

func scalaCallableStatementRange(source []byte, ownerRange backend.Range, startLine, endLine int) (backend.Range, string, error) {
	if startLine < 1 || endLine < startLine {
		return backend.Range{}, "", fmt.Errorf("scala callable extraction requires a positive contiguous line range")
	}
	startLineOffset, err := scalaSourceLineOffset(source, startLine-1)
	if err != nil {
		return backend.Range{}, "", err
	}
	lineEnd := len(source)
	if newline := bytes.IndexByte(source[startLineOffset:], '\n'); newline >= 0 {
		lineEnd = startLineOffset + newline
	}
	first := source[startLineOffset:lineEnd]
	start := startLineOffset + len(first) - len(bytes.TrimLeft(first, " \t"))
	lastStart, err := scalaSourceLineOffset(source, endLine-1)
	if err != nil {
		return backend.Range{}, "", err
	}
	lastEnd := len(source)
	if newline := bytes.IndexByte(source[lastStart:], '\n'); newline >= 0 {
		lastEnd = lastStart + newline
	}
	last := source[lastStart:lastEnd]
	end := lastStart + len(bytes.TrimRight(last, " \t\r"))
	if start >= end {
		return backend.Range{}, "", fmt.Errorf("scala callable selection contains no complete statements")
	}
	statement := string(source[start:end])
	ownerStart, ownerStartErr := scalaSourceOffset(source, ownerRange.Start)
	ownerHeaderEnd, ownerEndErr := scalaSourceOffset(source, ownerRange.End)
	if ownerStartErr != nil || ownerEndErr != nil || start < ownerStart || !scalaSelectionWithinOwnerBody(source, ownerStart, ownerHeaderEnd, start, end) {
		return backend.Range{}, "", fmt.Errorf("scala callable selection is outside the selected method")
	}
	lines := strings.Split(statement, "\n")
	callee := ""
	for _, selectedLine := range lines {
		match := regexp.MustCompile(`^([A-Za-z_][A-Za-z0-9_]*)\(`).FindStringSubmatch(strings.TrimSpace(selectedLine))
		if len(match) != 2 || (callee != "" && match[1] != callee) {
			return backend.Range{}, "", fmt.Errorf("scala callable selection requires contiguous direct calls to one callee")
		}
		callee = match[1]
	}
	bodyOpen, bodyClose, bodyOK := scalaOwnerBodyRange(source, ownerStart, ownerHeaderEnd)
	if !bodyOK || start <= bodyOpen || end > bodyClose {
		return backend.Range{}, "", fmt.Errorf("scala callable selection is outside the selected method")
	}
	ownerSource := source[ownerStart : bodyClose+1]
	name := regexp.QuoteMeta(callee)
	if regexp.MustCompile(`(?m)\b(?:lazy\s+)?(?:val|var|def)\s+` + name + `\b`).Match(ownerSource) {
		return backend.Range{}, "", fmt.Errorf("scala callable extraction refuses a call shadowed by an owner-local binding")
	}
	header := regexp.MustCompile(`(?s)^\s*def\s+[A-Za-z_][A-Za-z0-9_]*\s*\(([^)]*)\)`).FindSubmatch(ownerSource)
	if len(header) != 2 {
		return backend.Range{}, "", fmt.Errorf("scala callable extraction requires one ordinary method parameter list")
	}
	parameter := regexp.MustCompile(`(?:^|,)\s*` + name + `\s*:`)
	if parameter.Match(header[1]) {
		return backend.Range{}, "", fmt.Errorf("scala callable extraction refuses a call shadowed by an owner parameter")
	}
	return backend.Range{Start: scalaPositionAt(source, start), End: scalaPositionAt(source, end)}, statement, nil
}

func scalaSelectionWithinOwnerBody(source []byte, ownerStart, ownerHeaderEnd, start, end int) bool {
	if ownerStart < 0 || ownerHeaderEnd <= ownerStart || start <= ownerStart || end <= start || end > len(source) {
		return false
	}
	open, close, ok := scalaOwnerBodyRange(source, ownerStart, ownerHeaderEnd)
	return ok && start > open && end <= close
}

func scalaOwnerBodyRange(source []byte, ownerStart, ownerHeaderEnd int) (int, int, bool) {
	bodyOpen, ok := scalaMethodBodyOpen(source, ownerStart, ownerHeaderEnd)
	if !ok {
		return 0, 0, false
	}
	depth := 1
	for index := bodyOpen + 1; index < len(source); index++ {
		next, skipped, unsupported := scalaSkipLexicalRegion(source, index)
		if unsupported {
			return 0, 0, false
		}
		if skipped {
			index = next - 1
			continue
		}
		switch source[index] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return bodyOpen, index, true
			}
		}
	}
	return 0, 0, false
}

func scalaMethodBodyOpen(source []byte, ownerStart, ownerHeaderEnd int) (int, bool) {
	parenDepth, bracketDepth := 0, 0
	for index := ownerStart; index < ownerHeaderEnd; index++ {
		next, skipped, unsupported := scalaSkipLexicalRegion(source, index)
		if unsupported {
			return 0, false
		}
		if skipped {
			index = next - 1
			continue
		}
		switch source[index] {
		case '(':
			parenDepth++
		case ')':
			parenDepth--
		case '[':
			bracketDepth++
		case ']':
			bracketDepth--
		case '}':
			if parenDepth == 0 && bracketDepth == 0 {
				return 0, false
			}
		case '=':
			if parenDepth == 0 && bracketDepth == 0 {
				nextCode, ok := scalaSkipTrivia(source, index+1)
				return nextCode, ok && source[nextCode] == '{'
			}
		}
	}
	return 0, false
}

func scalaSkipLexicalRegion(source []byte, index int) (int, bool, bool) {
	if index < 0 || index >= len(source) {
		return index, false, false
	}
	if index+1 < len(source) && source[index] == '/' && source[index+1] == '/' {
		if end := bytes.IndexByte(source[index:], '\n'); end >= 0 {
			return index + end + 1, true, false
		}
		return len(source), true, false
	}
	if index+1 < len(source) && source[index] == '/' && source[index+1] == '*' {
		end, closed := scalaSkipBlockComment(source, index)
		return end, true, !closed
	}
	if source[index] == '"' || source[index] == '\'' {
		return scalaSkipQuoted(source, index)
	}
	if source[index] == '`' {
		return index, false, true
	}
	if source[index] == '<' && index+1 < len(source) {
		next := source[index+1]
		if next == '!' || next == '?' || next == '_' || next == ':' || next >= 0x80 || next >= 'a' && next <= 'z' || next >= 'A' && next <= 'Z' {
			return index, false, true
		}
	}
	return index, false, false
}

func scalaSkipBlockComment(source []byte, index int) (int, bool) {
	depth, cursor := 1, index+2
	for cursor < len(source) && depth > 0 {
		switch {
		case cursor+1 < len(source) && source[cursor] == '/' && source[cursor+1] == '*':
			depth++
			cursor += 2
		case cursor+1 < len(source) && source[cursor] == '*' && source[cursor+1] == '/':
			depth--
			cursor += 2
		default:
			cursor++
		}
	}
	return cursor, depth == 0
}

func scalaSkipQuoted(source []byte, index int) (int, bool, bool) {
	quote, prefix := source[index], 1
	if quote != '"' && quote != '\'' {
		return index, false, false
	}
	// Scala 2 symbol literals and quoted-code syntax can contain apostrophes
	// without a closing quote, so a brace scanner cannot safely skip them.
	if quote == '\'' {
		return index, false, true
	}
	if quote == '"' && index+2 < len(source) && string(source[index:index+3]) == `"""` {
		prefix = 3
	}
	if quote == '"' && scalaHasStringPrefix(source, index) {
		return index, false, true
	}
	for cursor := index + prefix; cursor < len(source); cursor++ {
		if prefix == 3 {
			if cursor+2 < len(source) && string(source[cursor:cursor+3]) == `"""` {
				return cursor + 3, true, false
			}
			continue
		}
		switch source[cursor] {
		case '\\':
			cursor++
		case quote:
			return cursor + 1, true, false
		}
	}
	return len(source), true, true
}

func scalaHasStringPrefix(source []byte, quote int) bool {
	if quote == 0 {
		return false
	}
	previous := source[quote-1]
	return previous == '_' || previous == '$' || previous >= '0' && previous <= '9' || previous >= 'a' && previous <= 'z' || previous >= 'A' && previous <= 'Z' || previous >= 0x80 || strings.ContainsRune("!#%&*+-/:<=>?@\\^|~", rune(previous))
}

func scalaSkipTrivia(source []byte, offset int) (int, bool) {
	for offset < len(source) {
		switch {
		case source[offset] == ' ' || source[offset] == '\t' || source[offset] == '\r' || source[offset] == '\n':
			offset++
		case offset+1 < len(source) && source[offset] == '/' && (source[offset+1] == '/' || source[offset+1] == '*'):
			next, _, unsupported := scalaSkipLexicalRegion(source, offset)
			if unsupported {
				return 0, false
			}
			offset = next
		default:
			return offset, true
		}
	}
	return 0, false
}

func scalaSourceLineOffset(source []byte, target int) (int, error) {
	offset := 0
	for range target {
		next := bytes.IndexByte(source[offset:], '\n')
		if next < 0 {
			return 0, fmt.Errorf("scala callable selection line is outside the source")
		}
		offset += next + 1
	}
	return offset, nil
}

func scalaSourceOffset(source []byte, position backend.Position) (int, error) {
	if position.Line < 0 || position.Character < 0 {
		return 0, fmt.Errorf("negative Scala source position")
	}
	lineStart := 0
	for range position.Line {
		next := bytes.IndexByte(source[lineStart:], '\n')
		if next < 0 {
			return 0, fmt.Errorf("scala source line is outside the file")
		}
		lineStart += next + 1
	}
	lineEnd := len(source)
	if next := bytes.IndexByte(source[lineStart:], '\n'); next >= 0 {
		lineEnd = lineStart + next
	}
	units := 0
	line := string(source[lineStart:lineEnd])
	for byteOffset, character := range line {
		if units == position.Character {
			return lineStart + byteOffset, nil
		}
		units += len(utf16.Encode([]rune{character}))
	}
	if units == position.Character {
		return lineEnd, nil
	}
	return 0, fmt.Errorf("scala source character is outside the selected line")
}

func scalaExtractionSelection(source []byte, owner, expression string) (backend.Range, error) {
	if strings.TrimSpace(owner) == "" || strings.TrimSpace(expression) == "" {
		return backend.Range{}, fmt.Errorf("scala extraction requires an owner method and exact expression discriminator")
	}
	pattern := regexp.MustCompile(`(?m)^([ \t]*def\s+` + regexp.QuoteMeta(owner) + `\s*\(\s*\)\s*:\s*Int\s*=\s*[A-Za-z_][A-Za-z0-9_]*\(\s*)(` + regexp.QuoteMeta(expression) + `)([ \t]*\)[ \t]*)$`)
	matches := pattern.FindAllSubmatchIndex(source, 2)
	if len(matches) != 1 {
		return backend.Range{}, fmt.Errorf("scala extraction requires one exact expression-bodied Int method selection")
	}
	start, end := matches[0][4], matches[0][5]
	return backend.Range{Start: scalaPositionAt(source, start), End: scalaPositionAt(source, end)}, nil
}

func scalaPositionAt(source []byte, offset int) backend.Position {
	line, lineStart := 0, 0
	for index := range offset {
		if source[index] == '\n' {
			line++
			lineStart = index + 1
		}
	}
	return backend.Position{Line: line, Character: len(utf16.Encode([]rune(string(source[lineStart:offset]))))}
}

func finalizeScalaExtractionOutput(cc CallContext, req ExtractVariableReq, result ExtractionResult) (ExtractionResult, error) {
	resolver := func(ctx context.Context, candidate SymbolTarget) (SymbolTarget, bool, error) {
		path := filepath.Join(cc.WorkDir, filepath.FromSlash(candidate.File))
		source, err := sourcebuf.ReadFile(ctx, path)
		if err != nil {
			return SymbolTarget{}, false, fmt.Errorf("read Scala extraction continuation: %w", err)
		}
		ownerPattern := regexp.MustCompile(`(?m)^\s*def\s+` + regexp.QuoteMeta(req.InFunction) + `\s*\(`)
		if !ownerPattern.Match(source) {
			return SymbolTarget{}, false, nil
		}
		return candidate, true, nil
	}
	return finalizeScalaExtractionReceipt(cc, result, resolver, "scala extraction")
}

func finalizeScalaCallableExtractionOutput(cc CallContext, req ExtractFunctionReq, result ExtractionResult) (ExtractionResult, error) {
	resolver := func(ctx context.Context, candidate SymbolTarget) (SymbolTarget, bool, error) {
		path := filepath.Join(cc.WorkDir, filepath.FromSlash(candidate.File))
		source, err := sourcebuf.ReadFile(ctx, path)
		if err != nil {
			return SymbolTarget{}, false, fmt.Errorf("read Scala callable extraction continuation: %w", err)
		}
		pattern := regexp.MustCompile(`(?m)^[ \t]*def\s+` + regexp.QuoteMeta(req.Symbol) + `\(\):\s*Unit\s*=`)
		if !pattern.Match(source) {
			return SymbolTarget{}, false, nil
		}
		return candidate, true, nil
	}
	return finalizeScalaExtractionReceipt(cc, result, resolver, "scala callable extraction")
}

func finalizeScalaExtractionReceipt(cc CallContext, result ExtractionResult, resolver func(context.Context, SymbolTarget) (SymbolTarget, bool, error), description string) (ExtractionResult, error) {
	receipt, err := finalizeCallReceipt(cc.Ctx, cc, cc.WorkDir, result.Draft, resolver)
	if err != nil {
		return ExtractionResult{}, fmt.Errorf("finalize %s receipt: %w", description, err)
	}
	result.Receipt = receipt
	return result, nil
}
