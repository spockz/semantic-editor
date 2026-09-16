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

const switchSource = `package main

func Handle(action string) string {
	switch action {
	case "start":
		return "starting"
	case "stop":
		return "stopping"
	default:
		return "unknown"
	}
}
`

func TestInsertCase_BeforeDefault(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "main.go")
	if err := os.WriteFile(filePath, []byte(switchSource), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}

	diff, err := astedit.InsertCase(context.Background(), filePath, "Handle", "action", `case "pause":
	return "paused"`, astedit.CaseOptions{
		Placement: astedit.CasePlacementBeforeDefault,
	})
	if err != nil {
		t.Fatalf("InsertCase failed: %v", err)
	}
	if diff == "" {
		t.Errorf("expected non-empty diff")
	}

	content, err := os.ReadFile(filepath.Clean(filePath))
	if err != nil {
		t.Fatalf("read file: %v", err)
	}

	str := string(content)
	idxPause := strings.Index(str, `case "pause":`)
	idxDefault := strings.Index(str, "default:")
	if idxPause == -1 || idxDefault == -1 || idxPause > idxDefault {
		t.Errorf("expected 'case pause' before 'default', got:\n%s", str)
	}
}

func TestInsertCase_First(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "main.go")
	if err := os.WriteFile(filePath, []byte(switchSource), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}

	_, err := astedit.InsertCase(context.Background(), filePath, "Handle", "action", `case "init":
	return "initialized"`, astedit.CaseOptions{
		Placement: astedit.CasePlacementFirst,
	})
	if err != nil {
		t.Fatalf("InsertCase first failed: %v", err)
	}

	content, err := os.ReadFile(filepath.Clean(filePath))
	if err != nil {
		t.Fatalf("read file: %v", err)
	}

	str := string(content)
	idxInit := strings.Index(str, `case "init":`)
	idxStart := strings.Index(str, `case "start":`)
	if idxInit == -1 || idxStart == -1 || idxInit > idxStart {
		t.Errorf("expected 'case init' before 'case start', got:\n%s", str)
	}
}

func TestInsertCase_Last(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "main.go")
	if err := os.WriteFile(filePath, []byte(switchSource), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}

	_, err := astedit.InsertCase(context.Background(), filePath, "Handle", "action", `case "extra":
	return "extra"`, astedit.CaseOptions{
		Placement: astedit.CasePlacementLast,
	})
	if err != nil {
		t.Fatalf("InsertCase last failed: %v", err)
	}

	content, err := os.ReadFile(filepath.Clean(filePath))
	if err != nil {
		t.Fatalf("read file: %v", err)
	}

	str := string(content)
	idxDefault := strings.Index(str, "default:")
	idxExtra := strings.Index(str, `case "extra":`)
	if idxDefault == -1 || idxExtra == -1 || idxExtra < idxDefault {
		t.Errorf("expected 'case extra' after 'default', got:\n%s", str)
	}
}

func TestInsertCase_BeforeAnchor(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "main.go")
	if err := os.WriteFile(filePath, []byte(switchSource), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}

	_, err := astedit.InsertCase(context.Background(), filePath, "Handle", "action", `case "restart":
	return "restarting"`, astedit.CaseOptions{
		Placement:  astedit.CasePlacementBefore,
		AnchorCase: `"stop"`,
	})
	if err != nil {
		t.Fatalf("InsertCase before anchor failed: %v", err)
	}

	content, err := os.ReadFile(filepath.Clean(filePath))
	if err != nil {
		t.Fatalf("read file: %v", err)
	}

	str := string(content)
	idxRestart := strings.Index(str, `case "restart":`)
	idxStop := strings.Index(str, `case "stop":`)
	if idxRestart == -1 || idxStop == -1 || idxRestart > idxStop {
		t.Errorf("expected 'case restart' before 'case stop', got:\n%s", str)
	}
}

func TestInsertCase_AfterAnchor(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "main.go")
	if err := os.WriteFile(filePath, []byte(switchSource), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}

	_, err := astedit.InsertCase(context.Background(), filePath, "Handle", "action", `case "restart":
	return "restarting"`, astedit.CaseOptions{
		Placement:  astedit.CasePlacementAfter,
		AnchorCase: `"start"`,
	})
	if err != nil {
		t.Fatalf("InsertCase after anchor failed: %v", err)
	}

	content, err := os.ReadFile(filepath.Clean(filePath))
	if err != nil {
		t.Fatalf("read file: %v", err)
	}

	str := string(content)
	idxStart := strings.Index(str, `case "start":`)
	idxRestart := strings.Index(str, `case "restart":`)
	idxStop := strings.Index(str, `case "stop":`)
	if idxStart == -1 || idxRestart == -1 || idxStop == -1 || idxRestart < idxStart || idxRestart > idxStop {
		t.Errorf("expected 'case restart' between start and stop, got:\n%s", str)
	}
}

func TestInsertCase_TaglessSwitch(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "tagless.go")
	initial := `package main

func Check(val int) string {
	switch {
	case val < 0:
		return "negative"
	default:
		return "positive"
	}
}
`
	if err := os.WriteFile(filePath, []byte(initial), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}

	_, err := astedit.InsertCase(context.Background(), filePath, "Check", "", `case val == 0:
	return "zero"`, astedit.CaseOptions{
		Placement: astedit.CasePlacementBeforeDefault,
	})
	if err != nil {
		t.Fatalf("InsertCase tagless failed: %v", err)
	}

	content, err := os.ReadFile(filepath.Clean(filePath))
	if err != nil {
		t.Fatalf("read file: %v", err)
	}

	if !strings.Contains(string(content), "case val == 0:") {
		t.Errorf("expected 'case val == 0:' inserted, got:\n%s", string(content))
	}
}

func TestInsertCase_Errors(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "main.go")
	if err := os.WriteFile(filePath, []byte(switchSource), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}

	// 1. Function not found
	_, err := astedit.InsertCase(context.Background(), filePath, "NoSuchFunc", "action", `case "x":`, astedit.CaseOptions{})
	if !errors.Is(err, symbol.ErrNotFound) {
		t.Errorf("expected ErrNotFound for non-existent func, got: %v", err)
	}

	// 2. Switch not found
	_, err = astedit.InsertCase(context.Background(), filePath, "Handle", "nonExistentExpr", `case "x":`, astedit.CaseOptions{})
	if !errors.Is(err, astedit.ErrSwitchNotFound) {
		t.Errorf("expected ErrSwitchNotFound, got: %v", err)
	}

	// 3. Anchor not found
	_, err = astedit.InsertCase(context.Background(), filePath, "Handle", "action", `case "x":`, astedit.CaseOptions{
		Placement:  astedit.CasePlacementBefore,
		AnchorCase: `"missing"`,
	})
	if !errors.Is(err, astedit.ErrAnchorNotFound) {
		t.Errorf("expected ErrAnchorNotFound, got: %v", err)
	}

	// 4. Invalid syntax
	_, err = astedit.InsertCase(context.Background(), filePath, "Handle", "action", `invalid case syntax {{`, astedit.CaseOptions{})
	if _, ok := errors.AsType[*astedit.SyntaxError](err); !ok {
		t.Errorf("expected SyntaxError, got: %v", err)
	}
}
