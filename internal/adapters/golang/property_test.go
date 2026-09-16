// Package golang_test validates mathematical refactoring invariants using rapid property-based testing.
package golang_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"pgregory.net/rapid"
	"semedit/internal/adapters/golang"
	"semedit/internal/pipeline"
	"semedit/internal/symbol"
)

// TestProperty_RoundTripInvertibility asserts R^-1(R(P)) == P across generated AST variations.
func TestProperty_RoundTripInvertibility(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping property test in short mode")
	}

	rapid.Check(t, func(rt *rapid.T) {
		typeName := rapid.StringMatching(`[A-Z][a-zA-Z]{3,8}`).Draw(rt, "typeName")
		oldMethod := rapid.StringMatching(`[A-Z][a-zA-Z]{3,8}`).Draw(rt, "oldMethod")
		newMethod := rapid.StringMatching(`[A-Z][a-zA-Z]{3,8}`).Draw(rt, "newMethod")

		if oldMethod == newMethod || oldMethod == typeName || newMethod == typeName {
			return
		}

		callCount := rapid.IntRange(1, 4).Draw(rt, "callCount")

		tmpDir, err := os.MkdirTemp("", "prop-*")
		if err != nil {
			rt.Fatalf("create temp dir: %v", err)
		}
		defer func() {
			_ = os.RemoveAll(tmpDir)
		}()

		modFile := filepath.Join(tmpDir, "go.mod")
		if err := os.WriteFile(modFile, []byte("module example.com/prop\n\ngo 1.23\n"), 0o600); err != nil {
			rt.Fatalf("write mod file: %v", err)
		}

		var calls strings.Builder
		for range callCount {
			fmt.Fprintf(&calls, "\ts.%s()\n", oldMethod)
		}

		source := fmt.Sprintf(`package prop

type %s struct{}

func (s *%s) %s() string {
	return "ok"
}

func Execute() {
	s := &%s{}
%s}
`, typeName, typeName, oldMethod, typeName, calls.String())

		targetFile := filepath.Join(tmpDir, "prop.go")
		if err := os.WriteFile(targetFile, []byte(source), 0o600); err != nil {
			rt.Fatalf("write source: %v", err)
		}

		ctx := context.Background()
		if err := pipeline.Format(ctx, tmpDir, targetFile); err != nil {
			rt.Fatalf("format initial source: %v", err)
		}

		originalBytes, err := os.ReadFile(filepath.Clean(targetFile))
		if err != nil {
			rt.Fatalf("read initial source: %v", err)
		}

		// Step 1: Forward transformation (A -> B)
		symQuery := typeName + "." + oldMethod
		res, err := symbol.Resolve(tmpDir, "prop.go", symQuery)
		if err != nil {
			rt.Fatalf("resolve forward symbol %s: %v", symQuery, err)
		}

		if err := golang.Rename(ctx, tmpDir, res.File, res.Line, res.Column, newMethod); err != nil {
			rt.Fatalf("forward rename failed: %v", err)
		}
		if err := pipeline.Format(ctx, tmpDir, targetFile); err != nil {
			rt.Fatalf("format forward source: %v", err)
		}

		// Invariant: Compilation health after transformation
		diags, err := pipeline.CheckDiagnostics(ctx, tmpDir)
		if err != nil || len(diags) > 0 {
			rt.Fatalf("compilation health broken after forward rename: %v (%v)", err, diags)
		}

		// Step 2: Inverse transformation (B -> A)
		invQuery := typeName + "." + newMethod
		resInv, err := symbol.Resolve(tmpDir, "prop.go", invQuery)
		if err != nil {
			rt.Fatalf("resolve inverse symbol %s: %v", invQuery, err)
		}

		if err := golang.Rename(ctx, tmpDir, resInv.File, resInv.Line, resInv.Column, oldMethod); err != nil {
			rt.Fatalf("inverse rename failed: %v", err)
		}
		if err := pipeline.Format(ctx, tmpDir, targetFile); err != nil {
			rt.Fatalf("format inverse source: %v", err)
		}

		roundTripBytes, err := os.ReadFile(filepath.Clean(targetFile))
		if err != nil {
			rt.Fatalf("read round-trip source: %v", err)
		}

		// Invariant: Round-trip invertibility R^-1(R(P)) == P
		if string(roundTripBytes) != string(originalBytes) {
			rt.Fatalf("invertibility invariant violated:\nGot:\n%s\nWant:\n%s", roundTripBytes, originalBytes)
		}
	})
}
