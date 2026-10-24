// WHY: The public Java type-expansion contract must reject edits that alter anything beyond its native var token.
package java

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spockz/semantic-editor/internal/backend"
	"github.com/spockz/semantic-editor/internal/backend/pathutil"
)

func TestJavaMakeTypeExplicitContract(t *testing.T) {
	for _, test := range []struct {
		name, mode string
		wantErr    bool
	}{
		{"primitive", "", false}, {"missing", "missing", true}, {"wrong kind", "kind", true}, {"wrong title", "title", true},
		{"disabled", "disabled", true}, {"command", "command", true}, {"lazy", "data", true}, {"duplicate", "duplicate", true},
		{"null edit", "null", true}, {"foreign file", "foreign", true}, {"resource", "resource", true}, {"annotations", "annotations", true},
		{"annotated edit", "annotationId", true}, {"extra edit", "extra", true}, {"initializer edit", "initializer", true},
		{"different replacement", "replacement", true}, {"non-denotable", "anonymous", true}, {"stale revision", "stale", true},
		{"missing revision", "revision", true}, {"non-var", "explicit", true},
		{"var array", "var[]", true}, {"generic var", "List<var>", true}, {"nested generic var", "Map<String, List<var>>", true},
	} {
		t.Run(test.name, func(t *testing.T) { runJavaMakeTypeExplicitContract(t, test.mode, test.wantErr) })
	}
}

func javaExplicitTypeTestResponse(t *testing.T, uri string, source []byte, selection backend.Range, mode string) json.RawMessage {
	t.Helper()
	typeStart := strings.Index(string(source), "total") - 4
	edit := map[string]any{"range": backend.Range{Start: backend.PositionFromByteOffset(source, typeStart), End: backend.PositionFromByteOffset(source, typeStart+3)}, "newText": "int"}
	changes := map[string]any{uri: []any{edit}}
	workspace := map[string]any{"changes": changes}
	action := map[string]any{"title": "Change type of 'total' to 'int'", "kind": "refactor", "diagnostics": []any{}, "edit": workspace}
	actions := make([]any, 0, 2)
	actions = append(actions, action)
	switch mode {
	case "missing":
		actions = actions[:0]
	case "kind":
		action["kind"] = "quickfix"
	case "title":
		action["title"] = "Change type of 'other' to 'int'"
	case "disabled":
		action["disabled"] = map[string]any{"reason": "not applicable"}
	case "command":
		action["command"] = map[string]any{"command": "java.apply"}
	case "data":
		action["data"] = map[string]any{"resolve": true}
	case "duplicate":
		actions = append(actions, action)
	case "null":
		action["edit"] = nil
	case "foreign":
		delete(changes, uri)
		changes[uri+".foreign"] = []any{edit}
	case "resource":
		workspace["documentChanges"] = []any{map[string]any{"kind": "rename"}}
	case "annotations":
		workspace["changeAnnotations"] = map[string]any{}
	case "annotationId":
		edit["annotationId"] = "hidden"
	case "extra":
		changes[uri] = []any{edit, edit}
	case "initializer":
		edit["range"] = selection
	case "replacement":
		edit["newText"] = "long"
	case "anonymous":
		action["title"] = "Change type of 'total' to 'Object & Runnable'"
		edit["newText"] = "Object & Runnable"

	case "var[]", "List<var>", "Map<String, List<var>>":
		action["title"] = "Change type of 'total' to '" + mode + "'"
		edit["newText"] = mode
	}
	raw, err := json.Marshal(actions)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func runJavaMakeTypeExplicitContract(t *testing.T, mode string, wantErr bool) {
	t.Helper()
	root := backend.CanonicalWorkspaceRoot(t.TempDir())
	file := filepath.Join(root, "Service.java")
	source := []byte("class Service { int count() { var total = 1 + 2; return total; } }\n")
	if mode == "explicit" {
		source = bytes.Replace(source, []byte("var total"), []byte("int total"), 1)
	}
	if err := os.WriteFile(filepath.Join(root, "pom.xml"), []byte("<project/>\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, source, 0o600); err != nil {
		t.Fatal(err)
	}
	uri := pathutil.FileURI(file)
	start := strings.Index(string(source), "total")
	selection := backend.Range{Start: backend.PositionFromByteOffset(source, start), End: backend.PositionFromByteOffset(source, start+5)}
	raw := javaExplicitTypeTestResponse(t, uri, source, selection, mode)
	session := &makeStaticTestSession{selectedURI: uri, response: raw}
	project := backend.ProjectContext{RootDir: root, File: file, Language: backend.LanguageJava, WorkspaceTrust: backend.NewWorkspaceTrust(root, true)}
	fingerprint, err := javaImportFingerprint(root, effectiveJavaConfig(project))
	if err != nil {
		t.Fatal(err)
	}
	service := &JavaBackend{root: root, fingerprint: fingerprint, session: session, documents: make(map[string]javaDocumentState)}
	digest := sha256.Sum256(source)
	revision := hex.EncodeToString(digest[:])
	if mode == "stale" {
		revision = strings.Repeat("0", 64)
	}
	if mode == "revision" {
		revision = ""
	}
	result, err := service.ApplyCodeAction(context.Background(), backend.CodeActionRequest{Operation: javaMakeTypeExplicitOperation, Project: project, File: file, Symbol: "total", EnclosingSymbol: "Service.count()", Range: selection, Kind: "refactor", ExpectedRevision: revision})
	got, readErr := os.ReadFile(file)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if wantErr {
		if err == nil {
			t.Fatalf("accepted unsupported response: %#v", result)
		}
		if !bytes.Equal(got, source) {
			t.Fatalf("refused edit changed bytes: %q", got)
		}
		return
	}
	if err != nil {
		t.Fatal(err)
	}
	want := bytes.Replace(source, []byte("var total"), []byte("int total"), 1)
	if !bytes.Equal(got, want) || result == nil || !bytes.Equal(result.Before, source) || !bytes.Equal(result.After, want) {
		t.Fatalf("result = %#v; disk = %q", result, got)
	}
}
