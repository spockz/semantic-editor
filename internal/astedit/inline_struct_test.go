// WHY: Exercises resolved struct-inline identity and byte-preserving refusal through the public Go edit entry point.

package astedit

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInlineStructVariable(t *testing.T) {
	t.Parallel()
	cases := []struct{ name, fields, body, want string }{
		{"same spelling field", "value int; Ready bool; Label string", "value := Result{value: 42, Ready: true, Label: \"ready\"}\nreturn value", "return Result{value: 42, Ready: true, Label: \"ready\"}"},
		{"omitted basic field", "Count int; Ready bool", "value := Result{Count: 7}\nreturn value", "return Result{Count: 7}"},
		{"false", "Ready bool", "value := Result{Ready: false}\nreturn value", "return Result{Ready: false}"},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			directory := t.TempDir()
			file := filepath.Join(directory, "value.go")
			before := "package sample\ntype Result struct {" + testCase.fields + "}\nfunc Build() any {\n" + testCase.body + "\n}\n"
			if err := os.WriteFile(file, []byte(before), 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := InlineVariable(context.Background(), file, "Build", "value"); err != nil {
				t.Fatal(err)
			}
			actual, err := os.ReadFile(file)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(actual), testCase.want) || strings.Contains(string(actual), "value :=") {
				t.Fatalf("unexpected source:\n%s", actual)
			}
		})
	}
}

func TestInlineStructVariableRefusesWithoutWriting(t *testing.T) {
	t.Parallel()
	cases := []struct{ name, declarations, result, body string }{
		{"call value", "", "any", "value := Result{Count: call()}\nreturn value"},
		{"variable read", "", "any", "count := 1\nvalue := Result{Count: count}\nreturn value"},
		{"conversion", "", "any", "value := Result{Count: int(1)}\nreturn value"},
		{"boolean shadow", "", "any", "true := false\nvalue := Result{Ready: true}\nreturn value"},
		{"parenthesized field", "", "any", "value := Result{Count: (1)}\nreturn value"},
		{"reference field omitted", "type Result struct { Count int; Ref *int }", "any", "value := Result{Count: 1}\nreturn value"},
		{"nested field omitted", "type Result struct { Count int; Nested struct{Flag bool} }", "any", "value := Result{Count: 1}\nreturn value"},
		{"defined scalar field", "type Count int\ntype Result struct { Count Count }", "any", "value := Result{Count: 1}\nreturn value"},
		{"generic", "type Result[T any] struct { Count int }", "any", "value := Result[int]{Count: 1}\nreturn value"},
		{"positional", "", "any", "value := Result{1, true}\nreturn value"},
		{"explicit", "", "any", "var value Result = Result{Count: 1}\nreturn value"},
		{"inferred var", "", "any", "var value = Result{Count: 1}\nreturn value"},
		{"extra reference", "", "any", "value := Result{Count: 1}\nprintln(value.Count)\nreturn value"},
		{"closure reference", "", "any", "value := Result{Count: 1}\nreturn func() any {return value}"},
		{"indirect use", "", "any", "value := Result{Count: 1}\nreturn &value"},
		{"later return", "", "any", "value := Result{Count: 1}\nprintln(1)\nreturn value"},
		{"control flow", "", "any", "value := Result{Count: 1}\nif true {return value}\nreturn nil"},
		{"named result", "", "(result any)", "value := Result{Count: 1}\nreturn value"},
		{"multiple results", "", "(any, int)", "value := Result{Count: 1}\nreturn value, 1"},
		{"literal comment", "", "any", "value := Result{Count: /* keep */ 1}\nreturn value"},
		{"adjacent declaration comment", "", "any", "// keep\nvalue := Result{Count: 1}\nreturn value"},
		{"return comment", "", "any", "value := Result{Count: 1}\nreturn value // keep"},
		{"directive", "", "any", "value := Result{Count: 1}\n//line other.go:100\nreturn value"},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			declarations := testCase.declarations
			if declarations == "" {
				declarations = "type Result struct {Count int; Ready bool}"
			}
			before := "package sample\n" + declarations + "\nfunc call() int {return 1}\nfunc Build() " + testCase.result + " {\n" + testCase.body + "\n}\n"
			file := filepath.Join(t.TempDir(), "value.go")
			if err := os.WriteFile(file, []byte(before), 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := InlineVariable(context.Background(), file, "Build", "value"); err == nil {
				t.Fatal("expected refusal")
			}
			actual, err := os.ReadFile(file)
			if err != nil {
				t.Fatal(err)
			}
			if string(actual) != before {
				t.Fatalf("refusal changed source:\n%s", actual)
			}
		})
	}
}
