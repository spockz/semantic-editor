// WHY: Uses real Go type checking to prove variant agreement and reject candidate changes that still compile.

package operation

import (
	"context"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"path/filepath"
	"slices"
	"testing"

	"github.com/spockz/semantic-editor/internal/astedit"
	"golang.org/x/tools/go/packages"
)

func TestSignaturePermutationTypedCandidateProof(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	source := []byte("package sample\nconst named=1\nfunc combine(left,right any)any{return left}\nfunc use()any{return combine(named,2)}\n")
	file := filepath.Join(root, "service.go")
	snapshot := structuralSnapshot{Root: root, File: file, Ctx: context.Background(), Sources: map[string][]byte{file: source}}
	snapshot.Packages = []*packages.Package{signaturePermutationTypedPackage(t, snapshot.Sources, "")}
	zero, one := 0, 1
	intent := signatureChangeIntent{Symbol: "combine", Mappings: []astedit.SignatureParameterMapping{{SourceIndex: &one, Name: "right", Type: "any"}, {SourceIndex: &zero, Name: "left", Type: "any"}}}
	plan, err := planSignatureChange(&snapshot, intent)
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	if len(plan.Files) != 1 {
		t.Fatalf("planned files = %d", len(plan.Files))
	}
	for _, scenario := range []struct {
		name, source string
		accepted     bool
	}{
		{"original pairing", "package sample\nconst named=1\nfunc combine(right any,left any)any{return left}\nfunc use()any{return combine(2,named)}\n", true},
		{"changed value", "package sample\nconst named=9\nfunc combine(right any,left any)any{return left}\nfunc use()any{return combine(2,named)}\n", false},
		{"changed dynamic type", "package sample\nconst named int64=1\nfunc combine(right any,left any)any{return left}\nfunc use()any{return combine(2,named)}\n", false},
		{"changed parameter type", "package sample\nconst named=1\nfunc combine(right any,left int)any{return left}\nfunc use()any{return combine(2,named)}\n", false},
		{"different binding", "package sample\nconst named=1\nfunc combine(right any,left any)any{return left}\nfunc other(right any,left any)any{return left}\nfunc use()any{return other(2,named)}\n", false},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			candidate := plan
			candidate.Files = slices.Clone(plan.Files)
			candidate.Files[0].After = []byte(scenario.source)
			loaded := []*packages.Package{signaturePermutationTypedPackage(t, map[string][]byte{file: []byte(scenario.source)}, "")}
			proofErr := validateSignatureProjectionCandidate(&snapshot, candidate, loaded)
			if (proofErr == nil) != scenario.accepted {
				t.Errorf("candidate accepted = %v, want %v: %v", proofErr == nil, scenario.accepted, proofErr)
			}
		})
	}
}

func TestSignaturePermutationVariantConstantAgreement(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	file, constants := filepath.Join(root, "service.go"), filepath.Join(root, "constants.go")
	sources := map[string][]byte{file: []byte("package sample\nfunc combine(left,right int)int{return left*100+right}\nfunc use()int{return combine(named,2)}\n"), constants: []byte("package sample\nconst named=1\n")}
	ordinary := signaturePermutationTypedPackage(t, sources, "")
	changed := map[string][]byte{file: sources[file], constants: []byte("package sample\nconst named=9\n")}
	variant := signaturePermutationTypedPackage(t, changed, " [sample.test]")
	snapshot := structuralSnapshot{Root: root, File: file, Ctx: context.Background(), Sources: sources, Packages: []*packages.Package{ordinary, variant}}
	proof, err := collectSignatureCallSetWithName(&snapshot, "combine", "")
	if err != nil {
		t.Fatalf("discover calls: %v", err)
	}
	if _, evidenceErr := collectSignatureProjectionEvidence(&snapshot, "combine", &proof, signatureProjectionPolicy{RequireConstants: true}); evidenceErr == nil {
		t.Error("disagreeing compiler constant values were accepted")
	}
	variant = signaturePermutationTypedPackage(t, sources, " [sample.test]")
	snapshot.Packages[1] = variant
	proof, err = collectSignatureCallSetWithName(&snapshot, "combine", "")
	if err != nil {
		t.Fatalf("discover agreeing calls: %v", err)
	}
	if _, evidenceErr := collectSignatureProjectionEvidence(&snapshot, "combine", &proof, signatureProjectionPolicy{RequireConstants: true}); evidenceErr != nil {
		t.Errorf("agreeing variants: %v", evidenceErr)
	}
}

func signaturePermutationTypedPackage(t *testing.T, sources map[string][]byte, variant string) *packages.Package {
	t.Helper()
	fset := token.NewFileSet()
	paths := make([]string, 0, len(sources))
	for path := range sources {
		paths = append(paths, path)
	}
	slices.Sort(paths)
	syntax := make([]*ast.File, 0, len(paths))
	for _, path := range paths {
		file, err := parser.ParseFile(fset, path, sources[path], parser.ParseComments)
		if err != nil {
			t.Fatalf("parse compiler fixture: %v", err)
		}
		syntax = append(syntax, file)
	}
	info := &types.Info{Types: make(map[ast.Expr]types.TypeAndValue), Defs: make(map[*ast.Ident]types.Object), Uses: make(map[*ast.Ident]types.Object), Scopes: make(map[ast.Node]*types.Scope)}
	config := types.Config{}
	pkg, err := config.Check("example.com/proof", fset, syntax, info)
	if err != nil {
		t.Fatalf("typecheck compiler fixture: %v", err)
	}
	return &packages.Package{ID: pkg.Path() + variant, PkgPath: pkg.Path(), Fset: fset, Syntax: syntax, Types: pkg, TypesInfo: info, CompiledGoFiles: paths}
}

func TestSignaturePermutationLocalTypeIdentity(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	file := filepath.Join(root, "service.go")
	source := []byte("package sample\nfunc combine(left,right any)any{return left}\nfunc use()any{type local int;const named local=1;{type local int;const other local=1;return combine(named,2)}}\n")
	snapshot := structuralSnapshot{Root: root, File: file, Ctx: context.Background(), Sources: map[string][]byte{file: source}}
	snapshot.Packages = []*packages.Package{signaturePermutationTypedPackage(t, snapshot.Sources, "")}
	zero, one := 0, 1
	intent := signatureChangeIntent{Symbol: "combine", Mappings: []astedit.SignatureParameterMapping{{SourceIndex: &one, Name: "right", Type: "any"}, {SourceIndex: &zero, Name: "left", Type: "any"}}}
	plan, err := planSignatureChange(&snapshot, intent)
	if err != nil {
		t.Fatalf("plan local constant: %v", err)
	}
	valid := map[string][]byte{file: plan.Files[0].After}
	if proofErr := validateSignatureProjectionCandidate(&snapshot, plan, []*packages.Package{signaturePermutationTypedPackage(t, valid, "")}); proofErr != nil {
		t.Errorf("unchanged local type refused: %v", proofErr)
	}
	candidate := plan
	candidate.Files = slices.Clone(plan.Files)
	candidate.Files[0].After = []byte("package sample\nfunc combine(right any,left any)any{return left}\nfunc use()any{type local int;const named local=1;{type local int;const named local=1;return combine(2,named)}}\n")
	changed := map[string][]byte{file: candidate.Files[0].After}
	if proofErr := validateSignatureProjectionCandidate(&snapshot, candidate, []*packages.Package{signaturePermutationTypedPackage(t, changed, "")}); proofErr == nil {
		t.Error("distinct same-named local type with equal value and source was accepted")
	}
}
