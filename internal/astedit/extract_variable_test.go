// Package astedit tests the supported expression extraction contract and fail-closed refusals.
package astedit_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"semedit/internal/astedit"
)

func TestExtractVariableRewritesWholeSingleCallArgument(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sample.go")
	before := "package sample\n\nfunc Print(int) {}\n\nfunc Render(count int) {\n\tPrint(count + 1)\n}\n"
	if err := os.WriteFile(path, []byte(before), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := astedit.ExtractVariable(context.Background(), path, "Render", "value", astedit.ExtractVariableOptions{Discriminator: "count + 1"}); err != nil {
		t.Fatalf("ExtractVariable() error = %v", err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	want := "package sample\n\nfunc Print(int) {}\n\nfunc Render(count int) {\n\tvalue := count + 1\n\tPrint(value)\n}\n"
	if string(got) != want {
		t.Fatalf("ExtractVariable() source mismatch\nwant:\n%s\ngot:\n%s", want, got)
	}
}

func TestExtractVariablePreservesLaterArgumentOrder(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sample.go")
	before := "package sample\n\nfunc Print(int, int) {}\n\nfunc Render(count int) {\n\tPrint(count + 1, count)\n}\n"
	if err := os.WriteFile(path, []byte(before), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := astedit.ExtractVariable(context.Background(), path, "Render", "value", astedit.ExtractVariableOptions{Discriminator: "count + 1"}); err != nil {
		t.Fatalf("ExtractVariable() error = %v", err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	want := "package sample\n\nfunc Print(int, int) {}\n\nfunc Render(count int) {\n\tvalue := count + 1\n\tPrint(value, count)\n}\n"
	if string(got) != want {
		t.Fatalf("ExtractVariable() source mismatch\nwant:\n%s\ngot:\n%s", want, got)
	}
}

func TestExtractVariableRequiresPathForAmbiguousExpression(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sample.go")
	before := "package sample\nfunc Print(int) {}\nfunc Render(count int) { Print(count); Print(count) }\n"
	if err := os.WriteFile(path, []byte(before), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := astedit.ExtractVariable(context.Background(), path, "Render", "value", astedit.ExtractVariableOptions{Discriminator: "count"})
	if err == nil || !strings.Contains(err.Error(), "construct_path") || !strings.Contains(err.Error(), "0, 1") {
		t.Fatalf("ambiguous selection error = %v", err)
	}
	got, readErr := os.ReadFile(path)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(got) != before {
		t.Fatalf("ambiguous selection changed source\nwant:\n%s\ngot:\n%s", before, got)
	}
}

func TestExtractVariableRejectsUnsafeAndUnrepresentableSelectionsWithoutWriting(t *testing.T) {
	cases := []struct {
		name, source, discriminator, wantError string
	}{
		{name: "nested call", source: "package sample\nfunc Print(int) {}\nfunc compute() int { return 1 }\nfunc Render(count int) { Print(count + compute()) }\n", discriminator: "count + compute()", wantError: "side-effect-free"},
		{name: "constant defaulting", source: "package sample\nfunc Print(int8) {}\nfunc Render(count int) { Print(1) }\n", discriminator: "1", wantError: "type may depend"},
		{name: "untyped shift left operand", source: "package sample\nfunc Print(uint64) {}\nfunc Render(power int) { Print(1 << power) }\n", discriminator: "1 << power", wantError: "type may depend"},
		{name: "selector label is not a typed input", source: "package sample\nimport \"example/constants\"\nfunc Print(any) {}\nfunc Render(p int) { Print(constants.p) }\n", discriminator: "constants.p", wantError: "type may depend"},
		{name: "comments", source: "package sample\nfunc Print(int) {}\nfunc Render(count int) { // keep\n Print(count)\n}\n", discriminator: "count", wantError: "comments inside"},
		{name: "non-first argument", source: "package sample\nfunc Print(int, int) {}\nfunc Render(count int) { Print(0, count) }\n", discriminator: "count", wantError: "no supported first argument"},
		{name: "import binding collision", source: "package sample\nimport \"gopkg.in/yaml.v3\"\nfunc Print(int) {}\nfunc Render(count int) { Print(count); _ = yaml.Unmarshal }\n", discriminator: "count", wantError: "local name \"yaml\" collides"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "sample.go")
			if err := os.WriteFile(path, []byte(tc.source), 0o600); err != nil {
				t.Fatal(err)
			}
			localName := "value"
			if tc.name == "import binding collision" {
				localName = "yaml"
			}
			_, err := astedit.ExtractVariable(context.Background(), path, "Render", localName, astedit.ExtractVariableOptions{Discriminator: tc.discriminator})
			if err == nil || !strings.Contains(err.Error(), tc.wantError) {
				t.Fatalf("ExtractVariable() error = %v, want substring %q", err, tc.wantError)
			}
			got, readErr := os.ReadFile(path)
			if readErr != nil {
				t.Fatal(readErr)
			}
			if string(got) != tc.source {
				t.Fatalf("refused extraction changed source\nwant:\n%s\ngot:\n%s", tc.source, got)
			}
		})
	}
}

func TestExtractVariableRejectsCallableParameterWithoutWriting(t *testing.T) {
	before := "package sample\nfunc Render(call func(int), value int) { call(value) }\n"
	path := filepath.Join(t.TempDir(), "sample.go")
	if err := os.WriteFile(path, []byte(before), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := astedit.ExtractVariable(context.Background(), path, "Render", "extracted", astedit.ExtractVariableOptions{Discriminator: "value"})
	if err == nil || !strings.Contains(err.Error(), "unshadowed same-file package function") {
		t.Fatalf("ExtractVariable() error = %v, want callable-parameter refusal", err)
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != before {
		t.Fatalf("rejected extraction changed source:\nbefore:\n%s\nafter:\n%s", before, after)
	}
}

func TestExtractVariableRejectsUnresolvedImportCalleeWithoutWriting(t *testing.T) {
	before := "package sample\nimport \"fmt\"\nfunc consume(int) {}\nfunc Render(value int) { consume(value); fmt.Println(value) }\n"
	path := filepath.Join(t.TempDir(), "sample.go")
	if err := os.WriteFile(path, []byte(before), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := astedit.ExtractVariable(context.Background(), path, "Render", "extracted", astedit.ExtractVariableOptions{Discriminator: "value"})
	if err == nil || !strings.Contains(err.Error(), "no unresolved import binding") {
		t.Fatalf("ExtractVariable() error = %v, want import-resolution refusal", err)
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != before {
		t.Fatalf("rejected extraction changed source:\nbefore:\n%s\nafter:\n%s", before, after)
	}
}
