// Package astedit contains bounded source transformations that can prove local semantic safety.
// InlineVariable tests exercise the literal-only inlining contract and its refusal cases.
package astedit

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInlineVariableLiteralImmediateCall(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "inline.go")
	before := "package p\nfunc f() int {\n\tvalue := 42\n\tprintln(value, value)\n\treturn 0\n}\n"
	if err := os.WriteFile(path, []byte(before), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := InlineVariable(context.Background(), path, "f", "value"); err != nil {
		t.Fatalf("InlineVariable: %v", err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	want := "package p\n\nfunc f() int {\n\tprintln(int(42), int(42))\n\treturn 0\n}\n"
	if string(got) != want {
		t.Fatalf("source mismatch\nwant:\n%s\ngot:\n%s", want, got)
	}
}

func TestInlineVariableRejectsUnsafeCasesWithoutWriting(t *testing.T) {
	t.Parallel()
	cases := map[string]string{
		"side effect":           "value := call()\nprintln(value)\n",
		"control-flow boundary": "value := 42\nif true { println(1) }\nprintln(value)\n",
		"reassignment":          "value := 42\nvalue = 7\nprintln(value)\n",
		"shadowing":             "value := 42\n{ value := 7; println(value) }\nprintln(value)\n",
		"unsupported use":       "value := 42\nprintln(value + 1)\n",
		"explicit type":         "var value int64 = 42\nprintln(value)\n",
		"attached comment":      "// keep this\nvalue := 42\nprintln(value)\n",
		"shadowed conversion":   "int := 7\nvalue := 42\nprintln(value)\n",
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "inline.go")
			before := "package p\nfunc f() {\n" + indentSource(body) + "}\n"
			if err := os.WriteFile(path, []byte(before), 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := InlineVariable(context.Background(), path, "f", "value"); err == nil {
				t.Fatal("expected refusal")
			}
			got, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != before {
				t.Fatalf("refusal changed source\nwant:\n%s\ngot:\n%s", before, got)
			}
		})
	}
}

func TestInlineVariableRejectsLocalConversionTypeShadowWithoutWriting(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "inline.go")
	before := "package p\nfunc f() {\n\ttype int = string\n\tvalue := 42\n\tprintln(value)\n}\n"
	if err := os.WriteFile(path, []byte(before), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := InlineVariable(context.Background(), path, "f", "value")
	if err == nil || !strings.Contains(err.Error(), "default type name int is shadowed") {
		t.Fatalf("expected local conversion type shadow refusal, got %v", err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != before {
		t.Fatalf("refusal changed source\nwant:\n%s\ngot:\n%s", before, got)
	}
}

func TestInlineVariableRejectsUncertainConversionBindings(t *testing.T) {
	t.Parallel()
	cases := map[string]struct {
		file    string
		sibling string
	}{
		"sibling package type": {
			file:    "package p\nfunc f() {\n\tvalue := 42\n\tprintln(value)\n}\n",
			sibling: "package p\ntype int int\n",
		},
		"unaliased import": {
			file: "package p\nimport \"example.com/elsewhere/custom\"\nfunc f() {\n\tvalue := 42\n\tprintln(value)\n}\n",
		},
		"malformed sibling": {
			file:    "package p\nfunc f() {\n\tvalue := 42\n\tprintln(value)\n}\n",
			sibling: "package p\nfunc broken( {\n",
		},
	}
	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "inline.go")
			if err := os.WriteFile(path, []byte(testCase.file), 0o600); err != nil {
				t.Fatal(err)
			}
			if testCase.sibling != "" {
				if err := os.WriteFile(filepath.Join(dir, "sibling.go"), []byte(testCase.sibling), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := InlineVariable(context.Background(), path, "f", "value"); err == nil {
				t.Fatal("expected refusal")
			}
			got, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != testCase.file {
				t.Fatalf("refusal changed source\nwant:\n%s\ngot:\n%s", testCase.file, got)
			}
		})
	}
}

func indentSource(source string) string {
	lines := strings.Split(source, "\n")
	for i, line := range lines {
		if line != "" {
			lines[i] = "\t" + line
		}
	}
	return strings.Join(lines, "\n")
}

func TestInlineVariableRejectsParameterShadowing(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "inline.go")
	before := "package p\nfunc f(value int) {\n\t{\n\t\tvalue := 42\n\t\tprintln(value)\n\t}\n}\n"
	if err := os.WriteFile(path, []byte(before), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := InlineVariable(context.Background(), path, "f", "value"); err == nil {
		t.Fatal("expected refusal")
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != before {
		t.Fatalf("refusal changed source\nwant:\n%s\ngot:\n%s", before, got)
	}
}

func TestInlineVariableLaterCallsAndReturnInSameBlock(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "inline.go")
	before := "package p\nfunc f() int {\n\tvalue := 42\n\tprintln(1)\n\tprintln(value)\n\treturn value\n}\n"
	if err := os.WriteFile(path, []byte(before), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := InlineVariable(context.Background(), path, "f", "value"); err != nil {
		t.Fatalf("InlineVariable: %v", err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	want := "package p\n\nfunc f() int {\n\tprintln(1)\n\tprintln(int(42))\n\treturn int(42)\n}\n"
	if string(got) != want {
		t.Fatalf("source mismatch\nwant:\n%s\ngot:\n%s", want, got)
	}
}

func TestInlineVariablePreservesUnrelatedFollowingComment(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "inline.go")
	before := "package p\nfunc f() {\n\tvalue := 42\n\tprintln(value)\n\t// unrelated comment\n\tprintln(\"done\")\n}\n"
	if err := os.WriteFile(path, []byte(before), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := InlineVariable(context.Background(), path, "f", "value"); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	want := "package p\n\nfunc f() {\n\tprintln(int(42))\n\t// unrelated comment\n\tprintln(\"done\")\n}\n"
	if string(got) != want {
		t.Fatalf("inline changed unexpected source\nwant:\n%s\ngot:\n%s", want, got)
	}
}
