// Package operation recovers selected-file candidates from native language outlines.
package operation

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spockz/semantic-editor/internal/backend"
	haskellbackend "github.com/spockz/semantic-editor/internal/backend/haskell"
	kotlinbackend "github.com/spockz/semantic-editor/internal/backend/kotlin"
	"github.com/spockz/semantic-editor/internal/backend/pathutil"
	rustbackend "github.com/spockz/semantic-editor/internal/backend/rust"
	scalabackend "github.com/spockz/semantic-editor/internal/backend/scala"
	"github.com/spockz/semantic-editor/internal/backend/textlsp"
	"github.com/spockz/semantic-editor/internal/backends"
	"github.com/spockz/semantic-editor/internal/suggestions"
)

const maxNativeSuggestionCandidates = 4096

type outlineSuggestionRequest struct {
	ctx         context.Context
	cancel      context.CancelFunc
	project     backend.ProjectContext
	query       string
	selectorKey string
	operation   string
	mode        string
	language    backend.LanguageID
	renameTo    string
	service     *backend.Service
}

func prepareOutlineSuggestionRequest(cc CallContext, entry Entry, raw map[string]any, language backend.LanguageID) (*outlineSuggestionRequest, bool) {
	mode := os.Getenv("SEMEDIT_SUGGESTIONS")
	if mode == "off" {
		return nil, false
	}
	normalized, err := entry.NormalizedInput(raw)
	if err != nil {
		return nil, false
	}
	parsed, err := entry.Parse(normalized)
	if err != nil {
		return nil, false
	}
	project := effectiveProject(cc, projectOf(parsed))
	resolvedLanguage, err := resolveLanguage(cc, project)
	if err != nil || (language != "" && resolvedLanguage != language) || strings.TrimSpace(project.File) == "" {
		return nil, false
	}
	project.Language = resolvedLanguage
	project.RootDir, err = filepath.Abs(project.RootDir)
	if err != nil {
		return nil, false
	}
	project.File, err = filepath.Abs(resolveWorkPath(project.RootDir, project.File))
	if err != nil {
		return nil, false
	}
	selectorKey, query := failedSourceSelector(normalized, nil)
	if selectorKey == "" || query == "" || len(query) > 256 {
		return nil, false
	}
	ctx := cc.Ctx
	if ctx == nil {
		ctx = context.Background()
	}
	budget := 1500 * time.Millisecond
	if entry.Key == string(backend.OperationFindReferences) || entry.Key == string(backend.OperationRename) {
		budget = 5 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, budget)
	service := cc.Service
	if service == nil {
		service = backends.NewDefaultService()
	}
	return &outlineSuggestionRequest{ctx: ctx, cancel: cancel, project: project, query: query, selectorKey: selectorKey, operation: entry.Key, mode: mode, language: resolvedLanguage, service: service}, true
}

// suggestNativeOutlineFailedTarget enriches only exact native misses for selected-file read operations.
func suggestNativeOutlineFailedTarget(cc CallContext, entry Entry, raw map[string]any, cause error) error {
	request, ok := prepareOutlineSuggestionRequest(cc, entry, raw, "")
	if !ok {
		return cause
	}
	defer request.cancel()
	language := request.language
	if !nativeOutlineFailureAllowed(language, entry.Key, cause) {
		return cause
	}
	native := make([]suggestions.Candidate, 0, min(maxNativeSuggestionCandidates, 128))
	if language == backend.LanguageHaskell {
		symbols, symbolErr := request.service.SymbolCandidates(request.ctx, request.project)
		if symbolErr != nil || !flattenOutlineCandidates(request.ctx, symbols, &native) {
			return cause
		}
	} else {
		outline, outlineErr := request.service.Outline(request.ctx, backend.OutlineRequest{Project: request.project, Path: request.project.File, IncludeUnexported: true})
		if outlineErr != nil || outline == nil || len(outline.Files) != 1 || !suggestionCandidateInSelectedFile(request.project.RootDir, request.project.File, outline.Files[0].File) || !flattenOutlineCandidates(request.ctx, outline.Files[0].Symbols, &native) {
			return cause
		}
	}
	if len(native) == 0 {
		return cause
	}
	for i := range native {
		native[i].Language = string(language)
	}
	query := suggestions.Query{Selector: request.query, Operation: request.operation, TargetParameter: request.selectorKey, Language: string(language), Scope: request.project.File, Limit: 50}
	var candidates []suggestions.Candidate
	if request.mode == "" || request.mode == "generic" || request.mode == suggestionModeCombined {
		candidates = append(candidates, native...)
	}
	if request.mode == "codanna" || request.mode == suggestionModeCombined {
		provider := cc.SuggestionProvider
		if provider == nil {
			provider = suggestions.Codanna{Binary: os.Getenv("SEMEDIT_CODANNA_BIN"), Dir: request.project.RootDir, Timeout: 500 * time.Millisecond}
		}
		providerCtx, cancel := context.WithTimeout(request.ctx, 500*time.Millisecond)
		indexed, providerErr := provider.Retrieve(providerCtx, query)
		cancel()
		if providerErr == nil {
			history := suggestionHistory(request.ctx)
			for _, candidate := range indexed {
				if suggestionCandidateInSelectedFile(request.project.RootDir, request.project.File, candidate.File) && !indexedCandidateInvalidated(request.project.RootDir, candidate, history.Invalidated) {
					candidates = append(candidates, candidate)
				}
			}
		}
	}
	query.Candidates = candidates
	ranked, err := (suggestions.Matcher{}).Retrieve(request.ctx, query)
	if err != nil || len(ranked) == 0 {
		return cause
	}
	accepted := acceptOutlineSuggestions(request, ranked)
	if len(accepted) == 0 {
		return cause
	}
	return &SuggestedTargetError{Cause: cause, Suggestions: accepted}
}

func nativeOutlineFailureAllowed(language backend.LanguageID, operation string, cause error) bool {
	if operation == string(backend.OperationInspect) {
		return language == backend.LanguageMake && errors.Is(cause, textlsp.ErrSymbolNotFound)
	}
	if operation != string(backend.OperationLookup) {
		return false
	}
	switch language {
	case backend.LanguageRust:
		return errors.Is(cause, rustbackend.ErrRustSymbolNotFound)
	case backend.LanguageHaskell:
		return operation == string(backend.OperationLookup) && errors.Is(cause, haskellbackend.ErrHaskellSymbolNotFound)
	case backend.LanguageScala:
		return errors.Is(cause, scalabackend.ErrScalaSymbolNotFound)
	case backend.LanguageKotlin:
		return errors.Is(cause, kotlinbackend.ErrKotlinSymbolNotFound)
	case backend.LanguageBash, backend.LanguageMake:
		return errors.Is(cause, textlsp.ErrSymbolNotFound)
	default:
		return false
	}
}

func flattenOutlineCandidates(ctx context.Context, symbols []backend.ReadSymbol, candidates *[]suggestions.Candidate) bool {
	for _, value := range symbols {
		if ctx.Err() != nil || len(*candidates) >= maxNativeSuggestionCandidates {
			return false
		}
		if value.QualifiedName != "" && value.Kind != "package" {
			identity := fmt.Sprintf("%s:%d:%d", value.File, value.SelectionRange.Start.Line, value.SelectionRange.Start.Character)
			*candidates = append(*candidates, suggestions.Candidate{Selector: value.QualifiedName, Name: value.Name, File: value.File, Kind: value.Kind, Signature: outlineSymbolSignature(value), Identity: identity})
		}
		if len(*candidates) > maxNativeSuggestionCandidates || !flattenOutlineCandidates(ctx, value.Children, candidates) {
			return false
		}
	}
	return true
}

func outlineSymbolSignature(value backend.ReadSymbol) string {
	if value.Signature != nil && *value.Signature != "" {
		return *value.Signature
	}
	if value.ServerDetail != nil {
		return *value.ServerDetail
	}
	return ""
}

func acceptOutlineSuggestions(request *outlineSuggestionRequest, ranked []suggestions.Candidate) []TargetSuggestion {
	accepted := make([]TargetSuggestion, 0, 3)
	seen := make(map[string]bool)
	for _, candidate := range ranked {
		if request.ctx.Err() != nil || len(accepted) == 3 {
			break
		}
		if !suggestionCandidateInSelectedFile(request.project.RootDir, request.project.File, candidate.File) {
			continue
		}
		lookup, err := request.service.Lookup(request.ctx, request.project, candidate.Selector)
		if err != nil || lookup == nil || lookup.Definition == nil || len(lookup.Candidates) != 0 {
			continue
		}
		definition := lookup.Definition
		file, err := pathutil.FilePathFromURI(definition.Location.URI)
		if err != nil || !suggestionCandidateInSelectedFile(request.project.RootDir, request.project.File, file) || definition.Identity() != candidate.Selector {
			continue
		}
		if request.operation == string(backend.OperationInspect) {
			inspection, inspectErr := request.service.Inspect(request.ctx, backend.InspectRequest{Project: request.project, Symbol: candidate.Selector})
			if inspectErr != nil || inspection == nil || len(inspection.Matches) != 1 || !suggestionCandidateInSelectedFile(request.project.RootDir, request.project.File, inspection.Matches[0].File) || inspection.Matches[0].QualifiedName != candidate.Selector || inspection.Matches[0].Kind != candidate.Kind {
				continue
			}
		}
		line, column := definition.Location.Range.Start.Line+1, definition.Location.Range.Start.Character+1
		identity := fmt.Sprintf("%s:%d:%d", file, line, column)
		if seen[identity] {
			continue
		}
		seen[identity] = true
		selector := definition.Identity()
		reason := "Similar spelling; native selected-file outline and lookup checks passed."
		if request.operation == string(backend.OperationInspect) {
			reason = "Similar spelling; native selected-file outline, lookup, and inspection checks passed."
		}
		accepted = append(accepted, TargetSuggestion{Arguments: map[string]string{request.selectorKey: selector, "file": displayWorkPath(request.project.RootDir, request.project.File)}, File: displayWorkPath(request.project.RootDir, file), Line: line, Column: column, Kind: definition.Kind, Signature: compactSuggestionSignature(definition.Kind, selector, candidate.Signature), Reason: reason})
	}
	return accepted
}
