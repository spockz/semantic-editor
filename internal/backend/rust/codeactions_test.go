// Package rust exercises safe extraction edit capture and publication boundaries.
package rust

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/spockz/semantic-editor/internal/backend"
	"github.com/spockz/semantic-editor/internal/backend/pathutil"
	"github.com/spockz/semantic-editor/internal/sourcebuf"
)

type extractRustSession struct {
	action      json.RawMessage
	resolved    json.RawMessage
	rename      json.RawMessage
	prepared    []json.RawMessage
	prepareErrs map[rustPosition]error
	prepareArgs []json.RawMessage
	resolveArgs []json.RawMessage
	calls       []string
	opened      []string
	openedURI   string
	closedURI   string
	init        json.RawMessage
	closeErr    error
	closed      int
}

type callableRustSession struct {
	*extractRustSession

	symbols     json.RawMessage
	status      json.RawMessage
	openErr     error
	closeCtxErr error
	cancel      context.CancelFunc
	onClose     func()
}

func (s *callableRustSession) Request(ctx context.Context, method string, params any) (json.RawMessage, error) {
	if method == "textDocument/documentSymbol" {
		s.calls = append(s.calls, method)
		return s.symbols, nil
	}
	response, err := s.extractRustSession.Request(ctx, method, params)
	if method == "textDocument/codeAction" && s.cancel != nil {
		s.cancel()
	}
	return response, err
}

func (s *callableRustSession) Notify(ctx context.Context, method string, params any) error {
	if method == "textDocument/didOpen" && s.openErr != nil {
		return s.openErr
	}
	if method == "textDocument/didClose" {
		s.closeCtxErr = ctx.Err()
	}
	err := s.extractRustSession.Notify(ctx, method, params)
	if method == "textDocument/didClose" && s.onClose != nil {
		s.onClose()
	}
	return err
}

func (s *callableRustSession) LatestServerStatus() json.RawMessage { return s.status }

func (s *callableRustSession) ServerStatuses() <-chan json.RawMessage {
	return make(chan json.RawMessage)
}

func (s *extractRustSession) Request(_ context.Context, method string, params any) (json.RawMessage, error) {
	s.calls = append(s.calls, method)
	switch method {
	case "initialize":
		s.init = mustJSON(params)
		return json.RawMessage(`{"capabilities":{"codeActionProvider":{"resolveProvider":true}}}`), nil
	case "textDocument/codeAction":
		return s.action, nil
	case "textDocument/documentSymbol":
		return mustJSON([]any{map[string]any{
			"name": "run", "kind": 12,
			"range": rustRange{Start: rustPosition{Line: 0}, End: rustPosition{Line: 2, Character: 1}}, "selectionRange": fakeRustRange(0, 3, 6),
		}}), nil
	case "codeAction/resolve":
		s.resolveArgs = append(s.resolveArgs, mustJSON(params))
		return s.resolved, nil
	case "textDocument/prepareRename":
		s.prepareArgs = append(s.prepareArgs, mustJSON(params))
		var request struct {
			Position rustPosition `json:"position"`
		}
		if err := json.Unmarshal(mustJSON(params), &request); err != nil {
			return nil, err
		}
		if prepareErr := s.prepareErrs[request.Position]; prepareErr != nil {
			return nil, prepareErr
		}
		if len(s.prepared) > 0 {
			response := s.prepared[0]
			s.prepared = s.prepared[1:]
			return response, nil
		}
		return mustJSON(map[string]any{"start": request.Position, "end": rustPosition{Line: request.Position.Line, Character: request.Position.Character + len("extracted")}}), nil
	case "textDocument/rename":
		return s.rename, nil
	default:
		return json.RawMessage(`{}`), nil
	}
}

func (s *extractRustSession) Notify(_ context.Context, method string, params any) error {
	s.calls = append(s.calls, method)
	var document struct {
		TextDocument struct {
			URI string `json:"uri"`
		} `json:"textDocument"`
	}
	if err := json.Unmarshal(mustJSON(params), &document); err != nil {
		return err
	}
	if method == "textDocument/didClose" {
		if document.TextDocument.URI == "" {
			return fmt.Errorf("invalid didClose payload: %s", mustJSON(params))
		}
		s.closedURI = document.TextDocument.URI
		s.closed++
		return s.closeErr
	}
	if method == "textDocument/didOpen" {
		s.openedURI = document.TextDocument.URI
		var request struct {
			TextDocument struct {
				Text string `json:"text"`
			} `json:"textDocument"`
		}
		if err := json.Unmarshal(mustJSON(params), &request); err != nil {
			return err
		}
		s.opened = append(s.opened, request.TextDocument.Text)
	}
	return nil
}

func (s *extractRustSession) Close() error { return nil }

func TestApplyCodeActionCapturesVersionedExtractAndRenamesOnlyIntroducedBinding(t *testing.T) {
	root, file, session, original := rustExtractFixture(t, 1, true, false)
	backendUnderTest := NewRustBackendWithFactory(func(context.Context, string) (RustSession, error) { return session, nil })
	result, err := backendUnderTest.ApplyCodeAction(context.Background(), rustExtractRequest(root, file))
	if err != nil {
		t.Fatalf("ApplyCodeAction() error = %v", err)
	}
	want := "fn run() {\n    let total = 1_i32 + 2_i32;\n    consume(total);\n}\n"
	if string(result.Before) != original || string(result.After) != want {
		t.Fatalf("result before/after = %q / %q, want %q / %q", result.Before, result.After, original, want)
	}
	got, err := os.ReadFile(file) // #nosec G304 -- test fixture path is created in t.TempDir.
	if err != nil || string(got) != want {
		t.Fatalf("published source = %q, err %v; want %q", got, err, want)
	}
	if !containsString(session.calls, "textDocument/codeAction") || !containsString(session.calls, "textDocument/prepareRename") || !containsString(session.calls, "textDocument/rename") {
		t.Fatalf("LSP calls = %v", session.calls)
	}
	assertRustDocumentLifecycle(t, session, file, true)
	if slices.Index(session.calls, "textDocument/didClose") <= slices.Index(session.calls, "textDocument/rename") {
		t.Fatalf("extraction document closed before final rename: %v", session.calls)
	}
	var init struct {
		Capabilities struct {
			TextDocument struct {
				DocumentSymbol struct {
					Hierarchical bool `json:"hierarchicalDocumentSymbolSupport"`
				} `json:"documentSymbol"`
			} `json:"textDocument"`
			Workspace struct {
				WorkspaceEdit struct {
					ChangeAnnotationSupport struct {
						GroupsOnLabel bool `json:"groupsOnLabel"`
					} `json:"changeAnnotationSupport"`
				} `json:"workspaceEdit"`
			} `json:"workspace"`
		} `json:"capabilities"`
	}
	if err := json.Unmarshal(session.init, &init); err != nil || !init.Capabilities.TextDocument.DocumentSymbol.Hierarchical {
		t.Fatalf("initialize documentSymbol capabilities = %s, err %v", session.init, err)
	}
	var annotationSupport struct {
		Capabilities struct {
			Workspace struct {
				WorkspaceEdit struct {
					ChangeAnnotationSupport *struct {
						GroupsOnLabel bool `json:"groupsOnLabel"`
					} `json:"changeAnnotationSupport"`
				} `json:"workspaceEdit"`
			} `json:"workspace"`
		} `json:"capabilities"`
	}
	if err := json.Unmarshal(session.init, &annotationSupport); err != nil || annotationSupport.Capabilities.Workspace.WorkspaceEdit.ChangeAnnotationSupport == nil {
		t.Fatalf("initialize change annotation capability = %s, err %v", session.init, err)
	}
}

func TestPromoteLocalToConstAcceptsExactVersionedScalarEditWithoutResolvedData(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "src"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "Cargo.toml"), []byte("[package]\nname = \"promote-local\"\nversion = \"0.1.0\"\nedition = \"2021\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(root, "src", "lib.rs")
	source := "fn run() {\n    let retry_limit: i32 = 7;\n    consume(retry_limit);\n}\n\nfn consume(value: i32) {}\n"
	if err := os.WriteFile(file, []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}
	uri := pathutil.FileURI(file)
	resolved := map[string]any{"title": "Promote local to constant", "kind": "refactor", "edit": map[string]any{
		"documentChanges": []any{map[string]any{
			"textDocument": map[string]any{"uri": uri, "version": 1},
			"edits": []any{
				map[string]any{"range": fakeRustRange(1, 4, 7), "newText": "const"},
				map[string]any{"range": fakeRustRange(1, 8, 19), "newText": "RETRY_LIMIT"},
				map[string]any{"range": fakeRustRange(2, 12, 23), "newText": "RETRY_LIMIT"},
			},
		}},
	}}
	session := &extractRustSession{
		action:   mustJSON([]any{map[string]any{"title": "Promote local to constant", "kind": "refactor", "data": map[string]any{"id": "promote_local_to_const:Refactor:1:"}}}),
		resolved: mustJSON(resolved),
	}
	if _, _, err := selectUniqueRustCodeAction(session.action, "promote-local-to-const", func(action rustCodeAction) bool {
		var data rustCodeActionData
		return action.Title == "Promote local to constant" && action.Kind == "refactor" && json.Unmarshal(action.Data, &data) == nil && rustPromoteActionID.MatchString(data.ID)
	}); err != nil {
		t.Fatalf("select fake promotion action: %v", err)
	}
	backendUnderTest := NewRustBackendWithFactory(func(context.Context, string) (RustSession, error) { return session, nil })
	request := backend.CodeActionRequest{
		Operation: backend.OperationPromoteLocalToConst, Project: backend.ProjectContext{RootDir: root, File: file, Language: backend.LanguageRust, WorkspaceTrust: backend.NewWorkspaceTrust(root, true)},
		File: file, Kind: "promote_local_to_const", Symbol: "retry_limit", EnclosingSymbol: "run",
		Range: backend.Range{Start: backend.Position{Line: 1, Character: 8}, End: backend.Position{Line: 1, Character: 19}},
	}
	result, err := backendUnderTest.ApplyCodeAction(context.Background(), request)
	if err != nil {
		t.Fatalf("ApplyCodeAction() error = %v", err)
	}
	want := strings.Replace(source, "let retry_limit: i32 = 7;\n    consume(retry_limit);", "const RETRY_LIMIT: i32 = 7;\n    consume(RETRY_LIMIT);", 1)
	if string(result.After) != want {
		t.Fatalf("result.After = %q, want %q", result.After, want)
	}
	if len(session.resolveArgs) != 1 {
		t.Fatalf("resolve calls = %d, want one", len(session.resolveArgs))
	}
	var resolvedRequest map[string]json.RawMessage
	if err := json.Unmarshal(session.resolveArgs[0], &resolvedRequest); err != nil {
		t.Fatal(err)
	}
	if _, exists := resolvedRequest["data"]; !exists {
		t.Fatal("resolve request lost original action data")
	}
}

func TestPromoteLocalToConstValidatesOptionalResolvedIdentity(t *testing.T) {
	source := []byte("fn run() {\n    let retry_limit: i32 = 7;\n    consume(retry_limit);\n}\n\nfn consume(value: i32) {}\n")
	plan, err := planRustPromoteLocalToConst(source, "run", "retry_limit")
	if err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(t.TempDir(), "src", "lib.rs")
	uri := pathutil.FileURI(file)
	workspaceEdit := map[string]any{"documentChanges": []any{map[string]any{
		"textDocument": map[string]any{"uri": uri, "version": 1},
		"edits": []any{
			map[string]any{"range": fakeRustRange(1, 4, 7), "newText": "const"},
			map[string]any{"range": fakeRustRange(1, 8, 19), "newText": "RETRY_LIMIT"},
			map[string]any{"range": fakeRustRange(2, 12, 23), "newText": "RETRY_LIMIT"},
		},
	}}}
	for _, test := range []struct {
		name       string
		resolvedID any
		wantError  bool
	}{
		{name: "null data is permitted", resolvedID: nil},
		{name: "changed identity is refused", resolvedID: map[string]any{"id": "promote_local_to_const:Refactor:2:"}, wantError: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			resolved := map[string]any{"title": "Promote local to constant", "kind": "refactor", "edit": workspaceEdit}
			resolved["data"] = test.resolvedID
			session := &extractRustSession{
				action:   mustJSON([]any{map[string]any{"title": "Promote local to constant", "kind": "refactor", "data": map[string]any{"id": "promote_local_to_const:Refactor:1:"}}}),
				resolved: mustJSON(resolved),
			}
			got, err := resolveAndValidateRustPromoteLocalToConst(context.Background(), session, file, uri, source, plan)
			if test.wantError {
				if err == nil || !strings.Contains(err.Error(), "identity changed") {
					t.Fatalf("resolveAndValidateRustPromoteLocalToConst() error = %v, want changed-identity refusal", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("resolveAndValidateRustPromoteLocalToConst() error = %v", err)
			}
			if string(got) != string(plan.expected) {
				t.Fatalf("resolved result = %q, want %q", got, plan.expected)
			}
		})
	}
}

func TestApplyCodeActionRejectsResolvedIdentityAndRetiresSessionAfterCloseFailure(t *testing.T) {
	t.Run("resolved identity changed", func(t *testing.T) {
		root, file, session, original := rustExtractFixture(t, 1, true, false)
		session.action = mustJSON([]any{map[string]any{
			"title": "Extract into variable", "kind": "refactor.extract",
			"data": map[string]any{"id": "extract_variable:RefactorExtract:0"},
		}})
		session.resolved = mustJSON(map[string]any{
			"title": "Extract into variable", "kind": "refactor.extract",
			"data": map[string]any{"id": "extract_variable:Different:0"},
			"edit": rustExtractWorkspaceEdit(file, new(1)),
		})
		backendUnderTest := NewRustBackendWithFactory(func(context.Context, string) (RustSession, error) { return session, nil })
		if _, err := backendUnderTest.ApplyCodeAction(context.Background(), rustExtractRequest(root, file)); err == nil {
			t.Fatal("ApplyCodeAction() accepted a resolved action with a different identity")
		}
		got, err := os.ReadFile(file) // #nosec G304 -- test fixture path is created in t.TempDir.
		if err != nil || string(got) != original {
			t.Fatalf("resolved identity changed source to %q, err %v", got, err)
		}
	})
	t.Run("close failure", func(t *testing.T) {
		root, file, session, _ := rustExtractFixture(t, 1, true, false)
		session.closeErr = errors.New("close transport failed")
		backendUnderTest := NewRustBackendWithFactory(func(context.Context, string) (RustSession, error) { return session, nil })
		if _, err := backendUnderTest.ApplyCodeAction(context.Background(), rustExtractRequest(root, file)); err == nil || !strings.Contains(err.Error(), "close transport failed") {
			t.Fatalf("ApplyCodeAction() close error = %v", err)
		}
		if backendUnderTest.session != nil {
			t.Fatal("session retained after close failure")
		}
	})
}

func TestApplyCodeActionRejectsDisabledActionsWithoutMutation(t *testing.T) {
	for _, tc := range []struct {
		name     string
		resolved bool
	}{
		{name: "disabled initial action"},
		{name: "disabled resolved action", resolved: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root, file, session, original := rustExtractFixture(t, 1, true, false)
			identity := map[string]any{"id": "extract_variable:RefactorExtract:0"}
			initial := map[string]any{
				"title": "Extract into variable", "kind": "refactor.extract", "data": identity,
			}
			if tc.resolved {
				session.action = mustJSON([]any{initial})
				session.resolved = mustJSON(map[string]any{
					"title": "Extract into variable", "kind": "refactor.extract", "data": identity,
					"disabled": map[string]any{"reason": "selection is no longer extractable"},
					"edit":     rustExtractWorkspaceEdit(file, new(1)),
				})
			} else {
				initial["disabled"] = map[string]any{"reason": "selection is not extractable"}
				initial["edit"] = rustExtractWorkspaceEdit(file, new(1))
				session.action = mustJSON([]any{initial})
			}
			backendUnderTest := NewRustBackendWithFactory(func(context.Context, string) (RustSession, error) { return session, nil })
			if _, err := backendUnderTest.ApplyCodeAction(context.Background(), rustExtractRequest(root, file)); err == nil || !strings.Contains(err.Error(), "disabled") {
				t.Fatalf("ApplyCodeAction() error = %v, want disabled-action refusal", err)
			}
			got, err := os.ReadFile(file) // #nosec G304 -- test fixture path is created in t.TempDir.
			if err != nil || string(got) != original {
				t.Fatalf("disabled action changed source to %q, err %v", got, err)
			}
			if containsString(session.calls, "codeAction/resolve") != tc.resolved {
				t.Fatalf("resolve calls = %v", session.calls)
			}
		})
	}
}

func TestPrepareRustExtractCandidateAcceptsPinnedServerStatementRewrite(t *testing.T) {
	source := []byte("fn run() {\n    consume(1_i32 + 2_i32);\n}\n")
	selectionStart := strings.Index(string(source), "1_i32 + 2_i32")
	selectionEnd := selectionStart + len("1_i32 + 2_i32")
	statementStart := strings.Index(string(source), "consume")
	edits := []rustTextEdit{
		{Range: fakeRustRange(1, 4, 26), NewText: "let value = 1_i32 + 2_i32"},
		{Range: fakeRustRange(1, 27, 27), NewText: "\n    consume(value);"},
	}
	got, renamePosition, useSpan, bindingSpan, err := prepareRustExtractCandidate(source, edits, selectionStart, selectionEnd, statementStart, "1_i32 + 2_i32")
	if err != nil {
		t.Fatalf("prepareRustExtractCandidate() error = %v", err)
	}
	want := "fn run() {\n    let value = 1_i32 + 2_i32;\n    consume(value);\n}\n"
	if string(got) != want {
		t.Fatalf("candidate = %q, want %q", got, want)
	}
	if string(got[useSpan.start:useSpan.end]) != "value" || string(got[bindingSpan.start:bindingSpan.end]) != "value" {
		t.Fatalf("rename spans = use %q / binding %q", got[useSpan.start:useSpan.end], got[bindingSpan.start:bindingSpan.end])
	}
	if renamePosition.Line != 2 || renamePosition.Character != 12 {
		t.Fatalf("rename position = %+v, want line 2 character 12", renamePosition)
	}
}

func TestWaitForRustAnalyzerReadyRequiresHealthyQuiescentStatus(t *testing.T) {
	t.Run("waits through loading", func(t *testing.T) {
		statuses := make(chan json.RawMessage, 2)
		statuses <- json.RawMessage(`{"health":"ok","quiescent":true}`)
		statuses <- json.RawMessage(`{"health":"ok","quiescent":false}`)
		calls := 0
		latest := func() json.RawMessage {
			calls++
			if calls < 3 {
				return json.RawMessage(`{"health":"ok","quiescent":false}`)
			}
			return json.RawMessage(`{"health":"ok","quiescent":true}`)
		}
		if err := waitForRustAnalyzerReady(context.Background(), latest, statuses); err != nil {
			t.Fatalf("waitForRustAnalyzerReady() error = %v", err)
		}
	})
	t.Run("reports workspace error", func(t *testing.T) {
		statuses := make(chan json.RawMessage, 1)
		statuses <- json.RawMessage(`{"health":"warning","quiescent":true,"message":"cargo metadata failed"}`)
		calls := 0
		latest := func() json.RawMessage {
			calls++
			if calls == 1 {
				return nil
			}
			return json.RawMessage(`{"health":"warning","quiescent":true,"message":"cargo metadata failed"}`)
		}
		if err := waitForRustAnalyzerReady(context.Background(), latest, statuses); err == nil || !strings.Contains(err.Error(), "cargo metadata failed") {
			t.Fatalf("waitForRustAnalyzerReady() error = %v", err)
		}
	})
	t.Run("accepts latest ready state without another notification", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()
		if err := waitForRustAnalyzerReady(ctx, func() json.RawMessage { return json.RawMessage(`{"health":"ok","quiescent":true}`) }, make(chan json.RawMessage)); err != nil {
			t.Fatalf("waitForRustAnalyzerReady() error = %v", err)
		}
	})
}

func TestApplyCodeActionResolvesByStableAssistAndRejectsUnsafeActionShapes(t *testing.T) {
	for _, tc := range []struct {
		name        string
		version     *int
		command     bool
		resolve     bool
		wantResolve bool
	}{
		{name: "resolved action", version: new(1), resolve: true, wantResolve: true},
		{name: "stale action", version: new(2)},
		{name: "unversioned action", version: nil, resolve: true, wantResolve: true},
		{name: "command action", version: new(1), command: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root, file, session, original := rustExtractFixture(t, valueOr(tc.version, 1), tc.version != nil, tc.command)
			if tc.version != nil && *tc.version == 2 {
				session.action = rustExtractAction(file, new(2), tc.command)
			}
			if tc.resolve && tc.name == "resolved action" {
				session.action = rustExtractAction(file, nil, false)
			} else {
				session.action = rustExtractAction(file, tc.version, tc.command)
			}
			if tc.name == "unversioned action" {
				session.resolved = mustJSON(map[string]any{
					"title": "Extract into variable", "kind": "refactor.extract",
					"data": map[string]any{"id": "extract_variable:RefactorExtract:0"},
					"edit": rustExtractWorkspaceEdit(file, nil),
				})
			}
			backendUnderTest := NewRustBackendWithFactory(func(context.Context, string) (RustSession, error) { return session, nil })
			_, err := backendUnderTest.ApplyCodeAction(context.Background(), rustExtractRequest(root, file))
			if tc.version == nil || *tc.version != 1 || tc.command {
				if err == nil {
					t.Fatal("ApplyCodeAction() succeeded for an unsafe action shape")
				}
				got, readErr := os.ReadFile(file) // #nosec G304 -- test fixture path is created in t.TempDir.
				if readErr != nil || string(got) != original {
					t.Fatalf("unsafe action changed source to %q, err %v", got, readErr)
				}
			} else if err != nil {
				t.Fatalf("ApplyCodeAction() error = %v", err)
			}
			if containsString(session.calls, "codeAction/resolve") != tc.wantResolve {
				t.Fatalf("resolve calls = %v", session.calls)
			}
		})
	}
}

func TestPrepareRustExtractCandidateRejectsWrappedInitializerAndTracksExactBinding(t *testing.T) {
	source := []byte("fn run() {\n    consume(1 + 2);\n}\n")
	selected := rustTextEdit{Range: fakeRustRange(1, 12, 17), NewText: "extracted"}
	insertion := rustTextEdit{Range: fakeRustRange(1, 4, 4), NewText: "let extracted = wrap(1 + 2);\n    "}
	start, _ := rustByteOffset(source, rustPosition{Line: 1, Character: 12})
	end, _ := rustByteOffset(source, rustPosition{Line: 1, Character: 17})
	insertionStart, _ := rustByteOffset(source, rustPosition{Line: 1, Character: 4})
	if _, _, _, _, err := prepareRustExtractCandidate(source, []rustTextEdit{selected, insertion}, start, end, insertionStart, "1 + 2"); err == nil {
		t.Fatal("wrapped initializer accepted")
	}
	insertion.NewText = "let extracted = 1 + 2;\n    "
	candidate, _, _, binding, err := prepareRustExtractCandidate(source, []rustTextEdit{selected, insertion}, start, end, insertionStart, "1 + 2")
	if err != nil {
		t.Fatalf("prepareRustExtractCandidate() error = %v", err)
	}
	if got := string(candidate[binding.start:binding.end]); got != "extracted" {
		t.Fatalf("binding span = %q, want exact identifier", got)
	}
	if containsRustBindingRename(candidate, []rustTextEdit{{Range: fakeRustRange(0, 4, 13), NewText: "total"}}, binding, "extracted", "total") {
		t.Fatal("binding rename matched a range outside the exact inserted binding")
	}
}

func TestRustSafeExtractionExpressionRequiresSuffixedIntegerOperations(t *testing.T) {
	for _, expression := range []string{"1_i32 + 2_i32", "(1_u8 * 2_u8) + 3_u8"} {
		if !rustSuffixedIntegerExpression(expression) {
			t.Errorf("rustSuffixedIntegerExpression(%q) = false", expression)
		}
	}
	for _, expression := range []string{"1 + 2", "&make_guard()", "first()", "target = other", "1_i32 + value"} {
		if rustSuffixedIntegerExpression(expression) {
			t.Errorf("rustSuffixedIntegerExpression(%q) = true", expression)
		}
	}
}

func TestRustSuffixedIntegerExpressionTypeProvesConstAdditionBounds(t *testing.T) {
	for _, expression := range []string{
		"255_u8 + 0_u8",
		"127_i8 + 0_i8",
		"126_i8 + 1_i8",
		"18446744073709551615_u64 + 0_u64",
	} {
		if _, ok := rustSuffixedIntegerExpressionType(expression); !ok {
			t.Errorf("rustSuffixedIntegerExpressionType(%q) refused an in-range sum", expression)
		}
	}
	for _, expression := range []string{
		"255_u8 + 1_u8",
		"127_i8 + 1_i8",
		"18446744073709551615_u64 + 1_u64",
		"1_usize + 2_usize",
		"1_isize + 2_isize",
	} {
		if _, ok := rustSuffixedIntegerExpressionType(expression); ok {
			t.Errorf("rustSuffixedIntegerExpressionType(%q) accepted an overflow or unknown-width sum", expression)
		}
	}
}

func TestPrepareRustExtractCandidateRejectsInsertionBeforeValidatedStatement(t *testing.T) {
	source := []byte("fn run() {\n    update_state();\n    consume(1_i32 + 2_i32);\n}\n")
	selected := rustTextEdit{Range: fakeRustRange(2, 12, 25), NewText: "extracted"}
	insertion := rustTextEdit{Range: fakeRustRange(1, 4, 4), NewText: "let extracted = read_state();\n    "}
	start, _ := rustByteOffset(source, rustPosition{Line: 2, Character: 12})
	end, _ := rustByteOffset(source, rustPosition{Line: 2, Character: 25})
	if _, _, _, _, err := prepareRustExtractCandidate(source, []rustTextEdit{selected, insertion}, start, end, 42, "1_i32 + 2_i32"); err == nil {
		t.Fatal("insertion before a different statement was accepted")
	}
}

func TestApplyCodeActionRejectsExtraRenameEditOutsideExtractedLocal(t *testing.T) {
	root, file, session, _ := rustExtractFixture(t, 1, true, false)
	original := "fn run() {\n    consume(1_i32 + 2_i32);\n}\n// extracted\n"
	if err := os.WriteFile(file, []byte(original), 0o600); err != nil {
		t.Fatal(err)
	}
	uri := pathutil.FileURI(file)
	session.rename = mustJSON(map[string]any{"documentChanges": []any{map[string]any{
		"textDocument": map[string]any{"uri": uri, "version": 2},
		"edits": []any{
			map[string]any{"range": fakeRustRange(1, 8, 17), "newText": "total"},
			map[string]any{"range": fakeRustRange(2, 12, 21), "newText": "total"},
			map[string]any{"range": fakeRustRange(3, 3, 12), "newText": "total"},
		},
	}}})
	backendUnderTest := NewRustBackendWithFactory(func(context.Context, string) (RustSession, error) { return session, nil })
	if _, err := backendUnderTest.ApplyCodeAction(context.Background(), rustExtractRequest(root, file)); err == nil {
		t.Fatal("ApplyCodeAction() accepted an unrelated third rename edit")
	}
	got, err := os.ReadFile(file) // #nosec G304 -- test fixture path is created in t.TempDir.
	if err != nil || string(got) != original {
		t.Fatalf("unrelated rename edit changed source to %q, err %v", got, err)
	}
}

func TestApplyCodeActionRejectsRequestedNameThatMayShadowExistingBinding(t *testing.T) {
	root, file, session, _ := rustExtractFixture(t, 1, true, false)
	source := "fn run() {\n    let total = 10;\n    consume(1_i32 + 2_i32);\n}\n"
	if err := os.WriteFile(file, []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}
	request := rustExtractRequest(root, file)
	request.Range = backend.Range{Start: backend.Position{Line: 2, Character: 12}, End: backend.Position{Line: 2, Character: 25}}
	backendUnderTest := NewRustBackendWithFactory(func(context.Context, string) (RustSession, error) { return session, nil })
	if _, err := backendUnderTest.ApplyCodeAction(context.Background(), request); err == nil || !strings.Contains(err.Error(), "shadowing conflict") {
		t.Fatalf("ApplyCodeAction() collision error = %v", err)
	}
	if len(session.calls) != 0 {
		t.Fatalf("collision reached Rust analyzer: calls %v", session.calls)
	}
}

func TestApplyCodeActionRejectsNamesThatAreNotSafeRustBindings(t *testing.T) {
	for _, name := range []string{"None", "Some", "Ok", "Err", "Total", "match", "self", "crate", "_"} {
		t.Run(name, func(t *testing.T) {
			root, file, session, _ := rustExtractFixture(t, 1, true, false)
			request := rustExtractRequest(root, file)
			request.NewName = name
			backendUnderTest := NewRustBackendWithFactory(func(context.Context, string) (RustSession, error) { return session, nil })
			if _, err := backendUnderTest.ApplyCodeAction(context.Background(), request); err == nil {
				t.Fatalf("ApplyCodeAction() accepted unsafe Rust binding name %q", name)
			}
			if len(session.calls) != 0 {
				t.Fatalf("unsafe binding name reached Rust analyzer: calls %v", session.calls)
			}
		})
	}
}

func TestRustIdentifierPresentMatchesUnicodeNormalizedSpelling(t *testing.T) {
	if !rustIdentifierPresent([]byte("fn run() { let K = 10; }"), "K") {
		t.Fatal("rustIdentifierPresent() failed to match canonically equivalent spellings")
	}
}

func TestApplyCodeActionRejectsUnresolvedGlobAndIncludeNames(t *testing.T) {
	for _, declaration := range []string{"use helper::{*};", `include  !("later.rs");`, `include/* comment */!("later.rs");`, `helper::load!("later.rs");`} {
		t.Run(declaration, func(t *testing.T) {
			root, file, session, _ := rustExtractFixture(t, 1, true, false)
			source := "fn run() {\n    consume(1_i32 + 2_i32);\n    " + declaration + "\n}\n"
			if err := os.WriteFile(file, []byte(source), 0o600); err != nil {
				t.Fatal(err)
			}
			request := rustExtractRequest(root, file)
			request.NewName = "total"
			backendUnderTest := NewRustBackendWithFactory(func(context.Context, string) (RustSession, error) { return session, nil })
			if _, err := backendUnderTest.ApplyCodeAction(context.Background(), request); err == nil || !strings.Contains(err.Error(), "unresolved imported or macro-expanded names") {
				t.Fatalf("ApplyCodeAction() unresolved expansion error = %v", err)
			}
			if len(session.calls) != 0 {
				t.Fatalf("unresolved expansion reached Rust analyzer: calls %v", session.calls)
			}
		})
	}
}

func rustExtractFixture(t *testing.T, version int, versioned, command bool) (string, string, *extractRustSession, string) {
	t.Helper()
	root := t.TempDir()
	canonicalRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(canonicalRoot, "Cargo.toml"), []byte("[package]\nname = \"extract\"\nversion = \"0.1.0\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(canonicalRoot, "lib.rs")
	original := "fn run() {\n    consume(1_i32 + 2_i32);\n}\n"
	if err := os.WriteFile(file, []byte(original), 0o600); err != nil {
		t.Fatal(err)
	}
	var versionPtr *int
	if versioned {
		versionPtr = &version
	}
	session := &extractRustSession{}
	session.action = rustExtractAction(file, versionPtr, command)
	resolvedVersion := 1
	session.resolved = mustJSON(map[string]any{
		"title": "Extract into variable", "kind": "refactor.extract",
		"data": map[string]any{"id": "extract_variable:RefactorExtract:0"},
		"edit": rustExtractWorkspaceEdit(file, &resolvedVersion),
	})
	uri := pathutil.FileURI(file)
	session.rename = mustJSON(map[string]any{"documentChanges": []any{map[string]any{
		"textDocument": map[string]any{"uri": uri, "version": 2},
		"edits": []any{
			map[string]any{"range": fakeRustRange(1, 8, 17), "newText": "total"},
			map[string]any{"range": fakeRustRange(2, 12, 21), "newText": "total"},
		},
	}}, "changeAnnotations": map[string]any{
		"workspace": map[string]any{"label": "Edit outside of the workspace"},
	}})
	return canonicalRoot, file, session, original
}

func rustExtractRequest(root, file string) backend.CodeActionRequest {
	return backend.CodeActionRequest{
		Operation: backend.OperationExtractVariable,
		Project:   backend.ProjectContext{RootDir: root, File: file, WorkspaceTrust: backend.NewWorkspaceTrust(root, true)},
		File:      file,
		Range:     backend.Range{Start: backend.Position{Line: 1, Character: 12}, End: backend.Position{Line: 1, Character: 25}},
		Kind:      "refactor.extract", NewName: "total", Expression: "1_i32 + 2_i32",
	}
}

func rustCallableFixture(t *testing.T) (string, string, *callableRustSession, backend.CodeActionRequest, []byte, []byte) {
	t.Helper()
	root := t.TempDir()
	root, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "Cargo.toml"), []byte("[package]\nname = \"callable\"\nversion = \"0.1.0\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(root, "lib.rs")
	before := []byte("fn consume(value: i32) {}\nfn run() {\n    consume(1_i32);\n}\n")
	after := []byte("fn consume(value: i32) {}\nfn run() {\n    invoke();\n}\nfn invoke() {\n    consume(1_i32);\n}\n")
	if err := os.WriteFile(file, before, 0o600); err != nil {
		t.Fatal(err)
	}
	uri := pathutil.FileURI(file)
	base := &extractRustSession{}
	insert := rustTextEdit{Range: fakeRustRange(3, 1, 1), NewText: "\nfn extracted() {\n    consume(1_i32);\n}"}
	replace := rustTextEdit{Range: fakeRustRange(2, 4, 19), NewText: "extracted();"}
	edit := map[string]any{"documentChanges": []any{map[string]any{
		"textDocument": map[string]any{"uri": uri, "version": 1},
		"edits":        []any{insert, replace},
	}}}
	base.action = mustJSON([]any{map[string]any{"title": "Extract function", "kind": rustRefactorExtract, "data": map[string]any{"id": "extract_function:ExtractFunction:0"}, "edit": edit}})
	base.rename = mustJSON(map[string]any{"documentChanges": []any{map[string]any{
		"textDocument": map[string]any{"uri": uri, "version": 2},
		"edits": []any{
			map[string]any{"range": fakeRustRange(2, 4, 13), "newText": "invoke"},
			map[string]any{"range": fakeRustRange(4, 3, 12), "newText": "invoke"},
		},
	}}})
	base.resolved = base.action
	symbols := mustJSON([]any{
		map[string]any{"name": "consume", "kind": 12, "range": fakeRustRange(0, 0, 25), "selectionRange": fakeRustRange(0, 3, 10)},
		map[string]any{"name": "run", "kind": 12, "range": rustRange{Start: rustPosition{Line: 1}, End: rustPosition{Line: 3, Character: 1}}, "selectionRange": fakeRustRange(1, 3, 6)},
	})
	request := backend.CodeActionRequest{
		Operation: backend.OperationExtractFunction,
		Project:   backend.ProjectContext{RootDir: root, File: file, WorkspaceTrust: backend.NewWorkspaceTrust(root, true)},
		File:      file, Kind: rustRefactorExtract, EnclosingSymbol: "run", Expression: "consume(1_i32);", NewName: "invoke",
		Range: backend.Range{Start: backend.Position{Line: 2, Character: 4}, End: backend.Position{Line: 2, Character: 19}},
	}
	session := &callableRustSession{extractRustSession: base, symbols: symbols, status: json.RawMessage(`{"health":"ok","quiescent":true}`)}
	return root, file, session, request, before, after
}

func TestRustCallableApplyCodeActionLifecycleAndPublication(t *testing.T) {
	t.Run("standalone publication flushes exact postimage", func(t *testing.T) {
		root, file, session, request, before, after := rustCallableFixture(t)
		backendUnderTest := NewRustBackendWithFactory(func(context.Context, string) (RustSession, error) { return session, nil })
		result, err := backendUnderTest.ApplyCodeAction(context.Background(), request)
		if err != nil {
			t.Fatalf("ApplyCodeAction() error = %v", err)
		}
		if string(result.Before) != string(before) || string(result.After) != string(after) {
			t.Fatalf("result before/after = %q / %q", result.Before, result.After)
		}
		got, err := os.ReadFile(file)
		if err != nil || string(got) != string(after) {
			t.Fatalf("published source = %q, err %v", got, err)
		}
		assertRustDocumentLifecycle(t, session.extractRustSession, file, true)
		if !containsString(session.calls, "textDocument/documentSymbol") || !containsString(session.calls, "textDocument/prepareRename") || !containsString(session.calls, "textDocument/rename") {
			t.Fatalf("callable proof calls = %v", session.calls)
		}
		if gotCalls := session.calls; slices.Index(gotCalls, "textDocument/didOpen") > slices.Index(gotCalls, "textDocument/documentSymbol") || slices.Index(gotCalls, "textDocument/didClose") < slices.Index(gotCalls, "textDocument/rename") {
			t.Fatalf("callable operation order = %v", gotCalls)
		}
		if root == "" {
			t.Fatal("fixture root is empty")
		}
	})
	t.Run("caller buffer stages exact postimage without disk publication", func(t *testing.T) {
		root, file, session, request, before, after := rustCallableFixture(t)
		buffer, err := sourcebuf.New(context.Background(), root)
		if err != nil {
			t.Fatal(err)
		}
		if authorizeErr := buffer.AuthorizeWrites(context.Background(), root); authorizeErr != nil {
			t.Fatal(authorizeErr)
		}
		ctx := sourcebuf.WithBuffer(context.Background(), buffer)
		backendUnderTest := NewRustBackendWithFactory(func(context.Context, string) (RustSession, error) { return session, nil })
		result, err := backendUnderTest.ApplyCodeAction(ctx, request)
		if err != nil {
			t.Fatalf("ApplyCodeAction() error = %v", err)
		}
		if string(result.After) != string(after) {
			t.Fatalf("result postimage = %q", result.After)
		}
		got, err := os.ReadFile(file)
		if err != nil || string(got) != string(before) {
			t.Fatalf("caller-owned buffer changed disk source to %q, err %v", got, err)
		}
		staged, err := sourcebuf.ReadFile(ctx, file)
		if err != nil || string(staged) != string(after) {
			t.Fatalf("staged source = %q, err %v", staged, err)
		}
	})
}

func TestRustCallableApplyCodeActionFailuresCleanupAndScopeRecheck(t *testing.T) {
	for _, scenario := range []string{"open failure", "readiness failure", "action failure", "close failure", "action and close failure"} {
		t.Run(scenario, func(t *testing.T) { assertRustCallableFailureCleanup(t, scenario) })
	}
}

func assertRustCallableFailureCleanup(t *testing.T, scenario string) {
	t.Helper()
	_, file, session, request, before, _ := rustCallableFixture(t)
	switch scenario {
	case "open failure":
		session.openErr = errors.New("open transport failed")
	case "readiness failure":
		session.status = json.RawMessage(`{"health":"warning","quiescent":true,"message":"workspace failed"}`)
	case "action failure":
		session.action = json.RawMessage(`[]`)
	case "close failure":
		session.closeErr = errors.New("close transport failed")
	case "action and close failure":
		session.action = json.RawMessage(`[]`)
		session.closeErr = errors.New("close transport failed")
	}
	backendUnderTest := NewRustBackendWithFactory(func(context.Context, string) (RustSession, error) { return session, nil })
	_, operationErr := backendUnderTest.ApplyCodeAction(context.Background(), request)
	if operationErr == nil {
		t.Fatal("ApplyCodeAction() unexpectedly succeeded")
	}
	if scenario == "action and close failure" && (!strings.Contains(operationErr.Error(), "invalid rust rename edit") || !strings.Contains(operationErr.Error(), "close transport failed")) {
		t.Fatalf("joined error = %v", operationErr)
	}
	got, readErr := os.ReadFile(file)
	if readErr != nil || string(got) != string(before) {
		t.Fatalf("failed operation published %q, err %v", got, readErr)
	}
	wantCloses := 1
	if scenario == "open failure" {
		wantCloses = 0
	}
	if session.closed != wantCloses {
		t.Fatalf("didClose count = %d, want %d; calls %v", session.closed, wantCloses, session.calls)
	}
	if scenario == "readiness failure" && containsString(session.calls, "textDocument/codeAction") {
		t.Fatal("requested action before readiness")
	}
	if scenario == "close failure" || scenario == "action and close failure" {
		if backendUnderTest.session != nil {
			t.Fatal("backend retained session after close failure")
		}
	}
	if scenario != "open failure" {
		assertRustDocumentLifecycle(t, session.extractRustSession, file, scenario != "readiness failure")
	}
}

func TestRustCallableCancellationCleanupUsesLiveContext(t *testing.T) {
	_, _, session, request, _, _ := rustCallableFixture(t)
	ctx, cancel := context.WithCancel(context.Background())
	session.cancel = cancel
	backendUnderTest := NewRustBackendWithFactory(func(context.Context, string) (RustSession, error) { return session, nil })
	_, _ = backendUnderTest.ApplyCodeAction(ctx, request)
	if ctx.Err() == nil {
		t.Fatal("code action request did not cancel context")
	}
	if session.closeCtxErr != nil || session.closed != 1 {
		t.Fatalf("cleanup context/count = %v / %d, want live / 1", session.closeCtxErr, session.closed)
	}
}

func TestRustCallableLeavesUnobservedSourceAdditionBehind(t *testing.T) {
	root, file, session, request, _, after := rustCallableFixture(t)
	other := filepath.Join(root, "other.rs")
	session.onClose = func() {
		if writeErr := os.WriteFile(other, []byte("fn other() {}\n"), 0o600); writeErr != nil {
			t.Errorf("create unrelated package source: %v", writeErr)
		}
	}
	backendUnderTest := NewRustBackendWithFactory(func(context.Context, string) (RustSession, error) { return session, nil })
	if _, operationErr := backendUnderTest.ApplyCodeAction(context.Background(), request); operationErr != nil {
		t.Fatalf("ApplyCodeAction() with an unrelated source addition = %v", operationErr)
	}
	got, readErr := os.ReadFile(file)
	if readErr != nil || string(got) != string(after) {
		t.Fatalf("refactored source = %q, err %v; want %q", got, readErr, after)
	}
	otherSource, otherErr := os.ReadFile(other)
	if otherErr != nil || string(otherSource) != "fn other() {}\n" {
		t.Fatalf("unrelated source = %q, err %v; want newly added source unchanged", otherSource, otherErr)
	}
}

func rustExtractAction(file string, version *int, command bool) json.RawMessage {
	edit := rustExtractWorkspaceEdit(file, version)
	action := map[string]any{"title": "Extract into variable", "kind": "refactor.extract", "data": map[string]any{"id": "extract_variable:RefactorExtract:0"}}
	if command {
		action["command"] = map[string]any{"command": "rust-analyzer.rename"}
	} else if version != nil {
		action["edit"] = edit
	}
	return mustJSON([]any{action})
}

func rustExtractWorkspaceEdit(file string, version *int) json.RawMessage {
	if version == nil {
		return mustJSON(map[string]any{"changes": map[string]any{pathutil.FileURI(file): []any{}}})
	}
	uri := pathutil.FileURI(file)
	return mustJSON(map[string]any{"documentChanges": []any{map[string]any{
		"textDocument": map[string]any{"uri": uri, "version": *version},
		"edits": []any{
			map[string]any{"range": fakeRustRange(1, 4, 4), "newText": "let extracted = 1_i32 + 2_i32;\n    "},
			map[string]any{"range": fakeRustRange(1, 12, 25), "newText": "extracted"},
		},
	}}, "changeAnnotations": map[string]any{
		"workspace": map[string]any{"label": "Edit outside of the workspace"},
	}})
}

func TestRustWorkspaceEditsAllowUnusedAnnotationsButRejectReferences(t *testing.T) {
	file := filepath.Join(t.TempDir(), "lib.rs")
	uri := pathutil.FileURI(file)
	annotation := map[string]any{"workspace": map[string]any{"label": "Edit outside of the workspace"}}
	edit := map[string]any{"range": fakeRustRange(0, 0, 5), "newText": "value"}
	workspaceEdit := map[string]any{
		"changes":           map[string]any{uri: []any{edit}},
		"changeAnnotations": annotation,
	}
	if _, err := decodeRustWorkspaceEdits(file, mustJSON(workspaceEdit)); err != nil {
		t.Fatalf("decodeRustWorkspaceEdits() rejected an unused annotation: %v", err)
	}
	edit["annotationId"] = "workspace"
	if _, err := decodeRustWorkspaceEdits(file, mustJSON(workspaceEdit)); err == nil {
		t.Fatal("decodeRustWorkspaceEdits() accepted a referenced annotation")
	}
}

func TestRustExtractConstantAppliesExactTypedAliasEdit(t *testing.T) {
	root := t.TempDir()
	canonicalRoot, canonicalErr := filepath.EvalSymlinks(root)
	if canonicalErr != nil {
		t.Fatal(canonicalErr)
	}
	root = canonicalRoot
	if manifestErr := os.WriteFile(filepath.Join(root, "Cargo.toml"), []byte("[package]\nname = \"extract-constant\"\nversion = \"0.1.0\"\n"), 0o600); manifestErr != nil {
		t.Fatal(manifestErr)
	}
	file := filepath.Join(root, "lib.rs")
	before := "fn run() {\n    let total = 1_i32 + 2_i32;\n    consume(total);\n}\n\nfn consume(value: i32) {}\n"
	if writeErr := os.WriteFile(file, []byte(before), 0o600); writeErr != nil {
		t.Fatal(writeErr)
	}
	uri := pathutil.FileURI(file)
	version := 1
	edit := map[string]any{"documentChanges": []any{map[string]any{
		"textDocument": map[string]any{"uri": uri, "version": version},
		"edits": []any{
			map[string]any{"range": fakeRustRange(1, 4, 7), "newText": "const"},
			map[string]any{"range": fakeRustRange(1, 8, 13), "newText": "EXTRACTED:"},
			map[string]any{"range": fakeRustRange(1, 14, 14), "newText": "i32 "},
			map[string]any{"range": fakeRustRange(2, 4, 4), "newText": "let total = EXTRACTED;\n    "},
		},
	}}}
	resolved := map[string]any{
		"title": "Extract into constant", "kind": rustRefactorExtract,
		"edit": edit,
	}
	session := &extractRustSession{
		action:   mustJSON([]any{map[string]any{"title": "Extract into constant", "kind": rustRefactorExtract, "data": map[string]any{"id": rustExtractConstantActionID, "version": 1}}}),
		resolved: mustJSON(resolved),
		prepareErrs: map[rustPosition]error{
			{Line: 1, Character: 21}: errors.New("JSON-RPC error -32602: No references found at position"),
		},
		rename: mustJSON(map[string]any{"documentChanges": []any{map[string]any{
			"textDocument": map[string]any{"uri": uri, "version": 2},
			"edits": []any{
				map[string]any{"range": fakeRustRange(1, 10, 19), "newText": "RETRY_MAX"},
				map[string]any{"range": fakeRustRange(2, 16, 25), "newText": "RETRY_MAX"},
			},
		}}}),
	}
	request := backend.CodeActionRequest{
		Operation: backend.OperationExtractVariable,
		Project:   backend.ProjectContext{RootDir: root, File: file, WorkspaceTrust: backend.NewWorkspaceTrust(root, true)},
		File:      file, Range: backend.Range{Start: backend.Position{Line: 1, Character: 16}, End: backend.Position{Line: 1, Character: 29}},
		Kind: "constant", NewName: "RETRY_MAX", Expression: "1_i32 + 2_i32",
	}
	service := NewRustBackendWithFactory(func(context.Context, string) (RustSession, error) { return session, nil })
	result, err := service.ApplyCodeAction(context.Background(), request)
	if err != nil {
		t.Fatalf("ApplyCodeAction() error = %v", err)
	}
	want := "fn run() {\n    const RETRY_MAX: i32 = 1_i32 + 2_i32;\n    let total = RETRY_MAX;\n    consume(total);\n}\n\nfn consume(value: i32) {}\n"
	if string(result.Before) != before || string(result.After) != want {
		t.Fatalf("extract constant before/after = %q / %q, want %q / %q", result.Before, result.After, before, want)
	}
	if slices.Index(session.calls, "textDocument/codeAction") < 0 || slices.Index(session.calls, "codeAction/resolve") < 0 || slices.Index(session.calls, "textDocument/rename") < 0 {
		t.Fatalf("Rust extract-constant protocol calls = %v", session.calls)
	}
	if len(session.resolveArgs) != 1 {
		t.Fatalf("resolve arguments = %s, want original opaque action", mustJSON(session.resolveArgs))
	}
	var resolvedOriginal struct {
		Data map[string]any `json:"data"`
	}
	if err := json.Unmarshal(session.resolveArgs[0], &resolvedOriginal); err != nil || resolvedOriginal.Data["id"] != rustExtractConstantActionID || resolvedOriginal.Data["version"] != float64(1) {
		t.Fatalf("resolved request identity = %+v, err %v", resolvedOriginal.Data, err)
	}
	if len(session.prepareArgs) != 1 {
		t.Fatalf("prepareRename arguments = %s, want only the introduced-use binding rename", mustJSON(session.prepareArgs))
	}
	var bindingPrepare struct {
		Position rustPosition `json:"position"`
	}
	if err := json.Unmarshal(session.prepareArgs[0], &bindingPrepare); err != nil || bindingPrepare.Position != (rustPosition{Line: 2, Character: 16}) {
		t.Fatalf("introduced-use prepareRename position = %+v, err %v", bindingPrepare.Position, err)
	}
	assertRustDocumentLifecycle(t, session, file, true)
}

func TestRustExtractConstantRefusesCommentedTypeAliasBeforePrimitivePrepareRename(t *testing.T) {
	root := t.TempDir()
	root, canonicalErr := filepath.EvalSymlinks(root)
	if canonicalErr != nil {
		t.Fatal(canonicalErr)
	}
	if manifestErr := os.WriteFile(filepath.Join(root, "Cargo.toml"), []byte("[package]\nname = \"extract-constant-alias\"\nversion = \"0.1.0\"\n"), 0o600); manifestErr != nil {
		t.Fatal(manifestErr)
	}
	file := filepath.Join(root, "lib.rs")
	before := "type /* alias */ i32 = u64;\nfn run() {\n    let total = 1_i32 + 2_i32;\n    let _ = total;\n}\n"
	if writeErr := os.WriteFile(file, []byte(before), 0o600); writeErr != nil {
		t.Fatal(writeErr)
	}
	request := backend.CodeActionRequest{
		Operation: backend.OperationExtractVariable,
		Project:   backend.ProjectContext{RootDir: root, File: file, WorkspaceTrust: backend.NewWorkspaceTrust(root, true)},
		File:      file, Range: backend.Range{Start: backend.Position{Line: 2, Character: 16}, End: backend.Position{Line: 2, Character: 29}},
		Kind: "constant", NewName: "RETRY_MAX", Expression: "1_i32 + 2_i32",
	}
	service := NewRustBackendWithFactory(func(context.Context, string) (RustSession, error) {
		t.Fatal("shadowed primitive type reached rust-analyzer")
		return nil, errors.New("unexpected session creation")
	})
	if _, applyErr := service.ApplyCodeAction(context.Background(), request); applyErr == nil || !strings.Contains(applyErr.Error(), "possibly shadowed primitive type") {
		t.Fatalf("ApplyCodeAction() shadowed alias error = %v", applyErr)
	}
	got, readErr := os.ReadFile(file)
	if readErr != nil || string(got) != before {
		t.Fatalf("shadowed alias source changed to %q, err %v", got, readErr)
	}
}

func TestRustExtractConstantSelectionAndCandidateRefuseUnprovenShapes(t *testing.T) {
	source := []byte("fn run() {\n    let total = 1_i32 + 2_i32;\n    consume(total);\n}\n")
	selected := backend.Range{Start: backend.Position{Line: 1, Character: 16}, End: backend.Position{Line: 1, Character: 29}}
	start, end, insertionStart, indent, localName, valueType, selectionErr := resolveRustConstantSelection(source, selected, "1_i32 + 2_i32")
	if selectionErr != nil || valueType != "i32" || localName != "total" || string(indent) != "    " {
		t.Fatalf("resolveRustConstantSelection() = %d, %d, %d, %q, %q, %q, %v", start, end, insertionStart, indent, localName, valueType, selectionErr)
	}
	if _, _, _, _, _, _, runtimeErr := resolveRustConstantSelection(source, selected, "make_value()"); runtimeErr == nil {
		t.Fatal("resolveRustConstantSelection() accepted a runtime expression")
	}
	multiline := []byte("fn run() {\n    let total = 1_i32 +\n        2_i32;\n    consume(total);\n}\n")
	multilineRange := backend.Range{Start: backend.Position{Line: 1, Character: 16}, End: backend.Position{Line: 2, Character: 13}}
	if _, _, _, _, _, _, multilineErr := resolveRustConstantSelection(multiline, multilineRange, "1_i32 +\n        2_i32"); multilineErr == nil {
		t.Fatal("resolveRustConstantSelection() accepted a multi-line expression")
	}
	edits := []rustTextEdit{
		{Range: fakeRustRange(1, 4, 7), NewText: "const"},
		{Range: fakeRustRange(1, 8, 13), NewText: "EXTRACTED:"},
		{Range: fakeRustRange(1, 14, 14), NewText: "i32 "},
		{Range: fakeRustRange(2, 4, 4), NewText: "let total = EXTRACTED;\n    "},
	}
	candidate, position, use, binding, candidateErr := prepareRustExtractConstantCandidate(source, edits, start, end, insertionStart, indent, localName, "1_i32 + 2_i32", valueType)
	if candidateErr != nil {
		t.Fatalf("prepareRustExtractConstantCandidate() error = %v", candidateErr)
	}
	if string(candidate) != "fn run() {\n    const EXTRACTED: i32 = 1_i32 + 2_i32;\n    let total = EXTRACTED;\n    consume(total);\n}\n" || position.Line != 2 || position.Character != 16 || string(candidate[binding.start:binding.end]) != "EXTRACTED" || string(candidate[use.start:use.end]) != "EXTRACTED" {
		t.Fatalf("constant candidate/rename spans = %q / %+v / %+v / %+v", candidate, position, use, binding)
	}
	corrupt := append([]rustTextEdit(nil), edits...)
	corrupt[2].NewText = "u32 "
	if _, _, _, _, err := prepareRustExtractConstantCandidate(source, corrupt, start, end, insertionStart, indent, localName, "1_i32 + 2_i32", valueType); err == nil {
		t.Fatal("prepareRustExtractConstantCandidate() accepted a type-mismatched declaration")
	}
	foreign := append([]rustTextEdit(nil), edits...)
	foreign[3] = rustTextEdit{Range: fakeRustRange(0, 0, 0), NewText: edits[3].NewText}
	if _, _, _, _, err := prepareRustExtractConstantCandidate(source, foreign, start, end, insertionStart, indent, localName, "1_i32 + 2_i32", valueType); err == nil {
		t.Fatal("prepareRustExtractConstantCandidate() accepted a foreign insertion location")
	}
}

func TestRustVisibleGenericTypeParameterUsesBalancedLexicalScopes(t *testing.T) {
	for name, source := range map[string]string{
		"nested bound and comment": `fn run<T: Into<u8>, /* outer /* nested */ tail */ i32>() { let total = 1_i32 + 2_i32; let _ = total; }`,
		"nested bound with arrow":  `fn run<T: Fn() -> i32, i32>() { let total = 1_i32 + 2_i32; let _ = total; }`,
		"impl parameter":           `struct Widget; impl<i32> Widget { fn run() { let total = 1_i32 + 2_i32; let _ = total; } }`,
		"trait parameter":          `trait Widget<i32> { fn run() { let total = 1_i32 + 2_i32; let _ = total; } }`,
	} {
		t.Run(name, func(t *testing.T) {
			position := strings.Index(source, "1_i32 + 2_i32")
			if position < 0 || !rustVisibleGenericTypeParameter([]byte(source), position, "i32") {
				t.Fatal("rustVisibleGenericTypeParameter() missed an enclosing generic type parameter")
			}
		})
	}
	unshadowed := `fn other<i32>() {} fn run() { let total = 1_i32 + 2_i32; let _ = total; }`
	position := strings.Index(unshadowed, "1_i32 + 2_i32")
	if rustVisibleGenericTypeParameter([]byte(unshadowed), position, "i32") {
		t.Fatal("rustVisibleGenericTypeParameter() treated an unrelated generic parameter as visible")
	}
}

func TestRustSourceGenericShadowScannerRejectsAnyGenericOwner(t *testing.T) {
	source := `struct Widget<i32>([u8; { fn run() { let total = 1_i32 + 2_i32; let _ = total; } 0 }], i32);`
	if !rustSourceHasGenericTypeParameter([]byte(source), "i32") {
		t.Fatal("rustSourceHasGenericTypeParameter() missed an enclosing tuple-struct generic")
	}
	unrelated := `fn other<i32>() {} fn run() { let total = 1_i32 + 2_i32; let _ = total; }`
	if !rustSourceHasGenericTypeParameter([]byte(unrelated), "i32") {
		t.Fatal("rustSourceHasGenericTypeParameter() did not conservatively reject an unrelated generic owner")
	}
	clean := `fn run<T>() { let total = 1_i32 + 2_i32; let _ = total; }`
	if rustSourceHasGenericTypeParameter([]byte(clean), "i32") {
		t.Fatal("rustSourceHasGenericTypeParameter() treated a different generic parameter as i32")
	}
}

func TestRustPrimitiveTypeShadowScannerRefusesVisibleTypeBindingsAndImports(t *testing.T) {
	for name, source := range map[string]string{
		"comment-separated alias": `type /* alias */ i32 = u64; fn run() { let total = 1_i32 + 2_i32; let _ = total; }`,
		"struct binding":          `struct i32; fn run() { let total = 1_i32 + 2_i32; let _ = total; }`,
		"enum binding":            `enum i32 { Value } fn run() { let total = 1_i32 + 2_i32; let _ = total; }`,
		"union binding":           `union i32 { value: u64 } fn run() { let total = 1_i32 + 2_i32; let _ = total; }`,
		"module binding":          `mod i32 {} fn run() { let total = 1_i32 + 2_i32; let _ = total; }`,
		"trait binding":           `trait i32 {} fn run() { let total = 1_i32 + 2_i32; let _ = total; }`,
		"explicit imported alias": `mod types { pub type i32 = u64; } use types::i32; fn run() { let total = 1_i32 + 2_i32; let _ = total; }`,
		"nested use tree alias":   `mod types { pub type Other = u64; } use types::{Other as i32}; fn run() { let total = 1_i32 + 2_i32; let _ = total; }`,
		"ancestor module import":  `mod parent { type i32 = u64; mod child { use super::i32; fn run() { let total = 1_i32 + 2_i32; let _ = total; } } }`,
		"extern crate alias":      `extern crate types as i32; fn run() { let total = 1_i32 + 2_i32; let _ = total; }`,
	} {
		t.Run(name, func(t *testing.T) {
			if !rustSourceMayShadowPrimitiveType([]byte(source), "i32") {
				t.Fatal("rustSourceMayShadowPrimitiveType() missed a possible type-namespace binding")
			}
		})
	}
	clean := `fn run() { let i32 = 1_i32; let total = 1_i32 + 2_i32; let _ = total; }`
	if rustSourceMayShadowPrimitiveType([]byte(clean), "i32") {
		t.Fatal("rustSourceMayShadowPrimitiveType() treated a value binding as a type binding")
	}
}

func TestSelectRustExtractConstantActionRequiresNativeIdentity(t *testing.T) {
	for _, id := range []string{
		"extract_variable:RefactorExtract:0",
		"extract_constant:RefactorExtract:0",
		"extract_constant:RefactorExtract:1",
		"extract_constant:Unrelated:0",
		"extract_static:RefactorExtract:0",
		"promote_local_to_const:RefactorExtract:0",
	} {
		raw := mustJSON([]any{map[string]any{
			"title": "Extract into constant", "kind": rustRefactorExtract,
			"data": map[string]any{"id": id},
		}})
		if _, _, err := selectRustExtractConstantAction(raw); err == nil {
			t.Errorf("selectRustExtractConstantAction() accepted unrelated identity %q", id)
		}
	}
	raw := mustJSON([]any{map[string]any{
		"title": "Extract into constant", "kind": rustRefactorExtract,
		"data": map[string]any{"id": rustExtractConstantActionID},
	}})
	if _, _, err := selectRustExtractConstantAction(raw); err != nil {
		t.Fatalf("selectRustExtractConstantAction() rejected native action: %v", err)
	}
}

func TestPrepareRustInlineCandidateRequiresAndComputesExactLiteralUse(t *testing.T) {
	source := []byte("fn run() {\n    let value = 42_i32;\n    consume(value, value);\n}\n")
	selected := fakeRustRange(1, 8, 13)
	got, err := prepareRustInlineCandidate(source, backend.Range{Start: backend.Position{Line: 1, Character: 8}, End: backend.Position{Line: 1, Character: 13}}, "value")
	if err != nil {
		t.Fatalf("prepareRustInlineCandidate() error = %v", err)
	}
	want := "fn run() {\n    consume(42_i32, 42_i32);\n}\n"
	if string(got) != want {
		t.Fatalf("candidate = %q, want %q (selected %v)", got, want, selected)
	}
}

func TestPrepareRustInlineCandidateRewritesOnlyExactArguments(t *testing.T) {
	source := []byte("fn run() {\n    let value = 42_i32;\n    consume_value(value);\n}\n")
	got, err := prepareRustInlineCandidate(source, backend.Range{Start: backend.Position{Line: 1, Character: 8}, End: backend.Position{Line: 1, Character: 13}}, "value")
	if err != nil {
		t.Fatalf("prepareRustInlineCandidate() error = %v", err)
	}
	want := "fn run() {\n    consume_value(42_i32);\n}\n"
	if string(got) != want {
		t.Fatalf("candidate = %q, want %q", got, want)
	}
}

func TestPrepareRustInlineCandidateRefusesUnsafeShapes(t *testing.T) {
	for _, source := range []string{
		"fn run() {\n    let value = make();\n    consume(value);\n}\n",
		"fn run() {\n    let value: i32 = 42_i32;\n    consume(value);\n}\n",
		"fn run() {\n    let value = 42;\n    consume(value);\n}\n",
		"fn run() {\n    let value = 42_i32;\n    consume(value);\n    use_value(value);\n}\n",
	} {
		_, err := prepareRustInlineCandidate([]byte(source), backend.Range{Start: backend.Position{Line: 1, Character: 8}, End: backend.Position{Line: 1, Character: 13}}, "value")
		if err == nil {
			t.Errorf("prepareRustInlineCandidate(%q) unexpectedly succeeded", source)
		}
	}
}

func fakeRustRange(line, start, end int) rustRange {
	return rustRange{Start: rustPosition{Line: line, Character: start}, End: rustPosition{Line: line, Character: end}}
}

func mustJSON(value any) json.RawMessage {
	encoded, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return encoded
}

func valueOr(value *int, fallback int) int {
	if value != nil {
		return *value
	}
	return fallback
}

func containsString(values []string, value string) bool {
	return slices.Contains(values, value)
}

func assertRustDocumentLifecycle(t *testing.T, session *extractRustSession, file string, actionExpected bool) {
	t.Helper()
	wantURI := pathutil.FileURI(file)
	if session.openedURI != wantURI || session.closedURI != wantURI || session.closed != 1 {
		t.Fatalf("document lifecycle URIs/count = %q / %q / %d, want %q / %q / 1", session.openedURI, session.closedURI, session.closed, wantURI, wantURI)
	}
	openAt, actionAt, closeAt := slices.Index(session.calls, "textDocument/didOpen"), slices.Index(session.calls, "textDocument/codeAction"), slices.Index(session.calls, "textDocument/didClose")
	if openAt < 0 || closeAt <= openAt || actionExpected && (actionAt <= openAt || closeAt <= actionAt) || !actionExpected && actionAt >= 0 {
		t.Fatalf("document lifecycle order = %v", session.calls)
	}
}

type inlineStatusRustSession struct {
	*extractRustSession

	status json.RawMessage
}

func (s *inlineStatusRustSession) LatestServerStatus() json.RawMessage { return s.status }

func (s *inlineStatusRustSession) ServerStatuses() <-chan json.RawMessage {
	return make(chan json.RawMessage)
}

func rustInlineProtocolFixture(t *testing.T, operation backend.Operation) (string, string, *inlineStatusRustSession, backend.CodeActionRequest, []byte, []byte) {
	t.Helper()
	root := t.TempDir()
	root, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "Cargo.toml"), []byte("[package]\nname = \"inline\"\nversion = \"0.1.0\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(root, "lib.rs")
	var before, after []byte
	var request backend.CodeActionRequest
	if operation == backend.OperationInlineVariable {
		before = []byte("fn run() {\n    let value = 5_i32;\n    consume(value);\n}\n")
		after = []byte("fn run() {\n    consume(5_i32);\n}\n")
		request = backend.CodeActionRequest{Operation: operation, Kind: rustRefactorInline, Symbol: "value", Range: backend.Range{Start: backend.Position{Line: 1, Character: 8}, End: backend.Position{Line: 1, Character: 13}}}
	} else {
		before = []byte("fn get() -> i32 { 5_i32 }\nfn run() { get() }\n")
		after = []byte("fn get() -> i32 { 5_i32 }\nfn run() { 5_i32 }\n")
		request = backend.CodeActionRequest{Operation: operation, Kind: rustRefactorInline, Symbol: "get", EnclosingSymbol: "run", Expression: "5_i32", Range: backend.Range{Start: backend.Position{Line: 1, Character: 11}, End: backend.Position{Line: 1, Character: 14}}}
	}
	if err := os.WriteFile(file, before, 0o600); err != nil {
		t.Fatal(err)
	}
	request.Project = backend.ProjectContext{RootDir: root, File: file, WorkspaceTrust: backend.NewWorkspaceTrust(root, true)}
	request.File = file
	session := &inlineStatusRustSession{extractRustSession: &extractRustSession{}, status: json.RawMessage(`{"health":"ok","quiescent":true}`)}
	uri := pathutil.FileURI(file)
	var edit map[string]any
	if operation == backend.OperationInlineVariable {
		edit = map[string]any{"range": (rustRange{Start: rustPosition{Line: 0}, End: rustPosition{Line: 3, Character: 1}}), "newText": strings.TrimSuffix(string(after), "\n")}
	} else {
		edit = map[string]any{"range": fakeRustRange(1, 11, 16), "newText": "5_i32"}
	}
	session.action = mustJSON([]any{map[string]any{"title": map[bool]string{true: "Inline variable", false: "Inline `get`"}[operation == backend.OperationInlineVariable], "kind": rustRefactorInline, "data": map[string]any{"id": map[bool]string{true: "inline_local_variable:RefactorInline:0", false: "inline_call:RefactorInline:0"}[operation == backend.OperationInlineVariable]}, "edit": map[string]any{"documentChanges": []any{map[string]any{"textDocument": map[string]any{"uri": uri, "version": 1}, "edits": []any{edit}}}}}})
	{
		var actions []rustCodeAction
		if err := json.Unmarshal(session.action, &actions); err != nil || len(actions) != 1 {
			t.Fatalf("inline action fixture decode = %v, actions %d", err, len(actions))
		}
		if operation == backend.OperationInlineVariable {
			if _, err := decodeRustSingleFileWorkspaceEdit(file, actions[0].Edit, 1); err != nil {
				t.Fatalf("inline variable fixture edit decode: %v", err)
			}
		} else if _, err := decodeRustWorkspaceEdits(file, actions[0].Edit); err != nil {
			t.Fatalf("inline function fixture edit decode: %v", err)
		}
	}
	if operation == backend.OperationInlineVariable {
		expected, err := prepareRustInlineCandidate(before, request.Range, request.Symbol)
		if err != nil {
			t.Fatalf("inline variable fixture candidate: %v", err)
		}
		if _, err := resolveRustInlineAction(context.Background(), session, file, before, session.action, expected); err != nil {
			t.Fatalf("inline variable fixture action: %v", err)
		}
	} else {
		expected, start, end, err := expectedRustInlineCall(before, request)
		if err != nil {
			t.Fatalf("inline function fixture candidate: %v", err)
		}
		if _, err := resolveRustInlineFunctionAction(context.Background(), session, file, before, session.action, request.Symbol, start, end, expected); err != nil {
			t.Fatalf("inline function fixture action: %v", err)
		}
	}
	return root, file, session, request, before, after
}

func TestRustInlineOperationsShareProtocolLifecycle(t *testing.T) {
	for _, operation := range []backend.Operation{backend.OperationInlineVariable, backend.OperationInlineFunction} {
		t.Run(string(operation), func(t *testing.T) {
			for _, scenario := range []string{"readiness failure", "action failure", "close failure", "action and close failure"} {
				t.Run(scenario, func(t *testing.T) {
					assertRustInlineProtocolFailure(t, operation, scenario)
				})
			}
			t.Run("success stages exact postimage in caller buffer", func(t *testing.T) {
				assertRustInlineBufferedSuccess(t, operation)
			})
		})
	}
}

func assertRustInlineProtocolFailure(t *testing.T, operation backend.Operation, scenario string) {
	t.Helper()
	_, file, session, request, before, _ := rustInlineProtocolFixture(t, operation)
	switch scenario {
	case "readiness failure":
		session.status = json.RawMessage(`{"health":"warning","quiescent":true,"message":"cargo metadata failed"}`)
	case "action failure":
		session.action = json.RawMessage(`[]`)
	case "close failure":
		session.closeErr = errors.New("close transport failed")
	case "action and close failure":
		session.action = json.RawMessage(`[]`)
		session.closeErr = errors.New("close transport failed")
	default:
		t.Fatalf("unknown failure scenario %q", scenario)
	}
	backendUnderTest := NewRustBackendWithFactory(func(context.Context, string) (RustSession, error) { return session, nil })
	if _, operationErr := backendUnderTest.ApplyCodeAction(context.Background(), request); operationErr == nil {
		t.Fatal("ApplyCodeAction() unexpectedly succeeded")
	} else if scenario == "action and close failure" && (!strings.Contains(operationErr.Error(), "close transport failed") || !strings.Contains(operationErr.Error(), "invalid rust rename edit")) {
		t.Fatalf("joined action and close error = %v", operationErr)
	}
	diskSource, readErr := os.ReadFile(file)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(diskSource) != string(before) {
		t.Fatalf("failed operation changed disk source: %q", diskSource)
	}
	if scenario == "close failure" && backendUnderTest.session != nil {
		t.Fatal("backend retained a session after close failure")
	}
	if scenario == "readiness failure" && containsString(session.calls, "textDocument/codeAction") {
		t.Fatal("code action was requested before readiness")
	}
	if !containsString(session.calls, "textDocument/didClose") {
		t.Fatal("opened document was not closed")
	}
	assertRustDocumentLifecycle(t, session.extractRustSession, file, scenario != "readiness failure")
}

func assertRustInlineBufferedSuccess(t *testing.T, operation backend.Operation) {
	t.Helper()
	root, file, session, request, before, after := rustInlineProtocolFixture(t, operation)
	buffer, bufferErr := sourcebuf.New(context.Background(), root)
	if bufferErr != nil {
		t.Fatal(bufferErr)
	}
	if authorizeErr := buffer.AuthorizeWrites(context.Background(), root); authorizeErr != nil {
		t.Fatal(authorizeErr)
	}
	ctx := sourcebuf.WithBuffer(context.Background(), buffer)
	backendUnderTest := NewRustBackendWithFactory(func(context.Context, string) (RustSession, error) { return session, nil })
	result, operationErr := backendUnderTest.ApplyCodeAction(ctx, request)
	if operationErr != nil {
		t.Fatalf("ApplyCodeAction() error = %v", operationErr)
	}
	if !containsString(session.calls, "textDocument/codeAction") {
		t.Fatal("missing textDocument/codeAction request")
	}
	if string(result.After) != string(after) {
		t.Fatalf("result postimage = %q, want %q", result.After, after)
	}
	diskSource, readErr := os.ReadFile(file)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(diskSource) != string(before) {
		t.Fatalf("buffered operation changed disk source: %q", diskSource)
	}
	stagedSource, stagedErr := sourcebuf.ReadFile(ctx, file)
	if stagedErr != nil {
		t.Fatal(stagedErr)
	}
	if string(stagedSource) != string(after) {
		t.Fatalf("staged source = %q, want %q", stagedSource, after)
	}
	if !containsString(session.calls, "textDocument/didClose") {
		t.Fatal("opened document was not closed")
	}
	assertRustDocumentLifecycle(t, session.extractRustSession, file, true)
}

type cancelInlineRustSession struct {
	*inlineStatusRustSession

	cancel          context.CancelFunc
	closeContextErr error
}

func (s *cancelInlineRustSession) Request(ctx context.Context, method string, params any) (json.RawMessage, error) {
	response, requestErr := s.inlineStatusRustSession.Request(ctx, method, params)
	if method == "textDocument/codeAction" {
		s.cancel()
	}
	return response, requestErr
}

func (s *cancelInlineRustSession) Notify(ctx context.Context, method string, params any) error {
	if method == "textDocument/didClose" {
		s.closeContextErr = ctx.Err()
	}
	return s.inlineStatusRustSession.Notify(ctx, method, params)
}

func TestRustInlineCleanupUsesLiveContext(t *testing.T) {
	for _, operation := range []backend.Operation{backend.OperationInlineVariable, backend.OperationInlineFunction} {
		t.Run(string(operation), func(t *testing.T) {
			assertRustInlineCancellationCleanup(t, operation)
		})
	}
}

func assertRustInlineCancellationCleanup(t *testing.T, operation backend.Operation) {
	t.Helper()
	_, _, baseSession, request, _, _ := rustInlineProtocolFixture(t, operation)
	baseSession.action = json.RawMessage(`[]`)
	ctx, cancel := context.WithCancel(context.Background())
	session := &cancelInlineRustSession{inlineStatusRustSession: baseSession, cancel: cancel}
	backendUnderTest := NewRustBackendWithFactory(func(context.Context, string) (RustSession, error) { return session, nil })
	if _, operationErr := backendUnderTest.ApplyCodeAction(ctx, request); operationErr == nil {
		t.Fatal("ApplyCodeAction() unexpectedly succeeded with an invalid resolver response")
	}
	if ctx.Err() == nil {
		t.Fatal("fake code action request did not cancel the operation context")
	}
	if session.closeContextErr != nil {
		t.Fatalf("didClose context error = %v, want live context", session.closeContextErr)
	}
	if session.closed != 1 {
		t.Fatalf("didClose count = %d, want 1", session.closed)
	}
}
