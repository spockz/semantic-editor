// Package astedit tests compiler-backed safe-delete scan invariants.
// WHY: The public removal preflight can refuse by textual evidence before reaching the compiler reference scan.
package astedit

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestSafeDeleteReferenceScanClassifiesCompilerResolvedUses(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/safescan\n\ngo 1.23\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "decls.go")
	source := "package sample\nconst usedLimit = 42\nconst unusedLimit = 7\nvar usedCount int = 3\nvar unusedCount int = 4\nfunc obsolete() {}\nfunc keep() { obsolete(); _ = usedLimit; _ = usedCount }\n"
	if err := os.WriteFile(path, []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}
	packages, err := loadMovePackages(context.Background(), root, nil)
	if err != nil {
		t.Fatalf("load compiler package: %v", err)
	}
	cases := []struct {
		name       string
		kind       string
		referenced bool
	}{
		{name: "obsolete", kind: "function", referenced: true},
		{name: "usedLimit", kind: "constant", referenced: true},
		{name: "usedCount", kind: "value", referenced: true},
		{name: "unusedLimit", kind: "constant"},
		{name: "unusedCount", kind: "value"},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			target := safeDeleteTarget{name: testCase.name, kind: testCase.kind}
			definition := safeDeleteDefinition(packages, root, path, target)
			if definition.object == nil {
				t.Fatal("safe-delete target did not resolve to a compiler object")
			}
			got := safeDeleteHasReference(packages, root, path, target, definition.object, definition.packagePath)
			if got != testCase.referenced {
				t.Fatalf("compiler reference = %t, want %t", got, testCase.referenced)
			}
		})
	}
}
