// WHY: Native explicit-type edits need precise, unique method-local tokens without accepting unsupported Java binding forms.
package javacst

import (
	"strings"
	"testing"

	"github.com/spockz/semantic-editor/internal/backend"
)

func TestJavaInferredLocalSelection(t *testing.T) {
	for _, test := range []struct {
		name, body string
		wantErr    bool
	}{
		{"simple", "var total = 1; return total;", false},
		{"nested block", "{ var total = 1; consume(total); }", false},
		{"explicit", "int total = 1;", true},
		{"duplicate", "{ var total = 1; } { var total = 2; }", true},
		{"annotated", "@Deprecated var total = 1;", true},
		{"final", "final var total = 1;", true},
		{"for binding", "for (var total = 0; total < 2; total++) {}", true},
		{"foreach", "for (var total : values) {}", true},
		{"while body", "while (true) { var total = 1; break; }", true},
		{"do body", "do { var total = 1; } while (false);", true},
		{"nested while body", "while (true) { { var total = 1; } break; }", true},
		{"nested do body", "do { { var total = 1; } } while (false);", true},
		{"lambda", "Runnable run = () -> { var total = 1; };", true},
		{"resource", "try (var total = open()) {}", true},
	} {
		t.Run(test.name, func(t *testing.T) {
			source := []byte("class Service { int count() { " + test.body + " } }")
			start := strings.Index(string(source), "total")
			selection := backend.Range{Start: backend.PositionFromByteOffset(source, start), End: backend.PositionFromByteOffset(source, start+5)}
			local, err := JavaInferredLocalAt(source, "total", selection)
			if test.wantErr {
				if err == nil {
					t.Fatalf("accepted %#v", local)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if string(source[local.Type.StartByte:local.Type.EndByte]) != "var" || local.Identifier != selection {
				t.Fatalf("local = %#v", local)
			}
		})
	}
}

func TestJavaExplicitTypeDenotability(t *testing.T) {
	for _, value := range []string{"int", "boolean", "String", "ArrayList<String>", "java.util.List<? extends Number>", "String[]", "T", "variety[]", "List<Variable>", "List<Var>"} {
		t.Run(value, func(t *testing.T) {
			if err := ValidateJavaExplicitType(value); err != nil {
				t.Fatal(err)
			}
		})
	}
	for _, value := range []string{"", "var", "var[]", "var[][]", "List<var>", "List<var[]>", "List<? extends var>", "java.util.var", "Map<String, List<var>>", "void", "Object & Runnable", "Runnable() {}", "@NonNull String", "int x; String", "int /* type */", "String\n", "<anonymous>"} {
		t.Run(value, func(t *testing.T) {
			if err := ValidateJavaExplicitType(value); err == nil {
				t.Fatalf("accepted %q", value)
			}
		})
	}
}
