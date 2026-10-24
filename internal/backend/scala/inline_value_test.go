// Package scala verifies the fail-closed inline-value source and workspace-edit contract.
// WHY: Metals supplies an unversioned asynchronous edit, so each accepted edit must equal one independently known result.
package scala

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/spockz/semantic-editor/internal/backend"
	"github.com/spockz/semantic-editor/internal/backend/pathutil"
)

func TestScalaInlineValueShapeAndWorkspaceEdit(t *testing.T) {
	source := []byte("object Thing {\n  def run(): Unit = {\n    val value = 42\n    consume(value, value)\n  }\n  def consume(first: Int, second: Int): Unit = ()\n}\n")
	request := backend.CodeActionRequest{Symbol: "value", EnclosingSymbol: "run"}
	shape, err := validateScalaInlineShape(source, request)
	if err != nil {
		t.Fatal(err)
	}
	want := []byte("object Thing {\n  def run(): Unit = {\n    consume(42, 42)\n  }\n  def consume(first: Int, second: Int): Unit = ()\n}\n")
	postimage, err := deriveScalaInlinePostimage(source, shape)
	if err != nil || string(postimage) != string(want) {
		t.Fatalf("independent postimage = %q, err %v", postimage, err)
	}
	uri := pathutil.FileURI("/workspace/Thing.scala")
	edit := map[string]any{"changes": map[string]any{uri: []map[string]any{
		{"range": shape.delete, "newText": ""},
		{"range": shape.uses[0], "newText": "42"},
		{"range": shape.uses[1], "newText": "42"},
	}}}
	raw, err := json.Marshal(edit)
	if err != nil {
		t.Fatal(err)
	}
	actual, err := decodeScalaInlineEdit("/workspace/Thing.scala", source, raw, shape)
	if err != nil || string(actual) != string(want) {
		t.Fatalf("captured postimage = %q, err %v", actual, err)
	}
	for name, invalid := range map[string]map[string]any{
		"foreign URI": {"changes": map[string]any{"file:///workspace/Other.scala": []map[string]any{
			{"range": shape.delete, "newText": ""},
			{"range": shape.uses[0], "newText": "42"},
			{"range": shape.uses[1], "newText": "42"},
		}}},
		"versioned edits":  {"documentChanges": []any{}},
		"extra occurrence": {"changes": map[string]any{uri: []map[string]any{{"range": shape.delete, "newText": ""}}}},
		"wrong replacement": {"changes": map[string]any{uri: []map[string]any{
			{"range": shape.delete, "newText": ""},
			{"range": shape.uses[0], "newText": "43"},
			{"range": shape.uses[1], "newText": "42"},
		}}},
	} {
		t.Run(name, func(t *testing.T) {
			encoded, marshalErr := json.Marshal(invalid)
			if marshalErr != nil {
				t.Fatal(marshalErr)
			}
			if _, decodeErr := decodeScalaInlineEdit("/workspace/Thing.scala", source, encoded, shape); decodeErr == nil {
				t.Fatal("invalid Metals edit was accepted")
			}
		})
	}
}

func TestScalaInlineValueAdvertisesOnlyTheBoundedAction(t *testing.T) {
	t.Parallel()
	scalaBackend := NewScalaBackend()
	if scalaBackend.Capabilities().Supports(backend.OperationInlineVariable) {
		t.Fatal("Scala inline-value capability is advertised without a live Metals action and captured edit")
	}
}

func TestSelectScalaInlineCommandRequiresExactActionAndTarget(t *testing.T) {
	uri := "file:///workspace/Thing.scala"
	position := scalaPosition{Line: 2, Character: 8}
	valid := []any{map[string]any{
		"title": "Inline `value`", "kind": "refactor.inline",
		"command": map[string]any{"title": "Inline value", "command": "inline-value", "arguments": []any{map[string]any{
			"textDocument": map[string]string{"uri": uri}, "position": position,
		}}},
	}}
	encode := func(value any) json.RawMessage {
		t.Helper()
		raw, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		return raw
	}
	if command, err := selectScalaInlineCommand(encode(valid), "value", uri, position); err != nil || command.Command != "inline-value" {
		t.Fatalf("valid command = %#v, err %v", command, err)
	}
	extra := []any{map[string]any{
		"title": "Inline `value`", "kind": "refactor.inline",
		"command": map[string]any{"title": "Inline value", "command": "inline-value", "arguments": []any{}, "unexpected": true},
	}}
	if _, err := selectScalaInlineCommand(encode(extra), "value", uri, position); err == nil {
		t.Fatal("command with an extra field was accepted")
	}
	wrongTarget := []any{map[string]any{
		"title": "Inline `value`", "kind": "refactor.inline",
		"command": map[string]any{"title": "Inline value", "command": "inline-value", "arguments": []any{map[string]any{
			"textDocument": map[string]string{"uri": uri}, "position": scalaPosition{Line: 2, Character: 9},
		}}},
	}}
	if _, err := selectScalaInlineCommand(encode(wrongTarget), "value", uri, position); err == nil {
		t.Fatal("command for a different source position was accepted")
	}
}

func TestScalaInlineValueShapeRefusesUnsafeUseSites(t *testing.T) {
	base := "object Thing {\n  def run(): Unit = {\n    val value = 42\n    consume(value, value)\n  }\n  def consume(first: Int, second: Int): Unit = ()\n}\n"
	for name, source := range map[string]string{
		"mutable":           strings.Replace(base, "val value", "var value", 1),
		"explicit type":     strings.Replace(base, "val value =", "val value: Int =", 1),
		"extra use":         strings.Replace(base, "consume(value, value)", "consume(value, value)\n    println(value)", 1),
		"non Int parameter": strings.Replace(base, "second: Int", "second: Long", 1),
		"extra method":      strings.Replace(base, "}\n", "  def other(): Unit = ()\n}\n", 1),
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := validateScalaInlineShape([]byte(source), backend.CodeActionRequest{Symbol: "value", EnclosingSymbol: "run"}); err == nil {
				t.Fatal("unsafe Scala inline shape was accepted")
			}
		})
	}
}
