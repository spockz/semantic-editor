// Package operation tests Java candidate retry validation through neutral backend APIs.
package operation

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/spockz/semantic-editor/internal/backend"
	javabackend "github.com/spockz/semantic-editor/internal/backend/java"
	"github.com/spockz/semantic-editor/internal/sourcebuf"
	"github.com/spockz/semantic-editor/internal/suggestions"
)

func TestSuggestJavaFailedTargetUsesOutlineAndNativeRetry(t *testing.T) {
	t.Setenv("SEMEDIT_SUGGESTIONS", "generic")
	root := javaSuggestionRoot(t, "class Gadget {}\n")
	file := filepath.Join(root, "Widget.java")
	fake := &javaSuggestionBackend{file: file, selector: "demo.Widget"}
	service := javaSuggestionService(t, fake)
	cc := NewCallContext(root, backend.ProjectContext{RootDir: root, Language: backend.LanguageJava, WorkspaceTrust: backend.NewWorkspaceTrust(root, true)})
	cc.Service = service
	entry, ok := DefaultRegistry().LookupKey(string(backend.OperationLookup))
	if !ok {
		t.Fatal("lookup entry missing")
	}
	raw := map[string]any{"file": "Widget.java", "language": "java", "trust_workspace": true, "symbol": "Widgte"}
	cause := &javabackend.JavaError{Op: "lookup", File: file, Symbol: "Widgte", Err: javabackend.ErrJavaSymbolNotFound}
	err := suggestJavaFailedTarget(cc, entry, raw, cause)
	if !errors.Is(err, javabackend.ErrJavaSymbolNotFound) {
		t.Fatalf("failure identity = %v", err)
	}
	suggestions := SuggestionsFromError(err)
	if len(suggestions) != 1 || suggestions[0].Arguments["symbol"] != "demo.Widget" || suggestions[0].Arguments["file"] != "Widget.java" {
		t.Fatalf("suggestions = %#v", suggestions)
	}
	if fake.outlineCalls != 1 || fake.lookupCalls != 1 || fake.inspectCalls != 1 {
		t.Fatalf("native calls outline=%d lookup=%d inspect=%d", fake.outlineCalls, fake.lookupCalls, fake.inspectCalls)
	}
	data, readErr := os.ReadFile(file)
	if readErr != nil || string(data) != "class Gadget {}\n" {
		t.Fatalf("source changed during suggestion: %q (%v)", data, readErr)
	}
}

func TestSuggestJavaExactFailureDoesNotEnumerate(t *testing.T) {
	root := javaSuggestionRoot(t, "class Widget {}\n")
	fake := &javaSuggestionBackend{file: filepath.Join(root, "Widget.java"), selector: "demo.Widget"}
	service := javaSuggestionService(t, fake)
	cc := NewCallContext(root, backend.ProjectContext{RootDir: root, Language: backend.LanguageJava})
	cc.Service = service
	entry, _ := DefaultRegistry().LookupKey(string(backend.OperationLookup))
	err := suggestJavaFailedTarget(cc, entry, map[string]any{"file": "Widget.java", "language": "java", "symbol": "Widget"}, errors.New("exact success path"))
	if err == nil || err.Error() != "exact success path" || fake.outlineCalls != 0 {
		t.Fatalf("result=%v outline calls=%d", err, fake.outlineCalls)
	}
}

func TestSuggestJavaInspectAndReferencesUseNativeChecks(t *testing.T) {
	t.Setenv("SEMEDIT_SUGGESTIONS", "generic")
	for _, key := range []string{string(backend.OperationInspect), string(backend.OperationFindReferences)} {
		t.Run(key, func(t *testing.T) {
			root := javaSuggestionRoot(t, "class Widget {}\n")
			file := filepath.Join(root, "Widget.java")
			fake := &javaSuggestionBackend{file: file, selector: "demo.Widget"}
			cc := NewCallContext(root, backend.ProjectContext{RootDir: root, Language: backend.LanguageJava})
			cc.Service = javaSuggestionService(t, fake)
			entry, ok := DefaultRegistry().LookupKey(key)
			if !ok {
				t.Fatalf("operation %q missing", key)
			}
			cause := &javabackend.JavaError{Op: key, File: file, Symbol: "demo.Widgte", Err: javabackend.ErrJavaSymbolNotFound}
			err := suggestJavaFailedTarget(cc, entry, map[string]any{"file": "Widget.java", "language": "java", "trust_workspace": true, "symbol": "demo.Widgte"}, cause)
			if values := SuggestionsFromError(err); len(values) != 1 || values[0].Arguments["symbol"] != "demo.Widget" {
				t.Fatalf("suggestions %#v (%v)", values, err)
			}
			if fake.lookupCalls != 1 || fake.inspectCalls != 1 || fake.outlineCalls != 1 {
				t.Fatalf("native counts lookup=%d inspect=%d outline=%d", fake.lookupCalls, fake.inspectCalls, fake.outlineCalls)
			}
			if key == string(backend.OperationFindReferences) && fake.referenceCalls != 1 {
				t.Fatalf("reference validation calls=%d", fake.referenceCalls)
			}
		})
	}
}

func TestSuggestJavaBatchAliasReturnsCurrentSelector(t *testing.T) {
	t.Setenv("SEMEDIT_SUGGESTIONS", "generic")
	root := javaSuggestionRoot(t, "class Gadget {}\n")
	file := filepath.Join(root, "Widget.java")
	buffer, err := sourcebuf.New(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	buffer.RecordSymbolRename(sourcebuf.SymbolAlias{Language: "java", File: file, NewFile: file, OldSelector: "demo.Widget", NewSelector: "demo.Gadget", Kind: "class"})
	fake := &javaSuggestionBackend{file: file, selector: "demo.Gadget"}
	service := javaSuggestionService(t, fake)
	cc := NewCallContext(root, backend.ProjectContext{RootDir: root, Language: backend.LanguageJava})
	cc.Ctx, cc.Service = sourcebuf.WithBuffer(context.Background(), buffer), service
	entry, _ := DefaultRegistry().LookupKey(string(backend.OperationLookup))
	cause := &javabackend.JavaError{Op: "lookup", File: file, Symbol: "demo.Widgte", Err: javabackend.ErrJavaSymbolNotFound}
	err = suggestJavaFailedTarget(cc, entry, map[string]any{"file": "Widget.java", "language": "java", "symbol": "demo.Widgte"}, cause)
	if !errors.Is(err, javabackend.ErrJavaSymbolNotFound) {
		t.Fatalf("failure identity = %v", err)
	}
	alternatives := SuggestionsFromError(err)
	if len(alternatives) != 1 || alternatives[0].Arguments["symbol"] != "demo.Gadget" {
		t.Fatalf("alias suggestions = %#v", alternatives)
	}
}

func TestJavaDispatchRenameAliasFeedsLaterSuggestion(t *testing.T) {
	t.Setenv("SEMEDIT_SUGGESTIONS", "generic")
	root := javaSuggestionRoot(t, "class Widget {}\n")
	file := filepath.Join(root, "Widget.java")
	buffer, err := sourcebuf.New(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	fake := &javaSuggestionBackend{file: file, selector: "demo.Widget"}
	cc := NewCallContext(root, backend.ProjectContext{RootDir: root, Language: backend.LanguageJava})
	cc.Service, cc.Ctx = javaSuggestionService(t, fake), sourcebuf.WithBuffer(context.Background(), buffer)
	if _, dispatchErr := cc.Registry.Dispatch(cc, string(backend.OperationRename), map[string]any{"file": "Widget.java", "language": "java", "trust_workspace": true, "symbol": "demo.Widget", "to": "Gadget"}); dispatchErr != nil {
		t.Fatalf("rename dispatch: %v", dispatchErr)
	}
	aliases := buffer.SymbolAliases()
	if len(aliases) != 1 || aliases[0].Language != "java" || aliases[0].OldSelector != "demo.Widget" || aliases[0].NewSelector != "demo.Gadget" {
		t.Fatalf("recorded aliases %#v", aliases)
	}
	entry, _ := DefaultRegistry().LookupKey(string(backend.OperationLookup))
	cause := &javabackend.JavaError{Op: "lookup", File: file, Symbol: "demo.Widgte", Err: javabackend.ErrJavaSymbolNotFound}
	err = suggestJavaFailedTarget(cc, entry, map[string]any{"file": "Widget.java", "language": "java", "symbol": "demo.Widgte"}, cause)
	values := SuggestionsFromError(err)
	if len(values) != 1 || values[0].Arguments["symbol"] != "demo.Gadget" {
		t.Fatalf("retry from dispatch alias %#v (%v)", values, err)
	}
}

func TestSuggestJavaAliasChainPreservesOverloadIdentity(t *testing.T) {
	t.Setenv("SEMEDIT_SUGGESTIONS", "generic")
	root := javaSuggestionRoot(t, "class Ledger {}\n")
	file := filepath.Join(root, "Widget.java")
	buffer, err := sourcebuf.New(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	buffer.RecordSymbolRename(sourcebuf.SymbolAlias{Language: "java", File: file, NewFile: file, OldSelector: "demo.Widget", NewSelector: "demo.Account", Kind: "class"})
	buffer.RecordSymbolRename(sourcebuf.SymbolAlias{Language: "java", File: file, NewFile: file, OldSelector: "demo.Account", NewSelector: "demo.Ledger", Kind: "class"})
	current := "demo.Ledger.send(int)"
	fake := &javaSuggestionBackend{file: file, selector: current, selectors: []string{current, "demo.Ledger.send(String)"}, kinds: map[string]string{current: "method", "demo.Ledger.send(String)": "method"}}
	cc := NewCallContext(root, backend.ProjectContext{RootDir: root, Language: backend.LanguageJava})
	cc.Ctx, cc.Service = sourcebuf.WithBuffer(context.Background(), buffer), javaSuggestionService(t, fake)
	entry, _ := DefaultRegistry().LookupKey(string(backend.OperationLookup))
	cause := &javabackend.JavaError{Op: "lookup", File: file, Symbol: "demo.Widget.sned(int)", Err: javabackend.ErrJavaSymbolNotFound}
	err = suggestJavaFailedTarget(cc, entry, map[string]any{"file": "Widget.java", "language": "java", "symbol": "demo.Widget.sned(int)"}, cause)
	alternatives := SuggestionsFromError(err)
	if len(alternatives) != 1 || alternatives[0].Arguments["symbol"] != current || alternatives[0].Kind != "method" {
		t.Fatalf("overload alias result %#v (%v)", alternatives, err)
	}
}

func TestSuggestJavaRenameUsesNativeNonMutatingValidator(t *testing.T) {
	t.Setenv("SEMEDIT_SUGGESTIONS", "generic")
	root := javaSuggestionRoot(t, "class Widget {}\n")
	file := filepath.Join(root, "Widget.java")
	fake := &javaSuggestionBackend{file: file, selector: "demo.Widget"}
	cc := NewCallContext(root, backend.ProjectContext{RootDir: root, Language: backend.LanguageJava})
	cc.Service = javaSuggestionService(t, fake)
	entry, _ := DefaultRegistry().LookupKey(string(backend.OperationRename))
	cause := &javabackend.JavaError{Op: "rename", File: file, Symbol: "demo.Widgte", Err: javabackend.ErrJavaSymbolNotFound}
	err := suggestJavaFailedTarget(cc, entry, map[string]any{"file": "Widget.java", "language": "java", "trust_workspace": true, "symbol": "demo.Widgte", "to": "Gadget"}, cause)
	alternatives := SuggestionsFromError(err)
	if len(alternatives) != 1 || alternatives[0].Arguments["symbol"] != "demo.Widget" || fake.renameValidationCalls != 1 {
		t.Fatalf("rename suggestions=%#v validations=%d err=%v", alternatives, fake.renameValidationCalls, err)
	}
}

func TestSuggestJavaAmbiguityKeepsOnlyExactNativeCandidates(t *testing.T) {
	t.Setenv("SEMEDIT_SUGGESTIONS", "codanna")
	t.Setenv("SEMEDIT_CODANNA_BIN", "/missing-codanna")
	root := javaSuggestionRoot(t, "class Widget {}\n")
	file := filepath.Join(root, "Widget.java")
	fake := &javaSuggestionBackend{file: file, selectors: []string{"demo.Widget", "other.Widget", "demo.Widgets"}, matches: map[string][]string{"Widget": {"demo.Widget", "other.Widget"}}}
	service := javaSuggestionService(t, fake)
	cc := NewCallContext(root, backend.ProjectContext{RootDir: root, Language: backend.LanguageJava})
	cc.Service = service
	entry, _ := DefaultRegistry().LookupKey(string(backend.OperationLookup))
	err := suggestJavaFailedTarget(cc, entry, map[string]any{"file": "Widget.java", "language": "java", "symbol": "Widget"}, backend.ErrAmbiguous)
	alternatives := SuggestionsFromError(err)
	if len(alternatives) != 2 || alternatives[0].Arguments["symbol"] != "demo.Widget" || alternatives[1].Arguments["symbol"] != "other.Widget" {
		t.Fatalf("ambiguous suggestions = %#v", alternatives)
	}
	for _, alternative := range alternatives {
		result, retryErr := service.Lookup(t.Context(), backend.ProjectContext{RootDir: root, File: file, Language: backend.LanguageJava}, alternative.Arguments["symbol"])
		if retryErr != nil || result == nil || result.Definition == nil || result.IsAmbiguous() {
			t.Fatalf("ambiguous candidate retry: %#v, %v", result, retryErr)
		}
	}
}

func TestSuggestJavaQualifiedQueryKeepsRequestedOwner(t *testing.T) {
	t.Setenv("SEMEDIT_SUGGESTIONS", "generic")
	root := javaSuggestionRoot(t, "class Widget {}\n")
	file := filepath.Join(root, "Widget.java")
	fake := &javaSuggestionBackend{file: file, selectors: []string{"demo.Other.read"}}
	cc := NewCallContext(root, backend.ProjectContext{RootDir: root, Language: backend.LanguageJava})
	cc.Service = javaSuggestionService(t, fake)
	entry, _ := DefaultRegistry().LookupKey(string(backend.OperationLookup))
	cause := &javabackend.JavaError{Op: "lookup", File: file, Symbol: "demo.Widget.readd", Err: javabackend.ErrJavaSymbolNotFound}
	err := suggestJavaFailedTarget(cc, entry, map[string]any{"file": "Widget.java", "language": "java", "symbol": "demo.Widget.readd"}, cause)
	if alternatives := SuggestionsFromError(err); len(alternatives) != 0 {
		t.Fatalf("different Java owner was suggested: %#v", alternatives)
	}
}

func TestSuggestJavaRejectsStaleCodannaCandidate(t *testing.T) {
	t.Setenv("SEMEDIT_SUGGESTIONS", "codanna")
	root := javaSuggestionRoot(t, "class Widget {}\n")
	file := filepath.Join(root, "Widget.java")
	fake := &javaSuggestionBackend{file: file, selector: "demo.Widget"}
	cc := NewCallContext(root, backend.ProjectContext{RootDir: root, Language: backend.LanguageJava})
	cc.Service = javaSuggestionService(t, fake)
	cc.SuggestionProvider = staticSuggestionProvider{values: []suggestions.Candidate{{Selector: "demo.Widgit", File: file, Kind: "class"}}}
	entry, _ := DefaultRegistry().LookupKey(string(backend.OperationLookup))
	cause := &javabackend.JavaError{Op: "lookup", File: file, Symbol: "demo.Widgit", Err: javabackend.ErrJavaSymbolNotFound}
	err := suggestJavaFailedTarget(cc, entry, map[string]any{"file": "Widget.java", "language": "java", "symbol": "demo.Widgit"}, cause)
	if suggestions := SuggestionsFromError(err); len(suggestions) != 0 || !errors.Is(err, cause) {
		t.Fatalf("stale index result produced suggestions: %#v (%v)", suggestions, err)
	}
}

type staticSuggestionProvider struct{ values []suggestions.Candidate }

func (provider staticSuggestionProvider) Retrieve(context.Context, suggestions.Query) ([]suggestions.Candidate, error) {
	return provider.values, nil
}

type javaSuggestionBackend struct {
	file                  string
	selector              string
	selectors             []string
	matches               map[string][]string
	kinds                 map[string]string
	outlineCalls          int
	lookupCalls           int
	inspectCalls          int
	referenceCalls        int
	renameValidationCalls int
}

func (*javaSuggestionBackend) Language() backend.LanguageID { return backend.LanguageJava }
func (*javaSuggestionBackend) Capabilities() backend.Capabilities {
	return backend.NewCapabilities(backend.OperationLookup, backend.OperationRename, backend.OperationOutline, backend.OperationInspect, backend.OperationFindReferences)
}
func (fake *javaSuggestionBackend) Lookup(_ context.Context, project backend.ProjectContext, selector string) (*backend.LookupResult, error) {
	fake.lookupCalls++
	if matches := fake.matches[selector]; len(matches) > 0 {
		result := &backend.LookupResult{}
		for _, match := range matches {
			result.Candidates = append(result.Candidates, &backend.NamedTarget{QualifiedName: match, Kind: "class", Location: backend.SourceLocation{URI: "file://" + fake.file}})
		}
		return result, nil
	}
	if selector != fake.selector && !containsString(fake.selectors, selector) {
		return nil, &javabackend.JavaError{Op: "lookup", Symbol: selector, Err: javabackend.ErrJavaSymbolNotFound}
	}
	return &backend.LookupResult{Definition: &backend.NamedTarget{Name: "Widget", QualifiedName: selector, Kind: "class", Location: backend.SourceLocation{URI: "file://" + fake.file, Range: backend.Range{Start: backend.Position{Line: 0, Character: 6}}}}}, nil
}
func (fake *javaSuggestionBackend) Rename(_ context.Context, request backend.RenameRequest) (*backend.RenameResult, error) {
	original := fake.selector
	end := strings.IndexByte(original, '(')
	if end < 0 {
		end = len(original)
	}
	start := strings.LastIndexByte(original[:end], '.') + 1
	newSelector := original[:start] + request.To + original[end:]
	fake.selector = newSelector
	definition := &backend.NamedTarget{Name: "Widget", QualifiedName: original, Kind: "class", Location: backend.SourceLocation{URI: "file://" + fake.file, Range: backend.Range{Start: backend.Position{Line: 0, Character: 6}}}}
	return &backend.RenameResult{Lookup: &backend.LookupResult{Definition: definition}, TargetFile: fake.file}, nil
}
func (fake *javaSuggestionBackend) ValidateRenameTarget(_ context.Context, request backend.RenameRequest) error {
	fake.renameValidationCalls++
	if request.Symbol != fake.selector || request.To != "Gadget" {
		return &javabackend.JavaError{Op: "rename", Symbol: request.Symbol, Err: javabackend.ErrJavaSymbolNotFound}
	}
	return nil
}
func (*javaSuggestionBackend) Verify(context.Context, backend.VerifyRequest) ([]backend.Diagnostic, error) {
	return nil, nil
}
func (fake *javaSuggestionBackend) Outline(_ context.Context, request backend.OutlineRequest) (*backend.OutlineResult, error) {
	fake.outlineCalls++
	selectors := fake.selectors
	if len(selectors) == 0 {
		selectors = []string{fake.selector}
	}
	items := make([]backend.ReadSymbol, 0, len(selectors))
	for index, selector := range selectors {
		name := selector[strings.LastIndex(selector, ".")+1:]
		if open := strings.IndexByte(name, '('); open >= 0 {
			name = name[:open]
		}
		kind := fake.kinds[selector]
		if kind == "" {
			kind = "class"
		}
		items = append(items, backend.ReadSymbol{Name: name, QualifiedName: selector, Kind: kind, File: fake.file, SelectionRange: backend.Range{Start: backend.Position{Line: 0, Character: 6 + index*8}}})
	}
	return &backend.OutlineResult{Files: []backend.OutlineFile{{File: fake.file, Language: backend.LanguageJava, Symbols: items}}}, nil
}
func (fake *javaSuggestionBackend) Inspect(_ context.Context, request backend.InspectRequest) (*backend.InspectResult, error) {
	fake.inspectCalls++
	if request.Symbol != fake.selector && !containsString(fake.selectors, request.Symbol) {
		return nil, &javabackend.JavaError{Op: "inspect", Symbol: request.Symbol, Err: javabackend.ErrJavaSymbolNotFound}
	}
	name := request.Symbol[strings.LastIndex(request.Symbol, ".")+1:]
	if open := strings.IndexByte(name, '('); open >= 0 {
		name = name[:open]
	}
	kind := fake.kinds[request.Symbol]
	if kind == "" {
		kind = "class"
	}
	position := 6
	for index, selector := range fake.selectors {
		if selector == request.Symbol {
			position += index * 8
			break
		}
	}
	item := backend.ReadSymbol{Name: name, QualifiedName: request.Symbol, Kind: kind, File: fake.file, SelectionRange: backend.Range{Start: backend.Position{Line: 0, Character: position}}}
	return &backend.InspectResult{Matches: []backend.InspectedSymbol{{ReadSymbol: item, Source: "class Widget {}", Revision: "rev"}}}, nil
}

func containsString(values []string, candidate string) bool {
	return slices.Contains(values, candidate)
}
func (fake *javaSuggestionBackend) FindReferences(_ context.Context, request backend.ReferencesRequest) (*backend.ReferencesResult, error) {
	fake.referenceCalls++
	return &backend.ReferencesResult{Symbol: backend.ReadSymbol{QualifiedName: request.Symbol}}, nil
}

func javaSuggestionService(t *testing.T, fake backend.Backend) *backend.Service {
	t.Helper()
	registry, err := backend.NewRegistry(fake)
	if err != nil {
		t.Fatal(err)
	}
	return backend.NewService(registry)
}

func javaSuggestionRoot(t *testing.T, source string) string {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "src", "main", "java"), 0700); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(root, "Widget.java")
	if err := os.WriteFile(file, []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
	return root
}
