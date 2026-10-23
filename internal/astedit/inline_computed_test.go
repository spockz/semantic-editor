// WHY: Exercises computed inline-value through the public editor with source-preserving safety refusals.

package astedit

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spockz/semantic-editor/internal/sourcebuf"
)

func TestInlineComputedVariable(t *testing.T) {
	t.Parallel()
	cases := []struct{ name, declarations, signature, body, want string }{
		{"local arithmetic", "", "(seed int) any", "base := seed + 1\nvalue := -(base * 2) + 3\nreturn value", "return -(base * 2) + 3"},
		{"immediate direct call argument", "type Count int\nfunc record(value Count) {}", "(seed Count)", "value := seed + 2\nrecord(value)", "record(seed + 2)"},
		{"immediate simple assignment", "", "(seed int) int", "var total int\nvalue := seed + 2\ntotal = value\nreturn total", "total = seed + 2"},
		{"immediate package assignment declared later", "var total int", "(seed int) int", "value := seed + 2\ntotal = value\nreturn total", "total = seed + 2"},
		{"named scalar", "type Count int", "(seed Count) any", "value := seed + 2\nreturn value", "return seed + 2"},
		{"boolean", "", "(ready bool) bool", "value := !ready\nreturn value", "return !ready"},
		{"package constant declared later", "const value = 4", "() any", "value := value + 1\nreturn value", "return value + 1"},
		{"constant expression", "", "() any", "value := 1E3 + 2\nreturn value", "return 1E3 + 2"},
		{"inferred boolean", "", "() any", "value := true\nreturn value", "return true"},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			file := filepath.Join(t.TempDir(), "value.go")
			before := "package sample\nfunc Build" + testCase.signature + " {\n" + testCase.body + "\n}\n" + testCase.declarations + "\n"
			if err := os.WriteFile(file, []byte(before), 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := InlineVariable(context.Background(), file, "Build", "value"); err != nil {
				t.Fatal(err)
			}
			after, err := os.ReadFile(file)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(after), testCase.want) || strings.Contains(string(after), "value :=") {
				t.Fatalf("unexpected source:\n%s", after)
			}
		})
	}
}

func TestInlineComputedVariableRefusesWithoutWriting(t *testing.T) {
	t.Parallel()
	cases := []struct{ name, signature, body string }{
		{"call", "(seed int) int", "value := next()\nreturn value"},
		{"conversion", "(seed int) int", "value := int(seed)\nreturn value"},
		{"global read", "(seed int) int", "value := global + 1\nreturn value"},
		{"changed dependency", "(seed int) int", "value := seed + 1\nseed++\nreturn value"},
		{"index", "(seed []int) int", "value := seed[0] + 1\nreturn value"},
		{"dereference", "(seed *int) int", "value := *seed + 1\nreturn value"},
		{"channel", "(seed <-chan int) int", "value := <-seed + 1\nreturn value"},
		{"inferred var", "(seed int) int", "var value = seed + 1\nreturn value"},
		{"named result", "(seed int) (result int)", "value := seed + 1\nreturn value"},
		{"indirect return", "(seed int) int", "value := seed + 1\nreturn value + 1"},
		{"multiple call arguments", "(seed int)", "value := seed + 1\nrecord(seed, value)"},
		{"method call argument", "(seed int)", "value := seed + 1\nreceiver.record(value)"},
		{"function value target", "(fn func(int), seed int)", "value := seed + 1\nfn(value)"},
		{"non-adjacent assignment", "(seed int) int", "var total int\nvalue := seed + 1\ntotal++\ntotal = value\nreturn total"},
		{"compound assignment", "(seed int) int", "var total int\nvalue := seed + 1\ntotal += value\nreturn total"},
		{"multiple assignment targets", "(seed int) int", "var left, total int\nvalue := seed + 1\nleft, total = 0, value\nreturn total"},
		{"selector assignment target", "(seed int) int", "value := seed + 1\nstate.total = value"},
		{"extra assignment use", "(seed int) int", "var total int\nvalue := seed + 1\ntotal = value\n_ = value\nreturn total"},
		{"extra reference", "(seed int) int", "value := seed + 1\nreturn value\n_ = value\nreturn 0"},
		{"comment", "(seed int) int", "value := seed + /* keep */ 1\nreturn value"},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			file := filepath.Join(t.TempDir(), "value.go")
			before := "package sample\nvar global = 1\ntype stateType struct { total int }\nvar state stateType\nfunc next() int { return 2 }\nfunc record(values ...int) {}\ntype recorder struct{}\nfunc (recorder) record(int) {}\nvar receiver recorder\nfunc Build" + testCase.signature + " {\n" + testCase.body + "\n}\n"
			if err := os.WriteFile(file, []byte(before), 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := InlineVariable(context.Background(), file, "Build", "value"); err == nil {
				t.Fatal("expected refusal")
			}
			after, err := os.ReadFile(file)
			if err != nil {
				t.Fatal(err)
			}
			if string(after) != before {
				t.Fatalf("refusal changed source:\n%s", after)
			}
		})
	}
}

func TestInlineVariableStagesNewBufferedFile(t *testing.T) {
	root := t.TempDir()
	canonicalRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	root = canonicalRoot
	if writeErr := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.test/inline\n\ngo 1.23\n"), 0o600); writeErr != nil {
		t.Fatal(writeErr)
	}
	base := filepath.Join(root, "base.go")
	if writeErr := os.WriteFile(base, []byte("package sample\n\nconst Seed = 2\n"), 0o600); writeErr != nil {
		t.Fatal(writeErr)
	}
	buffer, err := sourcebuf.New(t.Context(), root)
	if err != nil {
		t.Fatal(err)
	}
	ctx := sourcebuf.WithBuffer(t.Context(), buffer)
	file := filepath.Join(root, "value.go")
	before := []byte("package sample\n\nfunc Build(seed int) int {\n\tvalue := seed + 1\n\treturn value\n}\n")
	if writeErr := sourcebuf.WriteFile(ctx, file, before); writeErr != nil {
		t.Fatal(writeErr)
	}

	if _, inlineErr := InlineVariable(ctx, file, "Build", "value"); inlineErr != nil {
		t.Fatalf("inline buffered new file: %v", inlineErr)
	}
	if _, statErr := os.Stat(file); !os.IsNotExist(statErr) {
		t.Fatalf("buffered new file was published early: %v", statErr)
	}
	after, err := sourcebuf.ReadFile(ctx, file)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(after), "return seed + 1") || strings.Contains(string(after), "value :=") {
		t.Fatalf("unexpected buffered inline result:\n%s", after)
	}
	if !sourcebuf.Dirty(ctx, file) {
		t.Fatal("inline result was not staged in the caller-owned buffer")
	}
}

func TestInlineComputedVariableShortDeclaration(t *testing.T) {
	t.Parallel()
	cases := []struct{ name, declarations, signature, body, want string }{
		{"local target", "", "(seed int) int", "value := seed + 2\nresult := value\nreturn result", "result := seed + 2"},
		{"package name shadowed by short declaration", "var result int", "(seed int) int", "value := seed + 2\nresult := value\nreturn result", "result := seed + 2"},
		{"single var initializer with inferred type", "", "(seed int) int", "value := seed + 2\nvar result = value\nreturn result", "var result = seed + 2"},
		{"single var initializer with explicit type", "", "(seed int) int", "value := seed + 2\nvar result int = value\nreturn result", "var result int = seed + 2"},
		{"destination shadows a package constant after initialization", "const seed = 5", "() int", "value := seed + 2\nvar seed = value\nreturn seed", "var seed = seed + 2"},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			file := filepath.Join(t.TempDir(), "value.go")
			before := "package sample\n" + testCase.declarations + "\nfunc Build" + testCase.signature + " {\n" + testCase.body + "\n}\n"
			if err := os.WriteFile(file, []byte(before), 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := InlineVariable(context.Background(), file, "Build", "value"); err != nil {
				t.Fatal(err)
			}
			after, err := os.ReadFile(file)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(after), testCase.want) || strings.Contains(string(after), "value :=") {
				t.Fatalf("unexpected source:\n%s", after)
			}
		})
	}
}

func TestInlineComputedVariableShortDeclarationRefusesWithoutWriting(t *testing.T) {
	t.Parallel()
	cases := []struct{ name, body string }{
		{"multiple targets", "value := seed + 1\nother, total := 0, value\nreturn total + other"},
		{"non-adjacent", "var counter int\nvalue := seed + 1\ncounter++\ntotal := value\nreturn total"},
		{"extra reference", "value := seed + 1\ntotal := value\n_ = value\nreturn total"},
		{"grouped var declaration", "value := seed + 1\nvar (\ntotal = value\n)\nreturn total"},
		{"multiple var names", "value := seed + 1\nvar total, other = value, 0\nreturn total + other"},
		{"multiple var values", "value := seed + 1\nvar total, other = value, seed\nreturn total + other"},
		{"initializer is not the complete RHS", "value := seed + 1\nvar total = value + seed\nreturn total"},
		{"intervening comment", "value := seed + 1\n// keep this boundary\nvar total = value\nreturn total"},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			file := filepath.Join(t.TempDir(), "value.go")
			before := "package sample\nfunc Build(seed int) int {\n" + testCase.body + "\n}\n"
			if err := os.WriteFile(file, []byte(before), 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := InlineVariable(context.Background(), file, "Build", "value"); err == nil {
				t.Fatal("expected refusal")
			}
			after, err := os.ReadFile(file)
			if err != nil {
				t.Fatal(err)
			}
			if string(after) != before {
				t.Fatalf("refusal changed source:\n%s", after)
			}
		})
	}
}

func TestInlineComputedVariableIfCondition(t *testing.T) {
	t.Parallel()
	file := filepath.Join(t.TempDir(), "value.go")
	before := "package sample\nfunc Build(ready bool) bool {\nvalue := ready && ready\nif value { return true }\nreturn false\n}\n"
	if err := os.WriteFile(file, []byte(before), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := InlineVariable(context.Background(), file, "Build", "value"); err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(after), "if ready && ready") || strings.Contains(string(after), "value :=") {
		t.Fatalf("unexpected source:\n%s", after)
	}
}

func TestInlineComputedVariableIfConditionRefusesWithoutWriting(t *testing.T) {
	t.Parallel()
	cases := []struct{ name, body string }{
		{"if init", "value := ready\nif condition := true; value { return condition }\nreturn false"},
		{"subexpression", "value := ready\nif !value { return true }\nreturn false"},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			file := filepath.Join(t.TempDir(), "value.go")
			before := "package sample\nfunc Build(ready bool) bool {\n" + testCase.body + "\n}\n"
			if err := os.WriteFile(file, []byte(before), 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := InlineVariable(context.Background(), file, "Build", "value"); err == nil {
				t.Fatal("expected refusal")
			}
			after, err := os.ReadFile(file)
			if err != nil {
				t.Fatal(err)
			}
			if string(after) != before {
				t.Fatalf("refusal changed source:\n%s", after)
			}
		})
	}
}

func TestInlineComputedVariableSwitchTag(t *testing.T) {
	t.Parallel()
	t.Run("complete tag", func(t *testing.T) {
		file := filepath.Join(t.TempDir(), "value.go")
		before := "package sample\nfunc Build(code int) int {\nvalue := code + 1\nswitch value { case 2: return 7; default: return 3 }\n}\n"
		if err := os.WriteFile(file, []byte(before), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := InlineVariable(context.Background(), file, "Build", "value"); err != nil {
			t.Fatal(err)
		}
		after, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(after), "switch code + 1") || strings.Contains(string(after), "value :=") {
			t.Fatalf("unexpected source:\n%s", after)
		}
	})
	t.Run("init and partial tag refuse unchanged", func(t *testing.T) {
		cases := []struct{ name, body string }{
			{"switch init", "value := ready + 1\nswitch ready = 10; value { case 2: return true }\nreturn false"},
			{"partial tag", "value := ready + 1\nswitch value + 1 { case 2: return true }\nreturn false"},
		}
		for _, testCase := range cases {
			t.Run(testCase.name, func(t *testing.T) {
				file := filepath.Join(t.TempDir(), "value.go")
				before := "package sample\nfunc Build(ready int) bool {\n" + testCase.body + "\n}\n"
				if err := os.WriteFile(file, []byte(before), 0o600); err != nil {
					t.Fatal(err)
				}
				if _, err := InlineVariable(context.Background(), file, "Build", "value"); err == nil {
					t.Fatal("expected refusal")
				}
				after, err := os.ReadFile(file)
				if err != nil {
					t.Fatal(err)
				}
				if string(after) != before {
					t.Fatalf("refusal changed source:\n%s", after)
				}
			})
		}
	})
}

func TestInlineComputedVariableSwitchTagBoundaries(t *testing.T) {
	t.Parallel()
	t.Run("void owner", func(t *testing.T) {
		file := filepath.Join(t.TempDir(), "value.go")
		before := "package sample\nfunc Build(code int) {\nvalue := code + 1\nswitch value { case 2: return; default: return }\n}\n"
		if err := os.WriteFile(file, []byte(before), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := InlineVariable(context.Background(), file, "Build", "value"); err != nil {
			t.Fatal(err)
		}
		after, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(after), "switch code + 1") || strings.Contains(string(after), "value :=") {
			t.Fatalf("unexpected source:\n%s", after)
		}
	})
	t.Run("parenthesized keyed struct tag refuses through scalar proof", func(t *testing.T) {
		file := filepath.Join(t.TempDir(), "value.go")
		before := "package sample\ntype Record struct { Count int }\nfunc Build() {\nvalue := (Record{Count: 1})\nswitch value { case Record{Count: 1}: return; default: return }\n}\n"
		if err := os.WriteFile(file, []byte(before), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := InlineVariable(context.Background(), file, "Build", "value"); err == nil {
			t.Fatal("expected composite switch tag refusal")
		} else if !strings.Contains(err.Error(), "computed initializer must have a resolved scalar type") {
			t.Fatalf("expected scalar-type refusal, got %v", err)
		}
		after, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		if string(after) != before {
			t.Fatalf("refusal changed source:\n%s", after)
		}
	})
}

func TestInlineComputedVariableDeferCall(t *testing.T) {
	t.Parallel()
	file := filepath.Join(t.TempDir(), "value.go")
	before := "package sample\nfunc consume(int) {}\nfunc Build(input int) {\nvalue := input + 1\ndefer consume(value)\n}\n"
	if err := os.WriteFile(file, []byte(before), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := InlineVariable(context.Background(), file, "Build", "value"); err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(after), "defer consume(input + 1)") || strings.Contains(string(after), "value :=") {
		t.Fatalf("unexpected source:\n%s", after)
	}
}

func TestInlineComputedVariableGoCall(t *testing.T) {
	t.Parallel()
	file := filepath.Join(t.TempDir(), "value.go")
	before := "package sample\nfunc consume(int) {}\nfunc Build(input int) {\nvalue := input + 1\ngo consume(value)\n}\n"
	if err := os.WriteFile(file, []byte(before), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := InlineVariable(context.Background(), file, "Build", "value"); err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(after), "go consume(input + 1)") || strings.Contains(string(after), "value :=") {
		t.Fatalf("unexpected source:\n%s", after)
	}
}
