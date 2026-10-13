// WHY: Proves that Java parameter-object rewrites preserve bound identifiers and every resolved caller argument.
package javacst

import (
	"strings"
	"testing"

	"semedit/internal/backend"
)

func TestParseJavaMethodParametersAndRewriteBoundReferences(t *testing.T) {
	method := []byte(`void send(int id, boolean enabled) { audit(id, enabled); logger.log("id", id); }`)
	parameters, err := ParseJavaMethodParameters(method)
	if err != nil {
		t.Fatal(err)
	}
	if len(parameters) != 2 || parameters[0] != (JavaParameter{Name: "id", Type: "int"}) || parameters[1] != (JavaParameter{Name: "enabled", Type: "boolean"}) {
		t.Fatalf("parameters = %#v", parameters)
	}
	updated, err := RewriteJavaParameterReferences(method, "options", []JavaParameterField{
		{SourceIndex: 0, ParameterName: "id", Name: "id", Type: "int"},
		{SourceIndex: 1, ParameterName: "enabled", Name: "enabled", Type: "boolean"},
	})
	if err != nil {
		t.Fatal(err)
	}
	want := `void send(int id, boolean enabled) { audit(options.id(), options.enabled()); logger.log("id", options.id()); }`
	if string(updated) != want {
		t.Fatalf("updated method = %s, want %s", updated, want)
	}
}

func TestJavaParameterObjectCallerTransformsPreserveArguments(t *testing.T) {
	source := []byte(`class Calls { void run() { send(new SendOptions()); send(new SendOptions()); } }`)
	updated, err := ReplaceJavaMethodCallsWithParameterObject(source, "send", "SendOptions", [][]string{{"loadId()", "enabled()"}, {"2", "false"}})
	if err != nil {
		t.Fatal(err)
	}
	want := `class Calls { void run() { send(new SendOptions(loadId(), enabled())); send(new SendOptions(2, false)); } }`
	if string(updated) != want {
		t.Fatalf("updated source = %s, want %s", updated, want)
	}
	if _, err := ReplaceJavaMethodCallsWithParameterObject(source, "send", "SendOptions", [][]string{{"loadId()", "enabled()"}}); err == nil {
		t.Fatal("caller count mismatch succeeded")
	}
}

func TestJavaMethodCallArgumentsAtResolvedNameRange(t *testing.T) {
	source := []byte("class Calls { void run() { send(first(), 2); } }")
	start := strings.Index(string(source), "send")
	rangeValue := backend.Range{Start: backend.PositionFromByteOffset(source, start), End: backend.PositionFromByteOffset(source, start+len("send"))}
	arguments, err := JavaMethodCallArgumentsAt(source, rangeValue)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(arguments, ",") != "first(),2" {
		t.Fatalf("arguments = %#v", arguments)
	}
}

func TestJavaMethodCallArgumentsAtInvocationRange(t *testing.T) {
	source := []byte("class Calls { void run() { service.send(first(), 2); } }")
	start := strings.Index(string(source), "send")
	end := start + strings.Index(string(source[start:]), ";")
	rangeValue := backend.Range{Start: backend.PositionFromByteOffset(source, start), End: backend.PositionFromByteOffset(source, end)}
	arguments, err := JavaMethodCallArgumentsAt(source, rangeValue)
	if err != nil {
		t.Fatalf("resolve invocation range: %v", err)
	}
	if strings.Join(arguments, ",") != "first(),2" {
		t.Fatalf("arguments = %#v", arguments)
	}
}

func TestJavaParameterObjectRejectsUnmodeledSyntaxAndCallComments(t *testing.T) {
	if _, err := ParseJavaMethodParameters([]byte("void send(@NonNull String value) {}")); err == nil {
		t.Fatal("annotated parameter type was accepted without preserving its annotation")
	}
	source := []byte("class Calls { void run() { send(first() /* keep */, 2); } }")
	start := strings.Index(string(source), "send")
	rangeValue := backend.Range{Start: backend.PositionFromByteOffset(source, start), End: backend.PositionFromByteOffset(source, start+len("send"))}
	if _, err := JavaMethodCallArgumentsAt(source, rangeValue); err == nil {
		t.Fatal("commented call arguments were accepted for rewriting")
	}
}

func TestInsertJavaParameterObjectAddsPackagePrivateType(t *testing.T) {
	source := []byte("package example;\nclass Service {}\n")
	updated, err := InsertJavaParameterObject(source, "SendOptions", []JavaParameterField{
		{Name: "id", Type: "int"}, {Name: "enabled", Type: "boolean"},
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"final class SendOptions", "private final int id;", "private final boolean enabled;", "this.id = id;", "this.enabled = enabled;", "int id()", "boolean enabled()"} {
		if !strings.Contains(string(updated), want) {
			t.Errorf("updated type is missing %q:\n%s", want, updated)
		}
	}
	if _, err := InsertJavaParameterObject(updated, "SendOptions", []JavaParameterField{{Name: "id", Type: "int"}}); err == nil {
		t.Fatal("duplicate type name succeeded")
	}
}
