// WHY: Prove removal preserves typed survivor bindings and checks every active target variant using the real Go compiler.
package operation

import (
	"context"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/spockz/semantic-editor/internal/astedit"
	"golang.org/x/tools/go/packages"
)

func TestSignatureRemovalTypedCandidateProof(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	file := filepath.Join(root, "service.go")
	source := []byte("package sample\nconst named int64=2\nfunc next()int{return 3}\nfunc other()int{return 4}\nfunc combine(left int,legacy string,right any)any{return right}\nfunc use()any{return combine(next(),\"old\",named)}\n")
	snapshot := structuralSnapshot{Root: root, File: file, Ctx: context.Background(), Sources: map[string][]byte{file: source}}
	snapshot.Packages = []*packages.Package{signaturePermutationTypedPackage(t, snapshot.Sources, "")}
	zero, two := 0, 2
	intent := signatureChangeIntent{Symbol: "combine", Mappings: []astedit.SignatureParameterMapping{{SourceIndex: &zero, Name: "left", Type: "int"}, {SourceIndex: &two, Name: "right", Type: "any"}}}
	plan, err := planSignatureChange(&snapshot, intent)
	if err != nil {
		t.Fatalf("plan removal: %v", err)
	}
	if len(plan.Files) != 1 {
		t.Fatalf("planned files = %d", len(plan.Files))
	}
	correct := string(plan.Files[0].After)
	for _, scenario := range []struct {
		name     string
		source   string
		accepted bool
	}{
		{"original survivors", correct, true},
		{"changed constant value", strings.Replace(correct, "named int64=2", "named int64=9", 1), false},
		{"changed dynamic type", strings.Replace(correct, "named int64=2", "named int32=2", 1), false},
		{"changed effectful survivor", strings.Replace(correct, "combine(next(), named)", "combine(other(), named)", 1), false},
		{"changed parameter type", strings.Replace(correct, "combine(left int, right any)", "combine(left any, right any)", 1), false},
		{"changed call binding", strings.Replace(correct, "return combine(next(), named)", "return named", 1), false},
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

func TestSignatureRemovalTargetVariantProof(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	file := filepath.Join(root, "service.go")
	source := []byte("package sample\nfunc keep(value int,legacy int)int{{legacy:=value;_=legacy};return value}\nfunc use()int{return keep(7,1)}\n")
	snapshot := structuralSnapshot{Root: root, File: file, Ctx: context.Background(), Sources: map[string][]byte{file: source}}
	snapshot.Packages = []*packages.Package{signaturePermutationTypedPackage(t, snapshot.Sources, "")}
	zero := 0
	intent := signatureChangeIntent{Symbol: "keep", Mappings: []astedit.SignatureParameterMapping{{SourceIndex: &zero, Name: "value", Type: "int"}}}
	if _, err := planSignatureChange(&snapshot, intent); err != nil {
		t.Fatalf("unrelated shadow should not use removed formal: %v", err)
	}
	t.Run("missing use evidence", func(t *testing.T) {
		incomplete := snapshot
		pkg := *snapshot.Packages[0]
		info := *pkg.TypesInfo
		info.Uses = nil
		pkg.TypesInfo = &info
		incomplete.Packages = []*packages.Package{&pkg}
		if _, err := planSignatureChange(&incomplete, intent); err == nil {
			t.Error("missing identifier-use evidence was accepted")
		}
	})
	variantSource := strings.Replace(string(source), "return value}", "_=legacy;return value}", 1)
	snapshot.Packages = append(snapshot.Packages, signaturePermutationTypedPackage(t, map[string][]byte{file: []byte(variantSource)}, "[test variant]"))
	if _, err := planSignatureChange(&snapshot, intent); err == nil || !strings.Contains(err.Error(), "removed parameter is used") {
		t.Errorf("variant-only formal use was accepted: %v", err)
	}
}
