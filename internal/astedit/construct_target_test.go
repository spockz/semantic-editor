// These tests ensure target preflight reuses native selectors without writing
// source or losing typed missing-target errors.
package astedit

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/spockz/semantic-editor/internal/symbol"
)

func TestValidateConstructTargetUsesNativeSelectorsWithoutMutation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "target.go")
	source := []byte(`package target
type Store struct{}
const Version = 1
func run() {
	if ready { use() }
	old()
}
`)
	if err := os.WriteFile(path, source, 0o600); err != nil {
		t.Fatal(err)
	}
	checks := []struct {
		name          string
		owner         string
		kind          ConstructKind
		discriminator string
		opts          ConstructOptions
	}{
		{name: "control", owner: "run", kind: ConstructIf, discriminator: "ready"},
		{name: "statement expression", owner: "run", kind: ConstructStatementOrExpr, discriminator: "old()", opts: ConstructOptions{ConstructPath: "1"}},
		{name: "declaration", owner: "Store", kind: ConstructType},
	}
	for _, check := range checks {
		t.Run(check.name, func(t *testing.T) {
			source := ""
			if check.kind == ConstructType {
				source = "type Store struct{}"
			}
			if err := ValidateConstructReplacementTarget(context.Background(), path, check.owner, check.kind, check.discriminator, source, check.opts); err != nil {
				t.Fatalf("ValidateConstructTarget() error = %v", err)
			}
		})
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(source) {
		t.Fatalf("preflight mutated source:\n%s", got)
	}
}

func TestValidateConstructTargetPreservesTypedNotFound(t *testing.T) {
	path := filepath.Join(t.TempDir(), "target.go")
	if err := os.WriteFile(path, []byte("package target\nfunc run() { old() }\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	err := ValidateConstructReplacementTarget(context.Background(), path, "Missing", ConstructFunction, "", "func Missing() {}", ConstructOptions{})
	if !errors.Is(err, symbol.ErrNotFound) {
		t.Fatalf("missing declaration error = %v, want symbol.ErrNotFound", err)
	}
	err = ValidateConstructReplacementTarget(context.Background(), path, "Missing", ConstructType, "", "type Missing struct{}", ConstructOptions{})
	if !errors.Is(err, symbol.ErrNotFound) {
		t.Fatalf("missing type error = %v, want symbol.ErrNotFound", err)
	}
	err = ValidateConstructReplacementTarget(context.Background(), path, "run", ConstructIf, "missing", "", ConstructOptions{})
	if constructSelectionErrorStatus(err) != "not_found" {
		t.Fatalf("missing construct error status = %q, want not_found (%v)", constructSelectionErrorStatus(err), err)
	}
}

func TestValidateConstructReplacementTargetChecksDeclarationIdentity(t *testing.T) {
	path := filepath.Join(t.TempDir(), "target.go")
	const source = "package target\ntype Store struct{}\nfunc (s *Store) Save() {}\n"
	if err := os.WriteFile(path, []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}
	checks := []struct {
		name   string
		kind   ConstructKind
		target string
		body   string
	}{
		{name: "receiver syntax", kind: ConstructMethod, target: "Store.Save", body: "func (s Store) Save() {}"},
		{name: "replacement name", kind: ConstructMethod, target: "Store.Save", body: "func (s *Store) Other() {}"},
		{name: "replacement kind", kind: ConstructType, target: "Store", body: "var Store = 1"},
	}
	for _, check := range checks {
		t.Run(check.name, func(t *testing.T) {
			if err := ValidateConstructReplacementTarget(context.Background(), path, check.target, check.kind, "", check.body, ConstructOptions{}); err == nil {
				t.Fatal("ValidateConstructReplacementTarget() succeeded for mismatched replacement identity")
			}
		})
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != source {
		t.Fatalf("preflight mutated source:\n%s", got)
	}
}
