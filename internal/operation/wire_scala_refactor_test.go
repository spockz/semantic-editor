// Package operation tests Scala callable selection boundaries against source text.
// WHY: A resolved Scala symbol range can stop before its full block, so callable selections must remain anchored to that method's parsed brace scope.
package operation

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spockz/semantic-editor/internal/backend"
)

func TestScalaCallableSelectionStaysWithinResolvedOwnerBody(t *testing.T) {
	source := []byte(`object A {
  def run(): Unit = /* {{ nested /* } */ */ {
    consume("{")
  }
}
object B {
  def run(): Unit = {
    consume("}")
  }
} // }
`)
	ownerStart := strings.Index(string(source), "def run")
	firstCall := strings.Index(string(source), `consume("{")`)
	secondCall := strings.Index(string(source), `consume("}")`)
	if ownerStart < 0 || firstCall < 0 || secondCall < 0 {
		t.Fatal("test fixture is malformed")
	}
	if !scalaSelectionWithinOwnerBody(source, ownerStart, len(source), firstCall, firstCall+len(`consume("{")`)) {
		t.Fatal("selection inside the resolved owner was rejected")
	}
	if scalaSelectionWithinOwnerBody(source, ownerStart, len(source), secondCall, secondCall+len(`consume("}")`)) {
		t.Fatal("selection in a later same-named method was accepted")
	}

	expressionOwner := []byte("object A { def run(): Unit = consume(0) }\nobject B { def run(): Unit = { consume(1) } }\n")
	expressionOwnerStart := strings.Index(string(expressionOwner), "def run")
	foreignCall := strings.LastIndex(string(expressionOwner), "consume(1)")
	if scalaSelectionWithinOwnerBody(expressionOwner, expressionOwnerStart, len(expressionOwner), foreignCall, foreignCall+len("consume(1)")) {
		t.Fatal("selection in a later method was accepted after an expression-bodied owner")
	}

	interpolatedOwner := []byte("object A { def run(): Unit = { val text = s\"${\"{\"}\"; consume(1) } }\n")
	interpolatedStart := strings.Index(string(interpolatedOwner), "def run")
	interpolatedCall := strings.Index(string(interpolatedOwner), "consume(1)")
	if scalaSelectionWithinOwnerBody(interpolatedOwner, interpolatedStart, len(interpolatedOwner), interpolatedCall, interpolatedCall+len("consume(1)")) {
		t.Fatal("owner with interpolation syntax was accepted by the bounded body scanner")
	}
	for _, prefix := range []string{"s", "custom", "custom$", "custom_+"} {
		owner := []byte("object A { def run(): Unit = { val text = " + prefix + `"${"{"}"; consume(1) } }` + "\n")
		methodStart := strings.Index(string(owner), "def run")
		call := strings.Index(string(owner), "consume(1)")
		if scalaSelectionWithinOwnerBody(owner, methodStart, len(owner), call, call+len("consume(1)")) {
			t.Fatalf("owner with %q interpolation syntax was accepted by the bounded body scanner", prefix)
		}
	}

	symbolOwner := []byte("object A { def run(): Unit = { val first = 'hello; consume(1) } }\nobject B { def run(): Unit = { consume(2); val second = 'world } }\n")
	symbolOwnerStart := strings.Index(string(symbolOwner), "def run")
	symbolCall := strings.Index(string(symbolOwner), "consume(1)")
	if scalaSelectionWithinOwnerBody(symbolOwner, symbolOwnerStart, len(symbolOwner), symbolCall, symbolCall+len("consume(1)")) {
		t.Fatal("owner with Symbol literals was accepted by the bounded body scanner")
	}
	backquotedOwner := []byte("object A { def run(): Unit = { val `x{` = 1; consume(1) } }\nobject B { def run(): Unit = { consume(2) } }\n")
	backquotedStart := strings.Index(string(backquotedOwner), "def run")
	backquotedCall := strings.Index(string(backquotedOwner), "consume(1)")
	if scalaSelectionWithinOwnerBody(backquotedOwner, backquotedStart, len(backquotedOwner), backquotedCall, backquotedCall+len("consume(1)")) {
		t.Fatal("owner with a brace in a backquoted identifier was accepted by the bounded body scanner")
	}
	xmlOwner := []byte("object A { def run(): Unit = { val ignored = <a><!-- { --></a>; consume(1) } }\nobject B { def run(): Unit = { consume(2) } }\n")
	xmlStart := strings.Index(string(xmlOwner), "def run")
	xmlCall := strings.Index(string(xmlOwner), "consume(1)")
	if scalaSelectionWithinOwnerBody(xmlOwner, xmlStart, len(xmlOwner), xmlCall, xmlCall+len("consume(1)")) {
		t.Fatal("owner with XML literal syntax was accepted by the bounded body scanner")
	}
	abstractOwner := []byte("trait A { def run(): Unit }\nobject B { def run(): Unit = { consume(1) } }\n")
	abstractStart := strings.Index(string(abstractOwner), "def run")
	laterOwnerStart := strings.Index(string(abstractOwner), "object B")
	abstractCall := strings.Index(string(abstractOwner), "consume(1)")
	if scalaSelectionWithinOwnerBody(abstractOwner, abstractStart, laterOwnerStart, abstractCall, abstractCall+len("consume(1)")) {
		t.Fatal("abstract owner borrowed the body of a later same-named method")
	}
}

func TestScalaSourceOffsetUsesSharedDecoder(t *testing.T) {
	source := []byte("first\r\nsecond")
	offset, err := scalaSourceOffset(source, backend.Position{Line: 1, Character: 0})
	if err != nil {
		t.Fatalf("map CRLF source position: %v", err)
	}
	if want := len("first\r\n"); offset != want {
		t.Fatalf("offset = %d, want %d", offset, want)
	}
	if _, err := scalaSourceOffset(source, backend.Position{Line: 0, Character: 6}); err == nil {
		t.Fatal("position inside the CRLF terminator was accepted")
	}
}

func TestScalaVariableExtractionRejectsStaleSelectionRevision(t *testing.T) {
	directory := t.TempDir()
	file := filepath.Join(directory, "service.scala")
	source := []byte("object Pipeline {\n  def update(): Int = calculate(3*60)\n}\n")
	if err := os.WriteFile(file, source, 0o600); err != nil {
		t.Fatalf("write Scala source: %v", err)
	}
	_, err := runExtractVariableScalaWithReceipt(context.Background(), CallContext{WorkDir: directory}, ExtractVariableReq{
		File: "service.scala", Symbol: "minutes", Kind: "variable", InFunction: "update",
		Discriminator: "3*60", ExpectedRevision: "stale-revision",
	})
	if err == nil || !strings.Contains(err.Error(), "source changed since selection") {
		t.Fatalf("stale extraction error = %v", err)
	}
}

type scalaLookupRaceBackend struct {
	file        string
	replacement []byte
	actionErr   error
	request     *backend.CodeActionRequest
}

func (*scalaLookupRaceBackend) Language() backend.LanguageID { return backend.LanguageScala }

func (*scalaLookupRaceBackend) Capabilities() backend.Capabilities {
	return backend.NewCapabilities(backend.OperationLookup, backend.OperationExtractFunction)
}

func (fake *scalaLookupRaceBackend) Lookup(_ context.Context, _ backend.ProjectContext, _ string) (*backend.LookupResult, error) {
	if err := os.WriteFile(fake.file, fake.replacement, 0o600); err != nil {
		return nil, err
	}
	return &backend.LookupResult{Definition: &backend.NamedTarget{
		QualifiedName: "A.emit", Kind: "method",
		DeclarationRange: &backend.Range{
			Start: backend.Position{Line: 1, Character: 2},
			End:   backend.Position{Line: 3, Character: 3},
		},
	}}, nil
}

func (*scalaLookupRaceBackend) Rename(context.Context, backend.RenameRequest) (*backend.RenameResult, error) {
	return nil, errors.New("unexpected Scala rename")
}

func (*scalaLookupRaceBackend) Verify(context.Context, backend.VerifyRequest) ([]backend.Diagnostic, error) {
	return nil, nil
}

func (fake *scalaLookupRaceBackend) ApplyCodeAction(_ context.Context, request backend.CodeActionRequest) (*backend.CodeActionResult, error) {
	fake.request = &request
	return nil, fake.actionErr
}

func TestScalaCallableLookupAndEditShareCapturedSource(t *testing.T) {
	directory := t.TempDir()
	file := filepath.Join(directory, "service.scala")
	source := []byte("object A {\n  def emit(): Unit = {\n    consume()\n  }\n}\n")
	replacement := []byte("object B {\n  def emit(): Unit = {\n    consume()\n  }\n}\n")
	if err := os.WriteFile(file, source, 0o600); err != nil {
		t.Fatalf("write Scala source: %v", err)
	}
	stop := errors.New("stop after action request")
	fake := &scalaLookupRaceBackend{file: file, replacement: replacement, actionErr: stop}
	registry, err := backend.NewRegistry(fake)
	if err != nil {
		t.Fatalf("register Scala test backend: %v", err)
	}
	cc := CallContext{
		WorkDir: directory,
		Project: backend.ProjectContext{RootDir: directory, Language: backend.LanguageScala},
		Service: &backend.Service{Registry: registry},
	}
	_, err = runExtractFunctionScalaWithReceipt(context.Background(), cc, ExtractFunctionReq{
		Project: cc.Project, File: "service.scala", Symbol: "extracted", InFunction: "A.emit",
		Kind: "function", StartLine: 3, EndLine: 3,
	})
	if !errors.Is(err, stop) {
		t.Fatalf("callable extraction error = %v, want action stop", err)
	}
	if fake.request == nil {
		t.Fatal("Scala code action was not called")
	}
	if got, want := fake.request.ExpectedRevision, structuralRevision(source); got != want {
		t.Fatalf("action revision = %q, want captured owner revision %q", got, want)
	}
	if fake.request.EnclosingSymbol != "A.emit" {
		t.Fatalf("action owner = %q, want A.emit", fake.request.EnclosingSymbol)
	}
	if got, err := os.ReadFile(file); err != nil || string(got) != string(replacement) {
		t.Fatalf("external source change was overwritten: got %q, err %v", got, err)
	}
}
