// Package astedit tests compiler-backed safe-delete scan invariants.
// WHY: The public removal preflight can refuse by textual evidence before reaching the compiler reference scan.
package astedit

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestSafeDeleteReferenceScanFindsCompilerResolvedUse(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/safescan\n\ngo 1.23\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "decls.go")
	source := "package sample\nfunc obsolete() {}\nfunc keep() { obsolete() }\n"
	if err := os.WriteFile(path, []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}
	packages, err := loadMovePackages(context.Background(), root, nil)
	if err != nil {
		t.Fatalf("load compiler package: %v", err)
	}
	target := safeDeleteTarget{name: "obsolete", kind: "function"}
	definition := safeDeleteDefinition(packages, root, path, target)
	if definition.object == nil {
		t.Fatal("safe-delete target did not resolve to a compiler object")
	}
	if !safeDeleteHasReference(packages, root, path, target, definition.object, definition.packagePath) {
		t.Fatal("compiler-resolved use of the target was not detected")
	}
}
