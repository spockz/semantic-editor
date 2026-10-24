// WHY: Proves that JDT LS workspace refactorings are staged and published across every affected Java file.
package java

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spockz/semantic-editor/internal/backend"
	"github.com/spockz/semantic-editor/internal/backend/pathutil"
)

type workspaceCodeActionSession struct {
	selectedURI string
	callerURI   string
}

type changeSignatureTestSession struct {
	workspaceCodeActionSession

	refactorParameters map[string]any
}

type moveStaticMemberTestSession struct {
	workspaceCodeActionSession

	params map[string]any
}

func TestSelectJavaCodeActionPrefersSingleOccurrenceExtraction(t *testing.T) {
	raw := json.RawMessage(`[{"title":"Extract to local variable (replace all occurrences)","kind":"refactor.extract.variable"},{"title":"Extract to local variable","kind":"refactor.extract.variable"}]`)
	action, err := selectJavaCodeAction(raw, "refactor.extract.variable")
	if err != nil || action.Title != "Extract to local variable" {
		t.Fatalf("selected action = %#v, err = %v", action, err)
	}

	raw = json.RawMessage(`[{"title":"Disabled extract","kind":"refactor.extract.variable","disabled":{"reason":"not applicable"}},{"title":"Enabled extract","kind":"refactor.extract.variable"}]`)
	action, err = selectJavaCodeAction(raw, "refactor.extract.variable")
	if err != nil || action.Title != "Enabled extract" {
		t.Fatalf("selected action = %#v, err = %v", action, err)
	}

	raw = json.RawMessage(`[{"title":"Disabled extract","kind":"refactor.extract.variable","disabled":{"reason":"not applicable"}}]`)
	if _, err := selectJavaCodeAction(raw, "refactor.extract.variable"); err == nil || !strings.Contains(err.Error(), "only disabled actions") {
		t.Fatalf("disabled-only actions error = %v", err)
	}
}

func TestSelectJavaChangedMethodSourceRange(t *testing.T) {
	source := []byte("class Service { void send(int id) {}\nvoid send(int id, boolean retry) {}\n}")
	oneArg := "int"
	twoArgs := "int, boolean"
	method := func(signature string, start, end backend.Position) backend.ReadSymbol {
		return backend.ReadSymbol{Kind: "method", QualifiedName: "Service.send", Signature: &signature, Range: backend.Range{Start: start, End: end}}
	}
	startLine1 := backend.Position{Line: 0, Character: 16}
	endLine1 := backend.Position{Line: 0, Character: 36}
	startLine2 := backend.Position{Line: 1, Character: 0}
	endLine2 := backend.Position{Line: 1, Character: 35}
	t.Run("overloads select expected parameter count", func(t *testing.T) {
		got, err := selectJavaChangedMethodSourceRange(source, []backend.ReadSymbol{method(oneArg, startLine1, endLine1), method(twoArgs, startLine2, endLine2)}, 2, "", "found %d", "invalid range")
		if err != nil || got.start != len("class Service { void send(int id) {}\n") || got.end != len(source)-2 {
			t.Fatalf("range = %#v, err = %v", got, err)
		}
	})
	t.Run("zero matches", func(t *testing.T) {
		_, err := selectJavaChangedMethodSourceRange(source, []backend.ReadSymbol{method(oneArg, startLine1, endLine1)}, 3, "", "found %d", "invalid range")
		if err == nil || err.Error() != "found 0" {
			t.Fatalf("error = %v", err)
		}
	})
	t.Run("multiple matches", func(t *testing.T) {
		_, err := selectJavaChangedMethodSourceRange(source, []backend.ReadSymbol{method(oneArg, startLine1, endLine1), method("String", startLine2, endLine2)}, 1, "", "found %d", "invalid range")
		if err == nil || err.Error() != "found 2" {
			t.Fatalf("error = %v", err)
		}
	})
	t.Run("invalid range", func(t *testing.T) {
		_, err := selectJavaChangedMethodSourceRange(source, []backend.ReadSymbol{method(oneArg, endLine1, startLine1)}, 1, "", "found %d", "invalid range")
		if err == nil || err.Error() != "invalid range" {
			t.Fatalf("error = %v", err)
		}
	})
	t.Run("parameter object rejects same-count wrong type", func(t *testing.T) {
		_, err := selectJavaChangedMethodSourceRange(source, []backend.ReadSymbol{method("OtherRequest", startLine1, endLine1)}, 1, "SendRequest", "found %d for %q", "invalid range")
		if err == nil || err.Error() != `found 0 for "SendRequest"` {
			t.Fatalf("error = %v", err)
		}
	})
}

func (s *moveStaticMemberTestSession) Request(ctx context.Context, method string, params any) (json.RawMessage, error) {
	if method != "java/move" {
		return s.workspaceCodeActionSession.Request(ctx, method, params)
	}
	encoded, err := json.Marshal(params)
	if err != nil {
		return nil, err
	}
	if decodeErr := json.Unmarshal(encoded, &s.params); decodeErr != nil {
		return nil, decodeErr
	}
	edit := map[string]any{"changes": map[string]any{
		s.selectedURI: []any{map[string]any{"range": javaRange{Start: javaPosition{Line: 1, Character: 15}, End: javaPosition{Line: 1, Character: 32}}, "newText": ""}},
		s.callerURI:   []any{map[string]any{"range": javaRange{Start: javaPosition{Line: 1, Character: 14}, End: javaPosition{Line: 1, Character: 14}}, "newText": "\n  static int value() { return 1; }\n"}},
	}}
	encodedEdit, err := json.Marshal(edit)
	if err != nil {
		return nil, err
	}
	response, err := json.Marshal(map[string]any{"edit": json.RawMessage(encodedEdit)})
	return response, err
}

func (s *changeSignatureTestSession) Request(ctx context.Context, method string, params any) (json.RawMessage, error) {
	switch method {
	case "java/getChangeSignatureInfo":
		return json.RawMessage(`{"methodIdentifier":"method-handle","methodName":"send","modifier":"public","returnType":"void","parameters":[{"originalIndex":0,"name":"id","type":"int"}],"exceptions":[]}`), nil
	case "java/getRefactorEdit":
		encoded, err := json.Marshal(params)
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal(encoded, &s.refactorParameters); err != nil {
			return nil, err
		}
		return json.RawMessage(`{"edit":{"changes":{"file:///tmp/Service.java":[]}}}`), nil
	default:
		return s.workspaceCodeActionSession.Request(ctx, method, params)
	}
}

func TestPrepareJavaChangeSignatureEditUsesJDTRefactoringContract(t *testing.T) {
	session := &changeSignatureTestSession{}
	info := javaCommand{Command: "java.apply.refactoring.command", Arguments: []json.RawMessage{json.RawMessage(`"changeSignature"`), json.RawMessage(`{}`), json.RawMessage(`null`)}}
	contextParams := map[string]any{"range": javaRange{}, "textDocument": map[string]any{"uri": "file:///tmp/Service.java"}}
	sourceIndex := 0
	edit, err := prepareJavaChangeSignatureEdit(context.Background(), session, contextParams, javaCodeAction{Command: &info}, []backend.CodeActionParameter{
		{SourceIndex: &sourceIndex, Name: "identifier", Type: "long"},
		{Name: "enabled", Type: "boolean", Argument: "true"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(edit), `"changes"`) {
		t.Fatalf("workspace edit = %s, want a returned JDT LS edit", edit)
	}
	arguments, ok := session.refactorParameters["commandArguments"].([]any)
	if !ok || len(arguments) != 8 {
		t.Fatalf("command arguments = %#v, want eight refactoring arguments", session.refactorParameters["commandArguments"])
	}
	parameters, ok := arguments[5].([]any)
	if !ok || len(parameters) != 2 {
		t.Fatalf("mapped parameters = %#v, want two parameters", arguments[5])
	}
	first, _ := parameters[0].(map[string]any)
	second, _ := parameters[1].(map[string]any)
	if first["originalIndex"] != float64(0) || first["name"] != "identifier" || second["originalIndex"] != float64(1) || second["defaultValue"] != "true" {
		t.Fatalf("JDT LS parameter mapping = %#v", parameters)
	}
}

func (s *workspaceCodeActionSession) Request(_ context.Context, method string, params any) (json.RawMessage, error) {
	if method == "textDocument/formatting" {
		return json.RawMessage("[]"), nil
	}
	if method == "java/listOverridableMethods" {
		return json.RawMessage(`{"type":"Service","methods":[{"bindingKey":"Lexample/Contract;.run()V","name":"run","parameters":[],"unimplemented":true,"declaringClass":"example.Contract","declaringClassType":"interface"}]}`), nil
	}
	if method == "java/addOverridableMethods" {
		edit := map[string]any{"documentChanges": []any{
			map[string]any{"textDocument": map[string]any{"uri": s.selectedURI, "version": 1}, "edits": []any{
				map[string]any{"range": javaRange{Start: javaPosition{Line: 1, Character: 15}, End: javaPosition{Line: 1, Character: 15}}, "newText": " // selected"},
			}},
			map[string]any{"textDocument": map[string]any{"uri": s.callerURI, "version": 1}, "edits": []any{
				map[string]any{"range": javaRange{Start: javaPosition{Line: 1, Character: 14}, End: javaPosition{Line: 1, Character: 14}}, "newText": " // caller"},
			}},
		}}
		response, err := json.Marshal(edit)
		return response, err
	}
	if method != "textDocument/codeAction" {
		return json.RawMessage("null"), nil
	}
	return json.RawMessage("[]"), nil
}

func (*workspaceCodeActionSession) Notify(context.Context, string, any) error { return nil }
func (*workspaceCodeActionSession) WaitDiagnostics(context.Context, string, int) ([]backend.Diagnostic, error) {
	return nil, nil
}
func (*workspaceCodeActionSession) Close() error { return nil }

func TestApplyCodeActionStagesMultiFileWorkspaceEdit(t *testing.T) {
	root := backend.CanonicalWorkspaceRoot(t.TempDir())
	if err := os.WriteFile(filepath.Join(root, "pom.xml"), []byte("<project/>\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	selected := filepath.Join(root, "Service.java")
	caller := filepath.Join(root, "Caller.java")
	selectedSource := []byte("package example;\nclass Service {}\n")
	callerSource := []byte("package example;\nclass Caller {}\n")
	for path, source := range map[string][]byte{selected: selectedSource, caller: callerSource} {
		if err := os.WriteFile(path, source, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	session := &workspaceCodeActionSession{selectedURI: pathutil.FileURI(selected), callerURI: pathutil.FileURI(caller)}
	fingerprint, err := javaImportFingerprint(root, effectiveJavaConfig(backend.ProjectContext{RootDir: root, WorkspaceTrust: backend.NewWorkspaceTrust(root, true)}))
	if err != nil {
		t.Fatal(err)
	}
	service := &JavaBackend{
		root: root, fingerprint: fingerprint, session: session,
		documents: make(map[string]javaDocumentState),
	}
	digest := sha256.Sum256(selectedSource)
	project := backend.ProjectContext{RootDir: root, File: selected, Language: backend.LanguageJava, WorkspaceTrust: backend.NewWorkspaceTrust(root, true)}
	result, err := service.ApplyCodeAction(context.Background(), backend.CodeActionRequest{
		Operation: backend.OperationImplementContract, Project: project, File: selected,
		Kind: "source.overrideMethods", ExpectedRevision: hex.EncodeToString(digest[:]),
	})
	if err != nil {
		t.Fatal(err)
	}
	if result == nil || len(result.Files) != 2 {
		t.Fatalf("workspace edit result = %#v, want two affected files", result)
	}
	for path, marker := range map[string]string{selected: "// selected", caller: "// caller"} {
		contents, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(contents), marker) {
			t.Fatalf("published %s = %q, missing %q", path, contents, marker)
		}
	}
}

func TestApplyCodeActionMovesStaticJavaMemberThroughJDTLS(t *testing.T) {
	root := backend.CanonicalWorkspaceRoot(t.TempDir())
	if err := os.WriteFile(filepath.Join(root, "pom.xml"), []byte("<project/>\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	selected := filepath.Join(root, "Source.java")
	destination := filepath.Join(root, "Target.java")
	selectedSource := []byte("package example;\nclass Source { static int value() { return 1; } }\n")
	destinationSource := []byte("package example;\nclass Target {}\n")
	for path, source := range map[string][]byte{selected: selectedSource, destination: destinationSource} {
		if err := os.WriteFile(path, source, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	session := &moveStaticMemberTestSession{selectedURI: pathutil.FileURI(selected), callerURI: pathutil.FileURI(destination)}
	fingerprint, err := javaImportFingerprint(root, effectiveJavaConfig(backend.ProjectContext{RootDir: root, WorkspaceTrust: backend.NewWorkspaceTrust(root, true)}))
	if err != nil {
		t.Fatal(err)
	}
	service := &JavaBackend{root: root, fingerprint: fingerprint, session: session, documents: make(map[string]javaDocumentState)}
	project := backend.ProjectContext{RootDir: root, File: selected, Language: backend.LanguageJava, WorkspaceTrust: backend.NewWorkspaceTrust(root, true)}
	result, err := service.ApplyCodeAction(context.Background(), backend.CodeActionRequest{
		Operation: backend.OperationMoveConstruct, Project: project, File: selected, DestinationFile: destination,
		DestinationType: "example.Target", Range: backend.Range{Start: backend.Position{Line: 1, Character: 30}, End: backend.Position{Line: 1, Character: 35}},
		Kind: "moveStaticMember", Symbol: "Source.value",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result == nil || len(result.Files) != 2 || !strings.Contains(string(result.Files[1].After), "static int value()") {
		t.Fatalf("Java move result = %#v, want the source and destination edits", result)
	}
	if session.params["moveKind"] != "moveStaticMember" || session.params["destination"] != "example.Target" || session.params["updateReferences"] != true {
		t.Fatalf("JDT LS move parameters = %#v", session.params)
	}
}

type moveTopLevelTypeTestSession struct {
	workspaceCodeActionSession

	moveParams    map[string]any
	notifications []string
	movedURI      string
}

func (s *moveTopLevelTypeTestSession) Request(ctx context.Context, method string, params any) (json.RawMessage, error) {
	if method == "textDocument/references" {
		response, err := json.Marshal([]any{map[string]any{"uri": s.callerURI, "range": javaRange{}}})
		return response, err
	}
	if method != "java/move" {
		return s.workspaceCodeActionSession.Request(ctx, method, params)
	}
	encoded, err := json.Marshal(params)
	if err != nil {
		return nil, err
	}
	if decodeErr := json.Unmarshal(encoded, &s.moveParams); decodeErr != nil {
		return nil, decodeErr
	}
	edit := map[string]any{"documentChanges": []any{
		map[string]any{"textDocument": map[string]any{"uri": s.selectedURI, "version": 1}, "edits": []any{
			map[string]any{"range": javaRange{Start: javaPosition{Line: 0, Character: 8}, End: javaPosition{Line: 0, Character: 14}}, "newText": "oldpkg"},
		}},
		map[string]any{"kind": "rename", "oldUri": s.selectedURI, "newUri": s.movedURI},
	}}
	encodedEdit, err := json.Marshal(edit)
	if err != nil {
		return nil, err
	}
	response, err := json.Marshal(map[string]any{"edit": json.RawMessage(encodedEdit)})
	return response, err
}

func (s *moveTopLevelTypeTestSession) Notify(_ context.Context, method string, _ any) error {
	s.notifications = append(s.notifications, method)
	return nil
}

func TestApplyCodeActionMovesPublicTopLevelTypeAcrossPackages(t *testing.T) {
	root := backend.CanonicalWorkspaceRoot(t.TempDir())
	oldPackage := filepath.Join(root, "src", "main", "java", "oldpkg")
	newPackage := filepath.Join(root, "src", "main", "java", "newpkg")
	callerPackage := filepath.Join(root, "src", "main", "java", "caller")
	for _, directory := range []string{oldPackage, newPackage, callerPackage} {
		if err := os.MkdirAll(directory, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "pom.xml"), []byte("<project/>\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	selected := filepath.Join(oldPackage, "Widget.java")
	destination := filepath.Join(newPackage, "Target.java")
	caller := filepath.Join(callerPackage, "Caller.java")
	selectedSource := []byte("package oldpkg;\npublic class Widget {}\n")
	destinationSource := []byte("package newpkg;\nclass Target {}\n")
	callerSource := []byte("package caller;\nimport oldpkg.Widget;\nclass Caller { Widget widget; }\n")
	for path, source := range map[string][]byte{selected: selectedSource, destination: destinationSource, caller: callerSource} {
		if err := os.WriteFile(path, source, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	moved := filepath.Join(newPackage, "Widget.java")
	session := &moveTopLevelTypeTestSession{
		selectedURI: pathutil.FileURI(selected), callerURI: pathutil.FileURI(caller), movedURI: pathutil.FileURI(moved)}
	fingerprint, err := javaImportFingerprint(root, effectiveJavaConfig(backend.ProjectContext{RootDir: root, WorkspaceTrust: backend.NewWorkspaceTrust(root, true)}))
	if err != nil {
		t.Fatal(err)
	}
	service := &JavaBackend{root: root, fingerprint: fingerprint, session: session, documents: make(map[string]javaDocumentState)}
	project := backend.ProjectContext{RootDir: root, File: selected, Language: backend.LanguageJava, WorkspaceTrust: backend.NewWorkspaceTrust(root, true)}
	result, err := service.ApplyCodeAction(context.Background(), backend.CodeActionRequest{
		Operation: backend.OperationMoveConstruct, Project: project, File: selected, DestinationFile: destination,
		DestinationType: "newpkg.Target", DestinationPackage: "newpkg", Kind: "moveResource", Symbol: "Widget",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result == nil || len(result.Files) != 2 {
		t.Fatalf("Java type move result = %#v, want source and caller edits", result)
	}
	if _, statErr := os.Stat(selected); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("old source path still exists or could not be checked: %v", statErr)
	}
	movedSource, err := os.ReadFile(moved)
	if err != nil || !strings.Contains(string(movedSource), "package newpkg;") {
		t.Fatalf("moved type source = %q, error = %v", movedSource, err)
	}
	updatedCaller, err := os.ReadFile(caller)
	if err != nil || !strings.Contains(string(updatedCaller), "import newpkg.Widget;") {
		t.Fatalf("updated caller = %q, error = %v", updatedCaller, err)
	}
	moveDestination, ok := session.moveParams["destination"].(map[string]any)
	if session.moveParams["moveKind"] != "moveResource" || !ok || moveDestination["uri"] != pathutil.FileURI(newPackage) {
		t.Fatalf("JDT LS resource move parameters = %#v", session.moveParams)
	}
	renamed := false
	for _, method := range session.notifications {
		renamed = renamed || method == "workspace/didRenameFiles"
	}
	if !renamed {
		t.Fatalf("JDT LS notifications = %v, want workspace/didRenameFiles", session.notifications)
	}
}
