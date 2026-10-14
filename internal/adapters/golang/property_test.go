// Package golang_test validates mathematical refactoring invariants using rapid property-based testing.
package golang_test

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spockz/semantic-editor/internal/adapters/golang"
	"github.com/spockz/semantic-editor/internal/pipeline"
	"github.com/spockz/semantic-editor/internal/symbol"

	"bytes"

	"pgregory.net/rapid"
)

func configureRapidChecks(t *testing.T, shortChecks int, defaultChecks int) {
	t.Helper()
	var cliSet bool
	flag.Visit(func(f *flag.Flag) {
		if f.Name == "rapid.checks" {
			cliSet = true
		}
	})
	if cliSet {
		return
	}
	if _, ok := os.LookupEnv("RAPID_CHECKS"); ok {
		return
	}
	if testing.Short() {
		// Rapid divides flags.checks by 5 when testing.Short() is true
		_ = flag.Set("rapid.checks", fmt.Sprintf("%d", shortChecks*5))
	} else {
		_ = flag.Set("rapid.checks", fmt.Sprintf("%d", defaultChecks))
	}
}

// TestProperty_RoundTripInvertibility asserts R^-1(R(P)) == P across generated AST variations.
func TestProperty_RoundTripInvertibility(t *testing.T) {
	configureRapidChecks(t, 2, 2)
	rapid.Check(t, checkRoundTripProperty)
}

type roundTripFixture struct {
	root, file, typeName, oldMethod, newMethod string
	original                                   []byte
}

func checkRoundTripProperty(rt *rapid.T) {
	typeName := rapid.StringMatching(`[A-Z][a-zA-Z]{3,8}`).Draw(rt, "typeName")
	oldMethod := rapid.StringMatching(`[A-Z][a-zA-Z]{3,8}`).Draw(rt, "oldMethod")
	newMethod := rapid.StringMatching(`[A-Z][a-zA-Z]{3,8}`).Draw(rt, "newMethod")
	if oldMethod == newMethod || oldMethod == typeName || newMethod == typeName {
		return
	}
	fixture := createRoundTripFixture(rt, typeName, oldMethod, newMethod)
	defer func() {
		if err := os.RemoveAll(fixture.root); err != nil {
			rt.Errorf("remove round-trip fixture: %v", err)
		}
	}()
	verifyRoundTripFixture(rt, *fixture)
}

func createRoundTripFixture(rt *rapid.T, typeName, oldMethod, newMethod string) *roundTripFixture {
	root, createErr := os.MkdirTemp("", "prop-*")
	if createErr != nil {
		rt.Fatalf("create temp dir: %v", createErr)
	}
	cleanupOnError := true
	defer func() {
		if cleanupOnError {
			if err := os.RemoveAll(root); err != nil {
				rt.Errorf("remove failed round-trip fixture: %v", err)
			}
		}
	}()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/prop\n\ngo 1.23\n"), 0o600); err != nil {
		rt.Fatalf("write mod file: %v", err)
	}
	callCount := rapid.IntRange(1, 4).Draw(rt, "callCount")
	source := roundTripSource(typeName, oldMethod, callCount)
	file := filepath.Join(root, "prop.go")
	if err := os.WriteFile(file, []byte(source), 0o600); err != nil {
		rt.Fatalf("write source: %v", err)
	}
	if err := pipeline.Format(context.Background(), root, file); err != nil {
		rt.Fatalf("format initial source: %v", err)
	}
	original, err := os.ReadFile(filepath.Clean(file))
	if err != nil {
		rt.Fatalf("read initial source: %v", err)
	}
	cleanupOnError = false
	return &roundTripFixture{root: root, file: file, typeName: typeName, oldMethod: oldMethod, newMethod: newMethod, original: original}
}

func roundTripSource(typeName, method string, callCount int) string {
	var calls strings.Builder
	for range callCount {
		fmt.Fprintf(&calls, "\ts.%s()\n", method)
	}
	return fmt.Sprintf(`package prop

type %s struct{}

func (s *%s) %s() string {
	return "ok"
}

func Execute() {
	s := &%s{}
%s}
`, typeName, typeName, method, typeName, calls.String())
}

func verifyRoundTripFixture(rt *rapid.T, fixture roundTripFixture) {
	ctx := context.Background()
	renamePropertyMethod(rt, fixture, fixture.oldMethod, fixture.newMethod, "forward")
	diags, err := pipeline.CheckDiagnostics(ctx, fixture.root)
	if err != nil || len(diags) > 0 {
		rt.Fatalf("compilation health broken after forward rename: %v (%v)", err, diags)
	}
	renamePropertyMethod(rt, fixture, fixture.newMethod, fixture.oldMethod, "inverse")
	result, err := os.ReadFile(filepath.Clean(fixture.file))
	if err != nil {
		rt.Fatalf("read round-trip source: %v", err)
	}
	if !bytes.Equal(result, fixture.original) {
		rt.Fatalf("invertibility invariant violated:\nGot:\n%s\nWant:\n%s", result, fixture.original)
	}
}

func renamePropertyMethod(rt *rapid.T, fixture roundTripFixture, from, to, direction string) {
	rt.Helper()
	query := fixture.typeName + "." + from
	resolved, err := symbol.Resolve(fixture.root, "prop.go", query)
	if err != nil {
		rt.Fatalf("resolve %s symbol %s: %v", direction, query, err)
	}
	if resolved.Definition == nil || resolved.IsAmbiguous() {
		rt.Fatalf("resolve %s symbol %s did not return a unique definition: %+v", direction, query, resolved)
	}
	definition := resolved.Definition
	if err := golang.Rename(context.Background(), fixture.root, definition.File, definition.Line, definition.Column, to); err != nil {
		rt.Fatalf("%s rename failed: %v", direction, err)
	}
	if err := pipeline.Format(context.Background(), fixture.root, fixture.file); err != nil {
		rt.Fatalf("format %s source: %v", direction, err)
	}
}
