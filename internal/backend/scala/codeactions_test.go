// Package scala tests the bounded extraction grammar and rejects unvalidated Metals edits.
// WHY: Source-shape and edit-envelope checks are the safety boundary before Scala changes are staged.
package scala

import (
	"context"
	"encoding/json"
	"maps"
	"strings"
	"testing"

	"github.com/spockz/semantic-editor/internal/backend/pathutil"
)

func TestValidateScalaExtractShapeRejectsShadowedAndUnrelatedBindings(t *testing.T) {
	source := []byte("object Thing {\n  def run(): Int = consume(1 + 2)\n  def consume(value: Int): Int = value\n}\n")
	start := strings.Index(string(source), "1 + 2")
	shape, err := validateScalaExtractShape(source, "run", "1 + 2", start, start+5)
	if err != nil || shape.callee != "consume" {
		t.Fatalf("valid shape = %#v, err %v", shape, err)
	}

	for name, unsafe := range map[string]string{
		"owner parameter shadows callee":    "object Thing {\n  def run(consume: Byte => Int): Int = consume(1 + 2)\n  def consume(value: Int): Int = value\n}\n",
		"unrelated object provides callee":  "object Other { def consume(value: Int): Int = value }\nobject Thing {\n  def run(): Int = consume(1 + 2)\n}\n",
		"extra import can alter resolution": "import conversions.given\nobject Thing {\n  def run(): Int = consume(1 + 2)\n  def consume(value: Int): Int = value\n}\n",
		"callee body adds a declaration":    "object Thing {\n  def run(): Int = consume(1 + 2)\n  def consume(value: Int): Int = value; type Int = Byte\n}\n",
	} {
		t.Run(name, func(t *testing.T) {
			start := strings.Index(unsafe, "1 + 2")
			if _, err := validateScalaExtractShape([]byte(unsafe), "run", "1 + 2", start, start+5); err == nil {
				t.Fatal("unsafe source shape was accepted")
			}
		})
	}
}

func TestDecodeScalaExtractEditsRejectsAdditionalRepresentationsAndAnnotations(t *testing.T) {
	source := []byte("object Thing {\n  def run(): Int = consume(1 + 2)\n  def consume(value: Int): Int = value\n}\n")
	start := strings.Index(string(source), "1 + 2")
	shape, err := validateScalaExtractShape(source, "run", "1 + 2", start, start+5)
	if err != nil {
		t.Fatal(err)
	}
	file := "/workspace/Thing.scala"
	uri := pathutil.FileURI(file)
	validEdits := []map[string]any{
		{"range": scalaRange{Start: scalaPositionAt(source, shape.start), End: scalaPositionAt(source, shape.end)}, "newText": "newValue"},
		{"range": scalaRange{Start: scalaPositionAt(source, shape.openStart), End: scalaPositionAt(source, shape.openEnd)}, "newText": " {\n    val newValue = 1 + 2\n    "},
		{"range": scalaRange{Start: scalaPositionAt(source, shape.callEnd), End: scalaPositionAt(source, shape.callEnd)}, "newText": "\n  }"},
	}
	encode := func(value any) []byte {
		t.Helper()
		raw, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		return raw
	}
	if _, err := decodeScalaExtractEdits(file, source, encode(map[string]any{"changes": map[string]any{uri: validEdits}}), shape, "1 + 2"); err != nil {
		t.Fatalf("valid three-edit response rejected: %v", err)
	}
	annotated := append([]map[string]any(nil), validEdits...)
	annotated[0] = map[string]any{"range": validEdits[0]["range"], "newText": "newValue", "annotationId": "confirm"}
	for name, envelope := range map[string]map[string]any{
		"mixed document changes":  {"changes": map[string]any{uri: validEdits}, "documentChanges": []any{map[string]any{"textDocument": map[string]any{"uri": "file:///workspace/Other.scala", "version": 1}, "edits": []any{}}}},
		"change annotations":      {"changes": map[string]any{uri: validEdits}, "changeAnnotations": map[string]any{"confirm": map[string]any{"label": "Confirm"}}},
		"unknown resource change": {"changes": map[string]any{uri: validEdits}, "resourceOperations": []any{}},
		"edit annotation":         {"changes": map[string]any{uri: annotated}},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := decodeScalaExtractEdits(file, source, encode(envelope), shape, "1 + 2"); err == nil {
				t.Fatal("unsupported workspace edit was accepted")
			}
		})
	}
}

type renameEnvelopeSession struct{ raw json.RawMessage }

func (s renameEnvelopeSession) Request(_ context.Context, method string, params any) (json.RawMessage, error) {
	if method == "textDocument/rename" {
		return s.raw, nil
	}
	request, _ := json.Marshal(params)
	var decoded struct {
		Position scalaPosition `json:"position"`
	}
	if err := json.Unmarshal(request, &decoded); err != nil {
		return nil, err
	}
	return json.Marshal(map[string]any{"start": decoded.Position, "end": scalaPosition{Line: decoded.Position.Line, Character: decoded.Position.Character + len("newValue")}})
}

func (renameEnvelopeSession) Notify(context.Context, string, any) error { return nil }
func (renameEnvelopeSession) Close() error                              { return nil }

func TestRenameScalaExtractedLocalValidatesVersionAndCompleteEnvelope(t *testing.T) {
	candidate := []byte("val newValue = 1 + 2\nconsume(newValue)\n")
	bindingStart := strings.Index(string(candidate), "newValue")
	useStart := strings.LastIndex(string(candidate), "newValue")
	uri := "file:///workspace/Thing.scala"
	edits := []scalaEdit{
		{Range: scalaRange{Start: scalaPositionAt(candidate, bindingStart), End: scalaPositionAt(candidate, bindingStart+len("newValue"))}, NewText: "total"},
		{Range: scalaRange{Start: scalaPositionAt(candidate, useStart), End: scalaPositionAt(candidate, useStart+len("newValue"))}, NewText: "total"},
	}
	workspaceEdit := func(version int, extras map[string]any) json.RawMessage {
		documentChange := map[string]any{"textDocument": map[string]any{"uri": uri, "version": version}, "edits": edits}
		object := map[string]any{"documentChanges": []any{documentChange}}
		maps.Copy(object, extras)
		raw, err := json.Marshal(object)
		if err != nil {
			t.Fatal(err)
		}
		return raw
	}
	for name, raw := range map[string]json.RawMessage{
		"valid":         workspaceEdit(2, nil),
		"stale version": workspaceEdit(1, nil),
		"mixed changes": workspaceEdit(2, map[string]any{"changes": map[string]any{uri: []scalaEdit{}}}),
		"annotations":   workspaceEdit(2, map[string]any{"changeAnnotations": map[string]any{"confirm": map[string]any{"label": "Confirm"}}}),
	} {
		t.Run(name, func(t *testing.T) {
			updated, err := renameScalaExtractedLocal(t.Context(), renameEnvelopeSession{raw: raw}, uri, candidate, "newValue", "total", bindingStart, useStart)
			if name == "valid" {
				if err != nil || !strings.Contains(string(updated), "consume(total)") || !strings.Contains(string(updated), "val total") {
					t.Fatalf("valid rename = %q, err %v", updated, err)
				}
				return
			}
			if err == nil {
				t.Fatal("invalid rename workspace edit was accepted")
			}
		})
	}
}

func TestDecodeScalaExtractMethodEditsValidatesMethodLocalShape(t *testing.T) {
	source := []byte("object Thing {\n  def run(): Unit = {\n    consume(1 + 2)\n  }\n}\n")
	selected := scalaRange{Start: scalaPosition{Line: 2, Character: 4}, End: scalaPosition{Line: 2, Character: 18}}
	uri := pathutil.FileURI("/workspace/Thing.scala")
	edits := []map[string]any{
		{"range": selected, "newText": "newMethod()"},
		{"range": scalaRange{Start: selected.Start, End: selected.Start}, "newText": "def newMethod(): Unit =\n      consume(1 + 2)\n\n    "},
	}
	encode := func(value any) json.RawMessage {
		t.Helper()
		raw, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		return raw
	}
	raw := encode(map[string]any{"changes": map[string]any{uri: edits}})
	decoded, generated, err := decodeScalaExtractMethodEdits("/workspace/Thing.scala", source, raw, selected, "consume(1 + 2)", "    ")
	if err != nil || generated != "newMethod" || len(decoded) != 2 {
		t.Fatalf("decode valid method extraction = (%#v, %q), err %v", decoded, generated, err)
	}
	annotated := append([]map[string]any(nil), edits...)
	annotated[0] = map[string]any{"range": selected, "newText": "newMethod()", "annotationId": "confirm"}
	for name, workspaceEdit := range map[string]map[string]any{
		"foreign file":            {"changes": map[string]any{uri: edits, "file:///workspace/Other.scala": []any{}}},
		"mixed document changes":  {"changes": map[string]any{uri: edits}, "documentChanges": []any{}},
		"change annotations":      {"changes": map[string]any{uri: edits}, "changeAnnotations": map[string]any{}},
		"unknown resource change": {"changes": map[string]any{uri: edits}, "resourceOperations": []any{}},
		"edit annotation":         {"changes": map[string]any{uri: annotated}},
		"extra parameter": {"changes": map[string]any{uri: []map[string]any{
			{"range": selected, "newText": "newMethod(value)"},
			{"range": scalaRange{Start: selected.Start, End: selected.Start}, "newText": "def newMethod(value: Int): Unit =\n      consume(value)\n\n    "},
		}}},
	} {
		t.Run(name, func(t *testing.T) {
			if _, _, err := decodeScalaExtractMethodEdits("/workspace/Thing.scala", source, encode(workspaceEdit), selected, "consume(1 + 2)", "    "); err == nil {
				t.Fatal("unsafe method extraction edit was accepted")
			}
		})
	}
	t.Run("trailing JSON", func(t *testing.T) {
		if _, _, err := decodeScalaExtractMethodEdits("/workspace/Thing.scala", source, json.RawMessage(string(raw)+" {}"), selected, "consume(1 + 2)", "    "); err == nil {
			t.Fatal("workspace edit with trailing JSON was accepted")
		}
	})
}

func TestScalaCallableSelectionRequiresOneCompleteDirectCall(t *testing.T) {
	for _, statement := range []string{"consume()", "consume(1 + 2)", `consume("value")`} {
		if !scalaCallableStatement(statement) {
			t.Errorf("scalaCallableStatement(%q) = false", statement)
		}
	}
	for _, statement := range []string{"consume(value)", "obj.consume()", "consume(if (ready) 1 else 2)", "return consume()", "consume(1); cleanup()"} {
		if scalaCallableStatement(statement) {
			t.Errorf("scalaCallableStatement(%q) = true", statement)
		}
	}
	source := []byte("object Thing {\n  def run(): Unit = {\n    consume()\n  }\n}\n")
	start := strings.Index(string(source), "consume()")
	if !scalaCallableSelectionIsWholeLine(source, start, start+len("consume()")) {
		t.Fatal("complete direct call was rejected")
	}
	if scalaCallableSelectionIsWholeLine(source, start+1, start+len("consume()")) {
		t.Fatal("partial call selection was accepted")
	}
}
