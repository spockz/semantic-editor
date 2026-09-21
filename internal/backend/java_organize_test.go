package backend

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"testing"
	"time"
)

func TestApplyJavaOrganizeImportsEdit(t *testing.T) {
	file := filepath.Join(t.TempDir(), "Thing.java")
	source := []byte("class Thing {}\n")
	raw := json.RawMessage(`[{"title":"Organize imports","kind":"source.organizeImports","edit":{"changes":{"file:///` + file + `":[{"range":{"start":{"line":0,"character":0},"end":{"line":0,"character":5}},"newText":"class"}]}}}]`)
	updated, err := applyJavaOrganizeImportsEdit(file, source, raw)
	if err != nil || string(updated) != "class Thing {}\n" {
		t.Fatalf("updated=%q err=%v", updated, err)
	}
}

func TestApplyJavaOrganizeImportsEditRejectsUnsafeActions(t *testing.T) {
	file := filepath.Join(t.TempDir(), "Thing.java")
	for name, raw := range map[string]json.RawMessage{
		"wrong-kind": json.RawMessage(`[{"kind":"quickfix","edit":{"changes":{}}}]`),
		"command":    json.RawMessage(`[{"edit":{"changes":{}},"command":{"title":"run"}}]`),
		"disabled":   json.RawMessage(`[{"disabled":"not available","edit":{"changes":{}}}]`),
		"foreign":    json.RawMessage(`[{"edit":{"changes":{"file:///tmp/other.java":[]}}}]`),
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := applyJavaOrganizeImportsEdit(file, []byte("class Thing {}"), raw); !errors.Is(err, ErrJavaRenameInvalidEdit) {
				t.Fatalf("err=%v", err)
			}
		})
	}
}

func TestApplyJavaOrganizeImportsEditAllowsNoAction(t *testing.T) {
	file := filepath.Join(t.TempDir(), "Thing.java")
	source := []byte("class Thing {}")
	updated, err := applyJavaOrganizeImportsEdit(file, source, json.RawMessage(`[]`))
	if err != nil || string(updated) != string(source) {
		t.Fatalf("updated=%q err=%v", updated, err)
	}
}

func TestApplyJavaFormattingEditsAllowsEmptyOrNull(t *testing.T) {
	source := []byte("class Thing {}")
	for _, raw := range []string{"[]", "null"} {
		updated, err := applyJavaFormattingEdits(source, json.RawMessage(raw))
		if err != nil || string(updated) != string(source) {
			t.Fatalf("raw %s: updated=%q err=%v", raw, updated, err)
		}
	}
}

func TestApplyJavaOrganizeImportsRejectsMixedOrUnsafeWorkspaceEdits(t *testing.T) {
	file := filepath.Join(t.TempDir(), "Thing.java")
	uri := fileURI(file)
	base := `{"kind":"source.organizeImports","edit":%s}`
	for name, edit := range map[string]string{
		"mixed":     `{"changes":{"` + uri + `":[]},"documentChanges":[]}`,
		"annotated": `{"changes":{"` + uri + `":[{"range":{"start":{"line":0,"character":0},"end":{"line":0,"character":0}},"newText":"","annotationId":"x"}]}}`,
		"unknown":   `{"documentChanges":[{"textDocument":{"uri":"` + uri + `","version":1},"edits":[],"unsafe":true}]}`,
	} {
		t.Run(name, func(t *testing.T) {
			raw := json.RawMessage(`[ ` + fmt.Sprintf(base, edit) + ` ]`)
			if _, err := applyJavaOrganizeImportsEdit(file, []byte("class Thing {}"), raw); !errors.Is(err, ErrJavaRenameInvalidEdit) {
				t.Fatalf("err=%v", err)
			}
		})
	}
}

func TestJavaProcessSessionDiagnosticsRejectWrongURIAndVersion(t *testing.T) {
	session := &javaProcessSession{diagnostics: map[string]javaDiagnosticReceipt{
		"file:///selected.java": {version: 1, diagnostics: []Diagnostic{{Message: "old"}}},
	}, wake: make(chan struct{}, 1)}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	if _, err := session.WaitDiagnostics(ctx, "file:///other.java", 1); !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, ErrJavaDiagnosticsTimeout) {
		t.Fatalf("wrong URI error = %v", err)
	}
	ctx, cancel = context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	if _, err := session.WaitDiagnostics(ctx, "file:///selected.java", 2); !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, ErrJavaDiagnosticsTimeout) {
		t.Fatalf("stale version error = %v", err)
	}
}

func TestJavaProcessSessionDiagnosticsRetainHighestVersion(t *testing.T) {
	session := &javaProcessSession{diagnostics: map[string]javaDiagnosticReceipt{}, wake: make(chan struct{}, 1)}
	session.recordDiagnostics("file:///selected.java", 4, []Diagnostic{{Message: "final"}})
	session.recordDiagnostics("file:///selected.java", 2, []Diagnostic{{Message: "stale"}})
	got, err := session.WaitDiagnostics(context.Background(), "file:///selected.java", 4)
	if err != nil || len(got) != 1 || got[0].Message != "final" {
		t.Fatalf("diagnostics=%#v err=%v", got, err)
	}
}
