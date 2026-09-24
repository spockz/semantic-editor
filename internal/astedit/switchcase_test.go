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

func TestInsertCase_BoundaryPlacement(t *testing.T) {
	tests := []struct {
		name      string
		placement astedit.CasePlacement
		source    string
		before    string
		after     string
	}{
		{"first", astedit.CasePlacementFirst, "case \"init\":\n\treturn \"initialized\"", "case \"init\":", "case \"start\":"},
		{"last", astedit.CasePlacementLast, "case \"extra\":\n\treturn \"extra\"", "default:", "case \"extra\":"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			filePath := filepath.Join(t.TempDir(), "main.go")
			if err := os.WriteFile(filePath, []byte(switchSource), 0o600); err != nil {
				t.Fatalf("write file: %v", err)
			}
			if _, err := astedit.InsertCase(context.Background(), filePath, "Handle", "action", test.source, astedit.CaseOptions{Placement: test.placement}); err != nil {
				t.Fatalf("InsertCase %s failed: %v", test.name, err)
			}
			content, err := os.ReadFile(filepath.Clean(filePath))
			if err != nil {
				t.Fatalf("read file: %v", err)
			}
			str := string(content)
			before := strings.Index(str, test.before)
			after := strings.Index(str, test.after)
			if before == -1 || after == -1 || before > after {
				t.Errorf("expected %q before %q, got:\n%s", test.before, test.after, str)
			}
		})
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

func TestInsertCase_AmbiguousSwitchReportsAndAcceptsPath(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "nested.go")
	initial := `package main
func Route(mode string) string {
	switch mode {
	case "outer":
		switch mode {
		case "inner": return "inner"
		default: return "nested fallback"
		}
	default: return "outer fallback"
	}
}
`
	if err := os.WriteFile(filePath, []byte(initial), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}
	_, err := astedit.InsertCase(context.Background(), filePath, "Route", "mode", `case "new": return "new"`, astedit.CaseOptions{})
	if !errors.Is(err, astedit.ErrSwitchAmbiguous) {
		t.Fatalf("expected ErrSwitchAmbiguous, got: %v", err)
	}
	for _, want := range []string{"0  cases:", "0.1  cases:", `"outer"`, `"inner"`, "switch_path"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("ambiguity error missing %q: %v", want, err)
		}
	}
	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("read unchanged file: %v", err)
	}
	if string(content) != initial {
		t.Fatalf("ambiguous request changed file:\n%s", content)
	}
	if _, err := astedit.InsertCase(context.Background(), filePath, "Route", "mode", `case "new": return "new"`, astedit.CaseOptions{SwitchPath: "0.1"}); err != nil {
		t.Fatalf("InsertCase with suggested nested path failed: %v", err)
	}
	content, err = os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("read updated file: %v", err)
	}
	updated := string(content)
	inner := strings.Index(updated, `case "inner"`)
	inserted := strings.Index(updated, `case "new"`)
	outerFallback := strings.LastIndex(updated, "default:")
	if inner < 0 || inserted < inner || outerFallback < inserted {
		t.Fatalf("path 0.1 should insert into the nested switch:\n%s", updated)
	}
}

func TestInsertCase_TypeSwitchMatchesAssertedExpression(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "typeswitch.go")
	initial := `package main

func Route(mode string, input any) string {
	switch mode {
	case "old":
		return "old"
	default:
		return "fallback"
	}
	switch modeExtra := input.(type) {
	case string:
		return modeExtra
	default:
		return "fallback"
	}
}
`
	if err := os.WriteFile(filePath, []byte(initial), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}

	if _, err := astedit.InsertCase(context.Background(), filePath, "Route", "mode", `case "new":
		return "new"`, astedit.CaseOptions{}); err != nil {
		t.Fatalf("InsertCase on mode switch failed: %v", err)
	}
	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("read file after mode switch insertion: %v", err)
	}
	updated := string(content)
	modeCase := strings.Index(updated, `case "new":`)
	typeSwitch := strings.Index(updated, "switch modeExtra := input.(type)")
	if modeCase < 0 || typeSwitch < 0 || modeCase > typeSwitch {
		t.Fatalf("switch_on=mode should select only the ordinary mode switch:\n%s", updated)
	}

	if _, err := astedit.InsertCase(context.Background(), filePath, "Route", "input", `case bool:
		return "bool"`, astedit.CaseOptions{}); err != nil {
		t.Fatalf("InsertCase on type-switch asserted expression failed: %v", err)
	}
	content, err = os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("read file after type-switch insertion: %v", err)
	}
	updated = string(content)
	typeSwitch = strings.Index(updated, "switch modeExtra := input.(type)")
	boolCase := strings.Index(updated, "case bool:")
	if typeSwitch < 0 || boolCase < typeSwitch {
		t.Fatalf("switch_on=input should select the type switch asserted expression:\n%s", updated)
	}
}
