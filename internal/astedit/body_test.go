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

import "fmt"

func Calculate(a, b int) int {
	return a * b
}
`
	if string(updated) != expected {
		t.Errorf("content mismatch:\ngot:\n%s\nwant:\n%s", string(updated), expected)
	}
}

func TestReplaceBody_MethodPointerReceiver(t *testing.T) {
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

	_, err := astedit.ReplaceBody(context.Background(), filePath, "PrintMsg", `fmt.Println(s)`, astedit.BodyOptions{
		AutoOrganizeImports: true,
	})
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
