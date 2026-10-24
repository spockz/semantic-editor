// Package operation enriches failed native target checks without executing candidate edits.
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

	"github.com/spockz/semantic-editor/internal/astedit"
	"github.com/spockz/semantic-editor/internal/backend"
	"github.com/spockz/semantic-editor/internal/backend/pathutil"
	"github.com/spockz/semantic-editor/internal/backends"
	"github.com/spockz/semantic-editor/internal/godiagnostics"
	"github.com/spockz/semantic-editor/internal/sourcebuf"
	"github.com/spockz/semantic-editor/internal/suggestions"
	"github.com/spockz/semantic-editor/internal/symbol"
)

const suggestionModeCombined = "codanna,generic"

// TargetSuggestion contains only the argument replacements and native target evidence needed for a retry.
type TargetSuggestion struct {
	Arguments map[string]string `json:"arguments"`
	File      string            `json:"file"`
	Line      int               `json:"line"`
	Column    int               `json:"column"`
	Kind      string            `json:"kind"`
	Signature string            `json:"signature,omitempty"`
	Reason    string            `json:"reason"`
}

// SuggestedTargetError retains the native error chain and its execution semantics.
type SuggestedTargetError struct {
	Cause       error
	Suggestions []TargetSuggestion
}

type suggestionHistoryKey struct{}
type suggestionProviderKey struct{}

// WithSuggestionHistory carries retained identities across a batch publication barrier.
func WithSuggestionHistory(ctx context.Context, history sourcebuf.SymbolHistory) context.Context {
	history.Renames = slices.Clone(history.Renames)
	history.Invalidated = slices.Clone(history.Invalidated)
	return context.WithValue(ctx, suggestionHistoryKey{}, history)
}

func suggestionHistory(ctx context.Context) sourcebuf.SymbolHistory {
	if buffer := sourcebuf.FromContext(ctx); buffer != nil {
		return buffer.SymbolHistory()
	}
	history, _ := ctx.Value(suggestionHistoryKey{}).(sourcebuf.SymbolHistory)
	return history
}

func (e *SuggestedTargetError) Error() string {
	var text strings.Builder
	text.WriteString(e.Cause.Error())
	for _, suggestion := range e.Suggestions {
		selector := suggestionSelector(suggestion.Arguments)
		fmt.Fprintf(&text, "\nSuggested target: %s (%s:%d)\n  %s\n  %s", selector, suggestion.File, suggestion.Line, suggestion.Signature, suggestion.Reason)
	}
	return text.String()
}

func suggestionSelector(arguments map[string]string) string {
	for _, key := range []string{"symbol", wireInFunction, "in_type", wireInDeclaration, "target_symbol"} {
		if selector := arguments[key]; selector != "" {
			return selector
		}
	}
	return ""
}

func (e *SuggestedTargetError) Unwrap() error { return e.Cause }

// SuggestionsFromError preserves structured alternatives across wrapped ingress errors.
func SuggestionsFromError(err error) []TargetSuggestion {
	if enriched, ok := errors.AsType[*SuggestedTargetError](err); ok {
		return enriched.Suggestions
	}
	return nil
}

// NativeTargetError removes alternatives from an immediate dispatch failure when its source view is discarded.
func NativeTargetError(err error) error {
	if enriched, ok := errors.AsType[*SuggestedTargetError](err); ok {
		return enriched.Cause
	}
	return err
}

func suggestFailedTarget(cc CallContext, entry Entry, raw map[string]any, cause error) error {
	if !errors.Is(cause, symbol.ErrNotFound) && !errors.Is(cause, backend.ErrAmbiguous) && !errors.Is(cause, symbol.ErrAmbiguous) {
		return suggestJavaFailedTarget(cc, entry, raw, cause)
	}
	target := prepareSuggestionTarget(cc, entry, raw, cause)
	if target == nil {
		return suggestJavaFailedTarget(cc, entry, raw, cause)
	}
	ctx := cc.Ctx
	if ctx == nil {
		ctx = context.Background()
	}
	budget := 1500 * time.Millisecond
	if key := backendOperationKey(entry.Key); entry.TargetPreflight != nil || key == string(backend.OperationRename) || key == string(backend.OperationFindReferences) || key == string(backend.OperationExtractFunction) || key == string(backend.OperationExtractVariable) {
		budget = 5 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, budget)
	defer cancel()
	if cc.SuggestionProvider != nil {
		ctx = context.WithValue(ctx, suggestionProviderKey{}, cc.SuggestionProvider)
	}
	ctx, release := godiagnostics.EnsureSessionManager(ctx)
	cc.Ctx = ctx
	var accepted []TargetSuggestion
	if err := target.validate(ctx, cc, target.project, target.query); err != nil {
		candidates := failedTargetCandidates(ctx, target, cause)
		accepted = acceptTargetSuggestions(ctx, cc, target, candidates)
	}
	if closeErr := release(); closeErr != nil {
		return cause
	}
	if len(accepted) == 0 {
		return cause
	}
	return &SuggestedTargetError{Cause: cause, Suggestions: accepted}
}

type suggestionTarget struct {
	project                  backend.ProjectContext
	query, selectorKey, mode string
	validate                 nativeTargetValidator
	retrieval                suggestions.Query
}

func prepareSuggestionTarget(cc CallContext, entry Entry, raw map[string]any, cause error) *suggestionTarget {
	raw, err := entry.NormalizedInput(raw)
	if err != nil {
		return nil
	}
	request, err := entry.Parse(raw)
	if err != nil {
		return nil
	}
	project := effectiveProject(cc, projectOf(request))
	if _, mapRequest := request.(map[string]any); mapRequest {
		selected, _ := raw["file"].(string)
		parsedProject, projectErr := parseReadProject(raw, selected)
		if projectErr != nil {
			return nil
		}
		project = effectiveProject(cc, parsedProject)
	}
	project.RootDir, err = filepath.Abs(project.RootDir)
	if err != nil {
		return nil
	}
	language, err := resolveLanguage(cc, project)
	if err != nil || language != backend.LanguageGo {
		return nil
	}
	selectorKey, query := failedSourceSelector(raw, cause)
	key := backendOperationKey(entry.Key)
	validate := targetValidator(key, raw, selectorKey)
	if entry.TargetPreflight != nil {
		validate = func(ctx context.Context, call CallContext, project backend.ProjectContext, selector string) error {
			retry := cloneRaw(raw)
			retry[selectorKey], retry["file"] = selector, project.File
			call.WorkDir, call.Project, call.Ctx = project.RootDir, project, ctx
			return entry.TargetPreflight(ctx, call, retry)
		}
	}
	mode := os.Getenv("SEMEDIT_SUGGESTIONS")
	if validate == nil || selectorKey == "" || query == "" || len(query) > 256 || mode == "off" {
		return nil
	}
	return &suggestionTarget{project: project, query: query, selectorKey: selectorKey, mode: mode, validate: validate, retrieval: suggestionRetrievalContext(key, selectorKey, query, raw)}
}

func suggestionRetrievalContext(key, selectorKey, selector string, raw map[string]any) suggestions.Query {
	query := suggestions.Query{Operation: key, TargetParameter: selectorKey}
	query.Signature = suggestionSignatureConstraint(raw)
	query.Receiver, _, _ = symbol.ParseIdentifier(selector)
	switch key {
	case string(backend.OperationReplaceBody), string(backend.OperationExtractFunction), string(backend.OperationExtractVariable):
		query.Kinds = []string{"function", "method"}
		query.Receiver, _, _ = symbol.ParseIdentifier(selector)
	case string(backend.OperationRemoveConstruct), string(backend.OperationReplaceConstruct):
		kind, _ := raw["kind"].(string)
		query.Kinds = []string{kind}
		if selectorKey == wireInFunction {
			query.Kinds = []string{"function", "method"}
		}
	case string(backend.OperationChangeSignature):
		query.Kinds = []string{"function"}
	}
	return query
}

func failedSourceSelector(raw map[string]any, cause error) (string, string) {
	selectorKey := "symbol"
	query, _ := raw["symbol"].(string)
	if targetErr, ok := errors.AsType[*symbol.SymbolError](cause); ok {
		query = targetErr.Symbol
		selectorKey = ""
		for _, key := range []string{wireInFunction, "in_type", wireInDeclaration, "symbol"} {
			if raw[key] == query {
				selectorKey = key
				break
			}
		}
	}
	return selectorKey, query
}

func failedTargetCandidates(ctx context.Context, target *suggestionTarget, cause error) []suggestions.Candidate {
	if errors.Is(cause, backend.ErrAmbiguous) || errors.Is(cause, symbol.ErrAmbiguous) {
		snapshot, resolveErr := symbol.ResolveSnapshotWithContext(ctx, target.project.RootDir, target.project.File, target.query)
		if resolveErr != nil || snapshot.Result == nil {
			return nil
		}
		var candidates []suggestions.Candidate
		for _, candidate := range snapshot.Result.Candidates {
			match := suggestionCandidate(candidate)
			match.Name = match.Selector
			candidates = append(candidates, match)
		}
		return candidates
	}
	return retrieveSuggestionCandidates(ctx, target.project, target.query, target.mode, target.retrieval)
}

func acceptTargetSuggestions(ctx context.Context, cc CallContext, target *suggestionTarget, candidates []suggestions.Candidate) []TargetSuggestion {
	accepted := make([]TargetSuggestion, 0, 3)
	seen := make(map[string]bool)
	for _, candidate := range candidates {
		if ctx.Err() != nil {
			break
		}
		value, key, valid := validateTargetSuggestion(ctx, cc, target, candidate)
		if !valid || seen[key] {
			continue
		}
		seen[key] = true
		accepted = append(accepted, value)
		if len(accepted) == 3 {
			break
		}
	}
	return accepted
}

func validateTargetSuggestion(ctx context.Context, cc CallContext, target *suggestionTarget, candidate suggestions.Candidate) (TargetSuggestion, string, bool) {
	project := target.project
	if !candidateWithinScope(project, target.query, candidate) {
		return TargetSuggestion{}, "", false
	}
	canonical, err := sourcebuf.CanonicalPath(ctx, resolveWorkPath(project.RootDir, candidate.File))
	root, rootErr := filepath.EvalSymlinks(project.RootDir)
	relative, relativeErr := filepath.Rel(root, canonical)
	if err != nil || rootErr != nil || relativeErr != nil || !filepath.IsLocal(relative) {
		return TargetSuggestion{}, "", false
	}
	project.File = candidate.File
	resolved, err := symbol.ResolveSnapshotWithContext(ctx, project.RootDir, candidate.File, candidate.Selector)
	if err != nil || resolved.Result.IsAmbiguous() || resolved.Result.Definition == nil {
		return TargetSuggestion{}, "", false
	}
	identity := resolved.Result.Definition
	if targetErr := target.validate(ctx, cc, project, candidate.Selector); targetErr != nil {
		return TargetSuggestion{}, "", false
	}
	inspection, err := astedit.InspectGoWithContext(ctx, project.RootDir, candidate.File, candidate.Selector)
	if err != nil || len(inspection.Matches) != 1 {
		return TargetSuggestion{}, "", false
	}
	match := inspection.Matches[0]
	preimage, present := resolved.Files[filepath.Clean(resolveWorkPath(project.RootDir, identity.File))]
	if !present || structuralRevision(preimage) != match.Revision {
		return TargetSuggestion{}, "", false
	}
	signature := ""
	if match.Signature != nil {
		signature = compactSuggestionSignature(match.Kind, candidate.Selector, *match.Signature)
	}
	reason := "Similar spelling; native target checks passed."
	_, requestedName, requestedErr := symbol.ParseIdentifier(target.query)
	if requestedErr == nil && requestedName == identity.Name {
		reason = "Exact matching declaration; qualified by its location."
	}
	if candidate.Name != "" && candidate.Name != candidate.Selector {
		reason = fmt.Sprintf("Matched former name %s; now %s after an earlier retained rename.", candidate.Name, candidate.Selector)
	}
	return TargetSuggestion{Arguments: map[string]string{target.selectorKey: candidate.Selector, "file": displayWorkPath(project.RootDir, candidate.File)}, File: identity.File, Line: identity.Line, Column: identity.Column, Kind: match.Kind, Signature: signature, Reason: reason}, fmt.Sprintf("%s:%d", identity.File, identity.Offset), true
}

type nativeTargetValidator func(context.Context, CallContext, backend.ProjectContext, string) error

func targetValidator(key string, raw map[string]any, selectorKey string) nativeTargetValidator {
	switch key {
	case string(backend.OperationLookup), string(backend.OperationInspect):
		return func(ctx context.Context, _ CallContext, project backend.ProjectContext, selector string) error {
			result, err := symbol.ResolveSnapshotWithContext(ctx, project.RootDir, project.File, selector)
			if err != nil {
				return err
			}
			if result.Result.IsAmbiguous() || result.Result.Definition == nil {
				return backend.ErrAmbiguous
			}
			return nil
		}
	case string(backend.OperationRename):
		return func(ctx context.Context, cc CallContext, project backend.ProjectContext, selector string) error {
			to, _ := raw["to"].(string)
			service := cc.Service
			if service == nil {
				service = backends.NewDefaultService()
			}
			return service.ValidateRenameTarget(ctx, backend.RenameRequest{Project: project, Symbol: selector, To: to})
		}
	case string(backend.OperationFindReferences):
		return func(ctx context.Context, cc CallContext, project backend.ProjectContext, selector string) error {
			_, err := referencesRun(ctx, cc, ReferencesReq{Project: project, Symbol: selector})
			return err
		}
	case string(backend.OperationReplaceBody):
		return func(ctx context.Context, _ CallContext, project backend.ProjectContext, selector string) error {
			return astedit.ValidateBodyTarget(ctx, resolveWorkPath(project.RootDir, project.File), selector)
		}
	case string(backend.OperationRemoveConstruct):
		return func(ctx context.Context, _ CallContext, project backend.ProjectContext, selector string) error {
			retry := cloneRaw(raw)
			retry[selectorKey] = selector
			request, err := parseRemove(retry)
			if err != nil {
				return err
			}
			return astedit.ValidateRemoveTarget(ctx, resolveWorkPath(project.RootDir, project.File), astedit.RemoveOptions{Kind: string(request.Kind), Symbol: request.Symbol, InFunction: request.InFunction, Discriminator: request.Discriminator, TargetCase: request.TargetCase, ConstructPath: request.ConstructPath, SafeDelete: request.SafeDelete})
		}
	case string(backend.OperationReplaceConstruct):
		return func(ctx context.Context, _ CallContext, project backend.ProjectContext, selector string) error {
			retry := cloneRaw(raw)
			retry[selectorKey] = selector
			request, err := parseReplaceConstruct(retry)
			if err != nil {
				return err
			}
			return astedit.ValidateConstructReplacementTarget(ctx, resolveWorkPath(project.RootDir, project.File), request.InFunction, request.Kind, request.Discriminator, request.Source, astedit.ConstructOptions{Symbol: request.Symbol, InFunction: request.InFunction, Discriminator: request.Discriminator, ConstructPath: request.ConstructPath, ExpectedSelectionRevision: request.ExpectedSelectionRevision})
		}
	case string(backend.OperationChangeSignature):
		return func(ctx context.Context, cc CallContext, project backend.ProjectContext, selector string) error {
			cc.WorkDir = project.RootDir
			snapshot, err := prepareStructuralSnapshot(ctx, cc, project.File)
			if err != nil {
				return err
			}
			retry := cloneRaw(raw)
			retry[selectorKey] = selector
			intent, err := parseSignatureChangeIntent(retry, snapshot.Sources[snapshot.File])
			if err != nil {
				return err
			}
			plan, err := planSignatureChange(snapshot, intent)
			if err != nil {
				return err
			}
			return validateSignatureOverlay(snapshot, plan)
		}
	}
	return extractionTargetValidator(key, raw, selectorKey)
}

func suggestionCandidate(value *symbol.Symbol) suggestions.Candidate {
	return suggestions.Candidate{Selector: value.BuildQualifiedName(), File: value.File, Name: value.Name, Kind: value.Kind, Identity: fmt.Sprintf("%s:%d", value.File, value.Offset)}
}

func compactSuggestionSignature(kind, selector, signature string) string {
	if kind == "type" || kind == "interface" || kind == "class" || kind == "record" || kind == "enum" || kind == "value" || kind == "constant" || kind == "variable" {
		return kind + " " + selector
	}
	compact := []rune(strings.Join(strings.Fields(signature), " "))
	if len(compact) > 240 {
		return string(compact[:237]) + "..."
	}
	return string(compact)
}

func retrieveSuggestionCandidates(ctx context.Context, project backend.ProjectContext, query, mode string, request suggestions.Query) []suggestions.Candidate {
	values, err := symbol.Enumerate(ctx, project.RootDir, project.File)
	if err != nil {
		values = nil
	}
	candidates, current := nativeSuggestionEvidence(ctx, project, values)
	candidates = qualifiedSuggestionEvidence(query, candidates)
	request.Selector, request.Candidates, request.Limit, request.Language, request.Scope = query, candidates, 50, "go", project.File
	var retrieved []suggestions.Candidate
	if mode == "codanna" || mode == suggestionModeCombined {
		retrieved = indexedSuggestionEvidence(ctx, project, request)
	}
	if mode == "" || mode == "generic" || mode == suggestionModeCombined {
		retrieved = append(retrieved, candidates...)
	}
	request.Candidates = retrieved
	result, err := (suggestions.Matcher{}).Retrieve(ctx, request)
	if err != nil {
		return nil
	}
	for i := range result {
		if selector := current[result[i].Identity+"\x00"+result[i].Selector]; selector != "" {
			result[i].Selector = selector
		}
	}
	return result
}

func nativeSuggestionEvidence(ctx context.Context, project backend.ProjectContext, values []*symbol.Symbol) ([]suggestions.Candidate, map[string]string) {
	candidates := make([]suggestions.Candidate, 0, len(values))
	current := make(map[string]string)
	aliases := suggestionHistory(ctx).Renames
	for _, value := range values {
		candidate := suggestionCandidate(value)
		candidate.Name = candidate.Selector
		candidates = append(candidates, candidate)
		former := candidate.Selector
		for _, alias := range slices.Backward(aliases) {
			if alias.Language != "" && alias.Language != "go" {
				continue
			}
			if sameSuggestionFile(project.RootDir, candidate.File, alias.File) && former == alias.NewSelector {
				former = alias.OldSelector
			} else if rebased, ok := formerTypeSelector(former, alias); alias.Kind == "type" && filepath.Dir(resolveWorkPath(project.RootDir, candidate.File)) == filepath.Dir(resolveWorkPath(project.RootDir, alias.File)) && ok {
				former = rebased
			} else {
				continue
			}
			evidence := candidate
			evidence.Selector, evidence.Name = former, former
			current[evidence.Identity+"\x00"+former] = candidate.Selector
			candidates = append(candidates, evidence)
		}
	}
	return candidates, current
}

func qualifiedSuggestionEvidence(query string, candidates []suggestions.Candidate) []suggestions.Candidate {
	if separator := strings.LastIndex(query, "."); separator >= 0 {
		prefix := query[:separator+1]
		for _, candidate := range slices.Clone(candidates) {
			if !strings.Contains(candidate.Selector, ".") {
				qualified := candidate
				qualified.Selector = prefix + candidate.Selector
				qualified.Name = qualified.Selector
				candidates = append(candidates, qualified)
			}
		}
	}
	return candidates
}

func indexedSuggestionEvidence(ctx context.Context, project backend.ProjectContext, request suggestions.Query) []suggestions.Candidate {
	providerCtx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
	provider, _ := ctx.Value(suggestionProviderKey{}).(suggestions.Provider)
	if provider == nil {
		provider = suggestions.Codanna{Binary: os.Getenv("SEMEDIT_CODANNA_BIN"), Dir: project.RootDir, Timeout: 500 * time.Millisecond}
	}
	indexed, providerErr := provider.Retrieve(providerCtx, request)
	cancel()
	if providerErr != nil {
		return nil
	}
	if len(indexed) > 50 {
		indexed = indexed[:50]
	}
	history := suggestionHistory(ctx)
	aliases := history.Renames
	var retrieved []suggestions.Candidate
	for _, indexedCandidate := range indexed {
		if indexedCandidateInvalidated(project.RootDir, indexedCandidate, history.Invalidated) {
			continue
		}
		matches := make(map[string]suggestions.Candidate)
		for _, native := range request.Candidates {
			if indexedCandidate.Selector == native.Selector && sameSuggestionFile(project.RootDir, indexedCandidate.File, native.File) {
				matches[native.Identity] = native
			}
		}
		if len(matches) == 1 {
			for _, native := range matches {
				retrieved = append(retrieved, native)
			}
		} else if len(matches) == 0 && len(aliases) == 0 {
			indexedCandidate.Name = indexedCandidate.Selector
			retrieved = append(retrieved, indexedCandidate)
		}
	}
	return retrieved
}

func formerTypeSelector(selector string, alias sourcebuf.SymbolAlias) (string, bool) {
	receiver, name, err := symbol.ParseIdentifier(selector)
	if err != nil || receiver != alias.NewSelector {
		return "", false
	}
	return alias.OldSelector + "." + name, true
}

func sameSuggestionFile(root, left, right string) bool {
	return filepath.Clean(resolveWorkPath(root, left)) == filepath.Clean(resolveWorkPath(root, right))
}

func recordSuccessfulTargetChange(ctx context.Context, key string, raw map[string]any, result any) {
	key = backendOperationKey(key)
	buffer := sourcebuf.FromContext(ctx)
	if buffer == nil {
		return
	}
	if key != string(backend.OperationRename) {
		InvalidateSuggestionAliases(buffer, key, raw)
		return
	}
	renamed, ok := result.(*backend.RenameResult)
	if !ok || renamed == nil || renamed.Source == nil || renamed.Source.Definition == nil {
		return
	}
	origin := renamed.Source.Definition
	selector := origin.Identity()
	file, err := pathutil.FilePathFromURI(origin.Location.URI)
	if err != nil {
		return
	}
	to, _ := raw["to"].(string)
	to = backend.NormalizeRenameInput(to)
	if filepath.Ext(file) == ".java" {
		recordJavaRenameAlias(buffer, file, selector, to, origin.Kind, renamed.TargetFile)
		return
	}
	if filepath.Ext(file) != ".go" {
		return
	}
	_, oldName, err := symbol.ParseIdentifier(selector)
	if err != nil {
		return
	}
	if to == "" || to == oldName {
		return
	}
	newSelector := strings.TrimSuffix(selector, oldName) + to
	buffer.RecordSymbolRename(sourcebuf.SymbolAlias{File: file, OldSelector: selector, NewSelector: newSelector, Kind: origin.Kind})
}

// InvalidateSuggestionAliases drops retained identity evidence after a declaration is removed or replaced.
func InvalidateSuggestionAliases(buffer *sourcebuf.Buffer, key string, raw map[string]any) {
	switch key {
	case string(backend.OperationRemoveConstruct), string(backend.OperationReplaceDecl), string(backend.OperationReplaceConstruct), string(backend.OperationMoveConstruct):
		file, _ := raw["file"].(string)
		selector, _ := raw["symbol"].(string)
		if selector != "" {
			buffer.InvalidateSymbolAliases(file, selector)
		}
	}
}

func candidateWithinScope(project backend.ProjectContext, query string, candidate suggestions.Candidate) bool {
	path := resolveWorkPath(project.RootDir, candidate.File)
	if filepath.Ext(path) != ".go" {
		return false
	}
	relative, err := filepath.Rel(project.RootDir, path)
	if err != nil || !filepath.IsLocal(relative) {
		return false
	}
	receiver, _, err := symbol.ParseIdentifier(query)
	if err != nil {
		return false
	}
	if receiver != "" {
		candidateReceiver, _, parseErr := symbol.ParseIdentifier(candidate.Selector)
		if parseErr != nil || receiver != candidateReceiver {
			formerReceiver, _, formerErr := symbol.ParseIdentifier(candidate.Name)
			if formerErr != nil || receiver != formerReceiver {
				return false
			}
		}
	}
	if project.File == "" {
		return true
	}
	selected := resolveWorkPath(project.RootDir, project.File)
	if filepath.Clean(path) == filepath.Clean(selected) {
		return true
	}
	info, err := os.Stat(selected)
	if err != nil || !info.IsDir() {
		return false
	}
	return filepath.Dir(path) == filepath.Clean(selected)
}

func suggestionSignatureConstraint(raw map[string]any) string {
	var constraints []string
	if typeName, ok := raw["type"].(string); ok && typeName != "" {
		constraints = append(constraints, typeName)
	}
	parameters, _ := raw["parameters"].([]any)
	for _, value := range parameters {
		mapping, _ := value.(map[string]any)
		parameter, _ := mapping["parameter"].(map[string]any)
		if typeName, ok := parameter["type"].(string); ok && typeName != "" {
			constraints = append(constraints, typeName)
		}
	}
	constraint := []rune(strings.Join(constraints, ", "))
	if len(constraint) > 256 {
		constraint = constraint[:256]
	}
	return string(constraint)
}
