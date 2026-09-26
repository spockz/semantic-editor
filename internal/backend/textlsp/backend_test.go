// Package textlsp tests selected-file lookup, trust, and diagnostics receipt contracts.
package textlsp

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"semedit/internal/backend"
	"semedit/internal/backend/pathutil"
	"semedit/internal/pipeline"
)

type fakeSession struct {
	response json.RawMessage
	opened   string
}

func (f *fakeSession) Request(_ context.Context, method string, _ any) (json.RawMessage, error) {
	if method == "textDocument/documentSymbol" {
		return json.RawMessage(strings.ReplaceAll(string(f.response), "SELECTED_URI", f.opened)), nil
	}
	return json.RawMessage(`{"capabilities":{}}`), nil
}
func (f *fakeSession) Notify(_ context.Context, method string, params any) error {
	if method == "textDocument/didOpen" {
		raw, _ := json.Marshal(params)
		var p struct {
			TextDocument struct {
				URI string `json:"uri"`
			} `json:"textDocument"`
		}
		_ = json.Unmarshal(raw, &p)
		f.opened = p.TextDocument.URI
	}
	return nil
}
func (*fakeSession) WaitDiagnostics(ctx context.Context, _ string, _ int) ([]backend.Diagnostic, error) {
	<-ctx.Done()
	return nil, ctx.Err()
}
func (*fakeSession) Close() error { return nil }

func TestLookupRequiresTrustBeforeStartingServer(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "a.sh")
	if err := pipeline.WriteAtomic(file, []byte("f() {}\n")); err != nil {
		t.Fatal(err)
	}
	started := false
	a := &Adapter{Config: Config{Language: backend.LanguageBash, Extensions: []string{".sh"}}, Factory: func(context.Context, string, string, []string) (Session, error) {
		started = true
		return nil, errors.New("unexpected")
	}}
	_, err := a.Lookup(context.Background(), backend.ProjectContext{RootDir: root, File: file}, "f")
	if !errors.Is(err, backend.ErrWorkspaceTrustRequired) {
		t.Fatalf("Lookup error=%v, want trust required", err)
	}
	if started {
		t.Fatal("server started before request trust")
	}
}

func TestLookupRejectsScratchSymlinkOutsideWorkspace(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	file := filepath.Join(root, "a.sh")
	if err := pipeline.WriteAtomic(file, []byte("f() {}\n")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, ".scratch")); err != nil {
		t.Fatal(err)
	}
	started := false
	a := &Adapter{Config: Config{Language: backend.LanguageBash, Extensions: []string{".sh"}}, Factory: func(context.Context, string, string, []string) (Session, error) {
		started = true
		return nil, errors.New("unexpected")
	}}
	_, err := a.Lookup(context.Background(), backend.ProjectContext{RootDir: root, File: file, WorkspaceTrust: backend.NewWorkspaceTrust(root, true)}, "f")
	if !errors.Is(err, ErrScratchOutsideRoot) {
		t.Fatalf("Lookup error=%v, want scratch path rejection", err)
	}
	if started {
		t.Fatal("server started with scratch symlink outside workspace")
	}
	entries, err := os.ReadDir(outside)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("outside scratch target contains %d entries", len(entries))
	}
}

func TestLookupUsesSelectedFileAndUTF16Ranges(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "a.sh")
	source := "# 😀f\n"
	if err := pipeline.WriteAtomic(file, []byte(source)); err != nil {
		t.Fatal(err)
	}
	session := &fakeSession{response: json.RawMessage(`[{"name":"f","kind":12,"range":{"start":{"line":0,"character":0},"end":{"line":0,"character":5}},"selectionRange":{"start":{"line":0,"character":4},"end":{"line":0,"character":5}}}]`)}
	a := &Adapter{Config: Config{Language: backend.LanguageBash, Extensions: []string{".sh"}, LanguageID: "shell"}, Factory: func(context.Context, string, string, []string) (Session, error) { return session, nil }}
	project := backend.ProjectContext{RootDir: root, File: file, WorkspaceTrust: backend.NewWorkspaceTrust(root, true)}
	got, err := a.Lookup(context.Background(), project, "f")
	if err != nil {
		t.Fatal(err)
	}
	canonicalFile, err := filepath.EvalSymlinks(file)
	if err != nil {
		t.Fatal(err)
	}
	if got.Offset != 6 || got.Line != 1 || got.Column != 5 || got.Location.URI != pathutil.FileURI(canonicalFile) {
		t.Fatalf("lookup result=%+v", got)
	}
	if got.File != "a.sh" {
		t.Fatalf("result file=%q, want selected path relative to root", got.File)
	}
}

func TestReadSymbolProjectionContracts(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "Makefile")
	source := []byte("all:\n\t@echo one\n\t@echo two\nnext:\n\t@echo next\n")
	if err := pipeline.WriteAtomic(file, source); err != nil {
		t.Fatal(err)
	}
	response := json.RawMessage(`[{"name":"all","kind":12,"range":{"start":{"line":0,"character":0},"end":{"line":2,"character":10}},"selectionRange":{"start":{"line":0,"character":0},"end":{"line":0,"character":3}}}]`)
	session := &fakeSession{response: response}
	adapter := &Adapter{
		Config:  Config{Language: backend.LanguageMake, Basenames: []string{"Makefile"}, LanguageID: "makefile", SymbolSyntax: "make"},
		Factory: func(context.Context, string, string, []string) (Session, error) { return session, nil },
	}
	project := backend.ProjectContext{RootDir: root, File: file, WorkspaceTrust: backend.NewWorkspaceTrust(root, true)}
	inspected, err := adapter.Inspect(context.Background(), backend.InspectRequest{Project: project, Symbol: "all"})
	if err != nil {
		t.Fatal(err)
	}
	if len(inspected.Matches) != 1 || inspected.Matches[0].Source != "all:\n\t@echo one\n\t@echo two" || inspected.Matches[0].SourceExtent != "server_range" {
		t.Fatalf("Inspect matches=%+v", inspected.Matches)
	}
	if inspected.Scope.Complete || len(inspected.Scope.Limitations) == 0 {
		t.Fatalf("Make inspect scope=%+v, want incomplete limitations", inspected.Scope)
	}
	outlined, err := adapter.Outline(context.Background(), backend.OutlineRequest{Project: project, Path: "Makefile", Kinds: []string{"target"}, IncludeUnexported: true})
	if err != nil {
		t.Fatal(err)
	}
	symbols := outlined.Files[0].Symbols
	if len(symbols) != 1 || symbols[0].Range.End.Line != 2 || symbols[0].Range.End.Character != 10 {
		t.Fatalf("Make outline symbols=%+v", symbols)
	}
	included := json.RawMessage(`[{"name":"external","kind":12,"range":{"start":{"line":0,"character":0},"end":{"line":0,"character":19}},"selectionRange":{"start":{"line":0,"character":8},"end":{"line":0,"character":16}}}]`)
	mapped, limitations, err := decodeReadSymbols(included, []byte("include external.mk\n"), "make", "uri", file)
	if err != nil || len(mapped) != 0 || len(limitations) == 0 {
		t.Fatalf("include mapping symbols=%+v limitations=%v error=%v", mapped, limitations, err)
	}
	conditional := json.RawMessage(`[{"name":"ifeq ($(MODE),debug)","kind":3,"range":{"start":{"line":0,"character":0},"end":{"line":2,"character":5}},"selectionRange":{"start":{"line":0,"character":0},"end":{"line":2,"character":5}}}]`)
	mapped, limitations, err = decodeReadSymbols(conditional, []byte("ifeq ($(MODE),debug)\nFLAGS := -g\nendif\n"), "make", "uri", file)
	if err != nil || len(mapped) != 0 || len(limitations) == 0 {
		t.Fatalf("conditional mapping symbols=%+v limitations=%v error=%v", mapped, limitations, err)
	}
	unknown := json.RawMessage(`[{"name":"unexpected","kind":2,"range":{"start":{"line":0,"character":0},"end":{"line":0,"character":1}},"selectionRange":{"start":{"line":0,"character":0},"end":{"line":0,"character":1}}}]`)
	if _, _, err := decodeReadSymbols(unknown, []byte("X\n"), "make", "uri", file); !errors.Is(err, ErrMalformedSymbols) {
		t.Fatalf("unexpected Make kind error=%v", err)
	}
	foreignBash := json.RawMessage(`[{"name":"f","kind":12,"location":{"uri":"foreign","range":{"start":{"line":0,"character":0},"end":{"line":0,"character":1}}}}]`)
	if _, _, err := decodeReadSymbols(foreignBash, []byte("f\n"), "bash", "selected", filepath.Join(root, "f.sh")); !errors.Is(err, ErrMalformedSymbols) {
		t.Fatalf("foreign Bash URI error=%v", err)
	}
}

func TestMakeReadRejectsRecipeContinuationBeforeServer(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "Makefile")
	source := []byte("all:\n\t@echo one \\\n\t  two\n")
	if err := pipeline.WriteAtomic(file, source); err != nil {
		t.Fatal(err)
	}
	started := false
	adapter := &Adapter{
		Config: Config{Language: backend.LanguageMake, Basenames: []string{"Makefile"}, LanguageID: "makefile", SymbolSyntax: "make"},
		Factory: func(context.Context, string, string, []string) (Session, error) {
			started = true
			return &fakeSession{}, nil
		},
	}
	project := backend.ProjectContext{RootDir: root, File: file, WorkspaceTrust: backend.NewWorkspaceTrust(root, true)}
	_, err := adapter.Outline(context.Background(), backend.OutlineRequest{Project: project, Path: "Makefile", IncludeUnexported: true})
	if err == nil || !strings.Contains(err.Error(), "continued declaration") {
		t.Fatalf("Outline error=%v, want continuation rejection", err)
	}
	if started {
		t.Fatal("language server started for a document with an unmappable continuation")
	}
}

func TestMakeReadDecoderValidatesEverySupportedRecord(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "Makefile")
	raw := json.RawMessage(`[
		{"name":"all","kind":12,"range":{"start":{"line":0,"character":0},"end":{"line":0,"character":4}},"selectionRange":{"start":{"line":0,"character":0},"end":{"line":0,"character":3}}},
		{"name":"not-a-name","kind":12,"range":{"start":{"line":1,"character":0},"end":{"line":1,"character":4}},"selectionRange":{"start":{"line":1,"character":0},"end":{"line":1,"character":3}}}
	]`)
	_, _, err := decodeReadSymbols(raw, []byte("all:\nbad:\n"), "make", "uri", file)
	if !errors.Is(err, ErrMalformedSymbols) {
		t.Fatalf("unmatched malformed target error=%v", err)
	}
}

func TestMakeReadRejectsForeignURIOnOmittedRecord(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "Conditional.mk")
	raw := json.RawMessage(`[{"name":"ifeq ($(MODE),debug)","kind":3,"uri":"file:///scratch/sibling.mk","range":{"start":{"line":0,"character":0},"end":{"line":2,"character":5}},"selectionRange":{"start":{"line":0,"character":0},"end":{"line":2,"character":5}}}]`)
	_, _, err := decodeReadSymbols(raw, []byte("ifeq ($(MODE),debug)\nFLAGS := -g\nendif\n"), "make", "file:///scratch/Conditional.mk", file)
	if !errors.Is(err, ErrMalformedSymbols) {
		t.Fatalf("foreign top-level URI error=%v", err)
	}
}

func TestMakeReadRejectsForeignURIInOmittedChild(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "Conditional.mk")
	raw := json.RawMessage(`[{"name":"ifeq ($(MODE),debug)","kind":3,"range":{"start":{"line":0,"character":0},"end":{"line":2,"character":5}},"selectionRange":{"start":{"line":0,"character":0},"end":{"line":2,"character":5}},"children":[{"name":"all","kind":12,"uri":"file:///scratch/sibling.mk","range":{"start":{"line":1,"character":0},"end":{"line":1,"character":4}},"selectionRange":{"start":{"line":1,"character":0},"end":{"line":1,"character":3}}}]}]`)
	_, _, err := decodeReadSymbols(raw, []byte("ifeq ($(MODE),debug)\nall:\nendif\n"), "make", "file:///scratch/Conditional.mk", file)
	if !errors.Is(err, ErrMalformedSymbols) {
		t.Fatalf("foreign child URI error=%v", err)
	}
}

func TestBashReadOutlineUsesCROnlyPhysicalLines(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "source.sh")
	source := []byte("# header\rf() {}\r")
	if err := pipeline.WriteAtomic(file, source); err != nil {
		t.Fatal(err)
	}
	session := &fakeSession{response: json.RawMessage(`[{"name":"f","kind":12,"location":{"uri":"SELECTED_URI","range":{"start":{"line":1,"character":0},"end":{"line":1,"character":6}}}}]`)}
	adapter := &Adapter{
		Config:  Config{Language: backend.LanguageBash, Extensions: []string{".sh"}, LanguageID: "shell", SymbolSyntax: "bash"},
		Factory: func(context.Context, string, string, []string) (Session, error) { return session, nil },
	}
	project := backend.ProjectContext{RootDir: root, WorkspaceTrust: backend.NewWorkspaceTrust(root, true)}
	result, err := adapter.Outline(context.Background(), backend.OutlineRequest{Project: project, Path: "source.sh", IncludeUnexported: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Files) != 1 || len(result.Files[0].Symbols) != 1 || result.Files[0].Symbols[0].Name != "f" || result.Files[0].Symbols[0].Range.Start.Line != 1 {
		t.Fatalf("CR-only Bash outline=%+v", result)
	}
}

func TestMakeReadRejectsEvenBackslashContinuationBeforeServer(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "Makefile")
	source := []byte("FLAG = first " + strings.Repeat(string(byte(92)), 2) + "\nFLAG = second\n")
	if err := pipeline.WriteAtomic(file, source); err != nil {
		t.Fatal(err)
	}
	started := false
	adapter := &Adapter{
		Config: Config{Language: backend.LanguageMake, Basenames: []string{"Makefile"}, LanguageID: "makefile", SymbolSyntax: "make"},
		Factory: func(context.Context, string, string, []string) (Session, error) {
			started = true
			return &fakeSession{}, nil
		},
	}
	project := backend.ProjectContext{RootDir: root, File: file, WorkspaceTrust: backend.NewWorkspaceTrust(root, true)}
	_, err := adapter.Inspect(context.Background(), backend.InspectRequest{Project: project, Symbol: "FLAG"})
	if err == nil || !strings.Contains(err.Error(), "continued declaration") {
		t.Fatalf("Inspect error=%v, want continuation rejection", err)
	}
	if started {
		t.Fatal("language server started after a trailing backslash continuation")
	}
}
