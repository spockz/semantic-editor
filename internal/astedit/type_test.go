// Tests for specialized type insertion, section placement, and import resolution.
package astedit

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInsertType_SectionPlacement(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "model.go")

	initial := `package model

import "fmt"

func Process() {
	fmt.Println("process")
}

func helper() {}
`
	if err := os.WriteFile(file, []byte(initial), 0o600); err != nil {
		t.Fatalf("write initial file: %v", err)
	}

	ctx := context.Background()

	// 1. Insert public struct Config -> should land in public section (before Process)
	err := InsertType(ctx, file, `type Config struct { Host string }`, TypeOptions{
		AccessModifier: AccessModifierInfer,
	})
	if err != nil {
		t.Fatalf("InsertType public struct failed: %v", err)
	}

	data, err := os.ReadFile(filepath.Clean(file))
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	content := string(data)

	posConfig := strings.Index(content, "type Config struct")
	posProcess := strings.Index(content, "func Process()")

	if posConfig == -1 || posProcess == -1 {
		t.Fatalf("expected Config and Process present, got:\n%s", content)
	}
	if posConfig >= posProcess {
		t.Errorf("expected Config to precede Process, got Config=%d, Process=%d", posConfig, posProcess)
	}

	// 2. Insert private struct state -> should land in private section (after Process, before helper)
	err = InsertType(ctx, file, `type state struct { count int }`, TypeOptions{
		AccessModifier: AccessModifierPrivate,
	})
	if err != nil {
		t.Fatalf("InsertType private struct failed: %v", err)
	}

	data, err = os.ReadFile(filepath.Clean(file))
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	content = string(data)

	posState := strings.Index(content, "type state struct")
	posHelper := strings.Index(content, "func helper()")

	if posState == -1 || posHelper == -1 {
		t.Fatalf("expected state and helper present, got:\n%s", content)
	}
	if posProcess >= posState || posState >= posHelper {
		t.Errorf("expected state to land in private section before helper, got Process=%d, state=%d, helper=%d", posProcess, posState, posHelper)
	}
}

func TestInsertType_Validation(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "model.go")

	initial := `package model

func Run() {}
`
	if err := os.WriteFile(file, []byte(initial), 0o600); err != nil {
		t.Fatalf("write initial file: %v", err)
	}

	ctx := context.Background()

	// 1. Non-type declaration rejected
	err := InsertType(ctx, file, `func NotAType() {}`, TypeOptions{})
	if err == nil {
		t.Fatal("expected error passing function to InsertType, got nil")
	}

	// 2. Section violation
	err = InsertType(ctx, file, `type PublicType struct {}`, TypeOptions{
		Placement: PlacementPrivateEnd,
	})
	if err == nil {
		t.Fatal("expected error placing public type in private section, got nil")
	}
}
