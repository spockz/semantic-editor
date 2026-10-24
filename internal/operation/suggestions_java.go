// Package operation enriches Java lookup failures with candidates that pass native backend checks.
package operation

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/spockz/semantic-editor/internal/backend"
	javabackend "github.com/spockz/semantic-editor/internal/backend/java"
	"github.com/spockz/semantic-editor/internal/sourcebuf"
	"github.com/spockz/semantic-editor/internal/suggestions"
)

const maxJavaSuggestionCandidates = 4096

// suggestJavaFailedTarget attaches only same-file candidates accepted by Java's native APIs.
func suggestJavaFailedTarget(cc CallContext, entry Entry, raw map[string]any, cause error) error {
	request, ok := prepareJavaSuggestionRequest(cc, entry, raw, cause)
	if !ok {
		return suggestNativeOutlineFailedTarget(cc, entry, raw, cause)
	}
	defer request.cancel()
	ranked, currentSelectors, err := retrieveJavaSuggestionCandidates(cc, request, cause)
	if err != nil || len(ranked) == 0 {
		return cause
	}
	accepted := acceptJavaSuggestionCandidates(request, backendOperationKey(entry.Key), ranked, currentSelectors)
	if len(accepted) == 0 {
		return cause
	}
	return &SuggestedTargetError{Cause: cause, Suggestions: accepted}
}

type javaSuggestionRequest = outlineSuggestionRequest

func prepareJavaSuggestionRequest(cc CallContext, entry Entry, raw map[string]any, cause error) (*javaSuggestionRequest, bool) {
	if !errors.Is(cause, javabackend.ErrJavaSymbolNotFound) && !errors.Is(cause, javabackend.ErrJavaSymbolAmbiguous) && !errors.Is(cause, backend.ErrAmbiguous) {
		return nil, false
	}
	if !javaSuggestionOperation(backendOperationKey(entry.Key)) {
		return nil, false
	}
	request, ok := prepareOutlineSuggestionRequest(cc, entry, raw, backend.LanguageJava)
	if !ok {
		return nil, false
	}
	normalized, err := entry.NormalizedInput(raw)
	if err != nil {
		request.cancel()
		return nil, false
	}
	selectorKey, query := failedSourceSelector(normalized, cause)
	if selectorKey == "" || query == "" || len(query) > 256 {
		request.cancel()
		return nil, false
	}
	request.selectorKey, request.query = selectorKey, query
	request.renameTo, _ = normalized["to"].(string)
	return request, true
}

func retrieveJavaSuggestionCandidates(cc CallContext, request *javaSuggestionRequest, cause error) ([]suggestions.Candidate, map[string]string, error) {
	outline, err := request.service.Outline(request.ctx, backend.OutlineRequest{Project: request.project, Path: request.project.File, IncludeUnexported: true})
	if err != nil || outline == nil || len(outline.Files) != 1 || len(outline.Files[0].Symbols) == 0 {
		return nil, nil, errors.New("java outline unavailable")
	}
	native := make([]suggestions.Candidate, 0, min(maxJavaSuggestionCandidates, 128))
	if !flattenJavaSuggestionSymbols(request.ctx, outline.Files[0].Symbols, &native) || len(native) == 0 {
		return nil, nil, errors.New("java outline candidates unavailable")
	}
	aliases := javaSuggestionAliases(request.ctx)
	currentSelectors := make(map[string]string)
	ambiguous := errors.Is(cause, backend.ErrAmbiguous) || errors.Is(cause, javabackend.ErrJavaSymbolAmbiguous)
	if len(aliases) != 0 && !ambiguous {
		native = javaFormerSelectorEvidence(request.project.RootDir, native, aliases, currentSelectors)
	}
	if ambiguous {
		lookup, lookupErr := request.service.Lookup(request.ctx, request.project, request.query)
		if lookupErr != nil || lookup == nil {
			return nil, nil, errors.New("java ambiguity candidates unavailable")
		}
		native = exactJavaAmbiguousCandidates(lookup, native)
	}
	native = constrainJavaOwner(request.query, native)
	if len(native) == 0 {
		return nil, nil, errors.New("no Java candidates match requested owner")
	}
	query := suggestions.Query{Selector: request.query, Operation: request.operation, TargetParameter: request.selectorKey, Language: "java", Scope: request.project.File, Limit: 50}
	indexed := retrieveJavaIndexedCandidates(cc, request, query, len(aliases) == 0 && !ambiguous)
	history := suggestionHistory(request.ctx)
	indexed = slices.DeleteFunc(indexed, func(candidate suggestions.Candidate) bool {
		return indexedCandidateInvalidated(request.project.RootDir, candidate, history.Invalidated)
	})
	candidates := make([]suggestions.Candidate, 0, len(native)+len(indexed))
	if ambiguous || request.mode == "" || request.mode == "generic" || request.mode == suggestionModeCombined {
		candidates = append(candidates, native...)
	}
	if request.mode == "codanna" || request.mode == suggestionModeCombined {
		candidates = append(candidates, indexed...)
	}
	query.Candidates = candidates
	ranked, err := (suggestions.Matcher{}).Retrieve(request.ctx, query)
	return ranked, currentSelectors, err
}

func retrieveJavaIndexedCandidates(cc CallContext, request *javaSuggestionRequest, query suggestions.Query, allow bool) []suggestions.Candidate {
	if (request.mode != "codanna" && request.mode != suggestionModeCombined) || !allow {
		return nil
	}
	ctx, cancel := context.WithTimeout(request.ctx, 500*time.Millisecond)
	defer cancel()
	provider := cc.SuggestionProvider
	if provider == nil {
		provider = suggestions.Codanna{Binary: os.Getenv("SEMEDIT_CODANNA_BIN"), Dir: request.project.RootDir, Timeout: 500 * time.Millisecond}
	}
	values, err := provider.Retrieve(ctx, query)
	if err != nil {
		return nil
	}
	result := make([]suggestions.Candidate, 0, len(values))
	for _, value := range values {
		if suggestionCandidateInSelectedFile(request.project.RootDir, request.project.File, value.File) && javaOwnerMatches(request.query, value.Selector) {
			result = append(result, value)
		}
	}
	return result
}

func acceptJavaSuggestionCandidates(request *javaSuggestionRequest, operation string, ranked []suggestions.Candidate, currentSelectors map[string]string) []TargetSuggestion {
	accepted := make([]TargetSuggestion, 0, 3)
	seen := make(map[string]bool)
	for _, candidate := range ranked {
		if request.ctx.Err() != nil || len(accepted) == 3 {
			break
		}
		validation := candidate
		if current := currentSelectors[candidate.Identity+"\x00"+candidate.Selector]; current != "" {
			validation.Selector = current
		}
		value, identity, valid := validateJavaSuggestion(request.ctx, request.service, request.project, operation, request.renameTo, validation)
		if !valid || seen[identity] {
			continue
		}
		seen[identity] = true
		value.Arguments = map[string]string{request.selectorKey: validation.Selector, "file": displayWorkPath(request.project.RootDir, request.project.File)}
		if validation.Selector != candidate.Selector {
			value.Reason = fmt.Sprintf("Matched former name %s; now %s after an earlier retained rename.", candidate.Selector, validation.Selector)
		}
		accepted = append(accepted, value)
	}
	return accepted
}

func javaSuggestionOperation(key string) bool {
	switch key {
	case string(backend.OperationLookup), string(backend.OperationInspect), string(backend.OperationFindReferences), string(backend.OperationRename):
		return true
	default:
		return false
	}
}

func flattenJavaSuggestionSymbols(ctx context.Context, symbols []backend.ReadSymbol, candidates *[]suggestions.Candidate) bool {
	for _, symbol := range symbols {
		if err := ctx.Err(); err != nil || len(*candidates) >= maxJavaSuggestionCandidates {
			return false
		}
		file := symbol.File
		if symbol.QualifiedName != "" && symbol.Kind != "package" {
			identity := fmt.Sprintf("%s:%d:%d", file, symbol.SelectionRange.Start.Line, symbol.SelectionRange.Start.Character)
			*candidates = append(*candidates, suggestions.Candidate{Selector: symbol.QualifiedName, Name: symbol.Name, File: file, Kind: symbol.Kind, Signature: javaSuggestionSignature(symbol), Identity: identity})
		}
		if len(*candidates) > maxJavaSuggestionCandidates || !flattenJavaSuggestionSymbols(ctx, symbol.Children, candidates) {
			return false
		}
	}
	return true
}

func javaSuggestionAliases(ctx context.Context) []sourcebuf.SymbolAlias {
	aliases := suggestionHistory(ctx).Renames
	filtered := aliases[:0]
	for _, alias := range aliases {
		if alias.Language == "java" {
			filtered = append(filtered, alias)
		}
	}
	return filtered
}

func javaFormerSelectorEvidence(root string, candidates []suggestions.Candidate, aliases []sourcebuf.SymbolAlias, current map[string]string) []suggestions.Candidate {
	result := append([]suggestions.Candidate(nil), candidates...)
	for _, candidate := range candidates {
		selector, file := candidate.Selector, candidate.File
		for _, alias := range slices.Backward(aliases) {
			newFile := alias.NewFile
			if newFile == "" {
				newFile = alias.File
			}
			if !sameSuggestionFile(root, file, newFile) {
				continue
			}
			oldSelector, matched := reverseJavaAlias(selector, alias)
			if !matched {
				continue
			}
			former := candidate
			former.Selector, former.Name = oldSelector, oldSelector
			current[former.Identity+"\x00"+oldSelector] = candidate.Selector
			result = append(result, former)
			selector, file = oldSelector, alias.File
		}
	}
	return result
}

func reverseJavaAlias(selector string, alias sourcebuf.SymbolAlias) (string, bool) {
	if selector == alias.NewSelector {
		return alias.OldSelector, true
	}
	if alias.Kind != "class" && alias.Kind != "interface" && alias.Kind != "record" && alias.Kind != "enum" {
		return "", false
	}
	prefix := alias.NewSelector + "."
	if strings.HasPrefix(selector, prefix) {
		return alias.OldSelector + strings.TrimPrefix(selector, alias.NewSelector), true
	}
	return "", false
}

func exactJavaAmbiguousCandidates(lookup *backend.LookupResult, candidates []suggestions.Candidate) []suggestions.Candidate {
	matched := make(map[string]bool, len(lookup.Candidates))
	for _, candidate := range lookup.Candidates {
		matched[candidate.QualifiedName] = true
	}
	if lookup.Definition != nil {
		matched[lookup.Definition.QualifiedName] = true
	}
	result := make([]suggestions.Candidate, 0)
	for _, candidate := range candidates {
		if matched[candidate.Selector] {
			result = append(result, candidate)
		}
	}
	return result
}

func javaSuggestionSignature(symbol backend.ReadSymbol) string {
	if symbol.Signature != nil && *symbol.Signature != "" {
		return *symbol.Signature
	}
	if symbol.ServerDetail != nil {
		return *symbol.ServerDetail
	}
	return ""
}

func suggestionCandidateInSelectedFile(root, selected, candidate string) bool {
	if strings.TrimSpace(candidate) == "" {
		return false
	}
	selectedPath, selectedErr := filepath.Abs(resolveWorkPath(root, selected))
	candidatePath, candidateErr := filepath.Abs(resolveWorkPath(root, candidate))
	if selectedErr != nil || candidateErr != nil {
		return false
	}
	selectedPath, selectedErr = filepath.EvalSymlinks(selectedPath)
	candidatePath, candidateErr = filepath.EvalSymlinks(candidatePath)
	rootPath, rootErr := filepath.Abs(root)
	if rootErr == nil {
		rootPath, rootErr = filepath.EvalSymlinks(rootPath)
	}
	if selectedErr != nil || candidateErr != nil || rootErr != nil || filepath.Clean(selectedPath) != filepath.Clean(candidatePath) {
		return false
	}
	relative, err := filepath.Rel(rootPath, selectedPath)
	return err == nil && filepath.IsLocal(relative)
}

func constrainJavaOwner(query string, candidates []suggestions.Candidate) []suggestions.Candidate {
	result := make([]suggestions.Candidate, 0, len(candidates))
	for _, candidate := range candidates {
		if javaOwnerMatches(query, candidate.Selector) {
			result = append(result, candidate)
		}
	}
	return result
}

func javaOwnerMatches(query, selector string) bool {
	query, selector = javaSelectorBase(query), javaSelectorBase(selector)
	queryDot, selectorDot := strings.LastIndex(query, "."), strings.LastIndex(selector, ".")
	if queryDot < 0 {
		return true
	}
	if selectorDot < 0 {
		return false
	}
	queryOwner, selectorOwner := query[:queryDot], selector[:selectorDot]
	return queryOwner == selectorOwner || strings.HasSuffix(queryOwner, "."+selectorOwner) || strings.HasSuffix(selectorOwner, "."+queryOwner)
}

func javaSelectorBase(selector string) string {
	if base, _, ok := strings.Cut(selector, "("); ok {
		return base
	}
	return selector
}

func validateJavaSuggestion(ctx context.Context, service *backend.Service, project backend.ProjectContext, operation, renameTo string, candidate suggestions.Candidate) (TargetSuggestion, string, bool) {
	if !suggestionCandidateInSelectedFile(project.RootDir, project.File, candidate.File) || strings.TrimSpace(candidate.Selector) == "" {
		return TargetSuggestion{}, "", false
	}
	if operation == string(backend.OperationRename) {
		if renameTo == "" || service.ValidateRenameTarget(ctx, backend.RenameRequest{Project: project, Symbol: candidate.Selector, To: renameTo}) != nil {
			return TargetSuggestion{}, "", false
		}
	} else {
		lookup, err := service.Lookup(ctx, project, candidate.Selector)
		if err != nil || lookup == nil || lookup.Definition == nil || len(lookup.Candidates) != 0 {
			return TargetSuggestion{}, "", false
		}
	}
	inspection, err := service.Inspect(ctx, backend.InspectRequest{Project: project, Symbol: candidate.Selector})
	if err != nil || inspection == nil || len(inspection.Matches) != 1 {
		return TargetSuggestion{}, "", false
	}
	match := inspection.Matches[0]
	if !suggestionCandidateInSelectedFile(project.RootDir, project.File, match.File) || match.Kind != candidate.Kind {
		return TargetSuggestion{}, "", false
	}
	if operation == string(backend.OperationFindReferences) {
		references, refErr := service.FindReferences(ctx, backend.ReferencesRequest{Project: project, Symbol: candidate.Selector})
		if refErr != nil || references == nil || references.Symbol.QualifiedName != match.QualifiedName {
			return TargetSuggestion{}, "", false
		}
	}
	line, column := match.SelectionRange.Start.Line+1, match.SelectionRange.Start.Character+1
	signature := compactSuggestionSignature(match.Kind, candidate.Selector, strings.TrimSpace(javaSuggestionSignature(match.ReadSymbol)))
	if signature == "" {
		signature = match.Kind + " " + candidate.Selector
	}
	reason := "Similar spelling; native Java lookup and inspection checks passed."
	return TargetSuggestion{File: match.File, Line: line, Column: column, Kind: match.Kind, Signature: signature, Reason: reason}, fmt.Sprintf("%s:%d:%d", match.File, line, column), true
}
