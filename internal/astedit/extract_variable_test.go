// Package astedit tests the supported expression extraction contract and fail-closed refusals.
package astedit_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spockz/semantic-editor/internal/astedit"
)

func TestExtractVariableRewritesWholeSingleCallArgument(t *testing.T) {
	t.Parallel()
	path := filepath.Join(extractVariableTestDir(t), "sample.go")
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

func TestExtractVariableRewritesEveryStableRepeatedCallArgument(t *testing.T) {
	t.Parallel()
	runVariableExtractionCase(t, `package sample

type Count int

func Print(Count, int, Count, int, Count) {}

func Render(count Count) {
	Print(count + 1, 0, count + 1, 2, count + 1)
}
`, "count + 1", `package sample

type Count int

func Print(Count, int, Count, int, Count) {}

func Render(count Count) {
	adjusted := count + 1
	Print(adjusted, 0, adjusted, 2, adjusted)
}
`)
}

func TestExtractVariableRewritesDirectReturnExpression(t *testing.T) {
	t.Parallel()
	runVariableExtractionCase(t, `package sample

func Render(count int) int {
	return count + 1
}
`, "count + 1", `package sample

func Render(count int) int {
	adjusted := count + 1
	return adjusted
}
`)
}

func TestExtractVariableRewritesNestedReturnExpressionInPlace(t *testing.T) {
	t.Parallel()
	runVariableExtractionCase(t, `package sample

func Render(count int) int {
	if count > 0 {
		return count + 1
	}
	return count
}
`, "count + 1", `package sample

func Render(count int) int {
	if count > 0 {
		adjusted := count + 1
		return adjusted
	}
	return count
}
`)
}

func TestExtractVariableSelectsNestedReturnByInspectedPath(t *testing.T) {
	t.Parallel()
	path := filepath.Join(extractVariableTestDir(t), "sample.go")
	before := "package sample\n\nfunc Render(count int) int {\n\tif count > 0 {\n\t\treturn count + 1\n\t}\n\treturn count + 1\n}\n"
	if err := os.WriteFile(path, []byte(before), 0o600); err != nil {
		t.Fatal(err)
	}
	revisionSum := sha256.Sum256([]byte(before))
	revision := hex.EncodeToString(revisionSum[:])
	_, err := astedit.ExtractVariable(context.Background(), path, "Render", "adjusted", astedit.ExtractVariableOptions{
		Discriminator: "count + 1", ConstructPath: "0.0.0", ExpectedRevision: revision,
	})
	if err != nil {
		t.Fatalf("ExtractVariable() with nested construct path error = %v", err)
	}
	got, readErr := os.ReadFile(path)
	if readErr != nil {
		t.Fatal(readErr)
	}
	want := "package sample\n\nfunc Render(count int) int {\n\tif count > 0 {\n\t\tadjusted := count + 1\n\t\treturn adjusted\n\t}\n\treturn count + 1\n}\n"
	if string(got) != want {
		t.Fatalf("nested path selection source mismatch\nwant:\n%s\ngot:\n%s", want, got)
	}
}

func TestExtractVariablePreservesLaterArgumentOrder(t *testing.T) {
	t.Parallel()
	path := filepath.Join(extractVariableTestDir(t), "sample.go")
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

func TestExtractVariableExtractsNonFirstArgumentWithConstantSiblings(t *testing.T) {
	t.Parallel()
	path := filepath.Join(extractVariableTestDir(t), "sample.go")
	before := "package sample\n\nfunc Print(int, int, string) {}\n\nfunc Render(count int) {\n\tPrint(0, count + 1, \"label\")\n}\n"
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
	want := "package sample\n\nfunc Print(int, int, string) {}\n\nfunc Render(count int) {\n\tvalue := count + 1\n\tPrint(0, value, \"label\")\n}\n"
	if string(got) != want {
		t.Fatalf("ExtractVariable() source mismatch\nwant:\n%s\ngot:\n%s", want, got)
	}
}

func TestExtractVariableAcceptsPredeclaredNilAsUnchangedSibling(t *testing.T) {
	t.Parallel()
	path := filepath.Join(extractVariableTestDir(t), "sample.go")
	before := "package sample\n\nfunc Print(*int, int) {}\n\nfunc Render(count int) {\n\tPrint(nil, count + 1)\n}\n"
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
	want := "package sample\n\nfunc Print(*int, int) {}\n\nfunc Render(count int) {\n\tvalue := count + 1\n\tPrint(nil, value)\n}\n"
	if string(got) != want {
		t.Fatalf("ExtractVariable() source mismatch\nwant:\n%s\ngot:\n%s", want, got)
	}
}

func TestExtractVariablePreservesSelectedBoundsCheckWithConstantSiblings(t *testing.T) {
	t.Parallel()
	path := filepath.Join(extractVariableTestDir(t), "sample.go")
	before := "package sample\n\nfunc Print(int, int, int) {}\n\nfunc Render(values []int, index int) {\n\tPrint(0, values[index], 1)\n}\n"
	if err := os.WriteFile(path, []byte(before), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := astedit.ExtractVariable(context.Background(), path, "Render", "value", astedit.ExtractVariableOptions{Discriminator: "values[index]"}); err != nil {
		t.Fatalf("ExtractVariable() error = %v", err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	want := "package sample\n\nfunc Print(int, int, int) {}\n\nfunc Render(values []int, index int) {\n\tvalue := values[index]\n\tPrint(0, value, 1)\n}\n"
	if string(got) != want {
		t.Fatalf("ExtractVariable() source mismatch\nwant:\n%s\ngot:\n%s", want, got)
	}
}

func TestExtractVariableRequiresPathForAmbiguousExpression(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
	cases := []struct {
		name, source, discriminator, wantError string
	}{
		{name: "nested call", source: "package sample\nfunc Print(int) {}\nfunc compute() int { return 1 }\nfunc Render(count int) { Print(count + compute()) }\n", discriminator: "count + compute()", wantError: "side-effect-free"},
		{name: "constant defaulting", source: "package sample\nfunc Print(int8) {}\nfunc Render(count int) { Print(1) }\n", discriminator: "1", wantError: "type may depend"},
		{name: "untyped shift left operand", source: "package sample\nfunc Print(uint64) {}\nfunc Render(power int) { Print(1 << power) }\n", discriminator: "1 << power", wantError: "type may depend"},
		{name: "selector label is not a typed input", source: "package sample\nimport \"example/constants\"\nfunc Print(any) {}\nfunc Render(p int) { Print(constants.p) }\n", discriminator: "constants.p", wantError: "type may depend"},
		{name: "comments", source: "package sample\nfunc Print(int) {}\nfunc Render(count int) { // keep\n Print(count)\n}\n", discriminator: "count", wantError: "comments inside"},
		{name: "first argument before later side effect", source: "package sample\nfunc Print(int, int) {}\nfunc mutate(input *int) int { *input = *input + 1; return 0 }\nfunc Render(input int) { Print(input, mutate(&input)) }\n", discriminator: "input", wantError: "sibling call arguments"},
		{name: "non-first argument after earlier side effect", source: "package sample\nfunc Print(int, int, int) {}\nfunc mutate(count *int) int { *count = *count + 1; return 0 }\nfunc Render(count int) { Print(mutate(&count), count + 1, 0) }\n", discriminator: "count + 1", wantError: "sibling call arguments"},
		{name: "non-first argument with panic-prone sibling", source: "package sample\nfunc Print(int, int) {}\nfunc Render(values []int, index, count int) { Print(values[index], count + 1) }\n", discriminator: "count + 1", wantError: "sibling call arguments"},
		{name: "duplicate mutable indexed arguments", source: "package sample\nfunc Print(int, int) {}\nfunc Render(values []int, index int) { Print(values[index], values[index]) }\n", discriminator: "values[index]", wantError: "stable scalar expression"},
		{name: "duplicate address-escaped input", source: "package sample\nfunc Print(int, int) {}\nfunc keep(*int) {}\nfunc Render(count int) { keep(&count); Print(count + 1, count + 1) }\n", discriminator: "count + 1", wantError: "escapes through its address"},
		{name: "duplicate implicit pointer receiver call", source: "package sample\ntype Count int\nvar saved *Count\nfunc (count *Count) Escape() { saved = count }\nfunc Print(Count, Count) {}\nfunc Render(count Count) { count.Escape(); Print(count + 1, count + 1) }\n", discriminator: "count + 1", wantError: "escapes through its address"},
		{name: "duplicate implicit pointer receiver method value", source: "package sample\ntype Count int\nvar saved func()\nfunc (count *Count) Escape() {}\nfunc Print(Count, Count) {}\nfunc Render(count Count) { saved = count.Escape; Print(count + 1, count + 1) }\n", discriminator: "count + 1", wantError: "escapes through its address"},
		{name: "duplicate closure-captured input", source: "package sample\nfunc Print(int, int) {}\nfunc Render(count int) { defer func() { _ = count }(); Print(count + 1, count + 1) }\n", discriminator: "count + 1", wantError: "escapes through its address or a nested function"},
		{name: "variadic callee", source: "package sample\nfunc Print(...int) {}\nfunc Render(count int) { Print(count) }\n", discriminator: "count", wantError: "variadic callees"},
		{name: "variadic expansion", source: "package sample\nfunc Print(values ...int) {}\nfunc Render(values []int) { Print(values...) }\n", discriminator: "values", wantError: "variadic expansion"},
		{name: "import binding collision", source: "package sample\nimport \"gopkg.in/yaml.v3\"\nfunc Print(int) {}\nfunc Render(count int) { Print(count); _ = yaml.Unmarshal }\n", discriminator: "count", wantError: "local name \"yaml\" collides"},
		{name: "multiple short declaration targets", source: "package sample\nfunc Render(count int) { total, other := count + 1, count; _, _ = total, other }\n", discriminator: "count + 1", wantError: "one target and one value"},
		{name: "compound assignment", source: "package sample\nfunc Render(count int) { var total int; total += count + 1 }\n", discriminator: "count + 1", wantError: "only simple assignments"},
		{name: "multiple assignment targets", source: "package sample\nfunc Render(count int) { var total, other int; total, other = count + 1, 0 }\n", discriminator: "count + 1", wantError: "one target and one value"},
		{name: "non-identifier assignment target", source: "package sample\nfunc Render(count int) { values := make([]int, 1); values[0] = count + 1 }\n", discriminator: "count + 1", wantError: "named identifier"},
		{name: "effectful assignment RHS", source: "package sample\nfunc compute() int { return 1 }\nfunc Render(count int) { var total int; total = count + compute() }\n", discriminator: "count + compute()", wantError: "side-effect-free"},
		{name: "untyped-only assignment RHS", source: "package sample\nfunc Render() { var total int; total = 1 + 2; _ = total }\n", discriminator: "1 + 2", wantError: "typed Go input"},
		{name: "multiple results", source: "package sample\nfunc Render(count int) (int, int) { return count + 1, count + 2 }\n", discriminator: "count + 1", wantError: "one whole return expression"},
		{name: "multi-result call", source: "package sample\nfunc pair(value int) (int, int) { return value, value }\nfunc Render(value int) (int, int) { return pair(value) }\n", discriminator: "pair(value)", wantError: "exactly one named or unnamed owner result"},
		{name: "blank result", source: "package sample\nfunc Render(count int) (_ int) { return count + 1 }\n", discriminator: "count + 1", wantError: "exactly one named or unnamed owner result"},
		{name: "multiple names in one result field", source: "package sample\nfunc pair(value int) (int, int) { return value, value }\nfunc Render(value int) (first, second int) { return pair(value) }\n", discriminator: "pair(value)", wantError: "exactly one named or unnamed owner result"},
		{name: "untyped-only return expression", source: "package sample\nfunc Render() int { return 1 + 2 }\n", discriminator: "1 + 2", wantError: "return expression must contain a resolved typed Go input"},
		{name: "closure return", source: "package sample\nfunc Render(count int) int { compute := func() int { return count + 1 }; return compute() }\n", discriminator: "count + 1", wantError: "no supported call argument, assignment RHS, return expression, or complete if condition"},
		{name: "closure assignment", source: "package sample\nfunc Render(count int) { var total int; run := func() { total = count + 1 }; run() }\n", discriminator: "count + 1", wantError: "no supported call argument, assignment RHS, return expression, or complete if condition"},
		{name: "nested compound assignment", source: "package sample\nfunc Render(count int) { var total int; if count > 0 { total += count + 1 } }\n", discriminator: "count + 1", wantError: "only simple assignments"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(extractVariableTestDir(t), "sample.go")
			if err := os.WriteFile(path, []byte(tc.source), 0o600); err != nil {
				t.Fatal(err)
			}
			localName := "adjusted"
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
	t.Parallel()
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
	t.Parallel()
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

func TestExtractVariableRewritesDirectAssignmentRHS(t *testing.T) {
	t.Parallel()
	path := filepath.Join(extractVariableTestDir(t), "sample.go")
	before := "package sample\n\nfunc Render(count int) {\n\tstep := count\n\tvar total int\n\ttotal = step + 1\n\t_ = total\n}\n"
	if err := os.WriteFile(path, []byte(before), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := astedit.ExtractVariable(context.Background(), path, "Render", "adjusted", astedit.ExtractVariableOptions{Discriminator: "step + 1"}); err != nil {
		t.Fatalf("ExtractVariable() error = %v", err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	want := "package sample\n\nfunc Render(count int) {\n\tstep := count\n\tvar total int\n\tadjusted := step + 1\n\ttotal = adjusted\n\t_ = total\n}\n"
	if string(got) != want {
		t.Fatalf("ExtractVariable() source mismatch\nwant:\n%s\ngot:\n%s", want, got)
	}
}

func TestExtractVariableRewritesShortDeclarationRHS(t *testing.T) {
	t.Parallel()
	path := filepath.Join(extractVariableTestDir(t), "sample.go")
	before := "package sample\n\nfunc Render(base int) int {\n\tvalue := base + 1\n\treturn value\n}\n"
	if err := os.WriteFile(path, []byte(before), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := astedit.ExtractVariable(context.Background(), path, "Render", "adjusted", astedit.ExtractVariableOptions{Discriminator: "base + 1"}); err != nil {
		t.Fatalf("ExtractVariable() error = %v", err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	want := "package sample\n\nfunc Render(base int) int {\n\tadjusted := base + 1\n\tvalue := adjusted\n\treturn value\n}\n"
	if string(got) != want {
		t.Fatalf("ExtractVariable() source mismatch\nwant:\n%s\ngot:\n%s", want, got)
	}
}

func TestExtractVariableRewritesNestedAssignmentRHSInPlace(t *testing.T) {
	t.Parallel()
	runVariableExtractionCase(t, `package sample

func Render(count int) int {
	total := 0
	if count > 0 {
		total = count + 1
	}
	return total
}
`, "count + 1", `package sample

func Render(count int) int {
	total := 0
	if count > 0 {
		adjusted := count + 1
		total = adjusted
	}
	return total
}
`)
}

func TestExtractVariableRewritesLoopAssignmentRHSInPlace(t *testing.T) {
	t.Parallel()
	runVariableExtractionCase(t, `package sample

func Render(count int) int {
	total := 0
	for index := 0; index < count; index++ {
		total = total + index
	}
	return total
}
`, "total + index", `package sample

func Render(count int) int {
	total := 0
	for index := 0; index < count; index++ {
		adjusted := total + index
		total = adjusted
	}
	return total
}
`)
}

func TestExtractVariableSelectsNestedAssignmentByInspectedPath(t *testing.T) {
	t.Parallel()
	path := filepath.Join(extractVariableTestDir(t), "sample.go")
	before := "package sample\n\nfunc Render(count int) int {\n\ttotal := 0\n\tif count > 0 {\n\t\ttotal = count + 1\n\t} else {\n\t\ttotal = count + 1\n\t}\n\treturn total\n}\n"
	if err := os.WriteFile(path, []byte(before), 0o600); err != nil {
		t.Fatal(err)
	}
	revisionSum := sha256.Sum256([]byte(before))
	revision := hex.EncodeToString(revisionSum[:])
	_, err := astedit.ExtractVariable(context.Background(), path, "Render", "adjusted", astedit.ExtractVariableOptions{
		Discriminator: "count + 1", ConstructPath: "1.0.0", ExpectedRevision: revision,
	})
	if err != nil {
		t.Fatalf("ExtractVariable() error = %v", err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	want := "package sample\n\nfunc Render(count int) int {\n\ttotal := 0\n\tif count > 0 {\n\t\tadjusted := count + 1\n\t\ttotal = adjusted\n\t} else {\n\t\ttotal = count + 1\n\t}\n\treturn total\n}\n"
	if string(got) != want {
		t.Fatalf("ExtractVariable() source mismatch\nwant:\n%s\ngot:\n%s", want, got)
	}
}

func TestExtractVariableAcceptsTypedInputBeforeUntypedConstant(t *testing.T) {
	runVariableExtractionCase(t, `package sample

const Extra = 1

func Render(count int) {
	var total int
	total = count + Extra
	_ = total
}
`, "count + Extra", `package sample

const Extra = 1

func Render(count int) {
	var total int
	adjusted := count + Extra
	total = adjusted
	_ = total
}
`)
}

func TestExtractVariableAcceptsUntypedConstantBeforeTypedInput(t *testing.T) {
	runVariableExtractionCase(t, `package sample

const Extra = 1

func Render(count int) {
	var total int
	total = Extra + count
	_ = total
}
`, "Extra + count", `package sample

const Extra = 1

func Render(count int) {
	var total int
	adjusted := Extra + count
	total = adjusted
	_ = total
}
`)
}

func TestExtractVariableAcceptsTypedConstantOnly(t *testing.T) {
	runVariableExtractionCase(t, `package sample

const Step int = 1

func Render() {
	var total int
	total = Step
	_ = total
}
`, "Step", `package sample

const Step int = 1

func Render() {
	var total int
	adjusted := Step
	total = adjusted
	_ = total
}
`)
}

func TestExtractVariableAcceptsUntypedShiftWithTypedInput(t *testing.T) {
	runVariableExtractionCase(t, `package sample

func Render(power int) {
	var total int
	total = 1 << power
	_ = total
}
`, "1 << power", `package sample

func Render(power int) {
	var total int
	adjusted := 1 << power
	total = adjusted
	_ = total
}
`)
}

func runVariableExtractionCase(t *testing.T, source, discriminator, want string) {
	t.Helper()
	path := filepath.Join(extractVariableTestDir(t), "sample.go")
	if err := os.WriteFile(path, []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := astedit.ExtractVariable(context.Background(), path, "Render", "adjusted", astedit.ExtractVariableOptions{Discriminator: discriminator}); err != nil {
		t.Fatalf("ExtractVariable() error = %v", err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != want {
		t.Fatalf("ExtractVariable() source mismatch\nwant:\n%s\ngot:\n%s", want, got)
	}
}

func extractVariableTestDir(t *testing.T) string {
	t.Helper()
	base, err := filepath.Abs(filepath.Join("..", "..", ".scratch"))
	if err != nil {
		t.Fatal(err)
	}
	dir, err := os.MkdirTemp(base, "extract-variable-test-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.RemoveAll(dir); err != nil {
			t.Errorf("remove extraction test directory: %v", err)
		}
	})
	return dir
}

func TestExtractVariableSharesStableExpressionAcrossDirectAssignmentRange(t *testing.T) {
	t.Parallel()
	before := `package sample

func Render(input int) int {
	var first int
	var second int
	first = input + 1
	second = input + 1
	return first + second
}
`
	path := filepath.Join(extractVariableTestDir(t), "sample.go")
	if err := os.WriteFile(path, []byte(before), 0o600); err != nil {
		t.Fatal(err)
	}
	revisionSum := sha256.Sum256([]byte(before))
	_, err := astedit.ExtractVariable(context.Background(), path, "Render", "adjusted", astedit.ExtractVariableOptions{
		Discriminator: "input + 1", ConstructPath: "2", EndConstructPath: "3",
		ExpectedRevision: hex.EncodeToString(revisionSum[:]),
	})
	if err != nil {
		t.Fatalf("ExtractVariable() range error = %v", err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	want := `package sample

func Render(input int) int {
	var first int
	var second int
	adjusted := input + 1
	first = adjusted
	second = adjusted
	return first + second
}
`
	if string(got) != want {
		t.Fatalf("assignment range extraction mismatch\nwant:\n%s\ngot:\n%s", want, got)
	}
}

func TestExtractVariableRefusesUnsafeAssignmentRanges(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, source, startPath, endPath, refusal string
	}{
		{
			name: "writes input",
			source: `package sample

func Render(input int) int {
	var total int
	total = input + 1
	input = input + 1
	return total
}
`,
			startPath: "1", endPath: "2", refusal: "existing owner-local variable",
		},
		{
			name: "short declaration",
			source: `package sample

func Render(input int) int {
	first := input + 1
	second := input + 1
	return first + second
}
`,
			startPath: "0", endPath: "1", refusal: "one-target simple assignments",
		},
		{
			name: "nested block",
			source: `package sample

func Render(input int) int {
	var first int
	var second int
	if input > 0 {
		first = input + 1
		second = input + 1
	}
	return first + second
}
`,
			startPath: "2.0.0", endPath: "2.0.1", refusal: "nested or control-flow assignments",
		},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			assertAssignmentRangeRefusal(t, test.source, test.startPath, test.endPath, test.refusal)
		})
	}
}

func assertAssignmentRangeRefusal(t *testing.T, before, startPath, endPath, refusal string) {
	t.Helper()
	path := filepath.Join(extractVariableTestDir(t), "sample.go")
	if err := os.WriteFile(path, []byte(before), 0o600); err != nil {
		t.Fatal(err)
	}
	revisionSum := sha256.Sum256([]byte(before))
	_, err := astedit.ExtractVariable(context.Background(), path, "Render", "adjusted", astedit.ExtractVariableOptions{
		Discriminator: "input + 1", ConstructPath: startPath, EndConstructPath: endPath,
		ExpectedRevision: hex.EncodeToString(revisionSum[:]),
	})
	if err == nil || !strings.Contains(err.Error(), refusal) {
		t.Fatalf("ExtractVariable() error = %v, want refusal containing %q", err, refusal)
	}
	got, readErr := os.ReadFile(path)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(got) != before {
		t.Fatalf("refused assignment range changed source\nwant:\n%s\ngot:\n%s", before, got)
	}
}

func TestExtractVariableRewritesVarInitializerRHS(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, source, discriminator, want string
	}{
		{
			name: "explicit named type",
			source: `package sample

type Count int

func Render(count Count) {
	var total Count = count + 1
	_ = total
}

`,
			discriminator: "count + 1",
			want: `package sample

type Count int

func Render(count Count) {
	adjusted := count + 1
	var total Count = adjusted
	_ = total
}
`,
		},
		{
			name: "nested inferred declaration",
			source: `package sample

func Render(count int) {
	if count > 0 {
		var total = count + 2
		_ = total
	}
}
`,
			discriminator: "count + 2",
			want: `package sample

func Render(count int) {
	if count > 0 {
		adjusted := count + 2
		var total = adjusted
		_ = total
	}
}
`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			runVariableExtractionCase(t, tc.source, tc.discriminator, tc.want)
		})
	}
}

func TestExtractVariableExtractsCompleteIfConditionAndRejectsShadowedInput(t *testing.T) {
	t.Parallel()

	t.Run("complete condition", func(t *testing.T) {
		t.Parallel()
		runVariableExtractionCase(t,
			"package sample\nfunc Render(count int) int {\n\tif count > 0 { return count }\n\treturn 0\n}\n",
			"count > 0",
			"package sample\n\nfunc Render(count int) int {\n\tadjusted := count > 0\n\tif adjusted {\n\t\treturn count\n\t}\n\treturn 0\n}\n",
		)
	})

	t.Run("shadowed owner parameter", func(t *testing.T) {
		t.Parallel()
		source := "package sample\nfunc Render(ready bool) bool {\n\t{\n\t\tconst ready = true\n\t\tif ready { return true }\n\t}\n\treturn false\n}\n"
		path := filepath.Join(extractVariableTestDir(t), "sample.go")
		if err := os.WriteFile(path, []byte(source), 0o600); err != nil {
			t.Fatal(err)
		}
		_, err := astedit.ExtractVariable(context.Background(), path, "Render", "adjusted", astedit.ExtractVariableOptions{Discriminator: "ready"})
		if err == nil || !strings.Contains(err.Error(), "resolved typed Go input") {
			t.Fatalf("ExtractVariable() error = %v, want refusal for shadowed untyped input", err)
		}
		got, readErr := os.ReadFile(path)
		if readErr != nil {
			t.Fatal(readErr)
		}
		if string(got) != source {
			t.Fatalf("refused extraction changed source\nwant:\n%s\ngot:\n%s", source, got)
		}
	})
}

func TestExtractVariableRewritesNamedResultReturnInPlace(t *testing.T) {
	source := `package sample

func Render(value int) (result int) {
	if value > 0 {
		return value + 1
	}
	return -value
}
`
	want := `package sample

func Render(value int) (result int) {
	if value > 0 {
		adjusted := value + 1
		return adjusted
	}
	return -value
}
`
	runVariableExtractionCase(t, source, "value + 1", want)
}
