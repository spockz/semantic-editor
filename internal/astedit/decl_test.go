// Tests for general top-level declaration insertion and group merging.
package astedit

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInsertDecl_GroupMerging(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "consts.go")

	initial := `package consts

const (
	StatusPending = "pending"
	StatusActive  = "active"
)

var (
	ErrNotFound = "not found"
)
`
	if err := os.WriteFile(file, []byte(initial), 0o600); err != nil {
		t.Fatalf("write initial file: %v", err)
	}

	ctx := context.Background()

	// 1. Append StatusArchived to const block
	err := InsertDecl(ctx, file, `const StatusArchived = "archived"`, DeclOptions{
		Group: "append",
	})
	if err != nil {
		t.Fatalf("InsertDecl append to const group failed: %v", err)
	}

	data, err := os.ReadFile(filepath.Clean(file))
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	content := string(data)

	// Verify StatusArchived is inside the const (...) group before var
	posActive := strings.Index(content, "StatusActive")
	posArchived := strings.Index(content, "StatusArchived")
	posVar := strings.Index(content, "var (")

	if posActive == -1 || posArchived == -1 || posVar == -1 {
		t.Fatalf("expected all symbols present, got:\n%s", content)
	}
	if posActive >= posArchived || posArchived >= posVar {
		t.Errorf("expected StatusArchived inside const group before var, got active=%d, archived=%d, var=%d", posActive, posArchived, posVar)
	}

	// 2. Append ErrTimeout to var block
	err = InsertDecl(ctx, file, `var ErrTimeout = "timeout"`, DeclOptions{})
	if err != nil {
		t.Fatalf("InsertDecl append to var group failed: %v", err)
	}

	data, err = os.ReadFile(filepath.Clean(file))
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	content = string(data)

	posErrNotFound := strings.Index(content, "ErrNotFound")
	posErrTimeout := strings.Index(content, "ErrTimeout")

	if posErrNotFound == -1 || posErrTimeout == -1 {
		t.Fatalf("expected both var errors present, got:\n%s", content)
	}
	if posErrNotFound >= posErrTimeout {
		t.Errorf("expected ErrTimeout to follow ErrNotFound inside var group, got notfound=%d, timeout=%d", posErrNotFound, posErrTimeout)
	}
}

func TestInsertDecl_SentinelVarSortedWithinGroup(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "errors.go")
	initial := `package errors

var (
	ErrBravo = errors.New("bravo")
)
`
	if err := os.WriteFile(file, []byte(initial), 0o600); err != nil {
		t.Fatalf("write initial file: %v", err)
	}
	if err := InsertDecl(context.Background(), file, `var ErrAlpha = errors.New("alpha")`, DeclOptions{}); err != nil {
		t.Fatalf("InsertDecl sentinel failed: %v", err)
	}
	// #nosec G304 -- test reads the controlled temporary source created above.
	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	content := string(data)
	if strings.Index(content, "ErrAlpha") > strings.Index(content, "ErrBravo") {
		t.Fatalf("expected ErrAlpha before ErrBravo, got:\n%s", content)
	}
}

func TestInsertDecl_Standalone(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "values.go")

	initial := `package values

const (
	MaxRetries = 3
)
`
	if err := os.WriteFile(file, []byte(initial), 0o600); err != nil {
		t.Fatalf("write initial file: %v", err)
	}

	ctx := context.Background()

	// Insert standalone constant rather than merging
	err := InsertDecl(ctx, file, `const TimeoutSeconds = 30`, DeclOptions{
		Group:     "standalone",
		Placement: PlacementFileEnd,
	})
	if err != nil {
		t.Fatalf("InsertDecl standalone failed: %v", err)
	}

	data, err := os.ReadFile(filepath.Clean(file))
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	content := string(data)

	if !strings.Contains(content, "const TimeoutSeconds = 30") {
		t.Errorf("expected standalone const declaration, got:\n%s", content)
	}
}

func TestInsertDecl_SectionViolation(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "consts.go")
	initial := `package consts

const PublicConst = 1

const privateConst = 2
`
	if err := os.WriteFile(file, []byte(initial), 0o600); err != nil {
		t.Fatalf("write initial file: %v", err)
	}

	ctx := context.Background()

	err := InsertDecl(ctx, file, `const AnotherPublic = 3`, DeclOptions{
		Placement: PlacementPrivateStart,
	})
	if !errors.Is(err, ErrSectionViolation) {
		t.Fatalf("expected ErrSectionViolation, got %v", err)
	}
	var pErr *PlacementError
	if !errors.As(err, &pErr) {
		t.Fatalf("expected *PlacementError via errors.As, got %T", err)
	}
	if pErr.Strategy != PlacementPrivateStart {
		t.Errorf("expected Strategy PlacementPrivateStart, got %q", pErr.Strategy)
	}

	// Syntax error
	err = InsertDecl(ctx, file, `const broken = {`, DeclOptions{})
	if !errors.Is(err, ErrSyntax) {
		t.Fatalf("expected ErrSyntax, got %v", err)
	}
	var synErr *SyntaxError
	if !errors.As(err, &synErr) {
		t.Fatalf("expected *SyntaxError via errors.As, got %T", err)
	}
	if !synErr.Pos.IsValid() {
		t.Errorf("expected valid Pos in SyntaxError, got %v", synErr.Pos)
	}
}
