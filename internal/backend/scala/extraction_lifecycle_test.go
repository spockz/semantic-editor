// Package scala tests extraction through the public code-action boundary.
// WHY: The tests prove that captured LSP edits follow one safe lifecycle before entering the caller's buffer.
package scala

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/spockz/semantic-editor/internal/backend"
	"github.com/spockz/semantic-editor/internal/backend/pathutil"
	"github.com/spockz/semantic-editor/internal/sourcebuf"
)

type extractionLifecycleSession struct {
	t         *testing.T
	file      string
	source    []byte
	mode      string
	cancel    context.CancelFunc
	events    []string
	candidate []byte
	closed    bool
}

func (s *extractionLifecycleSession) Request(ctx context.Context, method string, params any) (json.RawMessage, error) {
	s.events = append(s.events, "request:"+method)
	switch method {
	case "initialize":
		return json.RawMessage(`{}`), nil
	case "textDocument/codeAction":
		if s.mode == "action" {
			return nil, errors.New("action failed")
		}
		return s.codeActions()
	case "codeAction/resolve":
		if s.mode == "resolve" {
			return nil, errors.New("resolve failed")
		}
		return s.resolveAction()
	case "textDocument/prepareRename":
		var request struct {
			Position scalaPosition `json:"position"`
		}
		s.decode(params, &request)
		return json.Marshal(map[string]any{"start": request.Position, "end": scalaPosition{Line: request.Position.Line, Character: request.Position.Character + len(s.generated())}})
	case "textDocument/rename":
		if s.mode == "rename" {
			return nil, errors.New("rename failed")
		}
		return s.renameAction(params)
	default:
		return json.RawMessage(`{}`), nil
	}
}

func (s *extractionLifecycleSession) Notify(ctx context.Context, method string, params any) error {
	s.events = append(s.events, "notify:"+method)
	switch method {
	case "textDocument/didOpen":
		if s.mode == "didOpen" {
			return errors.New("open failed")
		}
		if s.mode == "cancel" && s.cancel != nil {
			s.cancel()
		}
	case "textDocument/didChange":
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if s.mode == "didChange" {
			return errors.New("change failed")
		}
		var request struct {
			TextDocument struct {
				Version int `json:"version"`
			} `json:"textDocument"`
			ContentChanges []struct {
				Text string `json:"text"`
			} `json:"contentChanges"`
		}
		s.decode(params, &request)
		if request.TextDocument.Version != 2 {
			s.t.Fatalf("didChange version %d, want 2", request.TextDocument.Version)
		}
		s.candidate = []byte(request.ContentChanges[0].Text)
	case "textDocument/didClose":
		encoded, err := json.Marshal(params)
		if err != nil {
			s.t.Fatal(err)
		}
		var envelope map[string]json.RawMessage
		if err := json.Unmarshal(encoded, &envelope); err != nil {
			s.t.Fatal(err)
		}
		document, ok := envelope["textDocument"]
		if len(envelope) != 1 || !ok {
			s.t.Fatalf("didClose envelope keys = %#v, want only textDocument", envelope)
		}
		var textDocument map[string]json.RawMessage
		if err := json.Unmarshal(document, &textDocument); err != nil {
			s.t.Fatal(err)
		}
		rawURI, ok := textDocument["uri"]
		var uri string
		if len(textDocument) != 1 || !ok || json.Unmarshal(rawURI, &uri) != nil || uri != pathutil.FileURI(s.file) {
			s.t.Fatalf("didClose textDocument = %#v, want {uri:%q}", textDocument, pathutil.FileURI(s.file))
		}
		if s.mode == "didClose" {
			return errors.New("close failed")
		}
	}
	return nil
}

func (s *extractionLifecycleSession) Close() error {
	s.closed = true
	s.events = append(s.events, "session:close")
	return nil
}

func (s *extractionLifecycleSession) decode(value, target any) {
	s.t.Helper()
	raw, err := json.Marshal(value)
	if err != nil {
		s.t.Fatal(err)
	}
	if err := json.Unmarshal(raw, target); err != nil {
		s.t.Fatal(err)
	}
}

func (s *extractionLifecycleSession) generated() string {
	if strings.Contains(string(s.source), "1 + 2") {
		return "newValue"
	}
	return "newMethod"
}

func (s *extractionLifecycleSession) codeActions() (json.RawMessage, error) {
	uri := pathutil.FileURI(s.file)
	if strings.Contains(string(s.source), "1 + 2") {
		start := strings.Index(string(s.source), "1 + 2")
		shape, err := validateScalaExtractShape(s.source, "run", "1 + 2", start, start+5)
		if err != nil {
			return nil, err
		}
		edits := []scalaEdit{
			{Range: scalaRange{Start: scalaPositionAt(s.source, shape.start), End: scalaPositionAt(s.source, shape.end)}, NewText: "newValue"},
			{Range: scalaRange{Start: scalaPositionAt(s.source, shape.openStart), End: scalaPositionAt(s.source, shape.openEnd)}, NewText: " {\n    val newValue = 1 + 2\n    "},
			{Range: scalaRange{Start: scalaPositionAt(s.source, shape.callEnd), End: scalaPositionAt(s.source, shape.callEnd)}, NewText: "\n  }"},
		}
		edit, err := json.Marshal(map[string]any{"changes": map[string]any{uri: edits}})
		if err != nil {
			return nil, err
		}
		return json.Marshal([]scalaAction{{Title: "Extract `1 + 2` as value", Kind: scalaRefactorExtract, Edit: edit}})
	}
	start := strings.Index(string(s.source), "consume(1)")
	selected := scalaRange{Start: scalaPositionAt(s.source, start), End: scalaPositionAt(s.source, start+len("consume(1)"))}
	data, err := json.Marshal(map[string]any{"param": map[string]string{"uri": uri}, "range": selected, "extractPosition": selected.Start, "codeActionName": "ExtractMethodCodeAction"})
	if err != nil {
		return nil, err
	}
	return json.Marshal([]scalaAction{{Title: "Extract selection as method in method `run`", Kind: scalaRefactorExtract, Data: data}})
}

func (s *extractionLifecycleSession) resolveAction() (json.RawMessage, error) {
	uri := pathutil.FileURI(s.file)
	start := strings.Index(string(s.source), "consume(1)")
	selected := scalaRange{Start: scalaPositionAt(s.source, start), End: scalaPositionAt(s.source, start+len("consume(1)"))}
	edits := []scalaEdit{
		{Range: selected, NewText: "newMethod()"},
		{Range: scalaRange{Start: selected.Start, End: selected.Start}, NewText: "def newMethod(): Unit =\n      consume(1)\n\n    "},
	}
	edit, err := json.Marshal(map[string]any{"changes": map[string]any{uri: edits}})
	if err != nil {
		return nil, err
	}
	data, err := json.Marshal(map[string]any{"param": map[string]string{"uri": uri}, "range": selected, "extractPosition": selected.Start, "codeActionName": "ExtractMethodCodeAction"})
	if err != nil {
		return nil, err
	}
	return json.Marshal(scalaAction{Title: "Extract selection as method in method `run`", Kind: scalaRefactorExtract, Data: data, Edit: edit})
}

func (s *extractionLifecycleSession) renameAction(params any) (json.RawMessage, error) {
	var request struct {
		NewName string `json:"newName"`
	}
	s.decode(params, &request)
	name := s.generated()
	text := string(s.candidate)
	first := strings.Index(text, name)
	if first < 0 {
		return nil, errors.New("generated name missing")
	}
	second := strings.Index(text[first+len(name):], name)
	if second < 0 {
		return nil, errors.New("generated reference missing")
	}
	second += first + len(name)
	edits := []scalaEdit{
		{Range: scalaRange{Start: scalaPositionAt(s.candidate, first), End: scalaPositionAt(s.candidate, first+len(name))}, NewText: request.NewName},
		{Range: scalaRange{Start: scalaPositionAt(s.candidate, second), End: scalaPositionAt(s.candidate, second+len(name))}, NewText: request.NewName},
	}
	return json.Marshal(map[string]any{"documentChanges": []any{map[string]any{"textDocument": map[string]any{"uri": pathutil.FileURI(s.file), "version": 2}, "edits": edits}}})
}

func TestScalaPublicExtractionLifecycle(t *testing.T) {
	operations := []backend.Operation{backend.OperationExtractVariable, backend.OperationExtractFunction}
	modes := []string{"success", "action", "resolve", "rename", "didOpen", "didChange", "didClose", "cancel", "overlay"}
	for _, operation := range operations {
		for _, mode := range modes {
			if operation == backend.OperationExtractVariable && mode == "resolve" {
				continue
			}
			t.Run(string(operation)+"/"+mode, func(t *testing.T) { testScalaPublicExtractionLifecycle(t, operation, mode) })
		}
	}
}

func testScalaPublicExtractionLifecycle(t *testing.T, operation backend.Operation, mode string) {
	scenario := newScalaExtractionLifecycleScenario(t, operation, mode)
	result, applyErr := scenario.adapter.ApplyCodeAction(scenario.ctx, scenario.request)
	scenario.assertResult(t, result, applyErr)
}

func assertLifecycleOrder(t *testing.T, events []string) {
	t.Helper()
	indices := map[string]int{}
	for i, event := range events {
		if _, ok := indices[event]; !ok {
			indices[event] = i
		}
	}
	want := []string{"notify:textDocument/didOpen", "request:textDocument/codeAction", "notify:textDocument/didChange", "request:textDocument/prepareRename", "request:textDocument/rename", "notify:textDocument/didClose"}
	for i, event := range want {
		at, ok := indices[event]
		if !ok {
			t.Fatalf("missing lifecycle event %q: %#v", event, events)
		}
		if i > 0 && at <= indices[want[i-1]] {
			t.Fatalf("lifecycle events out of order: %#v", events)
		}
	}
}

func hasLifecycleEvent(events []string, want string) bool {
	return slices.Contains(events, want)
}

type scalaExtractionLifecycleScenario struct {
	ctx     context.Context
	file    string
	source  []byte
	request backend.CodeActionRequest
	session *extractionLifecycleSession
	adapter *ScalaBackend
	mode    string
	buffer  *sourcebuf.Buffer
}

func newScalaExtractionLifecycleScenario(t *testing.T, operation backend.Operation, mode string) *scalaExtractionLifecycleScenario {
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	scenario := &scalaExtractionLifecycleScenario{ctx: context.Background(), file: filepath.Join(root, "Thing.scala"), mode: mode}
	if operation == backend.OperationExtractVariable {
		scenario.source = []byte("object Thing {\n  def run(): Int = consume(1 + 2)\n  def consume(value: Int): Int = value\n}\n")
		start := strings.Index(string(scenario.source), "1 + 2")
		position := scalaPositionAt(scenario.source, start)
		end := scalaPositionAt(scenario.source, start+len("1 + 2"))
		scenario.request = backend.CodeActionRequest{Operation: operation, Kind: scalaRefactorExtract, NewName: "total", Symbol: "run", Expression: "1 + 2", Range: backend.Range{Start: backend.Position{Line: position.Line, Character: position.Character}, End: backend.Position{Line: end.Line, Character: end.Character}}}
	} else {
		scenario.source = []byte("object Thing {\n  def run(): Unit = {\n    consume(1)\n  }\n  def consume(value: Int): Unit = ()\n}\n")
		scenario.request = backend.CodeActionRequest{Operation: operation, Kind: scalaRefactorExtract, NewName: "perform", EnclosingSymbol: "run", Expression: "consume(1)", Range: backend.Range{Start: backend.Position{Line: 2, Character: 4}, End: backend.Position{Line: 2, Character: 14}}}
	}
	if writeErr := os.WriteFile(scenario.file, scenario.source, 0o600); writeErr != nil {
		t.Fatal(writeErr)
	}
	if mode == "overlay" {
		scenario.buffer, err = sourcebuf.New(scenario.ctx, root)
		if err != nil {
			t.Fatal(err)
		}
		if err := scenario.buffer.AuthorizeWrites(scenario.ctx, root); err != nil {
			t.Fatal(err)
		}
		scenario.ctx = sourcebuf.WithBuffer(scenario.ctx, scenario.buffer)
	}
	project := backend.ProjectContext{RootDir: root, File: scenario.file, WorkspaceTrust: backend.NewWorkspaceTrust(root, true)}
	scenario.request.Project, scenario.request.File = project, scenario.file
	scenario.session = &extractionLifecycleSession{t: t, file: scenario.file, source: scenario.source, mode: mode}
	if mode == "cancel" {
		var cancel context.CancelFunc
		scenario.ctx, cancel = context.WithCancel(scenario.ctx)
		scenario.session.cancel = cancel
		t.Cleanup(cancel)
	}
	scenario.adapter = NewScalaBackendWithFactory(func(context.Context, string, backend.ScalaConfig) (ScalaSession, error) { return scenario.session, nil })
	return scenario
}

func (scenario *scalaExtractionLifecycleScenario) assertResult(t *testing.T, result *backend.CodeActionResult, applyErr error) {
	t.Helper()
	successful := scenario.mode == "success" || scenario.mode == "overlay"
	if successful {
		if applyErr != nil || result == nil {
			t.Fatalf("ApplyCodeAction() = (%#v, %v)", result, applyErr)
		}
		assertLifecycleOrder(t, scenario.session.events)
		if scenario.mode == "overlay" {
			disk, err := os.ReadFile(scenario.file)
			if err != nil || string(disk) != string(scenario.source) {
				t.Fatalf("disk source = %q, err %v; want unchanged", disk, err)
			}
			_, _, staged, err := resolveScalaProject(scenario.ctx, scenario.request.Project)
			if err != nil || string(staged) != string(result.After) {
				t.Fatalf("caller overlay = %q, err %v; want result", staged, err)
			}
		}
	} else {
		if applyErr == nil || result != nil {
			t.Fatalf("expected %s failure, got (%#v, %v)", scenario.mode, result, applyErr)
		}
		disk, err := os.ReadFile(scenario.file)
		if err != nil || string(disk) != string(scenario.source) {
			t.Fatalf("disk source after %s failure = %q, %v", scenario.mode, disk, err)
		}
	}
	if (scenario.mode == "didOpen" || scenario.mode == "didClose") && (!scenario.session.closed || scenario.adapter.session != nil) {
		t.Fatalf("%s failure did not retire session: closed=%v, session=%#v", scenario.mode, scenario.session.closed, scenario.adapter.session)
	}
	if scenario.mode == "didOpen" && hasLifecycleEvent(scenario.session.events, "request:textDocument/codeAction") {
		t.Fatalf("failed document open reached code actions: %#v", scenario.session.events)
	}
	if scenario.mode == "cancel" && !hasLifecycleEvent(scenario.session.events, "notify:textDocument/didClose") {
		t.Fatalf("canceled request did not close live document: %#v", scenario.session.events)
	}
}
