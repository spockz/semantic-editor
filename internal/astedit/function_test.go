// Tests for specialized function and method insertion, receiver clustering, and access validation.
package astedit

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInsertFunction_ReceiverClustering(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "service.go")

	initial := `package service

import "fmt"

type Service struct {
	name string
}

func (s *Service) Start() {
	fmt.Println("start")
}

func (s *Service) stop() {
	fmt.Println("stop")
}
`
	if err := os.WriteFile(file, []byte(initial), 0o600); err != nil {
		t.Fatalf("write initial file: %v", err)
	}

	ctx := context.Background()

	// 1. Insert public method on Service -> should cluster after Start(), before stop()
	err := InsertFunction(ctx, file, `func (s *Service) Status() string { return s.name }`, FunctionOptions{
		AccessModifier: AccessModifierInfer,
	})
	if err != nil {
		t.Fatalf("InsertFunction public method failed: %v", err)
	}

	data, err := os.ReadFile(filepath.Clean(file))
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	content := string(data)

	posStart := strings.Index(content, "func (s *Service) Start()")
	posStatus := strings.Index(content, "func (s *Service) Status()")
	posStop := strings.Index(content, "func (s *Service) stop()")

	if posStart == -1 || posStatus == -1 || posStop == -1 {
		t.Fatalf("expected all methods present, got:\n%s", content)
	}

	if posStart >= posStatus || posStatus >= posStop {
		t.Errorf("expected Status to cluster between Start and stop, got Start=%d, Status=%d, stop=%d", posStart, posStatus, posStop)
	}

	// 2. Insert private method on Service -> should cluster after stop()
	err = InsertFunction(ctx, file, `func (s *Service) restart() { fmt.Println("restart") }`, FunctionOptions{
		AccessModifier: AccessModifierPrivate,
	})
	if err != nil {
		t.Fatalf("InsertFunction private method failed: %v", err)
	}

	data, err = os.ReadFile(filepath.Clean(file))
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	content = string(data)

	posStop = strings.Index(content, "func (s *Service) stop()")
	posRestart := strings.Index(content, "func (s *Service) restart()")

	if posStop == -1 || posRestart == -1 {
		t.Fatalf("expected both private methods present, got:\n%s", content)
	}

	if posStop >= posRestart {
		t.Errorf("expected restart to cluster after stop, got stop=%d, restart=%d", posStop, posRestart)
	}
}

func TestInsertFunction_PointerReceiverAnchor(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "transaction.go")
	initial := `package audit

type Event struct{}

type BufferedSink struct {
	queue []Event
}

func (b *BufferedSink) Begin() error {
	b.queue = b.queue[:0]
	return nil
}

func (b *BufferedSink) Commit() error {
	return nil
}
`
	if err := os.WriteFile(file, []byte(initial), 0o600); err != nil {
		t.Fatalf("write initial file: %v", err)
	}

	err := InsertFunction(context.Background(), file, `func (b *BufferedSink) Write(event Event) error {
	b.queue = append(b.queue, event)
	return nil
}`, FunctionOptions{
		Placement:    PlacementAfterSymbol,
		TargetSymbol: "(*BufferedSink).Begin",
	})
	if err != nil {
		t.Fatalf("insert after pointer receiver anchor: %v", err)
	}

	data, err := os.ReadFile(filepath.Clean(file))
	if err != nil {
		t.Fatalf("read result: %v", err)
	}
	content := string(data)
	begin := strings.Index(content, "func (b *BufferedSink) Begin()")
	write := strings.Index(content, "func (b *BufferedSink) Write(event Event)")
	commit := strings.Index(content, "func (b *BufferedSink) Commit()")
	if begin == -1 || write == -1 || commit == -1 || begin >= write || write >= commit {
		t.Fatalf("Write was not inserted after Begin and before Commit:\n%s", content)
	}
}

func TestInsertFunction_SectionViolationAndValidation(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "util.go")

	initial := `package util

func PublicOne() {}

func privateOne() {}
`
	if err := os.WriteFile(file, []byte(initial), 0o600); err != nil {
		t.Fatalf("write initial file: %v", err)
	}

	ctx := context.Background()

	// 1. Section violation: public function in private section
	err := InsertFunction(ctx, file, `func PublicTwo() {}`, FunctionOptions{
		Placement: PlacementPrivateStart,
	})
	if !errors.Is(err, ErrSectionViolation) {
		t.Fatalf("expected ErrSectionViolation placing public function in private section, got %v", err)
	}
	var pErr *PlacementError
	if !errors.As(err, &pErr) {
		t.Fatalf("expected *PlacementError via errors.As, got %T", err)
	}
	if pErr.Strategy != PlacementPrivateStart {
		t.Errorf("expected Strategy PlacementPrivateStart, got %q", pErr.Strategy)
	}
	if pErr.TargetSymbol != "PublicTwo" {
		t.Errorf("expected TargetSymbol 'PublicTwo', got %q", pErr.TargetSymbol)
	}

	// 2. Section violation: private function in public section
	err = InsertFunction(ctx, file, `func privateTwo() {}`, FunctionOptions{
		Placement: PlacementPublicStart,
	})
	if !errors.Is(err, ErrSectionViolation) {
		t.Fatalf("expected ErrSectionViolation placing private function in public section, got %v", err)
	}

	// 3. Unsupported access modifier in Go
	err = InsertFunction(ctx, file, `func ProtectedHelper() {}`, FunctionOptions{
		AccessModifier: AccessModifierProtected,
	})
	if !errors.Is(err, ErrUnsupportedModifier) {
		t.Fatalf("expected ErrUnsupportedModifier for protected access modifier in Go, got %v", err)
	}

	// 4. Non-function snippet
	err = InsertFunction(ctx, file, `type NonFunction struct {}`, FunctionOptions{})
	if !errors.Is(err, ErrUnexpectedDeclType) {
		t.Fatalf("expected ErrUnexpectedDeclType when passing struct type to InsertFunction, got %v", err)
	}

	// 5. Syntax error in snippet
	err = InsertFunction(ctx, file, `func broken( {`, FunctionOptions{})
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
