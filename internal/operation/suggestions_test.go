// Package operation proves that failed-target alternatives can be explicitly retried without suggestion-time edits.
package operation

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/spockz/semantic-editor/internal/atomicfile"
	"github.com/spockz/semantic-editor/internal/backend"
	"github.com/spockz/semantic-editor/internal/sourcebuf"
	"github.com/spockz/semantic-editor/internal/suggestions"
	"github.com/spockz/semantic-editor/internal/symbol"
)

func suggestionFixture(t *testing.T, files map[string]string) (CallContext, *sourcebuf.Buffer) {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for name, source := range files {
		path := filepath.Join(root, name)
		if mkdirErr := os.MkdirAll(filepath.Dir(path), 0o700); mkdirErr != nil {
			t.Fatal(mkdirErr)
		}
		if writeErr := atomicfile.WriteAtomic(path, []byte(source)); writeErr != nil {
			t.Fatal(writeErr)
		}
	}
	buffer, err := sourcebuf.New(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	cc := NewCallContext(root, backend.ProjectContext{RootDir: root, Language: backend.LanguageGo})
	cc.Ctx = sourcebuf.WithBuffer(context.Background(), buffer)
	cc.DeferImports, cc.DeferVerification = true, true
	return cc, buffer
}

func TestFailedTargetSuggestionsValidateExplicitRetries(t *testing.T) {
	t.Setenv("SEMEDIT_SUGGESTIONS", "generic")
	cc, buffer := suggestionFixture(t, map[string]string{
		"go.mod":          "module example.com/suggestions\n\ngo 1.23\n",
		"config.go":       "package config\nfunc ParseConfig() {}\ntype ParseConfg int\ntype Server struct{}\nfunc (Server) ParseConfig() {}\n",
		"other/config.go": "package other\nfunc ParseConfig() {}\n",
	})
	for _, test := range []struct {
		name, key, query, file string
		want                   int
	}{
		{"transposition", "replace_body", "PraseConfig", "config.go", 1},
		{"receiver", "replace_body", "Server.ParseConfg", "config.go", 1},
		{"wrong receiver", "replace_body", "Other.ParseConfg", "config.go", 0},
		{"workspace alternatives", "refactor_rename", "PraseConfig", "", 3},
		{"file restriction", "replace_body", "ParseConfg", "other/config.go", 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			assertFailedTargetSuggestionCase(t, cc, buffer, test.key, test.query, test.file, test.want)
		})
	}
}

func assertFailedTargetSuggestionCase(t *testing.T, cc CallContext, buffer *sourcebuf.Buffer, key, query, file string, want int) {
	t.Helper()
	raw := map[string]any{"symbol": query}
	if file != "" {
		raw["file"] = file
	}
	if key == "replace_body" {
		raw["body"] = "return"
	} else {
		raw["to"] = "ReadConfig"
	}
	before := buffer.ChangedPaths()
	_, err := cc.Registry.Dispatch(cc, key, raw)
	if err == nil || !errors.Is(err, symbol.ErrNotFound) {
		t.Fatalf("failure identity = %v", err)
	}
	alternatives := SuggestionsFromError(err)
	if len(alternatives) != want {
		t.Fatalf("suggestions = %#v, want %d; error %v", alternatives, want, err)
	}
	if len(buffer.ChangedPaths()) != len(before) {
		t.Fatal("suggestion generation changed the working source")
	}
	for _, alternative := range alternatives {
		if key == "replace_body" && alternative.Arguments["symbol"] == "ParseConfg" {
			t.Fatal("wrong-kind type was accepted as a body target")
		}
		retry := cloneRaw(raw)
		for name, value := range alternative.Arguments {
			retry[name] = value
		}
		retryCall := cc
		retryCall.Ctx = sourcebuf.WithBuffer(context.Background(), buffer.Fork())
		if _, retryErr := retryCall.Registry.Dispatch(retryCall, key, retry); retryErr != nil {
			t.Fatalf("explicit retry %#v: %v", retry, retryErr)
		}
	}
}

func TestReferenceSuggestionsUseActiveTypedTargets(t *testing.T) {
	t.Setenv("SEMEDIT_SUGGESTIONS", "generic")
	cc, buffer := suggestionFixture(t, map[string]string{
		"go.mod":      "module example.com/references\n\ngo 1.23\n",
		"source.go":   "package config\nfunc ParseConfig() {}\nfunc use() { ParseConfig() }\n",
		"inactive.go": "//go:build ignore\n\npackage config\nfunc PraseConfig() {}\n",
	})
	_, err := cc.Registry.Dispatch(cc, "find_references", map[string]any{"symbol": "ParseConfg"})
	alternatives := SuggestionsFromError(err)
	if !errors.Is(err, symbol.ErrNotFound) || len(alternatives) != 1 || alternatives[0].Arguments["symbol"] != "ParseConfig" {
		t.Fatalf("typed reference alternatives: %#v (%v)", alternatives, err)
	}
	for _, alternative := range alternatives {
		retry := map[string]any{}
		for key, value := range alternative.Arguments {
			retry[key] = value
		}
		if _, retryErr := cc.Registry.Dispatch(cc, "find_references", retry); retryErr != nil {
			t.Fatalf("typed reference retry: %v", retryErr)
		}
	}
	if len(buffer.ChangedPaths()) != 0 {
		t.Fatal("reference suggestions changed sources")
	}
}

func TestSuggestionProvidersAreFailureOnly(t *testing.T) {
	cc, _ := suggestionFixture(t, map[string]string{"go.mod": "module example.com/provider\n\ngo 1.23\n", "source.go": "package sample\nfunc ParseConfig() {}\n"})
	provider := &countingSuggestionProvider{}
	cc.SuggestionProvider = provider
	t.Setenv("SEMEDIT_SUGGESTIONS", "codanna,generic")
	if _, err := cc.Registry.Dispatch(cc, "lookup", map[string]any{"symbol": "ParseConfig"}); err != nil {
		t.Fatal(err)
	}
	if provider.calls.Load() != 0 {
		t.Fatal("provider ran for an exact target")
	}
	_, err := cc.Registry.Dispatch(cc, "refactor_rename", map[string]any{"symbol": "ParseConfig", "to": "invalid-name"})
	if err == nil || len(SuggestionsFromError(err)) != 0 {
		t.Fatalf("introduced name was corrected: %v", err)
	}
	if provider.calls.Load() != 0 {
		t.Fatal("provider ran for a valid original target")
	}
	_, err = cc.Registry.Dispatch(cc, "lookup", map[string]any{"symbol": "ParseConfg"})
	if !errors.Is(err, symbol.ErrNotFound) || len(SuggestionsFromError(err)) != 1 {
		t.Fatalf("malformed provider replaced failure/fallback: %v", err)
	}
	if provider.calls.Load() == 0 {
		t.Fatal("configured provider was not queried on failure")
	}
	t.Setenv("SEMEDIT_SUGGESTIONS", "off")
	_, err = cc.Registry.Dispatch(cc, "lookup", map[string]any{"symbol": "ParseConfg"})
	if !errors.Is(err, symbol.ErrNotFound) || len(SuggestionsFromError(err)) != 0 {
		t.Fatalf("disabled suggestions: %v", err)
	}
}

func TestAmbiguousTargetsPreferExactNativeCandidates(t *testing.T) {
	t.Setenv("SEMEDIT_SUGGESTIONS", "codanna")
	t.Setenv("SEMEDIT_CODANNA_BIN", "/missing-codanna")
	cc, buffer := suggestionFixture(t, map[string]string{
		"go.mod":         "module example.com/ambiguous\n\ngo 1.23\n",
		"one/parse.go":   "package one\nfunc ParseConfig() {}\n",
		"two/parse.go":   "package two\nfunc ParseConfig() {}\n",
		"three/parse.go": "package three\nfunc PraseConfig() {}\n",
	})
	_, err := cc.Registry.Dispatch(cc, "refactor_rename", map[string]any{"symbol": "ParseConfig", "to": "ReadConfig"})
	if !errors.Is(err, backend.ErrAmbiguous) {
		t.Fatalf("ambiguity identity lost: %v", err)
	}
	alternatives := SuggestionsFromError(err)
	if len(alternatives) != 2 {
		t.Fatalf("exact alternatives: %#v (%v)", alternatives, err)
	}
	for _, alternative := range alternatives {
		if alternative.Arguments["symbol"] != "ParseConfig" || alternative.File == "three/parse.go" {
			t.Fatalf("broad fuzzy search on ambiguity: %#v", alternative)
		}
		retryCall := cc
		retryCall.Ctx = sourcebuf.WithBuffer(context.Background(), buffer.Fork())
		retry := map[string]any{"to": "ReadConfig"}
		for key, value := range alternative.Arguments {
			retry[key] = value
		}
		if _, err := retryCall.Registry.Dispatch(retryCall, "refactor_rename", retry); err != nil {
			t.Fatalf("retry exact ambiguity alternative: %v", err)
		}
	}
	if len(buffer.ChangedPaths()) != 0 {
		t.Fatal("ambiguity suggestions changed sources")
	}
}

type countingSuggestionProvider struct{ calls atomic.Int32 }

func (p *countingSuggestionProvider) Retrieve(context.Context, suggestions.Query) ([]suggestions.Candidate, error) {
	p.calls.Add(1)
	return nil, errors.New("malformed provider response")
}

type fixedSuggestionProvider struct{ candidates []suggestions.Candidate }

func (p fixedSuggestionProvider) Retrieve(context.Context, suggestions.Query) ([]suggestions.Candidate, error) {
	return p.candidates, nil
}

func TestIndexedProviderCanReachPastNativeEnumerationBound(t *testing.T) {
	t.Setenv("SEMEDIT_SUGGESTIONS", "codanna")
	files := map[string]string{"ztarget.go": "package config\nfunc ParseConfig() {}\n"}
	for i := range 256 {
		files[fmt.Sprintf("a%03d.go", i)] = "package config\n"
	}
	cc, buffer := suggestionFixture(t, files)
	cc.SuggestionProvider = fixedSuggestionProvider{candidates: []suggestions.Candidate{
		{Selector: "ParseConfig", File: "gone.go", Kind: "function"},
		{Selector: "ParseConfig", File: "ztarget.go", Kind: "type", Signature: "untrusted provider metadata"},
	}}
	_, err := cc.Registry.Dispatch(cc, "lookup", map[string]any{"symbol": "ParseConfg"})
	if !errors.Is(err, symbol.ErrNotFound) {
		t.Fatalf("indexed failure identity: %v", err)
	}
	alternatives := SuggestionsFromError(err)
	if len(alternatives) != 1 || alternatives[0].File != "ztarget.go" || alternatives[0].Kind != "function" {
		t.Fatalf("indexed candidates were not natively normalized: %#v (%v)", alternatives, err)
	}
	retry := map[string]any{}
	for key, value := range alternatives[0].Arguments {
		retry[key] = value
	}
	if _, retryErr := cc.Registry.Dispatch(cc, "lookup", retry); retryErr != nil {
		t.Fatalf("indexed target retry: %v", retryErr)
	}
	if len(buffer.ChangedPaths()) != 0 {
		t.Fatal("indexed suggestion generation changed sources")
	}
}

func TestIndexedSuggestionPreservesInheritedSelectorScope(t *testing.T) {
	t.Setenv("SEMEDIT_SUGGESTIONS", "codanna")
	cc, buffer := suggestionFixture(t, map[string]string{
		"base.go":  "package config\ntype Base interface { ParseConfig() }\n",
		"outer.go": "package config\ntype Outer interface { Base }\n",
	})
	cc.SuggestionProvider = fixedSuggestionProvider{candidates: []suggestions.Candidate{
		{Selector: "Outer.ParseConfig", File: "outer.go", Kind: "method"},
	}}
	_, err := cc.Registry.Dispatch(cc, "lookup", map[string]any{"symbol": "Outer.ParseConfg"})
	alternatives := SuggestionsFromError(err)
	if len(alternatives) != 1 || alternatives[0].Arguments["file"] != "outer.go" || alternatives[0].File != "base.go" {
		t.Fatalf("inherited selector scope: %#v (%v)", alternatives, err)
	}
	retry := map[string]any{}
	for key, value := range alternatives[0].Arguments {
		retry[key] = value
	}
	if _, retryErr := cc.Registry.Dispatch(cc, "lookup", retry); retryErr != nil {
		t.Fatalf("inherited target retry: %v", retryErr)
	}
	if len(buffer.ChangedPaths()) != 0 {
		t.Fatal("suggestion generation changed sources")
	}
}

func TestIndexedSuggestionRejectsWorkspaceEscape(t *testing.T) {
	t.Setenv("SEMEDIT_SUGGESTIONS", "codanna")
	cc, buffer := suggestionFixture(t, map[string]string{"local.go": "package config\n"})
	outside := filepath.Join(t.TempDir(), "outside.go")
	if err := atomicfile.WriteAtomic(outside, []byte("package config\nfunc ParseConfig() {}\n")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(cc.WorkDir, "escape.go")); err != nil {
		t.Fatal(err)
	}
	cc.SuggestionProvider = fixedSuggestionProvider{candidates: []suggestions.Candidate{
		{Selector: "ParseConfig", File: "escape.go", Kind: "function"},
	}}
	_, err := cc.Registry.Dispatch(cc, "lookup", map[string]any{"symbol": "ParseConfg"})
	if !errors.Is(err, symbol.ErrNotFound) || len(SuggestionsFromError(err)) != 0 {
		t.Fatalf("workspace escape accepted: %v", err)
	}
	if len(buffer.ChangedPaths()) != 0 {
		t.Fatal("suggestion generation changed sources")
	}
}

func TestSuggestionsProjectContainingTypeRename(t *testing.T) {
	t.Setenv("SEMEDIT_SUGGESTIONS", "generic")
	cc, buffer := suggestionFixture(t, map[string]string{
		"owner.go":  "package config\ntype Reader struct{}\n",
		"method.go": "package config\nfunc (Reader) ParseConfig() {}\n",
	})
	buffer.RecordSymbolRename(sourcebuf.SymbolAlias{File: filepath.Join(cc.WorkDir, "owner.go"), OldSelector: "Parser", NewSelector: "Reader", Kind: "type"})
	_, err := cc.Registry.Dispatch(cc, "lookup", map[string]any{"symbol": "Parser.ParseConfg"})
	alternatives := SuggestionsFromError(err)
	if len(alternatives) != 1 || alternatives[0].Arguments["symbol"] != "Reader.ParseConfig" {
		t.Fatalf("qualified former owner: %#v (%v)", alternatives, err)
	}
	retry := map[string]any{}
	for key, value := range alternatives[0].Arguments {
		retry[key] = value
	}
	if _, retryErr := cc.Registry.Dispatch(cc, "lookup", retry); retryErr != nil {
		t.Fatalf("retry projected owner: %v", retryErr)
	}
}

func TestSuggestionsRequireNativeSignatureCompatibility(t *testing.T) {
	t.Setenv("SEMEDIT_SUGGESTIONS", "generic")
	cc, buffer := suggestionFixture(t, map[string]string{
		"go.mod":    "module example.com/signature\n\ngo 1.23\n",
		"source.go": "package signature\nfunc process(x int) {}\nfunc procesx(x string) {}\nfunc Process(x int) {}\n",
	})
	raw := map[string]any{"file": "source.go", "symbol": "pro" + "ces", "parameters": []any{
		map[string]any{"source_index": 0, "parameter": map[string]any{"name": "x", "type": "int"}},
		map[string]any{"parameter": map[string]any{"name": "mode", "type": "int"}, "argument": "0"},
	}}
	_, err := cc.Registry.Dispatch(cc, "refactor_change_signature", raw)
	if !errors.Is(err, symbol.ErrNotFound) {
		t.Fatalf("signature failure identity: %v", err)
	}
	alternatives := SuggestionsFromError(err)
	if len(alternatives) != 1 || alternatives[0].Arguments["symbol"] != "process" {
		t.Fatalf("unproven signature candidates: %#v (%v)", alternatives, err)
	}
	if len(buffer.ChangedPaths()) != 0 {
		t.Fatal("signature candidate planning changed sources")
	}
	for _, alternative := range alternatives {
		retry := cloneRaw(raw)
		for key, value := range alternative.Arguments {
			retry[key] = value
		}
		retryCall := cc
		retryCall.Ctx = sourcebuf.WithBuffer(context.Background(), buffer.Fork())
		if _, err := cc.Registry.Dispatch(retryCall, "refactor_change_signature", retry); err != nil {
			t.Fatalf("signature retry: %v", err)
		}
	}
}

func TestSuggestedMutationsPassOriginalPreflight(t *testing.T) {
	t.Setenv("SEMEDIT_SUGGESTIONS", "generic")
	for _, test := range []struct {
		name, key, selectorKey string
		raw                    map[string]any
	}{
		{"remove kind", "remove_construct", "symbol", map[string]any{"kind": "function", "symbol": "ParseConfg"}},
		{"nested owner and discriminator", "replace_construct", "in_function", map[string]any{"kind": "if", "in_function": "ParseConfg", "discriminator": "ready", "source": "if ready { use() }"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			cc, buffer := suggestionFixture(t, map[string]string{
				"go.mod":    "module example.com/mutationsuggestions\n\ngo 1.24\n",
				"source.go": "package config\nconst ParseConff = 1\nvar ready = true\nfunc use() {}\nfunc ParseConfig() { if ready { use() } }\nfunc ParseConfl() {}\n",
			})
			raw := cloneRaw(test.raw)
			raw["file"] = "source.go"
			_, err := cc.Registry.Dispatch(cc, test.key, raw)
			if !errors.Is(err, symbol.ErrNotFound) {
				t.Fatalf("original failure: %v", err)
			}
			alternatives := SuggestionsFromError(err)
			if len(alternatives) == 0 {
				t.Fatalf("no compatible alternatives: %v", err)
			}
			for _, alternative := range alternatives {
				if alternative.Kind != "function" {
					t.Fatalf("wrong-kind suggestion: %#v", alternative)
				}
				if test.selectorKey == "in_function" && alternative.Arguments[test.selectorKey] != "ParseConfig" {
					t.Fatalf("owner lacks required construct: %#v", alternative)
				}
				retry := cloneRaw(raw)
				for key, value := range alternative.Arguments {
					retry[key] = value
				}
				retryCall := cc
				retryCall.Ctx = sourcebuf.WithBuffer(context.Background(), buffer.Fork())
				if _, err := cc.Registry.Dispatch(retryCall, test.key, retry); err != nil {
					t.Fatalf("explicit original mutation retry: %v", err)
				}
			}
			if len(buffer.ChangedPaths()) != 0 {
				t.Fatal("suggestion generation mutated sources")
			}
		})
	}
}

func TestSuggestionsFollowRetainedRenameIdentity(t *testing.T) {
	t.Setenv("SEMEDIT_SUGGESTIONS", "generic")
	cc, buffer := suggestionFixture(t, map[string]string{"source.go": "package config\nfunc ReadConfig() {}\nfunc ParseConfig() {}\n"})
	file := filepath.Join(cc.WorkDir, "source.go")
	buffer.RecordSymbolRename(sourcebuf.SymbolAlias{File: file, OldSelector: "ParseConfig", NewSelector: "LoadConfig", Kind: "function"})
	buffer.RecordSymbolRename(sourcebuf.SymbolAlias{File: file, OldSelector: "LoadConfig", NewSelector: "ReadConfig", Kind: "function"})
	_, err := cc.Registry.Dispatch(cc, "lookup", map[string]any{"symbol": "ParseConfg"})
	alternatives := SuggestionsFromError(err)
	if len(alternatives) != 2 {
		t.Fatalf("renamed and reused identities: %#v (%v)", alternatives, err)
	}
	bySelector := make(map[string]TargetSuggestion)
	for _, alternative := range alternatives {
		bySelector[alternative.Arguments["symbol"]] = alternative
	}
	if len(bySelector) != 2 || bySelector["ParseConfig"].File == "" || bySelector["ReadConfig"].File == "" {
		t.Fatalf("identities = %#v", alternatives)
	}
	if !strings.Contains(bySelector["ReadConfig"].Reason, "former name ParseConfig") {
		t.Fatalf("missing rename evidence: %#v", bySelector["ReadConfig"])
	}
	for _, alternative := range alternatives {
		retry := map[string]any{}
		for key, value := range alternative.Arguments {
			retry[key] = value
		}
		if _, retryErr := cc.Registry.Dispatch(cc, "lookup", retry); retryErr != nil {
			t.Fatalf("retry: %v", retryErr)
		}
	}
	buffer.InvalidateSymbolAliases(file, "ReadConfig")
	_, err = cc.Registry.Dispatch(cc, "lookup", map[string]any{"symbol": "ParseConfg"})
	for _, alternative := range SuggestionsFromError(err) {
		if alternative.Arguments["symbol"] == "ReadConfig" {
			t.Fatal("invalidated identity retained its former name")
		}
	}
}
