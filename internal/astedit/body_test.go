// Package astedit_test verifies AST editing functionality.
package astedit_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"semedit/internal/astedit"
	"semedit/internal/symbol"
)

func TestReplaceBody_TopLevelFunction(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "main.go")
	initial := `package main

import "fmt"

func Calculate(a, b int) int {
	return a + b
}
`
	if err := os.WriteFile(filePath, []byte(initial), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}

	diff, err := astedit.ReplaceBody(context.Background(), filePath, "Calculate", "return a * b", astedit.BodyOptions{})
	if err != nil {
		t.Fatalf("ReplaceBody failed: %v", err)
	}

	if !strings.Contains(diff, "--- a/Calculate") || !strings.Contains(diff, "-return a + b") || !strings.Contains(diff, "+return a * b") {
		t.Errorf("unexpected diff: %s", diff)
	}

	updated, err := os.ReadFile(filepath.Clean(filePath))
	if err != nil {
		t.Fatalf("read updated file: %v", err)
	}

	expected := `package main

func Calculate(a, b int) int {
	return a * b
}
`
	if string(updated) != expected {
		t.Errorf("content mismatch:\ngot:\n%s\nwant:\n%s", string(updated), expected)
	}
}

func TestReplaceBody_MethodPointerReceiver(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "server.go")
	initial := `package main

type Server struct {
	running bool
}

func (s *Server) Start() error {
	s.running = true
	return nil
}
`
	if err := os.WriteFile(filePath, []byte(initial), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}

	_, err := astedit.ReplaceBody(context.Background(), filePath, "(*Server).Start", `if s.running {
	return nil
}
s.running = true
return nil`, astedit.BodyOptions{})
	if err != nil {
		t.Fatalf("ReplaceBody method failed: %v", err)
	}

	updated, err := os.ReadFile(filepath.Clean(filePath))
	if err != nil {
		t.Fatalf("read updated file: %v", err)
	}

	if !strings.Contains(string(updated), "if s.running {") {
		t.Errorf("expected updated method body in:\n%s", string(updated))
	}
}

func TestReplaceBody_SymbolNotFound(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "calc.go")
	initial := `package calc

func Add(a, b int) int { return a + b }
`
	if err := os.WriteFile(filePath, []byte(initial), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}

	_, err := astedit.ReplaceBody(context.Background(), filePath, "Subtract", "return a - b", astedit.BodyOptions{})
	if err == nil {
		t.Fatalf("expected error for non-existent symbol")
	}

	if _, ok := errors.AsType[*symbol.SymbolError](err); !ok {
		t.Fatalf("expected *symbol.SymbolError, got %T: %v", err, err)
	}
	if !errors.Is(err, symbol.ErrNotFound) {
		t.Fatalf("expected ErrNotFound in chain, got %v", err)
	}
}

func TestReplaceBody_SyntaxErrorZeroDiskMutation(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "calc.go")
	initial := `package calc

func Add(a, b int) int {
	return a + b
}
`
	if err := os.WriteFile(filePath, []byte(initial), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}

	_, err := astedit.ReplaceBody(context.Background(), filePath, "Add", "invalid Go syntax {{", astedit.BodyOptions{})
	if err == nil {
		t.Fatalf("expected error for invalid syntax")
	}

	if _, ok := errors.AsType[*astedit.SyntaxError](err); !ok {
		t.Fatalf("expected *astedit.SyntaxError, got %T: %v", err, err)
	}

	current, err := os.ReadFile(filepath.Clean(filePath))
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	if string(current) != initial {
		t.Fatalf("file was mutated despite syntax error:\n%s", string(current))
	}
}

func TestReplaceBody_AutoOrganizeImports(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "printer.go")
	initial := `package main

import (
	"fmt"
	"strings"
)

func PrintMsg(s string) {
	fmt.Println(strings.ToUpper(s))
}
`
	if err := os.WriteFile(filePath, []byte(initial), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}
	_, err := astedit.ReplaceBody(context.Background(), filePath, "PrintMsg", `fmt.Println(s)`, astedit.BodyOptions{AutoOrganizeImports: true})
	if err != nil {
		t.Fatalf("ReplaceBody with auto imports failed: %v", err)
	}
	updated, err := os.ReadFile(filepath.Clean(filePath))
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	if strings.Contains(string(updated), `"strings"`) {
		t.Errorf("expected unused import 'strings' to be removed by AutoOrganizeImports:\n%s", string(updated))
	}
}

func TestReplaceBodyPrunesImportsUsingResolvedPackageName(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	write := func(path, content string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatalf("create directory: %v", err)
		}
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}
	write(filepath.Join(root, "go.mod"), "module example.com/body\n\ngo 1.23\n")
	write(filepath.Join(root, "renamed", "package.go"), "package actualname\n\nfunc Value() string { return \"x\" }\n")
	file := filepath.Join(root, "body.go")
	write(file, `package body

import (
	text "strings"
	"example.com/body/renamed"
)

func Value(input string) string {
	return renamed.Value() + text.TrimSpace(input)
}
`)
	if _, err := astedit.ReplaceBody(context.Background(), file, "Value", "return text.TrimSpace(input)", astedit.BodyOptions{}); err != nil {
		t.Fatalf("ReplaceBody: %v", err)
	}
	updated, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("read result: %v", err)
	}
	if strings.Contains(string(updated), "example.com/body/renamed") || !strings.Contains(string(updated), `text "strings"`) {
		t.Fatalf("imports were not pruned by resolved package name while retaining the used import:\n%s", updated)
	}
}

func TestReplaceBodyPrunesOrdinaryImportsAndPreservesCgo(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	file := filepath.Join(root, "body.go")
	initial := `package body

/*
#include <stdlib.h>
*/
import "C"

import "fmt"

func Value() int { fmt.Println("old"); return 1 }
`
	if err := os.WriteFile(file, []byte(initial), 0o600); err != nil {
		t.Fatalf("write source: %v", err)
	}
	if _, err := astedit.ReplaceBody(context.Background(), file, "Value", "return 2", astedit.BodyOptions{}); err != nil {
		t.Fatalf("ReplaceBody: %v", err)
	}
	updated, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("read result: %v", err)
	}
	if !strings.Contains(string(updated), "#include <stdlib.h>") || !strings.Contains(string(updated), `import "C"`) {
		t.Fatalf("cgo preamble or pseudo-import was removed:\n%s", updated)
	}
	if strings.Contains(string(updated), `"fmt"`) {
		t.Fatalf("orphaned fmt import was retained:\n%s", updated)
	}
}

func TestReplaceBody_RejectsWrapperEscapeWithoutMutation(t *testing.T) {
	t.Parallel()
	filePath := filepath.Join(t.TempDir(), "body.go")
	initial := "package sample\nfunc Run() { old() }\n"
	if err := os.WriteFile(filePath, []byte(initial), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}

	_, err := astedit.ReplaceBody(context.Background(), filePath, "Run", "return\n}\nfunc Injected() {", astedit.BodyOptions{})
	if err == nil {
		t.Fatal("expected wrapper escape to be rejected")
	}
	after, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	if string(after) != initial {
		t.Fatalf("file changed after wrapper escape: %s", after)
	}
}

func TestReplaceBodyStripsOnlyOneASTProvenOuterBlock(t *testing.T) {
	tests := []struct {
		name                 string
		body                 string
		wantStandaloneBlocks int
		wantComments         []string
	}{
		{
			name: "outer block with braces in literals and comments",
			body: "{\nvalue := \"raw } string\"\nobject := map[string]int{\"key\": 1}\n// }\n_ = value\n_ = object\n}",
		},
		{
			name:                 "sequential blocks remain separate",
			body:                 "{ first() }\n{ second() }",
			wantStandaloneBlocks: 2,
		},
		{
			name:                 "comments outside braces remain outside",
			body:                 "// before\n{\nreturn\n}\n// after",
			wantStandaloneBlocks: 1,
			wantComments:         []string{"// before", "// after"},
		},
		{
			name: "labels and nested control flow",
			body: "{\nouter:\nfor {\nif ready { break outer }\n}\n}",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			path := filepath.Join(root, "main.go")
			if err := os.WriteFile(path, []byte("package main\n\nfunc Run() {}\n"), 0o600); err != nil {
				t.Fatalf("write source: %v", err)
			}
			if _, err := astedit.ReplaceBody(context.Background(), path, "Run", test.body, astedit.BodyOptions{}); err != nil {
				t.Fatalf("ReplaceBody: %v", err)
			}
			updated, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read source: %v", err)
			}
			content := string(updated)
			if got := strings.Count(content, "\n\t{"); got != test.wantStandaloneBlocks {
				t.Errorf("standalone body blocks = %d, want %d:\n%s", got, test.wantStandaloneBlocks, content)
			}
			for _, comment := range test.wantComments {
				if !strings.Contains(content, comment) {
					t.Errorf("updated source lost %q:\n%s", comment, content)
				}
			}
		})
	}
}
