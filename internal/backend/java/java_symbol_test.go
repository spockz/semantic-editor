// WHY: Pins Java overload normalization and signature-qualified addressing to the backend selector contract.
package java

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spockz/semantic-editor/internal/backend/pathutil"
)

func TestSelectJavaSymbolReportsAndResolvesOverloads(t *testing.T) {
	symbols := []javaDocumentSymbol{{
		Name: "OrderService", Kind: 5, Children: []javaDocumentSymbol{
			{Name: "process", Kind: 6, Detail: "Order process(Order order)"},
			{Name: "process", Kind: 6, Detail: "Order process(Order order, boolean retry)"},
		},
	}}

	ambiguous, err := selectJavaSymbol("OrderService.process", "/tmp/OrderService.java", symbols)
	if err != nil {
		t.Fatalf("select overloads: %v", err)
	}
	if len(ambiguous.Candidates) != 2 {
		t.Fatalf("candidate count = %d, want 2", len(ambiguous.Candidates))
	}
	if got, want := ambiguous.Candidates[0].QualifiedName, "OrderService.process(Order)"; got != want {
		t.Fatalf("first qualified name = %q, want %q", got, want)
	}
	if got, want := ambiguous.Candidates[1].QualifiedName, "OrderService.process(Order, boolean)"; got != want {
		t.Fatalf("second qualified name = %q, want %q", got, want)
	}

	selected, err := selectJavaSymbol("OrderService.process(Order, boolean)", "/tmp/OrderService.java", symbols)
	if err != nil {
		t.Fatalf("select exact overload: %v", err)
	}
	if selected.Definition == nil || selected.Definition.QualifiedName != "OrderService.process(Order, boolean)" {
		t.Fatalf("selected definition = %#v", selected.Definition)
	}
}

func TestMapJavaReadSymbolsUsesErasedSignatures(t *testing.T) {
	symbols, err := mapJavaReadSymbols([]javaDocumentSymbol{{
		Name: "Widget", Kind: 5, Children: []javaDocumentSymbol{
			{Name: "convert", Kind: 6, Detail: "java.util.List<String> convert(java.util.List<String> input)"},
			{Name: "convert", Kind: 6, Detail: "String convert(int value)"},
		},
	}}, nil, "/tmp/Widget.java")
	if err != nil {
		t.Fatalf("map Java symbols: %v", err)
	}
	methods := symbols[0].Children
	if got, want := methods[0].QualifiedName, "Widget.convert(List)"; got != want {
		t.Fatalf("first qualified name = %q, want %q", got, want)
	}
	if methods[0].Signature == nil || *methods[0].Signature != "List" {
		t.Fatalf("first signature = %#v", methods[0].Signature)
	}
	if got, want := methods[1].QualifiedName, "Widget.convert(int)"; got != want {
		t.Fatalf("second qualified name = %q, want %q", got, want)
	}
}

func TestMapJavaReadSymbolsSupportsConstantsAndEnumMembers(t *testing.T) {
	symbols, err := mapJavaReadSymbols([]javaDocumentSymbol{{
		Name: "Ledger", Kind: 5, Children: []javaDocumentSymbol{
			{Name: "MAX_ENTRIES", Kind: 14},
			{Name: "DEBIT", Kind: 22},
		},
	}}, nil, "/tmp/Ledger.java")
	if err != nil {
		t.Fatalf("map Java constants: %v", err)
	}
	for _, symbol := range symbols[0].Children {
		if symbol.Kind != "field" {
			t.Fatalf("Java member %q kind = %q, want field", symbol.Name, symbol.Kind)
		}
	}
}

func TestParseJavaSymbolQueryAllowsZeroArgumentSignature(t *testing.T) {
	parts, signature, specified, err := parseJavaSymbolQuery("Widget.initialize()")
	if err != nil || !specified || signature != "" || len(parts) != 2 || parts[1] != "initialize" {
		t.Fatalf("parsed query = %#v %q %t %v", parts, signature, specified, err)
	}
}

func TestPrepareJavaWorkspaceEditsValidatesAndAppliesMultipleFiles(t *testing.T) {
	root := t.TempDir()
	selected := filepath.Join(root, "Widget.java")
	usage := filepath.Join(root, "Use.java")
	selectedSource := []byte("class Widget {}\n")
	usageSource := []byte("class Use { Widget value; }\n")
	if err := os.WriteFile(selected, selectedSource, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(usage, usageSource, 0o600); err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(map[string]any{"changes": map[string]any{
		pathutil.FileURI(selected): []any{map[string]any{"range": javaRange{Start: javaPosition{Character: 6}, End: javaPosition{Character: 12}}, "newText": "Gadget"}},
		pathutil.FileURI(usage):    []any{map[string]any{"range": javaRange{Start: javaPosition{Character: 12}, End: javaPosition{Character: 18}}, "newText": "Gadget"}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	edit, err := prepareJavaWorkspaceEdits(context.Background(), root, selected, selectedSource, raw, "Widget", "Gadget", true)
	if err != nil {
		t.Fatalf("prepare multi-file edit: %v", err)
	}
	if len(edit.files) != 2 {
		t.Fatalf("prepared file count = %d, want 2", len(edit.files))
	}
	updated := map[string]string{edit.files[0].file: string(edit.files[0].after), edit.files[1].file: string(edit.files[1].after)}
	if updated[canonicalJavaFilePath(selected)] != "class Gadget {}\n" || updated[canonicalJavaFilePath(usage)] != "class Use { Gadget value; }\n" {
		t.Fatalf("prepared contents = %#v", updated)
	}
}

func TestPrepareJavaWorkspaceEditsAcceptsVersionedTextDocumentEdits(t *testing.T) {
	root := t.TempDir()
	selected := filepath.Join(root, "Widget.java")
	source := []byte("class Widget {}\n")
	if err := os.WriteFile(selected, source, 0o600); err != nil {
		t.Fatal(err)
	}
	version := 4
	raw, err := json.Marshal(map[string]any{"documentChanges": []any{map[string]any{
		"textDocument": map[string]any{"uri": pathutil.FileURI(selected), "version": version},
		"edits":        []any{map[string]any{"range": javaRange{Start: javaPosition{Character: 6}, End: javaPosition{Character: 12}}, "newText": "Gadget"}},
	}}})
	if err != nil {
		t.Fatal(err)
	}
	edit, err := prepareJavaWorkspaceEdits(context.Background(), root, selected, source, raw, "Widget", "Gadget", true)
	if err != nil {
		t.Fatalf("prepare versioned edit: %v", err)
	}
	if len(edit.files) != 1 || edit.files[0].version == nil || *edit.files[0].version != version || string(edit.files[0].after) != "class Gadget {}\n" {
		t.Fatalf("prepared versioned edit = %#v", edit.files)
	}
}

func TestPrepareJavaWorkspaceEditsValidatesPublicTypeRename(t *testing.T) {
	root := t.TempDir()
	selected := filepath.Join(root, "Widget.java")
	newPath := filepath.Join(root, "Gadget.java")
	source := []byte("public class Widget {}\n")
	if err := os.WriteFile(selected, source, 0o600); err != nil {
		t.Fatal(err)
	}
	oldURI, newURI := pathutil.FileURI(selected), pathutil.FileURI(newPath)
	raw, err := json.Marshal(map[string]any{"documentChanges": []any{
		map[string]any{"textDocument": map[string]any{"uri": oldURI, "version": 1}, "edits": []any{
			map[string]any{"range": javaRange{Start: javaPosition{Character: 13}, End: javaPosition{Character: 19}}, "newText": "Gadget"},
		}},
		map[string]any{"kind": "rename", "oldUri": oldURI, "newUri": newURI, "options": map[string]bool{"overwrite": false}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	edit, err := prepareJavaWorkspaceEdits(context.Background(), root, selected, source, raw, "Widget", "Gadget", true)
	if err != nil {
		t.Fatal(err)
	}
	if len(edit.files) != 1 || len(edit.renames) != 1 || edit.renames[0].oldPath != canonicalJavaFilePath(selected) || edit.renames[0].newPath != canonicalJavaFilePath(newPath) {
		t.Fatalf("prepared Java resource edit = %#v", edit)
	}
	if _, err := prepareJavaWorkspaceEdits(context.Background(), root, selected, source, raw, "Widget", "Gadget", false); !errors.Is(err, ErrJavaBatchResourceRenameUnsupported) {
		t.Fatalf("buffered resource rename error = %v", err)
	}
}

func TestPrepareJavaWorkspaceEditsAcceptsBoundedNonRenameChanges(t *testing.T) {
	root := t.TempDir()
	selected := filepath.Join(root, "Service.java")
	caller := filepath.Join(root, "Caller.java")
	selectedSource := []byte("class Service { void send() {} }\n")
	callerSource := []byte("class Caller { void run() { new Service().send(); } }\n")
	for path, source := range map[string][]byte{selected: selectedSource, caller: callerSource} {
		if err := os.WriteFile(path, source, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	selectedStart := bytes.Index(selectedSource, []byte("send"))
	callerStart := bytes.LastIndex(callerSource, []byte("send"))
	selectedRange := javaRange{Start: javaPosition{Character: selectedStart}, End: javaPosition{Character: selectedStart + len("send")}}
	callerRange := javaRange{Start: javaPosition{Character: callerStart}, End: javaPosition{Character: callerStart + len("send")}}
	raw, err := json.Marshal(map[string]any{"documentChanges": []any{
		map[string]any{"textDocument": map[string]any{"uri": pathutil.FileURI(selected), "version": 1}, "edits": []any{
			map[string]any{"range": selectedRange, "newText": "publish"},
		}},
		map[string]any{"textDocument": map[string]any{"uri": pathutil.FileURI(caller), "version": 1}, "edits": []any{
			map[string]any{"range": callerRange, "newText": "publish"},
		}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	edit, err := prepareJavaWorkspaceEdits(context.Background(), root, selected, selectedSource, raw, "", "", false)
	if err != nil {
		t.Fatal(err)
	}
	if len(edit.files) != 2 || len(edit.renames) != 0 {
		t.Fatalf("prepared workspace edit = %#v, want two text edits and no resource operations", edit)
	}
	updated := make(map[string]string, len(edit.files))
	for _, file := range edit.files {
		updated[filepath.Base(file.file)] = string(file.after)
	}
	if !strings.Contains(updated["Service.java"], "void publish()") || !strings.Contains(updated["Caller.java"], ".publish()") {
		t.Fatalf("prepared edits = %#v, want declaration and caller updated", updated)
	}
}
