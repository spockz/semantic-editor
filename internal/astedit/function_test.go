// Tests for specialized function and method insertion, receiver clustering, and access validation.
package astedit

import (
	"context"
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
	if err == nil {
		t.Fatal("expected error placing public function in private section, got nil")
	}

	// 2. Section violation: private function in public section
	err = InsertFunction(ctx, file, `func privateTwo() {}`, FunctionOptions{
		Placement: PlacementPublicStart,
	})
	if err == nil {
		t.Fatal("expected error placing private function in public section, got nil")
	}

	// 3. Unsupported access modifier in Go
	err = InsertFunction(ctx, file, `func ProtectedHelper() {}`, FunctionOptions{
		AccessModifier: AccessModifierProtected,
	})
	if err == nil {
		t.Fatal("expected error for protected access modifier in Go, got nil")
	}

	// 4. Non-function snippet
	err = InsertFunction(ctx, file, `type NonFunction struct {}`, FunctionOptions{})
	if err == nil {
		t.Fatal("expected error when passing struct type to InsertFunction, got nil")
	}
}
